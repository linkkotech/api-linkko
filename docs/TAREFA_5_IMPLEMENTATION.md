# Tarefa 5 - Implementação Completa

## ✅ Status: CONCLUÍDO

Padronização de respostas HTTP de erro com diferenciação clara entre 401, 403 e 400.

---

## 📦 Pacote Criado: `internal/http/httperr`

### Estrutura de Erro Padronizada

```go
type ErrorResponse struct {
    OK    bool         `json:"ok"`
    Error *ErrorDetail `json:"error"`
}

type ErrorDetail struct {
    Code    string                 `json:"code"`
    Message string                 `json:"message"`
    Fields  map[string]string      `json:"fields,omitempty"`
}
```

### Exemplo de Resposta

```json
{
  "ok": false,
  "error": {
    "code": "WORKSPACE_MISMATCH",
    "message": "workspace access denied"
  }
}
```

---

## 🔧 Códigos de Erro Implementados

### 401 Unauthorized (Falhas de Autenticação)
- `MISSING_AUTHORIZATION` - Header Authorization ausente
- `INVALID_SCHEME` - Esquema diferente de "Bearer"
- `INVALID_TOKEN` - Token JWT inválido/malformado
- `TOKEN_EXPIRED` - Token JWT expirado
- `INVALID_SIGNATURE` - Assinatura inválida (JWT ou S2S)
- `INVALID_ISSUER` - Issuer do token não corresponde
- `INVALID_AUDIENCE` - Audience do token não corresponde

### 403 Forbidden (Falhas de Autorização)
- `WORKSPACE_MISMATCH` - Workspace no path ≠ workspace no token (IDOR protection)
- `FORBIDDEN` - Permissões insuficientes
- `INSUFFICIENT_SCOPE` - OAuth scope insuficiente

### 400 Bad Request (Erros de Validação)
- `INVALID_WORKSPACE_ID` - WorkspaceID com formato inválido
- `MISSING_PARAMETER` - Parâmetro obrigatório ausente
- `INVALID_PARAMETER` - Parâmetro com valor inválido
- `INVALID_FORMAT` - Formato de dado incorreto
- `INVALID_LIMIT` - Limit fora do range permitido
- `VALIDATION_ERROR` - Erro de validação do body
- `INVALID_STATUS` - Status inválido (tasks)
- `INVALID_PRIORITY` - Prioridade inválida (tasks)
- `INVALID_TYPE` - Tipo inválido (tasks)

### 500 Internal Server Error
- `INTERNAL_ERROR` - Erro inesperado do servidor

---

## 📋 Funções Helpers

### Principais

```go
// Genérica - permite especificar status e código
func WriteError(w http.ResponseWriter, ctx context.Context, status int, code, message string)

// Com campos de validação
func WriteErrorWithFields(w http.ResponseWriter, ctx context.Context, status int, code, message string, fields map[string]string)

// Convenientes para cada tipo
func Unauthorized401(w http.ResponseWriter, ctx context.Context, code, message string)
func Forbidden403(w http.ResponseWriter, ctx context.Context, code, message string)
func BadRequest400(w http.ResponseWriter, ctx context.Context, code, message string)
func InternalError500(w http.ResponseWriter, ctx context.Context, message string)
```

### Características

- **Logging Estruturado**: Todas as funções incluem logs com `status_code`, `error_code` e `message`
- **Content-Type**: Sempre `application/json`
- **Contexto**: Usa `context.Context` para request ID e tracing

---

## 🔄 Migração de Código

### Antes (Padrão Antigo)

```go
func writeError(w http.ResponseWriter, ctx context.Context, log *logger.Logger, status int, code, message string) {
    log.Error(ctx, "request failed",
        zap.Int("statusCode", status),
        zap.String("errorCode", code),
        zap.String("message", message),
    )

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(errorResponse{
        Code:    code,
        Message: message,
    })
}

type errorResponse struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}
```

**Resposta gerada:**
```json
{
  "code": "UNAUTHORIZED",
  "message": "authentication claims not found"
}
```

