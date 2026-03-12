#!/bin/bash

set -e

echo "Setting up notification service..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed"
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | awk '{print $3}')
echo "Go version: $GO_VERSION"

# Install dependencies
echo "Installing dependencies..."
go mod download

# Create required directories
echo "Creating directories..."
mkdir -p bin
mkdir -p logs
mkdir -p credentials

# Check if PostgreSQL is running
if command -v psql &> /dev/null; then
    echo "PostgreSQL client found"
else
    echo "Warning: PostgreSQL client not found"
fi

# Check if Redis is running
if command -v redis-cli &> /dev/null; then
    echo "Redis client found"
else
    echo "Warning: Redis client not found"
fi

echo "Setup complete!"
echo "Run 'make run' to start the service"
