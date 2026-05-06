# Layer 2 Substrate Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete the Layer 2 substrate so Layer 3 v2 rule encoding has a baseline-aware substrate to reason against — running baselines + delta-on-write at scale, active concerns lifecycle, care intensity tagging, capacity assessment, hospital discharge reconciliation, MHR + pathology auto-ingestion, EvidenceTrace bidirectional graph queries.

**Architecture:** Builds on Phase 1B-β.2's shipped substrate entities (Resident, Person+Role, MedicineUse-with-intent, Observation-with-delta, BaselineProvider seam) by completing the Clinical state machine, ingestion pipeline, and EvidenceTrace graph. Streaming pipeline (Apache Flink or Kafka Streams) for compute-on-write at facility scale. Relational+edge-table store with materialised views for the EvidenceTrace graph. SOAP/CDA → MHR FHIR Gateway transition for pathology. PDF + MHR-structured discharge summary reconciliation.

**Tech Stack:** Go 1.22+ (extends shared/v2_substrate + kb-20 patterns), PostgreSQL 16, Apache Flink or Kafka Streams (decision deferred to Wave 2), HAPI FHIR client libraries for MHR Gateway, lib/pq, gin, github.com/google/uuid.

**Predecessor:** Phase 1B-β.2 (β.1 actor model + β.2-A MedicineUse + β.2-B Observation + β.2-C Delta-on-write — all shipped on `main` at HEAD `ee6c5c1d`)

---

## Executive Summary

Layer 2 build is sized at **22-24 weeks of dedicated engineering** (3-4 engineers + clinical informatics partnership). Phase 1B-β.2 has already delivered the core entity scaffolding for Layer 2 doc Wave 1, so this plan starts from Wave 0 (history hygiene) and continues into Waves 2-6.

| Wave | Focus | Duration | Maps to Layer 2 doc |
|------|-------|----------|---------------------|
| 0 | Commit history cleanup, retroactive β.1/β.2-A landing | 1-2 days | Pre-flight |
| 1 (residual) | Wave 1 carry-over: Event entity + EvidenceTrace v0 + Identity matching | 3 weeks | Part 5.1 |
| 2 | Clinical state machine completion | 6 weeks | Part 2, Part 5.2 |
| 3 | MHR + pathology integration | 4 weeks | Part 3.2 (MHR), Part 5.3 |
| 4 | Hospital discharge reconciliation | 4 weeks | Part 3.2 (discharge), Part 5.4 |
| 5 | EvidenceTrace bidirectional graph hardening | 4 weeks | Part 1.6, Part 6 Failure 6 |
| 6 | Stabilisation + hardening | 4 weeks | Part 8 (4 weeks of hardening) |

Note: the Layer 2 doc's "Wave 1 — substrate entities and basic ingestion" (6 weeks) is partially shipped (entities + delta-on-write), and the residual three weeks (Event, EvidenceTrace v0, Identity matching, CSV ingestor) are folded into Wave 1-residual below.

---

## File Structure

New top-level dirs created during this plan:

```
backend/shared-infrastructure/knowledge-base-services/
├── shared/v2_substrate/
│   ├── models/                        # extend: event.go, evidence_trace.go,
│   │                                  # active_concern.go, care_intensity.go,
│   │                                  # capacity_assessment.go, daa_schedule.go,
│   │                                  # cfs_score.go, akps_score.go, dbi_score.go, acb_score.go
│   ├── validation/                    # extend with corresponding validators
│   ├── fhir/                          # extend: provenance_mapper.go, audit_event_mapper.go,
│   │                                  # diagnostic_report_mapper.go, document_reference_mapper.go
│   ├── delta/                         # extend: persistent_baseline_provider.go,
│   │                                  # baseline_config.go, trajectory_detector.go,
│   │                                  # active_concern_exclusion.go
│   ├── identity/                      # NEW: matcher.go, ihi_matcher.go, fuzzy_matcher.go
│   ├── ingestion/                     # NEW: csv_enrmc.go, mhr_soap_cda.go, mhr_fhir_gateway.go,
│   │                                  # discharge_pdf.go, discharge_mhr.go, normaliser.go
│   ├── reconciliation/                # NEW: diff.go, classifier.go, worklist.go
│   ├── evidence_trace/                # NEW: graph.go, edge_store.go, query.go, materialised_views.go
│   ├── clinical_state/                # NEW: state_updater.go, baseline_engine.go,
│   │                                  # active_concerns.go, care_intensity_engine.go
│   ├── streaming/                     # NEW (Wave 2): flink_jobs/ or kafka_streams/ scaffolding
│   ├── client/                        # extend: kb20_client_evidencetrace.go,
│   │                                  # kb20_client_event.go, kb20_client_concern.go
│   └── interfaces/                    # extend: BaselineProvider (persistent),
│                                      # IdentityMatcher, ReconciliationStore, EvidenceTraceStore
│
├── kb-20-patient-profile/
│   ├── internal/api/                  # extend handlers for new entities
│   ├── internal/storage/              # extend store with new tables
│   └── migrations/
│       ├── 009_event_evidencetrace.sql
│       ├── 010_active_concerns.sql
│       ├── 011_care_intensity.sql
│       ├── 012_capacity_assessment.sql
│       ├── 013_baseline_persistent_store.sql
│       ├── 014_baseline_config.sql
│       ├── 015_scoring_instruments.sql
│       ├── 016_identity_mapping.sql
│       ├── 017_pathology_ingest.sql
│       ├── 018_discharge_reconciliation.sql
│       ├── 019_evidencetrace_edges.sql
│       ├── 020_evidencetrace_views.sql
│       └── 021_daa_schedule.sql
│
└── kb-26-metabolic-digital-twin/      # NEW adapter (Wave 2.1): provides KB20-facing
    └── pkg/baseline_adapter/          # impl of BaselineProvider backed by twin time-series
```

Tests follow existing pattern: `_test.go` colocated with source for unit, integration tests in `kb-20-patient-profile/tests/integration/`.

---

## Wave 0 — Commit history cleanup + retroactive β.1/β.2-A landing (1-2 days)

**Goal:** Make `main` buildable from a fresh clone. β.1 entity files and β.2-A MedicineUse files were authored during the phase but never committed; recent β.2-B/β.2-C commits assumed they existed. A `git clone` of `main` today will not compile because untracked files in working tree are referenced by committed code.

**Background:** `git status --short` currently shows ~79 paths in three buckets:
1. **β.1 actor model + β.2-A MedicineUse files** that must land on main (untracked files in `shared/v2_substrate/models/`, `shared/v2_substrate/validation/`, `shared/v2_substrate/fhir/`, `shared/v2_substrate/client/` and migration `008_part1_actor_model.sql`, `008_part2_clinical_primitives_partA.sql`).
2. **kb-20 storage + handler tests** (`v2_substrate_handlers_test.go`, `v2_substrate_store_test.go`) that were written during β.2-A/B but never committed.
3. **Unrelated v4/v5 channel/specialist work** (modified files in `shared/extraction/v4/`, `shared/tools/guideline-atomiser/`, kb-22 migration `007_au_regulatory_extension.sql`, kb-3 regulatory dirs, Layer1-Layer3 spec docs) that belong to a different phase and must NOT be commingled.

### 0.1 Audit working tree and bucket every entry
- [ ] Run `git status --short` and capture full list to `/tmp/wave0_status.txt`.
- [ ] For each path, classify into one of four buckets: (A) β.1, (B) β.2-A MedicineUse, (C) β.2 tests, (D) unrelated/v5/v4/Layer-3 docs.
- [ ] Document the bucketing in a Wave 0 audit note (transient — delete after Wave 0 done).

**Acceptance:** every path in `git status --short` has a bucket assignment and a target action (commit on this branch, stash, or leave untracked).

### 0.2 Verify build is currently broken without these files
- [ ] `cd backend/shared-infrastructure/knowledge-base-services` and run `go build ./shared/v2_substrate/... ./kb-20-patient-profile/...` from a stashed-clean tree (`git stash --include-untracked`). Confirm build fails with missing-symbol errors referencing β.1 entity types.
- [ ] `git stash pop` to restore working tree.

**Acceptance:** documented evidence that without bucket-A and bucket-B paths, the codebase does not build.

