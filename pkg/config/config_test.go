package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// clearEnvVars clears all Orbita-related environment variables.
func clearEnvVars() {
	envVars := []string{
		"APP_ENV", "LOG_LEVEL", "ORBITA_USER_ID", "ORBITA_ENCRYPTION_KEY",
		"DATABASE_URL", "DATABASE_DRIVER", "SQLITE_PATH", "ORBITA_LOCAL_MODE",
		"REDIS_URL", "RABBITMQ_URL",
		"OUTBOX_POLL_INTERVAL", "OUTBOX_BATCH_SIZE", "OUTBOX_MAX_RETRIES",
		"OUTBOX_STATS_INTERVAL", "OUTBOX_RETENTION_DAYS", "OUTBOX_CLEANUP_INTERVAL",
		"OUTBOX_PROCESSOR_ENABLED", "WORKER_HEALTH_ADDR",
		"OAUTH_PROVIDER", "OAUTH_CLIENT_ID", "OAUTH_CLIENT_SECRET",
		"OAUTH_AUTH_URL", "OAUTH_TOKEN_URL", "OAUTH_REDIRECT_URL", "OAUTH_SCOPES",
		"CALENDAR_DELETE_MISSING", "CALENDAR_ID",
		"CALENDAR_SYNC_ENABLED", "CALENDAR_SYNC_INTERVAL", "CALENDAR_SYNC_LOOK_AHEAD_DAYS",
		"CALENDAR_CONFLICT_STRATEGY", "CALENDAR_AUTO_SCHEDULE_TASKS",
		"CALENDAR_AUTO_SCHEDULE_HABITS", "CALENDAR_AUTO_SCHEDULE_MEETINGS",
		"STRIPE_API_KEY", "STRIPE_WEBHOOK_SECRET",
		"MCP_ADDR", "MCP_AUTH_TOKEN",
		"ORBITA_ORBIT_PATH", "ORBITA_ENGINE_PATH",
		"ORBITA_MARKETPLACE_URL", "ORBITA_INSTALL_DIR",
	}
	for _, v := range envVars {
		os.Unsetenv(v)
	}
}

func TestLoad_DefaultValues(t *testing.T) {
	clearEnvVars()
	defer clearEnvVars()

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Application defaults
	assert.Equal(t, "development", cfg.AppEnv)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", cfg.UserID)
	assert.Equal(t, "", cfg.EncryptionKey)

	// Local mode is enabled by default when no DATABASE_URL is set
	assert.True(t, cfg.LocalMode)
	assert.Equal(t, "sqlite", cfg.DatabaseDriver)

	// Outbox defaults
	assert.Equal(t, 100*time.Millisecond, cfg.OutboxPollInterval)
	assert.Equal(t, 100, cfg.OutboxBatchSize)
	assert.Equal(t, 5, cfg.OutboxMaxRetries)
	assert.Equal(t, 30*time.Second, cfg.OutboxStatsInterval)
	assert.Equal(t, 14, cfg.OutboxRetentionDays)
	assert.Equal(t, 24*time.Hour, cfg.OutboxCleanupInterval)
	assert.True(t, cfg.OutboxProcessorEnabled)

	// Worker defaults
	assert.Equal(t, "0.0.0.0:8081", cfg.WorkerHealthAddr)

	// Calendar defaults
	assert.False(t, cfg.CalendarDeleteMissing)
	assert.Equal(t, "primary", cfg.CalendarID)
	assert.True(t, cfg.CalendarSyncEnabled)
	assert.Equal(t, 5*time.Minute, cfg.CalendarSyncInterval)
	assert.Equal(t, 7, cfg.CalendarSyncLookAheadDays)
	assert.Equal(t, "time_first", cfg.CalendarConflictStrategy)
	assert.True(t, cfg.CalendarAutoScheduleTasks)
	assert.True(t, cfg.CalendarAutoScheduleHabits)
	assert.True(t, cfg.CalendarAutoScheduleMeetings)

	// MCP defaults
	assert.Equal(t, "0.0.0.0:8082", cfg.MCPAddr)
	assert.Equal(t, "", cfg.MCPAuthToken)

	// Marketplace defaults
	assert.Equal(t, "https://marketplace.orbita.dev", cfg.MarketplaceURL)
}

