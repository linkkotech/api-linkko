package auth

import "errors"

// AuthFailureReason categorizes authentication failures
type AuthFailureReason string

const (
	AuthFailureMissingAuthorization AuthFailureReason = "missing_authorization"
	AuthFailureInvalidScheme        AuthFailureReason = "invalid_scheme"
	AuthFailureInvalidSignature     AuthFailureReason = "invalid_signature"
	AuthFailureInvalidIssuer        AuthFailureReason = "invalid_issuer"
	AuthFailureInvalidAudience      AuthFailureReason = "invalid_audience"
	AuthFailureTokenExpired         AuthFailureReason = "token_expired"
	AuthFailureWorkspaceMismatch    AuthFailureReason = "workspace_mismatch"
	AuthFailureUnknown              AuthFailureReason = "unknown"
)

// AuthError represents a categorized authentication error
type AuthError struct {
	Reason  AuthFailureReason
	Message string
	Err     error
}

// Error implements the error interface
func (e *AuthError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap implements error unwrapping
func (e *AuthError) Unwrap() error {
	return e.Err
}

// NewAuthError creates a new AuthError
func NewAuthError(reason AuthFailureReason, message string, err error) *AuthError {
	return &AuthError{
		Reason:  reason,
		Message: message,
		Err:     err,
	}
}

// IsAuthError checks if an error is an AuthError and returns it
func IsAuthError(err error) (*AuthError, bool) {
	var authErr *AuthError
	if errors.As(err, &authErr) {
		return authErr, true
	}
	return nil, false
}

// maskToken masks a JWT token for safe logging
// Shows only the first 12 characters followed by "..."
func maskToken(token string) string {
	if len(token) <= 12 {
		return "***"
	}
	return token[:12] + "..."
}
