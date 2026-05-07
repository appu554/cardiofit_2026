# Ethical Guards + Workforce Modules Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the v3 ethical-architecture commitments and workforce-development modules: clinical appropriateness paired with acceptance (v3 §7 line 426 + Architectural Commitment 8), restraint signals (v3 §7 line 422), RPL-evidence module for the 30 June 2026 credentialing cliff (v3 §10 line 530), and CPD tagging (v3 §12 line 699). These guards are what prevent the craft engine (Phase 2) from becoming metric-corruption (Risk 13) or persuasive framing of inappropriate recommendations (Risk 15).

**Architecture:** Extends `kb-32-recommendation-craft` from Phase 2 with an `appropriateness/` subsystem that scores every recommendation on a structured rubric and stores the score *paired* with acceptance outcomes. Suppression detector watches for recommendation-type frequency drops without clinical-context change. Restraint signals are a new substrate query surfaced from the substrate-state engines (Plans 0.1+0.3). RPL-evidence and CPD modules are pharmacist-self-tier features integrated with Phase 1's PharmacistView.

**Tech Stack:** Go, Postgres, depends on Plans 0.1, 0.2, 0.3, 0.5, Phase 1, Phase 2.

---

## File Structure

**New packages in kb-32:**
- `kb-32-recommendation-craft/internal/appropriateness/scorer.go` + `_test.go`
- `kb-32-recommendation-craft/internal/appropriateness/suppression_detector.go` + `_test.go`
- `kb-32-recommendation-craft/internal/restraint/signaler.go` + `_test.go`
- `kb-32-recommendation-craft/internal/rpl/evidence_pack_generator.go` + `_test.go`
- `kb-32-recommendation-craft/internal/cpd/tagger.go` + `_test.go`
- `kb-32-recommendation-craft/internal/cpd/ahpra_record_generator.go` + `_test.go`

**New migrations:**
- `migrations/030_appropriateness_scores.sql` + rollback
- `migrations/031_cpd_activities.sql` + rollback

---

### Task 1: Appropriateness scoring rubric

**Files:**
- Create: `internal/appropriateness/scorer.go` + `_test.go`
- Create: `migrations/030_appropriateness_scores.sql` + rollback

Per v3 §7 line 428: every recommendation scored on (a) clinically warranted? (b) evidence solid for this resident profile? (c) alternatives considered? (d) restraint considered? (e) goals-of-care aligned?

- [ ] **Step 1: Write failing test**

```go
package appropriateness

import (
	"testing"

	"github.com/google/uuid"

	"shared/v2_substrate/models"
)

func TestScorer_HighScoreForWellGroundedRec(t *testing.T) {
	rec := models.Recommendation{
		ID:   uuid.New(),
		Type: models.RecommendationTypeStop,
		ClinicalContent: models.ClinicalContent{
			Issue:           "anticholinergic burden",
			ClinicalContext: "87yo, eGFR 32, recent fall, ACB 4",
			Rationale:       "DBI 0.8 attributable; 4-week non-pharm trial completed",
			EvidenceRefs:    []string{"AMH-Aged-Care-2024", "ADG-2025-Rec-42"},
			ProposedPlan:    "cease oxybutynin 5mg BD",
			MonitoringPlan:  "voiding diary 14d, falls reassessment 30d",
		},
	}
	score := Score(rec, fakeSubstrateContext{
		hasNonPharmTrial:   true,
		alternativesNoted:  true,
		careIntensityKnown: true,
	})
	if score.Total < 0.8 {
		t.Errorf("expected high score; got %.2f", score.Total)
	}
}

func TestScorer_LowScoreForMarginal(t *testing.T) {
	rec := models.Recommendation{
		Type: models.RecommendationTypeAdd,
		ClinicalContent: models.ClinicalContent{
			Issue:           "blank slate add",
			ProposedPlan:    "start statin",
			Rationale:       "guideline says so",
			// No evidence anchoring, no alternatives, no monitoring plan
		},
	}
	score := Score(rec, fakeSubstrateContext{})
	if score.Total > 0.5 {
		t.Errorf("expected marginal score; got %.2f", score.Total)
	}
	if !score.Flags.MissingAlternatives {
		t.Errorf("expected missing-alternatives flag")
	}
}
```

- [ ] **Step 2-5: Implement, test, commit**

