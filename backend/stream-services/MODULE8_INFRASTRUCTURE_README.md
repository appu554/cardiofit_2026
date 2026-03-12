# Module 8 Storage Infrastructure

Comprehensive Docker Compose infrastructure for Module 8 storage projectors including MongoDB, Elasticsearch, ClickHouse, and Redis.

## Architecture Overview

```
Stream Processing Pipeline
        ↓
┌───────────────────────────────────────────┐
│     Module 8 Storage Projectors           │
├───────────────────────────────────────────┤
│  MongoDB         → Document Storage        │
│  Elasticsearch   → Search & Analytics      │
│  ClickHouse      → Time-Series Analytics   │
│  Redis           → Caching & Real-time     │
└───────────────────────────────────────────┘
```

## Infrastructure Services

### 1. MongoDB (Port 27017)
- **Purpose**: Document-oriented storage for clinical data
- **Image**: `mongo:7`
- **Database**: `module8_clinical`
- **Authentication**: Disabled (development mode)
- **Resources**: 2 CPU cores, 2GB RAM
- **Collections**:
  - `patients` - Patient demographic and clinical records
  - `observations` - Clinical observations and vital signs
  - `encounters` - Patient encounters and visits
  - `medications` - Medication orders and administration
  - `alerts` - Clinical alerts and notifications
  - `clinical_events` - Processed clinical events

### 2. Elasticsearch (Ports 9200, 9300)
- **Purpose**: Full-text search and analytics engine
- **Image**: `elasticsearch:8.11.0`
- **Cluster**: `module8-clinical-cluster`
- **Mode**: Single-node (development)
- **Security**: Disabled (development mode)
- **Memory**: 2GB heap size
- **Resources**: 3 CPU cores, 4GB RAM
- **Indices**:
  - `clinical_events_*` - Time-based clinical event indices
  - `patient_search` - Patient search index
  - `alert_search` - Alert search index

### 3. ClickHouse (Ports 8123 HTTP, 9000 Native)
- **Purpose**: Analytics and time-series data storage
- **Image**: `clickhouse/clickhouse-server:23.11`
- **Database**: `module8_analytics`
- **Credentials**: `module8_user` / `module8_password`
- **Resources**: 2 CPU cores, 3GB RAM
- **Tables**:
  - `patient_events` - All patient events with temporal ordering
  - `vital_signs` - Vital sign measurements over time

### 4. Redis (Port 6379)
- **Purpose**: Caching and real-time data storage
- **Image**: `redis:7-alpine`
- **Persistence**: AOF enabled
- **Max Memory**: 1GB with LRU eviction
- **Resources**: 1 CPU core, 1.5GB RAM

## Quick Start

### Starting Infrastructure

```bash
# Start all services
./manage-module8-infrastructure.sh start

# Initialize databases with schema
./manage-module8-infrastructure.sh init

# Check status
./manage-module8-infrastructure.sh status
```

### Stopping Infrastructure

```bash
# Stop all services (preserves data)
./manage-module8-infrastructure.sh stop

# Stop and remove all data
./manage-module8-infrastructure.sh clean
```

## Management Commands

### Basic Operations

```bash
# Start services
./manage-module8-infrastructure.sh start

# Stop services
./manage-module8-infrastructure.sh stop

# Restart services
./manage-module8-infrastructure.sh restart

# Check status
./manage-module8-infrastructure.sh status

# Check health
./manage-module8-infrastructure.sh health
```

### Monitoring

```bash
# View all logs
./manage-module8-infrastructure.sh logs

# View specific service logs
./manage-module8-infrastructure.sh logs mongodb
./manage-module8-infrastructure.sh logs elasticsearch
./manage-module8-infrastructure.sh logs clickhouse
./manage-module8-infrastructure.sh logs redis
```

### Database Management

```bash
# Show connection URLs
./manage-module8-infrastructure.sh urls

# Initialize databases
./manage-module8-infrastructure.sh init

# Clean all data (WARNING: destructive)
./manage-module8-infrastructure.sh clean
```

## Connection Information

### MongoDB

```bash
# Connection URL
mongodb://localhost:27017

# Database
module8_clinical

# Shell Access
docker exec -it module8-mongodb mongosh

# Python Connection
from pymongo import MongoClient
client = MongoClient("mongodb://localhost:27017")
db = client.module8_clinical
```