### 0.3 Land β.1 entity files retroactively (bucket A)
- [ ] Stage and commit in this exact order, one commit per logical unit:
  1. `feat(v2_substrate): add Resident model + validator` — `models/resident.go`, `models/resident_test.go`, `validation/resident_validator.go`
  2. `feat(v2_substrate): add Person model + validator` — `models/person.go`, `models/person_test.go`, `validation/person_validator.go`
  3. `feat(v2_substrate): add Role model + validator with credential subschema` — `models/role.go`, `models/role_test.go`, `validation/role_validator.go`
  4. `feat(v2_substrate/fhir): Patient mapper + Practitioner mapper` — `fhir/patient_mapper.go`, `fhir/patient_mapper_test.go`, `fhir/practitioner_mapper.go`, `fhir/practitioner_mapper_test.go`
  5. `feat(kb-20): migration 008_part1 actor model schema` — `migrations/008_part1_actor_model.sql`
- [ ] After each commit, `go build ./shared/v2_substrate/...` must pass.

**Acceptance:** five commits land; `go build ./shared/v2_substrate/...` passes after each.

### 0.4 Land β.2-A MedicineUse files retroactively (bucket B)
- [ ] Stage and commit:
  1. `feat(v2_substrate): add MedicineUse model with intent/target/stop_criteria` — `models/medicine_use.go`, `models/medicine_use_test.go`, `models/target_schemas.go`, `models/target_schemas_test.go`, `models/stop_criteria_schemas.go`, `models/stop_criteria_schemas_test.go`
  2. `feat(v2_substrate): MedicineUse + target validators` — `validation/medicine_use_validator.go`, `validation/target_validator.go`
  3. `feat(v2_substrate/fhir): MedicationRequest mapper for MedicineUse` — `fhir/medication_request_mapper.go`, `fhir/medication_request_mapper_test.go`
  4. `feat(kb-20): migration 008_part2_partA MedicineUse schema` — `migrations/008_part2_clinical_primitives_partA.sql`
- [ ] Build check after each commit.

**Acceptance:** four commits land; `go build ./shared/v2_substrate/...` passes; full `go test ./shared/v2_substrate/...` passes.

### 0.5 Land kb-20 test files (bucket C)
- [ ] Commit:
  1. `test(kb-20): V2SubstrateStore unit tests` — `internal/storage/v2_substrate_store_test.go`
  2. `test(kb-20): v2 substrate REST handler tests` — `internal/api/v2_substrate_handlers_test.go`
- [ ] `go test ./kb-20-patient-profile/...` passes.

**Acceptance:** two test commits land; full kb-20 test suite green.

### 0.6 Quarantine unrelated work (bucket D)
- [ ] For modified files in `shared/extraction/v4/`, `shared/tools/guideline-atomiser/`: this is v4/v5 atomiser work that belongs to a separate phase. Do NOT commit on this branch. Run `git stash push -m "v5-atomiser-wip-2026-05-04" -- <paths>` to set aside.
- [ ] For untracked Layer 1c, Layer 2, Layer 3 docs (`Layer2_Implementation_Guidelines.md`, `Layer3_v2_Rule_Encoding_Implementation_Guidelines (1).md`, plans/specs under `docs/superpowers/`): commit as one `docs(layer2,layer3): add implementation guidelines and supporting plans/specs` commit. These are reference docs; the Layer 2 plan being saved is part of this commit.
- [ ] For kb-22 migration `007_au_regulatory_extension.sql` and kb-3 regulatory subdirs: confirm whether these belong to a current phase. If yes, commit as `feat(kb-22,kb-3): AU regulatory extension scaffolding`; if no, stash.
- [ ] `.gradle/` modifications: gitignore-only noise — discard with `git checkout -- .gradle/`.

**Acceptance:** working tree is either clean or contains only known-stashed/quarantined entries; no β.1/β.2/Layer 2 prerequisite work is left untracked.

### 0.7 Fresh-clone build verification
- [ ] In a separate working dir: `git clone <main-remote> /tmp/wave0-fresh-clone && cd /tmp/wave0-fresh-clone/backend/shared-infrastructure/knowledge-base-services`.
- [ ] Run `go build ./shared/v2_substrate/... ./kb-20-patient-profile/...` — must pass.
- [ ] Run `go test ./shared/v2_substrate/... ./kb-20-patient-profile/...` — must pass.
- [ ] Bring up Postgres via docker-compose, run `make migrate` (or equivalent) — migrations 001 through 008_part2_partB must apply cleanly in order.
- [ ] Document the verification in a Wave 0 completion note.

**Acceptance:** fresh clone of `main` builds and tests pass; migrations apply cleanly; Wave 0 complete.

**Wave 0 effort: 1-2 days. Dependencies: none. Blocks: all subsequent waves.**

---

## Wave 1-residual — Event + EvidenceTrace v0 + Identity matching + CSV ingestion (3 weeks)

**Goal:** Complete the residual scope of Layer 2 doc Part 5.1 (Wave 1) that β.2 did not cover: the Event entity, the EvidenceTrace v0 graph, the IdentityMatcher service, and a working CSV ingestor for one pilot eNRMC vendor format. Unlocks running baselines (Wave 2) by giving the ingestion pipeline an entry point.

### 1R.1 Event entity (3 days)

**Goal:** Event as first-class substrate entity per Layer 2 doc §1.5.

**Files:**
- `shared/v2_substrate/models/event.go` + `_test.go` — Event struct with type enum (29 types from §1.5), severity, structured/free-text descriptions, related-Observation/MedicineUse refs, triggered_state_changes, reportable_under list.
- `shared/v2_substrate/validation/event_validator.go` + `_test.go` — required-field, enum, reference-validity checks; per-event-type required-field tables (e.g. `fall` requires `severity`, `witnessed_by_refs`).
- `shared/v2_substrate/fhir/event_mapper.go` + `_test.go` — bidirectional map to FHIR `Encounter` for clinical events, `Communication` for system events; egress + ingress validation.
- `kb-20-patient-profile/migrations/009_event_evidencetrace.sql` (Event portion only) — `events` table, indexes on `(resident_ref, occurred_at DESC)`, `(event_type, occurred_at DESC)`, `(reportable_under, occurred_at DESC)`.
- `kb-20-patient-profile/internal/storage/event_store.go` + `_test.go` — CRUD + list-by-resident, list-by-type, list-by-reportable-bucket.
- `kb-20-patient-profile/internal/api/event_handlers.go` + `_test.go` — POST /events, GET /events/:id, GET /residents/:id/events, GET /events?type=&from=&to=.
- `shared/v2_substrate/client/kb20_client_event.go` + `_test.go` — KB20Client Event methods.

**Acceptance:** Event POST/GET round-trips through REST→store→DB; FHIR map round-trips; validator rejects missing required fields per event type; 100% test coverage on validator branches.

**Effort:** 3 days. **Dependencies:** Wave 0.

### 1R.2 EvidenceTrace v0 graph (5 days)

**Goal:** EvidenceTrace nodes + bidirectional edges queryable from day 1 per Layer 2 doc §1.6 and Recommendation 3 of Part 7. Wave 5 will harden; this is the foundational schema and write path.

**Files:**
- `shared/v2_substrate/models/evidence_trace.go` + `_test.go` — `EvidenceTraceNode` struct with state_machine enum, state_change_type, actor (role+person+authority), inputs[] (input_type, input_ref, role_in_decision), reasoning_summary, outputs[], graph_edges (upstream[], downstream[]).
- `shared/v2_substrate/validation/evidence_trace_validator.go` + `_test.go`.
- `shared/v2_substrate/fhir/provenance_mapper.go` + `_test.go` — map EvidenceTrace → FHIR Provenance (resource-change layer); preserve Vaidshala-specific reasoning_summary/graph_edges via extensions.
- `shared/v2_substrate/fhir/audit_event_mapper.go` + `_test.go` — map system EvidenceTrace nodes → FHIR AuditEvent (operational layer).
- `shared/v2_substrate/evidence_trace/graph.go` + `_test.go` — pure graph types: `Node`, `Edge{From, To, EdgeKind}`, `EdgeKind` enum (`led_to`, `derived_from`, `evidence_for`, `suppressed`).
- `shared/v2_substrate/evidence_trace/edge_store.go` interface + tests.
- `shared/v2_substrate/evidence_trace/query.go` + `_test.go` — `TraceForward(nodeID, maxDepth)` and `TraceBackward(nodeID, maxDepth)` over the edge store; cycle detection; depth cap.
- `kb-20-patient-profile/migrations/009_event_evidencetrace.sql` (EvidenceTrace portion) — `evidence_trace_nodes` table, `evidence_trace_edges` table with `(from_node, edge_kind)` and `(to_node, edge_kind)` indexes for bidirectional traversal.
- `kb-20-patient-profile/internal/storage/evidence_trace_store.go` + `_test.go` — CRUD + edge insert + `Forward`/`Backward` traversal queries (recursive CTE in PG).
- `kb-20-patient-profile/internal/api/evidence_trace_handlers.go` + `_test.go` — POST /evidence-trace/nodes, POST /evidence-trace/edges, GET /evidence-trace/:id/forward, GET /evidence-trace/:id/backward.
- `shared/v2_substrate/client/kb20_client_evidencetrace.go` + `_test.go`.

