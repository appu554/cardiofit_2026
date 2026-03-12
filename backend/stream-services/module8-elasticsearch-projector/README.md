# Elasticsearch Projector Service

High-performance clinical event search and analytics service that consumes enriched events from Kafka and indexes them to Elasticsearch for full-text search, real-time dashboards, and aggregations.

## Features

- **Multi-Index Strategy**: Separate indices for events, patients, documents, and alerts
- **Bulk Indexing**: High-throughput batch processing with optimistic concurrency
- **Full-Text Search**: Clinical analyzer with synonym support for medical terminology
- **Real-Time Alerts**: Sub-second indexing for critical patient alerts
- **Time-Based Partitioning**: Automatic index rotation by year for scalability
- **Patient State Tracking**: Current patient state with latest vitals and risk scores
- **Clinical Documents**: Full-text searchable clinical notes and documentation
- **RESTful Search API**: FastAPI endpoints for search, aggregations, and analytics

## Architecture

```
Kafka Topic: prod.ehr.events.enriched
       ↓
ElasticsearchProjector (Batch Consumer)
       ↓
   Bulk Operations
       ↓
┌──────────────────────────────────────┐
│ Elasticsearch Indices                │
├──────────────────────────────────────┤
│ 1. clinical_events-YYYY              │
│    - All enriched events             │
│    - Full ML predictions             │
│    - Semantic annotations            │
│                                      │
│ 2. patients                          │
│    - Current patient state           │
│    - Latest vitals summary           │
│    - Risk level tracking             │
│                                      │
│ 3. clinical_documents-YYYY           │
│    - Full-text searchable notes      │
│    - Clinical synonym support        │
│                                      │
│ 4. alerts-YYYY                       │
│    - Real-time alerts                │
│    - Risk-based notifications        │
└──────────────────────────────────────┘
```

## Index Mappings

### Clinical Events Index
```json
{
  "eventId": "keyword",
  "patientId": "keyword",
  "timestamp": "date",
  "rawData": {
    "heartRate": "integer",
    "bloodPressure": {"systolic": "integer", "diastolic": "integer"},
    "oxygenSaturation": "float"
  },
  "enrichments": {"fhirResources": "object"},
  "semanticAnnotations": {"medicalConcepts": "nested"},
  "mlPredictions": {
    "riskScore": "float",
    "riskLevel": "keyword",
    "predictions": "nested"
  }
}
```

### Patients Index
```json
{
  "patientId": "keyword",
  "demographics": {"name": "text", "age": "integer"},
  "currentState": {
    "latestEventId": "keyword",
    "currentRiskLevel": "keyword",
    "currentRiskScore": "float"
  },
  "vitalsSummary": {
    "latestHeartRate": "integer",
    "latestBP": "object"
  }
}
```

### Alerts Index
```json
{
  "alertId": "keyword",
  "patientId": "keyword",
  "severity": "keyword",  // LOW, MEDIUM, HIGH, CRITICAL
  "riskScore": "float",
  "triggeredBy": {
    "metric": "keyword",
    "value": "float",
    "threshold": "float"
  },
  "acknowledged": "boolean"
}
```

## Installation

```bash
# Install dependencies
pip install -r requirements.txt

# Copy environment configuration
cp .env.example .env

# Edit configuration
nano .env
```

## Configuration

Key environment variables:

```bash
# Elasticsearch
ELASTICSEARCH_URL=http://elasticsearch:9200

# Kafka
KAFKA_BOOTSTRAP_SERVERS=localhost:9092

# Performance
BATCH_SIZE=100          # Events per bulk operation
FLUSH_TIMEOUT=5         # Seconds before forcing flush
```

## Running the Service

### Standalone
```bash
cd src
python main.py
```

### Docker
```bash
docker build -t elasticsearch-projector .
docker run -p 8052:8052 \
  -e ELASTICSEARCH_URL=http://elasticsearch:9200 \
  -e KAFKA_BOOTSTRAP_SERVERS=kafka:9092 \
  elasticsearch-projector
```

### Docker Compose
```bash
# Add to your docker-compose.yml
docker-compose up elasticsearch-projector
```

## API Endpoints

### Health Check
```bash
curl http://localhost:8052/health
```

Response:
```json
{
  "status": "healthy",
  "elasticsearch": {
    "connected": true,
    "cluster_status": "green",
    "number_of_nodes": 3
  },
  "index_statistics": {
    "patients": 1250,
    "clinical_events-2024": 45230,
    "clinical_documents-2024": 8920,
    "alerts-2024": 342
  },
  "processing_statistics": {
    "events_indexed": 45230,
    "patients_updated": 1250,
    "documents_created": 8920,
    "alerts_created": 342
  }
}
```

### Full-Text Search
```bash
curl -X POST http://localhost:8052/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "high blood pressure AND diabetes",
    "index": "clinical_events-*",
    "size": 10
  }'
```

