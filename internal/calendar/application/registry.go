package application

import (
	"context"
	"fmt"
	"sync"

	"github.com/felixgeelhaar/orbita/internal/calendar/domain"
	"github.com/google/uuid"
)

// SyncerFactory creates a Syncer for a specific user and calendar configuration.
type SyncerFactory func(ctx context.Context, calendar *domain.ConnectedCalendar) (Syncer, error)

// ImporterFactory creates an Importer for a specific user and calendar configuration.
type ImporterFactory func(ctx context.Context, calendar *domain.ConnectedCalendar) (Importer, error)

// BidirectionalSyncerFactory creates a BidirectionalSyncer for a specific user and calendar configuration.
type BidirectionalSyncerFactory func(ctx context.Context, calendar *domain.ConnectedCalendar) (BidirectionalSyncer, error)

// ProviderRegistry manages calendar provider implementations.
// It uses the factory pattern to create syncers/importers based on provider type and user configuration.
type ProviderRegistry struct {
	mu              sync.RWMutex
	syncerFactories map[domain.ProviderType]SyncerFactory
	importerFactories map[domain.ProviderType]ImporterFactory
	bidirectionalFactories map[domain.ProviderType]BidirectionalSyncerFactory
}

// NewProviderRegistry creates a new provider registry.
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		syncerFactories:        make(map[domain.ProviderType]SyncerFactory),
		importerFactories:      make(map[domain.ProviderType]ImporterFactory),
		bidirectionalFactories: make(map[domain.ProviderType]BidirectionalSyncerFactory),
	}
}

// RegisterSyncer registers a syncer factory for a provider type.
func (r *ProviderRegistry) RegisterSyncer(provider domain.ProviderType, factory SyncerFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.syncerFactories[provider] = factory
}

// RegisterImporter registers an importer factory for a provider type.
func (r *ProviderRegistry) RegisterImporter(provider domain.ProviderType, factory ImporterFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.importerFactories[provider] = factory
}

// RegisterBidirectional registers a bidirectional syncer factory for a provider type.
// This also registers the syncer and importer interfaces automatically.
func (r *ProviderRegistry) RegisterBidirectional(provider domain.ProviderType, factory BidirectionalSyncerFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.bidirectionalFactories[provider] = factory

	// Also register as syncer and importer
	r.syncerFactories[provider] = func(ctx context.Context, cal *domain.ConnectedCalendar) (Syncer, error) {
		return factory(ctx, cal)
	}
	r.importerFactories[provider] = func(ctx context.Context, cal *domain.ConnectedCalendar) (Importer, error) {
		return factory(ctx, cal)
	}
}

// CreateSyncer creates a syncer for the given calendar.
func (r *ProviderRegistry) CreateSyncer(ctx context.Context, calendar *domain.ConnectedCalendar) (Syncer, error) {
	r.mu.RLock()
	factory, ok := r.syncerFactories[calendar.Provider()]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("no syncer registered for provider: %s", calendar.Provider())
	}
	return factory(ctx, calendar)
}

// CreateImporter creates an importer for the given calendar.
func (r *ProviderRegistry) CreateImporter(ctx context.Context, calendar *domain.ConnectedCalendar) (Importer, error) {
	r.mu.RLock()
	factory, ok := r.importerFactories[calendar.Provider()]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("no importer registered for provider: %s", calendar.Provider())
	}
	return factory(ctx, calendar)
}

// CreateBidirectional creates a bidirectional syncer for the given calendar.
func (r *ProviderRegistry) CreateBidirectional(ctx context.Context, calendar *domain.ConnectedCalendar) (BidirectionalSyncer, error) {
	r.mu.RLock()
	factory, ok := r.bidirectionalFactories[calendar.Provider()]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("no bidirectional syncer registered for provider: %s", calendar.Provider())
	}
	return factory(ctx, calendar)
}

// HasProvider returns true if a provider is registered.
func (r *ProviderRegistry) HasProvider(provider domain.ProviderType) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, hasSyncer := r.syncerFactories[provider]
	_, hasImporter := r.importerFactories[provider]
	return hasSyncer || hasImporter
}

// SupportedProviders returns all registered provider types.
func (r *ProviderRegistry) SupportedProviders() []domain.ProviderType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make(map[domain.ProviderType]bool)
	for p := range r.syncerFactories {
		providers[p] = true
	}
	for p := range r.importerFactories {
		providers[p] = true
	}

	result := make([]domain.ProviderType, 0, len(providers))
	for p := range providers {
		result = append(result, p)
	}
	return result
}

// MultiSyncResult aggregates sync results from multiple providers.
type MultiSyncResult struct {
	Results  map[domain.ProviderType]*SyncResult
	Errors   map[domain.ProviderType]error
	Total    *SyncResult
}

// NewMultiSyncResult creates a new multi-sync result.
func NewMultiSyncResult() *MultiSyncResult {
	return &MultiSyncResult{
		Results: make(map[domain.ProviderType]*SyncResult),
		Errors:  make(map[domain.ProviderType]error),
		Total:   &SyncResult{},
	}
}

// AddResult adds a provider's sync result.
func (m *MultiSyncResult) AddResult(provider domain.ProviderType, result *SyncResult) {
	m.Results[provider] = result
	if result != nil {
		m.Total.Created += result.Created
		m.Total.Updated += result.Updated
		m.Total.Deleted += result.Deleted
		m.Total.Failed += result.Failed
	}
}

// AddError adds a provider's error.
func (m *MultiSyncResult) AddError(provider domain.ProviderType, err error) {
	m.Errors[provider] = err
}

