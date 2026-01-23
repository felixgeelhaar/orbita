package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/felixgeelhaar/orbita/internal/calendar/domain"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/google/uuid"
)

// DisconnectCalendarCommand contains the data needed to disconnect a calendar.
type DisconnectCalendarCommand struct {
	UserID   uuid.UUID
	Provider domain.ProviderType
}

// DisconnectCalendarResult is the result of disconnecting calendars.
type DisconnectCalendarResult struct {
	Disconnected    int
	CalendarNames   []string
	HadPrimary      bool
}

// DisconnectCalendarService handles the use case of disconnecting calendars.
type DisconnectCalendarService struct {
	calendarRepo domain.ConnectedCalendarRepository
	outboxRepo   outbox.Repository
	uow          UnitOfWork
	logger       *slog.Logger
}

// NewDisconnectCalendarService creates a new DisconnectCalendarService.
func NewDisconnectCalendarService(
	repo domain.ConnectedCalendarRepository,
	outboxRepo outbox.Repository,
	uow UnitOfWork,
	logger *slog.Logger,
) *DisconnectCalendarService {
	if logger == nil {
		logger = slog.Default()
	}
	return &DisconnectCalendarService{
		calendarRepo: repo,
		outboxRepo:   outboxRepo,
		uow:          uow,
		logger:       logger,
	}
}

// DisconnectByProvider disconnects all calendars for a user from a specific provider.
func (s *DisconnectCalendarService) DisconnectByProvider(ctx context.Context, cmd DisconnectCalendarCommand) (*DisconnectCalendarResult, error) {
	// Find all calendars for this provider
	calendars, err := s.calendarRepo.FindByUserAndProvider(ctx, cmd.UserID, cmd.Provider)
	if err != nil {
		return nil, fmt.Errorf("failed to find calendars: %w", err)
	}

	if len(calendars) == 0 {
		return &DisconnectCalendarResult{
			Disconnected:  0,
			CalendarNames: []string{},
			HadPrimary:    false,
		}, nil
	}

	// Start transaction if UoW is available
	txCtx := ctx
	var committed bool
	if s.uow != nil {
		var beginErr error
		txCtx, beginErr = s.uow.Begin(ctx)
		if beginErr != nil {
			return nil, fmt.Errorf("failed to begin transaction: %w", beginErr)
		}
		defer func() {
			if !committed {
				if rollbackErr := s.uow.Rollback(txCtx); rollbackErr != nil {
					s.logger.Error("failed to rollback transaction",
						slog.String("operation", "disconnect_by_provider"),
						slog.String("user_id", cmd.UserID.String()),
						slog.String("provider", string(cmd.Provider)),
						slog.String("error", rollbackErr.Error()),
					)
				}
			}
		}()
	}

	result := &DisconnectCalendarResult{
		CalendarNames: make([]string, 0, len(calendars)),
	}

	// Mark calendars as disconnected and collect names
	for _, cal := range calendars {
		cal.MarkDisconnected()
		result.CalendarNames = append(result.CalendarNames, cal.Name())
		if cal.IsPrimary() {
			result.HadPrimary = true
		}
	}

	// Delete all calendars for this provider
	if deleteErr := s.calendarRepo.DeleteByUserAndProvider(txCtx, cmd.UserID, cmd.Provider); deleteErr != nil {
		return nil, fmt.Errorf("failed to delete calendars: %w", deleteErr)
	}

	result.Disconnected = len(calendars)

	// Save domain events to outbox within transaction
	for _, cal := range calendars {
		if saveErr := s.saveEventsToOutbox(txCtx, cal); saveErr != nil {
			return nil, fmt.Errorf("failed to save events to outbox: %w", saveErr)
		}
	}

	if s.uow != nil {
		if commitErr := s.uow.Commit(txCtx); commitErr != nil {
			return nil, fmt.Errorf("failed to commit transaction: %w", commitErr)
		}
		committed = true
	}

	return result, nil
}

// DisconnectByID disconnects a specific calendar by ID.
func (s *DisconnectCalendarService) DisconnectByID(ctx context.Context, userID uuid.UUID, calendarID uuid.UUID) (*DisconnectCalendarResult, error) {
	calendar, err := s.calendarRepo.FindByID(ctx, calendarID)
	if err != nil {
		return nil, fmt.Errorf("calendar not found: %w", err)
	}

	if calendar.UserID() != userID {
		return nil, fmt.Errorf("access denied") // Don't reveal calendar existence
	}

	// Start transaction if UoW is available
	txCtx := ctx
	var committed bool
	if s.uow != nil {
		var beginErr error
		txCtx, beginErr = s.uow.Begin(ctx)
		if beginErr != nil {
			return nil, fmt.Errorf("failed to begin transaction: %w", beginErr)
		}
		defer func() {
			if !committed {
				if rollbackErr := s.uow.Rollback(txCtx); rollbackErr != nil {
					s.logger.Error("failed to rollback transaction",
						slog.String("operation", "disconnect_by_id"),
						slog.String("user_id", userID.String()),
						slog.String("calendar_id", calendarID.String()),
						slog.String("error", rollbackErr.Error()),
					)
				}
			}
		}()
	}

	// Mark as disconnected
	calendar.MarkDisconnected()

	result := &DisconnectCalendarResult{
		Disconnected:  1,
		CalendarNames: []string{calendar.Name()},
		HadPrimary:    calendar.IsPrimary(),
	}

	// Delete the calendar
	if deleteErr := s.calendarRepo.Delete(txCtx, calendarID); deleteErr != nil {
		return nil, fmt.Errorf("failed to delete calendar: %w", deleteErr)
	}

	// Save domain events to outbox within transaction
	if saveErr := s.saveEventsToOutbox(txCtx, calendar); saveErr != nil {
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

func (s *DisconnectCalendarService) saveEventsToOutbox(ctx context.Context, calendar *domain.ConnectedCalendar) error {
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

// ListConnectedCalendars returns all connected calendars for a user.
func (s *DisconnectCalendarService) ListConnectedCalendars(ctx context.Context, userID uuid.UUID) ([]*domain.ConnectedCalendar, error) {
	return s.calendarRepo.FindByUser(ctx, userID)
}

// GetCalendarsByProvider returns calendars for a specific provider.
func (s *DisconnectCalendarService) GetCalendarsByProvider(ctx context.Context, userID uuid.UUID, provider domain.ProviderType) ([]*domain.ConnectedCalendar, error) {
	return s.calendarRepo.FindByUserAndProvider(ctx, userID, provider)
}
