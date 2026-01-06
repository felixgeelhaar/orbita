package habit

import (
	"context"
	"os"
	"testing"

	"github.com/felixgeelhaar/orbita/adapter/cli"
	internalApp "github.com/felixgeelhaar/orbita/internal/app"
	habitQueries "github.com/felixgeelhaar/orbita/internal/habits/application/queries"
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

func TestCLIHabitEndToEnd(t *testing.T) {
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
	_, _ = container.DB.Exec(ctx, "DELETE FROM habit_completions")
	_, _ = container.DB.Exec(ctx, "DELETE FROM habits")

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
		container.GetScheduleHandler,
		container.FindAvailableSlotsHandler,
		container.ListRescheduleAttemptsHandler,
		container.BillingService,
	)
	userID := uuid.New()
	cliApp.SetCurrentUserID(userID)
	cli.SetApp(cliApp)
	defer cli.SetApp(nil)

	createCmd.SetContext(ctx)
	require.NoError(t, createCmd.Flags().Set("frequency", "daily"))
	require.NoError(t, createCmd.Flags().Set("duration", "15"))
	err = createCmd.RunE(createCmd, []string{"Test Habit"})
	require.NoError(t, err)

	habits, err := container.ListHabitsHandler.Handle(ctx, habitQueries.ListHabitsQuery{UserID: userID})
	require.NoError(t, err)
	require.Len(t, habits, 1)

	logCmd.SetContext(ctx)
	err = logCmd.RunE(logCmd, []string{habits[0].ID.String()})
	require.NoError(t, err)
}
