package outbox

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/convert"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/eventbus"
)

// ProcessorConfig holds configuration for the outbox processor.
type ProcessorConfig struct {
	PollInterval     time.Duration
	BatchSize        int
	MaxRetries       int
	RetryBackoffBase time.Duration
	RetryBackoffMax  time.Duration
}

// DefaultProcessorConfig returns sensible defaults.
func DefaultProcessorConfig() ProcessorConfig {
	return ProcessorConfig{
		PollInterval:     100 * time.Millisecond,
		BatchSize:        100,
		MaxRetries:       5,
		RetryBackoffBase: 1 * time.Second,
		RetryBackoffMax:  1 * time.Minute,
	}
}

// Processor polls the outbox and publishes events to the message broker.
type Processor struct {
	repo      Repository
	publisher eventbus.Publisher
	config    ProcessorConfig
	logger    *slog.Logger

	wg       sync.WaitGroup
	stopChan chan struct{}
	running  bool
	mu       sync.Mutex

	statsMu sync.Mutex
	stats   Stats
}

// NewProcessor creates a new outbox processor.
func NewProcessor(repo Repository, publisher eventbus.Publisher, config ProcessorConfig, logger *slog.Logger) *Processor {
	if logger == nil {
		logger = slog.Default()
	}
	return &Processor{
		repo:      repo,
		publisher: publisher,
		config:    config,
		logger:    logger,
		stopChan:  make(chan struct{}),
	}
}

// Start begins the polling loop in a goroutine.
func (p *Processor) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return nil
	}
	p.running = true
	p.stopChan = make(chan struct{})
	p.mu.Unlock()

	p.wg.Add(1)
	go p.run(ctx)

	p.logger.Info("outbox processor started",
		"poll_interval", p.config.PollInterval,
		"batch_size", p.config.BatchSize,
	)

	return nil
}

// Stop gracefully stops the processor.
func (p *Processor) Stop() {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return
	}
	p.running = false
	close(p.stopChan)
	p.mu.Unlock()

	p.wg.Wait()
	p.logger.Info("outbox processor stopped")
}

// IsRunning returns true if the processor is running.
func (p *Processor) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

func (p *Processor) run(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopChan:
			return
		case <-ticker.C:
			if err := p.processBatch(ctx); err != nil {
				p.logger.Error("failed to process outbox batch", "error", err)
			}
		}
	}
}

func (p *Processor) processBatch(ctx context.Context) error {
	messages, err := p.repo.GetUnpublished(ctx, p.config.BatchSize)
	if err != nil {
		p.recordError(err)
		return err
	}

	p.recordProcessed(messages)

	for _, msg := range messages {
		metaFields := p.metadataFields(msg)
		if err := p.publishMessage(ctx, msg); err != nil {
			p.logger.Warn("failed to publish message",
				"id", msg.ID,
				"routing_key", msg.RoutingKey,
				"event_id", msg.EventID,
				"correlation_id", metaFields.CorrelationID,
				"causation_id", metaFields.CausationID,
				"user_id", metaFields.UserID,
				"error", err,
			)
			errStr := err.Error()
			if p.shouldDeadLetter(msg) {
				p.recordDead(err)
				if markErr := p.repo.MarkDead(ctx, msg.ID, errStr); markErr != nil {
					p.logger.Error("failed to mark message as dead-lettered",
						"id", msg.ID,
						"error", markErr,
					)
				}
			} else {
				p.recordFailed(err)
				nextRetryAt := time.Now().Add(p.retryBackoff(msg.RetryCount + 1))
				if markErr := p.repo.MarkFailed(ctx, msg.ID, errStr, nextRetryAt); markErr != nil {
					p.logger.Error("failed to mark message as failed",
						"id", msg.ID,
						"error", markErr,
					)
				}
			}
			continue
		}

		if err := p.repo.MarkPublished(ctx, msg.ID); err != nil {
			p.logger.Error("failed to mark message as published",
				"id", msg.ID,
				"event_id", msg.EventID,
				"error", err,
			)
		} else {
			p.recordPublished()
		}
	}

	return nil
}