### Elasticsearch

```bash
# HTTP API
http://localhost:9200

# Cluster Health
curl http://localhost:9200/_cluster/health

# List Indices
curl http://localhost:9200/_cat/indices?v

# Python Connection
from elasticsearch import Elasticsearch
es = Elasticsearch(["http://localhost:9200"])
```

### ClickHouse

```bash
# HTTP Interface
http://localhost:8123

# Native Client Port
localhost:9000

# Credentials
User: module8_user
Password: module8_password
Database: module8_analytics

# Client Access
docker exec -it module8-clickhouse clickhouse-client

# Python Connection
from clickhouse_driver import Client
client = Client(
    host='localhost',
    port=9000,
    user='module8_user',
    password='module8_password',
    database='module8_analytics'
)
```

### Redis

```bash
# Connection
localhost:6379

# CLI Access
docker exec -it module8-redis redis-cli

# Python Connection
import redis
r = redis.Redis(host='localhost', port=6379, decode_responses=True)
```

## Health Checks

All services include automatic health checks:

- **MongoDB**: Ping command every 10s
- **Elasticsearch**: Cluster health check every 15s
- **ClickHouse**: HTTP ping every 10s
- **Redis**: Ping command every 5s

Check health status:
```bash
./manage-module8-infrastructure.sh health
```

## Resource Limits

### CPU Allocation
- MongoDB: 0.5-2.0 cores
- Elasticsearch: 1.0-3.0 cores
- ClickHouse: 0.5-2.0 cores
- Redis: 0.25-1.0 core

### Memory Allocation
- MongoDB: 512MB-2GB
- Elasticsearch: 2GB-4GB (2GB heap)
- ClickHouse: 512MB-3GB
- Redis: 256MB-1.5GB (1GB max memory)

## Persistent Volumes

All data is persisted in Docker volumes:

```bash
# MongoDB volumes
module8-mongodb-data
module8-mongodb-config

# Elasticsearch volumes
module8-elasticsearch-data
module8-elasticsearch-logs

# ClickHouse volumes
module8-clickhouse-data
module8-clickhouse-logs

# Redis volumes
module8-redis-data
```

View volumes:
```bash
docker volume ls | grep module8
```

## Network Configuration

All services run in the `module8-network` bridge network:
- Subnet: 172.28.0.0/16
- Driver: bridge
- Inter-service communication enabled

## Existing Infrastructure

Module 8 works alongside existing services:
- **PostgreSQL**: Container a2f55d83b1fa
- **Neo4j**: Container e8b3df4d8a02
- **InfluxDB**: Container 8502fd5d078d

These are managed separately and not part of this Docker Compose stack.

## Troubleshooting

### Services Not Starting

```bash
# Check Docker status
docker ps -a | grep module8

# Check logs
./manage-module8-infrastructure.sh logs

# Restart services
./manage-module8-infrastructure.sh restart
```

### Port Conflicts

If ports are already in use:
1. Check existing services: `lsof -i :27017`
2. Stop conflicting services
3. Or modify ports in `docker-compose.module8-infrastructure.yml`

### Memory Issues

Elasticsearch requires significant memory:
```bash
# Increase Docker memory limit (Docker Desktop)
# Settings → Resources → Memory: 8GB+

# Or reduce heap size in docker-compose.yml
ES_JAVA_OPTS=-Xms1g -Xmx1g
```

### Permission Issues

```bash
# Fix volume permissions
docker-compose -f docker-compose.module8-infrastructure.yml down -v
./manage-module8-infrastructure.sh start
```

## Production Considerations

**WARNING**: This configuration is for DEVELOPMENT ONLY.

For production:

1. **Enable Security**:
   - MongoDB: Enable authentication
   - Elasticsearch: Enable X-Pack security
   - ClickHouse: Strong passwords
   - Redis: Enable AUTH

2. **Clustering**:
   - Elasticsearch: Multi-node cluster
   - ClickHouse: Distributed tables
   - MongoDB: Replica sets

3. **Backups**:
   - Regular snapshots
   - Point-in-time recovery
   - Disaster recovery plan

