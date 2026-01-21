package handler

import (
	"encoding/json"
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

	params := domain.ListTasksParams{
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

	// Filtros opcionais
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		status := domain.TaskStatus(statusStr)
		if !status.IsValid() {
			writeError(w, ctx, log, http.StatusBadRequest, "INVALID_STATUS", "status must be one of: TODO, IN_PROGRESS, DONE, CANCELLED")
			return
		}
		params.Status = &status
	}

	if priorityStr := r.URL.Query().Get("priority"); priorityStr != "" {
		priority := domain.Priority(priorityStr)
		if !priority.IsValid() {
			writeError(w, ctx, log, http.StatusBadRequest, "INVALID_PRIORITY", "priority must be one of: LOW, MEDIUM, HIGH, URGENT")
			return
		}
		params.Priority = &priority
	}

	if typeStr := r.URL.Query().Get("type"); typeStr != "" {
		taskType := domain.TaskType(typeStr)
		if !taskType.IsValid() {
			writeError(w, ctx, log, http.StatusBadRequest, "INVALID_TYPE", "type must be one of: task, bug, feature, improvement, research")
			return
		}
		params.Type = &taskType
	}

	if assignedToStr := r.URL.Query().Get("assignedTo"); assignedToStr != "" {
		assignedToID, err := uuid.Parse(assignedToStr)
		if err != nil {
			writeError(w, ctx, log, http.StatusBadRequest, "INVALID_ASSIGNED_TO", "assignedTo must be a valid UUID")
			return
		}
		params.AssignedTo = &assignedToID
	}

	if actorIDStr := r.URL.Query().Get("actorId"); actorIDStr != "" {
		actorFilterID, err := uuid.Parse(actorIDStr)
		if err != nil {
			writeError(w, ctx, log, http.StatusBadRequest, "INVALID_ACTOR_ID", "actorId must be a valid UUID")
			return
		}
		params.ActorID = &actorFilterID
	}

	if contactIDStr := r.URL.Query().Get("contactId"); contactIDStr != "" {
		contactID, err := uuid.Parse(contactIDStr)
		if err != nil {
			writeError(w, ctx, log, http.StatusBadRequest, "INVALID_CONTACT_ID", "contactId must be a valid UUID")
			return
		}
		params.ContactID = &contactID
	}

	if search := r.URL.Query().Get("q"); search != "" {
		params.Query = &search
	}

	log.Info(ctx, "listing tasks",
		zap.String("workspaceId", workspaceID.String()),
		zap.String("actorId", actorID.String()),
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

	workspaceIDStr := chi.URLParam(r, "workspaceId")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_WORKSPACE_ID", "workspaceId must be a valid UUID")
		return
	}

	taskIDStr := chi.URLParam(r, "taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_TASK_ID", "taskId must be a valid UUID")
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

	log.Info(ctx, "getting task",
		zap.String("workspaceId", workspaceID.String()),
		zap.String("taskId", taskID.String()),
		zap.String("actorId", actorID.String()),
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

	var req domain.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
		return
	}

	log.Info(ctx, "creating task",
		zap.String("workspaceId", workspaceID.String()),
		zap.String("actorId", actorID.String()),
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

	workspaceIDStr := chi.URLParam(r, "workspaceId")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_WORKSPACE_ID", "workspaceId must be a valid UUID")
		return
	}

	taskIDStr := chi.URLParam(r, "taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_TASK_ID", "taskId must be a valid UUID")
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

	var req domain.UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
		return
	}

	log.Info(ctx, "updating task",
		zap.String("workspaceId", workspaceID.String()),
		zap.String("taskId", taskID.String()),
		zap.String("actorId", actorID.String()),
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

	workspaceIDStr := chi.URLParam(r, "workspaceId")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_WORKSPACE_ID", "workspaceId must be a valid UUID")
		return
	}

	taskIDStr := chi.URLParam(r, "taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_TASK_ID", "taskId must be a valid UUID")
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

	log.Info(ctx, "deleting task",
		zap.String("workspaceId", workspaceID.String()),
		zap.String("taskId", taskID.String()),
		zap.String("actorId", actorID.String()),
	)

	err = h.service.DeleteTask(ctx, workspaceID, taskID, actorID)
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

	workspaceIDStr := chi.URLParam(r, "workspaceId")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_WORKSPACE_ID", "workspaceId must be a valid UUID")
		return
	}

	taskIDStr := chi.URLParam(r, "taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_TASK_ID", "taskId must be a valid UUID")
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

	var req domain.MoveTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
		return
	}

	// Validar status destino
	if !req.ToStatus.IsValid() {
		writeError(w, ctx, log, http.StatusBadRequest, "INVALID_STATUS", "toStatus must be one of: TODO, IN_PROGRESS, DONE, CANCELLED")
		return
	}

	log.Info(ctx, "moving task",
		zap.String("workspaceId", workspaceID.String()),
		zap.String("taskId", taskID.String()),
		zap.String("actorId", actorID.String()),
		zap.String("toStatus", string(req.ToStatus)),
	)

	task, err := h.service.MoveTask(ctx, workspaceID, taskID, actorID, &req)
	if err != nil {
		handleServiceError(w, ctx, log, err)
		return
	}

	writeJSON(w, http.StatusOK, task)
}
