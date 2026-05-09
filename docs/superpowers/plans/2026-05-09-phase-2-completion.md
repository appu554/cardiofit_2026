# Phase 2-Completion — Production Blockers + Vocabulary + HTTP Endpoint Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the three production blockers + four other partial items surfaced by the 2026-05-09 Phase 2a/2b implementation gap analysis. Phase 2a + 2b shipped a structurally complete craft engine (257 race-clean tests, 5 migrations, 18 internal packages) but three concrete dependencies remain placeholders that gate shadow deployment, and four guideline-distinctive items are partial or vocabulary-mismatched. This plan closes them.

**Why a separate plan:** Phase 2a + 2b were scoped to land the substrate. Concrete substrate-backed implementations of the source interfaces (Postgres-backed `SubstrateClient`, real `AppropriatenessSource`, Stage 7 EvidenceTrace emission) are the production-readiness layer. Bundling them into 2a or 2b would have made those plans unwieldy. This completion plan tracks them explicitly so they don't fall off the radar.

**Architecture:** No new packages or services — extends kb-32 with concrete implementations of interfaces declared in 2a/2b. Touches `cmd/server/main.go` to swap placeholders for real implementations. Adds one HTTP endpoint (`POST /v1/framing/optout`) and one Postgres-backed registry implementation (`PostgresCitationRegistry`).

**Tech Stack:** Go, Gin, Postgres. Depends on Phase 2a + 2b complete on `feat/phase-1-trust-architecture-plans` branch.

---

## File Structure

**Modified files:**
- `kb-32-recommendation-craft/cmd/server/main.go` — swap `inMemorySubstrateClient` → `postgresSubstrateClient`, swap `DefaultAppropriatenessSource` → `SubstrateBackedAppropriatenessSource`, mount new optout endpoint
- `kb-32-recommendation-craft/internal/api/pipeline.go` — wire Stage 7 EvidenceTrace emission

**New files:**
- `kb-32-recommendation-craft/internal/store/postgres/substrate_client.go` + `_test.go` — Postgres-backed SubstrateClient reading kb-20-patient-profile data
- `kb-32-recommendation-craft/internal/appropriateness/substrate_scorer.go` + `_test.go` — five-dimension scorer driven by ClinicalSnapshot fields
- `kb-32-recommendation-craft/internal/store/postgres/citation_registry.go` + `_test.go` — Postgres-backed citations.Registry
- `kb-32-recommendation-craft/internal/api/optout_handlers.go` + `_test.go` — `POST /v1/framing/optout` endpoint for prescriber opt-out
- `kb-32-recommendation-craft/internal/lifecycle/evidence_trace.go` + `_test.go` — Stage 7 EvidenceTrace emission on `detected → drafted` transition
- `kb-32-recommendation-craft/tests/integration/end_to_end_with_real_stores_test.go` — integration test using all-Postgres-backed dependencies (skips without VAIDSHALA_TEST_DSN)

---

### Task 1: Postgres-backed SubstrateClient

**Files:**
- Create: `internal/store/postgres/substrate_client.go` + `_test.go`

Replace `inMemorySubstrateClient` (which returns zero-value `ClinicalSnapshot` for every resident) with a Postgres-backed reader that joins kb-20-patient-profile data into the kb-32 ClinicalSnapshot shape.

**Mapping from kb-20 to ClinicalSnapshot:**
- `EGFR` → kb-20 patient_strata.egfr_value
- `DBI` → kb-20 patient_strata.dbi_value
- `ACB` → kb-20 patient_strata.acb_score
- `CFS` → kb-20 patient_strata.cfs_score
- `CareIntensity` → kb-20 patient_strata.care_intensity
- `RecentFall72h` / `RecentAdmission72h` → kb-20 events table with bounded-window query (last 72h)
- `FamilyDistress` / `CapacityLapse` / `FrailtyStepIncrease30d` / `RestrictivePracticeActive` → kb-20 events / consent state machine (Plan 0.2) joins

