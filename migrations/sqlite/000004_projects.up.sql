-- Projects table for SQLite
CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'planning' CHECK (status IN ('planning', 'active', 'on_hold', 'completed', 'archived')),
    start_date TEXT,
    due_date TEXT,
    health_overall REAL NOT NULL DEFAULT 1.0 CHECK (health_overall >= 0 AND health_overall <= 1),
    health_on_track INTEGER NOT NULL DEFAULT 1,
    health_risk_factors TEXT NOT NULL DEFAULT '[]', -- JSON array
    health_last_updated TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    metadata TEXT NOT NULL DEFAULT '{}', -- JSON object
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_projects_user_id ON projects (user_id);
CREATE INDEX IF NOT EXISTS idx_projects_user_status ON projects (user_id, status);
CREATE INDEX IF NOT EXISTS idx_projects_due_date ON projects (due_date);

-- Milestones table for SQLite
CREATE TABLE IF NOT EXISTS milestones (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    due_date TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'planning' CHECK (status IN ('planning', 'active', 'on_hold', 'completed', 'archived')),
    progress REAL NOT NULL DEFAULT 0.0 CHECK (progress >= 0 AND progress <= 1),
    display_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_milestones_project_id ON milestones (project_id);
CREATE INDEX IF NOT EXISTS idx_milestones_due_date ON milestones (due_date);

-- Project task links (tasks directly linked to projects, not through milestones)
CREATE TABLE IF NOT EXISTS project_task_links (
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'subtask' CHECK (role IN ('blocker', 'dependency', 'deliverable', 'subtask')),
    display_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    PRIMARY KEY (project_id, task_id)
);

CREATE INDEX IF NOT EXISTS idx_project_task_links_task_id ON project_task_links (task_id);

-- Milestone task links (tasks linked to milestones)
CREATE TABLE IF NOT EXISTS milestone_task_links (
    milestone_id TEXT NOT NULL REFERENCES milestones(id) ON DELETE CASCADE,
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'subtask' CHECK (role IN ('blocker', 'dependency', 'deliverable', 'subtask')),
    display_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    PRIMARY KEY (milestone_id, task_id)
);

CREATE INDEX IF NOT EXISTS idx_milestone_task_links_task_id ON milestone_task_links (task_id);

-- Triggers for updated_at
CREATE TRIGGER IF NOT EXISTS update_projects_updated_at AFTER UPDATE ON projects
BEGIN
    UPDATE projects SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_milestones_updated_at AFTER UPDATE ON milestones
BEGIN
    UPDATE milestones SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = NEW.id;
END;
