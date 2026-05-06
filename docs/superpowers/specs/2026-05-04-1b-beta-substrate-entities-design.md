# Phase 1B-β — V2 Substrate Entities Design

**Date:** 2026-05-04
**Phase:** 1B-β (substrate entities for v2 reasoning-continuity infrastructure)
**Status:** Design — pending implementation
**Spec it implements:** Vaidshala v2 Revision Mapping MVP-1 (substrate entities + foundation for 5 state machines)
**Companion docs:**
- `kb-6-formulary/Vaidshala_Final_Product_Proposal_v2_Revision_Mapping.md` Parts 3 + 6
- `docs/superpowers/specs/2026-05-04-layer1c-procurement-design.md` (precedent — Source Registry / ScopeRules schema)

---

## 1. Context

The v2 substrate is the foundation the Vaidshala v2 reasoning-continuity infrastructure builds on. Per Revision Mapping Part 3, five interlocking state machines (Authorisation, Recommendation, Monitoring, Clinical state, Consent) all share a common substrate of clinical entities: Resident, Person+Role, MedicineUse with intent+target+stop_criteria, Observation with delta-on-write, Event, EvidenceTrace.

This phase delivers those substrate entities. It does NOT build the state machines, the Authorisation evaluator, or any Layer 1B adapter — those are downstream phases.

### 1.1 Current state

The codebase already has ~70% of substrate primitives distributed across KBs:

| Substrate entity | Existing | Where | Gap |
|---|---|---|---|
| Resident | 🟡 patient_profiles | kb-20 migration 001 | aged-care fields, SDM linkage, care_intensity tag |
| Person + Role | ❌ | nowhere | greenfield |
| MedicineUse | 🟡 medication_states | kb-20 migrations 006/007 | needs intent, target, stop_criteria (the v2-distinguishing fields) |
| Observation | 🟡 lab_entries (kb-20) + ObservationEvent (kb-26) | two places, different shapes | needs delta-on-write + reconciliation |
| Event | 🟡 event_outbox + kb-26 *Event Go types | kb-20 + kb-26 | needs unified canonical Event entity |
| EvidenceTrace | ❌ | nowhere | greenfield (graph model) |

Plus: Neo4j is already in use across kb-25, kb-3, kb-7, clinical-reasoning-service. Graph database for EvidenceTrace is "use existing infrastructure," not "introduce new technology."

### 1.2 Why this is not greenfield

The phase is fundamentally promotion + reconciliation of existing patterns rather than ground-up creation. The implementation strategy reflects this: extend existing tables in place where they exist, add new tables only for what's genuinely new (Person/Role/Event/EvidenceTrace), and provide compatibility views so existing consumers (medication-service, KB-26 baselines) keep working unchanged.

---

## 2. Architectural commitments

### 2.1 Shared library + designated canonical-storage KB per entity (Option C.ii)

Substrate types live in a new shared Go package at `backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/`. This follows the existing pattern of `shared/factstore/`, `shared/governance/`, etc. The `shared/` module is already a Go module that two KBs (kb-5, kb-9) currently import; the new package extends that pattern.

**Storage:** one designated KB per entity owns the canonical row. Other KBs import the shared types and call gRPC/REST endpoints on the canonical KB to read.

| Entity | Canonical KB | Rationale |
|---|---|---|
| Resident | kb-20-patient-profile | Already hosts patient_profiles |
| Person | kb-20-patient-profile | Same bounded context as Resident (actors paired) |
| Role | kb-20-patient-profile | Tied to Person (1:N) |
| MedicineUse | kb-20-patient-profile | Already hosts medication_states |
| Observation | kb-20-patient-profile | Already hosts lab_entries; consolidates kb-26 ObservationEvent |
| Event | kb-22-hpi-engine | kb-22 already has reasoning_chain + session_provenance |
| EvidenceTrace | kb-22-hpi-engine | Same audit/reasoning bounded context as Event |

