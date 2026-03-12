# CardioFit Kafka Docker Setup Guide

## Overview

This guide provides step-by-step instructions to set up a complete Kafka infrastructure with all 68 CardioFit topics using Docker.

## Prerequisites

- Docker Desktop (or Docker Engine + Docker Compose)
- At least 8GB RAM available to Docker
- 10GB free disk space
- Ports 2181, 8080, 8081, 8083, 8088, 9000, 9092-9094 available

## Quick Start

### 1. Start the Infrastructure

```bash
cd backend/shared-infrastructure/kafka
./start-kafka.sh
```

This will:
- Start a 3-node Kafka cluster
- Create all 68 topics with proper configuration
- Start management UIs and supporting services

### 2. Verify Installation

```bash
./health-check.sh
```

### 3. Access Management UIs

- **Kafka UI**: http://localhost:8080 (Primary UI)
- **Kafdrop**: http://localhost:9000 (Alternative UI)
- **Schema Registry**: http://localhost:8081
- **KSQL DB**: http://localhost:8088

## Components

### Core Infrastructure

| Service | Container | Port | Purpose |
|---------|-----------|------|---------|
| Zookeeper | cardiofit-zookeeper | 2181 | Cluster coordination |
| Kafka Broker 1 | cardiofit-kafka1 | 9092 | Primary broker |
| Kafka Broker 2 | cardiofit-kafka2 | 9093 | Secondary broker |
| Kafka Broker 3 | cardiofit-kafka3 | 9094 | Third broker |

### Management & Monitoring

| Service | Container | Port | Purpose |
|---------|-----------|------|---------|
| Kafka UI | cardiofit-kafka-ui | 8080 | Topic management |
| Kafdrop | cardiofit-kafdrop | 9000 | Alternative UI |
| Schema Registry | cardiofit-schema-registry | 8081 | Schema management |
| Kafka Connect | cardiofit-kafka-connect | 8083 | Data integration |
| KSQL DB | cardiofit-ksqldb-server | 8088 | Stream processing |

## Topic Overview

All 68 topics are automatically created with these categories:

### Clinical Events (9 topics)
- `patient-events.v1`
- `medication-events.v1`
- `observation-events.v1`
- `safety-events.v1`
- `vital-signs-events.v1`
- `lab-result-events.v1`
- `encounter-events.v1`
- `diagnostic-events.v1`
- `procedure-events.v1`

### Runtime Processing (5 topics)
- `enriched-patient-events.v1`
- `clinical-patterns.v1`
- `pathway-adherence-events.v1`
- `semantic-mesh-updates.v1`
- `patient-context-snapshots.v1`

### Knowledge Base CDC (8 topics)
- `kb3.clinical_protocols.changes`
- `kb4.drug_calculations.changes`
- `kb4.dosing_rules.changes`
- `kb4.weight_adjustments.changes`
- `kb5.drug_interactions.changes`
- `kb6.validation_rules.changes`
- `kb7.terminology.changes`
- `semantic-mesh.changes`

### Plus 46 more topics across:
- Device Data (4 topics)
- Evidence Management (6 topics)
- Workflow & Orchestration (6 topics)
- SLA & Monitoring (6 topics)
- Cache & Optimization (4 topics)
- Dead Letter Queues (9 topics)
- Real-time Collaboration (5 topics)
- External Integration (6 topics)

## Management Commands

### Topic Management

```bash
# List all topics
./manage-topics.sh list

# Describe a specific topic
./manage-topics.sh describe patient-events.v1

# Show topic statistics
./manage-topics.sh stats

# Validate against reference
./manage-topics.sh validate

# Send test message
./manage-topics.sh produce patient-events.v1

# Read messages
./manage-topics.sh consume patient-events.v1
```

### Infrastructure Management

```bash
# Check cluster health
./health-check.sh

# View service logs
docker-compose logs -f kafka1

# Stop infrastructure
./stop-kafka.sh

# Restart specific service
docker-compose restart kafka-ui
```

## Configuration

### Environment Variables (.env)

Key settings can be modified in the `.env` file:

```bash
# Kafka settings
KAFKA_DEFAULT_REPLICATION_FACTOR=3
KAFKA_MIN_INSYNC_REPLICAS=2
KAFKA_LOG_RETENTION_HOURS=168

# Performance settings
KAFKA_NUM_NETWORK_THREADS=8
KAFKA_NUM_IO_THREADS=8

# Ports
KAFKA_UI_PORT=8080
KAFDROP_PORT=9000
```

### Topic Configuration

Topics are created with production-ready settings:

| Setting | Value | Purpose |
|---------|-------|---------|
| Replication Factor | 3 | Fault tolerance |
| Min In-Sync Replicas | 2-3 | Durability |
| Compression | snappy/lz4/gzip | Efficiency |
| Retention | 1 day - 365 days | Compliance |

## Production Usage

### Client Configuration

```python
# Python producer example
producer = Producer({
    'bootstrap.servers': 'localhost:9092,localhost:9093,localhost:9094',
    'acks': 'all',
    'compression.type': 'snappy',
    'batch.size': 16384,
    'linger.ms': 10,
    'retries': 3
})
```

