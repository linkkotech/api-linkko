package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"linkko-api/internal/http/middleware"
	"linkko-api/internal/observability/logger"
	"linkko-api/internal/observability/requestid"
)

func TestRequestIDMiddleware_GeneratesID(t *testing.T) {
	handler := middleware.RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request ID is in context
		reqID := requestid.GetRequestID(r.Context())
		if reqID == "" {
			t.Error("expected request ID in context, got empty string")
		}

		// Verify request ID starts with "req_"
		if !strings.HasPrefix(reqID, "req_") {
			t.Errorf("expected request ID to start with 'req_', got: %s", reqID)
		}

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Verify response has X-Request-Id header
	respReqID := rec.Header().Get("X-Request-Id")
	if respReqID == "" {
		t.Error("expected X-Request-Id in response header, got empty")
	}
	if !strings.HasPrefix(respReqID, "req_") {
		t.Errorf("expected response X-Request-Id to start with 'req_', got: %s", respReqID)
	}
}

func TestRequestIDMiddleware_PreservesExistingID(t *testing.T) {
	const testReqID = "test-request-id-123"

	handler := middleware.RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request ID matches what we sent
		reqID := requestid.GetRequestID(r.Context())
		if reqID != testReqID {
			t.Errorf("expected request ID %q, got %q", testReqID, reqID)
		}

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-Id", testReqID)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Verify response echoes the same request ID
	respReqID := rec.Header().Get("X-Request-Id")
	if respReqID != testReqID {
		t.Errorf("expected response X-Request-Id %q, got %q", testReqID, respReqID)
	}
}

func TestRequestLoggingMiddleware_LogsRequest(t *testing.T) {
	log, err := logger.New("test-service", "info")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer log.Sync()

	handler := middleware.RequestLoggingMiddleware(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify logger is in context
		contextLog := logger.GetLogger(r.Context())
		if contextLog == nil {
			t.Error("expected logger in context, got nil")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test?foo=bar", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestRequestLoggingMiddleware_CapturesStatusCode(t *testing.T) {
	log, err := logger.New("test-service", "info")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer log.Sync()

	tests := []struct {
		name       string
		statusCode int
	}{
		{"200 OK", http.StatusOK},
		{"201 Created", http.StatusCreated},
		{"400 Bad Request", http.StatusBadRequest},
		{"404 Not Found", http.StatusNotFound},
		{"500 Internal Server Error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := middleware.RequestLoggingMiddleware(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.statusCode {
				t.Errorf("expected status %d, got %d", tt.statusCode, rec.Code)
			}
		})
	}
}

func TestRecoveryMiddleware_RecoversPanic(t *testing.T) {
	log, err := logger.New("test-service", "info")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer log.Sync()

	handler := middleware.RecoveryMiddleware(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Should not panic
	handler.ServeHTTP(rec, req)

	// Should return 500
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}

	// Should return standardized JSON
	expectedBody := `{"ok":false,"error":{"code":"INTERNAL_ERROR","message":"Internal Server Error"}}`
	actualBody := strings.TrimSpace(rec.Body.String())
	if actualBody != expectedBody {
		t.Errorf("expected body %q, got %q", expectedBody, actualBody)
	}
}

func TestRecoveryMiddleware_DevModeIncludeErrorID(t *testing.T) {
	t.Setenv("APP_ENV", "dev")

	log, err := logger.New("test-service", "info")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer log.Sync()

	handler := middleware.RequestIDMiddleware(
		middleware.RecoveryMiddleware(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("dev panic")
		})),
	)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}

	// Verify request_id (error_id) is present in body
	body := rec.Body.String()
	if !strings.Contains(body, `"error_id":"req_`) {
		t.Errorf("expected body to contain error_id, got %q", body)
	}

	// Verify it echoes the X-Request-Id header
	reqID := rec.Header().Get("X-Request-Id")
	if !strings.Contains(body, reqID) {
		t.Errorf("expected error_id %q in body, got %q", reqID, body)
	}
}

func TestRecoveryMiddleware_DoesNotAffectNormalFlow(t *testing.T) {
	log, err := logger.New("test-service", "info")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer log.Sync()

	handler := middleware.RecoveryMiddleware(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if body := rec.Body.String(); body != "success" {
		t.Errorf("expected body 'success', got %q", body)
	}
}

func TestMiddlewareStack_Integration(t *testing.T) {
	log, err := logger.New("test-service", "info")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer log.Sync()

	// Stack: RequestID -> Logging -> Recovery -> Handler
	handler := middleware.RequestIDMiddleware(
		middleware.RequestLoggingMiddleware(log)(
			middleware.RecoveryMiddleware(log)(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Verify all context values are present
					reqID := requestid.GetRequestID(r.Context())
					if reqID == "" {
						t.Error("expected request ID in context")
					}

					contextLog := logger.GetLogger(r.Context())
					if contextLog == nil {
						t.Error("expected logger in context")
					}

					w.WriteHeader(http.StatusOK)
				}),
			),
		),
	)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Verify X-Request-Id in response
	if respReqID := rec.Header().Get("X-Request-Id"); respReqID == "" {
		t.Error("expected X-Request-Id in response header")
	}
}

func TestWithWorkspaceID(t *testing.T) {
	const testWorkspaceID = "workspace-123"

	handler := middleware.WithWorkspaceID(testWorkspaceID)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		workspaceID := logger.GetWorkspaceIDFromContext(r.Context())
		if workspaceID != testWorkspaceID {
			t.Errorf("expected workspace ID %q, got %q", testWorkspaceID, workspaceID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
}

func TestWithUserID(t *testing.T) {
	const testUserID = "user-456"

	handler := middleware.WithUserID(testUserID)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := logger.GetUserIDFromContext(r.Context())
		if userID != testUserID {
			t.Errorf("expected user ID %q, got %q", testUserID, userID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
}

func BenchmarkRequestIDMiddleware(b *testing.B) {
	handler := middleware.RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}
