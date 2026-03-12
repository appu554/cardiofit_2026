# KB-0 Governance Platform Dockerfile
# Multi-stage build for production-grade Go service

# =============================================================================
# STAGE 1: Build
# =============================================================================
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files first for caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=$(date +%Y%m%d)" \
    -o /kb0-server \
    ./cmd/server

# =============================================================================
# STAGE 2: Runtime
# =============================================================================
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates curl tzdata

# Create non-root user for security
RUN addgroup -g 1001 -S kb0 && \
    adduser -u 1001 -S kb0 -G kb0

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /kb0-server /app/kb0-server

# Set ownership
RUN chown -R kb0:kb0 /app

# Switch to non-root user
USER kb0

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Run the server
ENTRYPOINT ["/app/kb0-server"]
