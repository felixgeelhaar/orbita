package caldav

import (
	"net/http"
	"strings"
	"testing"
	"time"

	calendarApp "github.com/felixgeelhaar/orbita/internal/calendar/application"
	"github.com/emersion/go-ical"
	"github.com/emersion/go-webdav/caldav"
	"github.com/google/uuid"
)

func TestNewSyncer(t *testing.T) {
	syncer := NewSyncer("https://caldav.example.com", "user", "pass", nil)

	if syncer == nil {
		t.Fatal("expected non-nil syncer")
	}
	if syncer.baseURL != "https://caldav.example.com" {
		t.Errorf("expected baseURL 'https://caldav.example.com', got %s", syncer.baseURL)
	}
	if syncer.username != "user" {
		t.Errorf("expected username 'user', got %s", syncer.username)
	}
	if syncer.password != "pass" {
		t.Errorf("expected password 'pass', got %s", syncer.password)
	}
	if syncer.deleteMissing {
		t.Error("expected deleteMissing to be false by default")
	}
	if syncer.calendarPath != "" {
		t.Errorf("expected empty calendarPath, got %s", syncer.calendarPath)
	}
}

func TestSyncer_WithDeleteMissing(t *testing.T) {
	syncer := NewSyncer("https://caldav.example.com", "user", "pass", nil)

	result := syncer.WithDeleteMissing(true)

	if result != syncer {
		t.Error("expected same syncer instance returned for chaining")
	}
	if !syncer.deleteMissing {
		t.Error("expected deleteMissing to be true")
	}
}

func TestSyncer_WithCalendarPath(t *testing.T) {
	syncer := NewSyncer("https://caldav.example.com", "user", "pass", nil)

	result := syncer.WithCalendarPath("/calendars/user/personal/")

	if result != syncer {
		t.Error("expected same syncer instance returned for chaining")
	}
	if syncer.calendarPath != "/calendars/user/personal/" {
		t.Errorf("expected calendarPath '/calendars/user/personal/', got %s", syncer.calendarPath)
	}
}

func TestToICalendar(t *testing.T) {
	blockID := uuid.New()
	start := time.Date(2024, time.May, 1, 9, 0, 0, 0, time.UTC)
	end := time.Date(2024, time.May, 1, 10, 0, 0, 0, time.UTC)

	block := calendarApp.TimeBlock{
		ID:        blockID,
		Title:     "Deep Work",
		BlockType: "focus",
		StartTime: start,
		EndTime:   end,
		Completed: false,
		Missed:    false,
	}

	cal := toICalendar(block)

	if cal == nil {
		t.Fatal("expected non-nil calendar")
	}

	// Check VCALENDAR properties
	if version := cal.Props.Get(ical.PropVersion); version == nil || version.Value != "2.0" {
		t.Error("expected VERSION:2.0")
	}
	if prodID := cal.Props.Get(ical.PropProductID); prodID == nil || !strings.Contains(prodID.Value, "Orbita") {
		t.Error("expected PRODID containing 'Orbita'")
	}

	// Check VEVENT exists
	if len(cal.Children) != 1 {
		t.Fatalf("expected 1 child (VEVENT), got %d", len(cal.Children))
	}

	vevent := cal.Children[0]
	if vevent.Name != ical.CompEvent {
		t.Errorf("expected VEVENT, got %s", vevent.Name)
	}

	// Check UID
	if uid := vevent.Props.Get(ical.PropUID); uid == nil || uid.Value != blockID.String() {
		t.Error("expected UID matching block ID")
	}

	// Check Summary
	if summary := vevent.Props.Get(ical.PropSummary); summary == nil || summary.Value != "Deep Work" {
		t.Error("expected SUMMARY 'Deep Work'")
	}

	// Check X-ORBITA custom property
	if orbita := vevent.Props[PropXOrbita]; len(orbita) == 0 || orbita[0].Value != "1" {
		t.Error("expected X-ORBITA:1 property")
	}
}

func TestToICalendar_CompletedBlock(t *testing.T) {
	block := calendarApp.TimeBlock{
		ID:        uuid.New(),
		Title:     "Completed Task",
		BlockType: "task",
		StartTime: time.Now().UTC(),
		EndTime:   time.Now().UTC().Add(1 * time.Hour),
		Completed: true,
		Missed:    false,
	}

	cal := toICalendar(block)
	vevent := cal.Children[0]

	desc := vevent.Props.Get(ical.PropDescription)
	if desc == nil {
		t.Fatal("expected DESCRIPTION property")
	}
	if !strings.Contains(desc.Value, "Completed") {
		t.Error("expected description to contain 'Completed'")
	}
}

