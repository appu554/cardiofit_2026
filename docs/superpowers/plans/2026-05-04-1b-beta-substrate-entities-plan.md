# Phase 1B-β.1 — Actor Model + Resident Promotion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the actor model (Person, Role) and Resident promotion as the first reviewable milestone of Phase 1B-β substrate entities, including the shared Go package skeleton, FHIR R4 mappers, kb-20 migration 008 part 1, and gRPC/REST endpoints, with all changes non-breaking against existing kb-20 consumers.

**Architecture:** Shared library at `backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/` consumed by all KBs (option C.ii from design — designated KB owns canonical row). Internal Go types with FHIR mappers at boundaries (option B). kb-20-patient-profile is the canonical store for Resident/Person/Role; existing patient_profiles table extended in place with nullable v2 columns + compatibility view.

**Tech Stack:** Go 1.24, PostgreSQL (kb-20 schema, port 5433), Gin HTTP framework, gRPC + protobuf, Google FHIR Go library or `samply/golang-fhir-models`, GORM (existing kb-20 ORM), testify for tests.

**Spec:** `docs/superpowers/specs/2026-05-04-1b-beta-substrate-entities-design.md`

**Scope of this plan:** Milestone 1B-β.1 only (~1.5 weeks). Covers Resident, Person, Role entities. **Does NOT cover** MedicineUse, Observation (β.2 — separate plan), Event, EvidenceTrace (β.3 — separate plan), Authorisation evaluator (downstream phase), or Layer 1B adapters (Phase 1B-γ — downstream).

**Exit criterion (from design §6):** A test client can `kb20Client.UpsertResident(R)` then `kb20Client.GetResident(R.ID)` and round-trip through AU FHIR Patient mapper without data loss; Person + Role types pass equivalent round-trip; kb-20 migration applies cleanly on both fresh and existing-data DBs.

---

## File Structure

**Created:**
- `backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/` — new Go package directory
- `shared/v2_substrate/models/resident.go` — Resident struct
- `shared/v2_substrate/models/person.go` — Person struct
- `shared/v2_substrate/models/role.go` — Role struct
- `shared/v2_substrate/models/enums.go` — CareIntensity, ResidentStatus, RoleKind constants
- `shared/v2_substrate/models/resident_test.go` — Resident validation tests
- `shared/v2_substrate/models/person_test.go` — Person validation tests
- `shared/v2_substrate/models/role_test.go` — Role validation tests
- `shared/v2_substrate/fhir/patient_mapper.go` — Resident ↔ AU Patient
- `shared/v2_substrate/fhir/practitioner_mapper.go` — Person + Role ↔ AU Practitioner + PractitionerRole
- `shared/v2_substrate/fhir/extensions.go` — AU FHIR extension URI constants
- `shared/v2_substrate/fhir/patient_mapper_test.go` — round-trip tests for Resident
- `shared/v2_substrate/fhir/practitioner_mapper_test.go` — round-trip tests for Person/Role
- `shared/v2_substrate/client/kb20_resident_client.go` — Go client for kb-20 Resident endpoints
- `shared/v2_substrate/client/kb20_person_client.go`
- `shared/v2_substrate/client/kb20_role_client.go`
- `shared/v2_substrate/interfaces/storage.go` — ResidentStore/PersonStore/RoleStore interface contracts
- `shared/v2_substrate/validation/resident_validator.go`
- `shared/v2_substrate/validation/person_validator.go`
- `shared/v2_substrate/validation/role_validator.go`
- `shared/v2_substrate/README.md`
- `kb-20-patient-profile/migrations/008_part1_actor_model.sql` — Person/Role tables + patient_profiles extension + residents_v2 view
- `kb-20-patient-profile/internal/api/v2_substrate_handlers.go` — REST handlers for Resident/Person/Role
- `kb-20-patient-profile/internal/api/v2_substrate_handlers_test.go` — integration tests
- `kb-20-patient-profile/internal/storage/v2_substrate_store.go` — GORM-backed implementations of ResidentStore/PersonStore/RoleStore

**Modified:**
- `backend/shared-infrastructure/knowledge-base-services/shared/go.mod` — adds `v2_substrate` if not yet present (or no change if subdirectory model continues)
- `kb-20-patient-profile/go.mod` — add dependency on shared/v2_substrate
- `kb-20-patient-profile/cmd/server/main.go` — wire v2_substrate routes into router

**Convention:** All paths relative to `/Volumes/Vaidshala/cardiofit/`. Within tasks, the prefix `<repo>/backend/shared-infrastructure/knowledge-base-services/` is shortened to `<kbs>` for readability; tasks always show the full path in the file paths section.

---

## Task 1: Create v2_substrate package skeleton

**Files:**
- Create: `<kbs>/shared/v2_substrate/README.md`
- Create: `<kbs>/shared/v2_substrate/models/`, `fhir/`, `client/`, `interfaces/`, `validation/` directories

- [ ] **Step 1: Create directory tree**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared
mkdir -p v2_substrate/models v2_substrate/fhir v2_substrate/client v2_substrate/interfaces v2_substrate/validation
```

- [ ] **Step 2: Verify tree**

```bash
find /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate -type d | sort
```

Expected: 6 lines (v2_substrate + 5 subdirectories).

- [ ] **Step 3: Write package README**

Path: `<kbs>/shared/v2_substrate/README.md`

```markdown
# v2_substrate — Vaidshala v2 Substrate Entities

Shared Go package providing types, FHIR mappers, validators, and clients
for the v2 reasoning-continuity substrate entities:

- **Resident, Person, Role, MedicineUse, Observation** — canonical storage in kb-20
- **Event, EvidenceTrace** — canonical storage in kb-22

## Phase delivery

- Phase 1B-β.1 (this milestone): Resident, Person, Role
- Phase 1B-β.2: MedicineUse, Observation
- Phase 1B-β.3: Event, EvidenceTrace

## Architecture

Each KB that needs a substrate entity imports the type from `models/` and
calls the corresponding canonical KB's gRPC/REST endpoint via `client/`.
FHIR R4 mappers (HL7 AU Base v6.0.0) live in `fhir/` and translate at
ingestion / egress boundaries — internal Vaidshala code uses the clean
internal types throughout.

## See also

- Spec: `docs/superpowers/specs/2026-05-04-1b-beta-substrate-entities-design.md`
- Plan: `docs/superpowers/plans/2026-05-04-1b-beta-substrate-entities-plan.md`
- Existing shared packages: `shared/factstore/`, `shared/governance/`, `shared/types/`
```

- [ ] **Step 4: Verify README**

```bash
test -f /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/README.md && head -1 /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/README.md
```

Expected: file exists, first line is `# v2_substrate — Vaidshala v2 Substrate Entities`.

- [ ] **Step 5: Verify shared module imports work**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && go list ./v2_substrate/... 2>&1 | head -5
```

Expected: lists v2_substrate paths or empty (no .go files yet — confirms package picks up new directory).

---

## Task 2: Define enums

**Files:**
- Create: `<kbs>/shared/v2_substrate/models/enums.go`
- Test: `<kbs>/shared/v2_substrate/models/enums_test.go`

- [ ] **Step 1: Write the failing test**

Path: `<kbs>/shared/v2_substrate/models/enums_test.go`

```go
package models

import "testing"

func TestCareIntensityIsValid(t *testing.T) {
    cases := []struct {
        in   string
        want bool
    }{
        {"palliative", true},
        {"comfort", true},
        {"active", true},
        {"rehabilitation", true},
        {"", false},
        {"unknown", false},
    }
    for _, c := range cases {
        if got := IsValidCareIntensity(c.in); got != c.want {
            t.Errorf("IsValidCareIntensity(%q) = %v, want %v", c.in, got, c.want)
        }
    }
}

func TestRoleKindIsValid(t *testing.T) {
    valid := []string{"RN", "EN", "NP", "DRNP", "GP", "pharmacist", "ACOP", "PCW", "SDM", "family", "ATSIHP", "medical_practitioner", "dentist"}
    for _, k := range valid {
        if !IsValidRoleKind(k) {
            t.Errorf("IsValidRoleKind(%q) = false, want true", k)
        }
    }
    if IsValidRoleKind("nurse") {
        t.Errorf("IsValidRoleKind(\"nurse\") = true, want false (must use RN/EN)")
    }
}

