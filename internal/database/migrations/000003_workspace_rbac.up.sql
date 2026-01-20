-- Migration: 000003_workspace_rbac.up.sql
-- Description: Create workspace RBAC tables (WorkspaceRole and WorkspaceMember)
-- Date: 2026-01-20
-- Author: Linkko Platform Team

-- =====================================================
-- Table: WorkspaceRole
-- Purpose: Define available roles in a workspace
-- =====================================================
CREATE TABLE IF NOT EXISTS "WorkspaceRole" (
    id TEXT PRIMARY KEY,                    -- Canonical role ID (e.g., 'work_admin')
    name VARCHAR(100) NOT NULL,             -- Display name (e.g., 'Workspace Admin')
    description TEXT,                       -- Human-readable description
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Insert default workspace roles
INSERT INTO "WorkspaceRole" (id, name, description) VALUES
    ('work_admin', 'Admin', 'Full access to workspace including member management'),
    ('work_manager', 'Manager', 'Can create, read, update resources but not manage members'),
    ('work_user', 'User', 'Can create and read resources but not modify others'' data'),
    ('work_viewer', 'Viewer', 'Read-only access to workspace resources')
ON CONFLICT (id) DO NOTHING;

-- =====================================================
-- Table: WorkspaceMember
-- Purpose: Map users to workspaces with roles (junction table)
-- =====================================================
CREATE TABLE IF NOT EXISTS "WorkspaceMember" (
    "userId" UUID NOT NULL,                 -- Actor ID (user or AI agent)
    "workspaceId" UUID NOT NULL,            -- Workspace ID
    "workspaceRoleId" TEXT NOT NULL,        -- Role ID (FK to WorkspaceRole)
    
    -- Metadata
    invited_by UUID,                        -- User who invited this member
    invited_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    accepted_at TIMESTAMPTZ,                -- NULL if invitation pending
    
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraints
    PRIMARY KEY ("userId", "workspaceId"),
    FOREIGN KEY ("workspaceRoleId") REFERENCES "WorkspaceRole"(id) ON DELETE RESTRICT
);

-- =====================================================
-- Indexes for Performance
-- =====================================================
-- Index for fast role lookup (primary use case: GetMemberRole)
CREATE INDEX IF NOT EXISTS idx_workspace_member_lookup 
    ON "WorkspaceMember" ("workspaceId", "userId");

-- Index for listing members of a workspace
CREATE INDEX IF NOT EXISTS idx_workspace_member_list 
    ON "WorkspaceMember" ("workspaceId", "workspaceRoleId");

-- Index for finding all workspaces a user belongs to
CREATE INDEX IF NOT EXISTS idx_user_workspaces 
    ON "WorkspaceMember" ("userId", "workspaceId");

-- =====================================================
-- Comments for Documentation
-- =====================================================
COMMENT ON TABLE "WorkspaceRole" IS 'Available workspace roles for RBAC (source of truth)';
COMMENT ON TABLE "WorkspaceMember" IS 'Junction table mapping users to workspaces with roles';
COMMENT ON COLUMN "WorkspaceMember"."userId" IS 'Actor ID (user or AI agent) - maps to owner_id in contacts table';
COMMENT ON COLUMN "WorkspaceMember"."workspaceId" IS 'Workspace ID for multi-tenant isolation';
COMMENT ON COLUMN "WorkspaceMember"."workspaceRoleId" IS 'Role ID determining permissions (work_admin, work_manager, work_user, work_viewer)';

-- =====================================================
-- Sample Seed Data for Development/Testing
-- =====================================================
-- Uncomment the following lines to add test data:
-- 
-- INSERT INTO "WorkspaceMember" ("userId", "workspaceId", "workspaceRoleId", invited_by, accepted_at) VALUES
--     ('00000000-0000-0000-0000-000000000001'::UUID, '11111111-1111-1111-1111-111111111111'::UUID, 'work_admin', NULL, NOW()),
--     ('00000000-0000-0000-0000-000000000002'::UUID, '11111111-1111-1111-1111-111111111111'::UUID, 'work_manager', '00000000-0000-0000-0000-000000000001'::UUID, NOW()),
--     ('00000000-0000-0000-0000-000000000003'::UUID, '11111111-1111-1111-1111-111111111111'::UUID, 'work_user', '00000000-0000-0000-0000-000000000001'::UUID, NOW()),
--     ('00000000-0000-0000-0000-000000000004'::UUID, '11111111-1111-1111-1111-111111111111'::UUID, 'work_viewer', '00000000-0000-0000-0000-000000000001'::UUID, NOW());
