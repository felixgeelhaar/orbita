package commands

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/felixgeelhaar/orbita/internal/marketplace/domain"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/security"
	"github.com/google/uuid"
)

var (
	// ErrManifestNotFound is returned when package manifest is not found.
	ErrManifestNotFound = errors.New("manifest file not found (orbit.json or engine.json)")
	// ErrInvalidManifest is returned when manifest is invalid.
	ErrInvalidManifest = errors.New("invalid manifest")
	// ErrPackageExists is returned when trying to publish an existing version.
	ErrPackageExists = errors.New("package version already exists")
	// ErrUnauthorized is returned when not authorized to publish.
	ErrUnauthorized = errors.New("unauthorized to publish this package")
)

// PackageManifest represents the manifest file for a package.
type PackageManifest struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Version       string   `json:"version"`
	Type          string   `json:"type"` // "orbit" or "engine"
	Author        string   `json:"author,omitempty"`
	Description   string   `json:"description,omitempty"`
	License       string   `json:"license,omitempty"`
	Homepage      string   `json:"homepage,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	MinAPIVersion string   `json:"min_api_version,omitempty"`
	Entitlement   string   `json:"entitlement,omitempty"`
}

// PublishPackageCommand represents a command to publish a package.
type PublishPackageCommand struct {
	PackagePath string    // Path to package directory
	PublisherID uuid.UUID // Publisher ID from authentication
	DryRun      bool      // If true, validate but don't publish
}

// PublishPackageResult represents the result of publishing a package.
type PublishPackageResult struct {
	PackageID string
	Version   string
	Checksum  string
	Message   string
	DryRun    bool
}

// PublishPackageHandler handles package publishing.
type PublishPackageHandler struct {
	packageRepo   domain.PackageRepository
	versionRepo   domain.VersionRepository
	publisherRepo domain.PublisherRepository
}

// NewPublishPackageHandler creates a new publish package handler.
func NewPublishPackageHandler(
	packageRepo domain.PackageRepository,
	versionRepo domain.VersionRepository,
	publisherRepo domain.PublisherRepository,
) *PublishPackageHandler {
	return &PublishPackageHandler{
		packageRepo:   packageRepo,
		versionRepo:   versionRepo,
		publisherRepo: publisherRepo,
	}
}

// Handle executes the publish package command.
func (h *PublishPackageHandler) Handle(ctx context.Context, cmd PublishPackageCommand) (*PublishPackageResult, error) {
	// Read manifest
	manifest, err := h.readManifest(cmd.PackagePath)
	if err != nil {
		return nil, err
	}

	// Validate manifest
	if err := h.validateManifest(manifest); err != nil {
		return nil, err
	}

	// Check if publisher exists and matches
	publisher, err := h.publisherRepo.GetByID(ctx, cmd.PublisherID)
	if err != nil || publisher == nil {
		return nil, ErrUnauthorized
	}

	// Check if package exists
	existingPkg, err := h.packageRepo.GetByPackageID(ctx, manifest.ID)
	if err == nil && existingPkg != nil {
		// Package exists, verify ownership
		if existingPkg.PublisherID != cmd.PublisherID {
			return nil, ErrUnauthorized
		}

		// Check if version already exists
		existingVersion, err := h.versionRepo.GetByPackageAndVersion(ctx, existingPkg.ID, manifest.Version)
		if err == nil && existingVersion != nil {
			return nil, ErrPackageExists
		}
	}

	if cmd.DryRun {
		return &PublishPackageResult{
			PackageID: manifest.ID,
			Version:   manifest.Version,
			Message:   "Dry run successful - package would be published",
			DryRun:    true,
		}, nil
	}

	// Create package archive
	archivePath, checksum, err := h.createArchive(cmd.PackagePath, manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to create archive: %w", err)
	}
	defer os.Remove(archivePath)

	// Create or update package
	var pkg *domain.Package
	if existingPkg != nil {
		pkg = existingPkg
		pkg.SetLatestVersion(manifest.Version)
	} else {
		pkgType := domain.PackageType(manifest.Type)
		pkg = domain.NewPackage(manifest.ID, pkgType, manifest.Name, manifest.Description)
		pkg.SetPublisher(cmd.PublisherID)
		pkg.SetAuthor(manifest.Author)
		pkg.SetLicense(manifest.License)
		pkg.SetHomepage(manifest.Homepage)
		pkg.SetTags(manifest.Tags)
		pkg.SetLatestVersion(manifest.Version)

		if err := h.packageRepo.Create(ctx, pkg); err != nil {
			return nil, fmt.Errorf("failed to create package: %w", err)
		}

		// Increment publisher package count
		publisher.IncrementPackageCount()
		_ = h.publisherRepo.Update(ctx, publisher)
	}

	// Create version
	version := domain.NewVersion(pkg.ID, manifest.Version)
	version.SetMinAPIVersion(manifest.MinAPIVersion)
	version.SetChecksum("sha256:" + checksum)
	// In production, this would upload to storage and get URL
	version.SetDownloadURL(fmt.Sprintf("https://marketplace.orbita.dev/packages/%s/%s/download", manifest.ID, manifest.Version))

	// Get archive size
	if stat, err := os.Stat(archivePath); err == nil {
		version.SetSize(stat.Size())
	}

	if err := h.versionRepo.Create(ctx, version); err != nil {
		return nil, fmt.Errorf("failed to create version: %w", err)
	}

	// Update package latest version if newer
	if existingPkg != nil {
		existingPkg.SetLatestVersion(manifest.Version)
		_ = h.packageRepo.Update(ctx, existingPkg)
	}

	return &PublishPackageResult{
		PackageID: manifest.ID,
		Version:   manifest.Version,
		Checksum:  checksum,
		Message:   fmt.Sprintf("Successfully published %s@%s", manifest.ID, manifest.Version),
		DryRun:    false,
	}, nil
}

func (h *PublishPackageHandler) readManifest(packagePath string) (*PackageManifest, error) {
	// Validate the package path first
	cleanPackagePath, err := security.ValidateFilePath(packagePath)
	if err != nil {
		return nil, fmt.Errorf("invalid package path: %w", err)
	}

	// Try orbit.json first, then engine.json
	manifestPaths := []string{
		filepath.Join(cleanPackagePath, "orbit.json"),
		filepath.Join(cleanPackagePath, "engine.json"),
	}

	var manifest PackageManifest
	var found bool

	for _, path := range manifestPaths {
		// Path is already validated since it's under cleanPackagePath
		data, err := security.SafeReadFileInDir(path, cleanPackagePath)
		if err != nil {
			continue
		}

		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidManifest, err)
		}

		found = true
		break
	}

	if !found {
		return nil, ErrManifestNotFound
	}

	return &manifest, nil
}

func (h *PublishPackageHandler) validateManifest(manifest *PackageManifest) error {
	if manifest.ID == "" {
		return fmt.Errorf("%w: missing id", ErrInvalidManifest)
	}
	if manifest.Name == "" {
		return fmt.Errorf("%w: missing name", ErrInvalidManifest)
	}
	if manifest.Version == "" {
		return fmt.Errorf("%w: missing version", ErrInvalidManifest)
	}
	if manifest.Type != "orbit" && manifest.Type != "engine" {
		return fmt.Errorf("%w: type must be 'orbit' or 'engine'", ErrInvalidManifest)
	}
	return nil
}

func (h *PublishPackageHandler) createArchive(packagePath string, manifest *PackageManifest) (string, string, error) {
	// Create temporary file for archive
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("%s-%s-*.tar.gz", manifest.ID, manifest.Version))
	if err != nil {
		return "", "", err
	}
	archivePath := tmpFile.Name()

	// Create gzip writer
	gzw := gzip.NewWriter(tmpFile)
	defer gzw.Close()

	// Create tar writer
	tw := tar.NewWriter(gzw)
	defer tw.Close()

	// Walk directory and add files
	err = filepath.Walk(packagePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden files and directories
		if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(packagePath, path)
		if err != nil {
			return err
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// Write file content
		if !info.IsDir() {
			// #nosec G304 - path comes from filepath.Walk which is bounded by packagePath
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(tw, file); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		_ = tmpFile.Close()       // Best-effort cleanup
		_ = os.Remove(archivePath) // Best-effort cleanup
		return "", "", err
	}

	// Close writers to flush data
	_ = tw.Close()      // Best-effort cleanup
	_ = gzw.Close()     // Best-effort cleanup
	_ = tmpFile.Close() // Best-effort cleanup

	// Calculate checksum
	checksum, err := calculateChecksum(archivePath)
	if err != nil {
		_ = os.Remove(archivePath) // Best-effort cleanup
		return "", "", err
	}

	return archivePath, checksum, nil
}

func calculateChecksum(filePath string) (string, error) {
	// #nosec G304 - filePath is from os.CreateTemp (internal path)
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
