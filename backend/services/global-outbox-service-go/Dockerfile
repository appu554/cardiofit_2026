# Multi-stage build for Go Global Outbox Service
# Using Debian (glibc) because confluent-kafka-go v2.3.0 vendors
# a glibc-linked librdkafka — Alpine/musl causes linker errors on ARM64.
FROM golang:1.24-bookworm AS builder

# Install dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    git ca-certificates tzdata protobuf-compiler && \
    rm -rf /var/lib/apt/lists/*

# Install protobuf Go plugins (pinned to versions compatible with go.mod deps)
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.32.0
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Generate protobuf code
RUN mkdir -p pkg/proto
RUN protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    pkg/proto/outbox.proto

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-s -w" \
    -trimpath \
    -o global-outbox-service \
    cmd/server/main.go

# Production image — Debian slim for glibc compatibility
FROM debian:bookworm-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates tzdata wget && \
    rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN groupadd -g 1001 appgroup && \
    useradd -u 1001 -g appgroup -s /bin/false appuser

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/global-outbox-service .

# Change ownership
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose ports
EXPOSE 8042 50052

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8042/health || exit 1

# Command to run
CMD ["./global-outbox-service"]
