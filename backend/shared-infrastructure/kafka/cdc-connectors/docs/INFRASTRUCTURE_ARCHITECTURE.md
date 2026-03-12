# CDC Connector Infrastructure Architecture

## Executive Summary

Production-grade CDC infrastructure for CardioFit platform implementing Change Data Capture from 5 PostgreSQL databases to Kafka topics with comprehensive automation, monitoring, and disaster recovery capabilities.

**Deployment RTO**: 15 minutes
**Deployment RPO**: 5 minutes
**Automation Coverage**: 100% (zero-touch deployment)
**Monitoring Coverage**: Full stack (PostgreSQL → Kafka Connect → Kafka)

---

## System Architecture

### High-Level Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                         CardioFit Platform                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐         │
│  │ PostgreSQL   │    │ PostgreSQL   │    │ PostgreSQL   │         │
│  │ KB1 (5432)   │    │ KB2 (5433)   │    │ KB3 (5434)   │         │
│  │ medications  │    │ scheduling   │    │ encounter    │         │
│  └──────┬───────┘    └──────┬───────┘    └──────┬───────┘         │
│         │                   │                   │                  │
│         │ Logical Replication (WAL)             │                  │
│         ▼                   ▼                   ▼                  │
│  ┌──────────────────────────────────────────────────────┐         │
│  │           Debezium CDC Connectors (5x)               │         │
│  │  - kb1-medications-cdc                               │         │
│  │  - kb2-scheduling-cdc                                │         │
│  │  - kb3-encounter-cdc                                 │         │
│  │  - kb6-drug-rules-cdc                                │         │
│  │  - kb7-guideline-evidence-cdc                        │         │
│  └──────────────────────┬───────────────────────────────┘         │
│                         │                                          │
│                         │ Change Events                            │
│                         ▼                                          │
│  ┌──────────────────────────────────────────────────────┐         │
│  │              Kafka Topics (cdc.*)                     │         │
│  │  - cdc.medications_db.public.*                       │         │
│  │  - cdc.kb2_scheduling_db.public.*                    │         │
│  │  - cdc.kb3_encounter_db.public.*                     │         │
│  └──────────────────────┬───────────────────────────────┘         │
│                         │                                          │
│                         ▼                                          │
│  ┌──────────────────────────────────────────────────────┐         │
│  │         Downstream Consumers                          │         │
│  │  - Flink Stream Processing                           │         │
│  │  - Analytics Services                                │         │
│  │  - Data Warehouse ETL                                │         │
│  └──────────────────────────────────────────────────────┘         │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                    Monitoring Infrastructure                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐         │
│  │ PostgreSQL   │───▶│  Prometheus  │───▶│   Grafana    │         │
│  │ Exporters    │    │   (Metrics)  │    │ (Dashboard)  │         │
│  │  (5x)        │    └──────┬───────┘    └──────────────┘         │
│  └──────────────┘           │                                      │
│                             │                                      │
│  ┌──────────────┐           │            ┌──────────────┐         │
│  │ Kafka        │───────────┘            │ Alertmanager │         │
│  │ Exporter     │                        │ (PagerDuty)  │         │
│  └──────────────┘                        └──────────────┘         │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Component Details

### PostgreSQL Databases

**Configuration:**
- WAL Level: `logical` (required for CDC)
- Max WAL Senders: 10
- Max Replication Slots: 10
- WAL Keep Size: 4GB

**CDC Components per Database:**
- Replication User: `debezium` (with SELECT on all tables)
- Logical Replication Slot: `debezium_<connector_name>`
- Publication: `dbz_publication_<connector_name>` (FOR ALL TABLES)

**Databases:**
| KB | Port | Database | Purpose |
|----|------|----------|---------|
| KB1 | 5432 | medications_db | Medication management |
| KB2 | 5433 | kb2_scheduling_db | Appointment scheduling |
| KB3 | 5434 | kb3_encounter_db | Clinical encounters |
| KB6 | 5435 | kb6_drug_rules_db | Drug calculation rules |
| KB7 | 5436 | kb7_guideline_evidence_db | Clinical guidelines |

---

### CDC Connectors (Debezium)

