package cli

import (
	billingApp "github.com/felixgeelhaar/orbita/internal/billing/application"
	calendarApp "github.com/felixgeelhaar/orbita/internal/calendar/application"
	"github.com/felixgeelhaar/orbita/internal/engine/registry"
	"github.com/felixgeelhaar/orbita/internal/engine/runtime"
	habitCommands "github.com/felixgeelhaar/orbita/internal/habits/application/commands"
	habitQueries "github.com/felixgeelhaar/orbita/internal/habits/application/queries"
	identitySettings "github.com/felixgeelhaar/orbita/internal/identity/application/settings"
	inboxCommands "github.com/felixgeelhaar/orbita/internal/inbox/application/commands"
	inboxQueries "github.com/felixgeelhaar/orbita/internal/inbox/application/queries"
	meetingCommands "github.com/felixgeelhaar/orbita/internal/meetings/application/commands"
	meetingQueries "github.com/felixgeelhaar/orbita/internal/meetings/application/queries"
	"github.com/felixgeelhaar/orbita/internal/productivity/application/commands"
	"github.com/felixgeelhaar/orbita/internal/productivity/application/queries"
	scheduleCommands "github.com/felixgeelhaar/orbita/internal/scheduling/application/commands"
	scheduleQueries "github.com/felixgeelhaar/orbita/internal/scheduling/application/queries"
	"github.com/google/uuid"
)

// App holds the CLI application dependencies.
type App struct {
	// Task Command Handlers
	CreateTaskHandler   *commands.CreateTaskHandler
	CompleteTaskHandler *commands.CompleteTaskHandler
	ArchiveTaskHandler  *commands.ArchiveTaskHandler

	// Task Query Handlers
	ListTasksHandler *queries.ListTasksHandler

	// Habit Command Handlers
	CreateHabitHandler          *habitCommands.CreateHabitHandler
	LogCompletionHandler        *habitCommands.LogCompletionHandler
	ArchiveHabitHandler         *habitCommands.ArchiveHabitHandler
	AdjustHabitFrequencyHandler *habitCommands.AdjustHabitFrequencyHandler

	// Habit Query Handlers
	ListHabitsHandler *habitQueries.ListHabitsHandler

	// Meeting Command Handlers
	CreateMeetingHandler        *meetingCommands.CreateMeetingHandler
	UpdateMeetingHandler        *meetingCommands.UpdateMeetingHandler
	ArchiveMeetingHandler       *meetingCommands.ArchiveMeetingHandler
	MarkMeetingHeldHandler      *meetingCommands.MarkMeetingHeldHandler
	AdjustMeetingCadenceHandler *meetingCommands.AdjustMeetingCadenceHandler

	// Meeting Query Handlers
	ListMeetingsHandler          *meetingQueries.ListMeetingsHandler
	ListMeetingCandidatesHandler *meetingQueries.ListMeetingCandidatesHandler

	// Schedule Command Handlers
	AddBlockHandler        *scheduleCommands.AddBlockHandler
	CompleteBlockHandler   *scheduleCommands.CompleteBlockHandler
	RemoveBlockHandler     *scheduleCommands.RemoveBlockHandler
	RescheduleBlockHandler *scheduleCommands.RescheduleBlockHandler
	AutoScheduleHandler    *scheduleCommands.AutoScheduleHandler
	AutoRescheduleHandler  *scheduleCommands.AutoRescheduleHandler

	// Schedule Query Handlers
	GetScheduleHandler            *scheduleQueries.GetScheduleHandler
	FindAvailableSlotsHandler     *scheduleQueries.FindAvailableSlotsHandler
	ListRescheduleAttemptsHandler *scheduleQueries.ListRescheduleAttemptsHandler

	// Inbox Command Handlers
	CaptureInboxItemHandler *inboxCommands.CaptureInboxItemHandler
	PromoteInboxItemHandler *inboxCommands.PromoteInboxItemHandler

	// Inbox Query Handlers
	ListInboxItemsHandler *inboxQueries.ListInboxItemsHandler

	// Calendar Sync
	CalendarSyncer calendarApp.Syncer

	// Settings
	SettingsService *identitySettings.Service
	BillingService  *billingApp.Service

	// Engine SDK
	EngineRegistry *registry.Registry
	EngineExecutor *runtime.Executor

	// Current user (configured per environment)
	CurrentUserID uuid.UUID
}

