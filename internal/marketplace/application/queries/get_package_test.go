package queries

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/marketplace/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockVersionRepository is a mock implementation of domain.VersionRepository.
type MockVersionRepository struct {
	mock.Mock
}

func (m *MockVersionRepository) Create(ctx context.Context, version *domain.Version) error {
	args := m.Called(ctx, version)
	return args.Error(0)
}

func (m *MockVersionRepository) Update(ctx context.Context, version *domain.Version) error {
	args := m.Called(ctx, version)
	return args.Error(0)
}

func (m *MockVersionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockVersionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Version, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Version), args.Error(1)
}

func (m *MockVersionRepository) GetByPackageAndVersion(ctx context.Context, packageID uuid.UUID, version string) (*domain.Version, error) {
	args := m.Called(ctx, packageID, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Version), args.Error(1)
}

func (m *MockVersionRepository) ListByPackage(ctx context.Context, packageID uuid.UUID) ([]*domain.Version, error) {
	args := m.Called(ctx, packageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Version), args.Error(1)
}

func (m *MockVersionRepository) GetLatestStable(ctx context.Context, packageID uuid.UUID) (*domain.Version, error) {
	args := m.Called(ctx, packageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Version), args.Error(1)
}

func (m *MockVersionRepository) IncrementDownloads(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockPublisherRepository is a mock implementation of domain.PublisherRepository.
type MockPublisherRepository struct {
	mock.Mock
}

func (m *MockPublisherRepository) Create(ctx context.Context, publisher *domain.Publisher) error {
	args := m.Called(ctx, publisher)
	return args.Error(0)
}

func (m *MockPublisherRepository) Update(ctx context.Context, publisher *domain.Publisher) error {
	args := m.Called(ctx, publisher)
	return args.Error(0)
}

func (m *MockPublisherRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPublisherRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Publisher, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Publisher), args.Error(1)
}

func (m *MockPublisherRepository) GetBySlug(ctx context.Context, slug string) (*domain.Publisher, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Publisher), args.Error(1)
}

func (m *MockPublisherRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Publisher, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Publisher), args.Error(1)
}

func (m *MockPublisherRepository) List(ctx context.Context, offset, limit int) ([]*domain.Publisher, int64, error) {
	args := m.Called(ctx, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.Publisher), args.Get(1).(int64), args.Error(2)
}

func (m *MockPublisherRepository) Search(ctx context.Context, query string, offset, limit int) ([]*domain.Publisher, int64, error) {
	args := m.Called(ctx, query, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.Publisher), args.Get(1).(int64), args.Error(2)
}

func createTestVersion(packageID uuid.UUID, version string) *domain.Version {
	return &domain.Version{
		ID:            uuid.New(),
		PackageID:     packageID,
		Version:       version,
		MinAPIVersion: "1.0.0",
		Changelog:     "Initial release",
		Checksum:      "sha256:abc123",
		DownloadURL:   "https://example.com/download",
		Size:          1024,
		Downloads:     50,
		Prerelease:    false,
		Deprecated:    false,
		PublishedAt:   time.Now().UTC(),
	}
}

func createTestPublisher(id uuid.UUID) *domain.Publisher {
	return &domain.Publisher{
		ID:             id,
		Name:           "Test Publisher",
		Slug:           "test-publisher",
		Website:        "https://example.com",
		Description:    "A test publisher",
		Verified:       true,
		AvatarURL:      "https://example.com/avatar.png",
		PackageCount:   5,
		TotalDownloads: 1000,
	}
}

