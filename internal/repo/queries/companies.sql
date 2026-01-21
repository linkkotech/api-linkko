-- =====================================================
-- COMPANIES QUERIES - SQLc Generated
-- =====================================================
-- Tabela: "Company"
-- Schema: camelCase com aspas duplas
-- IDs: TEXT (não UUID)
-- ENUMs: CompanyLifecycleStage, CompanySize (UPPERCASE)
-- =====================================================

-- name: GetCompany :one
SELECT 
    "id", "workspaceId", "name", "website", "linkedin",
    "legalName", "phone", "instagram", "policyUrl", "socialUrls",
    "addressLine", "city", "state", "country", "timezone",
    "currency", "locale", "businessHours", "supportHours",
    "deletedAt", "deletedById", "size", "revenue",
    "companyScore", "lifecycleStage", "assignedToId",
    "createdById", "updatedById", "createdAt", "updatedAt"
FROM "Company"
WHERE "id" = $1
  AND "workspaceId" = $2
  AND "deletedAt" IS NULL;

-- name: ListCompanies :many
SELECT 
    "id", "workspaceId", "name", "website", "linkedin",
    "legalName", "phone", "instagram", "policyUrl", "socialUrls",
    "addressLine", "city", "state", "country", "timezone",
    "currency", "locale", "businessHours", "supportHours",
    "deletedAt", "deletedById", "size", "revenue",
    "companyScore", "lifecycleStage", "assignedToId",
    "createdById", "updatedById", "createdAt", "updatedAt"
FROM "Company"
WHERE "workspaceId" = $1
  AND "deletedAt" IS NULL
  AND ($2::TEXT IS NULL OR "lifecycleStage"::TEXT = $2)
  AND ($3::TEXT IS NULL OR "size"::TEXT = $3)
  AND ($4::TEXT IS NULL OR "assignedToId" = $4)
  AND ($5::TEXT IS NULL OR to_tsvector('simple', "name" || ' ' || COALESCE("website", '')) @@ plainto_tsquery('simple', $5))
  AND ($6::TIMESTAMP IS NULL OR "createdAt" < $6)
ORDER BY "createdAt" DESC
LIMIT $7;

-- name: CreateCompany :one
INSERT INTO "Company" (
    "id", "workspaceId", "name", "website", "linkedin",
    "legalName", "phone", "instagram", "policyUrl", "socialUrls",
    "addressLine", "city", "state", "country", "timezone",
    "currency", "locale", "businessHours", "supportHours",
    "size", "revenue", "companyScore", "lifecycleStage",
    "assignedToId", "createdById", "updatedById",
    "createdAt", "updatedAt"
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9, $10,
    $11, $12, $13, $14, $15,
    $16, $17, $18, $19,
    $20, $21, $22, $23,
    $24, $25, $26,
    $27, $28
)
RETURNING 
    "id", "workspaceId", "name", "website", "linkedin",
    "legalName", "phone", "instagram", "policyUrl", "socialUrls",
    "addressLine", "city", "state", "country", "timezone",
    "currency", "locale", "businessHours", "supportHours",
    "deletedAt", "deletedById", "size", "revenue",
    "companyScore", "lifecycleStage", "assignedToId",
    "createdById", "updatedById", "createdAt", "updatedAt";

-- name: UpdateCompany :one
UPDATE "Company"
SET
    "name" = COALESCE($3, "name"),
    "website" = COALESCE($4, "website"),
    "linkedin" = COALESCE($5, "linkedin"),
    "legalName" = COALESCE($6, "legalName"),
    "phone" = COALESCE($7, "phone"),
    "instagram" = COALESCE($8, "instagram"),
    "policyUrl" = COALESCE($9, "policyUrl"),
    "socialUrls" = COALESCE($10, "socialUrls"),
    "addressLine" = COALESCE($11, "addressLine"),
    "city" = COALESCE($12, "city"),
    "state" = COALESCE($13, "state"),
    "country" = COALESCE($14, "country"),
    "timezone" = COALESCE($15, "timezone"),
    "currency" = COALESCE($16, "currency"),
    "locale" = COALESCE($17, "locale"),
    "businessHours" = COALESCE($18, "businessHours"),
    "supportHours" = COALESCE($19, "supportHours"),
    "size" = COALESCE($20, "size"),
    "revenue" = COALESCE($21, "revenue"),
    "companyScore" = COALESCE($22, "companyScore"),
    "lifecycleStage" = COALESCE($23, "lifecycleStage"),
    "assignedToId" = COALESCE($24, "assignedToId"),
    "updatedById" = $25,
    "updatedAt" = $26
WHERE "id" = $1
  AND "workspaceId" = $2
  AND "deletedAt" IS NULL
  AND "updatedAt" = $27
RETURNING 
    "id", "workspaceId", "name", "website", "linkedin",
    "legalName", "phone", "instagram", "policyUrl", "socialUrls",
    "addressLine", "city", "state", "country", "timezone",
    "currency", "locale", "businessHours", "supportHours",
    "deletedAt", "deletedById", "size", "revenue",
    "companyScore", "lifecycleStage", "assignedToId",
    "createdById", "updatedById", "createdAt", "updatedAt";

-- name: SoftDeleteCompany :exec
UPDATE "Company"
SET
    "deletedAt" = $3,
    "deletedById" = $4,
    "updatedAt" = $3
WHERE "id" = $1
  AND "workspaceId" = $2
  AND "deletedAt" IS NULL;

-- name: CompanyExistsInWorkspace :one
SELECT EXISTS(
    SELECT 1
    FROM "Company"
    WHERE "id" = $1
      AND "workspaceId" = $2
      AND "deletedAt" IS NULL
) AS "exists";
