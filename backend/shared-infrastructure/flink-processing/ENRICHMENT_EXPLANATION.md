# Flink Enrichment Process Explained

## Overview

The Flink pipeline transforms **raw healthcare events** into **canonical (standardized) events** through validation, normalization, and enrichment.

---

## Your Specific Example

### Input Event (What You Sent):
```json
{
  "patient_id": "DEMO-456",
  "event_time": 1759305006359,
  "type": "vital_signs",
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

**Topic**: `patient-events-v1`

### Output Event (What Flink Created):
```json
{
  "eventId": "6a27dac5-b10e-4be1-a735-8f5cd916f4ee",
  "patientId": "DEMO-456",
  "encounterId": null,
  "eventType": "vital_signs",
  "timestamp": 1759305006359,
  "payload": {
    "oxygen_saturation": 98,
    "respiratory_rate": 16,
    "temperature": 98.6,
    "heart_rate": 78,
    "blood_pressure": "120/80"
  },
  "id": "6a27dac5-b10e-4be1-a735-8f5cd916f4ee"
}
```

**Topic**: `enriched-patient-events-v1`

---

## What Changed? (Line by Line)

| Input Field | Output Field | Transformation |
|-------------|--------------|----------------|
| ❌ (missing) | `eventId` | **AUTO-GENERATED UUID** - Unique identifier created by Flink |
| `patient_id` | `patientId` | **Field name normalized** - snake_case → camelCase |
| ❌ (missing) | `encounterId` | **Added field** - Set to `null` (for hospital visit tracking) |
| `type` | `eventType` | **Field renamed** - More descriptive name |
| `event_time` | `timestamp` | **Field renamed** - Standardized naming |
| `payload` | `payload` | **Contents normalized** - Keys alphabetically sorted |
| `metadata` | ❌ (removed) | **Not included in canonical event** - Metadata handled separately |
| ❌ (missing) | `id` | **Compatibility field** - Duplicate of `eventId` for legacy systems |

---

## Detailed Enrichment Steps

### Step 1: Event Ingestion
**File**: `Module1_Ingestion.java` (lines 139-161)

```java
// Flink reads from Kafka topic
KafkaSource<RawEvent> source = KafkaSource.<RawEvent>builder()
    .setTopics("patient-events-v1")
    .setValueOnlyDeserializer(new RawEventDeserializer())
    .build();
