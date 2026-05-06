# Phase 1B-β.2-B + β.2-C — Observation + Delta-on-write Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver canonical Observation entity (kb-20-owned, kind discriminator over vital/lab/behavioural/mobility/weight) and Delta-on-write service (BaselineProvider interface + pure ComputeDelta) so that UpsertObservation populates Delta when a baseline is reachable and falls back to Delta.Flag=no_baseline otherwise.

**Architecture:** Mirrors β.1 + β.2-A. Internal Go types in `shared/v2_substrate/`, FHIR mappers at boundaries, validation at egress AND ingress (defense-in-depth), kb-20 owns canonical row, observations table greenfield with kind discriminator, lab_entries unchanged + UNION view. Delta computed at service layer (not DB trigger) via BaselineProvider interface so kb-26's AcuteRepository plugs in as a thin adapter.

**Tech Stack:** Go 1.22+, raw SQL + lib/pq, gin REST, Postgres 16, github.com/google/uuid, github.com/cardiofit/shared module path.

**Spec:** `docs/superpowers/specs/2026-05-04-1b-beta-2-clinical-primitives-design.md` (focus §2.1, §2.2, §2.4, §3.6, §3.7, §4, §5.2, §5.4, §6.β.2-B, §6.β.2-C, §9 items 1, 2, 4, 5, 6, 8, 9, 10, 12).

**Predecessor:** β.2-A (MedicineUse) — complete; this plan reuses the rowScanner / ErrNotFound / respondError / nilIfEmpty / doJSON patterns established there.

**Scope:** Observation entity end-to-end + Delta-on-write service. Out of scope: state machines, Event/EvidenceTrace (β.3), kb-26 baseline-store migration (consumer migration deferred), Layer 1B adapters (γ).

**Exit criterion:** UpsertObservation populates Delta from a kb-26-backed BaselineProvider when baseline data exists; falls back to `Delta.Flag=no_baseline` when the provider returns ErrNoBaseline OR Observation.Value is nil OR Kind=behavioural. Round-trip Observation of all 5 kinds through {KB20Client → handler → store → DB → handler → KB20Client → FHIR mapper}; observations_v2 view returns rows from both observations table and lab_entries with kind='lab' for legacy rows.

---

## File Structure

**Created:**
- `<kbs>/shared/v2_substrate/models/observation.go` — Observation + Delta types
- `<kbs>/shared/v2_substrate/models/observation_test.go` — JSON round-trip + Value-nil cases
- `<kbs>/shared/v2_substrate/validation/observation_validator.go` — value-or-text + kind enum + per-kind ranges
- `<kbs>/shared/v2_substrate/fhir/observation_mapper.go` — Observation ↔ AU FHIR Observation
- `<kbs>/shared/v2_substrate/fhir/observation_mapper_test.go` — round-trip per kind + _RejectsInvalid + _WrongResourceType + _WireFormat
- `<kbs>/shared/v2_substrate/delta/interfaces.go` — Baseline + BaselineProvider + ErrNoBaseline
- `<kbs>/shared/v2_substrate/delta/compute.go` — ComputeDelta pure function
- `<kbs>/shared/v2_substrate/delta/compute_test.go` — table-driven flag-correctness tests
- `<kbs>/kb-20-patient-profile/migrations/008_part2_clinical_primitives_partB.sql` — observations table + observations_v2 view
- `<kbs>/kb-20-patient-profile/internal/storage/kb26_baseline_provider.go` — in-memory MVP BaselineProvider
- `<kbs>/kb-20-patient-profile/internal/storage/kb26_baseline_provider_test.go`
- `<kbs>/kb-20-patient-profile/internal/storage/v2_substrate_observation_test.go` — DB-gated Observation storage tests
- `<kbs>/kb-20-patient-profile/internal/api/v2_substrate_observation_handlers_test.go` — httptest-backed handler tests + Delta integration test

**Modified:**
- `<kbs>/shared/v2_substrate/models/enums.go` — verify ObservationKind* + DeltaFlag* constants present (Task 1 verifies; adds them only if missing)
- `<kbs>/shared/v2_substrate/fhir/extensions.go` — verify Vaidshala extension URI for Delta + ObservedKind already declared; add if missing (Task 4 verifies)
- `<kbs>/shared/v2_substrate/validation/validators_test.go` — add Observation validator cases
- `<kbs>/shared/v2_substrate/interfaces/storage.go` — add ObservationStore interface alongside existing MedicineUseStore
- `<kbs>/shared/v2_substrate/client/kb20_client.go` — add 4 Observation methods alongside existing MedicineUse methods
- `<kbs>/kb-20-patient-profile/internal/storage/v2_substrate_store.go` — add observationColumns + scanObservation + GetObservation + UpsertObservation (calls ComputeDelta) + ListObservationsByResident + ListObservationsByResidentAndKind
- `<kbs>/kb-20-patient-profile/internal/api/v2_substrate_handlers.go` — add 4 routes around line 60, parallel to MedicineUse block; pass BaselineProvider through V2SubstrateHandlers constructor

**Convention:** All paths relative to `/Volumes/Vaidshala/cardiofit/`. Within tasks the prefix `<repo>/backend/shared-infrastructure/knowledge-base-services/` is shortened to `<kbs>` for readability; tasks always show the full path in the file paths section.

**Module path:** `github.com/cardiofit/shared` (verified by reading `<kbs>/shared/go.mod`).

---

## Task 1: Verify ObservationKind + DeltaFlag enum constants (and add if missing)

**Files:**
- Verify (or modify): `<kbs>/shared/v2_substrate/models/enums.go`
- Verify (or modify): `<kbs>/shared/v2_substrate/models/enums_test.go`

- [ ] **Step 1: Inspect existing enums.go for the constants**

```bash
grep -n "ObservationKind\|DeltaFlag" /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/models/enums.go || echo "NOT PRESENT"
```

Expected outcomes:
- If grep returns matches for `ObservationKindVital`, `ObservationKindLab`, `ObservationKindBehavioural`, `ObservationKindMobility`, `ObservationKindWeight`, `IsValidObservationKind`, `DeltaFlagWithinBaseline`, `DeltaFlagElevated`, `DeltaFlagSeverelyElevated`, `DeltaFlagLow`, `DeltaFlagSeverelyLow`, `DeltaFlagNoBaseline`, `IsValidDeltaFlag` — skip Steps 2-5 (constants already shipped in β.2-A) and proceed to **Step 6** (verify-only test) only.
- If `NOT PRESENT` or partial — proceed with Steps 2-5.

- [ ] **Step 2: Write the failing test (only if Step 1 said NOT PRESENT)**

Path: `<kbs>/shared/v2_substrate/models/enums_test.go` (append to existing file)

```go
func TestIsValidObservationKind(t *testing.T) {
    cases := []struct {
        in   string
        want bool
    }{
        {"vital", true},
        {"lab", true},
        {"behavioural", true},
        {"mobility", true},
        {"weight", true},
        {"", false},
        {"behavioral", false}, // US spelling rejected — AU spelling only
        {"unknown", false},
    }
    for _, c := range cases {
        if got := IsValidObservationKind(c.in); got != c.want {
            t.Errorf("IsValidObservationKind(%q) = %v, want %v", c.in, got, c.want)
        }
    }
}

func TestIsValidDeltaFlag(t *testing.T) {
    valid := []string{"within_baseline", "elevated", "severely_elevated", "low", "severely_low", "no_baseline"}
    for _, f := range valid {
        if !IsValidDeltaFlag(f) {
            t.Errorf("IsValidDeltaFlag(%q) = false, want true", f)
        }
    }
    if IsValidDeltaFlag("") {
        t.Errorf("IsValidDeltaFlag(\"\") = true, want false")
    }
    if IsValidDeltaFlag("normal") {
        t.Errorf("IsValidDeltaFlag(\"normal\") = true, want false (must use within_baseline)")
    }
}
```

- [ ] **Step 3: Run test — expect FAIL**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && \
  go test ./v2_substrate/models/... -run "TestIsValidObservationKind|TestIsValidDeltaFlag" -v
```

Expected: FAIL with `undefined: IsValidObservationKind` (or DeltaFlag).

- [ ] **Step 4: Add the constants and validators (only if Step 1 said NOT PRESENT)**

Path: `<kbs>/shared/v2_substrate/models/enums.go` (append at end of file)

```go
// ObservationKind discriminates the row kind in the observations table.
// vital — BP, HR, temp, SpO2; lab — eGFR, HbA1c (also surfaced from lab_entries);
// behavioural — BPSD events, agitation; mobility — mobility scores, falls;
// weight — weight, BMI.
// Values are stored as plain strings to round-trip cleanly through FHIR code elements.
const (
    ObservationKindVital       = "vital"
    ObservationKindLab         = "lab"
    ObservationKindBehavioural = "behavioural"
    ObservationKindMobility    = "mobility"
    ObservationKindWeight      = "weight"
)

// IsValidObservationKind reports whether s is one of the recognized
// ObservationKind values. AU spelling ("behavioural") is canonical;
// US "behavioral" is intentionally rejected.
func IsValidObservationKind(s string) bool {
    switch s {
    case ObservationKindVital, ObservationKindLab,
        ObservationKindBehavioural, ObservationKindMobility, ObservationKindWeight:
        return true
    }
    return false
}

// DeltaFlag enumerates the directional flag emitted by the delta-on-write
// service. Threshold semantics live in shared/v2_substrate/delta/compute.go.
// Values are stored as plain strings to round-trip cleanly through FHIR code elements.
const (
    DeltaFlagWithinBaseline   = "within_baseline"
    DeltaFlagElevated         = "elevated"
    DeltaFlagSeverelyElevated = "severely_elevated"
    DeltaFlagLow              = "low"
    DeltaFlagSeverelyLow      = "severely_low"
    DeltaFlagNoBaseline       = "no_baseline"
)

// IsValidDeltaFlag reports whether s is one of the recognized DeltaFlag values.
func IsValidDeltaFlag(s string) bool {
    switch s {
    case DeltaFlagWithinBaseline, DeltaFlagElevated, DeltaFlagSeverelyElevated,
        DeltaFlagLow, DeltaFlagSeverelyLow, DeltaFlagNoBaseline:
        return true
    }
    return false
}
```

- [ ] **Step 5: Run test — expect PASS**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && \
  go test ./v2_substrate/models/... -run "TestIsValidObservationKind|TestIsValidDeltaFlag" -v
```

Expected: PASS for both tests.

- [ ] **Step 6: Verify-only path (if Step 1 said constants ARE present)**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && \
  go test ./v2_substrate/models/... -v 2>&1 | tail -20
```

Expected: existing TestIsValidObservationKind / TestIsValidDeltaFlag (or whatever β.2-A named them) pass; record the test names in the commit message rather than adding duplicates.

- [ ] **Step 7: Commit (only if anything changed in this task)**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/models/enums.go \
        backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/models/enums_test.go
git commit -m "feat(v2_substrate): add ObservationKind + DeltaFlag enum constants for β.2-B"
```

If Step 1 found constants already shipped in β.2-A: skip the commit, add a note `[verified-only]` to the next task's commit message.

---

## Task 2: Define Observation + Delta types

**Files:**
- Create: `<kbs>/shared/v2_substrate/models/observation.go`
- Create: `<kbs>/shared/v2_substrate/models/observation_test.go`

- [ ] **Step 1: Write the failing test**

Path: `<kbs>/shared/v2_substrate/models/observation_test.go`

