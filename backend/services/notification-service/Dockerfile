# Multi-stage build for CardioFit Notification Service (Go)
# Stage 1: Build the Go binary
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

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
    -ldflags="-w -s" \
    -o notification-service \
    ./cmd/notification-service/main.go

# Stage 2: Create minimal runtime image
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user
RUN addgroup -g 1000 notification && \
    adduser -D -u 1000 -G notification notification

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/notification-service .

# Copy migrations if they exist
COPY --from=builder /app/migrations ./migrations 2>/dev/null || true

# Copy config files if they exist
COPY --from=builder /app/configs ./configs 2>/dev/null || true

# Change ownership
RUN chown -R notification:notification /app

# Switch to non-root user
USER notification

# Expose ports
EXPOSE 8080 50051

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["/app/notification-service"]
