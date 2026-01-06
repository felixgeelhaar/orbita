package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	habitCommands "github.com/felixgeelhaar/orbita/internal/habits/application/commands"
	"github.com/felixgeelhaar/orbita/internal/inbox/domain"
	meetingCommands "github.com/felixgeelhaar/orbita/internal/meetings/application/commands"
	productivityCommands "github.com/felixgeelhaar/orbita/internal/productivity/application/commands"
	"github.com/google/uuid"
)

// PromoteTarget represents the type to promote an inbox item into.
type PromoteTarget string

const (
	PromoteTargetTask    PromoteTarget = "task"
	PromoteTargetHabit   PromoteTarget = "habit"
	PromoteTargetMeeting PromoteTarget = "meeting"
)

type taskCreator interface {
	Handle(context.Context, productivityCommands.CreateTaskCommand) (*productivityCommands.CreateTaskResult, error)
}

type habitCreator interface {
	Handle(context.Context, habitCommands.CreateHabitCommand) (*habitCommands.CreateHabitResult, error)
}

type meetingCreator interface {
	Handle(context.Context, meetingCommands.CreateMeetingCommand) (*meetingCommands.CreateMeetingResult, error)
}

// ParsePromoteTarget normalizes and validates the provided target value.
func ParsePromoteTarget(raw string) (PromoteTarget, error) {
	switch PromoteTarget(strings.ToLower(raw)) {
	case PromoteTargetTask, PromoteTargetHabit, PromoteTargetMeeting:
		return PromoteTarget(strings.ToLower(raw)), nil
	default:
		return "", fmt.Errorf("unsupported promote target: %s", raw)
	}
}

// PromoteInboxItemCommand contains the arguments for promotion.
type PromoteInboxItemCommand struct {
	UserID      uuid.UUID
	ItemID      uuid.UUID
	Target      PromoteTarget
	TaskArgs    *productivityCommands.CreateTaskCommand
	HabitArgs   *habitCommands.CreateHabitCommand
	MeetingArgs *meetingCommands.CreateMeetingCommand
}

// PromoteInboxItemResult contains the promotion outcome.
type PromoteInboxItemResult struct {
	PromotedID uuid.UUID
	Target     PromoteTarget
}

// PromoteInboxItemHandler wires promotion dependencies.
type PromoteInboxItemHandler struct {
	repo           domain.InboxRepository
	taskHandler    taskCreator
	habitHandler   habitCreator
	meetingHandler meetingCreator
}

// NewPromoteInboxItemHandler builds a handler.
func NewPromoteInboxItemHandler(
	repo domain.InboxRepository,
	taskHandler taskCreator,
	habitHandler habitCreator,
	meetingHandler meetingCreator,
) *PromoteInboxItemHandler {
	return &PromoteInboxItemHandler{
		repo:           repo,
		taskHandler:    taskHandler,
		habitHandler:   habitHandler,
		meetingHandler: meetingHandler,
	}
}

// Handle executes the promotion flow.
func (h *PromoteInboxItemHandler) Handle(ctx context.Context, cmd PromoteInboxItemCommand) (*PromoteInboxItemResult, error) {
	item, err := h.repo.FindByID(ctx, cmd.UserID, cmd.ItemID)
	if err != nil {
		return nil, err
	}

	if item.Promoted {
		return nil, fmt.Errorf("inbox item %s already promoted to %s", item.ID, item.PromotedTo)
	}

	var promotedID uuid.UUID
	switch cmd.Target {
	case PromoteTargetTask:
		if cmd.TaskArgs == nil {
			cmd.TaskArgs = &productivityCommands.CreateTaskCommand{}
		}
		cmd.TaskArgs.UserID = cmd.UserID
		if cmd.TaskArgs.Title == "" {
			cmd.TaskArgs.Title = item.Content
		}
		result, err := h.taskHandler.Handle(ctx, *cmd.TaskArgs)
		if err != nil {
			return nil, err
		}
		promotedID = result.TaskID
	case PromoteTargetHabit:
		if cmd.HabitArgs == nil {
			cmd.HabitArgs = &habitCommands.CreateHabitCommand{}
		}
		cmd.HabitArgs.UserID = cmd.UserID
		if cmd.HabitArgs.Name == "" {
			cmd.HabitArgs.Name = item.Content
		}
		result, err := h.habitHandler.Handle(ctx, *cmd.HabitArgs)
		if err != nil {
			return nil, err
		}
		promotedID = result.HabitID
	case PromoteTargetMeeting:
		if cmd.MeetingArgs == nil {
			cmd.MeetingArgs = &meetingCommands.CreateMeetingCommand{}
		}
		cmd.MeetingArgs.UserID = cmd.UserID
		if cmd.MeetingArgs.Name == "" {
			cmd.MeetingArgs.Name = item.Content
		}
		result, err := h.meetingHandler.Handle(ctx, *cmd.MeetingArgs)
		if err != nil {
			return nil, err
		}
		promotedID = result.MeetingID
	default:
		return nil, fmt.Errorf("unknown promote target: %s", cmd.Target)
	}

	if err := h.repo.MarkPromoted(ctx, item.ID, string(cmd.Target), promotedID, time.Now().UTC()); err != nil {
		return nil, err
	}

	return &PromoteInboxItemResult{
		PromotedID: promotedID,
		Target:     cmd.Target,
	}, nil
}
