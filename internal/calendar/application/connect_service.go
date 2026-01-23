package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/felixgeelhaar/orbita/internal/calendar/domain"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/google/uuid"
)

// ConnectCalendarCommand contains the data needed to connect a calendar.
type ConnectCalendarCommand struct {
	UserID       uuid.UUID
	Provider     domain.ProviderType
	CalendarID   string
	Name         string
	SetAsPrimary bool
	EnablePush   bool
	EnablePull   bool
	Config       map[string]string
}

// ConnectCalendarResult is the result of connecting a calendar.
type ConnectCalendarResult struct {
	Calendar  *domain.ConnectedCalendar
	IsUpdate  bool
	WasPrimary bool // True if this calendar was already primary
}

// OAuthService defines the interface for OAuth operations.
// This is used by the application layer to handle OAuth flows.
type OAuthService interface {
	// AuthURL returns the OAuth authorization URL with the given state.
	AuthURL(state string) string
	// ExchangeAndStore exchanges the auth code for tokens and stores them.
	ExchangeAndStore(ctx context.Context, userID uuid.UUID, code string) (any, error)
}

// OAuthServiceProvider returns an OAuthService for a given provider.
type OAuthServiceProvider func(provider domain.ProviderType) OAuthService

// CalDAVCredentialStore stores CalDAV credentials securely.
type CalDAVCredentialStore interface {
	StoreCredentials(ctx context.Context, userID uuid.UUID, provider domain.ProviderType, username, password string) error
}

// CalDAVCredentialValidator validates CalDAV credentials before storing.
type CalDAVCredentialValidator interface {
	ValidateCredentials(ctx context.Context, serverURL, username, password string) error
}

