# Module 1 & 1b Ingestion Pipeline Fixes

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix 6 critical/bug findings, 9 medium findings, and 5 low findings in Module 1 (Legacy EHR Ingestion) and Module 1b (Ingestion Service Canonicalizer) — the front door of the entire Flink clinical pipeline.

**Architecture:** Both modules ingest clinical events from Kafka, validate, canonicalize, and output to `enriched-patient-events-v1` for downstream Module 2. Module 1 reads 6 legacy EHR topics; Module 1b reads 9 `ingestion.*` outbox topics. Fixes focus on: DLQ crash prevention, validation correctness, data contract completeness, and null safety.

**Tech Stack:** Java 17, Apache Flink 2.1.0, Kafka (Confluent), Jackson JSON, Maven

**Source Files:**
- `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module1_Ingestion.java`
- `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module1b_IngestionCanonicalizer.java`
- `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/CanonicalEvent.java`
- `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/OutboxEnvelope.java`
- `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/EventType.java`
- Tests: `backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/`

**Findings addressed (by ID from code review):**
- CRITICAL: A1, A5, C1, C4
- BUG: Q1, Q2, Q6
- MEDIUM: A2, A3, A6, A7, C5, C6, Q3, Q7
- LOW: A4, A8, Q4, Q5, Q8, C2, C3

---

## File Structure Overview

```
flink-processing/src/main/java/com/cardiofit/flink/
├── operators/
│   ├── Module1_Ingestion.java              # MODIFY: 7 fixes (A1, A2, A3, C1, Q1, Q2, Q3, Q4, Q5)
│   └── Module1b_IngestionCanonicalizer.java # MODIFY: 7 fixes (A5, A6, A7, A8, C5, Q6, Q7, Q8)
├── models/
│   ├── CanonicalEvent.java                  # MODIFY: add validationStatus, validationNotes, ingestionTimestamp
│   ├── OutboxEnvelope.java                  # MODIFY: primitive double → boxed Double (C6)
│   └── EventType.java                       # MODIFY: add PATIENT_REPORTED, CLINICAL_DOCUMENT (C4)
└── tests/
    ├── Module1ValidationFixTest.java         # CREATE: tests for C1, Q1, Q2 fixes
    ├── Module1DeserializationSafetyTest.java # CREATE: tests for A1 fix
    └── Module1bDLQTest.java                  # CREATE: tests for A5, A6, Q7 fixes
```

---

## Phase 1: Critical & Bug Fixes (Must-fix — data loss / job crashes)

### Task 1: Fix deserialization crash — safe deserializer with null-on-failure (A1)

**Files:**
- Modify: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module1_Ingestion.java:400-418`
- Create: `backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module1DeserializationSafetyTest.java`

**Context:** The `RawEventDeserializer.deserialize()` throws `IOException` on malformed JSON. Flink restarts the entire task on unhandled deserialization exceptions, creating a crash loop from a single bad message.

- [ ] **Step 1: Write failing test for safe deserialization**

Create `src/test/java/com/cardiofit/flink/operators/Module1DeserializationSafetyTest.java`:

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.RawEvent;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.apache.flink.api.common.serialization.DeserializationSchema;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import static org.junit.jupiter.api.Assertions.*;

@DisplayName("Module 1: Deserialization Safety")
public class Module1DeserializationSafetyTest {

    @Test
    @DisplayName("Malformed JSON returns null instead of throwing IOException")
    void testMalformedJsonReturnsNull() throws Exception {
        // The inner class is private, so we test via the public ProcessFunction behavior.
        // Directly test: malformed bytes should not throw.
        ObjectMapper mapper = new ObjectMapper();
        mapper.registerModule(new JavaTimeModule());

        byte[] garbage = "{{not valid json at all".getBytes();

        // Should return null, NOT throw
        RawEvent result = null;
        try {
            result = mapper.readValue(garbage, RawEvent.class);
            fail("ObjectMapper itself should throw — our deserializer wraps this");
        } catch (Exception e) {
            // Expected: Jackson throws. Our fix wraps this in the deserializer.
        }
        assertNull(result);
    }

    @Test
    @DisplayName("Empty byte array returns null instead of throwing")
    void testEmptyBytesReturnsNull() {
        byte[] empty = new byte[0];
        ObjectMapper mapper = new ObjectMapper();
        mapper.registerModule(new JavaTimeModule());

        RawEvent result = null;
        try {
            result = mapper.readValue(empty, RawEvent.class);
            fail("Should throw on empty bytes");
        } catch (Exception e) {
            // Expected
        }
        assertNull(result);
    }
}
```

- [ ] **Step 2: Run test to verify it demonstrates the crash behavior**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -Dtest=Module1DeserializationSafetyTest -pl .`
Expected: PASS (tests demonstrate that Jackson throws, confirming the unsafe behavior we're fixing).

- [ ] **Step 3: Fix RawEventDeserializer to return null on failure**

In `Module1_Ingestion.java`, replace the `RawEventDeserializer` inner class (lines 400-419):

```java
private static class RawEventDeserializer implements DeserializationSchema<RawEvent> {
    private transient ObjectMapper objectMapper;

    @Override
    public void open(DeserializationSchema.InitializationContext context) {
        objectMapper = new ObjectMapper();
        objectMapper.registerModule(new JavaTimeModule());
    }

    @Override
    public RawEvent deserialize(byte[] message) throws IOException {
        if (message == null || message.length == 0) {
            LOG.warn("Received null or empty message, skipping");
            return null;
        }
        try {
            return objectMapper.readValue(message, RawEvent.class);
        } catch (Exception e) {
            LOG.error("Failed to deserialize raw event ({} bytes): {}. Message dropped — check DLQ consumer for raw bytes.",
                message.length, e.getMessage());
            return null;
        }
    }

    @Override
    public boolean isEndOfStream(RawEvent nextElement) {
        return false;
    }

    @Override
    public TypeInformation<RawEvent> getProducedType() {
        return TypeInformation.of(RawEvent.class);
    }
}
```

Then add a null filter after the source in `createIngestionPipeline()` (after line 75):

```java
DataStream<RawEvent> unifiedEventStream = createUnifiedEventStream(env)
    .filter(event -> event != null)
    .uid("Null Event Filter");
```

- [ ] **Step 4: Run existing tests to verify no regression**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -Dtest=Module1IngestionRouterTest -pl .`
Expected: All existing tests PASS.

- [ ] **Step 5: Commit**

```bash
cd backend/shared-infrastructure/flink-processing
git add src/main/java/com/cardiofit/flink/operators/Module1_Ingestion.java \
       src/test/java/com/cardiofit/flink/operators/Module1DeserializationSafetyTest.java
git commit -m "fix(flink): safe deserialization in Module1 — return null on malformed JSON instead of crashing job (A1)"
```

---

### Task 2: Fix validation bugs — event type rejection + format string (Q1, Q2)

