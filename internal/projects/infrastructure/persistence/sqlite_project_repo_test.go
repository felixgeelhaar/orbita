package persistence

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	db "github.com/felixgeelhaar/orbita/db/generated/sqlite"
	"github.com/felixgeelhaar/orbita/internal/projects/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

// setupProjectTestDB creates an in-memory SQLite database with schema applied.
func setupProjectTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Open in-memory database
	sqlDB, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	// Apply migrations in order
	migrations := []string{
		"000001_initial_schema.up.sql",
		"000004_projects.up.sql",
	}

	for _, migration := range migrations {
		schemaPath := filepath.Join("..", "..", "..", "..", "migrations", "sqlite", migration)
		schema, err := os.ReadFile(schemaPath)
		require.NoError(t, err, "Failed to read SQLite schema file: %s", migration)

		_, err = sqlDB.Exec(string(schema))
		require.NoError(t, err, "Failed to apply SQLite schema: %s", migration)
	}

	return sqlDB
}

// createProjectTestUser creates a user in the database for foreign key constraints.
func createProjectTestUser(t *testing.T, sqlDB *sql.DB, userID uuid.UUID) {
	t.Helper()

	queries := db.New(sqlDB)
	_, err := queries.CreateUser(context.Background(), db.CreateUserParams{
		ID:        userID.String(),
		Email:     "test-" + userID.String()[:8] + "@example.com",
		Name:      "Test User",
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
	})
	require.NoError(t, err)
}

func TestSQLiteProjectRepository_Save_Create(t *testing.T) {
	sqlDB := setupProjectTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createProjectTestUser(t, sqlDB, userID)

	repo := NewSQLiteProjectRepository(sqlDB)
	ctx := context.Background()

	// Create a new project
	project := domain.NewProject(userID, "Test Project")
	project.SetDescription("A test project description")

	// Save it
	err := repo.Save(ctx, project)
	require.NoError(t, err)

	// Verify it was created
	found, err := repo.FindByID(ctx, project.ID(), userID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, project.ID(), found.ID())
	assert.Equal(t, "Test Project", found.Name())
	assert.Equal(t, "A test project description", found.Description())
	assert.Equal(t, userID, found.UserID())
	assert.Equal(t, domain.StatusPlanning, found.Status())
}

func TestSQLiteProjectRepository_Save_Update(t *testing.T) {
	sqlDB := setupProjectTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createProjectTestUser(t, sqlDB, userID)

	repo := NewSQLiteProjectRepository(sqlDB)
	ctx := context.Background()

	// Create and save a project
	project := domain.NewProject(userID, "Original Name")
	err := repo.Save(ctx, project)
	require.NoError(t, err)

	// Reload, modify, and save again
	found, err := repo.FindByID(ctx, project.ID(), userID)
	require.NoError(t, err)

	err = found.SetName("Updated Name")
	require.NoError(t, err)
	found.SetDescription("Updated description")

	err = repo.Save(ctx, found)
	require.NoError(t, err)

	// Verify the update
	updated, err := repo.FindByID(ctx, project.ID(), userID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.Name())
	assert.Equal(t, "Updated description", updated.Description())
}

func TestSQLiteProjectRepository_FindByID_NotFound(t *testing.T) {
	sqlDB := setupProjectTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createProjectTestUser(t, sqlDB, userID)

	repo := NewSQLiteProjectRepository(sqlDB)
	ctx := context.Background()

	found, err := repo.FindByID(ctx, uuid.New(), userID)
	assert.Error(t, err)
	assert.Nil(t, found)
	assert.ErrorIs(t, err, domain.ErrProjectNotFound)
}

func TestSQLiteProjectRepository_FindByUser(t *testing.T) {
	sqlDB := setupProjectTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createProjectTestUser(t, sqlDB, userID)

	otherUserID := uuid.New()
	createProjectTestUser(t, sqlDB, otherUserID)

	repo := NewSQLiteProjectRepository(sqlDB)
	ctx := context.Background()

	// Create projects for the user
	project1 := domain.NewProject(userID, "Project 1")
	project2 := domain.NewProject(userID, "Project 2")
	project3 := domain.NewProject(otherUserID, "Other User Project")

	require.NoError(t, repo.Save(ctx, project1))
	require.NoError(t, repo.Save(ctx, project2))
	require.NoError(t, repo.Save(ctx, project3))

	// Find projects for the user
	projects, err := repo.FindByUser(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, projects, 2)

	// Verify we got the right projects
	projectIDs := make(map[uuid.UUID]bool)
	for _, p := range projects {
		projectIDs[p.ID()] = true
	}
	assert.True(t, projectIDs[project1.ID()])
	assert.True(t, projectIDs[project2.ID()])
	assert.False(t, projectIDs[project3.ID()])
}

