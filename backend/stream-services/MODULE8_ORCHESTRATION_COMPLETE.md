# Module 8 Storage Projectors - Complete Orchestration Guide

## Overview

Complete orchestration system for all 8 Module 8 storage projectors with automated network configuration, health monitoring, and infrastructure management.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Kafka (Confluent Cloud)                   │
│              clinical.fhir.enriched.v1 Topic                 │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
         ┌────────────────────────────────────────┐
         │      Module 8 Storage Projectors       │
         │              (8 Services)               │
         └────────────────────────────────────────┘
                              │
          ┌───────────────────┴───────────────────┐
          │                                       │
    ┌─────▼──────┐                    ┌──────────▼────────┐
    │ Internal   │                    │  External         │
    │ Databases  │                    │  Databases        │
    │            │                    │                   │
    │ - MongoDB  │                    │ - PostgreSQL      │
    │ - Elastic  │                    │   (a2f55d83b1fa)  │
    │ - ClickHse │                    │ - InfluxDB        │
    │ - Redis    │                    │   (8502fd5d078d)  │
    │            │                    │ - Neo4j           │
    │            │                    │   (e8b3df4d8a02)  │
    │            │                    │ - Google FHIR     │
    └────────────┘                    └───────────────────┘
```

## Components

### Storage Projectors (8 Services)

| Service | Port | Purpose | Database |
|---------|------|---------|----------|
| **postgresql-projector** | 8050 | Structured queries and analytics | PostgreSQL (external) |
| **mongodb-projector** | 8051 | Document storage, flexible schemas | MongoDB (internal) |
| **elasticsearch-projector** | 8052 | Full-text search, analytics | Elasticsearch (internal) |
| **clickhouse-projector** | 8053 | High-performance analytics | ClickHouse (internal) |
| **influxdb-projector** | 8054 | Time-series metrics | InfluxDB (external) |
| **ups-projector** | 8055 | Universal persistence service | PostgreSQL (external) |
| **fhir-store-projector** | 8056 | Google Healthcare API | Google Cloud FHIR Store |
| **neo4j-graph-projector** | 8057 | Clinical knowledge graph | Neo4j (external) |

### Infrastructure Services

| Service | Port | Purpose |
|---------|------|---------|
| **MongoDB** | 27017 | Document database |
| **Elasticsearch** | 9200, 9300 | Search engine |
| **ClickHouse** | 8123, 9000 | Analytics database |
| **Redis** | 6379 | Cache and real-time data |

### External Containers

| Container | ID | Network IP | Purpose |
|-----------|----|-----------:|---------|
| **cardiofit-postgres-analytics** | a2f55d83b1fa | 172.21.0.4 | PostgreSQL analytics |
| **cardiofit-influxdb** | 8502fd5d078d | 172.21.0.3 | InfluxDB time-series |
| **neo4j** | e8b3df4d8a02 | Auto-detect | Neo4j graph database |

## Quick Start

### 1. Initial Setup

```bash
# Navigate to stream-services directory
cd /Users/apoorvabk/Downloads/cardiofit/backend/stream-services

# Configure network and detect container IPs
./configure-network-module8.sh

# Copy example environment file
cp .env.module8.example .env.module8

# Edit with your credentials
nano .env.module8
```

### 2. Required Configuration

Edit `.env.module8` with your credentials:

```bash
# Kafka (Confluent Cloud)
KAFKA_BOOTSTRAP_SERVERS=pkc-xxxxx.us-east-1.aws.confluent.cloud:9092
KAFKA_SASL_USERNAME=your-kafka-api-key
KAFKA_SASL_PASSWORD=your-kafka-api-secret

# PostgreSQL (auto-detected or manual)
POSTGRES_HOST=172.21.0.4
POSTGRES_PASSWORD=your-postgres-password

# InfluxDB (auto-detected or manual)
INFLUXDB_URL=http://172.21.0.3:8086
INFLUXDB_TOKEN=your-influxdb-token

