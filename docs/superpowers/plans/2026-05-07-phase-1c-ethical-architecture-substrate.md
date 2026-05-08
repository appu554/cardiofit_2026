# Phase 1c — Ethical Architecture Substrate Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Land the cross-cutting ethical-architecture substrate that all platform services inherit from. Per *Ethical Architecture Implementation Guidelines v1.0*: ERM (Ethical Reasoning Module), `EthicalDecisionMetadata` + `EthicsLog` parallel substrate, vulnerability assessment, restrictive-practice consent gating, three pattern detectors, bias-detection foundation, and incident classification with hold mechanism. This is the substrate that makes the seven §9 principles operationally enforceable rather than aspirational.

**Architecture:** New package `shared/v2_substrate/ethics/` containing: ERM module + per-decision-type reasoners; decision-metadata recorder; ethics-log substrate; pattern-detection workers; vulnerability assessment; restrictive-practice consent gating extending Plan 0.2 Consent. Standalone service `backend/services/ethics-monitoring/` runs the daily/weekly automated detection layer.

**Tech Stack:** Go, PostgreSQL, depends on Plan 0.1 (Recommendation entity for ERM hooks), Plan 0.2 (Consent state machine extension for restrictive practice), Phase 1a (visibility-class substrate + DataAggregationConsent for ERM visibility-decision review). No new third-party deps.

---

## File Structure

**ERM package:**
- `shared/v2_substrate/ethics/erm/module.go` + `module_test.go`
- `shared/v2_substrate/ethics/erm/reasoners/recommendation.go` + `_test.go`
- `shared/v2_substrate/ethics/erm/reasoners/visibility.go` + `_test.go`
- `shared/v2_substrate/ethics/erm/reasoners/authorisation.go` + `_test.go`
- `shared/v2_substrate/ethics/erm/escalation.go` + `escalation_test.go`

**Decision metadata + EthicsLog:**
- `shared/v2_substrate/ethics/decision_metadata/recorder.go` + `_test.go`
- `shared/v2_substrate/ethics/decision_metadata/store.go`
- `shared/v2_substrate/ethics/ethics_log/logger.go` + `_test.go`
- `shared/v2_substrate/ethics/ethics_log/querier.go` + `_test.go`

**Pattern detection:**
- `shared/v2_substrate/ethics/pattern_detection/acceptance_appropriateness.go` + `_test.go`
- `shared/v2_substrate/ethics/pattern_detection/suppression.go` + `_test.go`
- `shared/v2_substrate/ethics/pattern_detection/surveillance.go` + `_test.go`
- `shared/v2_substrate/ethics/pattern_detection/bias.go` + `_test.go`

**Vulnerability + consent extensions:**
- `shared/v2_substrate/ethics/vulnerability/assessment.go` + `_test.go`
- `shared/v2_substrate/ethics/vulnerability/adapter.go`
- `shared/v2_substrate/ethics/consent_extension/restrictive_practice.go` + `_test.go`
- `shared/v2_substrate/ethics/consent_extension/sdm_integration.go`

**Incident response:**
- `shared/v2_substrate/ethics/incident_response/classifier.go` + `_test.go`
- `shared/v2_substrate/ethics/incident_response/hold.go` + `_test.go`
- `shared/v2_substrate/ethics/incident_response/notifier.go`

**Standalone service:**
- `backend/services/ethics-monitoring/cmd/server/main.go`
- `backend/services/ethics-monitoring/internal/monitors/daily/run.go`
- `backend/services/ethics-monitoring/internal/monitors/weekly/run.go`
- `backend/services/ethics-monitoring/internal/api/http.go`

**Migrations:**
- `migrations/035_ethical_decision_metadata.sql` + rollback
- `migrations/036_ethics_log.sql` + rollback
- `migrations/037_vulnerability_assessment.sql` + rollback
- `migrations/038_restrictive_practice_consent.sql` + rollback
- `migrations/039_incidents.sql` + rollback

---

### Task 1: EthicalDecisionMetadata entity + migration 035

**Files:**
- Create: `shared/v2_substrate/ethics/decision_metadata/recorder.go` + `recorder_test.go`
- Create: `shared/v2_substrate/ethics/decision_metadata/store.go`
- Create: `migrations/035_ethical_decision_metadata.sql` + rollback

Per Guidelines §14.1. Every algorithmic decision in the platform attaches metadata: which component, decision type, affected subject, principles implicated, ERM outcome, contestation enabled flag, audit trace ref. Queries against this metadata power detection mechanisms.

- [ ] **Step 1: Write failing test**

```go
package decision_metadata

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRecorder_RecordRoundTrip(t *testing.T) {
	store := NewInMemoryStore()
	rec := NewRecorder(store)

	id := uuid.New()
	traceRef := uuid.New()
	subjectID := uuid.New().String()
	if err := rec.Record(context.Background(), Metadata{
		DecisionID:           id,
		Component:            "kb-30",
		DecisionType:         "recommendation_draft",
		AffectedSubjectID:    subjectID,
		AffectedSubjectClass: "resident",
		PrinciplesImplicated: []string{"P2", "P3"},
		ERMReviewed:          true,
		ERMOutcome:           ptr("approve_with_monitoring"),
		ContestationEnabled:  true,
		AuditTraceRef:        traceRef,
		Timestamp:            time.Now().UTC(),
	}); err != nil {
		t.Fatalf("record: %v", err)
	}

	got, err := store.Get(context.Background(), id)
	if err != nil || got == nil {
		t.Fatalf("get: err=%v got=%v", err, got)
	}
	if got.AffectedSubjectClass != "resident" {
		t.Errorf("subject class roundtrip fail")
	}
	if len(got.PrinciplesImplicated) != 2 {
		t.Errorf("principles roundtrip fail: %v", got.PrinciplesImplicated)
	}
}

func TestRecorder_QueryBySubject(t *testing.T) {
	store := NewInMemoryStore()
	rec := NewRecorder(store)
	subj := uuid.New().String()
	for i := 0; i < 3; i++ {
		_ = rec.Record(context.Background(), Metadata{
			DecisionID: uuid.New(), Component: "kb-30",
			AffectedSubjectID: subj, AffectedSubjectClass: "resident",
			Timestamp: time.Now().UTC(),
		})
	}
	list, _ := store.QueryBySubject(context.Background(), subj)
	if len(list) != 3 {
		t.Errorf("got %d entries for subject", len(list))
	}
}

func ptr(s string) *string { return &s }
```

- [ ] **Step 2-3: Implement**

