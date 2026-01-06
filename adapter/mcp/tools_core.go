package mcp

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/felixgeelhaar/mcp-go"
	"github.com/felixgeelhaar/orbita/adapter/cli"
	billingDomain "github.com/felixgeelhaar/orbita/internal/billing/domain"
	calendarApp "github.com/felixgeelhaar/orbita/internal/calendar/application"
	googleCalendar "github.com/felixgeelhaar/orbita/internal/calendar/infrastructure/google"
	habitCommands "github.com/felixgeelhaar/orbita/internal/habits/application/commands"
	habitQueries "github.com/felixgeelhaar/orbita/internal/habits/application/queries"
	meetingCommands "github.com/felixgeelhaar/orbita/internal/meetings/application/commands"
	"github.com/felixgeelhaar/orbita/internal/productivity/application/commands"
	"github.com/felixgeelhaar/orbita/internal/productivity/application/queries"
	scheduleQueries "github.com/felixgeelhaar/orbita/internal/scheduling/application/queries"
)

type addInput struct {
	Description string `json:"description" jsonschema:"required"`
}

type doneInput struct {
	Prefix string `json:"prefix,omitempty"`
}

type statsInput struct {
	Period string `json:"period,omitempty"`
}

type planInput struct {
	Date    string `json:"date,omitempty"`
	Auto    bool   `json:"auto,omitempty"`
	Preview bool   `json:"preview,omitempty"`
}

type focusInput struct {
	DurationMinutes int    `json:"duration_minutes,omitempty"`
	BreakMinutes    int    `json:"break_minutes,omitempty"`
	TaskID          string `json:"task_id,omitempty"`
}

type exportInput struct {
	Format string `json:"format,omitempty"`
	Days   int    `json:"days,omitempty"`
}

type syncInput struct {
	Days              int      `json:"days,omitempty"`
	DeleteMissing     bool     `json:"delete_missing,omitempty"`
	CalendarID        string   `json:"calendar_id,omitempty"`
	UseConfigCalendar bool     `json:"use_config_calendar,omitempty"`
	Attendees         []string `json:"attendees,omitempty"`
	Reminders         []int    `json:"reminders,omitempty"`
}

type adaptInput struct {
	Habits     bool `json:"habits,omitempty"`
	Meetings   bool `json:"meetings,omitempty"`
	WindowDays int  `json:"window_days,omitempty"`
}

