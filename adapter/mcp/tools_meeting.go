package mcp

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/felixgeelhaar/mcp-go"
	"github.com/felixgeelhaar/orbita/adapter/cli"
	billingDomain "github.com/felixgeelhaar/orbita/internal/billing/domain"
	"github.com/felixgeelhaar/orbita/internal/meetings/application/commands"
	"github.com/felixgeelhaar/orbita/internal/meetings/application/queries"
)

type meetingCreateInput struct {
	Name         string `json:"name" jsonschema:"required"`
	Cadence      string `json:"cadence,omitempty"`
	CadenceDays  int    `json:"cadence_days,omitempty"`
	DurationMins int    `json:"duration_mins,omitempty"`
	Time         string `json:"time,omitempty"`
}

type meetingListInput struct {
	IncludeArchived bool `json:"include_archived,omitempty"`
}

type meetingUpdateInput struct {
	MeetingID    string `json:"meeting_id" jsonschema:"required"`
	Name         string `json:"name,omitempty"`
	Cadence      string `json:"cadence,omitempty"`
	CadenceDays  int    `json:"cadence_days,omitempty"`
	DurationMins int    `json:"duration_mins,omitempty"`
	Time         string `json:"time,omitempty"`
}

type meetingHeldInput struct {
	MeetingID string `json:"meeting_id" jsonschema:"required"`
	Date      string `json:"date,omitempty"`
	Time      string `json:"time,omitempty"`
}

type meetingArchiveInput struct {
	MeetingID string `json:"meeting_id" jsonschema:"required"`
}

type meetingCandidatesInput struct {
	Date string `json:"date,omitempty"`
}

