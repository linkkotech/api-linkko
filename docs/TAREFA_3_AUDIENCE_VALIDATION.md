# Tarefa 3: Validar Audience Exatamente como o CRM Emite

## ✅ Objetivo
Validar o audience do JWT via `JWT_AUDIENCE`, exigindo match exato com o `aud` emitido pelo CRM (linkko-api-gateway).

## 📝 Status da Implementação

### ✅ Validação Já Existente
A validação de audience **já estava implementada** no resolver desde o início. Esta tarefa adiciona **testes abrangentes** para garantir que a validação funciona corretamente em todos os cenários.

**Código existente (`internal/auth/resolver.go`):**
```go
// Verify audience
if !kr.validAudience(claims.Audience) {
    return nil, NewAuthError(
        AuthFailureInvalidAudience, 
        fmt.Sprintf("invalid audience: %v", claims.Audience), 
        nil,
    )
}

// validAudience checks if any audience claim matches allowed audiences
func (kr *KeyResolver) validAudience(audiences []string) bool {
    for _, aud := range audiences {
        for _, allowed := range kr.allowedAudiences {
            if aud == allowed {
                return true
            }
        }
    }
    return false
}
```

### 🔒 Comportamento de Validação

#### Match Exato
- ✅ **Case-sensitive**: "linkko-api-gateway" ≠ "Linkko-Api-Gateway"
- ✅ **Exact string**: "linkko-api-gateway" ≠ "linkko-api-gateway-v2"
- ✅ **No partial match**: não aceita substrings

#### Múltiplos Audiences
- ✅ Token pode ter múltiplos audiences: `["other-service", "linkko-api-gateway"]`
- ✅ Validação passa se **pelo menos 1** audience bater
- ✅ Resolver pode aceitar múltiplos audiences: `["linkko-api-gateway", "linkko-admin-api"]`

#### Erro 401
- ✅ Retorna `AuthFailureInvalidAudience` (reason: "invalid_audience")
- ✅ Mapeia para HTTP 401 Unauthorized via `httperr.ErrCodeInvalidAudience`
- ✅ Mensagem clara: "invalid audience: [list-of-audiences]"

## 🧪 Testes Implementados

### 1. TestKeyResolver_AudienceValidation (6 subtestes)
**Cenário:** Resolver configurado com `JWT_AUDIENCE=linkko-api-gateway`

| Subteste | Audience Token | Resultado | Descrição |
|----------|---------------|-----------|-----------|
| `exact_match` | `["linkko-api-gateway"]` | ✅ Aceito | Match exato do audience |
| `wrong_audience` | `["linkko-api-gateway-wrong"]` | ❌ 401 | Audience diferente rejeitado |
| `empty_audience` | `[]` | ❌ 401 | Token sem audience rejeitado |
| `multiple_audiences_with_match` | `["other-service", "linkko-api-gateway"]` | ✅ Aceito | Um dos audiences bate |
| `multiple_audiences_no_match` | `["other-service", "another-service"]` | ❌ 401 | Nenhum audience bate |
| `case_sensitive_mismatch` | `["Linkko-Api-Gateway"]` | ❌ 401 | Validação case-sensitive |

**Código:**
```go
func TestKeyResolver_AudienceValidation(t *testing.T) {
    // Setup resolver with single allowed audience
    resolver := NewKeyResolver(
        []string{testIssuer}, 
        []string{"linkko-api-gateway"},
    )
    
    tests := []struct {
        name        string
        audience    []string
        shouldPass  bool
        description string
    }{
        {
            name:       "exact_match",
            audience:   []string{"linkko-api-gateway"},
            shouldPass: true,
        },
        {
            name:       "wrong_audience",
            audience:   []string{"linkko-api-gateway-wrong"},
            shouldPass: false,
        },
        // ... mais 4 testes
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Create token with specified audience
            // Validate
            // Assert expected result
        })
    }
}
```

### 2. TestKeyResolver_MultipleAllowedAudiences (4 subtestes)
**Cenário:** Resolver configurado com múltiplos audiences permitidos

```go
resolver := NewKeyResolver(
    []string{testIssuer},
    []string{"linkko-api-gateway", "linkko-admin-api", "linkko-mobile-api"},
)
```

| Subteste | Audience Token | Resultado | Descrição |
|----------|---------------|-----------|-----------|
| `first_allowed` | `["linkko-api-gateway"]` | ✅ Aceito | Primeiro audience da lista |
| `second_allowed` | `["linkko-admin-api"]` | ✅ Aceito | Segundo audience da lista |
| `third_allowed` | `["linkko-mobile-api"]` | ✅ Aceito | Terceiro audience da lista |
| `not_in_allowed_list` | `["linkko-unknown-api"]` | ❌ 401 | Audience não permitido |

