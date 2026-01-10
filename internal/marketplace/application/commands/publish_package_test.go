package commands

import (
	"context"
	"encoding/json"
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

// mockPublisherRepo is defined in auth_test.go

func createTestPublisher(id uuid.UUID) *domain.Publisher {
	now := time.Now()
	return &domain.Publisher{
		ID:           id,
		Name:         "Test Publisher",
		Slug:         "test-publisher",
		Email:        "test@example.com",
		PackageCount: 0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// createTestPackageDir creates a temporary directory with a manifest file.
func createTestPackageDir(t *testing.T, manifest PackageManifest) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "test-package-*")
	require.NoError(t, err)

	manifestFile := "orbit.json"
	if manifest.Type == "engine" {
		manifestFile = "engine.json"
	}

	data, err := json.Marshal(manifest)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, manifestFile), data, 0644)
	require.NoError(t, err)

	// Add a dummy README
	err = os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Test Package"), 0644)
	require.NoError(t, err)

	return tmpDir, func() { os.RemoveAll(tmpDir) }
}

func TestPublishPackageHandler_Handle(t *testing.T) {
	t.Run("successfully publishes new package", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		publisherRepo := new(mockPublisherRepo)
		handler := NewPublishPackageHandler(packageRepo, versionRepo, publisherRepo)

		publisherID := uuid.New()
		publisher := createTestPublisher(publisherID)

		manifest := PackageManifest{
			ID:          "acme.test-orbit",
			Name:        "Test Orbit",
			Version:     "1.0.0",
			Type:        "orbit",
			Description: "A test orbit package",
		}

		packageDir, cleanup := createTestPackageDir(t, manifest)
		defer cleanup()

		publisherRepo.On("GetByID", mock.Anything, publisherID).Return(publisher, nil)
		packageRepo.On("GetByPackageID", mock.Anything, manifest.ID).Return(nil, errors.New("not found"))
		packageRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Package")).Return(nil)
		publisherRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Publisher")).Return(nil)
		versionRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Version")).Return(nil)

		cmd := PublishPackageCommand{
			PackagePath: packageDir,
			PublisherID: publisherID,
			DryRun:      false,
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, manifest.ID, result.PackageID)
		assert.Equal(t, manifest.Version, result.Version)
		assert.NotEmpty(t, result.Checksum)
		assert.False(t, result.DryRun)
		assert.Contains(t, result.Message, "Successfully published")

		publisherRepo.AssertExpectations(t)
		packageRepo.AssertExpectations(t)
		versionRepo.AssertExpectations(t)
	})

	t.Run("successfully publishes engine package", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		publisherRepo := new(mockPublisherRepo)
		handler := NewPublishPackageHandler(packageRepo, versionRepo, publisherRepo)

		publisherID := uuid.New()
		publisher := createTestPublisher(publisherID)

		manifest := PackageManifest{
			ID:          "acme.priority-engine",
			Name:        "Priority Engine",
			Version:     "1.0.0",
			Type:        "engine",
			Description: "A test engine package",
		}

		packageDir, cleanup := createTestPackageDir(t, manifest)
		defer cleanup()

		publisherRepo.On("GetByID", mock.Anything, publisherID).Return(publisher, nil)
		packageRepo.On("GetByPackageID", mock.Anything, manifest.ID).Return(nil, errors.New("not found"))
		packageRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Package")).Return(nil)
		publisherRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Publisher")).Return(nil)
		versionRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Version")).Return(nil)

		cmd := PublishPackageCommand{
			PackagePath: packageDir,
			PublisherID: publisherID,
			DryRun:      false,
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, manifest.ID, result.PackageID)

		publisherRepo.AssertExpectations(t)
		packageRepo.AssertExpectations(t)
		versionRepo.AssertExpectations(t)
	})

	t.Run("successfully publishes new version of existing package", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		publisherRepo := new(mockPublisherRepo)
		handler := NewPublishPackageHandler(packageRepo, versionRepo, publisherRepo)

		publisherID := uuid.New()
		publisher := createTestPublisher(publisherID)

		existingPkg := createTestPackage("acme.test-orbit", domain.PackageTypeOrbit)
		existingPkg.PublisherID = publisherID

		manifest := PackageManifest{
			ID:      "acme.test-orbit",
			Name:    "Test Orbit",
			Version: "2.0.0",
			Type:    "orbit",
		}

		packageDir, cleanup := createTestPackageDir(t, manifest)
		defer cleanup()

		publisherRepo.On("GetByID", mock.Anything, publisherID).Return(publisher, nil)
		packageRepo.On("GetByPackageID", mock.Anything, manifest.ID).Return(existingPkg, nil)
		versionRepo.On("GetByPackageAndVersion", mock.Anything, existingPkg.ID, manifest.Version).Return(nil, errors.New("not found"))
		versionRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Version")).Return(nil)
		packageRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Package")).Return(nil)

		cmd := PublishPackageCommand{
			PackagePath: packageDir,
			PublisherID: publisherID,
			DryRun:      false,
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "2.0.0", result.Version)

		publisherRepo.AssertExpectations(t)
		packageRepo.AssertExpectations(t)
		versionRepo.AssertExpectations(t)
	})

	t.Run("dry run returns success without creating records", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		publisherRepo := new(mockPublisherRepo)
		handler := NewPublishPackageHandler(packageRepo, versionRepo, publisherRepo)

		publisherID := uuid.New()
		publisher := createTestPublisher(publisherID)

		manifest := PackageManifest{
			ID:      "acme.test-orbit",
			Name:    "Test Orbit",
			Version: "1.0.0",
			Type:    "orbit",
		}

		packageDir, cleanup := createTestPackageDir(t, manifest)
		defer cleanup()

		publisherRepo.On("GetByID", mock.Anything, publisherID).Return(publisher, nil)
		packageRepo.On("GetByPackageID", mock.Anything, manifest.ID).Return(nil, errors.New("not found"))

		cmd := PublishPackageCommand{
			PackagePath: packageDir,
			PublisherID: publisherID,
			DryRun:      true,
		}

		result, err := handler.Handle(context.Background(), cmd)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.DryRun)
		assert.Empty(t, result.Checksum)
		assert.Contains(t, result.Message, "Dry run successful")

		// Verify Create was NOT called
		packageRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
		versionRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)

		publisherRepo.AssertExpectations(t)
		packageRepo.AssertExpectations(t)
	})

	t.Run("returns ErrManifestNotFound when no manifest exists", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		publisherRepo := new(mockPublisherRepo)
		handler := NewPublishPackageHandler(packageRepo, versionRepo, publisherRepo)

		tmpDir, err := os.MkdirTemp("", "test-no-manifest-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		cmd := PublishPackageCommand{
			PackagePath: tmpDir,
			PublisherID: uuid.New(),
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.ErrorIs(t, err, ErrManifestNotFound)
		assert.Nil(t, result)
	})

	t.Run("returns ErrInvalidManifest when manifest is invalid JSON", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		publisherRepo := new(mockPublisherRepo)
		handler := NewPublishPackageHandler(packageRepo, versionRepo, publisherRepo)

		tmpDir, err := os.MkdirTemp("", "test-invalid-manifest-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		err = os.WriteFile(filepath.Join(tmpDir, "orbit.json"), []byte("invalid json"), 0644)
		require.NoError(t, err)

		cmd := PublishPackageCommand{
			PackagePath: tmpDir,
			PublisherID: uuid.New(),
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidManifest)
		assert.Nil(t, result)
	})

	t.Run("returns ErrInvalidManifest when id is missing", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		publisherRepo := new(mockPublisherRepo)
		handler := NewPublishPackageHandler(packageRepo, versionRepo, publisherRepo)

		tmpDir, err := os.MkdirTemp("", "test-missing-id-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		manifest := map[string]string{"name": "Test", "version": "1.0.0", "type": "orbit"}
		data, _ := json.Marshal(manifest)
		err = os.WriteFile(filepath.Join(tmpDir, "orbit.json"), data, 0644)
		require.NoError(t, err)

		cmd := PublishPackageCommand{
			PackagePath: tmpDir,
			PublisherID: uuid.New(),
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidManifest)
		assert.Contains(t, err.Error(), "missing id")
		assert.Nil(t, result)
	})

	t.Run("returns ErrInvalidManifest when name is missing", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		publisherRepo := new(mockPublisherRepo)
		handler := NewPublishPackageHandler(packageRepo, versionRepo, publisherRepo)

		tmpDir, err := os.MkdirTemp("", "test-missing-name-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		manifest := map[string]string{"id": "acme.test", "version": "1.0.0", "type": "orbit"}
		data, _ := json.Marshal(manifest)
		err = os.WriteFile(filepath.Join(tmpDir, "orbit.json"), data, 0644)
		require.NoError(t, err)

		cmd := PublishPackageCommand{
			PackagePath: tmpDir,
			PublisherID: uuid.New(),
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidManifest)
		assert.Contains(t, err.Error(), "missing name")
		assert.Nil(t, result)
	})

	t.Run("returns ErrInvalidManifest when version is missing", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		publisherRepo := new(mockPublisherRepo)
		handler := NewPublishPackageHandler(packageRepo, versionRepo, publisherRepo)

		tmpDir, err := os.MkdirTemp("", "test-missing-version-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		manifest := map[string]string{"id": "acme.test", "name": "Test", "type": "orbit"}
		data, _ := json.Marshal(manifest)
		err = os.WriteFile(filepath.Join(tmpDir, "orbit.json"), data, 0644)
		require.NoError(t, err)

		cmd := PublishPackageCommand{
			PackagePath: tmpDir,
			PublisherID: uuid.New(),
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidManifest)
		assert.Contains(t, err.Error(), "missing version")
		assert.Nil(t, result)
	})

	t.Run("returns ErrInvalidManifest when type is invalid", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		publisherRepo := new(mockPublisherRepo)
		handler := NewPublishPackageHandler(packageRepo, versionRepo, publisherRepo)

		tmpDir, err := os.MkdirTemp("", "test-invalid-type-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		manifest := map[string]string{"id": "acme.test", "name": "Test", "version": "1.0.0", "type": "invalid"}
		data, _ := json.Marshal(manifest)
		err = os.WriteFile(filepath.Join(tmpDir, "orbit.json"), data, 0644)
		require.NoError(t, err)

		cmd := PublishPackageCommand{
			PackagePath: tmpDir,
			PublisherID: uuid.New(),
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidManifest)
		assert.Contains(t, err.Error(), "type must be")
		assert.Nil(t, result)
	})

	t.Run("returns ErrUnauthorized when publisher not found", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		publisherRepo := new(mockPublisherRepo)
		handler := NewPublishPackageHandler(packageRepo, versionRepo, publisherRepo)

		publisherID := uuid.New()

		manifest := PackageManifest{
			ID:      "acme.test-orbit",
			Name:    "Test Orbit",
			Version: "1.0.0",
			Type:    "orbit",
		}

		packageDir, cleanup := createTestPackageDir(t, manifest)
		defer cleanup()

		publisherRepo.On("GetByID", mock.Anything, publisherID).Return(nil, errors.New("not found"))

		cmd := PublishPackageCommand{
			PackagePath: packageDir,
			PublisherID: publisherID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.ErrorIs(t, err, ErrUnauthorized)
		assert.Nil(t, result)

		publisherRepo.AssertExpectations(t)
	})

	t.Run("returns ErrUnauthorized when wrong publisher owns package", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		publisherRepo := new(mockPublisherRepo)
		handler := NewPublishPackageHandler(packageRepo, versionRepo, publisherRepo)

		publisherID := uuid.New()
		otherPublisherID := uuid.New()
		publisher := createTestPublisher(publisherID)

		existingPkg := createTestPackage("acme.test-orbit", domain.PackageTypeOrbit)
		existingPkg.PublisherID = otherPublisherID

		manifest := PackageManifest{
			ID:      "acme.test-orbit",
			Name:    "Test Orbit",
			Version: "2.0.0",
			Type:    "orbit",
		}

		packageDir, cleanup := createTestPackageDir(t, manifest)
		defer cleanup()

		publisherRepo.On("GetByID", mock.Anything, publisherID).Return(publisher, nil)
		packageRepo.On("GetByPackageID", mock.Anything, manifest.ID).Return(existingPkg, nil)

		cmd := PublishPackageCommand{
			PackagePath: packageDir,
			PublisherID: publisherID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.ErrorIs(t, err, ErrUnauthorized)
		assert.Nil(t, result)

		publisherRepo.AssertExpectations(t)
		packageRepo.AssertExpectations(t)
	})

	t.Run("returns ErrPackageExists when version already exists", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		publisherRepo := new(mockPublisherRepo)
		handler := NewPublishPackageHandler(packageRepo, versionRepo, publisherRepo)

		publisherID := uuid.New()
		publisher := createTestPublisher(publisherID)

		existingPkg := createTestPackage("acme.test-orbit", domain.PackageTypeOrbit)
		existingPkg.PublisherID = publisherID

		existingVersion := &domain.Version{
			ID:        uuid.New(),
			PackageID: existingPkg.ID,
			Version:   "1.0.0",
		}

		manifest := PackageManifest{
			ID:      "acme.test-orbit",
			Name:    "Test Orbit",
			Version: "1.0.0",
			Type:    "orbit",
		}

		packageDir, cleanup := createTestPackageDir(t, manifest)
		defer cleanup()

		publisherRepo.On("GetByID", mock.Anything, publisherID).Return(publisher, nil)
		packageRepo.On("GetByPackageID", mock.Anything, manifest.ID).Return(existingPkg, nil)
		versionRepo.On("GetByPackageAndVersion", mock.Anything, existingPkg.ID, manifest.Version).Return(existingVersion, nil)

		cmd := PublishPackageCommand{
			PackagePath: packageDir,
			PublisherID: publisherID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.ErrorIs(t, err, ErrPackageExists)
		assert.Nil(t, result)

		publisherRepo.AssertExpectations(t)
		packageRepo.AssertExpectations(t)
		versionRepo.AssertExpectations(t)
	})

	t.Run("fails when create package fails", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		publisherRepo := new(mockPublisherRepo)
		handler := NewPublishPackageHandler(packageRepo, versionRepo, publisherRepo)

		publisherID := uuid.New()
		publisher := createTestPublisher(publisherID)

		manifest := PackageManifest{
			ID:      "acme.test-orbit",
			Name:    "Test Orbit",
			Version: "1.0.0",
			Type:    "orbit",
		}

		packageDir, cleanup := createTestPackageDir(t, manifest)
		defer cleanup()

		publisherRepo.On("GetByID", mock.Anything, publisherID).Return(publisher, nil)
		packageRepo.On("GetByPackageID", mock.Anything, manifest.ID).Return(nil, errors.New("not found"))
		packageRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Package")).Return(errors.New("database error"))

		cmd := PublishPackageCommand{
			PackagePath: packageDir,
			PublisherID: publisherID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create package")
		assert.Nil(t, result)

		publisherRepo.AssertExpectations(t)
		packageRepo.AssertExpectations(t)
	})

	t.Run("fails when create version fails", func(t *testing.T) {
		packageRepo := new(mockPackageRepo)
		versionRepo := new(mockVersionRepo)
		publisherRepo := new(mockPublisherRepo)
		handler := NewPublishPackageHandler(packageRepo, versionRepo, publisherRepo)

		publisherID := uuid.New()
		publisher := createTestPublisher(publisherID)

		manifest := PackageManifest{
			ID:      "acme.test-orbit",
			Name:    "Test Orbit",
			Version: "1.0.0",
			Type:    "orbit",
		}

		packageDir, cleanup := createTestPackageDir(t, manifest)
		defer cleanup()

		publisherRepo.On("GetByID", mock.Anything, publisherID).Return(publisher, nil)
		packageRepo.On("GetByPackageID", mock.Anything, manifest.ID).Return(nil, errors.New("not found"))
		packageRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Package")).Return(nil)
		publisherRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Publisher")).Return(nil)
		versionRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Version")).Return(errors.New("version error"))

		cmd := PublishPackageCommand{
			PackagePath: packageDir,
			PublisherID: publisherID,
		}

		result, err := handler.Handle(context.Background(), cmd)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create version")
		assert.Nil(t, result)

		publisherRepo.AssertExpectations(t)
		packageRepo.AssertExpectations(t)
		versionRepo.AssertExpectations(t)
	})
}

func TestPublishPackageHandler_validateManifest(t *testing.T) {
	handler := &PublishPackageHandler{}

	t.Run("validates valid orbit manifest", func(t *testing.T) {
		manifest := &PackageManifest{
			ID:      "acme.test",
			Name:    "Test Package",
			Version: "1.0.0",
			Type:    "orbit",
		}

		err := handler.validateManifest(manifest)
		assert.NoError(t, err)
	})

	t.Run("validates valid engine manifest", func(t *testing.T) {
		manifest := &PackageManifest{
			ID:      "acme.priority",
			Name:    "Priority Engine",
			Version: "2.0.0",
			Type:    "engine",
		}

		err := handler.validateManifest(manifest)
		assert.NoError(t, err)
	})
}

func TestNewPublishPackageHandler(t *testing.T) {
	packageRepo := new(mockPackageRepo)
	versionRepo := new(mockVersionRepo)
	publisherRepo := new(mockPublisherRepo)

	handler := NewPublishPackageHandler(packageRepo, versionRepo, publisherRepo)

	require.NotNil(t, handler)
}