// UnitOfWork defines the interface for transaction management.
type UnitOfWork interface {
	Begin(ctx context.Context) (context.Context, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// ConnectCalendarService handles the use case of connecting a calendar.
type ConnectCalendarService struct {
	calendarRepo domain.ConnectedCalendarRepository
	outboxRepo   outbox.Repository
	uow          UnitOfWork
	logger       *slog.Logger
}

// NewConnectCalendarService creates a new ConnectCalendarService.
func NewConnectCalendarService(
	repo domain.ConnectedCalendarRepository,
	outboxRepo outbox.Repository,
	uow UnitOfWork,
	logger *slog.Logger,
) *ConnectCalendarService {
	if logger == nil {
		logger = slog.Default()
	}
	return &ConnectCalendarService{
		calendarRepo: repo,
		outboxRepo:   outboxRepo,
		uow:          uow,
		logger:       logger,
	}
}

// Connect connects a new calendar or updates an existing one.
// This handles the primary calendar logic atomically.
func (s *ConnectCalendarService) Connect(ctx context.Context, cmd ConnectCalendarCommand) (*ConnectCalendarResult, error) {
	// Start transaction if UoW is available
	txCtx := ctx
	var committed bool
	if s.uow != nil {
		var err error
		txCtx, err = s.uow.Begin(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer func() {
			if !committed {
				if rollbackErr := s.uow.Rollback(txCtx); rollbackErr != nil {
					s.logger.Error("failed to rollback transaction",
						slog.String("operation", "connect_calendar"),
						slog.String("user_id", cmd.UserID.String()),
						slog.String("error", rollbackErr.Error()),
					)
				}
			}
		}()
	}

	// Check if already connected
	existing, err := s.calendarRepo.FindByUserProviderAndCalendar(txCtx, cmd.UserID, cmd.Provider, cmd.CalendarID)
	if err == nil && existing != nil {
		// Update existing calendar
		result, updateErr := s.updateExistingCalendar(txCtx, existing, cmd)
		if updateErr != nil {
			return nil, updateErr
		}

		// Save domain events to outbox within transaction
		if saveErr := s.saveEventsToOutbox(txCtx, existing); saveErr != nil {
			return nil, fmt.Errorf("failed to save events to outbox: %w", saveErr)
		}

		if s.uow != nil {
			if commitErr := s.uow.Commit(txCtx); commitErr != nil {
				return nil, fmt.Errorf("failed to commit transaction: %w", commitErr)
			}
			committed = true
		}

		return result, nil
	}

	// Create new calendar
	result, createErr := s.createNewCalendar(txCtx, cmd)
	if createErr != nil {
		return nil, createErr
	}

	// Save domain events to outbox within transaction
	if saveErr := s.saveEventsToOutbox(txCtx, result.Calendar); saveErr != nil {
		return nil, fmt.Errorf("failed to save events to outbox: %w", saveErr)
	}

	if s.uow != nil {
		if commitErr := s.uow.Commit(txCtx); commitErr != nil {
			return nil, fmt.Errorf("failed to commit transaction: %w", commitErr)
		}
		committed = true
	}

	return result, nil
}

func (s *ConnectCalendarService) updateExistingCalendar(ctx context.Context, calendar *domain.ConnectedCalendar, cmd ConnectCalendarCommand) (*ConnectCalendarResult, error) {
	wasPrimary := calendar.IsPrimary()

	calendar.SetName(cmd.Name)
	calendar.SetSyncPush(cmd.EnablePush)
	calendar.SetSyncPull(cmd.EnablePull)

	for k, v := range cmd.Config {
		calendar.SetConfig(k, v)
	}

	if cmd.SetAsPrimary && !calendar.IsPrimary() {
		prevID, err := s.clearExistingPrimary(ctx, cmd.UserID, calendar.ID())
		if err != nil {
			return nil, err
		}
		calendar.SetPrimary(true, prevID)
	}

	if err := s.calendarRepo.Save(ctx, calendar); err != nil {
		return nil, fmt.Errorf("failed to save calendar: %w", err)
	}

	return &ConnectCalendarResult{
		Calendar:   calendar,
		IsUpdate:   true,
		WasPrimary: wasPrimary,
	}, nil
}

func (s *ConnectCalendarService) createNewCalendar(ctx context.Context, cmd ConnectCalendarCommand) (*ConnectCalendarResult, error) {
	calendar, err := domain.NewConnectedCalendar(cmd.UserID, cmd.Provider, cmd.CalendarID, cmd.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar: %w", err)
	}
	calendar.SetSyncPush(cmd.EnablePush)
	calendar.SetSyncPull(cmd.EnablePull)

	for k, v := range cmd.Config {
		calendar.SetConfig(k, v)
	}

	var previousPrimaryID *uuid.UUID
	if cmd.SetAsPrimary {
		prevID, err := s.clearExistingPrimary(ctx, cmd.UserID, calendar.ID())
		if err != nil {
			return nil, err
		}
		previousPrimaryID = prevID
		calendar.SetPrimary(true, previousPrimaryID)
	}

	if err := s.calendarRepo.Save(ctx, calendar); err != nil {
		return nil, fmt.Errorf("failed to save calendar: %w", err)
	}

	return &ConnectCalendarResult{
		Calendar:   calendar,
		IsUpdate:   false,
		WasPrimary: false,
	}, nil
}

// clearExistingPrimary clears the primary flag from any existing primary calendar.
// Returns the ID of the previous primary calendar if there was one.
func (s *ConnectCalendarService) clearExistingPrimary(ctx context.Context, userID uuid.UUID, excludeID uuid.UUID) (*uuid.UUID, error) {
	existing, err := s.calendarRepo.FindPrimaryForUser(ctx, userID)
	if err != nil {
		// No primary found, that's OK
		return nil, nil
	}

	if existing != nil && existing.ID() != excludeID {
		existing.ClearPrimary()
		if err := s.calendarRepo.Save(ctx, existing); err != nil {
			return nil, fmt.Errorf("failed to clear previous primary: %w", err)
		}
		id := existing.ID()
		return &id, nil
	}

	return nil, nil
}

func (s *ConnectCalendarService) saveEventsToOutbox(ctx context.Context, calendar *domain.ConnectedCalendar) error {
	if s.outboxRepo == nil {
		return nil
	}

	events := calendar.DomainEvents()
	if len(events) == 0 {
		return nil
	}

	msgs := make([]*outbox.Message, 0, len(events))
	for _, event := range events {
		msg, err := outbox.NewMessage(event)
		if err != nil {
			s.logger.Error("failed to create outbox message",
				slog.String("routing_key", event.RoutingKey()),
				slog.String("calendar_id", calendar.ID().String()),
				slog.String("user_id", calendar.UserID().String()),
				slog.String("error", err.Error()),
			)
			return err
		}
		msgs = append(msgs, msg)
	}

	if err := s.outboxRepo.SaveBatch(ctx, msgs); err != nil {
		s.logger.Error("failed to save events to outbox",
			slog.String("calendar_id", calendar.ID().String()),
			slog.String("user_id", calendar.UserID().String()),
			slog.String("error", err.Error()),
		)
		return err
	}

	calendar.ClearDomainEvents()
	return nil
}

// ConnectMultipleCommand is used to connect multiple calendars at once.
type ConnectMultipleCommand struct {
	UserID          uuid.UUID
	Provider        domain.ProviderType
	Calendars       []CalendarSelection
	SetFirstPrimary bool
	EnablePush      bool
	EnablePull      bool
	Config          map[string]string
}

// CalendarSelection represents a calendar to be connected.
type CalendarSelection struct {
	ID   string
	Name string
}

// ConnectMultipleResult contains the results of connecting multiple calendars.
type ConnectMultipleResult struct {
	Connected int
	Failed    int
	Calendars []*domain.ConnectedCalendar
	Errors    []error
}

// ConnectMultiple connects multiple calendars in a single operation.
func (s *ConnectCalendarService) ConnectMultiple(ctx context.Context, cmd ConnectMultipleCommand) (*ConnectMultipleResult, error) {
	result := &ConnectMultipleResult{
		Calendars: make([]*domain.ConnectedCalendar, 0, len(cmd.Calendars)),
		Errors:    make([]error, 0),
	}

	for i, cal := range cmd.Calendars {
		connectCmd := ConnectCalendarCommand{
			UserID:       cmd.UserID,
			Provider:     cmd.Provider,
			CalendarID:   cal.ID,
			Name:         cal.Name,
			SetAsPrimary: cmd.SetFirstPrimary && i == 0,
			EnablePush:   cmd.EnablePush,
			EnablePull:   cmd.EnablePull,
			Config:       cmd.Config,
		}

		connectResult, err := s.Connect(ctx, connectCmd)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Errorf("failed to connect %s: %w", cal.Name, err))
			continue
		}

		result.Connected++
		result.Calendars = append(result.Calendars, connectResult.Calendar)
	}

	return result, nil
}

// ListAvailableCalendars lists available calendars from a provider.
// This requires a temporary calendar to be created to access the provider's API.
func (s *ConnectCalendarService) ListAvailableCalendars(
	ctx context.Context,
	registry *ProviderRegistry,
	userID uuid.UUID,
	provider domain.ProviderType,
	config map[string]string,
) ([]Calendar, error) {
	// Create a temporary connected calendar to get an importer
	tempCal, err := domain.NewConnectedCalendar(userID, provider, "temp", "temp")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary calendar: %w", err)
	}
	for k, v := range config {
		tempCal.SetConfig(k, v)
	}
	// Clear the events from the temporary calendar since we won't save it
	tempCal.ClearDomainEvents()

	importer, err := registry.CreateImporter(ctx, tempCal)
	if err != nil {
		return nil, fmt.Errorf("failed to create importer: %w", err)
	}

	calendars, err := importer.ListCalendars(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list calendars: %w", err)
	}

	return calendars, nil
}
