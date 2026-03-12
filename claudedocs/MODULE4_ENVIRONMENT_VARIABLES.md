# Module 4 Environment Variables Configuration Guide

## Overview

Module 4 (Pattern Detection) now supports complete configuration externalization via environment variables. All Kafka topics and bootstrap servers can be overridden without code changes.

## Environment Variables

### Kafka Connection

| Variable | Default Value | Description |
|----------|---------------|-------------|
| `KAFKA_BOOTSTRAP_SERVERS` | `localhost:9092` | Kafka cluster bootstrap servers (comma-separated) |

### Input Topics

| Variable | Default Value | Description |
|----------|---------------|-------------|
| `MODULE4_SEMANTIC_INPUT_TOPIC` | `semantic-mesh-updates.v1` | Semantic events from Module 3 Semantic Mesh |
| `MODULE4_ENRICHED_INPUT_TOPIC` | `clinical-patterns.v1` | Enriched events from Module 2 with RiskIndicators |

### Output Topics

| Variable | Default Value | Description |
|----------|---------------|-------------|
| `MODULE4_PATTERN_EVENTS_TOPIC` | `pattern-events.v1` | All pattern events (unified stream) |
| `MODULE4_DETERIORATION_TOPIC` | `alert-management.v1` | Clinical deterioration alerts (MEWS, sepsis, rapid deterioration) |
| `MODULE4_PATHWAY_ADHERENCE_TOPIC` | `pathway-adherence-events.v1` | Care pathway compliance events (sepsis pathway, drug-lab monitoring) |
| `MODULE4_ANOMALY_DETECTION_TOPIC` | `safety-events.v1` | Clinical anomalies and safety events (vital variability) |
| `MODULE4_TREND_ANALYSIS_TOPIC` | `clinical-reasoning-events.v1` | Lab and vital trend analysis (creatinine, glucose, vital trends) |

## Configuration Examples

### Development Environment

```bash
# Use local Kafka
export KAFKA_BOOTSTRAP_SERVERS=localhost:9092

# Use default topic names
# (no topic overrides needed)
```

### Staging Environment

```bash
# Use staging Kafka cluster
export KAFKA_BOOTSTRAP_SERVERS=kafka-staging1:9092,kafka-staging2:9092,kafka-staging3:9092

# Override input topics with staging suffixes
export MODULE4_SEMANTIC_INPUT_TOPIC=semantic-mesh-updates.staging.v1
export MODULE4_ENRICHED_INPUT_TOPIC=clinical-patterns.staging.v1

# Override output topics
export MODULE4_PATTERN_EVENTS_TOPIC=pattern-events.staging.v1
export MODULE4_DETERIORATION_TOPIC=alert-management.staging.v1
export MODULE4_PATHWAY_ADHERENCE_TOPIC=pathway-adherence-events.staging.v1
export MODULE4_ANOMALY_DETECTION_TOPIC=safety-events.staging.v1
export MODULE4_TREND_ANALYSIS_TOPIC=clinical-reasoning-events.staging.v1
```

### Production Environment

```bash
# Use production Kafka cluster
export KAFKA_BOOTSTRAP_SERVERS=kafka-prod1.example.com:9092,kafka-prod2.example.com:9092,kafka-prod3.example.com:9092

# Use production topics
export MODULE4_SEMANTIC_INPUT_TOPIC=semantic-mesh-updates.prod.v1
export MODULE4_ENRICHED_INPUT_TOPIC=clinical-patterns.prod.v1
export MODULE4_PATTERN_EVENTS_TOPIC=pattern-events.prod.v1
export MODULE4_DETERIORATION_TOPIC=alert-management.prod.v1
export MODULE4_PATHWAY_ADHERENCE_TOPIC=pathway-adherence-events.prod.v1
export MODULE4_ANOMALY_DETECTION_TOPIC=safety-events.prod.v1
export MODULE4_TREND_ANALYSIS_TOPIC=clinical-reasoning-events.prod.v1
```

### Docker Compose

