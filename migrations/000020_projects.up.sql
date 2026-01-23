-- Projects table
CREATE TABLE projects (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'planning',
    start_date TIMESTAMPTZ,
    due_date TIMESTAMPTZ,
    health_overall DECIMAL(4,3) NOT NULL DEFAULT 1.0,
    health_on_track BOOLEAN NOT NULL DEFAULT TRUE,
    health_risk_factors JSONB NOT NULL DEFAULT '[]'::JSONB,
    health_last_updated TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_project_status CHECK (status IN ('planning', 'active', 'on_hold', 'completed', 'archived')),
    CONSTRAINT chk_health_overall CHECK (health_overall >= 0 AND health_overall <= 1)
);

CREATE INDEX idx_projects_user_id ON projects (user_id);
CREATE INDEX idx_projects_user_status ON projects (user_id, status);
CREATE INDEX idx_projects_due_date ON projects (due_date) WHERE due_date IS NOT NULL;

-- Milestones table
CREATE TABLE milestones (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    due_date TIMESTAMPTZ NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'planning',
    progress DECIMAL(4,3) NOT NULL DEFAULT 0.0,
    display_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_milestone_status CHECK (status IN ('planning', 'active', 'on_hold', 'completed', 'archived')),
    CONSTRAINT chk_milestone_progress CHECK (progress >= 0 AND progress <= 1)
);

CREATE INDEX idx_milestones_project_id ON milestones (project_id);
CREATE INDEX idx_milestones_due_date ON milestones (due_date);

-- Project task links (tasks directly linked to projects, not through milestones)
CREATE TABLE project_task_links (
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL DEFAULT 'subtask',
    display_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (project_id, task_id),
    CONSTRAINT chk_project_task_role CHECK (role IN ('blocker', 'dependency', 'deliverable', 'subtask'))
);

CREATE INDEX idx_project_task_links_task_id ON project_task_links (task_id);

-- Milestone task links (tasks linked to milestones)
CREATE TABLE milestone_task_links (
    milestone_id UUID NOT NULL REFERENCES milestones(id) ON DELETE CASCADE,
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL DEFAULT 'subtask',
    display_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (milestone_id, task_id),
    CONSTRAINT chk_milestone_task_role CHECK (role IN ('blocker', 'dependency', 'deliverable', 'subtask'))
);

CREATE INDEX idx_milestone_task_links_task_id ON milestone_task_links (task_id);

-- Apply updated_at triggers
CREATE TRIGGER update_projects_updated_at
    BEFORE UPDATE ON projects
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_milestones_updated_at
    BEFORE UPDATE ON milestones
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