```go
// recorder.go
package decision_metadata

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Metadata struct {
	DecisionID           uuid.UUID
	Component            string
	DecisionType         string
	AffectedSubjectID    string
	AffectedSubjectClass string // "resident" / "pharmacist" / "gp" / etc
	PrinciplesImplicated []string
	ERMReviewed          bool
	ERMOutcome           *string
	ContestationEnabled  bool
	AuditTraceRef        uuid.UUID
	Timestamp            time.Time
}

type Store interface {
	Put(ctx context.Context, m Metadata) error
	Get(ctx context.Context, id uuid.UUID) (*Metadata, error)
	QueryBySubject(ctx context.Context, subjectID string) ([]Metadata, error)
}

type Recorder struct{ store Store }

func NewRecorder(s Store) *Recorder { return &Recorder{store: s} }

func (r *Recorder) Record(ctx context.Context, m Metadata) error {
	if m.Timestamp.IsZero() {
		m.Timestamp = time.Now().UTC()
	}
	return r.store.Put(ctx, m)
}
```

```go
// store.go
package decision_metadata

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

type InMemoryStore struct {
	mu sync.RWMutex
	m  map[uuid.UUID]Metadata
}

func NewInMemoryStore() *InMemoryStore { return &InMemoryStore{m: map[uuid.UUID]Metadata{}} }

func (s *InMemoryStore) Put(_ context.Context, m Metadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[m.DecisionID] = m
	return nil
}

func (s *InMemoryStore) Get(_ context.Context, id uuid.UUID) (*Metadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if m, ok := s.m[id]; ok {
		return &m, nil
	}
	return nil, nil
}

func (s *InMemoryStore) QueryBySubject(_ context.Context, subjectID string) ([]Metadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []Metadata{}
	for _, m := range s.m {
		if m.AffectedSubjectID == subjectID {
			out = append(out, m)
		}
	}
	return out, nil
}
```

- [ ] **Step 4: Migration 035**

```sql
-- 035_ethical_decision_metadata.sql
BEGIN;
CREATE TABLE ethical_decision_metadata (
    decision_id            UUID PRIMARY KEY,
    component              VARCHAR(64) NOT NULL,
    decision_type          VARCHAR(64) NOT NULL,
    affected_subject_id    VARCHAR(64) NOT NULL,
    affected_subject_class VARCHAR(32) NOT NULL,
    principles_implicated  TEXT[] NOT NULL DEFAULT '{}',
    erm_reviewed           BOOLEAN NOT NULL DEFAULT FALSE,
    erm_outcome            VARCHAR(32),
    contestation_enabled   BOOLEAN NOT NULL DEFAULT FALSE,
    audit_trace_ref        UUID NOT NULL,
    timestamp              TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_edm_subject    ON ethical_decision_metadata (affected_subject_id);
CREATE INDEX idx_edm_component  ON ethical_decision_metadata (component);
CREATE INDEX idx_edm_timestamp  ON ethical_decision_metadata (timestamp);
CREATE INDEX idx_edm_principles ON ethical_decision_metadata USING GIN (principles_implicated);
COMMIT;
```

- [ ] **Step 5: Commit**

```bash
git commit -m "feat(ethics): EthicalDecisionMetadata recorder with subject + principles indices"
```

---

### Task 2: EthicsLog substrate + migration 036

**Files:**
- Create: `shared/v2_substrate/ethics/ethics_log/logger.go` + `logger_test.go`
- Create: `shared/v2_substrate/ethics/ethics_log/querier.go` + `querier_test.go`
- Create: `migrations/036_ethics_log.sql` + rollback

Per Guidelines §14.2. Parallel to EvidenceTrace. EntryType enum: `decision`, `concern_flagged`, `review_requested`, `pattern_detected`, `incident`. Severity 1–5. Status `open` / `investigating` / `remediated` / `verified` / `closed`.

- [ ] **Step 1-3: Test + implement**

```go
package ethics_log

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestLogger_AppendAndQuery(t *testing.T) {
	store := NewInMemoryStore()
	l := NewLogger(store)
	q := NewQuerier(store)

	decisionID := uuid.New()
	if err := l.Append(context.Background(), Entry{
		DecisionID:  decisionID,
		EntryType:   EntryTypePatternDetected,
		Severity:    3,
		Description: "acceptance-appropriateness divergence",
		Status:      StatusOpen,
	}); err != nil {
		t.Fatalf("append: %v", err)
	}

	list, _ := q.ByDecision(context.Background(), decisionID)
	if len(list) != 1 || list[0].Severity != 3 {
		t.Errorf("query roundtrip fail: %v", list)
	}

	openSev3, _ := q.OpenAtSeverity(context.Background(), 3)
	if len(openSev3) != 1 {
		t.Errorf("open-at-severity-3 query: %d", len(openSev3))
	}
	_ = time.Now()
}
```

```go
// logger.go
package ethics_log

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

type EntryType string

const (
	EntryTypeDecision         EntryType = "decision"
	EntryTypeConcernFlagged   EntryType = "concern_flagged"
	EntryTypeReviewRequested  EntryType = "review_requested"
	EntryTypePatternDetected  EntryType = "pattern_detected"
	EntryTypeIncident         EntryType = "incident"
)

type Status string

const (
	StatusOpen          Status = "open"
	StatusInvestigating Status = "investigating"
	StatusRemediated    Status = "remediated"
	StatusVerified      Status = "verified"
	StatusClosed        Status = "closed"
)

type Entry struct {
	ID                 uuid.UUID
	DecisionID         uuid.UUID
	EntryType          EntryType
	Severity           int // 1..5
	Description        string
	Reviewer           *string
	ReviewOutcome      *string
	RemediationActions []string
	Status             Status
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type Store interface {
	Append(ctx context.Context, e Entry) error
	List(ctx context.Context) ([]Entry, error)
}

type Logger struct{ store Store }

func NewLogger(s Store) *Logger { return &Logger{store: s} }

func (l *Logger) Append(ctx context.Context, e Entry) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	now := time.Now().UTC()
	e.CreatedAt = now
	e.UpdatedAt = now
	if e.Status == "" {
		e.Status = StatusOpen
	}
	return l.store.Append(ctx, e)
}

type InMemoryStore struct {
	mu      sync.RWMutex
	entries []Entry
}

func NewInMemoryStore() *InMemoryStore { return &InMemoryStore{} }
func (s *InMemoryStore) Append(_ context.Context, e Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, e)
	return nil
}
func (s *InMemoryStore) List(_ context.Context) ([]Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Entry, len(s.entries))
	copy(out, s.entries)
	return out, nil
}
```

