# Elasticsearch Projector - Quick Start Guide

Get the Elasticsearch Projector running in under 5 minutes.

## Prerequisites

- Docker and Docker Compose installed
- Kafka cluster running (or configured in environment)
- Elasticsearch service available

## Step 1: Start Infrastructure

Start the required infrastructure services:

```bash
# From stream-services directory
cd /Users/apoorvabk/Downloads/cardiofit/backend/stream-services

# Start infrastructure (Elasticsearch, MongoDB, etc.)
docker-compose -f docker-compose.module8-infrastructure.yml up -d elasticsearch

# Wait for Elasticsearch to be healthy (about 60 seconds)
docker-compose -f docker-compose.module8-infrastructure.yml ps
```

Verify Elasticsearch is running:
```bash
curl http://localhost:9200/_cluster/health?pretty
```

Expected output:
```json
{
  "cluster_name": "module8-clinical-cluster",
  "status": "green",
  "number_of_nodes": 1
}
```

## Step 2: Configure Environment

```bash
# Copy environment template
cd module8-elasticsearch-projector
cp .env.example .env

# Edit configuration (use your Kafka credentials)
nano .env
```

Minimal configuration:
```bash
KAFKA_BOOTSTRAP_SERVERS=your-kafka-server:9092
KAFKA_SECURITY_PROTOCOL=SASL_SSL
KAFKA_SASL_MECHANISM=PLAIN
KAFKA_SASL_USERNAME=your_username
KAFKA_SASL_PASSWORD=your_password
ELASTICSEARCH_URL=http://localhost:9200
```

## Step 3: Install Dependencies

```bash
# Install Python dependencies
pip install -r requirements.txt
```

## Step 4: Start the Service

### Option A: Standalone Python
```bash
# Run directly
python run_service.py
```

### Option B: Docker
```bash
# Build and run
docker build -t elasticsearch-projector .
docker run -p 8052:8052 --env-file .env elasticsearch-projector
```

### Option C: Docker Compose
```bash
# From stream-services directory
docker-compose -f docker-compose.module8-services.yml up elasticsearch-projector
```

## Step 5: Verify Service is Running

Check health endpoint:
```bash
curl http://localhost:8052/health | jq
```

Expected output:
```json
{
  "status": "healthy",
  "elasticsearch": {
    "connected": true,
    "cluster_status": "green"
  },
  "index_statistics": {
    "patients": 0,
    "clinical_events-2024": 0,
    "clinical_documents-2024": 0,
    "alerts-2024": 0
  }
}
```

## Step 6: Test Indexing

Wait for events to flow through the pipeline (if producers are running), or check current state:

```bash
# Check processing statistics
curl http://localhost:8052/stats | jq
```

Output:
```json
{
  "statistics": {
    "events_indexed": 1234,
    "patients_updated": 56,
    "documents_created": 89,
    "alerts_created": 12,
    "errors": 0
  }
}
```

## Step 7: Test Search

Perform a full-text search:

```bash
curl -X POST http://localhost:8052/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "heart rate",
    "size": 5
  }' | jq
```

Search for high-risk patients:
```bash
curl -X POST http://localhost:8052/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mlPredictions.riskLevel:HIGH",
    "size": 10
  }' | jq
```

## Step 8: View Active Alerts

Get all active alerts:
```bash
curl http://localhost:8052/alerts/active | jq
```

Get only critical alerts:
```bash
curl "http://localhost:8052/alerts/active?severity=CRITICAL" | jq
```

## Step 9: Run Comprehensive Tests

```bash
# Run full test suite
python test_elasticsearch_projector.py
```

Expected output:
```
============================================================
ELASTICSEARCH PROJECTOR TEST SUITE
============================================================

=== Test 1: Elasticsearch Connection ===
✅ Elasticsearch connection successful
✅ Cluster status: green
✅ Number of nodes: 1

=== Test 2: Index Template Creation ===
✅ Template exists: clinical_events
✅ Template exists: patients
✅ Template exists: clinical_documents
✅ Template exists: alerts

... (more tests) ...

============================================================
TEST SUMMARY
============================================================
✅ PASS: Elasticsearch Connection
✅ PASS: Index Creation
✅ PASS: Event Indexing
... (all tests)

Total: 9/9 tests passed (100.0%)
============================================================
```

