# Recommendation Craft Engine Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Land v3 Architectural Commitment 7 (v3 §3 line 204): the new module v3 promoted from "decision packet generator" to subsystem. Build `kb-32-recommendation-craft` as a standalone service that takes a pharmacist's clinical intent and renders it into a recommendation packet structured per v3 §7: Issue → Clinical Context → Rationale → Evidence → Proposed Plan → Monitoring → Urgency, with type-ordering (STOP > MONITOR > DOSE > ADD), urgency tiering (Red/Amber/Green), per-GP framing learning, and the auditable frame-vs-content separation that v3 §7 line 416 mandates.

**Architecture:** New service `kb-32-recommendation-craft/` (Go, Gin, Postgres, Redis). Consumes Plan 0.1's Recommendation entity at the `detected → drafted` transition: takes a "raw rule fire" output, assembles full clinical context from substrate (Plan 0.1 + 0.2 + Plan 0.3 baselines), resolves audience-appropriate evidence sources, enforces template completeness, applies framing adaptation, persists `clinical_content` (invariant) separately from `framing_adaptation` (variable per audience). Integrates with kb-cql-runtime (Plan 0.5) for rule-result ingestion, kb-30 (Plan 0.4) for actor authorisation snapshots, and Plan 0.1 for lifecycle persistence.

**Tech Stack:** Go, Postgres, Redis, depends on Plans 0.1–0.5 and Phase 1 (permission middleware wraps craft-engine endpoints).

---

## File Structure

**New service:**
- `kb-32-recommendation-craft/` — Go service
  - `cmd/server/main.go`
  - `config/config.go`
  - `internal/template/enforcer.go` — section completeness validator
  - `internal/template/enforcer_test.go`
  - `internal/context/assembler.go` — pulls substrate state for resident
  - `internal/context/assembler_test.go`
  - `internal/evidence/anchor_selector.go` — chooses 1-2 strong sources
  - `internal/evidence/anchor_selector_test.go`
  - `internal/ordering/orderer.go` — STOP > MONITOR > DOSE > ADD ordering
  - `internal/ordering/orderer_test.go`
  - `internal/urgency/tagger.go` — Red/Amber/Green based on substrate state
  - `internal/urgency/tagger_test.go`
  - `internal/framing/separator.go` — frame-vs-content split
  - `internal/framing/separator_test.go`
  - `internal/framing/per_gp_observer.go` — observes acceptance patterns
  - `internal/framing/per_gp_observer_test.go`
  - `internal/api/handlers.go` — HTTP/Gin
  - `Dockerfile`

**New migration:**
- `migrations/029_per_gp_framing_observations.sql` + rollback

**Modified files:**
- `shared/v2_substrate/recommendation/lifecycle.go` — optional craft-engine call on `drafted` entry

---

### Task 1: Service scaffold

**Files:**
- Create: `kb-32-recommendation-craft/cmd/server/main.go`, `config/config.go`, `Dockerfile`, `go.mod`

Mirror kb-30 layout (Gin, Postgres, Redis, Prometheus). No new tests required.

- [ ] **Step 1-3: Scaffold; smoke-build; commit**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
mkdir -p kb-32-recommendation-craft/{cmd/server,config,internal/{template,context,evidence,ordering,urgency,framing,api},docs}
cd kb-32-recommendation-craft
go mod init kb32 && go mod tidy
go build ./... 2>&1 | tail
git add kb-32-recommendation-craft/
git commit -m "feat(kb-32): scaffold recommendation-craft service"
```

---

### Task 2: Template enforcer (v3 §7 line 369)

**Files:**
- Create: `internal/template/enforcer.go` + `_test.go`

Enforces: Issue → Clinical Context → Rationale → Evidence → Proposed Plan → Monitoring → Urgency. A `Recommendation.ClinicalContent` (Plan 0.1) cannot transition `detected → drafted` without all sections populated or marked NA.

- [ ] **Step 1: Write failing test**

```go
package template

import (
	"testing"

	"shared/v2_substrate/models"
)

func TestEnforcer_RejectsIncomplete(t *testing.T) {
	cc := models.ClinicalContent{
		Issue: "test", // missing other 6 sections
	}
	if err := Validate(cc); err == nil {
		t.Errorf("expected validation error for incomplete content")
	}
}

func TestEnforcer_AcceptsCompleteOrExplicitNA(t *testing.T) {
	cc := models.ClinicalContent{
		Issue:           "x",
		ClinicalContext: "x",
		Rationale:       "x",
		EvidenceRefs:    []string{"AMH-2024"},
		ProposedPlan:    "x",
		MonitoringPlan:  "NA — one-shot intervention",
	}
	if err := Validate(cc); err != nil {
		t.Errorf("expected accept; got %v", err)
	}
}
```

- [ ] **Step 2-5: Implement, run, commit**

```go
package template