- [ ] **Step 1: Test fixture using VAIDSHALA_TEST_DSN; skip otherwise**

```go
package postgres

import (
    "context"
    "database/sql"
    "os"
    "testing"

    _ "github.com/lib/pq"

    kb32ctx "github.com/cardiofit/kb32/internal/context"
)

func TestPostgresSubstrateClient_RoundTrip(t *testing.T) {
    dsn := os.Getenv("VAIDSHALA_TEST_DSN")
    if dsn == "" {
        t.Skip("VAIDSHALA_TEST_DSN not set; skipping Postgres integration test")
    }
    db, err := sql.Open("postgres", dsn)
    if err != nil { t.Fatal(err) }
    defer db.Close()
    
    client := NewPostgresSubstrateClient(db)
    // Seed a kb-20 patient_strata row, then query.
    // Assert: ClinicalSnapshot fields populated correctly.
    _ = client
    _ = context.Background()
}
```

- [ ] **Step 2-5: Implement, test, commit**

```go
type PostgresSubstrateClient struct{ db *sql.DB }

func NewPostgresSubstrateClient(db *sql.DB) *PostgresSubstrateClient {
    return &PostgresSubstrateClient{db: db}
}

// SnapshotFor satisfies kb32ctx.SubstrateClient.
func (p *PostgresSubstrateClient) SnapshotFor(ctx context.Context, residentID uuid.UUID) (kb32ctx.ClinicalSnapshot, error) {
    // Single LEFT JOIN query against kb-20 patient_strata + recent events.
    // Returns a fully-populated ClinicalSnapshot; missing data → zero values.
    var snap kb32ctx.ClinicalSnapshot
    snap.ResidentID = residentID
    
    err := p.db.QueryRowContext(ctx, `
        SELECT egfr_value, dbi_value, acb_score, cfs_score, care_intensity, assessed_at
        FROM patient_strata
        WHERE resident_id = $1
        ORDER BY assessed_at DESC LIMIT 1
    `, residentID).Scan(&snap.EGFR, &snap.DBI, &snap.ACB, &snap.CFS, &snap.CareIntensity, &snap.AssessedAt)
    if err != nil && err != sql.ErrNoRows { return snap, err }
    
    // Recent fall (72h)
    err = p.db.QueryRowContext(ctx, `
        SELECT EXISTS(SELECT 1 FROM resident_events
                      WHERE resident_id = $1 AND event_type = 'fall' AND occurred_at >= NOW() - INTERVAL '72 hours')
    `, residentID).Scan(&snap.RecentFall72h)
    if err != nil { return snap, err }
    
    // Same shape for: RecentAdmission72h, FamilyDistress, CapacityLapse, FrailtyStepIncrease30d, RestrictivePracticeActive
    // ... (additional queries here) ...
    
    return snap, nil
}

var _ kb32ctx.SubstrateClient = (*PostgresSubstrateClient)(nil)
```

- [ ] **Step 6: Commit**

```bash
git commit -m "feat(kb-32): PostgresSubstrateClient over kb-20 patient strata + events"
```

---

### Task 2: Substrate-backed AppropriatenessSource

**Files:**
- Create: `internal/appropriateness/substrate_scorer.go` + `_test.go`

Replace `DefaultAppropriatenessSource` (which returns 3 across all 5 dimensions, always passing the gate) with a real scorer driven by `ClinicalSnapshot` + `Recommendation` content.

**Five-dimension scoring rules (per Guidelines Part 9):**

1. **ClinicalWarrant** — does the recommendation address a real clinical issue at this resident's substrate state?
   - Score 5: rule type matches a substrate signal (e.g., STOP for ACB ≥ 3 + cognitive concern)
   - Score 3: rule fires but no direct substrate corroboration
   - Score 1: rule fires against contraindicated state (e.g., STOP psychotropic in newly-admitted resident)