**Schema sketch:**

```sql
CREATE TABLE evidence_trace_nodes (
  id UUID PRIMARY KEY,
  state_machine TEXT NOT NULL CHECK (state_machine IN ('Authorisation','Recommendation','Monitoring','ClinicalState','Consent')),
  state_change_type TEXT NOT NULL,
  recorded_at TIMESTAMPTZ NOT NULL,
  occurred_at TIMESTAMPTZ NOT NULL,
  actor_role_ref UUID,
  actor_person_ref UUID,
  authority_basis_ref UUID,
  inputs JSONB NOT NULL DEFAULT '[]',
  reasoning_summary JSONB,
  outputs JSONB NOT NULL DEFAULT '[]',
  resident_ref UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_etn_resident_recorded ON evidence_trace_nodes(resident_ref, recorded_at DESC);
CREATE INDEX idx_etn_state_machine ON evidence_trace_nodes(state_machine, recorded_at DESC);

CREATE TABLE evidence_trace_edges (
  from_node UUID NOT NULL REFERENCES evidence_trace_nodes(id),
  to_node UUID NOT NULL REFERENCES evidence_trace_nodes(id),
  edge_kind TEXT NOT NULL CHECK (edge_kind IN ('led_to','derived_from','evidence_for','suppressed')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (from_node, to_node, edge_kind)
);
CREATE INDEX idx_ete_from ON evidence_trace_edges(from_node, edge_kind);
CREATE INDEX idx_ete_to ON evidence_trace_edges(to_node, edge_kind);
```

**Bidirectional query sketch (recursive CTE):**

```sql
-- forward traversal
WITH RECURSIVE downstream AS (
  SELECT to_node, 1 AS depth FROM evidence_trace_edges WHERE from_node = $1
  UNION
  SELECT e.to_node, d.depth + 1 FROM evidence_trace_edges e
  JOIN downstream d ON e.from_node = d.to_node
  WHERE d.depth < $2
)
SELECT * FROM evidence_trace_nodes WHERE id IN (SELECT to_node FROM downstream);
```

**Acceptance:** node + edge CRUD works; forward and backward traversal returns expected nodes within depth cap on a 100-node fixture graph; cycle detection prevents infinite recursion; FHIR Provenance map round-trips.

**Effort:** 5 days. **Dependencies:** Wave 1R.1 (Event references EvidenceTrace).

### 1R.3 Identity matching service (5 days)

**Goal:** IdentityMatcher per Layer 2 doc §3.3 — IHI-primary with confidence-tiered fallback and a manual-review queue.

**Files:**
- `shared/v2_substrate/identity/matcher.go` — `IdentityMatcher` interface: `Match(IncomingIdentifier) (MatchResult, error)`, `MatchResult{ResidentRef, Confidence, MatchPath, RequiresReview}`.
- `shared/v2_substrate/identity/ihi_matcher.go` — IHI exact-match path.
- `shared/v2_substrate/identity/fuzzy_matcher.go` — Medicare+name+DOB and name+DOB+facility fuzzy paths with Levenshtein on names; confidence scoring.
- `shared/v2_substrate/identity/matcher_test.go` — golden cases per Layer 2 doc §3.3.
- `kb-20-patient-profile/migrations/016_identity_mapping.sql` — `identity_mappings(identifier_kind, identifier_value, resident_ref, confidence, match_path, verified_by, verified_at)`; `identity_review_queue` for low-confidence matches.
- `kb-20-patient-profile/internal/storage/identity_store.go` + `_test.go`.
- `kb-20-patient-profile/internal/api/identity_handlers.go` + `_test.go` — POST /identity/match, GET /identity/review-queue, POST /identity/review/:id/resolve.

**Match algorithm pseudocode:**

```
Match(incoming):
  if incoming.IHI:
    if mapping = lookup(IHI, incoming.IHI): return {ResidentRef: mapping.ref, Confidence: HIGH, Path: "ihi"}
  if incoming.Medicare and incoming.Name and incoming.DOB:
    candidates = lookup_by_medicare(incoming.Medicare)
    for c in candidates:
      if levenshtein(c.name, incoming.name) <= 2 and c.dob == incoming.dob:
        return {ResidentRef: c.id, Confidence: MEDIUM, Path: "medicare+name+dob"}
  if incoming.Name and incoming.DOB and incoming.Facility:
    candidates = lookup_by_facility_dob(incoming.Facility, incoming.DOB)
    for c in candidates:
      if levenshtein(c.name, incoming.name) <= 3:
        return {ResidentRef: c.id, Confidence: LOW, Path: "name+dob+facility", RequiresReview: true}
  return {Confidence: NONE, RequiresReview: true}  // queue for human verification
```

Every match decision (success, fuzzy hit, queued) writes an EvidenceTrace node with `state_machine=ClinicalState`, `state_change_type=identity_match` and inputs referencing the incoming identifier set.

**Acceptance:** IHI exact-match returns HIGH; Medicare+name+DOB fuzzy returns MEDIUM; name+DOB+facility returns LOW with review-queue entry; mismatch creates a queue entry; every decision creates an EvidenceTrace node; manual override re-routes any historical data tagged with the prior resident_ref.

**Effort:** 5 days. **Dependencies:** Wave 1R.2 (EvidenceTrace).

### 1R.4 CSV eNRMC ingestor for pilot vendor (4 days)

**Goal:** end-to-end ingest from one CSV format (Telstra Health MedPoint export shape) through identity match → AMT normalisation → MedicineUse + Observation writes.

**Files:**
- `shared/v2_substrate/ingestion/csv_enrmc.go` + `_test.go` — column-mapped row → MedicineUse + Resident-link; tolerates missing intent (writes with intent_class=`unknown`).
- `shared/v2_substrate/ingestion/normaliser.go` + `_test.go` — AMT lookup, SNOMED-CT-AU lookup; uses kb-7-terminology.
- `shared/v2_substrate/ingestion/runner.go` + `_test.go` — orchestrator: read CSV → parse → identity-match → normalise → validate → write to kb-20.
- `kb-20-patient-profile/cmd/ingest-csv/main.go` — CLI for nightly cron.
- Integration test fixture: 50-row CSV from one pilot facility → expected substrate state.

**Acceptance:** running `ingest-csv --file <csv> --facility <id>` produces correct MedicineUse + Resident records; identity-match low-confidence rows land in review queue; every ingestion run writes an EvidenceTrace `extraction_pipeline` node; ingestion is idempotent (running twice produces no duplicates).

**Effort:** 4 days. **Dependencies:** Wave 1R.3.

### Wave 1-residual exit criterion

A pilot facility's week of CSV exports lands in the substrate; identity-matched residents have a queryable longitudinal profile; every ingestion event has an EvidenceTrace node; bidirectional traversal works on the seeded graph.

---

## Wave 2 — Clinical state machine completion (Weeks 1-6 of Wave 2 budget)

**Goal:** Layer 2 doc Part 2 in full — running baselines + delta-on-write at scale, active concerns lifecycle, care intensity tagging with transition events, capacity assessment with domain scope, scoring instrument integration, streaming pipeline scaffolding.

### Sub-task 2.1: Replace InMemoryBaselineProvider with persistent backing (Week 1, 5 days)

**Goal:** β.2-C shipped an InMemoryBaselineProvider stub. Replace with a Postgres-backed `PersistentBaselineProvider` that maintains running baselines per `(resident_id, observation_kind)` keyed by `vitalTypeKey`.