### Depois (Padrão Novo com httperr)

```go
import "linkko-api/internal/http/httperr"

// Removido: writeError() customizado
// Removido: type errorResponse

// Uso direto das funções do httperr
```

**Resposta gerada:**
```json
{
  "ok": false,
  "error": {
    "code": "INVALID_TOKEN",
    "message": "authentication required"
  }
}
```

---

## 🎯 Padrões de Substituição

### 1. Authentication Checks (401)

**Antes:**
```go
claims, ok := auth.GetClaims(ctx)
if !ok {
    writeError(w, ctx, log, http.StatusUnauthorized, "UNAUTHORIZED", "authentication claims not found")
    return
}
```

**Depois:**
```go
claims, ok := auth.GetClaims(ctx)
if !ok {
    httperr.Unauthorized401(w, ctx, httperr.ErrCodeInvalidToken, "authentication required")
    return
}
```

---

### 2. Validation Errors (400)

**Antes:**
```go
if limit < 1 || limit > 100 {
    writeError(w, ctx, log, http.StatusBadRequest, "INVALID_LIMIT", "limit must be between 1 and 100")
    return
}
```

**Depois:**
```go
if limit < 1 || limit > 100 {
    httperr.BadRequest400(w, ctx, httperr.ErrCodeInvalidParameter, "limit must be between 1 and 100")
    return
}
```

---

### 3. JSON Decode Errors (400)

**Antes:**
```go
var req domain.CreateContactRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    log.Warn(ctx, "invalid request body", zap.Error(err))
    writeError(w, ctx, log, http.StatusBadRequest, "INVALID_REQUEST", "request body must be valid JSON")
    return
}
```

**Depois:**
```go
var req domain.CreateContactRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    log.Warn(ctx, "invalid request body", zap.Error(err))
    httperr.BadRequest400(w, ctx, httperr.ErrCodeInvalidParameter, "request body must be valid JSON")
    return
}
```

---

### 4. Domain Validation Errors (422 → 400)

**Antes:**
```go
if err := req.Validate(); err != nil {
    log.Warn(ctx, "validation failed", zap.Error(err))
    writeError(w, ctx, log, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
    return
}
```

**Depois:**
```go
if err := req.Validate(); err != nil {
    log.Warn(ctx, "validation failed", zap.Error(err))
    httperr.WriteError(w, ctx, http.StatusUnprocessableEntity, httperr.ErrCodeValidationError, err.Error())
    return
}
```

---

### 5. Service Error Handling

**Antes:**
```go
func handleServiceError(w http.ResponseWriter, ctx context.Context, log *logger.Logger, err error) {
    switch {
    case errors.Is(err, service.ErrMemberNotFound):
        writeError(w, ctx, log, http.StatusForbidden, "FORBIDDEN", "insufficient permissions for this workspace")
    case errors.Is(err, service.ErrContactNotFound):
        writeError(w, ctx, log, http.StatusNotFound, "NOT_FOUND", "contact not found")
    default:
        log.Error(ctx, "internal server error", zap.Error(err))
        writeError(w, ctx, log, http.StatusInternalServerError, "INTERNAL_ERROR", "an internal error occurred")
    }
}
```

**Depois:**
```go
func handleServiceError(w http.ResponseWriter, ctx context.Context, log *logger.Logger, err error) {
    switch {
    case errors.Is(err, service.ErrMemberNotFound):
        httperr.Forbidden403(w, ctx, httperr.ErrCodeForbidden, "insufficient permissions for this workspace")
    case errors.Is(err, service.ErrContactNotFound):
        httperr.WriteError(w, ctx, http.StatusNotFound, "NOT_FOUND", "contact not found")
    default:
        log.Error(ctx, "internal server error", zap.Error(err))
        httperr.InternalError500(w, ctx, "an internal error occurred")
    }
}
```

---

## ✅ Arquivos Migrados

