package api

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/felixgeelhaar/orbita/internal/marketplace/application/queries"
	"github.com/felixgeelhaar/orbita/internal/marketplace/domain"
	"github.com/google/uuid"
)

// MarketplaceHandler handles marketplace API requests.
type MarketplaceHandler struct {
	listPackages   *queries.ListPackagesHandler
	searchPackages *queries.SearchPackagesHandler
	getPackage     *queries.GetPackageHandler
	getFeatured    *queries.GetFeaturedHandler
	versionRepo    domain.VersionRepository
	publisherRepo  domain.PublisherRepository
	packageRepo    domain.PackageRepository
	logger         *slog.Logger
}

// MarketplaceHandlerConfig holds dependencies for the marketplace handler.
type MarketplaceHandlerConfig struct {
	ListPackages   *queries.ListPackagesHandler
	SearchPackages *queries.SearchPackagesHandler
	GetPackage     *queries.GetPackageHandler
	GetFeatured    *queries.GetFeaturedHandler
	VersionRepo    domain.VersionRepository
	PublisherRepo  domain.PublisherRepository
	PackageRepo    domain.PackageRepository
	Logger         *slog.Logger
}

// NewMarketplaceHandler creates a new marketplace handler.
func NewMarketplaceHandler(cfg MarketplaceHandlerConfig) *MarketplaceHandler {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	return &MarketplaceHandler{
		listPackages:   cfg.ListPackages,
		searchPackages: cfg.SearchPackages,
		getPackage:     cfg.GetPackage,
		getFeatured:    cfg.GetFeatured,
		versionRepo:    cfg.VersionRepo,
		publisherRepo:  cfg.PublisherRepo,
		packageRepo:    cfg.PackageRepo,
		logger:         cfg.Logger,
	}
}

