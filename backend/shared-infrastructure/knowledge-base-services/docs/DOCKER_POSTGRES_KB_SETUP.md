# 🐳 Docker PostgreSQL Setup for Knowledge Base Services

## 📋 Overview

This document provides specific guidance for implementing the Knowledge Base services using Docker PostgreSQL as the primary database infrastructure. All 7 KB services will utilize containerized PostgreSQL instances with proper isolation, performance optimization, and data persistence.

---

## 🗄️ Docker PostgreSQL Architecture

### Container Strategy
```yaml
# Each KB service gets its own logical database within shared PostgreSQL containers
PostgreSQL Containers:
  1. kb-postgres-primary (Port 5433)
     - kb_drug_rules
     - kb_terminology  
     - kb_ddi
     - kb_formulary
     
  2. kb-postgres-governance (Port 5434)
     - clinical_governance (Evidence Envelope)
     - kb_audit_log
     
  3. kb-postgres-timescale (Port 5435)
     - kb_patient_safety (TimescaleDB)
```

---

## 🚀 Phase 0: Docker PostgreSQL Foundation Setup

### Step 1: Create Docker Compose Configuration

```yaml
# docker-compose.postgres-kb.yml
version: '3.8'

services:
  # Primary PostgreSQL for KB Services
  kb-postgres-primary:
    image: postgres:15-alpine
    container_name: kb_postgres_primary
    restart: unless-stopped
    ports:
      - "5433:5432"
    environment:
      POSTGRES_USER: kb_admin
      POSTGRES_PASSWORD: ${KB_POSTGRES_PASSWORD:-kb_secure_password_2025}
      POSTGRES_DB: postgres
      POSTGRES_INITDB_ARGS: "--encoding=UTF8 --locale=en_US.UTF-8"
    volumes:
      - kb_postgres_primary_data:/var/lib/postgresql/data
      - ./init-scripts/primary:/docker-entrypoint-initdb.d
      - ./backups/primary:/backups
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U kb_admin"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - kb_network
    command: >
      postgres
      -c shared_buffers=256MB
      -c max_connections=200
      -c effective_cache_size=1GB
      -c maintenance_work_mem=128MB
      -c checkpoint_completion_target=0.9
      -c wal_buffers=16MB
      -c default_statistics_target=100
      -c random_page_cost=1.1
      -c effective_io_concurrency=200
      -c work_mem=4MB
      -c min_wal_size=1GB
      -c max_wal_size=4GB
      -c max_worker_processes=8
      -c max_parallel_workers_per_gather=4
      -c max_parallel_workers=8
      -c max_parallel_maintenance_workers=4

  # Governance PostgreSQL (Evidence Envelope)
  kb-postgres-governance:
    image: postgres:15-alpine
    container_name: kb_postgres_governance
    restart: unless-stopped
    ports:
      - "5434:5432"
    environment:
      POSTGRES_USER: governance_admin
      POSTGRES_PASSWORD: ${GOVERNANCE_POSTGRES_PASSWORD:-governance_secure_2025}
      POSTGRES_DB: clinical_governance
    volumes:
      - kb_postgres_governance_data:/var/lib/postgresql/data
      - ./init-scripts/governance:/docker-entrypoint-initdb.d
      - ./backups/governance:/backups
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U governance_admin"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - kb_network
    command: >
      postgres
      -c shared_buffers=256MB
      -c max_connections=100
      -c log_statement=all
      -c log_duration=on

  # TimescaleDB for Patient Safety
  kb-postgres-timescale:
    image: timescale/timescaledb:latest-pg15
    container_name: kb_postgres_timescale
    restart: unless-stopped
    ports:
      - "5435:5432"
    environment:
      POSTGRES_USER: safety_admin
      POSTGRES_PASSWORD: ${SAFETY_POSTGRES_PASSWORD:-safety_secure_2025}
      POSTGRES_DB: kb_patient_safety
      TIMESCALEDB_TELEMETRY: "off"
    volumes:
      - kb_postgres_timescale_data:/var/lib/postgresql/data
      - ./init-scripts/timescale:/docker-entrypoint-initdb.d
      - ./backups/timescale:/backups
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U safety_admin"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - kb_network

  # pgAdmin for database management
  kb-pgadmin:
    image: dpage/pgadmin4:latest
    container_name: kb_pgadmin
    restart: unless-stopped
    ports:
      - "5050:80"
    environment:
      PGADMIN_DEFAULT_EMAIL: admin@kbservices.local
      PGADMIN_DEFAULT_PASSWORD: ${PGADMIN_PASSWORD:-pgadmin_2025}
      PGADMIN_CONFIG_SERVER_MODE: 'False'
      PGADMIN_CONFIG_MASTER_PASSWORD_REQUIRED: 'False'
    volumes:
      - kb_pgadmin_data:/var/lib/pgadmin
      - ./pgadmin/servers.json:/pgadmin4/servers.json
    networks:
      - kb_network
    depends_on:
      - kb-postgres-primary
      - kb-postgres-governance
      - kb-postgres-timescale

  # Backup service
  kb-postgres-backup:
    image: postgres:15-alpine
    container_name: kb_postgres_backup
    restart: unless-stopped
    environment:
      - BACKUP_SCHEDULE=0 2 * * *  # Daily at 2 AM
    volumes:
      - ./backups:/backups
      - ./backup-scripts:/scripts
    networks:
      - kb_network
    command: /scripts/backup-cron.sh
    depends_on:
      - kb-postgres-primary
      - kb-postgres-governance
      - kb-postgres-timescale

volumes:
  kb_postgres_primary_data:
    driver: local
  kb_postgres_governance_data:
    driver: local
  kb_postgres_timescale_data:
    driver: local
  kb_pgadmin_data:
    driver: local

networks:
  kb_network:
    driver: bridge
    ipam:
      config:
        - subnet: 172.28.0.0/16
```