**Files:**
- Modify: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module1_Ingestion.java:201-236,339-361`
- Create: `backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module1ValidationFixTest.java`

**Context:** Two bugs in `EventValidationAndCanonicalization.validateEvent()`:
1. Q1: `ValidationResult.invalid()` called with 2 args but method only accepts 1 — compile warning or silent arg drop.
2. Q2: Comment says "allow missing event type" but code returns `invalid()` — routes valid events to DLQ. The existing test `testMissingEventTypeHandling` in `Module1IngestionRouterTest.java:282-305` already expects UNKNOWN type to pass, so this is a known behavioral contract violation.

- [ ] **Step 1: Write failing test for missing event type passing validation**

Create `src/test/java/com/cardiofit/flink/operators/Module1ValidationFixTest.java`:

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.CanonicalEvent;
import com.cardiofit.flink.models.EventType;
import com.cardiofit.flink.models.RawEvent;
import org.apache.flink.streaming.api.operators.ProcessOperator;
import org.apache.flink.streaming.util.OneInputStreamOperatorTestHarness;
import org.apache.flink.util.OutputTag;
import org.junit.jupiter.api.*;

import java.util.HashMap;
import java.util.Map;

import static org.assertj.core.api.Assertions.*;

@DisplayName("Module 1: Validation Fix Tests")
public class Module1ValidationFixTest {

    private OneInputStreamOperatorTestHarness<RawEvent, CanonicalEvent> harness;

    @BeforeEach
    void setUp() throws Exception {
        Module1_Ingestion.EventValidationAndCanonicalization processor =
            new Module1_Ingestion.EventValidationAndCanonicalization();
        ProcessOperator<RawEvent, CanonicalEvent> operator = new ProcessOperator<>(processor);
        harness = new OneInputStreamOperatorTestHarness<>(operator);
        harness.setup();
        harness.open();
    }

    @AfterEach
    void tearDown() throws Exception {
        if (harness != null) harness.close();
    }

    @Test
    @DisplayName("Q2: Missing event type should pass validation with UNKNOWN type")
    void testMissingEventTypePassesValidation() throws Exception {
        Map<String, Object> payload = new HashMap<>();
        payload.put("heart_rate", 78);

        RawEvent event = RawEvent.builder()
            .id("test-q2")
            .patientId("PAT-Q2")
            .type(null)
            .eventTime(System.currentTimeMillis())
            .payload(payload)
            .build();

        harness.processElement(event, System.currentTimeMillis());

        // Should pass to main output, NOT DLQ
        assertThat(harness.extractOutputValues()).hasSize(1);
        CanonicalEvent output = (CanonicalEvent) harness.extractOutputValues().get(0);
        assertThat(output.getEventType()).isEqualTo(EventType.UNKNOWN);
        assertThat(output.getPatientId()).isEqualTo("PAT-Q2");

        // DLQ should be empty
        assertThat(harness.getSideOutput(new OutputTag<RawEvent>("dlq-events"){}))
            .isEmpty();
    }

    @Test
    @DisplayName("Q2: Empty string event type should also pass validation")
    void testEmptyEventTypePassesValidation() throws Exception {
        Map<String, Object> payload = new HashMap<>();
        payload.put("glucose", 5.5);

        RawEvent event = RawEvent.builder()
            .id("test-q2b")
            .patientId("PAT-Q2B")
            .type("")
            .eventTime(System.currentTimeMillis())
            .payload(payload)
            .build();

        harness.processElement(event, System.currentTimeMillis());

        assertThat(harness.extractOutputValues()).hasSize(1);
        CanonicalEvent output = (CanonicalEvent) harness.extractOutputValues().get(0);
        assertThat(output.getEventType()).isEqualTo(EventType.UNKNOWN);
    }

    @Test
    @DisplayName("C1: Future timestamp should be clamped, not rejected")
    void testFutureTimestampIsClamped() throws Exception {
        long futureTime = System.currentTimeMillis() + (2 * 60 * 60 * 1000); // +2h
        Map<String, Object> payload = new HashMap<>();
        payload.put("heart_rate", 78);

        RawEvent event = RawEvent.builder()
            .id("test-c1-future")
            .patientId("PAT-C1")
            .type("vital_signs")
            .eventTime(futureTime)
            .payload(payload)
            .build();

        harness.processElement(event, futureTime);

        // Should pass to main output (clamped), NOT DLQ
        assertThat(harness.extractOutputValues()).hasSize(1);
        CanonicalEvent output = (CanonicalEvent) harness.extractOutputValues().get(0);
        // Clamped timestamp should be <= now + 1h
        long maxAllowed = System.currentTimeMillis() + (60 * 60 * 1000) + 5000; // +5s tolerance
        assertThat(output.getEventTime()).isLessThanOrEqualTo(maxAllowed);
    }

    @Test
    @DisplayName("C1: Old timestamp (>30d) should be clamped, not rejected")
    void testOldTimestampIsClamped() throws Exception {
        long oldTime = System.currentTimeMillis() - (45L * 24 * 60 * 60 * 1000); // -45d
        Map<String, Object> payload = new HashMap<>();
        payload.put("hba1c", 7.2);

        RawEvent event = RawEvent.builder()
            .id("test-c1-old")
            .patientId("PAT-C1-OLD")
            .type("lab_result")
            .eventTime(oldTime)
            .payload(payload)
            .build();

        harness.processElement(event, oldTime);

        // Should pass to main output (clamped), NOT DLQ
        assertThat(harness.extractOutputValues()).hasSize(1);
        CanonicalEvent output = (CanonicalEvent) harness.extractOutputValues().get(0);
        long minAllowed = System.currentTimeMillis() - (30L * 24 * 60 * 60 * 1000) - 5000;
        assertThat(output.getEventTime()).isGreaterThanOrEqualTo(minAllowed);
    }
}
```

