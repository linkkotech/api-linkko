-- Drop audit_log table
DROP INDEX IF EXISTS idx_audit_actor;
DROP INDEX IF EXISTS idx_audit_workspace_time;
DROP TABLE IF EXISTS audit_log;

-- Drop idempotency_keys table
DROP INDEX IF EXISTS idx_idempotency_expires;
DROP TABLE IF EXISTS idempotency_keys;