2. **EvidenceSolidity** — strength of evidence anchors
   - Score 5: 2+ AU-jurisdiction anchors, recent (<3 years)
   - Score 3: 1 AU anchor or 2+ international
   - Score 1: no anchors or only retracted/superseded

3. **AlternativesConsidered** — has the rule's underlying CQL evaluated alternatives?
   - Score 5: rule explicitly checked alternative interventions and excluded them
   - Score 3: rule has alternative-check logic but unfilled
   - Score 1: rule has no alternative-consideration metadata

4. **RestraintConsidered** — did the rule run restraint signal detectors?
   - Score 5: restraint signaler ran; output considered in recommendation
   - Score 3: restraint signaler ran but output ignored
   - Score 1: restraint signaler not invoked

5. **GoalsOfCareAlignment** — does the recommendation align with documented care intensity?
   - Score 5: STOP/MONITOR aligned with comfort/palliative; ADD aligned with active
   - Score 3: neutral
   - Score 1: misaligned (e.g., ADD aggressive intervention for end_of_life resident)

- [ ] **Step 1-5: Test, implement, commit**

```go
type SubstrateBackedScorer struct{}

func NewSubstrateBackedScorer() *SubstrateBackedScorer { return &SubstrateBackedScorer{} }

func (s *SubstrateBackedScorer) Assess(
    ctx context.Context,
    pkt *generator.Packet,
    snap kb32ctx.ClinicalSnapshot,
    rule reasoning.ApplicableRule,
) (Assessment, error) {
    return Assessment{
        ClinicalWarrant:        s.scoreClinicalWarrant(rule, snap),
        EvidenceSolidity:       s.scoreEvidenceSolidity(pkt),
        AlternativesConsidered: s.scoreAlternativesConsidered(rule),
        RestraintConsidered:    s.scoreRestraintConsidered(snap),
        GoalsOfCareAlignment:   s.scoreGoalsOfCareAlignment(rule, snap),
    }, nil
}

// Each score* method implements the rules above.
```

Tests for each dimension's score paths (5 × 3 = 15 path tests) + integration test asserting that an end-of-life resident receiving an ADD recommendation scores ≤ 2 on GoalsOfCareAlignment → gate holds.

```bash
git commit -m "feat(kb-32): SubstrateBackedScorer — 5-dimension appropriateness scoring driven by snapshot + rule"
```

---

### Task 3: Postgres-backed citations.Registry

**Files:**
- Create: `internal/store/postgres/citation_registry.go` + `_test.go`

Replace `citations.NewInMemoryRegistry()` placeholder in main.go with a Postgres-backed registry over migration 043's `source_versions` and `recommendation_citations` tables.

- [ ] **Step 1-5: Test (skip without DSN), implement, commit**

```go
type PostgresCitationRegistry struct{ db *sql.DB }

func NewPostgresCitationRegistry(db *sql.DB) *PostgresCitationRegistry {
    return &PostgresCitationRegistry{db: db}
}

// Implements citations.Registry: Register, Amend, Retract, Supersede, ActiveVersion, SaveCitation, GetCitation
// Each method is a transaction over migration 043's tables.
```

```bash
git commit -m "feat(kb-32): PostgresCitationRegistry implementing citations.Registry over migration 043"
```

---

### Task 4: Stage 7 EvidenceTrace emission

**Files:**
- Create: `internal/lifecycle/evidence_trace.go` + `_test.go`
- Modify: `internal/api/pipeline.go` to call EvidenceTrace emission after a successful run

When a recommendation transitions `detected → drafted`, emit an EvidenceTrace entry recording: rule fired, ClinicalSnapshot at fire time, Assessment scores, framing.ContentHash, citation pin set, urgency tag. This is the audit-defensibility ledger entry for every recommendation.

- [ ] **Step 1-5: Implement EvidenceTraceEmitter interface; wire into pipeline; test**

