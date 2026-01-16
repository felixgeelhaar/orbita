package setup

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/felixgeelhaar/orbita/internal/calendar/application"
	"github.com/felixgeelhaar/orbita/internal/calendar/domain"
	"github.com/felixgeelhaar/orbita/internal/calendar/infrastructure/caldav"
	googleCal "github.com/felixgeelhaar/orbita/internal/calendar/infrastructure/google"
	microsoftCal "github.com/felixgeelhaar/orbita/internal/calendar/infrastructure/microsoft"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

// OAuthTokenProvider provides OAuth2 tokens for a user.
type OAuthTokenProvider interface {
	TokenSource(ctx context.Context, userID uuid.UUID) (oauth2.TokenSource, error)
}

// CalDAVCredentialProvider provides CalDAV credentials for a user.
type CalDAVCredentialProvider interface {
	GetCredentials(ctx context.Context, userID uuid.UUID, provider domain.ProviderType) (username, password string, err error)
}

// ProviderConfig holds configuration for creating provider factories.
type ProviderConfig struct {
	GoogleOAuth    OAuthTokenProvider
	MicrosoftOAuth OAuthTokenProvider
	CalDAVCreds    CalDAVCredentialProvider
	Logger         *slog.Logger
}

// RegisterProviders registers all available calendar providers with the registry.
func RegisterProviders(registry *application.ProviderRegistry, config ProviderConfig) {
	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}

	// Register Google Calendar provider
	if config.GoogleOAuth != nil {
		registry.RegisterBidirectional(domain.ProviderGoogle, func(ctx context.Context, cal *domain.ConnectedCalendar) (application.BidirectionalSyncer, error) {
			syncer := googleCal.NewSyncer(config.GoogleOAuth, logger)
			if cal.CalendarID() != "" && cal.CalendarID() != "primary" {
				syncer.WithCalendarID(cal.CalendarID())
			}
			return syncer, nil
		})
		logger.Debug("registered Google Calendar provider")
	}

	// Register Microsoft Calendar provider
	if config.MicrosoftOAuth != nil {
		registry.RegisterBidirectional(domain.ProviderMicrosoft, func(ctx context.Context, cal *domain.ConnectedCalendar) (application.BidirectionalSyncer, error) {
			syncer := microsoftCal.NewSyncer(config.MicrosoftOAuth, logger)
			if cal.CalendarID() != "" && cal.CalendarID() != "primary" {
				syncer.WithCalendarID(cal.CalendarID())
			}
			return syncer, nil
		})
		logger.Debug("registered Microsoft Calendar provider")
	}

	// Register CalDAV providers (Apple, generic CalDAV)
	if config.CalDAVCreds != nil {
		// Apple Calendar (iCloud)
		registry.RegisterBidirectional(domain.ProviderApple, func(ctx context.Context, cal *domain.ConnectedCalendar) (application.BidirectionalSyncer, error) {
			username, password, err := config.CalDAVCreds.GetCredentials(ctx, cal.UserID(), domain.ProviderApple)
			if err != nil {
				return nil, fmt.Errorf("failed to get Apple credentials: %w", err)
			}

			baseURL := caldav.AppleCalDAVURL
			if url := cal.CalDAVURL(); url != "" {
				baseURL = url
			}

			syncer := caldav.NewSyncer(baseURL, username, password, logger)
			if cal.CalendarID() != "" {
				syncer.WithCalendarPath(cal.CalendarID())
			}
			return syncer, nil
		})
		logger.Debug("registered Apple Calendar provider")

		// Generic CalDAV (Fastmail, Nextcloud, etc.)
		registry.RegisterBidirectional(domain.ProviderCalDAV, func(ctx context.Context, cal *domain.ConnectedCalendar) (application.BidirectionalSyncer, error) {
			username, password, err := config.CalDAVCreds.GetCredentials(ctx, cal.UserID(), domain.ProviderCalDAV)
			if err != nil {
				return nil, fmt.Errorf("failed to get CalDAV credentials: %w", err)
			}

			baseURL := cal.CalDAVURL()
			if baseURL == "" {
				return nil, fmt.Errorf("CalDAV URL not configured")
			}

			syncer := caldav.NewSyncer(baseURL, username, password, logger)
			if cal.CalendarID() != "" {
				syncer.WithCalendarPath(cal.CalendarID())
			}
			return syncer, nil
		})
		logger.Debug("registered CalDAV provider")
	}
}

// CreateGoogleSyncer creates a Google Calendar syncer for direct use.
func CreateGoogleSyncer(oauthProvider OAuthTokenProvider, calendarID string, logger *slog.Logger) *googleCal.Syncer {
	syncer := googleCal.NewSyncer(oauthProvider, logger)
	if calendarID != "" && calendarID != "primary" {
		syncer.WithCalendarID(calendarID)
	}
	return syncer
}

// CreateMicrosoftSyncer creates a Microsoft Calendar syncer for direct use.
func CreateMicrosoftSyncer(oauthProvider OAuthTokenProvider, calendarID string, logger *slog.Logger) *microsoftCal.Syncer {
	syncer := microsoftCal.NewSyncer(oauthProvider, logger)
	if calendarID != "" && calendarID != "primary" {
		syncer.WithCalendarID(calendarID)
	}
	return syncer
}

// CreateCalDAVSyncer creates a CalDAV syncer for direct use.
func CreateCalDAVSyncer(baseURL, username, password, calendarPath string, logger *slog.Logger) *caldav.Syncer {
	syncer := caldav.NewSyncer(baseURL, username, password, logger)
	if calendarPath != "" {
		syncer.WithCalendarPath(calendarPath)
	}
	return syncer
}
