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
				log.Warn("authentication failed",
					zap.String("auth_failure_reason", string(AuthFailureMissingAuthorization)),
					zap.String("remote_addr", r.RemoteAddr),
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
				)
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			// Check Bearer format
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				log.Warn("authentication failed",
					zap.String("auth_failure_reason", string(AuthFailureInvalidScheme)),
					zap.String("remote_addr", r.RemoteAddr),
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
				)
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			// Validate token
			claims, err := resolver.Resolve(r.Context(), tokenString)
			if err != nil {
				// Extract categorized auth error
				authErr, ok := IsAuthError(err)
				var failureReason string
				if ok {
					failureReason = string(authErr.Reason)
				} else {
					failureReason = string(AuthFailureUnknown)
				}

				// Log with detailed context (token masked for security)
				log.Warn("authentication failed",
					zap.String("auth_failure_reason", failureReason),
					zap.String("token_prefix", maskToken(tokenString)),
					zap.String("remote_addr", r.RemoteAddr),
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.Error(err),
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
