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

type ContactHandler struct {
	service *service.ContactService
}

func NewContactHandler(service *service.ContactService) *ContactHandler {
	return &ContactHandler{service: service}
}

// ListContacts handles GET /v1/workspaces/{workspaceId}/contacts
func (h *ContactHandler) ListContacts(w http.ResponseWriter, r *http.Request) {
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

	params := domain.ListContactsParams{
		Limit: 50, // default
	}

	if cursor := r.URL.Query().Get("cursor"); cursor != "" {
		params.Cursor = &cursor
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 100 {
			writeError(w, ctx, log, http.StatusBadRequest, "INVALID_LIMIT", "limit must be between 1 and 100")
			return
		}
		params.Limit = limit
	}

	if actorIdStr := r.URL.Query().Get("actorId"); actorIdStr != "" {
		actorFilterID, err := uuid.Parse(actorIdStr)
		if err != nil {
			writeError(w, ctx, log, http.StatusBadRequest, "INVALID_ACTOR_ID", "actorId must be a valid UUID")
			return
		}
		params.ActorID = &actorFilterID
	}

	if companyIdStr := r.URL.Query().Get("companyId"); companyIdStr != "" {
		companyID, err := uuid.Parse(companyIdStr)
		if err != nil {
			writeError(w, ctx, log, http.StatusBadRequest, "INVALID_COMPANY_ID", "companyId must be a valid UUID")
			return
		}
		params.CompanyID = &companyID
	}

	if search := r.URL.Query().Get("q"); search != "" {
		params.Query = &search
	}

	log.Info(ctx, "listing contacts",
		zap.String("workspaceId", workspaceID.String()),
		zap.String("actorId", actorID.String()),
		zap.Int("limit", params.Limit),
	)

	// Service now fetches role from database internally
	response, err := h.service.ListContacts(ctx, workspaceID, actorID, params)
	if err != nil {
		handleServiceError(w, ctx, log, err)
		return
	}

	log.Info(ctx, "contacts listed successfully",
		zap.Int("count", len(response.Data)),
		zap.Bool("hasNextPage", response.Meta.HasNextPage),
	)

	writeJSON(w, http.StatusOK, response)
}

// GetContact handles GET /v1/workspaces/{workspaceId}/contacts/{contactId}
func (h *ContactHandler) GetContact(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceIDStr := chi.URLParam(r, "workspaceId")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_WORKSPACE_ID", "workspaceId must be a valid UUID")
		return
	}

	contactIDStr := chi.URLParam(r, "contactId")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_CONTACT_ID", "contactId must be a valid UUID")
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

	log.Info(ctx, "fetching contact",
		zap.String("workspaceId", workspaceID.String()),
		zap.String("contactId", contactID.String()),
		zap.String("actorId", actorID.String()),
	)

	// Service now fetches role from database internally
	contact, err := h.service.GetContact(ctx, workspaceID, contactID, actorID)
	if err != nil {
		handleServiceError(w, ctx, log, err)
		return
	}

	log.Info(ctx, "contact fetched successfully",
		zap.String("contactId", contact.ID.String()),
	)

	writeJSON(w, http.StatusOK, contact)
}

// CreateContact handles POST /v1/workspaces/{workspaceId}/contacts
func (h *ContactHandler) CreateContact(w http.ResponseWriter, r *http.Request) {
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

	var req domain.CreateContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn(ctx, "invalid request body", zap.Error(err))
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_REQUEST", "request body must be valid JSON")
		return
	}

	if err := req.Validate(); err != nil {
		log.Warn(ctx, "validation failed", zap.Error(err))
		writeError(w, ctx, log, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}

	log.Info(ctx, "creating contact",
		zap.String("workspaceId", workspaceID.String()),
		zap.String("email", req.Email),
		zap.String("actorId", actorID.String()),
	)

	// Service now fetches role from database internally
	contact, err := h.service.CreateContact(ctx, workspaceID, actorID, &req)
	if err != nil {
		handleServiceError(w, ctx, log, err)
		return
	}

	log.Info(ctx, "contact created successfully",
		zap.String("contactId", contact.ID.String()),
		zap.String("email", contact.Email),
	)

	w.Header().Set("Location", "/v1/workspaces/"+workspaceID.String()+"/contacts/"+contact.ID.String())
	writeJSON(w, http.StatusCreated, contact)
}

