package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	calendarApp "github.com/felixgeelhaar/orbita/internal/calendar/application"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

type stubTokenSourceProvider struct {
	source oauth2.TokenSource
	err    error
}

func (s stubTokenSourceProvider) TokenSource(ctx context.Context, userID uuid.UUID) (oauth2.TokenSource, error) {
	return s.source, s.err
}

func TestSyncer_Sync_CreateAndUpdate(t *testing.T) {
	updateID := uuid.New()
	createID := uuid.New()
	postCount := 0
	putCount := 0
	deleteCount := 0
	lastPath := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastPath = r.URL.Path
		if !strings.HasPrefix(r.URL.Path, "/calendars/primary/events") {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		switch r.Method {
		case http.MethodPost:
			postCount++
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if payload["id"] == updateID.String() {
				w.WriteHeader(http.StatusConflict)
				return
			}
			w.WriteHeader(http.StatusOK)
		case http.MethodPut:
			putCount++
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			if strings.Contains(r.URL.RawQuery, "privateExtendedProperty=orbita=1") {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"items": []map[string]any{
						{"id": updateID.String()},
						{"id": createID.String()},
						{"id": uuid.New().String()},
					},
				})
				return
			}
			w.WriteHeader(http.StatusNotFound)
		case http.MethodDelete:
			deleteCount++
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	source := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	syncer := NewSyncerWithBaseURL(stubTokenSourceProvider{source: source}, nil, server.URL).WithDeleteMissing(true)

	blocks := []calendarApp.TimeBlock{
		{
			ID:        updateID,
			Title:     "Update event",
			BlockType: "task",
			StartTime: time.Now().Add(1 * time.Hour),
			EndTime:   time.Now().Add(2 * time.Hour),
		},
		{
			ID:        createID,
			Title:     "Create event",
			BlockType: "habit",
			StartTime: time.Now().Add(3 * time.Hour),
			EndTime:   time.Now().Add(4 * time.Hour),
		},
	}

	result, err := syncer.Sync(context.Background(), uuid.New(), blocks)
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	if result.Created != 1 || result.Updated != 1 || result.Failed != 0 || result.Deleted != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if postCount != 2 || putCount != 1 || deleteCount != 1 {
		t.Fatalf("unexpected request counts: post=%d put=%d delete=%d", postCount, putCount, deleteCount)
	}
	if lastPath == "" {
		t.Fatalf("expected requests to hit calendar endpoints")
	}
}

func TestSyncer_Sync_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	source := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	syncer := NewSyncerWithBaseURL(stubTokenSourceProvider{source: source}, nil, server.URL)

	blocks := []calendarApp.TimeBlock{
		{
			ID:        uuid.New(),
			Title:     "Fail event",
			BlockType: "task",
			StartTime: time.Now().Add(1 * time.Hour),
			EndTime:   time.Now().Add(2 * time.Hour),
		},
	}

	result, err := syncer.Sync(context.Background(), uuid.New(), blocks)
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	if result.Failed != 1 || result.Created != 0 || result.Updated != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestSyncer_Sync_CustomCalendarID(t *testing.T) {
	calendarID := "custom-calendar"
	var seenPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	source := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	syncer := NewSyncerWithBaseURL(stubTokenSourceProvider{source: source}, nil, server.URL).
		WithCalendarID(calendarID)

	blocks := []calendarApp.TimeBlock{
		{
			ID:        uuid.New(),
			Title:     "Custom calendar",
			BlockType: "task",
			StartTime: time.Now().Add(1 * time.Hour),
			EndTime:   time.Now().Add(2 * time.Hour),
		},
	}

	_, err := syncer.Sync(context.Background(), uuid.New(), blocks)
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	expectedPrefix := "/calendars/" + calendarID + "/events"
	if !strings.HasPrefix(seenPath, expectedPrefix) {
		t.Fatalf("expected path prefix %q, got %q", expectedPrefix, seenPath)
	}
}

