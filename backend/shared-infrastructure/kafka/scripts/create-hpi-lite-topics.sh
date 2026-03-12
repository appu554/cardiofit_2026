#!/usr/bin/env bash
#
# Creates the three core HPI telemetry topics on the lightweight Kafka stack.

set -euo pipefail

BOOTSTRAP="${BOOTSTRAP_SERVERS:-localhost:9092}"
REPLICATION_FACTOR="${REPLICATION_FACTOR:-1}"
PARTITIONS="${PARTITIONS:-3}"

TOPICS=(
  "hpi.session.events"
  "hpi.escalation.events"
  "hpi.calibration.data"
)

for topic in "${TOPICS[@]}"; do
  echo "Creating topic ${topic}..."
  docker exec cardiofit-kafka-lite kafka-topics \
    --bootstrap-server "${BOOTSTRAP}" \
    --create \
    --if-not-exists \
    --replication-factor "${REPLICATION_FACTOR}" \
    --partitions "${PARTITIONS}" \
    --topic "${topic}"
done

echo "✅ All HPI topics ensured on ${BOOTSTRAP}"
