package service

import (
	"context"
	"errors"
	"fmt"

	"linkko-api/internal/domain"
	"linkko-api/internal/repo"
)

var (
	ErrCompanyNotFound       = repo.ErrCompanyNotFound
	ErrCompanyDomainConflict = repo.ErrCompanyDomainConflict
)

type CompanyService struct {
	companyRepo   *repo.CompanyRepository
	auditRepo     *repo.AuditRepo
	workspaceRepo *repo.WorkspaceRepository
}

func NewCompanyService(companyRepo *repo.CompanyRepository, auditRepo *repo.AuditRepo, workspaceRepo *repo.WorkspaceRepository) *CompanyService {
	return &CompanyService{
		companyRepo:   companyRepo,
		auditRepo:     auditRepo,
		workspaceRepo: workspaceRepo,
	}
}

// ListCompanies retrieves companies with RBAC validation.
// Permission: all workspace members can list companies.
// Role is fetched from database to enforce real-time authorization.
func (s *CompanyService) ListCompanies(ctx context.Context, workspaceID, actorID string, params domain.ListCompaniesParams) (*domain.CompanyListResponse, error) {
	// Fetch user's role in this workspace from database
	role, err := s.workspaceRepo.GetMemberRole(ctx, actorID, workspaceID)
	if err != nil {
		if errors.Is(err, repo.ErrMemberNotFound) {
			return nil, ErrMemberNotFound
		}
		return nil, fmt.Errorf("get member role: %w", err)
	}

	// RBAC: all workspace members (admin, manager, user, viewer) can list companies
	if !domain.IsWorkspaceMember(role) {
		return nil, ErrUnauthorized
	}

	params.WorkspaceID = workspaceID

	companies, nextCursor, err := s.companyRepo.List(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list companies: %w", err)
	}

	response := &domain.CompanyListResponse{
		Data: companies,
	}
	response.Meta.HasNextPage = nextCursor != ""
	if nextCursor != "" {
		response.Meta.NextCursor = &nextCursor
	}
	return response, nil
}

// GetCompany retrieves a single company with RBAC validation.
// Permission: all workspace members can view companies.
// Role is fetched from database to enforce real-time authorization.
func (s *CompanyService) GetCompany(ctx context.Context, workspaceID, companyID, actorID string) (*domain.Company, error) {
	// Fetch user's role in this workspace from database
	role, err := s.workspaceRepo.GetMemberRole(ctx, actorID, workspaceID)
	if err != nil {
		if errors.Is(err, repo.ErrMemberNotFound) {
			return nil, ErrMemberNotFound
		}
		return nil, fmt.Errorf("get member role: %w", err)
	}

	// RBAC: all workspace members can view companies
	if !domain.IsWorkspaceMember(role) {
		return nil, ErrUnauthorized
	}

	company, err := s.companyRepo.Get(ctx, workspaceID, companyID)
	if err != nil {
		return nil, fmt.Errorf("get company: %w", err)
	}

	return company, nil
}

// CreateCompany creates a new company with RBAC and business validation.
// Permission: admin, manager, user can create companies. Viewer cannot.
// Role is fetched from database to enforce real-time authorization.
func (s *CompanyService) CreateCompany(ctx context.Context, workspaceID, actorID string, req *domain.CreateCompanyRequest) (*domain.Company, error) {
	// Fetch user's role in this workspace from database
	role, err := s.workspaceRepo.GetMemberRole(ctx, actorID, workspaceID)
	if err != nil {
		if errors.Is(err, repo.ErrMemberNotFound) {
			return nil, ErrMemberNotFound
		}
		return nil, fmt.Errorf("get member role: %w", err)
	}

	// RBAC: admin, manager, user can create (viewer cannot)
	if !domain.CanModifyContacts(role) { // Reusing permission for companies
		return nil, ErrUnauthorized
	}

	company := &domain.Company{
		ID:             generateID(),
		WorkspaceID:    workspaceID,
		Name:           req.Name,
		LifecycleStage: *req.LifecycleStage,
		Size:           *req.CompanySize,
		OwnerID:        actorID, // Default: creator is owner
	}

	// Optional fields
	if req.Domain != nil {
		company.Domain = req.Domain
	}
	if req.Industry != nil {
		company.Industry = req.Industry
	}
	if req.OwnerID != nil {
		company.OwnerID = *req.OwnerID
	}
	if req.Website != nil {
		company.Website = req.Website
	}
	if req.Phone != nil {
		company.Phone = req.Phone
	}
	if req.AnnualRevenue != nil {
		company.AnnualRevenue = req.AnnualRevenue
	}
	if req.EmployeeCount != nil {
		company.EmployeeCount = req.EmployeeCount
	}
	if req.Address != nil {
		company.Address = req.Address
	}
	if req.Tags != nil {
		company.Tags = req.Tags
	} else {
		company.Tags = []string{}
	}
	if req.CustomFields != nil {
		company.CustomFields = req.CustomFields
	} else {
		company.CustomFields = make(map[string]interface{})
	}

	err = s.companyRepo.Create(ctx, company)
	if err != nil {
		return nil, fmt.Errorf("create company: %w", err)
	}

	// Audit: log company creation
	companyIDStr := company.ID
	auditErr := s.auditRepo.LogAction(
		ctx,
		workspaceID,
		actorID,
		"create",
		"company",
		&companyIDStr,
		nil,
		"",
		"",
	)
	if auditErr != nil {
		// Log audit failure but don't fail the operation
	}

	return company, nil
}

