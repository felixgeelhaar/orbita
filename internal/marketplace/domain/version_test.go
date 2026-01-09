package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewVersion(t *testing.T) {
	packageID := uuid.New()
	version := NewVersion(packageID, "1.0.0")

	assert.NotEqual(t, uuid.Nil, version.ID)
	assert.Equal(t, packageID, version.PackageID)
	assert.Equal(t, "1.0.0", version.Version)
	assert.Empty(t, version.MinAPIVersion)
	assert.Empty(t, version.Changelog)
	assert.Empty(t, version.Checksum)
	assert.Empty(t, version.DownloadURL)
	assert.Equal(t, int64(0), version.Size)
	assert.Equal(t, int64(0), version.Downloads)
	assert.False(t, version.Prerelease)
	assert.False(t, version.Deprecated)
	assert.Empty(t, version.DeprecationMessage)
	assert.False(t, version.PublishedAt.IsZero())
	assert.False(t, version.CreatedAt.IsZero())
}

func TestVersion_SetMinAPIVersion(t *testing.T) {
	packageID := uuid.New()
	version := NewVersion(packageID, "1.0.0")

	version.SetMinAPIVersion("1.0.0")

	assert.Equal(t, "1.0.0", version.MinAPIVersion)
}

func TestVersion_SetChangelog(t *testing.T) {
	packageID := uuid.New()
	version := NewVersion(packageID, "1.0.0")

	version.SetChangelog("- Initial release\n- Added features")

	assert.Equal(t, "- Initial release\n- Added features", version.Changelog)
}

func TestVersion_SetChecksum(t *testing.T) {
	packageID := uuid.New()
	version := NewVersion(packageID, "1.0.0")

	version.SetChecksum("sha256:abc123def456")

	assert.Equal(t, "sha256:abc123def456", version.Checksum)
}

func TestVersion_SetDownloadURL(t *testing.T) {
	packageID := uuid.New()
	version := NewVersion(packageID, "1.0.0")

	version.SetDownloadURL("https://cdn.example.com/packages/test/1.0.0.tar.gz")

	assert.Equal(t, "https://cdn.example.com/packages/test/1.0.0.tar.gz", version.DownloadURL)
}

func TestVersion_SetSize(t *testing.T) {
	packageID := uuid.New()
	version := NewVersion(packageID, "1.0.0")

	version.SetSize(1024 * 1024) // 1MB

	assert.Equal(t, int64(1024*1024), version.Size)
}

func TestVersion_SetPrerelease(t *testing.T) {
	packageID := uuid.New()
	version := NewVersion(packageID, "1.0.0-beta.1")

	version.SetPrerelease(true)

	assert.True(t, version.Prerelease)
}

func TestVersion_Deprecate(t *testing.T) {
	packageID := uuid.New()
	version := NewVersion(packageID, "1.0.0")

	version.Deprecate("This version has a critical security vulnerability. Please upgrade to 1.0.1")

	assert.True(t, version.Deprecated)
	assert.Equal(t, "This version has a critical security vulnerability. Please upgrade to 1.0.1", version.DeprecationMessage)
}

func TestVersion_Undeprecate(t *testing.T) {
	packageID := uuid.New()
	version := NewVersion(packageID, "1.0.0")
	version.Deprecate("Test reason")

	version.Undeprecate()

	assert.False(t, version.Deprecated)
	assert.Empty(t, version.DeprecationMessage)
}

func TestVersion_IncrementDownloads(t *testing.T) {
	packageID := uuid.New()
	version := NewVersion(packageID, "1.0.0")

	version.IncrementDownloads()
	assert.Equal(t, int64(1), version.Downloads)

	version.IncrementDownloads()
	assert.Equal(t, int64(2), version.Downloads)
}

func TestVersion_SetPublishedAt(t *testing.T) {
	packageID := uuid.New()
	version := NewVersion(packageID, "1.0.0")
	publishedAt := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	version.SetPublishedAt(publishedAt)

	assert.Equal(t, publishedAt, version.PublishedAt)
}

func TestVersion_IsStable(t *testing.T) {
	tests := []struct {
		name       string
		prerelease bool
		deprecated bool
		expected   bool
	}{
		{"stable version", false, false, true},
		{"prerelease version", true, false, false},
		{"deprecated version", false, true, false},
		{"deprecated prerelease", true, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packageID := uuid.New()
			version := NewVersion(packageID, "1.0.0")
			version.SetPrerelease(tt.prerelease)
			if tt.deprecated {
				version.Deprecate("test")
			}

			assert.Equal(t, tt.expected, version.IsStable())
		})
	}
}
