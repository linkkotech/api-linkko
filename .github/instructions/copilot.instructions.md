# Copilot instructions

# Engineering Standards & Code Documentation Guide

You are an AI coding assistant for a Go-based multi-tenant SaaS platform. Follow these standards strictly.

---

## ⚠️ Architectural Authority

- The Copilot MUST NOT invent new architectural patterns, layers, or abstractions.
- Any new domain, cross-cutting concern, or structural change must be proposed, not implemented automatically.
- When uncertain, generate a proposal section instead of code.

---

## 🎯 Core Principles (Non-Negotiable)

1. **Determinism**: `local build = CI build = Docker build = runtime`
2. **Security by default**: Deny by default, allow explicitly
3. **Multi-tenant always explicit**: No queries without `workspaceId` (or equivalent scope)
4. **Contract before implementation**: Define routes + schema + errors + authz before handlers
5. **Observability first-class**: Structured logs + request-id + minimal metrics

---

## Scope Boundary

- These instructions apply to:
  - CRM API
  - MCP Client
  - External Adapters
- Admin / Platform MCP follows a separate instruction set and must not be merged implicitly.

---

## 📡 API Conventions

### Versioning

- All public routes: `/v1/...`
- Future versions: `/v2/...` (no breaking changes within same version)

### REST Standards

**CRUD Operations:**

`GET    /resources
POST   /resources
GET    /resources/{id}
PATCH  /resources/{id}
DELETE /resources/{id}  # soft delete preferred`

**Non-CRUD Actions:**

`POST /resources/{id}:action  # e.g., :publish, :merge, :close`

**Forbidden:** GET with side effects

### IDs & Slugs

- **IDs**: UUID (or ULID) as string
- **Slug**: kebab-case, unique per workspace when applicable

### Multi-Tenant (MANDATORY)

`/v1/workspaces/{workspaceId}/...`

**Requirements:**

- ✅ Every handler MUST filter by `workspaceId`
- ✅ MANDATORY `RequireWorkspaceAccess(workspaceId)` middleware on tenant routes
- ❌ FORBIDDEN: queries without workspace scope

---

## 🔐 Authentication & Authorization

### AuthN

**Required:** `Authorization: Bearer <token>` except:

- `/health`, `/ready`, `/version`
- Explicitly marked `/public/...` endpoints
- Webhooks with verified signatures

### AuthZ (RBAC)

- ✅ Check user role in workspace for every endpoint
- ✅ Principle of least privilege
- ❌ FORBIDDEN: trust token claims without validating membership in DB

### Webhooks

- ✅ Validate signature (HMAC/secret) before processing
- ✅ Enforce idempotency via `event_id` to prevent duplicates

---

## 📋 Contracts (Request/Response)

### Input Validation

- ✅ Validate request body/query/path params with schema (OpenAPI/JSON Schema)
- ✅ Reject unknown fields (strict mode)
- ❌ FORBIDDEN: accept payload without validation

### Response Format

**Success:**

- `200` (read/update)
- `201` (create)
- `204` (delete)
- Always JSON (except 204)

**Error (always JSON):**

json

`{
  "code": "string",
  "message": "string",
  "details": { "optional": true }
}
```

### Status Codes (MANDATORY)
```
400 - Validation error
401 - Unauthenticated
403 - Forbidden (no permission)
404 - Not found (within workspace)
409 - Conflict (unique/index violation)
422 - Business rule violation
429 - Rate limit / quota exceeded
500 - Unexpected error
```

---

## 📄 Pagination, Filters & Search

### Pagination (cursor-based)
```
?limit=50&cursor=...
```
❌ FORBIDDEN: offset-based pagination for large endpoints (CRM, messages, activity)

### Sorting
```
?sort=createdAt   # ascending
?sort=-createdAt  # descending (prefix -)
```

### Search
```
?q=              # text search
?ownerId=        # explicit filters
?status=
?stageId=
```

---

## 🗄️ Database & Data Integrity

### Transactions
- ✅ MANDATORY for operations modifying multiple tables
- ❌ FORBIDDEN: silent "best effort" approaches

### Soft Delete (default)
- Add `deletedAt` to core models (Contacts, Companies, Deals, Tasks, etc.)
- Listings exclude `deletedAt != null` by default
- Restore via explicit action: `POST /{id}:restore` when applicable

### Concurrency Control
- ✅ MANDATORY for sensitive updates:
  - `updatedAt` check (optimistic concurrency) OR etag
- ❌ FORBIDDEN: overwrite without detecting conflicts on critical entities (deal, contact)

---

## ⚡ Rate Limits & Quotas

**Mental separation (MANDATORY):**
- **Rate limit** = technical protection (`429`)
- **Quota** = plan limit (`403`/`429` depending on context)

❌ FORBIDDEN: confuse these in new routes
- Legacy `MessageRateLimit` must be documented for migration

---

## 📊 Observability & Diagnostics

### Structured Logs
**MANDATORY fields (JSON):**
```
requestId, workspaceId, userId, route, status, latencyMs
```

**FORBIDDEN to log:**
- Tokens, passwords, secrets, sensitive payloads, `DATABASE_URL`

### Request ID
- Accept `X-Request-Id`, generate if missing
- Return `X-Request-Id` in response

### Health Checks
```
/health  - liveness (no DB check)
/ready   - readiness (validates DB + critical dependencies)
```

---

## 🛡️ Application Security

### CORS
- ✅ Explicit per environment
- ❌ FORBIDDEN: `*` in production

