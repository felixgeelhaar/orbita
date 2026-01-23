package commands

import (
	"context"
	"time"

	"github.com/felixgeelhaar/orbita/internal/meetings/application/services"
	"github.com/felixgeelhaar/orbita/internal/meetings/domain"
	schedulingDomain "github.com/felixgeelhaar/orbita/internal/scheduling/domain"
	sharedApplication "github.com/felixgeelhaar/orbita/internal/shared/application"
	"github.com/felixgeelhaar/orbita/internal/shared/infrastructure/outbox"
	"github.com/google/uuid"
)

// ScheduleMeetingCommand represents a request to schedule a meeting for a specific date.
type ScheduleMeetingCommand struct {
	UserID    uuid.UUID
	MeetingID uuid.UUID
	Date      time.Time
}

// ScheduleMeetingResult contains the result of scheduling a meeting.
type ScheduleMeetingResult struct {
	MeetingID    uuid.UUID
	MeetingName  string
	Scheduled    bool
	StartTime    time.Time
	EndTime      time.Time
	SlotQuality  int
	Reason       string
	Alternatives []AlternativeSlot
}

// AlternativeSlot represents an alternative time slot suggestion.
type AlternativeSlot struct {
	StartTime time.Time
	EndTime   time.Time
	Quality   int
	Reason    string
}

// ScheduleMeetingHandler handles the ScheduleMeetingCommand.
type ScheduleMeetingHandler struct {
	meetingRepo    domain.Repository
	scheduleRepo   schedulingDomain.ScheduleRepository
	slotFinder     *services.OptimalSlotFinder
	outboxRepo     outbox.Repository
	uow            sharedApplication.UnitOfWork
}

// NewScheduleMeetingHandler creates a new handler.
func NewScheduleMeetingHandler(
	meetingRepo domain.Repository,
	scheduleRepo schedulingDomain.ScheduleRepository,
	slotFinder *services.OptimalSlotFinder,
	outboxRepo outbox.Repository,
	uow sharedApplication.UnitOfWork,
) *ScheduleMeetingHandler {
	return &ScheduleMeetingHandler{
		meetingRepo:  meetingRepo,
		scheduleRepo: scheduleRepo,
		slotFinder:   slotFinder,
		outboxRepo:   outboxRepo,
		uow:          uow,
	}
}

// Handle executes the command.
func (h *ScheduleMeetingHandler) Handle(ctx context.Context, cmd ScheduleMeetingCommand) (*ScheduleMeetingResult, error) {
	// Get the meeting
	meeting, err := h.meetingRepo.FindByID(ctx, cmd.MeetingID)
	if err != nil {
		return nil, err
	}
	if meeting == nil {
		return &ScheduleMeetingResult{
			MeetingID: cmd.MeetingID,
			Scheduled: false,
			Reason:    "Meeting not found",
		}, nil
	}

	if meeting.IsArchived() {
		return &ScheduleMeetingResult{
			MeetingID:   cmd.MeetingID,
			MeetingName: meeting.Name(),
			Scheduled:   false,
			Reason:      "Meeting is archived",
		}, nil
	}

	result := &ScheduleMeetingResult{
		MeetingID:   meeting.ID(),
		MeetingName: meeting.Name(),
	}

	// Normalize date
	date := time.Date(cmd.Date.Year(), cmd.Date.Month(), cmd.Date.Day(), 0, 0, 0, 0, cmd.Date.Location())

	// Find optimal slot
	suggestions, err := h.slotFinder.FindMultipleSlots(
		ctx,
		cmd.UserID,
		date,
		meeting.Duration(),
		meeting.PreferredTime(),
		5, // Get up to 5 suggestions
	)
	if err != nil {
		return nil, err
	}

	if len(suggestions) == 0 {
		result.Scheduled = false
		result.Reason = "No available time slots found"
		return result, nil
	}

	// Use the best slot
	bestSlot := suggestions[0]
	slotStart := bestSlot.StartTime
	slotEnd := slotStart.Add(meeting.Duration())

	// Get or create schedule for the day
	scheduleDate := time.Date(slotStart.Year(), slotStart.Month(), slotStart.Day(), 0, 0, 0, 0, slotStart.Location())
	schedule, err := h.scheduleRepo.FindByUserAndDate(ctx, cmd.UserID, scheduleDate)
	if err != nil {
		return nil, err
	}
	if schedule == nil {
		schedule = schedulingDomain.NewSchedule(cmd.UserID, scheduleDate)
	}

	// Add block to schedule
	block, err := schedule.AddBlock(
		schedulingDomain.BlockTypeMeeting,
		meeting.ID(),
		meeting.Name(),
		slotStart,
		slotEnd,
	)
	if err != nil {
		result.Scheduled = false
		result.Reason = err.Error()
		// Add alternatives
		for i, suggestion := range suggestions {
			if i == 0 {
				continue // Skip the failed one
			}
			result.Alternatives = append(result.Alternatives, AlternativeSlot{
				StartTime: suggestion.StartTime,
				EndTime:   suggestion.StartTime.Add(meeting.Duration()),
				Quality:   int(suggestion.Quality),
				Reason:    suggestion.Reason,
			})
		}
		return result, nil
	}

	// Persist within transaction
	err = sharedApplication.WithUnitOfWork(ctx, h.uow, func(ctx context.Context) error {
		if err := h.scheduleRepo.Save(ctx, schedule); err != nil {
			return err
		}

		// Publish scheduled event
		event := schedulingDomain.NewBlockScheduled(schedule.ID(), block)
		msg, err := outbox.NewMessage(event)
		if err != nil {
			return err
		}
		return h.outboxRepo.Save(ctx, msg)
	})
	if err != nil {
		return nil, err
	}

	result.Scheduled = true
	result.StartTime = slotStart
	result.EndTime = slotEnd
	result.SlotQuality = int(bestSlot.Quality)
	result.Reason = bestSlot.Reason

	// Add remaining suggestions as alternatives
	for i, suggestion := range suggestions {
		if i == 0 {
			continue
		}
		result.Alternatives = append(result.Alternatives, AlternativeSlot{
			StartTime: suggestion.StartTime,
			EndTime:   suggestion.StartTime.Add(meeting.Duration()),
			Quality:   int(suggestion.Quality),
			Reason:    suggestion.Reason,
		})
	}

	return result, nil
}

