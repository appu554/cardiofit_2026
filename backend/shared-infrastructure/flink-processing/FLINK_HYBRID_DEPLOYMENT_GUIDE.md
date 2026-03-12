# Flink EHR Intelligence Engine - Hybrid Architecture Deployment Guide

This guide provides complete instructions for deploying the updated Flink EHR Intelligence Engine with the hybrid Kafka topic architecture and TransactionalMultiSinkRouter.

## Architecture Overview

The complete pipeline processes clinical events through multiple stages and routes them to specialized Kafka topics:

```
┌─ Patient Events Source ─┐    ┌─ Event Processing Pipeline ─┐    ┌─ Hybrid Topic Routing ─┐    ┌─ Data Stores ─┐
│                         │    │                              │    │                        │    │              │
│ Kafka Source Topics:    │ ─▷ │ 1. Event Filter             │ ─▷ │ TransactionalMulti     │ ─▷ │ Google FHIR  │
│ • patient-events        │    │ 2. Event Enrichment         │    │ SinkRouter             │    │ ClickHouse   │
│ • medication-events     │    │ 3. Pattern Detection        │    │                        │    │ Redis Cache  │
│ • safety-events         │    │ 4. Transform to Hybrid      │    │ ┌─ prod.ehr.events.   │    │ Neo4j Graph  │
│ • vital-signs-events    │    │    Format                    │    │ │  enriched           │    │ Audit Store  │
│ • lab-result-events     │    │                              │    │ ├─ prod.ehr.fhir.     │    │              │
│                         │    │                              │    │ │  upsert             │    │              │
└─────────────────────────┘    └──────────────────────────────┘    │ ├─ prod.ehr.alerts.   │    └──────────────┘
                                                                    │ │  critical           │
                                                                    │ ├─ prod.ehr.analytics.│
                                                                    │ │  events             │
                                                                    │ ├─ prod.ehr.graph.    │
                                                                    │ │  mutations          │
                                                                    │ └─ prod.ehr.audit.    │
                                                                    │    logs               │
                                                                    └───────────────────────┘
```

## Prerequisites

### 1. Infrastructure Requirements

**Kafka Cluster**:
- 3+ brokers for high availability
- Replication factor ≥ 2
- Min in-sync replicas = 2
- Hybrid topics created (see Kafka Topics section)

**Flink Cluster**:
- JobManager: 4GB+ heap, 2+ CPU cores
- TaskManager: 8GB+ heap per slot, 4+ CPU cores
- Minimum 4 task slots total
- Checkpointing storage (S3, HDFS, or persistent volume)

**Dependencies**:
- Java 11+ (recommended: Java 17)
- Kafka 2.8+
- Flink 1.17+
- Avro schema registry (optional but recommended)

### 2. Create Hybrid Kafka Topics

```bash
cd /backend/shared-infrastructure/kafka
bash create-hybrid-architecture-topics.sh
```

Verify topics are created:
```bash
kafka-topics --bootstrap-server localhost:9092 --list | grep "prod.ehr"
```

Expected output:
```
prod.ehr.analytics.events
prod.ehr.audit.logs
prod.ehr.events.enriched
prod.ehr.fhir.upsert
prod.ehr.graph.mutations
prod.ehr.alerts.critical
prod.ehr.semantic.mesh
```

### 3. Deploy Kafka Connect Connectors

```bash
cd /backend/shared-infrastructure/flink-processing/kafka-connect
bash deploy-hybrid-connectors.sh
```

Verify connectors are running:
```bash
curl http://localhost:8083/connectors | jq '.[]' | while read connector; do
  echo "Checking: $connector"
  curl -s "http://localhost:8083/connectors/$connector/status" | jq '.connector.state'
done
```

## Flink Job Compilation and Packaging

### 1. Build the Flink Job JAR

```bash
cd /backend/shared-infrastructure/flink-processing

# Clean and compile
mvn clean compile

# Run tests to verify implementation
mvn test

# Package into executable JAR
mvn package -DskipTests

# Verify JAR creation
ls -la target/cardiofit-flink-processing-*.jar
```

### 2. Copy Dependencies

The job requires these key dependencies (included in JAR):
- Kafka connector for Flink
- Avro serialization libraries
- Jackson JSON processing
- Clinical event model classes
- TransactionalMultiSinkRouter and supporting classes

### 3. Validate JAR Contents

```bash
# Check if key classes are included
jar tf target/cardiofit-flink-processing-*.jar | grep -E "(TransactionalMultiSinkRouter|PatientEventEnrichmentJob|RoutedEventToEnrichedEventMapper)"
```

Expected output:
```
com/cardiofit/stream/jobs/PatientEventEnrichmentJob.class
com/cardiofit/flink/operators/TransactionalMultiSinkRouter.class
com/cardiofit/flink/mappers/RoutedEventToEnrichedEventMapper.class
```

## Flink Cluster Deployment

### 1. Start Flink Cluster

