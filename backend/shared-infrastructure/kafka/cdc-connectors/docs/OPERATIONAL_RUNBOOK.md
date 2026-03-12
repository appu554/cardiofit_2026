# CDC Connector Operational Runbook

## Overview

This runbook provides operational procedures for managing CDC connectors in the CardioFit platform. It covers deployment, monitoring, troubleshooting, and disaster recovery scenarios.

## Table of Contents

1. [Deployment Procedures](#deployment-procedures)
2. [Monitoring and Alerting](#monitoring-and-alerting)
3. [Troubleshooting Guide](#troubleshooting-guide)
4. [Disaster Recovery](#disaster-recovery)
5. [Performance Tuning](#performance-tuning)
6. [Maintenance Windows](#maintenance-windows)

---

## Deployment Procedures

### Initial Deployment

**Prerequisites:**
- Kafka cluster running and accessible
- Kafka Connect cluster deployed
- PostgreSQL instances configured with WAL level = logical
- Debezium PostgreSQL connector plugin installed
- Network connectivity between all components

**Deployment Steps:**

```bash
# 1. Verify infrastructure readiness
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/kafka/cdc-connectors/scripts
chmod +x *.sh
./verify-infrastructure.sh

# 2. Setup PostgreSQL for CDC
./setup-postgresql-cdc.sh setup

# 3. Deploy all connectors
./deploy-all-cdc-connectors.sh deploy

# 4. Verify deployment
./verify-cdc-deployment.sh full
```

**Expected Results:**
- All 5 connectors (KB1, KB2, KB3, KB6, KB7) in RUNNING state
- PostgreSQL replication slots active
- Kafka topics created for each database table
- No errors in connector logs

**Rollback Plan:**
If deployment fails:
```bash
./rollback-cdc.sh pause-all     # Emergency pause
./rollback-cdc.sh rollback-all  # Full rollback (destructive)
```

---

## Monitoring and Alerting

### Key Metrics to Monitor

**Connector Health:**
- Connector state (RUNNING/FAILED/PAUSED)
- Task state per connector
- Connector restart count
- Error rate per connector

**Replication Lag:**
- Time-based lag (seconds behind source)
- Offset-based lag (number of events behind)
- WAL lag on PostgreSQL side

**Throughput:**
- Events per second per connector
- Topic message rate
- Bytes processed per second

**Resource Utilization:**
- Kafka Connect JVM heap usage
- PostgreSQL replication slot disk usage
- Kafka topic disk usage

### Alert Severity Levels

**Critical (P1) - Immediate Action Required:**
- Connector FAILED state for >2 minutes
- Replication lag >15 minutes
- PostgreSQL replication slot WAL accumulation >1GB
- Kafka Connect cluster degraded

**Warning (P2) - Action Required Soon:**
- Connector PAUSED state for >5 minutes
- Replication lag >5 minutes
- High error rate (>0.1 errors/sec)
- PostgreSQL slot inactive for >5 minutes

**Info (P3) - Informational:**
- Schema evolution detected
- Connector restarted
- Snapshot completed

### Accessing Monitoring Dashboards

**Grafana Dashboard:**
```
URL: http://grafana:3000/d/cdc-connector-monitoring
Credentials: admin / [retrieve from secrets]
```

**Prometheus Alerts:**
```
URL: http://prometheus:9090/alerts
```

**Kafka Connect REST API:**
```bash
# List all connectors
curl http://localhost:8083/connectors

# Get connector status
curl http://localhost:8083/connectors/kb1-medications-cdc/status

# Get connector config
curl http://localhost:8083/connectors/kb1-medications-cdc/config
```

---

## Troubleshooting Guide

### Connector in FAILED State

**Symptoms:**
- Connector state shows FAILED
- Tasks are not processing events
- Error trace visible in status

**Diagnosis:**
```bash
# Check connector status
curl http://localhost:8083/connectors/kb1-medications-cdc/status | jq

# Check connector logs
docker logs kafka-connect-container | grep kb1-medications-cdc

# Check PostgreSQL replication slot
PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d medications_db \
  -c "SELECT * FROM pg_replication_slots WHERE slot_name='debezium_kb1_medications_cdc';"
```

**Common Causes & Solutions:**

1. **PostgreSQL Connection Lost**
   - Check PostgreSQL is running
   - Verify network connectivity
   - Check credentials in connector config
   - Solution: Restart connector after fixing connectivity

2. **Replication Slot Deleted**
   - Check if slot exists in pg_replication_slots
   - Solution: Recreate slot and restart connector
   ```bash
   ./setup-postgresql-cdc.sh setup
   ./deploy-all-cdc-connectors.sh deploy
   ```

3. **Schema Change Without Compatibility**
   - Check connector logs for schema errors
   - Solution: Update connector config with schema handling:
   ```json
   {
     "schema.history.internal.kafka.bootstrap.servers": "kafka:9092",
     "schema.history.internal.kafka.topic": "schema-changes.kb1"
   }
   ```

**Resolution:**
```bash
# Attempt automatic restart
./rollback-cdc.sh restart-all

# If restart fails, full redeployment
AUTO_REPLACE=true ./deploy-all-cdc-connectors.sh deploy
```

---

### High Replication Lag

**Symptoms:**
- Replication lag >5 minutes
- Events not appearing in Kafka topics in real-time
- PostgreSQL WAL files accumulating

**Diagnosis:**
```bash
# Check current lag
./verify-cdc-deployment.sh quick

# Check PostgreSQL replication slot lag
PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d medications_db \
  -c "SELECT slot_name, pg_size_pretty(pg_wal_lsn_diff(pg_current_wal_lsn(), confirmed_flush_lsn)) AS lag FROM pg_replication_slots;"

# Check Kafka Connect worker resources
docker stats kafka-connect-container
```

**Common Causes & Solutions:**

1. **High Write Volume**
   - Increase connector parallelism:
   ```json
   {
     "tasks.max": "4",
     "max.batch.size": "4096",
     "poll.interval.ms": "500"
   }
   ```

2. **Slow Kafka Broker**
   - Check Kafka broker metrics
   - Increase Kafka topic partitions
   - Solution: Scale Kafka cluster

3. **Network Bandwidth Limitation**
   - Check network throughput
   - Solution: Increase network capacity or compress data:
   ```json
   {
     "compression.type": "snappy"
   }
   ```

4. **Kafka Connect Resource Starvation**
   - Increase Kafka Connect worker resources
   - Solution: Scale horizontally or increase JVM heap:
   ```bash
   KAFKA_HEAP_OPTS="-Xmx4G -Xms4G"
   ```

---

### Connector Not Capturing Changes

**Symptoms:**
- Connector running but no events in Kafka topics
- Database changes not reflected in topics

**Diagnosis:**
```bash
# Check if connector is actually running
curl http://localhost:8083/connectors/kb1-medications-cdc/status

# Check if PostgreSQL publication exists
PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d medications_db \
  -c "SELECT * FROM pg_publication WHERE pubname='dbz_publication_kb1_medications_cdc';"

# Check if tables are included in publication
PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d medications_db \
  -c "SELECT * FROM pg_publication_tables WHERE pubname='dbz_publication_kb1_medications_cdc';"

# Make a test change to verify
PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d medications_db \
  -c "UPDATE medications SET updated_at=NOW() WHERE id=1;"

# Check Kafka topic for new events
docker exec kafka-container kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic cdc.medications_db.public.medications \
  --from-beginning --max-messages 1
```

**Common Causes & Solutions:**

1. **Publication Not Covering Tables**
   - Recreate publication for all tables:
   ```bash
   ./setup-postgresql-cdc.sh setup
   ```

2. **Replication Slot Not Active**
   - Restart connector to activate slot
   - Check slot is not held by another process

3. **Table Filter Excluding Tables**
   - Review connector config table.include.list
   - Update to include desired tables

---

### PostgreSQL Replication Slot Disk Accumulation

**Symptoms:**
- PostgreSQL disk usage increasing rapidly
- WAL files not being removed
- Alert: PostgreSQLReplicationSlotWALAccumulation

**Diagnosis:**
```bash
# Check WAL accumulation
PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d medications_db \
  -c "SELECT slot_name, active, pg_size_pretty(pg_wal_lsn_diff(pg_current_wal_lsn(), restart_lsn)) AS retained_wal FROM pg_replication_slots;"

# Check disk usage
df -h /var/lib/postgresql/data
```

**Common Causes & Solutions:**

1. **Connector Paused/Stopped**
   - Resume connector to consume WAL:
   ```bash
   ./rollback-cdc.sh resume-all
   ```

2. **Connector Lag Behind**
   - Increase connector throughput (see High Replication Lag)
   - Consider increasing max.queue.size

3. **Orphaned Replication Slot**
   - If connector deleted but slot remains:
   ```bash
   # DANGEROUS - Only if connector truly deleted
   PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d medications_db \
     -c "SELECT pg_drop_replication_slot('debezium_kb1_medications_cdc');"
   ```

**Emergency Mitigation:**
If disk space critical (<10% free):
1. Pause connector temporarily
2. Advance replication slot manually (CAUTION: causes data loss):
```sql
-- Advance slot to current position (loses intervening events)
SELECT pg_replication_slot_advance('debezium_kb1_medications_cdc', pg_current_wal_lsn());
```
3. Resume connector
4. Accept data loss and re-snapshot if necessary

---

## Disaster Recovery

### RTO/RPO Objectives

**Recovery Time Objective (RTO):** 15 minutes
**Recovery Point Objective (RPO):** 5 minutes

### Scenario 1: Single Connector Failure

**Detection:**
- Alert: CDCConnectorDown
- Monitoring dashboard shows FAILED state

**Recovery Steps:**
```bash
# 1. Backup current configuration
./rollback-cdc.sh backup

# 2. Attempt restart
curl -X POST http://localhost:8083/connectors/kb1-medications-cdc/restart

# 3. If restart fails, full redeployment
AUTO_REPLACE=true ./deploy-all-cdc-connectors.sh deploy

# 4. Verify recovery
./verify-cdc-deployment.sh connector kb1-medications-cdc
```

**Expected RTO:** 5 minutes
**Expected RPO:** 0 (no data loss if replication slot intact)

---

### Scenario 2: Kafka Connect Cluster Failure

**Detection:**
- All connectors unreachable
- Kafka Connect REST API not responding

**Recovery Steps:**
```bash
# 1. Restart Kafka Connect cluster
docker restart kafka-connect-container

# 2. Wait for cluster to stabilize (2-3 minutes)
sleep 120

# 3. Verify connectors auto-recover
./verify-cdc-deployment.sh quick

# 4. If connectors don't auto-recover, redeploy
./deploy-all-cdc-connectors.sh deploy

# 5. Full verification
./verify-cdc-deployment.sh full
```

**Expected RTO:** 10 minutes
**Expected RPO:** 0 (connectors resume from last checkpoint)

---

### Scenario 3: PostgreSQL Database Failure

**Detection:**
- Connector errors indicating database unreachable
- PostgreSQL monitoring shows instance down

**Recovery Steps:**
```bash
# 1. Pause affected connectors
./rollback-cdc.sh pause-all

# 2. Restore PostgreSQL from backup (outside scope)
# Follow PostgreSQL disaster recovery procedures

# 3. After PostgreSQL recovery, verify replication slot
PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d medications_db \
  -c "SELECT * FROM pg_replication_slots;"

# 4. If slot missing, recreate CDC infrastructure
./setup-postgresql-cdc.sh setup

# 5. Resume connectors
./rollback-cdc.sh resume-all

# 6. Verify recovery
./verify-cdc-deployment.sh full
```

**Expected RTO:** Depends on PostgreSQL recovery time + 15 minutes
**Expected RPO:** Depends on PostgreSQL backup strategy

---

### Scenario 4: Complete System Failure

**Recovery Steps:**
```bash
# 1. Restore Kafka cluster (outside scope)
# 2. Restore Kafka Connect cluster (outside scope)
# 3. Restore PostgreSQL instances (outside scope)

# 4. Verify infrastructure
./verify-infrastructure.sh

# 5. Setup PostgreSQL CDC
./setup-postgresql-cdc.sh setup

# 6. Deploy all connectors
./deploy-all-cdc-connectors.sh deploy

# 7. Full verification
./verify-cdc-deployment.sh full

# 8. Monitor for 24 hours to ensure stability
```

**Expected RTO:** 1-2 hours (depends on infrastructure recovery)
**Expected RPO:** 5-15 minutes (depends on last checkpoint)

---

## Performance Tuning

### Connector Configuration Tuning

**High Throughput Scenario:**
```json
{
  "tasks.max": "4",
  "max.batch.size": "4096",
  "max.queue.size": "16384",
  "poll.interval.ms": "500",
  "snapshot.fetch.size": "10240"
}
```

**Low Latency Scenario:**
```json
{
  "tasks.max": "2",
  "max.batch.size": "1024",
  "poll.interval.ms": "100",
  "heartbeat.interval.ms": "1000"
}
```

**Resource Constrained Scenario:**
```json
{
  "tasks.max": "1",
  "max.batch.size": "2048",
  "max.queue.size": "8192",
  "snapshot.fetch.size": "2048"
}
```

### Kafka Topic Configuration

**Recommended Settings:**
```bash
# Create topic with optimal settings
docker exec kafka-container kafka-topics \
  --bootstrap-server localhost:9092 \
  --create \
  --topic cdc.medications_db.public.medications \
  --partitions 4 \
  --replication-factor 3 \
  --config retention.ms=604800000 \
  --config compression.type=snappy \
  --config segment.bytes=536870912
```

### PostgreSQL Configuration

**Recommended Settings:**
```conf
# postgresql.conf
wal_level = logical
max_wal_senders = 10
max_replication_slots = 10
wal_sender_timeout = 60s
wal_keep_size = 4GB  # PostgreSQL 13+
checkpoint_timeout = 15min
max_wal_size = 4GB
```

---

## Maintenance Windows

### Planned Connector Restart

**When to Perform:**
- After configuration changes
- After connector plugin updates
- Quarterly maintenance

**Procedure:**
```bash
# 1. Announce maintenance window
# 2. Backup configurations
./rollback-cdc.sh backup

# 3. Pause connectors
./rollback-cdc.sh pause-all

# 4. Perform maintenance (config changes, updates, etc.)

# 5. Resume connectors one at a time
for connector in kb1-medications-cdc kb2-scheduling-cdc kb3-encounter-cdc kb6-drug-rules-cdc kb7-guideline-evidence-cdc; do
  curl -X PUT http://localhost:8083/connectors/$connector/resume
  sleep 30
  ./verify-cdc-deployment.sh connector $connector
done

# 6. Full verification
./verify-cdc-deployment.sh full
```

**Expected Downtime:** 15-30 minutes

---

### PostgreSQL Maintenance

**Vacuum Replication Slots:**
```bash
# Pause connectors
./rollback-cdc.sh pause-all

# Run vacuum
PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d medications_db \
  -c "VACUUM FULL;"

# Resume connectors
./rollback-cdc.sh resume-all
```

---

## Emergency Contacts

**Escalation Path:**
1. **L1 Support:** DevOps on-call rotation
2. **L2 Support:** Data Engineering team
3. **L3 Support:** Platform Engineering lead

**Communication Channels:**
- Slack: #cdc-alerts
- PagerDuty: CDC Connector Escalation Policy
- Email: data-engineering@cardiofit.com

---

## Appendix

### Useful Commands Reference

```bash
# Quick health check
./verify-cdc-deployment.sh quick

# Full validation
./verify-cdc-deployment.sh full

# Pause all connectors (emergency)
./rollback-cdc.sh pause-all

# Resume all connectors
./rollback-cdc.sh resume-all

# Backup configurations
./rollback-cdc.sh backup

# List backups
./rollback-cdc.sh list-backups

# PostgreSQL diagnostic
./setup-postgresql-cdc.sh diagnostic
```

### Log Locations

- **Kafka Connect Logs:** `docker logs kafka-connect-container`
- **PostgreSQL Logs:** `/var/lib/postgresql/data/log/`
- **Connector Metrics:** `http://localhost:8083/metrics`
- **Prometheus Metrics:** `http://localhost:9090`
- **Grafana Dashboard:** `http://localhost:3000/d/cdc-connector-monitoring`

---

**Document Version:** 1.0
**Last Updated:** 2025-11-20
**Maintained By:** Data Engineering Team