# Neo4j (auto-detected or manual)
NEO4J_URI=bolt://neo4j-ip:7687
NEO4J_PASSWORD=your-neo4j-password

# Google FHIR Store
GOOGLE_PROJECT_ID=your-gcp-project-id
GOOGLE_CREDENTIALS_PATH=/path/to/google-credentials.json
```

### 3. Start All Services

```bash
# Start all 8 projectors + infrastructure
./start-module8-projectors.sh
```

This script will:
1. Check prerequisites (Docker, credentials)
2. Detect external container IPs
3. Create/verify network bridge
4. Start infrastructure services (MongoDB, Elasticsearch, ClickHouse, Redis)
5. Start all 8 projector services
6. Run health checks
7. Display service URLs

### 4. Verify Health

```bash
# Comprehensive health check
./health-check-module8.sh

# Check specific service
curl http://localhost:8050/health  # PostgreSQL projector
curl http://localhost:8051/health  # MongoDB projector
curl http://localhost:8052/health  # Elasticsearch projector
# ... etc
```

## Management Scripts

### Start Services

```bash
./start-module8-projectors.sh
```

**Features**:
- Prerequisite validation
- External container detection
- Network configuration
- Infrastructure startup
- Projector service startup
- Health verification
- Service URL display

### Stop Services

```bash
./stop-module8-projectors.sh
```

**Features**:
- Statistics collection before shutdown
- Graceful projector shutdown
- Optional infrastructure shutdown
- Optional container removal
- Optional volume cleanup (data deletion)
- Final status report

### Health Monitoring

```bash
# Interactive health check
./health-check-module8.sh

# Generate detailed report
./health-check-module8.sh  # Select 'y' when prompted
```

**Checks**:
- Projector service health
- Infrastructure service health
- External container status
- Database connectivity
- Kafka consumer lag
- Resource usage

### Log Viewing

```bash
# Follow logs from specific service
./logs-module8.sh -f postgresql-projector

# Show last 50 lines from MongoDB projector
./logs-module8.sh -n 50 mongodb-projector

# Search for errors in all services
./logs-module8.sh -a -s "error"

# Show only errors from FHIR Store projector
./logs-module8.sh -e fhir-store-projector

# Follow all projector logs
./logs-module8.sh -f -a

# Analyze logs
./logs-module8.sh --analyze postgresql-projector

# Export logs
./logs-module8.sh --export postgresql-projector
```

### Network Configuration

```bash
# Auto-detect IPs and configure network
./configure-network-module8.sh
```

**Features**:
- Detect external container IPs
- Test network connectivity
- Create module8-network bridge
- Connect external containers
- Update .env.module8 file
- Validate configuration

## Service Endpoints

### Projector Health Endpoints

```bash
# PostgreSQL Projector
http://localhost:8050/health
http://localhost:8050/metrics

# MongoDB Projector
http://localhost:8051/health
http://localhost:8051/metrics

# Elasticsearch Projector
http://localhost:8052/health
http://localhost:8052/metrics

# ClickHouse Projector
http://localhost:8053/health
http://localhost:8053/metrics

# InfluxDB Projector
http://localhost:8054/health
http://localhost:8054/metrics

# UPS Projector
http://localhost:8055/health
http://localhost:8055/metrics

# FHIR Store Projector
http://localhost:8056/health
http://localhost:8056/metrics

# Neo4j Graph Projector
http://localhost:8057/health
http://localhost:8057/metrics
```

### Infrastructure Endpoints

```bash
# MongoDB
mongodb://localhost:27017

# Elasticsearch
http://localhost:9200
http://localhost:9200/_cluster/health

# ClickHouse
http://localhost:8123/ping
http://localhost:8123

# Redis
redis://localhost:6379
```

## Docker Compose Commands

### Individual Service Control

```bash
# Start specific projector
docker-compose -f docker-compose.module8-complete.yml up -d postgresql-projector

# Stop specific projector
docker-compose -f docker-compose.module8-complete.yml stop postgresql-projector

# View logs
docker-compose -f docker-compose.module8-complete.yml logs -f postgresql-projector

