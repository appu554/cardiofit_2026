# KB-9 Care Gaps Service Dockerfile
# Multi-stage build for minimal production image
# Build context: cardiofit root (for vaidshala dependency access)

# ============================================================
# Stage 1: Build
# ============================================================
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy vaidshala dependency first (required for go.mod replace directive)
COPY vaidshala/clinical-runtime-platform /build/vaidshala/clinical-runtime-platform

# Copy kb-9 source
COPY backend/shared-infrastructure/knowledge-base-services/kb-9-care-gaps /build/kb-9

# Update go.mod replace directive for Docker context
WORKDIR /build/kb-9
RUN sed -i 's|replace vaidshala/clinical-runtime-platform => ../../../../vaidshala/clinical-runtime-platform|replace vaidshala/clinical-runtime-platform => /build/vaidshala/clinical-runtime-platform|g' go.mod

# Download dependencies and regenerate go.sum
RUN go mod tidy && go mod download

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o kb-9-care-gaps \
    ./cmd/server

# ============================================================
# Stage 2: Runtime
# ============================================================
FROM alpine:3.19

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata wget

# Create non-root user for security
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/kb-9/kb-9-care-gaps .

# Set ownership
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8089

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8089/health || exit 1

# Run the service
CMD ["./kb-9-care-gaps"]
