package commands

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/felixgeelhaar/orbita/internal/marketplace/domain"
	"github.com/google/uuid"
)

var (
	// ErrPackageNotFound is returned when a package is not found in the marketplace.
	ErrPackageNotFound = errors.New("package not found")
	// ErrVersionNotFound is returned when a specific version is not found.
	ErrVersionNotFound = errors.New("version not found")
	// ErrPackageAlreadyInstalled is returned when trying to install an already installed package.
	ErrPackageAlreadyInstalled = errors.New("package already installed")
	// ErrChecksumMismatch is returned when the downloaded package checksum doesn't match.
	ErrChecksumMismatch = errors.New("checksum mismatch")
	// ErrDownloadFailed is returned when package download fails.
	ErrDownloadFailed = errors.New("download failed")
	// ErrFileTooLarge is returned when an extracted file exceeds the size limit.
	ErrFileTooLarge = errors.New("extracted file exceeds size limit")
)

const (
	// maxExtractedFileSize is the maximum size of a single extracted file (100MB).
	maxExtractedFileSize = 100 * 1024 * 1024
	// maxTotalExtractedSize is the maximum total size of all extracted files (1GB).
	maxTotalExtractedSize = 1024 * 1024 * 1024
)

// InstallPackageCommand represents a command to install a marketplace package.
type InstallPackageCommand struct {
	PackageID string
	Version   string // Optional, defaults to latest
	UserID    uuid.UUID
}

// InstallPackageResult represents the result of installing a package.
type InstallPackageResult struct {
	InstalledPackage *domain.InstalledPackage
	Message          string
}

// InstallPackageHandler handles package installation.
type InstallPackageHandler struct {
	packageRepo   domain.PackageRepository
	versionRepo   domain.VersionRepository
	installedRepo domain.InstalledPackageRepository
	installDir    string
	httpClient    *http.Client
}

// NewInstallPackageHandler creates a new install package handler.
func NewInstallPackageHandler(
	packageRepo domain.PackageRepository,
	versionRepo domain.VersionRepository,
	installedRepo domain.InstalledPackageRepository,
	installDir string,
) *InstallPackageHandler {
	return &InstallPackageHandler{
		packageRepo:   packageRepo,
		versionRepo:   versionRepo,
		installedRepo: installedRepo,
		installDir:    installDir,
		httpClient:    &http.Client{},
	}
}

// Handle executes the install package command.
func (h *InstallPackageHandler) Handle(ctx context.Context, cmd InstallPackageCommand) (*InstallPackageResult, error) {
	// Check if already installed
	existing, err := h.installedRepo.GetByPackageID(ctx, cmd.PackageID, cmd.UserID)
	if err == nil && existing != nil {
		return nil, ErrPackageAlreadyInstalled
	}

	// Get package from marketplace
	pkg, err := h.packageRepo.GetByPackageID(ctx, cmd.PackageID)
	if err != nil {
		return nil, ErrPackageNotFound
	}

	// Determine version to install
	versionStr := cmd.Version
	if versionStr == "" {
		versionStr = pkg.LatestVersion
	}

	// Get version details
	version, err := h.versionRepo.GetByPackageAndVersion(ctx, pkg.ID, versionStr)
	if err != nil {
		return nil, ErrVersionNotFound
	}

	// Create installation directory
	installPath := filepath.Join(h.installDir, string(pkg.Type)+"s", cmd.PackageID, versionStr)
	if err := os.MkdirAll(installPath, 0750); err != nil {
		return nil, fmt.Errorf("failed to create install directory: %w", err)
	}

	// Download package
	archivePath := filepath.Join(installPath, "package.tar.gz")
	if err := h.downloadPackage(ctx, version.DownloadURL, archivePath); err != nil {
		_ = os.RemoveAll(installPath) // Best-effort cleanup
		return nil, fmt.Errorf("failed to download package: %w", err)
	}

	// Verify checksum
	if version.Checksum != "" {
		if err := h.verifyChecksum(archivePath, version.Checksum); err != nil {
			_ = os.RemoveAll(installPath) // Best-effort cleanup
			return nil, err
		}
	}

	// Extract package
	if err := h.extractPackage(archivePath, installPath); err != nil {
		_ = os.RemoveAll(installPath) // Best-effort cleanup
		return nil, fmt.Errorf("failed to extract package: %w", err)
	}

	// Remove archive after extraction
	_ = os.Remove(archivePath) // Best-effort cleanup

	// Create installed package record
	installed := domain.NewInstalledPackage(cmd.PackageID, versionStr, pkg.Type, installPath, cmd.UserID)
	installed.SetChecksum(version.Checksum)

	if err := h.installedRepo.Create(ctx, installed); err != nil {
		_ = os.RemoveAll(installPath) // Best-effort cleanup
		return nil, fmt.Errorf("failed to save installation record: %w", err)
	}

	// Increment download count
	_ = h.packageRepo.IncrementDownloads(ctx, pkg.ID)

	return &InstallPackageResult{
		InstalledPackage: installed,
		Message:          fmt.Sprintf("Successfully installed %s@%s", cmd.PackageID, versionStr),
	}, nil
}

