# 🔍 AUDITORIA DE LIXO TÉCNICO - Linkko API

**Data**: 20/01/2026  
**Objetivo**: Identificar arquivos obsoletos e inconsistências técnicas após sincronização com schema real Prisma

---

## 📋 RESUMO EXECUTIVO

Após a recepção do **Schema Real do Supabase** (arquivo `NEXTCRM_DATABASE_SCHEMAS_SQL.sql`), identificamos que uma grande parte do código foi gerada por "adivinhação", criando conflitos com a realidade do banco de dados.

### ⚠️ Principais Problemas Identificados

1. **Migrations Obsoletas**: 6 migrations criadas antes do schema real (usam UUID, snake_case incorreto, ENUMs em lowercase)
2. **Repositórios Manuais Conflitantes**: 4 repositórios (contact, task, company, pipeline) com queries que contradizem o schema real
3. **Dívida de Tipagem**: 62+ ocorrências de `uuid.UUID` em código que deveria usar `string` (TEXT no Prisma)
4. **ENUMs em Lowercase**: 12 validações `validate:"oneof=..."` usando lowercase (banco usa UPPERCASE)

---

## 🗑️ [DELETAR] - Arquivos para Remoção Imediata

### 1. Migrations Obsoletas (6 pares = 12 arquivos)

#### 🚨 **000002_contacts.up.sql** & **000002_contacts.down.sql**
**Razão**: Migration cria tabela com UUID e snake_case. Schema real usa TEXT IDs e camelCase.

**Conflitos identificados**:
```sql
-- Migration (ERRADO):
id UUID PRIMARY KEY DEFAULT gen_random_uuid()
workspace_id UUID NOT NULL
name VARCHAR(255) NOT NULL  -- Schema real usa "fullName"

-- Schema Real (CORRETO):
"id" TEXT NOT NULL
"workspaceId" TEXT NOT NULL
"fullName" TEXT NOT NULL
```

**Impacto**: Esta migration nunca pode rodar no banco real (schema incompatível).

---

#### 🚨 **000004_init_tasks_and_position.up.sql** & **000004_init_tasks_and_position.down.sql**
**Razão**: Cria ENUMs em lowercase e usa UUID. Schema real usa UPPERCASE e TEXT.

**Conflitos identificados**:
```sql
-- Migration (ERRADO):
CREATE TYPE public."Priority" AS ENUM ('low', 'medium', 'high', 'urgent');
CREATE TYPE public."TaskStatus" AS ENUM ('backlog', 'todo', 'in_progress', 'in_review', 'done', 'cancelled');
id UUID PRIMARY KEY DEFAULT gen_random_uuid()

-- Schema Real (CORRETO):
CREATE TYPE "Priority" AS ENUM ('LOW', 'MEDIUM', 'HIGH', 'URGENT');
CREATE TYPE "TaskStatus" AS ENUM ('TODO', 'IN_PROGRESS', 'DONE', 'CANCELLED');
"id" TEXT NOT NULL
```

**Impacto**: ENUM values não batem (lowercase vs UPPERCASE). Queries falharão.

---

#### 🚨 **000005_companies.up.sql** & **000005_companies.down.sql**
**Razão**: Usa ENUMs em lowercase e nomenclatura incorreta. Schema real usa UPPERCASE e campos diferentes.

**Conflitos identificados**:
```sql
-- Migration (ERRADO):
CREATE TYPE public."CompanyLifecycleStage" AS ENUM ('lead', 'prospect', 'customer', 'partner', 'inactive', 'evangelist');
CREATE TYPE public."CompanySize" AS ENUM ('solopreneur', 'small', 'medium', 'midmarket', 'enterprise', 'large_enterprise');
"companySize" public."CompanySize" NOT NULL DEFAULT 'small'

-- Schema Real (CORRETO):
CREATE TYPE "CompanyLifecycleStage" AS ENUM ('LEAD', 'MQL', 'SQL', 'CUSTOMER', 'CHURNED');
CREATE TYPE "CompanySize" AS ENUM ('STARTUP', 'SMB', 'MID_MARKET', 'ENTERPRISE');
"size" "CompanySize" NOT NULL
```

**Impacto**: ENUMs diferentes (prospect/partner/inactive vs MQL/SQL/CHURNED). Field name diferente (companySize vs size).

