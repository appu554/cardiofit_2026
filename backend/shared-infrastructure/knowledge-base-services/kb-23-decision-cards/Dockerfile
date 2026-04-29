# KB-23 Decision Cards Engine Dockerfile
# Build context is the knowledge-base-services parent directory so the
# sibling kb-26-metabolic-digital-twin module (used via go.mod replace)
# is accessible. See docker-compose.yml: context: ..

# ============================================================
# Stage 1: Build
# ============================================================
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

# Copy the kb-26 dependency first (referenced by go.mod replace directive)
COPY kb-26-metabolic-digital-twin /build/kb-26-metabolic-digital-twin

# Copy kb-23 source
COPY kb-23-decision-cards /build/kb-23

# Update go.mod replace directive for the in-container path
WORKDIR /build/kb-23
RUN sed -i 's|replace kb-26-metabolic-digital-twin => ../kb-26-metabolic-digital-twin|replace kb-26-metabolic-digital-twin => /build/kb-26-metabolic-digital-twin|g' go.mod

RUN go mod tidy && go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# ============================================================
# Stage 2: Runtime
# ============================================================
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata wget

RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /build/kb-23/main .
COPY --from=builder /build/kb-23/migrations ./migrations
COPY --from=builder /build/kb-23/templates ./templates

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8134

# Use GET (not --spider, which sends HEAD); the service only registers GET handlers.
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 -O /dev/null http://localhost:8134/health || exit 1

CMD ["./main"]
