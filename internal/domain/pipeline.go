package domain

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

// StageGroup representa o grupo de um estágio no pipeline (native PostgreSQL ENUM).
// Schema: public."StageGroup" ('OPEN', 'ACTIVE', 'DONE', 'CLOSED') - UPPERCASE no Prisma
type StageGroup string

const (
	StageGroupOpen   StageGroup = "OPEN"   // Initial stage
	StageGroupActive StageGroup = "ACTIVE" // Deal is in progress
	StageGroupDone   StageGroup = "DONE"   // Deal completed
	StageGroupClosed StageGroup = "CLOSED" // Deal closed (won or lost)
	StageGroupWon    StageGroup = "WON"    // Deal won (alias for CLOSED/DONE in some contexts)
)

// IsValid valida se o valor de StageGroup é válido.
func (s StageGroup) IsValid() bool {
	switch s {
	case StageGroupOpen, StageGroupActive, StageGroupDone, StageGroupClosed:
		return true
	}
	return false
}

// Scan implementa sql.Scanner para ler ENUM do PostgreSQL.
func (s *StageGroup) Scan(src interface{}) error {
	if src == nil {
		*s = StageGroupOpen // default
		return nil
	}

	var str string
	switch v := src.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return fmt.Errorf("cannot scan %T into StageGroup", src)
	}

	*s = StageGroup(str)
	if !s.IsValid() {
		return fmt.Errorf("invalid StageGroup value: %s", str)
	}
	return nil
}

// Value implementa driver.Valuer para escrever ENUM no PostgreSQL.
func (s StageGroup) Value() (driver.Value, error) {
	if !s.IsValid() {
		return nil, fmt.Errorf("invalid StageGroup value: %s", string(s))
	}
	return string(s), nil
}

// PipelineType representa o tipo de pipeline (native PostgreSQL ENUM).
// Schema: public."PipelineType" ('TASK', 'DEAL', 'TICKET', 'CONTACT') - UPPERCASE no Prisma
type PipelineType string

const (
	PipelineTypeTask    PipelineType = "TASK"    // Task pipeline
	PipelineTypeDeal    PipelineType = "DEAL"    // Standard B2B sales
	PipelineTypeTicket  PipelineType = "TICKET"  // Support tickets
	PipelineTypeContact PipelineType = "CONTACT" // Contact nurturing
	PipelineTypeSales   PipelineType = "DEAL"    // Alias for DEAL
)

// IsValid valida se o valor de PipelineType é válido.
func (t PipelineType) IsValid() bool {
	switch t {
	case PipelineTypeTask, PipelineTypeDeal, PipelineTypeTicket, PipelineTypeContact:
		return true
	}
	return false
}

// Scan implementa sql.Scanner para ler ENUM do PostgreSQL.
func (t *PipelineType) Scan(src interface{}) error {
	if src == nil {
		*t = PipelineTypeDeal // default
		return nil
	}

	var str string
	switch v := src.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return fmt.Errorf("cannot scan %T into PipelineType", src)
	}

	*t = PipelineType(str)
	if !t.IsValid() {
		return fmt.Errorf("invalid PipelineType value: %s", str)
	}
	return nil
}

// Value implementa driver.Valuer para escrever ENUM no PostgreSQL.
func (t PipelineType) Value() (driver.Value, error) {
	if !t.IsValid() {
		return nil, fmt.Errorf("invalid PipelineType value: %s", string(t))
	}
	return string(t), nil
}

// Pipeline representa um funil de vendas/processo no CRM.
// Schema: public."Pipeline" (schema real do Prisma)
// IMPORTANTE: Schema real é mais simples - apenas name, description, isDefault
type Pipeline struct {
	// Identificadores - IDs são TEXT no Prisma
	ID          string `json:"id" db:"id"`
	WorkspaceID string `json:"workspaceId" db:"workspaceId"`

	// Dados básicos
	Name        string  `json:"name" db:"name"`
	Description *string `json:"description,omitempty" db:"description"`

	// Configuração - schema real só tem isDefault
	PipelineType PipelineType `json:"pipelineType" db:"pipeline_type"` // Added for service compatibility
	IsActive     bool         `json:"isActive" db:"is_active"`         // Added for service compatibility
	IsDefault    bool         `json:"isDefault" db:"isDefault"`
	OwnerID      string       `json:"ownerId" db:"owner_id"` // Added for service compatibility

	// Timestamps
	CreatedAt time.Time  `json:"createdAt" db:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt" db:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty" db:"deletedAt"`

	// Stages (eager loaded quando necessário)
	Stages []PipelineStage `json:"stages,omitempty" db:"-"`
}

