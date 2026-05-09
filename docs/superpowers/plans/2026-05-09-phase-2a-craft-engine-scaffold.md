# Phase 2a — Recommendation Craft Engine Scaffold Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Land the Recommendation Craft Engine scaffold per v3 Architectural Commitment 7 (v3 §3 line 204) and the rendering pipeline (Stages 1–6) from *Recommendation Craft Implementation Guidelines v1.0*. Build the new service `kb-32-recommendation-craft` as a standalone Go service that takes a rule-fire signal, assembles clinical context, builds a reasoning chain, runs an appropriateness check, generates the recommendation packet, applies frame-vs-content separation, and renders the four-layer brevity-budgeted output. Phase 2b adds the clinical-safety + audit-moat extensions.

**Phase 2a / 2b split:**
This plan delivers the rendering scaffold + 6 of 7 pipeline stages: scaffold, lifecycle-state additions, template enforcer, context assembler, reasoning chain builder, recommendation generator, evidence selector, orderer + urgency tagger, appropriateness checker, frame-vs-content separator, per-GP framing observer, brevity formatter, HTTP API + lifecycle integration + E2E test. P0 blockers from the 2026-05-09 gap analysis are fixed inline:
- **P0-1: Migration 029 → 040 renumber** (029 already taken by Phase 1a's `data_aggregation_consents`)
- **P0-2: `replace github.com/cardiofit/shared` directive** in Task 1 go.mod scaffold

Phase 2b (`2026-05-09-phase-2b-clinical-safety-and-audit-moat.md`) lands the four guideline-distinctive items: override-reason taxonomy, citation versioning, negative-evidence patterns, restraint signals.

**Architecture:** New service `kb-32-recommendation-craft/` (Go, Gin, Postgres, Redis). Consumes Plan 0.1's Recommendation entity at the `detected → drafted` transition. The transition gate is now appropriateness-gated (Stage 4). Pipeline:

1. Stage 1 (Task 4): Context assembler — substrate state for resident
2. Stage 2 (Task 5): Reasoning chain builder — calls kb-cql-runtime, gets applicable rules
3. Stage 3 (Task 6): Recommendation generator — applies template, produces draft packet
4. Stage 4 (Task 9): Appropriateness checker — five-dimension rubric; gates the lifecycle transition
5. Stage 5 (Task 10): Framing adapter — frame-vs-content separation
6. Stage 6 (Task 12): Brevity formatter — four-layer rendering with word-budget enforcement

**Tech Stack:** Go, Gin, Postgres, Redis. Depends on Plans 0.1–0.5 and Phase 1a (permission middleware).

---

## File Structure

**New service files:**
- `kb-32-recommendation-craft/cmd/server/main.go`
- `kb-32-recommendation-craft/config/config.go`
- `kb-32-recommendation-craft/Dockerfile`
- `kb-32-recommendation-craft/go.mod` (with `replace github.com/cardiofit/shared => ../shared`)
- `kb-32-recommendation-craft/internal/template/enforcer.go` + `_test.go`
- `kb-32-recommendation-craft/internal/context/assembler.go` + `_test.go`
- `kb-32-recommendation-craft/internal/reasoning/chain_builder.go` + `_test.go`
- `kb-32-recommendation-craft/internal/reasoning/hapi_client.go` + `_test.go`
- `kb-32-recommendation-craft/internal/generator/recommendation.go` + `_test.go`
- `kb-32-recommendation-craft/internal/evidence/anchor_selector.go` + `_test.go`
- `kb-32-recommendation-craft/internal/ordering/orderer.go` + `_test.go`
- `kb-32-recommendation-craft/internal/urgency/tagger.go` + `_test.go`
- `kb-32-recommendation-craft/internal/appropriateness/checker.go` + `_test.go`
- `kb-32-recommendation-craft/internal/framing/separator.go` + `_test.go`
- `kb-32-recommendation-craft/internal/framing/per_gp_observer.go` + `_test.go`
- `kb-32-recommendation-craft/internal/formatter/formatter.go` + `_test.go`
- `kb-32-recommendation-craft/internal/formatter/layers.go`
- `kb-32-recommendation-craft/internal/lifecycle/transitions.go` + `_test.go`
- `kb-32-recommendation-craft/internal/api/handlers.go` + `_test.go`
- `kb-32-recommendation-craft/tests/integration/sunday_night_fall_test.go`

**New migration:**
- `migrations/040_per_gp_framing_observations.sql` + rollback (NOT 029 — that's Phase 1a's)
- `migrations/041_prescriber_framing_optout.sql` + rollback

**Modified files:**
- `shared/v2_substrate/recommendation/lifecycle.go` — wire kb-32 craft call into `detected → drafted` with appropriateness gate
- `shared/v2_substrate/models/enums.go` — add `RecommendationStateRejected`, `RecommendationStateWithdrawn`, `RecommendationStateSuperseded`
- `shared/v2_substrate/models/recommendation.go` — extend `validTransitions` map for new states

---

### Task 1: Service scaffold + go.mod with replace directive

**Files:**
- Create: `kb-32-recommendation-craft/cmd/server/main.go`, `config/config.go`, `Dockerfile`, `go.mod`

Mirror kb-30 layout. The P0 fix is the `replace` directive — without it, `import "github.com/cardiofit/shared/..."` fails at build.

- [ ] **Step 1: Create directory structure**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
mkdir -p kb-32-recommendation-craft/{cmd/server,config,internal/{template,context,reasoning,generator,evidence,ordering,urgency,appropriateness,framing,formatter,lifecycle,api},tests/integration}
cd kb-32-recommendation-craft
```

- [ ] **Step 2: go.mod with replace directive**

```
module github.com/cardiofit/kb32

go 1.22

require (
    github.com/cardiofit/shared v0.0.0
    github.com/gin-gonic/gin v1.9.1
    github.com/google/uuid v1.5.0
    github.com/lib/pq v1.10.9
)

replace github.com/cardiofit/shared => ../shared
```

- [ ] **Step 3: main.go skeleton**

```go
package main

import (
    "database/sql"
    "log"
    "net/http"
    "os"

    "github.com/gin-gonic/gin"
    _ "github.com/lib/pq"
)

const Version = "0.1.0-phase-2a"

func main() {
    dsn := os.Getenv("VAIDSHALA_DSN")
    if dsn == "" {
        log.Fatal("VAIDSHALA_DSN is required")
    }
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        log.Fatalf("db open: %v", err)
    }
    defer db.Close()

    r := gin.New()
    r.Use(gin.Recovery())
    r.GET("/healthz", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"status": "ok", "version": Version})
    })

    log.Printf("kb-32 %s listening on :%s", Version, getenv("PORT", "8150"))
    if err := r.Run(":" + getenv("PORT", "8150")); err != nil {
        log.Fatal(err)
    }
}