func registerCoreTools(srv *mcp.Server, deps ToolDependencies) error {
	app := deps.App

	srv.Tool("cli.health").
		Description("Check CLI wiring health").
		Handler(func(ctx context.Context, input struct{}) (map[string]string, error) {
			if app == nil {
				return nil, errors.New("app not initialized")
			}
			return map[string]string{"status": "ok"}, nil
		})

	srv.Tool("cli.version").
		Description("Get CLI version information").
		Handler(func(ctx context.Context, input struct{}) (map[string]string, error) {
			return map[string]string{
				"version":   cli.Version,
				"commit":    cli.Commit,
				"buildDate": cli.BuildDate,
			}, nil
		})

	srv.Tool("cli.add").
		Description("Quick add a task with natural language").
		Handler(func(ctx context.Context, input addInput) (map[string]any, error) {
			if app == nil || app.CreateTaskHandler == nil {
				return nil, errors.New("quick add requires database connection")
			}
			if strings.TrimSpace(input.Description) == "" {
				return nil, errors.New("description is required")
			}

			parsed := parseNaturalLanguage(input.Description)
			durationMins := 0
			if parsed.duration > 0 {
				durationMins = int(parsed.duration.Minutes())
			}

			cmd := commands.CreateTaskCommand{
				UserID:          app.CurrentUserID,
				Title:           parsed.title,
				Priority:        parsed.priority,
				DurationMinutes: durationMins,
				DueDate:         parsed.dueDate,
			}

			result, err := app.CreateTaskHandler.Handle(ctx, cmd)
			if err != nil {
				return nil, err
			}

			return map[string]any{
				"task_id":  result.TaskID,
				"title":    parsed.title,
				"priority": parsed.priority,
				"duration": durationMins,
				"due_date": parsed.dueDate,
			}, nil
		})

	srv.Tool("cli.done").
		Description("Mark a task or habit complete by ID prefix or list completable items").
		Handler(func(ctx context.Context, input doneInput) (any, error) {
			if app == nil {
				return nil, errors.New("done requires database connection")
			}
			if strings.TrimSpace(input.Prefix) == "" {
				return listCompletableItems(ctx, app)
			}
			return completeByPrefix(ctx, app, strings.ToLower(input.Prefix))
		})

	srv.Tool("cli.stats").
		Description("Show productivity statistics").
		Handler(func(ctx context.Context, input statsInput) (map[string]any, error) {
			if app == nil {
				return nil, errors.New("stats requires database connection")
			}

			taskStats := buildTaskStats(ctx, app)
			habitStats := buildHabitStats(ctx, app)
			scheduleStats := buildScheduleStats(ctx, app, time.Now())

			return map[string]any{
				"period":   input.Period,
				"tasks":    taskStats,
				"habits":   habitStats,
				"schedule": scheduleStats,
			}, nil
		})

	srv.Tool("cli.review").
		Description("Review items needing attention").
		Handler(func(ctx context.Context, input struct{}) (map[string]any, error) {
			if app == nil {
				return nil, errors.New("review requires database connection")
			}

			now := time.Now()
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

			overdue := reviewOverdueTasks(ctx, app, today)
			missed := reviewMissedBlocks(ctx, app, now)
			habits := reviewHabits(ctx, app)

			total := overdue.TotalIssues + missed.TotalIssues + habits.TotalIssues
			return map[string]any{
				"overdue_tasks": overdue,
				"missed_blocks": missed,
				"habits":        habits,
				"total_issues":  total,
			}, nil
		})

	srv.Tool("cli.plan").
		Description("Plan your day and optionally auto-schedule").
		Handler(func(ctx context.Context, input planInput) (map[string]any, error) {
			if app == nil {
				return nil, errors.New("planning requires database connection")
			}

			var targetDate time.Time
			if input.Date != "" {
				parsed, err := time.Parse(dateLayout, input.Date)
				if err != nil {
					return nil, fmt.Errorf("invalid date format, use YYYY-MM-DD: %w", err)
				}
				targetDate = parsed
			} else {
				now := time.Now()
				targetDate = time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
			}

			schedule := fetchSchedule(ctx, app, targetDate)
			slots := fetchAvailableSlots(ctx, app, targetDate)
			tasks := fetchSchedulableTasks(ctx, app)
			habits := fetchSchedulableHabits(ctx, app)

			var autoResult any
			if input.Auto && !input.Preview {
				autoResult = runAutoSchedule(ctx, app, targetDate, tasks, habits, nil)
			}

			return map[string]any{
				"date":         targetDate,
				"schedule":     schedule,
				"slots":        slots,
				"tasks":        tasks,
				"habits":       habits,
				"auto_result":  autoResult,
				"auto_enabled": input.Auto,
				"preview":      input.Preview,
			}, nil
		})

	srv.Tool("cli.focus").
		Description("Create a focus session (metadata only, no timer)").
		Handler(func(ctx context.Context, input focusInput) (map[string]any, error) {
			if input.DurationMinutes <= 0 {
				input.DurationMinutes = 25
			}
			if input.BreakMinutes < 0 {
				return nil, errors.New("break_minutes must be >= 0")
			}

			start := time.Now()
			end := start.Add(time.Duration(input.DurationMinutes) * time.Minute)
			return map[string]any{
				"task_id":          input.TaskID,
				"duration_minutes": input.DurationMinutes,
				"break_minutes":    input.BreakMinutes,
				"started_at":       start,
				"ends_at":          end,
				"note":             "timer not executed in MCP",
			}, nil
		})

	srv.Tool("cli.export").
		Description("Export schedule to ICS").
		Handler(func(ctx context.Context, input exportInput) (map[string]any, error) {
			if app == nil || app.GetScheduleHandler == nil {
				return nil, errors.New("export requires database connection")
			}
			if input.Format == "" {
				input.Format = "ics"
			}
			if input.Format != "ics" && input.Format != "ical" {
				return nil, fmt.Errorf("unsupported format: %s (supported: ics)", input.Format)
			}
			if input.Days <= 0 {
				input.Days = 7
			}

			blocks := gatherScheduleBlocks(ctx, app, input.Days)
			if len(blocks) == 0 {
				return map[string]any{
					"blocks": 0,
					"ics":    "",
				}, nil
			}

			ics := generateICS(blocks)
			return map[string]any{
				"blocks": len(blocks),
				"ics":    ics,
			}, nil
		})

	srv.Tool("cli.today").
		Description("Show today's dashboard data").
		Handler(func(ctx context.Context, input struct{}) (map[string]any, error) {
			if app == nil {
				return nil, errors.New("dashboard requires database connection")
			}

			today := time.Now()
			schedule := fetchSchedule(ctx, app, today)
			tasks := fetchPendingTasks(ctx, app, 5)
			habits := fetchDueHabits(ctx, app)

			return map[string]any{
				"date":     today,
				"schedule": schedule,
				"tasks":    tasks,
				"habits":   habits,
			}, nil
		})

	srv.Tool("cli.sync").
		Description("Sync schedule to external calendar").
		Handler(func(ctx context.Context, input syncInput) (map[string]any, error) {
			if app == nil || app.GetScheduleHandler == nil {
				return nil, errors.New("sync requires database connection")
			}
			if app.CalendarSyncer == nil {
				return nil, errors.New("calendar sync not configured")
			}
			if input.Days <= 0 {
				input.Days = 7
			}

			blocks := gatherScheduleBlocks(ctx, app, input.Days)
			if len(blocks) == 0 {
				return map[string]any{
					"created": 0,
					"updated": 0,
					"deleted": 0,
					"failed":  0,
				}, nil
			}

			if input.UseConfigCalendar && app.SettingsService != nil {
				if input.CalendarID == "" {
					if storedID, err := app.SettingsService.GetCalendarID(ctx, app.CurrentUserID); err == nil && storedID != "" {
						input.CalendarID = storedID
					}
				}
				if !input.DeleteMissing {
					if storedDelete, err := app.SettingsService.GetDeleteMissing(ctx, app.CurrentUserID); err == nil && storedDelete {
						input.DeleteMissing = true
					}
				}
			}

			syncer := app.CalendarSyncer
			if googleSyncer, ok := syncer.(*googleCalendar.Syncer); ok {
				if input.DeleteMissing {
					googleSyncer = googleSyncer.WithDeleteMissing(true)
				}
				if input.CalendarID != "" {
					googleSyncer = googleSyncer.WithCalendarID(input.CalendarID)
				} else if !input.UseConfigCalendar {
					googleSyncer = googleSyncer.WithCalendarID("primary")
				}
				if len(input.Attendees) > 0 {
					googleSyncer = googleSyncer.WithAttendees(input.Attendees)
				}
				if len(input.Reminders) > 0 {
					googleSyncer = googleSyncer.WithReminders(input.Reminders)
				}
				syncer = googleSyncer
			}

			result, err := syncer.Sync(ctx, app.CurrentUserID, blocks)
			if err != nil {
				return nil, err
			}

			return map[string]any{
				"created": result.Created,
				"updated": result.Updated,
				"deleted": result.Deleted,
				"failed":  result.Failed,
			}, nil
		})

	srv.Tool("cli.adapt").
		Description("Adjust habit and meeting cadences").
		Handler(func(ctx context.Context, input adaptInput) (map[string]any, error) {
			if app == nil {
				return nil, errors.New("adaptive frequency requires database connection")
			}
			if err := cli.RequireEntitlement(ctx, app, billingDomain.ModuleAdaptiveFrequency); err != nil {
				return nil, err
			}
			if input.WindowDays <= 0 {
				input.WindowDays = 14
			}

			runHabits := input.Habits
			runMeetings := input.Meetings
			if !runHabits && !runMeetings {
				runHabits = true
				runMeetings = true
			}

			result := map[string]any{}
			if runHabits {
				if app.AdjustHabitFrequencyHandler == nil {
					return nil, errors.New("habit adaptive frequency not configured")
				}
				out, err := app.AdjustHabitFrequencyHandler.Handle(ctx, habitCommands.AdjustHabitFrequencyCommand{
					UserID:     app.CurrentUserID,
					WindowDays: input.WindowDays,
				})
				if err != nil {
					return nil, err
				}
				result["habits"] = out
			}
			if runMeetings {
				if app.AdjustMeetingCadenceHandler == nil {
					return nil, errors.New("meeting adaptive cadence not configured")
				}
				out, err := app.AdjustMeetingCadenceHandler.Handle(ctx, meetingCommands.AdjustMeetingCadenceCommand{
					UserID: app.CurrentUserID,
				})
				if err != nil {
					return nil, err
				}
				result["meetings"] = out
			}

			return result, nil
		})

	return nil
}

