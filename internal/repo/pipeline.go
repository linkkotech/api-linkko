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
	ErrPipelineNotFound      = errors.New("pipeline not found in workspace")
	ErrPipelineNameConflict  = errors.New("pipeline with this name already exists in workspace")
	ErrStageNotFound         = errors.New("stage not found in pipeline")
	ErrStageNameConflict     = errors.New("stage with this name already exists in pipeline")
	ErrDefaultPipelineExists = errors.New("another pipeline is already set as default")
)

type PipelineRepository struct {
	pool *pgxpool.Pool
}

func NewPipelineRepository(pool *pgxpool.Pool) *PipelineRepository {
	return &PipelineRepository{pool: pool}
}

// BeginTx inicia uma transação.
func (r *PipelineRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.pool.Begin(ctx)
}

// List retrieves pipelines for a workspace with optional filters.
// IMPORTANT: Uses camelCase column names with double quotes.
func (r *PipelineRepository) List(ctx context.Context, params domain.ListPipelinesParams) ([]domain.Pipeline, string, error) {
	query := `
		SELECT id, "workspaceId", name, description, "isDefault",
		       "createdAt", "updatedAt", "deletedAt"
		FROM public."Pipeline"
		WHERE "workspaceId" = $1 AND "deletedAt" IS NULL
	`
	args := []interface{}{params.WorkspaceID}
	argIdx := 2

	// Filtros opcionais
	if params.IsDefault != nil {
		query += fmt.Sprintf(` AND "isDefault" = $%d`, argIdx)
		args = append(args, *params.IsDefault)
		argIdx++
	}

	// Busca textual
	if params.Query != nil && *params.Query != "" {
		query += fmt.Sprintf(` AND to_tsvector('simple', name || ' ' || COALESCE(description, '')) @@ plainto_tsquery('simple', $%d)`, argIdx)
		args = append(args, *params.Query)
		argIdx++
	}

	// Cursor-based pagination
	if params.Cursor != nil && *params.Cursor != "" {
		cursorTime, err := time.Parse(time.RFC3339, *params.Cursor)
		if err != nil {
			return nil, "", fmt.Errorf("invalid cursor format: %w", err)
		}
		query += fmt.Sprintf(` AND "createdAt" < $%d`, argIdx)
		args = append(args, cursorTime)
		argIdx++
	}

	query += ` ORDER BY "createdAt" DESC`
	query += fmt.Sprintf(` LIMIT $%d`, argIdx)
	args = append(args, params.Limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("query pipelines: %w", err)
	}
	defer rows.Close()

	pipelines := make([]domain.Pipeline, 0, params.Limit)
	for rows.Next() {
		var p domain.Pipeline
		var deletedAt sql.NullTime
		err := rows.Scan(
			&p.ID, &p.WorkspaceID, &p.Name, &p.Description, &p.IsDefault,
			&p.CreatedAt, &p.UpdatedAt, &deletedAt,
		)
		if err != nil {
			return nil, "", fmt.Errorf("scan pipeline: %w", err)
		}
		if deletedAt.Valid {
			p.DeletedAt = &deletedAt.Time
		}
		pipelines = append(pipelines, p)
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("iterate pipelines: %w", err)
	}

	// Load stages if requested
	if params.IncludeStages && len(pipelines) > 0 {
		for i := range pipelines {
			stages, err := r.ListStagesByPipeline(ctx, pipelines[i].WorkspaceID, &pipelines[i].ID)
			if err != nil {
				return nil, "", fmt.Errorf("load stages for pipeline %s: %w", pipelines[i].ID, err)
			}
			pipelines[i].Stages = stages
		}
	}

	var nextCursor string
	if len(pipelines) > params.Limit {
		nextCursor = pipelines[params.Limit-1].CreatedAt.Format(time.RFC3339)
		pipelines = pipelines[:params.Limit]
	}

	return pipelines, nextCursor, nil
}