func registerMeetingTools(srv *mcp.Server, deps ToolDependencies) error {
	app := deps.App

	srv.Tool("meeting.create").
		Description("Create a meeting").
		Handler(func(ctx context.Context, input meetingCreateInput) (*commands.CreateMeetingResult, error) {
			if app == nil || app.CreateMeetingHandler == nil {
				return nil, errors.New("meeting creation requires database connection")
			}
			if err := cli.RequireEntitlement(ctx, app, billingDomain.ModuleSmartMeetings); err != nil {
				return nil, err
			}
			if input.Name == "" {
				return nil, errors.New("name is required")
			}
			if input.Cadence == "" {
				input.Cadence = "weekly"
			}
			if input.DurationMins == 0 {
				input.DurationMins = 30
			}
			if input.Time == "" {
				input.Time = "10:00"
			}

			return app.CreateMeetingHandler.Handle(ctx, commands.CreateMeetingCommand{
				UserID:        app.CurrentUserID,
				Name:          input.Name,
				Cadence:       input.Cadence,
				CadenceDays:   input.CadenceDays,
				DurationMins:  input.DurationMins,
				PreferredTime: input.Time,
			})
		})

	srv.Tool("meeting.list").
		Description("List meetings").
		Handler(func(ctx context.Context, input meetingListInput) ([]queries.MeetingDTO, error) {
			if app == nil || app.ListMeetingsHandler == nil {
				return nil, errors.New("meeting listing requires database connection")
			}
			if err := cli.RequireEntitlement(ctx, app, billingDomain.ModuleSmartMeetings); err != nil {
				return nil, err
			}

			return app.ListMeetingsHandler.Handle(ctx, queries.ListMeetingsQuery{
				UserID:          app.CurrentUserID,
				IncludeArchived: input.IncludeArchived,
			})
		})

	srv.Tool("meeting.update").
		Description("Update a meeting").
		Handler(func(ctx context.Context, input meetingUpdateInput) (map[string]any, error) {
			if app == nil || app.UpdateMeetingHandler == nil {
				return nil, errors.New("meeting update requires database connection")
			}
			if err := cli.RequireEntitlement(ctx, app, billingDomain.ModuleSmartMeetings); err != nil {
				return nil, err
			}
			meetingID, err := parseUUID(input.MeetingID)
			if err != nil {
				return nil, err
			}

			if err := app.UpdateMeetingHandler.Handle(ctx, commands.UpdateMeetingCommand{
				UserID:        app.CurrentUserID,
				MeetingID:     meetingID,
				Name:          input.Name,
				Cadence:       input.Cadence,
				CadenceDays:   input.CadenceDays,
				DurationMins:  input.DurationMins,
				PreferredTime: input.Time,
			}); err != nil {
				return nil, err
			}
			return map[string]any{"meeting_id": meetingID, "updated": true}, nil
		})

	srv.Tool("meeting.held").
		Description("Mark a meeting as held").
		Handler(func(ctx context.Context, input meetingHeldInput) (map[string]any, error) {
			if app == nil || app.MarkMeetingHeldHandler == nil {
				return nil, errors.New("meeting held requires database connection")
			}
			if err := cli.RequireEntitlement(ctx, app, billingDomain.ModuleSmartMeetings); err != nil {
				return nil, err
			}
			meetingID, err := parseUUID(input.MeetingID)
			if err != nil {
				return nil, err
			}

			date := time.Now()
			if input.Date != "" {
				date, err = parseDate(input.Date, date)
				if err != nil {
					return nil, err
				}
			}

			var heldAt time.Time
			if input.Time != "" {
				heldAt, err = parseTimeOnDate(date, input.Time)
				if err != nil {
					return nil, err
				}
			} else {
				heldAt = date
			}

			if err := app.MarkMeetingHeldHandler.Handle(ctx, commands.MarkMeetingHeldCommand{
				UserID:    app.CurrentUserID,
				MeetingID: meetingID,
				HeldAt:    heldAt,
			}); err != nil {
				return nil, err
			}
			return map[string]any{"meeting_id": meetingID, "held_at": heldAt}, nil
		})

	srv.Tool("meeting.archive").
		Description("Archive a meeting").
		Handler(func(ctx context.Context, input meetingArchiveInput) (map[string]any, error) {
			if app == nil || app.ArchiveMeetingHandler == nil {
				return nil, errors.New("meeting archive requires database connection")
			}
			if err := cli.RequireEntitlement(ctx, app, billingDomain.ModuleSmartMeetings); err != nil {
				return nil, err
			}
			meetingID, err := parseUUID(input.MeetingID)
			if err != nil {
				return nil, err
			}

			if err := app.ArchiveMeetingHandler.Handle(ctx, commands.ArchiveMeetingCommand{
				UserID:    app.CurrentUserID,
				MeetingID: meetingID,
			}); err != nil {
				return nil, err
			}
			return map[string]any{"meeting_id": meetingID, "archived": true}, nil
		})

	srv.Tool("meeting.adjust_cadence").
		Description("Adjust meeting cadence based on attendance").
		Handler(func(ctx context.Context, input struct{}) (*commands.AdjustMeetingCadenceResult, error) {
			if app == nil || app.AdjustMeetingCadenceHandler == nil {
				return nil, errors.New("meeting cadence adjustment requires database connection")
			}
			if err := cli.RequireEntitlement(ctx, app, billingDomain.ModuleSmartMeetings); err != nil {
				return nil, err
			}
			return app.AdjustMeetingCadenceHandler.Handle(ctx, commands.AdjustMeetingCadenceCommand{
				UserID: app.CurrentUserID,
			})
		})

	srv.Tool("meeting.candidates").
		Description("List meeting scheduling candidates for a date").
		Handler(func(ctx context.Context, input meetingCandidatesInput) ([]queries.MeetingCandidateDTO, error) {
			if app == nil || app.ListMeetingCandidatesHandler == nil {
				return nil, errors.New("meeting candidates require database connection")
			}
			if err := cli.RequireEntitlement(ctx, app, billingDomain.ModuleSmartMeetings); err != nil {
				return nil, err
			}
			date := time.Now()
			if input.Date != "" {
				parsed, err := time.Parse(dateLayout, input.Date)
				if err != nil {
					return nil, fmt.Errorf("invalid date format, use YYYY-MM-DD: %w", err)
				}
				date = parsed
			}

			return app.ListMeetingCandidatesHandler.Handle(ctx, queries.ListMeetingCandidatesQuery{
				UserID: app.CurrentUserID,
				Date:   date,
			})
		})

	return nil
}
