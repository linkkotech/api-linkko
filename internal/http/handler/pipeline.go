package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"linkko-api/internal/auth"
	"linkko-api/internal/domain"
	"linkko-api/internal/observability/logger"
	"linkko-api/internal/service"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type PipelineHandler struct {
	service *service.PipelineService
}

func NewPipelineHandler(service *service.PipelineService) *PipelineHandler {
	return &PipelineHandler{service: service}
}

// ListPipelines handles GET /v1/workspaces/{workspaceId}/pipelines
func (h *PipelineHandler) ListPipelines(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	if workspaceID == "" {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_WORKSPACE_ID", "workspaceId is required")
		return
	}

	claims, ok := auth.GetClaims(ctx)
	if !ok {
		writeError(w, ctx, log, http.StatusUnauthorized, "UNAUTHORIZED", "authentication claims not found")
		return
	}

	actorID := claims.ActorID
	if actorID == "" {
		writeError(w, ctx, log, http.StatusUnauthorized, "UNAUTHORIZED", "actorID not found in claims")
		return
	}

	params := domain.ListPipelinesParams{
		WorkspaceID: workspaceID,
		Limit:       50, // Default
	}

	// Parse query parameters
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 100 {
			writeError(w, ctx, log, http.StatusBadRequest, "INVALID_LIMIT", "limit must be between 1 and 100")
			return
		}
		params.Limit = limit
	}

	if cursor := r.URL.Query().Get("cursor"); cursor != "" {
		params.Cursor = &cursor
	}

	// includeStages flag
	if includeStagesStr := r.URL.Query().Get("includeStages"); includeStagesStr == "true" {
		params.IncludeStages = true
	}

	// Optional filters
	if isDefault := r.URL.Query().Get("isDefault"); isDefault != "" {
		isDefaultBool := isDefault == "true"
		params.IsDefault = &isDefaultBool
	}

	if search := r.URL.Query().Get("q"); search != "" {
		params.Query = &search
	}

	log.Info(ctx, "listing pipelines",
		zap.String("workspaceId", workspaceID),
		zap.String("actorId", actorID),
		zap.Int("limit", params.Limit),
		zap.Bool("includeStages", params.IncludeStages),
	)

	response, err := h.service.ListPipelines(ctx, workspaceID, actorID, params)
	if err != nil {
		handlePipelineServiceError(w, ctx, log, err)
		return
	}

	log.Info(ctx, "pipelines listed successfully",
		zap.String("workspaceId", workspaceID),
		zap.Int("count", len(response.Data)),
		zap.Bool("hasNextPage", response.Meta.HasNextPage),
	)

	writeJSON(w, http.StatusOK, response)
}

// GetPipeline handles GET /v1/workspaces/{workspaceId}/pipelines/{pipelineId}
func (h *PipelineHandler) GetPipeline(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	if workspaceID == "" {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_WORKSPACE_ID", "workspaceId is required")
		return
	}

	pipelineID := chi.URLParam(r, "pipelineId")
	if pipelineID == "" {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_PIPELINE_ID", "pipelineId is required")
		return
	}

	claims, ok := auth.GetClaims(ctx)
	if !ok {
		writeError(w, ctx, log, http.StatusUnauthorized, "UNAUTHORIZED", "authentication claims not found")
		return
	}

	actorID := claims.ActorID
	if actorID == "" {
		writeError(w, ctx, log, http.StatusUnauthorized, "UNAUTHORIZED", "actorID not found in claims")
		return
	}

	log.Info(ctx, "fetching pipeline",
		zap.String("workspaceId", workspaceID),
		zap.String("pipelineId", pipelineID),
		zap.String("actorId", actorID),
	)

	pipeline, err := h.service.GetPipeline(ctx, workspaceID, pipelineID, actorID)
	if err != nil {
		handlePipelineServiceError(w, ctx, log, err)
		return
	}

	log.Info(ctx, "pipeline fetched successfully",
		zap.String("pipelineId", pipeline.ID),
	)

	writeJSON(w, http.StatusOK, pipeline)
}