type parsedInput struct {
	title    string
	priority string
	duration time.Duration
	dueDate  *time.Time
}

func parseNaturalLanguage(input string) parsedInput {
	result := parsedInput{
		title: input,
	}

	result.priority, result.title = extractPriority(result.title)
	result.duration, result.title = extractDuration(result.title)
	result.dueDate, result.title = extractDueDate(result.title)
	result.title = cleanTitle(result.title)
	return result
}

func extractPriority(input string) (string, string) {
	if strings.Contains(input, "!!!") {
		return "urgent", strings.ReplaceAll(input, "!!!", "")
	}
	if strings.Contains(input, "!!") {
		return "high", strings.ReplaceAll(input, "!!", "")
	}
	if strings.Contains(input, "!") && !strings.Contains(input, "!!") {
		return "medium", strings.ReplaceAll(input, "!", "")
	}

	lower := strings.ToLower(input)
	priorities := map[string]string{
		"urgent priority": "urgent",
		"high priority":   "high",
		"medium priority": "medium",
		"low priority":    "low",
		"urgent":          "urgent",
		"high":            "high",
		"low":             "low",
	}

	for keyword, priority := range priorities {
		if strings.Contains(lower, keyword) {
			re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(keyword) + `\b`)
			return priority, re.ReplaceAllString(input, "")
		}
	}

	return "", input
}

