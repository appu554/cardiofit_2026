# Multi-stage Dockerfile for kb-32-recommendation-craft
# Matches the Phase 1b-completion convention used by pharmacist-self-visibility
# and kb-30-authorisation-evaluator.
#
# Required environment variables at runtime:
#   VAIDSHALA_DSN   — PostgreSQL DSN for the Vaidshala shared schema
#   JWT_SECRET      — HMAC secret for JWT validation (Task 13)
#
# Optional:
#   PORT            — HTTP listen port (default: 8150)
#   KB32_DEV_MODE   — "true" to skip JWT_SECRET requirement (never in prod)

# ── Stage 1: build ──────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /src

# Copy go.mod/go.sum first for layer-cache efficiency.
COPY go.mod go.sum ./

# The replace directive points to ../shared which is outside this context.
# We copy the shared module alongside the service source before running tidy.
COPY ../shared /shared

# Rewrite the replace directive to the in-context path.
# go mod tidy resolves all transitive deps with the local shared module.
RUN sed -i 's|replace github.com/cardiofit/shared => ../shared|replace github.com/cardiofit/shared => /shared|' go.mod && \
    go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /server ./cmd/server

# ── Stage 2: runtime ────────────────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /server /app/server

EXPOSE 8150

ENTRYPOINT ["/app/server"]
