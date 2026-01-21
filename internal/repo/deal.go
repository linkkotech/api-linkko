package repo

import (
	"context"
	"errors"

	"linkko-api/internal/domain"
	"linkko-api/internal/repo/sqlc"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrDealNotFound = errors.New("deal not found in workspace")
)

type DealRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewDealRepository(pool *pgxpool.Pool) *DealRepository {
	return &DealRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// WithTx retorna uma instância do repositório vinculada a uma transação.
func (r *DealRepository) WithTx(tx pgx.Tx) *DealRepository {
	return &DealRepository{
		pool:    r.pool,
		queries: r.queries.WithTx(tx),
	}
}

// BeginTx inicia uma transação no pool.
func (r *DealRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.pool.Begin(ctx)
}

func (r *DealRepository) Create(ctx context.Context, d *domain.Deal) (*domain.Deal, error) {
	params := sqlc.CreateDealParams{
		ID:                d.ID,
		WorkspaceId:       d.WorkspaceID,
		PipelineId:        d.PipelineID,
		StageId:           d.StageID,
		ContactId:         d.ContactID,
		CompanyId:         d.CompanyID,
		Name:              d.Name,
		Value:             d.Value,
		Currency:          d.Currency,
		Stage:             sqlc.DealStage(d.Stage),
		Probability:       d.Probability,
		ExpectedCloseDate: pgtype.Timestamp{Time: getTime(d.ExpectedCloseDate), Valid: d.ExpectedCloseDate != nil},
		OwnerId:           d.OwnerID,
		CreatedById:       d.CreatedByID,
		Description:       d.Description,
	}

	if d.ExpectedCloseDate != nil {
		params.ExpectedCloseDate = pgtype.Timestamp{Time: *d.ExpectedCloseDate, Valid: true}
	}

	row, err := r.queries.CreateDeal(ctx, params)
	if err != nil {
		return nil, err
	}

	return r.sqlcDealToDomain(&row), nil
}

func (r *DealRepository) Get(ctx context.Context, workspaceID, dealID string) (*domain.Deal, error) {
	row, err := r.queries.GetDeal(ctx, sqlc.GetDealParams{
		ID:          dealID,
		WorkspaceId: workspaceID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDealNotFound
		}
		return nil, err
	}

	return r.sqlcGetDealRowToDomain(&row), nil
}

func (r *DealRepository) List(ctx context.Context, workspaceID string, pipelineID, stageID, ownerID *string) ([]domain.Deal, error) {
	rows, err := r.queries.ListDeals(ctx, sqlc.ListDealsParams{
		WorkspaceId: workspaceID,
		PipelineId:  pipelineID,
		StageId:     stageID,
		OwnerId:     ownerID,
	})
	if err != nil {
		return nil, err
	}

	deals := make([]domain.Deal, len(rows))
	for i, row := range rows {
		deals[i] = *r.sqlcListDealsRowToDomain(&row)
	}
	return deals, nil
}

func (r *DealRepository) Update(ctx context.Context, workspaceID, dealID string, d *domain.UpdateDealRequest, updatedByID string) (*domain.Deal, error) {
	params := sqlc.UpdateDealParams{
		ID:          dealID,
		WorkspaceId: workspaceID,
		UpdatedById: &updatedByID,
	}

	if d.Name != nil {
		params.Name = d.Name
	}
	if d.Value != nil {
		params.Value = d.Value
	}
	if d.Currency != nil {
		params.Currency = d.Currency
	}
	if d.Probability != nil {
		params.Probability = d.Probability
	}
	if d.ExpectedCloseDate != nil {
		params.ExpectedCloseDate = pgtype.Timestamp{Time: *d.ExpectedCloseDate, Valid: true}
	}
	if d.Description != nil {
		params.Description = d.Description
	}
	if d.OwnerID != nil {
		params.OwnerId = d.OwnerID
	}

	row, err := r.queries.UpdateDeal(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDealNotFound
		}
		return nil, err
	}

	return r.sqlcDealToDomain(&row), nil
}

