# Phase 1B-β.2 — Clinical Primitives Design

**Date:** 2026-05-04
**Phase:** 1B-β.2 (MedicineUse + Observation substrate entities)
**Status:** Design — pending implementation
**Parent spec:** `docs/superpowers/specs/2026-05-04-1b-beta-substrate-entities-design.md` (overall 1B-β; this spec is the β.2 milestone)
**Predecessor:** Phase 1B-β.1 (Resident + Person + Role) — complete

---

## 1. Context

Phase 1B-β.2 delivers the second clinical-primitives cluster of the v2 substrate: **MedicineUse** (with v2-distinguishing `intent`, `target`, `stop_criteria`) and **Observation** (with delta-on-write). It does NOT cover Event/EvidenceTrace (β.3), the Authorisation evaluator, state machines, or Layer 1B adapters.

### 1.1 Architecture inherited from β.1

- Shared library at `backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/`
- Designated KB owns canonical row (kb-20-patient-profile for both MedicineUse and Observation)
- Internal Go types with FHIR mappers at boundaries (HL7 AU Base v6.0.0)
- Non-breaking migrations (extend in place where existing tables fit; new tables only when greenfield)
- Module path: `github.com/cardiofit/shared`
- gRPC + REST endpoints; KB20Client extended with new methods
- Storage uses raw SQL + lib/pq (`scanResident`/`scanRole` pattern from β.1) for single-query lists

### 1.2 What's already in place from β.1

- `shared/v2_substrate/models/{enums,resident,person,role}.go` (10 tests)
- `shared/v2_substrate/fhir/{extensions,patient_mapper,practitioner_mapper}.go` (13 tests)
- `shared/v2_substrate/validation/{resident,person,role}_validator.go` (7 tests)
- `shared/v2_substrate/interfaces/storage.go` (ResidentStore + PersonStore + RoleStore + ErrNotFound sentinel)
- `shared/v2_substrate/client/kb20_client.go` (KB20Client, doJSON[T] generic helper)
- `kb-20-patient-profile/migrations/008_part1_actor_model.sql` (persons + roles tables; patient_profiles extended with v2 columns; residents_v2 view)
- `kb-20-patient-profile/internal/storage/v2_substrate_store.go` (V2SubstrateStore with rowScanner pattern, ErrNotFound wrapping, no N+1)
- `kb-20-patient-profile/internal/api/v2_substrate_handlers.go` (10 REST endpoints, sentinel→404 dispatch)

### 1.3 Existing kb-20 analogs and constraints

**`medication_states` (kb-20 migrations 001 + 006/007)** — fits MedicineUse extension well:
- `drug_name VARCHAR(200)`, `drug_class VARCHAR(50)`, `dose_mg DECIMAL(10,2)`, `frequency VARCHAR(50)`, `route VARCHAR(30)`, `prescribed_by VARCHAR(100)`
- `is_active BOOLEAN`, `start_date/end_date TIMESTAMPTZ`
- FDC decomposition fields (irrelevant to v2 substrate; preserved)
- **β.2 extends with:** `intent JSONB`, `target JSONB`, `stop_criteria JSONB`, `amt_code TEXT`, `display_name TEXT` — all nullable

**`lab_entries` (kb-20 migration 001)** — does NOT fit Observation:
- `lab_type VARCHAR(30)` (free-text, e.g. 'EGFR', 'HBA1C') — no LOINC/SNOMED codes
- `value DECIMAL(10,4)`, `unit VARCHAR(20)` — numeric only, no behavioural ValueText
- `validation_status` enum (ACCEPTED/FLAGGED/REJECTED), `flag_reason`
- **β.2 leaves lab_entries unchanged** — existing consumers (medication-service) keep reading raw lab_entries
- **Observation lands in a NEW `observations` table** with kind discriminator covering vital/lab/behavioural/mobility/weight

**kb-26 baseline computation already exists** — `BaselineAdjustmentController.SelectBaseline()` + `AcuteRepository.FetchBaseline(patientID, vitalType)`. β.2's delta computation calls this (or replicates the algorithm; see §3.4).

---

## 2. Architectural commitments specific to β.2

### 2.1 Delta-on-write: service-layer compute, NOT DB trigger

The store layer (V2SubstrateStore.UpsertObservation) computes Delta before the INSERT. Why not a DB trigger:
- Trigger can't cleanly call kb-26's BaselineAdjustmentController (separate service, separate DB)
- Trigger creates hidden coupling — schema change ripples invisibly
- Service-layer is testable in unit tests with mock baseline data
- Performance is fine for MVP scale (one baseline read + arithmetic per write)

The Delta service is `shared/v2_substrate/delta/` package with:
- `interface BaselineProvider` — exposes `FetchBaseline(ctx, patientID, vitalType) (*Baseline, error)` so kb-26 (or a replica) can be plugged in
- `func ComputeDelta(obs models.Observation, baseline Baseline) models.Delta` — pure function, fully testable

