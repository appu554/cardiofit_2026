# Flink EHR Intelligence Engine - Data Store Integration

Complete integration guide for connecting Flink processing with all shared data stores.

## Overview

The Flink EHR Intelligence Engine uses a Kafka-first architecture for reliable data distribution to 6 major data stores:

```
Clinical Events → Flink Processing → Unified Topic → Kafka Connect → Data Stores
                       ↓                  ↓
               clinical-events-unified.v1  ↓
                                          ↓
        ┌─────────────────────────────────┼─────────────────────────────────┐
        ↓                                 ↓                                 ↓
    🗄️ Neo4j                         📈 ClickHouse                   🔍 Elasticsearch
    (Knowledge Graph)                 (Time-Series)                  (Search & Analytics)
        ↓                                 ↓                                 ↓
    🚀 Redis Master                   🚀 Redis Replica              🏥 Google FHIR Store
    (Real-time Cache)                 (Read Scaling)                (Clinical Persistence)
```

**Architecture Benefits:**
- **Data Durability**: All events persisted in Kafka before distribution
- **Reliability**: Connector failures don't cause data loss
- **Scalability**: Independent scaling of each data store
- **Recovery**: Full replay capability from Kafka topic

## Quick Start

### 1. Start Shared Data Stores

```bash
# Start all shared infrastructure
cd backend/shared-infrastructure
./start-datastores.sh
```

### 2. Start Flink with Data Store Integration

```bash
# Start Flink with automatic connection testing
cd backend/shared-infrastructure/flink-processing
./start-flink-with-datastores.sh
```

### 3. Deploy Kafka Connect Connectors

```bash
# Deploy all data store connectors
cd backend/shared-infrastructure/flink-processing/kafka-connect
./deploy-connectors.sh

# Check connector status
./deploy-connectors.sh --list
```

### 4. Verify Integration

```bash
# Test all data store connections
java -cp target/flink-ehr-intelligence-1.0.0.jar \
  com.cardiofit.flink.utils.DataStoreConnectionTest

# Verify unified topic exists
kafka-topics --bootstrap-server localhost:9092 --list | grep clinical-events-unified
```

## Data Store Configurations

### 🗄️ Neo4j - Clinical Knowledge Graph

**Purpose**: Patient relationships, clinical pathways, treatment patterns

```properties
NEO4J_URI=bolt://localhost:7687
NEO4J_USERNAME=neo4j
NEO4J_PASSWORD=CardioFit2024!
NEO4J_DATABASE=cardiofit
```

**Data Types Stored**:
- Patient contexts and relationships
- Clinical concepts and semantic events
- Pattern events and treatment pathways
- ML predictions for decision support

**Example Usage**:
```java
// Events automatically routed to Neo4j
transformedEvents
    .filter(event -> event.hasDestination("neo4j"))
    .addSink(new Neo4jGraphSink())
    .name("Neo4j Graph Sink");
```

### 📈 ClickHouse - Time-Series Analytics

**Purpose**: OLAP analytics, aggregated metrics, performance monitoring

```properties
CLICKHOUSE_HOST=localhost
CLICKHOUSE_PORT=8123
CLICKHOUSE_USERNAME=cardiofit_user
CLICKHOUSE_PASSWORD=ClickHouse2024!
CLICKHOUSE_DATABASE=cardiofit_analytics
```

**Data Types Stored**:
- Clinical events with timestamps
- Pattern detection metrics
- ML prediction results with confidence scores
- Aggregated patient statistics

**Tables Created**:
- `clinical_events` - All clinical events with patient context
- `pattern_metrics` - Detected clinical patterns and trends
- `ml_predictions` - Machine learning inference results
- `hourly_metrics` - Aggregated metrics (materialized view)

### 🔍 Elasticsearch - Search & Analytics

**Purpose**: Full-text search, clinical document indexing