```yaml
version: '3.8'
services:
  module4-pattern-detection:
    image: flink-ehr-intelligence:1.0.0
    environment:
      - KAFKA_BOOTSTRAP_SERVERS=kafka:9092
      - MODULE4_SEMANTIC_INPUT_TOPIC=semantic-mesh-updates.v1
      - MODULE4_ENRICHED_INPUT_TOPIC=clinical-patterns.v1
      - MODULE4_PATTERN_EVENTS_TOPIC=pattern-events.v1
      - MODULE4_DETERIORATION_TOPIC=alert-management.v1
      - MODULE4_PATHWAY_ADHERENCE_TOPIC=pathway-adherence-events.v1
      - MODULE4_ANOMALY_DETECTION_TOPIC=safety-events.v1
      - MODULE4_TREND_ANALYSIS_TOPIC=clinical-reasoning-events.v1
```

### Kubernetes ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: module4-config
data:
  KAFKA_BOOTSTRAP_SERVERS: "kafka.default.svc.cluster.local:9092"
  MODULE4_SEMANTIC_INPUT_TOPIC: "semantic-mesh-updates.v1"
  MODULE4_ENRICHED_INPUT_TOPIC: "clinical-patterns.v1"
  MODULE4_PATTERN_EVENTS_TOPIC: "pattern-events.v1"
  MODULE4_DETERIORATION_TOPIC: "alert-management.v1"
  MODULE4_PATHWAY_ADHERENCE_TOPIC: "pathway-adherence-events.v1"
  MODULE4_ANOMALY_DETECTION_TOPIC: "safety-events.v1"
  MODULE4_TREND_ANALYSIS_TOPIC: "clinical-reasoning-events.v1"
```

## Pattern Type Routing

Module 4 routes pattern events to specialized topics based on pattern type:

| Pattern Type | Routed To | Clinical Use |
|--------------|-----------|--------------|
| `MEWS_ALERT` | DETERIORATION_TOPIC | ICU/rapid response team notification |
| `SEPSIS_PATTERN` | DETERIORATION_TOPIC | Sepsis protocol activation |
| `RAPID_DETERIORATION` | DETERIORATION_TOPIC | Emergency response |
| `AKI_PATTERN` | DETERIORATION_TOPIC | Nephrology consult trigger |
| `DRUG_LAB_MONITORING` | PATHWAY_ADHERENCE_TOPIC | Medication safety compliance |
| `SEPSIS_PATHWAY_COMPLIANCE` | PATHWAY_ADHERENCE_TOPIC | Sepsis bundle monitoring |
| `LAB_TREND_ALERT` | TREND_ANALYSIS_TOPIC | Clinical decision support |
| `VITAL_VARIABILITY_ALERT` | ANOMALY_DETECTION_TOPIC | Physiological instability detection |

## Verification

After setting environment variables, verify configuration:

```bash
# Check Kafka connectivity
kafka-topics --bootstrap-server $KAFKA_BOOTSTRAP_SERVERS --list

# Verify input topics exist
kafka-topics --bootstrap-server $KAFKA_BOOTSTRAP_SERVERS --describe --topic $MODULE4_SEMANTIC_INPUT_TOPIC
kafka-topics --bootstrap-server $KAFKA_BOOTSTRAP_SERVERS --describe --topic $MODULE4_ENRICHED_INPUT_TOPIC

# Create output topics if needed
kafka-topics --bootstrap-server $KAFKA_BOOTSTRAP_SERVERS --create --topic $MODULE4_PATTERN_EVENTS_TOPIC --partitions 6 --replication-factor 3
kafka-topics --bootstrap-server $KAFKA_BOOTSTRAP_SERVERS --create --topic $MODULE4_DETERIORATION_TOPIC --partitions 6 --replication-factor 3
kafka-topics --bootstrap-server $KAFKA_BOOTSTRAP_SERVERS --create --topic $MODULE4_PATHWAY_ADHERENCE_TOPIC --partitions 3 --replication-factor 3
kafka-topics --bootstrap-server $KAFKA_BOOTSTRAP_SERVERS --create --topic $MODULE4_ANOMALY_DETECTION_TOPIC --partitions 3 --replication-factor 3
kafka-topics --bootstrap-server $KAFKA_BOOTSTRAP_SERVERS --create --topic $MODULE4_TREND_ANALYSIS_TOPIC --partitions 3 --replication-factor 3
```

## Backward Compatibility

All environment variables have sensible defaults matching the previous hardcoded values. Module 4 will work without any environment variable configuration, using:

- `localhost:9092` for Kafka
- Default topic names as specified in the table above

This ensures zero-disruption deployment for existing installations.