```go
package models

import (
    "encoding/json"
    "strings"
    "testing"
    "time"

    "github.com/google/uuid"
)

func TestObservationJSONRoundTripVital(t *testing.T) {
    val := 142.0
    src := uuid.New()
    in := Observation{
        ID:         uuid.New(),
        ResidentID: uuid.New(),
        LOINCCode:  "8480-6",
        Kind:       ObservationKindVital,
        Value:      &val,
        Unit:       "mmHg",
        ObservedAt: time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC),
        SourceID:   &src,
        CreatedAt:  time.Now().UTC(),
    }
    b, err := json.Marshal(in)
    if err != nil {
        t.Fatalf("marshal: %v", err)
    }
    var out Observation
    if err := json.Unmarshal(b, &out); err != nil {
        t.Fatalf("unmarshal: %v", err)
    }
    if out.ID != in.ID || out.Kind != in.Kind || out.LOINCCode != in.LOINCCode {
        t.Errorf("round-trip mismatch: got %+v want %+v", out, in)
    }
    if out.Value == nil || *out.Value != *in.Value {
        t.Errorf("Value pointer round-trip lost: got %v want %v", out.Value, in.Value)
    }
    if out.SourceID == nil || *out.SourceID != src {
        t.Errorf("SourceID round-trip lost: got %v want %v", out.SourceID, src)
    }
}

func TestObservationJSONRoundTripBehaviouralValueText(t *testing.T) {
    in := Observation{
        ID:         uuid.New(),
        ResidentID: uuid.New(),
        Kind:       ObservationKindBehavioural,
        Value:      nil, // intentionally nil — ValueText carries the data
        ValueText:  "agitation episode 14:30, paced corridor 22 minutes",
        ObservedAt: time.Now().UTC(),
        CreatedAt:  time.Now().UTC(),
    }
    b, err := json.Marshal(in)
    if err != nil {
        t.Fatalf("marshal: %v", err)
    }
    if strings.Contains(string(b), `"value":`) && !strings.Contains(string(b), `"value":null`) {
        // omitempty on *float64 nil should drop the key entirely
        t.Errorf("value should be omitted when nil, got: %s", string(b))
    }
    var out Observation
    if err := json.Unmarshal(b, &out); err != nil {
        t.Fatalf("unmarshal: %v", err)
    }
    if out.Value != nil {
        t.Errorf("Value should remain nil after round-trip, got %v", *out.Value)
    }
    if out.ValueText != in.ValueText {
        t.Errorf("ValueText round-trip lost: got %q want %q", out.ValueText, in.ValueText)
    }
}

func TestObservationDeltaRoundTrip(t *testing.T) {
    val := 8.2
    in := Observation{
        ID:         uuid.New(),
        ResidentID: uuid.New(),
        Kind:       ObservationKindLab,
        LOINCCode:  "4548-4",
        Value:      &val,
        Unit:       "%",
        ObservedAt: time.Now().UTC(),
        Delta: &Delta{
            BaselineValue:   7.0,
            DeviationStdDev: 2.4,
            DirectionalFlag: DeltaFlagSeverelyElevated,
            ComputedAt:      time.Now().UTC(),
        },
        CreatedAt: time.Now().UTC(),
    }
    b, _ := json.Marshal(in)
    var out Observation
    if err := json.Unmarshal(b, &out); err != nil {
        t.Fatalf("unmarshal: %v", err)
    }
    if out.Delta == nil {
        t.Fatalf("Delta lost in round-trip")
    }
    if out.Delta.DirectionalFlag != DeltaFlagSeverelyElevated {
        t.Errorf("Delta.DirectionalFlag: got %q want %q", out.Delta.DirectionalFlag, DeltaFlagSeverelyElevated)
    }
    if out.Delta.BaselineValue != 7.0 || out.Delta.DeviationStdDev != 2.4 {
        t.Errorf("Delta numeric fields drifted: %+v", out.Delta)
    }
}

func TestObservationOmitsEmptyOptionalFields(t *testing.T) {
    in := Observation{
        ID:         uuid.New(),
        ResidentID: uuid.New(),
        Kind:       ObservationKindWeight,
        ValueText:  "78.4",
        ObservedAt: time.Now().UTC(),
    }
    b, _ := json.Marshal(in)
    s := string(b)
    for _, k := range []string{`"loinc_code"`, `"snomed_code"`, `"unit"`, `"source_id"`, `"delta"`} {
        if strings.Contains(s, k) {
            t.Errorf("expected %s to be omitted, got: %s", k, s)
        }
    }
}
```

- [ ] **Step 2: Run test — expect FAIL**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && \
  go test ./v2_substrate/models/... -run "TestObservation" -v
```

Expected: FAIL with `undefined: Observation` or `undefined: Delta`.

- [ ] **Step 3: Write the type definitions**

Path: `<kbs>/shared/v2_substrate/models/observation.go`

```go
package models

import (
    "time"

    "github.com/google/uuid"
)

// Observation represents a v2 substrate clinical observation for a Resident.
// Distinguished from kb-20's legacy lab_entries by the kind discriminator
// (vital | lab | behavioural | mobility | weight) and the optional Delta
// computed at write time by the delta-on-write service.
//
// Value is *float64 (pointer-nullable) to distinguish "no numeric value, see
// ValueText" from "value=0.0". One of Value or ValueText MUST be present
// (enforced by the DB CHECK constraint observations_value_or_text + by the
// validator in shared/v2_substrate/validation/observation_validator.go).
//
// Canonical storage: kb-20-patient-profile (observations table, greenfield in
// migration 008_part2_partB; observations_v2 view UNIONs lab_entries with
// kind='lab' for backward compatibility).
//
// FHIR boundary: maps to AU FHIR Observation at integration boundaries via
// shared/v2_substrate/fhir/observation_mapper.go. Delta has no native FHIR
// representation and is encoded as a Vaidshala-namespaced FHIR extension.
type Observation struct {
    ID         uuid.UUID  `json:"id"`
    ResidentID uuid.UUID  `json:"resident_id"`
    LOINCCode  string     `json:"loinc_code,omitempty"`
    SNOMEDCode string     `json:"snomed_code,omitempty"`
    Kind       string     `json:"kind"` // see ObservationKind* constants in enums.go
    Value      *float64   `json:"value,omitempty"`
    ValueText  string     `json:"value_text,omitempty"`
    Unit       string     `json:"unit,omitempty"`
    ObservedAt time.Time  `json:"observed_at"`
    SourceID   *uuid.UUID `json:"source_id,omitempty"` // application-validated UUID reference to kb-22.clinical_sources; no DB FK (cross-DB)
    Delta      *Delta     `json:"delta,omitempty"`     // populated on write by delta-on-write service
    CreatedAt  time.Time  `json:"created_at"`
}

// Delta is the directional deviation of an Observation from the resident's
// baseline. Populated at write time by shared/v2_substrate/delta/compute.go.
//
// DirectionalFlag is one of the DeltaFlag* constants. When the baseline is
// unavailable (no historical data, behavioural kind, or nil Value),
// DirectionalFlag is DeltaFlagNoBaseline and BaselineValue + DeviationStdDev
// are zero.
type Delta struct {
    BaselineValue   float64   `json:"baseline_value"`
    DeviationStdDev float64   `json:"deviation_stddev"`
    DirectionalFlag string    `json:"flag"` // see DeltaFlag* constants in enums.go
    ComputedAt      time.Time `json:"computed_at"`
}
```

- [ ] **Step 4: Run test — expect PASS**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && \
  go test ./v2_substrate/models/... -run "TestObservation" -v
```

Expected: PASS for all 4 Observation tests.

- [ ] **Step 5: Run full models package tests (regression)**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && \
  go test ./v2_substrate/models/... -v 2>&1 | tail -10
```

Expected: all existing model tests still pass; ok line shows new test count.

- [ ] **Step 6: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/models/observation.go \
        backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/models/observation_test.go
git commit -m "feat(v2_substrate): add Observation + Delta types with pointer-nullable Value and JSON round-trip tests"
```

---

## Task 3: Observation validator

**Files:**
- Create: `<kbs>/shared/v2_substrate/validation/observation_validator.go`
- Modify: `<kbs>/shared/v2_substrate/validation/validators_test.go` (append cases)

- [ ] **Step 1: Append failing tests to existing validators_test.go**

Path: `<kbs>/shared/v2_substrate/validation/validators_test.go` (append at end)

```go
func TestValidateObservationRequiresValueOrText(t *testing.T) {
    o := models.Observation{
        ID:         uuid.New(),
        ResidentID: uuid.New(),
        Kind:       models.ObservationKindVital,
        ObservedAt: time.Now(),
    }
    if err := ValidateObservation(o); err == nil {
        t.Errorf("expected error when both Value and ValueText empty; got nil")
    }
}

func TestValidateObservationAcceptsValueOnly(t *testing.T) {
    v := 120.0
    o := models.Observation{
        ID:         uuid.New(),
        ResidentID: uuid.New(),
        Kind:       models.ObservationKindVital,
        Value:      &v,
        Unit:       "mmHg",
        ObservedAt: time.Now(),
    }
    if err := ValidateObservation(o); err != nil {
        t.Errorf("expected pass for valid vital observation; got %v", err)
    }
}

func TestValidateObservationAcceptsValueTextOnly(t *testing.T) {
    o := models.Observation{
        ID:         uuid.New(),
        ResidentID: uuid.New(),
        Kind:       models.ObservationKindBehavioural,
        ValueText:  "agitation episode",
        ObservedAt: time.Now(),
    }
    if err := ValidateObservation(o); err != nil {
        t.Errorf("expected pass for behavioural with ValueText only; got %v", err)
    }
}

func TestValidateObservationRejectsInvalidKind(t *testing.T) {
    v := 1.0
    o := models.Observation{
        ID:         uuid.New(),
        ResidentID: uuid.New(),
        Kind:       "behavioral", // US spelling
        Value:      &v,
        ObservedAt: time.Now(),
    }
    if err := ValidateObservation(o); err == nil {
        t.Errorf("expected error for invalid kind; got nil")
    }
}

func TestValidateObservationRejectsZeroResidentID(t *testing.T) {
    v := 1.0
    o := models.Observation{
        ID:         uuid.New(),
        ResidentID: uuid.Nil,
        Kind:       models.ObservationKindLab,
        Value:      &v,
        ObservedAt: time.Now(),
    }
    if err := ValidateObservation(o); err == nil {
        t.Errorf("expected error for zero resident_id; got nil")
    }
}

func TestValidateObservationRejectsZeroObservedAt(t *testing.T) {
    v := 1.0
    o := models.Observation{
        ID:         uuid.New(),
        ResidentID: uuid.New(),
        Kind:       models.ObservationKindLab,
        Value:      &v,
        ObservedAt: time.Time{},
    }
    if err := ValidateObservation(o); err == nil {
        t.Errorf("expected error for zero observed_at; got nil")
    }
}

func TestValidateObservationVitalRange(t *testing.T) {
    bad := 999.0
    o := models.Observation{
        ID:         uuid.New(),
        ResidentID: uuid.New(),
        Kind:       models.ObservationKindVital,
        LOINCCode:  "8480-6", // systolic BP
        Value:      &bad,
        Unit:       "mmHg",
        ObservedAt: time.Now(),
    }
    if err := ValidateObservation(o); err == nil {
        t.Errorf("expected error for BP=999; got nil")
    }
    good := 130.0
    o.Value = &good
    if err := ValidateObservation(o); err != nil {
        t.Errorf("expected pass for BP=130; got %v", err)
    }
}

func TestValidateObservationWeightPositive(t *testing.T) {
    bad := 0.0
    o := models.Observation{
        ID:         uuid.New(),
        ResidentID: uuid.New(),
        Kind:       models.ObservationKindWeight,
        Value:      &bad,
        Unit:       "kg",
        ObservedAt: time.Now(),
    }
    if err := ValidateObservation(o); err == nil {
        t.Errorf("expected error for weight=0; got nil")
    }
}
```

- [ ] **Step 2: Run test — expect FAIL**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && \
  go test ./v2_substrate/validation/... -run "TestValidateObservation" -v
```

Expected: FAIL with `undefined: ValidateObservation`.

- [ ] **Step 3: Write the validator**

Path: `<kbs>/shared/v2_substrate/validation/observation_validator.go`

```go
package validation

import (
    "errors"
    "fmt"

    "github.com/google/uuid"

    "github.com/cardiofit/shared/v2_substrate/models"
)

// ValidateObservation reports any structural problem with o.
//
// Rules (mirrors DB observations_value_or_text CHECK + spec §3.6 + §9.10):
//   - ResidentID must be non-zero
//   - Kind must be one of the ObservationKind* enum values
//   - ObservedAt must be non-zero
//   - At least one of Value or ValueText must be set
//   - Per-kind sanity ranges on Value when present:
//       vital   — systolic BP (LOINC 8480-6) 1-300; diastolic BP (LOINC 8462-4) 1-200;
//                 default vital range 0-1000 (broad guard against absurd inputs)
//       lab     — value > 0 (labs are never <= 0; rejected values surface as ValidationStatus elsewhere)
//       weight  — value > 0
//       mobility — value >= 0
//       behavioural — Value is allowed but ValueText is the canonical carrier
func ValidateObservation(o models.Observation) error {
    if o.ResidentID == uuid.Nil {
        return errors.New("resident_id is required")
    }
    if !models.IsValidObservationKind(o.Kind) {
        return fmt.Errorf("invalid kind %q", o.Kind)
    }
    if o.ObservedAt.IsZero() {
        return errors.New("observed_at is required")
    }
    if o.Value == nil && o.ValueText == "" {
        return errors.New("one of value or value_text must be provided")
    }

    if o.Value != nil {
        v := *o.Value
        switch o.Kind {
        case models.ObservationKindVital:
            switch o.LOINCCode {
            case "8480-6": // systolic
                if v < 1 || v > 300 {
                    return fmt.Errorf("systolic BP %v out of range [1,300]", v)
                }
            case "8462-4": // diastolic
                if v < 1 || v > 200 {
                    return fmt.Errorf("diastolic BP %v out of range [1,200]", v)
                }
            default:
                if v < 0 || v > 1000 {
                    return fmt.Errorf("vital value %v out of broad range [0,1000]", v)
                }
            }
        case models.ObservationKindLab:
            if v <= 0 {
                return fmt.Errorf("lab value must be > 0, got %v", v)
            }
        case models.ObservationKindWeight:
            if v <= 0 {
                return fmt.Errorf("weight must be > 0, got %v", v)
            }
        case models.ObservationKindMobility:
            if v < 0 {
                return fmt.Errorf("mobility score must be >= 0, got %v", v)
            }
        }
    }
    return nil
}
```

- [ ] **Step 4: Run test — expect PASS**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && \
  go test ./v2_substrate/validation/... -v 2>&1 | tail -20
```

Expected: PASS for all new TestValidateObservation* tests; existing validator tests still green.

- [ ] **Step 5: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/validation/observation_validator.go \
        backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/validation/validators_test.go