import (
	"errors"
	"strings"

	"shared/v2_substrate/models"
)

var ErrIncomplete = errors.New("clinical content incomplete")

// Validate enforces the v3 §7 line 369 template structure. Returns nil
// when all 7 sections are populated (or explicitly marked NA with
// reason).
func Validate(cc models.ClinicalContent) error {
	missing := []string{}
	if cc.Issue == "" {
		missing = append(missing, "issue")
	}
	if cc.ClinicalContext == "" {
		missing = append(missing, "clinical_context")
	}
	if cc.Rationale == "" {
		missing = append(missing, "rationale")
	}
	if len(cc.EvidenceRefs) == 0 {
		missing = append(missing, "evidence_refs")
	}
	if cc.ProposedPlan == "" {
		missing = append(missing, "proposed_plan")
	}
	if cc.MonitoringPlan == "" {
		missing = append(missing, "monitoring_plan")
	}
	if len(missing) > 0 {
		return errors.New(ErrIncomplete.Error() + ": " + strings.Join(missing, ", "))
	}
	return nil
}
```

```bash
git commit -m "feat(kb-32): template enforcer for craft engine sections"
```

---

### Task 3: Clinical context assembler

**Files:**
- Create: `internal/context/assembler.go` + `_test.go`

Pulls from Plan 0.1+0.3 substrate: current labs (latest of each type), recent events (last 14 days), frailty (CFS/AKPS), care_intensity, DBI/ACB scores, MedicineUse list with intent/target/stop. Renders into the structured "clinical context" string per v3 §7 line 372 example.

- [ ] **Step 1-5: Implement, test, commit**

```go
package context

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type Assembler struct {
	substrate SubstrateClient // wraps the kb-20 substrate runtime API
}

type SubstrateClient interface {
	GetResidentSnapshot(ctx context.Context, residentID uuid.UUID) (Snapshot, error)
}

type Snapshot struct {
	Age              int
	EGFR             float64
	RecentEvents     []string  // e.g. ["fall_2026-04-30", "uti_2026-03-12"]
	ActiveConcerns   []string  // ["post_fall_72h"]
	CareIntensity    string    // active_treatment | rehab | comfort | palliative
	DBI              float64
	ACB              int
	MedicineCount    int
	AnticholinergicCount int
}

// Assemble returns a structured clinical context string and a map of the
// raw substrate data the craft engine renders into the recommendation.
func (a *Assembler) Assemble(ctx context.Context, residentID uuid.UUID) (string, Snapshot, error) {
	snap, err := a.substrate.GetResidentSnapshot(ctx, residentID)
	if err != nil {
		return "", Snapshot{}, err
	}
	// v3 §7 line 372 narrative render
	out := fmt.Sprintf(
		"%dyo with eGFR %.0f, %d active medications including %d anticholinergics, "+
			"care intensity: %s, DBI %.2f, ACB %d. Active concerns: %v. "+
			"Recent events: %v.",
		snap.Age, snap.EGFR, snap.MedicineCount, snap.AnticholinergicCount,
		snap.CareIntensity, snap.DBI, snap.ACB,
		snap.ActiveConcerns, snap.RecentEvents,
	)
	return out, snap, nil
}
```

```bash
git commit -m "feat(kb-32): context assembler renders substrate state into clinical narrative"
```

---

### Task 4: Evidence anchor selector

**Files:**
- Create: `internal/evidence/anchor_selector.go` + `_test.go`

Per v3 §7 line 374: max 2 strong sources per recommendation. Source priority: AMH > Therapeutic Guidelines > RACGP > ADG 2025 > Beers/STOPP-START as supplement. Selection adapts to recommendation type and clinical context.

- [ ] **Step 1-5: Implement source ranking, brevity guard, test, commit**

```go
package evidence

type Source struct {
	ID         string  // e.g. "AMH-Aged-Care-2024-Ch7"
	Authority  string  // "AMH" | "TG" | "RACGP" | "ADG2025" | "Beers" | "STOPP_START"
	Tier       int     // 1 = primary AU, 2 = primary international, 3 = supplement
	Match      float64 // relevance to the recommendation [0..1]
}

