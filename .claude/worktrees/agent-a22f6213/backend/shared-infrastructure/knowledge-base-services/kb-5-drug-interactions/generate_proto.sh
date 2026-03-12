#!/bin/bash
# Generate protobuf code for KB-5 Drug Interactions service

set -e

echo "Generating protobuf code for KB-5..."

# Create api/pb directory if it doesn't exist
mkdir -p api/pb

# Generate Go protobuf code
protoc --go_out=api/pb --go_opt=paths=source_relative \
       --go-grpc_out=api/pb --go-grpc_opt=paths=source_relative \
       api/kb5.proto

echo "✅ Protobuf code generation complete!"
echo "Generated files:"
echo "  - api/pb/kb5.pb.go"
echo "  - api/pb/kb5_grpc.pb.go"

# Check if files were created successfully
if [ -f "api/pb/kb5.pb.go" ] && [ -f "api/pb/kb5_grpc.pb.go" ]; then
    echo "✅ All protobuf files generated successfully!"
else
    echo "❌ Error: Some protobuf files failed to generate"
    exit 1
fi

# Run go mod tidy to ensure dependencies are properly managed
echo "Running go mod tidy..."
go mod tidy

echo "🚀 KB-5 protobuf generation complete and ready for integration!"