```go
// querier.go
package ethics_log

import (
	"context"

	"github.com/google/uuid"
)

type Querier struct{ store Store }

func NewQuerier(s Store) *Querier { return &Querier{store: s} }

func (q *Querier) ByDecision(ctx context.Context, id uuid.UUID) ([]Entry, error) {
	all, err := q.store.List(ctx)
	if err != nil {
		return nil, err
	}
	out := []Entry{}
	for _, e := range all {
		if e.DecisionID == id {
			out = append(out, e)
		}
	}
	return out, nil
}

func (q *Querier) OpenAtSeverity(ctx context.Context, sev int) ([]Entry, error) {
	all, err := q.store.List(ctx)
	if err != nil {
		return nil, err
	}
	out := []Entry{}
	for _, e := range all {
		if e.Severity == sev && e.Status == StatusOpen {
			out = append(out, e)
		}
	}
	return out, nil
}
```

- [ ] **Step 4: Migration 036**

```sql
-- 036_ethics_log.sql
BEGIN;
CREATE TABLE ethics_log (
    id                  UUID PRIMARY KEY,
    decision_id         UUID NOT NULL REFERENCES ethical_decision_metadata(decision_id),
    entry_type          VARCHAR(32) NOT NULL CHECK (entry_type IN
                            ('decision','concern_flagged','review_requested','pattern_detected','incident')),
    severity            INT NOT NULL CHECK (severity BETWEEN 1 AND 5),
    description         TEXT NOT NULL,
    reviewer            VARCHAR(64),
    review_outcome      VARCHAR(64),
    remediation_actions TEXT[] NOT NULL DEFAULT '{}',
    status              VARCHAR(16) NOT NULL CHECK (status IN
                            ('open','investigating','remediated','verified','closed')),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_log_decision ON ethics_log (decision_id);
CREATE INDEX idx_log_severity ON ethics_log (severity);
CREATE INDEX idx_log_status   ON ethics_log (status);
CREATE INDEX idx_log_created  ON ethics_log (created_at DESC);
COMMIT;
```

- [ ] **Step 5: Commit**

```bash
git commit -m "feat(ethics): EthicsLog substrate with severity + status state machine"
```

---

### Task 3: ERM scaffold + decision-point hook interface

**Files:**
- Create: `shared/v2_substrate/ethics/erm/module.go` + `module_test.go`

Per Guidelines §4.1–4.2. ERM identifies decisions requiring ethical attention and applies established review patterns. Per §4.6, **ERM is NOT autonomous** — human judgment remains the final ethical authority for any non-routine case.

- [ ] **Step 1: Write failing test**

```go
package erm

import (
	"context"
	"testing"
)

func TestERM_RoutesToRegisteredReasoner(t *testing.T) {
	called := false
	r := ReasonerFunc(func(_ context.Context, dp DecisionPoint) (Outcome, []Concern) {
		called = true
		return OutcomeApprove, nil
	})
	m := NewModule()
	m.Register(DecisionTypeRecommendationDraft, r)

	out, _, err := m.Review(context.Background(), DecisionPoint{
		Component: "kb-30", DecisionType: DecisionTypeRecommendationDraft,
	})
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if !called {
		t.Errorf("registered reasoner was not invoked")
	}
	if out != OutcomeApprove {
		t.Errorf("outcome = %v, want approve", out)
	}
}

func TestERM_RejectsUnknownDecisionType(t *testing.T) {
	m := NewModule()
	_, _, err := m.Review(context.Background(), DecisionPoint{DecisionType: "bogus"})
	if err == nil {
		t.Errorf("unknown decision type should error")
	}
}
```

- [ ] **Step 2-3: Implement**

```go
package erm

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type DecisionType string

const (
	DecisionTypeRecommendationDraft DecisionType = "recommendation_draft"
	DecisionTypeVisibilityAggregate DecisionType = "visibility_aggregate"
	DecisionTypeAuthorisation       DecisionType = "authorisation"
)

type Outcome string

const (
	OutcomeApprove               Outcome = "approve"
	OutcomeApproveWithMonitoring Outcome = "approve_with_monitoring"
	OutcomeHold                  Outcome = "hold"
	OutcomeReject                Outcome = "reject"
)

type Concern struct {
	Principle    string // "P1".."P7"
	ConcernLevel int    // 1..5
	Reasoning    string
}

type DecisionPoint struct {
	DecisionID     uuid.UUID
	Component      string
	DecisionType   DecisionType
	Inputs         interface{}
	ProposedOutput interface{}
}

type Reasoner interface {
	Review(ctx context.Context, dp DecisionPoint) (Outcome, []Concern)
}

type ReasonerFunc func(ctx context.Context, dp DecisionPoint) (Outcome, []Concern)

func (f ReasonerFunc) Review(ctx context.Context, dp DecisionPoint) (Outcome, []Concern) {
	return f(ctx, dp)
}

var ErrUnknownDecisionType = errors.New("erm: no reasoner registered for this decision type")

type Module struct {
	reasoners map[DecisionType]Reasoner
}

func NewModule() *Module {
	return &Module{reasoners: map[DecisionType]Reasoner{}}
}

func (m *Module) Register(dt DecisionType, r Reasoner) {
	m.reasoners[dt] = r
}

func (m *Module) Review(ctx context.Context, dp DecisionPoint) (Outcome, []Concern, error) {
	r, ok := m.reasoners[dp.DecisionType]
	if !ok {
		return "", nil, ErrUnknownDecisionType
	}
	out, concerns := r.Review(ctx, dp)
	return out, concerns, nil
}
```

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(ethics): ERM scaffold with 4-outcome decision routing"
```

---

### Task 4: ERM reasoner — recommendation-draft review

**Files:**
- Create: `shared/v2_substrate/ethics/erm/reasoners/recommendation.go` + `recommendation_test.go`

Per craft engine §9 (appropriateness threshold) + Guidelines §1 (Principle 2 — acceptance follows appropriateness). The recommendation reasoner enforces: appropriateness ≥ 3.0; restraint signals not silently overridden; rule not in 30-day acceptance-appropriateness divergence list.

- [ ] **Step 1-3: Test + implement**

```go
package reasoners

import (
	"context"
	"testing"

	"github.com/cardiofit/shared/v2_substrate/ethics/erm"
)

func TestRecommendationReasoner_HoldsLowAppropriateness(t *testing.T) {
	r := NewRecommendationReasoner(3.0, &fakeDivergence{})
	dp := erm.DecisionPoint{
		DecisionType:   erm.DecisionTypeRecommendationDraft,
		ProposedOutput: RecommendationProposal{AppropriatenessScore: 2.4, RuleID: "R1"},
	}
	out, concerns := r.Review(context.Background(), dp)
	if out != erm.OutcomeHold {
		t.Errorf("expected hold for low appropriateness, got %v", out)
	}
	if len(concerns) == 0 || concerns[0].Principle != "P2" {
		t.Errorf("expected P2 concern, got %v", concerns)
	}
}