// CreatePipeline handles POST /v1/workspaces/{workspaceId}/pipelines
func (h *PipelineHandler) CreatePipeline(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	if workspaceID == "" {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_WORKSPACE_ID", "workspaceId is required")
		return
	}

	claims, ok := auth.GetClaims(ctx)
	if !ok {
		writeError(w, ctx, log, http.StatusUnauthorized, "UNAUTHORIZED", "authentication claims not found")
		return
	}

	actorID := claims.ActorID
	if actorID == "" {
		writeError(w, ctx, log, http.StatusUnauthorized, "UNAUTHORIZED", "actorID not found in claims")
		return
	}

	var req domain.CreatePipelineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error(ctx, "failed to decode request body", zap.Error(err))
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_REQUEST_BODY", "request body must be valid JSON")
		return
	}

	log.Info(ctx, "creating pipeline",
		zap.String("workspaceId", workspaceID),
		zap.String("actorId", actorID),
		zap.String("name", req.Name),
	)

	pipeline, err := h.service.CreatePipeline(ctx, workspaceID, actorID, &req)
	if err != nil {
		handlePipelineServiceError(w, ctx, log, err)
		return
	}

	log.Info(ctx, "pipeline created successfully",
		zap.String("pipelineId", pipeline.ID),
	)

	writeJSON(w, http.StatusCreated, pipeline)
}

// CreatePipelineWithStages handles POST /v1/workspaces/{workspaceId}/pipelines:create-with-stages
func (h *PipelineHandler) CreatePipelineWithStages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	if workspaceID == "" {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_WORKSPACE_ID", "workspaceId is required")
		return
	}

	claims, ok := auth.GetClaims(ctx)
	if !ok {
		writeError(w, ctx, log, http.StatusUnauthorized, "UNAUTHORIZED", "authentication claims not found")
		return
	}

	actorID := claims.ActorID
	if actorID == "" {
		writeError(w, ctx, log, http.StatusUnauthorized, "UNAUTHORIZED", "actorID not found in claims")
		return
	}

	var req domain.CreatePipelineWithStagesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error(ctx, "failed to decode request body", zap.Error(err))
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_REQUEST_BODY", "request body must be valid JSON")
		return
	}

	log.Info(ctx, "creating pipeline with stages",
		zap.String("workspaceId", workspaceID),
		zap.String("actorId", actorID),
		zap.String("name", req.Pipeline.Name),
		zap.Int("stageCount", len(req.Stages)),
	)

	pipeline, err := h.service.CreatePipelineWithStages(ctx, workspaceID, actorID, &req)
	if err != nil {
		handlePipelineServiceError(w, ctx, log, err)
		return
	}

	log.Info(ctx, "pipeline created successfully with stages",
		zap.String("pipelineId", pipeline.ID),
	)

	writeJSON(w, http.StatusCreated, pipeline)
}

// UpdatePipeline handles PATCH /v1/workspaces/{workspaceId}/pipelines/{pipelineId}
func (h *PipelineHandler) UpdatePipeline(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	pipelineID := chi.URLParam(r, "pipelineId")
	if workspaceID == "" || pipelineID == "" {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_ID", "workspaceId and pipelineId are required")
		return
	}

	claims, ok := auth.GetClaims(ctx)
	if !ok {
		writeError(w, ctx, log, http.StatusUnauthorized, "UNAUTHORIZED", "authentication claims not found")
		return
	}

	actorID := claims.ActorID
	if actorID == "" {
		writeError(w, ctx, log, http.StatusUnauthorized, "UNAUTHORIZED", "actorID not found in claims")
		return
	}

	var req domain.UpdatePipelineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error(ctx, "failed to decode request body", zap.Error(err))
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_REQUEST_BODY", "request body must be valid JSON")
		return
	}

	log.Info(ctx, "updating pipeline",
		zap.String("workspaceId", workspaceID),
		zap.String("pipelineId", pipelineID),
		zap.String("actorId", actorID),
	)

	pipeline, err := h.service.UpdatePipeline(ctx, workspaceID, pipelineID, actorID, &req)
	if err != nil {
		handlePipelineServiceError(w, ctx, log, err)
		return
	}

	log.Info(ctx, "pipeline updated successfully",
		zap.String("pipelineId", pipeline.ID),
	)

	writeJSON(w, http.StatusOK, pipeline)
}

// DeletePipeline handles DELETE /v1/workspaces/{workspaceId}/pipelines/{pipelineId}
func (h *PipelineHandler) DeletePipeline(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	pipelineID := chi.URLParam(r, "pipelineId")
	if workspaceID == "" || pipelineID == "" {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_ID", "workspaceId and pipelineId are required")
		return
	}

	claims, ok := auth.GetClaims(ctx)
	if !ok {
		writeError(w, ctx, log, http.StatusUnauthorized, "UNAUTHORIZED", "authentication claims not found")
		return
	}

	actorID := claims.ActorID
	if actorID == "" {
		writeError(w, ctx, log, http.StatusUnauthorized, "UNAUTHORIZED", "actorID not found in claims")
		return
	}

	log.Info(ctx, "deleting pipeline",
		zap.String("workspaceId", workspaceID),
		zap.String("pipelineId", pipelineID),
		zap.String("actorId", actorID),
	)

	err := h.service.DeletePipeline(ctx, workspaceID, pipelineID, actorID)
	if err != nil {
		handlePipelineServiceError(w, ctx, log, err)
		return
	}

	log.Info(ctx, "pipeline deleted successfully",
		zap.String("pipelineId", pipelineID),
	)

	w.WriteHeader(http.StatusNoContent)
}

