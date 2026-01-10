package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/felixgeelhaar/mcp-go"
	productivityQueries "github.com/felixgeelhaar/orbita/internal/productivity/application/queries"
	schedulingQueries "github.com/felixgeelhaar/orbita/internal/scheduling/application/queries"
)

// DashboardSummary provides a comprehensive overview of the user's productivity state.
type DashboardSummary struct {
	Timestamp       string           `json:"timestamp"`
	TasksSummary    TasksSummary     `json:"tasks_summary"`
	ScheduleSummary ScheduleSummary  `json:"schedule_summary"`
	HabitsSummary   HabitsSummary    `json:"habits_summary"`
	Recommendations []string         `json:"recommendations"`
}

// TasksSummary summarizes task status.
type TasksSummary struct {
	TotalActive  int `json:"total_active"`
	Overdue      int `json:"overdue"`
	DueToday     int `json:"due_today"`
	DueThisWeek  int `json:"due_this_week"`
	HighPriority int `json:"high_priority"`
	Completed7d  int `json:"completed_7d"`
}

// ScheduleSummary summarizes schedule for today and week.
type ScheduleSummary struct {
	BlocksToday      int     `json:"blocks_today"`
	FreeTimeToday    string  `json:"free_time_today"`
	NextBlock        string  `json:"next_block,omitempty"`
	UtilizationToday float64 `json:"utilization_today_percent"`
	BlocksThisWeek   int     `json:"blocks_this_week"`
}

// HabitsSummary summarizes habit tracking.
type HabitsSummary struct {
	ActiveHabits   int `json:"active_habits"`
	DueToday       int `json:"due_today"`
	CompletedToday int `json:"completed_today"`
	CurrentStreak  int `json:"current_streak_avg"`
}

