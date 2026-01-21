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
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type CompanyHandler struct {
	service *service.CompanyService
}

func NewCompanyHandler(service *service.CompanyService) *CompanyHandler {
	return &CompanyHandler{service: service}
}

// ListCompanies handles GET /v1/workspaces/{workspaceId}/companies
func (h *CompanyHandler) ListCompanies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceIDStr := chi.URLParam(r, "workspaceId")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_WORKSPACE_ID", "workspaceId must be a valid UUID")
		return
	}

	claims, ok := auth.GetClaims(ctx)
	if !ok {
		writeError(w, ctx, log, http.StatusUnauthorized, "UNAUTHORIZED", "authentication claims not found")
		return
	}

	actorID, err := uuid.Parse(claims.ActorID)
	if err != nil {
		log.Error(ctx, "invalid actorID in claims", zap.String("actorId", claims.ActorID), zap.Error(err))
		writeError(w, ctx, log, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid authentication claims")
		return
	}

	params := domain.ListCompaniesParams{
		WorkspaceID: workspaceID,
		Limit:       50, // Default
		Sort:        "createdAt:desc",
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

	if sort := r.URL.Query().Get("sort"); sort != "" {
		params.Sort = sort
	}

	// Filtros opcionais
	if lifecycleStr := r.URL.Query().Get("lifecycleStage"); lifecycleStr != "" {
		lifecycleStage := domain.CompanyLifecycleStage(lifecycleStr)
		if !lifecycleStage.IsValid() {
			writeError(w, ctx, log, http.StatusBadRequest, "INVALID_LIFECYCLE_STAGE", "invalid lifecycleStage value")
			return
		}
		params.LifecycleStage = &lifecycleStage
	}

	if sizeStr := r.URL.Query().Get("companySize"); sizeStr != "" {
		companySize := domain.CompanySize(sizeStr)
		if !companySize.IsValid() {
			writeError(w, ctx, log, http.StatusBadRequest, "INVALID_COMPANY_SIZE", "invalid companySize value")
			return
		}
		params.CompanySize = &companySize
	}

	if industry := r.URL.Query().Get("industry"); industry != "" {
		params.Industry = &industry
	}

	if ownerIDStr := r.URL.Query().Get("ownerId"); ownerIDStr != "" {
		ownerID, err := uuid.Parse(ownerIDStr)
		if err != nil {
			writeError(w, ctx, log, http.StatusBadRequest, "INVALID_OWNER_ID", "ownerId must be a valid UUID")
			return
		}
		params.OwnerID = &ownerID
	}

	if search := r.URL.Query().Get("q"); search != "" {
		params.Query = &search
	}

	log.Info(ctx, "listing companies",
		zap.String("workspaceId", workspaceID.String()),
		zap.String("actorId", actorID.String()),
		zap.Int("limit", params.Limit),
	)

	response, err := h.service.ListCompanies(ctx, workspaceID, actorID, params)
	if err != nil {
		handleCompanyServiceError(w, ctx, log, err)
		return
	}

	log.Info(ctx, "companies listed successfully",
		zap.String("workspaceId", workspaceID.String()),
		zap.Int("count", len(response.Data)),
		zap.Bool("hasNextPage", response.Meta.HasNextPage),
	)

	writeJSON(w, http.StatusOK, response)
}

// GetCompany handles GET /v1/workspaces/{workspaceId}/companies/{companyId}
func (h *CompanyHandler) GetCompany(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceIDStr := chi.URLParam(r, "workspaceId")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_WORKSPACE_ID", "workspaceId must be a valid UUID")
		return
	}

	companyIDStr := chi.URLParam(r, "companyId")
	companyID, err := uuid.Parse(companyIDStr)
	if err != nil {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_COMPANY_ID", "companyId must be a valid UUID")
		return
	}

	claims, ok := auth.GetClaims(ctx)
	if !ok {
		writeError(w, ctx, log, http.StatusUnauthorized, "UNAUTHORIZED", "authentication claims not found")
		return
	}

	actorID, err := uuid.Parse(claims.ActorID)
	if err != nil {
		log.Error(ctx, "invalid actorID in claims", zap.String("actorId", claims.ActorID), zap.Error(err))
		writeError(w, ctx, log, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid authentication claims")
		return
	}

	log.Info(ctx, "fetching company",
		zap.String("workspaceId", workspaceID.String()),
		zap.String("companyId", companyID.String()),
		zap.String("actorId", actorID.String()),
	)

	company, err := h.service.GetCompany(ctx, workspaceID, companyID, actorID)
	if err != nil {
		handleCompanyServiceError(w, ctx, log, err)
		return
	}

	log.Info(ctx, "company fetched successfully",
		zap.String("companyId", company.ID.String()),
	)

	writeJSON(w, http.StatusOK, company)
}

// CreateCompany handles POST /v1/workspaces/{workspaceId}/companies
func (h *CompanyHandler) CreateCompany(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceIDStr := chi.URLParam(r, "workspaceId")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_WORKSPACE_ID", "workspaceId must be a valid UUID")
		return
	}

	claims, ok := auth.GetClaims(ctx)
	if !ok {
		writeError(w, ctx, log, http.StatusUnauthorized, "UNAUTHORIZED", "authentication claims not found")
		return
	}

	actorID, err := uuid.Parse(claims.ActorID)
	if err != nil {
		log.Error(ctx, "invalid actorID in claims", zap.String("actorId", claims.ActorID), zap.Error(err))
		writeError(w, ctx, log, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid authentication claims")
		return
	}

	var req domain.CreateCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error(ctx, "failed to decode request body", zap.Error(err))
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_REQUEST_BODY", "request body must be valid JSON")
		return
	}

	log.Info(ctx, "creating company",
		zap.String("workspaceId", workspaceID.String()),
		zap.String("actorId", actorID.String()),
		zap.String("name", req.Name),
	)

	company, err := h.service.CreateCompany(ctx, workspaceID, actorID, &req)
	if err != nil {
		handleCompanyServiceError(w, ctx, log, err)
		return
	}

	log.Info(ctx, "company created successfully",
		zap.String("companyId", company.ID.String()),
	)

	writeJSON(w, http.StatusCreated, company)
}