func TestSyncer_ListCalendars(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/me/calendarList" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{"id": "primary", "summary": "Primary", "primary": true},
				{"id": "work", "summary": "Work", "primary": false},
			},
		})
	}))
	defer server.Close()

	source := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	syncer := NewSyncerWithBaseURL(stubTokenSourceProvider{source: source}, nil, server.URL)

	calendars, err := syncer.ListCalendars(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(calendars) != 2 {
		t.Fatalf("unexpected calendars: %+v", calendars)
	}
	if calendars[0].ID != "primary" || !calendars[0].Primary {
		t.Fatalf("unexpected primary calendar: %+v", calendars[0])
	}
}

func TestSyncer_DeleteEvent(t *testing.T) {
	deleteID := uuid.New()
	deletePath := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		deletePath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	source := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	syncer := NewSyncerWithBaseURL(stubTokenSourceProvider{source: source}, nil, server.URL).
		WithCalendarID("primary")

	if err := syncer.DeleteEvent(context.Background(), uuid.New(), deleteID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	if !strings.Contains(deletePath, deleteID.String()) {
		t.Fatalf("expected delete path to include ID, got %q", deletePath)
	}
}

func TestSyncer_ListEvents(t *testing.T) {
	start := time.Date(2024, time.May, 2, 9, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	var seenQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/calendars/primary/events" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		seenQuery = r.URL.RawQuery
		query := r.URL.Query()
		if query.Get("timeMin") != start.UTC().Format(time.RFC3339) ||
			query.Get("timeMax") != end.UTC().Format(time.RFC3339) ||
			query.Get("singleEvents") != "true" ||
			query.Get("orderBy") != "startTime" ||
			query.Get("privateExtendedProperty") != "orbita=1" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{
					"id":      "event-1",
					"summary": "Imported",
					"start": map[string]any{
						"dateTime": "2024-05-02T09:00:00Z",
					},
					"end": map[string]any{
						"dateTime": "2024-05-02T10:00:00Z",
					},
				},
				{
					"id":      "event-2",
					"summary": "All Day",
					"start": map[string]any{
						"date": "2024-05-02",
					},
					"end": map[string]any{
						"date": "2024-05-02",
					},
				},
			},
		})
	}))
	defer server.Close()

	source := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	syncer := NewSyncerWithBaseURL(stubTokenSourceProvider{source: source}, nil, server.URL)

	events, err := syncer.ListEvents(context.Background(), uuid.New(), start, end, true)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got: %+v", events)
	}

	// Verify timed event
	if events[0].ID != "event-1" || events[0].Summary != "Imported" {
		t.Fatalf("unexpected timed event: %+v", events[0])
	}
	if events[0].IsAllDay {
		t.Fatalf("event-1 should not be all-day")
	}

	// Verify all-day event
	if events[1].ID != "event-2" || events[1].Summary != "All Day" {
		t.Fatalf("unexpected all-day event: %+v", events[1])
	}
	if !events[1].IsAllDay {
		t.Fatalf("event-2 should be all-day")
	}

	if seenQuery == "" {
		t.Fatalf("expected query parameters to be set")
	}
}

