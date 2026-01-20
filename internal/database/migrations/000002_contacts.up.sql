-- Create contacts table
CREATE TABLE IF NOT EXISTS contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    phone VARCHAR(50),
    company_id UUID,
    owner_id UUID NOT NULL,
    tags TEXT[],
    custom_fields JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    
    -- Constraints
    CONSTRAINT unique_email_per_workspace UNIQUE(workspace_id, email) WHERE deleted_at IS NULL
);

-- Indexes for performance
CREATE INDEX idx_contacts_workspace ON contacts(workspace_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_contacts_owner ON contacts(owner_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_contacts_company ON contacts(company_id) WHERE deleted_at IS NULL AND company_id IS NOT NULL;
CREATE INDEX idx_contacts_email ON contacts(workspace_id, email) WHERE deleted_at IS NULL;
CREATE INDEX idx_contacts_created_at ON contacts(workspace_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_contacts_search ON contacts USING gin(to_tsvector('simple', name || ' ' || email)) WHERE deleted_at IS NULL;

-- Updated at trigger
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_contacts_updated_at
    BEFORE UPDATE ON contacts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