func registerDashboardTools(srv *mcp.Server, deps ToolDependencies) error {
	app := deps.App

	// Dashboard summary tool
	srv.Tool("dashboard.summary").
		Description("Get a comprehensive productivity dashboard with tasks, schedule, habits, and recommendations").
		Handler(func(ctx context.Context, input struct{}) (*DashboardSummary, error) {
			if app == nil {
				return nil, fmt.Errorf("dashboard requires database connection")
			}

			summary := &DashboardSummary{
				Timestamp:       time.Now().Format(time.RFC3339),
				Recommendations: []string{},
			}

			// Gather task summary
			if app.ListTasksHandler != nil {
				// Active tasks
				activeTasks, _ := app.ListTasksHandler.Handle(ctx, productivityQueries.ListTasksQuery{
					UserID: app.CurrentUserID,
					Status: "pending",
					Limit:  100,
				})
				summary.TasksSummary.TotalActive = len(activeTasks)

				// Overdue tasks
				overdueTasks, _ := app.ListTasksHandler.Handle(ctx, productivityQueries.ListTasksQuery{
					UserID:  app.CurrentUserID,
					Overdue: true,
					Limit:   100,
				})
				summary.TasksSummary.Overdue = len(overdueTasks)

				// Due today
				todayTasks, _ := app.ListTasksHandler.Handle(ctx, productivityQueries.ListTasksQuery{
					UserID:   app.CurrentUserID,
					DueToday: true,
					Limit:    100,
				})
				summary.TasksSummary.DueToday = len(todayTasks)

				// High priority
				highPriorityTasks, _ := app.ListTasksHandler.Handle(ctx, productivityQueries.ListTasksQuery{
					UserID:   app.CurrentUserID,
					Status:   "pending",
					Priority: "high",
					Limit:    100,
				})
				summary.TasksSummary.HighPriority = len(highPriorityTasks)

				// Generate recommendations based on tasks
				if summary.TasksSummary.Overdue > 0 {
					summary.Recommendations = append(summary.Recommendations,
						fmt.Sprintf("You have %d overdue tasks. Consider rescheduling or archiving them.", summary.TasksSummary.Overdue))
				}
				if summary.TasksSummary.HighPriority > 3 {
					summary.Recommendations = append(summary.Recommendations,
						"You have many high-priority tasks. Consider re-evaluating priorities.")
				}
			}

			// Gather schedule summary
			if app.GetScheduleHandler != nil {
				now := time.Now()
				today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

				todaySchedule, err := app.GetScheduleHandler.Handle(ctx, schedulingQueries.GetScheduleQuery{
					UserID: app.CurrentUserID,
					Date:   today,
				})
				if err == nil && todaySchedule != nil {
					summary.ScheduleSummary.BlocksToday = len(todaySchedule.Blocks)

					// Calculate utilization (assuming 8 work hours)
					var totalMinutes int
					for _, block := range todaySchedule.Blocks {
						totalMinutes += block.DurationMin
					}
					summary.ScheduleSummary.UtilizationToday = float64(totalMinutes) / (8 * 60) * 100

					// Free time calculation
					freeMinutes := (8 * 60) - totalMinutes
					if freeMinutes > 0 {
						hours := freeMinutes / 60
						mins := freeMinutes % 60
						summary.ScheduleSummary.FreeTimeToday = fmt.Sprintf("%dh %dm", hours, mins)
					} else {
						summary.ScheduleSummary.FreeTimeToday = "Fully booked"
					}

					// Find next block
					for _, block := range todaySchedule.Blocks {
						if block.StartTime.After(now) {
							summary.ScheduleSummary.NextBlock = fmt.Sprintf("%s at %s",
								block.Title, block.StartTime.Format("15:04"))
							break
						}
					}
				}

				// Week schedule - count blocks for next 7 days
				totalWeekBlocks := 0
				for i := 0; i < 7; i++ {
					date := today.AddDate(0, 0, i)
					daySchedule, err := app.GetScheduleHandler.Handle(ctx, schedulingQueries.GetScheduleQuery{
						UserID: app.CurrentUserID,
						Date:   date,
					})
					if err == nil && daySchedule != nil {
						totalWeekBlocks += len(daySchedule.Blocks)
					}
				}
				summary.ScheduleSummary.BlocksThisWeek = totalWeekBlocks

				// Schedule recommendations
				if summary.ScheduleSummary.UtilizationToday > 90 {
					summary.Recommendations = append(summary.Recommendations,
						"Your day is heavily scheduled. Consider adding buffer time between blocks.")
				}
				if summary.ScheduleSummary.BlocksToday == 0 {
					summary.Recommendations = append(summary.Recommendations,
						"No time blocks scheduled today. Consider using auto-schedule for your tasks.")
				}
			}

			// Add general recommendations
			if len(summary.Recommendations) == 0 {
				summary.Recommendations = append(summary.Recommendations,
					"Looking good! Consider doing a quick inbox review.")
			}

			return summary, nil
		})

	// Quick status tool
	type quickStatusInput struct {
		Verbose bool `json:"verbose,omitempty"`
	}

	srv.Tool("dashboard.quick_status").
		Description("Get a quick one-line status of your productivity state").
		Handler(func(ctx context.Context, input quickStatusInput) (map[string]any, error) {
			if app == nil {
				return nil, fmt.Errorf("status requires database connection")
			}

			status := map[string]any{
				"timestamp": time.Now().Format(time.RFC3339),
			}

			// Quick task count
			if app.ListTasksHandler != nil {
				activeTasks, _ := app.ListTasksHandler.Handle(ctx, productivityQueries.ListTasksQuery{
					UserID: app.CurrentUserID,
					Status: "pending",
					Limit:  100,
				})
				overdueTasks, _ := app.ListTasksHandler.Handle(ctx, productivityQueries.ListTasksQuery{
					UserID:  app.CurrentUserID,
					Overdue: true,
					Limit:   100,
				})

				status["active_tasks"] = len(activeTasks)
				status["overdue_tasks"] = len(overdueTasks)

				// Generate status message
				if len(overdueTasks) > 0 {
					status["status"] = "attention_needed"
					status["message"] = fmt.Sprintf("%d active tasks, %d overdue - needs attention",
						len(activeTasks), len(overdueTasks))
				} else if len(activeTasks) > 10 {
					status["status"] = "busy"
					status["message"] = fmt.Sprintf("%d active tasks - consider prioritizing", len(activeTasks))
				} else {
					status["status"] = "good"
					status["message"] = fmt.Sprintf("%d active tasks - on track", len(activeTasks))
				}
			}

			return status, nil
		})

	// Today's focus tool
	srv.Tool("dashboard.today_focus").
		Description("Get the recommended focus areas for today based on priorities and deadlines").
		Handler(func(ctx context.Context, input struct{}) (map[string]any, error) {
			if app == nil {
				return nil, fmt.Errorf("focus recommendations require database connection")
			}

			focus := map[string]any{
				"date":            time.Now().Format("2006-01-02"),
				"focus_areas":     []map[string]any{},
				"time_allocation": map[string]string{},
			}

			focusAreas := []map[string]any{}

			if app.ListTasksHandler != nil {
				// Get overdue tasks first
				overdueTasks, _ := app.ListTasksHandler.Handle(ctx, productivityQueries.ListTasksQuery{
					UserID:  app.CurrentUserID,
					Overdue: true,
					Limit:   3,
				})
				if len(overdueTasks) > 0 {
					focusAreas = append(focusAreas, map[string]any{
						"priority": 1,
						"category": "overdue",
						"reason":   "Address overdue tasks first",
						"tasks":    overdueTasks,
					})
				}

				// Get high priority tasks due today
				todayHighPriority, _ := app.ListTasksHandler.Handle(ctx, productivityQueries.ListTasksQuery{
					UserID:   app.CurrentUserID,
					DueToday: true,
					Priority: "high",
					Limit:    3,
				})
				if len(todayHighPriority) > 0 {
					focusAreas = append(focusAreas, map[string]any{
						"priority": 2,
						"category": "today_high_priority",
						"reason":   "High priority items due today",
						"tasks":    todayHighPriority,
					})
				}

				// Get other high priority tasks
				highPriority, _ := app.ListTasksHandler.Handle(ctx, productivityQueries.ListTasksQuery{
					UserID:   app.CurrentUserID,
					Status:   "pending",
					Priority: "high",
					Limit:    5,
				})
				if len(highPriority) > 0 {
					focusAreas = append(focusAreas, map[string]any{
						"priority": 3,
						"category": "high_priority",
						"reason":   "Important tasks to make progress on",
						"tasks":    highPriority,
					})
				}
			}

			focus["focus_areas"] = focusAreas

			// Time allocation suggestions
			focus["time_allocation"] = map[string]string{
				"deep_work": "9:00 AM - 12:00 PM (peak focus time)",
				"meetings":  "1:00 PM - 3:00 PM (post-lunch)",
				"admin":     "3:00 PM - 4:00 PM (lower energy)",
				"planning":  "4:00 PM - 5:00 PM (end of day review)",
			}

			return focus, nil
		})

	return nil
}
