package domain

import (
	"time"
)

// ActivityType representa o tipo de interação na timeline.
type ActivityType string

const (
	ActivityTypeNote            ActivityType = "NOTE"
	ActivityTypeTask            ActivityType = "TASK"
	ActivityTypeEmail           ActivityType = "EMAIL"
	ActivityTypeCall            ActivityType = "CALL"
	ActivityTypeMeeting         ActivityType = "MEETING"
	ActivityTypeMessage         ActivityType = "MESSAGE"
	ActivityTypeLifecycleChange ActivityType = "LIFECYCLE_CHANGE"
)

// MessageDirection representa se a comunicação foi receptiva ou ativa.
type MessageDirection string

const (
	MessageDirectionInbound  MessageDirection = "INBOUND"
	MessageDirectionOutbound MessageDirection = "OUTBOUND"
)

// Activity representa um registro genérico na timeline.
type Activity struct {
	ID           string       `json:"id"`
	WorkspaceID  string       `json:"workspaceId"`
	CompanyID    *string      `json:"companyId"`
	ContactID    *string      `json:"contactId"`
	DealID       *string      `json:"dealId"`
	Type         ActivityType `json:"activityType"`
	ActivityID   *string      `json:"activityId"` // ID do recurso específico (NoteID, CallID, etc.)
	UserID       string       `json:"userId"`
	Metadata     []byte       `json:"metadata"`
	CreatedAt    time.Time    `json:"createdAt"`
}

// Note representa uma anotação na timeline.
type Note struct {
	ID          string     `json:"id"`
	WorkspaceID string     `json:"workspaceId"`
	CompanyID   *string    `json:"companyId"`
	ContactID   *string    `json:"contactId"`
	DealID      *string    `json:"dealId"`
	Content     string     `json:"content"`
	IsPinned    bool       `json:"isPinned"`
	UserID      string     `json:"userId"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	DeletedAt   *time.Time `json:"deletedAt"`
}

// Call representa o registro de uma chamada telefônica.
type Call struct {
	ID           string           `json:"id"`
	WorkspaceID  string           `json:"workspaceId"`
	ContactID    string           `json:"contactId"`
	CompanyID    *string          `json:"companyId"`
	Direction    MessageDirection `json:"direction"`
	Duration     *int32           `json:"duration"`
	RecordingURL *string          `json:"recordingUrl"`
	Summary      *string          `json:"summary"`
	UserID       string           `json:"userId"`
	CalledAt     time.Time        `json:"calledAt"`
	CreatedAt    time.Time        `json:"createdAt"`
}

// CreateNoteRequest DTO para criação de Notas.
type CreateNoteRequest struct {
	Content   string  `json:"content" validate:"required"`
	CompanyID *string `json:"companyId"`
	ContactID *string `json:"contactId"`
	DealID    *string `json:"dealId"`
}

// CreateCallRequest DTO para registro de Chamadas.
type CreateCallRequest struct {
	ContactID    string           `json:"contactId" validate:"required"`
	CompanyID    *string          `json:"companyId"`
	Direction    MessageDirection `json:"direction" validate:"required"`
	Duration     *int32           `json:"duration"`
	RecordingURL *string          `json:"recordingUrl"`
	Summary      *string          `json:"summary"`
	CalledAt     time.Time        `json:"calledAt"`
}

// Outros tipos como Meeting e Message podem ser expandidos conforme necessário.
// Por agora, focamos nos principais solicitados.