// UpdateContact handles PATCH /v1/workspaces/{workspaceId}/contacts/{contactId}
func (h *ContactHandler) UpdateContact(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceIDStr := chi.URLParam(r, "workspaceId")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_WORKSPACE_ID", "workspaceId must be a valid UUID")
		return
	}

	contactIDStr := chi.URLParam(r, "contactId")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_CONTACT_ID", "contactId must be a valid UUID")
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

	var req domain.UpdateContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn(ctx, "invalid request body", zap.Error(err))
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_REQUEST", "request body must be valid JSON")
		return
	}

	if err := req.Validate(); err != nil {
		log.Warn(ctx, "validation failed", zap.Error(err))
		writeError(w, ctx, log, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}

	log.Info(ctx, "updating contact",
		zap.String("workspaceId", workspaceID.String()),
		zap.String("contactId", contactID.String()),
		zap.String("actorId", actorID.String()),
	)

	// Service now fetches role from database internally
	contact, err := h.service.UpdateContact(ctx, workspaceID, contactID, actorID, &req)
	if err != nil {
		handleServiceError(w, ctx, log, err)
		return
	}

	log.Info(ctx, "contact updated successfully",
		zap.String("contactId", contact.ID.String()),
	)

	writeJSON(w, http.StatusOK, contact)
}

// DeleteContact handles DELETE /v1/workspaces/{workspaceId}/contacts/{contactId}
func (h *ContactHandler) DeleteContact(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceIDStr := chi.URLParam(r, "workspaceId")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_WORKSPACE_ID", "workspaceId must be a valid UUID")
		return
	}

	contactIDStr := chi.URLParam(r, "contactId")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_CONTACT_ID", "contactId must be a valid UUID")
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

	log.Info(ctx, "deleting contact",
		zap.String("workspaceId", workspaceID.String()),
		zap.String("contactId", contactID.String()),
		zap.String("actorId", actorID.String()),
	)

	// Service now fetches role from database internally and validates delete permission
	err = h.service.DeleteContact(ctx, workspaceID, contactID, actorID)
	if err != nil {
		handleServiceError(w, ctx, log, err)
		return
	}

	log.Info(ctx, "contact deleted successfully",
		zap.String("contactId", contactID.String()),
	)

	w.WriteHeader(http.StatusNoContent)
}

// Helper functions for standardized responses

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

func writeError(w http.ResponseWriter, ctx context.Context, log *logger.Logger, status int, code, message string) {
	log.Error(ctx, "request failed",
		zap.Int("statusCode", status),
		zap.String("errorCode", code),
		zap.String("message", message),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{
		Code:    code,
		Message: message,
	})
}

func handleServiceError(w http.ResponseWriter, ctx context.Context, log *logger.Logger, err error) {
	switch {
	case errors.Is(err, service.ErrMemberNotFound):
		writeError(w, ctx, log, http.StatusForbidden, "FORBIDDEN", "insufficient permissions for this workspace")
	case errors.Is(err, service.ErrUnauthorized):
		writeError(w, ctx, log, http.StatusForbidden, "FORBIDDEN", "insufficient permissions for this action")
	case errors.Is(err, service.ErrContactNotFound):
		writeError(w, ctx, log, http.StatusNotFound, "NOT_FOUND", "contact not found")
	case errors.Is(err, service.ErrEmailConflict):
		writeError(w, ctx, log, http.StatusConflict, "CONFLICT", "contact with this email already exists")
	case errors.Is(err, service.ErrConcurrencyConflict):
		writeError(w, ctx, log, http.StatusConflict, "CONFLICT", "contact was modified by another request")
	case errors.Is(err, service.ErrInvalidOwner):
		writeError(w, ctx, log, http.StatusUnprocessableEntity, "INVALID_OWNER", "owner does not belong to workspace")
	case errors.Is(err, service.ErrInvalidCompany):
		writeError(w, ctx, log, http.StatusUnprocessableEntity, "INVALID_COMPANY", "company does not belong to workspace")
	default:
		log.Error(ctx, "internal server error", zap.Error(err))
		writeError(w, ctx, log, http.StatusInternalServerError, "INTERNAL_ERROR", "an internal error occurred")
	}
}
