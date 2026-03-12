# MongoDB Projector Service - Module 8

MongoDB projector that consumes enriched clinical events from `prod.ehr.events.enriched` and writes to MongoDB collections with rich metadata, aggregations, and ML explanations.

## Overview

The MongoDB Projector service is part of Module 8's multi-sink projection layer. It consumes fully enriched events (validation + FHIR transformation + semantic enrichment + ML predictions) and projects them into three MongoDB collections optimized for different access patterns.

## MongoDB Collections

### 1. `clinical_documents`
Full clinical event documents with complete enrichment data.

**Purpose**: Primary event storage with rich metadata for detailed queries and analytics.

**Schema**:
```javascript
{
  _id: "event_id",                    // Unique event ID
  patientId: "patient_id",
  timestamp: ISODate,
  eventType: "vital_signs|lab_result|device_data",
  deviceType: "monitor|ventilator|pump",

  // Original data
  vitalSigns: {
    heartRate: 85,
    bloodPressureSystolic: 120,
    bloodPressureDiastolic: 80,
    temperature: 37.2,
    oxygenSaturation: 98
  },
  labResults: { /* ... */ },

  // Enrichments
  enrichments: {
    riskLevel: "NORMAL|ELEVATED|HIGH|CRITICAL",
    earlyWarningScore: 2,
    clinicalContext: { /* ... */ },
    deviceContext: { /* ... */ }
  },

  mlPredictions: {
    predictions: {
      sepsis_risk_24h: {
        modelName: "sepsis_xgboost_v1",
        prediction: 0.35,
        confidence: 0.82,
        threshold: 0.5,
        alertTriggered: false,
        shapValues: { /* ... */ },
        limeExplanation: { /* ... */ }
      }
    },
    featureImportance: { /* ... */ }
  },

  // Metadata
  ingestionTime: ISODate,
  processingTime: ISODate,
  createdAt: ISODate,
  summary: "vital_signs event | Vitals: HR 85, BP 120/80, Temp 37.2°C | Risk: NORMAL"
}
```

**Indexes**:
- `{patientId: 1, timestamp: -1}` - Patient event history
- `{eventType: 1}` - Event type filtering
- `{enrichments.riskLevel: 1}` - Risk-based queries
- `{timestamp: -1}` - Temporal queries

### 2. `patient_timelines`
Aggregated patient event history (max 1000 most recent events per patient).

**Purpose**: Fast patient timeline retrieval without scanning full event history.

**Schema**:
```javascript
{
  _id: "patient_id",                  // Patient ID as primary key
  events: [                           // Array of latest 1000 events (sorted newest first)
    {
      eventId: "event_id",
      timestamp: ISODate,
      eventType: "vital_signs",
      summary: "HR 85, BP 120/80",
      riskLevel: "NORMAL",
      vitalSigns: { heartRate: 85, ... },
      predictions: { sepsis_risk_24h: 0.35 }
    }
  ],
  lastUpdated: ISODate,
  eventCount: 1523,                   // Total events for this patient
  firstEventTime: ISODate,
  latestEventTime: ISODate
}
```

**Update Strategy**:
- Use `$push` with `$each`, `$sort`, and `$slice` to maintain sorted array of latest 1000 events
- Automatically removes oldest events when limit exceeded
- Single document per patient for fast timeline queries

**Indexes**:
- `{_id: 1}` - Primary key (patient ID)
- `{lastUpdated: -1}` - Recently active patients

### 3. `ml_explanations`
ML model predictions with interpretability data (SHAP, LIME).

**Purpose**: Model explainability and audit trail for clinical AI decisions.

**Schema**:
```javascript
{
  patientId: "patient_id",
  eventId: "event_id",
  timestamp: ISODate,
  predictions: {
    sepsis_risk_24h: {
      model_name: "sepsis_xgboost_v1",
      prediction: 0.35,
      confidence: 0.82,
      threshold: 0.5,
      alert_triggered: false,
      shap_values: {
        heart_rate: 0.05,
        temperature: 0.12,
        wbc_count: 0.08
      },
      lime_explanation: {
        features: [...],
        weights: [...]
      }
    },
    mortality_risk_48h: { /* ... */ }
  },
  feature_importance: {
    heart_rate: 0.25,
    temperature: 0.18,
    wbc_count: 0.15
  },
  created_at: ISODate
}
```

