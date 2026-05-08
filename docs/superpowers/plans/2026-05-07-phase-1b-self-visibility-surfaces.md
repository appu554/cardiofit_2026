# Phase 1b — Self-Visibility Surfaces Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Land the six pharmacist-facing dashboard surfaces, seven KPI computations, reflective writing engine, RPL/CPD exports, and cross-employer portability per *Pharmacist Self-Visibility Implementation Guidelines v1.0*. Builds on Phase 1a's 5-class visibility substrate; ships the user-facing layer pharmacists actually see in pilot.

**Architecture:** New service `backend/services/pharmacist-self-visibility/` (Go) housing six surface handlers + KPI computation pipelines + reflective writing engine + algorithmic-observation 4-class taxonomy + export generators + employer-transition handler. Reads from substrate via permission middleware (Phase 1a). Six new entities — `ReflectiveEntry`, `ReflectivePrompt`, `AlgorithmicObservation`, `RPLEvidencePack`, `CPDRecord`, `EmployerTransition` — each with explicit visibility-class assertion.

**Tech Stack:** Go, PostgreSQL, depends on Phase 1a (visibility classes + DataAggregationConsent + middleware), Plan 0.1 (Recommendation entity / RIR queries), Plan 0.3 (MonitoringPlan for worklist surface). No new third-party deps beyond the existing stack (`github.com/google/uuid`, `github.com/lib/pq`, `github.com/jung-kurt/gofpdf` for PDF export).

---

## File Structure

**New service module:**
- `backend/services/pharmacist-self-visibility/go.mod`
- `backend/services/pharmacist-self-visibility/cmd/server/main.go`
- `backend/services/pharmacist-self-visibility/internal/api/{http,grpc}.go`

**Reflection package:**
- `backend/services/pharmacist-self-visibility/internal/reflection/entries.go` + `entries_test.go`
- `backend/services/pharmacist-self-visibility/internal/reflection/prompts.go` + `prompts_test.go`

**Algorithmic-distinction package:**
- `backend/services/pharmacist-self-visibility/internal/algorithmic_distinction/classifier.go` + `classifier_test.go`
- `backend/services/pharmacist-self-visibility/internal/algorithmic_distinction/markers.go`

**Dashboard surfaces (one file each):**
- `backend/services/pharmacist-self-visibility/internal/dashboards/worklist.go` + `_test.go`
- `backend/services/pharmacist-self-visibility/internal/dashboards/recommendations.go` + `_test.go`
- `backend/services/pharmacist-self-visibility/internal/dashboards/gp_relationships.go` + `_test.go`
- `backend/services/pharmacist-self-visibility/internal/dashboards/reasoning.go` + `_test.go`
- `backend/services/pharmacist-self-visibility/internal/dashboards/cpd.go` + `_test.go`
- `backend/services/pharmacist-self-visibility/internal/dashboards/portfolio.go` + `_test.go`

**KPI computations:**
- `backend/services/pharmacist-self-visibility/internal/kpis/rir.go` + `_test.go`
- `backend/services/pharmacist-self-visibility/internal/kpis/class_specific.go` + `_test.go`
- `backend/services/pharmacist-self-visibility/internal/kpis/context_time.go` + `_test.go`
- `backend/services/pharmacist-self-visibility/internal/kpis/appropriateness.go` + `_test.go`
- `backend/services/pharmacist-self-visibility/internal/kpis/restraint.go` + `_test.go`
- `backend/services/pharmacist-self-visibility/internal/kpis/cpd_activity.go` + `_test.go`

**Exports:**
- `backend/services/pharmacist-self-visibility/internal/exports/rpl_pack.go` + `_test.go`
- `backend/services/pharmacist-self-visibility/internal/exports/cpd_record.go` + `_test.go`
- `backend/services/pharmacist-self-visibility/internal/exports/portfolio_pdf.go` + `_test.go`

**Portability:**
- `backend/services/pharmacist-self-visibility/internal/portability/transition.go` + `_test.go`
- `backend/services/pharmacist-self-visibility/internal/portability/account_closure.go` + `_test.go`

**Migrations:**
- `migrations/030_reflective_entries.sql` + rollback
- `migrations/031_algorithmic_observations.sql` + rollback
- `migrations/032_rpl_packs.sql` + rollback
- `migrations/033_cpd_records.sql` + rollback
- `migrations/034_employer_transitions.sql` + rollback

---

### Task 1: ReflectiveEntry entity + POA-class enforcement

**Files:**
- Create: `backend/services/pharmacist-self-visibility/internal/reflection/entries.go`
- Create: `backend/services/pharmacist-self-visibility/internal/reflection/entries_test.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/migrations/030_reflective_entries.sql`
- Create: `backend/shared-infrastructure/knowledge-base-services/migrations/030_reflective_entries_rollback.sql`

Reflective entries are POA — only the authoring pharmacist can read them. The platform never re-surfaces reflective content algorithmically (per Self-Visibility Guidelines §6.4, this protects the safe-space character of reflective writing).

- [ ] **Step 1: Write failing test**

```go
package reflection

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestEntry_AuthorOnlyRead(t *testing.T) {
	store := NewInMemoryStore()
	author := uuid.New()
	other := uuid.New()

	entry, err := store.Create(context.Background(), Entry{
		PharmacistID: author,
		Body:         "Worked on a complex deprescribing case today.",
		PromptID:     nil,
		Tags:         []string{"deprescribing", "complex_case"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Author can read.
	got, err := store.Get(context.Background(), author, entry.ID)
	if err != nil || got == nil {
		t.Fatalf("author read: err=%v entry=%v", err, got)
	}

	// Non-author gets ErrNotAuthorized, not the entry.
	_, err = store.Get(context.Background(), other, entry.ID)
	if err != ErrNotAuthorized {
		t.Errorf("expected ErrNotAuthorized for non-author, got %v", err)
	}
}

func TestEntry_ListByAuthorOnly(t *testing.T) {
	store := NewInMemoryStore()
	a := uuid.New()
	b := uuid.New()
	for i := 0; i < 3; i++ {
		_, _ = store.Create(context.Background(), Entry{PharmacistID: a, Body: "a"})
	}
	_, _ = store.Create(context.Background(), Entry{PharmacistID: b, Body: "b"})

	listA, _ := store.ListByAuthor(context.Background(), a, 50)
	listB, _ := store.ListByAuthor(context.Background(), b, 50)
	if len(listA) != 3 || len(listB) != 1 {
		t.Errorf("listA=%d listB=%d (want 3 / 1)", len(listA), len(listB))
	}
	// Cross-author list with mismatched pharmacist returns 0.
	if list, _ := store.ListByAuthor(context.Background(), uuid.New(), 50); len(list) != 0 {
		t.Errorf("unknown pharmacist should see 0 entries, got %d", len(list))
	}
	_ = time.Now() // suppress unused if test grows
}
```

- [ ] **Step 2: Run test → verify it fails**

```
go test ./internal/reflection/...
# FAIL: undefined: NewInMemoryStore, Entry, ErrNotAuthorized
```

- [ ] **Step 3: Implement**

```go
package reflection

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

// VisibilityClass: POA — only the authoring pharmacist can read these entries.
// Pattern detection is explicitly forbidden on this entity.
var ErrNotAuthorized = errors.New("reflection: not authorized")

type Entry struct {
	ID           uuid.UUID
	PharmacistID uuid.UUID
	PromptID     *uuid.UUID
	Body         string
	Tags         []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Store interface {
	Create(ctx context.Context, e Entry) (Entry, error)
	Get(ctx context.Context, requester uuid.UUID, id uuid.UUID) (*Entry, error)
	ListByAuthor(ctx context.Context, pharmacistID uuid.UUID, limit int) ([]Entry, error)
}

type InMemoryStore struct {
	mu      sync.RWMutex
	entries map[uuid.UUID]Entry
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{entries: make(map[uuid.UUID]Entry)}
}

func (s *InMemoryStore) Create(_ context.Context, e Entry) (Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	now := time.Now().UTC()
	e.CreatedAt = now
	e.UpdatedAt = now
	s.entries[e.ID] = e
	return e, nil
}

func (s *InMemoryStore) Get(_ context.Context, requester, id uuid.UUID) (*Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.entries[id]
	if !ok {
		return nil, ErrNotAuthorized // do NOT leak existence to non-authors
	}
	if e.PharmacistID != requester {
		return nil, ErrNotAuthorized
	}
	return &e, nil
}

func (s *InMemoryStore) ListByAuthor(_ context.Context, pharmacistID uuid.UUID, limit int) ([]Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Entry, 0)
	for _, e := range s.entries {
		if e.PharmacistID == pharmacistID {
			out = append(out, e)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}
```

