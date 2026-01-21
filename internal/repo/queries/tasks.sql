-- =====================================================
-- Task Queries (Schema Real Sincronizado)
-- =====================================================
-- IMPORTANTE: Campos seguem schema real do Prisma
-- IDs são TEXT, não UUID
-- =====================================================

-- name: GetTask :one
-- Buscar task por ID com isolamento multi-tenant
SELECT 
    "id", "title", "workspaceId", "description",
    "status", "priority", "type", "taskType", "reminderType",
    "dueDate", "completedAt", "deletedAt",
    "companyId", "contactId", "dealId", "assignedToId", "stageId",
    "createdAt", "updatedAt"
FROM "Task"
WHERE "id" = $1 
  AND "workspaceId" = $2 
  AND "deletedAt" IS NULL;

-- name: ListTasks :many
-- Listar tasks com filtros opcionais
SELECT 
    "id", "title", "workspaceId", "description",
    "status", "priority", "type", "taskType", "reminderType",
    "dueDate", "completedAt", "deletedAt",
    "companyId", "contactId", "dealId", "assignedToId", "stageId",
    "createdAt", "updatedAt"
FROM "Task"
WHERE "workspaceId" = $1 
  AND "deletedAt" IS NULL
  AND (sqlc.narg('filter_status')::"TaskStatus" IS NULL OR "status" = sqlc.narg('filter_status'))
  AND (sqlc.narg('filter_priority')::"Priority" IS NULL OR "priority" = sqlc.narg('filter_priority'))
ORDER BY "createdAt" DESC
LIMIT $2;

-- name: CreateTask :one
-- Criar nova task retornando o registro completo
INSERT INTO "Task" (
    "id", "title", "workspaceId", "description",
    "status", "priority", "type", "dueDate", "assignedToId"
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING 
    "id", "title", "workspaceId", "description",
    "status", "priority", "type", "taskType", "reminderType",
    "dueDate", "completedAt", "deletedAt",
    "companyId", "contactId", "dealId", "assignedToId", "stageId",
    "createdAt", "updatedAt";
