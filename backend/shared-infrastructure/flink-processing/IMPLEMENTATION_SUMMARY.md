# Module 1 Schema Improvements - Quick Implementation Guide

## 🎯 What We're Fixing

Your Module 1 FlinkPipeline currently **drops metadata** during event processing. This means downstream systems lose critical clinical context like device IDs, locations, and source systems.

## 📊 Current vs Desired Output

### ❌ Current Output (Missing Metadata)
```json
{
  "id": "6a27dac5-b10e-4be1-a735-8f5cd916f4ee",
  "patientId": "DEMO-456",
  "encounterId": null,
  "eventType": "vital_signs",
  "timestamp": 1759305006359,
  "payload": {
    "heart_rate": 78,
    "blood_pressure": "120/80"
  }
  // ❌ No metadata - device/location info lost!
}
```

### ✅ Desired Output (Metadata Preserved)
```json
{
  "eventId": "6a27dac5-b10e-4be1-a735-8f5cd916f4ee",
  "patientId": "DEMO-456",
  "encounterId": null,
  "eventType": "vital_signs",
  "timestamp": 1759305006359,
  "payload": {
    "heart_rate": 78,
    "blood_pressure": "120/80"
  },
  "metadata": {
    "source": "Python Test Script",
    "location": "ICU Ward",
    "device_id": "MON-001"
  }
}
```

## 🔧 Implementation Checklist

### Step 1: Update CanonicalEvent Model ✅
**File**: `src/main/java/com/cardiofit/flink/models/CanonicalEvent.java`

Add this inner class:
```java
public static class EventMetadata implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("source")
    private String source;

    @JsonProperty("location")
    private String location;

    @JsonProperty("device_id")
    private String deviceId;

    // Constructor, getters, setters
}
```

Add field and builder method:
```java
@JsonProperty("metadata")
private EventMetadata metadata;

public Builder metadata(EventMetadata metadata) {
    event.metadata = metadata;
    return this;
}
```

### Step 2: Update Canonicalization Logic
**File**: `src/main/java/com/cardiofit/flink/operators/Module1_Ingestion.java:235`

Replace `canonicalizeEvent` method:
```java
private CanonicalEvent canonicalizeEvent(RawEvent raw, Context ctx) {
    // Extract metadata from RawEvent
    CanonicalEvent.EventMetadata metadata = extractMetadata(raw);

    return CanonicalEvent.builder()
        .eventId(raw.getId() != null ? raw.getId() : UUID.randomUUID().toString())
        .patientId(raw.getPatientId())
        .encounterId(null)
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
```

### Step 3: Enhanced Validation
Add explicit checks in `validateEvent`:
```java
// Null timestamp check
if (event.getEventTime() <= 0) {
    return ValidationResult.invalid("Invalid or zero event time");
}

// Blank patient ID check
if (event.getPatientId() == null || event.getPatientId().trim().isEmpty()) {
    return ValidationResult.invalid("Missing or blank patient ID");
}
```

### Step 4: Test
Create test with metadata:
```java
RawEvent testEvent = RawEvent.builder()
    .id("test-001")
    .patientId("PAT-123")
    .type("vital_signs")
    .eventTime(System.currentTimeMillis())
    .payload(Map.of("heart_rate", 78))
    .metadata(Map.of(
        "source", "ICU Monitor",
        "location", "ICU-A",
        "device_id", "DEV-456"
    ))
    .build();
```

Run and verify output includes metadata field.

## 🎯 Why This Matters

### Impact on Downstream Systems

| System | How Metadata Helps |
|--------|-------------------|
| **Audit Logging** | Track which device generated each reading |
| **ICU Analytics** | Correlate readings with physical locations |
| **Device Monitoring** | Identify unreliable monitors (e.g., MON-001 failures) |
| **Clinical Context** | Weight ICU readings differently than general ward |
| **Compliance** | HIPAA audit trails require source tracking |

### Real-World Example
```
Without metadata: "Patient heart rate is 78"
With metadata: "Patient heart rate is 78 from MON-001 in ICU Ward"
```

The second provides actionable context for clinicians and auditors.

## 📚 Full Documentation

See [WORKFLOW_MODULE1_IMPROVEMENTS.md](WORKFLOW_MODULE1_IMPROVEMENTS.md) for:
- Detailed implementation phases
- Complete test cases
- Integration validation steps
- Deployment strategy
- Success criteria

## ⏱️ Estimated Time

- **Schema Updates**: 1 hour
- **Logic Changes**: 2 hours
- **Testing**: 2 hours
- **Integration Validation**: 1 hour

**Total**: 4-6 hours

## ✅ Success Validation

Run this command to verify:
```bash
# Start Flink pipeline
cd backend/shared-infrastructure/flink-processing
mvn clean package
java -jar target/flink-processing-1.0.0.jar

# Consume output and check for metadata field
kafka-console-consumer --bootstrap-server localhost:9092 \
  --topic enriched-patient-events-v1 \
  --from-beginning | jq '.metadata'
```

Expected output:
```json
{
  "source": "Python Test Script",
  "location": "ICU Ward",
  "device_id": "MON-001"
}
```

## 🚨 Common Pitfalls

1. **Forgetting to add metadata to builder** → Build won't compile
2. **Not handling null metadata** → NullPointerExceptions downstream
3. **Breaking Module 2** → Test integration after changes
4. **Performance impact** → Benchmark before/after (should be negligible)

## 🔗 Related Files

- [CanonicalEvent.java](src/main/java/com/cardiofit/flink/models/CanonicalEvent.java) - Schema definition
- [Module1_Ingestion.java](src/main/java/com/cardiofit/flink/operators/Module1_Ingestion.java) - Processing logic
- [RawEvent.java](src/main/java/com/cardiofit/flink/models/RawEvent.java) - Input schema

---

**Questions?** Refer to the detailed workflow document or test with sample data first.