```go
type EvidenceTraceEmitter interface {
    EmitDraftedTransition(ctx context.Context, entry DraftedTransitionEntry) error
}

type DraftedTransitionEntry struct {
    RecommendationID  uuid.UUID
    AuthorID          uuid.UUID
    RuleID            string
    ContentHash       string
    Assessment        appropriateness.Assessment
    Citations         []citations.RecommendationCitation
    Urgency           string
    FiredAt           time.Time
}

// In pipeline.Run(), after successful citation pinning + formatter validation:
if p.evidenceTracer != nil {
    if err := p.evidenceTracer.EmitDraftedTransition(ctx, entry); err != nil {
        // Log but don't fail — the recommendation drafted; audit emission is best-effort
        // OR fail hard, depending on Guidelines §4 Stage 7 strictness. Pick fail-hard:
        return nil, fmt.Errorf("pipeline stage7 (evidence trace): %w", err)
    }
}
```

Production implementation: writes to Phase 1c's EthicsLog (severity 1, EntryType decision) AND a separate Postgres `evidence_trace_entries` table (new migration 045 in this plan).

```bash
git commit -m "feat(kb-32): Stage 7 EvidenceTrace emission on detected→drafted transition"
```

---

### Task 5: Override taxonomy vocabulary alignment

**Files:**
- Modify: `internal/overrides/taxonomy.go`
- Modify: `migrations/042_recommendation_override_reasons.sql` (rename or add column)
- Modify: `internal/overrides/taxonomy_test.go`

The 2026-05-09 audit flagged that the implementation uses verbose snake_case codes (`alert_fatigue`, `patient_preference`) while Guidelines Part 5 specifies 3-letter taxonomy codes (`WMP`, `NCS`, `BOR`, etc.). The plan diverged from the Guidelines.

**Decision required from clinical informatics lead** before implementing:
- **Option A:** keep snake_case (current state); update Guidelines Part 5 to match (Guidelines doc rev)
- **Option B:** migrate to 3-letter codes; add a translation table and update the 20-code constants
- **Option C:** dual-vocabulary — store both; expose both via API; let regulator/dashboard pick

**Recommended:** Option B — Guidelines Part 5 codifies the published audit-trail vocabulary; the implementation should match. Translation map:
- alert_fatigue → ALF
- irrelevant_to_patient → IRP
- patient_preference → PPF
- clinical_judgment → CJG
- alternative_pursued → AAP
- monitoring_in_place → MIP
- low_priority → LPR
- documentation_concern → DCN
- uncertain_evidence → UNE
- system_error → SYS
- workflow_constraint → WFC
- duplicative_alert → DPA
- goals_of_care_aligned → GCA
- deprescribing_underway → DUW
- frailty_consideration → FRC
- family_consensus_pending → FCP
- sdm_review_required → SDR
- trial_period_active → TPA
- audit_visit_imminent → AVI
- cross_resident_pattern → CRP

(Codes above are illustrative — clinical lead should confirm the canonical mapping.)

- [ ] **Step 1: Confirm vocabulary with clinical informatics lead** — DO NOT proceed without sign-off

- [ ] **Step 2-5: Migration adding new code column; backfill; tests; commit**

```bash
git commit -m "feat(kb-32): align override-reason codes with Guidelines Part 5 3-letter taxonomy"
```

---

### Task 6: Prescriber framing opt-out HTTP endpoint

**Files:**
- Create: `internal/api/optout_handlers.go` + `_test.go`
- Modify: `cmd/server/main.go` to mount the new endpoint
- Modify: `internal/framing/per_gp_observer.go` to extend `ObservationSource.RecordOptOut` if not present

`POST /v1/framing/optout/{gp_id}` — registers a prescriber's opt-out from framing learning. `DELETE /v1/framing/optout/{gp_id}` — revokes. Both behind permissions middleware.

- [ ] **Step 1-5: Test, implement, commit**