func TestRecommendationReasoner_ApprovesHighAppropriateness(t *testing.T) {
	r := NewRecommendationReasoner(3.0, &fakeDivergence{})
	dp := erm.DecisionPoint{
		DecisionType:   erm.DecisionTypeRecommendationDraft,
		ProposedOutput: RecommendationProposal{AppropriatenessScore: 4.2, RuleID: "R1"},
	}
	out, _ := r.Review(context.Background(), dp)
	if out != erm.OutcomeApprove {
		t.Errorf("expected approve, got %v", out)
	}
}

func TestRecommendationReasoner_HoldsDivergentRule(t *testing.T) {
	r := NewRecommendationReasoner(3.0, &fakeDivergence{divergent: map[string]bool{"R1": true}})
	dp := erm.DecisionPoint{
		DecisionType:   erm.DecisionTypeRecommendationDraft,
		ProposedOutput: RecommendationProposal{AppropriatenessScore: 4.5, RuleID: "R1"},
	}
	out, _ := r.Review(context.Background(), dp)
	if out != erm.OutcomeHold {
		t.Errorf("expected hold for divergent rule, got %v", out)
	}
}

type fakeDivergence struct{ divergent map[string]bool }

func (f *fakeDivergence) IsDivergent(_ context.Context, ruleID string) (bool, error) {
	return f.divergent[ruleID], nil
}
```

```go
package reasoners

import (
	"context"

	"github.com/cardiofit/shared/v2_substrate/ethics/erm"
)

type RecommendationProposal struct {
	AppropriatenessScore float64
	RuleID               string
	RestraintOverridden  bool
	RestraintReasoning   string
}

type DivergenceSource interface {
	IsDivergent(ctx context.Context, ruleID string) (bool, error)
}

type RecommendationReasoner struct {
	minAppropriateness float64
	divergence         DivergenceSource
}

func NewRecommendationReasoner(minAppr float64, div DivergenceSource) *RecommendationReasoner {
	return &RecommendationReasoner{minAppropriateness: minAppr, divergence: div}
}

func (r *RecommendationReasoner) Review(ctx context.Context, dp erm.DecisionPoint) (erm.Outcome, []erm.Concern) {
	prop, ok := dp.ProposedOutput.(RecommendationProposal)
	if !ok {
		return erm.OutcomeHold, []erm.Concern{{Principle: "P6", ConcernLevel: 2, Reasoning: "malformed proposal"}}
	}
	if prop.AppropriatenessScore < r.minAppropriateness {
		return erm.OutcomeHold, []erm.Concern{{Principle: "P2", ConcernLevel: 4,
			Reasoning: "appropriateness below minimum threshold"}}
	}
	if prop.RestraintOverridden && prop.RestraintReasoning == "" {
		return erm.OutcomeHold, []erm.Concern{{Principle: "P3", ConcernLevel: 3,
			Reasoning: "restraint override without documented reasoning"}}
	}
	if r.divergence != nil {
		divergent, _ := r.divergence.IsDivergent(ctx, prop.RuleID)
		if divergent {
			return erm.OutcomeHold, []erm.Concern{{Principle: "P2", ConcernLevel: 4,
				Reasoning: "rule on 30-day acceptance-appropriateness divergence list"}}
		}
	}
	return erm.OutcomeApprove, nil
}
```

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(ethics): ERM recommendation-draft reasoner enforcing P2/P3"
```

---

### Task 5: ERM reasoner — visibility-class + aggregation review

**Files:**
- Create: `shared/v2_substrate/ethics/erm/reasoners/visibility.go` + `visibility_test.go`

Reviews proposed aggregations from Phase 1a permission middleware. Enforces: PFA aggregation gate satisfied; no re-identification risk in small subsets (min 5 pharmacists per Guidelines §11.1 Risk 11); employer query patterns not matching surveillance heuristics (§9.7).

- [ ] **Step 1-3: Test + implement**

```go
package reasoners

import (
	"context"
	"testing"

	"github.com/cardiofit/shared/v2_substrate/ethics/erm"
)

func TestVisibilityReasoner_HoldsBelowReidentificationFloor(t *testing.T) {
	r := NewVisibilityReasoner(5)
	dp := erm.DecisionPoint{
		DecisionType:   erm.DecisionTypeVisibilityAggregate,
		ProposedOutput: AggregationProposal{PharmacistCount: 3, GateSatisfied: true},
	}
	out, _ := r.Review(context.Background(), dp)
	if out != erm.OutcomeHold {
		t.Errorf("expected hold for re-identification risk")
	}
}

func TestVisibilityReasoner_HoldsGateUnsatisfied(t *testing.T) {
	r := NewVisibilityReasoner(5)
	dp := erm.DecisionPoint{
		DecisionType:   erm.DecisionTypeVisibilityAggregate,
		ProposedOutput: AggregationProposal{PharmacistCount: 20, GateSatisfied: false},
	}
	out, _ := r.Review(context.Background(), dp)
	if out != erm.OutcomeHold {
		t.Errorf("expected hold when gate not satisfied")
	}
}

func TestVisibilityReasoner_ApproveHappy(t *testing.T) {
	r := NewVisibilityReasoner(5)
	dp := erm.DecisionPoint{
		DecisionType:   erm.DecisionTypeVisibilityAggregate,
		ProposedOutput: AggregationProposal{PharmacistCount: 30, GateSatisfied: true, SurveillanceFlag: false},
	}
	out, _ := r.Review(context.Background(), dp)
	if out != erm.OutcomeApprove {
		t.Errorf("expected approve, got %v", out)
	}
}
```