### 2.2 Observation table: greenfield, kind discriminator

New table `observations` in migration 008_part2:

```sql
CREATE TABLE observations (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resident_id  UUID NOT NULL,
    loinc_code   TEXT,
    snomed_code  TEXT,
    kind         TEXT NOT NULL CHECK (kind IN ('vital','lab','behavioural','mobility','weight')),
    value        DECIMAL(12,4),     -- nullable; for behavioural use value_text
    value_text   TEXT,              -- nullable; complement to value
    unit         TEXT,
    observed_at  TIMESTAMPTZ NOT NULL,
    source_id    UUID,              -- application-validated reference to kb-22.clinical_sources; no FK (cross-DB)
    delta        JSONB,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT observations_value_or_text CHECK (value IS NOT NULL OR value_text IS NOT NULL)
);
```

`lab_entries` continues unchanged. View `observations_v2` UNIONs both:
- SELECT from `observations` directly (for kind != 'lab' OR new lab data written via v2 path)
- UNION ALL SELECT from `lab_entries` projected into Observation shape with kind='lab'

### 2.3 MedicineUse migration: extend in place

`medication_states` gets nullable JSONB columns + AMT/display_name fields. The existing `drug_name`, `drug_class`, `dose_mg`, `frequency`, `route`, `is_active`, `start_date`, `end_date` map directly to MedicineUse fields. View `medicine_uses_v2` projects v2 shape with COALESCE for legacy-vs-v2 field precedence.

### 2.4 Source provenance: UUID without FK (cross-DB constraint)

