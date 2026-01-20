# Linkko Structured Logging Standard (v1.0)

**Effective Date:** January 20, 2026  
**Scope:** All Go services in Linkko platform  
**Status:** Mandatory

---

## Overview

All services MUST emit structured JSON logs that are machine-readable, privacy-safe, and include mandatory contextual fields for observability.

**Core Principle:**  
Logs explain **WHY** something happened, not just **WHAT** happened. Code explains "what"; logs explain "why this decision was made" or "why this path was taken".

---

## Mandatory Top-Level Fields

Every log line MUST include:

| Field       | Type   | Description                                      | Example                  |
|-------------|--------|--------------------------------------------------|--------------------------|
| `timestamp` | string | RFC3339Nano format                               | `2026-01-20T15:04:05.999999999Z` |
| `level`     | string | `debug`, `info`, `warn`, `error`                 | `info`                   |
| `service`   | string | Service name (e.g., `linkko-crm-api`)            | `linkko-crm-api`         |
| `module`    | string | Domain/component (e.g., `contacts`, `auth`)      | `contacts`               |
| `action`    | string | Operation/action (e.g., `create_contact`)        | `create_contact`         |
| `message`   | string | Human-readable narrative explaining WHY          | `contact created successfully` |

---

## Contextual Mandatory Fields

When applicable, include:

| Field          | Type   | When to Include                          | Example                      |
|----------------|--------|------------------------------------------|------------------------------|
| `request_id`   | string | All request-scoped operations            | `req_1737373445999_a1b2c3d4` |
| `workspace_id` | string | All tenant-scoped operations             | `550e8400-e29b-41d4-a716-446655440000` |
| `user_id`      | string | When authenticated user exists           | `7c9e6679-7425-40de-944b-e07fc1f90ae7` |
| `route`        | string | HTTP request logs                        | `/v1/workspaces/{id}/contacts` |
| `method`       | string | HTTP request logs                        | `POST`                       |
| `status`       | int    | HTTP request logs                        | `201`                        |
| `latency_ms`   | float  | HTTP request logs                        | `45.23`                      |

---

## Log Levels

Use levels appropriately:

- **`debug`**: Detailed diagnostics for development (not in production by default)
- **`info`**: Normal operational events (service started, request completed)
- **`warn`**: Unexpected but recoverable situations (rate limit hit, deprecated API used)
- **`error`**: Errors requiring attention (database connection failed, external API error)

---

## Example Logs

### HTTP Request Completed (Success)

```json
{
  "timestamp": "2026-01-20T15:04:05.123456789Z",
  "level": "info",
  "service": "linkko-crm-api",
  "module": "http",
  "action": "request",
  "message": "http request completed",
  "request_id": "req_1737373445123_a1b2c3d4",
  "workspace_id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "method": "POST",
  "route": "/v1/workspaces/550e8400-e29b-41d4-a716-446655440000/contacts",
  "status": 201,
  "latency_ms": 45.23,
  "remote_addr": "203.0.113.42",
  "user_agent": "Mozilla/5.0..."
}
```

### Business Logic Event

```json
{
  "timestamp": "2026-01-20T15:04:05.234567890Z",
  "level": "info",
  "service": "linkko-crm-api",
  "module": "contacts",
  "action": "create_contact",
  "message": "contact created successfully",
  "request_id": "req_1737373445123_a1b2c3d4",
  "workspace_id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "contact_id": "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d"
}
```

### Error Event

```json
{
  "timestamp": "2026-01-20T15:04:06.345678901Z",
  "level": "error",
  "service": "linkko-crm-api",
  "module": "postgres",
  "action": "query_contacts",
  "message": "database query failed",
  "request_id": "req_1737373446345_e5f6g7h8",
  "workspace_id": "550e8400-e29b-41d4-a716-446655440000",
  "error": "pq: connection refused",
  "query": "SELECT * FROM contacts WHERE workspace_id = $1"
}
```

### Panic Recovery

```json
{
  "timestamp": "2026-01-20T15:04:07.456789012Z",
  "level": "error",
  "service": "linkko-crm-api",
  "module": "http",
  "action": "panic_recovery",
  "message": "panic recovered",
  "request_id": "req_1737373447456_i9j0k1l2",
  "panic": "runtime error: invalid memory address or nil pointer dereference",
  "stack": "goroutine 42 [running]:\n...",
  "method": "PATCH",
  "route": "/v1/workspaces/550e8400/contacts/123"
}
```

