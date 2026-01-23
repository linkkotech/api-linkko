package service

import (
	"context"
	"time"

	"linkko-api/internal/domain"
	"linkko-api/internal/observability/logger"
	"linkko-api/internal/repo"

	"go.uber.org/zap"
)

type ActivityService struct {
	activityRepo  *repo.ActivityRepository
	workspaceRepo *repo.WorkspaceRepository
	auditRepo     *repo.AuditRepo
	log           *logger.Logger
}

func NewActivityService(activityRepo *repo.ActivityRepository, workspaceRepo *repo.WorkspaceRepository, auditRepo *repo.AuditRepo, log *logger.Logger) *ActivityService {
	return &ActivityService{
		activityRepo:  activityRepo,
		workspaceRepo: workspaceRepo,
		auditRepo:     auditRepo,
		log:           log,
	}
}

// getMemberRoleWithLogging wraps GetMemberRole with authorization audit logging.
func (s *ActivityService) getMemberRoleWithLogging(ctx context.Context, actorID, workspaceID string) (domain.Role, error) {
	role, err := s.workspaceRepo.GetMemberRole(ctx, actorID, workspaceID)
	if err != nil {
		s.log.Error(ctx, "failed to get member role",
			logger.Module("activity"),
			logger.Action("authorization"),
			zap.String("actor_id", actorID),
			zap.String("workspace_id", workspaceID),
			zap.Error(err),
		)
		return "", err
	}

	s.log.Info(ctx, "workspace access granted",
		logger.Module("activity"),
		logger.Action("authorization"),
		zap.String("actor_id", actorID),
		zap.String("workspace_id", workspaceID),
		zap.String("role", string(role)),
	)
	return role, nil
}

func (s *ActivityService) CreateNote(ctx context.Context, workspaceID, actorID string, req *domain.CreateNoteRequest) (*domain.Note, error) {
	role, err := s.getMemberRoleWithLogging(ctx, actorID, workspaceID)
	if err != nil {
		return nil, err
	}
	if !domain.CanModifyContacts(role) {
		return nil, ErrUnauthorized
	}

	note := &domain.Note{
		ID:          generateDealID(), // reuse same cuid gen
		WorkspaceID: workspaceID,
		CompanyID:   req.CompanyID,
		ContactID:   req.ContactID,
		DealID:      req.DealID,
		Content:     req.Content,
		UserID:      actorID,
	}

	created, err := s.activityRepo.CreateNote(ctx, note)
	if err != nil {
		return nil, err
	}

	// Create Timeline Activity
	activity := &domain.Activity{
		ID:          generateDealID(),
		WorkspaceID: workspaceID,
		CompanyID:   req.CompanyID,
		ContactID:   req.ContactID,
		DealID:      req.DealID,
		Type:        domain.ActivityTypeNote,
		ActivityID:  &created.ID,
		UserID:      actorID,
		CreatedAt:   time.Now(),
	}

	_, err = s.activityRepo.CreateActivity(ctx, activity)
	if err != nil {
		// Log error but don't fail note creation
	}

	return created, nil
}

func (s *ActivityService) CreateCall(ctx context.Context, workspaceID, actorID string, req *domain.CreateCallRequest) (*domain.Call, error) {
	role, err := s.getMemberRoleWithLogging(ctx, actorID, workspaceID)
	if err != nil {
		return nil, err
	}
	if !domain.CanModifyContacts(role) {
		return nil, ErrUnauthorized
	}

	call := &domain.Call{
		ID:           generateDealID(),
		WorkspaceID:  workspaceID,
		ContactID:    req.ContactID,
		CompanyID:    req.CompanyID,
		Direction:    req.Direction,
		Duration:     req.Duration,
		RecordingURL: req.RecordingURL,
		Summary:      req.Summary,
		UserID:       actorID,
		CalledAt:     req.CalledAt,
	}

	if call.CalledAt.IsZero() {
		call.CalledAt = time.Now()
	}

	created, err := s.activityRepo.CreateCall(ctx, call)
	if err != nil {
		return nil, err
	}

	// Create Timeline Activity
	activity := &domain.Activity{
		ID:          generateDealID(),
		WorkspaceID: workspaceID,
		CompanyID:   req.CompanyID,
		ContactID:   &req.ContactID,
		Type:        domain.ActivityTypeCall,
		ActivityID:  &created.ID,
		UserID:      actorID,
		CreatedAt:   time.Now(),
	}

	_, err = s.activityRepo.CreateActivity(ctx, activity)
	if err != nil {
		// Log error
	}

	return created, nil
}

func (s *ActivityService) ListTimeline(ctx context.Context, workspaceID, actorID string, contactID, companyID, dealID *string) ([]domain.Activity, error) {
	role, err := s.getMemberRoleWithLogging(ctx, actorID, workspaceID)
	if err != nil {
		return nil, err
	}
	if !domain.IsWorkspaceMember(role) {
		return nil, ErrUnauthorized
	}

	return s.activityRepo.List(ctx, workspaceID, contactID, companyID, dealID)
}
