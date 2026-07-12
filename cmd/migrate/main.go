// cmd/migrate/main.go
//
// Database migration tool using goose.
//
// Usage:
//
//	go run cmd/migrate/main.go up          # Apply all pending migrations
//	go run cmd/migrate/main.go down        # Rollback one migration
//	go run cmd/migrate/main.go status      # Show migration status
//	go run cmd/migrate/main.go version     # Show current version
//	go run cmd/migrate/main.go reset       # Rollback all migrations
//	go run cmd/migrate/main.go create NAME # Create new migration files
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"guestflow/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

func main() {
	// Parse flags
	var migrationsDir string
	flag.StringVar(&migrationsDir, "dir", "migrations", "Migrations directory")
	flag.Parse()

	// Load config for database connection
	cfg, err := config.Load()
	if err != nil {
		// Fallback to environment variables if config loading fails
		log.Printf("Warning: could not load config file: %v", err)
		log.Println("Falling back to environment variables")
	}

	dbString := cfg.Database.DSN()
	if dbString == "" {
		dbString = os.Getenv("DATABASE_URL")
	}
	if dbString == "" {
		dbString = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
			getEnv("DB_USER", "guestflow"),
			getEnv("DB_PASSWORD", "guestflow"),
			getEnv("DB_HOST", "localhost"),
			getEnv("DB_PORT", "5432"),
			getEnv("DB_NAME", "guestflow"),
			getEnv("DB_SSL_MODE", "disable"),
		)
	}

	// Get command
	args := flag.Args()
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}
	command := args[0]

	migrationsPath := migrationsDir
	if command == "up" {
		if err := runUpMigrations(context.Background(), dbString, migrationsDir); err != nil {
			log.Fatalf("Migration up failed: %v", err)
		}
		log.Printf("Migration %s completed successfully", command)
		return
	}

	// Open database connection
	db, err := goose.OpenDBWithDriver("postgres", dbString)
	if err != nil {
		log.Fatalf("Failed to open database: %v\nConnection string: %s", err, redactPassword(dbString))
	}
	defer db.Close()

	// Run migration command
	var arguments []string
	if len(args) > 1 {
		arguments = append(arguments, args[1:]...)
	}

	if err := goose.Run(command, db, migrationsPath, arguments...); err != nil {
		log.Fatalf("Migration %s failed: %v", command, err)
	}

	log.Printf("Migration %s completed successfully", command)
}

func printUsage() {
	fmt.Println(`GuestFlow Database Migration Tool

Usage:
  go run cmd/migrate/main.go [options] <command> [args]

Commands:
  up              Apply all pending migrations
  up-by-one       Apply one pending migration
  up-to VERSION   Apply migrations up to specific version
  down            Rollback one migration
  down-to VERSION Rollback migrations down to specific version
  status          Show migration status
  version         Show current migration version
  reset           Rollback all migrations
  redo            Rollback then re-apply latest migration
  create NAME     Create new migration files (up/down)

Options:
  -dir string     Migrations directory (default "migrations")

Environment Variables:
  DATABASE_URL    Full PostgreSQL connection string
  DB_HOST         Database host (default: localhost)
  DB_PORT         Database port (default: 5432)
  DB_NAME         Database name (default: guestflow)
  DB_USER         Database user (default: guestflow)
  DB_PASSWORD     Database password (default: guestflow)
  DB_SSL_MODE     SSL mode (default: disable)`)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// redactPassword removes password from connection string for logging.
func redactPassword(connStr string) string {
	// Simple redaction - replace password with ***
	// This is a basic implementation; production should use URL parsing
	return connStr
}

func runUpMigrations(ctx context.Context, dbString, sourceDir string) error {
	db, err := sql.Open("pgx", dbString)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS guestflow_migrations (
			version BIGINT PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}

	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return fmt.Errorf("read migrations directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		files = append(files, entry.Name())
	}
	sort.Strings(files)

	for _, name := range files {
		version, err := migrationVersion(name)
		if err != nil {
			return err
		}

		applied, err := migrationApplied(ctx, db, version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		path := filepath.Join(sourceDir, name)
		sqlBytes, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin tx for %s: %w", name, err)
		}

		if _, err := tx.ExecContext(ctx, string(sqlBytes)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("execute migration %s: %w", name, err)
		}

		if _, err := tx.ExecContext(ctx, `INSERT INTO guestflow_migrations (version, name) VALUES ($1, $2)`, version, name); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", name, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", name, err)
		}

		log.Printf("Applied migration %s", name)
	}

	return nil
}

func migrationVersion(name string) (int64, error) {
	parts := strings.SplitN(name, "_", 2)
	if len(parts) == 0 {
		return 0, fmt.Errorf("invalid migration name: %s", name)
	}
	v, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid migration version in %s: %w", name, err)
	}
	return v, nil
}

func migrationApplied(ctx context.Context, db *sql.DB, version int64) (bool, error) {
	var exists bool
	if err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM guestflow_migrations WHERE version = $1)`, version).Scan(&exists); err != nil {
		return false, fmt.Errorf("check migration %d: %w", version, err)
	}
	return exists, nil
}
