-- name: CreateActivity :one
INSERT INTO "Activity" (
    id, "workspaceId", "companyId", "contactId", "dealId",
    "activityType", "activityId", "userId", metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: CreateNote :one
INSERT INTO "Note" (
    id, "workspaceId", "companyId", "contactId", "dealId",
    content, "isPinned", "userId"
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: CreateCall :one
INSERT INTO "Call" (
    id, "workspaceId", "contactId", "companyId",
    direction, duration, "recordingUrl", summary, "userId", "calledAt"
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: CreateMeeting :one
INSERT INTO "Meeting" (
    id, "workspaceId", title, description, "meetingType",
    "startTime", "endTime", location, "meetingUrl", "externalId", "userId"
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: CreateMessage :one
INSERT INTO "Message" (
    id, "workspaceId", "contactId", "companyId",
    direction, platform, content, status, "sentAt", "userId"
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: ListActivities :many
SELECT * FROM "Activity"
WHERE "workspaceId" = $1
    AND (sqlc.narg('contactId')::TEXT IS NULL OR "contactId" = sqlc.narg('contactId'))
    AND (sqlc.narg('companyId')::TEXT IS NULL OR "companyId" = sqlc.narg('companyId'))
    AND (sqlc.narg('dealId')::TEXT IS NULL OR "dealId" = sqlc.narg('dealId'))
ORDER BY "createdAt" DESC;
