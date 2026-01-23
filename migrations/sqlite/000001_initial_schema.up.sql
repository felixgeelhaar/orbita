-- SQLite Schema for Orbita (consolidated from PostgreSQL migrations)
-- This file contains the complete schema for local/SQLite mode

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);

-- Tasks table
CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'in_progress', 'completed', 'archived')),
    priority TEXT NOT NULL DEFAULT 'none' CHECK (priority IN ('none', 'low', 'medium', 'high', 'urgent')),
    duration_minutes INTEGER,
    due_date TEXT,
    completed_at TEXT,
    version INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_tasks_user_id ON tasks (user_id);
CREATE INDEX IF NOT EXISTS idx_tasks_user_status ON tasks (user_id, status);
CREATE INDEX IF NOT EXISTS idx_tasks_due_date ON tasks (due_date);

-- Outbox table for reliable event publishing
CREATE TABLE IF NOT EXISTS outbox (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id TEXT UNIQUE,
    aggregate_type TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    routing_key TEXT NOT NULL,
    payload TEXT NOT NULL,
    metadata TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    published_at TEXT,
    retry_count INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    next_retry_at TEXT,
    dead_lettered_at TEXT,
    dead_letter_reason TEXT
);

CREATE INDEX IF NOT EXISTS idx_outbox_unpublished ON outbox (created_at);
CREATE INDEX IF NOT EXISTS idx_outbox_aggregate ON outbox (aggregate_type, aggregate_id);
CREATE INDEX IF NOT EXISTS idx_outbox_retry ON outbox (retry_count, created_at);
CREATE INDEX IF NOT EXISTS idx_outbox_event_id ON outbox (event_id);
CREATE INDEX IF NOT EXISTS idx_outbox_next_retry ON outbox (next_retry_at, created_at);

