CREATE TABLE IF NOT EXISTS priority_scores (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    score DOUBLE PRECISION NOT NULL,
    explanation TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, task_id)
);

CREATE INDEX IF NOT EXISTS idx_priority_scores_user_id ON priority_scores(user_id);
CREATE INDEX IF NOT EXISTS idx_priority_scores_task_id ON priority_scores(task_id);