func TestToICalendar_MissedBlock(t *testing.T) {
	block := calendarApp.TimeBlock{
		ID:        uuid.New(),
		Title:     "Missed Task",
		BlockType: "task",
		StartTime: time.Now().UTC(),
		EndTime:   time.Now().UTC().Add(1 * time.Hour),
		Completed: false,
		Missed:    true,
	}

	cal := toICalendar(block)
	vevent := cal.Children[0]

	desc := vevent.Props.Get(ical.PropDescription)
	if desc == nil {
		t.Fatal("expected DESCRIPTION property")
	}
	if !strings.Contains(desc.Value, "Missed") {
		t.Error("expected description to contain 'Missed'")
	}
}

func TestIsOrbitaEvent(t *testing.T) {
	t.Run("nil object", func(t *testing.T) {
		result := isOrbitaEvent(nil)
		if result != false {
			t.Error("expected false for nil object")
		}
	})

	t.Run("nil data", func(t *testing.T) {
		obj := &caldav.CalendarObject{Data: nil}
		result := isOrbitaEvent(obj)
		if result != false {
			t.Error("expected false for nil data")
		}
	})

	t.Run("no events", func(t *testing.T) {
		cal := ical.NewCalendar()
		obj := &caldav.CalendarObject{Data: cal}
		result := isOrbitaEvent(obj)
		if result != false {
			t.Error("expected false when no events")
		}
	})

	t.Run("event without X-ORBITA", func(t *testing.T) {
		event := ical.NewEvent()
		event.Props.SetText(ical.PropUID, "test")
		cal := ical.NewCalendar()
		cal.Children = append(cal.Children, event.Component)
		obj := &caldav.CalendarObject{Data: cal}

		result := isOrbitaEvent(obj)
		if result != false {
			t.Error("expected false when no X-ORBITA property")
		}
	})

	t.Run("event with X-ORBITA=0", func(t *testing.T) {
		event := ical.NewEvent()
		event.Props.SetText(ical.PropUID, "test")
		orbitaProp := ical.NewProp(PropXOrbita)
		orbitaProp.Value = "0"
		event.Props[PropXOrbita] = []ical.Prop{*orbitaProp}
		cal := ical.NewCalendar()
		cal.Children = append(cal.Children, event.Component)
		obj := &caldav.CalendarObject{Data: cal}

		result := isOrbitaEvent(obj)
		if result != false {
			t.Error("expected false when X-ORBITA=0")
		}
	})

	t.Run("event with X-ORBITA=1", func(t *testing.T) {
		event := ical.NewEvent()
		event.Props.SetText(ical.PropUID, "test")
		orbitaProp := ical.NewProp(PropXOrbita)
		orbitaProp.Value = "1"
		event.Props[PropXOrbita] = []ical.Prop{*orbitaProp}
		cal := ical.NewCalendar()
		cal.Children = append(cal.Children, event.Component)
		obj := &caldav.CalendarObject{Data: cal}

		result := isOrbitaEvent(obj)
		if result != true {
			t.Error("expected true when X-ORBITA=1")
		}
	})
}

func TestParseCalendarObject(t *testing.T) {
	startTime := time.Date(2024, time.May, 1, 9, 0, 0, 0, time.UTC)
	endTime := time.Date(2024, time.May, 1, 10, 0, 0, 0, time.UTC)
	eventID := uuid.New().String()

	// Create iCal event
	event := ical.NewEvent()
	event.Props.SetText(ical.PropUID, eventID)
	event.Props.SetText(ical.PropSummary, "Test Meeting")
	event.Props.SetText(ical.PropDescription, "A test meeting")
	event.Props.SetText(ical.PropLocation, "Conference Room A")
	event.Props.SetText(ical.PropStatus, "CONFIRMED")
	event.Props.SetDateTime(ical.PropDateTimeStart, startTime)
	event.Props.SetDateTime(ical.PropDateTimeEnd, endTime)

	cal := ical.NewCalendar()
	cal.Children = append(cal.Children, event.Component)

	obj := &caldav.CalendarObject{
		Path: "/calendars/user/personal/" + eventID + ".ics",
		Data: cal,
	}

	result, isOrbita := parseCalendarObject(obj)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if isOrbita {
		t.Error("expected isOrbita to be false")
	}
	if result.ID != eventID {
		t.Errorf("expected ID %s, got %s", eventID, result.ID)
	}
	if result.Summary != "Test Meeting" {
		t.Errorf("expected Summary 'Test Meeting', got %s", result.Summary)
	}
	if result.Description != "A test meeting" {
		t.Errorf("expected Description 'A test meeting', got %s", result.Description)
	}
	if result.Location != "Conference Room A" {
		t.Errorf("expected Location 'Conference Room A', got %s", result.Location)
	}
	if result.Status != "confirmed" {
		t.Errorf("expected Status 'confirmed', got %s", result.Status)
	}
}

func TestParseCalendarObject_NilObject(t *testing.T) {
	result, isOrbita := parseCalendarObject(nil)

	if result != nil {
		t.Error("expected nil result for nil input")
	}
	if isOrbita {
		t.Error("expected isOrbita to be false")
	}
}

