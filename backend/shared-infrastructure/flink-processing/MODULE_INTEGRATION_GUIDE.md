# Module Integration & Testing Strategy

## How Module 2 Links with Module 1

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    Module 1: Ingestion                          │
├─────────────────────────────────────────────────────────────────┤
│ Input:  patient-events-v1, medication-events-v1, etc.          │
│ Process: Validate, Generate ID, Normalize                       │
│ Output: enriched-patient-events-v1                             │
└─────────────────────────────────────────────────────────────────┘
                              ↓
                    (Kafka Topic: enriched-patient-events-v1)
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│                 Module 2: Context Assembly                      │
├─────────────────────────────────────────────────────────────────┤
│ Input:  enriched-patient-events-v1 (from Module 1)             │
│ Process: Add patient demographics, encounter info               │
│ Output: context-enriched-events-v1                             │
└─────────────────────────────────────────────────────────────────┘
                              ↓
                    (Kafka Topic: context-enriched-events-v1)
                              ↓
                          Module 3, 4, 5, 6...
```

### Data Flow Example

**Event Journey Through Modules:**

#### Stage 1: You Send (Raw Input)
```json
{
  "patient_id": "P12345",
  "event_time": 1759305006359,
  "type": "vital_signs",
  "payload": {
    "heart_rate": 78,
    "blood_pressure": "120/80"
  }
}
```
**Topic**: `patient-events-v1`

---

#### Stage 2: Module 1 Output (Basic Enrichment)
```json
{
  "eventId": "abc-123-def",
  "patientId": "P12345",
  "eventType": "vital_signs",
  "timestamp": 1759305006359,
  "payload": {
    "heart_rate": 78,
    "blood_pressure": "120/80"
  }
}
```
**Topic**: `enriched-patient-events-v1` ← Module 2 reads from here

---

#### Stage 3: Module 2 Output (Context Enrichment)
```json
{
  "eventId": "abc-123-def",
  "patientId": "P12345",
  "eventType": "vital_signs",
  "timestamp": 1759305006359,
  "payload": {
    "heart_rate": 78,
    "blood_pressure": "120/80"
  },
  // ↓↓↓ Module 2 ADDS these fields ↓↓↓
  "patient": {
    "firstName": "John",
    "lastName": "Doe",
    "dateOfBirth": "1980-05-15",
    "age": 44,
    "gender": "M",
    "mrn": "MRN-67890"
  },
  "encounter": {
    "encounterId": "ENC-2025-001",
    "admitDate": "2025-09-28",
    "department": "ICU",
    "bedNumber": "ICU-5A",
    "attendingPhysician": "Dr. Sarah Johnson"
  },
  "facility": {
    "facilityId": "HOSP-001",
    "facilityName": "CardioFit Medical Center",
    "location": "Building A, Floor 3"
  }
}
```
**Topic**: `context-enriched-events-v1` ← Module 3 reads from here

---

## Integration Pattern

### How Modules Connect

Module 2 is a **Kafka consumer AND producer**:

1. **Consumes** from: `enriched-patient-events-v1` (Module 1's output)
2. **Enriches** by: Looking up patient/encounter data from databases
3. **Produces** to: `context-enriched-events-v1` (Module 3's input)

**Code Pattern** (Module2_ContextAssembly.java):
```java
public static void createContextAssemblyPipeline(StreamExecutionEnvironment env) {
    // Step 1: Read from Module 1's output
    KafkaSource<CanonicalEvent> source = KafkaSource.<CanonicalEvent>builder()
        .setTopics("enriched-patient-events-v1")  // ← Module 1 output
        .build();

    DataStream<CanonicalEvent> enrichedEvents = env.fromSource(source, ...);

    // Step 2: Enrich with context data
    DataStream<ContextEnrichedEvent> contextEnriched = enrichedEvents
        .map(new PatientContextEnricher())     // Add patient demographics
        .map(new EncounterContextEnricher())   // Add encounter info
        .map(new FacilityContextEnricher());   // Add facility info

    // Step 3: Write to next module's input
    KafkaSink<ContextEnrichedEvent> sink = KafkaSink.<ContextEnrichedEvent>builder()
        .setTopic("context-enriched-events-v1")  // ← Module 3 input
        .build();

    contextEnriched.sinkTo(sink);
}
```

### Data Lookups Module 2 Performs

Module 2 needs to fetch data from external systems:

**Patient Demographics** (from database/API):
```java
// Pseudo-code
PatientInfo lookupPatient(String patientId) {
    // Query MongoDB, PostgreSQL, or FHIR server
    return patientRepository.findById(patientId);
}
```

**Encounter Context** (from EHR system):
```java
EncounterInfo lookupEncounter(String patientId) {
    // Query current active encounter
    return encounterRepository.findActiveEncounter(patientId);
}
```

---

## Testing Strategy: Modular → Full Pipeline

### Strategy 1: Bottom-Up Testing (Recommended)

Test each module individually, then combine:

#### Phase 1: Test Module 1 Alone ✅ (DONE)

**Status**: Already completed and working!

```bash
# Submit Module 1 only
bash submit-job.sh ingestion-only development

