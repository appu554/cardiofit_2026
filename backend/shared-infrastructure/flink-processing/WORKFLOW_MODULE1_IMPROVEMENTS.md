# Module 1 FlinkPipeline - Schema Improvements Workflow

## 🎯 Objective
Enhance Module 1 (Ingestion & Gateway) output schema to retain metadata, improve consistency, and strengthen validation for downstream clinical analytics and audit logging.

## 📋 Current State Analysis

### Identified Issues
1. **Metadata Loss** - RawEvent.metadata (device_id, location, source) dropped during canonicalization
2. **Field Redundancy** - Inconsistent use of 'id' vs 'eventId' naming
3. **Encounter Context** - Properly nullable but needs explicit schema support
4. **Naming Inconsistency** - Mixed snake_case (input) vs camelCase (output)
5. **Validation Gaps** - Missing robust checks for null timestamps and blank patientIds

### Current Data Flow
```
RawEvent (Kafka) → Module1_Ingestion → CanonicalEvent (Output)
                      ↓
                   Validation & Canonicalization
                      ↓
                   Metadata DROPPED ❌
```

---

## 🔄 Implementation Workflow

### Phase 1: Schema Model Updates

#### Task 1.1: Update CanonicalEvent Model
**File**: [CanonicalEvent.java](src/main/java/com/cardiofit/flink/models/CanonicalEvent.java)

**Changes Required**:
1. Add `EventMetadata` inner class to preserve clinical context:
```java
public static class EventMetadata implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("source")
    private String source;

    @JsonProperty("location")
    private String location;

    @JsonProperty("device_id")
    private String deviceId;

    // Constructors, getters, setters
}
```

2. Add metadata field to CanonicalEvent:
```java
@JsonProperty("metadata")
private EventMetadata metadata;
```

3. Update builder to include metadata:
```java
public Builder metadata(EventMetadata metadata) {
    event.metadata = metadata;
    return this;
}
```

4. **JSON Naming Consistency** - Consider renaming for consistency:
```java
@JsonProperty("eventId")  // Instead of "id"
private String eventId;
```

**Rationale**:
- Preserves device/location context for ICU analytics and device reliability monitoring
- Aligns with FHIR Observation and HL7 standards that include provenance metadata
- Supports downstream audit logging requirements

---

#### Task 1.2: Verify EventType Enum
**File**: [EventType.java](src/main/java/com/cardiofit/flink/models/EventType.java)

**Action**:
- Ensure `UNKNOWN` event type exists for malformed events
- Add if missing to support validation fallback

---

### Phase 2: Processing Logic Updates

