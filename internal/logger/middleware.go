package logger

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type contextKey string

const loggerKey contextKey = "logger"

// LoggerMiddleware injects logger with trace context into request context
func LoggerMiddleware(baseLogger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get trace context
			span := trace.SpanFromContext(r.Context())
			spanContext := span.SpanContext()

			// Get request ID from chi middleware
			requestID := middleware.GetReqID(r.Context())

			// Create logger with trace context
			contextLogger := baseLogger.With(
				zap.String("trace_id", spanContext.TraceID().String()),
				zap.String("span_id", spanContext.SpanID().String()),
				zap.String("request_id", requestID),
			)

			// Add logger to context
			ctx := context.WithValue(r.Context(), loggerKey, contextLogger)

			// Add trace ID to response header
			w.Header().Set("X-Trace-Id", spanContext.TraceID().String())
			w.Header().Set("X-Request-Id", requestID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetLogger retrieves logger from context
func GetLogger(ctx context.Context) *zap.Logger {
	if logger, ok := ctx.Value(loggerKey).(*zap.Logger); ok {
		return logger
	}
	return zap.L() // fallback to global logger
}