func TestSQLiteProjectRepository_FindByStatus(t *testing.T) {
	sqlDB := setupProjectTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createProjectTestUser(t, sqlDB, userID)

	repo := NewSQLiteProjectRepository(sqlDB)
	ctx := context.Background()

	// Create projects with different statuses
	planningProject := domain.NewProject(userID, "Planning Project")

	activeProject := domain.NewProject(userID, "Active Project")
	require.NoError(t, activeProject.Start())

	completedProject := domain.NewProject(userID, "Completed Project")
	require.NoError(t, completedProject.Start())
	require.NoError(t, completedProject.Complete())

	require.NoError(t, repo.Save(ctx, planningProject))
	require.NoError(t, repo.Save(ctx, activeProject))
	require.NoError(t, repo.Save(ctx, completedProject))

	// Find by status
	activeProjects, err := repo.FindByStatus(ctx, userID, domain.StatusActive)
	require.NoError(t, err)
	assert.Len(t, activeProjects, 1)
	assert.Equal(t, "Active Project", activeProjects[0].Name())

	planningProjects, err := repo.FindByStatus(ctx, userID, domain.StatusPlanning)
	require.NoError(t, err)
	assert.Len(t, planningProjects, 1)
	assert.Equal(t, "Planning Project", planningProjects[0].Name())
}

func TestSQLiteProjectRepository_FindActive(t *testing.T) {
	sqlDB := setupProjectTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createProjectTestUser(t, sqlDB, userID)

	repo := NewSQLiteProjectRepository(sqlDB)
	ctx := context.Background()

	// Create projects with various statuses
	planningProject := domain.NewProject(userID, "Planning Project")

	activeProject := domain.NewProject(userID, "Active Project")
	require.NoError(t, activeProject.Start())

	onHoldProject := domain.NewProject(userID, "On Hold Project")
	require.NoError(t, onHoldProject.Start())
	require.NoError(t, onHoldProject.PutOnHold())

	completedProject := domain.NewProject(userID, "Completed Project")
	require.NoError(t, completedProject.Start())
	require.NoError(t, completedProject.Complete())

	archivedProject := domain.NewProject(userID, "Archived Project")
	require.NoError(t, archivedProject.Archive())

	require.NoError(t, repo.Save(ctx, planningProject))
	require.NoError(t, repo.Save(ctx, activeProject))
	require.NoError(t, repo.Save(ctx, onHoldProject))
	require.NoError(t, repo.Save(ctx, completedProject))
	require.NoError(t, repo.Save(ctx, archivedProject))

	// Find active projects (should include planning, active, on_hold)
	activeProjects, err := repo.FindActive(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, activeProjects, 3)

	// Verify statuses
	for _, p := range activeProjects {
		status := p.Status()
		assert.True(t, status == domain.StatusPlanning || status == domain.StatusActive || status == domain.StatusOnHold,
			"Expected planning, active or on_hold, got %s", status)
	}
}

func TestSQLiteProjectRepository_Delete(t *testing.T) {
	sqlDB := setupProjectTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createProjectTestUser(t, sqlDB, userID)

	repo := NewSQLiteProjectRepository(sqlDB)
	ctx := context.Background()

	// Create and save a project
	project := domain.NewProject(userID, "Project to Delete")
	require.NoError(t, repo.Save(ctx, project))

	// Verify it exists
	found, err := repo.FindByID(ctx, project.ID(), userID)
	require.NoError(t, err)
	require.NotNil(t, found)

	// Delete it
	err = repo.Delete(ctx, project.ID(), userID)
	require.NoError(t, err)

	// Verify it's gone
	found, err = repo.FindByID(ctx, project.ID(), userID)
	assert.Error(t, err)
	assert.Nil(t, found)
}

