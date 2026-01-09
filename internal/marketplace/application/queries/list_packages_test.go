package queries

import (
	"context"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/marketplace/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockPackageRepository is a mock implementation of domain.PackageRepository.
type MockPackageRepository struct {
	mock.Mock
}

func (m *MockPackageRepository) Create(ctx context.Context, pkg *domain.Package) error {
	args := m.Called(ctx, pkg)
	return args.Error(0)
}

func (m *MockPackageRepository) Update(ctx context.Context, pkg *domain.Package) error {
	args := m.Called(ctx, pkg)
	return args.Error(0)
}

func (m *MockPackageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPackageRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Package, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Package), args.Error(1)
}

func (m *MockPackageRepository) GetByPackageID(ctx context.Context, packageID string) (*domain.Package, error) {
	args := m.Called(ctx, packageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Package), args.Error(1)
}

func (m *MockPackageRepository) List(ctx context.Context, filter domain.PackageFilter) ([]*domain.Package, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.Package), args.Get(1).(int64), args.Error(2)
}

func (m *MockPackageRepository) Search(ctx context.Context, query string, filter domain.PackageFilter) ([]*domain.Package, int64, error) {
	args := m.Called(ctx, query, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.Package), args.Get(1).(int64), args.Error(2)
}

func (m *MockPackageRepository) GetFeatured(ctx context.Context, limit int) ([]*domain.Package, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Package), args.Error(1)
}

func (m *MockPackageRepository) GetByPublisher(ctx context.Context, publisherID uuid.UUID, filter domain.PackageFilter) ([]*domain.Package, int64, error) {
	args := m.Called(ctx, publisherID, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.Package), args.Get(1).(int64), args.Error(2)
}

func (m *MockPackageRepository) IncrementDownloads(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func createTestPackage(id string, name string, pkgType domain.PackageType) *domain.Package {
	now := time.Now().UTC()
	return &domain.Package{
		ID:            uuid.New(),
		PackageID:     id,
		Type:          pkgType,
		Name:          name,
		Description:   "Test description",
		Author:        "Test Author",
		Homepage:      "https://example.com",
		License:       "MIT",
		Tags:          []string{"test"},
		LatestVersion: "1.0.0",
		Downloads:     100,
		Rating:        4.5,
		RatingCount:   10,
		Verified:      true,
		Featured:      false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func TestListPackagesHandler_Handle_Success(t *testing.T) {
	mockRepo := new(MockPackageRepository)
	handler := NewListPackagesHandler(mockRepo)

	pkg1 := createTestPackage("test.orbit1", "Test Orbit 1", domain.PackageTypeOrbit)
	pkg2 := createTestPackage("test.orbit2", "Test Orbit 2", domain.PackageTypeOrbit)
	packages := []*domain.Package{pkg1, pkg2}

	query := ListPackagesQuery{
		Offset:   0,
		Limit:    20,
		SortBy:   domain.SortByDownloads,
		SortDesc: true,
	}

	mockRepo.On("List", mock.Anything, mock.MatchedBy(func(filter domain.PackageFilter) bool {
		return filter.Offset == 0 && filter.Limit == 20
	})).Return(packages, int64(2), nil)

	result, err := handler.Handle(context.Background(), query)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Packages, 2)
	assert.Equal(t, int64(2), result.Total)
	assert.Equal(t, 0, result.Offset)
	assert.Equal(t, 20, result.Limit)
	mockRepo.AssertExpectations(t)
}

func TestListPackagesHandler_Handle_WithTypeFilter(t *testing.T) {
	mockRepo := new(MockPackageRepository)
	handler := NewListPackagesHandler(mockRepo)

	orbitType := domain.PackageTypeOrbit
	pkg := createTestPackage("test.orbit", "Test Orbit", domain.PackageTypeOrbit)
	packages := []*domain.Package{pkg}

	query := ListPackagesQuery{
		Offset: 0,
		Limit:  20,
		Type:   &orbitType,
	}

	mockRepo.On("List", mock.Anything, mock.MatchedBy(func(filter domain.PackageFilter) bool {
		return filter.Type != nil && *filter.Type == domain.PackageTypeOrbit
	})).Return(packages, int64(1), nil)

	result, err := handler.Handle(context.Background(), query)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Packages, 1)
	assert.Equal(t, "orbit", result.Packages[0].Type)
	mockRepo.AssertExpectations(t)
}

func TestListPackagesHandler_Handle_WithTagsFilter(t *testing.T) {
	mockRepo := new(MockPackageRepository)
	handler := NewListPackagesHandler(mockRepo)

	pkg := createTestPackage("test.orbit", "Test Orbit", domain.PackageTypeOrbit)
	packages := []*domain.Package{pkg}

	query := ListPackagesQuery{
		Offset: 0,
		Limit:  20,
		Tags:   []string{"productivity"},
	}

	mockRepo.On("List", mock.Anything, mock.MatchedBy(func(filter domain.PackageFilter) bool {
		return len(filter.Tags) == 1 && filter.Tags[0] == "productivity"
	})).Return(packages, int64(1), nil)

	result, err := handler.Handle(context.Background(), query)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Packages, 1)
	mockRepo.AssertExpectations(t)
}

