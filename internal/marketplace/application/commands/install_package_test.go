package commands

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
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

// mockPackageRepo is a mock implementation of domain.PackageRepository.
type mockPackageRepo struct {
	mock.Mock
}

func (m *mockPackageRepo) Create(ctx context.Context, pkg *domain.Package) error {
	args := m.Called(ctx, pkg)
	return args.Error(0)
}

func (m *mockPackageRepo) Update(ctx context.Context, pkg *domain.Package) error {
	args := m.Called(ctx, pkg)
	return args.Error(0)
}

func (m *mockPackageRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockPackageRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Package, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Package), args.Error(1)
}

func (m *mockPackageRepo) GetByPackageID(ctx context.Context, packageID string) (*domain.Package, error) {
	args := m.Called(ctx, packageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Package), args.Error(1)
}

func (m *mockPackageRepo) List(ctx context.Context, filter domain.PackageFilter) ([]*domain.Package, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.Package), args.Get(1).(int64), args.Error(2)
}

func (m *mockPackageRepo) Search(ctx context.Context, query string, filter domain.PackageFilter) ([]*domain.Package, int64, error) {
	args := m.Called(ctx, query, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.Package), args.Get(1).(int64), args.Error(2)
}

func (m *mockPackageRepo) GetFeatured(ctx context.Context, limit int) ([]*domain.Package, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Package), args.Error(1)
}

func (m *mockPackageRepo) GetByPublisher(ctx context.Context, publisherID uuid.UUID, filter domain.PackageFilter) ([]*domain.Package, int64, error) {
	args := m.Called(ctx, publisherID, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.Package), args.Get(1).(int64), args.Error(2)
}

func (m *mockPackageRepo) IncrementDownloads(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// mockVersionRepo is a mock implementation of domain.VersionRepository.
type mockVersionRepo struct {
	mock.Mock
}

func (m *mockVersionRepo) Create(ctx context.Context, version *domain.Version) error {
	args := m.Called(ctx, version)
	return args.Error(0)
}

func (m *mockVersionRepo) Update(ctx context.Context, version *domain.Version) error {
	args := m.Called(ctx, version)
	return args.Error(0)
}

func (m *mockVersionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockVersionRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Version, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Version), args.Error(1)
}

func (m *mockVersionRepo) GetByPackageAndVersion(ctx context.Context, packageID uuid.UUID, version string) (*domain.Version, error) {
	args := m.Called(ctx, packageID, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Version), args.Error(1)
}

func (m *mockVersionRepo) ListByPackage(ctx context.Context, packageID uuid.UUID) ([]*domain.Version, error) {
	args := m.Called(ctx, packageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Version), args.Error(1)
}

func (m *mockVersionRepo) GetLatestStable(ctx context.Context, packageID uuid.UUID) (*domain.Version, error) {
	args := m.Called(ctx, packageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Version), args.Error(1)
}

func (m *mockVersionRepo) IncrementDownloads(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func createTestPackage(packageID string, pkgType domain.PackageType) *domain.Package {
	now := time.Now()
	return &domain.Package{
		ID:            uuid.New(),
		PackageID:     packageID,
		Type:          pkgType,
		Name:          "Test Package",
		Description:   "A test package",
		LatestVersion: "1.0.0",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func createTestVersion(packageID uuid.UUID, version string) *domain.Version {
	now := time.Now()
	return &domain.Version{
		ID:          uuid.New(),
		PackageID:   packageID,
		Version:     version,
		Checksum:    "",
		DownloadURL: "",
		PublishedAt: now,
		CreatedAt:   now,
	}
}

// createTestTarGz creates a test tar.gz archive for testing extraction.
func createTestTarGz(t *testing.T, destPath string, files map[string]string) string {
	t.Helper()

	f, err := os.Create(destPath)
	require.NoError(t, err)
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	for name, content := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}
		err := tw.WriteHeader(header)
		require.NoError(t, err)

		_, err = tw.Write([]byte(content))
		require.NoError(t, err)
	}

	return destPath
}

// testCalculateChecksum calculates SHA256 checksum of a file for testing.
func testCalculateChecksum(t *testing.T, filePath string) string {
	t.Helper()

	checksum, err := calculateChecksum(filePath)
	require.NoError(t, err)

	return checksum
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return data
}

// setupTestServer creates an HTTP test server serving a tar.gz archive.
func setupTestServer(t *testing.T, archivePath string) *httptest.Server {
	t.Helper()
	archiveData := mustReadFile(t, archivePath)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(archiveData)
	}))
}