### Step 2: Initialize Databases

Create initialization scripts for each PostgreSQL container:

#### Primary Database Initialization
```sql
-- init-scripts/primary/01-create-databases.sql

-- Create KB service databases
CREATE DATABASE kb_drug_rules;
CREATE DATABASE kb_terminology;
CREATE DATABASE kb_ddi;
CREATE DATABASE kb_formulary;

-- Create service users
CREATE USER drug_rules_user WITH PASSWORD 'drug_rules_pass_2025';
CREATE USER terminology_user WITH PASSWORD 'terminology_pass_2025';
CREATE USER ddi_user WITH PASSWORD 'ddi_pass_2025';
CREATE USER formulary_user WITH PASSWORD 'formulary_pass_2025';

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE kb_drug_rules TO drug_rules_user;
GRANT ALL PRIVILEGES ON DATABASE kb_terminology TO terminology_user;
GRANT ALL PRIVILEGES ON DATABASE kb_ddi TO ddi_user;
GRANT ALL PRIVILEGES ON DATABASE kb_formulary TO formulary_user;

-- Performance optimizations per database
ALTER DATABASE kb_drug_rules SET random_page_cost = 1.1;
ALTER DATABASE kb_terminology SET random_page_cost = 1.1;
ALTER DATABASE kb_ddi SET random_page_cost = 1.1;
ALTER DATABASE kb_formulary SET random_page_cost = 1.1;

-- Enable extensions
\c kb_drug_rules
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

\c kb_terminology
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";  -- For fuzzy text search
CREATE EXTENSION IF NOT EXISTS "unaccent"; -- For accent-insensitive search

\c kb_ddi
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "btree_gin"; -- For optimized indexing

\c kb_formulary
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "tablefunc"; -- For crosstab reports
```

