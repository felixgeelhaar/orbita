package settings

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	googleCalendar "github.com/felixgeelhaar/orbita/internal/calendar/infrastructure/google"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "settings",
	Short: "Manage user settings",
}

var calendarCmd = &cobra.Command{
	Use:   "calendar",
	Short: "Manage calendar settings",
}

var calendarGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get calendar ID",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.SettingsService == nil {
			return errors.New("settings service not configured")
		}
		if app.CurrentUserID == uuid.Nil {
			return errors.New("current user not configured")
		}

		calendarID, err := app.SettingsService.GetCalendarID(cmd.Context(), app.CurrentUserID)
		if err != nil {
			return err
		}
		if calendarID == "" {
			calendarID = "primary"
		}
		if settingsJSON {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
				"calendar_id": calendarID,
			})
		}
		fmt.Fprintln(cmd.OutOrStdout(), calendarID)
		return nil
	},
}

var calendarSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set calendar ID",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.SettingsService == nil {
			return errors.New("settings service not configured")
		}
		if app.CurrentUserID == uuid.Nil {
			return errors.New("current user not configured")
		}
		if calendarID == "" {
			return errors.New("missing --calendar")
		}

		if err := app.SettingsService.SetCalendarID(cmd.Context(), app.CurrentUserID, calendarID); err != nil {
			return err
		}
		if settingsJSON {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
				"calendar_id": calendarID,
				"updated":     true,
			})
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Calendar ID saved.")
		return nil
	},
}

var calendarListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available calendars",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.CalendarSyncer == nil {
			return errors.New("calendar sync not configured")
		}
		if app.CurrentUserID == uuid.Nil {
			return errors.New("current user not configured")
		}

		googleSyncer, ok := app.CalendarSyncer.(*googleCalendar.Syncer)
		if !ok {
			return errors.New("calendar listing not supported for this provider")
		}

		calendars, err := googleSyncer.ListCalendars(cmd.Context(), app.CurrentUserID)
		if err != nil {
			return err
		}
		if len(calendars) == 0 {
			fmt.Println("No calendars found.")
			return nil
		}

		if calendarListJSON {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(calendars)
		}

		for _, cal := range calendars {
			if calendarPrimaryOnly && !cal.Primary {
				continue
			}
			primary := ""
			if cal.Primary {
				primary = " (primary)"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s - %s%s\n", cal.ID, cal.Name, primary)
		}
		return nil
	},
}

var deleteMissingCmd = &cobra.Command{
	Use:   "delete-missing",
	Short: "Manage delete-missing preference",
}

var deleteMissingGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get delete-missing preference",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.SettingsService == nil {
			return errors.New("settings service not configured")
		}
		if app.CurrentUserID == uuid.Nil {
			return errors.New("current user not configured")
		}

		value, err := app.SettingsService.GetDeleteMissing(cmd.Context(), app.CurrentUserID)
		if err != nil {
			return err
		}
		if settingsJSON {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
				"delete_missing": value,
			})
		}
		fmt.Fprintln(cmd.OutOrStdout(), value)
		return nil
	},
}

var deleteMissingSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set delete-missing preference",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.SettingsService == nil {
			return errors.New("settings service not configured")
		}
		if app.CurrentUserID == uuid.Nil {
			return errors.New("current user not configured")
		}

		if err := app.SettingsService.SetDeleteMissing(cmd.Context(), app.CurrentUserID, deleteMissingValue); err != nil {
			return err
		}
		if settingsJSON {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
				"delete_missing": deleteMissingValue,
				"updated":        true,
			})
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Delete-missing preference saved.")
		return nil
	},
}

var calendarID string
var deleteMissingValue bool
var calendarPrimaryOnly bool
var calendarListJSON bool
var settingsJSON bool

func init() {
	calendarSetCmd.Flags().StringVar(&calendarID, "calendar", "", "calendar ID to store")
	_ = calendarSetCmd.MarkFlagRequired("calendar")

	calendarCmd.AddCommand(calendarGetCmd)
	calendarCmd.AddCommand(calendarSetCmd)
	calendarCmd.AddCommand(calendarListCmd)

	deleteMissingSetCmd.Flags().BoolVar(&deleteMissingValue, "value", false, "delete-missing preference")
	deleteMissingCmd.AddCommand(deleteMissingGetCmd)
	deleteMissingCmd.AddCommand(deleteMissingSetCmd)
	calendarCmd.AddCommand(deleteMissingCmd)

	calendarListCmd.Flags().BoolVar(&calendarPrimaryOnly, "primary-only", false, "show only the primary calendar")
	calendarListCmd.Flags().BoolVar(&calendarListJSON, "json", false, "output as JSON")
	calendarGetCmd.Flags().BoolVar(&settingsJSON, "json", false, "output as JSON")
	calendarSetCmd.Flags().BoolVar(&settingsJSON, "json", false, "output as JSON")
	deleteMissingGetCmd.Flags().BoolVar(&settingsJSON, "json", false, "output as JSON")
	deleteMissingSetCmd.Flags().BoolVar(&settingsJSON, "json", false, "output as JSON")

	Cmd.AddCommand(calendarCmd)
}
