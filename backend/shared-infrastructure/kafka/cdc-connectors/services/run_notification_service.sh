#!/bin/bash
# Run the Terminology Notification Service
# Consumes CDC events and notifies downstream KB services

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Default environment variables
export KAFKA_BOOTSTRAP_SERVERS="${KAFKA_BOOTSTRAP_SERVERS:-localhost:9092}"
export REDIS_URL="${REDIS_URL:-redis://localhost:6379}"

# KB Service webhook URLs (override as needed)
export KB1_WEBHOOK_URL="${KB1_WEBHOOK_URL:-http://localhost:8081/webhooks/terminology-update}"
export KB2_WEBHOOK_URL="${KB2_WEBHOOK_URL:-http://localhost:8086/webhooks/terminology-update}"
export KB3_WEBHOOK_URL="${KB3_WEBHOOK_URL:-http://localhost:8087/webhooks/terminology-update}"
export KB4_WEBHOOK_URL="${KB4_WEBHOOK_URL:-http://localhost:8088/webhooks/terminology-update}"
export KB5_WEBHOOK_URL="${KB5_WEBHOOK_URL:-http://localhost:8089/webhooks/terminology-update}"
export KB6_WEBHOOK_URL="${KB6_WEBHOOK_URL:-http://localhost:8091/webhooks/terminology-update}"
export KB7_WEBHOOK_URL="${KB7_WEBHOOK_URL:-http://localhost:8092/webhooks/terminology-update}"

echo "==================================="
echo "Terminology Notification Service"
echo "==================================="
echo "Kafka: $KAFKA_BOOTSTRAP_SERVERS"
echo "Redis: $REDIS_URL"
echo "==================================="

# Install dependencies if needed
if [ ! -d "$SCRIPT_DIR/venv" ]; then
    echo "Creating virtual environment..."
    python3 -m venv "$SCRIPT_DIR/venv"
    source "$SCRIPT_DIR/venv/bin/activate"
    pip install -r "$SCRIPT_DIR/requirements.txt"
else
    source "$SCRIPT_DIR/venv/bin/activate"
fi

# Run the service
python3 "$SCRIPT_DIR/terminology_notification_service.py"
