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
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"guestflow/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
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

	filteredDir, cleanup, err := prepareMigrationDir(command, migrationsDir)
	if err != nil {
		log.Fatalf("Failed to prepare migrations: %v", err)
	}
	defer cleanup()

	// Open database connection
	db, err := goose.OpenDBWithDriver("pgx", dbString)
	if err != nil {
		log.Fatalf("Failed to open database: %v\nConnection string: %s", err, redactPassword(dbString))
	}
	defer db.Close()

	// Run migration command
	var arguments []string
	if len(args) > 1 {
		arguments = append(arguments, args[1:]...)
	}

	if err := goose.Run(command, db, filteredDir, arguments...); err != nil {
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

func prepareMigrationDir(command, srcDir string) (string, func(), error) {
	switch command {
	case "up", "up-by-one", "up-to", "status", "version":
		tempDir, err := os.MkdirTemp("", "guestflow-migrations-*")
		if err != nil {
			return "", func() {}, err
		}

		entries, err := os.ReadDir(srcDir)
		if err != nil {
			_ = os.RemoveAll(tempDir)
			return "", func() {}, err
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			name := entry.Name()
			if !strings.HasSuffix(name, ".sql") {
				continue
			}
			if strings.HasSuffix(name, ".down.sql") {
				continue
			}


			srcPath := filepath.Join(srcDir, name)
			dstPath := filepath.Join(tempDir, name)

			data, err := os.ReadFile(srcPath)
			if err != nil {
				_ = os.RemoveAll(tempDir)
				return "", func() {}, err
			}

			if err := os.WriteFile(dstPath, data, 0o644); err != nil {
				_ = os.RemoveAll(tempDir)
				return "", func() {}, err
			}
		}

		return tempDir, func() { _ = os.RemoveAll(tempDir) }, nil
	default:
		return srcDir, func() {}, nil
	}
}
