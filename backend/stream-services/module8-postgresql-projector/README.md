# PostgreSQL Projector Service

Projects enriched clinical events from Kafka topic `prod.ehr.events.enriched` to PostgreSQL database with optimized schema design and batch processing.

## Architecture

```
Kafka Topic (prod.ehr.events.enriched)
    ↓
PostgreSQL Projector (this service)
    ↓
PostgreSQL Database (module8_projections schema)
    ├── enriched_events (raw JSONB storage)
    ├── patient_vitals (normalized vital signs)
    ├── clinical_scores (risk scores and predictions)
    └── event_metadata (searchable attributes)
```

## Features

- **Batch Processing**: Configurable batch size (default: 100) and timeout (default: 5s)
- **Transaction Safety**: Full ACID compliance with rollback on failures
- **Optimized Schema**: 4 tables with proper indexes for fast queries
- **DLQ Support**: Failed messages sent to `prod.ehr.dlq.postgresql`
- **Monitoring**: Prometheus metrics and health checks
- **Type Safety**: Pydantic models for data validation
- **Auto-Schema Creation**: Automatically creates schema and tables on startup

## Database Schema

### Table 1: enriched_events
```sql
CREATE TABLE enriched_events (
    event_id VARCHAR(255) PRIMARY KEY,
    patient_id VARCHAR(255) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    event_data JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

**Indexes**:
- `idx_enriched_events_patient_timestamp` (patient_id, timestamp DESC)
- `idx_enriched_events_type` (event_type)
- `idx_enriched_events_data_gin` GIN index on JSONB

### Table 2: patient_vitals
```sql
CREATE TABLE patient_vitals (
    id SERIAL PRIMARY KEY,
    event_id VARCHAR(255) UNIQUE NOT NULL,
    patient_id VARCHAR(255) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    heart_rate INTEGER,
    bp_systolic INTEGER,
    bp_diastolic INTEGER,
    spo2 NUMERIC(5, 2),
    temperature_celsius NUMERIC(5, 2)
);
```

**Indexes**:
- `idx_patient_vitals_patient_timestamp` (patient_id, timestamp DESC)

### Table 3: clinical_scores
```sql
CREATE TABLE clinical_scores (
    id SERIAL PRIMARY KEY,
    event_id VARCHAR(255) UNIQUE NOT NULL,
    patient_id VARCHAR(255) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    news2_score INTEGER,
    qsofa_score INTEGER,
    risk_level VARCHAR(50),
    sepsis_risk_24h NUMERIC(5, 4),
    cardiac_risk_7d NUMERIC(5, 4),
    readmission_risk_30d NUMERIC(5, 4)
);
```

**Indexes**:
- `idx_clinical_scores_patient_timestamp` (patient_id, timestamp DESC)
- `idx_clinical_scores_risk_level` (risk_level)

### Table 4: event_metadata
```sql
CREATE TABLE event_metadata (
    event_id VARCHAR(255) PRIMARY KEY,
    patient_id VARCHAR(255) NOT NULL,
    encounter_id VARCHAR(255),
    department_id VARCHAR(255),
    device_id VARCHAR(255),
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    event_type VARCHAR(50) NOT NULL
);
```

**Indexes**:
- `idx_event_metadata_patient_timestamp` (patient_id, timestamp DESC)
- `idx_event_metadata_encounter_id` (encounter_id)
- `idx_event_metadata_department_id` (department_id)

### Views

**latest_patient_vitals**: Latest vitals per patient
**high_risk_patients**: Patients with high/critical risk levels
**complete_event_detail**: Comprehensive event view joining all tables

## Installation

### Prerequisites
- Python 3.11+
- PostgreSQL 13+ (existing container a2f55d83b1fa)
- Access to Confluent Cloud Kafka cluster
- module8-shared installed

### Setup

1. **Install Dependencies**:
```bash
cd module8-postgresql-projector
pip install -r requirements.txt
```

2. **Configure Environment**:
```bash
cp .env.example .env
# Edit .env with your Kafka credentials and PostgreSQL settings
```

3. **Initialize Database** (automatic on first run):
```bash
# Schema will be created automatically from schema/init.sql
# Or manually run:
psql -h 172.21.0.4 -U cardiofit_user -d cardiofit -f schema/init.sql
```

4. **Run Service**:
```bash
python -m app.main
```

Service will start on `http://localhost:8050`

## Docker Deployment

### Build Image
```bash
docker build -t postgresql-projector:latest .
```

### Run Container
```bash
docker run -d \
  --name postgresql-projector \
  --env-file .env \
  -p 8050:8050 \
  -p 9090:9090 \
  postgresql-projector:latest
```

## API Endpoints

### Health Check
```bash
GET http://localhost:8050/health
```

**Response**:
```json
{
  "status": "healthy",
  "timestamp": "2025-11-15T20:30:00Z",
  "service": "postgresql-projector",
  "version": "1.0.0"
}
```

### Metrics (Prometheus)
```bash
GET http://localhost:8050/metrics
```

**Metrics Available**:
- `projector_messages_consumed_total`: Total messages consumed from Kafka
- `projector_messages_processed_total`: Total messages successfully processed
- `projector_messages_failed_total`: Total messages failed
- `projector_batches_processed_total`: Total batches processed
- `projector_consumer_lag`: Current consumer lag

### Status
```bash
GET http://localhost:8050/status
```