### Security Headers (MANDATORY)
```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY  # or SAMEORIGIN if needed
```

### Payload Limits
- ✅ Configure max body size

### Input Sanitization
- ✅ Sanitize inputs used in logs/SQL/HTML

---

## 🏗️ Go Code Structure

### Organization
```
cmd/              # entrypoints
internal/         # application code
  ├── http/handlers/  # thin layer
  ├── domain/         # business rules
  ├── repository/     # DB access
  └── middleware/     # auth, tenant, rate limit
pkg/              # reusable libraries only`

### Dependency Injection

- ✅ Explicit injection (struct constructors)
- ❌ FORBIDDEN: global variables for DB, logger, config

### Context

- ✅ MANDATORY `context.Context` in all methods performing I/O

---

## 📖 OpenAPI Documentation (MANDATORY)

- ✅ `openapi.yaml` is source of truth
- ❌ FORBIDDEN: undocumented endpoints

**Every route must have:**

- Summary/description
- Auth requirements
- Request/response schemas
- Examples (minimum 1)

---
## API-First Rule (MANDATORY)

- OpenAPI MUST be written or updated before any handler implementation.
- Handlers without OpenAPI coverage are forbidden.

---

## 🐳 Docker / Deploy Guidelines

Learned from Prisma issues:

- ❌ FORBIDDEN: mutate config at runtime via `sed`/`awk` in Docker build
- ✅ MANDATORY: deterministic configs (`tsconfig.build.json`, explicit env)
- ✅ `/health` must return `200` without auth dependency
- ✅ Bind to `0.0.0.0` inside container (not `localhost`)

---

## ✅ Pre-PR Checklist (MANDATORY)

- [ ]  OpenAPI updated
- [ ]  Unit tests (minimum: service layer)
- [ ]  Lint/format passing
- [ ]  `/health` and `/ready` working
- [ ]  Structured logs with `requestId`
- [ ]  Tenant guard applied
- [ ]  No secrets in logs/config
- [ ]  DB migrations/rules reviewed

---

## 📝 Code Documentation Standards

### Core Principle

**Comments explain WHY, not WHAT.**

- Code explains what it does
- Documentation explains why it exists this way

---

### ✅ When to Document (MANDATORY)

### 1. Architectural Decisions

When code represents non-obvious decisions:

go

`// This endpoint intentionally does not expose direct CRUD access.
// Reason:
// - Deals can only transition via controlled stage changes
// - Prevents inconsistent funnel state across integrations`

**Document:**

- Why this route doesn't follow standard CRUD
- Why cursor instead of offset
- Why async flow
- Why seemingly redundant field exists
- Why service communicates with two domains

### 2. Business Rules

Rules that cannot be inferred from code:

go

`// A deal cannot be moved to WON if there is no associated company.
// This rule exists to guarantee CRM revenue attribution consistency.`

### 3. External Integrations

Always document:

- The system
- Expected contract
- Risks involved

go

`// Integration with WhatsApp Cloud API
// Important:
// - Meta may resend the same webhook multiple times
// - Idempotency is enforced via messageId`

### 4. Critical Guardrails

Where breaking this causes severe bugs:

go

`// DO NOT remove workspaceId filter.
// Multi-tenant isolation depends on this constraint.
// Removing it will cause data leakage across tenants.`

### 5. Counter-Intuitive Code

If it looks wrong but is correct:

go

`// This looks redundant, but is required because
// Supabase RLS does not trigger on nested updates.`

---

### ❌ When NOT to Document (FORBIDDEN)

**Never write obvious comments:**

go

`// ❌ BAD
// increment i by 1
i++

// ❌ BAD
// get user by id
user := repo.GetUser(id)`

This is noise. Never generate these.

---

### 📐 Comment Style (MANDATORY)

### Always in English

Code is a strategic global asset.

### Complete Comments, Not Fragments

go

`// ❌ BAD
// important

// ✅ GOOD
// Important:
// This validation must happen before persistence
// because downstream services assume normalized data.`

### Technical, Neutral, Direct Tone

- ❌ "this is weird but works"
- ✅ "This approach is intentional to guarantee deterministic behavior."

---

### 🧭 Documentation Levels

### Level 1 — File

At top of important files:

go

`// Package deals contains all domain logic related to CRM opportunities.
// This layer is responsible for enforcing funnel integrity,
// lifecycle transitions and revenue consistency rules.`

### Level 2 — Function

Only if there's a rule, context, or impact:

go

`// CloseDeal performs a controlled deal closure.
// It validates business rules before persisting state
// and emits domain events for automation workflows.`

### Level 3 — Critical Block

Inside function, only when necessary:

go

`// Guardrail:
// This must run inside the transaction.
// Partial updates would corrupt pipeline analytics.`

---
## Logging Standard

All services MUST follow the Linkko Structured Logging Standard (v1.0).

- Unstructured logs are forbidden.
- fmt.Println / log.Printf are forbidden.
- Logs must include request_id and workspace_id when applicable.

Refer to:
docs/logging-standard.md

---

### 🧠 Golden Rule

**If a new developer joining in 1 year needs to understand WHY it exists (not just HOW it works), document it.**

---

### 🤖 Special Rules for AI Assistants

You MUST:

- ✅ Document decisions, not actions
- ✅ Document impact, not syntax
- ✅ Document future constraints
- ❌ Never comment obvious lines
- ❌ Never comment "for politeness"
- ❌ Never comment "what the code does"

---

### 🧱 Final Guardrail (ABSOLUTE)

**If the comment can be removed without losing future understanding, it shouldn't exist.**