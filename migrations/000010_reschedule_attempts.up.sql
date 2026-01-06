-- Reschedule attempts for audit and analytics
CREATE TABLE IF NOT EXISTS reschedule_attempts (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    schedule_id UUID NOT NULL REFERENCES schedules(id) ON DELETE CASCADE,
    block_id UUID NOT NULL REFERENCES time_blocks(id) ON DELETE CASCADE,
    attempt_type VARCHAR(50) NOT NULL,
    success BOOLEAN NOT NULL,
    failure_reason TEXT,
    old_start_time TIMESTAMPTZ NOT NULL,
    old_end_time TIMESTAMPTZ NOT NULL,
    new_start_time TIMESTAMPTZ,
    new_end_time TIMESTAMPTZ,
    attempted_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_reschedule_attempts_user_id ON reschedule_attempts(user_id);
CREATE INDEX IF NOT EXISTS idx_reschedule_attempts_schedule_id ON reschedule_attempts(schedule_id);
CREATE INDEX IF NOT EXISTS idx_reschedule_attempts_block_id ON reschedule_attempts(block_id);
CREATE INDEX IF NOT EXISTS idx_reschedule_attempts_attempted_at ON reschedule_attempts(attempted_at);