**Standalone Mode**:
```bash
# Start JobManager
$FLINK_HOME/bin/jobmanager.sh start

# Start TaskManager(s)
$FLINK_HOME/bin/taskmanager.sh start

# Verify cluster is running
curl http://localhost:8081/overview
```

**Docker Mode**:
```yaml
# docker-compose.yml
version: '3.8'
services:
  jobmanager:
    image: flink:1.17-java11
    ports:
      - "8081:8081"
    command: jobmanager
    environment:
      - FLINK_PROPERTIES=jobmanager.rpc.address: jobmanager

  taskmanager:
    image: flink:1.17-java11
    depends_on:
      - jobmanager
    command: taskmanager
    scale: 2
    environment:
      - FLINK_PROPERTIES=jobmanager.rpc.address: jobmanager
```

### 2. Configure Flink for Clinical Processing

**flink-conf.yaml additions**:
```yaml
# Checkpointing configuration
state.backend: filesystem
state.checkpoints.dir: file:///tmp/flink-checkpoints
state.savepoints.dir: file:///tmp/flink-savepoints
execution.checkpointing.interval: 30s
execution.checkpointing.min-pause: 10s
execution.checkpointing.timeout: 60s

# Performance tuning for clinical events
taskmanager.memory.process.size: 8g
taskmanager.memory.flink.size: 6g
taskmanager.numberOfTaskSlots: 4
parallelism.default: 4

# Latency optimization
execution.buffer-timeout: 100ms
pipeline.object-reuse: true

# Metrics and monitoring
metrics.reporter.prom.class: org.apache.flink.metrics.prometheus.PrometheusReporter
metrics.reporter.prom.port: 9249
```

## Job Deployment

### 1. Submit the EHR Intelligence Job

```bash
# Submit job to Flink cluster
$FLINK_HOME/bin/flink run \
  --class com.cardiofit.stream.jobs.PatientEventEnrichmentJob \
  --parallelism 4 \
  --detached \
  target/cardiofit-flink-processing-1.0.0.jar
```

### 2. Monitor Job Startup

```bash
# Check job status
$FLINK_HOME/bin/flink list

# View job in Web UI
open http://localhost:8081
```

### 3. Verify Pipeline Stages

The job should show these operators in the Flink Web UI:

1. **Patient Events Source** - Kafka source consumer
2. **Event Filter** - Clinical relevance filtering
3. **Event Enrichment** - Semantic mesh enrichment
4. **Pattern Detection** - Clinical pattern analysis
5. **Transform to Enriched Clinical Events** - Format transformation
6. **Transactional Multi-Sink Router** - Hybrid topic routing

### 4. Check Metrics

Key metrics to monitor:

```bash
# Job metrics via REST API
curl http://localhost:8081/jobs/{job-id}/metrics

# Kafka consumer lag
kafka-consumer-groups --bootstrap-server localhost:9092 \
  --describe --group patient-event-enrichment

# Topic production rates
kafka-run-class kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 \
  --topic prod.ehr.events.enriched
```

## Configuration Tuning

### 1. Performance Optimization

**For High Throughput (>5K events/sec)**:
```yaml
# flink-conf.yaml
taskmanager.numberOfTaskSlots: 8
parallelism.default: 8
execution.buffer-timeout: 50ms
taskmanager.memory.network.fraction: 0.2
```

**For Low Latency (<200ms)**:
```yaml
# flink-conf.yaml
execution.buffer-timeout: 10ms
pipeline.object-reuse: true
execution.checkpointing.interval: 10s
taskmanager.memory.managed.fraction: 0.3
```

### 2. Memory Configuration

**For Large State (patient context)**:
```yaml
# flink-conf.yaml
taskmanager.memory.process.size: 16g
taskmanager.memory.flink.size: 12g
taskmanager.memory.managed.size: 4g
state.backend.rocksdb.memory.managed: true
```

### 3. Fault Tolerance

**Production Resilience**:
```yaml
# flink-conf.yaml
restart-strategy: exponential-delay
restart-strategy.exponential-delay.initial-backoff: 1s
restart-strategy.exponential-delay.max-backoff: 60s
restart-strategy.exponential-delay.backoff-multiplier: 1.2
restart-strategy.exponential-delay.reset-backoff-threshold: 10min
```

## Testing and Validation

### 1. End-to-End Flow Test

```bash
# 1. Produce test event to input topic
kafka-console-producer --bootstrap-server localhost:9092 \
  --topic patient-events << EOF
{
  "eventId": "test-001",
  "patientId": "patient-12345",
  "eventType": "medication_order",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)",
  "isCritical": true,
  "payload": {
    "medication": "warfarin",
    "dosage": "5mg",
    "frequency": "daily"
  }
}
EOF

# 2. Verify event appears in hybrid topics
kafka-console-consumer --bootstrap-server localhost:9092 \
  --topic prod.ehr.events.enriched --from-beginning --max-messages 1

kafka-console-consumer --bootstrap-server localhost:9092 \
  --topic prod.ehr.alerts.critical --from-beginning --max-messages 1
```

### 2. Performance Test

