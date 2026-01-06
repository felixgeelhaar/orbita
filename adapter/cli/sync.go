package cli

import (
	"errors"
	"fmt"
	"time"

	calendarApp "github.com/felixgeelhaar/orbita/internal/calendar/application"
	googleCalendar "github.com/felixgeelhaar/orbita/internal/calendar/infrastructure/google"
	scheduleQueries "github.com/felixgeelhaar/orbita/internal/scheduling/application/queries"
	"github.com/spf13/cobra"
)

var (
	syncDays              int
	syncDeleteMissing     bool
	syncCalendarID        string
	syncUseConfigCalendar bool
	syncAttendees         []string
	syncReminders         []int
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync schedule to external calendar",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil || app.GetScheduleHandler == nil {
			return errors.New("sync requires database connection")
		}
		if app.CalendarSyncer == nil {
			return errors.New("calendar sync not configured")
		}

		blocks, err := gatherBlocks(cmd, app, syncDays)
		if err != nil {
			return err
		}
		if len(blocks) == 0 {
			fmt.Println("No scheduled blocks to sync.")
			return nil
		}

		if syncUseConfigCalendar && app.SettingsService != nil {
			if syncCalendarID == "" {
				if storedID, err := app.SettingsService.GetCalendarID(cmd.Context(), app.CurrentUserID); err == nil && storedID != "" {
					syncCalendarID = storedID
				}
			}
			if !syncDeleteMissing {
				if storedDelete, err := app.SettingsService.GetDeleteMissing(cmd.Context(), app.CurrentUserID); err == nil && storedDelete {
					syncDeleteMissing = true
				}
			}
		}

		syncer := app.CalendarSyncer
		if googleSyncer, ok := syncer.(*googleCalendar.Syncer); ok {
			if syncDeleteMissing {
				googleSyncer = googleSyncer.WithDeleteMissing(true)
			}
			if syncCalendarID != "" {
				googleSyncer = googleSyncer.WithCalendarID(syncCalendarID)
			} else if !syncUseConfigCalendar {
				googleSyncer = googleSyncer.WithCalendarID("primary")
			}
			if len(syncAttendees) > 0 {
				googleSyncer = googleSyncer.WithAttendees(syncAttendees)
			}
			if len(syncReminders) > 0 {
				googleSyncer = googleSyncer.WithReminders(syncReminders)
			}
			syncer = googleSyncer
		}

		result, err := syncer.Sync(cmd.Context(), app.CurrentUserID, blocks)
		if err != nil {
			return err
		}

		fmt.Printf("Synced blocks: created=%d updated=%d deleted=%d failed=%d\n", result.Created, result.Updated, result.Deleted, result.Failed)
		return nil
	},
}

func gatherBlocks(cmd *cobra.Command, app *App, days int) ([]calendarApp.TimeBlock, error) {
	now := time.Now()
	allBlocks := make([]calendarApp.TimeBlock, 0)

	for i := 0; i < days; i++ {
		day := now.AddDate(0, 0, i)
		query := scheduleQueries.GetScheduleQuery{
			UserID: app.CurrentUserID,
			Date:   day,
		}

		schedule, err := app.GetScheduleHandler.Handle(cmd.Context(), query)
		if err != nil {
			return nil, err
		}

		if schedule != nil {
			for _, block := range schedule.Blocks {
				allBlocks = append(allBlocks, calendarApp.TimeBlock{
					ID:        block.ID,
					Title:     block.Title,
					BlockType: block.BlockType,
					StartTime: block.StartTime,
					EndTime:   block.EndTime,
					Completed: block.Completed,
					Missed:    block.Missed,
				})
			}
		}
	}

	return allBlocks, nil
}

func init() {
	syncCmd.Flags().IntVarP(&syncDays, "days", "d", 7, "number of days to sync")
	syncCmd.Flags().BoolVar(&syncDeleteMissing, "delete-missing", false, "delete remote events missing from this sync set")
	syncCmd.Flags().StringVar(&syncCalendarID, "calendar", "", "calendar ID to sync to (default: primary)")
	syncCmd.Flags().BoolVar(&syncUseConfigCalendar, "use-config-calendar", true, "use CALENDAR_ID from config when no --calendar is provided")
	syncCmd.Flags().StringSliceVar(&syncAttendees, "attendee", nil, "attendee email to include in synced events (repeatable)")
	syncCmd.Flags().IntSliceVar(&syncReminders, "reminder", nil, "reminder minutes for synced events (repeatable)")
	rootCmd.AddCommand(syncCmd)
}