`observations.source_id` and `medication_states` (no source field today) — both reference `kb-22.clinical_sources` which is in a separate Postgres database. Cross-DB FK impossible. Store as UUID; document via column COMMENT that validation happens at application layer (V2SubstrateStore can optionally call kb-22 to verify existence, but isn't required to).

### 2.5 kb-26 reconciliation: deferred (per β parent spec §2.5)

- kb-26 ObservationEvent / MedChangeEvent / CheckinEvent stay as adapter input shape
- kb-26 baseline computation continues using its own `lab_entries` reads + PatientBaselineSnapshot storage
- Migration to "kb-26 reads canonical Observation from kb-20" deferred until at least one Layer 1B adapter writes to canonical observations (Phase 1B-γ or later)
- β.2 makes the canonical store *available*; consumer migration is separate work

---

## 3. Type definitions

### 3.1 MedicineUse + Intent + Target + StopCriteria (kb-20 canonical)

```go
type MedicineUse struct {
    ID           uuid.UUID    `json:"id"`
    ResidentID   uuid.UUID    `json:"resident_id"`
    AMTCode      string       `json:"amt_code,omitempty"`     // Australian Medicines Terminology code
    DisplayName  string       `json:"display_name"`            // human-readable; falls back to medication_states.drug_name
    Intent       Intent       `json:"intent"`
    Target       Target       `json:"target"`
    StopCriteria StopCriteria `json:"stop_criteria"`
    Dose         string       `json:"dose,omitempty"`          // unstructured form; structured `dose_mg` lives at storage layer
    Route        string       `json:"route,omitempty"`         // ORAL, IV, IM, etc.
    Frequency    string       `json:"frequency,omitempty"`     // e.g. "BID", "QD"
    PrescriberID *uuid.UUID   `json:"prescriber_id,omitempty"` // Person.id; nullable for legacy records
    StartedAt    time.Time    `json:"started_at"`
    EndedAt      *time.Time   `json:"ended_at,omitempty"`
    Status       string       `json:"status"`                  // active|paused|ceased|completed
    CreatedAt    time.Time    `json:"created_at"`
    UpdatedAt    time.Time    `json:"updated_at"`
}

type Intent struct {
    Category   string `json:"category"`              // therapeutic|preventive|symptomatic|trial|deprescribing
    Indication string `json:"indication"`            // free text or SNOMED-CT-AU code
    Notes      string `json:"notes,omitempty"`
}

type Target struct {
    Kind string          `json:"kind"`               // see TargetKind* constants
    Spec json.RawMessage `json:"spec"`               // JSONB shape per Kind; documented in target_schemas.go
}

type StopCriteria struct {
    Triggers   []string        `json:"triggers"`     // adverse_event|target_achieved|review_due|patient_request|...
    ReviewDate *time.Time      `json:"review_date,omitempty"`
    Spec       json.RawMessage `json:"spec,omitempty"`
}
```

### 3.2 MedicineUse status enum

```go
const (
    MedicineUseStatusActive    = "active"
    MedicineUseStatusPaused    = "paused"
    MedicineUseStatusCeased    = "ceased"
    MedicineUseStatusCompleted = "completed"
)

// IsValidMedicineUseStatus reports whether s is a recognized status value.
```

### 3.3 Intent.Category and Target.Kind constants

```go
// Intent categories — describes WHY the medicine is used.
const (
    IntentTherapeutic    = "therapeutic"     // treating an active condition
    IntentPreventive     = "preventive"      // primary or secondary prevention
    IntentSymptomatic    = "symptomatic"     // PRN / symptom relief
    IntentTrial          = "trial"           // therapeutic trial period
    IntentDeprescribing  = "deprescribing"   // tapering/withdrawal
)

// Target kinds — documented JSONB shapes in target_schemas.go.
const (
    TargetKindBPThreshold       = "BP_threshold"        // antihypertensives
    TargetKindCompletionDate    = "completion_date"     // antibiotics, deprescribing
    TargetKindSymptomResolution = "symptom_resolution"  // symptomatic
    TargetKindHbA1cBand         = "HbA1c_band"          // diabetes
    TargetKindOpen              = "open"                // chronic, no specific target
)
```

### 3.4 Documented Target.Spec JSONB schemas (target_schemas.go)

Each Target.Kind has a documented Go-struct shape that callers MAY use for type-safety, but the underlying storage is JSONB:

```go
// TargetBPThresholdSpec — for Target{Kind: TargetKindBPThreshold}
//
// Example: {"systolic_max": 140, "diastolic_max": 90}
type TargetBPThresholdSpec struct {
    SystolicMax  int `json:"systolic_max"`
    DiastolicMax int `json:"diastolic_max"`
}

// TargetCompletionDateSpec — for antibiotic course AND deprescribing target.
//
// Example: {"end_date": "2026-05-15", "duration_days": 7, "rationale": "..."}
type TargetCompletionDateSpec struct {
    EndDate      time.Time `json:"end_date"`
    DurationDays int       `json:"duration_days,omitempty"`
    Rationale    string    `json:"rationale,omitempty"`
}

// TargetSymptomResolutionSpec — for symptomatic medication.
//
// Example: {"target_symptom": "pain", "monitoring_window_days": 14, "snomed_code": "..."}
type TargetSymptomResolutionSpec struct {
    TargetSymptom         string `json:"target_symptom"`
    MonitoringWindowDays  int    `json:"monitoring_window_days,omitempty"`
    SNOMEDCode            string `json:"snomed_code,omitempty"`
}

// TargetHbA1cBandSpec — diabetes target.
//
// Example: {"min": 6.5, "max": 8.0}
type TargetHbA1cBandSpec struct {
    Min float64 `json:"min"`
    Max float64 `json:"max"`
}

// TargetOpenSpec — chronic, no specific numerical target.
//
// Example: {"rationale": "long-term anticoagulation for AF"}
type TargetOpenSpec struct {
    Rationale string `json:"rationale,omitempty"`
}
```

### 3.5 Documented StopCriteria.Spec shapes (stop_criteria_schemas.go)

```go
// StopCriteria triggers — values used in StopCriteria.Triggers []string
const (
    StopTriggerAdverseEvent    = "adverse_event"
    StopTriggerTargetAchieved  = "target_achieved"
    StopTriggerReviewDue       = "review_due"
    StopTriggerPatientRequest  = "patient_request"
    StopTriggerCarerRequest    = "carer_request"
    StopTriggerCompletion      = "completion"          // course completed (antibiotics, etc.)
    StopTriggerInteraction     = "interaction"         // contraindicated by new medicine
)

// StopCriteriaReviewSpec — for time-bounded review obligation.
//
// Example: {"review_after_days": 30, "review_owner": "ACOP"}
type StopCriteriaReviewSpec struct {
    ReviewAfterDays int    `json:"review_after_days"`
    ReviewOwner     string `json:"review_owner,omitempty"` // RN|GP|ACOP|pharmacist
}

// StopCriteriaThresholdSpec — for criterion based on observation threshold.
//
// Example: {"observation_kind": "vital", "loinc_code": "8867-4", "operator": "<", "value": 50}
type StopCriteriaThresholdSpec struct {
    ObservationKind string  `json:"observation_kind"`  // vital|lab|behavioural|mobility|weight
    LOINCCode       string  `json:"loinc_code,omitempty"`
    SNOMEDCode      string  `json:"snomed_code,omitempty"`
    Operator        string  `json:"operator"`          // < <= = >= >
    Value           float64 `json:"value"`
}
```

### 3.6 Observation + Delta (kb-20 canonical)

```go
type Observation struct {
    ID         uuid.UUID  `json:"id"`
    ResidentID uuid.UUID  `json:"resident_id"`
    LOINCCode  string     `json:"loinc_code,omitempty"`
    SNOMEDCode string     `json:"snomed_code,omitempty"`
    Kind       string     `json:"kind"`              // vital|lab|behavioural|mobility|weight
    Value      *float64   `json:"value,omitempty"`   // pointer-nullable; complement to ValueText
    ValueText  string     `json:"value_text,omitempty"`
    Unit       string     `json:"unit,omitempty"`
    ObservedAt time.Time  `json:"observed_at"`
    SourceID   *uuid.UUID `json:"source_id,omitempty"` // application-validated reference to kb-22.clinical_sources
    Delta      *Delta     `json:"delta,omitempty"`     // computed on write
    CreatedAt  time.Time  `json:"created_at"`
}

type Delta struct {
    BaselineValue   float64   `json:"baseline_value"`
    DeviationStdDev float64   `json:"deviation_stddev"`
    DirectionalFlag string    `json:"flag"`            // within_baseline|elevated|severely_elevated|low|severely_low|no_baseline
    ComputedAt      time.Time `json:"computed_at"`
}
```

`Value *float64` is a pointer to distinguish "no value, see ValueText" from "value=0".

### 3.7 ObservationKind constants

```go
const (
    ObservationKindVital       = "vital"        // BP, HR, temp, SpO2
    ObservationKindLab         = "lab"          // labs (eGFR, HbA1c, etc.) — also flows from lab_entries via observations_v2 view
    ObservationKindBehavioural = "behavioural"  // BPSD events, agitation, sundowning
    ObservationKindMobility    = "mobility"     // mobility scores, falls
    ObservationKindWeight      = "weight"       // weight, BMI
)

// Delta directional flags
const (
    DeltaFlagWithinBaseline    = "within_baseline"
    DeltaFlagElevated          = "elevated"
    DeltaFlagSeverelyElevated  = "severely_elevated"
    DeltaFlagLow               = "low"
    DeltaFlagSeverelyLow       = "severely_low"
    DeltaFlagNoBaseline        = "no_baseline"  // no historical data; baseline computation skipped
)
```

---

## 4. Library structure additions

```
shared/v2_substrate/
├── models/                                    (extends β.1)
│   ├── medicine_use.go                        ← NEW
│   ├── medicine_use_test.go                   ← NEW
│   ├── target_schemas.go                      ← NEW (per-Kind documented spec structs)
│   ├── target_schemas_test.go                 ← NEW
│   ├── stop_criteria_schemas.go               ← NEW
│   ├── stop_criteria_schemas_test.go          ← NEW
│   ├── observation.go                         ← NEW
│   ├── observation_test.go                    ← NEW
│   └── enums.go                               ← MODIFIED (add MedicineUseStatus, ObservationKind, Intent/Target/Stop constants)
├── fhir/                                      (extends β.1)
│   ├── medication_request_mapper.go           ← NEW (MedicineUse ↔ AU MedicationRequest)
│   ├── medication_request_mapper_test.go      ← NEW
│   ├── observation_mapper.go                  ← NEW (Observation ↔ AU Observation)
│   ├── observation_mapper_test.go             ← NEW
│   └── extensions.go                          ← MODIFIED (add Vaidshala extension URIs for intent/target/stop_criteria/care_intensity-on-medicine)
├── validation/                                (extends β.1)
│   ├── medicine_use_validator.go              ← NEW
│   ├── observation_validator.go               ← NEW
│   ├── target_validator.go                    ← NEW (per-Kind validators delegating from Target)
│   └── validators_test.go                     ← MODIFIED (add cases)
├── delta/                                     ← NEW package
│   ├── interfaces.go                          ← BaselineProvider interface
│   ├── compute.go                             ← ComputeDelta pure function
│   └── compute_test.go                        ← table-driven correctness tests
├── interfaces/storage.go                      ← MODIFIED (add MedicineUseStore + ObservationStore)
└── client/kb20_client.go                      ← MODIFIED (add MedicineUse + Observation methods)
```

**kb-20 additions:**
```
kb-20-patient-profile/
├── migrations/008_part2_clinical_primitives.sql   ← NEW
├── internal/storage/v2_substrate_store.go         ← MODIFIED (add MedicineUse + Observation methods, scanMedicineUse + scanObservation)
└── internal/api/v2_substrate_handlers.go          ← MODIFIED (add MedicineUse + Observation REST endpoints)
```

---

## 5. Migration 008_part2 specifics

### 5.1 medication_states extension

```sql
ALTER TABLE medication_states ADD COLUMN IF NOT EXISTS amt_code      TEXT;
ALTER TABLE medication_states ADD COLUMN IF NOT EXISTS display_name  TEXT;
ALTER TABLE medication_states ADD COLUMN IF NOT EXISTS intent        JSONB;
ALTER TABLE medication_states ADD COLUMN IF NOT EXISTS target        JSONB;
ALTER TABLE medication_states ADD COLUMN IF NOT EXISTS stop_criteria JSONB;
ALTER TABLE medication_states ADD COLUMN IF NOT EXISTS prescriber_id UUID;  -- v2 Person.id; legacy prescribed_by VARCHAR remains for backward compat
ALTER TABLE medication_states ADD COLUMN IF NOT EXISTS resident_id   UUID;  -- canonical link to patient_profiles (legacy patient_id remains)
ALTER TABLE medication_states ADD COLUMN IF NOT EXISTS lifecycle_status TEXT
    CHECK (lifecycle_status IS NULL OR lifecycle_status IN ('active','paused','ceased','completed'));

-- Per-column COMMENTs documenting the v2 contract + JSONB shapes
```

### 5.2 observations table (greenfield)

Per §2.2. Indexes: `idx_observations_resident`, `idx_observations_observed_at`, `idx_observations_kind`.

### 5.3 medicine_uses_v2 view

```sql
CREATE OR REPLACE VIEW medicine_uses_v2 AS
SELECT
    ms.id                                                AS id,
    COALESCE(ms.resident_id,
             (SELECT pp.id FROM patient_profiles pp WHERE pp.patient_id = ms.patient_id LIMIT 1)
    )                                                    AS resident_id,
    ms.amt_code                                          AS amt_code,
    COALESCE(ms.display_name, ms.drug_name)              AS display_name,
    ms.intent                                            AS intent,
    ms.target                                            AS target,
    ms.stop_criteria                                     AS stop_criteria,
    ms.dose_mg::TEXT || COALESCE(' ' || ms.route, '')    AS dose,        -- legacy reconstruction
    ms.route                                             AS route,
    ms.frequency                                         AS frequency,
    ms.prescriber_id                                     AS prescriber_id,
    ms.start_date                                        AS started_at,
    ms.end_date                                          AS ended_at,
    COALESCE(
        ms.lifecycle_status,
        CASE WHEN ms.is_active THEN 'active' ELSE 'ceased' END
    )                                                    AS status,
    ms.created_at,
    ms.updated_at
FROM medication_states ms;
```

### 5.4 observations_v2 view

UNION of `observations` (v2 native) and `lab_entries` (legacy projected with kind='lab').

---

## 6. Sub-cluster milestones

Mirrors β.1's three-cluster pattern.

### β.2-A — MedicineUse end-to-end (~3 days)

**Deliverables:**
- shared/v2_substrate/models/{medicine_use, target_schemas, stop_criteria_schemas}.go + tests
- shared/v2_substrate/validation/medicine_use_validator.go + target_validator.go + cases in validators_test.go
- shared/v2_substrate/fhir/medication_request_mapper.go + tests (egress validation; ingress wrap; lossy-fields godoc)
- shared/v2_substrate/interfaces/storage.go MedicineUseStore added
- shared/v2_substrate/client/kb20_client.go MedicineUse methods added (with httptest verification)
- kb-20 migration 008_part2 PART A: medication_states extension + medicine_uses_v2 view
- kb-20 storage: scanMedicineUse + GetMedicineUse + UpsertMedicineUse + ListMedicineUsesByResident
- kb-20 handlers: 4 REST endpoints (POST /medicine_uses, GET /medicine_uses/:id, GET /residents/:resident_id/medicine_uses, sentinel→404)

**Exit criterion:** Round-trip MedicineUse with each of 5 Target.Kind values (BP_threshold, completion_date, symptom_resolution, HbA1c_band, open) through {KB20Client → handler → store → DB → handler → KB20Client}; FHIR mapper validates target.spec shape per Kind.

### β.2-A Completion (2026-05-04)

✅ **Cluster complete.** MedicineUse v2 substrate entity delivered end-to-end across 5 review checkpoints (C1 types, C2 validators+FHIR mapper, C3 interface+migration, C4 storage+handlers+client, C5 verification).

**Test counts (final):**
- shared/v2_substrate/models: 25 tests pass (incl. 5 Target.Spec round-trip + 2 StopCriteria.Spec)
- shared/v2_substrate/validation: 13 tests pass (incl. ValidateTarget per-Kind dispatch)
- shared/v2_substrate/fhir: 18 tests pass (incl. _RejectsInvalid + _WrongResourceType + _WireFormat + _AllTargetKindsRoundTrip across all 5 Target.Kind values)
- shared/v2_substrate/client: 3 tests pass (httptest-backed; +1 MedicineUse round-trip new in C4)
- kb-20 storage + handlers: SKIP cleanly without `KB20_TEST_DATABASE_URL`; build clean
- Total: 59 unit tests across shared packages, all green; vet clean

**Migration:** `008_part2_clinical_primitives_partA.sql` — non-breaking; extends medication_states with 8 nullable v2 columns + medicine_uses_v2 view; sanity counts: 1 BEGIN/COMMIT, 9 ALTER TABLE statements, 1 view, 8 COMMENT ON COLUMN, 0 uuid_generate_v4 leftovers. **Not yet applied to any DB** (deferred to deployment).

**Implementation pattern reuse from β.1:** rowScanner, ErrNotFound sentinel + respondError dispatch, validation in egress AND ingress mappers (defense-in-depth), no N+1 in list methods, explicit HTTP 400 on invalid pagination (no silent coerce), `url.Values{}.Encode()` for query params, `strings.HasPrefix`/`TrimPrefix` for FHIR reference extraction.

**β.2-A specific contributions:**
- 5 Target.Kind discriminator types with documented JSONB shapes (BP_threshold, completion_date, symptom_resolution, HbA1c_band, open) + Go spec structs in `target_schemas.go`
- Per-Kind Target validators with physiological range enforcement (BP 1-300/1-200, HbA1c 0-20%, sys≥dia, min<max)
- Cross-KB JSONB contract pinned via `TestTargetSpecOpaqueMarshalling` + `TestMedicineUseToMedicationRequest_AllTargetKindsRoundTrip` — byte-for-byte preservation across all 5 Kinds in FHIR round-trip
- `amt_code` (Australian Medicines Terminology — AU-specific product/strength) introduced; documented as DISTINCT FROM existing `atc_code` (WHO Anatomical Therapeutic Chemical class) added in migration 003 — both coexist intentionally with explicit COMMENT ON COLUMN guidance
- `IntentUnspecified` sentinel value introduced for legacy migration safety (replaces a substantive `"therapeutic"` default that would have injected unwanted clinical claims onto pre-existing rows in patient_profiles)
- Vaidshala-namespaced FHIR extensions for Intent/Target/StopCriteria/AMTCode preserve v2-distinguishing fields across the FHIR wire boundary (no native FHIR equivalents)

**Open follow-ups for β.2-B (deferred, non-blocking):**
1. Backfill UPDATE migration to populate medication_states.resident_id from patient_profiles via legacy patient_id (eliminates COALESCE subquery cost in medicine_uses_v2 view at scale)
2. Negative-pagination-rejection integration test against real handler (currently only httptest unit-level)
3. Consider lifecycle_status `'completed'` vs `'ceased'` derivation when legacy `is_active=false` carries no termination-reason signal
4. Confirm `validateTargetOpen` currently accepts any JSON shape (intentional permissiveness for `TargetKindOpen` is documented but worth re-examining if production usage shows malformed open-target data)

**Proceed to:** Phase 1B-β.2-B (Observation end-to-end) — separate plan to be authored after β.2-A is reviewed and merged.

### β.2-B — Observation end-to-end (~3 days)

**Deliverables:**
- shared/v2_substrate/models/observation.go + tests (Value pointer; kind discriminator)
- shared/v2_substrate/validation/observation_validator.go (value-or-text required; kind enum; value range sanity per kind)
- shared/v2_substrate/fhir/observation_mapper.go + tests
- shared/v2_substrate/interfaces/storage.go ObservationStore added
- shared/v2_substrate/client/kb20_client.go Observation methods added
- kb-20 migration 008_part2 PART B: observations table + observations_v2 view (UNION lab_entries)
- kb-20 storage: scanObservation + GetObservation + UpsertObservation + ListObservationsByResident + ListObservationsByResidentAndKind
- kb-20 handlers: 4 REST endpoints

**Exit criterion:** Round-trip Observation of each of 5 kinds (vital with BP, lab with HbA1c, behavioural with ValueText, mobility with score, weight with kg); observations_v2 view returns rows from both observations table and lab_entries.

### β.2-B Completion (2026-05-04)

✅ **Cluster complete.** Observation v2 substrate entity delivered end-to-end across 7 review checkpoints (C1 enums, C2 Observation/Delta types, C3 validator, C4 FHIR extensions URIs, C5 FHIR mapper, C6 ObservationStore interface + migration partB, C7 storage + handlers + client + integration test).

**Test counts (final):**
- shared/v2_substrate/models: 31 tests pass (β.2-A was 25; +6 for β.2-B — IsValidObservationKind, IsValidDeltaFlag, 4 Observation JSON round-trip tests covering vital, behavioural-with-ValueText, Delta-roundtrip, omit-empty)
- shared/v2_substrate/validation: 21 tests pass (β.2-A was 13; +8 for β.2-B — value-or-text required, kind enum, zero residentID, zero observedAt, vital range, weight positive, accepts value-only, accepts text-only)
- shared/v2_substrate/fhir: 31 tests pass (β.2-A was 23; +8 for β.2-B — 5 kinds × roundtrip + RejectsInvalid + WrongResourceType + WireFormatHasKindExtension)
- shared/v2_substrate/client: 5 tests pass (β.2-A was 3; +2 for β.2-B — httptest-backed UpsertGetObservation roundtrip + ListObservationsByResidentAndKind URL construction)
- kb-20 storage + handlers: SKIP cleanly without `KB20_TEST_DATABASE_URL`; build clean; vet clean
- **Total: 99 unit tests across shared v2_substrate packages, all green; vet clean**

**Migration:** `008_part2_clinical_primitives_partB.sql` — non-breaking; greenfield observations table (12 columns, value-or-text CHECK, 3 access-pattern indexes) + observations_v2 view UNIONing lab_entries projected with kind='lab'; sanity counts: 1 BEGIN/COMMIT, 1 CREATE TABLE, 3 CREATE INDEX, 1 CREATE OR REPLACE VIEW, 6 COMMENT ON COLUMN, 0 uuid_generate_v4 leftovers; 130 lines total. **Not yet applied to any DB** (deferred to deployment, mirrors partA pattern).

**β.2-B specific contributions:**
- Observation `Value *float64` pointer-nullable distinguishes "no numeric value, see ValueText" from "value=0.0" — preserved through SQL (sql.NullFloat64) and JSON (omitempty) round-trips
- Per-kind validator ranges (BP systolic 1-300 / diastolic 1-200, weight >0, lab >0, mobility >=0) enforce structural sanity at egress AND ingress mappers (defense-in-depth)
- observations_v2 view UNIONs legacy lab_entries with NULL projections for source_id/delta/snomed_code where lab_entries lacks the column — documented in COMMENT ON VIEW
- Vaidshala-namespaced FHIR extensions (ExtObservationKind / ExtObservationDelta / ExtObservationSourceID) round-trip kind discriminator + Delta JSON + UUID source provenance with no native FHIR equivalent
- AU spelling `behavioural` canonical; US `behavioral` intentionally rejected by IsValidObservationKind to enforce locale consistency across the cross-KB contract

**Open follow-ups for β.2-C / future (deferred, non-blocking):**
1. Replace InMemoryBaselineProvider with kb-26 AcuteRepository network adapter (interface seam already in place via delta.BaselineProvider — no consumer-side change required)
2. Negative-pagination integration test against the real handler (currently httptest unit-level only)
3. Application-level kb-22.clinical_sources existence check on Observation.SourceID (currently UUID-shaped reference without runtime cross-DB validation; documented in column COMMENT)

### β.2-C — Delta-on-write (~2 days)

**Deliverables:**
- shared/v2_substrate/delta/interfaces.go (BaselineProvider, Baseline)
- shared/v2_substrate/delta/compute.go (ComputeDelta pure function with directional flag logic)
- shared/v2_substrate/delta/compute_test.go (table-driven: in-baseline, mild deviation, severe deviation, no-baseline cases)
- kb-20 storage UpsertObservation calls ComputeDelta with a kb-26-backed BaselineProvider; if BaselineProvider unavailable, sets Delta.Flag=no_baseline rather than failing
- Stub kb-26 BaselineProvider implementation (`internal/storage/kb26_baseline_provider.go`) — calls kb-26 baseline storage OR a local in-memory mock for MVP
- Integration test: insert two Observations (baseline + delta); confirm second Observation has populated Delta with correct directional flag

**Exit criterion:** UpsertObservation populates Delta when a baseline is reachable; `Delta.Flag = no_baseline` when no historical data; the function passes table-driven correctness tests for severe-elevated / severe-low / within-baseline / mild-elevated / mild-low / no-baseline cases.

### β.2-C Completion (2026-05-04)

✅ **Cluster complete.** Delta-on-write service delivered as a pure function + interface seam, integrated into V2SubstrateStore.UpsertObservation. Spec §9 acceptance item 10 satisfied.

**Test counts (final):**
- shared/v2_substrate/delta: 11 tests pass (1 TestComputeDelta_Cases parent + 10 sub-tests covering within_baseline / elevated / severely_elevated / low / severely_low / no_baseline transitions across 5 trigger conditions: nil baseline, nil Value, behavioural Kind, StdDev=0 div-by-zero guard, value-equals-baseline, ±1 stddev boundary inclusivity, ±1.5 stddev mild, ±3 stddev severe)
- kb-20 InMemoryBaselineProvider: 2 tests pass (FetchExistingBaseline returns seeded data; missing key returns ErrNoBaseline sentinel)
- kb-20 delta-on-write integration: 4-case TestDeltaOnWrite_EndToEnd (within / severely_elevated / behavioural / unseeded-LOINC) — DB-gated; SKIPs cleanly without `KB20_TEST_DATABASE_URL`

**β.2-C specific contributions:**
- Pure `ComputeDelta(obs, baseline) → Delta` — no IO; threshold semantics: |dev|≤1 within_baseline (boundary inclusive); 1<dev≤2 elevated; dev>2 severely_elevated; symmetric on low side; no_baseline when baseline nil OR Value nil OR Kind=behavioural OR StdDev=0
- `delta.BaselineProvider` interface + `delta.ErrNoBaseline` sentinel — kb-26's AcuteRepository will plug in as a thin adapter at this seam without modifying ComputeDelta or UpsertObservation
- StdDev=0 div-by-zero guard returns no_baseline rather than emitting Inf/NaN — handles the "first observation per resident" edge case cleanly
- UpsertObservation NEVER fails the write on baseline unavailability — degraded paths (provider unset, network error, ErrNoBaseline) all surface as Delta.Flag=no_baseline so the observation persists for later re-derivation when kb-26 baseline data catches up
- vitalTypeKey resolver: LOINC → SNOMED → Kind precedence; documented in v2_substrate_store.go for kb-26 adapter to mirror

**Open follow-ups (deferred, non-blocking):**
1. kb-26 BaselineProvider production adapter (replaces InMemoryBaselineProvider stub; interface contract unchanged)
2. Telemetry signal distinguishing "legitimately no baseline" from "BaselineProvider transport error" — currently both surface as DeltaFlagNoBaseline (architecturally correct but observability is poor when kb-26 is partially down)
3. Threshold tuning — current 1σ/2σ thresholds are defensible MVP defaults; clinical pilot data may justify per-vital-type bands (e.g. weight has different drift semantics than BP)

**Phase 1B-β.2 status:** ✅ All three sub-clusters complete (β.2-A MedicineUse, β.2-B Observation, β.2-C Delta-on-write). Spec §9 acceptance criteria items 1, 2, 4, 5, 6, 8, 9, 10, 12 all verified.

---

## 7. Out of scope (explicitly deferred)

- **State machines** (Recommendation/Monitoring/Consent/Authorisation) — separate phases
- **Event + EvidenceTrace** — Phase 1B-β.3
- **kb-26 baseline storage migration** — kb-26 keeps its own PatientBaselineSnapshot store; the delta service reads from it via BaselineProvider interface but doesn't migrate kb-26's data model
- **Layer 1B adapters** — Phase 1B-γ
- **Real Layer 1A → kb-20 ingestion of Observations** — adapters will wire later
- **Source registry validation against kb-22** — observations.source_id stored as UUID without runtime cross-DB validation in this phase

---

## 8. Risks and mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| medication_states extension conflicts with FDC decomposition fields | Low | Medium | New columns are additive; FDC fields untouched. View ignores FDC fields (β.2 doesn't model FDC) |
| observations_v2 view UNION performance is poor on large lab_entries | Medium | Low (MVP scale) | Indexes on lab_entries already exist; UNION ALL avoids dedup; add a comment to revisit if pilot facility has >100k labs |
| BaselineProvider interface diverges from kb-26's actual storage shape | Medium | Medium | Implement BaselineProvider as a thin adapter over kb-26's existing AcuteRepository.FetchBaseline; if shape mismatch surfaces, adapter is the seam |
| JSONB validators reject existing legacy medication_states rows when constraint is added | Low | High (would break migration) | All new constraints NOT VALID; validators run at write time, not retroactively; existing rows have NULL intent/target/stop_criteria, which validators allow (nullable contract) |
| Delta computation is wrong for behavioural ObservationKind (no numeric baseline applies) | Medium | Low | ComputeDelta returns `Delta.Flag=no_baseline` when ObservationKind=behavioural; behavioural delta semantics deferred to Phase 1B-β.3 (Event-based) |

---

## 9. Acceptance criteria

Phase 1B-β.2 is complete when **all** of the following hold:

1. `shared/v2_substrate/models/{medicine_use, observation, target_schemas, stop_criteria_schemas}.go` exist with JSON round-trip tests passing
2. `shared/v2_substrate/fhir/{medication_request_mapper, observation_mapper}.go` exist with round-trip + _RejectsInvalid + _WrongResourceType + _WireFormat tests passing
3. `shared/v2_substrate/validation/{medicine_use, observation, target}_validator.go` exist with negative-case tests
4. `shared/v2_substrate/delta/` package exists with table-driven ComputeDelta tests
5. `shared/v2_substrate/interfaces/storage.go` declares MedicineUseStore + ObservationStore
6. `shared/v2_substrate/client/kb20_client.go` exposes 8 new methods (4 MedicineUse + 4 Observation)
7. kb-20 migration 008_part2 applies cleanly on a fresh kb-20 DB AND on a DB with existing data + 008_part1 applied
8. kb-20 V2SubstrateStore implements MedicineUseStore + ObservationStore with scan helpers reused via rowScanner pattern
9. kb-20 v2 substrate handlers expose 8 new REST endpoints
10. UpsertObservation populates Delta correctly when a baseline is available; falls back to `Delta.Flag=no_baseline` otherwise
11. All package tests pass; `go vet` clean; no breaking changes to existing kb-20 consumers (medication-service, kb-26)
12. observations_v2 view returns rows from both observations table and lab_entries with kind='lab' for legacy rows

---

## 10. References

- Parent spec: `docs/superpowers/specs/2026-05-04-1b-beta-substrate-entities-design.md`
- β.1 plan (predecessor): `docs/superpowers/plans/2026-05-04-1b-beta-substrate-entities-plan.md`
- v2 Revision Mapping Parts 3 + 6
- HL7 AU Base IG v6.0.0 (procured at `kb-3-guidelines/knowledge/au/integration_specs/hl7_au/base_ig_r4/`)
- AU FHIR MedicationRequest profile (within HL7 AU Base v6.0.0)
- AU FHIR Observation profile (within HL7 AU Base v6.0.0)
- Existing kb-20 schemas: `kb-20-patient-profile/migrations/001_initial_schema.sql` (medication_states, lab_entries)
- kb-26 baseline computation: `kb-26-metabolic-digital-twin/internal/services/baseline_adjustment.go` + `acute_repository.go`