func (p *Processor) publishMessage(ctx context.Context, msg *Message) error {
	return p.publisher.Publish(ctx, msg.RoutingKey, msg.Payload)
}

func (p *Processor) shouldDeadLetter(msg *Message) bool {
	if p.config.MaxRetries <= 0 {
		return true
	}
	return msg.RetryCount+1 >= p.config.MaxRetries
}

func (p *Processor) retryBackoff(nextRetryCount int) time.Duration {
	base := p.config.RetryBackoffBase
	if base <= 0 {
		base = time.Second
	}
	max := p.config.RetryBackoffMax
	if max <= 0 {
		max = time.Minute
	}
	if nextRetryCount < 1 {
		nextRetryCount = 1
	}

	backoff := base * time.Duration(1<<convert.IntToUintSafe(nextRetryCount-1))
	if backoff > max {
		return max
	}
	return backoff
}

type metadataFields struct {
	CorrelationID string
	CausationID   string
	UserID        string
}

func (p *Processor) metadataFields(msg *Message) metadataFields {
	if len(msg.Metadata) == 0 {
		return metadataFields{}
	}

	var metadata domain.EventMetadata
	if err := json.Unmarshal(msg.Metadata, &metadata); err != nil {
		return metadataFields{}
	}

	return metadataFields{
		CorrelationID: metadata.CorrelationID.String(),
		CausationID:   metadata.CausationID.String(),
		UserID:        metadata.UserID.String(),
	}
}

// ProcessOnce processes a single batch synchronously (useful for testing).
func (p *Processor) ProcessOnce(ctx context.Context) error {
	return p.processBatch(ctx)
}

// Stats returns processor statistics.
type Stats struct {
	IsRunning       bool
	PublishedCount  uint64
	FailedCount     uint64
	DeadCount       uint64
	LagSeconds      float64
	LastError       string
	LastErrorAt     *time.Time
	LastProcessedAt *time.Time
	OldestMessageAt *time.Time
}

// GetStats returns current processor statistics.
func (p *Processor) GetStats() Stats {
	p.statsMu.Lock()
	defer p.statsMu.Unlock()

	return Stats{
		IsRunning:       p.IsRunning(),
		PublishedCount:  p.stats.PublishedCount,
		FailedCount:     p.stats.FailedCount,
		DeadCount:       p.stats.DeadCount,
		LagSeconds:      p.stats.LagSeconds,
		LastError:       p.stats.LastError,
		LastErrorAt:     p.stats.LastErrorAt,
		LastProcessedAt: p.stats.LastProcessedAt,
		OldestMessageAt: p.stats.OldestMessageAt,
	}
}

func (p *Processor) recordPublished() {
	p.statsMu.Lock()
	defer p.statsMu.Unlock()
	p.stats.PublishedCount++
}

func (p *Processor) recordFailed(err error) {
	p.statsMu.Lock()
	defer p.statsMu.Unlock()
	p.stats.FailedCount++
	now := time.Now()
	p.stats.LastError = err.Error()
	p.stats.LastErrorAt = &now
}

func (p *Processor) recordDead(err error) {
	p.statsMu.Lock()
	defer p.statsMu.Unlock()
	p.stats.DeadCount++
	now := time.Now()
	p.stats.LastError = err.Error()
	p.stats.LastErrorAt = &now
}

func (p *Processor) recordError(err error) {
	p.statsMu.Lock()
	defer p.statsMu.Unlock()
	now := time.Now()
	p.stats.LastError = err.Error()
	p.stats.LastErrorAt = &now
}

func (p *Processor) recordProcessed(messages []*Message) {
	p.statsMu.Lock()
	defer p.statsMu.Unlock()
	now := time.Now()
	p.stats.LastProcessedAt = &now
	if len(messages) == 0 {
		p.stats.LagSeconds = 0
		p.stats.OldestMessageAt = nil
		return
	}

	oldest := messages[0].CreatedAt
	for _, msg := range messages[1:] {
		if msg.CreatedAt.Before(oldest) {
			oldest = msg.CreatedAt
		}
	}
	p.stats.OldestMessageAt = &oldest
	p.stats.LagSeconds = now.Sub(oldest).Seconds()
}
