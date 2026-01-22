package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testSecret   = "test-secret-key-must-be-at-least-32-chars-long-for-hmac"
	testIssuer   = "linkko-crm-web"
	testAudience = "linkko-api-gateway"
)

// Helper function to create a valid JWT token for testing
func createTestToken(secret string, claims *CustomClaims, exp time.Time) string {
	claims.RegisteredClaims = jwt.RegisteredClaims{
		Issuer:    testIssuer,
		Audience:  jwt.ClaimStrings{testAudience},
		ExpiresAt: jwt.NewNumericDate(exp),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

func TestHS256Validator_ValidToken(t *testing.T) {
	// Setup
	keyStore := NewKeyStore()
	keyStore.LoadHS256Key(testIssuer, "v1", []byte(testSecret))
	validator := NewHS256Validator(keyStore, testIssuer, 60*time.Second)

	// Create valid token
	claims := &CustomClaims{
		WorkspaceID: "ws-12345",
		ActorID:     "user-67890",
	}
	token := createTestToken(testSecret, claims, time.Now().Add(1*time.Hour))

	// Test
	result, err := validator.Validate(token, "v1")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "ws-12345", result.WorkspaceID)
	assert.Equal(t, "user-67890", result.ActorID)
	assert.Equal(t, testIssuer, result.Issuer)
}

func TestHS256Validator_InvalidSignature(t *testing.T) {
	// Setup
	keyStore := NewKeyStore()
	keyStore.LoadHS256Key(testIssuer, "v1", []byte(testSecret))
	validator := NewHS256Validator(keyStore, testIssuer, 60*time.Second)

	// Create token with different secret
	wrongSecret := "wrong-secret-key-must-be-at-least-32-chars-long"
	claims := &CustomClaims{
		WorkspaceID: "ws-12345",
		ActorID:     "user-67890",
	}
	token := createTestToken(wrongSecret, claims, time.Now().Add(1*time.Hour))

	// Test
	result, err := validator.Validate(token, "v1")

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	
	authErr, ok := IsAuthError(err)
	require.True(t, ok)
	assert.Equal(t, AuthFailureInvalidSignature, authErr.Reason)
}

func TestHS256Validator_ExpiredToken(t *testing.T) {
	// Setup
	keyStore := NewKeyStore()
	keyStore.LoadHS256Key(testIssuer, "v1", []byte(testSecret))
	validator := NewHS256Validator(keyStore, testIssuer, 5*time.Second) // Short clock skew

	// Create expired token (expired 10 seconds ago, beyond clock skew)
	claims := &CustomClaims{
		WorkspaceID: "ws-12345",
		ActorID:     "user-67890",
	}
	token := createTestToken(testSecret, claims, time.Now().Add(-10*time.Second))

	// Test
	result, err := validator.Validate(token, "v1")

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	
	authErr, ok := IsAuthError(err)
	require.True(t, ok)
	assert.Equal(t, AuthFailureTokenExpired, authErr.Reason)
}

func TestHS256Validator_ExpiredTokenWithinClockSkew(t *testing.T) {
	// Setup
	keyStore := NewKeyStore()
	keyStore.LoadHS256Key(testIssuer, "v1", []byte(testSecret))
	validator := NewHS256Validator(keyStore, testIssuer, 60*time.Second) // 60 second clock skew

	// Create token expired 30 seconds ago (within clock skew)
	claims := &CustomClaims{
		WorkspaceID: "ws-12345",
		ActorID:     "user-67890",
	}
	token := createTestToken(testSecret, claims, time.Now().Add(-30*time.Second))

	// Test
	result, err := validator.Validate(token, "v1")

	// Assert - should be valid due to clock skew
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "ws-12345", result.WorkspaceID)
}

func TestHS256Validator_MissingWorkspaceID(t *testing.T) {
	// Setup
	keyStore := NewKeyStore()
	keyStore.LoadHS256Key(testIssuer, "v1", []byte(testSecret))
	validator := NewHS256Validator(keyStore, testIssuer, 60*time.Second)

	// Create token without workspaceId
	claims := &CustomClaims{
		WorkspaceID: "", // Missing
		ActorID:     "user-67890",
	}
	token := createTestToken(testSecret, claims, time.Now().Add(1*time.Hour))

	// Test
	result, err := validator.Validate(token, "v1")

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	
	authErr, ok := IsAuthError(err)
	require.True(t, ok)
	assert.Equal(t, AuthFailureUnknown, authErr.Reason)
}

func TestHS256Validator_MissingActorID(t *testing.T) {
	// Setup
	keyStore := NewKeyStore()
	keyStore.LoadHS256Key(testIssuer, "v1", []byte(testSecret))
	validator := NewHS256Validator(keyStore, testIssuer, 60*time.Second)

	// Create token without actorId
	claims := &CustomClaims{
		WorkspaceID: "ws-12345",
		ActorID:     "", // Missing
	}
	token := createTestToken(testSecret, claims, time.Now().Add(1*time.Hour))

	// Test
	result, err := validator.Validate(token, "v1")

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	
	authErr, ok := IsAuthError(err)
	require.True(t, ok)
	assert.Equal(t, AuthFailureUnknown, authErr.Reason)
}

func TestHS256Validator_InvalidKID(t *testing.T) {
	// Setup
	keyStore := NewKeyStore()
	keyStore.LoadHS256Key(testIssuer, "v1", []byte(testSecret))
	validator := NewHS256Validator(keyStore, testIssuer, 60*time.Second)

	// Create valid token
	claims := &CustomClaims{
		WorkspaceID: "ws-12345",
		ActorID:     "user-67890",
	}
	token := createTestToken(testSecret, claims, time.Now().Add(1*time.Hour))

	// Test with wrong kid
	result, err := validator.Validate(token, "v2") // Wrong kid

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "key not found")
}

func TestHS256Validator_MalformedToken(t *testing.T) {
	// Setup
	keyStore := NewKeyStore()
	keyStore.LoadHS256Key(testIssuer, "v1", []byte(testSecret))
	validator := NewHS256Validator(keyStore, testIssuer, 60*time.Second)

	// Test with malformed token
	result, err := validator.Validate("not.a.valid.jwt.token", "v1")

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	
	authErr, ok := IsAuthError(err)
	require.True(t, ok)
	assert.Equal(t, AuthFailureUnknown, authErr.Reason)
}

func TestHS256Validator_WrongAlgorithm(t *testing.T) {
	// Setup
	keyStore := NewKeyStore()
	keyStore.LoadHS256Key(testIssuer, "v1", []byte(testSecret))
	validator := NewHS256Validator(keyStore, testIssuer, 60*time.Second)

	// Create token with HS512 instead of HS256
	claims := &CustomClaims{
		WorkspaceID: "ws-12345",
		ActorID:     "user-67890",
	}
	claims.RegisteredClaims = jwt.RegisteredClaims{
		Issuer:    testIssuer,
		Audience:  jwt.ClaimStrings{testAudience},
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
	}

	// Use a longer secret for HS512
	longSecret := "test-secret-key-must-be-at-least-64-chars-long-for-hmac-sha512-algorithm"
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims) // Wrong algorithm
	tokenString, _ := token.SignedString([]byte(longSecret))

	// Test
	result, err := validator.Validate(tokenString, "v1")

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	
	// The validator will try to parse with HS256 key and fail
	authErr, ok := IsAuthError(err)
	require.True(t, ok)
	// Could be invalid signature or parse error depending on implementation
	assert.True(t, authErr.Reason == AuthFailureInvalidSignature || authErr.Reason == AuthFailureUnknown)
}