func getenv(k, def string) string {
    if v := os.Getenv(k); v != "" {
        return v
    }
    return def
}
```

- [ ] **Step 4: Smoke build**

```bash
go mod tidy
go build ./cmd/server
```

Must pass.

- [ ] **Step 5: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-32-recommendation-craft/
git commit -m "feat(kb-32): scaffold recommendation-craft service with replace directive"
```

---

### Task 2: Lifecycle states fix (rejected/withdrawn/superseded)

**Files:**
- Modify: `shared/v2_substrate/models/enums.go`
- Modify: `shared/v2_substrate/models/recommendation.go`

Per Guidelines Part 3, the state machine has 13 states. Current code has 10. Add the 3 missing.

- [ ] **Step 1: Failing test in `recommendation_test.go`**

```go
func TestRecommendation_RejectedWithdrawnSupersededAreValid(t *testing.T) {
    for _, s := range []string{"rejected", "withdrawn", "superseded"} {
        if !IsValidRecommendationState(s) {
            t.Errorf("%q should be a valid state", s)
        }
    }
}

func TestRecommendation_RejectedFromDecided(t *testing.T) {
    if !validTransitions["decided"]["rejected"] {
        t.Errorf("decided → rejected should be allowed")
    }
}
```

- [ ] **Step 2: Add constants**

```go
// In enums.go
const (
    RecommendationStateRejected   = "rejected"
    RecommendationStateWithdrawn  = "withdrawn"
    RecommendationStateSuperseded = "superseded"
)
```

Update `IsValidRecommendationState` to accept these.

- [ ] **Step 3: Extend `validTransitions`**

Add: `decided → rejected`, `drafted → withdrawn`, `implemented → superseded`, `monitoring-active → superseded`.

- [ ] **Step 4: Verify**

```bash
cd backend/shared-infrastructure/knowledge-base-services/shared
go test -race ./v2_substrate/models/... -v
```

- [ ] **Step 5: Commit**

```bash
git commit -m "feat(substrate): add rejected/withdrawn/superseded recommendation states (Guidelines Part 3)"
```