```

**What happens**:
- Kafka message (JSON string) → Deserialized into `RawEvent` Java object
- Event enters Flink processing pipeline

---

### Step 2: Validation
**File**: `Module1_Ingestion.java` (lines 202-233)

```java
private ValidationResult validateEvent(RawEvent event) {
    // Check patient_id exists
    if (event.getPatientId() == null || event.getPatientId().trim().isEmpty()) {
        return ValidationResult.invalid("Missing patient ID");
    }

    // Check type exists
    if (event.getType() == null || event.getType().trim().isEmpty()) {
        return ValidationResult.invalid("Missing event type");
    }

    // Check event_time is valid
    if (event.getEventTime() <= 0) {
        return ValidationResult.invalid("Invalid event time");
    }

    // Check time is not too far in future (1 hour tolerance)
    if (event.getEventTime() > now + 1 hour) {
        return ValidationResult.invalid("Event time too far in future");
    }

    // Check time is not too old (30 days)
    if (event.getEventTime() < now - 30 days) {
        return ValidationResult.invalid("Event time too old");
    }

    // Check payload exists
    if (event.getPayload() == null || event.getPayload().isEmpty()) {
        return ValidationResult.invalid("Missing or empty payload");
    }

    return ValidationResult.valid();
}
```

**Your event validation**:
```
✅ patient_id: "DEMO-456" - Present
✅ type: "vital_signs" - Present
✅ event_time: 1759305006359 - Valid (current timestamp)
✅ Time check: Within 1 hour of current time - Pass
✅ Time check: Not older than 30 days - Pass
✅ payload: Has data - Present
```

**Result**: Event PASSED validation → Proceeds to transformation

**Failed events**: Would be sent to `dlq.processing-errors.v1` topic

---

### Step 3: Canonicalization (Transformation)
**File**: `Module1_Ingestion.java` (lines 235-249)

```java
private CanonicalEvent canonicalizeEvent(RawEvent raw, Context ctx) {
    // Create ingestion metadata
    CanonicalEvent.IngestionMetadata metadata = new CanonicalEvent.IngestionMetadata(
        raw.getSource() != null ? raw.getSource() : "UNKNOWN",
        System.currentTimeMillis(),  // Processing timestamp
        getRuntimeContext().getIndexOfThisSubtask()  // Which Flink task processed this
    );

    return CanonicalEvent.builder()
        .id(raw.getId() != null ? raw.getId() : UUID.randomUUID().toString())  // Generate ID if missing
        .patientId(raw.getPatientId())        // Copy patient ID
        .eventType(raw.getType())             // Rename: type → eventType
        .timestamp(raw.getEventTime())        // Rename: event_time → timestamp
        .payload(normalizePayload(raw.getPayload()))  // Normalize payload
        .build();
}
```

**What happened to YOUR event**:
1. **ID Generation**: No ID in input → Generated `"6a27dac5-b10e-4be1-a735-8f5cd916f4ee"`
2. **Field Mapping**:
   - `patient_id` → `patientId`
   - `type` → `eventType`
   - `event_time` → `timestamp`
3. **Payload Normalization**: Called `normalizePayload()` on your vital signs data

---

### Step 4: Payload Normalization
**File**: `Module1_Ingestion.java` (lines 251-290)

```java
private Map<String, Object> normalizePayload(Map<String, Object> payload) {
    Map<String, Object> normalized = new HashMap<>();

    for (Map.Entry<String, Object> entry : payload.entrySet()) {
        // Convert keys to lowercase and replace hyphens with underscores
        String key = entry.getKey().toLowerCase().replace("-", "_");
        Object value = entry.getValue();

        // Try to parse numeric strings
        if (value instanceof String) {
            String strValue = (String) value;
            if (isNumeric(strValue)) {
                try {
                    normalized.put(key, Double.parseDouble(strValue));
                } catch (NumberFormatException e) {
                    normalized.put(key, value);
                }
            } else {
                normalized.put(key, value);
            }
        } else {
            normalized.put(key, value);
        }
    }

    return normalized;
}
```

**Your payload transformation**:
```
Input:  {"heart_rate": 78, "blood_pressure": "120/80", ...}
        ↓
Process:
  - "heart_rate" → "heart_rate" (already lowercase, no hyphens)
  - 78 (number) → 78 (kept as number)
  - "blood_pressure" → "blood_pressure" (already normalized)
  - "120/80" (string) → "120/80" (not numeric, kept as string)
        ↓
