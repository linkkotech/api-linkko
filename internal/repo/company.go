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
	ErrCompanyNotFound       = errors.New("company not found in workspace")
	ErrCompanyDomainConflict = errors.New("company with this domain already exists in workspace")
)

type CompanyRepository struct {
	pool *pgxpool.Pool
}

func NewCompanyRepository(pool *pgxpool.Pool) *CompanyRepository {
	return &CompanyRepository{pool: pool}
}

// List retrieves companies for a workspace with optional filters.
// IMPORTANT: Uses camelCase column names with double quotes.
func (r *CompanyRepository) List(ctx context.Context, params domain.ListCompaniesParams) ([]domain.Company, string, error) {
	query := `
		SELECT id, "workspaceId", name, domain, industry, "lifecycleStage", "size",
		       phone, email, website, address, "revenue", "employeeCount",
		       "ownerId", tags, "customFields", notes,
		       "createdAt", "updatedAt", "deletedAt"
		FROM public."Company"
		WHERE "workspaceId" = $1 AND "deletedAt" IS NULL
	`
	args := []interface{}{params.WorkspaceID}
	argIdx := 2

	// Filtro por lifecycle stage
	if params.LifecycleStage != nil {
		query += fmt.Sprintf(` AND "lifecycleStage" = $%d`, argIdx)
		args = append(args, *params.LifecycleStage)
		argIdx++
	}

	// Filtro por company size
	if params.Size != nil {
		query += fmt.Sprintf(` AND "size" = $%d`, argIdx)
		args = append(args, *params.Size)
		argIdx++
	}

	// Filtro por industry
	if params.Industry != nil {
		query += fmt.Sprintf(` AND industry = $%d`, argIdx)
		args = append(args, *params.Industry)
		argIdx++
	}

	// Filtro por owner
	if params.OwnerID != nil {
		query += fmt.Sprintf(` AND "ownerId" = $%d`, argIdx)
		args = append(args, *params.OwnerID)
		argIdx++
	}

	// Busca textual (name + domain)
	if params.Query != nil && *params.Query != "" {
		query += fmt.Sprintf(` AND to_tsvector('simple', name || ' ' || COALESCE(domain, '')) @@ plainto_tsquery('simple', $%d)`, argIdx)
		args = append(args, *params.Query)
		argIdx++
	}

	// Cursor-based pagination
	if params.Cursor != nil && *params.Cursor != "" {
		cursorTime, err := time.Parse(time.RFC3339, *params.Cursor)
		if err != nil {
			return nil, "", fmt.Errorf("invalid cursor format: %w", err)
		}
		query += fmt.Sprintf(` AND "createdAt" < $%d`, argIdx)
		args = append(args, cursorTime)
		argIdx++
	}

	// Ordenação (default: createdAt desc)
	query += ` ORDER BY "createdAt" DESC`
	query += fmt.Sprintf(` LIMIT $%d`, argIdx)
	args = append(args, params.Limit+1) // +1 to check if there's next page

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("query companies: %w", err)
	}
	defer rows.Close()

	companies := make([]domain.Company, 0, params.Limit)
	for rows.Next() {
		var c domain.Company
		var deletedAt sql.NullTime
		err := rows.Scan(
			&c.ID, &c.WorkspaceID, &c.Name, &c.Domain, &c.Industry,
			&c.LifecycleStage, &c.Size,
			&c.Phone, &c.Email, &c.Website, &c.Address,
			&c.Revenue, &c.EmployeeCount,
			&c.OwnerID, &c.Tags, &c.CustomFields, &c.Notes,
			&c.CreatedAt, &c.UpdatedAt, &deletedAt,
		)
		if err != nil {
			return nil, "", fmt.Errorf("scan company: %w", err)
		}
		if deletedAt.Valid {
			c.DeletedAt = &deletedAt.Time
		}
		companies = append(companies, c)
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("iterate companies: %w", err)
	}

	var nextCursor string
	if len(companies) > params.Limit {
		nextCursor = companies[params.Limit-1].CreatedAt.Format(time.RFC3339)
		companies = companies[:params.Limit]
	}

	return companies, nextCursor, nil
}

// Get retrieves a single company by ID, scoped to workspace.
// IDOR protection: returns not found if company exists but belongs to another workspace.
func (r *CompanyRepository) Get(ctx context.Context, workspaceID, companyID uuid.UUID) (*domain.Company, error) {
	query := `
		SELECT id, "workspaceId", name, domain, industry, "lifecycleStage", "size",
		       phone, email, website, address, "revenue", "employeeCount",
		       "ownerId", tags, "customFields", notes,
		       "createdAt", "updatedAt", "deletedAt"
		FROM public."Company"
		WHERE id = $1 AND "workspaceId" = $2 AND "deletedAt" IS NULL
	`

	var c domain.Company
	var deletedAt sql.NullTime
	err := r.pool.QueryRow(ctx, query, companyID, workspaceID).Scan(
		&c.ID, &c.WorkspaceID, &c.Name, &c.Domain, &c.Industry,
		&c.LifecycleStage, &c.Size,
		&c.Phone, &c.Email, &c.Website, &c.Address,
		&c.Revenue, &c.EmployeeCount,
		&c.OwnerID, &c.Tags, &c.CustomFields, &c.Notes,
		&c.CreatedAt, &c.UpdatedAt, &deletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCompanyNotFound
		}
		return nil, fmt.Errorf("query company: %w", err)
	}

	if deletedAt.Valid {
		c.DeletedAt = &deletedAt.Time
	}

	return &c, nil
}