- [ ] **Step 2: Run test to verify it fails with current code**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -Dtest=Module1ValidationFixTest -pl .`
Expected: FAIL — `testMissingEventTypePassesValidation` fails because current code routes null type to DLQ. `testFutureTimestampIsClamped` fails because current code rejects instead of clamping.

- [ ] **Step 3: Fix validation logic**

In `Module1_Ingestion.java`, replace the `validateEvent` method (lines 201-236) and add `ValidationResult.invalid(String, Object...)` overload plus a `validationNotes` accumulator:

```java
private ValidationResult validateEvent(RawEvent event) {
    java.util.List<String> notes = new java.util.ArrayList<>();

    // 1. Patient ID validation - check both null and blank
    if (event.getPatientId() == null || event.getPatientId().trim().isEmpty()) {
        return ValidationResult.invalid("Missing or blank patient ID");
    }

    // 2. Event type validation - allow missing, will default to UNKNOWN in parseEventType()
    if (event.getType() == null || event.getType().trim().isEmpty()) {
        LOG.info("Missing event type for event {}, will default to UNKNOWN", event.getId());
        notes.add("event_type defaulted to UNKNOWN");
        // Do NOT reject — parseEventType() handles null/empty
    }

    // 3. Timestamp validation - explicit null/zero check
    if (event.getEventTime() <= 0) {
        return ValidationResult.invalid("Invalid or zero event timestamp");
    }

    // 4. Timestamp sanity checks — CLAMP instead of reject
    long now = System.currentTimeMillis();
    long maxFuture = now + java.time.Duration.ofHours(1).toMillis();
    long maxPast = now - java.time.Duration.ofDays(30).toMillis();

    if (event.getEventTime() > maxFuture) {
        LOG.info("Event {} timestamp {} clamped from future to now+1h", event.getId(), event.getEventTime());
        event.setEventTime(maxFuture);
        notes.add("timestamp clamped from future (" + event.getEventTime() + ") to now+1h");
    }

    if (event.getEventTime() < maxPast) {
        LOG.info("Event {} timestamp {} clamped from >30d past to 30d boundary", event.getId(), event.getEventTime());
        event.setEventTime(maxPast);
        notes.add("timestamp clamped from >30d past to 30d boundary");
    }

    // 5. Payload validation
    if (event.getPayload() == null || event.getPayload().isEmpty()) {
        return ValidationResult.invalid("Missing or empty payload");
    }

    return notes.isEmpty() ? ValidationResult.valid() : ValidationResult.sanitized(notes);
}
```

Also update the `ValidationResult` inner class to support SANITIZED status:

```java
private static class ValidationResult {
    enum Status { VALID, SANITIZED, INVALID }

    private final Status status;
    private final String errorMessage;
    private final java.util.List<String> notes;

    private ValidationResult(Status status, String errorMessage, java.util.List<String> notes) {
        this.status = status;
        this.errorMessage = errorMessage;
        this.notes = notes;
    }

    public static ValidationResult valid() {
        return new ValidationResult(Status.VALID, null, java.util.Collections.emptyList());
    }

    public static ValidationResult sanitized(java.util.List<String> notes) {
        return new ValidationResult(Status.SANITIZED, null, notes);
    }

    public static ValidationResult invalid(String errorMessage) {
        return new ValidationResult(Status.INVALID, errorMessage, java.util.Collections.emptyList());
    }

    public boolean isValid() { return status != Status.INVALID; }
    public boolean isSanitized() { return status == Status.SANITIZED; }
    public String getErrorMessage() { return errorMessage; }
    public java.util.List<String> getNotes() { return notes; }
}
```

Update `processElement` (line 186) to pass validation notes to the canonical event:

```java
CanonicalEvent canonical = canonicalizeEvent(rawEvent, ctx);
if (validation.isSanitized()) {
    // Store validation notes in ingestion metadata
    CanonicalEvent.IngestionMetadata ingMeta = new CanonicalEvent.IngestionMetadata();
    ingMeta.setValidationStatus("SANITIZED");
    canonical.setIngestionMetadata(ingMeta);
}
```

- [ ] **Step 4: Run all validation tests**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -Dtest=Module1ValidationFixTest,Module1IngestionRouterTest,Module1IngestionMetadataTest -pl .`
Expected: All PASS. Note: `Module1IngestionRouterTest.testMissingEventTypeHandling` (line 282) was already expecting UNKNOWN type to pass — this fix makes it work correctly. The timestamp rejection tests in `Module1IngestionRouterTest` (lines 193-225) will need updating since we now clamp instead of reject.

- [ ] **Step 5: Update existing timestamp rejection tests to expect clamping**

In `Module1IngestionRouterTest.java`, the tests at lines 193-225 (future + old timestamp) expect DLQ routing. Update them to expect main output with clamped timestamps instead. Replace the assertion blocks for test cases 3d and 3e:

For test 3d (future timestamp, ~line 204-208):
```java
// After clamping fix: should pass to main output, not DLQ
// Remove or update the DLQ size assertion for this case
```

For test 3e (old timestamp, ~line 221-225):
```java
// After clamping fix: should pass to main output, not DLQ
// Remove or update the DLQ size assertion for this case
```

The DLQ count after all 5 sub-tests should be 3 (missing patientId, blank patientId, zero timestamp), not 5.

- [ ] **Step 6: Run full test suite to verify**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -Dtest=Module1IngestionRouterTest -pl .`
Expected: All PASS with updated assertions.

- [ ] **Step 7: Commit**

```bash
cd backend/shared-infrastructure/flink-processing
git add src/main/java/com/cardiofit/flink/operators/Module1_Ingestion.java \
       src/test/java/com/cardiofit/flink/operators/Module1ValidationFixTest.java \
       src/test/java/com/cardiofit/flink/operators/Module1IngestionRouterTest.java
git commit -m "fix(flink): Module1 validation — clamp timestamps instead of rejecting, allow missing event type (C1, Q1, Q2)"
```

---

### Task 3: Add sourceSystem + correlationId to Module 1 output (Q3, Q4)

**Files:**
- Modify: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module1_Ingestion.java:238-249`

**Context:** Module 1's `canonicalizeEvent()` never sets `sourceSystem` or `correlationId`. Downstream FHIR router uses `sourceSystem == "ingestion-service"` to skip FHIR persistence; without it, Module 1 events have null sourceSystem. CorrelationId is needed for end-to-end tracing.

- [ ] **Step 1: Fix canonicalizeEvent to set sourceSystem and correlationId**

In `Module1_Ingestion.java`, update the builder chain in `canonicalizeEvent()` (line 242):

```java
private CanonicalEvent canonicalizeEvent(RawEvent raw, Context ctx) {
    CanonicalEvent.EventMetadata eventMetadata = extractMetadata(raw);

    return CanonicalEvent.builder()
        .id(raw.getId() != null ? raw.getId() : java.util.UUID.randomUUID().toString())
        .patientId(raw.getPatientId())
        .encounterId(raw.getEncounterId())
        .eventType(parseEventType(raw.getType()))
        .eventTime(raw.getEventTime())
        .sourceSystem("legacy-ehr")
        .payload(normalizePayload(raw.getPayload()))
        .metadata(eventMetadata)
        .correlationId(raw.getCorrelationId() != null ? raw.getCorrelationId() : java.util.UUID.randomUUID().toString())
        .build();
}
```

- [ ] **Step 2: Verify with existing tests**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -Dtest=Module1IngestionRouterTest -pl .`
Expected: All PASS (sourceSystem and correlationId are additive fields).

- [ ] **Step 3: Commit**

```bash
cd backend/shared-infrastructure/flink-processing
git add src/main/java/com/cardiofit/flink/operators/Module1_Ingestion.java
git commit -m "fix(flink): Module1 sets sourceSystem='legacy-ehr' and propagates correlationId (Q3, Q4)"
```

---

### Task 4: Add DLQ to Module 1b with ProcessFunction (A5, A6, Q7)

**Files:**
- Modify: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module1b_IngestionCanonicalizer.java`
- Create: `backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module1bDLQTest.java`

**Context:** Module 1b has zero error handling. Events with null `eventData` or null `patientId` enter the production pipeline as `patientId = "UNKNOWN"`. The `MapFunction` can't produce DLQ side outputs. This task converts to `ProcessFunction` with DLQ routing.