// SeedDefaultPipeline handles POST /v1/workspaces/{workspaceId}/pipelines:seed-default
func (h *PipelineHandler) SeedDefaultPipeline(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	if workspaceID == "" {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_WORKSPACE_ID", "workspaceId is required")
		return
	}

	claims, ok := auth.GetClaims(ctx)
	if !ok {
		writeError(w, ctx, log, http.StatusUnauthorized, "UNAUTHORIZED", "authentication claims not found")
		return
	}

	actorID := claims.ActorID
	if actorID == "" {
		writeError(w, ctx, log, http.StatusUnauthorized, "UNAUTHORIZED", "actorID not found in claims")
		return
	}

	log.Info(ctx, "seeding default pipeline",
		zap.String("workspaceId", workspaceID),
		zap.String("actorId", actorID),
	)

	pipeline, err := h.service.SeedDefaultPipeline(ctx, workspaceID, actorID)
	if err != nil {
		handlePipelineServiceError(w, ctx, log, err)
		return
	}

	log.Info(ctx, "default pipeline seeded successfully",
		zap.String("pipelineId", pipeline.ID),
	)

	writeJSON(w, http.StatusCreated, pipeline)
}

// ===== PIPELINE STAGE HANDLERS =====

// ListStages handles GET /v1/workspaces/{workspaceId}/pipelines/{pipelineId}/stages
func (h *PipelineHandler) ListStages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	pipelineID := chi.URLParam(r, "pipelineId")
	if workspaceID == "" || pipelineID == "" {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_ID", "workspaceId and pipelineId are required")
		return
	}

	claims, ok := auth.GetClaims(ctx)
	if !ok {
		writeError(w, ctx, log, http.StatusUnauthorized, "UNAUTHORIZED", "authentication claims not found")
		return
	}

	actorID := claims.ActorID
	if actorID == "" {
		writeError(w, ctx, log, http.StatusUnauthorized, "UNAUTHORIZED", "actorID not found in claims")
		return
	}

	log.Info(ctx, "listing stages",
		zap.String("workspaceId", workspaceID),
		zap.String("pipelineId", pipelineID),
		zap.String("actorId", actorID),
	)

	stages, err := h.service.ListStages(ctx, workspaceID, pipelineID, actorID)
	if err != nil {
		handlePipelineServiceError(w, ctx, log, err)
		return
	}

	log.Info(ctx, "stages listed successfully",
		zap.String("pipelineId", pipelineID),
		zap.Int("count", len(stages)),
	)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": stages,
	})
}

// CreateStage handles POST /v1/workspaces/{workspaceId}/pipelines/{pipelineId}/stages
func (h *PipelineHandler) CreateStage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	pipelineID := chi.URLParam(r, "pipelineId")
	if workspaceID == "" || pipelineID == "" {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_ID", "workspaceId and pipelineId are required")
		return
	}

	claims, ok := auth.GetClaims(ctx)
	if !ok {
		writeError(w, ctx, log, http.StatusUnauthorized, "UNAUTHORIZED", "authentication claims not found")
		return
	}

	actorID := claims.ActorID
	if actorID == "" {
		writeError(w, ctx, log, http.StatusUnauthorized, "UNAUTHORIZED", "actorID not found in claims")
		return
	}

	var req domain.CreateStageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error(ctx, "failed to decode request body", zap.Error(err))
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_REQUEST_BODY", "request body must be valid JSON")
		return
	}

	log.Info(ctx, "creating stage",
		zap.String("workspaceId", workspaceID),
		zap.String("pipelineId", pipelineID),
		zap.String("actorId", actorID),
		zap.String("name", req.Name),
	)

	stage, err := h.service.CreateStage(ctx, workspaceID, pipelineID, actorID, &req)
	if err != nil {
		handlePipelineServiceError(w, ctx, log, err)
		return
	}

	log.Info(ctx, "stage created successfully",
		zap.String("stageId", stage.ID),
	)

	writeJSON(w, http.StatusCreated, stage)
}