---

#### 🚨 **000006_pipelines.up.sql** & **000006_pipelines.down.sql**
**Razão**: Usa ENUMs em lowercase e campos obsoletos (pipelineType, isActive, ownerId removidos no schema real).

**Conflitos identificados**:
```sql
-- Migration (ERRADO):
CREATE TYPE public."StageGroup" AS ENUM ('active', 'won', 'lost');
CREATE TYPE public."PipelineType" AS ENUM ('sales', 'recruitment', 'onboarding', 'custom');
"pipelineType" public."PipelineType" NOT NULL DEFAULT 'sales'
"isActive" BOOLEAN NOT NULL DEFAULT true
"ownerId" UUID NOT NULL

-- Schema Real (CORRETO):
CREATE TYPE "StageGroup" AS ENUM ('OPEN', 'ACTIVE', 'DONE', 'CLOSED');
CREATE TYPE "PipelineType" AS ENUM ('TASK', 'DEAL', 'TICKET', 'CONTACT');
-- Campos pipelineType, isActive, ownerId NÃO EXISTEM no schema real
```

**Impacto**: Schema structure completamente diferente. Migration inútil.

---

### 2. Queries Manuais em Repositórios (Parcial)

Estes arquivos **NÃO DEVEM SER DELETADOS**, mas suas queries manuais estão conflitantes e devem ser **SUBSTITUÍDAS** por sqlc-generated code.

#### ⚠️ **internal/repo/contact.go**
**Problema**: Query usa `name` (coluna não existe). Schema real usa `"fullName"`.

```go
// ERRADO (linha 36):
SELECT id, workspace_id, name, email, phone, owner_id, company_id, ...
FROM contacts

// CORRETO:
SELECT "id", "workspaceId", "fullName", email, phone, "ownerId", "companyId", ...
FROM "Contact"
```

**Status**: ✅ Já corrigido no commit 8cbf051 (FullName implementado), mas queries manuais ainda existem.

---

#### ⚠️ **internal/repo/task.go**
**Problema**: Queries usam lowercase ENUMs. Schema real exige UPPERCASE.

```go
// ERRADO (linha ~150):
WHERE workspace_id = $1 AND status = 'todo' AND deleted_at IS NULL

// CORRETO:
WHERE "workspaceId" = $1 AND status = 'TODO' AND "deletedAt" IS NULL
```

**Impacto**: Queries falham (PostgreSQL ENUM case-sensitive).

---

#### ⚠️ **internal/repo/company.go**
**Problema**: Usa `"companySize"` e `"annualRevenue"`. Schema real usa `"size"` e `"revenue"`.

```go
// ERRADO (linhas 208-327):
UPDATE public."Company" SET "companySize" = $1, "annualRevenue" = $2 ...

// CORRETO:
UPDATE public."Company" SET "size" = $1, "revenue" = $2 ...
```

**Status**: ✅ Já corrigido no commit 8cbf051 (Size/Revenue implementados), mas queries manuais ainda existem.

---

#### ⚠️ **internal/repo/pipeline.go**
**Problema**: Usa campos obsoletos (`"pipelineType"`, `"isActive"`, `"ownerId"`) removidos do schema real.

```go
// ERRADO (linha ~100):
INSERT INTO public."Pipeline" ("id", "workspaceId", name, "pipelineType", "isActive", "ownerId", ...)

// CORRETO (campos removidos do schema):
INSERT INTO public."Pipeline" ("id", "workspaceId", name, description, ...)
```

**Impacto**: INSERT falhará (colunas não existem).

---

### 3. ❌ NÃO DELETAR - Migrations de Infraestrutura

Estas migrations são válidas e devem ser mantidas (não conflitam com schema real):

- ✅ **000001_idempotency_audit.up/down.sql**: Tabelas auxiliares (`idempotency_keys`, `audit_log`) - não afetam schema core
- ✅ **000003_workspace_rbac.up/down.sql**: Tabelas RBAC (`WorkspaceRole`, `WorkspaceMember`) - estrutura correta (TEXT IDs)

---

## 🔧 [REFATORAR] - Arquivos Core com Dívida Técnica

### 1. Dívida de Tipagem (UUID → string)

