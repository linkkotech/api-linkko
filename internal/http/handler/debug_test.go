package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"linkko-api/internal/auth"
	"linkko-api/internal/observability/logger"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDebugHandler_GetAuthDebug_ProductionBlocked(t *testing.T) {
	// Save original APP_ENV and restore after test
	originalEnv := os.Getenv("APP_ENV")
	defer os.Setenv("APP_ENV", originalEnv)

	os.Setenv("APP_ENV", "production")

	handler := NewDebugHandler(nil)

	log, _ := logger.New("test", "info")
	req := httptest.NewRequest("GET", "/debug/auth", nil)
	req = req.WithContext(logger.SetLoggerInContext(context.Background(), log))

	// Set auth context (even with valid auth, should return 404 in production)
	authCtx := &auth.AuthContext{
		AuthMethod:  "jwt",
		WorkspaceID: "workspace-123",
		ActorID:     "user-456",
		ActorType:   "user",
		Issuer:      "linkko-crm-web",
	}
	req = req.WithContext(auth.SetAuthContextForTesting(req.Context(), authCtx))

	rec := httptest.NewRecorder()
	handler.GetAuthDebug(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code, "should return 404 in production")
}

func TestDebugHandler_GetAuthDebug_DevAllowed(t *testing.T) {
	// Save original APP_ENV and restore after test
	originalEnv := os.Getenv("APP_ENV")
	defer os.Setenv("APP_ENV", originalEnv)

	os.Setenv("APP_ENV", "dev")

	handler := NewDebugHandler(nil)

	log, _ := logger.New("test", "info")
	req := httptest.NewRequest("GET", "/debug/auth", nil)
	req = req.WithContext(logger.SetLoggerInContext(context.Background(), log))

	// Set auth context
	authCtx := &auth.AuthContext{
		AuthMethod:  "jwt",
		WorkspaceID: "workspace-123",
		ActorID:     "user-456",
		ActorType:   "user",
		Issuer:      "linkko-crm-web",
	}
	req = req.WithContext(auth.SetAuthContextForTesting(req.Context(), authCtx))

	rec := httptest.NewRecorder()
	handler.GetAuthDebug(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response DebugAuthResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	assert.True(t, response.OK)
	assert.NotNil(t, response.Data)
	assert.Equal(t, "jwt", response.Data.AuthMethod)
	assert.Equal(t, "user-456", response.Data.ActorID)
	assert.Equal(t, "user", response.Data.ActorType)
	assert.NotNil(t, response.Data.WorkspaceIDFromToken)
	assert.Equal(t, "workspace-123", *response.Data.WorkspaceIDFromToken)
	assert.NotNil(t, response.Data.TokenIssuer)
	assert.Equal(t, "linkko-crm-web", *response.Data.TokenIssuer)
	assert.True(t, response.Data.WorkspaceValidationPass)
}

func TestDebugHandler_GetAuthDebug_NoAuth(t *testing.T) {
	// Save original APP_ENV and restore after test
	originalEnv := os.Getenv("APP_ENV")
	defer os.Setenv("APP_ENV", originalEnv)

	os.Setenv("APP_ENV", "dev")

	handler := NewDebugHandler(nil)

	log, _ := logger.New("test", "info")
	req := httptest.NewRequest("GET", "/debug/auth", nil)
	req = req.WithContext(logger.SetLoggerInContext(context.Background(), log))

	// No auth context set

	rec := httptest.NewRecorder()
	handler.GetAuthDebug(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	// Validate error response structure
	var errResponse map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&errResponse)
	require.NoError(t, err)

	assert.False(t, errResponse["ok"].(bool))
	assert.NotNil(t, errResponse["error"])
}

func TestDebugHandler_GetAuthDebug_JWTAuth(t *testing.T) {
	// Save original APP_ENV and restore after test
	originalEnv := os.Getenv("APP_ENV")
	defer os.Setenv("APP_ENV", originalEnv)

	os.Setenv("APP_ENV", "development") // Test with "development" as well

	handler := NewDebugHandler(nil)

	log, _ := logger.New("test", "info")
	req := httptest.NewRequest("GET", "/debug/auth", nil)
	req = req.WithContext(logger.SetLoggerInContext(context.Background(), log))

	authCtx := &auth.AuthContext{
		AuthMethod:  "jwt",
		WorkspaceID: "my-workspace",
		ActorID:     "user-abc-123",
		ActorType:   "user",
		Issuer:      "linkko-crm-web",
	}
	req = req.WithContext(auth.SetAuthContextForTesting(req.Context(), authCtx))

	rec := httptest.NewRecorder()
	handler.GetAuthDebug(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response DebugAuthResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	assert.True(t, response.OK)
	data := response.Data
	assert.Equal(t, "jwt", data.AuthMethod)
	assert.Equal(t, "user-abc-123", data.ActorID)
	assert.Equal(t, "user", data.ActorType)
	assert.NotNil(t, data.WorkspaceIDFromToken)
	assert.Equal(t, "my-workspace", *data.WorkspaceIDFromToken)
	assert.Nil(t, data.WorkspaceIDFromHeader) // JWT doesn't use header
	assert.Nil(t, data.Client)                // JWT doesn't have client
	assert.NotNil(t, data.TokenIssuer)
	assert.Equal(t, "linkko-crm-web", *data.TokenIssuer)
}

func TestDebugHandler_GetAuthDebug_S2SAuth(t *testing.T) {
	// Save original APP_ENV and restore after test
	originalEnv := os.Getenv("APP_ENV")
	defer os.Setenv("APP_ENV", originalEnv)

	os.Setenv("APP_ENV", "dev")

	handler := NewDebugHandler(nil)

	log, _ := logger.New("test", "info")
	req := httptest.NewRequest("GET", "/debug/auth", nil)
	req = req.WithContext(logger.SetLoggerInContext(context.Background(), log))

	authCtx := &auth.AuthContext{
		AuthMethod:  "s2s",
		WorkspaceID: "workspace-xyz",
		ActorID:     "service-crm",
		ActorType:   "service",
		Client:      "crm",
	}
	req = req.WithContext(auth.SetAuthContextForTesting(req.Context(), authCtx))

	rec := httptest.NewRecorder()
	handler.GetAuthDebug(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response DebugAuthResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	assert.True(t, response.OK)
	data := response.Data
	assert.Equal(t, "s2s", data.AuthMethod)
	assert.Equal(t, "service-crm", data.ActorID)
	assert.Equal(t, "service", data.ActorType)
	assert.NotNil(t, data.WorkspaceIDFromHeader)
	assert.Equal(t, "workspace-xyz", *data.WorkspaceIDFromHeader)
	assert.Nil(t, data.WorkspaceIDFromToken) // S2S doesn't use token claim
	assert.NotNil(t, data.Client)
	assert.Equal(t, "crm", *data.Client)
	assert.Nil(t, data.TokenIssuer) // S2S doesn't have issuer
}

func TestDebugHandler_GetAuthDebugWithWorkspace(t *testing.T) {
	// Save original APP_ENV and restore after test
	originalEnv := os.Getenv("APP_ENV")
	defer os.Setenv("APP_ENV", originalEnv)

	os.Setenv("APP_ENV", "dev")

	handler := NewDebugHandler(nil)

	// Create router to test path parameter extraction
	r := chi.NewRouter()
	r.Get("/debug/auth/workspaces/{workspaceId}", handler.GetAuthDebugWithWorkspace)

	log, _ := logger.New("test", "info")
	req := httptest.NewRequest("GET", "/debug/auth/workspaces/test-workspace-456", nil)
	req = req.WithContext(logger.SetLoggerInContext(context.Background(), log))

	authCtx := &auth.AuthContext{
		AuthMethod:  "jwt",
		WorkspaceID: "test-workspace-456",
		ActorID:     "user-999",
		ActorType:   "user",
		Issuer:      "linkko-crm-web",
	}
	req = req.WithContext(auth.SetAuthContextForTesting(req.Context(), authCtx))

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response DebugAuthResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	assert.True(t, response.OK)
	data := response.Data
	assert.Equal(t, "jwt", data.AuthMethod)
	assert.NotNil(t, data.WorkspaceIDFromPath)
	assert.Equal(t, "test-workspace-456", *data.WorkspaceIDFromPath)
	assert.NotNil(t, data.WorkspaceIDFromToken)
	assert.Equal(t, "test-workspace-456", *data.WorkspaceIDFromToken)
	assert.True(t, data.WorkspaceValidationPass)
}

func TestDebugHandler_DefaultAppEnv(t *testing.T) {
	// Save original APP_ENV and restore after test
	originalEnv := os.Getenv("APP_ENV")
	defer func() {
		if originalEnv != "" {
			os.Setenv("APP_ENV", originalEnv)
		} else {
			os.Unsetenv("APP_ENV")
		}
	}()

	// Unset APP_ENV to test default behavior
	os.Unsetenv("APP_ENV")

	handler := NewDebugHandler(nil)

	// Default should be "production" for safety
	assert.Equal(t, "production", handler.appEnv)

	log, _ := logger.New("test", "info")
	req := httptest.NewRequest("GET", "/debug/auth", nil)
	req = req.WithContext(logger.SetLoggerInContext(context.Background(), log))

	authCtx := &auth.AuthContext{
		AuthMethod:  "jwt",
		WorkspaceID: "workspace-123",
		ActorID:     "user-456",
		ActorType:   "user",
	}
	req = req.WithContext(auth.SetAuthContextForTesting(req.Context(), authCtx))

	rec := httptest.NewRecorder()
	handler.GetAuthDebug(rec, req)

	// Should return 404 since default is production
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestPingDB_Success_DevMode(t *testing.T) {
	// Save original APP_ENV and restore after test
	originalEnv := os.Getenv("APP_ENV")
	defer os.Setenv("APP_ENV", originalEnv)

	os.Setenv("APP_ENV", "dev")

	// Create a mock pool that succeeds
	mockPool := &mockPgxPool{
		shouldFail: false,
	}

	handler := &DebugHandler{
		appEnv: "dev",
		pool:   mockPool,
	}

	log, _ := logger.New("test", "info")
	req := httptest.NewRequest("GET", "/debug/db/ping", nil)
	req = req.WithContext(logger.SetLoggerInContext(context.Background(), log))

	rec := httptest.NewRecorder()
	handler.PingDB(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var response map[string]bool
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	assert.True(t, response["ok"])
}

func TestPingDB_Failure_DevMode(t *testing.T) {
	// Save original APP_ENV and restore after test
	originalEnv := os.Getenv("APP_ENV")
	defer os.Setenv("APP_ENV", originalEnv)

	os.Setenv("APP_ENV", "dev")

	// Create a mock pool that fails
	mockPool := &mockPgxPool{
		shouldFail: true,
	}

	handler := &DebugHandler{
		appEnv: "dev",
		pool:   mockPool,
	}

	log, _ := logger.New("test", "info")
	req := httptest.NewRequest("GET", "/debug/db/ping", nil)
	req = req.WithContext(logger.SetLoggerInContext(context.Background(), log))

	rec := httptest.NewRecorder()
	handler.PingDB(rec, req)

	// Should return 500 with standardized error
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var errResponse map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&errResponse)
	require.NoError(t, err)

	assert.False(t, errResponse["ok"].(bool))
	assert.NotNil(t, errResponse["error"])
}

func TestPingDB_ProductionMode(t *testing.T) {
	// Save original APP_ENV and restore after test
	originalEnv := os.Getenv("APP_ENV")
	defer os.Setenv("APP_ENV", originalEnv)

	os.Setenv("APP_ENV", "production")

	mockPool := &mockPgxPool{
		shouldFail: false,
	}

	handler := &DebugHandler{
		appEnv: "production",
		pool:   mockPool,
	}

	log, _ := logger.New("test", "info")
	req := httptest.NewRequest("GET", "/debug/db/ping", nil)
	req = req.WithContext(logger.SetLoggerInContext(context.Background(), log))

	rec := httptest.NewRecorder()
	handler.PingDB(rec, req)

	// Should return 404 in production
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// mockPgxPool implements the minimal interface needed for testing PingDB
type mockPgxPool struct {
	shouldFail bool
}

func (m *mockPgxPool) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return &mockRow{shouldFail: m.shouldFail}
}

func (m *mockPgxPool) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (m *mockPgxPool) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return nil, nil
}

func (m *mockPgxPool) Begin(ctx context.Context) (pgx.Tx, error) {
	return nil, nil
}

func (m *mockPgxPool) Ping(ctx context.Context) error {
	return nil
}

func (m *mockPgxPool) Close() {}

type mockRow struct {
	shouldFail bool
}

func (m *mockRow) Scan(dest ...interface{}) error {
	if m.shouldFail {
		return context.DeadlineExceeded
	}
	// Simulate successful SELECT 1
	if len(dest) > 0 {
		if intPtr, ok := dest[0].(*int); ok {
			*intPtr = 1
		}
	}
	return nil
}
