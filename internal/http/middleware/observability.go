package middleware

import (
	"net/http"
	"runtime/debug"
	"time"

	"linkko-api/internal/observability/logger"
	"linkko-api/internal/observability/requestid"

	"go.uber.org/zap"
)

// RequestIDMiddleware reads or generates request ID and propagates it
// - Reads X-Request-Id header
// - Generates new ULID if missing
// - Injects into context
// - Writes X-Request-Id header to response
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read X-Request-Id from request
		reqID := r.Header.Get("X-Request-Id")

		// Generate new ID if missing
		if reqID == "" {
			reqID = requestid.NewRequestID()
		}

		// Inject into context
		ctx := requestid.SetRequestID(r.Context(), reqID)

		// Write X-Request-Id to response header (for client-side tracing)
		w.Header().Set("X-Request-Id", reqID)

		// Continue with enriched context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestLoggingMiddleware logs HTTP requests with mandatory fields
// Logs at request END to include status code and latency
// MUST include: request_id, route, method, status, latency_ms
// MUST NOT include: sensitive headers, request/response bodies
func RequestLoggingMiddleware(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Inject logger into context
			ctx := logger.SetLoggerInContext(r.Context(), log)

			// Wrap response writer to capture status code
			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK, // default
			}

			// Process request
			next.ServeHTTP(wrapped, r.WithContext(ctx))

			// Calculate latency
			latency := time.Since(start)
			latencyMs := float64(latency.Milliseconds())

			// Log request summary
			log.Info(
				ctx,
				"http request completed",
				logger.Module("http"),
				logger.Action("request"),
				zap.String("method", r.Method),
				zap.String("route", r.URL.Path),
				zap.String("path", r.URL.Path),
				zap.String("query", sanitizeQuery(r.URL.RawQuery)),
				zap.Int("status", wrapped.statusCode),
				zap.Float64("latency_ms", latencyMs),
				zap.String("remote_addr", sanitizeRemoteAddr(r.RemoteAddr)),
				zap.String("user_agent", sanitizeUserAgent(r.UserAgent())),
			)
		})
	}
}

// RecoveryMiddleware recovers from panics and logs with stack trace
// Prevents service crash while preserving error context
// Stack trace is allowed (no secrets in stack)
func RecoveryMiddleware(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Get stack trace
					stack := string(debug.Stack())

					// Log panic with request_id
					log.Error(
						r.Context(),
						"panic recovered",
						logger.Module("http"),
						logger.Action("panic_recovery"),
						zap.Any("panic", err),
						zap.String("stack", stack),
						zap.String("method", r.Method),
						zap.String("route", r.URL.Path),
					)

					// Return 500 to client
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	return rw.ResponseWriter.Write(b)
}

// sanitizeQuery removes sensitive query parameters
// SECURITY: prevent logging tokens, passwords in query strings
func sanitizeQuery(query string) string {
	if query == "" {
		return ""
	}

	// For production: parse and filter sensitive keys (token, password, etc.)
	// For now: truncate long queries to prevent log bloat
	const maxLen = 200
	if len(query) > maxLen {
		return query[:maxLen] + "..."
	}
	return query
}

// sanitizeRemoteAddr removes port from remote address for privacy
// Example: 192.168.1.100:54321 -> 192.168.1.100
func sanitizeRemoteAddr(addr string) string {
	// Simple implementation: could use net.SplitHostPort for production
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i]
		}
	}
	return addr
}

// sanitizeUserAgent truncates user agent to prevent log bloat
func sanitizeUserAgent(ua string) string {
	const maxLen = 100
	if len(ua) > maxLen {
		return ua[:maxLen] + "..."
	}
	return ua
}

// WithWorkspaceID is a helper middleware to inject workspace_id into context
// Used after workspace extraction from JWT or path parameter
func WithWorkspaceID(workspaceID string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := logger.SetWorkspaceIDInContext(r.Context(), workspaceID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// WithUserID is a helper middleware to inject user_id into context
// Used after authentication
func WithUserID(userID string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := logger.SetUserIDInContext(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
