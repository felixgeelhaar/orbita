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

// MockInstalledPackageRepository is a mock implementation of domain.InstalledPackageRepository.
type MockInstalledPackageRepository struct {
	mock.Mock
}

func (m *MockInstalledPackageRepository) Create(ctx context.Context, pkg *domain.InstalledPackage) error {
	args := m.Called(ctx, pkg)
	return args.Error(0)
}

func (m *MockInstalledPackageRepository) Update(ctx context.Context, pkg *domain.InstalledPackage) error {
	args := m.Called(ctx, pkg)
	return args.Error(0)
}

func (m *MockInstalledPackageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockInstalledPackageRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.InstalledPackage, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.InstalledPackage), args.Error(1)
}

func (m *MockInstalledPackageRepository) GetByPackageID(ctx context.Context, packageID string, userID uuid.UUID) (*domain.InstalledPackage, error) {
	args := m.Called(ctx, packageID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.InstalledPackage), args.Error(1)
}

func (m *MockInstalledPackageRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.InstalledPackage, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.InstalledPackage), args.Error(1)
}

func (m *MockInstalledPackageRepository) ListByType(ctx context.Context, userID uuid.UUID, pkgType domain.PackageType) ([]*domain.InstalledPackage, error) {
	args := m.Called(ctx, userID, pkgType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.InstalledPackage), args.Error(1)
}

func createTestInstalledPackage(packageID string, pkgType domain.PackageType, userID uuid.UUID) *domain.InstalledPackage {
	now := time.Now().UTC()
	return &domain.InstalledPackage{
		ID:          uuid.New(),
		PackageID:   packageID,
		Version:     "1.0.0",
		Type:        pkgType,
		InstallPath: "/path/to/" + packageID,
		Checksum:    "sha256:abc123",
		InstalledAt: now,
		UpdatedAt:   now,
		Enabled:     true,
		UserID:      userID,
	}
}

func TestListInstalledHandler_Handle(t *testing.T) {
	t.Run("successfully lists all installed packages", func(t *testing.T) {
		mockRepo := new(MockInstalledPackageRepository)
		handler := NewListInstalledHandler(mockRepo)

		userID := uuid.New()
		pkg1 := createTestInstalledPackage("test.orbit", domain.PackageTypeOrbit, userID)
		pkg2 := createTestInstalledPackage("test.engine", domain.PackageTypeEngine, userID)
		packages := []*domain.InstalledPackage{pkg1, pkg2}

		mockRepo.On("ListByUser", mock.Anything, userID).Return(packages, nil)

		query := ListInstalledQuery{
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Packages, 2)
		assert.Equal(t, 2, result.Total)
		assert.Equal(t, "test.orbit", result.Packages[0].PackageID)
		assert.Equal(t, "test.engine", result.Packages[1].PackageID)
		mockRepo.AssertExpectations(t)
	})

	t.Run("successfully lists installed packages by type", func(t *testing.T) {
		mockRepo := new(MockInstalledPackageRepository)
		handler := NewListInstalledHandler(mockRepo)

		userID := uuid.New()
		orbitType := domain.PackageTypeOrbit
		pkg := createTestInstalledPackage("test.orbit", domain.PackageTypeOrbit, userID)
		packages := []*domain.InstalledPackage{pkg}

		mockRepo.On("ListByType", mock.Anything, userID, orbitType).Return(packages, nil)

		query := ListInstalledQuery{
			UserID: userID,
			Type:   &orbitType,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Packages, 1)
		assert.Equal(t, "orbit", result.Packages[0].Type)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns empty list when no packages installed", func(t *testing.T) {
		mockRepo := new(MockInstalledPackageRepository)
		handler := NewListInstalledHandler(mockRepo)

		userID := uuid.New()

		mockRepo.On("ListByUser", mock.Anything, userID).Return([]*domain.InstalledPackage{}, nil)

		query := ListInstalledQuery{
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.Packages)
		assert.Equal(t, 0, result.Total)
		mockRepo.AssertExpectations(t)
	})

	t.Run("fails when ListByUser returns error", func(t *testing.T) {
		mockRepo := new(MockInstalledPackageRepository)
		handler := NewListInstalledHandler(mockRepo)

		userID := uuid.New()

		mockRepo.On("ListByUser", mock.Anything, userID).Return(nil, errors.New("database error"))

		query := ListInstalledQuery{
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "database error")
		mockRepo.AssertExpectations(t)
	})

	t.Run("fails when ListByType returns error", func(t *testing.T) {
		mockRepo := new(MockInstalledPackageRepository)
		handler := NewListInstalledHandler(mockRepo)

		userID := uuid.New()
		engineType := domain.PackageTypeEngine

		mockRepo.On("ListByType", mock.Anything, userID, engineType).Return(nil, errors.New("database error"))

		query := ListInstalledQuery{
			UserID: userID,
			Type:   &engineType,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "database error")
		mockRepo.AssertExpectations(t)
	})

	t.Run("correctly maps installed package to DTO", func(t *testing.T) {
		mockRepo := new(MockInstalledPackageRepository)
		handler := NewListInstalledHandler(mockRepo)

		userID := uuid.New()
		pkg := createTestInstalledPackage("test.orbit", domain.PackageTypeOrbit, userID)
		pkg.Enabled = false
		packages := []*domain.InstalledPackage{pkg}

		mockRepo.On("ListByUser", mock.Anything, userID).Return(packages, nil)

		query := ListInstalledQuery{
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Packages, 1)

		dto := result.Packages[0]
		assert.Equal(t, pkg.ID.String(), dto.ID)
		assert.Equal(t, "test.orbit", dto.PackageID)
		assert.Equal(t, "1.0.0", dto.Version)
		assert.Equal(t, "orbit", dto.Type)
		assert.Equal(t, "/path/to/test.orbit", dto.InstallPath)
		assert.Equal(t, "sha256:abc123", dto.Checksum)
		assert.False(t, dto.Enabled)
		mockRepo.AssertExpectations(t)
	})

	t.Run("lists packages with different types and enabled states", func(t *testing.T) {
		mockRepo := new(MockInstalledPackageRepository)
		handler := NewListInstalledHandler(mockRepo)

		userID := uuid.New()
		pkg1 := createTestInstalledPackage("test.orbit1", domain.PackageTypeOrbit, userID)
		pkg1.Enabled = true
		pkg2 := createTestInstalledPackage("test.orbit2", domain.PackageTypeOrbit, userID)
		pkg2.Enabled = false
		pkg3 := createTestInstalledPackage("test.engine", domain.PackageTypeEngine, userID)
		pkg3.Enabled = true
		packages := []*domain.InstalledPackage{pkg1, pkg2, pkg3}

		mockRepo.On("ListByUser", mock.Anything, userID).Return(packages, nil)

		query := ListInstalledQuery{
			UserID: userID,
		}

		result, err := handler.Handle(context.Background(), query)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Packages, 3)
		assert.Equal(t, 3, result.Total)

		// Verify enabled states
		assert.True(t, result.Packages[0].Enabled)
		assert.False(t, result.Packages[1].Enabled)
		assert.True(t, result.Packages[2].Enabled)
		mockRepo.AssertExpectations(t)
	})
}

func TestNewListInstalledHandler(t *testing.T) {
	mockRepo := new(MockInstalledPackageRepository)

	handler := NewListInstalledHandler(mockRepo)

	assert.NotNil(t, handler)
}
