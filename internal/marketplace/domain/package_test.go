package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestPackageType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		pt       PackageType
		expected bool
	}{
		{"orbit type is valid", PackageTypeOrbit, true},
		{"engine type is valid", PackageTypeEngine, true},
		{"empty type is invalid", PackageType(""), false},
		{"unknown type is invalid", PackageType("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.pt.IsValid())
		})
	}
}

func TestNewPackage(t *testing.T) {
	pkg := NewPackage("test.orbit", PackageTypeOrbit, "Test Orbit", "A test orbit for testing")

	assert.NotEqual(t, uuid.Nil, pkg.ID)
	assert.Equal(t, "test.orbit", pkg.PackageID)
	assert.Equal(t, PackageTypeOrbit, pkg.Type)
	assert.Equal(t, "Test Orbit", pkg.Name)
	assert.Equal(t, "A test orbit for testing", pkg.Description)
	assert.Empty(t, pkg.Author)
	assert.Empty(t, pkg.Homepage)
	assert.Empty(t, pkg.License)
	assert.Empty(t, pkg.Tags)
	assert.Empty(t, pkg.LatestVersion)
	assert.Equal(t, int64(0), pkg.Downloads)
	assert.Equal(t, 0.0, pkg.Rating)
	assert.Equal(t, 0, pkg.RatingCount)
	assert.False(t, pkg.Verified)
	assert.False(t, pkg.Featured)
	assert.Equal(t, uuid.Nil, pkg.PublisherID)
	assert.False(t, pkg.CreatedAt.IsZero())
	assert.False(t, pkg.UpdatedAt.IsZero())
}

func TestPackage_SetAuthor(t *testing.T) {
	pkg := NewPackage("test.orbit", PackageTypeOrbit, "Test", "Description")
	beforeUpdate := pkg.UpdatedAt

	time.Sleep(1 * time.Millisecond)
	pkg.SetAuthor("John Doe")

	assert.Equal(t, "John Doe", pkg.Author)
	assert.True(t, pkg.UpdatedAt.After(beforeUpdate))
}

func TestPackage_SetHomepage(t *testing.T) {
	pkg := NewPackage("test.orbit", PackageTypeOrbit, "Test", "Description")
	beforeUpdate := pkg.UpdatedAt

	time.Sleep(1 * time.Millisecond)
	pkg.SetHomepage("https://example.com")

	assert.Equal(t, "https://example.com", pkg.Homepage)
	assert.True(t, pkg.UpdatedAt.After(beforeUpdate))
}

func TestPackage_SetLicense(t *testing.T) {
	pkg := NewPackage("test.orbit", PackageTypeOrbit, "Test", "Description")
	beforeUpdate := pkg.UpdatedAt

	time.Sleep(1 * time.Millisecond)
	pkg.SetLicense("MIT")

	assert.Equal(t, "MIT", pkg.License)
	assert.True(t, pkg.UpdatedAt.After(beforeUpdate))
}

func TestPackage_SetTags(t *testing.T) {
	pkg := NewPackage("test.orbit", PackageTypeOrbit, "Test", "Description")
	beforeUpdate := pkg.UpdatedAt

	time.Sleep(1 * time.Millisecond)
	pkg.SetTags([]string{"productivity", "calendar"})

	assert.Equal(t, []string{"productivity", "calendar"}, pkg.Tags)
	assert.True(t, pkg.UpdatedAt.After(beforeUpdate))
}

func TestPackage_SetLatestVersion(t *testing.T) {
	pkg := NewPackage("test.orbit", PackageTypeOrbit, "Test", "Description")
	beforeUpdate := pkg.UpdatedAt

	time.Sleep(1 * time.Millisecond)
	pkg.SetLatestVersion("1.0.0")

	assert.Equal(t, "1.0.0", pkg.LatestVersion)
	assert.True(t, pkg.UpdatedAt.After(beforeUpdate))
}

func TestPackage_SetVerified(t *testing.T) {
	pkg := NewPackage("test.orbit", PackageTypeOrbit, "Test", "Description")
	beforeUpdate := pkg.UpdatedAt

	time.Sleep(1 * time.Millisecond)
	pkg.SetVerified(true)

	assert.True(t, pkg.Verified)
	assert.True(t, pkg.UpdatedAt.After(beforeUpdate))
}

func TestPackage_SetFeatured(t *testing.T) {
	pkg := NewPackage("test.orbit", PackageTypeOrbit, "Test", "Description")
	beforeUpdate := pkg.UpdatedAt

	time.Sleep(1 * time.Millisecond)
	pkg.SetFeatured(true)

	assert.True(t, pkg.Featured)
	assert.True(t, pkg.UpdatedAt.After(beforeUpdate))
}

func TestPackage_SetPublisher(t *testing.T) {
	pkg := NewPackage("test.orbit", PackageTypeOrbit, "Test", "Description")
	beforeUpdate := pkg.UpdatedAt
	publisherID := uuid.New()

	time.Sleep(1 * time.Millisecond)
	pkg.SetPublisher(publisherID)

	assert.Equal(t, publisherID, pkg.PublisherID)
	assert.True(t, pkg.UpdatedAt.After(beforeUpdate))
}

func TestPackage_IncrementDownloads(t *testing.T) {
	pkg := NewPackage("test.orbit", PackageTypeOrbit, "Test", "Description")
	beforeUpdate := pkg.UpdatedAt

	time.Sleep(1 * time.Millisecond)
	pkg.IncrementDownloads()

	assert.Equal(t, int64(1), pkg.Downloads)
	assert.True(t, pkg.UpdatedAt.After(beforeUpdate))

	pkg.IncrementDownloads()
	assert.Equal(t, int64(2), pkg.Downloads)
}

func TestPackage_SetRating(t *testing.T) {
	pkg := NewPackage("test.orbit", PackageTypeOrbit, "Test", "Description")
	beforeUpdate := pkg.UpdatedAt

	time.Sleep(1 * time.Millisecond)
	pkg.SetRating(4.5, 10)

	assert.Equal(t, 4.5, pkg.Rating)
	assert.Equal(t, 10, pkg.RatingCount)
	assert.True(t, pkg.UpdatedAt.After(beforeUpdate))
}

func TestPackageFilter_Defaults(t *testing.T) {
	filter := PackageFilter{}

	assert.Equal(t, 0, filter.Offset)
	assert.Equal(t, 0, filter.Limit)
	assert.Nil(t, filter.Type)
	assert.Nil(t, filter.Tags)
	assert.Nil(t, filter.Verified)
	assert.Nil(t, filter.Featured)
}

func TestSortField_Values(t *testing.T) {
	assert.Equal(t, PackageSortField("created_at"), SortByCreatedAt)
	assert.Equal(t, PackageSortField("updated_at"), SortByUpdatedAt)
	assert.Equal(t, PackageSortField("downloads"), SortByDownloads)
	assert.Equal(t, PackageSortField("rating"), SortByRating)
	assert.Equal(t, PackageSortField("name"), SortByName)
}

func TestSortOrder_Values(t *testing.T) {
	assert.Equal(t, SortOrder("asc"), SortAsc)
	assert.Equal(t, SortOrder("desc"), SortDesc)
}
