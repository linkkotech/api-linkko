package repo

import (
	"context"
	"errors"
	"fmt"

	"linkko-api/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// =====================================================
// Error Definitions
// =====================================================

var (
	// ErrMemberNotFound indicates the user is not a member of the workspace
	ErrMemberNotFound = errors.New("user is not a member of this workspace")

	// ErrInvalidRole indicates the role ID does not exist in WorkspaceRole table
	ErrInvalidRole = errors.New("invalid workspace role")
)

// =====================================================
// Repository Definition
// =====================================================

// WorkspaceRepository handles database operations for workspace membership and roles.
// Follows the repository pattern established in contact.go (concrete struct, no interface).
type WorkspaceRepository struct {
	pool *pgxpool.Pool
}

// NewWorkspaceRepository creates a new WorkspaceRepository instance.
// Dependency injection pattern: requires a pgxpool.Pool for database access.
func NewWorkspaceRepository(pool *pgxpool.Pool) *WorkspaceRepository {
	return &WorkspaceRepository{pool: pool}
}

// =====================================================
// Core Methods
// =====================================================

// GetMemberRole retrieves the workspace role for a specific user in a workspace.
// This is the primary authorization check method called by service layers.
//
// Returns:
//   - Role name (e.g., "work_admin", "work_manager") if member exists
//   - ErrMemberNotFound if user is not a member of the workspace
//   - Other errors for database failures
//
// Performance: Uses indexed lookup on (workspaceId, userId) with JOIN to WorkspaceRole - ~1-5ms typical query time.
//
// Security: This method enforces multi-tenant isolation. A user cannot access
// resources in a workspace they don't belong to.
//
// Implementation Note: WorkspaceRole.id contains CUIDs (e.g., 'clworkspace_admin')
// while WorkspaceRole.name contains semantic role names (e.g., 'work_admin').
// This JOIN maps CUID to semantic name for Go domain validation.
func (r *WorkspaceRepository) GetMemberRole(ctx context.Context, userID string, workspaceID string) (domain.Role, error) {
	query := `
		SELECT r.name
		FROM "WorkspaceMember" m
		JOIN "WorkspaceRole" r ON m."workspaceRoleId" = r.id
		WHERE m."userId" = $1 AND m."workspaceId" = $2
	`

	var roleName string
	err := r.pool.QueryRow(ctx, query, userID, workspaceID).Scan(&roleName)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// User is not a member of this workspace
			// Return ErrMemberNotFound for handlers to return 403 Forbidden
			return "", ErrMemberNotFound
		}
		// Database error (connection issue, syntax error, etc.)
		return "", fmt.Errorf("query workspace member role: %w", err)
	}

	// Convert string to domain.Role type for type safety
	role := domain.Role(roleName)

	// Validate role is one of the expected values
	// This protects against data corruption in the database
	if !role.IsValid() {
		return "", fmt.Errorf("invalid role '%s' for user %s in workspace %s: %w", roleName, userID, workspaceID, ErrInvalidRole)
	}

	return role, nil
}

// IsMember checks if a user is a member of a workspace (any role).
// This is a lighter check than GetMemberRole if you only need membership verification.
//
// Returns:
//   - true if user has any role in the workspace
//   - false if user is not a member
//   - error for database failures
func (r *WorkspaceRepository) IsMember(ctx context.Context, userID string, workspaceID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM "WorkspaceMember"
			WHERE "userId" = $1 AND "workspaceId" = $2
		)
	`

	var exists bool
	err := r.pool.QueryRow(ctx, query, userID, workspaceID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check workspace membership: %w", err)
	}

	return exists, nil
}

// =====================================================
// Additional Helper Methods (Future Expansion)
// =====================================================
// These methods are not needed for the current CRM Contacts implementation,
// but are included for future workspace management features.

// ListMembersByWorkspace retrieves all members of a workspace.
// Useful for workspace member management UI.
func (r *WorkspaceRepository) ListMembersByWorkspace(ctx context.Context, workspaceID string) ([]domain.WorkspaceMember, error) {
	query := `
		SELECT 
			"userId", "workspaceId", "workspaceRoleId",
			invited_by, invited_at, accepted_at,
			created_at, updated_at
		FROM "WorkspaceMember"
		WHERE "workspaceId" = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("query workspace members: %w", err)
	}
	defer rows.Close()

	var members []domain.WorkspaceMember
	for rows.Next() {
		var m domain.WorkspaceMember
		err := rows.Scan(
			&m.UserID, &m.WorkspaceID, &m.WorkspaceRoleID,
			&m.InvitedBy, &m.InvitedAt, &m.AcceptedAt,
			&m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan workspace member: %w", err)
		}
		members = append(members, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspace members: %w", err)
	}

	return members, nil
}

// ListWorkspacesByUser retrieves all workspaces a user is a member of.
// Useful for workspace switcher UI or multi-workspace dashboards.
func (r *WorkspaceRepository) ListWorkspacesByUser(ctx context.Context, userID string) ([]domain.WorkspaceMember, error) {
	query := `
		SELECT 
			"userId", "workspaceId", "workspaceRoleId",
			invited_by, invited_at, accepted_at,
			created_at, updated_at
		FROM "WorkspaceMember"
		WHERE "userId" = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query user workspaces: %w", err)
	}
	defer rows.Close()

	var memberships []domain.WorkspaceMember
	for rows.Next() {
		var m domain.WorkspaceMember
		err := rows.Scan(
			&m.UserID, &m.WorkspaceID, &m.WorkspaceRoleID,
			&m.InvitedBy, &m.InvitedAt, &m.AcceptedAt,
			&m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan workspace membership: %w", err)
		}
		memberships = append(memberships, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspace memberships: %w", err)
	}

	return memberships, nil
}
