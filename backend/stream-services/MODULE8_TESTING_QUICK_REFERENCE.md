# Module 8 Testing & Monitoring - Quick Reference Card

Quick commands for daily testing and monitoring operations.

---

## 🚀 Quick Start (5 Minutes)

```bash
# 1. Start infrastructure
./manage-module8-infrastructure.sh start

# 2. Start all projectors
docker-compose -f docker-compose.module8-services.yml up -d

# 3. Run smoke test
./smoke-test-module8.sh

# 4. Check Grafana dashboard
# Open: http://localhost:3000
```

---

## 🧪 Testing Commands

### Smoke Test (30 seconds)
```bash
./smoke-test-module8.sh
# Exit code 0 = PASS, 1 = FAIL
```

### Integration Tests (2-3 minutes)
```bash
# All tests
pytest test-module8-integration.py -v

# Specific test
pytest test-module8-integration.py -v -k "test_enriched_event_fanout"

# With logs
pytest test-module8-integration.py -v --log-cli-level=INFO
```

### Benchmark Tests (10-15 minutes)
```bash
python benchmark-module8.py

# View results
cat BENCHMARK_REPORT.md
open benchmark-results.csv
```

### Load Test (30 minutes)
```bash
# Headless mode
locust -f locustfile-module8.py --headless -u 100 -r 10 -t 30m

# Web UI mode
locust -f locustfile-module8.py
# Then open: http://localhost:8089
```

---

## 📊 Monitoring Commands

### Check Service Health
```bash
# All services
for port in 8050 8051 8052 8053 8054 8055 8056 8057; do
    echo "Port $port: $(curl -s http://localhost:$port/health | jq -r .status)"
done

# Single service
curl http://localhost:8050/health | jq
```

### Check Metrics
```bash
# Prometheus metrics
curl http://localhost:8050/metrics | grep projector_

# Consumer lag
curl http://localhost:8050/metrics | grep consumer_lag

# Throughput
curl http://localhost:8050/metrics | grep messages_processed_total
```

### Check Kafka Consumer Groups
```bash
# List all consumer groups
kafka-consumer-groups.sh --bootstrap-server localhost:9092 --list

# Describe specific group
kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
    --describe --group module8-postgresql-projector
```

### Check Database Counts
```bash
# PostgreSQL
psql -h localhost -U postgres -d clinical_events \
    -c "SELECT COUNT(*) FROM enriched_events;"

# MongoDB
mongosh --eval "db.getSiblingDB('clinical_events').clinical_documents.countDocuments({})"

# Elasticsearch
curl -X GET "http://localhost:9200/clinical_events/_count" | jq

# ClickHouse
clickhouse-client --query "SELECT COUNT(*) FROM clinical_analytics.clinical_events_fact"
```

---

## 🔧 Data Generation Commands

### Generate Test Data
```bash
# To Kafka (100 patients, 50 events each)
python generate-test-data.py --kafka --patients 100 --events-per-patient 50

# To JSON files
python generate-test-data.py --output ./test-data --patients 10 --events-per-patient 20

# Historical data
python generate-test-data.py --kafka \
    --start-date 2024-01-01 \
    --end-date 2024-12-31 \
    --patients 50 \
    --events-per-patient 100
```

---

## 🐛 Troubleshooting Commands

### Restart Services
```bash
# Restart all projectors
docker-compose -f docker-compose.module8-services.yml restart

# Restart single projector
docker-compose -f docker-compose.module8-services.yml restart postgresql-projector

# Restart infrastructure
./manage-module8-infrastructure.sh restart
```

### View Logs
```bash
# All projector logs
docker-compose -f docker-compose.module8-services.yml logs -f

# Single projector logs
docker logs -f module8-postgresql-projector

# Last 100 lines
docker logs --tail 100 module8-postgresql-projector

# Grep for errors
docker logs module8-postgresql-projector 2>&1 | grep -i error
```

### Check Resource Usage
```bash
# Docker stats
docker stats

# Specific container
docker stats module8-postgresql-projector

# System resources
top -o cpu
htop  # If installed
```

### Reset Consumer Offsets
```bash
# Reset to earliest
kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
    --group module8-postgresql-projector \
    --reset-offsets --to-earliest --execute --all-topics

# Reset to latest
kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
    --group module8-postgresql-projector \
    --reset-offsets --to-latest --execute --all-topics
```

---

## 📈 Performance Tuning

### Scale Projectors
```bash
# Scale up to 2 replicas
docker-compose -f docker-compose.module8-services.yml \
    scale postgresql-projector=2

# Scale down to 1 replica
docker-compose -f docker-compose.module8-services.yml \
    scale postgresql-projector=1
```

### Add Kafka Partitions
```bash
# Increase partitions for enriched events topic
kafka-topics.sh --bootstrap-server localhost:9092 \
    --alter --topic prod.ehr.events.enriched --partitions 8
```

### Database Optimization
```bash
# PostgreSQL - Add index
psql -h localhost -U postgres -d clinical_events -c \
    "CREATE INDEX CONCURRENTLY idx_enriched_events_patient_time
     ON enriched_events(patient_id, event_time);"

# MongoDB - Add index
mongosh --eval "
    db.getSiblingDB('clinical_events').clinical_documents.createIndex(
        {patientId: 1, timestamp: -1},
        {background: true}
    )
"

# Elasticsearch - Refresh interval
curl -X PUT "http://localhost:9200/clinical_events/_settings" \
    -H 'Content-Type: application/json' \
    -d '{"index": {"refresh_interval": "30s"}}'
```

