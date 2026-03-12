# Multi-stage Dockerfile for Medication Service V2
# Optimized for healthcare workloads with security and performance focus

# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o medication-service-v2 \
    ./cmd/server

# Runtime stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    && update-ca-certificates

# Create non-root user for security
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set timezone
ENV TZ=UTC

# Create necessary directories
RUN mkdir -p /app/logs /app/config /app/migrations && \
    chown -R appuser:appgroup /app

# Copy built binary
COPY --from=builder --chown=appuser:appgroup /app/medication-service-v2 /app/
COPY --from=builder --chown=appuser:appgroup /app/config/ /app/config/
COPY --from=builder --chown=appuser:appgroup /app/migrations/ /app/migrations/

# Switch to non-root user
USER appuser

# Set working directory
WORKDIR /app

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD curl -f http://localhost:8005/health/live || exit 1

# Expose ports
EXPOSE 8005 50005

# Set environment variables
ENV GIN_MODE=release
ENV GOMAXPROCS=4

# Run the application
CMD ["./medication-service-v2"]

# Metadata labels
LABEL maintainer="Clinical Synthesis Hub Team"
LABEL version="1.0.0"
LABEL description="Medication Service V2 - FHIR-compliant medication management with Recipe & Snapshot architecture"
LABEL healthcare.compliance="HIPAA"
LABEL security.scan="required"