# Tarefa 7 - Endpoint /debug/auth ✅

## Status: IMPLEMENTADO

Endpoint de debug protegido para validar claims e informações de autenticação rapidamente durante desenvolvimento.

---

## 📝 Implementação

### Arquivos Criados

1. **[internal/http/handler/debug.go](../../internal/http/handler/debug.go)**
   - Handler `DebugHandler` com endpoint `/debug/auth`
   - Validação de ambiente (`APP_ENV=dev` ou `development`)
   - Retorna 404 em produção/staging/outros ambientes
   - Exige autenticação (JWT ou S2S)

2. **[internal/http/handler/debug_test.go](../../internal/http/handler/debug_test.go)**
   - 8 testes cobrindo todos os cenários:
     - ✅ Bloqueio em produção (404)
     - ✅ Permitido em dev
     - ✅ Sem autenticação (401)
     - ✅ JWT auth
     - ✅ S2S auth
     - ✅ Com workspace no path
     - ✅ Default env (production)

---

## 🔐 Endpoint Specification

### GET /debug/auth

**Disponibilidade:** Apenas em `APP_ENV=dev` ou `APP_ENV=development`

**Autenticação:** Obrigatória (JWT ou S2S)

**Resposta de Sucesso (200 OK):**

```json
{
  "ok": true,
  "data": {
    "authMethod": "jwt",
    "actorId": "user-abc-123",
    "actorType": "user",
    "workspaceIdFromToken": "my-workspace",
    "tokenIssuer": "linkko-crm-web",
    "workspaceValidationPass": true
  }
}
```

**Campos:**

| Campo | Tipo | Descrição |
|-------|------|-----------|
| `authMethod` | string | "jwt" ou "s2s" |
| `client` | string? | Nome do cliente S2S ("crm" ou "mcp") - apenas para S2S |
| `actorId` | string | ID do usuário ou serviço |
| `actorType` | string | "user" ou "service" |
| `workspaceIdFromToken` | string? | Workspace ID do JWT claim - apenas para JWT |
| `workspaceIdFromHeader` | string? | Workspace ID do header X-Workspace-Id - apenas para S2S |
| `workspaceIdFromPath` | string? | Workspace ID do path da URL (se presente) |
| `tokenIssuer` | string? | Issuer do JWT - apenas para JWT |
| `workspaceValidationPass` | boolean | Se true, o workspace middleware validou com sucesso |

**Erros:**

- **401 Unauthorized**: Autenticação ausente ou inválida
- **404 Not Found**: Endpoint acessado em ambiente não-dev

---

### GET /debug/auth/workspaces/{workspaceId}

**Disponibilidade:** Apenas em `APP_ENV=dev` ou `APP_ENV=development`

**Autenticação:** Obrigatória (JWT ou S2S)

**Validação Adicional:** Workspace no path deve coincidir com token/header (WorkspaceMiddleware)

**Resposta de Sucesso (200 OK):**

```json
{
  "ok": true,
  "data": {
    "authMethod": "jwt",
    "actorId": "user-abc-123",
    "actorType": "user",
    "workspaceIdFromToken": "my-workspace",
    "workspaceIdFromPath": "my-workspace",
    "tokenIssuer": "linkko-crm-web",
    "workspaceValidationPass": true
  }
}
```

**Erros:**

- **401 Unauthorized**: Autenticação ausente ou inválida
- **403 Forbidden**: Workspace mismatch (token workspace ≠ path workspace)
- **404 Not Found**: Endpoint acessado em ambiente não-dev

---

## 🧪 Exemplos de Uso

### Configuração Inicial

Adicione ao `.env`:

```bash
APP_ENV=dev  # Habilita endpoint de debug
```

### 1. Testar JWT Auth

```bash
# Gerar JWT em jwt.io com:
# {
#   "iss": "linkko-crm-web",
#   "aud": "linkko-api-gateway",
#   "workspace_id": "my-workspace",
#   "actor_id": "user-123",
#   "exp": 1737763200
# }
# Secret: valor de JWT_HS256_SECRET

export JWT_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

curl -X GET http://localhost:8080/debug/auth \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" | jq
```

