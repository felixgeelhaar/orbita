// Package queries provides query handlers for the marketplace.
package queries

import (
	"context"

	"github.com/felixgeelhaar/orbita/internal/marketplace/domain"
)

// ListPackagesQuery represents a query to list marketplace packages.
type ListPackagesQuery struct {
	Type     *domain.PackageType
	Tags     []string
	Verified *bool
	Featured *bool
	SortBy   domain.PackageSortField
	SortDesc bool
	Offset   int
	Limit    int
}

// ListPackagesResult represents the result of listing packages.
type ListPackagesResult struct {
	Packages []*PackageDTO
	Total    int64
	Offset   int
	Limit    int
}

// PackageDTO represents a package in query results.
type PackageDTO struct {
	ID            string   `json:"id"`
	PackageID     string   `json:"package_id"`
	Type          string   `json:"type"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Author        string   `json:"author"`
	Homepage      string   `json:"homepage,omitempty"`
	License       string   `json:"license,omitempty"`
	Tags          []string `json:"tags"`
	LatestVersion string   `json:"latest_version"`
	Downloads     int64    `json:"downloads"`
	Rating        float64  `json:"rating"`
	RatingCount   int      `json:"rating_count"`
	Verified      bool     `json:"verified"`
	Featured      bool     `json:"featured"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

// ListPackagesHandler handles listing marketplace packages.
type ListPackagesHandler struct {
	repo domain.PackageRepository
}

// NewListPackagesHandler creates a new list packages handler.
func NewListPackagesHandler(repo domain.PackageRepository) *ListPackagesHandler {
	return &ListPackagesHandler{repo: repo}
}

// Handle executes the list packages query.
func (h *ListPackagesHandler) Handle(ctx context.Context, query ListPackagesQuery) (*ListPackagesResult, error) {
	filter := domain.PackageFilter{
		Type:     query.Type,
		Tags:     query.Tags,
		Verified: query.Verified,
		Featured: query.Featured,
		Offset:   query.Offset,
		Limit:    query.Limit,
	}

	if query.SortBy != "" {
		filter.SortBy = query.SortBy
	} else {
		filter.SortBy = domain.SortByDownloads
	}

	if query.SortDesc {
		filter.SortOrder = domain.SortDesc
	} else {
		filter.SortOrder = domain.SortAsc
	}

	if filter.Limit == 0 {
		filter.Limit = 20
	}

	packages, total, err := h.repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	dtos := make([]*PackageDTO, len(packages))
	for i, pkg := range packages {
		dtos[i] = packageToDTO(pkg)
	}

	return &ListPackagesResult{
		Packages: dtos,
		Total:    total,
		Offset:   query.Offset,
		Limit:    query.Limit,
	}, nil
}

func packageToDTO(pkg *domain.Package) *PackageDTO {
	return &PackageDTO{
		ID:            pkg.ID.String(),
		PackageID:     pkg.PackageID,
		Type:          string(pkg.Type),
		Name:          pkg.Name,
		Description:   pkg.Description,
		Author:        pkg.Author,
		Homepage:      pkg.Homepage,
		License:       pkg.License,
		Tags:          pkg.Tags,
		LatestVersion: pkg.LatestVersion,
		Downloads:     pkg.Downloads,
		Rating:        pkg.Rating,
		RatingCount:   pkg.RatingCount,
		Verified:      pkg.Verified,
		Featured:      pkg.Featured,
		CreatedAt:     pkg.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:     pkg.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
