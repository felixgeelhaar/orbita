package domain

import (
	"context"
	"encoding/json"
	"time"

	sharedDomain "github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/google/uuid"
)

// ConnectedCalendar represents a user's connected external calendar.
// Users can connect multiple calendars from different providers.
type ConnectedCalendar struct {
	sharedDomain.BaseEntity
	userID     uuid.UUID
	provider   ProviderType
	calendarID string            // External calendar ID (e.g., "primary", "work@example.com")
	name       string            // Display name for the calendar
	isPrimary  bool              // Primary calendar for imports
	isEnabled  bool              // Whether sync is enabled
	syncPush   bool              // Push Orbita blocks to this calendar
	syncPull   bool              // Pull events from this calendar
	config     map[string]string // Provider-specific configuration (CalDAV URL, etc.)
	lastSyncAt time.Time         // Last successful sync time
}

// NewConnectedCalendar creates a new connected calendar.
func NewConnectedCalendar(
	userID uuid.UUID,
	provider ProviderType,
	calendarID string,
	name string,
) *ConnectedCalendar {
	return &ConnectedCalendar{
		BaseEntity: sharedDomain.NewBaseEntity(),
		userID:     userID,
		provider:   provider,
		calendarID: calendarID,
		name:       name,
		isPrimary:  false,
		isEnabled:  true,
		syncPush:   true,
		syncPull:   false,
		config:     make(map[string]string),
		lastSyncAt: time.Time{},
	}
}

// Getters
func (c *ConnectedCalendar) UserID() uuid.UUID       { return c.userID }
func (c *ConnectedCalendar) Provider() ProviderType  { return c.provider }
func (c *ConnectedCalendar) CalendarID() string      { return c.calendarID }
func (c *ConnectedCalendar) Name() string            { return c.name }
func (c *ConnectedCalendar) IsPrimary() bool         { return c.isPrimary }
func (c *ConnectedCalendar) IsEnabled() bool         { return c.isEnabled }
func (c *ConnectedCalendar) SyncPush() bool          { return c.syncPush }
func (c *ConnectedCalendar) SyncPull() bool          { return c.syncPull }
func (c *ConnectedCalendar) Config() map[string]string { return c.config }
func (c *ConnectedCalendar) LastSyncAt() time.Time   { return c.lastSyncAt }

// ConfigValue returns a specific configuration value.
func (c *ConnectedCalendar) ConfigValue(key string) string {
	if c.config == nil {
		return ""
	}
	return c.config[key]
}

