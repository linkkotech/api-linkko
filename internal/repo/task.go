package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"linkko-api/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrTaskNotFound = errors.New("task not found in workspace")
)

type TaskRepository struct {
	pool *pgxpool.Pool
}

func NewTaskRepository(pool *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{pool: pool}
}

// BeginTx inicia uma transação.
// Deve ser usado em conjunto com defer tx.Rollback(ctx) e tx.Commit(ctx).
func (r *TaskRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.pool.Begin(ctx)
}

// List retrieves tasks for a workspace with optional filters.
// Multi-tenant isolation enforced by workspace_id filter.
// Default ordering: position ASC (Kanban order within each status).
func (r *TaskRepository) List(ctx context.Context, params domain.ListTasksParams) ([]domain.Task, string, error) {
	query := `
		SELECT id, workspace_id, title, description, status, priority, type, 
		       position, owner_id, assigned_to, contact_id, 
		       due_date, completed_at, created_at, updated_at, deleted_at
		FROM public."Task"
		WHERE workspace_id = $1 AND deleted_at IS NULL
	`
	args := []interface{}{params.WorkspaceID}
	argIdx := 2

	// Filtros opcionais
	if params.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *params.Status)
		argIdx++
	}

	if params.Priority != nil {
		query += fmt.Sprintf(" AND priority = $%d", argIdx)
		args = append(args, *params.Priority)
		argIdx++
	}

	if params.Type != nil {
		query += fmt.Sprintf(" AND type = $%d", argIdx)
		args = append(args, *params.Type)
		argIdx++
	}

	if params.AssignedTo != nil {
		query += fmt.Sprintf(" AND assigned_to = $%d", argIdx)
		args = append(args, *params.AssignedTo)
		argIdx++
	}

	if params.ActorID != nil {
		query += fmt.Sprintf(" AND owner_id = $%d", argIdx)
		args = append(args, *params.ActorID)
		argIdx++
	}

	if params.ContactID != nil {
		query += fmt.Sprintf(" AND contact_id = $%d", argIdx)
		args = append(args, *params.ContactID)
		argIdx++
	}

	if params.Query != nil && *params.Query != "" {
		query += fmt.Sprintf(" AND to_tsvector('simple', title || ' ' || COALESCE(description, '')) @@ plainto_tsquery('simple', $%d)", argIdx)
		args = append(args, *params.Query)
		argIdx++
	}

	// Cursor-based pagination (default: position ASC for Kanban)
	if params.Cursor != nil && *params.Cursor != "" {
		cursorTime, err := time.Parse(time.RFC3339, *params.Cursor)
		if err != nil {
			return nil, "", fmt.Errorf("invalid cursor format: %w", err)
		}
		query += fmt.Sprintf(" AND created_at < $%d", argIdx)
		args = append(args, cursorTime)
		argIdx++
	}

	// Ordenação (default: position ASC para Kanban)
	query += " ORDER BY position ASC"
	query += fmt.Sprintf(" LIMIT $%d", argIdx)
	args = append(args, params.Limit+1) // +1 to check if there's next page

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("query tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]domain.Task, 0, params.Limit)
	for rows.Next() {
		var t domain.Task
		var deletedAt sql.NullTime
		err := rows.Scan(
			&t.ID, &t.WorkspaceID, &t.Title, &t.Description,
			&t.Status, &t.Priority, &t.Type, &t.Position,
			&t.ActorID, &t.AssignedTo, &t.ContactID,
			&t.DueDate, &t.CompletedAt,
			&t.CreatedAt, &t.UpdatedAt, &deletedAt,
		)
		if err != nil {
			return nil, "", fmt.Errorf("scan task: %w", err)
		}
		if deletedAt.Valid {
			t.DeletedAt = &deletedAt.Time
		}
		tasks = append(tasks, t)
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("iterate tasks: %w", err)
	}

	var nextCursor string
	if len(tasks) > params.Limit {
		nextCursor = tasks[params.Limit-1].CreatedAt.Format(time.RFC3339)
		tasks = tasks[:params.Limit]
	}

	return tasks, nextCursor, nil
}

