# CardioFit Runtime Layer - Complete Deployment Guide

## 🚀 Quick Start

```bash
# 1. Clone and navigate to runtime layer
cd backend/shared-infrastructure/runtime-layer

# 2. Copy and configure environment
cp .env.example .env
# Edit .env with your credentials

# 3. Start entire runtime layer
./start-runtime.sh

# 4. Verify all services are healthy
./health-check.sh
```

## 📋 Overview

The Runtime Layer is a shared, high-performance infrastructure that provides:
- **Real-time event processing** with Apache Flink
- **Dual-stream graph databases** with Neo4j
- **Semantic reasoning** with GraphDB
- **Multi-layer caching** with Redis
- **Analytics engine** with ClickHouse
- **Clinical audit trail** with Evidence Envelopes
- **SLA monitoring** and performance guarantees

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Microservices Layer                      │
│  (Patient, Medication, Observation, Auth, FHIR Services)    │
└────────────┬────────────────────────────────────┬───────────┘
             │                                    │
             ▼                                    ▼
┌─────────────────────────────────────────────────────────────┐
│                      Runtime Layer                           │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │ Query Router │  │Cache Prefetch│  │   Evidence   │     │
│  │   (8070)     │  │    (8055)    │  │  Envelope    │     │
│  └──────┬───────┘  └──────┬───────┘  │   (8060)     │     │
│         │                  │          └──────┬───────┘     │
│         ▼                  ▼                 ▼              │
│  ┌────────────────────────────────────────────────────┐    │
│  │           Core Infrastructure Services              │    │
│  │                                                     │    │
│  │  Neo4j │ GraphDB │ Kafka │ Redis │ ClickHouse     │    │
│  │  (7687)│  (7200) │ (9092)│ (6379)│   (8123)       │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                              │
│  ┌────────────────────────────────────────────────────┐    │
│  │              Stream Processing (Flink)              │    │
│  │                    (8081)                           │    │
│  └─────────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────┘
```

## 🔌 Service Endpoints for Microservices

### Primary Integration Points

| Service | Internal Endpoint | External Endpoint | Purpose |
|---------|------------------|-------------------|---------|
| **Query Router** | `query-router:8070` | `localhost:8070` | Intelligent data source routing |
| **Neo4j (Bolt)** | `neo4j:7687` | `localhost:7687` | Graph database queries |
| **Redis** | `redis:6379` | `localhost:6379` | Caching layer |
| **Kafka** | `kafka:9092` | `localhost:29092` | Event streaming |
| **ClickHouse** | `clickhouse:8123` | `localhost:8123` | Analytics queries |

### Microservice Connection Examples

#### Python FastAPI Service
```python
# Example: Medication Service connecting to Runtime Layer

import os
from neo4j import GraphDatabase
import redis
from clickhouse_driver import Client
from aiokafka import AIOKafkaProducer

# Configuration from environment
NEO4J_URI = os.getenv("NEO4J_URI", "bolt://localhost:7687")
NEO4J_USER = os.getenv("NEO4J_USER", "neo4j")
NEO4J_PASSWORD = os.getenv("NEO4J_PASSWORD", "runtime_password")
REDIS_URL = os.getenv("REDIS_URL", "redis://localhost:6379/0")
CLICKHOUSE_HOST = os.getenv("CLICKHOUSE_HOST", "localhost")
KAFKA_BOOTSTRAP_SERVERS = os.getenv("KAFKA_BOOTSTRAP_SERVERS", "localhost:29092")
QUERY_ROUTER_URL = os.getenv("QUERY_ROUTER_URL", "http://localhost:8070")

# Neo4j connection
neo4j_driver = GraphDatabase.driver(
    NEO4J_URI,
    auth=(NEO4J_USER, NEO4J_PASSWORD)
)

# Redis connection
redis_client = redis.from_url(REDIS_URL)

# ClickHouse connection
clickhouse_client = Client(host=CLICKHOUSE_HOST)

# Kafka producer
async def get_kafka_producer():
    return AIOKafkaProducer(
        bootstrap_servers=KAFKA_BOOTSTRAP_SERVERS
    )

# Query Router usage
import httpx

