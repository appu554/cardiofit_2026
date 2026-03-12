#!/bin/bash

# Generate Go code from Protocol Buffers
# This script generates gRPC service definitions and message types

set -e

echo "🔧 Generating Protocol Buffer code for Context Gateway..."

# Check if protoc is installed
if ! command -v protoc &> /dev/null; then
    echo "❌ protoc is not installed. Please install Protocol Buffers compiler."
    echo "   On Ubuntu/Debian: sudo apt-get install protobuf-compiler"
    echo "   On macOS: brew install protobuf"
    echo "   On Windows: Download from https://github.com/protocolbuffers/protobuf/releases"
    exit 1
fi

# Check if protoc-gen-go is installed
if ! command -v protoc-gen-go &> /dev/null; then
    echo "📦 Installing protoc-gen-go..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

# Check if protoc-gen-go-grpc is installed
if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo "📦 Installing protoc-gen-go-grpc..."
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

# Create output directory
mkdir -p proto

# Generate Go code from proto files
echo "🏗️ Generating Go code from proto files..."

protoc \
    --go_out=. \
    --go_opt=paths=source_relative \
    --go-grpc_out=. \
    --go-grpc_opt=paths=source_relative \
    proto/context_gateway.proto

echo "✅ Protocol Buffer code generation completed successfully!"
echo "📁 Generated files:"
echo "   - proto/context_gateway.pb.go (message types)"
echo "   - proto/context_gateway_grpc.pb.go (gRPC service definitions)"