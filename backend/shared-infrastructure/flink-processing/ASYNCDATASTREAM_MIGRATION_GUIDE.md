# AsyncDataStream Migration Guide

## Executive Summary

**Current Problem**: Module2_ContextAssembly uses `.get()` on CompletableFuture, which **blocks Flink threads** and defeats the purpose of async I/O.

**Solution**: Migrate to Flink's `AsyncDataStream` API for truly non-blocking async operations.

**Impact**: **10x-50x throughput improvement** by eliminating thread blocking.

---

## Current Implementation (BLOCKING)

### Location: `Module2_ContextAssembly.java:310-327`

```java
// ❌ WRONG - This blocks the Flink operator thread!
CompletableFuture<FHIRPatientData> fhirPatientFuture = fhirClient.getPatientAsync(patientId);
CompletableFuture<List<Condition>> conditionsFuture = fhirClient.getConditionsAsync(patientId);
CompletableFuture<List<Medication>> medicationsFuture = fhirClient.getMedicationsAsync(patientId);
CompletableFuture<GraphData> neo4jFuture = neo4jClient.queryGraphAsync(patientId);

// Wait for all lookups with 500ms timeout
CompletableFuture.allOf(fhirPatientFuture, conditionsFuture, medicationsFuture, neo4jFuture)
    .get(500, TimeUnit.MILLISECONDS);  // ❌ BLOCKS THREAD!
```

**Problem**: The `.get()` call blocks the Flink operator thread for up to 500ms per first-time patient.

**Impact on Throughput**:
- 1 Flink operator thread
- 500ms blocked per first-time patient
- **Maximum throughput = 2 events/sec per operator**

Under load with 100 first-time patients/sec:
- Need 50 operator threads to handle load
- Memory overhead: 50 threads × ~1MB stack = 50MB+ just for threads
- Context switching overhead degrades performance further

---

## Target Implementation (NON-BLOCKING)

### New Class: `AsyncPatientEnricher.java`

