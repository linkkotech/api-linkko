package domain

import (
	"encoding/json"
	"errors"
	"time"
)

// PortfolioCategoryEnum representa a categoria do item.
type PortfolioCategoryEnum string

const (
	PortfolioCategoryProduct    PortfolioCategoryEnum = "PRODUCT"
	PortfolioCategoryService    PortfolioCategoryEnum = "SERVICE"
	PortfolioCategoryRealEstate PortfolioCategoryEnum = "REAL_ESTATE"
	PortfolioCategoryLodging    PortfolioCategoryEnum = "LODGING"
	PortfolioCategoryEvent      PortfolioCategoryEnum = "EVENT"
)

// PortfolioVertical representa a vertical de negócio.
type PortfolioVertical string

const (
	PortfolioVerticalGeneral       PortfolioVertical = "GENERAL"
	PortfolioVerticalHealthcare    PortfolioVertical = "HEALTHCARE"
	PortfolioVerticalAesthetics    PortfolioVertical = "AESTHETICS"
	PortfolioVerticalBeauty        PortfolioVertical = "BEAUTY"
	PortfolioVerticalRetail        PortfolioVertical = "RETAIL"
	PortfolioVerticalRealEstate    PortfolioVertical = "REAL_ESTATE"
	PortfolioVerticalHosting       PortfolioVertical = "HOSTING"
	PortfolioVerticalEvents        PortfolioVertical = "EVENTS"
	PortfolioVerticalGeneralLocal  PortfolioVertical = "GENERAL_LOCAL"
	PortfolioVerticalB2BCorporate PortfolioVertical = "B2B_CORPORATE"
)

// PortfolioStatus representa o status do item no catálogo.
type PortfolioStatus string

const (
	PortfolioStatusDraft       PortfolioStatus = "DRAFT"
	PortfolioStatusActive      PortfolioStatus = "ACTIVE"
	PortfolioStatusInactive    PortfolioStatus = "INACTIVE"
	PortfolioStatusUnavailable PortfolioStatus = "UNAVAILABLE"
	PortfolioStatusArchived    PortfolioStatus = "ARCHIVED"
)

// PortfolioVisibility representa a visibilidade do item.
type PortfolioVisibility string

const (
	PortfolioVisibilityPublic   PortfolioVisibility = "PUBLIC"
	PortfolioVisibilityInternal PortfolioVisibility = "INTERNAL"
)

// PortfolioItem representa um item no catálogo/portfólio.
type PortfolioItem struct {
	ID          string                `json:"id"`
	WorkspaceID string                `json:"workspaceId"`
	Name        string                `json:"name"`
	Description *string               `json:"description"`
	SKU         *string               `json:"sku"`
	Category    PortfolioCategoryEnum `json:"category"`
	Vertical    PortfolioVertical     `json:"vertical"`
	Status      PortfolioStatus       `json:"status"`
	Visibility  PortfolioVisibility   `json:"visibility"`
	BasePrice   float64               `json:"basePrice"`
	Currency    string                `json:"currency"`
	ImageURL    *string               `json:"imageUrl"`
	Metadata    json.RawMessage       `json:"metadata"`
	Tags        []string              `json:"tags"`
	CreatedByID string                `json:"createdById"`
	UpdatedByID *string               `json:"updatedById"`
	CreatedAt   time.Time             `json:"createdAt"`
	UpdatedAt   time.Time             `json:"updatedAt"`
	DeletedAt   *time.Time            `json:"deletedAt"`
}

// CreatePortfolioItemRequest DTO para criação.
type CreatePortfolioItemRequest struct {
	Name        string                `json:"name" validate:"required"`
	Description *string               `json:"description"`
	SKU         *string               `json:"sku"`
	Category    PortfolioCategoryEnum `json:"category" validate:"required"`
	Vertical    PortfolioVertical     `json:"vertical" validate:"required"`
	Status      PortfolioStatus       `json:"status"`
	Visibility  PortfolioVisibility   `json:"visibility"`
	BasePrice   float64               `json:"basePrice"`
	Currency    string                `json:"currency"`
	ImageURL    *string               `json:"imageUrl"`
	Metadata    json.RawMessage       `json:"metadata"`
	Tags        []string              `json:"tags"`
}

// UpdatePortfolioItemRequest DTO para atualização parcial.
type UpdatePortfolioItemRequest struct {
	Name        *string                `json:"name"`
	Description *string                `json:"description"`
	SKU         *string                `json:"sku"`
	Category    *PortfolioCategoryEnum `json:"category"`
	Vertical    *PortfolioVertical     `json:"vertical"`
	Status      *PortfolioStatus       `json:"status"`
	Visibility  *PortfolioVisibility   `json:"visibility"`
	BasePrice   *float64               `json:"basePrice"`
	Currency    *string                `json:"currency"`
	ImageURL    *string                `json:"imageUrl"`
	Metadata    json.RawMessage        `json:"metadata"`
	Tags        []string               `json:"tags"`
}

// ValidatePortfolioContext valida se a combinação de Vertical e Categoria é aceitável.
func ValidatePortfolioContext(cat PortfolioCategoryEnum, vert PortfolioVertical) error {
	// Regra de exemplo: Real Estate vertical exige Real Estate ou Service category
	if vert == PortfolioVerticalRealEstate {
		if cat != PortfolioCategoryRealEstate && cat != PortfolioCategoryService {
			return errors.New("real estate vertical requires real_estate or service category")
		}
	}
	return nil
}
