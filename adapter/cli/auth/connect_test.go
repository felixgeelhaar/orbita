package auth

import (
	"bufio"
	"context"
	"errors"
	"strings"
	"testing"

	calendarApp "github.com/felixgeelhaar/orbita/internal/calendar/application"
	calendarDomain "github.com/felixgeelhaar/orbita/internal/calendar/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock implementations for testing

type mockCalendarRepo struct {
	mock.Mock
}

func (m *mockCalendarRepo) Save(ctx context.Context, cal *calendarDomain.ConnectedCalendar) error {
	args := m.Called(ctx, cal)
	return args.Error(0)
}

func (m *mockCalendarRepo) FindByID(ctx context.Context, id uuid.UUID) (*calendarDomain.ConnectedCalendar, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*calendarDomain.ConnectedCalendar), args.Error(1)
}

func (m *mockCalendarRepo) FindByUser(ctx context.Context, userID uuid.UUID) ([]*calendarDomain.ConnectedCalendar, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*calendarDomain.ConnectedCalendar), args.Error(1)
}

func (m *mockCalendarRepo) FindByUserAndProvider(ctx context.Context, userID uuid.UUID, provider calendarDomain.ProviderType) ([]*calendarDomain.ConnectedCalendar, error) {
	args := m.Called(ctx, userID, provider)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*calendarDomain.ConnectedCalendar), args.Error(1)
}

func (m *mockCalendarRepo) FindByUserProviderAndCalendar(ctx context.Context, userID uuid.UUID, provider calendarDomain.ProviderType, calendarID string) (*calendarDomain.ConnectedCalendar, error) {
	args := m.Called(ctx, userID, provider, calendarID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*calendarDomain.ConnectedCalendar), args.Error(1)
}

func (m *mockCalendarRepo) FindPrimaryForUser(ctx context.Context, userID uuid.UUID) (*calendarDomain.ConnectedCalendar, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*calendarDomain.ConnectedCalendar), args.Error(1)
}

func (m *mockCalendarRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockCalendarRepo) DeleteByUserAndProvider(ctx context.Context, userID uuid.UUID, provider calendarDomain.ProviderType) error {
	args := m.Called(ctx, userID, provider)
	return args.Error(0)
}

func (m *mockCalendarRepo) FindEnabledPullCalendars(ctx context.Context, userID uuid.UUID) ([]*calendarDomain.ConnectedCalendar, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*calendarDomain.ConnectedCalendar), args.Error(1)
}

func (m *mockCalendarRepo) FindEnabledPushCalendars(ctx context.Context, userID uuid.UUID) ([]*calendarDomain.ConnectedCalendar, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*calendarDomain.ConnectedCalendar), args.Error(1)
}

type mockOAuthService struct {
	mock.Mock
}

func (m *mockOAuthService) AuthURL(state string) string {
	args := m.Called(state)
	return args.String(0)
}

func (m *mockOAuthService) ExchangeAndStore(ctx context.Context, userID uuid.UUID, code string) (any, error) {
	args := m.Called(ctx, userID, code)
	return args.Get(0), args.Error(1)
}

type mockCalDAVCredStore struct {
	mock.Mock
}

func (m *mockCalDAVCredStore) StoreCredentials(ctx context.Context, userID uuid.UUID, provider calendarDomain.ProviderType, username, password string) error {
	args := m.Called(ctx, userID, provider, username, password)
	return args.Error(0)
}

type mockCalDAVValidator struct {
	mock.Mock
}

func (m *mockCalDAVValidator) ValidateCredentials(ctx context.Context, serverURL, username, password string) error {
	args := m.Called(ctx, serverURL, username, password)
	return args.Error(0)
}

// Tests

func TestIsValidProvider(t *testing.T) {
	tests := []struct {
		provider calendarDomain.ProviderType
		valid    bool
	}{
		{calendarDomain.ProviderGoogle, true},
		{calendarDomain.ProviderMicrosoft, true},
		{calendarDomain.ProviderApple, true},
		{calendarDomain.ProviderCalDAV, true},
		{calendarDomain.ProviderType("invalid"), false},
		{calendarDomain.ProviderType(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			result := isValidProvider(tt.provider)
			assert.Equal(t, tt.valid, result)
		})
	}
}

