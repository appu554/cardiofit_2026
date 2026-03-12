# ClickHouse Projector - Quick Start Guide

## 30-Second Setup

```bash
# 1. Install dependencies
pip install -r requirements.txt

# 2. Initialize ClickHouse
python run.py --init --skip-service

# 3. Run tests
python run.py --test --skip-service

# 4. Start service
python run.py
```

Service ready at: http://localhost:8053

## Verify Installation

```bash
# Health check
curl http://localhost:8053/health

# View metrics
curl http://localhost:8053/metrics

# Test analytics query
curl -X POST http://localhost:8053/analytics/query \
  -H "Content-Type: application/json" \
  -d '{"sql": "SELECT count() FROM clinical_events_fact"}'
```

## Docker Quick Start

```bash
# Start ClickHouse + Projector
docker-compose up -d

# Check logs
docker-compose logs -f clickhouse-projector

# Stop services
docker-compose down
```

## Sample Analytics Queries

### Current Risk Status
```sql
SELECT risk_level, count() as count
FROM clinical_events_fact
WHERE timestamp >= now() - INTERVAL 1 HOUR
GROUP BY risk_level;
```

### Department Performance
```sql
SELECT
    department_id,
    count() as events,
    countIf(risk_level = 'CRITICAL') as critical_events
FROM clinical_events_fact
WHERE timestamp >= today() - INTERVAL 7 DAY
GROUP BY department_id;
```

### High-Risk Patients
```sql
SELECT
    patient_id,
    count() as events,
    max(news2_score) as max_news2
FROM clinical_events_fact
WHERE risk_level IN ('HIGH', 'CRITICAL')
  AND timestamp >= today() - INTERVAL 7 DAY
GROUP BY patient_id
ORDER BY max_news2 DESC
LIMIT 10;
```

## Common Commands

```bash
# Initialize database
python init_clickhouse.py

# Run tests
python test_projector.py

# Start service
python app/main.py

# All-in-one
python run.py --init --test
```

## Configuration

Edit `config.yaml` for custom settings:
- Kafka bootstrap servers
- ClickHouse connection
- Batch size and timeout
- Service port

## Monitoring

```bash
# Service metrics
curl http://localhost:8053/metrics

# Analytics summary
curl http://localhost:8053/analytics/summary

# ClickHouse stats
clickhouse-client --query "SELECT * FROM system.parts WHERE database = 'module8_analytics'"
```

## Troubleshooting

### Can't connect to ClickHouse
```bash
# Check ClickHouse is running
curl http://localhost:8123/ping

# Check credentials in config.yaml
```

### No data in tables
```bash
# Check Kafka topic has data
kafka-console-consumer --topic prod.ehr.events.enriched --bootstrap-server localhost:9092

# Check service logs
tail -f /var/log/clickhouse-projector.log
```

### Slow queries
```sql
-- Check partitions being scanned
EXPLAIN SELECT * FROM clinical_events_fact WHERE timestamp >= today();

-- Add timestamp filter for partition pruning
SELECT * FROM clinical_events_fact WHERE timestamp >= today() - INTERVAL 7 DAY;
```

## Next Steps

1. Review `analytics_examples.sql` for query examples
2. Set up dashboards using ClickHouse queries
3. Configure alerts on high-risk events
4. Scale ClickHouse cluster for production

## Support

- README.md - Complete documentation
- IMPLEMENTATION_SUMMARY.md - Architecture details
- analytics_examples.sql - Query examples
