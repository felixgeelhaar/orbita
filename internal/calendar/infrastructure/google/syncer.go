package google

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	calendarApp "github.com/felixgeelhaar/orbita/internal/calendar/application"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

const defaultBaseURL = "https://www.googleapis.com/calendar/v3"

type tokenSourceProvider interface {
	TokenSource(ctx context.Context, userID uuid.UUID) (oauth2.TokenSource, error)
}

// Syncer syncs schedule blocks to Google Calendar.
type Syncer struct {
	oauthService  tokenSourceProvider
	logger        *slog.Logger
	baseURL       string
	deleteMissing bool
	calendarID    string
	attendees     []string
	reminders     []int
}

// NewSyncer creates a Google Calendar syncer.
func NewSyncer(oauthService tokenSourceProvider, logger *slog.Logger) *Syncer {
	if logger == nil {
		logger = slog.Default()
	}
	return &Syncer{
		oauthService:  oauthService,
		logger:        logger,
		baseURL:       defaultBaseURL,
		deleteMissing: false,
		calendarID:    "primary",
		attendees:     nil,
		reminders:     nil,
	}
}

// NewSyncerWithBaseURL creates a Google Calendar syncer with a custom base URL.
func NewSyncerWithBaseURL(oauthService tokenSourceProvider, logger *slog.Logger, baseURL string) *Syncer {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Syncer{
		oauthService:  oauthService,
		logger:        logger,
		baseURL:       baseURL,
		deleteMissing: false,
		calendarID:    "primary",
		attendees:     nil,
		reminders:     nil,
	}
}

// WithDeleteMissing enables deletion of events missing from the current sync set.
func (s *Syncer) WithDeleteMissing(enabled bool) *Syncer {
	s.deleteMissing = enabled
	return s
}

// WithCalendarID sets the calendar ID for sync.
func (s *Syncer) WithCalendarID(calendarID string) *Syncer {
	if calendarID != "" {
		s.calendarID = calendarID
	}
	return s
}

// WithAttendees sets attendee emails for sync.
func (s *Syncer) WithAttendees(attendees []string) *Syncer {
	s.attendees = attendees
	return s
}

// WithReminders sets reminder minutes for sync.
func (s *Syncer) WithReminders(reminders []int) *Syncer {
	s.reminders = reminders
	return s
}

// Sync pushes schedule blocks into the primary calendar.
func (s *Syncer) Sync(ctx context.Context, userID uuid.UUID, blocks []calendarApp.TimeBlock) (*calendarApp.SyncResult, error) {
	if s.oauthService == nil {
		return nil, fmt.Errorf("oauth service not configured")
	}
	tokenSource, err := s.oauthService.TokenSource(ctx, userID)
	if err != nil {
		return nil, err
	}
	token, err := tokenSource.Token()
	if err != nil {
		s.logger.Warn("oauth token refresh failed", "error", err)
		return nil, err
	}
	if !token.Expiry.IsZero() && time.Until(token.Expiry) < 24*time.Hour {
		s.logger.Warn("oauth token nearing expiry", "expires_at", token.Expiry)
	}

	client := http.Client{
		Timeout: 15 * time.Second,
		Transport: &oauthTransport{
			base:   http.DefaultTransport,
			source: tokenSource,
		},
	}

	result := &calendarApp.SyncResult{}
	keepIDs := make(map[string]struct{}, len(blocks))
	for _, block := range blocks {
		event := toGoogleEvent(block, s.attendees, s.reminders)
		keepIDs[event.ID] = struct{}{}
		updated, err := upsertEvent(ctx, &client, s.baseURL, s.calendarID, event)
		if err != nil {
			s.logger.Warn("calendar sync failed", "event_id", event.ID, "error", err)
			result.Failed++
			continue
		}
		if updated {
			result.Updated++
		} else {
			result.Created++
		}
	}

	if s.deleteMissing {
		deleted, err := deleteMissingEvents(ctx, &client, s.baseURL, s.calendarID, keepIDs)
		if err != nil {
			s.logger.Warn("calendar delete missing failed", "error", err)
		} else {
			result.Deleted = deleted
		}
	}

	return result, nil
}

