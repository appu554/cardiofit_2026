# KB-19 Protocol Orchestrator Dockerfile
# Multi-stage build for optimized production image

# Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')" \
    -o /kb-19-protocol-orchestrator \
    ./cmd/server

# Runtime stage
FROM alpine:3.19

# Add labels
LABEL org.opencontainers.image.title="KB-19 Protocol Orchestrator"
LABEL org.opencontainers.image.description="Clinical Protocol Orchestrator - Decision Synthesis Engine"
LABEL org.opencontainers.image.version="1.0.0"

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -g '' appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /kb-19-protocol-orchestrator .

# Copy protocol and conflict definitions
COPY protocols/ ./protocols/
COPY conflicts/ ./conflicts/
COPY migrations/ ./migrations/

# Change ownership
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8099

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8099/health || exit 1

# Set environment variables
ENV PORT=8099
ENV ENVIRONMENT=production
ENV LOG_LEVEL=info
ENV LOG_FORMAT=json

# Run the binary
ENTRYPOINT ["./kb-19-protocol-orchestrator"]