#### Governance Database Initialization
```sql
-- init-scripts/governance/01-evidence-envelope.sql

-- Already connected to clinical_governance database
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Evidence Envelope tables with partitioning
CREATE TABLE kb_version_sets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version_set_name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    kb_versions JSONB NOT NULL DEFAULT '{}',
    validated BOOLEAN DEFAULT FALSE,
    validation_results JSONB,
    environment VARCHAR(50) NOT NULL,
    active BOOLEAN DEFAULT FALSE,
    activated_at TIMESTAMPTZ,
    created_by VARCHAR(100) NOT NULL,
    approved_by VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT unique_active_per_env EXCLUDE (environment WITH =) WHERE (active = true)
);

-- Evidence tracking with partitioning
CREATE TABLE evidence_envelopes (
    id UUID DEFAULT gen_random_uuid(),
    transaction_id VARCHAR(100) UNIQUE NOT NULL,
    version_set_id UUID REFERENCES kb_version_sets(id),
    kb_versions JSONB NOT NULL,
    decision_chain JSONB NOT NULL DEFAULT '[]',
    safety_attestations JSONB NOT NULL DEFAULT '[]',
    patient_id VARCHAR(100),
    clinical_domain VARCHAR(50),
    orchestrator_version VARCHAR(50),
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    checksum VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create monthly partitions for 2025
DO $$
DECLARE
    start_date DATE := '2025-01-01';
    end_date DATE;
    partition_name TEXT;
BEGIN
    FOR i IN 0..11 LOOP
        end_date := start_date + INTERVAL '1 month';
        partition_name := 'evidence_envelopes_' || to_char(start_date, 'YYYY_MM');
        
        EXECUTE format('
            CREATE TABLE %I PARTITION OF evidence_envelopes
            FOR VALUES FROM (%L) TO (%L)',
            partition_name, start_date, end_date
        );
        
        start_date := end_date;
    END LOOP;
END $$;

-- Indexes for performance
CREATE INDEX idx_kb_version_sets_environment ON kb_version_sets(environment);
CREATE INDEX idx_kb_version_sets_active ON kb_version_sets(active);
CREATE INDEX idx_evidence_envelopes_transaction_id ON evidence_envelopes(transaction_id);
CREATE INDEX idx_evidence_envelopes_patient_id ON evidence_envelopes(patient_id);

-- Audit log table
CREATE TABLE kb_audit_log (
    id BIGSERIAL PRIMARY KEY,
    entity_type VARCHAR(50) NOT NULL,
    entity_id VARCHAR(100) NOT NULL,
    action VARCHAR(50) NOT NULL,
    old_value JSONB,
    new_value JSONB,
    user_id VARCHAR(100) NOT NULL,
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    metadata JSONB
);

CREATE INDEX idx_audit_entity ON kb_audit_log(entity_type, entity_id);
CREATE INDEX idx_audit_timestamp ON kb_audit_log(timestamp);
```

#### TimescaleDB Initialization
```sql
-- init-scripts/timescale/01-patient-safety.sql

-- Enable TimescaleDB
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Safety alerts table
CREATE TABLE safety_alerts (
    time TIMESTAMPTZ NOT NULL,
    alert_id UUID DEFAULT gen_random_uuid(),
    patient_id VARCHAR(100) NOT NULL,
    alert_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) CHECK (severity IN ('critical', 'high', 'medium', 'low')),
    description TEXT,
    source_system VARCHAR(50),
    triggering_values JSONB,
    recommendations JSONB,
    acknowledged BOOLEAN DEFAULT FALSE,
    acknowledged_by VARCHAR(100),
    acknowledged_at TIMESTAMPTZ,
    metadata JSONB
);

-- Convert to hypertable
SELECT create_hypertable('safety_alerts', 'time', chunk_time_interval => INTERVAL '1 day');

-- Create indexes
CREATE INDEX idx_safety_alerts_patient ON safety_alerts(patient_id, time DESC);
CREATE INDEX idx_safety_alerts_severity ON safety_alerts(severity, time DESC);

-- Continuous aggregate for hourly summaries
CREATE MATERIALIZED VIEW safety_alerts_hourly
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 hour', time) AS hour,
    alert_type,
    severity,
    COUNT(*) as alert_count,
    COUNT(DISTINCT patient_id) as unique_patients
FROM safety_alerts
WHERE time > NOW() - INTERVAL '30 days'
GROUP BY hour, alert_type, severity
WITH NO DATA;

-- Refresh policy
SELECT add_continuous_aggregate_policy('safety_alerts_hourly',
    start_offset => INTERVAL '3 hours',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour');

-- Retention policy (90 days)
SELECT add_retention_policy('safety_alerts', INTERVAL '90 days');
```

### Step 3: Backup and Recovery Scripts

