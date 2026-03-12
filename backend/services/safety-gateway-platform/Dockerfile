# Multi-stage build for Safety Gateway Platform

# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.version=$(git describe --tags --always --dirty) -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o safety-gateway ./cmd/server

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/safety-gateway .

# Copy configuration file
COPY --from=builder /app/config.yaml .

# Create directories for logs and data
RUN mkdir -p /var/log/safety-gateway /var/lib/safety-gateway && \
    chown -R appuser:appgroup /app /var/log/safety-gateway /var/lib/safety-gateway

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8030

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ./safety-gateway -version || exit 1

# Set environment variables
ENV SGP_PORT=8030
ENV SGP_ENVIRONMENT=production

# Run the binary
ENTRYPOINT ["./safety-gateway"]
CMD ["-config=config.yaml"]