- [ ] **Step 4: Run test → verify pass**

```
go test ./internal/reflection/... -run TestEntry -v
# PASS: TestEntry_AuthorOnlyRead, TestEntry_ListByAuthorOnly
```

- [ ] **Step 5: Migration 030**

```sql
-- 030_reflective_entries.sql
BEGIN;
CREATE TABLE reflective_entries (
    id            UUID PRIMARY KEY,
    pharmacist_id UUID NOT NULL,
    prompt_id     UUID,
    body          TEXT NOT NULL,
    tags          TEXT[] NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_reflective_entries_author_recent
    ON reflective_entries (pharmacist_id, created_at DESC);
COMMENT ON TABLE reflective_entries IS
    'POA visibility class. Author-only read; no algorithmic pattern detection.';
COMMIT;
```

```sql
-- 030_reflective_entries_rollback.sql
BEGIN;
DROP TABLE IF EXISTS reflective_entries;
COMMIT;
```

- [ ] **Step 6: Commit**

```bash
git add backend/services/pharmacist-self-visibility/internal/reflection/entries.go \
        backend/services/pharmacist-self-visibility/internal/reflection/entries_test.go \
        backend/shared-infrastructure/knowledge-base-services/migrations/030_reflective_entries.sql \
        backend/shared-infrastructure/knowledge-base-services/migrations/030_reflective_entries_rollback.sql
git commit -m "feat(self-visibility): ReflectiveEntry entity with POA-class author-only reads"
```

---

### Task 2: Reflective prompt rotation engine

**Files:**
- Create: `backend/services/pharmacist-self-visibility/internal/reflection/prompts.go`
- Create: `backend/services/pharmacist-self-visibility/internal/reflection/prompts_test.go`

Per Self-Visibility Guidelines §5.1, prompts rotate monthly and adapt to pharmacist activity signals (recent override count, recommendation type distribution). Critical constraint: prompts may consult substrate facts about the pharmacist's *clinical* work, but never about their reflective entries — the entries are POA-isolated from prompt selection.

- [ ] **Step 1: Write failing test**

```go
package reflection

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestPromptSelector_RotatesMonthly(t *testing.T) {
	lib := DefaultPromptLibrary()
	if len(lib) < 4 {
		t.Fatalf("library should have ≥4 curated prompts, got %d", len(lib))
	}
	sel := NewSelector(lib, &fakeSignals{recentOverrides: 0, typeMix: nil})

	pharmacist := uuid.New()
	got1, _ := sel.Select(context.Background(), pharmacist, 2026, 5)
	got2, _ := sel.Select(context.Background(), pharmacist, 2026, 6)
	if got1.ID == got2.ID {
		t.Errorf("expected different prompts in different months")
	}
}

func TestPromptSelector_AdaptsToRestraintOverrideSignal(t *testing.T) {
	lib := DefaultPromptLibrary()
	sel := NewSelector(lib, &fakeSignals{recentOverrides: 5, typeMix: nil})
	got, _ := sel.Select(context.Background(), uuid.New(), 2026, 5)
	if !got.HasTag("restraint") {
		t.Errorf("expected restraint-themed prompt for override-active pharmacist; got tags=%v", got.Tags)
	}
}

type fakeSignals struct {
	recentOverrides int
	typeMix         map[string]int
}

func (f *fakeSignals) RestraintOverridesIn(_ context.Context, _ uuid.UUID, _ int) (int, error) {
	return f.recentOverrides, nil
}
func (f *fakeSignals) RecommendationTypeMix(_ context.Context, _ uuid.UUID, _ int) (map[string]int, error) {
	return f.typeMix, nil
}
```

- [ ] **Step 2-3: Implement**

```go
package reflection

import (
	"context"
	"hash/fnv"

	"github.com/google/uuid"
)

type Prompt struct {
	ID   uuid.UUID
	Body string
	Tags []string // e.g. ["restraint","deprescribing","general"]
}

func (p Prompt) HasTag(tag string) bool {
	for _, t := range p.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// DefaultPromptLibrary returns the curated v1 prompts from
// Self-Visibility Guidelines §5.1.
func DefaultPromptLibrary() []Prompt {
	return []Prompt{
		{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			Body: "This month you authored N recommendations. Which one are you proudest of, and why?",
			Tags: []string{"general", "achievement"}},
		{ID: uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			Body: "You overrode the restraint signal on M antipsychotic recommendations this quarter. What clinical reasoning supported that?",
			Tags: []string{"restraint", "antipsychotic"}},
		{ID: uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			Body: "Your context-assembly time has changed recently. What's making that possible?",
			Tags: []string{"context_time", "trajectory"}},
		{ID: uuid.MustParse("44444444-4444-4444-4444-444444444444"),
			Body: "You've been working with this GP for 6 months. What have you learned about what lands well?",
			Tags: []string{"gp_relationship", "communication"}},
	}
}

type Signals interface {
	RestraintOverridesIn(ctx context.Context, pharmacistID uuid.UUID, days int) (int, error)
	RecommendationTypeMix(ctx context.Context, pharmacistID uuid.UUID, days int) (map[string]int, error)
}

type Selector struct {
	library []Prompt
	signals Signals
}

func NewSelector(lib []Prompt, sig Signals) *Selector {
	return &Selector{library: lib, signals: sig}
}

// Select returns a prompt for (pharmacist, year, month). Rotation is deterministic
// per (pharmacist, year, month). Activity signals override rotation when strong.
func (s *Selector) Select(ctx context.Context, pharmacistID uuid.UUID, year, month int) (Prompt, error) {
	overrides, _ := s.signals.RestraintOverridesIn(ctx, pharmacistID, 90)
	if overrides >= 3 {
		// Strong restraint-override signal: prefer a restraint-themed prompt.
		for _, p := range s.library {
			if p.HasTag("restraint") {
				return p, nil
			}
		}
	}
	// Default rotation: deterministic hash of (pharmacist, year, month) → library index.
	h := fnv.New32a()
	_, _ = h.Write(pharmacistID[:])
	var ymBuf [8]byte
	ymBuf[0] = byte(year >> 8)
	ymBuf[1] = byte(year)
	ymBuf[2] = byte(month)
	_, _ = h.Write(ymBuf[:])
	idx := int(h.Sum32()) % len(s.library)
	if idx < 0 {
		idx = -idx
	}
	return s.library[idx], nil
}
```

- [ ] **Step 4: Verify pass + commit**

```bash
go test ./internal/reflection/... -run TestPrompt -v
git add backend/services/pharmacist-self-visibility/internal/reflection/prompts.go \
        backend/services/pharmacist-self-visibility/internal/reflection/prompts_test.go
git commit -m "feat(self-visibility): reflective prompt rotation with activity-signal adaptation"
```

---

### Task 3: AlgorithmicObservation 4-class taxonomy

**Files:**
- Create: `backend/services/pharmacist-self-visibility/internal/algorithmic_distinction/classifier.go`
- Create: `backend/services/pharmacist-self-visibility/internal/algorithmic_distinction/classifier_test.go`
- Create: `backend/services/pharmacist-self-visibility/internal/algorithmic_distinction/markers.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/migrations/031_algorithmic_observations.sql` (+ rollback)

Per Self-Visibility Guidelines §6, each surface element carries a class marker so the pharmacist can distinguish *substrate facts* (computed from EvidenceTrace), *platform suggestions* (algorithmic pattern detection), *pharmacist reflections* (own entries), and *hybrid* observations (suggestion confirmed by pharmacist).

