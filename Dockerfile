# GuestFlow - Multi-stage Dockerfile
# Build stages: builder -> final

# ------------------------------------------------------------------------------
# Builder Stage
# ------------------------------------------------------------------------------
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy dependency files first for better layer caching
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the server binary with optimizations
# CGO_ENABLED=0 for static binary, ldflags to strip debug info
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o bin/server \
    ./cmd/server/main.go

# Build the migration binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o bin/migrate \
    ./cmd/migrate/main.go

# ------------------------------------------------------------------------------
# Final Stage
# ------------------------------------------------------------------------------
FROM gcr.io/distroless/static:nonroot AS final

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/bin/server /app/server
COPY --from=builder /build/bin/migrate /app/migrate

# Copy migration files
COPY --from=builder /build/migrations /app/migrations

# Copy CA certificates for HTTPS calls
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Use non-root user for security
USER nonroot:nonroot

# Expose application port
EXPOSE 8080

# Health check endpoint
HEALTHCHECK --interval=30s --timeout=10s --start-period=15s --retries=3 \
    CMD ["/app/server", "-health-check"]

# Run the server
ENTRYPOINT ["/app/server"]
