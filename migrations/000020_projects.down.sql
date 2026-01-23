-- Drop triggers
DROP TRIGGER IF EXISTS update_projects_updated_at ON projects;
DROP TRIGGER IF EXISTS update_milestones_updated_at ON milestones;

-- Drop tables in reverse order of creation
DROP TABLE IF EXISTS milestone_task_links;
DROP TABLE IF EXISTS project_task_links;
DROP TABLE IF EXISTS milestones;
DROP TABLE IF EXISTS projects;