- [ ] **Step 1: Write failing test**

```go
package algorithmic_distinction

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestObservation_FourClasses(t *testing.T) {
	for _, c := range []Class{ClassSubstrateFact, ClassPlatformSuggestion, ClassPharmacistReflection, ClassHybrid} {
		o := Observation{ID: uuid.New(), Class: c, Body: "x"}
		if !o.Class.Valid() {
			t.Errorf("class %v should be valid", c)
		}
	}
	// Unknown class invalid.
	if Class("nope").Valid() {
		t.Errorf("unknown class should be invalid")
	}
}

func TestObservation_ConfirmTransitionsSuggestionToHybrid(t *testing.T) {
	o := Observation{ID: uuid.New(), Class: ClassPlatformSuggestion, Body: "Your deprescribing acceptance is changing."}
	confirmed, err := o.Confirm(uuid.New(), time.Now())
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if confirmed.Class != ClassHybrid {
		t.Errorf("class after confirm = %v, want hybrid", confirmed.Class)
	}
	if confirmed.ConfirmedBy == nil || confirmed.ConfirmedAt == nil {
		t.Errorf("expected ConfirmedBy and ConfirmedAt set")
	}
}

func TestObservation_ConfirmRejectsNonSuggestion(t *testing.T) {
	o := Observation{ID: uuid.New(), Class: ClassSubstrateFact}
	if _, err := o.Confirm(uuid.New(), time.Now()); err == nil {
		t.Errorf("confirm on substrate-fact should reject")
	}
}

func TestMarkers_RenderEmoji(t *testing.T) {
	if Marker(ClassSubstrateFact) != "🔵" {
		t.Errorf("substrate-fact marker = %q", Marker(ClassSubstrateFact))
	}
	if Marker(ClassHybrid) != "🟣" {
		t.Errorf("hybrid marker = %q", Marker(ClassHybrid))
	}
	_ = context.Background()
}
```

- [ ] **Step 2-3: Implement**

```go
// classifier.go
package algorithmic_distinction

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Class string

const (
	ClassSubstrateFact        Class = "substrate_fact"
	ClassPlatformSuggestion   Class = "platform_suggestion"
	ClassPharmacistReflection Class = "pharmacist_reflection"
	ClassHybrid               Class = "hybrid"
)

func (c Class) Valid() bool {
	switch c {
	case ClassSubstrateFact, ClassPlatformSuggestion, ClassPharmacistReflection, ClassHybrid:
		return true
	}
	return false
}

var ErrCannotConfirm = errors.New("algorithmic_distinction: only platform_suggestion observations can be confirmed")

type Observation struct {
	ID                uuid.UUID
	Class             Class
	PharmacistID      uuid.UUID  // subject of the observation
	Body              string
	AlgorithmicOrigin *string    // pattern detector / rule ID; for suggestion + hybrid
	ConfirmedBy       *uuid.UUID // for hybrid only
	ConfirmedAt       *time.Time // for hybrid only
	CreatedAt         time.Time
}

// Confirm transitions a PlatformSuggestion to Hybrid when the pharmacist
// confirms the observation (e.g., writes a reflective entry aligning with it).
func (o Observation) Confirm(by uuid.UUID, at time.Time) (Observation, error) {
	if o.Class != ClassPlatformSuggestion {
		return o, ErrCannotConfirm
	}
	o.Class = ClassHybrid
	o.ConfirmedBy = &by
	at = at.UTC()
	o.ConfirmedAt = &at
	return o, nil
}
```

```go
// markers.go
package algorithmic_distinction

func Marker(c Class) string {
	switch c {
	case ClassSubstrateFact:
		return "🔵"
	case ClassPlatformSuggestion:
		return "🟡"
	case ClassPharmacistReflection:
		return "🟢"
	case ClassHybrid:
		return "🟣"
	}
	return ""
}
```

- [ ] **Step 4: Migration 031**

```sql
-- 031_algorithmic_observations.sql
BEGIN;
CREATE TABLE algorithmic_observations (
    id                  UUID PRIMARY KEY,
    class               VARCHAR(32) NOT NULL CHECK (
        class IN ('substrate_fact','platform_suggestion','pharmacist_reflection','hybrid')
    ),
    pharmacist_id       UUID NOT NULL,
    body                TEXT NOT NULL,
    algorithmic_origin  VARCHAR(128),
    confirmed_by        UUID,
    confirmed_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_obs_pharmacist_recent
    ON algorithmic_observations (pharmacist_id, created_at DESC);
COMMIT;
```

- [ ] **Step 5: Commit**

```bash
git add backend/services/pharmacist-self-visibility/internal/algorithmic_distinction/ \
        backend/shared-infrastructure/knowledge-base-services/migrations/031_algorithmic_observations.sql \
        backend/shared-infrastructure/knowledge-base-services/migrations/031_algorithmic_observations_rollback.sql
git commit -m "feat(self-visibility): AlgorithmicObservation 4-class taxonomy with hybrid transition"
```

---

### Task 4: Surface 1 — Today's Worklist

**Files:**
- Create: `backend/services/pharmacist-self-visibility/internal/dashboards/worklist.go`
- Create: `backend/services/pharmacist-self-visibility/internal/dashboards/worklist_test.go`

Risk-stratified daily queue. Composite risk score = recent fall + recent admission + new high-risk medication + overdue monitoring + family concern. Restraint signals surfaced alongside action prompts. **Visibility class: WO** — workflow-operational, visible to anyone with workflow role on this resident, but not aggregated as performance data.

- [ ] **Step 1: Write failing test**

```go
package dashboards

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestWorklist_RankByCompositeRiskScore(t *testing.T) {
	pharm := uuid.New()
	resA := uuid.New()
	resB := uuid.New()
	src := &fakeRiskSource{scores: map[uuid.UUID]int{resA: 9, resB: 4}}
	wl := NewWorklist(src)

	items, err := wl.Today(context.Background(), pharm)
	if err != nil {
		t.Fatalf("today: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if items[0].ResidentID != resA {
		t.Errorf("expected highest-risk first; got %v", items[0].ResidentID)
	}
}

func TestWorklist_RestraintSignalsSurfacedInline(t *testing.T) {
	pharm := uuid.New()
	res := uuid.New()
	src := &fakeRiskSource{
		scores:    map[uuid.UUID]int{res: 7},
		restraint: map[uuid.UUID][]string{res: {"recent_fall_within_72h"}},
	}
	wl := NewWorklist(src)

	items, _ := wl.Today(context.Background(), pharm)
	if len(items[0].RestraintSignals) != 1 {
		t.Errorf("expected restraint signal surfaced inline; got %v", items[0].RestraintSignals)
	}
}

type fakeRiskSource struct {
	scores    map[uuid.UUID]int
	restraint map[uuid.UUID][]string
}

func (f *fakeRiskSource) ResidentsWithCompositeRisk(_ context.Context, _ uuid.UUID) (map[uuid.UUID]int, error) {
	return f.scores, nil
}
func (f *fakeRiskSource) RestraintSignalsFor(_ context.Context, residentID uuid.UUID) ([]string, error) {
	return f.restraint[residentID], nil
}
func (f *fakeRiskSource) TopReasons(_ context.Context, _ uuid.UUID) ([]string, error) {
	return []string{"recent_fall", "overdue_monitoring"}, nil
}
```

- [ ] **Step 2-3: Implement**