```bash
#!/bin/bash
# backup-scripts/backup-cron.sh

# Backup configuration
BACKUP_DIR="/backups"
RETENTION_DAYS=30

# Function to perform backup
backup_database() {
    local host=$1
    local port=$2
    local user=$3
    local password=$4
    local database=$5
    local backup_name=$6
    
    timestamp=$(date +%Y%m%d_%H%M%S)
    backup_file="${BACKUP_DIR}/${backup_name}_${timestamp}.sql.gz"
    
    echo "Backing up ${database} to ${backup_file}"
    PGPASSWORD=${password} pg_dump \
        -h ${host} \
        -p ${port} \
        -U ${user} \
        -d ${database} \
        --verbose \
        --no-owner \
        --no-acl \
        --format=plain \
        | gzip > ${backup_file}
    
    if [ $? -eq 0 ]; then
        echo "Backup successful: ${backup_file}"
        # Clean old backups
        find ${BACKUP_DIR} -name "${backup_name}_*.sql.gz" -mtime +${RETENTION_DAYS} -delete
    else
        echo "Backup failed for ${database}"
        exit 1
    fi
}

# Backup all databases
backup_database kb-postgres-primary 5432 kb_admin ${KB_POSTGRES_PASSWORD} kb_drug_rules drug_rules
backup_database kb-postgres-primary 5432 kb_admin ${KB_POSTGRES_PASSWORD} kb_terminology terminology
backup_database kb-postgres-primary 5432 kb_admin ${KB_POSTGRES_PASSWORD} kb_ddi ddi
backup_database kb-postgres-primary 5432 kb_admin ${KB_POSTGRES_PASSWORD} kb_formulary formulary
backup_database kb-postgres-governance 5432 governance_admin ${GOVERNANCE_POSTGRES_PASSWORD} clinical_governance governance
backup_database kb-postgres-timescale 5432 safety_admin ${SAFETY_POSTGRES_PASSWORD} kb_patient_safety safety

echo "All backups completed at $(date)"
```

### Step 4: Connection Configuration

```yaml
# .env.docker-postgres
# Docker PostgreSQL Configuration for KB Services

# Primary PostgreSQL
KB_POSTGRES_HOST=localhost
KB_POSTGRES_PORT=5433
KB_POSTGRES_USER=kb_admin
KB_POSTGRES_PASSWORD=kb_secure_password_2025

# Individual service connections
DRUG_RULES_DB_URL=postgresql://drug_rules_user:drug_rules_pass_2025@localhost:5433/kb_drug_rules?sslmode=disable
TERMINOLOGY_DB_URL=postgresql://terminology_user:terminology_pass_2025@localhost:5433/kb_terminology?sslmode=disable
DDI_DB_URL=postgresql://ddi_user:ddi_pass_2025@localhost:5433/kb_ddi?sslmode=disable
FORMULARY_DB_URL=postgresql://formulary_user:formulary_pass_2025@localhost:5433/kb_formulary?sslmode=disable

# Governance PostgreSQL
GOVERNANCE_DB_URL=postgresql://governance_admin:governance_secure_2025@localhost:5434/clinical_governance?sslmode=disable

# TimescaleDB
SAFETY_DB_URL=postgresql://safety_admin:safety_secure_2025@localhost:5435/kb_patient_safety?sslmode=disable

# PgAdmin
PGADMIN_URL=http://localhost:5050
PGADMIN_EMAIL=admin@kbservices.local
PGADMIN_PASSWORD=pgadmin_2025
```

---

## 🔧 Service-Specific Docker PostgreSQL Configurations

### KB-1: Drug Rules Service
```go
// kb-drug-rules/internal/database/connection.go
package database

import (
    "fmt"
    "os"
    "time"
    
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
)

func InitDockerPostgres() (*gorm.DB, error) {
    dsn := os.Getenv("DRUG_RULES_DB_URL")
    if dsn == "" {
        // Fallback to constructed DSN
        dsn = fmt.Sprintf(
            "host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
            getEnvOrDefault("DB_HOST", "localhost"),
            getEnvOrDefault("DB_PORT", "5433"),
            getEnvOrDefault("DB_USER", "drug_rules_user"),
            getEnvOrDefault("DB_PASSWORD", "drug_rules_pass_2025"),
            getEnvOrDefault("DB_NAME", "kb_drug_rules"),
        )
    }
    
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Info),
        NowFunc: func() time.Time {
            return time.Now().UTC()
        },
        PrepareStmt: true,
    })
    
    if err != nil {
        return nil, fmt.Errorf("failed to connect to Docker PostgreSQL: %w", err)
    }
    
    // Configure connection pool
    sqlDB, err := db.DB()
    if err != nil {
        return nil, err
    }
    
    sqlDB.SetMaxIdleConns(10)
    sqlDB.SetMaxOpenConns(100)
    sqlDB.SetConnMaxLifetime(time.Hour)
    
    // Run migrations
    if err := db.AutoMigrate(&DrugRulePack{}); err != nil {
        return nil, fmt.Errorf("failed to run migrations: %w", err)
    }
    
    return db, nil
}
```

