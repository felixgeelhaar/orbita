package domain

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	sharedDomain "github.com/felixgeelhaar/orbita/internal/shared/domain"
	"github.com/google/uuid"
)

// Domain errors for ConnectedCalendar validation.
var (
	ErrEmptyUserID     = errors.New("user ID cannot be empty")
	ErrInvalidProvider = errors.New("invalid provider type")
	ErrEmptyCalendarID = errors.New("calendar ID cannot be empty")
	ErrEmptyName       = errors.New("calendar name cannot be empty")
)

// ConnectedCalendar represents a user's connected external calendar.
// Users can connect multiple calendars from different providers.
// This is an Aggregate Root that publishes domain events.
type ConnectedCalendar struct {
	sharedDomain.BaseAggregateRoot
	userID     uuid.UUID
	provider   ProviderType
	calendarID string            // External calendar ID (e.g., "primary", "work@example.com")
	name       string            // Display name for the calendar
	isPrimary  bool              // Primary calendar for imports
	isEnabled  bool              // Whether sync is enabled
	syncPush   bool              // Push Orbita blocks to this calendar
	syncPull   bool              // Pull events from this calendar
	config     map[string]string // Provider-specific configuration
	lastSyncAt time.Time         // Last successful sync time
}

// NewConnectedCalendar creates a new connected calendar and records a CalendarConnectedEvent.
// It validates all inputs and returns an error if any are invalid.
func NewConnectedCalendar(
	userID uuid.UUID,
	provider ProviderType,
	calendarID string,
	name string,
) (*ConnectedCalendar, error) {
	// Validate inputs
	if userID == uuid.Nil {
		return nil, ErrEmptyUserID
	}
	if !provider.IsValid() {
		return nil, ErrInvalidProvider
	}
	if strings.TrimSpace(calendarID) == "" {
		return nil, ErrEmptyCalendarID
	}
	if strings.TrimSpace(name) == "" {
		return nil, ErrEmptyName
	}

	c := &ConnectedCalendar{
		BaseAggregateRoot: sharedDomain.NewBaseAggregateRoot(),
		userID:            userID,
		provider:          provider,
		calendarID:        calendarID,
		name:              name,
		isPrimary:         false,
		isEnabled:         true,
		syncPush:          true,
		syncPull:          false,
		config:            make(map[string]string),
		lastSyncAt:        time.Time{},
	}

	// Record domain event
	c.AddDomainEvent(NewCalendarConnectedEvent(
		c.ID(),
		userID,
		provider,
		calendarID,
		name,
		false,
	))

	return c, nil
}

// Getters
func (c *ConnectedCalendar) UserID() uuid.UUID        { return c.userID }
func (c *ConnectedCalendar) Provider() ProviderType   { return c.provider }
func (c *ConnectedCalendar) CalendarID() string       { return c.calendarID }
func (c *ConnectedCalendar) Name() string             { return c.name }
func (c *ConnectedCalendar) IsPrimary() bool          { return c.isPrimary }
func (c *ConnectedCalendar) IsEnabled() bool          { return c.isEnabled }
func (c *ConnectedCalendar) SyncPush() bool           { return c.syncPush }
func (c *ConnectedCalendar) SyncPull() bool           { return c.syncPull }
func (c *ConnectedCalendar) Config() map[string]string {
	if c.config == nil {
		return nil
	}
	copy := make(map[string]string, len(c.config))
	for k, v := range c.config {
		copy[k] = v
	}
	return copy
}
func (c *ConnectedCalendar) LastSyncAt() time.Time    { return c.lastSyncAt }

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
	if c.name != name {
		c.name = name
		c.Touch()
		c.AddDomainEvent(NewCalendarUpdatedEvent(
			c.ID(),
			c.userID,
			c.provider,
			c.calendarID,
			[]string{"name"},
		))
	}
}

// SetPrimary marks this calendar as the primary for imports.
// The previousPrimaryID should be provided when changing primary status
// to enable proper event tracking.
func (c *ConnectedCalendar) SetPrimary(primary bool, previousPrimaryID *uuid.UUID) {
	if c.isPrimary != primary {
		c.isPrimary = primary
		c.Touch()

		if primary {
			c.AddDomainEvent(NewCalendarPrimarySetEvent(
				c.ID(),
				c.userID,
				c.provider,
				c.calendarID,
				previousPrimaryID,
			))
		}
	}
}

