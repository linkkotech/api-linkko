package middleware

import (
	"context"
	"net/http"
	"regexp"

	"linkko-api/internal/auth"
	"linkko-api/internal/http/httperr"
	"linkko-api/internal/logger"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type contextKey string

const workspaceIDKey contextKey = "workspace_id"

// workspaceIDPattern validates workspace ID format: alphanumeric, underscore, hyphen
var workspaceIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// validateWorkspaceIDFormat checks if workspaceID is a valid string format
func validateWorkspaceIDFormat(workspaceID string) bool {
	if workspaceID == "" {
		return false
	}
	if len(workspaceID) > 64 {
		return false
	}
	return workspaceIDPattern.MatchString(workspaceID)
}

// WorkspaceMiddleware validates workspace access and prevents IDOR attacks
func WorkspaceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := logger.GetLogger(r.Context())

		// Extract workspace ID from URL path parameter
		workspaceID := chi.URLParam(r, "workspaceId")
		if workspaceID == "" {
			log.Warn("workspace_id not found in path")
			httperr.BadRequest400(w, r.Context(), httperr.ErrCodeMissingParameter, "workspaceId is required in path")
			return
		}

		// Validate workspace ID format (string, not UUID)
		if !validateWorkspaceIDFormat(workspaceID) {
			log.Warn("invalid workspace_id format",
				zap.String("workspace_id", workspaceID),
			)
			httperr.BadRequest400(w, r.Context(), httperr.ErrCodeInvalidWorkspaceID, "workspaceId must contain only alphanumeric characters, hyphens, and underscores (max 64 chars)")
			return
		}

		// Get auth context (set by AuthMiddleware)
		authCtx, ok := auth.GetAuthContext(r.Context())
		if !ok {
			log.Error("auth context not found")
			httperr.Unauthorized401(w, r.Context(), httperr.ErrCodeInvalidToken, "authentication required")
			return
		}

		// IDOR Prevention: Verify workspace_id matches authenticated context
		// For JWT: claims.workspaceId must match path
		// For S2S: X-Workspace-Id header (if present) must match path
		if authCtx.WorkspaceID != "" && authCtx.WorkspaceID != workspaceID {
			// Use structured observability logging
			fields := []zap.Field{
				zap.String("auth_failure_reason", "workspace_mismatch"),
				zap.String("auth_type", authCtx.AuthMethod),
				zap.String("authenticated_workspace_id", authCtx.WorkspaceID),
				zap.String("path_workspace_id", workspaceID),
				zap.String("actor_id", authCtx.ActorID),
				zap.String("remote_addr", r.RemoteAddr),
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
			}
			if authCtx.Client != "" {
				fields = append(fields, zap.String("client", authCtx.Client))
			}
			if authCtx.Issuer != "" {
				fields = append(fields, zap.String("issuer", authCtx.Issuer))
			}
			log.Warn("workspace access denied - mismatch detected", fields...)
			httperr.Forbidden403(w, r.Context(), httperr.ErrCodeWorkspaceMismatch, "workspace access denied")
			return
		}

		// Add workspace_id as span attribute for tracing
		span := trace.SpanFromContext(r.Context())
		span.SetAttributes(attribute.String("workspace_id", workspaceID))

		// Inject validated workspace_id into context for downstream handlers
		ctx := context.WithValue(r.Context(), workspaceIDKey, workspaceID)

		log.Debug("workspace access granted",
			zap.String("workspace_id", workspaceID),
			zap.String("actor_id", authCtx.ActorID),
		)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetWorkspaceID retrieves validated workspace ID from context
func GetWorkspaceID(ctx context.Context) (string, bool) {
	workspaceID, ok := ctx.Value(workspaceIDKey).(string)
	return workspaceID, ok
}