func extractDuration(input string) (time.Duration, string) {
	patterns := []struct {
		regex      *regexp.Regexp
		multiplier time.Duration
	}{
		{regexp.MustCompile(`(\d+(?:\.\d+)?)\s*h(?:ours?)?`), time.Hour},
		{regexp.MustCompile(`(\d+)\s*min(?:utes?)?`), time.Minute},
		{regexp.MustCompile(`for\s+(\d+(?:\.\d+)?)\s*h(?:ours?)?`), time.Hour},
		{regexp.MustCompile(`for\s+(\d+)\s*min(?:utes?)?`), time.Minute},
	}

	for _, p := range patterns {
		if matches := p.regex.FindStringSubmatch(strings.ToLower(input)); len(matches) > 1 {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				duration := time.Duration(val * float64(p.multiplier))
				cleaned := p.regex.ReplaceAllString(input, "")
				return duration, cleaned
			}
		}
	}

	return 0, input
}

func extractDueDate(input string) (*time.Time, string) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	lower := strings.ToLower(input)

	relativeDates := map[string]time.Time{
		"today":     today,
		"tomorrow":  today.AddDate(0, 0, 1),
		"next week": today.AddDate(0, 0, 7),
	}

	for keyword, date := range relativeDates {
		if strings.Contains(lower, keyword) {
			d := date
			re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(keyword) + `\b`)
			return &d, re.ReplaceAllString(input, "")
		}
	}

	days := map[string]time.Weekday{
		"monday":    time.Monday,
		"tuesday":   time.Tuesday,
		"wednesday": time.Wednesday,
		"thursday":  time.Thursday,
		"friday":    time.Friday,
		"saturday":  time.Saturday,
		"sunday":    time.Sunday,
	}

	for dayName, weekday := range days {
		if strings.Contains(lower, dayName) {
			target := nextWeekday(today, weekday)
			re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(dayName) + `\b`)
			return &target, re.ReplaceAllString(input, "")
		}
	}

	if dateMatch := regexp.MustCompile(`\b(\d{4}-\d{2}-\d{2})\b`).FindStringSubmatch(input); len(dateMatch) > 1 {
		if date, err := time.Parse("2006-01-02", dateMatch[1]); err == nil {
			re := regexp.MustCompile(`\b` + regexp.QuoteMeta(dateMatch[1]) + `\b`)
			return &date, re.ReplaceAllString(input, "")
		}
	}

	return nil, input
}

