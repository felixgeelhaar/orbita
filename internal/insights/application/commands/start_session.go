// Package commands contains command handlers for the insights bounded context.
package commands

import (
	"context"
	"errors"

	"github.com/felixgeelhaar/orbita/internal/insights/domain"
	"github.com/google/uuid"
)

// StartSessionCommand represents the command to start a focus session.
type StartSessionCommand struct {
	UserID      uuid.UUID
	SessionType domain.SessionType
	Title       string
	Category    string
	ReferenceID *uuid.UUID
}

// StartSessionHandler handles start session commands.
type StartSessionHandler struct {
	sessionRepo domain.SessionRepository
}

// NewStartSessionHandler creates a new start session handler.
func NewStartSessionHandler(sessionRepo domain.SessionRepository) *StartSessionHandler {
	return &StartSessionHandler{
		sessionRepo: sessionRepo,
	}
}

// Handle executes the start session command.
func (h *StartSessionHandler) Handle(ctx context.Context, cmd StartSessionCommand) (*domain.TimeSession, error) {
	// Check for existing active session
	existing, err := h.sessionRepo.GetActive(ctx, cmd.UserID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, ErrSessionAlreadyActive
	}

	// Create new session
	session := domain.NewTimeSession(cmd.UserID, cmd.SessionType, cmd.Title)
	if cmd.Category != "" {
		session.WithCategory(cmd.Category)
	}
	if cmd.ReferenceID != nil {
		session.WithReference(*cmd.ReferenceID)
	}

	if err := h.sessionRepo.Create(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}

// ErrSessionAlreadyActive indicates a session is already active.
var ErrSessionAlreadyActive = errors.New("a session is already active")

// ErrNotFound indicates a resource was not found.
var ErrNotFound = errors.New("not found")
