# CDC Connector Quick Reference

Fast reference guide for common operations and troubleshooting.

---

## Deployment (4 Commands)

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/kafka/cdc-connectors/scripts

# 1. Verify infrastructure (30 seconds)
./verify-infrastructure.sh

# 2. Setup PostgreSQL (2 minutes)
./setup-postgresql-cdc.sh setup

# 3. Deploy connectors (3 minutes)
./deploy-all-cdc-connectors.sh deploy

# 4. Validate deployment (1 minute)
./verify-cdc-deployment.sh full
```

---

## Health Checks

```bash
# Quick status check (30 seconds)
./verify-cdc-deployment.sh quick

# Full validation (2 minutes)
./verify-cdc-deployment.sh full

# Single connector
./verify-cdc-deployment.sh connector kb1-medications-cdc

# Check all via API
curl http://localhost:8083/connectors | jq
```

---

## Emergency Operations

```bash
# PAUSE ALL (immediate)
./rollback-cdc.sh pause-all

# RESUME ALL
./rollback-cdc.sh resume-all

# RESTART ALL
./rollback-cdc.sh restart-all

# BACKUP configurations
./rollback-cdc.sh backup

# ROLLBACK single connector
./rollback-cdc.sh rollback kb1-medications-cdc

# ROLLBACK everything (WARNING: DESTRUCTIVE)
./rollback-cdc.sh rollback-all
```

---

## Connector Operations

```bash
# Get connector status
curl http://localhost:8083/connectors/kb1-medications-cdc/status | jq

# Restart connector
curl -X POST http://localhost:8083/connectors/kb1-medications-cdc/restart

# Pause connector
curl -X PUT http://localhost:8083/connectors/kb1-medications-cdc/pause

# Resume connector
curl -X PUT http://localhost:8083/connectors/kb1-medications-cdc/resume

# Delete connector
curl -X DELETE http://localhost:8083/connectors/kb1-medications-cdc
```

---

## PostgreSQL Diagnostics

```bash
# Check replication slots
PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d medications_db \
  -c "SELECT * FROM pg_replication_slots WHERE slot_name LIKE 'debezium%';"

# Check slot lag
PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d medications_db \
  -c "SELECT slot_name, pg_size_pretty(pg_wal_lsn_diff(pg_current_wal_lsn(), confirmed_flush_lsn)) AS lag FROM pg_replication_slots;"

# Check publications
PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d medications_db \
  -c "SELECT * FROM pg_publication WHERE pubname LIKE 'dbz_publication%';"

# Check WAL files
PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d medications_db \
  -c "SELECT COUNT(*) as wal_files, pg_size_pretty(SUM(size)) as total_size FROM pg_ls_waldir();"
```

---

## Kafka Operations

```bash
KAFKA_CONTAINER="3c7ffa06d20db1674c249c3ec2dda1bad58a1e92036abed8f504d1fdb0978754"

# List CDC topics
docker exec $KAFKA_CONTAINER kafka-topics \
  --bootstrap-server localhost:9092 --list | grep cdc

# Describe topic
docker exec $KAFKA_CONTAINER kafka-topics \
  --bootstrap-server localhost:9092 \
  --describe \
  --topic cdc.medications_db.public.medications

# Consume messages
docker exec $KAFKA_CONTAINER kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic cdc.medications_db.public.medications \
  --from-beginning \
  --max-messages 10

# Check consumer groups
docker exec $KAFKA_CONTAINER kafka-consumer-groups \
  --bootstrap-server localhost:9092 --list
```

---

## Monitoring Access

| Service | URL | Credentials |
|---------|-----|-------------|
| Grafana | http://localhost:3000 | admin/admin |
| Prometheus | http://localhost:9090 | None |
| Alertmanager | http://localhost:9093 | None |
| Kafka Connect | http://localhost:8083 | None |

---

## Common Issues

### Connector FAILED

```bash
# 1. Check status
curl http://localhost:8083/connectors/kb1-medications-cdc/status | jq

# 2. Check logs
docker logs kafka-connect-container | grep kb1-medications-cdc

# 3. Restart
curl -X POST http://localhost:8083/connectors/kb1-medications-cdc/restart

# 4. If restart fails, redeploy
AUTO_REPLACE=true ./deploy-all-cdc-connectors.sh deploy
```

### High Replication Lag

```bash
# 1. Check lag
./verify-cdc-deployment.sh quick

# 2. Check PostgreSQL slot
PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d medications_db \
  -c "SELECT slot_name, pg_size_pretty(pg_wal_lsn_diff(pg_current_wal_lsn(), confirmed_flush_lsn)) FROM pg_replication_slots;"

# 3. Increase connector throughput
# Edit connector config: increase tasks.max, max.batch.size
```

### No Topics Created

```bash
# 1. Verify connector is running
curl http://localhost:8083/connectors/kb1-medications-cdc/status | jq

# 2. Make a change in database
PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d medications_db \
  -c "UPDATE medications SET updated_at=NOW() WHERE id=1;"

# 3. Wait 5 seconds and check topics
docker exec $KAFKA_CONTAINER kafka-topics \
  --bootstrap-server localhost:9092 --list | grep cdc
```

### WAL Accumulation

```bash
# 1. Check WAL usage
PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d medications_db \
  -c "SELECT COUNT(*) FROM pg_ls_waldir();"

# 2. Check if connector is paused
curl http://localhost:8083/connectors/kb1-medications-cdc/status | jq '.connector.state'

# 3. Resume if paused
./rollback-cdc.sh resume-all

# 4. If critical, advance slot (CAUTION: DATA LOSS)
# PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d medications_db \
#   -c "SELECT pg_replication_slot_advance('debezium_kb1_medications_cdc', pg_current_wal_lsn());"
```

---

## Database Ports Reference

| KB | Port | Database | Connector |
|----|------|----------|-----------|
| KB1 | 5432 | medications_db | kb1-medications-cdc |
| KB2 | 5433 | kb2_scheduling_db | kb2-scheduling-cdc |
| KB3 | 5434 | kb3_encounter_db | kb3-encounter-cdc |
| KB6 | 5435 | kb6_drug_rules_db | kb6-drug-rules-cdc |
| KB7 | 5436 | kb7_guideline_evidence_db | kb7-guideline-evidence-cdc |

---

## Alert Severity

**Critical (P1):**
- Connector down >2 minutes
- Replication lag >15 minutes
- WAL accumulation >1GB

**Warning (P2):**
- Connector paused >5 minutes
- Replication lag >5 minutes
- Error rate >0.1/sec

**Info (P3):**
- Schema evolution
- Connector restarted

---

## Performance Benchmarks

**Normal Operation:**
- Replication lag: <5 seconds
- Throughput: 1,000-5,000 events/sec per connector
- Latency p99: <500ms

**Thresholds:**
- Lag Warning: 5 minutes
- Lag Critical: 15 minutes
- WAL Warning: 100MB
- WAL Critical: 1GB

---

## Escalation

**L1:** DevOps on-call → Slack #cdc-alerts
**L2:** Data Engineering → data-engineering@cardiofit.com
**L3:** Platform Engineering Lead → PagerDuty

---

## File Locations

**Scripts:** `/backend/shared-infrastructure/kafka/cdc-connectors/scripts/`
**Configs:** `/backend/shared-infrastructure/kafka/cdc-connectors/configs/`
**Monitoring:** `/backend/shared-infrastructure/kafka/cdc-connectors/monitoring/`
**Docs:** `/backend/shared-infrastructure/kafka/cdc-connectors/docs/`

---

**Version:** 1.0
**Last Updated:** 2025-11-20
