package schedule

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	googleCalendar "github.com/felixgeelhaar/orbita/internal/calendar/infrastructure/google"
	"github.com/felixgeelhaar/orbita/internal/scheduling/application/commands"
	"github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/spf13/cobra"
)

var (
	importDays              int
	importBlockType         string
	importTaggedOnly        bool
	importCalendarID        string
	importUseConfigCalendar bool
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import calendar events into your schedule",
	Long: `Import events from your external calendar into Orbita schedule blocks.

Examples:
  orbita schedule import --days 7 --type meeting
  orbita schedule import --days 3 --tagged-only
  orbita schedule import --calendar <id> --type focus`,
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.AddBlockHandler == nil {
			fmt.Fprintln(cmd.OutOrStdout(), "Schedule commands require database connection.")
			fmt.Fprintln(cmd.OutOrStdout(), "Start services with: docker-compose up -d")
			return nil
		}
		if app.CalendarSyncer == nil {
			return errors.New("calendar sync not configured")
		}

		googleSyncer, ok := app.CalendarSyncer.(*googleCalendar.Syncer)
		if !ok {
			return errors.New("calendar import only supported for Google Calendar")
		}

		blockType := domain.BlockType(strings.ToLower(importBlockType))
		validTypes := []domain.BlockType{
			domain.BlockTypeTask,
			domain.BlockTypeHabit,
			domain.BlockTypeMeeting,
			domain.BlockTypeFocus,
			domain.BlockTypeBreak,
		}
		isValid := false
		for _, t := range validTypes {
			if blockType == t {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("invalid block type: %s (valid: task, habit, meeting, focus, break)", importBlockType)
		}

		if importUseConfigCalendar && app.SettingsService != nil && importCalendarID == "" {
			if storedID, err := app.SettingsService.GetCalendarID(cmd.Context(), app.CurrentUserID); err == nil && storedID != "" {
				importCalendarID = storedID
			}
		}

		if importCalendarID != "" {
			googleSyncer = googleSyncer.WithCalendarID(importCalendarID)
		} else if !importUseConfigCalendar {
			googleSyncer = googleSyncer.WithCalendarID("primary")
		}

		start := time.Now()
		end := start.AddDate(0, 0, importDays)
		events, err := googleSyncer.ListEvents(cmd.Context(), app.CurrentUserID, start, end, importTaggedOnly)
		if err != nil {
			return err
		}
		if len(events) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No events to import.")
			return nil
		}

		created := 0
		failed := 0
		for _, event := range events {
			title := strings.TrimSpace(event.Summary)
			if title == "" {
				title = "Imported event"
			}
			startTime := event.StartTime.In(time.Local)
			endTime := event.EndTime.In(time.Local)
			if !endTime.After(startTime) {
				failed++
				fmt.Fprintf(cmd.ErrOrStderr(), "Skipping event %s: invalid time range\n", event.ID)
				continue
			}

			cmdData := commands.AddBlockCommand{
				UserID:    app.CurrentUserID,
				Date:      startTime,
				BlockType: string(blockType),
				Title:     title,
				StartTime: startTime,
				EndTime:   endTime,
			}
			if _, err := app.AddBlockHandler.Handle(cmd.Context(), cmdData); err != nil {
				failed++
				fmt.Fprintf(cmd.ErrOrStderr(), "Failed to import event %s: %v\n", event.ID, err)
				continue
			}
			created++
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Imported events: created=%d failed=%d\n", created, failed)
		return nil
	},
}

func init() {
	importCmd.Flags().IntVarP(&importDays, "days", "d", 7, "number of days to import")
	importCmd.Flags().StringVarP(&importBlockType, "type", "t", "focus", "block type (task, habit, meeting, focus, break)")
	importCmd.Flags().BoolVar(&importTaggedOnly, "tagged-only", false, "only import events tagged as orbita")
	importCmd.Flags().StringVar(&importCalendarID, "calendar", "", "calendar ID to import from (default: primary)")
	importCmd.Flags().BoolVar(&importUseConfigCalendar, "use-config-calendar", true, "use CALENDAR_ID from config when no --calendar is provided")
}