```bash
# Generate load test events
for i in {1..1000}; do
  kafka-console-producer --bootstrap-server localhost:9092 \
    --topic patient-events << EOF
{
  "eventId": "perf-test-$i",
  "patientId": "patient-$(( $i % 100 ))",
  "eventType": "vital_signs",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)",
  "isCritical": false,
  "payload": {"heartRate": $(( 60 + $i % 40 ))}
}
EOF
done

# Monitor processing latency
curl -s http://localhost:8081/jobs/{job-id}/metrics | grep latency
```

### 3. Failure Recovery Test

```bash
# Kill TaskManager to test recovery
pkill -f taskmanager

# Job should restart automatically
# Check job status after 30 seconds
$FLINK_HOME/bin/flink list

# Restart TaskManager
$FLINK_HOME/bin/taskmanager.sh start
```

## Monitoring and Alerting

### 1. Key Metrics to Monitor

**Job Health**:
- Job status (RUNNING/FAILED/RESTARTING)
- Checkpoint success rate (>95%)
- Checkpoint duration (<30s)
- Recovery time (<2 minutes)

**Performance**:
- End-to-end latency (<500ms for critical events)
- Throughput (events/second processed)
- Backpressure (should be minimal)
- Memory usage (<80% of allocated)

**Business Logic**:
- Critical events routed to alerts topic
- All events routed to central topic
- FHIR events routed to upsert topic
- Zero data loss during processing

### 2. Alerting Thresholds

```yaml
alerts:
  job_down:
    condition: "job_status != 'RUNNING'"
    severity: "critical"

  high_latency:
    condition: "avg_latency > 1000ms"
    severity: "warning"

  checkpoint_failures:
    condition: "checkpoint_failure_rate > 5%"
    severity: "warning"

  low_throughput:
    condition: "events_per_second < 1000"
    severity: "info"
```

## Troubleshooting

### 1. Common Issues

**Job Won't Start**:
- Check classpath includes all dependencies
- Verify Kafka cluster is accessible
- Check Flink logs: `tail -f $FLINK_HOME/log/flink-*-jobmanager-*.log`

**High Latency**:
- Reduce checkpoint interval
- Increase parallelism
- Optimize serialization
- Check Kafka consumer lag

**Memory Issues**:
- Increase TaskManager heap size
- Enable object reuse
- Optimize state size
- Use RocksDB backend for large state

**Data Loss**:
- Verify exactly-once semantics enabled
- Check checkpoint success rate
- Monitor Kafka replication
- Validate transactional producer configuration

### 2. Log Analysis

```bash
# Job-specific logs
tail -f $FLINK_HOME/log/flink-*-taskexecutor-*.log | grep "PatientEventEnrichmentJob"

# TransactionalMultiSinkRouter logs
tail -f $FLINK_HOME/log/flink-*-taskexecutor-*.log | grep "TransactionalMultiSinkRouter"

# Checkpoint logs
tail -f $FLINK_HOME/log/flink-*-jobmanager-*.log | grep -i checkpoint
```

### 3. Recovery Procedures

**Job Restart**:
```bash
# Cancel current job
$FLINK_HOME/bin/flink cancel {job-id}

# Restart from latest checkpoint
$FLINK_HOME/bin/flink run \
  --class com.cardiofit.stream.jobs.PatientEventEnrichmentJob \
  --fromSavepoint /path/to/savepoint \
  target/cardiofit-flink-processing-1.0.0.jar
```

**Rollback to Previous Version**:
```bash
# Create savepoint first
$FLINK_HOME/bin/flink savepoint {job-id} /path/to/rollback-savepoint

# Cancel current job
$FLINK_HOME/bin/flink cancel {job-id}

# Deploy previous JAR version
$FLINK_HOME/bin/flink run \
  --fromSavepoint /path/to/rollback-savepoint \
  target/cardiofit-flink-processing-previous.jar
```

## Production Checklist

### Pre-Deployment
- [ ] Hybrid Kafka topics created with correct configurations
- [ ] Kafka Connect connectors deployed and tested
- [ ] Flink cluster configured for production workload
- [ ] JAR file built and validated
- [ ] Dependencies verified in classpath
- [ ] Monitoring and alerting configured

### Post-Deployment
- [ ] Job submitted successfully and shows RUNNING status
- [ ] All pipeline stages visible in Flink Web UI
- [ ] Metrics collection working (Prometheus/Grafana)
- [ ] End-to-end test passes (input → hybrid topics → data stores)
- [ ] Performance meets SLA requirements (<500ms latency)
- [ ] Checkpoint success rate >95%
- [ ] Error rates <0.1%

### Ongoing Operations
- [ ] Monitor Kafka consumer lag daily
- [ ] Review checkpoint duration trends
- [ ] Validate data quality in downstream systems
- [ ] Perform disaster recovery tests monthly
- [ ] Update clinical patterns and enrichment rules as needed

The EHR Intelligence Engine with hybrid Kafka architecture provides significant performance improvements while maintaining exactly-once processing guarantees for critical clinical data.