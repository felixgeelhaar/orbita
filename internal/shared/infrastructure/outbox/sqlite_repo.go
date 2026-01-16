package outbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	db "github.com/felixgeelhaar/orbita/db/generated/sqlite"
	sharedPersistence "github.com/felixgeelhaar/orbita/internal/shared/infrastructure/persistence"
	"github.com/google/uuid"
)

// SQLiteRepository implements Repository using SQLite.
type SQLiteRepository struct {
	dbConn *sql.DB
}

// NewSQLiteRepository creates a new SQLite outbox repository.
func NewSQLiteRepository(dbConn *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{dbConn: dbConn}
}

// getQuerier returns the appropriate querier (transaction or connection) based on context.
func (r *SQLiteRepository) getQuerier(ctx context.Context) *db.Queries {
	if info, ok := sharedPersistence.SQLiteTxInfoFromContext(ctx); ok {
		return db.New(info.Tx)
	}
	return db.New(r.dbConn)
}

// Save stores a new outbox message.
func (r *SQLiteRepository) Save(ctx context.Context, msg *Message) error {
	queries := r.getQuerier(ctx)
	result, err := queries.InsertOutboxEvent(ctx, db.InsertOutboxEventParams{
		EventID:       sql.NullString{String: msg.EventID.String(), Valid: true},
		AggregateType: msg.AggregateType,
		AggregateID:   msg.AggregateID.String(),
		EventType:     msg.EventType,
		RoutingKey:    msg.RoutingKey,
		Payload:       string(msg.Payload),
		Metadata:      sql.NullString{String: string(msg.Metadata), Valid: len(msg.Metadata) > 0},
		CreatedAt:     msg.CreatedAt.Format(time.RFC3339),
	})
	if err != nil {
		return err
	}
	msg.ID = result.ID
	return nil
}

// SaveBatch stores multiple outbox messages atomically.
func (r *SQLiteRepository) SaveBatch(ctx context.Context, msgs []*Message) error {
	if len(msgs) == 0 {
		return nil
	}

	// Check if we're already in a transaction (e.g., from UnitOfWork)
	if info, ok := sharedPersistence.SQLiteTxInfoFromContext(ctx); ok {
		// Use existing transaction
		queries := db.New(info.Tx)
		for _, msg := range msgs {
			result, err := queries.InsertOutboxEvent(ctx, db.InsertOutboxEventParams{
				EventID:       sql.NullString{String: msg.EventID.String(), Valid: true},
				AggregateType: msg.AggregateType,
				AggregateID:   msg.AggregateID.String(),
				EventType:     msg.EventType,
				RoutingKey:    msg.RoutingKey,
				Payload:       string(msg.Payload),
				Metadata:      sql.NullString{String: string(msg.Metadata), Valid: len(msg.Metadata) > 0},
				CreatedAt:     msg.CreatedAt.Format(time.RFC3339),
			})
			if err != nil {
				return err
			}
			msg.ID = result.ID
		}
		return nil
	}

	// Start our own transaction
	tx, err := r.dbConn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	queries := db.New(tx)
	for _, msg := range msgs {
		result, err := queries.InsertOutboxEvent(ctx, db.InsertOutboxEventParams{
			EventID:       sql.NullString{String: msg.EventID.String(), Valid: true},
			AggregateType: msg.AggregateType,
			AggregateID:   msg.AggregateID.String(),
			EventType:     msg.EventType,
			RoutingKey:    msg.RoutingKey,
			Payload:       string(msg.Payload),
			Metadata:      sql.NullString{String: string(msg.Metadata), Valid: len(msg.Metadata) > 0},
			CreatedAt:     msg.CreatedAt.Format(time.RFC3339),
		})
		if err != nil {
			return err
		}
		msg.ID = result.ID
	}

	return tx.Commit()
}

// GetUnpublished retrieves unpublished messages ordered by creation time.
func (r *SQLiteRepository) GetUnpublished(ctx context.Context, limit int) ([]*Message, error) {
	queries := r.getQuerier(ctx)
	rows, err := queries.GetUnpublishedEvents(ctx, int64(limit))
	if err != nil {
		return nil, err
	}
	return r.rowsToMessages(rows), nil
}

