# CDC Connector Infrastructure

Production-ready Change Data Capture (CDC) infrastructure for CardioFit platform with comprehensive automation, monitoring, and operational tooling.

## Overview

This repository contains deployment automation, monitoring configuration, and operational procedures for 5 CDC connectors capturing changes from PostgreSQL databases to Kafka topics using Debezium.

### Connectors

| Connector | Database | Port | Purpose |
|-----------|----------|------|---------|
| kb1-medications-cdc | medications_db | 5432 | Medication management |
| kb2-scheduling-cdc | kb2_scheduling_db | 5433 | Appointment scheduling |
| kb3-encounter-cdc | kb3_encounter_db | 5434 | Clinical encounters |
| kb6-drug-rules-cdc | kb6_drug_rules_db | 5435 | Drug calculation rules |
| kb7-guideline-evidence-cdc | kb7_guideline_evidence_db | 5436 | Clinical guidelines |

## Quick Start

### Prerequisites

- Kafka cluster running (container ID: `3c7ffa06d20db1674c249c3ec2dda1bad58a1e92036abed8f504d1fdb0978754`)
- Kafka Connect cluster deployed with Debezium PostgreSQL plugin
- PostgreSQL instances running on ports 5432-5436
- Docker and Docker Compose installed
- Required tools: `psql`, `curl`, `jq`

### Deployment (3 Steps)

```bash
# 1. Verify infrastructure
cd scripts
chmod +x *.sh
./verify-infrastructure.sh

# 2. Setup PostgreSQL for CDC
./setup-postgresql-cdc.sh setup

# 3. Deploy all connectors
./deploy-all-cdc-connectors.sh deploy

# 4. Verify deployment
./verify-cdc-deployment.sh full
```

### Monitoring Setup

```bash
cd monitoring
docker-compose -f docker-compose.monitoring.yml up -d

# Access dashboards
# Grafana: http://localhost:3000 (admin/admin)
# Prometheus: http://localhost:9090
# Alertmanager: http://localhost:9093
```

## Repository Structure

```
cdc-connectors/
├── README.md                          # This file
├── configs/                           # Connector configurations
│   ├── kb1-medications-cdc.json
│   ├── kb2-scheduling-cdc.json
│   ├── kb3-encounter-cdc.json
│   ├── kb6-drug-rules-cdc.json
│   └── kb7-guideline-evidence-cdc.json
├── scripts/                           # Automation scripts
│   ├── verify-infrastructure.sh       # Infrastructure health checks
│   ├── setup-postgresql-cdc.sh        # PostgreSQL CDC preparation
│   ├── deploy-all-cdc-connectors.sh   # Connector deployment
│   ├── verify-cdc-deployment.sh       # Deployment validation
│   └── rollback-cdc.sh                # Rollback and recovery
├── monitoring/                        # Monitoring infrastructure
│   ├── docker-compose.monitoring.yml  # Monitoring stack
│   ├── prometheus/                    # Prometheus configuration
│   │   ├── prometheus.yml
│   │   └── cdc-connector-rules.yml    # Alert rules
│   ├── grafana/                       # Grafana dashboards
│   │   ├── cdc-dashboard.json
│   │   └── provisioning/
│   ├── alertmanager/                  # Alert routing
│   │   └── alertmanager.yml
│   └── exporters/                     # Metric exporters
│       └── postgres-queries.yaml
└── docs/                              # Documentation
    ├── DEPLOYMENT_GUIDE.md            # Comprehensive deployment guide
    └── OPERATIONAL_RUNBOOK.md         # Operations and troubleshooting
```

## Key Features

### Deployment Automation

- **Infrastructure Verification**: Pre-deployment health checks for Kafka, Kafka Connect, PostgreSQL
- **PostgreSQL CDC Setup**: Automated WAL configuration, replication slot, and publication creation
- **Connector Deployment**: Idempotent deployment with automatic validation
- **Health Verification**: Comprehensive post-deployment validation

### Monitoring and Observability

**Metrics Collection:**
- Kafka Connect JMX metrics (connector health, throughput, errors)
- PostgreSQL replication metrics (slot lag, WAL accumulation)
- Kafka topic metrics (message rate, partition health)
- System metrics (CPU, memory, disk, network)

**Alerting:**
- Critical alerts: Connector down, replication lag >15min, WAL accumulation >1GB
- Warning alerts: Connector paused, lag >5min, high error rate
- Info alerts: Schema evolution, configuration changes

**Dashboards:**
- Connector status overview
- Replication lag monitoring
- Throughput and performance metrics
- PostgreSQL replication health
- Resource utilization

### Disaster Recovery

**RTO/RPO Objectives:**
- Recovery Time Objective (RTO): 15 minutes
- Recovery Point Objective (RPO): 5 minutes

**Recovery Procedures:**
- Single connector failure: Automatic restart or redeployment
- Kafka Connect failure: Cluster restart and connector recovery
- PostgreSQL failure: Database restore and CDC reconfiguration
- Complete system failure: Full infrastructure rebuild

### Operational Tooling