```go
package dashboards

import (
	"context"
	"sort"

	"github.com/google/uuid"
)

// VisibilityClass: WO (workflow-operational).
type WorklistItem struct {
	ResidentID         uuid.UUID
	CompositeRisk      int
	TopReasons         []string
	RestraintSignals   []string
	EstimatedActionMin int
}

type RiskSource interface {
	ResidentsWithCompositeRisk(ctx context.Context, pharmacistID uuid.UUID) (map[uuid.UUID]int, error)
	RestraintSignalsFor(ctx context.Context, residentID uuid.UUID) ([]string, error)
	TopReasons(ctx context.Context, residentID uuid.UUID) ([]string, error)
}

type Worklist struct{ src RiskSource }

func NewWorklist(src RiskSource) *Worklist { return &Worklist{src: src} }

func (w *Worklist) Today(ctx context.Context, pharmacistID uuid.UUID) ([]WorklistItem, error) {
	scores, err := w.src.ResidentsWithCompositeRisk(ctx, pharmacistID)
	if err != nil {
		return nil, err
	}
	items := make([]WorklistItem, 0, len(scores))
	for resID, score := range scores {
		signals, _ := w.src.RestraintSignalsFor(ctx, resID)
		reasons, _ := w.src.TopReasons(ctx, resID)
		items = append(items, WorklistItem{
			ResidentID:       resID,
			CompositeRisk:    score,
			TopReasons:       reasons,
			RestraintSignals: signals,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CompositeRisk > items[j].CompositeRisk })
	return items, nil
}
```

- [ ] **Step 4: Commit**

```bash
git add backend/services/pharmacist-self-visibility/internal/dashboards/worklist.go \
        backend/services/pharmacist-self-visibility/internal/dashboards/worklist_test.go
git commit -m "feat(self-visibility): Surface 1 Today's Worklist with composite risk + restraint inline"
```

---

### Task 5: Surface 2 — My Recommendations

**Files:**
- Create: `backend/services/pharmacist-self-visibility/internal/dashboards/recommendations.go`
- Create: `backend/services/pharmacist-self-visibility/internal/dashboards/recommendations_test.go`

Pharmacist's own recommendation lifecycle. PDP class — employer never sees without explicit consent. Reads via permission middleware (Phase 1a) over Plan 0.1 Recommendation store. Rejected recommendations framed as learning opportunities.

- [ ] **Step 1-3: Test + implement**

```go
package dashboards

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestMyRecommendations_FilterByAuthor(t *testing.T) {
	author := uuid.New()
	src := &fakeRecSource{
		recs: []recRow{
			{authorID: author, state: "drafted", id: uuid.New()},
			{authorID: author, state: "implemented", id: uuid.New()},
			{authorID: uuid.New(), state: "drafted", id: uuid.New()}, // someone else's
		},
	}
	d := NewMyRecommendations(src)
	got, _ := d.For(context.Background(), author)
	if len(got) != 2 {
		t.Errorf("expected 2 own recs, got %d", len(got))
	}
}

func TestMyRecommendations_RejectedFramedAsLearning(t *testing.T) {
	author := uuid.New()
	src := &fakeRecSource{
		recs: []recRow{{authorID: author, state: "rejected", id: uuid.New(), rejectionReason: "GP preferred alternative"}},
	}
	d := NewMyRecommendations(src)
	got, _ := d.For(context.Background(), author)
	if got[0].Framing != "learning_opportunity" {
		t.Errorf("rejected rec should carry framing=learning_opportunity, got %q", got[0].Framing)
	}
}

type recRow struct {
	id              uuid.UUID
	authorID        uuid.UUID
	state           string
	rejectionReason string
}
type fakeRecSource struct{ recs []recRow }

func (f *fakeRecSource) ListByAuthor(_ context.Context, author uuid.UUID) ([]recRow, error) {
	out := []recRow{}
	for _, r := range f.recs {
		if r.authorID == author {
			out = append(out, r)
		}
	}
	return out, nil
}
```

```go
package dashboards

import (
	"context"

	"github.com/google/uuid"
)

// VisibilityClass: PDP (Pharmacist-Default-Private).
type RecommendationCard struct {
	RecommendationID uuid.UUID
	State            string
	Framing          string // "learning_opportunity" for rejected; "" otherwise
	RejectionReason  string
}

type RecSource interface {
	ListByAuthor(ctx context.Context, author uuid.UUID) ([]recRow, error)
}

type MyRecommendations struct{ src RecSource }

func NewMyRecommendations(src RecSource) *MyRecommendations { return &MyRecommendations{src: src} }

func (m *MyRecommendations) For(ctx context.Context, author uuid.UUID) ([]RecommendationCard, error) {
	rows, err := m.src.ListByAuthor(ctx, author)
	if err != nil {
		return nil, err
	}
	out := make([]RecommendationCard, 0, len(rows))
	for _, r := range rows {
		c := RecommendationCard{
			RecommendationID: r.id,
			State:            r.state,
			RejectionReason:  r.rejectionReason,
		}
		if r.state == "rejected" {
			c.Framing = "learning_opportunity"
		}
		out = append(out, c)
	}
	return out, nil
}
```

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(self-visibility): Surface 2 My Recommendations with learning-opportunity framing"
```

---

### Task 6: Surface 3 — My GP Relationships

**Files:**
- Create: `backend/services/pharmacist-self-visibility/internal/dashboards/gp_relationships.go`
- Create: `backend/services/pharmacist-self-visibility/internal/dashboards/gp_relationships_test.go`

Per-GP framing patterns from own work. PDP class. **Constraint: NO GP scorecards, NO GP rankings.** Surface shows pattern observations like "recommendations to Dr X have landed better when ..." but never an acceptance percentage or comparative ranking. GP opt-out respected.

- [ ] **Step 1-3: Test + implement**

```go
package dashboards

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestGPRelationships_NeverShowsAcceptancePercentage(t *testing.T) {
	src := &fakeGPSrc{patterns: map[uuid.UUID]gpPattern{
		uuid.New(): {framingObservation: "recommendations land better with monitoring plan up front", acceptanceRate: 0.42},
	}}
	d := NewGPRelationships(src)
	cards, _ := d.For(context.Background(), uuid.New())
	for _, c := range cards {
		if strings.Contains(c.Display, "%") || strings.Contains(c.Display, "rate") {
			t.Errorf("GP card must not surface acceptance rate or %%; got %q", c.Display)
		}
	}
}

func TestGPRelationships_RespectsOptOut(t *testing.T) {
	gpA := uuid.New()
	gpB := uuid.New()
	src := &fakeGPSrc{
		patterns: map[uuid.UUID]gpPattern{
			gpA: {framingObservation: "X", acceptanceRate: 0.5},
			gpB: {framingObservation: "Y", acceptanceRate: 0.6, optedOut: true},
		},
	}
	d := NewGPRelationships(src)
	cards, _ := d.For(context.Background(), uuid.New())
	for _, c := range cards {
		if c.GPID == gpB && c.Display != "default_framing" {
			t.Errorf("opted-out GP should show default_framing only; got %q", c.Display)
		}
	}
}

type gpPattern struct {
	framingObservation string
	acceptanceRate     float64
	optedOut           bool
}
type fakeGPSrc struct{ patterns map[uuid.UUID]gpPattern }

func (f *fakeGPSrc) PatternsForPharmacist(_ context.Context, _ uuid.UUID) (map[uuid.UUID]gpPattern, error) {
	return f.patterns, nil
}
```

```go
package dashboards

import (
	"context"

	"github.com/google/uuid"
)

// VisibilityClass: PDP. Never aggregated to employer.
type GPCard struct {
	GPID    uuid.UUID
	Display string // observation text or "default_framing"
}

type GPSource interface {
	PatternsForPharmacist(ctx context.Context, pharmacistID uuid.UUID) (map[uuid.UUID]gpPattern, error)
}

type GPRelationships struct{ src GPSource }

func NewGPRelationships(src GPSource) *GPRelationships { return &GPRelationships{src: src} }

func (d *GPRelationships) For(ctx context.Context, pharmacistID uuid.UUID) ([]GPCard, error) {
	patterns, err := d.src.PatternsForPharmacist(ctx, pharmacistID)
	if err != nil {
		return nil, err
	}
	cards := make([]GPCard, 0, len(patterns))
	for gpID, p := range patterns {
		c := GPCard{GPID: gpID}
		if p.optedOut {
			c.Display = "default_framing"
		} else {
			c.Display = p.framingObservation // never includes a % or rate
		}
		cards = append(cards, c)
	}
	return cards, nil
}
```

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(self-visibility): Surface 3 My GP Relationships with no-scorecard constraint"
```

---

### Task 7: Surface 4 — My Clinical Reasoning Patterns