// Get retrieves a single task by ID, scoped to workspace.
// IDOR protection: returns not found if task exists but belongs to another workspace.
func (r *TaskRepository) Get(ctx context.Context, workspaceID, taskID string) (*domain.Task, error) {
	query := `
		SELECT id, workspace_id, title, description, status, priority, type, 
		       position, owner_id, assigned_to, contact_id, 
		       due_date, completed_at, created_at, updated_at, deleted_at
		FROM public."Task"
		WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
	`

	var t domain.Task
	var deletedAt sql.NullTime
	err := r.pool.QueryRow(ctx, query, taskID, workspaceID).Scan(
		&t.ID, &t.WorkspaceID, &t.Title, &t.Description,
		&t.Status, &t.Priority, &t.Type, &t.Position,
		&t.ActorID, &t.AssignedTo, &t.ContactID,
		&t.DueDate, &t.CompletedAt,
		&t.CreatedAt, &t.UpdatedAt, &deletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("query task: %w", err)
	}

	if deletedAt.Valid {
		t.DeletedAt = &deletedAt.Time
	}

	return &t, nil
}

// GetForUpdate retrieves a task with pessimistic lock (SELECT ... FOR UPDATE).
// MANDATORY para operações de reordenação (Kanban drag-and-drop) para evitar race conditions.
// Deve ser chamado dentro de uma transação.
func (r *TaskRepository) GetForUpdate(ctx context.Context, tx pgx.Tx, workspaceID, taskID string) (*domain.Task, error) {
	query := `
		SELECT id, workspace_id, title, description, status, priority, type, 
		       position, owner_id, assigned_to, contact_id, 
		       due_date, completed_at, created_at, updated_at, deleted_at
		FROM public."Task"
		WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
		FOR UPDATE
	`

	var t domain.Task
	var deletedAt sql.NullTime
	err := tx.QueryRow(ctx, query, taskID, workspaceID).Scan(
		&t.ID, &t.WorkspaceID, &t.Title, &t.Description,
		&t.Status, &t.Priority, &t.Type, &t.Position,
		&t.ActorID, &t.AssignedTo, &t.ContactID,
		&t.DueDate, &t.CompletedAt,
		&t.CreatedAt, &t.UpdatedAt, &deletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("query task for update: %w", err)
	}

	if deletedAt.Valid {
		t.DeletedAt = &deletedAt.Time
	}

	return &t, nil
}

// GetPositionBounds retorna as posições das tarefas vizinhas (before e after) com lock.
// MANDATORY para cálculo de nova position durante drag-and-drop.
// Retorna (posBefore, posAfter, error). Nil = não existe vizinho naquela direção.
func (r *TaskRepository) GetPositionBounds(ctx context.Context, tx pgx.Tx, workspaceID string, status domain.TaskStatus, beforeID, afterID *string) (*float64, *float64, error) {
	var posBefore, posAfter *float64

	// Lock beforeTask se fornecido
	if beforeID != nil {
		query := `
			SELECT position
			FROM public."Task"
			WHERE id = $1 AND workspace_id = $2 AND status = $3 AND deleted_at IS NULL
			FOR UPDATE
		`
		var pos float64
		err := tx.QueryRow(ctx, query, *beforeID, workspaceID, status).Scan(&pos)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, nil, fmt.Errorf("beforeTask not found or wrong status")
			}
			return nil, nil, fmt.Errorf("query beforeTask position: %w", err)
		}
		posBefore = &pos
	}

	// Lock afterTask se fornecido
	if afterID != nil {
		query := `
			SELECT position
			FROM public."Task"
			WHERE id = $1 AND workspace_id = $2 AND status = $3 AND deleted_at IS NULL
			FOR UPDATE
		`
		var pos float64
		err := tx.QueryRow(ctx, query, *afterID, workspaceID, status).Scan(&pos)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, nil, fmt.Errorf("afterTask not found or wrong status")
			}
			return nil, nil, fmt.Errorf("query afterTask position: %w", err)
		}
		posAfter = &pos
	}

	return posBefore, posAfter, nil
}

