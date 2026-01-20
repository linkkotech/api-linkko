package middleware

import (
	"context"
	"net/http"

	"linkko-api/internal/auth"
	"linkko-api/internal/logger"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type contextKey string

const workspaceIDKey contextKey = "workspace_id"

// WorkspaceMiddleware validates workspace access and prevents IDOR attacks
func WorkspaceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := logger.GetLogger(r.Context())

		// Extract workspace ID from URL path parameter
		workspaceID := chi.URLParam(r, "workspaceId")
		if workspaceID == "" {
			log.Warn("workspace_id not found in path")
			http.Error(w, "workspace_id not found in path", http.StatusBadRequest)
			return
		}

		// Get claims from context (set by JWTAuthMiddleware)
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			log.Error("claims not found in context")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// IDOR Prevention: Verify workspace_id from JWT matches path parameter
		if claims.WorkspaceID != workspaceID {
			log.Warn("workspace access denied - IDOR attempt detected",
				zap.String("jwt_workspace_id", claims.WorkspaceID),
				zap.String("path_workspace_id", workspaceID),
				zap.String("actor_id", claims.ActorID),
			)
			http.Error(w, "workspace access denied", http.StatusForbidden)
			return
		}

		// Add workspace_id as span attribute for tracing
		span := trace.SpanFromContext(r.Context())
		span.SetAttributes(attribute.String("workspace_id", workspaceID))

		// Inject validated workspace_id into context for downstream handlers
		ctx := context.WithValue(r.Context(), workspaceIDKey, workspaceID)

		log.Debug("workspace access granted",
			zap.String("workspace_id", workspaceID),
			zap.String("actor_id", claims.ActorID),
		)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetWorkspaceID retrieves validated workspace ID from context
func GetWorkspaceID(ctx context.Context) (string, bool) {
	workspaceID, ok := ctx.Value(workspaceIDKey).(string)
	return workspaceID, ok
}