---

### Task 3: Template enforcer

**Files:**
- Create: `internal/template/enforcer.go` + `_test.go`

Validates the 7-section v3 §7 template completeness: Issue → Clinical Context → Rationale → Evidence → Proposed Plan → Monitoring → Urgency. Reject partial; allow explicit `NA`.

- [ ] **Step 1-3: Test + implement**

```go
package template

import (
    "errors"
    "strings"
)

// Section names per v3 §7 line 369.
const (
    SectionIssue           = "issue"
    SectionClinicalContext = "clinical_context"
    SectionRationale       = "rationale"
    SectionEvidence        = "evidence"
    SectionProposedPlan    = "proposed_plan"
    SectionMonitoring      = "monitoring"
    SectionUrgency         = "urgency"
)

var requiredSections = []string{
    SectionIssue, SectionClinicalContext, SectionRationale,
    SectionEvidence, SectionProposedPlan, SectionMonitoring, SectionUrgency,
}

// NA is the explicit not-applicable marker. Empty strings are rejected;
// only "NA" satisfies completeness for a section that legitimately has no content.
const NA = "NA"

var (
    ErrMissingSection = errors.New("template: required section missing or empty")
    ErrInvalidSection = errors.New("template: unknown section name")
)

// Enforce validates that every required section has non-empty content (or "NA").
func Enforce(packet map[string]string) error {
    for _, s := range requiredSections {
        v, ok := packet[s]
        if !ok || strings.TrimSpace(v) == "" {
            return ErrMissingSection
        }
    }
    return nil
}
```

Tests: missing field, "NA" passes, all 7 present passes, completely empty fails.

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(kb-32): template enforcer for 7-section v3 §7 packet"
```

---

### Task 4: Context assembler (Stage 1)

**Files:**
- Create: `internal/context/assembler.go` + `_test.go`

Pulls resident state from substrate (Plan 0.1 + 0.2 + 0.3 baselines) into a `ClinicalSnapshot`.

- [ ] **Step 1-3: Test + implement**

```go
package context

import (
    "context"
    "time"

    "github.com/google/uuid"
)

// ClinicalSnapshot is the substrate state assembled for a single resident
// at recommendation craft time. Per Guidelines §4 Stage 1.
type ClinicalSnapshot struct {
    ResidentID         uuid.UUID
    EGFR               float64   // ml/min/1.73m²
    DBI                float64   // Drug Burden Index
    ACB                int       // Anticholinergic Cognitive Burden
    CFS                int       // Clinical Frailty Scale 1-9
    CareIntensity      string    // "active" | "comfort" | "palliative" | "end_of_life"
    RecentFall72h      bool
    RecentAdmission72h bool
    AssessedAt         time.Time
}

type SubstrateClient interface {
    SnapshotFor(ctx context.Context, residentID uuid.UUID) (ClinicalSnapshot, error)
}

type Assembler struct{ src SubstrateClient }

func NewAssembler(src SubstrateClient) *Assembler { return &Assembler{src: src} }

func (a *Assembler) Assemble(ctx context.Context, residentID uuid.UUID) (ClinicalSnapshot, error) {
    if err := ctx.Err(); err != nil {
        return ClinicalSnapshot{}, err
    }
    return a.src.SnapshotFor(ctx, residentID)
}
```

Tests: happy path, source error propagates, ctx cancellation.

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(kb-32): context assembler pulls substrate state into ClinicalSnapshot"
```

---

### Task 5: Reasoning chain builder (Stage 2) + HAPI client

**Files:**
- Create: `internal/reasoning/chain_builder.go` + `_test.go`
- Create: `internal/reasoning/hapi_client.go` + `_test.go`

NEW per gap analysis. Calls kb-cql-runtime's `$evaluate-rule`, ingests `applicable_rules`. Includes a stub fallback for the Phase 0.5 placeholder shape.

- [ ] **Step 1-3: Test + implement**

