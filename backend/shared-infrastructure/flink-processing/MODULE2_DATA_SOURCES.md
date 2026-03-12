# Module 2: Data Sources for Patient Context

## Overview

Module 2 (Context Assembly) enriches events by looking up patient, encounter, and facility data from **your existing backend services and databases**.

---

## Available Data Sources in Your System

### ✅ 1. Patient Service (Port 8003)

**Location**: `/backend/services/patient-service/`

**Database**: MongoDB or Google Healthcare API (FHIR)

**What it provides**:
```json
{
  "patientId": "P12345",
  "firstName": "John",
  "lastName": "Doe",
  "dateOfBirth": "1980-05-15",
  "gender": "male",
  "mrn": "MRN-67890",
  "address": {
    "street": "123 Main St",
    "city": "Boston",
    "state": "MA",
    "zip": "02101"
  },
  "contactInfo": {
    "phone": "+1-555-0123",
    "email": "john.doe@example.com"
  }
}
```

**API Endpoints**:
```bash
# Get patient by ID
GET http://localhost:8003/patients/{patientId}

# Search patients
GET http://localhost:8003/patients?search=John

# FHIR format
GET http://localhost:8003/fhir/Patient/{patientId}
```

**How Module 2 uses it**:
```java
// Flink connects to patient-service API or MongoDB directly
PatientInfo patient = patientServiceClient.getPatient(event.getPatientId());
enrichedEvent.setPatient(patient);
```

---

### ✅ 2. Encounter Service (Port varies)

**Location**: `/backend/services/encounter-service/`

**Database**: MongoDB/PostgreSQL

**What it provides**:
```json
{
  "encounterId": "ENC-2025-001",
  "patientId": "P12345",
  "status": "in-progress",
  "class": "inpatient",
  "period": {
    "start": "2025-09-28T08:00:00Z",
    "end": null
  },
  "location": {
    "department": "ICU",
    "room": "ICU-5A",
    "bed": "Bed-1",
    "facilityId": "HOSP-001"
  },
  "careTeam": {
    "attendingPhysician": "Dr. Sarah Johnson",
    "primaryNurse": "Nurse Smith"
  },
  "diagnosis": [
    {
      "condition": "Hypertension",
      "code": "I10"
    }
  ]
}
```

**API Endpoints**:
```bash
# Get active encounter for patient
GET http://localhost:8XXX/encounters/patient/{patientId}/active

# Get encounter by ID
GET http://localhost:8XXX/encounters/{encounterId}
```

**How Module 2 uses it**:
```java
// Get current active encounter for the patient
EncounterInfo encounter = encounterService.getActiveEncounter(event.getPatientId());
enrichedEvent.setEncounter(encounter);
```

---

### ✅ 3. Organization Service

**Location**: `/backend/services/organization-service/`

**What it provides**:
```json
{
  "facilityId": "HOSP-001",
  "facilityName": "CardioFit Medical Center",
  "type": "Hospital",
  "address": {
    "street": "456 Healthcare Blvd",
    "city": "Boston",
    "state": "MA",
    "zip": "02115"
  },
  "departments": [
    {
      "departmentId": "ICU",
      "departmentName": "Intensive Care Unit",
      "floor": 3,
      "building": "A"
    }
  ]
}
```

---

### ✅ 4. MongoDB (Direct Access)

**Patient Collection** (`patients` database):
```javascript
db.patients.findOne({"patientId": "P12345"})
```

**Encounter Collection**:
```javascript
db.encounters.findOne({
  "patientId": "P12345",
  "status": "active"
})
```

**Connection String** (from patient-service config):
```python
# Typical MongoDB connection
MONGODB_URL = "mongodb://localhost:27017"
# Or MongoDB Atlas
MONGODB_URL = "mongodb+srv://user:pass@cluster.mongodb.net/patients"
```

---

### ✅ 5. Google Healthcare API (FHIR)

**If configured** (from patient-service/app/core/config.py):

```python
USE_GOOGLE_HEALTHCARE_API = True
PROJECT_ID = "your-project"
LOCATION = "us-central1"
DATASET_ID = "cardiofit-dataset"
FHIR_STORE_ID = "patient-store"
```

**FHIR Endpoints**:
```
GET https://healthcare.googleapis.com/v1/projects/{project}/locations/{location}/datasets/{dataset}/fhirStores/{store}/fhir/Patient/{patientId}
```

---

## Module 2 Integration Options

### Option 1: REST API Calls (Recommended for Development)

**Advantages**:
- ✅ Uses existing services (no duplication)
- ✅ Respects business logic and validation
- ✅ Easy to test and debug
- ✅ Automatic updates when services change

