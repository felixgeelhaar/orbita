package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewPublisher(t *testing.T) {
	publisher := NewPublisher("ACME Corp", "acme", "contact@acme.com")

	assert.NotEqual(t, uuid.Nil, publisher.ID)
	assert.Equal(t, "ACME Corp", publisher.Name)
	assert.Equal(t, "acme", publisher.Slug)
	assert.Equal(t, "contact@acme.com", publisher.Email)
	assert.Empty(t, publisher.Website)
	assert.Empty(t, publisher.Description)
	assert.False(t, publisher.Verified)
	assert.Empty(t, publisher.AvatarURL)
	assert.Equal(t, 0, publisher.PackageCount)
	assert.Equal(t, int64(0), publisher.TotalDownloads)
	assert.Nil(t, publisher.UserID)
	assert.False(t, publisher.CreatedAt.IsZero())
	assert.False(t, publisher.UpdatedAt.IsZero())
}

func TestPublisher_SetWebsite(t *testing.T) {
	publisher := NewPublisher("ACME Corp", "acme", "contact@acme.com")
	beforeUpdate := publisher.UpdatedAt

	time.Sleep(1 * time.Millisecond)
	publisher.SetWebsite("https://acme.com")

	assert.Equal(t, "https://acme.com", publisher.Website)
	assert.True(t, publisher.UpdatedAt.After(beforeUpdate))
}

func TestPublisher_SetDescription(t *testing.T) {
	publisher := NewPublisher("ACME Corp", "acme", "contact@acme.com")
	beforeUpdate := publisher.UpdatedAt

	time.Sleep(1 * time.Millisecond)
	publisher.SetDescription("Leading provider of productivity tools")

	assert.Equal(t, "Leading provider of productivity tools", publisher.Description)
	assert.True(t, publisher.UpdatedAt.After(beforeUpdate))
}

func TestPublisher_SetVerified(t *testing.T) {
	publisher := NewPublisher("ACME Corp", "acme", "contact@acme.com")
	beforeUpdate := publisher.UpdatedAt

	time.Sleep(1 * time.Millisecond)
	publisher.SetVerified(true)

	assert.True(t, publisher.Verified)
	assert.True(t, publisher.UpdatedAt.After(beforeUpdate))
}

func TestPublisher_SetAvatarURL(t *testing.T) {
	publisher := NewPublisher("ACME Corp", "acme", "contact@acme.com")
	beforeUpdate := publisher.UpdatedAt

	time.Sleep(1 * time.Millisecond)
	publisher.SetAvatarURL("https://cdn.example.com/avatars/acme.png")

	assert.Equal(t, "https://cdn.example.com/avatars/acme.png", publisher.AvatarURL)
	assert.True(t, publisher.UpdatedAt.After(beforeUpdate))
}

func TestPublisher_SetUserID(t *testing.T) {
	publisher := NewPublisher("ACME Corp", "acme", "contact@acme.com")
	beforeUpdate := publisher.UpdatedAt
	userID := uuid.New()

	time.Sleep(1 * time.Millisecond)
	publisher.SetUserID(&userID)

	assert.Equal(t, &userID, publisher.UserID)
	assert.True(t, publisher.UpdatedAt.After(beforeUpdate))
}

func TestPublisher_SetUserID_Nil(t *testing.T) {
	publisher := NewPublisher("ACME Corp", "acme", "contact@acme.com")
	userID := uuid.New()
	publisher.SetUserID(&userID)
	beforeUpdate := publisher.UpdatedAt

	time.Sleep(1 * time.Millisecond)
	publisher.SetUserID(nil)

	assert.Nil(t, publisher.UserID)
	assert.True(t, publisher.UpdatedAt.After(beforeUpdate))
}

func TestPublisher_IncrementPackageCount(t *testing.T) {
	publisher := NewPublisher("ACME Corp", "acme", "contact@acme.com")
	beforeUpdate := publisher.UpdatedAt

	time.Sleep(1 * time.Millisecond)
	publisher.IncrementPackageCount()

	assert.Equal(t, 1, publisher.PackageCount)
	assert.True(t, publisher.UpdatedAt.After(beforeUpdate))

	publisher.IncrementPackageCount()
	assert.Equal(t, 2, publisher.PackageCount)
}

func TestPublisher_DecrementPackageCount(t *testing.T) {
	publisher := NewPublisher("ACME Corp", "acme", "contact@acme.com")
	publisher.PackageCount = 5
	beforeUpdate := publisher.UpdatedAt

	time.Sleep(1 * time.Millisecond)
	publisher.DecrementPackageCount()

	assert.Equal(t, 4, publisher.PackageCount)
	assert.True(t, publisher.UpdatedAt.After(beforeUpdate))
}

func TestPublisher_DecrementPackageCount_NoNegative(t *testing.T) {
	publisher := NewPublisher("ACME Corp", "acme", "contact@acme.com")
	publisher.PackageCount = 0

	publisher.DecrementPackageCount()

	assert.Equal(t, 0, publisher.PackageCount)
}

func TestPublisher_AddDownloads(t *testing.T) {
	publisher := NewPublisher("ACME Corp", "acme", "contact@acme.com")
	beforeUpdate := publisher.UpdatedAt

	time.Sleep(1 * time.Millisecond)
	publisher.AddDownloads(100)

	assert.Equal(t, int64(100), publisher.TotalDownloads)
	assert.True(t, publisher.UpdatedAt.After(beforeUpdate))

	publisher.AddDownloads(50)
	assert.Equal(t, int64(150), publisher.TotalDownloads)
}

func TestPublisher_UpdateName(t *testing.T) {
	publisher := NewPublisher("ACME Corp", "acme", "contact@acme.com")
	beforeUpdate := publisher.UpdatedAt

	time.Sleep(1 * time.Millisecond)
	publisher.UpdateName("ACME Corporation")

	assert.Equal(t, "ACME Corporation", publisher.Name)
	assert.True(t, publisher.UpdatedAt.After(beforeUpdate))
}

func TestPublisher_UpdateEmail(t *testing.T) {
	publisher := NewPublisher("ACME Corp", "acme", "contact@acme.com")
	beforeUpdate := publisher.UpdatedAt

	time.Sleep(1 * time.Millisecond)
	publisher.UpdateEmail("support@acme.com")

	assert.Equal(t, "support@acme.com", publisher.Email)
	assert.True(t, publisher.UpdatedAt.After(beforeUpdate))
}
