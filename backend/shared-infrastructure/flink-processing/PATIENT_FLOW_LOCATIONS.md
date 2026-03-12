# Patient Snapshot & Demographics Flow - Code Locations

## Complete Flow Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│ EVENT ARRIVES: Patient P-12345, First Time (no state exists)       │
└─────────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────────┐
│ 📍 MODULE2_CONTEXTASSEMBLY.JAVA : LINE 244                          │
│                                                                       │
│   PatientSnapshot snapshot = patientSnapshotState.value();          │
│   // Returns NULL for first-time patient                            │
└─────────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────────┐
│ 📍 MODULE2_CONTEXTASSEMBLY.JAVA : LINE 246-252                      │
│                                                                       │
│   if (snapshot == null) {                                           │
│       LOG.info("First-time patient detected");                      │
│       snapshot = handleFirstTimePatient(patientId, event); // ⬅️ 1  │
│       patientSnapshotState.update(snapshot);               // ⬅️ 2  │
│   }                                                                  │
└─────────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────────┐
│ 📍 MODULE2_CONTEXTASSEMBLY.JAVA : LINE 304-369                      │
│ handleFirstTimePatient() - CREATES NEW SNAPSHOT                     │
│                                                                       │
│   1. Makes async FHIR API calls:                                    │
│      - fhirClient.getPatientAsync(patientId)        // Line 311     │
│      - fhirClient.getConditionsAsync(patientId)     // Line 314     │
│      - fhirClient.getMedicationsAsync(patientId)    // Line 317     │
│      - neo4jClient.queryGraphAsync(patientId)       // Line 320     │
│                                                                       │
│   2. Wait 500ms for results (Line 326-327)                          │
│                                                                       │
│   3. Decision Point (Line 336-346):                                 │
│      ┌──────────────────────────────────────────────────┐           │
│      │ If fhirPatient == null (404 from FHIR)          │           │
│      │   ↓                                              │           │
│      │ PatientSnapshot.createEmpty(patientId)  ⬅️ 3    │           │
│      │   → isNewPatient = TRUE                         │           │
│      │   → No demographics, empty lists                │           │
│      └──────────────────────────────────────────────────┘           │
│      ┌──────────────────────────────────────────────────┐           │
│      │ If fhirPatient != null (found in FHIR)          │           │
│      │   ↓                                              │           │
│      │ PatientSnapshot.hydrateFromHistory() ⬅️ 4       │           │
│      │   → isNewPatient = FALSE                        │           │
│      │   → Demographics populated from FHIR            │           │
│      │   → Medications, conditions loaded              │           │
│      └──────────────────────────────────────────────────┘           │
└─────────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────────┐
│ 📍 PATIENTSNAPSHOT.JAVA : LINE 160-171                              │
│ createEmpty() - FOR NEW PATIENTS                                    │
│                                                                       │
│   PatientSnapshot snapshot = new PatientSnapshot(patientId);        │
│   snapshot.isNewPatient = true;              // ⬅️ MARKED AS NEW    │
│   snapshot.firstName = null;                 // No demographics yet │
│   snapshot.lastName = null;                                         │
│   snapshot.age = null;                                              │
│   snapshot.gender = null;                                           │
│   snapshot.activeConditions = new ArrayList<>();  // Empty          │
│   snapshot.activeMedications = new ArrayList<>(); // Empty          │
│   return snapshot;                                                  │
└─────────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────────┐
│ 📍 PATIENTSNAPSHOT.JAVA : LINE 186-226                              │
│ hydrateFromHistory() - FOR EXISTING PATIENTS                        │
│                                                                       │
│   PatientSnapshot snapshot = new PatientSnapshot(patientId);        │
│   snapshot.isNewPatient = false;                                    │
│                                                                       │
│   // DEMOGRAPHICS ADDED HERE ⬅️ 5                                   │
│   if (fhirPatient != null) {                                        │
│       snapshot.firstName = fhirPatient.getFirstName();    // Line 198│
│       snapshot.lastName = fhirPatient.getLastName();      // Line 199│
│       snapshot.dateOfBirth = fhirPatient.getDateOfBirth();// Line 200│
│       snapshot.gender = fhirPatient.getGender();          // Line 201│
│       snapshot.age = fhirPatient.getAge();                // Line 202│
│       snapshot.mrn = fhirPatient.getMrn();                // Line 203│
│       snapshot.allergies = fhirPatient.getAllergies();    // Line 204│
│   }                                                                  │
│                                                                       │
│   // Clinical data from FHIR                                        │
│   snapshot.activeConditions = new ArrayList<>(conditions);          │
│   snapshot.activeMedications = new ArrayList<>(medications);        │
│                                                                       │
│   // Graph data from Neo4j                                          │
│   snapshot.careTeam = graphData.getCareTeam();                      │
│   snapshot.riskCohorts = graphData.getRiskCohorts();                │
│                                                                       │
│   return snapshot;                                                  │
└─────────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────────┐
│ 📍 MODULE2_CONTEXTASSEMBLY.JAVA : LINE 249                          │
│ SNAPSHOT STORED IN FLINK STATE ⬅️ 6                                 │
│                                                                       │
│   patientSnapshotState.update(snapshot);                            │
│   // Stored in RocksDB with 7-day TTL                               │
└─────────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────────┐
│ EVENTS PROCESSED: admission → vitals → medications                  │
│                                                                       │
│ 📍 MODULE2_CONTEXTASSEMBLY.JAVA : LINE 256-257                      │
│   snapshot.updateWithEvent(event);     // Progressive enrichment    │
│   patientSnapshotState.update(snapshot); // Update state            │
└─────────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────────┐
│ 📍 MODULE2_CONTEXTASSEMBLY.JAVA : LINE 277-280                      │
│ DISCHARGE EVENT TRIGGERS ENCOUNTER CLOSURE                          │
│                                                                       │
│   if (isEncounterClosure(event)) {                                  │
│       LOG.info("Encounter closure detected");                       │
│       flushStateToExternalSystems(snapshot); // ⬅️ 7                │
│   }                                                                  │
└─────────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────────┐
│ 📍 MODULE2_CONTEXTASSEMBLY.JAVA : LINE 475-509                      │
│ flushStateToExternalSystems() - WRITES TO FHIR & NEO4J             │
│                                                                       │
│   // WRITE TO FHIR STORE ⬅️ 8                                       │
│   fhirClient.flushSnapshot(snapshot)                                │
│       .thenAccept(result -> LOG.info("Success"))                    │
│                                                                       │
│   // WRITE TO NEO4J ⬅️ 9                                            │
│   neo4jClient.updateCareNetwork(snapshot);                          │
└─────────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────────┐
│ 📍 GOOGLEFHIRCLIENT.JAVA : LINE 265-290                             │
│ flushSnapshot() - CREATE FHIR BUNDLE                                │
│                                                                       │
│   // Build FHIR transaction bundle                                  │
│   String bundleJson = buildFHIRBundle(snapshot); // Line 270        │
│                                                                       │
│   // Submit to Google Healthcare API                                │
│   return executePostRequest(baseUrl, bundleJson); // Line 273       │
└─────────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────────┐
│ 📍 GOOGLEFHIRCLIENT.JAVA : LINE 305-341                             │
│ buildFHIRBundle() - CREATES TRANSACTION BUNDLE                      │
│                                                                       │
│   List<String> entries = new ArrayList<>();                         │
│                                                                       │
│   // 1. PATIENT RESOURCE (IF NEW) ⬅️ 10                             │
│   if (snapshot.isNewPatient()) {                // Line 315         │
│       entries.add(buildPatientEntry(snapshot)); // Line 316         │
│   }                                                                  │
│                                                                       │
│   // 2. Condition resources                                         │
│   for (Condition c : snapshot.getActiveConditions()) {              │
│       entries.add(buildConditionEntry(patientId, c));               │
│   }                                                                  │
│                                                                       │
│   // 3. MedicationRequest resources                                 │
│   for (Medication m : snapshot.getActiveMedications()) {            │
│       entries.add(buildMedicationEntry(patientId, m));              │
│   }                                                                  │
│                                                                       │
│   // 4. Observation resources (vitals)                              │
│   List<VitalSign> vitals = snapshot.getVitalsHistory().getRecent(5);│
│   for (VitalSign v : vitals) {                                      │
│       entries.add(buildVitalObservationEntry(patientId, v));        │
│   }                                                                  │
│                                                                       │
│   return bundle JSON;                                               │
└─────────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────────┐
│ 📍 GOOGLEFHIRCLIENT.JAVA : LINE 346-395                             │
│ buildPatientEntry() - CREATES PATIENT RESOURCE WITH DEMOGRAPHICS    │
│                                                                       │
│   {                                                                  │
│     "fullUrl": "urn:uuid:patient-P-12345",                          │
│     "resource": {                                                    │
│       "resourceType": "Patient",                                    │
│       "id": "P-12345",                                              │
│       "name": [{                                     // Line 355-364 │
│         "use": "official",                                          │
│         "given": [snapshot.getFirstName()],  ⬅️ DEMOGRAPHICS USED  │
│         "family": snapshot.getLastName()                            │
│       }],                                                            │
│       "gender": snapshot.getGender(),        ⬅️ Line 368-370        │
│       "birthDate": snapshot.getDateOfBirth(),⬅️ Line 373-375        │
│       "identifier": [{                                              │
│         "system": "urn:cardiofit:mrn",                              │
│         "value": snapshot.getMrn()           ⬅️ Line 378-383        │
│       }],                                                            │
│       "active": true                                                │
│     },                                                               │
│     "request": {                                                     │
│       "method": "POST",                      ⬅️ CREATE NEW PATIENT  │
│       "url": "Patient"                                              │
│     }                                                                │
│   }                                                                  │
└─────────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────────┐
│ 📍 GOOGLEFHIRCLIENT.JAVA : LINE 554-607                             │
│ executePostRequest() - SUBMIT BUNDLE TO FHIR API ⬅️ 11              │
│                                                                       │
│   POST https://healthcare.googleapis.com/v1/projects/cardiofit/...  │
│   Headers:                                                           │
│     - Authorization: Bearer {OAuth2 token}                          │
│     - Content-Type: application/fhir+json                           │
│   Body: FHIR Bundle JSON                                            │
│                                                                       │
│   Response: HTTP 201 Created                                        │
│   → Patient now exists in Google Cloud Healthcare FHIR Store!      │
└─────────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────────┐
│ 📍 NEO4JGRAPHCLIENT.JAVA : updateCareNetwork() ⬅️ 12                │
│                                                                       │
│   MERGE (p:Patient {id: 'P-12345'})                                │
│   SET p.lastEncounter = timestamp()                                 │
│                                                                       │
│   → Patient node created/updated in Neo4j graph database           │
└─────────────────────────────────────────────────────────────────────┘
```

## Summary of Key Locations

### 🆕 WHERE NEW PATIENT SNAPSHOT IS CREATED

| #  | Location | Line | Description |
|----|----------|------|-------------|
| 1️⃣ | `Module2_ContextAssembly.java` | 248 | `handleFirstTimePatient()` called |
| 2️⃣ | `Module2_ContextAssembly.java` | 249 | `patientSnapshotState.update(snapshot)` |
| 3️⃣ | `PatientSnapshot.java` | 160-171 | `createEmpty()` - New patient, no demographics |
| 4️⃣ | `PatientSnapshot.java` | 186-226 | `hydrateFromHistory()` - Existing patient |

### 📋 WHERE DEMOGRAPHICS ARE ADDED

| #  | Location | Line | Description |
|----|----------|------|-------------|
| 5️⃣ | `PatientSnapshot.java` | 197-205 | **Demographics populated from FHIR API** |
|    |  | 198 | `firstName = fhirPatient.getFirstName()` |
|    |  | 199 | `lastName = fhirPatient.getLastName()` |
|    |  | 200 | `dateOfBirth = fhirPatient.getDateOfBirth()` |
|    |  | 201 | `gender = fhirPatient.getGender()` |
|    |  | 202 | `age = fhirPatient.getAge()` |
|    |  | 203 | `mrn = fhirPatient.getMrn()` |

### 💾 WHERE SNAPSHOT IS STORED IN FLINK STATE

| #  | Location | Line | Description |
|----|----------|------|-------------|
| 6️⃣ | `Module2_ContextAssembly.java` | 249 | `patientSnapshotState.update(snapshot)` (first time) |
|    | `Module2_ContextAssembly.java` | 257 | `patientSnapshotState.update(snapshot)` (updates) |

### 🏥 WHERE PATIENT IS WRITTEN TO FHIR STORE

| #  | Location | Line | Description |
|----|----------|------|-------------|
| 7️⃣ | `Module2_ContextAssembly.java` | 279 | `flushStateToExternalSystems()` called on discharge |
| 8️⃣ | `Module2_ContextAssembly.java` | 480 | `fhirClient.flushSnapshot(snapshot)` |
| 🔟 | `GoogleFHIRClient.java` | 315-316 | **Patient resource added to bundle** |
| 1️⃣1️⃣ | `GoogleFHIRClient.java` | 554-607 | **POST bundle to Google Healthcare API** |

### 🕸️ WHERE PATIENT IS WRITTEN TO NEO4J

| #  | Location | Line | Description |
|----|----------|------|-------------|
| 9️⃣ | `Module2_ContextAssembly.java` | 492-500 | `neo4jClient.updateCareNetwork(snapshot)` |
| 1️⃣2️⃣ | `Neo4jGraphClient.java` | - | `MERGE (p:Patient {id: ...})` Cypher query |

## Data Flow States

### NEW PATIENT (404 from FHIR)
```
Event → Flink State (empty snapshot) → Progressive enrichment → Discharge → FHIR Store (CREATE) + Neo4j (MERGE)
```

### EXISTING PATIENT (found in FHIR)
```
Event → FHIR API lookup → Flink State (hydrated snapshot) → Progressive enrichment → Discharge → FHIR Store (UPDATE) + Neo4j (MERGE)
```

## Key Insight

**For NEW patients**: Demographics start as `null` in the empty snapshot (line 160-171), but the snapshot still gets written to FHIR on encounter closure. The FHIR Patient resource is created with whatever demographic data is available at that time (could be minimal or complete depending on what events were received).

**For EXISTING patients**: Demographics are loaded from FHIR during `hydrateFromHistory()` (lines 197-205) when the patient is first encountered in the stream.