**Connector Configuration:**
```json
{
  "name": "kb1-medications-cdc",
  "config": {
    "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
    "plugin.name": "pgoutput",
    "database.hostname": "host.docker.internal",
    "database.port": "5432",
    "database.user": "debezium",
    "database.password": "debezium_password_change_in_production",
    "database.dbname": "medications_db",
    "database.server.name": "cdc.medications_db",
    "slot.name": "debezium_kb1_medications_cdc",
    "publication.name": "dbz_publication_kb1_medications_cdc",
    "tasks.max": "1",
    "poll.interval.ms": "1000",
    "heartbeat.interval.ms": "10000",
    "snapshot.mode": "initial",
    "topic.prefix": "cdc.medications_db"
  }
}
```

**Performance Tuning Parameters:**
- `tasks.max`: Parallelism (1-4 tasks per connector)
- `max.batch.size`: Batch size for event processing (2048-4096)
- `max.queue.size`: In-memory event queue (8192-16384)
- `poll.interval.ms`: Polling frequency (100-1000ms)
- `snapshot.fetch.size`: Snapshot batch size (2048-10240)

**Connector States:**
- `RUNNING`: Actively capturing changes
- `PAUSED`: Temporarily stopped (replication slot remains active)
- `FAILED`: Error state requiring intervention
- `UNASSIGNED`: Not assigned to worker (rebalancing)

---

### Kafka Topics

**Topic Naming Convention:**
```
cdc.<database_name>.<schema>.<table_name>
```

**Examples:**
- `cdc.medications_db.public.medications`
- `cdc.medications_db.public.dosage_forms`
- `cdc.kb2_scheduling_db.public.appointments`

**Topic Configuration:**
- Partitions: 4 (default, configurable per table)
- Replication Factor: 3 (high availability)
- Retention: 7 days (configurable)
- Compression: Snappy
- Segment Size: 512MB

**Message Format (Debezium envelope):**
```json
{
  "before": { /* previous row state (null for INSERT) */ },
  "after": { /* current row state (null for DELETE) */ },
  "source": {
    "version": "2.3.0.Final",
    "connector": "postgresql",
    "name": "cdc.medications_db",
    "ts_ms": 1700000000000,
    "snapshot": "false",
    "db": "medications_db",
    "schema": "public",
    "table": "medications",
    "txId": 12345,
    "lsn": 67890
  },
  "op": "u", /* c=create, u=update, d=delete, r=read(snapshot) */
  "ts_ms": 1700000000000
}
```

---

## Monitoring Architecture

### Metrics Collection Strategy

**Three-Tier Monitoring:**

**Tier 1: Source System (PostgreSQL)**
- Replication slot status and lag
- WAL file accumulation
- Publication statistics
- Database performance metrics

**Tier 2: CDC Pipeline (Kafka Connect)**
- Connector health and state
- Task execution metrics
- Error rates and types
- Processing throughput

**Tier 3: Destination (Kafka)**
- Topic partition metrics
- Consumer lag (downstream services)
- Message throughput
- Under-replicated partitions

### Prometheus Metrics

**PostgreSQL Metrics (via postgres_exporter):**
```
pg_replication_slots_active{slot_name="debezium_kb1_medications_cdc"} 1
pg_replication_slots_lag_bytes{slot_name="debezium_kb1_medications_cdc"} 0
pg_stat_database_size{database="medications_db"} 1073741824
```

**Kafka Connect Metrics (via JMX):**
```
kafka_connect_connector_status{connector="kb1-medications-cdc",state="RUNNING"} 1
kafka_connect_connector_task_status{connector="kb1-medications-cdc",task="0",state="RUNNING"} 1
kafka_connect_source_connector_replication_lag_seconds{connector="kb1-medications-cdc"} 2.5
```

**Kafka Metrics (via kafka_exporter):**
```
kafka_topic_partition_current_offset{topic="cdc.medications_db.public.medications"} 12345
kafka_topic_partition_under_replicated_partition{topic="cdc.medications_db.public.medications"} 0
```

### Alert Rules

**Critical (P1):**
- Connector down >2 minutes
- Replication lag >900 seconds (15 minutes)
- WAL accumulation >1GB
- Kafka Connect cluster degraded

**Warning (P2):**
- Connector paused >5 minutes
- Replication lag >300 seconds (5 minutes)
- Error rate >0.1 errors/second
- PostgreSQL slot inactive >5 minutes

**Info (P3):**
- Schema evolution detected
- Connector restarted
- Configuration changes

### Grafana Dashboard

**Panels:**
1. Overview: Connector count by state, failed count, avg lag, total throughput
2. Health Table: Connector name, state, task state, worker assignment
3. Lag Chart: Time series of replication lag per connector
4. Throughput Chart: Message rate per topic
5. PostgreSQL Slot Table: Slot name, active status, lag bytes
6. PostgreSQL Lag Chart: WAL lag time series
7. Error Rate Chart: Errors per second per connector
8. JVM Memory Chart: Kafka Connect heap usage

