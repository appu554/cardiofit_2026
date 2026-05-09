# Phase 2b — Clinical Safety + Audit Moat Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Land the four guideline-distinctive items the *Recommendation Craft Implementation Guidelines v1.0* identifies as the platform's audit moat: (1) override-reason taxonomy with 20 categories + appropriateness pairing + rule-tuning feedback loop; (2) citation versioning with effective-date semantics + supersession workflow; (3) negative-evidence patterns with CQL absence queries + evidence_checks materialised views; (4) restraint signals with 9 signal detectors surfaced inline with action. Plus the Stage-5-of-5-test-categories expansion: clinical safety tests, metric integrity tests, performance tests beyond Phase 2a's integration test.

**Phase 2a / 2b split:**
This plan extends Phase 2a (`2026-05-09-phase-2a-craft-engine-scaffold.md`). 2a must complete cleanly before 2b begins. 2b adds new internal packages (`overrides/`, `citations/`, `negative_evidence/`, `restraint/`) plus expanded test coverage to the kb-32 service.

**Architecture:** Extends kb-32 from Phase 2a. Four new sub-packages, three new migrations, expanded test suite. The override taxonomy creates the rule-tuning feedback loop (overrides → appropriateness pairing → rule curator review → rule tuning) and integrates with Phase 1c's EthicsLog (`pattern_detected` entries when override patterns exceed thresholds). Citation versioning adds an `effective_at` snapshot at recommendation fire time so source amendments don't retroactively invalidate already-fired recommendations. Negative-evidence makes deprescribing recommendations defensible against a regulator who asks "what evidence does an absent observation support?". Restraint signals surface stop-the-line context inline with action, fulfilling Principle 4.

**Tech Stack:** Go, Gin, Postgres. Depends on Phase 2a (kb-32 scaffold), Plan 0.1 (Recommendation entity), Plan 0.5 (kb-cql-runtime for absence queries), Phase 1c (EthicsLog for feedback loop).

---

## File Structure

**New files in kb-32 (extending Phase 2a):**

- `internal/overrides/taxonomy.go` + `_test.go` — 20-category enum, OverrideReason struct, validation
- `internal/overrides/store.go` + `_test.go` — Postgres + InMemory store
- `internal/overrides/feedback_loop.go` + `_test.go` — patterns → EthicsLog
- `internal/citations/source_registry.go` + `_test.go`
- `internal/citations/versioning.go` + `_test.go` — amendment / retraction / supersession workflows
- `internal/citations/snapshot.go` + `_test.go` — pin citation at fire time
- `internal/negative_evidence/absence_patterns.go` + `_test.go` — three CQL query patterns
- `internal/negative_evidence/evidence_checks.go` + `_test.go` — table reads + view refresh
- `internal/restraint/signaler.go` + `_test.go` — 9 signal detectors
- `internal/restraint/surfacing.go` + `_test.go` — inline-with-action data shaping
- `internal/api/override_handlers.go` + `_test.go` — POST /craft/override capture endpoint

