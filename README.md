# GuestFlow

**Guest Relationship Management and Invitation Service Platform**

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-MVP%20v1.0-green.svg)]()

A comprehensive SaaS platform for managing guest invitations, RSVPs, check-ins, and event-day services. Built for weddings, corporate events, government functions, and VIP/protocol events — starting with the Indonesian market.

---

## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [Quick Start](#quick-start)
- [API Documentation](#api-documentation)
- [Project Structure](#project-structure)
- [Technology Stack](#technology-stack)
- [Configuration](#configuration)
- [Development](#development)
- [Deployment](#deployment)
- [Testing](#testing)
- [Security](#security)
- [Compliance](#compliance)
- [Contributing](#contributing)
- [Roadmap](#roadmap)

---

## Features

### Core Modules (MVP v1.0)

| Module | Status | Description |
|--------|--------|-------------|
| **Authentication** | ✅ | JWT with refresh token rotation, bcrypt hashing, MFA-ready |
| **Multi-Tenancy** | ✅ | Tenant isolation via PostgreSQL RLS, organization management |
| **RBAC** | ✅ | 7 roles, 32 permissions — tenant and event-scoped access |
| **Audit Logging** | ✅ | Comprehensive mutation tracking for compliance |
| **Event Management** | ✅ | CRUD, multi-session (Akad, Resepsi, etc.), status workflow |
| **Guest Management** | ✅ | CRUD, household grouping, tags, CSV import/export, duplicate detection |
| **Invitation & QR** | ✅ | Opaque token generation (256-bit), SHA-256 hash storage, no PII in QR |
| **RSVP** | ✅ | Public form, capacity validation, deadline check, multi-session support |
| **Check-in** | ✅ | QR scan, manual search, walk-in registration, real-time stats |
| **Seating** | ✅ | Table CRUD, seat assignment, auto-assign, occupancy tracking |
| **Communication** | ✅ | Template management, WhatsApp/Email campaigns, variable substitution |
| **Dashboard** | ✅ | Aggregated real-time metrics (RSVP, check-in, seating) |
| **Invitation Microsite** | ✅ | Public-facing mobile-first site at `/i/{token}` with RSVP form |
| **Admin Dashboard** | ✅ | Web-based admin UI with stats, tables, modals |

### Security Features

- JWT authentication with 15-min access / 7-day refresh tokens
- bcrypt password hashing (cost 12)
- Token rotation on refresh
- SHA-256 hash for opaque invitation tokens
- Rate limiting per IP and per user
- CORS protection
- Security headers (XSS, CSRF, HSTS)
- Structured audit logging
- Tenant isolation at query layer

### Compliance

- **UU PDP Indonesia No. 27/2022** — Privacy by Design implementation
- **OWASP ASVS 4.0.3 Level 2** — Application security verification
- **WCAG 2.2 AA** — Accessibility compliance (invitation microsite)
- Data minimization, consent management, retention policies

---

## Architecture

### System Overview

```
                    ┌─────────────────┐
                    │   Nginx (SSL)   │
                    └────────┬────────┘
                             │
          ┌──────────────────┼──────────────────┐
          ▼                  ▼                  ▼
   ┌────────────┐   ┌──────────────┐   ┌──────────────┐
   │  Web App   │   │    API       │   │  Invitation  │
   │  (Admin)   │   │  (/api/v1)   │   │  (/i/{token})│
   └────────────┘   └──────┬───────┘   └──────────────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
        ┌─────────┐ ┌──────────┐ ┌──────────┐
        │PostgreSQL│ │  Redis   │ │  S3/MinIO│
        │   16    │ │    7     │ │  (Files) │
        └─────────┘ └──────────┘ └──────────┘
```

### Modular Monolith

```
guestflow/
├── internal/
│   ├── auth/          # JWT, password hashing
│   ├── rbac/          # Role-based access control
│   ├── audit/         # Audit logging
│   ├── domain/        # Domain models ( structs only)
│   ├── repository/    # Database access (sqlx)
│   ├── service/       # Business logic
│   ├── handler/       # HTTP handlers (JSON + HTML)
│   └── middleware/    # Echo middleware
├── pkg/               # Shared packages
├── web/               # Static assets & templates
├── migrations/        # Goose SQL migrations
└── tests/             # Feature tests
```

---

## Quick Start

### Prerequisites

- **Docker** 24+ and Docker Compose 2.20+
- **Go** 1.22+ (for local development)
- **Make**

### Docker Compose (Recommended)

```bash
# Clone the repository
git clone https://github.com/guestflow/guestflow.git
cd guestflow

# Copy environment file
cp .env.example .env

# Start all services (PostgreSQL, Redis, Go app, Nginx)
make docker-up

# Run migrations
docker compose exec app make migrate-up

# Seed demo data
docker compose exec -T db psql -U guestflow -d guestflow < migrations/999_seed_data.up.sql

# Access the application
# API:     http://localhost:8080/api/v1
# Admin:   http://localhost:8080/admin
# Health:  http://localhost:8080/health
```

### Local Development

```bash
# Install Go dependencies
go mod download

# Start infrastructure (PostgreSQL + Redis)
docker compose up -d db redis

# Set environment variables
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=guestflow
export DB_USER=guestflow
export DB_PASSWORD=guestflow
export REDIS_HOST=localhost
export REDIS_PORT=6379

# Run migrations
go run cmd/migrate/main.go up

# Start server
go run cmd/server/main.go

# Or use Makefile
make dev
```

### Demo Credentials

All accounts below use password `password123` and belong to tenant `demo-wo`.
These accounts are for local/demo testing only and must not be used in production.

| Role | Email |
|------|-------|
| Tenant Owner | `owner@guestflow.id` |
| Event Manager | `manager@guestflow.id` |
| RSVP Officer | `rsvp@guestflow.id` |
| Registration Officer | `registration@guestflow.id` |
| Usher | `usher@guestflow.id` |
| Gift Officer | `gift@guestflow.id` |
| Viewer | `viewer@guestflow.id` |

The original workspace account remains available as `demo@guestflow.id` with the
same password. Its tenant membership is `Tenant Owner`.

Tenant membership determines who can access the workspace. Operational roles can
then be assigned to individual events through `event_members`.

### Role Scope

- `tenant_owner`: full access across the tenant and all events.
- `event_manager`: manages events and event staff across the tenant, without changing tenant membership roles.
- `rsvp_officer`: event-scoped access for invitations, RSVP, and communication.
- `registration_officer`: event-scoped access for guest registration and check-in.
- `usher`: event-scoped access for check-in and seating visibility.
- `gift_officer`: event-scoped read access for guest and event reporting.
- `viewer`: event-scoped read-only access; can view event assignments when assigned.

Existing event-scoped memberships are backfilled during migration `1001_event_members`.
New event-scoped users must be assigned from **Tim Acara** in the UI before they can
open that event. The API exposes the effective role through:

```text
GET /api/v1/tenants/:tenantId/events/:eventId/members/access
GET /api/v1/tenants/:tenantId/events/:eventId/members
POST /api/v1/tenants/:tenantId/events/:eventId/members
```

To inspect demo tenant memberships:

```bash
curl http://localhost:8080/api/v1/tenants/TENANT_ID/users \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "X-Tenant-ID: TENANT_ID"
```

### API Quick Test

```bash
# Register
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123","full_name":"Test User"}'

# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"demo@guestflow.id","password":"password123"}'

# Request password reset or passwordless login link
curl -X POST http://localhost:8080/api/v1/auth/forgot-password \
  -H "Content-Type: application/json" \
  -d '{"email":"member@example.com"}'
curl -X POST http://localhost:8080/api/v1/auth/magic-link \
  -H "Content-Type: application/json" \
  -d '{"email":"member@example.com"}'

# List events (with Bearer token)
curl http://localhost:8080/api/v1/tenants/TENANT_ID/events \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "X-Tenant-ID: TENANT_ID"
```

---

## API Documentation

OpenAPI 3.0 specification available at `docs/api/openapi.yaml`.

### Key Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/auth/register` | User registration |
| `POST` | `/api/v1/auth/login` | User login |
| `POST` | `/api/v1/auth/refresh` | Refresh token |
| `GET`  | `/api/v1/auth/verify-email` | Verify registration email |
| `POST` | `/api/v1/auth/resend-verification` | Resend registration verification |
| `POST` | `/api/v1/auth/forgot-password` | Send password reset link |
| `POST` | `/api/v1/auth/reset-password` | Set a new password from a reset token |
| `POST` | `/api/v1/auth/magic-link` | Send passwordless login link |
| `POST` | `/api/v1/auth/magic-link/consume` | Consume passwordless login token |
| `POST` | `/api/v1/auth/logout` | Logout |
| `GET`  | `/api/v1/auth/me` | Current user |
| `POST` | `/api/v1/tenants` | Create tenant |
| `GET`  | `/api/v1/tenants/:id` | Get tenant |
| `GET`  | `/api/v1/tenants/:id/events` | List events |
| `POST` | `/api/v1/tenants/:id/events` | Create event |
| `GET`  | `/api/v1/tenants/:id/events/:eventId/members/access` | Get effective event access |
| `GET`  | `/api/v1/tenants/:id/events/:eventId/members` | List assigned event staff |
| `POST` | `/api/v1/tenants/:id/events/:eventId/members` | Assign staff to an event |
| `GET`  | `/api/v1/tenants/:id/guests` | List guests |
| `POST` | `/api/v1/tenants/:id/guests` | Create guest |
| `POST` | `/api/v1/tenants/:id/guests/import` | Import CSV |
| `POST` | `/api/v1/tenants/:id/events/:eventId/invitations` | Create invitations |
| `GET`  | `/api/v1/tenants/:id/events/:eventId/invitations/:invitationId/qr` | Get QR code |
| `POST` | `/api/v1/tenants/:id/templates/defaults` | Generate default WhatsApp and email invitation templates |
| `POST` | `/api/v1/tenants/:id/events/:eventId/messages/send` | Send a WhatsApp template to one or more guests |
| `POST` | `/api/v1/tenants/:id/events/:eventId/campaigns/:campaignId/launch` | Send a campaign to filtered guests |
| `POST` | `/api/v1/rsvp` | Submit RSVP (public) |
| `GET`  | `/api/v1/tenants/:id/events/:eventId/rsvp/dashboard` | RSVP dashboard |
| `POST` | `/api/v1/tenants/:id/events/:eventId/checkin` | Process check-in |
| `GET`  | `/api/v1/tenants/:id/events/:eventId/checkin/stats` | Check-in stats |
| `POST` | `/api/v1/tenants/:id/events/:eventId/checkin/walkin` | Walk-in registration |
| `GET`  | `/api/v1/tenants/:id/events/:eventId/dashboard` | Full dashboard |
| `GET`  | `/i/:token` | **Invitation microsite (public)** |
| `GET`  | `/admin` | **Admin dashboard** |
| `GET`  | `/health` | Health check |

### Authentication

All protected endpoints require:
- `Authorization: Bearer <access_token>` header
- `X-Tenant-ID: <tenant_uuid>` header

### WhatsApp Delivery Flow

WhatsApp delivery is available from the guest detail action, invitation row action, selected invitations, and communication campaigns. All paths use the same `messages/send` service so every attempt is logged per guest in `communication_messages`. The invitation lifecycle (`draft`, `opened`, `responded`, `revoked`) is separate from delivery status (`not_sent`, `queued`, `sent`, `failed`, `delivered`, `read`). A `sent` message means Blastr accepted the request at the recorded `provider_http_status`; `delivered` and `read` require provider receipts.

Every new tenant is automatically provisioned with the standard WhatsApp and email invitation templates. Adding another member to an existing tenant does not create duplicates; the generator endpoint can be called again safely if a tenant needs to repair or restore its defaults.

```json
{
  "guest_ids": ["GUEST_UUID"],
  "template_id": "WHATSAPP_TEMPLATE_UUID"
}
```

Before a batch is sent, the API verifies that every guest belongs to the selected event and has a valid WhatsApp number. An empty or invalid number returns `422` and no message in that batch is sent. Provider credentials that are missing or disabled return `503`.

---

## Project Structure

```
guestflow/
├── cmd/
│   ├── server/              # HTTP server entry point
│   │   └── main.go          # 5-layer DI: Config → Infra → Repo → Service → Handler
│   └── migrate/             # Database migration tool
│
├── internal/
│   ├── auth/                # JWT service, password hashing, refresh tokens
│   ├── rbac/                # RBAC service with ServiceInterface pattern
│   ├── audit/               # Audit logging service
│   ├── config/              # Viper-based configuration
│   ├── domain/              # 15 domain model files (event, guest, rsvp, etc.)
│   ├── repository/          # 19 repository files (sqlx + PostgreSQL)
│   ├── service/             # 14 service files (business logic)
│   ├── handler/             # 13 handler files (JSON API + HTML views)
│   ├── middleware/          # Auth, tenant, rate limit, logger middleware
│   └── validator/           # Echo-compatible request validator
│
├── pkg/
│   ├── crypto/              # Password hashing, token generation
│   ├── errors/              # Custom error types with codes
│   └── response/            # Standard API response helpers
│
├── web/
│   ├── static/
│   │   ├── css/
│   │   │   ├── invitation.css   # Mobile-first invitation styles
│   │   │   └── admin.css      # Admin dashboard styles
│   │   └── js/
│   │       └── invitation.js   # Countdown, RSVP form, animations
│   └── templates/
│       ├── invitation.html      # Guest-facing invitation microsite
│       └── admin.html          # Admin dashboard SPA
│
├── migrations/              # 21 up/down migration pairs (42 files)
│   ├── 001_users.up.sql
│   ├── 001_users.down.sql
│   └── ...
│
├── tests/
│   └── feature/             # Feature/end-to-end tests
│       ├── health_test.go
│       └── auth_test.go
│
├── docker-compose.yml       # PostgreSQL + Redis + Go + Nginx
├── Dockerfile               # Multi-stage Go build
├── Makefile                 # Build, test, migrate commands
├── .env.example             # Configuration template
└── go.mod / go.sum          # Go module definitions
```

---

## Technology Stack

| Layer | Technology | Version |
|-------|-----------|---------|
| Language | Go | 1.22+ |
| Web Framework | Echo | v4.12 |
| Database | PostgreSQL | 16 |
| Cache/Queue | Redis | 7 |
| SQL | sqlx + goose | latest |
| Auth | JWT (golang-jwt) | v5 |
| Validation | go-playground/validator | v10 |
| Config | Viper | v1.19 |
| Frontend | HTML + CSS + JS | (server-rendered) |
| Container | Docker + Compose | 24+ |
| Proxy | Nginx | 1.24+ |

---

## Configuration

All configuration is via environment variables. Copy `.env.example` to `.env` and customize.

### Required Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `APP_ENV` | Environment (development/staging/production) | `development` |
| `APP_PORT` | Server port | `8080` |
| `DB_HOST` | PostgreSQL host | `localhost` |
| `DB_PORT` | PostgreSQL port | `5432` |
| `DB_NAME` | Database name | `guestflow` |
| `DB_USER` | Database user | `guestflow` |
| `DB_PASSWORD` | Database password | *(required)* |
| `REDIS_HOST` | Redis host | `localhost` |
| `REDIS_PORT` | Redis port | `6379` |
| `JWT_ACCESS_SECRET` | JWT access token secret | *(required)* |
| `JWT_REFRESH_SECRET` | JWT refresh token secret | *(required)* |
| `WHATSAPP_ENABLED` | Enable Blastr WhatsApp delivery | `false` |
| `WHATSAPP_API_URL` | Blastr send endpoint | `https://app.blastr.id/api/pub/send` |
| `WHATSAPP_ACCOUNT_TOKEN` | Blastr account bearer token | *(required when enabled)* |
| `WHATSAPP_SENDER_TOKEN` | Blastr sender token | *(required when enabled)* |

See `.env.example` for full configuration options.

---

## Development

### Available Commands (Makefile)

```bash
make build          # Build the Go binary
make run            # Run the server
make test           # Run all tests
make test-unit      # Run unit tests only
make test-feature   # Run feature tests only
make migrate-up     # Run database migrations
make migrate-down   # Rollback migrations
make seed           # Insert demo data
make docker-up      # Start Docker Compose
make docker-down    # Stop Docker Compose
make lint           # Run linter
make fmt            # Format Go code
```

### Database Migrations

Uses [goose](https://github.com/pressly/goose) for migration management.

```bash
# Create new migration
goose -dir migrations create add_guest_notes

# Run migrations
go run cmd/migrate/main.go up

# Rollback one
go run cmd/migrate/main.go down

# Status
go run cmd/migrate/main.go status
```

### Adding a New Module

1. **Domain**: Add structs to `internal/domain/your_model.go`
2. **Migration**: Create `migrations/XXX_your_table.up.sql` and `.down.sql`
3. **Repository**: Create `internal/repository/your_repository.go`
4. **Service**: Create `internal/service/your_service.go`
5. **Handler**: Create `internal/handler/your_handler.go`
6. **Routes**: Register in `internal/handler/routes.go`
7. **DI**: Wire in `cmd/server/main.go`

---

## Deployment

### Docker (Recommended)

```bash
# Build and start
make docker-up

# View logs
docker compose logs -f app

# Scale workers
docker compose up -d --scale worker=3
```

### Environment Variables for Production

| Variable | Production Value |
|----------|-----------------|
| `APP_ENV` | `production` |
| `DB_SSL_MODE` | `require` |
| `REDIS_PASSWORD` | *(strong password)* |
| `JWT_ACCESS_TTL` | `15m` |
| `JWT_REFRESH_TTL` | `168h` |
| `RATE_LIMIT_RPS` | `100` |

### Health Checks

- `GET /health` — Liveness probe
- `GET /ready` — Readiness probe (checks DB + Redis)

---

## Testing

```bash
# Run all tests
make test

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific test
go test -v ./internal/auth/...
go test -v ./tests/feature/...

# Race detection
go test -race ./...
```

### Test Structure

| Type | Location | Scope |
|------|----------|-------|
| Unit | `internal/*/...` | Individual functions |
| Feature | `tests/feature/...` | HTTP endpoints |
| Load | `tests/load/...` | Performance (future) |

---

## Security

See `docs/architecture/security.md` for detailed security documentation.

### Key Security Measures

- **Authentication**: JWT with short-lived access tokens and refresh token rotation
- **Authorization**: RBAC with 7 roles and fine-grained permissions
- **Data Protection**: AES-256 encryption at rest for PII, TLS in transit
- **Token Security**: 256-bit opaque tokens, SHA-256 hash storage, revocable
- **Input Validation**: Server-side validation on all inputs
- **Rate Limiting**: Per-IP and per-user rate limits via Redis
- **Audit**: All mutations logged with user, timestamp, and changes

---

## Compliance

### UU PDP Indonesia No. 27/2022

- ✅ Privacy notice per event
- ✅ Consent management (communication consent tracked per guest)
- ✅ Data minimization (only required fields collected)
- ✅ Purpose specification (data used only for event purposes)
- ✅ Retention policies (configurable per tenant)
- ✅ Audit trail (comprehensive logging)

### OWASP ASVS 4.0.3

Target: **Level 2** (Application handling sensitive data)

- V1: Architecture ✅ (Modular design, defense in depth)
- V2: Authentication ✅ (JWT, bcrypt, MFA-ready)
- V3: Session Management ✅ (Refresh token rotation)
- V4: Access Control ✅ (RBAC, tenant isolation)
- V5: Validation ✅ (Server-side, parameterized queries)
- V6: Cryptography ✅ (bcrypt, AES-256)
- V7: Error Handling ✅ (No sensitive data in errors)
- V8: Data Protection ✅ (Encryption at rest + in transit)
- V9: Communication ✅ (TLS 1.3)
- V12: File Upload ✅ (Validation planned)

### WCAG 2.2 AA

- ✅ Keyboard accessible navigation
- ✅ Color contrast ratio ≥ 4.5:1
- ✅ Reduced motion support (`prefers-reduced-motion`)
- ✅ Form labels and error messages
- ✅ Focus indicators

---

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/your-feature`
3. Commit your changes: `git commit -am 'feat: Add new feature'`
4. Push to the branch: `git push origin feature/your-feature`
5. Submit a pull request

### Code Standards

- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` for formatting
- Write tests for new features
- Update documentation for API changes

---

## Roadmap

### MVP v1.0 ✅ (Current)
- Core modules: Auth, Tenant, Event, Guest, Invitation, RSVP, Check-in, Seating, Communication, Dashboard
- Invitation microsite and admin dashboard
- Docker development environment

### Phase 2 (Planned)
- [x] Blastr WhatsApp delivery integration with per-guest and batch actions
- [ ] Automated reminder scheduling
- [ ] Offline-first PWA check-in scanner
- [ ] Thermal label printing
- [ ] Advanced seating planner (drag-and-drop)
- [ ] Usher dashboard and welcome screen
- [ ] Souvenir tracking
- [ ] Multi-session event support
- [ ] Waitlist management
- [ ] White-label and custom domain
- [ ] Billing and subscription management
- [ ] Advanced analytics

### Phase 3 (Future)
- [ ] Self-service kiosk mode
- [ ] Selfie check-in
- [ ] Guest photo gallery
- [ ] Gift and angpao management
- [ ] AI attendance prediction
- [ ] Staffing recommendation
- [ ] Vendor marketplace
- [ ] RFID/NFC support
- [ ] Enterprise SSO (SAML/OIDC)
- [ ] Public API and integration marketplace

---

## License

MIT License. See [LICENSE](LICENSE) for details.

---

## Support

- 📧 Email: support@guestflow.id
- 🐛 Issues: [GitHub Issues](https://github.com/guestflow/guestflow/issues)
- 📖 Documentation: See `docs/` directory

---

Built with ❤️ in Indonesia for the world.
