package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/felixgeelhaar/mcp-go"
	"github.com/felixgeelhaar/orbita/adapter/cli"
	billingDomain "github.com/felixgeelhaar/orbita/internal/billing/domain"
	googleCalendar "github.com/felixgeelhaar/orbita/internal/calendar/infrastructure/google"
	habitQueries "github.com/felixgeelhaar/orbita/internal/habits/application/queries"
	meetingQueries "github.com/felixgeelhaar/orbita/internal/meetings/application/queries"
	"github.com/felixgeelhaar/orbita/internal/productivity/application/queries"
	scheduleCommands "github.com/felixgeelhaar/orbita/internal/scheduling/application/commands"
	scheduleQueries "github.com/felixgeelhaar/orbita/internal/scheduling/application/queries"
	"github.com/felixgeelhaar/orbita/internal/scheduling/domain"
)

type scheduleShowInput struct {
	Date string `json:"date,omitempty"`
}

type scheduleWeekInput struct {
	Offset int  `json:"offset,omitempty"`
	Next   bool `json:"next,omitempty"`
}

type scheduleAvailableInput struct {
	Date  string `json:"date,omitempty"`
	Min   int    `json:"min,omitempty"`
	Start string `json:"start,omitempty"`
	End   string `json:"end,omitempty"`
}

type scheduleAddInput struct {
	Type  string `json:"type" jsonschema:"required"`
	Title string `json:"title" jsonschema:"required"`
	Date  string `json:"date,omitempty"`
	Start string `json:"start" jsonschema:"required"`
	End   string `json:"end" jsonschema:"required"`
	Ref   string `json:"ref,omitempty"`
}

type scheduleCompleteInput struct {
	ScheduleID string `json:"schedule_id" jsonschema:"required"`
	BlockID    string `json:"block_id" jsonschema:"required"`
}

type scheduleRemoveInput struct {
	BlockID string `json:"block_id" jsonschema:"required"`
	Date    string `json:"date,omitempty"`
}

type scheduleRescheduleInput struct {
	BlockID string `json:"block_id" jsonschema:"required"`
	Date    string `json:"date,omitempty"`
	Start   string `json:"start" jsonschema:"required"`
	End     string `json:"end" jsonschema:"required"`
}

type scheduleRescheduleMissedInput struct {
	Date  string `json:"date,omitempty"`
	After string `json:"after,omitempty"`
}

type scheduleAttemptsInput struct {
	Date string `json:"date,omitempty"`
}

type scheduleAutoInput struct {
	Date     string `json:"date,omitempty"`
	Habits   bool   `json:"habits,omitempty"`
	Meetings bool   `json:"meetings,omitempty"`
}

type scheduleImportInput struct {
	Days              int    `json:"days,omitempty"`
	Type              string `json:"type,omitempty"`
	TaggedOnly        bool   `json:"tagged_only,omitempty"`
	CalendarID        string `json:"calendar_id,omitempty"`
	UseConfigCalendar bool   `json:"use_config_calendar,omitempty"`
}