func nextWeekday(from time.Time, weekday time.Weekday) time.Time {
	daysAhead := (int(weekday) - int(from.Weekday()) + 7) % 7
	if daysAhead == 0 {
		daysAhead = 7
	}
	return from.AddDate(0, 0, daysAhead)
}

func cleanTitle(input string) string {
	cleaned := strings.TrimSpace(input)
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	return cleaned
}

func listCompletableItems(ctx context.Context, app *cli.App) (any, error) {
	result := map[string]any{
		"tasks":  []queries.TaskDTO{},
		"habits": []habitQueries.HabitDTO{},
	}

	if app.ListTasksHandler != nil {
		query := queries.ListTasksQuery{
			UserID: app.CurrentUserID,
			Status: "pending",
			Limit:  10,
		}
		tasks, err := app.ListTasksHandler.Handle(ctx, query)
		if err == nil && len(tasks) > 0 {
			result["tasks"] = tasks
		}
	}

	if app.ListHabitsHandler != nil {
		query := habitQueries.ListHabitsQuery{
			UserID:       app.CurrentUserID,
			OnlyDueToday: true,
		}
		habits, err := app.ListHabitsHandler.Handle(ctx, query)
		if err == nil {
			filtered := make([]habitQueries.HabitDTO, 0, len(habits))
			for _, habit := range habits {
				if !habit.CompletedToday {
					filtered = append(filtered, habit)
				}
			}
			result["habits"] = filtered
		}
	}

	return result, nil
}

func completeByPrefix(ctx context.Context, app *cli.App, prefix string) (any, error) {
	if app.ListTasksHandler != nil && app.CompleteTaskHandler != nil {
		query := queries.ListTasksQuery{
			UserID: app.CurrentUserID,
			Status: "pending",
		}
		tasks, err := app.ListTasksHandler.Handle(ctx, query)
		if err == nil {
			matches := make([]queries.TaskDTO, 0)
			for _, task := range tasks {
				if strings.HasPrefix(strings.ToLower(task.ID.String()), prefix) {
					matches = append(matches, task)
				}
			}
			if len(matches) == 1 {
				cmd := commands.CompleteTaskCommand{
					TaskID: matches[0].ID,
					UserID: app.CurrentUserID,
				}
				if err := app.CompleteTaskHandler.Handle(ctx, cmd); err != nil {
					return nil, err
				}
				return map[string]any{"task_completed": matches[0]}, nil
			}
			if len(matches) > 1 {
				return map[string]any{"task_matches": matches}, nil
			}
		}
	}

	if app.ListHabitsHandler != nil && app.LogCompletionHandler != nil {
		query := habitQueries.ListHabitsQuery{
			UserID:       app.CurrentUserID,
			OnlyDueToday: true,
		}
		habits, err := app.ListHabitsHandler.Handle(ctx, query)
		if err == nil {
			matches := make([]habitQueries.HabitDTO, 0)
			for _, habit := range habits {
				if !habit.CompletedToday && strings.HasPrefix(strings.ToLower(habit.ID.String()), prefix) {
					matches = append(matches, habit)
				}
			}
			if len(matches) == 1 {
				cmd := habitCommands.LogCompletionCommand{
					HabitID: matches[0].ID,
					UserID:  app.CurrentUserID,
				}
				result, err := app.LogCompletionHandler.Handle(ctx, cmd)
				if err != nil {
					return nil, err
				}
				return map[string]any{
					"habit_completed": matches[0],
					"streak":          result.Streak,
				}, nil
			}
			if len(matches) > 1 {
				return map[string]any{"habit_matches": matches}, nil
			}
		}
	}

	return map[string]any{"message": "no pending task or due habit found"}, nil
}