func TestSQLiteProjectRepository_WithDates(t *testing.T) {
	sqlDB := setupProjectTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createProjectTestUser(t, sqlDB, userID)

	repo := NewSQLiteProjectRepository(sqlDB)
	ctx := context.Background()

	// Create project with dates
	project := domain.NewProject(userID, "Project with Dates")

	startDate := time.Now().Truncate(time.Second)
	project.SetStartDate(&startDate)

	dueDate := time.Now().Add(30 * 24 * time.Hour).Truncate(time.Second)
	require.NoError(t, project.SetDueDate(&dueDate))

	err := repo.Save(ctx, project)
	require.NoError(t, err)

	// Verify dates are persisted correctly
	found, err := repo.FindByID(ctx, project.ID(), userID)
	require.NoError(t, err)
	require.NotNil(t, found.StartDate())
	require.NotNil(t, found.DueDate())

	// Compare unix timestamps to avoid nanosecond differences
	assert.Equal(t, startDate.Unix(), found.StartDate().Unix())
	assert.Equal(t, dueDate.Unix(), found.DueDate().Unix())
}

func TestSQLiteProjectRepository_WithTaskLinks(t *testing.T) {
	sqlDB := setupProjectTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createProjectTestUser(t, sqlDB, userID)

	repo := NewSQLiteProjectRepository(sqlDB)
	ctx := context.Background()

	// Create project with task links
	project := domain.NewProject(userID, "Project with Tasks")

	taskID1 := uuid.New()
	taskID2 := uuid.New()
	require.NoError(t, project.AddTask(taskID1, domain.RoleBlocker))
	require.NoError(t, project.AddTask(taskID2, domain.RoleDeliverable))

	err := repo.Save(ctx, project)
	require.NoError(t, err)

	// Verify task links are persisted
	found, err := repo.FindByID(ctx, project.ID(), userID)
	require.NoError(t, err)
	assert.Len(t, found.Tasks(), 2)

	// Verify task link details
	taskMap := make(map[uuid.UUID]domain.TaskLink)
	for _, link := range found.Tasks() {
		taskMap[link.TaskID] = link
	}

	assert.Equal(t, domain.RoleBlocker, taskMap[taskID1].Role)
	assert.Equal(t, domain.RoleDeliverable, taskMap[taskID2].Role)
}

func TestSQLiteProjectRepository_Milestone_Save_Create(t *testing.T) {
	sqlDB := setupProjectTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createProjectTestUser(t, sqlDB, userID)

	repo := NewSQLiteProjectRepository(sqlDB)
	ctx := context.Background()

	// Create project with milestone
	project := domain.NewProject(userID, "Project with Milestone")
	dueDate := time.Now().Add(7 * 24 * time.Hour).Truncate(time.Second)
	milestone := project.AddMilestone("First Milestone", dueDate)
	milestone.SetDescription("Milestone description")

	err := repo.Save(ctx, project)
	require.NoError(t, err)

	// Verify milestone was saved
	foundMilestone, err := repo.FindMilestoneByID(ctx, milestone.ID())
	require.NoError(t, err)
	require.NotNil(t, foundMilestone)
	assert.Equal(t, "First Milestone", foundMilestone.Name())
	assert.Equal(t, "Milestone description", foundMilestone.Description())
	assert.Equal(t, project.ID(), foundMilestone.ProjectID())
}

func TestSQLiteProjectRepository_Milestone_Update(t *testing.T) {
	sqlDB := setupProjectTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createProjectTestUser(t, sqlDB, userID)

	repo := NewSQLiteProjectRepository(sqlDB)
	ctx := context.Background()

	// Create and save project with milestone
	project := domain.NewProject(userID, "Project")
	dueDate := time.Now().Add(7 * 24 * time.Hour)
	milestone := project.AddMilestone("Original Name", dueDate)

	err := repo.Save(ctx, project)
	require.NoError(t, err)

	// Modify and save milestone
	milestone.SetName("Updated Name")
	milestone.SetDescription("Added description")

	err = repo.SaveMilestone(ctx, milestone)
	require.NoError(t, err)

	// Verify update
	found, err := repo.FindMilestoneByID(ctx, milestone.ID())
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", found.Name())
	assert.Equal(t, "Added description", found.Description())
}

func TestSQLiteProjectRepository_Milestone_FindByID_NotFound(t *testing.T) {
	sqlDB := setupProjectTestDB(t)
	defer sqlDB.Close()

	repo := NewSQLiteProjectRepository(sqlDB)
	ctx := context.Background()

	found, err := repo.FindMilestoneByID(ctx, uuid.New())
	assert.Error(t, err)
	assert.Nil(t, found)
	assert.ErrorIs(t, err, domain.ErrMilestoneNotFound)
}