### KB-7: Terminology Service with Full-Text Search
```sql
-- Additional setup for terminology service in Docker PostgreSQL

\c kb_terminology;

-- Configure full-text search
ALTER DATABASE kb_terminology SET default_text_search_config = 'pg_catalog.english';

-- Create custom text search configuration for medical terms
CREATE TEXT SEARCH CONFIGURATION medical_english ( COPY = english );

-- Add medical abbreviations dictionary
CREATE TEXT SEARCH DICTIONARY medical_abbrev (
    TEMPLATE = synonym,
    SYNONYMS = medical_synonyms
);

-- Create synonym file (needs to be mounted as volume)
-- File: /var/lib/postgresql/data/pg_config/medical_synonyms.syn
-- Content:
-- htn hypertension
-- dm diabetes mellitus
-- ckd chronic kidney disease
-- mi myocardial infarction
-- chf congestive heart failure

ALTER TEXT SEARCH CONFIGURATION medical_english
    ALTER MAPPING FOR asciiword, asciihword, hword_asciipart, word, hword, hword_part
    WITH medical_abbrev, english_stem;

-- Create GIN indexes for fast full-text search
CREATE INDEX idx_terminology_fts ON terminology_concepts 
    USING GIN(to_tsvector('medical_english', preferred_term || ' ' || COALESCE(definition, '')));

CREATE INDEX idx_drug_terminology_fts ON drug_terminology 
    USING GIN(to_tsvector('medical_english', drug_name || ' ' || COALESCE(generic_name, '')));
```

---

## 🚀 Docker Deployment Commands

### Starting the Services

```bash
# 1. Create necessary directories
mkdir -p ./init-scripts/{primary,governance,timescale}
mkdir -p ./backups/{primary,governance,timescale}
mkdir -p ./backup-scripts
mkdir -p ./pgadmin

# 2. Copy initialization scripts to appropriate directories
cp primary-init.sql ./init-scripts/primary/01-create-databases.sql
cp governance-init.sql ./init-scripts/governance/01-evidence-envelope.sql
cp timescale-init.sql ./init-scripts/timescale/01-patient-safety.sql

# 3. Start PostgreSQL containers
docker-compose -f docker-compose.postgres-kb.yml up -d

# 4. Wait for databases to be ready
docker-compose -f docker-compose.postgres-kb.yml exec kb-postgres-primary pg_isready -U kb_admin
docker-compose -f docker-compose.postgres-kb.yml exec kb-postgres-governance pg_isready -U governance_admin
docker-compose -f docker-compose.postgres-kb.yml exec kb-postgres-timescale pg_isready -U safety_admin

# 5. Verify databases were created
docker-compose -f docker-compose.postgres-kb.yml exec kb-postgres-primary psql -U kb_admin -c "\l"

# 6. Test connections
docker-compose -f docker-compose.postgres-kb.yml exec kb-postgres-primary \
    psql -U drug_rules_user -d kb_drug_rules -c "SELECT version();"
```

### Health Checks

