// Package domain provides the core entities for the Orbita marketplace.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// PackageType represents the type of marketplace package.
type PackageType string

const (
	// PackageTypeOrbit is an orbit module package.
	PackageTypeOrbit PackageType = "orbit"

	// PackageTypeEngine is an engine plugin package.
	PackageTypeEngine PackageType = "engine"
)

// IsValid checks if the package type is valid.
func (pt PackageType) IsValid() bool {
	return pt == PackageTypeOrbit || pt == PackageTypeEngine
}

// Package represents a marketplace package (orbit or engine).
type Package struct {
	// ID is the unique identifier for this package.
	ID uuid.UUID

	// PackageID is the unique package identifier (e.g., "acme.priority-engine").
	PackageID string

	// Type is the package type (orbit or engine).
	Type PackageType

	// Name is the human-readable package name.
	Name string

	// Description is a brief description of what the package does.
	Description string

	// Author is the package author/publisher name.
	Author string

	// Homepage is the URL to the package documentation or homepage.
	Homepage string

	// License is the license under which the package is distributed.
	License string

	// Tags are searchable tags for the package.
	Tags []string

	// LatestVersion is the most recent stable version.
	LatestVersion string

	// Downloads is the total download count.
	Downloads int64

	// Rating is the average user rating (0-5).
	Rating float64

	// RatingCount is the number of ratings.
	RatingCount int

	// Verified indicates if this is a verified publisher package.
	Verified bool

	// Featured indicates if this package is featured in the marketplace.
	Featured bool

	// PublisherID is the ID of the publisher who owns this package.
	PublisherID uuid.UUID

	// CreatedAt is when the package was first published.
	CreatedAt time.Time

	// UpdatedAt is when the package was last updated.
	UpdatedAt time.Time
}

// NewPackage creates a new marketplace package.
func NewPackage(packageID string, packageType PackageType, name, description string) *Package {
	now := time.Now().UTC()
	return &Package{
		ID:          uuid.New(),
		PackageID:   packageID,
		Type:        packageType,
		Name:        name,
		Description: description,
		Tags:        []string{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// SetAuthor sets the package author.
func (p *Package) SetAuthor(author string) {
	p.Author = author
	p.UpdatedAt = time.Now().UTC()
}

// SetHomepage sets the package homepage.
func (p *Package) SetHomepage(homepage string) {
	p.Homepage = homepage
	p.UpdatedAt = time.Now().UTC()
}

// SetLicense sets the package license.
func (p *Package) SetLicense(license string) {
	p.License = license
	p.UpdatedAt = time.Now().UTC()
}

// SetTags sets the package tags.
func (p *Package) SetTags(tags []string) {
	p.Tags = tags
	p.UpdatedAt = time.Now().UTC()
}

// SetLatestVersion updates the latest version.
func (p *Package) SetLatestVersion(version string) {
	p.LatestVersion = version
	p.UpdatedAt = time.Now().UTC()
}

// IncrementDownloads increments the download count.
func (p *Package) IncrementDownloads() {
	p.Downloads++
	p.UpdatedAt = time.Now().UTC()
}

// SetPublisher sets the publisher ID.
func (p *Package) SetPublisher(publisherID uuid.UUID) {
	p.PublisherID = publisherID
	p.UpdatedAt = time.Now().UTC()
}

// SetRating updates the rating.
func (p *Package) SetRating(rating float64, count int) {
	p.Rating = rating
	p.RatingCount = count
	p.UpdatedAt = time.Now().UTC()
}

// SetVerified sets the verified status.
func (p *Package) SetVerified(verified bool) {
	p.Verified = verified
	p.UpdatedAt = time.Now().UTC()
}

// SetFeatured sets the featured status.
func (p *Package) SetFeatured(featured bool) {
	p.Featured = featured
	p.UpdatedAt = time.Now().UTC()
}
