package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"linkko-api/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrContactNotFound      = errors.New("contact not found in workspace")
	ErrContactEmailConflict = errors.New("contact with this email already exists in workspace")
)

type ContactRepository struct {
	pool *pgxpool.Pool
}

func NewContactRepository(pool *pgxpool.Pool) *ContactRepository {
	return &ContactRepository{pool: pool}
}

// List retrieves contacts for a workspace with cursor-based pagination.
// Multi-tenant isolation enforced by workspace_id filter.
func (r *ContactRepository) List(ctx context.Context, params domain.ListContactsParams) ([]domain.Contact, string, error) {
	query := `
		SELECT id, workspace_id, name, email, phone, owner_id, company_id, 
		       tags, custom_fields, created_at, updated_at, deleted_at
		FROM contacts
		WHERE workspace_id = $1 AND deleted_at IS NULL
	`
	args := []interface{}{params.WorkspaceID}
	argIdx := 2

	// Cursor-based pagination: created_at descending
	if params.Cursor != nil && *params.Cursor != "" {
		cursorTime, err := time.Parse(time.RFC3339, *params.Cursor)
		if err != nil {
			return nil, "", fmt.Errorf("invalid cursor format: %w", err)
		}
		query += fmt.Sprintf(" AND created_at < $%d", argIdx)
		args = append(args, cursorTime)
		argIdx++
	}

	if params.ActorID != nil {
		query += fmt.Sprintf(" AND owner_id = $%d", argIdx)
		args = append(args, *params.ActorID)
		argIdx++
	}

	if params.CompanyID != nil {
		query += fmt.Sprintf(" AND company_id = $%d", argIdx)
		args = append(args, *params.CompanyID)
		argIdx++
	}

	if params.Query != nil && *params.Query != "" {
		query += fmt.Sprintf(" AND to_tsvector('simple', name || ' ' || email) @@ plainto_tsquery('simple', $%d)", argIdx)
		args = append(args, *params.Query)
		argIdx++
	}

	query += " ORDER BY created_at DESC"
	query += fmt.Sprintf(" LIMIT $%d", argIdx)
	args = append(args, params.Limit+1) // +1 to check if there's next page

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("query contacts: %w", err)
	}
	defer rows.Close()

	contacts := make([]domain.Contact, 0, params.Limit)
	for rows.Next() {
		var c domain.Contact
		var deletedAt sql.NullTime
		err := rows.Scan(
			&c.ID, &c.WorkspaceID, &c.Name, &c.Email, &c.Phone,
			&c.ActorID, &c.CompanyID, &c.Tags, &c.CustomFields,
			&c.CreatedAt, &c.UpdatedAt, &deletedAt,
		)
		if err != nil {
			return nil, "", fmt.Errorf("scan contact: %w", err)
		}
		if deletedAt.Valid {
			c.DeletedAt = &deletedAt.Time
		}
		contacts = append(contacts, c)
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("iterate contacts: %w", err)
	}

	var nextCursor string
	if len(contacts) > params.Limit {
		nextCursor = contacts[params.Limit-1].CreatedAt.Format(time.RFC3339)
		contacts = contacts[:params.Limit]
	}

	return contacts, nextCursor, nil
}

// Get retrieves a single contact by ID, scoped to workspace.
// IDOR protection: returns not found if contact exists but belongs to another workspace.
func (r *ContactRepository) Get(ctx context.Context, workspaceID, contactID uuid.UUID) (*domain.Contact, error) {
	query := `
		SELECT id, workspace_id, name, email, phone, owner_id, company_id, 
		       tags, custom_fields, created_at, updated_at, deleted_at
		FROM contacts
		WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
	`

	var c domain.Contact
	var deletedAt sql.NullTime
	err := r.pool.QueryRow(ctx, query, contactID, workspaceID).Scan(
		&c.ID, &c.WorkspaceID, &c.Name, &c.Email, &c.Phone,
		&c.ActorID, &c.CompanyID, &c.Tags, &c.CustomFields,
		&c.CreatedAt, &c.UpdatedAt, &deletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrContactNotFound
		}
		return nil, fmt.Errorf("query contact: %w", err)
	}

	if deletedAt.Valid {
		c.DeletedAt = &deletedAt.Time
	}

	return &c, nil
}

