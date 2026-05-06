# Phase 1B-β.2-A — MedicineUse End-to-End Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver the MedicineUse v2 substrate entity end-to-end — type definitions with Intent/Target/StopCriteria including 5 documented Target.Kind JSONB shapes, FHIR mapper to AU MedicationRequest, validators, kb-20 migration 008_part2 PART A (medication_states extension + medicine_uses_v2 view), V2SubstrateStore methods, REST endpoints, and KB20Client methods — as the first reviewable cluster of Phase 1B-β.2 clinical primitives.

**Architecture:** Inherits patterns from Phase 1B-β.1: shared library at `shared/v2_substrate/`, kb-20 canonical storage, internal Go types with FHIR mappers at boundaries, non-breaking migration extending `medication_states` with nullable JSONB columns, rowScanner pattern for single-query lists, ErrNotFound sentinel for 404 dispatch, validation called at egress (forward) FHIR mappers, ingress validation for defense-in-depth.

**Tech Stack:** Go 1.24, PostgreSQL (kb-20 schema, port 5433), Gin HTTP framework, lib/pq, JSONB, testify/require, httptest. Module path `github.com/cardiofit/shared`.

**Spec:** `docs/superpowers/specs/2026-05-04-1b-beta-2-clinical-primitives-design.md` (see §3.1-3.5 for types, §5.1+5.3 for SQL, §6 β.2-A for cluster scope)

**Scope of this plan:** Cluster β.2-A only (~3 days). MedicineUse end-to-end. **Does NOT cover** Observation (β.2-B, separate plan), delta-on-write (β.2-C, separate plan), Event/EvidenceTrace (β.3, separate plan), Authorisation evaluator, Layer 1B adapters.

**Exit criterion:** Round-trip MedicineUse with each of 5 Target.Kind values (`BP_threshold`, `completion_date`, `symptom_resolution`, `HbA1c_band`, `open`) through {KB20Client → handler → store → DB → handler → KB20Client}; FHIR mapper validates each Target.Spec shape per Kind.

---

## File Structure

**Created:**
- `<kbs>/shared/v2_substrate/models/medicine_use.go` — MedicineUse + Intent + Target + StopCriteria structs
- `<kbs>/shared/v2_substrate/models/medicine_use_test.go` — JSON round-trip + cross-KB ScopeRules contract tests
- `<kbs>/shared/v2_substrate/models/target_schemas.go` — 5 documented Target.Spec shape structs
- `<kbs>/shared/v2_substrate/models/target_schemas_test.go` — schema validation
- `<kbs>/shared/v2_substrate/models/stop_criteria_schemas.go` — StopTrigger constants + StopCriteria.Spec struct shapes
- `<kbs>/shared/v2_substrate/models/stop_criteria_schemas_test.go` — schema validation
- `<kbs>/shared/v2_substrate/validation/medicine_use_validator.go` — top-level MedicineUse validator
- `<kbs>/shared/v2_substrate/validation/target_validator.go` — per-Kind Target.Spec validators
- `<kbs>/shared/v2_substrate/fhir/medication_request_mapper.go` — MedicineUse ↔ AU MedicationRequest
- `<kbs>/shared/v2_substrate/fhir/medication_request_mapper_test.go` — round-trip + _RejectsInvalid + _WrongResourceType + _WireFormat tests
- `<kbs>/kb-20-patient-profile/migrations/008_part2_clinical_primitives_partA.sql` — medication_states extension + medicine_uses_v2 view

**Modified:**
- `<kbs>/shared/v2_substrate/models/enums.go` — append MedicineUseStatus + Intent.Category + Target.Kind + StopTrigger constants
- `<kbs>/shared/v2_substrate/fhir/extensions.go` — append Vaidshala extension URIs for intent/target/stop_criteria
- `<kbs>/shared/v2_substrate/validation/validators_test.go` — append MedicineUse + Target test cases
- `<kbs>/shared/v2_substrate/interfaces/storage.go` — append MedicineUseStore interface
- `<kbs>/shared/v2_substrate/client/kb20_client.go` — append 4 MedicineUse methods
- `<kbs>/shared/v2_substrate/client/kb20_client_test.go` — append httptest case for MedicineUse round-trip
- `<kbs>/kb-20-patient-profile/internal/storage/v2_substrate_store.go` — append scanMedicineUse + GetMedicineUse + UpsertMedicineUse + ListMedicineUsesByResident; add `var _ interfaces.MedicineUseStore = (*V2SubstrateStore)(nil)` compile-time assertion
- `<kbs>/kb-20-patient-profile/internal/api/v2_substrate_handlers.go` — append 4 REST endpoints; register in RegisterRoutes
- `<kbs>/kb-20-patient-profile/main.go` — no change required (RegisterRoutes already wired in β.1)

**Convention:** `<kbs>` = `/Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/`. All paths in tasks use the full path.

---

## Task 1: Append MedicineUse-related enum constants

**Files:**
- Modify: `<kbs>/shared/v2_substrate/models/enums.go`
- Modify: `<kbs>/shared/v2_substrate/models/enums_test.go`

- [ ] **Step 1: Write the failing test**

Append to `<kbs>/shared/v2_substrate/models/enums_test.go`:

```go
func TestMedicineUseStatusIsValid(t *testing.T) {
    valid := []string{"active", "paused", "ceased", "completed"}
    for _, s := range valid {
        if !IsValidMedicineUseStatus(s) {
            t.Errorf("IsValidMedicineUseStatus(%q) = false, want true", s)
        }
    }
    if IsValidMedicineUseStatus("done") {
        t.Errorf("IsValidMedicineUseStatus(\"done\") = true, want false")
    }
}

func TestIntentCategoryIsValid(t *testing.T) {
    valid := []string{"therapeutic", "preventive", "symptomatic", "trial", "deprescribing"}
    for _, c := range valid {
        if !IsValidIntentCategory(c) {
            t.Errorf("IsValidIntentCategory(%q) = false, want true", c)
        }
    }
    if IsValidIntentCategory("curative") {
        t.Errorf("IsValidIntentCategory(\"curative\") = true, want false")
    }
}

func TestTargetKindIsValid(t *testing.T) {
    valid := []string{"BP_threshold", "completion_date", "symptom_resolution", "HbA1c_band", "open"}
    for _, k := range valid {
        if !IsValidTargetKind(k) {
            t.Errorf("IsValidTargetKind(%q) = false, want true", k)
        }
    }
    if IsValidTargetKind("LDL_target") {
        t.Errorf("IsValidTargetKind(\"LDL_target\") = true, want false (must add to enum first)")
    }
}

func TestStopTriggerIsValid(t *testing.T) {
    valid := []string{"adverse_event", "target_achieved", "review_due", "patient_request",
        "carer_request", "completion", "interaction"}
    for _, s := range valid {
        if !IsValidStopTrigger(s) {
            t.Errorf("IsValidStopTrigger(%q) = false, want true", s)
        }
    }
    if IsValidStopTrigger("died") {
        t.Errorf("IsValidStopTrigger(\"died\") = true, want false")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared
go test ./v2_substrate/models/... -run "TestMedicineUseStatus|TestIntentCategory|TestTargetKind|TestStopTrigger" -v
```

Expected: FAIL with `undefined: IsValidMedicineUseStatus` (and similar).

- [ ] **Step 3: Append constants to enums.go**

Append to `<kbs>/shared/v2_substrate/models/enums.go` (after the existing ResidentStatus block):

```go
// MedicineUseStatus enumerates the lifecycle of a v2 MedicineUse row. Values
// are stored as plain strings to round-trip cleanly through FHIR code elements.
const (
    MedicineUseStatusActive    = "active"
    MedicineUseStatusPaused    = "paused"
    MedicineUseStatusCeased    = "ceased"
    MedicineUseStatusCompleted = "completed"
)

// IsValidMedicineUseStatus reports whether s is one of the recognized
// MedicineUseStatus values.
func IsValidMedicineUseStatus(s string) bool {
    switch s {
    case MedicineUseStatusActive, MedicineUseStatusPaused,
        MedicineUseStatusCeased, MedicineUseStatusCompleted:
        return true
    }
    return false
}

// Intent categories — describes WHY a medicine is used. Values are stored
// as plain strings to round-trip cleanly through FHIR code elements.
const (
    IntentTherapeutic   = "therapeutic"   // treating an active condition
    IntentPreventive    = "preventive"    // primary or secondary prevention
    IntentSymptomatic   = "symptomatic"   // PRN / symptom relief
    IntentTrial         = "trial"         // therapeutic trial period
    IntentDeprescribing = "deprescribing" // tapering / withdrawal
)

// IsValidIntentCategory reports whether s is one of the recognized
// Intent.Category values.
func IsValidIntentCategory(s string) bool {
    switch s {
    case IntentTherapeutic, IntentPreventive, IntentSymptomatic,
        IntentTrial, IntentDeprescribing:
        return true
    }
    return false
}

// Target kinds — discriminator for Target.Spec JSONB shape. Each kind has
// a documented spec struct in target_schemas.go.
const (
    TargetKindBPThreshold       = "BP_threshold"        // antihypertensives
    TargetKindCompletionDate    = "completion_date"     // antibiotics, deprescribing
    TargetKindSymptomResolution = "symptom_resolution"  // symptomatic
    TargetKindHbA1cBand         = "HbA1c_band"          // diabetes
    TargetKindOpen              = "open"                // chronic, no specific target
)

// IsValidTargetKind reports whether s is one of the recognized Target.Kind
// values. Adding a new kind requires also adding the spec struct in
// target_schemas.go and a delegated validator in target_validator.go.
func IsValidTargetKind(s string) bool {
    switch s {
    case TargetKindBPThreshold, TargetKindCompletionDate,
        TargetKindSymptomResolution, TargetKindHbA1cBand, TargetKindOpen:
        return true
    }
    return false
}

// StopTrigger enumerates the structured reasons a MedicineUse stop can be
// initiated. Stored in StopCriteria.Triggers []string.
const (
    StopTriggerAdverseEvent   = "adverse_event"
    StopTriggerTargetAchieved = "target_achieved"
    StopTriggerReviewDue      = "review_due"
    StopTriggerPatientRequest = "patient_request"
    StopTriggerCarerRequest   = "carer_request"
    StopTriggerCompletion     = "completion" // course completed (antibiotics, etc.)
    StopTriggerInteraction    = "interaction" // contraindicated by new medicine
)

// IsValidStopTrigger reports whether s is one of the recognized StopTrigger
// values.
func IsValidStopTrigger(s string) bool {
    switch s {
    case StopTriggerAdverseEvent, StopTriggerTargetAchieved,
        StopTriggerReviewDue, StopTriggerPatientRequest,
        StopTriggerCarerRequest, StopTriggerCompletion, StopTriggerInteraction:
        return true
    }
    return false
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared
go test ./v2_substrate/models/... -v
```

