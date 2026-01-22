package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"linkko-api/internal/auth"
	"linkko-api/internal/http/httperr"
	"linkko-api/internal/observability/logger"

	"github.com/go-chi/chi/v5"
)

// setupTestContext creates a context with logger for testing
func setupTestContext() context.Context {
	log, _ := logger.New("test", "info")
	return logger.SetLoggerInContext(context.Background(), log)
}

// validateErrorResponse validates JSON error response
func validateErrorResponse(t *testing.T, body string, expectedCode string) {
	var errResp httperr.ErrorResponse
	if err := json.Unmarshal([]byte(body), &errResp); err != nil {
		t.Fatalf("failed to parse error response: %v, body: %s", err, body)
	}

	if errResp.OK {
		t.Error("expected ok=false in error response")
	}

	if errResp.Error == nil {
		t.Fatal("expected error detail, got nil")
	}

	if errResp.Error.Code != expectedCode {
		t.Errorf("expected error code %s, got %s", expectedCode, errResp.Error.Code)
	}
}

func TestValidateWorkspaceIDFormat(t *testing.T) {
	tests := []struct {
		name        string
		workspaceID string
		expected    bool
	}{
		{
			name:        "ValidAlphanumeric",
			workspaceID: "workspace123",
			expected:    true,
		},
		{
			name:        "ValidWithHyphen",
			workspaceID: "workspace-123",
			expected:    true,
		},
		{
			name:        "ValidWithUnderscore",
			workspaceID: "workspace_123",
			expected:    true,
		},
		{
			name:        "ValidMixed",
			workspaceID: "WS-2024_prod-01",
			expected:    true,
		},
		{
			name:        "EmptyString",
			workspaceID: "",
			expected:    false,
		},
		{
			name:        "TooLong",
			workspaceID: "a123456789012345678901234567890123456789012345678901234567890123456",
			expected:    false,
		},
		{
			name:        "InvalidCharacters_Slash",
			workspaceID: "workspace/123",
			expected:    false,
		},
		{
			name:        "InvalidCharacters_Dot",
			workspaceID: "workspace.123",
			expected:    false,
		},
		{
			name:        "InvalidCharacters_Space",
			workspaceID: "workspace 123",
			expected:    false,
		},
		{
			name:        "InvalidCharacters_Special",
			workspaceID: "workspace@123",
			expected:    false,
		},
		{
			name:        "JustHyphens",
			workspaceID: "---",
			expected:    true,
		},
		{
			name:        "ExactlyMaxLength",
			workspaceID: "a12345678901234567890123456789012345678901234567890123456789012",
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateWorkspaceIDFormat(tt.workspaceID)
			if result != tt.expected {
				t.Errorf("validateWorkspaceIDFormat(%q) = %v, expected %v", tt.workspaceID, result, tt.expected)
			}
		})
	}
}

func TestWorkspaceMiddleware_InvalidFormat(t *testing.T) {
	tests := []struct {
		name           string
		workspaceID    string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "EmptyWorkspaceID",
			workspaceID:    "",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "workspace_id not found in path",
		},
		{
			name:           "InvalidCharacters",
			workspaceID:    "workspace.dot",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid workspace_id format",
		},
		{
			name:           "TooLong",
			workspaceID:    "a123456789012345678901234567890123456789012345678901234567890123456",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid workspace_id format",
		},
		{
			name:           "SpecialCharacters",
			workspaceID:    "workspace@123",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid workspace_id format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup logger
			ctx := setupTestContext()

			// Add dummy auth context for tests that need it
			if tt.workspaceID != "" {
				authCtx := auth.AuthContext{
					WorkspaceID: "dummy-ws",
					ActorID:     "test-user",
					ActorType:   "user",
					AuthMethod:  "jwt",
				}
				ctx = auth.SetAuthContextForTesting(ctx, &authCtx)
			}

			// Create router with workspace middleware
			r := chi.NewRouter()
			r.Route("/v1/workspaces/{workspaceId}", func(r chi.Router) {
				r.Use(WorkspaceMiddleware)
				r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})
			})

			// Create request - use valid path structure
			path := "/v1/workspaces/" + tt.workspaceID + "/test"
			if tt.workspaceID == "" {
				path = "/v1/workspaces//test" // Chi won't match empty param
			}
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()

			// Execute
			r.ServeHTTP(rr, req)

			// Validate
			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d, body: %s", tt.expectedStatus, rr.Code, rr.Body.String())
			}

			// Validate JSON error response structure
			var expectedCode string
			if tt.workspaceID == "" {
				expectedCode = httperr.ErrCodeMissingParameter
			} else {
				expectedCode = httperr.ErrCodeInvalidWorkspaceID
			}
			validateErrorResponse(t, rr.Body.String(), expectedCode)
		})
	}
}