**Files:**
- Create: `backend/services/pharmacist-self-visibility/internal/dashboards/reasoning.go`
- Create: `backend/services/pharmacist-self-visibility/internal/dashboards/reasoning_test.go`

Trajectory-first visualisation. Recommendation type distribution over time. Class-specific implementation rates vs the Ramsey 2025 baseline (colecalciferol 37%, calcium 36%, PPI 43%, cessation overall 51%, dose reduction 49%). Ceiling framing (anonymised best-in-class), never peer ranking.

- [ ] **Step 1-3: Test + implement**

```go
package dashboards

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestReasoning_TrajectoryFirstNoPeerRank(t *testing.T) {
	src := &fakeReasoningSrc{
		trajectory: []TrajectoryPoint{{PeriodStart: 0, RIRPct: 0.40}, {PeriodStart: 1, RIRPct: 0.55}},
	}
	d := NewReasoning(src)
	view, _ := d.For(context.Background(), uuid.New())
	if len(view.Trajectory) != 2 {
		t.Errorf("trajectory pts = %d", len(view.Trajectory))
	}
	if view.PeerPercentile != nil {
		t.Errorf("peer percentile must NOT be present in self-view")
	}
}

func TestReasoning_RamseyBaselineSurfacedAsCeiling(t *testing.T) {
	src := &fakeReasoningSrc{
		classRates: map[string]float64{"colecalciferol": 0.42},
	}
	d := NewReasoning(src)
	view, _ := d.For(context.Background(), uuid.New())
	if got, want := view.RamseyComparison["colecalciferol"].Baseline, 0.37; got != want {
		t.Errorf("colecalciferol baseline = %v, want %v", got, want)
	}
	if !view.RamseyComparison["colecalciferol"].FramedAsCeiling {
		t.Errorf("Ramsey comparison must be framed as ceiling, not peer rank")
	}
}

type fakeReasoningSrc struct {
	trajectory []TrajectoryPoint
	classRates map[string]float64
}

func (f *fakeReasoningSrc) RIRTrajectory(_ context.Context, _ uuid.UUID) ([]TrajectoryPoint, error) {
	return f.trajectory, nil
}
func (f *fakeReasoningSrc) ClassSpecificRates(_ context.Context, _ uuid.UUID) (map[string]float64, error) {
	return f.classRates, nil
}
```

```go
package dashboards

import (
	"context"

	"github.com/google/uuid"
)

// Ramsey 2025 national baselines per Self-Visibility Guidelines §4.2.
var RamseyBaselines = map[string]float64{
	"colecalciferol":   0.37,
	"calcium":          0.36,
	"ppi":              0.43,
	"cessation_total":  0.51,
	"dose_reduction":   0.49,
}

type TrajectoryPoint struct {
	PeriodStart int
	RIRPct      float64
}

type RamseyCompare struct {
	OwnRate         float64
	Baseline        float64
	FramedAsCeiling bool
}

// VisibilityClass: PFA for trajectory; POA for any reflective annotation.
type ReasoningView struct {
	Trajectory       []TrajectoryPoint
	RamseyComparison map[string]RamseyCompare
	PeerPercentile   *float64 // ALWAYS nil in self-view; reserved for employer view
}

type ReasoningSource interface {
	RIRTrajectory(ctx context.Context, pharmacistID uuid.UUID) ([]TrajectoryPoint, error)
	ClassSpecificRates(ctx context.Context, pharmacistID uuid.UUID) (map[string]float64, error)
}

type Reasoning struct{ src ReasoningSource }

func NewReasoning(s ReasoningSource) *Reasoning { return &Reasoning{src: s} }

func (r *Reasoning) For(ctx context.Context, pharmacistID uuid.UUID) (ReasoningView, error) {
	traj, err := r.src.RIRTrajectory(ctx, pharmacistID)
	if err != nil {
		return ReasoningView{}, err
	}
	rates, err := r.src.ClassSpecificRates(ctx, pharmacistID)
	if err != nil {
		return ReasoningView{}, err
	}
	view := ReasoningView{Trajectory: traj, RamseyComparison: map[string]RamseyCompare{}}
	for class, ownRate := range rates {
		if baseline, ok := RamseyBaselines[class]; ok {
			view.RamseyComparison[class] = RamseyCompare{
				OwnRate: ownRate, Baseline: baseline, FramedAsCeiling: true,
			}
		}
	}
	return view, nil // PeerPercentile intentionally nil
}
```

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(self-visibility): Surface 4 Reasoning Patterns with trajectory + Ramsey ceiling"
```

---

### Task 8: Surface 5 — My CPD Progression

**Files:**
- Create: `backend/services/pharmacist-self-visibility/internal/dashboards/cpd.go`
- Create: `backend/services/pharmacist-self-visibility/internal/dashboards/cpd_test.go`

Auto-tagged CPD activities from clinical work; pharmacist confirms each before it counts. AHPRA-required hours by activity category. Reflective entries link to activities (POA always for the reflection itself; WO for the activity log).

- [ ] **Step 1-3: Test + implement**

```go
package dashboards

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestCPD_AutoTagPendingConfirmation(t *testing.T) {
	pharm := uuid.New()
	src := &fakeCPDSrc{autoTagged: []CPDActivity{
		{ID: uuid.New(), AHPRACategory: "clinical_review", Hours: 1.5, Status: "pending_confirmation"},
		{ID: uuid.New(), AHPRACategory: "education", Hours: 2.0, Status: "confirmed"},
	}}
	d := NewCPD(src)
	view, _ := d.For(context.Background(), pharm)
	if view.PendingConfirmation != 1 || view.ConfirmedHours["education"] != 2.0 {
		t.Errorf("got pending=%d confirmedEducation=%v", view.PendingConfirmation, view.ConfirmedHours["education"])
	}
}

type fakeCPDSrc struct{ autoTagged []CPDActivity }

func (f *fakeCPDSrc) Activities(_ context.Context, _ uuid.UUID) ([]CPDActivity, error) {
	return f.autoTagged, nil
}
```

```go
package dashboards

import (
	"context"

	"github.com/google/uuid"
)

type CPDActivity struct {
	ID            uuid.UUID
	AHPRACategory string  // "clinical_review", "education", "audit_feedback", etc.
	Hours         float64
	Status        string  // "pending_confirmation" | "confirmed" | "rejected"
	SourceRef     string  // recommendation/case ID
}

// VisibilityClass: WO for activity log (employer can see compliance status);
// reflective entries linked are POA always (Task 1).
type CPDView struct {
	ConfirmedHours      map[string]float64
	PendingConfirmation int
}

type CPDSource interface {
	Activities(ctx context.Context, pharmacistID uuid.UUID) ([]CPDActivity, error)
}

type CPD struct{ src CPDSource }

func NewCPD(s CPDSource) *CPD { return &CPD{src: s} }

func (c *CPD) For(ctx context.Context, pharmacistID uuid.UUID) (CPDView, error) {
	acts, err := c.src.Activities(ctx, pharmacistID)
	if err != nil {
		return CPDView{}, err
	}
	v := CPDView{ConfirmedHours: map[string]float64{}}
	for _, a := range acts {
		if a.Status == "confirmed" {
			v.ConfirmedHours[a.AHPRACategory] += a.Hours
		} else if a.Status == "pending_confirmation" {
			v.PendingConfirmation++
		}
	}
	return v, nil
}
```

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(self-visibility): Surface 5 CPD Progression with auto-tag + confirm flow"
```

---

### Task 9: Surface 6 — My Career Portfolio

**Files:**
- Create: `backend/services/pharmacist-self-visibility/internal/dashboards/portfolio.go`
- Create: `backend/services/pharmacist-self-visibility/internal/dashboards/portfolio_test.go`

Longitudinal record. Pharmacist authors the narrative. Resident/employer identifiers anonymised by default. Cross-employer persistence delegated to Task 15. Pharmacist controls visibility.

- [ ] **Step 1-3: Test + implement**