**Implementation**:
```java
// Module2_ContextAssembly.java
public class PatientAPIEnricher extends RichMapFunction<CanonicalEvent, ContextEnrichedEvent> {
    private transient AsyncHttpClient httpClient;

    @Override
    public void open(Configuration parameters) {
        // Initialize HTTP client
        httpClient = new DefaultAsyncHttpClient();
    }

    @Override
    public ContextEnrichedEvent map(CanonicalEvent event) throws Exception {
        // Call patient service API
        String url = "http://localhost:8003/patients/" + event.getPatientId();
        Response response = httpClient.prepareGet(url).execute().get();

        PatientInfo patient = parsePatientResponse(response.getResponseBody());

        // Create enriched event with patient context
        return ContextEnrichedEvent.builder()
            .fromCanonicalEvent(event)
            .patient(patient)
            .build();
    }
}
```

**Dependency** (add to pom.xml):
```xml
<dependency>
    <groupId>org.asynchttpclient</groupId>
    <artifactId>async-http-client</artifactId>
    <version>2.12.3</version>
</dependency>
```

---

### Option 2: Direct Database Access (Best Performance)

**Advantages**:
- ✅ Faster (no HTTP overhead)
- ✅ Lower latency
- ✅ Better for high-volume streams

**Implementation**:
```java
// Use Flink's MongoDB connector
public class PatientMongoEnricher extends RichMapFunction<CanonicalEvent, ContextEnrichedEvent> {
    private transient MongoClient mongoClient;
    private transient MongoCollection<Document> patients;

    @Override
    public void open(Configuration parameters) {
        // Connect to MongoDB
        mongoClient = MongoClients.create("mongodb://localhost:27017");
        MongoDatabase db = mongoClient.getDatabase("patients");
        patients = db.getCollection("patients");
    }

    @Override
    public ContextEnrichedEvent map(CanonicalEvent event) throws Exception {
        // Query patient from MongoDB
        Document query = new Document("patientId", event.getPatientId());
        Document patientDoc = patients.find(query).first();

        PatientInfo patient = parsePatientDocument(patientDoc);

        return ContextEnrichedEvent.builder()
            .fromCanonicalEvent(event)
            .patient(patient)
            .build();
    }

    @Override
    public void close() {
        if (mongoClient != null) {
            mongoClient.close();
        }
    }
}
```

**Dependency** (add to pom.xml):
```xml
<dependency>
    <groupId>org.mongodb</groupId>
    <artifactId>mongodb-driver-sync</artifactId>
    <version>4.10.2</version>
</dependency>
```

---

### Option 3: Async Lookups with Flink AsyncDataStream (Best for Production)

**Advantages**:
- ✅ Non-blocking I/O
- ✅ High throughput
- ✅ Handles slow APIs gracefully
- ✅ Production-ready

**Implementation**:
```java
public class AsyncPatientEnricher extends RichAsyncFunction<CanonicalEvent, ContextEnrichedEvent> {
    private transient AsyncHttpClient httpClient;

    @Override
    public void open(Configuration parameters) {
        httpClient = new DefaultAsyncHttpClient();
    }

    @Override
    public void asyncInvoke(CanonicalEvent event, ResultFuture<ContextEnrichedEvent> resultFuture) {
        // Async HTTP call
        String url = "http://localhost:8003/patients/" + event.getPatientId();

        httpClient.prepareGet(url).execute().toCompletableFuture()
            .thenApply(response -> parsePatientResponse(response.getResponseBody()))
            .thenApply(patient -> ContextEnrichedEvent.builder()
                .fromCanonicalEvent(event)
                .patient(patient)
                .build())
            .whenComplete((enriched, error) -> {
                if (error != null) {
                    // Handle error - maybe set patient to null or retry
                    resultFuture.complete(Collections.singleton(
                        ContextEnrichedEvent.builder()
                            .fromCanonicalEvent(event)
                            .build()));
                } else {
                    resultFuture.complete(Collections.singleton(enriched));
                }
            });
    }
}

// Usage in pipeline
DataStream<ContextEnrichedEvent> enriched = AsyncDataStream.unorderedWait(
    canonicalEvents,
    new AsyncPatientEnricher(),
    10000,  // 10 second timeout
    TimeUnit.MILLISECONDS,
    100     // Max concurrent requests
);
```

---

### Option 4: Caching with Redis (Hybrid Approach)

**Advantages**:
- ✅ Reduces database load
- ✅ Faster lookups for repeated patients
- ✅ Cost-effective for high volume

**Implementation**:
```java
public class CachedPatientEnricher extends RichMapFunction<CanonicalEvent, ContextEnrichedEvent> {
    private transient JedisPool jedisPool;
    private transient AsyncHttpClient httpClient;

    @Override
    public void open(Configuration parameters) {
        // Redis cache
        jedisPool = new JedisPool("localhost", 6379);
        // Fallback API
        httpClient = new DefaultAsyncHttpClient();
    }

    @Override
    public ContextEnrichedEvent map(CanonicalEvent event) throws Exception {
        String patientId = event.getPatientId();
        PatientInfo patient = null;

        // Try cache first
        try (Jedis jedis = jedisPool.getResource()) {
            String cached = jedis.get("patient:" + patientId);
            if (cached != null) {
                patient = parsePatientJson(cached);
            }
        }

        // Cache miss - fetch from API
        if (patient == null) {
            String url = "http://localhost:8003/patients/" + patientId;
            Response response = httpClient.prepareGet(url).execute().get();
            patient = parsePatientResponse(response.getResponseBody());

            // Store in cache (TTL: 1 hour)
            try (Jedis jedis = jedisPool.getResource()) {
                jedis.setex("patient:" + patientId, 3600, toJson(patient));
            }
        }

        return ContextEnrichedEvent.builder()
            .fromCanonicalEvent(event)
            .patient(patient)
            .build();
    }
}
```