#### 📁 **internal/repo/contact.go**
**Ocorrências**: 3 métodos usam `uuid.UUID` em assinaturas.

```go
// ERRADO (linhas 115, 174, 264):
func (r *ContactRepository) Get(ctx context.Context, workspaceID, contactID uuid.UUID) (*domain.Contact, error)
func (r *ContactRepository) Update(ctx context.Context, workspaceID, contactID uuid.UUID, updates *domain.UpdateContactRequest, expectedUpdatedAt time.Time) (*domain.Contact, error)
func (r *ContactRepository) SoftDelete(ctx context.Context, workspaceID, contactID uuid.UUID) error

// CORRETO:
func (r *ContactRepository) Get(ctx context.Context, workspaceID, contactID string) (*domain.Contact, error)
func (r *ContactRepository) Update(ctx context.Context, workspaceID, contactID string, updates *domain.UpdateContactRequest, expectedUpdatedAt time.Time) (*domain.Contact, error)
func (r *ContactRepository) SoftDelete(ctx context.Context, workspaceID, contactID string) error
```

**Ação**: Remover `import "github.com/google/uuid"` + trocar todas ocorrências de `uuid.UUID` por `string`.

---

#### 📁 **internal/repo/task.go**
**Ocorrências**: 7 métodos usam `uuid.UUID` em assinaturas.

```go
// ERRADO (linhas 150, 186, 223, 295, 367, 387, 408):
func (r *TaskRepository) Get(ctx context.Context, workspaceID, taskID uuid.UUID) (*domain.Task, error)
func (r *TaskRepository) GetForUpdate(ctx context.Context, tx pgx.Tx, workspaceID, taskID uuid.UUID) (*domain.Task, error)
func (r *TaskRepository) GetPositionBounds(ctx context.Context, tx pgx.Tx, workspaceID uuid.UUID, status domain.TaskStatus, beforeID, afterID *uuid.UUID) (*float64, *float64, error)
// ... (4 more methods)

// CORRETO:
func (r *TaskRepository) Get(ctx context.Context, workspaceID, taskID string) (*domain.Task, error)
func (r *TaskRepository) GetPositionBounds(ctx context.Context, tx pgx.Tx, workspaceID string, status domain.TaskStatus, beforeID, afterID *string) (*float64, *float64, error)
// ... etc
```

**Ação**: Substituir `uuid.UUID` por `string` + remover import uuid.

---

#### 📁 **internal/repo/company.go**
**Ocorrências**: 4 métodos usam `uuid.UUID`.

```go
// ERRADO (linhas 138, 208, 327, 348):
func (r *CompanyRepository) Get(ctx context.Context, workspaceID, companyID uuid.UUID) (*domain.Company, error)
func (r *CompanyRepository) Update(ctx context.Context, workspaceID, companyID uuid.UUID, req *domain.UpdateCompanyRequest) error
func (r *CompanyRepository) SoftDelete(ctx context.Context, workspaceID, companyID uuid.UUID) error
func (r *CompanyRepository) ExistsInWorkspace(ctx context.Context, workspaceID, companyID uuid.UUID) (bool, error)

// CORRETO:
func (r *CompanyRepository) Get(ctx context.Context, workspaceID, companyID string) (*domain.Company, error)
// ... etc
```

---

#### 📁 **internal/repo/workspace.go**
**Ocorrências**: 5 métodos usam `uuid.UUID`.

```go
// ERRADO (linhas 59, 98, 124, 164):
func (r *WorkspaceRepository) GetMemberRole(ctx context.Context, userID uuid.UUID, workspaceID uuid.UUID) (domain.Role, error)
func (r *WorkspaceRepository) IsMember(ctx context.Context, userID uuid.UUID, workspaceID uuid.UUID) (bool, error)
func (r *WorkspaceRepository) ListMembersByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]domain.WorkspaceMember, error)
func (r *WorkspaceRepository) ListWorkspacesByUser(ctx context.Context, userID uuid.UUID) ([]domain.WorkspaceMember, error)

// CORRETO:
func (r *WorkspaceRepository) GetMemberRole(ctx context.Context, userID string, workspaceID string) (domain.Role, error)
// ... etc
```

---

#### 📁 **internal/domain/workspace.go**
**Ocorrências**: 3 campos usam `uuid.UUID`.

