#!/usr/bin/env bash
#
# Lightweight single-broker Kafka stack for local HPI development.

set -euo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
COMPOSE_FILE="${SCRIPT_DIR}/docker-compose.hpi-lite.yml"

if ! command -v docker &>/dev/null; then
  echo "Docker CLI not found. Install Docker Desktop before running this script."
  exit 1
fi

if ! docker info &>/dev/null; then
  echo "Docker daemon is not running. Start Docker Desktop and retry."
  exit 1
fi

echo "Starting lightweight Kafka stack (Zookeeper + single broker)..."
docker compose -f "${COMPOSE_FILE}" up -d

echo "Kafka broker is ready at PLAINTEXT://localhost:9092"
echo "Use scripts/create-hpi-lite-topics.sh to provision required topics."