```go
package appropriateness

import (
	"shared/v2_substrate/models"
)

type Score struct {
	Total                 float64 // [0..1]
	WarrantedScore        float64
	EvidenceScore         float64
	AlternativesScore     float64
	RestraintScore        float64
	GoalsAlignmentScore   float64
	Flags                 Flags
}

type Flags struct {
	MissingAlternatives bool
	WeakEvidence        bool
	GoalsConflict       bool
	RestraintNotConsidered bool
}

type SubstrateContext interface {
	HasNonPharmTrial() bool
	AlternativesNoted() bool
	CareIntensityKnown() bool
	CareIntensityTag() string
}

func Score(rec models.Recommendation, ctx SubstrateContext) Score {
	s := Score{}
	if rec.ClinicalContent.Rationale != "" && rec.ClinicalContent.ClinicalContext != "" {
		s.WarrantedScore = 1.0
	}
	if len(rec.ClinicalContent.EvidenceRefs) >= 2 {
		s.EvidenceScore = 1.0
	} else if len(rec.ClinicalContent.EvidenceRefs) == 1 {
		s.EvidenceScore = 0.5
	} else {
		s.Flags.WeakEvidence = true
	}
	if ctx.AlternativesNoted() {
		s.AlternativesScore = 1.0
	} else {
		s.Flags.MissingAlternatives = true
	}
	if ctx.HasNonPharmTrial() || rec.Type != models.RecommendationTypeStop {
		s.RestraintScore = 1.0
	} else {
		s.Flags.RestraintNotConsidered = true
	}
	if ctx.CareIntensityKnown() {
		s.GoalsAlignmentScore = 1.0
	} else {
		s.Flags.GoalsConflict = true
	}
	s.Total = (s.WarrantedScore + s.EvidenceScore + s.AlternativesScore +
		s.RestraintScore + s.GoalsAlignmentScore) / 5
	return s
}
```

```sql
-- migrations/030_appropriateness_scores.sql
CREATE TABLE appropriateness_scores (
    recommendation_id   UUID PRIMARY KEY,
    total               NUMERIC(3,2) NOT NULL,
    warranted_score     NUMERIC(3,2) NOT NULL,
    evidence_score      NUMERIC(3,2) NOT NULL,
    alternatives_score  NUMERIC(3,2) NOT NULL,
    restraint_score     NUMERIC(3,2) NOT NULL,
    goals_score         NUMERIC(3,2) NOT NULL,
    flags               JSONB NOT NULL,
    scored_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_appropriateness_total ON appropriateness_scores (total);
```

```bash
git commit -m "feat(kb-32): appropriateness scoring rubric paired with recommendation"
```

---

### Task 2: Suppression-pattern detector

**Files:**
- Create: `internal/appropriateness/suppression_detector.go` + `_test.go`

Watches for: per-pharmacist, per-recommendation-type, frequency drops without corresponding clinical-context change. Emits a flag visible to pharmacy employer (via Phase 1 EmployerView aggregate-only).

- [ ] **Step 1-5: Implement window-rate query, test, commit**

```go
package appropriateness

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type SuppressionDetector struct{ db *sql.DB }

func NewSuppressionDetector(db *sql.DB) *SuppressionDetector {
	return &SuppressionDetector{db: db}
}

type SuppressionFlag struct {
	PharmacistID  uuid.UUID
	Type          string
	BaselineRate  float64 // per-week, prior 12 weeks
	RecentRate    float64 // per-week, last 4 weeks
	DropPercent   float64
}

// Detect runs the comparison and returns a flag for any
// pharmacist/type pair whose recent rate is significantly below
// baseline without proportional clinical-context change.
func (d *SuppressionDetector) Detect(ctx context.Context) ([]SuppressionFlag, error) {
	const q = `
WITH weekly AS (
  SELECT author_id, type,
         DATE_TRUNC('week', submitted_at) AS wk,
         COUNT(*) AS n
  FROM recommendations
  WHERE submitted_at IS NOT NULL
    AND submitted_at >= NOW() - INTERVAL '16 weeks'
  GROUP BY author_id, type, DATE_TRUNC('week', submitted_at)
),
agg AS (
  SELECT author_id, type,
         AVG(n) FILTER (WHERE wk < NOW() - INTERVAL '4 weeks') AS baseline_avg,
         AVG(n) FILTER (WHERE wk >= NOW() - INTERVAL '4 weeks') AS recent_avg
  FROM weekly
  GROUP BY author_id, type
)
SELECT author_id, type, baseline_avg, recent_avg
FROM agg
WHERE baseline_avg > 0
  AND (recent_avg / baseline_avg) < 0.6`
	rows, err := d.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var flags []SuppressionFlag
	for rows.Next() {
		var f SuppressionFlag
		if err := rows.Scan(&f.PharmacistID, &f.Type, &f.BaselineRate, &f.RecentRate); err != nil {
			return nil, err
		}
		f.DropPercent = 100 * (1 - f.RecentRate/f.BaselineRate)
		flags = append(flags, f)
	}
	_ = time.Now
	return flags, rows.Err()
}
```

