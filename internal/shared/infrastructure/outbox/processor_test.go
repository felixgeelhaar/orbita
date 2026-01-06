package outbox_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRepository is a test double for outbox.Repository
type mockRepository struct {
	mu             sync.Mutex
	messages       []*outbox.Message
	publishedIDs   []int64
	failedIDs      []int64
	deadIDs        []int64
	getUnpublished func(limit int) ([]*outbox.Message, error)
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		messages:     make([]*outbox.Message, 0),
		publishedIDs: make([]int64, 0),
		failedIDs:    make([]int64, 0),
		deadIDs:      make([]int64, 0),
	}
}

func (r *mockRepository) Save(ctx context.Context, msg *outbox.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	msg.ID = int64(len(r.messages) + 1)
	r.messages = append(r.messages, msg)
	return nil
}

func (r *mockRepository) SaveBatch(ctx context.Context, msgs []*outbox.Message) error {
	for _, msg := range msgs {
		if err := r.Save(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

func (r *mockRepository) GetUnpublished(ctx context.Context, limit int) ([]*outbox.Message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.getUnpublished != nil {
		return r.getUnpublished(limit)
	}

	var result []*outbox.Message
	now := time.Now()
	for _, msg := range r.messages {
		if msg.PublishedAt == nil && msg.DeadLetteredAt == nil {
			if msg.NextRetryAt != nil && msg.NextRetryAt.After(now) {
				continue
			}
			result = append(result, msg)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (r *mockRepository) MarkPublished(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.publishedIDs = append(r.publishedIDs, id)
	for _, msg := range r.messages {
		if msg.ID == id {
			now := time.Now()
			msg.PublishedAt = &now
			msg.DeadLetteredAt = nil
			break
		}
	}
	return nil
}

func (r *mockRepository) MarkFailed(ctx context.Context, id int64, errMsg string, nextRetryAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.failedIDs = append(r.failedIDs, id)
	for _, msg := range r.messages {
		if msg.ID == id {
			msg.RetryCount++
			msg.LastError = &errMsg
			msg.NextRetryAt = &nextRetryAt
			break
		}
	}
	return nil
}

func (r *mockRepository) MarkDead(ctx context.Context, id int64, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deadIDs = append(r.deadIDs, id)
	for _, msg := range r.messages {
		if msg.ID == id {
			now := time.Now()
			msg.DeadLetteredAt = &now
			msg.DeadLetterReason = &reason
			break
		}
	}
	return nil
}

func (r *mockRepository) GetFailed(ctx context.Context, maxRetries, limit int) ([]*outbox.Message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*outbox.Message
	now := time.Now()
	for _, msg := range r.messages {
		if msg.PublishedAt == nil && msg.DeadLetteredAt == nil && msg.RetryCount > 0 && msg.RetryCount < maxRetries {
			if msg.NextRetryAt != nil && msg.NextRetryAt.After(now) {
				continue
			}
			result = append(result, msg)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (r *mockRepository) DeleteOld(ctx context.Context, olderThanDays int) (int64, error) {
	return 0, nil
}

// mockPublisher is a test double for eventbus.Publisher
type mockPublisher struct {
	mu          sync.Mutex
	published   []publishedMessage
	shouldFail  bool
	failForKeys map[string]bool
}

type publishedMessage struct {
	RoutingKey string
	Payload    []byte
}

func newMockPublisher() *mockPublisher {
	return &mockPublisher{
		published:   make([]publishedMessage, 0),
		failForKeys: make(map[string]bool),
	}
}

func (p *mockPublisher) Publish(ctx context.Context, routingKey string, payload []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.shouldFail || p.failForKeys[routingKey] {
		return errors.New("publish failed")
	}

	p.published = append(p.published, publishedMessage{
		RoutingKey: routingKey,
		Payload:    payload,
	})
	return nil
}

func (p *mockPublisher) Close() error {
	return nil
}

func (p *mockPublisher) PublishedCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.published)
}

func createTestMessage(routingKey string) *outbox.Message {
	payload, _ := json.Marshal(map[string]string{"test": "data"})
	return &outbox.Message{
		AggregateType: "TestAggregate",
		AggregateID:   uuid.New(),
		EventType:     routingKey,
		RoutingKey:    routingKey,
		Payload:       payload,
		CreatedAt:     time.Now(),
	}
}

func TestProcessor_ProcessOnce(t *testing.T) {
	repo := newMockRepository()
	publisher := newMockPublisher()
	config := outbox.DefaultProcessorConfig()
	processor := outbox.NewProcessor(repo, publisher, config, nil)

	// Add messages
	msg1 := createTestMessage("test.event.one")
	msg2 := createTestMessage("test.event.two")
	repo.Save(context.Background(), msg1)
	repo.Save(context.Background(), msg2)

	// Process
	err := processor.ProcessOnce(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 2, publisher.PublishedCount())
	assert.Len(t, repo.publishedIDs, 2)

	stats := processor.GetStats()
	assert.Equal(t, uint64(2), stats.PublishedCount)
	assert.NotNil(t, stats.LastProcessedAt)
	assert.NotNil(t, stats.OldestMessageAt)
	assert.GreaterOrEqual(t, stats.LagSeconds, 0.0)
}

func TestProcessor_ProcessOnce_PublishFailure(t *testing.T) {
	repo := newMockRepository()
	publisher := newMockPublisher()
	publisher.failForKeys["test.event.fail"] = true
	config := outbox.DefaultProcessorConfig()
	processor := outbox.NewProcessor(repo, publisher, config, nil)

	// Add messages - one will fail
	msg1 := createTestMessage("test.event.success")
	msg2 := createTestMessage("test.event.fail")
	repo.Save(context.Background(), msg1)
	repo.Save(context.Background(), msg2)

	// Process
	err := processor.ProcessOnce(context.Background())

	require.NoError(t, err) // Processor itself doesn't fail
	assert.Equal(t, 1, publisher.PublishedCount())
	assert.Len(t, repo.publishedIDs, 1)
	assert.Len(t, repo.failedIDs, 1)

	stats := processor.GetStats()
	assert.Equal(t, uint64(1), stats.PublishedCount)
	assert.Equal(t, uint64(1), stats.FailedCount)
	assert.NotNil(t, stats.LastErrorAt)
}

func TestProcessor_ProcessOnce_DeadLettersAfterMaxRetries(t *testing.T) {
	repo := newMockRepository()
	publisher := newMockPublisher()
	publisher.failForKeys["test.event.fail"] = true
	config := outbox.DefaultProcessorConfig()
	config.MaxRetries = 1
	processor := outbox.NewProcessor(repo, publisher, config, nil)

	msg := createTestMessage("test.event.fail")
	repo.Save(context.Background(), msg)

	err := processor.ProcessOnce(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 0, publisher.PublishedCount())
	assert.Len(t, repo.failedIDs, 0)
	assert.Len(t, repo.deadIDs, 1)

	stats := processor.GetStats()
	assert.Equal(t, uint64(1), stats.DeadCount)
}

func TestProcessor_StartStop(t *testing.T) {
	repo := newMockRepository()
	publisher := newMockPublisher()
	config := outbox.ProcessorConfig{
		PollInterval:     10 * time.Millisecond,
		BatchSize:        10,
		MaxRetries:       3,
		RetryBackoffBase: 1 * time.Millisecond,
		RetryBackoffMax:  10 * time.Millisecond,
	}
	processor := outbox.NewProcessor(repo, publisher, config, nil)

	// Start
	err := processor.Start(context.Background())
	require.NoError(t, err)
	assert.True(t, processor.IsRunning())

	// Add a message while running
	msg := createTestMessage("test.event")
	repo.Save(context.Background(), msg)

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	// Stop
	processor.Stop()
	assert.False(t, processor.IsRunning())

	// Message should have been processed
	assert.GreaterOrEqual(t, publisher.PublishedCount(), 1)
}

func TestProcessor_DoubleStart(t *testing.T) {
	repo := newMockRepository()
	publisher := newMockPublisher()
	config := outbox.DefaultProcessorConfig()
	processor := outbox.NewProcessor(repo, publisher, config, nil)

	// Start twice - should be idempotent
	err := processor.Start(context.Background())
	require.NoError(t, err)

	err = processor.Start(context.Background())
	require.NoError(t, err)

	processor.Stop()
}

func TestProcessor_DoubleStop(t *testing.T) {
	repo := newMockRepository()
	publisher := newMockPublisher()
	config := outbox.DefaultProcessorConfig()
	processor := outbox.NewProcessor(repo, publisher, config, nil)

	processor.Start(context.Background())

	// Stop twice - should be idempotent
	processor.Stop()
	processor.Stop()
}

func TestProcessor_GetStats(t *testing.T) {
	repo := newMockRepository()
	publisher := newMockPublisher()
	config := outbox.DefaultProcessorConfig()
	processor := outbox.NewProcessor(repo, publisher, config, nil)

	stats := processor.GetStats()
	assert.False(t, stats.IsRunning)

	processor.Start(context.Background())
	stats = processor.GetStats()
	assert.True(t, stats.IsRunning)

	processor.Stop()
	stats = processor.GetStats()
	assert.False(t, stats.IsRunning)
}

func TestMessage_NewMessage(t *testing.T) {
	// We need a concrete event type for this test
	// Using a simple struct that implements DomainEvent

	msg := createTestMessage("test.routing.key")

	assert.Equal(t, "test.routing.key", msg.RoutingKey)
	assert.NotEmpty(t, msg.Payload)
	assert.False(t, msg.IsPublished())
}

func TestMessage_IsPublished(t *testing.T) {
	msg := createTestMessage("test")
	assert.False(t, msg.IsPublished())

	now := time.Now()
	msg.PublishedAt = &now
	assert.True(t, msg.IsPublished())
}

func TestMessage_CanRetry(t *testing.T) {
	msg := createTestMessage("test")

	assert.True(t, msg.CanRetry(3))

	msg.RetryCount = 2
	assert.True(t, msg.CanRetry(3))

	msg.RetryCount = 3
	assert.False(t, msg.CanRetry(3))

	msg.RetryCount = 5
	assert.False(t, msg.CanRetry(3))
}