func TestLoad_WithCustomEnvVars(t *testing.T) {
	clearEnvVars()
	defer clearEnvVars()

	// Set custom values
	os.Setenv("APP_ENV", "production")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("ORBITA_USER_ID", "test-user-id")
	os.Setenv("ORBITA_ENCRYPTION_KEY", "my-secret-key")
	os.Setenv("OUTBOX_BATCH_SIZE", "200")
	os.Setenv("OUTBOX_POLL_INTERVAL", "500ms")
	os.Setenv("OUTBOX_PROCESSOR_ENABLED", "false")
	os.Setenv("CALENDAR_DELETE_MISSING", "true")
	os.Setenv("CALENDAR_SYNC_INTERVAL", "10m")
	os.Setenv("CALENDAR_SYNC_LOOK_AHEAD_DAYS", "14")

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "production", cfg.AppEnv)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "test-user-id", cfg.UserID)
	assert.Equal(t, "my-secret-key", cfg.EncryptionKey)
	assert.Equal(t, 200, cfg.OutboxBatchSize)
	assert.Equal(t, 500*time.Millisecond, cfg.OutboxPollInterval)
	assert.False(t, cfg.OutboxProcessorEnabled)
	assert.True(t, cfg.CalendarDeleteMissing)
	assert.Equal(t, 10*time.Minute, cfg.CalendarSyncInterval)
	assert.Equal(t, 14, cfg.CalendarSyncLookAheadDays)
}

func TestLoad_WithDatabaseURL(t *testing.T) {
	clearEnvVars()
	defer clearEnvVars()

	// When DATABASE_URL is set, local mode should be disabled
	os.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/orbita")

	cfg, err := Load()
	require.NoError(t, err)

	assert.False(t, cfg.LocalMode)
	assert.Equal(t, "postgres://user:pass@localhost:5432/orbita", cfg.DatabaseURL)
}

func TestLoad_ExplicitLocalMode(t *testing.T) {
	clearEnvVars()
	defer clearEnvVars()

	// Explicit local mode even with DATABASE_URL
	os.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/orbita")
	os.Setenv("ORBITA_LOCAL_MODE", "true")

	cfg, err := Load()
	require.NoError(t, err)

	assert.True(t, cfg.LocalMode)
	assert.Equal(t, "sqlite", cfg.DatabaseDriver)
}

func TestLoad_ExplicitDatabaseDriver(t *testing.T) {
	clearEnvVars()
	defer clearEnvVars()

	os.Setenv("DATABASE_DRIVER", "postgres")
	os.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/orbita")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "postgres", cfg.DatabaseDriver)
}

func TestConfig_IsDevelopment(t *testing.T) {
	tests := []struct {
		appEnv   string
		expected bool
	}{
		{"development", true},
		{"production", false},
		{"staging", false},
		{"test", false},
	}

	for _, tt := range tests {
		t.Run(tt.appEnv, func(t *testing.T) {
			cfg := &Config{AppEnv: tt.appEnv}
			assert.Equal(t, tt.expected, cfg.IsDevelopment())
		})
	}
}

func TestConfig_IsProduction(t *testing.T) {
	tests := []struct {
		appEnv   string
		expected bool
	}{
		{"development", false},
		{"production", true},
		{"staging", false},
		{"test", false},
	}

	for _, tt := range tests {
		t.Run(tt.appEnv, func(t *testing.T) {
			cfg := &Config{AppEnv: tt.appEnv}
			assert.Equal(t, tt.expected, cfg.IsProduction())
		})
	}
}

func TestConfig_IsLocalMode(t *testing.T) {
	cfg := &Config{LocalMode: true}
	assert.True(t, cfg.IsLocalMode())

	cfg = &Config{LocalMode: false}
	assert.False(t, cfg.IsLocalMode())
}

