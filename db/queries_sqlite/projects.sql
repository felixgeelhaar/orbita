-- name: CreateProject :one
INSERT INTO projects (
    id, user_id, name, description, status,
    start_date, due_date, health_overall, health_on_track,
    health_risk_factors, health_last_updated, metadata,
    created_at, updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetProjectByID :one
SELECT * FROM projects WHERE id = ? AND user_id = ?;

-- name: GetProjectsByUserID :many
SELECT * FROM projects
WHERE user_id = ?
ORDER BY created_at DESC;

-- name: GetProjectsByStatus :many
SELECT * FROM projects
WHERE user_id = ? AND status = ?
ORDER BY created_at DESC;

-- name: GetActiveProjects :many
SELECT * FROM projects
WHERE user_id = ? AND status NOT IN ('completed', 'archived')
ORDER BY
    CASE WHEN due_date IS NULL THEN 1 ELSE 0 END,
    due_date,
    created_at DESC;

-- name: UpdateProject :one
UPDATE projects
SET
    name = ?,
    description = ?,
    status = ?,
    start_date = ?,
    due_date = ?,
    health_overall = ?,
    health_on_track = ?,
    health_risk_factors = ?,
    health_last_updated = ?,
    metadata = ?,
    updated_at = datetime('now')
WHERE id = ? AND user_id = ?
RETURNING *;

-- name: DeleteProject :exec
DELETE FROM projects WHERE id = ? AND user_id = ?;

-- Milestones

-- name: CreateMilestone :one
INSERT INTO milestones (
    id, project_id, name, description, due_date,
    status, progress, display_order, created_at, updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetMilestoneByID :one
SELECT * FROM milestones WHERE id = ?;

-- name: GetMilestonesByProjectID :many
SELECT * FROM milestones
WHERE project_id = ?
ORDER BY display_order, due_date;

-- name: UpdateMilestone :one
UPDATE milestones
SET
    name = ?,
    description = ?,
    due_date = ?,
    status = ?,
    progress = ?,
    display_order = ?,
    updated_at = datetime('now')
WHERE id = ?
RETURNING *;

-- name: DeleteMilestone :exec
DELETE FROM milestones WHERE id = ?;

-- Project Task Links

-- name: CreateProjectTaskLink :exec
INSERT INTO project_task_links (project_id, task_id, role, display_order, created_at)
VALUES (?, ?, ?, ?, datetime('now'));

-- name: GetProjectTaskLinks :many
SELECT * FROM project_task_links
WHERE project_id = ?
ORDER BY display_order;

-- name: DeleteProjectTaskLink :exec
DELETE FROM project_task_links WHERE project_id = ? AND task_id = ?;

-- name: DeleteAllProjectTaskLinks :exec
DELETE FROM project_task_links WHERE project_id = ?;

-- Milestone Task Links

-- name: CreateMilestoneTaskLink :exec
INSERT INTO milestone_task_links (milestone_id, task_id, role, display_order, created_at)
VALUES (?, ?, ?, ?, datetime('now'));

-- name: GetMilestoneTaskLinks :many
SELECT * FROM milestone_task_links
WHERE milestone_id = ?
ORDER BY display_order;

-- name: DeleteMilestoneTaskLink :exec
DELETE FROM milestone_task_links WHERE milestone_id = ? AND task_id = ?;

-- name: DeleteAllMilestoneTaskLinks :exec
DELETE FROM milestone_task_links WHERE milestone_id = ?;
