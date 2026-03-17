# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy shared module first (for replace directive)
COPY shared/ /shared/

# Copy go mod files
COPY kb-1-drug-rules/go.mod kb-1-drug-rules/go.sum ./
RUN go mod download

# Copy source code
COPY kb-1-drug-rules/ .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kb-1-drug-rules ./cmd/server

# Final stage
FROM alpine:3.19

WORKDIR /app

# Add ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/kb-1-drug-rules .

# Expose port
EXPOSE 8081

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8081/health || exit 1

# Run the service
CMD ["./kb-1-drug-rules"]
