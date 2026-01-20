package repo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// IdempotencyRepo handles idempotency key storage and retrieval
type IdempotencyRepo struct {
	pool *pgxpool.Pool
}

// NewIdempotencyRepo creates a new IdempotencyRepo
func NewIdempotencyRepo(pool *pgxpool.Pool) *IdempotencyRepo {
	return &IdempotencyRepo{pool: pool}
}

// CachedResponse represents a cached response from an idempotent request
type CachedResponse struct {
	Status  int
	Body    json.RawMessage
	Headers map[string]string
}

// HashKey generates SHA256 hash of idempotency key
func HashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// CheckKey checks if an idempotency key exists and returns cached response
func (r *IdempotencyRepo) CheckKey(ctx context.Context, workspaceID, keyHash string) (*CachedResponse, error) {
	query := `
		SELECT response_status, response_body, response_headers
		FROM idempotency_keys
		WHERE workspace_id = $1 AND key_hash = $2 AND expires_at > NOW()
	`

	var status int
	var body json.RawMessage
	var headersJSON []byte

	err := r.pool.QueryRow(ctx, query, workspaceID, keyHash).Scan(&status, &body, &headersJSON)
	if err == pgx.ErrNoRows {
		return nil, nil // Key not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to check idempotency key: %w", err)
	}

	var headers map[string]string
	if headersJSON != nil {
		if err := json.Unmarshal(headersJSON, &headers); err != nil {
			return nil, fmt.Errorf("failed to unmarshal headers: %w", err)
		}
	}

	return &CachedResponse{
		Status:  status,
		Body:    body,
		Headers: headers,
	}, nil
}

// StoreResult stores the result of an idempotent request
func (r *IdempotencyRepo) StoreResult(
	ctx context.Context,
	workspaceID, keyHash, originalKey, method, path string,
	requestPayload json.RawMessage,
	status int,
	responseBody json.RawMessage,
	responseHeaders map[string]string,
) error {
	headersJSON, err := json.Marshal(responseHeaders)
	if err != nil {
		return fmt.Errorf("failed to marshal headers: %w", err)
	}

	query := `
		INSERT INTO idempotency_keys (
			key_hash, workspace_id, original_key, request_method, request_path,
			request_payload, response_status, response_body, response_headers, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW() + INTERVAL '24 hours')
		ON CONFLICT (workspace_id, key_hash) DO NOTHING
	`

	_, err = r.pool.Exec(ctx, query,
		keyHash, workspaceID, originalKey, method, path,
		requestPayload, status, responseBody, headersJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to store idempotency result: %w", err)
	}

	return nil
}

// CleanupExpired removes expired idempotency keys
func (r *IdempotencyRepo) CleanupExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM idempotency_keys WHERE created_at < NOW() - INTERVAL '24 hours'`

	result, err := r.pool.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired keys: %w", err)
	}

	return result.RowsAffected(), nil
}
