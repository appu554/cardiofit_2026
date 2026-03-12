# UPS (Unified Patient Summary) Read Model Projector

Consumes enriched events from `prod.ehr.events.enriched` and maintains a denormalized patient summary table in PostgreSQL optimized for sub-10ms queries.

## Overview

The UPS Projector creates and maintains a comprehensive, denormalized view of each patient's current state by consuming enriched clinical events. This read model is optimized for real-time dashboard queries and clinical decision support systems.

## Architecture

```
prod.ehr.events.enriched → UPS Projector → PostgreSQL (module8_projections.ups_read_model)
                                                    ↓
                                            Real-time Dashboards
                                            Clinical Alerts UI
                                            Patient Summary API
```

## Performance Targets

- **Single patient lookup**: <10ms
- **Throughput**: 500 updates/sec
- **Batch processing**: 100 events per batch, 5s timeout
- **Hot path queries**: Department summaries, risk alerts

## Database Schema

### Main Table: `ups_read_model`

```sql
CREATE TABLE ups_read_model (
    patient_id VARCHAR(255) PRIMARY KEY,

    -- Demographics
    demographics JSONB,

    -- Location
    current_department VARCHAR(100),
    current_location VARCHAR(255),

    -- Latest Vitals
    latest_vitals JSONB,
    latest_vitals_timestamp BIGINT,

    -- Clinical Scores
    news2_score INTEGER,
    news2_category VARCHAR(20),
    qsofa_score INTEGER,
    sofa_score INTEGER,
    risk_level VARCHAR(20),

    -- ML Predictions
    ml_predictions JSONB,
    ml_predictions_timestamp BIGINT,

    -- Active Alerts
    active_alerts JSONB,
    active_alerts_count INTEGER,

    -- Protocol Compliance
    protocol_compliance JSONB,
    protocol_status VARCHAR(50),

    -- Metadata
    last_event_id VARCHAR(255),
    last_event_type VARCHAR(100),
    last_updated BIGINT,
    event_count INTEGER,
    updated_at TIMESTAMP,
    created_at TIMESTAMP
);
```

### Indexes

- **Primary**: `patient_id`
- **GIN indexes**: `latest_vitals`, `ml_predictions`, `active_alerts`, `demographics`
- **Common queries**: `risk_level`, `current_department`, `current_location`, `last_updated`
- **Composite**: `(current_department, risk_level)`, `active_alerts_count`

## UPSERT Logic

The projector uses intelligent UPSERT logic:

1. **Always update**: `last_event_id`, `last_event_type`, `last_updated`, `updated_at`
2. **Conditional update** (timestamp-based):
   - `latest_vitals` (only if newer)
   - `ml_predictions` (only if newer)
3. **Merge update**: `event_count` (increment)
4. **COALESCE update**: Most fields (prefer new value, keep old if new is NULL)

```sql
INSERT INTO ups_read_model (...)
VALUES (...)
ON CONFLICT (patient_id) DO UPDATE SET
    latest_vitals = CASE
        WHEN EXCLUDED.latest_vitals_timestamp > ups_read_model.latest_vitals_timestamp
        THEN EXCLUDED.latest_vitals
        ELSE ups_read_model.latest_vitals
    END,
    event_count = ups_read_model.event_count + EXCLUDED.event_count,
    ...
```

## Running the Service

### Prerequisites

1. **PostgreSQL** (existing container `a2f55d83b1fa`)
2. **Kafka** (Confluent Cloud with `prod.ehr.events.enriched` topic)
3. **Module8 Shared Library** (`../module8-shared`)

### Installation

```bash
cd backend/stream-services/module8-ups-projector
pip install -r requirements.txt
```

### Initialize Database

```bash
# Connect to PostgreSQL container
docker exec -it a2f55d83b1fa psql -U cardiofit_user -d cardiofit

# Run schema initialization
\i /path/to/schema/init.sql
```

Or copy schema file into container:
```bash
docker cp schema/init.sql a2f55d83b1fa:/tmp/
docker exec -it a2f55d83b1fa psql -U cardiofit_user -d cardiofit -f /tmp/init.sql
```

### Start Service

```bash
python run_service.py
```

Service runs on port **8055**.

## API Endpoints

### Health Check
```bash
GET http://localhost:8055/health

Response:
{
  "status": "healthy",
  "service": "UPS Projector",
  "kafka": {
    "status": "connected",
    "consumer_group": "module8-ups-projector",
    "topics": ["prod.ehr.events.enriched"]
  },
  "projector": {
    "events_processed": 1250,
    "patients_updated": 142,
    "batches_processed": 13,
    "avg_batch_processing_ms": 45.3,
    "avg_upsert_ms": 12.7
  },
  "database": {
    "status": "healthy",
    "total_patients": 142
  }
}
```

### Metrics
```bash
GET http://localhost:8055/metrics
```

## Query Examples

### Single Patient Lookup (Primary Use Case)

```sql
-- Get complete patient state (<10ms target)
SELECT * FROM module8_projections.ups_read_model
WHERE patient_id = 'P12345';
```

