package workers

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/felixgeelhaar/orbita/internal/calendar/application"
	"github.com/felixgeelhaar/orbita/internal/calendar/domain"
	"github.com/google/uuid"
)

// DefaultImportInterval is the default interval between import cycles.
const DefaultImportInterval = 5 * time.Minute

// DefaultLookAheadDays is how far ahead to look for events.
const DefaultLookAheadDays = 7

// DefaultMaxSyncErrors is the maximum number of consecutive sync errors before giving up.
const DefaultMaxSyncErrors = 5

// ConflictHandler handles conflicts between external events and Orbita blocks.
type ConflictHandler interface {
	HandleConflict(ctx context.Context, external application.CalendarEvent, existing interface{}) error
}

// CalendarImportWorkerConfig configures the import worker.
type CalendarImportWorkerConfig struct {
	Interval       time.Duration
	LookAheadDays  int
	MaxSyncErrors  int
	BatchSize      int
	SkipOrbitaEvents bool
}

// DefaultImportWorkerConfig returns the default configuration.
func DefaultImportWorkerConfig() CalendarImportWorkerConfig {
	return CalendarImportWorkerConfig{
		Interval:         DefaultImportInterval,
		LookAheadDays:    DefaultLookAheadDays,
		MaxSyncErrors:    DefaultMaxSyncErrors,
		BatchSize:        10,
		SkipOrbitaEvents: true,
	}
}

// CalendarImportWorker periodically imports events from external calendars.
type CalendarImportWorker struct {
	importer        application.Importer
	syncStateRepo   domain.SyncStateRepository
	conflictHandler ConflictHandler
	config          CalendarImportWorkerConfig
	logger          *slog.Logger
	running         atomic.Bool
	stopCh          chan struct{}
}

// NewCalendarImportWorker creates a new calendar import worker.
func NewCalendarImportWorker(
	importer application.Importer,
	syncStateRepo domain.SyncStateRepository,
	conflictHandler ConflictHandler,
	config CalendarImportWorkerConfig,
	logger *slog.Logger,
) *CalendarImportWorker {
	if logger == nil {
		logger = slog.Default()
	}
	return &CalendarImportWorker{
		importer:        importer,
		syncStateRepo:   syncStateRepo,
		conflictHandler: conflictHandler,
		config:          config,
		logger:          logger,
		stopCh:          make(chan struct{}),
	}
}

// Run starts the worker and blocks until context is cancelled or Stop() is called.
func (w *CalendarImportWorker) Run(ctx context.Context) error {
	if w.importer == nil {
		w.logger.Warn("calendar importer not configured, worker will not start")
		return nil
	}

	w.running.Store(true)
	w.logger.Info("calendar import worker started",
		"interval", w.config.Interval,
		"look_ahead_days", w.config.LookAheadDays,
	)

	// Run immediately on start
	w.runImportCycle(ctx)

	ticker := time.NewTicker(w.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.running.Store(false)
			w.logger.Info("calendar import worker stopped (context cancelled)")
			return ctx.Err()
		case <-w.stopCh:
			w.running.Store(false)
			w.logger.Info("calendar import worker stopped (stop signal)")
			return nil
		case <-ticker.C:
			w.runImportCycle(ctx)
		}
	}
}

// Stop signals the worker to stop gracefully.
func (w *CalendarImportWorker) Stop() {
	if w.running.Load() {
		close(w.stopCh)
	}
}

// IsRunning returns true if the worker is currently running.
func (w *CalendarImportWorker) IsRunning() bool {
	return w.running.Load()
}

// runImportCycle runs a single import cycle for all users needing sync.
func (w *CalendarImportWorker) runImportCycle(ctx context.Context) {
	w.logger.Debug("starting import cycle")

	// Find users that need syncing
	pendingStates, err := w.syncStateRepo.FindPendingSync(ctx, w.config.Interval, w.config.BatchSize)
	if err != nil {
		w.logger.Error("failed to find pending sync states", "error", err)
		return
	}

	if len(pendingStates) == 0 {
		w.logger.Debug("no users need syncing")
		return
	}

	w.logger.Debug("found users needing sync", "count", len(pendingStates))

	for _, state := range pendingStates {
		if err := ctx.Err(); err != nil {
			return // Context cancelled
		}
		w.importForUser(ctx, state)
	}

	w.logger.Debug("import cycle completed")
}

