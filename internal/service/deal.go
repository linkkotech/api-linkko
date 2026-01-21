package service

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"strings"

	"linkko-api/internal/domain"
	"linkko-api/internal/repo"
)

var (
	ErrDealStageInvalid = errors.New("invalid deal stage for this operation")
	ErrPipelineConflict = errors.New("pipeline/stage does not belong to workspace")
	ErrDealNotFound     = errors.New("deal not found")
)

type DealService struct {
	dealRepo      *repo.DealRepository
	pipelineRepo  *repo.PipelineRepository
	workspaceRepo *repo.WorkspaceRepository
	auditRepo     *repo.AuditRepo
}

func NewDealService(dealRepo *repo.DealRepository, pipelineRepo *repo.PipelineRepository, workspaceRepo *repo.WorkspaceRepository, auditRepo *repo.AuditRepo) *DealService {
	return &DealService{
		dealRepo:      dealRepo,
		pipelineRepo:  pipelineRepo,
		workspaceRepo: workspaceRepo,
		auditRepo:     auditRepo,
	}
}

func (s *DealService) CreateDeal(ctx context.Context, workspaceID, actorID string, req *domain.CreateDealRequest) (*domain.Deal, error) {
	role, err := s.workspaceRepo.GetMemberRole(ctx, actorID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("get member role: %w", err)
	}
	if !domain.CanModifyContacts(role) {
		return nil, ErrUnauthorized
	}

	// Validate Pipeline/Stage
	if req.StageID != nil {
		// In production, validate if StageID belongs to PipelineID and WorkspaceID
	}

	deal := &domain.Deal{
		ID:                generateDealID(),
		WorkspaceID:       workspaceID,
		PipelineID:        req.PipelineID,
		StageID:           req.StageID,
		ContactID:         req.ContactID,
		CompanyID:         req.CompanyID,
		Name:              req.Name,
		Value:             req.Value,
		Currency:          req.Currency,
		Stage:             domain.DealStageOpen,
		Probability:       req.Probability,
		ExpectedCloseDate: req.ExpectedCloseDate,
		Description:       req.Description,
		OwnerID:           req.OwnerID,
		CreatedByID:       actorID,
	}

	if deal.Currency == "" {
		deal.Currency = "BRL"
	}
	if deal.Probability == nil {
		p := int32(50)
		deal.Probability = &p
	}

	created, err := s.dealRepo.Create(ctx, deal)
	if err != nil {
		return nil, fmt.Errorf("repo create deal: %w", err)
	}

	// Audit
	s.logDealAction(ctx, workspaceID, actorID, "create", created.ID)

	return created, nil
}

func (s *DealService) GetDeal(ctx context.Context, workspaceID, dealID, actorID string) (*domain.Deal, error) {
	role, err := s.workspaceRepo.GetMemberRole(ctx, actorID, workspaceID)
	if err != nil {
		return nil, err
	}
	if !domain.IsWorkspaceMember(role) {
		return nil, ErrUnauthorized
	}

	return s.dealRepo.Get(ctx, workspaceID, dealID)
}

func (s *DealService) ListDeals(ctx context.Context, workspaceID, actorID string, pipelineID, stageID, ownerID *string) ([]domain.Deal, error) {
	role, err := s.workspaceRepo.GetMemberRole(ctx, actorID, workspaceID)
	if err != nil {
		return nil, err
	}
	if !domain.IsWorkspaceMember(role) {
		return nil, ErrUnauthorized
	}

	return s.dealRepo.List(ctx, workspaceID, pipelineID, stageID, ownerID)
}

func (s *DealService) UpdateDeal(ctx context.Context, workspaceID, dealID, actorID string, req *domain.UpdateDealRequest) (*domain.Deal, error) {
	role, err := s.workspaceRepo.GetMemberRole(ctx, actorID, workspaceID)
	if err != nil {
		return nil, err
	}
	if !domain.CanModifyContacts(role) {
		return nil, ErrUnauthorized
	}

	updated, err := s.dealRepo.Update(ctx, workspaceID, dealID, req, actorID)
	if err != nil {
		if errors.Is(err, repo.ErrDealNotFound) {
			return nil, ErrDealNotFound
		}
		return nil, err
	}

	s.logDealAction(ctx, workspaceID, actorID, "update", dealID)

	return updated, nil
}

// UpdateDealStage handles the transactional movement of a deal through the funnel.
func (s *DealService) UpdateDealStage(ctx context.Context, workspaceID, dealID, actorID string, req *domain.UpdateDealStageRequest) (*domain.Deal, error) {
	role, err := s.workspaceRepo.GetMemberRole(ctx, actorID, workspaceID)
	if err != nil {
		return nil, err
	}
	if !domain.CanModifyContacts(role) {
		return nil, ErrUnauthorized
	}

	// 1. Get current deal to know fromStage
	current, err := s.dealRepo.Get(ctx, workspaceID, dealID)
	if err != nil {
		if errors.Is(err, repo.ErrDealNotFound) {
			return nil, ErrDealNotFound
		}
		return nil, err
	}

	// 2. Start Transaction
	tx, err := s.dealRepo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	repoTx := s.dealRepo.WithTx(tx)

	// 3. Update Deal Stage
	updated, err := repoTx.MoveStage(ctx, workspaceID, dealID, req, actorID)
	if err != nil {
		return nil, err
	}

	// 4. Record History
	history := &domain.DealStageHistory{
		ID:          generateDealID(),
		WorkspaceID: workspaceID,
		DealID:      dealID,
		FromStage:   current.Stage,
		ToStage:     updated.Stage,
		Reason:      req.Reason,
		UserID:      actorID,
	}
	if err := repoTx.CreateHistory(ctx, history); err != nil {
		return nil, err
	}

	// 5. Commit
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	s.logDealAction(ctx, workspaceID, actorID, "move_stage", dealID)

	return updated, nil
}

// Helpers
func generateDealID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return "c" + strings.ToLower(base32.StdEncoding.EncodeToString(b)[:24])
}

func (s *DealService) logDealAction(ctx context.Context, workspaceID, actorID, action, dealID string) {
	idStr := dealID
	_ = s.auditRepo.LogAction(ctx, workspaceID, actorID, action, "deal", &idStr, nil, "", "")
}