func TestSyncer_GoldenRequestPayload(t *testing.T) {
	expectedBytes, err := os.ReadFile(filepath.Join("testdata", "event_request.json"))
	if err != nil {
		t.Fatalf("failed to read golden file: %v", err)
	}

	var expected map[string]any
	if err := json.Unmarshal(expectedBytes, &expected); err != nil {
		t.Fatalf("invalid golden json: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var actual map[string]any
		if err := json.NewDecoder(r.Body).Decode(&actual); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if !reflect.DeepEqual(actual, expected) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	source := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	syncer := NewSyncerWithBaseURL(stubTokenSourceProvider{source: source}, nil, server.URL).
		WithAttendees([]string{"alice@example.com", "bob@example.com"}).
		WithReminders([]int{10, 30})

	start := time.Date(2024, time.May, 1, 9, 0, 0, 0, time.UTC)
	end := time.Date(2024, time.May, 1, 10, 0, 0, 0, time.UTC)
	blocks := []calendarApp.TimeBlock{
		{
			ID:        uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			Title:     "Deep Work",
			BlockType: "focus",
			StartTime: start,
			EndTime:   end,
			Completed: true,
		},
	}

	_, err = syncer.Sync(context.Background(), uuid.New(), blocks)
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}
}

func TestNewSyncer(t *testing.T) {
	source := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	provider := stubTokenSourceProvider{source: source}

	syncer := NewSyncer(provider, nil)

	if syncer == nil {
		t.Fatal("expected non-nil syncer")
	}
	if syncer.baseURL != defaultBaseURL {
		t.Errorf("expected baseURL %s, got %s", defaultBaseURL, syncer.baseURL)
	}
	if syncer.calendarID != "primary" {
		t.Errorf("expected calendarID 'primary', got %s", syncer.calendarID)
	}
	if syncer.deleteMissing {
		t.Error("expected deleteMissing to be false")
	}
}

func TestSyncer_ListEventsSimple(t *testing.T) {
	now := time.Now().UTC()
	eventID := uuid.New().String()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{
					"id":      eventID,
					"summary": "Test Event",
					"start": map[string]any{
						"dateTime": now.Add(1 * time.Hour).Format(time.RFC3339),
					},
					"end": map[string]any{
						"dateTime": now.Add(2 * time.Hour).Format(time.RFC3339),
					},
				},
			},
		})
	}))
	defer server.Close()

	source := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	syncer := NewSyncerWithBaseURL(stubTokenSourceProvider{source: source}, nil, server.URL)

	events, err := syncer.ListEventsSimple(context.Background(), uuid.New(), now, now.Add(24*time.Hour), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].ID != eventID {
		t.Errorf("expected event ID %s, got %s", eventID, events[0].ID)
	}
	if events[0].Summary != "Test Event" {
		t.Errorf("expected summary 'Test Event', got %s", events[0].Summary)
	}
}