git commit -m "feat(v2_substrate): Observation validator with value-or-text + per-kind range checks"
```

---

## Task 4: Verify FHIR extension URIs and add Vaidshala Observation extensions if missing

**Files:**
- Verify (or modify): `<kbs>/shared/v2_substrate/fhir/extensions.go`

- [ ] **Step 1: Inspect extensions.go for Observation extension URIs**

```bash
grep -n "ExtObservationDelta\|ExtObservationKind\|ExtObservationSourceID" /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/fhir/extensions.go || echo "NOT PRESENT"
```

If matches found: skip Steps 2-3, record `[verified-only]` in Task 5's commit.

- [ ] **Step 2: Add Vaidshala Observation extension URI constants (only if Step 1 said NOT PRESENT)**

Path: `<kbs>/shared/v2_substrate/fhir/extensions.go` (append after the existing MedicineUse extensions block, before the SystemRouteCode line)

```go
// Vaidshala FHIR extension URIs for Observation v2-distinguishing fields.
// AU FHIR Observation has no native equivalents for Vaidshala's kind
// discriminator, source provenance reference, or computed Delta — these
// are encoded as Vaidshala-namespaced extensions on the resource.
const (
    ExtObservationKind     = "https://vaidshala.health/fhir/StructureDefinition/observation-kind"
    ExtObservationDelta    = "https://vaidshala.health/fhir/StructureDefinition/observation-delta"
    ExtObservationSourceID = "https://vaidshala.health/fhir/StructureDefinition/observation-source-id"
)
```

- [ ] **Step 3: Run vet to confirm syntax**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && \
  go vet ./v2_substrate/fhir/...
```

Expected: no errors.

- [ ] **Step 4: Commit (only if anything changed)**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/fhir/extensions.go
git commit -m "feat(v2_substrate/fhir): Vaidshala extension URIs for Observation kind/delta/source_id"
```

---

## Task 5: Observation FHIR mapper

**Files:**
- Create: `<kbs>/shared/v2_substrate/fhir/observation_mapper.go`
- Create: `<kbs>/shared/v2_substrate/fhir/observation_mapper_test.go`

- [ ] **Step 1: Write the failing test (round-trip per kind + reject-invalid + wrong-resource-type + wire-format)**

Path: `<kbs>/shared/v2_substrate/fhir/observation_mapper_test.go`

```go
package fhir

import (
    "encoding/json"
    "strings"
    "testing"
    "time"

    "github.com/google/uuid"

    "github.com/cardiofit/shared/v2_substrate/models"
)

func TestObservationToAUObservation_VitalRoundTrip(t *testing.T) {
    val := 142.0
    in := models.Observation{
        ID:         uuid.New(),
        ResidentID: uuid.New(),
        LOINCCode:  "8480-6",
        Kind:       models.ObservationKindVital,
        Value:      &val,
        Unit:       "mmHg",
        ObservedAt: time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC),
    }
    fhir, err := ObservationToAUObservation(in)
    if err != nil {
        t.Fatalf("egress: %v", err)
    }
    if fhir["resourceType"] != "Observation" {
        t.Errorf("resourceType: got %v want Observation", fhir["resourceType"])
    }
    b, _ := json.Marshal(fhir)
    var rt map[string]interface{}
    _ = json.Unmarshal(b, &rt)
    out, err := AUObservationToObservation(rt)
    if err != nil {
        t.Fatalf("ingress: %v", err)
    }
    if out.LOINCCode != in.LOINCCode || out.Kind != in.Kind {
        t.Errorf("round-trip drift: got %+v want %+v", out, in)
    }
    if out.Value == nil || *out.Value != *in.Value {
        t.Errorf("Value lost: got %v want %v", out.Value, in.Value)
    }
}

func TestObservationToAUObservation_LabRoundTrip(t *testing.T) {
    val := 7.4
    in := models.Observation{
        ID: uuid.New(), ResidentID: uuid.New(),
        LOINCCode: "4548-4", Kind: models.ObservationKindLab,
        Value: &val, Unit: "%", ObservedAt: time.Now().UTC(),
    }
    fhir, err := ObservationToAUObservation(in)
    if err != nil {
        t.Fatalf("egress: %v", err)
    }
    b, _ := json.Marshal(fhir)
    var rt map[string]interface{}
    _ = json.Unmarshal(b, &rt)
    out, err := AUObservationToObservation(rt)
    if err != nil {
        t.Fatalf("ingress: %v", err)
    }
    if out.Kind != models.ObservationKindLab {
        t.Errorf("Kind lost: got %q", out.Kind)
    }
}

func TestObservationToAUObservation_BehaviouralValueText(t *testing.T) {
    in := models.Observation{
        ID: uuid.New(), ResidentID: uuid.New(),
        Kind: models.ObservationKindBehavioural,
        ValueText: "agitation episode 14:30",
        ObservedAt: time.Now().UTC(),
    }
    fhir, err := ObservationToAUObservation(in)
    if err != nil {
        t.Fatalf("egress: %v", err)
    }
    b, _ := json.Marshal(fhir)
    var rt map[string]interface{}
    _ = json.Unmarshal(b, &rt)
    out, err := AUObservationToObservation(rt)
    if err != nil {
        t.Fatalf("ingress: %v", err)
    }
    if out.Value != nil {
        t.Errorf("Value should be nil for behavioural; got %v", *out.Value)
    }
    if out.ValueText != in.ValueText {
        t.Errorf("ValueText lost: got %q want %q", out.ValueText, in.ValueText)
    }
}

func TestObservationToAUObservation_MobilityRoundTrip(t *testing.T) {
    val := 4.0
    in := models.Observation{
        ID: uuid.New(), ResidentID: uuid.New(),
        Kind: models.ObservationKindMobility,
        Value: &val, Unit: "score",
        ObservedAt: time.Now().UTC(),
    }
    fhir, err := ObservationToAUObservation(in)
    if err != nil {
        t.Fatalf("egress: %v", err)
    }
    b, _ := json.Marshal(fhir)
    var rt map[string]interface{}
    _ = json.Unmarshal(b, &rt)
    out, _ := AUObservationToObservation(rt)
    if out.Kind != models.ObservationKindMobility {
        t.Errorf("Kind lost: got %q", out.Kind)
    }
}

func TestObservationToAUObservation_WeightRoundTrip(t *testing.T) {
    val := 78.4
    in := models.Observation{
        ID: uuid.New(), ResidentID: uuid.New(),
        Kind: models.ObservationKindWeight,
        Value: &val, Unit: "kg",
        ObservedAt: time.Now().UTC(),
    }
    fhir, err := ObservationToAUObservation(in)
    if err != nil {
        t.Fatalf("egress: %v", err)
    }
    b, _ := json.Marshal(fhir)
    var rt map[string]interface{}
    _ = json.Unmarshal(b, &rt)
    out, _ := AUObservationToObservation(rt)
    if out.Value == nil || *out.Value != val {
        t.Errorf("weight Value lost: got %v want %v", out.Value, val)
    }
}

func TestObservationToAUObservation_RejectsInvalid(t *testing.T) {
    bad := models.Observation{ID: uuid.New(), ResidentID: uuid.New(), Kind: "behavioral" /* US spelling */, ObservedAt: time.Now()}
    if _, err := ObservationToAUObservation(bad); err == nil {
        t.Errorf("expected egress validation error for invalid kind; got nil")
    }
}

func TestAUObservationToObservation_WrongResourceType(t *testing.T) {
    payload := map[string]interface{}{"resourceType": "Patient", "id": uuid.NewString()}
    if _, err := AUObservationToObservation(payload); err == nil {
        t.Errorf("expected error for resourceType=Patient; got nil")
    }
}

func TestObservationToAUObservation_WireFormatHasKindExtension(t *testing.T) {
    val := 7.0
    in := models.Observation{
        ID: uuid.MustParse("11111111-2222-3333-4444-555555555555"),
        ResidentID: uuid.MustParse("99999999-8888-7777-6666-555555555555"),
        Kind: models.ObservationKindLab, LOINCCode: "4548-4",
        Value: &val, Unit: "%", ObservedAt: time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC),
    }
    fhir, err := ObservationToAUObservation(in)
    if err != nil {
        t.Fatalf("egress: %v", err)
    }
    b, _ := json.Marshal(fhir)
    s := string(b)
    if !strings.Contains(s, ExtObservationKind) {
        t.Errorf("wire format missing ExtObservationKind URI; got: %s", s)
    }
    if !strings.Contains(s, `"resourceType":"Observation"`) {
        t.Errorf("wire format missing resourceType; got: %s", s)
    }
    if !strings.Contains(s, `"loinc_code"`) && !strings.Contains(s, `"4548-4"`) {
        t.Errorf("wire format missing LOINC code; got: %s", s)
    }
}
```

- [ ] **Step 2: Run test — expect FAIL**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && \
  go test ./v2_substrate/fhir/... -run "TestObservationToAUObservation|TestAUObservationToObservation" -v
```

Expected: FAIL with `undefined: ObservationToAUObservation`.

- [ ] **Step 3: Write the mapper**

Path: `<kbs>/shared/v2_substrate/fhir/observation_mapper.go`

```go
package fhir

import (
    "encoding/json"
    "fmt"
    "strings"
    "time"

    "github.com/google/uuid"

    "github.com/cardiofit/shared/v2_substrate/models"
    "github.com/cardiofit/shared/v2_substrate/validation"
)

// ObservationToAUObservation translates a v2 Observation to an AU FHIR
// Observation (HL7 AU Base v6.0.0) as a generic map[string]interface{}
// suitable for json.Marshal to wire format.
//
// AU-specific extensions used:
//   - ExtObservationKind     (Vaidshala-internal; FHIR has no native discriminator)
//   - ExtObservationDelta    (Vaidshala-internal; encodes Delta as JSON)
//   - ExtObservationSourceID (Vaidshala-internal; UUID reference to kb-22.clinical_sources)
//
// Lossy fields (NOT round-tripped):
//   - CreatedAt — managed by canonical store, not part of FHIR shape
//
// Egress validates the input via validation.ValidateObservation before
// constructing the FHIR map. Ingress validates the output.
func ObservationToAUObservation(o models.Observation) (map[string]interface{}, error) {
    if err := validation.ValidateObservation(o); err != nil {
        return nil, fmt.Errorf("egress validation: %w", err)
    }

    out := map[string]interface{}{
        "resourceType": "Observation",
        "id":           o.ID.String(),
        "status":       "final",
        "subject": map[string]interface{}{
            "reference": "Patient/" + o.ResidentID.String(),
        },
        "effectiveDateTime": o.ObservedAt.UTC().Format(time.RFC3339),
    }

    // code.coding from LOINC + SNOMED (whichever present)
    coding := []map[string]interface{}{}
    if o.LOINCCode != "" {
        coding = append(coding, map[string]interface{}{
            "system": "http://loinc.org",
            "code":   o.LOINCCode,
        })
    }
    if o.SNOMEDCode != "" {
        coding = append(coding, map[string]interface{}{
            "system": "http://snomed.info/sct",
            "code":   o.SNOMEDCode,
        })
    }
    out["code"] = map[string]interface{}{"coding": coding}

    // valueQuantity OR valueString
    if o.Value != nil {
        vq := map[string]interface{}{"value": *o.Value}
        if o.Unit != "" {
            vq["unit"] = o.Unit
        }
        out["valueQuantity"] = vq
    } else if o.ValueText != "" {
        out["valueString"] = o.ValueText
    }

    // Vaidshala extensions
    extensions := []map[string]interface{}{
        {"url": ExtObservationKind, "valueCode": o.Kind},
    }
    if o.SourceID != nil {
        extensions = append(extensions, map[string]interface{}{
            "url": ExtObservationSourceID, "valueString": o.SourceID.String(),
        })
    }
    if o.Delta != nil {
        deltaJSON, err := json.Marshal(o.Delta)
        if err != nil {
            return nil, fmt.Errorf("marshal delta: %w", err)
        }
        extensions = append(extensions, map[string]interface{}{
            "url": ExtObservationDelta, "valueString": string(deltaJSON),
        })
    }
    out["extension"] = extensions

    return out, nil
}

// AUObservationToObservation translates an AU FHIR Observation JSON map back
// to a v2 Observation. Returns an error if resourceType != "Observation"
// or if the resulting Observation fails ValidateObservation.
func AUObservationToObservation(in map[string]interface{}) (*models.Observation, error) {
    if rt, _ := in["resourceType"].(string); rt != "Observation" {
        return nil, fmt.Errorf("resourceType: got %q want Observation", rt)
    }

    var o models.Observation

    if idStr, _ := in["id"].(string); idStr != "" {
        if id, err := uuid.Parse(idStr); err == nil {
            o.ID = id
        }
    }

    // subject.reference -> ResidentID
    if subj, ok := in["subject"].(map[string]interface{}); ok {
        if ref, _ := subj["reference"].(string); strings.HasPrefix(ref, "Patient/") {
            if rid, err := uuid.Parse(strings.TrimPrefix(ref, "Patient/")); err == nil {
                o.ResidentID = rid
            }
        }
    }

    // effectiveDateTime
    if eff, _ := in["effectiveDateTime"].(string); eff != "" {
        if t, err := time.Parse(time.RFC3339, eff); err == nil {
            o.ObservedAt = t
        }
    }

    // code.coding -> LOINC / SNOMED
    if code, ok := in["code"].(map[string]interface{}); ok {
        if codings, ok := code["coding"].([]interface{}); ok {
            for _, c := range codings {
                cm, _ := c.(map[string]interface{})
                sys, _ := cm["system"].(string)
                val, _ := cm["code"].(string)
                switch sys {
                case "http://loinc.org":
                    o.LOINCCode = val
                case "http://snomed.info/sct":
                    o.SNOMEDCode = val
                }
            }
        }
    }

    // valueQuantity OR valueString
    if vq, ok := in["valueQuantity"].(map[string]interface{}); ok {
        if v, ok := vq["value"].(float64); ok {
            o.Value = &v
        }
        if u, _ := vq["unit"].(string); u != "" {
            o.Unit = u
        }
    } else if vs, ok := in["valueString"].(string); ok {
        o.ValueText = vs
    }

    // extensions
    if exts, ok := in["extension"].([]interface{}); ok {
        for _, e := range exts {
            em, _ := e.(map[string]interface{})
            url, _ := em["url"].(string)
            switch url {
            case ExtObservationKind:
                if k, _ := em["valueCode"].(string); k != "" {
                    o.Kind = k
                }
            case ExtObservationSourceID:
                if s, _ := em["valueString"].(string); s != "" {
                    if sid, err := uuid.Parse(s); err == nil {
                        o.SourceID = &sid
                    }
                }
            case ExtObservationDelta:
                if s, _ := em["valueString"].(string); s != "" {
                    var d models.Delta
                    if err := json.Unmarshal([]byte(s), &d); err == nil {
                        o.Delta = &d
                    }
                }
            }
        }
    }

    if err := validation.ValidateObservation(o); err != nil {
        return nil, fmt.Errorf("ingress validation: %w", err)
    }
    return &o, nil
}
```