```go
// hapi_client.go
package reasoning

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "net/http"

    "github.com/google/uuid"
)

type HAPIClient struct {
    baseURL string
    http    *http.Client
}

func NewHAPIClient(baseURL string) *HAPIClient {
    return &HAPIClient{baseURL: baseURL, http: &http.Client{}}
}

type EvaluateRuleResult struct {
    RuleID       string         `json:"ruleId"`
    LibraryFound bool           `json:"libraryFound"`
    Status       string         `json:"status"`
    Triggered    bool           `json:"triggered,omitempty"`
    Type         string         `json:"type,omitempty"`
    Urgency      string         `json:"urgency,omitempty"`
}

var ErrCQLPlaceholderResponse = errors.New("reasoning: kb-cql-runtime returned engine-pending placeholder")

func (c *HAPIClient) EvaluateRule(ctx context.Context, ruleID string, residentID uuid.UUID) (*EvaluateRuleResult, error) {
    url := fmt.Sprintf("%s/Library/%s/$evaluate-rule?residentId=%s", c.baseURL, ruleID, residentID)
    req, _ := http.NewRequestWithContext(ctx, "POST", url, nil)
    resp, err := c.http.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    if resp.StatusCode != 200 {
        body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
        return nil, fmt.Errorf("hapi: status %d: %s", resp.StatusCode, body)
    }
    var out EvaluateRuleResult
    if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
        return nil, err
    }
    if out.Status == "library_found_engine_pending" {
        return &out, ErrCQLPlaceholderResponse
    }
    return &out, nil
}
```

```go
// chain_builder.go
package reasoning

import (
    "context"
    "errors"

    "github.com/google/uuid"
)

// ApplicableRule is one rule that fired against the resident's substrate state.
type ApplicableRule struct {
    RuleID  string
    Type    string  // "stop" | "monitor" | "dose_change" | "add"
    Urgency string  // "red" | "amber" | "green"
}

// ReasoningSource abstracts the rule-evaluation layer for testability.
type ReasoningSource interface {
    EvaluateRule(ctx context.Context, ruleID string, residentID uuid.UUID) (*EvaluateRuleResult, error)
}

type ChainBuilder struct{ src ReasoningSource }

func NewChainBuilder(s ReasoningSource) *ChainBuilder { return &ChainBuilder{src: s} }

// Build evaluates a list of candidate rules and returns those that triggered.
// CQL placeholder responses are silently skipped (Phase 0.5 deferred state) — log
// a warning in production but do not fail the chain.
func (b *ChainBuilder) Build(ctx context.Context, residentID uuid.UUID, candidates []string) ([]ApplicableRule, error) {
    out := []ApplicableRule{}
    for _, ruleID := range candidates {
        res, err := b.src.EvaluateRule(ctx, ruleID, residentID)
        if errors.Is(err, ErrCQLPlaceholderResponse) {
            continue // engine pending; treat as non-firing
        }
        if err != nil {
            return nil, err
        }
        if res.Triggered {
            out = append(out, ApplicableRule{RuleID: ruleID, Type: res.Type, Urgency: res.Urgency})
        }
    }
    return out, nil
}
```

Tests: happy path with one rule fires, placeholder shape skipped, source error propagates, mixed candidate list.

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(kb-32): reasoning chain builder + HAPI client with placeholder fallback"
```

---

### Task 6: Recommendation generator (Stage 3)

**Files:**
- Create: `internal/generator/recommendation.go` + `_test.go`

Takes context + applicable_rules, produces a packet matching Plan 0.1's struct shape. Uses the template enforcer (Task 3).

- [ ] **Step 1-3: Test + implement**

```go
package generator

import (
    "errors"
    "fmt"

    "github.com/google/uuid"

    cardcontext "github.com/cardiofit/kb32/internal/context"
    "github.com/cardiofit/kb32/internal/reasoning"
    "github.com/cardiofit/kb32/internal/template"
)

// Packet is the rendered recommendation in template-enforced form.
type Packet struct {
    RecommendationID uuid.UUID
    AuthorID         uuid.UUID
    Type             string  // STOP / MONITOR / DOSE_CHANGE / ADD
    Sections         map[string]string  // 7 sections per template.Enforce
    AppliedRule      string
    SnapshotRef      uuid.UUID
}

var ErrNoApplicableRules = errors.New("generator: no applicable rules to render")