-- Habits table
CREATE TABLE IF NOT EXISTS habits (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    frequency TEXT NOT NULL DEFAULT 'daily',
    times_per_week INTEGER NOT NULL DEFAULT 7,
    duration_minutes INTEGER NOT NULL DEFAULT 30,
    preferred_time TEXT DEFAULT 'anytime',
    streak INTEGER NOT NULL DEFAULT 0,
    best_streak INTEGER NOT NULL DEFAULT 0,
    total_done INTEGER NOT NULL DEFAULT 0,
    archived INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_habits_user_id ON habits (user_id);
CREATE INDEX IF NOT EXISTS idx_habits_user_archived ON habits (user_id, archived);

-- Habit completions tracking
CREATE TABLE IF NOT EXISTS habit_completions (
    id TEXT PRIMARY KEY,
    habit_id TEXT NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
    completed_at TEXT NOT NULL,
    notes TEXT DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_habit_completions_habit_id ON habit_completions (habit_id);
CREATE INDEX IF NOT EXISTS idx_habit_completions_completed_at ON habit_completions (habit_id, completed_at);

-- Schedules table
CREATE TABLE IF NOT EXISTS schedules (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    schedule_date TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE(user_id, schedule_date)
);

CREATE INDEX IF NOT EXISTS idx_schedules_user_id ON schedules (user_id);
CREATE INDEX IF NOT EXISTS idx_schedules_user_date ON schedules (user_id, schedule_date);

-- Time blocks table for scheduled activities
CREATE TABLE IF NOT EXISTS time_blocks (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    schedule_id TEXT NOT NULL REFERENCES schedules(id) ON DELETE CASCADE,
    block_type TEXT NOT NULL,
    reference_id TEXT,
    title TEXT NOT NULL,
    start_time TEXT NOT NULL,
    end_time TEXT NOT NULL,
    completed INTEGER NOT NULL DEFAULT 0,
    missed INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    CHECK (end_time > start_time)
);

CREATE INDEX IF NOT EXISTS idx_time_blocks_schedule_id ON time_blocks (schedule_id);
CREATE INDEX IF NOT EXISTS idx_time_blocks_user_id ON time_blocks (user_id);
CREATE INDEX IF NOT EXISTS idx_time_blocks_start_time ON time_blocks (schedule_id, start_time);
CREATE INDEX IF NOT EXISTS idx_time_blocks_reference ON time_blocks (reference_id);

-- OAuth tokens table
CREATE TABLE IF NOT EXISTS oauth_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    access_token BLOB NOT NULL,
    refresh_token BLOB,
    token_type TEXT,
    expiry TEXT,
    scopes TEXT, -- comma-separated list (PostgreSQL uses TEXT[])
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE (user_id, provider)
);

-- User settings table
CREATE TABLE IF NOT EXISTS user_settings (
    user_id TEXT PRIMARY KEY,
    calendar_id TEXT NOT NULL DEFAULT 'primary',
    delete_missing INTEGER NOT NULL DEFAULT 0,
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- Meetings table
CREATE TABLE IF NOT EXISTS meetings (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    cadence TEXT NOT NULL DEFAULT 'weekly',
    cadence_days INTEGER NOT NULL DEFAULT 7,
    duration_minutes INTEGER NOT NULL DEFAULT 30,
    preferred_time_minutes INTEGER NOT NULL DEFAULT 600,
    last_held_at TEXT,
    archived INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_meetings_user_id ON meetings (user_id);
CREATE INDEX IF NOT EXISTS idx_meetings_user_archived ON meetings (user_id, archived);

-- Billing: Entitlements table (simplified from modules)
CREATE TABLE IF NOT EXISTS entitlements (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- Billing: User entitlements table
CREATE TABLE IF NOT EXISTS user_entitlements (
    user_id TEXT NOT NULL,
    entitlement_id TEXT NOT NULL REFERENCES entitlements(id),
    stripe_subscription_id TEXT,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'cancelled', 'past_due', 'trialing')),
    expires_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    PRIMARY KEY (user_id, entitlement_id)
);

CREATE INDEX IF NOT EXISTS idx_user_entitlements_user ON user_entitlements (user_id);
CREATE INDEX IF NOT EXISTS idx_user_entitlements_stripe ON user_entitlements (stripe_subscription_id);

-- Reschedule attempts table
CREATE TABLE IF NOT EXISTS reschedule_attempts (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    schedule_id TEXT NOT NULL REFERENCES schedules(id) ON DELETE CASCADE,
    block_id TEXT NOT NULL REFERENCES time_blocks(id) ON DELETE CASCADE,
    attempt_type TEXT NOT NULL,
    success INTEGER NOT NULL,
    failure_reason TEXT,
    old_start_time TEXT NOT NULL,
    old_end_time TEXT NOT NULL,
    new_start_time TEXT,
    new_end_time TEXT,
    attempted_at TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_reschedule_attempts_schedule ON reschedule_attempts (schedule_id);
CREATE INDEX IF NOT EXISTS idx_reschedule_attempts_user ON reschedule_attempts (user_id);
CREATE INDEX IF NOT EXISTS idx_reschedule_attempts_block ON reschedule_attempts (block_id);
CREATE INDEX IF NOT EXISTS idx_reschedule_attempts_attempted ON reschedule_attempts (attempted_at);

-- Inbox items table
CREATE TABLE IF NOT EXISTS inbox_items (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    metadata TEXT NOT NULL DEFAULT '{}', -- JSON object
    tags TEXT NOT NULL DEFAULT '[]', -- JSON array
    source TEXT NOT NULL,
    classification TEXT NOT NULL DEFAULT '',
    captured_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    promoted INTEGER NOT NULL DEFAULT 0,
    promoted_to TEXT,
    promoted_id TEXT,
    promoted_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_inbox_items_user_id ON inbox_items (user_id);
CREATE INDEX IF NOT EXISTS idx_inbox_items_user_promoted ON inbox_items (user_id, promoted);
CREATE INDEX IF NOT EXISTS idx_inbox_items_captured_at ON inbox_items (user_id, captured_at);

-- Marketplace: Packages table
CREATE TABLE IF NOT EXISTS marketplace_packages (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    author TEXT NOT NULL,
    description TEXT,
    category TEXT NOT NULL,
    version TEXT NOT NULL,
    manifest TEXT NOT NULL, -- JSON
    downloads INTEGER NOT NULL DEFAULT 0,
    rating REAL,
    review_count INTEGER NOT NULL DEFAULT 0,
    published_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_marketplace_packages_category ON marketplace_packages (category);
CREATE INDEX IF NOT EXISTS idx_marketplace_packages_author ON marketplace_packages (author);

-- Installed packages table
CREATE TABLE IF NOT EXISTS installed_packages (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    package_id TEXT NOT NULL REFERENCES marketplace_packages(id),
    installed_version TEXT NOT NULL,
    config TEXT, -- JSON
    enabled INTEGER NOT NULL DEFAULT 1,
    installed_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE (user_id, package_id)
);

CREATE INDEX IF NOT EXISTS idx_installed_packages_user ON installed_packages (user_id);

-- Automation rules table
CREATE TABLE IF NOT EXISTS automation_rules (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    enabled INTEGER NOT NULL DEFAULT 1,
    priority INTEGER NOT NULL DEFAULT 0,
    trigger_type TEXT NOT NULL CHECK (trigger_type IN ('event', 'schedule', 'state_change', 'pattern')),
    trigger_config TEXT NOT NULL DEFAULT '{}', -- JSON
    conditions TEXT NOT NULL DEFAULT '[]', -- JSON array
    condition_operator TEXT NOT NULL DEFAULT 'AND' CHECK (condition_operator IN ('AND', 'OR')),
    actions TEXT NOT NULL DEFAULT '[]', -- JSON array
    cooldown_seconds INTEGER NOT NULL DEFAULT 0,
    max_executions_per_hour INTEGER,
    tags TEXT DEFAULT '[]', -- JSON array (PostgreSQL uses TEXT[])
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    last_triggered_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_automation_rules_user_id ON automation_rules (user_id);
CREATE INDEX IF NOT EXISTS idx_automation_rules_trigger_type ON automation_rules (trigger_type);

-- Automation rule executions
CREATE TABLE IF NOT EXISTS automation_rule_executions (
    id TEXT PRIMARY KEY,
    rule_id TEXT NOT NULL REFERENCES automation_rules(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    trigger_event_type TEXT,
    trigger_event_payload TEXT, -- JSON
    status TEXT NOT NULL CHECK (status IN ('success', 'failed', 'skipped', 'pending', 'partial')),
    actions_executed TEXT NOT NULL DEFAULT '[]', -- JSON array
    error_message TEXT,
    error_details TEXT, -- JSON
    started_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    completed_at TEXT,
    duration_ms INTEGER,
    skip_reason TEXT
);

CREATE INDEX IF NOT EXISTS idx_automation_executions_rule_id ON automation_rule_executions (rule_id);
CREATE INDEX IF NOT EXISTS idx_automation_executions_user_id ON automation_rule_executions (user_id);
CREATE INDEX IF NOT EXISTS idx_automation_executions_status ON automation_rule_executions (status);
CREATE INDEX IF NOT EXISTS idx_automation_executions_started_at ON automation_rule_executions (started_at);

-- Automation pending actions
CREATE TABLE IF NOT EXISTS automation_pending_actions (
    id TEXT PRIMARY KEY,
    execution_id TEXT NOT NULL REFERENCES automation_rule_executions(id) ON DELETE CASCADE,
    rule_id TEXT NOT NULL REFERENCES automation_rules(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action_type TEXT NOT NULL,
    action_params TEXT NOT NULL DEFAULT '{}', -- JSON
    scheduled_for TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'executed', 'cancelled', 'failed')),
    executed_at TEXT,
    result TEXT, -- JSON
    error_message TEXT,
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 3,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_pending_actions_status_scheduled ON automation_pending_actions (status, scheduled_for);
CREATE INDEX IF NOT EXISTS idx_pending_actions_user_id ON automation_pending_actions (user_id);
CREATE INDEX IF NOT EXISTS idx_pending_actions_rule_id ON automation_pending_actions (rule_id);

-- Productivity snapshots for insights
CREATE TABLE IF NOT EXISTS productivity_snapshots (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    snapshot_date TEXT NOT NULL,
    tasks_created INTEGER NOT NULL DEFAULT 0,
    tasks_completed INTEGER NOT NULL DEFAULT 0,
    tasks_overdue INTEGER NOT NULL DEFAULT 0,
    task_completion_rate REAL DEFAULT 0,
    avg_task_duration_minutes INTEGER DEFAULT 0,
    blocks_scheduled INTEGER NOT NULL DEFAULT 0,
    blocks_completed INTEGER NOT NULL DEFAULT 0,
    blocks_missed INTEGER NOT NULL DEFAULT 0,
    scheduled_minutes INTEGER NOT NULL DEFAULT 0,
    completed_minutes INTEGER NOT NULL DEFAULT 0,
    block_completion_rate REAL DEFAULT 0,
    habits_due INTEGER NOT NULL DEFAULT 0,
    habits_completed INTEGER NOT NULL DEFAULT 0,
    habit_completion_rate REAL DEFAULT 0,
    longest_streak INTEGER NOT NULL DEFAULT 0,
    focus_sessions INTEGER NOT NULL DEFAULT 0,
    total_focus_minutes INTEGER NOT NULL DEFAULT 0,
    avg_focus_session_minutes INTEGER DEFAULT 0,
    productivity_score INTEGER NOT NULL DEFAULT 0,
    peak_hours TEXT DEFAULT '[]', -- JSON array
    time_by_category TEXT DEFAULT '{}', -- JSON object
    computed_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE(user_id, snapshot_date)
);

CREATE INDEX IF NOT EXISTS idx_productivity_snapshots_user_date ON productivity_snapshots (user_id, snapshot_date);

-- Time sessions for tracking work periods
CREATE TABLE IF NOT EXISTS time_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_type TEXT NOT NULL CHECK (session_type IN ('task', 'habit', 'focus', 'meeting', 'other')),
    reference_id TEXT,
    title TEXT NOT NULL,
    category TEXT,
    started_at TEXT NOT NULL,
    ended_at TEXT,
    duration_minutes INTEGER,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'completed', 'interrupted', 'abandoned')),
    interruptions INTEGER NOT NULL DEFAULT 0,
    notes TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_time_sessions_user_id ON time_sessions (user_id);
CREATE INDEX IF NOT EXISTS idx_time_sessions_user_started ON time_sessions (user_id, started_at);
CREATE INDEX IF NOT EXISTS idx_time_sessions_reference ON time_sessions (reference_id);

-- Weekly summaries for trends
CREATE TABLE IF NOT EXISTS weekly_summaries (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    week_start TEXT NOT NULL,
    week_end TEXT NOT NULL,
    total_tasks_completed INTEGER NOT NULL DEFAULT 0,
    total_habits_completed INTEGER NOT NULL DEFAULT 0,
    total_blocks_completed INTEGER NOT NULL DEFAULT 0,
    total_focus_minutes INTEGER NOT NULL DEFAULT 0,
    avg_daily_productivity_score REAL DEFAULT 0,
    avg_daily_focus_minutes INTEGER DEFAULT 0,
    productivity_trend REAL DEFAULT 0,
    focus_trend REAL DEFAULT 0,
    most_productive_day TEXT,
    least_productive_day TEXT,
    habits_with_streak INTEGER NOT NULL DEFAULT 0,
    longest_streak INTEGER NOT NULL DEFAULT 0,
    computed_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE(user_id, week_start)
);

CREATE INDEX IF NOT EXISTS idx_weekly_summaries_user_week ON weekly_summaries (user_id, week_start);

-- Productivity goals
CREATE TABLE IF NOT EXISTS productivity_goals (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    goal_type TEXT NOT NULL CHECK (goal_type IN (
        'daily_tasks', 'daily_focus_minutes', 'daily_habits',
        'weekly_tasks', 'weekly_focus_minutes', 'weekly_habits',
        'monthly_tasks', 'monthly_focus_minutes', 'habit_streak'
    )),
    target_value INTEGER NOT NULL,
    current_value INTEGER NOT NULL DEFAULT 0,
    period_type TEXT NOT NULL CHECK (period_type IN ('daily', 'weekly', 'monthly')),
    period_start TEXT NOT NULL,
    period_end TEXT NOT NULL,
    achieved INTEGER NOT NULL DEFAULT 0,
    achieved_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_productivity_goals_user_period ON productivity_goals (user_id, period_end);

-- Projects table
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

-- Milestones table
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

-- Project task links
CREATE TABLE IF NOT EXISTS project_task_links (
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'subtask' CHECK (role IN ('blocker', 'dependency', 'deliverable', 'subtask')),
    display_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    PRIMARY KEY (project_id, task_id)
);

CREATE INDEX IF NOT EXISTS idx_project_task_links_task_id ON project_task_links (task_id);

-- Milestone task links
CREATE TABLE IF NOT EXISTS milestone_task_links (
    milestone_id TEXT NOT NULL REFERENCES milestones(id) ON DELETE CASCADE,
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'subtask' CHECK (role IN ('blocker', 'dependency', 'deliverable', 'subtask')),
    display_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    PRIMARY KEY (milestone_id, task_id)
);

CREATE INDEX IF NOT EXISTS idx_milestone_task_links_task_id ON milestone_task_links (task_id);

-- Seed core entitlements
INSERT OR IGNORE INTO entitlements (id, name, description, created_at) VALUES
    ('core-tasks', 'Core Tasks', 'Basic task management', strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    ('core-calendar', 'Core Calendar', 'Calendar integration', strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    ('smart-habits', 'Smart Habits', 'Adaptive habit tracking', strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    ('smart-1on1', 'Smart 1:1 Scheduler', 'Intelligent meeting scheduling', strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    ('auto-reschedule', 'Auto-Rescheduler', 'Automatic rescheduling on conflicts', strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    ('ai-inbox', 'AI Inbox Pro', 'AI-powered inbox processing', strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    ('priority-engine', 'Priority Engine Pro', 'Advanced prioritization', strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    ('focus-mode', 'Focus Mode Pro', 'Deep work protection', strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    ('ideal-week', 'Ideal Week Designer', 'Weekly template planning', strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    ('project-ai', 'Project AI Assistant', 'AI project breakdown', strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    ('time-insights', 'Time Insights', 'Time tracking analytics', strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    ('family-scheduler', 'Couples & Family Scheduler', 'Shared scheduling', strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    ('automations-pro', 'Automations Pro', 'Advanced automation features', strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    ('wellness', 'Wellness Sync', 'Health integration', strftime('%Y-%m-%dT%H:%M:%SZ', 'now'));

-- SQLite triggers for updated_at
CREATE TRIGGER IF NOT EXISTS update_users_updated_at AFTER UPDATE ON users
BEGIN
    UPDATE users SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_tasks_updated_at AFTER UPDATE ON tasks
BEGIN
    UPDATE tasks SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_habits_updated_at AFTER UPDATE ON habits
BEGIN
    UPDATE habits SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_schedules_updated_at AFTER UPDATE ON schedules
BEGIN
    UPDATE schedules SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_time_blocks_updated_at AFTER UPDATE ON time_blocks
BEGIN
    UPDATE time_blocks SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_automation_rules_updated_at AFTER UPDATE ON automation_rules
BEGIN
    UPDATE automation_rules SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_productivity_snapshots_updated_at AFTER UPDATE ON productivity_snapshots
BEGIN
    UPDATE productivity_snapshots SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_time_sessions_updated_at AFTER UPDATE ON time_sessions
BEGIN
    UPDATE time_sessions SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_productivity_goals_updated_at AFTER UPDATE ON productivity_goals
BEGIN
    UPDATE productivity_goals SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_projects_updated_at AFTER UPDATE ON projects
BEGIN
    UPDATE projects SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_milestones_updated_at AFTER UPDATE ON milestones
BEGIN
    UPDATE milestones SET updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = NEW.id;
END;
