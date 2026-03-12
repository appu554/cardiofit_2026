# Configuration and Deployment Guide

This document provides comprehensive guidance for deploying and configuring the Neo4j Multi-KB Stream Manager in various environments.

## Table of Contents

- [System Requirements](#system-requirements)
- [Installation](#installation)
- [Configuration](#configuration)
- [Neo4j Setup](#neo4j-setup)
- [Deployment Scenarios](#deployment-scenarios)
- [Monitoring and Maintenance](#monitoring-and-maintenance)
- [Troubleshooting](#troubleshooting)

## System Requirements

### Minimum Requirements

#### Development Environment
- **Python**: 3.8 or higher
- **Neo4j**: 5.12+ (Community Edition)
- **Memory**: 8GB RAM
- **Storage**: 50GB available space
- **CPU**: 4 cores

#### Production Environment
- **Python**: 3.10 or higher (recommended)
- **Neo4j**: 5.15+ (Enterprise Edition recommended)
- **Memory**: 32GB RAM minimum, 64GB+ recommended
- **Storage**: 500GB+ SSD storage
- **CPU**: 8+ cores
- **Network**: Low-latency connection to Neo4j cluster

### Software Dependencies

See `requirements.txt` for complete Python dependency list. Key dependencies:

- `neo4j>=5.12.0` - Neo4j Python driver
- `neo4j-driver>=5.12.0` - Async Neo4j driver
- `loguru>=0.7.2` - Structured logging
- `pydantic>=2.5.0` - Data validation

## Installation

### From Source

```bash
# Clone the repository (if part of larger project)
cd /path/to/cardiofit/backend/shared-infrastructure/runtime-layer/neo4j-dual-stream

# Install dependencies
pip install -r ../requirements.txt

# Verify installation
python -c "from multi_kb_stream_manager import MultiKBStreamManager; print('Installation successful')"
```

### Docker Installation

```dockerfile
# Dockerfile
FROM python:3.11-slim

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Install Python dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY multi_kb_stream_manager.py .

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD python -c "from multi_kb_stream_manager import MultiKBStreamManager; print('OK')" || exit 1

EXPOSE 8000

CMD ["python", "-c", "print('Multi-KB Stream Manager container ready')"]
```

### Build and Run Docker Container

```bash
# Build the container
docker build -t cardiofit/multi-kb-stream-manager:latest .

# Run with environment variables
docker run -d \
  --name multi-kb-stream \
  -e NEO4J_URI=bolt://neo4j:7687 \
  -e NEO4J_USER=neo4j \
  -e NEO4J_PASSWORD=your_password \
  cardiofit/multi-kb-stream-manager:latest
```

## Configuration

### Environment Variables

```bash
# Neo4j Connection
export NEO4J_URI="bolt://localhost:7687"
export NEO4J_USER="neo4j"
export NEO4J_PASSWORD="your_secure_password"

# Connection Pool Settings
export NEO4J_MAX_POOL_SIZE="100"
export NEO4J_CONNECTION_TIMEOUT="30"

# Application Settings
export LOG_LEVEL="INFO"
export ENVIRONMENT="production"
export HEALTH_CHECK_INTERVAL="60"
```

### Configuration File

Create `config/multi_kb_config.yaml`:

```yaml
# Neo4j Configuration
neo4j:
  uri: "bolt://localhost:7687"
  user: "neo4j"
  password: "${NEO4J_PASSWORD}"  # Use environment variable
  pool_size: 100
  connection_timeout: 30
  encrypted: true
  trust: "TRUST_ALL_CERTIFICATES"  # For development only

# Knowledge Base Configuration
knowledge_bases:
  kb1_patient:
    enabled: true
    indexes:
      - "patient_id"
      - "patient_mrn"
    constraints:
      - "unique_patient_id"

  kb2_guidelines:
    enabled: true
    indexes:
      - "guideline_id"
      - "condition_category"

  kb3_drug_calculations:
    enabled: true
    indexes:
      - "drug_rxnorm"
      - "indication"

  kb4_safety_rules:
    enabled: true
    indexes:
      - "rule_category"
      - "severity"

  kb5_drug_interactions:
    enabled: true
    indexes:
      - "drug_combination"
      - "severity"

  kb6_evidence:
    enabled: true
    indexes:
      - "study_id"
      - "evidence_level"

  kb7_terminology:
    enabled: true
    indexes:
      - "concept_code"
      - "system"

  kb8_workflows:
    enabled: true
    indexes:
      - "workflow_id"
      - "status"

# Shared Streams Configuration
shared_streams:
  semantic_mesh:
    enabled: true
    cross_kb_indexing: true

  global_patient:
    enabled: true
    privacy_mode: "strict"

# Logging Configuration
logging:
  level: "INFO"
  format: "json"
  file: "/var/log/multi-kb-stream-manager.log"
  rotation: "daily"
  retention: "30 days"

# Health Check Configuration
health_check:
  interval: 60  # seconds
  timeout: 10   # seconds
  retry_count: 3

# Security Configuration
security:
  audit_logging: true
  data_lineage: true
  user_attribution: true
  encryption_at_rest: true
```

### Loading Configuration in Python

```python
import yaml
import os
from pathlib import Path

def load_config(config_path: str = "config/multi_kb_config.yaml") -> dict:
    """Load configuration with environment variable substitution"""

    config_file = Path(config_path)
    if not config_file.exists():
        raise FileNotFoundError(f"Configuration file not found: {config_path}")

    with open(config_file, 'r') as f:
        config_content = f.read()

    # Substitute environment variables
    import re
    env_vars = re.findall(r'\$\{([^}]+)\}', config_content)
    for var in env_vars:
        env_value = os.getenv(var)
        if env_value is None:
            raise ValueError(f"Required environment variable not set: {var}")
        config_content = config_content.replace(f"${{{var}}}", env_value)

    return yaml.safe_load(config_content)

# Usage
config = load_config()
manager = MultiKBStreamManager(config['neo4j'])
```

## Neo4j Setup

### Community Edition Setup

```bash
# Download and install Neo4j Community
wget https://neo4j.com/artifact.php?name=neo4j-community-5.15.0-unix.tar.gz
tar -xf neo4j-community-5.15.0-unix.tar.gz
cd neo4j-community-5.15.0

# Configure Neo4j
cat << EOF >> conf/neo4j.conf
# Basic configuration
server.default_listen_address=0.0.0.0
server.bolt.listen_address=:7687
server.http.listen_address=:7474

# Memory settings for healthcare workloads
server.memory.heap.initial_size=4G
server.memory.heap.max_size=8G
server.memory.pagecache.size=16G

# Performance tuning
db.tx_log.rotation.retention_policy=7 days
dbms.checkpoint.interval.time=15m
EOF

# Start Neo4j
bin/neo4j start
```

### Enterprise Edition Setup

```bash
# Configure for healthcare production use
cat << EOF >> conf/neo4j.conf
# Enterprise features
causal_clustering.minimum_core_cluster_size_at_formation=3
causal_clustering.minimum_core_cluster_size_at_runtime=3
causal_clustering.initial_discovery_members=neo4j-core-1:5000,neo4j-core-2:5000,neo4j-core-3:5000

# Security settings
dbms.security.auth_enabled=true
dbms.security.procedures.unrestricted=apoc.*

# LDAP integration (if required)
dbms.security.realms=ldap
dbms.security.ldap.authorization.use_system_account=true
dbms.security.ldap.host=ldap.example.com
dbms.security.ldap.port=389

# SSL/TLS configuration
dbms.ssl.policy.bolt.enabled=true
dbms.ssl.policy.bolt.base_directory=certificates/bolt
dbms.ssl.policy.bolt.client_auth=REQUIRE

# Audit logging for healthcare compliance
dbms.security.log_successful_authentication=true
dbms.logs.security.level=INFO
EOF
```

### Database Initialization

```python
# Initialize all KB streams
import asyncio
from multi_kb_stream_manager import MultiKBStreamManager

async def initialize_production_database():
    config = {
        'neo4j_uri': 'bolt://neo4j-cluster:7687',
        'neo4j_user': 'cardiofit_admin',
        'neo4j_password': os.getenv('NEO4J_PASSWORD')
    }

    manager = MultiKBStreamManager(config)

    print("Initializing all knowledge base streams...")
    success = await manager.initialize_all_streams()

    if not success:
        print("❌ Database initialization failed")
        return False

    print("✅ Database initialization completed")

    # Verify all streams are healthy
    health_status = await manager.health_check_all_streams()

    for kb_name, status in health_status.items():
        if status.get('healthy', False):
            print(f"✅ {kb_name}: Healthy")
        else:
            print(f"❌ {kb_name}: Unhealthy - {status.get('error', 'Unknown error')}")

    await manager.close()
    return success

# Run initialization
if __name__ == "__main__":
    asyncio.run(initialize_production_database())
```

## Deployment Scenarios

### Development Environment

```yaml
# docker-compose.dev.yml
version: '3.8'

services:
  neo4j:
    image: neo4j:5.15.0-community
    ports:
      - "7474:7474"  # HTTP
      - "7687:7687"  # Bolt
    environment:
      NEO4J_AUTH: neo4j/devpassword
      NEO4J_server_memory_heap_max__size: 4G
      NEO4J_server_memory_pagecache_size: 8G
    volumes:
      - neo4j_data:/data
      - neo4j_logs:/logs
    networks:
      - cardiofit_dev

  multi-kb-stream:
    build: .
    depends_on:
      - neo4j
    environment:
      NEO4J_URI: bolt://neo4j:7687
      NEO4J_USER: neo4j
      NEO4J_PASSWORD: devpassword
      LOG_LEVEL: DEBUG
    networks:
      - cardiofit_dev

volumes:
  neo4j_data:
  neo4j_logs:

networks:
  cardiofit_dev:
```

### Production Cluster Deployment

```yaml
# docker-compose.prod.yml
version: '3.8'

services:
  # Neo4j Cluster Core Servers
  neo4j-core-1:
    image: neo4j:5.15.0-enterprise
    environment:
      NEO4J_AUTH: ${NEO4J_USER}/${NEO4J_PASSWORD}
      NEO4J_causal__clustering_minimum__core__cluster__size__at__formation: 3
      NEO4J_causal__clustering_minimum__core__cluster__size__at__runtime: 3
      NEO4J_causal__clustering_initial__discovery__members: "neo4j-core-1:5000,neo4j-core-2:5000,neo4j-core-3:5000"
      NEO4J_causal__clustering_discovery__advertised__address: neo4j-core-1:5000
      NEO4J_causal__clustering_transaction__advertised__address: neo4j-core-1:6000
      NEO4J_causal__clustering_raft__advertised__address: neo4j-core-1:7000
      NEO4J_server_memory_heap_max__size: 16G
      NEO4J_server_memory_pagecache_size: 32G
    ports:
      - "7474:7474"
      - "7687:7687"
    volumes:
      - neo4j_core_1_data:/data
    networks:
      - cardiofit_cluster

  neo4j-core-2:
    image: neo4j:5.15.0-enterprise
    environment:
      NEO4J_AUTH: ${NEO4J_USER}/${NEO4J_PASSWORD}
      NEO4J_causal__clustering_minimum__core__cluster__size__at__formation: 3
      NEO4J_causal__clustering_minimum__core__cluster__size__at__runtime: 3
      NEO4J_causal__clustering_initial__discovery__members: "neo4j-core-1:5000,neo4j-core-2:5000,neo4j-core-3:5000"
      NEO4J_causal__clustering_discovery__advertised__address: neo4j-core-2:5000
      NEO4J_causal__clustering_transaction__advertised__address: neo4j-core-2:6000
      NEO4J_causal__clustering_raft__advertised__address: neo4j-core-2:7000
    volumes:
      - neo4j_core_2_data:/data
    networks:
      - cardiofit_cluster

  neo4j-core-3:
    image: neo4j:5.15.0-enterprise
    environment:
      NEO4J_AUTH: ${NEO4J_USER}/${NEO4J_PASSWORD}
      NEO4J_causal__clustering_minimum__core__cluster__size__at__formation: 3
      NEO4J_causal__clustering_minimum__core__cluster__size__at__runtime: 3
      NEO4J_causal__clustering_initial__discovery__members: "neo4j-core-1:5000,neo4j-core-2:5000,neo4j-core-3:5000"
      NEO4J_causal__clustering_discovery__advertised__address: neo4j-core-3:5000
      NEO4J_causal__clustering_transaction__advertised__address: neo4j-core-3:6000
      NEO4J_causal__clustering_raft__advertised__address: neo4j-core-3:7000
    volumes:
      - neo4j_core_3_data:/data
    networks:
      - cardiofit_cluster

  # Read Replicas for Analytics
  neo4j-replica-1:
    image: neo4j:5.15.0-enterprise
    environment:
      NEO4J_AUTH: ${NEO4J_USER}/${NEO4J_PASSWORD}
      NEO4J_causal__clustering_discovery__advertised__address: neo4j-replica-1:5000
      NEO4J_causal__clustering_initial__discovery__members: "neo4j-core-1:5000,neo4j-core-2:5000,neo4j-core-3:5000"
      NEO4J_server_mode: READ_REPLICA
    volumes:
      - neo4j_replica_1_data:/data
    networks:
      - cardiofit_cluster

volumes:
  neo4j_core_1_data:
  neo4j_core_2_data:
  neo4j_core_3_data:
  neo4j_replica_1_data:

networks:
  cardiofit_cluster:
```

### Kubernetes Deployment

```yaml
# k8s-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: multi-kb-stream-manager
  namespace: cardiofit
spec:
  replicas: 3
  selector:
    matchLabels:
      app: multi-kb-stream-manager
  template:
    metadata:
      labels:
        app: multi-kb-stream-manager
    spec:
      containers:
      - name: multi-kb-stream
        image: cardiofit/multi-kb-stream-manager:latest
        env:
        - name: NEO4J_URI
          value: "bolt://neo4j-cluster.cardiofit.svc.cluster.local:7687"
        - name: NEO4J_USER
          valueFrom:
            secretKeyRef:
              name: neo4j-credentials
              key: username
        - name: NEO4J_PASSWORD
          valueFrom:
            secretKeyRef:
              name: neo4j-credentials
              key: password
        resources:
          requests:
            memory: "1Gi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "1000m"
        livenessProbe:
          exec:
            command:
            - python
            - -c
            - "from multi_kb_stream_manager import MultiKBStreamManager; print('OK')"
          initialDelaySeconds: 30
          periodSeconds: 60
        readinessProbe:
          exec:
            command:
            - python
            - -c
            - "import asyncio; from multi_kb_stream_manager import MultiKBStreamManager; print('Ready')"
          initialDelaySeconds: 10
          periodSeconds: 30

---
apiVersion: v1
kind: Secret
metadata:
  name: neo4j-credentials
  namespace: cardiofit
type: Opaque
data:
  username: bmVvNGo=  # base64 encoded 'neo4j'
  password: <base64-encoded-password>
```

## Monitoring and Maintenance

### Health Check Script

```python
#!/usr/bin/env python3
"""
Health check script for Multi-KB Stream Manager
"""

import asyncio
import sys
import json
from datetime import datetime
from multi_kb_stream_manager import MultiKBStreamManager

async def comprehensive_health_check():
    config = {
        'neo4j_uri': os.getenv('NEO4J_URI', 'bolt://localhost:7687'),
        'neo4j_user': os.getenv('NEO4J_USER', 'neo4j'),
        'neo4j_password': os.getenv('NEO4J_PASSWORD')
    }

    manager = MultiKBStreamManager(config)

    health_report = {
        'timestamp': datetime.utcnow().isoformat(),
        'overall_status': 'unknown',
        'knowledge_bases': {},
        'performance_metrics': {}
    }

    try:
        # Check all KB streams
        kb_health = await manager.health_check_all_streams()
        health_report['knowledge_bases'] = kb_health

        # Calculate overall status
        all_healthy = all(status.get('healthy', False) for status in kb_health.values())
        health_report['overall_status'] = 'healthy' if all_healthy else 'unhealthy'

        # Performance metrics
        start_time = datetime.utcnow()

        # Test query performance
        test_results = await manager.query_kb_stream(
            'kb1',
            'PATIENT',
            'RETURN count(n) as node_count LIMIT 1',
            {}
        )

        query_time = (datetime.utcnow() - start_time).total_seconds()
        health_report['performance_metrics'] = {
            'query_response_time': query_time,
            'connection_status': 'ok'
        }

        # Output results
        print(json.dumps(health_report, indent=2))

        # Exit code for monitoring systems
        sys.exit(0 if all_healthy else 1)

    except Exception as e:
        health_report['overall_status'] = 'error'
        health_report['error'] = str(e)
        print(json.dumps(health_report, indent=2))
        sys.exit(2)

    finally:
        await manager.close()

if __name__ == "__main__":
    asyncio.run(comprehensive_health_check())
```

### Monitoring with Prometheus

```python
# prometheus_metrics.py
from prometheus_client import Counter, Histogram, Gauge, start_http_server
import time
import asyncio

class MultiKBMetrics:
    def __init__(self):
        # Query metrics
        self.query_duration = Histogram(
            'neo4j_query_duration_seconds',
            'Time spent on Neo4j queries',
            ['kb_name', 'stream_type', 'query_type']
        )

        self.query_total = Counter(
            'neo4j_queries_total',
            'Total number of queries',
            ['kb_name', 'stream_type', 'status']
        )

        # Health metrics
        self.kb_health_status = Gauge(
            'neo4j_kb_health_status',
            'Health status of knowledge bases (1=healthy, 0=unhealthy)',
            ['kb_name']
        )

        self.kb_node_count = Gauge(
            'neo4j_kb_node_count',
            'Number of nodes in each KB stream',
            ['kb_name', 'stream_type']
        )

        # Connection metrics
        self.connection_pool_usage = Gauge(
            'neo4j_connection_pool_usage',
            'Connection pool usage percentage'
        )

    def record_query(self, kb_name: str, stream_type: str, query_type: str, duration: float, success: bool):
        """Record query metrics"""
        self.query_duration.labels(
            kb_name=kb_name,
            stream_type=stream_type,
            query_type=query_type
        ).observe(duration)

        self.query_total.labels(
            kb_name=kb_name,
            stream_type=stream_type,
            status='success' if success else 'error'
        ).inc()

    async def update_health_metrics(self, manager: MultiKBStreamManager):
        """Update health-related metrics"""
        health_status = await manager.health_check_all_streams()

        for kb_name, status in health_status.items():
            # Health status
            self.kb_health_status.labels(kb_name=kb_name).set(
                1 if status.get('healthy', False) else 0
            )

            # Node counts
            if 'primary_nodes' in status:
                self.kb_node_count.labels(
                    kb_name=kb_name,
                    stream_type='primary'
                ).set(status['primary_nodes'])

            if 'semantic_nodes' in status:
                self.kb_node_count.labels(
                    kb_name=kb_name,
                    stream_type='semantic'
                ).set(status['semantic_nodes'])

# Start metrics server
metrics = MultiKBMetrics()
start_http_server(8000)  # Prometheus metrics endpoint
```

### Backup and Recovery

```bash
#!/bin/bash
# backup-script.sh

NEO4J_HOME="/var/lib/neo4j"
BACKUP_DIR="/backup/neo4j"
DATE=$(date +%Y%m%d_%H%M%S)

echo "Starting Neo4j backup - $DATE"

# Create backup directory
mkdir -p "$BACKUP_DIR/$DATE"

# Stop Neo4j for consistent backup (if using single instance)
# systemctl stop neo4j

# Perform backup using neo4j-admin
neo4j-admin database backup \
    --from=bolt://localhost:7687 \
    --database=neo4j \
    --to="$BACKUP_DIR/$DATE" \
    --verbose

# Restart Neo4j
# systemctl start neo4j

# Compress backup
cd "$BACKUP_DIR"
tar -czf "neo4j_backup_$DATE.tar.gz" "$DATE"
rm -rf "$DATE"

# Clean old backups (keep last 7 days)
find "$BACKUP_DIR" -name "neo4j_backup_*.tar.gz" -mtime +7 -delete

echo "Backup completed: neo4j_backup_$DATE.tar.gz"

# Upload to cloud storage (optional)
# aws s3 cp "neo4j_backup_$DATE.tar.gz" s3://cardiofit-backups/neo4j/
```

## Troubleshooting

### Common Issues and Solutions

#### Connection Issues

```python
# Debug connection problems
async def debug_connection():
    config = {
        'neo4j_uri': 'bolt://localhost:7687',
        'neo4j_user': 'neo4j',
        'neo4j_password': 'password'
    }

    try:
        manager = MultiKBStreamManager(config)
        # Test basic connectivity
        async with manager.driver.session() as session:
            result = await session.run("RETURN 1 as test")
            record = await result.single()
            print(f"Connection successful: {record['test']}")
    except Exception as e:
        print(f"Connection failed: {e}")
        # Check common issues
        if "authentication" in str(e).lower():
            print("Check Neo4j username/password")
        elif "connection refused" in str(e).lower():
            print("Check if Neo4j is running and accessible")
        elif "ssl" in str(e).lower():
            print("Check SSL/TLS configuration")

asyncio.run(debug_connection())
```

#### Performance Issues

```bash
# Check Neo4j query performance
# Connect to Neo4j browser (http://localhost:7474)
# Run these diagnostic queries:

# Check slow queries
CALL dbms.listQueries() YIELD query, elapsedTimeMillis
WHERE elapsedTimeMillis > 1000
RETURN query, elapsedTimeMillis
ORDER BY elapsedTimeMillis DESC;

# Check index usage
CALL db.indexes() YIELD name, state, populationPercent
WHERE state <> 'ONLINE'
RETURN name, state, populationPercent;

# Memory usage
CALL dbms.queryJmx("java.lang:type=Memory")
YIELD attributes
RETURN attributes.HeapMemoryUsage, attributes.NonHeapMemoryUsage;
```

#### Data Integrity Issues

```python
# Check cross-KB data consistency
async def validate_data_integrity(manager):
    integrity_issues = []

    # Check for orphaned references
    orphaned_refs = await manager.cross_kb_query(
        ['kb1', 'kb5'],
        """
        MATCH (p:Patient:KB1_PatientStream)
        WHERE size(p.current_medications) > 0
        UNWIND p.current_medications AS med

        OPTIONAL MATCH (i:Interaction:KB5_InteractionStream)
        WHERE i.drug1_name = med OR i.drug2_name = med

        WITH p, med, count(i) AS interaction_count
        WHERE interaction_count = 0

        RETURN p.mrn, med, 'no_interaction_data' AS issue
        """
    )

    integrity_issues.extend(orphaned_refs)

    # Check for missing patient data
    missing_patient_data = await manager.query_kb_stream(
        'kb1',
        'PATIENT',
        """
        WHERE n.mrn IS NULL OR n.name IS NULL
        RETURN n.id, 'missing_required_fields' AS issue
        """,
        {}
    )

    integrity_issues.extend(missing_patient_data)

    return integrity_issues

# Run integrity check
issues = await validate_data_integrity(manager)
if issues:
    print(f"Found {len(issues)} data integrity issues")
    for issue in issues[:10]:  # Show first 10
        print(f"  - {issue}")
```

### Support and Maintenance

For production deployments:

1. **Monitor Performance**: Set up Prometheus metrics and Grafana dashboards
2. **Regular Backups**: Implement automated backup procedures
3. **Health Checks**: Deploy health check endpoints for load balancers
4. **Log Monitoring**: Configure structured logging with appropriate retention
5. **Security Updates**: Keep Neo4j and Python dependencies updated
6. **Capacity Planning**: Monitor growth trends for proactive scaling

For additional support, refer to the CardioFit platform documentation or contact the infrastructure team.