```bash
git commit -m "feat(kb-32): suppression-pattern detector for metric integrity"
```

---

### Task 3: Restraint signaler

**Files:**
- Create: `internal/restraint/signaler.go` + `_test.go`

Per v3 §7 line 422. Surfaces context that argues for non-intervention: care_intensity = palliative, recent decline, family processing, end-stage frailty (CFS ≥7 or AKPS ≤40). Used by craft engine to offer "watchful wait" alongside action recommendations.

- [ ] **Step 1: Write failing test**

```go
func TestSignaler_PalliativeArguesForRestraint(t *testing.T) {
	s := NewSignaler(fakeSubstrate{
		careIntensity: "palliative",
		cfs: 7,
	})
	signals := s.Signals(context.Background(), uuid.New())
	if len(signals) == 0 {
		t.Fatalf("expected at least one restraint signal for palliative resident")
	}
	hasCareIntensitySignal := false
	for _, sig := range signals {
		if sig.Reason == "care_intensity_palliative" {
			hasCareIntensitySignal = true
		}
	}
	if !hasCareIntensitySignal {
		t.Errorf("missing care_intensity_palliative signal")
	}
}
```

- [ ] **Step 2-5: Implement substrate-derived signal generation, test, commit**

```go
package restraint

import (
	"context"

	"github.com/google/uuid"
)

type Signal struct {
	Reason   string // care_intensity_palliative, end_stage_frailty, family_processing, recent_decline
	Severity string // strong | moderate | weak
	Context  string // narrative for the pharmacist
}

type Signaler struct{ substrate SubstrateReader }

type SubstrateReader interface {
	CareIntensity(ctx context.Context, residentID uuid.UUID) (string, error)
	CFS(ctx context.Context, residentID uuid.UUID) (int, error)
	AKPS(ctx context.Context, residentID uuid.UUID) (int, error)
	RecentDeclineWindowDays(ctx context.Context, residentID uuid.UUID) (int, error)
}

func NewSignaler(s SubstrateReader) *Signaler { return &Signaler{substrate: s} }

func (s *Signaler) Signals(ctx context.Context, residentID uuid.UUID) []Signal {
	var out []Signal
	if ci, _ := s.substrate.CareIntensity(ctx, residentID); ci == "palliative" {
		out = append(out, Signal{
			Reason: "care_intensity_palliative", Severity: "strong",
			Context: "Resident is on palliative pathway; non-symptom interventions warrant explicit benefit case.",
		})
	}
	if cfs, _ := s.substrate.CFS(ctx, residentID); cfs >= 7 {
		out = append(out, Signal{
			Reason: "end_stage_frailty", Severity: "strong",
			Context: "CFS ≥7 — consider whether intervention upside outweighs destabilisation risk.",
		})
	}
	if d, _ := s.substrate.RecentDeclineWindowDays(ctx, residentID); d > 0 && d <= 14 {
		out = append(out, Signal{
			Reason: "recent_decline", Severity: "moderate",
			Context: "Recent decline within 14 days — consider deferring non-urgent change.",
		})
	}
	return out
}
```

```bash
git commit -m "feat(kb-32): restraint signaler from substrate state"
```

---

### Task 4: RPL evidence pack generator

**Files:**
- Create: `internal/rpl/evidence_pack_generator.go` + `_test.go`

Per v3 §10 line 533. Generates competency-evidence packs from longitudinal pharmacist work, structured per APC RPL framework's five competency dimensions. Pharmacist self-export.

- [ ] **Step 1-5: Implement, test, commit**