### 3. TestKeyResolver_InvalidAudience (teste existente)
**Cenário:** Teste original para validação básica de audience inválido

```go
func TestKeyResolver_InvalidAudience(t *testing.T) {
    // Setup
    resolver := NewKeyResolver([]string{testIssuer}, []string{testAudience})
    
    // Create token with wrong audience
    claims.Audience = jwt.ClaimStrings{"wrong-audience"}
    
    // Assert
    authErr, ok := IsAuthError(err)
    require.True(t, ok)
    assert.Equal(t, AuthFailureInvalidAudience, authErr.Reason)
}
```

## 📊 Resultados dos Testes

### Novos Testes (10 subtestes)
```bash
$ go test -v ./internal/auth -run "TestKeyResolver_(AudienceValidation|MultipleAllowedAudiences)"

=== RUN   TestKeyResolver_AudienceValidation
=== RUN   TestKeyResolver_AudienceValidation/exact_match
--- PASS: TestKeyResolver_AudienceValidation/exact_match (0.00s)
=== RUN   TestKeyResolver_AudienceValidation/wrong_audience
--- PASS: TestKeyResolver_AudienceValidation/wrong_audience (0.00s)
=== RUN   TestKeyResolver_AudienceValidation/empty_audience
--- PASS: TestKeyResolver_AudienceValidation/empty_audience (0.00s)
=== RUN   TestKeyResolver_AudienceValidation/multiple_audiences_with_match
--- PASS: TestKeyResolver_AudienceValidation/multiple_audiences_with_match (0.00s)
=== RUN   TestKeyResolver_AudienceValidation/multiple_audiences_no_match
--- PASS: TestKeyResolver_AudienceValidation/multiple_audiences_no_match (0.00s)
=== RUN   TestKeyResolver_AudienceValidation/case_sensitive_mismatch
--- PASS: TestKeyResolver_AudienceValidation/case_sensitive_mismatch (0.00s)
--- PASS: TestKeyResolver_AudienceValidation (0.00s)

=== RUN   TestKeyResolver_MultipleAllowedAudiences
=== RUN   TestKeyResolver_MultipleAllowedAudiences/first_allowed
--- PASS: TestKeyResolver_MultipleAllowedAudiences/first_allowed (0.00s)
=== RUN   TestKeyResolver_MultipleAllowedAudiences/second_allowed
--- PASS: TestKeyResolver_MultipleAllowedAudiences/second_allowed (0.00s)
=== RUN   TestKeyResolver_MultipleAllowedAudiences/third_allowed
--- PASS: TestKeyResolver_MultipleAllowedAudiences/third_allowed (0.00s)
=== RUN   TestKeyResolver_MultipleAllowedAudiences/not_in_allowed_list
--- PASS: TestKeyResolver_MultipleAllowedAudiences/not_in_allowed_list (0.00s)
--- PASS: TestKeyResolver_MultipleAllowedAudiences (0.00s)

PASS
ok      linkko-api/internal/auth        0.040s
```

### Todos Auth Tests (34 testes + 10 novos subtestes)
```bash
$ go test -v ./internal/auth
PASS
ok      linkko-api/internal/auth        0.065s
```

## 🔒 Validação de Audience - Fluxo Completo

### 1. Configuração no .env
```dotenv
# JWT Audience - identifies who the token is intended for
# REQUIRED: Must match exactly the "aud" claim in JWT tokens
# CRM frontend should emit tokens with aud="linkko-api-gateway"
# Validation is case-sensitive and requires exact match
JWT_AUDIENCE=linkko-api-gateway
```

### 2. Startup do Servidor
```go
// cmd/linkko-api/serve.go
resolver := auth.NewKeyResolver(allowedIssuers, []string{cfg.JWTAudience})
//                                                         ^^^^^^^^^^^^^^^^
//                                                         "linkko-api-gateway"
```

### 3. Token Emitido pelo CRM
```json
{
  "iss": "linkko-crm-web",
  "aud": "linkko-api-gateway",    // ✅ Deve bater exatamente
  "workspaceId": "ws-12345",
  "actorId": "user-67890",
  "exp": 1737568800,
  "iat": 1737565200
}
```

### 4. Validação no Resolver
```go
// internal/auth/resolver.go
func (kr *KeyResolver) Resolve(ctx context.Context, tokenString string) (*CustomClaims, error) {
    // ... (validação de issuer, assinatura, etc.)
    
    // Verify audience
    if !kr.validAudience(claims.Audience) {
        return nil, NewAuthError(
            AuthFailureInvalidAudience, 
            fmt.Sprintf("invalid audience: %v", claims.Audience), 
            nil,
        )
    }
    
    return claims, nil
}
```

