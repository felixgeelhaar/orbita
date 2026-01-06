package auth

import (
	"errors"
	"fmt"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	identityOAuth "github.com/felixgeelhaar/orbita/internal/identity/application/oauth"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var service *identityOAuth.Service

// SetService wires the OAuth service for CLI commands.
func SetService(s *identityOAuth.Service) {
	service = s
}

var Cmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication helpers",
}

var authURLCmd = &cobra.Command{
	Use:   "url",
	Short: "Generate OAuth2 authorization URL",
	RunE: func(cmd *cobra.Command, args []string) error {
		if service == nil {
			return errors.New("auth service not configured")
		}
		state := uuid.New().String()
		url := service.AuthURL(state)
		fmt.Println(url)
		fmt.Printf("State: %s\n", state)
		return nil
	},
}

var authExchangeCmd = &cobra.Command{
	Use:   "exchange",
	Short: "Exchange OAuth2 code for tokens and store them",
	RunE: func(cmd *cobra.Command, args []string) error {
		if service == nil {
			return errors.New("auth service not configured")
		}
		if authCode == "" {
			return errors.New("missing --code")
		}

		app := cli.GetApp()
		if app == nil || app.CurrentUserID == uuid.Nil {
			return errors.New("current user not configured")
		}

		_, err := service.ExchangeAndStore(cmd.Context(), app.CurrentUserID, authCode)
		if err != nil {
			return err
		}

		fmt.Println("Tokens stored.")
		return nil
	},
}

var authCode string

func init() {
	authExchangeCmd.Flags().StringVar(&authCode, "code", "", "authorization code")

	Cmd.AddCommand(authURLCmd)
	Cmd.AddCommand(authExchangeCmd)
}