- [ ] **Step 4: Run test — expect PASS**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && \
  go test ./v2_substrate/fhir/... -run "TestObservation|TestAUObservation" -v
```

Expected: PASS for all 8 Observation FHIR tests. Existing β.1/β.2-A FHIR tests still green.

- [ ] **Step 5: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/fhir/observation_mapper.go \
        backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/fhir/observation_mapper_test.go
git commit -m "feat(v2_substrate/fhir): Observation ↔ AU FHIR Observation mapper with egress+ingress validation"
```

---

## Task 6: ObservationStore interface

**Files:**
- Modify: `<kbs>/shared/v2_substrate/interfaces/storage.go`

- [ ] **Step 1: Append ObservationStore interface to existing storage.go**

Path: `<kbs>/shared/v2_substrate/interfaces/storage.go` (append at end of file, after the existing MedicineUseStore block)

```go
// ObservationStore is the canonical storage contract for Observation entities.
// kb-20-patient-profile is the only KB expected to implement this. List
// methods take limit/offset; the implementation may apply a maximum cap
// (e.g. 1000) but caller should not rely on that.
//
// Implementations of UpsertObservation MUST compute Delta before insert via
// shared/v2_substrate/delta.ComputeDelta with an injected BaselineProvider;
// when the provider returns delta.ErrNoBaseline (or Value is nil or
// Kind=behavioural), the resulting Delta.DirectionalFlag must be
// DeltaFlagNoBaseline.
type ObservationStore interface {
    GetObservation(ctx context.Context, id uuid.UUID) (*models.Observation, error)
    UpsertObservation(ctx context.Context, o models.Observation) (*models.Observation, error)
    ListObservationsByResident(ctx context.Context, residentID uuid.UUID, limit, offset int) ([]models.Observation, error)
    ListObservationsByResidentAndKind(ctx context.Context, residentID uuid.UUID, kind string, limit, offset int) ([]models.Observation, error)
}
```

Note: `interfaces/` has no test file by design (interface declarations are not unit-tested in this codebase). Verification is via `go vet` and successful compilation by Task 10's storage methods.

- [ ] **Step 2: Verify compile**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && \
  go vet ./v2_substrate/interfaces/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/interfaces/storage.go
git commit -m "feat(v2_substrate): add ObservationStore interface alongside MedicineUseStore"
```

---

## Task 7: Migration 008_part2_partB — observations table + observations_v2 view

**Files:**
- Create: `<kbs>/kb-20-patient-profile/migrations/008_part2_clinical_primitives_partB.sql`

- [ ] **Step 1: Confirm existing lab_entries DDL (so the UNION columns match reality)**

```bash
sed -n '44,67p' /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/migrations/001_initial_schema.sql
```

Expected: shows lab_entries has columns `id UUID, patient_id VARCHAR(100), lab_type VARCHAR(30), value DECIMAL(10,4), unit VARCHAR(20), measured_at TIMESTAMPTZ, source VARCHAR(50), is_derived BOOLEAN, validation_status, flag_reason, created_at`. Note: lab_entries has NO resident_id, NO source_id UUID — the view must backfill resident_id from patient_profiles via patient_id, and lab_entries.source_id projects as NULL.

- [ ] **Step 2: Write the migration file**

Path: `<kbs>/kb-20-patient-profile/migrations/008_part2_clinical_primitives_partB.sql`

```sql
-- ============================================================================
-- Migration 008 part 2 PART B: Observation v2 Substrate (Phase 1B-β.2-B)
--
-- Implements the v2 substrate Observation entity:
--   - observations table (greenfield) — kind discriminator over
--     vital | lab | behavioural | mobility | weight; pointer-nullable value
--     paired with optional value_text via CHECK; delta JSONB populated by
--     the application-layer delta-on-write service (NOT a DB trigger)
--   - observations_v2 view (UNION of observations + lab_entries projected
--     with kind='lab') — provides a single read shape for v2 substrate
--     consumers while leaving lab_entries unchanged for legacy consumers
--
-- Source provenance: observations.source_id is a UUID reference to
-- kb-22.clinical_sources. NO foreign key (cross-DB). Application validates
-- existence at write time when source_id is non-NULL.
--
-- All existing kb-20 consumers continue reading raw lab_entries unchanged.
--
-- Spec: docs/superpowers/specs/2026-05-04-1b-beta-2-clinical-primitives-design.md (§2.2, §2.4, §5.2, §5.4)
-- Plan: docs/superpowers/plans/2026-05-04-1b-beta-2-clinical-primitives-plan.md
-- Date: 2026-05-04
-- ============================================================================

BEGIN;

-- pgcrypto already enabled by 001; defensive re-declare for self-contained safety
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ============================================================================
-- Section 1 — observations table (greenfield)
-- ============================================================================
CREATE TABLE IF NOT EXISTS observations (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resident_id  UUID NOT NULL,
    loinc_code   TEXT,
    snomed_code  TEXT,
    kind         TEXT NOT NULL CHECK (kind IN ('vital','lab','behavioural','mobility','weight')),
    value        DECIMAL(12,4),
    value_text   TEXT,
    unit         TEXT,
    observed_at  TIMESTAMPTZ NOT NULL,
    source_id    UUID,
    delta        JSONB,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT observations_value_or_text CHECK (value IS NOT NULL OR value_text IS NOT NULL)
);

-- Access pattern indexes
CREATE INDEX IF NOT EXISTS idx_observations_resident    ON observations(resident_id);
CREATE INDEX IF NOT EXISTS idx_observations_observed_at ON observations(resident_id, observed_at DESC);
CREATE INDEX IF NOT EXISTS idx_observations_kind        ON observations(resident_id, kind, observed_at DESC);

-- ============================================================================
-- Section 2 — Per-column COMMENTs documenting the v2 contract
-- ============================================================================
COMMENT ON COLUMN observations.resident_id IS
    'v2 Resident.id reference (kb-20 patient_profiles via residents_v2). FK enforced at write time, not as DB FK to keep migration non-breaking. Added 2026-05-04 in migration 008_part2_partB.';
COMMENT ON COLUMN observations.kind IS
    'v2 Observation discriminator: vital | lab | behavioural | mobility | weight. AU spelling "behavioural" is canonical. Added 2026-05-04 in migration 008_part2_partB.';
COMMENT ON COLUMN observations.value IS
    'v2 Observation numeric value (NULL when value_text carries the data — e.g. behavioural narratives). One of value or value_text MUST be present (CHECK observations_value_or_text). Added 2026-05-04 in migration 008_part2_partB.';
COMMENT ON COLUMN observations.value_text IS
    'v2 Observation text value (e.g. behavioural episode narrative). Complement to value; one MUST be present. Added 2026-05-04 in migration 008_part2_partB.';
COMMENT ON COLUMN observations.source_id IS
    'v2 source provenance — UUID reference to kb-22.clinical_sources. NO FK (cross-DB). Application validates existence at write time when non-NULL. Added 2026-05-04 in migration 008_part2_partB.';
COMMENT ON COLUMN observations.delta IS
    'v2 Observation Delta JSONB shape: {"baseline_value":<float>, "deviation_stddev":<float>, "flag":"within_baseline|elevated|severely_elevated|low|severely_low|no_baseline", "computed_at":"<RFC3339>"}. Populated at write time by the application-layer delta-on-write service (shared/v2_substrate/delta/compute.go). Added 2026-05-04 in migration 008_part2_partB.';

-- ============================================================================
-- Section 3 — observations_v2 view
-- ============================================================================
-- UNION of:
--   1. observations (v2 native rows of any kind)
--   2. lab_entries (legacy lab rows projected with kind='lab')
--
-- Legacy lab_entries projection notes:
--   - resident_id is backfilled via patient_profiles lookup (lab_entries.patient_id is VARCHAR)
--   - source_id is NULL (legacy lab_entries has no UUID provenance link)
--   - delta is NULL (legacy lab_entries has no Delta)
--   - snomed_code is NULL (legacy lab_entries has no SNOMED column)
--   - loinc_code falls back to lab_type (legacy free-text values like 'EGFR'/'HBA1C')
--
-- Existing consumers (medication-service) keep reading raw lab_entries unchanged.
-- ============================================================================
CREATE OR REPLACE VIEW observations_v2 AS
SELECT
    o.id,
    o.resident_id,
    o.loinc_code,
    o.snomed_code,
    o.kind,
    o.value,
    o.value_text,
    o.unit,
    o.observed_at,
    o.source_id,
    o.delta,
    o.created_at
FROM observations o
UNION ALL
SELECT
    le.id                                                                      AS id,
    (SELECT pp.id FROM patient_profiles pp WHERE pp.patient_id = le.patient_id LIMIT 1) AS resident_id,
    le.lab_type                                                                AS loinc_code,
    NULL::TEXT                                                                 AS snomed_code,
    'lab'::TEXT                                                                AS kind,
    le.value                                                                   AS value,
    NULL::TEXT                                                                 AS value_text,
    le.unit                                                                    AS unit,
    le.measured_at                                                             AS observed_at,
    NULL::UUID                                                                 AS source_id,
    NULL::JSONB                                                                AS delta,
    le.created_at                                                              AS created_at
FROM lab_entries le;

COMMENT ON VIEW observations_v2 IS
    'Compatibility read shape for v2 substrate Observation consumers. UNIONs the greenfield observations table (any kind) with legacy lab_entries projected as kind=''lab''. Legacy lab rows surface with source_id=NULL and delta=NULL because lab_entries does not carry those columns. Added 2026-05-04 in migration 008_part2_partB.';

COMMIT;

-- ============================================================================
-- Acceptance check (run after applying):
--   SELECT column_name FROM information_schema.columns
--     WHERE table_name='observations'
--     ORDER BY ordinal_position;
--   -- expect 12 rows: id, resident_id, loinc_code, snomed_code, kind, value,
--   --                 value_text, unit, observed_at, source_id, delta, created_at
--   SELECT * FROM observations_v2 LIMIT 1;
--   -- view executes (may be 0 rows on fresh DB)
-- ============================================================================
```

- [ ] **Step 3: Sanity-count the migration shape**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/migrations && \
  printf "BEGIN: %s\n" "$(grep -c '^BEGIN;' 008_part2_clinical_primitives_partB.sql)" && \
  printf "COMMIT: %s\n" "$(grep -c '^COMMIT;' 008_part2_clinical_primitives_partB.sql)" && \
  printf "CREATE TABLE: %s\n" "$(grep -c '^CREATE TABLE' 008_part2_clinical_primitives_partB.sql)" && \
  printf "CREATE INDEX: %s\n" "$(grep -c 'CREATE INDEX' 008_part2_clinical_primitives_partB.sql)" && \
  printf "CREATE OR REPLACE VIEW: %s\n" "$(grep -c 'CREATE OR REPLACE VIEW' 008_part2_clinical_primitives_partB.sql)" && \
  printf "COMMENT ON COLUMN: %s\n" "$(grep -c 'COMMENT ON COLUMN' 008_part2_clinical_primitives_partB.sql)" && \
  printf "uuid_generate_v4 leftovers: %s\n" "$(grep -c 'uuid_generate_v4' 008_part2_clinical_primitives_partB.sql)"
```

Expected:
- BEGIN: 1
- COMMIT: 1
- CREATE TABLE: 1
- CREATE INDEX: 3
- CREATE OR REPLACE VIEW: 1
- COMMENT ON COLUMN: 6
- uuid_generate_v4 leftovers: 0

