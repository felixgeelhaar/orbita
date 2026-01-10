package commands

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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

// Test helper to create an existing installed package directory
func createInstalledPackageDir(t *testing.T, baseDir, packageID, version string) string {
	t.Helper()

	installPath := filepath.Join(baseDir, "orbits", packageID, version)
	err := os.MkdirAll(installPath, 0755)
	require.NoError(t, err)

	// Create dummy files
	err = os.WriteFile(filepath.Join(installPath, "README.md"), []byte("# Old Version"), 0644)
	require.NoError(t, err)

	return installPath
}

func TestUpdatePackageHandler_Handle(t *testing.T) {
	t.Run("successfully updates package to latest version", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		installedRepo := new(mockInstalledPackageRepo)

		tmpDir, err := os.MkdirTemp("", "test-update-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		archivePath, checksum, cleanup := createTestArchive(t)
		defer cleanup()

		server := setupTestServer(t, archivePath)
		defer server.Close()

		handler := NewUpdatePackageHandler(packageRepo, versionRepo, installedRepo, tmpDir)

		userID := uuid.New()
		packageID := "acme.test-orbit"

		// Create existing installation
		installPath := createInstalledPackageDir(t, tmpDir, packageID, "1.0.0")

		installed := &domain.InstalledPackage{
			ID:          uuid.New(),
			PackageID:   packageID,
			Version:     "1.0.0",
			Type:        domain.PackageTypeOrbit,
			InstallPath: installPath,
			Checksum:    "sha256:oldchecksum",
			InstalledAt: time.Now(),
			UpdatedAt:   time.Now(),
			Enabled:     true,
			UserID:      userID,
		}

		pkg := createTestPackage(packageID, domain.PackageTypeOrbit)
		pkg.LatestVersion = "2.0.0"

		newVersion := &domain.Version{
			ID:          uuid.New(),
			PackageID:   pkg.ID,
			Version:     "2.0.0",
			Checksum:    "sha256:" + checksum,
			DownloadURL: server.URL + "/package.tar.gz",
		}

		installedRepo.On("GetByPackageID", mock.Anything, packageID, userID).Return(installed, nil)
		packageRepo.On("GetByPackageID", mock.Anything, packageID).Return(pkg, nil)
		versionRepo.On("GetByPackageAndVersion", mock.Anything, pkg.ID, "2.0.0").Return(newVersion, nil)
		installedRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.InstalledPackage")).Return(nil)
		packageRepo.On("IncrementDownloads", mock.Anything, pkg.ID).Return(nil)

		cmd := UpdatePackageCommand{
			PackageID: packageID,
			UserID:    userID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "1.0.0", result.OldVersion)
		assert.Equal(t, "2.0.0", result.NewVersion)
		assert.Contains(t, result.Message, "Successfully updated")

		installedRepo.AssertExpectations(t)
		packageRepo.AssertExpectations(t)
		versionRepo.AssertExpectations(t)
	})

	t.Run("successfully updates to specific version", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		installedRepo := new(mockInstalledPackageRepo)

		tmpDir, err := os.MkdirTemp("", "test-update-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		archivePath, checksum, cleanup := createTestArchive(t)
		defer cleanup()

		server := setupTestServer(t, archivePath)
		defer server.Close()

		handler := NewUpdatePackageHandler(packageRepo, versionRepo, installedRepo, tmpDir)

		userID := uuid.New()
		packageID := "acme.test-orbit"

		installPath := createInstalledPackageDir(t, tmpDir, packageID, "1.0.0")

		installed := &domain.InstalledPackage{
			ID:          uuid.New(),
			PackageID:   packageID,
			Version:     "1.0.0",
			Type:        domain.PackageTypeOrbit,
			InstallPath: installPath,
			InstalledAt: time.Now(),
			UpdatedAt:   time.Now(),
			Enabled:     true,
			UserID:      userID,
		}

		pkg := createTestPackage(packageID, domain.PackageTypeOrbit)
		pkg.LatestVersion = "3.0.0"

		targetVersion := &domain.Version{
			ID:          uuid.New(),
			PackageID:   pkg.ID,
			Version:     "2.5.0",
			Checksum:    "sha256:" + checksum,
			DownloadURL: server.URL + "/package.tar.gz",
		}

		installedRepo.On("GetByPackageID", mock.Anything, packageID, userID).Return(installed, nil)
		packageRepo.On("GetByPackageID", mock.Anything, packageID).Return(pkg, nil)
		versionRepo.On("GetByPackageAndVersion", mock.Anything, pkg.ID, "2.5.0").Return(targetVersion, nil)
		installedRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.InstalledPackage")).Return(nil)
		packageRepo.On("IncrementDownloads", mock.Anything, pkg.ID).Return(nil)

		cmd := UpdatePackageCommand{
			PackageID: packageID,
			Version:   "2.5.0",
			UserID:    userID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "1.0.0", result.OldVersion)
		assert.Equal(t, "2.5.0", result.NewVersion)

		installedRepo.AssertExpectations(t)
		packageRepo.AssertExpectations(t)
		versionRepo.AssertExpectations(t)
	})

	t.Run("returns already at version when already at target", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		installedRepo := new(mockInstalledPackageRepo)

		handler := NewUpdatePackageHandler(packageRepo, versionRepo, installedRepo, "/tmp")

		userID := uuid.New()
		packageID := "acme.test-orbit"

		installed := &domain.InstalledPackage{
			ID:          uuid.New(),
			PackageID:   packageID,
			Version:     "2.0.0",
			Type:        domain.PackageTypeOrbit,
			InstallPath: "/tmp/orbits/acme.test-orbit/2.0.0",
			InstalledAt: time.Now(),
			UpdatedAt:   time.Now(),
			Enabled:     true,
			UserID:      userID,
		}

		pkg := createTestPackage(packageID, domain.PackageTypeOrbit)
		pkg.LatestVersion = "2.0.0"

		installedRepo.On("GetByPackageID", mock.Anything, packageID, userID).Return(installed, nil)
		packageRepo.On("GetByPackageID", mock.Anything, packageID).Return(pkg, nil)

		cmd := UpdatePackageCommand{
			PackageID: packageID,
			UserID:    userID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "2.0.0", result.OldVersion)
		assert.Equal(t, "2.0.0", result.NewVersion)
		assert.Contains(t, result.Message, "already at version")

		installedRepo.AssertExpectations(t)
		packageRepo.AssertExpectations(t)
	})

	t.Run("returns ErrPackageNotInstalled when not installed", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		installedRepo := new(mockInstalledPackageRepo)

		handler := NewUpdatePackageHandler(packageRepo, versionRepo, installedRepo, "/tmp")

		userID := uuid.New()
		packageID := "acme.test-orbit"

		installedRepo.On("GetByPackageID", mock.Anything, packageID, userID).Return(nil, nil)

		cmd := UpdatePackageCommand{
			PackageID: packageID,
			UserID:    userID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.ErrorIs(t, err, ErrPackageNotInstalled)
		assert.Nil(t, result)

		installedRepo.AssertExpectations(t)
	})

	t.Run("returns ErrPackageNotInstalled when repo returns error", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		installedRepo := new(mockInstalledPackageRepo)

		handler := NewUpdatePackageHandler(packageRepo, versionRepo, installedRepo, "/tmp")

		userID := uuid.New()
		packageID := "acme.test-orbit"

		installedRepo.On("GetByPackageID", mock.Anything, packageID, userID).Return(nil, errors.New("database error"))

		cmd := UpdatePackageCommand{
			PackageID: packageID,
			UserID:    userID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.ErrorIs(t, err, ErrPackageNotInstalled)
		assert.Nil(t, result)

		installedRepo.AssertExpectations(t)
	})

	t.Run("returns ErrPackageNotFound when package not in marketplace", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		installedRepo := new(mockInstalledPackageRepo)

		handler := NewUpdatePackageHandler(packageRepo, versionRepo, installedRepo, "/tmp")

		userID := uuid.New()
		packageID := "acme.test-orbit"

		installed := &domain.InstalledPackage{
			ID:        uuid.New(),
			PackageID: packageID,
			Version:   "1.0.0",
			UserID:    userID,
		}

		installedRepo.On("GetByPackageID", mock.Anything, packageID, userID).Return(installed, nil)
		packageRepo.On("GetByPackageID", mock.Anything, packageID).Return(nil, errors.New("not found"))

		cmd := UpdatePackageCommand{
			PackageID: packageID,
			UserID:    userID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.ErrorIs(t, err, ErrPackageNotFound)
		assert.Nil(t, result)

		installedRepo.AssertExpectations(t)
		packageRepo.AssertExpectations(t)
	})

	t.Run("returns ErrVersionNotFound when target version not found", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		installedRepo := new(mockInstalledPackageRepo)

		handler := NewUpdatePackageHandler(packageRepo, versionRepo, installedRepo, "/tmp")

		userID := uuid.New()
		packageID := "acme.test-orbit"

		installed := &domain.InstalledPackage{
			ID:        uuid.New(),
			PackageID: packageID,
			Version:   "1.0.0",
			UserID:    userID,
		}

		pkg := createTestPackage(packageID, domain.PackageTypeOrbit)
		pkg.LatestVersion = "2.0.0"

		installedRepo.On("GetByPackageID", mock.Anything, packageID, userID).Return(installed, nil)
		packageRepo.On("GetByPackageID", mock.Anything, packageID).Return(pkg, nil)
		versionRepo.On("GetByPackageAndVersion", mock.Anything, pkg.ID, "2.0.0").Return(nil, errors.New("not found"))

		cmd := UpdatePackageCommand{
			PackageID: packageID,
			UserID:    userID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.ErrorIs(t, err, ErrVersionNotFound)
		assert.Nil(t, result)

		installedRepo.AssertExpectations(t)
		packageRepo.AssertExpectations(t)
		versionRepo.AssertExpectations(t)
	})

	t.Run("fails when download fails", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		installedRepo := new(mockInstalledPackageRepo)

		tmpDir, err := os.MkdirTemp("", "test-update-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Server that returns 404
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		handler := NewUpdatePackageHandler(packageRepo, versionRepo, installedRepo, tmpDir)

		userID := uuid.New()
		packageID := "acme.test-orbit"

		installPath := createInstalledPackageDir(t, tmpDir, packageID, "1.0.0")

		installed := &domain.InstalledPackage{
			ID:          uuid.New(),
			PackageID:   packageID,
			Version:     "1.0.0",
			Type:        domain.PackageTypeOrbit,
			InstallPath: installPath,
			UserID:      userID,
		}

		pkg := createTestPackage(packageID, domain.PackageTypeOrbit)
		pkg.LatestVersion = "2.0.0"

		newVersion := &domain.Version{
			ID:          uuid.New(),
			PackageID:   pkg.ID,
			Version:     "2.0.0",
			DownloadURL: server.URL + "/package.tar.gz",
		}

		installedRepo.On("GetByPackageID", mock.Anything, packageID, userID).Return(installed, nil)
		packageRepo.On("GetByPackageID", mock.Anything, packageID).Return(pkg, nil)
		versionRepo.On("GetByPackageAndVersion", mock.Anything, pkg.ID, "2.0.0").Return(newVersion, nil)

		cmd := UpdatePackageCommand{
			PackageID: packageID,
			UserID:    userID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "download")
		assert.Nil(t, result)

		installedRepo.AssertExpectations(t)
		packageRepo.AssertExpectations(t)
		versionRepo.AssertExpectations(t)
	})

	t.Run("fails on checksum mismatch", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		installedRepo := new(mockInstalledPackageRepo)

		tmpDir, err := os.MkdirTemp("", "test-update-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		archivePath, _, cleanup := createTestArchive(t)
		defer cleanup()

		server := setupTestServer(t, archivePath)
		defer server.Close()

		handler := NewUpdatePackageHandler(packageRepo, versionRepo, installedRepo, tmpDir)

		userID := uuid.New()
		packageID := "acme.test-orbit"

		installPath := createInstalledPackageDir(t, tmpDir, packageID, "1.0.0")

		installed := &domain.InstalledPackage{
			ID:          uuid.New(),
			PackageID:   packageID,
			Version:     "1.0.0",
			Type:        domain.PackageTypeOrbit,
			InstallPath: installPath,
			UserID:      userID,
		}

		pkg := createTestPackage(packageID, domain.PackageTypeOrbit)
		pkg.LatestVersion = "2.0.0"

		newVersion := &domain.Version{
			ID:          uuid.New(),
			PackageID:   pkg.ID,
			Version:     "2.0.0",
			Checksum:    "sha256:invalidchecksum",
			DownloadURL: server.URL + "/package.tar.gz",
		}

		installedRepo.On("GetByPackageID", mock.Anything, packageID, userID).Return(installed, nil)
		packageRepo.On("GetByPackageID", mock.Anything, packageID).Return(pkg, nil)
		versionRepo.On("GetByPackageAndVersion", mock.Anything, pkg.ID, "2.0.0").Return(newVersion, nil)

		cmd := UpdatePackageCommand{
			PackageID: packageID,
			UserID:    userID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrChecksumMismatch)
		assert.Nil(t, result)

		installedRepo.AssertExpectations(t)
		packageRepo.AssertExpectations(t)
		versionRepo.AssertExpectations(t)
	})

	t.Run("fails when update record fails", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		installedRepo := new(mockInstalledPackageRepo)

		tmpDir, err := os.MkdirTemp("", "test-update-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		archivePath, checksum, cleanup := createTestArchive(t)
		defer cleanup()

		server := setupTestServer(t, archivePath)
		defer server.Close()

		handler := NewUpdatePackageHandler(packageRepo, versionRepo, installedRepo, tmpDir)

		userID := uuid.New()
		packageID := "acme.test-orbit"

		installPath := createInstalledPackageDir(t, tmpDir, packageID, "1.0.0")

		installed := &domain.InstalledPackage{
			ID:          uuid.New(),
			PackageID:   packageID,
			Version:     "1.0.0",
			Type:        domain.PackageTypeOrbit,
			InstallPath: installPath,
			UserID:      userID,
		}

		pkg := createTestPackage(packageID, domain.PackageTypeOrbit)
		pkg.LatestVersion = "2.0.0"

		newVersion := &domain.Version{
			ID:          uuid.New(),
			PackageID:   pkg.ID,
			Version:     "2.0.0",
			Checksum:    "sha256:" + checksum,
			DownloadURL: server.URL + "/package.tar.gz",
		}

		installedRepo.On("GetByPackageID", mock.Anything, packageID, userID).Return(installed, nil)
		packageRepo.On("GetByPackageID", mock.Anything, packageID).Return(pkg, nil)
		versionRepo.On("GetByPackageAndVersion", mock.Anything, pkg.ID, "2.0.0").Return(newVersion, nil)
		installedRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.InstalledPackage")).Return(errors.New("database error"))

		cmd := UpdatePackageCommand{
			PackageID: packageID,
			UserID:    userID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update installation record")
		assert.Nil(t, result)

		installedRepo.AssertExpectations(t)
		packageRepo.AssertExpectations(t)
		versionRepo.AssertExpectations(t)
	})
}

func TestNewUpdatePackageHandler(t *testing.T) {
	packageRepo := new(mockPackageRepo)
	versionRepo := new(mockVersionRepo)
	installedRepo := new(mockInstalledPackageRepo)

	handler := NewUpdatePackageHandler(packageRepo, versionRepo, installedRepo, "/test/dir")

	require.NotNil(t, handler)
}
