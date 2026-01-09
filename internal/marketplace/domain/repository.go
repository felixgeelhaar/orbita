package domain

import (
	"context"

	"github.com/google/uuid"
)

// PackageRepository defines the interface for package persistence.
type PackageRepository interface {
	// Create creates a new package.
	Create(ctx context.Context, pkg *Package) error

	// Update updates an existing package.
	Update(ctx context.Context, pkg *Package) error

	// Delete deletes a package by ID.
	Delete(ctx context.Context, id uuid.UUID) error

	// GetByID retrieves a package by ID.
	GetByID(ctx context.Context, id uuid.UUID) (*Package, error)

	// GetByPackageID retrieves a package by its package ID (e.g., "acme.orbit").
	GetByPackageID(ctx context.Context, packageID string) (*Package, error)

	// List retrieves packages with filtering and pagination.
	List(ctx context.Context, filter PackageFilter) ([]*Package, int64, error)

	// Search searches packages by query string.
	Search(ctx context.Context, query string, filter PackageFilter) ([]*Package, int64, error)

	// GetFeatured retrieves featured packages.
	GetFeatured(ctx context.Context, limit int) ([]*Package, error)

	// GetByPublisher retrieves packages by publisher ID.
	GetByPublisher(ctx context.Context, publisherID uuid.UUID, filter PackageFilter) ([]*Package, int64, error)

	// IncrementDownloads increments the download count for a package.
	IncrementDownloads(ctx context.Context, id uuid.UUID) error
}

// PackageFilter defines filtering options for package queries.
type PackageFilter struct {
	// Type filters by package type.
	Type *PackageType

	// Tags filters by tags (any match).
	Tags []string

	// Verified filters for verified packages only.
	Verified *bool

	// Featured filters for featured packages only.
	Featured *bool

	// SortBy specifies the sort field.
	SortBy PackageSortField

	// SortOrder specifies the sort direction.
	SortOrder SortOrder

	// Offset is the pagination offset.
	Offset int

	// Limit is the maximum number of results.
	Limit int
}

// PackageSortField defines fields that can be sorted.
type PackageSortField string

const (
	SortByCreatedAt  PackageSortField = "created_at"
	SortByUpdatedAt  PackageSortField = "updated_at"
	SortByDownloads  PackageSortField = "downloads"
	SortByRating     PackageSortField = "rating"
	SortByName       PackageSortField = "name"
)

// SortOrder defines sort direction.
type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

// DefaultPackageFilter returns a filter with default values.
func DefaultPackageFilter() PackageFilter {
	return PackageFilter{
		SortBy:    SortByDownloads,
		SortOrder: SortDesc,
		Offset:    0,
		Limit:     20,
	}
}

// VersionRepository defines the interface for version persistence.
type VersionRepository interface {
	// Create creates a new version.
	Create(ctx context.Context, version *Version) error

	// Update updates an existing version.
	Update(ctx context.Context, version *Version) error

	// Delete deletes a version by ID.
	Delete(ctx context.Context, id uuid.UUID) error

	// GetByID retrieves a version by ID.
	GetByID(ctx context.Context, id uuid.UUID) (*Version, error)

	// GetByPackageAndVersion retrieves a specific version of a package.
	GetByPackageAndVersion(ctx context.Context, packageID uuid.UUID, version string) (*Version, error)

	// ListByPackage retrieves all versions of a package.
	ListByPackage(ctx context.Context, packageID uuid.UUID) ([]*Version, error)

	// GetLatestStable retrieves the latest stable version of a package.
	GetLatestStable(ctx context.Context, packageID uuid.UUID) (*Version, error)

	// IncrementDownloads increments the download count for a version.
	IncrementDownloads(ctx context.Context, id uuid.UUID) error
}

// PublisherRepository defines the interface for publisher persistence.
type PublisherRepository interface {
	// Create creates a new publisher.
	Create(ctx context.Context, publisher *Publisher) error

	// Update updates an existing publisher.
	Update(ctx context.Context, publisher *Publisher) error

	// Delete deletes a publisher by ID.
	Delete(ctx context.Context, id uuid.UUID) error

	// GetByID retrieves a publisher by ID.
	GetByID(ctx context.Context, id uuid.UUID) (*Publisher, error)

	// GetBySlug retrieves a publisher by slug.
	GetBySlug(ctx context.Context, slug string) (*Publisher, error)

	// GetByUserID retrieves a publisher by user ID.
	GetByUserID(ctx context.Context, userID uuid.UUID) (*Publisher, error)

	// List retrieves publishers with pagination.
	List(ctx context.Context, offset, limit int) ([]*Publisher, int64, error)

	// Search searches publishers by name.
	Search(ctx context.Context, query string, offset, limit int) ([]*Publisher, int64, error)
}