```go
type OptOutHandler struct{ store framing.OptOutStore }

func (h *OptOutHandler) HandleRegister(c *gin.Context) {
    gpID, err := uuid.Parse(c.Param("gp_id"))
    if err != nil { /* 400 */ return }
    var req struct { Reason string `json:"reason"` }
    _ = c.ShouldBindJSON(&req)
    err = h.store.RegisterOptOut(c.Request.Context(), gpID, req.Reason)
    // 201 + record
}

func (h *OptOutHandler) HandleRevoke(c *gin.Context) {
    gpID, err := uuid.Parse(c.Param("gp_id"))
    // ...
    err = h.store.RevokeOptOut(c.Request.Context(), gpID)
    // 204
}
```

Tests: register happy path; revoke happy path; bad UUID → 400; missing reason allowed (optional); register-twice idempotent.

```bash
git commit -m "feat(kb-32): POST/DELETE /v1/framing/optout endpoint for prescriber opt-out"
```

---

### Task 7: Mount PDP middleware on /v1/craft/draft and /v1/craft/override

**Files:**
- Modify: `cmd/server/main.go` to wrap craft routes with permissions middleware
- Add config flag `KB32_PERMISSIONS_ENFORCED` (default false; matching Phase 1b-completion's kb-30 pattern)

The craft and override endpoints are currently unwrapped per Phase 2a/2b's documented deferral. Now mount the Phase 1a `permissions.Middleware` over both, gated by `KB32_PERMISSIONS_ENFORCED=true`.

- [ ] **Step 1-5: Wire middleware + integration test + commit**

```bash
git commit -m "feat(kb-32): mount Phase 1a permissions middleware on craft/override routes"
```

---

### Task 8: End-to-end integration test with all-Postgres-backed dependencies

**Files:**
- Create: `tests/integration/end_to_end_with_real_stores_test.go`

A new integration test that wires every Phase 2-completion component (Postgres SubstrateClient + SubstrateBackedScorer + Postgres CitationRegistry + EvidenceTraceEmitter + permissions middleware) and exercises the Sunday-night-fall scenario end-to-end against a real Postgres instance.

Skips when `VAIDSHALA_TEST_DSN` is unset.

- [ ] **Step 1-5: Test, run against local Postgres, commit**

```bash
git commit -m "test(kb-32): end-to-end pipeline with all-Postgres-backed dependencies"
```

---

## Spec coverage

- [x] Postgres-backed SubstrateClient (closes blocker #1) — Task 1
- [x] SubstrateBackedScorer (closes blocker #2 — replaces DefaultAppropriatenessSource) — Task 2
- [x] Postgres-backed citations.Registry (closes blocker #3 dependency) — Task 3
- [x] Stage 7 EvidenceTrace emission (closes blocker #3 — full audit trail) — Task 4
- [x] Override taxonomy vocabulary alignment with Guidelines Part 5 — Task 5
- [x] Prescriber framing opt-out HTTP endpoint — Task 6
- [x] PDP middleware mounted on craft/override routes — Task 7
- [x] End-to-end integration test with all-Postgres-backed deps — Task 8

**Out of scope (still deferred):**
- Layer 4 surfaces UX (worklist UI, GP communication hub)
- Pharmacy-employer view design (uses citations + override patterns)
- Regulator audit interface (consumes EvidenceTrace via gRPC)
- gRPC RecommendationCraftService surface — REST sufficient for pilot
- Pharmacist Advisory Group constitution (operational/process)

**Pre-pilot acceptance gate (per 2026-05-09 audit):**
1. ✅ Tasks 1–4 complete (3 production blockers resolved)
2. ✅ Task 5 complete with clinical informatics sign-off on canonical vocabulary
3. ✅ Task 7 complete with `KB32_PERMISSIONS_ENFORCED=true` confirmed in production deploy
4. ✅ Task 8 passes against a live staging Postgres

Plan complete and saved.
