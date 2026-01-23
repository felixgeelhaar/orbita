package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/felixgeelhaar/orbita/internal/identity/domain"
	sharedDomain "github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresUserRepository handles persistence for users using PostgreSQL.
type PostgresUserRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresUserRepository creates a new PostgresUserRepository.
func NewPostgresUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{pool: pool}
}

// Save persists a user to the database.
func (r *PostgresUserRepository) Save(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, email, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			updated_at = NOW()
	`

	_, err := r.pool.Exec(ctx, query,
		user.ID(),
		user.Email().String(),
		user.Name().String(),
		user.CreatedAt(),
		user.UpdatedAt(),
	)
	return err
}

// FindByID retrieves a user by their ID.
func (r *PostgresUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, email, name, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var (
		userID                 uuid.UUID
		email, name            string
		createdAt, updatedAt   time.Time
	)

	err := r.pool.QueryRow(ctx, query, id).Scan(&userID, &email, &name, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return r.toDomain(userID, email, name, createdAt, updatedAt)
}

// FindByEmail retrieves a user by their email address.
func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email domain.Email) (*domain.User, error) {
	query := `
		SELECT id, email, name, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var (
		userID                 uuid.UUID
		emailStr, name         string
		createdAt, updatedAt   time.Time
	)

	err := r.pool.QueryRow(ctx, query, email.String()).Scan(&userID, &emailStr, &name, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return r.toDomain(userID, emailStr, name, createdAt, updatedAt)
}

// Delete removes a user from the database.
func (r *PostgresUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// ExistsByEmail checks if a user with the given email exists.
func (r *PostgresUserRepository) ExistsByEmail(ctx context.Context, email domain.Email) (bool, error) {
	query := `SELECT COUNT(*) FROM users WHERE email = $1`

	var count int64
	err := r.pool.QueryRow(ctx, query, email.String()).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// toDomain converts database values to a domain User.
func (r *PostgresUserRepository) toDomain(id uuid.UUID, emailStr, nameStr string, createdAt, updatedAt time.Time) (*domain.User, error) {
	email, err := domain.NewEmail(emailStr)
	if err != nil {
		return nil, err
	}

	name, err := domain.NewName(nameStr)
	if err != nil {
		return nil, err
	}

	// Reconstruct the aggregate root with proper timestamps
	baseEntity := sharedDomain.RehydrateBaseEntity(id, createdAt, updatedAt)
	baseAggregate := sharedDomain.RehydrateBaseAggregateRoot(baseEntity, 0)

	return domain.RehydrateUser(baseAggregate, email, name), nil
}