func TestDeriveCalendarName(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://caldav.fastmail.com", "Fastmail Calendar"},
		{"https://calendar.FASTMAIL.com/dav", "Fastmail Calendar"},
		{"https://cloud.nextcloud.example.com", "Nextcloud Calendar"},
		{"https://caldav.icloud.com", "Apple Calendar"},
		{"https://some-unknown-server.com", "CalDAV Calendar"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := deriveCalendarName(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetDefaultCalendarName(t *testing.T) {
	tests := []struct {
		name     string
		provider calendarDomain.ProviderType
		config   map[string]string
		expected string
	}{
		{
			name:     "Apple provider",
			provider: calendarDomain.ProviderApple,
			config:   nil,
			expected: "Apple Calendar",
		},
		{
			name:     "CalDAV with Fastmail URL",
			provider: calendarDomain.ProviderCalDAV,
			config:   map[string]string{calendarDomain.ConfigCalDAVURL: "https://caldav.fastmail.com"},
			expected: "Fastmail Calendar",
		},
		{
			name:     "CalDAV without URL",
			provider: calendarDomain.ProviderCalDAV,
			config:   nil,
			expected: "CalDAV Calendar",
		},
		{
			name:     "Google provider",
			provider: calendarDomain.ProviderGoogle,
			config:   nil,
			expected: "Google Calendar",
		},
		{
			name:     "Microsoft provider",
			provider: calendarDomain.ProviderMicrosoft,
			config:   nil,
			expected: "Microsoft Outlook",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getDefaultCalendarName(tt.provider, tt.config)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindCalendarByID(t *testing.T) {
	calendars := []calendarApp.Calendar{
		{ID: "cal-1", Name: "Work", Primary: false},
		{ID: "cal-2", Name: "Personal", Primary: true},
		{ID: "cal-3", Name: "Shared", Primary: false},
	}

	t.Run("find existing calendar", func(t *testing.T) {
		result := findCalendarByID(calendars, "cal-2")
		require.NotNil(t, result)
		assert.Equal(t, "Personal", result.Name)
		assert.True(t, result.Primary)
	})

	t.Run("calendar not found", func(t *testing.T) {
		result := findCalendarByID(calendars, "cal-unknown")
		assert.Nil(t, result)
	})

	t.Run("empty list", func(t *testing.T) {
		result := findCalendarByID(nil, "cal-1")
		assert.Nil(t, result)
	})
}

func TestSaveCalendar_NewCalendar(t *testing.T) {
	mockRepo := new(mockCalendarRepo)
	userID := uuid.New()
	ctx := context.Background()

	// Create service with mock repo
	service := calendarApp.NewConnectCalendarService(mockRepo, nil, nil, nil)

	// Save original and restore after test
	originalService := connectCalendarService
	connectCalendarService = service
	defer func() { connectCalendarService = originalService }()

	// Setup mocks
	mockRepo.On("FindByUserProviderAndCalendar", ctx, userID, calendarDomain.ProviderGoogle, "cal-123").
		Return(nil, errors.New("not found"))
	mockRepo.On("Save", ctx, mock.AnythingOfType("*domain.ConnectedCalendar")).
		Return(nil)

	opts := ConnectOptions{
		EnablePush: true,
		EnablePull: false,
		SetPrimary: false,
	}

	err := saveCalendar(ctx, userID, calendarDomain.ProviderGoogle, "cal-123", "Test Calendar", nil, opts)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSaveCalendar_UpdateExisting(t *testing.T) {
	mockRepo := new(mockCalendarRepo)
	userID := uuid.New()
	ctx := context.Background()

	// Create service with mock repo
	service := calendarApp.NewConnectCalendarService(mockRepo, nil, nil, nil)

	// Save original and restore after test
	originalService := connectCalendarService
	connectCalendarService = service
	defer func() { connectCalendarService = originalService }()

	// Create existing calendar
	existingCal, err := calendarDomain.NewConnectedCalendar(userID, calendarDomain.ProviderGoogle, "cal-123", "Old Name")
	require.NoError(t, err)

	// Setup mocks
	mockRepo.On("FindByUserProviderAndCalendar", ctx, userID, calendarDomain.ProviderGoogle, "cal-123").
		Return(existingCal, nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*domain.ConnectedCalendar")).
		Return(nil)

	opts := ConnectOptions{
		EnablePush: true,
		EnablePull: true,
		SetPrimary: false,
	}

	err = saveCalendar(ctx, userID, calendarDomain.ProviderGoogle, "cal-123", "New Name", nil, opts)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSaveCalendar_SetPrimary(t *testing.T) {
	mockRepo := new(mockCalendarRepo)
	userID := uuid.New()
	ctx := context.Background()

	// Create service with mock repo
	service := calendarApp.NewConnectCalendarService(mockRepo, nil, nil, nil)

	// Save original and restore after test
	originalService := connectCalendarService
	connectCalendarService = service
	defer func() { connectCalendarService = originalService }()

	// Setup mocks
	mockRepo.On("FindByUserProviderAndCalendar", ctx, userID, calendarDomain.ProviderGoogle, "cal-123").
		Return(nil, errors.New("not found"))
	// No existing primary
	mockRepo.On("FindPrimaryForUser", ctx, userID).Return(nil, errors.New("not found"))
	mockRepo.On("Save", ctx, mock.MatchedBy(func(cal *calendarDomain.ConnectedCalendar) bool {
		return cal.IsPrimary()
	})).Return(nil)

	opts := ConnectOptions{
		EnablePush: true,
		EnablePull: false,
		SetPrimary: true,
	}

	err := saveCalendar(ctx, userID, calendarDomain.ProviderGoogle, "cal-123", "Primary Calendar", nil, opts)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSaveCalendar_ServiceNotConfigured(t *testing.T) {
	// Save original and restore after test
	originalService := connectCalendarService
	connectCalendarService = nil
	defer func() { connectCalendarService = originalService }()

	opts := ConnectOptions{}
	err := saveCalendar(context.Background(), uuid.New(), calendarDomain.ProviderGoogle, "cal-123", "Test", nil, opts)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connect calendar service not configured")
}

func TestValidateCalDAVCredentials_WithValidator(t *testing.T) {
	mockValidator := new(mockCalDAVValidator)
	ctx := context.Background()

	// Save original and restore after test
	originalValidator := caldavCredValidator
	caldavCredValidator = mockValidator
	defer func() { caldavCredValidator = originalValidator }()

	t.Run("valid credentials", func(t *testing.T) {
		mockValidator.On("ValidateCredentials", ctx, "https://caldav.example.com", "user", "pass").
			Return(nil).Once()

		err := validateCalDAVCredentials(ctx, "https://caldav.example.com", "user", "pass")
		assert.NoError(t, err)
		mockValidator.AssertExpectations(t)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		mockValidator.On("ValidateCredentials", ctx, "https://caldav.example.com", "user", "wrong").
			Return(errors.New("authentication failed")).Once()

		err := validateCalDAVCredentials(ctx, "https://caldav.example.com", "user", "wrong")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "authentication failed")
		mockValidator.AssertExpectations(t)
	})
}

func TestValidateCalDAVCredentials_NoValidator(t *testing.T) {
	// Save original and restore after test
	originalValidator := caldavCredValidator
	caldavCredValidator = nil
	defer func() { caldavCredValidator = originalValidator }()

	// Should return nil when no validator is configured (best effort)
	err := validateCalDAVCredentials(context.Background(), "https://caldav.example.com", "user", "pass")
	assert.NoError(t, err)
}

func TestConnectOptions_Defaults(t *testing.T) {
	opts := ConnectOptions{}

	// Verify zero values
	assert.Equal(t, "", opts.CalendarID)
	assert.Equal(t, "", opts.CalendarName)
	assert.False(t, opts.SetPrimary)
	assert.False(t, opts.EnablePush)
	assert.False(t, opts.EnablePull)
	assert.False(t, opts.ListOnly)
	assert.False(t, opts.ConnectAll)
	assert.Equal(t, "", opts.CalDAVURL)
}

func TestInteractiveCalendarSelection_SelectPrimary(t *testing.T) {
	mockRepo := new(mockCalendarRepo)
	userID := uuid.New()
	ctx := context.Background()

	// Create service with mock repo
	service := calendarApp.NewConnectCalendarService(mockRepo, nil, nil, nil)

	// Save original and restore after test
	originalService := connectCalendarService
	connectCalendarService = service
	defer func() { connectCalendarService = originalService }()

	calendars := []calendarApp.Calendar{
		{ID: "cal-1", Name: "Work", Primary: false},
		{ID: "cal-2", Name: "Personal", Primary: true},
	}

	// User enters empty string - should select primary
	reader := bufio.NewReader(strings.NewReader("\n"))

	mockRepo.On("FindByUserProviderAndCalendar", ctx, userID, calendarDomain.ProviderGoogle, "cal-2").
		Return(nil, errors.New("not found"))
	mockRepo.On("Save", ctx, mock.MatchedBy(func(cal *calendarDomain.ConnectedCalendar) bool {
		return cal.CalendarID() == "cal-2" && cal.Name() == "Personal"
	})).Return(nil)

	opts := ConnectOptions{EnablePush: true}

	err := interactiveCalendarSelection(ctx, userID, calendarDomain.ProviderGoogle, calendars, nil, opts, reader)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestInteractiveCalendarSelection_SelectByNumber(t *testing.T) {
	mockRepo := new(mockCalendarRepo)
	userID := uuid.New()
	ctx := context.Background()

	// Create service with mock repo
	service := calendarApp.NewConnectCalendarService(mockRepo, nil, nil, nil)

	// Save original and restore after test
	originalService := connectCalendarService
	connectCalendarService = service
	defer func() { connectCalendarService = originalService }()

	calendars := []calendarApp.Calendar{
		{ID: "cal-1", Name: "Work", Primary: false},
		{ID: "cal-2", Name: "Personal", Primary: true},
		{ID: "cal-3", Name: "Shared", Primary: false},
	}

	// User enters "1" - should select first calendar
	reader := bufio.NewReader(strings.NewReader("1\n"))

	mockRepo.On("FindByUserProviderAndCalendar", ctx, userID, calendarDomain.ProviderGoogle, "cal-1").
		Return(nil, errors.New("not found"))
	mockRepo.On("Save", ctx, mock.MatchedBy(func(cal *calendarDomain.ConnectedCalendar) bool {
		return cal.CalendarID() == "cal-1" && cal.Name() == "Work"
	})).Return(nil)

	opts := ConnectOptions{EnablePush: true}

	err := interactiveCalendarSelection(ctx, userID, calendarDomain.ProviderGoogle, calendars, nil, opts, reader)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestInteractiveCalendarSelection_SelectAll(t *testing.T) {
	mockRepo := new(mockCalendarRepo)
	userID := uuid.New()
	ctx := context.Background()

	// Create service with mock repo
	service := calendarApp.NewConnectCalendarService(mockRepo, nil, nil, nil)

	// Save original and restore after test
	originalService := connectCalendarService
	connectCalendarService = service
	defer func() { connectCalendarService = originalService }()

	calendars := []calendarApp.Calendar{
		{ID: "cal-1", Name: "Work", Primary: false},
		{ID: "cal-2", Name: "Personal", Primary: true},
	}

	// User enters "all" - should select all calendars
	reader := bufio.NewReader(strings.NewReader("all\n"))

	mockRepo.On("FindByUserProviderAndCalendar", ctx, userID, calendarDomain.ProviderGoogle, mock.Anything).
		Return(nil, errors.New("not found"))
	mockRepo.On("Save", ctx, mock.AnythingOfType("*domain.ConnectedCalendar")).
		Return(nil)

	opts := ConnectOptions{EnablePush: true}

	err := interactiveCalendarSelection(ctx, userID, calendarDomain.ProviderGoogle, calendars, nil, opts, reader)

	assert.NoError(t, err)
	// Should have saved 2 calendars
	mockRepo.AssertNumberOfCalls(t, "Save", 2)
}

func TestInteractiveCalendarSelection_InvalidNumber(t *testing.T) {
	mockRepo := new(mockCalendarRepo)
	userID := uuid.New()
	ctx := context.Background()

	// Create service with mock repo
	service := calendarApp.NewConnectCalendarService(mockRepo, nil, nil, nil)

	// Save original and restore after test
	originalService := connectCalendarService
	connectCalendarService = service
	defer func() { connectCalendarService = originalService }()

	calendars := []calendarApp.Calendar{
		{ID: "cal-1", Name: "Work", Primary: false},
		{ID: "cal-2", Name: "Personal", Primary: true},
	}

	// User enters "5" - invalid number
	reader := bufio.NewReader(strings.NewReader("5\n"))

	opts := ConnectOptions{EnablePush: true}

	err := interactiveCalendarSelection(ctx, userID, calendarDomain.ProviderGoogle, calendars, nil, opts, reader)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid calendar number")
}

func TestInteractiveCalendarSelection_MultipleNumbers(t *testing.T) {
	mockRepo := new(mockCalendarRepo)
	userID := uuid.New()
	ctx := context.Background()

	// Create service with mock repo
	service := calendarApp.NewConnectCalendarService(mockRepo, nil, nil, nil)

	// Save original and restore after test
	originalService := connectCalendarService
	connectCalendarService = service
	defer func() { connectCalendarService = originalService }()

	calendars := []calendarApp.Calendar{
		{ID: "cal-1", Name: "Work", Primary: false},
		{ID: "cal-2", Name: "Personal", Primary: true},
		{ID: "cal-3", Name: "Shared", Primary: false},
	}

	// User enters "1,3" - should select first and third calendars
	reader := bufio.NewReader(strings.NewReader("1,3\n"))

	mockRepo.On("FindByUserProviderAndCalendar", ctx, userID, calendarDomain.ProviderGoogle, mock.Anything).
		Return(nil, errors.New("not found"))
	mockRepo.On("Save", ctx, mock.AnythingOfType("*domain.ConnectedCalendar")).
		Return(nil)

	opts := ConnectOptions{EnablePush: true}

	err := interactiveCalendarSelection(ctx, userID, calendarDomain.ProviderGoogle, calendars, nil, opts, reader)

	assert.NoError(t, err)
	// Should have saved 2 calendars
	mockRepo.AssertNumberOfCalls(t, "Save", 2)
}

func TestConnectHandler_Creation(t *testing.T) {
	mockRepo := new(mockCalendarRepo)
	mockStore := new(mockCalDAVCredStore)

	handler := NewConnectHandler(mockRepo, nil, nil, mockStore)

	assert.NotNil(t, handler)
	assert.Equal(t, mockRepo, handler.calendarRepo)
	assert.Equal(t, mockStore, handler.caldavStore)
	assert.NotNil(t, handler.reader)
}

func TestHandleCalendarSelection_ListOnly(t *testing.T) {
	mockRepo := new(mockCalendarRepo)
	userID := uuid.New()
	ctx := context.Background()

	// Create service with mock repo
	service := calendarApp.NewConnectCalendarService(mockRepo, nil, nil, nil)

	// Save original and restore after test
	originalService := connectCalendarService
	connectCalendarService = service
	originalRegistry := providerRegistry
	providerRegistry = nil
	defer func() {
		connectCalendarService = originalService
		providerRegistry = originalRegistry
	}()

	// When provider registry is nil, handleCalendarSelection falls back to default calendar
	opts := ConnectOptions{
		ListOnly:   true,
		EnablePush: true,
	}

	reader := bufio.NewReader(strings.NewReader(""))

	// Since provider registry is nil, it will fall back to saving a default calendar
	mockRepo.On("FindByUserProviderAndCalendar", ctx, userID, calendarDomain.ProviderGoogle, "default").
		Return(nil, errors.New("not found"))
	mockRepo.On("Save", ctx, mock.AnythingOfType("*domain.ConnectedCalendar")).
		Return(nil)

	err := handleCalendarSelection(ctx, userID, calendarDomain.ProviderGoogle, nil, opts, reader)

	// Without provider registry, it will fall back to default calendar
	assert.NoError(t, err) // Falls back to default
}

func TestHandleCalendarSelection_SpecificCalendar(t *testing.T) {
	mockRepo := new(mockCalendarRepo)
	userID := uuid.New()
	ctx := context.Background()

	// Create service with mock repo
	service := calendarApp.NewConnectCalendarService(mockRepo, nil, nil, nil)

	// Save original and restore after test
	originalService := connectCalendarService
	connectCalendarService = service
	originalRegistry := providerRegistry
	providerRegistry = nil
	defer func() {
		connectCalendarService = originalService
		providerRegistry = originalRegistry
	}()

	opts := ConnectOptions{
		CalendarID:   "cal-specific",
		CalendarName: "My Calendar",
		EnablePush:   true,
	}

	// Without registry, falls back to default calendar
	mockRepo.On("FindByUserProviderAndCalendar", ctx, userID, calendarDomain.ProviderGoogle, "default").
		Return(nil, errors.New("not found"))
	mockRepo.On("Save", ctx, mock.AnythingOfType("*domain.ConnectedCalendar")).
		Return(nil)

	reader := bufio.NewReader(strings.NewReader(""))

	err := handleCalendarSelection(ctx, userID, calendarDomain.ProviderGoogle, nil, opts, reader)

	assert.NoError(t, err)
}

func TestConnectMultipleCalendars_PartialFailure(t *testing.T) {
	mockRepo := new(mockCalendarRepo)
	userID := uuid.New()
	ctx := context.Background()

	// Create service with mock repo
	service := calendarApp.NewConnectCalendarService(mockRepo, nil, nil, nil)

	// Save original and restore after test
	originalService := connectCalendarService
	connectCalendarService = service
	defer func() { connectCalendarService = originalService }()

	calendars := []calendarApp.Calendar{
		{ID: "cal-1", Name: "Success", Primary: false},
		{ID: "cal-2", Name: "Failure", Primary: false},
		{ID: "cal-3", Name: "Success2", Primary: false},
	}

	// First and third succeed, second fails
	mockRepo.On("FindByUserProviderAndCalendar", ctx, userID, calendarDomain.ProviderGoogle, "cal-1").
		Return(nil, errors.New("not found"))
	mockRepo.On("FindByUserProviderAndCalendar", ctx, userID, calendarDomain.ProviderGoogle, "cal-2").
		Return(nil, errors.New("not found"))
	mockRepo.On("FindByUserProviderAndCalendar", ctx, userID, calendarDomain.ProviderGoogle, "cal-3").
		Return(nil, errors.New("not found"))

	mockRepo.On("Save", ctx, mock.MatchedBy(func(cal *calendarDomain.ConnectedCalendar) bool {
		return cal.CalendarID() == "cal-1"
	})).Return(nil)
	mockRepo.On("Save", ctx, mock.MatchedBy(func(cal *calendarDomain.ConnectedCalendar) bool {
		return cal.CalendarID() == "cal-2"
	})).Return(errors.New("save failed"))
	mockRepo.On("Save", ctx, mock.MatchedBy(func(cal *calendarDomain.ConnectedCalendar) bool {
		return cal.CalendarID() == "cal-3"
	})).Return(nil)

	opts := ConnectOptions{EnablePush: true}

	// Should not return error even with partial failures
	err := connectMultipleCalendars(ctx, userID, calendarDomain.ProviderGoogle, calendars, nil, opts)

	assert.NoError(t, err) // Partial failures are logged but not returned as error
	mockRepo.AssertExpectations(t)
}

func TestConnectMultipleCalendars_FirstAsPrimary(t *testing.T) {
	mockRepo := new(mockCalendarRepo)
	userID := uuid.New()
	ctx := context.Background()

	// Create service with mock repo
	service := calendarApp.NewConnectCalendarService(mockRepo, nil, nil, nil)

	// Save original and restore after test
	originalService := connectCalendarService
	connectCalendarService = service
	defer func() { connectCalendarService = originalService }()

	calendars := []calendarApp.Calendar{
		{ID: "cal-1", Name: "First", Primary: false},
		{ID: "cal-2", Name: "Second", Primary: false},
	}

	mockRepo.On("FindByUserProviderAndCalendar", ctx, userID, calendarDomain.ProviderGoogle, "cal-1").
		Return(nil, errors.New("not found"))
	mockRepo.On("FindByUserProviderAndCalendar", ctx, userID, calendarDomain.ProviderGoogle, "cal-2").
		Return(nil, errors.New("not found"))
	// No existing primary
	mockRepo.On("FindPrimaryForUser", ctx, userID).Return(nil, errors.New("not found")).Once()
	mockRepo.On("Save", ctx, mock.MatchedBy(func(cal *calendarDomain.ConnectedCalendar) bool {
		// First calendar should be primary
		return cal.CalendarID() == "cal-1" && cal.IsPrimary()
	})).Return(nil)
	mockRepo.On("Save", ctx, mock.MatchedBy(func(cal *calendarDomain.ConnectedCalendar) bool {
		// Second calendar should NOT be primary
		return cal.CalendarID() == "cal-2" && !cal.IsPrimary()
	})).Return(nil)

	opts := ConnectOptions{
		EnablePush: true,
		SetPrimary: true, // --primary flag set
	}

	err := connectMultipleCalendars(ctx, userID, calendarDomain.ProviderGoogle, calendars, nil, opts)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	// FindPrimaryForUser should only be called once (for first calendar)
	mockRepo.AssertNumberOfCalls(t, "FindPrimaryForUser", 1)
}
