package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/calendar/application"
	"github.com/felixgeelhaar/orbita/internal/calendar/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementations
type mockSyncer struct {
	result *application.SyncResult
	err    error
}

func (m *mockSyncer) Sync(ctx context.Context, userID uuid.UUID, blocks []application.TimeBlock) (*application.SyncResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

type mockImporter struct {
	events    []application.CalendarEvent
	calendars []application.Calendar
	err       error
}

func (m *mockImporter) ListEvents(ctx context.Context, userID uuid.UUID, start, end time.Time, onlyOrbitaEvents bool) ([]application.CalendarEvent, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.events, nil
}

func (m *mockImporter) ListCalendars(ctx context.Context, userID uuid.UUID) ([]application.Calendar, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.calendars, nil
}

type mockBidirectionalSyncer struct {
	mockSyncer
	mockImporter
}

type mockCalendarRepo struct {
	calendars       []*domain.ConnectedCalendar
	primary         *domain.ConnectedCalendar
	pushCalendars   []*domain.ConnectedCalendar
	pullCalendars   []*domain.ConnectedCalendar
	savedCalendars  []*domain.ConnectedCalendar
	findByUserErr   error
	findPrimaryErr  error
	findPushErr     error
	findPullErr     error
	findByProviderErr error
	saveErr         error
}

func (m *mockCalendarRepo) Save(ctx context.Context, cal *domain.ConnectedCalendar) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.savedCalendars = append(m.savedCalendars, cal)
	return nil
}

func (m *mockCalendarRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.ConnectedCalendar, error) {
	for _, c := range m.calendars {
		if c.ID() == id {
			return c, nil
		}
	}
	return nil, nil
}

func (m *mockCalendarRepo) FindByUserAndProvider(ctx context.Context, userID uuid.UUID, provider domain.ProviderType) ([]*domain.ConnectedCalendar, error) {
	if m.findByProviderErr != nil {
		return nil, m.findByProviderErr
	}
	var result []*domain.ConnectedCalendar
	for _, c := range m.calendars {
		if c.UserID() == userID && c.Provider() == provider {
			result = append(result, c)
		}
	}
	return result, nil
}

func (m *mockCalendarRepo) FindByUserProviderAndCalendar(ctx context.Context, userID uuid.UUID, provider domain.ProviderType, calendarID string) (*domain.ConnectedCalendar, error) {
	for _, c := range m.calendars {
		if c.UserID() == userID && c.Provider() == provider && c.CalendarID() == calendarID {
			return c, nil
		}
	}
	return nil, nil
}