**Files:**
- `shared/v2_substrate/delta/persistent_baseline_provider.go` + `_test.go`.
- `kb-20-patient-profile/migrations/013_baseline_persistent_store.sql` — `baseline_state(resident_id, vital_type_key, baseline_value, baseline_window_days, n_observations, iqr, confidence, last_updated_at, last_observation_id)` with PK `(resident_id, vital_type_key)`.
- `kb-20-patient-profile/internal/storage/baseline_store.go` + `_test.go` — read + upsert; transactional with observation insert.
- Modify `kb-20-patient-profile/main.go` to wire `PersistentBaselineProvider` instead of `InMemoryBaselineProvider`.
- Modify `V2SubstrateStore` Observation insert path: after observation insert, in same transaction, call `RecomputeBaseline(resident, vital_type)` and upsert the row.

**Algorithm (per Layer 2 doc §2.2):**

```
RecomputeBaseline(resident_id, vital_type, lookback_days):
  obs = SELECT value, observed_at FROM observations
        WHERE resident_id=? AND vital_type_key=?
          AND observed_at >= now() - INTERVAL ? DAY
        ORDER BY observed_at DESC
        LIMIT GREATEST(5, count_in_window)
  filter obs by exclude_during_active_concerns(resident_id, observation_kind)
  if len(obs) < 3:
    return {baseline_value: nil, confidence: insufficient_data}
  median = percentile(obs.value, 50)
  iqr = percentile(75) - percentile(25)
  confidence =
    if len(obs) >= 7 and iqr < 0.25 * median: HIGH
    elif len(obs) >= 4 and iqr < 0.50 * median: MEDIUM
    else: LOW
  return {baseline_value: median, baseline_window_days: lookback_days,
          n_observations: len(obs), iqr, confidence, last_updated_at: now()}
```

**Acceptance:** observation insert updates baseline_state row in same transaction; ten back-to-back inserts produce a stable baseline that matches a reference computation in the test; baseline_state survives restart (proves persistence vs in-memory); confidence tier transitions HIGH→MEDIUM→LOW correctly as IQR widens.

**Effort:** 5 days. **Dependencies:** Wave 1-residual.

### Sub-task 2.2: Per-observation-type baseline configuration (Week 1-2, 4 days)

**Goal:** Layer 2 doc §2.2 — per-observation-type configuration table driving lookback window, exclusion rules, morning-only filters, velocity flags.

**Files:**
- `shared/v2_substrate/delta/baseline_config.go` + `_test.go` — `BaselineConfig{ObservationType, WindowDays, MinObsForHighConfidence, ExcludeDuringActiveConcerns []string, MorningOnly bool, FlagVelocity bool}`.
- `kb-20-patient-profile/migrations/014_baseline_config.sql` — `baseline_configs` seed table; CSV seed with rows for potassium, systolic_BP, weight, agitation_episode_count, eGFR (per Layer 2 doc sample).
- Modify `RecomputeBaseline` to consult config table and apply morning-only filter (06:00–11:00 local), exclude_during_active_concerns filter (Sub-task 2.3 dependency), velocity flag detection.

**Acceptance:** seed configs load on first migration; potassium uses 14-day window, weight uses 90; morning-only filter on systolic_BP excludes afternoon readings; eGFR velocity flag triggers when ≥20% decline in 14 days.

**Effort:** 4 days. **Dependencies:** Sub-task 2.1.

### Sub-task 2.3: Active concern lifecycle entity + state machine (Week 2-3, 8 days)

**Goal:** Layer 2 doc §2.3 — active concerns as first-class entities with start triggers, expected resolution windows, owner roles, monitoring links, resolution paths.

**Files:**
- `shared/v2_substrate/models/active_concern.go` + `_test.go` — `ActiveConcern{ID, ResidentRef, Type (enum from §2.3), StartedAt, StartedByEventRef, ExpectedResolutionAt, OwnerRoleRef, RelatedMonitoringPlanRef, ResolutionStatus (open|resolved_stop_criteria|escalated|expired_unresolved), ResolvedAt, ResolutionEvidenceTraceRef}`.
- `shared/v2_substrate/validation/active_concern_validator.go` + `_test.go`.
- `shared/v2_substrate/fhir/active_concern_mapper.go` — map to FHIR `Condition` with `clinical-status` extension.
- `kb-20-patient-profile/migrations/010_active_concerns.sql` — `active_concerns` table + indexes on `(resident_id, resolution_status)`, `(expected_resolution_at)` for expiry sweeper.
- `kb-20-patient-profile/internal/storage/active_concern_store.go`.
- `kb-20-patient-profile/internal/api/active_concern_handlers.go` — POST /residents/:id/active-concerns, GET, PATCH (resolve), GET /active-concerns/expiring?within=Xh.
- `shared/v2_substrate/clinical_state/active_concerns.go` — engine: `OnEvent(event)` opens concerns per type→trigger map; `OnObservation(obs)` may resolve concerns (e.g., 3-day-zero-agitation resolves `behavioural_titration_window`); cron `SweepExpired()` marks concerns past `expected_resolution_at` as `expired_unresolved` and emits a `concern_expired_unresolved` Event (which itself triggers downstream rules per §2.3 closing example: "fall risk reassessment fires when post_fall_72h expires unresolved").
- Wire active-concern membership into `RecomputeBaseline`'s exclusion filter (closes Sub-task 2.2 dependency).

**Concern type → trigger map seed:** post_fall_72h triggered by `event_type=fall`; antibiotic_course_active triggered by MedicineUse insert with intent_class=treatment + ATC J01; new_psychotropic_titration_window triggered by MedicineUse insert with ATC N05; etc. (full 10-row seed in migration).

**Acceptance:** fall Event opens `post_fall_72h` concern with `expected_resolution_at = occurred_at + 72h`; cron after 72h with no resolving event marks `expired_unresolved` and emits the cascade Event; baselines computed during the window exclude that window's observations for configured kinds; FHIR Condition map round-trips.

**Effort:** 8 days. **Dependencies:** Sub-task 2.2.

### Sub-task 2.4: Care intensity tag with transition events (Week 3, 5 days)

**Goal:** Layer 2 doc §2.4 — care intensity as resident state with transition events that propagate.

**Files:**
- `shared/v2_substrate/models/care_intensity.go` + `_test.go` — `CareIntensity{ResidentRef, Tag (active_treatment|rehabilitation|comfort_focused|palliative), EffectiveDate, DocumentedByRoleRef, ReviewDueDate, RationaleStructured, RationaleFreeText}`.
- `shared/v2_substrate/validation/care_intensity_validator.go`.
- `kb-20-patient-profile/migrations/011_care_intensity.sql` — `care_intensity_history` (append-only, current row = latest by effective_date) + materialised view `care_intensity_current`.
- `kb-20-patient-profile/internal/storage/care_intensity_store.go`.
- `kb-20-patient-profile/internal/api/care_intensity_handlers.go` — POST (transition), GET current, GET history.
- `shared/v2_substrate/clinical_state/care_intensity_engine.go` — `OnTransition(from, to)` emits Event of type `care_intensity_transition`, scans recommendations/monitoring plans for ones marked `revisit_on_care_intensity_change=true`, writes EvidenceTrace nodes for each cascade.

**Acceptance:** transition from active_treatment→palliative writes new history row, updates materialised view, emits transition Event, surfaces a worklist item for ACOP pharmacist review of preventive medications, writes one EvidenceTrace node per cascade.

**Effort:** 5 days. **Dependencies:** Sub-task 2.3.

### Sub-task 2.5: Capacity assessment objects with domain scope (Week 4, 5 days)

**Goal:** Layer 2 doc §2.5 — capacity as separate domain-scoped object, not a single resident attribute.

**Files:**
- `shared/v2_substrate/models/capacity_assessment.go` + `_test.go` — fields per §2.5.
- `shared/v2_substrate/validation/capacity_assessment_validator.go`.
- `shared/v2_substrate/fhir/capacity_assessment_mapper.go` — map to FHIR `Observation` with category=`assessment` and code from a Vaidshala-defined CodeSystem.
- `kb-20-patient-profile/migrations/012_capacity_assessment.sql` — append-only `capacity_assessments`; current view per `(resident_id, domain)`.
- `kb-20-patient-profile/internal/storage/capacity_assessment_store.go`.
- `kb-20-patient-profile/internal/api/capacity_handlers.go` — POST, GET current per domain, GET history per domain.
- Hook: when capacity outcome=`impaired` is written for domain `medical_decisions`, emit Event `capacity_change` and write EvidenceTrace node tagged with `state_machine=Consent` (Consent state machine consumes this in Layer 3).