```bash
#!/bin/bash
# scripts/check-postgres-health.sh

echo "Checking Docker PostgreSQL Health Status..."

# Function to check database
check_db() {
    local container=$1
    local user=$2
    local db=$3
    local port=$4
    
    echo -n "Checking ${db} on ${container}:${port}... "
    
    if docker exec ${container} pg_isready -U ${user} -d ${db} > /dev/null 2>&1; then
        echo "✅ OK"
        
        # Get connection count
        conn_count=$(docker exec ${container} psql -U ${user} -d ${db} -t -c "SELECT count(*) FROM pg_stat_activity WHERE datname='${db}';" 2>/dev/null | tr -d ' ')
        echo "  Active connections: ${conn_count}"
        
        # Get database size
        size=$(docker exec ${container} psql -U ${user} -d ${db} -t -c "SELECT pg_size_pretty(pg_database_size('${db}'));" 2>/dev/null | tr -d ' ')
        echo "  Database size: ${size}"
    else
        echo "❌ FAILED"
        return 1
    fi
}

# Check all databases
check_db kb_postgres_primary drug_rules_user kb_drug_rules 5433
check_db kb_postgres_primary terminology_user kb_terminology 5433
check_db kb_postgres_primary ddi_user kb_ddi 5433
check_db kb_postgres_primary formulary_user kb_formulary 5433
check_db kb_postgres_governance governance_admin clinical_governance 5434
check_db kb_postgres_timescale safety_admin kb_patient_safety 5435

echo "Health check complete!"
```

---

## 🔍 Monitoring Docker PostgreSQL

### Prometheus Metrics Export

```yaml
# docker-compose.postgres-monitoring.yml
version: '3.8'

services:
  postgres-exporter-primary:
    image: prometheuscommunity/postgres-exporter:latest
    container_name: postgres_exporter_primary
    environment:
      DATA_SOURCE_NAME: "postgresql://kb_admin:kb_secure_password_2025@kb-postgres-primary:5432/postgres?sslmode=disable"
    ports:
      - "9187:9187"
    networks:
      - kb_network
    depends_on:
      - kb-postgres-primary

  postgres-exporter-governance:
    image: prometheuscommunity/postgres-exporter:latest
    container_name: postgres_exporter_governance
    environment:
      DATA_SOURCE_NAME: "postgresql://governance_admin:governance_secure_2025@kb-postgres-governance:5432/clinical_governance?sslmode=disable"
    ports:
      - "9188:9187"
    networks:
      - kb_network
    depends_on:
      - kb-postgres-governance

  postgres-exporter-timescale:
    image: prometheuscommunity/postgres-exporter:latest
    container_name: postgres_exporter_timescale
    environment:
      DATA_SOURCE_NAME: "postgresql://safety_admin:safety_secure_2025@kb-postgres-timescale:5432/kb_patient_safety?sslmode=disable"
    ports:
      - "9189:9187"
    networks:
      - kb_network
    depends_on:
      - kb-postgres-timescale
```

### Grafana Dashboard Configuration

```json
{
  "dashboard": {
    "title": "KB Services Docker PostgreSQL",
    "panels": [
      {
        "title": "Database Size",
        "targets": [{
          "expr": "pg_database_size_bytes{datname=~\"kb_.*|clinical_governance\"}"
        }]
      },
      {
        "title": "Active Connections",
        "targets": [{
          "expr": "pg_stat_database_numbackends{datname=~\"kb_.*|clinical_governance\"}"
        }]
      },
      {
        "title": "Transaction Rate",
        "targets": [{
          "expr": "rate(pg_stat_database_xact_commit{datname=~\"kb_.*\"}[5m])"
        }]
      },
      {
        "title": "Cache Hit Ratio",
        "targets": [{
          "expr": "pg_stat_database_blks_hit{datname=~\"kb_.*\"} / (pg_stat_database_blks_hit{datname=~\"kb_.*\"} + pg_stat_database_blks_read{datname=~\"kb_.*\"})"
        }]
      }
    ]
  }
}
```

---

## 🛠️ Maintenance Operations

### Backup and Restore

```bash
# Manual backup
docker exec kb_postgres_primary pg_dump -U drug_rules_user kb_drug_rules | gzip > kb_drug_rules_backup_$(date +%Y%m%d).sql.gz

# Restore from backup
gunzip < kb_drug_rules_backup_20250115.sql.gz | docker exec -i kb_postgres_primary psql -U drug_rules_user -d kb_drug_rules

# Automated backup all databases
docker exec kb_postgres_backup /scripts/backup-all.sh
```

### Performance Tuning

