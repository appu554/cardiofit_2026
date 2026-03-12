# Kafka Infrastructure for CardioFit Platform

## Overview

This directory contains the complete Kafka event streaming infrastructure for the CardioFit Clinical Synthesis Hub platform. It provides a unified event backbone for real-time clinical data processing, FHIR compliance, and cross-service communication.

## 🏗️ Architecture

The Kafka infrastructure supports:
- **Real-time Clinical Events**: Patient data, medications, observations, and safety alerts
- **Stream Processing**: Apache Flink integration for event enrichment and pattern detection
- **Change Data Capture (CDC)**: Knowledge base synchronization via Debezium
- **Evidence Management**: Clinical evidence envelope and audit trail
- **Performance Monitoring**: SLA tracking and system metrics
- **Error Handling**: Comprehensive DLQ strategy for data resilience

## 📁 Directory Structure

```
kafka/
├── config/
│   └── topics-config.yaml      # Centralized topic configuration
├── scripts/
│   └── setup-all-topics.py     # Topic management script
├── schemas/
│   ├── avro/                   # Avro schemas for structured events
│   └── json/                   # JSON schemas for flexible events
└── README.md                   # This file
```

## 🚀 Quick Start

### Option A: Lightweight local stack (recommended for HPI telemetry work)

This repo now includes a single-broker Kafka + Zookeeper combo that starts in seconds and is enough for
`hpi.session.events`, `hpi.escalation.events`, and `hpi.calibration.data`.

```bash
cd backend/shared-infrastructure/kafka
./start-kafka-lite.sh
./scripts/create-hpi-lite-topics.sh

# Optional: tail events
python scripts/hpi_telemetry_consumer.py --topic hpi.session.events

# Run KB-22 with the lightweight broker
export KAFKA_ENABLED=true
export KAFKA_BOOTSTRAP_SERVERS=localhost:9092
```

The broker listens on `localhost:9092`. Use `docker compose -f docker-compose.hpi-lite.yml down -v`
to stop and clean up when you are finished.

### Prerequisites

- Python 3.8+
- Kafka cluster (local or Confluent Cloud)
- Required Python packages:
  ```bash
  pip install confluent-kafka pyyaml
  ```

### Option B: Full Confluent stack (multi-broker)

Use the existing `start-kafka.sh` script when you need schema registry, Kafka Connect, or KSQL DB.
This will pull gigabytes of images; expect the first run to take several minutes.

### Initial Setup (full stack)

1. **Review Configuration**
   ```bash
   cd backend/shared-infrastructure/kafka
   cat config/topics-config.yaml
   ```

2. **Create All Topics (Development)**
   ```bash
   python scripts/setup-all-topics.py --action create --environment development
   ```

3. **Validate Topics**
   ```bash
   python scripts/setup-all-topics.py --action validate --environment development
   ```

## 📊 Topic Categories

### 1. Clinical Events (`clinical_events`)
Core patient and clinical data events:
- `patient-events.v1` - Patient admissions, discharges, updates
- `medication-events.v1` - Prescriptions and administrations
- `observation-events.v1` - Vital signs and lab results
- `safety-events.v1` - Clinical alerts and contraindications

### 2. Device Data (`device_data`)
Medical device and IoT integration:
- `raw-device-data.v1` - Raw device readings
- `validated-device-data.v1` - Validated device data
- `waveform-data.v1` - High-frequency ECG/EEG data

### 3. Runtime Layer (`runtime_layer`)
Flink processing and enrichment:
- `enriched-patient-events.v1` - Semantically enriched events
- `clinical-patterns.v1` - Detected patterns and anomalies
- `pathway-adherence-events.v1` - Protocol compliance

### 4. Knowledge Base CDC (`knowledge_base_cdc`)
Real-time knowledge updates:
- `kb4.drug_calculations.changes` - Drug calculation rules
- `kb5.drug_interactions.changes` - Interaction database
- `kb3.clinical_protocols.changes` - Protocol updates

### 5. Evidence Management (`evidence_management`)
Clinical evidence and audit:
- `audit-events.v1` - HIPAA/FDA compliance audit trail
- `envelope-events.v1` - Evidence lifecycle events
- `clinical-reasoning-events.v1` - AI/ML outputs

### 6. Monitoring (`monitoring`)
Performance and SLA tracking:
- `sla-measurements.v1` - Real-time SLA metrics
- `performance-metrics.v1` - System performance
- `alert-notifications.v1` - System alerts