// ScheduleAllDueMeetingsCommand represents a request to schedule all meetings due on a date.
type ScheduleAllDueMeetingsCommand struct {
	UserID uuid.UUID
	Date   time.Time
}

// ScheduleAllDueMeetingsResult contains the results of scheduling multiple meetings.
type ScheduleAllDueMeetingsResult struct {
	Date              time.Time
	MeetingsProcessed int
	MeetingsScheduled int
	MeetingsFailed    int
	Details           []ScheduleMeetingResult
}

// ScheduleAllDueMeetingsHandler handles the ScheduleAllDueMeetingsCommand.
type ScheduleAllDueMeetingsHandler struct {
	meetingRepo      domain.Repository
	scheduleMeeting  *ScheduleMeetingHandler
}

// NewScheduleAllDueMeetingsHandler creates a new handler.
func NewScheduleAllDueMeetingsHandler(
	meetingRepo domain.Repository,
	scheduleMeeting *ScheduleMeetingHandler,
) *ScheduleAllDueMeetingsHandler {
	return &ScheduleAllDueMeetingsHandler{
		meetingRepo:     meetingRepo,
		scheduleMeeting: scheduleMeeting,
	}
}

// Handle schedules all meetings due on the specified date.
func (h *ScheduleAllDueMeetingsHandler) Handle(ctx context.Context, cmd ScheduleAllDueMeetingsCommand) (*ScheduleAllDueMeetingsResult, error) {
	// Normalize date
	date := time.Date(cmd.Date.Year(), cmd.Date.Month(), cmd.Date.Day(), 0, 0, 0, 0, cmd.Date.Location())

	// Get all active meetings
	meetings, err := h.meetingRepo.FindActiveByUserID(ctx, cmd.UserID)
	if err != nil {
		return nil, err
	}

	result := &ScheduleAllDueMeetingsResult{
		Date:    date,
		Details: make([]ScheduleMeetingResult, 0),
	}

	// Filter and schedule meetings due today
	for _, meeting := range meetings {
		if !meeting.IsDueOn(date) {
			continue
		}

		result.MeetingsProcessed++

		scheduleResult, err := h.scheduleMeeting.Handle(ctx, ScheduleMeetingCommand{
			UserID:    cmd.UserID,
			MeetingID: meeting.ID(),
			Date:      date,
		})
		if err != nil {
			result.MeetingsFailed++
			result.Details = append(result.Details, ScheduleMeetingResult{
				MeetingID:   meeting.ID(),
				MeetingName: meeting.Name(),
				Scheduled:   false,
				Reason:      err.Error(),
			})
			continue
		}

		result.Details = append(result.Details, *scheduleResult)
		if scheduleResult.Scheduled {
			result.MeetingsScheduled++
		} else {
			result.MeetingsFailed++
		}
	}

	return result, nil
}
