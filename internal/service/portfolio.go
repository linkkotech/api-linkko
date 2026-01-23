package service

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"

	"linkko-api/internal/domain"
	"linkko-api/internal/observability/logger"
	"linkko-api/internal/repo"

	"go.uber.org/zap"
)

type PortfolioService struct {
	portfolioRepo *repo.PortfolioRepository
	workspaceRepo *repo.WorkspaceRepository
	auditRepo     *repo.AuditRepo
	log           *logger.Logger
}

func NewPortfolioService(portfolioRepo *repo.PortfolioRepository, workspaceRepo *repo.WorkspaceRepository, auditRepo *repo.AuditRepo, log *logger.Logger) *PortfolioService {
	return &PortfolioService{
		portfolioRepo: portfolioRepo,
		workspaceRepo: workspaceRepo,
		auditRepo:     auditRepo,
		log:           log,
	}
}

// getMemberRoleWithLogging wraps GetMemberRole with authorization audit logging.
func (s *PortfolioService) getMemberRoleWithLogging(ctx context.Context, actorID, workspaceID string) (domain.Role, error) {
	role, err := s.workspaceRepo.GetMemberRole(ctx, actorID, workspaceID)
	if err != nil {
		s.log.Error(ctx, "failed to get member role",
			logger.Module("portfolio"),
			logger.Action("authorization"),
			zap.String("actor_id", actorID),
			zap.String("workspace_id", workspaceID),
			zap.Error(err),
		)
		return "", fmt.Errorf("get member role: %w", err)
	}

	s.log.Info(ctx, "workspace access granted",
		logger.Module("portfolio"),
		logger.Action("authorization"),
		zap.String("actor_id", actorID),
		zap.String("workspace_id", workspaceID),
		zap.String("role", string(role)),
	)
	return role, nil
}

func (s *PortfolioService) CreatePortfolioItem(ctx context.Context, workspaceID, actorID string, req *domain.CreatePortfolioItemRequest) (*domain.PortfolioItem, error) {
	// RBAC
	role, err := s.getMemberRoleWithLogging(ctx, actorID, workspaceID)
	if err != nil {
		return nil, err
	}
	if !domain.CanModifyContacts(role) { // Using same permission level as Contacts for now
		return nil, ErrUnauthorized
	}

	// Business Logic: Context Validation
	if err := domain.ValidatePortfolioContext(req.Category, req.Vertical); err != nil {
		return nil, err
	}

	item := &domain.PortfolioItem{
		ID:          generatePortfolioID(),
		WorkspaceID: workspaceID,
		Name:        req.Name,
		Description: req.Description,
		SKU:         req.SKU,
		Category:    req.Category,
		Vertical:    req.Vertical,
		Status:      req.Status,
		Visibility:  req.Visibility,
		BasePrice:   req.BasePrice,
		Currency:    req.Currency,
		ImageURL:    req.ImageURL,
		Metadata:    req.Metadata,
		Tags:        req.Tags,
		CreatedByID: actorID,
	}

	if item.Status == "" {
		item.Status = domain.PortfolioStatusActive
	}
	if item.Visibility == "" {
		item.Visibility = domain.PortfolioVisibilityPublic
	}
	if item.Currency == "" {
		item.Currency = "BRL"
	}

	created, err := s.portfolioRepo.Create(ctx, item)
	if err != nil {
		return nil, err
	}

	// Audit
	s.logPortfolioAction(ctx, workspaceID, actorID, "create", created.ID)

	return created, nil
}

func (s *PortfolioService) GetPortfolioItem(ctx context.Context, workspaceID, itemID, actorID string) (*domain.PortfolioItem, error) {
	role, err := s.getMemberRoleWithLogging(ctx, actorID, workspaceID)
	if err != nil {
		return nil, err
	}
	if !domain.IsWorkspaceMember(role) {
		return nil, ErrUnauthorized
	}

	return s.portfolioRepo.Get(ctx, workspaceID, itemID)
}

func (s *PortfolioService) ListPortfolioItems(ctx context.Context, workspaceID, actorID string, status *domain.PortfolioStatus, category *domain.PortfolioCategoryEnum, query *string) ([]domain.PortfolioItem, error) {
	role, err := s.getMemberRoleWithLogging(ctx, actorID, workspaceID)
	if err != nil {
		return nil, err
	}
	if !domain.IsWorkspaceMember(role) {
		return nil, ErrUnauthorized
	}

	return s.portfolioRepo.List(ctx, workspaceID, status, category, query)
}

func (s *PortfolioService) UpdatePortfolioItem(ctx context.Context, workspaceID, itemID, actorID string, req *domain.UpdatePortfolioItemRequest) (*domain.PortfolioItem, error) {
	role, err := s.getMemberRoleWithLogging(ctx, actorID, workspaceID)
	if err != nil {
		return nil, err
	}
	if !domain.CanModifyContacts(role) {
		return nil, ErrUnauthorized
	}

	// If updating cat/vert, validate again
	if req.Category != nil || req.Vertical != nil {
		// We'd need current state or full validation logic here
	}

	updated, err := s.portfolioRepo.Update(ctx, workspaceID, itemID, req, actorID)
	if err != nil {
		return nil, err
	}

	s.logPortfolioAction(ctx, workspaceID, actorID, "update", itemID)

	return updated, nil
}

func (s *PortfolioService) DeletePortfolioItem(ctx context.Context, workspaceID, itemID, actorID string) error {
	role, err := s.getMemberRoleWithLogging(ctx, actorID, workspaceID)
	if err != nil {
		return err
	}
	if !domain.CanModifyContacts(role) {
		return ErrUnauthorized
	}

	if err := s.portfolioRepo.Delete(ctx, workspaceID, itemID); err != nil {
		return err
	}

	s.logPortfolioAction(ctx, workspaceID, actorID, "delete", itemID)
	return nil
}

// Helpers
func generatePortfolioID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return "pit_" + strings.ToLower(base32.StdEncoding.EncodeToString(b)[:24])
}

func (s *PortfolioService) logPortfolioAction(ctx context.Context, workspaceID, actorID, action, itemID string) {
	idStr := itemID
	_ = s.auditRepo.LogAction(ctx, workspaceID, actorID, action, "portfolio_item", &idStr, nil, "", "")
}
