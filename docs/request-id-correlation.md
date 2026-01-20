# Request ID & Correlation Propagation (v1.0)

**Effective Date:** January 20, 2026  
**Scope:** All Go services in Linkko platform  
**Status:** Mandatory

---

## Overview

Request ID correlation enables end-to-end tracing of requests across distributed services. Every request must carry a unique `request_id` that propagates through:

- Inbound HTTP requests
- Outbound HTTP requests to internal services (MCP server, adapters)
- Background goroutines spawned within request context
- External API calls
- Log entries

**Core Principle:**  
A single request_id flows from API gateway → Go CRM API → MCP Server → Adapters → External APIs, enabling full traceability.

---

## Standard Header

**Header Name:** `X-Request-Id`

**Format:** ULID-inspired  
`req_<timestamp_millis>_<random_hex>`

**Example:** `req_1737373445123_a1b2c3d4e5f6`

---

## Propagation Rules

### Rule 1: Inbound Requests (API receives)

1. **Read `X-Request-Id`** from incoming request header
2. **If present:** Use it (preserve client-provided ID)
3. **If missing:** Generate new ULID
4. **Inject into context:** Store in `context.Context` for downstream use
5. **Echo in response:** Return `X-Request-Id` header to caller

**Implementation:** `RequestIDMiddleware` in `internal/http/middleware/observability.go`

### Rule 2: Outbound Requests (API calls other services)

1. **Extract `request_id`** from `context.Context`
2. **Set `X-Request-Id` header** on outbound HTTP request
3. **Preserve explicit headers:** If caller already set `X-Request-Id`, don't overwrite

**Implementation:** `RequestIDTransport` in `internal/http/client/request_id_transport.go`

### Rule 3: Log Entries

1. **All logs** produced during a request **MUST** include `request_id` field
2. **Automatic extraction:** Logger wrapper reads `request_id` from context

**Implementation:** `Logger.Info/Warn/Error` in `internal/observability/logger/logger.go`

### Rule 4: Background Goroutines

1. **Pass context** when spawning goroutines: `go processInBackground(ctx, ...)`
2. **Do NOT** use `context.Background()` inside request handlers
3. **Timeout contexts** should derive from request context: `ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)`

---

## Request Flow Example

### Scenario: Create Contact → Notify via MCP → Send Email via Adapter

```
Client                  Go CRM API              MCP Server           Email Adapter
  |                         |                       |                     |
  |--- POST /contacts ----->|                       |                     |
  |    (no X-Request-Id)    |                       |                     |
  |                         |                       |                     |
  |                    [Generate req_123]           |                     |
  |                         |                       |                     |
  |<-- 201 Created ---------|                       |                     |
  |    X-Request-Id: req_123|                       |                     |
  |                         |                       |                     |
  |                         |--- POST /notify ----->|                     |
  |                         |   X-Request-Id: req_123                     |
  |                         |                       |                     |
  |                         |                  [Log: req_123]             |
  |                         |                       |                     |
  |                         |                       |--- POST /send ----->|
  |                         |                       | X-Request-Id: req_123
  |                         |                       |                     |
  |                         |                       |              [Log: req_123]
  |                         |                       |                     |
  |                         |                       |<--- 200 OK ---------|
  |                         |                       |                     |
  |                         |<--- 200 OK -----------|                     |
  |                         |                       |                     |
```

### Logs Produced

**Go CRM API:**
```json
{
  "timestamp": "2026-01-20T15:04:05.123Z",
  "level": "info",
  "service": "linkko-crm-api",
  "module": "contacts",
  "action": "create_contact",
  "message": "contact created successfully",
  "request_id": "req_123",
  "workspace_id": "550e8400",
  "user_id": "7c9e6679",
  "contact_id": "9b1deb4d"
}
```

**MCP Server:**
```json
{
  "timestamp": "2026-01-20T15:04:05.234Z",
  "level": "info",
  "service": "linkko-mcp-server",
  "module": "notifications",
  "action": "send_notification",
  "message": "notification queued",
  "request_id": "req_123",
  "notification_type": "contact_created"
}
```

