package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/felixgeelhaar/orbita/internal/marketplace/domain"
	"github.com/google/uuid"
)

var (
	// ErrPackageAlreadyDisabled is returned when trying to disable an already disabled package.
	ErrPackageAlreadyDisabled = errors.New("package is already disabled")
)

// DisablePackageCommand represents a command to disable an installed package.
type DisablePackageCommand struct {
	PackageID string
	UserID    uuid.UUID
}

// DisablePackageResult represents the result of disabling a package.
type DisablePackageResult struct {
	PackageID string
	Version   string
	Message   string
}

// DisablePackageHandler handles package disabling.
type DisablePackageHandler struct {
	installedRepo domain.InstalledPackageRepository
}

// NewDisablePackageHandler creates a new disable package handler.
func NewDisablePackageHandler(installedRepo domain.InstalledPackageRepository) *DisablePackageHandler {
	return &DisablePackageHandler{
		installedRepo: installedRepo,
	}
}

// Handle executes the disable package command.
func (h *DisablePackageHandler) Handle(ctx context.Context, cmd DisablePackageCommand) (*DisablePackageResult, error) {
	// Get installed package
	installed, err := h.installedRepo.GetByPackageID(ctx, cmd.PackageID, cmd.UserID)
	if err != nil || installed == nil {
		return nil, ErrPackageNotInstalled
	}

	// Check if already disabled
	if !installed.Enabled {
		return nil, ErrPackageAlreadyDisabled
	}

	// Disable the package
	installed.Enabled = false

	// Save the updated installation record
	if err := h.installedRepo.Update(ctx, installed); err != nil {
		return nil, fmt.Errorf("failed to disable package: %w", err)
	}

	return &DisablePackageResult{
		PackageID: cmd.PackageID,
		Version:   installed.Version,
		Message:   fmt.Sprintf("Successfully disabled %s@%s", cmd.PackageID, installed.Version),
	}, nil
}
