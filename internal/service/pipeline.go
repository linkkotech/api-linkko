package service

import (
	"context"
	"errors"
	"fmt"

	"linkko-api/internal/domain"
	"linkko-api/internal/observability/logger"
	"linkko-api/internal/repo"

	"go.uber.org/zap"
)

var (
	ErrPipelineNotFound      = repo.ErrPipelineNotFound
	ErrPipelineNameConflict  = repo.ErrPipelineNameConflict
	ErrStageNotFound         = repo.ErrStageNotFound
	ErrStageNameConflict     = repo.ErrStageNameConflict
	ErrDefaultPipelineExists = repo.ErrDefaultPipelineExists
	ErrCannotDeleteDefault   = errors.New("cannot delete default pipeline")
)

type PipelineService struct {
	pipelineRepo  *repo.PipelineRepository
	auditRepo     *repo.AuditRepo
	workspaceRepo *repo.WorkspaceRepository
	log           *logger.Logger
}

func NewPipelineService(pipelineRepo *repo.PipelineRepository, auditRepo *repo.AuditRepo, workspaceRepo *repo.WorkspaceRepository, log *logger.Logger) *PipelineService {
	return &PipelineService{
		pipelineRepo:  pipelineRepo,
		auditRepo:     auditRepo,
		workspaceRepo: workspaceRepo,
		log:           log,
	}
}