async def query_through_router(query_type: str, params: dict):
    async with httpx.AsyncClient() as client:
        response = await client.post(
            f"{QUERY_ROUTER_URL}/query",
            json={
                "query_type": query_type,
                "params": params
            }
        )
        return response.json()
```

#### Node.js Apollo Federation
```javascript
// Example: Apollo Federation connecting to Runtime Layer

const neo4j = require('neo4j-driver');
const redis = require('redis');
const { Kafka } = require('kafkajs');
const { ClickHouse } = require('clickhouse');

// Configuration
const config = {
  neo4j: {
    uri: process.env.NEO4J_URI || 'bolt://localhost:7687',
    user: process.env.NEO4J_USER || 'neo4j',
    password: process.env.NEO4J_PASSWORD || 'runtime_password'
  },
  redis: {
    url: process.env.REDIS_URL || 'redis://localhost:6379'
  },
  kafka: {
    brokers: [process.env.KAFKA_BROKER || 'localhost:29092']
  },
  clickhouse: {
    url: process.env.CLICKHOUSE_URL || 'http://localhost:8123'
  },
  queryRouter: {
    url: process.env.QUERY_ROUTER_URL || 'http://localhost:8070'
  }
};

// Neo4j driver
const neo4jDriver = neo4j.driver(
  config.neo4j.uri,
  neo4j.auth.basic(config.neo4j.user, config.neo4j.password)
);

// Redis client
const redisClient = redis.createClient({
  url: config.redis.url
});

// Kafka client
const kafka = new Kafka({
  clientId: 'apollo-federation',
  brokers: config.kafka.brokers
});

// Query Router client
const queryRouter = {
  async query(queryType, params) {
    const response = await fetch(`${config.queryRouter.url}/query`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ query_type: queryType, params })
    });
    return response.json();
  }
};
```

## 📊 Data Flows

### 1. Patient Event Flow (Global Processing)
```
ALL Microservices → Kafka (patient.events) → GLOBAL Flink → Neo4j (patient_data) → Multi-Sink
                                                  ↑
                                           Semantic Mesh
                                           (Broadcast State)
```

### 2. Knowledge Update Flow (Global CDC)
```
ALL KB Services → PostgreSQL → Debezium CDC → Kafka (kb.changes) → GLOBAL Flink → Adapter → Neo4j
(KB-3/4/5/6/7)                                                           ↓
                                                                  Knowledge Broadcast
```

### 3. Clinical Query Flow
```
Microservice → Query Router → {Neo4j | ClickHouse | Redis | GraphDB} → Evidence Envelope → Response
```

## 🗂️ Kafka Topics

| Topic | Purpose | Producers | Consumers |
|-------|---------|-----------|-----------|
| `patient.events` | Patient data changes | Microservices | Flink, Cache Prefetcher |
| `medication.events` | Medication updates | Medication Service | Flink, SLA Monitor |
| `kb.changes.terminology` | KB-7 terminology updates | Debezium | Adapter Service |
| `kb.changes.protocols` | KB-3 protocol updates | Debezium | Adapter Service |
| `kb.changes.safety_rules` | KB-4 safety updates | Debezium | Adapter Service |
| `workflow.events` | Workflow execution | Workflow Engine | Cache Prefetcher |
| `audit.events` | Audit trail | All Services | Evidence Envelope |

## 🔐 Security Configuration

### Default Credentials (Change in Production!)
```env
# Neo4j
NEO4J_USER=neo4j
NEO4J_PASSWORD=runtime_password

# MongoDB
MONGODB_ROOT_USER=admin
MONGODB_ROOT_PASSWORD=admin_password

# Grafana
GRAFANA_PASSWORD=admin

# ClickHouse
CLICKHOUSE_USER=default
CLICKHOUSE_PASSWORD=clickhouse_password
```

### Network Security
- All services run in Docker network `runtime-network`
- External ports are exposed only as needed
- Use environment variables for all credentials
- Enable TLS in production

## 📈 Performance Guarantees

| Metric | SLA | Monitoring |
|--------|-----|------------|
| Patient Event Latency | < 500ms | Flink metrics |
| Knowledge Sync | < 5 minutes | CDC lag monitor |
| Cache Hit Rate | > 80% | Redis metrics |
| Query Response p95 | < 100ms | Query Router metrics |
| Snapshot Creation | < 20ms | Snapshot Manager |

## 🔧 Deployment Options

### Option 1: Full Stack (Recommended)
```bash
docker-compose up -d
```

### Option 2: Core Only (Minimum)
```bash
docker-compose up -d \
  neo4j graphdb kafka zookeeper redis clickhouse