**Email Adapter:**
```json
{
  "timestamp": "2026-01-20T15:04:05.345Z",
  "level": "info",
  "service": "linkko-email-adapter",
  "module": "smtp",
  "action": "send_email",
  "message": "email sent successfully",
  "request_id": "req_123",
  "recipient": "user@example.com"
}
```

---

## Debugging Workflow

### Problem: "Contact creation failed but I don't know where"

#### Step 1: Find the request_id

From client response:
```bash
curl -i https://api.linkko.app/v1/workspaces/{id}/contacts
HTTP/1.1 500 Internal Server Error
X-Request-Id: req_1737373445123_a1b2c3d4e5f6
```

#### Step 2: Grep logs across all services

```bash
# Search in Go CRM API logs
kubectl logs -l app=linkko-crm-api | jq 'select(.request_id == "req_1737373445123_a1b2c3d4e5f6")'

# Search in MCP Server logs
kubectl logs -l app=linkko-mcp-server | jq 'select(.request_id == "req_1737373445123_a1b2c3d4e5f6")'

# Search in Email Adapter logs
kubectl logs -l app=linkko-email-adapter | jq 'select(.request_id == "req_1737373445123_a1b2c3d4e5f6")'
```

#### Step 3: Reconstruct timeline

All logs with same `request_id` form a complete trace:
```
15:04:05.123 [CRM API] contact created successfully
15:04:05.234 [MCP] notification queued
15:04:05.345 [Adapter] email sent successfully
15:04:05.456 [Adapter] ERROR: SMTP timeout
```

**Root cause identified:** Email adapter timed out.

---

## Code Examples

### Example 1: Handler Making Outbound Call

```go
func (h *ContactHandler) CreateContact(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context() // Contains request_id from middleware
    log := logger.GetLogger(ctx)

    // Business logic
    contact, err := h.service.Create(ctx, req)
    if err != nil {
        log.Error(ctx, "failed to create contact",
            logger.Module("contacts"),
            logger.Action("create_contact"),
            zap.Error(err),
        )
        // Log automatically includes request_id from ctx
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }

    // Notify MCP Server (automatically propagates request_id)
    if err := h.mcpClient.NotifyContactCreated(ctx, contact.ID); err != nil {
        log.Warn(ctx, "failed to notify mcp",
            logger.Module("contacts"),
            logger.Action("notify_mcp"),
            zap.Error(err),
        )
        // Non-fatal: continue
    }

    writeJSON(w, http.StatusCreated, contact)
}
```

### Example 2: MCP Client with Request ID Propagation

```go
package mcp

import (
    "context"
    "fmt"
    "net/http"

    "linkko-api/internal/http/client"
)

type MCPClient struct {
    httpClient *http.Client
    baseURL    string
}

func NewMCPClient(baseURL string) *MCPClient {
    return &MCPClient{
        httpClient: client.NewInternalHTTPClient(), // Includes RequestIDTransport
        baseURL:    baseURL,
    }
}

func (c *MCPClient) NotifyContactCreated(ctx context.Context, contactID string) error {
    url := fmt.Sprintf("%s/v1/notifications/contact-created", c.baseURL)
    
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }

    // X-Request-Id automatically added by RequestIDTransport from ctx
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("unexpected status: %d", resp.StatusCode)
    }

    return nil
}
```

### Example 3: Background Goroutine with Context

```go
func (h *ContactHandler) CreateContactAsync(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    log := logger.GetLogger(ctx)

    // Spawn background goroutine with context (preserves request_id)
    go func(ctx context.Context) {
        // Add timeout to prevent infinite wait
        ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
        defer cancel()

        if err := h.processContactEnrichment(ctx); err != nil {
            log.Error(ctx, "enrichment failed",
                logger.Module("contacts"),
                logger.Action("enrich_contact"),
                zap.Error(err),
            )
            // Log includes request_id from original request
        }
    }(ctx)

    w.WriteHeader(http.StatusAccepted)
}
```

---

## Anti-Patterns (DO NOT DO)

### ❌ WRONG: Not passing context

