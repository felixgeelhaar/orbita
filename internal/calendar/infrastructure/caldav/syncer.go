package caldav

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	calendarApp "github.com/felixgeelhaar/orbita/internal/calendar/application"
	"github.com/emersion/go-ical"
	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/caldav"
	"github.com/google/uuid"
)

// Common CalDAV server URLs
const (
	AppleCalDAVURL    = "https://caldav.icloud.com"
	FastmailCalDAVURL = "https://caldav.fastmail.com"
)

// Custom property for Orbita events
const PropXOrbita = "X-ORBITA"

// Syncer syncs schedule blocks to a CalDAV calendar (Apple Calendar, Fastmail, Nextcloud, etc.).
type Syncer struct {
	baseURL       string
	username      string
	password      string // App-specific password for Apple
	calendarPath  string // Specific calendar path, or empty for default
	logger        *slog.Logger
	deleteMissing bool
}

// NewSyncer creates a CalDAV calendar syncer.
func NewSyncer(baseURL, username, password string, logger *slog.Logger) *Syncer {
	if logger == nil {
		logger = slog.Default()
	}
	return &Syncer{
		baseURL:       baseURL,
		username:      username,
		password:      password,
		logger:        logger,
		deleteMissing: false,
	}
}

// WithDeleteMissing enables deletion of events missing from the current sync set.
func (s *Syncer) WithDeleteMissing(enabled bool) *Syncer {
	s.deleteMissing = enabled
	return s
}

// WithCalendarPath sets the specific calendar path to use.
func (s *Syncer) WithCalendarPath(path string) *Syncer {
	s.calendarPath = path
	return s
}

// Sync pushes schedule blocks into the CalDAV calendar.
func (s *Syncer) Sync(ctx context.Context, userID uuid.UUID, blocks []calendarApp.TimeBlock) (*calendarApp.SyncResult, error) {
	client, err := s.getClient()
	if err != nil {
		return nil, err
	}

	// Find the calendar to use
	calPath, err := s.findCalendarPath(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to find calendar: %w", err)
	}

	result := &calendarApp.SyncResult{}
	keepPaths := make(map[string]struct{}, len(blocks))

	for _, block := range blocks {
		eventPath := fmt.Sprintf("%s%s.ics", calPath, block.ID.String())
		keepPaths[eventPath] = struct{}{}

		cal := toICalendar(block)
		updated, err := s.upsertEvent(ctx, client, eventPath, cal)
		if err != nil {
			s.logger.Warn("caldav sync failed", "event_path", eventPath, "error", err)
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
		deleted, err := s.deleteMissingEvents(ctx, client, calPath, keepPaths)
		if err != nil {
			s.logger.Warn("caldav delete missing failed", "error", err)
		} else {
			result.Deleted = deleted
		}
	}

	return result, nil
}

// ListCalendars returns calendars accessible to the user.
func (s *Syncer) ListCalendars(ctx context.Context, userID uuid.UUID) ([]calendarApp.Calendar, error) {
	client, err := s.getClient()
	if err != nil {
		return nil, err
	}

	principal, err := client.FindCurrentUserPrincipal(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find principal: %w", err)
	}

	homeSet, err := client.FindCalendarHomeSet(ctx, principal)
	if err != nil {
		return nil, fmt.Errorf("failed to find calendar home set: %w", err)
	}

	cals, err := client.FindCalendars(ctx, homeSet)
	if err != nil {
		return nil, fmt.Errorf("failed to find calendars: %w", err)
	}

	calendars := make([]calendarApp.Calendar, 0, len(cals))
	for i, cal := range cals {
		calendars = append(calendars, calendarApp.Calendar{
			ID:      cal.Path,
			Name:    cal.Name,
			Primary: i == 0, // First calendar is usually the default
		})
	}
	return calendars, nil
}

// ListEvents returns events within the given time range.
func (s *Syncer) ListEvents(ctx context.Context, userID uuid.UUID, start, end time.Time, onlyOrbitaEvents bool) ([]calendarApp.CalendarEvent, error) {
	client, err := s.getClient()
	if err != nil {
		return nil, err
	}

	calPath, err := s.findCalendarPath(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to find calendar: %w", err)
	}

	query := &caldav.CalendarQuery{
		CompRequest: caldav.CalendarCompRequest{
			Name:  "VCALENDAR",
			Props: []string{"VERSION"},
			Comps: []caldav.CalendarCompRequest{
				{
					Name:  "VEVENT",
					Props: []string{"SUMMARY", "DTSTART", "DTEND", "UID", "DESCRIPTION", "LOCATION", "STATUS", "ORGANIZER", "ATTENDEE", PropXOrbita},
				},
			},
		},
		CompFilter: caldav.CompFilter{
			Name: "VCALENDAR",
			Comps: []caldav.CompFilter{
				{
					Name:  "VEVENT",
					Start: start,
					End:   end,
				},
			},
		},
	}

	objects, err := client.QueryCalendar(ctx, calPath, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query calendar: %w", err)
	}

	events := make([]calendarApp.CalendarEvent, 0, len(objects))
	for _, obj := range objects {
		event, isOrbita := parseCalendarObject(&obj)
		if event == nil {
			continue
		}
		if onlyOrbitaEvents && !isOrbita {
			continue
		}
		events = append(events, *event)
	}

	return events, nil
}

// DeleteEvent deletes a calendar event by block ID.
func (s *Syncer) DeleteEvent(ctx context.Context, userID uuid.UUID, blockID uuid.UUID) error {
	client, err := s.getClient()
	if err != nil {
		return err
	}

	calPath, err := s.findCalendarPath(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to find calendar: %w", err)
	}

	eventPath := fmt.Sprintf("%s%s.ics", calPath, blockID.String())
	return client.RemoveAll(ctx, eventPath)
}

// Helper methods

func (s *Syncer) getClient() (*caldav.Client, error) {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &basicAuthTransport{
			username: s.username,
			password: s.password,
			base:     http.DefaultTransport,
		},
	}

	client, err := caldav.NewClient(webdav.HTTPClientWithBasicAuth(httpClient, s.username, s.password), s.baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create caldav client: %w", err)
	}
	return client, nil
}

