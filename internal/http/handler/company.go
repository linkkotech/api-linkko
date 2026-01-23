package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"linkko-api/internal/auth"
	"linkko-api/internal/domain"
	"linkko-api/internal/http/httperr"
	"linkko-api/internal/observability/logger"
	"linkko-api/internal/service"

	"github.com/go-chi/chi/v5"
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

	workspaceID := chi.URLParam(r, "workspaceId")
	if workspaceID == "" {
		httperr.BadRequest400(w, ctx, httperr.ErrCodeInvalidParameter, "workspaceId is required")
		return
	}

	claims, ok := auth.GetClaims(ctx)
	if !ok {
		httperr.Unauthorized401(w, ctx, httperr.ErrCodeInvalidToken, "authentication claims not found")
		return
	}

	actorID := claims.ActorID
	if actorID == "" {
		httperr.Unauthorized401(w, ctx, httperr.ErrCodeInvalidToken, "actorID not found in claims")
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
			httperr.BadRequest400(w, ctx, httperr.ErrCodeInvalidParameter, "limit must be between 1 and 100")
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
			httperr.BadRequest400(w, ctx, httperr.ErrCodeInvalidParameter, "invalid lifecycleStage value")
			return
		}
		params.LifecycleStage = &lifecycleStage
	}

	if sizeStr := r.URL.Query().Get("companySize"); sizeStr != "" {
		companySize := domain.CompanySize(sizeStr)
		if !companySize.IsValid() {
			httperr.BadRequest400(w, ctx, httperr.ErrCodeInvalidParameter, "invalid companySize value")
			return
		}
		params.Size = &companySize
	}

	if industry := r.URL.Query().Get("industry"); industry != "" {
		params.Industry = &industry
	}

	if ownerID := r.URL.Query().Get("ownerId"); ownerID != "" {
		params.OwnerID = &ownerID
	}

	if search := r.URL.Query().Get("q"); search != "" {
		params.Query = &search
	}

	log.Info(ctx, "listing companies",
		zap.String("workspaceId", workspaceID),
		zap.String("actorId", actorID),
		zap.Int("limit", params.Limit),
	)

	response, err := h.service.ListCompanies(ctx, workspaceID, actorID, params)
	if err != nil {
		handleCompanyServiceError(w, ctx, log, err)
		return
	}

	log.Info(ctx, "companies listed successfully",
		zap.String("workspaceId", workspaceID),
		zap.Int("count", len(response.Data)),
		zap.Bool("hasNextPage", response.Meta.HasNextPage),
	)

	writeJSON(w, http.StatusOK, response)
}

// GetCompany handles GET /v1/workspaces/{workspaceId}/companies/{companyId}
func (h *CompanyHandler) GetCompany(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	companyID := chi.URLParam(r, "companyId")
	if workspaceID == "" || companyID == "" {
		httperr.BadRequest400(w, ctx, httperr.ErrCodeInvalidParameter, "workspaceId and companyId are required")
		return
	}

	claims, ok := auth.GetClaims(ctx)
	if !ok {
		httperr.Unauthorized401(w, ctx, httperr.ErrCodeInvalidToken, "authentication claims not found")
		return
	}

	actorID := claims.ActorID

	log.Info(ctx, "fetching company",
		zap.String("workspaceId", workspaceID),
		zap.String("companyId", companyID),
		zap.String("actorId", actorID),
	)

	company, err := h.service.GetCompany(ctx, workspaceID, companyID, actorID)
	if err != nil {
		handleCompanyServiceError(w, ctx, log, err)
		return
	}

	log.Info(ctx, "company fetched successfully",
		zap.String("companyId", company.ID),
	)

	writeJSON(w, http.StatusOK, company)
}

// CreateCompany handles POST /v1/workspaces/{workspaceId}/companies
func (h *CompanyHandler) CreateCompany(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	if workspaceID == "" {
		httperr.BadRequest400(w, ctx, httperr.ErrCodeInvalidParameter, "workspaceId is required")
		return
	}

	claims, ok := auth.GetClaims(ctx)
	if !ok {
		httperr.Unauthorized401(w, ctx, httperr.ErrCodeInvalidToken, "authentication claims not found")
		return
	}

	actorID := claims.ActorID
	if actorID == "" {
		httperr.Unauthorized401(w, ctx, httperr.ErrCodeInvalidToken, "actorID not found in claims")
		return
	}

	var req domain.CreateCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error(ctx, "failed to decode request body", zap.Error(err))
		httperr.BadRequest400(w, ctx, httperr.ErrCodeInvalidParameter, "request body must be valid JSON")
		return
	}

	log.Info(ctx, "creating company",
		zap.String("workspaceId", workspaceID),
		zap.String("actorId", actorID),
		zap.String("name", req.Name),
	)

	company, err := h.service.CreateCompany(ctx, workspaceID, actorID, &req)
	if err != nil {
		handleCompanyServiceError(w, ctx, log, err)
		return
	}

	log.Info(ctx, "company created successfully",
		zap.String("companyId", company.ID),
	)

	writeJSON(w, http.StatusCreated, company)
}

