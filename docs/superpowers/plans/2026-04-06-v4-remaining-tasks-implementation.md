# V4 Remaining Tasks — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement 11 remaining V4 North Star tasks in strict sequential order (C0→C2→C3→C4→C5→C6), completing the dual-domain clinical intelligence platform.

**Architecture:** Go KB services (KB-20, KB-23, KB-26) extended with V4 state fields, IOR persistence, dual-domain cards, and feedback capture. Java Flink operators extended with domain classification and cross-domain CEP patterns. Python batch pipeline for phenotype clustering. YAML market configs loaded at service startup.

**Tech Stack:** Go 1.21 (Gin, GORM), Java 17 (Flink 1.18), Python 3.11 (UMAP, HDBSCAN), PostgreSQL 15, Kafka, YAML

**Spec:** `docs/superpowers/specs/2026-04-05-v4-remaining-tasks-design.md`

---

## Phase C0: Foundation Wiring

### Task 3b: Flink-Side V4 Field Propagation

**Files:**
- Modify: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module1b_IngestionCanonicalizer.java:209-241`
- Modify: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/OutboxEnvelope.java` (add V4 field getters)
- Test: `backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module1bV4FieldTest.java` (new)

**Context:** The ingestion service emits 13 V4 fields in `CanonicalObservation` JSON. Module 1b extracts fields into a `Map<String, Object> payload` (line 209). Currently it extracts `loinc_code`, `value`, `unit`, `observation_type`, `quality_score`, `source_type`, `source_id`, `fhir_resource_id`, `flags`, `data_tier`, `cgm_active`. The remaining 12 V4 fields (`linked_meal_id`, `sodium_estimated_mg`, etc.) are present in the upstream JSON but not extracted because `IngestionEventData` lacks the corresponding `@JsonProperty` fields.

- [ ] **Step 1: Write the failing test for V4 field extraction**

Create `Module1bV4FieldTest.java`:

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.CanonicalEvent;
import com.cardiofit.flink.models.OutboxEnvelope;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.junit.jupiter.api.Test;
import java.util.Map;
import static org.junit.jupiter.api.Assertions.*;

public class Module1bV4FieldTest {

    private static final ObjectMapper MAPPER = new ObjectMapper();

    @Test
    void testV4MealFieldsExtracted() throws Exception {
        String json = """
            {
              "eventId": "evt-001", "patientId": "P-100",
              "observationType": "MEAL", "timestamp": "2026-04-06T08:00:00Z",
              "loincCode": "LP32886-4", "value": "320", "unit": "kcal",
              "sourceType": "PATIENT_REPORTED",
              "linked_meal_id": "meal-abc-123",
              "sodium_estimated_mg": 850.5,
              "preparation_method": "FRIED",
              "food_name_local": "Poha",
              "source_protocol": "MEAL_PHOTO_AI"
            }
            """;
        OutboxEnvelope envelope = buildEnvelope(json);
        CanonicalEvent result = processThrough1b(envelope);
        Map<String, Object> payload = result.getPayload();

        assertEquals("meal-abc-123", payload.get("linked_meal_id"));
        assertEquals(850.5, ((Number) payload.get("sodium_estimated_mg")).doubleValue(), 0.01);
        assertEquals("FRIED", payload.get("preparation_method"));
        assertEquals("Poha", payload.get("food_name_local"));
        assertEquals("MEAL_PHOTO_AI", payload.get("source_protocol"));
    }

    @Test
    void testV4BPFieldsExtracted() throws Exception {
        String json = """
            {
              "eventId": "evt-002", "patientId": "P-100",
              "observationType": "VITALS", "timestamp": "2026-04-06T07:00:00Z",
              "loincCode": "85354-9", "value": "138", "unit": "mmHg",
              "sourceType": "DEVICE",
              "bp_device_type": "OSCILLOMETRIC_HOME",
              "clinical_grade": true,
              "measurement_method": "SEATED_REST_5MIN",
              "linked_seated_reading_id": "read-xyz-789",
              "waking_time": "06:30",
              "sleep_time": "22:30"
            }
            """;
        OutboxEnvelope envelope = buildEnvelope(json);
        CanonicalEvent result = processThrough1b(envelope);
        Map<String, Object> payload = result.getPayload();

        assertEquals("OSCILLOMETRIC_HOME", payload.get("bp_device_type"));
        assertEquals(true, payload.get("clinical_grade"));
        assertEquals("SEATED_REST_5MIN", payload.get("measurement_method"));
        assertEquals("read-xyz-789", payload.get("linked_seated_reading_id"));
        assertEquals("06:30", payload.get("waking_time"));
        assertEquals("22:30", payload.get("sleep_time"));
    }

    @Test
    void testV4FieldsMissingGracefullyIgnored() throws Exception {
        // V3 event without V4 fields — must not NPE
        String json = """
            {
              "eventId": "evt-003", "patientId": "P-100",
              "observationType": "VITALS", "timestamp": "2026-04-06T07:00:00Z",
              "loincCode": "85354-9", "value": "120", "unit": "mmHg",
              "sourceType": "DEVICE"
            }
            """;
        OutboxEnvelope envelope = buildEnvelope(json);
        CanonicalEvent result = processThrough1b(envelope);
        Map<String, Object> payload = result.getPayload();

        assertNull(payload.get("linked_meal_id"));
        assertNull(payload.get("bp_device_type"));
        assertNotNull(payload.get("loinc_code")); // core field still present
    }

    @Test
    void testSymptomAwarenessFieldExtracted() throws Exception {
        String json = """
            {
              "eventId": "evt-004", "patientId": "P-100",
              "observationType": "PATIENT_REPORTED",
              "timestamp": "2026-04-06T09:00:00Z",
              "loincCode": "LP200964-0", "value": "1", "unit": "boolean",
              "sourceType": "PATIENT_REPORTED",
              "symptom_awareness": false
            }
            """;
        OutboxEnvelope envelope = buildEnvelope(json);
        CanonicalEvent result = processThrough1b(envelope);
        assertEquals(false, result.getPayload().get("symptom_awareness"));
    }

    private OutboxEnvelope buildEnvelope(String eventDataJson) throws Exception {
        OutboxEnvelope env = new OutboxEnvelope();
        env.setCorrelationId("corr-" + System.nanoTime());
        OutboxEnvelope.IngestionEventData data =
            MAPPER.readValue(eventDataJson, OutboxEnvelope.IngestionEventData.class);
        env.setEventData(data);
        return env;
    }

