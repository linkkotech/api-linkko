package auth

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyResolver_ValidToken(t *testing.T) {
	// Setup
	keyStore := NewKeyStore()
	keyStore.LoadHS256Key(testIssuer, "v1", []byte(testSecret))

	validator := NewHS256Validator(keyStore, testIssuer, 60*time.Second)
	resolver := NewKeyResolver([]string{testIssuer}, []string{testAudience})
	resolver.RegisterValidator(testIssuer, validator)

	// Create valid token
	claims := &CustomClaims{
		WorkspaceID: "ws-12345",
		ActorID:     "user-67890",
	}
	token := createTestToken(testSecret, claims, time.Now().Add(1*time.Hour))

	// Test
	ctx := context.Background()
	result, err := resolver.Resolve(ctx, token)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "ws-12345", result.WorkspaceID)
	assert.Equal(t, "user-67890", result.ActorID)
	assert.Equal(t, testIssuer, result.Issuer)
}

func TestKeyResolver_InvalidIssuer(t *testing.T) {
	// Setup
	keyStore := NewKeyStore()
	keyStore.LoadHS256Key(testIssuer, "v1", []byte(testSecret))

	validator := NewHS256Validator(keyStore, testIssuer, 60*time.Second)
	resolver := NewKeyResolver([]string{testIssuer}, []string{testAudience})
	resolver.RegisterValidator(testIssuer, validator)

	// Create token with different issuer
	claims := &CustomClaims{
		WorkspaceID: "ws-12345",
		ActorID:     "user-67890",
	}
	claims.RegisteredClaims = jwt.RegisteredClaims{
		Issuer:    "unauthorized-issuer",
		Audience:  jwt.ClaimStrings{testAudience},
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(testSecret))

	// Test
	ctx := context.Background()
	result, err := resolver.Resolve(ctx, tokenString)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)

	authErr, ok := IsAuthError(err)
	require.True(t, ok)
	assert.Equal(t, AuthFailureInvalidIssuer, authErr.Reason)
}

func TestKeyResolver_InvalidAudience(t *testing.T) {
	// Setup
	keyStore := NewKeyStore()
	keyStore.LoadHS256Key(testIssuer, "v1", []byte(testSecret))

	validator := NewHS256Validator(keyStore, testIssuer, 60*time.Second)
	resolver := NewKeyResolver([]string{testIssuer}, []string{testAudience})
	resolver.RegisterValidator(testIssuer, validator)

	// Create token with wrong audience
	claims := &CustomClaims{
		WorkspaceID: "ws-12345",
		ActorID:     "user-67890",
	}
	claims.RegisteredClaims = jwt.RegisteredClaims{
		Issuer:    testIssuer,
		Audience:  jwt.ClaimStrings{"wrong-audience"},
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(testSecret))

	// Test
	ctx := context.Background()
	result, err := resolver.Resolve(ctx, tokenString)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)

	authErr, ok := IsAuthError(err)
	require.True(t, ok)
	assert.Equal(t, AuthFailureInvalidAudience, authErr.Reason)
}

func TestKeyResolver_NoValidatorForIssuer(t *testing.T) {
	// Setup
	keyStore := NewKeyStore()
	keyStore.LoadHS256Key(testIssuer, "v1", []byte(testSecret))

	// Create resolver without registering validator
	resolver := NewKeyResolver([]string{testIssuer}, []string{testAudience})
	// Note: not registering validator

	// Create valid token
	claims := &CustomClaims{
		WorkspaceID: "ws-12345",
		ActorID:     "user-67890",
	}
	token := createTestToken(testSecret, claims, time.Now().Add(1*time.Hour))

	// Test
	ctx := context.Background()
	result, err := resolver.Resolve(ctx, token)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)

	authErr, ok := IsAuthError(err)
	require.True(t, ok)
	assert.Equal(t, AuthFailureInvalidIssuer, authErr.Reason)
	assert.Contains(t, authErr.Message, "no validator found")
}

func TestKeyResolver_MalformedToken(t *testing.T) {
	// Setup
	resolver := NewKeyResolver([]string{testIssuer}, []string{testAudience})

	// Test with malformed token
	ctx := context.Background()
	result, err := resolver.Resolve(ctx, "malformed-token")

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)

	authErr, ok := IsAuthError(err)
	require.True(t, ok)
	assert.Equal(t, AuthFailureUnknown, authErr.Reason)
}