- [ ] **Step 4: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/migrations/008_part2_clinical_primitives_partB.sql
git commit -m "feat(kb-20): migration 008_part2_partB observations table + observations_v2 view (UNION lab_entries)"
```

---

## Task 8: Delta package interfaces (BaselineProvider + Baseline + ErrNoBaseline)

**Files:**
- Create: `<kbs>/shared/v2_substrate/delta/interfaces.go`

- [ ] **Step 1: Write the interfaces file**

Path: `<kbs>/shared/v2_substrate/delta/interfaces.go`

```go
// Package delta provides the delta-on-write service for v2 substrate
// Observations. ComputeDelta is a pure function: given an Observation and an
// optional Baseline, it returns a Delta describing the directional deviation
// from baseline. Baselines are sourced via the BaselineProvider interface,
// which kb-26 (or any KB owning baseline data) implements as a thin adapter.
//
// Why service-layer (not DB trigger): triggers cannot cleanly call kb-26's
// AcuteRepository (separate service, separate DB); triggers create hidden
// coupling; service-layer is testable with mock BaselineProviders. See
// spec §2.1 for the architectural rationale.
package delta

import (
    "context"
    "errors"
    "time"

    "github.com/google/uuid"
)

// ErrNoBaseline is returned by BaselineProvider.FetchBaseline when no
// historical data exists for (residentID, vitalType). ComputeDelta callers
// MUST translate this sentinel into a Delta with DirectionalFlag =
// models.DeltaFlagNoBaseline rather than failing the write.
var ErrNoBaseline = errors.New("delta: no baseline available")

// Baseline is the historical reference point for a single vital type at a
// single resident. Sourced from kb-26 (or a replica). SampleSize is the
// number of historical observations the BaselineValue + StdDev were derived
// from; ComputedAt is when kb-26 last refreshed the baseline.
//
// StdDev is the population standard deviation in the same unit as the
// associated Observation.Value. ComputeDelta divides (value - BaselineValue)
// by StdDev to derive the deviation in standard-deviation units; thresholds
// for the directional flag are defined in compute.go.
type Baseline struct {
    BaselineValue float64   `json:"baseline_value"`
    StdDev        float64   `json:"stddev"`
    SampleSize    int       `json:"sample_size"`
    ComputedAt    time.Time `json:"computed_at"`
}

// BaselineProvider exposes baseline data for delta computation. kb-26's
// AcuteRepository is the production implementation; tests use in-memory
// mocks. vitalType is the LOINC code (vitals/labs) or model-internal kind
// identifier (e.g. "weight"); the provider resolves it to its own internal
// vital-type key.
//
// FetchBaseline returns ErrNoBaseline when no data exists. Other errors
// (network, decode) propagate to the caller; UpsertObservation translates
// them into Delta with DirectionalFlag = no_baseline + logs the error
// (decision: do not fail the write because the observation must persist
// regardless of baseline availability).
type BaselineProvider interface {
    FetchBaseline(ctx context.Context, residentID uuid.UUID, vitalType string) (*Baseline, error)
}
```

- [ ] **Step 2: Verify compile**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && \
  go vet ./v2_substrate/delta/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/delta/interfaces.go
git commit -m "feat(v2_substrate/delta): BaselineProvider interface + Baseline type + ErrNoBaseline sentinel"
```

---

## Task 9: ComputeDelta pure function + table-driven tests

**Files:**
- Create: `<kbs>/shared/v2_substrate/delta/compute.go`
- Create: `<kbs>/shared/v2_substrate/delta/compute_test.go`

- [ ] **Step 1: Write the failing table-driven test**

Path: `<kbs>/shared/v2_substrate/delta/compute_test.go`

```go
package delta

import (
    "testing"
    "time"

    "github.com/google/uuid"

    "github.com/cardiofit/shared/v2_substrate/models"
)

func ptr(f float64) *float64 { return &f }

func TestComputeDelta_Cases(t *testing.T) {
    bl := &Baseline{
        BaselineValue: 120.0,
        StdDev:        10.0,
        SampleSize:    50,
        ComputedAt:    time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
    }

    cases := []struct {
        name      string
        obs       models.Observation
        baseline  *Baseline
        wantFlag  string
    }{
        {
            name: "within_baseline_value_equals_baseline",
            obs:  models.Observation{ResidentID: uuid.New(), Kind: models.ObservationKindVital, Value: ptr(120.0), ObservedAt: time.Now()},
            baseline: bl,
            wantFlag: models.DeltaFlagWithinBaseline,
        },
        {
            name: "within_baseline_one_stddev_high",
            obs:  models.Observation{ResidentID: uuid.New(), Kind: models.ObservationKindVital, Value: ptr(130.0), ObservedAt: time.Now()},
            baseline: bl,
            wantFlag: models.DeltaFlagWithinBaseline, // |dev|=1.0 → within (boundary inclusive)
        },
        {
            name: "elevated_1pt5_stddev",
            obs:  models.Observation{ResidentID: uuid.New(), Kind: models.ObservationKindVital, Value: ptr(135.0), ObservedAt: time.Now()},
            baseline: bl,
            wantFlag: models.DeltaFlagElevated,
        },
        {
            name: "severely_elevated_3_stddev",
            obs:  models.Observation{ResidentID: uuid.New(), Kind: models.ObservationKindVital, Value: ptr(150.0), ObservedAt: time.Now()},
            baseline: bl,
            wantFlag: models.DeltaFlagSeverelyElevated,
        },
        {
            name: "low_1pt5_stddev_below",
            obs:  models.Observation{ResidentID: uuid.New(), Kind: models.ObservationKindVital, Value: ptr(105.0), ObservedAt: time.Now()},
            baseline: bl,
            wantFlag: models.DeltaFlagLow,
        },
        {
            name: "severely_low_3_stddev_below",
            obs:  models.Observation{ResidentID: uuid.New(), Kind: models.ObservationKindVital, Value: ptr(90.0), ObservedAt: time.Now()},
            baseline: bl,
            wantFlag: models.DeltaFlagSeverelyLow,
        },
        {
            name: "no_baseline_when_baseline_nil",
            obs:  models.Observation{ResidentID: uuid.New(), Kind: models.ObservationKindVital, Value: ptr(120.0), ObservedAt: time.Now()},
            baseline: nil,
            wantFlag: models.DeltaFlagNoBaseline,
        },
        {
            name: "no_baseline_when_value_nil",
            obs:  models.Observation{ResidentID: uuid.New(), Kind: models.ObservationKindBehavioural, ValueText: "agitation", ObservedAt: time.Now()},
            baseline: bl,
            wantFlag: models.DeltaFlagNoBaseline,
        },
        {
            name: "no_baseline_when_kind_behavioural",
            obs:  models.Observation{ResidentID: uuid.New(), Kind: models.ObservationKindBehavioural, Value: ptr(1.0), ObservedAt: time.Now()},
            baseline: bl,
            wantFlag: models.DeltaFlagNoBaseline,
        },
        {
            name: "no_baseline_when_stddev_zero_guards_div0",
            obs:  models.Observation{ResidentID: uuid.New(), Kind: models.ObservationKindVital, Value: ptr(150.0), ObservedAt: time.Now()},
            baseline: &Baseline{BaselineValue: 120.0, StdDev: 0.0, SampleSize: 1, ComputedAt: time.Now()},
            wantFlag: models.DeltaFlagNoBaseline,
        },
    }

    for _, c := range cases {
        t.Run(c.name, func(t *testing.T) {
            d := ComputeDelta(c.obs, c.baseline)
            if d.DirectionalFlag != c.wantFlag {
                t.Errorf("ComputeDelta flag: got %q want %q (case %s)", d.DirectionalFlag, c.wantFlag, c.name)
            }
            if c.wantFlag != models.DeltaFlagNoBaseline {
                if d.BaselineValue != c.baseline.BaselineValue {
                    t.Errorf("BaselineValue: got %v want %v", d.BaselineValue, c.baseline.BaselineValue)
                }
            } else {
                if d.BaselineValue != 0 || d.DeviationStdDev != 0 {
                    t.Errorf("no_baseline must zero numeric fields, got BL=%v dev=%v", d.BaselineValue, d.DeviationStdDev)
                }
            }
            if d.ComputedAt.IsZero() {
                t.Errorf("ComputedAt must be set, got zero")
            }
        })
    }
}
```

- [ ] **Step 2: Run test — expect FAIL**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && \
  go test ./v2_substrate/delta/... -v
```

Expected: FAIL with `undefined: ComputeDelta`.

- [ ] **Step 3: Write the pure function**

Path: `<kbs>/shared/v2_substrate/delta/compute.go`

```go
package delta

import (
    "math"
    "time"

    "github.com/cardiofit/shared/v2_substrate/models"
)

// Threshold constants (in standard-deviation units).
//
// |dev| <= 1.0           → within_baseline
// 1.0 < dev <= 2.0       → elevated      (high side)
// dev > 2.0              → severely_elevated
// -2.0 <= dev < -1.0     → low           (low side)
// dev < -2.0             → severely_low
//
// Boundary semantics: thresholds are inclusive on the within_baseline side
// (|dev|=1.0 stays within_baseline; |dev|=2.0 stays elevated/low). This
// matches spec §3.7 description "elevated when 1<dev<=2".
const (
    thresholdWithin   = 1.0
    thresholdSevere   = 2.0
)

// ComputeDelta returns the directional Delta for obs given baseline.
// Pure function: no IO, no time.Now beyond stamping ComputedAt.
//
// Returns DeltaFlagNoBaseline (with zeroed numeric fields) when:
//   - baseline is nil
//   - obs.Value is nil (e.g. behavioural ValueText-only observation)
//   - obs.Kind == ObservationKindBehavioural (no numeric semantics — see spec §8 risk row)
//   - baseline.StdDev == 0 (would yield Inf/NaN; treat as insufficient data)
func ComputeDelta(obs models.Observation, baseline *Baseline) models.Delta {
    now := time.Now().UTC()

    if baseline == nil ||
        obs.Value == nil ||
        obs.Kind == models.ObservationKindBehavioural ||
        baseline.StdDev == 0 {
        return models.Delta{
            BaselineValue:   0,
            DeviationStdDev: 0,
            DirectionalFlag: models.DeltaFlagNoBaseline,
            ComputedAt:      now,
        }
    }

    deviation := (*obs.Value - baseline.BaselineValue) / baseline.StdDev
    abs := math.Abs(deviation)

    var flag string
    switch {
    case abs <= thresholdWithin:
        flag = models.DeltaFlagWithinBaseline
    case deviation > thresholdSevere:
        flag = models.DeltaFlagSeverelyElevated
    case deviation > thresholdWithin: // 1 < dev <= 2
        flag = models.DeltaFlagElevated
    case deviation < -thresholdSevere:
        flag = models.DeltaFlagSeverelyLow
    default: // -2 <= dev < -1
        flag = models.DeltaFlagLow
    }

    return models.Delta{
        BaselineValue:   baseline.BaselineValue,
        DeviationStdDev: deviation,
        DirectionalFlag: flag,
        ComputedAt:      now,
    }
}
```

- [ ] **Step 4: Run test — expect PASS**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && \
  go test ./v2_substrate/delta/... -v
```

Expected: PASS for all 10 sub-cases of TestComputeDelta_Cases.

- [ ] **Step 5: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/delta/compute.go \
        backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/delta/compute_test.go
git commit -m "feat(v2_substrate/delta): pure ComputeDelta with directional flag thresholds + 10-case test"
```

---

## Task 10: kb26 in-memory BaselineProvider stub

**Files:**
- Create: `<kbs>/kb-20-patient-profile/internal/storage/kb26_baseline_provider.go`
- Create: `<kbs>/kb-20-patient-profile/internal/storage/kb26_baseline_provider_test.go`

- [ ] **Step 1: Write the failing test**

Path: `<kbs>/kb-20-patient-profile/internal/storage/kb26_baseline_provider_test.go`

```go
package storage

import (
    "context"
    "errors"
    "testing"
    "time"

    "github.com/google/uuid"

    "github.com/cardiofit/shared/v2_substrate/delta"
)

func TestInMemoryBaselineProvider_FetchExistingBaseline(t *testing.T) {
    p := NewInMemoryBaselineProvider()
    rid := uuid.New()
    p.Seed(rid, "8480-6", delta.Baseline{
        BaselineValue: 130.0, StdDev: 8.0, SampleSize: 30, ComputedAt: time.Now(),
    })
    bl, err := p.FetchBaseline(context.Background(), rid, "8480-6")
    if err != nil {
        t.Fatalf("FetchBaseline: %v", err)
    }
    if bl.BaselineValue != 130.0 || bl.StdDev != 8.0 {
        t.Errorf("baseline drift: got %+v", bl)
    }
}

func TestInMemoryBaselineProvider_MissingReturnsErrNoBaseline(t *testing.T) {
    p := NewInMemoryBaselineProvider()
    _, err := p.FetchBaseline(context.Background(), uuid.New(), "8480-6")
    if !errors.Is(err, delta.ErrNoBaseline) {
        t.Errorf("expected ErrNoBaseline, got %v", err)
    }
}
```

- [ ] **Step 2: Run test — expect FAIL**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile && \
  go test ./internal/storage/... -run "TestInMemoryBaselineProvider" -v
```

Expected: FAIL with `undefined: NewInMemoryBaselineProvider`.

- [ ] **Step 3: Write the implementation**

Path: `<kbs>/kb-20-patient-profile/internal/storage/kb26_baseline_provider.go`

