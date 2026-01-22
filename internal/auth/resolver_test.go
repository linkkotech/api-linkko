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

func TestKeyResolver_IssuerMismatch(t *testing.T) {
	// Setup
	keyStore := NewKeyStore()
	keyStore.LoadHS256Key("issuer-a", "v1", []byte(testSecret))
	
	validator := NewHS256Validator(keyStore, "issuer-a", 60*time.Second)
	resolver := NewKeyResolver([]string{"issuer-a"}, []string{testAudience})
	resolver.RegisterValidator("issuer-a", validator)

	// Create token that claims issuer-b but is signed by issuer-a's key
	claims := &CustomClaims{
		WorkspaceID: "ws-12345",
		ActorID:     "user-67890",
	}
	claims.RegisteredClaims = jwt.RegisteredClaims{
		Issuer:    "issuer-b", // Mismatch
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
