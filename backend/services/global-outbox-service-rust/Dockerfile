# Multi-stage build for Rust Global Outbox Service
FROM rust:1.75-alpine AS builder

# Install dependencies
RUN apk add --no-cache \
    musl-dev \
    protobuf-dev \
    protoc \
    pkgconfig \
    openssl-dev \
    ca-certificates

# Set working directory
WORKDIR /app

# Create dummy project for dependency caching
RUN cargo init --name global-outbox-service-rust

# Copy Cargo files
COPY Cargo.toml Cargo.lock ./

# Copy build script
COPY build.rs ./

# Copy proto files
COPY proto/ proto/

# Build dependencies (cache layer)
RUN cargo build --release
RUN rm src/*.rs target/release/deps/global_outbox_service_rust*

# Copy source code
COPY src/ src/

# Build the application
RUN cargo build --release --bin server

# Production image
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    libgcc

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/target/release/server ./global-outbox-service-rust
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Change ownership
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose ports
EXPOSE 8043 50053

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8043/health || exit 1

# Command to run
CMD ["./global-outbox-service-rust"]