```java
package com.cardiofit.flink.operators;

import org.apache.flink.streaming.api.functions.async.ResultFuture;
import org.apache.flink.streaming.api.functions.async.RichAsyncFunction;
import org.apache.flink.configuration.Configuration;
import com.cardiofit.flink.clients.GoogleFHIRClient;
import com.cardiofit.flink.clients.Neo4jGraphClient;
import com.cardiofit.flink.models.*;
import com.cardiofit.flink.metrics.AsyncIOMetrics;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.Collections;
import java.util.List;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.TimeUnit;

/**
 * Async function for patient enrichment using Flink AsyncDataStream API.
 *
 * This replaces the blocking .get() calls in Module2_ContextAssembly with
 * truly non-blocking async I/O.
 *
 * Key Benefits:
 * - No thread blocking (Flink threads remain free)
 * - Unordered processing (fast lookups complete first)
 * - Automatic backpressure (capacity management)
 * - Built-in timeout handling
 */
public class AsyncPatientEnricher extends RichAsyncFunction<CanonicalEvent, EnrichedEvent> {

    private static final Logger LOG = LoggerFactory.getLogger(AsyncPatientEnricher.class);

    private transient GoogleFHIRClient fhirClient;
    private transient Neo4jGraphClient neo4jClient;
    private transient AsyncIOMetrics fhirMetrics;
    private transient AsyncIOMetrics neo4jMetrics;

    @Override
    public void open(Configuration parameters) throws Exception {
        super.open(parameters);

        // Initialize FHIR client
        String fhirStoreId = parameters.getString("fhir.store.id", "cardiofit-fhir-store");
        String projectId = parameters.getString("gcp.project.id", "cardiofit-project");
        String location = parameters.getString("gcp.location", "us-central1");
        String datasetId = parameters.getString("fhir.dataset.id", "cardiofit-dataset");
        String credentialsPath = parameters.getString("gcp.credentials.path", "/path/to/credentials.json");

        this.fhirClient = new GoogleFHIRClient(fhirStoreId, projectId, location, datasetId, credentialsPath);
        this.fhirClient.initialize();

        // Initialize Neo4j client
        String neo4jUri = parameters.getString("neo4j.uri", "bolt://localhost:7687");
        this.neo4jClient = new Neo4jGraphClient(neo4jUri);

        // Initialize metrics
        this.fhirMetrics = new AsyncIOMetrics(getRuntimeContext().getMetricGroup(), "fhir");
        this.neo4jMetrics = new AsyncIOMetrics(getRuntimeContext().getMetricGroup(), "neo4j");

        LOG.info("AsyncPatientEnricher initialized successfully");
    }

    @Override
    public void asyncInvoke(CanonicalEvent event, ResultFuture<EnrichedEvent> resultFuture) throws Exception {
        String patientId = event.getPatientId();

        // Start metrics tracking
        long fhirStartTime = fhirMetrics.recordRequestStart();
        long neo4jStartTime = neo4jMetrics.recordRequestStart();

        // Initiate parallel async lookups
        CompletableFuture<FHIRPatientData> fhirPatientFuture = fhirClient.getPatientAsync(patientId);
        CompletableFuture<List<Condition>> conditionsFuture = fhirClient.getConditionsAsync(patientId);
        CompletableFuture<List<Medication>> medicationsFuture = fhirClient.getMedicationsAsync(patientId);
        CompletableFuture<GraphData> neo4jFuture = neo4jClient.queryGraphAsync(patientId);

        // Wait for all lookups to complete (non-blocking)
        CompletableFuture.allOf(fhirPatientFuture, conditionsFuture, medicationsFuture, neo4jFuture)
            .whenComplete((result, throwable) -> {
                try {
                    if (throwable != null) {
                        // Handle failure
                        LOG.error("Async lookup failed for patient: {}", patientId, throwable);
                        fhirMetrics.recordFailure(fhirStartTime);
                        neo4jMetrics.recordFailure(neo4jStartTime);

                        // Fallback: create enriched event with empty state
                        EnrichedEvent enriched = createEnrichedEventWithEmptyState(event);
                        resultFuture.complete(Collections.singletonList(enriched));
                    } else {
                        // Success: get results
                        FHIRPatientData fhirPatient = fhirPatientFuture.join();
                        List<Condition> conditions = conditionsFuture.join();
                        List<Medication> medications = medicationsFuture.join();
                        GraphData graphData = neo4jFuture.join();

                        // Record metrics
                        fhirMetrics.recordSuccess(fhirStartTime);
                        neo4jMetrics.recordSuccess(neo4jStartTime);

                        // Create patient snapshot
                        PatientSnapshot snapshot;
                        if (fhirPatient == null) {
                            // 404 from FHIR → new patient
                            LOG.info("Patient {} not found in FHIR store (404) - new patient", patientId);
                            snapshot = PatientSnapshot.createEmpty(patientId);
                        } else {
                            // Existing patient → hydrate from history
                            LOG.info("Patient {} found - hydrating from history", patientId);
                            snapshot = PatientSnapshot.hydrateFromHistory(
                                patientId, fhirPatient, conditions, medications, graphData);
                        }

                        // Create enriched event
                        EnrichedEvent enriched = createEnrichedEvent(event, snapshot);
                        resultFuture.complete(Collections.singletonList(enriched));
                    }
                } catch (Exception e) {
                    LOG.error("Error completing async enrichment for patient: {}", patientId, e);
                    resultFuture.completeExceptionally(e);
                }
            });
    }

    @Override
    public void timeout(CanonicalEvent event, ResultFuture<EnrichedEvent> resultFuture) throws Exception {
        String patientId = event.getPatientId();
        LOG.warn("Timeout (500ms) fetching patient {} - initializing empty state", patientId);

        // Fallback on timeout: create enriched event with empty state
        EnrichedEvent enriched = createEnrichedEventWithEmptyState(event);
        resultFuture.complete(Collections.singletonList(enriched));
    }

    private EnrichedEvent createEnrichedEvent(CanonicalEvent event, PatientSnapshot snapshot) {
        // Create EnrichedEvent from snapshot
        return EnrichedEvent.builder()
            .id(event.getId())
            .patientId(event.getPatientId())
            .encounterId(event.getEncounterId())
            .eventType(event.getEventType())
            .eventTime(event.getEventTime())
            .payload(event.getPayload())
            .patientContext(snapshot.toPatientContext())
            .enrichmentTimestamp(System.currentTimeMillis())
            .build();
    }

    private EnrichedEvent createEnrichedEventWithEmptyState(CanonicalEvent event) {
        PatientSnapshot emptySnapshot = PatientSnapshot.createEmpty(event.getPatientId());
        return createEnrichedEvent(event, emptySnapshot);
    }

    @Override
    public void close() throws Exception {
        super.close();
        if (fhirClient != null) {
            fhirClient.close();
        }
        if (neo4jClient != null) {
            neo4jClient.close();
        }
        LOG.info("AsyncPatientEnricher closed");
    }
}
```

