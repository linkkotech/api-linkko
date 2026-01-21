package domain

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

// Priority representa a prioridade de uma tarefa (native PostgreSQL ENUM).
// Schema: public."Priority" ('LOW', 'MEDIUM', 'HIGH', 'URGENT') - UPPERCASE no Prisma
type Priority string

const (
	PriorityLow    Priority = "LOW"
	PriorityMedium Priority = "MEDIUM"
	PriorityHigh   Priority = "HIGH"
	PriorityUrgent Priority = "URGENT"
)

// IsValid valida se o valor de Priority é válido.
func (p Priority) IsValid() bool {
	switch p {
	case PriorityLow, PriorityMedium, PriorityHigh, PriorityUrgent:
		return true
	}
	return false
}

// Scan implementa sql.Scanner para ler ENUM do PostgreSQL.
func (p *Priority) Scan(src interface{}) error {
	if src == nil {
		*p = PriorityMedium // default
		return nil
	}

	var str string
	switch v := src.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return fmt.Errorf("cannot scan %T into Priority", src)
	}

	*p = Priority(str)
	if !p.IsValid() {
		return fmt.Errorf("invalid Priority value: %s", str)
	}
	return nil
}

// Value implementa driver.Valuer para escrever ENUM no PostgreSQL.
func (p Priority) Value() (driver.Value, error) {
	if !p.IsValid() {
		return nil, fmt.Errorf("invalid Priority value: %s", string(p))
	}
	return string(p), nil
}

// TaskStatus representa o status de uma tarefa no Kanban (native PostgreSQL ENUM).
// Schema: public."TaskStatus" ('TODO', 'IN_PROGRESS', 'DONE', 'CANCELLED') - UPPERCASE no Prisma
type TaskStatus string

const (
	TaskStatusTodo       TaskStatus = "TODO"
	TaskStatusInProgress TaskStatus = "IN_PROGRESS"
	TaskStatusDone       TaskStatus = "DONE"
	TaskStatusCancelled  TaskStatus = "CANCELLED"
)

// IsValid valida se o valor de TaskStatus é válido.
func (s TaskStatus) IsValid() bool {
	switch s {
	case TaskStatusTodo, TaskStatusInProgress, TaskStatusDone, TaskStatusCancelled:
		return true
	}
	return false
}

// Scan implementa sql.Scanner para ler ENUM do PostgreSQL.
func (s *TaskStatus) Scan(src interface{}) error {
	if src == nil {
		*s = TaskStatusTodo // default
		return nil
	}

	var str string
	switch v := src.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return fmt.Errorf("cannot scan %T into TaskStatus", src)
	}

	*s = TaskStatus(str)
	if !s.IsValid() {
		return fmt.Errorf("invalid TaskStatus value: %s", str)
	}
	return nil
}

// Value implementa driver.Valuer para escrever ENUM no PostgreSQL.
func (s TaskStatus) Value() (driver.Value, error) {
	if !s.IsValid() {
		return nil, fmt.Errorf("invalid TaskStatus value: %s", string(s))
	}
	return string(s), nil
}

// TaskType representa o tipo de uma tarefa (native PostgreSQL ENUM).
// Schema: public."TaskType" ('CALL', 'EMAIL', 'MEETING', 'FOLLOWUP', 'OTHER') - UPPERCASE no Prisma
type TaskType string

const (
	TaskTypeCall     TaskType = "CALL"
	TaskTypeEmail    TaskType = "EMAIL"
	TaskTypeMeeting  TaskType = "MEETING"
	TaskTypeFollowup TaskType = "FOLLOWUP"
	TaskTypeOther    TaskType = "OTHER"
)

// IsValid valida se o valor de TaskType é válido.
func (t TaskType) IsValid() bool {
	switch t {
	case TaskTypeCall, TaskTypeEmail, TaskTypeMeeting, TaskTypeFollowup, TaskTypeOther:
		return true
	}
	return false
}

// Scan implementa sql.Scanner para ler ENUM do PostgreSQL.
func (t *TaskType) Scan(src interface{}) error {
	if src == nil {
		*t = TaskTypeOther // default
		return nil
	}

	var str string
	switch v := src.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return fmt.Errorf("cannot scan %T into TaskType", src)
	}

	*t = TaskType(str)
	if !t.IsValid() {
		return fmt.Errorf("invalid TaskType value: %s", str)
	}
	return nil
}

// Value implementa driver.Valuer para escrever ENUM no PostgreSQL.
func (t TaskType) Value() (driver.Value, error) {
	if !t.IsValid() {
		return nil, fmt.Errorf("invalid TaskType value: %s", string(t))
	}
	return string(t), nil
}

// Task representa uma tarefa no CRM com Kanban positioning.
// Schema: public."Task" (schema real do Prisma)
//
// IMPORTANTE: Position usa float64 (DECIMAL(20,10) no DB) para fractional positioning.
// IMPORTANTE: ActorID (conceito Go) mapeia para ownerId (campo DB).
type Task struct {
	// Identificadores - IDs são TEXT no Prisma
	ID          string `json:"id" db:"id"`
	WorkspaceID string `json:"workspaceId" db:"workspaceId"`

	// Dados da tarefa
	Title       string  `json:"title" db:"title"`
	Description *string `json:"description,omitempty" db:"description"`

	// Estados e classificação (native ENUMs)
	Status   TaskStatus `json:"status" db:"status"`
	Priority Priority   `json:"priority" db:"priority"`
	Type     TaskType   `json:"type" db:"type"`

	// Kanban positioning (DECIMAL(20,10) para precisão)
	Position float64 `json:"position" db:"position"`

	// Relacionamentos - IDs são TEXT
	ActorID    string  `json:"actorId" db:"ownerId"`                 // Owner/creator
	AssignedTo *string `json:"assignedTo,omitempty" db:"assignedTo"` // Assignee
	ContactID  *string `json:"contactId,omitempty" db:"contactId"`   // Related contact

	// Datas
	DueDate     *time.Time `json:"dueDate,omitempty" db:"due_date"`
	CompletedAt *time.Time `json:"completedAt,omitempty" db:"completed_at"`

	// Timestamps
	CreatedAt time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time  `json:"updatedAt" db:"updated_at"`
	DeletedAt *time.Time `json:"deletedAt,omitempty" db:"deleted_at"`
}

