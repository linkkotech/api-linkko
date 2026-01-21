package domain

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// DealStage representa o estado do funil de vendas (native PostgreSQL ENUM).
// Schema: public."DealStage" ('OPEN', 'WON', 'LOST')
type DealStage string

const (
	DealStageOpen DealStage = "OPEN"
	DealStageWon  DealStage = "WON"
	DealStageLost DealStage = "LOST"
)

func (s DealStage) IsValid() bool {
	switch s {
	case DealStageOpen, DealStageWon, DealStageLost:
		return true
	}
	return false
}

func (s *DealStage) Scan(src interface{}) error {
	if src == nil {
		*s = DealStageOpen
		return nil
	}
	var str string
	switch v := src.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return fmt.Errorf("cannot scan %T into DealStage", src)
	}
	*s = DealStage(str)
	return nil
}

func (s DealStage) Value() (driver.Value, error) {
	return string(s), nil
}

// Deal representa um negócio no CRM.
type Deal struct {
	ID                string     `json:"id"`
	WorkspaceID       string     `json:"workspaceId"`
	PipelineID        string     `json:"pipelineId"`
	StageID           *string    `json:"stageId"`
	ContactID         *string    `json:"contactId"`
	CompanyID         *string    `json:"companyId"`
	Name              string     `json:"name"`
	Value             *float64   `json:"value"`
	Currency          string     `json:"currency"`
	Stage             DealStage  `json:"stage"`
	Probability       *int32     `json:"probability"`
	ExpectedCloseDate *time.Time `json:"expectedCloseDate"`
	ClosedAt          *time.Time `json:"closedAt"`
	LostReason        *string    `json:"lostReason"`
	Description       *string    `json:"description"`
	OwnerID           *string    `json:"ownerId"`
	CreatedByID       string     `json:"createdById"`
	UpdatedByID       *string    `json:"updatedById"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`

	// Relational fields (Joins)
	ContactName *string `json:"contactName,omitempty"`
	CompanyName *string `json:"companyName,omitempty"`
}

// DealStageHistory registra a movimentação de um Deal entre estágios.
type DealStageHistory struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspaceId"`
	DealID      string    `json:"dealId"`
	FromStage   DealStage `json:"fromStage"`
	ToStage     DealStage `json:"toStage"`
	Reason      *string   `json:"reason"`
	UserID      string    `json:"userId"`
	CreatedAt   time.Time `json:"createdAt"`
}

// CreateDealRequest é o DTO para criação de Negócios.
type CreateDealRequest struct {
	Name              string     `json:"name" validate:"required"`
	PipelineID        string     `json:"pipelineId" validate:"required"`
	StageID           *string    `json:"stageId"`
	ContactID         *string    `json:"contactId"`
	CompanyID         *string    `json:"companyId"`
	Value             *float64   `json:"value"`
	Currency          string     `json:"currency"`
	Probability       *int32     `json:"probability"`
	ExpectedCloseDate *time.Time `json:"expectedCloseDate"`
	Description       *string    `json:"description"`
	OwnerID           *string    `json:"ownerId"`
}

// UpdateDealRequest é o DTO para atualização de Negócios.
type UpdateDealRequest struct {
	Name              *string    `json:"name"`
	Value             *float64   `json:"value"`
	Currency          *string    `json:"currency"`
	Probability       *int32     `json:"probability"`
	ExpectedCloseDate *time.Time `json:"expectedCloseDate"`
	Description       *string    `json:"description"`
	OwnerID           *string    `json:"ownerId"`
}

// UpdateDealStageRequest é o DTO para movimentação de estágio (Pipeline).
type UpdateDealStageRequest struct {
	StageID   string     `json:"stageId" validate:"required"`
	Stage     *DealStage `json:"stage"` // OPEN, WON, LOST
	Reason    *string    `json:"reason"`
	ClosedAt  *time.Time `json:"closedAt"`
}
