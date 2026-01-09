// Package persistence provides PostgreSQL implementations for marketplace repositories.
package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/felixgeelhaar/orbita/internal/marketplace/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"
)

// PostgresPackageRepository implements domain.PackageRepository using PostgreSQL.
type PostgresPackageRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresPackageRepository creates a new PostgreSQL package repository.
func NewPostgresPackageRepository(pool *pgxpool.Pool) *PostgresPackageRepository {
	return &PostgresPackageRepository{pool: pool}
}

// Create creates a new package.
func (r *PostgresPackageRepository) Create(ctx context.Context, pkg *domain.Package) error {
	query := `
		INSERT INTO marketplace_packages (
			id, package_id, type, name, description, author, homepage, license,
			tags, latest_version, downloads, rating, rating_count, verified,
			featured, publisher_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`

	var publisherID *uuid.UUID
	if pkg.PublisherID != uuid.Nil {
		publisherID = &pkg.PublisherID
	}

	_, err := r.pool.Exec(ctx, query,
		pkg.ID, pkg.PackageID, pkg.Type, pkg.Name, pkg.Description, pkg.Author,
		pkg.Homepage, pkg.License, pq.Array(pkg.Tags), pkg.LatestVersion, pkg.Downloads,
		pkg.Rating, pkg.RatingCount, pkg.Verified, pkg.Featured, publisherID,
		pkg.CreatedAt, pkg.UpdatedAt,
	)
	return err
}