func registerScheduleTools(srv *mcp.Server, deps ToolDependencies) error {
	app := deps.App

	srv.Tool("schedule.show").
		Description("Get the schedule for a date").
		Handler(func(ctx context.Context, input scheduleShowInput) (*scheduleQueries.ScheduleDTO, error) {
			if app == nil || app.GetScheduleHandler == nil {
				return nil, errors.New("schedule requires database connection")
			}
			date, err := parseDate(input.Date, time.Now())
			if err != nil {
				return nil, err
			}

			return app.GetScheduleHandler.Handle(ctx, scheduleQueries.GetScheduleQuery{
				UserID: app.CurrentUserID,
				Date:   date,
			})
		})

	srv.Tool("schedule.week").
		Description("Get schedule for a week").
		Handler(func(ctx context.Context, input scheduleWeekInput) (map[string]any, error) {
			if app == nil || app.GetScheduleHandler == nil {
				return nil, errors.New("schedule requires database connection")
			}
			offset := input.Offset
			if input.Next {
				offset = 1
			}

			now := time.Now()
			weekStart := getWeekStart(now).AddDate(0, 0, offset*7)
			weekEnd := weekStart.AddDate(0, 0, 6)

			days := make([]map[string]any, 0, 7)
			totalBlocks := 0
			totalMinutes := 0
			completedBlocks := 0

			for i := 0; i < 7; i++ {
				day := weekStart.AddDate(0, 0, i)
				schedule, err := app.GetScheduleHandler.Handle(ctx, scheduleQueries.GetScheduleQuery{
					UserID: app.CurrentUserID,
					Date:   day,
				})
				if err != nil {
					continue
				}
				if schedule != nil {
					totalBlocks += len(schedule.Blocks)
					totalMinutes += schedule.TotalScheduledMins
					completedBlocks += schedule.CompletedCount
				}
				days = append(days, map[string]any{
					"date":     day,
					"schedule": schedule,
				})
			}

			return map[string]any{
				"week_start": weekStart,
				"week_end":   weekEnd,
				"days":       days,
				"summary": map[string]any{
					"total_blocks":  totalBlocks,
					"total_minutes": totalMinutes,
					"completed":     completedBlocks,
				},
			}, nil
		})

	srv.Tool("schedule.available").
		Description("Find available time slots").
		Handler(func(ctx context.Context, input scheduleAvailableInput) ([]scheduleQueries.TimeSlotDTO, error) {
			if app == nil || app.FindAvailableSlotsHandler == nil {
				return nil, errors.New("schedule requires database connection")
			}
			if input.Min <= 0 {
				input.Min = 15
			}
			if input.Start == "" {
				input.Start = "08:00"
			}
			if input.End == "" {
				input.End = "18:00"
			}

			date, err := parseDate(input.Date, time.Now())
			if err != nil {
				return nil, err
			}
			dayStart, err := parseTimeOnDate(date, input.Start)
			if err != nil {
				return nil, err
			}
			dayEnd, err := parseTimeOnDate(date, input.End)
			if err != nil {
				return nil, err
			}

			return app.FindAvailableSlotsHandler.Handle(ctx, scheduleQueries.FindAvailableSlotsQuery{
				UserID:      app.CurrentUserID,
				Date:        date,
				DayStart:    dayStart,
				DayEnd:      dayEnd,
				MinDuration: time.Duration(input.Min) * time.Minute,
			})
		})

	srv.Tool("schedule.add").
		Description("Add a time block to schedule").
		Handler(func(ctx context.Context, input scheduleAddInput) (*scheduleCommands.AddBlockResult, error) {
			if app == nil || app.AddBlockHandler == nil {
				return nil, errors.New("schedule requires database connection")
			}
			blockType := domain.BlockType(strings.ToLower(input.Type))
			if !isValidBlockType(blockType) {
				return nil, fmt.Errorf("invalid block type: %s", input.Type)
			}

			date, err := parseDate(input.Date, time.Now())
			if err != nil {
				return nil, err
			}
			startTime, err := parseTimeOnDate(date, input.Start)
			if err != nil {
				return nil, err
			}
			endTime, err := parseTimeOnDate(date, input.End)
			if err != nil {
				return nil, err
			}
			refID, err := parseOptionalUUID(input.Ref)
			if err != nil {
				return nil, err
			}

			return app.AddBlockHandler.Handle(ctx, scheduleCommands.AddBlockCommand{
				UserID:      app.CurrentUserID,
				Date:        date,
				BlockType:   string(blockType),
				ReferenceID: refID,
				Title:       input.Title,
				StartTime:   startTime,
				EndTime:     endTime,
			})
		})

	srv.Tool("schedule.complete").
		Description("Mark a schedule block as completed").
		Handler(func(ctx context.Context, input scheduleCompleteInput) (map[string]any, error) {
			if app == nil || app.CompleteBlockHandler == nil {
				return nil, errors.New("schedule requires database connection")
			}
			scheduleID, err := parseUUID(input.ScheduleID)
			if err != nil {
				return nil, err
			}
			blockID, err := parseUUID(input.BlockID)
			if err != nil {
				return nil, err
			}

			if err := app.CompleteBlockHandler.Handle(ctx, scheduleCommands.CompleteBlockCommand{
				ScheduleID: scheduleID,
				BlockID:    blockID,
			}); err != nil {
				return nil, err
			}
			return map[string]any{"schedule_id": scheduleID, "block_id": blockID, "completed": true}, nil
		})

	srv.Tool("schedule.remove").
		Description("Remove a time block").
		Handler(func(ctx context.Context, input scheduleRemoveInput) (map[string]any, error) {
			if app == nil || app.RemoveBlockHandler == nil {
				return nil, errors.New("schedule requires database connection")
			}
			blockID, err := parseUUID(input.BlockID)
			if err != nil {
				return nil, err
			}
			date, err := parseDate(input.Date, time.Now())
			if err != nil {
				return nil, err
			}

			if err := app.RemoveBlockHandler.Handle(ctx, scheduleCommands.RemoveBlockCommand{
				UserID:  app.CurrentUserID,
				BlockID: blockID,
				Date:    date,
			}); err != nil {
				return nil, err
			}

			warnings := []string{}
			if app.SettingsService != nil && app.CalendarSyncer != nil {
				if deleteMissing, err := app.SettingsService.GetDeleteMissing(ctx, app.CurrentUserID); err == nil && deleteMissing {
					if googleSyncer, ok := app.CalendarSyncer.(*googleCalendar.Syncer); ok {
						if err := googleSyncer.DeleteEvent(ctx, app.CurrentUserID, blockID); err != nil {
							warnings = append(warnings, fmt.Sprintf("calendar delete failed: %v", err))
						}
					}
				}
			}

			return map[string]any{"block_id": blockID, "removed": true, "warnings": warnings}, nil
		})

	srv.Tool("schedule.reschedule").
		Description("Reschedule a time block").
		Handler(func(ctx context.Context, input scheduleRescheduleInput) (map[string]any, error) {
			if app == nil || app.RescheduleBlockHandler == nil {
				return nil, errors.New("schedule requires database connection")
			}
			blockID, err := parseUUID(input.BlockID)
			if err != nil {
				return nil, err
			}
			date, err := parseDate(input.Date, time.Now())
			if err != nil {
				return nil, err
			}
			startTime, err := parseTimeOnDate(date, input.Start)
			if err != nil {
				return nil, err
			}
			endTime, err := parseTimeOnDate(date, input.End)
			if err != nil {
				return nil, err
			}

			if err := app.RescheduleBlockHandler.Handle(ctx, scheduleCommands.RescheduleBlockCommand{
				UserID:   app.CurrentUserID,
				BlockID:  blockID,
				Date:     date,
				NewStart: startTime,
				NewEnd:   endTime,
			}); err != nil {
				return nil, err
			}
			return map[string]any{"block_id": blockID, "rescheduled": true}, nil
		})

	srv.Tool("schedule.reschedule_missed").
		Description("Auto-reschedule missed blocks").
		Handler(func(ctx context.Context, input scheduleRescheduleMissedInput) (*scheduleCommands.AutoRescheduleResult, error) {
			if app == nil || app.AutoRescheduleHandler == nil {
				return nil, errors.New("schedule requires database connection")
			}
			if err := cli.RequireEntitlement(ctx, app, billingDomain.ModuleAutoRescheduler); err != nil {
				return nil, err
			}
			date, err := parseDate(input.Date, time.Now())
			if err != nil {
				return nil, err
			}
			after, err := parseOptionalTime(date, input.After)
			if err != nil {
				return nil, err
			}

			return app.AutoRescheduleHandler.Handle(ctx, scheduleCommands.AutoRescheduleCommand{
				UserID: app.CurrentUserID,
				Date:   date,
				After:  after,
			})
		})

	srv.Tool("schedule.reschedule_attempts").
		Description("List reschedule attempts for a date").
		Handler(func(ctx context.Context, input scheduleAttemptsInput) ([]scheduleQueries.RescheduleAttemptDTO, error) {
			if app == nil || app.ListRescheduleAttemptsHandler == nil {
				return nil, errors.New("schedule requires database connection")
			}
			date, err := parseDate(input.Date, time.Now())
			if err != nil {
				return nil, err
			}

			return app.ListRescheduleAttemptsHandler.Handle(ctx, scheduleQueries.ListRescheduleAttemptsQuery{
				UserID: app.CurrentUserID,
				Date:   date,
			})
		})

	srv.Tool("schedule.auto").
		Description("Auto-schedule pending tasks, habits, and meetings").
		Handler(func(ctx context.Context, input scheduleAutoInput) (*scheduleCommands.AutoScheduleResult, error) {
			if app == nil || app.AutoScheduleHandler == nil {
				return nil, errors.New("schedule requires database connection")
			}
			date, err := parseDate(input.Date, time.Now())
			if err != nil {
				return nil, err
			}

			items := buildSchedulableItems(ctx, app, date, input.Habits, input.Meetings)
			if len(items) == 0 {
				return &scheduleCommands.AutoScheduleResult{
					ScheduledCount: 0,
					FailedCount:    0,
				}, nil
			}

			return app.AutoScheduleHandler.Handle(ctx, scheduleCommands.AutoScheduleCommand{
				UserID: app.CurrentUserID,
				Date:   date,
				Tasks:  items,
			})
		})

	srv.Tool("schedule.import").
		Description("Import calendar events into schedule").
		Handler(func(ctx context.Context, input scheduleImportInput) (map[string]any, error) {
			if app == nil || app.AddBlockHandler == nil {
				return nil, errors.New("schedule requires database connection")
			}
			if app.CalendarSyncer == nil {
				return nil, errors.New("calendar sync not configured")
			}

			if input.Days <= 0 {
				input.Days = 7
			}
			if input.Type == "" {
				input.Type = "focus"
			}

			googleSyncer, ok := app.CalendarSyncer.(*googleCalendar.Syncer)
			if !ok {
				return nil, errors.New("calendar import only supported for Google Calendar")
			}

			blockType := domain.BlockType(strings.ToLower(input.Type))
			if !isValidBlockType(blockType) {
				return nil, fmt.Errorf("invalid block type: %s", input.Type)
			}

			if input.UseConfigCalendar && app.SettingsService != nil && input.CalendarID == "" {
				if storedID, err := app.SettingsService.GetCalendarID(ctx, app.CurrentUserID); err == nil && storedID != "" {
					input.CalendarID = storedID
				}
			}

			if input.CalendarID != "" {
				googleSyncer = googleSyncer.WithCalendarID(input.CalendarID)
			} else if !input.UseConfigCalendar {
				googleSyncer = googleSyncer.WithCalendarID("primary")
			}

			start := time.Now()
			end := start.AddDate(0, 0, input.Days)
			events, err := googleSyncer.ListEvents(ctx, app.CurrentUserID, start, end, input.TaggedOnly)
			if err != nil {
				return nil, err
			}

			created := 0
			failed := 0
			for _, event := range events {
				title := strings.TrimSpace(event.Summary)
				if title == "" {
					title = "Imported event"
				}
				startTime := event.Start.In(time.Local)
				endTime := event.End.In(time.Local)
				if !endTime.After(startTime) {
					failed++
					continue
				}

				if _, err := app.AddBlockHandler.Handle(ctx, scheduleCommands.AddBlockCommand{
					UserID:    app.CurrentUserID,
					Date:      startTime,
					BlockType: string(blockType),
					Title:     title,
					StartTime: startTime,
					EndTime:   endTime,
				}); err != nil {
					failed++
					continue
				}
				created++
			}

			return map[string]any{"created": created, "failed": failed}, nil
		})

	return nil
}