// PipelineStage representa um estágio dentro de um pipeline.
// Schema: public."PipelineStage" (schema real do Prisma)
type PipelineStage struct {
	// Identificadores - IDs são TEXT no Prisma
	ID          string  `json:"id" db:"id"`
	PipelineID  *string `json:"pipelineId,omitempty" db:"pipelineId"` // Nullable no schema real
	WorkspaceID string  `json:"workspaceId" db:"workspaceId"`

	// Dados básicos
	Name        string  `json:"name" db:"name"`
	Description *string `json:"description,omitempty" db:"description"`

	// Configuração - schema real usa group, type, color, isLocked
	Group           StageGroup   `json:"group" db:"group"`
	Type            PipelineType `json:"type" db:"type"`
	OrderIndex      int          `json:"orderIndex" db:"orderIndex"`
	Color           *string      `json:"color,omitempty" db:"color"`
	IsLocked        bool         `json:"isLocked" db:"isLocked"`
	Probability     int          `json:"probability" db:"probability"`
	AutoArchiveDays *int         `json:"autoArchiveDays,omitempty" db:"auto_archive_after_days"`

	// Timestamps
	CreatedAt time.Time  `json:"createdAt" db:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt" db:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty" db:"deletedAt"`
}

// CreatePipelineRequest DTO para criação de pipeline.
// WorkspaceID é injetado do path parameter.
type CreatePipelineRequest struct {
	// Dados obrigatórios
	Name string `json:"name" validate:"required,min=1,max=255"`

	// Dados opcionais
	Description  *string       `json:"description,omitempty" validate:"omitempty,max=5000"`
	IsDefault    *bool         `json:"isDefault,omitempty"`
	PipelineType *PipelineType `json:"pipelineType,omitempty"`
	IsActive     *bool         `json:"isActive,omitempty"`
	OwnerID      *string       `json:"ownerId,omitempty"`
}

// CreatePipelineWithStagesRequest DTO para criar pipeline + stages em uma operação.
type CreatePipelineWithStagesRequest struct {
	// Pipeline data
	Pipeline CreatePipelineRequest `json:"pipeline" validate:"required"`

	// Stages (opcional - se vazio, cria pipeline sem stages)
	Stages []CreateStageRequest `json:"stages,omitempty" validate:"omitempty,dive"`
}

// CreateStageRequest DTO para criação de estágio.
type CreateStageRequest struct {
	// Dados obrigatórios
	Name string `json:"name" validate:"required,min=1,max=255"`

	// Dados opcionais
	Description          *string     `json:"description,omitempty" validate:"omitempty,max=5000"`
	StageGroup           *StageGroup `json:"stageGroup,omitempty" validate:"omitempty,oneof=OPEN ACTIVE DONE CLOSED"`
	OrderIndex           *int        `json:"orderIndex,omitempty" validate:"omitempty,gte=0"`
	Probability          *int        `json:"probability,omitempty" validate:"omitempty,gte=0,lte=100"`
	AutoArchiveDays      *int        `json:"autoArchiveDays,omitempty" validate:"omitempty,gte=1"`
	Color                *string     `json:"color,omitempty"`
}

// UpdatePipelineRequest DTO para atualização parcial de pipeline (PATCH semântico).
type UpdatePipelineRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=5000"`
	IsDefault   *bool   `json:"isDefault,omitempty"`
}

// UpdateStageRequest DTO para atualização parcial de estágio.
type UpdateStageRequest struct {
	Name        *string       `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string       `json:"description,omitempty" validate:"omitempty,max=5000"`
	Group       *StageGroup   `json:"group,omitempty" validate:"omitempty,oneof=OPEN ACTIVE DONE CLOSED"`
	Type        *PipelineType `json:"type,omitempty" validate:"omitempty,oneof=TASK DEAL TICKET CONTACT"`
	OrderIndex  *int          `json:"orderIndex,omitempty" validate:"omitempty,gte=0"`
	Color       *string       `json:"color,omitempty"`
	IsLocked    *bool         `json:"isLocked,omitempty"`
}

// ReorderStagesRequest DTO para reordenar stages (batch update).
type ReorderStagesRequest struct {
	StageOrders []struct {
		StageID    string `json:"stageId" validate:"required"`
		OrderIndex int    `json:"orderIndex" validate:"required,gte=0"`
	} `json:"stageOrders" validate:"required,min=1,dive"`
}

// ListPipelinesParams parâmetros para listagem de pipelines.
type ListPipelinesParams struct {
	// Multi-tenant isolation (obrigatório) - ID é TEXT
	WorkspaceID string

	// Filtros opcionais
	IsDefault *bool

	// Busca textual (name + description)
	Query *string

	// Include stages
	IncludeStages bool

	// Paginação
	Limit  int
	Cursor *string // RFC3339 timestamp
	Sort   string  // "name:asc", "createdAt:desc", etc.
}

// Normalize normaliza os parâmetros de listagem (defaults e validação).
func (p *ListPipelinesParams) Normalize() {
	if p.Limit <= 0 || p.Limit > 100 {
		p.Limit = 50
	}
	if p.Sort == "" {
		p.Sort = "createdAt:desc"
	}
	if p.Query != nil {
		q := strings.TrimSpace(*p.Query)
		if q == "" {
			p.Query = nil
		} else {
			p.Query = &q
		}
	}
}

// PipelineListResponse resposta paginada de pipelines.
type PipelineListResponse struct {
	Data []Pipeline `json:"data"`
	Meta struct {
		HasNextPage bool    `json:"hasNextPage"`
		NextCursor  *string `json:"nextCursor,omitempty"`
	} `json:"meta"`
}
