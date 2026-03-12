# PostgreSQL Projector Service - Quick Start

## Service Created Successfully

### Database Schema Initialized
- **Schema**: `module8_projections` in `cardiofit_analytics` database
- **Tables**: 4 tables created and indexed
- **Views**: 3 materialized views for common queries
- **Functions**: 2 helper functions

### Service Components
```
module8-postgresql-projector/
├── app/
│   ├── main.py                    # FastAPI application
│   ├── config.py                  # Configuration
│   ├── services/
│   │   ├── kafka_consumer.py      # Consumer service wrapper
│   │   └── projector.py           # PostgreSQL projection logic
│   └── models/
│       └── schemas.py             # Pydantic response models
├── schema/
│   └── init.sql                   # Database initialization (COMPLETED)
├── Dockerfile                      # Production container
├── requirements.txt                # Python dependencies
├── .env.example                    # Environment template
├── test_projector.py              # Database test script
└── README.md                       # Full documentation
```

## Start the Service

### Option 1: Local Development
```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/stream-services/module8-postgresql-projector

# 1. Install dependencies
pip install -r requirements.txt

# 2. Create environment file
cp .env.example .env
# Edit .env with your Kafka credentials

# 3. Start service
python3 -m app.main
```

Service will start on `http://localhost:8050`

### Option 2: Docker
```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/stream-services/module8-postgresql-projector

# 1. Build image
docker build -t postgresql-projector:latest .

# 2. Run container
docker run -d \
  --name postgresql-projector \
  -e KAFKA_API_KEY=your-key \
  -e KAFKA_API_SECRET=your-secret \
  -p 8050:8050 \
  -p 9090:9090 \
  postgresql-projector:latest
```

## Verify Service

### Check Health
```bash
curl http://localhost:8050/health
```

Expected response:
```json
{
  "status": "healthy",
  "timestamp": "2025-11-15T20:30:00Z",
  "service": "postgresql-projector",
  "version": "1.0.0"
}
```

### Check Status
```bash
curl http://localhost:8050/status
```

### View Metrics (Prometheus)
```bash
curl http://localhost:8050/metrics
```

## Database Access

### Connection Details
- **Host**: 172.21.0.4 (Docker container a2f55d83b1fa)
- **Port**: 5432
- **Database**: cardiofit_analytics
- **User**: cardiofit
- **Schema**: module8_projections

### Query Data
```bash
# Connect to database
docker exec -it a2f55d83b1fa psql -U cardiofit -d cardiofit_analytics

# View tables
\dt module8_projections.*

# Query events
SELECT COUNT(*) FROM module8_projections.enriched_events;

# View latest vitals
SELECT * FROM module8_projections.latest_patient_vitals LIMIT 10;

# View high-risk patients
SELECT * FROM module8_projections.high_risk_patients;
```

## Database Schema Details

### Table 1: enriched_events
```sql
-- Raw event storage with full JSONB data
event_id (PK), patient_id, timestamp, event_type, event_data (JSONB)
-- Indexes: patient_id+timestamp, event_type, JSONB GIN
```

### Table 2: patient_vitals
```sql
-- Normalized vital signs for VITAL_SIGNS events
id, event_id (FK), patient_id, timestamp,
heart_rate, bp_systolic, bp_diastolic, spo2, temperature_celsius
-- Indexes: patient_id+timestamp
```

### Table 3: clinical_scores
```sql
-- Risk scores and ML predictions
id, event_id (FK), patient_id, timestamp,
news2_score, qsofa_score, risk_level,
sepsis_risk_24h, cardiac_risk_7d, readmission_risk_30d
-- Indexes: patient_id+timestamp, risk_level
```

### Table 4: event_metadata
```sql
-- Searchable event attributes for fast filtering
event_id (PK, FK), patient_id, encounter_id, department_id,
device_id, timestamp, event_type
-- Indexes: All searchable fields + patient_id+timestamp
```

### Views
- **latest_patient_vitals**: Latest vitals per patient (DISTINCT ON patient_id)
- **high_risk_patients**: Patients with HIGH/CRITICAL risk levels
- **complete_event_detail**: Join all 4 tables for comprehensive event view

## Configuration