// createTestArchive creates a test tar.gz archive and returns its path and checksum.
func createTestArchive(t *testing.T) (archivePath, checksum string, cleanup func()) {
	t.Helper()

	archiveDir, err := os.MkdirTemp("", "test-archive-*")
	require.NoError(t, err)

	archivePath = filepath.Join(archiveDir, "test.tar.gz")
	createTestTarGz(t, archivePath, map[string]string{
		"README.md":  "# Test Package",
		"config.yml": "name: test",
	})

	checksum = testCalculateChecksum(t, archivePath)

	return archivePath, checksum, func() { os.RemoveAll(archiveDir) }
}

func TestInstallPackageHandler_Handle(t *testing.T) {
	t.Run("successfully installs package from server", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		installedRepo := new(mockInstalledPackageRepo)

		tmpDir, err := os.MkdirTemp("", "test-install-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		archivePath, checksum, cleanup := createTestArchive(t)
		defer cleanup()

		server := setupTestServer(t, archivePath)
		defer server.Close()

		handler := NewInstallPackageHandler(packageRepo, versionRepo, installedRepo, tmpDir)

		userID := uuid.New()
		packageID := "acme.test-orbit"
		pkg := createTestPackage(packageID, domain.PackageTypeOrbit)
		version := createTestVersion(pkg.ID, "1.0.0")
		version.DownloadURL = server.URL + "/package.tar.gz"
		version.Checksum = "sha256:" + checksum

		installedRepo.On("GetByPackageID", mock.Anything, packageID, userID).Return(nil, nil)
		packageRepo.On("GetByPackageID", mock.Anything, packageID).Return(pkg, nil)
		versionRepo.On("GetByPackageAndVersion", mock.Anything, pkg.ID, "1.0.0").Return(version, nil)
		installedRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.InstalledPackage")).Return(nil)
		packageRepo.On("IncrementDownloads", mock.Anything, pkg.ID).Return(nil)

		cmd := InstallPackageCommand{
			PackageID: packageID,
			UserID:    userID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, packageID, result.InstalledPackage.PackageID)
		assert.Equal(t, "1.0.0", result.InstalledPackage.Version)
		assert.Contains(t, result.Message, "Successfully installed")

		packageRepo.AssertExpectations(t)
		versionRepo.AssertExpectations(t)
		installedRepo.AssertExpectations(t)
	})

	t.Run("successfully installs package with specific version", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		installedRepo := new(mockInstalledPackageRepo)

		tmpDir, err := os.MkdirTemp("", "test-install-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		archivePath, checksum, cleanup := createTestArchive(t)
		defer cleanup()

		server := setupTestServer(t, archivePath)
		defer server.Close()

		handler := NewInstallPackageHandler(packageRepo, versionRepo, installedRepo, tmpDir)

		userID := uuid.New()
		packageID := "acme.test-orbit"
		pkg := createTestPackage(packageID, domain.PackageTypeOrbit)
		pkg.LatestVersion = "2.0.0"
		version := createTestVersion(pkg.ID, "1.5.0")
		version.DownloadURL = server.URL + "/package.tar.gz"
		version.Checksum = "sha256:" + checksum

		installedRepo.On("GetByPackageID", mock.Anything, packageID, userID).Return(nil, nil)
		packageRepo.On("GetByPackageID", mock.Anything, packageID).Return(pkg, nil)
		versionRepo.On("GetByPackageAndVersion", mock.Anything, pkg.ID, "1.5.0").Return(version, nil)
		installedRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.InstalledPackage")).Return(nil)
		packageRepo.On("IncrementDownloads", mock.Anything, pkg.ID).Return(nil)

		cmd := InstallPackageCommand{
			PackageID: packageID,
			Version:   "1.5.0",
			UserID:    userID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "1.5.0", result.InstalledPackage.Version)
		assert.Contains(t, result.Message, "1.5.0")

		packageRepo.AssertExpectations(t)
		versionRepo.AssertExpectations(t)
		installedRepo.AssertExpectations(t)
	})

	t.Run("returns ErrPackageAlreadyInstalled when already installed", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		installedRepo := new(mockInstalledPackageRepo)

		handler := NewInstallPackageHandler(packageRepo, versionRepo, installedRepo, "/tmp")

		userID := uuid.New()
		packageID := "acme.test-orbit"
		existing := createTestInstalledPackage(packageID, "1.0.0", userID)

		installedRepo.On("GetByPackageID", mock.Anything, packageID, userID).Return(existing, nil)

		cmd := InstallPackageCommand{
			PackageID: packageID,
			UserID:    userID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.ErrorIs(t, err, ErrPackageAlreadyInstalled)
		assert.Nil(t, result)

		installedRepo.AssertExpectations(t)
	})

	t.Run("returns ErrPackageNotFound when package not in marketplace", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		installedRepo := new(mockInstalledPackageRepo)

		handler := NewInstallPackageHandler(packageRepo, versionRepo, installedRepo, "/tmp")

		userID := uuid.New()
		packageID := "unknown.package"

		installedRepo.On("GetByPackageID", mock.Anything, packageID, userID).Return(nil, nil)
		packageRepo.On("GetByPackageID", mock.Anything, packageID).Return(nil, errors.New("not found"))

		cmd := InstallPackageCommand{
			PackageID: packageID,
			UserID:    userID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.ErrorIs(t, err, ErrPackageNotFound)
		assert.Nil(t, result)

		installedRepo.AssertExpectations(t)
		packageRepo.AssertExpectations(t)
	})

	t.Run("returns ErrVersionNotFound when version not found", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		installedRepo := new(mockInstalledPackageRepo)

		handler := NewInstallPackageHandler(packageRepo, versionRepo, installedRepo, "/tmp")

		userID := uuid.New()
		packageID := "acme.test-orbit"
		pkg := createTestPackage(packageID, domain.PackageTypeOrbit)

		installedRepo.On("GetByPackageID", mock.Anything, packageID, userID).Return(nil, nil)
		packageRepo.On("GetByPackageID", mock.Anything, packageID).Return(pkg, nil)
		versionRepo.On("GetByPackageAndVersion", mock.Anything, pkg.ID, "1.0.0").Return(nil, errors.New("not found"))

		cmd := InstallPackageCommand{
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

	t.Run("fails on checksum mismatch", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		installedRepo := new(mockInstalledPackageRepo)

		tmpDir, err := os.MkdirTemp("", "test-install-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Create a test tar.gz archive
		archiveDir, err := os.MkdirTemp("", "test-archive-*")
		require.NoError(t, err)
		defer os.RemoveAll(archiveDir)

		archivePath := filepath.Join(archiveDir, "test.tar.gz")
		createTestTarGz(t, archivePath, map[string]string{
			"README.md": "# Test Package",
		})

		archiveData := mustReadFile(t, archivePath)

		// Setup HTTP test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write(archiveData)
		}))
		defer server.Close()

		handler := NewInstallPackageHandler(packageRepo, versionRepo, installedRepo, tmpDir)

		userID := uuid.New()
		packageID := "acme.test-orbit"
		pkg := createTestPackage(packageID, domain.PackageTypeOrbit)
		version := createTestVersion(pkg.ID, "1.0.0")
		version.DownloadURL = server.URL + "/package.tar.gz"
		version.Checksum = "sha256:invalidchecksum"

		installedRepo.On("GetByPackageID", mock.Anything, packageID, userID).Return(nil, nil)
		packageRepo.On("GetByPackageID", mock.Anything, packageID).Return(pkg, nil)
		versionRepo.On("GetByPackageAndVersion", mock.Anything, pkg.ID, "1.0.0").Return(version, nil)

		cmd := InstallPackageCommand{
			PackageID: packageID,
			UserID:    userID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrChecksumMismatch)
		assert.Nil(t, result)

		// Verify install directory was cleaned up
		installPath := filepath.Join(tmpDir, "orbits", packageID, "1.0.0")
		_, statErr := os.Stat(installPath)
		assert.True(t, os.IsNotExist(statErr), "install directory should be cleaned up on failure")

		installedRepo.AssertExpectations(t)
		packageRepo.AssertExpectations(t)
		versionRepo.AssertExpectations(t)
	})

	t.Run("fails when download returns non-200 status", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		installedRepo := new(mockInstalledPackageRepo)

		tmpDir, err := os.MkdirTemp("", "test-install-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Setup HTTP test server returning 404
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		handler := NewInstallPackageHandler(packageRepo, versionRepo, installedRepo, tmpDir)

		userID := uuid.New()
		packageID := "acme.test-orbit"
		pkg := createTestPackage(packageID, domain.PackageTypeOrbit)
		version := createTestVersion(pkg.ID, "1.0.0")
		version.DownloadURL = server.URL + "/package.tar.gz"

		installedRepo.On("GetByPackageID", mock.Anything, packageID, userID).Return(nil, nil)
		packageRepo.On("GetByPackageID", mock.Anything, packageID).Return(pkg, nil)
		versionRepo.On("GetByPackageAndVersion", mock.Anything, pkg.ID, "1.0.0").Return(version, nil)

		cmd := InstallPackageCommand{
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

	t.Run("fails when create installation record fails", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		installedRepo := new(mockInstalledPackageRepo)

		tmpDir, err := os.MkdirTemp("", "test-install-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		archivePath, _, cleanup := createTestArchive(t)
		defer cleanup()

		server := setupTestServer(t, archivePath)
		defer server.Close()

		handler := NewInstallPackageHandler(packageRepo, versionRepo, installedRepo, tmpDir)

		userID := uuid.New()
		packageID := "acme.test-orbit"
		pkg := createTestPackage(packageID, domain.PackageTypeOrbit)
		version := createTestVersion(pkg.ID, "1.0.0")
		version.DownloadURL = server.URL + "/package.tar.gz"

		installedRepo.On("GetByPackageID", mock.Anything, packageID, userID).Return(nil, nil)
		packageRepo.On("GetByPackageID", mock.Anything, packageID).Return(pkg, nil)
		versionRepo.On("GetByPackageAndVersion", mock.Anything, pkg.ID, "1.0.0").Return(version, nil)
		installedRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.InstalledPackage")).Return(errors.New("database error"))

		cmd := InstallPackageCommand{
			PackageID: packageID,
			UserID:    userID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save installation record")
		assert.Nil(t, result)

		// Verify install directory was cleaned up
		installPath := filepath.Join(tmpDir, "orbits", packageID, "1.0.0")
		_, statErr := os.Stat(installPath)
		assert.True(t, os.IsNotExist(statErr), "install directory should be cleaned up on failure")

		installedRepo.AssertExpectations(t)
		packageRepo.AssertExpectations(t)
		versionRepo.AssertExpectations(t)
	})

	t.Run("installs engine package with correct path", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		installedRepo := new(mockInstalledPackageRepo)

		tmpDir, err := os.MkdirTemp("", "test-install-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		archivePath, checksum, cleanup := createTestArchive(t)
		defer cleanup()

		server := setupTestServer(t, archivePath)
		defer server.Close()

		handler := NewInstallPackageHandler(packageRepo, versionRepo, installedRepo, tmpDir)

		userID := uuid.New()
		packageID := "acme.priority-engine"
		pkg := createTestPackage(packageID, domain.PackageTypeEngine)
		version := createTestVersion(pkg.ID, "1.0.0")
		version.DownloadURL = server.URL + "/package.tar.gz"
		version.Checksum = "sha256:" + checksum

		installedRepo.On("GetByPackageID", mock.Anything, packageID, userID).Return(nil, nil)
		packageRepo.On("GetByPackageID", mock.Anything, packageID).Return(pkg, nil)
		versionRepo.On("GetByPackageAndVersion", mock.Anything, pkg.ID, "1.0.0").Return(version, nil)
		installedRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.InstalledPackage")).Return(nil)
		packageRepo.On("IncrementDownloads", mock.Anything, pkg.ID).Return(nil)

		cmd := InstallPackageCommand{
			PackageID: packageID,
			UserID:    userID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, domain.PackageTypeEngine, result.InstalledPackage.Type)
		assert.Contains(t, result.InstalledPackage.InstallPath, "engines")

		packageRepo.AssertExpectations(t)
		versionRepo.AssertExpectations(t)
		installedRepo.AssertExpectations(t)
	})

	t.Run("continues even when IncrementDownloads fails", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		installedRepo := new(mockInstalledPackageRepo)

		tmpDir, err := os.MkdirTemp("", "test-install-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		archivePath, checksum, cleanup := createTestArchive(t)
		defer cleanup()

		server := setupTestServer(t, archivePath)
		defer server.Close()

		handler := NewInstallPackageHandler(packageRepo, versionRepo, installedRepo, tmpDir)

		userID := uuid.New()
		packageID := "acme.test-orbit"
		pkg := createTestPackage(packageID, domain.PackageTypeOrbit)
		version := createTestVersion(pkg.ID, "1.0.0")
		version.DownloadURL = server.URL + "/package.tar.gz"
		version.Checksum = "sha256:" + checksum

		installedRepo.On("GetByPackageID", mock.Anything, packageID, userID).Return(nil, nil)
		packageRepo.On("GetByPackageID", mock.Anything, packageID).Return(pkg, nil)
		versionRepo.On("GetByPackageAndVersion", mock.Anything, pkg.ID, "1.0.0").Return(version, nil)
		installedRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.InstalledPackage")).Return(nil)
		packageRepo.On("IncrementDownloads", mock.Anything, pkg.ID).Return(errors.New("increment error"))

		cmd := InstallPackageCommand{
			PackageID: packageID,
			UserID:    userID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		// Should still succeed despite IncrementDownloads error
		require.NoError(t, err)
		require.NotNil(t, result)

		packageRepo.AssertExpectations(t)
		versionRepo.AssertExpectations(t)
		installedRepo.AssertExpectations(t)
	})
}

func TestInstallPackageHandler_verifyChecksum(t *testing.T) {
	t.Run("verifies checksum with sha256 prefix", func(t *testing.T) {
		handler := &InstallPackageHandler{}

		tmpDir, err := os.MkdirTemp("", "test-checksum-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		testFile := filepath.Join(tmpDir, "test.txt")
		content := []byte("test content for checksum")
		err = os.WriteFile(testFile, content, 0644)
		require.NoError(t, err)

		// Calculate expected checksum
		h := sha256.New()
		h.Write(content)
		expectedChecksum := "sha256:" + hex.EncodeToString(h.Sum(nil))

		err = handler.verifyChecksum(testFile, expectedChecksum)
		assert.NoError(t, err)
	})

	t.Run("verifies checksum without sha256 prefix", func(t *testing.T) {
		handler := &InstallPackageHandler{}

		tmpDir, err := os.MkdirTemp("", "test-checksum-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		testFile := filepath.Join(tmpDir, "test.txt")
		content := []byte("test content for checksum")
		err = os.WriteFile(testFile, content, 0644)
		require.NoError(t, err)

		// Calculate expected checksum without prefix
		h := sha256.New()
		h.Write(content)
		expectedChecksum := hex.EncodeToString(h.Sum(nil))

		err = handler.verifyChecksum(testFile, expectedChecksum)
		assert.NoError(t, err)
	})

	t.Run("returns error on checksum mismatch", func(t *testing.T) {
		handler := &InstallPackageHandler{}

		tmpDir, err := os.MkdirTemp("", "test-checksum-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		testFile := filepath.Join(tmpDir, "test.txt")
		err = os.WriteFile(testFile, []byte("test content"), 0644)
		require.NoError(t, err)

		err = handler.verifyChecksum(testFile, "sha256:invalidchecksum")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrChecksumMismatch)
	})

	t.Run("returns error when file does not exist", func(t *testing.T) {
		handler := &InstallPackageHandler{}

		err := handler.verifyChecksum("/nonexistent/file.txt", "sha256:abc123")
		assert.Error(t, err)
	})
}

func TestInstallPackageHandler_extractPackage(t *testing.T) {
	t.Run("successfully extracts tar.gz archive", func(t *testing.T) {
		handler := &InstallPackageHandler{}

		tmpDir, err := os.MkdirTemp("", "test-extract-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		archivePath := filepath.Join(tmpDir, "archive.tar.gz")
		createTestTarGz(t, archivePath, map[string]string{
			"file1.txt":        "content1",
			"subdir/file2.txt": "content2",
		})

		destDir := filepath.Join(tmpDir, "extracted")
		err = os.MkdirAll(destDir, 0755)
		require.NoError(t, err)

		err = handler.extractPackage(archivePath, destDir)
		require.NoError(t, err)

		// Verify files were extracted
		content1, err := os.ReadFile(filepath.Join(destDir, "file1.txt"))
		require.NoError(t, err)
		assert.Equal(t, "content1", string(content1))

		content2, err := os.ReadFile(filepath.Join(destDir, "subdir/file2.txt"))
		require.NoError(t, err)
		assert.Equal(t, "content2", string(content2))
	})

	t.Run("returns error for non-existent archive", func(t *testing.T) {
		handler := &InstallPackageHandler{}

		err := handler.extractPackage("/nonexistent/archive.tar.gz", "/tmp")
		assert.Error(t, err)
	})

	t.Run("returns error for invalid gzip file", func(t *testing.T) {
		handler := &InstallPackageHandler{}

		tmpDir, err := os.MkdirTemp("", "test-extract-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		invalidFile := filepath.Join(tmpDir, "invalid.tar.gz")
		err = os.WriteFile(invalidFile, []byte("not a gzip file"), 0644)
		require.NoError(t, err)

		err = handler.extractPackage(invalidFile, tmpDir)
		assert.Error(t, err)
	})
}

func TestNewInstallPackageHandler(t *testing.T) {
	packageRepo := new(mockPackageRepo)
	versionRepo := new(mockVersionRepo)
	installedRepo := new(mockInstalledPackageRepo)

	handler := NewInstallPackageHandler(packageRepo, versionRepo, installedRepo, "/test/dir")

	require.NotNil(t, handler)
	assert.Equal(t, "/test/dir", handler.installDir)
}