func TestConfig_IsSQLite(t *testing.T) {
	tests := []struct {
		name     string
		driver   string
		local    bool
		expected bool
	}{
		{"explicit sqlite", "sqlite", false, true},
		{"local mode", "auto", true, true},
		{"postgres driver", "postgres", false, false},
		{"auto with local", "auto", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{DatabaseDriver: tt.driver, LocalMode: tt.local}
			assert.Equal(t, tt.expected, cfg.IsSQLite())
		})
	}
}

func TestConfig_IsPostgres(t *testing.T) {
	tests := []struct {
		name     string
		driver   string
		local    bool
		expected bool
	}{
		{"explicit postgres", "postgres", false, true},
		{"auto without local", "auto", false, true},
		{"auto with local", "auto", true, false},
		{"sqlite driver", "sqlite", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{DatabaseDriver: tt.driver, LocalMode: tt.local}
			assert.Equal(t, tt.expected, cfg.IsPostgres())
		})
	}
}

func TestConfig_LicenseFilePath(t *testing.T) {
	cfg := &Config{}
	path := cfg.LicenseFilePath()

	// Should contain .orbita/license.json
	assert.Contains(t, path, ".orbita/license.json")
}

func TestGetEnv(t *testing.T) {
	// Test default value
	value := getEnv("NON_EXISTENT_VAR", "default")
	assert.Equal(t, "default", value)

	// Test with set value
	os.Setenv("TEST_VAR", "custom")
	defer os.Unsetenv("TEST_VAR")
	value = getEnv("TEST_VAR", "default")
	assert.Equal(t, "custom", value)

	// Test with empty string (should use default)
	os.Setenv("TEST_EMPTY", "")
	defer os.Unsetenv("TEST_EMPTY")
	value = getEnv("TEST_EMPTY", "default")
	assert.Equal(t, "default", value)
}

func TestGetIntEnv(t *testing.T) {
	// Test default value
	value := getIntEnv("NON_EXISTENT_INT", 42)
	assert.Equal(t, 42, value)

	// Test with valid int
	os.Setenv("TEST_INT", "100")
	defer os.Unsetenv("TEST_INT")
	value = getIntEnv("TEST_INT", 42)
	assert.Equal(t, 100, value)

	// Test with invalid int (should use default)
	os.Setenv("TEST_INVALID_INT", "not-a-number")
	defer os.Unsetenv("TEST_INVALID_INT")
	value = getIntEnv("TEST_INVALID_INT", 42)
	assert.Equal(t, 42, value)
}

func TestGetDurationEnv(t *testing.T) {
	// Test default value
	value := getDurationEnv("NON_EXISTENT_DUR", 5*time.Second)
	assert.Equal(t, 5*time.Second, value)

	// Test with valid duration
	os.Setenv("TEST_DUR", "10m")
	defer os.Unsetenv("TEST_DUR")
	value = getDurationEnv("TEST_DUR", 5*time.Second)
	assert.Equal(t, 10*time.Minute, value)

	// Test with invalid duration (should use default)
	os.Setenv("TEST_INVALID_DUR", "not-a-duration")
	defer os.Unsetenv("TEST_INVALID_DUR")
	value = getDurationEnv("TEST_INVALID_DUR", 5*time.Second)
	assert.Equal(t, 5*time.Second, value)
}

func TestGetBoolEnv(t *testing.T) {
	// Test default value
	value := getBoolEnv("NON_EXISTENT_BOOL", true)
	assert.True(t, value)

	// Test with true values
	trueValues := []string{"true", "1", "True", "TRUE"}
	for _, tv := range trueValues {
		os.Setenv("TEST_BOOL", tv)
		value = getBoolEnv("TEST_BOOL", false)
		assert.True(t, value, "Expected true for value: %s", tv)
	}

	// Test with false values
	falseValues := []string{"false", "0", "False", "FALSE"}
	for _, fv := range falseValues {
		os.Setenv("TEST_BOOL", fv)
		value = getBoolEnv("TEST_BOOL", true)
		assert.False(t, value, "Expected false for value: %s", fv)
	}
	os.Unsetenv("TEST_BOOL")

	// Test with invalid bool (should use default)
	os.Setenv("TEST_INVALID_BOOL", "not-a-bool")
	defer os.Unsetenv("TEST_INVALID_BOOL")
	value = getBoolEnv("TEST_INVALID_BOOL", true)
	assert.True(t, value)
}