// importForUser imports events for a specific user's calendar.
func (w *CalendarImportWorker) importForUser(ctx context.Context, state *domain.SyncState) {
	w.logger.Debug("importing events for user",
		"user_id", state.UserID(),
		"calendar_id", state.CalendarID(),
	)

	// Calculate time range
	start := time.Now()
	end := start.AddDate(0, 0, w.config.LookAheadDays)

	// Fetch events from external calendar
	events, err := w.importer.ListEvents(ctx, state.UserID(), start, end, !w.config.SkipOrbitaEvents)
	if err != nil {
		w.logger.Error("failed to list events from calendar",
			"user_id", state.UserID(),
			"calendar_id", state.CalendarID(),
			"error", err,
		)
		state.MarkSyncFailure(err.Error())
		if saveErr := w.syncStateRepo.Save(ctx, state); saveErr != nil {
			w.logger.Error("failed to save sync state", "error", saveErr)
		}
		return
	}

	w.logger.Debug("fetched events from calendar",
		"user_id", state.UserID(),
		"event_count", len(events),
	)

	// Process events
	imported, skipped, conflicts := 0, 0, 0
	for _, event := range events {
		if event.IsOrbitaEvent && w.config.SkipOrbitaEvents {
			skipped++
			continue
		}

		// TODO: Check for conflicts with existing blocks
		// For now, just count external events
		if w.conflictHandler != nil {
			if err := w.conflictHandler.HandleConflict(ctx, event, nil); err != nil {
				w.logger.Warn("conflict detected for event",
					"event_id", event.ID,
					"event_summary", event.Summary,
				)
				conflicts++
				continue
			}
		}

		imported++
	}

	// Calculate sync hash for change detection
	syncHash := calculateSyncHash(events)

	// Update sync state
	state.MarkSyncSuccess("", syncHash)
	if err := w.syncStateRepo.Save(ctx, state); err != nil {
		w.logger.Error("failed to save sync state", "error", err)
		return
	}

	w.logger.Info("import completed for user",
		"user_id", state.UserID(),
		"imported", imported,
		"skipped", skipped,
		"conflicts", conflicts,
	)
}

// InitializeSyncState creates a sync state for a user if it doesn't exist.
func (w *CalendarImportWorker) InitializeSyncState(ctx context.Context, userID uuid.UUID, calendarID, provider string) (*domain.SyncState, error) {
	// Check if sync state already exists
	existing, err := w.syncStateRepo.FindByUserAndCalendar(ctx, userID, calendarID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	// Create new sync state
	state := domain.NewSyncState(userID, calendarID, provider)
	if err := w.syncStateRepo.Save(ctx, state); err != nil {
		return nil, err
	}

	w.logger.Info("initialized sync state for user",
		"user_id", userID,
		"calendar_id", calendarID,
		"provider", provider,
	)

	return state, nil
}

// ForceSync triggers an immediate sync for a user.
func (w *CalendarImportWorker) ForceSync(ctx context.Context, userID uuid.UUID, calendarID string) error {
	state, err := w.syncStateRepo.FindByUserAndCalendar(ctx, userID, calendarID)
	if err != nil {
		return err
	}
	if state == nil {
		state = domain.NewSyncState(userID, calendarID, "google")
	}

	w.importForUser(ctx, state)
	return nil
}

// calculateSyncHash creates a simple hash from events for change detection.
func calculateSyncHash(events []application.CalendarEvent) string {
	if len(events) == 0 {
		return ""
	}

	// Simple hash based on event count and first/last event IDs
	hash := ""
	if len(events) > 0 {
		hash = events[0].ID
		if len(events) > 1 {
			hash += "_" + events[len(events)-1].ID
		}
		hash += "_" + string(rune(len(events)))
	}
	return hash
}