This split mirrors the conceptual seam: the first 5 entities are clinical state primitives (kb-20 = state); the last 2 are reasoning + audit primitives (kb-22 = reasoning).

### 2.2 FHIR alignment: internal model + FHIR mappers at boundaries (Option B)

`v2_substrate.Resident` is a clean Go struct designed for Vaidshala's domain — only the fields aged-care-medication-stewardship actually uses. A separate `shared/v2_substrate/fhir/` package converts to/from AU FHIR R4 profiles (HL7 AU Base v6.0.0, MHR FHIR Gateway v5.0).

**Why not FHIR-aligned types?** AU Patient has 25+ fields with loose cardinality (0..*); the existing kb-20.patient_profiles has ~15 fields with tight required-vs-optional. FHIR's verbosity drags nil-checking ceremony into every consumer. The v2-specific fields (MedicineUse.intent/target/stop_criteria, Observation.delta, care_intensity) don't have native FHIR representations and would shoehorn into FHIR extensions where they'd be second-class citizens.

**Why not hybrid (single struct with FHIR JSON tags)?** The struct ends up a compromise — neither clean internal nor strict FHIR. Custom marshallers would be required for FHIR's nesting (`Patient.identifier[].value` flattens awkwardly into Go).

The mapper-at-boundary pattern is what HL7 AU Base IG itself documents — most production AU FHIR consumers run internal canonical models with FHIR adapters, not raw FHIR types throughout.

### 2.3 EvidenceTrace = Neo4j sidecar with relational source-of-truth

Postgres is wrong for a graph that's "queryable bidirectionally" with n-hop traversal — recursive CTEs work but get slow + ugly past 3 hops. Neo4j is already in the codebase (kb-25, kb-3, kb-7, clinical-reasoning-service); using it here is "leverage existing infrastructure," not "introduce new technology."

**Single source of truth is relational.** kb-22 stores `events` table + `evidence_trace_nodes` + `evidence_trace_edges` tables in Postgres. These are transactionally durable. Neo4j is a denormalised projection rebuilt from those relational rows. If Neo4j is lost, it's rebuilt; if Postgres is lost, the audit chain is lost.

This protects transactional integrity (Postgres is the system of record) while giving the EvidenceTrace consumer (the audit query API, future state machines) the graph-traversal performance it needs.

### 2.4 MedicineUse v2 fields as structured JSONB

`intent`, `target`, `stop_criteria` ship as JSONB columns rather than separate columns per sub-field. The shape varies by clinical context — antihypertensive `target` is a BP threshold; deprescribing `target` is a date; antibiotic `target` is course completion. Fixed columns constrain future rule patterns. JSONB with documented schemas in `medicine_use.go` keeps shape evolvable.

### 2.5 Migration strategy: non-breaking promotion

All migrations are additive or extend-with-nullable-columns. No existing kb-20 consumer needs to change behaviour. Compatibility views (`resident_v2_view`, `medicine_use_v2_view`) provide v2-shaped reads for new consumers while existing consumers continue reading raw `patient_profiles` etc.

Existing kb-26 ObservationEvent / MedChangeEvent / CheckinEvent types stay as adapter input shape (Event-like). kb-26 baseline computation will eventually read canonical Observation from kb-20 via gRPC client, but that migration is out of scope here — deferred until at least one Layer 1B adapter writes canonical Observation to kb-20.

---

## 3. Substrate type definitions

Types live in `shared/v2_substrate/models/`. Each file defines one entity. JSONB shapes are documented in struct comments.

### 3.1 Resident (kb-20 canonical)