func TestGetPathListEnv(t *testing.T) {
	// Test empty value
	value := getPathListEnv("NON_EXISTENT_PATH")
	assert.Nil(t, value)

	// Test with single path
	os.Setenv("TEST_PATH", "/path/to/dir")
	defer os.Unsetenv("TEST_PATH")
	value = getPathListEnv("TEST_PATH")
	assert.Equal(t, []string{"/path/to/dir"}, value)

	// Test with multiple paths (Unix-style colon separator)
	os.Setenv("TEST_PATHS", "/path1:/path2:/path3")
	defer os.Unsetenv("TEST_PATHS")
	value = getPathListEnv("TEST_PATHS")
	assert.Equal(t, []string{"/path1", "/path2", "/path3"}, value)
}

func TestSplitPaths(t *testing.T) {
	// Test empty string
	result := splitPaths("")
	assert.Empty(t, result)

	// Test single path
	result = splitPaths("/single/path")
	assert.Equal(t, []string{"/single/path"}, result)

	// Test multiple paths (using colon as separator on Unix)
	// Note: This test assumes Unix path separator
	result = splitPaths("/path1:/path2:/path3")
	assert.Equal(t, []string{"/path1", "/path2", "/path3"}, result)

	// Test with trailing separator
	result = splitPaths("/path1:/path2:")
	assert.Equal(t, []string{"/path1", "/path2"}, result)

	// Test with leading separator
	result = splitPaths(":/path1:/path2")
	assert.Equal(t, []string{"/path1", "/path2"}, result)
}

func TestGetDefaultInstallDir(t *testing.T) {
	path := getDefaultInstallDir()
	// Should contain .orbita/packages
	assert.Contains(t, path, ".orbita/packages")
}

func TestGetDefaultSQLitePath(t *testing.T) {
	path := getDefaultSQLitePath()
	// Should contain .orbita/data.db
	assert.Contains(t, path, ".orbita/data.db")
}

func TestLoad_OAuthConfig(t *testing.T) {
	clearEnvVars()
	defer clearEnvVars()

	os.Setenv("OAUTH_PROVIDER", "google")
	os.Setenv("OAUTH_CLIENT_ID", "client-id")
	os.Setenv("OAUTH_CLIENT_SECRET", "client-secret")
	os.Setenv("OAUTH_AUTH_URL", "https://auth.example.com")
	os.Setenv("OAUTH_TOKEN_URL", "https://token.example.com")
	os.Setenv("OAUTH_REDIRECT_URL", "http://localhost:8080/callback")
	os.Setenv("OAUTH_SCOPES", "email profile")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "google", cfg.OAuthProvider)
	assert.Equal(t, "client-id", cfg.OAuthClientID)
	assert.Equal(t, "client-secret", cfg.OAuthClientSecret)
	assert.Equal(t, "https://auth.example.com", cfg.OAuthAuthURL)
	assert.Equal(t, "https://token.example.com", cfg.OAuthTokenURL)
	assert.Equal(t, "http://localhost:8080/callback", cfg.OAuthRedirectURL)
	assert.Equal(t, "email profile", cfg.OAuthScopes)
}

func TestLoad_StripeConfig(t *testing.T) {
	clearEnvVars()
	defer clearEnvVars()

	os.Setenv("STRIPE_API_KEY", "sk_test_123")
	os.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_123")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "sk_test_123", cfg.StripeAPIKey)
	assert.Equal(t, "whsec_123", cfg.StripeWebhookSecret)
}
