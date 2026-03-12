# Flink as Global Stream Processing Infrastructure

## Executive Summary

Apache Flink operates as a **GLOBAL, SHARED** stream processing infrastructure within the CardioFit Runtime Layer, serving ALL microservices and knowledge bases. It is NOT service-specific but rather a critical platform component that enables real-time clinical intelligence across the entire system.

## Why Flink Must Be Global

### 1. Cross-Service Event Correlation
Flink must correlate events from multiple microservices to detect clinical patterns:

```
Example: Clinical Pathway Adherence Detection

Patient Service   → Event: "Patient admitted with diabetes"
     ↓
Medication Service → Event: "Metformin prescribed"
     ↓
Observation Service → Event: "Glucose reading: 250 mg/dL"
     ↓
GLOBAL FLINK CORRELATION → Alert: "Hyperglycemic despite treatment - protocol deviation"
```

### 2. Unified Patient Context
Each patient's complete context spans multiple services and must be maintained centrally:

```java
// Flink's Global Patient State (RocksDB)
KeyedState<PatientId> = {
    // From Patient Service
    demographics: {...},
    conditions: ["T2DM", "HTN"],

    // From Medication Service
    activeMedications: ["metformin", "lisinopril"],

    // From Observation Service
    latestVitals: {glucose: 250, bp: "140/90"},

    // From KB-3 Protocol Service
    protocolState: "STEP_3_INTENSIFICATION",

    // From KB-5 Drug Interactions
    interactionRisks: ["moderate"]
}
```

### 3. Broadcast State Pattern for Knowledge Distribution
The semantic mesh from GraphDB must be available to ALL processing tasks:

```
GraphDB Semantic Mesh (Knowledge from KB-3/4/5/6/7)
         ↓
    Flink Broadcast
         ↓
    ┌────┴────┬────────┬─────────┐
    ↓         ↓        ↓         ↓
Task-1    Task-2   Task-3    Task-N
(All tasks have local copy of knowledge)
```

## Global Flink Architecture

### Stream Topology Design

```
┌──────────────────────────────────────────────────────────┐
│                 GLOBAL FLINK CLUSTER                      │
│                                                           │
│  JobManager (Coordinator)                                │
│  ├── Job: Patient Event Processing Pipeline              │
│  ├── Job: CDC Knowledge Synchronization Pipeline         │
│  ├── Job: Clinical Pattern Detection Pipeline            │
│  └── Job: Multi-Sink Distribution Pipeline               │
│                                                           │
│  TaskManager-1 (Worker)                                  │
│  ├── Partition: Patients A-F                            │
│  ├── State: RocksDB (patient context)                   │
│  └── Broadcast State: Semantic Mesh v2.1                │
│                                                           │
│  TaskManager-2 (Worker)                                  │
│  ├── Partition: Patients G-M                            │
│  ├── State: RocksDB (patient context)                   │
│  └── Broadcast State: Semantic Mesh v2.1                │
│                                                           │
│  TaskManager-3 (Worker)                                  │
│  ├── Partition: Patients N-S                            │
│  ├── State: RocksDB (patient context)                   │
│  └── Broadcast State: Semantic Mesh v2.1                │
│                                                           │
│  TaskManager-4 (Worker)                                  │
│  ├── Partition: Patients T-Z                            │
│  ├── State: RocksDB (patient context)                   │
│  └── Broadcast State: Semantic Mesh v2.1                │
└──────────────────────────────────────────────────────────┘
```

### Global Processing Patterns

#### Pattern 1: Clinical Pathway Adherence (Cross-Service)
```java
public class GlobalPathwayAdherenceFunction extends KeyedProcessFunction<String, Event, Alert> {

    // Global state spanning all services
    private ValueState<PatientContext> patientState;
    private MapState<String, ProtocolState> protocolStates;

    @Override
    public void processBroadcastElement(SemanticMesh mesh, Context ctx) {
        // Receive global knowledge updates from ALL KBs
        ctx.getBroadcastState(meshDescriptor).put("current", mesh);
    }

    @Override
    public void processElement(Event event, Context ctx) {
        // Process events from ANY microservice
        PatientContext patient = patientState.value();
        SemanticMesh mesh = ctx.getBroadcastState(meshDescriptor).get("current");

        // Detect cross-service patterns
        if (event.source == "MedicationService" &&
            patient.hasCondition("T2DM") &&
            !mesh.isRecommendedMedication(event.medication, patient.protocol)) {

            ctx.output(deviationTag, new Alert(
                "Protocol deviation detected across services",
                Alert.Priority.HIGH
            ));
        }
    }
}
```

#### Pattern 2: Population Health Trends (Multi-Service Aggregation)
```java
public class GlobalTrendAnalysis extends ProcessWindowFunction<Event, Insight, String, TimeWindow> {

    @Override
    public void process(String cohortKey, Context ctx, Iterable<Event> events, Collector<Insight> out) {
        // Aggregate events from multiple services
        List<Event> labEvents = filterBySource(events, "ObservationService");
        List<Event> medEvents = filterBySource(events, "MedicationService");
        List<Event> encounterEvents = filterBySource(events, "PatientService");

        // Cross-service correlation
        double avgGlucose = calculateAverage(labEvents, "glucose");
        boolean onProtocol = checkMedicationAdherence(medEvents);
        int admissions = countAdmissions(encounterEvents);

        // Generate population-level insight
        if (avgGlucose > 180 && onProtocol && admissions > 2) {
            out.collect(new Insight(
                "Treatment-resistant diabetes cohort identified",
                cohortKey,
                events.size()
            ));
        }
    }
}
```