```go
type Resident struct {
    ID                uuid.UUID  `json:"id"`
    IHI               string     `json:"ihi,omitempty"`            // AU Individual Healthcare Identifier
    GivenName         string     `json:"given_name"`
    FamilyName        string     `json:"family_name"`
    DOB               time.Time  `json:"dob"`
    Sex               string     `json:"sex"`                       // FHIR AdministrativeGender
    IndigenousStatus  string     `json:"indigenous_status,omitempty"` // AU-specific
    FacilityID        uuid.UUID  `json:"facility_id"`
    AdmissionDate     *time.Time `json:"admission_date,omitempty"`
    CareIntensity     string     `json:"care_intensity"`            // palliative|comfort|active|rehabilitation
    SDMs              []uuid.UUID `json:"sdms,omitempty"`           // SubstituteDecisionMaker Person IDs
    Status            string     `json:"status"`                    // active|deceased|transferred|discharged
    CreatedAt         time.Time  `json:"created_at"`
    UpdatedAt         time.Time  `json:"updated_at"`
}
```

### 3.2 Person + Role (kb-20 canonical, greenfield)

```go
type Person struct {
    ID                uuid.UUID       `json:"id"`
    GivenName         string          `json:"given_name"`
    FamilyName        string          `json:"family_name"`
    HPII              string          `json:"hpii,omitempty"`              // AU Healthcare Provider Identifier — Individual
    AHPRARegistration string          `json:"ahpra_registration,omitempty"`
    ContactDetails    json.RawMessage `json:"contact,omitempty"`
}

type Role struct {
    ID             uuid.UUID       `json:"id"`
    PersonID       uuid.UUID       `json:"person_id"`
    Kind           string          `json:"kind"` // RN|EN|NP|DRNP|GP|pharmacist|ACOP|PCW|SDM|family|...
    Qualifications json.RawMessage `json:"qualifications,omitempty"`
    FacilityID     *uuid.UUID      `json:"facility_id,omitempty"` // role scoped to facility (or nil = portable)
    ValidFrom      time.Time       `json:"valid_from"`
    ValidTo        *time.Time      `json:"valid_to,omitempty"`
    EvidenceURL    string          `json:"evidence_url,omitempty"`
}
```

`Role.Qualifications` documented JSONB shape examples:

```json
// EN with notation:
{"notation": true}
// EN without notation + NMBA medication qualification:
{"notation": false, "nmba_medication_qual": true}
// Designated RN Prescriber:
{"endorsement": "designated_rn_prescriber", "valid_from": "2025-09-30", "prescribing_agreement_id": "..."}
// ACOP-credentialed pharmacist:
{"apc_training_complete": true, "valid_from": "2026-07-01", "tier": 1}
```

These map directly to `regulatory_scope_rules.role_qualifications` keys (Phase 1C-γ schema), which is intentional — Authorisation evaluator (future phase) uses the same key shape on both sides.

### 3.3 MedicineUse (kb-20 canonical, v2-distinguishing fields)

```go
type MedicineUse struct {
    ID           uuid.UUID    `json:"id"`
    ResidentID   uuid.UUID    `json:"resident_id"`
    AMTCode      string       `json:"amt_code"`            // AU Medicines Terminology
    DisplayName  string       `json:"display_name"`
    Intent       Intent       `json:"intent"`              // ← v2-distinguishing
    Target       Target       `json:"target"`              // ← v2-distinguishing
    StopCriteria StopCriteria `json:"stop_criteria"`       // ← v2-distinguishing
    Dose         string       `json:"dose"`
    Route        string       `json:"route"`
    Frequency    string       `json:"frequency"`
    PrescriberID uuid.UUID    `json:"prescriber_id"`        // Person.id
    StartedAt    time.Time    `json:"started_at"`
    EndedAt      *time.Time   `json:"ended_at,omitempty"`
    Status       string       `json:"status"`               // active|paused|ceased|completed
}

type Intent struct {
    Category   string `json:"category"`   // therapeutic|preventive|symptomatic|trial|deprescribing
    Indication string `json:"indication"` // free text or SNOMED code
    Notes      string `json:"notes,omitempty"`
}

type Target struct {
    Kind string          `json:"kind"` // BP_threshold|completion_date|symptom_resolution|HbA1c_band|trial_outcome
    Spec json.RawMessage `json:"spec"` // shape varies by Kind; documented in target_schemas.go
}

type StopCriteria struct {
    Triggers   []string        `json:"triggers"` // adverse_event|target_achieved|review_due|patient_request|...
    ReviewDate *time.Time      `json:"review_date,omitempty"`
    Spec       json.RawMessage `json:"spec,omitempty"`
}
```