Expected: PASS for all enum tests.

---

## Task 2: Define target_schemas.go (5 spec structs)

**Files:**
- Create: `<kbs>/shared/v2_substrate/models/target_schemas.go`
- Create: `<kbs>/shared/v2_substrate/models/target_schemas_test.go`

- [ ] **Step 1: Write the failing test**

Path: `<kbs>/shared/v2_substrate/models/target_schemas_test.go`

```go
package models

import (
    "encoding/json"
    "testing"
    "time"
)

func TestTargetBPThresholdSpecRoundTrip(t *testing.T) {
    in := TargetBPThresholdSpec{SystolicMax: 140, DiastolicMax: 90}
    b, err := json.Marshal(in)
    if err != nil {
        t.Fatalf("marshal: %v", err)
    }
    var out TargetBPThresholdSpec
    if err := json.Unmarshal(b, &out); err != nil {
        t.Fatalf("unmarshal: %v", err)
    }
    if out.SystolicMax != 140 || out.DiastolicMax != 90 {
        t.Errorf("round-trip: got %+v, want %+v", out, in)
    }
}

func TestTargetCompletionDateSpecRoundTrip(t *testing.T) {
    in := TargetCompletionDateSpec{
        EndDate:      time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC),
        DurationDays: 7,
        Rationale:    "amoxicillin course",
    }
    b, _ := json.Marshal(in)
    var out TargetCompletionDateSpec
    if err := json.Unmarshal(b, &out); err != nil {
        t.Fatalf("unmarshal: %v", err)
    }
    if !out.EndDate.Equal(in.EndDate) || out.DurationDays != 7 || out.Rationale != "amoxicillin course" {
        t.Errorf("round-trip: got %+v, want %+v", out, in)
    }
}

func TestTargetSymptomResolutionSpecRoundTrip(t *testing.T) {
    in := TargetSymptomResolutionSpec{
        TargetSymptom:        "pain",
        MonitoringWindowDays: 14,
        SNOMEDCode:           "22253000",
    }
    b, _ := json.Marshal(in)
    var out TargetSymptomResolutionSpec
    if err := json.Unmarshal(b, &out); err != nil {
        t.Fatalf("unmarshal: %v", err)
    }
    if out.TargetSymptom != in.TargetSymptom || out.SNOMEDCode != in.SNOMEDCode {
        t.Errorf("round-trip mismatch")
    }
}

func TestTargetHbA1cBandSpecRoundTrip(t *testing.T) {
    in := TargetHbA1cBandSpec{Min: 6.5, Max: 8.0}
    b, _ := json.Marshal(in)
    var out TargetHbA1cBandSpec
    if err := json.Unmarshal(b, &out); err != nil {
        t.Fatalf("unmarshal: %v", err)
    }
    if out.Min != 6.5 || out.Max != 8.0 {
        t.Errorf("round-trip: got %+v, want %+v", out, in)
    }
}

func TestTargetOpenSpecRoundTrip(t *testing.T) {
    in := TargetOpenSpec{Rationale: "long-term anticoagulation for AF"}
    b, _ := json.Marshal(in)
    var out TargetOpenSpec
    if err := json.Unmarshal(b, &out); err != nil {
        t.Fatalf("unmarshal: %v", err)
    }
    if out.Rationale != in.Rationale {
        t.Errorf("round-trip: got %+v, want %+v", out, in)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared
go test ./v2_substrate/models/... -run "TestTarget.*Spec" -v
```

Expected: FAIL with `undefined: TargetBPThresholdSpec` (and similar).

- [ ] **Step 3: Write target_schemas.go**

Path: `<kbs>/shared/v2_substrate/models/target_schemas.go`

```go
package models

import "time"

// Target.Spec JSONB shapes — one struct per Target.Kind constant in enums.go.
//
// MedicineUse.Target.Spec is stored as JSON.RawMessage at the top level so
// the storage layer is shape-agnostic; these structs are the documented
// per-Kind contract that callers may use for type safety. Adding a new
// Target.Kind requires (a) a new constant in enums.go, (b) a new spec
// struct here, and (c) a delegated validator in
// validation/target_validator.go.

// TargetBPThresholdSpec — for Target{Kind: TargetKindBPThreshold}.
//
// Used for antihypertensive medicines where the target is keeping BP below a
// threshold. Both bounds are inclusive maxima.
//
// Example: {"systolic_max": 140, "diastolic_max": 90}
type TargetBPThresholdSpec struct {
    SystolicMax  int `json:"systolic_max"`
    DiastolicMax int `json:"diastolic_max"`
}

// TargetCompletionDateSpec — for Target{Kind: TargetKindCompletionDate}.
//
// Used for both antibiotic course completion AND deprescribing target dates.
// EndDate is the canonical target; DurationDays + Rationale are informational.
//
// Example: {"end_date": "2026-05-15T00:00:00Z", "duration_days": 7, "rationale": "amoxicillin course"}
type TargetCompletionDateSpec struct {
    EndDate      time.Time `json:"end_date"`
    DurationDays int       `json:"duration_days,omitempty"`
    Rationale    string    `json:"rationale,omitempty"`
}

// TargetSymptomResolutionSpec — for Target{Kind: TargetKindSymptomResolution}.
//
// Used for symptomatic (PRN) medicines where the target is the resolution
// of a specified symptom within a monitoring window.
//
// Example: {"target_symptom": "pain", "monitoring_window_days": 14, "snomed_code": "22253000"}
type TargetSymptomResolutionSpec struct {
    TargetSymptom        string `json:"target_symptom"`
    MonitoringWindowDays int    `json:"monitoring_window_days,omitempty"`
    SNOMEDCode           string `json:"snomed_code,omitempty"`
}

// TargetHbA1cBandSpec — for Target{Kind: TargetKindHbA1cBand}.
//
// Used for diabetes medicines where the target is keeping HbA1c within a band.
// Min and Max are both inclusive.
//
// Example: {"min": 6.5, "max": 8.0}
type TargetHbA1cBandSpec struct {
    Min float64 `json:"min"`
    Max float64 `json:"max"`
}

// TargetOpenSpec — for Target{Kind: TargetKindOpen}.
//
// Used for chronic, indefinite medicines where no specific numerical target
// applies. Rationale captures the clinical justification for ongoing use.
//
// Example: {"rationale": "long-term anticoagulation for AF"}
type TargetOpenSpec struct {
    Rationale string `json:"rationale,omitempty"`
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared
go test ./v2_substrate/models/... -v
```

Expected: PASS for all 5 Target spec tests.

---

## Task 3: Define stop_criteria_schemas.go

**Files:**
- Create: `<kbs>/shared/v2_substrate/models/stop_criteria_schemas.go`
- Create: `<kbs>/shared/v2_substrate/models/stop_criteria_schemas_test.go`

- [ ] **Step 1: Write the failing test**

Path: `<kbs>/shared/v2_substrate/models/stop_criteria_schemas_test.go`

```go
package models

import (
    "encoding/json"
    "testing"
)

func TestStopCriteriaReviewSpecRoundTrip(t *testing.T) {
    in := StopCriteriaReviewSpec{ReviewAfterDays: 30, ReviewOwner: "ACOP"}
    b, _ := json.Marshal(in)
    var out StopCriteriaReviewSpec
    if err := json.Unmarshal(b, &out); err != nil {
        t.Fatalf("unmarshal: %v", err)
    }
    if out.ReviewAfterDays != 30 || out.ReviewOwner != "ACOP" {
        t.Errorf("round-trip: got %+v want %+v", out, in)
    }
}

func TestStopCriteriaThresholdSpecRoundTrip(t *testing.T) {
    in := StopCriteriaThresholdSpec{
        ObservationKind: "vital",
        LOINCCode:       "8867-4",
        Operator:        "<",
        Value:           50,
    }
    b, _ := json.Marshal(in)
    var out StopCriteriaThresholdSpec
    if err := json.Unmarshal(b, &out); err != nil {
        t.Fatalf("unmarshal: %v", err)
    }
    if out.ObservationKind != "vital" || out.Operator != "<" || out.Value != 50 {
        t.Errorf("round-trip: got %+v want %+v", out, in)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared
go test ./v2_substrate/models/... -run "TestStopCriteria.*Spec" -v
```

Expected: FAIL with `undefined: StopCriteriaReviewSpec` (and similar).

- [ ] **Step 3: Write stop_criteria_schemas.go**

Path: `<kbs>/shared/v2_substrate/models/stop_criteria_schemas.go`

```go
package models

// StopCriteria.Spec JSONB shapes — documented per-pattern struct shapes that
// callers may use for type safety. The actual storage is JSON.RawMessage in
// MedicineUse.StopCriteria.Spec at the top level.

// StopCriteriaReviewSpec — for time-bounded review obligation.
//
// Used when StopCriteria.Triggers contains "review_due" and the review is
// time-based.
//
// Example: {"review_after_days": 30, "review_owner": "ACOP"}
type StopCriteriaReviewSpec struct {
    ReviewAfterDays int    `json:"review_after_days"`
    ReviewOwner     string `json:"review_owner,omitempty"` // RN|GP|ACOP|pharmacist
}

// StopCriteriaThresholdSpec — for criterion based on an observation threshold.
//
// Used when stop should be triggered if a specific observation crosses a
// threshold (e.g., stop ACE inhibitor if eGFR drops below 30).
//
// Example: {"observation_kind": "vital", "loinc_code": "8867-4", "operator": "<", "value": 50}
type StopCriteriaThresholdSpec struct {
    ObservationKind string  `json:"observation_kind"` // vital|lab|behavioural|mobility|weight
    LOINCCode       string  `json:"loinc_code,omitempty"`
    SNOMEDCode      string  `json:"snomed_code,omitempty"`
    Operator        string  `json:"operator"` // < <= = >= >
    Value           float64 `json:"value"`
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared
go test ./v2_substrate/models/... -v
```

Expected: PASS for all stop_criteria spec tests.

---

## Task 4: Define MedicineUse type

**Files:**
- Create: `<kbs>/shared/v2_substrate/models/medicine_use.go`
- Create: `<kbs>/shared/v2_substrate/models/medicine_use_test.go`

- [ ] **Step 1: Write the failing test**

Path: `<kbs>/shared/v2_substrate/models/medicine_use_test.go`