---

## Recommended Setup for Your System

### Step 1: Choose Your Data Source Strategy

**For Testing/Development**:
- Use **Option 1** (REST API calls to patient-service)
- Easy to set up, uses existing infrastructure

**For Production**:
- Use **Option 3** (Async lookups) + **Option 4** (Redis caching)
- High performance, scalable

### Step 2: Ensure Patient Service is Running

```bash
# Check if patient service is running
curl http://localhost:8003/health

# If not running, start it
cd /Users/apoorvabk/Downloads/cardiofit/backend/services/patient-service
python run_service.py
```

### Step 3: Add Test Patient Data

**Via API**:
```bash
curl -X POST http://localhost:8003/patients \
  -H "Content-Type: application/json" \
  -d '{
    "patientId": "P12345",
    "firstName": "John",
    "lastName": "Doe",
    "dateOfBirth": "1980-05-15",
    "gender": "male",
    "mrn": "MRN-67890"
  }'
```

**Via MongoDB**:
```javascript
// Connect to MongoDB
use patients

// Insert test patient
db.patients.insertOne({
  "patientId": "P12345",
  "firstName": "John",
  "lastName": "Doe",
  "dateOfBirth": ISODate("1980-05-15"),
  "gender": "male",
  "mrn": "MRN-67890",
  "address": {
    "city": "Boston",
    "state": "MA"
  }
})
```

### Step 4: Test Patient Lookup

```bash
# Test patient API
curl http://localhost:8003/patients/P12345

# Expected response:
# {
#   "patientId": "P12345",
#   "firstName": "John",
#   "lastName": "Doe",
#   ...
# }
```

### Step 5: Configure Module 2

**Create config file** (`Module2Config.java`):
```java
public class Module2Config {
    // Patient service endpoint
    public static final String PATIENT_SERVICE_URL = "http://localhost:8003";

    // Encounter service endpoint
    public static final String ENCOUNTER_SERVICE_URL = "http://localhost:8010";

    // MongoDB connection (if using direct access)
    public static final String MONGODB_URL = "mongodb://localhost:27017";
    public static final String MONGODB_DATABASE = "patients";

    // Redis cache (if using caching)
    public static final String REDIS_HOST = "localhost";
    public static final int REDIS_PORT = 6379;

    // Timeouts
    public static final int API_TIMEOUT_MS = 5000;
    public static final int CACHE_TTL_SECONDS = 3600;
}
```

---

## Data Flow in Module 2

```
1. Event arrives: {patientId: "P12345", ...}
                      ↓
2. Extract patientId: "P12345"
                      ↓
3. Lookup patient data:
   ├─→ Try Redis cache first
   ├─→ If not found, call patient-service API
   └─→ Or query MongoDB directly
                      ↓
4. Get patient info: {firstName: "John", lastName: "Doe", ...}
                      ↓
5. Lookup encounter data:
   └─→ Call encounter-service API with patientId
                      ↓
6. Get encounter info: {encounterId: "ENC-001", department: "ICU", ...}
                      ↓
7. Create enriched event:
   {
     ...original event fields...,
     patient: {...patient data...},
     encounter: {...encounter data...}
   }
                      ↓
8. Write to: context-enriched-events-v1
```

---

## Summary

### Your Data Sources:

| Data Type | Source | Location | Port |
|-----------|--------|----------|------|
| Patient Demographics | patient-service | `/backend/services/patient-service/` | 8003 |
| Encounter/Visit | encounter-service | `/backend/services/encounter-service/` | TBD |
| Facility/Org | organization-service | `/backend/services/organization-service/` | TBD |
| Direct DB Access | MongoDB | `localhost:27017` or Atlas | 27017 |
| FHIR Resources | Google Healthcare API | Cloud | HTTPS |

### Recommended Approach:

1. ✅ **Start patient-service** (port 8003)
2. ✅ **Add test patient data** (P12345, P999, etc.)
3. ✅ **Test API** (`curl http://localhost:8003/patients/P12345`)
4. ✅ **Configure Module 2** to call patient-service API
5. ✅ **Add async lookups** for production performance
6. ✅ **Add Redis caching** for high-volume scenarios

### Next Steps:

1. Check if patient-service is running
2. Add test patient data
3. Implement Module 2 with REST API calls
4. Test Module 2 alone
5. Test Modules 1+2 together

**The data is already in your existing services - Module 2 just needs to query them!**