**Indexes**:
- `{patientId: 1, timestamp: -1}` - Patient prediction history
- `{predictions.sepsis_risk_24h.prediction: -1}` - High-risk patient queries

## Architecture

```
Kafka: prod.ehr.events.enriched
         ↓
    [MongoDB Projector]
         ↓
    [Batch Processing]
    (bulk_write operations)
         ↓
    ┌────────────────┐
    │ MongoDB        │
    ├────────────────┤
    │ clinical_docs  │ ← Full events with enrichments
    │ timelines      │ ← Aggregated patient histories
    │ ml_explanations│ ← Model interpretability
    └────────────────┘
```

## Processing Logic

### Batch Processing
1. **Consume**: Read batch of enriched events from Kafka (batch_size=50, timeout=10s)
2. **Transform**: Convert to MongoDB documents with proper structure
3. **Bulk Write**: Use MongoDB bulk_write for performance
   - Clinical documents: UpdateOne with upsert=True
   - Patient timelines: UpdateOne with $push, $sort, $slice
   - ML explanations: InsertMany for new predictions
4. **Commit**: Commit Kafka offsets after successful write

### Event Summary Generation
Automatic human-readable summaries:
```python
"vital_signs event | Vitals: HR 85, BP 120/80, Temp 37.2°C, SpO2 98% | Risk: NORMAL"
"lab_result event | Labs: WBC 12.5, CRP 45 | Risk: ELEVATED | Alerts: sepsis_risk_24h"
```

### Patient Timeline Aggregation
Smart aggregation using MongoDB's array operators:
```javascript
{
  $push: {
    events: {
      $each: [new_event],
      $sort: { timestamp: -1 },  // Keep newest first
      $slice: 1000               // Limit to 1000 events
    }
  }
}
```

## Configuration

Environment variables (see `.env.example`):

**Service**:
- `SERVICE_PORT`: 8051

**Kafka**:
- `KAFKA_BOOTSTRAP_SERVERS`: Kafka broker addresses
- `KAFKA_TOPIC`: prod.ehr.events.enriched
- `KAFKA_GROUP_ID`: mongodb-projector-group

**MongoDB**:
- `MONGODB_URI`: mongodb://localhost:27017 (use mongodb://mongodb:27017 for Docker)
- `MONGODB_DATABASE`: module8_clinical
- `MONGODB_MAX_POOL_SIZE`: 50
- `MONGODB_MIN_POOL_SIZE`: 10

**Batch Processing**:
- `BATCH_SIZE`: 50
- `BATCH_TIMEOUT_SECONDS`: 10
- `MAX_RETRIES`: 3

**Patient Timeline**:
- `MAX_EVENTS_PER_PATIENT`: 1000

## API Endpoints

### Health & Status
- `GET /health` - Service health check
- `GET /metrics` - Processing metrics
- `GET /status` - Detailed projector status

### MongoDB Stats
- `GET /collections/stats` - Collection statistics (count, size, indexes)

### Control (Future)
- `POST /control/pause` - Pause projection
- `POST /control/resume` - Resume projection

## Running the Service

### Local Development
```bash
# Install dependencies
pip install -r requirements.txt

# Configure environment
cp .env.example .env
# Edit .env with your settings

# Start MongoDB locally
docker run -d -p 27017:27017 --name mongodb mongo:7

# Run the service
python -m uvicorn app.main:app --host 0.0.0.0 --port 8051 --reload
```

### Docker
```bash
# Build image
docker build -t mongodb-projector:latest .

# Run container
docker run -d \
  --name mongodb-projector \
  -p 8051:8051 \
  --env-file .env \
  mongodb-projector:latest
```

