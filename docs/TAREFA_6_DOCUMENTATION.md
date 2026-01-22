# Tarefa 6 - Documentação Completa ✅

## Status: CONCLUÍDO

Atualização completa do `.env.example` e `README.md` para refletir o novo modelo de autenticação (JWT HS256 + S2S) com exemplos práticos de teste via Postman/Insomnia/cURL.

---

## 📝 Arquivos Atualizados

### 1. `.env.example`

**Mudanças principais:**

#### Novas variáveis JWT HS256:
```bash
JWT_HS256_SECRET=your-super-secret-key-min-32-chars-please-change-this-now
JWT_ISSUER=linkko-crm-web
JWT_AUDIENCE=linkko-api-gateway
JWT_CLOCK_SKEW_SECONDS=60
```

#### Novas variáveis S2S:
```bash
S2S_TOKEN_CRM=your-crm-service-token-here-min-32-chars-change-this
S2S_TOKEN_MCP=your-mcp-service-token-here-min-32-chars-change-this
```

#### Removidas/Deprecated:
- `JWT_SECRET_CRM_V1` → renomeado para `JWT_HS256_SECRET`
- `JWT_ALLOWED_ISSUERS` → substituído por `JWT_ISSUER` (single issuer)
- `JWT_PUBLIC_KEY_MCP_V1` → deprecated (comentado), usar S2S tokens

#### Melhorias:
- Organização por seções com separadores visuais
- Comentários explicativos para cada variável
- Instruções de geração de secrets (openssl)
- PORT atualizado de 3002 para 8080 (padrão)

---

### 2. `README.md`

**Mudanças principais:**

#### Nova seção: "🔐 Testando com Postman/Insomnia/cURL"

Localização: Após "Quick Start", antes de "Comandos CLI"

**Conteúdo:**

1. **Pré-requisitos**
   - Lista de variáveis necessárias no .env
   - Links para ferramentas (jwt.io)

2. **Opção 1: JWT HS256 (Frontend)**
   - Como gerar JWT de teste (jwt.io + Node.js script)
   - Exemplo cURL completo com token
   - Exemplos Postman/Insomnia
   - Respostas de sucesso e erro:
     - 200 OK (sucesso)
     - 401 INVALID_TOKEN
     - 401 TOKEN_EXPIRED
     - 403 WORKSPACE_MISMATCH

3. **Opção 2: S2S Authentication (Backend)**
   - Exemplo cURL com headers S2S
   - Respostas de sucesso e erro:
     - 201 Created (sucesso)
     - 401 INVALID_SIGNATURE
     - 400 INVALID_PARAMETER (headers ausentes)
     - 403 WORKSPACE_MISMATCH

4. **Regra Crítica: WorkspaceId Mismatch (IDOR Protection)**
   - Explicação detalhada da validação
   - Exemplos de ataque bloqueado
   - Fluxo completo de validação

5. **Outros Erros Comuns**
   - 400 INVALID_WORKSPACE_ID
   - 400 MISSING_PARAMETER
   - 401 MISSING_AUTHORIZATION
   - 429 RATE_LIMIT_EXCEEDED (com headers)

6. **Coleção Postman/Insomnia**
   - JSON completo de coleção Postman
   - Variáveis configuráveis (base_url, workspace_id, tokens)
   - Exemplos JWT e S2S prontos para importar

#### Seção "🔐 Segurança" - Reescrita completa

**Antes:**
- Multi-Issuer JWT (HS256 + RS256)
- Tabela com linkko-crm-web e linkko-mcp-server

**Depois:**
- **Autenticação Dual (JWT + S2S)**
- Tabela simplificada: JWT HS256 vs S2S Token
- Claims obrigatórios do JWT com exemplo JSON
- Headers obrigatórios do S2S
- Validações detalhadas para cada método

**IDOR Prevention expandido:**
- Fluxo completo de validação (com código)
- Exemplo de request bloqueado
- Explicação de por que é crítico (cross-tenant, IDOR attacks)

#### Seção "📦 Variáveis de Ambiente" - Atualizada

**Mudanças:**
- Organização por categorias (Database, Redis, JWT HS256, S2S, OTel, Server, Rate Limit)
- Adicionadas novas variáveis:
  - `JWT_HS256_SECRET`
  - `JWT_ISSUER`
  - `JWT_CLOCK_SKEW_SECONDS`
  - `S2S_TOKEN_CRM`
  - `S2S_TOKEN_MCP`