**Resposta esperada:**

```json
{
  "ok": true,
  "data": {
    "authMethod": "jwt",
    "actorId": "user-123",
    "actorType": "user",
    "workspaceIdFromToken": "my-workspace",
    "tokenIssuer": "linkko-crm-web",
    "workspaceValidationPass": true
  }
}
```

---

### 2. Testar S2S Auth

```bash
# Usar token do .env
source .env

curl -X GET http://localhost:8080/debug/auth \
  -H "Authorization: Bearer $S2S_TOKEN_CRM" \
  -H "X-Workspace-Id: my-workspace" \
  -H "X-Actor-Id: service-crm" \
  -H "Content-Type: application/json" | jq
```

**Resposta esperada:**

```json
{
  "ok": true,
  "data": {
    "authMethod": "s2s",
    "client": "crm",
    "actorId": "service-crm",
    "actorType": "service",
    "workspaceIdFromHeader": "my-workspace",
    "workspaceValidationPass": true
  }
}
```

---

### 3. Testar com Workspace no Path

```bash
# JWT com workspace_id: "my-workspace"
curl -X GET http://localhost:8080/debug/auth/workspaces/my-workspace \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" | jq
```

**Resposta esperada:**

```json
{
  "ok": true,
  "data": {
    "authMethod": "jwt",
    "actorId": "user-123",
    "actorType": "user",
    "workspaceIdFromToken": "my-workspace",
    "workspaceIdFromPath": "my-workspace",
    "tokenIssuer": "linkko-crm-web",
    "workspaceValidationPass": true
  }
}
```

---

### 4. Testar Workspace Mismatch (403)

```bash
# JWT com workspace_id: "workspace-A"
# Path com workspaceId: "workspace-B"

curl -X GET http://localhost:8080/debug/auth/workspaces/workspace-B \
  -H "Authorization: Bearer $JWT_TOKEN_WORKSPACE_A" \
  -H "Content-Type: application/json" | jq
```

**Resposta esperada (403 Forbidden):**

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

### 5. Testar sem Autenticação (401)

```bash
curl -X GET http://localhost:8080/debug/auth \
  -H "Content-Type: application/json" | jq
```

**Resposta esperada (401 Unauthorized):**

```json
{
  "ok": false,
  "error": {
    "code": "MISSING_AUTHORIZATION",
    "message": "missing authorization header"
  }
}
```

---

### 6. Testar em Produção (404)

```bash
# Configurar APP_ENV=production
export APP_ENV=production

curl -X GET http://localhost:8080/debug/auth \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json"
```

**Resposta esperada (404 Not Found):**

```
404 page not found
```

**Log gerado:**

```json
{
  "level": "warn",
  "msg": "debug endpoint accessed in non-dev environment",
  "app_env": "production",
  "remote_addr": "127.0.0.1:xxxxx"
}
```

---

## 🚀 Integração com Servidor

Para habilitar o endpoint no servidor, adicione as rotas em `cmd/linkko-api/serve.go`:

```go
// Debug routes (only in dev)
debugHandler := handler.NewDebugHandler()
r.Get("/debug/auth", debugHandler.GetAuthDebug)
r.Get("/debug/auth/workspaces/{workspaceId}", debugHandler.GetAuthDebugWithWorkspace)
```

**Importante:** As rotas devem estar **após** o middleware de autenticação:

```go
// Protected routes
r.Group(func(r chi.Router) {
    // Auth middleware
    r.Use(auth.AuthMiddleware(keyStore))
    
    // Debug routes (protected by auth)
    debugHandler := handler.NewDebugHandler()
    r.Get("/debug/auth", debugHandler.GetAuthDebug)
    
    // Workspace-specific routes
    r.Route("/workspaces/{workspaceId}", func(r chi.Router) {
        // Workspace middleware
        r.Use(middleware.WorkspaceMiddleware)
        
        // Debug route with workspace validation
        r.Get("/debug/auth", debugHandler.GetAuthDebugWithWorkspace)
        
        // ... other routes
    })
})
```

---

## 🎯 Casos de Uso