```go
// ERRADO (linhas 66-67, 73):
type WorkspaceMember struct {
	UserID      uuid.UUID `json:"userId" db:"userId"`
	WorkspaceID uuid.UUID `json:"workspaceId" db:"workspaceId"`
	// ...
	InvitedBy  *uuid.UUID `json:"invitedBy,omitempty" db:"invited_by"`
}

// CORRETO:
type WorkspaceMember struct {
	UserID      string `json:"userId" db:"userId"`
	WorkspaceID string `json:"workspaceId" db:"workspaceId"`
	// ...
	InvitedBy  *string `json:"invitedBy,omitempty" db:"invited_by"`
}
```

**Ação**: Remover `import "github.com/google/uuid"`.

---

### 2. Service Layer - uuid.New() e uuid.UUID params

#### 📁 **internal/service/contact.go**
**Ocorrências**: 5 métodos + 1 uuid.New().

```go
// ERRADO (linhas 43, 80, 107, 140, 196, 264):
func (s *ContactService) ListContacts(ctx context.Context, workspaceID, actorID uuid.UUID, params domain.ListContactsParams) (*domain.ContactListResponse, error)
func (s *ContactService) CreateContact(ctx context.Context, workspaceID, actorID uuid.UUID, req *domain.CreateContactRequest) (*domain.Contact, error) {
	// ...
	ID: uuid.New(),  // ERRADO
}

// CORRETO:
func (s *ContactService) ListContacts(ctx context.Context, workspaceID, actorID string, params domain.ListContactsParams) (*domain.ContactListResponse, error)
func (s *ContactService) CreateContact(ctx context.Context, workspaceID, actorID string, req *domain.CreateContactRequest) (*domain.Contact, error) {
	// ...
	ID: generateID(),  // Usar função própria ou deixar banco gerar
}
```

**Ação**: Substituir todos `uuid.UUID` por `string` + remover `uuid.New()` (usar TEXT generation do banco).

---

#### 📁 **internal/service/task.go**
**Ocorrências**: 5 métodos + 1 uuid.New().

```go
// ERRADO (linhas 47, 82, 107, 124, 187, 242, 305):
func (s *TaskService) CreateTask(ctx context.Context, workspaceID, actorID uuid.UUID, req *domain.CreateTaskRequest) (*domain.Task, error) {
	// ...
	ID: uuid.New(),  // ERRADO
	Status: domain.TaskStatusBacklog, // ERRADO (deveria ser TODO)
	Priority: domain.PriorityMedium,  // CORRETO
}

// CORRETO:
func (s *TaskService) CreateTask(ctx context.Context, workspaceID, actorID string, req *domain.CreateTaskRequest) (*domain.Task, error) {
	// ...
	ID: generateID(),
	Status: domain.TaskStatusTodo, // Schema real usa TODO, não BACKLOG
}
```

**Ação**: Substituir `uuid.UUID` → `string` + remover `uuid.New()`.

---

#### 📁 **internal/service/company.go**
**Ocorrências**: 4 métodos + 1 uuid.New().

```go
// ERRADO (linhas 36, 71, 97, 113, 185, 240):
func (s *CompanyService) CreateCompany(ctx context.Context, workspaceID, actorID uuid.UUID, req *domain.CreateCompanyRequest) (*domain.Company, error) {
	// ...
	ID: uuid.New(),  // ERRADO
}

// CORRETO:
func (s *CompanyService) CreateCompany(ctx context.Context, workspaceID, actorID string, req *domain.CreateCompanyRequest) (*domain.Company, error) {
	// ...
	ID: generateID(),
}
```

---

#### 📁 **internal/service/pipeline.go**
**Ocorrências**: 8 métodos + 4 uuid.New().

```go
// ERRADO (linhas 39, 73, 99, 121, 195, 224, 257, 322, 405, 459, 491, 525, 569, 628, 684, 736, 754, 791):
func (s *PipelineService) CreatePipeline(ctx context.Context, workspaceID, actorID uuid.UUID, req *domain.CreatePipelineRequest) (*domain.Pipeline, error) {
	// ...
	ID: uuid.New(),  // ERRADO (4 ocorrências)
}

// CORRETO:
func (s *PipelineService) CreatePipeline(ctx context.Context, workspaceID, actorID string, req *domain.CreatePipelineRequest) (*domain.Pipeline, error) {
	// ...
	ID: generateID(),
}
```

