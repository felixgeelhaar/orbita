package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	db "github.com/felixgeelhaar/orbita/db/generated/sqlite"
	"github.com/felixgeelhaar/orbita/internal/projects/domain"
	sharedPersistence "github.com/felixgeelhaar/orbita/internal/shared/infrastructure/persistence"
	"github.com/google/uuid"
)

// SQLiteProjectRepository implements domain.Repository using SQLite.
type SQLiteProjectRepository struct {
	dbConn *sql.DB
}

// NewSQLiteProjectRepository creates a new SQLite project repository.
func NewSQLiteProjectRepository(dbConn *sql.DB) *SQLiteProjectRepository {
	return &SQLiteProjectRepository{dbConn: dbConn}
}

// getQuerier returns the appropriate querier based on context.
func (r *SQLiteProjectRepository) getQuerier(ctx context.Context) *db.Queries {
	if info, ok := sharedPersistence.SQLiteTxInfoFromContext(ctx); ok {
		return db.New(info.Tx)
	}
	return db.New(r.dbConn)
}

// Save persists a project to the database.
func (r *SQLiteProjectRepository) Save(ctx context.Context, p *domain.Project) error {
	queries := r.getQuerier(ctx)

	riskFactorsJSON, err := json.Marshal(p.Health().RiskFactors)
	if err != nil {
		return fmt.Errorf("failed to marshal risk factors: %w", err)
	}

	metadataJSON, err := json.Marshal(p.Metadata())
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	var description sql.NullString
	if p.Description() != "" {
		description = sql.NullString{String: p.Description(), Valid: true}
	}

	var startDate sql.NullString
	if p.StartDate() != nil {
		startDate = sql.NullString{String: p.StartDate().Format(time.RFC3339), Valid: true}
	}

	var dueDate sql.NullString
	if p.DueDate() != nil {
		dueDate = sql.NullString{String: p.DueDate().Format(time.RFC3339), Valid: true}
	}

	healthOnTrack := int64(0)
	if p.Health().OnTrack {
		healthOnTrack = 1
	}

	// Try to update first
	_, err = queries.UpdateProject(ctx, db.UpdateProjectParams{
		Name:              p.Name(),
		Description:       description,
		Status:            p.Status().String(),
		StartDate:         startDate,
		DueDate:           dueDate,
		HealthOverall:     p.Health().Overall,
		HealthOnTrack:     healthOnTrack,
		HealthRiskFactors: string(riskFactorsJSON),
		HealthLastUpdated: p.Health().LastUpdated.Format(time.RFC3339),
		Metadata:          string(metadataJSON),
		ID:                p.ID().String(),
		UserID:            p.UserID().String(),
	})

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Project doesn't exist, create it
			_, err = queries.CreateProject(ctx, db.CreateProjectParams{
				ID:                p.ID().String(),
				UserID:            p.UserID().String(),
				Name:              p.Name(),
				Description:       description,
				Status:            p.Status().String(),
				StartDate:         startDate,
				DueDate:           dueDate,
				HealthOverall:     p.Health().Overall,
				HealthOnTrack:     healthOnTrack,
				HealthRiskFactors: string(riskFactorsJSON),
				HealthLastUpdated: p.Health().LastUpdated.Format(time.RFC3339),
				Metadata:          string(metadataJSON),
				CreatedAt:         p.CreatedAt().Format(time.RFC3339),
				UpdatedAt:         p.UpdatedAt().Format(time.RFC3339),
			})
			if err != nil {
				return fmt.Errorf("failed to create project: %w", err)
			}
		} else {
			return fmt.Errorf("failed to update project: %w", err)
		}
	}

	// Save task links
	if err := r.saveTaskLinks(ctx, queries, p); err != nil {
		return fmt.Errorf("failed to save task links: %w", err)
	}

	// Save milestones
	for _, m := range p.Milestones() {
		if err := r.SaveMilestone(ctx, m); err != nil {
			return fmt.Errorf("failed to save milestone: %w", err)
		}
	}

	return nil
}