func (h *InstallPackageHandler) downloadPackage(ctx context.Context, url, destPath string) error {
	if url == "" {
		// For local/development mode, skip download
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d", ErrDownloadFailed, resp.StatusCode)
	}

	// #nosec G304 - destPath is internally constructed from installPath + "package.tar.gz"
	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

func (h *InstallPackageHandler) verifyChecksum(filePath, expectedChecksum string) error {
	// #nosec G304 - filePath is internally constructed from installPath + "package.tar.gz"
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return err
	}

	actualChecksum := hex.EncodeToString(hasher.Sum(nil))

	// Handle checksum with or without sha256: prefix
	expected := strings.TrimPrefix(expectedChecksum, "sha256:")
	if actualChecksum != expected {
		return fmt.Errorf("%w: expected %s, got %s", ErrChecksumMismatch, expected, actualChecksum)
	}

	return nil
}

func (h *InstallPackageHandler) extractPackage(archivePath, destDir string) error {
	// #nosec G304 - archivePath is internally constructed from installPath + "package.tar.gz"
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	var totalExtracted int64

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Check for decompression bomb - individual file size
		if header.Size > maxExtractedFileSize {
			return fmt.Errorf("%w: %s is %d bytes (max %d)", ErrFileTooLarge, header.Name, header.Size, maxExtractedFileSize)
		}

		// Check for decompression bomb - total extracted size
		totalExtracted += header.Size
		if totalExtracted > maxTotalExtractedSize {
			return fmt.Errorf("%w: total extracted size exceeds %d bytes", ErrFileTooLarge, maxTotalExtractedSize)
		}

		// Sanitize path to prevent directory traversal
		// #nosec G305 - path traversal is checked below before any file operations
		target := filepath.Join(destDir, header.Name)
		if !strings.HasPrefix(target, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0750); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0750); err != nil {
				return err
			}
			// #nosec G304 - target path is validated above against directory traversal
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}
			// Use io.LimitReader to prevent decompression bombs
			// #nosec G110 - size is checked above and limited via LimitReader
			written, err := io.Copy(outFile, io.LimitReader(tr, maxExtractedFileSize))
			_ = outFile.Close() // Best-effort; file was successfully written
			if err != nil {
				return err
			}
			if written >= maxExtractedFileSize {
				return fmt.Errorf("%w: %s exceeded size limit during extraction", ErrFileTooLarge, header.Name)
			}

			// Set executable permission for binaries
			if header.Mode&0111 != 0 {
				// #nosec G302 - 0750 required for executable binaries
				if err := os.Chmod(target, 0750); err != nil {
					return fmt.Errorf("failed to set permissions: %w", err)
				}
			}
		}
	}

	return nil
}