func TestGetPackageHandler_Handle(t *testing.T) {
	t.Run("successfully gets package by ID", func(t *testing.T) {
		mockPackageRepo := new(MockPackageRepository)
		mockVersionRepo := new(MockVersionRepository)
		mockPublisherRepo := new(MockPublisherRepository)
		handler := NewGetPackageHandler(mockPackageRepo, mockVersionRepo, mockPublisherRepo)

		packageID := uuid.New()
		publisherID := uuid.New()
		pkg := createTestPackage("test.orbit", "Test Orbit", domain.PackageTypeOrbit)
		pkg.ID = packageID
		pkg.PublisherID = publisherID
		version := createTestVersion(packageID, "1.0.0")
		publisher := createTestPublisher(publisherID)

		mockPackageRepo.On("GetByID", mock.Anything, packageID).Return(pkg, nil)
		mockVersionRepo.On("ListByPackage", mock.Anything, packageID).Return([]*domain.Version{version}, nil)
		mockPublisherRepo.On("GetByID", mock.Anything, publisherID).Return(publisher, nil)

		query := GetPackageQuery{
			ID: &packageID,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "test.orbit", result.PackageID)
		assert.Equal(t, "Test Orbit", result.Name)
		assert.Len(t, result.Versions, 1)
		assert.NotNil(t, result.Publisher)
		assert.Equal(t, "Test Publisher", result.Publisher.Name)
		mockPackageRepo.AssertExpectations(t)
		mockVersionRepo.AssertExpectations(t)
		mockPublisherRepo.AssertExpectations(t)
	})

	t.Run("successfully gets package by package ID string", func(t *testing.T) {
		mockPackageRepo := new(MockPackageRepository)
		mockVersionRepo := new(MockVersionRepository)
		mockPublisherRepo := new(MockPublisherRepository)
		handler := NewGetPackageHandler(mockPackageRepo, mockVersionRepo, mockPublisherRepo)

		packageIDStr := "test.orbit"
		pkg := createTestPackage(packageIDStr, "Test Orbit", domain.PackageTypeOrbit)

		mockPackageRepo.On("GetByPackageID", mock.Anything, packageIDStr).Return(pkg, nil)
		mockVersionRepo.On("ListByPackage", mock.Anything, pkg.ID).Return([]*domain.Version{}, nil)

		query := GetPackageQuery{
			PackageID: &packageIDStr,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "test.orbit", result.PackageID)
		mockPackageRepo.AssertExpectations(t)
		mockVersionRepo.AssertExpectations(t)
	})

	t.Run("returns error when neither ID nor PackageID provided", func(t *testing.T) {
		mockPackageRepo := new(MockPackageRepository)
		mockVersionRepo := new(MockVersionRepository)
		mockPublisherRepo := new(MockPublisherRepository)
		handler := NewGetPackageHandler(mockPackageRepo, mockVersionRepo, mockPublisherRepo)

		query := GetPackageQuery{}

		result, err := handler.Handle(context.Background(), query)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "either ID or PackageID must be provided")
	})

	t.Run("returns ErrPackageNotFound when package does not exist", func(t *testing.T) {
		mockPackageRepo := new(MockPackageRepository)
		mockVersionRepo := new(MockVersionRepository)
		mockPublisherRepo := new(MockPublisherRepository)
		handler := NewGetPackageHandler(mockPackageRepo, mockVersionRepo, mockPublisherRepo)

		packageID := uuid.New()
		mockPackageRepo.On("GetByID", mock.Anything, packageID).Return(nil, nil)

		query := GetPackageQuery{
			ID: &packageID,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.ErrorIs(t, err, ErrPackageNotFound)
		assert.Nil(t, result)
		mockPackageRepo.AssertExpectations(t)
	})

	t.Run("fails when package repo returns error", func(t *testing.T) {
		mockPackageRepo := new(MockPackageRepository)
		mockVersionRepo := new(MockVersionRepository)
		mockPublisherRepo := new(MockPublisherRepository)
		handler := NewGetPackageHandler(mockPackageRepo, mockVersionRepo, mockPublisherRepo)

		packageID := uuid.New()
		mockPackageRepo.On("GetByID", mock.Anything, packageID).Return(nil, errors.New("database error"))

		query := GetPackageQuery{
			ID: &packageID,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "database error")
		mockPackageRepo.AssertExpectations(t)
	})

	t.Run("fails when version repo returns error", func(t *testing.T) {
		mockPackageRepo := new(MockPackageRepository)
		mockVersionRepo := new(MockVersionRepository)
		mockPublisherRepo := new(MockPublisherRepository)
		handler := NewGetPackageHandler(mockPackageRepo, mockVersionRepo, mockPublisherRepo)

		packageID := uuid.New()
		pkg := createTestPackage("test.orbit", "Test Orbit", domain.PackageTypeOrbit)
		pkg.ID = packageID

		mockPackageRepo.On("GetByID", mock.Anything, packageID).Return(pkg, nil)
		mockVersionRepo.On("ListByPackage", mock.Anything, packageID).Return(nil, errors.New("version error"))

		query := GetPackageQuery{
			ID: &packageID,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "version error")
		mockPackageRepo.AssertExpectations(t)
		mockVersionRepo.AssertExpectations(t)
	})

	t.Run("returns package without publisher when publisher not found", func(t *testing.T) {
		mockPackageRepo := new(MockPackageRepository)
		mockVersionRepo := new(MockVersionRepository)
		mockPublisherRepo := new(MockPublisherRepository)
		handler := NewGetPackageHandler(mockPackageRepo, mockVersionRepo, mockPublisherRepo)

		packageID := uuid.New()
		publisherID := uuid.New()
		pkg := createTestPackage("test.orbit", "Test Orbit", domain.PackageTypeOrbit)
		pkg.ID = packageID
		pkg.PublisherID = publisherID

		mockPackageRepo.On("GetByID", mock.Anything, packageID).Return(pkg, nil)
		mockVersionRepo.On("ListByPackage", mock.Anything, packageID).Return([]*domain.Version{}, nil)
		mockPublisherRepo.On("GetByID", mock.Anything, publisherID).Return(nil, errors.New("not found"))

		query := GetPackageQuery{
			ID: &packageID,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Nil(t, result.Publisher)
		mockPackageRepo.AssertExpectations(t)
		mockVersionRepo.AssertExpectations(t)
		mockPublisherRepo.AssertExpectations(t)
	})

	t.Run("returns package without publisher when PublisherID is nil", func(t *testing.T) {
		mockPackageRepo := new(MockPackageRepository)
		mockVersionRepo := new(MockVersionRepository)
		mockPublisherRepo := new(MockPublisherRepository)
		handler := NewGetPackageHandler(mockPackageRepo, mockVersionRepo, mockPublisherRepo)

		packageID := uuid.New()
		pkg := createTestPackage("test.orbit", "Test Orbit", domain.PackageTypeOrbit)
		pkg.ID = packageID
		pkg.PublisherID = uuid.Nil

		mockPackageRepo.On("GetByID", mock.Anything, packageID).Return(pkg, nil)
		mockVersionRepo.On("ListByPackage", mock.Anything, packageID).Return([]*domain.Version{}, nil)

		query := GetPackageQuery{
			ID: &packageID,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Nil(t, result.Publisher)
		mockPackageRepo.AssertExpectations(t)
		mockVersionRepo.AssertExpectations(t)
	})
}

func TestNewGetPackageHandler(t *testing.T) {
	mockPackageRepo := new(MockPackageRepository)
	mockVersionRepo := new(MockVersionRepository)
	mockPublisherRepo := new(MockPublisherRepository)

	handler := NewGetPackageHandler(mockPackageRepo, mockVersionRepo, mockPublisherRepo)

	assert.NotNil(t, handler)
}