```go
package reasoners

import (
	"context"

	"github.com/cardiofit/shared/v2_substrate/ethics/erm"
)

type AggregationProposal struct {
	PharmacistCount  int
	GateSatisfied    bool
	SurveillanceFlag bool
}

type VisibilityReasoner struct {
	reidentificationFloor int
}

func NewVisibilityReasoner(floor int) *VisibilityReasoner {
	return &VisibilityReasoner{reidentificationFloor: floor}
}

func (r *VisibilityReasoner) Review(_ context.Context, dp erm.DecisionPoint) (erm.Outcome, []erm.Concern) {
	prop, ok := dp.ProposedOutput.(AggregationProposal)
	if !ok {
		return erm.OutcomeHold, []erm.Concern{{Principle: "P6", ConcernLevel: 2, Reasoning: "malformed aggregation"}}
	}
	if !prop.GateSatisfied {
		return erm.OutcomeHold, []erm.Concern{{Principle: "P5", ConcernLevel: 4,
			Reasoning: "PFA aggregation gate not satisfied"}}
	}
	if prop.PharmacistCount < r.reidentificationFloor {
		return erm.OutcomeHold, []erm.Concern{{Principle: "P5", ConcernLevel: 5,
			Reasoning: "re-identification risk: subset below minimum"}}
	}
	if prop.SurveillanceFlag {
		return erm.OutcomeHold, []erm.Concern{{Principle: "P5", ConcernLevel: 4,
			Reasoning: "query pattern matches surveillance heuristic"}}
	}
	return erm.OutcomeApprove, nil
}
```

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(ethics): ERM visibility reasoner enforcing P5 + reidentification floor"
```

---

### Task 6: VulnerabilityAssessment entity + migration 037

**Files:**
- Create: `shared/v2_substrate/ethics/vulnerability/assessment.go` + `assessment_test.go`
- Create: `migrations/037_vulnerability_assessment.sql` + rollback

Per Guidelines §5.3. Structured context that shifts what the platform considers appropriate — not a label. Read by ERM and craft engine.

- [ ] **Step 1-3: Test + implement**

```go
package vulnerability

import (
	"testing"
	"time"
)

func TestAssessment_FreshIsValid(t *testing.T) {
	a := Assessment{AssessedAt: time.Now().UTC()}
	if a.Stale(30 * 24 * time.Hour) {
		t.Errorf("fresh assessment should not be stale")
	}
}

func TestAssessment_StaleAfterTTL(t *testing.T) {
	a := Assessment{AssessedAt: time.Now().Add(-100 * 24 * time.Hour)}
	if !a.Stale(30 * 24 * time.Hour) {
		t.Errorf("100-day-old assessment should be stale at 30-day TTL")
	}
}

func TestCareIntensity_ValidEnum(t *testing.T) {
	for _, ci := range []CareIntensity{CareActive, CareComfort, CarePalliative, CareEndOfLife} {
		if !ci.Valid() {
			t.Errorf("%v should be valid", ci)
		}
	}
	if CareIntensity("nope").Valid() {
		t.Errorf("unknown care intensity should be invalid")
	}
}
```

```go
package vulnerability

import (
	"time"

	"github.com/google/uuid"
)

type CognitiveCapacity string

const (
	CapacityIntact             CognitiveCapacity = "intact"
	CapacityMildImpairment     CognitiveCapacity = "mild_impairment"
	CapacityModerateImpairment CognitiveCapacity = "moderate_impairment"
	CapacitySevereImpairment   CognitiveCapacity = "severe_impairment"
	CapacityUncertain          CognitiveCapacity = "uncertain"
)

type CareIntensity string

const (
	CareActive     CareIntensity = "active"
	CareComfort    CareIntensity = "comfort"
	CarePalliative CareIntensity = "palliative"
	CareEndOfLife  CareIntensity = "end_of_life"
)

func (c CareIntensity) Valid() bool {
	switch c {
	case CareActive, CareComfort, CarePalliative, CareEndOfLife:
		return true
	}
	return false
}

type Assessment struct {
	ResidentID            uuid.UUID
	CognitiveCapacity     CognitiveCapacity
	FrailtyTier           string // CFS-based 1..9
	CareIntensity         CareIntensity
	SDMRequired           bool
	FamilyAdvocacyPresent bool
	RestrictivePractice   bool
	RecentDeterioration   bool
	AssessedAt            time.Time
}

