package outbox

import (
	"context"
	"time"

	sharedPersistence "github.com/felixgeelhaar/orbita/internal/shared/infrastructure/persistence"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL outbox repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// Save stores a new outbox message.
func (r *PostgresRepository) Save(ctx context.Context, msg *Message) error {
	query := `
		INSERT INTO outbox (
			event_id, aggregate_type, aggregate_id, event_type, routing_key,
			payload, metadata, created_at, next_retry_at, dead_lettered_at, dead_letter_reason
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`

	execer := sharedPersistence.Executor(ctx, r.pool)
	return execer.QueryRow(ctx, query,
		msg.EventID,
		msg.AggregateType,
		msg.AggregateID,
		msg.EventType,
		msg.RoutingKey,
		msg.Payload,
		msg.Metadata,
		msg.CreatedAt,
		msg.NextRetryAt,
		msg.DeadLetteredAt,
		msg.DeadLetterReason,
	).Scan(&msg.ID)
}

// SaveBatch stores multiple outbox messages atomically.
func (r *PostgresRepository) SaveBatch(ctx context.Context, msgs []*Message) error {
	if len(msgs) == 0 {
		return nil
	}

	if info, ok := sharedPersistence.TxInfoFromContext(ctx); ok {
		for _, msg := range msgs {
			query := `
				INSERT INTO outbox (
					event_id, aggregate_type, aggregate_id, event_type, routing_key,
					payload, metadata, created_at, next_retry_at, dead_lettered_at, dead_letter_reason
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
				RETURNING id
			`
			err := info.Tx.QueryRow(ctx, query,
				msg.EventID,
				msg.AggregateType,
				msg.AggregateID,
				msg.EventType,
				msg.RoutingKey,
				msg.Payload,
				msg.Metadata,
				msg.CreatedAt,
				msg.NextRetryAt,
				msg.DeadLetteredAt,
				msg.DeadLetterReason,
			).Scan(&msg.ID)
			if err != nil {
				return err
			}
		}
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, msg := range msgs {
		query := `
			INSERT INTO outbox (
				event_id, aggregate_type, aggregate_id, event_type, routing_key,
				payload, metadata, created_at, next_retry_at, dead_lettered_at, dead_letter_reason
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			RETURNING id
		`
		err := tx.QueryRow(ctx, query,
			msg.EventID,
			msg.AggregateType,
			msg.AggregateID,
			msg.EventType,
			msg.RoutingKey,
			msg.Payload,
			msg.Metadata,
			msg.CreatedAt,
			msg.NextRetryAt,
			msg.DeadLetteredAt,
			msg.DeadLetterReason,
		).Scan(&msg.ID)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// GetUnpublished retrieves unpublished messages ordered by creation time.
func (r *PostgresRepository) GetUnpublished(ctx context.Context, limit int) ([]*Message, error) {
	query := `
		SELECT id, event_id, aggregate_type, aggregate_id, event_type, routing_key,
		       payload, metadata, created_at, published_at, next_retry_at, retry_count,
		       last_error, dead_lettered_at, dead_letter_reason
		FROM outbox
		WHERE published_at IS NULL
		  AND dead_lettered_at IS NULL
		  AND (next_retry_at IS NULL OR next_retry_at <= NOW())
		ORDER BY created_at
		LIMIT $1
	`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanMessages(rows)
}

// MarkPublished marks a message as successfully published.
func (r *PostgresRepository) MarkPublished(ctx context.Context, id int64) error {
	query := `UPDATE outbox SET published_at = NOW(), dead_lettered_at = NULL WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// MarkFailed records a publish failure with error message.
func (r *PostgresRepository) MarkFailed(ctx context.Context, id int64, errMsg string, nextRetryAt time.Time) error {
	query := `
		UPDATE outbox
		SET retry_count = retry_count + 1,
			last_error = $2,
			next_retry_at = $3
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, id, errMsg, nextRetryAt)
	return err
}

// MarkDead marks a message as dead-lettered.
func (r *PostgresRepository) MarkDead(ctx context.Context, id int64, reason string) error {
	query := `
		UPDATE outbox
		SET dead_lettered_at = NOW(),
			dead_letter_reason = $2
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, id, reason)
	return err
}

// GetFailed retrieves failed messages eligible for retry.
func (r *PostgresRepository) GetFailed(ctx context.Context, maxRetries, limit int) ([]*Message, error) {
	query := `
		SELECT id, event_id, aggregate_type, aggregate_id, event_type, routing_key,
		       payload, metadata, created_at, published_at, next_retry_at, retry_count,
		       last_error, dead_lettered_at, dead_letter_reason
		FROM outbox
		WHERE published_at IS NULL
		  AND dead_lettered_at IS NULL
		  AND retry_count > 0
		  AND retry_count < $1
		  AND (next_retry_at IS NULL OR next_retry_at <= NOW())
		ORDER BY created_at
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, maxRetries, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanMessages(rows)
}

// DeleteOld removes successfully published messages older than the retention period.
func (r *PostgresRepository) DeleteOld(ctx context.Context, olderThanDays int) (int64, error) {
	query := `
		DELETE FROM outbox
		WHERE published_at IS NOT NULL
		  AND published_at < NOW() - INTERVAL '1 day' * $1
	`
	result, err := r.pool.Exec(ctx, query, olderThanDays)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

func (r *PostgresRepository) scanMessages(rows pgx.Rows) ([]*Message, error) {
	var messages []*Message

	for rows.Next() {
		var msg Message
		err := rows.Scan(
			&msg.ID,
			&msg.EventID,
			&msg.AggregateType,
			&msg.AggregateID,
			&msg.EventType,
			&msg.RoutingKey,
			&msg.Payload,
			&msg.Metadata,
			&msg.CreatedAt,
			&msg.PublishedAt,
			&msg.NextRetryAt,
			&msg.RetryCount,
			&msg.LastError,
			&msg.DeadLetteredAt,
			&msg.DeadLetterReason,
		)
		if err != nil {
			return nil, err
		}
		messages = append(messages, &msg)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

// InMemoryRepository is an in-memory implementation for testing/development.
type InMemoryRepository struct {
	messages []*Message
	nextID   int64
}

// NewInMemoryRepository creates a new in-memory outbox repository.
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		messages: make([]*Message, 0),
		nextID:   1,
	}
}

func (r *InMemoryRepository) Save(ctx context.Context, msg *Message) error {
	msg.ID = r.nextID
	r.nextID++
	msg.CreatedAt = time.Now()
	r.messages = append(r.messages, msg)
	return nil
}

func (r *InMemoryRepository) SaveBatch(ctx context.Context, msgs []*Message) error {
	for _, msg := range msgs {
		if err := r.Save(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

func (r *InMemoryRepository) GetUnpublished(ctx context.Context, limit int) ([]*Message, error) {
	var result []*Message
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

func (r *InMemoryRepository) MarkPublished(ctx context.Context, id int64) error {
	for _, msg := range r.messages {
		if msg.ID == id {
			now := time.Now()
			msg.PublishedAt = &now
			msg.DeadLetteredAt = nil
			return nil
		}
	}
	return nil
}

func (r *InMemoryRepository) MarkFailed(ctx context.Context, id int64, errMsg string, nextRetryAt time.Time) error {
	for _, msg := range r.messages {
		if msg.ID == id {
			msg.RetryCount++
			msg.LastError = &errMsg
			msg.NextRetryAt = &nextRetryAt
			return nil
		}
	}
	return nil
}

func (r *InMemoryRepository) MarkDead(ctx context.Context, id int64, reason string) error {
	for _, msg := range r.messages {
		if msg.ID == id {
			now := time.Now()
			msg.DeadLetteredAt = &now
			msg.DeadLetterReason = &reason
			return nil
		}
	}
	return nil
}

func (r *InMemoryRepository) GetFailed(ctx context.Context, maxRetries, limit int) ([]*Message, error) {
	var result []*Message
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

func (r *InMemoryRepository) DeleteOld(ctx context.Context, olderThanDays int) (int64, error) {
	return 0, nil
}
