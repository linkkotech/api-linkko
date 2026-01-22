package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenValidator validates JWT tokens
type TokenValidator interface {
	Validate(tokenString string, kid string) (*CustomClaims, error)
}

// HS256Validator validates HS256 JWT tokens
type HS256Validator struct {
	keyStore  *KeyStore
	issuer    string
	clockSkew time.Duration
}

// NewHS256Validator creates a new HS256 validator
func NewHS256Validator(keyStore *KeyStore, issuer string, clockSkew time.Duration) *HS256Validator {
	return &HS256Validator{
		keyStore:  keyStore,
		issuer:    issuer,
		clockSkew: clockSkew,
	}
}

// Validate validates an HS256 JWT token
func (v *HS256Validator) Validate(tokenString string, kid string) (*CustomClaims, error) {
	// Get secret from key store
	secret, ok := v.keyStore.GetHS256Key(v.issuer, kid)
	if !ok {
		return nil, fmt.Errorf("key not found for issuer %s and kid %s", v.issuer, kid)
	}

	// Parse token with clock skew
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	}, jwt.WithLeeway(v.clockSkew))

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, NewAuthError(AuthFailureTokenExpired, "token expired", err)
		}
		if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
			return nil, NewAuthError(AuthFailureInvalidSignature, "invalid signature", err)
		}
		return nil, NewAuthError(AuthFailureUnknown, "failed to parse token", err)
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, NewAuthError(AuthFailureUnknown, fmt.Sprintf("invalid token: valid=%v", token.Valid), nil)
	}

	// Validate custom claims
	if err := claims.Validate(); err != nil {
		return nil, NewAuthError(AuthFailureUnknown, "invalid claims", err)
	}

	return claims, nil
}

// RS256Validator validates RS256 JWT tokens
type RS256Validator struct {
	keyStore  *KeyStore
	issuer    string
	clockSkew time.Duration
}

// NewRS256Validator creates a new RS256 validator
func NewRS256Validator(keyStore *KeyStore, issuer string, clockSkew time.Duration) *RS256Validator {
	return &RS256Validator{
		keyStore:  keyStore,
		issuer:    issuer,
		clockSkew: clockSkew,
	}
}

// Validate validates an RS256 JWT token
func (v *RS256Validator) Validate(tokenString string, kid string) (*CustomClaims, error) {
	// Get public key from key store
	publicKey, ok := v.keyStore.GetRS256Key(v.issuer, kid)
	if !ok {
		return nil, NewAuthError(AuthFailureUnknown, fmt.Sprintf("key not found for issuer %s and kid %s", v.issuer, kid), nil)
	}

	// Parse token with clock skew
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	}, jwt.WithLeeway(v.clockSkew))

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, NewAuthError(AuthFailureTokenExpired, "token expired", err)
		}
		if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
			return nil, NewAuthError(AuthFailureInvalidSignature, "invalid signature", err)
		}
		return nil, NewAuthError(AuthFailureUnknown, "failed to parse token", err)
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, NewAuthError(AuthFailureUnknown, fmt.Sprintf("invalid token: valid=%v", token.Valid), nil)
	}

	// Validate custom claims
	if err := claims.Validate(); err != nil {
		return nil, NewAuthError(AuthFailureUnknown, "invalid claims", err)
	}

	return claims, nil
}