func (a Assessment) Stale(ttl time.Duration) bool {
	return time.Since(a.AssessedAt) > ttl
}
```

- [ ] **Step 4: Migration 037**

```sql
-- 037_vulnerability_assessment.sql
BEGIN;
CREATE TABLE vulnerability_assessments (
    resident_id              UUID PRIMARY KEY,
    cognitive_capacity       VARCHAR(32) NOT NULL,
    frailty_tier             VARCHAR(16) NOT NULL,
    care_intensity           VARCHAR(16) NOT NULL CHECK (care_intensity IN
                                ('active','comfort','palliative','end_of_life')),
    sdm_required             BOOLEAN NOT NULL DEFAULT FALSE,
    family_advocacy_present  BOOLEAN NOT NULL DEFAULT FALSE,
    restrictive_practice     BOOLEAN NOT NULL DEFAULT FALSE,
    recent_deterioration     BOOLEAN NOT NULL DEFAULT FALSE,
    assessed_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_vuln_care_intensity ON vulnerability_assessments (care_intensity);
COMMIT;
```

- [ ] **Step 5: Commit**

```bash
git commit -m "feat(ethics): VulnerabilityAssessment entity with care-intensity enum"
```

---

### Task 7: Restrictive-practice consent gating + migration 038

**Files:**
- Create: `shared/v2_substrate/ethics/consent_extension/restrictive_practice.go` + `restrictive_practice_test.go`
- Create: `migrations/038_restrictive_practice_consent.sql` + rollback

Per Guidelines §6.3. Extends Plan 0.2 Consent state machine for psychotropic / physical-restraint / environmental-restraint / seclusion. ERM gates recommendations involving these practices on `active` consent for the specific practice type.

- [ ] **Step 1-3: Test + implement**

```go
package consent_extension

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRestrictivePractice_ActiveConsentAllows(t *testing.T) {
	c := RestrictivePracticeConsent{
		PracticeType:     PracticeChemicalRestraint,
		Status:           "active",
		MaxDuration:      12 * 7 * 24 * time.Hour,
		GrantedAt:        time.Now().Add(-7 * 24 * time.Hour),
	}
	if !c.Allows(time.Now()) {
		t.Errorf("active consent within max duration should allow")
	}
}

func TestRestrictivePractice_ExpiredDenies(t *testing.T) {
	c := RestrictivePracticeConsent{
		PracticeType: PracticeChemicalRestraint,
		Status:       "active",
		MaxDuration:  12 * 7 * 24 * time.Hour,
		GrantedAt:    time.Now().Add(-100 * 24 * time.Hour),
	}
	if c.Allows(time.Now()) {
		t.Errorf("expired consent should deny")
	}
}

func TestRestrictivePractice_MissingAlternativesDenies(t *testing.T) {
	c := RestrictivePracticeConsent{
		PracticeType:                       PracticeChemicalRestraint,
		Status:                             "active",
		LessRestrictiveAlternativesDocumented: false,
		GrantedAt:                          time.Now(),
		MaxDuration:                        12 * 7 * 24 * time.Hour,
	}
	if c.Allows(time.Now()) {
		t.Errorf("missing less-restrictive-alternatives documentation should deny")
	}
}
```

```go
package consent_extension

import (
	"time"

	"github.com/google/uuid"
)

type PracticeType string

const (
	PracticeChemicalRestraint      PracticeType = "chemical_restraint"
	PracticePhysicalRestraint      PracticeType = "physical_restraint"
	PracticeEnvironmentalRestraint PracticeType = "environmental_restraint"
	PracticeSeclusion              PracticeType = "seclusion"
)

// VisibilityClass: AD (audit-defensible). All transitions traced.
type RestrictivePracticeConsent struct {
	ID                                    uuid.UUID
	ConsentID                             uuid.UUID // FK to Plan 0.2 consents
	PracticeType                          PracticeType
	Status                                string // "requested" / "discussed" / "active" / "expired" / "withdrawn"
	LessRestrictiveAlternativesDocumented bool
	BehaviourSupportPlanRef               *uuid.UUID
	SDMConsentRecordRef                   *uuid.UUID
	GrantedAt                             time.Time
	MaxDuration                           time.Duration // ≤12 weeks for chemical
	DesignatedPractitionerID              uuid.UUID
	MandatoryReviewDueAt                  time.Time
}

func (c RestrictivePracticeConsent) Allows(asOf time.Time) bool {
	if c.Status != "active" {
		return false
	}
	if !c.LessRestrictiveAlternativesDocumented {
		return false
	}
	expiry := c.GrantedAt.Add(c.MaxDuration)
	if asOf.After(expiry) {
		return false
	}
	return true
}
```

- [ ] **Step 4: Migration 038**

```sql
-- 038_restrictive_practice_consent.sql
BEGIN;
CREATE TABLE restrictive_practice_consents (
    id                                        UUID PRIMARY KEY,
    consent_id                                UUID NOT NULL REFERENCES consents(id),
    practice_type                             VARCHAR(32) NOT NULL CHECK (practice_type IN
                                                  ('chemical_restraint','physical_restraint',
                                                   'environmental_restraint','seclusion')),
    status                                    VARCHAR(16) NOT NULL,
    less_restrictive_alternatives_documented  BOOLEAN NOT NULL DEFAULT FALSE,
    behaviour_support_plan_ref                UUID,
    sdm_consent_record_ref                    UUID,
    granted_at                                TIMESTAMPTZ NOT NULL,
    max_duration_hours                        INT NOT NULL,
    designated_practitioner_id                UUID NOT NULL,
    mandatory_review_due_at                   TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_rpc_active ON restrictive_practice_consents (consent_id, practice_type)
    WHERE status = 'active';
COMMIT;
```

- [ ] **Step 5: Commit**

```bash
git commit -m "feat(ethics): restrictive-practice consent gating extending Plan 0.2"
```

---

### Task 8: Pattern detector — acceptance-appropriateness divergence

**Files:**
- Create: `shared/v2_substrate/ethics/pattern_detection/acceptance_appropriateness.go` + `_test.go`

Per Guidelines §1 Principle 2 + §10 daily detection. Worker queries 30-day rolling: any rule where acceptance rises ≥10pp without appropriateness rising in parallel → flag in EthicsLog severity 3.

- [ ] **Step 1-3: Test + implement**

```go
package pattern_detection

import "testing"

func TestDivergence_FlagsRisingAcceptanceFlatAppropriateness(t *testing.T) {
	prior := RuleSnapshot{AcceptanceRate: 0.55, AppropriatenessMean: 3.8}
	current := RuleSnapshot{AcceptanceRate: 0.70, AppropriatenessMean: 3.85}
	if !DetectDivergence(prior, current, 0.10) {
		t.Errorf("expected divergence for +15pp acceptance with flat appropriateness")
	}
}

func TestDivergence_DoesNotFlagCorrelatedRise(t *testing.T) {
	prior := RuleSnapshot{AcceptanceRate: 0.55, AppropriatenessMean: 3.8}
	current := RuleSnapshot{AcceptanceRate: 0.70, AppropriatenessMean: 4.4}
	if DetectDivergence(prior, current, 0.10) {
		t.Errorf("correlated acceptance + appropriateness rise should not divergence-flag")
	}
}
```

```go
package pattern_detection

type RuleSnapshot struct {
	RuleID              string
	AcceptanceRate      float64
	AppropriatenessMean float64
}

// DetectDivergence flags when acceptance rises by ≥thresholdPP percentage points
// without a parallel rise in appropriateness mean (Δ ≥ 0.3 considered "parallel").
func DetectDivergence(prior, current RuleSnapshot, thresholdPP float64) bool {
	deltaAcceptance := current.AcceptanceRate - prior.AcceptanceRate
	deltaAppropriateness := current.AppropriatenessMean - prior.AppropriatenessMean
	if deltaAcceptance < thresholdPP {
		return false
	}
	return deltaAppropriateness < 0.3
}
```

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(ethics): acceptance-appropriateness divergence detector"
```

---

### Task 9: Pattern detector — suppression + surveillance

**Files:**
- Create: `shared/v2_substrate/ethics/pattern_detection/suppression.go` + `_test.go`
- Create: `shared/v2_substrate/ethics/pattern_detection/surveillance.go` + `_test.go`

**Suppression detector:** flags rules whose recommendations are systematically deferred without reasoning. **Surveillance detector:** flags employer queries on individuals exceeding 95th percentile, queries timed to coincide with employment review cycles, aggregation queries enabling re-identification (Guidelines §9.7).

- [ ] **Step 1-3: Test + implement**

```go
// suppression_test.go
package pattern_detection

import "testing"

func TestSuppression_FlagsHighDeferralWithoutReasoning(t *testing.T) {
	if !DetectSuppression(SuppressionInputs{
		TotalRecommendations:        100,
		DeferredCount:               40,
		DeferredWithReasoningCount:  5,
	}, 0.30, 0.20) {
		t.Errorf("expected suppression flag")
	}
}

func TestSuppression_DoesNotFlagBalanced(t *testing.T) {
	if DetectSuppression(SuppressionInputs{
		TotalRecommendations:        100,
		DeferredCount:               20,
		DeferredWithReasoningCount:  18,
	}, 0.30, 0.20) {
		t.Errorf("balanced deferral should not flag")
	}
}
```

```go
// suppression.go
package pattern_detection

type SuppressionInputs struct {
	RuleID                     string
	TotalRecommendations       int
	DeferredCount              int
	DeferredWithReasoningCount int
}

func DetectSuppression(in SuppressionInputs, deferralThreshold, undocumentedThreshold float64) bool {
	if in.TotalRecommendations == 0 {
		return false
	}
	deferralRate := float64(in.DeferredCount) / float64(in.TotalRecommendations)
	if deferralRate < deferralThreshold {
		return false
	}
	undocumented := in.DeferredCount - in.DeferredWithReasoningCount
	if in.DeferredCount == 0 {
		return false
	}
	undocumentedRate := float64(undocumented) / float64(in.DeferredCount)
	return undocumentedRate >= 1-undocumentedThreshold
}
```

```go
// surveillance_test.go
package pattern_detection

import "testing"

func TestSurveillance_FlagsAboveP95IndividualQueries(t *testing.T) {
	if !DetectSurveillanceP95(IndividualQueryRate{Employer: "A", QueryCountP95: 50, EmployerQueryCount: 120}) {
		t.Errorf("expected p95 flag")
	}
}

func TestSurveillance_FlagsReidentificationRisk(t *testing.T) {
	if !DetectReidentificationRisk(AggregationSubset{PharmacistCount: 3}, 5) {
		t.Errorf("subset of 3 below floor of 5 should flag")
	}
}
```

```go
// surveillance.go
package pattern_detection

type IndividualQueryRate struct {
	Employer           string
	EmployerQueryCount int
	QueryCountP95      int
}

func DetectSurveillanceP95(r IndividualQueryRate) bool {
	return r.EmployerQueryCount > r.QueryCountP95
}

type AggregationSubset struct {
	PharmacistCount int
}

func DetectReidentificationRisk(s AggregationSubset, floor int) bool {
	return s.PharmacistCount < floor
}
```

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(ethics): suppression + surveillance pattern detectors"
```

---

### Task 10: Bias detection foundation

**Files:**
- Create: `shared/v2_substrate/ethics/pattern_detection/bias.go` + `bias_test.go`

Per Guidelines §7.2. Demographic stratification of metrics by age band, sex, frailty tier, CALD background, socioeconomic indicator, facility type, geography. Disparity threshold configurable (default 1.5x ratio). Foundation only — full equity audit is Phase 2+.

- [ ] **Step 1-3: Test + implement**

```go
package pattern_detection

import "testing"

func TestBias_FlagsHighDisparityRatio(t *testing.T) {
	stratified := map[string]float64{
		"65-74": 0.60,
		"75-84": 0.55,
		"85+":   0.30, // disparity vs 65-74 = 2.0x
	}
	if !DetectBiasDisparity(stratified, 1.5) {
		t.Errorf("expected disparity flag")
	}
}

func TestBias_DoesNotFlagUniform(t *testing.T) {
	stratified := map[string]float64{"a": 0.5, "b": 0.55, "c": 0.52}
	if DetectBiasDisparity(stratified, 1.5) {
		t.Errorf("uniform stratification should not flag")
	}
}
```

```go
package pattern_detection

func DetectBiasDisparity(stratified map[string]float64, ratioThreshold float64) bool {
	if len(stratified) < 2 {
		return false
	}
	var maxV, minV float64 = -1, -1
	for _, v := range stratified {
		if maxV < 0 || v > maxV {
			maxV = v
		}
		if minV < 0 || v < minV {
			minV = v
		}
	}
	if minV <= 0 {
		return maxV > 0 // any non-zero against zero is infinite disparity
	}
	return (maxV / minV) >= ratioThreshold
}
```

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(ethics): bias detection foundation with configurable disparity threshold"
```

---

### Task 11: Incident classification + hold mechanism + migration 039

**Files:**
- Create: `shared/v2_substrate/ethics/incident_response/classifier.go` + `classifier_test.go`
- Create: `shared/v2_substrate/ethics/incident_response/hold.go` + `hold_test.go`
- Create: `migrations/039_incidents.sql` + rollback

Per Guidelines §11.1–11.2. Severity 1–4. Hold mechanism: any service registers a `HoldHandler`; when an incident opens with severity ≤ 2, all registered handlers are notified to hold the affected component.

- [ ] **Step 1-3: Test + implement**

```go
// classifier_test.go
package incident_response

import "testing"

func TestClassifier_AssignsCorrectSeverity(t *testing.T) {
	cases := []struct {
		kind     string
		expected int
	}{
		{"clinical_safety", 1},
		{"trust_violation", 2},
		{"bias_concern", 3},
		{"procedural", 4},
	}
	for _, tc := range cases {
		if got := Classify(tc.kind); got != tc.expected {
			t.Errorf("kind=%q got %d want %d", tc.kind, got, tc.expected)
		}
	}
}

// hold_test.go
package incident_response

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestHoldOrchestrator_TriggersOnSeverity2(t *testing.T) {
	called := false
	h := NewOrchestrator()
	h.Register(HoldHandlerFunc(func(_ context.Context, _ Incident) error {
		called = true
		return nil
	}))
	_ = h.Trigger(context.Background(), Incident{
		ID: uuid.New(), Severity: 2, Kind: "trust_violation",
	})
	if !called {
		t.Errorf("expected handler invoked at severity 2")
	}
}

func TestHoldOrchestrator_DoesNotTriggerOnSeverity4(t *testing.T) {
	called := false
	h := NewOrchestrator()
	h.Register(HoldHandlerFunc(func(_ context.Context, _ Incident) error {
		called = true
		return nil
	}))
	_ = h.Trigger(context.Background(), Incident{
		ID: uuid.New(), Severity: 4, Kind: "procedural",
	})
	if called {
		t.Errorf("severity-4 should not trigger hold handlers")
	}
}
```

```go
// classifier.go
package incident_response

func Classify(kind string) int {
	switch kind {
	case "clinical_safety":
		return 1
	case "trust_violation":
		return 2
	case "bias_concern":
		return 3
	case "procedural":
		return 4
	}
	return 4 // default conservative
}
```

```go
// hold.go
package incident_response

import (
	"context"

	"github.com/google/uuid"
)

type Incident struct {
	ID                 uuid.UUID
	Severity           int
	Kind               string
	AffectedComponents []string
	HoldActive         bool
}

type HoldHandler interface {
	Hold(ctx context.Context, inc Incident) error
}

type HoldHandlerFunc func(ctx context.Context, inc Incident) error

func (f HoldHandlerFunc) Hold(ctx context.Context, inc Incident) error { return f(ctx, inc) }

type Orchestrator struct {
	handlers []HoldHandler
}

func NewOrchestrator() *Orchestrator { return &Orchestrator{} }

func (o *Orchestrator) Register(h HoldHandler) { o.handlers = append(o.handlers, h) }

func (o *Orchestrator) Trigger(ctx context.Context, inc Incident) error {
	if inc.Severity > 2 {
		return nil // only sev 1+2 trigger holds
	}
	for _, h := range o.handlers {
		if err := h.Hold(ctx, inc); err != nil {
			return err
		}
	}
	return nil
}
```

- [ ] **Step 4: Migration 039**

```sql
-- 039_incidents.sql
BEGIN;
CREATE TABLE incidents (
    id                   UUID PRIMARY KEY,
    severity             INT NOT NULL CHECK (severity BETWEEN 1 AND 4),
    kind                 VARCHAR(64) NOT NULL,
    affected_components  TEXT[] NOT NULL DEFAULT '{}',
    hold_active          BOOLEAN NOT NULL DEFAULT FALSE,
    description          TEXT,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at          TIMESTAMPTZ
);
CREATE INDEX idx_incidents_severity ON incidents (severity);
CREATE INDEX idx_incidents_open     ON incidents (resolved_at) WHERE resolved_at IS NULL;
COMMIT;
```

- [ ] **Step 5: Commit**

```bash
git commit -m "feat(ethics): incident classification + hold orchestrator"
```

---

### Task 12: Integration test — end-to-end ERM intercept

**Files:**
- Create: `shared/v2_substrate/ethics/integration_test.go`

Cross-cutting test: a synthetic recommendation with appropriateness 2.0 is drafted; ERM `recommendation-draft` reasoner intercepts; outcome `Hold`; EthicsLog entry created severity 3; recommendation does not progress to `drafted`; pharmacist sees ERM concern in the response.

- [ ] **Step 1: Write integration test**

```go
package ethics_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/ethics/decision_metadata"
	"github.com/cardiofit/shared/v2_substrate/ethics/erm"
	"github.com/cardiofit/shared/v2_substrate/ethics/erm/reasoners"
	"github.com/cardiofit/shared/v2_substrate/ethics/ethics_log"
)

type fakeDivergence struct{}

func (f *fakeDivergence) IsDivergent(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func TestE2E_LowAppropriatenessHeldByERMAndLogged(t *testing.T) {
	ctx := context.Background()
	logStore := ethics_log.NewInMemoryStore()
	logger := ethics_log.NewLogger(logStore)
	mdStore := decision_metadata.NewInMemoryStore()
	recorder := decision_metadata.NewRecorder(mdStore)

	module := erm.NewModule()
	module.Register(erm.DecisionTypeRecommendationDraft,
		reasoners.NewRecommendationReasoner(3.0, &fakeDivergence{}))

	decisionID := uuid.New()
	dp := erm.DecisionPoint{
		DecisionID:   decisionID,
		Component:    "kb-30",
		DecisionType: erm.DecisionTypeRecommendationDraft,
		ProposedOutput: reasoners.RecommendationProposal{
			AppropriatenessScore: 2.0,
			RuleID:               "R-low-appropriateness",
		},
	}

	outcome, concerns, err := module.Review(ctx, dp)
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if outcome != erm.OutcomeHold {
		t.Fatalf("expected Hold, got %v", outcome)
	}
	if len(concerns) == 0 {
		t.Fatalf("expected non-empty concerns")
	}

	if err := recorder.Record(ctx, decision_metadata.Metadata{
		DecisionID:           decisionID,
		Component:            dp.Component,
		DecisionType:         string(dp.DecisionType),
		AffectedSubjectID:    "resident-X",
		AffectedSubjectClass: "resident",
		PrinciplesImplicated: []string{concerns[0].Principle},
		ERMReviewed:          true,
		ERMOutcome:           ptrStr(string(outcome)),
		ContestationEnabled:  true,
		AuditTraceRef:        uuid.New(),
		Timestamp:            time.Now().UTC(),
	}); err != nil {
		t.Fatalf("record metadata: %v", err)
	}

	if err := logger.Append(ctx, ethics_log.Entry{
		DecisionID:  decisionID,
		EntryType:   ethics_log.EntryTypeConcernFlagged,
		Severity:    3,
		Description: concerns[0].Reasoning,
		Status:      ethics_log.StatusOpen,
	}); err != nil {
		t.Fatalf("append log: %v", err)
	}

	got, err := mdStore.Get(ctx, decisionID)
	if err != nil || got == nil || !got.ERMReviewed {
		t.Errorf("metadata: got=%v err=%v", got, err)
	}

	q := ethics_log.NewQuerier(logStore)
	entries, _ := q.ByDecision(ctx, decisionID)
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
	if entries[0].Severity != 3 || entries[0].Status != ethics_log.StatusOpen {
		t.Errorf("log entry shape: %+v", entries[0])
	}
}

func ptrStr(s string) *string { return &s }
```

- [ ] **Step 2: Run + verify pass**

```
go test -tags=integration ./shared/v2_substrate/ethics/... -run TestE2E -v
# PASS
```

- [ ] **Step 3: Commit**

```bash
git commit -m "test(ethics): end-to-end ERM intercept + metadata + ethics-log integration"
```

---

## Spec coverage

- [x] EthicalDecisionMetadata + EthicsLog substrate — Tasks 1, 2 (Guidelines §14)
- [x] ERM scaffold — Task 3 (Guidelines §4.1–4.6)
- [x] Per-decision-type reasoners (recommendation, visibility) — Tasks 4, 5
- [x] VulnerabilityAssessment on Resident — Task 6 (Guidelines §5.3)
- [x] Restrictive-practice consent gating — Task 7 (Guidelines §6.3)
- [x] Pattern detectors (acceptance-appropriateness, suppression, surveillance) — Tasks 8, 9
- [x] Bias detection foundation — Task 10 (Guidelines §7.2)
- [x] Incident classification + hold mechanism — Task 11 (Guidelines §11.1–11.2)
- [x] End-to-end ERM intercept integration — Task 12

**Out of scope for 1c (deferred to follow-up plans):**
- Authorisation reasoner (Task 4 covers recommendation-draft and Task 5 covers visibility-aggregate; the third reasoner type — authorisation — has an interface placeholder via `DecisionTypeAuthorisation` but no concrete reasoner. Implement when kb-30 needs ERM hooks.)
- Standalone `ethics-monitoring` service binary (`backend/services/ethics-monitoring/`) — package layout reserved in File Structure; the daily/weekly worker implementations are a Phase 2 deliverable wired against the detectors landed here.
- Demographic stratification full coverage — Task 10 lands the disparity-detection primitive; the stratification pipeline that feeds it (age band, sex, frailty tier, CALD, socioeconomic, facility, geography sources) is a Phase 2 follow-up.
- ERM rule refinement loop (quarterly review) — process, not code.

**Out of scope for 1c (operational/process, not code):**
- Ethics Steering Committee charter (governance document; lives in `claudedocs/`)
- Pharmacist Advisory Group constitution (governance)
- Annual external ethics audit specification (process)
- Academic partnership review (process)
- Plain-language summaries (UI copy belongs to Phase 1b surface concern)
- Incident-response runbook with external-communication templates (operational doc; lives in `claudedocs/runbooks/`)

Plan complete and saved.