func TestSyncer_ListEventsSimple_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	source := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	syncer := NewSyncerWithBaseURL(stubTokenSourceProvider{source: source}, nil, server.URL)

	now := time.Now().UTC()
	_, err := syncer.ListEventsSimple(context.Background(), uuid.New(), now, now.Add(24*time.Hour), false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSyncer_Sync_MissedBlock(t *testing.T) {
	var seenDescription string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if desc, ok := payload["description"].(string); ok {
			seenDescription = desc
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	source := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	syncer := NewSyncerWithBaseURL(stubTokenSourceProvider{source: source}, nil, server.URL)

	blocks := []calendarApp.TimeBlock{
		{
			ID:        uuid.New(),
			Title:     "Missed event",
			BlockType: "task",
			StartTime: time.Now().Add(1 * time.Hour),
			EndTime:   time.Now().Add(2 * time.Hour),
			Completed: false,
			Missed:    true,
		},
	}

	_, err := syncer.Sync(context.Background(), uuid.New(), blocks)
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	if !strings.Contains(seenDescription, "Missed") {
		t.Errorf("expected description to contain 'Missed', got %q", seenDescription)
	}
}

func TestSyncer_WithEmptyAttendees(t *testing.T) {
	var seenAttendees []any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if att, ok := payload["attendees"].([]any); ok {
			seenAttendees = att
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	source := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	syncer := NewSyncerWithBaseURL(stubTokenSourceProvider{source: source}, nil, server.URL).
		WithAttendees([]string{"valid@example.com", "", "another@example.com"})

	blocks := []calendarApp.TimeBlock{
		{
			ID:        uuid.New(),
			Title:     "Meeting",
			BlockType: "meeting",
			StartTime: time.Now().Add(1 * time.Hour),
			EndTime:   time.Now().Add(2 * time.Hour),
		},
	}

	_, err := syncer.Sync(context.Background(), uuid.New(), blocks)
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	// Empty emails should be filtered out, leaving only 2 valid attendees
	if len(seenAttendees) != 2 {
		t.Errorf("expected 2 attendees (empty filtered), got %d", len(seenAttendees))
	}
}

func TestSyncer_WithInvalidReminders(t *testing.T) {
	var seenReminders map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if rem, ok := payload["reminders"].(map[string]any); ok {
			seenReminders = rem
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	source := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	syncer := NewSyncerWithBaseURL(stubTokenSourceProvider{source: source}, nil, server.URL).
		WithReminders([]int{10, 0, -5, 30})

	blocks := []calendarApp.TimeBlock{
		{
			ID:        uuid.New(),
			Title:     "Meeting",
			BlockType: "meeting",
			StartTime: time.Now().Add(1 * time.Hour),
			EndTime:   time.Now().Add(2 * time.Hour),
		},
	}

	_, err := syncer.Sync(context.Background(), uuid.New(), blocks)
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	// Zero and negative reminders should be filtered out
	if overrides, ok := seenReminders["overrides"].([]any); ok {
		if len(overrides) != 2 {
			t.Errorf("expected 2 valid reminders, got %d", len(overrides))
		}
	}
}

func TestSyncer_UpdateAfterConflictFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(http.StatusConflict)
		case http.MethodPut:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	source := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	syncer := NewSyncerWithBaseURL(stubTokenSourceProvider{source: source}, nil, server.URL)

	blocks := []calendarApp.TimeBlock{
		{
			ID:        uuid.New(),
			Title:     "Update fail",
			BlockType: "task",
			StartTime: time.Now().Add(1 * time.Hour),
			EndTime:   time.Now().Add(2 * time.Hour),
		},
	}

	result, err := syncer.Sync(context.Background(), uuid.New(), blocks)
	if err != nil {
		t.Fatalf("sync should not return error: %v", err)
	}
	if result.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", result.Failed)
	}
}

func TestSyncer_DeleteMissingEvents_DeleteFailure(t *testing.T) {
	keepID := uuid.New()
	deleteID := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			if strings.Contains(r.URL.RawQuery, "privateExtendedProperty=orbita=1") {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"items": []map[string]any{
						{"id": keepID.String()},
						{"id": deleteID.String()},
					},
				})
				return
			}
			w.WriteHeader(http.StatusNotFound)
		case http.MethodDelete:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	source := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	syncer := NewSyncerWithBaseURL(stubTokenSourceProvider{source: source}, nil, server.URL).WithDeleteMissing(true)

	blocks := []calendarApp.TimeBlock{
		{
			ID:        keepID,
			Title:     "Keep",
			BlockType: "task",
			StartTime: time.Now().Add(1 * time.Hour),
			EndTime:   time.Now().Add(2 * time.Hour),
		},
	}

	result, err := syncer.Sync(context.Background(), uuid.New(), blocks)
	if err != nil {
		t.Fatalf("sync should not return error: %v", err)
	}
	// Delete failures don't cause sync to fail, they're logged
	if result.Created != 1 {
		t.Errorf("expected 1 created, got %d", result.Created)
	}
}

func TestSyncer_ListCalendars_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	source := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	syncer := NewSyncerWithBaseURL(stubTokenSourceProvider{source: source}, nil, server.URL)

	_, err := syncer.ListCalendars(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSyncer_ListCalendars_DecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	source := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	syncer := NewSyncerWithBaseURL(stubTokenSourceProvider{source: source}, nil, server.URL)

	_, err := syncer.ListCalendars(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSyncer_DeleteEvent_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	source := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	syncer := NewSyncerWithBaseURL(stubTokenSourceProvider{source: source}, nil, server.URL)

	err := syncer.DeleteEvent(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSyncer_ListEvents_AllEvents(t *testing.T) {
	// Test with onlyOrbitaEvents = false (all events)
	start := time.Date(2024, time.May, 2, 9, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify no orbita filter
		if strings.Contains(r.URL.RawQuery, "orbita") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{
					"id":      "event-1",
					"summary": "External Event",
					"start":   map[string]any{"dateTime": "2024-05-02T09:00:00Z"},
					"end":     map[string]any{"dateTime": "2024-05-02T10:00:00Z"},
				},
			},
		})
	}))
	defer server.Close()

	source := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	syncer := NewSyncerWithBaseURL(stubTokenSourceProvider{source: source}, nil, server.URL)

	events, err := syncer.ListEvents(context.Background(), uuid.New(), start, end, false)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].IsOrbitaEvent {
		t.Error("expected event to not be orbita event")
	}
}