// NewApp creates a new CLI application with the provided handlers.
func NewApp(
	createTaskHandler *commands.CreateTaskHandler,
	completeTaskHandler *commands.CompleteTaskHandler,
	archiveTaskHandler *commands.ArchiveTaskHandler,
	listTasksHandler *queries.ListTasksHandler,
	createHabitHandler *habitCommands.CreateHabitHandler,
	logCompletionHandler *habitCommands.LogCompletionHandler,
	archiveHabitHandler *habitCommands.ArchiveHabitHandler,
	adjustHabitFrequencyHandler *habitCommands.AdjustHabitFrequencyHandler,
	listHabitsHandler *habitQueries.ListHabitsHandler,
	createMeetingHandler *meetingCommands.CreateMeetingHandler,
	updateMeetingHandler *meetingCommands.UpdateMeetingHandler,
	archiveMeetingHandler *meetingCommands.ArchiveMeetingHandler,
	markMeetingHeldHandler *meetingCommands.MarkMeetingHeldHandler,
	adjustMeetingCadenceHandler *meetingCommands.AdjustMeetingCadenceHandler,
	listMeetingsHandler *meetingQueries.ListMeetingsHandler,
	listMeetingCandidatesHandler *meetingQueries.ListMeetingCandidatesHandler,
	addBlockHandler *scheduleCommands.AddBlockHandler,
	completeBlockHandler *scheduleCommands.CompleteBlockHandler,
	removeBlockHandler *scheduleCommands.RemoveBlockHandler,
	rescheduleBlockHandler *scheduleCommands.RescheduleBlockHandler,
	autoScheduleHandler *scheduleCommands.AutoScheduleHandler,
	autoRescheduleHandler *scheduleCommands.AutoRescheduleHandler,
	getScheduleHandler *scheduleQueries.GetScheduleHandler,
	findAvailableSlotsHandler *scheduleQueries.FindAvailableSlotsHandler,
	listRescheduleAttemptsHandler *scheduleQueries.ListRescheduleAttemptsHandler,
	captureInboxItemHandler *inboxCommands.CaptureInboxItemHandler,
	promoteInboxItemHandler *inboxCommands.PromoteInboxItemHandler,
	listInboxItemsHandler *inboxQueries.ListInboxItemsHandler,
	billingService *billingApp.Service,
) *App {
	return &App{
		CreateTaskHandler:             createTaskHandler,
		CompleteTaskHandler:           completeTaskHandler,
		ArchiveTaskHandler:            archiveTaskHandler,
		ListTasksHandler:              listTasksHandler,
		CreateHabitHandler:            createHabitHandler,
		LogCompletionHandler:          logCompletionHandler,
		ArchiveHabitHandler:           archiveHabitHandler,
		AdjustHabitFrequencyHandler:   adjustHabitFrequencyHandler,
		ListHabitsHandler:             listHabitsHandler,
		CreateMeetingHandler:          createMeetingHandler,
		UpdateMeetingHandler:          updateMeetingHandler,
		ArchiveMeetingHandler:         archiveMeetingHandler,
		MarkMeetingHeldHandler:        markMeetingHeldHandler,
		AdjustMeetingCadenceHandler:   adjustMeetingCadenceHandler,
		ListMeetingsHandler:           listMeetingsHandler,
		ListMeetingCandidatesHandler:  listMeetingCandidatesHandler,
		AddBlockHandler:               addBlockHandler,
		CompleteBlockHandler:          completeBlockHandler,
		RemoveBlockHandler:            removeBlockHandler,
		RescheduleBlockHandler:        rescheduleBlockHandler,
		AutoScheduleHandler:           autoScheduleHandler,
		AutoRescheduleHandler:         autoRescheduleHandler,
		GetScheduleHandler:            getScheduleHandler,
		FindAvailableSlotsHandler:     findAvailableSlotsHandler,
		ListRescheduleAttemptsHandler: listRescheduleAttemptsHandler,
		CaptureInboxItemHandler:       captureInboxItemHandler,
		PromoteInboxItemHandler:       promoteInboxItemHandler,
		ListInboxItemsHandler:         listInboxItemsHandler,
		BillingService:                billingService,
		CurrentUserID:                 uuid.Nil,
	}
}

// SetCurrentUserID updates the current user ID.
func (a *App) SetCurrentUserID(id uuid.UUID) {
	a.CurrentUserID = id
}

// SetCalendarSyncer updates the calendar syncer.
func (a *App) SetCalendarSyncer(syncer calendarApp.Syncer) {
	a.CalendarSyncer = syncer
}

// SetSettingsService updates the settings service.
func (a *App) SetSettingsService(service *identitySettings.Service) {
	a.SettingsService = service
}

// SetBillingService updates the billing service.
func (a *App) SetBillingService(service *billingApp.Service) {
	a.BillingService = service
}

// SetEngineRegistry updates the engine registry.
func (a *App) SetEngineRegistry(reg *registry.Registry) {
	a.EngineRegistry = reg
}

// SetEngineExecutor updates the engine executor.
func (a *App) SetEngineExecutor(exec *runtime.Executor) {
	a.EngineExecutor = exec
}

// app is the global CLI application instance
var app *App

// SetApp sets the global CLI application instance.
func SetApp(a *App) {
	app = a
}

// GetApp returns the global CLI application instance.
func GetApp() *App {
	return app
}
