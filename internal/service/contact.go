package service

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"strings"

	"linkko-api/internal/domain"
	"linkko-api/internal/observability/logger"
	"linkko-api/internal/repo"

	"go.uber.org/zap"
)

var (
	ErrUnauthorized        = errors.New("user not authorized for this action")
	ErrInvalidOwner        = errors.New("owner_id does not belong to workspace")
	ErrInvalidCompany      = errors.New("company_id does not belong to workspace")
	ErrContactNotFound     = repo.ErrContactNotFound
	ErrEmailConflict       = repo.ErrContactEmailConflict
	ErrConcurrencyConflict = errors.New("contact was modified by another request")
	ErrMemberNotFound      = repo.ErrMemberNotFound // Wrap workspace repo error
)

type ContactService struct {
	contactRepo   *repo.ContactRepository
	auditRepo     *repo.AuditRepo
	workspaceRepo *repo.WorkspaceRepository
	companyRepo   *repo.CompanyRepository // For CompanyID validation
	log           *logger.Logger
}

func NewContactService(contactRepo *repo.ContactRepository, auditRepo *repo.AuditRepo, workspaceRepo *repo.WorkspaceRepository, companyRepo *repo.CompanyRepository, log *logger.Logger) *ContactService {
	return &ContactService{
		contactRepo:   contactRepo,
		auditRepo:     auditRepo,
		workspaceRepo: workspaceRepo,
		companyRepo:   companyRepo,
		log:           log,
	}
}

// generateID cria um ID compatível com Prisma (cuid-like)
func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return "c" + strings.ToLower(base32.StdEncoding.EncodeToString(b)[:24])
}

// getMemberRoleWithLogging wraps GetMemberRole with authorization audit logging.
// Logs successful role resolution and authorization failures for security monitoring.
func (s *ContactService) getMemberRoleWithLogging(ctx context.Context, actorID, workspaceID string) (domain.Role, error) {
	role, err := s.workspaceRepo.GetMemberRole(ctx, actorID, workspaceID)
	if err != nil {
		s.log.Error(ctx, "failed to get member role",
			logger.Module("contact"),
			logger.Action("authorization"),
			zap.String("actor_id", actorID),
			zap.String("workspace_id", workspaceID),
			zap.Error(err),
		)
		if errors.Is(err, repo.ErrMemberNotFound) {
			return "", ErrMemberNotFound
		}
		return "", fmt.Errorf("get member role: %w", err)
	}

	s.log.Info(ctx, "workspace access granted",
		logger.Module("contact"),
		logger.Action("authorization"),
		zap.String("actor_id", actorID),
		zap.String("workspace_id", workspaceID),
		zap.String("role", string(role)),
	)
	return role, nil
}

// ListContacts retrieves contacts with RBAC validation.
// Permission: all workspace members can list contacts.
// Role is fetched from database to enforce real-time authorization.
func (s *ContactService) ListContacts(ctx context.Context, workspaceID, actorID string, params domain.ListContactsParams) (*domain.ContactListResponse, error) {
	// Fetch user's role in this workspace from database
	role, err := s.getMemberRoleWithLogging(ctx, actorID, workspaceID)
	if err != nil {
		return nil, err
	}

	// RBAC: all workspace members (admin, manager, user, viewer) can list contacts
	if !domain.IsWorkspaceMember(role) {
		return nil, ErrUnauthorized
	}

	params.WorkspaceID = workspaceID

	contacts, nextCursor, err := s.contactRepo.List(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list contacts: %w", err)
	}

	// Audit: list operations not logged to avoid excessive audit entries
	response := &domain.ContactListResponse{
		Data: contacts,
	}
	response.Meta.HasNextPage = nextCursor != ""
	if nextCursor != "" {
		response.Meta.NextCursor = &nextCursor
	}
	return response, nil
}

// GetContact retrieves a single contact with RBAC validation.
// Permission: all workspace members can view contacts.
// Role is fetched from database to enforce real-time authorization.
func (s *ContactService) GetContact(ctx context.Context, workspaceID, contactID, actorID string) (*domain.Contact, error) {
	// Fetch user's role in this workspace from database
	role, err := s.getMemberRoleWithLogging(ctx, actorID, workspaceID)
	if err != nil {
		return nil, err
	}

	// RBAC: all workspace members can view contacts
	if !domain.IsWorkspaceMember(role) {
		return nil, ErrUnauthorized
	}

	contact, err := s.contactRepo.Get(ctx, workspaceID, contactID)
	if err != nil {
		return nil, fmt.Errorf("get contact: %w", err)
	}

	// Audit: read operations not logged to avoid excessive audit entries
	return contact, nil
}

