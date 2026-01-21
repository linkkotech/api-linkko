package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"linkko-api/internal/logger"

	"go.uber.org/zap"
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
func (kr *KeyResolver) Resolve(ctx context.Context, tokenString string) (*CustomClaims, error) {
	log := logger.GetLogger(ctx)

	// Extract issuer and kid from JWT header without validating signature
	issuer, kid, originalKid, err := kr.extractHeaderInfo(tokenString)
	if err != nil {
		return nil, fmt.Errorf("failed to extract header info: %w", err)
	}

	// Log kid selection for debugging
	if originalKid == "" {
		log.Debug("jwt kid fallback applied",
			zap.String("issuer", issuer),
			zap.String("original_kid", "<empty>"),
			zap.String("selected_kid", kid),
		)
	} else {
		log.Debug("jwt kid extracted",
			zap.String("issuer", issuer),
			zap.String("kid", kid),
		)
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
// Returns: issuer, selectedKid, originalKid, error
func (kr *KeyResolver) extractHeaderInfo(tokenString string) (string, string, string, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("invalid token format")
	}

	// Decode header
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", "", "", fmt.Errorf("failed to decode header: %w", err)
	}

	var header struct {
		Kid string `json:"kid"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return "", "", "", fmt.Errorf("failed to unmarshal header: %w", err)
	}

	// Decode payload to get issuer
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", "", fmt.Errorf("failed to decode payload: %w", err)
	}

	var payload struct {
		Issuer string `json:"iss"`
	}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return "", "", "", fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	originalKid := header.Kid
	selectedKid := originalKid

	// HOTFIX: Fallback to "v1" for linkko-crm-web when kid is empty
	if selectedKid == "" {
		if payload.Issuer == "linkko-crm-web" {
			selectedKid = "v1"
		} else {
			// For other issuers, also default to v1 if kid is missing
			selectedKid = "v1"
		}
	}

	return payload.Issuer, selectedKid, originalKid, nil
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
