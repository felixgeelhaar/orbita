package commands

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/marketplace/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockInstalledPackageRepo is a mock implementation of domain.InstalledPackageRepository.
type mockInstalledPackageRepo struct {
	mock.Mock
}

func (m *mockInstalledPackageRepo) Create(ctx context.Context, pkg *domain.InstalledPackage) error {
	args := m.Called(ctx, pkg)
	return args.Error(0)
}

func (m *mockInstalledPackageRepo) Update(ctx context.Context, pkg *domain.InstalledPackage) error {
	args := m.Called(ctx, pkg)
	return args.Error(0)
}

func (m *mockInstalledPackageRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockInstalledPackageRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.InstalledPackage, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.InstalledPackage), args.Error(1)
}

func (m *mockInstalledPackageRepo) GetByPackageID(ctx context.Context, packageID string, userID uuid.UUID) (*domain.InstalledPackage, error) {
	args := m.Called(ctx, packageID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.InstalledPackage), args.Error(1)
}

func (m *mockInstalledPackageRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.InstalledPackage, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.InstalledPackage), args.Error(1)
}

func (m *mockInstalledPackageRepo) ListByType(ctx context.Context, userID uuid.UUID, pkgType domain.PackageType) ([]*domain.InstalledPackage, error) {
	args := m.Called(ctx, userID, pkgType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.InstalledPackage), args.Error(1)
}

func createTestInstalledPackage(packageID, version string, userID uuid.UUID) *domain.InstalledPackage {
	now := time.Now()
	return &domain.InstalledPackage{
		ID:          uuid.New(),
		PackageID:   packageID,
		Version:     version,
		Type:        domain.PackageTypeOrbit,
		InstallPath: "",
		Checksum:    "sha256:abc123",
		InstalledAt: now,
		UpdatedAt:   now,
		Enabled:     true,
		UserID:      userID,
	}
}

func TestUninstallPackageHandler_Handle(t *testing.T) {
	userID := uuid.New()
	packageID := "acme.test-orbit"

	t.Run("successfully uninstalls a package without install path", func(t *testing.T) {
		installedRepo := new(mockInstalledPackageRepo)
		handler := NewUninstallPackageHandler(installedRepo)

		installed := createTestInstalledPackage(packageID, "1.0.0", userID)

		installedRepo.On("GetByPackageID", mock.Anything, packageID, userID).Return(installed, nil)
		installedRepo.On("Delete", mock.Anything, installed.ID).Return(nil)

		cmd := UninstallPackageCommand{
			PackageID: packageID,
			UserID:    userID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, packageID, result.PackageID)
		assert.Equal(t, "1.0.0", result.Version)
		assert.Contains(t, result.Message, "Successfully uninstalled")

		installedRepo.AssertExpectations(t)
	})

	t.Run("successfully uninstalls a package with install path", func(t *testing.T) {
		installedRepo := new(mockInstalledPackageRepo)
		handler := NewUninstallPackageHandler(installedRepo)

		// Create a temporary directory to simulate install path
		tmpDir, err := os.MkdirTemp("", "test-uninstall-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Create a test file in the directory
		testFile := filepath.Join(tmpDir, "test.txt")
		err = os.WriteFile(testFile, []byte("test"), 0644)
		require.NoError(t, err)

		installed := createTestInstalledPackage(packageID, "1.0.0", userID)
		installed.InstallPath = tmpDir

		installedRepo.On("GetByPackageID", mock.Anything, packageID, userID).Return(installed, nil)
		installedRepo.On("Delete", mock.Anything, installed.ID).Return(nil)

		cmd := UninstallPackageCommand{
			PackageID: packageID,
			UserID:    userID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, packageID, result.PackageID)

		// Verify directory was removed
		_, statErr := os.Stat(tmpDir)
		assert.True(t, os.IsNotExist(statErr))

		installedRepo.AssertExpectations(t)
	})

	t.Run("returns ErrPackageNotInstalled when package not found", func(t *testing.T) {
		installedRepo := new(mockInstalledPackageRepo)
		handler := NewUninstallPackageHandler(installedRepo)

		installedRepo.On("GetByPackageID", mock.Anything, packageID, userID).Return(nil, nil)

		cmd := UninstallPackageCommand{
			PackageID: packageID,
			UserID:    userID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.ErrorIs(t, err, ErrPackageNotInstalled)
		assert.Nil(t, result)

		installedRepo.AssertExpectations(t)
	})

	t.Run("returns ErrPackageNotInstalled when repository returns error", func(t *testing.T) {
		installedRepo := new(mockInstalledPackageRepo)
		handler := NewUninstallPackageHandler(installedRepo)

		installedRepo.On("GetByPackageID", mock.Anything, packageID, userID).Return(nil, errors.New("database error"))

		cmd := UninstallPackageCommand{
			PackageID: packageID,
			UserID:    userID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.ErrorIs(t, err, ErrPackageNotInstalled)
		assert.Nil(t, result)

		installedRepo.AssertExpectations(t)
	})

	t.Run("fails when delete fails", func(t *testing.T) {
		installedRepo := new(mockInstalledPackageRepo)
		handler := NewUninstallPackageHandler(installedRepo)

		installed := createTestInstalledPackage(packageID, "1.0.0", userID)

		installedRepo.On("GetByPackageID", mock.Anything, packageID, userID).Return(installed, nil)
		installedRepo.On("Delete", mock.Anything, installed.ID).Return(errors.New("delete error"))

		cmd := UninstallPackageCommand{
			PackageID: packageID,
			UserID:    userID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to remove installation record")
		assert.Nil(t, result)

		installedRepo.AssertExpectations(t)
	})
}

func TestNewUninstallPackageHandler(t *testing.T) {
	installedRepo := new(mockInstalledPackageRepo)

	handler := NewUninstallPackageHandler(installedRepo)

	require.NotNil(t, handler)
}
