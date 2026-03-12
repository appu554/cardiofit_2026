#!/bin/bash

# KB-2 Clinical Context Service - Startup Script
# This script helps start the KB-2 service with proper configuration

set -e

SERVICE_DIR="/Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-2-clinical-context"
SERVICE_NAME="kb-2-clinical-context"
BINARY="$SERVICE_DIR/bin/$SERVICE_NAME"
PORT=8082

echo "========================================="
echo "KB-2 Clinical Context Service Startup"
echo "========================================="
echo ""

# Check if binary exists
if [ ! -f "$BINARY" ]; then
    echo "❌ Binary not found at: $BINARY"
    echo "Building service..."
    cd "$SERVICE_DIR"
    go build -o "$BINARY"
    echo "✅ Build complete"
fi

# Check MongoDB
echo "Checking MongoDB connection..."
if nc -z localhost 27017 2>/dev/null; then
    echo "✅ MongoDB is running on localhost:27017"
else
    echo "⚠️  MongoDB not detected on localhost:27017"
    echo "   Service will attempt to connect but may fail"
    echo "   To start MongoDB:"
    echo "   - Use Docker: docker run -d -p 27017:27017 mongo:latest"
    echo "   - Or start local MongoDB instance"
fi

# Check Redis
echo "Checking Redis connection..."
if nc -z localhost 6380 2>/dev/null; then
    echo "✅ Redis is running on localhost:6380"
elif nc -z localhost 6379 2>/dev/null; then
    echo "⚠️  Redis found on port 6379, but service expects 6380"
    echo "   Set REDIS_URL=localhost:6379 or start Redis on 6380"
else
    echo "⚠️  Redis not detected on localhost:6380"
    echo "   Service will attempt to connect but may fail"
    echo "   To start Redis:"
    echo "   - Use Docker: docker run -d -p 6380:6379 redis:latest"
    echo "   - Or start local Redis instance"
fi

echo ""
echo "========================================="
echo "Starting KB-2 Clinical Context Service"
echo "========================================="
echo "Port: $PORT"
echo "MongoDB: ${MONGODB_URI:-mongodb://localhost:27017}"
echo "Redis: ${REDIS_URL:-localhost:6380}"
echo ""

# Export default environment variables if not set
export PORT="${PORT:-8082}"
export ENVIRONMENT="${ENVIRONMENT:-development}"
export DEBUG="${DEBUG:-true}"
export MONGODB_URI="${MONGODB_URI:-mongodb://localhost:27017}"
export MONGODB_DATABASE="${MONGODB_DATABASE:-clinical_context}"
export REDIS_URL="${REDIS_URL:-localhost:6380}"
export METRICS_ENABLED="${METRICS_ENABLED:-true}"

# Start the service
cd "$SERVICE_DIR"
echo "Executing: $BINARY"
echo ""
exec "$BINARY"
