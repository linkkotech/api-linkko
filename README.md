# Linkko API Go

API transacional em Go para o ecossistema Linkko. Serviço production-ready com isolamento multi-tenant via `workspaceId`, autenticação JWT HS256 + S2S (Service-to-Service), rate limiting distribuído, idempotência, e observabilidade completa (OpenTelemetry).

## 🎯 Visão Geral

API independente focada em performance e segurança, projetada para suportar:

- **Multi-tenant**: Isolamento estrito por `workspaceId` no path
- **Dual Authentication**: JWT HS256 (frontend) + S2S tokens (backend services)
- **IDOR Prevention**: Validação automática de workspace entre JWT e path (HTTP 403)
- **Rate Limiting**: Sliding window distribuído via Redis por workspace
- **Idempotency**: SHA256 hash de keys com cache de 24h
- **Observability**: OpenTelemetry (traces + métricas RED) com sampling 10%
- **Graceful Shutdown**: 30s timeout com flush de telemetria

## 🚀 Stack Técnica

| Componente | Tecnologia | Propósito |
|------------|-----------|-----------|
| **Framework** | Go 1.22 | Performance e concorrência |
| **Router** | chi/v5 | HTTP routing rápido e idiomático |
| **Database** | PostgreSQL 16 + pgx/v5 | Pool de conexões eficiente |
| **Cache/Rate Limit** | Redis 7 | State distribuído |
| **Migrations** | golang-migrate/v4 | Versionamento de schema |
| **Auth** | golang-jwt/jwt/v5 | Validação JWT com JWKS |
| **Logging** | zap | Logs estruturados |
| **Tracing** | OpenTelemetry | Observabilidade distribuída |
| **CLI** | Cobra | Interface de linha de comando |

## 📁 Estrutura do Projeto

```
api-linkko/
├── cmd/
│   └── linkko-api/          # CLI entrypoints
│       ├── main.go          # Root command
│       ├── serve.go         # HTTP server
│       ├── migrate.go       # Database migrations
│       └── cleanup.go       # Idempotency cleanup
├── internal/
│   ├── config/              # Environment configuration
│   ├── database/            # PostgreSQL connection & migrations
│   │   └── migrations/      # SQL migration files
│   ├── auth/                # JWT HS256 + S2S authentication
│   │   ├── claims.go        # JWT claims structure
│   │   ├── keys.go          # HS256 key management
│   │   ├── validator.go     # JWT token validator
│   │   ├── s2s.go           # S2S token validator + middleware
│   │   └── middleware.go    # Deprecated (use s2s.go)
│   ├── http/
│   │   ├── httperr/         # Standardized error responses
│   │   │   └── error.go     # 401/403/400 error handling
│   │   ├── handler/         # HTTP request handlers
│   │   └── middleware/      # HTTP middlewares
│   │       ├── workspace.go     # IDOR prevention (403 on mismatch)
│   │       ├── ratelimit.go     # Rate limiting
│   │       ├── idempotency.go   # Idempotent requests
│   │       └── observability.go # Request ID + logging
│   ├── repo/                # Data repositories
│   │   ├── idempotency.go   # Idempotency storage
│   │   └── audit.go         # Audit logging
│   ├── ratelimit/           # Redis rate limiter
│   ├── telemetry/           # OpenTelemetry setup
│   │   ├── tracer.go        # Trace provider
│   │   ├── metrics.go       # Metrics provider
│   │   └── middleware.go    # Instrumentation
│   └── logger/              # Structured logging
├── Dockerfile               # Multi-stage build
├── docker-compose.yml       # Local development stack
├── Makefile                 # Development tasks
└── .env.example             # Environment template
```

## 🏗️ Arquitetura

### Pipeline de Middlewares

```
Request → RequestID → OTel Tracing → Metrics → Logger → Recovery
         → Auth (JWT or S2S) → Workspace Validation (IDOR) → Rate Limit → Idempotency
         → Handler
```

