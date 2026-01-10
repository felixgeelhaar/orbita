package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/felixgeelhaar/orbita/internal/marketplace/domain"
	"github.com/google/uuid"
)

// UpdatePackageCommand represents a command to update an installed package.
type UpdatePackageCommand struct {
	PackageID string
	Version   string // Optional, defaults to latest
	UserID    uuid.UUID
}

// UpdatePackageResult represents the result of updating a package.
type UpdatePackageResult struct {
	InstalledPackage *domain.InstalledPackage
	OldVersion       string
	NewVersion       string
	Message          string
}

// UpdatePackageHandler handles package updates.
type UpdatePackageHandler struct {
	packageRepo   domain.PackageRepository
	versionRepo   domain.VersionRepository
	installedRepo domain.InstalledPackageRepository
	installHandler *InstallPackageHandler
}

// NewUpdatePackageHandler creates a new update package handler.
func NewUpdatePackageHandler(
	packageRepo domain.PackageRepository,
	versionRepo domain.VersionRepository,
	installedRepo domain.InstalledPackageRepository,
	installDir string,
) *UpdatePackageHandler {
	return &UpdatePackageHandler{
		packageRepo:   packageRepo,
		versionRepo:   versionRepo,
		installedRepo: installedRepo,
		installHandler: NewInstallPackageHandler(packageRepo, versionRepo, installedRepo, installDir),
	}
}

// Handle executes the update package command.
func (h *UpdatePackageHandler) Handle(ctx context.Context, cmd UpdatePackageCommand) (*UpdatePackageResult, error) {
	// Get installed package
	installed, err := h.installedRepo.GetByPackageID(ctx, cmd.PackageID, cmd.UserID)
	if err != nil || installed == nil {
		return nil, ErrPackageNotInstalled
	}

	oldVersion := installed.Version

	// Get package from marketplace
	pkg, err := h.packageRepo.GetByPackageID(ctx, cmd.PackageID)
	if err != nil {
		return nil, ErrPackageNotFound
	}

	// Determine target version
	targetVersion := cmd.Version
	if targetVersion == "" {
		targetVersion = pkg.LatestVersion
	}

	// Check if already at target version
	if installed.Version == targetVersion {
		return &UpdatePackageResult{
			InstalledPackage: installed,
			OldVersion:       oldVersion,
			NewVersion:       targetVersion,
			Message:          fmt.Sprintf("Package %s is already at version %s", cmd.PackageID, targetVersion),
		}, nil
	}

	// Get version details
	version, err := h.versionRepo.GetByPackageAndVersion(ctx, pkg.ID, targetVersion)
	if err != nil {
		return nil, ErrVersionNotFound
	}

	// Backup old installation path
	oldInstallPath := installed.InstallPath

	// Create new installation directory
	newInstallPath := filepath.Join(filepath.Dir(filepath.Dir(oldInstallPath)), targetVersion)
	if err := os.MkdirAll(newInstallPath, 0750); err != nil {
		return nil, fmt.Errorf("failed to create install directory: %w", err)
	}

	// Download new version
	archivePath := filepath.Join(newInstallPath, "package.tar.gz")
	if err := h.installHandler.downloadPackage(ctx, version.DownloadURL, archivePath); err != nil {
		_ = os.RemoveAll(newInstallPath) // Best-effort cleanup
		return nil, fmt.Errorf("failed to download package: %w", err)
	}

	// Verify checksum
	if version.Checksum != "" {
		if err := h.installHandler.verifyChecksum(archivePath, version.Checksum); err != nil {
			_ = os.RemoveAll(newInstallPath) // Best-effort cleanup
			return nil, err
		}
	}

	// Extract package
	if err := h.installHandler.extractPackage(archivePath, newInstallPath); err != nil {
		_ = os.RemoveAll(newInstallPath) // Best-effort cleanup
		return nil, fmt.Errorf("failed to extract package: %w", err)
	}

	// Remove archive
	_ = os.Remove(archivePath) // Best-effort cleanup

	// Update installation record
	installed.UpdateVersion(targetVersion)
	installed.InstallPath = newInstallPath
	installed.SetChecksum(version.Checksum)

	if err := h.installedRepo.Update(ctx, installed); err != nil {
		_ = os.RemoveAll(newInstallPath) // Best-effort cleanup
		return nil, fmt.Errorf("failed to update installation record: %w", err)
	}

	// Remove old installation
	if oldInstallPath != newInstallPath {
		_ = os.RemoveAll(oldInstallPath) // Best-effort cleanup
	}

	// Increment download count
	_ = h.packageRepo.IncrementDownloads(ctx, pkg.ID)

	return &UpdatePackageResult{
		InstalledPackage: installed,
		OldVersion:       oldVersion,
		NewVersion:       targetVersion,
		Message:          fmt.Sprintf("Successfully updated %s from %s to %s", cmd.PackageID, oldVersion, targetVersion),
	}, nil
}