// CreateTaskRequest DTO para criação de tarefa.
//
// ActorID pode ser omitido - será inferido do JWT claims.ActorID.
// WorkspaceID é sempre injetado do path parameter.
// Position é calculado automaticamente pelo service (baseado em toStatus).
type CreateTaskRequest struct {
	// Dados obrigatórios
	Title string `json:"title" validate:"required,min=1,max=500"`

	// Dados opcionais
	Description *string `json:"description,omitempty" validate:"omitempty,max=5000"`

	// Estados e classificação
	Status   *TaskStatus `json:"status,omitempty" validate:"omitempty,oneof=backlog todo in_progress in_review done cancelled"`
	Priority *Priority   `json:"priority,omitempty" validate:"omitempty,oneof=low medium high urgent"`
	Type     *TaskType   `json:"type,omitempty" validate:"omitempty,oneof=task bug feature improvement research"`

	// Relacionamentos opcionais - IDs são TEXT
	ActorID    *string `json:"actorId,omitempty"`
	AssignedTo *string `json:"assignedTo,omitempty"`
	ContactID  *string `json:"contactId,omitempty"`

	// Datas
	DueDate *time.Time `json:"dueDate,omitempty"`
}

// UpdateTaskRequest DTO para atualização parcial de tarefa (PATCH semântico).
//
// Todos os campos são ponteiros - nil = não modificar.
// Para mover tarefa (Kanban drag-and-drop), usar MoveTaskRequest no endpoint :move.
type UpdateTaskRequest struct {
	// Dados
	Title       *string `json:"title,omitempty" validate:"omitempty,min=1,max=500"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=5000"`

	// Estados e classificação (sem Position - usar :move para reordenação)
	Priority *Priority `json:"priority,omitempty" validate:"omitempty,oneof=low medium high urgent"`
	Type     *TaskType `json:"type,omitempty" validate:"omitempty,oneof=task bug feature improvement research"`

	// Relacionamentos - IDs são TEXT
	AssignedTo *string `json:"assignedTo,omitempty"`
	ContactID  *string `json:"contactId,omitempty"`

	// Datas
	DueDate     *time.Time `json:"dueDate,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// MoveTaskRequest DTO para mover tarefa no Kanban (drag-and-drop com reordenação).
//
// Endpoint: POST /workspaces/{workspaceId}/tasks/{taskId}:move
// Lógica: Calcula position = (beforeTask.position + afterTask.position) / 2
//
// Cenários:
// - beforeTaskID e afterTaskID nil: position = 1000.0 (primeiro da coluna)
// - Só beforeTaskID: position = beforeTask.position - 1000.0
// - Só afterTaskID: position = afterTask.position + 1000.0
// - Ambos: position = (beforeTask.position + afterTask.position) / 2
type MoveTaskRequest struct {
	// Status destino (obrigatório)
	ToStatus TaskStatus `json:"toStatus" validate:"required,oneof=backlog todo in_progress in_review done cancelled"`

	// Posicionamento relativo (opcional - vazio = final da coluna) - IDs são TEXT
	BeforeTaskID *string `json:"beforeTaskId,omitempty"`
	AfterTaskID  *string `json:"afterTaskId,omitempty"`
}

// ListTasksParams parâmetros para listagem de tarefas.
//
// WorkspaceID é sempre obrigatório (multi-tenant isolation).
// Filtros opcionais: Status, AssignedTo, ActorID, ContactID.
type ListTasksParams struct {
	// Multi-tenant isolation (obrigatório) - ID é TEXT
	WorkspaceID string

	// Filtros opcionais - IDs são TEXT
	Status     *TaskStatus
	Priority   *Priority
	Type       *TaskType
	AssignedTo *string
	ActorID    *string // Owner
	ContactID  *string

	// Busca textual (título + descrição)
	Query *string

	// Paginação
	Limit  int
	Cursor *string // RFC3339 timestamp ou ULID
	Sort   string  // Padrão: "position:asc" dentro de cada status
}

// Normalize normaliza os parâmetros de listagem (defaults e validação).
func (p *ListTasksParams) Normalize() {
	if p.Limit <= 0 || p.Limit > 100 {
		p.Limit = 50
	}
	if p.Sort == "" {
		p.Sort = "position:asc" // Kanban order
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

// TaskListResponse resposta paginada de tarefas.
//
// Meta.NextCursor contém o cursor para a próxima página.
// Meta.HasNextPage indica se há mais resultados.
type TaskListResponse struct {
	Data []Task `json:"data"`
	Meta struct {
		HasNextPage bool    `json:"hasNextPage"`
		NextCursor  *string `json:"nextCursor,omitempty"`
	} `json:"meta"`
}
