# Module 8 Infrastructure - Quick Start Guide

Get the Module 8 storage infrastructure running in 5 minutes.

## Prerequisites

- Docker and Docker Compose installed
- Python 3.11+ (for testing)
- At least 8GB RAM available for Docker

## Step 1: Start Infrastructure (1 minute)

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/stream-services

# Start all services
./manage-module8-infrastructure.sh start
```

This starts:
- MongoDB (port 27017)
- Elasticsearch (ports 9200, 9300)
- ClickHouse (ports 8123, 9000)
- Redis (port 6379)

## Step 2: Check Status (30 seconds)

```bash
# Check health of all services
./manage-module8-infrastructure.sh health
```

You should see:
```
✓ MongoDB: Healthy (port 27017)
✓ Elasticsearch: Healthy - Status: green (ports 9200, 9300)
✓ ClickHouse: Healthy (ports 8123 HTTP, 9000 native)
✓ Redis: Healthy (port 6379)
```

## Step 3: Initialize Databases (30 seconds)

```bash
# Create collections, indices, and tables
./manage-module8-infrastructure.sh init
```

This creates:
- **MongoDB**: Collections for patients, observations, encounters, medications, alerts, clinical_events
- **Elasticsearch**: Index templates for clinical events
- **ClickHouse**: Tables for patient_events and vital_signs

## Step 4: Install Python Dependencies (1 minute)

```bash
# Install storage client libraries
pip install -r requirements-module8.txt
```

## Step 5: Test Everything (1 minute)

```bash
# Run comprehensive test suite
python test-module8-infrastructure.py
```

Expected output:
```
Module 8 Infrastructure Tests
============================================================

1. Connecting to all services...
✓ All services connected

============================================================
Testing MongoDB
============================================================
  Inserted document: 65abc123...
  Found document: test_mongo_1
  Updated document
  Document count: 1
  Deleted 1 document(s)
✓ MongoDB tests passed

[... similar for Elasticsearch, ClickHouse, Redis ...]

============================================================
Final Health Check
============================================================
✓ mongodb: Healthy
✓ elasticsearch: Healthy
✓ clickhouse: Healthy
✓ redis: Healthy

✓ ALL TESTS PASSED
```

## Quick Access

### MongoDB Shell
```bash
docker exec -it module8-mongodb mongosh
use module8_clinical
db.patients.find()
```

### Elasticsearch
```bash
# Check cluster health
curl http://localhost:9200/_cluster/health?pretty

# List indices
curl http://localhost:9200/_cat/indices?v
```

### ClickHouse Client
```bash
docker exec -it module8-clickhouse clickhouse-client

# Query tables
SELECT * FROM module8_analytics.patient_events LIMIT 10;
```

### Redis CLI
```bash
docker exec -it module8-redis redis-cli

# Test commands
PING
KEYS *
```

## Using in Python Code

```python
from module8_storage_clients import Module8Storage

# Create storage instance
with Module8Storage() as storage:
    # MongoDB - Document storage
    patient = {
        "patient_id": "P123",
        "name": "John Doe",
        "age": 45
    }
    storage.mongo.patients.insert_one(patient)

    # Elasticsearch - Search
    storage.es.index(
        index="clinical_events",
        document={"event": "admission"}
    )

    # ClickHouse - Analytics
    events = storage.clickhouse.get_patient_events("P123")

    # Redis - Caching
    storage.redis.cache_patient("P123", patient, ttl=3600)
```

## Common Commands

### Start/Stop
```bash
./manage-module8-infrastructure.sh start    # Start all services
./manage-module8-infrastructure.sh stop     # Stop all services
./manage-module8-infrastructure.sh restart  # Restart all services
```

### Monitoring
```bash
./manage-module8-infrastructure.sh status   # Show status
./manage-module8-infrastructure.sh health   # Health check
./manage-module8-infrastructure.sh logs     # View all logs
./manage-module8-infrastructure.sh logs mongodb  # View specific service
```

### Information
```bash
./manage-module8-infrastructure.sh urls     # Show connection URLs
./manage-module8-infrastructure.sh help     # Show all commands
```

## Troubleshooting

### Services Won't Start

```bash
# Check if ports are in use
lsof -i :27017
lsof -i :9200
lsof -i :8123
lsof -i :6379

# Check Docker resources
docker system df

# Restart with fresh volumes
./manage-module8-infrastructure.sh clean
./manage-module8-infrastructure.sh start
```

### Elasticsearch Memory Error

```bash
# Increase Docker memory in Docker Desktop
# Settings → Resources → Memory: 8GB+

# Or reduce heap size in docker-compose.module8-infrastructure.yml
ES_JAVA_OPTS=-Xms1g -Xmx1g  # Change from 2g to 1g
```

### Connection Refused

```bash
# Wait for services to be ready (especially Elasticsearch)
./manage-module8-infrastructure.sh health

# Check if containers are running
docker ps | grep module8
```

## Next Steps

1. **Integrate with Stream Services**: Connect Stage 1 and Stage 2 to storage
2. **Add Custom Indices**: Create application-specific Elasticsearch indices
3. **Optimize Queries**: Add MongoDB indices for common queries
4. **Configure Monitoring**: Set up Grafana dashboards for metrics
5. **Production Setup**: Enable security and clustering

## Service URLs Summary

| Service | URL | Purpose |
|---------|-----|---------|
| MongoDB | `mongodb://localhost:27017` | Document storage |
| Elasticsearch | `http://localhost:9200` | Search & analytics |
| ClickHouse HTTP | `http://localhost:8123` | Analytics queries |
| ClickHouse Native | `localhost:9000` | High-performance queries |
| Redis | `localhost:6379` | Caching |

## Files Created

- `docker-compose.module8-infrastructure.yml` - Infrastructure definition
- `manage-module8-infrastructure.sh` - Management script
- `module8_storage_clients.py` - Python client library
- `test-module8-infrastructure.py` - Test suite
- `requirements-module8.txt` - Python dependencies
- `MODULE8_INFRASTRUCTURE_README.md` - Comprehensive documentation
- `MODULE8_QUICKSTART.md` - This file

## Support

For detailed documentation, see: `MODULE8_INFRASTRUCTURE_README.md`

For issues:
1. Check logs: `./manage-module8-infrastructure.sh logs [service]`
2. Verify health: `./manage-module8-infrastructure.sh health`
3. Review Docker resources: `docker stats`