// Get retrieves a single pipeline by ID, scoped to workspace.
func (r *PipelineRepository) Get(ctx context.Context, workspaceID, pipelineID string) (*domain.Pipeline, error) {
	query := `
		SELECT id, "workspaceId", name, description, "isDefault",
		       "createdAt", "updatedAt", "deletedAt"
		FROM public."Pipeline"
		WHERE id = $1 AND "workspaceId" = $2 AND "deletedAt" IS NULL
	`

	var p domain.Pipeline
	var deletedAt sql.NullTime
	err := r.pool.QueryRow(ctx, query, pipelineID, workspaceID).Scan(
		&p.ID, &p.WorkspaceID, &p.Name, &p.Description, &p.IsDefault,
		&p.CreatedAt, &p.UpdatedAt, &deletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPipelineNotFound
		}
		return nil, fmt.Errorf("query pipeline: %w", err)
	}

	if deletedAt.Valid {
		p.DeletedAt = &deletedAt.Time
	}

	return &p, nil
}

// GetWithStages retrieves pipeline with all its stages ordered by orderIndex.
func (r *PipelineRepository) GetWithStages(ctx context.Context, workspaceID, pipelineID string) (*domain.Pipeline, error) {
	pipeline, err := r.Get(ctx, workspaceID, pipelineID)
	if err != nil {
		return nil, err
	}

	stages, err := r.ListStagesByPipeline(ctx, workspaceID, &pipelineID)
	if err != nil {
		return nil, fmt.Errorf("load stages: %w", err)
	}

	pipeline.Stages = stages
	return pipeline, nil
}