// saveTaskLinks saves task links for a project.
func (r *SQLiteProjectRepository) saveTaskLinks(ctx context.Context, queries *db.Queries, p *domain.Project) error {
	// Delete existing links and recreate
	if err := queries.DeleteAllProjectTaskLinks(ctx, p.ID().String()); err != nil {
		return err
	}

	for _, link := range p.Tasks() {
		err := queries.CreateProjectTaskLink(ctx, db.CreateProjectTaskLinkParams{
			ProjectID:    p.ID().String(),
			TaskID:       link.TaskID.String(),
			Role:         link.Role.String(),
			DisplayOrder: int64(link.Order),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// FindByID retrieves a project by ID.
func (r *SQLiteProjectRepository) FindByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.Project, error) {
	queries := r.getQuerier(ctx)
	row, err := queries.GetProjectByID(ctx, db.GetProjectByIDParams{
		ID:     id.String(),
		UserID: userID.String(),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return r.rowToProject(ctx, queries, row)
}

// FindByUser retrieves all projects for a user.
func (r *SQLiteProjectRepository) FindByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Project, error) {
	queries := r.getQuerier(ctx)
	rows, err := queries.GetProjectsByUserID(ctx, userID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get projects: %w", err)
	}

	projects := make([]*domain.Project, 0, len(rows))
	for _, row := range rows {
		p, err := r.rowToProject(ctx, queries, row)
		if err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}

	return projects, nil
}

// FindByStatus retrieves projects by status.
func (r *SQLiteProjectRepository) FindByStatus(ctx context.Context, userID uuid.UUID, status domain.Status) ([]*domain.Project, error) {
	queries := r.getQuerier(ctx)
	rows, err := queries.GetProjectsByStatus(ctx, db.GetProjectsByStatusParams{
		UserID: userID.String(),
		Status: status.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get projects by status: %w", err)
	}

	projects := make([]*domain.Project, 0, len(rows))
	for _, row := range rows {
		p, err := r.rowToProject(ctx, queries, row)
		if err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}

	return projects, nil
}

// FindActive retrieves active projects.
func (r *SQLiteProjectRepository) FindActive(ctx context.Context, userID uuid.UUID) ([]*domain.Project, error) {
	queries := r.getQuerier(ctx)
	rows, err := queries.GetActiveProjects(ctx, userID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get active projects: %w", err)
	}

	projects := make([]*domain.Project, 0, len(rows))
	for _, row := range rows {
		p, err := r.rowToProject(ctx, queries, row)
		if err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}

	return projects, nil
}

// Delete removes a project.
func (r *SQLiteProjectRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	queries := r.getQuerier(ctx)
	return queries.DeleteProject(ctx, db.DeleteProjectParams{
		ID:     id.String(),
		UserID: userID.String(),
	})
}

// SaveMilestone persists a milestone.
func (r *SQLiteProjectRepository) SaveMilestone(ctx context.Context, m *domain.Milestone) error {
	queries := r.getQuerier(ctx)

	var description sql.NullString
	if m.Description() != "" {
		description = sql.NullString{String: m.Description(), Valid: true}
	}

	// Try to update first
	_, err := queries.UpdateMilestone(ctx, db.UpdateMilestoneParams{
		Name:         m.Name(),
		Description:  description,
		DueDate:      m.DueDate().Format(time.RFC3339),
		Status:       m.Status().String(),
		Progress:     m.Progress(),
		DisplayOrder: int64(m.Order()),
		ID:           m.ID().String(),
	})

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Milestone doesn't exist, create it
			_, err = queries.CreateMilestone(ctx, db.CreateMilestoneParams{
				ID:           m.ID().String(),
				ProjectID:    m.ProjectID().String(),
				Name:         m.Name(),
				Description:  description,
				DueDate:      m.DueDate().Format(time.RFC3339),
				Status:       m.Status().String(),
				Progress:     m.Progress(),
				DisplayOrder: int64(m.Order()),
				CreatedAt:    m.CreatedAt().Format(time.RFC3339),
				UpdatedAt:    m.UpdatedAt().Format(time.RFC3339),
			})
			if err != nil {
				return fmt.Errorf("failed to create milestone: %w", err)
			}
		} else {
			return fmt.Errorf("failed to update milestone: %w", err)
		}
	}

	// Save milestone task links
	if err := r.saveMilestoneTaskLinks(ctx, queries, m); err != nil {
		return fmt.Errorf("failed to save milestone task links: %w", err)
	}

	return nil
}

// saveMilestoneTaskLinks saves task links for a milestone.
func (r *SQLiteProjectRepository) saveMilestoneTaskLinks(ctx context.Context, queries *db.Queries, m *domain.Milestone) error {
	// Delete existing links and recreate
	if err := queries.DeleteAllMilestoneTaskLinks(ctx, m.ID().String()); err != nil {
		return err
	}

	for _, link := range m.Tasks() {
		err := queries.CreateMilestoneTaskLink(ctx, db.CreateMilestoneTaskLinkParams{
			MilestoneID:  m.ID().String(),
			TaskID:       link.TaskID.String(),
			Role:         link.Role.String(),
			DisplayOrder: int64(link.Order),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// FindMilestoneByID retrieves a milestone by ID.
func (r *SQLiteProjectRepository) FindMilestoneByID(ctx context.Context, id uuid.UUID) (*domain.Milestone, error) {
	queries := r.getQuerier(ctx)
	row, err := queries.GetMilestoneByID(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrMilestoneNotFound
		}
		return nil, fmt.Errorf("failed to get milestone: %w", err)
	}

	return r.rowToMilestone(ctx, queries, row)
}

// FindMilestonesByProject retrieves all milestones for a project.
func (r *SQLiteProjectRepository) FindMilestonesByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.Milestone, error) {
	queries := r.getQuerier(ctx)
	rows, err := queries.GetMilestonesByProjectID(ctx, projectID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get milestones: %w", err)
	}

	milestones := make([]*domain.Milestone, 0, len(rows))
	for _, row := range rows {
		m, err := r.rowToMilestone(ctx, queries, row)
		if err != nil {
			return nil, err
		}
		milestones = append(milestones, m)
	}

	return milestones, nil
}

// DeleteMilestone removes a milestone.
func (r *SQLiteProjectRepository) DeleteMilestone(ctx context.Context, id uuid.UUID) error {
	queries := r.getQuerier(ctx)
	return queries.DeleteMilestone(ctx, id.String())
}

// rowToProject converts a database row to a domain Project.
func (r *SQLiteProjectRepository) rowToProject(ctx context.Context, queries *db.Queries, row db.Project) (*domain.Project, error) {
	id, err := uuid.Parse(row.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid project id: %w", err)
	}

	userID, err := uuid.Parse(row.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id: %w", err)
	}

	status, err := domain.ParseStatus(row.Status)
	if err != nil {
		return nil, fmt.Errorf("invalid status: %w", err)
	}

	var startDate *time.Time
	if row.StartDate.Valid {
		t, err := time.Parse(time.RFC3339, row.StartDate.String)
		if err != nil {
			return nil, fmt.Errorf("invalid start_date: %w", err)
		}
		startDate = &t
	}

	var dueDate *time.Time
	if row.DueDate.Valid {
		t, err := time.Parse(time.RFC3339, row.DueDate.String)
		if err != nil {
			return nil, fmt.Errorf("invalid due_date: %w", err)
		}
		dueDate = &t
	}

	// Parse risk factors
	var riskFactors []domain.RiskFactor
	if err := json.Unmarshal([]byte(row.HealthRiskFactors), &riskFactors); err != nil {
		return nil, fmt.Errorf("failed to unmarshal risk factors: %w", err)
	}

	healthLastUpdated, err := time.Parse(time.RFC3339, row.HealthLastUpdated)
	if err != nil {
		return nil, fmt.Errorf("invalid health_last_updated: %w", err)
	}

	health := domain.HealthScore{
		Overall:     row.HealthOverall,
		OnTrack:     row.HealthOnTrack == 1,
		RiskFactors: riskFactors,
		LastUpdated: healthLastUpdated,
	}

	// Parse metadata
	metadata := make(map[string]any)
	if err := json.Unmarshal([]byte(row.Metadata), &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	// Get task links
	taskLinkRows, err := queries.GetProjectTaskLinks(ctx, row.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task links: %w", err)
	}

	tasks := make([]domain.TaskLink, 0, len(taskLinkRows))
	for _, link := range taskLinkRows {
		taskID, err := uuid.Parse(link.TaskID)
		if err != nil {
			return nil, fmt.Errorf("invalid task_id in link: %w", err)
		}
		tasks = append(tasks, domain.TaskLink{
			TaskID: taskID,
			Role:   domain.TaskRole(link.Role),
			Order:  int(link.DisplayOrder),
		})
	}

	// Get milestones
	milestoneRows, err := queries.GetMilestonesByProjectID(ctx, row.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get milestones: %w", err)
	}

	milestones := make([]*domain.Milestone, 0, len(milestoneRows))
	for _, mRow := range milestoneRows {
		m, err := r.rowToMilestone(ctx, queries, mRow)
		if err != nil {
			return nil, err
		}
		milestones = append(milestones, m)
	}

	createdAt, err := time.Parse(time.RFC3339, row.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("invalid created_at: %w", err)
	}

	updatedAt, err := time.Parse(time.RFC3339, row.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("invalid updated_at: %w", err)
	}

	description := ""
	if row.Description.Valid {
		description = row.Description.String
	}

	return domain.RehydrateProject(
		id, userID,
		row.Name, description,
		status, startDate, dueDate,
		milestones, tasks,
		health, metadata,
		createdAt, updatedAt,
	), nil
}

// rowToMilestone converts a database row to a domain Milestone.
func (r *SQLiteProjectRepository) rowToMilestone(ctx context.Context, queries *db.Queries, row db.Milestone) (*domain.Milestone, error) {
	id, err := uuid.Parse(row.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid milestone id: %w", err)
	}

	projectID, err := uuid.Parse(row.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project_id: %w", err)
	}

	status, err := domain.ParseStatus(row.Status)
	if err != nil {
		return nil, fmt.Errorf("invalid status: %w", err)
	}

	dueDate, err := time.Parse(time.RFC3339, row.DueDate)
	if err != nil {
		return nil, fmt.Errorf("invalid due_date: %w", err)
	}

	createdAt, err := time.Parse(time.RFC3339, row.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("invalid created_at: %w", err)
	}

	updatedAt, err := time.Parse(time.RFC3339, row.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("invalid updated_at: %w", err)
	}

	// Get milestone task links
	taskLinkRows, err := queries.GetMilestoneTaskLinks(ctx, row.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get milestone task links: %w", err)
	}

	tasks := make([]domain.TaskLink, 0, len(taskLinkRows))
	for _, link := range taskLinkRows {
		taskID, err := uuid.Parse(link.TaskID)
		if err != nil {
			return nil, fmt.Errorf("invalid task_id in milestone link: %w", err)
		}
		tasks = append(tasks, domain.TaskLink{
			TaskID: taskID,
			Role:   domain.TaskRole(link.Role),
			Order:  int(link.DisplayOrder),
		})
	}

	description := ""
	if row.Description.Valid {
		description = row.Description.String
	}

	return domain.RehydrateMilestone(
		id, projectID,
		row.Name, description,
		dueDate, status, tasks,
		row.Progress, int(row.DisplayOrder),
		createdAt, updatedAt,
	), nil
}

// Ensure SQLiteProjectRepository implements domain.Repository.
var _ domain.Repository = (*SQLiteProjectRepository)(nil)