```sql
-- Connect to specific database
docker exec -it kb_postgres_primary psql -U kb_admin -d kb_drug_rules

-- Analyze query performance
EXPLAIN (ANALYZE, BUFFERS) 
SELECT * FROM drug_rule_packs 
WHERE drug_id = 'metformin' 
AND version = '2.1.0';

-- Update statistics
ANALYZE drug_rule_packs;

-- Check index usage
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes
ORDER BY idx_scan DESC;

-- Find slow queries
SELECT 
    query,
    calls,
    total_time,
    mean_time,
    max_time
FROM pg_stat_statements
WHERE mean_time > 10
ORDER BY mean_time DESC
LIMIT 10;
```

### Container Management

```bash
# View logs
docker-compose -f docker-compose.postgres-kb.yml logs -f kb-postgres-primary

# Restart container
docker-compose -f docker-compose.postgres-kb.yml restart kb-postgres-primary

# Scale for development/testing
docker-compose -f docker-compose.postgres-kb.yml up -d --scale kb-postgres-primary=2

# Clean up volumes (CAUTION: Deletes data)
docker-compose -f docker-compose.postgres-kb.yml down -v
```

---

## 📋 Docker PostgreSQL Checklist

### Initial Setup
- [ ] Docker and Docker Compose installed
- [ ] Directory structure created
- [ ] Environment variables configured
- [ ] Initialization scripts in place
- [ ] docker-compose.postgres-kb.yml created

### Container Deployment  
- [ ] PostgreSQL containers started
- [ ] All databases created successfully
- [ ] User permissions configured
- [ ] Extensions enabled
- [ ] Health checks passing

### Data Initialization
- [ ] Evidence Envelope schema created
- [ ] Partitions configured
- [ ] Indexes created
- [ ] TimescaleDB hypertables set up
- [ ] Initial data loaded (if applicable)

### Backup & Recovery
- [ ] Backup scripts configured
- [ ] Backup schedule set
- [ ] Test restore procedure
- [ ] Backup retention policy active

### Monitoring
- [ ] PostgreSQL exporters running
- [ ] Prometheus scraping metrics
- [ ] Grafana dashboards configured
- [ ] Alerts configured

### Security
- [ ] Strong passwords set
- [ ] Network isolation configured  
- [ ] SSL/TLS configured (for production)
- [ ] Audit logging enabled
- [ ] Access controls verified

### Performance
- [ ] Connection pooling configured
- [ ] Query optimization completed
- [ ] Indexes optimized
- [ ] Statistics updated
- [ ] Vacuum schedule set

---

## 🚨 Troubleshooting Docker PostgreSQL

### Common Issues and Solutions

| Issue | Solution |
|-------|----------|
| Container won't start | Check logs: `docker logs kb_postgres_primary` |
| Connection refused | Verify port mapping and firewall rules |
| Disk space issues | Check volume usage: `docker system df` |
| Slow queries | Run `ANALYZE` and check indexes |
| High memory usage | Adjust shared_buffers and work_mem |
| Backup failures | Check permissions and disk space |
| Replication lag | Monitor wal_sender and wal_receiver processes |

### Performance Optimization Tips

1. **Memory Configuration**
   - shared_buffers: 25% of RAM
   - effective_cache_size: 50-75% of RAM
   - work_mem: RAM / max_connections / 2

2. **Connection Pooling**
   - Use PgBouncer for production
   - Configure appropriate pool sizes per service

3. **Indexing Strategy**
   - Create indexes for foreign keys
   - Use partial indexes where appropriate
   - Consider BRIN indexes for time-series data

4. **Monitoring**
   - Enable pg_stat_statements
   - Use pg_stat_activity for active queries
   - Monitor checkpoint frequency

---

## 📚 References

- [PostgreSQL Docker Official Image](https://hub.docker.com/_/postgres)
- [TimescaleDB Docker Documentation](https://docs.timescale.com/install/latest/docker/)
- [PostgreSQL Performance Tuning](https://wiki.postgresql.org/wiki/Tuning_Your_PostgreSQL_Server)
- [Docker Compose Networking](https://docs.docker.com/compose/networking/)

---

*This Docker PostgreSQL setup guide ensures reliable, scalable, and maintainable database infrastructure for your Knowledge Base services.*

**Document Version**: 1.0.0  
**Last Updated**: 2025-01-15  
**Status**: Ready for Implementation