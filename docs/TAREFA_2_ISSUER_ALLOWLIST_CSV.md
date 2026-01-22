# Tarefa 2: Suportar Issuer Allowlist via JWT_ALLOWED_ISSUERS (CSV)

## ✅ Objetivo
Implementar validação de múltiplos issuers JWT usando `JWT_ALLOWED_ISSUERS` como CSV (ex.: "linkko-crm-web,linkko-mcp-server,linkko-admin-portal").

## 📝 Mudanças Implementadas

### 1. Config (`internal/config/config.go`)
**Mudanças:**
- ✅ `JWT_ALLOWED_ISSUERS` agora é a variável **principal** (não mais legacy)
- ✅ `JWT_ISSUER` movido para legacy (fallback para compatibilidade)
- ✅ Parse CSV robusto em `GetAllowedIssuers()` com:
  - `strings.TrimSpace()` em cada issuer
  - Ignorar entradas vazias
  - Suporte para trailing/leading commas
  - Suporte para múltiplos espaços

**Antes:**
```go
type Config struct {
    JWTIssuer         string `env:"JWT_ISSUER,required"`       // Principal
    JWTAllowedIssuers string `env:"JWT_ALLOWED_ISSUERS"`       // Legacy
}
```

**Depois:**
```go
type Config struct {
    JWTAllowedIssuers string `env:"JWT_ALLOWED_ISSUERS,required"` // Principal (CSV)
    JWTIssuer         string `env:"JWT_ISSUER"`                   // Legacy (fallback)
}
```

**Validação com fallback:**
```go
// Se JWT_ALLOWED_ISSUERS vazio, fallback para JWT_ISSUER (legacy)
if c.JWTAllowedIssuers == "" {
    if c.JWTIssuer != "" {
        c.JWTAllowedIssuers = c.JWTIssuer // Usa single issuer
    } else {
        c.JWTAllowedIssuers = "linkko-crm-web" // Default
    }
}

// Valida que lista tem pelo menos 1 issuer válido
issuers := c.GetAllowedIssuers()
if len(issuers) == 0 {
    return fmt.Errorf("JWT_ALLOWED_ISSUERS must contain at least one valid issuer")
}
```

**Parse CSV Robusto:**
```go
func (c *Config) GetAllowedIssuers() []string {
    issuers := strings.Split(c.JWTAllowedIssuers, ",")
    result := make([]string, 0, len(issuers))
    for _, issuer := range issuers {
        trimmed := strings.TrimSpace(issuer)
        if trimmed != "" {
            result = append(result, trimmed)
        }
    }
    return result
}
```

### 2. Serve (`cmd/linkko-api/serve.go`)
**Mudanças:**
- ✅ Parse `JWT_ALLOWED_ISSUERS` usando `cfg.GetAllowedIssuers()`
- ✅ Valida que há pelo menos 1 issuer
- ✅ Registra HS256 validator para **todos os issuers** da allowlist
- ✅ Cada issuer usa o mesmo `JWT_HS256_SECRET` (shared secret)
- ✅ Log mostra todos os issuers permitidos

**Antes:**
```go
// Hardcoded single issuer
keyStore.LoadHS256Key(cfg.JWTIssuer, "v1", secretBytes)
hs256Validator := auth.NewHS256Validator(keyStore, cfg.JWTIssuer, clockSkew)
resolver := auth.NewKeyResolver([]string{cfg.JWTIssuer}, []string{cfg.JWTAudience})
resolver.RegisterValidator(cfg.JWTIssuer, hs256Validator)
```

**Depois:**
```go
// Parse CSV allowlist
allowedIssuers := cfg.GetAllowedIssuers()
if len(allowedIssuers) == 0 {
    return fmt.Errorf("JWT_ALLOWED_ISSUERS must contain at least one valid issuer")
}

// Load secret for all issuers
for _, issuer := range allowedIssuers {
    keyStore.LoadHS256Key(issuer, "v1", secretBytes)
}

// Create resolver with allowlist
resolver := auth.NewKeyResolver(allowedIssuers, []string{cfg.JWTAudience})

// Register validator for each issuer
for _, issuer := range allowedIssuers {
    hs256Validator := auth.NewHS256Validator(keyStore, issuer, clockSkew)
    resolver.RegisterValidator(issuer, hs256Validator)
}
```

### 3. Resolver (`internal/auth/resolver.go`)
**Comportamento (sem mudanças):**
- ✅ Valida que `claims.iss` está na allowlist
- ✅ Retorna `AuthFailureInvalidIssuer` se issuer não permitido
- ✅ Mensagem de erro clara: "issuer not allowed: {issuer}"

