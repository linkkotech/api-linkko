package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"linkko-api/internal/observability/logger"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createJWTTestToken creates a JWT for testing with flexible params
func createJWTTestToken(secret string, claims *CustomClaims, issuer, audience string, expiresAt time.Time) string {
	claims.RegisteredClaims = jwt.RegisteredClaims{
		Issuer:    issuer,
		Audience:  jwt.ClaimStrings{audience},
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

func TestIsJWTToken(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected bool
	}{
		{
			name:     "Valid JWT format",
			token:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
			expected: true,
		},
		{
			name:     "S2S token (random string)",
			token:    "s2s-fixed-token-crm-12345",
			expected: false,
		},
		{
			name:     "S2S token (starts with eyJ but no dots)",
			token:    "eyJnotajwttoken",
			expected: false,
		},
		{
			name:     "Empty token",
			token:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isJWTToken(tt.token)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestS2STokenStore(t *testing.T) {
	t.Run("ValidToken", func(t *testing.T) {
		store := NewS2STokenStore()
		store.RegisterToken("test-token-crm", "crm-web")

		client, ok := store.ValidateToken("test-token-crm")
		assert.True(t, ok)
		assert.Equal(t, "crm-web", client)
	})

	t.Run("InvalidToken", func(t *testing.T) {
		store := NewS2STokenStore()
		store.RegisterToken("test-token-crm", "crm-web")

		client, ok := store.ValidateToken("wrong-token")
		assert.False(t, ok)
		assert.Empty(t, client)
	})

	t.Run("EmptyToken", func(t *testing.T) {
		store := NewS2STokenStore()
		store.RegisterToken("", "crm-web") // Should not register

		client, ok := store.ValidateToken("")
		assert.False(t, ok)
		assert.Empty(t, client)
	})

	t.Run("MultipleTokens", func(t *testing.T) {
		store := NewS2STokenStore()
		store.RegisterToken("token-crm", "crm-web")
		store.RegisterToken("token-mcp", "mcp")

		client1, ok1 := store.ValidateToken("token-crm")
		assert.True(t, ok1)
		assert.Equal(t, "crm-web", client1)

		client2, ok2 := store.ValidateToken("token-mcp")
		assert.True(t, ok2)
		assert.Equal(t, "mcp", client2)
	})
}

func TestValidateS2SHeaders(t *testing.T) {
	t.Run("BothHeadersPresent", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Workspace-Id", "ws-123")
		req.Header.Set("X-Actor-Id", "user-456")

		workspaceID, actorID, err := validateS2SHeaders(req)
		require.NoError(t, err)
		assert.Equal(t, "ws-123", workspaceID)
		assert.Equal(t, "user-456", actorID)
	})

	t.Run("NoHeadersPresent", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)

		workspaceID, actorID, err := validateS2SHeaders(req)
		require.NoError(t, err)
		assert.Empty(t, workspaceID)
		assert.Empty(t, actorID)
	})

	t.Run("OnlyWorkspaceId", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Workspace-Id", "ws-123")

		workspaceID, actorID, err := validateS2SHeaders(req)
		require.NoError(t, err)
		assert.Equal(t, "ws-123", workspaceID)
		assert.Empty(t, actorID)
	})

	t.Run("OnlyActorId", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Actor-Id", "user-456")

		workspaceID, actorID, err := validateS2SHeaders(req)
		require.NoError(t, err)
		assert.Empty(t, workspaceID)
		assert.Equal(t, "user-456", actorID)
	})

	t.Run("EmptyWorkspaceId", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Workspace-Id", "")
		req.Header.Set("X-Actor-Id", "user-456")

		workspaceID, _, err := validateS2SHeaders(req)
		require.NoError(t, err) // Empty is OK (treated as not present)
		assert.Empty(t, workspaceID)
	})

	t.Run("WhitespaceWorkspaceId", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Workspace-Id", "   ")

		_, _, err := validateS2SHeaders(req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "X-Workspace-Id must be non-empty")
	})

	t.Run("WhitespaceActorId", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Actor-Id", "   ")

		_, _, err := validateS2SHeaders(req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "X-Actor-Id must be non-empty")
	})
}