### 3.4 Observation (kb-20 canonical, with delta-on-write)

```go
type Observation struct {
    ID         uuid.UUID  `json:"id"`
    ResidentID uuid.UUID  `json:"resident_id"`
    LOINCCode  string     `json:"loinc_code,omitempty"`
    SNOMEDCode string     `json:"snomed_code,omitempty"`
    Kind       string     `json:"kind"`            // vital|lab|behavioural|mobility|weight
    Value      float64    `json:"value,omitempty"`
    ValueText  string     `json:"value_text,omitempty"`
    Unit       string     `json:"unit,omitempty"`
    ObservedAt time.Time  `json:"observed_at"`
    SourceID   uuid.UUID  `json:"source_id"`        // FK to clinical_sources
    Delta      *Delta     `json:"delta,omitempty"`  // ← computed on write
}

type Delta struct {
    BaselineValue   float64   `json:"baseline_value"`
    DeviationStdDev float64   `json:"deviation_stddev"`
    DirectionalFlag string    `json:"flag"`             // within_baseline|elevated|severely_elevated|low|severely_low
    ComputedAt      time.Time `json:"computed_at"`
}
```

Delta computation strategy decision (DB trigger vs service-layer) is deferred to implementation phase — both will be measured for performance impact under realistic write rates.

### 3.5 Event (kb-22 canonical, audit substrate)

```go
type Event struct {
    ID                   uuid.UUID       `json:"id"`
    Kind                 string          `json:"kind"`              // medication_administered|observation_recorded|recommendation_drafted|consent_granted|...
    SubjectID            uuid.UUID       `json:"subject_id"`        // typically Resident.id
    ActorID              *uuid.UUID      `json:"actor_id,omitempty"` // Person.id who took action
    ActorRoleID          *uuid.UUID      `json:"actor_role_id,omitempty"` // Role.id under which actor was acting
    AuthorisationVerdict json.RawMessage `json:"authorisation_verdict,omitempty"` // ScopeRules result at action time
    Inputs               []uuid.UUID     `json:"inputs,omitempty"`  // upstream Event/Observation/MedicineUse IDs
    Outputs              []uuid.UUID     `json:"outputs,omitempty"` // downstream IDs
    Payload              json.RawMessage `json:"payload"`
    OccurredAt           time.Time       `json:"occurred_at"`
}
```

`AuthorisationVerdict` is captured at action time so the audit trail is self-contained. Even if ScopeRules change later, the original verdict survives.

### 3.6 EvidenceTrace (kb-22 + Neo4j projection)

```go
type ETNode struct {
    ID        uuid.UUID `json:"id"`
    Kind      string    `json:"kind"`     // event|recommendation|observation|rule|source|consent|authorisation
    EntityID  uuid.UUID `json:"entity_id"` // FK back to the canonical entity
    Summary   string    `json:"summary"`
    CreatedAt time.Time `json:"created_at"`
}

type ETEdge struct {
    ID         uuid.UUID `json:"id"`
    SourceID   uuid.UUID `json:"source_id"`
    TargetID   uuid.UUID `json:"target_id"`
    Relation   string    `json:"relation"` // caused_by|justified_by|supersedes|monitors|references
    Confidence float64   `json:"confidence,omitempty"`
    CreatedAt  time.Time `json:"created_at"`
}
```

The Neo4j projection translates each ETNode to a labeled node and each ETEdge to a typed relationship. The graph is queryable in Cypher: `MATCH (r:Recommendation)-[:JUSTIFIED_BY*1..3]->(s:Source) WHERE r.id = $rid RETURN s` for forward traversal; `MATCH (e:Event)<-[:CAUSED_BY*1..3]-(o:Observation) WHERE e.id = $eid RETURN o` for reverse.

