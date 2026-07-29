# GuestFlow Backend — Agent Guide

This document is intended for AI coding agents working on the **GuestFlow Backend** project. It describes the architecture, conventions, build/test process, and deployment setup as it actually exists in the repository today.

---

## Project Overview

GuestFlow Backend is a Go 1.22 modular monolith that powers the GuestFlow platform — a guest-management SaaS for weddings, corporate events, government functions, and VIP/protocol events. It exposes a REST API (`/api/v1`), serves public invitation microsites (`/i/{token}`), and provides an admin HTML dashboard via HTMX fragments.

The platform is multi-tenant: data is scoped by tenant (workspace) and often further by event. Authorization is role-based (RBAC) with tenant-scoped and event-scoped permissions.

User-facing API responses and HTML fragments are mostly in **Bahasa Indonesia**. Source-code comments, identifiers, and commit messages should remain in English.

---

## Technology Stack

- **Language:** Go 1.22
- **HTTP Framework:** Echo v4 (labstack/echo)
- **Database:** PostgreSQL 16 (accessed via `sqlx` + `pgx`)
- **Migrations:** Goose (`pressly/goose/v3`)
- **Cache / Queue:** Redis 7 (`go-redis/v9`)
- **Authentication:** JWT (`golang-jwt/jwt/v5`) + bcrypt (`golang.org/x/crypto`)
- **Validation:** `go-playground/validator/v10`
- **Configuration:** Viper (`spf13/viper`) with `.env` support
- **WhatsApp:** GOWA (self-hosted WhatsApp Web REST API) via `go-whatsapp-web-multidevice`
- **Email:** SMTP (Mailpit in development)
- **Payments:** Midtrans (`midtrans-go`)
- **QR Codes:** `skip2/go-qrcode`
- **Templates:** Go `html/template` (invitation microsite + admin/HTMX fragments)
- **Hot Reload:** Air (`.air.toml`)
- **Testing:** `go test` (unit + feature tests with `testify`)
- **Linting:** `golangci-lint` (via Makefile) and `go vet`

---

## Repository Layout

```
guestbook/
├── cmd/
│   ├── server/main.go         # HTTP server entry point (dependency injection + routes)
│   ├── migrate/main.go          # Goose migration runner binary
│   └── worker/main.go           # Background queue worker entry point
├── internal/
│   ├── auth/                    # JWT service, password hashing, refresh tokens
│   ├── rbac/                    # Role-based access control service
│   ├── audit/                   # Audit logging service
│   ├── config/                  # Viper-based configuration structs and loader
│   ├── domain/                  # Domain models, roles, permissions, validation helpers
│   ├── repository/              # Data access layer (sqlx + PostgreSQL, Redis)
│   ├── service/                 # Business logic layer
│   ├── handler/                 # HTTP handlers (JSON API + HTML templates + HTMX)
│   ├── middleware/              # Echo middleware (JWT, RBAC, tenant resolver, rate limit, CORS, etc.)
│   ├── security/                # Security utilities
│   ├── email/                   # SMTP mailer
│   ├── payment/                 # Midtrans client
│   ├── whatsapp/                # GOWA WhatsApp client
│   └── worker/                  # Background cron jobs (e.g. billing)
├── pkg/                         # Shared packages usable outside the app
│   ├── crypto/                  # Encryption helpers
│   ├── errors/                  # Domain / HTTP error types
│   └── response/                # Standard JSON response helpers
├── web/                         # Static assets and Go HTML templates
│   ├── static/                  # CSS, JS, vendor libraries (e.g. ZXing for QR scanning)
│   └── templates/               # Admin dashboard and invitation templates
├── migrations/                  # Goose SQL migrations (83+ files)
├── tests/                       # Feature / end-to-end tests
├── scripts/                     # Development helper scripts
├── bin/                         # Build output directory
├── docs/                        # Additional documentation
├── nginx/                       # Nginx config for local development
├── deploy/                      # Deployment scripts
├── Makefile                     # Common dev tasks (build, test, lint, migrate, etc.)
├── Dockerfile                   # Multi-stage build: builder → distroless
├── docker-compose.yml           # Full development stack (PostgreSQL, Redis, GOWA, app, nginx, mailpit)
├── docker-compose.production.yml# Production stack (PostgreSQL, Redis, GOWA, app, migrate)
├── .air.toml                    # Air hot-reload configuration
├── .env.example                 # Example environment variables
└── .github/workflows/
    ├── deploy-production.yml    # CI/CD: test, SSH deploy, docker compose up
    └── notify-whatsapp.yml      # Sends GitHub issue events to GuestFlow webhook
```

---

## Build and Development Commands

All commands below are run from the repository root unless noted.

