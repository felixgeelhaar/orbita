-- Remove entitlement
DELETE FROM entitlements WHERE id = 'automations-pro';

-- Remove trigger
DROP TRIGGER IF EXISTS tr_automation_rules_updated_at ON automation_rules;
DROP FUNCTION IF EXISTS update_automation_rule_updated_at();

-- Drop tables
DROP TABLE IF EXISTS automation_pending_actions;
DROP TABLE IF EXISTS automation_rule_executions;
DROP TABLE IF EXISTS automation_rules;