// UpdateCompany updates a company with RBAC and business validation.
// Permission: admin, manager, user can update. Viewer cannot.
// Role is fetched from database to enforce real-time authorization.
func (s *CompanyService) UpdateCompany(ctx context.Context, workspaceID, companyID, actorID string, req *domain.UpdateCompanyRequest) (*domain.Company, error) {
	// Fetch user's role in this workspace from database
	role, err := s.workspaceRepo.GetMemberRole(ctx, actorID, workspaceID)
	if err != nil {
		if errors.Is(err, repo.ErrMemberNotFound) {
			return nil, ErrMemberNotFound
		}
		return nil, fmt.Errorf("get member role: %w", err)
	}

	// RBAC: admin, manager, user can update (viewer cannot)
	if !domain.CanModifyContacts(role) { // Reusing permission for companies
		return nil, ErrUnauthorized
	}

	// Verify company exists before update
	_, err = s.companyRepo.Get(ctx, workspaceID, companyID)
	if err != nil {
		return nil, fmt.Errorf("get company: %w", err)
	}

	err = s.companyRepo.Update(ctx, workspaceID, companyID, req)
	if err != nil {
		return nil, fmt.Errorf("update company: %w", err)
	}

	// Fetch updated company
	company, err := s.companyRepo.Get(ctx, workspaceID, companyID)
	if err != nil {
		return nil, fmt.Errorf("get updated company: %w", err)
	}

	// Audit: log company update
	companyIDStr := companyID
	auditErr := s.auditRepo.LogAction(
		ctx,
		workspaceID,
		actorID,
		"update",
		"company",
		&companyIDStr,
		nil,
		"",
		"",
	)
	if auditErr != nil {
		// Log audit failure but don't fail the operation
	}

	return company, nil
}

// DeleteCompany soft deletes a company with RBAC validation.
// Permission: only admin and manager can delete companies.
// Role is fetched from database to enforce real-time authorization.
func (s *CompanyService) DeleteCompany(ctx context.Context, workspaceID, companyID, actorID string) error {
	// Fetch user's role in this workspace from database
	role, err := s.workspaceRepo.GetMemberRole(ctx, actorID, workspaceID)
	if err != nil {
		if errors.Is(err, repo.ErrMemberNotFound) {
			return ErrMemberNotFound
		}
		return fmt.Errorf("get member role: %w", err)
	}

	// RBAC: only admin and manager can delete
	if !domain.CanDeleteContacts(role) { // Reusing permission for companies
		return ErrUnauthorized
	}

	err = s.companyRepo.SoftDelete(ctx, workspaceID, companyID)
	if err != nil {
		return fmt.Errorf("delete company: %w", err)
	}

	// Audit: log company deletion
	companyIDStr := companyID
	auditErr := s.auditRepo.LogAction(
		ctx,
		workspaceID,
		actorID,
		"delete",
		"company",
		&companyIDStr,
		nil,
		"",
		"",
	)
	if auditErr != nil {
		// Log audit failure but don't fail the operation
	}

	return nil
}