# Send test event
python3 test_kafka_pipeline.py send vital_signs P12345

# Verify output
# Check topic: enriched-patient-events-v1
```

**Verify**:
- ✅ Events validated correctly
- ✅ IDs generated
- ✅ Fields renamed
- ✅ Output in enriched-patient-events-v1

---

#### Phase 2: Test Module 2 Alone

**Prerequisites**:
1. Module 1 must have created events in `enriched-patient-events-v1`
2. Patient/encounter data must exist in lookup databases

**Steps**:

```bash
# Step 1: Stop Module 1 (or keep it running)
docker exec cardiofit-flink-jobmanager /opt/flink/bin/flink cancel <job-id>

# Step 2: Start Module 2 only
docker exec cardiofit-flink-jobmanager /opt/flink/bin/flink run \
  --detached \
  --class com.cardiofit.flink.FlinkJobOrchestrator \
  /opt/flink/usrlib/flink-ehr-intelligence-1.0.0.jar \
  context-assembly development

# Step 3: Module 2 reads from enriched-patient-events-v1
# (Uses events already there from Module 1)

# Step 4: Check Module 2 output
docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic context-enriched-events-v1 \
  --from-beginning --max-messages 1
```

**Verify**:
- ✅ Patient demographics added
- ✅ Encounter context added
- ✅ Facility info added
- ✅ Original Module 1 data preserved

---

#### Phase 3: Test Modules 1+2 Together

Run both modules as separate jobs:

```bash
# Job 1: Module 1 (ingestion)
bash submit-job.sh ingestion-only development

# Job 2: Module 2 (context assembly)
docker exec cardiofit-flink-jobmanager /opt/flink/bin/flink run \
  --detached \
  --class com.cardiofit.flink.FlinkJobOrchestrator \
  /opt/flink/usrlib/flink-ehr-intelligence-1.0.0.jar \
  context-assembly development

# Send new test event
python3 test_kafka_pipeline.py send vital_signs P12345

# Event flows:
# Input → Module 1 → enriched-patient-events-v1 → Module 2 → context-enriched-events-v1
```

**Verify**:
- ✅ Event appears in `enriched-patient-events-v1`
- ✅ Event appears in `context-enriched-events-v1` with context
- ✅ Both modules running without errors

---

#### Phase 4: Test Each Additional Module

Repeat for Modules 3, 4, 5, 6:

```bash
# Test Module 3 alone
flink run ... semantic-mesh development

# Test Modules 1+2+3 together
# (Three separate jobs or use full pipeline)

# Test Module 4 alone
flink run ... pattern-detection development

# And so on...
```

---

#### Phase 5: Full Pipeline Test

Once all modules tested individually:

```bash
# Stop all individual module jobs
docker exec cardiofit-flink-jobmanager /opt/flink/bin/flink list -r
docker exec cardiofit-flink-jobmanager /opt/flink/bin/flink cancel <job-id-1>
docker exec cardiofit-flink-jobmanager /opt/flink/bin/flink cancel <job-id-2>

# Start full pipeline (all 6 modules in one job)
bash submit-job.sh full-pipeline production

# Send test event
python3 test_kafka_pipeline.py send vital_signs P12345

# Event flows through ALL modules:
# Input → M1 → M2 → M3 → M4 → M5 → M6 → Multiple outputs
```

**Verify**:
- ✅ All intermediate topics have events
- ✅ Final outputs include all enrichments
- ✅ No errors across any module

---

### Strategy 2: Top-Down Testing (Alternative)

Start with full pipeline, then isolate problems:

```bash
# 1. Run full pipeline
bash submit-job.sh full-pipeline development

# 2. Send test event
python3 test_kafka_pipeline.py send vital_signs P12345

# 3. If issues arise, check each module's output topic
docker exec kafka kafka-run-class kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 --topic enriched-patient-events-v1 --time -1

docker exec kafka kafka-run-class kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 --topic context-enriched-events-v1 --time -1

