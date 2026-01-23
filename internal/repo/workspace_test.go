package repo_test

import (
	"context"
	"os"
	"testing"

	"linkko-api/internal/database"
	"linkko-api/internal/domain"
	"linkko-api/internal/repo"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorkspaceRepository_GetMemberRole_Integration validates the CUID -> semantic role name mapping
// using a real database. This test ensures the JOIN between WorkspaceMember and WorkspaceRole
// correctly returns semantic role names (work_admin, work_manager, etc.) instead of CUIDs.
//
// Prerequisites:
//   - DATABASE_URL environment variable must be set
//   - Database must have WorkspaceRole table with CUID ids and semantic names
//   - Migration 000003_workspace_rbac must be applied
//
// Run with: go test -v ./internal/repo -run TestWorkspaceRepository_GetMemberRole_Integration
func TestWorkspaceRepository_GetMemberRole_Integration(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	ctx := context.Background()
	databaseURL := os.Getenv("DATABASE_URL")

	// Connect to database
	pool, err := database.NewPool(ctx, databaseURL)
	require.NoError(t, err, "failed to connect to database")
	defer pool.Close()

	// Initialize repository
	workspaceRepo := repo.NewWorkspaceRepository(pool)

	// Test data setup
	testWorkspaceID := "test-workspace-id-001"
	testUserID := "test-user-id-001"

	// Cleanup: Remove test data before and after test
	cleanup := func() {
		_, _ = pool.Exec(ctx, `DELETE FROM "WorkspaceMember" WHERE "workspaceId" = $1`, testWorkspaceID)
	}
	cleanup()
	defer cleanup()

	tests := []struct {
		name             string
		workspaceRoleID  string // CUID format
		expectedRole     domain.Role
		setupMember      bool
		expectedError    error
		errorContains    string
		validateIsValid  bool
	}{
		{
			name:            "work_admin role mapping (CUID to semantic name)",
			workspaceRoleID: "clworkspace_admin",
			expectedRole:    domain.RoleAdmin,
			setupMember:     true,
			validateIsValid: true,
		},
		{
			name:            "work_manager role mapping",
			workspaceRoleID: "clworkspace_manager",
			expectedRole:    domain.RoleManager,
			setupMember:     true,
			validateIsValid: true,
		},
		{
			name:            "work_user role mapping",
			workspaceRoleID: "clworkspace_user",
			expectedRole:    domain.RoleUser,
			setupMember:     true,
			validateIsValid: true,
		},
		{
			name:            "work_viewer role mapping",
			workspaceRoleID: "clworkspace_viewer",
			expectedRole:    domain.RoleViewer,
			setupMember:     true,
			validateIsValid: true,
		},
		{
			name:          "member not found",
			setupMember:   false,
			expectedError: repo.ErrMemberNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup: Insert WorkspaceMember if needed
			if tt.setupMember {
				_, err := pool.Exec(ctx, `
					INSERT INTO "WorkspaceMember" ("userId", "workspaceId", "workspaceRoleId", invited_at)
					VALUES ($1, $2, $3, NOW())
					ON CONFLICT ("userId", "workspaceId") DO UPDATE SET "workspaceRoleId" = $3
				`, testUserID, testWorkspaceID, tt.workspaceRoleID)
				require.NoError(t, err, "failed to setup test member")
			}

			// Execute: Call GetMemberRole
			role, err := workspaceRepo.GetMemberRole(ctx, testUserID, testWorkspaceID)

			// Assert: Verify results
			if tt.expectedError != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedRole, role, "role mismatch: expected %s, got %s", tt.expectedRole, role)

				// Validate role passes domain.Role.IsValid() check
				if tt.validateIsValid {
					assert.True(t, role.IsValid(), "role should be valid: %s", role)
				}
			}

			// Cleanup after each test case
			if tt.setupMember {
				cleanup()
			}
		})
	}
}

// TestWorkspaceRepository_GetMemberRole_InvalidRole validates that invalid role names
// in the database are properly rejected by the IsValid() check.
func TestWorkspaceRepository_GetMemberRole_InvalidRole(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	ctx := context.Background()
	databaseURL := os.Getenv("DATABASE_URL")

	pool, err := database.NewPool(ctx, databaseURL)
	require.NoError(t, err)
	defer pool.Close()

	workspaceRepo := repo.NewWorkspaceRepository(pool)

	testWorkspaceID := "test-workspace-invalid-role"
	testUserID := "test-user-invalid-role"

	// Cleanup
	cleanup := func() {
		_, _ = pool.Exec(ctx, `DELETE FROM "WorkspaceRole" WHERE id = 'invalid_role_cuid'`)
		_, _ = pool.Exec(ctx, `DELETE FROM "WorkspaceMember" WHERE "workspaceId" = $1`, testWorkspaceID)
	}
	cleanup()
	defer cleanup()

	// Setup: Insert a WorkspaceRole with an invalid semantic name
	_, err = pool.Exec(ctx, `
		INSERT INTO "WorkspaceRole" (id, name, created_at)
		VALUES ('invalid_role_cuid', 'invalid_role', NOW())
	`)
	require.NoError(t, err)

	// Insert WorkspaceMember with invalid role
	_, err = pool.Exec(ctx, `
		INSERT INTO "WorkspaceMember" ("userId", "workspaceId", "workspaceRoleId", invited_at)
		VALUES ($1, $2, 'invalid_role_cuid', NOW())
	`, testUserID, testWorkspaceID)
	require.NoError(t, err)

	// Execute: Should return error for invalid role
	role, err := workspaceRepo.GetMemberRole(ctx, testUserID, testWorkspaceID)

	// Assert: Should error on invalid role
	require.Error(t, err)
	assert.ErrorIs(t, err, repo.ErrInvalidRole)
	assert.Empty(t, role)
	assert.Contains(t, err.Error(), "invalid role")
}

// TestWorkspaceRepository_IsMember validates membership check without role details
func TestWorkspaceRepository_IsMember(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	ctx := context.Background()
	databaseURL := os.Getenv("DATABASE_URL")

	pool, err := database.NewPool(ctx, databaseURL)
	require.NoError(t, err)
	defer pool.Close()

	workspaceRepo := repo.NewWorkspaceRepository(pool)

	testWorkspaceID := "test-workspace-membership"
	testUserID := "test-user-membership"

	// Cleanup
	cleanup := func() {
		_, _ = pool.Exec(ctx, `DELETE FROM "WorkspaceMember" WHERE "workspaceId" = $1`, testWorkspaceID)
	}
	cleanup()
	defer cleanup()

	// Test: Non-member
	isMember, err := workspaceRepo.IsMember(ctx, testUserID, testWorkspaceID)
	require.NoError(t, err)
	assert.False(t, isMember, "user should not be a member initially")

	// Setup: Add member
	_, err = pool.Exec(ctx, `
		INSERT INTO "WorkspaceMember" ("userId", "workspaceId", "workspaceRoleId", invited_at)
		VALUES ($1, $2, 'clworkspace_user', NOW())
	`, testUserID, testWorkspaceID)
	require.NoError(t, err)

	// Test: Is member
	isMember, err = workspaceRepo.IsMember(ctx, testUserID, testWorkspaceID)
	require.NoError(t, err)
	assert.True(t, isMember, "user should be a member after insert")
}