// Create inserts a new contact with workspace isolation.
// Returns conflict error if email already exists for this workspace.
func (r *ContactRepository) Create(ctx context.Context, contact *domain.Contact) error {
	query := `
		INSERT INTO contacts (id, workspace_id, name, email, phone, 
		                      owner_id, company_id, tags, custom_fields)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at, updated_at
	`

	err := r.pool.QueryRow(
		ctx, query,
		contact.ID, contact.WorkspaceID, contact.Name, contact.Email, contact.Phone,
		contact.ActorID, contact.CompanyID, contact.Tags, contact.CustomFields,
	).Scan(&contact.CreatedAt, &contact.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return ErrContactEmailConflict
		}
		return fmt.Errorf("insert contact: %w", err)
	}

	return nil
}

// Update modifies an existing contact with optimistic concurrency control.
// Only updates non-nil fields from the request.
func (r *ContactRepository) Update(ctx context.Context, workspaceID, contactID uuid.UUID, updates *domain.UpdateContactRequest, expectedUpdatedAt time.Time) (*domain.Contact, error) {
	// Optimistic concurrency check: verify current updated_at matches expected
	var currentUpdatedAt time.Time
	checkQuery := `SELECT updated_at FROM contacts WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL`
	err := r.pool.QueryRow(ctx, checkQuery, contactID, workspaceID).Scan(&currentUpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrContactNotFound
		}
		return nil, fmt.Errorf("check contact version: %w", err)
	}

	if !currentUpdatedAt.Equal(expectedUpdatedAt) {
		return nil, errors.New("contact was modified by another request")
	}

	// Build dynamic update query based on non-nil fields
	query := "UPDATE contacts SET "
	args := []interface{}{}
	argIdx := 1
	setClauses := []string{}

	if updates.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *updates.Name)
		argIdx++
	}
	if updates.Email != nil {
		setClauses = append(setClauses, fmt.Sprintf("email = $%d", argIdx))
		args = append(args, *updates.Email)
		argIdx++
	}
	if updates.Phone != nil {
		setClauses = append(setClauses, fmt.Sprintf("phone = $%d", argIdx))
		args = append(args, *updates.Phone)
		argIdx++
	}
	if updates.ActorID != nil {
		setClauses = append(setClauses, fmt.Sprintf("owner_id = $%d", argIdx))
		args = append(args, *updates.ActorID)
		argIdx++
	}
	if updates.CompanyID != nil {
		setClauses = append(setClauses, fmt.Sprintf("company_id = $%d", argIdx))
		args = append(args, *updates.CompanyID)
		argIdx++
	}
	if updates.Tags != nil {
		setClauses = append(setClauses, fmt.Sprintf("tags = $%d", argIdx))
		args = append(args, updates.Tags)
		argIdx++
	}
	if updates.CustomFields != nil {
		setClauses = append(setClauses, fmt.Sprintf("custom_fields = $%d", argIdx))
		args = append(args, updates.CustomFields)
		argIdx++
	}

	if len(setClauses) == 0 {
		// No fields to update, return current contact
		return r.Get(ctx, workspaceID, contactID)
	}

	query += fmt.Sprintf("%s WHERE id = $%d AND workspace_id = $%d AND deleted_at IS NULL RETURNING id, workspace_id, name, email, phone, owner_id, company_id, tags, custom_fields, created_at, updated_at",
		joinClauses(setClauses), argIdx, argIdx+1)
	args = append(args, contactID, workspaceID)

	var c domain.Contact
	err = r.pool.QueryRow(ctx, query, args...).Scan(
		&c.ID, &c.WorkspaceID, &c.Name, &c.Email, &c.Phone,
		&c.ActorID, &c.CompanyID, &c.Tags, &c.CustomFields,
		&c.CreatedAt, &c.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrContactNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrContactEmailConflict
		}
		return nil, fmt.Errorf("update contact: %w", err)
	}

	return &c, nil
}

// SoftDelete marks a contact as deleted without removing from database.
// Preserves data for audit and potential recovery.
func (r *ContactRepository) SoftDelete(ctx context.Context, workspaceID, contactID uuid.UUID) error {
	query := `
		UPDATE contacts
		SET deleted_at = NOW()
		WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
	`

	result, err := r.pool.Exec(ctx, query, contactID, workspaceID)
	if err != nil {
		return fmt.Errorf("soft delete contact: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrContactNotFound
	}

	return nil
}

func joinClauses(clauses []string) string {
	result := ""
	for i, clause := range clauses {
		if i > 0 {
			result += ", "
		}
		result += clause
	}
	return result
}