```

### Option 3: With External Services
```bash
# If using external Neo4j/Kafka/Redis
docker-compose up -d \
  query-router cache-prefetcher evidence-envelope \
  flink-jobmanager flink-taskmanager
```

## 🏥 Health Checks

### Automated Health Check
```bash
./health-check.sh
```

### Manual Service Checks
```bash
# Neo4j
curl http://localhost:7474

# GraphDB
curl http://localhost:7200/rest/info

# Kafka
docker exec runtime-kafka kafka-topics --bootstrap-server localhost:9092 --list

# Redis
redis-cli ping

# ClickHouse
curl http://localhost:8123/ping

# Flink
curl http://localhost:8081/config

# Query Router
curl http://localhost:8070/health

# Cache Prefetcher
curl http://localhost:8055/health

# Evidence Envelope
curl http://localhost:8060/health

# SLA Monitoring
curl http://localhost:8050/health
```

## 📊 Monitoring & Observability

### Grafana Dashboards
Access at http://localhost:3000 (admin/admin)

Available dashboards:
- Runtime Layer Overview
- Service Health Status
- Query Performance Metrics
- Cache Hit Rates
- Kafka Lag Monitoring
- Flink Job Status
- SLA Compliance

### Prometheus Metrics
Access at http://localhost:9090

Key metrics:
- `runtime_query_duration_seconds`
- `runtime_cache_hit_ratio`
- `runtime_kafka_lag_messages`
- `runtime_flink_checkpoint_duration`
- `runtime_evidence_envelope_count`

## 🛠️ Troubleshooting

### Service Won't Start
```bash
# Check logs
docker-compose logs [service-name]

# Verify port availability
lsof -i :[port]

# Check resource usage
docker stats
```

### Connection Issues
```bash
# Test network connectivity
docker exec [microservice-container] ping neo4j
docker exec [microservice-container] nc -zv kafka 9092

# Verify DNS resolution
docker exec [microservice-container] nslookup query-router
```

### Performance Issues
```bash
# Check resource usage
docker stats

# Monitor Kafka lag
docker exec runtime-kafka kafka-consumer-groups \
  --bootstrap-server localhost:9092 \
  --all-groups --describe

# Check Flink jobs
curl http://localhost:8081/jobs
```

## 📦 Resource Requirements

### Minimum (Development)
- CPU: 8 cores
- RAM: 16 GB
- Storage: 100 GB SSD

### Recommended (Production)
- CPU: 16+ cores
- RAM: 32 GB
- Storage: 500 GB SSD
- Network: 1 Gbps

## 🔄 Maintenance

### Backup Strategy
```bash
# Neo4j backup
docker exec neo4j neo4j-admin backup \
  --database=patient_data \
  --backup-dir=/backups

# ClickHouse backup
docker exec clickhouse clickhouse-backup create

# Redis backup
docker exec redis redis-cli BGSAVE
```

### Updates
```bash
# Pull latest images
docker-compose pull

# Restart with updates
docker-compose up -d --force-recreate
```

## 📚 Additional Resources

- [Neo4j Documentation](https://neo4j.com/docs/)
- [Apache Flink Documentation](https://flink.apache.org/docs/)
- [Kafka Documentation](https://kafka.apache.org/documentation/)
- [ClickHouse Documentation](https://clickhouse.com/docs/)
- [GraphDB Documentation](https://graphdb.ontotext.com/documentation/)

## 🤝 Support

For issues or questions:
1. Check logs: `docker-compose logs [service]`
2. Review health checks: `./health-check.sh`
3. Consult service-specific documentation
4. Contact platform team

---
**Version**: 1.0.0
**Last Updated**: 2024-01
**Maintained By**: CardioFit Platform Team