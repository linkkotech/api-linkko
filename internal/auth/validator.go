package auth

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// TokenValidator validates JWT tokens
type TokenValidator interface {
	Validate(tokenString string, kid string) (*CustomClaims, error)
}

// HS256Validator validates HS256 JWT tokens
type HS256Validator struct {
	keyStore *KeyStore
	issuer   string
}

// NewHS256Validator creates a new HS256 validator
func NewHS256Validator(keyStore *KeyStore, issuer string) *HS256Validator {
	return &HS256Validator{
		keyStore: keyStore,
		issuer:   issuer,
	}
}

// Validate validates an HS256 JWT token
func (v *HS256Validator) Validate(tokenString string, kid string) (*CustomClaims, error) {
	// Get secret from key store
	secret, ok := v.keyStore.GetHS256Key(v.issuer, kid)
	if !ok {
		return nil, fmt.Errorf("key not found for issuer %s and kid %s", v.issuer, kid)
	}

	// Parse token
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, fmt.Errorf("token expired: %w", err)
		}
		if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
			return nil, fmt.Errorf("invalid signature: %w", err)
		}
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token: valid=%v", token.Valid)
	}

	// Validate custom claims
	if err := claims.Validate(); err != nil {
		return nil, fmt.Errorf("invalid claims: %w", err)
	}

	return claims, nil
}

// RS256Validator validates RS256 JWT tokens
type RS256Validator struct {
	keyStore *KeyStore
	issuer   string
}

// NewRS256Validator creates a new RS256 validator
func NewRS256Validator(keyStore *KeyStore, issuer string) *RS256Validator {
	return &RS256Validator{
		keyStore: keyStore,
		issuer:   issuer,
	}
}

// Validate validates an RS256 JWT token
func (v *RS256Validator) Validate(tokenString string, kid string) (*CustomClaims, error) {
	// Get public key from key store
	publicKey, ok := v.keyStore.GetRS256Key(v.issuer, kid)
	if !ok {
		return nil, fmt.Errorf("key not found for issuer %s and kid %s", v.issuer, kid)
	}

	// Parse token
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, fmt.Errorf("token expired: %w", err)
		}
		if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
			return nil, fmt.Errorf("invalid signature: %w", err)
		}
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token: valid=%v", token.Valid)
	}

	// Validate custom claims
	if err := claims.Validate(); err != nil {
		return nil, fmt.Errorf("invalid claims: %w", err)
	}

	return claims, nil
}
