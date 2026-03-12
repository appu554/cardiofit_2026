# Kafka Topics Reference Guide - CardioFit Platform

## Table of Contents
- [Overview](#overview)
- [Topic Categories](#topic-categories)
- [Complete Topic List](#complete-topic-list)
- [Topic Specifications](#topic-specifications)
- [Integration Matrix](#integration-matrix)
- [Usage Examples](#usage-examples)
- [Quick Reference](#quick-reference)

## Overview

This document serves as the comprehensive reference for all Kafka topics in the CardioFit Clinical Synthesis Hub platform. Each topic is documented with its purpose, configuration, producers, consumers, and integration patterns.

**IMPORTANT UPDATE**: The platform now uses a **Hybrid Kafka Topic Architecture** for the EHR Intelligence Engine, providing both a central system of record and purpose-built action topics with transactional guarantees.

### Key Metrics
- **Total Topics**: 75 (68 original + 7 hybrid architecture topics)
- **Categories**: 12 (11 original + 1 hybrid architecture)
- **Max Throughput**: 10,000+ events/second
- **Retention Range**: 1 day to 7 years
- **Partition Range**: 2 to 32

## Topic Categories

| Category | Count | Purpose | Retention Policy |
|----------|-------|---------|-----------------|
| **🆕 Hybrid Architecture** | **7** | **Transactional multi-sink routing** | **7 days - 7 years** |
| Clinical Events | 9 | Core patient and clinical data | 3-7 days |
| Device Data | 4 | Medical device and IoT integration | 1-3 days |
| Runtime Layer | 5 | Stream processing and enrichment | 7-30 days |
| Knowledge Base CDC | 8 | Real-time knowledge synchronization | 7 days (compacted) |
| Evidence Management | 6 | Clinical evidence and audit trail | 7-365 days |
| Workflow & Orchestration | 6 | Process automation | 3-90 days |
| SLA & Monitoring | 6 | Performance tracking | 7-90 days |
| Cache & Optimization | 4 | Performance layer | 1-7 days |
| Dead Letter Queues | 9 | Error handling | 7-365 days |
| Real-time Collaboration | 5 | Team communication | 1-7 days |
| External Integration | 6 | Third-party systems | 3-90 days |

## Complete Topic List

### 🆕 0. Hybrid Architecture Topics (7 topics) - EHR Intelligence Engine

**These topics implement the recommended hybrid architecture for the Flink EHR Intelligence Engine, providing transactional guarantees with EXACTLY_ONCE semantics.**

#### Architecture Overview
```
Flink TransactionalMultiSinkRouter → Atomic Writes
                    ↓
    ┌───────────────┼───────────────┐
    ↓               ↓               ↓
Central Topic   Critical Actions   Supporting Systems
```

| Topic Name | Partitions | Retention | Policy | Purpose |
|------------|------------|-----------|--------|---------|
| **`prod.ehr.events.enriched`** | 24 | 90 days | delete | **📊 CENTRAL SYSTEM OF RECORD** - Complete audit trail, replay capability, new consumer onboarding |
| **`prod.ehr.alerts.critical`** | 16 | 7 days | delete | **🚨 CRITICAL ALERTS** - Latency-sensitive alerts requiring immediate action |
| **`prod.ehr.fhir.upsert`** | 12 | 365 days | **compact** | **🏥 FHIR RESOURCES** - Latest state of FHIR resources (compacted for state management) |
| **`prod.ehr.analytics.events`** | 32 | 180 days | delete | **📈 ANALYTICS** - High-throughput feed for OLAP systems (ClickHouse, etc.) |
| **`prod.ehr.graph.mutations`** | 16 | 30 days | delete | **🕸️ GRAPH UPDATES** - Batched mutations for Neo4j clinical knowledge graph |
| **`prod.ehr.semantic.mesh`** | 4 | 365 days | **compact** | **📚 REFERENCE DATA** - Clinical knowledge updates for Flink broadcast state |
| **`prod.ehr.audit.logs`** | 8 | 2555 days | delete | **📋 AUDIT TRAIL** - 7-year retention for compliance and regulatory requirements |

#### Key Features
- **Transactional Guarantees**: All writes are atomic across topics
- **Intelligent Routing**: Content-based routing with clinical significance scoring
- **Compacted Topics**: State management for FHIR resources and semantic mesh
- **High Partitioning**: Analytics topic with 32 partitions for massive parallel consumption
- **Compliance**: 7-year audit retention for regulatory requirements

#### Producers
- **Primary**: `TransactionalMultiSinkRouter` (Flink Module 6)
- **Input Sources**: Modules 1-5 (Ingestion, Context, Semantic, Pattern, ML)

#### Consumers
- **Central Topic**: Audit tools, replay systems, new service onboarding
- **Critical Alerts**: Pager systems, clinical dashboards, emergency response
- **FHIR Upsert**: Google FHIR Store sink, clinical persistence layer
- **Analytics**: ClickHouse, Elasticsearch, reporting systems
- **Graph Mutations**: Neo4j sink, relationship analytics
- **Semantic Mesh**: Flink broadcast state, knowledge distribution
- **Audit Logs**: Compliance systems, regulatory reporting

#### Migration Notes
- Replaces the deprecated `clinical-events-unified.v1` single-topic approach
- Enables phased rollout: Phase 1 (Central) → Phase 2 (Critical) → Phase 3 (Supporting)
- Maintains legacy compatibility during transition period

---

### 1. Clinical Events (9 topics)

| Topic Name | Partitions | Retention | Compression | Description |
|------------|------------|-----------|-------------|-------------|
| `patient-events.v1` | 12 | 3 days | snappy | Patient admissions, discharges, updates |
| `medication-events.v1` | 12 | 3 days | snappy | Medication prescriptions, administrations |
| `observation-events.v1` | 12 | 3 days | snappy | Vital signs, lab results, observations |
| `safety-events.v1` | 12 | 7 days | snappy | Clinical alerts, contraindications |
| `vital-signs-events.v1` | 12 | 3 days | snappy | Processed vital signs from devices |
| `lab-result-events.v1` | 12 | 7 days | snappy | Laboratory results from instruments |
| `encounter-events.v1` | 8 | 3 days | snappy | Clinical encounters, appointments |
| `diagnostic-events.v1` | 8 | 7 days | snappy | Diagnoses, conditions, problem lists |
| `procedure-events.v1` | 8 | 7 days | snappy | Procedures, interventions, treatments |

### 2. Device Data (4 topics)

| Topic Name | Partitions | Retention | Compression | Description |
|------------|------------|-----------|-------------|-------------|
| `raw-device-data.v1` | 12 | 3 days | lz4 | Raw incoming device data |
| `validated-device-data.v1` | 12 | 3 days | snappy | Validated and enriched device data |
| `waveform-data.v1` | 24 | 1 day | lz4 | High-frequency waveform data |
| `device-telemetry.v1` | 4 | 7 days | snappy | Device status and health monitoring |

### 3. Runtime Layer (5 topics)

| Topic Name | Partitions | Retention | Policy | Description |
|------------|------------|-----------|--------|-------------|
| `enriched-patient-events.v1` | 12 | 7 days | delete | Semantically enriched patient events |
| `clinical-patterns.v1` | 8 | 30 days | delete | Detected clinical patterns |
| `pathway-adherence-events.v1` | 8 | 30 days | delete | Clinical pathway compliance |
| `semantic-mesh-updates.v1` | 4 | 7 days | compact | Knowledge broadcast updates |
| `patient-context-snapshots.v1` | 12 | 7 days | compact | Patient state snapshots |

### 4. Knowledge Base CDC (8 topics)

| Topic Name | Partitions | Retention | Policy | Description |
|------------|------------|-----------|--------|-------------|
| `kb3.clinical_protocols.changes` | 4 | 7 days | compact | Clinical protocol updates |
| `kb4.drug_calculations.changes` | 4 | 7 days | compact | Drug calculation rule updates |
| `kb4.dosing_rules.changes` | 4 | 7 days | compact | Dosing rule modifications |
| `kb4.weight_adjustments.changes` | 4 | 7 days | compact | Weight adjustment changes |
| `kb5.drug_interactions.changes` | 4 | 7 days | compact | Drug interaction updates |
| `kb6.validation_rules.changes` | 4 | 7 days | compact | Validation rule changes |
| `kb7.terminology.changes` | 4 | 7 days | compact | Medical terminology updates |
| `semantic-mesh.changes` | 4 | 7 days | compact | GraphDB semantic mesh updates |

### 5. Evidence Management (6 topics)

| Topic Name | Partitions | Retention | Compression | Description |
|------------|------------|-----------|-------------|-------------|
| `audit-events.v1` | 6 | 365 days | gzip | HIPAA/FDA compliance audit trail |
| `envelope-events.v1` | 6 | 90 days | snappy | Evidence envelope lifecycle |
| `evidence-requests.v1` | 4 | 7 days | snappy | Evidence retrieval requests |
| `evidence-validations.v1` | 4 | 30 days | snappy | Evidence validation results |
| `clinical-reasoning-events.v1` | 8 | 30 days | snappy | AI/ML reasoning outputs |
| `inference-results.v1` | 8 | 30 days | snappy | Clinical inference chain results |

### 6. Workflow & Orchestration (6 topics)

| Topic Name | Partitions | Retention | Description |
|------------|------------|-----------|-------------|
| `workflow-events.v1` | 8 | 7 days | Workflow state transitions |
| `workflow-ui-interactions.v1` | 8 | 3 days | UI events requiring processing |
| `clinical-overrides.v1` | 4 | 90 days | Clinical override decisions |
| `task-assignments.v1` | 8 | 7 days | Clinical task assignments |
| `decision-support-events.v1` | 8 | 30 days | Clinical decision triggers |
| `orchestration-commands.v1` | 4 | 3 days | Cross-service orchestration |

### 7. SLA & Monitoring (6 topics)

| Topic Name | Partitions | Retention | Description |
|------------|------------|-----------|-------------|
| `sla-measurements.v1` | 8 | 7 days | Real-time SLA metrics |
| `sla-violations.v1` | 4 | 30 days | SLA violation events |
| `performance-metrics.v1` | 8 | 7 days | System performance indicators |
| `clinical-metrics.v1` | 6 | 90 days | Clinical outcome metrics |
| `usage-analytics.v1` | 4 | 30 days | Feature usage tracking |
| `alert-notifications.v1` | 6 | 7 days | System and clinical alerts |

### 8. Cache & Optimization (4 topics)

| Topic Name | Partitions | Retention | Description |
|------------|------------|-----------|-------------|
| `cache-invalidation.v1` | 8 | 1 day | L1/L2 cache invalidation |
| `prefetch-predictions.v1` | 4 | 3 days | ML-based prefetch predictions |
| `cache-warmup.v1` | 4 | 1 day | Cache warming commands |
| `query-patterns.v1` | 4 | 7 days | Query pattern analysis |

### 9. Dead Letter Queues (9 topics)

| Topic Name | Partitions | Retention | Min ISR | Description |
|------------|------------|-----------|---------|-------------|
| `failed-validation.v1` | 4 | 30 days | 2 | Validation failures |
| `critical-data-dlq.v1` | 4 | 90 days | 3 | Critical data failures |
| `poison-messages.v1` | 2 | 365 days | 2 | Repeatedly failing messages |
| `sink-write-failures.v1` | 6 | 14 days | 2 | Sink write failures |
| `critical-sink-failures.v1` | 4 | 90 days | 3 | Critical sink failures |
| `poison-messages-stage2.v1` | 2 | 365 days | 2 | Stage 2 poison messages |
| `processing-errors.v1` | 4 | 7 days | 2 | General processing errors |
| `integration-failures.v1` | 4 | 30 days | 2 | External system failures |
| `flink-checkpoint-failures.v1` | 2 | 7 days | 2 | Flink checkpoint issues |

### 10. Real-time Collaboration (5 topics)

| Topic Name | Partitions | Retention | Policy | Description |
|------------|------------|-----------|--------|-------------|
| `clinical-chat.v1` | 8 | 7 days | delete | Clinical team communications |
| `notification-push.v1` | 8 | 3 days | delete | Push notification events |
| `presence-updates.v1` | 4 | 1 day | compact | User presence/availability |
| `collaboration-events.v1` | 6 | 3 days | delete | Multi-user collaboration |
| `graphql-subscriptions.v1` | 8 | 1 day | delete | GraphQL subscription updates |

### 11. External Integration (6 topics)

| Topic Name | Partitions | Retention | Max Message | Description |
|------------|------------|-----------|-------------|-------------|
| `hl7-messages.v1` | 8 | 7 days | 1MB | HL7 message integration |
| `fhir-bundles.v1` | 8 | 7 days | 10MB | FHIR bundle exchanges |
| `external-lab-results.v1` | 6 | 30 days | 1MB | External lab results |
| `pharmacy-orders.v1` | 6 | 7 days | 1MB | Pharmacy system orders |
| `billing-events.v1` | 4 | 90 days | 1MB | Billing and claims |
| `google-healthcare-sync.v1` | 6 | 3 days | 1MB | Google Healthcare sync |

## Topic Specifications

### Message Format Standards

```json
{
  "event_id": "uuid-v4",
  "event_type": "string",
  "timestamp": "ISO-8601",
  "version": "1.0.0",
  "source": "service-name",
  "correlation_id": "uuid-v4",
  "patient_id": "string (for clinical events)",
  "data": {
    // Event-specific payload
  },
  "metadata": {
    "user_id": "string",
    "session_id": "string",
    "client_ip": "string",
    "trace_id": "string"
  }
}
```

### Key Patterns

| Topic Category | Key Pattern | Partitioning Strategy |
|----------------|-------------|----------------------|
| Clinical Events | `patient_id` | Hash partitioning |
| Device Data | `device_id` | Hash partitioning |
| CDC Topics | `entity_id` | Hash partitioning |
| DLQ Topics | `original_topic:offset` | Round-robin |
| Monitoring | `service_name` | Hash partitioning |

## Integration Matrix

### Flink Processing Integration

| Flink Job | Input Topics | Output Topics | Parallelism |
|-----------|--------------|---------------|-------------|
| Patient Enrichment | 5 clinical topics | `enriched-patient-events.v1` | 4 |
| Pattern Detection | `enriched-patient-events.v1` | `clinical-patterns.v1` | 2 |
| Pathway Adherence | 2 enriched + workflow | `pathway-adherence-events.v1` | 2 |
| CDC Processor | 8 CDC topics | `semantic-mesh-updates.v1` | 1 |
| Snapshot Generator | `enriched-patient-events.v1` | `patient-context-snapshots.v1` | 2 |

### Runtime Layer Services

| Service | Consumes | Produces | Port |
|---------|----------|----------|------|
| Evidence Envelope | 2 topics | 3 topics | 8020 |
| SLA Monitoring | 2 topics | 3 topics | 8021 |
| L1 Cache Prefetcher | 7 topics | 1 topic | 8022 |
| Event Bus Orchestrator | 3 topics | 4 topics | 8023 |
| Query Router | - | 2 topics | 8024 |

## Usage Examples

### Producer Example (Python)

```python
from confluent_kafka import Producer
import json
from datetime import datetime
import uuid

# Configure producer
producer = Producer({
    'bootstrap.servers': 'localhost:9092',
    'client.id': 'patient-service',
    'acks': 'all',
    'compression.type': 'snappy'
})

# Create event
event = {
    'event_id': str(uuid.uuid4()),
    'event_type': 'patient.admission',
    'timestamp': datetime.utcnow().isoformat(),
    'version': '1.0.0',
    'source': 'patient-service',
    'patient_id': 'P12345',
    'data': {
        'admission_type': 'emergency',
        'department': 'cardiology',
        'reason': 'chest pain'
    },
    'metadata': {
        'user_id': 'U67890',
        'session_id': str(uuid.uuid4()),
        'trace_id': str(uuid.uuid4())
    }
}

# Publish to topic
producer.produce(
    topic='patient-events.v1',
    key=event['patient_id'],
    value=json.dumps(event),
    callback=delivery_report
)

producer.flush()
```

### Consumer Example (Python)

```python
from confluent_kafka import Consumer, KafkaError
import json

# Configure consumer
consumer = Consumer({
    'bootstrap.servers': 'localhost:9092',
    'group.id': 'clinical-reasoning-service',
    'auto.offset.reset': 'earliest',
    'enable.auto.commit': True
})

# Subscribe to topics
consumer.subscribe([
    'enriched-patient-events.v1',
    'clinical-patterns.v1'
])

# Consume messages
while True:
    msg = consumer.poll(1.0)

    if msg is None:
        continue

    if msg.error():
        if msg.error().code() == KafkaError._PARTITION_EOF:
            continue
        else:
            print(f"Consumer error: {msg.error()}")
            break

    # Process message
    event = json.loads(msg.value().decode('utf-8'))
    process_clinical_event(event)

    # Commit offset
    consumer.commit()
```

### Flink Integration (Java)

```java
// Configure Kafka source
KafkaSource<PatientEvent> kafkaSource = KafkaSource.<PatientEvent>builder()
    .setBootstrapServers("localhost:9092")
    .setTopics("patient-events.v1", "medication-events.v1")
    .setGroupId("flink-enrichment-job")
    .setStartingOffsets(OffsetsInitializer.earliest())
    .setValueOnlyDeserializer(new PatientEventSchema())
    .build();

// Create stream
DataStream<PatientEvent> patientStream = env
    .fromSource(kafkaSource, WatermarkStrategy.noWatermarks(), "patient-events");

// Process and enrich
DataStream<EnrichedPatientEvent> enrichedStream = patientStream
    .keyBy(PatientEvent::getPatientId)
    .process(new EventEnrichmentFunction());

// Sink to output topic
enrichedStream.sinkTo(
    KafkaSink.<EnrichedPatientEvent>builder()
        .setBootstrapServers("localhost:9092")
        .setRecordSerializer(new EnrichedEventSerializer())
        .setDeliverGuarantee(DeliveryGuarantee.EXACTLY_ONCE)
        .build()
);
```

## Quick Reference

### Common Commands

```bash
# Create all topics
python scripts/setup-all-topics.py --action create --environment production

# Validate topics
python scripts/setup-all-topics.py --action validate

# List topics by category
python scripts/setup-all-topics.py --action list --category clinical_events

# Describe specific topic
python scripts/setup-all-topics.py --action describe --topic patient-events.v1

# Monitor consumer lag
kafka-consumer-groups --bootstrap-server localhost:9092 \
  --group flink-stream-processor --describe

# View topic metrics
kafka-topics --bootstrap-server localhost:9092 \
  --describe --topic patient-events.v1
```

### Environment Variables

```bash
# Development
export KAFKA_BOOTSTRAP_SERVERS=localhost:9092

# Staging/Production (Confluent Cloud)
export KAFKA_BOOTSTRAP_SERVERS=pkc-619z3.us-east1.gcp.confluent.cloud:9092
export KAFKA_API_KEY=your-api-key
export KAFKA_API_SECRET=your-api-secret
export KAFKA_SECURITY_PROTOCOL=SASL_SSL
export KAFKA_SASL_MECHANISM=PLAIN
```

### Topic Naming Convention

```
Pattern: {domain}-{event-type}.v{version}

Examples:
- patient-events.v1
- clinical-patterns.v2
- kb4.drug_calculations.changes
```

### Retention Policy Guidelines

| Data Type | Retention | Justification |
|-----------|-----------|---------------|
| Clinical Events | 3-7 days | Replicated to permanent storage |
| Audit Events | 365 days | HIPAA compliance requirement |
| CDC Events | 7 days | Knowledge synchronization window |
| DLQ Critical | 90-365 days | Critical failure recovery |
| Cache Events | 1 day | Temporary optimization data |
| Monitoring | 7-30 days | Performance analysis window |

## Performance Considerations

### Partition Count Guidelines

| Throughput | Partitions | Use Case |
|------------|------------|----------|
| < 100 msg/s | 2-4 | Low-volume topics, DLQs |
| 100-1000 msg/s | 4-8 | Standard clinical events |
| 1000-5000 msg/s | 8-12 | High-volume patient events |
| > 5000 msg/s | 12-24 | Waveform data, device streams |

### Replication Settings

| Environment | Replication Factor | Min ISR | Acks |
|-------------|-------------------|---------|------|
| Development | 1 | 1 | 1 |
| Staging | 2 | 2 | all |
| Production | 3 | 2 | all |
| Critical (Audit/DLQ) | 3 | 3 | all |

## Troubleshooting

### Common Issues

1. **High Consumer Lag**
   - Increase consumer instances
   - Optimize processing logic
   - Check partition distribution

2. **Message Loss**
   - Verify producer acks setting
   - Check min.insync.replicas
   - Monitor broker health

3. **DLQ Messages Accumulating**
   - Review validation rules
   - Check schema compatibility
   - Implement retry logic

4. **Performance Degradation**
   - Monitor partition skew
   - Check compression settings
   - Review batch configurations

## Compliance & Security

### HIPAA Requirements
- Audit topic retention: 365 days minimum (Hybrid architecture: 7 years for `prod.ehr.audit.logs`)
- Encryption in transit: TLS 1.2+
- Access control: ACLs per service
- PHI handling: Proper key management

### Data Classification

| Classification | Topics | Security Requirements |
|----------------|--------|----------------------|
| PHI (Protected) | Clinical events, audit, hybrid architecture topics | Encryption, access control |
| Sensitive | Evidence, reasoning | Restricted access |
| Internal | Monitoring, cache | Standard security |
| Public | None | N/A |

## 🆕 Hybrid Architecture Quick Start

### Creating the Topics

```bash
# Run the automated creation script
cd /backend/shared-infrastructure/kafka
./create-hybrid-architecture-topics.sh

# Or create manually with proper settings
kafka-topics --create \
  --bootstrap-server localhost:9092 \
  --topic prod.ehr.events.enriched \
  --partitions 24 \
  --replication-factor 3 \
  --config retention.ms=7776000000 \
  --config cleanup.policy=delete \
  --config compression.type=snappy \
  --config min.insync.replicas=2
```

### Verifying Topic Configuration

```bash
# List all hybrid topics
kafka-topics --bootstrap-server localhost:9092 --list | grep "^prod.ehr"

# Describe specific topic
kafka-topics --bootstrap-server localhost:9092 \
  --describe --topic prod.ehr.events.enriched
```

### Monitoring

```bash
# Check consumer lag for central topic
kafka-consumer-groups --bootstrap-server localhost:9092 \
  --group flink-consumer --describe

# Monitor topic throughput
kafka-run-class kafka.tools.JmxTool \
  --object-name kafka.server:type=BrokerTopicMetrics,name=MessagesInPerSec,topic=prod.ehr.events.enriched
```

### Testing the Pipeline

```bash
# Produce test event to verify routing
echo '{"patientId":"test-123","eventType":"CRITICAL","significance":0.95}' | \
  kafka-console-producer \
    --bootstrap-server localhost:9092 \
    --topic prod.ehr.events.enriched

# Consume from action topics to verify distribution
kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic prod.ehr.alerts.critical \
  --from-beginning
```

---

**Version**: 1.0.0
**Last Updated**: January 26, 2025
**Maintained By**: CardioFit Platform Team
**Repository**: `/backend/shared-infrastructure/kafka/`