```go
func (c *MCPClient) NotifyContactCreated(contactID string) error {
    // WRONG: No context = no request_id propagation
    req, _ := http.NewRequest(http.MethodPost, c.baseURL, nil)
    resp, _ := http.DefaultClient.Do(req)
    // ...
}
```

**Why wrong:** Request_id is lost. Logs in MCP server can't correlate back to CRM API request.

### ❌ WRONG: Using context.Background() in handlers

```go
func (h *ContactHandler) CreateContact(w http.ResponseWriter, r *http.Request) {
    ctx := context.Background() // WRONG: Discards request context
    
    // request_id is lost
    contact, err := h.service.Create(ctx, req)
}
```

**Why wrong:** Breaks correlation chain. All downstream logs lose request_id.

### ❌ WRONG: Using http.DefaultClient

```go
func callExternalAPI() error {
    req, _ := http.NewRequest(http.MethodGet, "https://api.example.com", nil)
    resp, _ := http.DefaultClient.Do(req) // WRONG: No request_id propagation
    // ...
}
```

**Why wrong:** No timeout (can hang forever) and no request_id propagation.

---

## Implementation Checklist

### For New Services

- [ ] Use `RequestIDMiddleware` as first middleware in HTTP stack
- [ ] Create HTTP clients with `client.NewInternalHTTPClient()`
- [ ] Pass `context.Context` to all functions that do I/O
- [ ] Use `logger.GetLogger(ctx)` to ensure request_id in logs
- [ ] Test request_id propagation with integration tests

### For Existing Services

- [ ] Audit all `http.NewRequest` calls → add context
- [ ] Replace `http.DefaultClient` with `client.NewInternalHTTPClient()`
- [ ] Verify background goroutines receive context: `go myFunc(ctx, ...)`
- [ ] Add tests for request_id propagation

---

## Testing

### Unit Test: Request ID Propagation

```go
func TestRequestIDPropagation(t *testing.T) {
    const testRequestID = "test-req-123"

    // Mock downstream service
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        gotID := r.Header.Get("X-Request-Id")
        if gotID != testRequestID {
            t.Errorf("expected X-Request-Id %q, got %q", testRequestID, gotID)
        }
        w.WriteHeader(http.StatusOK)
    }))
    defer ts.Close()

    // Create client with RequestIDTransport
    client := client.NewInternalHTTPClient()

    // Create request with context containing request_id
    ctx := context.Background()
    ctx = requestid.SetRequestID(ctx, testRequestID)

    req, _ := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
    resp, _ := client.Do(req)
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        t.Errorf("expected status 200, got %d", resp.StatusCode)
    }
}
```

### Integration Test: End-to-End Correlation

```bash
#!/bin/bash
# Test end-to-end request_id correlation

REQUEST_ID="test-$(date +%s)"

# Make request with explicit request_id
curl -H "X-Request-Id: $REQUEST_ID" \
     http://localhost:8080/v1/workspaces/{id}/contacts \
     -X POST -d '{"name":"Test","email":"test@example.com"}'

# Wait for async processing
sleep 2

# Verify request_id appears in all service logs
kubectl logs -l app=linkko-crm-api --since=5m | grep "$REQUEST_ID" || exit 1
kubectl logs -l app=linkko-mcp-server --since=5m | grep "$REQUEST_ID" || exit 1

echo "✅ Request ID correlation verified: $REQUEST_ID"
```

---

## Enforcement

### Pre-commit Hooks

- Lint: Detect `http.DefaultClient.Do(` usage
- Lint: Detect `http.NewRequest(` without context (should use `NewRequestWithContext`)

### Code Review Checklist

- [ ] All HTTP clients created via `client.New*HTTPClient()`
- [ ] All outbound calls pass `context.Context`
- [ ] No `context.Background()` in request handlers
- [ ] Background goroutines receive context as parameter

### CI/CD

- Integration tests verify request_id propagation
- E2E tests grep logs for request_id correlation

---

## References

- [Structured Logging Standard](./logging-standard.md)
- [Google Dapper Paper - Distributed Tracing](https://research.google/pubs/pub36356/)
- [OpenTelemetry Trace Context](https://www.w3.org/TR/trace-context/)
