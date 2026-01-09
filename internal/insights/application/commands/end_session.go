package commands

import (
	"context"
	"errors"

	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/google/uuid"
)

// EndSessionCommand represents the command to end a focus session.
type EndSessionCommand struct {
	UserID uuid.UUID
	Status domain.SessionStatus
	Notes  string
}

// EndSessionHandler handles end session commands.
type EndSessionHandler struct {
	sessionRepo domain.SessionRepository
}

// NewEndSessionHandler creates a new end session handler.
func NewEndSessionHandler(sessionRepo domain.SessionRepository) *EndSessionHandler {
	return &EndSessionHandler{
		sessionRepo: sessionRepo,
	}
}

// Handle executes the end session command.
func (h *EndSessionHandler) Handle(ctx context.Context, cmd EndSessionCommand) (*domain.TimeSession, error) {
	// Get active session
	session, err := h.sessionRepo.GetActive(ctx, cmd.UserID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNoActiveSession
		}
		return nil, err
	}
	if session == nil {
		return nil, ErrNoActiveSession
	}

	// End the session
	if err := session.End(cmd.Status); err != nil {
		return nil, err
	}

	// Add notes if provided
	if cmd.Notes != "" {
		session.AddNotes(cmd.Notes)
	}

	// Update in repository
	if err := h.sessionRepo.Update(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}

// ErrNoActiveSession indicates no active session was found.
var ErrNoActiveSession = errors.New("no active session found")
