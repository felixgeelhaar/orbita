package domain

// ProviderType represents a calendar provider type.
type ProviderType string

const (
	// ProviderGoogle is Google Calendar (OAuth2 + Google Calendar API).
	ProviderGoogle ProviderType = "google"
	// ProviderMicrosoft is Microsoft Outlook/365 (OAuth2 + Microsoft Graph API).
	ProviderMicrosoft ProviderType = "microsoft"
	// ProviderApple is Apple Calendar (CalDAV with app-specific password).
	ProviderApple ProviderType = "apple"
	// ProviderCalDAV is generic CalDAV (Fastmail, Nextcloud, self-hosted).
	ProviderCalDAV ProviderType = "caldav"
)

// String returns the string representation of the provider type.
func (p ProviderType) String() string {
	return string(p)
}

// IsValid returns true if the provider type is recognized.
func (p ProviderType) IsValid() bool {
	switch p {
	case ProviderGoogle, ProviderMicrosoft, ProviderApple, ProviderCalDAV:
		return true
	default:
		return false
	}
}

// RequiresOAuth returns true if the provider uses OAuth2 for authentication.
func (p ProviderType) RequiresOAuth() bool {
	switch p {
	case ProviderGoogle, ProviderMicrosoft:
		return true
	default:
		return false
	}
}

// RequiresCalDAV returns true if the provider uses CalDAV protocol.
func (p ProviderType) RequiresCalDAV() bool {
	switch p {
	case ProviderApple, ProviderCalDAV:
		return true
	default:
		return false
	}
}

// DisplayName returns a human-readable name for the provider.
func (p ProviderType) DisplayName() string {
	switch p {
	case ProviderGoogle:
		return "Google Calendar"
	case ProviderMicrosoft:
		return "Microsoft Outlook"
	case ProviderApple:
		return "Apple Calendar"
	case ProviderCalDAV:
		return "CalDAV"
	default:
		return string(p)
	}
}

// AllProviderTypes returns all supported provider types.
func AllProviderTypes() []ProviderType {
	return []ProviderType{
		ProviderGoogle,
		ProviderMicrosoft,
		ProviderApple,
		ProviderCalDAV,
	}
}
