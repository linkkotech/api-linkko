package repo

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"linkko-api/internal/domain"
	"linkko-api/internal/repo/sqlc"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrCompanyNotFound       = errors.New("company not found in workspace")
	ErrCompanyDomainConflict = errors.New("company with this domain already exists in workspace")
)

type CompanyRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewCompanyRepository(pool *pgxpool.Pool) *CompanyRepository {
	return &CompanyRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// List retrieves companies for a workspace with optional filters.
func (r *CompanyRepository) List(ctx context.Context, params domain.ListCompaniesParams) ([]domain.Company, string, error) {
	// Prepare SQLc params
	sqlcParams := sqlc.ListCompaniesParams{
		WorkspaceId: params.WorkspaceID,
		Column2:     "",
		Column3:     "",
		Column4:     "",
		Column5:     "",
		Limit:       int32(params.Limit + 1), // +1 to check next page
	}

	if params.LifecycleStage != nil {
		stage := string(*params.LifecycleStage)
		sqlcParams.Column2 = stage
	}

	if params.Size != nil {
		size := string(*params.Size)
		sqlcParams.Column3 = size
	}

	if params.OwnerID != nil {
		sqlcParams.Column4 = *params.OwnerID
	}

	if params.Query != nil {
		sqlcParams.Column5 = *params.Query
	}

	if params.Cursor != nil && *params.Cursor != "" {
		// Parse cursor as timestamp
		// TODO: Parse cursor properly
	}

	rows, err := r.queries.ListCompanies(ctx, sqlcParams)
	if err != nil {
		return nil, "", err
	}

	companies := make([]domain.Company, 0, params.Limit)
	for _, row := range rows {
		companies = append(companies, sqlcRowToDomainCompany(row))
	}

	var nextCursor string
	if len(companies) > params.Limit {
		nextCursor = companies[params.Limit-1].CreatedAt.Format("2006-01-02T15:04:05Z07:00")
		companies = companies[:params.Limit]
	}

	return companies, nextCursor, nil
}

// Get retrieves a single company by ID, scoped to workspace.
// IDOR protection: returns not found if company exists but belongs to another workspace.
func (r *CompanyRepository) Get(ctx context.Context, workspaceID, companyID string) (*domain.Company, error) {
	row, err := r.queries.GetCompany(ctx, sqlc.GetCompanyParams{
		ID:          companyID,
		WorkspaceId: workspaceID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCompanyNotFound
		}
		return nil, err
	}

	company := sqlcRowToDomainCompany(row)
	return &company, nil
}

// Create inserts a new company with workspace isolation.
func (r *CompanyRepository) Create(ctx context.Context, company *domain.Company) error {
	now := pgtype.Timestamp{Time: time.Now(), Valid: true}

	// Convert domain ENUMs to SQLc ENUMs
	var size sqlc.NullCompanySize
	if company.Size.IsValid() {
		size = sqlc.NullCompanySize{
			CompanySize: sqlc.CompanySize(company.Size),
			Valid:       true,
		}
	}

	// Marshal JSONB fields
	socialUrls, _ := json.Marshal([]string{})
	businessHours, _ := json.Marshal(map[string]interface{}{})
	supportHours, _ := json.Marshal(map[string]interface{}{})

	_, err := r.queries.CreateCompany(ctx, sqlc.CreateCompanyParams{
		ID:             company.ID,
		WorkspaceId:    company.WorkspaceID,
		Name:           company.Name,
		Website:        company.Domain,
		Linkedin:       nil,
		LegalName:      nil,
		Phone:          company.Phone,
		Instagram:      nil,
		PolicyUrl:      nil,
		SocialUrls:     socialUrls,
		AddressLine:    nil,
		City:           nil,
		State:          nil,
		Country:        nil,
		Timezone:       nil,
		Currency:       nil,
		Locale:         nil,
		BusinessHours:  businessHours,
		SupportHours:   supportHours,
		Size:           size,
		Revenue:        company.AnnualRevenue,
		CompanyScore:   0,
		LifecycleStage: sqlc.CompanyLifecycleStage(company.LifecycleStage),
		AssignedToId:   &company.OwnerID,
		CreatedById:    &company.OwnerID,
		UpdatedById:    &company.OwnerID,
		CreatedAt:      now,
		UpdatedAt:      now,
	})

	return err
}

// Update atualiza campos de uma empresa (PATCH semântico).
func (r *CompanyRepository) Update(ctx context.Context, workspaceID, companyID string, req *domain.UpdateCompanyRequest) error {
	// SQLc UpdateCompany usa COALESCE, então precisamos passar valores atuais ou novos
	// Primeiro, buscamos a empresa atual
	current, err := r.Get(ctx, workspaceID, companyID)
	if err != nil {
		return err
	}

	now := pgtype.Timestamp{Time: time.Now(), Valid: true}

	// Merge: use req.* se não for nil, senão use current.*
	name := current.Name
	if req.Name != nil {
		name = *req.Name
	}

	website := current.Domain
	if req.Domain != nil {
		website = req.Domain
	}

	phone := current.Phone
	if req.Phone != nil {
		phone = req.Phone
	}

	lifecycleStage := current.LifecycleStage
	if req.LifecycleStage != nil {
		lifecycleStage = *req.LifecycleStage
	}

	size := sqlc.NullCompanySize{
		CompanySize: sqlc.CompanySize(current.Size),
		Valid:       current.Size.IsValid(),
	}
	if req.CompanySize != nil {
		size = sqlc.NullCompanySize{
			CompanySize: sqlc.CompanySize(*req.CompanySize),
			Valid:       true,
		}
	}

	revenue := current.AnnualRevenue
	if req.AnnualRevenue != nil {
		revenue = req.AnnualRevenue
	}

	assignedToId := &current.OwnerID
	if req.OwnerID != nil {
		assignedToId = req.OwnerID
	}

	// Campos JSONB fixos
	socialUrls, _ := json.Marshal([]string{})
	businessHours, _ := json.Marshal(map[string]interface{}{})
	supportHours, _ := json.Marshal(map[string]interface{}{})

	result, err := r.queries.UpdateCompany(ctx, sqlc.UpdateCompanyParams{
		ID:             companyID,
		WorkspaceId:    workspaceID,
		Name:           name,
		Website:        website,
		Linkedin:       nil,
		LegalName:      nil,
		Phone:          phone,
		Instagram:      nil,
		PolicyUrl:      nil,
		SocialUrls:     socialUrls,
		AddressLine:    nil,
		City:           nil,
		State:          nil,
		Country:        nil,
		Timezone:       nil,
		Currency:       nil,
		Locale:         nil,
		BusinessHours:  businessHours,
		SupportHours:   supportHours,
		Size:           size,
		Revenue:        revenue,
		CompanyScore:   0,
		LifecycleStage: sqlc.CompanyLifecycleStage(lifecycleStage),
		AssignedToId:   assignedToId,
		UpdatedById:    assignedToId,
		UpdatedAt:      now,
		UpdatedAt_2:    pgtype.Timestamp{Time: current.UpdatedAt, Valid: true}, // optimistic lock
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrCompanyNotFound
		}
		return err
	}

	// Verificar se houve atualização (optimistic lock pode falhar)
	if result.ID == "" {
		return ErrCompanyNotFound
	}

	return nil
}

// SoftDelete marca uma empresa como deletada (soft delete).
func (r *CompanyRepository) SoftDelete(ctx context.Context, workspaceID, companyID string) error {
	now := pgtype.Timestamp{Time: time.Now(), Valid: true}

	err := r.queries.SoftDeleteCompany(ctx, sqlc.SoftDeleteCompanyParams{
		ID:          companyID,
		WorkspaceId: workspaceID,
		DeletedAt:   now,
		DeletedById: nil, // TODO: passar userID do context
	})

	return err
}

// ExistsInWorkspace verifica se uma empresa existe no workspace.
// Usado para validação de Contact.CompanyID.
func (r *CompanyRepository) ExistsInWorkspace(ctx context.Context, workspaceID, companyID string) (bool, error) {
	return r.queries.CompanyExistsInWorkspace(ctx, sqlc.CompanyExistsInWorkspaceParams{
		ID:          companyID,
		WorkspaceId: workspaceID,
	})
}

// sqlcRowToDomainCompany converte um row SQLc para domain.Company
func sqlcRowToDomainCompany(row interface{}) domain.Company {
	var c domain.Company

	switch r := row.(type) {
	case sqlc.GetCompanyRow:
		c.ID = r.ID
		c.WorkspaceID = r.WorkspaceId
		c.Name = r.Name
		c.Domain = r.Website
		c.LifecycleStage = domain.CompanyLifecycleStage(r.LifecycleStage)
		c.Phone = r.Phone
		c.Website = r.Website
		c.AnnualRevenue = r.Revenue
		c.Tags = []string{}
		c.CustomFields = map[string]interface{}{}
		c.Address = map[string]interface{}{}

		// Convert ENUMs
		if r.Size.Valid {
			c.Size = domain.CompanySize(r.Size.CompanySize)
			c.CompanySize = domain.CompanySize(r.Size.CompanySize)
		}

		// Convert OwnerID
		if r.AssignedToId != nil {
			c.OwnerID = *r.AssignedToId
		}

		// Convert timestamps
		if r.CreatedAt.Valid {
			c.CreatedAt = r.CreatedAt.Time
		}
		if r.UpdatedAt.Valid {
			c.UpdatedAt = r.UpdatedAt.Time
		}
		if r.DeletedAt.Valid {
			deletedAt := r.DeletedAt.Time
			c.DeletedAt = &deletedAt
		}

	case sqlc.ListCompaniesRow:
		c.ID = r.ID
		c.WorkspaceID = r.WorkspaceId
		c.Name = r.Name
		c.Domain = r.Website
		c.LifecycleStage = domain.CompanyLifecycleStage(r.LifecycleStage)
		c.Phone = r.Phone
		c.Website = r.Website
		c.AnnualRevenue = r.Revenue
		c.Tags = []string{}
		c.CustomFields = map[string]interface{}{}
		c.Address = map[string]interface{}{}

		// Convert ENUMs
		if r.Size.Valid {
			c.Size = domain.CompanySize(r.Size.CompanySize)
			c.CompanySize = domain.CompanySize(r.Size.CompanySize)
		}

		// Convert OwnerID
		if r.AssignedToId != nil {
			c.OwnerID = *r.AssignedToId
		}

		// Convert timestamps
		if r.CreatedAt.Valid {
			c.CreatedAt = r.CreatedAt.Time
		}
		if r.UpdatedAt.Valid {
			c.UpdatedAt = r.UpdatedAt.Time
		}
		if r.DeletedAt.Valid {
			deletedAt := r.DeletedAt.Time
			c.DeletedAt = &deletedAt
		}
	}

	return c
}