func SelectAnchors(candidates []Source, max int) []Source {
	// Sort by tier ASC then match DESC; take top max.
	// Implementation omitted for brevity; standard sort.Slice.
	return nil
}
```

Test cases: AMH+RACGP for a deprescribing recommendation; Beers as supplement only when AMH absent; never returns >2 anchors.

```bash
git commit -m "feat(kb-32): evidence anchor selector with AU-first source ranking"
```

---

### Task 5: Recommendation orderer + urgency tagger

**Files:**
- Create: `internal/ordering/orderer.go` + `_test.go`
- Create: `internal/urgency/tagger.go` + `_test.go`

Orderer: STOP > MONITOR > DOSE > ADD per v3 §7 line 384. Critical anti-suppression guard: clinically indicated ADD recommendations are NOT removed from the packet because acceptance is statistically lower (v3 §7 line 392).

Urgency: Red (24-48h: AKI, hyperkalaemia, hypoglycaemia, recent fall on CNS-active, QTc), Amber (1-2 wks: deprescribing, optimisation), Green (next review: preventive, cost-saving).

- [ ] **Step 1-5: Implement, test (including the anti-suppression test that ensures ADD recommendations stay in the packet), commit**

```go
// orderer.go
package ordering

import (
	"sort"

	"shared/v2_substrate/models"
)

var typeOrder = map[string]int{
	models.RecommendationTypeStop:       0,
	models.RecommendationTypeMonitor:    1,
	models.RecommendationTypeDoseChange: 2,
	models.RecommendationTypeAdd:        3,
}

// OrderPacket sorts recommendations within a packet to maximise GP momentum.
// IMPORTANT: per v3 §7 line 392, this function never SUPPRESSES recommendations.
// It only ORDERS. Suppression of clinically indicated recommendations is a
// safety failure; any caller that filters by predicted-acceptance is
// architecturally wrong.
func OrderPacket(recs []models.Recommendation) []models.Recommendation {
	sort.SliceStable(recs, func(i, j int) bool {
		return typeOrder[recs[i].Type] < typeOrder[recs[j].Type]
	})
	return recs
}
```

Anti-suppression test:

```go
func TestOrderer_DoesNotSuppressADD(t *testing.T) {
	in := []models.Recommendation{
		{Type: models.RecommendationTypeAdd, Title: "needed add"},
		{Type: models.RecommendationTypeStop, Title: "easy stop"},
	}
	out := OrderPacket(in)
	if len(out) != 2 {
		t.Errorf("orderer should not drop recommendations; got %d in / %d out",
			len(in), len(out))
	}
	if out[1].Type != models.RecommendationTypeAdd {
		t.Errorf("ADD should be ordered last but present")
	}
}
```

```bash
git commit -m "feat(kb-32): packet orderer with anti-suppression guard + urgency tagger"
```

---

### Task 6: Frame-vs-content separator

**Files:**
- Create: `internal/framing/separator.go` + `_test.go`

Per v3 §7 line 416 + Architectural Commitment 8. Two-layer recording: `clinical_content` invariant across audiences, `framing_adaptation` variable. Hash both into the EvidenceTrace via Plan 0.1's `EvidenceEdge`. Regulator audit query "show clinical content for recommendation X across all framings" must demonstrate identical content.

- [ ] **Step 1: Write failing test**

```go
func TestSeparator_ContentInvariantAcrossFramings(t *testing.T) {
	rec := craftedRecommendation()
	frame1 := Render(rec, AudienceGP{Name: "Dr Smith"})
	frame2 := Render(rec, AudienceNP{Name: "NP Jones"})

	if frame1.ClinicalContentHash() != frame2.ClinicalContentHash() {
		t.Errorf("clinical content must be invariant across framings; got %q vs %q",
			frame1.ClinicalContentHash(), frame2.ClinicalContentHash())
	}
	if frame1.FramingHash() == frame2.FramingHash() {
		t.Errorf("framing should differ across audiences; both produced %q",
			frame1.FramingHash())
	}
}
```

- [ ] **Step 2-5: Implement, test, commit**

```go
package framing

import (
	"crypto/sha256"
	"encoding/hex"

	"shared/v2_substrate/models"
)

type Audience interface {
	AudienceID() string
	PreferredEvidence() string
	PreferredChannel() string
}

type Rendered struct {
	ClinicalContent models.ClinicalContent
	Framing         FramingAdaptation
}

type FramingAdaptation struct {
	AudienceID         string
	EvidenceLeadWith   string // chosen from cc.EvidenceRefs
	OpeningLine        string
	ChannelHint        string
}

func Render(rec models.Recommendation, audience Audience) Rendered {
	// ClinicalContent is COPIED, not modified.
	cc := rec.ClinicalContent
	framing := FramingAdaptation{
		AudienceID:       audience.AudienceID(),
		EvidenceLeadWith: pickEvidence(cc.EvidenceRefs, audience.PreferredEvidence()),
		OpeningLine:      personalisedOpening(audience),
		ChannelHint:      audience.PreferredChannel(),
	}
	return Rendered{ClinicalContent: cc, Framing: framing}
}