// UpdateCompany handles PATCH /v1/workspaces/{workspaceId}/companies/{companyId}
func (h *CompanyHandler) UpdateCompany(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	companyID := chi.URLParam(r, "companyId")
	if workspaceID == "" || companyID == "" {
		httperr.BadRequest400(w, ctx, httperr.ErrCodeInvalidParameter, "workspaceId and companyId are required")
		return
	}

	claims, ok := auth.GetClaims(ctx)
	if !ok {
		httperr.Unauthorized401(w, ctx, httperr.ErrCodeInvalidToken, "authentication claims not found")
		return
	}

	actorID := claims.ActorID
	if actorID == "" {
		httperr.Unauthorized401(w, ctx, httperr.ErrCodeInvalidToken, "actorID not found in claims")
		return
	}

	var req domain.UpdateCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error(ctx, "failed to decode request body", zap.Error(err))
		httperr.BadRequest400(w, ctx, httperr.ErrCodeInvalidParameter, "request body must be valid JSON")
		return
	}

	log.Info(ctx, "updating company",
		zap.String("workspaceId", workspaceID),
		zap.String("companyId", companyID),
		zap.String("actorId", actorID),
	)

	company, err := h.service.UpdateCompany(ctx, workspaceID, companyID, actorID, &req)
	if err != nil {
		handleCompanyServiceError(w, ctx, log, err)
		return
	}

	log.Info(ctx, "company updated successfully",
		zap.String("companyId", company.ID),
	)

	writeJSON(w, http.StatusOK, company)
}

// DeleteCompany handles DELETE /v1/workspaces/{workspaceId}/companies/{companyId}
func (h *CompanyHandler) DeleteCompany(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	companyID := chi.URLParam(r, "companyId")
	if workspaceID == "" || companyID == "" {
		httperr.BadRequest400(w, ctx, httperr.ErrCodeInvalidParameter, "workspaceId and companyId are required")
		return
	}

	claims, ok := auth.GetClaims(ctx)
	if !ok {
		httperr.Unauthorized401(w, ctx, httperr.ErrCodeInvalidToken, "authentication claims not found")
		return
	}

	actorID := claims.ActorID
	if actorID == "" {
		httperr.Unauthorized401(w, ctx, httperr.ErrCodeInvalidToken, "actorID not found in claims")
		return
	}

	log.Info(ctx, "deleting company",
		zap.String("workspaceId", workspaceID),
		zap.String("companyId", companyID),
		zap.String("actorId", actorID),
	)

	err := h.service.DeleteCompany(ctx, workspaceID, companyID, actorID)
	if err != nil {
		handleCompanyServiceError(w, ctx, log, err)
		return
	}

	log.Info(ctx, "company deleted successfully",
		zap.String("companyId", companyID),
	)

	w.WriteHeader(http.StatusNoContent)
}

// handleCompanyServiceError maps service errors to HTTP responses
func handleCompanyServiceError(w http.ResponseWriter, ctx context.Context, log *logger.Logger, err error) {
	// Tarefa B: Capturar o erro real para observabilidade
	logger.SetRootError(ctx, err)

	switch {
	case errors.Is(err, service.ErrMemberNotFound):
		httperr.Forbidden403(w, ctx, httperr.ErrCodeForbidden, "insufficient permissions for this workspace")
	case errors.Is(err, service.ErrUnauthorized):
		httperr.Forbidden403(w, ctx, httperr.ErrCodeForbidden, "insufficient permissions for this action")
	case errors.Is(err, service.ErrCompanyNotFound):
		httperr.WriteError(w, ctx, http.StatusNotFound, httperr.ErrCodeNotFound, "company not found")
	case errors.Is(err, service.ErrCompanyDomainConflict):
		httperr.WriteError(w, ctx, http.StatusConflict, httperr.ErrCodeConflict, "company with this domain already exists")
	default:
		log.Error(ctx, "unexpected service error", zap.Error(err))
		httperr.InternalError(w, ctx)
	}
}