# Restart service
docker-compose -f docker-compose.module8-complete.yml restart postgresql-projector
```

### Infrastructure Control

```bash
# Start infrastructure only
docker-compose -f docker-compose.module8-complete.yml up -d mongodb elasticsearch clickhouse redis

# Stop infrastructure
docker-compose -f docker-compose.module8-complete.yml stop mongodb elasticsearch clickhouse redis
```

### Complete System Control

```bash
# Start everything
docker-compose -f docker-compose.module8-complete.yml up -d

# Stop everything
docker-compose -f docker-compose.module8-complete.yml stop

# Remove containers (keeps volumes)
docker-compose -f docker-compose.module8-complete.yml down

# Remove containers and volumes (data deletion!)
docker-compose -f docker-compose.module8-complete.yml down -v
```

## Network Configuration

### Network Details

```bash
# Network name: module8-network
# Subnet: 172.28.0.0/16
# Driver: bridge

# View network details
docker network inspect module8-network

# List containers on network
docker network inspect module8-network | grep -A 5 "Containers"
```

### External Container Connection

```bash
# Connect external containers to module8-network
docker network connect module8-network a2f55d83b1fa  # PostgreSQL
docker network connect module8-network 8502fd5d078d  # InfluxDB
docker network connect module8-network e8b3df4d8a02  # Neo4j

# Disconnect external containers
docker network disconnect module8-network a2f55d83b1fa
docker network disconnect module8-network 8502fd5d078d
docker network disconnect module8-network e8b3df4d8a02
```

## Troubleshooting

### Service Not Starting

1. **Check logs**:
   ```bash
   ./logs-module8.sh -e [service-name]
   ```

2. **Verify environment variables**:
   ```bash
   cat .env.module8 | grep -v "^#"
   ```

3. **Check database connectivity**:
   ```bash
   ./health-check-module8.sh
   ```

### Network Issues

1. **Detect IPs**:
   ```bash
   ./configure-network-module8.sh
   ```

2. **Verify network**:
   ```bash
   docker network ls | grep module8
   docker network inspect module8-network
   ```

3. **Test connectivity**:
   ```bash
   nc -z 172.21.0.4 5432  # PostgreSQL
   nc -z 172.21.0.3 8086  # InfluxDB
   ```

### Kafka Connection Issues

1. **Verify credentials**:
   ```bash
   grep KAFKA .env.module8
   ```

2. **Test Kafka connection** (from any projector):
   ```bash
   docker exec module8-postgresql-projector python -c "from kafka import KafkaConsumer; print('OK')"
   ```

### Google FHIR Store Issues

1. **Verify credentials file**:
   ```bash
   ls -la /path/to/google-credentials.json
   ```

2. **Check environment variable**:
   ```bash
   grep GOOGLE_CREDENTIALS_PATH .env.module8
   ```

3. **Test FHIR Store access**:
   ```bash
   docker exec module8-fhir-store-projector curl -H "Authorization: Bearer $(gcloud auth print-access-token)" \
     https://healthcare.googleapis.com/v1/projects/$GOOGLE_PROJECT_ID/locations/$GOOGLE_LOCATION/datasets/$GOOGLE_DATASET_ID/fhirStores/$GOOGLE_FHIR_STORE_ID
   ```

## Performance Tuning

### Batch Sizes

Adjust in `.env.module8`:

```bash
# High throughput (more memory)
BATCH_SIZE=500

# Low memory (slower processing)
BATCH_SIZE=50

# Balanced (default)
BATCH_SIZE=100
```

### Resource Limits

Edit `docker-compose.module8-complete.yml`:

```yaml
deploy:
  resources:
    limits:
      cpus: '2.0'
      memory: 2G
    reservations:
      cpus: '0.5'
      memory: 512M
```

### Flush Intervals

```bash
# Fast writes (higher CPU)
FLUSH_INTERVAL=5

# Slow writes (lower CPU)
FLUSH_INTERVAL=30