func TestSQLiteProjectRepository_Milestone_FindByProject(t *testing.T) {
	sqlDB := setupProjectTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createProjectTestUser(t, sqlDB, userID)

	repo := NewSQLiteProjectRepository(sqlDB)
	ctx := context.Background()

	// Create project with multiple milestones
	project := domain.NewProject(userID, "Project with Milestones")
	dueDate1 := time.Now().Add(7 * 24 * time.Hour)
	dueDate2 := time.Now().Add(14 * 24 * time.Hour)
	dueDate3 := time.Now().Add(21 * 24 * time.Hour)

	milestone1 := project.AddMilestone("Milestone 1", dueDate1)
	milestone2 := project.AddMilestone("Milestone 2", dueDate2)
	milestone3 := project.AddMilestone("Milestone 3", dueDate3)

	err := repo.Save(ctx, project)
	require.NoError(t, err)

	// Find milestones by project
	milestones, err := repo.FindMilestonesByProject(ctx, project.ID())
	require.NoError(t, err)
	assert.Len(t, milestones, 3)

	// Verify IDs
	milestoneIDs := make(map[uuid.UUID]bool)
	for _, m := range milestones {
		milestoneIDs[m.ID()] = true
	}
	assert.True(t, milestoneIDs[milestone1.ID()])
	assert.True(t, milestoneIDs[milestone2.ID()])
	assert.True(t, milestoneIDs[milestone3.ID()])
}

func TestSQLiteProjectRepository_Milestone_Delete(t *testing.T) {
	sqlDB := setupProjectTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createProjectTestUser(t, sqlDB, userID)

	repo := NewSQLiteProjectRepository(sqlDB)
	ctx := context.Background()

	// Create project with milestone
	project := domain.NewProject(userID, "Project")
	dueDate := time.Now().Add(7 * 24 * time.Hour)
	milestone := project.AddMilestone("Milestone to Delete", dueDate)

	err := repo.Save(ctx, project)
	require.NoError(t, err)

	// Verify milestone exists
	found, err := repo.FindMilestoneByID(ctx, milestone.ID())
	require.NoError(t, err)
	require.NotNil(t, found)

	// Delete it
	err = repo.DeleteMilestone(ctx, milestone.ID())
	require.NoError(t, err)

	// Verify it's gone
	found, err = repo.FindMilestoneByID(ctx, milestone.ID())
	assert.Error(t, err)
	assert.Nil(t, found)
}

func TestSQLiteProjectRepository_Milestone_WithTaskLinks(t *testing.T) {
	sqlDB := setupProjectTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createProjectTestUser(t, sqlDB, userID)

	repo := NewSQLiteProjectRepository(sqlDB)
	ctx := context.Background()

	// Create project with milestone that has tasks
	project := domain.NewProject(userID, "Project")
	dueDate := time.Now().Add(7 * 24 * time.Hour)
	milestone := project.AddMilestone("Milestone with Tasks", dueDate)

	taskID1 := uuid.New()
	taskID2 := uuid.New()
	milestone.AddTask(taskID1, domain.RoleBlocker)
	milestone.AddTask(taskID2, domain.RoleSubtask)

	err := repo.Save(ctx, project)
	require.NoError(t, err)

	// Verify task links are persisted
	found, err := repo.FindMilestoneByID(ctx, milestone.ID())
	require.NoError(t, err)
	assert.Len(t, found.Tasks(), 2)

	// Verify task link details
	taskMap := make(map[uuid.UUID]domain.TaskLink)
	for _, link := range found.Tasks() {
		taskMap[link.TaskID] = link
	}

	assert.Equal(t, domain.RoleBlocker, taskMap[taskID1].Role)
	assert.Equal(t, domain.RoleSubtask, taskMap[taskID2].Role)
}