### Docker Compose (with dependencies)
```yaml
version: '3.8'

services:
  mongodb:
    image: mongo:7
    ports:
      - "27017:27017"
    volumes:
      - mongodb_data:/data/db

  mongodb-projector:
    build: .
    ports:
      - "8051:8051"
    environment:
      MONGODB_URI: mongodb://mongodb:27017
      KAFKA_BOOTSTRAP_SERVERS: kafka:9092
    depends_on:
      - mongodb

volumes:
  mongodb_data:
```

## Monitoring

### Metrics Available
```json
{
  "messages_consumed": 15234,
  "batches_processed": 305,
  "documents_written": 15234,
  "timelines_updated": 2341,
  "explanations_written": 12456,
  "errors": 0,
  "total_clinical_docs": 125678,
  "total_patient_timelines": 2341,
  "total_ml_explanations": 98765
}
```

### Performance Characteristics
- **Throughput**: ~500-1000 events/second (with batch_size=50)
- **Latency**: <100ms per batch (50 events)
- **MongoDB Write Performance**: Bulk operations provide 10-50x speedup vs individual writes

### Index Usage
Monitor index usage with:
```javascript
db.clinical_documents.aggregate([{ $indexStats: {} }])
```

## Query Examples

### Get Patient Timeline
```javascript
db.patient_timelines.findOne({ _id: "patient_123" })
```

### Find High-Risk Events
```javascript
db.clinical_documents.find({
  "enrichments.riskLevel": "CRITICAL",
  timestamp: { $gte: ISODate("2024-01-01") }
}).sort({ timestamp: -1 })
```

### Get ML Explanations for High-Risk Predictions
```javascript
db.ml_explanations.find({
  "predictions.sepsis_risk_24h.prediction": { $gte: 0.7 },
  "predictions.sepsis_risk_24h.alert_triggered": true
}).sort({ timestamp: -1 })
```

### Aggregation: Risk Level Distribution
```javascript
db.clinical_documents.aggregate([
  { $group: {
      _id: "$enrichments.riskLevel",
      count: { $sum: 1 }
  }},
  { $sort: { count: -1 } }
])
```

## Testing

### Test Queries
```bash
# Health check
curl http://localhost:8051/health

# Get metrics
curl http://localhost:8051/metrics

# Get collection stats
curl http://localhost:8051/collections/stats

# Get detailed status
curl http://localhost:8051/status
```

### MongoDB Queries
```bash
# Connect to MongoDB
mongosh mongodb://localhost:27017/module8_clinical

# Check collection counts
db.clinical_documents.countDocuments()
db.patient_timelines.countDocuments()
db.ml_explanations.countDocuments()

# View recent events
db.clinical_documents.find().sort({ timestamp: -1 }).limit(5)

# Check indexes
db.clinical_documents.getIndexes()
db.patient_timelines.getIndexes()
db.ml_explanations.getIndexes()
```

## Performance Tuning

### Batch Size Optimization
- Smaller batches (10-25): Lower latency, higher throughput for low-volume streams
- Larger batches (50-100): Better MongoDB bulk write performance, higher latency

### MongoDB Connection Pool
- Adjust `MONGODB_MAX_POOL_SIZE` based on concurrent writes
- Monitor connection pool usage with MongoDB metrics

### Index Tuning
- Monitor slow queries with MongoDB profiler
- Add compound indexes for common query patterns
- Consider TTL indexes for data retention policies

## Troubleshooting

### Projector Not Writing
1. Check Kafka connection and topic availability
2. Verify MongoDB connection and authentication
3. Check logs for batch processing errors
4. Verify enriched events have correct schema

### Performance Issues
1. Monitor MongoDB index usage
2. Check batch size and timeout settings
3. Monitor MongoDB connection pool
4. Review bulk write performance metrics

### Data Quality Issues
1. Verify enriched events contain all required fields
2. Check event summary generation logic
3. Validate MongoDB schema constraints
4. Review ML prediction data structure

## Future Enhancements

1. **TTL Indexes**: Automatic data retention policies
2. **Change Streams**: Real-time notifications for critical events
3. **Aggregation Pipelines**: Pre-computed analytics views
4. **Sharding**: Horizontal scaling for large datasets
5. **Backup/Restore**: Automated backup strategies
6. **Query Optimization**: Materialized views for common queries