func buildTaskStats(ctx context.Context, app *cli.App) map[string]any {
	if app.ListTasksHandler == nil {
		return map[string]any{}
	}

	allTasks, err := app.ListTasksHandler.Handle(ctx, queries.ListTasksQuery{
		UserID:     app.CurrentUserID,
		IncludeAll: true,
	})
	if err != nil {
		return map[string]any{"error": err.Error()}
	}

	pending := 0
	inProgress := 0
	completed := 0
	archived := 0
	urgent := 0
	high := 0
	medium := 0
	low := 0
	overdue := 0

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	for _, task := range allTasks {
		switch task.Status {
		case "pending":
			pending++
		case "in_progress":
			inProgress++
		case "completed":
			completed++
		case "archived":
			archived++
		}

		switch task.Priority {
		case "urgent":
			urgent++
		case "high":
			high++
		case "medium":
			medium++
		case "low":
			low++
		}

		if task.DueDate != nil && task.DueDate.Before(today) && task.Status != "completed" && task.Status != "archived" {
			overdue++
		}
	}

	total := len(allTasks)
	completionRate := 0.0
	if total > 0 {
		completionRate = float64(completed) / float64(total) * 100
	}

	return map[string]any{
		"total":           total,
		"pending":         pending,
		"in_progress":     inProgress,
		"completed":       completed,
		"archived":        archived,
		"urgent":          urgent,
		"high":            high,
		"medium":          medium,
		"low":             low,
		"overdue":         overdue,
		"completion_rate": completionRate,
	}
}

func buildHabitStats(ctx context.Context, app *cli.App) map[string]any {
	if app.ListHabitsHandler == nil {
		return map[string]any{}
	}

	habits, err := app.ListHabitsHandler.Handle(ctx, habitQueries.ListHabitsQuery{
		UserID: app.CurrentUserID,
	})
	if err != nil {
		return map[string]any{"error": err.Error()}
	}

	if len(habits) == 0 {
		return map[string]any{"total": 0}
	}

	totalHabits := len(habits)
	activeStreaks := 0
	brokenStreaks := 0
	longestStreak := 0
	longestStreakHabit := ""
	totalCompletions := 0
	completedToday := 0
	dueToday := 0

	for _, habit := range habits {
		totalCompletions += habit.TotalDone
		if habit.Streak > 0 {
			activeStreaks++
		}
		if habit.BestStreak > 0 && habit.Streak == 0 {
			brokenStreaks++
		}
		if habit.BestStreak > longestStreak {
			longestStreak = habit.BestStreak
			longestStreakHabit = habit.Name
		}
		if habit.CompletedToday {
			completedToday++
		}
		if habit.IsDueToday {
			dueToday++
		}
	}

	return map[string]any{
		"total":              totalHabits,
		"active_streaks":     activeStreaks,
		"broken_streaks":     brokenStreaks,
		"longest_streak":     longestStreak,
		"longest_streak_for": longestStreakHabit,
		"total_completions":  totalCompletions,
		"completed_today":    completedToday,
		"due_today":          dueToday,
	}
}

