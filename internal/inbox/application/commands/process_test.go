package commands

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/engine/builtin"
	"github.com/felixgeelhaar/orbita/internal/engine/sdk"
	"github.com/felixgeelhaar/orbita/internal/inbox/domain"
	"github.com/felixgeelhaar/orbita/internal/inbox/services"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockInboxRepo implements domain.InboxRepository for testing.
type mockInboxRepo struct {
	items  map[uuid.UUID]domain.InboxItem
	err    error
}

func newMockInboxRepo() *mockInboxRepo {
	return &mockInboxRepo{
		items: make(map[uuid.UUID]domain.InboxItem),
	}
}

func (m *mockInboxRepo) Save(ctx context.Context, item domain.InboxItem) error {
	if m.err != nil {
		return m.err
	}
	m.items[item.ID] = item
	return nil
}

func (m *mockInboxRepo) ListByUser(ctx context.Context, userID uuid.UUID, includePromoted bool) ([]domain.InboxItem, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []domain.InboxItem
	for _, item := range m.items {
		if item.UserID == userID {
			if includePromoted || !item.Promoted {
				result = append(result, item)
			}
		}
	}
	return result, nil
}

func (m *mockInboxRepo) FindByID(ctx context.Context, userID, id uuid.UUID) (*domain.InboxItem, error) {
	if m.err != nil {
		return nil, m.err
	}
	item, ok := m.items[id]
	if !ok || item.UserID != userID {
		return nil, nil
	}
	return &item, nil
}