func isValidBlockType(blockType domain.BlockType) bool {
	valid := []domain.BlockType{
		domain.BlockTypeTask,
		domain.BlockTypeHabit,
		domain.BlockTypeMeeting,
		domain.BlockTypeFocus,
		domain.BlockTypeBreak,
	}
	for _, t := range valid {
		if blockType == t {
			return true
		}
	}
	return false
}

func getWeekStart(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	daysToMonday := weekday - 1
	return time.Date(t.Year(), t.Month(), t.Day()-daysToMonday, 0, 0, 0, 0, t.Location())
}

func buildSchedulableItems(ctx context.Context, app *cli.App, date time.Time, includeHabits, includeMeetings bool) []scheduleCommands.SchedulableItem {
	items := make([]scheduleCommands.SchedulableItem, 0)

	if app.ListTasksHandler != nil {
		tasks, err := app.ListTasksHandler.Handle(ctx, queries.ListTasksQuery{
			UserID: app.CurrentUserID,
			Status: "pending",
		})
		if err == nil {
			for _, task := range tasks {
				priority := 3
				switch task.Priority {
				case "urgent":
					priority = 1
				case "high":
					priority = 2
				case "medium":
					priority = 3
				case "low":
					priority = 4
				case "none":
					priority = 5
				}

				duration := time.Duration(task.DurationMinutes) * time.Minute
				if duration == 0 {
					duration = 30 * time.Minute
				}

				items = append(items, scheduleCommands.SchedulableItem{
					ID:       task.ID,
					Type:     "task",
					Title:    task.Title,
					Priority: priority,
					Duration: duration,
					DueDate:  task.DueDate,
				})
			}
		}
	}

	if includeHabits && app.ListHabitsHandler != nil {
		habits, err := app.ListHabitsHandler.Handle(ctx, habitQueries.ListHabitsQuery{
			UserID:       app.CurrentUserID,
			OnlyDueToday: true,
		})
		if err == nil {
			for _, habit := range habits {
				if habit.CompletedToday {
					continue
				}
				priority := 2
				switch habit.PreferredTime {
				case "morning":
					priority = 1
				case "evening":
					priority = 4
				}
				duration := time.Duration(habit.DurationMins) * time.Minute
				if duration == 0 {
					duration = 30 * time.Minute
				}
				items = append(items, scheduleCommands.SchedulableItem{
					ID:       habit.ID,
					Type:     "habit",
					Title:    habit.Name,
					Priority: priority,
					Duration: duration,
				})
			}
		}
	}

	if includeMeetings && app.ListMeetingCandidatesHandler != nil {
		meetings, err := app.ListMeetingCandidatesHandler.Handle(ctx, meetingQueries.ListMeetingCandidatesQuery{
			UserID: app.CurrentUserID,
			Date:   date,
		})
		if err == nil {
			for _, meeting := range meetings {
				priority := priorityForMeetingTime(meeting.PreferredTime)
				duration := time.Duration(meeting.DurationMins) * time.Minute
				if duration == 0 {
					duration = 30 * time.Minute
				}
				dueAt := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location()).Add(meeting.PreferredTime)
				items = append(items, scheduleCommands.SchedulableItem{
					ID:       meeting.ID,
					Type:     "meeting",
					Title:    meeting.Name,
					Priority: priority,
					Duration: duration,
					DueDate:  &dueAt,
				})
			}
		}
	}

	return items
}