### 1. Verificar se JWT está correto

```bash
# Gerar JWT e testar imediatamente
curl http://localhost:8080/debug/auth \
  -H "Authorization: Bearer $NEW_JWT_TOKEN" | jq .data
```

**Validações:**
- ✅ Token está assinado corretamente?
- ✅ Claims estão presentes (workspace_id, actor_id)?
- ✅ Issuer está correto?

---

### 2. Debugar Workspace Mismatch

```bash
# Ver exatamente quais workspaces estão sendo comparados
curl http://localhost:8080/debug/auth/workspaces/test-workspace \
  -H "Authorization: Bearer $JWT_TOKEN" | jq
```

**Output mostra:**
- `workspaceIdFromToken`: workspace no JWT
- `workspaceIdFromPath`: workspace na URL
- `workspaceValidationPass`: se passou na validação

---

### 3. Verificar S2S Headers

```bash
# Ver se headers S2S estão sendo lidos corretamente
curl http://localhost:8080/debug/auth \
  -H "Authorization: Bearer $S2S_TOKEN_CRM" \
  -H "X-Workspace-Id: my-workspace" \
  -H "X-Actor-Id: service-crm" | jq .data
```

**Validações:**
- ✅ Client identificado corretamente (crm ou mcp)?
- ✅ Workspace ID lido do header?
- ✅ Actor ID correto?

---

### 4. Testar Actor Type

```bash
# Ver se actor_type está sendo inferido corretamente
curl http://localhost:8080/debug/auth \
  -H "Authorization: Bearer $JWT_TOKEN" | jq .data.actorType
```

**Esperado:**
- JWT → `"user"`
- S2S → `"service"`

---

## 🔒 Segurança

### Proteções Implementadas

1. **Ambiente-specific:**
   - Endpoint **só funciona** em `APP_ENV=dev` ou `development`
   - Retorna 404 em produção/staging
   - Logs de acesso suspeito (warn em produção)

2. **Autenticação obrigatória:**
   - Mesmo em dev, exige JWT ou S2S válido
   - Retorna 401 se auth ausente/inválida

3. **Não expõe tokens:**
   - Response **nunca** retorna o token raw
   - Apenas metadata (issuer, client, IDs)

4. **Default seguro:**
   - Se `APP_ENV` não configurado → assume "production"
   - Princípio de fail-safe

---

## 📊 Testes

### Executar Testes

```bash
# Todos os testes do debug handler
go test ./internal/http/handler/ -run TestDebug -v

# Teste específico
go test ./internal/http/handler/ -run TestDebugHandler_GetAuthDebug_ProductionBlocked -v
```

### Cobertura

```bash
go test ./internal/http/handler/ -run TestDebug -cover
```

**Cenários testados:**

- ✅ Bloqueio em produção (APP_ENV=production)
- ✅ Permitido em development (APP_ENV=development)
- ✅ Permitido em dev (APP_ENV=dev)
- ✅ Sem autenticação → 401
- ✅ JWT auth → campos corretos
- ✅ S2S auth → campos corretos
- ✅ Com workspace no path → workspaceIdFromPath populado
- ✅ Default env (sem APP_ENV) → production

---

## 🎉 Benefícios

1. **Debug Rápido:**
   - Ver exatamente o que a API está lendo do token/headers
   - Identificar problemas de autenticação em segundos

2. **Testes de Integração:**
   - Validar tokens gerados por outros serviços
   - Confirmar que claims estão corretos antes de chamar endpoints reais

3. **Onboarding:**
   - Novos desenvolvedores podem entender autenticação rapidamente
   - Exemplos práticos de JWT e S2S

4. **Troubleshooting:**
   - Verificar workspace mismatch
   - Confirmar issuer/audience corretos
   - Ver tipo de auth detectado (jwt vs s2s)

5. **Seguro:**
   - Não funciona em produção
   - Não expõe tokens sensíveis
   - Exige autenticação válida mesmo em dev

---

## 📋 Checklist de Implementação