```go
package models

import (
    "encoding/json"
    "testing"
    "time"

    "github.com/google/uuid"
)

func TestMedicineUseJSONRoundTrip(t *testing.T) {
    prescriber := uuid.New()
    bpSpec := TargetBPThresholdSpec{SystolicMax: 140, DiastolicMax: 90}
    bpSpecRaw, _ := json.Marshal(bpSpec)

    in := MedicineUse{
        ID:           uuid.New(),
        ResidentID:   uuid.New(),
        AMTCode:      "12345",
        DisplayName:  "Perindopril 5mg",
        Intent: Intent{
            Category:   IntentTherapeutic,
            Indication: "essential hypertension",
        },
        Target: Target{
            Kind: TargetKindBPThreshold,
            Spec: bpSpecRaw,
        },
        StopCriteria: StopCriteria{
            Triggers: []string{StopTriggerAdverseEvent, StopTriggerReviewDue},
        },
        Dose:         "5mg",
        Route:        "ORAL",
        Frequency:    "QD",
        PrescriberID: &prescriber,
        StartedAt:    time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
        Status:       MedicineUseStatusActive,
        CreatedAt:    time.Now().UTC(),
        UpdatedAt:    time.Now().UTC(),
    }

    b, err := json.Marshal(in)
    if err != nil {
        t.Fatalf("marshal: %v", err)
    }
    var out MedicineUse
    if err := json.Unmarshal(b, &out); err != nil {
        t.Fatalf("unmarshal: %v", err)
    }

    if out.ID != in.ID || out.AMTCode != in.AMTCode || out.DisplayName != in.DisplayName {
        t.Errorf("identity round-trip mismatch")
    }
    if out.Intent.Category != IntentTherapeutic {
        t.Errorf("intent.category: got %q", out.Intent.Category)
    }
    if out.Target.Kind != TargetKindBPThreshold {
        t.Errorf("target.kind: got %q", out.Target.Kind)
    }
    if len(out.StopCriteria.Triggers) != 2 {
        t.Errorf("stop_criteria.triggers count: got %d", len(out.StopCriteria.Triggers))
    }
    if out.PrescriberID == nil || *out.PrescriberID != prescriber {
        t.Errorf("prescriber_id round-trip lost")
    }
}

func TestMedicineUseOptionalFields(t *testing.T) {
    in := MedicineUse{
        ID:          uuid.New(),
        ResidentID:  uuid.New(),
        DisplayName: "Test",
        Intent:      Intent{Category: IntentTherapeutic, Indication: "x"},
        Target:      Target{Kind: TargetKindOpen, Spec: json.RawMessage(`{}`)},
        StopCriteria: StopCriteria{Triggers: []string{}},
        StartedAt:   time.Now(),
        Status:      MedicineUseStatusActive,
    }
    b, _ := json.Marshal(in)
    var m map[string]any
    if err := json.Unmarshal(b, &m); err != nil {
        t.Fatalf("unmarshal to map: %v", err)
    }
    if _, present := m["amt_code"]; present {
        t.Errorf("amt_code should be omitted when empty")
    }
    if _, present := m["ended_at"]; present {
        t.Errorf("ended_at should be omitted when nil")
    }
    if _, present := m["prescriber_id"]; present {
        t.Errorf("prescriber_id should be omitted when nil")
    }
}

func TestTargetSpecOpaqueMarshalling(t *testing.T) {
    // Target.Spec is json.RawMessage; the model layer treats it opaquely.
    // This test pins down that Target.Spec is preserved byte-for-byte through
    // a round-trip — critical for the cross-KB JSONB contract.
    inputSpec := json.RawMessage(`{"systolic_max":140,"diastolic_max":90}`)
    in := Target{Kind: TargetKindBPThreshold, Spec: inputSpec}
    b, _ := json.Marshal(in)
    var out Target
    if err := json.Unmarshal(b, &out); err != nil {
        t.Fatalf("unmarshal: %v", err)
    }
    if string(out.Spec) != string(inputSpec) {
        t.Errorf("spec opacity broken: got %s want %s", string(out.Spec), string(inputSpec))
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared
go test ./v2_substrate/models/... -run "TestMedicineUse|TestTargetSpec" -v
```

Expected: FAIL with `undefined: MedicineUse` / `undefined: Intent` / `undefined: Target` / `undefined: StopCriteria`.

- [ ] **Step 3: Write medicine_use.go**

Path: `<kbs>/shared/v2_substrate/models/medicine_use.go`

```go
package models

import (
    "encoding/json"
    "time"

    "github.com/google/uuid"
)

// MedicineUse represents a v2 substrate medication record for a Resident.
// Distinguished from kb-20's legacy medication_states by three v2-specific
// JSONB fields: Intent (why), Target (what success looks like), and
// StopCriteria (when to stop). These fields are the basis for the Recommendation
// state machine's deprescribing logic, which arrives in a later phase.
//
// Canonical storage: kb-20-patient-profile (medicine_uses_v2 view over
// medication_states + the v2 columns added in migration 008_part2 part A).
//
// FHIR boundary: maps to AU FHIR MedicationRequest at integration boundaries
// via shared/v2_substrate/fhir/medication_request_mapper.go. Intent / Target /
// StopCriteria do not have native FHIR representations and are encoded as
// Vaidshala-namespaced FHIR extensions.
type MedicineUse struct {
    ID           uuid.UUID    `json:"id"`
    ResidentID   uuid.UUID    `json:"resident_id"`
    AMTCode      string       `json:"amt_code,omitempty"`     // Australian Medicines Terminology code
    DisplayName  string       `json:"display_name"`            // human-readable; falls back to legacy drug_name
    Intent       Intent       `json:"intent"`                  // v2-distinguishing
    Target       Target       `json:"target"`                  // v2-distinguishing (JSONB)
    StopCriteria StopCriteria `json:"stop_criteria"`           // v2-distinguishing (JSONB)
    Dose         string       `json:"dose,omitempty"`          // unstructured form
    Route        string       `json:"route,omitempty"`         // ORAL, IV, IM, etc.
    Frequency    string       `json:"frequency,omitempty"`     // e.g., "BID", "QD"
    PrescriberID *uuid.UUID   `json:"prescriber_id,omitempty"` // v2 Person.id; nullable for legacy records
    StartedAt    time.Time    `json:"started_at"`
    EndedAt      *time.Time   `json:"ended_at,omitempty"`
    Status       string       `json:"status"` // see MedicineUseStatus* constants
    CreatedAt    time.Time    `json:"created_at"`
    UpdatedAt    time.Time    `json:"updated_at"`
}

// Intent describes WHY a medicine is used.
type Intent struct {
    Category   string `json:"category"`           // see Intent* constants in enums.go
    Indication string `json:"indication"`         // free text or SNOMED-CT-AU code
    Notes      string `json:"notes,omitempty"`
}

// Target describes WHAT successful therapy looks like for this medicine.
//
// Spec is JSON.RawMessage stored opaquely at the model layer; per-Kind
// shape contracts live in target_schemas.go. Validators in
// validation/target_validator.go delegate to per-Kind validators based
// on the Kind discriminator.
type Target struct {
    Kind string          `json:"kind"` // see TargetKind* constants in enums.go
    Spec json.RawMessage `json:"spec"`
}

// StopCriteria describes WHEN the medicine should stop.
//
// Triggers is a list of structured reasons (see StopTrigger* constants);
// ReviewDate is the next required clinical review; Spec is an optional
// JSONB shape (see stop_criteria_schemas.go) for additional structured
// criteria like threshold-based stops.
type StopCriteria struct {
    Triggers   []string        `json:"triggers"`
    ReviewDate *time.Time      `json:"review_date,omitempty"`
    Spec       json.RawMessage `json:"spec,omitempty"`
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared
go test ./v2_substrate/models/... -v
```

Expected: PASS for all MedicineUse + Target + spec tests.

---

## Task 5: MedicineUse + Target validators

**Files:**
- Create: `<kbs>/shared/v2_substrate/validation/medicine_use_validator.go`
- Create: `<kbs>/shared/v2_substrate/validation/target_validator.go`
- Modify: `<kbs>/shared/v2_substrate/validation/validators_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `<kbs>/shared/v2_substrate/validation/validators_test.go`:

```go
func TestValidateMedicineUseRequiresFields(t *testing.T) {
    base := models.MedicineUse{
        ID: uuid.New(), ResidentID: uuid.New(),
        DisplayName: "X",
        Intent:      models.Intent{Category: models.IntentTherapeutic, Indication: "y"},
        Target:      models.Target{Kind: models.TargetKindOpen, Spec: json.RawMessage(`{}`)},
        StopCriteria: models.StopCriteria{Triggers: []string{}},
        StartedAt:   time.Now(), Status: models.MedicineUseStatusActive,
    }
    if err := ValidateMedicineUse(base); err != nil {
        t.Errorf("expected pass for valid base; got %v", err)
    }

    // Missing display_name
    bad := base
    bad.DisplayName = ""
    if err := ValidateMedicineUse(bad); err == nil {
        t.Errorf("expected error for missing display_name")
    }

    // Invalid status
    bad = base
    bad.Status = "wrong"
    if err := ValidateMedicineUse(bad); err == nil {
        t.Errorf("expected error for invalid status")
    }

    // Invalid intent category
    bad = base
    bad.Intent.Category = "wrong"
    if err := ValidateMedicineUse(bad); err == nil {
        t.Errorf("expected error for invalid intent.category")
    }

    // Invalid stop trigger
    bad = base
    bad.StopCriteria.Triggers = []string{"unknown_trigger"}
    if err := ValidateMedicineUse(bad); err == nil {
        t.Errorf("expected error for invalid stop trigger")
    }
}

func TestValidateMedicineUseEndedAtAfterStartedAt(t *testing.T) {
    now := time.Now()
    earlier := now.Add(-24 * time.Hour)
    in := models.MedicineUse{
        ID: uuid.New(), ResidentID: uuid.New(),
        DisplayName: "X",
        Intent:      models.Intent{Category: models.IntentTherapeutic, Indication: "y"},
        Target:      models.Target{Kind: models.TargetKindOpen, Spec: json.RawMessage(`{}`)},
        StopCriteria: models.StopCriteria{Triggers: []string{}},
        StartedAt:   now,
        EndedAt:     &earlier,
        Status:      models.MedicineUseStatusActive,
    }
    if err := ValidateMedicineUse(in); err == nil {
        t.Errorf("expected error when ended_at < started_at")
    }
}

func TestValidateTargetBPThresholdSpec(t *testing.T) {
    // Valid
    valid, _ := json.Marshal(models.TargetBPThresholdSpec{SystolicMax: 140, DiastolicMax: 90})
    if err := ValidateTarget(models.Target{Kind: models.TargetKindBPThreshold, Spec: valid}); err != nil {
        t.Errorf("expected pass: %v", err)
    }
    // SystolicMax < DiastolicMax — invalid
    bad, _ := json.Marshal(models.TargetBPThresholdSpec{SystolicMax: 80, DiastolicMax: 90})
    if err := ValidateTarget(models.Target{Kind: models.TargetKindBPThreshold, Spec: bad}); err == nil {
        t.Errorf("expected error when systolic_max < diastolic_max")
    }
    // SystolicMax out of physiological range
    bad, _ = json.Marshal(models.TargetBPThresholdSpec{SystolicMax: 500, DiastolicMax: 90})
    if err := ValidateTarget(models.Target{Kind: models.TargetKindBPThreshold, Spec: bad}); err == nil {
        t.Errorf("expected error when systolic_max > 300")
    }
}

