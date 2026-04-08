# KB-26 Metabolic Digital Twin Service — Multi-stage Alpine build
# Context must be knowledge-base-services/ (parent) so shared/ is accessible
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

# Copy shared module first (required by go.mod replace directive)
COPY shared/ /shared/

WORKDIR /app
COPY kb-26-metabolic-digital-twin/go.mod kb-26-metabolic-digital-twin/go.sum ./
ENV GOTOOLCHAIN=auto
RUN go mod download

COPY kb-26-metabolic-digital-twin/ .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app
COPY --from=builder /app/main .
COPY --from=builder /app/migrations ./migrations

RUN chown -R appuser:appgroup /app
USER appuser

EXPOSE 8137

HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8137/health || exit 1

CMD ["./main"]