### Required Environment Variables
```bash
# Kafka
KAFKA_BOOTSTRAP_SERVERS=pkc-p11w6.us-east-1.aws.confluent.cloud:9092
KAFKA_API_KEY=your-api-key
KAFKA_API_SECRET=your-api-secret

# PostgreSQL (defaults work for existing container)
POSTGRES_HOST=172.21.0.4
POSTGRES_PORT=5432
POSTGRES_DB=cardiofit_analytics
POSTGRES_USER=cardiofit
POSTGRES_PASSWORD=cardiofit_analytics_pass
```

### Optional Configuration
```bash
BATCH_SIZE=100                    # Messages per batch
BATCH_TIMEOUT_SECONDS=5.0         # Batch timeout
SERVICE_PORT=8050                 # FastAPI port
LOG_LEVEL=INFO                    # Logging level
```

## Monitoring

### Prometheus Metrics Available
- `projector_messages_consumed_total`: Total messages from Kafka
- `projector_messages_processed_total`: Successfully written to PostgreSQL
- `projector_messages_failed_total`: Failed writes (sent to DLQ)
- `projector_batches_processed_total`: Total batches processed
- `projector_consumer_lag`: Current Kafka consumer lag

### Health Monitoring
```bash
# Continuous health check
watch -n 5 'curl -s http://localhost:8050/health | jq'

# Check consumer lag
curl -s http://localhost:8050/status | jq '.metrics.consumer_lag'

# View processing rate
curl -s http://localhost:8050/status | jq '.metrics'
```

## Troubleshooting

### Service won't start
1. Check Kafka credentials in .env
2. Verify PostgreSQL connection: `docker ps | grep postgres`
3. Check logs: `docker logs postgresql-projector`

### Consumer lag increasing
1. Increase BATCH_SIZE: `export BATCH_SIZE=500`
2. Check PostgreSQL performance
3. Verify network connectivity to Kafka

### Database connection issues
```bash
# Test connection
docker exec a2f55d83b1fa psql -U cardiofit -d cardiofit_analytics -c "SELECT 1;"

# Check tables exist
docker exec a2f55d83b1fa psql -U cardiofit -d cardiofit_analytics -c "\dt module8_projections.*"

# Recreate schema
docker exec -i a2f55d83b1fa psql -U cardiofit -d cardiofit_analytics < schema/init.sql
```

## Integration with Module 8 Pipeline

### Data Flow
```
Kafka Topic: prod.ehr.events.enriched
    ↓
PostgreSQL Projector Service (port 8050)
    ↓
PostgreSQL Database (module8_projections schema)
    ├── enriched_events
    ├── patient_vitals
    ├── clinical_scores
    └── event_metadata
```

### Consumed Messages
- **Topic**: `prod.ehr.events.enriched`
- **Consumer Group**: `module8-postgresql-projector`
- **Message Format**: `EnrichedClinicalEvent` (Pydantic model from module8-shared)

### Failed Messages
- **DLQ Topic**: `prod.ehr.dlq.postgresql`
- **Error Handling**: Automatic retry with exponential backoff
- **Monitoring**: Track via `projector_messages_failed_total` metric

## Next Steps

1. **Start Service**: Follow Option 1 or 2 above
2. **Verify Health**: Check `/health` and `/status` endpoints
3. **Monitor Metrics**: Set up Prometheus scraping
4. **Query Data**: Use provided SQL queries to access projected data
5. **Integration**: Connect BI tools or analytics services to PostgreSQL

## Performance Tuning

### Batch Processing
- Default: 100 messages per batch, 5 second timeout
- High throughput: Increase to 500-1000 messages
- Low latency: Decrease to 50-100 messages

### PostgreSQL Optimization
```sql
-- Run after bulk loads
ANALYZE module8_projections.enriched_events;
ANALYZE module8_projections.patient_vitals;
ANALYZE module8_projections.clinical_scores;
ANALYZE module8_projections.event_metadata;

-- Reindex periodically
REINDEX SCHEMA module8_projections;
```

### Index Performance
All tables have optimized indexes:
- Patient lookups: `idx_*_patient_timestamp`
- Time-series queries: `idx_*_timestamp`
- JSONB queries: GIN index on `event_data`
- Join performance: Foreign keys with indexes

## Support

- **Documentation**: See README.md for complete details
- **Example Queries**: Check schema/init.sql for helper functions
- **Testing**: Run test_projector.py for database validation
- **Logs**: Check structured JSON logs for debugging

---

**Service Status**: ✅ Ready for deployment
**Database Schema**: ✅ Initialized successfully
**Tables Created**: 4 tables + 3 views
**Service URL**: http://localhost:8050
