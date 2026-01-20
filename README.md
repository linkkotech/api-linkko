# Linkko API Go

API transacional em Go para o ecossistema Linkko. Serviço production-ready com isolamento multi-tenant via `workspaceId`, autenticação S2S com multi-issuer JWT (JWKS), rate limiting distribuído, idempotência, e observabilidade completa (OpenTelemetry).

## 🎯 Visão Geral

API independente focada em performance e segurança, projetada para suportar:

- **Multi-tenant**: Isolamento estrito por `workspaceId` no path
- **S2S Authentication**: JWT com múltiplos issuers (crm-web HS256 + mcp-server RS256)
- **IDOR Prevention**: Validação automática de workspace entre JWT e path
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
│   ├── auth/                # Multi-issuer JWT with JWKS
│   │   ├── claims.go        # Custom claims
│   │   ├── keys.go          # Key store (HS256/RS256)
│   │   ├── validator.go     # Token validators
│   │   ├── resolver.go      # Dynamic key resolution
│   │   └── middleware.go    # Auth middleware
│   ├── http/
│   │   └── middleware/      # HTTP middlewares
│   │       ├── workspace.go     # IDOR prevention
│   │       ├── ratelimit.go     # Rate limiting
│   │       └── idempotency.go   # Idempotent requests
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
         → JWT Auth → Workspace Validation → Rate Limit → Idempotency
         → Handler
```

### Fluxo de Autenticação Multi-Issuer

1. **Extração**: Bearer token do header `Authorization`
2. **Pre-decode**: JWT header/payload sem validar assinatura
3. **Resolução**: Extrair `iss` (issuer) e `kid` (key ID)
4. **Lookup**: Buscar validator no KeyResolver por issuer
5. **Validação**: Validator específico (HS256/RS256) valida token
6. **Claims**: Extrair `workspace_id`, `actor_id`, verificar `aud`
7. **Injeção**: Claims no context para middlewares downstream

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
# - JWT_SECRET_CRM_V1 (mínimo 32 caracteres)
# - JWT_PUBLIC_KEY_MCP_V1 (chave pública RSA em PEM)
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

```bash
# Gerar um JWT de teste (exemplo com HS256)
# Payload deve conter: workspace_id, actor_id, iss, aud, exp

curl -X GET http://localhost:8080/v1/workspaces/{workspace_id}/example \
  -H "Authorization: Bearer {seu-jwt-token}"
```

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

### Multi-Issuer JWT

| Issuer | Algoritmo | Key Type | Uso |
|--------|-----------|----------|-----|
| `linkko-crm-web` | HS256 | Secret | Frontend Next.js |
| `linkko-mcp-server` | RS256 | Public Key | Agentes IA |

**Claims Obrigatórios**:
- `iss`: Issuer (linkko-crm-web ou linkko-mcp-server)
- `aud`: Audience (linkko-api-gateway)
- `workspace_id`: UUID do workspace
- `actor_id`: UUID do usuário/agente
- `exp`: Expiration timestamp

### IDOR Prevention

O `WorkspaceMiddleware` valida:
```go
if claims.WorkspaceID != pathWorkspaceID {
    return 403 Forbidden
}
```

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

### Redis connection failed

Verifique se Redis está rodando:
```bash
docker-compose ps redis
docker-compose logs redis
```

### Migrations locked

```bash
# Forçar unlock (CUIDADO em produção)
docker-compose run --rm api migrate -force 1
```

### Rate limit não funciona em multi-instance

Certifique-se de que todas as instâncias apontam para o mesmo Redis.

### JWT validation failed

- Verifique `kid` no header JWT
- Confirme que issuer está em `JWT_ALLOWED_ISSUERS`
- Valide formato da chave pública (PEM)

## 📦 Variáveis de Ambiente

| Variável | Descrição | Exemplo | Obrigatório |
|----------|-----------|---------|-------------|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://user:pass@host:5432/db` | ✅ |
| `REDIS_URL` | Redis connection string | `redis://:pass@host:6379` | ✅ |
| `JWT_SECRET_CRM_V1` | Secret HS256 para crm-web | `your-secret-min-32-chars` | ✅ |
| `JWT_PUBLIC_KEY_MCP_V1` | Public key RS256 para mcp | `-----BEGIN PUBLIC KEY-----...` | ✅ |
| `JWT_ALLOWED_ISSUERS` | Lista de issuers permitidos | `linkko-crm-web,linkko-mcp-server` | ✅ |
| `JWT_AUDIENCE` | Audience esperado | `linkko-api-gateway` | ✅ |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Endpoint do coletor OTLP | `localhost:4317` | ❌ (default) |
| `OTEL_SERVICE_NAME` | Nome do serviço | `linkko-api-go` | ❌ (default) |
| `OTEL_SAMPLING_RATIO` | Taxa de sampling (0-1) | `0.1` | ❌ (default: 0.1) |
| `PORT` | Porta HTTP | `8080` | ❌ (default: 8080) |
| `RATE_LIMIT_PER_WORKSPACE_PER_MIN` | Limite por workspace | `100` | ❌ (default: 100) |

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
