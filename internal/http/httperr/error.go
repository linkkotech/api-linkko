package httperr

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	"linkko-api/internal/observability/logger"

	"go.uber.org/zap"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	OK    bool         `json:"ok"`
	Error *ErrorDetail `json:"error"`
}

// ErrorDetail contains the error information
type ErrorDetail struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
	ErrorID string            `json:"error_id,omitempty"`
}

// Error codes for 401 Unauthorized (authentication failures)
const (
	ErrCodeMissingAuthorization = "MISSING_AUTHORIZATION"
	ErrCodeInvalidScheme        = "INVALID_SCHEME"
	ErrCodeInvalidToken         = "INVALID_TOKEN"
	ErrCodeInvalidSignature     = "INVALID_SIGNATURE"
	ErrCodeTokenExpired         = "TOKEN_EXPIRED"
	ErrCodeInvalidIssuer        = "INVALID_ISSUER"
	ErrCodeInvalidAudience      = "INVALID_AUDIENCE"
)

// Error codes for 403 Forbidden (authorized but insufficient permissions)
const (
	ErrCodeWorkspaceMismatch = "WORKSPACE_MISMATCH"
	ErrCodeForbidden         = "FORBIDDEN"
	ErrCodeInsufficientScope = "INSUFFICIENT_SCOPE"
	ErrCodeNotFound          = "NOT_FOUND" // Added
)

// Error codes for 400 Bad Request (validation errors)
const (
	ErrCodeInvalidWorkspaceID = "INVALID_WORKSPACE_ID"
	ErrCodeInvalidParameter   = "INVALID_PARAMETER"
	ErrCodeInvalidFormat      = "INVALID_FORMAT"
	ErrCodeMissingParameter   = "MISSING_PARAMETER"
	ErrCodeInvalidLimit       = "INVALID_LIMIT"
	ErrCodeValidationError    = "VALIDATION_ERROR"
	ErrCodeInvalidStatus      = "INVALID_STATUS"
	ErrCodeInvalidPriority    = "INVALID_PRIORITY"
	ErrCodeInvalidType        = "INVALID_TYPE"
	ErrCodeConflict           = "CONFLICT" // Added
)

// Error codes for 500 Internal Server Error
const (
	ErrCodeInternalError = "INTERNAL_ERROR"
)

// WriteError writes a standardized error response
func WriteError(w http.ResponseWriter, ctx context.Context, status int, code, message string) {
	log := logger.GetLogger(ctx)
	reqID := logger.GetRequestIDFromContext(ctx)

	log.Error(ctx, "request failed",
		zap.Int("status_code", status),
		zap.String("error_code", code),
		zap.String("message", message),
		zap.String("request_id", reqID),
	)

	response := ErrorResponse{
		OK: false,
		Error: &ErrorDetail{
			Code:    code,
			Message: message,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

// WriteErrorWithFields writes a standardized error response with field-level details
func WriteErrorWithFields(w http.ResponseWriter, ctx context.Context, status int, code, message string, fields map[string]string) {
	log := logger.GetLogger(ctx)

	fieldPairs := make([]zap.Field, 0, len(fields)+3)
	fieldPairs = append(fieldPairs,
		zap.Int("status_code", status),
		zap.String("error_code", code),
		zap.String("message", message),
	)
	for k, v := range fields {
		fieldPairs = append(fieldPairs, zap.String("field_"+k, v))
	}

	log.Error(ctx, "request failed with field errors", fieldPairs...)

	response := ErrorResponse{
		OK: false,
		Error: &ErrorDetail{
			Code:    code,
			Message: message,
			Fields:  fields,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

// Unauthorized401 writes a 401 Unauthorized response
func Unauthorized401(w http.ResponseWriter, ctx context.Context, code, message string) {
	WriteError(w, ctx, http.StatusUnauthorized, code, message)
}

// Forbidden403 writes a 403 Forbidden response
func Forbidden403(w http.ResponseWriter, ctx context.Context, code, message string) {
	WriteError(w, ctx, http.StatusForbidden, code, message)
}

// BadRequest400 writes a 400 Bad Request response
func BadRequest400(w http.ResponseWriter, ctx context.Context, code, message string) {
	WriteError(w, ctx, http.StatusBadRequest, code, message)
}

// BadRequest400WithFields writes a 400 Bad Request response with field-level errors
func BadRequest400WithFields(w http.ResponseWriter, ctx context.Context, code, message string, fields map[string]string) {
	WriteErrorWithFields(w, ctx, http.StatusBadRequest, code, message, fields)
}

// InternalError500 writes a 500 Internal Server Error response
func InternalError500(w http.ResponseWriter, ctx context.Context, message string) {
	reqID := logger.GetRequestIDFromContext(ctx)

	log := logger.GetLogger(ctx)
	log.Error(ctx, "internal server error",
		zap.String("message", message),
		zap.String("request_id", reqID),
	)

	// In prod, return generic message for security
	response := ErrorResponse{
		OK: false,
		Error: &ErrorDetail{
			Code:    ErrCodeInternalError,
			Message: "Internal Server Error",
		},
	}

	// APP_ENV detection
	if os.Getenv("APP_ENV") == "dev" {
		response.Error.ErrorID = reqID
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(response)
}

// InternalError is an alias for InternalError500 as requested
func InternalError(w http.ResponseWriter, ctx context.Context) {
	InternalError500(w, ctx, "internal server error")
}
