package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"linkko-api/internal/auth"
	"linkko-api/internal/domain"
	"linkko-api/internal/observability/logger"
	"linkko-api/internal/service"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type PortfolioHandler struct {
	service *service.PortfolioService
}

func NewPortfolioHandler(service *service.PortfolioService) *PortfolioHandler {
	return &PortfolioHandler{
		service: service,
	}
}

func (h *PortfolioHandler) CreatePortfolioItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	claims, _ := auth.GetClaims(ctx)
	actorID := claims.ActorID

	var req domain.CreatePortfolioItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorPortfolio(w, ctx, log, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
		return
	}

	item, err := h.service.CreatePortfolioItem(ctx, workspaceID, actorID, &req)
	if err != nil {
		handlePortfolioError(w, ctx, log, err)
		return
	}

	writeOKPortfolio(w, http.StatusCreated, item)
}

func (h *PortfolioHandler) GetPortfolioItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	itemID := chi.URLParam(r, "itemID")
	claims, _ := auth.GetClaims(ctx)
	actorID := claims.ActorID

	item, err := h.service.GetPortfolioItem(ctx, workspaceID, itemID, actorID)
	if err != nil {
		handlePortfolioError(w, ctx, log, err)
		return
	}

	writeOKPortfolio(w, http.StatusOK, item)
}

func (h *PortfolioHandler) ListPortfolioItems(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	claims, _ := auth.GetClaims(ctx)
	actorID := claims.ActorID

	// Query params
	statusStr := r.URL.Query().Get("status")
	categoryStr := r.URL.Query().Get("category")
	queryParam := r.URL.Query().Get("q")

	var status *domain.PortfolioStatus
	if statusStr != "" {
		s := domain.PortfolioStatus(statusStr)
		status = &s
	}

	var category *domain.PortfolioCategoryEnum
	if categoryStr != "" {
		c := domain.PortfolioCategoryEnum(categoryStr)
		category = &c
	}

	var query *string
	if queryParam != "" {
		query = &queryParam
	}

	items, err := h.service.ListPortfolioItems(ctx, workspaceID, actorID, status, category, query)
	if err != nil {
		handlePortfolioError(w, ctx, log, err)
		return
	}

	writeOKPortfolio(w, http.StatusOK, items)
}

func (h *PortfolioHandler) UpdatePortfolioItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	itemID := chi.URLParam(r, "itemID")
	claims, _ := auth.GetClaims(ctx)
	actorID := claims.ActorID

	var req domain.UpdatePortfolioItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorPortfolio(w, ctx, log, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
		return
	}

	item, err := h.service.UpdatePortfolioItem(ctx, workspaceID, itemID, actorID, &req)
	if err != nil {
		handlePortfolioError(w, ctx, log, err)
		return
	}

	writeOKPortfolio(w, http.StatusOK, item)
}

func (h *PortfolioHandler) DeletePortfolioItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	itemID := chi.URLParam(r, "itemID")
	claims, _ := auth.GetClaims(ctx)
	actorID := claims.ActorID

	if err := h.service.DeletePortfolioItem(ctx, workspaceID, itemID, actorID); err != nil {
		handlePortfolioError(w, ctx, log, err)
		return
	}

	writeOKPortfolio(w, http.StatusOK, map[string]bool{"deleted": true})
}

// Helpers
func writeOKPortfolio(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":   true,
		"data": data,
	})
}

func writeErrorPortfolio(w http.ResponseWriter, ctx context.Context, log *logger.Logger, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok": false,
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

func handlePortfolioError(w http.ResponseWriter, ctx context.Context, log *logger.Logger, err error) {
	switch {
	case errors.Is(err, service.ErrUnauthorized):
		writeErrorPortfolio(w, ctx, log, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
	default:
		log.Error(ctx, "internal error", zap.Error(err))
		writeErrorPortfolio(w, ctx, log, http.StatusInternalServerError, "INTERNAL_ERROR", "an internal error occurred")
	}
}
