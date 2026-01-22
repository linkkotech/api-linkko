package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"linkko-api/internal/http/httperr"
	"linkko-api/internal/observability/logger"

	"go.uber.org/zap"
)

// mapAuthErrorToCode maps auth failure reasons to HTTP error codes
func mapAuthErrorToCode(authErr *AuthError) string {
	if authErr == nil {
		return httperr.ErrCodeInvalidToken
	}

	switch authErr.Reason {
	case AuthFailureMissingAuthorization:
		return httperr.ErrCodeMissingAuthorization
	case AuthFailureInvalidScheme:
		return httperr.ErrCodeInvalidScheme
	case AuthFailureInvalidSignature:
		return httperr.ErrCodeInvalidSignature
	case AuthFailureTokenExpired:
		return httperr.ErrCodeTokenExpired
	case AuthFailureInvalidIssuer:
		return httperr.ErrCodeInvalidIssuer
	case AuthFailureInvalidAudience:
		return httperr.ErrCodeInvalidAudience
	case AuthFailureWorkspaceMismatch:
		return httperr.ErrCodeWorkspaceMismatch
	default:
		return httperr.ErrCodeInvalidToken
	}
}

// S2STokenStore stores service-to-service authentication tokens
type S2STokenStore struct {
	tokens map[string]string // token -> client name
}

// NewS2STokenStore creates a new S2S token store
func NewS2STokenStore() *S2STokenStore {
	return &S2STokenStore{
		tokens: make(map[string]string),
	}
}

// RegisterToken registers an S2S token for a client
func (s *S2STokenStore) RegisterToken(token, clientName string) {
	if token != "" {
		s.tokens[token] = clientName
	}
}

// ValidateToken validates an S2S token and returns the client name
func (s *S2STokenStore) ValidateToken(token string) (string, bool) {
	client, ok := s.tokens[token]
	return client, ok
}

// isJWTToken checks if a token looks like a JWT (starts with "eyJ" and has two dots)
func isJWTToken(token string) bool {
	return strings.HasPrefix(token, "eyJ") && strings.Count(token, ".") == 2
}

// validateS2SHeaders validates optional X-Actor-Id and X-Workspace-Id headers
func validateS2SHeaders(r *http.Request) (workspaceID, actorID string, err error) {
	workspaceID = r.Header.Get("X-Workspace-Id")
	actorID = r.Header.Get("X-Actor-Id")

	// Headers are optional, but if provided must be non-empty
	if workspaceID == "" && actorID == "" {
		return "", "", nil // Both optional, no error
	}

	if workspaceID != "" && strings.TrimSpace(workspaceID) == "" {
		return "", "", fmt.Errorf("X-Workspace-Id must be non-empty")
	}

	if actorID != "" && strings.TrimSpace(actorID) == "" {
		return "", "", fmt.Errorf("X-Actor-Id must be non-empty")
	}

	return workspaceID, actorID, nil
}