func (r *DealRepository) MoveStage(ctx context.Context, workspaceID, dealID string, req *domain.UpdateDealStageRequest, updatedByID string) (*domain.Deal, error) {
	params := sqlc.UpdateDealParams{
		ID:          dealID,
		WorkspaceId: workspaceID,
		StageId:     &req.StageID,
		UpdatedById: &updatedByID,
	}
	

	if req.Stage != nil {
		params.Stage = sqlc.NullDealStage{DealStage: sqlc.DealStage(*req.Stage), Valid: true}
	}
	if req.ClosedAt != nil {
		params.ClosedAt = pgtype.Timestamp{Time: *req.ClosedAt, Valid: true}
	}
	if req.Reason != nil {
		params.LostReason = req.Reason
	}

	row, err := r.queries.UpdateDeal(ctx, params)
	if err != nil {
		return nil, err
	}

	return r.sqlcDealToDomain(&row), nil
}

func (r *DealRepository) CreateHistory(ctx context.Context, h *domain.DealStageHistory) error {
	_, err := r.queries.CreateDealHistory(ctx, sqlc.CreateDealHistoryParams{
		ID:          h.ID,
		WorkspaceId: h.WorkspaceID,
		DealId:      h.DealID,
		FromStage:   sqlc.DealStage(h.FromStage),
		ToStage:     sqlc.DealStage(h.ToStage),
		Reason:      h.Reason,
		UserId:      h.UserID,
	})
	return err
}

// Mappers
func (r *DealRepository) sqlcDealToDomain(row *sqlc.Deal) *domain.Deal {
	return &domain.Deal{
		ID:                row.ID,
		WorkspaceID:       row.WorkspaceId,
		PipelineID:        row.PipelineId,
		StageID:           row.StageId,
		ContactID:         row.ContactId,
		CompanyID:         row.CompanyId,
		Name:              row.Name,
		Value:             row.Value,
		Currency:          row.Currency,
		Stage:             domain.DealStage(row.Stage),
		Probability:       row.Probability,
		ExpectedCloseDate: toTimePtr(row.ExpectedCloseDate),
		ClosedAt:          toTimePtr(row.ClosedAt),
		LostReason:        row.LostReason,
		Description:       row.Description,
		OwnerID:           row.OwnerId,
		CreatedByID:       row.CreatedById,
		UpdatedByID:       row.UpdatedById,
		CreatedAt:         row.CreatedAt.Time,
		UpdatedAt:         row.UpdatedAt.Time,
	}
}

func (r *DealRepository) sqlcGetDealRowToDomain(row *sqlc.GetDealRow) *domain.Deal {
	return &domain.Deal{
		ID:                row.ID,
		WorkspaceID:       row.WorkspaceId,
		PipelineID:        row.PipelineId,
		StageID:           row.StageId,
		ContactID:         row.ContactId,
		CompanyID:         row.CompanyId,
		Name:              row.Name,
		Value:             row.Value,
		Currency:          row.Currency,
		Stage:             domain.DealStage(row.Stage),
		Probability:       row.Probability,
		ExpectedCloseDate: toTimePtr(row.ExpectedCloseDate),
		ClosedAt:          toTimePtr(row.ClosedAt),
		LostReason:        row.LostReason,
		Description:       row.Description,
		OwnerID:           row.OwnerId,
		CreatedByID:       row.CreatedById,
		UpdatedByID:       row.UpdatedById,
		CreatedAt:         row.CreatedAt.Time,
		UpdatedAt:         row.UpdatedAt.Time,
		ContactName:       row.Contactname,
		CompanyName:       row.Companyname,
	}
}

func (r *DealRepository) sqlcListDealsRowToDomain(row *sqlc.ListDealsRow) *domain.Deal {
	return &domain.Deal{
		ID:                row.ID,
		WorkspaceID:       row.WorkspaceId,
		PipelineID:        row.PipelineId,
		StageID:           row.StageId,
		ContactID:         row.ContactId,
		CompanyID:         row.CompanyId,
		Name:              row.Name,
		Value:             row.Value,
		Currency:          row.Currency,
		Stage:             domain.DealStage(row.Stage),
		Probability:       row.Probability,
		ExpectedCloseDate: toTimePtr(row.ExpectedCloseDate),
		ClosedAt:          toTimePtr(row.ClosedAt),
		LostReason:        row.LostReason,
		Description:       row.Description,
		OwnerID:           row.OwnerId,
		CreatedByID:       row.CreatedById,
		UpdatedByID:       row.UpdatedById,
		CreatedAt:         row.CreatedAt.Time,
		UpdatedAt:         row.UpdatedAt.Time,
		ContactName:       row.Contactname,
		CompanyName:       row.Companyname,
	}
}

// Helpers
func toFloat64PtrDeal(f pgtype.Float8) *float64 {
	if f.Valid {
		return &f.Float64
	}
	return nil
}