**Ação**: Substituir 8 métodos + remover 4 chamadas `uuid.New()`.

---

### 3. Handler Layer - uuid.Parse()

#### 📁 **internal/http/handler/contact.go**
**Ocorrências**: 15 chamadas `uuid.Parse()`.

```go
// ERRADO (linhas 34, 46, 71, 80, 119, 126, 138, 171, 183, 231, 238, 250, 296, 303, 315):
workspaceID, err := uuid.Parse(workspaceIDStr)
if err != nil {
	writeError(w, ctx, log, http.StatusBadRequest, "INVALID_WORKSPACE_ID", "Invalid workspace ID format")
	return
}

// CORRETO:
workspaceID := workspaceIDStr // IDs já são strings no path param
```

**Ação**: Remover TODAS as chamadas `uuid.Parse()` + usar strings diretamente.

---

#### 📁 **internal/http/handler/task.go**
**Ocorrências**: 17 chamadas `uuid.Parse()`.

```go
// ERRADO (linhas 32, 44, 97, 106, 115, 148, 155, 167, 195, 207, 241, 248, 260, 294, 301, 313, 342, 349, 361):
workspaceID, err := uuid.Parse(workspaceIDStr)
actorID, err := uuid.Parse(claims.ActorID)
taskID, err := uuid.Parse(taskIDStr)
// ... (17 total)

// CORRETO:
workspaceID := workspaceIDStr
actorID := claims.ActorID
taskID := taskIDStr
```

---

#### 📁 **internal/http/handler/company.go**
**Ocorrências**: 15 chamadas `uuid.Parse()`.

```go
// ERRADO (linhas 34, 46, 101, 140, 147, 159, 191, 203, 242, 249, 261, 300, 307, 319):
workspaceID, err := uuid.Parse(workspaceIDStr)
// ... (15 total)

// CORRETO:
workspaceID := workspaceIDStr
```

---

#### 📁 **internal/http/handler/pipeline.go**
**Ocorrências**: 17 chamadas `uuid.Parse()`.

```go
// ERRADO (linhas 34, 46, 98, 138, 145, 157, 189, 201, 240, 252, 292, 299, 311, 350, 357, 369, 401, 413, 446, 453, 465):
workspaceID, err := uuid.Parse(workspaceIDStr)
// ... (17 total)

// CORRETO:
workspaceID := workspaceIDStr
```

**Total Handler Layer**: **64 chamadas uuid.Parse()** para remover.

---

### 4. Validações com ENUMs em Lowercase

#### 📁 **internal/domain/task.go**
**Problema**: Validações `oneof=...` usam lowercase (schema usa UPPERCASE).

```go
// ERRADO (linhas 217-219, 240-241, 264):
Status   *TaskStatus `json:"status,omitempty" validate:"omitempty,oneof=backlog todo in_progress in_review done cancelled"`
Priority *Priority   `json:"priority,omitempty" validate:"omitempty,oneof=low medium high urgent"`
Type     *TaskType   `json:"type,omitempty" validate:"omitempty,oneof=task bug feature improvement research"`
ToStatus TaskStatus  `json:"toStatus" validate:"required,oneof=backlog todo in_progress in_review done cancelled"`

// CORRETO:
Status   *TaskStatus `json:"status,omitempty" validate:"omitempty,oneof=TODO IN_PROGRESS DONE CANCELLED"`
Priority *Priority   `json:"priority,omitempty" validate:"omitempty,oneof=LOW MEDIUM HIGH URGENT"`
Type     *TaskType   `json:"type,omitempty" validate:"omitempty,oneof=CALL EMAIL MEETING FOLLOWUP OTHER"`
ToStatus TaskStatus  `json:"toStatus" validate:"required,oneof=TODO IN_PROGRESS DONE CANCELLED"`
```

**Nota**: Schema real **NÃO TEM** `BACKLOG`, `IN_REVIEW`, `TASK`, `BUG`, `FEATURE`, `IMPROVEMENT`, `RESEARCH`. Validação deve refletir ENUMs reais.

---

#### 📁 **internal/domain/company.go**
**Problema**: Validações usam lowercase + ENUMs incorretos.

