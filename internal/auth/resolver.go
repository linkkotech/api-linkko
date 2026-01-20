package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// KeyResolver resolves JWT validators based on issuer and kid
type KeyResolver struct {
	validators       map[string]TokenValidator
	allowedIssuers   map[string]bool
	allowedAudiences []string
}

// NewKeyResolver creates a new KeyResolver
func NewKeyResolver(allowedIssuers []string, allowedAudiences []string) *KeyResolver {
	issuersMap := make(map[string]bool)
	for _, issuer := range allowedIssuers {
		issuersMap[issuer] = true
	}

	return &KeyResolver{
		validators:       make(map[string]TokenValidator),
		allowedIssuers:   issuersMap,
		allowedAudiences: allowedAudiences,
	}
}

// RegisterValidator registers a validator for an issuer
func (kr *KeyResolver) RegisterValidator(issuer string, validator TokenValidator) {
	kr.validators[issuer] = validator
}

// Resolve validates a JWT token by resolving the appropriate validator
func (kr *KeyResolver) Resolve(tokenString string) (*CustomClaims, error) {
	// Extract issuer and kid from JWT header without validating signature
	issuer, kid, err := kr.extractHeaderInfo(tokenString)
	if err != nil {
		return nil, fmt.Errorf("failed to extract header info: %w", err)
	}

	// Check if issuer is allowed
	if !kr.allowedIssuers[issuer] {
		return nil, fmt.Errorf("issuer not allowed: %s", issuer)
	}

	// Get validator for issuer
	validator, ok := kr.validators[issuer]
	if !ok {
		return nil, fmt.Errorf("no validator found for issuer: %s", issuer)
	}

	// Validate token
	claims, err := validator.Validate(tokenString, kid)
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	// Verify issuer claim
	if claims.Issuer != issuer {
		return nil, fmt.Errorf("issuer mismatch: expected %s, got %s", issuer, claims.Issuer)
	}

	// Verify audience
	if !kr.validAudience(claims.Audience) {
		return nil, fmt.Errorf("invalid audience: %v", claims.Audience)
	}

	return claims, nil
}

// extractHeaderInfo extracts issuer and kid from JWT without validating signature
func (kr *KeyResolver) extractHeaderInfo(tokenString string) (string, string, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return "", "", fmt.Errorf("invalid token format")
	}

	// Decode header
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", "", fmt.Errorf("failed to decode header: %w", err)
	}

	var header struct {
		Kid string `json:"kid"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal header: %w", err)
	}

	// Decode payload to get issuer
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", fmt.Errorf("failed to decode payload: %w", err)
	}

	var payload struct {
		Issuer string `json:"iss"`
	}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	kid := header.Kid
	if kid == "" {
		kid = "v1" // default kid if not present
	}

	return payload.Issuer, kid, nil
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
