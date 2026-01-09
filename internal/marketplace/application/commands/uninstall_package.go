package commands

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/felixgeelhaar/orbita/internal/marketplace/domain"
	"github.com/google/uuid"
)

var (
	// ErrPackageNotInstalled is returned when trying to uninstall a package that isn't installed.
	ErrPackageNotInstalled = errors.New("package not installed")
)

// UninstallPackageCommand represents a command to uninstall a marketplace package.
type UninstallPackageCommand struct {
	PackageID string
	UserID    uuid.UUID
	KeepData  bool // If true, keep user data but remove package files
}

// UninstallPackageResult represents the result of uninstalling a package.
type UninstallPackageResult struct {
	PackageID string
	Version   string
	Message   string
}

// UninstallPackageHandler handles package uninstallation.
type UninstallPackageHandler struct {
	installedRepo domain.InstalledPackageRepository
}

// NewUninstallPackageHandler creates a new uninstall package handler.
func NewUninstallPackageHandler(installedRepo domain.InstalledPackageRepository) *UninstallPackageHandler {
	return &UninstallPackageHandler{
		installedRepo: installedRepo,
	}
}

// Handle executes the uninstall package command.
func (h *UninstallPackageHandler) Handle(ctx context.Context, cmd UninstallPackageCommand) (*UninstallPackageResult, error) {
	// Get installed package
	installed, err := h.installedRepo.GetByPackageID(ctx, cmd.PackageID, cmd.UserID)
	if err != nil || installed == nil {
		return nil, ErrPackageNotInstalled
	}

	// Remove package files
	if installed.InstallPath != "" {
		if err := os.RemoveAll(installed.InstallPath); err != nil {
			return nil, fmt.Errorf("failed to remove package files: %w", err)
		}
	}

	// Remove installation record
	if err := h.installedRepo.Delete(ctx, installed.ID); err != nil {
		return nil, fmt.Errorf("failed to remove installation record: %w", err)
	}

	return &UninstallPackageResult{
		PackageID: cmd.PackageID,
		Version:   installed.Version,
		Message:   fmt.Sprintf("Successfully uninstalled %s@%s", cmd.PackageID, installed.Version),
	}, nil
}
