package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"linkko-api/internal/domain"
	"linkko-api/internal/repo/sqlc"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrContactNotFound      = errors.New("contact not found in workspace")
	ErrContactEmailConflict = errors.New("contact with this email already exists in workspace")
)

type ContactRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewContactRepository(pool *pgxpool.Pool) *ContactRepository {
	return &ContactRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// Helper: converte sqlc row para domain.Contact
func sqlcRowToDomainContact(row interface{}) *domain.Contact {
	var c domain.Contact

	switch r := row.(type) {
	case sqlc.GetContactRow:
		c.ID = r.ID
		c.WorkspaceID = r.WorkspaceId
		c.FullName = r.FullName
		if r.Email != nil {
			c.Email = *r.Email
		}
		c.Phone = r.Phone
		if r.OwnerId != nil {
			c.ActorID = *r.OwnerId
		}
		c.CompanyID = r.CompanyId
		c.Tags = r.TagLabels
		// TODO: converter SocialUrls ([]byte) para map[string]interface{}
		c.CustomFields = make(map[string]interface{})
		c.CreatedAt = r.CreatedAt.Time
		c.UpdatedAt = r.UpdatedAt.Time
		if r.DeletedAt.Valid {
			c.DeletedAt = &r.DeletedAt.Time
		}
	case sqlc.CreateContactRow:
		c.ID = r.ID
		c.WorkspaceID = r.WorkspaceId
		c.FullName = r.FullName
		if r.Email != nil {
			c.Email = *r.Email
		}
		c.Phone = r.Phone
		if r.OwnerId != nil {
			c.ActorID = *r.OwnerId
		}
		c.CompanyID = r.CompanyId
		c.Tags = r.TagLabels
		c.CustomFields = make(map[string]interface{})
		c.CreatedAt = r.CreatedAt.Time
		c.UpdatedAt = r.UpdatedAt.Time
		if r.DeletedAt.Valid {
			c.DeletedAt = &r.DeletedAt.Time
		}
	case sqlc.UpdateContactRow:
		c.ID = r.ID
		c.WorkspaceID = r.WorkspaceId
		c.FullName = r.FullName
		if r.Email != nil {
			c.Email = *r.Email
		}
		c.Phone = r.Phone
		if r.OwnerId != nil {
			c.ActorID = *r.OwnerId
		}
		c.CompanyID = r.CompanyId
		c.Tags = r.TagLabels
		c.CustomFields = make(map[string]interface{})
		c.CreatedAt = r.CreatedAt.Time
		c.UpdatedAt = r.UpdatedAt.Time
		if r.DeletedAt.Valid {
			c.DeletedAt = &r.DeletedAt.Time
		}
	case sqlc.ListContactsRow:
		c.ID = r.ID
		c.WorkspaceID = r.WorkspaceId
		c.FullName = r.FullName
		if r.Email != nil {
			c.Email = *r.Email
		}
		c.Phone = r.Phone
		if r.OwnerId != nil {
			c.ActorID = *r.OwnerId
		}
		c.CompanyID = r.CompanyId
		c.Tags = r.TagLabels
		c.CustomFields = make(map[string]interface{})
		c.CreatedAt = r.CreatedAt.Time
		c.UpdatedAt = r.UpdatedAt.Time
		if r.DeletedAt.Valid {
			c.DeletedAt = &r.DeletedAt.Time
		}
	}

	return &c
}

// List retrieves contacts for a workspace with cursor-based pagination.
// Multi-tenant isolation enforced by workspace_id filter.
func (r *ContactRepository) List(ctx context.Context, params domain.ListContactsParams) ([]domain.Contact, string, error) {
	// Preparar parâmetros opcionais
	var ownerID, companyID, lifecycleStage, queryText string
	var cursorTime pgtype.Timestamp

	if params.ActorID != nil {
		ownerID = *params.ActorID
	}
	if params.CompanyID != nil {
		companyID = *params.CompanyID
	}
	if params.Cursor != nil && *params.Cursor != "" {
		t, err := time.Parse(time.RFC3339, *params.Cursor)
		if err != nil {
			return nil, "", fmt.Errorf("invalid cursor format: %w", err)
		}
		cursorTime = pgtype.Timestamp{Time: t, Valid: true}
	}
	if params.Query != nil {
		queryText = *params.Query
	}

	// Chamar SQLc query
	rows, err := r.queries.ListContacts(ctx, sqlc.ListContactsParams{
		WorkspaceId: params.WorkspaceID,
		Column2:     ownerID,
		Column3:     companyID,
		Column4:     lifecycleStage,
		Column5:     queryText,
		Column6:     cursorTime,
		Limit:       int32(params.Limit + 1), // +1 para detectar se há próxima página
	})
	if err != nil {
		return nil, "", fmt.Errorf("query contacts: %w", err)
	}

	// Converter para domain.Contact
	contacts := make([]domain.Contact, 0, params.Limit)
	for _, row := range rows {
		c := sqlcRowToDomainContact(row)
		contacts = append(contacts, *c)
	}

	// Calcular nextCursor
	var nextCursor string
	if len(contacts) > params.Limit {
		nextCursor = contacts[params.Limit-1].CreatedAt.Format(time.RFC3339)
		contacts = contacts[:params.Limit]
	}

	return contacts, nextCursor, nil
}

// Get retrieves a single contact by ID, scoped to workspace.
// IDOR protection: returns not found if contact exists but belongs to another workspace.
func (r *ContactRepository) Get(ctx context.Context, workspaceID, contactID string) (*domain.Contact, error) {
	row, err := r.queries.GetContact(ctx, sqlc.GetContactParams{
		ID:          contactID,
		WorkspaceId: workspaceID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrContactNotFound
		}
		return nil, fmt.Errorf("query contact: %w", err)
	}

	return sqlcRowToDomainContact(row), nil
}

// Create inserts a new contact with workspace isolation.
func (r *ContactRepository) Create(ctx context.Context, contact *domain.Contact) error {
	row, err := r.queries.CreateContact(ctx, sqlc.CreateContactParams{
		ID:                contact.ID,
		FullName:          contact.FullName,
		WorkspaceId:       contact.WorkspaceID,
		Email:             &contact.Email,
		Phone:             contact.Phone,
		Whatsapp:          nil,
		Notes:             nil,
		FirstName:         nil,
		LastName:          nil,
		Image:             nil,
		LinkedinUrl:       nil,
		Language:          nil,
		Timezone:          nil,
		City:              nil,
		State:             nil,
		Country:           nil,
		JobTitle:          nil,
		Department:        nil,
		DecisionRole:      nil,
		TagLabels:         contact.Tags,
		Source:            nil,
		LastInteractionAt: pgtype.Timestamp{},
		OwnerId:           &contact.ActorID,
		SocialUrls:        nil, // TODO: converter map para JSONB
		CompanyId:         contact.CompanyID,
		ContactScore:      0,
		LifecycleStage:    sqlc.ContactLifecycleStageLEAD,
		AssignedToId:      nil,
		CreatedById:       &contact.ActorID,
		UpdatedById:       &contact.ActorID,
		CreatedAt:         pgtype.Timestamp{Time: contact.CreatedAt, Valid: true},
		UpdatedAt:         pgtype.Timestamp{Time: contact.UpdatedAt, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("insert contact: %w", err)
	}

	// Atualizar contact com valores retornados
	contact.CreatedAt = row.CreatedAt.Time
	contact.UpdatedAt = row.UpdatedAt.Time

	return nil
}

// Update modifies an existing contact with optimistic concurrency control.
// Only updates non-nil fields from the request.
func (r *ContactRepository) Update(ctx context.Context, workspaceID, contactID string, updates *domain.UpdateContactRequest, expectedUpdatedAt time.Time) (*domain.Contact, error) {
	now := time.Now()

	// Converter Tags opcional
	var tagLabels []string
	if updates.Tags != nil {
		tagLabels = *updates.Tags
	}

	row, err := r.queries.UpdateContact(ctx, sqlc.UpdateContactParams{
		ID:                contactID,
		WorkspaceId:       workspaceID,
		FullName:          getStringOrEmpty(updates.FullName),
		Email:             updates.Email,
		Phone:             updates.Phone,
		Whatsapp:          nil,
		Notes:             nil,
		FirstName:         nil,
		LastName:          nil,
		Image:             nil,
		LinkedinUrl:       nil,
		Language:          nil,
		Timezone:          nil,
		City:              nil,
		State:             nil,
		Country:           nil,
		JobTitle:          nil,
		Department:        nil,
		DecisionRole:      nil,
		TagLabels:         tagLabels,
		Source:            nil,
		LastInteractionAt: pgtype.Timestamp{},
		OwnerId:           updates.ActorID,
		SocialUrls:        nil, // TODO: converter map para JSONB
		CompanyId:         updates.CompanyID,
		ContactScore:      0,
		LifecycleStage:    sqlc.ContactLifecycleStageLEAD,
		AssignedToId:      nil,
		UpdatedById:       updates.ActorID,
		UpdatedAt:         pgtype.Timestamp{Time: now, Valid: true},
		UpdatedAt_2:       pgtype.Timestamp{Time: expectedUpdatedAt, Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrContactNotFound
		}
		return nil, fmt.Errorf("update contact: %w", err)
	}

	return sqlcRowToDomainContact(row), nil
}

// SoftDelete marks a contact as deleted without removing from database.
// Preserves data for audit and potential recovery.
func (r *ContactRepository) SoftDelete(ctx context.Context, workspaceID, contactID string) error {
	now := time.Now()

	err := r.queries.SoftDeleteContact(ctx, sqlc.SoftDeleteContactParams{
		ID:          contactID,
		WorkspaceId: workspaceID,
		DeletedAt:   pgtype.Timestamp{Time: now, Valid: true},
		DeletedById: nil, // TODO: adicionar actorID quando disponível
	})
	if err != nil {
		return fmt.Errorf("soft delete contact: %w", err)
	}

	return nil
}

// Helper: retorna string vazia se pointer nil
func getStringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
