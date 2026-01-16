package domain

import "context"

// Repository defines the interface for license storage.
type Repository interface {
	// Load retrieves the current license from storage.
	// Returns nil, nil if no license file exists (first run).
	Load(ctx context.Context) (*License, error)

	// Save persists a license to storage.
	Save(ctx context.Context, license *License) error

	// Delete removes the license from storage.
	Delete(ctx context.Context) error

	// Exists checks if a license file exists.
	Exists(ctx context.Context) bool
}