func (m *mockCalendarRepo) FindByUser(ctx context.Context, userID uuid.UUID) ([]*domain.ConnectedCalendar, error) {
	if m.findByUserErr != nil {
		return nil, m.findByUserErr
	}
	var result []*domain.ConnectedCalendar
	for _, c := range m.calendars {
		if c.UserID() == userID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (m *mockCalendarRepo) FindPrimaryForUser(ctx context.Context, userID uuid.UUID) (*domain.ConnectedCalendar, error) {
	if m.findPrimaryErr != nil {
		return nil, m.findPrimaryErr
	}
	return m.primary, nil
}

func (m *mockCalendarRepo) FindEnabledPushCalendars(ctx context.Context, userID uuid.UUID) ([]*domain.ConnectedCalendar, error) {
	if m.findPushErr != nil {
		return nil, m.findPushErr
	}
	return m.pushCalendars, nil
}

func (m *mockCalendarRepo) FindEnabledPullCalendars(ctx context.Context, userID uuid.UUID) ([]*domain.ConnectedCalendar, error) {
	if m.findPullErr != nil {
		return nil, m.findPullErr
	}
	return m.pullCalendars, nil
}

func (m *mockCalendarRepo) ClearPrimaryForUser(ctx context.Context, userID uuid.UUID) error {
	return nil
}

func (m *mockCalendarRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockCalendarRepo) DeleteByUserAndProvider(ctx context.Context, userID uuid.UUID, provider domain.ProviderType) error {
	return nil
}

// Tests

func TestNewProviderRegistry(t *testing.T) {
	registry := application.NewProviderRegistry()
	assert.NotNil(t, registry)
	assert.Empty(t, registry.SupportedProviders())
}

func TestProviderRegistry_RegisterSyncer(t *testing.T) {
	registry := application.NewProviderRegistry()

	factory := func(ctx context.Context, cal *domain.ConnectedCalendar) (application.Syncer, error) {
		return &mockSyncer{result: &application.SyncResult{Created: 1}}, nil
	}

	registry.RegisterSyncer(domain.ProviderGoogle, factory)

	assert.True(t, registry.HasProvider(domain.ProviderGoogle))
	assert.Contains(t, registry.SupportedProviders(), domain.ProviderGoogle)
}

func TestProviderRegistry_RegisterImporter(t *testing.T) {
	registry := application.NewProviderRegistry()

	factory := func(ctx context.Context, cal *domain.ConnectedCalendar) (application.Importer, error) {
		return &mockImporter{}, nil
	}

	registry.RegisterImporter(domain.ProviderMicrosoft, factory)

	assert.True(t, registry.HasProvider(domain.ProviderMicrosoft))
	assert.Contains(t, registry.SupportedProviders(), domain.ProviderMicrosoft)
}

func TestProviderRegistry_RegisterBidirectional(t *testing.T) {
	registry := application.NewProviderRegistry()

	factory := func(ctx context.Context, cal *domain.ConnectedCalendar) (application.BidirectionalSyncer, error) {
		return &mockBidirectionalSyncer{}, nil
	}

	registry.RegisterBidirectional(domain.ProviderCalDAV, factory)

	// Should register as both syncer and importer
	assert.True(t, registry.HasProvider(domain.ProviderCalDAV))

	// Should be able to create syncer
	cal := domain.NewConnectedCalendar(uuid.New(), domain.ProviderCalDAV, "cal", "Test")
	syncer, err := registry.CreateSyncer(context.Background(), cal)
	assert.NoError(t, err)
	assert.NotNil(t, syncer)

	// Should be able to create importer
	importer, err := registry.CreateImporter(context.Background(), cal)
	assert.NoError(t, err)
	assert.NotNil(t, importer)

	// Should be able to create bidirectional
	bidir, err := registry.CreateBidirectional(context.Background(), cal)
	assert.NoError(t, err)
	assert.NotNil(t, bidir)
}

func TestProviderRegistry_CreateSyncer_NotRegistered(t *testing.T) {
	registry := application.NewProviderRegistry()
	cal := domain.NewConnectedCalendar(uuid.New(), domain.ProviderGoogle, "primary", "Test")

	syncer, err := registry.CreateSyncer(context.Background(), cal)

	assert.Error(t, err)
	assert.Nil(t, syncer)
	assert.Contains(t, err.Error(), "no syncer registered")
}

func TestProviderRegistry_CreateImporter_NotRegistered(t *testing.T) {
	registry := application.NewProviderRegistry()
	cal := domain.NewConnectedCalendar(uuid.New(), domain.ProviderMicrosoft, "work", "Work")

	importer, err := registry.CreateImporter(context.Background(), cal)

	assert.Error(t, err)
	assert.Nil(t, importer)
	assert.Contains(t, err.Error(), "no importer registered")
}

func TestProviderRegistry_CreateBidirectional_NotRegistered(t *testing.T) {
	registry := application.NewProviderRegistry()
	cal := domain.NewConnectedCalendar(uuid.New(), domain.ProviderCalDAV, "cal", "Cal")

	bidir, err := registry.CreateBidirectional(context.Background(), cal)

	assert.Error(t, err)
	assert.Nil(t, bidir)
	assert.Contains(t, err.Error(), "no bidirectional syncer registered")
}

func TestProviderRegistry_HasProvider_False(t *testing.T) {
	registry := application.NewProviderRegistry()
	assert.False(t, registry.HasProvider(domain.ProviderApple))
}

func TestProviderRegistry_SupportedProviders(t *testing.T) {
	registry := application.NewProviderRegistry()

	syncerFactory := func(ctx context.Context, cal *domain.ConnectedCalendar) (application.Syncer, error) {
		return &mockSyncer{}, nil
	}
	importerFactory := func(ctx context.Context, cal *domain.ConnectedCalendar) (application.Importer, error) {
		return &mockImporter{}, nil
	}

	registry.RegisterSyncer(domain.ProviderGoogle, syncerFactory)
	registry.RegisterImporter(domain.ProviderMicrosoft, importerFactory)

	providers := registry.SupportedProviders()
	assert.Len(t, providers, 2)
	assert.Contains(t, providers, domain.ProviderGoogle)
	assert.Contains(t, providers, domain.ProviderMicrosoft)
}

func TestMultiSyncResult(t *testing.T) {
	result := application.NewMultiSyncResult()

	assert.NotNil(t, result.Results)
	assert.NotNil(t, result.Errors)
	assert.NotNil(t, result.Total)
	assert.False(t, result.HasErrors())
	assert.Equal(t, 0, result.SuccessCount())

	// Add a result
	result.AddResult(domain.ProviderGoogle, &application.SyncResult{
		Created: 2,
		Updated: 1,
		Deleted: 1,
	})

	assert.Equal(t, 1, result.SuccessCount())
	assert.Equal(t, 2, result.Total.Created)
	assert.Equal(t, 1, result.Total.Updated)
	assert.Equal(t, 1, result.Total.Deleted)

	// Add another result
	result.AddResult(domain.ProviderMicrosoft, &application.SyncResult{
		Created: 3,
		Failed:  1,
	})

	assert.Equal(t, 2, result.SuccessCount())
	assert.Equal(t, 5, result.Total.Created)
	assert.Equal(t, 1, result.Total.Failed)

	// Add an error
	result.AddError(domain.ProviderCalDAV, errors.New("sync failed"))

	assert.True(t, result.HasErrors())
	assert.Equal(t, 2, result.SuccessCount()) // Errors don't affect success count
}

func TestMultiSyncResult_AddResult_NilResult(t *testing.T) {
	result := application.NewMultiSyncResult()
	result.AddResult(domain.ProviderGoogle, nil)

	// Should not panic and totals should remain zero
	assert.Equal(t, 0, result.Total.Created)
}

func TestNewSyncCoordinator(t *testing.T) {
	registry := application.NewProviderRegistry()
	repo := &mockCalendarRepo{}

	coordinator := application.NewSyncCoordinator(registry, repo)

	assert.NotNil(t, coordinator)
}

func TestSyncCoordinator_SyncAll(t *testing.T) {
	registry := application.NewProviderRegistry()
	userID := uuid.New()

	cal := domain.NewConnectedCalendar(userID, domain.ProviderGoogle, "primary", "Personal")

	repo := &mockCalendarRepo{
		pushCalendars: []*domain.ConnectedCalendar{cal},
	}

	registry.RegisterSyncer(domain.ProviderGoogle, func(ctx context.Context, c *domain.ConnectedCalendar) (application.Syncer, error) {
		return &mockSyncer{result: &application.SyncResult{Created: 2}}, nil
	})

	coordinator := application.NewSyncCoordinator(registry, repo)

	blocks := []application.TimeBlock{
		{ID: uuid.New(), Title: "Task 1"},
	}

	result, err := coordinator.SyncAll(context.Background(), userID, blocks)

	require.NoError(t, err)
	assert.Equal(t, 1, result.SuccessCount())
	assert.Equal(t, 2, result.Total.Created)
	assert.Len(t, repo.savedCalendars, 1) // Calendar was saved after sync
}

func TestSyncCoordinator_SyncAll_RepoError(t *testing.T) {
	registry := application.NewProviderRegistry()
	repo := &mockCalendarRepo{
		findPushErr: errors.New("database error"),
	}

	coordinator := application.NewSyncCoordinator(registry, repo)

	result, err := coordinator.SyncAll(context.Background(), uuid.New(), nil)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestSyncCoordinator_SyncAll_SyncerError(t *testing.T) {
	registry := application.NewProviderRegistry()
	userID := uuid.New()

	cal := domain.NewConnectedCalendar(userID, domain.ProviderGoogle, "primary", "Personal")

	repo := &mockCalendarRepo{
		pushCalendars: []*domain.ConnectedCalendar{cal},
	}

	registry.RegisterSyncer(domain.ProviderGoogle, func(ctx context.Context, c *domain.ConnectedCalendar) (application.Syncer, error) {
		return &mockSyncer{err: errors.New("sync failed")}, nil
	})

	coordinator := application.NewSyncCoordinator(registry, repo)

	result, err := coordinator.SyncAll(context.Background(), userID, nil)

	require.NoError(t, err) // SyncAll doesn't fail, it collects errors
	assert.True(t, result.HasErrors())
	assert.Equal(t, 0, result.SuccessCount())
}

func TestSyncCoordinator_SyncToProvider(t *testing.T) {
	registry := application.NewProviderRegistry()
	userID := uuid.New()

	cal := domain.NewConnectedCalendar(userID, domain.ProviderGoogle, "primary", "Personal")

	repo := &mockCalendarRepo{
		calendars: []*domain.ConnectedCalendar{cal},
	}

	registry.RegisterSyncer(domain.ProviderGoogle, func(ctx context.Context, c *domain.ConnectedCalendar) (application.Syncer, error) {
		return &mockSyncer{result: &application.SyncResult{Created: 3}}, nil
	})

	coordinator := application.NewSyncCoordinator(registry, repo)

	result, err := coordinator.SyncToProvider(context.Background(), userID, domain.ProviderGoogle, nil)

	require.NoError(t, err)
	assert.Equal(t, 3, result.Created)
}

func TestSyncCoordinator_SyncToProvider_NoCalendar(t *testing.T) {
	registry := application.NewProviderRegistry()
	repo := &mockCalendarRepo{}

	coordinator := application.NewSyncCoordinator(registry, repo)

	result, err := coordinator.SyncToProvider(context.Background(), uuid.New(), domain.ProviderGoogle, nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no Google Calendar calendar connected")
}

func TestSyncCoordinator_SyncToProvider_CalendarDisabled(t *testing.T) {
	registry := application.NewProviderRegistry()
	userID := uuid.New()

	cal := domain.NewConnectedCalendar(userID, domain.ProviderGoogle, "primary", "Personal")
	cal.SetEnabled(false)

	repo := &mockCalendarRepo{
		calendars: []*domain.ConnectedCalendar{cal},
	}

	coordinator := application.NewSyncCoordinator(registry, repo)

	result, err := coordinator.SyncToProvider(context.Background(), userID, domain.ProviderGoogle, nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no enabled")
}

func TestSyncCoordinator_GetPrimaryImporter(t *testing.T) {
	registry := application.NewProviderRegistry()
	userID := uuid.New()

	primary := domain.NewConnectedCalendar(userID, domain.ProviderGoogle, "primary", "Primary")
	primary.SetPrimary(true)

	repo := &mockCalendarRepo{
		primary: primary,
	}

	registry.RegisterImporter(domain.ProviderGoogle, func(ctx context.Context, c *domain.ConnectedCalendar) (application.Importer, error) {
		return &mockImporter{}, nil
	})

	coordinator := application.NewSyncCoordinator(registry, repo)

	importer, cal, err := coordinator.GetPrimaryImporter(context.Background(), userID)

	require.NoError(t, err)
	assert.NotNil(t, importer)
	assert.Equal(t, primary.ID(), cal.ID())
}

func TestSyncCoordinator_GetPrimaryImporter_FallbackToPull(t *testing.T) {
	registry := application.NewProviderRegistry()
	userID := uuid.New()

	pullCal := domain.NewConnectedCalendar(userID, domain.ProviderMicrosoft, "work", "Work")
	pullCal.SetSyncPull(true)

	repo := &mockCalendarRepo{
		primary:       nil, // No primary
		pullCalendars: []*domain.ConnectedCalendar{pullCal},
	}

	registry.RegisterImporter(domain.ProviderMicrosoft, func(ctx context.Context, c *domain.ConnectedCalendar) (application.Importer, error) {
		return &mockImporter{}, nil
	})

	coordinator := application.NewSyncCoordinator(registry, repo)

	importer, cal, err := coordinator.GetPrimaryImporter(context.Background(), userID)

	require.NoError(t, err)
	assert.NotNil(t, importer)
	assert.Equal(t, pullCal.ID(), cal.ID())
}

func TestSyncCoordinator_GetPrimaryImporter_NoCalendar(t *testing.T) {
	registry := application.NewProviderRegistry()
	repo := &mockCalendarRepo{}

	coordinator := application.NewSyncCoordinator(registry, repo)

	importer, cal, err := coordinator.GetPrimaryImporter(context.Background(), uuid.New())

	assert.Error(t, err)
	assert.Nil(t, importer)
	assert.Nil(t, cal)
	assert.Contains(t, err.Error(), "no calendar configured for import")
}

func TestSyncCoordinator_GetImporterForProvider(t *testing.T) {
	registry := application.NewProviderRegistry()
	userID := uuid.New()

	cal := domain.NewConnectedCalendar(userID, domain.ProviderCalDAV, "cal", "CalDAV")

	repo := &mockCalendarRepo{
		calendars: []*domain.ConnectedCalendar{cal},
	}

	registry.RegisterImporter(domain.ProviderCalDAV, func(ctx context.Context, c *domain.ConnectedCalendar) (application.Importer, error) {
		return &mockImporter{}, nil
	})

	coordinator := application.NewSyncCoordinator(registry, repo)

	importer, resultCal, err := coordinator.GetImporterForProvider(context.Background(), userID, domain.ProviderCalDAV)

	require.NoError(t, err)
	assert.NotNil(t, importer)
	assert.Equal(t, cal.ID(), resultCal.ID())
}

func TestSyncCoordinator_GetImporterForProvider_NoCalendar(t *testing.T) {
	registry := application.NewProviderRegistry()
	repo := &mockCalendarRepo{}

	coordinator := application.NewSyncCoordinator(registry, repo)

	importer, cal, err := coordinator.GetImporterForProvider(context.Background(), uuid.New(), domain.ProviderApple)

	assert.Error(t, err)
	assert.Nil(t, importer)
	assert.Nil(t, cal)
}

func TestSyncCoordinator_GetImporterForProvider_CalendarDisabled(t *testing.T) {
	registry := application.NewProviderRegistry()
	userID := uuid.New()

	cal := domain.NewConnectedCalendar(userID, domain.ProviderApple, "cal", "Apple")
	cal.SetEnabled(false)

	repo := &mockCalendarRepo{
		calendars: []*domain.ConnectedCalendar{cal},
	}

	coordinator := application.NewSyncCoordinator(registry, repo)

	importer, resultCal, err := coordinator.GetImporterForProvider(context.Background(), userID, domain.ProviderApple)

	assert.Error(t, err)
	assert.Nil(t, importer)
	assert.Nil(t, resultCal)
	assert.Contains(t, err.Error(), "no enabled")
}

func TestSyncCoordinator_ListConnectedCalendars(t *testing.T) {
	registry := application.NewProviderRegistry()
	userID := uuid.New()

	cal1 := domain.NewConnectedCalendar(userID, domain.ProviderGoogle, "primary", "Personal")
	cal2 := domain.NewConnectedCalendar(userID, domain.ProviderMicrosoft, "work", "Work")

	repo := &mockCalendarRepo{
		calendars: []*domain.ConnectedCalendar{cal1, cal2},
	}

	coordinator := application.NewSyncCoordinator(registry, repo)

	calendars, err := coordinator.ListConnectedCalendars(context.Background(), userID)

	require.NoError(t, err)
	assert.Len(t, calendars, 2)
}

func TestSyncCoordinator_SyncAll_CreateSyncerError(t *testing.T) {
	registry := application.NewProviderRegistry()
	userID := uuid.New()

	cal := domain.NewConnectedCalendar(userID, domain.ProviderGoogle, "primary", "Personal")

	repo := &mockCalendarRepo{
		pushCalendars: []*domain.ConnectedCalendar{cal},
	}

	// Factory returns error when creating syncer
	registry.RegisterSyncer(domain.ProviderGoogle, func(ctx context.Context, c *domain.ConnectedCalendar) (application.Syncer, error) {
		return nil, errors.New("failed to create syncer")
	})

	coordinator := application.NewSyncCoordinator(registry, repo)

	result, err := coordinator.SyncAll(context.Background(), userID, nil)

	require.NoError(t, err) // SyncAll doesn't fail, it collects errors
	assert.True(t, result.HasErrors())
	assert.Equal(t, 0, result.SuccessCount())
}

func TestSyncCoordinator_SyncAll_SaveError(t *testing.T) {
	registry := application.NewProviderRegistry()
	userID := uuid.New()

	cal := domain.NewConnectedCalendar(userID, domain.ProviderGoogle, "primary", "Personal")

	repo := &mockCalendarRepo{
		pushCalendars: []*domain.ConnectedCalendar{cal},
		saveErr:       errors.New("save failed"),
	}

	registry.RegisterSyncer(domain.ProviderGoogle, func(ctx context.Context, c *domain.ConnectedCalendar) (application.Syncer, error) {
		return &mockSyncer{result: &application.SyncResult{Created: 2}}, nil
	})

	coordinator := application.NewSyncCoordinator(registry, repo)

	result, err := coordinator.SyncAll(context.Background(), userID, nil)

	require.NoError(t, err)
	// Sync succeeded but save failed - result shows success with error note
	assert.True(t, result.HasErrors()) // Save error is recorded
	assert.Equal(t, 1, result.SuccessCount())
}

func TestSyncCoordinator_SyncToProvider_RepoError(t *testing.T) {
	registry := application.NewProviderRegistry()
	repo := &mockCalendarRepo{
		findByProviderErr: errors.New("database error"),
	}

	coordinator := application.NewSyncCoordinator(registry, repo)

	result, err := coordinator.SyncToProvider(context.Background(), uuid.New(), domain.ProviderGoogle, nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "database error")
}

func TestSyncCoordinator_SyncToProvider_CreateSyncerError(t *testing.T) {
	registry := application.NewProviderRegistry()
	userID := uuid.New()

	cal := domain.NewConnectedCalendar(userID, domain.ProviderGoogle, "primary", "Personal")

	repo := &mockCalendarRepo{
		calendars: []*domain.ConnectedCalendar{cal},
	}

	registry.RegisterSyncer(domain.ProviderGoogle, func(ctx context.Context, c *domain.ConnectedCalendar) (application.Syncer, error) {
		return nil, errors.New("failed to create syncer")
	})

	coordinator := application.NewSyncCoordinator(registry, repo)

	result, err := coordinator.SyncToProvider(context.Background(), userID, domain.ProviderGoogle, nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to create syncer")
}

func TestSyncCoordinator_SyncToProvider_SyncError(t *testing.T) {
	registry := application.NewProviderRegistry()
	userID := uuid.New()

	cal := domain.NewConnectedCalendar(userID, domain.ProviderGoogle, "primary", "Personal")

	repo := &mockCalendarRepo{
		calendars: []*domain.ConnectedCalendar{cal},
	}

	registry.RegisterSyncer(domain.ProviderGoogle, func(ctx context.Context, c *domain.ConnectedCalendar) (application.Syncer, error) {
		return &mockSyncer{err: errors.New("sync failed")}, nil
	})

	coordinator := application.NewSyncCoordinator(registry, repo)

	result, err := coordinator.SyncToProvider(context.Background(), userID, domain.ProviderGoogle, nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "sync failed")
}

func TestSyncCoordinator_GetPrimaryImporter_RepoError(t *testing.T) {
	registry := application.NewProviderRegistry()
	repo := &mockCalendarRepo{
		findPrimaryErr: errors.New("database error"),
	}

	coordinator := application.NewSyncCoordinator(registry, repo)

	importer, cal, err := coordinator.GetPrimaryImporter(context.Background(), uuid.New())

	assert.Error(t, err)
	assert.Nil(t, importer)
	assert.Nil(t, cal)
}

func TestSyncCoordinator_GetPrimaryImporter_FallbackRepoError(t *testing.T) {
	registry := application.NewProviderRegistry()
	// primary is nil, fallback to pull calendars will fail
	repo := &mockCalendarRepo{
		primary:     nil,
		findPullErr: errors.New("database error on pull calendars"),
	}

	coordinator := application.NewSyncCoordinator(registry, repo)

	importer, cal, err := coordinator.GetPrimaryImporter(context.Background(), uuid.New())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pull calendars")
	assert.Nil(t, importer)
	assert.Nil(t, cal)
}

func TestSyncCoordinator_GetImporterForProvider_RepoError(t *testing.T) {
	registry := application.NewProviderRegistry()
	repo := &mockCalendarRepo{
		findByProviderErr: errors.New("database error"),
	}

	coordinator := application.NewSyncCoordinator(registry, repo)

	importer, cal, err := coordinator.GetImporterForProvider(context.Background(), uuid.New(), domain.ProviderGoogle)

	assert.Error(t, err)
	assert.Nil(t, importer)
	assert.Nil(t, cal)
}

func TestSyncCoordinator_GetImporterForProvider_CreateImporterError(t *testing.T) {
	registry := application.NewProviderRegistry()
	userID := uuid.New()

	cal := domain.NewConnectedCalendar(userID, domain.ProviderGoogle, "primary", "Personal")
	cal.SetSyncPull(true)

	repo := &mockCalendarRepo{
		calendars: []*domain.ConnectedCalendar{cal},
	}

	registry.RegisterImporter(domain.ProviderGoogle, func(ctx context.Context, c *domain.ConnectedCalendar) (application.Importer, error) {
		return nil, errors.New("failed to create importer")
	})

	coordinator := application.NewSyncCoordinator(registry, repo)

	importer, resultCal, err := coordinator.GetImporterForProvider(context.Background(), userID, domain.ProviderGoogle)

	assert.Error(t, err)
	assert.Nil(t, importer)
	assert.Nil(t, resultCal)
}