// Generate produces a Packet from the snapshot + applicable rules, then validates
// it against the template. The first applicable rule wins (orderer reorders later).
func Generate(snap cardcontext.ClinicalSnapshot, rules []reasoning.ApplicableRule, authorID uuid.UUID) (*Packet, error) {
    if len(rules) == 0 {
        return nil, ErrNoApplicableRules
    }
    r := rules[0]
    p := &Packet{
        RecommendationID: uuid.New(),
        AuthorID:         authorID,
        Type:             r.Type,
        AppliedRule:      r.RuleID,
        SnapshotRef:      snap.ResidentID,
        Sections: map[string]string{
            template.SectionIssue:           fmt.Sprintf("Rule %s fired (%s)", r.RuleID, r.Type),
            template.SectionClinicalContext: fmt.Sprintf("eGFR=%.1f, DBI=%.2f, CFS=%d, CareIntensity=%s",
                                                snap.EGFR, snap.DBI, snap.CFS, snap.CareIntensity),
            template.SectionRationale:       template.NA,  // filled by Stage 5 framing adapter
            template.SectionEvidence:        template.NA,  // filled by Task 7 evidence selector
            template.SectionProposedPlan:    template.NA,  // filled by Stage 5
            template.SectionMonitoring:      template.NA,  // filled by Stage 5
            template.SectionUrgency:         r.Urgency,
        },
    }
    if err := template.Enforce(p.Sections); err != nil {
        return nil, err
    }
    return p, nil
}
```

Tests: happy path, empty rules → ErrNoApplicableRules, template violation rejected.

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(kb-32): recommendation generator producing template-enforced packets"
```

---

### Task 7: Evidence anchor selector

**Files:**
- Create: `internal/evidence/anchor_selector.go` + `_test.go`

AU-first ranking: Australian Deprescribing Guideline 2025 first, then international (Beers, STOPP/START). Max 2 anchors per recommendation.

- [ ] **Step 1-3: Test + implement**

```go
package evidence

import "sort"

type Anchor struct {
    SourceID    string
    Title       string
    Jurisdiction string  // "AU" | "US" | "EU" | "INTL"
    Rank        int      // lower = stronger
}

const MaxAnchorsPerRec = 2

// Select returns up to MaxAnchorsPerRec anchors for the given rule, preferring
// Australian sources first, then international. Stable sort ensures deterministic
// tie-break by SourceID.
func Select(candidates []Anchor) []Anchor {
    sort.SliceStable(candidates, func(i, j int) bool {
        ai, aj := candidates[i], candidates[j]
        if (ai.Jurisdiction == "AU") != (aj.Jurisdiction == "AU") {
            return ai.Jurisdiction == "AU"
        }
        if ai.Rank != aj.Rank {
            return ai.Rank < aj.Rank
        }
        return ai.SourceID < aj.SourceID
    })
    if len(candidates) > MaxAnchorsPerRec {
        candidates = candidates[:MaxAnchorsPerRec]
    }
    return candidates
}
```

Tests: AU-first when mixed jurisdictions, max-2-cap, tie-break by SourceID, empty input returns empty.

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(kb-32): evidence anchor selector with AU-first ranking"
```

---

### Task 8: Recommendation orderer + urgency tagger

**Files:**
- Create: `internal/ordering/orderer.go` + `_test.go`
- Create: `internal/urgency/tagger.go` + `_test.go`

STOP > MONITOR > DOSE > ADD ordering; orderer-cannot-suppress assertion (only reorders, never drops). Red/Amber/Green urgency from substrate state.

- [ ] **Step 1-3: Test + implement**

```go
// orderer.go
package ordering

import (
    "sort"

    "github.com/cardiofit/kb32/internal/generator"
)

var typeRank = map[string]int{
    "STOP": 0, "MONITOR": 1, "DOSE_CHANGE": 2, "ADD": 3,
}

// Order applies the canonical type ordering. Anti-suppression invariant:
// len(out) == len(in). The orderer cannot drop a recommendation, only reorder.
func Order(in []*generator.Packet) []*generator.Packet {
    out := make([]*generator.Packet, len(in))
    copy(out, in)
    sort.SliceStable(out, func(i, j int) bool {
        return typeRank[out[i].Type] < typeRank[out[j].Type]
    })
    return out
}
```

```go
// tagger.go
package urgency

import "github.com/cardiofit/kb32/internal/context"

// Tag returns Red/Amber/Green urgency from substrate signals. Per Guidelines §5.
func Tag(snap context.ClinicalSnapshot) string {
    if snap.RecentFall72h || snap.RecentAdmission72h {
        return "red"
    }
    if snap.ACB >= 3 || snap.DBI >= 1.0 {
        return "amber"
    }
    if snap.CareIntensity == "end_of_life" || snap.CareIntensity == "palliative" {
        return "amber"
    }
    return "green"
}
```

Tests: ordering happy path, anti-suppression assertion (input length == output length), each urgency tier path.

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(kb-32): orderer (anti-suppression) + urgency tagger (Red/Amber/Green)"
```