### 7. Dead Letter Queues (`dlq`)
Error handling and recovery:
- `failed-validation.v1` - Validation failures
- `critical-data-dlq.v1` - Critical data failures
- `poison-messages.v1` - Repeatedly failing messages

## 🛠️ Topic Management

### Script Usage

```bash
python scripts/setup-all-topics.py [OPTIONS]

Options:
  --action, -a        Action to perform [create|validate|list|describe|document|delete]
  --environment, -e   Environment [development|staging|production]
  --category         Filter by topic category
  --topic            Specific topic name (for describe/delete)
  --dry-run          Preview changes without applying
  --confirm          Confirm destructive operations
```

### Common Operations

#### Create Topics by Category
```bash
# Create only clinical event topics
python scripts/setup-all-topics.py --action create --category clinical_events

# Create only DLQ topics
python scripts/setup-all-topics.py --action create --category dlq
```

#### List All Topics
```bash
python scripts/setup-all-topics.py --action list
```

#### Describe a Topic
```bash
python scripts/setup-all-topics.py --action describe --topic patient-events.v1
```

#### Generate Documentation
```bash
python scripts/setup-all-topics.py --action document
```

#### Validate Configuration
```bash
python scripts/setup-all-topics.py --action validate
```

## 🔧 Integration Guide

### For Microservices

#### Producer Example (Python)
```python
from confluent_kafka import Producer
import json

# Configure producer
producer = Producer({
    'bootstrap.servers': 'localhost:9092',
    'client.id': 'patient-service'
})

# Publish event
event = {
    'patient_id': 'P12345',
    'event_type': 'admission',
    'timestamp': '2025-01-26T10:00:00Z',
    'data': {...}
}

producer.produce(
    topic='patient-events.v1',
    key=event['patient_id'],
    value=json.dumps(event)
)
producer.flush()
```

#### Consumer Example (Python)
```python
from confluent_kafka import Consumer

# Configure consumer
consumer = Consumer({
    'bootstrap.servers': 'localhost:9092',
    'group.id': 'clinical-reasoning-service',
    'auto.offset.reset': 'earliest'
})

# Subscribe to topics
consumer.subscribe(['patient-events.v1', 'medication-events.v1'])

# Consume events
while True:
    msg = consumer.poll(1.0)
    if msg is None:
        continue

    event = json.loads(msg.value())
    process_event(event)
```

### For Flink Processing

#### Flink Configuration
```properties
# backend/shared-infrastructure/flink-processing/config/kafka-topics.properties
input.topics=patient-events.v1,medication-events.v1,safety-events.v1
output.topic.enriched=enriched-patient-events.v1
output.topic.patterns=clinical-patterns.v1
```

#### Flink Job Example (Java)
```java
// Configure Kafka source
KafkaSource<PatientEvent> source = KafkaSource.<PatientEvent>builder()
    .setBootstrapServers("localhost:9092")
    .setTopics("patient-events.v1", "medication-events.v1")
    .setGroupId("flink-processor")
    .setDeserializer(new PatientEventSchema())
    .build();

// Process stream
DataStream<PatientEvent> stream = env
    .fromSource(source, WatermarkStrategy.noWatermarks(), "patient-events");
```

### For Runtime Layer Services

#### Service Configuration
```yaml
# backend/shared-infrastructure/runtime-layer/config/kafka-topics.yaml
evidence_envelope:
  audit: audit-events.v1
  events: envelope-events.v1

sla_monitoring:
  measurements: sla-measurements.v1
  violations: sla-violations.v1
```

## 📋 Best Practices

### Topic Naming
- **Pattern**: `{domain}-{event-type}.v{version}`
- **Examples**: `patient-events.v1`, `clinical-patterns.v2`
- **Versioning**: Major changes require new version

### Partitioning Strategy
- **Patient-keyed topics**: 12 partitions (hash by patient_id)
- **High-throughput**: 12-24 partitions
- **Audit/compliance**: 6 partitions
- **DLQ**: 4 partitions
- **Low-volume**: 2 partitions

### Retention Policies
- **Clinical events**: 3-7 days (replicated to permanent storage)
- **Audit events**: 365 days (compliance requirement)
- **CDC events**: 7 days
- **Metrics**: 7 days
- **DLQ**: 30-365 days based on criticality