func TestSQLiteProjectRepository_FullCRUDCycle(t *testing.T) {
	sqlDB := setupProjectTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createProjectTestUser(t, sqlDB, userID)

	repo := NewSQLiteProjectRepository(sqlDB)
	ctx := context.Background()

	// CREATE project with all features
	project := domain.NewProject(userID, "Full Cycle Project")
	project.SetDescription("A comprehensive project")

	startDate := time.Now().Truncate(time.Second)
	project.SetStartDate(&startDate)

	dueDate := time.Now().Add(60 * 24 * time.Hour).Truncate(time.Second)
	require.NoError(t, project.SetDueDate(&dueDate))

	project.SetMetadata("priority", "high")
	project.SetMetadata("team", "engineering")

	// Add task links
	taskID := uuid.New()
	require.NoError(t, project.AddTask(taskID, domain.RoleDeliverable))

	// Add milestone
	milestoneDueDate := time.Now().Add(30 * 24 * time.Hour)
	milestone := project.AddMilestone("First Release", milestoneDueDate)
	milestone.SetDescription("Initial release milestone")
	milestoneTaskID := uuid.New()
	milestone.AddTask(milestoneTaskID, domain.RoleSubtask)

	err := repo.Save(ctx, project)
	require.NoError(t, err)

	// READ
	found, err := repo.FindByID(ctx, project.ID(), userID)
	require.NoError(t, err)
	assert.Equal(t, "Full Cycle Project", found.Name())
	assert.Equal(t, "A comprehensive project", found.Description())
	assert.Equal(t, domain.StatusPlanning, found.Status())
	assert.Len(t, found.Tasks(), 1)
	assert.Len(t, found.Milestones(), 1)

	// Verify metadata
	priority, ok := found.GetMetadata("priority")
	assert.True(t, ok)
	assert.Equal(t, "high", priority)

	// UPDATE - Start the project
	err = found.Start()
	require.NoError(t, err)
	err = repo.Save(ctx, found)
	require.NoError(t, err)

	// Verify status update
	updated, err := repo.FindByID(ctx, project.ID(), userID)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusActive, updated.Status())

	// UPDATE - Complete the project
	err = updated.Complete()
	require.NoError(t, err)
	err = repo.Save(ctx, updated)
	require.NoError(t, err)

	// Verify completion
	completed, err := repo.FindByID(ctx, project.ID(), userID)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusCompleted, completed.Status())

	// DELETE
	err = repo.Delete(ctx, project.ID(), userID)
	require.NoError(t, err)

	// Verify deletion
	_, err = repo.FindByID(ctx, project.ID(), userID)
	assert.ErrorIs(t, err, domain.ErrProjectNotFound)
}

func TestSQLiteProjectRepository_HealthScore_Persisted(t *testing.T) {
	sqlDB := setupProjectTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createProjectTestUser(t, sqlDB, userID)

	repo := NewSQLiteProjectRepository(sqlDB)
	ctx := context.Background()

	// Create project with health update
	project := domain.NewProject(userID, "Project with Health")

	// Update health with risk factors
	risks := []domain.RiskFactor{
		{
			Type:        domain.RiskOverdueTasks,
			Severity:    domain.SeverityHigh,
			Description: "Project is overdue",
		},
		{
			Type:        domain.RiskBlockedMilestone,
			Severity:    domain.SeverityMedium,
			Description: "Some milestones are blocked",
		},
	}
	project.UpdateHealth(risks)

	err := repo.Save(ctx, project)
	require.NoError(t, err)

	// Verify health is persisted
	found, err := repo.FindByID(ctx, project.ID(), userID)
	require.NoError(t, err)

	health := found.Health()
	assert.Len(t, health.RiskFactors, 2)

	// Verify risk factors
	riskTypes := make(map[domain.RiskType]domain.RiskFactor)
	for _, r := range health.RiskFactors {
		riskTypes[r.Type] = r
	}

	assert.Equal(t, domain.SeverityHigh, riskTypes[domain.RiskOverdueTasks].Severity)
	assert.Equal(t, domain.SeverityMedium, riskTypes[domain.RiskBlockedMilestone].Severity)
}

func TestSQLiteProjectRepository_ProjectWithMultipleMilestones_Ordering(t *testing.T) {
	sqlDB := setupProjectTestDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	createProjectTestUser(t, sqlDB, userID)

	repo := NewSQLiteProjectRepository(sqlDB)
	ctx := context.Background()

	// Create project with multiple milestones (added in order)
	project := domain.NewProject(userID, "Project")
	dueDate1 := time.Now().Add(7 * 24 * time.Hour)
	dueDate2 := time.Now().Add(14 * 24 * time.Hour)
	dueDate3 := time.Now().Add(21 * 24 * time.Hour)

	m1 := project.AddMilestone("First", dueDate1)
	m2 := project.AddMilestone("Second", dueDate2)
	m3 := project.AddMilestone("Third", dueDate3)

	err := repo.Save(ctx, project)
	require.NoError(t, err)

	// Reload and verify order is preserved
	found, err := repo.FindByID(ctx, project.ID(), userID)
	require.NoError(t, err)
	require.Len(t, found.Milestones(), 3)

	// Check orders
	milestoneOrders := make(map[uuid.UUID]int)
	for _, m := range found.Milestones() {
		milestoneOrders[m.ID()] = m.Order()
	}

	assert.Equal(t, 0, milestoneOrders[m1.ID()])
	assert.Equal(t, 1, milestoneOrders[m2.ID()])
	assert.Equal(t, 2, milestoneOrders[m3.ID()])
}