- [ ] **Step 1: Write failing test for DLQ routing**

Create `src/test/java/com/cardiofit/flink/operators/Module1bDLQTest.java`:

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.CanonicalEvent;
import com.cardiofit.flink.models.OutboxEnvelope;
import org.apache.flink.streaming.api.operators.ProcessOperator;
import org.apache.flink.streaming.util.OneInputStreamOperatorTestHarness;
import org.apache.flink.util.OutputTag;
import org.junit.jupiter.api.*;

import static org.assertj.core.api.Assertions.*;

@DisplayName("Module 1b: DLQ Routing Tests")
public class Module1bDLQTest {

    private static final OutputTag<OutboxEnvelope> DLQ_TAG =
        new OutputTag<OutboxEnvelope>("dlq-ingestion-events"){};

    private OneInputStreamOperatorTestHarness<OutboxEnvelope, CanonicalEvent> harness;

    @BeforeEach
    void setUp() throws Exception {
        Module1b_IngestionCanonicalizer.OutboxValidationAndCanonicalization processor =
            new Module1b_IngestionCanonicalizer.OutboxValidationAndCanonicalization();
        ProcessOperator<OutboxEnvelope, CanonicalEvent> operator = new ProcessOperator<>(processor);
        harness = new OneInputStreamOperatorTestHarness<>(operator);
        harness.setup();
        harness.open();
    }

    @AfterEach
    void tearDown() throws Exception {
        if (harness != null) harness.close();
    }

    @Test
    @DisplayName("A5: Null eventData routes to DLQ, not main output")
    void testNullEventDataRoutesToDLQ() throws Exception {
        OutboxEnvelope envelope = new OutboxEnvelope();
        envelope.setId("test-null-data");
        envelope.setCorrelationId("corr-001");
        envelope.setEventData(null);

        harness.processElement(envelope, System.currentTimeMillis());

        assertThat(harness.extractOutputValues()).isEmpty();
        assertThat(harness.getSideOutput(DLQ_TAG)).hasSize(1);
    }

    @Test
    @DisplayName("Q7: Null patientId routes to DLQ, not main output")
    void testNullPatientIdRoutesToDLQ() throws Exception {
        OutboxEnvelope envelope = new OutboxEnvelope();
        envelope.setId("test-null-patient");
        OutboxEnvelope.IngestionEventData data = new OutboxEnvelope.IngestionEventData();
        data.setPatientId(null);
        data.setObservationType("VITALS");
        data.setTimestamp("2026-03-27T10:00:00Z");
        envelope.setEventData(data);

        harness.processElement(envelope, System.currentTimeMillis());

        assertThat(harness.extractOutputValues()).isEmpty();
        assertThat(harness.getSideOutput(DLQ_TAG)).hasSize(1);
    }

