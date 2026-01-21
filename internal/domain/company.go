package domain

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

// CompanyLifecycleStage representa o estágio do ciclo de vida da empresa (native PostgreSQL ENUM).
// Schema: public."CompanyLifecycleStage" ('LEAD', 'MQL', 'SQL', 'CUSTOMER', 'CHURNED') - UPPERCASE no Prisma
type CompanyLifecycleStage string

const (
	LifecycleLead     CompanyLifecycleStage = "LEAD"     // Initial contact, not qualified
	LifecycleMQL      CompanyLifecycleStage = "MQL"      // Marketing Qualified Lead
	LifecycleSQL      CompanyLifecycleStage = "SQL"      // Sales Qualified Lead
	LifecycleCustomer CompanyLifecycleStage = "CUSTOMER" // Active customer
	LifecycleChurned  CompanyLifecycleStage = "CHURNED"  // Former customer/churned
)

// IsValid valida se o valor de CompanyLifecycleStage é válido.
func (s CompanyLifecycleStage) IsValid() bool {
	switch s {
	case LifecycleLead, LifecycleMQL, LifecycleSQL, LifecycleCustomer, LifecycleChurned:
		return true
	}
	return false
}

// Scan implementa sql.Scanner para ler ENUM do PostgreSQL.
func (s *CompanyLifecycleStage) Scan(src interface{}) error {
	if src == nil {
		*s = LifecycleLead // default
		return nil
	}

	var str string
	switch v := src.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return fmt.Errorf("cannot scan %T into CompanyLifecycleStage", src)
	}

	*s = CompanyLifecycleStage(str)
	if !s.IsValid() {
		return fmt.Errorf("invalid CompanyLifecycleStage value: %s", str)
	}
	return nil
}

// Value implementa driver.Valuer para escrever ENUM no PostgreSQL.
func (s CompanyLifecycleStage) Value() (driver.Value, error) {
	if !s.IsValid() {
		return nil, fmt.Errorf("invalid CompanyLifecycleStage value: %s", string(s))
	}
	return string(s), nil
}

// CompanySize representa o tamanho da empresa (native PostgreSQL ENUM).
// Schema: public."CompanySize" ('STARTUP', 'SMB', 'MID_MARKET', 'ENTERPRISE') - UPPERCASE no Prisma
type CompanySize string

const (
	SizeStartup    CompanySize = "STARTUP"    // Startup/early stage
	SizeSMB        CompanySize = "SMB"        // Small/Medium Business
	SizeMidMarket  CompanySize = "MID_MARKET" // Mid-market
	SizeEnterprise CompanySize = "ENTERPRISE" // Enterprise
)

// IsValid valida se o valor de CompanySize é válido.
func (s CompanySize) IsValid() bool {
	switch s {
	case SizeStartup, SizeSMB, SizeMidMarket, SizeEnterprise:
		return true
	}
	return false
}

// Scan implementa sql.Scanner para ler ENUM do PostgreSQL.
func (s *CompanySize) Scan(src interface{}) error {
	if src == nil {
		*s = SizeSMB // default
		return nil
	}

	var str string
	switch v := src.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return fmt.Errorf("cannot scan %T into CompanySize", src)
	}

	*s = CompanySize(str)
	if !s.IsValid() {
		return fmt.Errorf("invalid CompanySize value: %s", str)
	}
	return nil
}

// Value implementa driver.Valuer para escrever ENUM no PostgreSQL.
func (s CompanySize) Value() (driver.Value, error) {
	if !s.IsValid() {
		return nil, fmt.Errorf("invalid CompanySize value: %s", string(s))
	}
	return string(s), nil
}

// Company representa uma empresa no CRM B2B.
// Schema: public."Company" (schema real do Prisma)
// IMPORTANTE: Colunas usam camelCase (Prisma) no DB, mapeadas via tags db.
type Company struct {
	// Identificadores - IDs são TEXT no Prisma
	ID          string `json:"id" db:"id"`
	WorkspaceID string `json:"workspaceId" db:"workspaceId"`

	// Dados básicos
	Name   string  `json:"name" db:"name"`
	Domain *string `json:"domain,omitempty" db:"website"` // website no schema real

	// Classificação
	Industry       *string               `json:"industry,omitempty" db:"industry"`
	LifecycleStage CompanyLifecycleStage `json:"lifecycleStage" db:"lifecycleStage"`
	Size           CompanySize           `json:"size" db:"size"` // Campo é "size" no schema real

	// Contato
	Phone   *string `json:"phone,omitempty" db:"phone"`
	Email   *string `json:"email,omitempty" db:"email"`
	Website *string `json:"website,omitempty" db:"website"`

	// Address (JSONB)
	Address map[string]interface{} `json:"address,omitempty" db:"address"`

	// Métricas de negócio - revenue no schema real
	Revenue       *float64 `json:"revenue,omitempty" db:"revenue"`
	EmployeeCount *int     `json:"employeeCount,omitempty" db:"employeeCount"`

	// Ownership - assignedToId no schema real
	OwnerID string `json:"ownerId" db:"assignedToId"`

	// Metadata
	Tags         []string               `json:"tags" db:"tags"`
	CustomFields map[string]interface{} `json:"customFields" db:"customFields"`
	Notes        *string                `json:"notes,omitempty" db:"notes"`

	// Timestamps
	CreatedAt time.Time  `json:"createdAt" db:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt" db:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty" db:"deletedAt"`
}