---

## 4. Library structure

```
backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/
├── go.mod  (or shared module continues)
├── README.md
├── models/
│   ├── resident.go
│   ├── person.go
│   ├── role.go
│   ├── medicine_use.go
│   ├── observation.go
│   ├── event.go
│   ├── evidence_trace.go
│   └── enums.go               # CareIntensity, ObservationKind, EventKind, RoleKind constants
├── fhir/
│   ├── patient_mapper.go      # Resident ↔ AU Patient
│   ├── practitioner_mapper.go # Person + Role ↔ AU Practitioner + PractitionerRole
│   ├── medication_mapper.go   # MedicineUse ↔ AU MedicationRequest
│   ├── observation_mapper.go  # Observation ↔ AU Observation
│   └── extensions.go          # AU FHIR extension URIs
├── client/
│   ├── kb20_client.go         # ResidentClient, PersonClient, RoleClient, MedicineUseClient, ObservationClient
│   ├── kb22_client.go         # EventClient, EvidenceTraceClient
│   └── interfaces.go
├── interfaces/
│   ├── storage.go             # ResidentStore, PersonStore, ... interface contracts
│   └── transport.go           # gRPC and REST surface contracts
├── validation/
│   ├── resident_validator.go
│   ├── medicine_use_validator.go
│   └── ...
└── proto/
    ├── substrate.proto        # gRPC service definitions
    └── substrate.pb.go        # generated
```

---

## 5. Migration strategy

### 5.1 kb-20 migration 008 — non-breaking substrate promotion

Three sub-migrations matching the three sub-phases:

**008_part1_actor_model.sql:**
- CREATE TABLE persons (greenfield)
- CREATE TABLE roles (greenfield, FK person_id)
- ALTER TABLE patient_profiles ADD COLUMN ihi, care_intensity, sdms (UUID[]), facility_id (all nullable)
- CREATE VIEW residents_v2 (mapping patient_profiles + extensions to Resident shape)

**008_part2_clinical_primitives.sql:**
- ALTER TABLE medication_states ADD COLUMN intent JSONB, target JSONB, stop_criteria JSONB (all nullable)
- CREATE TABLE observations (greenfield, for non-lab observations: vitals, behavioural)
- ALTER TABLE lab_entries ADD COLUMN delta JSONB (nullable)
- CREATE VIEW medicine_uses_v2, observations_v2 (unified read shape)

### 5.2 kb-22 migration 008 — additive

**008_events_and_evidence_trace.sql:**
- CREATE TABLE events (canonical Event)
- CREATE TABLE evidence_trace_nodes
- CREATE TABLE evidence_trace_edges
- CREATE INDEX idx_events_subject, idx_events_occurred_at, idx_etedge_source/target

### 5.3 Neo4j projection bootstrap

Separate script `kb-22-hpi-engine/scripts/bootstrap_evidence_trace_neo4j.py`:
- Reads `events` + `evidence_trace_*` tables from Postgres
- Creates Neo4j nodes labeled by ETNode.kind
- Creates Neo4j relationships typed by ETEdge.relation
- Idempotent — safe to re-run as projection from current Postgres state

Future Event writes into Postgres trigger an outbox event consumed by a small projector that mirrors into Neo4j. (Outbox pattern infrastructure already exists in kb-20: `event_outbox` + `kafka_outbox` tables.)

---

## 6. Sub-phase milestones

This design ships as **one project with three sequential milestones** for review checkpoints. Total ~4 weeks per the v2 Revision Mapping.

### Milestone 1B-β.1 — Actor model + Resident promotion (~1.5 weeks)