**Acceptance:** independent capacity per domain (impaired-medical does not imply impaired-financial); appending a new assessment for same domain becomes the current; impaired-medical write triggers Consent-tagged EvidenceTrace node.

**Effort:** 5 days. **Dependencies:** Sub-task 2.4.

### Sub-task 2.6: CFS v2.0 + AKPS + DBI + ACB scoring integration (Week 5, 5 days)

**Goal:** Layer 2 doc §2.4 — capture structured prognostic + functional scores that inform (do not automate) care intensity decisions.

**Files:**
- `shared/v2_substrate/models/cfs_score.go`, `akps_score.go`, `dbi_score.go`, `acb_score.go` (each + `_test.go`) — score value, instrument version, assessor role, computed_at, components (DBI and ACB are computed from MedicineUse list, so include `computation_inputs` as MedicineUse refs).
- `shared/v2_substrate/scoring/dbi_calculator.go` + `_test.go` — Drug Burden Index: anticholinergic + sedative load; iterate MedicineUse list, look up DBI weights from a seed table, sum.
- `shared/v2_substrate/scoring/acb_calculator.go` + `_test.go` — Anticholinergic Cognitive Burden: weighted sum from seed table.
- `shared/v2_substrate/scoring/cfs_capture.go`, `akps_capture.go` — these are clinician-entered, not computed; capture+validate.
- `kb-20-patient-profile/migrations/015_scoring_instruments.sql` — four tables, append-only; current view per (resident, instrument); seed tables `dbi_drug_weights` and `acb_drug_weights`.
- DBI/ACB recompute trigger: any MedicineUse insert/update/end touches the per-resident burden recompute (eventual consistency, runs on outbox).
- `kb-20-patient-profile/internal/api/scoring_handlers.go` — POST CFS/AKPS, GET current per resident, GET DBI/ACB current.
- Care intensity engine surfaces a worklist hint when CFS≥7 or AKPS≤40 (does not automate transition).

**Acceptance:** seed weights cover top-100 aged-care medications; DBI compute on a fixture matches clinical reference; CFS≥7 surfaces a hint without auto-transitioning; recompute triggers on MedicineUse change.

**Effort:** 5 days. **Dependencies:** Sub-task 2.4.

### Sub-task 2.7: Streaming pipeline decision + scaffolding (Week 6, 5 days)

**Goal:** Layer 2 doc §3.4 — pick Flink vs Kafka Streams, scaffold the topology, validate compute-on-write throughput.

**Decision criteria (capture in an ADR `docs/adr/2026-05-XX-streaming-pipeline-choice.md`):**
- Existing Kafka footprint (Confluent Cloud already in use per CLAUDE.md): both work.
- Stage 1 stream service is Java; team Java skill exists: Flink and Kafka Streams both viable.
- Operational complexity: Kafka Streams is library-mode (deploys with the Go service via a sidecar), Flink is a separate cluster.
- **Recommendation (subject to ADR):** Kafka Streams library mode with one Java sidecar per kb-20 instance; revisit Flink only if window-join requirements outgrow Streams.

**Files:**
- `docs/adr/2026-05-XX-streaming-pipeline-choice.md` — decision record.
- `shared/v2_substrate/streaming/topology.md` — topic + processor diagram per Layer 2 doc §3.4.
- `backend/stream-services/substrate-pipeline/` — new module: Java Kafka Streams app with three processors (identity_matching, normalisation, substrate_writer); each processor exposes metrics on Prometheus.
- Topics: `raw_inbound_events`, `identified_events`, `normalised_events`, `substrate_updates`. Add to `setup-kafka-topics.py`.
- Substrate writer is a thin layer that calls kb-20 REST (preserves transactional ownership of substrate writes in Go).
- Load test: synthesise 2,000 observations/day for 200 residents and confirm end-to-end p95 <5s, baseline_state lag <30s.

**Acceptance:** ADR signed; topology deployed in dev; load test green; identity_matching processor consumes raw events and emits identified events; baseline_state lag observable on Grafana.

**Effort:** 5 days. **Dependencies:** Sub-task 2.1 (persistent baseline must exist before async write path).

### Wave 2 exit criterion

Running baselines exist for pilot facility residents; rules can query "is this observation abnormal relative to baseline?" via the `delta_flag` column on observations_v2; care intensity transitions propagate to dependent state machines via Event + EvidenceTrace; CFS/AKPS/DBI/ACB scores are queryable; load test demonstrates 2,000 obs/day/facility headroom.

**Wave 2 effort: 32 working days ≈ 6 weeks. Dependencies: Wave 1-residual. Layer 2 doc coverage: Part 2 in full, Part 5.2.**

---

## Wave 3 — MHR + pathology integration (4 weeks)

**Goal:** Layer 2 doc §3.2 (MHR), §5.3 — pathology auto-ingestion via MHR for residents with consent; per-vendor HL7 fallback for non-MHR residents; SOAP/CDA path now, FHIR Gateway path as ADHA matures.

### Sub-task 3.1: MHR SOAP/CDA gateway client (Week 1, 5 days)

**Goal:** ADHA B2B Gateway SOAP/CDA client per Layer 2 doc §3.2 — production-mature path for July 2026 Sharing-by-Default.

**Files:**
- `shared/v2_substrate/ingestion/mhr_soap_cda.go` + `_test.go` — SOAP envelope construction, NASH PKI auth headers, CDA document fetch + parse.
- `shared/v2_substrate/ingestion/cda_parser.go` + `_test.go` — CDA → internal pathology DTO; uses LOINC-AU code map.
- `shared/v2_substrate/identity/mhr_ihi_resolver.go` — MHR responses are IHI-keyed, so this is a thin wrapper over IdentityMatcher's IHI path.
- Test fixtures from ADHA conformance test pack (sample CDA documents).
- `kb-20-patient-profile/migrations/017_pathology_ingest.sql` — `pathology_ingest_log(source, document_id, ihi, ingested_at, status, error)` for idempotency + audit.
- `kb-20-patient-profile/cmd/mhr-poll/main.go` — CLI that polls MHR for new pathology per consenting resident, runs per scheduled cron.

**Acceptance:** sample CDA from ADHA conformance pack parses into Observation list; LOINC-AU codes resolve; observation inserts trigger baseline recompute; idempotency: re-polling the same document does not duplicate.

**Effort:** 5 days. **Dependencies:** Wave 2.

### Sub-task 3.2: MHR FHIR Gateway client (Week 2, 5 days)

**Goal:** ADHA FHIR Gateway IG v1.4.0 client. Co-exists with SOAP/CDA; configuration flag selects per-resident.

**Files:**
- `shared/v2_substrate/ingestion/mhr_fhir_gateway.go` + `_test.go` — HAPI FHIR client; OAuth2 + NASH; DiagnosticReport + Observation fetch.
- `shared/v2_substrate/fhir/diagnostic_report_mapper.go` + `_test.go` — FHIR DiagnosticReport → internal pathology DTO; reuse cda_parser's downstream contract.
- Configuration: `mhr_gateway_mode` per facility (`soap_cda` | `fhir_gateway` | `dual`).
- Integration test against a FHIR Gateway sandbox.

**Acceptance:** FHIR Gateway path produces same internal DTO as SOAP/CDA path on equivalent test data; both can run concurrently in `dual` mode without duplicate inserts (idempotency keyed on document_id + source).

**Effort:** 5 days. **Dependencies:** Sub-task 3.1.

### Sub-task 3.3: Per-pathology-vendor HL7 fallback (Week 3, 5 days)

**Goal:** Non-MHR-consenting residents need direct HL7 from one or two pilot pathology vendors per Layer 2 doc §3.2.

**Files:**
- `shared/v2_substrate/ingestion/hl7_oru.go` + `_test.go` — HL7 v2.5 ORU^R01 parser; LOINC mapping.
- `backend/stream-services/substrate-pipeline/hl7_listener/` — MLLP listener Java module that publishes to `raw_inbound_events`.
- Per-vendor adapter table for the 2-3 pilot vendors (vendor-specific OBX subfield quirks).

