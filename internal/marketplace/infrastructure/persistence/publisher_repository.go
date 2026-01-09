package persistence

import (
	"context"
	"database/sql"

	"github.com/felixgeelhaar/orbita/internal/marketplace/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresPublisherRepository implements domain.PublisherRepository using PostgreSQL.
type PostgresPublisherRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresPublisherRepository creates a new PostgreSQL publisher repository.
func NewPostgresPublisherRepository(pool *pgxpool.Pool) *PostgresPublisherRepository {
	return &PostgresPublisherRepository{pool: pool}
}

// Create creates a new publisher.
func (r *PostgresPublisherRepository) Create(ctx context.Context, publisher *domain.Publisher) error {
	query := `
		INSERT INTO marketplace_publishers (
			id, name, slug, email, website, description, verified,
			avatar_url, package_count, total_downloads, user_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := r.pool.Exec(ctx, query,
		publisher.ID, publisher.Name, publisher.Slug, publisher.Email,
		publisher.Website, publisher.Description, publisher.Verified,
		publisher.AvatarURL, publisher.PackageCount, publisher.TotalDownloads,
		publisher.UserID, publisher.CreatedAt, publisher.UpdatedAt,
	)
	return err
}

// Update updates an existing publisher.
func (r *PostgresPublisherRepository) Update(ctx context.Context, publisher *domain.Publisher) error {
	query := `
		UPDATE marketplace_publishers SET
			name = $2, email = $3, website = $4, description = $5, verified = $6,
			avatar_url = $7, package_count = $8, total_downloads = $9, updated_at = $10
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query,
		publisher.ID, publisher.Name, publisher.Email, publisher.Website,
		publisher.Description, publisher.Verified, publisher.AvatarURL,
		publisher.PackageCount, publisher.TotalDownloads, publisher.UpdatedAt,
	)
	return err
}

// Delete deletes a publisher by ID.
func (r *PostgresPublisherRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM marketplace_publishers WHERE id = $1", id)
	return err
}

// GetByID retrieves a publisher by ID.
func (r *PostgresPublisherRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Publisher, error) {
	query := `
		SELECT id, name, slug, email, website, description, verified,
			avatar_url, package_count, total_downloads, user_id, created_at, updated_at
		FROM marketplace_publishers WHERE id = $1
	`
	return r.scanPublisher(r.pool.QueryRow(ctx, query, id))
}

// GetBySlug retrieves a publisher by slug.
func (r *PostgresPublisherRepository) GetBySlug(ctx context.Context, slug string) (*domain.Publisher, error) {
	query := `
		SELECT id, name, slug, email, website, description, verified,
			avatar_url, package_count, total_downloads, user_id, created_at, updated_at
		FROM marketplace_publishers WHERE slug = $1
	`
	return r.scanPublisher(r.pool.QueryRow(ctx, query, slug))
}

// GetByUserID retrieves a publisher by user ID.
func (r *PostgresPublisherRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Publisher, error) {
	query := `
		SELECT id, name, slug, email, website, description, verified,
			avatar_url, package_count, total_downloads, user_id, created_at, updated_at
		FROM marketplace_publishers WHERE user_id = $1
	`
	return r.scanPublisher(r.pool.QueryRow(ctx, query, userID))
}

// List retrieves publishers with pagination.
func (r *PostgresPublisherRepository) List(ctx context.Context, offset, limit int) ([]*domain.Publisher, int64, error) {
	// Count total
	var total int64
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM marketplace_publishers").Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, name, slug, email, website, description, verified,
			avatar_url, package_count, total_downloads, user_id, created_at, updated_at
		FROM marketplace_publishers
		ORDER BY package_count DESC, name ASC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	publishers, err := r.scanPublishers(rows)
	if err != nil {
		return nil, 0, err
	}

	return publishers, total, nil
}

// Search searches publishers by name.
func (r *PostgresPublisherRepository) Search(ctx context.Context, query string, offset, limit int) ([]*domain.Publisher, int64, error) {
	searchPattern := "%" + query + "%"

	// Count total
	var total int64
	countQuery := `SELECT COUNT(*) FROM marketplace_publishers WHERE name ILIKE $1 OR slug ILIKE $1`
	if err := r.pool.QueryRow(ctx, countQuery, searchPattern).Scan(&total); err != nil {
		return nil, 0, err
	}

	selectQuery := `
		SELECT id, name, slug, email, website, description, verified,
			avatar_url, package_count, total_downloads, user_id, created_at, updated_at
		FROM marketplace_publishers
		WHERE name ILIKE $1 OR slug ILIKE $1
		ORDER BY package_count DESC, name ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, selectQuery, searchPattern, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	publishers, err := r.scanPublishers(rows)
	if err != nil {
		return nil, 0, err
	}

	return publishers, total, nil
}

func (r *PostgresPublisherRepository) scanPublisher(row pgx.Row) (*domain.Publisher, error) {
	p := &domain.Publisher{}
	var userID sql.NullString

	err := row.Scan(
		&p.ID, &p.Name, &p.Slug, &p.Email, &p.Website, &p.Description,
		&p.Verified, &p.AvatarURL, &p.PackageCount, &p.TotalDownloads,
		&userID, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if userID.Valid {
		id, _ := uuid.Parse(userID.String)
		p.UserID = &id
	}

	return p, nil
}

func (r *PostgresPublisherRepository) scanPublishers(rows pgx.Rows) ([]*domain.Publisher, error) {
	var publishers []*domain.Publisher

	for rows.Next() {
		p := &domain.Publisher{}
		var userID sql.NullString

		err := rows.Scan(
			&p.ID, &p.Name, &p.Slug, &p.Email, &p.Website, &p.Description,
			&p.Verified, &p.AvatarURL, &p.PackageCount, &p.TotalDownloads,
			&userID, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if userID.Valid {
			id, _ := uuid.Parse(userID.String)
			p.UserID = &id
		}

		publishers = append(publishers, p)
	}

	return publishers, rows.Err()
}
