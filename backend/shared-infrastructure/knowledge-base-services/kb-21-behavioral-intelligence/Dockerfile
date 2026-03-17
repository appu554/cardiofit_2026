# KB-21 Behavioral Intelligence Service — Multi-stage Alpine build
# Follows the standard CardioFit KB service Dockerfile pattern

# Build stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

# Non-root user for security (HIPAA compliance)
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app
COPY --from=builder /app/main .
COPY --from=builder /app/migrations ./migrations

RUN chown -R appuser:appgroup /app
USER appuser

EXPOSE 8133

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8133/health || exit 1

CMD ["./main"]