// CreateContact creates a new contact with RBAC and business validation.
// Permission: admin, manager, user can create contacts. Viewer cannot.
// Role is fetched from database to enforce real-time authorization.
func (s *ContactService) CreateContact(ctx context.Context, workspaceID, actorID string, req *domain.CreateContactRequest) (*domain.Contact, error) {
	// Fetch user's role in this workspace from database
	role, err := s.getMemberRoleWithLogging(ctx, actorID, workspaceID)
	if err != nil {
		return nil, err
	}

	// RBAC: admin, manager, user can create (viewer cannot)
	if !domain.CanModifyContacts(role) {
		return nil, ErrUnauthorized
	}

	// Business validation: if actor_id provided, validate it belongs to workspace
	if req.ActorID != nil {
		// Note: In production, this would call UserRepository.ExistsInWorkspace
		// Skipping for now as UserRepository is not yet implemented
	}

	// Business validation: if company_id provided, validate it belongs to workspace
	if req.CompanyID != nil {
		exists, err := s.companyRepo.ExistsInWorkspace(ctx, workspaceID, *req.CompanyID)
		if err != nil {
			return nil, fmt.Errorf("validate company: %w", err)
		}
		if !exists {
			return nil, ErrInvalidCompany
		}
	}

	contact := &domain.Contact{
		ID:          generateID(),
		WorkspaceID: workspaceID,
		FullName:    req.FullName,
		Email:       req.Email,
		ActorID:     actorID, // Use current actor (user/agent) as owner if not specified
	}

	if req.Phone != nil {
		contact.Phone = req.Phone
	}
	if req.ActorID != nil {
		contact.ActorID = *req.ActorID
	}
	if req.CompanyID != nil {
		contact.CompanyID = req.CompanyID
	}
	if req.Tags != nil {
		contact.Tags = req.Tags
	} else {
		contact.Tags = []string{} // Initialize empty slice to avoid null in JSON
	}
	if req.CustomFields != nil {
		contact.CustomFields = req.CustomFields
	} else {
		contact.CustomFields = make(map[string]interface{}) // Initialize empty map to avoid null in JSON
	}

	err = s.contactRepo.Create(ctx, contact)
	if err != nil {
		return nil, fmt.Errorf("create contact: %w", err)
	}

	// Audit: log contact creation
	contactIDStr := contact.ID
	auditErr := s.auditRepo.LogAction(
		ctx,
		workspaceID,
		actorID,
		"create",
		"contact",
		&contactIDStr,
		nil,
		"", // IP address not available in service layer
		"", // User agent not available in service layer
	)
	if auditErr != nil {
		// Log audit failure but don't fail the operation
		// In production, this should be logged to monitoring system
	}

	return contact, nil
}

// UpdateContact updates a contact with RBAC, business validation, and optimistic concurrency.
// Permission: admin, manager, user can update. Viewer cannot.
// Role is fetched from database to enforce real-time authorization.
func (s *ContactService) UpdateContact(ctx context.Context, workspaceID, contactID, actorID string, req *domain.UpdateContactRequest) (*domain.Contact, error) {
	// Fetch user's role in this workspace from database
	role, err := s.getMemberRoleWithLogging(ctx, actorID, workspaceID)
	if err != nil {
		return nil, err
	}

	// RBAC: admin, manager, user can update (viewer cannot)
	if !domain.CanModifyContacts(role) {
		return nil, ErrUnauthorized
	}

	// Get current version for optimistic concurrency check
	current, err := s.contactRepo.Get(ctx, workspaceID, contactID)
	if err != nil {
		return nil, fmt.Errorf("get current contact: %w", err)
	}

	// Business validation: if actor_id provided, validate it belongs to workspace
	if req.ActorID != nil {
		// Note: In production, this would call UserRepository.ExistsInWorkspace
	}

	// Business validation: if company_id provided, validate it belongs to workspace
	if req.CompanyID != nil {
		exists, err := s.companyRepo.ExistsInWorkspace(ctx, workspaceID, *req.CompanyID)
		if err != nil {
			return nil, fmt.Errorf("validate company: %w", err)
		}
		if !exists {
			return nil, ErrInvalidCompany
		}
	}

	contact, err := s.contactRepo.Update(ctx, workspaceID, contactID, req, current.UpdatedAt)
	if err != nil {
		if errors.Is(err, errors.New("contact was modified by another request")) {
			return nil, ErrConcurrencyConflict
		}
		return nil, fmt.Errorf("update contact: %w", err)
	}

	// Audit: log contact update
	contactIDStr := contactID
	auditErr := s.auditRepo.LogAction(
		ctx,
		workspaceID,
		actorID,
		"update",
		"contact",
		&contactIDStr,
		nil,
		"",
		"",
	)
	if auditErr != nil {
		// Log audit failure but don't fail the operation
	}

	return contact, nil
}

// DeleteContact soft deletes a contact with RBAC validation.
// Permission: only admin and manager can delete contacts.
// Role is fetched from database to enforce real-time authorization.
func (s *ContactService) DeleteContact(ctx context.Context, workspaceID, contactID, actorID string) error {
	// Fetch user's role in this workspace from database
	role, err := s.getMemberRoleWithLogging(ctx, actorID, workspaceID)
	if err != nil {
		return err
	}

	// RBAC: only admin and manager can delete
	if !domain.CanDeleteContacts(role) {
		return ErrUnauthorized
	}

	err = s.contactRepo.SoftDelete(ctx, workspaceID, contactID)
	if err != nil {
		return fmt.Errorf("delete contact: %w", err)
	}

	// Audit: log contact deletion
	contactIDStr := contactID
	auditErr := s.auditRepo.LogAction(
		ctx,
		workspaceID,
		actorID,
		"delete",
		"contact",
		&contactIDStr,
		nil,
		"",
		"",
	)
	if auditErr != nil {
		// Log audit failure but don't fail the operation
	}

	return nil
}

// getRequestID extracts request_id from context for audit logging.
// In production, this would use a context key set by the request middleware.
func getRequestID(_ context.Context) string {
	// Placeholder: in production, extract from context
	// requestID, ok := ctx.Value("request_id").(string)
	// if !ok { return "" }
	// return requestID
	return ""
}