```properties
ELASTICSEARCH_HOST=localhost
ELASTICSEARCH_PORT=9200
ELASTICSEARCH_USERNAME=elastic
ELASTICSEARCH_PASSWORD=ElasticCardioFit2024!
ELASTICSEARCH_INDEX_PREFIX=cardiofit-clinical
```

**Data Types Stored**:
- Enriched clinical events for search
- Patient summaries and clinical notes
- Semantic annotations and concepts
- Analytics data for dashboards

**Index Strategy**:
- Time-based indices: `cardiofit-clinical-2024.01.15`
- Bulk indexing with 100 documents per batch
- TTL-based data retention

### 🚀 Redis - Real-time Caching

**Purpose**: Hot data caching, session management, critical alerts

```properties
# Redis Master (Write operations)
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=RedisCardioFit2024!

# Redis Replica (Read operations)
REDIS_REPLICA_HOST=localhost
REDIS_REPLICA_PORT=6380
```

**Data Types Cached**:
- Patient context (TTL: 1 hour)
- Critical alerts (TTL: 24 hours)
- ML predictions (TTL: 2-24 hours based on type)
- Pattern cache (TTL: 2 hours)
- Session data (TTL: 30 minutes)

**Key Patterns**:
```
cardiofit:patient:context:{patientId}
cardiofit:alerts:critical:{patientId}
cardiofit:predictions:{patientId}:{predictionType}
cardiofit:patterns:{patientId}:{patternType}
cardiofit:session:{sessionId}
```

### 🏥 Google FHIR Store - Clinical Persistence

**Purpose**: Long-term FHIR R4 compliant clinical data storage

```properties
GOOGLE_CLOUD_PROJECT_ID=cardiofit-905a8
GOOGLE_CLOUD_LOCATION=us-central1
GOOGLE_CLOUD_DATASET_ID=clinical_synthesis_hub
GOOGLE_CLOUD_FHIR_STORE_ID=fhir_store
```

**Data Types Stored**:
- FHIR R4 resources (Patient, Observation, etc.)
- Clinical events with FHIR transformations
- Semantic events as FHIR DiagnosticReport
- Pattern events as FHIR ClinicalImpression

## Routing Configuration

### Kafka-First Architecture

All events flow through a single unified topic (`clinical-events-unified.v1`) and are distributed via Kafka Connect with Single Message Transforms (SMTs).

### Event Routing Logic

Events are automatically routed to appropriate data stores via Kafka Connect SMT filters based on:

1. **Event Type**: Semantic events → Neo4j + FHIR Store
2. **Priority**: Critical events → Redis + all stores
3. **Content**: Analytics data → ClickHouse + Elasticsearch
4. **Destinations**: Explicit routing via `event.getDestinations()`

### Kafka Connect Routing Rules

Each connector filters events using SMT predicates:

```json
// Neo4j Connector
"transforms.routeToNeo4j.predicate.destinations": "neo4j,graph,knowledge"

// ClickHouse Connector
"transforms.routeToClickHouse.predicate.destinations": "clickhouse,analytics,time-series"

// Elasticsearch Connector
"transforms.routeToElasticsearch.predicate.destinations": "elasticsearch,search,analytics"

// Redis Connector
"transforms.routeToRedis.predicate.destinations": "redis,cache,real-time"

// FHIR Store Connector
"transforms.routeToFHIR.predicate.destinations": "fhir,fhir_store,clinical_persistence"
```

### Default Routing Rules

```java
// Semantic Events
destinations.add("neo4j");         // Knowledge graph relationships
destinations.add("fhir_store");    // Clinical persistence
destinations.add("analytics");     // ClickHouse analytics

// Pattern Events
destinations.add("clickhouse");    // Time-series analytics
destinations.add("neo4j");         // Pattern relationships
destinations.add("redis");         // Pattern cache

// ML Predictions
destinations.add("redis");         // Real-time predictions
destinations.add("clickhouse");    // ML metrics
destinations.add("fhir_store");    // Clinical predictions

// Critical Events (all priorities)
destinations.add("redis");         // Immediate caching
destinations.add("neo4j");         // Critical relationships
destinations.add("fhir_store");    // Clinical persistence
destinations.add("elasticsearch"); // Search/alerting
```

