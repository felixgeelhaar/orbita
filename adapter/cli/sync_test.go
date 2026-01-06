package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	googleCalendar "github.com/felixgeelhaar/orbita/internal/calendar/infrastructure/google"
	identitySettings "github.com/felixgeelhaar/orbita/internal/identity/application/settings"
	scheduleQueries "github.com/felixgeelhaar/orbita/internal/scheduling/application/queries"
	scheduleDomain "github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

type stubTokenProvider struct {
	source oauth2.TokenSource
}

func (s stubTokenProvider) TokenSource(ctx context.Context, userID uuid.UUID) (oauth2.TokenSource, error) {
	return s.source, nil
}

type stubSettingsRepo struct {
	calendarID    string
	deleteMissing bool
}

func (s stubSettingsRepo) GetCalendarID(ctx context.Context, userID uuid.UUID) (string, error) {
	return s.calendarID, nil
}

func (s stubSettingsRepo) SetCalendarID(ctx context.Context, userID uuid.UUID, calendarID string) error {
	return nil
}

func (s stubSettingsRepo) GetDeleteMissing(ctx context.Context, userID uuid.UUID) (bool, error) {
	return s.deleteMissing, nil
}

func (s stubSettingsRepo) SetDeleteMissing(ctx context.Context, userID uuid.UUID, deleteMissing bool) error {
	return nil
}

type stubScheduleRepo struct {
	schedule *scheduleDomain.Schedule
}

func (s stubScheduleRepo) Save(ctx context.Context, schedule *scheduleDomain.Schedule) error {
	return nil
}

func (s stubScheduleRepo) FindByID(ctx context.Context, id uuid.UUID) (*scheduleDomain.Schedule, error) {
	return s.schedule, nil
}

func (s stubScheduleRepo) FindByUserAndDate(ctx context.Context, userID uuid.UUID, date time.Time) (*scheduleDomain.Schedule, error) {
	return s.schedule, nil
}

func (s stubScheduleRepo) FindByUserDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]*scheduleDomain.Schedule, error) {
	return []*scheduleDomain.Schedule{s.schedule}, nil
}

func (s stubScheduleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func TestSyncCommand_UsesSettings(t *testing.T) {
	userID := uuid.New()
	calendarID := "work"
	deleteMissingCalled := 0
	lastPath := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastPath = r.URL.Path
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			if r.URL.Path != "/calendars/"+calendarID+"/events" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			if r.URL.RawQuery == "privateExtendedProperty=orbita=1" {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"items": []map[string]any{
						{"id": "missing"},
					},
				})
				return
			}
			w.WriteHeader(http.StatusNotFound)
		case http.MethodDelete:
			deleteMissingCalled++
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	source := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	syncer := googleCalendar.NewSyncerWithBaseURL(stubTokenProvider{source: source}, nil, server.URL)

	now := time.Now()
	schedule := scheduleDomain.NewSchedule(userID, now)
	_, err := schedule.AddBlock(
		scheduleDomain.BlockTypeTask,
		uuid.New(),
		"Test Block",
		now.Add(1*time.Hour),
		now.Add(2*time.Hour),
	)
	if err != nil {
		t.Fatalf("failed to add block: %v", err)
	}

	scheduleHandler := scheduleQueries.NewGetScheduleHandler(stubScheduleRepo{schedule: schedule})
	settingsService := identitySettings.NewService(stubSettingsRepo{
		calendarID:    calendarID,
		deleteMissing: true,
	})

	app := &App{
		GetScheduleHandler: scheduleHandler,
		CalendarSyncer:     syncer,
		SettingsService:    settingsService,
		CurrentUserID:      userID,
	}

	SetApp(app)
	defer SetApp(nil)

	syncDays = 1
	syncDeleteMissing = false
	syncCalendarID = ""
	syncUseConfigCalendar = true

	cmd := syncCmd
	cmd.SetContext(context.Background())

	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	if deleteMissingCalled == 0 {
		t.Fatalf("expected delete-missing to be invoked")
	}
	if lastPath == "" || lastPath[:len("/calendars/"+calendarID)] != "/calendars/"+calendarID {
		t.Fatalf("expected calendar ID %q in path, got %q", calendarID, lastPath)
	}
}