### Department Dashboard Queries

```sql
-- High-risk patients in ICU
SELECT
    patient_id,
    risk_level,
    news2_score,
    latest_vitals,
    ml_predictions,
    active_alerts_count
FROM module8_projections.ups_read_model
WHERE current_department = 'ICU_01'
  AND risk_level IN ('HIGH', 'CRITICAL')
ORDER BY last_updated DESC;

-- Department summary (using materialized view)
SELECT * FROM module8_projections.department_summary
WHERE current_department = 'ICU_01';
```

### JSONB Queries on Vitals

```sql
-- Patients with high heart rate
SELECT patient_id, latest_vitals->>'heart_rate' as hr
FROM module8_projections.ups_read_model
WHERE (latest_vitals->>'heart_rate')::int > 100;

-- Patients with low SpO2
SELECT patient_id, latest_vitals->>'spo2' as spo2, risk_level
FROM module8_projections.ups_read_model
WHERE (latest_vitals->>'spo2')::int < 90;
```

### Active Alerts

```sql
-- All patients with critical alerts
SELECT
    patient_id,
    current_department,
    active_alerts_count,
    active_alerts
FROM module8_projections.ups_read_model
WHERE active_alerts_count > 0
ORDER BY active_alerts_count DESC;
```

### ML Predictions

```sql
-- Patients with high sepsis risk
SELECT
    patient_id,
    ml_predictions->>'sepsis_probability' as sepsis_prob,
    risk_level,
    current_department
FROM module8_projections.ups_read_model
WHERE (ml_predictions->>'sepsis_probability')::numeric > 0.7;
```

## Performance Optimization

### Indexes

The schema includes optimized indexes for common query patterns:
- GIN indexes for JSONB columns (flexible nested queries)
- Composite indexes for multi-column filters
- Partial indexes for high-value subsets (e.g., risk alerts)

### Materialized View

Department summaries are cached in a materialized view, refreshed every 1 minute:

```sql
REFRESH MATERIALIZED VIEW CONCURRENTLY module8_projections.department_summary;
```

### Batch Processing

- Events are batched (100 per batch, 5s timeout)
- Uses `psycopg2.extras.execute_batch` for efficient bulk UPSERT
- Connection pooling (2-10 connections)

## Monitoring

### Key Metrics

- **avg_batch_processing_ms**: Total time per batch (target: <100ms)
- **avg_upsert_ms**: Database UPSERT time (target: <20ms)
- **patients_updated**: Unique patients processed
- **events_processed**: Total events consumed

### Alerts

Set up alerts for:
- `avg_upsert_ms > 50ms` (database performance degradation)
- `database.status != "healthy"` (connection issues)
- `kafka.status != "connected"` (consumer lag)

## Testing

### Unit Tests

```bash
cd tests
pytest test_projector.py -v
```

### Integration Test

```bash
# Produce test event to enriched topic
python test_integration.py
```

### Query Performance Test

```bash
# Connect to PostgreSQL
psql -U cardiofit_user -d cardiofit

# Run EXPLAIN ANALYZE
EXPLAIN ANALYZE
SELECT * FROM module8_projections.ups_read_model
WHERE patient_id = 'P12345';

# Should show execution time <10ms
```

## Configuration

Environment variables (from `module8-shared` config):

```bash
# Kafka
KAFKA_BOOTSTRAP_SERVERS=your-cluster.confluent.cloud:9092
KAFKA_SASL_USERNAME=your-username
KAFKA_SASL_PASSWORD=your-password

# PostgreSQL (existing container)
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=cardiofit
POSTGRES_USER=cardiofit_user
POSTGRES_PASSWORD=cardiofit_password
```

## Troubleshooting

### UPSERT Performance Degradation

```sql
-- Check index usage
SELECT schemaname, tablename, indexname, idx_scan
FROM pg_stat_user_indexes
WHERE schemaname = 'module8_projections'
ORDER BY idx_scan ASC;

-- Analyze table statistics
ANALYZE module8_projections.ups_read_model;
```

### Missing Events

```bash
# Check consumer lag
kafka-consumer-groups --bootstrap-server ... \
  --group module8-ups-projector --describe
```

### Connection Pool Exhaustion

Increase pool size in `projector.py`:
```python
self.db_pool = ThreadedConnectionPool(
    minconn=5,     # Increase from 2
    maxconn=20,    # Increase from 10
    ...
)
```

## Future Enhancements

1. **State Change Tracking**: Implement `_track_state_changes()` for audit log
2. **Trend Analysis**: Calculate `vitals_trend` and `trend_confidence`
3. **Demographics Enrichment**: Integrate with Patient Service API
4. **Real-time Webhooks**: Trigger webhooks on critical state changes
5. **Time-series Data**: Add separate table for historical vitals trends

## Related Services

- **Enrichment Service** (port 8053): Produces to `prod.ehr.events.enriched`
- **PostgreSQL Projector** (port 8054): Time-series event storage
- **Patient Service** (port 8003): Demographics source
- **Dashboard API**: Consumes from UPS read model

## License

Copyright 2025 CardioFit Platform
