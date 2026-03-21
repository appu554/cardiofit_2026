# Ingestion Service — Multi-stage build
FROM golang:1.25-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates

# Copy parent module for replace directive
COPY go.mod go.sum ./
COPY pkg/ pkg/

# Copy service source
COPY services/ingestion-service/ services/ingestion-service/

WORKDIR /app/services/ingestion-service
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /ingestion-service ./cmd/ingestion

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -g '' appuser
COPY --from=builder /ingestion-service /app/ingestion-service
USER appuser
EXPOSE 8140

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8140/healthz || exit 1

CMD ["/app/ingestion-service"]