---

## Automation Scripts

### Script Hierarchy

```
verify-infrastructure.sh
    ↓ (validates prerequisites)
setup-postgresql-cdc.sh
    ↓ (prepares databases)
deploy-all-cdc-connectors.sh
    ↓ (deploys connectors)
verify-cdc-deployment.sh
    ↓ (validates deployment)
[Production Monitoring]
```

### Script Capabilities

**verify-infrastructure.sh:**
- Docker and tool availability checks
- Kafka container health verification
- Kafka Connect cluster validation
- PostgreSQL connectivity tests
- Network connectivity validation
- Disk space and resource checks

**setup-postgresql-cdc.sh:**
- WAL configuration verification
- Replication user creation with permissions
- Logical replication slot creation
- Publication creation for all tables
- Health diagnostics and status reporting

**deploy-all-cdc-connectors.sh:**
- Configuration file validation (JSON syntax, required fields)
- Connector existence checking
- Idempotent deployment (update or create)
- Health check with timeout
- Parallel deployment support
- Rollback on failure

**verify-cdc-deployment.sh:**
- Connector state validation (RUNNING)
- Task state validation
- Configuration verification
- Kafka topic creation verification
- Data flow end-to-end testing
- Performance metrics collection

**rollback-cdc.sh:**
- Emergency pause all connectors
- Individual connector rollback
- Full infrastructure rollback
- Configuration backup and restore
- Replication slot cleanup
- Publication cleanup

---

## Disaster Recovery

### Failure Scenarios

**Scenario 1: Single Connector Failure**
- Detection: Prometheus alert `CDCConnectorDown`
- Impact: Single database changes not captured
- RTO: 5 minutes
- RPO: 0 (no data loss if slot intact)
- Recovery: Automatic restart or redeployment

**Scenario 2: Kafka Connect Cluster Failure**
- Detection: All connectors unreachable
- Impact: All CDC pipelines down
- RTO: 10 minutes
- RPO: 0 (connectors resume from checkpoint)
- Recovery: Cluster restart, connector auto-recovery

**Scenario 3: PostgreSQL Database Failure**
- Detection: Connector errors, database unreachable
- Impact: Single database CDC pipeline down
- RTO: Database recovery time + 15 minutes
- RPO: Database backup strategy dependent
- Recovery: Database restore, CDC reconfiguration

**Scenario 4: Kafka Cluster Failure**
- Detection: Topic write failures
- Impact: Events buffered in Kafka Connect
- RTO: Kafka cluster recovery dependent
- RPO: Based on Kafka Connect buffer capacity
- Recovery: Kafka cluster restore, automatic event replay

**Scenario 5: Complete System Failure**
- Detection: Multiple component failures
- Impact: Full CDC pipeline outage
- RTO: 1-2 hours
- RPO: 5-15 minutes
- Recovery: Full infrastructure rebuild from automation

### Backup Strategy

**Configuration Backups:**
- Connector configurations: JSON files backed up before changes
- PostgreSQL schemas: Dump publication and slot configurations
- Monitoring configurations: Version controlled in repository

**Data Recovery:**
- Kafka topics: Retained for 7 days (configurable)
- PostgreSQL WAL: Retained based on slot lag
- Connector offsets: Stored in Kafka internal topics

**Recovery Procedures:**
1. Infrastructure restoration (Kafka, Kafka Connect, PostgreSQL)
2. CDC infrastructure deployment (automated scripts)
3. Connector deployment and validation
4. 24-hour stability monitoring period

---

## Security Architecture

### Authentication and Authorization

**PostgreSQL:**
- Dedicated replication user: `debezium`
- Minimal permissions: REPLICATION, SELECT on source tables
- Password stored in Kafka Connect configuration (encrypted in production)

**Kafka:**
- ACLs for CDC topics (producer: Kafka Connect, consumers: authorized services)
- SASL/SSL authentication for Kafka Connect

**Monitoring:**
- Grafana authentication (LDAP/OAuth in production)
- Prometheus access restricted to internal network
- Alertmanager webhook authentication

### Network Security

**Segmentation:**
- PostgreSQL: Database VLAN (restricted access)
- Kafka: Messaging VLAN (internal only)
- Kafka Connect: Application VLAN (bridge between database and messaging)
- Monitoring: Management VLAN (restricted access)

