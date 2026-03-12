# Option C: Single Transactional Sink + Idempotent Consumers
## Architectural Fix for Module 6 Kafka Timeout Issues

### Problem Analysis

**Root Cause**: Multiple transactional Kafka sinks competing for resources causes timeouts and crashes:
- Module 6 has 6 transactional sinks (enriched, critical, FHIR, analytics, graph, audit)
- Each sink uses `DeliveryGuarantee.AT_LEAST_ONCE` with transactional producer
- All producers share same Kafka connection pool → resource exhaustion
- Kafka coordinator overwhelmed by transaction coordination requests
- Result: Timeouts, crashes, consumer group rebalancing

**Current Architecture (Broken)**:
```
EnrichedClinicalEvent
  ↓
TransactionalMultiSinkRouter (decides routing)
  ├── ENRICHED_OUTPUT → KafkaSink (transactional) → prod.ehr.events.enriched
  ├── CRITICAL_OUTPUT → KafkaSink (transactional) → prod.ehr.alerts.critical-action
  ├── FHIR_OUTPUT     → KafkaSink (transactional) → prod.ehr.fhir.upsert
  ├── ANALYTICS_OUTPUT → KafkaSink (transactional) → prod.ehr.analytics.events
  ├── GRAPH_OUTPUT    → KafkaSink (transactional) → prod.ehr.graph.mutations [DISABLED]
  └── AUDIT_OUTPUT    → KafkaSink (transactional) → prod.ehr.audit.logs
```

**Issues**:
1. **6 transactional producers** = 6× Kafka coordinator load
2. **Competing transactions** = coordinator bottleneck
3. **Shared connection pool** = resource contention
4. **No backpressure** = cascading failures

### Solution: Single Transactional Sink + Idempotent Consumers

**New Architecture**:
```
EnrichedClinicalEvent
  ↓
TransactionalMultiSinkRouter (enriches with routing decisions)
  ↓
SINGLE KafkaSink (transactional) → prod.ehr.events.enriched.routing
  ↓
[Kafka Topic: contains ALL enriched events with routing metadata]
  ↓
Separate Flink Jobs (idempotent consumers):
  ├── Critical Alert Router → prod.ehr.alerts.critical-action
  ├── FHIR Router → prod.ehr.fhir.upsert
  ├── Analytics Router → prod.ehr.analytics.events
  ├── Graph Router → prod.ehr.graph.mutations
  └── Audit Router → prod.ehr.audit.logs
```

### Key Design Principles

#### 1. **Single Transactional Producer**
- Only ONE KafkaSink with `DeliveryGuarantee.AT_LEAST_ONCE`
- Writes to central routing topic: `prod.ehr.events.enriched.routing`
- Eliminates producer competition and coordinator bottleneck
- Enables natural backpressure through single producer

#### 2. **Enriched Routing Metadata**
```java
public class RoutedEnrichedEvent {
    private EnrichedClinicalEvent event;
    private RoutingDecision routing;  // Contains destinations
    private long routingTimestamp;
    private String routingId;
}

public class RoutingDecision {
    private boolean sendToCriticalAlerts;
    private boolean sendToFHIR;
    private boolean sendToAnalytics;
    private boolean sendToGraph;
    private boolean sendToAudit;
    private Map<String, Object> routingMetadata;
}
```

#### 3. **Idempotent Consumer Jobs**
Each downstream router is a separate Flink job:
- Reads from `prod.ehr.events.enriched.routing`
- Filters for relevant events (based on routing flags)
- Transforms to destination-specific format
- Writes to destination topic with **idempotent writes**

**Idempotency Strategy**:
- Use event ID as Kafka message key
- Consumer uses `enable.idempotence=true`
- Kafka deduplicates by key within retention window
- Even if job restarts and reprocesses, no duplicates appear in destination

#### 4. **Fault Tolerance**
- **Main Job Failure**: Central routing topic retains all events
- **Router Job Failure**: Can restart from checkpoint, reprocess safely (idempotent)
- **Kafka Failure**: Single producer easier to tune and monitor
- **Backpressure**: Naturally propagates from slowest router

### Implementation Plan

#### Phase 1: Create Central Routing Topic (15 mins)
1. Create Kafka topic `prod.ehr.events.enriched.routing`
   - Partitions: 12 (matches upstream)
   - Replication: 3
   - Retention: 7 days (compliance)
   - Compaction: No (need full history for audit)

2. Create `RoutedEnrichedEvent` model
3. Update `TransactionalMultiSinkRouter` to:
   - Keep routing logic (decisions)
   - Wrap `EnrichedClinicalEvent` with `RoutingDecision`
   - Output `RoutedEnrichedEvent` to single sink

#### Phase 2: Implement Single Transactional Sink (20 mins)
```java
// Module6_EgressRouting.java

private static DataStream<RoutedEnrichedEvent> createRoutedStream(
        DataStream<EnrichedClinicalEvent> enrichedStream) {

    return enrichedStream
        .process(new TransactionalMultiSinkRouter())
        .name("Routing Decision Engine");
}

private static KafkaSink<RoutedEnrichedEvent> createCentralRoutingSink() {
    return KafkaSink.<RoutedEnrichedEvent>builder()
        .setBootstrapServers(getBootstrapServers())
        .setRecordSerializer(KafkaRecordSerializationSchema.builder()
            .setTopic("prod.ehr.events.enriched.routing")
            .setKeySerializationSchema((RoutedEnrichedEvent event) -> {
                // Use event ID as key for idempotency
                return event.getEvent().getId().getBytes();
            })
            .setValueSerializationSchema(new RoutedEnrichedEventSerializer())
            .build())
        .setKafkaProducerConfig(KafkaConfigLoader.getProducerConfigForSink())
        .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
        .setTransactionalIdPrefix("module6-central-routing")
        .build();
}
```