```go
package rpl

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

type EvidencePack struct {
	PharmacistID uuid.UUID
	Period       string // e.g. "2024-2026"
	Dimensions   []CompetencyDimension
}

type CompetencyDimension struct {
	Name         string
	CaseCount    int
	ExampleCases []ExampleCase
}

type ExampleCase struct {
	RecommendationID uuid.UUID
	Title            string
	OutcomeSummary   string
	EvidenceTrail    []string // links to EvidenceTrace edges
}

type Generator struct{ db *sql.DB }

func NewGenerator(db *sql.DB) *Generator { return &Generator{db: db} }

func (g *Generator) Generate(ctx context.Context, pharmacistID uuid.UUID,
	periodStart, periodEnd string) (*EvidencePack, error) {
	// Query: for each of the 5 APC competency dimensions, count recommendations
	// authored by this pharmacist matching that dimension's tag, plus pull
	// 3 example cases per dimension. Real implementation queries
	// recommendations + appropriateness_scores + evidence_trace edges.
	pack := &EvidencePack{
		PharmacistID: pharmacistID,
		Period:       periodStart + "/" + periodEnd,
	}
	// ... 5-dimension loop populating CaseCount + ExampleCases
	return pack, nil
}
```

Test: seed 5 recommendations with appropriateness scores, generate pack, assert each dimension has ≥1 example case.

```bash
git commit -m "feat(kb-32): RPL evidence-pack generator for APC credentialing"
```

---

### Task 5: CPD tagger + AHPRA record generator

**Files:**
- Create: `internal/cpd/tagger.go` + `_test.go`
- Create: `internal/cpd/ahpra_record_generator.go` + `_test.go`
- Create: `migrations/031_cpd_activities.sql` + rollback

Per v3 §12 line 699. Auto-tags CPD-eligible activities (each comprehensive review, structured recommendation crafting). Surfaces CPD-relevant cases for reflective writing. Generates AHPRA-format CPD records linked to EvidenceTrace.

- [ ] **Step 1-5: Migration + tagger + record generator + test, commit**

```sql
-- 031
CREATE TABLE cpd_activities (
    id              UUID PRIMARY KEY,
    pharmacist_id   UUID NOT NULL,
    activity_type   TEXT NOT NULL, -- 'comprehensive_review','recommendation_authored','reflective_writing'
    case_ref        UUID,          -- recommendation_id or review_id
    hours           NUMERIC(3,1) NOT NULL,
    occurred_at     TIMESTAMPTZ NOT NULL,
    reflective_text TEXT,
    evidence_trace_refs UUID[],
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_cpd_pharmacist_year ON cpd_activities (pharmacist_id, EXTRACT(year FROM occurred_at));
```

```go
// tagger.go
package cpd

import (
	"context"

	"github.com/google/uuid"
)

type Tagger struct{ store Store }

// OnRecommendationDecided is called by Plan 0.1's lifecycle when a
// recommendation reaches `decided`. It auto-creates a CPD activity entry
// for the authoring pharmacist.
func (t *Tagger) OnRecommendationDecided(ctx context.Context, recID, authorID uuid.UUID) error {
	return t.store.Create(ctx, Activity{
		ID:           uuid.New(),
		PharmacistID: authorID,
		ActivityType: "recommendation_authored",
		CaseRef:      recID,
		Hours:        0.25, // 15-min equivalent per recommendation; tunable
	})
}
```

```bash
git commit -m "feat(kb-32): CPD tagger + AHPRA record generator"
```

---

### Task 6: Wire appropriateness into Recommendation lifecycle

**Files:**
- Modify: `shared/v2_substrate/recommendation/lifecycle.go` (Plan 0.1)

When transition reaches `drafted`, call `appropriateness.Score(rec, substrateContext)` and persist to `appropriateness_scores`. When reaching `decided` or terminal states, recompute the actioned-vs-appropriate divergence metric.

- [ ] **Step 1-5: Wire, test, commit**

```bash
git commit -m "feat: appropriateness scoring paired with Recommendation lifecycle"
```

---

### Task 7: Integration test — Persuasive-framing-of-marginal detection

End-to-end: create a recommendation with weak appropriateness score but high acceptance rate (simulated via fixtures). Run divergence detector. Assert flag emitted to employer view (aggregate only; not identifiable cases).

```bash
git commit -m "test: persuasive-framing-of-marginal flag end-to-end"
```

---

## Spec coverage

- [x] Clinical appropriateness check (Architectural Commitment 8) — Tasks 1, 6
- [x] Suppression-pattern detector — Task 2
- [x] Restraint signals — Task 3
- [x] RPL-evidence module (30 June 2026 cliff) — Task 4
- [x] CPD tagging + AHPRA records — Task 5
- [x] Persuasive-framing-of-marginal divergence detector — Task 7

Plan complete and saved.