**Deliverables:**
- shared/v2_substrate/models/{resident,person,role,enums}.go
- shared/v2_substrate/fhir/{patient_mapper,practitioner_mapper}.go
- shared/v2_substrate/client/kb20_client.go (Resident/Person/Role methods)
- shared/v2_substrate/interfaces/storage.go (Resident/Person/Role contracts)
- shared/v2_substrate/validation/{resident,person,role}_validator.go
- kb-20 migration 008_part1_actor_model.sql
- kb-20 internal/api/v2_handlers.go — Resident/Person/Role REST + gRPC endpoints
- Tests: type unit tests, FHIR mapper round-trip tests, kb-20 endpoint integration tests

**Exit criterion:** A test client can `kb20Client.UpsertResident(R)` then `kb20Client.GetResident(R.ID)` and round-trip through AU FHIR Patient mapper without data loss.

#### 1B-β.1 Completion (2026-05-04)

✅ **Milestone complete.** Substrate Resident + Person + Role types delivered.

**Test counts:**
- shared/v2_substrate/models: 10 tests pass
- shared/v2_substrate/validation: 7 tests pass
- shared/v2_substrate/fhir: 13 tests pass (incl. _RejectsInvalid + _WrongResourceType + _WireFormat + _DropsInvalidJSONQualifications)
- shared/v2_substrate/client: 2 tests pass
- kb-20 storage: 2 tests SKIP without KB20_TEST_DATABASE_URL (integration only)
- kb-20 api: 28 tests pass, 1 SKIP (integration)

**Carry-overs applied in Cluster 5:** C-1 ingress validation (Patient + Practitioner + PractitionerRole), C-2 json.Valid guard on Qualifications unwrap, C-3 lossy active-flag note (TODO removed, intentional design documented), C-4 negative tests for wrong resourceType (3x), C-5 wire-format assertion test, C-6 v2 routes wired into kb-20 main.go at /v2 via `db.DB.DB() → NewV2SubstrateStoreWithDB`, C-7 explicit HTTP 400 for invalid/out-of-range limit and offset.

**Open follow-ups for 1B-β.2:** None.

### Milestone 1B-β.2 — Clinical primitives (~1.5 weeks)

**Deliverables:**
- shared/v2_substrate/models/{medicine_use,observation}.go
- Documented JSONB schemas in target_schemas.go, stop_criteria_schemas.go
- shared/v2_substrate/fhir/{medication_mapper,observation_mapper}.go
- shared/v2_substrate/client/kb20_client.go (MedicineUse/Observation methods)
- kb-20 migration 008_part2_clinical_primitives.sql
- kb-20 internal/api/v2_handlers.go — MedicineUse/Observation endpoints
- Delta-on-write strategy chosen + implemented (trigger or service layer per perf measurement)
- Tests: JSONB validator tests, FHIR mapper round-trip tests, delta computation correctness tests

**Exit criterion:** Writing an Observation row produces a populated `delta` JSONB consistent with KB-26 baseline computation (or with a direct reference run if KB-26 is not yet integrated).

### Milestone 1B-β.3 — Event + EvidenceTrace (~1 week)

**Deliverables:**
- shared/v2_substrate/models/{event,evidence_trace}.go
- shared/v2_substrate/client/kb22_client.go (Event/EvidenceTrace methods)
- kb-22 migration 008_events_and_evidence_trace.sql
- kb-22 internal/api/v2_handlers.go — Event + EvidenceTrace endpoints
- Neo4j projection bootstrap script + outbox-driven incremental projector
- Tests: Event write integration tests, ET node/edge creation tests, Neo4j projection consistency tests

**Exit criterion:** Writing an Event with Inputs[]/Outputs[] populates corresponding ETNode + ETEdge rows in Postgres AND mirrors them into Neo4j; bidirectional Cypher query returns expected traversal.

---

## 7. Out of scope (explicitly deferred)

These are NOT in this phase to keep scope tight:

