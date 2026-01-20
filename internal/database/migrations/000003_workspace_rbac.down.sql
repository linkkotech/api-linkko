-- Migration: 000003_workspace_rbac.down.sql
-- Description: Rollback workspace RBAC tables
-- Date: 2026-01-20

-- Drop tables in reverse order (respecting FK constraints)
DROP TABLE IF EXISTS "WorkspaceMember";
DROP TABLE IF EXISTS "WorkspaceRole";
