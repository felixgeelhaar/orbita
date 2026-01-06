package inbox

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	billingDomain "github.com/felixgeelhaar/orbita/internal/billing/domain"
	"github.com/felixgeelhaar/orbita/internal/inbox/application/queries"
	"github.com/spf13/cobra"
)

var includePromoted bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List captured inbox items",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.ListInboxItemsHandler == nil {
			fmt.Println("Inbox commands require a database connection.")
			fmt.Println("Start services with: docker-compose up -d")
			return nil
		}

		if err := cli.RequireEntitlement(cmd.Context(), app, billingDomain.ModuleAIInbox); err != nil {
			return err
		}

		query := queries.ListInboxItemsQuery{
			UserID:          app.CurrentUserID,
			IncludePromoted: includePromoted,
		}

		items, err := app.ListInboxItemsHandler.Handle(cmd.Context(), query)
		if err != nil {
			return fmt.Errorf("failed to list inbox items: %w", err)
		}

		if len(items) == 0 {
			fmt.Println("No inbox items found.")
			return nil
		}

		for _, item := range items {
			promoted := ""
			if item.Promoted {
				promoted = fmt.Sprintf(" promoted=%s", item.PromotedTo)
			}
			fmt.Printf("%s [%s]%s\n", item.ID, item.Classification, promoted)
			fmt.Printf("  Content: %s\n", item.Content)
			if len(item.Tags) > 0 {
				fmt.Printf("  Tags: %s\n", strings.Join(item.Tags, ", "))
			}
			fmt.Printf("  Captured: %s\n", item.CapturedAt)
			if item.PromotedAt != nil {
				fmt.Printf("  Promoted at: %s\n", *item.PromotedAt)
			}
			fmt.Println(strings.Repeat("-", 60))
		}

		return nil
	},
}

func init() {
	listCmd.Flags().BoolVar(&includePromoted, "include-promoted", false, "include items that already been promoted")
}