// UpdateStage handles PATCH /v1/workspaces/{workspaceId}/pipelines/{pipelineId}/stages/{stageId}
func (h *PipelineHandler) UpdateStage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	stageID := chi.URLParam(r, "stageId")
	if workspaceID == "" || stageID == "" {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_ID", "workspaceId and stageId are required")
		return
	}

	claims, ok := auth.GetClaims(ctx)
	if !ok {
		writeError(w, ctx, log, http.StatusUnauthorized, "UNAUTHORIZED", "authentication claims not found")
		return
	}

	actorID := claims.ActorID
	if actorID == "" {
		writeError(w, ctx, log, http.StatusUnauthorized, "UNAUTHORIZED", "actorID not found in claims")
		return
	}

	var req domain.UpdateStageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error(ctx, "failed to decode request body", zap.Error(err))
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_REQUEST_BODY", "request body must be valid JSON")
		return
	}

	log.Info(ctx, "updating stage",
		zap.String("workspaceId", workspaceID),
		zap.String("stageId", stageID),
		zap.String("actorId", actorID),
	)

	stage, err := h.service.UpdateStage(ctx, workspaceID, stageID, actorID, &req)
	if err != nil {
		handlePipelineServiceError(w, ctx, log, err)
		return
	}

	log.Info(ctx, "stage updated successfully",
		zap.String("stageId", stage.ID),
	)

	writeJSON(w, http.StatusOK, stage)
}

// DeleteStage handles DELETE /v1/workspaces/{workspaceId}/pipelines/{pipelineId}/stages/{stageId}
func (h *PipelineHandler) DeleteStage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	stageID := chi.URLParam(r, "stageId")
	if workspaceID == "" || stageID == "" {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_ID", "workspaceId and stageId are required")
		return
	}

	claims, ok := auth.GetClaims(ctx)
	if !ok {
		writeError(w, ctx, log, http.StatusUnauthorized, "UNAUTHORIZED", "authentication claims not found")
		return
	}

	actorID := claims.ActorID
	if actorID == "" {
		writeError(w, ctx, log, http.StatusUnauthorized, "UNAUTHORIZED", "actorID not found in claims")
		return
	}

	log.Info(ctx, "deleting stage",
		zap.String("workspaceId", workspaceID),
		zap.String("stageId", stageID),
		zap.String("actorId", actorID),
	)

	err := h.service.DeleteStage(ctx, workspaceID, stageID, actorID)
	if err != nil {
		handlePipelineServiceError(w, ctx, log, err)
		return
	}

	log.Info(ctx, "stage deleted successfully",
		zap.String("stageId", stageID),
	)

	w.WriteHeader(http.StatusNoContent)
}

// handlePipelineServiceError maps service errors to HTTP responses
func handlePipelineServiceError(w http.ResponseWriter, ctx context.Context, log *logger.Logger, err error) {
	switch {
	case errors.Is(err, service.ErrMemberNotFound):
		writeError(w, ctx, log, http.StatusForbidden, "FORBIDDEN", "insufficient permissions for this workspace")
	case errors.Is(err, service.ErrUnauthorized):
		writeError(w, ctx, log, http.StatusForbidden, "FORBIDDEN", "insufficient permissions for this action")
	case errors.Is(err, service.ErrPipelineNotFound):
		writeError(w, ctx, log, http.StatusNotFound, "NOT_FOUND", "pipeline not found")
	case errors.Is(err, service.ErrPipelineNameConflict):
		writeError(w, ctx, log, http.StatusConflict, "CONFLICT", "pipeline with this name already exists")
	case errors.Is(err, service.ErrStageNotFound):
		writeError(w, ctx, log, http.StatusNotFound, "NOT_FOUND", "stage not found")
	case errors.Is(err, service.ErrStageNameConflict):
		writeError(w, ctx, log, http.StatusConflict, "CONFLICT", "stage with this name already exists in pipeline")
	case errors.Is(err, service.ErrDefaultPipelineExists):
		writeError(w, ctx, log, http.StatusConflict, "CONFLICT", "another pipeline is already set as default")
	case errors.Is(err, service.ErrCannotDeleteDefault):
		writeError(w, ctx, log, http.StatusUnprocessableEntity, "CANNOT_DELETE_DEFAULT", "cannot delete default pipeline; set another as default first")
	default:
		log.Error(ctx, "unexpected service error", zap.Error(err))
		writeError(w, ctx, log, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
	}
}
