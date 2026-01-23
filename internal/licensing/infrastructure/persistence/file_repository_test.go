package persistence

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/felixgeelhaar/orbita/internal/licensing/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileRepository(t *testing.T) {
	repo := NewFileRepository("/tmp/test-license.json")
	assert.NotNil(t, repo)
	assert.Equal(t, "/tmp/test-license.json", repo.FilePath())
}

func TestFileRepository_Load_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "nonexistent.json")

	repo := NewFileRepository(filePath)
	ctx := context.Background()

	license, err := repo.Load(ctx)
	assert.NoError(t, err)
	assert.Nil(t, license) // No file = nil license, no error
}

func TestFileRepository_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "license.json")

	repo := NewFileRepository(filePath)
	ctx := context.Background()

	license := &domain.License{
		Version:      1,
		LicenseKey:   "ORB-TEST-1234-5678",
		LicenseID:    uuid.New(),
		Email:        "test@example.com",
		Plan:         "pro",
		Entitlements: []string{"smart-habits", "ai-inbox"},
		IssuedAt:     time.Now(),
		ExpiresAt:    time.Now().Add(365 * 24 * time.Hour),
		Signature:    "test-signature",
	}

	// Save
	err := repo.Save(ctx, license)
	require.NoError(t, err)

	// Load
	loaded, err := repo.Load(ctx)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, license.LicenseKey, loaded.LicenseKey)
	assert.Equal(t, license.LicenseID, loaded.LicenseID)
	assert.Equal(t, license.Email, loaded.Email)
	assert.Equal(t, license.Plan, loaded.Plan)
	assert.Equal(t, license.Entitlements, loaded.Entitlements)
	assert.Equal(t, license.Signature, loaded.Signature)
}

func TestFileRepository_Save_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "nested", "deep", "license.json")

	repo := NewFileRepository(filePath)
	ctx := context.Background()

	license := &domain.License{
		Version:    1,
		LicenseKey: "ORB-TEST-1234-5678",
	}

	err := repo.Save(ctx, license)
	require.NoError(t, err)

	// Verify file was created
	_, err = os.Stat(filePath)
	assert.NoError(t, err)
}

func TestFileRepository_Exists_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "nonexistent.json")

	repo := NewFileRepository(filePath)
	ctx := context.Background()

	assert.False(t, repo.Exists(ctx))
}

func TestFileRepository_Exists_WithFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "license.json")

	repo := NewFileRepository(filePath)
	ctx := context.Background()

	// Create file
	err := repo.Save(ctx, &domain.License{Version: 1})
	require.NoError(t, err)

	assert.True(t, repo.Exists(ctx))
}

func TestFileRepository_Delete_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "nonexistent.json")

	repo := NewFileRepository(filePath)
	ctx := context.Background()

	// Delete non-existent file should not error
	err := repo.Delete(ctx)
	assert.NoError(t, err)
}

func TestFileRepository_Delete_WithFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "license.json")

	repo := NewFileRepository(filePath)
	ctx := context.Background()

	// Create file first
	err := repo.Save(ctx, &domain.License{Version: 1})
	require.NoError(t, err)
	require.True(t, repo.Exists(ctx))

	// Delete
	err = repo.Delete(ctx)
	assert.NoError(t, err)

	// Verify deleted
	assert.False(t, repo.Exists(ctx))
}

func TestFileRepository_Load_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "license.json")

	// Write invalid JSON
	err := os.WriteFile(filePath, []byte("not valid json"), 0600)
	require.NoError(t, err)

	repo := NewFileRepository(filePath)
	ctx := context.Background()

	license, err := repo.Load(ctx)
	assert.Error(t, err)
	assert.Nil(t, license)
}

func TestFileRepository_FilePath(t *testing.T) {
	expectedPath := "/some/path/to/license.json"
	repo := NewFileRepository(expectedPath)

	assert.Equal(t, expectedPath, repo.FilePath())
}

func TestFileRepository_FilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "license.json")

	repo := NewFileRepository(filePath)
	ctx := context.Background()

	license := &domain.License{Version: 1, LicenseKey: "ORB-TEST-1234-5678"}
	err := repo.Save(ctx, license)
	require.NoError(t, err)

	// Check file permissions (0600 = owner read/write only)
	info, err := os.Stat(filePath)
	require.NoError(t, err)

	// On Unix-like systems, verify restrictive permissions
	mode := info.Mode().Perm()
	assert.Equal(t, os.FileMode(0600), mode)
}