func TestSyncer_ListEvents_WithOrbitaProperty(t *testing.T) {
	start := time.Date(2024, time.May, 2, 9, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{
					"id":      "event-1",
					"summary": "Orbita Block",
					"extendedProperties": map[string]any{
						"private": map[string]any{
							"orbita": "1",
						},
					},
					"start": map[string]any{"dateTime": "2024-05-02T09:00:00Z"},
					"end":   map[string]any{"dateTime": "2024-05-02T10:00:00Z"},
				},
			},
		})
	}))
	defer server.Close()

	source := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	syncer := NewSyncerWithBaseURL(stubTokenSourceProvider{source: source}, nil, server.URL)

	events, err := syncer.ListEvents(context.Background(), uuid.New(), start, end, false)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if !events[0].IsOrbitaEvent {
		t.Error("expected event to be orbita event")
	}
}

func TestSyncer_ListEvents_WithAttendees(t *testing.T) {
	start := time.Date(2024, time.May, 2, 9, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{
					"id":          "event-1",
					"summary":     "Meeting",
					"description": "Team sync",
					"location":    "Room A",
					"status":      "confirmed",
					"organizer":   map[string]any{"email": "organizer@example.com"},
					"attendees": []map[string]any{
						{"email": "alice@example.com"},
						{"email": "bob@example.com"},
					},
					"recurringEventId": "recurring-123",
					"start":            map[string]any{"dateTime": "2024-05-02T09:00:00Z"},
					"end":              map[string]any{"dateTime": "2024-05-02T10:00:00Z"},
				},
			},
		})
	}))
	defer server.Close()

	source := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	syncer := NewSyncerWithBaseURL(stubTokenSourceProvider{source: source}, nil, server.URL)

	events, err := syncer.ListEvents(context.Background(), uuid.New(), start, end, false)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Description != "Team sync" {
		t.Errorf("expected description 'Team sync', got %s", events[0].Description)
	}
	if events[0].Location != "Room A" {
		t.Errorf("expected location 'Room A', got %s", events[0].Location)
	}
	if events[0].Organizer != "organizer@example.com" {
		t.Errorf("expected organizer, got %s", events[0].Organizer)
	}
	if len(events[0].Attendees) != 2 {
		t.Errorf("expected 2 attendees, got %d", len(events[0].Attendees))
	}
	if !events[0].IsRecurring {
		t.Error("expected event to be recurring")
	}
}

func TestSyncer_ListEvents_InvalidTimes(t *testing.T) {
	start := time.Date(2024, time.May, 2, 9, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{
					"id":      "valid-event",
					"summary": "Valid",
					"start":   map[string]any{"dateTime": "2024-05-02T09:00:00Z"},
					"end":     map[string]any{"dateTime": "2024-05-02T10:00:00Z"},
				},
				{
					"id":      "invalid-start",
					"summary": "Invalid Start",
					"start":   map[string]any{"dateTime": "not-a-date"},
					"end":     map[string]any{"dateTime": "2024-05-02T10:00:00Z"},
				},
				{
					"id":      "invalid-end",
					"summary": "Invalid End",
					"start":   map[string]any{"dateTime": "2024-05-02T09:00:00Z"},
					"end":     map[string]any{"dateTime": "not-a-date"},
				},
				{
					"id":      "no-times",
					"summary": "No Times",
				},
			},
		})
	}))
	defer server.Close()

	source := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	syncer := NewSyncerWithBaseURL(stubTokenSourceProvider{source: source}, nil, server.URL)

	events, err := syncer.ListEvents(context.Background(), uuid.New(), start, end, false)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	// Only the valid event should be returned
	if len(events) != 1 {
		t.Fatalf("expected 1 valid event, got %d", len(events))
	}
	if events[0].ID != "valid-event" {
		t.Errorf("expected valid-event, got %s", events[0].ID)
	}
}