---

### Task 9: Appropriateness checker (Stage 4)

**Files:**
- Create: `internal/appropriateness/checker.go` + `_test.go`

NEW per gap analysis. Five-dimension rubric per Guidelines Part 9. **Hold threshold: any dimension ≤ 2 → recommendation HELD in `detected` state, does NOT advance to `drafted`.** This is the clinical-safety gate.

- [ ] **Step 1-3: Test + implement**

```go
package appropriateness

import "errors"

// Assessment scores each of 5 dimensions on a 1-5 scale.
type Assessment struct {
    ClinicalWarrant       int  // is intervention warranted?
    EvidenceSolidity      int  // strength of supporting evidence
    AlternativesConsidered int  // were alternatives weighed?
    RestraintConsidered   int  // was non-action evaluated?
    GoalsOfCareAlignment  int  // alignment with documented care intensity
}

const HoldThreshold = 2

var ErrAppropriatenessHold = errors.New("appropriateness: dimension below hold threshold; recommendation held in detected")

// Check returns nil if all 5 dimensions are > HoldThreshold; ErrAppropriatenessHold otherwise.
// Caller MUST honor the hold by leaving the recommendation in `detected` state.
func Check(a Assessment) error {
    for _, score := range []int{
        a.ClinicalWarrant, a.EvidenceSolidity, a.AlternativesConsidered,
        a.RestraintConsidered, a.GoalsOfCareAlignment,
    } {
        if score <= HoldThreshold {
            return ErrAppropriatenessHold
        }
    }
    return nil
}

// PassesGate is a sugar method for the lifecycle transition.
func (a Assessment) PassesGate() bool { return Check(a) == nil }
```

Tests: each dimension hold path (5 separate tests), all-5-pass case, boundary (score == 2 holds; score == 3 passes).

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(kb-32): appropriateness checker — 5-dimension rubric, hold threshold ≤ 2"
```

---

### Task 10: Frame-vs-content separator (Stage 5)

**Files:**
- Create: `internal/framing/separator.go` + `_test.go`

Persist `clinical_content` (invariant) separately from `framing_adaptation` (variable per audience). Compute `content_hash` invariance.

- [ ] **Step 1-3: Test + implement**

```go
package framing

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "sort"
)

// ClinicalContent is the audience-invariant payload. Same content_hash regardless
// of how the same recommendation is later re-framed for a different audience.
type ClinicalContent struct {
    RuleID         string
    Type           string
    EvidenceAnchors []string
    Urgency        string
}

// FramingAdaptation is the audience-variable layer. Multiple framings can attach
// to the same ClinicalContent without altering its hash.
type FramingAdaptation struct {
    Audience    string  // "gp" | "pharmacist" | "rach_staff" | "regulator"
    OpeningLine string
    ClosingCall string
}

// ContentHash deterministically hashes ClinicalContent. Sorts EvidenceAnchors
// alphabetically before hashing so anchor-list ordering does not affect the hash.
func ContentHash(c ClinicalContent) string {
    sort.Strings(c.EvidenceAnchors)
    b, _ := json.Marshal(c)
    sum := sha256.Sum256(b)
    return hex.EncodeToString(sum[:])
}
```

Tests: same content → same hash; different audiences with same content → same hash; different content → different hash; anchor reordering doesn't affect hash.

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(kb-32): frame-vs-content separator with audience-invariant content_hash"
```

---

### Task 11: Per-GP framing observer + Migration 040

**Files:**
- Create: `internal/framing/per_gp_observer.go` + `_test.go`
- Create: `migrations/040_per_gp_framing_observations.sql` + rollback
- Create: `migrations/041_prescriber_framing_optout.sql` + rollback

**Toxicity guard fixes from gap analysis:**
- 30-observation floor before learner returns non-default framing
- Aggregate-only across pharmacists (NO per-pharmacist attribution in schema)
- Prescriber opt-out via separate `prescriber_framing_optout` table

- [ ] **Step 1-3: Test + implement**

