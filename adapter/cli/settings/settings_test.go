package settings

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	googleCalendar "github.com/felixgeelhaar/orbita/internal/calendar/infrastructure/google"
	identitySettings "github.com/felixgeelhaar/orbita/internal/identity/application/settings"
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

func resetFlags() {
	calendarPrimaryOnly = false
	calendarListJSON = false
	settingsJSON = false
}

func TestCalendarListJSON(t *testing.T) {
	resetFlags()
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
	syncer := googleCalendar.NewSyncerWithBaseURL(stubTokenProvider{source: source}, nil, server.URL)

	app := &cli.App{
		CalendarSyncer: syncer,
		CurrentUserID:  uuid.New(),
	}
	cli.SetApp(app)
	defer cli.SetApp(nil)

	cmd := calendarListCmd
	cmd.SetContext(context.Background())
	cmd.SetOut(&strings.Builder{})
	calendarListJSON = true

	var output strings.Builder
	cmd.SetOut(&output)

	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("list failed: %v", err)
	}

	if !strings.Contains(output.String(), "\"primary\"") {
		t.Fatalf("expected JSON output, got: %s", output.String())
	}
}

func TestCalendarGetJSON(t *testing.T) {
	resetFlags()
	app := &cli.App{
		SettingsService: identitySettings.NewService(stubSettingsRepo{calendarID: "work"}),
		CurrentUserID:   uuid.New(),
	}
	cli.SetApp(app)
	defer cli.SetApp(nil)

	var output strings.Builder
	cmd := calendarGetCmd
	cmd.SetContext(context.Background())
	cmd.SetOut(&output)
	settingsJSON = true

	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("get failed: %v", err)
	}

	if !strings.Contains(output.String(), "\"calendar_id\"") {
		t.Fatalf("expected JSON output, got: %s", output.String())
	}
}

func TestDeleteMissingGetJSON(t *testing.T) {
	resetFlags()
	app := &cli.App{
		SettingsService: identitySettings.NewService(stubSettingsRepo{deleteMissing: true}),
		CurrentUserID:   uuid.New(),
	}
	cli.SetApp(app)
	defer cli.SetApp(nil)

	var output strings.Builder
	cmd := deleteMissingGetCmd
	cmd.SetContext(context.Background())
	cmd.SetOut(&output)
	settingsJSON = true

	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("get failed: %v", err)
	}

	if !strings.Contains(output.String(), "\"delete_missing\"") {
		t.Fatalf("expected JSON output, got: %s", output.String())
	}
}

func TestSettingsErrors(t *testing.T) {
	resetFlags()
	app := &cli.App{
		SettingsService: identitySettings.NewService(stubSettingsRepo{}),
		CurrentUserID:   uuid.Nil,
	}
	cli.SetApp(app)
	defer cli.SetApp(nil)

	cmd := calendarGetCmd
	cmd.SetContext(context.Background())

	if err := cmd.RunE(cmd, []string{}); err == nil {
		t.Fatalf("expected error for missing user")
	}

	cli.SetApp(&cli.App{CurrentUserID: uuid.New()})
	if err := calendarGetCmd.RunE(calendarGetCmd, []string{}); err == nil {
		t.Fatalf("expected error for missing settings service")
	}
}

func TestCalendarSetJSON(t *testing.T) {
	resetFlags()
	app := &cli.App{
		SettingsService: identitySettings.NewService(stubSettingsRepo{}),
		CurrentUserID:   uuid.New(),
	}
	cli.SetApp(app)
	defer cli.SetApp(nil)

	var output strings.Builder
	cmd := calendarSetCmd
	cmd.SetContext(context.Background())
	cmd.SetOut(&output)
	calendarID = "work"
	settingsJSON = true

	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	if !strings.Contains(output.String(), "\"calendar_id\"") {
		t.Fatalf("expected JSON output, got: %s", output.String())
	}
}

func TestDeleteMissingSetJSON(t *testing.T) {
	resetFlags()
	app := &cli.App{
		SettingsService: identitySettings.NewService(stubSettingsRepo{}),
		CurrentUserID:   uuid.New(),
	}
	cli.SetApp(app)
	defer cli.SetApp(nil)

	var output strings.Builder
	cmd := deleteMissingSetCmd
	cmd.SetContext(context.Background())
	cmd.SetOut(&output)
	deleteMissingValue = true
	settingsJSON = true

	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	if !strings.Contains(output.String(), "\"delete_missing\"") {
		t.Fatalf("expected JSON output, got: %s", output.String())
	}
}

type errTokenProvider struct{}

func (errTokenProvider) TokenSource(ctx context.Context, userID uuid.UUID) (oauth2.TokenSource, error) {
	return nil, errors.New("token error")
}

func TestCalendarListTokenError(t *testing.T) {
	resetFlags()
	syncer := googleCalendar.NewSyncer(errTokenProvider{}, nil)
	app := &cli.App{
		CalendarSyncer: syncer,
		CurrentUserID:  uuid.New(),
	}
	cli.SetApp(app)
	defer cli.SetApp(nil)

	cmd := calendarListCmd
	cmd.SetContext(context.Background())

	if err := cmd.RunE(cmd, []string{}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestCalendarListPrimaryOnly(t *testing.T) {
	resetFlags()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{"id": "primary", "summary": "Primary", "primary": true},
				{"id": "work", "summary": "Work", "primary": false},
			},
		})
	}))
	defer server.Close()

	source := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	syncer := googleCalendar.NewSyncerWithBaseURL(stubTokenProvider{source: source}, nil, server.URL)

	app := &cli.App{
		CalendarSyncer: syncer,
		CurrentUserID:  uuid.New(),
	}
	cli.SetApp(app)
	defer cli.SetApp(nil)

	var output strings.Builder
	cmd := calendarListCmd
	cmd.SetContext(context.Background())
	cmd.SetOut(&output)
	calendarPrimaryOnly = true

	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("list failed: %v", err)
	}

	if strings.Contains(output.String(), "work") {
		t.Fatalf("expected primary-only output, got: %s", output.String())
	}
}

func TestCalendarListOutput(t *testing.T) {
	resetFlags()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{"id": "primary", "summary": "Primary", "primary": true},
			},
		})
	}))
	defer server.Close()

	source := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	syncer := googleCalendar.NewSyncerWithBaseURL(stubTokenProvider{source: source}, nil, server.URL)

	app := &cli.App{
		CalendarSyncer: syncer,
		CurrentUserID:  uuid.New(),
	}
	cli.SetApp(app)
	defer cli.SetApp(nil)

	var output strings.Builder
	cmd := calendarListCmd
	cmd.SetContext(context.Background())
	cmd.SetOut(&output)

	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("list failed: %v", err)
	}

	if !strings.Contains(output.String(), "primary") {
		t.Fatalf("expected output to contain calendar ID, got: %s", output.String())
	}
}
