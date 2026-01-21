package repo

import (
	"context"

	"linkko-api/internal/domain"
	"linkko-api/internal/repo/sqlc"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PortfolioRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewPortfolioRepository(pool *pgxpool.Pool) *PortfolioRepository {
	return &PortfolioRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

func (r *PortfolioRepository) Create(ctx context.Context, item *domain.PortfolioItem) (*domain.PortfolioItem, error) {
	row, err := r.queries.CreatePortfolioItem(ctx, sqlc.CreatePortfolioItemParams{
		ID:          item.ID,
		WorkspaceId: item.WorkspaceID,
		Name:        item.Name,
		Description: item.Description,
		Sku:         item.SKU,
		Category:    sqlc.PortfolioCategoryEnum(item.Category),
		Vertical:    sqlc.PortfolioVertical(item.Vertical),
		Status:      sqlc.PortfolioStatus(item.Status),
		Visibility:  sqlc.PortfolioVisibility(item.Visibility),
		BasePrice:   item.BasePrice,
		Currency:    item.Currency,
		ImageUrl:    item.ImageURL,
		Metadata:    item.Metadata,
		Tags:        item.Tags,
		CreatedById: item.CreatedByID,
	})
	if err != nil {
		return nil, err
	}
	return r.sqlcPortfolioToDomain(&row), nil
}

func (r *PortfolioRepository) Get(ctx context.Context, workspaceID, id string) (*domain.PortfolioItem, error) {
	row, err := r.queries.GetPortfolioItem(ctx, sqlc.GetPortfolioItemParams{
		WorkspaceId: workspaceID,
		ID:          id,
	})
	if err != nil {
		return nil, err
	}
	return r.sqlcPortfolioToDomain(&row), nil
}

func (r *PortfolioRepository) List(ctx context.Context, workspaceID string, status *domain.PortfolioStatus, category *domain.PortfolioCategoryEnum, query *string) ([]domain.PortfolioItem, error) {
	var sqlcStatus sqlc.NullPortfolioStatus
	if status != nil {
		sqlcStatus = sqlc.NullPortfolioStatus{
			PortfolioStatus: sqlc.PortfolioStatus(*status),
			Valid:           true,
		}
	}

	var sqlcCategory sqlc.NullPortfolioCategoryEnum
	if category != nil {
		sqlcCategory = sqlc.NullPortfolioCategoryEnum{
			PortfolioCategoryEnum: sqlc.PortfolioCategoryEnum(*category),
			Valid:                 true,
		}
	}

	rows, err := r.queries.ListPortfolioItems(ctx, sqlc.ListPortfolioItemsParams{
		WorkspaceId: workspaceID,
		Status:      sqlcStatus,
		Category:    sqlcCategory,
		Query:       query,
	})
	if err != nil {
		return nil, err
	}

	items := make([]domain.PortfolioItem, len(rows))
	for i, row := range rows {
		items[i] = *r.sqlcPortfolioToDomain(&row)
	}
	return items, nil
}

func (r *PortfolioRepository) Update(ctx context.Context, workspaceID, id string, req *domain.UpdatePortfolioItemRequest, actorID string) (*domain.PortfolioItem, error) {
	params := sqlc.UpdatePortfolioItemParams{
		WorkspaceId: workspaceID,
		ID:          id,
		UpdatedById: &actorID,
	}

	if req.Name != nil {
		params.Name = req.Name
	}
	if req.Description != nil {
		params.Description = req.Description
	}
	if req.SKU != nil {
		params.Sku = req.SKU
	}
	if req.Category != nil {
		params.Category = sqlc.NullPortfolioCategoryEnum{
			PortfolioCategoryEnum: sqlc.PortfolioCategoryEnum(*req.Category),
			Valid:                 true,
		}
	}
	if req.Vertical != nil {
		params.Vertical = sqlc.NullPortfolioVertical{
			PortfolioVertical: sqlc.PortfolioVertical(*req.Vertical),
			Valid:             true,
		}
	}
	if req.Status != nil {
		params.Status = sqlc.NullPortfolioStatus{
			PortfolioStatus: sqlc.PortfolioStatus(*req.Status),
			Valid:           true,
		}
	}
	if req.Visibility != nil {
		params.Visibility = sqlc.NullPortfolioVisibility{
			PortfolioVisibility: sqlc.PortfolioVisibility(*req.Visibility),
			Valid:               true,
		}
	}
	if req.BasePrice != nil {
		params.BasePrice = req.BasePrice
	}
	if req.Currency != nil {
		params.Currency = req.Currency
	}
	if req.ImageURL != nil {
		params.ImageUrl = req.ImageURL
	}
	if req.Metadata != nil {
		params.Metadata = req.Metadata
	}
	if req.Tags != nil {
		params.Tags = req.Tags
	}

	row, err := r.queries.UpdatePortfolioItem(ctx, params)
	if err != nil {
		return nil, err
	}
	return r.sqlcPortfolioToDomain(&row), nil
}

func (r *PortfolioRepository) Delete(ctx context.Context, workspaceID, id string) error {
	return r.queries.DeletePortfolioItem(ctx, sqlc.DeletePortfolioItemParams{
		WorkspaceId: workspaceID,
		ID:          id,
	})
}

// Mapper
func (r *PortfolioRepository) sqlcPortfolioToDomain(row *sqlc.PortfolioItem) *domain.PortfolioItem {
	return &domain.PortfolioItem{
		ID:          row.ID,
		WorkspaceID: row.WorkspaceId,
		Name:        row.Name,
		Description: row.Description,
		SKU:         row.Sku,
		Category:    domain.PortfolioCategoryEnum(row.Category),
		Vertical:    domain.PortfolioVertical(row.Vertical),
		Status:      domain.PortfolioStatus(row.Status),
		Visibility:  domain.PortfolioVisibility(row.Visibility),
		BasePrice:   row.BasePrice,
		Currency:    row.Currency,
		ImageURL:    row.ImageUrl,
		Metadata:    row.Metadata,
		Tags:        row.Tags,
		CreatedByID: row.CreatedById,
		UpdatedByID: row.UpdatedById,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
		DeletedAt:   *row.DeletedAt,
	}
}