type googleEvent struct {
	ID                 string `json:"id,omitempty"`
	Summary            string `json:"summary"`
	Description        string `json:"description,omitempty"`
	ExtendedProperties struct {
		Private map[string]string `json:"private,omitempty"`
	} `json:"extendedProperties,omitempty"`
	Attendees []struct {
		Email string `json:"email"`
	} `json:"attendees,omitempty"`
	Reminders struct {
		UseDefault bool `json:"useDefault"`
		Overrides  []struct {
			Method  string `json:"method"`
			Minutes int    `json:"minutes"`
		} `json:"overrides,omitempty"`
	} `json:"reminders,omitempty"`
	Start struct {
		DateTime string `json:"dateTime"`
	} `json:"start"`
	End struct {
		DateTime string `json:"dateTime"`
	} `json:"end"`
}

func toGoogleEvent(block calendarApp.TimeBlock, attendees []string, reminders []int) googleEvent {
	event := googleEvent{
		ID:      block.ID.String(),
		Summary: block.Title,
	}
	event.Description = fmt.Sprintf("Type: %s", block.BlockType)
	if block.Completed {
		event.Description += "\nStatus: Completed"
	} else if block.Missed {
		event.Description += "\nStatus: Missed"
	}
	event.ExtendedProperties.Private = map[string]string{
		"orbita": "1",
	}
	event.Start.DateTime = block.StartTime.Format(time.RFC3339)
	event.End.DateTime = block.EndTime.Format(time.RFC3339)

	if len(attendees) > 0 {
		event.Attendees = make([]struct {
			Email string `json:"email"`
		}, 0, len(attendees))
		for _, email := range attendees {
			if email == "" {
				continue
			}
			event.Attendees = append(event.Attendees, struct {
				Email string `json:"email"`
			}{Email: email})
		}
	}

	if len(reminders) > 0 {
		overrides := make([]struct {
			Method  string `json:"method"`
			Minutes int    `json:"minutes"`
		}, 0, len(reminders))
		for _, minutes := range reminders {
			if minutes <= 0 {
				continue
			}
			overrides = append(overrides, struct {
				Method  string `json:"method"`
				Minutes int    `json:"minutes"`
			}{
				Method:  "popup",
				Minutes: minutes,
			})
		}
		if len(overrides) > 0 {
			event.Reminders.UseDefault = false
			event.Reminders.Overrides = overrides
		}
	}

	return event
}

func upsertEvent(ctx context.Context, client *http.Client, baseURL, calendarID string, event googleEvent) (bool, error) {
	body, err := json.Marshal(event)
	if err != nil {
		return false, err
	}

	insertURL := fmt.Sprintf("%s/calendars/%s/events", baseURL, calendarID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, insertURL, bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		updateURL := fmt.Sprintf("%s/%s", insertURL, event.ID)
		req, err = http.NewRequestWithContext(ctx, http.MethodPut, updateURL, bytes.NewReader(body))
		if err != nil {
			return false, err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err = client.Do(req)
		if err != nil {
			return false, err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return true, nil
		}
		return false, responseError(resp)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return false, nil
	}
	return false, responseError(resp)
}

func responseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("calendar sync failed: status=%d body=%s", resp.StatusCode, string(body))
}

func deleteMissingEvents(ctx context.Context, client *http.Client, baseURL, calendarID string, keepIDs map[string]struct{}) (int, error) {
	listURL := fmt.Sprintf("%s/calendars/%s/events?privateExtendedProperty=orbita=1", baseURL, calendarID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		return 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, responseError(resp)
	}

	var list struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return 0, err
	}

	deleted := 0
	for _, item := range list.Items {
		if _, ok := keepIDs[item.ID]; ok {
			continue
		}
		deleteURL := fmt.Sprintf("%s/calendars/%s/events/%s", baseURL, calendarID, item.ID)
		req, err := http.NewRequestWithContext(ctx, http.MethodDelete, deleteURL, nil)
		if err != nil {
			return deleted, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return deleted, err
		}
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			deleted++
		} else {
			return deleted, responseError(resp)
		}
	}

	return deleted, nil
}