**Response**:
```json
{
  "service": "postgresql-projector",
  "status": "running",
  "kafka_connected": true,
  "postgres_connected": true,
  "consumer_group": "module8-postgresql-projector",
  "topics": ["prod.ehr.events.enriched"],
  "batch_size": 100,
  "batch_timeout_seconds": 5.0,
  "metrics": {
    "messages_consumed": 15420,
    "messages_processed": 15420,
    "messages_failed": 0,
    "batches_processed": 155,
    "consumer_lag": 0,
    "uptime_seconds": 3600.5
  },
  "last_processed": "2025-11-15T20:29:45Z"
}
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `KAFKA_BOOTSTRAP_SERVERS` | Kafka broker addresses | pkc-p11w6... |
| `KAFKA_API_KEY` | Confluent Cloud API key | Required |
| `KAFKA_API_SECRET` | Confluent Cloud API secret | Required |
| `POSTGRES_HOST` | PostgreSQL host | 172.21.0.4 |
| `POSTGRES_PORT` | PostgreSQL port | 5432 |
| `POSTGRES_DB` | Database name | cardiofit |
| `POSTGRES_USER` | Database user | cardiofit_user |
| `POSTGRES_PASSWORD` | Database password | Required |
| `BATCH_SIZE` | Batch size for processing | 100 |
| `BATCH_TIMEOUT_SECONDS` | Batch timeout in seconds | 5.0 |
| `SERVICE_PORT` | FastAPI service port | 8050 |

## Querying Data

### Latest Vitals for Patient
```sql
SELECT * FROM module8_projections.latest_patient_vitals
WHERE patient_id = 'PAT-001';
```

### High-Risk Patients
```sql
SELECT * FROM module8_projections.high_risk_patients
ORDER BY sepsis_risk_24h DESC
LIMIT 10;
```

### Patient Event Timeline
```sql
SELECT
    timestamp,
    event_type,
    heart_rate,
    bp_systolic,
    news2_score,
    risk_level
FROM module8_projections.complete_event_detail
WHERE patient_id = 'PAT-001'
ORDER BY timestamp DESC;
```

### Events by Department
```sql
SELECT
    department_id,
    COUNT(*) as event_count,
    COUNT(DISTINCT patient_id) as unique_patients
FROM module8_projections.event_metadata
WHERE timestamp > NOW() - INTERVAL '24 hours'
GROUP BY department_id;
```

## Monitoring

### Health Monitoring
```bash
# Check service health
curl http://localhost:8050/health

# Check consumer lag
curl http://localhost:8050/status | jq '.metrics.consumer_lag'
```

### Prometheus Integration
Add to Prometheus scrape config:
```yaml
scrape_configs:
  - job_name: 'postgresql-projector'
    static_configs:
      - targets: ['localhost:8050']
    metrics_path: '/metrics'
```

### Database Monitoring
```sql
-- Table sizes
SELECT
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'module8_projections'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Row counts
SELECT
    'enriched_events' as table_name,
    COUNT(*) as row_count
FROM module8_projections.enriched_events
UNION ALL
SELECT 'patient_vitals', COUNT(*) FROM module8_projections.patient_vitals
UNION ALL
SELECT 'clinical_scores', COUNT(*) FROM module8_projections.clinical_scores
UNION ALL
SELECT 'event_metadata', COUNT(*) FROM module8_projections.event_metadata;
```

## Troubleshooting

### Connection Issues
```bash
# Test PostgreSQL connection
psql -h 172.21.0.4 -U cardiofit_user -d cardiofit -c "SELECT version();"

# Check PostgreSQL container
docker ps | grep postgres
docker logs a2f55d83b1fa
```

### Consumer Lag
```bash
# Check consumer lag
curl http://localhost:8050/status | jq '.metrics.consumer_lag'

# If lag is high, increase batch size
export BATCH_SIZE=500
```

### Schema Issues
```bash
# Manually recreate schema
psql -h 172.21.0.4 -U cardiofit_user -d cardiofit -f schema/init.sql

# Drop and recreate
psql -h 172.21.0.4 -U cardiofit_user -d cardiofit -c "DROP SCHEMA IF EXISTS module8_projections CASCADE;"
psql -h 172.21.0.4 -U cardiofit_user -d cardiofit -f schema/init.sql
```

## Performance Tuning

### Batch Size Optimization
- **Small batches (50-100)**: Lower latency, higher CPU usage
- **Large batches (500-1000)**: Higher throughput, higher memory usage
- **Recommended**: Start with 100, increase based on consumer lag

### PostgreSQL Tuning
```sql
-- Increase work memory for complex queries
SET work_mem = '256MB';

-- Increase maintenance work memory for index creation
SET maintenance_work_mem = '512MB';

-- Enable parallel queries
SET max_parallel_workers_per_gather = 4;
```

### Index Maintenance
```sql
-- Analyze tables for query optimization
ANALYZE module8_projections.enriched_events;
ANALYZE module8_projections.patient_vitals;
ANALYZE module8_projections.clinical_scores;
ANALYZE module8_projections.event_metadata;

-- Reindex if needed
REINDEX SCHEMA module8_projections;
```

## Architecture Notes

- **Shared Module**: Uses `module8-shared` KafkaConsumerBase for Kafka logic
- **Data Validation**: Pydantic models from shared module ensure type safety
- **Error Handling**: Failed messages sent to DLQ topic `prod.ehr.dlq.postgresql`
- **Graceful Shutdown**: Handles SIGTERM/SIGINT with proper cleanup
- **Thread Safety**: Consumer runs in background thread, FastAPI in main thread

## License

CardioFit Platform - Clinical Synthesis Hub
