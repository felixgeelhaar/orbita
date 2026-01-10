package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/felixgeelhaar/orbita/internal/marketplace/domain"
	"github.com/google/uuid"
)

var (
	// ErrPackageAlreadyEnabled is returned when trying to enable an already enabled package.
	ErrPackageAlreadyEnabled = errors.New("package is already enabled")
)

// EnablePackageCommand represents a command to enable an installed package.
type EnablePackageCommand struct {
	PackageID string
	UserID    uuid.UUID
}

// EnablePackageResult represents the result of enabling a package.
type EnablePackageResult struct {
	PackageID string
	Version   string
	Message   string
}

// EnablePackageHandler handles package enabling.
type EnablePackageHandler struct {
	installedRepo domain.InstalledPackageRepository
}

// NewEnablePackageHandler creates a new enable package handler.
func NewEnablePackageHandler(installedRepo domain.InstalledPackageRepository) *EnablePackageHandler {
	return &EnablePackageHandler{
		installedRepo: installedRepo,
	}
}

// Handle executes the enable package command.
func (h *EnablePackageHandler) Handle(ctx context.Context, cmd EnablePackageCommand) (*EnablePackageResult, error) {
	// Get installed package
	installed, err := h.installedRepo.GetByPackageID(ctx, cmd.PackageID, cmd.UserID)
	if err != nil || installed == nil {
		return nil, ErrPackageNotInstalled
	}

	// Check if already enabled
	if installed.Enabled {
		return nil, ErrPackageAlreadyEnabled
	}

	// Enable the package
	installed.Enabled = true

	// Save the updated installation record
	if err := h.installedRepo.Update(ctx, installed); err != nil {
		return nil, fmt.Errorf("failed to enable package: %w", err)
	}

	return &EnablePackageResult{
		PackageID: cmd.PackageID,
		Version:   installed.Version,
		Message:   fmt.Sprintf("Successfully enabled %s@%s", cmd.PackageID, installed.Version),
	}, nil
}
