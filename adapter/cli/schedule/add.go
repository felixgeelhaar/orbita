package schedule

import (
	"fmt"
	"strings"
	"time"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/felixgeelhaar/orbita/internal/scheduling/application/commands"
	"github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var (
	addBlockType   string
	addTitle       string
	addDate        string
	addStartTime   string
	addEndTime     string
	addReferenceID string
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a time block to your schedule",
	Long: `Add a new time block to your daily schedule.

Block types: task, habit, meeting, focus, break

Examples:
  orbita schedule add --type focus --title "Deep work" --start 09:00 --end 11:00
  orbita schedule add --type meeting --title "Team standup" --start 10:00 --end 10:30
  orbita schedule add --type task --title "Review PRs" --start 14:00 --end 15:00 --ref <task-id>
  orbita schedule add --type break --title "Lunch" --start 12:00 --end 13:00 --date 2024-01-15`,
	Aliases: []string{"block", "new"},
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cli.GetApp()
		if app == nil || app.AddBlockHandler == nil {
			fmt.Println("Schedule commands require database connection.")
			fmt.Println("Start services with: docker-compose up -d")
			return nil
		}

		// Validate block type
		blockType := domain.BlockType(strings.ToLower(addBlockType))
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
			return fmt.Errorf("invalid block type: %s (valid: task, habit, meeting, focus, break)", addBlockType)
		}

		// Parse date
		var date time.Time
		var err error
		if addDate != "" {
			date, err = time.Parse("2006-01-02", addDate)
			if err != nil {
				return fmt.Errorf("invalid date format, use YYYY-MM-DD: %w", err)
			}
		} else {
			date = time.Now()
		}

		// Parse times
		startParsed, err := time.Parse("15:04", addStartTime)
		if err != nil {
			return fmt.Errorf("invalid start time format, use HH:MM: %w", err)
		}

		endParsed, err := time.Parse("15:04", addEndTime)
		if err != nil {
			return fmt.Errorf("invalid end time format, use HH:MM: %w", err)
		}

		// Combine date with times
		startTime := time.Date(date.Year(), date.Month(), date.Day(),
			startParsed.Hour(), startParsed.Minute(), 0, 0, time.Local)
		endTime := time.Date(date.Year(), date.Month(), date.Day(),
			endParsed.Hour(), endParsed.Minute(), 0, 0, time.Local)

		// Parse reference ID if provided
		var refID uuid.UUID
		if addReferenceID != "" {
			refID, err = uuid.Parse(addReferenceID)
			if err != nil {
				return fmt.Errorf("invalid reference ID: %w", err)
			}
		}

		cmdData := commands.AddBlockCommand{
			UserID:      app.CurrentUserID,
			Date:        date,
			BlockType:   string(blockType),
			ReferenceID: refID,
			Title:       addTitle,
			StartTime:   startTime,
			EndTime:     endTime,
		}

		result, err := app.AddBlockHandler.Handle(cmd.Context(), cmdData)
		if err != nil {
			return fmt.Errorf("failed to add block: %w", err)
		}

		duration := endTime.Sub(startTime)
		fmt.Printf("Added %s block to schedule\n", blockType)
		fmt.Println(strings.Repeat("-", 40))
		fmt.Printf("  Title: %s\n", addTitle)
		fmt.Printf("  Time:  %s - %s (%s)\n", addStartTime, addEndTime, formatDuration(duration))
		fmt.Printf("  Date:  %s\n", date.Format("Monday, January 2, 2006"))
		fmt.Printf("  Block ID: %s\n", result.BlockID)

		return nil
	},
}

func init() {
	addCmd.Flags().StringVarP(&addBlockType, "type", "t", "focus", "block type (task, habit, meeting, focus, break)")
	addCmd.Flags().StringVar(&addTitle, "title", "", "block title (required)")
	addCmd.Flags().StringVarP(&addDate, "date", "d", "", "date for the block (YYYY-MM-DD, default: today)")
	addCmd.Flags().StringVar(&addStartTime, "start", "", "start time (HH:MM, required)")
	addCmd.Flags().StringVar(&addEndTime, "end", "", "end time (HH:MM, required)")
	addCmd.Flags().StringVar(&addReferenceID, "ref", "", "reference ID for task/habit/meeting")

	addCmd.MarkFlagRequired("title")
	addCmd.MarkFlagRequired("start")
	addCmd.MarkFlagRequired("end")
}
