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
	habitCommands "github.com/felixgeelhaar/orbita/internal/habits/application/commands"
	inboxCommands "github.com/felixgeelhaar/orbita/internal/inbox/application/commands"
	"github.com/felixgeelhaar/orbita/internal/inbox/application/queries"
	"github.com/felixgeelhaar/orbita/internal/inbox/domain"
	meetingCommands "github.com/felixgeelhaar/orbita/internal/meetings/application/commands"
	productivityCommands "github.com/felixgeelhaar/orbita/internal/productivity/application/commands"
)

type inboxCaptureInput struct {
	Content  string            `json:"content" jsonschema:"required"`
	Source   string            `json:"source,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Tags     []string          `json:"tags,omitempty"`
}

type inboxListInput struct {
	IncludePromoted bool `json:"include_promoted,omitempty"`
}

type inboxPromoteInput struct {
	ItemID               string `json:"item_id" jsonschema:"required"`
	Target               string `json:"target,omitempty"`
	TaskTitle            string `json:"task_title,omitempty"`
	TaskDescription      string `json:"task_description,omitempty"`
	TaskPriority         string `json:"task_priority,omitempty"`
	TaskDurationMinutes  int    `json:"task_duration,omitempty"`
	TaskDue              string `json:"task_due,omitempty"`
	HabitName            string `json:"habit_name,omitempty"`
	HabitDescription     string `json:"habit_description,omitempty"`
	HabitFrequency       string `json:"habit_frequency,omitempty"`
	HabitTimesPerWeek    int    `json:"habit_times_per_week,omitempty"`
	HabitDuration        int    `json:"habit_duration,omitempty"`
	HabitPreferredTime   string `json:"habit_preferred_time,omitempty"`
	MeetingName          string `json:"meeting_name,omitempty"`
	MeetingCadence       string `json:"meeting_cadence,omitempty"`
	MeetingCadenceDays   int    `json:"meeting_cadence_days,omitempty"`
	MeetingDuration      int    `json:"meeting_duration,omitempty"`
	MeetingPreferredTime string `json:"meeting_preferred_time,omitempty"`
}

func registerInboxTools(srv *mcp.Server, deps ToolDependencies) error {
	if srv == nil {
		return errors.New("server is required")
	}
	app := deps.App

	srv.Tool("inbox.capture").
		Description("Capture an idea into the AI Inbox").
		Handler(func(ctx context.Context, input inboxCaptureInput) (*inboxCommands.CaptureInboxItemResult, error) {
			if app == nil || app.CaptureInboxItemHandler == nil {
				return nil, errors.New("inbox capture requires database connection")
			}
			if err := cli.RequireEntitlement(ctx, app, billingDomain.ModuleAIInbox); err != nil {
				return nil, err
			}
			if strings.TrimSpace(input.Content) == "" {
				return nil, errors.New("content is required")
			}

			metadata := domain.InboxMetadata{}
			for k, v := range input.Metadata {
				metadata[k] = v
			}

			cmd := inboxCommands.CaptureInboxItemCommand{
				UserID:   app.CurrentUserID,
				Content:  strings.TrimSpace(input.Content),
				Source:   input.Source,
				Metadata: metadata,
				Tags:     input.Tags,
			}

			return app.CaptureInboxItemHandler.Handle(ctx, cmd)
		})

	srv.Tool("inbox.list").
		Description("List captured inbox items for the current user").
		Handler(func(ctx context.Context, input inboxListInput) ([]queries.InboxItemDTO, error) {
			if app == nil || app.ListInboxItemsHandler == nil {
				return nil, errors.New("inbox listing requires database connection")
			}
			if err := cli.RequireEntitlement(ctx, app, billingDomain.ModuleAIInbox); err != nil {
				return nil, err
			}
			return app.ListInboxItemsHandler.Handle(ctx, queries.ListInboxItemsQuery{
				UserID:          app.CurrentUserID,
				IncludePromoted: input.IncludePromoted,
			})
		})

	srv.Tool("inbox.promote").
		Description("Promote an inbox item into a task, habit, or meeting").
		Handler(func(ctx context.Context, input inboxPromoteInput) (*inboxCommands.PromoteInboxItemResult, error) {
			if app == nil || app.PromoteInboxItemHandler == nil {
				return nil, errors.New("inbox promotion requires database connection")
			}
			if err := cli.RequireEntitlement(ctx, app, billingDomain.ModuleAIInbox); err != nil {
				return nil, err
			}
			itemID, err := parseUUID(input.ItemID)
			if err != nil {
				return nil, err
			}

			if strings.TrimSpace(input.Target) == "" {
				input.Target = string(inboxCommands.PromoteTargetTask)
			}

			target, err := inboxCommands.ParsePromoteTarget(input.Target)
			if err != nil {
				return nil, err
			}

			promo := inboxCommands.PromoteInboxItemCommand{
				UserID: app.CurrentUserID,
				ItemID: itemID,
				Target: target,
			}

			switch target {
			case inboxCommands.PromoteTargetTask:
				var due *time.Time
				if strings.TrimSpace(input.TaskDue) != "" {
					parsed, err := time.Parse(dateLayout, input.TaskDue)
					if err != nil {
						return nil, fmt.Errorf("invalid task due format, use YYYY-MM-DD: %w", err)
					}
					due = &parsed
				}
				promo.TaskArgs = &productivityCommands.CreateTaskCommand{
					Title:           input.TaskTitle,
					Description:     input.TaskDescription,
					Priority:        input.TaskPriority,
					DurationMinutes: input.TaskDurationMinutes,
					DueDate:         due,
				}
			case inboxCommands.PromoteTargetHabit:
				promo.HabitArgs = &habitCommands.CreateHabitCommand{
					Name:          input.HabitName,
					Description:   input.HabitDescription,
					Frequency:     input.HabitFrequency,
					TimesPerWeek:  input.HabitTimesPerWeek,
					DurationMins:  input.HabitDuration,
					PreferredTime: input.HabitPreferredTime,
				}
			case inboxCommands.PromoteTargetMeeting:
				promo.MeetingArgs = &meetingCommands.CreateMeetingCommand{
					Name:          input.MeetingName,
					Cadence:       input.MeetingCadence,
					CadenceDays:   input.MeetingCadenceDays,
					DurationMins:  input.MeetingDuration,
					PreferredTime: input.MeetingPreferredTime,
				}
			}

			return app.PromoteInboxItemHandler.Handle(ctx, promo)
		})

	return nil
}
