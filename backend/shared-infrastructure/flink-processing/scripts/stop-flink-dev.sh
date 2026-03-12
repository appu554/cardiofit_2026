#!/bin/bash

# Stop Flink Development Environment

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "=========================================="
echo "Stopping Flink EHR Intelligence Engine"
echo "=========================================="

cd "$PROJECT_ROOT"

echo "📛 Stopping Flink cluster..."
docker-compose down

echo "✅ Flink cluster stopped"

echo ""
echo "Note: Kafka infrastructure is still running."
echo "To stop Kafka, run: cd ../kafka && ./stop-kafka.sh"