package commands

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/wellness/domain"
	"github.com/google/uuid"
)

// LogWellnessEntryCommand contains the data to log a wellness entry.
type LogWellnessEntryCommand struct {
	UserID  uuid.UUID
	Type    domain.WellnessType
	Value   int
	Date    time.Time // Optional, defaults to today
	Source  domain.WellnessSource
	Notes   string
}

// LogWellnessEntryResult contains the result of logging an entry.
type LogWellnessEntryResult struct {
	EntryID  uuid.UUID
	Type     domain.WellnessType
	Value    int
	Date     time.Time
	Source   domain.WellnessSource
	TypeInfo domain.WellnessTypeInfo
}

// LogWellnessEntryHandler handles logging wellness entries.
type LogWellnessEntryHandler struct {
	entryRepo domain.WellnessEntryRepository
	goalRepo  domain.WellnessGoalRepository
}

// NewLogWellnessEntryHandler creates a new log entry handler.
func NewLogWellnessEntryHandler(entryRepo domain.WellnessEntryRepository, goalRepo domain.WellnessGoalRepository) *LogWellnessEntryHandler {
	return &LogWellnessEntryHandler{
		entryRepo: entryRepo,
		goalRepo:  goalRepo,
	}
}

// Handle executes the log wellness entry command.
func (h *LogWellnessEntryHandler) Handle(ctx context.Context, cmd LogWellnessEntryCommand) (*LogWellnessEntryResult, error) {
	date := cmd.Date
	if date.IsZero() {
		date = time.Now()
	}

	source := cmd.Source
	if source == "" {
		source = domain.WellnessSourceManual
	}

	entry, err := domain.NewWellnessEntry(cmd.UserID, date, cmd.Type, cmd.Value, source)
	if err != nil {
		return nil, err
	}

	if cmd.Notes != "" {
		entry.SetNotes(cmd.Notes)
	}

	if err := h.entryRepo.Create(ctx, entry); err != nil {
		return nil, err
	}

	// Update any related goals
	if h.goalRepo != nil {
		goal, err := h.goalRepo.GetByUserAndType(ctx, cmd.UserID, cmd.Type)
		if err == nil && goal != nil && !goal.Achieved {
			if goal.NeedsReset() {
				goal.ResetForNewPeriod()
			}
			goal.AddProgress(cmd.Value)
			_ = h.goalRepo.Update(ctx, goal)
		}
	}

	return &LogWellnessEntryResult{
		EntryID:  entry.ID(),
		Type:     entry.Type,
		Value:    entry.Value,
		Date:     entry.Date,
		Source:   entry.Source,
		TypeInfo: domain.GetWellnessTypeInfo(entry.Type),
	}, nil
}

// WellnessCheckinCommand contains data for a quick wellness check-in.
type WellnessCheckinCommand struct {
	UserID    uuid.UUID
	Date      time.Time
	Mood      *int
	Energy    *int
	Sleep     *int
	Stress    *int
	Exercise  *int
	Hydration *int
	Nutrition *int
	Notes     string
}

// WellnessCheckinResult contains the result of a check-in.
type WellnessCheckinResult struct {
	Date         time.Time
	EntriesLogged int
	Entries      []*LogWellnessEntryResult
	GoalsUpdated int
}

// WellnessCheckinHandler handles quick wellness check-ins.
type WellnessCheckinHandler struct {
	logHandler *LogWellnessEntryHandler
}

// NewWellnessCheckinHandler creates a new check-in handler.
func NewWellnessCheckinHandler(logHandler *LogWellnessEntryHandler) *WellnessCheckinHandler {
	return &WellnessCheckinHandler{
		logHandler: logHandler,
	}
}

// Handle executes the wellness check-in command.
func (h *WellnessCheckinHandler) Handle(ctx context.Context, cmd WellnessCheckinCommand) (*WellnessCheckinResult, error) {
	date := cmd.Date
	if date.IsZero() {
		date = time.Now()
	}

	result := &WellnessCheckinResult{
		Date:    date,
		Entries: make([]*LogWellnessEntryResult, 0),
	}

	logEntry := func(wellnessType domain.WellnessType, value *int) error {
		if value == nil {
			return nil
		}
		logResult, err := h.logHandler.Handle(ctx, LogWellnessEntryCommand{
			UserID: cmd.UserID,
			Type:   wellnessType,
			Value:  *value,
			Date:   date,
			Source: domain.WellnessSourceManual,
			Notes:  cmd.Notes,
		})
		if err != nil {
			return err
		}
		result.Entries = append(result.Entries, logResult)
		result.EntriesLogged++
		return nil
	}

	// Log all provided metrics
	if err := logEntry(domain.WellnessTypeMood, cmd.Mood); err != nil {
		return nil, err
	}
	if err := logEntry(domain.WellnessTypeEnergy, cmd.Energy); err != nil {
		return nil, err
	}
	if err := logEntry(domain.WellnessTypeSleep, cmd.Sleep); err != nil {
		return nil, err
	}
	if err := logEntry(domain.WellnessTypeStress, cmd.Stress); err != nil {
		return nil, err
	}
	if err := logEntry(domain.WellnessTypeExercise, cmd.Exercise); err != nil {
		return nil, err
	}
	if err := logEntry(domain.WellnessTypeHydration, cmd.Hydration); err != nil {
		return nil, err
	}
	if err := logEntry(domain.WellnessTypeNutrition, cmd.Nutrition); err != nil {
		return nil, err
	}

	return result, nil
}