// Create inserts a new pipeline with workspace isolation.
func (r *PipelineRepository) Create(ctx context.Context, pipeline *domain.Pipeline) error {
	query := `
		INSERT INTO public."Pipeline" (
			id, "workspaceId", name, description, "isDefault"
		)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.pool.Exec(ctx, query,
		pipeline.ID, pipeline.WorkspaceID, pipeline.Name, pipeline.Description, pipeline.IsDefault,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" { // unique_violation
				if pgErr.ConstraintName == "unique_pipeline_name_per_workspace" {
					return ErrPipelineNameConflict
				}
				if pgErr.ConstraintName == "idx_unique_default_pipeline" {
					return ErrDefaultPipelineExists
				}
			}
		}
		return fmt.Errorf("insert pipeline: %w", err)
	}

	return nil
}

// SetAsDefault marca um pipeline como default (transação: desativa outros defaults + ativa novo).
// MANDATORY: deve ser chamado dentro de uma transação.
func (r *PipelineRepository) SetAsDefault(ctx context.Context, tx pgx.Tx, workspaceID, pipelineID string) error {
	// Step 1: Desativar todos os defaults do workspace
	updateAllQuery := `
		UPDATE public."Pipeline"
		SET "isDefault" = false, "updatedAt" = NOW()
		WHERE "workspaceId" = $1 AND "isDefault" = true AND "deletedAt" IS NULL
	`
	_, err := tx.Exec(ctx, updateAllQuery, workspaceID)
	if err != nil {
		return fmt.Errorf("deactivate existing defaults: %w", err)
	}

	// Step 2: Ativar o novo default
	updateNewQuery := `
		UPDATE public."Pipeline"
		SET "isDefault" = true, "updatedAt" = NOW()
		WHERE id = $1 AND "workspaceId" = $2 AND "deletedAt" IS NULL
	`
	result, err := tx.Exec(ctx, updateNewQuery, pipelineID, workspaceID)
	if err != nil {
		return fmt.Errorf("set new default: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrPipelineNotFound
	}

	return nil
}

// Update atualiza campos de um pipeline (PATCH semântico).
func (r *PipelineRepository) Update(ctx context.Context, workspaceID, pipelineID string, req *domain.UpdatePipelineRequest) error {
	query := `UPDATE public."Pipeline" SET "updatedAt" = NOW()`
	args := []interface{}{}
	argIdx := 1

	if req.Name != nil {
		query += fmt.Sprintf(`, name = $%d`, argIdx)
		args = append(args, *req.Name)
		argIdx++
	}

	if req.Description != nil {
		query += fmt.Sprintf(`, description = $%d`, argIdx)
		args = append(args, *req.Description)
		argIdx++
	}

	query += fmt.Sprintf(` WHERE id = $%d AND "workspaceId" = $%d AND "deletedAt" IS NULL`, argIdx, argIdx+1)
	args = append(args, pipelineID, workspaceID)

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" && pgErr.ConstraintName == "unique_pipeline_name_per_workspace" {
				return ErrPipelineNameConflict
			}
		}
		return fmt.Errorf("update pipeline: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrPipelineNotFound
	}

	return nil
}

// SoftDelete marca um pipeline como deletado (CASCADE deleta stages via FK).
func (r *PipelineRepository) SoftDelete(ctx context.Context, workspaceID, pipelineID string) error {
	query := `
		UPDATE public."Pipeline"
		SET "deletedAt" = NOW(), "updatedAt" = NOW()
		WHERE id = $1 AND "workspaceId" = $2 AND "deletedAt" IS NULL
	`

	result, err := r.pool.Exec(ctx, query, pipelineID, workspaceID)
	if err != nil {
		return fmt.Errorf("soft delete pipeline: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrPipelineNotFound
	}

	return nil
}

// ===== PIPELINE STAGE METHODS =====

// ListStagesByPipeline retorna todos os stages de um pipeline ordenados por orderIndex.
func (r *PipelineRepository) ListStagesByPipeline(ctx context.Context, workspaceID string, pipelineID *string) ([]domain.PipelineStage, error) {
	query := `
		SELECT id, "workspaceId", "pipelineId", name, description, "group", "type", color,
		       "isLocked", "orderIndex", "createdAt", "updatedAt", "deletedAt"
		FROM public."PipelineStage"
		WHERE "workspaceId" = $1
	`
	args := []interface{}{workspaceID}

	if pipelineID != nil {
		query += ` AND "pipelineId" = $2`
		args = append(args, *pipelineID)
	}

	query += ` AND "deletedAt" IS NULL ORDER BY "orderIndex" ASC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query stages: %w", err)
	}
	defer rows.Close()

	stages := make([]domain.PipelineStage, 0)
	for rows.Next() {
		var s domain.PipelineStage
		var deletedAt sql.NullTime
		err := rows.Scan(
			&s.ID, &s.WorkspaceID, &s.PipelineID, &s.Name, &s.Description,
			&s.Group, &s.Type, &s.Color, &s.IsLocked, &s.OrderIndex,
			&s.CreatedAt, &s.UpdatedAt, &deletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan stage: %w", err)
		}
		if deletedAt.Valid {
			s.DeletedAt = &deletedAt.Time
		}
		stages = append(stages, s)
	}

	return stages, rows.Err()
}

// GetStage retrieves a single stage by ID.
func (r *PipelineRepository) GetStage(ctx context.Context, stageID string) (*domain.PipelineStage, error) {
	query := `
		SELECT id, "workspaceId", "pipelineId", name, description, "group", "type", color,
		       "isLocked", "orderIndex", "createdAt", "updatedAt", "deletedAt"
		FROM public."PipelineStage"
		WHERE id = $1 AND "deletedAt" IS NULL
	`

	var s domain.PipelineStage
	var deletedAt sql.NullTime
	err := r.pool.QueryRow(ctx, query, stageID).Scan(
		&s.ID, &s.WorkspaceID, &s.PipelineID, &s.Name, &s.Description,
		&s.Group, &s.Type, &s.Color, &s.IsLocked, &s.OrderIndex,
		&s.CreatedAt, &s.UpdatedAt, &deletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStageNotFound
		}
		return nil, fmt.Errorf("query stage: %w", err)
	}

	if deletedAt.Valid {
		s.DeletedAt = &deletedAt.Time
	}

	return &s, nil
}

