# Module 2 (Context Assembly) Implementation Plan

Following the architecture specification in `C01_10 Flink - Module 2 patient context architecture.txt`

**IMPORTANT**: Module 2 uses **Google Cloud Healthcare FHIR API** (not REST API to patient-service) for patient context lookups.

---

## Implementation Overview

Module 2 transforms basic enriched events into context-rich events by adding patient demographics, medical history, encounter context, and risk scores through a three-tier state management system.

---

## Data Source Configuration

### Google Cloud Healthcare API Integration

Module 2 will use the **same Google Cloud Healthcare FHIR API configuration** as the patient-service microservice.

**Configuration Source**: [`/Users/apoorvabk/Downloads/cardiofit/backend/services/patient-service/`](file:///Users/apoorvabk/Downloads/cardiofit/backend/services/patient-service/)

**Credentials Location**: [`credentials/google-credentials.json`](file:///Users/apoorvabk/Downloads/cardiofit/backend/services/patient-service/credentials/google-credentials.json)

**API Configuration** (from [run_service.py:28-35](file:///Users/apoorvabk/Downloads/cardiofit/backend/services/patient-service/run_service.py#L28-L35)):
```python
GOOGLE_CLOUD_PROJECT_ID = "cardiofit-905a8"
GOOGLE_CLOUD_LOCATION = "asia-south1"
GOOGLE_CLOUD_DATASET_ID = "clinical-synthesis-hub"
GOOGLE_CLOUD_FHIR_STORE_ID = "fhir-store"
GOOGLE_CLOUD_CREDENTIALS_PATH = "credentials/google-credentials.json"
```

**Service Account**:
- Email: `healthcare-api-client@cardiofit-905a8.iam.gserviceaccount.com`
- Project: `cardiofit-905a8`

**FHIR Store Endpoint Pattern**:
```
https://healthcare.googleapis.com/v1/projects/{project_id}/locations/{location}/datasets/{dataset_id}/fhirStores/{fhir_store_id}/fhir/{resource_type}/{id}
```

**Actual Endpoint**:
```
https://healthcare.googleapis.com/v1/projects/cardiofit-905a8/locations/asia-south1/datasets/clinical-synthesis-hub/fhirStores/fhir-store/fhir/Patient/{patientId}
```

---

## Phase 1: Core Data Models (Foundation)

### 1.1 PatientSnapshot.java (NEW)
Main state container stored in Flink's keyed state:

**Fields**:
- Demographics: `firstName`, `lastName`, `dateOfBirth`, `gender`, `mrn`
- Clinical: `activeConditions` (List<Condition>), `activeMedications` (List<Medication>)
- History: `vitalsHistory` (VitalsHistory), `labHistory` (LabHistory)
- Risk Scores: `sepsisScore`, `deteriorationScore`, `readmissionRisk`
- Encounter: `currentEncounterId`, `admissionTime`, `department`, `room`
- Metadata: `lastUpdated`, `stateVersion`

**Methods**:
- `createEmpty(String patientId)` - Initialize new patient with empty state
- `hydrateFromHistory(patientId, fhirPatient, graphData)` - Populate from FHIR and Neo4j
- `updateWithEvent(CanonicalEvent event)` - Progressive enrichment logic
- `toEnrichedEvent(CanonicalEvent baseEvent)` - Merge state into output event

### 1.2 EncounterContext.java (NEW)
Current encounter details:
- `encounterId`, `encounterType`, `admissionTime`, `department`, `room`, `attendingPhysician`, `careTeam`

### 1.3 VitalsHistory.java (NEW)
Circular buffer for last 10 vital signs:
- `add(VitalSign vital)` - Add new vital, evict oldest if buffer full
- `getRecent(int count)` - Get last N vitals
- `getTrend(String vitalType)` - Calculate trend (improving/declining)

### 1.4 LabHistory.java (NEW)
Circular buffer for last 20 lab results:
- Similar pattern to VitalsHistory

### 1.5 FHIRPatientData.java (NEW)
Data transfer object for Google FHIR API responses:
- Parse FHIR Patient resource JSON into Java object
- Extract demographics, identifiers, contacts
- Handle FHIR-specific structures (HumanName, Address, ContactPoint)

---

## Phase 2: External Integration Clients

### 2.1 GoogleFHIRClient.java (NEW)
**Google Cloud Healthcare API client for FHIR resources**

**Authentication**:
- Use Google Cloud credentials from `credentials/google-credentials.json`
- Service account: `healthcare-api-client@cardiofit-905a8.iam.gserviceaccount.com`
- OAuth2 token-based authentication with automatic refresh

**Methods**:
```java
// Async lookup with 500ms timeout
CompletableFuture<FHIRPatientData> getPatientAsync(String patientId)
    → GET https://healthcare.googleapis.com/.../fhir/Patient/{patientId}

CompletableFuture<List<Condition>> getConditionsAsync(String patientId)
    → GET https://healthcare.googleapis.com/.../fhir/Condition?patient={patientId}

CompletableFuture<List<Medication>> getMedicationsAsync(String patientId)
    → GET https://healthcare.googleapis.com/.../fhir/MedicationRequest?patient={patientId}

CompletableFuture<List<Observation>> getVitalsAsync(String patientId)
    → GET https://healthcare.googleapis.com/.../fhir/Observation?patient={patientId}&category=vital-signs

// State flush on encounter closure
void flushSnapshot(PatientSnapshot snapshot)
    → POST/PUT to FHIR store to persist encounter summary
```

**Configuration**:
```java
public class GoogleFHIRClient {
    private final String projectId = "cardiofit-905a8";
    private final String location = "asia-south1";
    private final String datasetId = "clinical-synthesis-hub";
    private final String fhirStoreId = "fhir-store";
    private final String credentialsPath = "/path/to/google-credentials.json";

    private final String baseUrl = String.format(
        "https://healthcare.googleapis.com/v1/projects/%s/locations/%s/datasets/%s/fhirStores/%s/fhir",
        projectId, location, datasetId, fhirStoreId
    );

    // OAuth2 HTTP client with async support
    private GoogleCredentials credentials;
    private AsyncHttpClient httpClient;
}
```

**Error Handling**:
- 404 Not Found → Patient doesn't exist in FHIR store (truly new patient)
- 403 Forbidden → Authentication issue (log and initialize empty state)
- Timeout (500ms) → Initialize empty state, log for async hydration
- 5xx Errors → Retry once, then initialize empty state

**Dependencies**:
```xml
<!-- Google Cloud Healthcare API -->
<dependency>
    <groupId>com.google.cloud</groupId>
    <artifactId>google-cloud-healthcare</artifactId>
    <version>0.32.0</version>
</dependency>

<!-- Google Auth Library -->
<dependency>
    <groupId>com.google.auth</groupId>
    <artifactId>google-auth-library-oauth2-http</artifactId>
    <version>1.19.0</version>
</dependency>

<!-- Async HTTP Client -->
<dependency>
    <groupId>org.asynchttpclient</groupId>
    <artifactId>async-http-client</artifactId>
    <version>2.12.3</version>
</dependency>
```

### 2.2 Neo4jGraphClient.java (NEW)
Async graph database queries for care relationships:

**Methods**:
- `CompletableFuture<GraphData> queryGraphAsync(String patientId)` - Get care network, relationships, cohort membership
- `void updateCareNetwork(PatientSnapshot snapshot)` - Update graph on encounter close

**Configuration**:
- Neo4j connection details from environment/config
- Cypher queries for patient relationships, care team networks

### 2.3 KafkaConfigLoader.java (ENHANCE)
Add Google FHIR API configuration methods:

```java
public class KafkaConfigLoader {
    // ... existing methods ...

    // Google Cloud Healthcare API configuration
    public static String getGoogleCloudProjectId() {
        return getProperty("google.cloud.project.id", "cardiofit-905a8");
    }

    public static String getGoogleCloudLocation() {
        return getProperty("google.cloud.location", "asia-south1");
    }

    public static String getGoogleCloudDatasetId() {
        return getProperty("google.cloud.dataset.id", "clinical-synthesis-hub");
    }

    public static String getGoogleCloudFhirStoreId() {
        return getProperty("google.cloud.fhir.store.id", "fhir-store");
    }

    public static String getGoogleCloudCredentialsPath() {
        return getProperty("google.cloud.credentials.path",
            "backend/services/patient-service/credentials/google-credentials.json");
    }

    // Neo4j configuration (optional - for Phase 2)
    public static String getNeo4jUri() {
        return getProperty("neo4j.uri", "bolt://localhost:7687");
    }

    public static String getNeo4jUsername() {
        return getProperty("neo4j.username", "neo4j");
    }

    public static String getNeo4jPassword() {
        return getProperty("neo4j.password", "");
    }
}
```

---

## Phase 3: Core Processing Logic

### 3.1 PatientContextProcessor Enhancement (CRITICAL)

**Current State**: Basic structure exists in [Module2_ContextAssembly.java](file:///Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module2_ContextAssembly.java)

**Enhancements Needed**:

#### 3.1.1 State Registration (open method)
```java
@Override
public void open(Configuration config) {
    // Configure 7-day TTL per architecture spec
    StateTtlConfig ttlConfig = StateTtlConfig
        .newBuilder(Time.days(7))
        .setUpdateType(StateTtlConfig.UpdateType.OnCreateAndWrite)
        .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
        .build();

    // Register patient state with TTL
    ValueStateDescriptor<PatientSnapshot> descriptor =
        new ValueStateDescriptor<>("patient-state", PatientSnapshot.class);
    descriptor.enableTimeToLive(ttlConfig);
    patientState = getRuntimeContext().getState(descriptor);

    // Initialize Google FHIR API client
    String credentialsPath = KafkaConfigLoader.getGoogleCloudCredentialsPath();
    fhirClient = new GoogleFHIRClient(
        KafkaConfigLoader.getGoogleCloudProjectId(),
        KafkaConfigLoader.getGoogleCloudLocation(),
        KafkaConfigLoader.getGoogleCloudDatasetId(),
        KafkaConfigLoader.getGoogleCloudFhirStoreId(),
        credentialsPath
    );

    // Initialize Neo4j client (optional)
    neo4jClient = new Neo4jGraphClient(
        KafkaConfigLoader.getNeo4jUri(),
        KafkaConfigLoader.getNeo4jUsername(),
        KafkaConfigLoader.getNeo4jPassword()
    );

    LOG.info("PatientContextProcessor initialized with Google FHIR API");
}
```

#### 3.1.2 First-Time Patient Detection (processElement)
```java
@Override
public void processElement(CanonicalEvent event, Context ctx, Collector<EnrichedEvent> out)
        throws Exception {
    String patientId = event.getPatientId();
    PatientSnapshot snapshot = patientState.value();

    // FIRST-TIME PATIENT LOGIC (state null check)
    if (snapshot == null) {
        LOG.info("First-time patient detected: {}", patientId);
        snapshot = handleFirstTimePatient(patientId, event);
        patientState.update(snapshot);
    }

    // PROGRESSIVE ENRICHMENT
    snapshot.updateWithEvent(event);
    patientState.update(snapshot);

    // CREATE ENRICHED OUTPUT EVENT
    EnrichedEvent enriched = snapshot.toEnrichedEvent(event);
    out.collect(enriched);

    // ENCOUNTER CLOSURE CHECK
    if (isEncounterClosure(event)) {
        LOG.info("Encounter closure detected for patient {}", patientId);
        flushStateToExternalSystems(snapshot);
    }
}
```

#### 3.1.3 Async Lookup with 500ms Timeout (Google FHIR API)
```java
private PatientSnapshot handleFirstTimePatient(String patientId, CanonicalEvent event) {
    LOG.info("Performing async lookups for patient: {}", patientId);

    // Parallel async lookups to Google FHIR API and Neo4j
    CompletableFuture<FHIRPatientData> fhirPatientFuture = fhirClient.getPatientAsync(patientId);
    CompletableFuture<List<Condition>> conditionsFuture = fhirClient.getConditionsAsync(patientId);
    CompletableFuture<List<Medication>> medicationsFuture = fhirClient.getMedicationsAsync(patientId);
    CompletableFuture<GraphData> neo4jFuture = neo4jClient.queryGraphAsync(patientId);

    try {
        // 500ms timeout per architecture specification
        CompletableFuture.allOf(fhirPatientFuture, conditionsFuture, medicationsFuture, neo4jFuture)
            .get(500, TimeUnit.MILLISECONDS);

        FHIRPatientData fhirPatient = fhirPatientFuture.get();
        List<Condition> conditions = conditionsFuture.get();
        List<Medication> medications = medicationsFuture.get();
        GraphData graphData = neo4jFuture.get();

        if (fhirPatient == null) {
            // 404 from Google FHIR API - truly new patient
            LOG.info("Patient {} not found in FHIR store - initializing empty state", patientId);
            return PatientSnapshot.createEmpty(patientId);
        } else {
            // Existing patient - hydrate from FHIR history
            LOG.info("Patient {} found in FHIR store - hydrating from history", patientId);
            return PatientSnapshot.hydrateFromHistory(patientId, fhirPatient, conditions,
                medications, graphData);
        }

    } catch (TimeoutException e) {
        // Timeout - initialize empty, schedule async hydration
        LOG.warn("Timeout fetching patient {} from Google FHIR API - initializing empty state",
            patientId);
        scheduleAsyncHydration(patientId);
        return PatientSnapshot.createEmpty(patientId);

    } catch (ExecutionException e) {
        if (e.getCause() instanceof HttpResponseException) {
            HttpResponseException httpEx = (HttpResponseException) e.getCause();
            if (httpEx.getStatusCode() == 404) {
                // 404 Not Found - new patient
                LOG.info("Patient {} returned 404 from FHIR API - new patient", patientId);
                return PatientSnapshot.createEmpty(patientId);
            }
        }
        LOG.error("Error fetching patient {} from Google FHIR API", patientId, e);
        return PatientSnapshot.createEmpty(patientId);

    } catch (Exception e) {
        LOG.error("Unexpected error fetching patient {}", patientId, e);
        return PatientSnapshot.createEmpty(patientId);
    }
}
```

#### 3.1.4 Progressive Enrichment Logic
```java
// In PatientSnapshot.updateWithEvent(CanonicalEvent event):
public void updateWithEvent(CanonicalEvent event) {
    switch (event.getEventType()) {
        case "vital_signs":
            // Add to vitals circular buffer
            this.vitalsHistory.add(parseVitalSigns(event.getPayload()));
            // Recalculate risk scores based on new vitals
            updateRiskScores();
            break;

        case "medication_events":
            // Update active medications list
            updateActiveMedications(event.getPayload());
            break;

        case "lab_result_events":
            // Add to labs circular buffer
            this.labHistory.add(parseLabResults(event.getPayload()));
            // Recalculate risk scores
            updateRiskScores();
            break;

        case "observation_events":
            // Update clinical observations and conditions
            updateActiveConditions(event.getPayload());
            break;

        case "encounter_events":
            // Update encounter context (admission, transfer, discharge)
            updateEncounterContext(event.getPayload());
            break;
    }

    // Update metadata
    this.lastUpdated = System.currentTimeMillis();
    this.stateVersion++;
}
```

#### 3.1.5 State Flush on Encounter Closure (to Google FHIR API)
```java
private void flushStateToExternalSystems(PatientSnapshot snapshot) {
    LOG.info("Flushing state for patient {} to external systems", snapshot.getPatientId());

    try {
        // Persist complete snapshot to Google FHIR store for historical record
        fhirClient.flushSnapshot(snapshot);
        LOG.info("Successfully flushed snapshot to Google FHIR store");

        // Update Neo4j care network graph
        neo4jClient.updateCareNetwork(snapshot);
        LOG.info("Successfully updated Neo4j care network");

        // State remains in Flink for 7 days (readmission correlation via TTL)
        LOG.info("State will expire after 7 days via TTL");

    } catch (Exception e) {
        LOG.error("Error flushing state for patient {}", snapshot.getPatientId(), e);
        // Don't fail the stream - log error and continue
    }
}

private boolean isEncounterClosure(CanonicalEvent event) {
    return "encounter_events".equals(event.getEventType()) &&
           "discharge".equals(event.getPayload().get("encounterType"));
}
```

### 3.2 EnrichedEvent.java (ENHANCE)
Add patient context fields to output event:

```java
public class EnrichedEvent extends CanonicalEvent {
    // Existing fields from CanonicalEvent
    // ... eventId, patientId, eventType, timestamp, payload ...

    // NEW: Patient context fields
    private PatientDemographics patientDemographics;  // firstName, lastName, DOB, gender, MRN
    private EncounterContext encounterContext;         // encounterId, department, room, care team
    private List<VitalSign> recentVitals;             // Last 3 vitals
    private List<LabResult> recentLabs;               // Last 5 labs
    private List<String> activeConditions;             // Active diagnoses
    private List<String> activeMedications;            // Active meds
    private RiskScores riskScores;                     // Sepsis, deterioration, readmission

    // Constructors, getters, setters
}
```

---

## Phase 4: Maven Dependencies

Add to `pom.xml`:

```xml
<!-- Google Cloud Healthcare API -->
<dependency>
    <groupId>com.google.cloud</groupId>
    <artifactId>google-cloud-healthcare</artifactId>
    <version>0.32.0</version>
</dependency>

<!-- Google Auth Library for OAuth2 -->
<dependency>
    <groupId>com.google.auth</groupId>
    <artifactId>google-auth-library-oauth2-http</artifactId>
    <version>1.19.0</version>
</dependency>

<!-- Async HTTP Client for non-blocking FHIR calls -->
<dependency>
    <groupId>org.asynchttpclient</groupId>
    <artifactId>async-http-client</artifactId>
    <version>2.12.3</version>
</dependency>

<!-- Neo4j Java Driver (optional - for care network graph) -->
<dependency>
    <groupId>org.neo4j.driver</groupId>
    <artifactId>neo4j-java-driver</artifactId>
    <version>5.14.0</version>
</dependency>

<!-- Jackson for JSON parsing (likely already present) -->
<dependency>
    <groupId>com.fasterxml.jackson.core</groupId>
    <artifactId>jackson-databind</artifactId>
    <version>2.15.2</version>
</dependency>
```

---

## Phase 5: Configuration Files

### 5.1 application.properties (or flink-conf.yaml)
```properties
# Google Cloud Healthcare API Configuration
google.cloud.project.id=cardiofit-905a8
google.cloud.location=asia-south1
google.cloud.dataset.id=clinical-synthesis-hub
google.cloud.fhir.store.id=fhir-store
google.cloud.credentials.path=backend/services/patient-service/credentials/google-credentials.json

# Neo4j Configuration (optional)
neo4j.uri=bolt://localhost:7687
neo4j.username=neo4j
neo4j.password=

# Module 2 Settings
module2.async.timeout.ms=500
module2.state.ttl.days=7
```

### 5.2 Credentials Setup
```bash
# Copy Google Cloud credentials to Flink project
cp backend/services/patient-service/credentials/google-credentials.json \
   backend/shared-infrastructure/flink-processing/credentials/

# Set environment variable (alternative to config file)
export GOOGLE_APPLICATION_CREDENTIALS="credentials/google-credentials.json"
```

---

## Phase 6: Testing Strategy

### 6.1 Prerequisites
1. **Google FHIR Store Access**: Verify credentials work
   ```bash
   curl -H "Authorization: Bearer $(gcloud auth print-access-token)" \
     "https://healthcare.googleapis.com/v1/projects/cardiofit-905a8/locations/asia-south1/datasets/clinical-synthesis-hub/fhirStores/fhir-store/fhir/Patient"
   ```

2. **Test Patient Data in FHIR Store**:
   - Patient P12345 (existing patient with history)
   - Patient P999 (new patient, should return 404)

3. **Kafka Topics**: Ensure `enriched-patient-events-v1` has test events from Module 1

### 6.2 Test Scenarios

**Test 1: Module 2 Standalone**
```bash
java -jar flink-ehr-intelligence-1.0.0.jar context-assembly development
# Verify: Reads enriched events, queries Google FHIR API, writes to context-enriched-events-v1
```

**Test 2: First-Time Patient Logic (404 Response)**
- Send event for patient P999 (not in FHIR store)
- Verify: Google FHIR API returns 404 → empty state initialization
- Check logs: "Patient P999 returned 404 from FHIR API - new patient"

**Test 3: Existing Patient Hydration (200 Response)**
- Send event for patient P12345 (exists in FHIR store)
- Verify: Async lookup succeeds → state hydrated with demographics, conditions, meds
- Output event contains patient context

**Test 4: Timeout Handling**
- Simulate slow FHIR API (>500ms response)
- Verify: Timeout exception caught → empty state initialization
- Check logs: "Timeout fetching patient X from Google FHIR API"

**Test 5: Progressive Enrichment**
- Send sequence: vital_signs → medication_event → lab_result
- Verify: State evolves with each event (vitals buffer fills, medications added)

**Test 6: State Persistence Across Flink Restarts**
- Send events, create Flink savepoint
- Kill and restart Flink job from savepoint
- Verify: State restored, no re-lookup to FHIR API

**Test 7: Encounter Closure and State Flush**
- Send discharge event
- Verify: State flushed to Google FHIR store (POST/PUT)
- State remains in Flink (TTL not expired)

**Test 8: Module 1 + Module 2 Integration**
- Start both modules (or use "full-pipeline" mode)
- Send raw event → Module 1 → enriched → Module 2 → context-enriched
- Verify end-to-end flow

### 6.3 Python Test Script Enhancement
Update `test_kafka_pipeline.py`:
```python
def check_context_enriched_events():
    """Check context-enriched-events-v1 topic for Module 2 output"""
    cmd = ['docker', 'exec', 'kafka', 'kafka-console-consumer',
           '--bootstrap-server', 'localhost:9092',
           '--topic', 'context-enriched-events-v1',
           '--from-beginning', '--max-messages', '5']
    # ... parse and display patient context fields ...

def verify_google_fhir_access():
    """Verify Google FHIR API credentials work"""
    # Use gcloud or direct HTTP request with credentials
    # Test patient lookup for P12345
```

---

## Implementation Order

1. **Phase 1**: Create all model classes
   - PatientSnapshot.java
   - EncounterContext.java
   - VitalsHistory.java
   - LabHistory.java
   - FHIRPatientData.java (parse Google FHIR responses)

2. **Phase 2**: Create Google FHIR API client
   - GoogleFHIRClient.java (async lookups, OAuth2 auth, timeout handling)
   - Neo4jGraphClient.java (optional)
   - Update KafkaConfigLoader with Google Cloud config

3. **Phase 3**: Enhance PatientContextProcessor
   - State management with 7-day TTL
   - First-time patient detection
   - Async lookups to Google FHIR API
   - Progressive enrichment
   - State flush on encounter closure

4. **Phase 4**: Update EnrichedEvent model
   - Add patient context fields
   - Update Module2_ContextAssembly pipeline

5. **Phase 5**: Add Maven dependencies and configuration
   - pom.xml updates
   - Copy google-credentials.json
   - application.properties configuration

6. **Phase 6**: Testing
   - Verify Google FHIR API access
   - Test Module 2 standalone
   - Test Module 1+2 integration
   - Verify state persistence and TTL

---

## Success Criteria

✅ PatientSnapshot state correctly initialized for new patients (404 from FHIR API)
✅ State hydrated from Google FHIR API for existing patients (200 response)
✅ Progressive enrichment works (state evolves with each event type)
✅ 500ms timeout handling works (doesn't block pipeline)
✅ State persists across Flink restarts (checkpointing/savepoints)
✅ Encounter closure triggers state flush to Google FHIR store
✅ 7-day TTL configured and working
✅ Output events contain rich patient context from FHIR
✅ Integration with Module 1 works end-to-end
✅ Google Cloud credentials authentication successful

---

## Estimated Effort

- Model classes: 2-3 hours
- Google FHIR API client: 3-4 hours (OAuth2 auth, async requests, error handling)
- Core processing logic: 4-5 hours
- Testing and debugging: 4-5 hours
- **Total: 13-17 hours**

---

## Architecture Compliance

This implementation follows the `C01_10 Flink - Module 2 patient context architecture.txt` specification:

✅ Three-tier state management (Flink State → Google FHIR → Neo4j)
✅ Async lookups with 500ms timeout
✅ First-time patient detection (404 vs. existing)
✅ State hydration from FHIR history
✅ Progressive enrichment pattern
✅ Encounter closure and state flush
✅ 7-day TTL for readmission correlation
✅ Non-blocking stream processing (async I/O)