    private CanonicalEvent processThrough1b(OutboxEnvelope envelope) {
        Module1b_IngestionCanonicalizer.OutboxValidationAndCanonicalization processor =
            new Module1b_IngestionCanonicalizer.OutboxValidationAndCanonicalization();
        return processor.canonicalize(envelope);
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend/shared-infrastructure/flink-processing
mvn test -pl . -Dtest=Module1bV4FieldTest -Dsurefire.failIfNoTests=false
```

Expected: FAIL — V4 fields not extracted into payload.

- [ ] **Step 3: Add V4 field declarations to OutboxEnvelope.IngestionEventData**

In `OutboxEnvelope.java`, inside the `IngestionEventData` inner class, add these fields with Jackson annotations:

```java
    // V4: Meal correlation fields (Module 10/10b consumers)
    @JsonProperty("linked_meal_id")       private String linkedMealId;
    @JsonProperty("sodium_estimated_mg")  private Double sodiumEstimatedMg;
    @JsonProperty("preparation_method")   private String preparationMethod;
    @JsonProperty("food_name_local")      private String foodNameLocal;
    @JsonProperty("source_protocol")      private String sourceProtocol;

    // V4: BP measurement context (Module 7 consumer)
    @JsonProperty("bp_device_type")            private String bpDeviceType;
    @JsonProperty("clinical_grade")            private Boolean clinicalGrade;
    @JsonProperty("measurement_method")        private String measurementMethod;
    @JsonProperty("linked_seated_reading_id")  private String linkedSeatedReadingId;
    @JsonProperty("waking_time")               private String wakingTime;
    @JsonProperty("sleep_time")                private String sleepTime;

    // V4: CID masking detection (Module 8 consumer)
    @JsonProperty("symptom_awareness")         private Boolean symptomAwareness;
```

Add corresponding getter methods (Lombok `@Data` may auto-generate; if not, add manually).

- [ ] **Step 4: Add V4 field extraction to Module 1b payload construction**

In `Module1b_IngestionCanonicalizer.java`, after line 241 (after the data tier classification block), add:

```java
        // V4: Meal correlation fields (Module 10/10b consumers)
        putIfNotNull(payload, "linked_meal_id", data.getLinkedMealId());
        putIfNotNull(payload, "sodium_estimated_mg", data.getSodiumEstimatedMg());
        putIfNotNull(payload, "preparation_method", data.getPreparationMethod());
        putIfNotNull(payload, "food_name_local", data.getFoodNameLocal());
        putIfNotNull(payload, "source_protocol", data.getSourceProtocol());

        // V4: BP measurement context fields (Module 7 consumer)
        putIfNotNull(payload, "bp_device_type", data.getBpDeviceType());
        putIfNotNull(payload, "clinical_grade", data.getClinicalGrade());
        putIfNotNull(payload, "measurement_method", data.getMeasurementMethod());
        putIfNotNull(payload, "linked_seated_reading_id", data.getLinkedSeatedReadingId());
        putIfNotNull(payload, "waking_time", data.getWakingTime());
        putIfNotNull(payload, "sleep_time", data.getSleepTime());

        // V4: CID masking detection field (Module 8 consumer)
        putIfNotNull(payload, "symptom_awareness", data.getSymptomAwareness());
```

Add helper at bottom of class:

```java
    private static void putIfNotNull(Map<String, Object> map, String key, Object value) {
        if (value != null) {
            map.put(key, value);
        }
    }
```

- [ ] **Step 5: Run new + existing Module 1b tests**

```bash
cd backend/shared-infrastructure/flink-processing
mvn test -pl . -Dtest="Module1bV4FieldTest,Module1IngestionRouterTest,Module1DeserializationSafetyTest,Module1ValidationFixTest,Module1bDLQTest"
```

Expected: All tests PASS (4 new + all existing).

- [ ] **Step 6: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module1b_IngestionCanonicalizer.java
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/OutboxEnvelope.java
git add backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module1bV4FieldTest.java
git commit -m "feat(module1b): extract 12 V4 fields from ingestion payload

Adds linked_meal_id, sodium_estimated_mg, bp_device_type, clinical_grade,
measurement_method, waking_time, sleep_time, symptom_awareness, and 4 more
V4 fields to the canonical payload map. Null-safe for V3 events."
```

---

### Task 3c: V3 Contract Wiring Audit + Fix

**Files:**
- Audit: `kb-20-patient-profile/internal/services/` (P12 event publication)
- Audit: `kb-22-hpi-engine/internal/services/` (P14, P15-a, P15-b)
- Modify: only files where audit finds gaps

**Context:** Code review shows P14 (KB-5 migration), P15-a (`/execute` endpoint), P15-b (minimum inclusion guard R-05) are already implemented. P12 needs verification for both LAB_RESULT and MEDICATION_CHANGE event publication.

- [ ] **Step 7: Audit P12 — verify KB-20 publishes LAB_RESULT events**

```bash
cd backend/shared-infrastructure/knowledge-base-services
grep -rn "LAB_RESULT\|LabResult\|lab_result" kb-20-patient-profile/internal/services/ --include="*.go"
```

Check matching files for Kafka event publication after lab writes. If `EventLabResult` is published in `lab_service.go` or `kafka_outbox_relay.go`, P12-labs is done.

- [ ] **Step 8: Audit P12 — verify MEDICATION_CHANGE event publication**

```bash
grep -rn "MEDICATION_CHANGE\|MedicationChange\|medication_change" kb-20-patient-profile/internal/services/ --include="*.go"
```

If no matches, add event publication in `medication_service.go` after successful medication writes:

```go
    // Publish MEDICATION_CHANGE event for downstream (Module 8 CID, Module 13)
    if ms.eventBus != nil {
        ms.eventBus.Publish(models.EventMedicationChange, models.EventPayload{
            PatientID:  patientID,
            EventType:  "MEDICATION_CHANGE",
            DrugClass:  med.DrugClass,
            ChangeType: changeType, // "START", "STOP", "DOSE_ADJUST"
            Timestamp:  time.Now(),
        })
    }
```

- [ ] **Step 9: Audit P14/P15-a/P15-b — verify KB-22 contracts**

```bash
# P14: should find KB-5 not KB-9
grep -rn "kb-9\|KB-9\|:8089\|kb9" kb-22-hpi-engine/ --include="*.go"
grep -rn "kb-5\|KB-5\|:8085" kb-22-hpi-engine/ --include="*.go" | head -5

# P15-a: should find /execute not /events
grep -rn "/events\|/execute" kb-22-hpi-engine/internal/services/outcome_publisher.go

# P15-b: should find minimum inclusion guard
grep -rn "minInclusion\|MinInclusion\|QuestionsAnswered\|R-05" kb-22-hpi-engine/internal/services/session_service.go
```

Expected: P14 → no KB-9 refs, P15-a → `/execute` found, P15-b → guard found.

- [ ] **Step 10: If P12 gap found — implement MEDICATION_CHANGE publication and test**

Only execute if Step 8 found no publication. Write test:

```go
// kb-20-patient-profile/internal/services/contract_audit_test.go
package services

import "testing"

func TestP12_EventTypesRegistered(t *testing.T) {
    validTypes := []string{"LAB_RESULT", "MEDICATION_CHANGE"}
    for _, et := range validTypes {
        if !isValidEventType(et) {
            t.Errorf("event type %s not registered in event bus", et)
        }
    }
}
```

- [ ] **Step 11: Run KB-20 and KB-22 test suites**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile
go test ./... -v -count=1

cd ../kb-22-hpi-engine
go test ./... -v -count=1
```

Expected: All tests PASS.

- [ ] **Step 12: Commit**

```bash
git add -A backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/
git add -A backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine/
git commit -m "audit(contracts): verify V3 wiring P12/P14/P15-a/P15-b

All 4 contract items verified. P14 (KB-5), P15-a (/execute), P15-b
(min inclusion R-05) confirmed. P12 event publication verified for
LAB_RESULT and MEDICATION_CHANGE."
```

---

---

## Phase C2: KB-20 Extensions + Dual-Domain Flink

### Task 6: KB-20 V4 State API Endpoints

**Files:**
- Create: `kb-20-patient-profile/internal/api/v4_state_handlers.go`
- Create: `kb-20-patient-profile/internal/api/v4_state_handlers_test.go`
- Modify: `kb-20-patient-profile/internal/api/routes.go:60-141`

**Context:** `PatientProfile` (in `models/patient_profile.go`) already has all V4 fields: `ARVSBP7d`, `DipClassification`, `EngagementComposite`, `PhenotypeCluster`, `MHRIScore`, `CKMStage`, etc. (lines 41-83). `ComputeCKMStage()` exists in `models/ckm_stage.go`. What's missing: API endpoints to read V4 state and a PATCH endpoint for Module 13 sink writes. The routes file (`routes.go`) uses Gin with a `/api/v1/patient/:id/` prefix and `resolveFHIRPatientID()` middleware.

- [ ] **Step 13: Write failing test for GET /patient/:id/v4-state**

```go
// kb-20-patient-profile/internal/api/v4_state_handlers_test.go
package api

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestGetV4State_ReturnsAllFields(t *testing.T) {
    srv := setupTestServer(t)
    patientID := createTestPatient(t, srv, map[string]interface{}{
        "patient_id": "V4-TEST-001",
        "age": 55, "sex": "M", "dm_type": "T2DM",
        "ckm_stage": 2,
        "mhri_score": 72.5,
        "engagement_composite": 0.85,
        "arv_sbp_7d": 12.3,
        "dip_classification": "NON_DIPPER",
        "phenotype_cluster": "CLUSTER_3",
    })

    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/api/v1/patient/"+patientID+"/v4-state", nil)
    srv.Router.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)
    var resp map[string]interface{}
    require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
    data := resp["data"].(map[string]interface{})
    assert.Equal(t, float64(2), data["ckm_stage"])
    assert.Equal(t, 72.5, data["mhri_score"])
    assert.Equal(t, 0.85, data["engagement_composite"])
    assert.Equal(t, 12.3, data["arv_sbp_7d"])
    assert.Equal(t, "NON_DIPPER", data["dip_classification"])
    assert.Equal(t, "CLUSTER_3", data["phenotype_cluster"])
}

func TestGetV4State_PatientNotFound(t *testing.T) {
    srv := setupTestServer(t)
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/api/v1/patient/NONEXISTENT/v4-state", nil)
    srv.Router.ServeHTTP(w, req)
    assert.Equal(t, http.StatusNotFound, w.Code)
}
```

- [ ] **Step 14: Run test to verify it fails**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile
go test ./internal/api/ -run TestGetV4State -v
```

Expected: FAIL — route not registered, 404.

- [ ] **Step 15: Implement GET and PATCH v4-state handlers**

```go
// kb-20-patient-profile/internal/api/v4_state_handlers.go
package api

import (
    "net/http"

    "github.com/gin-gonic/gin"

    "kb-patient-profile/internal/models"
)

// V4StateResponse contains all cached V4 state fields.
type V4StateResponse struct {
    PatientID           string   `json:"patient_id"`
    CKMStage            int      `json:"ckm_stage"`
    HasClinicalCVD      bool     `json:"has_clinical_cvd"`
    ASCVDRisk10y        *float64 `json:"ascvd_risk_10y,omitempty"`
    MHRIScore           *float64 `json:"mhri_score,omitempty"`
    MHRITrajectory      string   `json:"mhri_trajectory,omitempty"`
    MHRIDataQuality     string   `json:"mhri_data_quality,omitempty"`
    EngagementComposite *float64 `json:"engagement_composite,omitempty"`
    EngagementStatus    string   `json:"engagement_status,omitempty"`
    ARVSBP7d            *float64 `json:"arv_sbp_7d,omitempty"`
    ARVSBP30d           *float64 `json:"arv_sbp_30d,omitempty"`
    MorningSurge7dAvg   *float64 `json:"morning_surge_7d_avg,omitempty"`
    DipClassification   string   `json:"dip_classification,omitempty"`
    BPControlStatus     string   `json:"bp_control_status,omitempty"`
    PhenotypeCluster    string   `json:"phenotype_cluster,omitempty"`
    PhenotypeConfidence *float64 `json:"phenotype_confidence,omitempty"`
    DataTier            string   `json:"data_tier"`
}

func (s *Server) getV4State(c *gin.Context) {
    patientID := c.Param("id")
    var profile models.PatientProfile
    if err := s.db.DB.Where("patient_id = ? AND active = true", patientID).
        First(&profile).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "patient not found"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": V4StateResponse{
        PatientID: profile.PatientID, CKMStage: profile.CKMStage,
        HasClinicalCVD: profile.HasClinicalCVD, ASCVDRisk10y: profile.ASCVDRisk10y,
        MHRIScore: profile.MHRIScore, MHRITrajectory: profile.MHRITrajectory,
        MHRIDataQuality: profile.MHRIDataQuality,
        EngagementComposite: profile.EngagementComposite,
        EngagementStatus: profile.EngagementStatus,
        ARVSBP7d: profile.ARVSBP7d, ARVSBP30d: profile.ARVSBP30d,
        MorningSurge7dAvg: profile.MorningSurge7dAvg,
        DipClassification: profile.DipClassification,
        BPControlStatus: profile.BPControlStatus,
        PhenotypeCluster: profile.PhenotypeCluster,
        PhenotypeConfidence: profile.PhenotypeConfidence,
        DataTier: profile.DataTier,
    }})
}

// V4StatePatchRequest for Module 13 sink writes.
type V4StatePatchRequest struct {
    CKMStage            *int     `json:"ckm_stage,omitempty"`
    MHRIScore           *float64 `json:"mhri_score,omitempty"`
    MHRITrajectory      *string  `json:"mhri_trajectory,omitempty"`
    MHRIDataQuality     *string  `json:"mhri_data_quality,omitempty"`
    EngagementComposite *float64 `json:"engagement_composite,omitempty"`
    EngagementStatus    *string  `json:"engagement_status,omitempty"`
    ARVSBP7d            *float64 `json:"arv_sbp_7d,omitempty"`
    ARVSBP30d           *float64 `json:"arv_sbp_30d,omitempty"`
    MorningSurge7dAvg   *float64 `json:"morning_surge_7d_avg,omitempty"`
    DipClassification   *string  `json:"dip_classification,omitempty"`
    BPControlStatus     *string  `json:"bp_control_status,omitempty"`
    PhenotypeCluster    *string  `json:"phenotype_cluster,omitempty"`
    PhenotypeConfidence *float64 `json:"phenotype_confidence,omitempty"`
}

func (s *Server) patchV4State(c *gin.Context) {
    patientID := c.Param("id")
    var req V4StatePatchRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    if req.CKMStage != nil && (*req.CKMStage < 0 || *req.CKMStage > 4) {
        c.JSON(http.StatusBadRequest, gin.H{"error": "ckm_stage must be 0-4"})
        return
    }
    var profile models.PatientProfile
    if err := s.db.DB.Where("patient_id = ? AND active = true", patientID).
        First(&profile).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "patient not found"})
        return
    }
    updates := map[string]interface{}{}
    if req.CKMStage != nil            { updates["ckm_stage"] = *req.CKMStage }
    if req.MHRIScore != nil           { updates["mhri_score"] = *req.MHRIScore }
    if req.MHRITrajectory != nil      { updates["mhri_trajectory"] = *req.MHRITrajectory }
    if req.MHRIDataQuality != nil     { updates["mhri_data_quality"] = *req.MHRIDataQuality }
    if req.EngagementComposite != nil { updates["engagement_composite"] = *req.EngagementComposite }
    if req.EngagementStatus != nil    { updates["engagement_status"] = *req.EngagementStatus }
    if req.ARVSBP7d != nil            { updates["arv_sbp_7d"] = *req.ARVSBP7d }
    if req.ARVSBP30d != nil           { updates["arv_sbp_30d"] = *req.ARVSBP30d }
    if req.MorningSurge7dAvg != nil   { updates["morning_surge_7d_avg"] = *req.MorningSurge7dAvg }
    if req.DipClassification != nil   { updates["dip_classification"] = *req.DipClassification }
    if req.BPControlStatus != nil     { updates["bp_control_status"] = *req.BPControlStatus }
    if req.PhenotypeCluster != nil    { updates["phenotype_cluster"] = *req.PhenotypeCluster }
    if req.PhenotypeConfidence != nil { updates["phenotype_confidence"] = *req.PhenotypeConfidence }
    if len(updates) == 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
        return
    }
    if err := s.db.DB.Model(&profile).Updates(updates).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"status": "updated", "fields_updated": len(updates)})
}
```

- [ ] **Step 16: Register V4 state routes**

In `routes.go`, inside the `patient.Use(s.resolveFHIRPatientID())` block, after the engagement-season route (line 95), add:

```go
            // V4 state cache (Module 13 sink writes + dashboard reads)
            patient.GET("/:id/v4-state", s.getV4State)
            patient.PATCH("/:id/v4-state", s.patchV4State)
```

- [ ] **Step 17: Write PATCH test and run all v4-state tests**

Add to `v4_state_handlers_test.go`:

```go
func TestPatchV4State_UpdatesMHRI(t *testing.T) {
    srv := setupTestServer(t)
    patientID := createTestPatient(t, srv, map[string]interface{}{
        "patient_id": "V4-PATCH-001", "age": 50, "sex": "F", "dm_type": "T2DM",
    })
    body := `{"mhri_score": 68.2, "mhri_trajectory": "DECLINING", "ckm_stage": 3}`
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("PATCH", "/api/v1/patient/"+patientID+"/v4-state",
        strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    srv.Router.ServeHTTP(w, req)
    assert.Equal(t, http.StatusOK, w.Code)

    // Read back
    w2 := httptest.NewRecorder()
    req2, _ := http.NewRequest("GET", "/api/v1/patient/"+patientID+"/v4-state", nil)
    srv.Router.ServeHTTP(w2, req2)
    var resp map[string]interface{}
    require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp))
    data := resp["data"].(map[string]interface{})
    assert.Equal(t, 68.2, data["mhri_score"])
    assert.Equal(t, float64(3), data["ckm_stage"])
}

func TestPatchV4State_RejectsInvalidCKM(t *testing.T) {
    srv := setupTestServer(t)
    patientID := createTestPatient(t, srv, map[string]interface{}{
        "patient_id": "V4-PATCH-002", "age": 50, "sex": "F", "dm_type": "T2DM",
    })
    body := `{"ckm_stage": 7}`
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("PATCH", "/api/v1/patient/"+patientID+"/v4-state",
        strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    srv.Router.ServeHTTP(w, req)
    assert.Equal(t, http.StatusBadRequest, w.Code)
}
```

Run:

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile
go test ./internal/api/ -run "TestGetV4State|TestPatchV4State" -v
```

Expected: All 4 tests PASS.

- [ ] **Step 18: Run full KB-20 test suite**

```bash
go test ./... -v -count=1
```

Expected: All tests PASS.

- [ ] **Step 19: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/api/v4_state_handlers.go
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/api/v4_state_handlers_test.go
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/api/routes.go
git commit -m "feat(kb20): add GET/PATCH /v4-state endpoints for Module 13 sink

GET returns all cached V4 fields (CKM, MHRI, engagement, BP variability,
phenotype). PATCH accepts partial updates with CKM stage 0-4 validation."
```

---

### Task 6b: V3 Flink Dual-Domain Extensions

**Files:**
- Modify: `flink-processing/.../operators/Module1b_IngestionCanonicalizer.java`
- Modify: `flink-processing/.../operators/Module3_ComprehensiveCDS.java`
- Create: `flink-processing/.../operators/Module1bDomainClassificationTest.java`
- Create: `flink-processing/.../operators/Module3BPTrajectoryTest.java`

**Context:** `SemanticEvent` already has `clinicalDomain` (line 99) and `trajectoryClass` (line 102) with builders and getters. Module 4 already has a cross-domain deterioration CEP pattern (lines 845-869) using `trajectoryClass`. Gaps: (1) Module 1b doesn't set `clinicalDomain` on output, (2) Module 3 doesn't compute BP trajectory slope.

- [ ] **Step 20: Write failing test for domain classification**

```java
// Module1bDomainClassificationTest.java
package com.cardiofit.flink.operators;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.CsvSource;
import static org.junit.jupiter.api.Assertions.*;

public class Module1bDomainClassificationTest {

    @ParameterizedTest
    @CsvSource({
        "FBG,        GLYCAEMIC",
        "PPBG,       GLYCAEMIC",
        "HBA1C,      GLYCAEMIC",
        "CGM,        GLYCAEMIC",
        "BP_SEATED,  HEMODYNAMIC",
        "BP_STANDING,HEMODYNAMIC",
        "BP_MORNING, HEMODYNAMIC",
        "VITALS,     HEMODYNAMIC",
        "CREATININE, RENAL",
        "ACR,        RENAL",
        "EGFR,       RENAL",
        "WEIGHT,     METABOLIC",
        "WAIST,      METABOLIC",
        "LIPIDS,     METABOLIC",
        "LDL,        METABOLIC"
    })
    void testDomainClassification(String obsType, String expectedDomain) {
        assertEquals(expectedDomain,
            Module1b_IngestionCanonicalizer.classifyClinicalDomain(obsType));
    }

    @Test
    void testUnknownReturnsGeneral() {
        assertEquals("GENERAL",
            Module1b_IngestionCanonicalizer.classifyClinicalDomain("UNKNOWN_TYPE"));
    }

    @Test
    void testNullReturnsGeneral() {
        assertEquals("GENERAL",
            Module1b_IngestionCanonicalizer.classifyClinicalDomain(null));
    }
}
```

- [ ] **Step 21: Run test to verify it fails**

```bash
cd backend/shared-infrastructure/flink-processing
mvn test -pl . -Dtest=Module1bDomainClassificationTest
```

Expected: FAIL — method does not exist.

- [ ] **Step 22: Implement domain classification in Module 1b**

Add to `Module1b_IngestionCanonicalizer.java`:

```java
    /**
     * Classifies observation type into clinical domain for dual-domain routing.
     */
    public static String classifyClinicalDomain(String observationType) {
        if (observationType == null) return "GENERAL";
        String upper = observationType.toUpperCase();
        if (upper.contains("FBG") || upper.contains("PPBG") || upper.contains("HBA1C")
            || upper.contains("CGM") || upper.contains("GLUCOSE")) return "GLYCAEMIC";
        if (upper.contains("BP") || upper.contains("VITAL") || upper.contains("HEART_RATE")
            || upper.contains("PULSE")) return "HEMODYNAMIC";
        if (upper.contains("CREATININE") || upper.contains("ACR") || upper.contains("EGFR")
            || upper.contains("BUN") || upper.contains("UREA")) return "RENAL";
        if (upper.contains("WEIGHT") || upper.contains("WAIST") || upper.contains("LIPID")
            || upper.contains("LDL") || upper.contains("HDL") || upper.contains("TRIGLYCERIDE")
            || upper.contains("CHOLESTEROL") || upper.contains("BMI")) return "METABOLIC";
        return "GENERAL";
    }
```

In `processElement`, after the data tier block, add:

```java
        payload.put("clinical_domain", classifyClinicalDomain(data.getObservationType()));
```

- [ ] **Step 23: Write failing test for BP trajectory in Module 3**

```java
// Module3BPTrajectoryTest.java
package com.cardiofit.flink.operators;

import org.junit.jupiter.api.Test;
import java.util.*;
import static org.junit.jupiter.api.Assertions.*;

public class Module3BPTrajectoryTest {

    @Test
    void testBPTrajectorySlope_Rising() {
        List<Double> sbp = new ArrayList<>();
        for (int i = 0; i < 14; i++) sbp.add(130.0 + i * (15.0 / 13.0));
        double slope = Module3_ComprehensiveCDS.computeBPTrajectorySlope(sbp);
        assertTrue(slope > 0.5, "Expected positive slope > 0.5, got " + slope);
    }

    @Test
    void testBPTrajectorySlope_Stable() {
        List<Double> sbp = Arrays.asList(
            130.0, 131.0, 129.0, 130.5, 131.5, 129.5, 130.0,
            130.0, 131.0, 129.0, 130.5, 131.5, 129.5, 130.0);
        double slope = Module3_ComprehensiveCDS.computeBPTrajectorySlope(sbp);
        assertTrue(Math.abs(slope) < 0.3, "Expected near-zero, got " + slope);
    }

    @Test
    void testBPTrajectorySlope_Declining() {
        List<Double> sbp = new ArrayList<>();
        for (int i = 0; i < 14; i++) sbp.add(150.0 - i * (20.0 / 13.0));
        double slope = Module3_ComprehensiveCDS.computeBPTrajectorySlope(sbp);
        assertTrue(slope < -0.5, "Expected negative slope, got " + slope);
    }

    @Test
    void testBPTrajectorySlope_InsufficientData() {
        assertEquals(0.0, Module3_ComprehensiveCDS.computeBPTrajectorySlope(
            Arrays.asList(130.0, 132.0)));
    }

    @Test
    void testClassification() {
        assertEquals("RAPID_RISING",   Module3_ComprehensiveCDS.classifyBPTrajectory(1.5));
        assertEquals("RISING",         Module3_ComprehensiveCDS.classifyBPTrajectory(0.8));
        assertEquals("STABLE",         Module3_ComprehensiveCDS.classifyBPTrajectory(0.2));
        assertEquals("DECLINING",      Module3_ComprehensiveCDS.classifyBPTrajectory(-0.8));
        assertEquals("RAPID_DECLINING",Module3_ComprehensiveCDS.classifyBPTrajectory(-1.5));
    }
}
```

- [ ] **Step 24: Implement BP trajectory in Module 3**

Add to `Module3_ComprehensiveCDS.java`:

```java
    /**
     * OLS linear regression for SBP trajectory over a 14-day window.
     * Returns slope in mmHg/day. Requires >= 5 readings.
     */
    public static double computeBPTrajectorySlope(List<Double> sbpReadings) {
        if (sbpReadings == null || sbpReadings.size() < 5) return 0.0;
        int n = sbpReadings.size();
        double sumX = 0, sumY = 0, sumXY = 0, sumX2 = 0;
        for (int i = 0; i < n; i++) {
            sumX += i; sumY += sbpReadings.get(i);
            sumXY += i * sbpReadings.get(i); sumX2 += (double) i * i;
        }
        double denom = n * sumX2 - sumX * sumX;
        if (Math.abs(denom) < 1e-10) return 0.0;
        return (n * sumXY - sumX * sumY) / denom;
    }

    /**
     * Classifies BP trajectory. Thresholds aligned with glucose trajectory.
     */
    public static String classifyBPTrajectory(double slopePerDay) {
        if (slopePerDay > 1.0)   return "RAPID_RISING";
        if (slopePerDay > 0.5)   return "RISING";
        if (slopePerDay >= -0.5) return "STABLE";
        if (slopePerDay >= -1.0) return "DECLINING";
        return "RAPID_DECLINING";
    }
```

- [ ] **Step 25: Run all Flink tests**

```bash
cd backend/shared-infrastructure/flink-processing
mvn test -pl . -Dtest="Module1bDomainClassificationTest,Module3BPTrajectoryTest"
mvn test -pl .  # full regression
```

Expected: All tests PASS.

- [ ] **Step 26: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module1b_IngestionCanonicalizer.java
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module3_ComprehensiveCDS.java
git add backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module1bDomainClassificationTest.java
git add backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module3BPTrajectoryTest.java
git commit -m "feat(flink): clinical domain classification + BP trajectory slope

Module 1b: classifyClinicalDomain() maps obs types to GLYCAEMIC/HEMODYNAMIC/
RENAL/METABOLIC. Module 3: OLS BP trajectory with 5-tier classification."
```

---

---

## Phase C3: Intelligence Core

### Task 7: MHRI 14-Day Trajectory in KB-26

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/services/mri_trajectory.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/mri_trajectory_test.go`
- Modify: `kb-26-metabolic-digital-twin/internal/api/mri_handlers.go` (add trajectory endpoint)
- Modify: `kb-26-metabolic-digital-twin/internal/api/routes.go`

**Context:** `mri_scorer.go` has `ComputeMRI()` (4-domain composite: glucose 35%, body_comp 25%, cardio 25%, behavioral 15%), `PersistScore()`, `ScaleToRange()` (sigmoid 0-100), `CategorizeMRI()` (OPTIMAL/MILD/MODERATE/HIGH). `mri_normalizer.go` normalizes 12 signals to z-scores. `mri_event_publisher.go` publishes deterioration events to KB-22 and KB-23. Existing `GetHistoryScores()` retrieves past MRI records. Gap: no 14-day sliding window trajectory with OLS slope, per-domain breakdown, or category crossing detection.

- [ ] **Step 27: Write failing test for MHRI trajectory**

```go
// kb-26-metabolic-digital-twin/internal/services/mri_trajectory_test.go
package services

import (
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
)

func TestComputeMRITrajectory_Improving(t *testing.T) {
    now := time.Now()
    scores := []MRIHistoryPoint{
        {Score: 55.0, Timestamp: now.Add(-13 * 24 * time.Hour)},
        {Score: 58.0, Timestamp: now.Add(-10 * 24 * time.Hour)},
        {Score: 62.0, Timestamp: now.Add(-7 * 24 * time.Hour)},
        {Score: 65.0, Timestamp: now.Add(-4 * 24 * time.Hour)},
        {Score: 70.0, Timestamp: now.Add(-1 * 24 * time.Hour)},
    }
    traj := ComputeMRITrajectory(scores)
    assert.Equal(t, "IMPROVING", traj.Trend)
    assert.True(t, traj.SlopePerDay > 0)
    assert.Equal(t, 55.0, traj.StartScore)
    assert.Equal(t, 70.0, traj.EndScore)
    assert.InDelta(t, 15.0, traj.DeltaScore, 0.1)
}

func TestComputeMRITrajectory_Declining(t *testing.T) {
    now := time.Now()
    scores := []MRIHistoryPoint{
        {Score: 80.0, Timestamp: now.Add(-12 * 24 * time.Hour)},
        {Score: 75.0, Timestamp: now.Add(-8 * 24 * time.Hour)},
        {Score: 68.0, Timestamp: now.Add(-4 * 24 * time.Hour)},
        {Score: 60.0, Timestamp: now.Add(-1 * 24 * time.Hour)},
    }
    traj := ComputeMRITrajectory(scores)
    assert.Equal(t, "DECLINING", traj.Trend)
    assert.True(t, traj.SlopePerDay < 0)
}

func TestComputeMRITrajectory_Stable(t *testing.T) {
    now := time.Now()
    scores := []MRIHistoryPoint{
        {Score: 70.0, Timestamp: now.Add(-13 * 24 * time.Hour)},
        {Score: 71.0, Timestamp: now.Add(-9 * 24 * time.Hour)},
        {Score: 69.5, Timestamp: now.Add(-5 * 24 * time.Hour)},
        {Score: 70.5, Timestamp: now.Add(-1 * 24 * time.Hour)},
    }
    traj := ComputeMRITrajectory(scores)
    assert.Equal(t, "STABLE", traj.Trend)
}

func TestComputeMRITrajectory_InsufficientData(t *testing.T) {
    traj := ComputeMRITrajectory([]MRIHistoryPoint{{Score: 70.0, Timestamp: time.Now()}})
    assert.Equal(t, "INSUFFICIENT_DATA", traj.Trend)
    assert.Equal(t, 0.0, traj.SlopePerDay)
}

func TestDetectCategoryCrossing(t *testing.T) {
    tests := []struct {
        prev, curr float64
        crossed    bool
        direction  string
    }{
        {72.0, 68.0, true, "WORSENED"},   // OPTIMAL→MILD
        {68.0, 72.0, true, "IMPROVED"},   // MILD→OPTIMAL
        {45.0, 38.0, true, "WORSENED"},   // MODERATE→HIGH
        {65.0, 63.0, false, ""},          // both MILD
    }
    for _, tt := range tests {
        c := DetectCategoryCrossing(tt.prev, tt.curr)
        assert.Equal(t, tt.crossed, c.Crossed, "prev=%.1f curr=%.1f", tt.prev, tt.curr)
        if tt.crossed {
            assert.Equal(t, tt.direction, c.Direction)
        }
    }
}
```

- [ ] **Step 28: Run test to verify it fails**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin
go test ./internal/services/ -run "TestComputeMRITrajectory|TestDetectCategoryCrossing" -v
```

Expected: FAIL — types and functions not defined.

- [ ] **Step 29: Implement MHRI trajectory engine**

```go
// kb-26-metabolic-digital-twin/internal/services/mri_trajectory.go
package services

import (
    "math"
    "time"
)

type MRIHistoryPoint struct {
    Score     float64   `json:"score"`
    Timestamp time.Time `json:"timestamp"`
}

type MRITrajectory struct {
    Trend       string  `json:"trend"`
    SlopePerDay float64 `json:"slope_per_day"`
    StartScore  float64 `json:"start_score"`
    EndScore    float64 `json:"end_score"`
    DeltaScore  float64 `json:"delta_score"`
    WindowDays  int     `json:"window_days"`
    DataPoints  int     `json:"data_points"`
}

type CategoryCrossing struct {
    Crossed      bool   `json:"crossed"`
    Direction    string `json:"direction,omitempty"`
    PrevCategory string `json:"prev_category,omitempty"`
    CurrCategory string `json:"curr_category,omitempty"`
}

// ComputeMRITrajectory computes 14-day MRI trajectory via OLS regression.
func ComputeMRITrajectory(scores []MRIHistoryPoint) MRITrajectory {
    if len(scores) < 3 {
        return MRITrajectory{Trend: "INSUFFICIENT_DATA"}
    }
    sorted := make([]MRIHistoryPoint, len(scores))
    copy(sorted, scores)
    sortByTimestamp(sorted)

    baseTime := sorted[0].Timestamp
    n := float64(len(sorted))
    var sumX, sumY, sumXY, sumX2 float64
    for _, pt := range sorted {
        x := pt.Timestamp.Sub(baseTime).Hours() / 24.0
        sumX += x
        sumY += pt.Score
        sumXY += x * pt.Score
        sumX2 += x * x
    }
    denom := n*sumX2 - sumX*sumX
    slope := 0.0
    if math.Abs(denom) > 1e-10 {
        slope = (n*sumXY - sumX*sumY) / denom
    }

    first, last := sorted[0], sorted[len(sorted)-1]
    windowDays := int(last.Timestamp.Sub(first.Timestamp).Hours() / 24.0)

    return MRITrajectory{
        Trend:       classifyMRITrend(slope),
        SlopePerDay: math.Round(slope*1000) / 1000,
        StartScore:  first.Score,
        EndScore:    last.Score,
        DeltaScore:  math.Round((last.Score-first.Score)*10) / 10,
        WindowDays:  windowDays,
        DataPoints:  len(sorted),
    }
}

func classifyMRITrend(slopePerDay float64) string {
    if slopePerDay > 0.3  { return "IMPROVING" }
    if slopePerDay < -0.3 { return "DECLINING" }
    return "STABLE"
}

// DetectCategoryCrossing checks if score crosses MRI category boundary.
// OPTIMAL(>=70), MILD(55-69), MODERATE(40-54), HIGH(<40).
func DetectCategoryCrossing(prevScore, currScore float64) CategoryCrossing {
    prevCat := categorizeMRIScore(prevScore)
    currCat := categorizeMRIScore(currScore)
    if prevCat == currCat {
        return CategoryCrossing{Crossed: false}
    }
    dir := "IMPROVED"
    if currScore < prevScore { dir = "WORSENED" }
    return CategoryCrossing{Crossed: true, Direction: dir,
        PrevCategory: prevCat, CurrCategory: currCat}
}

func categorizeMRIScore(score float64) string {
    if score >= 70 { return "OPTIMAL" }
    if score >= 55 { return "MILD_DYSREGULATION" }
    if score >= 40 { return "MODERATE_DETERIORATION" }
    return "HIGH_DETERIORATION"
}

func sortByTimestamp(pts []MRIHistoryPoint) {
    for i := 1; i < len(pts); i++ {
        for j := i; j > 0 && pts[j].Timestamp.Before(pts[j-1].Timestamp); j-- {
            pts[j], pts[j-1] = pts[j-1], pts[j]
        }
    }
}
```

- [ ] **Step 30: Run trajectory tests**

```bash
go test ./internal/services/ -run "TestComputeMRITrajectory|TestDetectCategoryCrossing" -v
```

Expected: All 5 tests PASS.

- [ ] **Step 31: Add trajectory API endpoint**

In `mri_handlers.go`, add:

```go
func (s *Server) getMRITrajectory(c *gin.Context) {
    patientID := c.Param("patientId")
    history, err := s.mriScorer.GetHistoryScores(patientID, 14)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch history"})
        return
    }
    points := make([]services.MRIHistoryPoint, len(history))
    for i, h := range history {
        points[i] = services.MRIHistoryPoint{Score: h.Score, Timestamp: h.Timestamp}
    }
    traj := services.ComputeMRITrajectory(points)
    c.JSON(http.StatusOK, gin.H{"data": traj})
}
```

Register in `routes.go` under the MRI group:

```go
    mri.GET("/:patientId/trajectory", s.getMRITrajectory)
