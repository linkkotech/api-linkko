package domain

import (
	"time"
)

// =====================================================
// Workspace Role Constants (Type Safety)
// =====================================================

// Role represents a workspace role ID (canonical identifier from DB)
type Role string

const (
	// RoleAdmin has full access including member management
	RoleAdmin Role = "work_admin"

	// RoleManager can create, read, update resources but not manage members
	RoleManager Role = "work_manager"

	// RoleUser can create and read resources but not modify others' data
	RoleUser Role = "work_user"

	// RoleViewer has read-only access to workspace resources
	RoleViewer Role = "work_viewer"
)

// String returns the string representation of the Role
func (r Role) String() string {
	return string(r)
}

// IsValid checks if the role is one of the defined constants
func (r Role) IsValid() bool {
	switch r {
	case RoleAdmin, RoleManager, RoleUser, RoleViewer:
		return true
	default:
		return false
	}
}

// =====================================================
// Workspace Role Entity (DB Model)
// =====================================================

// WorkspaceRole represents a role definition in the system.
// This maps to the WorkspaceRole table which is the source of truth.
type WorkspaceRole struct {
	ID          string    `json:"id" db:"id"`                   // Canonical role ID (e.g., 'work_admin')
	Name        string    `json:"name" db:"name"`               // Display name (e.g., 'Workspace Admin')
	Description *string   `json:"description" db:"description"` // Human-readable description
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`    // When role was defined
}

// =====================================================
// Workspace Member Entity (DB Model)
// =====================================================

// WorkspaceMember represents the membership of a user/agent in a workspace.
// This is a junction table mapping actors to workspaces with their assigned roles.
type WorkspaceMember struct {
	// Identity
	UserID      string `json:"userId" db:"userId"`           // Actor ID (user or AI agent)
	WorkspaceID string `json:"workspaceId" db:"workspaceId"` // Workspace ID

	// Authorization
	WorkspaceRoleID Role `json:"workspaceRoleId" db:"workspaceRoleId"` // Role ID (work_admin, etc.)

	// Invitation metadata
	InvitedBy  *string    `json:"invitedBy,omitempty" db:"invited_by"`   // User who invited this member
	InvitedAt  time.Time  `json:"invitedAt" db:"invited_at"`             // When invitation was sent
	AcceptedAt *time.Time `json:"acceptedAt,omitempty" db:"accepted_at"` // When invitation was accepted (NULL if pending)

	// Timestamps
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// IsPending returns true if the member invitation has not been accepted yet
func (m *WorkspaceMember) IsPending() bool {
	return m.AcceptedAt == nil
}

// =====================================================
// RBAC Permission Helpers
// =====================================================
// These functions define the permission hierarchy for workspace operations.
// Moved from service layer to domain for reusability across services.

// IsWorkspaceMember checks if the role is valid for workspace access
func IsWorkspaceMember(role Role) bool {
	return role == RoleAdmin || role == RoleManager || role == RoleUser || role == RoleViewer
}

// CanModifyContacts checks if the role can create/update contacts
func CanModifyContacts(role Role) bool {
	return role == RoleAdmin || role == RoleManager || role == RoleUser
}

// CanDeleteContacts checks if the role can delete contacts
func CanDeleteContacts(role Role) bool {
	return role == RoleAdmin || role == RoleManager
}

// CanManageMembers checks if the role can invite/remove workspace members
func CanManageMembers(role Role) bool {
	return role == RoleAdmin
}

// CanManageWorkspace checks if the role can modify workspace settings
func CanManageWorkspace(role Role) bool {
	return role == RoleAdmin
}

// =====================================================
// Permission Matrix Documentation
// =====================================================
//
// | Operation          | Admin | Manager | User | Viewer |
// |--------------------|-------|---------|------|--------|
// | List Contacts      | ✅    | ✅      | ✅   | ✅     |
// | View Contact       | ✅    | ✅      | ✅   | ✅     |
// | Create Contact     | ✅    | ✅      | ✅   | ❌     |
// | Update Contact     | ✅    | ✅      | ✅   | ❌     |
// | Delete Contact     | ✅    | ✅      | ❌   | ❌     |
// | Invite Member      | ✅    | ❌      | ❌   | ❌     |
// | Remove Member      | ✅    | ❌      | ❌   | ❌     |
// | Update Workspace   | ✅    | ❌      | ❌   | ❌     |
//
// Note: This matrix is enforced by the helper functions above.
// To modify permissions, update the corresponding helper function.
