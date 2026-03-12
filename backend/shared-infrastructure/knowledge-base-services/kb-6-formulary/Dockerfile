# KB-6 Formulary Management Service - Production Dockerfile
# Multi-stage build for optimized image size and security

# ============================================================================
# Build Stage
# ============================================================================
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with optimizations
# -ldflags "-w -s" strips debug info for smaller binary
# CGO_ENABLED=0 ensures static linking
ARG VERSION=1.0.0
ARG BUILD_TIME
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}" \
    -o kb6-formulary ./main.go

# Download grpc_health_probe for gRPC health checks
RUN GRPC_HEALTH_PROBE_VERSION=v0.4.22 && \
    wget -qO/bin/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64 && \
    chmod +x /bin/grpc_health_probe

# ============================================================================
# Production Stage
# ============================================================================
FROM alpine:3.19

# Add labels for container metadata
LABEL maintainer="CardioFit Platform Team"
LABEL service="kb-6-formulary"
LABEL description="Knowledge Base 6 - Formulary Management Service"

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/kb6-formulary .

# Copy grpc_health_probe for health checks
COPY --from=builder /bin/grpc_health_probe /bin/grpc_health_probe

# Copy migrations directory (may be needed at runtime)
COPY migrations ./migrations

# Create non-root user for security
RUN addgroup -g 1000 appgroup && \
    adduser -D -u 1000 -G appgroup appuser && \
    chown -R appuser:appgroup /app

USER appuser

# Expose ports
# 8086 - gRPC API
# 8087 - HTTP REST API
EXPOSE 8086 8087

# Health check using grpc_health_probe
HEALTHCHECK --interval=30s --timeout=10s --start-period=45s --retries=3 \
    CMD ["/bin/grpc_health_probe", "-addr=:8086"]

# Default environment variables
ENV PORT=8086 \
    ENVIRONMENT=production \
    DEBUG=false \
    METRICS_ENABLED=true \
    METRICS_PATH=/metrics

# Run the service
ENTRYPOINT ["./kb6-formulary"]
