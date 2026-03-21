# Intake Onboarding Service — Multi-stage build
FROM golang:1.25-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates

# Copy parent module for replace directive
COPY go.mod go.sum ./
COPY pkg/ pkg/

# Copy service source
COPY services/intake-onboarding-service/ services/intake-onboarding-service/

WORKDIR /app/services/intake-onboarding-service
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /intake-onboarding-service ./cmd/intake

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -g '' appuser
COPY --from=builder /intake-onboarding-service /app/intake-onboarding-service
USER appuser
EXPOSE 8141

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8141/healthz || exit 1

CMD ["/app/intake-onboarding-service"]