### Fluxo de Autenticação

#### 1. JWT HS256 (Frontend)

```
1. Client → Authorization: Bearer <JWT>
2. Extract JWT from header
3. Validate signature with JWT_HS256_SECRET
4. Verify claims: iss, aud, exp, workspace_id, actor_id
5. Check clock skew (JWT_CLOCK_SKEW_SECONDS)
6. Inject claims into context
7. WorkspaceMiddleware validates: JWT.workspace_id == path.workspaceId
   → If mismatch: HTTP 403 WORKSPACE_MISMATCH
```

#### 2. S2S Authentication (Backend Services)

```
1. Service → Authorization: Bearer <S2S_TOKEN>
          → X-Workspace-Id: <workspace_id>
          → X-Actor-Id: <actor_id>
2. Compare token with S2S_TOKEN_CRM or S2S_TOKEN_MCP
3. Validate required headers (X-Workspace-Id, X-Actor-Id)
4. Inject context with workspace_id and actor_id
5. WorkspaceMiddleware validates: header.workspace_id == path.workspaceId
   → If mismatch: HTTP 403 WORKSPACE_MISMATCH
```

**Key Differences:**
- **JWT**: Workspace ID embedded in signed token (frontend)
- **S2S**: Workspace ID in header, validated by pre-shared token (services)

### Idempotency

- **Hash**: SHA256 do `Idempotency-Key` header
- **Storage**: PostgreSQL com (workspace_id, key_hash) unique constraint
- **TTL**: 24 horas (expires_at)
- **Replay**: Retorna response cached com status `X-Idempotency-Replay: true`
- **Cleanup**: Cloud Scheduler executa `linkko-api cleanup` diariamente

### Rate Limiting (Sliding Window)

```redis
Key: ratelimit:workspace:{workspaceId}
Algorithm:
  1. ZREMRANGEBYSCORE (remove timestamps fora da janela)
  2. ZADD (adiciona timestamp atual)
  3. ZCOUNT (conta requests na janela)
  4. EXPIRE (TTL de 2x a janela)
```

### Observabilidade

- **Sampling**: ParentBased com 10% ratio (honra decisões upstream)
- **Traces**: OTLP gRPC → Jaeger (dev) / Cloud Trace (prod)
- **Métricas RED**:
  - `http_requests_total` (counter)
  - `http_request_duration_seconds` (histogram)
  - `rate_limit_rejections_total` (counter)
- **Logs**: Zap com trace_id, span_id, request_id correlacionados

## 🚦 Quick Start

### Pré-requisitos

- Docker & Docker Compose
- Go 1.22+ (para desenvolvimento local)

### 1. Setup Inicial

```bash
# Clone o repositório
cd g:\github-crm-projects\api-linkko

# Copie o arquivo de configuração
cp .env.example .env

# Edite .env e configure:
# 1. JWT_HS256_SECRET (mínimo 32 caracteres)
#    Gere com: openssl rand -base64 32
#
# 2. S2S_TOKEN_CRM e S2S_TOKEN_MCP (mínimo 32 caracteres)
#    Gere com: openssl rand -hex 32
#
# 3. JWT_ISSUER=linkko-crm-web (deve coincidir com JWT claim 'iss')
#
# 4. JWT_AUDIENCE=linkko-api-gateway (deve coincidir com JWT claim 'aud')
```

### 2. Desenvolvimento com Docker

```bash
# Inicia toda a stack (Postgres, Redis, Jaeger, API)
make dev

# OU manualmente
docker-compose up --build
```

### 3. Acessar Serviços

- **API**: http://localhost:8080
- **Health Check**: http://localhost:8080/health
- **Ready Check**: http://localhost:8080/ready
- **Jaeger UI**: http://localhost:16686

### 4. Testar Autenticação

