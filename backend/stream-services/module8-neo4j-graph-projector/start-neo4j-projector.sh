#!/bin/bash

# Navigate to the script directory
cd "$(dirname "$0")"

# Load environment variables from .env file
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
    echo "✓ Loaded environment variables from .env"
else
    echo "✗ Error: .env file not found"
    exit 1
fi

# Verify required environment variables
# Only require KAFKA_API_KEY if using SASL_SSL
if [ "$KAFKA_SECURITY_PROTOCOL" = "SASL_SSL" ] && [ -z "$KAFKA_API_KEY" ]; then
    echo "✗ Error: KAFKA_API_KEY required for SASL_SSL"
    exit 1
fi

if [ -z "$NEO4J_PASSWORD" ]; then
    echo "✗ Error: NEO4J_PASSWORD not set"
    exit 1
fi

echo "✓ Environment variables validated"
echo "Starting Neo4j Graph Projector Service on port ${SERVICE_PORT:-8057}..."

# Start service with uvicorn
python3 -m uvicorn app.main:app \
    --host "${SERVICE_HOST:-0.0.0.0}" \
    --port "${SERVICE_PORT:-8057}" \
    --log-level info