```go
package storage

import (
    "context"
    "sync"

    "github.com/google/uuid"

    "github.com/cardiofit/shared/v2_substrate/delta"
)

// InMemoryBaselineProvider is the MVP delta.BaselineProvider implementation.
// It serves baselines from an in-memory map keyed by (residentID, vitalType)
// and is suitable for unit tests + early-pilot deployments where kb-26 may
// not yet expose its AcuteRepository over the network.
//
// Production wiring (deferred — non-blocking for β.2-C exit): replace with
// a thin adapter over kb-26's AcuteRepository.FetchBaseline(patientID,
// vitalType) using the kb-26 internal HTTP/gRPC API. The interface contract
// (delta.BaselineProvider) does not change; this seam is exactly where the
// migration lands.
type InMemoryBaselineProvider struct {
    mu        sync.RWMutex
    baselines map[string]delta.Baseline
}

// NewInMemoryBaselineProvider returns an empty provider; seed with Seed().
func NewInMemoryBaselineProvider() *InMemoryBaselineProvider {
    return &InMemoryBaselineProvider{baselines: map[string]delta.Baseline{}}
}

func keyFor(residentID uuid.UUID, vitalType string) string {
    return residentID.String() + "::" + vitalType
}

// Seed inserts or replaces a baseline for (residentID, vitalType). Test-only
// helper; production code populates via the (deferred) kb-26 adapter.
func (p *InMemoryBaselineProvider) Seed(residentID uuid.UUID, vitalType string, b delta.Baseline) {
    p.mu.Lock()
    defer p.mu.Unlock()
    p.baselines[keyFor(residentID, vitalType)] = b
}

// FetchBaseline implements delta.BaselineProvider. Returns delta.ErrNoBaseline
// when no entry exists for (residentID, vitalType).
func (p *InMemoryBaselineProvider) FetchBaseline(ctx context.Context, residentID uuid.UUID, vitalType string) (*delta.Baseline, error) {
    p.mu.RLock()
    defer p.mu.RUnlock()
    if b, ok := p.baselines[keyFor(residentID, vitalType)]; ok {
        return &b, nil
    }
    return nil, delta.ErrNoBaseline
}
```

- [ ] **Step 4: Run test — expect PASS**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile && \
  go test ./internal/storage/... -run "TestInMemoryBaselineProvider" -v
```

Expected: PASS for both tests.

- [ ] **Step 5: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/storage/kb26_baseline_provider.go \
        backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/storage/kb26_baseline_provider_test.go
git commit -m "feat(kb-20): InMemoryBaselineProvider MVP for delta-on-write (kb-26 adapter deferred)"
```

---

## Task 11: V2SubstrateStore Observation methods (scan + Get + Upsert + List + ListByKind)

**Files:**
- Modify: `<kbs>/kb-20-patient-profile/internal/storage/v2_substrate_store.go`
- Create: `<kbs>/kb-20-patient-profile/internal/storage/v2_substrate_observation_test.go`

- [ ] **Step 1: Write the failing DB-gated test**

Path: `<kbs>/kb-20-patient-profile/internal/storage/v2_substrate_observation_test.go`

```go
package storage

import (
    "context"
    "errors"
    "os"
    "testing"
    "time"

    "github.com/google/uuid"

    "github.com/cardiofit/shared/v2_substrate/delta"
    "github.com/cardiofit/shared/v2_substrate/interfaces"
    "github.com/cardiofit/shared/v2_substrate/models"
)

func openTestStore(t *testing.T) (*V2SubstrateStore, *InMemoryBaselineProvider) {
    t.Helper()
    dsn := os.Getenv("KB20_TEST_DATABASE_URL")
    if dsn == "" {
        t.Skip("KB20_TEST_DATABASE_URL not set; skipping DB-gated observation storage test")
    }
    store, err := NewV2SubstrateStore(dsn)
    if err != nil {
        t.Fatalf("open store: %v", err)
    }
    bp := NewInMemoryBaselineProvider()
    store.SetBaselineProvider(bp)
    return store, bp
}

func ptr(f float64) *float64 { return &f }

func TestUpsertGetObservation_RoundTrip(t *testing.T) {
    store, _ := openTestStore(t)
    defer store.Close()

    in := models.Observation{
        ID:         uuid.New(),
        ResidentID: uuid.New(),
        LOINCCode:  "8480-6",
        Kind:       models.ObservationKindVital,
        Value:      ptr(132.0),
        Unit:       "mmHg",
        ObservedAt: time.Now().UTC().Truncate(time.Second),
    }
    out, err := store.UpsertObservation(context.Background(), in)
    if err != nil {
        t.Fatalf("upsert: %v", err)
    }
    if out.Delta == nil || out.Delta.DirectionalFlag != models.DeltaFlagNoBaseline {
        t.Errorf("expected Delta.Flag=no_baseline (no provider seed), got %+v", out.Delta)
    }

    got, err := store.GetObservation(context.Background(), in.ID)
    if err != nil {
        t.Fatalf("get: %v", err)
    }
    if got.LOINCCode != in.LOINCCode || got.Kind != in.Kind {
        t.Errorf("round-trip drift: got %+v want %+v", got, in)
    }
    if got.Value == nil || *got.Value != *in.Value {
        t.Errorf("Value drift: got %v want %v", got.Value, in.Value)
    }
}

func TestGetObservation_NotFoundSentinel(t *testing.T) {
    store, _ := openTestStore(t)
    defer store.Close()
    _, err := store.GetObservation(context.Background(), uuid.New())
    if !errors.Is(err, interfaces.ErrNotFound) {
        t.Errorf("expected ErrNotFound, got %v", err)
    }
}

func TestUpsertObservation_BehaviouralValueText(t *testing.T) {
    store, _ := openTestStore(t)
    defer store.Close()

    in := models.Observation{
        ID:         uuid.New(),
        ResidentID: uuid.New(),
        Kind:       models.ObservationKindBehavioural,
        ValueText:  "agitation episode 14:30, paced corridor",
        ObservedAt: time.Now().UTC().Truncate(time.Second),
    }
    out, err := store.UpsertObservation(context.Background(), in)
    if err != nil {
        t.Fatalf("upsert: %v", err)
    }
    if out.Value != nil {
        t.Errorf("Value should be nil for behavioural; got %v", *out.Value)
    }
    if out.ValueText != in.ValueText {
        t.Errorf("ValueText drift: got %q want %q", out.ValueText, in.ValueText)
    }
    if out.Delta == nil || out.Delta.DirectionalFlag != models.DeltaFlagNoBaseline {
        t.Errorf("behavioural must yield no_baseline Delta, got %+v", out.Delta)
    }
}

func TestListObservationsByResident(t *testing.T) {
    store, _ := openTestStore(t)
    defer store.Close()
    rid := uuid.New()
    for i := 0; i < 3; i++ {
        v := 120.0 + float64(i)
        _, err := store.UpsertObservation(context.Background(), models.Observation{
            ID: uuid.New(), ResidentID: rid, Kind: models.ObservationKindVital,
            LOINCCode: "8480-6", Value: &v, Unit: "mmHg",
            ObservedAt: time.Now().UTC().Add(-time.Duration(i) * time.Hour).Truncate(time.Second),
        })
        if err != nil {
            t.Fatalf("upsert %d: %v", i, err)
        }
    }
    got, err := store.ListObservationsByResident(context.Background(), rid, 100, 0)
    if err != nil {
        t.Fatalf("list: %v", err)
    }
    if len(got) != 3 {
        t.Errorf("expected 3 observations, got %d", len(got))
    }
}

func TestListObservationsByResidentAndKind(t *testing.T) {
    store, _ := openTestStore(t)
    defer store.Close()
    rid := uuid.New()
    v := 132.0
    _, _ = store.UpsertObservation(context.Background(), models.Observation{
        ID: uuid.New(), ResidentID: rid, Kind: models.ObservationKindVital,
        LOINCCode: "8480-6", Value: &v, Unit: "mmHg", ObservedAt: time.Now().UTC().Truncate(time.Second),
    })
    w := 78.0
    _, _ = store.UpsertObservation(context.Background(), models.Observation{
        ID: uuid.New(), ResidentID: rid, Kind: models.ObservationKindWeight,
        Value: &w, Unit: "kg", ObservedAt: time.Now().UTC().Truncate(time.Second),
    })
    got, err := store.ListObservationsByResidentAndKind(context.Background(), rid, models.ObservationKindWeight, 100, 0)
    if err != nil {
        t.Fatalf("list-by-kind: %v", err)
    }
    if len(got) != 1 || got[0].Kind != models.ObservationKindWeight {
        t.Errorf("expected exactly 1 weight observation, got %d (%+v)", len(got), got)
    }
}
```

- [ ] **Step 2: Run test — expect SKIP locally / FAIL when DB set**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile && \
  go test ./internal/storage/... -run "TestUpsertGetObservation|TestGetObservation_NotFound|TestUpsertObservation_Behavioural|TestListObservations" -v
```

Expected without DB env: SKIP cleanly. With DB env: FAIL with `undefined: V2SubstrateStore.SetBaselineProvider` etc.

- [ ] **Step 3: Append observation methods + SetBaselineProvider to V2SubstrateStore**

Path: `<kbs>/kb-20-patient-profile/internal/storage/v2_substrate_store.go` (append at end of file)

```go
// ============================================================================
// Observation
// ============================================================================

// SetBaselineProvider injects the delta.BaselineProvider used by
// UpsertObservation. Must be called before UpsertObservation; if unset,
// UpsertObservation falls back to Delta.Flag=no_baseline for every write.
func (s *V2SubstrateStore) SetBaselineProvider(bp delta.BaselineProvider) {
    s.baselineProvider = bp
}

// observationColumns matches the projection of observations_v2 (which UNIONs
// observations + lab_entries with kind='lab').
const observationColumns = `id, resident_id, loinc_code, snomed_code, kind,
       value, value_text, unit, observed_at, source_id, delta, created_at`

// scanObservation reads one row's columns (in observationColumns order) into
// a fully-populated Observation. Handles nullable LOINC/SNOMED, pointer-nullable
// Value, optional ValueText/Unit, optional SourceID, and JSONB Delta payload.
func scanObservation(sc rowScanner) (models.Observation, error) {
    var (
        o          models.Observation
        loinc      sql.NullString
        snomed     sql.NullString
        value      sql.NullFloat64
        valueText  sql.NullString
        unit       sql.NullString
        sourceID   uuid.NullUUID
        deltaBytes []byte
    )
    if err := sc.Scan(
        &o.ID, &o.ResidentID, &loinc, &snomed, &o.Kind,
        &value, &valueText, &unit, &o.ObservedAt,
        &sourceID, &deltaBytes, &o.CreatedAt,
    ); err != nil {
        return models.Observation{}, err
    }
    if loinc.Valid {
        o.LOINCCode = loinc.String
    }
    if snomed.Valid {
        o.SNOMEDCode = snomed.String
    }
    if value.Valid {
        v := value.Float64
        o.Value = &v
    }
    if valueText.Valid {
        o.ValueText = valueText.String
    }
    if unit.Valid {
        o.Unit = unit.String
    }
    if sourceID.Valid {
        sid := sourceID.UUID
        o.SourceID = &sid
    }
    if len(deltaBytes) > 0 {
        var d models.Delta
        if err := json.Unmarshal(deltaBytes, &d); err != nil {
            return models.Observation{}, fmt.Errorf("unmarshal delta: %w", err)
        }
        o.Delta = &d
    }
    return o, nil
}

// GetObservation reads a single Observation through the observations_v2 view.
func (s *V2SubstrateStore) GetObservation(ctx context.Context, id uuid.UUID) (*models.Observation, error) {
    q := `SELECT ` + observationColumns + ` FROM observations_v2 WHERE id = $1`
    o, err := scanObservation(s.db.QueryRowContext(ctx, q, id))
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, fmt.Errorf("get observation %s: %w", id, interfaces.ErrNotFound)
        }
        return nil, fmt.Errorf("get observation %s: %w", id, err)
    }
    return &o, nil
}

// vitalTypeKey resolves an Observation to the BaselineProvider vital-type key.
// Priority: LOINC code, then SNOMED code, then a fallback derived from Kind.
// kb-26's AcuteRepository keys on LOINC for vitals/labs and on a model-internal
// string for weight/mobility — this resolver mirrors that precedence.
func vitalTypeKey(o models.Observation) string {
    if o.LOINCCode != "" {
        return o.LOINCCode
    }
    if o.SNOMEDCode != "" {
        return o.SNOMEDCode
    }
    return o.Kind
}