| Command | Description |
|---------|-------------|
| `go mod download` | Download Go dependencies |
| `make build` | Build `bin/server` from `cmd/server/main.go` with version ldflags |
| `make test` | Run all tests with race detection and coverage report |
| `make test-short` | Run short tests only |
| `make dev` | Start development environment (runs `scripts/dev.sh`) |
| `make dev-hot` | Start server with Air hot reload (`air`) |
| `make docker-up` | Start full Docker development stack (`docker compose up --build -d`) |
| `make docker-down` | Stop Docker stack |
| `make migrate-up` | Run Goose migrations up against configured PostgreSQL |
| `make migrate-down` | Roll back last Goose migration |
| `make migrate-create` | Create a new SQL migration file interactively |
| `make seed` | Run seed script (`go run ./cmd/seed/main.go`) |
| `make lint` | Run `golangci-lint run ./...` |
| `make fmt` | Format with `go fmt` and `gofumpt` |
| `make vet` | Run `go vet ./...` |
| `make security` | Run `gosec ./...` |
| `make clean` | Remove build artifacts and coverage files |

### Typical Local Development

```bash
cp .env.example .env
# edit .env as needed
make docker-up
# wait for services to be healthy, then:
docker compose exec app make migrate-up
docker compose exec -T db psql -U guestflow -d guestflow < migrations/999_seed_data.up.sql
# API: http://localhost:8080/api/v1
# Admin: http://localhost:8080/admin
# Health: http://localhost:8080/health
```

---

## Runtime Architecture

### Server Entry Point (`cmd/server/main.go`)

The server boots via explicit dependency injection:

```
Config → DB/Redis → Repositories → Services → Handlers → Echo Routes
```

Key setup steps:
1. Load configuration via `config.Load()`.
2. Connect to PostgreSQL (`repository.NewPostgresConnection`) and Redis (`repository.NewRedisConnection`).
3. Build the service layer (`auth`, `rbac`, `tenant`, `event`, `guest`, `invitation`, `rsvp`, `checkin`, `seating`, `communication`, `billing`, `whatsapp`, `dashboard`, etc.).
4. Instantiate handlers and register routes via `handler.RegisterRoutes(e, ...)`. Starts background cron jobs (`worker.StartCronJob`).
5. Start the Echo server with graceful shutdown on `SIGINT`/`SIGTERM`.

### Middleware Stack (order matters)

1. Recovery
2. Request ID
3. Logger
4. CORS (`AllowCredentials: true`, `X-Tenant-ID` added to allowed headers)
5. Rate limiting (Redis-backed, configurable RPS/burst/window)
6. Body limit (`10M`)
7. Gzip (skipped for `/api/*` and `/health` to avoid stream issues)
8. Secure headers (HSTS, CSP, XSS, frame options, content-type nosniff)
9. Echo validator (`internal/validator.New()`)

### Route Layout

- **Public API:** `/api/v1/auth/*`, `/api/v1/billing/plans`, `/api/v1/billing/webhook`, `/api/v1/rsvp`, `/api/v1/webhooks/*`, `/health`, `/healthz`, `/ready`.
- **Protected API:** `/api/v1/*` routes after `middleware.JWTAuth`.
- **Tenant-scoped routes:** nested under `/api/v1/tenants/:id/...` (guests, templates, integrations, billing).
- **Event-scoped routes:** nested under `/api/v1/tenants/:id/events/:eventId/...` (event guests, gifts, invitations, RSVP, check-in, seating, messages, dashboard).
- **Public invitation site:** `/i/{token}` (handled by `invitationSiteHandler.RegisterSiteRoutes`).
- **Static assets:** `/static/*` → `web/static`.
- **HTMX admin dashboard:** HTML fragments rendered via `handler.NewTemplateRenderer()` and registered by `htmxDashboardHandler.RegisterHTMXRoutes`.

### Multi-Tenancy and Authorization

- **Tenant header:** `X-Tenant-ID` (configurable via `TENANT_HEADER`).
- **Tenant resolver middleware:** resolves `tenantId` from the header or URL parameter and validates membership.
- **RBAC roles:**
  - `tenant_owner` — full tenant access
  - `event_manager` — manage events and event staff
  - `rsvp_officer` — invitations, RSVP, communication
  - `registration_officer` — guest registration and check-in
  - `usher` — check-in and seating visibility
  - `gift_officer` — gift records + reporting
  - `viewer` — read-only event access
- **Permissions:** fine-grained constants like `guest:read`, `event:write`, `communication:send`, `billing:read`, etc. See `internal/domain/role.go`.
- **Event access:** `event_members` table maps users to events with an operational role. `EventAccessService` resolves effective permissions per event.

### Background Workers

- `cmd/worker/main.go` is the background worker entry point.
- `internal/worker/cron.go` schedules cron jobs (e.g. billing-related tasks).
- Workers use the same service layer but run as separate containers in production.