// getMemberRoleWithLogging wraps GetMemberRole with authorization audit logging.
func (s *PipelineService) getMemberRoleWithLogging(ctx context.Context, actorID, workspaceID string) (domain.Role, error) {
	role, err := s.workspaceRepo.GetMemberRole(ctx, actorID, workspaceID)
	if err != nil {
		s.log.Error(ctx, "failed to get member role",
			logger.Module("pipeline"),
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
		logger.Module("pipeline"),
		logger.Action("authorization"),
		zap.String("actor_id", actorID),
		zap.String("workspace_id", workspaceID),
		zap.String("role", string(role)),
	)
	return role, nil
}

// ListPipelines retrieves pipelines with optional stages.
// Permission: all workspace members can list pipelines.
func (s *PipelineService) ListPipelines(ctx context.Context, workspaceID, actorID string, params domain.ListPipelinesParams) (*domain.PipelineListResponse, error) {
	// Fetch user's role in this workspace from database
	role, err := s.getMemberRoleWithLogging(ctx, actorID, workspaceID)
	if err != nil {
		return nil, err
	}

	// RBAC: all workspace members can list pipelines
	if !domain.IsWorkspaceMember(role) {
		return nil, ErrUnauthorized
	}

	params.WorkspaceID = workspaceID

	pipelines, nextCursor, err := s.pipelineRepo.List(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list pipelines: %w", err)
	}

	response := &domain.PipelineListResponse{
		Data: pipelines,
	}
	response.Meta.HasNextPage = nextCursor != ""
	if nextCursor != "" {
		response.Meta.NextCursor = &nextCursor
	}
	return response, nil
}

// GetPipeline retrieves a single pipeline with all stages.
// Permission: all workspace members can view pipelines.
func (s *PipelineService) GetPipeline(ctx context.Context, workspaceID, pipelineID, actorID string) (*domain.Pipeline, error) {
	// Fetch user's role in this workspace from database
	role, err := s.getMemberRoleWithLogging(ctx, actorID, workspaceID)
	if err != nil {
		return nil, err
	}

	// RBAC: all workspace members can view pipelines
	if !domain.IsWorkspaceMember(role) {
		return nil, ErrUnauthorized
	}

	pipeline, err := s.pipelineRepo.GetWithStages(ctx, workspaceID, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("get pipeline: %w", err)
	}

	return pipeline, nil
}

// CreatePipeline creates a new pipeline with RBAC validation.
// Permission: only admin and manager can create pipelines.
// If isDefault is true, sets this pipeline as the workspace default (transaction).
func (s *PipelineService) CreatePipeline(ctx context.Context, workspaceID, actorID string, req *domain.CreatePipelineRequest) (*domain.Pipeline, error) {
	// Fetch user's role in this workspace from database
	role, err := s.getMemberRoleWithLogging(ctx, actorID, workspaceID)
	if err != nil {
		return nil, err
	}

	// RBAC: only admin and manager can create pipelines
	if !domain.CanDeleteContacts(role) { // Reusing manager-level permission
		return nil, ErrUnauthorized
	}

	// Default values for optional fields
	defaultType := domain.PipelineTypeSales
	if req.PipelineType == nil {
		req.PipelineType = &defaultType
	}

	pipeline := &domain.Pipeline{
		ID:           generateID(),
		WorkspaceID:  workspaceID,
		Name:         req.Name,
		PipelineType: *req.PipelineType,
		IsActive:     true,    // Default active
		IsDefault:    false,   // Will be set via SetAsDefault if needed
		OwnerID:      actorID, // Creator is owner
	}

	if req.Description != nil {
		pipeline.Description = req.Description
	}
	if req.IsActive != nil {
		pipeline.IsActive = *req.IsActive
	}
	if req.OwnerID != nil {
		pipeline.OwnerID = *req.OwnerID
	}

	// If isDefault requested, use transaction to set as default
	if req.IsDefault != nil && *req.IsDefault {
		tx, err := s.pipelineRepo.BeginTx(ctx)
		if err != nil {
			return nil, fmt.Errorf("begin transaction: %w", err)
		}
		defer tx.Rollback(ctx)

		// Create pipeline first
		err = s.pipelineRepo.Create(ctx, pipeline)
		if err != nil {
			return nil, fmt.Errorf("create pipeline: %w", err)
		}

		// Set as default (deactivates other defaults)
		err = s.pipelineRepo.SetAsDefault(ctx, tx, workspaceID, pipeline.ID)
		if err != nil {
			return nil, fmt.Errorf("set as default: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit transaction: %w", err)
		}

		pipeline.IsDefault = true
	} else {
		// Simple create without default logic
		err = s.pipelineRepo.Create(ctx, pipeline)
		if err != nil {
			return nil, fmt.Errorf("create pipeline: %w", err)
		}
	}

	// Audit: log pipeline creation
	pipelineIDStr := pipeline.ID
	auditErr := s.auditRepo.LogAction(
		ctx,
		workspaceID,
		actorID,
		"create",
		"pipeline",
		&pipelineIDStr,
		nil,
		"",
		"",
	)
	if auditErr != nil {
		// Log audit failure but don't fail the operation
	}

	return pipeline, nil
}

// CreatePipelineWithStages creates a pipeline and its stages in a single operation.
// Permission: only admin and manager can create pipelines.
func (s *PipelineService) CreatePipelineWithStages(ctx context.Context, workspaceID, actorID string, req *domain.CreatePipelineWithStagesRequest) (*domain.Pipeline, error) {
	// Fetch user's role in this workspace from database
	role, err := s.getMemberRoleWithLogging(ctx, actorID, workspaceID)
	if err != nil {
		return nil, err
	}

	// RBAC: only admin and manager can create pipelines
	if !domain.CanDeleteContacts(role) {
		return nil, ErrUnauthorized
	}

	// Default values for optional fields
	defaultType := domain.PipelineTypeSales
	if req.Pipeline.PipelineType == nil {
		req.Pipeline.PipelineType = &defaultType
	}

	tx, err := s.pipelineRepo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Create pipeline
	pipeline := &domain.Pipeline{
		ID:           generateID(),
		WorkspaceID:  workspaceID,
		Name:         req.Pipeline.Name,
		PipelineType: *req.Pipeline.PipelineType,
		IsActive:     true,
		IsDefault:    false,
		OwnerID:      actorID,
	}

	if req.Pipeline.Description != nil {
		pipeline.Description = req.Pipeline.Description
	}
	if req.Pipeline.IsActive != nil {
		pipeline.IsActive = *req.Pipeline.IsActive
	}
	if req.Pipeline.OwnerID != nil {
		pipeline.OwnerID = *req.Pipeline.OwnerID
	}

	err = s.pipelineRepo.Create(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("create pipeline: %w", err)
	}

	// Create stages
	for i, stageReq := range req.Stages {
		// Default values for optional fields
		defaultGroup := domain.StageGroupActive
		if stageReq.StageGroup == nil {
			stageReq.StageGroup = &defaultGroup
		}

		stage := &domain.PipelineStage{
			ID:         generateID(),
			PipelineID: &pipeline.ID,
			WorkspaceID: workspaceID,
			Name:       stageReq.Name,
			Group:      *stageReq.StageGroup,
			OrderIndex: i + 1, // Auto-assign sequential orderIndex
		}

		if stageReq.Description != nil {
			stage.Description = stageReq.Description
		}
		if stageReq.Probability != nil {
			stage.Probability = *stageReq.Probability
		}
		if stageReq.AutoArchiveDays != nil {
			stage.AutoArchiveDays = stageReq.AutoArchiveDays
		}

		err = s.pipelineRepo.CreateStage(ctx, stage)
		if err != nil {
			return nil, fmt.Errorf("create stage %s: %w", stageReq.Name, err)
		}
	}

	// Set as default if requested
	if req.Pipeline.IsDefault != nil && *req.Pipeline.IsDefault {
		err = s.pipelineRepo.SetAsDefault(ctx, tx, workspaceID, pipeline.ID)
		if err != nil {
			return nil, fmt.Errorf("set as default: %w", err)
		}
		pipeline.IsDefault = true
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	// Load stages for response
	result, err := s.pipelineRepo.GetWithStages(ctx, workspaceID, pipeline.ID)
	if err != nil {
		return nil, fmt.Errorf("get created pipeline: %w", err)
	}

	// Audit: log pipeline creation
	pipelineIDStr := pipeline.ID
	auditErr := s.auditRepo.LogAction(
		ctx,
		workspaceID,
		actorID,
		"create",
		"pipeline_with_stages",
		&pipelineIDStr,
		nil,
		"",
		"",
	)
	if auditErr != nil {
		// Log audit failure but don't fail the operation
	}

	return result, nil
}

// UpdatePipeline updates a pipeline with RBAC validation.
// Permission: only admin and manager can update pipelines.
// If isDefault changes to true, uses SetAsDefault transaction.
func (s *PipelineService) UpdatePipeline(ctx context.Context, workspaceID, pipelineID, actorID string, req *domain.UpdatePipelineRequest) (*domain.Pipeline, error) {
	// Fetch user's role in this workspace from database
	role, err := s.getMemberRoleWithLogging(ctx, actorID, workspaceID)
	if err != nil {
		return nil, err
	}

	// RBAC: only admin and manager can update pipelines
	if !domain.CanDeleteContacts(role) {
		return nil, ErrUnauthorized
	}

	// Verify pipeline exists
	_, err = s.pipelineRepo.Get(ctx, workspaceID, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("get pipeline: %w", err)
	}

	// If changing isDefault to true, use transaction
	if req.IsDefault != nil && *req.IsDefault {
		tx, err := s.pipelineRepo.BeginTx(ctx)
		if err != nil {
			return nil, fmt.Errorf("begin transaction: %w", err)
		}
		defer tx.Rollback(ctx)

		// Update pipeline fields (excluding isDefault, handled by SetAsDefault)
		updateReqCopy := *req
		updateReqCopy.IsDefault = nil
		err = s.pipelineRepo.Update(ctx, workspaceID, pipelineID, &updateReqCopy)
		if err != nil {
			return nil, fmt.Errorf("update pipeline: %w", err)
		}

		// Set as default
		err = s.pipelineRepo.SetAsDefault(ctx, tx, workspaceID, pipelineID)
		if err != nil {
			return nil, fmt.Errorf("set as default: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit transaction: %w", err)
		}
	} else {
		// Regular update without default logic
		err = s.pipelineRepo.Update(ctx, workspaceID, pipelineID, req)
		if err != nil {
			return nil, fmt.Errorf("update pipeline: %w", err)
		}
	}

	// Fetch updated pipeline
	pipeline, err := s.pipelineRepo.GetWithStages(ctx, workspaceID, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("get updated pipeline: %w", err)
	}

	// Audit: log pipeline update
	pipelineIDStr := pipelineID
	auditErr := s.auditRepo.LogAction(
		ctx,
		workspaceID,
		actorID,
		"update",
		"pipeline",
		&pipelineIDStr,
		nil,
		"",
		"",
	)
	if auditErr != nil {
		// Log audit failure but don't fail the operation
	}

	return pipeline, nil
}

// DeletePipeline soft deletes a pipeline with RBAC validation.
// Permission: only admin and manager can delete pipelines.
// Cannot delete default pipeline (must set another as default first).
func (s *PipelineService) DeletePipeline(ctx context.Context, workspaceID, pipelineID, actorID string) error {
	// Fetch user's role in this workspace from database
	role, err := s.getMemberRoleWithLogging(ctx, actorID, workspaceID)
	if err != nil {
		return err
	}

	// RBAC: only admin and manager can delete pipelines
	if !domain.CanDeleteContacts(role) {
		return ErrUnauthorized
	}

	// Check if pipeline is default
	pipeline, err := s.pipelineRepo.Get(ctx, workspaceID, pipelineID)
	if err != nil {
		return fmt.Errorf("get pipeline: %w", err)
	}

	if pipeline.IsDefault {
		return ErrCannotDeleteDefault
	}

	err = s.pipelineRepo.SoftDelete(ctx, workspaceID, pipelineID)
	if err != nil {
		return fmt.Errorf("delete pipeline: %w", err)
	}

	// Audit: log pipeline deletion
	pipelineIDStr := pipelineID
	auditErr := s.auditRepo.LogAction(
		ctx,
		workspaceID,
		actorID,
		"delete",
		"pipeline",
		&pipelineIDStr,
		nil,
		"",
		"",
	)
	if auditErr != nil {
		// Log audit failure but don't fail the operation
	}

	return nil
}

// ===== PIPELINE STAGE METHODS =====

// ListStages retrieves all stages for a pipeline.
// Permission: all workspace members can list stages.
func (s *PipelineService) ListStages(ctx context.Context, workspaceID, pipelineID, actorID string) ([]domain.PipelineStage, error) {
	// Fetch user's role in this workspace from database
	role, err := s.getMemberRoleWithLogging(ctx, actorID, workspaceID)
	if err != nil {
		return nil, err
	}

	// RBAC: all workspace members can list stages
	if !domain.IsWorkspaceMember(role) {
		return nil, ErrUnauthorized
	}

	// Verify pipeline belongs to workspace
	_, err = s.pipelineRepo.Get(ctx, workspaceID, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("get pipeline: %w", err)
	}

	stages, err := s.pipelineRepo.ListStagesByPipeline(ctx, workspaceID, &pipelineID)
	if err != nil {
		return nil, fmt.Errorf("list stages: %w", err)
	}

	return stages, nil
}

// CreateStage creates a new stage in a pipeline.
// Permission: only admin and manager can create stages.
// Auto-assigns orderIndex as max+1.
func (s *PipelineService) CreateStage(ctx context.Context, workspaceID, pipelineID, actorID string, req *domain.CreateStageRequest) (*domain.PipelineStage, error) {
	// Fetch user's role in this workspace from database
	role, err := s.getMemberRoleWithLogging(ctx, actorID, workspaceID)
	if err != nil {
		return nil, err
	}

	// RBAC: only admin and manager can create stages
	if !domain.CanDeleteContacts(role) {
		return nil, ErrUnauthorized
	}

	// Verify pipeline belongs to workspace
	_, err = s.pipelineRepo.Get(ctx, workspaceID, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("get pipeline: %w", err)
	}

	// Auto-assign orderIndex
	maxOrder, err := s.pipelineRepo.GetMaxOrderIndex(ctx, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("get max order: %w", err)
	}

	// Default values for optional fields
	defaultGroup := domain.StageGroupActive
	if req.StageGroup == nil {
		req.StageGroup = &defaultGroup
	}

	stage := &domain.PipelineStage{
		ID:         generateID(),
		PipelineID: &pipelineID,
		WorkspaceID: workspaceID,
		Name:       req.Name,
		Group:      *req.StageGroup,
		OrderIndex: maxOrder + 1,
	}

	if req.Description != nil {
		stage.Description = req.Description
	}
	if req.Probability != nil {
		stage.Probability = *req.Probability
	}
	if req.AutoArchiveDays != nil {
		stage.AutoArchiveDays = req.AutoArchiveDays
	}

	err = s.pipelineRepo.CreateStage(ctx, stage)
	if err != nil {
		return nil, fmt.Errorf("create stage: %w", err)
	}

	// Audit: log stage creation
	stageIDStr := stage.ID
	auditErr := s.auditRepo.LogAction(
		ctx,
		workspaceID,
		actorID,
		"create",
		"pipeline_stage",
		&stageIDStr,
		nil,
		"",
		"",
	)
	if auditErr != nil {
		// Log audit failure but don't fail the operation
	}

	return stage, nil
}

// UpdateStage updates a stage with RBAC validation.
// Permission: only admin and manager can update stages.
func (s *PipelineService) UpdateStage(ctx context.Context, workspaceID, stageID, actorID string, req *domain.UpdateStageRequest) (*domain.PipelineStage, error) {
	// Fetch user's role in this workspace from database
	role, err := s.getMemberRoleWithLogging(ctx, actorID, workspaceID)
	if err != nil {
		return nil, err
	}

	// RBAC: only admin and manager can update stages
	if !domain.CanDeleteContacts(role) {
		return nil, ErrUnauthorized
	}

	// Verify stage exists and belongs to workspace pipeline
	stage, err := s.pipelineRepo.GetStage(ctx, stageID)
	if err != nil {
		return nil, fmt.Errorf("get stage: %w", err)
	}

	_, err = s.pipelineRepo.Get(ctx, workspaceID, *stage.PipelineID)
	if err != nil {
		return nil, fmt.Errorf("get pipeline: %w", err)
	}

	err = s.pipelineRepo.UpdateStage(ctx, stageID, req)
	if err != nil {
		return nil, fmt.Errorf("update stage: %w", err)
	}

	// Fetch updated stage
	updatedStage, err := s.pipelineRepo.GetStage(ctx, stageID)
	if err != nil {
		return nil, fmt.Errorf("get updated stage: %w", err)
	}

	// Audit: log stage update
	stageIDStr := stageID
	auditErr := s.auditRepo.LogAction(
		ctx,
		workspaceID,
		actorID,
		"update",
		"pipeline_stage",
		&stageIDStr,
		nil,
		"",
		"",
	)
	if auditErr != nil {
		// Log audit failure but don't fail the operation
	}

	return updatedStage, nil
}

// DeleteStage soft deletes a stage with RBAC validation.
// Permission: only admin and manager can delete stages.
func (s *PipelineService) DeleteStage(ctx context.Context, workspaceID, stageID, actorID string) error {
	// Fetch user's role in this workspace from database
	role, err := s.getMemberRoleWithLogging(ctx, actorID, workspaceID)
	if err != nil {
		return err
	}

	// RBAC: only admin and manager can delete stages
	if !domain.CanDeleteContacts(role) {
		return ErrUnauthorized
	}

	// Verify stage exists and belongs to workspace pipeline
	stage, err := s.pipelineRepo.GetStage(ctx, stageID)
	if err != nil {
		return fmt.Errorf("get stage: %w", err)
	}

	_, err = s.pipelineRepo.Get(ctx, workspaceID, *stage.PipelineID)
	if err != nil {
		return fmt.Errorf("get pipeline: %w", err)
	}

	err = s.pipelineRepo.SoftDeleteStage(ctx, stageID)
	if err != nil {
		return fmt.Errorf("delete stage: %w", err)
	}

	stageIDStr := stageID
	auditErr := s.auditRepo.LogAction(
		ctx,
		workspaceID,
		actorID,
		"delete",
		"pipeline_stage",
		&stageIDStr,
		nil,
		"",
		"",
	)
	if auditErr != nil {
		// Log audit failure but don't fail the operation
	}

	return nil
}

// ===== SEEDING METHODS =====

// CreateDefaultPipeline creates a default "Vendas Padrão" pipeline with 5 standard stages.
// This is called automatically when a workspace is created.
// Permission: internal service method (no RBAC check).
func (s *PipelineService) CreateDefaultPipeline(ctx context.Context, workspaceID string, ownerID string) (*domain.Pipeline, error) {
	req := &domain.CreatePipelineWithStagesRequest{
		Pipeline: domain.CreatePipelineRequest{
			Name:         "Vendas Padrão",
			Description:  strPtr("Pipeline de vendas padrão criado automaticamente"),
			PipelineType: pipelineTypePtr(domain.PipelineTypeSales),
			IsActive:     boolPtr(true),
			IsDefault:    boolPtr(true),
			OwnerID:      &ownerID,
		},
		Stages: []domain.CreateStageRequest{
			{
				Name:        "Lead",
				Description: strPtr("Novos leads gerados"),
				StageGroup:  stageGroupPtr(domain.StageGroupActive),
				Probability: intPtr(10),
			},
			{
				Name:        "Qualificado",
				Description: strPtr("Lead qualificado e validado"),
				StageGroup:  stageGroupPtr(domain.StageGroupActive),
				Probability: intPtr(30),
			},
			{
				Name:        "Proposta",
				Description: strPtr("Proposta comercial enviada"),
				StageGroup:  stageGroupPtr(domain.StageGroupActive),
				Probability: intPtr(50),
			},
			{
				Name:        "Negociação",
				Description: strPtr("Em negociação final"),
				StageGroup:  stageGroupPtr(domain.StageGroupActive),
				Probability: intPtr(80),
			},
			{
				Name:        "Fechado",
				Description: strPtr("Venda concluída com sucesso"),
				StageGroup:  stageGroupPtr(domain.StageGroupWon),
				Probability: intPtr(100),
			},
		},
	}

	tx, err := s.pipelineRepo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Create pipeline
	pipeline := &domain.Pipeline{
		ID:           generateID(),
		WorkspaceID:  workspaceID,
		Name:         req.Pipeline.Name,
		Description:  req.Pipeline.Description,
		PipelineType: *req.Pipeline.PipelineType,
		IsActive:     true,
		IsDefault:    true,
		OwnerID:      ownerID,
	}

	err = s.pipelineRepo.Create(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("create default pipeline: %w", err)
	}

	// Create stages
	for i, stageReq := range req.Stages {
		stage := &domain.PipelineStage{
			ID:              generateID(),
			PipelineID:      &pipeline.ID,
			WorkspaceID:     workspaceID,
			Name:            stageReq.Name,
			Description:     stageReq.Description,
			Group:           *stageReq.StageGroup, // Renamed from StageGroup to Group
			OrderIndex:      i + 1,
			Color:           stageReq.Color,
			IsLocked:        false,
			Probability:     *stageReq.Probability,
			AutoArchiveDays: stageReq.AutoArchiveDays,
		}

		err = s.pipelineRepo.CreateStage(ctx, stage)
		if err != nil {
			return nil, fmt.Errorf("create default stage %s: %w", stageReq.Name, err)
		}
	}

	// Set as default
	err = s.pipelineRepo.SetAsDefault(ctx, tx, workspaceID, pipeline.ID)
	if err != nil {
		return nil, fmt.Errorf("set as default: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	// Load full pipeline with stages
	result, err := s.pipelineRepo.GetWithStages(ctx, workspaceID, pipeline.ID)
	if err != nil {
		return nil, fmt.Errorf("get created pipeline: %w", err)
	}

	return result, nil
}

// SeedDefaultPipeline is a manual endpoint to create default pipeline (fallback for repairs).
// Permission: only admin can seed default pipeline.
func (s *PipelineService) SeedDefaultPipeline(ctx context.Context, workspaceID, actorID string) (*domain.Pipeline, error) {
	// Fetch user's role in this workspace from database
	role, err := s.getMemberRoleWithLogging(ctx, actorID, workspaceID)
	if err != nil {
		return nil, err
	}

	// RBAC: only admin can seed default pipeline
	if role != domain.RoleAdmin {
		return nil, ErrUnauthorized
	}

	pipeline, err := s.CreateDefaultPipeline(ctx, workspaceID, actorID)
	if err != nil {
		return nil, fmt.Errorf("seed default pipeline: %w", err)
	}

	// Audit: log seeding action
	pipelineIDStr := pipeline.ID
	auditErr := s.auditRepo.LogAction(
		ctx,
		workspaceID,
		actorID,
		"seed",
		"pipeline",
		&pipelineIDStr,
		nil,
		"",
		"",
	)
	if auditErr != nil {
		// Log audit failure but don't fail the operation
	}

	return pipeline, nil
}

// Helper functions
func strPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func intPtr(i int) *int {
	return &i
}

func pipelineTypePtr(t domain.PipelineType) *domain.PipelineType {
	return &t
}

func stageGroupPtr(g domain.StageGroup) *domain.StageGroup {
	return &g
}
