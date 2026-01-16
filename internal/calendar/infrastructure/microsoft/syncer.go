package microsoft

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	calendarApp "github.com/felixgeelhaar/orbita/internal/calendar/application"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

const defaultBaseURL = "https://graph.microsoft.com/v1.0"

// Microsoft OAuth2 endpoints
const (
	MicrosoftAuthURL  = "https://login.microsoftonline.com/common/oauth2/v2.0/authorize"
	MicrosoftTokenURL = "https://login.microsoftonline.com/common/oauth2/v2.0/token"
)

// Default scopes for Microsoft Calendar API
var DefaultScopes = []string{
	"https://graph.microsoft.com/Calendars.ReadWrite",
	"https://graph.microsoft.com/User.Read",
	"offline_access",
}

type tokenSourceProvider interface {
	TokenSource(ctx context.Context, userID uuid.UUID) (oauth2.TokenSource, error)
}

// Syncer syncs schedule blocks to Microsoft Outlook Calendar via Graph API.
type Syncer struct {
	oauthService  tokenSourceProvider
	logger        *slog.Logger
	baseURL       string
	deleteMissing bool
	calendarID    string // "primary" uses default calendar, or specific calendar ID
}

// NewSyncer creates a Microsoft Calendar syncer.
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
	}
}

// NewSyncerWithBaseURL creates a Microsoft Calendar syncer with a custom base URL.
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

