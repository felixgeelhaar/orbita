package commands

import (
	"context"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/scheduling/application/services"
	"github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type stubScheduleRepo struct {
	schedule *domain.Schedule
}

func (s *stubScheduleRepo) Save(ctx context.Context, schedule *domain.Schedule) error {
	s.schedule = schedule
	return nil
}

func (s *stubScheduleRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Schedule, error) {
	return s.schedule, nil
}

func (s *stubScheduleRepo) FindByUserAndDate(ctx context.Context, userID uuid.UUID, date time.Time) (*domain.Schedule, error) {
	return s.schedule, nil
}

func (s *stubScheduleRepo) FindByUserDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]*domain.Schedule, error) {
	return []*domain.Schedule{s.schedule}, nil
}

func (s *stubScheduleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

type stubUnitOfWork struct{}

func (s stubUnitOfWork) Begin(ctx context.Context) (context.Context, error) { return ctx, nil }
func (s stubUnitOfWork) Commit(ctx context.Context) error                   { return nil }
func (s stubUnitOfWork) Rollback(ctx context.Context) error                 { return nil }

type stubAttemptRepo struct {
	attempts []domain.RescheduleAttempt
}

func (s *stubAttemptRepo) Create(ctx context.Context, attempt domain.RescheduleAttempt) error {
	s.attempts = append(s.attempts, attempt)
	return nil
}

func (s *stubAttemptRepo) ListByUserAndDate(ctx context.Context, userID uuid.UUID, date time.Time) ([]domain.RescheduleAttempt, error) {
	return s.attempts, nil
}

func TestAutoReschedule_MissedBlocks(t *testing.T) {
	userID := uuid.New()
	date := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)

	schedule := domain.NewSchedule(userID, date)
	block1Start := time.Date(2024, time.January, 1, 9, 0, 0, 0, time.UTC)
	block1End := block1Start.Add(60 * time.Minute)
	block1, err := schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Missed", block1Start, block1End)
	require.NoError(t, err)

	block2Start := time.Date(2024, time.January, 1, 12, 0, 0, 0, time.UTC)
	block2End := block2Start.Add(60 * time.Minute)
	_, err = schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Occupied", block2Start, block2End)
	require.NoError(t, err)

	require.NoError(t, schedule.MissBlock(block1.ID()))
	schedule.ClearDomainEvents()

	repo := &stubScheduleRepo{schedule: schedule}
	attemptRepo := &stubAttemptRepo{}
	handler := NewAutoRescheduleHandler(repo, attemptRepo, outbox.NewInMemoryRepository(), stubUnitOfWork{}, services.NewSchedulerEngine(services.DefaultSchedulerConfig()))

	result, err := handler.Handle(context.Background(), AutoRescheduleCommand{UserID: userID, Date: date})
	require.NoError(t, err)
	require.Equal(t, 1, result.Rescheduled)
	require.Equal(t, 0, result.Failed)
	require.False(t, block1.IsMissed())
	require.Len(t, attemptRepo.attempts, 1)
	require.True(t, attemptRepo.attempts[0].Success)
}

func TestAutoReschedule_RespectsAfterTime(t *testing.T) {
	userID := uuid.New()
	date := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)

	schedule := domain.NewSchedule(userID, date)
	block1Start := time.Date(2024, time.January, 1, 9, 0, 0, 0, time.UTC)
	block1End := block1Start.Add(60 * time.Minute)
	block1, err := schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Missed", block1Start, block1End)
	require.NoError(t, err)

	occupiedStart := time.Date(2024, time.January, 1, 12, 0, 0, 0, time.UTC)
	occupiedEnd := occupiedStart.Add(60 * time.Minute)
	_, err = schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Occupied", occupiedStart, occupiedEnd)
	require.NoError(t, err)

	require.NoError(t, schedule.MissBlock(block1.ID()))
	schedule.ClearDomainEvents()

	repo := &stubScheduleRepo{schedule: schedule}
	attemptRepo := &stubAttemptRepo{}
	handler := NewAutoRescheduleHandler(repo, attemptRepo, outbox.NewInMemoryRepository(), stubUnitOfWork{}, services.NewSchedulerEngine(services.DefaultSchedulerConfig()))

	after := time.Date(2024, time.January, 1, 13, 0, 0, 0, time.UTC)
	result, err := handler.Handle(context.Background(), AutoRescheduleCommand{
		UserID: userID,
		Date:   date,
		After:  &after,
	})
	require.NoError(t, err)
	require.Equal(t, 1, result.Rescheduled)
	require.Equal(t, 0, result.Failed)
	require.False(t, block1.IsMissed())

	minBreak := services.DefaultSchedulerConfig().MinBreakBetween
	require.False(t, block1.StartTime().Before(after.Add(minBreak)))
	require.Len(t, attemptRepo.attempts, 1)
	require.True(t, attemptRepo.attempts[0].Success)
}

func TestAutoReschedule_FailsWhenNoSlots(t *testing.T) {
	userID := uuid.New()
	date := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)

	schedule := domain.NewSchedule(userID, date)
	block1Start := time.Date(2024, time.January, 1, 8, 0, 0, 0, time.UTC)
	block1End := block1Start.Add(60 * time.Minute)
	block1, err := schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Missed", block1Start, block1End)
	require.NoError(t, err)

	occupiedStart := time.Date(2024, time.January, 1, 9, 0, 0, 0, time.UTC)
	occupiedEnd := occupiedStart.Add(8 * time.Hour)
	_, err = schedule.AddBlock(domain.BlockTypeTask, uuid.New(), "Occupied", occupiedStart, occupiedEnd)
	require.NoError(t, err)

	require.NoError(t, schedule.MissBlock(block1.ID()))
	schedule.ClearDomainEvents()

	repo := &stubScheduleRepo{schedule: schedule}
	attemptRepo := &stubAttemptRepo{}
	handler := NewAutoRescheduleHandler(repo, attemptRepo, outbox.NewInMemoryRepository(), stubUnitOfWork{}, services.NewSchedulerEngine(services.DefaultSchedulerConfig()))

	result, err := handler.Handle(context.Background(), AutoRescheduleCommand{UserID: userID, Date: date})
	require.NoError(t, err)
	require.Equal(t, 0, result.Rescheduled)
	require.Equal(t, 1, result.Failed)
	require.True(t, block1.IsMissed())
	require.Len(t, attemptRepo.attempts, 1)
	require.False(t, attemptRepo.attempts[0].Success)
}