// MarkPublished marks a message as successfully published.
func (r *SQLiteRepository) MarkPublished(ctx context.Context, id int64) error {
	queries := r.getQuerier(ctx)
	return queries.MarkEventPublished(ctx, id)
}

// MarkFailed records a publish failure with error message.
func (r *SQLiteRepository) MarkFailed(ctx context.Context, id int64, errMsg string, nextRetryAt time.Time) error {
	queries := r.getQuerier(ctx)
	return queries.MarkEventFailed(ctx, db.MarkEventFailedParams{
		ID:          id,
		LastError:   sql.NullString{String: errMsg, Valid: true},
		NextRetryAt: sql.NullString{String: nextRetryAt.Format(time.RFC3339), Valid: true},
	})
}

// MarkDead marks a message as dead-lettered.
func (r *SQLiteRepository) MarkDead(ctx context.Context, id int64, reason string) error {
	queries := r.getQuerier(ctx)
	return queries.MarkEventDead(ctx, db.MarkEventDeadParams{
		ID:               id,
		DeadLetterReason: sql.NullString{String: reason, Valid: true},
	})
}

// GetFailed retrieves failed messages eligible for retry.
func (r *SQLiteRepository) GetFailed(ctx context.Context, maxRetries, limit int) ([]*Message, error) {
	queries := r.getQuerier(ctx)
	rows, err := queries.GetFailedEvents(ctx, db.GetFailedEventsParams{
		RetryCount: int64(maxRetries),
		Limit:      int64(limit),
	})
	if err != nil {
		return nil, err
	}
	return r.rowsToMessages(rows), nil
}

// DeleteOld removes successfully published messages older than the retention period.
func (r *SQLiteRepository) DeleteOld(ctx context.Context, olderThanDays int) (int64, error) {
	queries := r.getQuerier(ctx)
	return queries.DeleteOldPublishedEvents(ctx, sql.NullString{
		String: fmt.Sprintf("%d", olderThanDays),
		Valid:  true,
	})
}

func (r *SQLiteRepository) rowsToMessages(rows []db.Outbox) []*Message {
	messages := make([]*Message, 0, len(rows))
	for _, row := range rows {
		messages = append(messages, r.rowToMessage(row))
	}
	return messages
}

func (r *SQLiteRepository) rowToMessage(row db.Outbox) *Message {
	var eventID uuid.UUID
	if row.EventID.Valid {
		eventID, _ = uuid.Parse(row.EventID.String)
	}

	aggregateID, _ := uuid.Parse(row.AggregateID)
	createdAt, _ := time.Parse(time.RFC3339, row.CreatedAt)

	msg := &Message{
		ID:            row.ID,
		EventID:       eventID,
		AggregateType: row.AggregateType,
		AggregateID:   aggregateID,
		EventType:     row.EventType,
		RoutingKey:    row.RoutingKey,
		Payload:       json.RawMessage(row.Payload),
		CreatedAt:     createdAt,
		RetryCount:    int(row.RetryCount),
	}

	if row.Metadata.Valid {
		msg.Metadata = json.RawMessage(row.Metadata.String)
	}

	if row.PublishedAt.Valid {
		t, _ := time.Parse(time.RFC3339, row.PublishedAt.String)
		msg.PublishedAt = &t
	}

	if row.NextRetryAt.Valid {
		t, _ := time.Parse(time.RFC3339, row.NextRetryAt.String)
		msg.NextRetryAt = &t
	}

	if row.LastError.Valid {
		msg.LastError = &row.LastError.String
	}

	if row.DeadLetteredAt.Valid {
		t, _ := time.Parse(time.RFC3339, row.DeadLetteredAt.String)
		msg.DeadLetteredAt = &t
	}

	if row.DeadLetterReason.Valid {
		msg.DeadLetterReason = &row.DeadLetterReason.String
	}

	return msg
}
