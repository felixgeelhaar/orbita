package persistence

import (
	"context"
	"errors"

	"github.com/felixgeelhaar/orbita/internal/marketplace/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// InstalledPackageRepository is a PostgreSQL implementation of domain.InstalledPackageRepository.
type InstalledPackageRepository struct {
	pool *pgxpool.Pool
}

// NewInstalledPackageRepository creates a new installed package repository.
func NewInstalledPackageRepository(pool *pgxpool.Pool) *InstalledPackageRepository {
	return &InstalledPackageRepository{pool: pool}
}

// Create saves a new installed package.
func (r *InstalledPackageRepository) Create(ctx context.Context, pkg *domain.InstalledPackage) error {
	query := `
		INSERT INTO installed_packages (
			id, package_id, version, type, install_path, checksum,
			installed_at, updated_at, enabled, user_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.pool.Exec(ctx, query,
		pkg.ID,
		pkg.PackageID,
		pkg.Version,
		pkg.Type,
		pkg.InstallPath,
		pkg.Checksum,
		pkg.InstalledAt,
		pkg.UpdatedAt,
		pkg.Enabled,
		pkg.UserID,
	)

	return err
}

// Update updates an existing installed package.
func (r *InstalledPackageRepository) Update(ctx context.Context, pkg *domain.InstalledPackage) error {
	query := `
		UPDATE installed_packages SET
			version = $2,
			install_path = $3,
			checksum = $4,
			updated_at = $5,
			enabled = $6
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query,
		pkg.ID,
		pkg.Version,
		pkg.InstallPath,
		pkg.Checksum,
		pkg.UpdatedAt,
		pkg.Enabled,
	)

	return err
}

// Delete removes an installed package.
func (r *InstalledPackageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM installed_packages WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// GetByID retrieves an installed package by ID.
func (r *InstalledPackageRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.InstalledPackage, error) {
	query := `
		SELECT id, package_id, version, type, install_path, checksum,
		       installed_at, updated_at, enabled, user_id
		FROM installed_packages
		WHERE id = $1
	`

	pkg := &domain.InstalledPackage{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&pkg.ID,
		&pkg.PackageID,
		&pkg.Version,
		&pkg.Type,
		&pkg.InstallPath,
		&pkg.Checksum,
		&pkg.InstalledAt,
		&pkg.UpdatedAt,
		&pkg.Enabled,
		&pkg.UserID,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return pkg, nil
}

// GetByPackageID retrieves an installed package by package ID and user.
func (r *InstalledPackageRepository) GetByPackageID(ctx context.Context, packageID string, userID uuid.UUID) (*domain.InstalledPackage, error) {
	query := `
		SELECT id, package_id, version, type, install_path, checksum,
		       installed_at, updated_at, enabled, user_id
		FROM installed_packages
		WHERE package_id = $1 AND user_id = $2
	`

	pkg := &domain.InstalledPackage{}
	err := r.pool.QueryRow(ctx, query, packageID, userID).Scan(
		&pkg.ID,
		&pkg.PackageID,
		&pkg.Version,
		&pkg.Type,
		&pkg.InstallPath,
		&pkg.Checksum,
		&pkg.InstalledAt,
		&pkg.UpdatedAt,
		&pkg.Enabled,
		&pkg.UserID,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return pkg, nil
}

// ListByUser retrieves all installed packages for a user.
func (r *InstalledPackageRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.InstalledPackage, error) {
	query := `
		SELECT id, package_id, version, type, install_path, checksum,
		       installed_at, updated_at, enabled, user_id
		FROM installed_packages
		WHERE user_id = $1
		ORDER BY installed_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var packages []*domain.InstalledPackage
	for rows.Next() {
		pkg := &domain.InstalledPackage{}
		if err := rows.Scan(
			&pkg.ID,
			&pkg.PackageID,
			&pkg.Version,
			&pkg.Type,
			&pkg.InstallPath,
			&pkg.Checksum,
			&pkg.InstalledAt,
			&pkg.UpdatedAt,
			&pkg.Enabled,
			&pkg.UserID,
		); err != nil {
			return nil, err
		}
		packages = append(packages, pkg)
	}

	return packages, rows.Err()
}

// ListByType retrieves installed packages by type for a user.
func (r *InstalledPackageRepository) ListByType(ctx context.Context, userID uuid.UUID, pkgType domain.PackageType) ([]*domain.InstalledPackage, error) {
	query := `
		SELECT id, package_id, version, type, install_path, checksum,
		       installed_at, updated_at, enabled, user_id
		FROM installed_packages
		WHERE user_id = $1 AND type = $2
		ORDER BY installed_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID, pkgType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var packages []*domain.InstalledPackage
	for rows.Next() {
		pkg := &domain.InstalledPackage{}
		if err := rows.Scan(
			&pkg.ID,
			&pkg.PackageID,
			&pkg.Version,
			&pkg.Type,
			&pkg.InstallPath,
			&pkg.Checksum,
			&pkg.InstalledAt,
			&pkg.UpdatedAt,
			&pkg.Enabled,
			&pkg.UserID,
		); err != nil {
			return nil, err
		}
		packages = append(packages, pkg)
	}

	return packages, rows.Err()
}