#### Task 2.1: Update Canonicalization Logic
**File**: [Module1_Ingestion.java:235-249](src/main/java/com/cardiofit/flink/operators/Module1_Ingestion.java#L235-L249)

**Current Implementation**:
```java
private CanonicalEvent canonicalizeEvent(RawEvent raw, Context ctx) {
    // ... existing code
    return CanonicalEvent.builder()
        .id(raw.getId())
        .patientId(raw.getPatientId())
        .eventType(raw.getType())
        .timestamp(raw.getEventTime())
        .payload(normalizePayload(raw.getPayload()))
        .build();  // ❌ Metadata not preserved
}
```

**Updated Implementation**:
```java
private CanonicalEvent canonicalizeEvent(RawEvent raw, Context ctx) {
    // Extract and preserve metadata
    CanonicalEvent.EventMetadata metadata = extractMetadata(raw);

    return CanonicalEvent.builder()
        .eventId(raw.getId() != null ? raw.getId() : UUID.randomUUID().toString())
        .patientId(raw.getPatientId())
        .encounterId(null)  // Explicitly null for Module 1, hydrated in Module 2
        .eventType(parseEventType(raw.getType()))
        .timestamp(raw.getEventTime())
        .payload(normalizePayload(raw.getPayload()))
        .metadata(metadata)  // ✅ Preserve metadata
        .build();
}

private CanonicalEvent.EventMetadata extractMetadata(RawEvent raw) {
    Map<String, String> rawMeta = raw.getMetadata();

    if (rawMeta == null || rawMeta.isEmpty()) {
        return new CanonicalEvent.EventMetadata("UNKNOWN", "UNKNOWN", "UNKNOWN");
    }

    return new CanonicalEvent.EventMetadata(
        rawMeta.getOrDefault("source", "UNKNOWN"),
        rawMeta.getOrDefault("location", "UNKNOWN"),
        rawMeta.getOrDefault("device_id", "UNKNOWN")
    );
}

private String parseEventType(String type) {
    // Default to "UNKNOWN" if missing or invalid
    return (type != null && !type.trim().isEmpty()) ? type : "UNKNOWN";
}
```

**Key Improvements**:
- ✅ Metadata retention for audit logging and device analytics
- ✅ Explicit encounterId null handling (Module 2 responsibility)
- ✅ Default "UNKNOWN" eventType for malformed events
- ✅ Consistent eventId naming

---

#### Task 2.2: Enhanced Validation Logic
**File**: [Module1_Ingestion.java:202-233](src/main/java/com/cardiofit/flink/operators/Module1_Ingestion.java#L202-L233)

**Additional Validations**:
```java
private ValidationResult validateEvent(RawEvent event) {
    // 1. Patient ID validation (not just null, but also blank)
    if (event.getPatientId() == null || event.getPatientId().trim().isEmpty()) {
        return ValidationResult.invalid("Missing or blank patient ID");
    }

    // 2. Event type validation with default fallback
    if (event.getType() == null || event.getType().trim().isEmpty()) {
        LOG.warn("Missing event type for event {}, will default to UNKNOWN", event.getId());
        // Don't fail validation, let canonicalization handle default
    }

    // 3. Timestamp validation - explicit null/zero check
    if (event.getEventTime() <= 0) {
        return ValidationResult.invalid("Invalid or zero event time");
    }

    // 4. Timestamp sanity checks (existing logic)
    long now = System.currentTimeMillis();
    if (event.getEventTime() > now + Duration.ofHours(1).toMillis()) {
        return ValidationResult.invalid("Event time too far in future");
    }

    if (event.getEventTime() < now - Duration.ofDays(30).toMillis()) {
        return ValidationResult.invalid("Event time too old (>30 days)");
    }

    // 5. Payload validation (existing)
    if (event.getPayload() == null || event.getPayload().isEmpty()) {
        return ValidationResult.invalid("Missing or empty payload");
    }

    return ValidationResult.valid();
}
```

**Testing Scenarios**:
- ✅ Null timestamp → Validation fails
- ✅ Blank patientId → Validation fails
- ✅ Missing eventType → Defaults to "UNKNOWN"
- ✅ Valid event with metadata → Metadata preserved

---

### Phase 3: Output Schema Verification

#### Task 3.1: Verify JSON Serialization
**Expected Output Format**:
```json
{
  "eventId": "6a27dac5-b10e-4be1-a735-8f5cd916f4ee",
  "patientId": "DEMO-456",
  "encounterId": null,
  "eventType": "vital_signs",
  "timestamp": 1759305006359,
  "payload": {
    "heart_rate": 78,
    "blood_pressure": "120/80",
    "temperature": 98.6,
    "respiratory_rate": 16,
    "oxygen_saturation": 98
  },
  "metadata": {
    "source": "Python Test Script",
    "location": "ICU Ward",
    "device_id": "MON-001"
  }
}
```

**Verification Steps**:
1. Run Module1_Ingestion with test data
2. Consume from `enriched-patient-events-v1` topic
3. Verify metadata field present and populated
4. Confirm camelCase naming consistency

---

### Phase 4: Testing Strategy

#### Test Case 4.1: Metadata Retention
**Input** (RawEvent):
```json
{
  "id": "test-001",
  "patient_id": "PAT-123",
  "type": "vital_signs",
  "event_time": 1759305006359,
  "payload": {"heart_rate": 78},
  "metadata": {
    "source": "ICU Monitor",
    "location": "ICU-A",
    "device_id": "DEV-456"
  }
}
```

**Expected Output** (CanonicalEvent):
```json
{
  "eventId": "test-001",
  "patientId": "PAT-123",
  "eventType": "vital_signs",
  "timestamp": 1759305006359,
  "payload": {"heart_rate": 78},
  "metadata": {
    "source": "ICU Monitor",
    "location": "ICU-A",
    "device_id": "DEV-456"
  }
}
```

---

#### Test Case 4.2: Validation Robustness
**Malformed Events**:
```java
// Test 1: Null timestamp
RawEvent nullTimestamp = RawEvent.builder()
    .patientId("PAT-123")
    .type("vital_signs")
    .eventTime(0)  // Invalid
    .build();
// Expected: Routed to DLQ

// Test 2: Blank patient ID
RawEvent blankPatient = RawEvent.builder()
    .patientId("   ")  // Blank
    .type("vital_signs")
    .eventTime(System.currentTimeMillis())
    .build();
// Expected: Routed to DLQ

// Test 3: Missing event type
RawEvent missingType = RawEvent.builder()
    .patientId("PAT-123")
    .type(null)  // Missing
    .eventTime(System.currentTimeMillis())
    .payload(Map.of("data", "test"))
    .build();
// Expected: Canonicalized with eventType="UNKNOWN"
```

---

#### Test Case 4.3: Missing Metadata Handling
**Input** (No metadata):
```json
{
  "id": "test-002",
  "patient_id": "PAT-456",
  "type": "medication",
  "event_time": 1759305006359,
  "payload": {"drug": "aspirin"}
}
```

**Expected Output**:
```json
{
  "eventId": "test-002",
  "patientId": "PAT-456",
  "eventType": "medication",
  "timestamp": 1759305006359,
  "payload": {"drug": "aspirin"},
  "metadata": {
    "source": "UNKNOWN",
    "location": "UNKNOWN",
    "device_id": "UNKNOWN"
  }
}
```

---

### Phase 5: Integration Validation

#### Task 5.1: Module 2 Compatibility Check
**Concern**: Module 2 (Context Assembly) expects to hydrate `encounterId`

**Action**:
- Verify Module2_ContextAssembly can handle nullable encounterId
- Confirm metadata field doesn't break downstream processing
- Update Module 2 if needed to leverage metadata for context enrichment

**File to Review**: [Module2_ContextAssembly.java](src/main/java/com/cardiofit/flink/operators/Module2_ContextAssembly.java)

---

#### Task 5.2: Downstream Sink Validation
**Affected Components**:
- Elasticsearch sink (analytics queries may benefit from metadata)
- Neo4j graph sink (device-patient relationships)
- Google FHIR Store (metadata as provenance)

**Action**: Verify sinks can handle new metadata field without errors

---

## 📊 Success Criteria

### ✅ Schema Completeness
- [ ] EventMetadata class added to CanonicalEvent
- [ ] Metadata field preserved during canonicalization
- [ ] JSON output includes metadata block
- [ ] eventId naming consistent throughout

### ✅ Validation Robustness
- [ ] Null/zero timestamp validation passes
- [ ] Blank patientId validation passes
- [ ] Missing eventType defaults to "UNKNOWN"
- [ ] Malformed events routed to DLQ correctly

### ✅ Integration Success
- [ ] Module 2 processes events without errors
- [ ] Downstream sinks accept new schema
- [ ] End-to-end test passes with metadata retention
- [ ] No performance degradation (benchmarked)

### ✅ Documentation
- [ ] Code comments updated to explain metadata preservation
- [ ] Schema changes documented for downstream teams
- [ ] Migration guide provided (if breaking changes)

---

## 🎯 Implementation Priority

### High Priority (P0)
1. **Metadata Retention** - Critical for audit logging and device analytics
2. **Validation Enhancements** - Prevents bad data propagation

### Medium Priority (P1)
3. **Naming Consistency** - Improves downstream API clarity
4. **Testing** - Ensures robustness before production

### Low Priority (P2)
5. **Documentation** - Important but can follow implementation

---

## 🚀 Deployment Strategy

### Development Environment
1. Implement changes in feature branch
2. Run local Flink cluster with test data
3. Validate output schema manually

### Staging Environment
1. Deploy to staging Kafka cluster
2. Run integration tests with Module 2-6
3. Monitor for errors in Flink UI

### Production Environment
1. Blue-green deployment strategy
2. Monitor DLQ topic for unexpected failures
3. Rollback plan: revert to previous schema version

---

## 📚 References

### Related Files
- [Module1_Ingestion.java](src/main/java/com/cardiofit/flink/operators/Module1_Ingestion.java)
- [CanonicalEvent.java](src/main/java/com/cardiofit/flink/models/CanonicalEvent.java)
- [RawEvent.java](src/main/java/com/cardiofit/flink/models/RawEvent.java)
- [Module2_ContextAssembly.java](src/main/java/com/cardiofit/flink/operators/Module2_ContextAssembly.java)

### Standards Alignment
- **FHIR R4**: Observation.device, Provenance resource patterns
- **HL7 v2.x**: MSH segment source/facility information
- **Healthcare Best Practices**: Audit trails require source tracking

---

## 💡 Key Insights

`★ Insight ─────────────────────────────────────`

**1. Metadata as Clinical Context**
Preserving device/location metadata isn't just "nice to have" - it's essential for:
- ICU ward analytics (which monitors are most reliable?)
- Device reliability monitoring (MON-001 failure patterns)
- Audit compliance (where did this reading originate?)
- Clinical decision support (ICU readings weighted differently than general ward)

**2. Validation vs Transformation**
The distinction between validation (reject bad data) and transformation (fix fixable data) is critical:
- Null timestamp → **Reject** (can't be fixed safely)
- Missing eventType → **Transform** (default to UNKNOWN, don't lose event)
- This balance prevents data loss while maintaining quality

**3. Schema Evolution Strategy**
Adding metadata field is backwards-compatible because:
- It's a new field (not modifying existing fields)
- Downstream consumers can ignore it if not needed
- But once available, enables new analytics capabilities

`─────────────────────────────────────────────────`

---

## 🔄 Next Steps

1. **Immediate**: Update CanonicalEvent model with EventMetadata class
2. **Next**: Modify canonicalization logic to preserve metadata
3. **Then**: Enhance validation with robust error handling
4. **Finally**: Create comprehensive test suite and validate with Module 2

**Estimated Implementation Time**: 4-6 hours (includes testing)

---

## ✅ Approval Checklist

Before merging:
- [ ] Code review completed
- [ ] Unit tests passing
- [ ] Integration tests with Module 2 passing
- [ ] No performance regression (benchmark comparison)
- [ ] Documentation updated
- [ ] Downstream teams notified of schema changes
