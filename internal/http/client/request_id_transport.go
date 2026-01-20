package client

import (
	"net/http"

	"linkko-api/internal/observability/logger"
	"linkko-api/internal/observability/requestid"
)

// RequestIDTransport is an http.RoundTripper that automatically propagates
// X-Request-Id header from context to outbound HTTP requests.
//
// This ensures end-to-end correlation across service boundaries.
// WHY: Without automatic propagation, developers must remember to manually
// set headers on every outbound call, which is error-prone and non-deterministic.
type RequestIDTransport struct {
	base http.RoundTripper
}

// NewRequestIDTransport creates a new RequestIDTransport wrapping the base transport.
// If base is nil, defaults to http.DefaultTransport.
func NewRequestIDTransport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &RequestIDTransport{base: base}
}

// RoundTrip implements http.RoundTripper interface.
// It extracts request_id from the request context and sets X-Request-Id header
// if not already present.
//
// IMPORTANT: Does NOT overwrite existing X-Request-Id header.
// This preserves explicit header values if set by caller.
func (t *RequestIDTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Check if X-Request-Id is already set
	if req.Header.Get("X-Request-Id") != "" {
		// Preserve existing header (explicit setting takes precedence)
		return t.base.RoundTrip(req)
	}

	// Extract request_id from context (try both context keys for compatibility)
	ctx := req.Context()
	reqID := logger.GetRequestIDFromContext(ctx)
	if reqID == "" {
		reqID = requestid.GetRequestID(ctx)
	}

	if reqID == "" {
		// No request_id in context; proceed without header
		// This is acceptable for background jobs or non-request-scoped operations
		return t.base.RoundTrip(req)
	}

	// Clone request to avoid mutating the original
	// WHY: http.Request.Header is shared; modifying it can cause race conditions
	clonedReq := req.Clone(ctx)
	clonedReq.Header.Set("X-Request-Id", reqID)

	return t.base.RoundTrip(clonedReq)
}