func (m *mockInboxRepo) MarkPromoted(ctx context.Context, id uuid.UUID, promotedTo string, promotedID uuid.UUID, promotedAt time.Time) error {
	if m.err != nil {
		return m.err
	}
	if item, ok := m.items[id]; ok {
		item.Promoted = true
		item.PromotedTo = promotedTo
		item.PromotedID = promotedID
		item.PromotedAt = &promotedAt
		m.items[id] = item
	}
	return nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestProcessInboxItemHandler_Success(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()

	repo := newMockInboxRepo()
	repo.items[itemID] = domain.InboxItem{
		ID:         itemID,
		UserID:     userID,
		Content:    "Review PR #123 by tomorrow - high priority",
		Source:     "cli",
		CapturedAt: time.Now(),
	}

	// Use the actual ClassifierEnginePro
	classifier := builtin.NewClassifierEnginePro()
	_ = classifier.Initialize(context.Background(), sdk.EngineConfig{})

	processor := services.NewAIProcessor(classifier, testLogger())
	handler := NewProcessInboxItemHandler(repo, processor)

	result, err := handler.Handle(context.Background(), ProcessInboxItemCommand{
		UserID: userID,
		ItemID: itemID,
	})

	require.NoError(t, err)
	assert.Equal(t, itemID, result.ItemID)
	assert.NotEmpty(t, result.Classification)
	assert.Greater(t, result.Confidence, 0.0)
	assert.NotEmpty(t, result.RoutingSuggestion.Target)
}

func TestProcessInboxItemHandler_ItemNotFound(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()

	repo := newMockInboxRepo()
	classifier := builtin.NewClassifierEnginePro()
	processor := services.NewAIProcessor(classifier, testLogger())
	handler := NewProcessInboxItemHandler(repo, processor)

	_, err := handler.Handle(context.Background(), ProcessInboxItemCommand{
		UserID: userID,
		ItemID: itemID,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestProcessAllPendingHandler_MultipleItems(t *testing.T) {
	userID := uuid.New()

	repo := newMockInboxRepo()
	// Add multiple unpromoted items
	for i := 0; i < 3; i++ {
		itemID := uuid.New()
		repo.items[itemID] = domain.InboxItem{
			ID:         itemID,
			UserID:     userID,
			Content:    "Task item " + string(rune('A'+i)),
			Source:     "cli",
			CapturedAt: time.Now(),
		}
	}

	classifier := builtin.NewClassifierEnginePro()
	_ = classifier.Initialize(context.Background(), sdk.EngineConfig{})

	processor := services.NewAIProcessor(classifier, testLogger())
	handler := NewProcessAllPendingHandler(repo, processor)

	result, err := handler.Handle(context.Background(), ProcessAllPendingCommand{
		UserID: userID,
	})

	require.NoError(t, err)
	assert.Equal(t, 3, result.TotalProcessed)
	assert.Len(t, result.Results, 3)
	assert.NotEmpty(t, result.ClassificationSummary)
}

func TestProcessAllPendingHandler_NoItems(t *testing.T) {
	userID := uuid.New()

	repo := newMockInboxRepo()
	classifier := builtin.NewClassifierEnginePro()
	processor := services.NewAIProcessor(classifier, testLogger())
	handler := NewProcessAllPendingHandler(repo, processor)

	result, err := handler.Handle(context.Background(), ProcessAllPendingCommand{
		UserID: userID,
	})

	require.NoError(t, err)
	assert.Equal(t, 0, result.TotalProcessed)
	assert.Len(t, result.Results, 0)
}

func TestProcessAllPendingHandler_ExcludesPromotedItems(t *testing.T) {
	userID := uuid.New()

	repo := newMockInboxRepo()
	// Add one unpromoted and one promoted item
	unpromotedID := uuid.New()
	repo.items[unpromotedID] = domain.InboxItem{
		ID:         unpromotedID,
		UserID:     userID,
		Content:    "Unpromoted item",
		Source:     "cli",
		Promoted:   false,
		CapturedAt: time.Now(),
	}

	promotedID := uuid.New()
	promotedAt := time.Now()
	repo.items[promotedID] = domain.InboxItem{
		ID:         promotedID,
		UserID:     userID,
		Content:    "Promoted item",
		Source:     "cli",
		Promoted:   true,
		PromotedTo: "task",
		PromotedAt: &promotedAt,
		CapturedAt: time.Now(),
	}

	classifier := builtin.NewClassifierEnginePro()
	processor := services.NewAIProcessor(classifier, testLogger())
	handler := NewProcessAllPendingHandler(repo, processor)

	result, err := handler.Handle(context.Background(), ProcessAllPendingCommand{
		UserID: userID,
	})

	require.NoError(t, err)
	// Should only process the unpromoted item
	assert.Equal(t, 1, result.TotalProcessed)
}

func TestProcessInboxItemHandler_TaskClassification(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()

	repo := newMockInboxRepo()
	repo.items[itemID] = domain.InboxItem{
		ID:         itemID,
		UserID:     userID,
		Content:    "Complete the project documentation by Friday",
		Source:     "cli",
		CapturedAt: time.Now(),
	}

	classifier := builtin.NewClassifierEnginePro()
	_ = classifier.Initialize(context.Background(), sdk.EngineConfig{})

	processor := services.NewAIProcessor(classifier, testLogger())
	handler := NewProcessInboxItemHandler(repo, processor)

	result, err := handler.Handle(context.Background(), ProcessInboxItemCommand{
		UserID: userID,
		ItemID: itemID,
	})

	require.NoError(t, err)
	// Should classify as task due to actionable language
	assert.Equal(t, "task", result.Classification)
	assert.Equal(t, "task", result.RoutingSuggestion.Target)
}

func TestProcessInboxItemHandler_HabitClassification(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()

	repo := newMockInboxRepo()
	repo.items[itemID] = domain.InboxItem{
		ID:         itemID,
		UserID:     userID,
		Content:    "Exercise for 30 minutes every day",
		Source:     "cli",
		CapturedAt: time.Now(),
	}

	classifier := builtin.NewClassifierEnginePro()
	_ = classifier.Initialize(context.Background(), sdk.EngineConfig{})

	processor := services.NewAIProcessor(classifier, testLogger())
	handler := NewProcessInboxItemHandler(repo, processor)

	result, err := handler.Handle(context.Background(), ProcessInboxItemCommand{
		UserID: userID,
		ItemID: itemID,
	})

	require.NoError(t, err)
	// Should classify as habit due to recurring activity markers
	assert.Equal(t, "habit", result.Classification)
	assert.Equal(t, "habit", result.RoutingSuggestion.Target)
}

func TestProcessInboxItemHandler_MeetingClassification(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()

	repo := newMockInboxRepo()
	repo.items[itemID] = domain.InboxItem{
		ID:         itemID,
		UserID:     userID,
		Content:    "Schedule 1:1 with John to discuss project status",
		Source:     "cli",
		CapturedAt: time.Now(),
	}

	classifier := builtin.NewClassifierEnginePro()
	_ = classifier.Initialize(context.Background(), sdk.EngineConfig{})

	processor := services.NewAIProcessor(classifier, testLogger())
	handler := NewProcessInboxItemHandler(repo, processor)

	result, err := handler.Handle(context.Background(), ProcessInboxItemCommand{
		UserID: userID,
		ItemID: itemID,
	})

	require.NoError(t, err)
	// Should classify as meeting
	assert.Equal(t, "meeting", result.Classification)
	assert.Equal(t, "meeting", result.RoutingSuggestion.Target)
}

func TestProcessAllPendingHandler_ClassificationSummary(t *testing.T) {
	userID := uuid.New()

	repo := newMockInboxRepo()
	// Add items with different expected classifications
	contents := []string{
		"Complete the report",           // task
		"Exercise every morning",        // habit
		"Meeting with Sarah tomorrow",   // meeting
	}
	for _, content := range contents {
		itemID := uuid.New()
		repo.items[itemID] = domain.InboxItem{
			ID:         itemID,
			UserID:     userID,
			Content:    content,
			Source:     "cli",
			CapturedAt: time.Now(),
		}
	}

	classifier := builtin.NewClassifierEnginePro()
	_ = classifier.Initialize(context.Background(), sdk.EngineConfig{})

	processor := services.NewAIProcessor(classifier, testLogger())
	handler := NewProcessAllPendingHandler(repo, processor)

	result, err := handler.Handle(context.Background(), ProcessAllPendingCommand{
		UserID: userID,
	})

	require.NoError(t, err)
	assert.Equal(t, 3, result.TotalProcessed)
	// Summary should have entries
	assert.NotEmpty(t, result.ClassificationSummary)
	// Total classifications should equal total processed
	totalClassified := 0
	for _, count := range result.ClassificationSummary {
		totalClassified += count
	}
	assert.Equal(t, result.TotalProcessed, totalClassified)
}