**Acceptance:** HL7 ORU from one pilot vendor lands in substrate; LOINC mapping correct; identity match at MLLP boundary.

**Effort:** 5 days. **Dependencies:** Sub-task 3.1.

### Sub-task 3.4: Pathology baseline integration + lab trajectory rules surface (Week 4, 5 days)

**Goal:** ensure pathology Observations participate fully in baselines and trajectory detection per Layer 2 doc §2.2.

**Files:**
- Extend baseline_configs seed for pathology kinds (potassium, eGFR, sodium, magnesium, INR, HbA1c) with proper windows.
- Trajectory detector: 3-consecutive-same-direction flag per Layer 2 doc §1.4 — `shared/v2_substrate/delta/trajectory_detector.go` + `_test.go`. Already partially implied by β.2-C; this finalises trending logic in the persistent provider.
- Velocity flagging for eGFR (≥20% decline in 14 days) per Layer 2 doc §2.2.
- Wire `flag_velocity=true` configs to write `velocity_flag` column in observations table.

**Acceptance:** ten consecutive eGFR observations with 25% drop produce `velocity_flag=high` on the latest observation; trajectory `is_trending=true` after 3 same-direction readings; rules can query both via observations_v2.

**Effort:** 5 days. **Dependencies:** Sub-tasks 3.1-3.3.

### Wave 3 exit criterion

Pathology results land in substrate within hours of MHR upload (or HL7 receipt) for both MHR and non-MHR residents; baselines and trajectories incorporate pathology; eGFR velocity flag fires on real-world decline patterns.

**Wave 3 effort: 4 weeks. Dependencies: Wave 2. Layer 2 doc coverage: Part 3.2 (MHR + pathology), Part 5.3, Part 6 Failure 4 partial.**

---

## Wave 4 — Hospital discharge reconciliation (4 weeks)

**Goal:** Layer 2 doc §3.2 (hospital discharge), §5.4 — three coding systems, three timestamp regimes, three signers reconciled into a substrate-coherent state with ACOP pharmacist worklist.

### Sub-task 4.1: Discharge document ingestion (Week 1, 5 days)

**Files:**
- `shared/v2_substrate/ingestion/discharge_pdf.go` + `_test.go` — PDF upload endpoint; OCR via existing Tesseract or hosted (off-the-shelf, not built here); raw text + metadata stored.
- `shared/v2_substrate/ingestion/discharge_mhr.go` + `_test.go` — MHR-pulled structured discharge documents (HL7 CDA Discharge Summary, IHE XDS).
- `shared/v2_substrate/fhir/document_reference_mapper.go` — wrap discharge documents as FHIR DocumentReference with category=discharge.
- `kb-20-patient-profile/migrations/018_discharge_reconciliation.sql` — `discharge_documents`, `discharge_medication_lines` (parsed lines pre-reconciliation), `reconciliation_worklists`, `reconciliation_decisions`.

**Acceptance:** PDF upload stores document + extracted text; MHR structured discharge produces parsed medication lines.

**Effort:** 5 days. **Dependencies:** Wave 2.

### Sub-task 4.2: Pre-admission vs discharge diff engine (Week 2, 5 days)

**Goal:** Layer 2 doc §3.2 reconciliation algorithm.

**Files:**
- `shared/v2_substrate/reconciliation/diff.go` + `_test.go` — pre-admission MedicineUse list vs discharge medication line list, normalise both to AMT, classify per line:
  - `new_medication`: in discharge, not in pre-admission
  - `ceased_medication`: in pre-admission, not in discharge
  - `dose_change`: same AMT, different dose/freq
  - `unchanged`: same AMT, same dose/freq

**Acceptance:** unit test fixtures cover all four classes; AMT normalisation handles brand vs generic.

**Effort:** 5 days. **Dependencies:** Sub-task 4.1.

### Sub-task 4.3: Reconciliation classifier + worklist (Week 3, 5 days)

**Goal:** classify each diff into intent buckets per Layer 2 doc §3.2 reconciliation algorithm steps 5-7.

**Files:**
- `shared/v2_substrate/reconciliation/classifier.go` + `_test.go` — heuristics:
  - `acute_illness_temporary` if discharge document text near the line mentions acute event keywords (infection, surgery, AMI) and class matches expected acute use
  - `new_chronic` if line has long-term-class signal (statin started, antihypertensive escalation)
  - `reconciled_change` if dose change with explicit explanatory note
  - `unclear` otherwise (conservative default)
- `shared/v2_substrate/reconciliation/worklist.go` + `_test.go` — worklist generator: one line item per non-`unchanged` diff, assigned to ACOP pharmacist role at the resident's facility, due within 24h of `hospital_discharge` Event.
- `kb-20-patient-profile/internal/api/reconciliation_handlers.go` — POST /reconciliation/start, GET /reconciliation/:worklist, POST /reconciliation/:worklist/lines/:line/decide.

**Acceptance:** discharge ingest auto-creates a worklist within 60s of `hospital_discharge` Event; ACOP can see it filtered by their role+facility; decision API records ACOP's intent capture per line; each decision writes an EvidenceTrace node.

**Effort:** 5 days. **Dependencies:** Sub-task 4.2.

### Sub-task 4.4: ACOP decision write-back to MedicineUse with intent (Week 4, 5 days)

**Goal:** ACOP confirmation/modification flows into MedicineUse with intent/target/stop_criteria populated per Layer 2 doc §1.3.

**Files:**
- `shared/v2_substrate/reconciliation/writeback.go` + `_test.go` — decision → MedicineUse insert (new), update (dose change), end (cease) with intent_class, primary_indication, expected_benefit_horizon_months, planned_duration_months, stop_criteria pulled from ACOP form.
- UI scope: out of scope for this plan (Layer 4); back-end provides the API contract.
- Care intensity hook: if discharge medications include acute-illness-temporary lines, those MedicineUse rows are tagged `expected_review_at = discharge_date + 14d` so deprescribing review fires automatically per intent in Wave 2 active concern engine.

**Acceptance:** ACOP confirms 5 of 5 worklist lines; resulting MedicineUse rows have populated intent fields; substrate state matches discharge document; EvidenceTrace nodes link discharge document → diff → worklist → ACOP decision → MedicineUse change.

**Effort:** 5 days. **Dependencies:** Sub-task 4.3.

### Wave 4 exit criterion

A pilot resident returning from hospital produces a reconciliation worklist within hours; pharmacist completes reconciliation in the back-end API; substrate captures full intent for changed medications; full forward and backward EvidenceTrace traversal is queryable from `hospital_discharge` Event to the resulting MedicineUse changes.

**Wave 4 effort: 4 weeks. Dependencies: Wave 3 (and Wave 2). Layer 2 doc coverage: Part 3.2 discharge, Part 5.4.**

---

## Wave 5 — EvidenceTrace bidirectional graph hardening (4 weeks)

**Goal:** Layer 2 doc Recommendation 3 ("build EvidenceTrace as queryable graph from day 1") + Failure 6 ("graph query performance"). Wave 1-residual shipped v0; Wave 5 hardens for production scale and adds materialised views, query API surface, and the dual-resource Provenance/AuditEvent split.

### Sub-task 5.1: Materialised views for common query patterns (Week 1, 5 days)

**Files:**
- `kb-20-patient-profile/migrations/020_evidencetrace_views.sql` — materialised views:
  - `mv_recommendation_lineage(recommendation_id, upstream_evidence_count, upstream_observation_refs, upstream_event_refs, decision_outcome, downstream_outcome_refs)` — refreshed on EvidenceTrace insert via trigger.
  - `mv_observation_consequences(observation_id, downstream_recommendation_count, downstream_recommendations, downstream_acted_count)` — for "given an observation, what reasoning did it produce?"
  - `mv_resident_reasoning_summary(resident_id, last_30d_recommendation_count, last_30d_decision_count, average_evidence_per_recommendation)` — regulator-audit-ready.
- Refresh strategy: incremental via outbox events; full refresh nightly.

**Acceptance:** views populate within 30s of EvidenceTrace writes; query latency p95 <100ms on a 1M-node fixture graph; nightly full refresh completes within 10min on the same fixture.