```

- [ ] **Step 32: Run full KB-26 tests**

```bash
go test ./... -v -count=1
```

Expected: All tests PASS.

- [ ] **Step 33: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/mri_trajectory.go
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/mri_trajectory_test.go
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/mri_handlers.go
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/routes.go
git commit -m "feat(kb26): 14-day MHRI trajectory with OLS slope and category crossing

OLS regression over sliding 14-day window. Trend: IMPROVING/STABLE/DECLINING.
Category crossing detection for threshold alerting. New GET /mri/:id/trajectory."
```

---

### Task 8: IOR System (Intervention-Outcome Records)

**Files:**
- Create: `kb-20-patient-profile/internal/models/ior.go`
- Create: `kb-20-patient-profile/internal/services/ior_store.go`
- Create: `kb-20-patient-profile/internal/services/ior_store_test.go`
- Create: `kb-20-patient-profile/internal/services/ior_generator.go`
- Create: `kb-20-patient-profile/internal/services/ior_generator_test.go`
- Create: `kb-20-patient-profile/internal/api/ior_handlers.go`
- Create: `kb-20-patient-profile/migrations/005_ior_tables.sql`
- Modify: `kb-20-patient-profile/internal/api/routes.go`

**Context:** KB-20 already has PatientProfile with labs, medications, protocols. IOR tracks interventions (medication starts, dose changes, lifestyle prescriptions) and their measured outcomes (delta HbA1c, delta SBP) at 4/12/26/52-week checkpoints. Confounder scoring accounts for concurrent med changes, adherence, and lifestyle. The similar-patient query filters by stratum, CKM stage, and phenotype cluster. KB-20 uses GORM with PostgreSQL (port 5433), Gin for HTTP.

- [ ] **Step 34: Create IOR data models**

```go
// kb-20-patient-profile/internal/models/ior.go
package models

import (
    "time"

    "github.com/google/uuid"
    "gorm.io/datatypes"
)

type InterventionRecord struct {
    ID               uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
    PatientID        string         `gorm:"size:100;index;not null" json:"patient_id"`
    InterventionType string         `gorm:"size:30;not null" json:"intervention_type"` // MEDICATION_START, DOSE_CHANGE, LIFESTYLE_RX, REFERRAL
    DrugClass        *string        `gorm:"size:30" json:"drug_class,omitempty"`
    DrugName         *string        `gorm:"size:100" json:"drug_name,omitempty"`
    DoseChangeMg     *float64       `json:"dose_change_mg,omitempty"`
    PrescribedBy     *uuid.UUID     `gorm:"type:uuid" json:"prescribed_by,omitempty"`
    CardID           *uuid.UUID     `gorm:"type:uuid" json:"card_id,omitempty"`
    ProtocolID       *string        `gorm:"size:20" json:"protocol_id,omitempty"`
    ProtocolPhase    *string        `gorm:"size:20" json:"protocol_phase,omitempty"`
    StartDate        time.Time      `gorm:"not null" json:"start_date"`
    EndDate          *time.Time     `json:"end_date,omitempty"`
    Status           string         `gorm:"size:20;default:'ACTIVE'" json:"status"` // ACTIVE, COMPLETED, DISCONTINUED, SUPERSEDED
    // Confounder capture (Gap G3)
    ConcurrentMedChanges datatypes.JSON `gorm:"type:jsonb" json:"concurrent_med_changes,omitempty"`
    AdherenceAtStart     *float64       `json:"adherence_at_start,omitempty"`
    AdherenceAtEnd       *float64       `json:"adherence_at_end,omitempty"`
    LifestyleChanges     datatypes.JSON `gorm:"type:jsonb" json:"lifestyle_changes,omitempty"`
    SeasonAtStart        *string        `gorm:"size:10" json:"season_at_start,omitempty"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

type OutcomeRecord struct {
    ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
    InterventionID  uuid.UUID `gorm:"type:uuid;index;not null" json:"intervention_id"`
    PatientID       string    `gorm:"size:100;index;not null" json:"patient_id"`
    OutcomeType     string    `gorm:"size:30;not null" json:"outcome_type"` // DELTA_HBA1C, DELTA_SBP, DELTA_EGFR, DELTA_WEIGHT
    BaselineValue   float64   `json:"baseline_value"`
    OutcomeValue    float64   `json:"outcome_value"`
    DeltaValue      float64   `json:"delta_value"`
    DeltaPercent    float64   `json:"delta_percent"`
    MeasurementDate time.Time `json:"measurement_date"`
    WindowWeeks     int       `json:"window_weeks"`           // 4, 12, 26, 52
    ConfidenceLevel string    `gorm:"size:10" json:"confidence_level"` // HIGH, MODERATE, LOW
    ConfounderScore float64   `json:"confounder_score"`       // 0-1
    CreatedAt       time.Time `json:"created_at"`
}

// SimilarOutcomeQuery filters for finding similar-patient outcomes.
type SimilarOutcomeQuery struct {
    InterventionType string  `json:"intervention_type"`
    DrugClass        *string `json:"drug_class,omitempty"`
    CKMStage         *int    `json:"ckm_stage,omitempty"`
    PhenotypeCluster *string `json:"phenotype_cluster,omitempty"`
    WindowWeeks      int     `json:"window_weeks"`
    OutcomeType      string  `json:"outcome_type"`
}

// SimilarOutcomeResult aggregates outcomes for similar patients.
type SimilarOutcomeResult struct {
    N            int     `json:"n"`
    MedianDelta  float64 `json:"median_delta"`
    Q1Delta      float64 `json:"q1_delta"`
    Q3Delta      float64 `json:"q3_delta"`
    MeanDelta    float64 `json:"mean_delta"`
    OutcomeType  string  `json:"outcome_type"`
    WindowWeeks  int     `json:"window_weeks"`
}
```

- [ ] **Step 35: Create migration SQL**

```sql
-- kb-20-patient-profile/migrations/005_ior_tables.sql
CREATE TABLE IF NOT EXISTS intervention_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id VARCHAR(100) NOT NULL,
    intervention_type VARCHAR(30) NOT NULL,
    drug_class VARCHAR(30),
    drug_name VARCHAR(100),
    dose_change_mg DECIMAL(8,2),
    prescribed_by UUID,
    card_id UUID,
    protocol_id VARCHAR(20),
    protocol_phase VARCHAR(20),
    start_date TIMESTAMPTZ NOT NULL,
    end_date TIMESTAMPTZ,
    status VARCHAR(20) DEFAULT 'ACTIVE',
    concurrent_med_changes JSONB,
    adherence_at_start DECIMAL(3,2),
    adherence_at_end DECIMAL(3,2),
    lifestyle_changes JSONB,
    season_at_start VARCHAR(10),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_ir_patient ON intervention_records(patient_id);
CREATE INDEX idx_ir_type_status ON intervention_records(intervention_type, status);

CREATE TABLE IF NOT EXISTS outcome_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    intervention_id UUID NOT NULL REFERENCES intervention_records(id),
    patient_id VARCHAR(100) NOT NULL,
    outcome_type VARCHAR(30) NOT NULL,
    baseline_value DECIMAL(10,2),
    outcome_value DECIMAL(10,2),
    delta_value DECIMAL(10,2),
    delta_percent DECIMAL(8,2),
    measurement_date TIMESTAMPTZ,
    window_weeks INT,
    confidence_level VARCHAR(10),
    confounder_score DECIMAL(3,2),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_or_intervention ON outcome_records(intervention_id);
CREATE INDEX idx_or_patient ON outcome_records(patient_id);
CREATE INDEX idx_or_type_window ON outcome_records(outcome_type, window_weeks);
```

- [ ] **Step 36: Write failing test for IOR store**

```go
// kb-20-patient-profile/internal/services/ior_store_test.go
package services

import (
    "testing"
    "time"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "kb-patient-profile/internal/models"
)

func TestCreateIntervention(t *testing.T) {
    store := setupIORTestStore(t)
    rec := &models.InterventionRecord{
        PatientID:        "IOR-TEST-001",
        InterventionType: "MEDICATION_START",
        DrugClass:        strPtr("SGLT2i"),
        DrugName:         strPtr("Dapagliflozin"),
        DoseChangeMg:     float64Ptr(10.0),
        StartDate:        time.Now(),
        Status:           "ACTIVE",
    }
    err := store.CreateIntervention(rec)
    require.NoError(t, err)
    assert.NotEqual(t, uuid.Nil, rec.ID)
}

func TestCreateOutcome(t *testing.T) {
    store := setupIORTestStore(t)
    intervention := createTestIntervention(t, store, "IOR-TEST-002")

    outcome := &models.OutcomeRecord{
        InterventionID:  intervention.ID,
        PatientID:       "IOR-TEST-002",
        OutcomeType:     "DELTA_HBA1C",
        BaselineValue:   8.5,
        OutcomeValue:    7.7,
        DeltaValue:      -0.8,
        DeltaPercent:    -9.4,
        MeasurementDate: time.Now(),
        WindowWeeks:     12,
        ConfidenceLevel: "HIGH",
        ConfounderScore: 0.15,
    }
    err := store.CreateOutcome(outcome)
    require.NoError(t, err)
    assert.NotEqual(t, uuid.Nil, outcome.ID)
}

func TestGetInterventionsByPatient(t *testing.T) {
    store := setupIORTestStore(t)
    createTestIntervention(t, store, "IOR-LIST-001")
    createTestIntervention(t, store, "IOR-LIST-001")

    results, err := store.GetInterventionsByPatient("IOR-LIST-001", nil)
    require.NoError(t, err)
    assert.Len(t, results, 2)
}

func TestCreateOutcome_InvalidInterventionFK(t *testing.T) {
    store := setupIORTestStore(t)
    outcome := &models.OutcomeRecord{
        InterventionID: uuid.New(), // non-existent
        PatientID:      "IOR-TEST-003",
        OutcomeType:    "DELTA_SBP",
        WindowWeeks:    4,
    }
    err := store.CreateOutcome(outcome)
    assert.Error(t, err)
}

func strPtr(s string) *string       { return &s }
func float64Ptr(f float64) *float64 { return &f }
```

- [ ] **Step 37: Implement IOR store**

```go
// kb-20-patient-profile/internal/services/ior_store.go
package services

import (
    "fmt"
    "math"
    "sort"

    "gorm.io/gorm"

    "kb-patient-profile/internal/models"
)

type IORStore struct {
    db *gorm.DB
}

func NewIORStore(db *gorm.DB) *IORStore {
    return &IORStore{db: db}
}

func (s *IORStore) CreateIntervention(rec *models.InterventionRecord) error {
    return s.db.Create(rec).Error
}

func (s *IORStore) CreateOutcome(rec *models.OutcomeRecord) error {
    // Verify intervention exists
    var count int64
    s.db.Model(&models.InterventionRecord{}).Where("id = ?", rec.InterventionID).Count(&count)
    if count == 0 {
        return fmt.Errorf("intervention %s not found", rec.InterventionID)
    }
    return s.db.Create(rec).Error
}

type InterventionFilter struct {
    Status           *string
    InterventionType *string
}

func (s *IORStore) GetInterventionsByPatient(patientID string, filter *InterventionFilter) ([]models.InterventionRecord, error) {
    q := s.db.Where("patient_id = ?", patientID).Order("start_date DESC")
    if filter != nil {
        if filter.Status != nil           { q = q.Where("status = ?", *filter.Status) }
        if filter.InterventionType != nil { q = q.Where("intervention_type = ?", *filter.InterventionType) }
    }
    var results []models.InterventionRecord
    return results, q.Find(&results).Error
}

func (s *IORStore) GetOutcomesByIntervention(interventionID string) ([]models.OutcomeRecord, error) {
    var results []models.OutcomeRecord
    return results, s.db.Where("intervention_id = ?", interventionID).
        Order("window_weeks ASC").Find(&results).Error
}

func (s *IORStore) QuerySimilarOutcomes(q models.SimilarOutcomeQuery) (*models.SimilarOutcomeResult, error) {
    query := s.db.Model(&models.OutcomeRecord{}).
        Joins("JOIN intervention_records ir ON ir.id = outcome_records.intervention_id").
        Where("outcome_records.outcome_type = ? AND outcome_records.window_weeks = ?",
            q.OutcomeType, q.WindowWeeks).
        Where("ir.intervention_type = ?", q.InterventionType)

    if q.DrugClass != nil {
        query = query.Where("ir.drug_class = ?", *q.DrugClass)
    }
    if q.CKMStage != nil {
        query = query.Joins("JOIN patient_profiles pp ON pp.patient_id = ir.patient_id").
            Where("pp.ckm_stage = ?", *q.CKMStage)
    }
    if q.PhenotypeCluster != nil {
        query = query.Joins("JOIN patient_profiles pp2 ON pp2.patient_id = ir.patient_id").
            Where("pp2.phenotype_cluster = ?", *q.PhenotypeCluster)
    }

    var deltas []float64
    rows, err := query.Select("outcome_records.delta_value").Rows()
    if err != nil { return nil, err }
    defer rows.Close()
    for rows.Next() {
        var d float64
        rows.Scan(&d)
        deltas = append(deltas, d)
    }

    if len(deltas) == 0 {
        return &models.SimilarOutcomeResult{N: 0, OutcomeType: q.OutcomeType, WindowWeeks: q.WindowWeeks}, nil
    }

    sort.Float64s(deltas)
    return &models.SimilarOutcomeResult{
        N:           len(deltas),
        MedianDelta: percentile(deltas, 0.5),
        Q1Delta:     percentile(deltas, 0.25),
        Q3Delta:     percentile(deltas, 0.75),
        MeanDelta:   mean(deltas),
        OutcomeType: q.OutcomeType,
        WindowWeeks: q.WindowWeeks,
    }, nil
}

func percentile(sorted []float64, p float64) float64 {
    idx := p * float64(len(sorted)-1)
    lower := int(math.Floor(idx))
    upper := int(math.Ceil(idx))
    if lower == upper { return sorted[lower] }
    frac := idx - float64(lower)
    return sorted[lower]*(1-frac) + sorted[upper]*frac
}

func mean(vals []float64) float64 {
    sum := 0.0
    for _, v := range vals { sum += v }
    return sum / float64(len(vals))
}
```

- [ ] **Step 38: Run IOR store tests**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile
go test ./internal/services/ -run "TestCreateIntervention|TestCreateOutcome|TestGetInterventions" -v
```

Expected: All 4 tests PASS.

- [ ] **Step 39: Write IOR batch generator test**

```go
// kb-20-patient-profile/internal/services/ior_generator_test.go
package services

import (
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestComputeConfounderScore(t *testing.T) {
    tests := []struct {
        name     string
        factors  ConfounderFactors
        expected float64 // approximate range
    }{
        {"no confounders", ConfounderFactors{ConcurrentMedCount: 0, AdherenceDrop: 0}, 0.0},
        {"high confounding", ConfounderFactors{
            ConcurrentMedCount: 3, AdherenceDrop: 0.3, LifestyleChangeCount: 2, SeasonChanged: true,
        }, 0.7},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            score := ComputeConfounderScore(tt.factors)
            assert.InDelta(t, tt.expected, score, 0.2)
        })
    }
}

func TestShouldGenerateOutcome(t *testing.T) {
    now := time.Now()
    // 4 weeks ago intervention, should generate at week 4
    start := now.Add(-4 * 7 * 24 * time.Hour)
    assert.True(t, shouldGenerateOutcome(start, now, 4))
    // Only 2 weeks ago, not yet at week 4
    start2 := now.Add(-2 * 7 * 24 * time.Hour)
    assert.False(t, shouldGenerateOutcome(start2, now, 4))
}
```

- [ ] **Step 40: Implement IOR batch generator**