func TestKeyResolver_EmptyKidFallback(t *testing.T) {
	// Setup
	keyStore := NewKeyStore()
	keyStore.LoadHS256Key(testIssuer, "v1", []byte(testSecret))

	validator := NewHS256Validator(keyStore, testIssuer, 60*time.Second)
	resolver := NewKeyResolver([]string{testIssuer}, []string{testAudience})
	resolver.RegisterValidator(testIssuer, validator)

	// Create token without kid in header (will fallback to v1)
	claims := &CustomClaims{
		WorkspaceID: "ws-12345",
		ActorID:     "user-67890",
	}
	token := createTestToken(testSecret, claims, time.Now().Add(1*time.Hour))

	// Test
	ctx := context.Background()
	result, err := resolver.Resolve(ctx, token)

	// Assert - should work with fallback
	require.NoError(t, err)
	assert.Equal(t, "ws-12345", result.WorkspaceID)
	assert.Equal(t, "user-67890", result.ActorID)
}

// TestKeyResolver_MultipleIssuers validates that the resolver can handle multiple allowed issuers
func TestKeyResolver_MultipleIssuers(t *testing.T) {
	// Setup
	keyStore := NewKeyStore()
	keyStore.LoadHS256Key("linkko-crm-web", "v1", []byte(testSecret))
	keyStore.LoadHS256Key("linkko-admin-portal", "v1", []byte(testSecret))

	validator1 := NewHS256Validator(keyStore, "linkko-crm-web", 60*time.Second)
	validator2 := NewHS256Validator(keyStore, "linkko-admin-portal", 60*time.Second)

	// Create resolver with multiple allowed issuers
	resolver := NewKeyResolver([]string{"linkko-crm-web", "linkko-admin-portal"}, []string{testAudience})
	resolver.RegisterValidator("linkko-crm-web", validator1)
	resolver.RegisterValidator("linkko-admin-portal", validator2)

	// Test token from first issuer
	claims1 := &CustomClaims{
		WorkspaceID: "ws-crm",
		ActorID:     "user-crm",
	}
	claims1.RegisteredClaims = jwt.RegisteredClaims{
		Issuer:    "linkko-crm-web",
		Audience:  jwt.ClaimStrings{testAudience},
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token1 := jwt.NewWithClaims(jwt.SigningMethodHS256, claims1)
	tokenString1, _ := token1.SignedString([]byte(testSecret))

	ctx := context.Background()
	result1, err := resolver.Resolve(ctx, tokenString1)
	require.NoError(t, err)
	assert.Equal(t, "linkko-crm-web", result1.Issuer)
	assert.Equal(t, "ws-crm", result1.WorkspaceID)

	// Test token from second issuer
	claims2 := &CustomClaims{
		WorkspaceID: "ws-admin",
		ActorID:     "user-admin",
	}
	claims2.RegisteredClaims = jwt.RegisteredClaims{
		Issuer:    "linkko-admin-portal",
		Audience:  jwt.ClaimStrings{testAudience},
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token2 := jwt.NewWithClaims(jwt.SigningMethodHS256, claims2)
	tokenString2, _ := token2.SignedString([]byte(testSecret))

	result2, err := resolver.Resolve(ctx, tokenString2)
	require.NoError(t, err)
	assert.Equal(t, "linkko-admin-portal", result2.Issuer)
	assert.Equal(t, "ws-admin", result2.WorkspaceID)
}

// TestKeyResolver_IssuerNotInAllowlist validates that tokens from non-allowed issuers are rejected
func TestKeyResolver_IssuerNotInAllowlist(t *testing.T) {
	// Setup: only allow "linkko-crm-web"
	keyStore := NewKeyStore()
	keyStore.LoadHS256Key("linkko-crm-web", "v1", []byte(testSecret))

	validator := NewHS256Validator(keyStore, "linkko-crm-web", 60*time.Second)
	resolver := NewKeyResolver([]string{"linkko-crm-web"}, []string{testAudience})
	resolver.RegisterValidator("linkko-crm-web", validator)

	// Create token from unauthorized issuer
	claims := &CustomClaims{
		WorkspaceID: "ws-12345",
		ActorID:     "user-67890",
	}
	claims.RegisteredClaims = jwt.RegisteredClaims{
		Issuer:    "unauthorized-issuer",
		Audience:  jwt.ClaimStrings{testAudience},
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(testSecret))

	// Test
	ctx := context.Background()
	result, err := resolver.Resolve(ctx, tokenString)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)

	authErr, ok := IsAuthError(err)
	require.True(t, ok)
	assert.Equal(t, AuthFailureInvalidIssuer, authErr.Reason)
	assert.Contains(t, authErr.Message, "issuer not allowed")
	assert.Contains(t, authErr.Message, "unauthorized-issuer")
}

// TestKeyResolver_AudienceValidation tests exact audience matching
func TestKeyResolver_AudienceValidation(t *testing.T) {
	// Setup
	keyStore := NewKeyStore()
	keyStore.LoadHS256Key(testIssuer, "v1", []byte(testSecret))

	validator := NewHS256Validator(keyStore, testIssuer, 60*time.Second)
	// Resolver expects exactly "linkko-api-gateway"
	resolver := NewKeyResolver([]string{testIssuer}, []string{"linkko-api-gateway"})
	resolver.RegisterValidator(testIssuer, validator)

	tests := []struct {
		name        string
		audience    []string
		shouldPass  bool
		description string
	}{
		{
			name:        "exact_match",
			audience:    []string{"linkko-api-gateway"},
			shouldPass:  true,
			description: "Token with exact audience should be accepted",
		},
		{
			name:        "wrong_audience",
			audience:    []string{"linkko-api-gateway-wrong"},
			shouldPass:  false,
			description: "Token with wrong audience should be rejected",
		},
		{
			name:        "empty_audience",
			audience:    []string{},
			shouldPass:  false,
			description: "Token with empty audience should be rejected",
		},
		{
			name:        "multiple_audiences_with_match",
			audience:    []string{"other-service", "linkko-api-gateway"},
			shouldPass:  true,
			description: "Token with multiple audiences (one matching) should be accepted",
		},
		{
			name:        "multiple_audiences_no_match",
			audience:    []string{"other-service", "another-service"},
			shouldPass:  false,
			description: "Token with multiple audiences (none matching) should be rejected",
		},
		{
			name:        "case_sensitive_mismatch",
			audience:    []string{"Linkko-Api-Gateway"}, // Different case
			shouldPass:  false,
			description: "Audience match is case-sensitive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create token with specified audience
			claims := &CustomClaims{
				WorkspaceID: "ws-12345",
				ActorID:     "user-67890",
			}
			claims.RegisteredClaims = jwt.RegisteredClaims{
				Issuer:    testIssuer,
				Audience:  jwt.ClaimStrings(tt.audience),
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			}

			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			tokenString, err := token.SignedString([]byte(testSecret))
			require.NoError(t, err)

			// Test
			ctx := context.Background()
			result, err := resolver.Resolve(ctx, tokenString)

			// Assert
			if tt.shouldPass {
				require.NoError(t, err, tt.description)
				assert.NotNil(t, result)
				assert.Equal(t, "ws-12345", result.WorkspaceID)
				assert.Equal(t, "user-67890", result.ActorID)
			} else {
				require.Error(t, err, tt.description)
				assert.Nil(t, result)

				authErr, ok := IsAuthError(err)
				require.True(t, ok)
				assert.Equal(t, AuthFailureInvalidAudience, authErr.Reason)
			}
		})
	}
}

