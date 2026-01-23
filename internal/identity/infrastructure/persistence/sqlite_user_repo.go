package persistence

import (
	"context"
	"database/sql"
	"errors"
	"time"

	db "github.com/felixgeelhaar/orbita/db/generated/sqlite"
	"github.com/felixgeelhaar/orbita/internal/identity/domain"
	sharedDomain "github.com/felixgeelhaar/orbita/internal/shared/domain"
	sharedPersistence "github.com/felixgeelhaar/orbita/internal/shared/infrastructure/persistence"
	"github.com/google/uuid"
)

// ErrUserNotFound is returned when a user is not found.
var ErrUserNotFound = errors.New("user not found")

// SQLiteUserRepository handles persistence for users using SQLite.
type SQLiteUserRepository struct {
	dbConn *sql.DB
}

// NewSQLiteUserRepository creates a new SQLiteUserRepository.
func NewSQLiteUserRepository(dbConn *sql.DB) *SQLiteUserRepository {
	return &SQLiteUserRepository{dbConn: dbConn}
}

// getQuerier returns the appropriate querier (transaction or connection) based on context.
func (r *SQLiteUserRepository) getQuerier(ctx context.Context) *db.Queries {
	if info, ok := sharedPersistence.SQLiteTxInfoFromContext(ctx); ok {
		return db.New(info.Tx)
	}
	return db.New(r.dbConn)
}

// Save persists a user to the database.
func (r *SQLiteUserRepository) Save(ctx context.Context, user *domain.User) error {
	queries := r.getQuerier(ctx)

	// Check if user exists
	existing, err := queries.GetUserByID(ctx, user.ID().String())
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	if errors.Is(err, sql.ErrNoRows) {
		// Create new user
		_, err = queries.CreateUser(ctx, db.CreateUserParams{
			ID:        user.ID().String(),
			Email:     user.Email().String(),
			Name:      user.Name().String(),
			CreatedAt: user.CreatedAt().Format(time.RFC3339),
			UpdatedAt: user.UpdatedAt().Format(time.RFC3339),
		})
		return err
	}

	// Update existing user (only if name changed)
	if existing.Name != user.Name().String() {
		_, err = queries.UpdateUser(ctx, db.UpdateUserParams{
			ID:   user.ID().String(),
			Name: user.Name().String(),
		})
		return err
	}

	return nil
}

// FindByID retrieves a user by their ID.
func (r *SQLiteUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	queries := r.getQuerier(ctx)

	row, err := queries.GetUserByID(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return r.toDomain(row)
}

// FindByEmail retrieves a user by their email address.
func (r *SQLiteUserRepository) FindByEmail(ctx context.Context, email domain.Email) (*domain.User, error) {
	queries := r.getQuerier(ctx)

	row, err := queries.GetUserByEmail(ctx, email.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return r.toDomain(row)
}

// Delete removes a user from the database.
func (r *SQLiteUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	queries := r.getQuerier(ctx)
	return queries.DeleteUser(ctx, id.String())
}

// ExistsByEmail checks if a user with the given email exists.
func (r *SQLiteUserRepository) ExistsByEmail(ctx context.Context, email domain.Email) (bool, error) {
	queries := r.getQuerier(ctx)
	count, err := queries.CountByEmail(ctx, email.String())
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// toDomain converts a database row to a domain User.
func (r *SQLiteUserRepository) toDomain(row db.User) (*domain.User, error) {
	id, err := uuid.Parse(row.ID)
	if err != nil {
		return nil, err
	}

	email, err := domain.NewEmail(row.Email)
	if err != nil {
		return nil, err
	}

	name, err := domain.NewName(row.Name)
	if err != nil {
		return nil, err
	}

	createdAt, err := time.Parse(time.RFC3339, row.CreatedAt)
	if err != nil {
		return nil, err
	}

	updatedAt, err := time.Parse(time.RFC3339, row.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Reconstruct the aggregate root with proper timestamps
	baseEntity := sharedDomain.RehydrateBaseEntity(id, createdAt, updatedAt)
	baseAggregate := sharedDomain.RehydrateBaseAggregateRoot(baseEntity, 0)

	return domain.RehydrateUser(baseAggregate, email, name), nil
}
