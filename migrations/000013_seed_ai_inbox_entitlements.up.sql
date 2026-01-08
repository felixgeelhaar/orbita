INSERT INTO entitlements (user_id, module, active, source, created_at, updated_at)
SELECT u.id, 'ai-inbox', true, 'default', NOW(), NOW()
FROM users u
WHERE NOT EXISTS (
    SELECT 1
    FROM entitlements e
    WHERE e.user_id = u.id AND e.module = 'ai-inbox'
);
