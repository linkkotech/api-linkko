-- name: CreatePortfolioItem :one
INSERT INTO "PortfolioItem" (
    "id",
    "workspaceId",
    "name",
    "description",
    "sku",
    "category",
    "vertical",
    "status",
    "visibility",
    "basePrice",
    "currency",
    "imageUrl",
    "metadata",
    "tags",
    "createdById",
    "updatedAt"
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, CURRENT_TIMESTAMP
)
RETURNING *;

-- name: GetPortfolioItem :one
SELECT * FROM "PortfolioItem"
WHERE "workspaceId" = $1 AND "id" = $2 AND "deletedAt" IS NULL;

-- name: ListPortfolioItems :many
SELECT * FROM "PortfolioItem"
WHERE "workspaceId" = $1
  AND "deletedAt" IS NULL
  AND (sqlc.narg('status')::"PortfolioStatus" IS NULL OR "status" = sqlc.narg('status'))
  AND (sqlc.narg('category')::"PortfolioCategoryEnum" IS NULL OR "category" = sqlc.narg('category'))
  AND (sqlc.narg('query')::TEXT IS NULL OR "name" ILIKE '%' || sqlc.narg('query') || '%' OR "description" ILIKE '%' || sqlc.narg('query') || '%')
ORDER BY "createdAt" DESC;

-- name: UpdatePortfolioItem :one
UPDATE "PortfolioItem"
SET
    "name" = COALESCE(sqlc.narg('name'), "name"),
    "description" = COALESCE(sqlc.narg('description'), "description"),
    "sku" = COALESCE(sqlc.narg('sku'), "sku"),
    "category" = COALESCE(sqlc.narg('category'), "category"),
    "vertical" = COALESCE(sqlc.narg('vertical'), "vertical"),
    "status" = COALESCE(sqlc.narg('status'), "status"),
    "visibility" = COALESCE(sqlc.narg('visibility'), "visibility"),
    "basePrice" = COALESCE(sqlc.narg('basePrice'), "basePrice"),
    "currency" = COALESCE(sqlc.narg('currency'), "currency"),
    "imageUrl" = COALESCE(sqlc.narg('imageUrl'), "imageUrl"),
    "metadata" = COALESCE(sqlc.narg('metadata'), "metadata"),
    "tags" = COALESCE(sqlc.narg('tags'), "tags"),
    "updatedById" = $3,
    "updatedAt" = CURRENT_TIMESTAMP
WHERE "workspaceId" = $1 AND "id" = $2 AND "deletedAt" IS NULL
RETURNING *;

-- name: DeletePortfolioItem :exec
UPDATE "PortfolioItem"
SET "deletedAt" = CURRENT_TIMESTAMP
WHERE "workspaceId" = $1 AND "id" = $2;
