package inbox

import (
	"fmt"
	"time"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	billingDomain "github.com/felixgeelhaar/orbita/internal/billing/domain"
	habitCommands "github.com/felixgeelhaar/orbita/internal/habits/application/commands"
	inboxCommands "github.com/felixgeelhaar/orbita/internal/inbox/application/commands"
	meetingCommands "github.com/felixgeelhaar/orbita/internal/meetings/application/commands"
	productivityCommands "github.com/felixgeelhaar/orbita/internal/productivity/application/commands"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var (
	promoteItemID        string
	promoteTarget        string
	taskTitle            string
	taskDescription      string
	taskPriority         string
	taskDurationMinutes  int
	taskDue              string
	habitName            string
	habitDescription     string
	habitFrequency       string
	habitTimesPerWeek    int
	habitDurationMinutes int
	habitPreferredTime   string
	meetingName          string
	meetingCadence       string
	meetingCadenceDays   int
	meetingDurationMins  int
	meetingPreferredTime string
)

var promoteCmd = &cobra.Command{
	Use:   "promote",
	Short: "Promote an inbox item into a task, habit, or meeting",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.PromoteInboxItemHandler == nil {
			fmt.Println("Inbox commands require a database connection.")
			fmt.Println("Start services with: docker-compose up -d")
			return nil
		}

		if err := cli.RequireEntitlement(cmd.Context(), app, billingDomain.ModuleAIInbox); err != nil {
			return err
		}

		itemID, err := uuid.Parse(promoteItemID)
		if err != nil {
			return fmt.Errorf("invalid item id: %w", err)
		}

		target, err := inboxCommands.ParsePromoteTarget(promoteTarget)
		if err != nil {
			return err
		}

		promotion := inboxCommands.PromoteInboxItemCommand{
			UserID: app.CurrentUserID,
			ItemID: itemID,
			Target: target,
		}

		switch target {
		case inboxCommands.PromoteTargetTask:
			dueDate, err := parseDate(taskDue)
			if err != nil {
				return err
			}

			promotion.TaskArgs = &productivityCommands.CreateTaskCommand{
				Title:           taskTitle,
				Description:     taskDescription,
				Priority:        taskPriority,
				DurationMinutes: taskDurationMinutes,
				DueDate:         dueDate,
			}
		case inboxCommands.PromoteTargetHabit:
			promotion.HabitArgs = &habitCommands.CreateHabitCommand{
				Name:          habitName,
				Description:   habitDescription,
				Frequency:     habitFrequency,
				TimesPerWeek:  habitTimesPerWeek,
				DurationMins:  habitDurationMinutes,
				PreferredTime: habitPreferredTime,
			}
		case inboxCommands.PromoteTargetMeeting:
			promotion.MeetingArgs = &meetingCommands.CreateMeetingCommand{
				Name:          meetingName,
				Cadence:       meetingCadence,
				CadenceDays:   meetingCadenceDays,
				DurationMins:  meetingDurationMins,
				PreferredTime: meetingPreferredTime,
			}
		default:
			return fmt.Errorf("unsupported promote target: %s", target)
		}

		result, err := app.PromoteInboxItemHandler.Handle(cmd.Context(), promotion)
		if err != nil {
			return fmt.Errorf("failed to promote inbox item: %w", err)
		}

		fmt.Printf("Promoted %s to %s (%s)\n", itemID, target, result.PromotedID)
		return nil
	},
}

func init() {
	promoteCmd.Flags().StringVar(&promoteItemID, "id", "", "inbox item ID to promote (required)")
	promoteCmd.Flags().StringVar(&promoteTarget, "target", "task", "target type (task|habit|meeting)")
	promoteCmd.Flags().StringVar(&taskTitle, "task-title", "", "task title")
	promoteCmd.Flags().StringVar(&taskDescription, "task-description", "", "task description")
	promoteCmd.Flags().StringVar(&taskPriority, "task-priority", "", "task priority (urgent|high|medium|low)")
	promoteCmd.Flags().IntVar(&taskDurationMinutes, "task-duration", 0, "task duration in minutes")
	promoteCmd.Flags().StringVar(&taskDue, "task-due", "", "task due date (YYYY-MM-DD)")
	promoteCmd.Flags().StringVar(&habitName, "habit-name", "", "habit name")
	promoteCmd.Flags().StringVar(&habitDescription, "habit-description", "", "habit description")
	promoteCmd.Flags().StringVar(&habitFrequency, "habit-frequency", "", "habit frequency (daily|weekly|custom)")
	promoteCmd.Flags().IntVar(&habitTimesPerWeek, "habit-times-per-week", 0, "habit occurrences per week (custom frequency)")
	promoteCmd.Flags().IntVar(&habitDurationMinutes, "habit-duration", 0, "habit duration in minutes")
	promoteCmd.Flags().StringVar(&habitPreferredTime, "habit-preferred-time", "", "habit preferred time (HH:MM)")
	promoteCmd.Flags().StringVar(&meetingName, "meeting-name", "", "meeting name")
	promoteCmd.Flags().StringVar(&meetingCadence, "meeting-cadence", "", "meeting cadence (daily|weekly|biweekly)")
	promoteCmd.Flags().IntVar(&meetingCadenceDays, "meeting-cadence-days", 0, "meeting cadence days")
	promoteCmd.Flags().IntVar(&meetingDurationMins, "meeting-duration", 0, "meeting duration in minutes")
	promoteCmd.Flags().StringVar(&meetingPreferredTime, "meeting-preferred-time", "", "meeting preferred time (HH:MM)")
	_ = promoteCmd.MarkFlagRequired("id")
}

func parseDate(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, fmt.Errorf("invalid date format, use YYYY-MM-DD: %w", err)
	}
	return &parsed, nil
}
