package domain

import (
	"time"

	"github.com/google/uuid"
)

// Publisher represents a marketplace publisher (developer or organization).
type Publisher struct {
	// ID is the unique identifier for this publisher.
	ID uuid.UUID

	// Name is the publisher's display name.
	Name string

	// Slug is the URL-friendly identifier (e.g., "acme-corp").
	Slug string

	// Email is the publisher's contact email.
	Email string

	// Website is the publisher's website URL.
	Website string

	// Description is a brief description of the publisher.
	Description string

	// Verified indicates if this is a verified publisher.
	Verified bool

	// AvatarURL is the URL to the publisher's avatar/logo.
	AvatarURL string

	// PackageCount is the number of packages published.
	PackageCount int

	// TotalDownloads is the total downloads across all packages.
	TotalDownloads int64

	// UserID is the owning user's ID (if individual publisher).
	UserID *uuid.UUID

	// CreatedAt is when the publisher was created.
	CreatedAt time.Time

	// UpdatedAt is when the publisher was last updated.
	UpdatedAt time.Time
}

// NewPublisher creates a new marketplace publisher.
func NewPublisher(name, slug, email string) *Publisher {
	now := time.Now().UTC()
	return &Publisher{
		ID:        uuid.New(),
		Name:      name,
		Slug:      slug,
		Email:     email,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// SetWebsite sets the publisher's website.
func (p *Publisher) SetWebsite(website string) {
	p.Website = website
	p.UpdatedAt = time.Now().UTC()
}

// SetDescription sets the publisher's description.
func (p *Publisher) SetDescription(description string) {
	p.Description = description
	p.UpdatedAt = time.Now().UTC()
}

// SetVerified sets the verified status.
func (p *Publisher) SetVerified(verified bool) {
	p.Verified = verified
	p.UpdatedAt = time.Now().UTC()
}

// SetAvatarURL sets the avatar URL.
func (p *Publisher) SetAvatarURL(avatarURL string) {
	p.AvatarURL = avatarURL
	p.UpdatedAt = time.Now().UTC()
}

// SetUserID sets the owning user ID.
func (p *Publisher) SetUserID(userID *uuid.UUID) {
	p.UserID = userID
	p.UpdatedAt = time.Now().UTC()
}

// IncrementPackageCount increments the package count.
func (p *Publisher) IncrementPackageCount() {
	p.PackageCount++
	p.UpdatedAt = time.Now().UTC()
}

// DecrementPackageCount decrements the package count.
func (p *Publisher) DecrementPackageCount() {
	if p.PackageCount > 0 {
		p.PackageCount--
	}
	p.UpdatedAt = time.Now().UTC()
}

// AddDownloads adds to the total download count.
func (p *Publisher) AddDownloads(count int64) {
	p.TotalDownloads += count
	p.UpdatedAt = time.Now().UTC()
}

// UpdateName updates the publisher's name.
func (p *Publisher) UpdateName(name string) {
	p.Name = name
	p.UpdatedAt = time.Now().UTC()
}

// UpdateEmail updates the publisher's email.
func (p *Publisher) UpdateEmail(email string) {
	p.Email = email
	p.UpdatedAt = time.Now().UTC()
}
