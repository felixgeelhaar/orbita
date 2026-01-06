package inbox

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	billingDomain "github.com/felixgeelhaar/orbita/internal/billing/domain"
	"github.com/felixgeelhaar/orbita/internal/inbox/application/commands"
	"github.com/felixgeelhaar/orbita/internal/inbox/domain"
	"github.com/spf13/cobra"
)

var (
	captureContent  string
	captureSource   string
	captureMetadata []string
	captureTags     []string
)

var captureCmd = &cobra.Command{
	Use:   "capture",
	Short: "Capture text into the AI Inbox",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.CaptureInboxItemHandler == nil {
			fmt.Println("Inbox commands require a database connection.")
			fmt.Println("Start services with: docker-compose up -d")
			return nil
		}

		if err := cli.RequireEntitlement(cmd.Context(), app, billingDomain.ModuleAIInbox); err != nil {
			return err
		}

		metadata, err := parseMetadata(captureMetadata)
		if err != nil {
			return err
		}

		command := commands.CaptureInboxItemCommand{
			UserID:   app.CurrentUserID,
			Content:  captureContent,
			Metadata: metadata,
			Tags:     captureTags,
			Source:   captureSource,
		}

		result, err := app.CaptureInboxItemHandler.Handle(cmd.Context(), command)
		if err != nil {
			return fmt.Errorf("failed to capture inbox item: %w", err)
		}

		fmt.Printf("Captured inbox item %s\n", result.ItemID)
		return nil
	},
}

func init() {
	captureCmd.Flags().StringVarP(&captureContent, "content", "c", "", "text to capture (required)")
	captureCmd.Flags().StringVar(&captureSource, "source", "", "source identifier (e.g. gmail, cli)")
	captureCmd.Flags().StringSliceVar(&captureMetadata, "metadata", nil, "metadata entry as key=value (can repeat)")
	captureCmd.Flags().StringSliceVar(&captureTags, "tag", nil, "tag for the item (can repeat)")
	_ = captureCmd.MarkFlagRequired("content")
}

func parseMetadata(values []string) (domain.InboxMetadata, error) {
	metadata := domain.InboxMetadata{}
	for _, entry := range values {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("metadata must be key=value: %s", entry)
		}
		key := strings.TrimSpace(parts[0])
		if key == "" {
			return nil, fmt.Errorf("metadata key cannot be empty: %s", entry)
		}
		metadata[key] = strings.TrimSpace(parts[1])
	}
	return metadata, nil
}
