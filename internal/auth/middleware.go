package auth

import (
	"context"
	"net/http"
	"strings"

	"linkko-api/internal/logger"

	"go.uber.org/zap"
)

type contextKey string

const claimsContextKey contextKey = "claims"

// JWTAuthMiddleware validates JWT tokens and injects claims into context
func JWTAuthMiddleware(resolver *KeyResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := logger.GetLogger(r.Context())

			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				log.Warn("missing authorization header")
				http.Error(w, "missing authorization header", http.StatusUnauthorized)
				return
			}

			// Check Bearer format
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				log.Warn("invalid authorization header format")
				http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			// Validate token
			claims, err := resolver.Resolve(r.Context(), tokenString)
			if err != nil {
				// Categorize error for better logging
				log.Warn("token validation failed",
					zap.Error(err),
					zap.String("remote_addr", r.RemoteAddr),
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
				)
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			// Add claims to context
			ctx := context.WithValue(r.Context(), claimsContextKey, claims)

			// Log successful authentication
			log.Info("authenticated request",
				zap.String("workspace_id", claims.WorkspaceID),
				zap.String("actor_id", claims.ActorID),
				zap.String("issuer", claims.Issuer),
			)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetClaims retrieves claims from context
func GetClaims(ctx context.Context) (*CustomClaims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(*CustomClaims)
	return claims, ok
}
