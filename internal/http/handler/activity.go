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

type ActivityHandler struct {
	service *service.ActivityService
}

func NewActivityHandler(service *service.ActivityService) *ActivityHandler {
	return &ActivityHandler{service: service}
}

func (h *ActivityHandler) CreateNote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	claims, _ := auth.GetClaims(ctx)
	actorID := claims.ActorID

	var req domain.CreateNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httperr.BadRequest400(w, ctx, httperr.ErrCodeInvalidParameter, "invalid JSON body")
		return
	}

	note, err := h.service.CreateNote(ctx, workspaceID, actorID, &req)
	if err != nil {
		handleActivityError(w, ctx, log, err)
		return
	}

	writeOK(w, http.StatusCreated, note)
}

func (h *ActivityHandler) CreateCall(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	claims, _ := auth.GetClaims(ctx)
	actorID := claims.ActorID

	var req domain.CreateCallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httperr.BadRequest400(w, ctx, httperr.ErrCodeInvalidParameter, "invalid JSON body")
		return
	}

	call, err := h.service.CreateCall(ctx, workspaceID, actorID, &req)
	if err != nil {
		handleActivityError(w, ctx, log, err)
		return
	}

	writeOK(w, http.StatusCreated, call)
}

func (h *ActivityHandler) ListTimeline(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	claims, _ := auth.GetClaims(ctx)
	actorID := claims.ActorID

	contactID := r.URL.Query().Get("contactId")
	companyID := r.URL.Query().Get("companyId")
	dealID := r.URL.Query().Get("dealId")

	var ctID, cpID, dID *string
	if contactID != "" { ctID = &contactID }
	if companyID != "" { cpID = &companyID }
	if dealID != "" { dID = &dealID }

	activities, err := h.service.ListTimeline(ctx, workspaceID, actorID, ctID, cpID, dID)
	if err != nil {
		handleActivityError(w, ctx, log, err)
		return
	}

	writeOK(w, http.StatusOK, activities)
}

// Helpers
func handleActivityError(w http.ResponseWriter, ctx context.Context, log *logger.Logger, err error) {
	switch {
	case errors.Is(err, service.ErrUnauthorized):
		httperr.Forbidden403(w, ctx, httperr.ErrCodeForbidden, "insufficient permissions")
	default:
		log.Error(ctx, "internal error", zap.Error(err))
		httperr.InternalError500(w, ctx, "an internal error occurred")
	}
}