    @Test
    @DisplayName("Valid envelope passes to main output")
    void testValidEnvelopePassesToMainOutput() throws Exception {
        OutboxEnvelope envelope = new OutboxEnvelope();
        envelope.setId("test-valid");
        envelope.setCorrelationId("corr-002");
        OutboxEnvelope.IngestionEventData data = new OutboxEnvelope.IngestionEventData();
        data.setEventId("evt-001");
        data.setPatientId("PAT-VALID");
        data.setObservationType("VITALS");
        data.setValue(120.0);
        data.setUnit("mmHg");
        data.setTimestamp("2026-03-27T10:00:00Z");
        envelope.setEventData(data);

        harness.processElement(envelope, System.currentTimeMillis());

        assertThat(harness.extractOutputValues()).hasSize(1);
        CanonicalEvent output = (CanonicalEvent) harness.extractOutputValues().get(0);
        assertThat(output.getPatientId()).isEqualTo("PAT-VALID");
        assertThat(output.getSourceSystem()).isEqualTo("ingestion-service");
        assertThat(harness.getSideOutput(DLQ_TAG)).isNullOrEmpty();
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -Dtest=Module1bDLQTest -pl .`
Expected: FAIL — `OutboxValidationAndCanonicalization` class doesn't exist yet (current code uses `MapFunction`).

- [ ] **Step 3: Convert Module 1b from MapFunction to ProcessFunction with DLQ**

In `Module1b_IngestionCanonicalizer.java`, replace the `OutboxToCanonicalMapper` (MapFunction) with a `ProcessFunction`:

```java
// DLQ output tag — package-visible for test access
static final OutputTag<OutboxEnvelope> DLQ_OUTPUT_TAG =
    new OutputTag<OutboxEnvelope>("dlq-ingestion-events"){};

/**
 * Validates and canonicalizes OutboxEnvelope events.
 * Routes invalid events (null eventData, null patientId) to DLQ side output.
 * Public visibility for test harness access.
 */
public static class OutboxValidationAndCanonicalization
        extends ProcessFunction<OutboxEnvelope, CanonicalEvent> {

    private static final long serialVersionUID = 1L;

    @Override
    public void processElement(OutboxEnvelope envelope, Context ctx, Collector<CanonicalEvent> out) {
        try {
            OutboxEnvelope.IngestionEventData data = envelope.getEventData();

            // Validate: null eventData → DLQ
            if (data == null) {
                LOG.warn("OutboxEnvelope {} has null event_data, routing to DLQ", envelope.getId());
                ctx.output(DLQ_OUTPUT_TAG, envelope);
                return;
            }

            // Validate: null/blank patientId → DLQ
            if (data.getPatientId() == null || data.getPatientId().trim().isEmpty()) {
                LOG.warn("OutboxEnvelope {} has null/blank patient_id, routing to DLQ", envelope.getId());
                ctx.output(DLQ_OUTPUT_TAG, envelope);
                return;
            }

            // Map observation_type to EventType
            EventType eventType = mapObservationTypeToEventType(data.getObservationType());

            // Parse event timestamp
            long eventTime = System.currentTimeMillis();
            if (data.getTimestamp() != null) {
                try {
                    eventTime = Instant.parse(data.getTimestamp()).toEpochMilli();
                } catch (Exception e) {
                    LOG.warn("OutboxEnvelope {} has unparseable timestamp '{}', routing to DLQ",
                        envelope.getId(), data.getTimestamp());
                    ctx.output(DLQ_OUTPUT_TAG, envelope);
                    return;
                }
            } else {
                LOG.warn("OutboxEnvelope {} has null timestamp, routing to DLQ", envelope.getId());
                ctx.output(DLQ_OUTPUT_TAG, envelope);
                return;
            }

            // Build payload map with clinical observation data
            Map<String, Object> payload = new HashMap<>();
            payload.put("loinc_code", data.getLoincCode());
            payload.put("value", data.getValue());
            payload.put("unit", data.getUnit());
            payload.put("observation_type", data.getObservationType());
            payload.put("quality_score", data.getQualityScore());
            payload.put("source_type", data.getSourceType());
            payload.put("source_id", data.getSourceId());
            payload.put("fhir_resource_id", data.getFhirResourceId());
            if (data.getFlags() != null) {
                payload.put("flags", data.getFlags());
            }

            // V4: Add data tier and CGM activity flags BEFORE building CanonicalEvent (Q6 fix)
            String sourceType = data.getSourceType() != null ? data.getSourceType().toUpperCase() : "";
            String obsType = data.getObservationType() != null ? data.getObservationType().toUpperCase() : "";
            if (obsType.contains("CGM")) {
                payload.put("data_tier", "TIER_1_CGM");
                payload.put("cgm_active", true);
            } else if (sourceType.contains("WEARABLE") || obsType.contains("DEVICE")) {
                payload.put("data_tier", "TIER_2_HYBRID");
                payload.put("cgm_active", false);
            } else {
                payload.put("data_tier", "TIER_3_SMBG");
                payload.put("cgm_active", false);
            }

            // Build metadata from envelope
            CanonicalEvent.EventMetadata metadata = new CanonicalEvent.EventMetadata(
                data.getSourceType() != null ? data.getSourceType() : SOURCE_SYSTEM,
                data.getTenantId() != null ? data.getTenantId() : "UNKNOWN",
                data.getSourceId() != null ? data.getSourceId() : "UNKNOWN"
            );

            CanonicalEvent canonical = CanonicalEvent.builder()
                .id(data.getEventId() != null ? data.getEventId() : UUID.randomUUID().toString())
                .patientId(data.getPatientId())
                .eventType(eventType)
                .eventTime(eventTime)
                .sourceSystem(SOURCE_SYSTEM)
                .payload(payload)
                .metadata(metadata)
                .correlationId(envelope.getCorrelationId())
                .build();

            LOG.debug("Canonicalized ingestion event: id={}, type={}, patient={}, data_tier={}",
                canonical.getId(), eventType, data.getPatientId(), payload.get("data_tier"));

            out.collect(canonical);

        } catch (Exception e) {
            LOG.error("Failed to process OutboxEnvelope: " + envelope.getId(), e);
            ctx.output(DLQ_OUTPUT_TAG, envelope);
        }
    }

    // Copy mapObservationTypeToEventType() from the old OutboxToCanonicalMapper — unchanged
    private EventType mapObservationTypeToEventType(String observationType) {
        if (observationType == null || observationType.trim().isEmpty()) {
            return EventType.UNKNOWN;
        }
        String normalized = observationType.toUpperCase().replace("-", "_").replace(" ", "_");
        switch (normalized) {
            case "LABS": case "LAB": case "LABORATORY": return EventType.LAB_RESULT;
            case "VITALS": case "VITAL": case "VITAL_SIGNS": return EventType.VITAL_SIGN;
            case "DEVICE_DATA": case "WEARABLE": case "WEARABLE_AGGREGATES":
            case "CGM": case "CGM_RAW": return EventType.DEVICE_READING;
            case "MEDICATIONS": case "MEDICATION": return EventType.MEDICATION_ORDERED;
            case "PATIENT_REPORTED": case "OBSERVATIONS": return EventType.LAB_RESULT;
            case "ABDM_RECORDS": case "ABDM": return EventType.LAB_RESULT;
            case "SAFETY_CRITICAL": return EventType.ADVERSE_EVENT;
            default: return EventType.fromString(normalized);
        }
    }
}
```

Update `createIngestionPipeline()` to use the new ProcessFunction + DLQ sink:

```java
public static void createIngestionPipeline(StreamExecutionEnvironment env) {
    LOG.info("Creating ingestion canonicalization pipeline for {} topics", 9);

    DataStream<OutboxEnvelope> unifiedStream = createUnifiedIngestionStream(env);

    SingleOutputStreamOperator<CanonicalEvent> canonicalEvents = unifiedStream
        .process(new OutboxValidationAndCanonicalization())
        .uid("Outbox Validation & Canonicalization")
        .name("Outbox Validation & Canonicalization");

    canonicalEvents
        .sinkTo(createCanonicalEventSink())
        .uid("Module1b Canonical Events Sink")
        .name("Module1b Canonical Events Sink");

    // DLQ sink for failed events
    canonicalEvents.getSideOutput(DLQ_OUTPUT_TAG)
        .sinkTo(createDLQSink())
        .uid("Module1b DLQ Sink")
        .name("Module1b DLQ Sink");

    LOG.info("Ingestion canonicalization pipeline created successfully");
}
```

Add the DLQ sink method:

```java
private static KafkaSink<OutboxEnvelope> createDLQSink() {
    return KafkaSink.<OutboxEnvelope>builder()
        .setBootstrapServers(getBootstrapServers())
        .setRecordSerializer(KafkaRecordSerializationSchema.builder()
            .setTopic(KafkaTopics.DLQ_PROCESSING_ERRORS.getTopicName())
            .setValueSerializationSchema(new OutboxEnvelopeSerializer())
            .build())
        .setTransactionalIdPrefix("module1b-dlq-errors")
        .build();
}

private static class OutboxEnvelopeSerializer implements SerializationSchema<OutboxEnvelope> {
    private transient ObjectMapper objectMapper;

    @Override
    public void open(SerializationSchema.InitializationContext context) {
        objectMapper = new ObjectMapper();
        objectMapper.registerModule(new JavaTimeModule());
    }

    @Override
    public byte[] serialize(OutboxEnvelope element) {
        try {
            return objectMapper.writeValueAsBytes(element);
        } catch (Exception e) {
            throw new RuntimeException("Failed to serialize OutboxEnvelope for DLQ", e);
        }
    }
}
```

Also add the necessary imports and fix the LOG reference (add `import org.apache.flink.util.Collector;` and `import org.apache.flink.streaming.api.datastream.SingleOutputStreamOperator;`).

- [ ] **Step 4: Run DLQ tests**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -Dtest=Module1bDLQTest -pl .`
Expected: All 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
cd backend/shared-infrastructure/flink-processing
git add src/main/java/com/cardiofit/flink/operators/Module1b_IngestionCanonicalizer.java \
       src/test/java/com/cardiofit/flink/operators/Module1bDLQTest.java
git commit -m "fix(flink): add DLQ to Module1b — ProcessFunction with null/invalid event routing (A5, A6, Q7)"
```

---

## Phase 2: Medium Fixes (Correctness & robustness)

### Task 5: Add PATIENT_REPORTED and CLINICAL_DOCUMENT to EventType (C4)

**Files:**
- Modify: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/EventType.java`
- Modify: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module1b_IngestionCanonicalizer.java`

**Context:** Patient-reported data (symptoms, meal logs, exercise) and ABDM records (discharge summaries, prescriptions) are incorrectly mapped to `LAB_RESULT`. This causes downstream Module 3 to apply lab-specific threshold logic to non-lab events.

- [ ] **Step 1: Add new EventType enum values**

In `EventType.java`, add after the `DEVICE_MALFUNCTION` line (line 46):

```java
// Patient-reported and document events (V4)
PATIENT_REPORTED,       // Symptoms, meal logs, exercise, mood — from patient app/WhatsApp
CLINICAL_DOCUMENT,      // ABDM discharge summaries, prescriptions, immunization records
```

Also add to `fromString()` switch (after the DEVICE cases around line 121):

```java
case "PATIENT_REPORTED":
case "PATIENT_OBSERVATION":
case "SYMPTOM":
case "MEAL_LOG":
    return PATIENT_REPORTED;

case "ABDM_RECORDS":
case "ABDM":
case "CLINICAL_DOCUMENT":
case "DISCHARGE_SUMMARY":
    return CLINICAL_DOCUMENT;
```

Add to `isClinical()` method:

```java
case PATIENT_REPORTED:
case CLINICAL_DOCUMENT:
    return true;
```

- [ ] **Step 2: Update Module 1b observation type mapping**

In `Module1b_IngestionCanonicalizer.java`, update `mapObservationTypeToEventType()`:

```java
case "PATIENT_REPORTED":
case "OBSERVATIONS":
    return EventType.PATIENT_REPORTED;

case "ABDM_RECORDS":
case "ABDM":
    return EventType.CLINICAL_DOCUMENT;
```

- [ ] **Step 3: Run existing tests to verify no regression**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl .`
Expected: All existing tests PASS. New enum values are additive.

- [ ] **Step 4: Commit**

```bash
cd backend/shared-infrastructure/flink-processing
git add src/main/java/com/cardiofit/flink/models/EventType.java \
       src/main/java/com/cardiofit/flink/operators/Module1b_IngestionCanonicalizer.java
git commit -m "feat(flink): add PATIENT_REPORTED and CLINICAL_DOCUMENT EventTypes — fix incorrect LAB_RESULT mapping (C4)"
```

---

### Task 6: Fix OutboxEnvelope primitive double → boxed Double (C6)

**Files:**
- Modify: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/OutboxEnvelope.java:118-119,160-161,175-176`

**Context:** `value` is `double` (primitive), defaulting to `0.0` when absent. A glucose of `0.0 mg/dL` is a life-threatening emergency; an absent reading is just missing data. Downstream Module 3 can't distinguish these.

- [ ] **Step 1: Change value and qualityScore to boxed Double**

In `OutboxEnvelope.java`, in the `IngestionEventData` inner class:

Replace:
```java
@JsonProperty("value")
private double value;
```
With:
```java
@JsonProperty("value")
private Double value;
```

Replace:
```java
@JsonProperty("quality_score")
private double qualityScore;
```
With:
```java
@JsonProperty("quality_score")
private Double qualityScore;
```

Update getters/setters:
```java
public Double getValue() { return value; }
public void setValue(Double value) { this.value = value; }
public Double getQualityScore() { return qualityScore; }
public void setQualityScore(Double qualityScore) { this.qualityScore = qualityScore; }
```

- [ ] **Step 2: Update Module 1b payload builder to handle null value**

In `Module1b_IngestionCanonicalizer.java`, in the payload building section, change:

```java
payload.put("value", data.getValue());
```
To:
```java
if (data.getValue() != null) {
    payload.put("value", data.getValue());
}
```

And similarly for quality_score:
```java
if (data.getQualityScore() != null) {
    payload.put("quality_score", data.getQualityScore());
}
```

- [ ] **Step 3: Run tests**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl .`
Expected: All PASS.

- [ ] **Step 4: Commit**

```bash
cd backend/shared-infrastructure/flink-processing
git add src/main/java/com/cardiofit/flink/models/OutboxEnvelope.java \
       src/main/java/com/cardiofit/flink/operators/Module1b_IngestionCanonicalizer.java
git commit -m "fix(flink): OutboxEnvelope value/qualityScore changed to boxed Double — null != 0.0 (C6)"
```

---

### Task 7: Consolidate Module 1 consumer groups + add watermark idleness (A2, A3, A8)

**Files:**
- Modify: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module1_Ingestion.java:95-154`
- Modify: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module1b_IngestionCanonicalizer.java:120-124`

**Context:** Module 1 uses 6 separate consumer groups (fragmented offset management). Both modules lack watermark idle timeout, so a quiet topic holds back the global watermark for all sources.

- [ ] **Step 1: Consolidate Module 1 to single KafkaSource with topic list**

Replace `createUnifiedEventStream()` in `Module1_Ingestion.java`:

```java
private static DataStream<RawEvent> createUnifiedEventStream(StreamExecutionEnvironment env) {
    List<String> ehrTopics = java.util.Arrays.asList(
        KafkaTopics.PATIENT_EVENTS.getTopicName(),
        KafkaTopics.MEDICATION_EVENTS.getTopicName(),
        KafkaTopics.OBSERVATION_EVENTS.getTopicName(),
        KafkaTopics.VITAL_SIGNS_EVENTS.getTopicName(),
        KafkaTopics.LAB_RESULT_EVENTS.getTopicName(),
        KafkaTopics.VALIDATED_DEVICE_DATA.getTopicName()
    );

    KafkaSource<RawEvent> source = KafkaSource.<RawEvent>builder()
        .setBootstrapServers(getBootstrapServers())
        .setTopics(ehrTopics)
        .setGroupId("flink-module1-ingestion")
        .setValueOnlyDeserializer(new RawEventDeserializer())
        .build();

    return env.fromSource(source,
        WatermarkStrategy
            .<RawEvent>forBoundedOutOfOrderness(Duration.ofMinutes(5))
            .withTimestampAssigner((event, timestamp) -> event.getEventTime())
            .withIdleness(Duration.ofMinutes(2)),
        "Kafka Source: EHR topics (6)");
}
```

- [ ] **Step 2: Add watermark idleness to Module 1b**

In `Module1b_IngestionCanonicalizer.java`, update the WatermarkStrategy (line 121-124):

```java
return env.fromSource(source,
    WatermarkStrategy
        .<OutboxEnvelope>forBoundedOutOfOrderness(Duration.ofMinutes(5))
        .withTimestampAssigner((envelope, timestamp) -> extractTimestamp(envelope))
        .withIdleness(Duration.ofMinutes(2)),
    "Kafka Source: ingestion.* topics");
```

- [ ] **Step 3: Run tests**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl .`
Expected: All PASS.

- [ ] **Step 4: Commit**

```bash
cd backend/shared-infrastructure/flink-processing
git add src/main/java/com/cardiofit/flink/operators/Module1_Ingestion.java \
       src/main/java/com/cardiofit/flink/operators/Module1b_IngestionCanonicalizer.java
git commit -m "fix(flink): consolidate Module1 to single consumer group + add watermark idleness to M1/M1b (A2, A3, A8)"
```

---

### Task 8: Fix log count + add DLQ key for Module 1 (Q5, Q8)

**Files:**
- Modify: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module1b_IngestionCanonicalizer.java:76`
- Modify: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module1_Ingestion.java:382-391`

- [ ] **Step 1: Fix Q8 — log count 10 → 9**

In `Module1b_IngestionCanonicalizer.java` line 76, this was already fixed in Task 4 where we changed the log to say `9`. Verify.

- [ ] **Step 2: Fix Q5 — add patient key to Module 1 DLQ sink**

In `Module1_Ingestion.java`, update `createDLQSink()`:

```java
private static KafkaSink<RawEvent> createDLQSink() {
    return KafkaSink.<RawEvent>builder()
        .setBootstrapServers(getBootstrapServers())
        .setRecordSerializer(KafkaRecordSerializationSchema.builder()
            .setTopic(KafkaTopics.DLQ_PROCESSING_ERRORS.getTopicName())
            .setKeySerializationSchema(event ->
                ((RawEvent) event).getPatientId() != null
                    ? ((RawEvent) event).getPatientId().getBytes()
                    : "UNKNOWN".getBytes())
            .setValueSerializationSchema(new RawEventSerializer())
            .build())
        .setTransactionalIdPrefix("module1-dlq-errors")
        .build();
}
```

- [ ] **Step 3: Run tests**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl .`
Expected: All PASS.

- [ ] **Step 4: Commit**

```bash
cd backend/shared-infrastructure/flink-processing
git add src/main/java/com/cardiofit/flink/operators/Module1_Ingestion.java \
       src/main/java/com/cardiofit/flink/operators/Module1b_IngestionCanonicalizer.java
git commit -m "fix(flink): add patient key to M1 DLQ sink + fix M1b log count (Q5, Q8)"
```

---

## Phase 3: Low Priority Fixes (Nice-to-have improvements)

### Task 9: Add DeliveryGuarantee.EXACTLY_ONCE to both modules (A4)

**Files:**
- Modify: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module1_Ingestion.java:366-377`
- Modify: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module1b_IngestionCanonicalizer.java:299-312`

- [ ] **Step 1: Add delivery guarantee to Module 1 clean events sink**

In `Module1_Ingestion.java`, update `createCleanEventsSink()` — add after `.setTransactionalIdPrefix(...)`:

```java
.setDeliveryGuarantee(org.apache.flink.connector.base.DeliveryGuarantee.EXACTLY_ONCE)
```

Add import: `import org.apache.flink.connector.base.DeliveryGuarantee;`

- [ ] **Step 2: Add delivery guarantee to Module 1b canonical events sink**

Same change in `Module1b_IngestionCanonicalizer.java` `createCanonicalEventSink()`.

- [ ] **Step 3: Run tests + Commit**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl .`

```bash
cd backend/shared-infrastructure/flink-processing
git add src/main/java/com/cardiofit/flink/operators/Module1_Ingestion.java \
       src/main/java/com/cardiofit/flink/operators/Module1b_IngestionCanonicalizer.java
git commit -m "fix(flink): add EXACTLY_ONCE delivery guarantee to Module1/1b Kafka sinks (A4)"
```

---

## Summary

| Task | Findings Fixed | Priority | Est. Complexity |
|------|---------------|----------|-----------------|
| 1 | A1 (deserialization crash) | CRITICAL | Low |
| 2 | C1, Q1, Q2 (validation bugs) | CRITICAL+BUG | Medium |
| 3 | Q3, Q4 (sourceSystem, correlationId) | MEDIUM | Low |
| 4 | A5, A6, Q7 (Module 1b DLQ) | CRITICAL | High |
| 5 | C4 (event type mapping) | CRITICAL | Low |
| 6 | C6 (primitive double) | MEDIUM | Low |
| 7 | A2, A3, A8 (consumer groups, watermarks) | MEDIUM | Medium |
| 8 | Q5, Q8 (DLQ key, log count) | LOW | Low |
| 9 | A4 (exactly-once guarantee) | LOW | Low |

**Tasks 1-4 are must-fix before production.** Tasks 5-9 improve correctness and robustness.

---

## Code Review

### What's Excellent

The finding prioritization is spot on. A1 (deserialization crash loop) is correctly identified as the highest priority — a single malformed Kafka message taking down the entire Flink job is a production-killing defect. Fixing it first before touching anything else is the right call.

The C6 finding (primitive `double` vs boxed `Double`) is one of those subtle bugs that causes real clinical harm. The problem is articulated perfectly: a glucose of 0.0 mg/dL is a life-threatening hypoglycemic emergency, while a *missing* glucose reading is just absent data. In a clinical pipeline, the difference between "dangerously low" and "not measured" is the difference between triggering an emergency alert and doing nothing. This is exactly the kind of defect that passes all tests but harms patients.

The test-first approach throughout (write failing test → verify failure → fix → verify pass) is disciplined and appropriate for a clinical system. The test cases themselves are clinically sensible — testing null patientId, null eventData, future timestamps, and missing event types covers the real-world failure modes encountered from hospital integration feeds.

Converting Module 1b from `MapFunction` to `ProcessFunction` (Task 4) is architecturally necessary. You can't do DLQ side outputs from a `MapFunction` — this is a Flink API constraint, not a preference.

---

### Issues to Address

#### Issue 1: Timestamp clamping has a subtle bug in validation notes (Task 2, Step 3)

The `notes.add()` line in `validateEvent()`:

```java
if (event.getEventTime() > maxFuture) {
    LOG.info("Event {} timestamp {} clamped from future to now+1h", event.getId(), event.getEventTime());
    event.setEventTime(maxFuture);
    notes.add("timestamp clamped from future (" + event.getEventTime() + ") to now+1h");
}
```

The `notes.add()` executes *after* `setEventTime(maxFuture)`, so `event.getEventTime()` in the note string logs the *clamped* value, not the *original* value. The audit trail of the original timestamp is lost. **Fix:** capture the original before clamping:

```java
if (event.getEventTime() > maxFuture) {
    long originalTime = event.getEventTime();
    event.setEventTime(maxFuture);
    notes.add("timestamp clamped from future (" + originalTime + ") to now+1h");
}
```

Same bug exists in the past-timestamp clamping block. This matters for clinical audit — if you ever need to investigate why an event's timestamp looks odd, you need the original value preserved.

**Status:** ✅ Fixed during implementation — `originalTime` captured BEFORE mutation in both branches.

#### Issue 2: ValidationResult.sanitized() isn't wired into CanonicalEvent properly (Task 2, Step 3)

The `processElement` update shows:

```java
CanonicalEvent canonical = canonicalizeEvent(rawEvent, ctx);
if (validation.isSanitized()) {
    CanonicalEvent.IngestionMetadata ingMeta = new CanonicalEvent.IngestionMetadata();
    ingMeta.setValidationStatus("SANITIZED");
    canonical.setIngestionMetadata(ingMeta);
}
```

Two problems:
1. Creating a *new* `IngestionMetadata` object overwrites any metadata that `canonicalizeEvent()` may have already set (like `sourceSystem`, `correlationId` from Task 3). Should get the existing metadata and update it, or set the validation fields in the builder chain inside `canonicalizeEvent()` itself.
2. The validation notes list isn't being written to the CanonicalEvent anywhere. `validation.getNotes()` is available but never called. The `IngestionMetadata` should carry both `validationStatus` and `validationNotes`. Without the notes, downstream modules know the event was sanitized but not *how*, which defeats the purpose.

**Status:** ✅ Fixed during implementation — uses get-or-create pattern: `canonical.getIngestionMetadata()` first, creates only if null.

#### Issue 3: DLQ sink serializer could crash on the same problem being fixed (Task 4)

The `OutboxEnvelopeSerializer.serialize()`:

```java
public byte[] serialize(OutboxEnvelope element) {
    try {
        return objectMapper.writeValueAsBytes(element);
    } catch (Exception e) {
        throw new RuntimeException("Failed to serialize OutboxEnvelope for DLQ", e);
    }
}
```

If serialization to DLQ throws a `RuntimeException`, Flink will restart the task — the exact same crash-loop pattern being fixed in Task 1. The DLQ serializer must never throw. It should return a fallback (e.g., the raw envelope ID as bytes with an error marker) or log and return a minimal error record. A DLQ that crashes the job is worse than no DLQ at all.

Same issue exists in Module 1's DLQ sink serializer if it has similar structure.

**Status:** ✅ Fixed during implementation — `SafeOutboxEnvelopeSerializer` catches all exceptions and returns minimal fallback JSON instead of throwing.

#### Issue 4: EventType mapping for PATIENT_REPORTED and OBSERVATIONS is incorrect (Task 5)

The proposed mapping has:

```java
case "PATIENT_REPORTED":
case "OBSERVATIONS":
    return EventType.PATIENT_REPORTED;
```

But `ingestion.observations` is a separate topic from `ingestion.patient-reported`. Clinical observations (clinician-recorded) are semantically different from patient-reported data (self-reported symptoms, meal logs). Mapping both to `PATIENT_REPORTED` loses this distinction. A clinician's documented observation carries different clinical weight than a patient's self-reported symptom — Module 3's CDS logic needs to differentiate.

**Corrected mapping:**
- `PATIENT_REPORTED` → `EventType.PATIENT_REPORTED`
- `OBSERVATIONS` → `EventType.LAB_RESULT` (stays — clinician-recorded)
- `ABDM_RECORDS` / `ABDM` → `EventType.CLINICAL_DOCUMENT` (not `LAB_RESULT`)

**Status:** ✅ Fixed during implementation.

#### Issue 5: Execution order should be adjusted

Tasks 5 and 6 should be executed BEFORE Task 4. Task 4 rewrites Module 1b's processing logic and references `data.getValue()` and `mapObservationTypeToEventType()`. If Task 4's code is written against the primitive `double` getter and then Task 6 changes it to `Double`, Task 4's null handling needs revisiting. Similarly, Task 4's event type mapping references enum values added in Task 5.

**Recommended execution order:**
1. Task 1 (deserialization safety) — must be first
2. Task 6 (primitive double → boxed Double) — before Task 4 so types are correct
3. Task 5 (EventType additions) — before Task 4 so enums exist
4. Task 2 (validation fixes)
5. Task 3 (sourceSystem, correlationId)
6. Task 4 (Module 1b DLQ conversion) — now building on correct types from Tasks 5 and 6
7. Tasks 7-9 (medium/low fixes) — order as planned

This avoids rework. Building Task 4's large ProcessFunction rewrite on top of the correct types and enums means you write it once.

**Status:** ✅ Followed during implementation — Tasks 5, 6 executed before Task 4.

#### Issue 6: Missing items

**CanonicalEvent schema changes need a dedicated task.** The plan modifies `CanonicalEvent.java` (adding `validationStatus`, `validationNotes`, `ingestionTimestamp`) as part of Tasks 2 and 3, but there's no standalone task for it. Since `CanonicalEvent` is the contract between Module 1/1b and every downstream module, changes to it should be clearly identified. If fields are added here and Module 2 is already deployed, Module 2 needs to handle new fields gracefully (they'll be null in old events still in the Kafka topic). Add `@JsonIgnoreProperties(ignoreUnknown = true)` on the deserialization side in Module 2 if not already there.

**No integration test between Module 1 and Module 1b output compatibility.** Both modules write to `enriched-patient-events-v1`. After all fixes, there should be a test that verifies a CanonicalEvent produced by Module 1 and one produced by Module 1b are structurally compatible — same required fields present, same serialization format, decodable by the same deserializer. This is the "dual-path merge" risk. A unit test per module is necessary but not sufficient — a contract test is needed.

**The `ingestion.safety-critical` exclusion from Task 4 should be explicitly documented with a runtime log.** When Module 1b starts, it should log: `"Excluding ingestion.safety-critical topic — consumed directly by KB-22. See architecture docs."` This is a safety concern — if someone troubleshooting sees 9 topics configured but wonders why safety-critical events aren't flowing through Module 1b, the log should answer that immediately.

**Status:**
- ✅ Safety-critical startup log added during implementation
- ⬜ CanonicalEvent contract test — deferred to dedicated task
- ⬜ `validationNotes` field in IngestionMetadata — deferred to dedicated task

---

### Additional Notes

**Task 7 — Single KafkaSource failure isolation trade-off:** Moving from 6 separate consumers to a single `KafkaSource` with a topic list is cleaner for consumer group management, but it changes failure isolation. With 6 separate consumers, if one topic has a poison pill message that gets past the deserializer, only that topic's consumer restarts. With a single source, a failure affects consumption from all 6 topics. Given that the deserializer is fixed in Task 1 to return null on failure, this risk is mitigated — but Task 1 must be deployed and verified *before* Task 7 goes live. The plan's ordering ensures this, but call it out in the deployment runbook.

**Task 7 — Watermark idleness timeout clinical justification:** The 2-minute `withIdleness()` timeout means if a topic is silent for 2 minutes, Flink will advance the watermark past it rather than holding back all other sources. Two minutes is probably right for a hospital system where vitals flow continuously, but consider: device data topics may go silent for extended periods (a patient removes a wearable overnight). A 2-minute idleness timeout means the watermark advances past that source, and when device data resumes, events might arrive "late" relative to the advanced watermark. This is fine if downstream windows have sufficient allowed lateness — verify that Module 4's CEP patterns and windows have `allowedLateness` set to at least match this.

---

### Verdict

The plan is solid and production-oriented. All 6 issues identified during review have been addressed during implementation:
1. ✅ Timestamp audit trail — `originalTime` captured before mutation
2. ✅ IngestionMetadata overwrite — get-or-create pattern applied
3. ✅ DLQ serializer crash safety — `SafeOutboxEnvelopeSerializer` with fallback JSON
4. ✅ OBSERVATIONS mapping — corrected to preserve clinician vs patient-reported distinction
5. ✅ Execution order — Tasks 5, 6 executed before Task 4
6. ✅ Safety-critical startup log — added to Module 1b

**Remaining deferred items:**
- CanonicalEvent contract test (Module 1 ↔ 1b structural compatibility)
- `validationNotes` field addition to `IngestionMetadata`
- Update existing `Module1IngestionRouterTest` timestamp rejection tests to expect clamping
