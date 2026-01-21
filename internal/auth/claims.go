package auth

import (
	"github.com/golang-jwt/jwt/v5"
)

// CustomClaims represents the custom JWT claims for the API
type CustomClaims struct {
	WorkspaceID string `json:"workspaceId"`
	ActorID     string `json:"actorId"`
	jwt.RegisteredClaims
}

// Validate performs additional validation on custom claims
func (c *CustomClaims) Validate() error {
	if c.WorkspaceID == "" {
		return jwt.ErrTokenInvalidClaims
	}
	if c.ActorID == "" {
		return jwt.ErrTokenInvalidClaims
	}
	return nil
}