## Common Commands

### Check Elasticsearch indices
```bash
curl http://localhost:9200/_cat/indices?v
```

### Search directly in Elasticsearch
```bash
curl -X POST http://localhost:9200/clinical_events-*/_search \
  -H "Content-Type: application/json" \
  -d '{
    "query": {"match_all": {}},
    "size": 10
  }' | jq
```

### View index mapping
```bash
curl http://localhost:9200/clinical_events-2024/_mapping?pretty
```

### Get patient details
```bash
curl http://localhost:8052/search/patient/P1001 | jq
```

### Risk distribution
```bash
curl http://localhost:8052/aggregations/risk-distribution | jq
```

## Troubleshooting

### Service won't start
```bash
# Check logs
docker logs module8-elasticsearch-projector

# Verify Elasticsearch is accessible
curl http://localhost:9200/_cluster/health
```

### No events being indexed
```bash
# Verify Kafka connection
docker logs module8-elasticsearch-projector | grep "Kafka"

# Check Kafka topic has messages
# (use your Kafka tools to verify prod.ehr.events.enriched has data)

# Check consumer group
# (verify elasticsearch-projector-group is consuming)
```

### Search not returning results
```bash
# Verify indices exist
curl http://localhost:9200/_cat/indices?v

# Check index document count
curl http://localhost:9200/clinical_events-*/_count

# Force index refresh
curl -X POST http://localhost:9200/clinical_events-*/_refresh
```

### Performance issues
```bash
# Check Elasticsearch cluster stats
curl http://localhost:9200/_nodes/stats?pretty

# Check JVM heap usage
curl http://localhost:9200/_nodes/stats/jvm?pretty

# Monitor batch processing
curl http://localhost:8052/stats
```

## Next Steps

1. **Set up Kibana** for visualization dashboards
2. **Configure index lifecycle management** for automatic index rotation
3. **Set up monitoring** with Prometheus metrics
4. **Create saved searches** for common clinical queries
5. **Build clinical dashboards** with aggregations
6. **Configure alerting** based on Elasticsearch queries

## Advanced Usage

### Create custom index
```bash
curl -X PUT http://localhost:9200/custom_clinical_data \
  -H "Content-Type: application/json" \
  -d '{
    "mappings": {
      "properties": {
        "customField": {"type": "keyword"}
      }
    }
  }'
```

### Bulk search across patients
```bash
curl -X POST http://localhost:9200/patients/_search \
  -H "Content-Type: application/json" \
  -d '{
    "query": {
      "range": {
        "currentState.currentRiskScore": {"gte": 0.7}
      }
    },
    "size": 100
  }' | jq
```

### Time-based aggregations
```bash
curl -X POST http://localhost:9200/clinical_events-*/_search \
  -H "Content-Type: application/json" \
  -d '{
    "size": 0,
    "aggs": {
      "events_per_hour": {
        "date_histogram": {
          "field": "timestamp",
          "calendar_interval": "1h"
        }
      }
    }
  }' | jq
```

## Production Considerations

1. **Security**: Enable Elasticsearch security (xpack.security.enabled=true)
2. **Cluster**: Run multi-node Elasticsearch cluster for high availability
3. **Backup**: Configure snapshot repository for index backups
4. **Monitoring**: Set up Elasticsearch monitoring and alerting
5. **Index Lifecycle**: Implement ILM policies for automatic index management
6. **Performance**: Tune JVM heap, shard count, and refresh intervals

## Support

For issues or questions:
1. Check service logs: `docker logs module8-elasticsearch-projector`
2. Verify Elasticsearch health: `curl http://localhost:9200/_cluster/health`
3. Review test output: `python test_elasticsearch_projector.py`
4. Check API documentation: `http://localhost:8052/docs`
