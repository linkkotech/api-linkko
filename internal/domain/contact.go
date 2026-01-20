package domain

import (
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// Contact representa um contato no CRM com isolamento multi-tenant.
// Campos mapeados para o schema contacts (000002_contacts.up.sql).
//
// IMPORTANTE: ActorID (conceito Go) mapeia para owner_id (campo DB).
// Actor = User ou AI Agent que "possui" o contato.
type Contact struct {
	// Identificadores - Type safety com uuid.UUID
	ID          uuid.UUID `json:"id" db:"id"`
	WorkspaceID uuid.UUID `json:"workspaceId" db:"workspace_id"` // Imutável após criação

	// Dados de contato
	Name  string  `json:"name" db:"name"`
	Email string  `json:"email" db:"email"`
	Phone *string `json:"phone,omitempty" db:"phone"`

	// Relacionamentos
	CompanyID *uuid.UUID `json:"companyId,omitempty" db:"company_id"`

	// Actor (owner) - Conceito unificado para User ou AI Agent
	// DB: owner_id | Conceito: ActorID
	ActorID uuid.UUID `json:"actorId" db:"owner_id"`

	// Metadata
	Tags         []string               `json:"tags" db:"tags"`
	CustomFields map[string]interface{} `json:"customFields" db:"custom_fields"`

	// Timestamps
	CreatedAt time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time  `json:"updatedAt" db:"updated_at"`
	DeletedAt *time.Time `json:"deletedAt,omitempty" db:"deleted_at"`
}

// CreateContactRequest DTO para criação de contato.
//
// ActorID pode ser omitido no request - será inferido do JWT claims.ActorID.
// WorkspaceID é sempre injetado do path parameter (nunca do body).
type CreateContactRequest struct {
	// Dados obrigatórios
	Name  string `json:"name" validate:"required,min=1,max=255"`
	Email string `json:"email" validate:"required,email,max=255"`

	// Dados opcionais
	Phone *string `json:"phone,omitempty" validate:"omitempty,max=50"`

	// Relacionamentos opcionais
	CompanyID *uuid.UUID `json:"companyId,omitempty" validate:"omitempty,uuid4"`

	// Actor (owner) - Opcional: se nil, usa claims.ActorID do JWT
	ActorID *uuid.UUID `json:"actorId,omitempty" validate:"omitempty,uuid4"`

	// Metadata
	Tags         []string               `json:"tags,omitempty" validate:"omitempty,max=20,dive,min=1"`
	CustomFields map[string]interface{} `json:"customFields,omitempty"`
}

// UpdateContactRequest DTO para atualização parcial de contato (PATCH semântico).
//
// Todos os campos são ponteiros:
// - nil = campo não enviado (não modificar)
// - *valor = atualizar campo com novo valor
// - *"" (string vazia) = limpar campo opcional (ex: phone)
//
// WorkspaceID e ID não podem ser atualizados (imutáveis).
type UpdateContactRequest struct {
	// Dados
	Name  *string `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Email *string `json:"email,omitempty" validate:"omitempty,email,max=255"`
	Phone *string `json:"phone,omitempty" validate:"omitempty,max=50"`

	// Relacionamentos
	CompanyID *uuid.UUID `json:"companyId,omitempty" validate:"omitempty,uuid4"`
	ActorID   *uuid.UUID `json:"actorId,omitempty" validate:"omitempty,uuid4"`

	// Metadata
	Tags         *[]string              `json:"tags,omitempty" validate:"omitempty,max=20,dive,min=1"`
	CustomFields map[string]interface{} `json:"customFields,omitempty"`
}

// ListContactsParams parâmetros para listagem de contatos.
//
// WorkspaceID é sempre obrigatório (multi-tenant isolation).
// ActorID filtra contatos por "dono" (antes chamado OwnerID).
type ListContactsParams struct {
	// Multi-tenant isolation (obrigatório)
	WorkspaceID uuid.UUID

	// Paginação
	Limit  int
	Cursor *string // RFC3339 timestamp ou ULID
	Sort   string  // "created_at:desc", "name:asc", etc.

	// Filtros
	Query     *string    // Full-text search (name + email)
	ActorID   *uuid.UUID // Filter by actor (owner)
	CompanyID *uuid.UUID // Filter by company
}

// ContactListResponse resposta paginada de contatos.
//
// Meta.NextCursor contém o cursor para a próxima página.
// Meta.HasNextPage indica se há mais resultados.
type ContactListResponse struct {
	Data []Contact `json:"data"`
	Meta struct {
		HasNextPage bool    `json:"hasNextPage"`
		NextCursor  *string `json:"nextCursor,omitempty"`
	} `json:"meta"`
}

// Validate valida o CreateContactRequest.
// Sanitiza Name (trim whitespace) antes da validação.
func (r *CreateContactRequest) Validate() error {
	// Sanitização: remover espaços em branco extras
	r.Name = strings.TrimSpace(r.Name)
	if r.Phone != nil {
		trimmed := strings.TrimSpace(*r.Phone)
		r.Phone = &trimmed
	}

	// Validação com go-playground/validator
	validate := validator.New()
	return validate.Struct(r)
}

// Validate valida o UpdateContactRequest.
// Sanitiza campos de string antes da validação.
func (r *UpdateContactRequest) Validate() error {
	// Sanitização: remover espaços em branco extras
	if r.Name != nil {
		trimmed := strings.TrimSpace(*r.Name)
		r.Name = &trimmed
	}
	if r.Phone != nil {
		trimmed := strings.TrimSpace(*r.Phone)
		r.Phone = &trimmed
	}

	// Validação com go-playground/validator
	validate := validator.New()
	return validate.Struct(r)
}