---

## Backend Conventions

- **Database schema:** managed by Goose migrations in `migrations/`.
- **Snake_case in DB and JSON:** the backend stores and returns `snake_case` field names. The frontend normalizes responses to `camelCase`.
- **Domain structs:** in `internal/domain/` embed a `Base` struct with `id`, `created_at`, `updated_at`, etc.
- **UUIDs:** primary keys use `github.com/google/uuid`.
- **Error handling:** domain errors are mapped to HTTP responses via `pkg/errors` and `pkg/response`. Handlers return consistent JSON envelopes.
- **Audit logging:** mutating operations are logged through `internal/audit` for compliance.
- **Security:**
  - JWT access tokens expire in 15 minutes, refresh tokens in 7 days.
  - bcrypt cost 12 for password hashing.
  - Invitation tokens are opaque 256-bit values; stored as SHA-256 hashes.
  - Rate limiting per IP and per user.
  - CORS allowed origins are explicit in production.

---

## Testing Instructions

- Unit tests are co-located with source files (`*_test.go` in `internal/`).
- Feature / end-to-end tests live in `tests/feature/`.
- Run all tests: `make test` (or `go test -race -count=1 ./...`).
- Run short tests: `make test-short`.
- Generate coverage HTML: `make test-coverage`.
- There are no separate mock contracts; tests use testify and real in-memory dependencies where appropriate.

---

## Deployment

### 1. GitHub Actions (Primary)

`.github/workflows/deploy-production.yml` runs on every push to `master` (and supports `workflow_dispatch`):

1. Checks out the repo.
2. Sets up Go from `go.mod`.
3. Runs `go test ./...`.
4. Deploys via SSH to `/home/ubuntu/apps/guestflow/backend`.
5. Builds `app` and `migrate` images, runs migrations, restarts the app container, and prunes old worker containers.
6. Health-checks `http://127.0.0.1:18080/health`.

Required GitHub secrets: `DEPLOY_HOST`, `DEPLOY_PORT`, `DEPLOY_USER`, `DEPLOY_SSH_KEY`, `DEPLOY_KNOWN_HOSTS`.

### 2. Docker / Docker Compose (Production)

- `docker-compose.production.yml` defines PostgreSQL, Redis, GOWA, `app` (port `18080` on loopback), and a `migrate` job.
- `Dockerfile` builds three binaries (`server`, `migrate`, `worker`) in a multi-stage build and copies them into a distroless `gcr.io/distroless/static:nonroot` image.
- Migrations run as a one-off container before the app starts.
- GOWA is used for WhatsApp integration; credentials come from `.env`.

### 3. Environment Variables

Copy `.env.example` to `.env` and adjust. Key variables:
- `APP_PUBLIC_URL` — public URL used for invitation links and callbacks.
- `DB_*`, `REDIS_*` — database and cache connection.
- `JWT_SECRET`, `JWT_ACCESS_TOKEN_EXPIRY`, `JWT_REFRESH_TOKEN_EXPIRY`.
- `CORS_ALLOWED_ORIGINS` — comma-separated allowed origins.
- `WHATSAPP_ENABLED`, `WHATSAPP_GOWA_*` — GOWA WhatsApp integration.
- `EMAIL_ENABLED`, `EMAIL_*` — SMTP configuration.
- `MIDTRANS_*` — payment gateway.
- `GITHUB_ISSUE_*` — development-only GitHub → WhatsApp issue notifications.

Never commit real secrets; `.env` is git-ignored.

---

## Known Issues / Notes

- The default branch is `master` (not `main`). CI triggers on pushes to `master`.
- GOWA WhatsApp integration requires a paired device and is disabled by default (`WHATSAPP_ENABLED=false`).
- Campaigns feature is gated by `CAMPAIGNS_ENABLED` and stays off by default while under research.
- The `web/static` and `web/templates` folders are embedded/served by the Go binary; the React admin UI is served separately (see `guestbook-ui` repo).
- The production compose does not expose MinIO; storage is currently local (`STORAGE_PROVIDER=local`) unless S3-compatible settings are configured.

---

## Quick Reference for Agents

- Start backend: `make docker-up` then `docker compose exec app make migrate-up`
- Run tests: `make test`
- Add a new API endpoint: add handler in `internal/handler/`, wire it in `internal/handler/routes.go`, apply RBAC middleware, and add a service method if needed.
- Add a new domain model: define it in `internal/domain/`, create a repository in `internal/repository/`, and add a service in `internal/service/`.
- Add a migration: `make migrate-create` (produces `migrations/{timestamp}_{name}.sql`).
- User-facing text in API errors and HTML templates: write in **Bahasa Indonesia**.
- Database columns and JSON payloads: use **snake_case**.
- Always verify RBAC permissions and tenant/event scoping when adding new routes.
