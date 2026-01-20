package middleware

import (
	"fmt"
	"net/http"
	"time"

	"linkko-api/internal/logger"
	"linkko-api/internal/ratelimit"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// RateLimitMiddleware enforces rate limiting per workspace
func RateLimitMiddleware(limiter *ratelimit.RedisRateLimiter, limitPerMin int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := logger.GetLogger(r.Context())

			// Get workspace ID from context (set by WorkspaceMiddleware)
			workspaceID, ok := GetWorkspaceID(r.Context())
			if !ok {
				log.Error("workspace_id not found in context for rate limiting")
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			// Check rate limit
			allowed, remaining, err := limiter.AllowRequest(r.Context(), workspaceID, limitPerMin, 60)
			if err != nil {
				log.Error("rate limit check failed", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			// Add rate limit headers
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limitPerMin))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(60*time.Second).Unix()))

			if !allowed {
				// Add span event for rate limit exceeded
				span := trace.SpanFromContext(r.Context())
				span.AddEvent("rate_limit_exceeded")

				log.Warn("rate limit exceeded",
					zap.String("workspace_id", workspaceID),
					zap.Int("limit", limitPerMin),
				)

				w.Header().Set("Retry-After", "60")
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