---

## Security & Privacy: DO NOT LOG

### FORBIDDEN: Secrets and Credentials

**NEVER** log these fields:

- `authorization` (Bearer tokens, API keys)
- `password`, `secret`, `api_key`, `token`, `jwt`
- `database_url`, `DATABASE_URL`
- Full `Authorization` header
- Cookie values containing session tokens

### FORBIDDEN: Personally Identifiable Information (PII)

**NEVER** log these fields directly:

- `email`, `phone`, `full_name`, `first_name`, `last_name`
- `address`, `credit_card`, `ssn`, `tax_id`
- IP addresses (log sanitized version without port)

**Allowed:**  
- Hashed IDs (e.g., `user_id`, `contact_id`)
- Obfuscated data for debugging (e.g., `em***@example.com`)

### FORBIDDEN: Request/Response Bodies

- Do NOT log full payloads
- Do NOT log full headers
- Do NOT log query strings with sensitive parameters

**Allowed:**  
- Sanitized query strings (truncated, filtered)
- Content-Length, Content-Type headers

---

## Request ID Propagation Contract

### Generation

- **Format:** `req_<timestamp_millis>_<random_hex>`
- **Example:** `req_1737373445123_a1b2c3d4e5f6`
- **Library:** ULID-inspired (better time-ordering than UUID)

### Propagation

1. **Incoming:** Read `X-Request-Id` header from client
2. **Generate:** If missing, generate new ULID
3. **Context:** Inject into `context.Context`
4. **Outgoing:** Write `X-Request-Id` header to response
5. **Downstream:** Include in all outgoing HTTP requests to other services

### Usage

```go
import (
    "linkko-api/internal/observability/logger"
    "linkko-api/internal/observability/requestid"
)

// Middleware injects request ID
ctx := requestid.SetRequestID(r.Context(), reqID)

// Logger automatically includes request_id from context
log.Info(ctx, "processing request", logger.Module("contacts"), logger.Action("create"))
```

---

## Implementation Rules

### ✅ MUST

1. Use `internal/observability/logger.Logger` for all logging
2. Include `module` and `action` on every log
3. Use context-aware logging: `log.Info(ctx, msg, fields...)`
4. Log at appropriate level (info for normal, error for failures)
5. Include `request_id` for all request-scoped logs

### ❌ FORBIDDEN

1. `fmt.Println`, `log.Printf`, `log.Print` (unstructured logs)
2. Logging secrets, PII, or sensitive headers
3. Logging full request/response bodies
4. Creating logs without `module` and `action`
5. Hardcoding values that should come from context (`request_id`, `workspace_id`)

---

## Code Example

### Handler with Structured Logging

```go
func (h *ContactHandler) CreateContact(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    log := logger.GetLogger(ctx) // Extracts logger from context
    
    // Log operation start
    log.Info(ctx, "creating contact",
        logger.Module("contacts"),
        logger.Action("create_contact"),
    )
    
    // Business logic...
    contact, err := h.service.Create(ctx, req)
    if err != nil {
        log.Error(ctx, "failed to create contact",
            logger.Module("contacts"),
            logger.Action("create_contact"),
            zap.Error(err),
        )
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }
    
    // Log success
    log.Info(ctx, "contact created successfully",
        logger.Module("contacts"),
        logger.Action("create_contact"),
        zap.String("contact_id", contact.ID),
    )
    
    writeJSON(w, http.StatusCreated, contact)
}
```

---

## Testing

### Verify Logs are JSON

```bash
# Run service and check logs
go run ./cmd/linkko-api serve | jq .

# All logs should parse as valid JSON
# All logs should include: timestamp, level, service, module, action, message
```

### Verify Request ID Propagation

```bash
# Send request with X-Request-Id
curl -H "X-Request-Id: test-123" http://localhost:8080/health -v

# Response should echo: X-Request-Id: test-123
```

---

## Enforcement

- **Pre-commit:** Lint rules to detect `fmt.Println`, `log.Print`
- **Code Review:** Verify all logs include `module` and `action`
- **CI:** Automated tests verify JSON structure and mandatory fields

---

## References

- [Uber Zap Documentation](https://github.com/uber-go/zap)
- [ULID Specification](https://github.com/ulid/spec)
- [RFC3339 Timestamp Format](https://www.rfc-editor.org/rfc/rfc3339)
