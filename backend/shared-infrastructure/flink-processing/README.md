# Flink Stream Processing Infrastructure

## Overview

Apache Flink serves as the **GLOBAL** stream processing infrastructure for the entire CardioFit platform. It operates as a peer to the Runtime Layer, processing events directly from ALL microservices and knowledge bases.

## Architectural Position

```
backend/
└── shared-infrastructure/
    ├── flink-processing/    # THIS COMPONENT (Global Stream Processing)
    └── runtime-layer/       # Peer component (Query & Storage Infrastructure)
```

## Why Flink is at Shared-Infrastructure Level

1. **Direct Service Integration**: Flink receives events directly from all microservices via Kafka, not through the runtime layer
2. **Global State Management**: Maintains patient context across ALL services
3. **Knowledge Broadcasting**: Distributes semantic mesh to all processing tasks
4. **Independent Scaling**: Can be scaled independently of runtime layer components
5. **Operational Independence**: Managed as a separate infrastructure component

## Core Responsibilities

### 1. Clinical Pattern Detection
- **Pattern 1**: Clinical Pathway Adherence (Protocol deviation detection)
- **Pattern 2**: Aggregate Lab Trend Analysis (Population health insights)
- **Pattern 3**: Real-time Drug Interaction Monitoring
- **Pattern 4**: Sepsis Early Warning System

### 2. Event Enrichment Pipeline
- Patient context assembly from multiple services
- Semantic mesh integration (KB-3/4/5/6/7 knowledge)
- Real-time enrichment with clinical intelligence

### 3. CDC Stream Processing
- Knowledge base change propagation
- Semantic mesh updates
- Protocol synchronization

### 4. Multi-Sink Distribution
- Critical alerts → Notification services
- Enriched events → FHIR Store
- Analytics data → ClickHouse
- Audit trails → Evidence Envelope service

## Directory Structure

```
flink-processing/
├── clinical-patterns/       # Clinical intelligence patterns
│   ├── pathway-adherence/   # Pattern 1: Protocol compliance
│   ├── lab-trends/          # Pattern 2: Trend analysis
│   ├── drug-interactions/   # Pattern 3: DDI monitoring
│   └── early-warning/       # Pattern 4: Sepsis detection
├── cdc-pipeline/           # Change Data Capture processing
│   ├── debezium-consumer/   # Consumes DB changes
│   ├── knowledge-sync/      # Syncs to semantic mesh
│   └── version-manager/     # Manages knowledge versions
├── stream-enrichment/      # Event enrichment pipeline
│   ├── context-assembly/    # Patient context builder
│   ├── semantic-broadcast/  # Knowledge distribution
│   └── multi-sink-router/   # Output routing
├── deployment/             # Deployment configurations
│   ├── docker/              # Docker images
│   ├── kubernetes/          # K8s manifests
│   └── flink-conf/          # Flink configurations
├── docs/                   # Documentation
│   ├── patterns/            # Pattern specifications
│   ├── api/                 # Stream APIs
│   └── operations/          # Operational guides
├── src/                    # Source code (Java/Scala)
│   └── main/
│       └── java/
│           └── com/
│               └── cardiofit/
│                   └── flink/
└── pom.xml                 # Maven configuration
```

## Integration Points

### Input Sources (Kafka Topics)

```yaml
# From ALL microservices
patient.events.all:
  - patient-service
  - medication-service
  - observation-service
  - encounter-service

# From knowledge bases (via CDC)
kb.changes.all:
  - kb-3-protocols
  - kb-4-safety
  - kb-5-interactions
  - kb-6-evidence
  - kb-7-terminology
```

### Output Destinations

```yaml
# To Runtime Layer components
- Neo4j (patient_data stream)
- ClickHouse (analytics)
- Redis (cache warming)

# Direct to services
- Notification service (alerts)
- FHIR Store (clinical records)
- Evidence Envelope (audit)

# To dashboards
- Grafana (metrics)
- Clinical dashboard (insights)
```

## Deployment

### Standalone Deployment
```bash
cd backend/shared-infrastructure/flink-processing
docker-compose -f deployment/docker/docker-compose.yml up -d
```

### Integration with Platform
```bash
# Start Flink alongside runtime-layer
cd backend/shared-infrastructure
./start-shared-infrastructure.sh
```

## Key Differentiators from Runtime Layer

| Aspect | Flink Processing | Runtime Layer |
|--------|-----------------|---------------|
| **Purpose** | Stream processing & pattern detection | Query routing & data storage |
| **Timing** | Real-time event processing | Request/response queries |
| **State** | Distributed patient context | Graph & analytical databases |
| **Input** | Event streams from Kafka | API calls from services |
| **Output** | Alerts, enriched events | Query results, cached data |
| **Scaling** | Horizontal (TaskManagers) | Service-specific scaling |

## Clinical Patterns Implementation

### Pattern 1: Clinical Pathway Adherence
- **Location**: `clinical-patterns/pathway-adherence/`
- **Function**: Detects deviations from KB-3 protocols
- **Input**: Medication, diagnosis, and procedure events
- **Output**: Pathway deviation alerts with recommendations

### Pattern 2: Aggregate Lab Trend Analysis
- **Location**: `clinical-patterns/lab-trends/`
- **Function**: Identifies population health trends
- **Input**: Lab results over sliding windows
- **Output**: Cohort insights and risk predictions

## Performance Requirements

- **Event Processing Latency**: < 500ms end-to-end
- **Throughput**: 100,000 events/second
- **State Size**: Up to 10GB per TaskManager
- **Checkpoint Interval**: 30 seconds
- **Exactly-Once Semantics**: Enabled for critical paths

## Monitoring

### Metrics Exposed
- `flink_event_processing_latency`
- `flink_pattern_detection_rate`
- `flink_state_size_bytes`
- `flink_checkpoint_duration`
- `flink_kafka_lag`

### Dashboards
- Flink Web UI: http://localhost:8081
- Prometheus metrics: http://localhost:9249/metrics
- Grafana dashboards: http://localhost:3000

## Team Ownership

- **Stream Processing Team**: Owns Flink infrastructure
- **Clinical Intelligence Team**: Owns pattern implementations
- **Platform Team**: Owns integration with runtime-layer

## Related Components

- **Runtime Layer**: Peer infrastructure component for storage/query
- **Kafka**: Event streaming backbone
- **Knowledge Bases**: Source of clinical intelligence
- **Microservices**: Event producers

---
*This component represents the real-time intelligence layer of CardioFit, processing millions of clinical events to detect patterns, ensure safety, and improve patient outcomes.*