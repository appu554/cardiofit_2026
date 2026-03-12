# CardioFit Shared Infrastructure

## Overview

The shared infrastructure layer provides global, platform-wide services that ALL CardioFit microservices depend on. It consists of two major peer components that work together to provide real-time clinical intelligence and high-performance data access.

## Architecture

```
backend/
└── shared-infrastructure/
    ├── flink-processing/      # Global stream processing (real-time intelligence)
    ├── runtime-layer/         # Storage, query, and caching infrastructure
    └── start-shared-infrastructure.sh  # Unified deployment
```

## Two Peer Components

### 1. Flink Processing (Stream Intelligence)
**Purpose**: Real-time event processing and clinical pattern detection

- **Global Scope**: Processes events from ALL microservices
- **Clinical Patterns**: Pathway adherence, lab trends, drug interactions
- **State Management**: Maintains patient context across services
- **Knowledge Broadcasting**: Distributes semantic mesh to all tasks

### 2. Runtime Layer (Storage & Query)
**Purpose**: High-performance data storage, querying, and caching

- **Neo4j Dual-Stream**: Patient data + semantic mesh graphs
- **Query Router**: Intelligent source selection
- **Cache Prefetcher**: Event-driven cache warming
- **Evidence Envelopes**: Clinical decision auditing

## Why This Architecture?

### Separation of Concerns
```
Events → Flink (Processing) → Runtime Layer (Storage)
                ↓
         Clinical Insights
```

- **Flink**: Handles stream processing, pattern detection, real-time analytics
- **Runtime Layer**: Handles storage, queries, caching, persistence

### Global vs Service-Specific
- **Shared Infrastructure**: Used by ALL services (Flink, Neo4j, Kafka, Redis)
- **Service-Specific**: Each microservice has its own business logic

## Quick Start

### Start Everything
```bash
cd backend/shared-infrastructure
./start-shared-infrastructure.sh
```

### Start Individual Components
```bash
# Start only Runtime Layer
cd runtime-layer
./start-runtime.sh

# Start only Flink Processing
cd runtime-layer
docker-compose up -d flink-jobmanager flink-taskmanager
```

## Service Endpoints

### Flink Processing
- **Flink JobManager UI**: http://localhost:8081
- **Flink Metrics**: http://localhost:9249/metrics

### Runtime Layer
- **Neo4j Browser**: http://localhost:7474
- **GraphDB Workbench**: http://localhost:7200
- **Query Router API**: http://localhost:8070
- **Cache Prefetcher**: http://localhost:8055
- **Evidence Envelope**: http://localhost:8060

### Monitoring
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000

## Data Flow

### 1. Event Processing Flow
```
Microservices
    ↓ (publish events)
Kafka Topics
    ↓ (consume)
Flink Processing
    ├── Pattern Detection
    ├── Event Enrichment
    └── CDC Processing
    ↓ (output)
Multiple Sinks
    ├── Runtime Layer (Neo4j, ClickHouse)
    ├── Notification Services
    └── FHIR Store
```

### 2. Query Flow
```
Microservices
    ↓ (query request)
Query Router (Runtime Layer)
    ↓ (route to optimal source)
    ├── Neo4j (graph queries)
    ├── ClickHouse (analytics)
    ├── Redis (cache)
    └── GraphDB (reasoning)
    ↓ (response)
Evidence Envelope (audit)
    ↓
Microservice (result)
```

## Integration Guide for Microservices

### Publishing Events (to Flink)
```python
# Python microservice example
from kafka import KafkaProducer

producer = KafkaProducer(
    bootstrap_servers=['localhost:29092'],
    value_serializer=lambda v: json.dumps(v).encode('utf-8')
)

# Publish patient event for Flink processing
event = {
    'patient_id': 'P123',
    'event_type': 'MEDICATION_PRESCRIBED',
    'medication': 'metformin',
    'timestamp': time.time()
}
producer.send('patient.events.all', event)
```

### Querying Data (from Runtime Layer)
```python
# Query through the Query Router
import httpx

async def query_patient_data(patient_id):
    async with httpx.AsyncClient() as client:
        response = await client.post(
            'http://localhost:8070/query',
            json={
                'query_type': 'patient_graph',
                'patient_id': patient_id
            }
        )
        return response.json()
```

## Performance Guarantees

| Component | SLA | Monitoring |
|-----------|-----|------------|
| **Flink Processing** | < 500ms latency | Flink metrics |
| **Query Router** | < 100ms p95 | Prometheus |
| **Cache Hit Rate** | > 80% | Redis metrics |
| **Knowledge Sync** | < 5 min | CDC lag monitor |

## Deployment Options

### Development
```bash
# Minimal setup for development
docker-compose up -d kafka neo4j redis flink-jobmanager
```

### Production
```bash
# Full setup with monitoring
./start-shared-infrastructure.sh --production
```

### Kubernetes
```bash
# Deploy to K8s cluster
kubectl apply -f flink-processing/deployment/kubernetes/
kubectl apply -f runtime-layer/deployment/kubernetes/
```

## Resource Requirements

### Minimum (Development)
- CPU: 8 cores
- RAM: 16 GB
- Storage: 100 GB

### Recommended (Production)
- CPU: 32 cores
- RAM: 64 GB
- Storage: 1 TB SSD
- Network: 10 Gbps

## Monitoring & Health

### Health Check
```bash
# Check all components
./runtime-layer/health-check.sh
```

### Metrics
- Prometheus: http://localhost:9090
- Grafana Dashboards:
  - Flink Processing Overview
  - Runtime Layer Performance
  - Clinical Pattern Detection
  - SLA Compliance

## Team Ownership

| Component | Team | Responsibilities |
|-----------|------|------------------|
| **Flink Processing** | Stream Processing Team | Flink infrastructure, job management |
| **Clinical Patterns** | Clinical Intelligence Team | Pattern implementation, rules |
| **Runtime Layer** | Platform Team | Storage, caching, query routing |
| **Monitoring** | DevOps Team | Observability, SLA enforcement |

## Documentation

- **Flink Processing**: [flink-processing/README.md](flink-processing/README.md)
- **Runtime Layer**: [runtime-layer/README.md](runtime-layer/README.md)
- **Deployment Guide**: [runtime-layer/RUNTIME_LAYER_DEPLOYMENT.md](runtime-layer/RUNTIME_LAYER_DEPLOYMENT.md)
- **Flink Architecture**: [runtime-layer/FLINK_GLOBAL_ARCHITECTURE.md](runtime-layer/FLINK_GLOBAL_ARCHITECTURE.md)

## Support

For issues or questions:
1. Check component logs: `docker-compose logs [service]`
2. Run health checks: `./runtime-layer/health-check.sh`
3. Review documentation above
4. Contact platform team

---
**Version**: 2.0.0 (Flink elevated to peer component)
**Last Updated**: 2024
**Maintained By**: CardioFit Platform Team