package queries

import (
	"context"

	"github.com/felixgeelhaar/orbita/internal/marketplace/domain"
)

// SearchPackagesQuery represents a query to search marketplace packages.
type SearchPackagesQuery struct {
	Query    string
	Type     *domain.PackageType
	Tags     []string
	Verified *bool
	SortBy   domain.PackageSortField
	SortDesc bool
	Offset   int
	Limit    int
}

// SearchPackagesResult represents the result of searching packages.
type SearchPackagesResult struct {
	Packages []*PackageDTO
	Total    int64
	Query    string
	Offset   int
	Limit    int
}

// SearchPackagesHandler handles searching marketplace packages.
type SearchPackagesHandler struct {
	repo domain.PackageRepository
}

// NewSearchPackagesHandler creates a new search packages handler.
func NewSearchPackagesHandler(repo domain.PackageRepository) *SearchPackagesHandler {
	return &SearchPackagesHandler{repo: repo}
}

// Handle executes the search packages query.
func (h *SearchPackagesHandler) Handle(ctx context.Context, query SearchPackagesQuery) (*SearchPackagesResult, error) {
	filter := domain.PackageFilter{
		Type:     query.Type,
		Tags:     query.Tags,
		Verified: query.Verified,
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

	packages, total, err := h.repo.Search(ctx, query.Query, filter)
	if err != nil {
		return nil, err
	}

	dtos := make([]*PackageDTO, len(packages))
	for i, pkg := range packages {
		dtos[i] = packageToDTO(pkg)
	}

	return &SearchPackagesResult{
		Packages: dtos,
		Total:    total,
		Query:    query.Query,
		Offset:   query.Offset,
		Limit:    query.Limit,
	}, nil
}
