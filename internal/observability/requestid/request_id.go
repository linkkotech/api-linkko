package requestid

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

type contextKey string

const requestIDContextKey contextKey = "request_id"

// NewRequestID generates a new ULID-like request ID
// Format: timestamp (10 chars base32) + randomness (16 chars hex) = 26 chars total
// ULID provides better time-based ordering than UUID for request tracing
func NewRequestID() string {
	// ULID-inspired: timestamp (milliseconds) + random bytes
	timestamp := time.Now().UnixMilli()

	// Generate 10 random bytes
	randomBytes := make([]byte, 10)
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback: use timestamp only if randomness fails
		return fmt.Sprintf("req_%d", timestamp)
	}

	// Format: req_<timestamp>_<hex>
	return fmt.Sprintf("req_%d_%s", timestamp, hex.EncodeToString(randomBytes))
}

// GetRequestID retrieves request ID from context
func GetRequestID(ctx context.Context) string {
	if v := ctx.Value(requestIDContextKey); v != nil {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

// SetRequestID stores request ID in context
func SetRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDContextKey, id)
}
