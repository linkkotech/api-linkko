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

	workspaceID := chi.URLParam(r, "workspaceId")

	claims, ok := auth.GetClaims(ctx)
	if !ok {
		httperr.Unauthorized401(w, ctx, httperr.ErrCodeInvalidToken, "authentication required")
		return
	}

	actorID := claims.ActorID

	params := domain.ListContactsParams{
		Limit: 50, // default
	}

	if cursor := r.URL.Query().Get("cursor"); cursor != "" {
		params.Cursor = &cursor
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 100 {
			httperr.BadRequest400(w, ctx, httperr.ErrCodeInvalidParameter, "limit must be between 1 and 100")
			return
		}
		params.Limit = limit
	}

	if actorId := r.URL.Query().Get("actorId"); actorId != "" {
		params.ActorID = &actorId
	}

	if companyId := r.URL.Query().Get("companyId"); companyId != "" {
		params.CompanyID = &companyId
	}

	if search := r.URL.Query().Get("q"); search != "" {
		params.Query = &search
	}

	log.Info(ctx, "listing contacts",
		zap.String("workspaceId", workspaceID),
		zap.String("actorId", actorID),
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

	workspaceID := chi.URLParam(r, "workspaceId")
	contactID := chi.URLParam(r, "contactId")

	claims, ok := auth.GetClaims(ctx)
	if !ok {
		httperr.Unauthorized401(w, ctx, httperr.ErrCodeInvalidToken, "authentication required")
		return
	}

	actorID := claims.ActorID

	log.Info(ctx, "fetching contact",
		zap.String("workspaceId", workspaceID),
		zap.String("contactId", contactID),
		zap.String("actorId", actorID),
	)

	// Service now fetches role from database internally
	contact, err := h.service.GetContact(ctx, workspaceID, contactID, actorID)
	if err != nil {
		handleServiceError(w, ctx, log, err)
		return
	}

	log.Info(ctx, "contact fetched successfully",
		zap.String("contactId", contact.ID),
	)

	writeJSON(w, http.StatusOK, contact)
}

// CreateContact handles POST /v1/workspaces/{workspaceId}/contacts
func (h *ContactHandler) CreateContact(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")

	claims, ok := auth.GetClaims(ctx)
	if !ok {
		httperr.Unauthorized401(w, ctx, httperr.ErrCodeInvalidToken, "authentication required")
		return
	}

	actorID := claims.ActorID

	var req domain.CreateContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn(ctx, "invalid request body", zap.Error(err))
		httperr.BadRequest400(w, ctx, httperr.ErrCodeInvalidParameter, "request body must be valid JSON")
		return
	}

	if err := req.Validate(); err != nil {
		log.Warn(ctx, "validation failed", zap.Error(err))
		httperr.WriteError(w, ctx, http.StatusUnprocessableEntity, httperr.ErrCodeValidationError, err.Error())
		return
	}

	log.Info(ctx, "creating contact",
		zap.String("workspaceId", workspaceID),
		zap.String("email", req.Email),
		zap.String("actorId", actorID),
	)

	// Service now fetches role from database internally
	contact, err := h.service.CreateContact(ctx, workspaceID, actorID, &req)
	if err != nil {
		handleServiceError(w, ctx, log, err)
		return
	}

	log.Info(ctx, "contact created successfully",
		zap.String("contactId", contact.ID),
		zap.String("email", contact.Email),
	)

	w.Header().Set("Location", "/v1/workspaces/"+workspaceID+"/contacts/"+contact.ID)
	writeJSON(w, http.StatusCreated, contact)
}

// UpdateContact handles PATCH /v1/workspaces/{workspaceId}/contacts/{contactId}
func (h *ContactHandler) UpdateContact(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	contactID := chi.URLParam(r, "contactId")

	claims, ok := auth.GetClaims(ctx)
	if !ok {
		httperr.Unauthorized401(w, ctx, httperr.ErrCodeInvalidToken, "authentication required")
		return
	}

	actorID := claims.ActorID

	var req domain.UpdateContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn(ctx, "invalid request body", zap.Error(err))
		httperr.BadRequest400(w, ctx, httperr.ErrCodeInvalidParameter, "request body must be valid JSON")
		return
	}

	if err := req.Validate(); err != nil {
		log.Warn(ctx, "validation failed", zap.Error(err))
		httperr.WriteError(w, ctx, http.StatusUnprocessableEntity, httperr.ErrCodeValidationError, err.Error())
		return
	}

	log.Info(ctx, "updating contact",
		zap.String("workspaceId", workspaceID),
		zap.String("contactId", contactID),
		zap.String("actorId", actorID),
	)

	// Service now fetches role from database internally
	contact, err := h.service.UpdateContact(ctx, workspaceID, contactID, actorID, &req)
	if err != nil {
		handleServiceError(w, ctx, log, err)
		return
	}

	log.Info(ctx, "contact updated successfully",
		zap.String("contactId", contact.ID),
	)

	writeJSON(w, http.StatusOK, contact)
}

// DeleteContact handles DELETE /v1/workspaces/{workspaceId}/contacts/{contactId}
func (h *ContactHandler) DeleteContact(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	contactID := chi.URLParam(r, "contactId")

	claims, ok := auth.GetClaims(ctx)
	if !ok {
		httperr.Unauthorized401(w, ctx, httperr.ErrCodeInvalidToken, "authentication required")
		return
	}

	actorID := claims.ActorID

	log.Info(ctx, "deleting contact",
		zap.String("workspaceId", workspaceID),
		zap.String("contactId", contactID),
		zap.String("actorId", actorID),
	)

	// Service now fetches role from database internally and validates delete permission
	err := h.service.DeleteContact(ctx, workspaceID, contactID, actorID)
	if err != nil {
		handleServiceError(w, ctx, log, err)
		return
	}

	log.Info(ctx, "contact deleted successfully",
		zap.String("contactId", contactID),
	)

	w.WriteHeader(http.StatusNoContent)
}

// Helper functions for standardized responses

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

func handleServiceError(w http.ResponseWriter, ctx context.Context, log *logger.Logger, err error) {
	switch {
	case errors.Is(err, service.ErrMemberNotFound):
		httperr.Forbidden403(w, ctx, httperr.ErrCodeForbidden, "insufficient permissions for this workspace")
	case errors.Is(err, service.ErrUnauthorized):
		httperr.Forbidden403(w, ctx, httperr.ErrCodeForbidden, "insufficient permissions for this action")
	case errors.Is(err, service.ErrContactNotFound):
		httperr.WriteError(w, ctx, http.StatusNotFound, "NOT_FOUND", "contact not found")
	case errors.Is(err, service.ErrEmailConflict):
		httperr.WriteError(w, ctx, http.StatusConflict, "CONFLICT", "contact with this email already exists")
	case errors.Is(err, service.ErrConcurrencyConflict):
		httperr.WriteError(w, ctx, http.StatusConflict, "CONFLICT", "contact was modified by another request")
	case errors.Is(err, service.ErrInvalidOwner):
		httperr.WriteError(w, ctx, http.StatusUnprocessableEntity, "INVALID_OWNER", "owner does not belong to workspace")
	case errors.Is(err, service.ErrInvalidCompany):
		httperr.WriteError(w, ctx, http.StatusUnprocessableEntity, "INVALID_COMPANY", "company does not belong to workspace")
	default:
		log.Error(ctx, "internal server error", zap.Error(err))
		httperr.InternalError500(w, ctx, "an internal error occurred")
	}
}