4. **Monitoring**:
   - Prometheus metrics
   - Grafana dashboards
   - Alert configuration

5. **Resource Limits**:
   - Production-grade hardware
   - SSD storage
   - Network optimization

## Integration with Stream Services

### Stage 1 (Java Validation)
```java
// MongoDB connection in Stage 1
MongoClient mongoClient = MongoClients.create("mongodb://localhost:27017");
MongoDatabase database = mongoClient.getDatabase("module8_clinical");
```

### Stage 2 (Python FHIR Transformation)
```python
# Multiple storage connections
from pymongo import MongoClient
from elasticsearch import Elasticsearch
from clickhouse_driver import Client
import redis

# MongoDB for documents
mongo = MongoClient("mongodb://localhost:27017")

# Elasticsearch for search
es = Elasticsearch(["http://localhost:9200"])

# ClickHouse for analytics
clickhouse = Client(host='localhost', user='module8_user', password='module8_password')

# Redis for caching
cache = redis.Redis(host='localhost', port=6379)
```

## Performance Optimization

### MongoDB
```javascript
// Create indices for common queries
db.patients.createIndex({ "patient_id": 1 })
db.observations.createIndex({ "patient_id": 1, "timestamp": -1 })
db.encounters.createIndex({ "encounter_id": 1 })
```

### Elasticsearch
```bash
# Optimize for bulk inserts
curl -X PUT "localhost:9200/clinical_events/_settings" -H 'Content-Type: application/json' -d'
{
  "index": {
    "refresh_interval": "30s",
    "number_of_replicas": 0
  }
}
'
```

### ClickHouse
```sql
-- Optimize table for time-series queries
OPTIMIZE TABLE module8_analytics.patient_events FINAL;
```

### Redis
```bash
# Monitor performance
docker exec -it module8-redis redis-cli INFO stats
docker exec -it module8-redis redis-cli MONITOR
```

## Monitoring and Metrics

### Service Metrics

```bash
# MongoDB stats
docker exec module8-mongodb mongosh --eval "db.serverStatus()"

# Elasticsearch cluster stats
curl http://localhost:9200/_cluster/stats?pretty

# ClickHouse system metrics
docker exec module8-clickhouse clickhouse-client --query "SELECT * FROM system.metrics"

# Redis info
docker exec module8-redis redis-cli INFO
```

### Container Metrics

```bash
# Resource usage
docker stats module8-mongodb module8-elasticsearch module8-clickhouse module8-redis

# Detailed stats
docker inspect module8-mongodb | jq '.[0].State'
```

## Backup and Restore

### MongoDB Backup
```bash
# Backup
docker exec module8-mongodb mongodump --out=/data/backup

# Restore
docker exec module8-mongodb mongorestore /data/backup
```

### Elasticsearch Snapshot
```bash
# Configure snapshot repository
curl -X PUT "localhost:9200/_snapshot/backup" -H 'Content-Type: application/json' -d'
{
  "type": "fs",
  "settings": {
    "location": "/usr/share/elasticsearch/backup"
  }
}
'

# Create snapshot
curl -X PUT "localhost:9200/_snapshot/backup/snapshot_1"
```

### ClickHouse Backup
```sql
-- Freeze table for backup
ALTER TABLE module8_analytics.patient_events FREEZE;
```

## Summary

The Module 8 infrastructure provides:

✅ **MongoDB** - Document storage with flexible schema
✅ **Elasticsearch** - Full-text search and analytics
✅ **ClickHouse** - High-performance time-series analytics
✅ **Redis** - Caching and real-time data
✅ **Health Checks** - Automatic service monitoring
✅ **Persistence** - Data preserved across restarts
✅ **Resource Limits** - Controlled resource usage
✅ **Easy Management** - Simple shell script interface

---

**Service URLs**:
- MongoDB: `mongodb://localhost:27017`
- Elasticsearch: `http://localhost:9200`
- ClickHouse: `http://localhost:8123` (HTTP), `localhost:9000` (Native)
- Redis: `localhost:6379`

**Management**: `./manage-module8-infrastructure.sh [command]`
**Documentation**: This file
**Compose File**: `docker-compose.module8-infrastructure.yml`
