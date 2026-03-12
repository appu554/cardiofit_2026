#!/bin/bash
# PostgreSQL Projector Startup Script

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export PYTHONPATH="${SCRIPT_DIR}/module8-shared:${PYTHONPATH}"

# Kafka Configuration (local Docker Kafka)
export KAFKA_BOOTSTRAP_SERVERS="localhost:9092"
export KAFKA_SECURITY_PROTOCOL="PLAINTEXT"

# PostgreSQL Configuration (using existing container)
export POSTGRES_HOST="localhost"
export POSTGRES_PORT="5433"
export POSTGRES_DB="cardiofit"
export POSTGRES_USER="cardiofit"
export POSTGRES_PASSWORD="cardiofit_analytics_pass"
export POSTGRES_SCHEMA="module8_projections"

# Batch Configuration
export BATCH_SIZE="100"
export BATCH_TIMEOUT_SECONDS="5.0"

# Service Configuration
export SERVICE_PORT="8050"
export LOG_LEVEL="INFO"

echo "Starting PostgreSQL Projector..."
echo "Kafka: ${KAFKA_BOOTSTRAP_SERVERS}"
echo "PostgreSQL: ${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}"
echo "PYTHONPATH: ${PYTHONPATH}"

cd "${SCRIPT_DIR}/module8-postgresql-projector"
python3 -m app.main