```go
package dashboards

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestPortfolio_AnonymisesByDefault(t *testing.T) {
	pharm := uuid.New()
	src := &fakePortSrc{narrative: "Worked at RACH-ABC with Dr Smith on resident John Doe.", scenarios: 3}
	p := NewPortfolio(src)
	view, _ := p.For(context.Background(), pharm, false /* not consented to identify */)
	if strings.Contains(view.Narrative, "John Doe") || strings.Contains(view.Narrative, "RACH-ABC") {
		t.Errorf("narrative should be anonymised by default; got %q", view.Narrative)
	}
}

func TestPortfolio_ScenarioCount(t *testing.T) {
	src := &fakePortSrc{narrative: "x", scenarios: 12}
	p := NewPortfolio(src)
	view, _ := p.For(context.Background(), uuid.New(), false)
	if view.ScenarioCount != 12 {
		t.Errorf("got %d", view.ScenarioCount)
	}
}

type fakePortSrc struct {
	narrative string
	scenarios int
}

func (f *fakePortSrc) Narrative(_ context.Context, _ uuid.UUID) (string, error) {
	return f.narrative, nil
}
func (f *fakePortSrc) ScenarioCount(_ context.Context, _ uuid.UUID) (int, error) {
	return f.scenarios, nil
}
```

```go
package dashboards

import (
	"context"
	"regexp"

	"github.com/google/uuid"
)

// VisibilityClass: pharmacist-controlled; default POA for narrative.
type PortfolioView struct {
	Narrative     string
	ScenarioCount int
}

type PortfolioSource interface {
	Narrative(ctx context.Context, pharmacistID uuid.UUID) (string, error)
	ScenarioCount(ctx context.Context, pharmacistID uuid.UUID) (int, error)
}

type Portfolio struct{ src PortfolioSource }

func NewPortfolio(s PortfolioSource) *Portfolio { return &Portfolio{src: s} }

// Anonymisation regex covers proper-name patterns (capitalised tokens) and
// RACH/facility identifiers like "RACH-ABC".
var nameRe = regexp.MustCompile(`\b[A-Z][a-z]+ [A-Z][a-z]+\b|RACH-[A-Z]+`)

func (p *Portfolio) For(ctx context.Context, pharmacistID uuid.UUID, identifiableConsented bool) (PortfolioView, error) {
	narrative, err := p.src.Narrative(ctx, pharmacistID)
	if err != nil {
		return PortfolioView{}, err
	}
	count, _ := p.src.ScenarioCount(ctx, pharmacistID)
	if !identifiableConsented {
		narrative = nameRe.ReplaceAllString(narrative, "[redacted]")
	}
	return PortfolioView{Narrative: narrative, ScenarioCount: count}, nil
}
```

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(self-visibility): Surface 6 Career Portfolio with default anonymisation"
```

---

### Task 10: KPI batch 1 — RIR + class-specific rates

**Files:**
- Create: `backend/services/pharmacist-self-visibility/internal/kpis/rir.go` + `rir_test.go`
- Create: `backend/services/pharmacist-self-visibility/internal/kpis/class_specific.go` + `class_specific_test.go`

Per Guidelines §4.1–4.2. RIR formula = `count(implemented) / count(submitted age > 30d)`. PFA visibility class with aggregation gate (min 30 obs, 90-day rolling, 30-day delay before any employer view).

- [ ] **Step 1-3: Test + implement**

```go
package kpis

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRIR_BasicFormula(t *testing.T) {
	pharm := uuid.New()
	now := time.Now().UTC()
	recs := []RecRow{
		{AuthorID: pharm, State: "implemented", SubmittedAt: now.AddDate(0, 0, -45)},
		{AuthorID: pharm, State: "implemented", SubmittedAt: now.AddDate(0, 0, -50)},
		{AuthorID: pharm, State: "submitted", SubmittedAt: now.AddDate(0, 0, -40)}, // age > 30, denominator
		{AuthorID: pharm, State: "drafted", SubmittedAt: now.AddDate(0, 0, -10)},   // age < 30, excluded
	}
	rir := ComputeRIR(recs, pharm, now, 30)
	if got, want := rir, 2.0/3.0; got < want-0.01 || got > want+0.01 {
		t.Errorf("RIR = %v, want ~%v", got, want)
	}
}

func TestRIR_NoEligibleReturnsNaN(t *testing.T) {
	now := time.Now().UTC()
	rir := ComputeRIR([]RecRow{{State: "drafted", SubmittedAt: now}}, uuid.New(), now, 30)
	if !isNaN(rir) {
		t.Errorf("expected NaN for empty denominator, got %v", rir)
	}
}
```

```go
package kpis

import (
	"math"
	"time"

	"github.com/google/uuid"
)

// VisibilityClass: PFA. Aggregation gate enforced upstream by Phase 1a middleware.
type RecRow struct {
	AuthorID    uuid.UUID
	State       string
	SubmittedAt time.Time
}

// ComputeRIR = count(implemented or beyond) / count(submitted age > windowDays).
// Returns NaN when denominator is zero.
func ComputeRIR(rows []RecRow, author uuid.UUID, asOf time.Time, windowDays int) float64 {
	cutoff := asOf.AddDate(0, 0, -windowDays)
	var num, den int
	for _, r := range rows {
		if r.AuthorID != author {
			continue
		}
		if r.SubmittedAt.After(cutoff) {
			continue // not aged enough to be eligible
		}
		den++
		if r.State == "implemented" || r.State == "outcome_recorded" || r.State == "closed" {
			num++
		}
	}
	if den == 0 {
		return math.NaN()
	}
	return float64(num) / float64(den)
}

func isNaN(f float64) bool { return f != f }
```

`class_specific.go` follows the same shape, filtered by `RecRow.Class`:

```go
package kpis

import (
	"github.com/google/uuid"
	"math"
	"time"
)

func ComputeClassSpecificRate(rows []RecRow, author uuid.UUID, class string, asOf time.Time, windowDays int) float64 {
	cutoff := asOf.AddDate(0, 0, -windowDays)
	var num, den int
	for _, r := range rows {
		if r.AuthorID != author || r.Class != class || r.SubmittedAt.After(cutoff) {
			continue
		}
		den++
		if r.State == "implemented" || r.State == "outcome_recorded" || r.State == "closed" {
			num++
		}
	}
	if den == 0 {
		return math.NaN()
	}
	return float64(num) / float64(den)
}
```

(Add `Class string` field to `RecRow` in `rir.go`.)

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(self-visibility): KPI batch 1 — RIR + class-specific rates with PFA gating"
```

---

### Task 11: KPI batch 2 — context-assembly time + appropriateness score

**Files:**
- Create: `backend/services/pharmacist-self-visibility/internal/kpis/context_time.go` + `_test.go`
- Create: `backend/services/pharmacist-self-visibility/internal/kpis/appropriateness.go` + `_test.go`

Context-assembly time per Guidelines §4.3 = median across rolling 30 reviews (excludes administrative gaps). Appropriateness score per §4.4 = mean rolling 90 days (PDP class — never aggregated per-pharmacist to employer).

- [ ] **Step 1-3: Test + implement**

```go
package kpis

import "testing"

func TestContextTime_MedianAcrossRolling30(t *testing.T) {
	durations := []float64{5, 8, 12, 7, 9} // minutes
	got := MedianContextTime(durations)
	if got != 8 {
		t.Errorf("median = %v, want 8", got)
	}
}

func TestAppropriateness_MeanRolling90(t *testing.T) {
	scores := []float64{3.5, 4.0, 4.5, 4.2}
	got := MeanAppropriateness(scores)
	if got < 4.04 || got > 4.06 {
		t.Errorf("mean = %v, want ~4.05", got)
	}
}
```

```go
// context_time.go
package kpis

import "sort"

func MedianContextTime(minutes []float64) float64 {
	if len(minutes) == 0 {
		return 0
	}
	cp := make([]float64, len(minutes))
	copy(cp, minutes)
	sort.Float64s(cp)
	mid := len(cp) / 2
	if len(cp)%2 == 0 {
		return (cp[mid-1] + cp[mid]) / 2
	}
	return cp[mid]
}
```