### Compression
- **Snappy**: Real-time topics (balance speed/compression)
- **GZIP**: Audit/compliance (maximize compression)
- **LZ4**: High-throughput streaming

### Producer Guidelines
1. Always set appropriate keys for partitioning
2. Use idempotent producers for exactly-once semantics
3. Implement proper error handling and retries
4. Monitor producer metrics

### Consumer Guidelines
1. Use consumer groups for scaling
2. Implement proper offset management
3. Handle rebalancing gracefully
4. Monitor consumer lag

## 🔍 Monitoring

### Key Metrics to Track
- **Topic Lag**: Consumer group lag per topic
- **Throughput**: Messages/sec per topic
- **Error Rate**: DLQ message rate
- **Partition Balance**: Even distribution across partitions

### Grafana Dashboards
Import dashboard configurations from:
```
backend/shared-infrastructure/monitoring/kafka-dashboards.json
```

### Alerts
Configure alerts for:
- Consumer lag > 1000 messages
- DLQ message rate > 10/min
- Topic unavailable
- Partition offline

## 🚨 Troubleshooting

### Common Issues

#### Topic Creation Fails
```bash
# Check Kafka connectivity
python scripts/setup-all-topics.py --action list

# Verify credentials (Confluent Cloud)
export KAFKA_USERNAME="your-api-key"
export KAFKA_PASSWORD="your-api-secret"
```

#### Consumer Lag High
```bash
# Check consumer group status
kafka-consumer-groups --bootstrap-server localhost:9092 \
  --group your-consumer-group --describe
```

#### Messages Going to DLQ
1. Check validation rules in Stage 1
2. Verify FHIR compliance in Stage 2
3. Review error messages in DLQ topics

### Recovery Procedures

#### Replay from DLQ
```python
# Consume from DLQ and reprocess
consumer = Consumer({'group.id': 'dlq-recovery'})
consumer.subscribe(['failed-validation.v1'])

while True:
    msg = consumer.poll(1.0)
    if msg:
        # Attempt reprocessing
        reprocess_message(msg)
```

## 🔐 Security

### Authentication (Confluent Cloud)
```python
config = {
    'bootstrap.servers': 'pkc-xxxxx.us-east1.gcp.confluent.cloud:9092',
    'security.protocol': 'SASL_SSL',
    'sasl.mechanism': 'PLAIN',
    'sasl.username': 'API_KEY',
    'sasl.password': 'API_SECRET'
}
```

### Authorization
- Use ACLs to control topic access
- Implement service-specific credentials
- Rotate API keys regularly

### Encryption
- TLS for data in transit
- Consider encryption at rest for sensitive topics

## 📚 Environment Configuration

### Development
- Local Kafka or Docker Compose
- 1-2 partitions per topic
- Shorter retention (1 day)

### Staging
- Confluent Cloud or dedicated cluster
- 4-8 partitions per topic
- Standard retention (3-7 days)

### Production
- Confluent Cloud with HA
- 8-24 partitions per topic
- Compliance-driven retention

## 🔄 Migration Guide

### Adding New Topics
1. Add configuration to `topics-config.yaml`
2. Run validation: `python scripts/setup-all-topics.py --action validate`
3. Create topic: `python scripts/setup-all-topics.py --action create --topic new-topic.v1`
4. Update consumer groups
5. Deploy producer changes

### Updating Topic Configuration
1. Modify `topics-config.yaml`
2. Some changes require topic recreation:
   - Partition count decrease
   - Replication factor change
3. Dynamic configuration updates:
   - Retention period
   - Compression type

## 📖 Additional Resources

- [Apache Kafka Documentation](https://kafka.apache.org/documentation/)
- [Confluent Cloud Documentation](https://docs.confluent.io/cloud/current/)
- [Flink Kafka Connector](https://nightlies.apache.org/flink/flink-docs-stable/docs/connectors/datastream/kafka/)
- [Debezium CDC](https://debezium.io/documentation/)

## 🤝 Contributing

1. Update `topics-config.yaml` for new topics
2. Document producer/consumer relationships
3. Add schemas to `schemas/` directory
4. Update this README with examples
5. Test in development environment first

## 📞 Support

For issues or questions:
- Check troubleshooting section
- Review Kafka logs
- Contact DevOps team
- Open issue in project repository

---

**Last Updated**: January 26, 2025
**Version**: 1.0.0
**Maintained By**: CardioFit Platform Team