func TestWorkspaceMiddleware_Mismatch_JWT(t *testing.T) {
	tests := []struct {
		name              string
		pathWorkspaceID   string
		claimsWorkspaceID string
		expectedStatus    int
		expectedBody      string
	}{
		{
			name:              "Match",
			pathWorkspaceID:   "ws-123",
			claimsWorkspaceID: "ws-123",
			expectedStatus:    http.StatusOK,
			expectedBody:      "",
		},
		{
			name:              "Mismatch",
			pathWorkspaceID:   "ws-123",
			claimsWorkspaceID: "ws-456",
			expectedStatus:    http.StatusForbidden,
			expectedBody:      "workspace access denied",
		},
		{
			name:              "EmptyClaimsWorkspaceID",
			pathWorkspaceID:   "ws-123",
			claimsWorkspaceID: "",
			expectedStatus:    http.StatusOK, // No validation when claims.workspaceId is empty
			expectedBody:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup logger
			ctx := setupTestContext()

			// Inject auth context (JWT)
			authCtx := auth.AuthContext{
				WorkspaceID: tt.claimsWorkspaceID,
				ActorID:     "user-123",
				ActorType:   "user",
				AuthMethod:  "jwt",
				Issuer:      "linkko-crm-web",
			}

			// Create router with workspace middleware
			r := chi.NewRouter()
			r.Route("/v1/workspaces/{workspaceId}", func(r chi.Router) {
				r.Use(WorkspaceMiddleware)
				r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})
			})

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/v1/workspaces/"+tt.pathWorkspaceID+"/test", nil)

			// Inject auth context AFTER creating request
			ctx = auth.SetAuthContextForTesting(ctx, &authCtx)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()

			// Execute
			r.ServeHTTP(rr, req)

			// Validate
			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d, body: %s", tt.expectedStatus, rr.Code, rr.Body.String())
			}
			if tt.expectedBody != "" && !contains(rr.Body.String(), tt.expectedBody) {
				t.Errorf("expected body to contain %q, got %q", tt.expectedBody, rr.Body.String())
			}
		})
	}
}

func TestWorkspaceMiddleware_Mismatch_S2S(t *testing.T) {
	tests := []struct {
		name              string
		pathWorkspaceID   string
		headerWorkspaceID string
		expectedStatus    int
		expectedBody      string
	}{
		{
			name:              "Match",
			pathWorkspaceID:   "ws-prod-01",
			headerWorkspaceID: "ws-prod-01",
			expectedStatus:    http.StatusOK,
			expectedBody:      "",
		},
		{
			name:              "Mismatch",
			pathWorkspaceID:   "ws-prod-01",
			headerWorkspaceID: "ws-dev-02",
			expectedStatus:    http.StatusForbidden,
			expectedBody:      "workspace access denied",
		},
		{
			name:              "NoHeaderWorkspaceID",
			pathWorkspaceID:   "ws-prod-01",
			headerWorkspaceID: "",
			expectedStatus:    http.StatusOK, // No validation when S2S header is absent
			expectedBody:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup logger
			ctx := setupTestContext()

			// Inject auth context (S2S)
			authCtx := auth.AuthContext{
				WorkspaceID: tt.headerWorkspaceID,
				ActorID:     "service-crm",
				ActorType:   "service",
				AuthMethod:  "s2s",
				Client:      "crm-web",
			}
			ctx = auth.SetAuthContextForTesting(ctx, &authCtx)

			// Create router with workspace middleware
			r := chi.NewRouter()
			r.Route("/v1/workspaces/{workspaceId}", func(r chi.Router) {
				r.Use(WorkspaceMiddleware)
				r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})
			})

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/v1/workspaces/"+tt.pathWorkspaceID+"/test", nil)
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()

			// Execute
			r.ServeHTTP(rr, req)

			// Validate
			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d, body: %s", tt.expectedStatus, rr.Code, rr.Body.String())
			}
			if tt.expectedBody != "" && !contains(rr.Body.String(), tt.expectedBody) {
				t.Errorf("expected body to contain %q, got %q", tt.expectedBody, rr.Body.String())
			}
		})
	}
}

func TestWorkspaceMiddleware_NoAuthContext(t *testing.T) {
	// Setup logger
	ctx := setupTestContext()

	// Create router with workspace middleware (no auth context)
	r := chi.NewRouter()
	r.Route("/v1/workspaces/{workspaceId}", func(r chi.Router) {
		r.Use(WorkspaceMiddleware)
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/v1/workspaces/ws-123/test", nil)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	// Execute
	r.ServeHTTP(rr, req)

	// Validate
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
	validateErrorResponse(t, rr.Body.String(), httperr.ErrCodeInvalidToken)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
