package queries

import (
	"context"

	"github.com/felixgeelhaar/orbita/internal/marketplace/domain"
)

// GetFeaturedQuery represents a query to get featured packages.
type GetFeaturedQuery struct {
	Limit int
}

// GetFeaturedResult represents the result of getting featured packages.
type GetFeaturedResult struct {
	Packages []*PackageDTO
}

// GetFeaturedHandler handles getting featured packages.
type GetFeaturedHandler struct {
	repo domain.PackageRepository
}

// NewGetFeaturedHandler creates a new get featured handler.
func NewGetFeaturedHandler(repo domain.PackageRepository) *GetFeaturedHandler {
	return &GetFeaturedHandler{repo: repo}
}

// Handle executes the get featured query.
func (h *GetFeaturedHandler) Handle(ctx context.Context, query GetFeaturedQuery) (*GetFeaturedResult, error) {
	limit := query.Limit
	if limit == 0 {
		limit = 10
	}

	packages, err := h.repo.GetFeatured(ctx, limit)
	if err != nil {
		return nil, err
	}

	dtos := make([]*PackageDTO, len(packages))
	for i, pkg := range packages {
		dtos[i] = packageToDTO(pkg)
	}

	return &GetFeaturedResult{
		Packages: dtos,
	}, nil
}