// ClearPrimary clears the primary flag without recording an event.
// Used internally when another calendar becomes primary.
func (c *ConnectedCalendar) ClearPrimary() {
	if c.isPrimary {
		c.isPrimary = false
		c.Touch()
	}
}

// SetEnabled enables or disables sync for this calendar.
func (c *ConnectedCalendar) SetEnabled(enabled bool) {
	if c.isEnabled != enabled {
		c.isEnabled = enabled
		c.Touch()
		c.AddDomainEvent(NewCalendarUpdatedEvent(
			c.ID(),
			c.userID,
			c.provider,
			c.calendarID,
			[]string{"enabled"},
		))
	}
}

// SetSyncPush enables or disables pushing Orbita blocks to this calendar.
func (c *ConnectedCalendar) SetSyncPush(push bool) {
	if c.syncPush != push {
		c.syncPush = push
		c.Touch()
		c.AddDomainEvent(NewCalendarUpdatedEvent(
			c.ID(),
			c.userID,
			c.provider,
			c.calendarID,
			[]string{"sync_push"},
		))
	}
}

// SetSyncPull enables or disables pulling events from this calendar.
func (c *ConnectedCalendar) SetSyncPull(pull bool) {
	if c.syncPull != pull {
		c.syncPull = pull
		c.Touch()
		c.AddDomainEvent(NewCalendarUpdatedEvent(
			c.ID(),
			c.userID,
			c.provider,
			c.calendarID,
			[]string{"sync_pull"},
		))
	}
}

// MarkSynced records a successful sync with results.
func (c *ConnectedCalendar) MarkSynced(created, updated, deleted, failed int) {
	c.lastSyncAt = time.Now().UTC()
	c.Touch()
	c.AddDomainEvent(NewCalendarSyncedEvent(
		c.ID(),
		c.userID,
		c.provider,
		c.calendarID,
		created,
		updated,
		deleted,
		failed,
	))
}

// MarkSyncedSimple records a successful sync without detailed results.
func (c *ConnectedCalendar) MarkSyncedSimple() {
	c.lastSyncAt = time.Now().UTC()
	c.Touch()
}

// HasSynced returns true if at least one sync has occurred.
func (c *ConnectedCalendar) HasSynced() bool {
	return !c.lastSyncAt.IsZero()
}

// MarkDisconnected records that this calendar is being disconnected.
func (c *ConnectedCalendar) MarkDisconnected() {
	c.AddDomainEvent(NewCalendarDisconnectedEvent(
		c.ID(),
		c.userID,
		c.provider,
		c.calendarID,
	))
}

// CalDAV configuration keys - these are infrastructure concerns but
// kept here for backward compatibility. Ideally these would be in
// the CalDAV adapter.
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
// This does NOT record domain events as it's rehydrating existing state.
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
	version int,
) *ConnectedCalendar {
	config := make(map[string]string)
	if configJSON != "" && configJSON != "{}" {
		_ = json.Unmarshal([]byte(configJSON), &config)
	}

	baseEntity := sharedDomain.RehydrateBaseEntity(id, createdAt, updatedAt)
	baseAggregate := sharedDomain.RehydrateBaseAggregateRoot(baseEntity, version)

	return &ConnectedCalendar{
		BaseAggregateRoot: baseAggregate,
		userID:            userID,
		provider:          provider,
		calendarID:        calendarID,
		name:              name,
		isPrimary:         isPrimary,
		isEnabled:         isEnabled,
		syncPush:          syncPush,
		syncPull:          syncPull,
		config:            config,
		lastSyncAt:        lastSyncAt,
	}
}

// ConnectedCalendarRepository defines the interface for connected calendar persistence.
// Note: ClearPrimaryForUser has been removed - use the application service to handle
// primary calendar changes atomically.
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

	// Delete removes a connected calendar.
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteByUserAndProvider removes all calendars for a user from a specific provider.
	DeleteByUserAndProvider(ctx context.Context, userID uuid.UUID, provider ProviderType) error
}