```go
package framing

import (
    "context"
    "errors"

    "github.com/google/uuid"
)

const MinObservationsThreshold = 30

var ErrFramingOptedOut = errors.New("per-gp-observer: prescriber has opted out of framing learning")

type FramingPattern struct {
    GPID             uuid.UUID
    BestFramingTone  string  // "concise" | "detailed" | "collaborative"
    ObservationCount int
}

type ObservationSource interface {
    PatternFor(ctx context.Context, gpID uuid.UUID) (FramingPattern, error)
    HasOptedOut(ctx context.Context, gpID uuid.UUID) (bool, error)
}

type PerGPObserver struct{ src ObservationSource }

func NewPerGPObserver(src ObservationSource) *PerGPObserver { return &PerGPObserver{src: src} }

// Suggest returns the learned framing tone for this GP, or "default" if:
//   - the GP has opted out
//   - observation count is below MinObservationsThreshold
func (o *PerGPObserver) Suggest(ctx context.Context, gpID uuid.UUID) (string, error) {
    optedOut, err := o.src.HasOptedOut(ctx, gpID)
    if err != nil {
        return "", err
    }
    if optedOut {
        return "default", nil
    }
    pat, err := o.src.PatternFor(ctx, gpID)
    if err != nil {
        return "", err
    }
    if pat.ObservationCount < MinObservationsThreshold {
        return "default", nil
    }
    return pat.BestFramingTone, nil
}
```

- [ ] **Step 4: Migration 040**

```sql
-- 040_per_gp_framing_observations.sql
-- Phase 2a Task 11. Per-GP framing patterns aggregated ACROSS all pharmacists.
-- NO pharmacist_id column — aggregation-only by architectural prohibition
-- (Recommendation Craft Guidelines §8 toxicity guard rule 2).
BEGIN;
CREATE TABLE per_gp_framing_observations (
    id                 UUID PRIMARY KEY,
    gp_id              UUID NOT NULL,
    framing_tone       TEXT NOT NULL CHECK (framing_tone IN ('concise','detailed','collaborative')),
    decision_outcome   TEXT NOT NULL CHECK (decision_outcome IN ('accepted','declined','deferred')),
    observed_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_pgfo_gp_recent ON per_gp_framing_observations (gp_id, observed_at DESC);
COMMIT;
```

- [ ] **Step 5: Migration 041 (opt-out)**

```sql
-- 041_prescriber_framing_optout.sql
BEGIN;
CREATE TABLE prescriber_framing_optout (
    gp_id      UUID PRIMARY KEY,
    opted_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reason     TEXT
);
COMMIT;
```

Tests: under-30-obs returns "default"; opted-out returns "default"; ≥ 30-obs and not-opted-out returns learned tone.

- [ ] **Step 6: Commit**

```bash
git commit -m "feat(kb-32): per-GP framing observer + migrations 040/041 (aggregate-only, opt-out, 30-obs floor)"
```

---

### Task 12: Brevity formatter (Stage 6) — four-layer rendering

**Files:**
- Create: `internal/formatter/formatter.go` + `_test.go`
- Create: `internal/formatter/layers.go`

NEW per gap analysis. Layer 1 ≤ 25 words (Signal), Layer 2 ≤ 100 words (Reasoning), Layer 3 structured list (Provenance), Layer 4 unbounded (Deep Audit). Word-budget enforcement at function level.

- [ ] **Step 1-3: Test + implement**

```go
// layers.go
package formatter

const (
    Layer1MaxWords = 25
    Layer2MaxWords = 100
)

// LayerOutput holds the four-layer rendering. Each layer is a string;
// Layer 3 is a slice of provenance entries.
type LayerOutput struct {
    L1Signal    string
    L2Reasoning string
    L3Provenance []string
    L4DeepAudit string
}
```

```go
// formatter.go
package formatter

import (
    "errors"
    "strings"
)

var (
    ErrLayer1OverBudget = errors.New("formatter: Layer 1 Signal exceeds 25 words")
    ErrLayer2OverBudget = errors.New("formatter: Layer 2 Reasoning exceeds 100 words")
)

func wordCount(s string) int {
    return len(strings.Fields(s))
}

// Validate enforces word budgets. Returns the appropriate error if any layer
// exceeds. Layer 3 and 4 have no word cap.
func Validate(out LayerOutput) error {
    if wordCount(out.L1Signal) > Layer1MaxWords {
        return ErrLayer1OverBudget
    }
    if wordCount(out.L2Reasoning) > Layer2MaxWords {
        return ErrLayer2OverBudget
    }
    return nil
}
```