### Unified Topic Architecture

```java
// All events route to single topic
transformedEvents
    .sinkTo(createClinicalEventsSink())  // → clinical-events-unified.v1
    .name("Clinical Events Unified Sink");

// Kafka Connect handles distribution with SMTs
// No direct database connections from Flink
```

## Performance Configuration

### Batch Sizes

| Data Store | Default Batch Size | Flush Interval |
|------------|-------------------|----------------|
| ClickHouse | 1000 events | 5 seconds |
| Elasticsearch | 100 events | 5 seconds |
| Neo4j | 1 event (transactional) | Immediate |
| Redis | Pipeline (100 ops) | Immediate |
| Google FHIR Store | 1 event | Immediate |

### Memory Settings

```properties
# Neo4j
NEO4J_HEAP_SIZE=2048  # MB

# ClickHouse
CLICKHOUSE_MAX_MEMORY_USAGE=4096  # MB

# Elasticsearch
ELASTICSEARCH_HEAP_SIZE=2048  # MB

# Redis
REDIS_MAX_MEMORY=2048  # MB
```

### Connection Pools

```properties
# Neo4j
NEO4J_MAX_CONNECTION_POOL_SIZE=100
NEO4J_CONNECTION_ACQUISITION_TIMEOUT=60000

# Redis
REDIS_MAX_CONNECTIONS=128
REDIS_MIN_IDLE_CONNECTIONS=16

# ClickHouse
CLICKHOUSE_MAX_CONNECTIONS=100
CLICKHOUSE_CONNECTION_TIMEOUT=30000
```

## Monitoring & Health Checks

### Health Check Endpoints

| Service | Health Check Command |
|---------|---------------------|
| Neo4j | `cypher-shell "RETURN 1"` |
| ClickHouse | `curl http://localhost:8123/ping` |
| Elasticsearch | `curl http://localhost:9200/_cluster/health` |
| Redis | `redis-cli ping` |

### Metrics Collection

All sinks expose metrics via Prometheus:

```yaml
# Example metrics
flink_sink_neo4j_records_out_total
flink_sink_clickhouse_records_out_total
flink_sink_elasticsearch_records_out_total
flink_sink_redis_records_out_total
flink_sink_fhir_store_records_out_total

flink_sink_latency_histogram
flink_sink_errors_total
flink_sink_backpressure_time
```

### Grafana Dashboards

Pre-configured dashboards available:

1. **Flink EHR Intelligence Overview**
   - Event processing rates
   - Sink performance
   - Error rates

2. **Data Store Performance**
   - Individual store metrics
   - Connection pool status
   - Query performance

3. **Clinical Data Flow**
   - Patient event patterns
   - Clinical insights generated
   - FHIR resource creation rates

## Error Handling

### Retry Policies

```properties
# Sink-specific retry settings
SINK_ERROR_RETRY_ATTEMPTS=3
SINK_ERROR_RETRY_DELAY_MS=1000

# FHIR Store specific
FHIR_STORE_RETRY_ATTEMPTS=3
FHIR_STORE_RETRY_DELAY_MS=1000
FHIR_STORE_TIMEOUT_MS=30000
```

### Dead Letter Queues

Failed events are routed to Kafka DLQ topics:

- `dlq-neo4j-errors`
- `dlq-clickhouse-errors`
- `dlq-elasticsearch-errors`
- `dlq-redis-errors`
- `dlq-fhir-store-errors`

### Circuit Breaker

Automatic circuit breaker activation:

```properties
SINK_CIRCUIT_BREAKER_THRESHOLD=10  # failures
CIRCUIT_BREAKER_TIMEOUT=60000      # ms
```