// TestKeyResolver_MultipleAllowedAudiences tests resolver with multiple allowed audiences
func TestKeyResolver_MultipleAllowedAudiences(t *testing.T) {
	// Setup
	keyStore := NewKeyStore()
	keyStore.LoadHS256Key(testIssuer, "v1", []byte(testSecret))

	validator := NewHS256Validator(keyStore, testIssuer, 60*time.Second)
	// Resolver allows multiple audiences
	resolver := NewKeyResolver(
		[]string{testIssuer},
		[]string{"linkko-api-gateway", "linkko-admin-api", "linkko-mobile-api"},
	)
	resolver.RegisterValidator(testIssuer, validator)

	tests := []struct {
		name       string
		audience   []string
		shouldPass bool
	}{
		{
			name:       "first_allowed",
			audience:   []string{"linkko-api-gateway"},
			shouldPass: true,
		},
		{
			name:       "second_allowed",
			audience:   []string{"linkko-admin-api"},
			shouldPass: true,
		},
		{
			name:       "third_allowed",
			audience:   []string{"linkko-mobile-api"},
			shouldPass: true,
		},
		{
			name:       "not_in_allowed_list",
			audience:   []string{"linkko-unknown-api"},
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := &CustomClaims{
				WorkspaceID: "ws-test",
				ActorID:     "user-test",
			}
			claims.RegisteredClaims = jwt.RegisteredClaims{
				Issuer:    testIssuer,
				Audience:  jwt.ClaimStrings(tt.audience),
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			}

			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			tokenString, _ := token.SignedString([]byte(testSecret))

			ctx := context.Background()
			result, err := resolver.Resolve(ctx, tokenString)

			if tt.shouldPass {
				require.NoError(t, err)
				assert.NotNil(t, result)
			} else {
				require.Error(t, err)
				assert.Nil(t, result)

				authErr, ok := IsAuthError(err)
				require.True(t, ok)
				assert.Equal(t, AuthFailureInvalidAudience, authErr.Reason)
			}
		})
	}
}