func buildScheduleStats(ctx context.Context, app *cli.App, date time.Time) map[string]any {
	if app.GetScheduleHandler == nil {
		return map[string]any{}
	}
	schedule, err := app.GetScheduleHandler.Handle(ctx, scheduleQueries.GetScheduleQuery{
		UserID: app.CurrentUserID,
		Date:   date,
	})
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	if schedule == nil {
		return map[string]any{"total_blocks": 0}
	}

	return map[string]any{
		"total_blocks":  len(schedule.Blocks),
		"total_minutes": schedule.TotalScheduledMins,
		"completed":     schedule.CompletedCount,
		"missed":        schedule.MissedCount,
		"pending":       schedule.PendingCount,
	}
}

type reviewSection struct {
	Items       any `json:"items"`
	TotalIssues int `json:"total_issues"`
}

func reviewOverdueTasks(ctx context.Context, app *cli.App, today time.Time) reviewSection {
	if app.ListTasksHandler == nil {
		return reviewSection{}
	}
	overdue, _ := app.ListTasksHandler.Handle(ctx, queries.ListTasksQuery{
		UserID:  app.CurrentUserID,
		Overdue: true,
	})
	dueToday, _ := app.ListTasksHandler.Handle(ctx, queries.ListTasksQuery{
		UserID:   app.CurrentUserID,
		DueToday: true,
	})

	issues := 0
	for _, task := range overdue {
		if task.DueDate != nil {
			issues++
		}
	}
	for _, task := range dueToday {
		if task.Status != "completed" {
			issues++
		}
	}

	return reviewSection{
		Items: map[string]any{
			"overdue":   overdue,
			"due_today": dueToday,
		},
		TotalIssues: issues,
	}
}

func reviewMissedBlocks(ctx context.Context, app *cli.App, now time.Time) reviewSection {
	if app.GetScheduleHandler == nil {
		return reviewSection{}
	}
	schedule, err := app.GetScheduleHandler.Handle(ctx, scheduleQueries.GetScheduleQuery{
		UserID: app.CurrentUserID,
		Date:   now,
	})
	if err != nil || schedule == nil {
		return reviewSection{}
	}

	missed := make([]scheduleQueries.TimeBlockDTO, 0)
	for _, block := range schedule.Blocks {
		if block.Missed {
			missed = append(missed, block)
		}
	}

	return reviewSection{
		Items:       missed,
		TotalIssues: len(missed),
	}
}

func reviewHabits(ctx context.Context, app *cli.App) reviewSection {
	if app.ListHabitsHandler == nil {
		return reviewSection{}
	}
	broken, _ := app.ListHabitsHandler.Handle(ctx, habitQueries.ListHabitsQuery{
		UserID:       app.CurrentUserID,
		BrokenStreak: true,
	})
	due, _ := app.ListHabitsHandler.Handle(ctx, habitQueries.ListHabitsQuery{
		UserID:       app.CurrentUserID,
		OnlyDueToday: true,
	})

	pending := make([]habitQueries.HabitDTO, 0)
	for _, habit := range due {
		if !habit.CompletedToday {
			pending = append(pending, habit)
		}
	}

	return reviewSection{
		Items: map[string]any{
			"broken_streaks": broken,
			"due_today":      pending,
		},
		TotalIssues: len(broken) + len(pending),
	}
}

func fetchSchedule(ctx context.Context, app *cli.App, date time.Time) *scheduleQueries.ScheduleDTO {
	if app.GetScheduleHandler == nil {
		return nil
	}
	schedule, err := app.GetScheduleHandler.Handle(ctx, scheduleQueries.GetScheduleQuery{
		UserID: app.CurrentUserID,
		Date:   date,
	})
	if err != nil {
		return nil
	}
	return schedule
}

func fetchAvailableSlots(ctx context.Context, app *cli.App, date time.Time) []scheduleQueries.TimeSlotDTO {
	if app.FindAvailableSlotsHandler == nil {
		return nil
	}
	query := scheduleQueries.FindAvailableSlotsQuery{
		UserID:      app.CurrentUserID,
		Date:        date,
		MinDuration: 15 * time.Minute,
	}
	slots, err := app.FindAvailableSlotsHandler.Handle(ctx, query)
	if err != nil {
		return nil
	}
	return slots
}