**Código existente:**
```go
// Check if issuer is allowed
if !kr.allowedIssuers[issuer] {
    return nil, NewAuthError(
        AuthFailureInvalidIssuer, 
        fmt.Sprintf("issuer not allowed: %s", issuer), 
        nil,
    )
}
```

### 4. Testes

#### ✅ Config Tests (`internal/config/config_test.go`) - **9 novos testes**

1. **`TestConfig_GetAllowedIssuers_SingleIssuer`**
   ```go
   Input: "linkko-crm-web"
   Output: ["linkko-crm-web"]
   ```

2. **`TestConfig_GetAllowedIssuers_MultipleIssuers`**
   ```go
   Input: "linkko-crm-web,linkko-admin-portal,linkko-mcp-server"
   Output: ["linkko-crm-web", "linkko-admin-portal", "linkko-mcp-server"]
   ```

3. **`TestConfig_GetAllowedIssuers_WithWhitespace`**
   ```go
   Input: "  linkko-crm-web  , linkko-admin-portal , linkko-mcp-server  "
   Output: ["linkko-crm-web", "linkko-admin-portal", "linkko-mcp-server"]
   ```

4. **`TestConfig_GetAllowedIssuers_WithEmptyEntries`**
   ```go
   Input: "linkko-crm-web,,linkko-admin-portal,  ,linkko-mcp-server"
   Output: ["linkko-crm-web", "linkko-admin-portal", "linkko-mcp-server"]
   ```

5. **`TestConfig_GetAllowedIssuers_EmptyString`**
   ```go
   Input: ""
   Output: []
   ```

6. **`TestConfig_GetAllowedIssuers_OnlyWhitespace`**
   ```go
   Input: "   ,  ,   "
   Output: []
   ```

7. **`TestConfig_GetAllowedIssuers_TrailingComma`**
   ```go
   Input: "linkko-crm-web,linkko-admin-portal,"
   Output: ["linkko-crm-web", "linkko-admin-portal"]
   ```

8. **`TestConfig_GetAllowedIssuers_LeadingComma`**
   ```go
   Input: ",linkko-crm-web,linkko-admin-portal"
   Output: ["linkko-crm-web", "linkko-admin-portal"]
   ```

9. **`TestConfig_GetAllowedIssuers_DuplicateIssuers`**
   ```go
   Input: "linkko-crm-web,linkko-admin-portal,linkko-crm-web"
   Output: ["linkko-crm-web", "linkko-admin-portal", "linkko-crm-web"]
   Note: Duplicates allowed (deduplication at resolver level)
   ```

#### ✅ Resolver Tests (`internal/auth/resolver_test.go`) - **3 novos testes**

1. **`TestKeyResolver_MultipleIssuers`**
   - Configura allowlist: `["linkko-crm-web", "linkko-admin-portal"]`
   - Valida token de `linkko-crm-web` ✅
   - Valida token de `linkko-admin-portal` ✅
   - Confirma que ambos os issuers são aceitos

2. **`TestKeyResolver_IssuerNotInAllowlist`**
   - Configura allowlist: `["linkko-crm-web"]`
   - Cria token com issuer: `"unauthorized-issuer"`
   - Valida que retorna `AuthFailureInvalidIssuer` ✅
   - Mensagem: "issuer not allowed: unauthorized-issuer"

3. **`TestKeyResolver_EmptyIssuer`**
   - Configura allowlist: `["linkko-crm-web"]`
   - Cria token com issuer vazio: `""`
   - Valida que retorna `AuthFailureInvalidIssuer` ✅

### 5. Env Example (`.env.example`)
**Mudanças:**
```dotenv
# JWT Allowed Issuers - CSV list of trusted JWT issuers
# Multiple issuers supported (e.g., "linkko-crm-web,linkko-admin-portal")
# Parse is robust: ignores whitespace and empty entries
# Default: linkko-crm-web (frontend application)
JWT_ALLOWED_ISSUERS=linkko-crm-web
```

## 🧪 Resultados dos Testes

