# KB7 Terminology Releases CDC Setup Guide

Complete guide for setting up Change Data Capture (CDC) for KB-7 Terminology releases using Debezium and Kafka.

## Overview

This CDC setup enables real-time notification of KB-7 terminology updates to downstream services. When the KB-7 Knowledge Factory pipeline completes loading terminology data into GraphDB, it commits a record to PostgreSQL, which triggers Debezium CDC to publish an event to Kafka.

### Architecture Flow

```
KB-7 Knowledge Factory Pipeline
    │
    ▼
GCS: gs://bucket/version/kb7-kernel.ttl
    │
    ▼
GraphDB: SPARQL LOAD (5-15 min for ~14M triples)
    │
    ▼
Health Check: Triple count + sample query
    │
    ▼
PostgreSQL: INSERT INTO kb_releases (status='ACTIVE')  ← Commit-Last Strategy
    │
    ▼
Debezium: kb7-terminology-releases-cdc connector
    │
    ▼
Kafka Topic: kb7.terminology.public.kb_releases
    │
    ▼
Downstream Services:
├── Flink BroadcastStream → Hot-swap terminology cache
├── Clinical Reasoning → Refresh SNOMED/RxNorm/LOINC mappings
├── KB Services → Update local terminology copies
└── Notification Service → Alert administrators
```

## Prerequisites

### Required Infrastructure

| Component | Version | Port | Purpose |
|-----------|---------|------|---------|
| PostgreSQL | 15+ | 5432 | Source database with logical replication |
| Apache Kafka | 7.5.0 | 9092/9093 | Event streaming platform |
| Zookeeper | 7.5.0 | 2181 | Kafka coordination |
| Debezium Connect | 2.5 | 8083 | CDC connector runtime |

### Docker Network

All containers must be on the same Docker network for inter-container communication:

```bash
docker network create kb-network
```

## Step 1: PostgreSQL Setup

### 1.1 Enable Logical Replication

PostgreSQL must be configured for logical replication (required for Debezium):

```bash
# Connect to PostgreSQL container
docker exec -it <postgres-container> psql -U postgres

# Enable logical replication
ALTER SYSTEM SET wal_level = 'logical';
ALTER SYSTEM SET max_replication_slots = 4;
ALTER SYSTEM SET max_wal_senders = 4;

# Restart PostgreSQL
docker restart <postgres-container>

# Verify configuration
SHOW wal_level;  -- Should return: logical
```

### 1.2 Create Database and User

```sql
-- Create database
CREATE DATABASE kb_terminology;

-- Create CDC user with replication privileges
CREATE ROLE debezium WITH LOGIN PASSWORD 'debezium_password' REPLICATION;

-- Grant permissions
\c kb_terminology
GRANT SELECT ON ALL TABLES IN SCHEMA public TO debezium;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO debezium;
```

### 1.3 Create kb_releases Table

Run the migration script:

```bash
# From the cdc-connectors directory
psql -h localhost -p 5432 -U postgres -d kb_terminology -f sql/kb7-releases-schema.sql
```

Or manually create the table:

```sql
-- KB-7 Terminology Releases Outbox Table
CREATE TABLE IF NOT EXISTS kb_releases (
    id SERIAL PRIMARY KEY,

    -- Version identification
    version_id VARCHAR(50) UNIQUE NOT NULL,

    -- Timestamps
    release_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    graphdb_load_started_at TIMESTAMP WITH TIME ZONE,
    graphdb_load_completed_at TIMESTAMP WITH TIME ZONE,

    -- Source terminology versions
    snomed_version VARCHAR(50),
    rxnorm_version VARCHAR(50),
    loinc_version VARCHAR(50),

    -- Content metrics
    triple_count BIGINT,
    concept_count INTEGER,

    -- File information
    kernel_checksum VARCHAR(64),
    gcs_uri VARCHAR(500),

    -- GraphDB information
    graphdb_repository VARCHAR(100) DEFAULT 'kb7-terminology',
    graphdb_endpoint VARCHAR(500),

    -- Status tracking
    status VARCHAR(20) DEFAULT 'PENDING'
        CHECK (status IN ('PENDING', 'LOADING', 'ACTIVE', 'ARCHIVED', 'FAILED')),
    error_message TEXT,

    -- Metadata
    created_by VARCHAR(100) DEFAULT 'kb-factory-pipeline',
    notes TEXT
);

-- Enable CDC (REQUIRED for Debezium)
ALTER TABLE kb_releases REPLICA IDENTITY FULL;

-- Create indexes
CREATE INDEX idx_kb_releases_version ON kb_releases(version_id);
CREATE INDEX idx_kb_releases_status ON kb_releases(status);
CREATE INDEX idx_kb_releases_date ON kb_releases(release_date DESC);

-- View for current active release
CREATE OR REPLACE VIEW current_kb_release AS
SELECT * FROM kb_releases
WHERE status = 'ACTIVE'
ORDER BY release_date DESC
LIMIT 1;

-- Auto-archive trigger
CREATE OR REPLACE FUNCTION archive_previous_releases()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'ACTIVE' THEN
        UPDATE kb_releases
        SET status = 'ARCHIVED'
        WHERE id != NEW.id AND status = 'ACTIVE';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_archive_previous_releases
    AFTER UPDATE ON kb_releases
    FOR EACH ROW
    WHEN (NEW.status = 'ACTIVE')
    EXECUTE FUNCTION archive_previous_releases();
```