- **Authorisation evaluator.** Separate phase (~4 weeks per v2 Revision Mapping). Needs Person/Role + ScopeRules to read from. ScopeRules already exist in kb-22 from Phase 1C-γ.
- **State machines.** Recommendation lifecycle, Monitoring lifecycle, Consent state machine — separate phases.
- **Migrating kb-26 to consume canonical Observation.** Existing kb-26 ObservationEvent/CheckinEvent/MedChangeEvent stay as adapter input shape. kb-26 baseline computation reads canonical Observation from kb-20 in a future phase, after at least one Layer 1B adapter writes to kb-20.
- **Layer 1B adapters.** Phase 1B-γ. eNRMC CSV adapter is the recommended first adapter once substrate lands.
- **EvidenceTrace UI / query API.** This phase delivers the storage + projection. Query API is for Phase 1C-δ (Authorisation evaluator) and beyond.

---

## 8. Risks and mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| kb-20 schema migration breaks existing medication-service consumers | Medium | High | All migrations additive or extend-with-nullable. Compatibility views (`residents_v2`, `medicine_uses_v2`) for new consumers. Existing consumers read raw tables unchanged. |
| Neo4j projection consistency diverges from Postgres source-of-truth | Medium | Medium | Projection script is idempotent — re-runnable from Postgres state. Outbox-driven incremental projector reuses existing kb-20 outbox pattern. Postgres remains system of record. |
| MedicineUse JSONB shapes diverge across consumers | High | Medium | Documented schemas in `target_schemas.go` + JSONB validators. Schema versioning via `target.kind` enum value. |
| Delta-on-write performance is inadequate at scale | Low | Medium | Decision deferred to implementation; both DB trigger and service-layer measured. KB-26 already does similar baseline computation; perf characteristics known. |
| FHIR mapper introduces lossy round-trip | Medium | Medium | Round-trip tests (`Resident → AU Patient → Resident`) in CI. Lossy fields explicitly documented in mapper code. |
| gRPC/REST surface bloats with v2 entity churn | Medium | Low | Use proto definitions as canonical contract; versioned `v1` namespace from day one. |

---

## 9. Acceptance criteria

Phase 1B-β is complete when **all** of the following hold:

1. `shared/v2_substrate/` package exists with all 7 entity types (Resident, Person, Role, MedicineUse, Observation, Event, EvidenceTrace).
2. FHIR mappers exist for the 4 FHIR-mappable entities (Resident, Person+Role, MedicineUse, Observation) with round-trip tests passing.
3. kb-20 migration 008 (parts 1 + 2) applies cleanly on a fresh kb-20 database AND on a database with existing kb-20 data (compatibility verified).
4. kb-22 migration 008 applies cleanly. Neo4j projection bootstrap runs successfully against a populated kb-22 database.
5. gRPC + REST endpoints exist on kb-20 for all 5 kb-20 entities and on kb-22 for Event + EvidenceTrace.
6. shared/v2_substrate/client/ provides typed Go clients to those endpoints.
7. Each milestone exit criterion (§6) passes.
8. No breaking changes to existing kb-20 consumers (medication-service, KB-26, KB-22).
9. Out-of-scope items (§7) are explicitly NOT delivered.

---

## 10. References

- Vaidshala v2 Revision Mapping Parts 3 (system architecture), 6 (MVP/V1/V2 sequencing)
- Layer 1 v2 Implementation Guidelines Part 3 (Category B patient state sources)
- HL7 AU Base Implementation Guide v6.0.0 — `kb-3-guidelines/knowledge/au/integration_specs/hl7_au/base_ig_r4/`
- MHR FHIR Gateway IG v5.0 — `kb-3-guidelines/knowledge/au/integration_specs/adha_fhir/mhr_gateway_ig_v1_4_0/`
- Existing Source Registry schema — `kb-22-hpi-engine/migrations/004_clinical_source_registry.sql`
- Existing ScopeRules schema — `kb-22-hpi-engine/migrations/007_au_regulatory_extension.sql`
- Existing patient_profile schema — `kb-20-patient-profile/migrations/001_initial_schema.sql` and 006/007
- Existing kb-26 event types — `kb-26-metabolic-digital-twin/internal/models/events.go`