func TestSyncer_ListEvents_InvalidAllDayDates(t *testing.T) {
	start := time.Date(2024, time.May, 2, 9, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{
					"id":      "valid-allday",
					"summary": "Valid All Day",
					"start":   map[string]any{"date": "2024-05-02"},
					"end":     map[string]any{"date": "2024-05-03"},
				},
				{
					"id":      "invalid-start-date",
					"summary": "Invalid Start Date",
					"start":   map[string]any{"date": "not-a-date"},
					"end":     map[string]any{"date": "2024-05-03"},
				},
				{
					"id":      "invalid-end-date",
					"summary": "Invalid End Date",
					"start":   map[string]any{"date": "2024-05-02"},
					"end":     map[string]any{"date": "not-a-date"},
				},
			},
		})
	}))
	defer server.Close()

	source := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	syncer := NewSyncerWithBaseURL(stubTokenSourceProvider{source: source}, nil, server.URL)

	events, err := syncer.ListEvents(context.Background(), uuid.New(), start, end, false)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	// Only the valid all-day event should be returned
	if len(events) != 1 {
		t.Fatalf("expected 1 valid event, got %d", len(events))
	}
	if events[0].ID != "valid-allday" {
		t.Errorf("expected valid-allday, got %s", events[0].ID)
	}
	if !events[0].IsAllDay {
		t.Error("expected event to be all-day")
	}
}

func TestSyncer_Sync_NilOAuthService(t *testing.T) {
	syncer := &Syncer{
		oauthService: nil,
	}

	_, err := syncer.Sync(context.Background(), uuid.New(), nil)
	if err == nil {
		t.Fatal("expected error for nil oauth service")
	}
	if !strings.Contains(err.Error(), "oauth service not configured") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSyncer_ListCalendars_NilOAuthService(t *testing.T) {
	syncer := &Syncer{
		oauthService: nil,
	}

	_, err := syncer.ListCalendars(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error for nil oauth service")
	}
}

func TestSyncer_DeleteEvent_NilOAuthService(t *testing.T) {
	syncer := &Syncer{
		oauthService: nil,
	}

	err := syncer.DeleteEvent(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected error for nil oauth service")
	}
}

func TestSyncer_ListEvents_NilOAuthService(t *testing.T) {
	syncer := &Syncer{
		oauthService: nil,
	}

	_, err := syncer.ListEvents(context.Background(), uuid.New(), time.Now(), time.Now().Add(time.Hour), false)
	if err == nil {
		t.Fatal("expected error for nil oauth service")
	}
}

func TestSyncer_TokenSourceError(t *testing.T) {
	provider := stubTokenSourceProvider{
		source: nil,
		err:    fmt.Errorf("token error"),
	}
	syncer := NewSyncer(provider, nil)

	t.Run("Sync", func(t *testing.T) {
		_, err := syncer.Sync(context.Background(), uuid.New(), nil)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("ListCalendars", func(t *testing.T) {
		_, err := syncer.ListCalendars(context.Background(), uuid.New())
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("DeleteEvent", func(t *testing.T) {
		err := syncer.DeleteEvent(context.Background(), uuid.New(), uuid.New())
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("ListEvents", func(t *testing.T) {
		_, err := syncer.ListEvents(context.Background(), uuid.New(), time.Now(), time.Now().Add(time.Hour), false)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestNewSyncerWithBaseURL_EmptyURL(t *testing.T) {
	source := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	syncer := NewSyncerWithBaseURL(stubTokenSourceProvider{source: source}, nil, "")

	if syncer.baseURL != defaultBaseURL {
		t.Errorf("expected default URL %s, got %s", defaultBaseURL, syncer.baseURL)
	}
}

func TestSyncer_WithCalendarID_Empty(t *testing.T) {
	source := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	syncer := NewSyncer(stubTokenSourceProvider{source: source}, nil)

	// Empty string should not change calendar ID
	syncer.WithCalendarID("")
	if syncer.calendarID != "primary" {
		t.Errorf("expected 'primary', got %s", syncer.calendarID)
	}
}
