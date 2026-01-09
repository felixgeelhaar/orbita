package persistence

import (
	"context"

	"github.com/felixgeelhaar/orbita/internal/marketplace/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresVersionRepository implements domain.VersionRepository using PostgreSQL.
type PostgresVersionRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresVersionRepository creates a new PostgreSQL version repository.
func NewPostgresVersionRepository(pool *pgxpool.Pool) *PostgresVersionRepository {
	return &PostgresVersionRepository{pool: pool}
}

// Create creates a new version.
func (r *PostgresVersionRepository) Create(ctx context.Context, version *domain.Version) error {
	query := `
		INSERT INTO marketplace_versions (
			id, package_id, version, min_api_version, changelog, checksum,
			download_url, size, downloads, prerelease, deprecated,
			deprecation_message, published_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err := r.pool.Exec(ctx, query,
		version.ID, version.PackageID, version.Version, version.MinAPIVersion,
		version.Changelog, version.Checksum, version.DownloadURL, version.Size,
		version.Downloads, version.Prerelease, version.Deprecated,
		version.DeprecationMessage, version.PublishedAt, version.CreatedAt,
	)
	return err
}

// Update updates an existing version.
func (r *PostgresVersionRepository) Update(ctx context.Context, version *domain.Version) error {
	query := `
		UPDATE marketplace_versions SET
			changelog = $2, checksum = $3, download_url = $4, size = $5,
			downloads = $6, prerelease = $7, deprecated = $8, deprecation_message = $9
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query,
		version.ID, version.Changelog, version.Checksum, version.DownloadURL,
		version.Size, version.Downloads, version.Prerelease, version.Deprecated,
		version.DeprecationMessage,
	)
	return err
}

// Delete deletes a version by ID.
func (r *PostgresVersionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM marketplace_versions WHERE id = $1", id)
	return err
}

// GetByID retrieves a version by ID.
func (r *PostgresVersionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Version, error) {
	query := `
		SELECT id, package_id, version, min_api_version, changelog, checksum,
			download_url, size, downloads, prerelease, deprecated,
			deprecation_message, published_at, created_at
		FROM marketplace_versions WHERE id = $1
	`
	return r.scanVersion(r.pool.QueryRow(ctx, query, id))
}

// GetByPackageAndVersion retrieves a specific version of a package.
func (r *PostgresVersionRepository) GetByPackageAndVersion(ctx context.Context, packageID uuid.UUID, version string) (*domain.Version, error) {
	query := `
		SELECT id, package_id, version, min_api_version, changelog, checksum,
			download_url, size, downloads, prerelease, deprecated,
			deprecation_message, published_at, created_at
		FROM marketplace_versions
		WHERE package_id = $1 AND version = $2
	`
	return r.scanVersion(r.pool.QueryRow(ctx, query, packageID, version))
}

// ListByPackage retrieves all versions of a package.
func (r *PostgresVersionRepository) ListByPackage(ctx context.Context, packageID uuid.UUID) ([]*domain.Version, error) {
	query := `
		SELECT id, package_id, version, min_api_version, changelog, checksum,
			download_url, size, downloads, prerelease, deprecated,
			deprecation_message, published_at, created_at
		FROM marketplace_versions
		WHERE package_id = $1
		ORDER BY published_at DESC
	`

	rows, err := r.pool.Query(ctx, query, packageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanVersions(rows)
}

// GetLatestStable retrieves the latest stable version of a package.
func (r *PostgresVersionRepository) GetLatestStable(ctx context.Context, packageID uuid.UUID) (*domain.Version, error) {
	query := `
		SELECT id, package_id, version, min_api_version, changelog, checksum,
			download_url, size, downloads, prerelease, deprecated,
			deprecation_message, published_at, created_at
		FROM marketplace_versions
		WHERE package_id = $1 AND prerelease = false AND deprecated = false
		ORDER BY published_at DESC
		LIMIT 1
	`
	return r.scanVersion(r.pool.QueryRow(ctx, query, packageID))
}

// IncrementDownloads increments the download count for a version.
func (r *PostgresVersionRepository) IncrementDownloads(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE marketplace_versions SET downloads = downloads + 1 WHERE id = $1",
		id,
	)
	return err
}

func (r *PostgresVersionRepository) scanVersion(row pgx.Row) (*domain.Version, error) {
	v := &domain.Version{}

	err := row.Scan(
		&v.ID, &v.PackageID, &v.Version, &v.MinAPIVersion, &v.Changelog,
		&v.Checksum, &v.DownloadURL, &v.Size, &v.Downloads, &v.Prerelease,
		&v.Deprecated, &v.DeprecationMessage, &v.PublishedAt, &v.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return v, nil
}

func (r *PostgresVersionRepository) scanVersions(rows pgx.Rows) ([]*domain.Version, error) {
	var versions []*domain.Version

	for rows.Next() {
		v := &domain.Version{}

		err := rows.Scan(
			&v.ID, &v.PackageID, &v.Version, &v.MinAPIVersion, &v.Changelog,
			&v.Checksum, &v.DownloadURL, &v.Size, &v.Downloads, &v.Prerelease,
			&v.Deprecated, &v.DeprecationMessage, &v.PublishedAt, &v.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		versions = append(versions, v)
	}

	return versions, rows.Err()
}