// Create inserts a new task with workspace isolation.
func (r *TaskRepository) Create(ctx context.Context, task *domain.Task) error {
	query := `
		INSERT INTO public."Task" (id, workspace_id, title, description, status, priority, type, 
		                           position, owner_id, assigned_to, contact_id, due_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := r.pool.Exec(ctx, query,
		task.ID, task.WorkspaceID, task.Title, task.Description,
		task.Status, task.Priority, task.Type, task.Position,
		task.ActorID, task.AssignedTo, task.ContactID, task.DueDate,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23503" { // foreign key violation
				return fmt.Errorf("invalid relationship: contact_id not found")
			}
		}
		return fmt.Errorf("insert task: %w", err)
	}

	return nil
}

// Update atualiza campos de uma tarefa (sem alterar position - usar UpdatePosition).
func (r *TaskRepository) Update(ctx context.Context, workspaceID, taskID string, req *domain.UpdateTaskRequest) error {
	// Dynamic query builder para PATCH semântico
	query := `UPDATE public."Task" SET updated_at = NOW()`
	args := []interface{}{}
	argIdx := 1

	if req.Title != nil {
		query += fmt.Sprintf(", title = $%d", argIdx)
		args = append(args, *req.Title)
		argIdx++
	}

	if req.Description != nil {
		query += fmt.Sprintf(", description = $%d", argIdx)
		args = append(args, *req.Description)
		argIdx++
	}

	if req.Priority != nil {
		query += fmt.Sprintf(", priority = $%d", argIdx)
		args = append(args, *req.Priority)
		argIdx++
	}

	if req.Type != nil {
		query += fmt.Sprintf(", type = $%d", argIdx)
		args = append(args, *req.Type)
		argIdx++
	}

	if req.AssignedTo != nil {
		query += fmt.Sprintf(", assigned_to = $%d", argIdx)
		args = append(args, *req.AssignedTo)
		argIdx++
	}

	if req.ContactID != nil {
		query += fmt.Sprintf(", contact_id = $%d", argIdx)
		args = append(args, *req.ContactID)
		argIdx++
	}

	if req.DueDate != nil {
		query += fmt.Sprintf(", due_date = $%d", argIdx)
		args = append(args, *req.DueDate)
		argIdx++
	}

	if req.CompletedAt != nil {
		query += fmt.Sprintf(", completed_at = $%d", argIdx)
		args = append(args, *req.CompletedAt)
		argIdx++
	}

	// WHERE clause com multi-tenant isolation
	query += fmt.Sprintf(" WHERE id = $%d AND workspace_id = $%d AND deleted_at IS NULL", argIdx, argIdx+1)
	args = append(args, taskID, workspaceID)

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrTaskNotFound
	}

	return nil
}

// UpdatePosition atualiza position e status de uma tarefa (Kanban drag-and-drop).
// MANDATORY: deve ser chamado dentro de uma transação após GetForUpdate/GetPositionBounds.
func (r *TaskRepository) UpdatePosition(ctx context.Context, tx pgx.Tx, workspaceID, taskID string, newPosition float64, newStatus domain.TaskStatus) error {
	query := `
		UPDATE public."Task"
		SET position = $1, status = $2, updated_at = NOW()
		WHERE id = $3 AND workspace_id = $4 AND deleted_at IS NULL
	`

	result, err := tx.Exec(ctx, query, newPosition, newStatus, taskID, workspaceID)
	if err != nil {
		return fmt.Errorf("update task position: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrTaskNotFound
	}

	return nil
}

// SoftDelete marca uma tarefa como deletada (soft delete).
func (r *TaskRepository) SoftDelete(ctx context.Context, workspaceID, taskID string) error {
	query := `
		UPDATE public."Task"
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
	`

	result, err := r.pool.Exec(ctx, query, taskID, workspaceID)
	if err != nil {
		return fmt.Errorf("soft delete task: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrTaskNotFound
	}

	return nil
}

// GetMaxPosition retorna a maior position em um status específico.
// Usado para adicionar novas tarefas ao final da coluna.
func (r *TaskRepository) GetMaxPosition(ctx context.Context, workspaceID string, status domain.TaskStatus) (float64, error) {
	query := `
		SELECT COALESCE(MAX(position), 0)
		FROM public."Task"
		WHERE workspace_id = $1 AND status = $2 AND deleted_at IS NULL
	`

	var maxPos float64
	err := r.pool.QueryRow(ctx, query, workspaceID, status).Scan(&maxPos)
	if err != nil {
		return 0, fmt.Errorf("query max position: %w", err)
	}

	return maxPos, nil
}