// UpsertObservation writes an Observation row, computing Delta first via the
// injected delta.BaselineProvider. If the provider is unset OR returns
// delta.ErrNoBaseline OR returns any other error, the resulting Delta has
// DirectionalFlag = no_baseline and the row still persists (writes are NOT
// blocked by baseline unavailability).
//
// Writes go to the greenfield observations table. Reads come back through the
// observations_v2 view (which UNIONs lab_entries) — so v2 writers see their
// own writes via GetObservation immediately.
func (s *V2SubstrateStore) UpsertObservation(ctx context.Context, o models.Observation) (*models.Observation, error) {
    // Resolve baseline (best-effort; failures degrade to no_baseline).
    var baseline *delta.Baseline
    if s.baselineProvider != nil && o.Value != nil && o.Kind != models.ObservationKindBehavioural {
        bl, err := s.baselineProvider.FetchBaseline(ctx, o.ResidentID, vitalTypeKey(o))
        if err == nil {
            baseline = bl
        }
        // err (incl. ErrNoBaseline) → baseline stays nil → ComputeDelta yields no_baseline
    }
    d := delta.ComputeDelta(o, baseline)
    o.Delta = &d

    deltaJSON, err := json.Marshal(o.Delta)
    if err != nil {
        return nil, fmt.Errorf("marshal delta: %w", err)
    }

    const q = `
        INSERT INTO observations
            (id, resident_id, loinc_code, snomed_code, kind,
             value, value_text, unit, observed_at, source_id, delta, created_at)
        VALUES
            ($1, $2, $3, $4, $5,
             $6, $7, $8, $9, $10, $11, NOW())
        ON CONFLICT (id) DO UPDATE SET
            resident_id = EXCLUDED.resident_id,
            loinc_code  = EXCLUDED.loinc_code,
            snomed_code = EXCLUDED.snomed_code,
            kind        = EXCLUDED.kind,
            value       = EXCLUDED.value,
            value_text  = EXCLUDED.value_text,
            unit        = EXCLUDED.unit,
            observed_at = EXCLUDED.observed_at,
            source_id   = EXCLUDED.source_id,
            delta       = EXCLUDED.delta
    `

    var valueArg interface{}
    if o.Value != nil {
        valueArg = *o.Value
    }
    var sourceArg interface{}
    if o.SourceID != nil {
        sourceArg = *o.SourceID
    }

    if _, err := s.db.ExecContext(ctx, q,
        o.ID,                       // $1
        o.ResidentID,               // $2
        nilIfEmpty(o.LOINCCode),    // $3
        nilIfEmpty(o.SNOMEDCode),   // $4
        o.Kind,                     // $5
        valueArg,                   // $6
        nilIfEmpty(o.ValueText),    // $7
        nilIfEmpty(o.Unit),         // $8
        o.ObservedAt,               // $9
        sourceArg,                  // $10
        deltaJSON,                  // $11
    ); err != nil {
        return nil, fmt.Errorf("upsert observation: %w", err)
    }

    return s.GetObservation(ctx, o.ID)
}

// ListObservationsByResident returns observations for a resident, paged.
// One round-trip; no N+1.
func (s *V2SubstrateStore) ListObservationsByResident(ctx context.Context, residentID uuid.UUID, limit, offset int) ([]models.Observation, error) {
    q := `SELECT ` + observationColumns + `
          FROM observations_v2
         WHERE resident_id = $1
         ORDER BY observed_at DESC
         LIMIT $2 OFFSET $3`
    rows, err := s.db.QueryContext(ctx, q, residentID, limit, offset)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var out []models.Observation
    for rows.Next() {
        o, err := scanObservation(rows)
        if err != nil {
            return nil, err
        }
        out = append(out, o)
    }
    return out, rows.Err()
}

// ListObservationsByResidentAndKind filters ListObservationsByResident on kind.
func (s *V2SubstrateStore) ListObservationsByResidentAndKind(ctx context.Context, residentID uuid.UUID, kind string, limit, offset int) ([]models.Observation, error) {
    q := `SELECT ` + observationColumns + `
          FROM observations_v2
         WHERE resident_id = $1 AND kind = $2
         ORDER BY observed_at DESC
         LIMIT $3 OFFSET $4`
    rows, err := s.db.QueryContext(ctx, q, residentID, kind, limit, offset)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var out []models.Observation
    for rows.Next() {
        o, err := scanObservation(rows)
        if err != nil {
            return nil, err
        }
        out = append(out, o)
    }
    return out, rows.Err()
}
```

- [ ] **Step 4: Add baselineProvider field to V2SubstrateStore struct**

Edit the existing struct definition near the top of `v2_substrate_store.go`:

```go
type V2SubstrateStore struct {
    db               *sql.DB
    baselineProvider delta.BaselineProvider // injected via SetBaselineProvider; nil → all writes get Delta.Flag=no_baseline
}
```

And ensure the import block includes `"github.com/cardiofit/shared/v2_substrate/delta"`. Verify with `go build ./...`.

- [ ] **Step 5: Build + run test**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile && \
  go build ./... && \
  go test ./internal/storage/... -v 2>&1 | tail -30
```

Expected: build clean. Without `KB20_TEST_DATABASE_URL`: tests skip cleanly (SKIP messages). With env set + migration 008_part2_partB applied: PASS for all observation storage tests.

- [ ] **Step 6: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/storage/v2_substrate_store.go \
        backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/storage/v2_substrate_observation_test.go
git commit -m "feat(kb-20): V2SubstrateStore Observation methods + Delta-on-write via injected BaselineProvider"
```

---

## Task 12: kb-20 REST handlers — 4 Observation routes

**Files:**
- Modify: `<kbs>/kb-20-patient-profile/internal/api/v2_substrate_handlers.go`

- [ ] **Step 1: Add the 4 routes to RegisterRoutes (around line 60, after the medicine_uses block)**

Path: `<kbs>/kb-20-patient-profile/internal/api/v2_substrate_handlers.go`

In `RegisterRoutes`, append after the existing `g.GET("/residents/:resident_id/medicine_uses", h.listMedicineUsesByResident)` line:

```go
    g.POST("/observations", h.upsertObservation)
    g.GET("/observations/:id", h.getObservation)
    g.GET("/residents/:resident_id/observations", h.listObservationsByResident)
    g.GET("/residents/:resident_id/observations/:kind", h.listObservationsByResidentAndKind)
```

- [ ] **Step 2: Append the handler implementations at end of file**

Path: same file (append at end)

```go
// ---------------------------------------------------------------------------
// Observation
// ---------------------------------------------------------------------------

func (h *V2SubstrateHandlers) upsertObservation(c *gin.Context) {
    var o models.Observation
    if err := c.BindJSON(&o); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    if err := validation.ValidateObservation(o); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    out, err := h.store.UpsertObservation(c.Request.Context(), o)
    if err != nil {
        respondError(c, err)
        return
    }
    c.JSON(http.StatusOK, out)
}

func (h *V2SubstrateHandlers) getObservation(c *gin.Context) {
    id, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return
    }
    o, err := h.store.GetObservation(c.Request.Context(), id)
    if err != nil {
        respondError(c, err)
        return
    }
    c.JSON(http.StatusOK, o)
}

func (h *V2SubstrateHandlers) listObservationsByResident(c *gin.Context) {
    residentID, err := uuid.Parse(c.Param("resident_id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resident_id"})
        return
    }
    limit, err := strconv.Atoi(c.DefaultQuery("limit", "100"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
        return
    }
    if limit <= 0 || limit > 1000 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be in (0, 1000]"})
        return
    }
    offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
    if err != nil || offset < 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset"})
        return
    }
    out, err := h.store.ListObservationsByResident(c.Request.Context(), residentID, limit, offset)
    if err != nil {
        respondError(c, err)
        return
    }
    c.JSON(http.StatusOK, out)
}

func (h *V2SubstrateHandlers) listObservationsByResidentAndKind(c *gin.Context) {
    residentID, err := uuid.Parse(c.Param("resident_id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resident_id"})
        return
    }
    kind := c.Param("kind")
    if !models.IsValidObservationKind(kind) {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid kind"})
        return
    }
    limit, err := strconv.Atoi(c.DefaultQuery("limit", "100"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
        return
    }
    if limit <= 0 || limit > 1000 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be in (0, 1000]"})
        return
    }
    offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
    if err != nil || offset < 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset"})
        return
    }
    out, err := h.store.ListObservationsByResidentAndKind(c.Request.Context(), residentID, kind, limit, offset)
    if err != nil {
        respondError(c, err)
        return
    }
    c.JSON(http.StatusOK, out)
}
```

- [ ] **Step 3: Build to confirm wiring is sound**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile && \
  go build ./...
```

Expected: build clean.

- [ ] **Step 4: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/api/v2_substrate_handlers.go
git commit -m "feat(kb-20): 4 Observation REST endpoints (POST /observations, GET by id, list by resident, list by resident+kind)"
```

---

## Task 13: KB20Client Observation methods

**Files:**
- Modify: `<kbs>/shared/v2_substrate/client/kb20_client.go`

- [ ] **Step 1: Append the 4 client methods after the existing MedicineUse block**

Path: `<kbs>/shared/v2_substrate/client/kb20_client.go` (append after `ListMedicineUsesByResident`, before the `// Internal helper` comment)

```go
// ---------------------------------------------------------------------------
// Observation
// ---------------------------------------------------------------------------

func (c *KB20Client) UpsertObservation(ctx context.Context, o models.Observation) (*models.Observation, error) {
    return doJSON[models.Observation](ctx, c.http, http.MethodPost, c.baseURL+"/v2/observations", o)
}

func (c *KB20Client) GetObservation(ctx context.Context, id uuid.UUID) (*models.Observation, error) {
    return doJSON[models.Observation](ctx, c.http, http.MethodGet, c.baseURL+"/v2/observations/"+id.String(), nil)
}

func (c *KB20Client) ListObservationsByResident(ctx context.Context, residentID uuid.UUID, limit, offset int) ([]models.Observation, error) {
    q := url.Values{}
    q.Set("limit", strconv.Itoa(limit))
    q.Set("offset", strconv.Itoa(offset))
    u := c.baseURL + "/v2/residents/" + residentID.String() + "/observations?" + q.Encode()
    out, err := doJSON[[]models.Observation](ctx, c.http, http.MethodGet, u, nil)
    if err != nil {
        return nil, err
    }
    return *out, nil
}

func (c *KB20Client) ListObservationsByResidentAndKind(ctx context.Context, residentID uuid.UUID, kind string, limit, offset int) ([]models.Observation, error) {
    q := url.Values{}
    q.Set("limit", strconv.Itoa(limit))
    q.Set("offset", strconv.Itoa(offset))
    u := c.baseURL + "/v2/residents/" + residentID.String() + "/observations/" + kind + "?" + q.Encode()
    out, err := doJSON[[]models.Observation](ctx, c.http, http.MethodGet, u, nil)
    if err != nil {
        return nil, err
    }
    return *out, nil
}
```

- [ ] **Step 2: Write httptest-backed client test**

Create `<kbs>/shared/v2_substrate/client/kb20_client_observation_test.go`:

```go
package client

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
    "time"

    "github.com/google/uuid"

    "github.com/cardiofit/shared/v2_substrate/models"
)

func TestKB20Client_UpsertGetObservation(t *testing.T) {
    val := 132.0
    captured := models.Observation{}
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method == http.MethodPost && r.URL.Path == "/v2/observations" {
            _ = json.NewDecoder(r.Body).Decode(&captured)
            w.Header().Set("Content-Type", "application/json")
            _ = json.NewEncoder(w).Encode(captured)
            return
        }
        if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/observations/") {
            w.Header().Set("Content-Type", "application/json")
            _ = json.NewEncoder(w).Encode(captured)
            return
        }
        http.Error(w, "unexpected route "+r.Method+" "+r.URL.Path, http.StatusNotFound)
    }))
    defer ts.Close()

    c := NewKB20Client(ts.URL)
    in := models.Observation{
        ID: uuid.New(), ResidentID: uuid.New(),
        Kind: models.ObservationKindVital, LOINCCode: "8480-6",
        Value: &val, Unit: "mmHg", ObservedAt: time.Now().UTC(),
    }
    out, err := c.UpsertObservation(context.Background(), in)
    if err != nil {
        t.Fatalf("Upsert: %v", err)
    }
    if out.ID != in.ID {
        t.Errorf("Upsert ID drift: got %v want %v", out.ID, in.ID)
    }
    got, err := c.GetObservation(context.Background(), in.ID)
    if err != nil {
        t.Fatalf("Get: %v", err)
    }
    if got.Kind != models.ObservationKindVital {
        t.Errorf("Get Kind drift: got %q", got.Kind)
    }
}

func TestKB20Client_ListObservationsByResidentAndKind_BuildsURL(t *testing.T) {
    var seenPath string
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        seenPath = r.URL.Path + "?" + r.URL.RawQuery
        w.Header().Set("Content-Type", "application/json")
        _, _ = w.Write([]byte(`[]`))
    }))
    defer ts.Close()

    c := NewKB20Client(ts.URL)
    rid := uuid.MustParse("11111111-2222-3333-4444-555555555555")
    _, err := c.ListObservationsByResidentAndKind(context.Background(), rid, models.ObservationKindWeight, 50, 10)
    if err != nil {
        t.Fatalf("List: %v", err)
    }
    expected := "/v2/residents/11111111-2222-3333-4444-555555555555/observations/weight?limit=50&offset=10"
    if seenPath != expected {
        t.Errorf("URL mismatch: got %q want %q", seenPath, expected)
    }
}
```

- [ ] **Step 3: Run test — expect PASS**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && \
  go test ./v2_substrate/client/... -v
```

Expected: PASS for both new tests + existing β.1/β.2-A client tests.

- [ ] **Step 4: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/client/kb20_client.go \
        backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/client/kb20_client_observation_test.go
git commit -m "feat(v2_substrate/client): KB20Client Observation methods + httptest-backed tests"
```

---

## Task 14: Integration test — Delta-on-write end-to-end

**Files:**
- Create: `<kbs>/kb-20-patient-profile/internal/api/v2_substrate_observation_handlers_test.go`

- [ ] **Step 1: Write the integration test (DB-gated)**

Path: `<kbs>/kb-20-patient-profile/internal/api/v2_substrate_observation_handlers_test.go`

