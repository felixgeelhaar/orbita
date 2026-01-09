package domain

import (
	"time"

	"github.com/google/uuid"
)

// Version represents a specific version of a marketplace package.
type Version struct {
	// ID is the unique identifier for this version.
	ID uuid.UUID

	// PackageID is the ID of the parent package.
	PackageID uuid.UUID

	// Version is the semantic version string (e.g., "1.0.0").
	Version string

	// MinAPIVersion is the minimum SDK version required.
	MinAPIVersion string

	// Changelog describes what changed in this version.
	Changelog string

	// Checksum is the SHA256 checksum of the package archive.
	Checksum string

	// DownloadURL is the URL to download this version.
	DownloadURL string

	// Size is the package size in bytes.
	Size int64

	// Downloads is the download count for this version.
	Downloads int64

	// Prerelease indicates if this is a prerelease version.
	Prerelease bool

	// Deprecated indicates if this version is deprecated.
	Deprecated bool

	// DeprecationMessage is the message explaining why this version is deprecated.
	DeprecationMessage string

	// PublishedAt is when this version was published.
	PublishedAt time.Time

	// CreatedAt is when this version record was created.
	CreatedAt time.Time
}

// NewVersion creates a new package version.
func NewVersion(packageID uuid.UUID, version string) *Version {
	now := time.Now().UTC()
	return &Version{
		ID:          uuid.New(),
		PackageID:   packageID,
		Version:     version,
		PublishedAt: now,
		CreatedAt:   now,
	}
}

// SetMinAPIVersion sets the minimum API version required.
func (v *Version) SetMinAPIVersion(minAPIVersion string) {
	v.MinAPIVersion = minAPIVersion
}

// SetChecksum sets the checksum for the package.
func (v *Version) SetChecksum(checksum string) {
	v.Checksum = checksum
}

// SetDownloadURL sets the download URL for the package.
func (v *Version) SetDownloadURL(url string) {
	v.DownloadURL = url
}

// SetSize sets the package size.
func (v *Version) SetSize(size int64) {
	v.Size = size
}

// SetChangelog sets the changelog for this version.
func (v *Version) SetChangelog(changelog string) {
	v.Changelog = changelog
}

// SetPrerelease marks this as a prerelease version.
func (v *Version) SetPrerelease(prerelease bool) {
	v.Prerelease = prerelease
}

// Deprecate marks this version as deprecated.
func (v *Version) Deprecate(message string) {
	v.Deprecated = true
	v.DeprecationMessage = message
}

// Undeprecate removes the deprecation status.
func (v *Version) Undeprecate() {
	v.Deprecated = false
	v.DeprecationMessage = ""
}

// IncrementDownloads increments the download count.
func (v *Version) IncrementDownloads() {
	v.Downloads++
}

// SetPublishedAt sets the published at timestamp.
func (v *Version) SetPublishedAt(publishedAt time.Time) {
	v.PublishedAt = publishedAt
}

// IsStable returns true if this version is stable (not prerelease and not deprecated).
func (v *Version) IsStable() bool {
	return !v.Prerelease && !v.Deprecated
}