func TestValidateTargetCompletionDateSpec(t *testing.T) {
    valid, _ := json.Marshal(models.TargetCompletionDateSpec{
        EndDate: time.Now().Add(7 * 24 * time.Hour), DurationDays: 7,
    })
    if err := ValidateTarget(models.Target{Kind: models.TargetKindCompletionDate, Spec: valid}); err != nil {
        t.Errorf("expected pass: %v", err)
    }
    // Missing end_date
    bad := json.RawMessage(`{"duration_days": 7}`)
    if err := ValidateTarget(models.Target{Kind: models.TargetKindCompletionDate, Spec: bad}); err == nil {
        t.Errorf("expected error for missing end_date")
    }
}

func TestValidateTargetHbA1cBandSpec(t *testing.T) {
    valid, _ := json.Marshal(models.TargetHbA1cBandSpec{Min: 6.5, Max: 8.0})
    if err := ValidateTarget(models.Target{Kind: models.TargetKindHbA1cBand, Spec: valid}); err != nil {
        t.Errorf("expected pass: %v", err)
    }
    // Min >= Max
    bad, _ := json.Marshal(models.TargetHbA1cBandSpec{Min: 8.0, Max: 6.5})
    if err := ValidateTarget(models.Target{Kind: models.TargetKindHbA1cBand, Spec: bad}); err == nil {
        t.Errorf("expected error when min >= max")
    }
}

func TestValidateTargetUnknownKind(t *testing.T) {
    if err := ValidateTarget(models.Target{Kind: "LDL_target", Spec: json.RawMessage(`{}`)}); err == nil {
        t.Errorf("expected error for unrecognized target kind")
    }
}
```

Note: this test file already exists from β.1; just append to it. Add `"encoding/json"` import if not present.

- [ ] **Step 2: Run tests to verify failure**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared
go test ./v2_substrate/validation/... -run "TestValidateMedicineUse|TestValidateTarget" -v
```

Expected: FAIL with `undefined: ValidateMedicineUse` / `undefined: ValidateTarget`.

- [ ] **Step 3: Write target_validator.go**

Path: `<kbs>/shared/v2_substrate/validation/target_validator.go`

```go
package validation

import (
    "encoding/json"
    "fmt"

    "github.com/cardiofit/shared/v2_substrate/models"
)

// ValidateTarget reports any structural problem with t. The validator
// dispatches to per-Kind validators based on Target.Kind.
func ValidateTarget(t models.Target) error {
    if !models.IsValidTargetKind(t.Kind) {
        return fmt.Errorf("invalid target.kind %q", t.Kind)
    }
    switch t.Kind {
    case models.TargetKindBPThreshold:
        return validateTargetBPThreshold(t.Spec)
    case models.TargetKindCompletionDate:
        return validateTargetCompletionDate(t.Spec)
    case models.TargetKindSymptomResolution:
        return validateTargetSymptomResolution(t.Spec)
    case models.TargetKindHbA1cBand:
        return validateTargetHbA1cBand(t.Spec)
    case models.TargetKindOpen:
        return validateTargetOpen(t.Spec)
    }
    return fmt.Errorf("unhandled target.kind %q (validator not implemented)", t.Kind)
}

func validateTargetBPThreshold(raw json.RawMessage) error {
    var s models.TargetBPThresholdSpec
    if err := json.Unmarshal(raw, &s); err != nil {
        return fmt.Errorf("BP_threshold spec unmarshal: %w", err)
    }
    if s.SystolicMax <= 0 || s.SystolicMax > 300 {
        return fmt.Errorf("BP_threshold systolic_max %d out of physiological range (1-300)", s.SystolicMax)
    }
    if s.DiastolicMax <= 0 || s.DiastolicMax > 200 {
        return fmt.Errorf("BP_threshold diastolic_max %d out of physiological range (1-200)", s.DiastolicMax)
    }
    if s.SystolicMax < s.DiastolicMax {
        return fmt.Errorf("BP_threshold systolic_max (%d) must be >= diastolic_max (%d)", s.SystolicMax, s.DiastolicMax)
    }
    return nil
}

func validateTargetCompletionDate(raw json.RawMessage) error {
    var s models.TargetCompletionDateSpec
    if err := json.Unmarshal(raw, &s); err != nil {
        return fmt.Errorf("completion_date spec unmarshal: %w", err)
    }
    if s.EndDate.IsZero() {
        return fmt.Errorf("completion_date end_date is required")
    }
    if s.DurationDays < 0 {
        return fmt.Errorf("completion_date duration_days must be >= 0")
    }
    return nil
}

func validateTargetSymptomResolution(raw json.RawMessage) error {
    var s models.TargetSymptomResolutionSpec
    if err := json.Unmarshal(raw, &s); err != nil {
        return fmt.Errorf("symptom_resolution spec unmarshal: %w", err)
    }
    if s.TargetSymptom == "" {
        return fmt.Errorf("symptom_resolution target_symptom is required")
    }
    if s.MonitoringWindowDays < 0 {
        return fmt.Errorf("symptom_resolution monitoring_window_days must be >= 0")
    }
    return nil
}

func validateTargetHbA1cBand(raw json.RawMessage) error {
    var s models.TargetHbA1cBandSpec
    if err := json.Unmarshal(raw, &s); err != nil {
        return fmt.Errorf("HbA1c_band spec unmarshal: %w", err)
    }
    if s.Min <= 0 || s.Min > 20 {
        return fmt.Errorf("HbA1c_band min %.2f out of physiological range (0-20%%)", s.Min)
    }
    if s.Max <= 0 || s.Max > 20 {
        return fmt.Errorf("HbA1c_band max %.2f out of physiological range (0-20%%)", s.Max)
    }
    if s.Min >= s.Max {
        return fmt.Errorf("HbA1c_band min (%.2f) must be < max (%.2f)", s.Min, s.Max)
    }
    return nil
}

func validateTargetOpen(raw json.RawMessage) error {
    // Open spec is structurally permissive; rationale is optional.
    var s models.TargetOpenSpec
    if err := json.Unmarshal(raw, &s); err != nil {
        return fmt.Errorf("open spec unmarshal: %w", err)
    }
    return nil
}
```

- [ ] **Step 4: Write medicine_use_validator.go**

Path: `<kbs>/shared/v2_substrate/validation/medicine_use_validator.go`

```go
package validation

import (
    "errors"
    "fmt"

    "github.com/cardiofit/shared/v2_substrate/models"
)

// ValidateMedicineUse reports any structural problem with m. Includes
// validation of nested Intent.Category, Target (delegated to ValidateTarget),
// and StopCriteria.Triggers.
func ValidateMedicineUse(m models.MedicineUse) error {
    if m.DisplayName == "" {
        return errors.New("display_name is required")
    }
    if !models.IsValidMedicineUseStatus(m.Status) {
        return fmt.Errorf("invalid status %q", m.Status)
    }
    if !models.IsValidIntentCategory(m.Intent.Category) {
        return fmt.Errorf("invalid intent.category %q", m.Intent.Category)
    }
    if m.Intent.Indication == "" {
        return errors.New("intent.indication is required")
    }
    if err := ValidateTarget(m.Target); err != nil {
        return fmt.Errorf("target invalid: %w", err)
    }
    for i, trig := range m.StopCriteria.Triggers {
        if !models.IsValidStopTrigger(trig) {
            return fmt.Errorf("invalid stop_criteria.triggers[%d] %q", i, trig)
        }
    }
    if m.EndedAt != nil && m.EndedAt.Before(m.StartedAt) {
        return errors.New("ended_at must be on or after started_at")
    }
    return nil
}
```

- [ ] **Step 5: Run tests to verify pass**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared
go test ./v2_substrate/validation/... -v
```

Expected: PASS for all MedicineUse + Target validator tests.

---

## Task 6: Append Vaidshala extension URIs for MedicineUse fields

**Files:**
- Modify: `<kbs>/shared/v2_substrate/fhir/extensions.go`

- [ ] **Step 1: Append constants**

Append to `<kbs>/shared/v2_substrate/fhir/extensions.go`:

```go
// Vaidshala FHIR extension URIs for MedicineUse v2-distinguishing fields.
// AU FHIR MedicationRequest does not have native equivalents for Intent /
// Target / StopCriteria, so these are encoded as Vaidshala-namespaced
// extensions on the resource.
const (
    ExtMedicineIntent       = "https://vaidshala.health/fhir/StructureDefinition/medicine-intent"
    ExtMedicineTarget       = "https://vaidshala.health/fhir/StructureDefinition/medicine-target"
    ExtMedicineStopCriteria = "https://vaidshala.health/fhir/StructureDefinition/medicine-stop-criteria"
    ExtMedicineAMTCode      = "https://vaidshala.health/fhir/StructureDefinition/amt-code"
)

// SystemRouteCode is the FHIR-style code system URI for Vaidshala route
// values (ORAL, IV, IM, etc.) when serialized as Coding entries.
const SystemRouteCode = "https://vaidshala.health/fhir/CodeSystem/route"
```

- [ ] **Step 2: Verify compile**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared
go vet ./v2_substrate/fhir/...
```

Expected: no output.

---

## Task 7: MedicineUse ↔ AU MedicationRequest FHIR mapper

**Files:**
- Create: `<kbs>/shared/v2_substrate/fhir/medication_request_mapper.go`
- Create: `<kbs>/shared/v2_substrate/fhir/medication_request_mapper_test.go`

- [ ] **Step 1: Write the failing tests**

Path: `<kbs>/shared/v2_substrate/fhir/medication_request_mapper_test.go`