func TestAuthMiddleware_S2S_Valid(t *testing.T) {
	// Setup
	log, _ := logger.New("test", "info")
	ctx := logger.SetLoggerInContext(context.Background(), log)

	store := NewS2STokenStore()
	store.RegisterToken("test-s2s-token-crm", "crm-web")

	resolver := NewKeyResolver([]string{}, []string{})
	middleware := AuthMiddleware(resolver, store)

	// Create request with S2S token
	req := httptest.NewRequest("GET", "/test", nil)
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer test-s2s-token-crm")
	req.Header.Set("X-Workspace-Id", "ws-123")
	req.Header.Set("X-Actor-Id", "service-456")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Handler to verify context
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authCtx, ok := GetAuthContext(r.Context())
		require.True(t, ok)
		assert.Equal(t, "ws-123", authCtx.WorkspaceID)
		assert.Equal(t, "service-456", authCtx.ActorID)
		assert.Equal(t, "service", authCtx.ActorType)
		assert.Equal(t, "s2s", authCtx.AuthMethod)
		assert.Equal(t, "crm-web", authCtx.Client)
		w.WriteHeader(http.StatusOK)
	})

	// Test
	middleware(handler).ServeHTTP(rr, req)

	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAuthMiddleware_S2S_NoHeaders(t *testing.T) {
	// Setup
	log, _ := logger.New("test", "info")
	ctx := logger.SetLoggerInContext(context.Background(), log)

	store := NewS2STokenStore()
	store.RegisterToken("test-s2s-token-mcp", "mcp")

	resolver := NewKeyResolver([]string{}, []string{})
	middleware := AuthMiddleware(resolver, store)

	// Create request without optional headers
	req := httptest.NewRequest("GET", "/test", nil)
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer test-s2s-token-mcp")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Handler to verify context
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authCtx, ok := GetAuthContext(r.Context())
		require.True(t, ok)
		assert.Empty(t, authCtx.WorkspaceID) // No header provided
		assert.Empty(t, authCtx.ActorID)     // No header provided
		assert.Equal(t, "service", authCtx.ActorType)
		assert.Equal(t, "s2s", authCtx.AuthMethod)
		assert.Equal(t, "mcp", authCtx.Client)
		w.WriteHeader(http.StatusOK)
	})

	// Test
	middleware(handler).ServeHTTP(rr, req)

	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAuthMiddleware_S2S_InvalidToken(t *testing.T) {
	// Setup
	log, _ := logger.New("test", "info")
	ctx := logger.SetLoggerInContext(context.Background(), log)

	store := NewS2STokenStore()
	store.RegisterToken("valid-token", "crm-web")

	resolver := NewKeyResolver([]string{}, []string{})
	middleware := AuthMiddleware(resolver, store)

	// Create request with invalid S2S token
	req := httptest.NewRequest("GET", "/test", nil)
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer invalid-token")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Handler should not be called
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("Handler should not be called for invalid token")
	})

	// Test
	middleware(handler).ServeHTTP(rr, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthMiddleware_JWT_vs_S2S_Precedence(t *testing.T) {
	// Setup
	log, _ := logger.New("test", "info")
	ctx := logger.SetLoggerInContext(context.Background(), log)

	keyStore := NewKeyStore()
	keyStore.LoadHS256Key(testIssuer, "v1", []byte(testSecret))

	validator := NewHS256Validator(keyStore, testIssuer, 60)
	resolver := NewKeyResolver([]string{testIssuer}, []string{testAudience})
	resolver.RegisterValidator(testIssuer, validator)

	store := NewS2STokenStore()
	store.RegisterToken("s2s-token", "crm-web")

	middleware := AuthMiddleware(resolver, store)

	t.Run("JWT token is validated as JWT", func(t *testing.T) {
		// Create valid JWT token
		claims := &CustomClaims{
			WorkspaceID: "ws-jwt",
			ActorID:     "user-jwt",
		}
		jwtToken := createJWTTestToken(testSecret, claims, testIssuer, testAudience, time.Now().Add(1*time.Hour))

		req := httptest.NewRequest("GET", "/test", nil)
		req = req.WithContext(ctx)
		req.Header.Set("Authorization", "Bearer "+jwtToken)

		rr := httptest.NewRecorder()

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCtx, ok := GetAuthContext(r.Context())
			require.True(t, ok)
			assert.Equal(t, "jwt", authCtx.AuthMethod)
			assert.Equal(t, "user", authCtx.ActorType)
			assert.Equal(t, "ws-jwt", authCtx.WorkspaceID)
			w.WriteHeader(http.StatusOK)
		})

		middleware(handler).ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("S2S token is validated as S2S", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req = req.WithContext(ctx)
		req.Header.Set("Authorization", "Bearer s2s-token")
		req.Header.Set("X-Workspace-Id", "ws-s2s")

		rr := httptest.NewRecorder()

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCtx, ok := GetAuthContext(r.Context())
			require.True(t, ok)
			assert.Equal(t, "s2s", authCtx.AuthMethod)
			assert.Equal(t, "service", authCtx.ActorType)
			assert.Equal(t, "ws-s2s", authCtx.WorkspaceID)
			assert.Equal(t, "crm-web", authCtx.Client)
			w.WriteHeader(http.StatusOK)
		})

		middleware(handler).ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	})
}
