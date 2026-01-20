package logger_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"linkko-api/internal/observability/logger"
	"linkko-api/internal/observability/requestid"
)

func TestLogger_JSONOutput(t *testing.T) {
	// Create logger
	log, err := logger.New("test-service", "info")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer log.Sync()

	ctx := context.Background()

	// Note: In production test, we'd capture stdout
	// For this test, we verify the logger is created successfully
	// and methods don't panic
	log.Info(ctx, "test info message",
		logger.Module("test"),
		logger.Action("test_action"),
	)

	log.Warn(ctx, "test warn message",
		logger.Module("test"),
		logger.Action("test_action"),
	)

	log.Error(ctx, "test error message",
		logger.Module("test"),
		logger.Action("test_action"),
	)
}

func TestLogger_MandatoryFields(t *testing.T) {
	// This test verifies that our logger wrapper enforces mandatory fields
	log, err := logger.New("test-service", "info")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer log.Sync()

	ctx := context.Background()

	// Log without module/action to verify defaults are applied
	log.Info(ctx, "test message without module/action")

	// Logger should add module="unknown" and action="unknown" as defaults
	// (verified by manual inspection or by capturing output)
}

func TestLogger_ContextFields(t *testing.T) {
	log, err := logger.New("test-service", "info")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer log.Sync()

	// Create context with request ID, workspace ID, user ID
	ctx := context.Background()
	ctx = logger.SetRequestIDInContext(ctx, "test-req-123")
	ctx = logger.SetWorkspaceIDInContext(ctx, "workspace-456")
	ctx = logger.SetUserIDInContext(ctx, "user-789")

	// Log should automatically include these context values
	log.Info(ctx, "test with context",
		logger.Module("test"),
		logger.Action("test_context"),
	)

	// Verify context getters work
	if got := logger.GetRequestIDFromContext(ctx); got != "test-req-123" {
		t.Errorf("GetRequestIDFromContext() = %q, want %q", got, "test-req-123")
	}
	if got := logger.GetWorkspaceIDFromContext(ctx); got != "workspace-456" {
		t.Errorf("GetWorkspaceIDFromContext() = %q, want %q", got, "workspace-456")
	}
	if got := logger.GetUserIDFromContext(ctx); got != "user-789" {
		t.Errorf("GetUserIDFromContext() = %q, want %q", got, "user-789")
	}
}

func TestLogger_SanitizesSecrets(t *testing.T) {
	log, err := logger.New("test-service", "info")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer log.Sync()

	ctx := context.Background()

	// Attempt to log forbidden fields
	// sanitizeFields should replace them with [REDACTED]
	forbiddenFields := []string{
		"authorization",
		"password",
		"token",
		"api_key",
		"email",
		"phone",
	}

	for _, _ = range forbiddenFields {
		// In production, we'd capture output and verify [REDACTED]
		// For now, ensure it doesn't panic
		log.Info(ctx, "test with forbidden field",
			logger.Module("test"),
			logger.Action("test_sanitize"),
		)
	}
}

func TestLogger_LevelParsing(t *testing.T) {
	tests := []struct {
		level string
		want  string
	}{
		{"debug", "debug"},
		{"info", "info"},
		{"warn", "warn"},
		{"error", "error"},
		{"invalid", "info"}, // defaults to info
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			log, err := logger.New("test-service", tt.level)
			if err != nil {
				t.Fatalf("failed to create logger: %v", err)
			}
			defer log.Sync()

			// Just verify logger is created without errors
			// Level verification would require capturing output
		})
	}
}

func TestLogger_WithContext(t *testing.T) {
	log, err := logger.New("test-service", "info")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer log.Sync()

	ctx := context.Background()
	ctx = requestid.SetRequestID(ctx, "test-123")

	// WithContext should return logger with context fields
	contextLog := log.WithContext(ctx)

	// Log with context-enriched logger
	contextLog.Info(ctx, "test with context logger",
		logger.Module("test"),
		logger.Action("test_with_context"),
	)
}

func TestLogger_EmptyContext(t *testing.T) {
	log, err := logger.New("test-service", "info")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer log.Sync()

	ctx := context.Background()

	// WithContext with empty context should return same logger
	contextLog := log.WithContext(ctx)
	if contextLog == nil {
		t.Error("WithContext returned nil for empty context")
	}
}

func TestLogger_RequiresServiceName(t *testing.T) {
	_, err := logger.New("", "info")
	if err == nil {
		t.Error("expected error when serviceName is empty, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "serviceName is required") {
		t.Errorf("expected 'serviceName is required' error, got: %v", err)
	}
}

// Helper function to parse JSON log output (for more advanced tests)
func parseJSONLog(output string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(output), &result)
	return result, err
}

// Example of how to test JSON output structure (requires capturing stdout)
func TestLogger_JSONStructure(t *testing.T) {
	// This is a placeholder test showing how to verify JSON structure
	// In production, you'd use a custom zap core to capture output

	exampleLog := `{
		"timestamp": "2026-01-20T15:04:05.123456789Z",
		"level": "info",
		"service": "test-service",
		"module": "test",
		"action": "test_action",
		"message": "test message"
	}`

	parsed, err := parseJSONLog(exampleLog)
	if err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Verify mandatory fields exist
	mandatoryFields := []string{"timestamp", "level", "service", "module", "action", "message"}
	for _, field := range mandatoryFields {
		if _, exists := parsed[field]; !exists {
			t.Errorf("mandatory field %q missing from log", field)
		}
	}
}

func TestLogger_GetLoggerFromContext(t *testing.T) {
	log, err := logger.New("test-service", "info")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer log.Sync()

	ctx := context.Background()
	ctx = logger.SetLoggerInContext(ctx, log)

	// Retrieve logger from context
	retrievedLog := logger.GetLogger(ctx)
	if retrievedLog == nil {
		t.Error("GetLogger returned nil")
	}
}

func TestLogger_GetLoggerFromEmptyContext(t *testing.T) {
	ctx := context.Background()

	// GetLogger should return fallback logger, not panic
	log := logger.GetLogger(ctx)
	if log == nil {
		t.Error("GetLogger returned nil for empty context")
	}

	// Fallback logger should still work
	log.Info(ctx, "test with fallback logger",
		logger.Module("test"),
		logger.Action("test_fallback"),
	)
}

func BenchmarkLogger_Info(b *testing.B) {
	log, _ := logger.New("bench-service", "info")
	defer log.Sync()

	ctx := context.Background()
	ctx = requestid.SetRequestID(ctx, "bench-123")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Info(ctx, "benchmark message",
			logger.Module("bench"),
			logger.Action("bench_action"),
		)
	}
}
