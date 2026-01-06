package cli

import (
	"context"
	"fmt"
	"strings"

	habitCommands "github.com/felixgeelhaar/orbita/internal/habits/application/commands"
	habitQueries "github.com/felixgeelhaar/orbita/internal/habits/application/queries"
	"github.com/felixgeelhaar/orbita/internal/productivity/application/commands"
	"github.com/felixgeelhaar/orbita/internal/productivity/application/queries"
	"github.com/spf13/cobra"
)

var doneCmd = &cobra.Command{
	Use:   "done <id-prefix>",
	Short: "Mark a task or habit as complete",
	Long: `Quickly mark a task or habit as complete using just the first few characters of its ID.

The command will search for matching tasks first, then habits.
If multiple items match, you'll be shown the options.

Examples:
  orbita done abc1      # Complete task/habit starting with abc1
  orbita done abc123    # More specific match
  orbita done           # Show completable items`,
	Aliases: []string{"complete", "finish", "x"},
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetApp()
		if app == nil {
			fmt.Println("Done command requires database connection.")
			fmt.Println("Start services with: docker-compose up -d")
			return nil
		}

		if len(args) == 0 {
			// Show completable items
			return showCompletableItems(cmd, app)
		}

		prefix := strings.ToLower(args[0])
		return completeByPrefix(cmd.Context(), app, prefix)
	},
}

func showCompletableItems(cmd *cobra.Command, app *App) error {
	fmt.Println("\n  COMPLETABLE ITEMS")
	fmt.Println(strings.Repeat("=", 60))

	// Show pending tasks
	if app.ListTasksHandler != nil {
		query := queries.ListTasksQuery{
			UserID: app.CurrentUserID,
			Status: "pending",
			Limit:  10,
		}
		tasks, err := app.ListTasksHandler.Handle(cmd.Context(), query)
		if err == nil && len(tasks) > 0 {
			fmt.Println("\n  Tasks:")
			for _, t := range tasks {
				fmt.Printf("    [%s] %s\n", t.ID.String()[:8], t.Title)
			}
		}
	}

	// Show due habits
	if app.ListHabitsHandler != nil {
		query := habitQueries.ListHabitsQuery{
			UserID:       app.CurrentUserID,
			OnlyDueToday: true,
		}
		habits, err := app.ListHabitsHandler.Handle(cmd.Context(), query)
		if err == nil {
			incomplete := 0
			for _, h := range habits {
				if !h.CompletedToday {
					incomplete++
				}
			}
			if incomplete > 0 {
				fmt.Println("\n  Habits due today:")
				for _, h := range habits {
					if !h.CompletedToday {
						fmt.Printf("    [%s] %s\n", h.ID.String()[:8], h.Name)
					}
				}
			}
		}
	}

	fmt.Println("\n  Usage: orbita done <id-prefix>")
	fmt.Println()

	return nil
}

func completeByPrefix(ctx context.Context, app *App, prefix string) error {
	// Search tasks first
	if app.ListTasksHandler != nil && app.CompleteTaskHandler != nil {
		query := queries.ListTasksQuery{
			UserID: app.CurrentUserID,
			Status: "pending",
		}
		tasks, err := app.ListTasksHandler.Handle(ctx, query)
		if err == nil {
			var matches []queries.TaskDTO
			for _, t := range tasks {
				if strings.HasPrefix(strings.ToLower(t.ID.String()), prefix) {
					matches = append(matches, t)
				}
			}

			if len(matches) == 1 {
				// Complete the task
				return completeTask(ctx, app, matches[0])
			} else if len(matches) > 1 {
				fmt.Println("Multiple tasks match. Be more specific:")
				for _, t := range matches {
					fmt.Printf("  [%s] %s\n", t.ID.String()[:8], t.Title)
				}
				return nil
			}
		}
	}

	// Search habits
	if app.ListHabitsHandler != nil && app.LogCompletionHandler != nil {
		query := habitQueries.ListHabitsQuery{
			UserID:       app.CurrentUserID,
			OnlyDueToday: true,
		}
		habits, err := app.ListHabitsHandler.Handle(ctx, query)
		if err == nil {
			var matches []habitQueries.HabitDTO
			for _, h := range habits {
				if !h.CompletedToday && strings.HasPrefix(strings.ToLower(h.ID.String()), prefix) {
					matches = append(matches, h)
				}
			}

			if len(matches) == 1 {
				// Complete the habit
				return completeHabit(ctx, app, matches[0])
			} else if len(matches) > 1 {
				fmt.Println("Multiple habits match. Be more specific:")
				for _, h := range matches {
					fmt.Printf("  [%s] %s\n", h.ID.String()[:8], h.Name)
				}
				return nil
			}
		}
	}

	fmt.Printf("No pending task or due habit found matching '%s'\n", prefix)
	return nil
}

func completeTask(ctx context.Context, app *App, task queries.TaskDTO) error {
	cmd := commands.CompleteTaskCommand{
		TaskID: task.ID,
		UserID: app.CurrentUserID,
	}

	if err := app.CompleteTaskHandler.Handle(ctx, cmd); err != nil {
		return fmt.Errorf("failed to complete task: %w", err)
	}

	fmt.Printf("Task completed: %s\n", task.Title)
	return nil
}

func completeHabit(ctx context.Context, app *App, habit habitQueries.HabitDTO) error {
	cmd := habitCommands.LogCompletionCommand{
		HabitID: habit.ID,
		UserID:  app.CurrentUserID,
	}

	result, err := app.LogCompletionHandler.Handle(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to log habit completion: %w", err)
	}

	streakMsg := ""
	if result.Streak > 1 {
		streakMsg = fmt.Sprintf(" (streak: %d)", result.Streak)
	} else {
		streakMsg = " (new streak started!)"
	}

	fmt.Printf("Habit completed: %s%s\n", habit.Name, streakMsg)
	if app.AdjustHabitFrequencyHandler != nil {
		_, _ = app.AdjustHabitFrequencyHandler.Handle(ctx, habitCommands.AdjustHabitFrequencyCommand{
			UserID:     app.CurrentUserID,
			WindowDays: 14,
		})
	}
	return nil
}

func init() {
	rootCmd.AddCommand(doneCmd)
}