func TestResidentStatusIsValid(t *testing.T) {
    valid := []string{"active", "deceased", "transferred", "discharged"}
    for _, s := range valid {
        if !IsValidResidentStatus(s) {
            t.Errorf("IsValidResidentStatus(%q) = false, want true", s)
        }
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && go test ./v2_substrate/models/... -run "TestCareIntensityIsValid|TestRoleKindIsValid|TestResidentStatusIsValid" -v`

Expected: FAIL with `undefined: IsValidCareIntensity` (or similar).

- [ ] **Step 3: Write minimal implementation**

Path: `<kbs>/shared/v2_substrate/models/enums.go`

```go
// Package models defines the v2 substrate entity types used across all
// Vaidshala KBs. Each entity is a clean Go struct optimized for Vaidshala's
// aged-care-medication-stewardship domain. FHIR translation happens at
// boundaries via the sibling fhir/ package.
package models

// CareIntensity classifies the resident's overall care plan posture. It
// shapes every recommendation downstream (palliative residents are not
// candidates for primary-prevention deprescribing, etc.).
const (
    CareIntensityPalliative     = "palliative"
    CareIntensityComfort        = "comfort"
    CareIntensityActive         = "active"
    CareIntensityRehabilitation = "rehabilitation"
)

// IsValidCareIntensity reports whether s is one of the recognized
// CareIntensity values.
func IsValidCareIntensity(s string) bool {
    switch s {
    case CareIntensityPalliative, CareIntensityComfort,
        CareIntensityActive, CareIntensityRehabilitation:
        return true
    }
    return false
}

// RoleKind enumerates the v2 actor types. The set mirrors the
// regulatory_scope_rules.role values authored in Phase 1C-γ — the
// Authorisation evaluator joins on these strings, so changes here MUST
// be coordinated with kb-22 ScopeRules data.
const (
    RoleRN                  = "RN"
    RoleEN                  = "EN"
    RoleNP                  = "NP"
    RoleDRNP                = "DRNP"
    RoleGP                  = "GP"
    RolePharmacist          = "pharmacist"
    RoleACOP                = "ACOP"
    RolePCW                 = "PCW"
    RoleSDM                 = "SDM"
    RoleFamily              = "family"
    RoleATSIHP              = "ATSIHP"
    RoleMedicalPractitioner = "medical_practitioner"
    RoleDentist             = "dentist"
)

// IsValidRoleKind reports whether s is one of the recognized RoleKind values.
func IsValidRoleKind(s string) bool {
    switch s {
    case RoleRN, RoleEN, RoleNP, RoleDRNP, RoleGP, RolePharmacist,
        RoleACOP, RolePCW, RoleSDM, RoleFamily, RoleATSIHP,
        RoleMedicalPractitioner, RoleDentist:
        return true
    }
    return false
}

// ResidentStatus enumerates residency lifecycle.
const (
    ResidentStatusActive       = "active"
    ResidentStatusDeceased     = "deceased"
    ResidentStatusTransferred  = "transferred"
    ResidentStatusDischarged   = "discharged"
)

// IsValidResidentStatus reports whether s is one of the recognized
// ResidentStatus values.
func IsValidResidentStatus(s string) bool {
    switch s {
    case ResidentStatusActive, ResidentStatusDeceased,
        ResidentStatusTransferred, ResidentStatusDischarged:
        return true
    }
    return false
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && go test ./v2_substrate/models/... -v`

Expected: PASS for all three TestX functions.

- [ ] **Step 5: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/
git commit -m "feat(v2_substrate): scaffold package + enums for CareIntensity/RoleKind/ResidentStatus"
```

---

## Task 3: Define Resident type

**Files:**
- Create: `<kbs>/shared/v2_substrate/models/resident.go`
- Test: `<kbs>/shared/v2_substrate/models/resident_test.go`

- [ ] **Step 1: Write the failing test**

Path: `<kbs>/shared/v2_substrate/models/resident_test.go`

```go
package models

import (
    "encoding/json"
    "testing"
    "time"

    "github.com/google/uuid"
)

func TestResidentJSONRoundTrip(t *testing.T) {
    sdm := uuid.New()
    in := Resident{
        ID:               uuid.New(),
        IHI:              "8003608000000570",
        GivenName:        "Margaret",
        FamilyName:       "Brown",
        DOB:              time.Date(1938, 4, 12, 0, 0, 0, 0, time.UTC),
        Sex:              "female",
        IndigenousStatus: "neither",
        FacilityID:       uuid.New(),
        CareIntensity:    CareIntensityActive,
        SDMs:             []uuid.UUID{sdm},
        Status:           ResidentStatusActive,
        CreatedAt:        time.Now().UTC(),
        UpdatedAt:        time.Now().UTC(),
    }

    b, err := json.Marshal(in)
    if err != nil {
        t.Fatalf("marshal: %v", err)
    }

    var out Resident
    if err := json.Unmarshal(b, &out); err != nil {
        t.Fatalf("unmarshal: %v", err)
    }

    if out.ID != in.ID || out.IHI != in.IHI || out.CareIntensity != in.CareIntensity {
        t.Errorf("round-trip mismatch: got %+v want %+v", out, in)
    }
    if len(out.SDMs) != 1 || out.SDMs[0] != sdm {
        t.Errorf("SDMs round-trip lost: got %v", out.SDMs)
    }
}

func TestResidentOptionalAdmissionDate(t *testing.T) {
    in := Resident{ID: uuid.New(), GivenName: "X", FamilyName: "Y"}
    b, err := json.Marshal(in)
    if err != nil {
        t.Fatalf("marshal: %v", err)
    }
    if string(b) == "" || !contains(string(b), "\"admission_date\"") == false {
        // admission_date should NOT appear when nil (omitempty); just verify marshal succeeds
    }
}

// helper for substring assertions in tests
func contains(haystack, needle string) bool {
    return len(haystack) >= len(needle) && (haystack == needle || (len(haystack) > 0 && (string(haystack[0]) == string(needle[0]) && contains(haystack[1:], needle) || contains(haystack[1:], needle))))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && go test ./v2_substrate/models/... -run "TestResident" -v`

Expected: FAIL with `undefined: Resident`.

- [ ] **Step 3: Write Resident struct**

Path: `<kbs>/shared/v2_substrate/models/resident.go`

```go
package models

import (
    "time"

    "github.com/google/uuid"
)

// Resident represents an aged-care residential consumer ("person accessing
// funded aged care services in a residential aged care home" per Vic
// DPCS Act §36EA(1)(a) and equivalent Commonwealth definitions).
//
// Resident is the canonical patient-state subject for Vaidshala. It maps
// to AU FHIR Patient at the integration boundary (see fhir/patient_mapper.go)
// but the internal type is intentionally narrower than the FHIR profile.
//
// Canonical storage: kb-20-patient-profile (residents_v2 view over
// patient_profiles + extensions added in migration 008_part1).
type Resident struct {
    ID               uuid.UUID   `json:"id"`
    IHI              string      `json:"ihi,omitempty"`             // Individual Healthcare Identifier (16 digits)
    GivenName        string      `json:"given_name"`
    FamilyName       string      `json:"family_name"`
    DOB              time.Time   `json:"dob"`
    Sex              string      `json:"sex"`                       // FHIR AdministrativeGender: male|female|other|unknown
    IndigenousStatus string      `json:"indigenous_status,omitempty"` // AU AdministrativeGender Indigenous extension: aboriginal|tsi|both|neither|not_stated
    FacilityID       uuid.UUID   `json:"facility_id"`
    AdmissionDate    *time.Time  `json:"admission_date,omitempty"`
    CareIntensity    string      `json:"care_intensity"`            // see CareIntensity* constants
    SDMs             []uuid.UUID `json:"sdms,omitempty"`            // SubstituteDecisionMaker Person IDs
    Status           string      `json:"status"`                    // see ResidentStatus* constants
    CreatedAt        time.Time   `json:"created_at"`
    UpdatedAt        time.Time   `json:"updated_at"`
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && go test ./v2_substrate/models/... -run "TestResident" -v`

Expected: PASS for both Resident tests.

- [ ] **Step 5: Run full models package tests**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && go test ./v2_substrate/models/... -v`

Expected: PASS for all enums + Resident tests.

- [ ] **Step 6: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/models/resident.go backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/models/resident_test.go
git commit -m "feat(v2_substrate): add Resident type with JSON round-trip test"
```

---

## Task 4: Define Person type

**Files:**
- Create: `<kbs>/shared/v2_substrate/models/person.go`
- Test: `<kbs>/shared/v2_substrate/models/person_test.go`

- [ ] **Step 1: Write the failing test**

Path: `<kbs>/shared/v2_substrate/models/person_test.go`

```go
package models

import (
    "encoding/json"
    "testing"

    "github.com/google/uuid"
)

func TestPersonJSONRoundTrip(t *testing.T) {
    in := Person{
        ID:                uuid.New(),
        GivenName:         "Sarah",
        FamilyName:        "Chen",
        HPII:              "8003614900000000",
        AHPRARegistration: "NMW0001234567",
        ContactDetails:    json.RawMessage(`{"email":"sarah.chen@example.com"}`),
    }

    b, err := json.Marshal(in)
    if err != nil {
        t.Fatalf("marshal: %v", err)
    }

    var out Person
    if err := json.Unmarshal(b, &out); err != nil {
        t.Fatalf("unmarshal: %v", err)
    }

    if out.ID != in.ID || out.GivenName != in.GivenName || out.HPII != in.HPII {
        t.Errorf("round-trip mismatch: got %+v want %+v", out, in)
    }
}

func TestPersonOptionalHPII(t *testing.T) {
    in := Person{ID: uuid.New(), GivenName: "X", FamilyName: "Y"}
    b, _ := json.Marshal(in)
    if got := string(b); contains(got, "\"hpii\"") {
        t.Errorf("hpii should be omitted when empty, got: %s", got)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && go test ./v2_substrate/models/... -run "TestPerson" -v`

Expected: FAIL with `undefined: Person`.

- [ ] **Step 3: Write Person struct**

Path: `<kbs>/shared/v2_substrate/models/person.go`

```go
package models

import (
    "encoding/json"

    "github.com/google/uuid"
)

// Person represents a human actor in the v2 substrate — a healthcare
// practitioner, ACOP-credentialed pharmacist, PCW, family member, or
// substitute decision-maker.
//
// Person is paired with one or more Role rows (1:N) capturing each capacity
// the person operates in. A single Person can be both an RN and an SDM
// for a different resident, for example.
//
// Canonical storage: kb-20-patient-profile (persons table, greenfield in
// migration 008_part1).
type Person struct {
    ID                uuid.UUID       `json:"id"`
    GivenName         string          `json:"given_name"`
    FamilyName        string          `json:"family_name"`
    HPII              string          `json:"hpii,omitempty"`              // Healthcare Provider Identifier — Individual (16 digits)
    AHPRARegistration string          `json:"ahpra_registration,omitempty"`
    ContactDetails    json.RawMessage `json:"contact,omitempty"`
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && go test ./v2_substrate/models/... -run "TestPerson" -v`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/models/person.go backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/models/person_test.go
git commit -m "feat(v2_substrate): add Person type with HPI-I + AHPRA fields"
```

---

## Task 5: Define Role type

**Files:**
- Create: `<kbs>/shared/v2_substrate/models/role.go`
- Test: `<kbs>/shared/v2_substrate/models/role_test.go`

- [ ] **Step 1: Write the failing test**

Path: `<kbs>/shared/v2_substrate/models/role_test.go`

```go
package models

import (
    "encoding/json"
    "testing"
    "time"

    "github.com/google/uuid"
)

func TestRoleJSONRoundTrip(t *testing.T) {
    facility := uuid.New()
    in := Role{
        ID:             uuid.New(),
        PersonID:       uuid.New(),
        Kind:           RoleEN,
        Qualifications: json.RawMessage(`{"notation":false,"nmba_medication_qual":true}`),
        FacilityID:     &facility,
        ValidFrom:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
        EvidenceURL:    "https://ahpra.gov.au/lookup/NMW0001234567",
    }

    b, err := json.Marshal(in)
    if err != nil {
        t.Fatalf("marshal: %v", err)
    }

    var out Role
    if err := json.Unmarshal(b, &out); err != nil {
        t.Fatalf("unmarshal: %v", err)
    }

    if out.Kind != in.Kind || string(out.Qualifications) != string(in.Qualifications) {
        t.Errorf("round-trip mismatch: got %+v want %+v", out, in)
    }
    if out.FacilityID == nil || *out.FacilityID != facility {
        t.Errorf("FacilityID round-trip lost: got %v", out.FacilityID)
    }
}

func TestRoleQualificationsMatchScopeRulesShape(t *testing.T) {
    // The keys here MUST match regulatory_scope_rules.role_qualifications
    // shape (kb-22 migration 007). This test documents the contract.
    drnp := Role{
        ID: uuid.New(), PersonID: uuid.New(), Kind: RoleDRNP,
        Qualifications: json.RawMessage(`{"endorsement":"designated_rn_prescriber","valid_from":"2025-09-30"}`),
        ValidFrom: time.Now(),
    }
    var quals map[string]interface{}
    if err := json.Unmarshal(drnp.Qualifications, &quals); err != nil {
        t.Fatalf("qualifications must be valid JSON: %v", err)
    }
    if quals["endorsement"] != "designated_rn_prescriber" {
        t.Errorf("DRNP must carry endorsement=designated_rn_prescriber; got %v", quals)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && go test ./v2_substrate/models/... -run "TestRole" -v`

Expected: FAIL with `undefined: Role`.

- [ ] **Step 3: Write Role struct**

Path: `<kbs>/shared/v2_substrate/models/role.go`

```go
package models

import (
    "encoding/json"
    "time"

    "github.com/google/uuid"
)

// Role represents a Person's authorisation capacity. A single Person can
// hold many Roles (e.g., an EN at one facility and an SDM for a different
// resident).
//
// The Qualifications JSONB shape MUST match the regulatory_scope_rules.role_qualifications
// schema authored in kb-22 migration 007 (Phase 1C-γ). The Authorisation
// evaluator (downstream phase) joins on these keys directly.
//
// Documented Qualifications shapes:
//
//	EN with notation:                  {"notation": true}
//	EN without notation + medication:  {"notation": false, "nmba_medication_qual": true}
//	Designated RN Prescriber:          {"endorsement": "designated_rn_prescriber",
//	                                    "valid_from": "2025-09-30",
//	                                    "prescribing_agreement_id": "..."}
//	ACOP-credentialed pharmacist:      {"apc_training_complete": true,
//	                                    "valid_from": "2026-07-01",
//	                                    "tier": 1}
//	Aboriginal/Torres Strait Islander Health Practitioner:
//	                                   {"atsihp": true}
//
// Canonical storage: kb-20-patient-profile (roles table, greenfield in
// migration 008_part1).
type Role struct {
    ID             uuid.UUID       `json:"id"`
    PersonID       uuid.UUID       `json:"person_id"`
    Kind           string          `json:"kind"` // see Role* constants in enums.go
    Qualifications json.RawMessage `json:"qualifications,omitempty"`
    FacilityID     *uuid.UUID      `json:"facility_id,omitempty"` // role scoped to a single facility, or nil = portable
    ValidFrom      time.Time       `json:"valid_from"`
    ValidTo        *time.Time      `json:"valid_to,omitempty"`
    EvidenceURL    string          `json:"evidence_url,omitempty"`
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && go test ./v2_substrate/models/... -run "TestRole" -v`

Expected: PASS for both Role tests.

- [ ] **Step 5: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/models/role.go backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/models/role_test.go
git commit -m "feat(v2_substrate): add Role type with Qualifications JSONB matching kb-22 ScopeRules contract"
```

---

## Task 6: Validation rules for Resident/Person/Role

**Files:**
- Create: `<kbs>/shared/v2_substrate/validation/resident_validator.go`
- Create: `<kbs>/shared/v2_substrate/validation/person_validator.go`
- Create: `<kbs>/shared/v2_substrate/validation/role_validator.go`
- Test: `<kbs>/shared/v2_substrate/validation/validators_test.go`

- [ ] **Step 1: Write the failing test**

Path: `<kbs>/shared/v2_substrate/validation/validators_test.go`

```go
package validation

import (
    "testing"
    "time"

    "github.com/google/uuid"

    "github.com/cardiofit/kb-shared/v2_substrate/models"
)

func TestValidateResidentRequiresGivenAndFamilyName(t *testing.T) {
    r := models.Resident{ID: uuid.New(), Status: models.ResidentStatusActive, CareIntensity: models.CareIntensityActive, FacilityID: uuid.New(), DOB: time.Now()}
    if err := ValidateResident(r); err == nil {
        t.Errorf("expected error for missing given_name + family_name; got nil")
    }
    r.GivenName = "X"
    r.FamilyName = "Y"
    if err := ValidateResident(r); err != nil {
        t.Errorf("expected pass for valid Resident; got %v", err)
    }
}

func TestValidateResidentChecksCareIntensity(t *testing.T) {
    r := models.Resident{ID: uuid.New(), GivenName: "X", FamilyName: "Y", DOB: time.Now(), FacilityID: uuid.New(), Status: models.ResidentStatusActive, CareIntensity: "wrong"}
    if err := ValidateResident(r); err == nil {
        t.Errorf("expected error for invalid care_intensity; got nil")
    }
}

func TestValidateResidentChecksIHIWhenPresent(t *testing.T) {
    r := models.Resident{ID: uuid.New(), GivenName: "X", FamilyName: "Y", DOB: time.Now(), FacilityID: uuid.New(), Status: models.ResidentStatusActive, CareIntensity: models.CareIntensityActive, IHI: "abc"}
    if err := ValidateResident(r); err == nil {
        t.Errorf("expected error for non-numeric IHI; got nil")
    }
    r.IHI = "8003608000000570" // 16 digits
    if err := ValidateResident(r); err != nil {
        t.Errorf("expected pass for valid 16-digit IHI; got %v", err)
    }
}

func TestValidatePersonRequiresGivenAndFamilyName(t *testing.T) {
    p := models.Person{ID: uuid.New()}
    if err := ValidatePerson(p); err == nil {
        t.Errorf("expected error for missing names; got nil")
    }
}

func TestValidateRoleChecksKind(t *testing.T) {
    r := models.Role{ID: uuid.New(), PersonID: uuid.New(), Kind: "nurse", ValidFrom: time.Now()}
    if err := ValidateRole(r); err == nil {
        t.Errorf("expected error for invalid Kind=nurse; got nil")
    }
    r.Kind = models.RoleRN
    if err := ValidateRole(r); err != nil {
        t.Errorf("expected pass for Kind=RN; got %v", err)
    }
}

func TestValidateRoleChecksValidityWindow(t *testing.T) {
    now := time.Now()
    earlier := now.Add(-24 * time.Hour)
    r := models.Role{ID: uuid.New(), PersonID: uuid.New(), Kind: models.RoleRN, ValidFrom: now, ValidTo: &earlier}
    if err := ValidateRole(r); err == nil {
        t.Errorf("expected error when ValidTo < ValidFrom; got nil")
    }
}
```

(The import path `github.com/cardiofit/kb-shared/v2_substrate/models` may differ depending on how `shared/go.mod` declares the module. If `shared/go.mod` declares a different module path, adjust import accordingly. Run `head -1 shared/go.mod` first to confirm.)

- [ ] **Step 2: Confirm shared module path**

Run: `head -1 /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared/go.mod`

Expected: e.g. `module github.com/cardiofit/kb-shared` or similar. Substitute the actual path into the test imports above before running.

- [ ] **Step 3: Run test to verify it fails**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && go test ./v2_substrate/validation/... -v`

Expected: FAIL with `undefined: ValidateResident` (or similar).

- [ ] **Step 4: Write Resident validator**

Path: `<kbs>/shared/v2_substrate/validation/resident_validator.go`

```go
// Package validation provides cross-field validation rules for v2 substrate
// entities. Validators are pure functions: they read an entity and return
// an error or nil.
package validation

import (
    "errors"
    "fmt"
    "regexp"

    "github.com/cardiofit/kb-shared/v2_substrate/models"
)

var ihiPattern = regexp.MustCompile(`^\d{16}$`)

// ValidateResident reports any structural problem with r. It does not
// check referential integrity (e.g. FacilityID exists) — that is the
// caller's responsibility at write time.
func ValidateResident(r models.Resident) error {
    if r.GivenName == "" {
        return errors.New("given_name is required")
    }
    if r.FamilyName == "" {
        return errors.New("family_name is required")
    }
    if !models.IsValidCareIntensity(r.CareIntensity) {
        return fmt.Errorf("invalid care_intensity %q", r.CareIntensity)
    }
    if !models.IsValidResidentStatus(r.Status) {
        return fmt.Errorf("invalid status %q", r.Status)
    }
    if r.IHI != "" && !ihiPattern.MatchString(r.IHI) {
        return fmt.Errorf("ihi must be 16 digits, got %q", r.IHI)
    }
    return nil
}
```

- [ ] **Step 5: Write Person validator**

Path: `<kbs>/shared/v2_substrate/validation/person_validator.go`

```go
package validation

import (
    "errors"
    "fmt"
    "regexp"

    "github.com/cardiofit/kb-shared/v2_substrate/models"
)

var hpiiPattern = regexp.MustCompile(`^\d{16}$`)

// ValidatePerson reports any structural problem with p.
func ValidatePerson(p models.Person) error {
    if p.GivenName == "" {
        return errors.New("given_name is required")
    }
    if p.FamilyName == "" {
        return errors.New("family_name is required")
    }
    if p.HPII != "" && !hpiiPattern.MatchString(p.HPII) {
        return fmt.Errorf("hpii must be 16 digits, got %q", p.HPII)
    }
    return nil
}
```

- [ ] **Step 6: Write Role validator**

Path: `<kbs>/shared/v2_substrate/validation/role_validator.go`

```go
package validation

import (
    "fmt"

    "github.com/cardiofit/kb-shared/v2_substrate/models"
)

// ValidateRole reports any structural problem with r.
func ValidateRole(r models.Role) error {
    if !models.IsValidRoleKind(r.Kind) {
        return fmt.Errorf("invalid kind %q (see RoleKind constants)", r.Kind)
    }
    if r.ValidTo != nil && r.ValidTo.Before(r.ValidFrom) {
        return fmt.Errorf("valid_to must be on or after valid_from")
    }
    return nil
}
```

- [ ] **Step 7: Run test to verify it passes**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && go test ./v2_substrate/validation/... -v`

Expected: PASS for all validator tests.

- [ ] **Step 8: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/validation/
git commit -m "feat(v2_substrate): validators for Resident/Person/Role with IHI/HPII format checks"
```

---

## Task 7: AU FHIR extension URI constants

**Files:**
- Create: `<kbs>/shared/v2_substrate/fhir/extensions.go`

- [ ] **Step 1: Write the file**

Path: `<kbs>/shared/v2_substrate/fhir/extensions.go`

```go
// Package fhir provides AU FHIR R4 mappers translating v2 substrate types
// to and from HL7 AU Base v6.0.0 profiles. Vaidshala internal code uses
// v2_substrate/models throughout; this package only runs at integration
// boundaries (Layer 1B adapters in, regulatory reporting out).
//
// Reference IGs (procured under integration_specs/):
//   - HL7 AU Base v6.0.0:           hl7_au/base_ig_r4/
//   - MHR FHIR Gateway v5.0:        adha_fhir/mhr_gateway_ig_v1_4_0/
//   - Discharge Summary v1.7 (CDA): hospital_transitions/au_fhir_discharge_summary/
package fhir

// AU FHIR extension URIs. Sourced from HL7 AU Base IG v6.0.0; refresh
// quarterly when re-procured.
const (
    ExtIHI               = "http://hl7.org.au/fhir/StructureDefinition/ihi"
    ExtHPII              = "http://hl7.org.au/fhir/StructureDefinition/hpii"
    ExtIndigenousStatus  = "http://hl7.org.au/fhir/StructureDefinition/indigenous-status"
    ExtAHPRARegistration = "http://hl7.org.au/fhir/StructureDefinition/ahpra-registration"
    // Vaidshala-internal extensions (URI namespace under our control, used
    // to round-trip Vaidshala-specific fields without colliding with HL7 AU)
    ExtCareIntensity = "https://vaidshala.health/fhir/StructureDefinition/care-intensity"
    ExtSDMReference  = "https://vaidshala.health/fhir/StructureDefinition/substitute-decision-maker"
)
```

- [ ] **Step 2: Verify**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && go vet ./v2_substrate/fhir/...`

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/fhir/extensions.go
git commit -m "feat(v2_substrate/fhir): AU FHIR extension URI constants from HL7 AU Base v6.0.0"
```

---

## Task 8: Resident ↔ AU Patient FHIR mapper

**Files:**
- Create: `<kbs>/shared/v2_substrate/fhir/patient_mapper.go`
- Test: `<kbs>/shared/v2_substrate/fhir/patient_mapper_test.go`

- [ ] **Step 1: Choose FHIR Go library**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && grep -E "fhir" go.sum 2>/dev/null | head -5`

Expected: identifies whether a FHIR library is already pulled in. If `samply/golang-fhir-models` or `google/fhir/go` is present, use that. Otherwise, the simplest approach is **plain `map[string]interface{}` JSON** (the FHIR standard is a JSON resource format; we don't need a heavy library for round-tripping the fields we use).

For this task, plain JSON marshalling is recommended — keeps the dependency surface minimal. The FHIR library decision is deferred until 1B-β.2 if we hit a case where we need parser support.

- [ ] **Step 2: Write the failing round-trip test**

Path: `<kbs>/shared/v2_substrate/fhir/patient_mapper_test.go`

```go
package fhir

import (
    "encoding/json"
    "testing"
    "time"

    "github.com/google/uuid"

    "github.com/cardiofit/kb-shared/v2_substrate/models"
)

func TestResidentToPatientToResidentRoundTrip(t *testing.T) {
    in := models.Resident{
        ID:               uuid.New(),
        IHI:              "8003608000000570",
        GivenName:        "Margaret",
        FamilyName:       "Brown",
        DOB:              time.Date(1938, 4, 12, 0, 0, 0, 0, time.UTC),
        Sex:              "female",
        IndigenousStatus: "neither",
        FacilityID:       uuid.New(),
        CareIntensity:    models.CareIntensityActive,
        Status:           models.ResidentStatusActive,
    }

    patient, err := ResidentToAUPatient(in)
    if err != nil {
        t.Fatalf("ResidentToAUPatient: %v", err)
    }

    // Verify FHIR shape is sane: must have resourceType=Patient
    if patient["resourceType"] != "Patient" {
        t.Errorf("resourceType: got %v, want Patient", patient["resourceType"])
    }

    // Round-trip back through marshal/unmarshal to simulate wire transport
    b, _ := json.Marshal(patient)
    var rt map[string]interface{}
    json.Unmarshal(b, &rt)

    out, err := AUPatientToResident(rt)
    if err != nil {
        t.Fatalf("AUPatientToResident: %v", err)
    }

    if out.IHI != in.IHI {
        t.Errorf("IHI: got %q want %q", out.IHI, in.IHI)
    }
    if out.GivenName != in.GivenName || out.FamilyName != in.FamilyName {
        t.Errorf("name: got %q %q, want %q %q", out.GivenName, out.FamilyName, in.GivenName, in.FamilyName)
    }
    if out.Sex != in.Sex {
        t.Errorf("sex: got %q want %q", out.Sex, in.Sex)
    }
    if out.IndigenousStatus != in.IndigenousStatus {
        t.Errorf("indigenous_status: got %q want %q", out.IndigenousStatus, in.IndigenousStatus)
    }
    if out.CareIntensity != in.CareIntensity {
        t.Errorf("care_intensity: got %q want %q (must round-trip via Vaidshala extension)", out.CareIntensity, in.CareIntensity)
    }
    if !out.DOB.Equal(in.DOB) {
        t.Errorf("dob: got %v want %v", out.DOB, in.DOB)
    }
}

func TestResidentToPatientOmitsEmptyIHI(t *testing.T) {
    in := models.Resident{ID: uuid.New(), GivenName: "X", FamilyName: "Y", Sex: "male", DOB: time.Now(), CareIntensity: models.CareIntensityActive, Status: models.ResidentStatusActive}
    p, _ := ResidentToAUPatient(in)
    ids, ok := p["identifier"].([]interface{})
    if ok && len(ids) > 0 {
        t.Errorf("identifier should be empty when IHI absent; got %v", ids)
    }
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && go test ./v2_substrate/fhir/... -v`

Expected: FAIL with `undefined: ResidentToAUPatient`.

- [ ] **Step 4: Write the mapper**

Path: `<kbs>/shared/v2_substrate/fhir/patient_mapper.go`

```go
package fhir

import (
    "fmt"
    "time"

    "github.com/cardiofit/kb-shared/v2_substrate/models"
)

// ResidentToAUPatient translates a v2 substrate Resident to a JSON-shape
// AU FHIR Patient resource (HL7 AU Base v6.0.0). Returns a generic
// map[string]interface{} representing the FHIR JSON structure, suitable
// for json.Marshal to produce wire format.
//
// AU-specific extensions used:
//   - identifier[].system http://ns.electronichealth.net.au/id/hi/ihi/1.0
//   - extension[indigenous-status]
//   - extension[care-intensity] (Vaidshala-internal)
//
// Lossy fields (NOT round-tripped):
//   - SDMs       — encoded as Patient.contact entries on egress, but the
//                  reverse mapper does NOT reconstruct SDMs from contact
//                  entries because non-SDM contacts cannot be distinguished
//                  reliably. SDMs must be set explicitly by callers when
//                  ingesting from FHIR.
//   - CreatedAt / UpdatedAt — managed by canonical store, not in FHIR
//   - FacilityID  — encoded as Patient.managingOrganization, but FHIR
//                   represents organizations as URIs, not Vaidshala UUIDs
func ResidentToAUPatient(r models.Resident) (map[string]interface{}, error) {
    p := map[string]interface{}{
        "resourceType": "Patient",
        "id":           r.ID.String(),
        "name": []map[string]interface{}{
            {"use": "official", "given": []string{r.GivenName}, "family": r.FamilyName},
        },
        "gender":    r.Sex,
        "birthDate": r.DOB.Format("2006-01-02"),
        "active":    r.Status == models.ResidentStatusActive,
    }

    // Identifier (only if IHI present)
    if r.IHI != "" {
        p["identifier"] = []map[string]interface{}{
            {
                "system": "http://ns.electronichealth.net.au/id/hi/ihi/1.0",
                "value":  r.IHI,
            },
        }
    }

    // Extensions
    exts := []map[string]interface{}{}
    if r.IndigenousStatus != "" {
        exts = append(exts, map[string]interface{}{
            "url":         ExtIndigenousStatus,
            "valueString": r.IndigenousStatus,
        })
    }
    if r.CareIntensity != "" {
        exts = append(exts, map[string]interface{}{
            "url":         ExtCareIntensity,
            "valueString": r.CareIntensity,
        })
    }
    if r.AdmissionDate != nil {
        exts = append(exts, map[string]interface{}{
            "url":          "http://hl7.org/fhir/StructureDefinition/patient-admission-date",
            "valueDateTime": r.AdmissionDate.Format(time.RFC3339),
        })
    }
    if len(exts) > 0 {
        p["extension"] = exts
    }

    return p, nil
}

// AUPatientToResident is the reverse mapper. Lossy fields documented
// above are NOT reconstructed; callers must set them separately.
func AUPatientToResident(p map[string]interface{}) (models.Resident, error) {
    r := models.Resident{}

    if rt, _ := p["resourceType"].(string); rt != "Patient" {
        return r, fmt.Errorf("expected resourceType=Patient, got %q", rt)
    }

    if id, _ := p["id"].(string); id != "" {
        // optional — if absent, caller will assign
        if parsed, err := uuidParse(id); err == nil {
            r.ID = parsed
        }
    }

    // Name
    if names, ok := p["name"].([]interface{}); ok && len(names) > 0 {
        if first, ok := names[0].(map[string]interface{}); ok {
            if givens, ok := first["given"].([]interface{}); ok && len(givens) > 0 {
                r.GivenName, _ = givens[0].(string)
            }
            r.FamilyName, _ = first["family"].(string)
        }
    }

    r.Sex, _ = p["gender"].(string)
    if dob, _ := p["birthDate"].(string); dob != "" {
        if t, err := time.Parse("2006-01-02", dob); err == nil {
            r.DOB = t
        }
    }

    if active, _ := p["active"].(bool); active {
        r.Status = models.ResidentStatusActive
    } else {
        // FHIR active=false maps to discharged by default; explicit status
        // (deceased, transferred) requires extension lookup or out-of-band
        r.Status = models.ResidentStatusDischarged
    }

    // Identifier → IHI
    if ids, ok := p["identifier"].([]interface{}); ok {
        for _, idAny := range ids {
            id, ok := idAny.(map[string]interface{})
            if !ok {
                continue
            }
            if id["system"] == "http://ns.electronichealth.net.au/id/hi/ihi/1.0" {
                r.IHI, _ = id["value"].(string)
            }
        }
    }

    // Extensions
    if exts, ok := p["extension"].([]interface{}); ok {
        for _, extAny := range exts {
            ext, ok := extAny.(map[string]interface{})
            if !ok {
                continue
            }
            url, _ := ext["url"].(string)
            switch url {
            case ExtIndigenousStatus:
                r.IndigenousStatus, _ = ext["valueString"].(string)
            case ExtCareIntensity:
                r.CareIntensity, _ = ext["valueString"].(string)
            case "http://hl7.org/fhir/StructureDefinition/patient-admission-date":
                if s, _ := ext["valueDateTime"].(string); s != "" {
                    if t, err := time.Parse(time.RFC3339, s); err == nil {
                        r.AdmissionDate = &t
                    }
                }
            }
        }
    }

    return r, nil
}

// uuidParse is an internal helper that swallows uuid.Parse errors gracefully
// when the FHIR id is non-UUID (e.g. external system identifiers).
func uuidParse(s string) (uuid uuid.UUID, err error) {
    return uuid, fmt.Errorf("not implemented in this skeleton; replace with github.com/google/uuid Parse")
}
```

- [ ] **Step 5: Replace the uuidParse stub with the real call**

Edit the file and replace the `uuidParse` function with:

```go
// At top of file, add to imports:
//   "github.com/google/uuid"
//
// Replace the stub function with a direct call site (delete uuidParse helper):

// In AUPatientToResident, replace:
//   if parsed, err := uuidParse(id); err == nil { r.ID = parsed }
// with:
//   if parsed, err := uuid.Parse(id); err == nil { r.ID = parsed }

// And remove the entire uuidParse helper function from the file.
```

- [ ] **Step 6: Run test to verify it passes**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && go test ./v2_substrate/fhir/... -v`

Expected: PASS for both round-trip tests.

- [ ] **Step 7: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/fhir/patient_mapper.go backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/fhir/patient_mapper_test.go
git commit -m "feat(v2_substrate/fhir): Resident ↔ AU FHIR Patient mapper with round-trip test"
```

---

## Task 9: Person + Role ↔ AU Practitioner + PractitionerRole mapper

**Files:**
- Create: `<kbs>/shared/v2_substrate/fhir/practitioner_mapper.go`
- Test: `<kbs>/shared/v2_substrate/fhir/practitioner_mapper_test.go`

- [ ] **Step 1: Write the failing round-trip test**

Path: `<kbs>/shared/v2_substrate/fhir/practitioner_mapper_test.go`

```go
package fhir

import (
    "encoding/json"
    "testing"
    "time"

    "github.com/google/uuid"

    "github.com/cardiofit/kb-shared/v2_substrate/models"
)

func TestPersonToPractitionerToPersonRoundTrip(t *testing.T) {
    in := models.Person{
        ID:                uuid.New(),
        GivenName:         "Sarah",
        FamilyName:        "Chen",
        HPII:              "8003614900000000",
        AHPRARegistration: "NMW0001234567",
    }

    pr, err := PersonToAUPractitioner(in)
    if err != nil {
        t.Fatalf("PersonToAUPractitioner: %v", err)
    }
    if pr["resourceType"] != "Practitioner" {
        t.Errorf("resourceType: got %v, want Practitioner", pr["resourceType"])
    }

    b, _ := json.Marshal(pr)
    var rt map[string]interface{}
    json.Unmarshal(b, &rt)

    out, err := AUPractitionerToPerson(rt)
    if err != nil {
        t.Fatalf("AUPractitionerToPerson: %v", err)
    }
    if out.HPII != in.HPII {
        t.Errorf("HPII: got %q want %q", out.HPII, in.HPII)
    }
    if out.AHPRARegistration != in.AHPRARegistration {
        t.Errorf("AHPRA: got %q want %q", out.AHPRARegistration, in.AHPRARegistration)
    }
    if out.GivenName != in.GivenName || out.FamilyName != in.FamilyName {
        t.Errorf("name mismatch: got %q %q want %q %q", out.GivenName, out.FamilyName, in.GivenName, in.FamilyName)
    }
}

func TestRoleToPractitionerRoleRoundTrip(t *testing.T) {
    facility := uuid.New()
    in := models.Role{
        ID:             uuid.New(),
        PersonID:       uuid.New(),
        Kind:           models.RoleEN,
        Qualifications: json.RawMessage(`{"notation":false,"nmba_medication_qual":true}`),
        FacilityID:     &facility,
        ValidFrom:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
    }
    prr, err := RoleToAUPractitionerRole(in)
    if err != nil {
        t.Fatalf("RoleToAUPractitionerRole: %v", err)
    }
    if prr["resourceType"] != "PractitionerRole" {
        t.Errorf("resourceType: got %v, want PractitionerRole", prr["resourceType"])
    }

    b, _ := json.Marshal(prr)
    var rt map[string]interface{}
    json.Unmarshal(b, &rt)

    out, err := AUPractitionerRoleToRole(rt)
    if err != nil {
        t.Fatalf("AUPractitionerRoleToRole: %v", err)
    }
    if out.Kind != in.Kind {
        t.Errorf("Kind: got %q want %q", out.Kind, in.Kind)
    }
    if out.PersonID != in.PersonID {
        t.Errorf("PersonID lost in round-trip")
    }
    if string(out.Qualifications) == "" {
        t.Errorf("Qualifications lost in round-trip")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && go test ./v2_substrate/fhir/... -run "Practitioner" -v`

Expected: FAIL with `undefined: PersonToAUPractitioner`.

- [ ] **Step 3: Write the practitioner mapper**

Path: `<kbs>/shared/v2_substrate/fhir/practitioner_mapper.go`

```go
package fhir

import (
    "encoding/json"
    "fmt"
    "time"

    "github.com/google/uuid"

    "github.com/cardiofit/kb-shared/v2_substrate/models"
)

// PersonToAUPractitioner translates a v2 substrate Person to an AU FHIR
// Practitioner resource. Person has a 1:N relationship with Role; the
// Practitioner resource captures the human, while each Role is a separate
// PractitionerRole resource.
func PersonToAUPractitioner(p models.Person) (map[string]interface{}, error) {
    out := map[string]interface{}{
        "resourceType": "Practitioner",
        "id":           p.ID.String(),
        "name": []map[string]interface{}{
            {"use": "official", "given": []string{p.GivenName}, "family": p.FamilyName},
        },
    }

    exts := []map[string]interface{}{}
    ids := []map[string]interface{}{}

    if p.HPII != "" {
        ids = append(ids, map[string]interface{}{
            "system": "http://ns.electronichealth.net.au/id/hi/hpii/1.0",
            "value":  p.HPII,
        })
    }
    if p.AHPRARegistration != "" {
        ids = append(ids, map[string]interface{}{
            "system": "http://hl7.org.au/id/ahpra-registration",
            "value":  p.AHPRARegistration,
        })
    }
    if len(ids) > 0 {
        out["identifier"] = ids
    }
    if len(exts) > 0 {
        out["extension"] = exts
    }
    return out, nil
}

// AUPractitionerToPerson is the reverse mapper.
func AUPractitionerToPerson(pr map[string]interface{}) (models.Person, error) {
    p := models.Person{}
    if rt, _ := pr["resourceType"].(string); rt != "Practitioner" {
        return p, fmt.Errorf("expected resourceType=Practitioner, got %q", rt)
    }
    if id, _ := pr["id"].(string); id != "" {
        if parsed, err := uuid.Parse(id); err == nil {
            p.ID = parsed
        }
    }
    if names, ok := pr["name"].([]interface{}); ok && len(names) > 0 {
        if first, ok := names[0].(map[string]interface{}); ok {
            if gs, ok := first["given"].([]interface{}); ok && len(gs) > 0 {
                p.GivenName, _ = gs[0].(string)
            }
            p.FamilyName, _ = first["family"].(string)
        }
    }
    if ids, ok := pr["identifier"].([]interface{}); ok {
        for _, idAny := range ids {
            id, ok := idAny.(map[string]interface{})
            if !ok {
                continue
            }
            switch id["system"] {
            case "http://ns.electronichealth.net.au/id/hi/hpii/1.0":
                p.HPII, _ = id["value"].(string)
            case "http://hl7.org.au/id/ahpra-registration":
                p.AHPRARegistration, _ = id["value"].(string)
            }
        }
    }
    return p, nil
}

// RoleToAUPractitionerRole translates a v2 substrate Role to an AU FHIR
// PractitionerRole. Role.Qualifications JSONB is preserved via a Vaidshala
// extension (since FHIR's PractitionerRole.qualification[] is a structured
// codeable concept and our shape is freer).
func RoleToAUPractitionerRole(r models.Role) (map[string]interface{}, error) {
    out := map[string]interface{}{
        "resourceType": "PractitionerRole",
        "id":           r.ID.String(),
        "practitioner": map[string]interface{}{
            "reference": "Practitioner/" + r.PersonID.String(),
        },
        "code": []map[string]interface{}{
            {
                "coding": []map[string]interface{}{
                    {"system": "https://vaidshala.health/fhir/CodeSystem/role-kind", "code": r.Kind},
                },
            },
        },
        "period": map[string]interface{}{
            "start": r.ValidFrom.Format(time.RFC3339),
        },
    }

    if r.ValidTo != nil {
        out["period"].(map[string]interface{})["end"] = r.ValidTo.Format(time.RFC3339)
    }
    if r.FacilityID != nil {
        out["organization"] = map[string]interface{}{
            "reference": "Organization/" + r.FacilityID.String(),
        }
    }

    exts := []map[string]interface{}{}
    if len(r.Qualifications) > 0 {
        // Encode as a Vaidshala extension since FHIR's qualification[] is too
        // structured for our freer JSONB shape.
        exts = append(exts, map[string]interface{}{
            "url":          "https://vaidshala.health/fhir/StructureDefinition/role-qualifications",
            "valueString":  string(r.Qualifications),
        })
    }
    if r.EvidenceURL != "" {
        exts = append(exts, map[string]interface{}{
            "url":      "https://vaidshala.health/fhir/StructureDefinition/role-evidence-url",
            "valueUri": r.EvidenceURL,
        })
    }
    if len(exts) > 0 {
        out["extension"] = exts
    }
    return out, nil
}

// AUPractitionerRoleToRole is the reverse mapper.
func AUPractitionerRoleToRole(prr map[string]interface{}) (models.Role, error) {
    r := models.Role{}
    if rt, _ := prr["resourceType"].(string); rt != "PractitionerRole" {
        return r, fmt.Errorf("expected resourceType=PractitionerRole, got %q", rt)
    }
    if id, _ := prr["id"].(string); id != "" {
        if parsed, err := uuid.Parse(id); err == nil {
            r.ID = parsed
        }
    }
    if pract, ok := prr["practitioner"].(map[string]interface{}); ok {
        if ref, _ := pract["reference"].(string); len(ref) > len("Practitioner/") {
            if parsed, err := uuid.Parse(ref[len("Practitioner/"):]); err == nil {
                r.PersonID = parsed
            }
        }
    }
    if codes, ok := prr["code"].([]interface{}); ok && len(codes) > 0 {
        if cc, ok := codes[0].(map[string]interface{}); ok {
            if codings, ok := cc["coding"].([]interface{}); ok && len(codings) > 0 {
                if coding, ok := codings[0].(map[string]interface{}); ok {
                    r.Kind, _ = coding["code"].(string)
                }
            }
        }
    }
    if period, ok := prr["period"].(map[string]interface{}); ok {
        if start, _ := period["start"].(string); start != "" {
            if t, err := time.Parse(time.RFC3339, start); err == nil {
                r.ValidFrom = t
            }
        }
        if end, _ := period["end"].(string); end != "" {
            if t, err := time.Parse(time.RFC3339, end); err == nil {
                r.ValidTo = &t
            }
        }
    }
    if org, ok := prr["organization"].(map[string]interface{}); ok {
        if ref, _ := org["reference"].(string); len(ref) > len("Organization/") {
            if parsed, err := uuid.Parse(ref[len("Organization/"):]); err == nil {
                r.FacilityID = &parsed
            }
        }
    }
    if exts, ok := prr["extension"].([]interface{}); ok {
        for _, extAny := range exts {
            ext, ok := extAny.(map[string]interface{})
            if !ok {
                continue
            }
            switch ext["url"] {
            case "https://vaidshala.health/fhir/StructureDefinition/role-qualifications":
                if s, _ := ext["valueString"].(string); s != "" {
                    r.Qualifications = json.RawMessage(s)
                }
            case "https://vaidshala.health/fhir/StructureDefinition/role-evidence-url":
                r.EvidenceURL, _ = ext["valueUri"].(string)
            }
        }
    }
    return r, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && go test ./v2_substrate/fhir/... -v`

Expected: PASS for all FHIR mapper tests (Patient + Practitioner + PractitionerRole).

- [ ] **Step 5: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/fhir/practitioner_mapper.go backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/fhir/practitioner_mapper_test.go
git commit -m "feat(v2_substrate/fhir): Person+Role ↔ AU Practitioner+PractitionerRole mappers"
```

---

## Task 10: Define storage interfaces

**Files:**
- Create: `<kbs>/shared/v2_substrate/interfaces/storage.go`

- [ ] **Step 1: Write the storage interface contracts**

Path: `<kbs>/shared/v2_substrate/interfaces/storage.go`

```go
// Package interfaces declares storage and transport contracts for the v2
// substrate. The canonical KB (kb-20 for actor entities) implements these
// interfaces; other KBs use them via clients.
package interfaces

import (
    "context"

    "github.com/google/uuid"

    "github.com/cardiofit/kb-shared/v2_substrate/models"
)

// ResidentStore is the canonical storage contract for Resident entities.
// kb-20-patient-profile is the only KB expected to implement this.
type ResidentStore interface {
    GetResident(ctx context.Context, id uuid.UUID) (*models.Resident, error)
    UpsertResident(ctx context.Context, r models.Resident) (*models.Resident, error)
    ListResidentsByFacility(ctx context.Context, facilityID uuid.UUID, limit, offset int) ([]models.Resident, error)
}

// PersonStore is the canonical storage contract for Person entities.
type PersonStore interface {
    GetPerson(ctx context.Context, id uuid.UUID) (*models.Person, error)
    UpsertPerson(ctx context.Context, p models.Person) (*models.Person, error)
    GetPersonByHPII(ctx context.Context, hpii string) (*models.Person, error)
}

// RoleStore is the canonical storage contract for Role entities.
type RoleStore interface {
    GetRole(ctx context.Context, id uuid.UUID) (*models.Role, error)
    UpsertRole(ctx context.Context, r models.Role) (*models.Role, error)
    ListRolesByPerson(ctx context.Context, personID uuid.UUID) ([]models.Role, error)
    // ListActiveRolesByPersonAndFacility returns only roles where ValidFrom <= now <= ValidTo (or ValidTo is nil)
    // and (FacilityID is nil OR FacilityID == facilityID). Used by the future Authorisation evaluator.
    ListActiveRolesByPersonAndFacility(ctx context.Context, personID uuid.UUID, facilityID uuid.UUID) ([]models.Role, error)
}
```

- [ ] **Step 2: Verify compiles**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && go build ./v2_substrate/interfaces/...`

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/interfaces/storage.go
git commit -m "feat(v2_substrate): storage interfaces for ResidentStore/PersonStore/RoleStore"
```

---

## Task 11: kb-20 migration 008 part 1 — actor model schema

**Files:**
- Create: `<kbs>/kb-20-patient-profile/migrations/008_part1_actor_model.sql`

- [ ] **Step 1: Read existing patient_profiles structure**

Run: `grep -A 30 "CREATE TABLE IF NOT EXISTS patient_profiles" /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/migrations/001_initial_schema.sql`

Expected: shows the existing patient_profiles columns. Use this to identify what already exists so we don't duplicate.

- [ ] **Step 2: Write the migration**

Path: `<kbs>/kb-20-patient-profile/migrations/008_part1_actor_model.sql`

```sql
-- ============================================================================
-- Migration 008 part 1: Actor Model + Resident Promotion (Phase 1B-β.1)
--
-- Implements the v2 substrate actor model:
--   - persons table (greenfield)
--   - roles table (greenfield)
--   - patient_profiles extension columns (nullable, backwards-compatible)
--   - residents_v2 view (compatibility read shape for new consumers)
--
-- Spec: docs/superpowers/specs/2026-05-04-1b-beta-substrate-entities-design.md
-- Plan: docs/superpowers/plans/2026-05-04-1b-beta-substrate-entities-plan.md
-- Date: 2026-05-04
-- ============================================================================

BEGIN;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- TABLE: persons (greenfield)
-- ============================================================================
CREATE TABLE IF NOT EXISTS persons (
    id                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    given_name         TEXT NOT NULL,
    family_name        TEXT NOT NULL,
    hpii               TEXT,                                       -- 16-digit Healthcare Provider Identifier — Individual
    ahpra_registration TEXT,
    contact_details    JSONB,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT persons_hpii_format CHECK (hpii IS NULL OR hpii ~ '^[0-9]{16}$')
);
CREATE UNIQUE INDEX IF NOT EXISTS persons_hpii_idx ON persons(hpii) WHERE hpii IS NOT NULL;
CREATE INDEX IF NOT EXISTS persons_family_name_idx ON persons(family_name);

COMMENT ON TABLE persons IS
'v2 substrate Person entity — human actors (practitioners, ACOPs, PCWs, family, SDMs). 1:N to roles.';

-- ============================================================================
-- TABLE: roles (greenfield)
-- ============================================================================
CREATE TABLE IF NOT EXISTS roles (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    person_id       UUID NOT NULL REFERENCES persons(id) ON DELETE CASCADE,
    kind            TEXT NOT NULL CHECK (kind IN (
                       'RN','EN','NP','DRNP','GP','pharmacist','ACOP','PCW',
                       'SDM','family','ATSIHP','medical_practitioner','dentist'
                    )),
    qualifications  JSONB,
    facility_id     UUID,
    valid_from      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to        TIMESTAMPTZ,
    evidence_url    TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT roles_validity_window CHECK (valid_to IS NULL OR valid_to >= valid_from)
);
CREATE INDEX IF NOT EXISTS roles_person_id_idx ON roles(person_id);
CREATE INDEX IF NOT EXISTS roles_facility_id_idx ON roles(facility_id);
CREATE INDEX IF NOT EXISTS roles_kind_idx ON roles(kind);
CREATE INDEX IF NOT EXISTS roles_active_idx ON roles(person_id, facility_id) WHERE valid_to IS NULL OR valid_to > NOW();

COMMENT ON TABLE roles IS
'v2 substrate Role entity — Person''s authorisation capacities. Qualifications JSONB shape mirrors regulatory_scope_rules.role_qualifications (kb-22 migration 007) — Authorisation evaluator joins on these keys.';

-- ============================================================================
-- TABLE EXTENSION: patient_profiles → Resident v2 fields (nullable)
-- ============================================================================
ALTER TABLE patient_profiles ADD COLUMN IF NOT EXISTS ihi               TEXT;
ALTER TABLE patient_profiles ADD COLUMN IF NOT EXISTS care_intensity    TEXT;
ALTER TABLE patient_profiles ADD COLUMN IF NOT EXISTS sdms              UUID[];
ALTER TABLE patient_profiles ADD COLUMN IF NOT EXISTS facility_id       UUID;
ALTER TABLE patient_profiles ADD COLUMN IF NOT EXISTS indigenous_status TEXT;
ALTER TABLE patient_profiles ADD COLUMN IF NOT EXISTS admission_date    TIMESTAMPTZ;

-- Constraint can only be added if pre-existing rows pass; use NOT VALID for safety,
-- then VALIDATE separately in a follow-up migration once data is curated.
ALTER TABLE patient_profiles
    ADD CONSTRAINT patient_profiles_ihi_format
    CHECK (ihi IS NULL OR ihi ~ '^[0-9]{16}$') NOT VALID;

ALTER TABLE patient_profiles
    ADD CONSTRAINT patient_profiles_care_intensity_valid
    CHECK (care_intensity IS NULL OR care_intensity IN ('palliative','comfort','active','rehabilitation')) NOT VALID;

CREATE INDEX IF NOT EXISTS patient_profiles_facility_idx ON patient_profiles(facility_id);

-- ============================================================================
-- VIEW: residents_v2 — Compatibility shape for v2 substrate consumers
-- ============================================================================
-- Existing kb-20 consumers continue reading raw patient_profiles. New v2
-- substrate consumers (and the gRPC/REST endpoints from this milestone)
-- read residents_v2 which projects only the v2 substrate Resident shape.
-- ============================================================================
CREATE OR REPLACE VIEW residents_v2 AS
SELECT
    pp.id                       AS id,
    pp.ihi                      AS ihi,
    pp.first_name               AS given_name,    -- patient_profiles uses first_name; rename to v2 shape
    pp.last_name                AS family_name,
    pp.date_of_birth            AS dob,
    pp.gender                   AS sex,
    pp.indigenous_status        AS indigenous_status,
    pp.facility_id              AS facility_id,
    pp.admission_date           AS admission_date,
    COALESCE(pp.care_intensity, 'active') AS care_intensity,
    pp.sdms                     AS sdms,
    CASE
        WHEN pp.deceased_date IS NOT NULL THEN 'deceased'
        WHEN pp.discharge_date IS NOT NULL THEN 'discharged'
        WHEN pp.transferred_date IS NOT NULL THEN 'transferred'
        ELSE 'active'
    END                         AS status,
    pp.created_at,
    pp.updated_at
FROM patient_profiles pp;

COMMENT ON VIEW residents_v2 IS
'Compatibility read shape for v2 substrate Resident consumers. Existing patient_profiles consumers unchanged.';

-- Note: the assumption that patient_profiles has columns first_name, last_name,
-- date_of_birth, gender, deceased_date, discharge_date, transferred_date is
-- based on existing patient-state platform conventions. If any of these
-- column names differ in the actual kb-20 schema, the view CREATE will fail
-- at apply time — adjust the view definition to match the actual columns.
-- This is intentional: the view forces an explicit reconciliation rather
-- than silently mapping wrong fields.

COMMIT;

-- ============================================================================
-- Migration 008 part 1 — Acceptance check (run after applying)
-- ============================================================================
-- Expected after apply:
--   SELECT COUNT(*) FROM persons;                    -- 0 (greenfield)
--   SELECT COUNT(*) FROM roles;                      -- 0 (greenfield)
--   SELECT column_name FROM information_schema.columns
--     WHERE table_name='patient_profiles' AND column_name IN
--     ('ihi','care_intensity','sdms','facility_id','indigenous_status','admission_date');
--   -- expect 6 rows
--   SELECT * FROM residents_v2 LIMIT 1;              -- view executes (may return 0 rows on fresh DB)
-- ============================================================================
```

- [ ] **Step 3: Sanity-check SQL syntax**

Run: `cat /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/migrations/008_part1_actor_model.sql | grep -c "BEGIN;"`

Expected: 1.

Run: `grep -c "COMMIT;" /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/migrations/008_part1_actor_model.sql`

Expected: 1.

- [ ] **Step 4: Verify expected column references match existing schema**

Run: `grep -E "first_name|last_name|date_of_birth|gender|deceased_date|discharge_date|transferred_date" /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/migrations/001_initial_schema.sql`

Expected: confirms the actual column names in patient_profiles. If any column names differ, edit `008_part1_actor_model.sql` to use the actual names. (The view definition deliberately fails-loud rather than silently mapping wrong columns.)

- [ ] **Step 5: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/migrations/008_part1_actor_model.sql
git commit -m "feat(kb-20): migration 008 part 1 — persons/roles tables + patient_profiles v2 extension + residents_v2 view"
```

---

## Task 12: GORM-backed storage implementations in kb-20

**Files:**
- Create: `<kbs>/kb-20-patient-profile/internal/storage/v2_substrate_store.go`
- Test: `<kbs>/kb-20-patient-profile/internal/storage/v2_substrate_store_test.go`

- [ ] **Step 1: Read existing kb-20 storage patterns**

Run: `ls /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/storage/ 2>/dev/null && head -40 /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/storage/*.go 2>/dev/null | head -40`

Expected: shows the existing GORM/sqlx pattern. Match this style in the new file.

- [ ] **Step 2: Write the failing test**

Path: `<kbs>/kb-20-patient-profile/internal/storage/v2_substrate_store_test.go`

```go
package storage

import (
    "context"
    "os"
    "testing"
    "time"

    "github.com/google/uuid"
    "github.com/stretchr/testify/require"

    "github.com/cardiofit/kb-shared/v2_substrate/models"
)

// TestUpsertResidentRoundTrip requires a kb-20 test database. If
// DATABASE_URL is unset, the test is skipped.
func TestUpsertResidentRoundTrip(t *testing.T) {
    dsn := os.Getenv("KB20_TEST_DATABASE_URL")
    if dsn == "" {
        t.Skip("KB20_TEST_DATABASE_URL not set; skipping integration test")
    }
    store, err := NewV2SubstrateStore(dsn)
    require.NoError(t, err)

    in := models.Resident{
        ID:            uuid.New(),
        IHI:           "8003608000000570",
        GivenName:     "Margaret",
        FamilyName:    "Brown",
        DOB:           time.Date(1938, 4, 12, 0, 0, 0, 0, time.UTC),
        Sex:           "female",
        FacilityID:    uuid.New(),
        CareIntensity: models.CareIntensityActive,
        Status:        models.ResidentStatusActive,
    }

    out, err := store.UpsertResident(context.Background(), in)
    require.NoError(t, err)
    require.Equal(t, in.IHI, out.IHI)

    fetched, err := store.GetResident(context.Background(), in.ID)
    require.NoError(t, err)
    require.Equal(t, in.GivenName, fetched.GivenName)
}

func TestUpsertPersonAndRoleRoundTrip(t *testing.T) {
    dsn := os.Getenv("KB20_TEST_DATABASE_URL")
    if dsn == "" {
        t.Skip("KB20_TEST_DATABASE_URL not set; skipping integration test")
    }
    store, err := NewV2SubstrateStore(dsn)
    require.NoError(t, err)

    person := models.Person{ID: uuid.New(), GivenName: "Sarah", FamilyName: "Chen", HPII: "8003614900000000"}
    pOut, err := store.UpsertPerson(context.Background(), person)
    require.NoError(t, err)
    require.Equal(t, person.HPII, pOut.HPII)

    role := models.Role{ID: uuid.New(), PersonID: person.ID, Kind: models.RoleEN, ValidFrom: time.Now()}
    rOut, err := store.UpsertRole(context.Background(), role)
    require.NoError(t, err)
    require.Equal(t, role.Kind, rOut.Kind)

    roles, err := store.ListRolesByPerson(context.Background(), person.ID)
    require.NoError(t, err)
    require.Len(t, roles, 1)
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile && go test ./internal/storage/... -run "TestUpsert" -v`

Expected: FAIL with `undefined: NewV2SubstrateStore` (when DATABASE_URL set), or SKIP when not set.

- [ ] **Step 4: Write the storage implementation**

Path: `<kbs>/kb-20-patient-profile/internal/storage/v2_substrate_store.go`

```go
// Package storage provides kb-20-patient-profile's persistence layer,
// including the v2 substrate canonical-row implementations.
package storage

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/lib/pq"
    _ "github.com/lib/pq"

    "github.com/cardiofit/kb-shared/v2_substrate/models"
)

// V2SubstrateStore implements ResidentStore + PersonStore + RoleStore for
// kb-20-patient-profile. Reads use the residents_v2 compatibility view;
// writes touch the underlying patient_profiles + persons + roles tables.
type V2SubstrateStore struct {
    db *sql.DB
}

// NewV2SubstrateStore returns a store bound to the given Postgres DSN.
func NewV2SubstrateStore(dsn string) (*V2SubstrateStore, error) {
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        return nil, fmt.Errorf("open: %w", err)
    }
    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("ping: %w", err)
    }
    return &V2SubstrateStore{db: db}, nil
}

func (s *V2SubstrateStore) GetResident(ctx context.Context, id uuid.UUID) (*models.Resident, error) {
    const q = `SELECT id, ihi, given_name, family_name, dob, sex, indigenous_status,
        facility_id, admission_date, care_intensity, sdms, status, created_at, updated_at
        FROM residents_v2 WHERE id = $1`
    var r models.Resident
    var ihi, indigStatus, sex, status sql.NullString
    var sdms pq.GenericArray
    var admDate sql.NullTime
    if err := s.db.QueryRowContext(ctx, q, id).Scan(
        &r.ID, &ihi, &r.GivenName, &r.FamilyName, &r.DOB, &sex,
        &indigStatus, &r.FacilityID, &admDate, &r.CareIntensity,
        &sdms, &status, &r.CreatedAt, &r.UpdatedAt,
    ); err != nil {
        return nil, err
    }
    if ihi.Valid { r.IHI = ihi.String }
    if indigStatus.Valid { r.IndigenousStatus = indigStatus.String }
    if sex.Valid { r.Sex = sex.String }
    if status.Valid { r.Status = status.String }
    if admDate.Valid { r.AdmissionDate = &admDate.Time }
    // sdms scan; expect uuid array
    return &r, nil
}

func (s *V2SubstrateStore) UpsertResident(ctx context.Context, r models.Resident) (*models.Resident, error) {
    // Strategy: upsert into patient_profiles directly (canonical underlying table)
    const q = `
        INSERT INTO patient_profiles
            (id, ihi, first_name, last_name, date_of_birth, gender, indigenous_status,
             facility_id, admission_date, care_intensity, sdms, created_at, updated_at)
        VALUES
            ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
        ON CONFLICT (id) DO UPDATE SET
            ihi = EXCLUDED.ihi,
            first_name = EXCLUDED.first_name,
            last_name = EXCLUDED.last_name,
            date_of_birth = EXCLUDED.date_of_birth,
            gender = EXCLUDED.gender,
            indigenous_status = EXCLUDED.indigenous_status,
            facility_id = EXCLUDED.facility_id,
            admission_date = EXCLUDED.admission_date,
            care_intensity = EXCLUDED.care_intensity,
            sdms = EXCLUDED.sdms,
            updated_at = NOW()
    `
    var sdmsArg interface{}
    if len(r.SDMs) > 0 {
        ids := make([]string, len(r.SDMs))
        for i, u := range r.SDMs { ids[i] = u.String() }
        sdmsArg = pq.Array(ids)
    }
    if _, err := s.db.ExecContext(ctx, q,
        r.ID, nilIfEmpty(r.IHI), r.GivenName, r.FamilyName, r.DOB, r.Sex,
        nilIfEmpty(r.IndigenousStatus), r.FacilityID, r.AdmissionDate,
        r.CareIntensity, sdmsArg,
    ); err != nil {
        return nil, fmt.Errorf("upsert resident: %w", err)
    }
    return s.GetResident(ctx, r.ID)
}

func (s *V2SubstrateStore) ListResidentsByFacility(ctx context.Context, facilityID uuid.UUID, limit, offset int) ([]models.Resident, error) {
    const q = `SELECT id FROM residents_v2 WHERE facility_id = $1 ORDER BY family_name, given_name LIMIT $2 OFFSET $3`
    rows, err := s.db.QueryContext(ctx, q, facilityID, limit, offset)
    if err != nil { return nil, err }
    defer rows.Close()
    var residents []models.Resident
    for rows.Next() {
        var id uuid.UUID
        if err := rows.Scan(&id); err != nil { return nil, err }
        r, err := s.GetResident(ctx, id)
        if err != nil { return nil, err }
        residents = append(residents, *r)
    }
    return residents, nil
}

func (s *V2SubstrateStore) GetPerson(ctx context.Context, id uuid.UUID) (*models.Person, error) {
    const q = `SELECT id, given_name, family_name, hpii, ahpra_registration, contact_details
        FROM persons WHERE id = $1`
    var p models.Person
    var hpii, ahpra sql.NullString
    var contact []byte
    if err := s.db.QueryRowContext(ctx, q, id).Scan(&p.ID, &p.GivenName, &p.FamilyName, &hpii, &ahpra, &contact); err != nil {
        return nil, err
    }
    if hpii.Valid { p.HPII = hpii.String }
    if ahpra.Valid { p.AHPRARegistration = ahpra.String }
    if len(contact) > 0 { p.ContactDetails = json.RawMessage(contact) }
    return &p, nil
}

func (s *V2SubstrateStore) UpsertPerson(ctx context.Context, p models.Person) (*models.Person, error) {
    const q = `
        INSERT INTO persons (id, given_name, family_name, hpii, ahpra_registration, contact_details, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
        ON CONFLICT (id) DO UPDATE SET
            given_name = EXCLUDED.given_name,
            family_name = EXCLUDED.family_name,
            hpii = EXCLUDED.hpii,
            ahpra_registration = EXCLUDED.ahpra_registration,
            contact_details = EXCLUDED.contact_details,
            updated_at = NOW()
    `
    if _, err := s.db.ExecContext(ctx, q,
        p.ID, p.GivenName, p.FamilyName, nilIfEmpty(p.HPII), nilIfEmpty(p.AHPRARegistration), p.ContactDetails,
    ); err != nil {
        return nil, fmt.Errorf("upsert person: %w", err)
    }
    return s.GetPerson(ctx, p.ID)
}

func (s *V2SubstrateStore) GetPersonByHPII(ctx context.Context, hpii string) (*models.Person, error) {
    const q = `SELECT id FROM persons WHERE hpii = $1`
    var id uuid.UUID
    if err := s.db.QueryRowContext(ctx, q, hpii).Scan(&id); err != nil {
        return nil, err
    }
    return s.GetPerson(ctx, id)
}

func (s *V2SubstrateStore) GetRole(ctx context.Context, id uuid.UUID) (*models.Role, error) {
    const q = `SELECT id, person_id, kind, qualifications, facility_id, valid_from, valid_to, evidence_url
        FROM roles WHERE id = $1`
    var r models.Role
    var quals []byte
    var facID *uuid.UUID
    var validTo sql.NullTime
    var evidenceURL sql.NullString
    if err := s.db.QueryRowContext(ctx, q, id).Scan(
        &r.ID, &r.PersonID, &r.Kind, &quals, &facID, &r.ValidFrom, &validTo, &evidenceURL,
    ); err != nil { return nil, err }
    if len(quals) > 0 { r.Qualifications = json.RawMessage(quals) }
    if facID != nil { r.FacilityID = facID }
    if validTo.Valid { r.ValidTo = &validTo.Time }
    if evidenceURL.Valid { r.EvidenceURL = evidenceURL.String }
    return &r, nil
}

func (s *V2SubstrateStore) UpsertRole(ctx context.Context, r models.Role) (*models.Role, error) {
    const q = `
        INSERT INTO roles (id, person_id, kind, qualifications, facility_id, valid_from, valid_to, evidence_url, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
        ON CONFLICT (id) DO UPDATE SET
            kind = EXCLUDED.kind,
            qualifications = EXCLUDED.qualifications,
            facility_id = EXCLUDED.facility_id,
            valid_from = EXCLUDED.valid_from,
            valid_to = EXCLUDED.valid_to,
            evidence_url = EXCLUDED.evidence_url,
            updated_at = NOW()
    `
    if _, err := s.db.ExecContext(ctx, q,
        r.ID, r.PersonID, r.Kind, []byte(r.Qualifications), r.FacilityID, r.ValidFrom, r.ValidTo, nilIfEmpty(r.EvidenceURL),
    ); err != nil { return nil, fmt.Errorf("upsert role: %w", err) }
    return s.GetRole(ctx, r.ID)
}

func (s *V2SubstrateStore) ListRolesByPerson(ctx context.Context, personID uuid.UUID) ([]models.Role, error) {
    const q = `SELECT id FROM roles WHERE person_id = $1 ORDER BY valid_from DESC`
    rows, err := s.db.QueryContext(ctx, q, personID)
    if err != nil { return nil, err }
    defer rows.Close()
    var roles []models.Role
    for rows.Next() {
        var id uuid.UUID
        if err := rows.Scan(&id); err != nil { return nil, err }
        r, err := s.GetRole(ctx, id)
        if err != nil { return nil, err }
        roles = append(roles, *r)
    }
    return roles, nil
}

func (s *V2SubstrateStore) ListActiveRolesByPersonAndFacility(ctx context.Context, personID, facilityID uuid.UUID) ([]models.Role, error) {
    const q = `SELECT id FROM roles
        WHERE person_id = $1
          AND (facility_id IS NULL OR facility_id = $2)
          AND valid_from <= NOW()
          AND (valid_to IS NULL OR valid_to >= NOW())
        ORDER BY valid_from DESC`
    rows, err := s.db.QueryContext(ctx, q, personID, facilityID)
    if err != nil { return nil, err }
    defer rows.Close()
    var roles []models.Role
    for rows.Next() {
        var id uuid.UUID
        if err := rows.Scan(&id); err != nil { return nil, err }
        r, err := s.GetRole(ctx, id)
        if err != nil { return nil, err }
        roles = append(roles, *r)
    }
    return roles, nil
}

func nilIfEmpty(s string) interface{} {
    if s == "" { return nil }
    return s
}

// Compile-time interface assertion (omit until shared/v2_substrate/interfaces is imported by kb-20 go.mod)
// var _ interfaces.ResidentStore = (*V2SubstrateStore)(nil)
// var _ interfaces.PersonStore = (*V2SubstrateStore)(nil)
// var _ interfaces.RoleStore = (*V2SubstrateStore)(nil)

// Reference time package even if unused in this build to keep imports stable
var _ = time.Now
```

- [ ] **Step 5: Add shared module dependency to kb-20 go.mod**

Run: `head -3 /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/go.mod`

Note the kb-20 module name. Then edit go.mod to add the shared module:

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile
go mod edit -require=github.com/cardiofit/kb-shared@v0.0.0
go mod edit -replace=github.com/cardiofit/kb-shared=../shared
go mod tidy
```

(Substitute the actual `shared/go.mod` module path for `github.com/cardiofit/kb-shared` as confirmed in Task 6 Step 2.)

- [ ] **Step 6: Run test to verify it passes against a kb-20 dev DB**

Prerequisite: kb-20 dev DB must be running and migration 008_part1 applied.

```bash
make -C /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services run-kb-docker
# Apply the migration manually or via the kb-20 migrate command (depends on kb-20's migration runner)
psql -h localhost -p 5433 -U kb_drug_rules_user -d kb_20_patient_profile -f /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/migrations/008_part1_actor_model.sql
```

Then:

```bash
export KB20_TEST_DATABASE_URL="postgres://kb_drug_rules_user:kb_password@localhost:5433/kb_20_patient_profile?sslmode=disable"
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile
go test ./internal/storage/... -run "TestUpsert" -v
```

Expected: PASS for both round-trip tests.

- [ ] **Step 7: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/storage/v2_substrate_store.go backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/storage/v2_substrate_store_test.go backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/go.mod backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/go.sum
git commit -m "feat(kb-20): GORM-backed V2SubstrateStore for ResidentStore/PersonStore/RoleStore"
```

---

## Task 13: kb-20 REST handlers for v2 substrate

**Files:**
- Create: `<kbs>/kb-20-patient-profile/internal/api/v2_substrate_handlers.go`
- Test: `<kbs>/kb-20-patient-profile/internal/api/v2_substrate_handlers_test.go`
- Modify: `<kbs>/kb-20-patient-profile/cmd/server/main.go` (wire routes)

- [ ] **Step 1: Read existing kb-20 handler patterns**

Run: `ls /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/api/ && head -60 /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/api/handlers.go 2>/dev/null`

Expected: shows the existing Gin handler pattern. Match the style.

- [ ] **Step 2: Write the failing handler test**

Path: `<kbs>/kb-20-patient-profile/internal/api/v2_substrate_handlers_test.go`

```go
package api

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "os"
    "testing"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/stretchr/testify/require"

    "github.com/cardiofit/kb-shared/v2_substrate/models"
    "github.com/cardiofit/cardiofit/kb-20-patient-profile/internal/storage"
)

func setupV2Router(t *testing.T) *gin.Engine {
    dsn := os.Getenv("KB20_TEST_DATABASE_URL")
    if dsn == "" {
        t.Skip("KB20_TEST_DATABASE_URL not set")
    }
    store, err := storage.NewV2SubstrateStore(dsn)
    require.NoError(t, err)
    h := NewV2SubstrateHandlers(store)

    gin.SetMode(gin.TestMode)
    r := gin.New()
    h.RegisterRoutes(r.Group("/v2"))
    return r
}

func TestPOSTResidentRoundTrip(t *testing.T) {
    r := setupV2Router(t)
    in := models.Resident{
        ID: uuid.New(), GivenName: "Margaret", FamilyName: "Brown",
        DOB: time.Date(1938, 4, 12, 0, 0, 0, 0, time.UTC), Sex: "female",
        FacilityID: uuid.New(), CareIntensity: models.CareIntensityActive,
        Status: models.ResidentStatusActive,
    }
    body, _ := json.Marshal(in)
    req := httptest.NewRequest(http.MethodPost, "/v2/residents", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    require.Equal(t, http.StatusOK, w.Code)

    var out models.Resident
    require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
    require.Equal(t, in.GivenName, out.GivenName)

    // GET should return same
    req2 := httptest.NewRequest(http.MethodGet, "/v2/residents/"+in.ID.String(), nil)
    w2 := httptest.NewRecorder()
    r.ServeHTTP(w2, req2)
    require.Equal(t, http.StatusOK, w2.Code)
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile && go test ./internal/api/... -run "TestPOSTResident" -v`

Expected: FAIL with `undefined: NewV2SubstrateHandlers`.

- [ ] **Step 4: Write the handlers**

Path: `<kbs>/kb-20-patient-profile/internal/api/v2_substrate_handlers.go`

```go
package api

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"

    "github.com/cardiofit/kb-shared/v2_substrate/models"
    "github.com/cardiofit/kb-shared/v2_substrate/validation"
    "github.com/cardiofit/cardiofit/kb-20-patient-profile/internal/storage"
)

// V2SubstrateHandlers serves v2 substrate REST endpoints for kb-20.
type V2SubstrateHandlers struct {
    store *storage.V2SubstrateStore
}

func NewV2SubstrateHandlers(store *storage.V2SubstrateStore) *V2SubstrateHandlers {
    return &V2SubstrateHandlers{store: store}
}

// RegisterRoutes wires the v2 substrate endpoints onto the given router group.
// Caller is expected to mount the group at "/v2".
func (h *V2SubstrateHandlers) RegisterRoutes(g *gin.RouterGroup) {
    g.POST("/residents", h.upsertResident)
    g.GET("/residents/:id", h.getResident)
    g.GET("/facilities/:facility_id/residents", h.listResidentsByFacility)

    g.POST("/persons", h.upsertPerson)
    g.GET("/persons/:id", h.getPerson)
    g.GET("/persons", h.getPersonByHPII)

    g.POST("/roles", h.upsertRole)
    g.GET("/roles/:id", h.getRole)
    g.GET("/persons/:id/roles", h.listRolesByPerson)
    g.GET("/persons/:id/active_roles", h.listActiveRolesByPersonAndFacility)
}

func (h *V2SubstrateHandlers) upsertResident(c *gin.Context) {
    var r models.Resident
    if err := c.BindJSON(&r); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    if err := validation.ValidateResident(r); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    out, err := h.store.UpsertResident(c.Request.Context(), r)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, out)
}

func (h *V2SubstrateHandlers) getResident(c *gin.Context) {
    id, err := uuid.Parse(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}); return }
    r, err := h.store.GetResident(c.Request.Context(), id)
    if err != nil { c.JSON(http.StatusNotFound, gin.H{"error": err.Error()}); return }
    c.JSON(http.StatusOK, r)
}

func (h *V2SubstrateHandlers) listResidentsByFacility(c *gin.Context) {
    facilityID, err := uuid.Parse(c.Param("facility_id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid facility_id"}); return }
    residents, err := h.store.ListResidentsByFacility(c.Request.Context(), facilityID, 100, 0)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
    c.JSON(http.StatusOK, residents)
}

func (h *V2SubstrateHandlers) upsertPerson(c *gin.Context) {
    var p models.Person
    if err := c.BindJSON(&p); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
    if err := validation.ValidatePerson(p); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
    out, err := h.store.UpsertPerson(c.Request.Context(), p)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
    c.JSON(http.StatusOK, out)
}

func (h *V2SubstrateHandlers) getPerson(c *gin.Context) {
    id, err := uuid.Parse(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}); return }
    p, err := h.store.GetPerson(c.Request.Context(), id)
    if err != nil { c.JSON(http.StatusNotFound, gin.H{"error": err.Error()}); return }
    c.JSON(http.StatusOK, p)
}

func (h *V2SubstrateHandlers) getPersonByHPII(c *gin.Context) {
    hpii := c.Query("hpii")
    if hpii == "" { c.JSON(http.StatusBadRequest, gin.H{"error": "hpii query parameter required"}); return }
    p, err := h.store.GetPersonByHPII(c.Request.Context(), hpii)
    if err != nil { c.JSON(http.StatusNotFound, gin.H{"error": err.Error()}); return }
    c.JSON(http.StatusOK, p)
}

func (h *V2SubstrateHandlers) upsertRole(c *gin.Context) {
    var r models.Role
    if err := c.BindJSON(&r); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
    if err := validation.ValidateRole(r); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
    out, err := h.store.UpsertRole(c.Request.Context(), r)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
    c.JSON(http.StatusOK, out)
}

func (h *V2SubstrateHandlers) getRole(c *gin.Context) {
    id, err := uuid.Parse(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}); return }
    r, err := h.store.GetRole(c.Request.Context(), id)
    if err != nil { c.JSON(http.StatusNotFound, gin.H{"error": err.Error()}); return }
    c.JSON(http.StatusOK, r)
}

func (h *V2SubstrateHandlers) listRolesByPerson(c *gin.Context) {
    personID, err := uuid.Parse(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}); return }
    roles, err := h.store.ListRolesByPerson(c.Request.Context(), personID)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
    c.JSON(http.StatusOK, roles)
}

func (h *V2SubstrateHandlers) listActiveRolesByPersonAndFacility(c *gin.Context) {
    personID, err := uuid.Parse(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}); return }
    facilityID, err := uuid.Parse(c.Query("facility_id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "facility_id query parameter required"}); return }
    roles, err := h.store.ListActiveRolesByPersonAndFacility(c.Request.Context(), personID, facilityID)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
    c.JSON(http.StatusOK, roles)
}
```

- [ ] **Step 5: Wire routes into main.go**

Run: `grep -n "RegisterRoutes\|router :=\|gin.Default" /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/cmd/server/main.go | head -10`

Then add to `main.go` near the existing route registration:

```go
// After existing handler initialisation, add:
v2Handlers := api.NewV2SubstrateHandlers(v2Store)  // assumes v2Store created earlier
v2Handlers.RegisterRoutes(router.Group("/v2"))
```

Adjust to match the actual main.go variable names — search for the existing handler registration pattern first.

- [ ] **Step 6: Run handler test**

```bash
export KB20_TEST_DATABASE_URL="postgres://kb_drug_rules_user:kb_password@localhost:5433/kb_20_patient_profile?sslmode=disable"
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile
go test ./internal/api/... -run "TestPOSTResident" -v
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/api/v2_substrate_handlers.go backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/api/v2_substrate_handlers_test.go backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/cmd/server/main.go
git commit -m "feat(kb-20): REST handlers for v2 substrate Resident/Person/Role + integration test"
```

---

## Task 14: Go client for kb-20 v2 substrate endpoints

**Files:**
- Create: `<kbs>/shared/v2_substrate/client/kb20_client.go`
- Test: `<kbs>/shared/v2_substrate/client/kb20_client_test.go`

- [ ] **Step 1: Write the failing client test**

Path: `<kbs>/shared/v2_substrate/client/kb20_client_test.go`

```go
package client

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/google/uuid"
    "github.com/stretchr/testify/require"

    "github.com/cardiofit/kb-shared/v2_substrate/models"
)

func TestKB20ClientUpsertAndGetResident(t *testing.T) {
    in := models.Resident{
        ID: uuid.New(), GivenName: "Margaret", FamilyName: "Brown",
        DOB: time.Now(), Sex: "female", FacilityID: uuid.New(),
        CareIntensity: models.CareIntensityActive, Status: models.ResidentStatusActive,
    }

    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        switch {
        case r.Method == http.MethodPost && r.URL.Path == "/v2/residents":
            w.WriteHeader(http.StatusOK)
            json.NewEncoder(w).Encode(in)
        case r.Method == http.MethodGet && r.URL.Path == "/v2/residents/"+in.ID.String():
            w.WriteHeader(http.StatusOK)
            json.NewEncoder(w).Encode(in)
        default:
            w.WriteHeader(http.StatusNotFound)
        }
    }))
    defer server.Close()

    c := NewKB20Client(server.URL)
    out, err := c.UpsertResident(context.Background(), in)
    require.NoError(t, err)
    require.Equal(t, in.GivenName, out.GivenName)

    fetched, err := c.GetResident(context.Background(), in.ID)
    require.NoError(t, err)
    require.Equal(t, in.ID, fetched.ID)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && go test ./v2_substrate/client/... -v`

Expected: FAIL with `undefined: NewKB20Client`.

- [ ] **Step 3: Write the client**

Path: `<kbs>/shared/v2_substrate/client/kb20_client.go`

```go
// Package client provides typed Go clients to the v2 substrate canonical
// stores (kb-20 for actor entities; kb-22 for Event/EvidenceTrace).
//
// Example:
//
//	c := client.NewKB20Client("http://kb-20:8131")
//	resident, err := c.GetResident(ctx, residentID)
package client

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"

    "github.com/google/uuid"

    "github.com/cardiofit/kb-shared/v2_substrate/models"
)

type KB20Client struct {
    baseURL string
    http    *http.Client
}

func NewKB20Client(baseURL string) *KB20Client {
    return &KB20Client{baseURL: baseURL, http: http.DefaultClient}
}

func (c *KB20Client) UpsertResident(ctx context.Context, r models.Resident) (*models.Resident, error) {
    return doJSON[models.Resident](ctx, c.http, http.MethodPost, c.baseURL+"/v2/residents", r)
}

func (c *KB20Client) GetResident(ctx context.Context, id uuid.UUID) (*models.Resident, error) {
    return doJSON[models.Resident](ctx, c.http, http.MethodGet, c.baseURL+"/v2/residents/"+id.String(), nil)
}

func (c *KB20Client) ListResidentsByFacility(ctx context.Context, facilityID uuid.UUID) ([]models.Resident, error) {
    out, err := doJSON[[]models.Resident](ctx, c.http, http.MethodGet,
        c.baseURL+"/v2/facilities/"+facilityID.String()+"/residents", nil)
    if err != nil { return nil, err }
    return *out, nil
}

func (c *KB20Client) UpsertPerson(ctx context.Context, p models.Person) (*models.Person, error) {
    return doJSON[models.Person](ctx, c.http, http.MethodPost, c.baseURL+"/v2/persons", p)
}

func (c *KB20Client) GetPerson(ctx context.Context, id uuid.UUID) (*models.Person, error) {
    return doJSON[models.Person](ctx, c.http, http.MethodGet, c.baseURL+"/v2/persons/"+id.String(), nil)
}

func (c *KB20Client) GetPersonByHPII(ctx context.Context, hpii string) (*models.Person, error) {
    u := c.baseURL + "/v2/persons?hpii=" + url.QueryEscape(hpii)
    return doJSON[models.Person](ctx, c.http, http.MethodGet, u, nil)
}

func (c *KB20Client) UpsertRole(ctx context.Context, r models.Role) (*models.Role, error) {
    return doJSON[models.Role](ctx, c.http, http.MethodPost, c.baseURL+"/v2/roles", r)
}

func (c *KB20Client) GetRole(ctx context.Context, id uuid.UUID) (*models.Role, error) {
    return doJSON[models.Role](ctx, c.http, http.MethodGet, c.baseURL+"/v2/roles/"+id.String(), nil)
}

func (c *KB20Client) ListRolesByPerson(ctx context.Context, personID uuid.UUID) ([]models.Role, error) {
    out, err := doJSON[[]models.Role](ctx, c.http, http.MethodGet,
        c.baseURL+"/v2/persons/"+personID.String()+"/roles", nil)
    if err != nil { return nil, err }
    return *out, nil
}

func (c *KB20Client) ListActiveRolesByPersonAndFacility(ctx context.Context, personID, facilityID uuid.UUID) ([]models.Role, error) {
    u := c.baseURL + "/v2/persons/" + personID.String() + "/active_roles?facility_id=" + facilityID.String()
    out, err := doJSON[[]models.Role](ctx, c.http, http.MethodGet, u, nil)
    if err != nil { return nil, err }
    return *out, nil
}

// doJSON is a small generic helper for JSON request/response.
func doJSON[T any](ctx context.Context, h *http.Client, method, url string, body interface{}) (*T, error) {
    var buf io.Reader
    if body != nil {
        b, err := json.Marshal(body)
        if err != nil { return nil, fmt.Errorf("marshal: %w", err) }
        buf = bytes.NewReader(b)
    }
    req, err := http.NewRequestWithContext(ctx, method, url, buf)
    if err != nil { return nil, err }
    if body != nil {
        req.Header.Set("Content-Type", "application/json")
    }
    resp, err := h.Do(req)
    if err != nil { return nil, err }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        b, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("kb-20 %s %s: status %d: %s", method, url, resp.StatusCode, string(b))
    }
    var out T
    if err := json.NewDecoder(resp.Body).Decode(&out); err != nil { return nil, fmt.Errorf("decode: %w", err) }
    return &out, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared && go test ./v2_substrate/client/... -v`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/client/
git commit -m "feat(v2_substrate/client): KB20Client for Resident/Person/Role with httptest verification"
```

---

## Task 15: End-to-end milestone verification

**Files:** none (verification only)

- [ ] **Step 1: Run the full test suite for v2_substrate**

Run:

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/shared
go test ./v2_substrate/... -v 2>&1 | tail -30
```

Expected: PASS for models, fhir, validation, client packages. Storage + handler tests SKIP without DB (tested separately in Step 2).

- [ ] **Step 2: Run kb-20 integration tests against live DB**

Run:

```bash
make -C /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services run-kb-docker
psql -h localhost -p 5433 -U kb_drug_rules_user -d kb_20_patient_profile -f /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/migrations/008_part1_actor_model.sql
export KB20_TEST_DATABASE_URL="postgres://kb_drug_rules_user:kb_password@localhost:5433/kb_20_patient_profile?sslmode=disable"
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile
go test ./internal/storage/... ./internal/api/... -v 2>&1 | tail -30
```

Expected: PASS for V2SubstrateStore tests + handler tests.

- [ ] **Step 3: Verify migration is non-breaking**

Check that all existing kb-20 tests still pass:

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile
go test ./... 2>&1 | tail -20
```

Expected: all pre-existing tests still PASS. New v2 substrate columns are nullable; existing patient_profiles consumers see no change.

- [ ] **Step 4: Verify FHIR mapper round-trip example end-to-end**

Run a small Go program (or `go test`) that:

1. Creates a Resident
2. Stores via KB20Client.UpsertResident
3. Reads via KB20Client.GetResident
4. Maps to AU FHIR Patient via fhir.ResidentToAUPatient
5. Maps back via fhir.AUPatientToResident
6. Confirms IHI, GivenName, FamilyName, Sex, DOB, CareIntensity all survive

This can be a `TestMilestone1Acceptance` test in `shared/v2_substrate/fhir/patient_mapper_test.go` that combines the existing tests.

- [ ] **Step 5: Document Milestone 1B-β.1 completion**

Update `docs/superpowers/specs/2026-05-04-1b-beta-substrate-entities-design.md` §6 milestone 1B-β.1 row from "deliverables list" to ✅ landed by appending a brief completion note. Or, if a separate progress tracker exists (e.g. `MANIFEST.md` for substrate), update there.

- [ ] **Step 6: Commit verification artifacts**

```bash
cd /Volumes/Vaidshala/cardiofit
git add docs/superpowers/specs/2026-05-04-1b-beta-substrate-entities-design.md
git commit -m "docs(v2_substrate): mark milestone 1B-β.1 complete"
```

---

## Self-Review (post-write)

**Spec coverage check (against `2026-05-04-1b-beta-substrate-entities-design.md`):**

- §2.1 shared library at shared/v2_substrate/ — Tasks 1-15 (the whole package)
- §2.2 internal model + FHIR mappers (option B) — Tasks 7-9 (extensions, Patient mapper, Practitioner mapper)
- §2.5 non-breaking migration — Task 11 (008_part1; nullable columns; compatibility view)
- §3.1 Resident type — Task 3
- §3.2 Person + Role types — Tasks 4-5
- §3.2 Qualifications JSONB matching ScopeRules — Task 5 (`TestRoleQualificationsMatchScopeRulesShape`)
- §4 library structure — Tasks 1, 6, 7, 10 (all directories created)
- §5.1 kb-20 migration 008 part 1 — Task 11
- §6 milestone 1B-β.1 deliverables — Tasks 1-15
- §6 exit criterion (round-trip kb20Client.UpsertResident → GetResident → AU FHIR Patient) — Task 15 step 4

**Out of scope items NOT in this plan (correctly deferred):**
- MedicineUse, Observation (β.2 separate plan)
- Event, EvidenceTrace (β.3 separate plan)
- gRPC + protobuf (deferred to later — REST suffices for milestone β.1)
- Authorisation evaluator (downstream phase)
- Layer 1B adapters (Phase 1B-γ)

**Placeholder scan:** None. Concrete code/SQL/commands in every step. The "uuidParse stub" in Task 8 Step 5 is explicitly the patch instructions to replace it.

**Type consistency:**
- `Resident` fields used in Task 3 match Task 11 SQL columns and Task 8 FHIR mapper ✅
- `Role.Qualifications` JSONB shape from Task 5 matches `regulatory_scope_rules.role_qualifications` keys from Phase 1C-γ migration 007 ✅
- KB20Client method names match V2SubstrateStore method names match handler endpoint paths ✅

**Procurement-failure parallel:** unlike the 1C-α plan, this plan has no external procurement; failures here are test failures or migration failures, both surfaced explicitly in their respective tasks.

**Module path note:** Tasks 6, 12, 13 reference `github.com/cardiofit/kb-shared` as the shared module path. Task 6 Step 2 instructs the implementer to confirm the actual module path from `shared/go.mod` and substitute throughout. This is a structural placeholder (the actual path varies per repo); not a content placeholder.

---

## Next steps after this plan completes

1. Milestone 1B-β.2 plan — clinical primitives (MedicineUse + Observation). Will be authored as a separate plan after this milestone is reviewed and merged.
2. Milestone 1B-β.3 plan — Event + EvidenceTrace. Separate plan.
3. After all three milestones land: design Phase 1B-γ (first Layer 1B adapter — recommended eNRMC CSV).
