package main

import (
	"context"
	"fmt"

	"linkko-api/internal/config"
	"linkko-api/internal/database"
	"linkko-api/internal/logger"
	"linkko-api/internal/repo"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Cleanup expired idempotency keys",
	Long:  `Remove idempotency keys older than 24 hours from the database`,
	RunE:  runCleanup,
}

func init() {
	rootCmd.AddCommand(cleanupCmd)
}

func runCleanup(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logger
	log, err := logger.NewLogger()
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}
	defer log.Sync()

	log.Info("starting idempotency keys cleanup")

	// Connect to database
	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer pool.Close()

	// Initialize repository
	idempotencyRepo := repo.NewIdempotencyRepo(pool)

	// Cleanup expired keys
	rowsDeleted, err := idempotencyRepo.CleanupExpired(ctx)
	if err != nil {
		log.Error("cleanup failed", zap.Error(err))
		return fmt.Errorf("failed to cleanup expired keys: %w", err)
	}

	log.Info("cleanup completed", zap.Int64("rows_deleted", rowsDeleted))
	fmt.Printf("✓ Cleanup completed: %d expired keys removed\n", rowsDeleted)

	return nil
}