// ListPackages handles GET /api/v1/packages
func (h *MarketplaceHandler) ListPackages(w http.ResponseWriter, r *http.Request) {
	query := queries.ListPackagesQuery{
		Offset:   parseIntParam(r, "offset", 0),
		Limit:    parseIntParam(r, "limit", 20),
		SortDesc: parseBoolParam(r, "desc", true),
	}

	// Parse optional filters
	if typeParam := r.URL.Query().Get("type"); typeParam != "" {
		t := domain.PackageType(typeParam)
		query.Type = &t
	}

	if tagsParam := r.URL.Query().Get("tags"); tagsParam != "" {
		query.Tags = strings.Split(tagsParam, ",")
	}

	if verifiedParam := r.URL.Query().Get("verified"); verifiedParam != "" {
		v := verifiedParam == "true"
		query.Verified = &v
	}

	if sortBy := r.URL.Query().Get("sort"); sortBy != "" {
		query.SortBy = domain.PackageSortField(sortBy)
	}

	result, err := h.listPackages.Handle(r.Context(), query)
	if err != nil {
		h.logger.Error("failed to list packages", "error", err)
		writeError(w, http.StatusInternalServerError, "Failed to list packages")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// SearchPackages handles GET /api/v1/packages/search
func (h *MarketplaceHandler) SearchPackages(w http.ResponseWriter, r *http.Request) {
	queryStr := r.URL.Query().Get("q")
	if queryStr == "" {
		writeError(w, http.StatusBadRequest, "Query parameter 'q' is required")
		return
	}

	query := queries.SearchPackagesQuery{
		Query:    queryStr,
		Offset:   parseIntParam(r, "offset", 0),
		Limit:    parseIntParam(r, "limit", 20),
		SortDesc: parseBoolParam(r, "desc", true),
	}

	// Parse optional filters
	if typeParam := r.URL.Query().Get("type"); typeParam != "" {
		t := domain.PackageType(typeParam)
		query.Type = &t
	}

	if tagsParam := r.URL.Query().Get("tags"); tagsParam != "" {
		query.Tags = strings.Split(tagsParam, ",")
	}

	if verifiedParam := r.URL.Query().Get("verified"); verifiedParam != "" {
		v := verifiedParam == "true"
		query.Verified = &v
	}

	result, err := h.searchPackages.Handle(r.Context(), query)
	if err != nil {
		h.logger.Error("failed to search packages", "error", err)
		writeError(w, http.StatusInternalServerError, "Failed to search packages")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetFeatured handles GET /api/v1/packages/featured
func (h *MarketplaceHandler) GetFeatured(w http.ResponseWriter, r *http.Request) {
	query := queries.GetFeaturedQuery{
		Limit: parseIntParam(r, "limit", 10),
	}

	result, err := h.getFeatured.Handle(r.Context(), query)
	if err != nil {
		h.logger.Error("failed to get featured packages", "error", err)
		writeError(w, http.StatusInternalServerError, "Failed to get featured packages")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetPackage handles GET /api/v1/packages/{packageID}
func (h *MarketplaceHandler) GetPackage(w http.ResponseWriter, r *http.Request) {
	packageID := r.PathValue("packageID")
	if packageID == "" {
		writeError(w, http.StatusBadRequest, "Package ID is required")
		return
	}

	query := queries.GetPackageQuery{
		PackageID: &packageID,
	}

	// Check if it's a UUID
	if id, err := uuid.Parse(packageID); err == nil {
		query.ID = &id
		query.PackageID = nil
	}

	result, err := h.getPackage.Handle(r.Context(), query)
	if err != nil {
		if err == queries.ErrPackageNotFound {
			writeError(w, http.StatusNotFound, "Package not found")
			return
		}
		h.logger.Error("failed to get package", "error", err)
		writeError(w, http.StatusInternalServerError, "Failed to get package")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetVersions handles GET /api/v1/packages/{packageID}/versions
func (h *MarketplaceHandler) GetVersions(w http.ResponseWriter, r *http.Request) {
	packageID := r.PathValue("packageID")
	if packageID == "" {
		writeError(w, http.StatusBadRequest, "Package ID is required")
		return
	}

	// First get the package to get its UUID
	pkg, err := h.packageRepo.GetByPackageID(r.Context(), packageID)
	if err != nil || pkg == nil {
		writeError(w, http.StatusNotFound, "Package not found")
		return
	}

	versions, err := h.versionRepo.ListByPackage(r.Context(), pkg.ID)
	if err != nil {
		h.logger.Error("failed to get versions", "error", err)
		writeError(w, http.StatusInternalServerError, "Failed to get versions")
		return
	}

	dtos := make([]*queries.VersionDTO, len(versions))
	for i, v := range versions {
		dtos[i] = &queries.VersionDTO{
			ID:                 v.ID.String(),
			Version:            v.Version,
			MinAPIVersion:      v.MinAPIVersion,
			Changelog:          v.Changelog,
			Checksum:           v.Checksum,
			Size:               v.Size,
			Downloads:          v.Downloads,
			Prerelease:         v.Prerelease,
			Deprecated:         v.Deprecated,
			DeprecationMessage: v.DeprecationMessage,
			PublishedAt:        v.PublishedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"versions": dtos,
	})
}

// GetVersion handles GET /api/v1/packages/{packageID}/versions/{version}
func (h *MarketplaceHandler) GetVersion(w http.ResponseWriter, r *http.Request) {
	packageID := r.PathValue("packageID")
	versionStr := r.PathValue("version")

	if packageID == "" || versionStr == "" {
		writeError(w, http.StatusBadRequest, "Package ID and version are required")
		return
	}

	// First get the package to get its UUID
	pkg, err := h.packageRepo.GetByPackageID(r.Context(), packageID)
	if err != nil || pkg == nil {
		writeError(w, http.StatusNotFound, "Package not found")
		return
	}

	version, err := h.versionRepo.GetByPackageAndVersion(r.Context(), pkg.ID, versionStr)
	if err != nil || version == nil {
		writeError(w, http.StatusNotFound, "Version not found")
		return
	}

	writeJSON(w, http.StatusOK, &queries.VersionDTO{
		ID:                 version.ID.String(),
		Version:            version.Version,
		MinAPIVersion:      version.MinAPIVersion,
		Changelog:          version.Changelog,
		Checksum:           version.Checksum,
		DownloadURL:        version.DownloadURL,
		Size:               version.Size,
		Downloads:          version.Downloads,
		Prerelease:         version.Prerelease,
		Deprecated:         version.Deprecated,
		DeprecationMessage: version.DeprecationMessage,
		PublishedAt:        version.PublishedAt.Format("2006-01-02T15:04:05Z"),
	})
}

// DownloadPackage handles GET /api/v1/packages/{packageID}/download
func (h *MarketplaceHandler) DownloadPackage(w http.ResponseWriter, r *http.Request) {
	packageID := r.PathValue("packageID")
	versionStr := r.URL.Query().Get("version")

	if packageID == "" {
		writeError(w, http.StatusBadRequest, "Package ID is required")
		return
	}

	// Get the package
	pkg, err := h.packageRepo.GetByPackageID(r.Context(), packageID)
	if err != nil || pkg == nil {
		writeError(w, http.StatusNotFound, "Package not found")
		return
	}

	// Get the version (latest if not specified)
	var version *domain.Version
	if versionStr == "" || versionStr == "latest" {
		version, err = h.versionRepo.GetLatestStable(r.Context(), pkg.ID)
	} else {
		version, err = h.versionRepo.GetByPackageAndVersion(r.Context(), pkg.ID, versionStr)
	}

	if err != nil || version == nil {
		writeError(w, http.StatusNotFound, "Version not found")
		return
	}

	// Increment download counts
	_ = h.versionRepo.IncrementDownloads(r.Context(), version.ID)
	_ = h.packageRepo.IncrementDownloads(r.Context(), pkg.ID)

	// Return download URL
	writeJSON(w, http.StatusOK, map[string]string{
		"download_url": version.DownloadURL,
		"checksum":     version.Checksum,
		"version":      version.Version,
	})
}

// ListPublishers handles GET /api/v1/publishers
func (h *MarketplaceHandler) ListPublishers(w http.ResponseWriter, r *http.Request) {
	offset := parseIntParam(r, "offset", 0)
	limit := parseIntParam(r, "limit", 20)

	publishers, total, err := h.publisherRepo.List(r.Context(), offset, limit)
	if err != nil {
		h.logger.Error("failed to list publishers", "error", err)
		writeError(w, http.StatusInternalServerError, "Failed to list publishers")
		return
	}

	dtos := make([]*queries.PublisherDTO, len(publishers))
	for i, p := range publishers {
		dtos[i] = &queries.PublisherDTO{
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

	writeJSON(w, http.StatusOK, map[string]any{
		"publishers": dtos,
		"total":      total,
		"offset":     offset,
		"limit":      limit,
	})
}

// GetPublisher handles GET /api/v1/publishers/{slug}
func (h *MarketplaceHandler) GetPublisher(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "Publisher slug is required")
		return
	}

	publisher, err := h.publisherRepo.GetBySlug(r.Context(), slug)
	if err != nil || publisher == nil {
		writeError(w, http.StatusNotFound, "Publisher not found")
		return
	}

	writeJSON(w, http.StatusOK, &queries.PublisherDTO{
		ID:             publisher.ID.String(),
		Name:           publisher.Name,
		Slug:           publisher.Slug,
		Website:        publisher.Website,
		Description:    publisher.Description,
		Verified:       publisher.Verified,
		AvatarURL:      publisher.AvatarURL,
		PackageCount:   publisher.PackageCount,
		TotalDownloads: publisher.TotalDownloads,
	})
}

// GetPublisherPackages handles GET /api/v1/publishers/{slug}/packages
func (h *MarketplaceHandler) GetPublisherPackages(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "Publisher slug is required")
		return
	}

	// Get the publisher
	publisher, err := h.publisherRepo.GetBySlug(r.Context(), slug)
	if err != nil || publisher == nil {
		writeError(w, http.StatusNotFound, "Publisher not found")
		return
	}

	filter := domain.PackageFilter{
		Offset:    parseIntParam(r, "offset", 0),
		Limit:     parseIntParam(r, "limit", 20),
		SortBy:    domain.SortByDownloads,
		SortOrder: domain.SortDesc,
	}

	packages, total, err := h.packageRepo.GetByPublisher(r.Context(), publisher.ID, filter)
	if err != nil {
		h.logger.Error("failed to get publisher packages", "error", err)
		writeError(w, http.StatusInternalServerError, "Failed to get publisher packages")
		return
	}

	dtos := make([]*queries.PackageDTO, len(packages))
	for i, pkg := range packages {
		dtos[i] = &queries.PackageDTO{
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

	writeJSON(w, http.StatusOK, map[string]any{
		"packages": dtos,
		"total":    total,
		"offset":   filter.Offset,
		"limit":    filter.Limit,
	})
}

// Helper functions

func parseIntParam(r *http.Request, key string, defaultVal int) int {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return i
}

func parseBoolParam(r *http.Request, key string, defaultVal bool) bool {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultVal
	}
	return val == "true" || val == "1"
}
