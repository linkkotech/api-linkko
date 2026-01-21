package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"linkko-api/internal/domain"
	"linkko-api/internal/auth"
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
		writeErrorActivity(w, ctx, log, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
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
		writeErrorActivity(w, ctx, log, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
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
func writeErrorActivity(w http.ResponseWriter, ctx context.Context, log *logger.Logger, status int, code, message string) {
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

func handleActivityError(w http.ResponseWriter, ctx context.Context, log *logger.Logger, err error) {
	switch {
	case errors.Is(err, service.ErrUnauthorized):
		writeErrorActivity(w, ctx, log, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
	default:
		log.Error(ctx, "internal error", zap.Error(err))
		writeErrorActivity(w, ctx, log, http.StatusInternalServerError, "INTERNAL_ERROR", "an internal error occurred")
	}
}
