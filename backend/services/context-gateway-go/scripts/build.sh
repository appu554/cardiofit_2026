#!/bin/bash

# Build script for Context Gateway Go Service
# Handles dependencies, code generation, and compilation

set -e

PROJECT_NAME="context-gateway-go"
BUILD_DIR="build"
BINARY_NAME="context-gateway"

echo "🏗️ Building Context Gateway Go Service..."

# Clean build directory
echo "🧹 Cleaning build directory..."
rm -rf $BUILD_DIR
mkdir -p $BUILD_DIR

# Download dependencies
echo "📦 Downloading Go dependencies..."
go mod download
go mod tidy

# Generate protocol buffer code
echo "🔧 Generating Protocol Buffer code..."
if [ -f "scripts/generate_proto.sh" ]; then
    chmod +x scripts/generate_proto.sh
    ./scripts/generate_proto.sh
else
    echo "⚠️ Warning: generate_proto.sh not found, assuming proto code exists"
fi

# Run tests
echo "🧪 Running tests..."
go test -v ./...

# Build for current platform
echo "🔨 Building binary for current platform..."
CGO_ENABLED=0 go build -ldflags="-w -s" -o $BUILD_DIR/$BINARY_NAME ./cmd/main.go

# Build for multiple platforms
echo "🌍 Building cross-platform binaries..."

# Linux AMD64
echo "  Building for Linux AMD64..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
    -ldflags="-w -s" \
    -o $BUILD_DIR/${BINARY_NAME}-linux-amd64 \
    ./cmd/main.go

# Linux ARM64
echo "  Building for Linux ARM64..."
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build \
    -ldflags="-w -s" \
    -o $BUILD_DIR/${BINARY_NAME}-linux-arm64 \
    ./cmd/main.go

# Windows AMD64
echo "  Building for Windows AMD64..."
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build \
    -ldflags="-w -s" \
    -o $BUILD_DIR/${BINARY_NAME}-windows-amd64.exe \
    ./cmd/main.go

# macOS AMD64
echo "  Building for macOS AMD64..."
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build \
    -ldflags="-w -s" \
    -o $BUILD_DIR/${BINARY_NAME}-darwin-amd64 \
    ./cmd/main.go

# macOS ARM64 (Apple Silicon)
echo "  Building for macOS ARM64..."
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build \
    -ldflags="-w -s" \
    -o $BUILD_DIR/${BINARY_NAME}-darwin-arm64 \
    ./cmd/main.go

echo "✅ Build completed successfully!"
echo "📁 Binaries created in $BUILD_DIR/:"
ls -la $BUILD_DIR/

echo ""
echo "🚀 To run the service locally:"
echo "  ./$BUILD_DIR/$BINARY_NAME"
echo ""
echo "🔧 Available command-line options:"
echo "  -grpc-port :8017    # gRPC server port"
echo "  -http-port :8117    # HTTP server port (metrics/health)"
echo "  -redis-addr localhost:6379   # Redis address"
echo "  -mongo-uri mongodb://localhost:27017  # MongoDB URI"
echo "  -db-name clinical_context_go  # Database name"
echo "  -env development    # Environment (development/production)"