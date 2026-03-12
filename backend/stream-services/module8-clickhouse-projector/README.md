# ClickHouse Projector Service

OLAP analytics projector that consumes enriched clinical events and writes to ClickHouse for fast analytical queries.

## Overview

The ClickHouse Projector provides columnar storage and OLAP analytics capabilities for the Clinical Synthesis Hub platform. It consumes enriched events from `prod.ehr.events.enriched` and projects them into optimized fact tables for analytics.

## Architecture

```
Kafka Topic                    ClickHouse Tables
┌─────────────────────────┐   ┌──────────────────────────┐
│ prod.ehr.events.enriched│──▶│ clinical_events_fact     │
│                         │   │ - Vitals & scores        │
│ - Enriched vitals       │   │ - Partitioned by month   │
│ - Clinical scores       │   │ - TTL: 2 years          │
│ - ML predictions        │   ├──────────────────────────┤
│ - Risk assessments      │   │ ml_predictions_fact      │
└─────────────────────────┘   │ - ML risk scores         │
                              │ - Partitioned by month   │
                              ├──────────────────────────┤
                              │ alerts_fact              │
                              │ - High-risk alerts       │
                              │ - Response times         │
                              └──────────────────────────┘
```

## Features

### Columnar Storage
- **Fast Aggregations**: Sub-second queries on billions of rows
- **Compression**: 10-100x compression ratios for clinical data
- **Vectorized Processing**: SIMD operations for analytics

### Partitioning Strategy
- **Monthly Partitions**: `PARTITION BY toYYYYMM(timestamp)`
- **Efficient Pruning**: Only scan relevant partitions for time-range queries
- **TTL Management**: Automatic data retention (2 years for clinical events)

### Fact Tables

#### clinical_events_fact
Main clinical events with vitals and scores:
- Patient vitals (heart rate, BP, SpO2, temperature)
- Clinical scores (NEWS2, qSOFA)
- Risk levels (LOW, MODERATE, HIGH, CRITICAL)
- Full event JSON for detailed analysis

#### ml_predictions_fact
ML model predictions:
- Sepsis risk (24h)
- Cardiac risk (7d)
- Readmission risk (30d)

#### alerts_fact
Clinical alerts for high-risk events:
- Alert type and severity
- Department information
- Response time tracking

### Materialized Views

#### daily_patient_stats_mv
Daily patient-level aggregations:
- Event counts
- Average vitals
- High-risk event counts

#### hourly_department_stats_mv
Hourly department-level metrics:
- Event volumes
- Critical/high-risk event counts
- Average NEWS2 scores

## Installation

```bash
pip install -r requirements.txt
```

## Configuration

Edit `config.yaml`:

```yaml
kafka:
  bootstrap_servers: "localhost:9092"
  topic: "prod.ehr.events.enriched"
  group_id: "module8-clickhouse-projector-v1"
  max_poll_records: 500  # Larger batches for analytics

clickhouse:
  host: "clickhouse"
  port: 9000  # Native protocol
  database: "module8_analytics"
  user: "module8_user"
  password: "module8_password"

batch:
  size: 500  # Large batches for analytics workloads
  timeout: 30  # 30 seconds
```

## Running the Service

### Standalone
```bash
python app/main.py
```

### Docker Compose
```bash
docker-compose up -d
```

This starts:
- ClickHouse server (ports 8123, 9000)
- ClickHouse projector (port 8053)

## API Endpoints

### Health Check
```bash
curl http://localhost:8053/health
```

### Metrics
```bash
curl http://localhost:8053/metrics
```

Response:
```json
{
  "projector_metrics": {
    "processed": 15000,
    "errors": 2,
    "last_processed": "2025-11-15T10:30:00Z"
  },
  "analytics_summary": {
    "total_events": 15000,
    "high_risk_events": 450,
    "ml_predictions": 14800,
    "alerts": 450,
    "storage_info": {
      "clinical_events_fact": {"size": "2.3 MB", "rows": 15000},
      "ml_predictions_fact": {"size": "890 KB", "rows": 14800},
      "alerts_fact": {"size": "45 KB", "rows": 450}
    }
  }
}
```

### Analytics Summary
```bash
curl http://localhost:8053/analytics/summary
```

