# KB-13 Quality Measures Engine - Dockerfile
# Multi-stage build for minimal production image

# Stage 1: Build
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o kb13-server ./cmd/server

# Stage 2: Production image
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates wget

# Copy binary from builder
COPY --from=builder /build/kb13-server /app/kb13-server

# Copy measure definitions and benchmarks
COPY measures/ /app/measures/
COPY benchmarks/ /app/benchmarks/

# Create non-root user
RUN adduser -D -g '' kb13user
USER kb13user

# Expose port
EXPOSE 8113

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost:8113/health || exit 1

# Run the service
ENTRYPOINT ["/app/kb13-server"]