# 4. Find where events stop flowing
# 5. Isolate and test that specific module alone
```

---

## Recommended Testing Workflow

### Week 1: Module 1 (✅ Complete)
- [x] Validate basic enrichment works
- [x] Test with various event types
- [x] Verify error handling (DLQ)
- [x] Performance test (high volume)

### Week 2: Module 2 Setup & Test
- [ ] Set up patient/encounter lookup databases
- [ ] Populate test patient data
- [ ] Test Module 2 alone reading from existing enriched events
- [ ] Verify patient demographics added correctly
- [ ] Test Modules 1+2 together with new events

### Week 3: Module 3 (Semantic)
- [ ] Set up terminology services (SNOMED, LOINC)
- [ ] Test Module 3 alone
- [ ] Test Modules 1+2+3 together

### Week 4: Modules 4, 5, 6
- [ ] Test Module 4 (Pattern Detection) alone
- [ ] Test Module 5 (ML Inference) alone
- [ ] Test Module 6 (Egress Routing) alone
- [ ] Verify each with test data

### Week 5: Full Pipeline
- [ ] Integration test: All modules together
- [ ] End-to-end test: Input → Final outputs
- [ ] Performance test: Handle expected load
- [ ] Failure recovery test: Module failures

---

## Module 2 Specific Setup

### What Module 2 Needs

**1. Patient Data Source**

Option A: MongoDB (if you have patient service running)
```javascript
// Patient collection
{
  "_id": "P12345",
  "firstName": "John",
  "lastName": "Doe",
  "dateOfBirth": "1980-05-15",
  "gender": "M",
  "mrn": "MRN-67890"
}
```

Option B: PostgreSQL
```sql
CREATE TABLE patients (
  patient_id VARCHAR PRIMARY KEY,
  first_name VARCHAR,
  last_name VARCHAR,
  date_of_birth DATE,
  gender CHAR(1),
  mrn VARCHAR
);
```

Option C: FHIR API
```bash
GET https://fhir-server/Patient/P12345
```

**2. Encounter Data Source**

```javascript
// Encounter collection
{
  "encounterId": "ENC-2025-001",
  "patientId": "P12345",
  "admitDate": "2025-09-28",
  "status": "active",
  "department": "ICU",
  "bedNumber": "ICU-5A"
}
```

**3. Flink Connector Configuration**

Module 2 needs database connectors:
```xml
<!-- pom.xml additions for Module 2 -->
<dependency>
    <groupId>org.apache.flink</groupId>
    <artifactId>flink-connector-mongodb</artifactId>
</dependency>
<dependency>
    <groupId>org.apache.flink</groupId>
    <artifactId>flink-connector-jdbc</artifactId>
</dependency>
```

---

## Testing Checklist

### Module 1 Testing ✅
- [x] Single event validation
- [x] Batch event processing
- [x] Invalid event handling (DLQ)
- [x] Field normalization
- [x] ID generation
- [x] Multiple input topics
- [x] Python script integration

### Module 2 Testing ⏸️
- [ ] Patient lookup works
- [ ] Encounter lookup works
- [ ] Facility lookup works
- [ ] Missing patient handling (patient not found)
- [ ] Multiple encounters handling
- [ ] Database connection failover
- [ ] Cache effectiveness (if using Redis)

### Module 1+2 Integration Testing ⏸️
- [ ] Event flows from M1 → M2
- [ ] M1 output compatible with M2 input
- [ ] Both modules running simultaneously
- [ ] No data loss between modules
- [ ] Timing/latency acceptable
- [ ] Memory usage within limits

### Full Pipeline Testing ⏸️
- [ ] All 6 modules running
- [ ] Event flows through all stages
- [ ] All enrichments present in final output
- [ ] Performance meets requirements
- [ ] Error handling works across modules
- [ ] Monitoring and alerting functional

---

## Quick Test Commands

### Test Module 1 Only (Current)
```bash
# Already working!
python3 test_kafka_pipeline.py
```

### Test Module 2 Alone
```bash
# 1. Ensure enriched-patient-events-v1 has events (from Module 1)
python3 test_kafka_pipeline.py send vital_signs P12345

# 2. Start Module 2
docker exec cardiofit-flink-jobmanager /opt/flink/bin/flink run \
  --detached \
  --class com.cardiofit.flink.FlinkJobOrchestrator \
  /opt/flink/usrlib/flink-ehr-intelligence-1.0.0.jar \
  context-assembly development

# 3. Check output
docker exec kafka kafka-run-class kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 --topic context-enriched-events-v1 --time -1
```

### Test Modules 1+2 Together
```bash
# 1. Start both modules
bash submit-job.sh ingestion-only development

docker exec cardiofit-flink-jobmanager /opt/flink/bin/flink run \
  --detached \
  --class com.cardiofit.flink.FlinkJobOrchestrator \
  /opt/flink/usrlib/flink-ehr-intelligence-1.0.0.jar \
  context-assembly development

# 2. Send event (will flow through both)
python3 test_kafka_pipeline.py send vital_signs P12345

# 3. Verify both outputs
# Check M1 output:
docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic enriched-patient-events-v1 \
  --from-beginning --max-messages 1

# Check M2 output:
docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic context-enriched-events-v1 \
  --from-beginning --max-messages 1
```

---

## Summary

**How Module 2 Links with Module 1:**
- Module 2 **reads** from Module 1's output topic (`enriched-patient-events-v1`)
- Module 2 **enriches** with patient/encounter data from databases
- Module 2 **writes** to next topic (`context-enriched-events-v1`)

**Testing Approach:**
1. ✅ **Test Module 1 alone** (DONE - working perfectly!)
2. ⏸️ **Test Module 2 alone** (Next - reads existing M1 events)
3. ⏸️ **Test M1+M2 together** (After - new events flow through both)
4. ⏸️ **Repeat for Modules 3-6**
5. ⏸️ **Test full pipeline** (Final - all modules integrated)

**Benefits of Modular Testing:**
- ✅ Isolate issues to specific modules
- ✅ Faster debugging (test small pieces)
- ✅ Independent development (teams work on different modules)
- ✅ Gradual rollout (deploy modules one by one)
- ✅ Easy rollback (remove problematic module)

**Next Step:**
Set up patient/encounter data sources for Module 2, then test it alone before combining with Module 1.
