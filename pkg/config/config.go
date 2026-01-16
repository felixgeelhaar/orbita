package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds application configuration.
type Config struct {
	// Application
	AppEnv        string
	LogLevel      string
	UserID        string
	EncryptionKey string

	// Database
	DatabaseURL    string
	DatabaseDriver string // "postgres", "sqlite", or "auto" (default)
	SQLitePath     string // Path to SQLite database file (default: ~/.orbita/data.db)
	LocalMode      bool   // If true, uses SQLite and disables external services

	// Redis
	RedisURL string

	// RabbitMQ
	RabbitMQURL string

	// Outbox
	OutboxPollInterval     time.Duration
	OutboxBatchSize        int
	OutboxMaxRetries       int
	OutboxStatsInterval    time.Duration
	OutboxRetentionDays    int
	OutboxCleanupInterval  time.Duration
	OutboxProcessorEnabled bool

	// Worker
	WorkerHealthAddr string

	// OAuth
	OAuthProvider     string
	OAuthClientID     string
	OAuthClientSecret string
	OAuthAuthURL      string
	OAuthTokenURL     string
	OAuthRedirectURL  string
	OAuthScopes       string

	// Calendar
	CalendarDeleteMissing bool
	CalendarID            string

	// Calendar Sync (bi-directional)
	CalendarSyncEnabled          bool          // Enable automatic calendar sync
	CalendarSyncInterval         time.Duration // How often to check for external changes
	CalendarSyncLookAheadDays    int           // How far ahead to look for events
	CalendarConflictStrategy     string        // orbita_wins, external_wins, manual, time_first
	CalendarAutoScheduleTasks    bool          // Auto-schedule new tasks
	CalendarAutoScheduleHabits   bool          // Auto-schedule habit sessions
	CalendarAutoScheduleMeetings bool          // Auto-schedule meeting blocks

	// Billing
	StripeAPIKey        string
	StripeWebhookSecret string

	// MCP
	MCPAddr      string
	MCPAuthToken string

	// Plugins
	OrbitSearchPaths  []string
	EngineSearchPaths []string

	// Marketplace
	MarketplaceURL       string
	MarketplaceInstallDir string
}

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

	// Detect local mode: enabled when no DATABASE_URL is set or explicitly requested
	localMode := getBoolEnv("ORBITA_LOCAL_MODE", os.Getenv("DATABASE_URL") == "")
	dbDriver := getEnv("DATABASE_DRIVER", "auto")
	dbURL := getEnv("DATABASE_URL", "")
	sqlitePath := getEnv("SQLITE_PATH", getDefaultSQLitePath())

	// In local mode, default to SQLite
	if localMode && dbDriver == "auto" {
		dbDriver = "sqlite"
	}

	// If no DATABASE_URL but not local mode, use default PostgreSQL URL for development
	if dbURL == "" && !localMode {
		dbURL = "postgres://orbita:orbita_dev@localhost:5432/orbita?sslmode=disable"
	}

	cfg := &Config{
		AppEnv:         getEnv("APP_ENV", "development"),
		LogLevel:       getEnv("LOG_LEVEL", "info"),
		UserID:         getEnv("ORBITA_USER_ID", "00000000-0000-0000-0000-000000000001"),
		EncryptionKey:  getEnv("ORBITA_ENCRYPTION_KEY", ""),
		DatabaseURL:    dbURL,
		DatabaseDriver: dbDriver,
		SQLitePath:     sqlitePath,
		LocalMode:      localMode,
		RedisURL:       getEnv("REDIS_URL", "redis://localhost:6379/0"),
		RabbitMQURL:    getEnv("RABBITMQ_URL", "amqp://orbita:orbita_dev@localhost:5672/"),

		OutboxPollInterval:     getDurationEnv("OUTBOX_POLL_INTERVAL", 100*time.Millisecond),
		OutboxBatchSize:        getIntEnv("OUTBOX_BATCH_SIZE", 100),
		OutboxMaxRetries:       getIntEnv("OUTBOX_MAX_RETRIES", 5),
		OutboxStatsInterval:    getDurationEnv("OUTBOX_STATS_INTERVAL", 30*time.Second),
		OutboxRetentionDays:    getIntEnv("OUTBOX_RETENTION_DAYS", 14),
		OutboxCleanupInterval:  getDurationEnv("OUTBOX_CLEANUP_INTERVAL", 24*time.Hour),
		OutboxProcessorEnabled: getBoolEnv("OUTBOX_PROCESSOR_ENABLED", true),

		WorkerHealthAddr: getEnv("WORKER_HEALTH_ADDR", "0.0.0.0:8081"),

		OAuthProvider:     getEnv("OAUTH_PROVIDER", ""),
		OAuthClientID:     getEnv("OAUTH_CLIENT_ID", ""),
		OAuthClientSecret: getEnv("OAUTH_CLIENT_SECRET", ""),
		OAuthAuthURL:      getEnv("OAUTH_AUTH_URL", ""),
		OAuthTokenURL:     getEnv("OAUTH_TOKEN_URL", ""),
		OAuthRedirectURL:  getEnv("OAUTH_REDIRECT_URL", ""),
		OAuthScopes:       getEnv("OAUTH_SCOPES", ""),

		CalendarDeleteMissing: getBoolEnv("CALENDAR_DELETE_MISSING", false),
		CalendarID:            getEnv("CALENDAR_ID", "primary"),

		CalendarSyncEnabled:          getBoolEnv("CALENDAR_SYNC_ENABLED", true),
		CalendarSyncInterval:         getDurationEnv("CALENDAR_SYNC_INTERVAL", 5*time.Minute),
		CalendarSyncLookAheadDays:    getIntEnv("CALENDAR_SYNC_LOOK_AHEAD_DAYS", 7),
		CalendarConflictStrategy:     getEnv("CALENDAR_CONFLICT_STRATEGY", "time_first"),
		CalendarAutoScheduleTasks:    getBoolEnv("CALENDAR_AUTO_SCHEDULE_TASKS", true),
		CalendarAutoScheduleHabits:   getBoolEnv("CALENDAR_AUTO_SCHEDULE_HABITS", true),
		CalendarAutoScheduleMeetings: getBoolEnv("CALENDAR_AUTO_SCHEDULE_MEETINGS", true),

		StripeAPIKey:        getEnv("STRIPE_API_KEY", ""),
		StripeWebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),

		MCPAddr:      getEnv("MCP_ADDR", "0.0.0.0:8082"),
		MCPAuthToken: getEnv("MCP_AUTH_TOKEN", ""),

		OrbitSearchPaths:  getPathListEnv("ORBITA_ORBIT_PATH"),
		EngineSearchPaths: getPathListEnv("ORBITA_ENGINE_PATH"),

		MarketplaceURL:        getEnv("ORBITA_MARKETPLACE_URL", "https://marketplace.orbita.dev"),
		MarketplaceInstallDir: getEnv("ORBITA_INSTALL_DIR", getDefaultInstallDir()),
	}

	return cfg, nil
}

