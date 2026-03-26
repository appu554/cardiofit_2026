#!/bin/bash
# create-v4-topics.sh — Creates 9 V4 Kafka topics
# Part of Phase C0: V4 North Star infrastructure
set -euo pipefail

BOOTSTRAP="${KAFKA_BOOTSTRAP_SERVERS:-localhost:9092}"

# Format: topic:partitions:retention_ms:compression:min_insync_replicas
TOPICS=(
  "flink.bp-variability-metrics:8:2592000000:snappy:2"
  "flink.meal-response:8:2592000000:snappy:2"
  "flink.meal-patterns:4:7776000000:snappy:2"
  "flink.engagement-signals:4:2592000000:snappy:2"
  "clinical.intervention-events:4:7776000000:gzip:3"
  "clinical.intervention-window-signals:4:7776000000:gzip:3"
  "clinical.decision-cards:4:2592000000:snappy:2"
  "alerts.comorbidity-interactions:4:7776000000:gzip:3"
  "alerts.engagement-drop:2:7776000000:gzip:2"
)

for entry in "${TOPICS[@]}"; do
  IFS=':' read -r topic partitions retention compression min_isr <<< "$entry"
  echo "Creating topic: $topic (partitions=$partitions, retention=${retention}ms, min.isr=$min_isr)"
  kafka-topics --bootstrap-server "$BOOTSTRAP" \
    --create --if-not-exists \
    --topic "$topic" \
    --partitions "$partitions" \
    --replication-factor 3 \
    --config retention.ms="$retention" \
    --config cleanup.policy=delete \
    --config compression.type="$compression" \
    --config min.insync.replicas="$min_isr"
done

echo "V4 topics created successfully."