```go
// kb-20-patient-profile/internal/services/ior_generator.go
package services

import (
    "math"
    "time"

    "go.uber.org/zap"
    "gorm.io/gorm"

    "kb-patient-profile/internal/models"
)

type IORGenerator struct {
    db     *gorm.DB
    store  *IORStore
    logger *zap.Logger
}

func NewIORGenerator(db *gorm.DB, store *IORStore, logger *zap.Logger) *IORGenerator {
    return &IORGenerator{db: db, store: store, logger: logger}
}

var outcomeCheckpoints = []int{4, 12, 26, 52} // weeks

type ConfounderFactors struct {
    ConcurrentMedCount   int
    AdherenceDrop        float64 // positive = adherence decreased
    LifestyleChangeCount int
    SeasonChanged        bool
}

// ComputeConfounderScore returns 0-1 where higher = more confounded.
func ComputeConfounderScore(f ConfounderFactors) float64 {
    score := 0.0
    score += math.Min(float64(f.ConcurrentMedCount)*0.15, 0.45)
    score += math.Min(f.AdherenceDrop*0.5, 0.25)
    score += math.Min(float64(f.LifestyleChangeCount)*0.1, 0.2)
    if f.SeasonChanged { score += 0.1 }
    return math.Min(score, 1.0)
}

func shouldGenerateOutcome(startDate, now time.Time, windowWeeks int) bool {
    elapsed := now.Sub(startDate)
    windowDuration := time.Duration(windowWeeks) * 7 * 24 * time.Hour
    tolerance := 3 * 24 * time.Hour // ± 3 days
    return elapsed >= windowDuration-tolerance
}

// GenerateOutcomes scans active interventions and creates outcome records at checkpoints.
func (g *IORGenerator) GenerateOutcomes() (int, error) {
    var interventions []models.InterventionRecord
    if err := g.db.Where("status = 'ACTIVE'").Find(&interventions).Error; err != nil {
        return 0, err
    }

    now := time.Now()
    created := 0
    for _, iv := range interventions {
        for _, weeks := range outcomeCheckpoints {
            if !shouldGenerateOutcome(iv.StartDate, now, weeks) {
                continue
            }
            // Check if outcome already exists for this checkpoint
            var count int64
            g.db.Model(&models.OutcomeRecord{}).
                Where("intervention_id = ? AND window_weeks = ?", iv.ID, weeks).
                Count(&count)
            if count > 0 { continue }

            // Fetch latest lab for outcome measurement
            outcome, err := g.buildOutcome(iv, weeks)
            if err != nil {
                g.logger.Warn("skipping outcome generation", zap.Error(err),
                    zap.String("intervention_id", iv.ID.String()))
                continue
            }
            if err := g.store.CreateOutcome(outcome); err != nil {
                g.logger.Error("failed to create outcome", zap.Error(err))
                continue
            }
            created++
        }
    }
    return created, nil
}

func (g *IORGenerator) buildOutcome(iv models.InterventionRecord, windowWeeks int) (*models.OutcomeRecord, error) {
    // Determine outcome type from intervention type
    outcomeType := "DELTA_HBA1C" // default for medication interventions
    if iv.InterventionType == "LIFESTYLE_RX" {
        outcomeType = "DELTA_WEIGHT"
    }

    // Fetch baseline and current values from patient labs
    baseline, current, err := g.fetchLabValues(iv.PatientID, outcomeType, iv.StartDate)
    if err != nil {
        return nil, err
    }

    delta := current - baseline
    deltaPct := 0.0
    if math.Abs(baseline) > 0.001 {
        deltaPct = (delta / baseline) * 100
    }

    confounders := g.assessConfounders(iv)
    confidence := "HIGH"
    if confounders > 0.5 { confidence = "LOW" } else if confounders > 0.2 { confidence = "MODERATE" }

    return &models.OutcomeRecord{
        InterventionID:  iv.ID,
        PatientID:       iv.PatientID,
        OutcomeType:     outcomeType,
        BaselineValue:   baseline,
        OutcomeValue:    current,
        DeltaValue:      math.Round(delta*100) / 100,
        DeltaPercent:    math.Round(deltaPct*10) / 10,
        MeasurementDate: time.Now(),
        WindowWeeks:     windowWeeks,
        ConfidenceLevel: confidence,
        ConfounderScore: math.Round(confounders*100) / 100,
    }, nil
}

func (g *IORGenerator) fetchLabValues(patientID, outcomeType string, startDate time.Time) (float64, float64, error) {
    // Map outcome type to lab LOINC/field
    labField := "hba1c"
    switch outcomeType {
    case "DELTA_SBP":    labField = "sbp"
    case "DELTA_EGFR":   labField = "egfr"
    case "DELTA_WEIGHT": labField = "weight_kg"
    }

    var baseline, current float64
    // Baseline: closest lab to start_date
    g.db.Raw("SELECT value FROM lab_entries WHERE patient_id = ? AND lab_type = ? AND recorded_at <= ? ORDER BY recorded_at DESC LIMIT 1",
        patientID, labField, startDate).Scan(&baseline)
    // Current: latest lab
    g.db.Raw("SELECT value FROM lab_entries WHERE patient_id = ? AND lab_type = ? ORDER BY recorded_at DESC LIMIT 1",
        patientID, labField).Scan(&current)
    return baseline, current, nil
}

func (g *IORGenerator) assessConfounders(iv models.InterventionRecord) float64 {
    factors := ConfounderFactors{}
    // Count concurrent med changes in the intervention window
    var medCount int64
    g.db.Model(&models.InterventionRecord{}).
        Where("patient_id = ? AND id != ? AND start_date BETWEEN ? AND ?",
            iv.PatientID, iv.ID, iv.StartDate.Add(-7*24*time.Hour), time.Now()).
        Count(&medCount)
    factors.ConcurrentMedCount = int(medCount)

    // Adherence drop
    if iv.AdherenceAtStart != nil && iv.AdherenceAtEnd != nil {
        factors.AdherenceDrop = *iv.AdherenceAtStart - *iv.AdherenceAtEnd
    }
    return ComputeConfounderScore(factors)
}
```

- [ ] **Step 41: Create IOR API handlers**

```go
// kb-20-patient-profile/internal/api/ior_handlers.go
package api

import (
    "net/http"

    "github.com/gin-gonic/gin"

    "kb-patient-profile/internal/models"
)

func (s *Server) createIntervention(c *gin.Context) {
    var rec models.InterventionRecord
    if err := c.ShouldBindJSON(&rec); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    if err := s.iorStore.CreateIntervention(&rec); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusCreated, gin.H{"data": rec})
}

func (s *Server) getInterventions(c *gin.Context) {
    patientID := c.Query("patient_id")
    if patientID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "patient_id required"})
        return
    }
    results, err := s.iorStore.GetInterventionsByPatient(patientID, nil)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": results})
}

func (s *Server) getOutcomes(c *gin.Context) {
    interventionID := c.Query("intervention_id")
    if interventionID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "intervention_id required"})
        return
    }
    results, err := s.iorStore.GetOutcomesByIntervention(interventionID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": results})
}

func (s *Server) querySimilarOutcomes(c *gin.Context) {
    var q models.SimilarOutcomeQuery
    if err := c.ShouldBindJSON(&q); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    result, err := s.iorStore.QuerySimilarOutcomes(q)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": result})
}

func (s *Server) triggerOutcomeGeneration(c *gin.Context) {
    created, err := s.iorGenerator.GenerateOutcomes()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"outcomes_created": created})
}
```

- [ ] **Step 42: Register IOR routes**

In `routes.go`, add a new route group after the `thresholds` group:

```go
        // IOR — Intervention-Outcome Records
        ior := v1.Group("/ior")
        {
            ior.POST("/interventions", s.createIntervention)
            ior.GET("/interventions", s.getInterventions)
            ior.GET("/outcomes", s.getOutcomes)
            ior.POST("/similar-outcomes", s.querySimilarOutcomes)
            ior.POST("/generate", s.triggerOutcomeGeneration)
        }
```

- [ ] **Step 43: Run all IOR tests + KB-20 regression**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile
go test ./internal/services/ -run "TestCreate|TestGet|TestCompute|TestShould" -v
go test ./... -v -count=1
```

Expected: All tests PASS.

- [ ] **Step 44: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/models/ior.go
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/ior_store.go
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/ior_store_test.go
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/ior_generator.go
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/ior_generator_test.go
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/api/ior_handlers.go
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/api/routes.go
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/migrations/005_ior_tables.sql
git commit -m "feat(kb20): IOR system — intervention tracking with outcome generation

InterventionRecord + OutcomeRecord models with PostgreSQL schema.
CRUD store with similar-patient aggregate query (filters by stratum,
CKM stage, phenotype cluster). Batch generator at 4/12/26/52-week
checkpoints with confounder scoring. 5 REST endpoints."
```

---

---

## Phase C4: Decision Layer

### Task 11: KB-23 Dual-Domain Decision Card Generator

**Files:**
- Create: `kb-23-decision-cards/internal/models/dual_domain.go`
- Create: `kb-23-decision-cards/internal/services/dual_domain_classifier.go`
- Create: `kb-23-decision-cards/internal/services/dual_domain_classifier_test.go`
- Create: `kb-23-decision-cards/internal/services/four_pillar_evaluator.go`
- Create: `kb-23-decision-cards/internal/services/four_pillar_evaluator_test.go`
- Create: `kb-23-decision-cards/internal/services/conflict_detector.go`
- Create: `kb-23-decision-cards/internal/services/conflict_detector_test.go`
- Create: `kb-23-decision-cards/internal/services/urgency_calculator.go`
- Create: `kb-23-decision-cards/internal/services/ior_insight_provider.go`
- Modify: `kb-23-decision-cards/internal/models/enums.go` (add new card types + sources)

**Context:** KB-23 has `card_builder.go` (orchestrates confidence tier → MCU gate → recommendations → CTL panels), `enums.go` with `CardSource`, `CardStatus`, `Urgency` types. No dual-domain models exist. The card builder creates single-domain cards from KB-22 session events or clinical signals. Existing `CardSource` constants: `KB22_SESSION`, `HYPOGLYCAEMIA_FAST_PATH`, `PERTURBATION_DECAY`, `BEHAVIORAL_GAP`, `CLINICAL_SIGNAL`. Existing `Urgency` constants: `IMMEDIATE`, `URGENT`, `ROUTINE`, `SCHEDULED`.

- [ ] **Step 45: Add new enum values for dual-domain cards**

In `enums.go`, add after line 80 (after `SourceClinicalSignal`):

```go
    SourceDualDomain    CardSource = "DUAL_DOMAIN"
    SourceFourPillarGap CardSource = "FOUR_PILLAR_GAP"
```

- [ ] **Step 46: Create dual-domain models**

```go
// kb-23-decision-cards/internal/models/dual_domain.go
package models

// DomainStatus classifies a single clinical domain.
type DomainStatus string

const (
    DomainControlled   DomainStatus = "CONTROLLED"
    DomainAtTarget     DomainStatus = "AT_TARGET"
    DomainUncontrolled DomainStatus = "UNCONTROLLED"
)

// DualDomainState is the combined glycaemic + hemodynamic classification.
type DualDomainState struct {
    Glycaemic   DomainStatus `json:"glycaemic"`
    Hemodynamic DomainStatus `json:"hemodynamic"`
    Label       string       `json:"label"` // e.g. "GU-HT" (Glycaemic Uncontrolled, Hemodynamic At-Target)
    CKMStage    int          `json:"ckm_stage"`
}

// DualDomainInput holds the clinical data needed for classification.
type DualDomainInput struct {
    HbA1c           *float64 `json:"hba1c"`
    FBGTrajectory   string   `json:"fbg_trajectory"`   // from Module 3
    SBPAverage30d   *float64 `json:"sbp_average_30d"`
    ARV             *float64 `json:"arv"`               // from Module 7
    CKMStage        int      `json:"ckm_stage"`
}

// ConflictRecord identifies a drug-domain conflict.
type ConflictRecord struct {
    DrugClass         string `json:"drug_class"`
    GlycaemicEffect   string `json:"glycaemic_effect"`   // NEUTRAL, RAISES, LOWERS, SYNERGISTIC
    HemodynamicEffect string `json:"hemodynamic_effect"`
    Severity          string `json:"severity"`            // HIGH, MODERATE, LOW
    Resolution        string `json:"resolution_suggestion"`
}

// PillarStatus represents one of four clinical pillars.
type PillarStatus string

const (
    PillarAdequate  PillarStatus = "ADEQUATE"
    PillarGap       PillarStatus = "GAP_DETECTED"
    PillarUrgentGap PillarStatus = "URGENT_GAP"
)

// FourPillarResult is the evaluation of all 4 pillars.
type FourPillarResult struct {
    Medication PillarEvaluation `json:"medication"`
    Lifestyle  PillarEvaluation `json:"lifestyle"`
    Monitoring PillarEvaluation `json:"monitoring"`
    Referral   PillarEvaluation `json:"referral"`
}

type PillarEvaluation struct {
    Status          PillarStatus `json:"status"`
    Recommendations []string     `json:"recommendations,omitempty"`
    Rationale       string       `json:"rationale,omitempty"`
}
```

- [ ] **Step 47: Write failing test for dual-domain classifier**

```go
// kb-23-decision-cards/internal/services/dual_domain_classifier_test.go
package services

import (
    "testing"

    "github.com/stretchr/testify/assert"

    "kb-decision-cards/internal/models"
)

func TestClassifyDualDomain_BothControlled(t *testing.T) {
    input := models.DualDomainInput{
        HbA1c: float64Ptr(6.5), FBGTrajectory: "STABLE",
        SBPAverage30d: float64Ptr(125), ARV: float64Ptr(8.0), CKMStage: 1,
    }
    state := ClassifyDualDomain(input)
    assert.Equal(t, models.DomainControlled, state.Glycaemic)
    assert.Equal(t, models.DomainControlled, state.Hemodynamic)
    assert.Equal(t, "GC-HC", state.Label)
}

func TestClassifyDualDomain_GlycaemicUncontrolled_BPAtTarget(t *testing.T) {
    input := models.DualDomainInput{
        HbA1c: float64Ptr(9.2), FBGTrajectory: "RISING",
        SBPAverage30d: float64Ptr(132), ARV: float64Ptr(10.0), CKMStage: 2,
    }
    state := ClassifyDualDomain(input)
    assert.Equal(t, models.DomainUncontrolled, state.Glycaemic)
    assert.Equal(t, models.DomainAtTarget, state.Hemodynamic)
    assert.Equal(t, "GU-HT", state.Label)
}

func TestClassifyDualDomain_BothUncontrolled(t *testing.T) {
    input := models.DualDomainInput{
        HbA1c: float64Ptr(10.1), FBGTrajectory: "RAPID_RISING",
        SBPAverage30d: float64Ptr(155), ARV: float64Ptr(18.0), CKMStage: 3,
    }
    state := ClassifyDualDomain(input)
    assert.Equal(t, models.DomainUncontrolled, state.Glycaemic)
    assert.Equal(t, models.DomainUncontrolled, state.Hemodynamic)
    assert.Equal(t, "GU-HU", state.Label)
}

func TestClassifyDualDomain_NilHbA1c_DefaultsControlled(t *testing.T) {
    input := models.DualDomainInput{
        SBPAverage30d: float64Ptr(120), ARV: float64Ptr(8.0), CKMStage: 0,
    }
    state := ClassifyDualDomain(input)
    assert.Equal(t, models.DomainControlled, state.Glycaemic) // no data → assume controlled
}

func float64Ptr(f float64) *float64 { return &f }
```

- [ ] **Step 48: Implement dual-domain classifier**

```go
// kb-23-decision-cards/internal/services/dual_domain_classifier.go
package services

import "kb-decision-cards/internal/models"

// ClassifyDualDomain maps clinical inputs to one of 9 combined states.
func ClassifyDualDomain(input models.DualDomainInput) models.DualDomainState {
    glycaemic := classifyGlycaemic(input.HbA1c, input.FBGTrajectory)
    hemodynamic := classifyHemodynamic(input.SBPAverage30d, input.ARV)

    labels := map[models.DomainStatus]string{
        models.DomainControlled: "C", models.DomainAtTarget: "T", models.DomainUncontrolled: "U",
    }
    label := "G" + labels[glycaemic] + "-H" + labels[hemodynamic]

    return models.DualDomainState{
        Glycaemic: glycaemic, Hemodynamic: hemodynamic,
        Label: label, CKMStage: input.CKMStage,
    }
}

func classifyGlycaemic(hba1c *float64, trajectory string) models.DomainStatus {
    if hba1c == nil { return models.DomainControlled }
    if *hba1c >= 8.0 || trajectory == "RAPID_RISING" { return models.DomainUncontrolled }
    if *hba1c >= 7.0 || trajectory == "RISING"        { return models.DomainAtTarget }
    return models.DomainControlled
}

func classifyHemodynamic(sbp *float64, arv *float64) models.DomainStatus {
    if sbp == nil { return models.DomainControlled }
    highVariability := arv != nil && *arv > 14.0
    if *sbp >= 140 || (*sbp >= 130 && highVariability) { return models.DomainUncontrolled }
    if *sbp >= 130                                      { return models.DomainAtTarget }
    return models.DomainControlled
}
```

- [ ] **Step 49: Run classifier tests**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards
go test ./internal/services/ -run TestClassifyDualDomain -v
```

Expected: All 4 tests PASS.

- [ ] **Step 50: Write failing test for conflict detector**

```go
// kb-23-decision-cards/internal/services/conflict_detector_test.go
package services