```go
// ERRADO (linhas 169-170, 202-203):
LifecycleStage *CompanyLifecycleStage `json:"lifecycleStage,omitempty" validate:"omitempty,oneof=lead prospect customer partner inactive evangelist"`
Size           *CompanySize           `json:"size,omitempty" validate:"omitempty,oneof=solopreneur small medium midmarket enterprise large_enterprise"`

// CORRETO (schema real):
LifecycleStage *CompanyLifecycleStage `json:"lifecycleStage,omitempty" validate:"omitempty,oneof=LEAD MQL SQL CUSTOMER CHURNED"`
Size           *CompanySize           `json:"size,omitempty" validate:"omitempty,oneof=STARTUP SMB MID_MARKET ENTERPRISE"`
```

**Nota**: Schema real **NÃO TEM** `prospect`, `partner`, `inactive`, `evangelist`, `solopreneur`, `small`, `medium`, `midmarket`, `large_enterprise`.

---

#### 📁 **internal/domain/pipeline.go**
**Problema**: Validações usam lowercase + ENUMs incorretos.

```go
// ERRADO (linhas 190, 207-208):
StageGroup    *StageGroup   `json:"stageGroup,omitempty" validate:"omitempty,oneof=active won lost"`
Group         *StageGroup   `json:"group,omitempty" validate:"omitempty,oneof=active won lost"`
Type          *PipelineType `json:"type,omitempty" validate:"omitempty,oneof=sales recruitment onboarding custom"`

// CORRETO (schema real):
StageGroup    *StageGroup   `json:"stageGroup,omitempty" validate:"omitempty,oneof=OPEN ACTIVE DONE CLOSED"`
Group         *StageGroup   `json:"group,omitempty" validate:"omitempty,oneof=OPEN ACTIVE DONE CLOSED"`
Type          *PipelineType `json:"type,omitempty" validate:"omitempty,oneof=TASK DEAL TICKET CONTACT"`
```

**Nota**: Schema real usa `OPEN/ACTIVE/DONE/CLOSED` (4 valores), não `active/won/lost` (3 valores).

---

#### 📁 **internal/http/handler/task.go**
**Problema**: Mensagens de erro usam lowercase.

```go
// ERRADO (linhas 72, 81, 376):
writeError(w, ctx, log, http.StatusBadRequest, "INVALID_STATUS", "status must be one of: backlog, todo, in_progress, in_review, done, cancelled")
writeError(w, ctx, log, http.StatusBadRequest, "INVALID_PRIORITY", "priority must be one of: low, medium, high, urgent")
writeError(w, ctx, log, http.StatusBadRequest, "INVALID_STATUS", "toStatus must be one of: backlog, todo, in_progress, in_review, done, cancelled")

// CORRETO:
writeError(w, ctx, log, http.StatusBadRequest, "INVALID_STATUS", "status must be one of: TODO, IN_PROGRESS, DONE, CANCELLED")
writeError(w, ctx, log, http.StatusBadRequest, "INVALID_PRIORITY", "priority must be one of: LOW, MEDIUM, HIGH, URGENT")
writeError(w, ctx, log, http.StatusBadRequest, "INVALID_STATUS", "toStatus must be one of: TODO, IN_PROGRESS, DONE, CANCELLED")
```

---

## ✅ [KEEP] - Arquivos Alinhados com Schema Real

### 1. SQLc Configuration & Generated Code

#### ✅ **sqlc.yaml**
**Status**: Configurado corretamente após commit 8cbf051.
- IDs configurados como TEXT (sem UUID overrides)
- JSONB → json.RawMessage
- TEXT[] → pgtype.StringSlice

#### ✅ **internal/repo/sqlc/schema.sql**
**Status**: Sincronizado com `NEXTCRM_DATABASE_SCHEMAS_SQL.sql`.
- 585 linhas, 9 tabelas, 9 ENUMs
- Todos IDs são TEXT
- ENUMs em UPPERCASE
- Naming convention camelCase

#### ✅ **internal/repo/sqlc/models.go**
**Status**: Gerado corretamente pelo sqlc (715 linhas).
- ENUMs com valores UPPERCASE (CompanyLifecycleStageLEAD, PriorityLOW, TaskStatusTODO, etc.)
- IDs são `string` (não uuid.UUID)

