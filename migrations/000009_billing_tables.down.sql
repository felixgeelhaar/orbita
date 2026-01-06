DROP TRIGGER IF EXISTS entitlements_updated_at_trigger ON entitlements;
DROP FUNCTION IF EXISTS update_entitlements_updated_at();
DROP TABLE IF EXISTS entitlements;

DROP TRIGGER IF EXISTS subscriptions_updated_at_trigger ON subscriptions;
DROP FUNCTION IF EXISTS update_subscriptions_updated_at();
DROP TABLE IF EXISTS subscriptions;