func priorityForMeetingTime(preferred time.Duration) int {
	hour := int(preferred.Hours())
	if hour < 12 {
		return 1
	}
	if hour < 17 {
		return 2
	}
	return 4
}

func runAutoSchedule(ctx context.Context, app *cli.App, date time.Time, tasks []queries.TaskDTO, habits []habitQueries.HabitDTO, meetings []meetingQueries.MeetingCandidateDTO) any {
	items := make([]scheduleCommands.SchedulableItem, 0)

	for _, task := range tasks {
		priority := 3
		switch task.Priority {
		case "urgent":
			priority = 1
		case "high":
			priority = 2
		case "medium":
			priority = 3
		case "low":
			priority = 4
		case "none":
			priority = 5
		}
		duration := time.Duration(task.DurationMinutes) * time.Minute
		if duration == 0 {
			duration = 30 * time.Minute
		}
		items = append(items, scheduleCommands.SchedulableItem{
			ID:       task.ID,
			Type:     "task",
			Title:    task.Title,
			Priority: priority,
			Duration: duration,
			DueDate:  task.DueDate,
		})
	}

	for _, habit := range habits {
		if habit.CompletedToday {
			continue
		}
		priority := 2
		switch habit.PreferredTime {
		case "morning":
			priority = 1
		case "evening":
			priority = 4
		}
		duration := time.Duration(habit.DurationMins) * time.Minute
		if duration == 0 {
			duration = 30 * time.Minute
		}
		items = append(items, scheduleCommands.SchedulableItem{
			ID:       habit.ID,
			Type:     "habit",
			Title:    habit.Name,
			Priority: priority,
			Duration: duration,
		})
	}

	for _, meeting := range meetings {
		priority := priorityForMeetingTime(meeting.PreferredTime)
		duration := time.Duration(meeting.DurationMins) * time.Minute
		if duration == 0 {
			duration = 30 * time.Minute
		}
		dueAt := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location()).Add(meeting.PreferredTime)
		items = append(items, scheduleCommands.SchedulableItem{
			ID:       meeting.ID,
			Type:     "meeting",
			Title:    meeting.Name,
			Priority: priority,
			Duration: duration,
			DueDate:  &dueAt,
		})
	}

	if len(items) == 0 {
		return nil
	}

	if app.AutoScheduleHandler == nil {
		return nil
	}

	result, err := app.AutoScheduleHandler.Handle(ctx, scheduleCommands.AutoScheduleCommand{
		UserID: app.CurrentUserID,
		Date:   date,
		Tasks:  items,
	})
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return result
}