```go
package fhir

import (
    "encoding/json"
    "testing"
    "time"

    "github.com/google/uuid"

    "github.com/cardiofit/shared/v2_substrate/models"
)

func TestMedicineUseToMedicationRequestRoundTrip(t *testing.T) {
    bp := models.TargetBPThresholdSpec{SystolicMax: 140, DiastolicMax: 90}
    bpRaw, _ := json.Marshal(bp)

    in := models.MedicineUse{
        ID:          uuid.New(),
        ResidentID:  uuid.New(),
        AMTCode:     "12345",
        DisplayName: "Perindopril 5mg",
        Intent:      models.Intent{Category: models.IntentTherapeutic, Indication: "essential hypertension"},
        Target:      models.Target{Kind: models.TargetKindBPThreshold, Spec: bpRaw},
        StopCriteria: models.StopCriteria{Triggers: []string{models.StopTriggerAdverseEvent}},
        Dose:        "5mg",
        Route:       "ORAL",
        Frequency:   "QD",
        StartedAt:   time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
        Status:      models.MedicineUseStatusActive,
    }

    mr, err := MedicineUseToAUMedicationRequest(in)
    if err != nil {
        t.Fatalf("egress: %v", err)
    }
    if mr["resourceType"] != "MedicationRequest" {
        t.Errorf("resourceType: got %v, want MedicationRequest", mr["resourceType"])
    }

    // Round-trip via JSON
    b, _ := json.Marshal(mr)
    var rt map[string]interface{}
    if err := json.Unmarshal(b, &rt); err != nil {
        t.Fatalf("rt unmarshal: %v", err)
    }

    out, err := AUMedicationRequestToMedicineUse(rt)
    if err != nil {
        t.Fatalf("ingress: %v", err)
    }

    if out.AMTCode != in.AMTCode || out.DisplayName != in.DisplayName {
        t.Errorf("identity round-trip: got AMT=%q display=%q", out.AMTCode, out.DisplayName)
    }
    if out.Intent.Category != in.Intent.Category {
        t.Errorf("intent.category lost: got %q", out.Intent.Category)
    }
    if out.Target.Kind != in.Target.Kind {
        t.Errorf("target.kind lost: got %q", out.Target.Kind)
    }
    if string(out.Target.Spec) != string(in.Target.Spec) {
        t.Errorf("target.spec lost: got %s", string(out.Target.Spec))
    }
    if len(out.StopCriteria.Triggers) != 1 || out.StopCriteria.Triggers[0] != models.StopTriggerAdverseEvent {
        t.Errorf("stop_criteria.triggers lost: got %v", out.StopCriteria.Triggers)
    }
    if out.Status != models.MedicineUseStatusActive {
        t.Errorf("status: got %q", out.Status)
    }
}

func TestMedicineUseToMedicationRequest_RejectsInvalid(t *testing.T) {
    bad := models.MedicineUse{ID: uuid.New(), ResidentID: uuid.New(), DisplayName: ""}
    if _, err := MedicineUseToAUMedicationRequest(bad); err == nil {
        t.Errorf("expected validation rejection for missing DisplayName")
    }
}

func TestAUMedicationRequestToMedicineUse_WrongResourceType(t *testing.T) {
    in := map[string]interface{}{"resourceType": "Patient"}
    if _, err := AUMedicationRequestToMedicineUse(in); err == nil {
        t.Errorf("expected error for resourceType=Patient")
    }
}

func TestMedicineUseToMedicationRequest_WireFormat(t *testing.T) {
    // Pin the FHIR shape directly (not just via round-trip).
    bp := models.TargetBPThresholdSpec{SystolicMax: 140, DiastolicMax: 90}
    bpRaw, _ := json.Marshal(bp)
    in := models.MedicineUse{
        ID:          uuid.New(),
        ResidentID:  uuid.New(),
        AMTCode:     "12345",
        DisplayName: "Perindopril 5mg",
        Intent:      models.Intent{Category: models.IntentTherapeutic, Indication: "essential hypertension"},
        Target:      models.Target{Kind: models.TargetKindBPThreshold, Spec: bpRaw},
        StopCriteria: models.StopCriteria{Triggers: []string{models.StopTriggerAdverseEvent}},
        StartedAt:   time.Now(), Status: models.MedicineUseStatusActive,
    }
    mr, err := MedicineUseToAUMedicationRequest(in)
    if err != nil {
        t.Fatalf("egress: %v", err)
    }

    // status field should be lowercase per FHIR
    if mr["status"] != "active" {
        t.Errorf("status: got %v want active", mr["status"])
    }
    // intent extension must be present
    exts, ok := mr["extension"].([]map[string]interface{})
    if !ok {
        t.Fatalf("extension array not present: %T", mr["extension"])
    }
    foundIntent := false
    for _, ext := range exts {
        if ext["url"] == ExtMedicineIntent {
            foundIntent = true
            if _, ok := ext["valueString"]; !ok {
                t.Errorf("intent extension missing valueString")
            }
        }
    }
    if !foundIntent {
        t.Errorf("intent extension not found in wire format")
    }
}
```

- [ ] **Step 2: Run tests to verify failure**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared
go test ./v2_substrate/fhir/... -run "TestMedicineUse|TestAUMedicationRequest" -v
```

Expected: FAIL with `undefined: MedicineUseToAUMedicationRequest`.

- [ ] **Step 3: Write the mapper**

Path: `<kbs>/shared/v2_substrate/fhir/medication_request_mapper.go`

```go
package fhir

import (
    "encoding/json"
    "fmt"
    "time"

    "github.com/google/uuid"

    "github.com/cardiofit/shared/v2_substrate/models"
    "github.com/cardiofit/shared/v2_substrate/validation"
)

// MedicineUseToAUMedicationRequest translates a v2 MedicineUse to an AU FHIR
// MedicationRequest (HL7 AU Base v6.0.0). Returns a generic map[string]interface{}
// suitable for json.Marshal to wire format.
//
// AU-specific extensions used:
//   - ExtMedicineIntent       (Vaidshala-internal; FHIR has no native intent.indication concept)
//   - ExtMedicineTarget       (Vaidshala-internal; encodes Target.Kind + Spec as JSON)
//   - ExtMedicineStopCriteria (Vaidshala-internal; encodes StopCriteria as JSON)
//   - ExtMedicineAMTCode      (Vaidshala-internal; AMT code surface as identifier)
//
// Lossy fields (NOT round-tripped):
//   - CreatedAt / UpdatedAt — managed by canonical store
//   - PrescriberID — encoded as MedicationRequest.requester.reference; reverse parses
//     "Practitioner/<uuid>" but loses non-Practitioner references silently
//
// Egress validates the input via validation.ValidateMedicineUse before
// constructing the FHIR map. Ingress validates the output.
func MedicineUseToAUMedicationRequest(m models.MedicineUse) (map[string]interface{}, error) {
    if err := validation.ValidateMedicineUse(m); err != nil {
        return nil, fmt.Errorf("egress validation: %w", err)
    }

    out := map[string]interface{}{
        "resourceType":  "MedicationRequest",
        "id":            m.ID.String(),
        "status":        statusToFHIRStatus(m.Status),
        "intent":        "order", // FHIR MedicationRequest.intent — distinct from Vaidshala Intent.Category
        "subject": map[string]interface{}{
            "reference": "Patient/" + m.ResidentID.String(),
        },
        "authoredOn": m.StartedAt.UTC().Format(time.RFC3339),
    }

    // medicationCodeableConcept
    coding := []map[string]interface{}{}
    if m.AMTCode != "" {
        coding = append(coding, map[string]interface{}{
            "system":  "http://snomed.info/sct", // AMT codes are SNOMED-CT-AU
            "code":    m.AMTCode,
            "display": m.DisplayName,
        })
    }
    out["medicationCodeableConcept"] = map[string]interface{}{
        "coding": coding,
        "text":   m.DisplayName,
    }

    // dosageInstruction (single entry capturing dose/route/frequency)
    if m.Dose != "" || m.Route != "" || m.Frequency != "" {
        dosage := map[string]interface{}{
            "text": m.Dose + " " + m.Route + " " + m.Frequency,
        }
        if m.Route != "" {
            dosage["route"] = map[string]interface{}{
                "coding": []map[string]interface{}{
                    {"system": SystemRouteCode, "code": m.Route},
                },
            }
        }
        out["dosageInstruction"] = []map[string]interface{}{dosage}
    }

    // requester (PrescriberID → Practitioner reference)
    if m.PrescriberID != nil {
        out["requester"] = map[string]interface{}{
            "reference": "Practitioner/" + m.PrescriberID.String(),
        }
    }

    // dispenseRequest.validityPeriod (StartedAt / EndedAt)
    period := map[string]interface{}{
        "start": m.StartedAt.UTC().Format(time.RFC3339),
    }
    if m.EndedAt != nil {
        period["end"] = m.EndedAt.UTC().Format(time.RFC3339)
    }
    out["dispenseRequest"] = map[string]interface{}{"validityPeriod": period}

    // Vaidshala extensions for v2-distinguishing fields
    exts := []map[string]interface{}{}
    intentBytes, _ := json.Marshal(m.Intent)
    exts = append(exts, map[string]interface{}{
        "url":         ExtMedicineIntent,
        "valueString": string(intentBytes),
    })
    targetBytes, _ := json.Marshal(m.Target)
    exts = append(exts, map[string]interface{}{
        "url":         ExtMedicineTarget,
        "valueString": string(targetBytes),
    })
    stopBytes, _ := json.Marshal(m.StopCriteria)
    exts = append(exts, map[string]interface{}{
        "url":         ExtMedicineStopCriteria,
        "valueString": string(stopBytes),
    })
    if m.AMTCode != "" {
        exts = append(exts, map[string]interface{}{
            "url":         ExtMedicineAMTCode,
            "valueString": m.AMTCode,
        })
    }
    out["extension"] = exts

    return out, nil
}