// Create inserts a new company with workspace isolation.
func (r *CompanyRepository) Create(ctx context.Context, company *domain.Company) error {
	query := `
		INSERT INTO public."Company" (
			id, "workspaceId", name, domain, industry, "lifecycleStage", "size",
			phone, email, website, address, "revenue", "employeeCount",
			"ownerId", tags, "customFields", notes
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`

	_, err := r.pool.Exec(ctx, query,
		company.ID, company.WorkspaceID, company.Name, company.Domain, company.Industry,
		company.LifecycleStage, company.Size,
		company.Phone, company.Email, company.Website, company.Address,
		company.Revenue, company.EmployeeCount,
		company.OwnerID, company.Tags, company.CustomFields, company.Notes,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" { // unique_violation
				if pgErr.ConstraintName == "unique_domain_per_workspace" {
					return ErrCompanyDomainConflict
				}
			}
		}
		return fmt.Errorf("insert company: %w", err)
	}

	return nil
}

// Update atualiza campos de uma empresa (PATCH semântico).
func (r *CompanyRepository) Update(ctx context.Context, workspaceID, companyID uuid.UUID, req *domain.UpdateCompanyRequest) error {
	// Dynamic query builder para PATCH semântico
	query := `UPDATE public."Company" SET "updatedAt" = NOW()`
	args := []interface{}{}
	argIdx := 1

	if req.Name != nil {
		query += fmt.Sprintf(`, name = $%d`, argIdx)
		args = append(args, *req.Name)
		argIdx++
	}

	if req.Domain != nil {
		query += fmt.Sprintf(`, domain = $%d`, argIdx)
		args = append(args, *req.Domain)
		argIdx++
	}

	if req.Industry != nil {
		query += fmt.Sprintf(`, industry = $%d`, argIdx)
		args = append(args, *req.Industry)
		argIdx++
	}

	if req.LifecycleStage != nil {
		query += fmt.Sprintf(`, "lifecycleStage" = $%d`, argIdx)
		args = append(args, *req.LifecycleStage)
		argIdx++
	}

	if req.Size != nil {
		query += fmt.Sprintf(`, "size" = $%d`, argIdx)
		args = append(args, *req.Size)
		argIdx++
	}

	if req.Phone != nil {
		query += fmt.Sprintf(`, phone = $%d`, argIdx)
		args = append(args, *req.Phone)
		argIdx++
	}

	if req.Email != nil {
		query += fmt.Sprintf(`, email = $%d`, argIdx)
		args = append(args, *req.Email)
		argIdx++
	}

	if req.Website != nil {
		query += fmt.Sprintf(`, website = $%d`, argIdx)
		args = append(args, *req.Website)
		argIdx++
	}

	if req.Address != nil {
		query += fmt.Sprintf(`, address = $%d`, argIdx)
		args = append(args, req.Address)
		argIdx++
	}

	if req.Revenue != nil {
		query += fmt.Sprintf(`, "revenue" = $%d`, argIdx)
		args = append(args, *req.Revenue)
		argIdx++
	}

	if req.EmployeeCount != nil {
		query += fmt.Sprintf(`, "employeeCount" = $%d`, argIdx)
		args = append(args, *req.EmployeeCount)
		argIdx++
	}

	if req.OwnerID != nil {
		query += fmt.Sprintf(`, "ownerId" = $%d`, argIdx)
		args = append(args, *req.OwnerID)
		argIdx++
	}

	if req.Tags != nil {
		query += fmt.Sprintf(`, tags = $%d`, argIdx)
		args = append(args, *req.Tags)
		argIdx++
	}

	if req.CustomFields != nil {
		query += fmt.Sprintf(`, "customFields" = $%d`, argIdx)
		args = append(args, req.CustomFields)
		argIdx++
	}

	if req.Notes != nil {
		query += fmt.Sprintf(`, notes = $%d`, argIdx)
		args = append(args, *req.Notes)
		argIdx++
	}

	// WHERE clause com multi-tenant isolation
	query += fmt.Sprintf(` WHERE id = $%d AND "workspaceId" = $%d AND "deletedAt" IS NULL`, argIdx, argIdx+1)
	args = append(args, companyID, workspaceID)

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" && pgErr.ConstraintName == "unique_domain_per_workspace" {
				return ErrCompanyDomainConflict
			}
		}
		return fmt.Errorf("update company: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrCompanyNotFound
	}

	return nil
}

// SoftDelete marca uma empresa como deletada (soft delete).
func (r *CompanyRepository) SoftDelete(ctx context.Context, workspaceID, companyID uuid.UUID) error {
	query := `
		UPDATE public."Company"
		SET "deletedAt" = NOW(), "updatedAt" = NOW()
		WHERE id = $1 AND "workspaceId" = $2 AND "deletedAt" IS NULL
	`

	result, err := r.pool.Exec(ctx, query, companyID, workspaceID)
	if err != nil {
		return fmt.Errorf("soft delete company: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrCompanyNotFound
	}

	return nil
}

// ExistsInWorkspace verifica se uma empresa existe no workspace.
// Usado para validação de Contact.CompanyID.
func (r *CompanyRepository) ExistsInWorkspace(ctx context.Context, workspaceID, companyID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM public."Company"
			WHERE id = $1 AND "workspaceId" = $2 AND "deletedAt" IS NULL
		)
	`

	var exists bool
	err := r.pool.QueryRow(ctx, query, companyID, workspaceID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check company existence: %w", err)
	}

	return exists, nil
}
