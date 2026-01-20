package client

import (
	"net/http"
	"time"
)

// NewInternalHTTPClient creates an http.Client with sane defaults for internal service calls.
//
// Includes:
// - Request ID propagation via RequestIDTransport
// - Sensible timeouts (no infinite waits)
// - Connection pooling via DefaultTransport
//
// WHY: http.DefaultClient has zero timeouts, which can cause goroutine leaks
// and indefinite hangs. This client enforces deterministic behavior.
func NewInternalHTTPClient() *http.Client {
	// Start with http.DefaultTransport to get connection pooling
	baseTransport := http.DefaultTransport.(*http.Transport).Clone()

	// Wrap with RequestIDTransport for automatic header propagation
	transport := NewRequestIDTransport(baseTransport)

	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second, // Global timeout for entire request lifecycle
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Limit redirects to prevent infinite loops
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
}

// NewExternalHTTPClient creates an http.Client with conservative defaults for external API calls.
//
// Differences from internal client:
// - Longer timeout (60s) for potentially slower external APIs
// - Same request ID propagation for observability
//
// Use this for calls to third-party services (Supabase, external webhooks, etc.)
func NewExternalHTTPClient() *http.Client {
	baseTransport := http.DefaultTransport.(*http.Transport).Clone()
	transport := NewRequestIDTransport(baseTransport)

	return &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second, // More lenient for external APIs
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
}

// NewCustomHTTPClient creates an http.Client with custom timeout.
//
// Use this when you need fine-grained control over timeout values,
// but still want automatic request ID propagation.
func NewCustomHTTPClient(timeout time.Duration) *http.Client {
	baseTransport := http.DefaultTransport.(*http.Transport).Clone()
	transport := NewRequestIDTransport(baseTransport)

	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
}