Tests: Layer 1 over budget rejected; Layer 2 over budget rejected; both within budget passes; Layers 3 & 4 unbounded confirmed.

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(kb-32): brevity formatter — four-layer rendering with word-budget enforcement"
```

---

### Task 13: HTTP API + Recommendation lifecycle integration + E2E test

**Files:**
- Create: `internal/api/handlers.go` + `_test.go`
- Create: `internal/lifecycle/transitions.go` + `_test.go`
- Create: `tests/integration/sunday_night_fall_test.go`
- Modify: `cmd/server/main.go` to mount handlers
- Modify: `shared/v2_substrate/recommendation/lifecycle.go` to call kb-32 craft on `detected → drafted`

The lifecycle transition gate calls `appropriateness.Check`. Recommendations with score ≤ 2 on any dimension stay in `detected`; others advance to `drafted`.

- [ ] **Step 1-3: Implement handler, lifecycle integration, E2E test**

```go
// handlers.go
package api

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

type DraftRequest struct {
    RuleID     string    `json:"rule_id"`
    ResidentID uuid.UUID `json:"resident_id"`
    AuthorID   uuid.UUID `json:"author_id"`
}

type DraftResponse struct {
    RecommendationID uuid.UUID         `json:"recommendation_id"`
    State            string            `json:"state"`  // "drafted" or "detected" (held)
    ContentHash      string            `json:"content_hash"`
    HoldReason       string            `json:"hold_reason,omitempty"`
}

// HandleDraft is mounted as POST /craft/draft, behind permissions middleware (PDP class).
// The full pipeline (Tasks 4-12) is invoked. Returns either a drafted recommendation
// or a held one with the appropriateness reason.
func HandleDraft(c *gin.Context) {
    var req DraftRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
        return
    }
    // Pipeline orchestration (Tasks 4 → 5 → 6 → 7 → 8 → 9 → 10 → 12) lives in
    // an internal Pipeline struct constructed by main.go; the handler is thin.
    c.JSON(http.StatusNotImplemented, gin.H{"todo": "pipeline orchestration"})
}
```

```go
// lifecycle/transitions.go
package lifecycle

import (
    "errors"

    "github.com/cardiofit/kb32/internal/appropriateness"
)

var ErrTransitionHeld = errors.New("lifecycle: appropriateness hold; remains in detected")

// AdvanceDetectedToDrafted is the kb-32 gate function called from
// shared/v2_substrate/recommendation/lifecycle.go on the detected→drafted transition.
// Returns ErrTransitionHeld if appropriateness check fails.
func AdvanceDetectedToDrafted(a appropriateness.Assessment) error {
    if !a.PassesGate() {
        return ErrTransitionHeld
    }
    return nil
}
```

E2E test (Sunday-night-fall): synthetic resident + fall event → kb-cql-runtime fires PostFall → kb-32 receives → context assembled → reasoning chain built → generator produces packet → appropriateness passes → framing applied → 4-layer formatter renders → recommendation transitions to `drafted`. Use InMemory fakes for the substrate client and HAPI client.

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(kb-32): HTTP API + lifecycle integration + Sunday-night-fall E2E test"
```

---

## Spec coverage

- [x] Service scaffold with replace directive (P0-2 fix) — Task 1
- [x] Lifecycle states rejected/withdrawn/superseded (Guidelines Part 3) — Task 2
- [x] Template enforcer (Guidelines Part 4 Stage 3) — Task 3
- [x] Stage 1 context assembler — Task 4
- [x] Stage 2 reasoning chain builder + HAPI placeholder fallback — Task 5
- [x] Stage 3 recommendation generator — Task 6
- [x] Evidence anchor selector AU-first ranking — Task 7
- [x] Orderer (anti-suppression) + urgency tagger — Task 8
- [x] Stage 4 appropriateness checker (5-dimension, hold ≤ 2) — Task 9
- [x] Stage 5 frame-vs-content separator with content_hash — Task 10
- [x] Per-GP framing observer with toxicity guard (Migrations 040/041, P0-1 fix) — Task 11
- [x] Stage 6 brevity formatter (4-layer, word budgets) — Task 12
- [x] HTTP API + lifecycle integration + E2E test — Task 13

**Out of scope for 2a (handled in 2b):**
- Override-reason taxonomy (20 categories + materialised view + capture API)
- Citation versioning (source_versions, recommendation_citations, supersession workflow)
- Negative-evidence patterns (CQL absence queries, evidence_checks table)
- Restraint signals (9 signal detectors)
- Expanded test suite (clinical-safety, metric-integrity, performance categories)

**Out of scope (UI/governance):**
- Layer 4 surfaces UX
- Pharmacy-employer view design
- Regulator audit interface

Plan complete and saved.