func fetchSchedulableTasks(ctx context.Context, app *cli.App) []queries.TaskDTO {
	if app.ListTasksHandler == nil {
		return nil
	}
	tasks, err := app.ListTasksHandler.Handle(ctx, queries.ListTasksQuery{
		UserID: app.CurrentUserID,
		Status: "pending",
		SortBy: "priority",
	})
	if err != nil {
		return nil
	}

	schedulable := make([]queries.TaskDTO, 0)
	for _, task := range tasks {
		if task.DurationMinutes > 0 {
			schedulable = append(schedulable, task)
		}
	}
	return schedulable
}

func fetchSchedulableHabits(ctx context.Context, app *cli.App) []habitQueries.HabitDTO {
	if app.ListHabitsHandler == nil {
		return nil
	}
	habits, err := app.ListHabitsHandler.Handle(ctx, habitQueries.ListHabitsQuery{
		UserID:       app.CurrentUserID,
		OnlyDueToday: true,
	})
	if err != nil {
		return nil
	}
	return habits
}

func fetchPendingTasks(ctx context.Context, app *cli.App, limit int) []queries.TaskDTO {
	if app.ListTasksHandler == nil {
		return nil
	}
	tasks, err := app.ListTasksHandler.Handle(ctx, queries.ListTasksQuery{
		UserID: app.CurrentUserID,
		Status: "pending",
		Limit:  limit,
	})
	if err != nil {
		return nil
	}
	return tasks
}

func fetchDueHabits(ctx context.Context, app *cli.App) []habitQueries.HabitDTO {
	if app.ListHabitsHandler == nil {
		return nil
	}
	habits, err := app.ListHabitsHandler.Handle(ctx, habitQueries.ListHabitsQuery{
		UserID:       app.CurrentUserID,
		OnlyDueToday: true,
	})
	if err != nil {
		return nil
	}
	return habits
}

func gatherScheduleBlocks(ctx context.Context, app *cli.App, days int) []calendarApp.TimeBlock {
	now := time.Now()
	all := make([]calendarApp.TimeBlock, 0)
	for i := 0; i < days; i++ {
		day := now.AddDate(0, 0, i)
		schedule := fetchSchedule(ctx, app, day)
		if schedule == nil {
			continue
		}
		for _, block := range schedule.Blocks {
			all = append(all, calendarApp.TimeBlock{
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
	return all
}

func generateICS(blocks []calendarApp.TimeBlock) string {
	var sb strings.Builder

	sb.WriteString("BEGIN:VCALENDAR\r\n")
	sb.WriteString("VERSION:2.0\r\n")
	sb.WriteString("PRODID:-//Orbita//Orbita MCP//EN\r\n")
	sb.WriteString("CALSCALE:GREGORIAN\r\n")
	sb.WriteString("METHOD:PUBLISH\r\n")
	sb.WriteString("X-WR-CALNAME:Orbita Schedule\r\n")

	for _, block := range blocks {
		sb.WriteString("BEGIN:VEVENT\r\n")
		sb.WriteString(fmt.Sprintf("UID:%s@orbita\r\n", block.ID.String()))
		sb.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", formatICSTime(time.Now())))
		sb.WriteString(fmt.Sprintf("DTSTART:%s\r\n", formatICSTime(block.StartTime)))
		sb.WriteString(fmt.Sprintf("DTEND:%s\r\n", formatICSTime(block.EndTime)))
		sb.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", escapeICS(block.Title)))

		desc := fmt.Sprintf("Type: %s", block.BlockType)
		if block.Completed {
			desc += "\\nStatus: Completed"
		} else if block.Missed {
			desc += "\\nStatus: Missed"
		}
		sb.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", desc))
		sb.WriteString(fmt.Sprintf("CATEGORIES:%s\r\n", strings.ToUpper(block.BlockType)))

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
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}