// AuthMiddleware validates both JWT and S2S tokens
func AuthMiddleware(resolver *KeyResolver, s2sStore *S2STokenStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := logger.GetLogger(r.Context())

			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				log.Warn(r.Context(), "authentication failed",
					zap.String("auth_failure_reason", string(AuthFailureMissingAuthorization)),
					zap.String("remote_addr", r.RemoteAddr),
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
				)
				httperr.Unauthorized401(w, r.Context(), httperr.ErrCodeMissingAuthorization, "missing authorization header")
				return
			}

			// Check Bearer format
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				log.Warn(r.Context(), "authentication failed",
					zap.String("auth_failure_reason", string(AuthFailureInvalidScheme)),
					zap.String("remote_addr", r.RemoteAddr),
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
				)
				httperr.Unauthorized401(w, r.Context(), httperr.ErrCodeInvalidScheme, "invalid authorization scheme, expected Bearer")
				return
			}

			tokenString := parts[1]
			var ctx context.Context

			// Determine if token is JWT or S2S
			if isJWTToken(tokenString) {
				// Handle JWT authentication
				ctx = handleJWTAuth(r.Context(), resolver, tokenString, log, w, r)
				if ctx == nil {
					return // Error already handled
				}
			} else {
				// Handle S2S authentication
				ctx = handleS2SAuth(r.Context(), s2sStore, tokenString, r, log, w)
				if ctx == nil {
					return // Error already handled
				}
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// handleJWTAuth handles JWT token validation
func handleJWTAuth(ctx context.Context, resolver *KeyResolver, tokenString string, log *logger.Logger, w http.ResponseWriter, r *http.Request) context.Context {
	// Validate token
	claims, err := resolver.Resolve(ctx, tokenString)
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
		log.Warn(ctx, "authentication failed",
			zap.String("auth_failure_reason", failureReason),
			zap.String("auth_type", "jwt"),
			zap.String("token_prefix", maskToken(tokenString)),
			zap.String("remote_addr", r.RemoteAddr),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Error(err),
		)
		// Map auth failure reason to HTTP error code
		errorCode := mapAuthErrorToCode(authErr)
		httperr.Unauthorized401(w, ctx, errorCode, "invalid or expired token")
		return nil
	}

	// Create auth context with metadata
	authCtx := &AuthContext{
		WorkspaceID: claims.WorkspaceID,
		ActorID:     claims.ActorID,
		ActorType:   "user", // Default actor type
		AuthMethod:  "jwt",  // Authentication method
		Issuer:      claims.Issuer,
	}

	// Add claims and auth context to request context
	ctx = context.WithValue(ctx, claimsContextKey, claims)
	ctx = context.WithValue(ctx, authContextKey, authCtx)

	// Log successful authentication
	log.Info(ctx, "authenticated request",
		zap.String("auth_type", "jwt"),
		zap.String("workspace_id", claims.WorkspaceID),
		zap.String("actor_id", claims.ActorID),
		zap.String("actor_type", authCtx.ActorType),
		zap.String("auth_method", authCtx.AuthMethod),
		zap.String("issuer", claims.Issuer),
	)

	return ctx
}

// handleS2SAuth handles S2S token validation
func handleS2SAuth(ctx context.Context, s2sStore *S2STokenStore, tokenString string, r *http.Request, log *logger.Logger, w http.ResponseWriter) context.Context {
	// Validate S2S token
	client, ok := s2sStore.ValidateToken(tokenString)
	if !ok {
		log.Warn(ctx, "authentication failed",
			zap.String("auth_failure_reason", string(AuthFailureInvalidSignature)),
			zap.String("auth_type", "s2s"),
			zap.String("token_prefix", maskToken(tokenString)),
			zap.String("remote_addr", r.RemoteAddr),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
		)
		httperr.Unauthorized401(w, ctx, httperr.ErrCodeInvalidSignature, "invalid S2S token")
		return nil
	}

	// Extract optional headers
	workspaceID, actorID, err := validateS2SHeaders(r)
	if err != nil {
		log.Warn(ctx, "authentication failed",
			zap.String("auth_failure_reason", string(AuthFailureUnknown)),
			zap.String("auth_type", "s2s"),
			zap.String("client", client),
			zap.String("remote_addr", r.RemoteAddr),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Error(err),
		)
		httperr.BadRequest400(w, ctx, httperr.ErrCodeInvalidParameter, "invalid X-Workspace-Id or X-Actor-Id header")
		return nil
	}

	// Create auth context for S2S
	authCtx := &AuthContext{
		WorkspaceID: workspaceID,
		ActorID:     actorID,
		ActorType:   "service",
		AuthMethod:  "s2s",
		Client:      client,
	}

	// Add auth context to request context
	ctx = context.WithValue(ctx, authContextKey, authCtx)

	// Log successful authentication
	logFields := []logger.Field{
		zap.String("auth_type", "s2s"),
		zap.String("client", client),
		zap.String("actor_type", authCtx.ActorType),
		zap.String("auth_method", authCtx.AuthMethod),
	}
	if workspaceID != "" {
		logFields = append(logFields, zap.String("workspace_id", workspaceID))
	}
	if actorID != "" {
		logFields = append(logFields, zap.String("actor_id", actorID))
	}
	log.Info(ctx, "authenticated request", logFields...)

	return ctx
}