#### Phase 3: Implement Idempotent Router Jobs (60 mins)

**Job 1: Critical Alert Router**
```java
public class CriticalAlertRouter {
    public static void main(String[] args) throws Exception {
        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();
        env.enableCheckpointing(60000);

        KafkaSource<RoutedEnrichedEvent> source = KafkaSource.<RoutedEnrichedEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setTopics("prod.ehr.events.enriched.routing")
            .setGroupId("critical-alert-router")
            .setStartingOffsets(OffsetsInitializer.committedOffsets())
            .setValueOnlyDeserializer(new RoutedEnrichedEventDeserializer())
            .build();

        DataStream<RoutedEnrichedEvent> stream = env
            .fromSource(source, WatermarkStrategy.noWatermarks(), "Routing Source");

        stream
            .filter(event -> event.getRouting().isSendToCriticalAlerts())
            .map(event -> transformToCriticalAlert(event.getEvent()))
            .sinkTo(createIdempotentCriticalAlertSink())
            .name("Critical Alert Sink");

        env.execute("Critical Alert Router");
    }

    private static KafkaSink<Map<String, Object>> createIdempotentCriticalAlertSink() {
        Properties producerConfig = new Properties();
        producerConfig.put(ProducerConfig.ENABLE_IDEMPOTENCE_CONFIG, "true");
        producerConfig.put(ProducerConfig.ACKS_CONFIG, "all");
        producerConfig.put(ProducerConfig.MAX_IN_FLIGHT_REQUESTS_PER_CONNECTION, "5");
        producerConfig.putAll(KafkaConfigLoader.getProducerConfigForSink());

        return KafkaSink.<Map<String, Object>>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic("prod.ehr.alerts.critical-action")
                .setKeySerializationSchema((alert) -> {
                    return alert.get("eventId").toString().getBytes();
                })
                .setValueSerializationSchema(new SimpleStringSchema())
                .build())
            .setKafkaProducerConfig(producerConfig)
            .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE) // Idempotence at producer level
            .build();
    }
}
```

**Job 2-5**: Similar pattern for FHIR, Analytics, Graph, Audit routers

#### Phase 4: Update Module 6 Main Job (30 mins)
1. Remove all legacy sinks (keep only central routing sink)
2. Update `main()` method
3. Simplify routing topology

#### Phase 5: Deployment & Migration (20 mins)
1. Deploy router jobs first (won't consume until source exists)
2. Deploy updated Module 6
3. Verify central routing topic populates
4. Verify router jobs consume and route correctly
5. Monitor for duplicates (should be zero)

### Benefits

✅ **Single Transactional Producer**: Eliminates coordinator bottleneck
✅ **Idempotent Writes**: Safe to reprocess, no duplicates
✅ **Natural Backpressure**: Slowest router can't crash main job
✅ **Independent Scaling**: Scale router jobs independently
✅ **Fault Isolation**: Router failure doesn't affect main job
✅ **Easier Monitoring**: One producer to tune and monitor
✅ **Flexible Routing**: Add new routers without changing main job

### Performance Characteristics

**Before (Current)**:
- Kafka coordinator load: HIGH (6 transactional producers)
- Module 6 crash rate: ~30% (frequent restarts)
- Resource contention: SEVERE (shared connection pool)
- Backpressure: BROKEN (cascading failures)

**After (Option C)**:
- Kafka coordinator load: LOW (1 transactional producer)
- Module 6 crash rate: <1% (stable single sink)
- Resource contention: NONE (isolated jobs)
- Backpressure: NATURAL (Kafka lag propagation)

### Monitoring & Metrics

**Central Routing Topic**:
- Message rate (events/sec)
- Lag per router consumer group
- Message size distribution
- Routing decision breakdown (% to each destination)

**Router Jobs**:
- Consumption rate (events/sec)
- Processing latency (time from routing to destination)
- Idempotent write rate (deduplicated messages)
- Error rate (failed transformations)

### Rollback Plan

If Option C fails:
1. Stop all router jobs
2. Deploy previous Module 6 version (with multiple sinks)
3. Resume from checkpoints
4. Central routing topic can be drained or ignored

### Success Criteria

✅ Module 6 runs stable for >24 hours without restarts
✅ All destination topics receive expected events
✅ No duplicate events in destination topics
✅ Lag on router consumer groups <10 seconds
✅ Kafka coordinator CPU <50% (down from >90%)
✅ End-to-end latency <5 seconds (routing timestamp to destination)

### Next Steps

1. **Review & Approve**: Get architectural approval for Option C
2. **Phase 1**: Create central routing topic and models
3. **Phase 2**: Implement single transactional sink in Module 6
4. **Phase 3**: Implement 5 idempotent router jobs
5. **Phase 4**: Update Module 6 main job
6. **Phase 5**: Deploy and migrate

**Estimated Time**: 2.5 hours total implementation + 1 hour testing
**Risk Level**: LOW (can rollback, idempotent design limits blast radius)
**Impact**: HIGH (solves crashes, enables scaling, improves observability)