// CreateCompanyRequest DTO para criação de empresa.
// WorkspaceID é injetado do path parameter, OwnerID do JWT claims.
type CreateCompanyRequest struct {
	// Dados obrigatórios
	Name string `json:"name" validate:"required,min=1,max=255"`

	// Dados opcionais
	Domain   *string `json:"domain,omitempty" validate:"omitempty,max=255"`
	Industry *string `json:"industry,omitempty" validate:"omitempty,max=100"`

	// Classificação
	LifecycleStage *CompanyLifecycleStage `json:"lifecycleStage,omitempty" validate:"omitempty,oneof=LEAD MQL SQL CUSTOMER CHURNED"`
	Size           *CompanySize           `json:"size,omitempty" validate:"omitempty,oneof=STARTUP SMB MID_MARKET ENTERPRISE"`

	// Contato
	Phone   *string `json:"phone,omitempty" validate:"omitempty,max=50"`
	Email   *string `json:"email,omitempty" validate:"omitempty,email,max=255"`
	Website *string `json:"website,omitempty" validate:"omitempty,url,max=500"`

	// Address
	Address map[string]interface{} `json:"address,omitempty"`

	// Métricas - revenue no schema real
	Revenue       *float64 `json:"revenue,omitempty" validate:"omitempty,gte=0"`
	EmployeeCount *int     `json:"employeeCount,omitempty" validate:"omitempty,gte=1"`

	// Ownership (opcional - default JWT claims) - ID é TEXT
	OwnerID *string `json:"ownerId,omitempty"`

	// Metadata
	Tags         []string               `json:"tags,omitempty" validate:"omitempty,max=20,dive,min=1"`
	CustomFields map[string]interface{} `json:"customFields,omitempty"`
	Notes        *string                `json:"notes,omitempty" validate:"omitempty,max=5000"`
}

// UpdateCompanyRequest DTO para atualização parcial (PATCH semântico).
// Todos os campos são ponteiros - nil = não modificar.
type UpdateCompanyRequest struct {
	// Dados básicos
	Name   *string `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Domain *string `json:"domain,omitempty" validate:"omitempty,max=255"`

	// Classificação
	Industry       *string                `json:"industry,omitempty" validate:"omitempty,max=100"`
	LifecycleStage *CompanyLifecycleStage `json:"lifecycleStage,omitempty" validate:"omitempty,oneof=LEAD MQL SQL CUSTOMER CHURNED"`
	Size           *CompanySize           `json:"size,omitempty" validate:"omitempty,oneof=STARTUP SMB MID_MARKET ENTERPRISE"`

	// Contato
	Phone   *string `json:"phone,omitempty" validate:"omitempty,max=50"`
	Email   *string `json:"email,omitempty" validate:"omitempty,email,max=255"`
	Website *string `json:"website,omitempty" validate:"omitempty,url,max=500"`

	// Address
	Address map[string]interface{} `json:"address,omitempty"`

	// Métricas - revenue no schema real
	Revenue       *float64 `json:"revenue,omitempty" validate:"omitempty,gte=0"`
	EmployeeCount *int     `json:"employeeCount,omitempty" validate:"omitempty,gte=1"`

	// Ownership - ID é TEXT
	OwnerID *string `json:"ownerId,omitempty"`

	// Metadata
	Tags         *[]string              `json:"tags,omitempty" validate:"omitempty,max=20,dive,min=1"`
	CustomFields map[string]interface{} `json:"customFields,omitempty"`
	Notes        *string                `json:"notes,omitempty" validate:"omitempty,max=5000"`
}

// ListCompaniesParams parâmetros para listagem de empresas.
type ListCompaniesParams struct {
	// Multi-tenant isolation (obrigatório) - ID é TEXT
	WorkspaceID string

	// Filtros opcionais - size no schema real
	LifecycleStage *CompanyLifecycleStage
	Size           *CompanySize
	Industry       *string
	OwnerID        *string

	// Busca textual (name + domain)
	Query *string

	// Paginação
	Limit  int
	Cursor *string // RFC3339 timestamp
	Sort   string  // "name:asc", "createdAt:desc", etc.
}

// Normalize normaliza os parâmetros de listagem (defaults e validação).
func (p *ListCompaniesParams) Normalize() {
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

// CompanyListResponse resposta paginada de empresas.
type CompanyListResponse struct {
	Data []Company `json:"data"`
	Meta struct {
		HasNextPage bool    `json:"hasNextPage"`
		NextCursor  *string `json:"nextCursor,omitempty"`
	} `json:"meta"`
}
