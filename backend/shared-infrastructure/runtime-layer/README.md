# Clinical Synthesis Hub Runtime Layer

**Purpose**: High-performance, event-driven data platform bridging authoritative knowledge bases with real-time clinical services

## Architecture Overview

```
Runtime Layer Components
├── Core Infrastructure (docker-compose.core.yml)
│   ├── neo4j (dual-stream: patient_data + semantic_mesh)
│   ├── graphdb (OWL reasoning)
│   ├── kafka + zookeeper (event streaming)
│   ├── redis (L1/L2 caching)
│   └── clickhouse (analytics + pre-computed scores)
├── Processing Services (docker-compose.services.yml)
│   ├── flink-jobmanager + flink-taskmanager (stream processing)
│   ├── l1-cache-prefetcher (intelligent cache warming)
│   ├── evidence-envelope (clinical decision auditing)
│   └── query-router (intelligent source routing)
└── Monitoring & SLA (docker-compose.monitoring.yml)
    ├── sla-monitoring (timing guarantees)
    ├── prometheus (metrics collection)
    └── grafana (visualization)
```

## Core Data Flows

### 1. Real-Time Patient Flow
```
Patient Event → Kafka → Flink → Neo4j patient_data → Multi-sink distribution
```

### 2. Knowledge Synchronization Flow
```
KB Update → PostgreSQL WAL → Debezium → Kafka → Adapter → Neo4j semantic_mesh
```

### 3. Clinical Service Request Flow
```
Service Request → Query Router → Snapshot Creation → Cache Check → Source Selection → Response with Evidence Envelope
```

## Quick Start

### 1. Start Core Infrastructure
```bash
docker-compose -f docker-compose.core.yml up -d
```

### 2. Start Processing Services
```bash
docker-compose -f docker-compose.services.yml up -d
```

### 3. Start Monitoring (Optional)
```bash
docker-compose -f docker-compose.monitoring.yml up -d
```

### 4. Verify Services
```bash
# Check Neo4j
curl http://localhost:7474

# Check Flink JobManager
curl http://localhost:8081

# Check SLA Monitoring
curl http://localhost:8050/health

# Check Grafana
open http://localhost:3000
```

## Key Components

### Neo4j Dual-Stream Architecture
- **patient_data**: Real-time patient events (medications, conditions, encounters)
- **semantic_mesh**: Reasoned knowledge from GraphDB (drug hierarchies, interactions)

### Query Router Intelligence
- Pattern recognition for optimal source selection
- Cost-based query planning with circuit breakers
- Routes between: PostgreSQL, Elasticsearch, Neo4j, ClickHouse, GraphDB

### Evidence Envelopes
- Audit trail for clinical decisions
- Provenance for regulatory compliance
- Snapshot consistency across workflow executions

### Timing Guarantees
- Patient Events: < 500ms end-to-end
- Knowledge Updates: < 5 minutes propagation
- Cache Warming: 50-100ms
- Query Response: p95 < 100ms

## Directory Structure
```
runtime-layer/
├── adapters/              # GraphDB-to-Neo4j transformation
├── cache-warming/         # L1/L2/L3 cache strategies
├── cdc-pipeline/          # Change data capture
├── clickhouse/            # Analytics + runtime query management
├── evidence-envelope-service/  # Clinical decision auditing
├── flink-jobs/            # Stream processing jobs
├── l1-cache-prefetcher-service/  # Event-driven cache warming
├── neo4j-dual-stream/     # Patient + semantic graph management
├── query-router/          # Intelligent source routing
├── sla-monitoring-service/  # Timing guarantee enforcement
└── tests/                 # Integration and unit tests
```