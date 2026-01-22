package handler

import (
	"encoding/json"
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

type TaskHandler struct {
	service *service.TaskService
}

func NewTaskHandler(service *service.TaskService) *TaskHandler {
	return &TaskHandler{service: service}
}

// ListTasks handles GET /v1/workspaces/{workspaceId}/tasks
func (h *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
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

	params := domain.ListTasksParams{
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

	// Filtros opcionais
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		status := domain.TaskStatus(statusStr)
		if !status.IsValid() {
			httperr.BadRequest400(w, ctx, httperr.ErrCodeInvalidParameter, "status must be one of: TODO, IN_PROGRESS, DONE, CANCELLED")
			return
		}
		params.Status = &status
	}

	if priorityStr := r.URL.Query().Get("priority"); priorityStr != "" {
		priority := domain.Priority(priorityStr)
		if !priority.IsValid() {
			httperr.BadRequest400(w, ctx, httperr.ErrCodeInvalidParameter, "priority must be one of: LOW, MEDIUM, HIGH, URGENT")
			return
		}
		params.Priority = &priority
	}

	if typeStr := r.URL.Query().Get("type"); typeStr != "" {
		taskType := domain.TaskType(typeStr)
		if !taskType.IsValid() {
			httperr.BadRequest400(w, ctx, httperr.ErrCodeInvalidParameter, "type must be one of: task, bug, feature, improvement, research")
			return
		}
		params.Type = &taskType
	}

	if assignedToID := r.URL.Query().Get("assignedTo"); assignedToID != "" {
		params.AssignedTo = &assignedToID
	}

	if actorFilterID := r.URL.Query().Get("actorId"); actorFilterID != "" {
		params.ActorID = &actorFilterID
	}

	if contactID := r.URL.Query().Get("contactId"); contactID != "" {
		params.ContactID = &contactID
	}

	if search := r.URL.Query().Get("q"); search != "" {
		params.Query = &search
	}

	log.Info(ctx, "listing tasks",
		zap.String("workspaceId", workspaceID),
		zap.String("actorId", actorID),
		zap.Int("limit", params.Limit),
	)

	response, err := h.service.ListTasks(ctx, workspaceID, actorID, params)
	if err != nil {
		handleServiceError(w, ctx, log, err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

// GetTask handles GET /v1/workspaces/{workspaceId}/tasks/{taskId}
func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	taskID := chi.URLParam(r, "taskId")
	if workspaceID == "" || taskID == "" {
		httperr.BadRequest400(w, ctx, httperr.ErrCodeInvalidParameter, "workspaceId and taskId are required")
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

	log.Info(ctx, "getting task",
		zap.String("workspaceId", workspaceID),
		zap.String("taskId", taskID),
		zap.String("actorId", actorID),
	)

	task, err := h.service.GetTask(ctx, workspaceID, taskID, actorID)
	if err != nil {
		handleServiceError(w, ctx, log, err)
		return
	}

	writeJSON(w, http.StatusOK, task)
}

// CreateTask handles POST /v1/workspaces/{workspaceId}/tasks
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
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

	var req domain.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httperr.BadRequest400(w, ctx, httperr.ErrCodeInvalidParameter, "invalid JSON body")
		return
	}

	log.Info(ctx, "creating task",
		zap.String("workspaceId", workspaceID),
		zap.String("actorId", actorID),
		zap.String("title", req.Title),
	)

	task, err := h.service.CreateTask(ctx, workspaceID, actorID, &req)
	if err != nil {
		handleServiceError(w, ctx, log, err)
		return
	}

	writeJSON(w, http.StatusCreated, task)
}

// UpdateTask handles PATCH /v1/workspaces/{workspaceId}/tasks/{taskId}
func (h *TaskHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	taskID := chi.URLParam(r, "taskId")
	if workspaceID == "" || taskID == "" {
		httperr.BadRequest400(w, ctx, httperr.ErrCodeInvalidParameter, "workspaceId and taskId are required")
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

	var req domain.UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httperr.BadRequest400(w, ctx, httperr.ErrCodeInvalidParameter, "invalid JSON body")
		return
	}

	log.Info(ctx, "updating task",
		zap.String("workspaceId", workspaceID),
		zap.String("taskId", taskID),
		zap.String("actorId", actorID),
	)

	task, err := h.service.UpdateTask(ctx, workspaceID, taskID, actorID, &req)
	if err != nil {
		handleServiceError(w, ctx, log, err)
		return
	}

	writeJSON(w, http.StatusOK, task)
}

// DeleteTask handles DELETE /v1/workspaces/{workspaceId}/tasks/{taskId}
func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	taskID := chi.URLParam(r, "taskId")
	if workspaceID == "" || taskID == "" {
		httperr.BadRequest400(w, ctx, httperr.ErrCodeInvalidParameter, "workspaceId and taskId are required")
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

	log.Info(ctx, "deleting task",
		zap.String("workspaceId", workspaceID),
		zap.String("taskId", taskID),
		zap.String("actorId", actorID),
	)

	err := h.service.DeleteTask(ctx, workspaceID, taskID, actorID)
	if err != nil {
		handleServiceError(w, ctx, log, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// MoveTask handles POST /v1/workspaces/{workspaceId}/tasks/{taskId}:move
// Kanban drag-and-drop com fractional positioning.
func (h *TaskHandler) MoveTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	workspaceID := chi.URLParam(r, "workspaceId")
	taskID := chi.URLParam(r, "taskId")
	if workspaceID == "" || taskID == "" {
		httperr.BadRequest400(w, ctx, httperr.ErrCodeInvalidParameter, "workspaceId and taskId are required")
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

	var req domain.MoveTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httperr.BadRequest400(w, ctx, httperr.ErrCodeInvalidParameter, "invalid JSON body")
		return
	}

	// Validar status destino
	if !req.ToStatus.IsValid() {
		httperr.BadRequest400(w, ctx, httperr.ErrCodeInvalidParameter, "toStatus must be one of: TODO, IN_PROGRESS, DONE, CANCELLED")
		return
	}

	log.Info(ctx, "moving task",
		zap.String("workspaceId", workspaceID),
		zap.String("taskId", taskID),
		zap.String("actorId", actorID),
		zap.String("toStatus", string(req.ToStatus)),
	)

	task, err := h.service.MoveTask(ctx, workspaceID, taskID, actorID, &req)
	if err != nil {
		handleServiceError(w, ctx, log, err)
		return
	}

	writeJSON(w, http.StatusOK, task)
}
