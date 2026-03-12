# Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies including SASL for Kafka
RUN apk add --no-cache gcc musl-dev librdkafka-dev pkgconfig cyrus-sasl-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with dynamic linking for librdkafka
RUN CGO_ENABLED=1 GOOS=linux go build -tags dynamic -o kb-17-population-registry ./cmd/server

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies including librdkafka and sasl
RUN apk add --no-cache ca-certificates librdkafka cyrus-sasl

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/kb-17-population-registry .

# Create non-root user
RUN adduser -D -g '' appuser
USER appuser

# Expose port
EXPOSE 8017

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8017/health || exit 1

# Run the application
CMD ["./kb-17-population-registry"]