# Balanced (default)
FLUSH_INTERVAL=10
```

## Monitoring

### Prometheus Metrics

All projectors expose Prometheus metrics at `/metrics`:

```bash
curl http://localhost:8050/metrics
```

### Grafana Dashboards

Import pre-built dashboards for:
- Message processing rates
- Database write latency
- Consumer lag
- Error rates
- Resource usage

### Health Check Automation

Add to cron for periodic health checks:

```bash
# Check health every 5 minutes
*/5 * * * * /path/to/health-check-module8.sh > /tmp/module8-health.log 2>&1
```

## Data Volumes

### Volume Names

```bash
# MongoDB
module8-mongodb-data
module8-mongodb-config

# Elasticsearch
module8-elasticsearch-data

# ClickHouse
module8-clickhouse-data

# Redis
module8-redis-data
```

### Backup Volumes

```bash
# Backup all volumes
docker run --rm -v module8-mongodb-data:/data -v $(pwd):/backup alpine tar czf /backup/mongodb-backup.tar.gz /data

# Restore volume
docker run --rm -v module8-mongodb-data:/data -v $(pwd):/backup alpine tar xzf /backup/mongodb-backup.tar.gz -C /
```

## Security Considerations

1. **Kafka Credentials**: Use SASL_SSL for production
2. **Database Passwords**: Strong passwords in `.env.module8`
3. **Google Credentials**: Secure key file with proper permissions
4. **Network Isolation**: Use Docker networks for service isolation
5. **Volume Encryption**: Enable for production deployments

## Production Deployment

1. **Environment Variables**: Use secrets management (Vault, AWS Secrets Manager)
2. **Monitoring**: Deploy Prometheus and Grafana
3. **Logging**: Centralized log aggregation (ELK stack)
4. **High Availability**: Multiple instances with load balancing
5. **Backup Strategy**: Automated volume backups
6. **Disaster Recovery**: Multi-region deployment

## Files Created

```
backend/stream-services/
├── docker-compose.module8-complete.yml     # Complete orchestration
├── .env.module8.example                    # Environment template
├── start-module8-projectors.sh            # Startup script
├── stop-module8-projectors.sh             # Shutdown script
├── health-check-module8.sh                # Health monitoring
├── logs-module8.sh                        # Log viewer
└── configure-network-module8.sh           # Network configuration
```

## Example Commands

### Complete Startup Sequence

```bash
# 1. Configure network and detect IPs
./configure-network-module8.sh

# 2. Edit environment file
nano .env.module8

# 3. Start all services
./start-module8-projectors.sh

# 4. Verify health
./health-check-module8.sh

# 5. Monitor logs
./logs-module8.sh -f -a
```

### Daily Operations

```bash
# Morning: Check health
./health-check-module8.sh

# Monitor specific service
./logs-module8.sh -f postgresql-projector

# Check errors
./logs-module8.sh -a -e

# Restart failing service
docker-compose -f docker-compose.module8-complete.yml restart postgresql-projector
```

### Maintenance Window

```bash
# 1. Collect statistics
./health-check-module8.sh  # Generate report

# 2. Stop services
./stop-module8-projectors.sh

# 3. Perform maintenance (backups, updates, etc.)

# 4. Restart services
./start-module8-projectors.sh

# 5. Verify health
./health-check-module8.sh
```

## Support

For issues or questions:
1. Check logs: `./logs-module8.sh -e [service-name]`
2. Run health check: `./health-check-module8.sh`
3. Review documentation in `MODULE8_*_COMPLETE.md` files
4. Check individual projector READMEs

## Next Steps

1. Configure `.env.module8` with your credentials
2. Run network configuration: `./configure-network-module8.sh`
3. Start services: `./start-module8-projectors.sh`
4. Verify health: `./health-check-module8.sh`
5. Monitor logs: `./logs-module8.sh -f -a`
6. Deploy monitoring (Prometheus/Grafana)
7. Set up automated backups
8. Configure alerting

## Conclusion

Complete orchestration system for Module 8 storage projectors is ready. All scripts are executable and production-ready with comprehensive error handling, health monitoring, and network configuration.
