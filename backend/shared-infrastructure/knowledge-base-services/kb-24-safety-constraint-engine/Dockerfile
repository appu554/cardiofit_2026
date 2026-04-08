# Multi-stage Dockerfile for KB-24 Safety Constraint Engine.
# Produces a minimal runtime image with only the compiled binary.

# ── Build stage ──────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Cache dependency downloads
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /bin/kb24-sce ./cmd/server

# ── Runtime stage ────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

# Non-root user for security
RUN addgroup -S sce && adduser -S sce -G sce

WORKDIR /app

COPY --from=builder /bin/kb24-sce /app/kb24-sce
COPY --from=builder /app/configs /app/configs

# Default node definitions mount point
RUN mkdir -p /app/nodes && chown -R sce:sce /app

USER sce

EXPOSE 8201

HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8201/health || exit 1

ENTRYPOINT ["/app/kb24-sce"]
