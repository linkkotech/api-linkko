package handler

import (
	"encoding/json"
	"net/http"
	"os"

	"linkko-api/internal/auth"
	"linkko-api/internal/http/httperr"
	"linkko-api/internal/observability/logger"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// DebugHandler provides debug endpoints for development
type DebugHandler struct {
	appEnv string
}

// NewDebugHandler creates a new debug handler
func NewDebugHandler() *DebugHandler {
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "production" // default to production for safety
	}
	return &DebugHandler{
		appEnv: appEnv,
	}
}

// DebugAuthResponse represents the authentication debug response
type DebugAuthResponse struct {
	OK   bool           `json:"ok"`
	Data *DebugAuthData `json:"data"`
}

// DebugAuthData contains authentication information for debugging
type DebugAuthData struct {
	AuthMethod              string  `json:"authMethod"`                      // "jwt" or "s2s"
	Client                  *string `json:"client,omitempty"`                // S2S client name (e.g., "crm", "mcp")
	ActorID                 string  `json:"actorId"`                         // User or service ID
	ActorType               string  `json:"actorType"`                       // "user" or "service"
	WorkspaceIDFromToken    *string `json:"workspaceIdFromToken,omitempty"`  // From JWT claim
	WorkspaceIDFromHeader   *string `json:"workspaceIdFromHeader,omitempty"` // From X-Workspace-Id header (S2S)
	WorkspaceIDFromPath     *string `json:"workspaceIdFromPath,omitempty"`   // From URL path parameter
	TokenIssuer             *string `json:"tokenIssuer,omitempty"`           // JWT issuer
	WorkspaceValidationPass bool    `json:"workspaceValidationPass"`         // Whether workspace middleware validated successfully
}

// GetAuthDebug returns authentication information for debugging
// Only available in development mode (APP_ENV=dev)
// GET /debug/auth
// GET /debug/auth/workspaces/{workspaceId}
func (h *DebugHandler) GetAuthDebug(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.GetLogger(ctx)

	// Only allow in development mode
	if h.appEnv != "dev" && h.appEnv != "development" {
		log.Warn(ctx, "debug endpoint accessed in non-dev environment",
			zap.String("app_env", h.appEnv),
			zap.String("remote_addr", r.RemoteAddr),
		)
		http.NotFound(w, r)
		return
	}

	// Get auth context (should be populated by AuthMiddleware)
	authCtx, ok := auth.GetAuthContext(ctx)
	if !ok {
		httperr.Unauthorized401(w, ctx, httperr.ErrCodeInvalidToken, "authentication required")
		return
	}

	log.Info(ctx, "debug auth endpoint accessed",
		zap.String("auth_method", authCtx.AuthMethod),
		zap.String("actor_id", authCtx.ActorID),
		zap.String("workspace_id", authCtx.WorkspaceID),
	)

	// Build debug response
	data := &DebugAuthData{
		AuthMethod:              authCtx.AuthMethod,
		ActorID:                 authCtx.ActorID,
		ActorType:               authCtx.ActorType,
		WorkspaceValidationPass: true, // If we reach here, workspace middleware passed
	}

	// Populate fields based on auth method
	if authCtx.AuthMethod == "jwt" {
		data.WorkspaceIDFromToken = &authCtx.WorkspaceID
		if authCtx.Issuer != "" {
			data.TokenIssuer = &authCtx.Issuer
		}
	} else if authCtx.AuthMethod == "s2s" {
		data.WorkspaceIDFromHeader = &authCtx.WorkspaceID
		if authCtx.Client != "" {
			data.Client = &authCtx.Client
		}
	}

	// Get workspace from path if present
	workspaceIDFromPath := chi.URLParam(r, "workspaceId")
	if workspaceIDFromPath != "" {
		data.WorkspaceIDFromPath = &workspaceIDFromPath
	}

	// Write response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(DebugAuthResponse{
		OK:   true,
		Data: data,
	})
}

// GetAuthDebugWithWorkspace is the same as GetAuthDebug but with workspace in path
// This tests the workspace middleware validation
// GET /debug/auth/workspaces/{workspaceId}
func (h *DebugHandler) GetAuthDebugWithWorkspace(w http.ResponseWriter, r *http.Request) {
	// Same implementation as GetAuthDebug
	// The workspace middleware will validate the workspaceId before this handler is called
	h.GetAuthDebug(w, r)
}
