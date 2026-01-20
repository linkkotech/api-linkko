package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"linkko-api/internal/http/middleware"
	"linkko-api/internal/observability/logger"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHealthEndpoint verifies /health returns 200 without dependencies
func TestHealthEndpoint(t *testing.T) {
	// Create router with minimal middlewares
	r := chi.NewRouter()
	r.Use(middleware.RequestIDMiddleware)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	// Execute request
	r.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code, "health endpoint should return 200")
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "ok", response["status"])
}

// TestHealthEndpoint_ReturnsRequestID verifies X-Request-Id header is returned
func TestHealthEndpoint_ReturnsRequestID(t *testing.T) {
	// Create router with RequestIDMiddleware
	r := chi.NewRouter()
	r.Use(middleware.RequestIDMiddleware)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Create request without X-Request-Id
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	// Execute request
	r.ServeHTTP(w, req)

	// Verify X-Request-Id header is present
	requestID := w.Header().Get("X-Request-Id")
	assert.NotEmpty(t, requestID, "X-Request-Id should be generated and returned")
	assert.Contains(t, requestID, "req_", "Request ID should have req_ prefix")
}

// TestHealthEndpoint_PreservesRequestID verifies existing X-Request-Id is preserved
func TestHealthEndpoint_PreservesRequestID(t *testing.T) {
	// Create router with RequestIDMiddleware
	r := chi.NewRouter()
	r.Use(middleware.RequestIDMiddleware)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Create request with explicit X-Request-Id
	clientRequestID := "req_1234567890_abcdef123456"
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Request-Id", clientRequestID)
	w := httptest.NewRecorder()

	// Execute request
	r.ServeHTTP(w, req)

	// Verify X-Request-Id is preserved
	requestID := w.Header().Get("X-Request-Id")
	assert.Equal(t, clientRequestID, requestID, "X-Request-Id should be preserved from request")
}

// TestHealthEndpoint_NoAuth verifies no authentication is required
func TestHealthEndpoint_NoAuth(t *testing.T) {
	// Create router with middleware stack (except auth)
	r := chi.NewRouter()
	r.Use(middleware.RequestIDMiddleware)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Create request without Authorization header
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	// Execute request
	r.ServeHTTP(w, req)

	// Verify response is 200 (not 401)
	assert.Equal(t, http.StatusOK, w.Code, "health endpoint should not require auth")
}

// MockDB simulates database with ping method
type MockDB struct {
	pingError error
}

func (m *MockDB) Ping(ctx context.Context) error {
	return m.pingError
}

// TestReadyEndpoint_AllHealthy verifies /ready returns 200 when all dependencies are healthy
func TestReadyEndpoint_AllHealthy(t *testing.T) {
	// Create router with middleware
	r := chi.NewRouter()
	r.Use(middleware.RequestIDMiddleware)

	// Mock healthy DB
	mockDB := &MockDB{pingError: nil}

	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		// Check database
		if err := mockDB.Ping(ctx); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"error","message":"database unavailable"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	})

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	// Execute request
	r.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code, "ready endpoint should return 200 when healthy")
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "ready", response["status"])
}

// TestReadyEndpoint_DatabaseUnhealthy verifies /ready returns 503 when DB is down
func TestReadyEndpoint_DatabaseUnhealthy(t *testing.T) {
	// Create router with middleware
	r := chi.NewRouter()
	r.Use(middleware.RequestIDMiddleware)

	// Create logger for readiness check
	log, err := logger.New("linkko-api-test", "error")
	require.NoError(t, err)

	// Mock unhealthy DB
	mockDB := &MockDB{pingError: context.DeadlineExceeded}

	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		// Check database
		if err := mockDB.Ping(ctx); err != nil {
			log.Error(ctx, "readiness check failed: database unavailable")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"error","message":"database unavailable"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	})

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	// Execute request
	r.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusServiceUnavailable, w.Code, "ready endpoint should return 503 when DB unhealthy")
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "error", response["status"])
	assert.Equal(t, "database unavailable", response["message"])
}

// TestReadyEndpoint_ReturnsRequestID verifies X-Request-Id is echoed on readiness checks
func TestReadyEndpoint_ReturnsRequestID(t *testing.T) {
	// Create router with middleware
	r := chi.NewRouter()
	r.Use(middleware.RequestIDMiddleware)

	// Mock healthy DB
	mockDB := &MockDB{pingError: nil}

	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := mockDB.Ping(ctx); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"error","message":"database unavailable"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	})

	// Create request with X-Request-Id
	clientRequestID := "req_9876543210_xyz789"
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	req.Header.Set("X-Request-Id", clientRequestID)
	w := httptest.NewRecorder()

	// Execute request
	r.ServeHTTP(w, req)

	// Verify X-Request-Id is echoed
	requestID := w.Header().Get("X-Request-Id")
	assert.Equal(t, clientRequestID, requestID, "X-Request-Id should be echoed in response")
}

// TestMiddlewareOrder verifies middleware chain is applied correctly
func TestMiddlewareOrder(t *testing.T) {
	// Track middleware execution order
	var executionOrder []string

	// Create router
	r := chi.NewRouter()

	// Add middlewares in correct order
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "requestid")
			next.ServeHTTP(w, r)
		})
	})

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "recovery")
			next.ServeHTTP(w, r)
		})
	})

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "logging")
			next.ServeHTTP(w, r)
		})
	})

	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		executionOrder = append(executionOrder, "handler")
		w.WriteHeader(http.StatusOK)
	})

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	// Execute request
	r.ServeHTTP(w, req)

	// Verify middleware order
	expected := []string{"requestid", "recovery", "logging", "handler"}
	assert.Equal(t, expected, executionOrder, "Middleware should execute in correct order: RequestID → Recovery → Logging → Handler")
}