**New migrations (sequential after Phase 2a's 040/041):**
- `migrations/042_recommendation_override_reasons.sql` + rollback (override taxonomy + materialised view `rule_override_patterns`)
- `migrations/043_source_versions.sql` + rollback (`source_versions` + `recommendation_citations`)
- `migrations/044_evidence_checks.sql` + rollback (negative-evidence materialised views)

**Test suite expansion:**
- `tests/clinical_safety/appropriateness_blocking_test.go`
- `tests/clinical_safety/negative_evidence_completeness_test.go`
- `tests/clinical_safety/citation_versioning_correctness_test.go`
- `tests/metric_integrity/suppression_detector_test.go`
- `tests/metric_integrity/appropriateness_pairing_test.go`
- `tests/perf/per_stage_latency_test.go`

---

### Task 1: Override-reason taxonomy

**Files:**
- Create: `internal/overrides/taxonomy.go` + `_test.go`

20 codes per Guidelines Part 5 — 12 Wright/McCoy foundation + 8 ACOP extensions. Reasons are paired with an `AppropriatenessFlag` (`appropriate_override` / `inappropriate_override` / `mixed`) so the rule-tuning loop can distinguish "rule fires too eagerly" from "rule fires correctly but pharmacist overrode for legitimate clinical reason."

- [ ] **Step 1: Failing test**

```go
package overrides

import (
    "errors"
    "testing"
)

func TestIsValidReasonCode_All20(t *testing.T) {
    for _, c := range allReasonCodes() {
        if !IsValidReasonCode(c) {
            t.Errorf("%q should be valid", c)
        }
    }
    if IsValidReasonCode("not_a_code") {
        t.Errorf("garbage should not be valid")
    }
}

func TestOverrideReason_Validate(t *testing.T) {
    o := OverrideReason{ReasonCode: "patient_preference", AppropriatenessFlag: "appropriate_override", Reasoning: "Patient declined"}
    if err := o.Validate(); err != nil { t.Errorf("valid override rejected: %v", err) }

    bad := OverrideReason{ReasonCode: "garbage"}
    if !errors.Is(bad.Validate(), ErrInvalidReasonCode) {
        t.Errorf("garbage code should fail")
    }
}

func allReasonCodes() []string { return ValidReasonCodes }
```

- [ ] **Step 2-3: Implement**

```go
package overrides

import (
    "errors"
    "time"

    "github.com/google/uuid"
)

// 20 reason codes per Recommendation Craft Guidelines Part 5.
// First 12 are Wright/McCoy foundation; last 8 are ACOP-specific extensions.
var ValidReasonCodes = []string{
    // Wright/McCoy 12-category foundation
    "alert_fatigue", "irrelevant_to_patient", "patient_preference", "clinical_judgment",
    "alternative_pursued", "monitoring_in_place", "low_priority", "documentation_concern",
    "uncertain_evidence", "system_error", "workflow_constraint", "duplicative_alert",
    // ACOP 8-category extensions
    "goals_of_care_aligned", "deprescribing_underway", "frailty_consideration",
    "family_consensus_pending", "sdm_review_required", "trial_period_active",
    "audit_visit_imminent", "cross_resident_pattern",
}

const (
    FlagAppropriateOverride  = "appropriate_override"
    FlagInappropriateOverride = "inappropriate_override"
    FlagMixed                = "mixed"
)

var validFlags = map[string]bool{
    FlagAppropriateOverride: true, FlagInappropriateOverride: true, FlagMixed: true,
}

type OverrideReason struct {
    ID                 uuid.UUID
    RecommendationID   uuid.UUID
    ReasonCode         string
    AppropriatenessFlag string
    Reasoning          string
    CapturedAt         time.Time
    CapturedBy         uuid.UUID
}

var (
    ErrInvalidReasonCode    = errors.New("overrides: invalid reason code")
    ErrInvalidFlag          = errors.New("overrides: invalid appropriateness flag")
    ErrEmptyReasoning       = errors.New("overrides: reasoning required")
)

func IsValidReasonCode(s string) bool {
    for _, c := range ValidReasonCodes {
        if c == s {
            return true
        }
    }
    return false
}

func IsValidFlag(s string) bool { return validFlags[s] }

func (o OverrideReason) Validate() error {
    if !IsValidReasonCode(o.ReasonCode) {
        return ErrInvalidReasonCode
    }
    if !IsValidFlag(o.AppropriatenessFlag) {
        return ErrInvalidFlag
    }
    if o.Reasoning == "" {
        return ErrEmptyReasoning
    }
    return nil
}
```

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(kb-32): override-reason taxonomy with 20 codes + appropriateness pairing"
```

---

### Task 2: Override store + Migration 042

**Files:**
- Create: `internal/overrides/store.go` + `_test.go`
- Create: `migrations/042_recommendation_override_reasons.sql` + rollback

Postgres + InMemory stores. `recommendation_override_reasons` table + materialised view `rule_override_patterns` aggregating overrides per rule per reason per quarter.

- [ ] **Step 1-3: Test + implement**

Store interface mirrors Phase 1a Task 3 conventions:

```go
type Store interface {
    Create(ctx context.Context, o OverrideReason) (OverrideReason, error)
    Get(ctx context.Context, id uuid.UUID) (*OverrideReason, error)
    ListByRule(ctx context.Context, ruleID string) ([]OverrideReason, error)
    PatternSummary(ctx context.Context, ruleID string, since time.Time) (map[string]int, error)
}
```

InMemory uses sync.RWMutex per Phase 1 convention.

- [ ] **Step 4: Migration 042**

```sql
-- 042_recommendation_override_reasons.sql
BEGIN;
CREATE TABLE recommendation_override_reasons (
    id                   UUID PRIMARY KEY,
    recommendation_id    UUID NOT NULL,
    reason_code          TEXT NOT NULL CHECK (reason_code IN (
        'alert_fatigue','irrelevant_to_patient','patient_preference','clinical_judgment',
        'alternative_pursued','monitoring_in_place','low_priority','documentation_concern',
        'uncertain_evidence','system_error','workflow_constraint','duplicative_alert',
        'goals_of_care_aligned','deprescribing_underway','frailty_consideration',
        'family_consensus_pending','sdm_review_required','trial_period_active',
        'audit_visit_imminent','cross_resident_pattern')),
    appropriateness_flag TEXT NOT NULL CHECK (appropriateness_flag IN
        ('appropriate_override','inappropriate_override','mixed')),
    reasoning            TEXT NOT NULL,
    captured_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    captured_by          UUID NOT NULL
);
CREATE INDEX idx_ror_recommendation ON recommendation_override_reasons (recommendation_id);
CREATE INDEX idx_ror_captured ON recommendation_override_reasons (captured_at DESC);

-- Materialised view for the rule-tuning feedback loop.
CREATE MATERIALIZED VIEW rule_override_patterns AS
SELECT
    -- recommendation_id resolves to rule_id via the recommendations table; cross-package join.
    -- The materialised view assumes a recommendations(id, rule_id) shape from Plan 0.1.
    r.rule_id,
    o.reason_code,
    o.appropriateness_flag,
    DATE_TRUNC('quarter', o.captured_at) AS quarter,
    COUNT(*)                              AS override_count
FROM recommendation_override_reasons o
JOIN recommendations r ON r.id = o.recommendation_id
GROUP BY r.rule_id, o.reason_code, o.appropriateness_flag, DATE_TRUNC('quarter', o.captured_at);

CREATE UNIQUE INDEX idx_rop_rule_reason_quarter
    ON rule_override_patterns (rule_id, reason_code, appropriateness_flag, quarter);

COMMIT;
```

- [ ] **Step 5: Commit**

```bash
git commit -m "feat(kb-32): override store + migration 042 with rule_override_patterns matview"
```

---

### Task 3: Override capture HTTP API

**Files:**
- Create: `internal/api/override_handlers.go` + `_test.go`

`POST /craft/override/{recommendation_id}` endpoint, behind permissions middleware (PDP class). Captures reason + appropriateness flag + free-text reasoning.

- [ ] **Step 1-3: Test + implement**

```go
type CaptureRequest struct {
    ReasonCode          string `json:"reason_code"`
    AppropriatenessFlag string `json:"appropriateness_flag"`
    Reasoning           string `json:"reasoning"`
}

func HandleCaptureOverride(store overrides.Store) gin.HandlerFunc {
    return func(c *gin.Context) {
        recID, err := uuid.Parse(c.Param("recommendation_id"))
        if err != nil { c.JSON(400, gin.H{"error":"bad_id"}); return }
        var req CaptureRequest
        if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error":"bad_request"}); return }
        viewerID := uuidFromContext(c.Request.Context())
        o := overrides.OverrideReason{
            RecommendationID: recID, ReasonCode: req.ReasonCode,
            AppropriatenessFlag: req.AppropriatenessFlag,
            Reasoning: req.Reasoning, CapturedBy: viewerID,
        }
        if err := o.Validate(); err != nil { c.JSON(422, gin.H{"error": err.Error()}); return }
        saved, err := store.Create(c.Request.Context(), o)
        if err != nil { c.JSON(500, gin.H{"error":"store_failed"}); return }
        c.JSON(201, saved)
    }
}
```

Tests: validation rejection (invalid code → 422), happy path (201 + persisted record), missing reasoning rejected.

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(kb-32): POST /craft/override capture endpoint"
```

---

### Task 4: Override → rule tuning feedback loop

**Files:**
- Create: `internal/overrides/feedback_loop.go` + `_test.go`

Aggregates patterns from `rule_override_patterns` weekly. When a rule's override count exceeds a threshold and most are flagged `inappropriate_override`, emits an `EthicsLog` entry (Phase 1c) of type `pattern_detected` severity 3.

- [ ] **Step 1-3: Test + implement**

```go
package overrides

import (
    "context"
    "fmt"

    "github.com/cardiofit/shared/v2_substrate/ethics/ethics_log"
)

const (
    OverrideThreshold       = 30
    InappropriateRatioFloor = 0.6
)

type Detector struct {
    store  Store
    logger ethics_log.Logger
}

func NewDetector(s Store, l ethics_log.Logger) *Detector {
    return &Detector{store: s, logger: l}
}

// Scan walks recent override patterns; flags rules where overrides ≥ threshold
// AND inappropriate_override fraction ≥ InappropriateRatioFloor → ethics_log entry.
func (d *Detector) Scan(ctx context.Context, ruleID string, since time.Time) error {
    summary, err := d.store.PatternSummary(ctx, ruleID, since)
    if err != nil {
        return err
    }
    total := 0
    for _, n := range summary {
        total += n
    }
    if total < OverrideThreshold {
        return nil
    }
    inappropriate := summary["inappropriate_override"]
    if float64(inappropriate)/float64(total) < InappropriateRatioFloor {
        return nil
    }
    return d.logger.Append(ctx, ethics_log.Entry{
        EntryType:   ethics_log.EntryTypePatternDetected,
        Severity:    3,
        Description: fmt.Sprintf("rule %s: %d overrides (%d inappropriate)", ruleID, total, inappropriate),
        Status:      ethics_log.StatusOpen,
    })
}
```

Tests: high-override + inappropriate-majority → log entry; below-threshold → no log; appropriate-majority → no log.

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(kb-32): override→rule-tuning feedback loop emits EthicsLog on inappropriate-override patterns"
```

---

### Task 5: Citation source registry + versioning + Migration 043

**Files:**
- Create: `internal/citations/source_registry.go` + `_test.go`
- Create: `internal/citations/versioning.go` + `_test.go`
- Create: `migrations/043_source_versions.sql` + rollback

Per Guidelines Part 6. Three workflows: amendment (new version, old still valid), retraction (mark all citations stale), supersession (new replaces old; ongoing recommendations re-cited).

- [ ] **Step 1-3: Test + implement**

```go
type SourceVersion struct {
    SourceID      string
    Version       string
    EffectiveFrom time.Time
    EffectiveTo   *time.Time  // nil = currently active
    ContentHash   string
    Status        string  // "active" | "amended" | "retracted" | "superseded"
}

type RecommendationCitation struct {
    RecommendationID uuid.UUID
    SourceID         string
    Version          string
    PinnedAt         time.Time  // == recommendation fire time
}

// Amend creates a new version; old continues to be valid for already-fired citations.
func (r *Registry) Amend(ctx context.Context, sourceID, newVersion, hash string) error { /* ... */ }

// Retract marks all citations of the source as stale. Already-fired recommendations
// must be re-evaluated; ongoing surfacing in dashboards must show retracted-source flag.
func (r *Registry) Retract(ctx context.Context, sourceID string, reason string) error { /* ... */ }

// Supersede replaces the source with a new SourceID; ongoing recommendations are re-cited.
func (r *Registry) Supersede(ctx context.Context, oldSourceID, newSourceID string) error { /* ... */ }
```

```sql
-- 043_source_versions.sql
BEGIN;
CREATE TABLE source_versions (
    source_id      TEXT NOT NULL,
    version        TEXT NOT NULL,
    effective_from TIMESTAMPTZ NOT NULL,
    effective_to   TIMESTAMPTZ,
    content_hash   TEXT NOT NULL,
    status         TEXT NOT NULL CHECK (status IN ('active','amended','retracted','superseded')),
    PRIMARY KEY (source_id, version)
);

CREATE TABLE recommendation_citations (
    recommendation_id UUID NOT NULL,
    source_id         TEXT NOT NULL,
    version           TEXT NOT NULL,
    pinned_at         TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (recommendation_id, source_id, version),
    FOREIGN KEY (source_id, version) REFERENCES source_versions (source_id, version)
);

CREATE INDEX idx_rc_recommendation ON recommendation_citations (recommendation_id);
CREATE INDEX idx_rc_source ON recommendation_citations (source_id, version);
COMMIT;
```

Tests for each workflow: amendment preserves old citations, retraction marks-stale, supersession re-cites ongoing.

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(kb-32): citation registry + versioning workflows + migration 043"
```

---

### Task 6: Citation snapshot at fire time

**Files:**
- Create: `internal/citations/snapshot.go` + `_test.go`

When a recommendation enters `drafted`, the citation is pinned with the current source version + `effective_at` = fire time. Source amendments after fire time do NOT invalidate the recommendation.

- [ ] **Step 1-3: Test + implement**

```go
// PinAtFireTime locks each evidence anchor to its current active version.
// Returns the citation records to be persisted alongside the recommendation.
func PinAtFireTime(ctx context.Context, registry *Registry, recID uuid.UUID, anchors []string, asOf time.Time) ([]RecommendationCitation, error) {
    out := make([]RecommendationCitation, 0, len(anchors))
    for _, sourceID := range anchors {
        v, err := registry.ActiveVersion(ctx, sourceID, asOf)
        if err != nil { return nil, err }
        out = append(out, RecommendationCitation{
            RecommendationID: recID, SourceID: sourceID, Version: v.Version, PinnedAt: asOf,
        })
    }
    return out, nil
}
```

Tests: pin then amend source — old citation still resolves to old version; pin then retract source — citation flagged stale on read; supersession workflow re-cites.

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(kb-32): citation snapshot at fire time with effective-date semantics"
```

---

### Task 7: Negative-evidence absence patterns + Migration 044

**Files:**
- Create: `internal/negative_evidence/absence_patterns.go` + `_test.go`
- Create: `migrations/044_evidence_checks.sql` + rollback

Per Guidelines Part 7. Three CQL query patterns:
1. **Bounded-window absence** — "no fall in past 90 days"
2. **Periodic-review absence** — "no medication review in past 12 months"
3. **Indication-documentation absence** — "no documented indication for PPI"

The `evidence_checks` table caches absence-query results; pre-computed materialised views per pattern. Performance budget: 2000ms p95.

- [ ] **Step 1-3: Test + implement**

```go
type AbsencePattern int

const (
    PatternBoundedWindow AbsencePattern = iota
    PatternPeriodicReview
    PatternIndicationDocumentation
)

type AbsenceQuery struct {
    Pattern        AbsencePattern
    ResidentID     uuid.UUID
    ObservationKind string
    WindowDays     int  // for bounded-window pattern
}

type AbsenceResult struct {
    Confirmed     bool      // true == observation is confirmed absent in window
    LastSeenAt    *time.Time
    QueriedAt     time.Time
    EvidenceText  string    // human-readable defensibility statement
}

type Querier interface {
    QueryAbsence(ctx context.Context, q AbsenceQuery) (AbsenceResult, error)
}
```

```sql
-- 044_evidence_checks.sql
BEGIN;
CREATE TABLE evidence_checks (
    id              UUID PRIMARY KEY,
    pattern         TEXT NOT NULL CHECK (pattern IN
                       ('bounded_window','periodic_review','indication_documentation')),
    resident_id     UUID NOT NULL,
    observation_kind TEXT NOT NULL,
    window_days     INT,
    confirmed       BOOLEAN NOT NULL,
    last_seen_at    TIMESTAMPTZ,
    queried_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    evidence_text   TEXT NOT NULL
);
CREATE INDEX idx_ec_resident_pattern ON evidence_checks (resident_id, pattern, observation_kind);
CREATE INDEX idx_ec_queried ON evidence_checks (queried_at DESC);
COMMIT;
```

Tests for each of the three patterns: confirmed absence path, presence-detected (negates absence claim), error propagation.

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(kb-32): negative-evidence absence patterns + migration 044"
```

---

### Task 8: Negative-evidence integration with deprescribing

**Files:**
- Create: `internal/negative_evidence/evidence_checks.go` + `_test.go`
- Modify: `internal/generator/recommendation.go` (Phase 2a Task 6) to pull negative-evidence anchors when `RecommendationType == STOP`

Wires absence-pattern queries into deprescribing recommendation generation.

- [ ] **Step 1-3: Test + implement**

```go
// AttachNegativeEvidence augments a deprescribing recommendation packet with
// absence-pattern findings. Called from generator.Generate when Type == STOP.
func AttachNegativeEvidence(ctx context.Context, q Querier, packet *generator.Packet, residentID uuid.UUID) error {
    if packet.Type != "STOP" {
        return nil
    }
    // For each STOP rule, run its associated absence patterns.
    res, err := q.QueryAbsence(ctx, AbsenceQuery{
        Pattern: PatternBoundedWindow,
        ResidentID: residentID,
        ObservationKind: "fall",
        WindowDays: 90,
    })
    if err != nil { return err }
    if res.Confirmed {
        packet.Sections["evidence"] += "\n\nNegative evidence: " + res.EvidenceText
    }
    return nil
}
```

Tests: STOP recommendation gets negative-evidence appended; non-STOP unchanged; querier error propagates.

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(kb-32): wire negative-evidence into deprescribing recommendation generation"
```

---

### Task 9: Restraint signals (9 detectors)

**Files:**
- Create: `internal/restraint/signaler.go` + `_test.go`
- Create: `internal/restraint/surfacing.go` + `_test.go`

Per Guidelines Part 10. Nine substrate signal types, each with Red or Amber severity.

- [ ] **Step 1: Failing test enumerating the 9 detectors**

```go
var expectedSignalTypes = []string{
    "recent_fall_72h",
    "acb_increase",
    "family_distress",
    "end_of_life_proximity",
    "capacity_lapse",
    "polypharmacy_threshold",
    "frailty_step_change",
    "recent_admission_72h",
    "restrictive_practice_active",
}

func TestAllSignalDetectorsRegistered(t *testing.T) {
    for _, name := range expectedSignalTypes {
        if _, ok := detectorsByName[name]; !ok {
            t.Errorf("detector %q is not registered", name)
        }
    }
}
```

- [ ] **Step 2-3: Implement**

```go
package restraint

import "github.com/cardiofit/kb32/internal/context"

type Severity string

const (
    SeverityRed   Severity = "red"
    SeverityAmber Severity = "amber"
)

type Signal struct {
    Type        string
    Severity    Severity
    Reasoning   string
    SuggestedPause string  // human-readable text suggesting clinical pause
}

type Detector func(snap context.ClinicalSnapshot) *Signal

var detectorsByName = map[string]Detector{
    "recent_fall_72h": func(s context.ClinicalSnapshot) *Signal {
        if !s.RecentFall72h { return nil }
        return &Signal{Type: "recent_fall_72h", Severity: SeverityRed,
            Reasoning: "Resident fell within 72h", SuggestedPause: "Review fall mechanism before changing meds"}
    },
    "recent_admission_72h": func(s context.ClinicalSnapshot) *Signal {
        if !s.RecentAdmission72h { return nil }
        return &Signal{Type: "recent_admission_72h", Severity: SeverityRed, /* ... */}
    },
    "acb_increase": func(s context.ClinicalSnapshot) *Signal {
        if s.ACB < 3 { return nil }
        return &Signal{Type: "acb_increase", Severity: SeverityAmber, /* ... */}
    },
    "polypharmacy_threshold": func(s context.ClinicalSnapshot) *Signal { /* DBI ≥ 1.0 */ return nil },
    "frailty_step_change":    func(s context.ClinicalSnapshot) *Signal { /* CFS step ≥ 2 in 30d */ return nil },
    "end_of_life_proximity":  func(s context.ClinicalSnapshot) *Signal {
        if s.CareIntensity == "end_of_life" {
            return &Signal{Type: "end_of_life_proximity", Severity: SeverityRed, /* ... */}
        }
        return nil
    },
    "capacity_lapse":              func(s context.ClinicalSnapshot) *Signal { /* substrate-driven */ return nil },
    "family_distress":             func(s context.ClinicalSnapshot) *Signal { /* substrate-driven */ return nil },
    "restrictive_practice_active": func(s context.ClinicalSnapshot) *Signal { /* substrate-driven */ return nil },
}

// DetectAll runs every registered detector and returns all signals that fired.
func DetectAll(snap context.ClinicalSnapshot) []Signal {
    out := []Signal{}
    for _, d := range detectorsByName {
        if s := d(snap); s != nil {
            out = append(out, *s)
        }
    }
    return out
}
```

Tests: each of the 9 detector trigger paths; composite (multiple fire); none-firing (Green substrate state returns empty).

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(kb-32): 9 restraint signal detectors with Red/Amber severity (Guidelines Part 10)"
```

---

### Task 10: Test suite expansion (5 categories)

**Files:**
- Create: `tests/clinical_safety/appropriateness_blocking_test.go`
- Create: `tests/clinical_safety/negative_evidence_completeness_test.go`
- Create: `tests/clinical_safety/citation_versioning_correctness_test.go`
- Create: `tests/metric_integrity/suppression_detector_test.go`
- Create: `tests/metric_integrity/appropriateness_pairing_test.go`
- Create: `tests/perf/per_stage_latency_test.go`

Stand up the four missing test categories from Guidelines Part 13.

**Clinical safety (3 tests):**

1. `appropriateness_blocking_test.go` — synthetic recommendation with low-score on each of the 5 dimensions; assert each fails the gate. All-pass case advances to drafted.

2. `negative_evidence_completeness_test.go` — for every STOP rule that fires, the resulting recommendation MUST have a negative-evidence anchor attached. Walk all sample STOP recommendations; assert anchor presence.

3. `citation_versioning_correctness_test.go` — fire a recommendation citing source S v1; amend S to v2; original recommendation's citation MUST still resolve to S v1 (effective-date semantics).

**Metric integrity (2 tests):**

4. `suppression_detector_test.go` — synthetic data: rule fires 100 times, only 30 reach drafted (70% suppression). Assert the suppression detector flags the rule for review.

5. `appropriateness_pairing_test.go` — synthetic data: rule has 50% acceptance rate but 1.5/5 mean appropriateness score. Assert the appropriateness-pairing metric flags this as a divergence.

**Performance (1 test):**

6. `per_stage_latency_test.go` — for each pipeline stage (1 through 6), measure p95 of 100 runs against in-memory fakes. Assert each is below the Guidelines hard cap:
   - Stage 1 (Context assembler): 100ms
   - Stage 2 (Reasoning chain): 500ms
   - Stage 3 (Generator): 50ms
   - Stage 4 (Appropriateness): 50ms
   - Stage 5 (Framing): 30ms
   - Stage 6 (Formatter): 20ms

- [ ] **Step 1-5: Write all 6 test files; run; commit**

```bash
go test -race ./tests/... -v
git commit -m "test(kb-32): clinical-safety + metric-integrity + perf test categories (Guidelines Part 13)"
```

---

## Spec coverage

- [x] Override-reason taxonomy 20 codes (Guidelines Part 5) — Task 1
- [x] Override store + materialised view (Migration 042) — Task 2
- [x] Override capture HTTP API — Task 3
- [x] Override→rule-tuning feedback loop with EthicsLog integration — Task 4
- [x] Citation source registry + versioning workflows (Guidelines Part 6, Migration 043) — Task 5
- [x] Citation snapshot at fire time with effective-date semantics — Task 6
- [x] Negative-evidence absence patterns (Guidelines Part 7, Migration 044) — Task 7
- [x] Negative-evidence wired into deprescribing generation — Task 8
- [x] Restraint signals 9 detectors (Guidelines Part 10) — Task 9
- [x] Test suite expansion — clinical safety, metric integrity, performance (Guidelines Part 13) — Task 10

**Out of scope (still deferred to future work):**
- Layer 4 surfaces UX (worklist UI rendering, GP communication hub)
- Pharmacy-employer view design (uses citations + override patterns; Phase 1b scaffolds the data layer, UX is downstream)
- Regulator audit interface
- Pharmacist Advisory Group constitution (operational/process, not code)

Plan complete and saved.