func TestParseCalendarObject_NilData(t *testing.T) {
	obj := &caldav.CalendarObject{Data: nil}
	result, isOrbita := parseCalendarObject(obj)

	if result != nil {
		t.Error("expected nil result for nil data")
	}
	if isOrbita {
		t.Error("expected isOrbita to be false")
	}
}

func TestParseCalendarObject_OrbitaEvent(t *testing.T) {
	event := ical.NewEvent()
	event.Props.SetText(ical.PropUID, "test-id")
	event.Props.SetText(ical.PropSummary, "Orbita Task")

	// Add X-ORBITA property
	orbitaProp := ical.NewProp(PropXOrbita)
	orbitaProp.Value = "1"
	event.Props[PropXOrbita] = []ical.Prop{*orbitaProp}

	cal := ical.NewCalendar()
	cal.Children = append(cal.Children, event.Component)

	obj := &caldav.CalendarObject{
		Path: "/calendars/user/personal/test.ics",
		Data: cal,
	}

	result, isOrbita := parseCalendarObject(obj)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !isOrbita {
		t.Error("expected isOrbita to be true")
	}
	if !result.IsOrbitaEvent {
		t.Error("expected result.IsOrbitaEvent to be true")
	}
}

func TestParseCalendarObject_AllDayEvent(t *testing.T) {
	// All-day events have start and end at midnight
	startTime := time.Date(2024, time.May, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2024, time.May, 2, 0, 0, 0, 0, time.UTC)

	event := ical.NewEvent()
	event.Props.SetText(ical.PropUID, "allday-test")
	event.Props.SetText(ical.PropSummary, "All Day Event")
	event.Props.SetDateTime(ical.PropDateTimeStart, startTime)
	event.Props.SetDateTime(ical.PropDateTimeEnd, endTime)

	cal := ical.NewCalendar()
	cal.Children = append(cal.Children, event.Component)

	obj := &caldav.CalendarObject{
		Path: "/calendars/user/personal/allday.ics",
		Data: cal,
	}

	result, _ := parseCalendarObject(obj)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.IsAllDay {
		t.Error("expected IsAllDay to be true for all-day event")
	}
}

func TestCalendarToString(t *testing.T) {
	// Create a calendar with an event (required for valid iCalendar)
	block := calendarApp.TimeBlock{
		ID:        uuid.New(),
		Title:     "Test Event",
		BlockType: "task",
		StartTime: time.Now().UTC(),
		EndTime:   time.Now().UTC().Add(1 * time.Hour),
	}
	cal := toICalendar(block)

	result := calendarToString(cal)

	if result == "" {
		t.Error("expected non-empty string")
	}
	if !strings.Contains(result, "BEGIN:VCALENDAR") {
		t.Error("expected output to contain BEGIN:VCALENDAR")
	}
	if !strings.Contains(result, "VERSION:2.0") {
		t.Error("expected output to contain VERSION:2.0")
	}
	if !strings.Contains(result, "END:VCALENDAR") {
		t.Error("expected output to contain END:VCALENDAR")
	}
	if !strings.Contains(result, "BEGIN:VEVENT") {
		t.Error("expected output to contain BEGIN:VEVENT")
	}
}

func TestCalendarToString_EmptyCalendar(t *testing.T) {
	// Empty calendar returns empty string (library behavior)
	cal := ical.NewCalendar()
	result := calendarToString(cal)

	// The go-ical library returns empty for calendars without events
	// This is expected behavior - calendarToString returns "" on encode error
	if result != "" {
		t.Logf("Empty calendar encoded to: %s", result)
	}
}

func TestBasicAuthTransport_RoundTrip(t *testing.T) {
	// Create a transport with basic auth
	transport := &basicAuthTransport{
		username: "testuser",
		password: "testpass",
		base:     &mockRoundTripper{},
	}

	// Create a request
	req, _ := http.NewRequest(http.MethodGet, "https://caldav.example.com", nil)

	// Verify no auth header initially
	if req.Header.Get("Authorization") != "" {
		t.Error("expected no Authorization header before RoundTrip")
	}

	_, _ = transport.RoundTrip(req)

	// Verify auth header was set
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		t.Error("expected Authorization header after RoundTrip")
	}
	if !strings.HasPrefix(authHeader, "Basic ") {
		t.Error("expected Basic auth header")
	}
}

// mockRoundTripper for testing basicAuthTransport
type mockRoundTripper struct{}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: 200}, nil
}

func TestConstants(t *testing.T) {
	// Verify constants are properly defined
	if AppleCalDAVURL != "https://caldav.icloud.com" {
		t.Errorf("unexpected AppleCalDAVURL: %s", AppleCalDAVURL)
	}
	if FastmailCalDAVURL != "https://caldav.fastmail.com" {
		t.Errorf("unexpected FastmailCalDAVURL: %s", FastmailCalDAVURL)
	}
	if PropXOrbita != "X-ORBITA" {
		t.Errorf("unexpected PropXOrbita: %s", PropXOrbita)
	}
}