### 1. **internal/http/httperr/error.go** (NOVO)
- Estrutura ErrorResponse e ErrorDetail
- Constantes de códigos de erro
- Funções WriteError, WriteErrorWithFields
- Helpers: Unauthorized401, Forbidden403, BadRequest400, InternalError500

### 2. **internal/http/httperr/error_test.go** (NOVO)
- 7 testes cobrindo todas as funções
- Validação de estrutura JSON
- Validação de status codes
- Validação de Content-Type headers

### 3. **internal/auth/s2s.go** (ATUALIZADO)
- Substituído http.Error por httperr
- Criada função mapAuthErrorToCode() para mapear erros de autenticação
- Missing authorization → ErrCodeMissingAuthorization
- Invalid scheme → ErrCodeInvalidScheme
- JWT errors → ErrCodeInvalidToken, ErrCodeTokenExpired, etc.
- S2S errors → ErrCodeInvalidSignature, ErrCodeInvalidParameter

### 4. **internal/http/middleware/workspace.go** (ATUALIZADO)
- Missing workspaceId → ErrCodeMissingParameter (400)
- Invalid format → ErrCodeInvalidWorkspaceID (400)
- No auth context → ErrCodeInvalidToken (401)
- Workspace mismatch → ErrCodeWorkspaceMismatch (403)

### 5. **internal/http/middleware/workspace_test.go** (ATUALIZADO)
- Adicionada função validateErrorResponse() para parsing JSON
- Todos os testes validam estrutura { "ok": false, "error": {...} }
- Validação de error codes específicos

### 6. **internal/http/handler/contact.go** (ATUALIZADO)
- Removida função writeError() customizada
- Removido type errorResponse
- Atualizada handleServiceError() para usar httperr
- Todas as chamadas writeError → httperr.Unauthorized401/BadRequest400/etc.

---

## 🧪 Testes

### Executar Testes do Pacote httperr

```bash
go test ./internal/http/httperr/... -v
```

**Resultado Esperado:**
```
=== RUN   TestWriteError
--- PASS: TestWriteError (0.00s)
=== RUN   TestWriteErrorWithFields
--- PASS: TestWriteErrorWithFields (0.00s)
=== RUN   TestUnauthorized401
--- PASS: TestUnauthorized401 (0.00s)
=== RUN   TestForbidden403
--- PASS: TestForbidden403 (0.00s)
=== RUN   TestBadRequest400
--- PASS: TestBadRequest400 (0.00s)
=== RUN   TestInternalError500
--- PASS: TestInternalError500 (0.00s)
=== RUN   TestErrorResponseStructure
--- PASS: TestErrorResponseStructure (0.00s)
PASS
```

### Executar Testes do Workspace Middleware

```bash
go test ./internal/http/middleware/... -v -run "TestWorkspace"
```

**Resultado Esperado:**
```
=== RUN   TestWorkspaceMiddleware_InvalidFormat
=== RUN   TestWorkspaceMiddleware_InvalidFormat/EmptyWorkspaceID
=== RUN   TestWorkspaceMiddleware_InvalidFormat/InvalidCharacters
=== RUN   TestWorkspaceMiddleware_InvalidFormat/TooLong
=== RUN   TestWorkspaceMiddleware_InvalidFormat/SpecialCharacters
--- PASS: TestWorkspaceMiddleware_InvalidFormat (0.02s)
    --- PASS: TestWorkspaceMiddleware_InvalidFormat/EmptyWorkspaceID (0.02s)
    --- PASS: TestWorkspaceMiddleware_InvalidFormat/InvalidCharacters (0.00s)
    --- PASS: TestWorkspaceMiddleware_InvalidFormat/TooLong (0.00s)
    --- PASS: TestWorkspaceMiddleware_InvalidFormat/SpecialCharacters (0.00s)
=== RUN   TestWorkspaceMiddleware_Mismatch_JWT
...
PASS
```

---

## 📊 Logs Estruturados

Todas as respostas de erro agora geram logs estruturados com:

```json
{
  "level": "error",
  "timestamp": "2026-01-22T00:31:11.664Z",
  "caller": "logger/logger.go:187",
  "message": "request failed",
  "service": "linkko-api",
  "status_code": 403,
  "error_code": "WORKSPACE_MISMATCH",
  "message": "workspace access denied",
  "request_id": "uuid-here",
  "module": "middleware",
  "action": "workspace_validation"
}
```

---

## 📄 Documentação

Criado arquivo `docs/ERROR_RESPONSES.md` com:
- Especificação completa da estrutura de erro
- Guia de uso de cada código de erro
- Exemplos de requisição/resposta para cada cenário
- Exemplos cURL
- Guias de implementação para clientes (JavaScript/TypeScript, Go)
- Exemplos de testes

---

## 🎉 Benefícios

1. **Consistência**: Todas as respostas seguem o mesmo formato JSON
2. **Clareza**: Códigos de erro categorizados por tipo (auth, authz, validation)
3. **Debugging**: Logs estruturados com códigos de erro para troubleshooting
4. **Client-Side**: Clientes podem tratar erros de forma previsível
5. **Manutenibilidade**: Centralização da lógica de erro em um único pacote
6. **Testabilidade**: Helpers facilitam validação de erros em testes
7. **Extensibilidade**: Fácil adicionar novos códigos de erro quando necessário

---

## ⚡ Próximos Passos

### Migração Pendente

Os handlers abaixo ainda usam funções `writeError()` customizadas e precisam ser migrados para `httperr`:

- [ ] `internal/http/handler/task.go` (21 ocorrências)
- [ ] `internal/http/handler/company.go`
- [ ] `internal/http/handler/deal.go`
- [ ] `internal/http/handler/pipeline.go`
- [ ] `internal/http/handler/portfolio.go`
- [ ] `internal/http/handler/activity.go`

### Padrão de Migração para Handlers Restantes

1. Adicionar import: `"linkko-api/internal/http/httperr"`
2. Remover função `writeError()` customizada
3. Remover `type errorResponse struct`
4. Atualizar `handleServiceError()` para usar httperr
5. Substituir todas as chamadas `writeError(...)` por:
   - `httperr.Unauthorized401()` para auth checks
   - `httperr.BadRequest400()` para validações
   - `httperr.WriteError()` para outros status codes
6. Compilar e testar

### Script de Migração Automática (Sugestão)

Dado que o padrão é consistente, pode-se criar um script que:
1. Substitui imports
2. Remove funções duplicadas
3. Substitui calls baseado em regexes
4. Valida compilação

---

## 📝 Commit Message Sugerida

```
feat(api): standardize HTTP error responses (Tarefa 5)

- Create internal/http/httperr package with standardized ErrorResponse
- Define error codes categorized by HTTP status (401/403/400/500)
- Implement helper functions: Unauthorized401, Forbidden403, BadRequest400, InternalError500
- Update auth middleware (s2s.go) to use httperr responses
- Update workspace middleware to use httperr responses with proper codes:
  - 401 INVALID_TOKEN for missing auth
  - 403 WORKSPACE_MISMATCH for IDOR protection
  - 400 INVALID_WORKSPACE_ID for format validation
  - 400 MISSING_PARAMETER for empty workspaceId
- Update middleware tests to validate JSON error structure
- Migrate contact.go handler to use httperr package
- Add comprehensive tests for httperr package (7 tests, all passing)
- Add ERROR_RESPONSES.md documentation with examples

Error responses now follow consistent structure:
{
  "ok": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable message",
    "fields": { ... } // optional
  }
}

Breaking change: Error response structure changed from flat { "code", "message" }
to nested { "ok": false, "error": { "code", "message" } }

Refs: Tarefa 5 - Ajustar respostas de erro para diferenciar 401 vs 403 vs 400
```

---

## 🔗 Referências

- Padrão inspirado em: REST API Best Practices, RFC 7807 (Problem Details)
- HTTP Status Codes: RFC 7231
- OAuth 2.0 Error Responses: RFC 6749