#### ✅ **internal/repo/sqlc/db.go** & **querier.go** & **tasks.sql.go**
**Status**: Código gerado validado. Pronto para uso.

---

### 2. Domain Layer ENUMs (Parcialmente Correto)

#### ✅ **internal/domain/contact.go**
**Status**: ✅ Completamente alinhado.
- IDs são `string`
- FullName implementado (Name removido)
- Nenhum import uuid

#### ✅ **internal/domain/task.go**
**Status**: ⚠️ ENUMs UPPERCASE corretos, mas validações precisam correção.
- ENUMs: `PriorityLOW`, `TaskStatusTODO`, `TaskTypeCALL` (UPPERCASE ✅)
- Validações: Ainda usam lowercase (INCONSISTENTE ❌)

#### ✅ **internal/domain/company.go**
**Status**: ⚠️ ENUMs UPPERCASE corretos, mas validações desatualizadas.
- ENUMs: `LifecycleLEAD`, `SizeSTARTUP` (UPPERCASE ✅)
- Validações: Usam valores obsoletos (`prospect`, `solopreneur`) (INCONSISTENTE ❌)

#### ✅ **internal/domain/pipeline.go**
**Status**: ⚠️ ENUMs UPPERCASE corretos, mas validações desatualizadas.
- ENUMs: `StageGroupOPEN`, `PipelineTypeTASK` (UPPERCASE ✅)
- Validações: Usam valores obsoletos (`active/won/lost` vs `OPEN/ACTIVE/DONE/CLOSED`) (INCONSISTENTE ❌)

---

### 3. Migrations de Infraestrutura (Válidas)

#### ✅ **000001_idempotency_audit.up.sql**
**Status**: Válida (não conflita com schema real).
- Cria tabelas auxiliares (`idempotency_keys`, `audit_log`)
- Usa UUID para IDs internos (OK, são tabelas separadas)

#### ✅ **000003_workspace_rbac.up.sql**
**Status**: Válida e alinhada.
- Tabelas `WorkspaceRole` e `WorkspaceMember`
- IDs usam TEXT (correto: `id TEXT PRIMARY KEY`)
- FKs usam `userId` UUID (correto para relacionamento com Supabase User)

---

## 📊 ESTATÍSTICAS DE DÍVIDA TÉCNICA

| Categoria | Arquivos Afetados | Ocorrências | Prioridade |
|-----------|-------------------|-------------|------------|
| **Migrations Obsoletas** | 6 pares (12 arquivos) | 6 migrations | 🔴 CRÍTICA |
| **Repositórios Manuais** | 4 arquivos | ~50 queries | 🔴 CRÍTICA |
| **uuid.UUID em Repos** | 4 arquivos | 19 métodos | 🟡 ALTA |
| **uuid.UUID em Services** | 4 arquivos | 22 métodos + 7 uuid.New() | 🟡 ALTA |
| **uuid.Parse() em Handlers** | 4 arquivos | 64 chamadas | 🟡 ALTA |
| **uuid.UUID em Domain** | 1 arquivo (workspace.go) | 3 campos | 🟢 MÉDIA |
| **Validações Lowercase** | 3 arquivos (domain) | 12 tags | 🟢 MÉDIA |
| **Mensagens Erro Lowercase** | 1 arquivo (handler) | 3 mensagens | 🟢 BAIXA |

**Total de Arquivos com Dívida**: 21 arquivos  
**Total de Ocorrências**: 200+ issues

---

## 🎯 PLANO DE LIMPEZA PROPOSTO

### FASE 1: Remoção de Migrations Obsoletas (IMEDIATO)
**Ação**: Deletar 12 arquivos de migrations (000002, 000004, 000005, 000006).

```bash
# Comando de exclusão:
rm internal/database/migrations/000002_contacts.{up,down}.sql
rm internal/database/migrations/000004_init_tasks_and_position.{up,down}.sql
rm internal/database/migrations/000005_companies.{up,down}.sql
rm internal/database/migrations/000006_pipelines.{up,down}.sql
```

**Justificativa**: Estas migrations NUNCA podem rodar no banco real (schemas incompatíveis).

**Impacto**: Zero (código não usa migrations para banco real, usa schema Prisma).

---