// Sync pushes schedule blocks into the Microsoft Calendar.
func (s *Syncer) Sync(ctx context.Context, userID uuid.UUID, blocks []calendarApp.TimeBlock) (*calendarApp.SyncResult, error) {
	if s.oauthService == nil {
		return nil, fmt.Errorf("oauth service not configured")
	}

	client, err := s.getHTTPClient(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := &calendarApp.SyncResult{}
	keepIDs := make(map[string]struct{}, len(blocks))

	for _, block := range blocks {
		event := toMicrosoftEvent(block)
		eventID := block.ID.String()
		keepIDs[eventID] = struct{}{}

		updated, err := s.upsertEvent(ctx, client, event, eventID)
		if err != nil {
			s.logger.Warn("calendar sync failed", "event_id", eventID, "error", err)
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
		deleted, err := s.deleteMissingEvents(ctx, client, keepIDs)
		if err != nil {
			s.logger.Warn("calendar delete missing failed", "error", err)
		} else {
			result.Deleted = deleted
		}
	}

	return result, nil
}

// ListCalendars returns calendars accessible to the user.
func (s *Syncer) ListCalendars(ctx context.Context, userID uuid.UUID) ([]calendarApp.Calendar, error) {
	if s.oauthService == nil {
		return nil, fmt.Errorf("oauth service not configured")
	}

	client, err := s.getHTTPClient(ctx, userID)
	if err != nil {
		return nil, err
	}

	listURL := fmt.Sprintf("%s/me/calendars", s.baseURL)
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
		Value []struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			IsDefaultCalendar bool `json:"isDefaultCalendar"`
		} `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	calendars := make([]calendarApp.Calendar, 0, len(payload.Value))
	for _, item := range payload.Value {
		calendars = append(calendars, calendarApp.Calendar{
			ID:      item.ID,
			Name:    item.Name,
			Primary: item.IsDefaultCalendar,
		})
	}
	return calendars, nil
}

// ListEvents returns events within the given time range.
func (s *Syncer) ListEvents(ctx context.Context, userID uuid.UUID, start, end time.Time, onlyOrbitaEvents bool) ([]calendarApp.CalendarEvent, error) {
	if s.oauthService == nil {
		return nil, fmt.Errorf("oauth service not configured")
	}

	client, err := s.getHTTPClient(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Build the events URL with time filter
	eventsURL := s.eventsURL()
	params := url.Values{}
	params.Set("$filter", fmt.Sprintf("start/dateTime ge '%s' and end/dateTime le '%s'",
		start.UTC().Format("2006-01-02T15:04:05"),
		end.UTC().Format("2006-01-02T15:04:05"),
	))
	params.Set("$orderby", "start/dateTime")
	params.Set("$top", "100")

	fullURL := fmt.Sprintf("%s?%s", eventsURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Prefer", "outlook.timezone=\"UTC\"")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, responseError(resp)
	}

	var payload struct {
		Value []msEvent `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	events := make([]calendarApp.CalendarEvent, 0, len(payload.Value))
	for _, item := range payload.Value {
		// Check if this is an Orbita event via categories
		isOrbitaEvent := false
		for _, cat := range item.Categories {
			if cat == "Orbita" {
				isOrbitaEvent = true
				break
			}
		}

		if onlyOrbitaEvents && !isOrbitaEvent {
			continue
		}

		event := calendarApp.CalendarEvent{
			ID:            item.ID,
			Summary:       item.Subject,
			Description:   item.Body.Content,
			Location:      item.Location.DisplayName,
			Status:        mapMicrosoftStatus(item.ShowAs),
			IsOrbitaEvent: isOrbitaEvent,
			IsRecurring:   item.Recurrence != nil,
		}

		// Parse organizer
		if item.Organizer.EmailAddress.Address != "" {
			event.Organizer = item.Organizer.EmailAddress.Address
		}

		// Parse attendees
		if len(item.Attendees) > 0 {
			event.Attendees = make([]string, 0, len(item.Attendees))
			for _, att := range item.Attendees {
				if att.EmailAddress.Address != "" {
					event.Attendees = append(event.Attendees, att.EmailAddress.Address)
				}
			}
		}

		// Parse times
		if item.IsAllDay {
			startTime, err := time.Parse("2006-01-02", item.Start.DateTime[:10])
			if err != nil {
				continue
			}
			endTime, err := time.Parse("2006-01-02", item.End.DateTime[:10])
			if err != nil {
				continue
			}
			event.StartTime = startTime
			event.EndTime = endTime
			event.IsAllDay = true
		} else {
			startTime, err := time.Parse("2006-01-02T15:04:05.0000000", item.Start.DateTime)
			if err != nil {
				// Try without fractional seconds
				startTime, err = time.Parse("2006-01-02T15:04:05", item.Start.DateTime)
				if err != nil {
					continue
				}
			}
			endTime, err := time.Parse("2006-01-02T15:04:05.0000000", item.End.DateTime)
			if err != nil {
				endTime, err = time.Parse("2006-01-02T15:04:05", item.End.DateTime)
				if err != nil {
					continue
				}
			}
			event.StartTime = startTime
			event.EndTime = endTime
			event.IsAllDay = false
		}

		events = append(events, event)
	}

	return events, nil
}

// DeleteEvent deletes a calendar event by block ID.
func (s *Syncer) DeleteEvent(ctx context.Context, userID uuid.UUID, blockID uuid.UUID) error {
	if s.oauthService == nil {
		return fmt.Errorf("oauth service not configured")
	}

	client, err := s.getHTTPClient(ctx, userID)
	if err != nil {
		return err
	}

	// First, find the event by searching for it (using the Orbita ID in subject or categories)
	eventID, err := s.findEventByOrbitaID(ctx, client, blockID.String())
	if err != nil {
		return err
	}
	if eventID == "" {
		return nil // Event not found, consider it deleted
	}

	deleteURL := fmt.Sprintf("%s/%s", s.eventsURL(), eventID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, deleteURL, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 || resp.StatusCode == http.StatusNotFound {
		return nil
	}
	return responseError(resp)
}

// Helper methods

func (s *Syncer) getHTTPClient(ctx context.Context, userID uuid.UUID) (*http.Client, error) {
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

	return &http.Client{
		Timeout: 15 * time.Second,
		Transport: &oauthTransport{
			base:   http.DefaultTransport,
			source: tokenSource,
		},
	}, nil
}

func (s *Syncer) eventsURL() string {
	if s.calendarID == "primary" || s.calendarID == "" {
		return fmt.Sprintf("%s/me/events", s.baseURL)
	}
	return fmt.Sprintf("%s/me/calendars/%s/events", s.baseURL, s.calendarID)
}

func (s *Syncer) upsertEvent(ctx context.Context, client *http.Client, event msEvent, orbitaID string) (bool, error) {
	// First, try to find an existing event with this Orbita ID
	existingID, err := s.findEventByOrbitaID(ctx, client, orbitaID)
	if err != nil {
		return false, err
	}

	body, err := json.Marshal(event)
	if err != nil {
		return false, err
	}

	if existingID != "" {
		// Update existing event
		updateURL := fmt.Sprintf("%s/%s", s.eventsURL(), existingID)
		req, err := http.NewRequestWithContext(ctx, http.MethodPatch, updateURL, bytes.NewReader(body))
		if err != nil {
			return false, err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return false, err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return true, nil
		}
		return false, responseError(resp)
	}

	// Create new event
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.eventsURL(), bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return false, nil
	}
	return false, responseError(resp)
}

func (s *Syncer) findEventByOrbitaID(ctx context.Context, client *http.Client, orbitaID string) (string, error) {
	// Search for events with Orbita category that contain the orbitaID in subject
	searchURL := fmt.Sprintf("%s?$filter=categories/any(c:c eq 'Orbita') and contains(subject, '[%s]')&$top=1",
		s.eventsURL(), orbitaID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// If filter fails, fall back to getting all Orbita events and searching manually
		return s.findEventByOrbitaIDFallback(ctx, client, orbitaID)
	}

	var payload struct {
		Value []struct {
			ID string `json:"id"`
		} `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}

	if len(payload.Value) > 0 {
		return payload.Value[0].ID, nil
	}
	return "", nil
}

func (s *Syncer) findEventByOrbitaIDFallback(ctx context.Context, client *http.Client, orbitaID string) (string, error) {
	// Fallback: Get all Orbita events and search manually
	searchURL := fmt.Sprintf("%s?$filter=categories/any(c:c eq 'Orbita')&$top=100", s.eventsURL())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil // Can't find, assume doesn't exist
	}

	var payload struct {
		Value []struct {
			ID      string `json:"id"`
			Subject string `json:"subject"`
		} `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}

	searchPattern := fmt.Sprintf("[%s]", orbitaID)
	for _, item := range payload.Value {
		if bytes.Contains([]byte(item.Subject), []byte(searchPattern)) {
			return item.ID, nil
		}
	}
	return "", nil
}

func (s *Syncer) deleteMissingEvents(ctx context.Context, client *http.Client, keepIDs map[string]struct{}) (int, error) {
	// Get all Orbita events
	searchURL := fmt.Sprintf("%s?$filter=categories/any(c:c eq 'Orbita')&$top=100", s.eventsURL())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
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

	var payload struct {
		Value []struct {
			ID      string `json:"id"`
			Subject string `json:"subject"`
		} `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, err
	}

	deleted := 0
	for _, item := range payload.Value {
		// Extract Orbita ID from subject
		orbitaID := extractOrbitaID(item.Subject)
		if orbitaID == "" {
			continue
		}
		if _, ok := keepIDs[orbitaID]; ok {
			continue
		}

		// Delete this event
		deleteURL := fmt.Sprintf("%s/%s", s.eventsURL(), item.ID)
		req, err := http.NewRequestWithContext(ctx, http.MethodDelete, deleteURL, nil)
		if err != nil {
			return deleted, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return deleted, err
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			deleted++
		}
	}

	return deleted, nil
}

// Microsoft Graph API event types

type msEvent struct {
	Subject    string       `json:"subject"`
	Body       msBody       `json:"body"`
	Start      msDateTime   `json:"start"`
	End        msDateTime   `json:"end"`
	Location   msLocation   `json:"location,omitempty"`
	Categories []string     `json:"categories,omitempty"`
	ShowAs     string       `json:"showAs,omitempty"`
	IsAllDay   bool         `json:"isAllDay,omitempty"`
	Organizer  msOrganizer  `json:"organizer,omitempty"`
	Attendees  []msAttendee `json:"attendees,omitempty"`
	Recurrence interface{}  `json:"recurrence,omitempty"`
	ID         string       `json:"id,omitempty"`
}

type msBody struct {
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
}

type msDateTime struct {
	DateTime string `json:"dateTime"`
	TimeZone string `json:"timeZone"`
}

type msLocation struct {
	DisplayName string `json:"displayName,omitempty"`
}

type msOrganizer struct {
	EmailAddress msEmailAddress `json:"emailAddress"`
}

type msAttendee struct {
	Type         string         `json:"type,omitempty"`
	Status       msStatus       `json:"status,omitempty"`
	EmailAddress msEmailAddress `json:"emailAddress"`
}

type msStatus struct {
	Response string `json:"response,omitempty"`
	Time     string `json:"time,omitempty"`
}

type msEmailAddress struct {
	Name    string `json:"name,omitempty"`
	Address string `json:"address,omitempty"`
}

func toMicrosoftEvent(block calendarApp.TimeBlock) msEvent {
	// Include Orbita ID in subject for tracking
	subject := fmt.Sprintf("[%s] %s", block.ID.String(), block.Title)

	description := fmt.Sprintf("Type: %s", block.BlockType)
	if block.Completed {
		description += "\nStatus: Completed"
	} else if block.Missed {
		description += "\nStatus: Missed"
	}
	description += "\n\nManaged by Orbita"

	return msEvent{
		Subject: subject,
		Body: msBody{
			ContentType: "text",
			Content:     description,
		},
		Start: msDateTime{
			DateTime: block.StartTime.UTC().Format("2006-01-02T15:04:05"),
			TimeZone: "UTC",
		},
		End: msDateTime{
			DateTime: block.EndTime.UTC().Format("2006-01-02T15:04:05"),
			TimeZone: "UTC",
		},
		Categories: []string{"Orbita"},
		ShowAs:     "busy",
	}
}

func extractOrbitaID(subject string) string {
	// Extract UUID from subject like "[uuid] Title"
	if len(subject) < 38 || subject[0] != '[' {
		return ""
	}
	endIdx := bytes.IndexByte([]byte(subject), ']')
	if endIdx < 37 {
		return ""
	}
	return subject[1:endIdx]
}

func mapMicrosoftStatus(showAs string) string {
	switch showAs {
	case "free":
		return "free"
	case "tentative":
		return "tentative"
	case "busy", "oof", "workingElsewhere":
		return "confirmed"
	default:
		return "confirmed"
	}
}

func responseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("microsoft calendar API failed: status=%d body=%s", resp.StatusCode, string(body))
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