```go
package api

import (
    "context"
    "os"
    "testing"
    "time"

    "github.com/google/uuid"

    "github.com/cardiofit/shared/v2_substrate/delta"
    "github.com/cardiofit/shared/v2_substrate/models"

    "kb-patient-profile/internal/storage"
)

func openIntegrationStore(t *testing.T) (*storage.V2SubstrateStore, *storage.InMemoryBaselineProvider) {
    t.Helper()
    dsn := os.Getenv("KB20_TEST_DATABASE_URL")
    if dsn == "" {
        t.Skip("KB20_TEST_DATABASE_URL not set; skipping delta-on-write integration test")
    }
    store, err := storage.NewV2SubstrateStore(dsn)
    if err != nil {
        t.Fatalf("open store: %v", err)
    }
    bp := storage.NewInMemoryBaselineProvider()
    store.SetBaselineProvider(bp)
    return store, bp
}

func ptr(f float64) *float64 { return &f }

// TestDeltaOnWrite_EndToEnd seeds a baseline, inserts 4 observations with
// distinct value/kind/baseline-availability profiles, and asserts each lands
// the expected DirectionalFlag. Covers spec §9 acceptance item 10.
func TestDeltaOnWrite_EndToEnd(t *testing.T) {
    store, bp := openIntegrationStore(t)
    defer store.Close()

    rid := uuid.New()
    bp.Seed(rid, "8480-6", delta.Baseline{
        BaselineValue: 130.0,
        StdDev:        8.0,
        SampleSize:    50,
        ComputedAt:    time.Now().UTC(),
    })

    // Case 1: within baseline (val=132, dev≈0.25 stddev)
    o1, err := store.UpsertObservation(context.Background(), models.Observation{
        ID: uuid.New(), ResidentID: rid, Kind: models.ObservationKindVital,
        LOINCCode: "8480-6", Value: ptr(132.0), Unit: "mmHg", ObservedAt: time.Now().UTC().Truncate(time.Second),
    })
    if err != nil {
        t.Fatalf("upsert 1: %v", err)
    }
    if o1.Delta == nil || o1.Delta.DirectionalFlag != models.DeltaFlagWithinBaseline {
        t.Errorf("case 1: expected within_baseline, got %+v", o1.Delta)
    }

    // Case 2: severely elevated (val=160, dev≈3.75 stddev)
    o2, err := store.UpsertObservation(context.Background(), models.Observation{
        ID: uuid.New(), ResidentID: rid, Kind: models.ObservationKindVital,
        LOINCCode: "8480-6", Value: ptr(160.0), Unit: "mmHg", ObservedAt: time.Now().UTC().Truncate(time.Second),
    })
    if err != nil {
        t.Fatalf("upsert 2: %v", err)
    }
    if o2.Delta == nil || o2.Delta.DirectionalFlag != models.DeltaFlagSeverelyElevated {
        t.Errorf("case 2: expected severely_elevated, got %+v", o2.Delta)
    }

    // Case 3: behavioural — must yield no_baseline regardless of seeded data
    o3, err := store.UpsertObservation(context.Background(), models.Observation{
        ID: uuid.New(), ResidentID: rid, Kind: models.ObservationKindBehavioural,
        ValueText: "agitation episode 14:30", ObservedAt: time.Now().UTC().Truncate(time.Second),
    })
    if err != nil {
        t.Fatalf("upsert 3: %v", err)
    }
    if o3.Delta == nil || o3.Delta.DirectionalFlag != models.DeltaFlagNoBaseline {
        t.Errorf("case 3: expected no_baseline for behavioural, got %+v", o3.Delta)
    }

    // Case 4: vital with NO seeded baseline (different LOINC) — no_baseline
    o4, err := store.UpsertObservation(context.Background(), models.Observation{
        ID: uuid.New(), ResidentID: rid, Kind: models.ObservationKindVital,
        LOINCCode: "8462-4", // diastolic — not seeded
        Value: ptr(85.0), Unit: "mmHg", ObservedAt: time.Now().UTC().Truncate(time.Second),
    })
    if err != nil {
        t.Fatalf("upsert 4: %v", err)
    }
    if o4.Delta == nil || o4.Delta.DirectionalFlag != models.DeltaFlagNoBaseline {
        t.Errorf("case 4: expected no_baseline for unseeded LOINC, got %+v", o4.Delta)
    }
}
```

- [ ] **Step 2: Build + run**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile && \
  go build ./... && \
  go test ./internal/api/... -run "TestDeltaOnWrite_EndToEnd" -v
```

Expected without DB env: SKIP. With DB env + migration applied: PASS for all 4 cases.

- [ ] **Step 3: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/api/v2_substrate_observation_handlers_test.go
git commit -m "test(kb-20): delta-on-write end-to-end integration test (4 cases: within / severe / behavioural / unseeded)"
```

---

## Task 15: Spec status update — append β.2-B + β.2-C completion blocks

**Files:**
- Modify: `<kbs>/../docs/superpowers/specs/2026-05-04-1b-beta-2-clinical-primitives-design.md`

> Only complete this task at the end after every prior task is green.

- [ ] **Step 1: Run the full test sweep + record exact counts**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && \
  go test ./v2_substrate/... -count=1 2>&1 | tail -20

cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile && \
  go build ./... && go vet ./... && go test ./internal/storage/... ./internal/api/... -count=1 2>&1 | tail -20
```

Record per-package PASS counts and the number of skipped DB-gated tests for inclusion in the completion block.

- [ ] **Step 2: Append completion blocks to the spec**

Path: `/Volumes/Vaidshala/cardiofit/docs/superpowers/specs/2026-05-04-1b-beta-2-clinical-primitives-design.md`

Append after the existing β.2-A Completion block. Mirror β.2-A's exact format (✅ marker, test counts, migration sanity counts, exit criteria checked off, follow-ups). Two blocks: one for β.2-B, one for β.2-C.

Suggested header for β.2-B block:

```markdown
### β.2-B Completion (2026-05-04)

✅ **Cluster complete.** Observation v2 substrate entity delivered end-to-end across 6 review checkpoints (C1 enums verified + Observation/Delta types, C2 validator, C3 FHIR mapper + extensions, C4 interface + migration, C5 storage + handlers + client, C6 verification).

**Test counts (final):**
- shared/v2_substrate/models: <N> tests pass (incl. 4 Observation cases — vital roundtrip, behavioural ValueText-only, Delta roundtrip, omit-empty)
- shared/v2_substrate/validation: <N> tests pass (incl. 8 Observation cases — value-or-text, kind enum, zero residency, zero observedAt, vital range, weight positive)
- shared/v2_substrate/fhir: <N> tests pass (incl. 8 Observation cases — 5 kinds × roundtrip + RejectsInvalid + WrongResourceType + WireFormat)
- shared/v2_substrate/client: <N> tests pass (incl. 2 Observation httptest cases)
- kb-20 storage + handlers: SKIP cleanly without `KB20_TEST_DATABASE_URL`; build clean
- Total: <N> unit tests across shared packages, all green; vet clean

**Migration:** `008_part2_clinical_primitives_partB.sql` — non-breaking; greenfield observations table (12 columns, value-or-text CHECK, 3 indexes) + observations_v2 view UNIONing lab_entries with kind='lab'; sanity counts: 1 BEGIN/COMMIT, 1 CREATE TABLE, 3 CREATE INDEX, 1 CREATE OR REPLACE VIEW, 6 COMMENT ON COLUMN, 0 uuid_generate_v4 leftovers. **Not yet applied to any DB** (deferred to deployment).

**β.2-B specific contributions:**
- Observation `Value *float64` pointer-nullable distinguishes "no numeric, see ValueText" from "value=0.0" — preserved through SQL (sql.NullFloat64) and JSON (omitempty)
- Per-kind validator ranges (BP 1-300/1-200, weight >0, lab >0, mobility >=0) enforce sanity at the egress AND ingress mappers (defense-in-depth)
- observations_v2 view UNIONs lab_entries with NULL projections for source_id/delta/snomed_code where legacy table has no equivalent column — documented in COMMENT ON VIEW
- Vaidshala-namespaced FHIR extensions (ExtObservationKind / ExtObservationDelta / ExtObservationSourceID) round-trip kind discriminator + Delta JSON + UUID source provenance with no native FHIR equivalent
- AU spelling "behavioural" canonical; US "behavioral" intentionally rejected by IsValidObservationKind

**Open follow-ups for β.2-C / future (deferred, non-blocking):**
1. Replace InMemoryBaselineProvider with kb-26 AcuteRepository network adapter (interface seam already in place)
2. Negative-pagination integration test against real handler (currently httptest unit-level only)
3. Application-level kb-22.clinical_sources existence check on Observation.SourceID (currently UUID-only without runtime cross-DB validation)
```

Suggested header for β.2-C block:

```markdown
### β.2-C Completion (2026-05-04)

✅ **Cluster complete.** Delta-on-write service delivered as a pure function + interface seam, integrated into V2SubstrateStore.UpsertObservation.

**Test counts (final):**
- shared/v2_substrate/delta: 10-case table-driven ComputeDelta test pass (within / elevated / severely_elevated / low / severely_low / no_baseline × 5 trigger conditions)
- kb-20 InMemoryBaselineProvider: 2 tests pass
- kb-20 delta-on-write integration: 4-case test (within / severely_elevated / behavioural / unseeded LOINC) — DB-gated, SKIPs cleanly without `KB20_TEST_DATABASE_URL`

**β.2-C specific contributions:**
- Pure ComputeDelta function — no IO, returns models.Delta with DirectionalFlag from threshold table (|dev|<=1 within; 1<dev<=2 elevated; dev>2 severely_elevated; symmetric on low side; no_baseline when baseline nil OR Value nil OR Kind=behavioural OR StdDev=0)
- delta.BaselineProvider interface + delta.ErrNoBaseline sentinel — kb-26 AcuteRepository plugs in as a thin adapter without changing the seam
- StdDev=0 div-by-zero guard returns no_baseline rather than emitting Inf/NaN
- UpsertObservation NEVER fails the write on baseline unavailability — degraded paths (provider unset, network error, ErrNoBaseline) all surface as Delta.Flag=no_baseline so the observation persists for later re-derivation when kb-26 catches up
```

- [ ] **Step 3: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add docs/superpowers/specs/2026-05-04-1b-beta-2-clinical-primitives-design.md
git commit -m "docs(v2_substrate): mark β.2-B + β.2-C complete with test counts and migration sanity counts"
```

---

## Acceptance Criteria Coverage Map (spec §9)

| Spec §9 Item | Task(s) |
|---|---|
| 1. models/observation.go with JSON round-trip tests | Task 2 |
| 2. fhir/observation_mapper.go with roundtrip + RejectsInvalid + WrongResourceType + WireFormat | Task 5 |
| 4. shared/v2_substrate/delta/ package with table-driven ComputeDelta tests | Tasks 8 + 9 |
| 5. interfaces/storage.go declares ObservationStore | Task 6 |
| 6. KB20Client exposes 4 new Observation methods | Task 13 |
| 8. V2SubstrateStore implements ObservationStore | Task 11 |
| 9. kb-20 v2 handlers expose 4 new REST endpoints | Task 12 |
| 10. UpsertObservation populates Delta correctly; falls back to no_baseline | Tasks 9 + 11 + 14 |
| 12. observations_v2 view returns rows from observations + lab_entries with kind='lab' | Task 7 |

---

## Spec gaps / clarifications surfaced during planning

1. **lab_entries projection in observations_v2 view** — spec §5.4 says "UNION ALL SELECT from lab_entries projected into Observation shape with kind='lab'" but does not specify how to populate `resident_id` (lab_entries.patient_id is VARCHAR(100), not a UUID). The plan resolves this with a correlated subquery `(SELECT pp.id FROM patient_profiles pp WHERE pp.patient_id = le.patient_id LIMIT 1)` mirroring β.2-A's medicine_uses_v2 backfill. Performance follow-up parallel to β.2-A's open-follow-up #1.

2. **vitalType key resolution** — spec §3.4/§7 reference kb-26's `AcuteRepository.FetchBaseline(patientID, vitalType)` but the spec does not pin the exact `vitalType` string format used by kb-26. The plan introduces `vitalTypeKey(o)` with precedence LOINC → SNOMED → Kind. When the kb-26 adapter lands, this resolver may need a translation table — flagged as part of Open follow-up #1 in the β.2-B completion block.

3. **observation Value SQL precision** — spec §2.2 declares `value DECIMAL(12,4)`. The plan stores Go float64 round-trip via sql.NullFloat64; this loses precision for numbers requiring more than 15 significant decimal digits. For aged-care vitals/labs/weight this is non-binding; flag for future labs requiring more precision.

4. **Threshold inclusivity** — spec §3.7 describes thresholds informally ("within when |dev|≤1, elevated when 1<dev≤2"). The plan pins these as `abs <= thresholdWithin` (1.0 inclusive on within side) and `deviation > thresholdSevere` (2.0 inclusive on elevated side). Boundary semantics codified in compute.go.

5. **ObservationKind + DeltaFlag enum constant origin** — Task 1 verifies whether β.2-A already added these constants (the spec listed them under §3.7 as part of β.2-A's deliverable scope but β.2-A's primary focus was MedicineUse). If verified absent, Task 1 adds them; if present, Task 1 short-circuits with `[verified-only]`. No spec change needed.
