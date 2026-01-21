package repo

import (
	"context"

	"linkko-api/internal/domain"
	"linkko-api/internal/repo/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ActivityRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewActivityRepository(pool *pgxpool.Pool) *ActivityRepository {
	return &ActivityRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

func (r *ActivityRepository) CreateActivity(ctx context.Context, a *domain.Activity) (*domain.Activity, error) {
	params := sqlc.CreateActivityParams{
		ID:           a.ID,
		WorkspaceId:  a.WorkspaceID,
		CompanyId:    a.CompanyID,
		ContactId:    a.ContactID,
		DealId:       a.DealID,
		ActivityType: sqlc.ActivityType(a.Type),
		ActivityId:   a.ActivityID,
		UserId:       a.UserID,
		Metadata:     a.Metadata,
	}

	row, err := r.queries.CreateActivity(ctx, params)
	if err != nil {
		return nil, err
	}

	return r.sqlcActivityToDomain(&row), nil
}

func (r *ActivityRepository) CreateNote(ctx context.Context, n *domain.Note) (*domain.Note, error) {
	params := sqlc.CreateNoteParams{
		ID:          n.ID,
		WorkspaceId: n.WorkspaceID,
		CompanyId:   n.CompanyID,
		ContactId:   n.ContactID,
		DealId:      n.DealID,
		Content:     n.Content,
		IsPinned:    n.IsPinned,
		UserId:      n.UserID,
	}

	row, err := r.queries.CreateNote(ctx, params)
	if err != nil {
		return nil, err
	}

	return r.sqlcNoteToDomain(&row), nil
}

func (r *ActivityRepository) CreateCall(ctx context.Context, c *domain.Call) (*domain.Call, error) {
	params := sqlc.CreateCallParams{
		ID:           c.ID,
		WorkspaceId:  c.WorkspaceID,
		ContactId:    c.ContactID,
		CompanyId:    c.CompanyID,
		Direction:    sqlc.MessageDirection(c.Direction),
		Duration:     c.Duration,
		RecordingUrl: c.RecordingURL,
		Summary:      c.Summary,
		UserId:       c.UserID,
		CalledAt:     pgtype.Timestamp{Time: c.CalledAt, Valid: true},
	}

	row, err := r.queries.CreateCall(ctx, params)
	if err != nil {
		return nil, err
	}

	return r.sqlcCallToDomain(&row), nil
}

func (r *ActivityRepository) List(ctx context.Context, workspaceID string, contactID, companyID, dealID *string) ([]domain.Activity, error) {
	rows, err := r.queries.ListActivities(ctx, sqlc.ListActivitiesParams{
		WorkspaceId: workspaceID,
		ContactId:   contactID,
		CompanyId:   companyID,
		DealId:      dealID,
	})
	if err != nil {
		return nil, err
	}

	activities := make([]domain.Activity, len(rows))
	for i, row := range rows {
		activities[i] = *r.sqlcActivityToDomain(&row)
	}
	return activities, nil
}

// Mappers
func (r *ActivityRepository) sqlcActivityToDomain(row *sqlc.Activity) *domain.Activity {
	return &domain.Activity{
		ID:           row.ID,
		WorkspaceID:  row.WorkspaceId,
		CompanyID:    row.CompanyId,
		ContactID:    row.ContactId,
		DealID:       row.DealId,
		Type:         domain.ActivityType(row.ActivityType),
		ActivityID:   row.ActivityId,
		UserID:       row.UserId,
		Metadata:     row.Metadata,
		CreatedAt:    row.CreatedAt.Time,
	}
}

func (r *ActivityRepository) sqlcNoteToDomain(row *sqlc.Note) *domain.Note {
	return &domain.Note{
		ID:          row.ID,
		WorkspaceID: row.WorkspaceId,
		CompanyID:   row.CompanyId,
		ContactID:   row.ContactId,
		DealID:      row.DealId,
		Content:     row.Content,
		IsPinned:    row.IsPinned,
		UserID:      row.UserId,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
		DeletedAt:   toTimePtr(row.DeletedAt),
	}
}

func (r *ActivityRepository) sqlcCallToDomain(row *sqlc.Call) *domain.Call {
	return &domain.Call{
		ID:           row.ID,
		WorkspaceID:  row.WorkspaceId,
		ContactID:    row.ContactId,
		CompanyID:    row.CompanyId,
		Direction:    domain.MessageDirection(row.Direction),
		Duration:     row.Duration,
		RecordingURL: row.RecordingUrl,
		Summary:      row.Summary,
		UserID:       row.UserId,
		CalledAt:     row.CalledAt.Time,
		CreatedAt:    row.CreatedAt.Time,
	}
}