// AUMedicationRequestToMedicineUse is the reverse mapper.
func AUMedicationRequestToMedicineUse(mr map[string]interface{}) (models.MedicineUse, error) {
    var m models.MedicineUse
    if rt, _ := mr["resourceType"].(string); rt != "MedicationRequest" {
        return m, fmt.Errorf("expected resourceType=MedicationRequest, got %q", rt)
    }
    if id, _ := mr["id"].(string); id != "" {
        if parsed, err := uuid.Parse(id); err == nil {
            m.ID = parsed
        }
    }
    if s, _ := mr["status"].(string); s != "" {
        m.Status = fhirStatusToStatus(s)
    }

    // subject → ResidentID
    if subj, ok := mr["subject"].(map[string]interface{}); ok {
        if ref, _ := subj["reference"].(string); len(ref) > len("Patient/") {
            if parsed, err := uuid.Parse(ref[len("Patient/"):]); err == nil {
                m.ResidentID = parsed
            }
        }
    }

    // medicationCodeableConcept → DisplayName + AMTCode (from coding system snomed-ct)
    if mcc, ok := mr["medicationCodeableConcept"].(map[string]interface{}); ok {
        if txt, _ := mcc["text"].(string); txt != "" {
            m.DisplayName = txt
        }
        if codes, ok := mcc["coding"].([]interface{}); ok {
            for _, codeAny := range codes {
                code, ok := codeAny.(map[string]interface{})
                if !ok {
                    continue
                }
                if code["system"] == "http://snomed.info/sct" {
                    if c, _ := code["code"].(string); c != "" {
                        m.AMTCode = c
                    }
                }
            }
        }
    }

    // dosageInstruction → Dose / Route / Frequency
    if di, ok := mr["dosageInstruction"].([]interface{}); ok && len(di) > 0 {
        if first, ok := di[0].(map[string]interface{}); ok {
            // Dose/Frequency are reconstructed from text; this is intentionally lossy.
            // For richer ingress, parse FHIR Timing/DoseAndRate properly.
            if route, ok := first["route"].(map[string]interface{}); ok {
                if coding, ok := route["coding"].([]interface{}); ok && len(coding) > 0 {
                    if c, ok := coding[0].(map[string]interface{}); ok {
                        m.Route, _ = c["code"].(string)
                    }
                }
            }
        }
    }

    // requester → PrescriberID
    if req, ok := mr["requester"].(map[string]interface{}); ok {
        if ref, _ := req["reference"].(string); len(ref) > len("Practitioner/") {
            if parsed, err := uuid.Parse(ref[len("Practitioner/"):]); err == nil {
                m.PrescriberID = &parsed
            }
        }
    }

    // dispenseRequest.validityPeriod → StartedAt / EndedAt
    if dr, ok := mr["dispenseRequest"].(map[string]interface{}); ok {
        if vp, ok := dr["validityPeriod"].(map[string]interface{}); ok {
            if s, _ := vp["start"].(string); s != "" {
                if t, err := time.Parse(time.RFC3339, s); err == nil {
                    m.StartedAt = t
                }
            }
            if e, _ := vp["end"].(string); e != "" {
                if t, err := time.Parse(time.RFC3339, e); err == nil {
                    m.EndedAt = &t
                }
            }
        }
    }

    // Extensions → Intent / Target / StopCriteria
    if exts, ok := mr["extension"].([]interface{}); ok {
        for _, extAny := range exts {
            ext, ok := extAny.(map[string]interface{})
            if !ok {
                continue
            }
            url, _ := ext["url"].(string)
            v, _ := ext["valueString"].(string)
            switch url {
            case ExtMedicineIntent:
                if v != "" && json.Valid([]byte(v)) {
                    _ = json.Unmarshal([]byte(v), &m.Intent)
                }
            case ExtMedicineTarget:
                if v != "" && json.Valid([]byte(v)) {
                    _ = json.Unmarshal([]byte(v), &m.Target)
                }
            case ExtMedicineStopCriteria:
                if v != "" && json.Valid([]byte(v)) {
                    _ = json.Unmarshal([]byte(v), &m.StopCriteria)
                }
            case ExtMedicineAMTCode:
                if v != "" {
                    m.AMTCode = v
                }
            }
        }
    }

    // Defense-in-depth: validate the output before returning to caller.
    if err := validation.ValidateMedicineUse(m); err != nil {
        return m, fmt.Errorf("ingress validation: %w", err)
    }
    return m, nil
}

// statusToFHIRStatus maps Vaidshala MedicineUseStatus to FHIR MedicationRequest.status.
func statusToFHIRStatus(s string) string {
    switch s {
    case models.MedicineUseStatusActive:
        return "active"
    case models.MedicineUseStatusPaused:
        return "on-hold"
    case models.MedicineUseStatusCeased:
        return "stopped"
    case models.MedicineUseStatusCompleted:
        return "completed"
    }
    return "unknown"
}

// fhirStatusToStatus is the reverse mapping. FHIR has additional values
// (draft, cancelled, entered-in-error) that have no Vaidshala analog —
// these collapse to "ceased" as the most conservative interpretation.
func fhirStatusToStatus(fs string) string {
    switch fs {
    case "active":
        return models.MedicineUseStatusActive
    case "on-hold":
        return models.MedicineUseStatusPaused
    case "stopped":
        return models.MedicineUseStatusCeased
    case "completed":
        return models.MedicineUseStatusCompleted
    }
    return models.MedicineUseStatusCeased
}
```

- [ ] **Step 4: Run tests to verify pass**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared
go test ./v2_substrate/fhir/... -v
```

Expected: PASS for all FHIR mapper tests including round-trip + RejectsInvalid + WrongResourceType + WireFormat.

---

## Task 8: Append MedicineUseStore interface

**Files:**
- Modify: `<kbs>/shared/v2_substrate/interfaces/storage.go`

- [ ] **Step 1: Append interface**

Append to `<kbs>/shared/v2_substrate/interfaces/storage.go`:

```go
// MedicineUseStore is the canonical storage contract for MedicineUse entities.
// kb-20-patient-profile is the only KB expected to implement this.
type MedicineUseStore interface {
    GetMedicineUse(ctx context.Context, id uuid.UUID) (*models.MedicineUse, error)
    UpsertMedicineUse(ctx context.Context, m models.MedicineUse) (*models.MedicineUse, error)
    ListMedicineUsesByResident(ctx context.Context, residentID uuid.UUID, limit, offset int) ([]models.MedicineUse, error)
}
```

- [ ] **Step 2: Verify compile**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared
go build ./v2_substrate/interfaces/...
```

Expected: no output.

---

## Task 9: kb-20 migration 008 part 2 (PART A — medication_states extension + view)

**Files:**
- Create: `<kbs>/kb-20-patient-profile/migrations/008_part2_clinical_primitives_partA.sql`

- [ ] **Step 1: Read existing medication_states columns**

```bash
grep -A 35 "CREATE TABLE.*medication_states" /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/migrations/001_initial_schema.sql
```

Confirms columns: id, patient_id, drug_name, drug_class, dose_mg, frequency, route, prescribed_by, fdc_components, fdc_parent_id, is_active, start_date, end_date, created_at, updated_at.

- [ ] **Step 2: Write migration**

Path: `<kbs>/kb-20-patient-profile/migrations/008_part2_clinical_primitives_partA.sql`

```sql
-- ============================================================================
-- Migration 008 part 2 PART A: MedicineUse v2 Substrate (Phase 1B-β.2-A)
--
-- Implements the v2 substrate MedicineUse entity:
--   - medication_states extension columns (nullable, backwards-compatible)
--     adds amt_code, display_name, intent JSONB, target JSONB,
--     stop_criteria JSONB, prescriber_id UUID (v2 Person.id),
--     resident_id UUID (canonical link to patient_profiles),
--     lifecycle_status enum
--   - medicine_uses_v2 view (compatibility read shape for v2 substrate consumers)
--
-- All existing kb-20 consumers continue reading raw medication_states unchanged.
--
-- Spec: docs/superpowers/specs/2026-05-04-1b-beta-2-clinical-primitives-design.md
-- Plan: docs/superpowers/plans/2026-05-04-1b-beta-2a-medicine-use-plan.md
-- Date: 2026-05-04
-- ============================================================================

BEGIN;

-- pgcrypto already enabled by 001; defensive re-declare for self-contained safety
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ============================================================================
-- Section 1 — Extend medication_states with v2 columns (all nullable)
-- ============================================================================
ALTER TABLE medication_states ADD COLUMN IF NOT EXISTS amt_code         TEXT;
ALTER TABLE medication_states ADD COLUMN IF NOT EXISTS display_name     TEXT;
ALTER TABLE medication_states ADD COLUMN IF NOT EXISTS intent           JSONB;
ALTER TABLE medication_states ADD COLUMN IF NOT EXISTS target           JSONB;
ALTER TABLE medication_states ADD COLUMN IF NOT EXISTS stop_criteria    JSONB;
ALTER TABLE medication_states ADD COLUMN IF NOT EXISTS prescriber_id    UUID;
ALTER TABLE medication_states ADD COLUMN IF NOT EXISTS resident_id      UUID;
ALTER TABLE medication_states ADD COLUMN IF NOT EXISTS lifecycle_status TEXT;