import (
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestDetectConflicts_Thiazide(t *testing.T) {
    meds := []string{"THIAZIDE"}
    conflicts := DetectDrugDomainConflicts(meds)
    assert.Len(t, conflicts, 1)
    assert.Equal(t, "THIAZIDE", conflicts[0].DrugClass)
    assert.Equal(t, "RAISES", conflicts[0].GlycaemicEffect)
    assert.Equal(t, "LOWERS", conflicts[0].HemodynamicEffect)
    assert.Equal(t, "MODERATE", conflicts[0].Severity)
}

func TestDetectConflicts_SGLT2i_Synergistic(t *testing.T) {
    meds := []string{"SGLT2i"}
    conflicts := DetectDrugDomainConflicts(meds)
    assert.Len(t, conflicts, 1)
    assert.Equal(t, "SYNERGISTIC", conflicts[0].GlycaemicEffect)
    assert.Equal(t, "SYNERGISTIC", conflicts[0].HemodynamicEffect)
    assert.Equal(t, "LOW", conflicts[0].Severity) // beneficial, not a conflict
}

func TestDetectConflicts_BetaBlocker(t *testing.T) {
    meds := []string{"BETA_BLOCKER"}
    conflicts := DetectDrugDomainConflicts(meds)
    assert.Len(t, conflicts, 1)
    assert.Equal(t, "MASKS_HYPO", conflicts[0].GlycaemicEffect)
    assert.Equal(t, "HIGH", conflicts[0].Severity)
}

func TestDetectConflicts_NoConflictMeds(t *testing.T) {
    meds := []string{"METFORMIN"}
    conflicts := DetectDrugDomainConflicts(meds)
    assert.Len(t, conflicts, 0) // metformin has no cross-domain conflict
}
```

- [ ] **Step 51: Implement conflict detector**

```go
// kb-23-decision-cards/internal/services/conflict_detector.go
package services

import "kb-decision-cards/internal/models"

// crossDomainEffects maps drug classes to their glycaemic/hemodynamic effects.
var crossDomainEffects = map[string]models.ConflictRecord{
    "THIAZIDE": {
        DrugClass: "THIAZIDE", GlycaemicEffect: "RAISES", HemodynamicEffect: "LOWERS",
        Severity: "MODERATE", Resolution: "Monitor FBG closely; consider switching to indapamide (lower glycaemic impact)",
    },
    "BETA_BLOCKER": {
        DrugClass: "BETA_BLOCKER", GlycaemicEffect: "MASKS_HYPO", HemodynamicEffect: "LOWERS",
        Severity: "HIGH", Resolution: "Ensure CGM or regular SMBG; prefer cardioselective beta-blockers (bisoprolol)",
    },
    "CORTICOSTEROID": {
        DrugClass: "CORTICOSTEROID", GlycaemicEffect: "RAISES", HemodynamicEffect: "RAISES",
        Severity: "HIGH", Resolution: "Use lowest effective dose; intensify glucose and BP monitoring",
    },
    "SGLT2i": {
        DrugClass: "SGLT2i", GlycaemicEffect: "SYNERGISTIC", HemodynamicEffect: "SYNERGISTIC",
        Severity: "LOW", Resolution: "Dual benefit — lowers glucose, BP, and renal risk",
    },
    "GLP1_RA": {
        DrugClass: "GLP1_RA", GlycaemicEffect: "SYNERGISTIC", HemodynamicEffect: "SYNERGISTIC",
        Severity: "LOW", Resolution: "Dual benefit — lowers glucose and modest BP reduction",
    },
}

// DetectDrugDomainConflicts returns conflicts for the given medication list.
// Only returns entries where there IS a cross-domain interaction (positive or negative).
func DetectDrugDomainConflicts(activeMedClasses []string) []models.ConflictRecord {
    var conflicts []models.ConflictRecord
    for _, med := range activeMedClasses {
        if effect, ok := crossDomainEffects[med]; ok {
            conflicts = append(conflicts, effect)
        }
    }
    return conflicts
}
```

- [ ] **Step 52: Implement four-pillar evaluator**

```go
// kb-23-decision-cards/internal/services/four_pillar_evaluator.go
package services

import "kb-decision-cards/internal/models"

// FourPillarInput holds the data needed for pillar evaluation.
type FourPillarInput struct {
    DualDomainState models.DualDomainState
    ActiveMeds      []string // drug classes
    ExerciseMinWeek int
    SodiumMgDay     *float64
    WeightTrend     string   // GAINING, STABLE, LOSING
    LabScheduleOK   bool     // are labs on schedule?
    BPMeasureFreq   int      // BP measurements per week
    EGFR            *float64
    CKMStage        int
}

// EvaluateFourPillars evaluates medication, lifestyle, monitoring, referral.
func EvaluateFourPillars(input FourPillarInput) models.FourPillarResult {
    return models.FourPillarResult{
        Medication: evaluateMedicationPillar(input),
        Lifestyle:  evaluateLifestylePillar(input),
        Monitoring: evaluateMonitoringPillar(input),
        Referral:   evaluateReferralPillar(input),
    }
}

func evaluateMedicationPillar(input FourPillarInput) models.PillarEvaluation {
    if input.DualDomainState.Glycaemic == models.DomainUncontrolled &&
        input.DualDomainState.Hemodynamic == models.DomainUncontrolled {
        return models.PillarEvaluation{
            Status: models.PillarUrgentGap,
            Recommendations: []string{
                "Both domains uncontrolled — review medication intensification",
                "Consider SGLT2i or GLP-1 RA for dual benefit",
            },
            Rationale: "Concordant uncontrolled state requires dual-domain medication review",
        }
    }
    if input.DualDomainState.Glycaemic == models.DomainUncontrolled ||
        input.DualDomainState.Hemodynamic == models.DomainUncontrolled {
        return models.PillarEvaluation{
            Status: models.PillarGap,
            Recommendations: []string{"Single domain uncontrolled — intensify domain-specific therapy"},
        }
    }
    return models.PillarEvaluation{Status: models.PillarAdequate}
}

func evaluateLifestylePillar(input FourPillarInput) models.PillarEvaluation {
    var recs []string
    if input.ExerciseMinWeek < 150 {
        recs = append(recs, "Increase aerobic exercise to >= 150 min/week (ADA/AHA)")
    }
    if input.SodiumMgDay != nil && *input.SodiumMgDay > 2300 {
        recs = append(recs, "Reduce sodium intake to < 2300 mg/day")
    }
    if input.WeightTrend == "GAINING" {
        recs = append(recs, "Address weight gain — review diet plan")
    }
    if len(recs) >= 2 {
        return models.PillarEvaluation{Status: models.PillarGap, Recommendations: recs}
    }
    if len(recs) == 1 {
        return models.PillarEvaluation{Status: models.PillarGap, Recommendations: recs}
    }
    return models.PillarEvaluation{Status: models.PillarAdequate}
}

func evaluateMonitoringPillar(input FourPillarInput) models.PillarEvaluation {
    var recs []string
    if !input.LabScheduleOK {
        recs = append(recs, "Labs overdue — schedule HbA1c, eGFR, ACR")
    }
    if input.BPMeasureFreq < 7 && input.DualDomainState.Hemodynamic != models.DomainControlled {
        recs = append(recs, "Increase BP measurement to daily while hemodynamic domain not controlled")
    }
    if len(recs) > 0 {
        return models.PillarEvaluation{Status: models.PillarGap, Recommendations: recs}
    }
    return models.PillarEvaluation{Status: models.PillarAdequate}
}

func evaluateReferralPillar(input FourPillarInput) models.PillarEvaluation {
    var recs []string
    if input.EGFR != nil && *input.EGFR < 30 {
        recs = append(recs, "Nephrology referral — eGFR < 30")
    }
    if input.CKMStage >= 3 {
        recs = append(recs, "Cardiology referral — CKM Stage >= 3")
    }
    status := models.PillarAdequate
    if len(recs) > 0 { status = models.PillarUrgentGap }
    return models.PillarEvaluation{Status: status, Recommendations: recs}
}
```

- [ ] **Step 53: Write four-pillar test**

```go
// kb-23-decision-cards/internal/services/four_pillar_evaluator_test.go
package services

import (
    "testing"

    "github.com/stretchr/testify/assert"

    "kb-decision-cards/internal/models"
)

func TestFourPillar_BothUncontrolled_MedicationUrgentGap(t *testing.T) {
    input := FourPillarInput{
        DualDomainState: models.DualDomainState{
            Glycaemic: models.DomainUncontrolled, Hemodynamic: models.DomainUncontrolled,
        },
        ExerciseMinWeek: 60, BPMeasureFreq: 3,
    }
    result := EvaluateFourPillars(input)
    assert.Equal(t, models.PillarUrgentGap, result.Medication.Status)
    assert.Contains(t, result.Medication.Recommendations[1], "SGLT2i")
}

func TestFourPillar_LowExercise_LifestyleGap(t *testing.T) {
    input := FourPillarInput{
        DualDomainState: models.DualDomainState{
            Glycaemic: models.DomainControlled, Hemodynamic: models.DomainControlled,
        },
        ExerciseMinWeek: 60,
    }
    result := EvaluateFourPillars(input)
    assert.Equal(t, models.PillarGap, result.Lifestyle.Status)
}

func TestFourPillar_LowEGFR_ReferralUrgent(t *testing.T) {
    input := FourPillarInput{
        DualDomainState: models.DualDomainState{
            Glycaemic: models.DomainControlled, Hemodynamic: models.DomainControlled,
        },
        EGFR: float64Ptr(25), CKMStage: 3,
    }
    result := EvaluateFourPillars(input)
    assert.Equal(t, models.PillarUrgentGap, result.Referral.Status)
    assert.Len(t, result.Referral.Recommendations, 2) // nephro + cardio
}

func TestFourPillar_AllAdequate(t *testing.T) {
    input := FourPillarInput{
        DualDomainState: models.DualDomainState{
            Glycaemic: models.DomainControlled, Hemodynamic: models.DomainControlled,
        },
        ExerciseMinWeek: 200, LabScheduleOK: true, BPMeasureFreq: 7,
        EGFR: float64Ptr(90), CKMStage: 0,
    }
    result := EvaluateFourPillars(input)
    assert.Equal(t, models.PillarAdequate, result.Medication.Status)
    assert.Equal(t, models.PillarAdequate, result.Lifestyle.Status)
    assert.Equal(t, models.PillarAdequate, result.Monitoring.Status)
    assert.Equal(t, models.PillarAdequate, result.Referral.Status)
}
```

- [ ] **Step 54: Implement urgency calculator + IOR insight provider**

```go
// kb-23-decision-cards/internal/services/urgency_calculator.go
package services

import "kb-decision-cards/internal/models"

// CalculateDualDomainUrgency combines domain urgencies.
func CalculateDualDomainUrgency(pillars models.FourPillarResult, state models.DualDomainState) models.Urgency {
    // If either domain is urgent gap → IMMEDIATE
    if pillars.Medication.Status == models.PillarUrgentGap ||
        pillars.Referral.Status == models.PillarUrgentGap {
        return models.UrgencyImmediate
    }
    // Concordant deterioration: both uncontrolled → URGENT
    if state.Glycaemic == models.DomainUncontrolled &&
        state.Hemodynamic == models.DomainUncontrolled {
        return models.UrgencyUrgent
    }
    // Any gap → ROUTINE
    if pillars.Medication.Status == models.PillarGap ||
        pillars.Lifestyle.Status == models.PillarGap ||
        pillars.Monitoring.Status == models.PillarGap {
        return models.UrgencyRoutine
    }
    return models.UrgencyScheduled
}
```

```go
// kb-23-decision-cards/internal/services/ior_insight_provider.go
package services

import (
    "fmt"
    "net/http"
    "encoding/json"
    "time"

    "kb-decision-cards/internal/models"
)

// IORInsightProvider fetches similar-patient outcomes from KB-20 IOR API.
type IORInsightProvider struct {
    kb20BaseURL string
    client      *http.Client
}

func NewIORInsightProvider(kb20URL string) *IORInsightProvider {
    return &IORInsightProvider{
        kb20BaseURL: kb20URL,
        client:      &http.Client{Timeout: 5 * time.Second},
    }
}

// IORInsight is a human-readable evidence summary for card enrichment.
type IORInsight struct {
    Summary     string  `json:"summary"`
    N           int     `json:"n"`
    MedianDelta float64 `json:"median_delta"`
    OutcomeType string  `json:"outcome_type"`
    WindowWeeks int     `json:"window_weeks"`
}

// GetInsight queries KB-20 for similar-patient outcomes.
func (p *IORInsightProvider) GetInsight(query models.SimilarOutcomeQuery) (*IORInsight, error) {
    body, _ := json.Marshal(query)
    resp, err := p.client.Post(p.kb20BaseURL+"/api/v1/ior/similar-outcomes",
        "application/json", bytes.NewReader(body))
    if err != nil { return nil, err }
    defer resp.Body.Close()

    var result struct {
        Data models.SimilarOutcomeResult `json:"data"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil { return nil, err }

    if result.Data.N == 0 { return nil, nil } // no data

    drugLabel := ""
    if query.DrugClass != nil { drugLabel = *query.DrugClass }

    return &IORInsight{
        Summary: fmt.Sprintf("Similar patients (n=%d) on %s showed median %s of %.1f at %d weeks",
            result.Data.N, drugLabel, query.OutcomeType, result.Data.MedianDelta, query.WindowWeeks),
        N: result.Data.N, MedianDelta: result.Data.MedianDelta,
        OutcomeType: query.OutcomeType, WindowWeeks: query.WindowWeeks,
    }, nil
}
```

- [ ] **Step 55: Run all Task 11 tests**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards
go test ./internal/services/ -run "TestClassify|TestDetect|TestFourPillar" -v
go test ./... -v -count=1
```

Expected: All new + existing tests PASS.

- [ ] **Step 56: Commit Task 11**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/models/dual_domain.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/models/enums.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/dual_domain_classifier.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/dual_domain_classifier_test.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/four_pillar_evaluator.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/four_pillar_evaluator_test.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/conflict_detector.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/conflict_detector_test.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/urgency_calculator.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/ior_insight_provider.go
git commit -m "feat(kb23): dual-domain card system — classifier, 4-pillar, conflicts, IOR

9-state dual-domain classifier (3×3 glycaemic×hemodynamic). Four-pillar
evaluator (medication, lifestyle, monitoring, referral). Drug-domain
conflict detector (thiazide, beta-blocker, corticosteroid, SGLT2i synergy).
Urgency calculator with concordant deterioration escalation. IOR insight
provider fetches similar-patient evidence from KB-20."
```

---

### Task 12: Physician Feedback Capture

**Files:**
- Create: `kb-23-decision-cards/internal/models/feedback.go`
- Create: `kb-23-decision-cards/internal/services/feedback_store.go`
- Create: `kb-23-decision-cards/internal/services/feedback_store_test.go`
- Create: `kb-23-decision-cards/internal/api/feedback_handlers.go`
- Create: `kb-23-decision-cards/migrations/005_feedback_table.sql`
- Modify: `kb-23-decision-cards/internal/api/routes.go`

**Context:** KB-23 uses Gin, GORM, PostgreSQL. No feedback models exist. The feedback model has 21 fields tracking physician responses to decision cards (ACCEPT/MODIFY/REJECT/DEFER) with modification details, rejection reasons, time-to-decision, and IOR insight viewing.

- [ ] **Step 57: Create feedback model**

```go
// kb-23-decision-cards/internal/models/feedback.go
package models

import (
    "time"

    "github.com/google/uuid"
    "gorm.io/datatypes"
)

type PhysicianFeedback struct {
    ID                 uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
    CardID             uuid.UUID      `gorm:"type:uuid;index;not null" json:"card_id"`
    PatientID          uuid.UUID      `gorm:"type:uuid;index" json:"patient_id"`
    PhysicianID        uuid.UUID      `gorm:"type:uuid;index;not null" json:"physician_id"`
    SessionID          *uuid.UUID     `gorm:"type:uuid" json:"session_id,omitempty"`
    ActionTaken        string         `gorm:"size:10;not null" json:"action_taken"` // ACCEPT, MODIFY, REJECT, DEFER
    // Modification details
    ModifiedDrugClass  *string        `gorm:"size:30" json:"modified_drug_class,omitempty"`
    ModifiedDose       *float64       `json:"modified_dose,omitempty"`
    ModifiedFrequency  *string        `gorm:"size:30" json:"modified_frequency,omitempty"`
    ModificationReason *string        `gorm:"size:200" json:"modification_reason,omitempty"`
    // Rejection details
    RejectionReason    *string        `gorm:"size:30" json:"rejection_reason,omitempty"`
    RejectionFreeText  *string        `gorm:"size:500" json:"rejection_free_text,omitempty"`
    // Deferral details
    DeferralReason     *string        `gorm:"size:200" json:"deferral_reason,omitempty"`
    DeferUntilDate     *time.Time     `json:"defer_until_date,omitempty"`
    // Context
    TimeToDecisionSec  *int           `json:"time_to_decision_sec,omitempty"`
    ViewedSections     datatypes.JSON `gorm:"type:jsonb" json:"viewed_sections,omitempty"`
    IORInsightViewed   bool           `gorm:"default:false" json:"ior_insight_viewed"`
    ConfidenceInAction *int           `json:"confidence_in_action,omitempty"` // 1-5
    // Metadata
    Platform           string         `gorm:"size:20;not null" json:"platform"` // WEB_DASHBOARD, MOBILE_APP
    CreatedAt          time.Time      `json:"created_at"`
    UpdatedAt          time.Time      `json:"updated_at"`
}

// FeedbackStats aggregates feedback metrics.
type FeedbackStats struct {
    TotalCards       int     `json:"total_cards"`
    AcceptRate       float64 `json:"accept_rate"`
    ModifyRate       float64 `json:"modify_rate"`
    RejectRate       float64 `json:"reject_rate"`
    DeferRate        float64 `json:"defer_rate"`
    MedianDecisionSec *int   `json:"median_decision_sec,omitempty"`
}
```

- [ ] **Step 58: Create migration SQL**

```sql
-- kb-23-decision-cards/migrations/005_feedback_table.sql
CREATE TABLE IF NOT EXISTS physician_feedbacks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    card_id UUID NOT NULL,
    patient_id UUID,
    physician_id UUID NOT NULL,
    session_id UUID,
    action_taken VARCHAR(10) NOT NULL CHECK (action_taken IN ('ACCEPT','MODIFY','REJECT','DEFER')),
    modified_drug_class VARCHAR(30),
    modified_dose DECIMAL(8,2),
    modified_frequency VARCHAR(30),
    modification_reason VARCHAR(200),
    rejection_reason VARCHAR(30),
    rejection_free_text VARCHAR(500),
    deferral_reason VARCHAR(200),
    defer_until_date TIMESTAMPTZ,
    time_to_decision_sec INT,
    viewed_sections JSONB,
    ior_insight_viewed BOOLEAN DEFAULT FALSE,
    confidence_in_action INT CHECK (confidence_in_action BETWEEN 1 AND 5),
    platform VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_fb_card ON physician_feedbacks(card_id);
CREATE INDEX idx_fb_physician ON physician_feedbacks(physician_id);
CREATE INDEX idx_fb_action ON physician_feedbacks(action_taken);
```

- [ ] **Step 59: Write failing test for feedback store**

```go
// kb-23-decision-cards/internal/services/feedback_store_test.go
package services

import (
    "testing"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "kb-decision-cards/internal/models"
)

func TestCreateFeedback_Accept(t *testing.T) {
    store := setupFeedbackTestStore(t)
    fb := &models.PhysicianFeedback{
        CardID:      uuid.New(),
        PhysicianID: uuid.New(),
        ActionTaken: "ACCEPT",
        Platform:    "WEB_DASHBOARD",
    }
    err := store.CreateFeedback(fb)
    require.NoError(t, err)
    assert.NotEqual(t, uuid.Nil, fb.ID)
}

func TestCreateFeedback_RejectWithReason(t *testing.T) {
    store := setupFeedbackTestStore(t)
    reason := "CLINICALLY_INAPPROPRIATE"
    fb := &models.PhysicianFeedback{
        CardID:          uuid.New(),
        PhysicianID:     uuid.New(),
        ActionTaken:     "REJECT",
        RejectionReason: &reason,
        Platform:        "MOBILE_APP",
    }
    err := store.CreateFeedback(fb)
    require.NoError(t, err)
}

func TestCreateFeedback_InvalidAction(t *testing.T) {
    store := setupFeedbackTestStore(t)
    fb := &models.PhysicianFeedback{
        CardID:      uuid.New(),
        PhysicianID: uuid.New(),
        ActionTaken: "INVALID",
        Platform:    "WEB_DASHBOARD",
    }
    err := store.CreateFeedback(fb)
    assert.Error(t, err)
}

func TestGetFeedbackStats(t *testing.T) {
    store := setupFeedbackTestStore(t)
    physID := uuid.New()
    // Create 3 accepts, 1 reject
    for i := 0; i < 3; i++ {
        store.CreateFeedback(&models.PhysicianFeedback{
            CardID: uuid.New(), PhysicianID: physID,
            ActionTaken: "ACCEPT", Platform: "WEB_DASHBOARD",
        })
    }
    store.CreateFeedback(&models.PhysicianFeedback{
        CardID: uuid.New(), PhysicianID: physID,
        ActionTaken: "REJECT", Platform: "WEB_DASHBOARD",
    })

    stats, err := store.GetFeedbackStats(physID)
    require.NoError(t, err)
    assert.Equal(t, 4, stats.TotalCards)
    assert.InDelta(t, 0.75, stats.AcceptRate, 0.01)
    assert.InDelta(t, 0.25, stats.RejectRate, 0.01)
}
```

- [ ] **Step 60: Implement feedback store**

```go
// kb-23-decision-cards/internal/services/feedback_store.go
package services

import (
    "fmt"

    "github.com/google/uuid"
    "gorm.io/gorm"

    "kb-decision-cards/internal/models"
)

type FeedbackStore struct {
    db *gorm.DB
}

func NewFeedbackStore(db *gorm.DB) *FeedbackStore {
    return &FeedbackStore{db: db}
}

var validActions = map[string]bool{
    "ACCEPT": true, "MODIFY": true, "REJECT": true, "DEFER": true,
}

func (s *FeedbackStore) CreateFeedback(fb *models.PhysicianFeedback) error {
    if !validActions[fb.ActionTaken] {
        return fmt.Errorf("invalid action_taken: %s", fb.ActionTaken)
    }
    return s.db.Create(fb).Error
}

func (s *FeedbackStore) GetFeedbackByCard(cardID uuid.UUID) ([]models.PhysicianFeedback, error) {
    var results []models.PhysicianFeedback
    return results, s.db.Where("card_id = ?", cardID).Order("created_at DESC").Find(&results).Error
}

func (s *FeedbackStore) GetFeedbackStats(physicianID uuid.UUID) (*models.FeedbackStats, error) {
    var feedbacks []models.PhysicianFeedback
    if err := s.db.Where("physician_id = ?", physicianID).Find(&feedbacks).Error; err != nil {
        return nil, err
    }
    total := len(feedbacks)
    if total == 0 {
        return &models.FeedbackStats{}, nil
    }

    counts := map[string]int{}
    for _, fb := range feedbacks {
        counts[fb.ActionTaken]++
    }
    n := float64(total)
    return &models.FeedbackStats{
        TotalCards: total,
        AcceptRate: float64(counts["ACCEPT"]) / n,
        ModifyRate: float64(counts["MODIFY"]) / n,
        RejectRate: float64(counts["REJECT"]) / n,
        DeferRate:  float64(counts["DEFER"]) / n,
    }, nil
}
```

- [ ] **Step 61: Create feedback API handlers and register routes**

```go
// kb-23-decision-cards/internal/api/feedback_handlers.go
package api

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

func (s *Server) createFeedback(c *gin.Context) {
    var fb models.PhysicianFeedback
    if err := c.ShouldBindJSON(&fb); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    if err := s.feedbackStore.CreateFeedback(&fb); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusCreated, gin.H{"data": fb})
}

func (s *Server) getFeedbackByCard(c *gin.Context) {
    cardID, err := uuid.Parse(c.Query("card_id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid card_id"})
        return
    }
    results, err := s.feedbackStore.GetFeedbackByCard(cardID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": results})
}

func (s *Server) getFeedbackStats(c *gin.Context) {
    physicianID, err := uuid.Parse(c.Query("physician_id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid physician_id"})
        return
    }
    stats, err := s.feedbackStore.GetFeedbackStats(physicianID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": stats})
}
```

Register in `routes.go`:

```go
        feedback := v1.Group("/feedback")
        {
            feedback.POST("", s.createFeedback)
            feedback.GET("", s.getFeedbackByCard)
            feedback.GET("/stats", s.getFeedbackStats)
        }
```

- [ ] **Step 62: Run all Task 12 tests + KB-23 regression**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards
go test ./internal/services/ -run "TestCreateFeedback|TestGetFeedback" -v
go test ./... -v -count=1
```

Expected: All tests PASS.

- [ ] **Step 63: Commit Task 12**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/models/feedback.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/feedback_store.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/feedback_store_test.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/api/feedback_handlers.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/api/routes.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/migrations/005_feedback_table.sql
git commit -m "feat(kb23): physician feedback capture — 21-field model with stats

PhysicianFeedback model tracking ACCEPT/MODIFY/REJECT/DEFER with
modification details, rejection reasons, time-to-decision, IOR insight
viewing. FeedbackStore with CRUD + per-physician acceptance stats.
POST/GET /feedback endpoints."
```

---

---

## Phase C5: Learning Layer

### Task 13: Phenotype Clustering Pipeline (Python)

**Files:**
- Create: `backend/shared-infrastructure/phenotype-clustering/requirements.txt`
- Create: `backend/shared-infrastructure/phenotype-clustering/src/feature_extractor.py`
- Create: `backend/shared-infrastructure/phenotype-clustering/src/clustering_pipeline.py`
- Create: `backend/shared-infrastructure/phenotype-clustering/src/therapy_mapper.py`
- Create: `backend/shared-infrastructure/phenotype-clustering/src/centroid_exporter.py`
- Create: `backend/shared-infrastructure/phenotype-clustering/src/kb20_updater.py`
- Create: `backend/shared-infrastructure/phenotype-clustering/tests/test_feature_extractor.py`
- Create: `backend/shared-infrastructure/phenotype-clustering/tests/test_clustering.py`
- Create: `backend/shared-infrastructure/phenotype-clustering/tests/test_therapy_mapper.py`
- Create: `backend/shared-infrastructure/phenotype-clustering/configs/clustering_config.yaml`

**Context:** Standalone Python batch pipeline. Reads 21 features from KB-20, clusters via UMAP→HDBSCAN, writes back via PATCH `/api/v1/patient/:id/v4-state`. Fixed random seeds for reproducibility.

- [ ] **Step 64: Create project structure and config**

```bash
mkdir -p backend/shared-infrastructure/phenotype-clustering/{src,tests,configs}
```

`requirements.txt`:
```
numpy>=1.24.0
pandas>=2.0.0
scikit-learn>=1.3.0
umap-learn>=0.5.4
hdbscan>=0.8.33
requests>=2.31.0
pyyaml>=6.0
pytest>=7.4.0
```

`configs/clustering_config.yaml`:
```yaml
kb20_base_url: "http://localhost:8131"
random_seed: 42
umap: { n_components: 5, n_neighbors: 15, min_dist: 0.1 }
hdbscan: { min_cluster_size: 20, min_samples: 5, cluster_selection_method: "eom" }
validation: { min_silhouette: 0.3, min_clusters: 4, max_clusters: 12 }
imputation: { strategy: "median", max_missing_pct: 0.3 }
```

- [ ] **Step 65: Write feature extractor test**

```python
# tests/test_feature_extractor.py
import pytest, numpy as np
from src.feature_extractor import extract_features, FEATURE_NAMES, encode_dip_classification

def test_feature_names_count():
    assert len(FEATURE_NAMES) == 21

def test_extract_from_profile():
    profile = {"profile": {"age": 55, "hba1c": 7.8, "egfr": 72.0, "bmi": 28.5, "waist_cm": 95},
               "data": {"arv_sbp_7d": 11.5, "dip_classification": "NON_DIPPER", "engagement_composite": 0.75}}
    features = extract_features(profile)
    assert len(features) == 21
    assert features[0] == 7.8   # hba1c
    assert features[20] == 55   # age

def test_missing_values_are_nan():
    features = extract_features({"profile": {"age": 40}, "data": {}})
    assert np.isnan(features[0])  # hba1c missing
    assert features[20] == 40

def test_dip_encoding():
    assert encode_dip_classification("DIPPER") == 0
    assert encode_dip_classification("REVERSE_DIPPER") == 2
    assert encode_dip_classification(None) == -1
```

- [ ] **Step 66: Implement feature extractor**

```python
# src/feature_extractor.py
import math
from typing import List, Optional

FEATURE_NAMES = [
    "hba1c", "fbg_mean", "fbg_variability", "ppbg_mean", "glucose_trajectory",
    "sbp_mean", "sbp_arv", "dbp_mean", "bp_trajectory", "dip_pattern",
    "egfr", "egfr_slope", "acr", "bmi", "waist_cm",
    "total_cholesterol", "ldl", "triglycerides",
    "engagement_score", "med_adherence", "age",
]

DIP_ENCODING = {"DIPPER": 0, "NON_DIPPER": 1, "REVERSE_DIPPER": 2, "EXTREME_DIPPER": 3}

def encode_dip_classification(dip: Optional[str]) -> int:
    if not dip: return -1
    return DIP_ENCODING.get(dip, -1)

def extract_features(patient_data: dict) -> List[float]:
    p, d = patient_data.get("profile", {}), patient_data.get("data", {})
    def get(key, *srcs):
        for s in srcs:
            v = s.get(key)
            if v is not None: return float(v)
        return float("nan")
    return [
        get("hba1c", p), get("fbg_mean", p, d), get("fbg_variability", p, d),
        get("ppbg_mean", p, d), get("glucose_trajectory", d),
        get("sbp_mean", p, d), get("arv_sbp_7d", p, d), get("dbp_mean", p, d),
        get("bp_trajectory", d),
        float(encode_dip_classification(d.get("dip_classification") or p.get("dip_classification"))),
        get("egfr", p), get("egfr_slope", p, d), get("uacr", p, d),
        get("bmi", p), get("waist_cm", p),
        get("total_cholesterol", p, d), get("ldl_cholesterol", p, d), get("triglycerides", p, d),
        get("engagement_composite", p, d), get("med_adherence", p, d), get("age", p),
    ]
```

- [ ] **Step 67: Run feature extractor tests**

```bash
cd backend/shared-infrastructure/phenotype-clustering
pip install -r requirements.txt && python -m pytest tests/test_feature_extractor.py -v
```

Expected: All 4 tests PASS.

- [ ] **Step 68: Write clustering pipeline test**

```python
# tests/test_clustering.py
import pytest, numpy as np
from src.clustering_pipeline import ClusteringPipeline

@pytest.fixture
def sample_features():
    rng = np.random.RandomState(42)
    c1 = rng.normal(loc=[6,100,15,130,0,120,8,80,0,0,90,0,20,32,100,200,120,150,0.9,0.85,35], scale=1, size=(35,21))
    c2 = rng.normal(loc=[8.5,140,25,180,1,145,16,90,1,2,35,-3,200,27,85,250,160,200,0.5,0.6,72], scale=1, size=(35,21))
    c3 = rng.normal(loc=[7,110,18,150,0,130,11,82,0,0,75,-1,40,26,88,210,130,160,0.75,0.8,55], scale=1, size=(30,21))
    return np.vstack([c1, c2, c3])

def test_produces_clusters(sample_features):
    p = ClusteringPipeline(random_seed=42, min_cluster_size=10, min_samples=3)
    labels = p.fit_predict(sample_features)
    assert len(labels) == 100
    assert len(set(labels) - {-1}) >= 2

def test_silhouette(sample_features):
    p = ClusteringPipeline(random_seed=42, min_cluster_size=10, min_samples=3)
    p.fit_predict(sample_features)
    assert p.silhouette_score >= 0.2

def test_centroids_shape(sample_features):
    p = ClusteringPipeline(random_seed=42, min_cluster_size=10, min_samples=3)
    p.fit_predict(sample_features)
    assert p.get_centroids().shape[1] == 21
```

- [ ] **Step 69: Implement clustering pipeline**

```python
# src/clustering_pipeline.py
import numpy as np
from scipy.stats.mstats import winsorize
from sklearn.preprocessing import StandardScaler
from sklearn.impute import SimpleImputer
from sklearn.metrics import silhouette_score as sk_silhouette
import umap, hdbscan

class ClusteringPipeline:
    def __init__(self, random_seed=42, n_components=5, n_neighbors=15,
                 min_dist=0.1, min_cluster_size=20, min_samples=5, max_missing_pct=0.3):
        self.random_seed = random_seed
        self.n_components, self.n_neighbors, self.min_dist = n_components, n_neighbors, min_dist
        self.min_cluster_size, self.min_samples = min_cluster_size, min_samples
        self.max_missing_pct = max_missing_pct
        self.labels_ = self.silhouette_score = self.embedding_ = None
        self._scaled_data = self._valid_mask = None

    def fit_predict(self, features: np.ndarray) -> np.ndarray:
        missing_pct = np.isnan(features).mean(axis=1)
        self._valid_mask = missing_pct <= self.max_missing_pct
        valid = features[self._valid_mask]
        imputed = SimpleImputer(strategy="median").fit_transform(valid)
        winsorized = np.apply_along_axis(lambda c: winsorize(c, limits=[0.01, 0.01]), 0, imputed)
        self._scaled_data = StandardScaler().fit_transform(winsorized)
        self.embedding_ = umap.UMAP(
            n_components=self.n_components, n_neighbors=self.n_neighbors,
            min_dist=self.min_dist, random_state=self.random_seed
        ).fit_transform(self._scaled_data)
        valid_labels = hdbscan.HDBSCAN(
            min_cluster_size=self.min_cluster_size, min_samples=self.min_samples,
            cluster_selection_method="eom"
        ).fit_predict(self.embedding_)
        non_noise = valid_labels != -1
        self.silhouette_score = (
            sk_silhouette(self.embedding_[non_noise], valid_labels[non_noise])
            if non_noise.sum() > 1 and len(set(valid_labels[non_noise])) >= 2 else 0.0
        )
        self.labels_ = np.full(len(features), -1, dtype=int)
        self.labels_[self._valid_mask] = valid_labels
        return self.labels_

    def get_centroids(self) -> np.ndarray:
        valid_labels = self.labels_[self._valid_mask]
        return np.array([self._scaled_data[valid_labels == l].mean(0)
                         for l in sorted(set(valid_labels) - {-1})])
```

- [ ] **Step 70: Run clustering tests**

```bash
python -m pytest tests/test_clustering.py -v
```

Expected: All 3 tests PASS.

- [ ] **Step 71: Write therapy mapper test and implementation**

```python
# tests/test_therapy_mapper.py
import numpy as np
from src.therapy_mapper import map_cluster_to_therapy

def test_high_glucose_high_bp_low_adherence():
    c = np.zeros(21); c[0]=9.0; c[5]=150; c[19]=0.4; c[20]=55
    assert map_cluster_to_therapy(c)["pathway"] == "INTENSIVE_DUAL_DOMAIN_ADHERENCE"

def test_elderly_ckd():
    c = np.zeros(21); c[0]=7.2; c[10]=28; c[20]=75
    assert map_cluster_to_therapy(c)["pathway"] == "CONSERVATIVE_DEPRESCRIBING"

def test_young_obese_engaged():
    c = np.zeros(21); c[0]=6.2; c[13]=32; c[18]=0.9; c[20]=35
    assert map_cluster_to_therapy(c)["pathway"] == "LIFESTYLE_FIRST_AGGRESSIVE"
```

```python
# src/therapy_mapper.py
import numpy as np

IDX = {"hba1c": 0, "sbp": 5, "arv": 6, "egfr": 10, "bmi": 13, "engagement": 18, "adherence": 19, "age": 20}

def map_cluster_to_therapy(centroid: np.ndarray) -> dict:
    g = {k: centroid[v] for k, v in IDX.items()}
    if g["age"] >= 70 and g["egfr"] < 30:
        return {"pathway": "CONSERVATIVE_DEPRESCRIBING",
                "rationale": "Elderly + advanced CKD — minimize polypharmacy"}
    if g["hba1c"] >= 8.0 and g["sbp"] >= 140 and g["adherence"] < 0.6:
        return {"pathway": "INTENSIVE_DUAL_DOMAIN_ADHERENCE",
                "rationale": "Dual uncontrolled + adherence gap"}
    if g["age"] < 45 and g["bmi"] >= 30 and g["engagement"] >= 0.7:
        return {"pathway": "LIFESTYLE_FIRST_AGGRESSIVE",
                "rationale": "Young, engaged, metabolic syndrome"}
    if g["sbp"] >= 140 and g["hba1c"] < 7.5:
        return {"pathway": "BP_VARIABILITY_FOCUSED" if g["arv"] > 14 else "BP_INTENSIFICATION",
                "rationale": "Glycaemic OK, BP uncontrolled"}
    if g["hba1c"] >= 8.0:
        return {"pathway": "GLYCAEMIC_INTENSIFICATION", "rationale": "BP OK, glycaemic uncontrolled"}
    return {"pathway": "MAINTENANCE_OPTIMIZATION", "rationale": "Near target — optimize"}
```

- [ ] **Step 72: Implement KB-20 updater and centroid exporter**

```python
# src/kb20_updater.py
import requests, logging
from typing import List, Tuple
logger = logging.getLogger(__name__)

def update_cluster_assignments(kb20_url: str,
                                assignments: List[Tuple[str, str, float]]) -> int:
    updated = 0
    for pid, label, conf in assignments:
        try:
            r = requests.patch(f"{kb20_url}/api/v1/patient/{pid}/v4-state",
                               json={"phenotype_cluster": label, "phenotype_confidence": round(conf, 2)}, timeout=5)
            if r.status_code == 200: updated += 1
            else: logger.warning(f"Failed {pid}: {r.status_code}")
        except Exception as e: logger.error(f"Error {pid}: {e}")
    return updated
```

```python
# src/centroid_exporter.py
import json, numpy as np
from pathlib import Path
from src.therapy_mapper import map_cluster_to_therapy

def export_centroids(centroids: np.ndarray, labels: list, output_dir: str) -> str:
    out = [{"cluster_id": int(l), "cluster_label": f"CLUSTER_{l}",
            "centroid": c.tolist(), **map_cluster_to_therapy(c)}
           for c, l in zip(centroids, labels)]
    path = Path(output_dir) / "cluster_centroids.json"
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(out, indent=2))
    return str(path)
```

- [ ] **Step 73: Run all clustering tests**

```bash
cd backend/shared-infrastructure/phenotype-clustering
python -m pytest tests/ -v
```

Expected: All tests PASS.

- [ ] **Step 74: Commit Task 13**

```bash
git add backend/shared-infrastructure/phenotype-clustering/
git commit -m "feat(clustering): phenotype pipeline — UMAP+HDBSCAN with therapy mapper

21-feature extractor from KB-20. UMAP 21D→5D + HDBSCAN clustering with
silhouette validation. Decision-tree therapy mapper (6 pathways).
Centroid exporter + KB-20 PATCH updater. Fixed seed reproducibility."
```

---

### Task 14: Feedback Analysis Pipelines + Governance

**Files:**
- Create: `kb-23-decision-cards/internal/models/rule_change_proposal.go`
- Create: `kb-23-decision-cards/internal/services/feedback_analyzer.go`
- Create: `kb-23-decision-cards/internal/services/feedback_analyzer_test.go`
- Create: `kb-23-decision-cards/internal/services/governance_service.go`
- Create: `kb-23-decision-cards/internal/services/governance_service_test.go`
- Create: `kb-23-decision-cards/internal/api/governance_handlers.go`
- Create: `kb-23-decision-cards/migrations/006_governance_table.sql`
- Modify: `kb-23-decision-cards/internal/api/routes.go`

**Context:** Feedback data from Task 12 (`PhysicianFeedback` model) feeds analysis pipelines. KB-23 uses GORM/Gin/PostgreSQL.

- [ ] **Step 75: Create governance model**

```go
// kb-23-decision-cards/internal/models/rule_change_proposal.go
package models

import (
    "time"
    "github.com/google/uuid"
    "gorm.io/datatypes"
)

type RuleChangeProposal struct {
    ID              uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
    ProposalType    string         `gorm:"size:30;not null" json:"proposal_type"`
    SourcePipeline  string         `gorm:"size:30;not null" json:"source_pipeline"`
    CurrentRule     datatypes.JSON `gorm:"type:jsonb" json:"current_rule,omitempty"`
    ProposedChange  datatypes.JSON `gorm:"type:jsonb;not null" json:"proposed_change"`
    EvidenceCount   int            `gorm:"not null" json:"evidence_count"`
    ConfidenceScore float64        `json:"confidence_score"`
    Status          string         `gorm:"size:20;default:'PROPOSED'" json:"status"`
    ReviewerID      *uuid.UUID     `gorm:"type:uuid" json:"reviewer_id,omitempty"`
    ReviewNotes     *string        `gorm:"size:500" json:"review_notes,omitempty"`
    ApprovedAt      *time.Time     `json:"approved_at,omitempty"`
    DeployedAt      *time.Time     `json:"deployed_at,omitempty"`
    CreatedAt       time.Time      `json:"created_at"`
}

type RejectionPattern struct {
    CardType      string   `json:"card_type"`
    Stratum       string   `json:"stratum"`
    RejectionRate float64  `json:"rejection_rate"`
    TopReasons    []string `json:"top_reasons"`
    SampleSize    int      `json:"sample_size"`
}
```

- [ ] **Step 76: Write feedback analyzer test**

```go
// kb-23-decision-cards/internal/services/feedback_analyzer_test.go
package services

import (
    "testing"
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "kb-decision-cards/internal/models"
)

func TestDetectRejectionPatterns(t *testing.T) {
    store := setupFeedbackTestStore(t)
    analyzer := NewFeedbackAnalyzer(store.db)
    for i := 0; i < 6; i++ {
        store.CreateFeedback(&models.PhysicianFeedback{CardID: uuid.New(), PhysicianID: uuid.New(), ActionTaken: "ACCEPT", Platform: "WEB_DASHBOARD"})
    }
    reason := "CLINICALLY_INAPPROPRIATE"
    for i := 0; i < 4; i++ {
        store.CreateFeedback(&models.PhysicianFeedback{CardID: uuid.New(), PhysicianID: uuid.New(), ActionTaken: "REJECT", RejectionReason: &reason, Platform: "WEB_DASHBOARD"})
    }
    patterns := analyzer.DetectRejectionPatterns(0.30)
    assert.NotEmpty(t, patterns)
    assert.True(t, patterns[0].RejectionRate >= 0.30)
}
```

- [ ] **Step 77: Implement feedback analyzer**

```go
// kb-23-decision-cards/internal/services/feedback_analyzer.go
package services

import (
    "github.com/google/uuid"
    "gorm.io/gorm"
    "kb-decision-cards/internal/models"
)

type FeedbackAnalyzer struct { db *gorm.DB }

func NewFeedbackAnalyzer(db *gorm.DB) *FeedbackAnalyzer { return &FeedbackAnalyzer{db: db} }

func (a *FeedbackAnalyzer) DetectRejectionPatterns(threshold float64) []models.RejectionPattern {
    var fbs []models.PhysicianFeedback
    a.db.Find(&fbs)
    if len(fbs) == 0 { return nil }
    rejects, reasons := 0, map[string]int{}
    for _, fb := range fbs {
        if fb.ActionTaken == "REJECT" {
            rejects++
            if fb.RejectionReason != nil { reasons[*fb.RejectionReason]++ }
        }
    }
    rate := float64(rejects) / float64(len(fbs))
    if rate < threshold { return nil }
    var topR []string
    for r := range reasons { topR = append(topR, r) }
    return []models.RejectionPattern{{CardType: "ALL", RejectionRate: rate, TopReasons: topR, SampleSize: len(fbs)}}
}

func (a *FeedbackAnalyzer) ComputeAcceptanceMetrics(physicianID uuid.UUID) models.FeedbackStats {
    var fbs []models.PhysicianFeedback
    a.db.Where("physician_id = ?", physicianID).Find(&fbs)
    total := len(fbs)
    if total == 0 { return models.FeedbackStats{} }
    counts := map[string]int{}
    for _, fb := range fbs { counts[fb.ActionTaken]++ }
    n := float64(total)
    return models.FeedbackStats{TotalCards: total, AcceptRate: float64(counts["ACCEPT"]) / n,
        ModifyRate: float64(counts["MODIFY"]) / n, RejectRate: float64(counts["REJECT"]) / n, DeferRate: float64(counts["DEFER"]) / n}
}
```

- [ ] **Step 78: Write governance service test**

```go
// kb-23-decision-cards/internal/services/governance_service_test.go
package services

import (
    "testing"
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "kb-decision-cards/internal/models"
)

func TestGovernance_ValidTransitions(t *testing.T) {
    svc := setupGovernanceTestService(t)
    p := &models.RuleChangeProposal{ProposalType: "SAFETY_RULE_MODIFY", SourcePipeline: "REJECTION_PATTERN", EvidenceCount: 15, ConfidenceScore: 0.85}
    require.NoError(t, svc.CreateProposal(p))
    assert.Equal(t, "PROPOSED", p.Status)
    require.NoError(t, svc.TransitionStatus(p.ID, "UNDER_REVIEW", nil))
    rid := uuid.New()
    require.NoError(t, svc.TransitionStatus(p.ID, "APPROVED", &rid))
    require.NoError(t, svc.TransitionStatus(p.ID, "DEPLOYED", nil))
}

func TestGovernance_InvalidTransition(t *testing.T) {
    svc := setupGovernanceTestService(t)
    p := &models.RuleChangeProposal{ProposalType: "SAFETY_RULE_MODIFY", SourcePipeline: "REJECTION_PATTERN", EvidenceCount: 5, ConfidenceScore: 0.7}
    svc.CreateProposal(p)
    assert.Error(t, svc.TransitionStatus(p.ID, "DEPLOYED", nil)) // skip UNDER_REVIEW
}
```

- [ ] **Step 79: Implement governance service**

```go
// kb-23-decision-cards/internal/services/governance_service.go
package services

import (
    "fmt"
    "time"
    "github.com/google/uuid"
    "gorm.io/gorm"
    "kb-decision-cards/internal/models"
)

type GovernanceService struct { db *gorm.DB }

func NewGovernanceService(db *gorm.DB) *GovernanceService { return &GovernanceService{db: db} }

var validTransitions = map[string][]string{
    "PROPOSED": {"UNDER_REVIEW"}, "UNDER_REVIEW": {"APPROVED", "REJECTED"},
    "APPROVED": {"DEPLOYED"}, "REJECTED": {}, "DEPLOYED": {},
}

func (g *GovernanceService) CreateProposal(p *models.RuleChangeProposal) error {
    p.Status = "PROPOSED"
    return g.db.Create(p).Error
}

func (g *GovernanceService) TransitionStatus(id uuid.UUID, newStatus string, reviewerID *uuid.UUID) error {
    var p models.RuleChangeProposal
    if err := g.db.First(&p, "id = ?", id).Error; err != nil { return err }
    valid := false
    for _, s := range validTransitions[p.Status] { if s == newStatus { valid = true; break } }
    if !valid { return fmt.Errorf("invalid transition: %s → %s", p.Status, newStatus) }
    upd := map[string]interface{}{"status": newStatus}
    if reviewerID != nil { upd["reviewer_id"] = *reviewerID }
    if newStatus == "APPROVED" { now := time.Now(); upd["approved_at"] = &now }
    if newStatus == "DEPLOYED" { now := time.Now(); upd["deployed_at"] = &now }
    return g.db.Model(&p).Updates(upd).Error
}

func (g *GovernanceService) ListProposals(status string) ([]models.RuleChangeProposal, error) {
    var results []models.RuleChangeProposal
    q := g.db.Order("created_at DESC")
    if status != "" { q = q.Where("status = ?", status) }
    return results, q.Find(&results).Error
}
```

- [ ] **Step 80: Create governance handlers, routes, migration**

`governance_handlers.go`: POST `/governance/proposals`, PUT `/governance/proposals/:id/transition`, GET `/governance/proposals`

`migrations/006_governance_table.sql`:
```sql
CREATE TABLE IF NOT EXISTS rule_change_proposals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    proposal_type VARCHAR(30) NOT NULL, source_pipeline VARCHAR(30) NOT NULL,
    current_rule JSONB, proposed_change JSONB NOT NULL,
    evidence_count INT NOT NULL, confidence_score DECIMAL(4,2),
    status VARCHAR(20) DEFAULT 'PROPOSED',
    reviewer_id UUID, review_notes VARCHAR(500),
    approved_at TIMESTAMPTZ, deployed_at TIMESTAMPTZ, created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_rcp_status ON rule_change_proposals(status);
```

- [ ] **Step 81: Run all Task 14 tests + KB-23 regression**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards
go test ./... -v -count=1
```

Expected: All tests PASS.

- [ ] **Step 82: Commit Task 14**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/models/rule_change_proposal.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/feedback_analyzer.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/feedback_analyzer_test.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/governance_service.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/governance_service_test.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/api/governance_handlers.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/api/routes.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/migrations/006_governance_table.sql
git commit -m "feat(kb23): feedback pipelines + governance lifecycle

Rejection pattern detector (>30% flagging). Per-physician acceptance
metrics. Governance state machine: PROPOSED→UNDER_REVIEW→APPROVED→DEPLOYED.
3 governance API endpoints."
```

---

## Phase C6: Market Shim

### Task 15: Market Configuration Infrastructure

**Files:**
- Create: `backend/shared-infrastructure/market-configs/shared/base_clinical_params.yaml`
- Create: `backend/shared-infrastructure/market-configs/india/clinical_params.yaml`
- Create: `backend/shared-infrastructure/market-configs/india/channels.yaml`
- Create: `backend/shared-infrastructure/market-configs/india/pharma_shim.yaml`
- Create: `backend/shared-infrastructure/market-configs/india/food_db_config.yaml`
- Create: `backend/shared-infrastructure/market-configs/india/compliance.yaml`
- Create: `backend/shared-infrastructure/market-configs/australia/clinical_params.yaml`
- Create: `backend/shared-infrastructure/market-configs/australia/channels.yaml`
- Create: `backend/shared-infrastructure/market-configs/australia/pharma_shim.yaml`
- Create: `backend/shared-infrastructure/market-configs/australia/food_db_config.yaml`
- Create: `backend/shared-infrastructure/market-configs/australia/compliance.yaml`
- Create: `backend/shared-infrastructure/market-configs/australia/indigenous_overrides.yaml`
- Create: `backend/shared-infrastructure/market-configs/loader/config.go`
- Create: `backend/shared-infrastructure/market-configs/loader/config_test.go`
- Create: `backend/shared-infrastructure/market-configs/loader/go.mod`
- Modify: `kb-20-patient-profile/cmd/server/main.go` (inject market config)
- Modify: `kb-23-decision-cards/cmd/server/main.go` (inject market config)
- Modify: `kb-26-metabolic-digital-twin/cmd/server/main.go` (inject market config)

**Context:** No market-configs directory exists. KB-20/KB-23/KB-26 currently hardcode clinical thresholds (e.g., HbA1c targets, BP targets, MHRI normalization ranges). India uses RSSDI/ISH guidelines and IFCT food database; Australia uses RACGP/Heart Foundation guidelines and AUSNUT food database. Australia requires indigenous population overrides per NACCHO guidelines. The config loader must merge shared base → market-specific → population overrides in that precedence order.

- [ ] **Step 83: Write test for config loader**

```go
// backend/shared-infrastructure/market-configs/loader/config_test.go
package marketconfig

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestLoad_India(t *testing.T) {
    cfg, err := Load("india", testConfigDir(t))
    require.NoError(t, err)
    assert.Equal(t, "india", cfg.Market)

    // RSSDI: HbA1c target 7.0%
    assert.Equal(t, 7.0, cfg.ClinicalParams.HbA1cTarget)
    // ISH: SBP target <130
    assert.Equal(t, 130.0, cfg.ClinicalParams.SBPTarget)
    // DBP target
    assert.Equal(t, 80.0, cfg.ClinicalParams.DBPTarget)
    // FPG target
    assert.Equal(t, 130.0, cfg.ClinicalParams.FPGTarget)

    // IFCT food database
    assert.Equal(t, "IFCT_2017", cfg.FoodDB.DatabaseID)

    // Channels — India uses WhatsApp
    assert.Contains(t, cfg.Channels.EnabledChannels, "whatsapp")

    // Compliance — ABDM
    assert.Equal(t, "ABDM", cfg.Compliance.HealthDataFramework)
}

func TestLoad_Australia(t *testing.T) {
    cfg, err := Load("australia", testConfigDir(t))
    require.NoError(t, err)
    assert.Equal(t, "australia", cfg.Market)

    // RACGP: HbA1c target 7.0%, with 8.0% for elderly
    assert.Equal(t, 7.0, cfg.ClinicalParams.HbA1cTarget)
    assert.Equal(t, 8.0, cfg.ClinicalParams.HbA1cTargetElderly)
    // Heart Foundation: SBP target <140
    assert.Equal(t, 140.0, cfg.ClinicalParams.SBPTarget)

    // AUSNUT food database
    assert.Equal(t, "AUSNUT_2011_13", cfg.FoodDB.DatabaseID)

    // Channels — Australia uses SMS + email
    assert.Contains(t, cfg.Channels.EnabledChannels, "sms")
    assert.Contains(t, cfg.Channels.EnabledChannels, "email")

    // Compliance — Privacy Act
    assert.Equal(t, "MY_HEALTH_RECORDS_ACT", cfg.Compliance.HealthDataFramework)
}

func TestLoad_AustraliaIndigenousOverrides(t *testing.T) {
    cfg, err := Load("australia", testConfigDir(t))
    require.NoError(t, err)

    // NACCHO: relaxed HbA1c target for indigenous populations
    assert.NotNil(t, cfg.IndigenousOverrides)
    assert.Equal(t, 8.0, cfg.IndigenousOverrides.HbA1cTarget)
    assert.Equal(t, 140.0, cfg.IndigenousOverrides.SBPTarget)
}

func TestLoad_SharedBaseDefaults(t *testing.T) {
    cfg, err := Load("india", testConfigDir(t))
    require.NoError(t, err)

    // Shared base values inherited
    assert.Equal(t, 70.0, cfg.ClinicalParams.HypoThresholdMgDL)
    assert.Equal(t, 54.0, cfg.ClinicalParams.SevereHypoThresholdMgDL)
    assert.Equal(t, 250.0, cfg.ClinicalParams.HyperThresholdMgDL)
}

func TestLoad_InvalidMarket(t *testing.T) {
    _, err := Load("brazil", testConfigDir(t))
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "market directory not found")
}

func TestLoad_MergeOrder(t *testing.T) {
    // Market-specific values override shared base
    cfg, err := Load("india", testConfigDir(t))
    require.NoError(t, err)
    // India overrides SBP target from shared base (140) to 130
    assert.Equal(t, 130.0, cfg.ClinicalParams.SBPTarget)
}

// testConfigDir returns the path to the test fixtures (the actual market-configs directory).
func testConfigDir(t *testing.T) string {
    t.Helper()
    // Walk up from loader/ to market-configs/
    dir, err := os.Getwd()
    require.NoError(t, err)
    return filepath.Dir(dir)
}
```

- [ ] **Step 84: Run test to verify it fails**

```bash
cd backend/shared-infrastructure/market-configs/loader
go test ./... -v
```

Expected: FAIL — `Load` function not defined, YAML files not present.

- [ ] **Step 85: Create shared base clinical params**

```yaml
# backend/shared-infrastructure/market-configs/shared/base_clinical_params.yaml

# Shared clinical parameter defaults — overridden by market-specific configs.
# Sources: WHO/IDF universal guidelines.

glycaemic:
  hba1c_target: 7.0            # % — WHO/IDF default
  fpg_target_mg_dl: 130.0      # mg/dL — fasting plasma glucose
  ppg_target_mg_dl: 180.0      # mg/dL — postprandial glucose
  hypo_threshold_mg_dl: 70.0   # mg/dL — Level 1 hypoglycaemia
  severe_hypo_threshold_mg_dl: 54.0  # mg/dL — Level 2 hypoglycaemia
  hyper_threshold_mg_dl: 250.0 # mg/dL — severe hyperglycaemia

hemodynamic:
  sbp_target: 140.0            # mmHg — WHO default
  dbp_target: 90.0             # mmHg — WHO default
  sbp_crisis: 180.0            # mmHg — hypertensive crisis
  dbp_crisis: 120.0            # mmHg — hypertensive crisis

renal:
  egfr_mild_threshold: 60.0    # mL/min/1.73m² — CKD stage 3a
  egfr_moderate_threshold: 45.0
  egfr_severe_threshold: 30.0
  uacr_microalbuminuria: 30.0  # mg/g
  uacr_macroalbuminuria: 300.0
  potassium_low: 3.5           # mEq/L
  potassium_high: 5.5

metabolic:
  bmi_overweight: 25.0         # kg/m² — WHO
  bmi_obese: 30.0
  ldl_target: 100.0            # mg/dL
  tg_hdl_ratio_risk: 3.5
```

- [ ] **Step 86: Create India clinical params**

```yaml
# backend/shared-infrastructure/market-configs/india/clinical_params.yaml

# India clinical parameters.
# Sources: RSSDI 2023, ISH 2020, ICMR guidelines.

glycaemic:
  hba1c_target: 7.0            # RSSDI: 7.0% for most adults
  hba1c_target_elderly: 8.0    # RSSDI: relaxed for elderly/frail
  fpg_target_mg_dl: 130.0
  ppg_target_mg_dl: 180.0

hemodynamic:
  sbp_target: 130.0            # ISH 2020: <130 for DM patients
  dbp_target: 80.0             # ISH 2020: <80 for DM patients

metabolic:
  bmi_overweight: 23.0         # Asian BMI cutoffs (WHO Asia-Pacific)
  bmi_obese: 25.0              # Asian BMI cutoffs
  waist_risk_male_cm: 90.0     # IDF Asian cutoff
  waist_risk_female_cm: 80.0   # IDF Asian cutoff
```

- [ ] **Step 87: Create India channels config**

```yaml
# backend/shared-infrastructure/market-configs/india/channels.yaml

enabled_channels:
  - whatsapp
  - sms
  - push_notification
  - ivr                         # Interactive Voice Response for low-literacy

whatsapp:
  provider: "meta_cloud_api"
  template_language: "en"
  fallback_languages: ["hi", "ta", "te", "kn", "mr"]
  max_message_length: 1024

sms:
  provider: "twilio"
  sender_id: "CARDFIT"
  unicode_support: true

push_notification:
  provider: "firebase"

ivr:
  provider: "exotel"
  languages: ["hi", "en"]
```

- [ ] **Step 88: Create India pharma shim, food DB, and compliance configs**

```yaml
# backend/shared-infrastructure/market-configs/india/pharma_shim.yaml

formulary_source: "NLEM_2022"   # National List of Essential Medicines
currency: "INR"
prescription_format: "CDSCO"
common_brands:
  metformin: ["Glycomet", "Glucophage", "Obimet"]
  glimepiride: ["Amaryl", "Glimestar"]
  empagliflozin: ["Jardiance"]
  telmisartan: ["Telma", "Telmikind"]
  amlodipine: ["Amlong", "Amlip"]
```

```yaml
# backend/shared-infrastructure/market-configs/india/food_db_config.yaml

database_id: "IFCT_2017"
database_name: "Indian Food Composition Tables 2017"
source_authority: "NIN_ICMR"
sodium_estimation_method: "IFCT_DIRECT"
regional_food_databases:
  - "IFCT_2017"
  - "NIN_2020_supplement"
default_portion_size_g: 150
meal_pattern: "3_MAIN_2_SNACK"
```

```yaml
# backend/shared-infrastructure/market-configs/india/compliance.yaml

health_data_framework: "ABDM"
data_residency: "IN"
consent_model: "ABDM_CONSENT_MANAGER"
identifier_system: "ABHA"
audit_standard: "ABDM_HEALTH_DATA_MANAGEMENT_POLICY"
data_retention_years: 7
encryption_standard: "AES_256"
```

- [ ] **Step 89: Create Australia clinical params**

```yaml
# backend/shared-infrastructure/market-configs/australia/clinical_params.yaml

# Australia clinical parameters.
# Sources: RACGP 2024, Heart Foundation 2022, ADS-ADEA.

glycaemic:
  hba1c_target: 7.0            # RACGP: 7.0% for most adults
  hba1c_target_elderly: 8.0    # RACGP: relaxed for elderly/frail
  fpg_target_mg_dl: 126.0
  ppg_target_mg_dl: 180.0

hemodynamic:
  sbp_target: 140.0            # Heart Foundation: <140 general
  sbp_target_dm: 130.0         # Heart Foundation: <130 for DM + high CV risk
  dbp_target: 90.0             # Heart Foundation: <90 general

metabolic:
  bmi_overweight: 25.0         # Standard WHO cutoffs
  bmi_obese: 30.0
  waist_risk_male_cm: 94.0     # IDF Caucasian cutoff
  waist_risk_female_cm: 80.0
```

- [ ] **Step 90: Create Australia channels, pharma, food DB, compliance, and indigenous overrides**

```yaml
# backend/shared-infrastructure/market-configs/australia/channels.yaml

enabled_channels:
  - sms
  - email
  - push_notification
  - my_health_record

sms:
  provider: "twilio"
  sender_id: "CardioFit"

email:
  provider: "ses"
  from_domain: "notifications.cardiofit.com.au"

push_notification:
  provider: "firebase"

my_health_record:
  integration: "MHR_FHIR_GATEWAY"
  document_types: ["shared_health_summary", "event_summary"]
```

```yaml
# backend/shared-infrastructure/market-configs/australia/pharma_shim.yaml

formulary_source: "PBS"         # Pharmaceutical Benefits Scheme
currency: "AUD"
prescription_format: "PBS_AUTHORITY"
common_brands:
  metformin: ["Diabex", "Diaformin", "Glucophage"]
  glimepiride: ["Amaryl", "Dimirel"]
  empagliflozin: ["Jardiance"]
  telmisartan: ["Micardis"]
  amlodipine: ["Norvasc", "Amlodipine Sandoz"]
```

```yaml
# backend/shared-infrastructure/market-configs/australia/food_db_config.yaml

database_id: "AUSNUT_2011_13"
database_name: "Australian Food and Nutrient Database 2011-13"
source_authority: "FSANZ"
sodium_estimation_method: "AUSNUT_DIRECT"
regional_food_databases:
  - "AUSNUT_2011_13"
  - "NUTTAB_2010"
default_portion_size_g: 150
meal_pattern: "3_MAIN_2_SNACK"
```

```yaml
# backend/shared-infrastructure/market-configs/australia/compliance.yaml

health_data_framework: "MY_HEALTH_RECORDS_ACT"
data_residency: "AU"
consent_model: "OPT_OUT_MY_HEALTH_RECORD"
identifier_system: "IHI"       # Individual Healthcare Identifier
audit_standard: "PRIVACY_ACT_1988"
data_retention_years: 30
encryption_standard: "AES_256"
```

```yaml
# backend/shared-infrastructure/market-configs/australia/indigenous_overrides.yaml

# NACCHO guidelines for Aboriginal and Torres Strait Islander populations.
# Applied as an additional override layer on top of Australia base config.

hba1c_target: 8.0              # Relaxed per NACCHO for remote communities
sbp_target: 140.0              # Maintain standard target
egfr_adjustment: true           # Use CKD-EPI without race coefficient
waist_risk_male_cm: 90.0       # Adjusted for indigenous populations
waist_risk_female_cm: 80.0
telehealth_priority: true       # Remote community preference
language_support:
  - "en"
  - "kriol"                     # Australian Kriol
```

- [ ] **Step 91: Create Go module for config loader**

```
# backend/shared-infrastructure/market-configs/loader/go.mod
module market-configs/loader

go 1.21

require (
    github.com/stretchr/testify v1.9.0
    gopkg.in/yaml.v3 v3.0.1
)
```

```bash
cd backend/shared-infrastructure/market-configs/loader
go mod tidy
```

- [ ] **Step 92: Implement config loader types and Load function**

```go
// backend/shared-infrastructure/market-configs/loader/config.go
package marketconfig

import (
    "fmt"
    "os"
    "path/filepath"

    "gopkg.in/yaml.v3"
)

// MarketConfig holds all market-specific configuration.
type MarketConfig struct {
    Market             string              `yaml:"-" json:"market"`
    ClinicalParams     ClinicalParams      `yaml:"-" json:"clinical_params"`
    Channels           ChannelConfig        `yaml:"-" json:"channels"`
    PharmaShim         PharmaConfig         `yaml:"-" json:"pharma_shim"`
    FoodDB             FoodDBConfig         `yaml:"-" json:"food_db"`
    Compliance         ComplianceConfig     `yaml:"-" json:"compliance"`
    IndigenousOverrides *IndigenousOverrides `yaml:"-" json:"indigenous_overrides,omitempty"`
}

// ClinicalParams holds clinical thresholds and targets.
type ClinicalParams struct {
    // Glycaemic
    HbA1cTarget              float64 `yaml:"hba1c_target" json:"hba1c_target"`
    HbA1cTargetElderly       float64 `yaml:"hba1c_target_elderly" json:"hba1c_target_elderly"`
    FPGTarget                float64 `yaml:"fpg_target_mg_dl" json:"fpg_target_mg_dl"`
    PPGTarget                float64 `yaml:"ppg_target_mg_dl" json:"ppg_target_mg_dl"`
    HypoThresholdMgDL        float64 `yaml:"hypo_threshold_mg_dl" json:"hypo_threshold_mg_dl"`
    SevereHypoThresholdMgDL  float64 `yaml:"severe_hypo_threshold_mg_dl" json:"severe_hypo_threshold_mg_dl"`
    HyperThresholdMgDL       float64 `yaml:"hyper_threshold_mg_dl" json:"hyper_threshold_mg_dl"`
    // Hemodynamic
    SBPTarget                float64 `yaml:"sbp_target" json:"sbp_target"`
    SBPTargetDM              float64 `yaml:"sbp_target_dm" json:"sbp_target_dm,omitempty"`
    DBPTarget                float64 `yaml:"dbp_target" json:"dbp_target"`
    SBPCrisis                float64 `yaml:"sbp_crisis" json:"sbp_crisis"`
    DBPCrisis                float64 `yaml:"dbp_crisis" json:"dbp_crisis"`
    // Renal
    EGFRMildThreshold        float64 `yaml:"egfr_mild_threshold" json:"egfr_mild_threshold"`
    EGFRModerateThreshold    float64 `yaml:"egfr_moderate_threshold" json:"egfr_moderate_threshold"`
    EGFRSevereThreshold      float64 `yaml:"egfr_severe_threshold" json:"egfr_severe_threshold"`
    UACRMicroalbuminuria     float64 `yaml:"uacr_microalbuminuria" json:"uacr_microalbuminuria"`
    UACRMacroalbuminuria     float64 `yaml:"uacr_macroalbuminuria" json:"uacr_macroalbuminuria"`
    PotassiumLow             float64 `yaml:"potassium_low" json:"potassium_low"`
    PotassiumHigh            float64 `yaml:"potassium_high" json:"potassium_high"`
    // Metabolic
    BMIOverweight            float64 `yaml:"bmi_overweight" json:"bmi_overweight"`
    BMIObese                 float64 `yaml:"bmi_obese" json:"bmi_obese"`
    WaistRiskMaleCm          float64 `yaml:"waist_risk_male_cm" json:"waist_risk_male_cm,omitempty"`
    WaistRiskFemaleCm        float64 `yaml:"waist_risk_female_cm" json:"waist_risk_female_cm,omitempty"`
    LDLTarget                float64 `yaml:"ldl_target" json:"ldl_target"`
    TGHDLRatioRisk           float64 `yaml:"tg_hdl_ratio_risk" json:"tg_hdl_ratio_risk"`
}

// ChannelConfig holds notification channel configuration.
type ChannelConfig struct {
    EnabledChannels []string               `yaml:"enabled_channels" json:"enabled_channels"`
    WhatsApp        map[string]interface{} `yaml:"whatsapp,omitempty" json:"whatsapp,omitempty"`
    SMS             map[string]interface{} `yaml:"sms,omitempty" json:"sms,omitempty"`
    Email           map[string]interface{} `yaml:"email,omitempty" json:"email,omitempty"`
    PushNotification map[string]interface{} `yaml:"push_notification,omitempty" json:"push_notification,omitempty"`
    IVR             map[string]interface{} `yaml:"ivr,omitempty" json:"ivr,omitempty"`
    MyHealthRecord  map[string]interface{} `yaml:"my_health_record,omitempty" json:"my_health_record,omitempty"`
}

// PharmaConfig holds pharmacy/formulary configuration.
type PharmaConfig struct {
    FormularySource    string              `yaml:"formulary_source" json:"formulary_source"`
    Currency           string              `yaml:"currency" json:"currency"`
    PrescriptionFormat string              `yaml:"prescription_format" json:"prescription_format"`
    CommonBrands       map[string][]string `yaml:"common_brands" json:"common_brands"`
}

// FoodDBConfig holds food database configuration.
type FoodDBConfig struct {
    DatabaseID             string   `yaml:"database_id" json:"database_id"`
    DatabaseName           string   `yaml:"database_name" json:"database_name"`
    SourceAuthority        string   `yaml:"source_authority" json:"source_authority"`
    SodiumEstimationMethod string   `yaml:"sodium_estimation_method" json:"sodium_estimation_method"`
    RegionalFoodDatabases  []string `yaml:"regional_food_databases" json:"regional_food_databases"`
    DefaultPortionSizeG    int      `yaml:"default_portion_size_g" json:"default_portion_size_g"`
    MealPattern            string   `yaml:"meal_pattern" json:"meal_pattern"`
}

// ComplianceConfig holds regulatory compliance configuration.
type ComplianceConfig struct {
    HealthDataFramework string `yaml:"health_data_framework" json:"health_data_framework"`
    DataResidency       string `yaml:"data_residency" json:"data_residency"`
    ConsentModel        string `yaml:"consent_model" json:"consent_model"`
    IdentifierSystem    string `yaml:"identifier_system" json:"identifier_system"`
    AuditStandard       string `yaml:"audit_standard" json:"audit_standard"`
    DataRetentionYears  int    `yaml:"data_retention_years" json:"data_retention_years"`
    EncryptionStandard  string `yaml:"encryption_standard" json:"encryption_standard"`
}

// IndigenousOverrides holds population-specific overrides (e.g., NACCHO for Australia).
type IndigenousOverrides struct {
    HbA1cTarget         float64  `yaml:"hba1c_target" json:"hba1c_target"`
    SBPTarget           float64  `yaml:"sbp_target" json:"sbp_target"`
    EGFRAdjustment      bool     `yaml:"egfr_adjustment" json:"egfr_adjustment"`
    WaistRiskMaleCm     float64  `yaml:"waist_risk_male_cm" json:"waist_risk_male_cm"`
    WaistRiskFemaleCm   float64  `yaml:"waist_risk_female_cm" json:"waist_risk_female_cm"`
    TelehealthPriority  bool     `yaml:"telehealth_priority" json:"telehealth_priority"`
    LanguageSupport     []string `yaml:"language_support" json:"language_support"`
}

// Load reads and merges market-specific YAML over shared base.
// Merge order: shared/base_clinical_params.yaml → <market>/clinical_params.yaml.
// Other configs (channels, pharma, food_db, compliance) are market-specific only.
// Indigenous overrides are loaded if the file exists.
func Load(market string, configDir string) (*MarketConfig, error) {
    marketDir := filepath.Join(configDir, market)
    if _, err := os.Stat(marketDir); os.IsNotExist(err) {
        return nil, fmt.Errorf("market directory not found: %s", marketDir)
    }

    cfg := &MarketConfig{Market: market}

    // 1. Load shared base clinical params
    basePath := filepath.Join(configDir, "shared", "base_clinical_params.yaml")
    if err := loadClinicalParams(basePath, &cfg.ClinicalParams); err != nil {
        return nil, fmt.Errorf("loading shared base: %w", err)
    }

    // 2. Merge market-specific clinical params over base
    marketClinicalPath := filepath.Join(marketDir, "clinical_params.yaml")
    if err := loadClinicalParams(marketClinicalPath, &cfg.ClinicalParams); err != nil {
        return nil, fmt.Errorf("loading market clinical params: %w", err)
    }

    // 3. Load market-specific configs (no base merging)
    if err := loadYAML(filepath.Join(marketDir, "channels.yaml"), &cfg.Channels); err != nil {
        return nil, fmt.Errorf("loading channels: %w", err)
    }
    if err := loadYAML(filepath.Join(marketDir, "pharma_shim.yaml"), &cfg.PharmaShim); err != nil {
        return nil, fmt.Errorf("loading pharma shim: %w", err)
    }
    if err := loadYAML(filepath.Join(marketDir, "food_db_config.yaml"), &cfg.FoodDB); err != nil {
        return nil, fmt.Errorf("loading food DB: %w", err)
    }
    if err := loadYAML(filepath.Join(marketDir, "compliance.yaml"), &cfg.Compliance); err != nil {
        return nil, fmt.Errorf("loading compliance: %w", err)
    }

    // 4. Load indigenous overrides if file exists
    indigenousPath := filepath.Join(marketDir, "indigenous_overrides.yaml")
    if _, err := os.Stat(indigenousPath); err == nil {
        var overrides IndigenousOverrides
        if err := loadYAML(indigenousPath, &overrides); err != nil {
            return nil, fmt.Errorf("loading indigenous overrides: %w", err)
        }
        cfg.IndigenousOverrides = &overrides
    }

    return cfg, nil
}

// loadClinicalParams loads a clinical params YAML into the struct.
// The YAML has nested keys (glycaemic.hba1c_target, hemodynamic.sbp_target, etc.)
// which are flattened into the ClinicalParams struct.
func loadClinicalParams(path string, params *ClinicalParams) error {
    data, err := os.ReadFile(path)
    if err != nil {
        return err
    }
    var raw map[string]map[string]interface{}
    if err := yaml.Unmarshal(data, &raw); err != nil {
        return err
    }

    // Flatten nested sections into ClinicalParams fields
    if g, ok := raw["glycaemic"]; ok {
        setFloat(g, "hba1c_target", &params.HbA1cTarget)
        setFloat(g, "hba1c_target_elderly", &params.HbA1cTargetElderly)
        setFloat(g, "fpg_target_mg_dl", &params.FPGTarget)
        setFloat(g, "ppg_target_mg_dl", &params.PPGTarget)
        setFloat(g, "hypo_threshold_mg_dl", &params.HypoThresholdMgDL)
        setFloat(g, "severe_hypo_threshold_mg_dl", &params.SevereHypoThresholdMgDL)
        setFloat(g, "hyper_threshold_mg_dl", &params.HyperThresholdMgDL)
    }
    if h, ok := raw["hemodynamic"]; ok {
        setFloat(h, "sbp_target", &params.SBPTarget)
        setFloat(h, "sbp_target_dm", &params.SBPTargetDM)
        setFloat(h, "dbp_target", &params.DBPTarget)
        setFloat(h, "sbp_crisis", &params.SBPCrisis)
        setFloat(h, "dbp_crisis", &params.DBPCrisis)
    }
    if r, ok := raw["renal"]; ok {
        setFloat(r, "egfr_mild_threshold", &params.EGFRMildThreshold)
        setFloat(r, "egfr_moderate_threshold", &params.EGFRModerateThreshold)
        setFloat(r, "egfr_severe_threshold", &params.EGFRSevereThreshold)
        setFloat(r, "uacr_microalbuminuria", &params.UACRMicroalbuminuria)
        setFloat(r, "uacr_macroalbuminuria", &params.UACRMacroalbuminuria)
        setFloat(r, "potassium_low", &params.PotassiumLow)
        setFloat(r, "potassium_high", &params.PotassiumHigh)
    }
    if m, ok := raw["metabolic"]; ok {
        setFloat(m, "bmi_overweight", &params.BMIOverweight)
        setFloat(m, "bmi_obese", &params.BMIObese)
        setFloat(m, "waist_risk_male_cm", &params.WaistRiskMaleCm)
        setFloat(m, "waist_risk_female_cm", &params.WaistRiskFemaleCm)
        setFloat(m, "ldl_target", &params.LDLTarget)
        setFloat(m, "tg_hdl_ratio_risk", &params.TGHDLRatioRisk)
    }
    return nil
}

// setFloat sets the target if the key exists and is non-zero.
func setFloat(m map[string]interface{}, key string, target *float64) {
    if v, ok := m[key]; ok {
        switch val := v.(type) {
        case float64:
            if val != 0 {
                *target = val
            }
        case int:
            if val != 0 {
                *target = float64(val)
            }
        }
    }
}

// loadYAML reads a YAML file into the target struct.
func loadYAML(path string, target interface{}) error {
    data, err := os.ReadFile(path)
    if err != nil {
        return err
    }
    return yaml.Unmarshal(data, target)
}
```

- [ ] **Step 93: Run tests to verify they pass**

```bash
cd backend/shared-infrastructure/market-configs/loader
go test ./... -v
```

Expected: All 6 tests PASS.

- [ ] **Step 94: Wire market config into KB-20 main.go**

In `kb-20-patient-profile/cmd/server/main.go`, add market config loading at startup. Add after existing config loading:

```go
import marketconfig "market-configs/loader"

// In main() or init(), after database setup:
market := os.Getenv("MARKET")
if market == "" {
    market = "india" // default market
}
configDir := os.Getenv("MARKET_CONFIG_DIR")
if configDir == "" {
    configDir = "../../../market-configs"
}
marketCfg, err := marketconfig.Load(market, configDir)
if err != nil {
    log.Fatalf("Failed to load market config: %v", err)
}
log.Printf("Loaded market config for %s", marketCfg.Market)

// Pass marketCfg to server for threshold lookups
server.MarketConfig = marketCfg
```

- [ ] **Step 95: Wire market config into KB-23 and KB-26 main.go**

Apply the same pattern from Step 94 to:

1. `kb-23-decision-cards/cmd/server/main.go` — uses `ClinicalParams` for card recommendation thresholds and `Channels` for delivery routing.

2. `kb-26-metabolic-digital-twin/cmd/server/main.go` — uses `ClinicalParams` for MHRI normalization ranges and `FoodDB` for sodium estimation method.

Both follow the identical pattern: read `MARKET` env var, load config, assign to server struct.

- [ ] **Step 96: Run full regression across KB-20, KB-23, KB-26**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile
go test ./... -v -count=1

cd ../kb-23-decision-cards
go test ./... -v -count=1

cd ../kb-26-metabolic-digital-twin
go test ./... -v -count=1
```

Expected: All existing tests PASS (market config is optional — services fall back to hardcoded defaults if `MARKET` env var is unset).

- [ ] **Step 97: Commit Task 15**

```bash
git add backend/shared-infrastructure/market-configs/
git commit -m "feat(market-shim): market configuration infrastructure for India and Australia

Shared base clinical params + market-specific overrides for clinical
thresholds (RSSDI/ISH for India, RACGP/Heart Foundation for Australia),
notification channels, pharma formularies, food databases, and
compliance frameworks. NACCHO indigenous overrides for Australia.
Go config loader with merge precedence: shared → market → population.
Wired into KB-20, KB-23, KB-26 startup."
```

---

---

## Plan Summary

| Phase | Tasks | Steps | Key Deliverables |
|-------|-------|-------|-----------------|
| C0 | 3b, 3c | 1–8 | V4 contract audit, Flink IngestionEventData V4 fields |
| C2 | 6, 6b | 9–22 | KB-20 V4 state API, Flink domain classification + BP trajectory |
| C3 | 7, 8 | 23–44 | MHRI trajectory engine, IOR system |
| C4 | 11, 12 | 45–67 | Dual-domain cards, physician feedback capture |
| C5 | 13, 14 | 68–82 | Phenotype clustering pipeline, feedback governance |
| C6 | 15 | 83–97 | Market config YAML + loader + service wiring |

**Total: 97 steps across 11 tasks in 6 phases.**

All steps follow TDD: write test → verify fail → implement → verify pass → commit.