// HasErrors returns true if any provider had an error.
func (m *MultiSyncResult) HasErrors() bool {
	return len(m.Errors) > 0
}

// SuccessCount returns the number of successful syncs.
func (m *MultiSyncResult) SuccessCount() int {
	return len(m.Results)
}

// SyncCoordinator orchestrates syncing across multiple calendar providers.
type SyncCoordinator struct {
	registry      *ProviderRegistry
	calendarRepo  domain.ConnectedCalendarRepository
}

// NewSyncCoordinator creates a new sync coordinator.
func NewSyncCoordinator(
	registry *ProviderRegistry,
	calendarRepo domain.ConnectedCalendarRepository,
) *SyncCoordinator {
	return &SyncCoordinator{
		registry:     registry,
		calendarRepo: calendarRepo,
	}
}

// SyncAll syncs blocks to all enabled push calendars for a user.
func (c *SyncCoordinator) SyncAll(ctx context.Context, userID uuid.UUID, blocks []TimeBlock) (*MultiSyncResult, error) {
	calendars, err := c.calendarRepo.FindEnabledPushCalendars(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find push calendars: %w", err)
	}

	result := NewMultiSyncResult()

	for _, cal := range calendars {
		syncer, err := c.registry.CreateSyncer(ctx, cal)
		if err != nil {
			result.AddError(cal.Provider(), fmt.Errorf("failed to create syncer: %w", err))
			continue
		}

		syncResult, err := syncer.Sync(ctx, userID, blocks)
		if err != nil {
			result.AddError(cal.Provider(), err)
			continue
		}

		result.AddResult(cal.Provider(), syncResult)
		cal.MarkSynced()
		if saveErr := c.calendarRepo.Save(ctx, cal); saveErr != nil {
			// Log but don't fail the sync
			result.AddError(cal.Provider(), fmt.Errorf("sync succeeded but failed to update last sync time: %w", saveErr))
		}
	}

	return result, nil
}

// SyncToProvider syncs blocks to a specific provider for a user.
func (c *SyncCoordinator) SyncToProvider(ctx context.Context, userID uuid.UUID, provider domain.ProviderType, blocks []TimeBlock) (*SyncResult, error) {
	calendars, err := c.calendarRepo.FindByUserAndProvider(ctx, userID, provider)
	if err != nil {
		return nil, fmt.Errorf("failed to find calendars: %w", err)
	}

	if len(calendars) == 0 {
		return nil, fmt.Errorf("no %s calendar connected", provider.DisplayName())
	}

	// Use first enabled calendar for the provider
	var targetCal *domain.ConnectedCalendar
	for _, cal := range calendars {
		if cal.IsEnabled() && cal.SyncPush() {
			targetCal = cal
			break
		}
	}

	if targetCal == nil {
		return nil, fmt.Errorf("no enabled %s calendar with push sync", provider.DisplayName())
	}

	syncer, err := c.registry.CreateSyncer(ctx, targetCal)
	if err != nil {
		return nil, fmt.Errorf("failed to create syncer: %w", err)
	}

	result, err := syncer.Sync(ctx, userID, blocks)
	if err != nil {
		return nil, err
	}

	targetCal.MarkSynced()
	if saveErr := c.calendarRepo.Save(ctx, targetCal); saveErr != nil {
		// Log but don't fail the sync
	}

	return result, nil
}

// GetPrimaryImporter returns an importer for the user's primary calendar.
func (c *SyncCoordinator) GetPrimaryImporter(ctx context.Context, userID uuid.UUID) (Importer, *domain.ConnectedCalendar, error) {
	primary, err := c.calendarRepo.FindPrimaryForUser(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find primary calendar: %w", err)
	}

	if primary == nil {
		// Try to find any enabled pull calendar
		calendars, err := c.calendarRepo.FindEnabledPullCalendars(ctx, userID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to find pull calendars: %w", err)
		}
		if len(calendars) > 0 {
			primary = calendars[0]
		}
	}

	if primary == nil {
		return nil, nil, fmt.Errorf("no calendar configured for import")
	}

	importer, err := c.registry.CreateImporter(ctx, primary)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create importer: %w", err)
	}

	return importer, primary, nil
}

// GetImporterForProvider returns an importer for a specific provider.
func (c *SyncCoordinator) GetImporterForProvider(ctx context.Context, userID uuid.UUID, provider domain.ProviderType) (Importer, *domain.ConnectedCalendar, error) {
	calendars, err := c.calendarRepo.FindByUserAndProvider(ctx, userID, provider)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find calendars: %w", err)
	}

	if len(calendars) == 0 {
		return nil, nil, fmt.Errorf("no %s calendar connected", provider.DisplayName())
	}

	// Use first enabled calendar for the provider
	var targetCal *domain.ConnectedCalendar
	for _, cal := range calendars {
		if cal.IsEnabled() {
			targetCal = cal
			break
		}
	}

	if targetCal == nil {
		return nil, nil, fmt.Errorf("no enabled %s calendar", provider.DisplayName())
	}

	importer, err := c.registry.CreateImporter(ctx, targetCal)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create importer: %w", err)
	}

	return importer, targetCal, nil
}

// ListConnectedCalendars returns all connected calendars for a user.
func (c *SyncCoordinator) ListConnectedCalendars(ctx context.Context, userID uuid.UUID) ([]*domain.ConnectedCalendar, error) {
	return c.calendarRepo.FindByUser(ctx, userID)
}