## Kafka Topics for Global Processing

### Input Topics (From ALL Services)
```yaml
# Patient events from all microservices
patient.events.all:
  producers:
    - patient-service
    - medication-service
    - observation-service
    - encounter-service
    - procedure-service

# Knowledge updates from all KBs
kb.changes.all:
  producers:
    - kb-3-protocols (via Debezium)
    - kb-4-safety (via Debezium)
    - kb-5-interactions (via Debezium)
    - kb-6-evidence (via Debezium)
    - kb-7-terminology (via Debezium)

# Workflow events
workflow.events:
  producers:
    - workflow-engine
    - care-plan-service
```

### Output Topics (To Multiple Consumers)
```yaml
# Clinical alerts for immediate action
clinical.alerts.critical:
  consumers:
    - notification-service
    - ehr-integration
    - clinician-dashboard

# Population insights
population.insights:
  consumers:
    - analytics-dashboard
    - quality-reporting
    - research-platform

# Enriched events
events.enriched:
  consumers:
    - fhir-store
    - clickhouse-analytics
    - audit-service
```

## Configuration for Global Deployment

### Flink Cluster Sizing for Global Operations
```yaml
# flink-conf.yaml for global processing

# JobManager (coordinates all jobs)
jobmanager.memory.process.size: 4096m
jobmanager.rpc.port: 6123

# TaskManagers (scale based on total event volume)
taskmanager.memory.process.size: 8192m
taskmanager.numberOfTaskSlots: 8  # Parallel tasks per TM
parallelism.default: 32  # Total parallelism across cluster

# State Backend for global patient context
state.backend: rocksdb
state.checkpoints.dir: hdfs://namenode/flink-checkpoints
state.backend.rocksdb.memory.managed: true

# Checkpointing for exactly-once semantics
execution.checkpointing.interval: 30s
execution.checkpointing.mode: EXACTLY_ONCE
execution.checkpointing.max-concurrent-checkpoints: 1

# Global resource management
taskmanager.network.memory.fraction: 0.2
taskmanager.memory.managed.fraction: 0.5

# Metrics for global monitoring
metrics.reporter.prom.class: org.apache.flink.metrics.prometheus.PrometheusReporter
metrics.reporter.prom.port: 9249
```

### Docker Scaling for Global Load
```yaml
# docker-compose.yml scaling configuration

flink-taskmanager:
  image: flink:1.18.0
  deploy:
    replicas: 4  # Scale based on patient volume
    resources:
      limits:
        cpus: '4'
        memory: 8G
      reservations:
        cpus: '2'
        memory: 4G
```

## Integration Guidelines for Microservices

### For Service Developers
```java
// WRONG: Service-specific Flink processing
@Service
public class MedicationService {
    private FlinkEnvironment flink;  // ❌ Don't embed Flink
}

// RIGHT: Publish to global Flink via Kafka
@Service
public class MedicationService {
    @Autowired
    private KafkaProducer<String, Event> eventProducer;

    public void prescribeMedication(Prescription rx) {
        // Business logic
        savePrescription(rx);

        // Publish to global Flink processing
        eventProducer.send(new ProducerRecord<>(
            "patient.events.all",  // Global topic
            rx.getPatientId(),     // Key for partitioning
            new MedicationEvent(rx) // Event for Flink
        ));
    }
}
```

### For DevOps Teams
```bash
# Monitor global Flink cluster
curl http://flink-jobmanager:8081/jobs

# Scale TaskManagers based on load
docker-compose scale flink-taskmanager=6

# View global processing metrics
curl http://localhost:9249/metrics | grep flink_
```

## Benefits of Global Flink Architecture

### 1. **Unified Clinical Intelligence**
- Single source of truth for patient context
- Consistent pattern detection across all services
- Centralized clinical rule evaluation

### 2. **Resource Efficiency**
- Shared cluster reduces infrastructure costs
- Better resource utilization through pooling
- Centralized state management

### 3. **Operational Simplicity**
- Single cluster to monitor and maintain
- Unified configuration and deployment
- Centralized logging and metrics

### 4. **Scalability**
- Horizontal scaling serves all services
- Shared state backend optimizations
- Efficient broadcast state distribution

### 5. **Data Consistency**
- Exactly-once processing guarantees
- Consistent checkpointing across all streams
- Unified watermarking for event-time processing

## Summary

Flink MUST operate as global infrastructure because:

1. **Clinical patterns span multiple services** - Detection requires cross-service correlation
2. **Patient context is distributed** - Complete view needs data from all services
3. **Knowledge is universal** - Semantic mesh must be available everywhere
4. **Resource efficiency** - Shared processing is more efficient than service-specific clusters
5. **Operational simplicity** - One cluster is easier to manage than many

The current deployment correctly positions Flink as global infrastructure, and all microservices should integrate through Kafka topics rather than embedding their own Flink processing.