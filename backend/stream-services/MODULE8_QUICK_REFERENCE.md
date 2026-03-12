# Module 8 Storage Projectors - Quick Reference

## One-Command Startup

```bash
./start-module8-projectors.sh
```

## Service Ports

| Service | Port | Health Check |
|---------|------|--------------|
| PostgreSQL Projector | 8050 | `curl localhost:8050/health` |
| MongoDB Projector | 8051 | `curl localhost:8051/health` |
| Elasticsearch Projector | 8052 | `curl localhost:8052/health` |
| ClickHouse Projector | 8053 | `curl localhost:8053/health` |
| InfluxDB Projector | 8054 | `curl localhost:8054/health` |
| UPS Projector | 8055 | `curl localhost:8055/health` |
| FHIR Store Projector | 8056 | `curl localhost:8056/health` |
| Neo4j Graph Projector | 8057 | `curl localhost:8057/health` |

## Quick Commands

### Setup
```bash
# Initial setup
./configure-network-module8.sh
cp .env.module8.example .env.module8
nano .env.module8
```

### Operations
```bash
# Start all
./start-module8-projectors.sh

# Stop all
./stop-module8-projectors.sh

# Health check
./health-check-module8.sh

# View logs (follow)
./logs-module8.sh -f -a

# View errors only
./logs-module8.sh -a -e
```

### Individual Services
```bash
# Start one service
docker-compose -f docker-compose.module8-complete.yml up -d postgresql-projector

# Stop one service
docker-compose -f docker-compose.module8-complete.yml stop postgresql-projector

# Restart one service
docker-compose -f docker-compose.module8-complete.yml restart postgresql-projector

# View service logs
./logs-module8.sh -f postgresql-projector
```

## Troubleshooting

```bash
# Check service health
curl localhost:8050/health | python -m json.tool

# View recent errors
./logs-module8.sh -e postgresql-projector

# Check database connectivity
nc -z 172.21.0.4 5432  # PostgreSQL
nc -z 172.21.0.3 8086  # InfluxDB

# Reconfigure network
./configure-network-module8.sh

# Check container status
docker-compose -f docker-compose.module8-complete.yml ps
```

## Required Environment Variables

```bash
# Kafka
KAFKA_BOOTSTRAP_SERVERS=pkc-xxxxx.us-east-1.aws.confluent.cloud:9092
KAFKA_SASL_USERNAME=your-api-key
KAFKA_SASL_PASSWORD=your-api-secret

# PostgreSQL
POSTGRES_HOST=172.21.0.4
POSTGRES_PASSWORD=your-password

# InfluxDB
INFLUXDB_URL=http://172.21.0.3:8086
INFLUXDB_TOKEN=your-token

# Neo4j
NEO4J_URI=bolt://neo4j-ip:7687
NEO4J_PASSWORD=your-password

# Google FHIR Store
GOOGLE_PROJECT_ID=your-project-id
GOOGLE_CREDENTIALS_PATH=/path/to/credentials.json
```

## Health Check Responses

Healthy response:
```json
{
  "status": "healthy",
  "service": "postgresql-projector",
  "version": "1.0.0",
  "uptime": "2h 15m 30s",
  "messages_processed": 12450,
  "last_message": "2024-11-15T21:30:45Z",
  "database_connected": true,
  "kafka_connected": true
}
```

Unhealthy response:
```json
{
  "status": "unhealthy",
  "service": "postgresql-projector",
  "error": "Database connection failed",
  "details": "..."
}
```

## Common Issues

### Service won't start
```bash
# Check logs
./logs-module8.sh -e [service-name]

# Check environment
cat .env.module8

# Reconfigure network
./configure-network-module8.sh
```

### Network issues
```bash
# Detect IPs
./configure-network-module8.sh

# Check network
docker network inspect module8-network

# Reconnect containers
docker network connect module8-network a2f55d83b1fa
docker network connect module8-network 8502fd5d078d
docker network connect module8-network e8b3df4d8a02
```

### Kafka connection failed
```bash
# Verify credentials
grep KAFKA .env.module8

# Test from container
docker exec module8-postgresql-projector env | grep KAFKA
```

## File Locations

```
backend/stream-services/
├── docker-compose.module8-complete.yml  # Main orchestration
├── .env.module8                         # Configuration
├── start-module8-projectors.sh         # Start script
├── stop-module8-projectors.sh          # Stop script
├── health-check-module8.sh             # Health check
├── logs-module8.sh                     # Log viewer
└── configure-network-module8.sh        # Network setup
```

## Monitoring URLs

```bash
# Projectors
http://localhost:8050-8057/health
http://localhost:8050-8057/metrics

# Infrastructure
http://localhost:27017  # MongoDB
http://localhost:9200   # Elasticsearch
http://localhost:8123   # ClickHouse
http://localhost:6379   # Redis
```

## External Containers

| Container | ID | Network IP | Port |
|-----------|----|-----------:|------|
| PostgreSQL | a2f55d83b1fa | 172.21.0.4 | 5432 |
| InfluxDB | 8502fd5d078d | 172.21.0.3 | 8086 |
| Neo4j | e8b3df4d8a02 | Auto-detect | 7687 |

## Emergency Commands

```bash
# Stop everything immediately
docker-compose -f docker-compose.module8-complete.yml stop

# Remove all containers (keeps data)
docker-compose -f docker-compose.module8-complete.yml down

# Nuclear option (deletes ALL data!)
docker-compose -f docker-compose.module8-complete.yml down -v
```

## Support

1. View logs: `./logs-module8.sh -e [service]`
2. Check health: `./health-check-module8.sh`
3. Review docs: `MODULE8_ORCHESTRATION_COMPLETE.md`
4. Check individual service docs: `MODULE8_*_COMPLETE.md`
