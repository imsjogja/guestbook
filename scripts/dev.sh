#!/usr/bin/env bash
# =============================================================================
# GuestFlow Development Script
# =============================================================================
# Quick-start helper for local development.
#
# Usage:
#   ./scripts/dev.sh setup    # First-time setup (DB + migrations + seed)
#   ./scripts/dev.sh server   # Start Go server
#   ./scripts/dev.sh migrate  # Run database migrations
#   ./scripts/dev.sh seed     # Insert demo data
#   ./scripts/dev.sh worker   # Start background worker
#   ./scripts/dev.sh test     # Run all tests
#   ./scripts/dev.sh down     # Stop Docker services
#   ./scripts/dev.sh reset    # Full reset (destroy all data)
#
# Prerequisites: Docker, Docker Compose, Go 1.22+
# =============================================================================

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Project root
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

# Configuration
DB_USER="${DB_USER:-guestflow}"
DB_PASSWORD="${DB_PASSWORD:-guestflow}"
DB_NAME="${DB_NAME:-guestflow}"
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
APP_PORT="${APP_PORT:-8080}"

# Helper functions
log_info() { echo -e "${BLUE}[INFO]${NC} $*"; }
log_ok()   { echo -e "${GREEN}[OK]${NC} $*"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_err()  { echo -e "${RED}[ERROR]${NC} $*"; }

wait_for_postgres() {
    log_info "Waiting for PostgreSQL..."
    local retries=30
    while ! pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" > /dev/null 2>&1; do
        retries=$((retries - 1))
        if [ $retries -eq 0 ]; then
            log_err "PostgreSQL failed to start after 30 seconds"
            exit 1
        fi
        sleep 1
    done
    log_ok "PostgreSQL is ready"
}

wait_for_redis() {
    log_info "Waiting for Redis..."
    local retries=30
    while ! redis-cli -h "${REDIS_HOST:-localhost}" -p "${REDIS_PORT:-6379}" ping > /dev/null 2>&1; do
        retries=$((retries - 1))
        if [ $retries -eq 0 ]; then
            log_err "Redis failed to start after 30 seconds"
            exit 1
        fi
        sleep 1
    done
    log_ok "Redis is ready"
}

# =============================================================================
# Commands
# =============================================================================

cmd_setup() {
    log_info "Setting up GuestFlow development environment..."

    # Check prerequisites
    if ! command -v docker &> /dev/null; then
        log_err "Docker is required but not installed."
        exit 1
    fi

    if ! command -v go &> /dev/null; then
        log_err "Go is required but not installed. Get it from https://go.dev/dl/"
        exit 1
    fi

    # Copy .env if not exists
    if [ ! -f .env ]; then
        cp .env.example .env
        log_ok "Created .env from .env.example"
    fi

    # Start infrastructure
    log_info "Starting Docker services (PostgreSQL, Redis)..."
    docker compose up -d db redis

    # Wait for services
    wait_for_postgres
    wait_for_redis

    # Run migrations
    cmd_migrate

    # Seed data
    cmd_seed

    log_ok "Setup complete!"
    log_info "Start the server with: ./scripts/dev.sh server"
    log_info "Start the worker with: ./scripts/dev.sh worker"
    echo ""
    log_info "Demo credentials:"
    echo "  Email:    demo@guestflow.id"
    echo "  Password: password123"
    echo "  Tenant:   demo-wo"
}

cmd_migrate() {
    log_info "Running database migrations..."
    go run cmd/migrate/main.go up
    log_ok "Migrations applied successfully"
}

cmd_seed() {
    log_info "Inserting demo data..."
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
        -f seeds/seed_data.sql 2>/dev/null || {
        log_warn "Could not seed data automatically. Run manually:"
        echo "  PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f seeds/seed_data.sql"
    }
    log_ok "Demo data inserted"
}

cmd_server() {
    log_info "Starting GuestFlow server on port $APP_PORT..."
    log_info "API:       http://localhost:$APP_PORT/api/v1"
    log_info "Admin:     http://localhost:$APP_PORT/admin"
    log_info "Health:    http://localhost:$APP_PORT/health"
    echo ""
    go run cmd/server/main.go
}

cmd_worker() {
    log_info "Starting background worker..."
    go run cmd/worker/main.go -queues=all -concurrency=3
}

cmd_test() {
    log_info "Running tests..."
    go test -v ./tests/feature/... ./internal/auth/... ./internal/middleware/...
    log_ok "Tests completed"
}

cmd_test_all() {
    log_info "Running all tests with coverage..."
    go test -race -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out | tail -1
    log_ok "All tests completed"
}

cmd_down() {
    log_info "Stopping Docker services..."
    docker compose down
    log_ok "Services stopped"
}

cmd_reset() {
    log_warn "This will DESTROY all data in the database!"
    read -p "Are you sure? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        docker compose down -v
        docker volume rm guestflow_postgres_data guestflow_redis_data 2>/dev/null || true
        log_ok "All data destroyed. Run './scripts/dev.sh setup' to start fresh."
    else
        log_info "Reset cancelled"
    fi
}

cmd_help() {
    echo "GuestFlow Development Script"
    echo ""
    echo "Usage: ./scripts/dev.sh <command>"
    echo ""
    echo "Commands:"
    echo "  setup    First-time environment setup"
    echo "  server   Start Go server"
    echo "  worker   Start background worker"
    echo "  migrate  Run database migrations"
    echo "  seed     Insert demo data"
    echo "  test     Run feature tests"
    echo "  test-all Run all tests with coverage"
    echo "  down     Stop Docker services"
    echo "  reset    Full reset (DESTROYS all data)"
    echo "  help     Show this help"
    echo ""
    echo "Environment Variables:"
    echo "  DB_HOST      Database host (default: localhost)"
    echo "  DB_PORT      Database port (default: 5432)"
    echo "  DB_USER      Database user (default: guestflow)"
    echo "  DB_PASSWORD  Database password (default: guestflow)"
    echo "  APP_PORT     Server port (default: 8080)"
}

# =============================================================================
# Main
# =============================================================================

COMMAND="${1:-help}"

case "$COMMAND" in
    setup)    cmd_setup ;;
    server)   cmd_server ;;
    worker)   cmd_worker ;;
    migrate)  cmd_migrate ;;
    seed)     cmd_seed ;;
    test)     cmd_test ;;
    test-all) cmd_test_all ;;
    down)     cmd_down ;;
    reset)    cmd_reset ;;
    help|*)   cmd_help ;;
esac
