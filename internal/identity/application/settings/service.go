package settings

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines storage for user settings.
type Repository interface {
	GetCalendarID(ctx context.Context, userID uuid.UUID) (string, error)
	SetCalendarID(ctx context.Context, userID uuid.UUID, calendarID string) error
	GetDeleteMissing(ctx context.Context, userID uuid.UUID) (bool, error)
	SetDeleteMissing(ctx context.Context, userID uuid.UUID, deleteMissing bool) error
}

// Service manages user settings.
type Service struct {
	repo Repository
}

// NewService creates a settings service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// GetCalendarID returns the calendar ID for a user.
func (s *Service) GetCalendarID(ctx context.Context, userID uuid.UUID) (string, error) {
	return s.repo.GetCalendarID(ctx, userID)
}

// SetCalendarID updates the calendar ID for a user.
func (s *Service) SetCalendarID(ctx context.Context, userID uuid.UUID, calendarID string) error {
	return s.repo.SetCalendarID(ctx, userID, calendarID)
}

// GetDeleteMissing returns the delete-missing preference.
func (s *Service) GetDeleteMissing(ctx context.Context, userID uuid.UUID) (bool, error) {
	return s.repo.GetDeleteMissing(ctx, userID)
}

// SetDeleteMissing updates the delete-missing preference.
func (s *Service) SetDeleteMissing(ctx context.Context, userID uuid.UUID, deleteMissing bool) error {
	return s.repo.SetDeleteMissing(ctx, userID, deleteMissing)
}
