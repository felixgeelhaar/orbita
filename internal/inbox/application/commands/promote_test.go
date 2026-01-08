package commands

import (
	"context"
	"testing"
	"time"

	habitCommands "github.com/felixgeelhaar/orbita/internal/habits/application/commands"
	"github.com/felixgeelhaar/orbita/internal/inbox/domain"
	meetingCommands "github.com/felixgeelhaar/orbita/internal/meetings/application/commands"
	productivityCommands "github.com/felixgeelhaar/orbita/internal/productivity/application/commands"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type stubInboxRepoForPromote struct {
	findItem *domain.InboxItem
	mark     struct {
		called     bool
		id         uuid.UUID
		promotedTo string
		promotedID uuid.UUID
		promotedAt time.Time
	}
}

func (s *stubInboxRepoForPromote) Save(ctx context.Context, item domain.InboxItem) error {
	return nil
}

func (s *stubInboxRepoForPromote) ListByUser(ctx context.Context, userID uuid.UUID, includePromoted bool) ([]domain.InboxItem, error) {
	return nil, nil
}

func (s *stubInboxRepoForPromote) FindByID(ctx context.Context, userID, id uuid.UUID) (*domain.InboxItem, error) {
	return s.findItem, nil
}

func (s *stubInboxRepoForPromote) MarkPromoted(ctx context.Context, id uuid.UUID, promotedTo string, promotedID uuid.UUID, promotedAt time.Time) error {
	s.mark.called = true
	s.mark.id = id
	s.mark.promotedTo = promotedTo
	s.mark.promotedID = promotedID
	s.mark.promotedAt = promotedAt
	return nil
}

type stubTaskHandler struct {
	last   productivityCommands.CreateTaskCommand
	result productivityCommands.CreateTaskResult
}

func (s *stubTaskHandler) Handle(ctx context.Context, cmd productivityCommands.CreateTaskCommand) (*productivityCommands.CreateTaskResult, error) {
	s.last = cmd
	if s.result.TaskID == uuid.Nil {
		s.result.TaskID = uuid.New()
	}
	return &s.result, nil
}

type stubHabitHandler struct {
	result habitCommands.CreateHabitResult
}

func (s *stubHabitHandler) Handle(ctx context.Context, cmd habitCommands.CreateHabitCommand) (*habitCommands.CreateHabitResult, error) {
	if s.result.HabitID == uuid.Nil {
		s.result.HabitID = uuid.New()
	}
	return &s.result, nil
}

type stubMeetingHandler struct {
	result meetingCommands.CreateMeetingResult
}

func (s *stubMeetingHandler) Handle(ctx context.Context, cmd meetingCommands.CreateMeetingCommand) (*meetingCommands.CreateMeetingResult, error) {
	if s.result.MeetingID == uuid.Nil {
		s.result.MeetingID = uuid.New()
	}
	return &s.result, nil
}

func TestPromoteInboxItemHandler_HandleTaskPromotion(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	repo := &stubInboxRepoForPromote{
		findItem: &domain.InboxItem{
			ID:      itemID,
			UserID:  userID,
			Content: "Prep strategy memo",
		},
	}
	taskHandler := &stubTaskHandler{}
	handler := NewPromoteInboxItemHandler(repo, taskHandler, &stubHabitHandler{}, &stubMeetingHandler{})

	result, err := handler.Handle(context.Background(), PromoteInboxItemCommand{
		UserID: userID,
		ItemID: itemID,
		Target: PromoteTargetTask,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, PromoteTargetTask, result.Target)
	require.True(t, repo.mark.called)
	require.Equal(t, itemID, repo.mark.id)
	require.Equal(t, string(PromoteTargetTask), repo.mark.promotedTo)
	require.Equal(t, taskHandler.result.TaskID, result.PromotedID)
	require.Equal(t, result.PromotedID, repo.mark.promotedID)
	require.False(t, repo.mark.promotedAt.IsZero())
	require.Equal(t, "Prep strategy memo", taskHandler.last.Title)
	require.Equal(t, userID, taskHandler.last.UserID)
}
