-- Time Insights tables for the insights bounded context

-- Daily productivity snapshots for pre-computed analytics
CREATE TABLE IF NOT EXISTS productivity_snapshots (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    snapshot_date DATE NOT NULL,

    -- Task metrics
    tasks_created INTEGER NOT NULL DEFAULT 0,
    tasks_completed INTEGER NOT NULL DEFAULT 0,
    tasks_overdue INTEGER NOT NULL DEFAULT 0,
    task_completion_rate DECIMAL(5,2) DEFAULT 0,  -- Percentage
    avg_task_duration_minutes INTEGER DEFAULT 0,

    -- Time block metrics
    blocks_scheduled INTEGER NOT NULL DEFAULT 0,
    blocks_completed INTEGER NOT NULL DEFAULT 0,
    blocks_missed INTEGER NOT NULL DEFAULT 0,
    scheduled_minutes INTEGER NOT NULL DEFAULT 0,
    completed_minutes INTEGER NOT NULL DEFAULT 0,
    block_completion_rate DECIMAL(5,2) DEFAULT 0,  -- Percentage

    -- Habit metrics
    habits_due INTEGER NOT NULL DEFAULT 0,
    habits_completed INTEGER NOT NULL DEFAULT 0,
    habit_completion_rate DECIMAL(5,2) DEFAULT 0,  -- Percentage
    longest_streak INTEGER NOT NULL DEFAULT 0,

    -- Focus metrics
    focus_sessions INTEGER NOT NULL DEFAULT 0,
    total_focus_minutes INTEGER NOT NULL DEFAULT 0,
    avg_focus_session_minutes INTEGER DEFAULT 0,

    -- Productivity score (0-100)
    productivity_score INTEGER NOT NULL DEFAULT 0,

    -- Peak hours (JSON array of {hour: int, completions: int})
    peak_hours JSONB DEFAULT '[]',

    -- Category breakdown (JSON: {category: minutes})
    time_by_category JSONB DEFAULT '{}',

    -- Metadata
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(user_id, snapshot_date)
);

-- Time sessions for tracking actual focused work periods
CREATE TABLE IF NOT EXISTS time_sessions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- What was being worked on
    session_type VARCHAR(50) NOT NULL,  -- 'task', 'habit', 'focus', 'meeting', 'other'
    reference_id UUID,  -- ID of the task/habit/meeting if applicable
    title VARCHAR(255) NOT NULL,
    category VARCHAR(100),

    -- Timing
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ,
    duration_minutes INTEGER,  -- Computed on end

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'active',  -- 'active', 'completed', 'interrupted', 'abandoned'

    -- Quality metrics
    interruptions INTEGER NOT NULL DEFAULT 0,
    notes TEXT,

    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_session_type CHECK (session_type IN ('task', 'habit', 'focus', 'meeting', 'other')),
    CONSTRAINT chk_session_status CHECK (status IN ('active', 'completed', 'interrupted', 'abandoned'))
);

-- Weekly summary for trend analysis
CREATE TABLE IF NOT EXISTS weekly_summaries (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    week_start DATE NOT NULL,  -- Monday of the week
    week_end DATE NOT NULL,    -- Sunday of the week

    -- Aggregated metrics
    total_tasks_completed INTEGER NOT NULL DEFAULT 0,
    total_habits_completed INTEGER NOT NULL DEFAULT 0,
    total_blocks_completed INTEGER NOT NULL DEFAULT 0,
    total_focus_minutes INTEGER NOT NULL DEFAULT 0,

    -- Averages
    avg_daily_productivity_score DECIMAL(5,2) DEFAULT 0,
    avg_daily_focus_minutes INTEGER DEFAULT 0,

    -- Comparison to previous week
    productivity_trend DECIMAL(5,2) DEFAULT 0,  -- Percentage change
    focus_trend DECIMAL(5,2) DEFAULT 0,  -- Percentage change

    -- Best/worst days
    most_productive_day DATE,
    least_productive_day DATE,

    -- Streaks maintained
    habits_with_streak INTEGER NOT NULL DEFAULT 0,
    longest_streak INTEGER NOT NULL DEFAULT 0,

    -- Metadata
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(user_id, week_start)
);

-- Goals for tracking personal targets
CREATE TABLE IF NOT EXISTS productivity_goals (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Goal type
    goal_type VARCHAR(50) NOT NULL,  -- 'daily_tasks', 'weekly_focus', 'habit_streak', etc.
    target_value INTEGER NOT NULL,
    current_value INTEGER NOT NULL DEFAULT 0,

    -- Time period
    period_type VARCHAR(20) NOT NULL,  -- 'daily', 'weekly', 'monthly'
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,

    -- Status
    achieved BOOLEAN NOT NULL DEFAULT FALSE,
    achieved_at TIMESTAMPTZ,

    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_goal_type CHECK (goal_type IN (
        'daily_tasks', 'daily_focus_minutes', 'daily_habits',
        'weekly_tasks', 'weekly_focus_minutes', 'weekly_habits',
        'monthly_tasks', 'monthly_focus_minutes', 'habit_streak'
    )),
    CONSTRAINT chk_period_type CHECK (period_type IN ('daily', 'weekly', 'monthly'))
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_productivity_snapshots_user_date ON productivity_snapshots(user_id, snapshot_date DESC);
CREATE INDEX IF NOT EXISTS idx_time_sessions_user_id ON time_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_time_sessions_user_started ON time_sessions(user_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_time_sessions_reference ON time_sessions(reference_id) WHERE reference_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_time_sessions_active ON time_sessions(user_id) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_weekly_summaries_user_week ON weekly_summaries(user_id, week_start DESC);
CREATE INDEX IF NOT EXISTS idx_productivity_goals_user_active ON productivity_goals(user_id, period_end) WHERE NOT achieved;

-- Triggers for updated_at
CREATE TRIGGER productivity_snapshots_updated_at_trigger
    BEFORE UPDATE ON productivity_snapshots
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER time_sessions_updated_at_trigger
    BEFORE UPDATE ON time_sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER productivity_goals_updated_at_trigger
    BEFORE UPDATE ON productivity_goals
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
