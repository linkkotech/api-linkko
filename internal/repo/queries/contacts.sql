-- =====================================================
-- CONTACTS QUERIES - SQLc Generated
-- =====================================================
-- Tabela: "Contact"
-- Schema: camelCase com aspas duplas
-- IDs: TEXT (não UUID)
-- =====================================================

-- name: GetContact :one
-- Retorna um contato específico de um workspace (IDOR protection).
SELECT 
    "id",
    "fullName",
    "workspaceId",
    "email",
    "phone",
    "whatsapp",
    "notes",
    "firstName",
    "lastName",
    "image",
    "linkedinUrl",
    "language",
    "timezone",
    "city",
    "state",
    "country",
    "jobTitle",
    "department",
    "decisionRole",
    "tagLabels",
    "source",
    "lastInteractionAt",
    "ownerId",
    "socialUrls",
    "companyId",
    "contactScore",
    "lifecycleStage",
    "assignedToId",
    "createdById",
    "updatedById",
    "createdAt",
    "updatedAt",
    "deletedAt",
    "deletedById"
FROM "Contact"
WHERE "id" = $1
  AND "workspaceId" = $2
  AND "deletedAt" IS NULL;

-- name: ListContacts :many
-- Lista contatos de um workspace com paginação cursor-based (created_at DESC).
-- Filtros opcionais: ownerId, companyId, lifecycleStage, query (fulltext search).
SELECT 
    "id",
    "fullName",
    "workspaceId",
    "email",
    "phone",
    "whatsapp",
    "notes",
    "firstName",
    "lastName",
    "image",
    "linkedinUrl",
    "language",
    "timezone",
    "city",
    "state",
    "country",
    "jobTitle",
    "department",
    "decisionRole",
    "tagLabels",
    "source",
    "lastInteractionAt",
    "ownerId",
    "socialUrls",
    "companyId",
    "contactScore",
    "lifecycleStage",
    "assignedToId",
    "createdById",
    "updatedById",
    "createdAt",
    "updatedAt",
    "deletedAt",
    "deletedById"
FROM "Contact"
WHERE "workspaceId" = $1
  AND "deletedAt" IS NULL
  AND ($2::TEXT IS NULL OR "ownerId" = $2)
  AND ($3::TEXT IS NULL OR "companyId" = $3)
  AND ($4::TEXT IS NULL OR "lifecycleStage"::TEXT = $4)
  AND ($5::TEXT IS NULL OR to_tsvector('simple', "fullName" || ' ' || COALESCE("email", '')) @@ plainto_tsquery('simple', $5))
  AND ($6::TIMESTAMP IS NULL OR "createdAt" < $6)
ORDER BY "createdAt" DESC
LIMIT $7;

-- name: CreateContact :one
-- Cria um novo contato no workspace (ID gerado pela aplicação).
INSERT INTO "Contact" (
    "id",
    "fullName",
    "workspaceId",
    "email",
    "phone",
    "whatsapp",
    "notes",
    "firstName",
    "lastName",
    "image",
    "linkedinUrl",
    "language",
    "timezone",
    "city",
    "state",
    "country",
    "jobTitle",
    "department",
    "decisionRole",
    "tagLabels",
    "source",
    "lastInteractionAt",
    "ownerId",
    "socialUrls",
    "companyId",
    "contactScore",
    "lifecycleStage",
    "assignedToId",
    "createdById",
    "updatedById",
    "createdAt",
    "updatedAt"
) VALUES (
    $1,  -- id
    $2,  -- fullName
    $3,  -- workspaceId
    $4,  -- email
    $5,  -- phone
    $6,  -- whatsapp
    $7,  -- notes
    $8,  -- firstName
    $9,  -- lastName
    $10, -- image
    $11, -- linkedinUrl
    $12, -- language
    $13, -- timezone
    $14, -- city
    $15, -- state
    $16, -- country
    $17, -- jobTitle
    $18, -- department
    $19, -- decisionRole
    $20, -- tagLabels
    $21, -- source
    $22, -- lastInteractionAt
    $23, -- ownerId
    $24, -- socialUrls
    $25, -- companyId
    $26, -- contactScore
    $27, -- lifecycleStage
    $28, -- assignedToId
    $29, -- createdById
    $30, -- updatedById
    $31, -- createdAt
    $32  -- updatedAt
)
RETURNING 
    "id",
    "fullName",
    "workspaceId",
    "email",
    "phone",
    "whatsapp",
    "notes",
    "firstName",
    "lastName",
    "image",
    "linkedinUrl",
    "language",
    "timezone",
    "city",
    "state",
    "country",
    "jobTitle",
    "department",
    "decisionRole",
    "tagLabels",
    "source",
    "lastInteractionAt",
    "ownerId",
    "socialUrls",
    "companyId",
    "contactScore",
    "lifecycleStage",
    "assignedToId",
    "createdById",
    "updatedById",
    "createdAt",
    "updatedAt",
    "deletedAt",
    "deletedById";

