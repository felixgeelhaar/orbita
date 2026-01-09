-- Automation Rules Table
CREATE TABLE IF NOT EXISTS automation_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    priority INT NOT NULL DEFAULT 0,

    -- Trigger configuration (stored as JSONB)
    trigger_type VARCHAR(50) NOT NULL, -- event, schedule, state_change, pattern
    trigger_config JSONB NOT NULL DEFAULT '{}',

    -- Conditions (array of condition objects)
    conditions JSONB NOT NULL DEFAULT '[]',
    condition_operator VARCHAR(10) NOT NULL DEFAULT 'AND', -- AND, OR

    -- Actions (array of action objects)
    actions JSONB NOT NULL DEFAULT '[]',

    -- Rate limiting
    cooldown_seconds INT NOT NULL DEFAULT 0, -- minimum seconds between triggers
    max_executions_per_hour INT, -- null means unlimited

    -- Metadata
    tags TEXT[] DEFAULT '{}',

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_triggered_at TIMESTAMPTZ,

    -- Constraints
    CONSTRAINT chk_trigger_type CHECK (trigger_type IN ('event', 'schedule', 'state_change', 'pattern')),
    CONSTRAINT chk_condition_operator CHECK (condition_operator IN ('AND', 'OR'))
);

-- Indexes for automation_rules
CREATE INDEX idx_automation_rules_user_id ON automation_rules(user_id);
CREATE INDEX idx_automation_rules_user_enabled ON automation_rules(user_id, enabled) WHERE enabled = TRUE;
CREATE INDEX idx_automation_rules_trigger_type ON automation_rules(trigger_type);
CREATE INDEX idx_automation_rules_tags ON automation_rules USING GIN(tags);

-- Automation Rule Executions (History)
CREATE TABLE IF NOT EXISTS automation_rule_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID NOT NULL REFERENCES automation_rules(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Trigger information
    trigger_event_type VARCHAR(100),
    trigger_event_payload JSONB,

    -- Execution result
    status VARCHAR(20) NOT NULL, -- success, failed, skipped, pending

    -- Actions executed
    actions_executed JSONB NOT NULL DEFAULT '[]', -- array of {action, status, result, error}

    -- Error information
    error_message TEXT,
    error_details JSONB,

    -- Timing
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    duration_ms INT,

    -- Skip reason if skipped
    skip_reason VARCHAR(100),

    CONSTRAINT chk_execution_status CHECK (status IN ('success', 'failed', 'skipped', 'pending', 'partial'))
);

-- Indexes for automation_rule_executions
CREATE INDEX idx_automation_executions_rule_id ON automation_rule_executions(rule_id);
CREATE INDEX idx_automation_executions_user_id ON automation_rule_executions(user_id);
CREATE INDEX idx_automation_executions_status ON automation_rule_executions(status);
CREATE INDEX idx_automation_executions_started_at ON automation_rule_executions(started_at DESC);
CREATE INDEX idx_automation_executions_rule_started ON automation_rule_executions(rule_id, started_at DESC);

-- Pending Actions (for delayed execution)
CREATE TABLE IF NOT EXISTS automation_pending_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES automation_rule_executions(id) ON DELETE CASCADE,
    rule_id UUID NOT NULL REFERENCES automation_rules(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Action details
    action_type VARCHAR(100) NOT NULL,
    action_params JSONB NOT NULL DEFAULT '{}',

    -- Scheduling
    scheduled_for TIMESTAMPTZ NOT NULL,

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, executed, cancelled, failed
    executed_at TIMESTAMPTZ,

    -- Result
    result JSONB,
    error_message TEXT,

    -- Retries
    retry_count INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 3,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_pending_action_status CHECK (status IN ('pending', 'executed', 'cancelled', 'failed'))
);

-- Indexes for automation_pending_actions
CREATE INDEX idx_pending_actions_status_scheduled ON automation_pending_actions(status, scheduled_for)
    WHERE status = 'pending';
CREATE INDEX idx_pending_actions_user_id ON automation_pending_actions(user_id);
CREATE INDEX idx_pending_actions_rule_id ON automation_pending_actions(rule_id);

-- Trigger to update updated_at
CREATE OR REPLACE FUNCTION update_automation_rule_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tr_automation_rules_updated_at
    BEFORE UPDATE ON automation_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_automation_rule_updated_at();

-- Seed entitlement for Automations Pro
INSERT INTO entitlements (id, name, description, created_at)
VALUES (
    'automations-pro',
    'Automations Pro',
    'Advanced automation features including webhooks, patterns, and delayed actions'
, NOW())
ON CONFLICT (id) DO NOTHING;
