package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// InstalledPackage represents a locally installed marketplace package.
type InstalledPackage struct {
	// ID is the unique identifier for this installation.
	ID uuid.UUID

	// PackageID is the marketplace package identifier (e.g., "acme.priority-v2").
	PackageID string

	// Version is the installed version.
	Version string

	// Type is the package type (orbit or engine).
	Type PackageType

	// InstallPath is the local filesystem path where the package is installed.
	InstallPath string

	// Checksum is the SHA256 checksum of the installed package.
	Checksum string

	// InstalledAt is when the package was installed.
	InstalledAt time.Time

	// UpdatedAt is when the package was last updated.
	UpdatedAt time.Time

	// Enabled indicates if the package is currently enabled.
	Enabled bool

	// UserID is the user who installed the package.
	UserID uuid.UUID
}

// NewInstalledPackage creates a new installed package record.
func NewInstalledPackage(packageID, version string, pkgType PackageType, installPath string, userID uuid.UUID) *InstalledPackage {
	now := time.Now().UTC()
	return &InstalledPackage{
		ID:          uuid.New(),
		PackageID:   packageID,
		Version:     version,
		Type:        pkgType,
		InstallPath: installPath,
		InstalledAt: now,
		UpdatedAt:   now,
		Enabled:     true,
		UserID:      userID,
	}
}

// SetChecksum sets the package checksum.
func (p *InstalledPackage) SetChecksum(checksum string) {
	p.Checksum = checksum
	p.UpdatedAt = time.Now().UTC()
}

// UpdateVersion updates the installed version.
func (p *InstalledPackage) UpdateVersion(version string) {
	p.Version = version
	p.UpdatedAt = time.Now().UTC()
}

// Enable enables the package.
func (p *InstalledPackage) Enable() {
	p.Enabled = true
	p.UpdatedAt = time.Now().UTC()
}

// Disable disables the package.
func (p *InstalledPackage) Disable() {
	p.Enabled = false
	p.UpdatedAt = time.Now().UTC()
}

// InstalledPackageRepository defines the interface for installed package persistence.
type InstalledPackageRepository interface {
	// Create saves a new installed package.
	Create(ctx context.Context, pkg *InstalledPackage) error

	// Update updates an existing installed package.
	Update(ctx context.Context, pkg *InstalledPackage) error

	// Delete removes an installed package.
	Delete(ctx context.Context, id uuid.UUID) error

	// GetByID retrieves an installed package by ID.
	GetByID(ctx context.Context, id uuid.UUID) (*InstalledPackage, error)

	// GetByPackageID retrieves an installed package by package ID and user.
	GetByPackageID(ctx context.Context, packageID string, userID uuid.UUID) (*InstalledPackage, error)

	// ListByUser retrieves all installed packages for a user.
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*InstalledPackage, error)

	// ListByType retrieves installed packages by type for a user.
	ListByType(ctx context.Context, userID uuid.UUID, pkgType PackageType) ([]*InstalledPackage, error)
}