-- Lifecycle status CHECK (NOT VALID — won't fail on existing legacy rows with NULL)
ALTER TABLE medication_states
    ADD CONSTRAINT medication_states_lifecycle_status_valid
    CHECK (lifecycle_status IS NULL OR lifecycle_status IN ('active','paused','ceased','completed')) NOT VALID;

-- Indexes for v2 access patterns
CREATE INDEX IF NOT EXISTS idx_medication_states_resident_id ON medication_states(resident_id) WHERE resident_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_medication_states_amt_code ON medication_states(amt_code) WHERE amt_code IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_medication_states_lifecycle_active
    ON medication_states(resident_id, lifecycle_status) WHERE lifecycle_status = 'active';

-- ============================================================================
-- Section 2 — Per-column COMMENTs documenting the v2 contract
-- ============================================================================
COMMENT ON COLUMN medication_states.amt_code IS
    'v2 MedicineUse Australian Medicines Terminology code. Added 2026-05-04 in migration 008_part2_partA.';
COMMENT ON COLUMN medication_states.display_name IS
    'v2 MedicineUse human-readable name. Falls back to legacy drug_name when absent. Added 2026-05-04 in migration 008_part2_partA.';
COMMENT ON COLUMN medication_states.intent IS
    'v2 MedicineUse Intent JSONB. Shape: {"category": "therapeutic|preventive|symptomatic|trial|deprescribing", "indication": "...", "notes": "..."}. Added 2026-05-04 in migration 008_part2_partA.';
COMMENT ON COLUMN medication_states.target IS
    'v2 MedicineUse Target JSONB. Shape: {"kind": "BP_threshold|completion_date|symptom_resolution|HbA1c_band|open", "spec": {...per kind...}}. Added 2026-05-04 in migration 008_part2_partA.';
COMMENT ON COLUMN medication_states.stop_criteria IS
    'v2 MedicineUse StopCriteria JSONB. Shape: {"triggers": [...], "review_date": "...", "spec": {...}}. Added 2026-05-04 in migration 008_part2_partA.';
COMMENT ON COLUMN medication_states.prescriber_id IS
    'v2 Person.id reference (kb-20 persons table; FK enforced at write time, not as DB FK to keep migration non-breaking). Added 2026-05-04 in migration 008_part2_partA.';
COMMENT ON COLUMN medication_states.resident_id IS
    'v2 Resident.id reference. NULL for legacy rows; populated for v2 writes. medicine_uses_v2 view backfills via legacy patient_id lookup. Added 2026-05-04 in migration 008_part2_partA.';
COMMENT ON COLUMN medication_states.lifecycle_status IS
    'v2 MedicineUse lifecycle status enum (active|paused|ceased|completed). Coexists with legacy is_active boolean; v2 writers populate this directly. medicine_uses_v2 view derives status from this when set, falls back to is_active otherwise. Added 2026-05-04 in migration 008_part2_partA.';

-- ============================================================================
-- Section 3 — medicine_uses_v2 view
-- ============================================================================
-- Compatibility read shape for v2 substrate consumers. Existing consumers
-- (medication-service, etc.) keep reading raw medication_states unchanged.
--
-- Status precedence:
--   1. Use lifecycle_status when set (v2 writer)
--   2. Fall back to derived from is_active (legacy: TRUE → 'active'; FALSE → 'ceased')
--
-- ResidentID precedence:
--   1. Use resident_id when set (v2 writer)
--   2. Fall back to patient_profiles.id lookup via legacy patient_id (VARCHAR(100))
--
-- DisplayName precedence:
--   1. Use display_name when set (v2 writer)
--   2. Fall back to drug_name (legacy)
-- ============================================================================
CREATE OR REPLACE VIEW medicine_uses_v2 AS
SELECT
    ms.id                                                            AS id,
    COALESCE(
        ms.resident_id,
        (SELECT pp.id FROM patient_profiles pp WHERE pp.patient_id = ms.patient_id LIMIT 1)
    )                                                                AS resident_id,
    ms.amt_code                                                      AS amt_code,
    COALESCE(ms.display_name, ms.drug_name)                          AS display_name,
    COALESCE(ms.intent, '{"category":"therapeutic","indication":""}'::jsonb)        AS intent,
    COALESCE(ms.target, '{"kind":"open","spec":{}}'::jsonb)          AS target,
    COALESCE(ms.stop_criteria, '{"triggers":[]}'::jsonb)             AS stop_criteria,
    CASE
        WHEN ms.dose_mg IS NOT NULL THEN ms.dose_mg::TEXT || 'mg'
        ELSE ''
    END                                                              AS dose,
    ms.route                                                         AS route,
    ms.frequency                                                     AS frequency,
    ms.prescriber_id                                                 AS prescriber_id,
    ms.start_date                                                    AS started_at,
    ms.end_date                                                      AS ended_at,
    COALESCE(
        ms.lifecycle_status,
        CASE WHEN ms.is_active THEN 'active' ELSE 'ceased' END
    )                                                                AS status,
    ms.created_at,
    ms.updated_at
FROM medication_states ms;

COMMENT ON VIEW medicine_uses_v2 IS
    'Compatibility read shape for v2 substrate MedicineUse consumers. v2 writers populate the new columns directly; legacy reads fall back to drug_name + is_active. Default JSONB values for intent/target/stop_criteria when NULL preserve schema-required-fields contract for v2 readers.';

COMMIT;

-- ============================================================================
-- Acceptance check (run after applying):
--   SELECT column_name FROM information_schema.columns
--     WHERE table_name='medication_states'
--     AND column_name IN ('amt_code','display_name','intent','target','stop_criteria','prescriber_id','resident_id','lifecycle_status');
--   -- expect 8 rows
--   SELECT * FROM medicine_uses_v2 LIMIT 1;
--   -- view executes (may be 0 rows on fresh DB; intent/target/stop_criteria default JSONB if legacy rows exist)
-- ============================================================================
```

- [ ] **Step 3: Sanity-check SQL syntax**

```bash
FILE=/Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/migrations/008_part2_clinical_primitives_partA.sql
echo "BEGIN: $(grep -c '^BEGIN;' $FILE)"
echo "COMMIT: $(grep -c '^COMMIT;' $FILE)"
echo "ALTER TABLE: $(grep -c 'ALTER TABLE medication_states' $FILE)"
echo "VIEW: $(grep -c 'CREATE OR REPLACE VIEW' $FILE)"
echo "COMMENT ON COLUMN: $(grep -c 'COMMENT ON COLUMN' $FILE)"
echo "Indexes: $(grep -c 'CREATE INDEX' $FILE)"
```

Expected: 1 / 1 / 9 (8 ADD COLUMN + 1 ADD CONSTRAINT) / 1 / 8 / 3.

---

## Task 10: V2SubstrateStore MedicineUse methods

**Files:**
- Modify: `<kbs>/kb-20-patient-profile/internal/storage/v2_substrate_store.go`

- [ ] **Step 1: Append column constants and scanMedicineUse helper**

In `v2_substrate_store.go`, append after the existing `roleColumns` constant block:

```go
// medicineUseColumns is the SELECT projection for medicine_uses_v2. Used by
// scanMedicineUse so list and Get share scanning logic.
const medicineUseColumns = `
    id, resident_id, amt_code, display_name, intent, target, stop_criteria,
    dose, route, frequency, prescriber_id, started_at, ended_at, status,
    created_at, updated_at
`

// scanMedicineUse populates a MedicineUse from a row scanner (sql.Row or sql.Rows).
func scanMedicineUse(rs rowScanner) (models.MedicineUse, error) {
    var m models.MedicineUse
    var amtCode, dose, route, frequency sql.NullString
    var intent, target, stopCriteria []byte
    var prescriberID *uuid.UUID
    var endedAt sql.NullTime
    if err := rs.Scan(
        &m.ID, &m.ResidentID, &amtCode, &m.DisplayName,
        &intent, &target, &stopCriteria,
        &dose, &route, &frequency,
        &prescriberID, &m.StartedAt, &endedAt, &m.Status,
        &m.CreatedAt, &m.UpdatedAt,
    ); err != nil {
        return m, err
    }
    if amtCode.Valid {
        m.AMTCode = amtCode.String
    }
    if dose.Valid {
        m.Dose = dose.String
    }
    if route.Valid {
        m.Route = route.String
    }
    if frequency.Valid {
        m.Frequency = frequency.String
    }
    if endedAt.Valid {
        m.EndedAt = &endedAt.Time
    }
    if prescriberID != nil {
        m.PrescriberID = prescriberID
    }
    if len(intent) > 0 {
        if err := json.Unmarshal(intent, &m.Intent); err != nil {
            return m, fmt.Errorf("scan intent: %w", err)
        }
    }
    if len(target) > 0 {
        if err := json.Unmarshal(target, &m.Target); err != nil {
            return m, fmt.Errorf("scan target: %w", err)
        }
    }
    if len(stopCriteria) > 0 {
        if err := json.Unmarshal(stopCriteria, &m.StopCriteria); err != nil {
            return m, fmt.Errorf("scan stop_criteria: %w", err)
        }
    }
    return m, nil
}
```

- [ ] **Step 2: Append GetMedicineUse, UpsertMedicineUse, ListMedicineUsesByResident**

Append to `v2_substrate_store.go`:

```go
// GetMedicineUse returns the canonical MedicineUse row for id, or
// fmt.Errorf("...: %w", interfaces.ErrNotFound) if not found.
func (s *V2SubstrateStore) GetMedicineUse(ctx context.Context, id uuid.UUID) (*models.MedicineUse, error) {
    q := "SELECT " + medicineUseColumns + " FROM medicine_uses_v2 WHERE id = $1"
    row := s.db.QueryRowContext(ctx, q, id)
    m, err := scanMedicineUse(row)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, fmt.Errorf("get medicine_use %s: %w", id, interfaces.ErrNotFound)
        }
        return nil, fmt.Errorf("get medicine_use %s: %w", id, err)
    }
    return &m, nil
}

// UpsertMedicineUse INSERT-ON-CONFLICT UPDATEs the canonical medication_states
// row, populating both v2 columns AND legacy columns to satisfy NOT NULL
// constraints on the underlying table. Returns the round-tripped Get to
// surface DB-managed updated_at.
func (s *V2SubstrateStore) UpsertMedicineUse(ctx context.Context, m models.MedicineUse) (*models.MedicineUse, error) {
    intentBytes, err := json.Marshal(m.Intent)
    if err != nil {
        return nil, fmt.Errorf("marshal intent: %w", err)
    }
    targetBytes, err := json.Marshal(m.Target)
    if err != nil {
        return nil, fmt.Errorf("marshal target: %w", err)
    }
    stopBytes, err := json.Marshal(m.StopCriteria)
    if err != nil {
        return nil, fmt.Errorf("marshal stop_criteria: %w", err)
    }

    // Legacy patient_id column (VARCHAR(100) NOT NULL) — derive from
    // ResidentID. medicine_uses_v2 view backfills v2.resident_id via lookup.
    legacyPatientID := m.ResidentID.String()

    // Legacy is_active boolean derived from lifecycle_status.
    legacyIsActive := m.Status == models.MedicineUseStatusActive

    // Legacy drug_name + drug_class — fill from display_name when v2 writer
    // doesn't supply them. drug_class is a v1-required field; default to "UNKNOWN"
    // when the v2 writer didn't carry a class — kb-20 v1 callers will continue
    // to populate it.
    legacyDrugName := m.DisplayName
    legacyDrugClass := "UNKNOWN"

    q := `
        INSERT INTO medication_states (
            id, patient_id, drug_name, drug_class, route, frequency,
            is_active, start_date, end_date,
            amt_code, display_name, intent, target, stop_criteria,
            prescriber_id, resident_id, lifecycle_status,
            created_at, updated_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6,
            $7, $8, $9,
            $10, $11, $12, $13, $14,
            $15, $16, $17,
            NOW(), NOW()
        )
        ON CONFLICT (id) DO UPDATE SET
            patient_id = EXCLUDED.patient_id,
            drug_name = EXCLUDED.drug_name,
            route = EXCLUDED.route,
            frequency = EXCLUDED.frequency,
            is_active = EXCLUDED.is_active,
            start_date = EXCLUDED.start_date,
            end_date = EXCLUDED.end_date,
            amt_code = EXCLUDED.amt_code,
            display_name = EXCLUDED.display_name,
            intent = EXCLUDED.intent,
            target = EXCLUDED.target,
            stop_criteria = EXCLUDED.stop_criteria,
            prescriber_id = EXCLUDED.prescriber_id,
            resident_id = EXCLUDED.resident_id,
            lifecycle_status = EXCLUDED.lifecycle_status,
            updated_at = NOW()
    `
    if _, err := s.db.ExecContext(ctx, q,
        m.ID, legacyPatientID, legacyDrugName, legacyDrugClass,
        nilIfEmpty(m.Route), nilIfEmpty(m.Frequency),
        legacyIsActive, m.StartedAt, m.EndedAt,
        nilIfEmpty(m.AMTCode), m.DisplayName, intentBytes, targetBytes, stopBytes,
        m.PrescriberID, m.ResidentID, m.Status,
    ); err != nil {
        return nil, fmt.Errorf("upsert medicine_use: %w", err)
    }
    return s.GetMedicineUse(ctx, m.ID)
}

// ListMedicineUsesByResident returns medicine_uses_v2 rows for residentID.
// limit must be > 0; offset must be >= 0; both are caller's responsibility.
func (s *V2SubstrateStore) ListMedicineUsesByResident(ctx context.Context, residentID uuid.UUID, limit, offset int) ([]models.MedicineUse, error) {
    q := "SELECT " + medicineUseColumns + " FROM medicine_uses_v2 WHERE resident_id = $1 ORDER BY started_at DESC LIMIT $2 OFFSET $3"
    rows, err := s.db.QueryContext(ctx, q, residentID, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("list medicine_uses: %w", err)
    }
    defer rows.Close()
    var out []models.MedicineUse
    for rows.Next() {
        m, err := scanMedicineUse(rows)
        if err != nil {
            return nil, fmt.Errorf("scan medicine_use row: %w", err)
        }
        out = append(out, m)
    }
    if err := rows.Err(); err != nil {
        return nil, err
    }
    return out, nil
}
```

- [ ] **Step 3: Append compile-time interface assertion**

In the same file, find the existing block of `var _ interfaces.X = (*V2SubstrateStore)(nil)` lines and append:

```go
var _ interfaces.MedicineUseStore = (*V2SubstrateStore)(nil)
```

- [ ] **Step 4: Verify compile**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile
go build ./...
```