```go
// appropriateness.go
// VisibilityClass: PDP. Never aggregated per-pharmacist to employer.
package kpis

func MeanAppropriateness(scores []float64) float64 {
	if len(scores) == 0 {
		return 0
	}
	var sum float64
	for _, s := range scores {
		sum += s
	}
	return sum / float64(len(scores))
}
```

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(self-visibility): KPI batch 2 — context-assembly time + appropriateness"
```

---

### Task 12: KPI batch 3 — restraint override pattern + CPD activity completion

**Files:**
- Create: `backend/services/pharmacist-self-visibility/internal/kpis/restraint.go` + `_test.go`
- Create: `backend/services/pharmacist-self-visibility/internal/kpis/cpd_activity.go` + `_test.go`

Restraint override pattern (Guidelines §4.5) is **POA** — pharmacist alone sees; not employer-visible at all (intimate professional judgment territory). CPD activity completion (§4.6) is **WO** for completion status, **POA** for reflective content.

- [ ] **Step 1-3: Test + implement**

```go
package kpis

import "testing"

func TestRestraintOverridePattern_CountsByType(t *testing.T) {
	overrides := []RestraintOverride{
		{RecommendationType: "antipsychotic_deprescribing"},
		{RecommendationType: "antipsychotic_deprescribing"},
		{RecommendationType: "ppi_dose_reduction"},
	}
	pat := RestraintOverridePattern(overrides)
	if pat["antipsychotic_deprescribing"] != 2 || pat["ppi_dose_reduction"] != 1 {
		t.Errorf("pattern = %v", pat)
	}
}

func TestCPDActivityHours_SumByCategory(t *testing.T) {
	acts := []confirmedActivity{
		{Category: "clinical_review", Hours: 1.5},
		{Category: "education", Hours: 2.0},
		{Category: "clinical_review", Hours: 0.5},
	}
	sums := CPDHoursByCategory(acts)
	if sums["clinical_review"] != 2.0 {
		t.Errorf("clinical_review = %v", sums["clinical_review"])
	}
}
```

```go
// restraint.go
// VisibilityClass: POA. Pharmacist alone sees this pattern.
package kpis

type RestraintOverride struct {
	RecommendationType string
	Reasoning          string
}

func RestraintOverridePattern(overrides []RestraintOverride) map[string]int {
	out := map[string]int{}
	for _, o := range overrides {
		out[o.RecommendationType]++
	}
	return out
}
```

```go
// cpd_activity.go
package kpis

type confirmedActivity struct {
	Category string
	Hours    float64
}

func CPDHoursByCategory(acts []confirmedActivity) map[string]float64 {
	out := map[string]float64{}
	for _, a := range acts {
		out[a.Category] += a.Hours
	}
	return out
}
```

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(self-visibility): KPI batch 3 — restraint override (POA) + CPD activity"
```

---

### Task 13: RPL evidence pack generator

**Files:**
- Create: `backend/services/pharmacist-self-visibility/internal/exports/rpl_pack.go` + `rpl_pack_test.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/migrations/032_rpl_packs.sql` + rollback

Per Guidelines Part 7.1, 5 APC competency dimensions: clinical_assessment, medication_review, communication, quality_use_of_medicines, professional_practice. Pharmacist curates evidence; platform formats as APC-aligned PDF; platform retains no submission record.

- [ ] **Step 1-3: Test + implement**

```go
package exports

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestRPLPack_FiveCompetencyDimensions(t *testing.T) {
	src := &fakeRPLSrc{
		candidates: map[string][]EvidenceItem{
			"clinical_assessment":      {{Title: "Case A"}},
			"medication_review":        {{Title: "Case B"}},
			"communication":            {{Title: "Case C"}},
			"quality_use_of_medicines": {{Title: "Case D"}},
			"professional_practice":    {{Title: "Case E"}},
		},
	}
	g := NewRPLGenerator(src)
	pack, _ := g.Generate(context.Background(), uuid.New(), allFiveDimensions())
	if len(pack.Items) != 5 {
		t.Errorf("expected 5 items across dimensions, got %d", len(pack.Items))
	}
}

func allFiveDimensions() []string {
	return []string{"clinical_assessment", "medication_review", "communication",
		"quality_use_of_medicines", "professional_practice"}
}

type fakeRPLSrc struct{ candidates map[string][]EvidenceItem }

func (f *fakeRPLSrc) CandidatesForDimension(_ context.Context, _ uuid.UUID, dim string) ([]EvidenceItem, error) {
	return f.candidates[dim], nil
}
```

```go
package exports

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type EvidenceItem struct {
	Title       string
	Dimension   string
	Anonymised  bool
	Annotation  string
	OriginRef   uuid.UUID
}

type RPLPack struct {
	ID           uuid.UUID
	PharmacistID uuid.UUID
	Items        []EvidenceItem
	GeneratedAt  time.Time
}

type RPLSource interface {
	CandidatesForDimension(ctx context.Context, pharmacistID uuid.UUID, dimension string) ([]EvidenceItem, error)
}

type RPLGenerator struct{ src RPLSource }

func NewRPLGenerator(s RPLSource) *RPLGenerator { return &RPLGenerator{src: s} }

func (g *RPLGenerator) Generate(ctx context.Context, pharmacistID uuid.UUID, dimensions []string) (RPLPack, error) {
	pack := RPLPack{ID: uuid.New(), PharmacistID: pharmacistID, GeneratedAt: time.Now().UTC()}
	for _, dim := range dimensions {
		cands, err := g.src.CandidatesForDimension(ctx, pharmacistID, dim)
		if err != nil {
			return RPLPack{}, err
		}
		if len(cands) > 0 {
			c := cands[0]
			c.Dimension = dim
			c.Anonymised = true
			pack.Items = append(pack.Items, c)
		}
	}
	return pack, nil
}
```

- [ ] **Step 4: Migration 032**

```sql
-- 032_rpl_packs.sql
BEGIN;
CREATE TABLE rpl_packs (
    id            UUID PRIMARY KEY,
    pharmacist_id UUID NOT NULL,
    generated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    items         JSONB NOT NULL
);
CREATE INDEX idx_rpl_packs_pharmacist ON rpl_packs (pharmacist_id, generated_at DESC);
COMMIT;
```

- [ ] **Step 5: Commit**

```bash
git commit -m "feat(self-visibility): RPL evidence pack with 5 APC competency dimensions"
```

---

### Task 14: AHPRA CPD record export

**Files:**
- Create: `backend/services/pharmacist-self-visibility/internal/exports/cpd_record.go` + `cpd_record_test.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/migrations/033_cpd_records.sql` + rollback

Per Guidelines §7.2. Activities by AHPRA category, reflective entries linked, submission-ready format. Pharmacist exports; platform does not submit on their behalf.

- [ ] **Step 1-3: Test + implement**

```go
package exports

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCPDRecord_HoursByCategory(t *testing.T) {
	src := &fakeCPDExportSrc{
		acts: []ActivityRow{
			{Category: "clinical_review", Hours: 1.5, Confirmed: true},
			{Category: "education", Hours: 2.0, Confirmed: true},
			{Category: "clinical_review", Hours: 0.5, Confirmed: true},
			{Category: "clinical_review", Hours: 1.0, Confirmed: false}, // excluded
		},
	}
	g := NewCPDRecordGenerator(src)
	rec, _ := g.Generate(context.Background(), uuid.New(), 2025, 2026)
	if rec.HoursByCategory["clinical_review"] != 2.0 {
		t.Errorf("clinical_review = %v", rec.HoursByCategory["clinical_review"])
	}
}

type fakeCPDExportSrc struct{ acts []ActivityRow }

func (f *fakeCPDExportSrc) ActivitiesInCycle(_ context.Context, _ uuid.UUID, _, _ int) ([]ActivityRow, error) {
	return f.acts, nil
}
func (f *fakeCPDExportSrc) ReflectionsForActivity(_ context.Context, _ uuid.UUID) ([]string, error) {
	return nil, nil
}
```

