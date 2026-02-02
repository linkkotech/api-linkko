package database

import (
	"embed"
	"errors"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// RunMigrations executes all pending database migrations
func RunMigrations() error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL environment variable is not set")
	}

	fmt.Println("Starting Database Migrations...")

	// Create iofs source from embedded migrations
	d, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create iofs source: %w", err)
	}

	// Create migrate instance using iofs source
	m, err := migrate.NewWithSourceInstance("iofs", d, databaseURL)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			fmt.Printf("Warning: failed to close migrate source: %v\n", srcErr)
		}
		if dbErr != nil {
			fmt.Printf("Warning: failed to close migrate database: %v\n", dbErr)
		}
	}()

	// Run migrations
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Println("No migrations to run")
			return nil
		}
		return fmt.Errorf("migration failed: %w", err)
	}

	fmt.Println("Database Migrations completed")
	return nil
}