**Effort:** 5 days. **Dependencies:** Wave 4.

### Sub-task 5.2: Query API surface for Layer 3 + audit (Week 2, 5 days)

**Files:**
- `shared/v2_substrate/evidence_trace/query.go` extends with:
  - `LineageOf(recommendationID) (Lineage, error)` — backward to all evidence inputs.
  - `ConsequencesOf(observationID) (Consequences, error)` — forward to all recommendations triggered.
  - `ReasoningWindow(residentID, from, to) (Summary, error)` — regulator-audit window query.
- `kb-20-patient-profile/internal/api/evidence_trace_handlers.go` extends with the three GET endpoints above.
- KB20Client extension methods.

**Acceptance:** Layer 3 (mock client) can query "given this BP-delta-baseline rule fire, what recommendation did it produce and what was the outcome?" in p95 <150ms; regulator-audit window query returns structured JSON suitable for ACQSC submission.

**Effort:** 5 days. **Dependencies:** Sub-task 5.1.

### Sub-task 5.3: FHIR Provenance + AuditEvent dual-resource alignment (Week 3, 5 days)

**Goal:** Layer 2 doc §1.6 dual-resource pattern. EvidenceTrace nodes that represent resource changes also produce a FHIR Provenance; system events also produce a FHIR AuditEvent.

**Files:**
- Extend `shared/v2_substrate/fhir/provenance_mapper.go` — every clinical EvidenceTrace node (Recommendation, Authorisation, Consent, ClinicalState transitions) maps to Provenance with `target=[entity refs]`, `agent=[role+person]`, `entity=[input refs with role]`.
- Extend `shared/v2_substrate/fhir/audit_event_mapper.go` — every system EvidenceTrace node (rule_fire, query, login propagated from auth-service) maps to AuditEvent.
- Outbound FHIR egress: an external auditor can pull Provenance + AuditEvent through the FHIR endpoint for ACQSC submission.

**Acceptance:** every EvidenceTrace insert produces exactly one Provenance OR one AuditEvent (mutually exclusive by node kind); FHIR validator accepts both resources; egress endpoint round-trips on a 100-node sample.

**Effort:** 5 days. **Dependencies:** Sub-task 5.2.

### Sub-task 5.4: Graph performance load test + indexing tune (Week 4, 5 days)

**Files:**
- Synthetic graph generator: 6 months of activity for 200 residents = ~500K nodes, ~1.5M edges.
- Load test with `pgbench` + custom forward/backward query workload.
- Tune indexes: confirm `(from_node, edge_kind)` and `(to_node, edge_kind)` BTREE plus a partial index for `edge_kind='led_to'` (the dominant traversal direction).
- Query analyser: rewrite recursive CTE traversals to use `LIMIT depth*fanout` short-circuit on max-depth.

**Acceptance:** forward traversal depth=5 p95 <200ms; backward depth=5 p95 <200ms; materialised view refresh <60s incremental, <10min full; 6-month synthetic dataset is queryable with all access patterns hitting their SLO.

**Effort:** 5 days. **Dependencies:** Sub-task 5.3.

### Wave 5 exit criterion

EvidenceTrace bidirectional graph queries hit production SLOs on a 6-month-of-activity synthetic dataset; FHIR Provenance/AuditEvent egress works for regulator audit; Layer 3 has a stable client API for lineage and consequence queries.

**Wave 5 effort: 4 weeks. Dependencies: Wave 4. Layer 2 doc coverage: Part 1.6, Part 6 Failure 6, Recommendation 3.**

---

## Wave 6 — Stabilisation + hardening (4 weeks)

**Goal:** Layer 2 doc Part 8 — "Plan for 4 additional weeks of hardening after the four waves complete." Cross-cutting integration testing, performance optimisation, failure-mode defence validation.

### Sub-task 6.1: Failure-mode defence validation (Week 1, 5 days)

For each of the six failure modes in Layer 2 doc Part 6, write an automated test that demonstrates the defence:

- **Failure 1 (compute-on-write performance):** load test that demonstrates p95 baseline-recompute lag <30s under 2,000 obs/day/facility × 5 facilities concurrent.
- **Failure 2 (identity match errors):** chaos test that injects an IHI typo and confirms the resulting low-confidence match queues for review rather than mis-routing.
- **Failure 3 (intent field sparseness):** rule-fire test confirming rules with `intent_required` predicate suppress (not fire) when intent_class=`unknown`.
- **Failure 4 (baseline contamination by acute periods):** end-to-end test: open active concern, ingest acute-period observations, close concern, verify post-resolution baseline excludes the acute window.
- **Failure 5 (care intensity transition lag):** alert test: CFS≥7 score writes a worklist hint within 60s of recording.
- **Failure 6 (graph query performance):** Wave 5 load test re-run as regression.

**Files:** `kb-20-patient-profile/tests/failure_modes/` — six test files, one per mode.

**Acceptance:** all six tests green in CI; documented in a Wave 6 failure-mode report.

**Effort:** 5 days. **Dependencies:** Waves 1R-5.

### Sub-task 6.2: Cross-state-machine integration test suite (Week 2, 5 days)

Layer 2 doc Part 4 specifies how Layer 2 integrates with each of the five state machines. Write integration tests against mocks for each:

- **Authorisation (§4.1):** credential expiry triggers cache invalidation in <50ms p95 read API.
- **Recommendation (§4.2):** baseline-delta event triggers a mock rule fire; Recommendation lifecycle write lands in EvidenceTrace.
- **Monitoring (§4.3):** expected-vs-received observation gap detection; trajectory triggers `abnormal` state.
- **Clinical (§4.4):** internal — already covered by Wave 2 tests.
- **Consent (§4.5):** capacity outcome change triggers Consent re-evaluation event.

**Files:** `kb-20-patient-profile/tests/state_machine_integration/` — five test files.

**Acceptance:** all five integration tests green; each writes the expected EvidenceTrace edges.

**Effort:** 5 days. **Dependencies:** Sub-task 6.1.

### Sub-task 6.3: End-to-end pilot scenario rehearsal (Week 3, 5 days)

Rehearse the Layer 2 doc Part 8 closing line ("the Sunday-night-fall walkthrough — exercising the substrate against a real-world scenario") against a synthetic scripted pilot:

- Sunday evening: fall Event ingested.
- Monday morning: post-fall vitals via nursing observation (delta-flagged), agitation episode logged on behavioural chart.
- Monday afternoon: hospital_admission Event for head CT.
- Tuesday morning: hospital_discharge Event with new medication (anti-emetic).
- Wednesday: ACOP pharmacist completes reconciliation.
- Thursday: pathology result via MHR (mild AKI) ingests, opens active concern AKI_watching, baselines for potassium recompute excluding the AKI window.
- Friday: care intensity transition from active_treatment→comfort_focused (family meeting outcome).
- Saturday: full forward/backward EvidenceTrace traversal from the Sunday fall to the Saturday care-intensity-transition demonstrates lineage intact.

**Acceptance:** end-to-end test passes; one runbook document captures the scenario for clinical-informatics partner review.

**Effort:** 5 days. **Dependencies:** Sub-task 6.2.

### Sub-task 6.4: Production readiness review + handoff (Week 4, 5 days)

- [ ] Performance SLO documentation for all read APIs.
- [ ] Operational runbooks: identity match queue triage, baseline drift investigation, EvidenceTrace audit query, MHR gateway error recovery.
- [ ] On-call rotation + alerting rules in Prometheus/Grafana.
- [ ] Layer 3 handoff document listing every substrate query Layer 3 will use, with example payloads and SLOs.
- [ ] Final security review (PHI access logging, NASH PKI rotation, IHI handling per Privacy Act).

**Acceptance:** runbook merged; SLO doc merged; Layer 3 handoff doc reviewed by Layer 3 lead; security review signed off.

**Effort:** 5 days. **Dependencies:** Sub-task 6.3.

### Wave 6 exit criterion

Layer 2 substrate is production-ready: failure-mode defences validated; cross-state-machine integration green; pilot scenario walkthrough passes; runbooks in place; Layer 3 has its handoff contract.

**Wave 6 effort: 4 weeks. Dependencies: Waves 1R-5. Layer 2 doc coverage: Part 4 in full, Part 6 in full, Part 8 hardening commitment.**

