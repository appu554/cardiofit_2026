#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════════
# Create V4 Kafka topics (Modules 7-13) on the cardiofit-kafka-lite container.
#
# Idempotent: --if-not-exists ensures safe re-runs.
#
# Usage:
#   bash scripts/create-v4-topics.sh                    # default container
#   KAFKA_CONTAINER=my-kafka bash scripts/create-v4-topics.sh
# ═══════════════════════════════════════════════════════════════════════
set -euo pipefail

KAFKA_CONTAINER="${KAFKA_CONTAINER:-cardiofit-kafka-lite}"
BOOTSTRAP="${BOOTSTRAP:-localhost:29092}"
PARTITIONS=4
REPLICATION=1

TOPICS=(
  # --- Input topics (consumed by Modules 7→13) ---
  "ingestion.vitals"                     # Module 7 input (BP readings)
  "enriched-patient-events-v1"           # Module 8, 9, 10, 11 input
  "flink.meal-response"                  # Module 10b input (from Module 10)
  "flink.activity-response"              # Module 11b input (from Module 11)

  # --- Output / intermediate topics ---
  # Module 7: BP Variability Engine
  "flink.bp-variability-metrics"
  # Module 8: Comorbidity Interaction
  "alerts.comorbidity-interactions"
  # Module 9: Engagement Monitor
  "flink.engagement-signals"
  "alerts.engagement-drop"
  # Module 10b: Meal Pattern Aggregator
  "flink.meal-patterns"
  # Module 11b: Fitness Pattern Aggregator
  "flink.fitness-patterns"
  # Module 12: Intervention Window Monitor
  "clinical.intervention-window-signals"
  # Module 12b: Intervention Delta Computer
  "flink.intervention-deltas"
  # Module 13: Clinical State Synchroniser (output)
  "clinical.state-change-events"
)

echo "Creating V4 topics on ${KAFKA_CONTAINER} (bootstrap: ${BOOTSTRAP})"
echo "───────────────────────────────────────────────────"

for topic in "${TOPICS[@]}"; do
  docker exec "${KAFKA_CONTAINER}" kafka-topics \
    --create \
    --bootstrap-server "${BOOTSTRAP}" \
    --topic "${topic}" \
    --partitions "${PARTITIONS}" \
    --replication-factor "${REPLICATION}" \
    --if-not-exists 2>&1 | grep -v "^WARNING:" || true
  echo "  ✓ ${topic}"
done

echo ""
echo "V4 topics ready. Total topics:"
docker exec "${KAFKA_CONTAINER}" kafka-topics --list --bootstrap-server "${BOOTSTRAP}" | wc -l
