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

func TestCalendarSetMissingCalendar(t *testing.T) {
	resetFlags()
	app := &cli.App{
		SettingsService: identitySettings.NewService(stubSettingsRepo{}),
		CurrentUserID:   uuid.New(),
	}
	cli.SetApp(app)
	defer cli.SetApp(nil)

	cmd := calendarSetCmd
	cmd.SetContext(context.Background())
	calendarID = ""

	if err := cmd.RunE(cmd, []string{}); err == nil {
		t.Fatalf("expected error for missing calendar")
	}
}

func TestCalendarGetPlainOutput(t *testing.T) {
	resetFlags()
	app := &cli.App{
		SettingsService: identitySettings.NewService(stubSettingsRepo{calendarID: "work@example.com"}),
		CurrentUserID:   uuid.New(),
	}
	cli.SetApp(app)
	defer cli.SetApp(nil)

	var output strings.Builder
	cmd := calendarGetCmd
	cmd.SetContext(context.Background())
	cmd.SetOut(&output)

	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("get failed: %v", err)
	}

	if !strings.Contains(output.String(), "work@example.com") {
		t.Fatalf("expected plain text output, got: %s", output.String())
	}
}

func TestCalendarGetPrimaryDefault(t *testing.T) {
	resetFlags()
	app := &cli.App{
		SettingsService: identitySettings.NewService(stubSettingsRepo{calendarID: ""}),
		CurrentUserID:   uuid.New(),
	}
	cli.SetApp(app)
	defer cli.SetApp(nil)

	var output strings.Builder
	cmd := calendarGetCmd
	cmd.SetContext(context.Background())
	cmd.SetOut(&output)

	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("get failed: %v", err)
	}

	if !strings.Contains(output.String(), "primary") {
		t.Fatalf("expected 'primary' as default, got: %s", output.String())
	}
}

func TestDeleteMissingGetPlainOutput(t *testing.T) {
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

	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("get failed: %v", err)
	}

	if !strings.Contains(output.String(), "true") {
		t.Fatalf("expected plain text output, got: %s", output.String())
	}
}

func TestCalendarSetPlainOutput(t *testing.T) {
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

	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	if !strings.Contains(output.String(), "Calendar ID saved") {
		t.Fatalf("expected confirmation message, got: %s", output.String())
	}
}

func TestDeleteMissingSetPlainOutput(t *testing.T) {
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

	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	if !strings.Contains(output.String(), "Delete-missing preference saved") {
		t.Fatalf("expected confirmation message, got: %s", output.String())
	}
}

func TestCalendarListNoApp(t *testing.T) {
	resetFlags()
	cli.SetApp(nil)

	cmd := calendarListCmd
	cmd.SetContext(context.Background())

	err := cmd.RunE(cmd, []string{})
	if err == nil {
		t.Fatalf("expected error for nil app")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("expected not configured error, got: %v", err)
	}
}

func TestCalendarListEmptyCalendars(t *testing.T) {
	resetFlags()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{},
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

	err := cmd.RunE(cmd, []string{})
	if err != nil {
		t.Fatalf("expected no error for empty list: %v", err)
	}
}

func TestCalendarListNoUserID(t *testing.T) {
	resetFlags()
	source := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	syncer := googleCalendar.NewSyncerWithBaseURL(stubTokenProvider{source: source}, nil, "http://localhost")

	app := &cli.App{
		CalendarSyncer: syncer,
		CurrentUserID:  uuid.Nil,
	}
	cli.SetApp(app)
	defer cli.SetApp(nil)

	cmd := calendarListCmd
	cmd.SetContext(context.Background())

	err := cmd.RunE(cmd, []string{})
	if err == nil {
		t.Fatalf("expected error for missing user")
	}
	if !strings.Contains(err.Error(), "current user not configured") {
		t.Fatalf("expected user not configured error, got: %v", err)
	}
}

func TestDeleteMissingGetNoApp(t *testing.T) {
	resetFlags()
	cli.SetApp(nil)

	cmd := deleteMissingGetCmd
	cmd.SetContext(context.Background())

	err := cmd.RunE(cmd, []string{})
	if err == nil {
		t.Fatalf("expected error for nil app")
	}
}

func TestDeleteMissingSetNoApp(t *testing.T) {
	resetFlags()
	cli.SetApp(nil)

	cmd := deleteMissingSetCmd
	cmd.SetContext(context.Background())

	err := cmd.RunE(cmd, []string{})
	if err == nil {
		t.Fatalf("expected error for nil app")
	}
}

func TestDeleteMissingGetNoUser(t *testing.T) {
	resetFlags()
	app := &cli.App{
		SettingsService: identitySettings.NewService(stubSettingsRepo{}),
		CurrentUserID:   uuid.Nil,
	}
	cli.SetApp(app)
	defer cli.SetApp(nil)

	cmd := deleteMissingGetCmd
	cmd.SetContext(context.Background())

	err := cmd.RunE(cmd, []string{})
	if err == nil {
		t.Fatalf("expected error for missing user")
	}
}

func TestDeleteMissingSetNoUser(t *testing.T) {
	resetFlags()
	app := &cli.App{
		SettingsService: identitySettings.NewService(stubSettingsRepo{}),
		CurrentUserID:   uuid.Nil,
	}
	cli.SetApp(app)
	defer cli.SetApp(nil)

	cmd := deleteMissingSetCmd
	cmd.SetContext(context.Background())

	err := cmd.RunE(cmd, []string{})
	if err == nil {
		t.Fatalf("expected error for missing user")
	}
}

func TestCalendarSetNoUser(t *testing.T) {
	resetFlags()
	app := &cli.App{
		SettingsService: identitySettings.NewService(stubSettingsRepo{}),
		CurrentUserID:   uuid.Nil,
	}
	cli.SetApp(app)
	defer cli.SetApp(nil)

	cmd := calendarSetCmd
	cmd.SetContext(context.Background())
	calendarID = "work"

	err := cmd.RunE(cmd, []string{})
	if err == nil {
		t.Fatalf("expected error for missing user")
	}
}
