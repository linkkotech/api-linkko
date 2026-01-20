package client_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"linkko-api/internal/http/client"
	"linkko-api/internal/observability/logger"
	"linkko-api/internal/observability/requestid"
)

func TestRequestIDTransport_PropagatesHeader(t *testing.T) {
	const testRequestID = "test-req-123"

	// Create test server that captures headers
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRequestID := r.Header.Get("X-Request-Id")
		if gotRequestID != testRequestID {
			t.Errorf("expected X-Request-Id %q, got %q", testRequestID, gotRequestID)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Create client with RequestIDTransport
	transport := client.NewRequestIDTransport(nil)
	httpClient := &http.Client{Transport: transport}

	// Create request with context containing request_id
	ctx := context.Background()
	ctx = requestid.SetRequestID(ctx, testRequestID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	// Execute request
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestRequestIDTransport_PreservesExistingHeader(t *testing.T) {
	const explicitRequestID = "explicit-req-456"
	const contextRequestID = "context-req-789"

	// Create test server that captures headers
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRequestID := r.Header.Get("X-Request-Id")
		// Should preserve explicit header, NOT use context value
		if gotRequestID != explicitRequestID {
			t.Errorf("expected X-Request-Id %q (explicit), got %q", explicitRequestID, gotRequestID)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Create client with RequestIDTransport
	transport := client.NewRequestIDTransport(nil)
	httpClient := &http.Client{Transport: transport}

	// Create request with context containing different request_id
	ctx := context.Background()
	ctx = requestid.SetRequestID(ctx, contextRequestID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	// Explicitly set header (takes precedence over context)
	req.Header.Set("X-Request-Id", explicitRequestID)

	// Execute request
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestRequestIDTransport_NoHeaderWithoutContext(t *testing.T) {
	// Create test server that captures headers
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRequestID := r.Header.Get("X-Request-Id")
		// Should NOT have X-Request-Id when context has no request_id
		if gotRequestID != "" {
			t.Errorf("expected no X-Request-Id, got %q", gotRequestID)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Create client with RequestIDTransport
	transport := client.NewRequestIDTransport(nil)
	httpClient := &http.Client{Transport: transport}

	// Create request WITHOUT request_id in context
	ctx := context.Background()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	// Execute request
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestRequestIDTransport_UsesLoggerContextKey(t *testing.T) {
	const testRequestID = "test-req-999"

	// Create test server that captures headers
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRequestID := r.Header.Get("X-Request-Id")
		if gotRequestID != testRequestID {
			t.Errorf("expected X-Request-Id %q, got %q", testRequestID, gotRequestID)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Create client with RequestIDTransport
	transport := client.NewRequestIDTransport(nil)
	httpClient := &http.Client{Transport: transport}

	// Create request with context using logger's SetRequestIDInContext
	ctx := context.Background()
	ctx = logger.SetRequestIDInContext(ctx, testRequestID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	// Execute request
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestRequestIDTransport_WithNilBase(t *testing.T) {
	// Verify NewRequestIDTransport(nil) doesn't panic and uses DefaultTransport
	transport := client.NewRequestIDTransport(nil)
	if transport == nil {
		t.Error("expected non-nil transport")
	}

	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	httpClient := &http.Client{Transport: transport}

	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	// Should not panic
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
}

func TestNewInternalHTTPClient(t *testing.T) {
	client := client.NewInternalHTTPClient()

	if client == nil {
		t.Fatal("expected non-nil client")
	}

	if client.Timeout == 0 {
		t.Error("expected non-zero timeout")
	}

	if client.Transport == nil {
		t.Error("expected non-nil transport")
	}
}

func TestNewExternalHTTPClient(t *testing.T) {
	client := client.NewExternalHTTPClient()

	if client == nil {
		t.Fatal("expected non-nil client")
	}

	if client.Timeout == 0 {
		t.Error("expected non-zero timeout")
	}

	if client.Transport == nil {
		t.Error("expected non-nil transport")
	}
}

func TestNewCustomHTTPClient(t *testing.T) {
	customTimeout := 15 * time.Second
	client := client.NewCustomHTTPClient(customTimeout)

	if client == nil {
		t.Fatal("expected non-nil client")
	}

	if client.Timeout != customTimeout {
		t.Errorf("expected timeout %v, got %v", customTimeout, client.Timeout)
	}

	if client.Transport == nil {
		t.Error("expected non-nil transport")
	}
}

func BenchmarkRequestIDTransport_WithContext(b *testing.B) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	transport := client.NewRequestIDTransport(nil)
	httpClient := &http.Client{Transport: transport}

	ctx := context.Background()
	ctx = requestid.SetRequestID(ctx, "bench-req-123")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
		resp, _ := httpClient.Do(req)
		resp.Body.Close()
	}
}