-- name: UpdateContact :one
-- Atualiza um contato existente (IDOR protection + optimistic locking via updatedAt).
UPDATE "Contact"
SET
    "fullName" = COALESCE($3, "fullName"),
    "email" = COALESCE($4, "email"),
    "phone" = COALESCE($5, "phone"),
    "whatsapp" = COALESCE($6, "whatsapp"),
    "notes" = COALESCE($7, "notes"),
    "firstName" = COALESCE($8, "firstName"),
    "lastName" = COALESCE($9, "lastName"),
    "image" = COALESCE($10, "image"),
    "linkedinUrl" = COALESCE($11, "linkedinUrl"),
    "language" = COALESCE($12, "language"),
    "timezone" = COALESCE($13, "timezone"),
    "city" = COALESCE($14, "city"),
    "state" = COALESCE($15, "state"),
    "country" = COALESCE($16, "country"),
    "jobTitle" = COALESCE($17, "jobTitle"),
    "department" = COALESCE($18, "department"),
    "decisionRole" = COALESCE($19, "decisionRole"),
    "tagLabels" = COALESCE($20, "tagLabels"),
    "source" = COALESCE($21, "source"),
    "lastInteractionAt" = COALESCE($22, "lastInteractionAt"),
    "ownerId" = COALESCE($23, "ownerId"),
    "socialUrls" = COALESCE($24, "socialUrls"),
    "companyId" = COALESCE($25, "companyId"),
    "contactScore" = COALESCE($26, "contactScore"),
    "lifecycleStage" = COALESCE($27, "lifecycleStage"),
    "assignedToId" = COALESCE($28, "assignedToId"),
    "updatedById" = $29,
    "updatedAt" = $30
WHERE "id" = $1
  AND "workspaceId" = $2
  AND "deletedAt" IS NULL
  AND "updatedAt" = $31  -- Optimistic locking
RETURNING 
    "id",
    "fullName",
    "workspaceId",
    "email",
    "phone",
    "whatsapp",
    "notes",
    "firstName",
    "lastName",
    "image",
    "linkedinUrl",
    "language",
    "timezone",
    "city",
    "state",
    "country",
    "jobTitle",
    "department",
    "decisionRole",
    "tagLabels",
    "source",
    "lastInteractionAt",
    "ownerId",
    "socialUrls",
    "companyId",
    "contactScore",
    "lifecycleStage",
    "assignedToId",
    "createdById",
    "updatedById",
    "createdAt",
    "updatedAt",
    "deletedAt",
    "deletedById";

-- name: SoftDeleteContact :exec
-- Soft delete de um contato (marca deletedAt + deletedById).
UPDATE "Contact"
SET
    "deletedAt" = $3,
    "deletedById" = $4,
    "updatedAt" = $3
WHERE "id" = $1
  AND "workspaceId" = $2
  AND "deletedAt" IS NULL;

-- name: SearchContactsByText :many
-- Busca fulltext em contatos (usado por autocomplete/search).
SELECT 
    "id",
    "fullName",
    "email",
    "phone",
    "companyId",
    "lifecycleStage",
    "contactScore"
FROM "Contact"
WHERE "workspaceId" = $1
  AND "deletedAt" IS NULL
  AND to_tsvector('simple', "fullName" || ' ' || COALESCE("email", '') || ' ' || COALESCE("phone", '')) @@ plainto_tsquery('simple', $2)
ORDER BY "contactScore" DESC, "createdAt" DESC
LIMIT $3;

-- name: ContactExistsInWorkspace :one
-- Verifica se um contato existe no workspace (usado por validações).
SELECT EXISTS(
    SELECT 1
    FROM "Contact"
    WHERE "id" = $1
      AND "workspaceId" = $2
      AND "deletedAt" IS NULL
) AS "exists";