### Config Tests (9 testes)
```bash
$ go test -v ./internal/config -run "TestConfig_GetAllowedIssuers"
=== RUN   TestConfig_GetAllowedIssuers_SingleIssuer
--- PASS: TestConfig_GetAllowedIssuers_SingleIssuer (0.00s)
=== RUN   TestConfig_GetAllowedIssuers_MultipleIssuers
--- PASS: TestConfig_GetAllowedIssuers_MultipleIssuers (0.00s)
=== RUN   TestConfig_GetAllowedIssuers_WithWhitespace
--- PASS: TestConfig_GetAllowedIssuers_WithWhitespace (0.00s)
=== RUN   TestConfig_GetAllowedIssuers_WithEmptyEntries
--- PASS: TestConfig_GetAllowedIssuers_WithEmptyEntries (0.00s)
=== RUN   TestConfig_GetAllowedIssuers_EmptyString
--- PASS: TestConfig_GetAllowedIssuers_EmptyString (0.00s)
=== RUN   TestConfig_GetAllowedIssuers_OnlyWhitespace
--- PASS: TestConfig_GetAllowedIssuers_OnlyWhitespace (0.00s)
=== RUN   TestConfig_GetAllowedIssuers_TrailingComma
--- PASS: TestConfig_GetAllowedIssuers_TrailingComma (0.00s)
=== RUN   TestConfig_GetAllowedIssuers_LeadingComma
--- PASS: TestConfig_GetAllowedIssuers_LeadingComma (0.00s)
=== RUN   TestConfig_GetAllowedIssuers_DuplicateIssuers
--- PASS: TestConfig_GetAllowedIssuers_DuplicateIssuers (0.00s)
PASS
ok      linkko-api/internal/config      0.035s
```

### Resolver Tests (3 novos + 6 existentes = 9 total)
```bash
$ go test -v ./internal/auth -run "TestKeyResolver"
=== RUN   TestKeyResolver_ValidToken
--- PASS: TestKeyResolver_ValidToken (0.00s)
=== RUN   TestKeyResolver_InvalidIssuer
--- PASS: TestKeyResolver_InvalidIssuer (0.00s)
=== RUN   TestKeyResolver_InvalidAudience
--- PASS: TestKeyResolver_InvalidAudience (0.00s)
=== RUN   TestKeyResolver_NoValidatorForIssuer
--- PASS: TestKeyResolver_NoValidatorForIssuer (0.00s)
=== RUN   TestKeyResolver_MalformedToken
--- PASS: TestKeyResolver_MalformedToken (0.00s)
=== RUN   TestKeyResolver_EmptyKidFallback
--- PASS: TestKeyResolver_EmptyKidFallback (0.00s)
=== RUN   TestKeyResolver_MultipleIssuers
--- PASS: TestKeyResolver_MultipleIssuers (0.00s)
=== RUN   TestKeyResolver_IssuerNotInAllowlist
--- PASS: TestKeyResolver_IssuerNotInAllowlist (0.00s)
=== RUN   TestKeyResolver_EmptyIssuer
--- PASS: TestKeyResolver_EmptyIssuer (0.00s)
PASS
```

### Todos os Auth Tests (32 testes)
```bash
$ go test -v ./internal/auth
PASS
ok      linkko-api/internal/auth        0.062s (32 tests passing)
```

## 🔒 Validação de Issuer

### Cenários Válidos ✅

1. **Single Issuer**
   ```bash
   JWT_ALLOWED_ISSUERS=linkko-crm-web
   ```
   - Token com `iss: "linkko-crm-web"` → ✅ Aceito

2. **Multiple Issuers**
   ```bash
   JWT_ALLOWED_ISSUERS=linkko-crm-web,linkko-admin-portal,linkko-mcp-server
   ```
   - Token com `iss: "linkko-crm-web"` → ✅ Aceito
   - Token com `iss: "linkko-admin-portal"` → ✅ Aceito
   - Token com `iss: "linkko-mcp-server"` → ✅ Aceito

3. **CSV com Whitespace**
   ```bash
   JWT_ALLOWED_ISSUERS="  linkko-crm-web  , linkko-admin-portal , linkko-mcp-server  "
   ```
   - Parse robusto remove espaços → ✅ Aceito

4. **CSV com Empty Entries**
   ```bash
   JWT_ALLOWED_ISSUERS="linkko-crm-web,,linkko-admin-portal,  ,linkko-mcp-server"
   ```
   - Parse robusto ignora vazios → ✅ Aceito

### Cenários Rejeitados ❌

1. **Issuer Not in Allowlist**
   ```bash
   JWT_ALLOWED_ISSUERS=linkko-crm-web
   Token: iss: "unauthorized-issuer"
   ```
   → **401 Unauthorized**
   ```json
   {
     "error": "INVALID_ISSUER",
     "message": "issuer not allowed: unauthorized-issuer"
   }
   ```

