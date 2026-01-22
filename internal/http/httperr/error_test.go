package httperr

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"linkko-api/internal/observability/logger"
)

func TestWriteError(t *testing.T) {
	log, _ := logger.New("test", "info")
	ctx := logger.SetLoggerInContext(context.Background(), log)

	tests := []struct {
		name           string
		status         int
		code           string
		message        string
		expectedStatus int
		expectedOK     bool
	}{
		{
			name:           "401 Unauthorized",
			status:         http.StatusUnauthorized,
			code:           ErrCodeInvalidToken,
			message:        "invalid token provided",
			expectedStatus: http.StatusUnauthorized,
			expectedOK:     false,
		},
		{
			name:           "403 Forbidden",
			status:         http.StatusForbidden,
			code:           ErrCodeWorkspaceMismatch,
			message:        "workspace mismatch detected",
			expectedStatus: http.StatusForbidden,
			expectedOK:     false,
		},
		{
			name:           "400 Bad Request",
			status:         http.StatusBadRequest,
			code:           ErrCodeInvalidWorkspaceID,
			message:        "invalid workspace ID format",
			expectedStatus: http.StatusBadRequest,
			expectedOK:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			WriteError(rr, ctx, tt.status, tt.code, tt.message)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			var response ErrorResponse
			if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if response.OK != tt.expectedOK {
				t.Errorf("expected ok=%v, got %v", tt.expectedOK, response.OK)
			}

			if response.Error == nil {
				t.Fatal("expected error detail, got nil")
			}

			if response.Error.Code != tt.code {
				t.Errorf("expected code %s, got %s", tt.code, response.Error.Code)
			}

			if response.Error.Message != tt.message {
				t.Errorf("expected message %s, got %s", tt.message, response.Error.Message)
			}

			contentType := rr.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", contentType)
			}
		})
	}
}

func TestWriteErrorWithFields(t *testing.T) {
	log, _ := logger.New("test", "info")
	ctx := logger.SetLoggerInContext(context.Background(), log)

	rr := httptest.NewRecorder()
	fields := map[string]string{
		"workspaceId": "must be alphanumeric",
		"limit":       "must be between 1 and 100",
	}

	WriteErrorWithFields(rr, ctx, http.StatusBadRequest, ErrCodeInvalidParameter, "validation failed", fields)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	var response ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.OK {
		t.Error("expected ok=false")
	}

	if response.Error == nil {
		t.Fatal("expected error detail, got nil")
	}

	if len(response.Error.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(response.Error.Fields))
	}

	if response.Error.Fields["workspaceId"] != "must be alphanumeric" {
		t.Errorf("unexpected workspaceId field value: %s", response.Error.Fields["workspaceId"])
	}
}

func TestUnauthorized401(t *testing.T) {
	log, _ := logger.New("test", "info")
	ctx := logger.SetLoggerInContext(context.Background(), log)

	rr := httptest.NewRecorder()
	Unauthorized401(rr, ctx, ErrCodeInvalidToken, "token is invalid")

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}

	var response ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Error.Code != ErrCodeInvalidToken {
		t.Errorf("expected code %s, got %s", ErrCodeInvalidToken, response.Error.Code)
	}
}

func TestForbidden403(t *testing.T) {
	log, _ := logger.New("test", "info")
	ctx := logger.SetLoggerInContext(context.Background(), log)

	rr := httptest.NewRecorder()
	Forbidden403(rr, ctx, ErrCodeWorkspaceMismatch, "workspace access denied")

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", rr.Code)
	}

	var response ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Error.Code != ErrCodeWorkspaceMismatch {
		t.Errorf("expected code %s, got %s", ErrCodeWorkspaceMismatch, response.Error.Code)
	}
}

func TestBadRequest400(t *testing.T) {
	log, _ := logger.New("test", "info")
	ctx := logger.SetLoggerInContext(context.Background(), log)

	rr := httptest.NewRecorder()
	BadRequest400(rr, ctx, ErrCodeInvalidWorkspaceID, "invalid workspace ID format")

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	var response ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Error.Code != ErrCodeInvalidWorkspaceID {
		t.Errorf("expected code %s, got %s", ErrCodeInvalidWorkspaceID, response.Error.Code)
	}
}

func TestInternalError500(t *testing.T) {
	log, _ := logger.New("test", "info")
	ctx := logger.SetLoggerInContext(context.Background(), log)

	rr := httptest.NewRecorder()
	InternalError500(rr, ctx, "database connection failed")

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}

	var response ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Error.Code != ErrCodeInternalError {
		t.Errorf("expected code %s, got %s", ErrCodeInternalError, response.Error.Code)
	}
}
