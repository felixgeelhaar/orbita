-- name: CreateProject :one
INSERT INTO projects (
    id, user_id, name, description, status,
    start_date, due_date, health_overall, health_on_track,
    health_risk_factors, health_last_updated, metadata,
    created_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING *;

-- name: GetProjectByID :one
SELECT * FROM projects WHERE id = $1 AND user_id = $2;

-- name: GetProjectsByUserID :many
SELECT * FROM projects
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetProjectsByStatus :many
SELECT * FROM projects
WHERE user_id = $1 AND status = $2
ORDER BY created_at DESC;

-- name: GetActiveProjects :many
SELECT * FROM projects
WHERE user_id = $1 AND status NOT IN ('completed', 'archived')
ORDER BY
    CASE WHEN due_date IS NULL THEN 1 ELSE 0 END,
    due_date,
    created_at DESC;

-- name: UpdateProject :one
UPDATE projects
SET
    name = $1,
    description = $2,
    status = $3,
    start_date = $4,
    due_date = $5,
    health_overall = $6,
    health_on_track = $7,
    health_risk_factors = $8,
    health_last_updated = $9,
    metadata = $10,
    updated_at = NOW()
WHERE id = $11 AND user_id = $12
RETURNING *;

-- name: DeleteProject :exec
DELETE FROM projects WHERE id = $1 AND user_id = $2;

-- Milestones

-- name: CreateMilestone :one
INSERT INTO milestones (
    id, project_id, name, description, due_date,
    status, progress, display_order, created_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetMilestoneByID :one
SELECT * FROM milestones WHERE id = $1;

-- name: GetMilestonesByProjectID :many
SELECT * FROM milestones
WHERE project_id = $1
ORDER BY display_order, due_date;

-- name: UpdateMilestone :one
UPDATE milestones
SET
    name = $1,
    description = $2,
    due_date = $3,
    status = $4,
    progress = $5,
    display_order = $6,
    updated_at = NOW()
WHERE id = $7
RETURNING *;

-- name: DeleteMilestone :exec
DELETE FROM milestones WHERE id = $1;

-- Project Task Links

-- name: CreateProjectTaskLink :exec
INSERT INTO project_task_links (project_id, task_id, role, display_order, created_at)
VALUES ($1, $2, $3, $4, NOW());

-- name: GetProjectTaskLinks :many
SELECT * FROM project_task_links
WHERE project_id = $1
ORDER BY display_order;

-- name: DeleteProjectTaskLink :exec
DELETE FROM project_task_links WHERE project_id = $1 AND task_id = $2;

-- name: DeleteAllProjectTaskLinks :exec
DELETE FROM project_task_links WHERE project_id = $1;

-- Milestone Task Links

-- name: CreateMilestoneTaskLink :exec
INSERT INTO milestone_task_links (milestone_id, task_id, role, display_order, created_at)
VALUES ($1, $2, $3, $4, NOW());

-- name: GetMilestoneTaskLinks :many
SELECT * FROM milestone_task_links
WHERE milestone_id = $1
ORDER BY display_order;

-- name: DeleteMilestoneTaskLink :exec
DELETE FROM milestone_task_links WHERE milestone_id = $1 AND task_id = $2;

-- name: DeleteAllMilestoneTaskLinks :exec
DELETE FROM milestone_task_links WHERE milestone_id = $1;
