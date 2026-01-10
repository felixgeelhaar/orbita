package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInstalledPackage(t *testing.T) {
	t.Run("creates new installed package with correct fields", func(t *testing.T) {
		userID := uuid.New()
		packageID := "test.orbit"
		version := "1.0.0"
		pkgType := PackageTypeOrbit
		installPath := "/path/to/package"

		before := time.Now().UTC()
		pkg := NewInstalledPackage(packageID, version, pkgType, installPath, userID)
		after := time.Now().UTC()

		require.NotNil(t, pkg)
		assert.NotEqual(t, uuid.Nil, pkg.ID)
		assert.Equal(t, packageID, pkg.PackageID)
		assert.Equal(t, version, pkg.Version)
		assert.Equal(t, pkgType, pkg.Type)
		assert.Equal(t, installPath, pkg.InstallPath)
		assert.Equal(t, userID, pkg.UserID)
		assert.True(t, pkg.Enabled)
		assert.Empty(t, pkg.Checksum)
		assert.True(t, pkg.InstalledAt.After(before) || pkg.InstalledAt.Equal(before))
		assert.True(t, pkg.InstalledAt.Before(after) || pkg.InstalledAt.Equal(after))
		assert.Equal(t, pkg.InstalledAt, pkg.UpdatedAt)
	})

	t.Run("creates package with engine type", func(t *testing.T) {
		userID := uuid.New()
		pkg := NewInstalledPackage("test.engine", "2.0.0", PackageTypeEngine, "/engines/test", userID)

		require.NotNil(t, pkg)
		assert.Equal(t, PackageTypeEngine, pkg.Type)
	})
}

func TestInstalledPackage_SetChecksum(t *testing.T) {
	t.Run("sets checksum and updates timestamp", func(t *testing.T) {
		userID := uuid.New()
		pkg := NewInstalledPackage("test.orbit", "1.0.0", PackageTypeOrbit, "/path", userID)
		originalUpdatedAt := pkg.UpdatedAt

		// Small delay to ensure timestamp difference
		time.Sleep(time.Millisecond)

		checksum := "sha256:abc123def456"
		pkg.SetChecksum(checksum)

		assert.Equal(t, checksum, pkg.Checksum)
		assert.True(t, pkg.UpdatedAt.After(originalUpdatedAt) || pkg.UpdatedAt.Equal(originalUpdatedAt))
	})
}

func TestInstalledPackage_UpdateVersion(t *testing.T) {
	t.Run("updates version and timestamp", func(t *testing.T) {
		userID := uuid.New()
		pkg := NewInstalledPackage("test.orbit", "1.0.0", PackageTypeOrbit, "/path", userID)
		originalUpdatedAt := pkg.UpdatedAt

		time.Sleep(time.Millisecond)

		newVersion := "2.0.0"
		pkg.UpdateVersion(newVersion)

		assert.Equal(t, newVersion, pkg.Version)
		assert.True(t, pkg.UpdatedAt.After(originalUpdatedAt) || pkg.UpdatedAt.Equal(originalUpdatedAt))
	})
}

func TestInstalledPackage_Enable(t *testing.T) {
	t.Run("enables package and updates timestamp", func(t *testing.T) {
		userID := uuid.New()
		pkg := NewInstalledPackage("test.orbit", "1.0.0", PackageTypeOrbit, "/path", userID)
		pkg.Enabled = false
		originalUpdatedAt := pkg.UpdatedAt

		time.Sleep(time.Millisecond)

		pkg.Enable()

		assert.True(t, pkg.Enabled)
		assert.True(t, pkg.UpdatedAt.After(originalUpdatedAt) || pkg.UpdatedAt.Equal(originalUpdatedAt))
	})
}

func TestInstalledPackage_Disable(t *testing.T) {
	t.Run("disables package and updates timestamp", func(t *testing.T) {
		userID := uuid.New()
		pkg := NewInstalledPackage("test.orbit", "1.0.0", PackageTypeOrbit, "/path", userID)
		assert.True(t, pkg.Enabled)
		originalUpdatedAt := pkg.UpdatedAt

		time.Sleep(time.Millisecond)

		pkg.Disable()

		assert.False(t, pkg.Enabled)
		assert.True(t, pkg.UpdatedAt.After(originalUpdatedAt) || pkg.UpdatedAt.Equal(originalUpdatedAt))
	})
}