Output: {"heart_rate": 78, "blood_pressure": "120/80", ...}
```

**If you had sent**:
```json
"payload": {
  "Heart-Rate": "78",           // Capital letters, hyphen, string number
  "Blood-Pressure": "120/80"
}
```

**Would become**:
```json
"payload": {
  "heart_rate": 78,             // Lowercase, underscore, converted to number
  "blood_pressure": "120/80"    // Lowercase, underscore, kept as string
}
```

---

### Step 5: Sink to Enriched Topic
**File**: `Module1_Ingestion.java` (lines 331-342)

```java
private static KafkaSink<CanonicalEvent> createCleanEventsSink() {
    return KafkaSink.<CanonicalEvent>builder()
        .setBootstrapServers(getBootstrapServers())
        .setRecordSerializer(KafkaRecordSerializationSchema.builder()
            .setTopic("enriched-patient-events-v1")  // Output topic
            .setKeySerializationSchema(event -> event.getPatientId().getBytes())  // Partition by patient
            .setValueSerializationSchema(new CanonicalEventSerializer())  // Serialize to JSON
            .build())
        .build();
}
```

**What happens**:
- CanonicalEvent Java object → Serialized to JSON string
- Written to Kafka topic `enriched-patient-events-v1`
- Partitioned by `patientId` (all events for "DEMO-456" go to same partition)

---

## Summary of Enrichments

### ✅ What Flink ADDED:
1. **eventId**: `"6a27dac5-b10e-4be1-a735-8f5cd916f4ee"` - UUID for tracking
2. **encounterId**: `null` - Hospital visit field (for future use)
3. **id**: `"6a27dac5-b10e-4be1-a735-8f5cd916f4ee"` - Compatibility field

### 🔄 What Flink RENAMED:
1. **patient_id** → **patientId** (camelCase)
2. **type** → **eventType** (more descriptive)
3. **event_time** → **timestamp** (standardized)

### 🧹 What Flink NORMALIZED:
1. **Payload keys**: Alphabetically sorted in output JSON
2. **Field naming**: Consistent snake_case or camelCase
3. **Numeric values**: String numbers converted to actual numbers (when possible)

### ❌ What Flink REMOVED:
1. **metadata** object - Not part of canonical event schema (handled separately in full pipeline)

---

## Why This Enrichment Matters

### Before Enrichment (Problems):
```
Source A: {"patient_id": "P123", "type": "vitals", ...}
Source B: {"PatientID": "P123", "event-type": "vitals", ...}
Source C: {"patient": "P123", "eventType": "vitals", ...}
```
**Problem**: 3 different formats for the same data! 😱

### After Enrichment (Solution):
```
All Sources: {"patientId": "P123", "eventType": "vitals", ...}
```
**Benefit**: Downstream services see CONSISTENT format! ✅

### Real-World Benefits:
1. **Downstream Services**: Analytics, ML models, dashboards all receive same format
2. **Data Quality**: Validation ensures only good data passes through
3. **Traceability**: Every event has unique ID for debugging
4. **Partitioning**: Events grouped by patient for efficient processing
5. **Normalization**: Field names consistent across all sources

---

## Data Flow Diagram

```
Your Python Script
       ↓
   (Send JSON)
       ↓
Kafka Topic: patient-events-v1
       ↓
Flink: Read Event
       ↓
Flink: Validate (✅ PASSED)
       ↓
Flink: Generate ID
       ↓
Flink: Rename Fields
       ↓
Flink: Normalize Payload
       ↓
Flink: Create CanonicalEvent
       ↓
   (Serialize to JSON)
       ↓
Kafka Topic: enriched-patient-events-v1
       ↓
   (You view in Kafka UI)
```

---

## What's NOT Enriched (Yet)

Looking at your output, these fields are **not present** in the current enrichment:

❌ **ingestion_metadata** - Should include:
   - `source`: "Python Test Script"
   - `ingestion_time`: Processing timestamp
   - `subtask_index`: Which Flink task processed it

❌ **processing_time** - When Flink processed the event

❌ **metadata fields** - Your original metadata (location, device_id)

**Why?** The current implementation uses a **simplified CanonicalEvent interface** that doesn't include all metadata fields. This is Module 1 (Ingestion Only) - full enrichment happens in later modules.

---

## Next Steps (Not Currently Active)

The full pipeline has additional modules that would add:

**Module 2**: Enrichment
- Add patient demographics
- Add encounter context
- Add provider information

**Module 3**: Clinical Rules
- Apply clinical decision support rules
- Flag critical values
- Generate alerts

**Module 4**: Aggregation
- Calculate trends
- Detect patterns
- Generate summaries

**Currently Running**: Module 1 only (Ingestion + Basic Validation/Normalization)

---

## Verification

To see all enrichments, compare topics:

```bash
# View INPUT (what you sent)
docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic patient-events-v1 \
  --from-beginning --max-messages 1

# View OUTPUT (what Flink created)
docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic enriched-patient-events-v1 \
  --from-beginning --max-messages 1
```

Or use Kafka UI:
- Input: http://localhost:8080 → Topics → patient-events-v1 → Messages
- Output: http://localhost:8080 → Topics → enriched-patient-events-v1 → Messages