### Ad-Hoc Queries
```bash
curl -X POST http://localhost:8053/analytics/query \
  -H "Content-Type: application/json" \
  -d '{
    "sql": "SELECT department_id, count() as events, avg(news2_score) as avg_news2 FROM clinical_events_fact WHERE timestamp >= today() - INTERVAL 7 DAY GROUP BY department_id"
  }'
```

## Example Analytics Queries

### Daily Patient Event Trends
```sql
SELECT
    toDate(timestamp) as day,
    count() as total_events,
    countIf(risk_level = 'CRITICAL') as critical_events,
    avg(news2_score) as avg_news2
FROM clinical_events_fact
WHERE timestamp >= today() - INTERVAL 30 DAY
GROUP BY day
ORDER BY day DESC;
```

### Department Risk Distribution
```sql
SELECT
    department_id,
    risk_level,
    count() as event_count,
    count() * 100.0 / sum(count()) OVER (PARTITION BY department_id) as percentage
FROM clinical_events_fact
WHERE timestamp >= today() - INTERVAL 7 DAY
GROUP BY department_id, risk_level
ORDER BY department_id, event_count DESC;
```

### ML Prediction Accuracy (requires labels)
```sql
SELECT
    toStartOfHour(timestamp) as hour,
    avg(sepsis_risk_24h) as avg_sepsis_risk,
    count() as prediction_count
FROM ml_predictions_fact
WHERE timestamp >= today() - INTERVAL 1 DAY
GROUP BY hour
ORDER BY hour DESC;
```

### Alert Response Time Analysis
```sql
SELECT
    severity,
    department_id,
    count() as alert_count,
    avg(response_time_seconds) as avg_response_time,
    quantile(0.95)(response_time_seconds) as p95_response_time
FROM alerts_fact
WHERE timestamp >= today() - INTERVAL 7 DAY
  AND response_time_seconds IS NOT NULL
GROUP BY severity, department_id
ORDER BY severity DESC, avg_response_time DESC;
```

## Performance Characteristics

### Insert Performance
- **Batch Size**: 500 events per batch
- **Throughput**: ~10,000 events/second (single node)
- **Latency**: 30-second batching window

### Query Performance
- **Simple Aggregations**: <100ms (millions of rows)
- **Complex Joins**: <1s (billions of rows)
- **Materialized Views**: Pre-aggregated, instant response

### Storage Efficiency
- **Compression**: 80-90% for clinical data
- **Partitioning**: Only scan relevant months
- **TTL**: Automatic data cleanup

## Monitoring

### Key Metrics
- **Throughput**: Events processed per second
- **Lag**: Consumer offset lag
- **Storage**: Table sizes and row counts
- **Query Performance**: Query execution times

### ClickHouse System Tables
```sql
-- Table statistics
SELECT * FROM system.parts WHERE database = 'module8_analytics';

-- Query log
SELECT * FROM system.query_log ORDER BY event_time DESC LIMIT 10;

-- Metrics
SELECT * FROM system.metrics;
```

## Scaling Considerations

### Horizontal Scaling
- **ClickHouse Cluster**: Distribute data across nodes
- **Replication**: Add replicas for read scaling
- **Sharding**: Partition by patient_id or department_id

### Vertical Scaling
- **Memory**: 64GB+ for large datasets
- **CPU**: More cores = faster parallel aggregations
- **Storage**: SSD recommended for query performance

## Troubleshooting

### High Consumer Lag
- Increase batch size (500 → 1000)
- Add more ClickHouse nodes
- Optimize table schemas (remove unnecessary columns)

### Slow Queries
- Check partition pruning (use timestamp filters)
- Add materialized views for common queries
- Optimize ORDER BY columns

### Storage Growth
- Verify TTL is working (check system.parts)
- Increase compression settings
- Consider aggregating older data

## References

- [ClickHouse Documentation](https://clickhouse.com/docs)
- [MergeTree Engine](https://clickhouse.com/docs/en/engines/table-engines/mergetree-family/mergetree)
- [Partitioning](https://clickhouse.com/docs/en/engines/table-engines/mergetree-family/custom-partitioning-key)
- [TTL](https://clickhouse.com/docs/en/engines/table-engines/mergetree-family/mergetree#table_engine-mergetree-ttl)