func TestListPackagesHandler_Handle_WithVerifiedFilter(t *testing.T) {
	mockRepo := new(MockPackageRepository)
	handler := NewListPackagesHandler(mockRepo)

	verified := true
	pkg := createTestPackage("test.orbit", "Test Orbit", domain.PackageTypeOrbit)
	pkg.Verified = true
	packages := []*domain.Package{pkg}

	query := ListPackagesQuery{
		Offset:   0,
		Limit:    20,
		Verified: &verified,
	}

	mockRepo.On("List", mock.Anything, mock.MatchedBy(func(filter domain.PackageFilter) bool {
		return filter.Verified != nil && *filter.Verified == true
	})).Return(packages, int64(1), nil)

	result, err := handler.Handle(context.Background(), query)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Packages, 1)
	assert.True(t, result.Packages[0].Verified)
	mockRepo.AssertExpectations(t)
}

func TestListPackagesHandler_Handle_EmptyResult(t *testing.T) {
	mockRepo := new(MockPackageRepository)
	handler := NewListPackagesHandler(mockRepo)

	query := ListPackagesQuery{
		Offset: 0,
		Limit:  20,
	}

	mockRepo.On("List", mock.Anything, mock.Anything).Return([]*domain.Package{}, int64(0), nil)

	result, err := handler.Handle(context.Background(), query)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Packages, 0)
	assert.Equal(t, int64(0), result.Total)
	mockRepo.AssertExpectations(t)
}

func TestListPackagesHandler_Handle_Pagination(t *testing.T) {
	mockRepo := new(MockPackageRepository)
	handler := NewListPackagesHandler(mockRepo)

	pkg := createTestPackage("test.orbit", "Test Orbit", domain.PackageTypeOrbit)
	packages := []*domain.Package{pkg}

	query := ListPackagesQuery{
		Offset: 10,
		Limit:  5,
	}

	mockRepo.On("List", mock.Anything, mock.MatchedBy(func(filter domain.PackageFilter) bool {
		return filter.Offset == 10 && filter.Limit == 5
	})).Return(packages, int64(15), nil)

	result, err := handler.Handle(context.Background(), query)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Packages, 1)
	assert.Equal(t, int64(15), result.Total)
	assert.Equal(t, 10, result.Offset)
	assert.Equal(t, 5, result.Limit)
	mockRepo.AssertExpectations(t)
}

func TestPackageDTO_Fields(t *testing.T) {
	dto := &PackageDTO{
		ID:            "test-id",
		PackageID:     "test.orbit",
		Type:          "orbit",
		Name:          "Test Orbit",
		Description:   "Test description",
		Author:        "Test Author",
		Homepage:      "https://example.com",
		License:       "MIT",
		Tags:          []string{"test"},
		LatestVersion: "1.0.0",
		Downloads:     100,
		Rating:        4.5,
		RatingCount:   10,
		Verified:      true,
		Featured:      false,
		CreatedAt:     "2024-01-01T00:00:00Z",
		UpdatedAt:     "2024-01-01T00:00:00Z",
	}

	assert.Equal(t, "test-id", dto.ID)
	assert.Equal(t, "test.orbit", dto.PackageID)
	assert.Equal(t, "orbit", dto.Type)
	assert.Equal(t, "Test Orbit", dto.Name)
	assert.Equal(t, "Test description", dto.Description)
	assert.Equal(t, "Test Author", dto.Author)
	assert.Equal(t, "https://example.com", dto.Homepage)
	assert.Equal(t, "MIT", dto.License)
	assert.Equal(t, []string{"test"}, dto.Tags)
	assert.Equal(t, "1.0.0", dto.LatestVersion)
	assert.Equal(t, int64(100), dto.Downloads)
	assert.Equal(t, 4.5, dto.Rating)
	assert.Equal(t, 10, dto.RatingCount)
	assert.True(t, dto.Verified)
	assert.False(t, dto.Featured)
}