## Step 2: Kafka Setup

### 2.1 Start Zookeeper

```bash
docker run -d \
  --name zookeeper \
  --network kb-network \
  -p 2181:2181 \
  -e ZOOKEEPER_CLIENT_PORT=2181 \
  confluentinc/cp-zookeeper:7.5.0
```

### 2.2 Start Kafka Broker

```bash
docker run -d \
  --name kafka \
  --network kb-network \
  -p 9092:9092 \
  -e KAFKA_BROKER_ID=1 \
  -e KAFKA_ZOOKEEPER_CONNECT=zookeeper:2181 \
  -e KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://kafka:9092 \
  -e KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1 \
  confluentinc/cp-kafka:7.5.0
```

### 2.3 Start Debezium Kafka Connect

```bash
docker run -d \
  --name kafka-connect \
  --network kb-network \
  -p 8083:8083 \
  -e BOOTSTRAP_SERVERS=kafka:9092 \
  -e GROUP_ID=1 \
  -e CONFIG_STORAGE_TOPIC=connect_configs \
  -e OFFSET_STORAGE_TOPIC=connect_offsets \
  -e STATUS_STORAGE_TOPIC=connect_statuses \
  quay.io/debezium/connect:2.5
```

### 2.4 Verify Kafka Connect

```bash
# Wait for startup (30-60 seconds)
curl http://localhost:8083/

# Check available connectors
curl http://localhost:8083/connector-plugins | jq '.[].class' | grep -i postgres
# Should show: io.debezium.connector.postgresql.PostgresConnector
```

## Step 3: Deploy CDC Connector

### 3.1 Connector Configuration

The connector config is stored at `configs/kb7-terminology-releases-cdc.json`:

```json
{
  "name": "kb7-terminology-releases-cdc",
  "config": {
    "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
    "plugin.name": "pgoutput",

    "database.hostname": "<postgres-container-name>",
    "database.port": "5432",
    "database.user": "postgres",
    "database.password": "password",
    "database.dbname": "kb_terminology",
    "database.server.name": "kb7_releases",

    "topic.prefix": "kb7.terminology",
    "table.include.list": "public.kb_releases",

    "transforms": "unwrap",
    "transforms.unwrap.type": "io.debezium.transforms.ExtractNewRecordState",
    "transforms.unwrap.drop.tombstones": "false",
    "transforms.unwrap.delete.handling.mode": "rewrite",

    "key.converter": "org.apache.kafka.connect.json.JsonConverter",
    "value.converter": "org.apache.kafka.connect.json.JsonConverter",
    "key.converter.schemas.enable": "false",
    "value.converter.schemas.enable": "false",

    "snapshot.mode": "initial",
    "slot.name": "kb7_releases_cdc_slot",
    "publication.name": "kb7_releases_cdc_publication",

    "heartbeat.interval.ms": "10000"
  }
}
```

### 3.2 Deploy the Connector

```bash
curl -X POST http://localhost:8083/connectors \
  -H "Content-Type: application/json" \
  -d @configs/kb7-terminology-releases-cdc.json
```

Or use the deployment script:

```bash
cd backend/shared-infrastructure/kafka/cdc-connectors/scripts
./deploy-all-cdc-connectors.sh deploy
```

### 3.3 Verify Connector Status

```bash
# Check connector status
curl http://localhost:8083/connectors/kb7-terminology-releases-cdc/status | jq

# Expected output:
{
  "name": "kb7-terminology-releases-cdc",
  "connector": {
    "state": "RUNNING",
    "worker_id": "172.23.0.6:8083"
  },
  "tasks": [
    {
      "id": 0,
      "state": "RUNNING",
      "worker_id": "172.23.0.6:8083"
    }
  ],
  "type": "source"
}
```

## Step 4: Verify CDC Flow

### 4.1 Check Kafka Topics

```bash
docker exec kafka kafka-topics --list --bootstrap-server localhost:9092 | grep kb7

# Expected topics:
# kb7.terminology.public.kb_releases  (CDC data)
# kb7.terminology.releases            (heartbeats)
```

### 4.2 Test with Sample Insert

```bash
# Insert test record
docker exec <postgres-container> psql -U postgres -d kb_terminology -c "
INSERT INTO kb_releases (
    version_id,
    snomed_version,
    rxnorm_version,
    loinc_version,
    triple_count,
    status
) VALUES (
    'v2024-12-03-test',
    '2024-09',
    '12012025',
    '2.77',
    14500000,
    'ACTIVE'
);"
```

### 4.3 Consume CDC Event

```bash
docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic kb7.terminology.public.kb_releases \
  --from-beginning \
  --max-messages 1 | jq
```

Expected output:

```json
{
  "id": 1,
  "version_id": "v2024-12-03-test",
  "release_date": "2025-12-03T08:20:02.811795Z",
  "snomed_version": "2024-09",
  "rxnorm_version": "12012025",
  "loinc_version": "2.77",
  "triple_count": 14500000,
  "status": "ACTIVE",
  "graphdb_repository": "kb7-terminology",
  "created_by": "kb-factory-pipeline",
  "__deleted": "false"
}
```

## Step 5: Downstream Consumer Integration

### 5.1 Flink BroadcastStream Consumer (Java)

```java
// Kafka source for CDC events
KafkaSource<String> cdcSource = KafkaSource.<String>builder()
    .setBootstrapServers("kafka:9092")
    .setTopics("kb7.terminology.public.kb_releases")
    .setGroupId("flink-terminology-consumer")
    .setStartingOffsets(OffsetsInitializer.earliest())
    .setValueOnlyDeserializer(new SimpleStringSchema())
    .build();

// Create BroadcastStream for terminology updates
DataStream<String> cdcStream = env.fromSource(
    cdcSource,
    WatermarkStrategy.noWatermarks(),
    "KB7 CDC Source"
);

// Broadcast to all parallel instances
MapStateDescriptor<String, TerminologyRelease> descriptor =
    new MapStateDescriptor<>("terminology-state", String.class, TerminologyRelease.class);

BroadcastStream<TerminologyRelease> broadcastStream =
    cdcStream
        .map(json -> objectMapper.readValue(json, TerminologyRelease.class))
        .filter(release -> "ACTIVE".equals(release.getStatus()))
        .broadcast(descriptor);
```

### 5.2 Python Consumer Example

```python
from kafka import KafkaConsumer
import json

consumer = KafkaConsumer(
    'kb7.terminology.public.kb_releases',
    bootstrap_servers=['kafka:9092'],
    auto_offset_reset='earliest',
    value_deserializer=lambda x: json.loads(x.decode('utf-8'))
)

for message in consumer:
    release = message.value
    if release.get('status') == 'ACTIVE':
        print(f"New terminology release: {release['version_id']}")
        print(f"  SNOMED: {release['snomed_version']}")
        print(f"  RxNorm: {release['rxnorm_version']}")
        print(f"  LOINC: {release['loinc_version']}")
        print(f"  Triples: {release['triple_count']:,}")

        # Trigger cache refresh
        refresh_terminology_cache(release)
```

## Troubleshooting

### Common Issues

#### 1. "wal_level property must be 'logical'"

```bash
# Fix: Enable logical replication in PostgreSQL
docker exec <postgres> psql -U postgres -c "ALTER SYSTEM SET wal_level = 'logical';"
docker restart <postgres>
```

#### 2. Connector FAILED state

```bash
# Check connector logs
docker logs kafka-connect 2>&1 | grep -i error | tail -20

# Common causes:
# - PostgreSQL not reachable (check network)
# - Invalid credentials
# - Missing replication permissions
```

#### 3. No messages in Kafka topic

```bash
# Check replication slot
docker exec <postgres> psql -U postgres -d kb_terminology -c \
  "SELECT slot_name, plugin, active FROM pg_replication_slots;"

# Check publication
docker exec <postgres> psql -U postgres -d kb_terminology -c \
  "SELECT * FROM pg_publication_tables WHERE tablename = 'kb_releases';"
```

#### 4. Container networking issues

```bash
# Verify all containers are on same network
docker network inspect kb-network

# Connect container to network
docker network connect kb-network <container-name>
```

### Monitoring

#### Connector Metrics

```bash
# List all connectors
curl http://localhost:8083/connectors

# Get connector config
curl http://localhost:8083/connectors/kb7-terminology-releases-cdc/config | jq

# Restart connector
curl -X POST http://localhost:8083/connectors/kb7-terminology-releases-cdc/restart

# Pause/Resume
curl -X PUT http://localhost:8083/connectors/kb7-terminology-releases-cdc/pause
curl -X PUT http://localhost:8083/connectors/kb7-terminology-releases-cdc/resume
```

#### Kafka Topic Monitoring

```bash
# Message count
docker exec kafka kafka-run-class kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 \
  --topic kb7.terminology.public.kb_releases \
  --time -1

# Consumer lag
docker exec kafka kafka-consumer-groups \
  --bootstrap-server localhost:9092 \
  --describe \
  --group <consumer-group>
```

## File Locations

| File | Purpose |
|------|---------|
| `configs/kb7-terminology-releases-cdc.json` | Debezium connector configuration |
| `sql/kb7-releases-schema.sql` | PostgreSQL table schema |
| `scripts/deploy-all-cdc-connectors.sh` | Deployment automation script |
| `docs/KB7_CDC_SETUP_GUIDE.md` | This documentation |

## Related Documentation

- [KB-7 Knowledge Factory Pipeline](../../knowledge-base-services/kb-7-terminology/knowledge-factory/README.md)
- [Flink Processing Pipeline](../../flink-processing/README.md)
- [CDC Connectors Overview](../README.md)

---

*Last Updated: December 2024*
*Author: CardioFit Platform Team*