- [x] Criar `internal/http/handler/debug.go`
- [x] Criar `internal/http/handler/debug_test.go`
- [x] Implementar `GetAuthDebug()`
- [x] Implementar `GetAuthDebugWithWorkspace()`
- [x] Validação de ambiente (APP_ENV)
- [x] Retornar 404 em produção
- [x] Exigir autenticação
- [x] Não expor tokens
- [x] Testes unitários (8 cenários)
- [ ] Adicionar rotas em `serve.go` (pendente)
- [ ] Documentar em README.md (pendente)
- [ ] Testar end-to-end com servidor rodando

---

## 🚧 Próximos Passos

### 1. Integrar no Servidor

Adicionar em `cmd/linkko-api/serve.go`:

```go
// Após configurar auth middleware
debugHandler := handler.NewDebugHandler()

// Route sem workspace (básico)
r.Get("/debug/auth", debugHandler.GetAuthDebug)

// Route com workspace (testa workspace middleware)
r.Route("/workspaces/{workspaceId}", func(r chi.Router) {
    r.Use(middleware.WorkspaceMiddleware)
    r.Get("/debug/auth", debugHandler.GetAuthDebugWithWorkspace)
})
```

### 2. Atualizar README.md

Adicionar seção "Debug Endpoints":

```markdown
## 🐛 Debug Endpoints (Dev Only)

### GET /debug/auth

Returns authentication information extracted from the request.
Only available when `APP_ENV=dev` or `APP_ENV=development`.

**Example:**
\`\`\`bash
curl http://localhost:8080/debug/auth \
  -H "Authorization: Bearer $JWT_TOKEN"
\`\`\`

See [docs/TAREFA_7_DEBUG_ENDPOINT.md](docs/TAREFA_7_DEBUG_ENDPOINT.md) for full documentation.
```

### 3. Testar End-to-End

```bash
# 1. Configurar ambiente
export APP_ENV=dev

# 2. Iniciar servidor
make dev

# 3. Gerar JWT de teste
# (usar jwt.io com JWT_HS256_SECRET do .env)

# 4. Testar endpoint
curl http://localhost:8080/debug/auth \
  -H "Authorization: Bearer $JWT_TOKEN" | jq

# 5. Testar com workspace
curl http://localhost:8080/debug/auth/workspaces/my-workspace \
  -H "Authorization: Bearer $JWT_TOKEN" | jq

# 6. Testar S2S
source .env
curl http://localhost:8080/debug/auth \
  -H "Authorization: Bearer $S2S_TOKEN_CRM" \
  -H "X-Workspace-Id: my-workspace" \
  -H "X-Actor-Id: service-crm" | jq
```

---

## 💡 Dicas

**Para Frontend Developers:**
- Use este endpoint para validar JWTs gerados pelo frontend
- Confirme que workspace_id está correto antes de chamar APIs reais

**Para Backend Developers:**
- Use para debugar S2S tokens
- Verifique se headers X-Workspace-Id e X-Actor-Id estão sendo enviados

**Para QA/Testing:**
- Valide diferentes cenários de autenticação rapidamente
- Teste workspace mismatch sem precisar criar recursos reais

**Para DevOps:**
- Confirme que APP_ENV está configurado corretamente em cada ambiente
- Valide que endpoint retorna 404 em staging/production

---

## 📚 Referências

- [RFC 7519 (JWT)](https://datatracker.ietf.org/doc/html/rfc7519)
- [Chi Router](https://github.com/go-chi/chi)
- [Tarefa 5 - Standardized Error Responses](./TAREFA_5_IMPLEMENTATION.md)
- [Tarefa 6 - Authentication Documentation](./TAREFA_6_DOCUMENTATION.md)

---

## ✅ Resumo

**Implementação completa do endpoint `/debug/auth`:**

✅ Handler criado com validação de ambiente
✅ 8 testes unitários cobrindo todos os cenários  
✅ Seguro (404 em produção, não expõe tokens)
✅ Útil (mostra exatamente o que a API vê)
✅ Documentação completa com exemplos cURL

**Pendente:**
- Integração com rotas do servidor (serve.go)
- Atualização do README.md
- Teste end-to-end

O código está pronto para uso! Basta adicionar as rotas no servidor e testar. 🎉