// ConfigJSON returns the config as a JSON string for persistence.
func (c *ConnectedCalendar) ConfigJSON() string {
	if c.config == nil || len(c.config) == 0 {
		return "{}"
	}
	data, err := json.Marshal(c.config)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// SetConfig sets a configuration value.
func (c *ConnectedCalendar) SetConfig(key, value string) {
	if c.config == nil {
		c.config = make(map[string]string)
	}
	c.config[key] = value
	c.Touch()
}

// SetName updates the calendar display name.
func (c *ConnectedCalendar) SetName(name string) {
	c.name = name
	c.Touch()
}

// SetPrimary marks this calendar as the primary for imports.
func (c *ConnectedCalendar) SetPrimary(primary bool) {
	c.isPrimary = primary
	c.Touch()
}

// SetEnabled enables or disables sync for this calendar.
func (c *ConnectedCalendar) SetEnabled(enabled bool) {
	c.isEnabled = enabled
	c.Touch()
}

// SetSyncPush enables or disables pushing Orbita blocks to this calendar.
func (c *ConnectedCalendar) SetSyncPush(push bool) {
	c.syncPush = push
	c.Touch()
}

// SetSyncPull enables or disables pulling events from this calendar.
func (c *ConnectedCalendar) SetSyncPull(pull bool) {
	c.syncPull = pull
	c.Touch()
}

// MarkSynced records a successful sync.
func (c *ConnectedCalendar) MarkSynced() {
	c.lastSyncAt = time.Now().UTC()
	c.Touch()
}

// HasSynced returns true if at least one sync has occurred.
func (c *ConnectedCalendar) HasSynced() bool {
	return !c.lastSyncAt.IsZero()
}

// CalDAV configuration keys
const (
	ConfigCalDAVURL      = "caldav_url"
	ConfigCalDAVUsername = "caldav_username"
	// Password is stored encrypted in oauth_tokens table, not here
)

// SetCalDAVConfig sets CalDAV-specific configuration.
func (c *ConnectedCalendar) SetCalDAVConfig(url, username string) {
	c.SetConfig(ConfigCalDAVURL, url)
	c.SetConfig(ConfigCalDAVUsername, username)
}

// CalDAVURL returns the CalDAV server URL.
func (c *ConnectedCalendar) CalDAVURL() string {
	return c.ConfigValue(ConfigCalDAVURL)
}

// CalDAVUsername returns the CalDAV username.
func (c *ConnectedCalendar) CalDAVUsername() string {
	return c.ConfigValue(ConfigCalDAVUsername)
}

// RehydrateConnectedCalendar recreates a connected calendar from persisted data.
func RehydrateConnectedCalendar(
	id uuid.UUID,
	userID uuid.UUID,
	provider ProviderType,
	calendarID string,
	name string,
	isPrimary bool,
	isEnabled bool,
	syncPush bool,
	syncPull bool,
	configJSON string,
	lastSyncAt time.Time,
	createdAt, updatedAt time.Time,
) *ConnectedCalendar {
	config := make(map[string]string)
	if configJSON != "" && configJSON != "{}" {
		_ = json.Unmarshal([]byte(configJSON), &config)
	}

	return &ConnectedCalendar{
		BaseEntity: sharedDomain.RehydrateBaseEntity(id, createdAt, updatedAt),
		userID:     userID,
		provider:   provider,
		calendarID: calendarID,
		name:       name,
		isPrimary:  isPrimary,
		isEnabled:  isEnabled,
		syncPush:   syncPush,
		syncPull:   syncPull,
		config:     config,
		lastSyncAt: lastSyncAt,
	}
}

// ConnectedCalendarRepository defines the interface for connected calendar persistence.
type ConnectedCalendarRepository interface {
	// Save persists a connected calendar (create or update).
	Save(ctx context.Context, calendar *ConnectedCalendar) error

	// FindByID finds a connected calendar by ID.
	FindByID(ctx context.Context, id uuid.UUID) (*ConnectedCalendar, error)

	// FindByUserAndProvider finds all calendars for a user from a specific provider.
	FindByUserAndProvider(ctx context.Context, userID uuid.UUID, provider ProviderType) ([]*ConnectedCalendar, error)

	// FindByUserProviderAndCalendar finds a specific calendar connection.
	FindByUserProviderAndCalendar(ctx context.Context, userID uuid.UUID, provider ProviderType, calendarID string) (*ConnectedCalendar, error)

	// FindByUser finds all connected calendars for a user.
	FindByUser(ctx context.Context, userID uuid.UUID) ([]*ConnectedCalendar, error)

	// FindPrimaryForUser finds the user's primary calendar for imports.
	FindPrimaryForUser(ctx context.Context, userID uuid.UUID) (*ConnectedCalendar, error)

	// FindEnabledPushCalendars finds all enabled calendars with push sync for a user.
	FindEnabledPushCalendars(ctx context.Context, userID uuid.UUID) ([]*ConnectedCalendar, error)

	// FindEnabledPullCalendars finds all enabled calendars with pull sync for a user.
	FindEnabledPullCalendars(ctx context.Context, userID uuid.UUID) ([]*ConnectedCalendar, error)

	// ClearPrimaryForUser removes the primary flag from all user calendars.
	// Used before setting a new primary calendar.
	ClearPrimaryForUser(ctx context.Context, userID uuid.UUID) error

	// Delete removes a connected calendar.
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteByUserAndProvider removes all calendars for a user from a specific provider.
	DeleteByUserAndProvider(ctx context.Context, userID uuid.UUID, provider ProviderType) error
}