- Removidas variáveis antigas:
  - `JWT_SECRET_CRM_V1`
  - `JWT_PUBLIC_KEY_MCP_V1`
  - `JWT_ALLOWED_ISSUERS`

**Nova subseção:** "Gerando Secrets"
```bash
# JWT HS256 Secret
openssl rand -base64 32

# S2S Tokens
openssl rand -hex 32
```

#### Seção "🧪 Troubleshooting" - Expandida

**Nova categoria:** "Autenticação"

Erros cobertos:
1. **JWT validation failed - INVALID_TOKEN**
   - Causas possíveis (token malformado, secret incorreto, algoritmo errado)
   - Como resolver (verificar .env, testar em jwt.io)

2. **JWT validation failed - TOKEN_EXPIRED**
   - Como gerar novo token
   - Explicação de clock skew

3. **JWT validation failed - INVALID_ISSUER**
   - JWT `iss` deve ser `linkko-crm-web`

4. **JWT validation failed - INVALID_AUDIENCE**
   - JWT `aud` deve ser `linkko-api-gateway`

5. **S2S authentication failed - INVALID_SIGNATURE**
   - Como verificar tokens no .env
   - Comparação case-sensitive

6. **Workspace mismatch - 403 WORKSPACE_MISMATCH**
   - Exemplos corretos e incorretos para JWT
   - Exemplos corretos e incorretos para S2S
   - Headers devem coincidir com path

#### Seção "Quick Start" - Setup Inicial

**Melhorias:**
- Instruções passo-a-passo para gerar secrets
- Comandos openssl para cada variável
- Explicação de quais claims devem coincidir

#### Outras Atualizações

**Visão Geral:**
- "S2S Authentication" → "Dual Authentication: JWT HS256 + S2S tokens"
- Menção explícita a HTTP 403 na IDOR prevention

**Estrutura do Projeto:**
- Adicionado `internal/http/httperr/` (novo pacote)
- Atualizado `internal/auth/` (s2s.go em vez de resolver.go)
- Adicionado middleware observability.go

**Arquitetura - Fluxo de Autenticação:**
- Seção completamente reescrita
- Fluxo separado para JWT vs S2S
- Diagrama de validação com passo-a-passo
- Key Differences entre JWT e S2S

---

## 🎯 Principais Diferenças

### Modelo Antigo (Multi-Issuer JWT)

```
- JWT HS256 (linkko-crm-web) + JWT RS256 (linkko-mcp-server)
- KeyResolver dinâmico baseado em issuer
- JWKS para múltiplos issuers
- Variáveis: JWT_SECRET_CRM_V1, JWT_PUBLIC_KEY_MCP_V1
```

### Modelo Novo (JWT + S2S)

```
- JWT HS256 (frontend único) + S2S tokens (backend services)
- Validação simplificada: 1 secret JWT + 2 tokens S2S
- Sem JWKS, sem múltiplos issuers
- Variáveis: JWT_HS256_SECRET, S2S_TOKEN_CRM, S2S_TOKEN_MCP
```

**Vantagens:**
- ✅ Mais simples de configurar (3 variáveis vs 4+)
- ✅ Sem necessidade de gerar chaves RSA
- ✅ S2S mais performático que RS256 JWT
- ✅ Separação clara: JWT (users) vs S2S (services)
- ✅ Mais fácil de debugar (token comparison vs signature validation)

---

## 📋 Exemplos Práticos

### 1. Gerar JWT de Teste

**Node.js:**
```javascript
const jwt = require('jsonwebtoken');

const token = jwt.sign(
  {
    iss: 'linkko-crm-web',
    aud: 'linkko-api-gateway',
    workspace_id: 'my-workspace-123',
    actor_id: 'user-abc-456',
    exp: Math.floor(Date.now() / 1000) + (60 * 60) // 1 hour
  },
  process.env.JWT_HS256_SECRET,
  { algorithm: 'HS256' }
);

console.log(token);
```

**jwt.io:**
1. Acesse https://jwt.io
2. Algorithm: HS256
3. Payload:
   ```json
   {
     "iss": "linkko-crm-web",
     "aud": "linkko-api-gateway",
     "workspace_id": "my-workspace-123",
     "actor_id": "user-abc-456",
     "exp": 1737763200
   }
   ```
4. Secret: valor de `JWT_HS256_SECRET` do .env
5. Copiar token gerado

### 2. Testar com cURL - JWT

```bash
curl -X GET http://localhost:8080/api/v1/workspaces/my-workspace-123/contacts \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json"
```

