package google

import (
	"context"
	"encoding/json"
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