### FASE 2: Correção de Validações Domain Layer (URGENTE)
**Ação**: Corrigir 12 tags `validate:"oneof=..."` em 3 arquivos (task.go, company.go, pipeline.go).

**Arquivos**:
- `internal/domain/task.go`: 6 validações
- `internal/domain/company.go`: 4 validações
- `internal/domain/pipeline.go`: 2 validações

**Impacto**: API rejeitará valores UPPERCASE corretos enquanto não for corrigido.

---

### FASE 3: Migração Repos Manuais → SQLc (GRADUAL)
**Ação**: Criar queries sqlc para cada domínio + wrapper repos.

**Ordem**:
1. **Contacts** (FASE 2 já iniciada): `contacts.sql` + wrapper
2. **Tasks** (FASE 3): Expandir `tasks.sql` + wrapper
3. **Companies**: `companies.sql` + wrapper
4. **Pipelines**: `pipelines.sql` + wrapper

**Impacto**: Elimina 50+ queries manuais conflitantes.

---

### FASE 4: Correção UUID → string (SISTEMÁTICA)
**Ação**: Substituir `uuid.UUID` por `string` em 21 arquivos.

**Ordem de Execução**:
1. Domain layer (workspace.go)
2. Repository layer (contact, task, company, pipeline, workspace)
3. Service layer (contact, task, company, pipeline)
4. Handler layer (contact, task, company, pipeline)

**Ferramenta**: Multi-replace batch com validação de compilação após cada layer.

---

### FASE 5: Limpeza de Mensagens de Erro (POLIMENTO)
**Ação**: Corrigir 3 mensagens de erro em `task.go` handler.

**Impacto**: Mensagens de erro corretas para frontend.

---

## ⚠️ RISCOS E BLOQUEIOS

### 🔴 BLOQUEIO CRÍTICO: Migrations Obsoletas
**Problema**: Se alguém tentar rodar migrations 000002-000006 no banco real, causará falha total (ENUMs duplicados, conflito de schemas).

**Solução**: Deletar imediatamente após aprovação.

---

### 🟡 RISCO ALTO: Queries Manuais Conflitantes
**Problema**: Repositórios atuais usam queries com colunas/ENUMs incorretos. Produção falhará.

**Solução**: Priorizar FASE 3 (migração para sqlc).

---

### 🟢 RISCO MÉDIO: Validações Domain
**Problema**: API aceita lowercase mas banco rejeita UPPERCASE (ou vice-versa).

**Solução**: Correção rápida (FASE 2) resolve.

---

## 📝 CHECKLIST DE APROVAÇÃO

Antes de executar o plano de limpeza, confirme:

- [ ] **Backup do repositório criado** (commit atual é rollback point)
- [ ] **Schema real Prisma confirmado como fonte da verdade** (NEXTCRM_DATABASE_SCHEMAS_SQL.sql)
- [ ] **Migrations obsoletas identificadas corretamente** (6 pares confirmados)
- [ ] **Repositórios manuais mapeados** (4 arquivos confirmados)
- [ ] **Dívida UUID quantificada** (200+ ocorrências confirmadas)
- [ ] **Plano de execução revisado** (FASE 1-5 aprovadas)
- [ ] **Equipe alinhada sobre breaking changes** (validações lowercase→UPPERCASE)

---

## 🚀 PRÓXIMOS PASSOS

Após aprovação deste relatório:

1. **GPTM aprova FASE 1** → Executar exclusão de migrations
2. **GPTM aprova FASE 2** → Corrigir validações domain layer
3. **GPTM aprova FASE 3** → Iniciar migração repos para sqlc (Contacts primeiro)
4. **FASE 4 e 5** → Aguardar FASE 3 completar

**Duração Estimada Total**: 3-5 dias de trabalho (distribuído em sprints).

---

## 📌 NOTAS FINAIS

Este relatório documenta **TODA** a dívida técnica identificada após sincronização com schema real Prisma. 

**Nenhuma ação foi executada automaticamente** - todas as exclusões/refatorações aguardam aprovação explícita do GPTM.

**Versão do Relatório**: 1.0  
**Última Atualização**: 20/01/2026 - 23:45 BRT  
**Gerado por**: GitHub Copilot (Agent: Claude Sonnet 4.5)

---

**🔐 CONFIDENCIAL - USO INTERNO LINKKO PLATFORM**
