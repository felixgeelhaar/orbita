package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/felixgeelhaar/orbita/internal/licensing/domain"
)

// FileRepository implements domain.Repository using file-based storage.
type FileRepository struct {
	filePath string
	mu       sync.RWMutex
}

// NewFileRepository creates a new file-based license repository.
func NewFileRepository(filePath string) *FileRepository {
	return &FileRepository{
		filePath: filePath,
	}
}

// Load retrieves the current license from the file.
// Returns nil, nil if no license file exists (first run).
func (r *FileRepository) Load(ctx context.Context) (*domain.License, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil // No license file yet
		}
		return nil, err
	}

	var license domain.License
	if err := json.Unmarshal(data, &license); err != nil {
		return nil, err
	}

	return &license, nil
}

// Save persists a license to the file.
func (r *FileRepository) Save(ctx context.Context, license *domain.License) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Ensure directory exists
	dir := filepath.Dir(r.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(license, "", "  ")
	if err != nil {
		return err
	}

	// Write with restrictive permissions (user read/write only)
	return os.WriteFile(r.filePath, data, 0600)
}

// Delete removes the license file.
func (r *FileRepository) Delete(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	err := os.Remove(r.filePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// Exists checks if a license file exists.
func (r *FileRepository) Exists(ctx context.Context) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, err := os.Stat(r.filePath)
	return err == nil
}

// FilePath returns the path to the license file.
func (r *FileRepository) FilePath() string {
	return r.filePath
}
