-- Drop triggers
DROP TRIGGER IF EXISTS update_projects_updated_at;
DROP TRIGGER IF EXISTS update_milestones_updated_at;

-- Drop indexes
DROP INDEX IF EXISTS idx_milestone_task_links_task_id;
DROP INDEX IF EXISTS idx_project_task_links_task_id;
DROP INDEX IF EXISTS idx_milestones_due_date;
DROP INDEX IF EXISTS idx_milestones_project_id;
DROP INDEX IF EXISTS idx_projects_due_date;
DROP INDEX IF EXISTS idx_projects_user_status;
DROP INDEX IF EXISTS idx_projects_user_id;

-- Drop tables in reverse order of creation
DROP TABLE IF EXISTS milestone_task_links;
DROP TABLE IF EXISTS project_task_links;
DROP TABLE IF EXISTS milestones;
DROP TABLE IF EXISTS projects;