## Data Retention Policies

### Automatic Cleanup

| Data Store | Retention Policy |
|------------|------------------|
| ClickHouse | 90 days (clinical), 180 days (patterns), 365 days (predictions) |
| Elasticsearch | 90 days via index lifecycle management |
| Redis | TTL-based: 1-24 hours depending on data type |
| Neo4j | Manual cleanup of old relationships (30 days) |
| Google FHIR Store | No automatic deletion (HIPAA compliance) |

### Backup Strategies

```bash
# Automated backup configuration
ENABLE_AUTOMATIC_BACKUPS=true
BACKUP_SCHEDULE_CRON="0 2 * * *"  # Daily at 2 AM
BACKUP_RETENTION_DAYS=30
```

## Security Configuration

### Authentication

All data stores use strong authentication:

- **Neo4j**: Username/password with role-based access
- **ClickHouse**: Database user with limited permissions
- **Elasticsearch**: Built-in security with user roles
- **Redis**: AUTH password protection
- **Google FHIR Store**: Service account with IAM roles

### Network Security

```yaml
# All services run on isolated Docker network
networks:
  cardiofit-network:
    external: true
    name: cardiofit-network
```

### Data Encryption

For production deployments:

```properties
# Enable TLS for all connections
TLS_ENABLED=true
ENCRYPTION_AT_REST=true
```

## Troubleshooting

### Common Issues

1. **Connection Failures**
   ```bash
   # Test individual connections
   java -cp target/flink-ehr-intelligence-1.0.0.jar \
     com.cardiofit.flink.utils.DataStoreConnectionTest neo4j
   ```

2. **Memory Issues**
   ```bash
   # Increase container memory limits
   docker-compose up --scale taskmanager-1=1 --scale taskmanager-2=1
   ```

3. **Network Issues**
   ```bash
   # Verify network connectivity
   docker network ls | grep cardiofit
   docker network inspect cardiofit-network
   ```

### Log Analysis

```bash
# View Flink sink logs
docker-compose logs -f jobmanager | grep -i sink

# View specific sink errors
docker-compose logs -f taskmanager-1 | grep -i "neo4j\|clickhouse\|elasticsearch"

# Monitor data store logs
cd ../
docker-compose -f docker-compose.datastores.yml logs -f neo4j
```

### Performance Tuning

```bash
# Monitor sink backpressure
curl http://localhost:8081/jobs/{job-id}/vertices/{vertex-id}/backpressure

# Check processing rates
curl http://localhost:8081/jobs/{job-id}/vertices/{vertex-id}/metrics
```

## Integration Examples

### Custom Event Processing

```java
// Custom event routing
public class CustomEventRouter extends ProcessFunction<SemanticEvent, RoutedEvent> {
    @Override
    public void processElement(SemanticEvent event, Context ctx, Collector<RoutedEvent> out) {
        RoutedEvent routed = new RoutedEvent();

        // Route based on clinical significance
        if (event.getClinicalSignificance() > 0.8) {
            routed.addDestination("neo4j");
            routed.addDestination("redis");
            routed.addDestination("fhir_store");
            routed.setPriority(RoutedEvent.Priority.HIGH);
        } else {
            routed.addDestination("clickhouse");
            routed.addDestination("elasticsearch");
            routed.setPriority(RoutedEvent.Priority.NORMAL);
        }

        out.collect(routed);
    }
}
```

### Analytics Queries

```sql
-- ClickHouse: Patient event trends
SELECT
    toDate(event_time) as date,
    patient_id,
    count() as event_count,
    avg(confidence_score) as avg_confidence
FROM clinical_events
WHERE event_time >= now() - INTERVAL 7 DAY
GROUP BY date, patient_id
ORDER BY date DESC, event_count DESC;

-- ClickHouse: Pattern detection summary
SELECT
    pattern_type,
    count() as pattern_count,
    avg(confidence) as avg_confidence,
    max(detection_time) as last_detected
FROM pattern_metrics
WHERE detection_time >= now() - INTERVAL 1 DAY
GROUP BY pattern_type;
```

