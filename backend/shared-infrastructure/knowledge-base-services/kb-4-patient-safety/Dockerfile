# KB-4 Patient Safety Service - Multi-stage Docker build
# Provides comprehensive medication safety checking with built-in drug database

FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application (main.go is in root)
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kb4-patient-safety .

# Final stage - minimal runtime image
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user for security
RUN addgroup -S kb4group && adduser -S kb4user -G kb4group
USER kb4user

# Copy binary from builder
COPY --from=builder /app/kb4-patient-safety .

# Copy migrations directory
COPY --from=builder --chown=kb4user:kb4group /app/migrations ./migrations

# Copy governed knowledge files (jurisdiction-aware structure)
COPY --from=builder --chown=kb4user:kb4group /app/knowledge ./knowledge

# Environment variables with defaults
ENV PORT=8088
ENV GIN_MODE=release
ENV KNOWLEDGE_PATH=/app/knowledge
ENV JURISDICTION=US

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT}/health || exit 1

# Expose the service port
EXPOSE 8088

# Run the service
CMD ["./kb4-patient-safety"]
