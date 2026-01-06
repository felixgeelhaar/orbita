package mcp

import (
	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/felixgeelhaar/orbita/internal/app"
	"github.com/google/uuid"
)

// NewCLIApp creates a CLI application instance backed by the provided container.
func NewCLIApp(container *app.Container, currentUser uuid.UUID) *cli.App {
	cliApp := cli.NewApp(
		container.CreateTaskHandler,
		container.CompleteTaskHandler,
		container.ArchiveTaskHandler,
		container.ListTasksHandler,
		container.CreateHabitHandler,
		container.LogCompletionHandler,
		container.ArchiveHabitHandler,
		container.AdjustHabitFrequencyHandler,
		container.ListHabitsHandler,
		container.CreateMeetingHandler,
		container.UpdateMeetingHandler,
		container.ArchiveMeetingHandler,
		container.MarkMeetingHeldHandler,
		container.AdjustMeetingCadenceHandler,
		container.ListMeetingsHandler,
		container.ListMeetingCandidatesHandler,
		container.AddBlockHandler,
		container.CompleteBlockHandler,
		container.RemoveBlockHandler,
		container.RescheduleBlockHandler,
		container.AutoScheduleHandler,
		container.AutoRescheduleHandler,
		container.PriorityRecalcHandler,
		container.GetScheduleHandler,
		container.FindAvailableSlotsHandler,
		container.ListRescheduleAttemptsHandler,
		container.BillingService,
	)

	cliApp.SetCurrentUserID(currentUser)

	if container.CalendarSyncer != nil {
		cliApp.SetCalendarSyncer(container.CalendarSyncer)
	}
	if container.SettingsService != nil {
		cliApp.SetSettingsService(container.SettingsService)
	}
	if container.BillingService != nil {
		cliApp.SetBillingService(container.BillingService)
	}

	return cliApp
}
