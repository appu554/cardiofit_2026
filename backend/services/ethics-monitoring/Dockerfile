# Multi-stage Dockerfile for ethics-monitoring (Phase 3 Task 1).
#
# Required environment variables at runtime:
#   VAIDSHALA_DSN — PostgreSQL DSN for the Vaidshala shared schema
#
# Optional:
#   PORT          — HTTP listen port (default 8160)

# ── Stage 1: build ──────────────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

WORKDIR /src

# go.mod / go.sum first for layer-cache efficiency.
COPY go.mod go.sum ./

# The replace directive points to ../../shared-infrastructure/knowledge-base-services/shared
# which is outside this build context. The shared module is copied in alongside the
# service source by the build orchestrator (matching the kb-32 convention).
COPY ../../shared-infrastructure/knowledge-base-services/shared /shared

RUN sed -i \
    's|replace github.com/cardiofit/shared => ../../shared-infrastructure/knowledge-base-services/shared|replace github.com/cardiofit/shared => /shared|' \
    go.mod && \
    go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /server ./cmd/server

# ── Stage 2: runtime ────────────────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /server /app/server

EXPOSE 8160

ENTRYPOINT ["/app/server"]
