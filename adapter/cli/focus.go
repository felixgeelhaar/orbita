package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var (
	focusDuration int
	focusBreak    int
	focusTask     string
)

var focusCmd = &cobra.Command{
	Use:   "focus",
	Short: "Start a focus session",
	Long: `Start a focused work session with a timer.

The focus command helps you concentrate on a single task or activity
using the Pomodoro technique. After the focus period, take a break.

Examples:
  orbita focus --duration 25           # 25 minute focus session
  orbita focus --duration 25 --break 5 # 25 min focus, 5 min break
  orbita focus --task abc123           # Focus on specific task`,
	Aliases: []string{"pomodoro", "timer"},
	RunE: func(cmd *cobra.Command, args []string) error {
		duration := time.Duration(focusDuration) * time.Minute
		breakDuration := time.Duration(focusBreak) * time.Minute

		// Get task title if specified
		taskTitle := ""
		if focusTask != "" {
			app := GetApp()
			if app != nil && app.ListTasksHandler != nil {
				taskID, err := uuid.Parse(focusTask)
				if err == nil {
					// Try to find the task title
					taskTitle = fmt.Sprintf("Task %s", taskID.String()[:8])
				}
			}
		}

		fmt.Println()
		fmt.Println(strings.Repeat("=", 50))
		fmt.Println("  FOCUS MODE")
		fmt.Println(strings.Repeat("=", 50))
		if taskTitle != "" {
			fmt.Printf("  Working on: %s\n", taskTitle)
		}
		fmt.Printf("  Duration: %d minutes\n", focusDuration)
		if focusBreak > 0 {
			fmt.Printf("  Break: %d minutes\n", focusBreak)
		}
		fmt.Println()
		fmt.Println("  Press Ctrl+C to end session early")
		fmt.Println(strings.Repeat("-", 50))

		// Create context for cancellation
		ctx, cancel := context.WithCancel(cmd.Context())
		defer cancel()

		// Handle interrupt signal
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			fmt.Println("\n\n  Session interrupted!")
			cancel()
		}()

		// Start focus timer
		startTime := time.Now()
		endTime := startTime.Add(duration)

		fmt.Println()
		completed := runTimer(ctx, "FOCUS", duration, endTime)

		if completed {
			fmt.Println("\n  Focus session complete!")

			// If break is configured, start break timer
			if focusBreak > 0 {
				fmt.Println()
				fmt.Println(strings.Repeat("-", 50))
				fmt.Println("  BREAK TIME")
				fmt.Println(strings.Repeat("-", 50))

				breakEndTime := time.Now().Add(breakDuration)
				breakCompleted := runTimer(ctx, "BREAK", breakDuration, breakEndTime)

				if breakCompleted {
					fmt.Println("\n  Break complete! Ready for next session.")
				}
			}
		}

		// Show session summary
		elapsed := time.Since(startTime)
		fmt.Println()
		fmt.Println(strings.Repeat("=", 50))
		fmt.Printf("  Session ended. Total time: %s\n", formatElapsed(elapsed))
		fmt.Println(strings.Repeat("=", 50))
		fmt.Println()

		return nil
	},
}

func runTimer(ctx context.Context, label string, duration time.Duration, endTime time.Time) bool {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case now := <-ticker.C:
			remaining := endTime.Sub(now)
			if remaining <= 0 {
				fmt.Printf("\r  [%s] %s - DONE!          ", label, formatDurationTimer(0))
				return true
			}

			// Progress bar
			progress := float64(duration-remaining) / float64(duration)
			barWidth := 30
			filled := int(progress * float64(barWidth))
			bar := strings.Repeat("=", filled) + strings.Repeat("-", barWidth-filled)

			fmt.Printf("\r  [%s] %s [%s] %.0f%%",
				label,
				formatDurationTimer(remaining),
				bar,
				progress*100,
			)
		}
	}
}

func formatDurationTimer(d time.Duration) string {
	d = d.Round(time.Second)
	m := d / time.Minute
	s := (d % time.Minute) / time.Second
	return fmt.Sprintf("%02d:%02d", m, s)
}

func formatElapsed(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	m := (d % time.Hour) / time.Minute
	s := (d % time.Minute) / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func init() {
	focusCmd.Flags().IntVarP(&focusDuration, "duration", "d", 25, "focus duration in minutes")
	focusCmd.Flags().IntVarP(&focusBreak, "break", "b", 0, "break duration in minutes (0 = no break)")
	focusCmd.Flags().StringVarP(&focusTask, "task", "t", "", "task ID to focus on")

	rootCmd.AddCommand(focusCmd)
}
