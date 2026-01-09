package api

import (
	"context"

	habitQueries "github.com/felixgeelhaar/orbita/internal/habits/application/queries"
	"github.com/felixgeelhaar/orbita/internal/orbit/sdk"
	"github.com/google/uuid"
)

// HabitAPIImpl implements sdk.HabitAPI with capability checking.
type HabitAPIImpl struct {
	handler      *habitQueries.ListHabitsHandler
	userID       uuid.UUID
	capabilities sdk.CapabilitySet
}

// NewHabitAPI creates a new HabitAPI implementation.
func NewHabitAPI(
	handler *habitQueries.ListHabitsHandler,
	userID uuid.UUID,
	caps sdk.CapabilitySet,
) *HabitAPIImpl {
	return &HabitAPIImpl{
		handler:      handler,
		userID:       userID,
		capabilities: caps,
	}
}

func (a *HabitAPIImpl) checkCapability() error {
	if !a.capabilities.Has(sdk.CapReadHabits) {
		return sdk.ErrCapabilityNotGranted
	}
	return nil
}

// List returns all habits for the user.
func (a *HabitAPIImpl) List(ctx context.Context) ([]sdk.HabitDTO, error) {
	if err := a.checkCapability(); err != nil {
		return nil, err
	}

	habits, err := a.handler.Handle(ctx, habitQueries.ListHabitsQuery{
		UserID:          a.userID,
		IncludeArchived: true,
	})
	if err != nil {
		return nil, err
	}

	return toHabitSDKDTOs(habits), nil
}

// Get returns a single habit by ID.
func (a *HabitAPIImpl) Get(ctx context.Context, id string) (*sdk.HabitDTO, error) {
	if err := a.checkCapability(); err != nil {
		return nil, err
	}

	// List all habits and find the one with matching ID
	// TODO: Add GetHabitHandler to habits domain for direct lookup
	habits, err := a.handler.Handle(ctx, habitQueries.ListHabitsQuery{
		UserID:          a.userID,
		IncludeArchived: true,
	})
	if err != nil {
		return nil, err
	}

	habitID, err := uuid.Parse(id)
	if err != nil {
		return nil, sdk.ErrResourceNotFound
	}

	for _, h := range habits {
		if h.ID == habitID {
			dto := toHabitSDKDTO(h)
			return &dto, nil
		}
	}

	return nil, sdk.ErrResourceNotFound
}

// GetActive returns all active (non-archived) habits.
func (a *HabitAPIImpl) GetActive(ctx context.Context) ([]sdk.HabitDTO, error) {
	if err := a.checkCapability(); err != nil {
		return nil, err
	}

	habits, err := a.handler.Handle(ctx, habitQueries.ListHabitsQuery{
		UserID:          a.userID,
		IncludeArchived: false,
	})
	if err != nil {
		return nil, err
	}

	return toHabitSDKDTOs(habits), nil
}

// GetDueToday returns habits that should be completed today.
func (a *HabitAPIImpl) GetDueToday(ctx context.Context) ([]sdk.HabitDTO, error) {
	if err := a.checkCapability(); err != nil {
		return nil, err
	}

	habits, err := a.handler.Handle(ctx, habitQueries.ListHabitsQuery{
		UserID:       a.userID,
		OnlyDueToday: true,
	})
	if err != nil {
		return nil, err
	}

	return toHabitSDKDTOs(habits), nil
}

func toHabitSDKDTOs(habits []habitQueries.HabitDTO) []sdk.HabitDTO {
	result := make([]sdk.HabitDTO, len(habits))
	for i, h := range habits {
		result[i] = toHabitSDKDTO(h)
	}
	return result
}

func toHabitSDKDTO(h habitQueries.HabitDTO) sdk.HabitDTO {
	return sdk.HabitDTO{
		ID:          h.ID.String(),
		Name:        h.Name,
		Description: h.Description,
		Frequency:   h.Frequency,
		Streak:      h.Streak,
		IsArchived:  h.IsArchived,
		CreatedAt:   h.CreatedAt,
	}
}
