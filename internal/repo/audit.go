package repo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AuditRepo handles audit log storage
type AuditRepo struct {
	pool *pgxpool.Pool
}

// NewAuditRepo creates a new AuditRepo
func NewAuditRepo(pool *pgxpool.Pool) *AuditRepo {
	return &AuditRepo{pool: pool}
}

// LogAction logs an action to the audit log
func (r *AuditRepo) LogAction(
	ctx context.Context,
	workspaceID, actorID, action, resourceType string,
	resourceID *string,
	metadata map[string]interface{},
	ipAddress, userAgent string,
) error {
	var metadataJSON []byte
	var err error

	if metadata != nil {
		metadataJSON, err = json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
		INSERT INTO audit_log (
			workspace_id, actor_id, action, resource_type, resource_id,
			metadata, ip_address, user_agent
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = r.pool.Exec(ctx, query,
		workspaceID, actorID, action, resourceType, resourceID,
		metadataJSON, ipAddress, userAgent,
	)
	if err != nil {
		return fmt.Errorf("failed to log action: %w", err)
	}

	return nil
}