### Updated Module2_ContextAssembly Usage

```java
// In Module2_ContextAssembly.main()

// OLD (BLOCKING):
// DataStream<EnrichedEvent> enrichedStream = canonicalEvents
//     .keyBy(CanonicalEvent::getPatientId)
//     .process(new PatientContextEnrichment());

// NEW (NON-BLOCKING):
DataStream<EnrichedEvent> enrichedStream = AsyncDataStream.unorderedWait(
    canonicalEvents,
    new AsyncPatientEnricher(),
    500,                        // timeout in milliseconds
    TimeUnit.MILLISECONDS,
    150                         // capacity (max concurrent requests)
);
```

---

## Capacity Calculation

### Formula
```
capacity = (events/sec × first_time_rate × avg_latency_sec) × safety_factor
```

### Example for CardioFit System
```
throughput = 1000 events/sec
first_time_patient_rate = 10% (100 patients/sec need async lookup)
avg_latency = 300ms (FHIR) + 200ms (Neo4j) = 500ms (parallel execution)
safety_factor = 1.5

capacity = (1000 × 0.1 × 0.5) × 1.5
capacity = 50 × 1.5
capacity = 75 concurrent requests

Recommended: 150 (2x safety margin for bursts)
```

### Capacity Configuration
```java
AsyncDataStream.unorderedWait(
    canonicalEvents,
    new AsyncPatientEnricher(),
    500,                    // timeout
    TimeUnit.MILLISECONDS,
    150                     // capacity = 150 concurrent requests
);
```

---

## Migration Steps

### Step 1: Create AsyncPatientEnricher Class
- Copy the code above into new file `AsyncPatientEnricher.java`
- Adjust package imports
- Verify compilation

### Step 2: Update Module2_ContextAssembly
- Replace `KeyedProcessFunction` approach with `AsyncDataStream.unorderedWait()`
- Remove blocking `.get()` calls
- Configure capacity based on throughput analysis

### Step 3: Add Flink Async I/O Dependency (Already Included)
```xml
<!-- Flink already includes async I/O support in flink-streaming-java -->
<dependency>
    <groupId>org.apache.flink</groupId>
    <artifactId>flink-streaming-java</artifactId>
    <version>${flink.version}</version>
</dependency>
```

### Step 4: Testing Strategy
```java
@Test
public void testAsyncEnrichment() throws Exception {
    // Create test environment
    StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();
    env.setParallelism(1);

    // Create test data
    DataStream<CanonicalEvent> testEvents = env.fromElements(
        createTestEvent("P-001", EventType.PATIENT_ADMISSION),
        createTestEvent("P-002", EventType.VITAL_SIGNS),
        createTestEvent("P-003", EventType.MEDICATION_ORDER)
    );

    // Apply async enrichment
    DataStream<EnrichedEvent> enriched = AsyncDataStream.unorderedWait(
        testEvents,
        new AsyncPatientEnricher(),
        500,
        TimeUnit.MILLISECONDS,
        10
    );

    // Collect results
    List<EnrichedEvent> results = new ArrayList<>();
    enriched.addSink(new CollectSink(results));

    env.execute("Async Enrichment Test");

    // Verify results
    assertEquals(3, results.size());
    assertTrue(results.stream().allMatch(e -> e.getPatientContext() != null));
}
```

### Step 5: Load Testing
```bash
# Generate load with 1000 events/sec
./load-test.sh --rate 1000 --duration 300s --first-time-rate 0.1

# Monitor metrics
curl http://localhost:9091/metrics | grep async_io

# Expected metrics:
# async_io.fhir.requests_inflight < 150 (capacity limit)
# async_io.fhir.latency_ms.p99 < 500 (99th percentile under 500ms)
# async_io.fhir.requests_success / async_io.fhir.requests_total > 0.99 (>99% success rate)
```

### Step 6: Rollback Plan
```java
// Keep old synchronous code in separate branch: feature/sync-enrichment
// If AsyncDataStream has issues, rollback by:
git checkout feature/sync-enrichment -- src/main/java/com/cardiofit/flink/operators/Module2_ContextAssembly.java

// Redeploy job
flink run -c com.cardiofit.flink.operators.Module2_ContextAssembly target/flink-ehr-intelligence-1.0.0.jar
```