---

## Acceptance Criteria Coverage Map

| Layer 2 doc section | Wave | Sub-task |
|---------------------|------|----------|
| §1.1 Resident | β.1 (shipped) | n/a |
| §1.2 Person + Role | β.1 (shipped) | n/a |
| §1.3 MedicineUse | β.2-A (shipped) | n/a |
| §1.4 Observation (with delta) | β.2-B + β.2-C (shipped); §1.4 trajectory finalised | Wave 3 §3.4 |
| §1.5 Event | Wave 1-residual | §1R.1 |
| §1.6 EvidenceTrace | Wave 1-residual + Wave 5 | §1R.2, §5.1-5.4 |
| §2.1-2.2 Running baselines | Wave 2 | §2.1, §2.2 |
| §2.3 Active concerns | Wave 2 | §2.3 |
| §2.4 Care intensity tag + scoring | Wave 2 | §2.4, §2.6 |
| §2.5 Capacity assessment | Wave 2 | §2.5 |
| §3.1 Pipeline shape | Wave 2 + Wave 3 | §2.7 (streaming), §3.1-3.3 (sources) |
| §3.2 eNRMC ingestion | Wave 1-residual | §1R.4 |
| §3.2 MHR ingestion | Wave 3 | §3.1, §3.2 |
| §3.2 Hospital discharge | Wave 4 | §4.1-4.4 |
| §3.2 Dispensing pharmacy DAA | Out of MVP scope; punt to V1 (post-Wave 6) | n/a |
| §3.2 Behavioural chart | Out of MVP scope; punt to V1 (post-Wave 6) | n/a |
| §3.3 Identity matching | Wave 1-residual | §1R.3 |
| §3.4 Streaming pipeline | Wave 2 | §2.7 |
| §4.1-4.5 State machine integration | Wave 6 | §6.2 |
| §5.1-5.4 Sequencing | n/a (this plan IS the sequencing) | — |
| §6 Failure modes | Wave 6 | §6.1 |
| §7 Sharp recommendations | All — recommendation 1 (Clinical state machine) is Wave 2; recommendation 2 (MedicineUse intent) shipped β.2-A; recommendation 3 (EvidenceTrace as graph from day 1) is Wave 1-residual + Wave 5 | — |
| §8 Hardening 4 weeks | Wave 6 | §6.1-6.4 |

**Gaps explicitly punted (will land in V1 post-MVP, NOT this plan):**
- §3.2 Dispensing pharmacy DAA timing (FRED API integration) — Layer 2 doc §5.5 calls this Wave 5 (V1).
- §3.2 Behavioural chart structured ingestion via care-management vendor APIs (Leecare, AutumnCare) — Layer 2 doc §5.5 calls this Wave 6 (V1).
- §3.2 Direct hospital ADT feeds — Layer 2 doc §5.5 Wave 7 (V2).
- §3.2 NLP for free-text indication extraction at scale — Layer 2 doc §5.5 Wave 8 (V2).
- §3.2 Multi-vendor eNRMC FHIR coverage (this plan covers one pilot vendor) — Layer 2 doc §5.5 Wave 9 (V2).

---

## Open dependencies / risks (mapping Layer 2 doc Part 6 failure modes to Wave defences)

| Failure mode | Defence wave | Residual risk |
|--------------|--------------|---------------|
| F1: compute-on-write performance | Wave 2.1 (persistent baseline), Wave 2.7 (streaming), Wave 6.1 (load test) | Long-tail observation bursts during mass updates may spike baseline recompute lag; mitigation in Wave 6.1 monitor + autoscaling. |
| F2: identity match errors | Wave 1R.3 (matcher with confidence tiers + review queue), Wave 6.1 (chaos test) | IHI uptake post-July-2026 not 100%; non-IHI residents will incur MEDIUM/LOW match paths longer than expected. Operational mitigation: weekly identity review meeting. |
| F3: intent field sparseness | Wave 4 (reconciliation captures intent on every discharge); shipped β.2-A intent fields default to `unknown` rather than blocking | NLP-driven intent (V2) deferred; coverage grows organically as ACOP reviews accumulate. |
| F4: baseline contamination by acute periods | Wave 2.3 (active concerns), Wave 2.2 (exclude_during_active_concerns config) | Concern type → exclusion mapping is incomplete at launch; new clinical patterns require config additions. |
| F5: care intensity transition lag | Wave 2.4 (transition Events propagate), Wave 2.6 (CFS/AKPS hint surface) | CFS/AKPS not auto-recorded — relies on clinician entry; lag risk persists. |
| F6: graph query performance | Wave 1R.2 (correct indexing day 1), Wave 5 (materialised views, load test) | If query patterns evolve significantly post-launch, materialised view design may need revision. ADR for graph database migration drafted in Wave 5 if Postgres approach proves insufficient. |

**Cross-cutting risks not in Part 6:**
- **NASH PKI rotation:** every 2 years; operational runbook in Wave 6.4.
- **Confluent Cloud cost at scale:** Wave 2.7 ADR includes capacity model.
- **AMT/SNOMED-CT-AU/LOINC-AU release cadence:** kb-7-terminology updates quarterly; Wave 1R.4 ingestion respects code-version snapshotting.

---

## Handoff to Layer 3

Layer 3 v2 rule encoding (per the companion `Layer3_v2_Rule_Encoding_Implementation_Guidelines.md`) depends on the following substrate features being live before Layer 3 Wave 2 begins:

1. **Running baselines as queryable state** (Wave 2.1, 2.2): rule predicates of the form `PotassiumDeltaFromBaseline7Days > 0.8` resolve via the `observations_v2` view's `delta_value`, `delta_flag`, and `baseline_value` columns. SLO: read p95 <50ms.

2. **Active concerns table queryable per-resident** (Wave 2.3): rule suppression predicates (`if active_concern(post_acute_GI_bleed_watching) then suppress`) resolve via `GET /residents/:id/active-concerns?status=open`.

3. **Care intensity current view** (Wave 2.4): rule context predicates (`if care_intensity = palliative then ...`) resolve via `mv_care_intensity_current`.

4. **MedicineUse intent + target + stop_criteria** (β.2-A shipped, intent populated by Wave 4 reconciliation): rule predicates of the form `medicineUse.intent.intent_class = 'symptom_control' AND review_due_date < now()` resolve via standard MedicineUse columns.

5. **EvidenceTrace bidirectional query API** (Wave 1R.2 + Wave 5): rules write recommendation lineage into the graph; Layer 4 audit UI reads via `LineageOf(recommendationID)` and `ConsequencesOf(observationID)`. SLO: backward-traversal p95 <200ms at depth 5.

6. **Authorisation evaluator data plumbing** (Wave 1R via Person+Role + credentials shipped β.1; Wave 6.2 cross-state-machine test): credential expiry cache invalidation <50ms p95; PrescribingAgreement scope read available before Layer 3 Authorisation evaluator deploys.

7. **CFS/AKPS/DBI/ACB scores queryable per resident** (Wave 2.6): rule predicates for frailty/burden-aware suppression.

8. **Capacity assessment per domain** (Wave 2.5): Consent state machine queries (`capacity_for(medical_decisions) = impaired`) resolve via `capacity_assessment_current` view.

**Layer 3 Wave 2 may begin in parallel with Layer 2 Wave 5/6** because Layer 3 Wave 1 (rule scaffolding, CQL grammar) does not depend on Layer 2 read APIs being SLO-stable; Layer 3 Wave 2 (rule firing in shadow mode) does.

---

## Closing

Layer 2 is the load-bearing piece of the platform. β.1 + β.2 shipped the entity scaffolding. This plan completes the substrate: Wave 0 stabilises history; Wave 1-residual finishes Wave 1 of the Layer 2 doc; Waves 2-5 deliver the Clinical state machine, MHR + pathology, hospital discharge reconciliation, and EvidenceTrace graph hardening; Wave 6 is the four weeks of cross-cutting hardening the Layer 2 doc Part 8 explicitly demands. Total realistic Layer 2 timeline from this plan's start: 22-24 weeks to Layer-3-ready substrate.

Without this work, Layer 3 v2 rules fire on snapshots and produce alert noise. With it, the platform owns the substrate that turns observation noise into clinical signal — the moat the v2 product proposal said was non-negotiable.
