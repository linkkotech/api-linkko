package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"linkko-api/internal/domain"
	"linkko-api/internal/auth"
	"linkko-api/internal/http/httperr"
	"linkko-api/internal/observability/logger"
	"linkko-api/internal/service"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type DealHandler struct {
	service *service.DealService
}

func NewDealHandler(service *service.DealService) *DealHandler {
	return &DealHandler{service: service}
}

func (h *DealHandler) CreateDeal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	claims, _ := auth.GetClaims(ctx)
	actorID := claims.ActorID

	var req domain.CreateDealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httperr.BadRequest400(w, ctx, httperr.ErrCodeInvalidParameter, "invalid JSON body")
		return
	}

	deal, err := h.service.CreateDeal(ctx, workspaceID, actorID, &req)
	if err != nil {
		handleDealError(w, ctx, log, err)
		return
	}

	writeOK(w, http.StatusCreated, deal)
}

func (h *DealHandler) GetDeal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	dealID := chi.URLParam(r, "dealId")
	claims, _ := auth.GetClaims(ctx)
	actorID := claims.ActorID

	deal, err := h.service.GetDeal(ctx, workspaceID, dealID, actorID)
	if err != nil {
		handleDealError(w, ctx, log, err)
		return
	}

	writeOK(w, http.StatusOK, deal)
}

func (h *DealHandler) ListDeals(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	claims, _ := auth.GetClaims(ctx)
	actorID := claims.ActorID

	pipelineID := r.URL.Query().Get("pipelineId")
	stageID := r.URL.Query().Get("stageId")
	ownerID := r.URL.Query().Get("ownerId")

	var pID, sID, oID *string
	if pipelineID != "" { pID = &pipelineID }
	if stageID != "" { sID = &stageID }
	if ownerID != "" { oID = &ownerID }

	deals, err := h.service.ListDeals(ctx, workspaceID, actorID, pID, sID, oID)
	if err != nil {
		handleDealError(w, ctx, log, err)
		return
	}

	writeOK(w, http.StatusOK, deals)
}

func (h *DealHandler) UpdateDeal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	dealID := chi.URLParam(r, "dealId")
	claims, _ := auth.GetClaims(ctx)
	actorID := claims.ActorID

	var req domain.UpdateDealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httperr.BadRequest400(w, ctx, httperr.ErrCodeInvalidParameter, "invalid JSON body")
		return
	}

	deal, err := h.service.UpdateDeal(ctx, workspaceID, dealID, actorID, &req)
	if err != nil {
		handleDealError(w, ctx, log, err)
		return
	}

	writeOK(w, http.StatusOK, deal)
}

func (h *DealHandler) UpdateDealStage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	dealID := chi.URLParam(r, "dealId")
	claims, _ := auth.GetClaims(ctx)
	actorID := claims.ActorID

	var req domain.UpdateDealStageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httperr.BadRequest400(w, ctx, httperr.ErrCodeInvalidParameter, "invalid JSON body")
		return
	}

	deal, err := h.service.UpdateDealStage(ctx, workspaceID, dealID, actorID, &req)
	if err != nil {
		handleDealError(w, ctx, log, err)
		return
	}

	writeOK(w, http.StatusOK, deal)
}

// Helpers
func writeOK(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":   true,
		"data": data,
	})
}

func handleDealError(w http.ResponseWriter, ctx context.Context, log *logger.Logger, err error) {
	switch {
	case errors.Is(err, service.ErrDealNotFound):
		httperr.WriteError(w, ctx, http.StatusNotFound, "NOT_FOUND", "deal not found")
	case errors.Is(err, service.ErrUnauthorized):
		httperr.Forbidden403(w, ctx, httperr.ErrCodeForbidden, "insufficient permissions")
	default:
		log.Error(ctx, "internal error", zap.Error(err))
		httperr.InternalError500(w, ctx, "an internal error occurred")
	}
}

