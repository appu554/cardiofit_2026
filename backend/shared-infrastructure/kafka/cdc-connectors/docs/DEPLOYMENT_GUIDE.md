# CDC Connector Deployment Guide

## Overview

Comprehensive deployment guide for CDC connectors with infrastructure automation, monitoring setup, and operational procedures.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Architecture Overview](#architecture-overview)
3. [Deployment Workflow](#deployment-workflow)
4. [Monitoring Setup](#monitoring-setup)
5. [Validation and Testing](#validation-and-testing)
6. [Troubleshooting](#troubleshooting)

---

## Prerequisites

### Required Infrastructure

**Kafka Cluster:**
- Kafka broker running and accessible
- Existing Kafka container: `3c7ffa06d20db1674c249c3ec2dda1bad58a1e92036abed8f504d1fdb0978754`
- Network: `cardiofit-network`

**Kafka Connect:**
- Kafka Connect cluster deployed
- Debezium PostgreSQL connector plugin installed
- REST API accessible at `http://localhost:8083`

**PostgreSQL Databases:**
- KB1: `localhost:5432` - medications_db
- KB2: `localhost:5433` - kb2_scheduling_db
- KB3: `localhost:5434` - kb3_encounter_db
- KB6: `localhost:5435` - kb6_drug_rules_db
- KB7: `localhost:5436` - kb7_guideline_evidence_db

**System Requirements:**
- Docker and Docker Compose installed
- Minimum 8GB RAM available
- Minimum 50GB disk space
- Network connectivity between all components

### Required Tools

```bash
# Verify tools are installed
command -v docker && echo "Docker: OK"
command -v psql && echo "PostgreSQL client: OK"
command -v curl && echo "curl: OK"
command -v jq && echo "jq: OK"
```

### Environment Variables

Create `.env` file:

```bash
# PostgreSQL passwords
export PGPASSWORD_KB1="postgres"
export PGPASSWORD_KB2="postgres"
export PGPASSWORD_KB3="postgres"
export PGPASSWORD_KB6="postgres"
export PGPASSWORD_KB7="postgres"

# Kafka Connect URL
export KAFKA_CONNECT_URL="http://localhost:8083"

# Auto-replace existing connectors
export AUTO_REPLACE="false"
```

---

## Architecture Overview

### Data Flow

```
PostgreSQL Database (KB1-7)
    ↓ (Logical Replication)
Debezium Connector (CDC)
    ↓ (Change Events)
Kafka Topics (cdc.*)
    ↓ (Consumed by downstream services)
```

### Components

**CDC Connectors:**
- `kb1-medications-cdc` - Captures medication database changes
- `kb2-scheduling-cdc` - Captures scheduling database changes
- `kb3-encounter-cdc` - Captures encounter database changes
- `kb6-drug-rules-cdc` - Captures drug rules database changes
- `kb7-guideline-evidence-cdc` - Captures guideline evidence database changes

**PostgreSQL Components:**
- WAL (Write-Ahead Log) with `wal_level=logical`
- Logical replication slots (one per connector)
- Publications (one per database)
- Replication user: `debezium`

**Kafka Components:**
- CDC topics (auto-created per table)
- Schema registry topics (for schema evolution)
- Connector configuration topics

---

## Deployment Workflow

### Phase 1: Infrastructure Verification

**Objective:** Validate all prerequisites are met

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/kafka/cdc-connectors/scripts

# Make scripts executable
chmod +x *.sh

# Run infrastructure verification
./verify-infrastructure.sh
```

**Expected Output:**
```
[INFO] All required tools are available
[INFO] Kafka container is running
[INFO] Kafka Connect is reachable
[INFO] PostgreSQL connectivity verified (5/5 instances reachable)
[SUCCESS] Infrastructure is ready for CDC connector deployment
```

**If verification fails:**
- Fix reported issues
- Re-run verification
- Do not proceed until all checks pass

---

### Phase 2: PostgreSQL CDC Setup

**Objective:** Prepare PostgreSQL instances for CDC capture

```bash
# Run PostgreSQL CDC setup
./setup-postgresql-cdc.sh setup
```

**What this does:**
1. Verifies WAL configuration (`wal_level=logical`)
2. Creates replication user `debezium`
3. Grants necessary permissions
4. Creates logical replication slots
5. Creates publications for all tables

**Expected Output:**
```
[KB1] WAL level is correctly set to 'logical'
[KB1] Created replication user 'debezium'
[KB1] Created replication slot 'debezium_kb1_medications_cdc'
[KB1] Created publication 'dbz_publication_kb1_medications_cdc' for all tables
[KB1] CDC setup completed successfully
```

**Verification:**

```bash
# Run diagnostic report
./setup-postgresql-cdc.sh diagnostic

# Manual verification for KB1
PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d medications_db <<EOF
SELECT * FROM pg_replication_slots WHERE slot_name LIKE 'debezium%';
SELECT * FROM pg_publication WHERE pubname LIKE 'dbz_publication%';
EOF
```

**If setup fails:**
- Check PostgreSQL logs
- Verify WAL configuration in postgresql.conf
- Ensure PostgreSQL user has superuser privileges
- Review error messages for specific issues

---

### Phase 3: Connector Deployment

**Objective:** Deploy all CDC connectors to Kafka Connect

```bash
# Deploy all connectors
./deploy-all-cdc-connectors.sh deploy
```

**What this does:**
1. Validates connector configuration files
2. Checks if connectors already exist
3. Creates new connectors via Kafka Connect REST API
4. Waits for connectors to reach RUNNING state
5. Validates connector health

**Expected Output:**
```
[kb1-medications-cdc] Configuration file is valid
[kb1-medications-cdc] Creating connector...
[kb1-medications-cdc] Connector created successfully
[kb1-medications-cdc] Waiting for connector to be RUNNING...
[kb1-medications-cdc] Connector is running successfully
[kb1-medications-cdc] Connector is healthy (state: RUNNING, tasks: 1)

Deployment Summary:
  Total connectors: 5
  Successfully deployed: 5
  Failed: 0
```

**Replace existing connectors:**
```bash
AUTO_REPLACE=true ./deploy-all-cdc-connectors.sh deploy
```

**Verification:**

```bash
# List all connectors
./deploy-all-cdc-connectors.sh list

# Check specific connector
curl http://localhost:8083/connectors/kb1-medications-cdc/status | jq
```

**If deployment fails:**
- Check connector logs: `docker logs kafka-connect-container`
- Verify Kafka Connect has Debezium plugin
- Ensure PostgreSQL replication slots are active
- Review connector configuration in `/configs/`

---

### Phase 4: Deployment Validation

**Objective:** Comprehensive validation of CDC deployment

```bash
# Full validation
./verify-cdc-deployment.sh full
```

**What this validates:**
1. Connector status (RUNNING)
2. Connector configuration (correct settings)
3. Kafka topics created
4. Data flow end-to-end
5. Replication lag

**Expected Output:**
```
[kb1-medications-cdc] Connector and task are running
[kb1-medications-cdc] Configuration is valid
[kb1-medications-cdc] Found 12 topic(s) with prefix: cdc.medications_db
[kb1-medications-cdc] Data flow verification completed

CDC Deployment Validation Report
========================================
Total connectors: 5
Passed: 5
Failed: 0
========================================
All CDC connectors are healthy and operational
```

**Quick health check:**
```bash
# Quick status check
./verify-cdc-deployment.sh quick
```

**Validate specific connector:**
```bash
./verify-cdc-deployment.sh connector kb1-medications-cdc
```

---

### Phase 5: Monitoring Setup

**Objective:** Deploy monitoring infrastructure

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/kafka/cdc-connectors/monitoring

# Start monitoring stack
docker-compose -f docker-compose.monitoring.yml up -d
```

**What this deploys:**
- Prometheus (metrics collection)
- Grafana (visualization)
- Alertmanager (alert routing)
- PostgreSQL exporters (5 instances)
- Kafka exporter
- Node exporter
- cAdvisor

**Access monitoring:**
- Grafana: http://localhost:3000 (admin/admin)
- Prometheus: http://localhost:9090
- Alertmanager: http://localhost:9093

**Load Grafana dashboard:**
1. Navigate to http://localhost:3000
2. Login (admin/admin)
3. Dashboard provisioned automatically at startup
4. Go to "CDC Connector Monitoring Dashboard"

**Verify monitoring:**
```bash
# Check Prometheus targets
curl http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | select(.health=="up")'

# Check Grafana health
curl http://localhost:3000/api/health
```

---

## Monitoring Setup

### Metrics Collection

**Kafka Connect Metrics:**
- Connector state and status
- Task state and errors
- Record processing throughput
- JVM memory and GC metrics

**PostgreSQL Metrics:**
- Replication slot status and lag
- WAL file accumulation
- Publication statistics
- Database size and growth

**Kafka Metrics:**
- Topic partition metrics
- Consumer lag
- Topic throughput
- Under-replicated partitions

### Alert Rules

**Critical Alerts (immediate action):**
- `CDCConnectorDown` - Connector not running
- `CDCConnectorTaskFailed` - Task failed
- `CDCReplicationLagCritical` - Lag >15 minutes
- `PostgreSQLReplicationSlotWALAccumulation` - WAL >1GB

**Warning Alerts (action required soon):**
- `CDCConnectorPaused` - Connector paused >5 minutes
- `CDCReplicationLagHigh` - Lag >5 minutes
- `CDCConnectorHighErrorRate` - Errors >0.1/sec
- `PostgreSQLReplicationSlotInactive` - Slot inactive

**Info Alerts (informational):**
- `CDCSchemaEvolution` - Schema change detected
- `CDCTopicNoActivity` - No messages for 1 hour

### Dashboard Panels

**Overview:**
- Connector status (running/failed counts)
- Average replication lag
- Total CDC throughput
- Failed connectors alert

**Health Status:**
- Connector health table
- Task status per connector
- Worker assignment

**Performance:**
- Replication lag by connector (time series)
- Topic message rate (time series)
- Error rate by connector

**PostgreSQL:**
- Replication slot status table
- Replication slot lag (time series)
- WAL accumulation

**Resources:**
- Kafka Connect JVM memory
- System resource utilization
- Container metrics

---

## Validation and Testing

### End-to-End Data Flow Test

**Test KB1 (Medications):**

```bash
# 1. Make a change in PostgreSQL
PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d medications_db <<EOF
UPDATE medications
SET updated_at = NOW()
WHERE id = 1;
EOF

# 2. Wait for propagation (should be <5 seconds)
sleep 5

# 3. Verify event in Kafka topic
docker exec 3c7ffa06d20db1674c249c3ec2dda1bad58a1e92036abed8f504d1fdb0978754 \
  kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic cdc.medications_db.public.medications \
  --max-messages 1 \
  --from-beginning
```

**Expected result:**
JSON event with change data capture information including before/after values.

### Connector Resilience Test

**Test connector recovery:**

```bash
# 1. Pause connector
curl -X PUT http://localhost:8083/connectors/kb1-medications-cdc/pause

# 2. Make changes while paused
PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d medications_db <<EOF
INSERT INTO medications (name, dosage) VALUES ('Test Med', '100mg');
EOF

# 3. Resume connector
curl -X PUT http://localhost:8083/connectors/kb1-medications-cdc/resume

# 4. Verify missed changes are captured
docker exec 3c7ffa06d20db1674c249c3ec2dda1bad58a1e92036abed8f504d1fdb0978754 \
  kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic cdc.medications_db.public.medications \
  --max-messages 1
```

### Performance Baseline

**Establish performance metrics:**

```bash
# Insert test load
PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d medications_db <<EOF
INSERT INTO medications (name, dosage)
SELECT 'Test Med ' || i, '100mg'
FROM generate_series(1, 1000) i;
EOF

# Monitor lag in Grafana
# Expected: Lag <5 seconds for 1000 inserts
```

---

## Troubleshooting

### Common Issues

**Issue: Connector fails with "replication slot does not exist"**

Solution:
```bash
./setup-postgresql-cdc.sh setup
./deploy-all-cdc-connectors.sh deploy
```

**Issue: High replication lag**

Diagnosis:
```bash
# Check connector throughput
curl http://localhost:8083/connectors/kb1-medications-cdc/status | jq

# Check PostgreSQL slot lag
PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d medications_db \
  -c "SELECT slot_name, pg_size_pretty(pg_wal_lsn_diff(pg_current_wal_lsn(), confirmed_flush_lsn)) FROM pg_replication_slots;"
```

Solution: Increase connector parallelism or Kafka Connect resources.

**Issue: No topics created**

Diagnosis:
```bash
# List Kafka topics
docker exec 3c7ffa06d20db1674c249c3ec2dda1bad58a1e92036abed8f504d1fdb0978754 \
  kafka-topics --bootstrap-server localhost:9092 --list | grep cdc
```

Solution: Verify connector is running and make a change in database to trigger topic creation.

### Emergency Procedures

**Emergency pause all connectors:**
```bash
./rollback-cdc.sh pause-all
```

**Emergency rollback:**
```bash
./rollback-cdc.sh rollback-all
```

**Backup configurations:**
```bash
./rollback-cdc.sh backup
```

---

## Appendix

### File Locations

**Scripts:**
```
/backend/shared-infrastructure/kafka/cdc-connectors/scripts/
├── verify-infrastructure.sh      # Infrastructure validation
├── setup-postgresql-cdc.sh       # PostgreSQL CDC setup
├── deploy-all-cdc-connectors.sh  # Connector deployment
├── verify-cdc-deployment.sh      # Deployment validation
└── rollback-cdc.sh               # Rollback and recovery
```

**Configurations:**
```
/backend/shared-infrastructure/kafka/cdc-connectors/configs/
├── kb1-medications-cdc.json
├── kb2-scheduling-cdc.json
├── kb3-encounter-cdc.json
├── kb6-drug-rules-cdc.json
└── kb7-guideline-evidence-cdc.json
```

**Monitoring:**
```
/backend/shared-infrastructure/kafka/cdc-connectors/monitoring/
├── docker-compose.monitoring.yml
├── prometheus/
│   ├── prometheus.yml
│   └── cdc-connector-rules.yml
├── grafana/
│   └── cdc-dashboard.json
└── alertmanager/
    └── alertmanager.yml
```

### Useful Commands

```bash
# Quick status check
./verify-cdc-deployment.sh quick

# List all connectors
curl http://localhost:8083/connectors | jq

# Get connector status
curl http://localhost:8083/connectors/kb1-medications-cdc/status | jq

# Restart connector
curl -X POST http://localhost:8083/connectors/kb1-medications-cdc/restart

# Pause connector
curl -X PUT http://localhost:8083/connectors/kb1-medications-cdc/pause

# Resume connector
curl -X PUT http://localhost:8083/connectors/kb1-medications-cdc/resume
```

---

**Document Version:** 1.0
**Last Updated:** 2025-11-20
**Maintained By:** Data Engineering Team