### 5. Resposta HTTP (se audience inválido)
```http
HTTP/1.1 401 Unauthorized
Content-Type: application/json

{
  "error": "INVALID_AUDIENCE",
  "message": "invalid audience: [wrong-audience]"
}
```

## 📚 Exemplos de Uso

### Cenário 1: Single Audience (Padrão)
```dotenv
JWT_AUDIENCE=linkko-api-gateway
```

**Token aceito:**
```json
{
  "iss": "linkko-crm-web",
  "aud": "linkko-api-gateway"  // ✅ Match exato
}
```

**Token rejeitado:**
```json
{
  "iss": "linkko-crm-web",
  "aud": "linkko-api-gateway-v2"  // ❌ Diferente
}
```

### Cenário 2: Token com Múltiplos Audiences
```dotenv
JWT_AUDIENCE=linkko-api-gateway
```

**Token aceito:**
```json
{
  "iss": "linkko-crm-web",
  "aud": ["other-service", "linkko-api-gateway"]  // ✅ Um deles bate
}
```

**Token rejeitado:**
```json
{
  "iss": "linkko-crm-web",
  "aud": ["service-a", "service-b"]  // ❌ Nenhum bate
}
```

### Cenário 3: Múltiplos Audiences Permitidos (Multi-API)
```dotenv
# Aceitar tokens destinados a qualquer uma das 3 APIs
JWT_AUDIENCE=linkko-api-gateway,linkko-admin-api,linkko-mobile-api
```

**Nota:** Atualmente `JWT_AUDIENCE` aceita apenas **um valor**. Para múltiplos audiences, seria necessário:
1. Modificar config para aceitar CSV: `JWT_AUDIENCE` → `JWT_ALLOWED_AUDIENCES`
2. Parse similar ao `JWT_ALLOWED_ISSUERS`

**Workaround atual:** Tokens podem ter múltiplos audiences, mas resolver aceita apenas 1 configurado.

## 🔄 Integração com Erros HTTP

### Mapeamento de Erro
```go
// internal/auth/s2s.go
func mapAuthErrorToHTTPError(reason AuthFailureReason) string {
    switch reason {
    case AuthFailureInvalidAudience:
        return httperr.ErrCodeInvalidAudience  // "INVALID_AUDIENCE"
    // ... outros casos
    }
}
```

### Resposta JSON Padrão
```json
{
  "error": "INVALID_AUDIENCE",
  "message": "invalid audience: [wrong-audience]"
}
```

## ✅ Requisitos Atendidos

- [x] Validação de audience via `JWT_AUDIENCE` (já implementada)
- [x] Retorna 401 com reason `invalid_audience` ✅
- [x] Teste com audience válido (6 subtestes) ✅
- [x] Teste com audience inválido (5 subtestes) ✅
- [x] Match exato (case-sensitive) ✅
- [x] Suporte para múltiplos audiences no token ✅
- [x] Validação robusta com testes abrangentes ✅

## 📊 Resumo de Arquivos Alterados

```
internal/auth/resolver_test.go     | +160 -8   (2 novos testes, 10 subtestes)
.env.example                       | +3   -1   (documentação audience)
docs/TAREFA_3_AUDIENCE_VALIDATION.md | +XXX -0   (documentação completa)
```

## 🎯 Próximos Passos

1. **Configurar .env de produção:**
   ```dotenv
   JWT_AUDIENCE=linkko-api-gateway
   ```

2. **CRM Frontend - Emitir tokens corretos:**
   ```javascript
   // Garantir que tokens incluem audience correto
   const token = await jwt.sign(payload, secret, {
       issuer: 'linkko-crm-web',
       audience: 'linkko-api-gateway',  // ✅ Match exato
       expiresIn: '1h'
   });
   ```

3. **Validar integração:**
   - Testar tokens do CRM com `aud=linkko-api-gateway` → ✅ Aceito
   - Testar tokens com audience errado → ❌ 401 INVALID_AUDIENCE

4. **(Opcional) Suportar múltiplos audiences permitidos:**
   - Criar variável `JWT_ALLOWED_AUDIENCES` (CSV)
   - Parse similar ao `JWT_ALLOWED_ISSUERS`
   - Atualizar resolver para aceitar lista

## 🔍 Observações

1. **Validação já existia** - Esta tarefa focou em **testes abrangentes**
2. **Case-sensitive** - "linkko-api-gateway" ≠ "Linkko-Api-Gateway"
3. **Exact match** - Não aceita substrings ou padrões
4. **Múltiplos audiences no token** - Aceito se pelo menos 1 bater
5. **Single audience no config** - Atualmente aceita apenas 1 valor
