#!/usr/bin/env bash
# GuestFlow Development Startup Script
# Starts the application in development mode with hot reload support.
# Requires: go, docker, docker-compose

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
APP_NAME="guestflow"

# ------------------------------------------------------------------------------
# Helper functions
# ------------------------------------------------------------------------------

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[OK]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if a command exists
check_command() {
    if ! command -v "$1" &> /dev/null; then
        log_error "$1 is required but not installed."
        exit 1
    fi
}

# Wait for a service to be available
wait_for_service() {
    local host=$1
    local port=$2
    local name=$3
    local max_attempts=${4:-30}
    local attempt=1

    log_info "Waiting for ${name} at ${host}:${port}..."
    while ! nc -z "${host}" "${port}" 2>/dev/null; do
        if [ $attempt -ge $max_attempts ]; then
            log_error "${name} did not start within ${max_attempts} seconds"
            return 1
        fi
        attempt=$((attempt + 1))
        sleep 1
    done
    log_success "${name} is ready"
}

# ------------------------------------------------------------------------------
# Main
# ------------------------------------------------------------------------------

cd "${PROJECT_DIR}"

log_info "Starting GuestFlow development environment..."
log_info "Project directory: ${PROJECT_DIR}"

# Check prerequisites
check_command go
check_command docker
check_command docker-compose

# Load environment variables from .env if it exists
if [ -f "${PROJECT_DIR}/.env" ]; then
    log_info "Loading environment from .env"
    set -a
    # shellcheck source=/dev/null
    source "${PROJECT_DIR}/.env"
    set +a
else
    log_warn "No .env file found. Using defaults."
    log_warn "Copy .env.example to .env and customize it for your setup."
fi

# Create uploads directory if it doesn't exist
mkdir -p "${PROJECT_DIR}/uploads"

# Start infrastructure services (Postgres, Redis)
log_info "Starting infrastructure services..."
docker-compose up -d postgres redis

# Wait for services to be ready
wait_for_service "${DB_HOST:-localhost}" "${DB_PORT:-5432}" "PostgreSQL"
wait_for_service "${REDIS_HOST:-localhost}" "${REDIS_PORT:-6379}" "Redis"

# Run database migrations
log_info "Running database migrations..."
if command -v goose &> /dev/null; then
    goose -dir "${PROJECT_DIR}/migrations" postgres \
        "postgres://${DB_USER:-guestflow}:${DB_PASSWORD:-changeme}@${DB_HOST:-localhost}:${DB_PORT:-5432}/${DB_NAME:-guestflow}?sslmode=disable" \
        up
    log_success "Migrations applied"
else
    log_warn "goose not installed. Attempting to install..."
    go install github.com/pressly/goose/v3/cmd/goose@latest
    goose -dir "${PROJECT_DIR}/migrations" postgres \
        "postgres://${DB_USER:-guestflow}:${DB_PASSWORD:-changeme}@${DB_HOST:-localhost}:${DB_PORT:-5432}/${DB_NAME:-guestflow}?sslmode=disable" \
        up
    log_success "Migrations applied"
fi

# Check for air (hot reload tool)
if command -v air &> /dev/null; then
    log_info "Starting application with hot reload (air)..."
    air
else
    log_warn "air not installed. Starting without hot reload."
    log_info "Install air for hot reload: go install github.com/air-verse/air@latest"

    # Set development environment variables
    export APP_ENV=development
    export APP_DEBUG=true
    export LOG_LEVEL=debug
    export LOG_FORMAT=text
    export SERVER_HOST=0.0.0.0
    export SERVER_PORT=8080
    export DB_HOST=localhost
    export REDIS_HOST=localhost

    log_info "Starting Go server..."
    go run ./cmd/server/main.go
fi