---

## 📊 Monitoring Dashboards

### Access URLs
- **Grafana**: http://localhost:3000
- **Prometheus**: http://localhost:9090
- **Locust**: http://localhost:8089 (during load test)
- **Kafka UI**: http://localhost:8080 (if installed)

### Import Grafana Dashboard
```bash
# Via API
curl -X POST http://localhost:3000/api/dashboards/db \
    -H 'Content-Type: application/json' \
    -u admin:admin \
    -d @monitoring/grafana-dashboard-module8.json

# Or via UI: Dashboards -> Import -> Upload JSON file
```

---

## 🎯 Daily Checklist

```bash
# Morning checks (5 minutes)
./smoke-test-module8.sh
curl http://localhost:3000  # Check Grafana
curl http://localhost:9090  # Check Prometheus

# Check for alerts
curl http://localhost:9090/api/v1/alerts | jq

# Review consumer lag
for port in 8050 8051 8052 8053 8054 8055 8056 8057; do
    curl -s http://localhost:$port/metrics | grep consumer_lag
done
```

---

## 🔔 Alert Response

### High Consumer Lag
```bash
# 1. Check if projector is running
docker ps | grep projector

# 2. Check consumer group
kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
    --describe --group module8-postgresql-projector

# 3. Scale up if needed
docker-compose -f docker-compose.module8-services.yml \
    scale postgresql-projector=2

# 4. Check logs
docker logs --tail 100 module8-postgresql-projector
```

### High Error Rate
```bash
# 1. Check DLQ topic
kafka-console-consumer.sh --bootstrap-server localhost:9092 \
    --topic prod.ehr.events.dlq --from-beginning --max-messages 10

# 2. Check error metrics
curl http://localhost:8050/metrics | grep messages_failed

# 3. Review logs
docker logs --tail 100 module8-postgresql-projector 2>&1 | grep -i error

# 4. Check database connectivity
psql -h localhost -U postgres -d clinical_events -c "SELECT 1;"
```

### Service Down
```bash
# 1. Check container status
docker ps -a | grep projector

# 2. Restart service
docker-compose -f docker-compose.module8-services.yml restart postgresql-projector

# 3. Check health
curl http://localhost:8050/health

# 4. View startup logs
docker logs module8-postgresql-projector
```

---

## 📝 Common Queries

### PostgreSQL
```sql
-- Event counts by patient
SELECT patient_id, COUNT(*) as event_count
FROM enriched_events
GROUP BY patient_id
ORDER BY event_count DESC
LIMIT 10;

-- Events in last hour
SELECT COUNT(*) FROM enriched_events
WHERE event_time > NOW() - INTERVAL '1 hour';

-- Average events per minute
SELECT DATE_TRUNC('minute', event_time) as minute,
       COUNT(*) as events
FROM enriched_events
WHERE event_time > NOW() - INTERVAL '1 hour'
GROUP BY minute
ORDER BY minute DESC;
```

### MongoDB
```javascript
// Events by type
db.clinical_documents.aggregate([
    {$group: {_id: "$eventType", count: {$sum: 1}}},
    {$sort: {count: -1}}
])

// Recent events
db.clinical_documents.find().sort({timestamp: -1}).limit(10)
```

### Elasticsearch
```bash
# Count by event type
curl -X GET "http://localhost:9200/clinical_events/_search?size=0" \
    -H 'Content-Type: application/json' -d'
{
    "aggs": {
        "event_types": {
            "terms": {"field": "eventType.keyword"}
        }
    }
}'

# Recent events
curl -X GET "http://localhost:9200/clinical_events/_search" \
    -H 'Content-Type: application/json' -d'
{
    "query": {"match_all": {}},
    "sort": [{"timestamp": {"order": "desc"}}],
    "size": 10
}'
```

---

## 🛠️ Maintenance Commands

### Clean Up Test Data
```bash
# PostgreSQL
psql -h localhost -U postgres -d clinical_events -c \
    "DELETE FROM enriched_events WHERE patient_id LIKE 'test-%' OR patient_id LIKE 'smoke-test-%';"

# MongoDB
mongosh --eval "
    db.getSiblingDB('clinical_events').clinical_documents.deleteMany({
        patientId: {$regex: /^(test-|smoke-test-|load-test-)/}
    })
"

# Elasticsearch
curl -X POST "http://localhost:9200/clinical_events/_delete_by_query" \
    -H 'Content-Type: application/json' -d'
{
    "query": {
        "regexp": {"patientId": "(test-|smoke-test-|load-test-).*"}
    }
}'
```

### Backup Configurations
```bash
# Backup monitoring configs
tar -czf monitoring-backup-$(date +%Y%m%d).tar.gz monitoring/

# Backup test scripts
tar -czf tests-backup-$(date +%Y%m%d).tar.gz \
    test-module8-integration.py \
    benchmark-module8.py \
    locustfile-module8.py \
    generate-test-data.py \
    smoke-test-module8.sh
```

---

## 📞 Support Resources

- **Full Documentation**: `MODULE8_TESTING_MONITORING_COMPLETE.md`
- **Infrastructure Setup**: `MODULE8_INFRASTRUCTURE_README.md`
- **Quick Start**: `MODULE8_QUICKSTART.md`
- **Projector Docs**: `MODULE8_*_PROJECTOR_COMPLETE.md`

---

**Quick Reference Card** - Keep this handy for daily operations!