Ver seção [🔐 Testando com Postman/Insomnia/cURL](#-testando-com-postmaninsomniacurl) abaixo para exemplos completos.

## 🔐 Testando com Postman/Insomnia/cURL

### Pré-requisitos

Antes de testar, garanta que o `.env` está configurado com:

```bash
# JWT Configuration
JWT_HS256_SECRET=your-secret-min-32-chars
JWT_ISSUER=linkko-crm-web
JWT_AUDIENCE=linkko-api-gateway
JWT_CLOCK_SKEW_SECONDS=60

# S2S Tokens
S2S_TOKEN_CRM=crm-service-token-here
S2S_TOKEN_MCP=mcp-service-token-here
```

### Opção 1: Autenticação com JWT HS256 (Frontend)

#### Gerar JWT de Teste

Use [jwt.io](https://jwt.io) ou o script abaixo:

```bash
# Payload exemplo
{
  "iss": "linkko-crm-web",
  "aud": "linkko-api-gateway",
  "workspace_id": "my-workspace-123",
  "actor_id": "user-abc-456",
  "exp": 1737763200  # Unix timestamp (2026-01-25 00:00:00 UTC)
}

# Header
{
  "alg": "HS256",
  "typ": "JWT"
}

# Secret: use o valor de JWT_HS256_SECRET do seu .env
```

**Gerar JWT com Node.js:**

```javascript
const jwt = require('jsonwebtoken');

const token = jwt.sign(
  {
    iss: 'linkko-crm-web',
    aud: 'linkko-api-gateway',
    workspace_id: 'my-workspace-123',
    actor_id: 'user-abc-456',
    exp: Math.floor(Date.now() / 1000) + (60 * 60) // expires in 1 hour
  },
  'your-secret-min-32-chars', // must match JWT_HS256_SECRET
  { algorithm: 'HS256' }
);

console.log(token);
```

#### Requisição cURL

```bash
# Listar contatos do workspace
curl -X GET http://localhost:8080/api/v1/workspaces/my-workspace-123/contacts \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJsaW5ra28tY3JtLXdlYiIsImF1ZCI6Imxpbmtrby1hcGktZ2F0ZXdheSIsIndvcmtzcGFjZV9pZCI6Im15LXdvcmtzcGFjZS0xMjMiLCJhY3Rvcl9pZCI6InVzZXItYWJjLTQ1NiIsImV4cCI6MTczNzc2MzIwMH0.SIGNATURE_HERE" \
  -H "Content-Type: application/json"
```

**Resposta de Sucesso (200 OK):**

```json
{
  "contacts": [...],
  "cursor": "next-page-token"
}
```

**Erro: Token inválido (401 Unauthorized):**

```json
{
  "ok": false,
  "error": {
    "code": "INVALID_TOKEN",
    "message": "invalid token"
  }
}
```

**Erro: Token expirado (401 Unauthorized):**

```json
{
  "ok": false,
  "error": {
    "code": "TOKEN_EXPIRED",
    "message": "token expired"
  }
}
```

**Erro: Workspace mismatch (403 Forbidden):**

```bash
# JWT com workspace_id: "workspace-A"
# Path com workspaceId: "workspace-B"
curl -X GET http://localhost:8080/api/v1/workspaces/workspace-B/contacts \
  -H "Authorization: Bearer <JWT_with_workspace_A>"
```

```json
{
  "ok": false,
  "error": {
    "code": "WORKSPACE_MISMATCH",
    "message": "workspace access denied"
  }
}
```

#### Postman/Insomnia

1. **Método**: GET
2. **URL**: `http://localhost:8080/api/v1/workspaces/my-workspace-123/contacts`
3. **Headers**:
   - `Authorization`: `Bearer <seu-jwt-token>`
   - `Content-Type`: `application/json`

---

### Opção 2: Autenticação S2S (Serviços Backend)

Usada por serviços confiáveis (CRM backend, MCP server) para chamar a API.

#### Requisição cURL

```bash
# Criar tarefa via serviço CRM
curl -X POST http://localhost:8080/api/v1/workspaces/my-workspace-123/tasks \
  -H "Authorization: Bearer crm-service-token-here" \
  -H "X-Workspace-Id: my-workspace-123" \
  -H "X-Actor-Id: service-crm" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Follow up with client",
    "status": "TODO",
    "priority": "HIGH"
  }'
```

**Resposta de Sucesso (201 Created):**

```json
{
  "id": "task-uuid-here",
  "title": "Follow up with client",
  "status": "TODO",
  "priority": "HIGH",
  "workspace_id": "my-workspace-123",
  "created_by": "service-crm",
  "created_at": "2026-01-22T12:00:00Z"
}
```

**Erro: Token S2S inválido (401 Unauthorized):**

```json
{
  "ok": false,
  "error": {
    "code": "INVALID_SIGNATURE",
    "message": "invalid S2S token"
  }
}
```

**Erro: Headers obrigatórios ausentes (400 Bad Request):**

```bash
# Missing X-Workspace-Id or X-Actor-Id
curl -X POST http://localhost:8080/api/v1/workspaces/my-workspace-123/tasks \
  -H "Authorization: Bearer crm-service-token-here"
```

```json
{
  "ok": false,
  "error": {
    "code": "INVALID_PARAMETER",
    "message": "invalid X-Workspace-Id or X-Actor-Id header"
  }
}
```

**Erro: Workspace mismatch (403 Forbidden):**

```bash
# Header X-Workspace-Id: "workspace-A"
# Path workspaceId: "workspace-B"
curl -X POST http://localhost:8080/api/v1/workspaces/workspace-B/tasks \
  -H "Authorization: Bearer crm-service-token-here" \
  -H "X-Workspace-Id: workspace-A" \
  -H "X-Actor-Id: service-crm"
```

```json
{
  "ok": false,
  "error": {
    "code": "WORKSPACE_MISMATCH",
    "message": "workspace access denied"
  }
}
```

#### Postman/Insomnia

1. **Método**: POST
2. **URL**: `http://localhost:8080/api/v1/workspaces/my-workspace-123/tasks`
3. **Headers**:
   - `Authorization`: `Bearer crm-service-token-here` (usar `S2S_TOKEN_CRM` do `.env`)
   - `X-Workspace-Id`: `my-workspace-123` (deve coincidir com path)
   - `X-Actor-Id`: `service-crm`
   - `Content-Type`: `application/json`
4. **Body** (JSON):
   ```json
   {
     "title": "Follow up with client",
     "status": "TODO",
     "priority": "HIGH"
   }
   ```

---

### Regra Crítica: WorkspaceId Mismatch (IDOR Protection)

A API **sempre valida** que o `workspaceId` no path da URL coincide com:

- **JWT**: `workspace_id` claim dentro do token
- **S2S**: `X-Workspace-Id` header

**Se houver mismatch → HTTP 403 Forbidden**

```json
{
  "ok": false,
  "error": {
    "code": "WORKSPACE_MISMATCH",
    "message": "workspace access denied"
  }
}
```

**Por que isso é importante?**

Previne IDOR (Insecure Direct Object Reference):
- Usuário do `workspace-A` não pode acessar dados de `workspace-B`
- Mesmo com token válido, acesso é negado se workspace_id não coincidir
- Proteção contra vazamento de dados entre tenants

**Exemplo de ataque bloqueado:**

```bash
# Atacante com JWT válido para workspace-A tenta acessar workspace-B
curl -X GET http://localhost:8080/api/v1/workspaces/workspace-B/contacts \
  -H "Authorization: Bearer <JWT_with_workspace_A>"

# Resposta: 403 Forbidden (WORKSPACE_MISMATCH)
```

---

### Outros Erros Comuns

#### 400 Bad Request - WorkspaceId inválido

```bash
# WorkspaceId com caracteres inválidos
curl -X GET http://localhost:8080/api/v1/workspaces/invalid@workspace!/contacts \
  -H "Authorization: Bearer <valid-jwt>"
```

```json
{
  "ok": false,
  "error": {
    "code": "INVALID_WORKSPACE_ID",
    "message": "workspaceId must contain only alphanumeric characters, hyphens, and underscores (max 64 chars)"
  }
}
```

#### 400 Bad Request - WorkspaceId ausente

```bash
# WorkspaceId vazio no path
curl -X GET http://localhost:8080/api/v1/workspaces//contacts \
  -H "Authorization: Bearer <valid-jwt>"
```

```json
{
  "ok": false,
  "error": {
    "code": "MISSING_PARAMETER",
    "message": "workspaceId is required in path"
  }
}
```

#### 401 Unauthorized - Authorization header ausente

```bash
curl -X GET http://localhost:8080/api/v1/workspaces/my-workspace/contacts
```

```json
{
  "ok": false,
  "error": {
    "code": "MISSING_AUTHORIZATION",
    "message": "missing authorization header"
  }
}
```

#### 429 Too Many Requests - Rate limit excedido

```bash
# Após 100 requests/min no mesmo workspace
curl -X GET http://localhost:8080/api/v1/workspaces/my-workspace/contacts \
  -H "Authorization: Bearer <valid-jwt>"
```

```json
{
  "ok": false,
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "rate limit exceeded"
  }
}
```

**Headers de resposta:**
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1737763260
Retry-After: 45
```

---

### Coleção Postman/Insomnia

Para facilitar testes, importe a coleção:

**Postman Collection JSON:**

```json
{
  "info": {
    "name": "Linkko API",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "variable": [
    {
      "key": "base_url",
      "value": "http://localhost:8080/api/v1"
    },
    {
      "key": "workspace_id",
      "value": "my-workspace-123"
    },
    {
      "key": "jwt_token",
      "value": "your-jwt-token-here"
    },
    {
      "key": "s2s_token_crm",
      "value": "crm-service-token-here"
    }
  ],
  "item": [
    {
      "name": "JWT - List Contacts",
      "request": {
        "method": "GET",
        "header": [
          {
            "key": "Authorization",
            "value": "Bearer {{jwt_token}}"
          }
        ],
        "url": "{{base_url}}/workspaces/{{workspace_id}}/contacts"
      }
    },
    {
      "name": "S2S - Create Task",
      "request": {
        "method": "POST",
        "header": [
          {
            "key": "Authorization",
            "value": "Bearer {{s2s_token_crm}}"
          },
          {
            "key": "X-Workspace-Id",
            "value": "{{workspace_id}}"
          },
          {
            "key": "X-Actor-Id",
            "value": "service-crm"
          },
          {
            "key": "Content-Type",
            "value": "application/json"
          }
        ],
        "body": {
          "mode": "raw",
          "raw": "{\n  \"title\": \"Follow up with client\",\n  \"status\": \"TODO\",\n  \"priority\": \"HIGH\"\n}"
        },
        "url": "{{base_url}}/workspaces/{{workspace_id}}/tasks"
      }
    }
  ]
}
```

Salve como `linkko-api.postman_collection.json` e importe no Postman/Insomnia.

## 🔧 Comandos CLI

```bash
# Iniciar servidor HTTP
linkko-api serve

# Executar migrations
linkko-api migrate

# Limpar idempotency keys expiradas
linkko-api cleanup
```

### Com Docker

```bash
# Migrations
docker-compose run --rm migrate

# Cleanup
docker-compose run --rm api cleanup

# Logs
make logs
```

## 🌐 Deployment

### EasyPanel (Recomendado para Fase 1)

1. **Containers**:
   - `postgres:16-alpine`
   - `redis:7-alpine`
   - `linkko-api:latest` (build do Dockerfile)

2. **Environment Variables**: Usar `.env` completo

3. **Command Override**:
   - Migrate: `["migrate"]` (executar antes do deploy)
   - API: `["serve"]` (service principal)

4. **Cron Job**: Cleanup diário
   ```bash
   docker run linkko-api:latest cleanup
   ```

### Cloud Run (Produção)

1. **Redis**: Usar Upstash (serverless, sem VPC)

2. **Database**: Cloud SQL PostgreSQL

3. **Migrations**:
   ```bash
   gcloud run jobs execute linkko-migrate \
     --image gcr.io/{project}/linkko-api:latest \
     --args migrate \
     --set-env-vars DATABASE_URL=$DATABASE_URL
   ```

4. **API Service**:
   ```bash
   gcloud run deploy linkko-api \
     --image gcr.io/{project}/linkko-api:latest \
     --args serve \
     --set-secrets=... \
     --allow-unauthenticated \
     --min-instances=1
   ```

5. **Cleanup Job**: Cloud Scheduler (diário às 2:00 UTC)
   ```bash
   gcloud scheduler jobs create http cleanup-idempotency \
     --schedule="0 2 * * *" \
     --uri="https://linkko-api-xxx.run.app/internal/cleanup" \
     --http-method=POST
   ```

## 🔐 Segurança

### Autenticação Dual (JWT + S2S)

| Tipo | Algoritmo | Uso | Validação |
|------|-----------|-----|-----------|
| **JWT HS256** | HMAC-SHA256 | Frontend (crm-web) | Shared secret (`JWT_HS256_SECRET`) |
| **S2S Token** | Pre-shared token | Backend services | Token comparison (`S2S_TOKEN_CRM`, `S2S_TOKEN_MCP`) |

#### JWT HS256 Claims Obrigatórios

```json
{
  "iss": "linkko-crm-web",           // Issuer (must match JWT_ISSUER)
  "aud": "linkko-api-gateway",       // Audience (must match JWT_AUDIENCE)
  "workspace_id": "my-workspace-123", // Workspace identifier
  "actor_id": "user-abc-456",        // User/service identifier
  "exp": 1737763200                  // Expiration (Unix timestamp)
}
```

**Validações:**
- Signature: HMAC-SHA256 com `JWT_HS256_SECRET`
- Clock skew: Tolera até `JWT_CLOCK_SKEW_SECONDS` (default: 60s)
- Required claims: `iss`, `aud`, `workspace_id`, `actor_id`, `exp`

#### S2S Authentication Headers

```http
Authorization: Bearer <S2S_TOKEN_CRM|S2S_TOKEN_MCP>
X-Workspace-Id: my-workspace-123
X-Actor-Id: service-crm
```

**Validações:**
- Token deve coincidir com `S2S_TOKEN_CRM` ou `S2S_TOKEN_MCP`
- Headers `X-Workspace-Id` e `X-Actor-Id` obrigatórios
- Token mínimo 32 caracteres

### IDOR Prevention (Workspace Mismatch)

O `WorkspaceMiddleware` **sempre valida**:

**Para JWT:**
```go
if jwtClaims.WorkspaceID != pathWorkspaceID {
    return 403 Forbidden // WORKSPACE_MISMATCH
}
```

**Para S2S:**
```go
if headerWorkspaceID != pathWorkspaceID {
    return 403 Forbidden // WORKSPACE_MISMATCH
}
```

**Fluxo completo:**

```
1. Request: GET /api/v1/workspaces/workspace-B/contacts
2. Auth: JWT with workspace_id: "workspace-A"
3. WorkspaceMiddleware: 
   - Extrai "workspace-B" do path
   - Compara com JWT.workspace_id ("workspace-A")
   - MISMATCH → 403 Forbidden
4. Response:
   {
     "ok": false,
     "error": {
       "code": "WORKSPACE_MISMATCH",
       "message": "workspace access denied"
     }
   }
```

**Por que é crítico?**
- Previne acesso cross-tenant (usuário workspace-A vendo dados workspace-B)
- Bloqueia IDOR (Insecure Direct Object Reference) attacks
- Garante isolamento multi-tenant mesmo com token válido

### Rate Limiting

- **Limite**: 100 req/min por workspace (configurável via `RATE_LIMIT_PER_WORKSPACE_PER_MIN`)
- **Resposta**: HTTP 429 com headers `X-RateLimit-*` e `Retry-After`
- **Distribuído**: Redis compartilhado entre instâncias

### Idempotency Key Hashing

- Aceita strings livres até 255 chars
- Hash SHA256 antes de armazenar
- Previne injection e garante performance do índice

## 📊 Observabilidade

### Traces

- **Sampling**: 10% das requisições (ParentBased)
- **Exportação**: OTLP gRPC para Jaeger/Cloud Trace
- **Correlation**: trace_id propagado em logs e headers

### Métricas

```
http_requests_total{method, route, status}
http_request_duration_seconds{method, route, status}
rate_limit_rejections_total
```

### Logs Estruturados

```json
{
  "timestamp": "2026-01-20T10:30:00Z",
  "level": "info",
  "msg": "authenticated request",
  "trace_id": "a1b2c3d4...",
  "span_id": "e5f6g7h8...",
  "request_id": "xyz123",
  "workspace_id": "uuid",
  "actor_id": "uuid"
}
```

## 🧪 Troubleshooting

### Autenticação

#### JWT validation failed - INVALID_TOKEN

```json
{
  "ok": false,
  "error": {
    "code": "INVALID_TOKEN",
    "message": "invalid token"
  }
}
```

**Possíveis causas:**
- Token malformado (não é um JWT válido)
- Secret incorreto: `JWT_HS256_SECRET` do .env ≠ secret usado para assinar o JWT
- Algoritmo errado: JWT assinado com RS256 mas API espera HS256

**Como resolver:**
```bash
# 1. Verifique o secret no .env
cat .env | grep JWT_HS256_SECRET

# 2. Teste decodificação em jwt.io com o mesmo secret
# 3. Valide que header JWT tenha: {"alg": "HS256", "typ": "JWT"}
```

#### JWT validation failed - TOKEN_EXPIRED

```json
{
  "ok": false,
  "error": {
    "code": "TOKEN_EXPIRED",
    "message": "token expired"
  }
}
```

**Como resolver:**
- Gere um novo JWT com `exp` futuro
- Verifique clock skew: `JWT_CLOCK_SKEW_SECONDS=60` (default) permite 1 min de diferença

#### JWT validation failed - INVALID_ISSUER

```json
{
  "ok": false,
  "error": {
    "code": "INVALID_ISSUER",
    "message": "invalid token issuer"
  }
}
```

**Como resolver:**
- JWT claim `iss` deve ser exatamente `linkko-crm-web` (valor de `JWT_ISSUER` no .env)

#### JWT validation failed - INVALID_AUDIENCE

```json
{
  "ok": false,
  "error": {
    "code": "INVALID_AUDIENCE",
    "message": "invalid token audience"
  }
}
```

**Como resolver:**
- JWT claim `aud` deve ser exatamente `linkko-api-gateway` (valor de `JWT_AUDIENCE` no .env)

#### S2S authentication failed - INVALID_SIGNATURE

```json
{
  "ok": false,
  "error": {
    "code": "INVALID_SIGNATURE",
    "message": "invalid S2S token"
  }
}
```

**Como resolver:**
```bash
# Verifique que o token enviado corresponde ao .env
cat .env | grep S2S_TOKEN_CRM
cat .env | grep S2S_TOKEN_MCP

# Token deve ser exatamente igual (case-sensitive)
curl ... -H "Authorization: Bearer <copie-exato-do-env>"
```

#### Workspace mismatch - 403 WORKSPACE_MISMATCH

```json
{
  "ok": false,
  "error": {
    "code": "WORKSPACE_MISMATCH",
    "message": "workspace access denied"
  }
}
```

**Como resolver:**

**Para JWT:**
```bash
# JWT claim workspace_id DEVE coincidir com path workspaceId
# Correto:
curl .../workspaces/my-workspace/contacts -H "Authorization: Bearer <JWT_with_workspace_id:my-workspace>"

# Incorreto (403):
curl .../workspaces/other-workspace/contacts -H "Authorization: Bearer <JWT_with_workspace_id:my-workspace>"
```

**Para S2S:**
```bash
# Header X-Workspace-Id DEVE coincidir com path workspaceId
# Correto:
curl .../workspaces/my-workspace/contacts \
  -H "Authorization: Bearer $S2S_TOKEN_CRM" \
  -H "X-Workspace-Id: my-workspace"

# Incorreto (403):
curl .../workspaces/other-workspace/contacts \
  -H "Authorization: Bearer $S2S_TOKEN_CRM" \
  -H "X-Workspace-Id: my-workspace"
```

### Database/Redis

#### Redis connection failed

Verifique se Redis está rodando:
```bash
docker-compose ps redis
docker-compose logs redis
```

### Migrations

#### Migrations locked

```bash
# Forçar unlock (CUIDADO em produção)
docker-compose run --rm api migrate -force 1
```

## 📦 Variáveis de Ambiente

| Variável | Descrição | Exemplo | Obrigatório |
|----------|-----------|---------|-------------|
| **Database** | | | |
| `DATABASE_URL` | PostgreSQL connection string | `postgres://user:pass@host:5432/db` | ✅ |
| **Redis** | | | |
| `REDIS_URL` | Redis connection string (rate limiting) | `redis://:pass@host:6379` | ✅ |
| **JWT HS256** | | | |
| `JWT_HS256_SECRET` | Shared secret (min 32 chars) | `your-secret-here` | ✅ |
| `JWT_ISSUER` | Expected issuer claim | `linkko-crm-web` | ✅ |
| `JWT_AUDIENCE` | Expected audience claim | `linkko-api-gateway` | ✅ |
| `JWT_CLOCK_SKEW_SECONDS` | Clock skew tolerance | `60` | ❌ (default: 60) |
| **S2S Tokens** | | | |
| `S2S_TOKEN_CRM` | Pre-shared token for CRM service | `crm-token-here` | ✅ |
| `S2S_TOKEN_MCP` | Pre-shared token for MCP service | `mcp-token-here` | ✅ |
| **OpenTelemetry** | | | |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP collector endpoint | `localhost:4317` | ❌ |
| `OTEL_SERVICE_NAME` | Service name for traces | `linkko-api-go` | ❌ |
| `OTEL_SAMPLING_RATIO` | Trace sampling ratio (0-1) | `0.1` | ❌ (default: 0.1) |
| **Server** | | | |
| `PORT` | HTTP server port | `8080` | ❌ (default: 8080) |
| **Rate Limiting** | | | |
| `RATE_LIMIT_PER_WORKSPACE_PER_MIN` | Max requests/min per workspace | `100` | ❌ (default: 100) |

### Gerando Secrets

```bash
# JWT HS256 Secret (min 32 chars)
openssl rand -base64 32

# S2S Tokens (recommended 32+ chars)
openssl rand -hex 32
```

## 🛠️ Desenvolvimento

### Instalar dependências

```bash
make install
# OU
go mod download
```

### Rodar testes

```bash
make test
# OU
go test -v -race ./...
```

### Formatar código

```bash
make format
# OU
go fmt ./...
```

### Build local

```bash
go build -o linkko-api ./cmd/linkko-api
./linkko-api serve
```

## 📝 Licença

Proprietary - Linkko © 2026