---

## Performance Comparison

### Before (Blocking .get())
```
Throughput per operator thread: 2 events/sec
100 first-time patients/sec → Need 50 operator threads
Memory: 50 threads × 1MB = 50MB
Context switching: HIGH (50 threads)
```

### After (AsyncDataStream)
```
Throughput per operator thread: 200+ events/sec
100 first-time patients/sec → Need 1 operator thread with capacity=150
Memory: 1 thread + 150 async requests = ~20MB
Context switching: NONE (non-blocking)
```

**Improvement**: **100x reduction in thread usage, 10x-50x throughput improvement**

---

## Monitoring and Observability

### Prometheus Metrics to Monitor

```promql
# Success rate (should be >99%)
rate(async_io_fhir_requests_success[5m]) / rate(async_io_fhir_requests_total[5m])

# P99 latency (should be <500ms)
histogram_quantile(0.99, rate(async_io_fhir_latency_ms_bucket[5m]))

# In-flight requests (backpressure indicator - should be <80% of capacity)
async_io_fhir_requests_inflight < 120  # 120 = 80% of capacity 150

# Timeout rate (should be <1%)
rate(async_io_fhir_requests_timeout[5m]) / rate(async_io_fhir_requests_total[5m]) < 0.01

# Circuit breaker state (0=healthy, 1=open, 2=half-open)
async_io_fhir_circuit_breaker_state == 0
```

### Grafana Dashboard Panels

**Panel 1: Async I/O Success Rate**
```
Query: rate(async_io_fhir_requests_success[5m]) / rate(async_io_fhir_requests_total[5m])
Threshold: Red if < 0.99, Yellow if < 0.995, Green if >= 0.995
```

**Panel 2: P99 Latency**
```
Query: histogram_quantile(0.99, rate(async_io_fhir_latency_ms_bucket[5m]))
Threshold: Red if > 500ms, Yellow if > 400ms, Green if <= 400ms
```

**Panel 3: In-Flight Requests (Backpressure)**
```
Query: async_io_fhir_requests_inflight
Threshold: Red if > 120 (80% capacity), Yellow if > 100, Green if <= 100
```

**Panel 4: Circuit Breaker State**
```
Query: async_io_fhir_circuit_breaker_state
Display: 0=Closed (Green), 1=Open (Red), 2=Half-Open (Yellow)
```

---

## Common Issues and Solutions

### Issue 1: Capacity Too Low (Backpressure)
**Symptom**: `async_io_fhir_requests_inflight` consistently at capacity limit

**Solution**: Increase capacity
```java
AsyncDataStream.unorderedWait(
    canonicalEvents,
    new AsyncPatientEnricher(),
    500,
    TimeUnit.MILLISECONDS,
    300  // Increased from 150 to 300
);
```

### Issue 2: High Timeout Rate
**Symptom**: `async_io_fhir_requests_timeout` > 1%

**Possible Causes**:
- FHIR API experiencing latency
- Network issues
- Timeout too aggressive (500ms)

**Solution**: Investigate FHIR API latency, consider increasing timeout to 750ms

### Issue 3: Unordered Results Breaking Logic
**Symptom**: Events processed out of order

**Solution**: Use `orderedWait()` instead of `unorderedWait()`
```java
AsyncDataStream.orderedWait(  // Preserves event order
    canonicalEvents,
    new AsyncPatientEnricher(),
    500,
    TimeUnit.MILLISECONDS,
    150
);
```
**Note**: `orderedWait()` has lower throughput than `unorderedWait()` due to head-of-line blocking

---

## Next Steps After Migration

1. **Circuit Breaker Integration** - Wrap FHIR client with Resilience4j circuit breaker
2. **L1 Cache (Caffeine)** - Add in-memory caching to reduce FHIR API calls
3. **Request Batching** - Batch multiple patient lookups into single FHIR request
4. **L2 Cache (Redis)** - Add distributed caching across Flink instances

---

## Conclusion

Migrating to `AsyncDataStream` is the **highest-impact improvement** for the async I/O implementation. This single change unlocks:

- **10x-50x throughput improvement**
- **100x reduction in thread usage**
- **Automatic backpressure handling**
- **Built-in timeout support**
- **True non-blocking async I/O**

**Estimated Effort**: 2-3 weeks (including testing and rollout)

**Recommended Approach**: Implement in staging environment first, validate with load tests, then progressive rollout to production with careful monitoring.