Response:
```json
{
  "total": 156,
  "hits": [
    {
      "id": "evt_abc123",
      "score": 8.42,
      "source": {
        "eventId": "evt_abc123",
        "patientId": "P1001",
        "timestamp": "2024-11-15T10:30:00Z",
        "mlPredictions": {
          "riskScore": 0.85,
          "riskLevel": "HIGH"
        }
      }
    }
  ],
  "took": 15
}
```

### Patient Events
```bash
curl http://localhost:8052/search/patient/P1001?limit=50
```

### Active Alerts
```bash
# All active alerts
curl http://localhost:8052/alerts/active

# Critical alerts only
curl "http://localhost:8052/alerts/active?severity=CRITICAL"
```

### Risk Distribution
```bash
curl http://localhost:8052/aggregations/risk-distribution
```

Response:
```json
{
  "distribution": [
    {"riskLevel": "LOW", "count": 850},
    {"riskLevel": "MEDIUM", "count": 320},
    {"riskLevel": "HIGH", "count": 65},
    {"riskLevel": "CRITICAL", "count": 15}
  ]
}
```

## Search Examples

### Clinical Terminology Search (with synonyms)
```bash
# Automatically expands to "blood pressure OR bp"
curl -X POST http://localhost:8052/search \
  -d '{"query": "bp > 140"}'
```

### Time-Range Search
```bash
curl -X POST http://localhost:8052/search \
  -d '{
    "query": "timestamp:[2024-11-01 TO 2024-11-15] AND riskLevel:HIGH"
  }'
```

### Nested Field Search
```bash
curl -X POST http://localhost:8052/search \
  -d '{
    "query": "mlPredictions.predictions.condition:diabetes AND mlPredictions.predictions.probability:>0.8"
  }'
```

## Performance Characteristics

- **Indexing Throughput**: 10,000+ events/second with bulk operations
- **Search Latency**: Sub-second for most queries (<100ms typical)
- **Concurrent Searches**: 100+ concurrent queries without degradation
- **Storage Efficiency**: ~2KB per event document (compressed)
- **Index Refresh**: 5-second near real-time visibility

## Monitoring

### Index Statistics
```bash
# Get all index stats
curl http://localhost:8052/stats
```

### Elasticsearch Cluster Health
```bash
curl http://elasticsearch:9200/_cluster/health?pretty
```

### Index Size
```bash
curl http://elasticsearch:9200/_cat/indices?v
```

## Index Management

### Manual Index Creation
```bash
# Service automatically creates indices on startup
# To manually create an index:
curl -X PUT http://elasticsearch:9200/clinical_events-2024
```

### Delete Old Indices
```bash
# Delete indices older than 2023
curl -X DELETE http://elasticsearch:9200/clinical_events-2023
curl -X DELETE http://elasticsearch:9200/alerts-2023
```

### Reindex Data
```bash
# Elasticsearch reindex API
curl -X POST http://elasticsearch:9200/_reindex \
  -H "Content-Type: application/json" \
  -d '{
    "source": {"index": "clinical_events-2023"},
    "dest": {"index": "clinical_events-archive-2023"}
  }'
```

## Troubleshooting

### Connection Issues
```bash
# Test Elasticsearch connectivity
curl http://elasticsearch:9200/_cluster/health

# Check service logs
docker logs elasticsearch-projector

# Verify Kafka connectivity
docker logs elasticsearch-projector | grep "Kafka"
```

### Indexing Errors
```bash
# Check processing statistics
curl http://localhost:8052/stats

# Look for bulk operation failures in logs
docker logs elasticsearch-projector | grep "failed"
```

### Performance Issues
```bash
# Check cluster performance
curl http://elasticsearch:9200/_nodes/stats?pretty

# Monitor JVM heap
curl http://elasticsearch:9200/_nodes/stats/jvm?pretty

# Check thread pools
curl http://elasticsearch:9200/_cat/thread_pool?v
```

## Advanced Features

### Custom Analyzers
The service uses clinical-specific analyzers:
- **clinical_analyzer**: Standard tokenization with medical term stemming
- **clinical_text_analyzer**: Full-text with synonym expansion (bp→blood pressure)

### Aggregation Queries
```bash
# Average risk score by hour
curl -X POST http://elasticsearch:9200/clinical_events-*/_search \
  -d '{
    "size": 0,
    "aggs": {
      "risk_over_time": {
        "date_histogram": {
          "field": "timestamp",
          "interval": "1h"
        },
        "aggs": {
          "avg_risk": {"avg": {"field": "mlPredictions.riskScore"}}
        }
      }
    }
  }'
```

## Integration with Other Services

- **PostgreSQL Projector**: Provides structured queries; Elasticsearch provides full-text search
- **MongoDB Projector**: Provides document flexibility; Elasticsearch provides search speed
- **ML Prediction Service**: Generates predictions indexed by Elasticsearch
- **Semantic Enrichment**: Annotations enable powerful faceted search

## License

Part of the Clinical Synthesis Hub CardioFit Platform
