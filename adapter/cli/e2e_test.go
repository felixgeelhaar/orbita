package cli

import (
	"context"
	"os"
	"testing"

	internalApp "github.com/felixgeelhaar/orbita/internal/app"
	"github.com/felixgeelhaar/orbita/internal/productivity/application/queries"
	"github.com/felixgeelhaar/orbita/pkg/config"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func setupCLITestDB(t *testing.T) string {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}
	return dbURL
}

func TestCLIAddTaskEndToEnd(t *testing.T) {
	dbURL := setupCLITestDB(t)

	cfg := &config.Config{
		AppEnv:      "development",
		DatabaseURL: dbURL,
		RabbitMQURL: "amqp://invalid",
	}

	ctx := context.Background()
	container, err := internalApp.NewContainer(ctx, cfg, nil)
	require.NoError(t, err)
	defer container.Close()

	_, _ = container.DB.Exec(ctx, "DELETE FROM outbox")
	_, _ = container.DB.Exec(ctx, "DELETE FROM tasks")

	cliApp := NewApp(
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
		container.GetScheduleHandler,
		container.FindAvailableSlotsHandler,
		container.ListRescheduleAttemptsHandler,
		container.CaptureInboxItemHandler,
		container.PromoteInboxItemHandler,
		container.ListInboxItemsHandler,
		container.BillingService,
	)
	userID := uuid.New()
	cliApp.SetCurrentUserID(userID)
	SetApp(cliApp)
	defer SetApp(nil)

	addCmd.SetContext(ctx)
	err = addCmd.RunE(addCmd, []string{"Write integration test"})
	require.NoError(t, err)

	tasks, err := container.ListTasksHandler.Handle(ctx, queries.ListTasksQuery{
		UserID: userID,
	})
	require.NoError(t, err)
	require.NotEmpty(t, tasks)
}