```java
// Java consumer example
Properties props = new Properties();
props.put("bootstrap.servers", "localhost:9092,localhost:9093,localhost:9094");
props.put("group.id", "cardiofit-service");
props.put("enable.auto.commit", "true");
props.put("auto.commit.interval.ms", "1000");
props.put("key.deserializer", "org.apache.kafka.common.serialization.StringDeserializer");
props.put("value.deserializer", "org.apache.kafka.common.serialization.StringDeserializer");
```

### Monitoring

1. **Kafka UI Dashboard** (http://localhost:8080)
   - Topic overview
   - Consumer group lag
   - Broker metrics
   - Schema registry

2. **JMX Metrics** (ports 9101-9103)
   - Broker performance
   - Topic metrics
   - JVM statistics

3. **Health Check Script**
   ```bash
   ./health-check.sh
   ```

## Troubleshooting

### Common Issues

1. **Port Conflicts**
   ```bash
   # Check port usage
   netstat -tulpn | grep :9092

   # Modify ports in docker-compose.yml if needed
   ```

2. **Memory Issues**
   ```bash
   # Increase Docker memory limit to 8GB+
   # Modify KAFKA_HEAP_OPTS in .env
   ```

3. **Topics Not Created**
   ```bash
   # Check topic initializer logs
   docker logs cardiofit-topic-initializer

   # Manually recreate topics
   docker exec cardiofit-kafka1 bash /usr/bin/create-topics.sh
   ```

4. **Connection Issues**
   ```bash
   # Test connectivity
   docker exec cardiofit-kafka1 kafka-broker-api-versions \
     --bootstrap-server kafka1:29092
   ```

### Log Analysis

```bash
# View all container logs
docker-compose logs

# View specific service logs
docker-compose logs -f kafka1
docker-compose logs -f zookeeper

# Check topic initializer
docker logs cardiofit-topic-initializer
```

### Recovery Procedures

1. **Restart Single Broker**
   ```bash
   docker-compose restart kafka1
   ```

2. **Full Cluster Restart**
   ```bash
   ./stop-kafka.sh
   ./start-kafka.sh
   ```

3. **Reset All Topics** (⚠️ Data Loss)
   ```bash
   ./manage-topics.sh reset
   ```

## Performance Optimization

### Resource Allocation

For production-like testing:

```yaml
# In docker-compose.yml
services:
  kafka1:
    deploy:
      resources:
        limits:
          memory: 2G
        reservations:
          memory: 1G
```

### JVM Tuning

```bash
# In .env
KAFKA_HEAP_OPTS=-Xmx1G -Xms1G
KAFKA_JVM_PERFORMANCE_OPTS=-server -XX:+UseG1GC -XX:MaxGCPauseMillis=20
```

### Network Optimization

```bash
# Increase network buffers
KAFKA_SOCKET_SEND_BUFFER_BYTES=102400
KAFKA_SOCKET_RECEIVE_BUFFER_BYTES=102400
```

## Security Considerations

### Development Setup (Current)
- No authentication (PLAINTEXT protocol)
- Internal Docker network isolation
- Local development only

### Production Recommendations
- Enable SASL/SSL authentication
- Use TLS encryption in transit
- Implement ACLs for topic access
- Network segmentation
- Regular security updates

## Backup & Recovery

### Volume Management

```bash
# List data volumes
docker volume ls | grep cardiofit

# Backup volume
docker run --rm -v cardiofit-kafka1-data:/data -v $(pwd):/backup alpine \
  tar czf /backup/kafka1-backup.tar.gz /data

# Restore volume
docker run --rm -v cardiofit-kafka1-data:/data -v $(pwd):/backup alpine \
  tar xzf /backup/kafka1-backup.tar.gz
```

### Configuration Backup

```bash
# Backup configurations
cp docker-compose.yml docker-compose.yml.bak
cp .env .env.bak
cp -r scripts scripts.bak
```

## Integration Examples

### Microservice Connection

```python
# FastAPI service integration
from confluent_kafka import Producer, Consumer
import asyncio

class KafkaService:
    def __init__(self):
        self.producer = Producer({
            'bootstrap.servers': 'localhost:9092,localhost:9093,localhost:9094',
            'acks': 'all'
        })

    async def publish_patient_event(self, patient_id: str, event_data: dict):
        self.producer.produce(
            'patient-events.v1',
            key=patient_id,
            value=json.dumps(event_data)
        )
        self.producer.flush()
```

### Flink Processing

```java
// Flink job configuration
KafkaSource<String> source = KafkaSource.<String>builder()
    .setBootstrapServers("kafka1:29092,kafka2:29093,kafka3:29094")
    .setTopics("patient-events.v1", "medication-events.v1")
    .setGroupId("flink-processor")
    .setStartingOffsets(OffsetsInitializer.earliest())
    .build();
```

## Next Steps

1. **Integration**: Connect your microservices using the client examples
2. **Monitoring**: Set up production monitoring with Prometheus/Grafana
3. **Testing**: Use the topic management tools to test event flows
4. **Production**: Migrate to a production Kafka cluster with security

## Support

- Documentation: `KAFKA_TOPICS_REFERENCE.md`
- Health Check: `./health-check.sh`
- Topic Management: `./manage-topics.sh --help`
- Logs: `docker-compose logs [service]`

---

**Version**: 1.0.0
**Last Updated**: January 26, 2025
**Maintained By**: CardioFit Platform Team