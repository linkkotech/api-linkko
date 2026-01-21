-- name: GetDeal :one
SELECT 
    d.*,
    c."fullName" as contactName,
    co.name as companyName
FROM "Deal" d
LEFT JOIN "Contact" c ON d."contactId" = c.id
LEFT JOIN "Company" co ON d."companyId" = co.id
WHERE d.id = $1 AND d."workspaceId" = $2 AND d."deletedAt" IS NULL;

-- name: ListDeals :many
SELECT 
    d.*,
    c."fullName" as contactName,
    co.name as companyName
FROM "Deal" d
LEFT JOIN "Contact" c ON d."contactId" = c.id
LEFT JOIN "Company" co ON d."companyId" = co.id
WHERE d."workspaceId" = $1 
    AND (sqlc.narg('pipelineId')::TEXT IS NULL OR d."pipelineId" = sqlc.narg('pipelineId'))
    AND (sqlc.narg('stageId')::TEXT IS NULL OR d."stageId" = sqlc.narg('stageId'))
    AND (sqlc.narg('ownerId')::TEXT IS NULL OR d."ownerId" = sqlc.narg('ownerId'))
    AND d."deletedAt" IS NULL
ORDER BY d."createdAt" DESC;

-- name: CreateDeal :one
INSERT INTO "Deal" (
    id, "workspaceId", "pipelineId", "stageId", "contactId", "companyId",
    name, value, currency, stage, probability, 
    "expectedCloseDate", "ownerId", "createdById", description
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
) RETURNING *;

-- name: UpdateDeal :one
UPDATE "Deal"
SET 
    "pipelineId" = COALESCE(sqlc.narg('pipelineId'), "pipelineId"),
    "stageId" = COALESCE(sqlc.narg('stageId'), "stageId"),
    name = COALESCE(sqlc.narg('name'), name),
    value = COALESCE(sqlc.narg('value'), value),
    currency = COALESCE(sqlc.narg('currency'), currency),
    stage = COALESCE(sqlc.narg('stage'), stage),
    probability = COALESCE(sqlc.narg('probability'), probability),
    "expectedCloseDate" = COALESCE(sqlc.narg('expectedCloseDate'), "expectedCloseDate"),
    "closedAt" = COALESCE(sqlc.narg('closedAt'), "closedAt"),
    "lostReason" = COALESCE(sqlc.narg('lostReason'), "lostReason"),
    "ownerId" = COALESCE(sqlc.narg('ownerId'), "ownerId"),
    description = COALESCE(sqlc.narg('description'), description),
    "updatedAt" = CURRENT_TIMESTAMP,
    "updatedById" = sqlc.narg('updatedById')
WHERE id = $1 AND "workspaceId" = $2 AND "deletedAt" IS NULL
RETURNING *;

-- name: DeleteDeal :exec
UPDATE "Deal"
SET 
    "deletedAt" = CURRENT_TIMESTAMP,
    "deletedById" = $3
WHERE id = $1 AND "workspaceId" = $2;

-- name: CreateDealHistory :one
INSERT INTO "DealStageHistory" (
    id, "workspaceId", "dealId", "fromStage", "toStage", reason, "userId"
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;
