package queries

import (
	"context"

	"github.com/felixgeelhaar/orbita/internal/marketplace/domain"
	"github.com/google/uuid"
)

// ListInstalledQuery represents a query to list installed packages.
type ListInstalledQuery struct {
	UserID uuid.UUID
	Type   *domain.PackageType // Optional filter by type
}

// ListInstalledResult represents the result of listing installed packages.
type ListInstalledResult struct {
	Packages []*InstalledPackageDTO
	Total    int
}

// InstalledPackageDTO represents an installed package in query results.
type InstalledPackageDTO struct {
	ID          string `json:"id"`
	PackageID   string `json:"package_id"`
	Version     string `json:"version"`
	Type        string `json:"type"`
	InstallPath string `json:"install_path"`
	Checksum    string `json:"checksum,omitempty"`
	InstalledAt string `json:"installed_at"`
	UpdatedAt   string `json:"updated_at"`
	Enabled     bool   `json:"enabled"`
}

// ListInstalledHandler handles listing installed packages.
type ListInstalledHandler struct {
	installedRepo domain.InstalledPackageRepository
}

// NewListInstalledHandler creates a new list installed handler.
func NewListInstalledHandler(installedRepo domain.InstalledPackageRepository) *ListInstalledHandler {
	return &ListInstalledHandler{
		installedRepo: installedRepo,
	}
}

// Handle executes the list installed query.
func (h *ListInstalledHandler) Handle(ctx context.Context, query ListInstalledQuery) (*ListInstalledResult, error) {
	var packages []*domain.InstalledPackage
	var err error

	if query.Type != nil {
		packages, err = h.installedRepo.ListByType(ctx, query.UserID, *query.Type)
	} else {
		packages, err = h.installedRepo.ListByUser(ctx, query.UserID)
	}

	if err != nil {
		return nil, err
	}

	dtos := make([]*InstalledPackageDTO, len(packages))
	for i, pkg := range packages {
		dtos[i] = installedPackageToDTO(pkg)
	}

	return &ListInstalledResult{
		Packages: dtos,
		Total:    len(dtos),
	}, nil
}

func installedPackageToDTO(pkg *domain.InstalledPackage) *InstalledPackageDTO {
	return &InstalledPackageDTO{
		ID:          pkg.ID.String(),
		PackageID:   pkg.PackageID,
		Version:     pkg.Version,
		Type:        string(pkg.Type),
		InstallPath: pkg.InstallPath,
		Checksum:    pkg.Checksum,
		InstalledAt: pkg.InstalledAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   pkg.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		Enabled:     pkg.Enabled,
	}
}
