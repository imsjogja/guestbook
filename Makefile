# GuestFlow - Makefile
# Common development and deployment tasks

.PHONY: all build test clean lint docker-up docker-down migrate-up migrate-down dev fresh help

# ------------------------------------------------------------------------------
# Variables
# ------------------------------------------------------------------------------
APP_NAME := guestflow
BUILD_DIR := ./bin
MIGRATIONS_DIR := ./migrations
DATABASE_URL := "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSL_MODE)"
GO_FILES := $(shell find . -name '*.go' -not -path './vendor/*' -not -path '*/_*')

# ------------------------------------------------------------------------------
# Default Target
# ------------------------------------------------------------------------------
all: build

# ------------------------------------------------------------------------------
# Build
# ------------------------------------------------------------------------------
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-w -s -X main.version=$$(git describe --tags --always 2>/dev/null || echo dev) -X main.buildTime=$$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
		-o $(BUILD_DIR)/server ./cmd/server/main.go
	@echo "Build complete: $(BUILD_DIR)/server"

# ------------------------------------------------------------------------------
# Test
# ------------------------------------------------------------------------------
test:
	@echo "Running tests..."
	go test -v -race -count=1 -coverprofile=coverage.out ./...
	@echo "Test coverage report:"
	@go tool cover -func=coverage.out | tail -1

test-short:
	@echo "Running short tests..."
	go test -v -short -count=1 ./...

test-coverage: test
	@echo "Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# ------------------------------------------------------------------------------
# Development
# ------------------------------------------------------------------------------
dev:
	@echo "Starting development environment..."
	@bash ./scripts/dev.sh

fresh:
	@echo "Resetting and bootstrapping the local stack..."
	@bash ./scripts/dev.sh fresh --yes

dev-hot:
	@echo "Starting development with hot reload..."
	air

# ------------------------------------------------------------------------------
# Docker
# ------------------------------------------------------------------------------
docker-up:
	@echo "Starting Docker services..."
	docker compose up --build -d

docker-down:
	@echo "Stopping Docker services..."
	docker compose down

docker-logs:
	@echo "Showing Docker logs..."
	docker compose logs -f app

docker-clean:
	@echo "Cleaning Docker resources..."
	docker compose down -v --remove-orphans
	docker system prune -f

# ------------------------------------------------------------------------------
# Database Migrations (using goose)
# ------------------------------------------------------------------------------
migrate-up:
	@echo "Running migrations up..."
	goose -dir $(MIGRATIONS_DIR) postgres $(DATABASE_URL) up

migrate-down:
	@echo "Running migrations down..."
	goose -dir $(MIGRATIONS_DIR) postgres $(DATABASE_URL) down

migrate-status:
	@echo "Checking migration status..."
	goose -dir $(MIGRATIONS_DIR) postgres $(DATABASE_URL) status

migrate-create:
	@read -p "Migration name: " name; \
	goose -dir $(MIGRATIONS_DIR) create $$name sql

# ------------------------------------------------------------------------------
# Seeding
# ------------------------------------------------------------------------------
seed:
	@echo "Seeding database..."
	go run ./cmd/seed/main.go

# ------------------------------------------------------------------------------
# Linting
# ------------------------------------------------------------------------------
lint:
	@echo "Running linter..."
	golangci-lint run ./...

lint-fix:
	@echo "Running linter with auto-fix..."
	golangci-lint run --fix ./...

fmt:
	@echo "Formatting code..."
	go fmt ./...
	gofumpt -w $(GO_FILES)

vet:
	@echo "Running go vet..."
	go vet ./...

# ------------------------------------------------------------------------------
# Dependencies
# ------------------------------------------------------------------------------
deps:
	@echo "Downloading dependencies..."
	go mod download

deps-update:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

deps-tidy:
	@echo "Tidying dependencies..."
	go mod tidy
	go mod verify

# ------------------------------------------------------------------------------
# Clean
# ------------------------------------------------------------------------------
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR) coverage.out coverage.html
	@echo "Clean complete"

# ------------------------------------------------------------------------------
# Security Scan
# ------------------------------------------------------------------------------
security:
	@echo "Running security scan..."
	gosec ./...

# ------------------------------------------------------------------------------
# Help
# ------------------------------------------------------------------------------
help:
	@echo "GuestFlow - Available Targets:"
	@echo ""
	@echo "  build         Build the server binary"
	@echo "  test          Run all tests with race detection and coverage"
	@echo "  test-short    Run short tests only"
	@echo "  test-coverage Run tests and generate HTML coverage report"
	@echo "  dev           Start development environment"
	@echo "  fresh         Reset volumes and rebuild the full local stack"
	@echo "  docker-up     Start all Docker services"
	@echo "  docker-down   Stop all Docker services"
	@echo "  docker-logs   Follow application logs"
	@echo "  docker-clean  Remove all containers, volumes, and images"
	@echo "  migrate-up    Run database migrations (up)"
	@echo "  migrate-down  Rollback last database migration"
	@echo "  migrate-status Show current migration status"
	@echo "  migrate-create Create a new migration file"
	@echo "  lint          Run linters"
	@echo "  lint-fix      Run linters with auto-fix"
	@echo "  fmt           Format Go code"
	@echo "  vet           Run go vet"
	@echo "  deps          Download dependencies"
	@echo "  deps-update   Update all dependencies"
	@echo "  deps-tidy     Tidy go.mod and go.sum"
	@echo "  clean         Remove build artifacts"
	@echo "  security      Run security scan"
	@echo "  help          Show this help message"