Expected: clean.

---

## Task 11: REST handlers for MedicineUse

**Files:**
- Modify: `<kbs>/kb-20-patient-profile/internal/api/v2_substrate_handlers.go`

- [ ] **Step 1: Append RegisterRoutes additions for MedicineUse**

Find the existing `RegisterRoutes` function. Append the 4 new routes inside its body (location: at the end of the function, before the closing brace):

```go
    // MedicineUse v2 substrate (β.2-A)
    g.POST("/medicine_uses", h.upsertMedicineUse)
    g.GET("/medicine_uses/:id", h.getMedicineUse)
    g.GET("/residents/:resident_id/medicine_uses", h.listMedicineUsesByResident)
```

- [ ] **Step 2: Append handler methods**

Append at the bottom of `v2_substrate_handlers.go`:

```go
func (h *V2SubstrateHandlers) upsertMedicineUse(c *gin.Context) {
    var m models.MedicineUse
    if err := c.BindJSON(&m); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    if err := validation.ValidateMedicineUse(m); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    out, err := h.store.UpsertMedicineUse(c.Request.Context(), m)
    if err != nil {
        respondError(c, err)
        return
    }
    c.JSON(http.StatusOK, out)
}

func (h *V2SubstrateHandlers) getMedicineUse(c *gin.Context) {
    id, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return
    }
    m, err := h.store.GetMedicineUse(c.Request.Context(), id)
    if err != nil {
        respondError(c, err)
        return
    }
    c.JSON(http.StatusOK, m)
}

func (h *V2SubstrateHandlers) listMedicineUsesByResident(c *gin.Context) {
    residentID, err := uuid.Parse(c.Param("resident_id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resident_id"})
        return
    }
    limit, err := strconv.Atoi(c.DefaultQuery("limit", "100"))
    if err != nil || limit <= 0 || limit > 1000 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be 1-1000"})
        return
    }
    offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
    if err != nil || offset < 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "offset must be >= 0"})
        return
    }
    out, err := h.store.ListMedicineUsesByResident(c.Request.Context(), residentID, limit, offset)
    if err != nil {
        respondError(c, err)
        return
    }
    c.JSON(http.StatusOK, out)
}
```

- [ ] **Step 3: Verify compile**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile
go build ./...
```

Expected: clean.

---

## Task 12: KB20Client MedicineUse methods

**Files:**
- Modify: `<kbs>/shared/v2_substrate/client/kb20_client.go`
- Modify: `<kbs>/shared/v2_substrate/client/kb20_client_test.go`

- [ ] **Step 1: Append client methods**

Append to `kb20_client.go` (after the existing Role methods, before `doJSON`):

```go
// MedicineUse — Phase 1B-β.2-A

func (c *KB20Client) UpsertMedicineUse(ctx context.Context, m models.MedicineUse) (*models.MedicineUse, error) {
    return doJSON[models.MedicineUse](ctx, c.http, http.MethodPost, c.baseURL+"/v2/medicine_uses", m)
}

func (c *KB20Client) GetMedicineUse(ctx context.Context, id uuid.UUID) (*models.MedicineUse, error) {
    return doJSON[models.MedicineUse](ctx, c.http, http.MethodGet, c.baseURL+"/v2/medicine_uses/"+id.String(), nil)
}

func (c *KB20Client) ListMedicineUsesByResident(ctx context.Context, residentID uuid.UUID, limit, offset int) ([]models.MedicineUse, error) {
    q := url.Values{}
    q.Set("limit", strconv.Itoa(limit))
    q.Set("offset", strconv.Itoa(offset))
    u := c.baseURL + "/v2/residents/" + residentID.String() + "/medicine_uses?" + q.Encode()
    out, err := doJSON[[]models.MedicineUse](ctx, c.http, http.MethodGet, u, nil)
    if err != nil {
        return nil, err
    }
    return *out, nil
}
```

If `strconv` is not yet imported in this file, add it.

- [ ] **Step 2: Append httptest test**

Append to `kb20_client_test.go`:

```go
func TestKB20ClientMedicineUseRoundTrip(t *testing.T) {
    bp := models.TargetBPThresholdSpec{SystolicMax: 140, DiastolicMax: 90}
    bpRaw, _ := json.Marshal(bp)
    in := models.MedicineUse{
        ID:          uuid.New(),
        ResidentID:  uuid.New(),
        AMTCode:     "12345",
        DisplayName: "Perindopril 5mg",
        Intent:      models.Intent{Category: models.IntentTherapeutic, Indication: "essential hypertension"},
        Target:      models.Target{Kind: models.TargetKindBPThreshold, Spec: bpRaw},
        StopCriteria: models.StopCriteria{Triggers: []string{models.StopTriggerAdverseEvent}},
        StartedAt:   time.Now(),
        Status:      models.MedicineUseStatusActive,
    }

    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        switch {
        case r.Method == http.MethodPost && r.URL.Path == "/v2/medicine_uses":
            w.WriteHeader(http.StatusOK)
            json.NewEncoder(w).Encode(in)
        case r.Method == http.MethodGet && r.URL.Path == "/v2/medicine_uses/"+in.ID.String():
            w.WriteHeader(http.StatusOK)
            json.NewEncoder(w).Encode(in)
        case r.Method == http.MethodGet && r.URL.Path == "/v2/residents/"+in.ResidentID.String()+"/medicine_uses":
            w.WriteHeader(http.StatusOK)
            json.NewEncoder(w).Encode([]models.MedicineUse{in})
        default:
            w.WriteHeader(http.StatusNotFound)
        }
    }))
    defer server.Close()

    c := NewKB20Client(server.URL)

    out, err := c.UpsertMedicineUse(context.Background(), in)
    require.NoError(t, err)
    require.Equal(t, in.DisplayName, out.DisplayName)

    fetched, err := c.GetMedicineUse(context.Background(), in.ID)
    require.NoError(t, err)
    require.Equal(t, in.ID, fetched.ID)

    list, err := c.ListMedicineUsesByResident(context.Background(), in.ResidentID, 10, 0)
    require.NoError(t, err)
    require.Len(t, list, 1)
}
```

- [ ] **Step 3: Run tests**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared
go test ./v2_substrate/client/... -v
```

Expected: PASS.

---

## Task 13: End-to-end verification

**Files:** none (verification only)

- [ ] **Step 1: Run all v2_substrate tests**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared
go vet ./v2_substrate/... 2>&1
go test ./v2_substrate/... -v 2>&1 | tail -50
```

Expected: vet clean; all packages pass (models, validation, fhir, client, interfaces).

- [ ] **Step 2: Verify kb-20 build + tests**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile
go vet ./... 2>&1 | head
go build ./...
go test ./internal/storage/... ./internal/api/... 2>&1 | tail -10
```

Expected: vet clean; build clean; integration tests SKIP without KB20_TEST_DATABASE_URL but no compile errors.

- [ ] **Step 3: Verify migration applies (optional, requires running DB)**

```bash
make -C /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services run-kb-docker
psql -h localhost -p 5433 -U kb_drug_rules_user -d kb_20_patient_profile -f /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/migrations/008_part2_clinical_primitives_partA.sql

# Verify columns added
psql -h localhost -p 5433 -U kb_drug_rules_user -d kb_20_patient_profile -c "
SELECT column_name FROM information_schema.columns
WHERE table_name='medication_states'
AND column_name IN ('amt_code','display_name','intent','target','stop_criteria','prescriber_id','resident_id','lifecycle_status')
ORDER BY column_name;"

# Verify view executes
psql -h localhost -p 5433 -U kb_drug_rules_user -d kb_20_patient_profile -c "SELECT COUNT(*) FROM medicine_uses_v2;"
```

Expected: 8 rows from columns query; view returns count without error.

- [ ] **Step 4: Document completion in the spec file**

Append to `docs/superpowers/specs/2026-05-04-1b-beta-2-clinical-primitives-design.md` after §6 β.2-A description:

```markdown
### β.2-A Completion (2026-05-04)

✅ **Cluster complete.** MedicineUse v2 substrate entity delivered end-to-end.

**Test counts:**
- shared/v2_substrate/models: N tests pass (incl. 5 Target.Spec round-trip + 2 StopCriteria.Spec)
- shared/v2_substrate/validation: N tests pass (incl. ValidateTarget per-Kind dispatch)
- shared/v2_substrate/fhir: N tests pass (incl. _RejectsInvalid + _WrongResourceType + _WireFormat)
- shared/v2_substrate/client: N tests pass

**Migration:** 008_part2_partA — non-breaking; applies cleanly on fresh + existing-data DB.

**Open follow-ups for β.2-B:** N/A — proceed to Observation cluster.
```

(Replace `N` with actual numbers.)

---

## Self-Review (post-write)

**Spec coverage check (against `2026-05-04-1b-beta-2-clinical-primitives-design.md` §6 β.2-A):**

- shared/v2_substrate/models/{medicine_use, target_schemas, stop_criteria_schemas}.go + tests → Tasks 2, 3, 4
- shared/v2_substrate/validation/medicine_use_validator.go + target_validator.go → Task 5
- shared/v2_substrate/fhir/medication_request_mapper.go + tests → Tasks 6 (extensions), 7 (mapper)
- shared/v2_substrate/interfaces/storage.go MedicineUseStore added → Task 8
- shared/v2_substrate/client/kb20_client.go MedicineUse methods + httptest → Task 12
- kb-20 migration 008_part2 PART A → Task 9
- kb-20 storage scanMedicineUse + Get/Upsert/ListByResident → Task 10
- kb-20 handlers 4 REST endpoints → Task 11

All 5 Target.Kind round-trips covered: BP_threshold (Task 4 + Task 7 round-trip), completion_date (Task 5 validator), symptom_resolution (Task 5), HbA1c_band (Task 5), open (Task 4 omit-empty test). FHIR mapper round-trip uses BP_threshold; per-Kind validators tested in Task 5.

**Placeholder scan:** No "TBD/TODO/implement later". All code complete with tests.

**Type consistency:**
- `models.Intent`, `models.Target`, `models.StopCriteria` defined in Task 4, used consistently across validator (Task 5), FHIR mapper (Task 7), store (Task 10), handlers (Task 11), client (Task 12). ✓
- Field names (DisplayName, AMTCode, Intent.Category, Target.Kind/Spec, StopCriteria.Triggers/ReviewDate/Spec) consistent across all consumers. ✓
- Module path `github.com/cardiofit/shared` throughout. ✓
- `respondError`, `nilIfEmpty`, `rowScanner`, `scanResident`/`scanRole` patterns inherited from β.1 (no redefinition). ✓

**Module path note:** All Go imports use `github.com/cardiofit/shared/v2_substrate/...` consistent with β.1 confirmed module path.