**Scripts:**
- `verify-infrastructure.sh` - Pre-deployment validation
- `setup-postgresql-cdc.sh` - PostgreSQL preparation (setup/cleanup/diagnostic modes)
- `deploy-all-cdc-connectors.sh` - Deploy/pause/resume/status operations
- `verify-cdc-deployment.sh` - Full/quick/single connector validation
- `rollback-cdc.sh` - Emergency pause, rollback, backup, restore operations

## Common Operations

### Check Connector Health

```bash
# Quick health check
./scripts/verify-cdc-deployment.sh quick

# Full validation
./scripts/verify-cdc-deployment.sh full

# Single connector
./scripts/verify-cdc-deployment.sh connector kb1-medications-cdc
```

### Pause/Resume Connectors

```bash
# Pause all connectors (emergency)
./scripts/rollback-cdc.sh pause-all

# Resume all connectors
./scripts/rollback-cdc.sh resume-all

# Restart all connectors
./scripts/rollback-cdc.sh restart-all
```

### Backup and Restore

```bash
# Backup all connector configurations
./scripts/rollback-cdc.sh backup

# List available backups
./scripts/rollback-cdc.sh list-backups

# Restore from backup
./scripts/rollback-cdc.sh restore kb1-medications-cdc /path/to/backup.json
```

### PostgreSQL Diagnostics

```bash
# Run diagnostic report
./scripts/setup-postgresql-cdc.sh diagnostic

# Check replication slot status
PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d medications_db \
  -c "SELECT * FROM pg_replication_slots WHERE slot_name LIKE 'debezium%';"

# Check publication
PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d medications_db \
  -c "SELECT * FROM pg_publication WHERE pubname LIKE 'dbz_publication%';"
```

### Kafka Operations

```bash
# List CDC topics
docker exec 3c7ffa06d20db1674c249c3ec2dda1bad58a1e92036abed8f504d1fdb0978754 \
  kafka-topics --bootstrap-server localhost:9092 --list | grep cdc

# Consume from topic
docker exec 3c7ffa06d20db1674c249c3ec2dda1bad58a1e92036abed8f504d1fdb0978754 \
  kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic cdc.medications_db.public.medications \
  --from-beginning \
  --max-messages 10
```

## Troubleshooting

### Connector in FAILED State

```bash
# Check connector status
curl http://localhost:8083/connectors/kb1-medications-cdc/status | jq

# Check logs
docker logs kafka-connect-container | grep kb1-medications-cdc

# Attempt restart
curl -X POST http://localhost:8083/connectors/kb1-medications-cdc/restart

# If restart fails, redeploy
AUTO_REPLACE=true ./scripts/deploy-all-cdc-connectors.sh deploy
```

### High Replication Lag

```bash
# Check current lag
./scripts/verify-cdc-deployment.sh quick

# Check PostgreSQL replication slot lag
PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d medications_db \
  -c "SELECT slot_name, pg_size_pretty(pg_wal_lsn_diff(pg_current_wal_lsn(), confirmed_flush_lsn)) AS lag FROM pg_replication_slots;"

# Monitor in Grafana
# http://localhost:3000/d/cdc-connector-monitoring
```

### WAL Disk Accumulation

```bash
# Check WAL usage
PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d medications_db \
  -c "SELECT slot_name, pg_size_pretty(pg_wal_lsn_diff(pg_current_wal_lsn(), restart_lsn)) AS retained_wal FROM pg_replication_slots;"

# If connector is lagging, increase throughput
# If connector is stopped, resume or cleanup slot
./scripts/rollback-cdc.sh resume-all
```

## Performance Tuning

### Connector Configuration

**High Throughput:**
```json
{
  "tasks.max": "4",
  "max.batch.size": "4096",
  "max.queue.size": "16384"
}
```

**Low Latency:**
```json
{
  "tasks.max": "2",
  "max.batch.size": "1024",
  "poll.interval.ms": "100"
}
```

### PostgreSQL Configuration

```conf
# postgresql.conf
wal_level = logical
max_wal_senders = 10
max_replication_slots = 10
wal_keep_size = 4GB
max_wal_size = 4GB
```

## Monitoring Access

- **Grafana Dashboard**: http://localhost:3000/d/cdc-connector-monitoring
- **Prometheus UI**: http://localhost:9090
- **Prometheus Alerts**: http://localhost:9090/alerts
- **Alertmanager**: http://localhost:9093
- **Kafka Connect REST API**: http://localhost:8083

## Documentation

- **[Deployment Guide](docs/DEPLOYMENT_GUIDE.md)**: Comprehensive deployment procedures
- **[Operational Runbook](docs/OPERATIONAL_RUNBOOK.md)**: Operations, troubleshooting, disaster recovery

## Support

**Escalation Path:**
1. L1 Support: DevOps on-call rotation
2. L2 Support: Data Engineering team
3. L3 Support: Platform Engineering lead

**Communication:**
- Slack: #cdc-alerts
- PagerDuty: CDC Connector Escalation Policy
- Email: data-engineering@cardiofit.com

## License

Proprietary - CardioFit Platform

---

**Version**: 1.0
**Last Updated**: 2025-11-20
**Maintained By**: Data Engineering Team