```cypher
-- Neo4j: Patient clinical pathways
MATCH (p:Patient {patientId: $patientId})-[:HAS_EVENT]->(e:Event)
MATCH (e)-[:DETECTED_FROM]->(pattern:Pattern)
RETURN p, e, pattern
ORDER BY e.timestamp DESC
LIMIT 50;

-- Neo4j: Drug interaction networks
MATCH (p:Patient)-[:TAKES_MEDICATION]->(m1:Medication)
MATCH (p)-[:TAKES_MEDICATION]->(m2:Medication)
WHERE m1 <> m2
MATCH (m1)-[:INTERACTS_WITH]->(m2)
RETURN p.patientId, m1.name, m2.name, 'potential_interaction' as alert;
```

## Kafka Connect Architecture

### Single-Topic Distribution Strategy

The new architecture uses a single unified topic with Kafka Connect for reliable data distribution:

```yaml
Architecture Benefits:
  data_durability: "Events never lost due to connector failures"
  scalability: "Independent scaling of each data store connector"
  reliability: "Built-in retry logic and error handling"
  flexibility: "Easy to add new data stores without Flink changes"
  monitoring: "Centralized monitoring via Kafka Connect REST API"
  recovery: "Full replay capability from Kafka topic"
```

### Connector Management

```bash
# Deploy all connectors
cd kafka-connect/
./deploy-connectors.sh

# Deploy specific connector
./deploy-connectors.sh --connector neo4j-sink

# Check connector health
curl http://localhost:8083/connectors/neo4j-clinical-events-sink/status

# Restart failed connector
curl -X POST http://localhost:8083/connectors/neo4j-clinical-events-sink/restart
```

### Single Message Transforms (SMTs)

Each connector uses sophisticated SMTs for:

1. **Event Filtering**: Route based on destination metadata
2. **Data Transformation**: Convert to target store format
3. **Field Extraction**: Pull relevant fields for each store
4. **Timestamp Handling**: Proper time conversion and indexing
5. **Error Recovery**: Retry logic and dead letter handling

### Topic Schema

The unified topic contains `RoutedEvent` objects:

```json
{
  "id": "event-uuid",
  "patientId": "patient-123",
  "sourceEventType": "SEMANTIC_EVENT",
  "routingTime": 1640995200000,
  "priority": "HIGH",
  "destinations": ["neo4j", "fhir_store", "analytics"],
  "originalPayload": {
    "clinicalData": "...",
    "patientContext": "...",
    "confidence": 0.95
  }
}
```

## Future Enhancements

### Planned Features

1. **Stream Processing Optimization**
   - Adaptive batching based on load
   - Intelligent routing based on content analysis
   - Real-time schema evolution

2. **Advanced Analytics**
   - Real-time OLAP cubes in ClickHouse
   - Graph ML models in Neo4j
   - Semantic search in Elasticsearch

3. **Enhanced Monitoring**
   - Custom clinical metrics
   - Automated anomaly detection
   - Predictive capacity planning

4. **Kafka Connect Enhancements**
   - Schema Registry integration
   - Advanced SMT transformations
   - Multi-region replication

### Configuration Evolution

The system is designed to be easily extensible:

- New data stores can be added via new Kafka Connect connectors
- Routing logic is configurable via SMT predicates
- Schema evolution is supported via Schema Registry
- No Flink code changes required for new data stores

## Support

For technical support and questions:

1. Check the troubleshooting section above
2. Review Flink job logs and metrics
3. Verify data store connectivity
4. Check shared infrastructure status

The data store integration provides a robust, scalable foundation for real-time clinical intelligence processing with comprehensive monitoring and error handling capabilities.