# Kafka Connect - Clinical Events Distribution

This directory contains Kafka Connect configurations for distributing clinical events from the unified Flink topic to multiple data stores.

## Architecture

```
Flink Egress Module → clinical-events-unified.v1 → Kafka Connect → Data Stores
                                ↓
        ┌───────────────────────┼───────────────────────┐
        ↓                       ↓                       ↓
    Neo4j Sink          ClickHouse Sink         Elasticsearch Sink
(Knowledge Graph)       (Analytics)             (Search)
        ↓                       ↓                       ↓
    Redis Sink              Google FHIR Store
    (Caching)               (Clinical Persistence)
```

## Single Message Transform (SMT) Strategy

Each connector uses SMTs to:
1. **Filter** events based on destination routing
2. **Transform** data format for target store
3. **Route** events intelligently based on content
4. **Handle errors** with retry and logging

## Connectors

### 1. Neo4j Clinical Knowledge Graph
- **File**: `connectors/neo4j-sink.json`
- **Purpose**: Store patient relationships and clinical pathways
- **Key SMTs**: Destination filtering, Cypher query generation
- **Target**: Clinical knowledge graph for relationships

### 2. ClickHouse Analytics
- **File**: `connectors/clickhouse-sink.json`
- **Purpose**: Time-series analytics and OLAP queries
- **Key SMTs**: Field extraction, timestamp conversion, batch optimization
- **Target**: High-performance analytics database

### 3. Elasticsearch Search
- **File**: `connectors/elasticsearch-sink.json`
- **Purpose**: Full-text search and clinical document indexing
- **Key SMTs**: Index routing, payload flattening, timestamp insertion
- **Target**: Search and analytics with time-based indices

### 4. Redis Real-time Cache
- **File**: `connectors/redis-sink.json`
- **Purpose**: Hot data caching and critical alerts
- **Key SMTs**: Key generation, TTL setting, priority filtering
- **Target**: Real-time caching with expiration

### 5. Google FHIR Store
- **File**: `connectors/fhir-store-sink.json`
- **Purpose**: FHIR R4 compliant clinical persistence
- **Key SMTs**: FHIR resource mapping, patient ID extraction
- **Target**: Long-term clinical data storage

## Deployment

### Prerequisites
1. Kafka Connect cluster running (port 8083)
2. All target data stores available and configured
3. Required connector plugins installed:
   - Neo4j Kafka Connect plugin
   - ClickHouse Kafka Connect plugin
   - Confluent Elasticsearch connector
   - Redis Kafka Connect plugin
   - Google Healthcare connector

### Deploy All Connectors
```bash
./deploy-connectors.sh
```

### Deploy Specific Connector
```bash
./deploy-connectors.sh --connector neo4j-sink
```

### List Connector Status
```bash
./deploy-connectors.sh --list
```

## Data Flow Routing

Events are routed based on destination metadata in the `RoutedEvent`:

```json
{
  "id": "event-uuid",
  "patientId": "patient-123",
  "sourceEventType": "SEMANTIC_EVENT",
  "routingTime": 1640995200000,
  "priority": "HIGH",
  "destinations": ["neo4j", "fhir_store", "analytics"],
  "originalPayload": { ... }
}
```

### Routing Rules
- **Neo4j**: Events with destinations: `neo4j`, `graph`, `knowledge`
- **ClickHouse**: Events with destinations: `clickhouse`, `analytics`, `time-series`
- **Elasticsearch**: Events with destinations: `elasticsearch`, `search`, `analytics`
- **Redis**: Events with destinations: `redis`, `cache`, `real-time`
- **FHIR Store**: Events with destinations: `fhir`, `fhir_store`, `clinical_persistence`

## Error Handling

All connectors include:
- **Retry Logic**: Configurable retry attempts with exponential backoff
- **Error Tolerance**: Continue processing on individual failures
- **Error Logging**: Detailed error logging with message content
- **Dead Letter Queues**: Failed messages routed to DLQ topics

## Monitoring

### Health Checks
```bash
# Check all connectors
curl http://localhost:8083/connectors

# Check specific connector status
curl http://localhost:8083/connectors/neo4j-clinical-events-sink/status

# Check connector configuration
curl http://localhost:8083/connectors/neo4j-clinical-events-sink/config
```

### Key Metrics
- **Throughput**: Records processed per second
- **Latency**: End-to-end processing time
- **Error Rate**: Failed vs successful operations
- **Connector State**: Running, Failed, Paused

## Configuration Customization

### Environment Variables
Set these before deployment:
```bash
export KAFKA_CONNECT_URL="http://localhost:8083"
export NEO4J_PASSWORD="your-password"
export CLICKHOUSE_PASSWORD="your-password"
export ELASTICSEARCH_PASSWORD="your-password"
export REDIS_PASSWORD="your-password"
```

### Custom SMTs
Add additional transforms as needed:
```json
"transforms": "customFilter,customTransform",
"transforms.customFilter.type": "your.custom.Transform",
"transforms.customFilter.config": "value"
```

## Troubleshooting

### Common Issues

1. **Connector Failed to Start**
   - Check plugin availability
   - Verify target store connectivity
   - Review configuration syntax

2. **No Data Flowing**
   - Verify topic exists: `clinical-events-unified.v1`
   - Check Flink is producing to topic
   - Review SMT filter conditions

3. **Performance Issues**
   - Increase batch sizes for high throughput
   - Adjust flush timeouts
   - Scale connector tasks

### Debug Commands
```bash
# Check topic has data
kafka-console-consumer --bootstrap-server localhost:9092 \
  --topic clinical-events-unified.v1 --from-beginning

# Check connector logs
docker logs kafka-connect-1 | grep neo4j-clinical-events-sink

# Restart failed connector
curl -X POST http://localhost:8083/connectors/neo4j-clinical-events-sink/restart
```

## Benefits of Single-Topic Architecture

1. **Data Durability**: Events persisted in Kafka before distribution
2. **Reliability**: Connector failures don't lose data
3. **Scalability**: Independent scaling of each data store connector
4. **Flexibility**: Easy to add new data stores without Flink changes
5. **Monitoring**: Centralized monitoring of data distribution
6. **Recovery**: Replay capability from Kafka for disaster recovery

This architecture ensures reliable, scalable distribution of clinical events while maintaining data durability and system resilience.