// IsDevelopment returns true if running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.AppEnv == "development"
}

// IsProduction returns true if running in production mode.
func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

// IsLocalMode returns true if using SQLite local mode.
func (c *Config) IsLocalMode() bool {
	return c.LocalMode
}

// IsSQLite returns true if using SQLite as the database.
func (c *Config) IsSQLite() bool {
	return c.DatabaseDriver == "sqlite" || c.LocalMode
}

// IsPostgres returns true if using PostgreSQL as the database.
func (c *Config) IsPostgres() bool {
	return c.DatabaseDriver == "postgres" || (c.DatabaseDriver == "auto" && !c.LocalMode)
}

// LicenseFilePath returns the path to the license file.
func (c *Config) LicenseFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".orbita/license.json"
	}
	return home + "/.orbita/license.json"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

func getPathListEnv(key string) []string {
	value := os.Getenv(key)
	if value == "" {
		return nil
	}
	// Split by colon (Unix) or semicolon (Windows)
	paths := []string{}
	for _, p := range splitPaths(value) {
		if p != "" {
			paths = append(paths, p)
		}
	}
	return paths
}

func getDefaultInstallDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".orbita/packages"
	}
	return home + "/.orbita/packages"
}

func getDefaultSQLitePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".orbita/data.db"
	}
	return home + "/.orbita/data.db"
}

func splitPaths(s string) []string {
	// Use colon as separator on Unix, semicolon on Windows
	separator := ":"
	if os.PathSeparator == '\\' {
		separator = ";"
	}
	result := []string{}
	current := ""
	for i := 0; i < len(s); i++ {
		if string(s[i]) == separator {
			if current != "" {
				result = append(result, current)
			}
			current = ""
		} else {
			current += string(s[i])
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