### 3. Testar com cURL - S2S

```bash
# Ler token do .env
source .env

curl -X POST http://localhost:8080/api/v1/workspaces/my-workspace-123/tasks \
  -H "Authorization: Bearer $S2S_TOKEN_CRM" \
  -H "X-Workspace-Id: my-workspace-123" \
  -H "X-Actor-Id: service-crm" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Follow up with client",
    "status": "TODO",
    "priority": "HIGH"
  }'
```

### 4. Importar Coleção Postman

1. Copiar JSON da seção "Coleção Postman/Insomnia" do README
2. Salvar como `linkko-api.postman_collection.json`
3. Postman → Import → File → Selecionar arquivo
4. Atualizar variáveis:
   - `jwt_token`: Gerar com jwt.io
   - `s2s_token_crm`: Copiar de `.env`
   - `workspace_id`: Usar workspace válido

### 5. Testar IDOR Protection

**Cenário: Usuário tenta acessar workspace diferente**

```bash
# JWT com workspace_id: "workspace-A"
# Gerar token com:
# {
#   "workspace_id": "workspace-A",
#   ...
# }

# Tentar acessar workspace-B
curl -X GET http://localhost:8080/api/v1/workspaces/workspace-B/contacts \
  -H "Authorization: Bearer <JWT_com_workspace_A>"

# Resposta esperada: 403 Forbidden
# {
#   "ok": false,
#   "error": {
#     "code": "WORKSPACE_MISMATCH",
#     "message": "workspace access denied"
#   }
# }
```

---

## ✅ Checklist de Validação

Use este checklist para validar a documentação:

- [x] `.env.example` contém todas as variáveis necessárias
- [x] `.env.example` não contém segredos reais (apenas placeholders)
- [x] README explica como gerar JWT de teste
- [x] README explica como usar S2S tokens
- [x] README contém exemplos cURL para JWT e S2S
- [x] README explica regra de workspace mismatch (403)
- [x] README inclui todos os códigos de erro possíveis
- [x] README inclui coleção Postman/Insomnia pronta
- [x] Seção de troubleshooting cobre erros de autenticação
- [x] Variáveis de ambiente documentadas com descrições claras
- [x] Exemplos práticos testáveis (copy-paste ready)

---

## 🚀 Próximos Passos

1. **Testar Exemplos:**
   - [ ] Gerar JWT com jwt.io e testar endpoint
   - [ ] Testar S2S com cURL
   - [ ] Importar coleção Postman e validar requests

2. **Feedback do Time:**
   - [ ] Validar se documentação está clara para novos desenvolvedores
   - [ ] Verificar se exemplos funcionam em diferentes ambientes

3. **Melhorias Futuras:**
   - [ ] Adicionar exemplos em outras linguagens (Python, Go, Java)
   - [ ] Criar script de geração automática de JWT para testes
   - [ ] Adicionar vídeo tutorial de setup

---

## 📚 Referências

- RFC 7519 (JWT): https://datatracker.ietf.org/doc/html/rfc7519
- jwt.io Debugger: https://jwt.io
- Postman Documentation: https://learning.postman.com/docs/
- OpenSSL Commands: https://www.openssl.org/docs/

---

## 💡 Dicas

**Para desenvolvedores frontend:**
- Use JWT HS256 com claims `workspace_id` e `actor_id`
- Token deve ser renovado antes de expirar
- Workspace no path deve sempre coincidir com claim

**Para desenvolvedores backend:**
- Use S2S tokens para comunicação service-to-service
- Sempre envie headers `X-Workspace-Id` e `X-Actor-Id`
- Tokens devem ter mínimo 32 caracteres (segurança)

**Para testes:**
- Use jwt.io para gerar tokens rapidamente
- Importe coleção Postman para testes automatizados
- Configure variáveis de ambiente no Postman para facilitar switches

---

## 🎉 Resultado Final

Documentação completa e prática para:
- ✅ Configurar ambiente (.env.example detalhado)
- ✅ Entender autenticação (JWT vs S2S)
- ✅ Testar API (exemplos cURL, Postman, Insomnia)
- ✅ Debugar problemas (troubleshooting expandido)
- ✅ Prevenir erros comuns (IDOR, workspace mismatch)

**README.md atualizado:** ~500 linhas → ~700 linhas (+200)
**.env.example atualizado:** ~25 linhas → ~70 linhas (+45)

**Tempo estimado para setup:** 5-10 minutos (vs 20-30 min antes)
