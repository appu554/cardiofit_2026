# Multi-stage build for KB Cross-Dependency Manager
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o kb-cross-dependency-manager cmd/server/main.go

# Final stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/kb-cross-dependency-manager .

# Create necessary directories
RUN mkdir -p /app/logs && \
    chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD curl -f http://localhost:${PORT:-8095}/health || exit 1

# Expose port
EXPOSE 8095

# Run the application
ENTRYPOINT ["./kb-cross-dependency-manager"]

# Metadata
LABEL maintainer="Clinical Synthesis Hub <clinical-team@example.com>"
LABEL version="1.0.0"
LABEL description="KB Cross-Dependency Manager - Manages dependencies, conflicts, and impact analysis across Knowledge Base services"
LABEL service="kb-cross-dependency-manager"
LABEL framework.compliance="2.0"
LABEL clinical.safety.critical="true"