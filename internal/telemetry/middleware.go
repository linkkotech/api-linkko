package telemetry

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// OTelMiddleware wraps the otelhttp handler with chi route pattern
func OTelMiddleware(serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(next, serviceName,
			otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
				rctx := chi.RouteContext(r.Context())
				if rctx != nil && rctx.RoutePattern() != "" {
					return fmt.Sprintf("%s %s", r.Method, rctx.RoutePattern())
				}
				return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
			}),
		)
	}
}

// MetricsMiddleware records RED metrics (Requests, Errors, Duration)
func MetricsMiddleware(metrics *Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Call next handler
			next.ServeHTTP(ww, r)

			// Record metrics
			duration := time.Since(start).Seconds()
			
			// Get route pattern
			rctx := chi.RouteContext(r.Context())
			route := r.URL.Path
			if rctx != nil && rctx.RoutePattern() != "" {
				route = rctx.RoutePattern()
			}

			// Prepare attributes
			attrs := []attribute.KeyValue{
				attribute.String("method", r.Method),
				attribute.String("route", route),
				attribute.Int("status", ww.statusCode),
			}

			// Record request count
			metrics.RequestsTotal.Add(r.Context(), 1, metric.WithAttributes(attrs...))

			// Record duration
			metrics.RequestDuration.Record(r.Context(), duration, metric.WithAttributes(attrs...))
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