type oauthTransport struct {
	base   http.RoundTripper
	source oauth2.TokenSource
}

func (t *oauthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := t.source.Token()
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	return t.base.RoundTrip(req)
}

// Calendar represents a Google Calendar list entry.
type Calendar struct {
	ID      string
	Summary string
	Primary bool
}

// ListCalendars returns calendars accessible to the user.
func (s *Syncer) ListCalendars(ctx context.Context, userID uuid.UUID) ([]Calendar, error) {
	if s.oauthService == nil {
		return nil, fmt.Errorf("oauth service not configured")
	}
	tokenSource, err := s.oauthService.TokenSource(ctx, userID)
	if err != nil {
		return nil, err
	}
	client := http.Client{
		Timeout: 15 * time.Second,
		Transport: &oauthTransport{
			base:   http.DefaultTransport,
			source: tokenSource,
		},
	}

	listURL := fmt.Sprintf("%s/users/me/calendarList", s.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, responseError(resp)
	}

	var payload struct {
		Items []struct {
			ID      string `json:"id"`
			Summary string `json:"summary"`
			Primary bool   `json:"primary"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	calendars := make([]Calendar, 0, len(payload.Items))
	for _, item := range payload.Items {
		calendars = append(calendars, Calendar{
			ID:      item.ID,
			Summary: item.Summary,
			Primary: item.Primary,
		})
	}
	return calendars, nil
}

// DeleteEvent deletes a calendar event by block ID.
func (s *Syncer) DeleteEvent(ctx context.Context, userID uuid.UUID, blockID uuid.UUID) error {
	if s.oauthService == nil {
		return fmt.Errorf("oauth service not configured")
	}
	tokenSource, err := s.oauthService.TokenSource(ctx, userID)
	if err != nil {
		return err
	}
	client := http.Client{
		Timeout: 15 * time.Second,
		Transport: &oauthTransport{
			base:   http.DefaultTransport,
			source: tokenSource,
		},
	}

	deleteURL := fmt.Sprintf("%s/calendars/%s/events/%s", s.baseURL, s.calendarID, blockID.String())
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, deleteURL, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return responseError(resp)
}

// Event represents a Google Calendar event summary.
type Event struct {
	ID      string
	Summary string
	Start   time.Time
	End     time.Time
}

// ListEvents returns events within the given time range.
func (s *Syncer) ListEvents(ctx context.Context, userID uuid.UUID, start, end time.Time, onlyTagged bool) ([]Event, error) {
	if s.oauthService == nil {
		return nil, fmt.Errorf("oauth service not configured")
	}
	tokenSource, err := s.oauthService.TokenSource(ctx, userID)
	if err != nil {
		return nil, err
	}
	client := http.Client{
		Timeout: 15 * time.Second,
		Transport: &oauthTransport{
			base:   http.DefaultTransport,
			source: tokenSource,
		},
	}

	query := fmt.Sprintf("timeMin=%s&timeMax=%s&singleEvents=true&orderBy=startTime",
		start.UTC().Format(time.RFC3339),
		end.UTC().Format(time.RFC3339),
	)
	if onlyTagged {
		query += "&privateExtendedProperty=orbita=1"
	}
	listURL := fmt.Sprintf("%s/calendars/%s/events?%s", s.baseURL, s.calendarID, query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, responseError(resp)
	}

	var payload struct {
		Items []struct {
			ID      string `json:"id"`
			Summary string `json:"summary"`
			Start   struct {
				DateTime string `json:"dateTime"`
			} `json:"start"`
			End struct {
				DateTime string `json:"dateTime"`
			} `json:"end"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	events := make([]Event, 0, len(payload.Items))
	for _, item := range payload.Items {
		if item.Start.DateTime == "" || item.End.DateTime == "" {
			continue
		}
		startTime, err := time.Parse(time.RFC3339, item.Start.DateTime)
		if err != nil {
			continue
		}
		endTime, err := time.Parse(time.RFC3339, item.End.DateTime)
		if err != nil {
			continue
		}
		events = append(events, Event{
			ID:      item.ID,
			Summary: item.Summary,
			Start:   startTime,
			End:     endTime,
		})
	}
	return events, nil
}
