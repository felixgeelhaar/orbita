package commands

import (
	"context"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/inbox/domain"
	"github.com/felixgeelhaar/orbita/internal/inbox/services"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type stubInboxRepoForCapture struct {
	saved domain.InboxItem
}

func (s *stubInboxRepoForCapture) Save(ctx context.Context, item domain.InboxItem) error {
	s.saved = item
	return nil
}

func (s *stubInboxRepoForCapture) ListByUser(ctx context.Context, userID uuid.UUID, includePromoted bool) ([]domain.InboxItem, error) {
	return nil, nil
}

func (s *stubInboxRepoForCapture) FindByID(ctx context.Context, userID, id uuid.UUID) (*domain.InboxItem, error) {
	return nil, nil
}

func (s *stubInboxRepoForCapture) MarkPromoted(ctx context.Context, id uuid.UUID, promotedTo string, promotedID uuid.UUID, promotedAt time.Time) error {
	return nil
}

type stubUnitOfWork struct{}

func (stubUnitOfWork) Begin(ctx context.Context) (context.Context, error) { return ctx, nil }
func (stubUnitOfWork) Commit(ctx context.Context) error                   { return nil }
func (stubUnitOfWork) Rollback(ctx context.Context) error                 { return nil }

func TestCaptureInboxItemHandler_Handle(t *testing.T) {
	repo := &stubInboxRepoForCapture{}
	handler := NewCaptureInboxItemHandler(repo, services.NewClassifier(), stubUnitOfWork{})
	userID := uuid.New()

	cmd := CaptureInboxItemCommand{
		UserID:  userID,
		Content: "Prepare meeting notes",
		Metadata: domain.InboxMetadata{
			"type": "meeting",
		},
		Tags:   []string{"sync", "prep"},
		Source: "cli",
	}

	result, err := handler.Handle(context.Background(), cmd)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, repo.saved.ID, result.ItemID)
	require.Equal(t, userID, repo.saved.UserID)
	require.Equal(t, "meeting", repo.saved.Classification)
	require.Contains(t, cmd.Tags, repo.saved.Tags[0])
	require.NotZero(t, repo.saved.CapturedAt)
}