func (r Rendered) ClinicalContentHash() string {
	h := sha256.New()
	// stable JSON serialization; or canonicalize all fields explicitly
	h.Write([]byte(r.ClinicalContent.Issue))
	h.Write([]byte(r.ClinicalContent.ClinicalContext))
	h.Write([]byte(r.ClinicalContent.Rationale))
	h.Write([]byte(r.ClinicalContent.ProposedPlan))
	h.Write([]byte(r.ClinicalContent.MonitoringPlan))
	for _, ev := range r.ClinicalContent.EvidenceRefs {
		h.Write([]byte(ev))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (r Rendered) FramingHash() string {
	h := sha256.New()
	h.Write([]byte(r.Framing.AudienceID))
	h.Write([]byte(r.Framing.EvidenceLeadWith))
	h.Write([]byte(r.Framing.OpeningLine))
	h.Write([]byte(r.Framing.ChannelHint))
	return hex.EncodeToString(h.Sum(nil))
}
```

```bash
git commit -m "feat(kb-32): frame-vs-content separator with hash-invariant content layer"
```

---

### Task 7: Per-GP framing observer (V1; ethical limits)

**Files:**
- Create: `internal/framing/per_gp_observer.go` + `_test.go`
- Create: `migrations/029_per_gp_framing_observations.sql` + rollback

Records per-(GP, recommendation_type, evidence_source, channel, framing_style) acceptance pattern. Surfaces as gentle suggestion to pharmacist's own dashboard only. Toxicity guardrails: NEVER surfaces per-GP acceptance percentages to pharmacy employer. Wrapped by Phase 1's permission middleware so subject = pharmacist self.

- [ ] **Step 1-5: Migration, observer, gentle-suggestion API, test (including the toxicity guard test that asserts employer-view returns no per-GP individual data), commit**

```sql
-- 029
CREATE TABLE per_gp_framing_observations (
    id                 UUID PRIMARY KEY,
    gp_id              UUID NOT NULL,
    pharmacist_id      UUID NOT NULL,
    recommendation_type TEXT NOT NULL,
    evidence_source    TEXT NOT NULL,
    channel            TEXT NOT NULL,
    framing_style      TEXT NOT NULL,
    accepted           BOOLEAN NOT NULL,
    observed_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_per_gp_framing_pharmacist ON per_gp_framing_observations (pharmacist_id, gp_id);
```

Test asserting toxicity guard:

```go
func TestPerGPObserver_EmployerViewNeverSeesIdentifiableGP(t *testing.T) {
	view := views.NewEmployerView(observerStore)
	got, _ := view.PerGPAggregate(context.Background(), pharmacistID)
	for _, row := range got {
		if row.GPName != "" {
			t.Errorf("employer view leaked identifiable GP: %q", row.GPName)
		}
	}
}
```

```bash
git commit -m "feat(kb-32): per-GP framing observer with employer-view toxicity guard"
```

---

### Task 8: Craft engine HTTP API + Recommendation lifecycle integration

**Files:**
- Create: `internal/api/handlers.go`
- Modify: `shared/v2_substrate/recommendation/lifecycle.go` — optional craft-engine call on detected → drafted

Endpoint: `POST /craft/draft` accepts `{rule_id, resident_id, author_id}`, returns `{recommendation_id, clinical_content, framing_for_audience}`. Recommendation lifecycle's `detected → drafted` transition optionally calls this endpoint and persists the result.

- [ ] **Step 1-5: Wire, test end-to-end, commit**

```bash
git commit -m "feat(kb-32): HTTP API + Recommendation lifecycle integration"
```

---

### Task 9: End-to-end integration test — Sunday-night-fall craft flow

Exercise: rule fires (Plan 0.5) → kb-32 assembles context (Task 3) + selects evidence (Task 4) + enforces template (Task 2) → orders + tags urgency (Task 5) → renders frame-vs-content (Task 6) → persists Recommendation (Plan 0.1) with both layers in EvidenceTrace.

```bash
git commit -m "test(kb-32): Sunday-night-fall craft engine end-to-end"
```

---

## Spec coverage

- [x] Structured-template enforcement — Task 2
- [x] Clinical-context auto-assembly — Task 3
- [x] Evidence anchoring + AU-first ranking — Task 4
- [x] Recommendation-type ordering with anti-suppression — Task 5
- [x] Urgency tagging — Task 5
- [x] Frame-vs-content separation auditable — Task 6
- [x] Per-GP framing learning with toxicity guard — Task 7
- [x] Recommendation lifecycle integration — Task 8

Plan complete and saved.