func (s *Syncer) findCalendarPath(ctx context.Context, client *caldav.Client) (string, error) {
	if s.calendarPath != "" {
		return s.calendarPath, nil
	}

	principal, err := client.FindCurrentUserPrincipal(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to find principal: %w", err)
	}

	homeSet, err := client.FindCalendarHomeSet(ctx, principal)
	if err != nil {
		return "", fmt.Errorf("failed to find calendar home set: %w", err)
	}

	cals, err := client.FindCalendars(ctx, homeSet)
	if err != nil {
		return "", fmt.Errorf("failed to find calendars: %w", err)
	}

	if len(cals) == 0 {
		return "", fmt.Errorf("no calendars found")
	}

	// Use first calendar as default
	return cals[0].Path, nil
}

func (s *Syncer) upsertEvent(ctx context.Context, client *caldav.Client, eventPath string, cal *ical.Calendar) (bool, error) {
	// Check if event exists first
	_, err := client.GetCalendarObject(ctx, eventPath)
	exists := err == nil

	// Put the event (creates or updates)
	_, err = client.PutCalendarObject(ctx, eventPath, cal)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (s *Syncer) deleteMissingEvents(ctx context.Context, client *caldav.Client, calPath string, keepPaths map[string]struct{}) (int, error) {
	// Query all Orbita events
	query := &caldav.CalendarQuery{
		CompRequest: caldav.CalendarCompRequest{
			Name: "VCALENDAR",
			Comps: []caldav.CalendarCompRequest{
				{
					Name:  "VEVENT",
					Props: []string{"UID", PropXOrbita},
				},
			},
		},
		CompFilter: caldav.CompFilter{
			Name: "VCALENDAR",
			Comps: []caldav.CompFilter{
				{Name: "VEVENT"},
			},
		},
	}

	objects, err := client.QueryCalendar(ctx, calPath, query)
	if err != nil {
		return 0, err
	}

	deleted := 0
	for _, obj := range objects {
		// Check if it's an Orbita event
		if !isOrbitaEvent(&obj) {
			continue
		}

		if _, ok := keepPaths[obj.Path]; ok {
			continue
		}

		if err := client.RemoveAll(ctx, obj.Path); err != nil {
			s.logger.Warn("failed to delete caldav event", "path", obj.Path, "error", err)
			continue
		}
		deleted++
	}

	return deleted, nil
}

// isOrbitaEvent checks if a calendar object has the X-ORBITA property set.
func isOrbitaEvent(obj *caldav.CalendarObject) bool {
	if obj == nil || obj.Data == nil {
		return false
	}

	// Check VCALENDAR children for VEVENT components
	for _, child := range obj.Data.Children {
		if child.Name == ical.CompEvent {
			// Check for X-ORBITA property
			if props := child.Props[PropXOrbita]; len(props) > 0 {
				if props[0].Value == "1" {
					return true
				}
			}
		}
	}

	return false
}

// toICalendar converts a TimeBlock to an ical.Calendar.
func toICalendar(block calendarApp.TimeBlock) *ical.Calendar {
	cal := ical.NewCalendar()
	cal.Props.SetText(ical.PropVersion, "2.0")
	cal.Props.SetText(ical.PropProductID, "-//Orbita//Calendar Sync//EN")

	event := ical.NewEvent()
	event.Props.SetText(ical.PropUID, block.ID.String())
	event.Props.SetDateTime(ical.PropDateTimeStamp, time.Now().UTC())
	event.Props.SetDateTime(ical.PropDateTimeStart, block.StartTime.UTC())
	event.Props.SetDateTime(ical.PropDateTimeEnd, block.EndTime.UTC())
	event.Props.SetText(ical.PropSummary, block.Title)

	description := fmt.Sprintf("Type: %s", block.BlockType)
	if block.Completed {
		description += "\nStatus: Completed"
	} else if block.Missed {
		description += "\nStatus: Missed"
	}
	description += "\n\nManaged by Orbita"
	event.Props.SetText(ical.PropDescription, description)

	// Custom property to identify Orbita-created events
	orbitaProp := ical.NewProp(PropXOrbita)
	orbitaProp.Value = "1"
	event.Props[PropXOrbita] = []ical.Prop{*orbitaProp}

	cal.Children = append(cal.Children, event.Component)

	return cal
}

func parseCalendarObject(obj *caldav.CalendarObject) (*calendarApp.CalendarEvent, bool) {
	if obj == nil || obj.Data == nil {
		return nil, false
	}

	isOrbita := isOrbitaEvent(obj)

	event := &calendarApp.CalendarEvent{
		ID:            obj.Path,
		IsOrbitaEvent: isOrbita,
	}

	// Find the VEVENT component
	for _, child := range obj.Data.Children {
		if child.Name != ical.CompEvent {
			continue
		}

		// Extract properties from the VEVENT
		if props := child.Props[ical.PropSummary]; len(props) > 0 {
			event.Summary = props[0].Value
		}
		if props := child.Props[ical.PropDescription]; len(props) > 0 {
			event.Description = props[0].Value
		}
		if props := child.Props[ical.PropLocation]; len(props) > 0 {
			event.Location = props[0].Value
		}
		if props := child.Props[ical.PropStatus]; len(props) > 0 {
			event.Status = strings.ToLower(props[0].Value)
		}
		if props := child.Props[ical.PropUID]; len(props) > 0 {
			event.ID = props[0].Value
		}

		// Parse date/time properties
		icalEvent := &ical.Event{Component: child}
		if start, err := icalEvent.DateTimeStart(time.UTC); err == nil {
			event.StartTime = start
		}
		if end, err := icalEvent.DateTimeEnd(time.UTC); err == nil {
			event.EndTime = end
		}

		// Check for all-day events (both start at midnight)
		if event.StartTime.Hour() == 0 && event.StartTime.Minute() == 0 &&
			event.EndTime.Hour() == 0 && event.EndTime.Minute() == 0 {
			event.IsAllDay = true
		}

		break // Only process first VEVENT
	}

	return event, isOrbita
}

// calendarToString serializes an ical.Calendar to a string (for debugging).
func calendarToString(cal *ical.Calendar) string {
	var buf bytes.Buffer
	enc := ical.NewEncoder(&buf)
	if err := enc.Encode(cal); err != nil {
		return ""
	}
	return buf.String()
}

type basicAuthTransport struct {
	username string
	password string
	base     http.RoundTripper
}

func (t *basicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(t.username, t.password)
	return t.base.RoundTrip(req)
}