// CreateStage inserts a new stage.
func (r *PipelineRepository) CreateStage(ctx context.Context, stage *domain.PipelineStage) error {
	query := `
		INSERT INTO public."PipelineStage" (
			id, "workspaceId", "pipelineId", name, description, "group", "type", color, "isLocked", "orderIndex"
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.pool.Exec(ctx, query,
		stage.ID, stage.WorkspaceID, stage.PipelineID, stage.Name, stage.Description,
		stage.Group, stage.Type, stage.Color, stage.IsLocked, stage.OrderIndex,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" && pgErr.ConstraintName == "unique_stage_name_per_pipeline" {
				return ErrStageNameConflict
			}
			if pgErr.Code == "23503" { // foreign key violation
				return ErrPipelineNotFound
			}
		}
		return fmt.Errorf("insert stage: %w", err)
	}

	return nil
}

// UpdateStage atualiza campos de um stage (PATCH semântico).
func (r *PipelineRepository) UpdateStage(ctx context.Context, stageID string, req *domain.UpdateStageRequest) error {
	query := `UPDATE public."PipelineStage" SET "updatedAt" = NOW()`
	args := []interface{}{}
	argIdx := 1

	if req.Name != nil {
		query += fmt.Sprintf(`, name = $%d`, argIdx)
		args = append(args, *req.Name)
		argIdx++
	}

	if req.Description != nil {
		query += fmt.Sprintf(`, description = $%d`, argIdx)
		args = append(args, *req.Description)
		argIdx++
	}

	if req.Group != nil {
		query += fmt.Sprintf(`, "group" = $%d`, argIdx)
		args = append(args, *req.Group)
		argIdx++
	}

	if req.Type != nil {
		query += fmt.Sprintf(`, "type" = $%d`, argIdx)
		args = append(args, *req.Type)
		argIdx++
	}

	if req.OrderIndex != nil {
		query += fmt.Sprintf(`, "orderIndex" = $%d`, argIdx)
		args = append(args, *req.OrderIndex)
		argIdx++
	}

	if req.Color != nil {
		query += fmt.Sprintf(`, color = $%d`, argIdx)
		args = append(args, *req.Color)
		argIdx++
	}

	if req.IsLocked != nil {
		query += fmt.Sprintf(`, "isLocked" = $%d`, argIdx)
		args = append(args, *req.IsLocked)
		argIdx++
	}

	query += fmt.Sprintf(` WHERE id = $%d AND "deletedAt" IS NULL`, argIdx)
	args = append(args, stageID)

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" && pgErr.ConstraintName == "unique_stage_name_per_pipeline" {
				return ErrStageNameConflict
			}
		}
		return fmt.Errorf("update stage: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrStageNotFound
	}

	return nil
}

// SoftDeleteStage marca um stage como deletado.
func (r *PipelineRepository) SoftDeleteStage(ctx context.Context, stageID string) error {
	query := `
		UPDATE public."PipelineStage"
		SET "deletedAt" = NOW(), "updatedAt" = NOW()
		WHERE id = $1 AND "deletedAt" IS NULL
	`

	result, err := r.pool.Exec(ctx, query, stageID)
	if err != nil {
		return fmt.Errorf("soft delete stage: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrStageNotFound
	}

	return nil
}

// GetMaxOrderIndex retorna o maior orderIndex em um pipeline (para adicionar novos stages no final).
func (r *PipelineRepository) GetMaxOrderIndex(ctx context.Context, pipelineID string) (int, error) {
	query := `
		SELECT COALESCE(MAX("orderIndex"), 0)
		FROM public."PipelineStage"
		WHERE "pipelineId" = $1 AND "deletedAt" IS NULL
	`

	var maxOrder int
	err := r.pool.QueryRow(ctx, query, pipelineID).Scan(&maxOrder)
	if err != nil {
		return 0, fmt.Errorf("query max order: %w", err)
	}

	return maxOrder, nil
}