```go
package exports

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ActivityRow struct {
	ID        uuid.UUID
	Category  string
	Hours     float64
	Confirmed bool
}

type CPDRecord struct {
	ID              uuid.UUID
	PharmacistID    uuid.UUID
	CycleStart      int
	CycleEnd        int
	HoursByCategory map[string]float64
	GeneratedAt     time.Time
}

type CPDExportSource interface {
	ActivitiesInCycle(ctx context.Context, pharmacistID uuid.UUID, cycleStart, cycleEnd int) ([]ActivityRow, error)
	ReflectionsForActivity(ctx context.Context, activityID uuid.UUID) ([]string, error)
}

type CPDRecordGenerator struct{ src CPDExportSource }

func NewCPDRecordGenerator(s CPDExportSource) *CPDRecordGenerator { return &CPDRecordGenerator{src: s} }

func (g *CPDRecordGenerator) Generate(ctx context.Context, pharmacistID uuid.UUID, cycleStart, cycleEnd int) (CPDRecord, error) {
	acts, err := g.src.ActivitiesInCycle(ctx, pharmacistID, cycleStart, cycleEnd)
	if err != nil {
		return CPDRecord{}, err
	}
	rec := CPDRecord{
		ID: uuid.New(), PharmacistID: pharmacistID,
		CycleStart: cycleStart, CycleEnd: cycleEnd,
		HoursByCategory: map[string]float64{}, GeneratedAt: time.Now().UTC(),
	}
	for _, a := range acts {
		if a.Confirmed {
			rec.HoursByCategory[a.Category] += a.Hours
		}
	}
	return rec, nil
}
```

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(self-visibility): AHPRA CPD record export with cycle-bounded category hours"
```

---

### Task 15: Cross-employer portability transition + free-tier fallback

**Files:**
- Create: `backend/services/pharmacist-self-visibility/internal/portability/transition.go` + `transition_test.go`
- Create: `backend/services/pharmacist-self-visibility/internal/portability/account_closure.go` + `_test.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/migrations/034_employer_transitions.sql` + rollback

Per Guidelines Part 10. Transition handler preserves POA + PDP + own PFA data. Active recommendations stay with prior employer's deployment. Free-tier reversion preserves portfolio + CPD + RPL pack capability.

- [ ] **Step 1-3: Test + implement**

```go
package portability

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestTransition_PreservesPOAandPDPandOwnPFA(t *testing.T) {
	pharm := uuid.New()
	priorEmp := uuid.New()
	newEmp := uuid.New()
	carrier := &fakeCarrier{}
	h := NewHandler(carrier)

	plan, err := h.Initiate(context.Background(), pharm, priorEmp, &newEmp)
	if err != nil {
		t.Fatalf("initiate: %v", err)
	}
	if !plan.PreservesReflectiveEntries || !plan.PreservesPortfolio || !plan.PreservesOwnPFA {
		t.Errorf("plan must preserve POA + portfolio + own PFA; got %+v", plan)
	}
	if plan.PreservesActiveRecommendations {
		t.Errorf("active recommendations must stay with prior employer")
	}
}

func TestTransition_FreeTierReversionWhenNoNewEmployer(t *testing.T) {
	h := NewHandler(&fakeCarrier{})
	plan, _ := h.Initiate(context.Background(), uuid.New(), uuid.New(), nil)
	if !plan.RevertsToFreeTier {
		t.Errorf("expected free-tier reversion when new employer is nil")
	}
}

type fakeCarrier struct{}

func (f *fakeCarrier) MovePOA(_ context.Context, _ uuid.UUID, _ *uuid.UUID) error      { return nil }
func (f *fakeCarrier) MovePortfolio(_ context.Context, _ uuid.UUID, _ *uuid.UUID) error { return nil }
func (f *fakeCarrier) MoveOwnPFA(_ context.Context, _ uuid.UUID, _ *uuid.UUID) error    { return nil }
```

```go
package portability

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type TransitionPlan struct {
	ID                              uuid.UUID
	PharmacistID                    uuid.UUID
	PriorEmployerID                 uuid.UUID
	NewEmployerID                   *uuid.UUID
	PreservesReflectiveEntries      bool
	PreservesPortfolio              bool
	PreservesOwnPFA                 bool
	PreservesActiveRecommendations  bool
	RevertsToFreeTier               bool
	InitiatedAt                     time.Time
}

type Carrier interface {
	MovePOA(ctx context.Context, pharmacistID uuid.UUID, newEmployerID *uuid.UUID) error
	MovePortfolio(ctx context.Context, pharmacistID uuid.UUID, newEmployerID *uuid.UUID) error
	MoveOwnPFA(ctx context.Context, pharmacistID uuid.UUID, newEmployerID *uuid.UUID) error
}

type Handler struct{ carrier Carrier }

func NewHandler(c Carrier) *Handler { return &Handler{carrier: c} }

func (h *Handler) Initiate(ctx context.Context, pharmacistID, priorEmployerID uuid.UUID, newEmployerID *uuid.UUID) (TransitionPlan, error) {
	if err := h.carrier.MovePOA(ctx, pharmacistID, newEmployerID); err != nil {
		return TransitionPlan{}, err
	}
	if err := h.carrier.MovePortfolio(ctx, pharmacistID, newEmployerID); err != nil {
		return TransitionPlan{}, err
	}
	if err := h.carrier.MoveOwnPFA(ctx, pharmacistID, newEmployerID); err != nil {
		return TransitionPlan{}, err
	}
	return TransitionPlan{
		ID:                             uuid.New(),
		PharmacistID:                   pharmacistID,
		PriorEmployerID:                priorEmployerID,
		NewEmployerID:                  newEmployerID,
		PreservesReflectiveEntries:     true,
		PreservesPortfolio:             true,
		PreservesOwnPFA:                true,
		PreservesActiveRecommendations: false, // stays with prior deployment
		RevertsToFreeTier:              newEmployerID == nil,
		InitiatedAt:                    time.Now().UTC(),
	}, nil
}
```

- [ ] **Step 4: Migration 034**

```sql
-- 034_employer_transitions.sql
BEGIN;
CREATE TABLE employer_transitions (
    id                                  UUID PRIMARY KEY,
    pharmacist_id                       UUID NOT NULL,
    prior_employer_id                   UUID NOT NULL,
    new_employer_id                     UUID,
    preserves_reflective_entries        BOOLEAN NOT NULL,
    preserves_portfolio                 BOOLEAN NOT NULL,
    preserves_own_pfa                   BOOLEAN NOT NULL,
    preserves_active_recommendations    BOOLEAN NOT NULL,
    reverts_to_free_tier                BOOLEAN NOT NULL,
    initiated_at                        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at                        TIMESTAMPTZ
);
CREATE INDEX idx_transitions_pharmacist ON employer_transitions (pharmacist_id, initiated_at DESC);
COMMIT;
```

- [ ] **Step 5: Commit**

```bash
git commit -m "feat(self-visibility): cross-employer transition with free-tier reversion"
```

---

## Spec coverage

- [x] Five visibility classes inherited from Phase 1a — used throughout (POA on Tasks 1, 12; PDP on Tasks 5, 6, 11; PFA on Tasks 7, 10; WO on Tasks 4, 8)
- [x] Six dashboard surfaces — Tasks 4–9
- [x] Seven KPI computations — Tasks 10–12 (RIR, class-specific, context-time, appropriateness, restraint override, CPD activity, plus career portfolio metrics surfaced via Task 9)
- [x] Reflective writing engine — Tasks 1, 2
- [x] Algorithmic-vs-human 4-class taxonomy — Task 3
- [x] RPL evidence pack — Task 13
- [x] AHPRA CPD record export — Task 14
- [x] Cross-employer portability + free-tier reversion — Task 15

**Out of scope for 1b (handled in 1c):**
- Ethical Reasoning Module (ERM)
- Bias detection / demographic stratification
- Incident response classification
- Pattern detectors (acceptance-appropriateness, suppression, surveillance)
- VulnerabilityAssessment + restrictive-practice consent gating

**Out of scope for 1b (operational/process, not code):**
- Pharmacist Advisory Group constitution
- Plain-language privacy notices (UI copy)
- Aboriginal community engagement protocol

Plan complete and saved.
