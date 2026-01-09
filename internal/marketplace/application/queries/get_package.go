package queries

import (
	"context"
	"errors"

	"github.com/felixgeelhaar/orbita/internal/marketplace/domain"
	"github.com/google/uuid"
)

// ErrPackageNotFound is returned when a package is not found.
var ErrPackageNotFound = errors.New("package not found")

// GetPackageQuery represents a query to get a package by ID or package ID.
type GetPackageQuery struct {
	ID        *uuid.UUID
	PackageID *string
}

// PackageDetailDTO represents detailed package information.
type PackageDetailDTO struct {
	PackageDTO
	Versions  []*VersionDTO  `json:"versions"`
	Publisher *PublisherDTO  `json:"publisher,omitempty"`
}

// VersionDTO represents a version in query results.
type VersionDTO struct {
	ID                 string `json:"id"`
	Version            string `json:"version"`
	MinAPIVersion      string `json:"min_api_version,omitempty"`
	Changelog          string `json:"changelog,omitempty"`
	Checksum           string `json:"checksum,omitempty"`
	DownloadURL        string `json:"download_url,omitempty"`
	Size               int64  `json:"size"`
	Downloads          int64  `json:"downloads"`
	Prerelease         bool   `json:"prerelease"`
	Deprecated         bool   `json:"deprecated"`
	DeprecationMessage string `json:"deprecation_message,omitempty"`
	PublishedAt        string `json:"published_at"`
}

// PublisherDTO represents a publisher in query results.
type PublisherDTO struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	Website        string `json:"website,omitempty"`
	Description    string `json:"description,omitempty"`
	Verified       bool   `json:"verified"`
	AvatarURL      string `json:"avatar_url,omitempty"`
	PackageCount   int    `json:"package_count"`
	TotalDownloads int64  `json:"total_downloads"`
}

// GetPackageHandler handles getting a single package.
type GetPackageHandler struct {
	packageRepo   domain.PackageRepository
	versionRepo   domain.VersionRepository
	publisherRepo domain.PublisherRepository
}

// NewGetPackageHandler creates a new get package handler.
func NewGetPackageHandler(
	packageRepo domain.PackageRepository,
	versionRepo domain.VersionRepository,
	publisherRepo domain.PublisherRepository,
) *GetPackageHandler {
	return &GetPackageHandler{
		packageRepo:   packageRepo,
		versionRepo:   versionRepo,
		publisherRepo: publisherRepo,
	}
}

// Handle executes the get package query.
func (h *GetPackageHandler) Handle(ctx context.Context, query GetPackageQuery) (*PackageDetailDTO, error) {
	var pkg *domain.Package
	var err error

	if query.ID != nil {
		pkg, err = h.packageRepo.GetByID(ctx, *query.ID)
	} else if query.PackageID != nil {
		pkg, err = h.packageRepo.GetByPackageID(ctx, *query.PackageID)
	} else {
		return nil, errors.New("either ID or PackageID must be provided")
	}

	if err != nil {
		return nil, err
	}
	if pkg == nil {
		return nil, ErrPackageNotFound
	}

	// Get versions
	versions, err := h.versionRepo.ListByPackage(ctx, pkg.ID)
	if err != nil {
		return nil, err
	}

	versionDTOs := make([]*VersionDTO, len(versions))
	for i, v := range versions {
		versionDTOs[i] = versionToDTO(v)
	}

	result := &PackageDetailDTO{
		PackageDTO: *packageToDTO(pkg),
		Versions:   versionDTOs,
	}

	// Get publisher if available
	if pkg.PublisherID != uuid.Nil {
		publisher, err := h.publisherRepo.GetByID(ctx, pkg.PublisherID)
		if err == nil && publisher != nil {
			result.Publisher = publisherToDTO(publisher)
		}
	}

	return result, nil
}

func versionToDTO(v *domain.Version) *VersionDTO {
	return &VersionDTO{
		ID:                 v.ID.String(),
		Version:            v.Version,
		MinAPIVersion:      v.MinAPIVersion,
		Changelog:          v.Changelog,
		Checksum:           v.Checksum,
		DownloadURL:        v.DownloadURL,
		Size:               v.Size,
		Downloads:          v.Downloads,
		Prerelease:         v.Prerelease,
		Deprecated:         v.Deprecated,
		DeprecationMessage: v.DeprecationMessage,
		PublishedAt:        v.PublishedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func publisherToDTO(p *domain.Publisher) *PublisherDTO {
	return &PublisherDTO{
		ID:             p.ID.String(),
		Name:           p.Name,
		Slug:           p.Slug,
		Website:        p.Website,
		Description:    p.Description,
		Verified:       p.Verified,
		AvatarURL:      p.AvatarURL,
		PackageCount:   p.PackageCount,
		TotalDownloads: p.TotalDownloads,
	}
}
