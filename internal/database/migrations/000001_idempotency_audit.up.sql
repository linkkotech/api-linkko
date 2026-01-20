-- Create idempotency_keys table for idempotent request handling
CREATE TABLE IF NOT EXISTS idempotency_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_hash VARCHAR(64) NOT NULL,
    workspace_id UUID NOT NULL,
    original_key VARCHAR(255),
    request_method VARCHAR(10),
    request_path VARCHAR(500),
    request_payload JSONB,
    response_status INT,
    response_body JSONB,
    response_headers JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT unique_workspace_key UNIQUE(workspace_id, key_hash)
);

-- Index for cleanup queries
CREATE INDEX idx_idempotency_expires ON idempotency_keys(expires_at) WHERE expires_at IS NOT NULL;

-- Create audit_log table for action tracking
CREATE TABLE IF NOT EXISTS audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    actor_id UUID,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50),
    resource_id UUID,
    metadata JSONB,
    ip_address INET,
    user_agent VARCHAR(500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX idx_audit_workspace_time ON audit_log(workspace_id, created_at DESC);
CREATE INDEX idx_audit_actor ON audit_log(actor_id, created_at DESC);
