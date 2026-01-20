package requestid_test

import (
	"context"
	"strings"
	"testing"

	"linkko-api/internal/observability/requestid"
)

func TestNewRequestID_Format(t *testing.T) {
	id := requestid.NewRequestID()

	// Verify format: req_<timestamp>_<hex>
	if !strings.HasPrefix(id, "req_") {
		t.Errorf("expected ID to start with 'req_', got: %s", id)
	}

	parts := strings.Split(id, "_")
	if len(parts) != 3 {
		t.Errorf("expected ID to have 3 parts separated by '_', got: %d parts", len(parts))
	}

	// Verify minimum length (req_ + timestamp + _ + 20 hex chars)
	if len(id) < 30 {
		t.Errorf("expected ID length >= 30, got: %d", len(id))
	}
}

func TestNewRequestID_Uniqueness(t *testing.T) {
	// Generate multiple IDs and verify they're unique
	ids := make(map[string]bool)
	const count = 1000

	for i := 0; i < count; i++ {
		id := requestid.NewRequestID()
		if ids[id] {
			t.Errorf("duplicate ID generated: %s", id)
		}
		ids[id] = true
	}

	if len(ids) != count {
		t.Errorf("expected %d unique IDs, got %d", count, len(ids))
	}
}

func TestGetRequestID_EmptyContext(t *testing.T) {
	ctx := context.Background()
	id := requestid.GetRequestID(ctx)

	if id != "" {
		t.Errorf("expected empty string for empty context, got: %s", id)
	}
}

func TestSetRequestID_AndGet(t *testing.T) {
	ctx := context.Background()
	testID := "test-req-123"

	ctx = requestid.SetRequestID(ctx, testID)
	got := requestid.GetRequestID(ctx)

	if got != testID {
		t.Errorf("expected %q, got %q", testID, got)
	}
}

func TestSetRequestID_Overwrite(t *testing.T) {
	ctx := context.Background()

	ctx = requestid.SetRequestID(ctx, "first-id")
	ctx = requestid.SetRequestID(ctx, "second-id")

	got := requestid.GetRequestID(ctx)
	if got != "second-id" {
		t.Errorf("expected 'second-id', got %q", got)
	}
}

func TestRequestID_ContextIsolation(t *testing.T) {
	ctx1 := context.Background()
	ctx2 := context.Background()

	ctx1 = requestid.SetRequestID(ctx1, "id-1")
	ctx2 = requestid.SetRequestID(ctx2, "id-2")

	if got := requestid.GetRequestID(ctx1); got != "id-1" {
		t.Errorf("ctx1: expected 'id-1', got %q", got)
	}

	if got := requestid.GetRequestID(ctx2); got != "id-2" {
		t.Errorf("ctx2: expected 'id-2', got %q", got)
	}
}

func BenchmarkNewRequestID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = requestid.NewRequestID()
	}
}

func BenchmarkSetRequestID(b *testing.B) {
	ctx := context.Background()
	id := "test-req-123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx = requestid.SetRequestID(ctx, id)
	}
}

func BenchmarkGetRequestID(b *testing.B) {
	ctx := context.Background()
	ctx = requestid.SetRequestID(ctx, "test-req-123")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = requestid.GetRequestID(ctx)
	}
}
