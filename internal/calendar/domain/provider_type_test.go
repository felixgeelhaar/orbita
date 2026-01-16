package domain_test

import (
	"testing"

	"github.com/felixgeelhaar/orbita/internal/calendar/domain"
	"github.com/stretchr/testify/assert"
)

func TestProviderType_String(t *testing.T) {
	tests := []struct {
		provider domain.ProviderType
		expected string
	}{
		{domain.ProviderGoogle, "google"},
		{domain.ProviderMicrosoft, "microsoft"},
		{domain.ProviderApple, "apple"},
		{domain.ProviderCalDAV, "caldav"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.provider.String())
		})
	}
}

func TestProviderType_IsValid(t *testing.T) {
	tests := []struct {
		provider domain.ProviderType
		valid    bool
	}{
		{domain.ProviderGoogle, true},
		{domain.ProviderMicrosoft, true},
		{domain.ProviderApple, true},
		{domain.ProviderCalDAV, true},
		{domain.ProviderType("unknown"), false},
		{domain.ProviderType(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.provider.IsValid())
		})
	}
}

func TestProviderType_RequiresOAuth(t *testing.T) {
	tests := []struct {
		provider domain.ProviderType
		oauth    bool
	}{
		{domain.ProviderGoogle, true},
		{domain.ProviderMicrosoft, true},
		{domain.ProviderApple, false},
		{domain.ProviderCalDAV, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			assert.Equal(t, tt.oauth, tt.provider.RequiresOAuth())
		})
	}
}

func TestProviderType_RequiresCalDAV(t *testing.T) {
	tests := []struct {
		provider domain.ProviderType
		caldav   bool
	}{
		{domain.ProviderGoogle, false},
		{domain.ProviderMicrosoft, false},
		{domain.ProviderApple, true},
		{domain.ProviderCalDAV, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			assert.Equal(t, tt.caldav, tt.provider.RequiresCalDAV())
		})
	}
}

func TestProviderType_DisplayName(t *testing.T) {
	tests := []struct {
		provider domain.ProviderType
		display  string
	}{
		{domain.ProviderGoogle, "Google Calendar"},
		{domain.ProviderMicrosoft, "Microsoft Outlook"},
		{domain.ProviderApple, "Apple Calendar"},
		{domain.ProviderCalDAV, "CalDAV"},
		{domain.ProviderType("custom"), "custom"},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			assert.Equal(t, tt.display, tt.provider.DisplayName())
		})
	}
}

func TestAllProviderTypes(t *testing.T) {
	providers := domain.AllProviderTypes()

	assert.Len(t, providers, 4)
	assert.Contains(t, providers, domain.ProviderGoogle)
	assert.Contains(t, providers, domain.ProviderMicrosoft)
	assert.Contains(t, providers, domain.ProviderApple)
	assert.Contains(t, providers, domain.ProviderCalDAV)
}