// Update updates an existing package.
func (r *PostgresPackageRepository) Update(ctx context.Context, pkg *domain.Package) error {
	query := `
		UPDATE marketplace_packages SET
			name = $2, description = $3, author = $4, homepage = $5, license = $6,
			tags = $7, latest_version = $8, downloads = $9, rating = $10,
			rating_count = $11, verified = $12, featured = $13, updated_at = $14
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query,
		pkg.ID, pkg.Name, pkg.Description, pkg.Author, pkg.Homepage, pkg.License,
		pq.Array(pkg.Tags), pkg.LatestVersion, pkg.Downloads, pkg.Rating,
		pkg.RatingCount, pkg.Verified, pkg.Featured, pkg.UpdatedAt,
	)
	return err
}

// Delete deletes a package by ID.
func (r *PostgresPackageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM marketplace_packages WHERE id = $1", id)
	return err
}

// GetByID retrieves a package by ID.
func (r *PostgresPackageRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Package, error) {
	query := `
		SELECT id, package_id, type, name, description, author, homepage, license,
			tags, latest_version, downloads, rating, rating_count, verified,
			featured, publisher_id, created_at, updated_at
		FROM marketplace_packages WHERE id = $1
	`
	return r.scanPackage(r.pool.QueryRow(ctx, query, id))
}

// GetByPackageID retrieves a package by its package ID.
func (r *PostgresPackageRepository) GetByPackageID(ctx context.Context, packageID string) (*domain.Package, error) {
	query := `
		SELECT id, package_id, type, name, description, author, homepage, license,
			tags, latest_version, downloads, rating, rating_count, verified,
			featured, publisher_id, created_at, updated_at
		FROM marketplace_packages WHERE package_id = $1
	`
	return r.scanPackage(r.pool.QueryRow(ctx, query, packageID))
}

// List retrieves packages with filtering and pagination.
func (r *PostgresPackageRepository) List(ctx context.Context, filter domain.PackageFilter) ([]*domain.Package, int64, error) {
	baseQuery := `FROM marketplace_packages WHERE 1=1`
	args := []any{}
	argIndex := 1

	// Build filter conditions
	conditions, args, argIndex := r.buildFilterConditions(filter, args, argIndex)
	baseQuery += conditions

	// Count total
	var total int64
	countQuery := "SELECT COUNT(*) " + baseQuery
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Build order clause
	orderClause := r.buildOrderClause(filter)

	// Query with pagination
	selectQuery := fmt.Sprintf(`
		SELECT id, package_id, type, name, description, author, homepage, license,
			tags, latest_version, downloads, rating, rating_count, verified,
			featured, publisher_id, created_at, updated_at
		%s %s LIMIT $%d OFFSET $%d
	`, baseQuery, orderClause, argIndex, argIndex+1)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	packages, err := r.scanPackages(rows)
	if err != nil {
		return nil, 0, err
	}

	return packages, total, nil
}

// Search searches packages by query string.
func (r *PostgresPackageRepository) Search(ctx context.Context, query string, filter domain.PackageFilter) ([]*domain.Package, int64, error) {
	baseQuery := `FROM marketplace_packages WHERE (
		name ILIKE $1 OR
		description ILIKE $1 OR
		package_id ILIKE $1 OR
		author ILIKE $1
	)`
	searchPattern := "%" + query + "%"
	args := []any{searchPattern}
	argIndex := 2

	// Build filter conditions
	conditions, args, argIndex := r.buildFilterConditions(filter, args, argIndex)
	baseQuery += conditions

	// Count total
	var total int64
	countQuery := "SELECT COUNT(*) " + baseQuery
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Build order clause
	orderClause := r.buildOrderClause(filter)

	// Query with pagination
	selectQuery := fmt.Sprintf(`
		SELECT id, package_id, type, name, description, author, homepage, license,
			tags, latest_version, downloads, rating, rating_count, verified,
			featured, publisher_id, created_at, updated_at
		%s %s LIMIT $%d OFFSET $%d
	`, baseQuery, orderClause, argIndex, argIndex+1)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	packages, err := r.scanPackages(rows)
	if err != nil {
		return nil, 0, err
	}

	return packages, total, nil
}

// GetFeatured retrieves featured packages.
func (r *PostgresPackageRepository) GetFeatured(ctx context.Context, limit int) ([]*domain.Package, error) {
	query := `
		SELECT id, package_id, type, name, description, author, homepage, license,
			tags, latest_version, downloads, rating, rating_count, verified,
			featured, publisher_id, created_at, updated_at
		FROM marketplace_packages
		WHERE featured = true
		ORDER BY downloads DESC
		LIMIT $1
	`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanPackages(rows)
}

// GetByPublisher retrieves packages by publisher ID.
func (r *PostgresPackageRepository) GetByPublisher(ctx context.Context, publisherID uuid.UUID, filter domain.PackageFilter) ([]*domain.Package, int64, error) {
	baseQuery := `FROM marketplace_packages WHERE publisher_id = $1`
	args := []any{publisherID}
	argIndex := 2

	// Build filter conditions
	conditions, args, argIndex := r.buildFilterConditions(filter, args, argIndex)
	baseQuery += conditions

	// Count total
	var total int64
	countQuery := "SELECT COUNT(*) " + baseQuery
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Build order clause
	orderClause := r.buildOrderClause(filter)

	// Query with pagination
	selectQuery := fmt.Sprintf(`
		SELECT id, package_id, type, name, description, author, homepage, license,
			tags, latest_version, downloads, rating, rating_count, verified,
			featured, publisher_id, created_at, updated_at
		%s %s LIMIT $%d OFFSET $%d
	`, baseQuery, orderClause, argIndex, argIndex+1)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	packages, err := r.scanPackages(rows)
	if err != nil {
		return nil, 0, err
	}

	return packages, total, nil
}

// IncrementDownloads increments the download count for a package.
func (r *PostgresPackageRepository) IncrementDownloads(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE marketplace_packages SET downloads = downloads + 1 WHERE id = $1",
		id,
	)
	return err
}

// Helper methods

func (r *PostgresPackageRepository) buildFilterConditions(filter domain.PackageFilter, args []any, argIndex int) (string, []any, int) {
	var conditions strings.Builder

	if filter.Type != nil {
		conditions.WriteString(fmt.Sprintf(" AND type = $%d", argIndex))
		args = append(args, *filter.Type)
		argIndex++
	}

	if len(filter.Tags) > 0 {
		conditions.WriteString(fmt.Sprintf(" AND tags && $%d", argIndex))
		args = append(args, pq.Array(filter.Tags))
		argIndex++
	}

	if filter.Verified != nil {
		conditions.WriteString(fmt.Sprintf(" AND verified = $%d", argIndex))
		args = append(args, *filter.Verified)
		argIndex++
	}

	if filter.Featured != nil {
		conditions.WriteString(fmt.Sprintf(" AND featured = $%d", argIndex))
		args = append(args, *filter.Featured)
		argIndex++
	}

	return conditions.String(), args, argIndex
}

func (r *PostgresPackageRepository) buildOrderClause(filter domain.PackageFilter) string {
	sortField := "downloads"
	switch filter.SortBy {
	case domain.SortByCreatedAt:
		sortField = "created_at"
	case domain.SortByUpdatedAt:
		sortField = "updated_at"
	case domain.SortByDownloads:
		sortField = "downloads"
	case domain.SortByRating:
		sortField = "rating"
	case domain.SortByName:
		sortField = "name"
	}

	order := "DESC"
	if filter.SortOrder == domain.SortAsc {
		order = "ASC"
	}

	return fmt.Sprintf("ORDER BY %s %s", sortField, order)
}

func (r *PostgresPackageRepository) scanPackage(row pgx.Row) (*domain.Package, error) {
	pkg := &domain.Package{}
	var publisherID sql.NullString
	var tags []string

	err := row.Scan(
		&pkg.ID, &pkg.PackageID, &pkg.Type, &pkg.Name, &pkg.Description,
		&pkg.Author, &pkg.Homepage, &pkg.License, pq.Array(&tags),
		&pkg.LatestVersion, &pkg.Downloads, &pkg.Rating, &pkg.RatingCount,
		&pkg.Verified, &pkg.Featured, &publisherID, &pkg.CreatedAt, &pkg.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	pkg.Tags = tags
	if publisherID.Valid {
		pkg.PublisherID, _ = uuid.Parse(publisherID.String)
	}

	return pkg, nil
}

func (r *PostgresPackageRepository) scanPackages(rows pgx.Rows) ([]*domain.Package, error) {
	var packages []*domain.Package

	for rows.Next() {
		pkg := &domain.Package{}
		var publisherID sql.NullString
		var tags []string

		err := rows.Scan(
			&pkg.ID, &pkg.PackageID, &pkg.Type, &pkg.Name, &pkg.Description,
			&pkg.Author, &pkg.Homepage, &pkg.License, pq.Array(&tags),
			&pkg.LatestVersion, &pkg.Downloads, &pkg.Rating, &pkg.RatingCount,
			&pkg.Verified, &pkg.Featured, &publisherID, &pkg.CreatedAt, &pkg.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		pkg.Tags = tags
		if publisherID.Valid {
			pkg.PublisherID, _ = uuid.Parse(publisherID.String)
		}

		packages = append(packages, pkg)
	}

	return packages, rows.Err()
}
