# 🚀 Setup Completo - Linkko API Go

## Passo 1: Instalar Go 1.22+

### Windows

1. **Download do Go:**
   - Acesse: https://go.dev/dl/
   - Baixe: `go1.22.x.windows-amd64.msi` (versão mais recente)

2. **Instalar:**
   - Execute o instalador MSI
   - Siga o wizard (next, next, install)
   - O Go será instalado em `C:\Program Files\Go`

3. **Verificar instalação:**
   ```powershell
   go version
   # Deve mostrar: go version go1.22.x windows/amd64
   ```

4. **Configurar variáveis de ambiente (se necessário):**
   - O instalador já configura automaticamente
   - Verifique: `$env:Path` deve conter `C:\Program Files\Go\bin`
   - GOPATH padrão: `C:\Users\{seu-usuario}\go`

### Alternativa: Winget (Windows Package Manager)

```powershell
winget install GoLang.Go
```

### Alternativa: Chocolatey

```powershell
choco install golang
```

## Passo 2: Configurar GOPATH (Opcional)

```powershell
# Ver configuração atual
go env

# O GOPATH padrão está OK, mas pode customizar:
# setx GOPATH "G:\go-workspace"
```

## Passo 3: Instalar Dependências do Projeto

```powershell
cd G:\github-crm-projects\api-linkko

# Download de todas as dependências
go mod download

# Limpar e reorganizar dependências
go mod tidy

# Verificar que tudo está OK
go mod verify
```

## Passo 4: Verificar Instalação

```powershell
# Compilar projeto (não executa, apenas verifica)
go build ./cmd/linkko-api

# Se compilou sem erros, está tudo certo!
```

## Passo 5: Rodar com Docker (Recomendado)

```powershell
# Copiar configuração
cp .env.example .env

# Iniciar stack completa
docker-compose up --build
```

## Passo 6: Rodar Localmente (Sem Docker)

### Pré-requisitos Locais

1. **PostgreSQL 16:**
   - Download: https://www.postgresql.org/download/windows/
   - OU Docker: `docker run -d -p 5432:5432 -e POSTGRES_PASSWORD=linkko postgres:16-alpine`

2. **Redis 7:**
   - Download Windows: https://github.com/microsoftarchive/redis/releases
   - OU Docker: `docker run -d -p 6379:6379 redis:7-alpine`

3. **Jaeger (Opcional - para traces):**
   - Docker: `docker run -d -p 16686:16686 -p 4317:4317 jaegertracing/all-in-one:1.54`

### Configurar .env

```bash
DATABASE_URL=postgres://linkko:linkko@localhost:5432/linkko?sslmode=disable
REDIS_URL=redis://localhost:6379
JWT_SECRET_CRM_V1=seu-secret-super-seguro-min-32-caracteres-aqui
JWT_PUBLIC_KEY_MCP_V1=-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----
JWT_ALLOWED_ISSUERS=linkko-crm-web,linkko-mcp-server
JWT_AUDIENCE=linkko-api-gateway
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
PORT=8080
```

### Executar

```powershell
# Migrations
go run ./cmd/linkko-api migrate

# Servidor
go run ./cmd/linkko-api serve
```

## Troubleshooting

### Go não é reconhecido após instalação

```powershell
# Fechar e reabrir o terminal PowerShell
# OU recarregar variáveis de ambiente:
$env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
```

### Erro: "go.mod: no such file"

```powershell
# Certifique-se de estar na pasta do projeto
cd G:\github-crm-projects\api-linkko
pwd  # Deve mostrar o caminho correto
```

### Erro ao compilar

```powershell
# Limpar cache
go clean -cache -modcache

# Re-download
go mod download
go mod tidy
```

### Portas em uso

```powershell
# Verificar o que está usando a porta 8080
netstat -ano | findstr :8080

# Matar processo (substitua PID)
taskkill /PID <numero> /F
```

## Próximos Passos

1. ✅ Go instalado
2. ✅ Dependências baixadas
3. ✅ Projeto compila
4. 🚀 Escolher: Docker (recomendado) ou Local
5. 🧪 Testar endpoints
6. 📊 Ver traces no Jaeger (http://localhost:16686)

## Links Úteis

- **Go Docs:** https://go.dev/doc/
- **Chi Router:** https://go-chi.io/
- **pgx:** https://github.com/jackc/pgx
- **OpenTelemetry Go:** https://opentelemetry.io/docs/languages/go/
