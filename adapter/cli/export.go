package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	scheduleQueries "github.com/felixgeelhaar/orbita/internal/scheduling/application/queries"
	"github.com/spf13/cobra"
)

var (
	exportFormat string
	exportOutput string
	exportDays   int
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export schedule to various formats",
	Long: `Export your schedule to ICS (iCalendar) format for import into
Google Calendar, Outlook, Apple Calendar, and other calendar apps.

Examples:
  orbita export --format ics              # Export to stdout
  orbita export --format ics -o cal.ics   # Export to file
  orbita export --format ics --days 7     # Export next 7 days`,
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil || app.GetScheduleHandler == nil {
			fmt.Println("Export requires database connection.")
			fmt.Println("Start services with: docker-compose up -d")
			return nil
		}

		switch exportFormat {
		case "ics", "ical":
			return exportICS(cmd, app)
		default:
			return fmt.Errorf("unsupported format: %s (supported: ics)", exportFormat)
		}
	},
}

func exportICS(cmd *cobra.Command, app *App) error {
	now := time.Now()
	var allBlocks []scheduleQueries.TimeBlockDTO

	// Gather blocks for the specified number of days
	for i := 0; i < exportDays; i++ {
		day := now.AddDate(0, 0, i)
		query := scheduleQueries.GetScheduleQuery{
			UserID: app.CurrentUserID,
			Date:   day,
		}

		schedule, err := app.GetScheduleHandler.Handle(cmd.Context(), query)
		if err != nil {
			continue
		}

		if schedule != nil {
			allBlocks = append(allBlocks, schedule.Blocks...)
		}
	}

	if len(allBlocks) == 0 {
		fmt.Fprintf(os.Stderr, "No scheduled blocks found in the next %d days.\n", exportDays)
		return nil
	}

	// Generate ICS content
	ics := generateICS(allBlocks)

	// Output
	if exportOutput != "" {
		if err := os.WriteFile(exportOutput, []byte(ics), 0600); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Exported %d blocks to %s\n", len(allBlocks), exportOutput)
	} else {
		fmt.Print(ics)
	}

	return nil
}

func generateICS(blocks []scheduleQueries.TimeBlockDTO) string {
	var sb strings.Builder

	// ICS header
	sb.WriteString("BEGIN:VCALENDAR\r\n")
	sb.WriteString("VERSION:2.0\r\n")
	sb.WriteString("PRODID:-//Orbita//Orbita CLI//EN\r\n")
	sb.WriteString("CALSCALE:GREGORIAN\r\n")
	sb.WriteString("METHOD:PUBLISH\r\n")
	sb.WriteString("X-WR-CALNAME:Orbita Schedule\r\n")

	for _, block := range blocks {
		sb.WriteString("BEGIN:VEVENT\r\n")

		// UID - unique identifier
		sb.WriteString(fmt.Sprintf("UID:%s@orbita\r\n", block.ID.String()))

		// Timestamps
		sb.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", formatICSTime(time.Now())))
		sb.WriteString(fmt.Sprintf("DTSTART:%s\r\n", formatICSTime(block.StartTime)))
		sb.WriteString(fmt.Sprintf("DTEND:%s\r\n", formatICSTime(block.EndTime)))

		// Summary (title)
		sb.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", escapeICS(block.Title)))

		// Description with metadata
		desc := fmt.Sprintf("Type: %s", block.BlockType)
		if block.Completed {
			desc += "\\nStatus: Completed"
		} else if block.Missed {
			desc += "\\nStatus: Missed"
		}
		sb.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", desc))

		// Categories based on block type
		sb.WriteString(fmt.Sprintf("CATEGORIES:%s\r\n", strings.ToUpper(block.BlockType)))

		// Status
		if block.Completed {
			sb.WriteString("STATUS:CONFIRMED\r\n")
		} else if block.Missed {
			sb.WriteString("STATUS:CANCELLED\r\n")
		} else {
			sb.WriteString("STATUS:TENTATIVE\r\n")
		}

		sb.WriteString("END:VEVENT\r\n")
	}

	sb.WriteString("END:VCALENDAR\r\n")

	return sb.String()
}

func formatICSTime(t time.Time) string {
	return t.UTC().Format("20060102T150405Z")
}

func escapeICS(s string) string {
	// Escape special characters in ICS format
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

func init() {
	exportCmd.Flags().StringVarP(&exportFormat, "format", "f", "ics", "export format (ics)")
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "output file (default: stdout)")
	exportCmd.Flags().IntVarP(&exportDays, "days", "d", 7, "number of days to export")

	rootCmd.AddCommand(exportCmd)
}
