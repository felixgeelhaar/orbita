package billing

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/security"
	"github.com/spf13/cobra"
)

var webhookEventPath string

var webhookCmd = &cobra.Command{
	Use:   "webhook",
	Short: "Handle a billing webhook payload",
	Long: `Placeholder command for Stripe webhook handling.

Examples:
  orbita billing webhook --event ./event.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if webhookEventPath == "" {
			return errors.New("event path is required")
		}

		payload, err := security.SafeReadFile(webhookEventPath)
		if err != nil {
			return err
		}

		var envelope map[string]any
		if err := json.Unmarshal(payload, &envelope); err != nil {
			return fmt.Errorf("invalid webhook payload: %w", err)
		}

		eventType, _ := envelope["type"].(string)
		if eventType == "" {
			eventType = "unknown"
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Received billing webhook event: %s\n", eventType)
		return nil
	},
}

func init() {
	webhookCmd.Flags().StringVar(&webhookEventPath, "event", "", "path to webhook event JSON")
}