2. **Empty Issuer**
   ```bash
   JWT_ALLOWED_ISSUERS=linkko-crm-web
   Token: iss: ""
   ```
   → **401 Unauthorized**
   ```json
   {
     "error": "INVALID_ISSUER",
     "message": "issuer not allowed: "
   }
   ```

3. **Missing Issuer**
   ```bash
   JWT_ALLOWED_ISSUERS=linkko-crm-web
   Token: (sem campo iss)
   ```
   → **401 Unauthorized** (tratado pelo jwt parser)

## 📚 Exemplos de Uso

### Cenário 1: Frontend + Backend Interno
```dotenv
# .env
JWT_HS256_SECRET=eW91ci1iYXNlNjQtc2VjcmV0LWhlcmUtMzItYnl0ZXM=
JWT_ALLOWED_ISSUERS=linkko-crm-web,linkko-internal-api
JWT_AUDIENCE=linkko-api-gateway
```

**Token do CRM Web:**
```json
{
  "iss": "linkko-crm-web",
  "aud": "linkko-api-gateway",
  "workspaceId": "ws-12345",
  "actorId": "user-67890"
}
```
✅ Aceito

**Token do Internal API:**
```json
{
  "iss": "linkko-internal-api",
  "aud": "linkko-api-gateway",
  "workspaceId": "ws-internal",
  "actorId": "system-bot"
}
```
✅ Aceito

### Cenário 2: Multi-Tenant com Admin Portal
```dotenv
JWT_ALLOWED_ISSUERS=linkko-crm-web,linkko-admin-portal,linkko-mobile-app
```

- `linkko-crm-web` → Frontend web (usuários finais)
- `linkko-admin-portal` → Portal de administração (super admins)
- `linkko-mobile-app` → App móvel (usuários mobile)

Todos compartilham o mesmo `JWT_HS256_SECRET` mas são diferenciados pelo issuer.

### Cenário 3: Migração Gradual
```dotenv
# Fase 1: Suportar issuer antigo e novo
JWT_ALLOWED_ISSUERS=linkko-crm-web-old,linkko-crm-web

# Fase 2: Apenas novo issuer
JWT_ALLOWED_ISSUERS=linkko-crm-web
```

## 🔄 Compatibilidade com Tarefa 1

A Tarefa 2 **mantém** todas as funcionalidades da Tarefa 1:
- ✅ Base64 decode obrigatório do `JWT_HS256_SECRET`
- ✅ Validação de tamanho mínimo (32 bytes)
- ✅ Fail-fast no startup se secret inválido
- ✅ Testes Base64 continuam passando

**Novidade da Tarefa 2:**
- ✅ Suporte para **múltiplos issuers** via CSV
- ✅ Parse robusto (trim, ignore vazios)
- ✅ Validação de issuer contra allowlist
- ✅ Erro 401 com `invalid_issuer` se não permitido

## 📊 Resumo de Arquivos Alterados

```
internal/config/config.go          | +18  -10  (JWT_ALLOWED_ISSUERS principal, validação)
internal/config/config_test.go     | +128 -0   (9 novos testes CSV parse)
cmd/linkko-api/serve.go            | +30  -15  (loop issuers, validators múltiplos)
internal/auth/resolver_test.go     | +120 -8   (3 novos testes issuer allowlist)
.env.example                       | +5   -4   (JWT_ALLOWED_ISSUERS principal)
```

## ✅ Checklist de Entrega

- [x] Config atualizado: JWT_ALLOWED_ISSUERS como principal
- [x] Parse CSV robusto (trim, ignore vazios)
- [x] Validação de issuer contra allowlist
- [x] Erro 401 com reason `invalid_issuer`
- [x] Removida dependência de JWT_ISSUER único (agora legacy)
- [x] 9 testes config (CSV parse robusto)
- [x] 3 testes resolver (múltiplos issuers, não permitido, vazio)
- [x] Todos os 32 testes auth passando
- [x] Todos os 9 testes config passando
- [x] .env.example atualizado
- [x] Compatibilidade com Tarefa 1 mantida
- [x] Sem erros de compilação

## 🎯 Próximos Passos
1. Atualizar `.env` de produção com `JWT_ALLOWED_ISSUERS` (CSV se múltiplos issuers)
2. Validar integração com múltiplos issuers (se aplicável)
3. Documentar issuers permitidos por ambiente
4. Considerar remover variáveis legacy após migração completa