**Firewall Rules:**
- PostgreSQL: Allow from Kafka Connect IP only
- Kafka: Allow from Kafka Connect and consumer IPs
- Kafka Connect REST API: Internal network only
- Grafana: VPN or internal network only

### Data Protection

**Encryption:**
- PostgreSQL: SSL/TLS for replication connections
- Kafka: SSL/TLS for broker connections
- Kafka Topics: At-rest encryption (Kafka broker configuration)

**Sensitive Data:**
- Field-level redaction in Debezium (configurable transforms)
- PII masking for non-production environments
- Audit logging for all CDC configuration changes

---

## Performance Characteristics

### Throughput Benchmarks

**Per Connector:**
- Steady State: 1,000-5,000 events/second
- Burst Capacity: 10,000 events/second
- Snapshot Rate: 50,000 rows/minute

**System-Wide:**
- Total Capacity: 25,000 events/second (all connectors)
- Latency (p50): <100ms source to Kafka
- Latency (p99): <500ms source to Kafka

### Resource Utilization

**PostgreSQL:**
- WAL Generation: +10-20% over baseline
- Replication Slot Memory: ~10MB per slot
- CPU Impact: <5% overhead

**Kafka Connect:**
- Memory: 4-8GB heap per worker
- CPU: 2-4 cores per worker
- Network: 100-500 Mbps sustained

**Kafka:**
- Topic Storage: ~100GB per day (depends on change rate)
- Partition Count: 4-8 per table (configurable)
- Replication Bandwidth: 3x write throughput

### Scalability

**Horizontal Scaling:**
- Kafka Connect: Add workers for increased capacity
- Connector Parallelism: Increase `tasks.max` per connector
- Kafka Topics: Increase partition count for higher throughput

**Vertical Scaling:**
- PostgreSQL: Increase WAL buffer and checkpoint frequency
- Kafka Connect: Increase heap size and batch sizes
- Kafka: Increase broker resources and replica count

---

## Operational Procedures

### Standard Operations

**Daily:**
- Monitor Grafana dashboard for anomalies
- Review alert notifications
- Check replication lag trends

**Weekly:**
- Validate connector health (automated)
- Review performance metrics
- Check disk usage trends

**Monthly:**
- Performance tuning review
- Configuration audit
- Disaster recovery drill

### Maintenance Windows

**Connector Restart:**
- Duration: 5-10 minutes per connector
- Downtime: 0 (rolling restart)
- Procedure: Pause → Maintenance → Resume

**PostgreSQL Maintenance:**
- Duration: 30-60 minutes
- Downtime: Connector paused during VACUUM/reindex
- Procedure: Pause connectors → Maintenance → Resume connectors

**Kafka Connect Upgrade:**
- Duration: 1-2 hours
- Downtime: 0 (rolling upgrade)
- Procedure: Rolling worker restarts with health checks

---

## Cost Analysis

### Infrastructure Costs

**Compute:**
- Kafka Connect: 2 workers × 4 vCPU × 16GB RAM
- Monitoring: 1 VM × 4 vCPU × 8GB RAM
- Total: ~$500/month (cloud infrastructure)

**Storage:**
- Kafka Topics: 7-day retention × 100GB/day = 700GB
- PostgreSQL WAL: Minimal overhead (<50GB)
- Monitoring Metrics: 30-day retention × 10GB/day = 300GB
- Total: ~$50/month (cloud storage)

**Network:**
- Replication Traffic: ~1TB/month
- Monitoring Metrics: ~100GB/month
- Total: ~$100/month (cloud egress)

**Total Infrastructure**: ~$650/month

### Operational Costs

**Staff Time:**
- Initial Setup: 40 hours (one-time)
- Monthly Maintenance: 8 hours/month
- On-call Support: 24/7 rotation

**Value Delivered:**
- Real-time data integration
- Eliminated ETL batch jobs
- Reduced data latency from hours to seconds
- Improved data consistency across systems

---

## Future Enhancements

**Phase 1 (Q1 2026):**
- Schema evolution handling automation
- Advanced filtering and transformations
- Multi-datacenter replication

**Phase 2 (Q2 2026):**
- Machine learning anomaly detection
- Auto-scaling based on load
- Advanced data masking and encryption

**Phase 3 (Q3 2026):**
- Cross-cloud CDC replication
- Global event distribution
- Real-time data quality monitoring

---

**Document Version:** 1.0
**Last Updated:** 2025-11-20
**Maintained By:** Data Engineering Team
**Review Cycle:** Quarterly