// UpdateCompany handles PATCH /v1/workspaces/{workspaceId}/companies/{companyId}
func (h *CompanyHandler) UpdateCompany(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceIDStr := chi.URLParam(r, "workspaceId")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_WORKSPACE_ID", "workspaceId must be a valid UUID")
		return
	}

	companyIDStr := chi.URLParam(r, "companyId")
	companyID, err := uuid.Parse(companyIDStr)
	if err != nil {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_COMPANY_ID", "companyId must be a valid UUID")
		return
	}

	claims, ok := auth.GetClaims(ctx)
	if !ok {
		writeError(w, ctx, log, http.StatusUnauthorized, "UNAUTHORIZED", "authentication claims not found")
		return
	}

	actorID, err := uuid.Parse(claims.ActorID)
	if err != nil {
		log.Error(ctx, "invalid actorID in claims", zap.String("actorId", claims.ActorID), zap.Error(err))
		writeError(w, ctx, log, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid authentication claims")
		return
	}

	var req domain.UpdateCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error(ctx, "failed to decode request body", zap.Error(err))
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_REQUEST_BODY", "request body must be valid JSON")
		return
	}

	log.Info(ctx, "updating company",
		zap.String("workspaceId", workspaceID.String()),
		zap.String("companyId", companyID.String()),
		zap.String("actorId", actorID.String()),
	)

	company, err := h.service.UpdateCompany(ctx, workspaceID, companyID, actorID, &req)
	if err != nil {
		handleCompanyServiceError(w, ctx, log, err)
		return
	}

	log.Info(ctx, "company updated successfully",
		zap.String("companyId", company.ID.String()),
	)

	writeJSON(w, http.StatusOK, company)
}

// DeleteCompany handles DELETE /v1/workspaces/{workspaceId}/companies/{companyId}
func (h *CompanyHandler) DeleteCompany(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceIDStr := chi.URLParam(r, "workspaceId")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_WORKSPACE_ID", "workspaceId must be a valid UUID")
		return
	}

	companyIDStr := chi.URLParam(r, "companyId")
	companyID, err := uuid.Parse(companyIDStr)
	if err != nil {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_COMPANY_ID", "companyId must be a valid UUID")
		return
	}

	claims, ok := auth.GetClaims(ctx)
	if !ok {
		writeError(w, ctx, log, http.StatusUnauthorized, "UNAUTHORIZED", "authentication claims not found")
		return
	}

	actorID, err := uuid.Parse(claims.ActorID)
	if err != nil {
		log.Error(ctx, "invalid actorID in claims", zap.String("actorId", claims.ActorID), zap.Error(err))
		writeError(w, ctx, log, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid authentication claims")
		return
	}

	log.Info(ctx, "deleting company",
		zap.String("workspaceId", workspaceID.String()),
		zap.String("companyId", companyID.String()),
		zap.String("actorId", actorID.String()),
	)

	err = h.service.DeleteCompany(ctx, workspaceID, companyID, actorID)
	if err != nil {
		handleCompanyServiceError(w, ctx, log, err)
		return
	}

	log.Info(ctx, "company deleted successfully",
		zap.String("companyId", companyID.String()),
	)

	w.WriteHeader(http.StatusNoContent)
}

// handleCompanyServiceError maps service errors to HTTP responses
func handleCompanyServiceError(w http.ResponseWriter, ctx context.Context, log *logger.Logger, err error) {
	switch {
	case errors.Is(err, service.ErrMemberNotFound):
		writeError(w, ctx, log, http.StatusForbidden, "FORBIDDEN", "insufficient permissions for this workspace")
	case errors.Is(err, service.ErrUnauthorized):
		writeError(w, ctx, log, http.StatusForbidden, "FORBIDDEN", "insufficient permissions for this action")
	case errors.Is(err, service.ErrCompanyNotFound):
		writeError(w, ctx, log, http.StatusNotFound, "NOT_FOUND", "company not found")
	case errors.Is(err, service.ErrCompanyDomainConflict):
		writeError(w, ctx, log, http.StatusConflict, "CONFLICT", "company with this domain already exists")
	default:
		log.Error(ctx, "unexpected service error", zap.Error(err))
		writeError(w, ctx, log, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
	}
}
