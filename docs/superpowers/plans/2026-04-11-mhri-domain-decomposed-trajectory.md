# MHRI Domain-Decomposed Trajectory Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add per-domain OLS trajectory computation to KB-26 and domain-aware decision cards to KB-23, enabling clinicians to see which MHRI domain is driving deterioration, detect divergent domain trends, and receive behavioral leading indicator alerts.

**Architecture:** Extends KB-26 with a `ComputeDecomposedTrajectory()` function that runs OLS regression per-domain (glucose, cardio, body_comp, behavioral), then derives divergence patterns, leading indicators, dominant drivers, and category crossings. KB-23 consumes the output to generate 5 trajectory card types and integrates into the four-pillar evaluator. A new `domain_trajectory_history` table persists snapshots for trend-over-time analysis.

**Tech Stack:** Go 1.25 (KB-26), Go 1.22 (KB-23), PostgreSQL 15, Gin, GORM, stdlib testing

**Spec:** `docs/superpowers/specs/2026-04-11-mhri-domain-decomposed-trajectory-design.md`

---

## File Structure

### KB-26 (Metabolic Digital Twin)
| Action | File | Responsibility |
|--------|------|---------------|
| Create | `kb-26-metabolic-digital-twin/internal/models/domain_trajectory.go` | All domain trajectory data models |
| Create | `kb-26-metabolic-digital-twin/internal/services/mri_domain_trajectory.go` | Core decomposition engine + helpers |
| Create | `kb-26-metabolic-digital-twin/internal/services/mri_domain_trajectory_test.go` | 8 unit tests for decomposition |
| Create | `kb-26-metabolic-digital-twin/internal/services/domain_divergence.go` | Pairwise divergence detection + mechanism inference |
| Create | `kb-26-metabolic-digital-twin/internal/services/domain_divergence_test.go` | 4 unit tests for divergence |
| Create | `kb-26-metabolic-digital-twin/internal/api/domain_trajectory_handlers.go` | API handler for decomposed trajectory |
| Modify | `kb-26-metabolic-digital-twin/internal/api/routes.go` | Register new endpoint |
| Create | `kb-26-metabolic-digital-twin/migrations/006_domain_trajectory.sql` | History table DDL |

### KB-23 (Decision Cards)
| Action | File | Responsibility |
|--------|------|---------------|
| Create | `kb-23-decision-cards/internal/services/trajectory_card_rules.go` | 5 trajectory card types |
| Create | `kb-23-decision-cards/internal/services/trajectory_card_rules_test.go` | 5 unit tests for cards |
| Modify | `kb-23-decision-cards/internal/services/four_pillar_evaluator.go` | Add trajectory field + monitoring pillar recs |

### Shared Config
| Action | File | Responsibility |
|--------|------|---------------|
| Create | `market-configs/shared/domain_trajectory_thresholds.yaml` | Canonical threshold reference |

---

## Task 1: Domain Trajectory Data Models

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/models/domain_trajectory.go`

- [ ] **Step 1: Create the domain trajectory models file**

```go
// kb-26-metabolic-digital-twin/internal/models/domain_trajectory.go
package models

import "time"

// MHRIDomain identifies each of the four MHRI domains.
type MHRIDomain string

const (
	DomainGlucose    MHRIDomain = "GLUCOSE"
	DomainCardio     MHRIDomain = "CARDIO"
	DomainBodyComp   MHRIDomain = "BODY_COMP"
	DomainBehavioral MHRIDomain = "BEHAVIORAL"
)

// AllMHRIDomains lists all four domains for iteration.
var AllMHRIDomains = []MHRIDomain{DomainGlucose, DomainCardio, DomainBodyComp, DomainBehavioral}

// DomainTrajectoryPoint stores a single snapshot of all domain scores at a point in time.
type DomainTrajectoryPoint struct {
	Timestamp       time.Time `json:"timestamp"`
	CompositeScore  float64   `json:"composite_score"`
	GlucoseScore    float64   `json:"glucose_score"`
	CardioScore     float64   `json:"cardio_score"`
	BodyCompScore   float64   `json:"body_comp_score"`
	BehavioralScore float64   `json:"behavioral_score"`
}

// DomainSlope captures the OLS regression result for a single domain.
type DomainSlope struct {
	Domain      MHRIDomain `json:"domain"`
	SlopePerDay float64    `json:"slope_per_day"`
	Trend       string     `json:"trend"`       // RAPID_IMPROVING, IMPROVING, STABLE, DECLINING, RAPID_DECLINING
	StartScore  float64    `json:"start_score"`
	EndScore    float64    `json:"end_score"`
	DeltaScore  float64    `json:"delta_score"`
	R2          float64    `json:"r_squared"`  // goodness of fit
	Confidence  string     `json:"confidence"` // HIGH (R² >= 0.5), MODERATE (0.25-0.5), LOW (<0.25)
}

// DivergencePattern describes when two domains move in opposite directions.
type DivergencePattern struct {
	ImprovingDomain   MHRIDomain `json:"improving_domain"`
	DecliningDomain   MHRIDomain `json:"declining_domain"`
	ImprovingSlope    float64    `json:"improving_slope"`
	DecliningSlope    float64    `json:"declining_slope"`
	DivergenceRate    float64    `json:"divergence_rate"`
	ClinicalConcern   string     `json:"clinical_concern"`
	PossibleMechanism string     `json:"possible_mechanism"`
}

// LeadingIndicator describes when behavioral domain decline precedes clinical domain decline.
type LeadingIndicator struct {
	LeadingDomain  MHRIDomain   `json:"leading_domain"`
	LaggingDomains []MHRIDomain `json:"lagging_domains"`
	LeadDays       int          `json:"lead_days"`
	Confidence     string       `json:"confidence"`
	Interpretation string       `json:"interpretation"`
}

// DomainCategoryCrossing detects when a domain crosses an MHRI category boundary.
type DomainCategoryCrossing struct {
	Domain       MHRIDomain `json:"domain"`
	PrevCategory string     `json:"prev_category"` // OPTIMAL, MILD, MODERATE, HIGH
	CurrCategory string     `json:"curr_category"`
	Direction    string     `json:"direction"` // WORSENED, IMPROVED
	CrossingDate time.Time  `json:"crossing_date"`
}

// DecomposedTrajectory is the full output of the domain decomposition engine.
type DecomposedTrajectory struct {
	PatientID           string                       `json:"patient_id"`
	WindowDays          int                          `json:"window_days"`
	DataPoints          int                          `json:"data_points"`
	ComputedAt          time.Time                    `json:"computed_at"`
	CompositeSlope      float64                      `json:"composite_slope_per_day"`
	CompositeTrend      string                       `json:"composite_trend"`
	CompositeStartScore float64                      `json:"composite_start_score"`
	CompositeEndScore   float64                      `json:"composite_end_score"`
	DomainSlopes        map[MHRIDomain]DomainSlope   `json:"domain_slopes"`
	DominantDriver      *MHRIDomain                  `json:"dominant_driver,omitempty"`
	DriverContribution  float64                      `json:"driver_contribution,omitempty"`
	Divergences         []DivergencePattern          `json:"divergences,omitempty"`
	LeadingIndicators   []LeadingIndicator           `json:"leading_indicators,omitempty"`
	DomainCrossings     []DomainCategoryCrossing     `json:"domain_crossings,omitempty"`
	HasDiscordantTrend  bool                         `json:"has_discordant_trend"`
	ConcordantDeterioration bool                     `json:"concordant_deterioration"`
	DomainsDeteriorating    int                      `json:"domains_deteriorating"`
}

// DomainTrajectoryHistory stores decomposed snapshots for trend-over-time analysis.
type DomainTrajectoryHistory struct {
	ID              string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID       string    `gorm:"size:100;index:idx_dth_patient,priority:1;not null" json:"patient_id"`
	SnapshotDate    time.Time `gorm:"index:idx_dth_patient,priority:2,sort:desc;not null" json:"snapshot_date"`
	WindowDays      int       `json:"window_days"`
	CompositeSlope  float64   `json:"composite_slope"`
	GlucoseSlope    float64   `json:"glucose_slope"`
	CardioSlope     float64   `json:"cardio_slope"`
	BodyCompSlope   float64   `json:"body_comp_slope"`
	BehavioralSlope float64   `json:"behavioral_slope"`
	HasDiscordance  bool      `json:"has_discordance"`
	DominantDriver  string    `json:"dominant_driver,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

func (DomainTrajectoryHistory) TableName() string { return "domain_trajectory_history" }
```

- [ ] **Step 2: Verify the file compiles**

Run:
```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go build ./internal/models/
```

Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/models/domain_trajectory.go
git commit -m "feat(kb26): add domain trajectory data models

MHRIDomain enum (GLUCOSE/CARDIO/BODY_COMP/BEHAVIORAL), DomainSlope with
R² confidence, DivergencePattern, LeadingIndicator, DomainCategoryCrossing,
DecomposedTrajectory output struct, DomainTrajectoryHistory GORM model."
```

---

## Task 2: Domain Trajectory Thresholds YAML

**Files:**
- Create: `backend/shared-infrastructure/market-configs/shared/domain_trajectory_thresholds.yaml`

- [ ] **Step 1: Create the thresholds YAML**

```yaml
# Domain trajectory classification and alerting thresholds.
# Canonical reference — Go engine uses matching constants.

# Per-domain trend classification (score units per day)
trend_thresholds:
  rapid_improving: 1.0
  improving: 0.3
  stable_upper: 0.3
  stable_lower: -0.3
  declining: -0.3
  rapid_declining: -1.0

# Divergence detection
divergence:
  min_divergence_rate: 0.5
  min_improving_slope: 0.3
  min_declining_slope: -0.3

# Leading indicator detection
leading_indicator:
  behavioral_lead_threshold_days: 7
  min_behavioral_decline_slope: -0.5
  min_data_points: 10

# Concordant deterioration
concordant:
  min_domains_declining: 2
  min_slope_per_domain: -0.3

# Dominant driver analysis
driver:
  min_contribution_pct: 40.0
  weight_map:
    GLUCOSE: 0.35
    CARDIO: 0.25
    BODY_COMP: 0.25
    BEHAVIORAL: 0.15

# R² confidence thresholds
r_squared:
  high: 0.5
  moderate: 0.25

# Domain category boundaries (MHRI score ranges)
category_boundaries:
  optimal: 70
  mild: 55
  moderate: 40
```

- [ ] **Step 2: Commit**

```bash
git add backend/shared-infrastructure/market-configs/shared/domain_trajectory_thresholds.yaml
git commit -m "config: add domain trajectory thresholds YAML

Canonical reference for trend classification, divergence detection,
leading indicator, concordant deterioration, dominant driver, R² confidence,
and category boundary thresholds."
```

---

## Task 3: Core Decomposition Engine — Failing Tests

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/mri_domain_trajectory_test.go`

- [ ] **Step 1: Write the test file with 8 test cases**

All tests use stdlib `testing` package (matching existing `egfr_trajectory_test.go` pattern — no testify).

```go
package services

import (
	"testing"
	"time"

	"kb-26-metabolic-digital-twin/internal/models"
)

// ---------------------------------------------------------------------------
// TestDomainTrajectory_GlucoseDeclining_CardioStable
// ---------------------------------------------------------------------------

func TestDomainTrajectory_GlucoseDeclining_CardioStable(t *testing.T) {
	now := time.Now()
	points := []models.DomainTrajectoryPoint{
		{Timestamp: now.Add(-13 * 24 * time.Hour), CompositeScore: 72, GlucoseScore: 75, CardioScore: 70, BodyCompScore: 68, BehavioralScore: 72},
		{Timestamp: now.Add(-10 * 24 * time.Hour), CompositeScore: 68, GlucoseScore: 65, CardioScore: 71, BodyCompScore: 67, BehavioralScore: 70},
		{Timestamp: now.Add(-7 * 24 * time.Hour), CompositeScore: 64, GlucoseScore: 55, CardioScore: 70, BodyCompScore: 68, BehavioralScore: 71},
		{Timestamp: now.Add(-4 * 24 * time.Hour), CompositeScore: 60, GlucoseScore: 48, CardioScore: 69, BodyCompScore: 67, BehavioralScore: 70},
		{Timestamp: now.Add(-1 * 24 * time.Hour), CompositeScore: 56, GlucoseScore: 42, CardioScore: 70, BodyCompScore: 68, BehavioralScore: 71},
	}

	result := ComputeDecomposedTrajectory("PAT-001", points)

	// Composite should show decline
	if result.CompositeTrend != "DECLINING" && result.CompositeTrend != "RAPID_DECLINING" {
		t.Errorf("expected composite DECLINING or RAPID_DECLINING, got %s", result.CompositeTrend)
	}

	// Glucose domain should be the primary decliner
	glucoseSlope := result.DomainSlopes[models.DomainGlucose]
	if glucoseSlope.Trend != "RAPID_DECLINING" {
		t.Errorf("expected glucose RAPID_DECLINING, got %s (slope=%.3f)", glucoseSlope.Trend, glucoseSlope.SlopePerDay)
	}
	if glucoseSlope.SlopePerDay >= -1.0 {
		t.Errorf("expected glucose slope < -1.0, got %.3f", glucoseSlope.SlopePerDay)
	}

	// Cardio should be stable
	cardioSlope := result.DomainSlopes[models.DomainCardio]
	if cardioSlope.Trend != "STABLE" {
		t.Errorf("expected cardio STABLE, got %s (slope=%.3f)", cardioSlope.Trend, cardioSlope.SlopePerDay)
	}

	// Dominant driver should be glucose
	if result.DominantDriver == nil {
		t.Fatal("expected non-nil DominantDriver")
	}
	if *result.DominantDriver != models.DomainGlucose {
		t.Errorf("expected dominant driver GLUCOSE, got %s", *result.DominantDriver)
	}
	if result.DriverContribution < 40.0 {
		t.Errorf("expected driver contribution >= 40%%, got %.1f%%", result.DriverContribution)
	}
}

// ---------------------------------------------------------------------------
// TestDomainTrajectory_AllDomainsImproving
// ---------------------------------------------------------------------------

func TestDomainTrajectory_AllDomainsImproving(t *testing.T) {
	now := time.Now()
	points := []models.DomainTrajectoryPoint{
		{Timestamp: now.Add(-13 * 24 * time.Hour), CompositeScore: 50, GlucoseScore: 45, CardioScore: 50, BodyCompScore: 52, BehavioralScore: 55},
		{Timestamp: now.Add(-7 * 24 * time.Hour), CompositeScore: 58, GlucoseScore: 55, CardioScore: 58, BodyCompScore: 58, BehavioralScore: 62},
		{Timestamp: now.Add(-1 * 24 * time.Hour), CompositeScore: 66, GlucoseScore: 65, CardioScore: 66, BodyCompScore: 64, BehavioralScore: 70},
	}

	result := ComputeDecomposedTrajectory("PAT-002", points)

	if result.CompositeTrend != "IMPROVING" && result.CompositeTrend != "RAPID_IMPROVING" {
		t.Errorf("expected composite IMPROVING or RAPID_IMPROVING, got %s", result.CompositeTrend)
	}
	for _, domain := range models.AllMHRIDomains {
		slope := result.DomainSlopes[domain]
		if slope.SlopePerDay <= 0 {
			t.Errorf("expected %s to have positive slope, got %.3f", domain, slope.SlopePerDay)
		}
	}
	if result.HasDiscordantTrend {
		t.Error("expected HasDiscordantTrend = false for all-improving")
	}
	if result.DomainsDeteriorating != 0 {
		t.Errorf("expected 0 domains deteriorating, got %d", result.DomainsDeteriorating)
	}
}

// ---------------------------------------------------------------------------
// TestDomainTrajectory_ConcordantDeterioration
// ---------------------------------------------------------------------------

func TestDomainTrajectory_ConcordantDeterioration(t *testing.T) {
	now := time.Now()
	points := []models.DomainTrajectoryPoint{
		{Timestamp: now.Add(-13 * 24 * time.Hour), CompositeScore: 70, GlucoseScore: 72, CardioScore: 70, BodyCompScore: 68, BehavioralScore: 68},
		{Timestamp: now.Add(-7 * 24 * time.Hour), CompositeScore: 62, GlucoseScore: 60, CardioScore: 58, BodyCompScore: 66, BehavioralScore: 65},
		{Timestamp: now.Add(-1 * 24 * time.Hour), CompositeScore: 52, GlucoseScore: 48, CardioScore: 45, BodyCompScore: 64, BehavioralScore: 55},
	}

	result := ComputeDecomposedTrajectory("PAT-003", points)

	if !result.ConcordantDeterioration {
		t.Error("expected ConcordantDeterioration = true")
	}
	if result.DomainsDeteriorating < 2 {
		t.Errorf("expected >= 2 domains deteriorating, got %d", result.DomainsDeteriorating)
	}
}

// ---------------------------------------------------------------------------
// TestDomainTrajectory_InsufficientData
// ---------------------------------------------------------------------------

func TestDomainTrajectory_InsufficientData(t *testing.T) {
	now := time.Now()
	points := []models.DomainTrajectoryPoint{
		{Timestamp: now, CompositeScore: 70, GlucoseScore: 72, CardioScore: 70, BodyCompScore: 68, BehavioralScore: 68},
	}

	result := ComputeDecomposedTrajectory("PAT-004", points)
	if result.CompositeTrend != "INSUFFICIENT_DATA" {
		t.Errorf("expected INSUFFICIENT_DATA for composite, got %s", result.CompositeTrend)
	}
	for _, domain := range models.AllMHRIDomains {
		if result.DomainSlopes[domain].Trend != "INSUFFICIENT_DATA" {
			t.Errorf("expected INSUFFICIENT_DATA for %s, got %s", domain, result.DomainSlopes[domain].Trend)
		}
	}
}

// ---------------------------------------------------------------------------
// TestDomainTrajectory_NoisyData_LowConfidence
// ---------------------------------------------------------------------------

func TestDomainTrajectory_NoisyData_LowConfidence(t *testing.T) {
	now := time.Now()
	// Glucose scores oscillating wildly — no clear trend
	points := []models.DomainTrajectoryPoint{
		{Timestamp: now.Add(-13 * 24 * time.Hour), GlucoseScore: 70, CardioScore: 65, BodyCompScore: 60, BehavioralScore: 65, CompositeScore: 65},
		{Timestamp: now.Add(-11 * 24 * time.Hour), GlucoseScore: 45, CardioScore: 65, BodyCompScore: 60, BehavioralScore: 65, CompositeScore: 58},
		{Timestamp: now.Add(-9 * 24 * time.Hour), GlucoseScore: 75, CardioScore: 65, BodyCompScore: 60, BehavioralScore: 65, CompositeScore: 66},
		{Timestamp: now.Add(-7 * 24 * time.Hour), GlucoseScore: 40, CardioScore: 65, BodyCompScore: 60, BehavioralScore: 65, CompositeScore: 57},
		{Timestamp: now.Add(-5 * 24 * time.Hour), GlucoseScore: 72, CardioScore: 65, BodyCompScore: 60, BehavioralScore: 65, CompositeScore: 65},
		{Timestamp: now.Add(-3 * 24 * time.Hour), GlucoseScore: 42, CardioScore: 65, BodyCompScore: 60, BehavioralScore: 65, CompositeScore: 58},
		{Timestamp: now.Add(-1 * 24 * time.Hour), GlucoseScore: 68, CardioScore: 65, BodyCompScore: 60, BehavioralScore: 65, CompositeScore: 64},
	}

	result := ComputeDecomposedTrajectory("PAT-005", points)
	glucoseSlope := result.DomainSlopes[models.DomainGlucose]
	if glucoseSlope.Confidence != "LOW" {
		t.Errorf("expected LOW confidence for noisy glucose, got %s (R²=%.3f)", glucoseSlope.Confidence, glucoseSlope.R2)
	}
}

// ---------------------------------------------------------------------------
// TestDomainCategoryCrossing_GlucoseOptimalToMild
// ---------------------------------------------------------------------------

func TestDomainCategoryCrossing_GlucoseOptimalToMild(t *testing.T) {
	now := time.Now()
	points := []models.DomainTrajectoryPoint{
		{Timestamp: now.Add(-7 * 24 * time.Hour), GlucoseScore: 72, CardioScore: 70, BodyCompScore: 68, BehavioralScore: 70, CompositeScore: 70},
		{Timestamp: now.Add(-1 * 24 * time.Hour), GlucoseScore: 66, CardioScore: 70, BodyCompScore: 68, BehavioralScore: 70, CompositeScore: 68},
	}

	result := ComputeDecomposedTrajectory("PAT-006", points)
	if len(result.DomainCrossings) == 0 {
		t.Fatal("expected at least one domain crossing")
	}

	found := false
	for _, c := range result.DomainCrossings {
		if c.Domain == models.DomainGlucose {
			found = true
			if c.PrevCategory != "OPTIMAL" {
				t.Errorf("expected prev category OPTIMAL, got %s", c.PrevCategory)
			}
			if c.CurrCategory != "MILD" {
				t.Errorf("expected curr category MILD, got %s", c.CurrCategory)
			}
			if c.Direction != "WORSENED" {
				t.Errorf("expected direction WORSENED, got %s", c.Direction)
			}
		}
	}
	if !found {
		t.Error("expected glucose domain crossing not found")
	}
}

// ---------------------------------------------------------------------------
// TestDomainTrajectory_ZeroPoints
// ---------------------------------------------------------------------------

func TestDomainTrajectory_ZeroPoints(t *testing.T) {
	result := ComputeDecomposedTrajectory("PAT-007", nil)
	if result.CompositeTrend != "INSUFFICIENT_DATA" {
		t.Errorf("expected INSUFFICIENT_DATA for nil points, got %s", result.CompositeTrend)
	}
}

// ---------------------------------------------------------------------------
// TestDomainTrajectory_RajeshKumar — E2E scenario
// ---------------------------------------------------------------------------

func TestDomainTrajectory_RajeshKumar(t *testing.T) {
	now := time.Now()
	// Rajesh: glucose declining, cardio declining, behavioral collapsing, body comp stable
	points := []models.DomainTrajectoryPoint{
		{Timestamp: now.Add(-13 * 24 * time.Hour), CompositeScore: 62, GlucoseScore: 55, CardioScore: 58, BodyCompScore: 65, BehavioralScore: 72},
		{Timestamp: now.Add(-10 * 24 * time.Hour), CompositeScore: 58, GlucoseScore: 50, CardioScore: 52, BodyCompScore: 65, BehavioralScore: 65},
		{Timestamp: now.Add(-7 * 24 * time.Hour), CompositeScore: 53, GlucoseScore: 45, CardioScore: 48, BodyCompScore: 64, BehavioralScore: 55},
		{Timestamp: now.Add(-4 * 24 * time.Hour), CompositeScore: 48, GlucoseScore: 40, CardioScore: 42, BodyCompScore: 64, BehavioralScore: 42},
		{Timestamp: now.Add(-1 * 24 * time.Hour), CompositeScore: 42, GlucoseScore: 35, CardioScore: 38, BodyCompScore: 63, BehavioralScore: 30},
	}

	result := ComputeDecomposedTrajectory("e2e-rajesh-kumar-002", points)

	// Composite declining
	if result.CompositeTrend != "DECLINING" && result.CompositeTrend != "RAPID_DECLINING" {
		t.Errorf("expected DECLINING or RAPID_DECLINING, got %s", result.CompositeTrend)
	}

	// Three domains declining
	if result.DomainsDeteriorating < 3 {
		t.Errorf("expected >= 3 domains deteriorating, got %d", result.DomainsDeteriorating)
	}
	if !result.ConcordantDeterioration {
		t.Error("expected ConcordantDeterioration = true")
	}

	// Body comp should be stable
	bcSlope := result.DomainSlopes[models.DomainBodyComp]
	if bcSlope.Trend != "STABLE" {
		t.Errorf("expected body comp STABLE, got %s (slope=%.3f)", bcSlope.Trend, bcSlope.SlopePerDay)
	}

	// Behavioral should be the fastest decliner (72 -> 30 = -42 over 12 days)
	behSlope := result.DomainSlopes[models.DomainBehavioral]
	if behSlope.Trend != "RAPID_DECLINING" {
		t.Errorf("expected behavioral RAPID_DECLINING, got %s (slope=%.3f)", behSlope.Trend, behSlope.SlopePerDay)
	}

	// Multiple category crossings should be detected
	if len(result.DomainCrossings) < 2 {
		t.Errorf("expected >= 2 category crossings, got %d", len(result.DomainCrossings))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:
```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/services/ -run "TestDomainTrajectory|TestDomainCategory" -v 2>&1 | head -20
```

Expected: FAIL — `ComputeDecomposedTrajectory` undefined.

- [ ] **Step 3: Commit failing tests**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/mri_domain_trajectory_test.go
git commit -m "test(kb26): add failing tests for domain trajectory computation

8 tests: glucose-declining/cardio-stable, all-improving, concordant
deterioration, insufficient data, noisy R², category crossing,
zero points, Rajesh Kumar E2E scenario."
```

---

## Task 4: Core Decomposition Engine — Implementation

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/mri_domain_trajectory.go`

- [ ] **Step 1: Create the decomposition engine**

```go
package services

import (
	"math"
	"time"

	"kb-26-metabolic-digital-twin/internal/models"
)

// MHRI domain weights (must match MHRI scorer weights).
var domainWeights = map[models.MHRIDomain]float64{
	models.DomainGlucose:    0.35,
	models.DomainCardio:     0.25,
	models.DomainBodyComp:   0.25,
	models.DomainBehavioral: 0.15,
}

// Trend thresholds (score units per day).
const (
	rapidImprovingThreshold = 1.0
	improvingThreshold      = 0.3
	decliningThreshold      = -0.3
	rapidDecliningThreshold = -1.0
)

// Category boundaries (MHRI score ranges).
const (
	categoryOptimal  = 70.0
	categoryMild     = 55.0
	categoryModerate = 40.0
)

// ComputeDecomposedTrajectory computes per-domain OLS trajectories and derived analytics.
func ComputeDecomposedTrajectory(patientID string, points []models.DomainTrajectoryPoint) models.DecomposedTrajectory {
	result := models.DecomposedTrajectory{
		PatientID:    patientID,
		DataPoints:   len(points),
		ComputedAt:   time.Now(),
		DomainSlopes: make(map[models.MHRIDomain]models.DomainSlope),
	}

	if len(points) < 2 {
		result.CompositeTrend = "INSUFFICIENT_DATA"
		for _, d := range models.AllMHRIDomains {
			result.DomainSlopes[d] = models.DomainSlope{Domain: d, Trend: "INSUFFICIENT_DATA"}
		}
		return result
	}

	// Sort by timestamp.
	sorted := make([]models.DomainTrajectoryPoint, len(points))
	copy(sorted, points)
	sortTrajectoryPoints(sorted)

	first, last := sorted[0], sorted[len(sorted)-1]
	result.WindowDays = int(last.Timestamp.Sub(first.Timestamp).Hours() / 24)

	// Compute composite trajectory.
	compositeScores := extractScores(sorted, func(p models.DomainTrajectoryPoint) float64 { return p.CompositeScore })
	compSlope, _ := computeOLSWithR2(sorted, compositeScores)
	result.CompositeSlope = roundTo3(compSlope)
	result.CompositeTrend = classifyDomainTrend(compSlope)
	result.CompositeStartScore = first.CompositeScore
	result.CompositeEndScore = last.CompositeScore

	// Compute per-domain trajectories.
	domainExtractors := map[models.MHRIDomain]func(models.DomainTrajectoryPoint) float64{
		models.DomainGlucose:    func(p models.DomainTrajectoryPoint) float64 { return p.GlucoseScore },
		models.DomainCardio:     func(p models.DomainTrajectoryPoint) float64 { return p.CardioScore },
		models.DomainBodyComp:   func(p models.DomainTrajectoryPoint) float64 { return p.BodyCompScore },
		models.DomainBehavioral: func(p models.DomainTrajectoryPoint) float64 { return p.BehavioralScore },
	}

	decliningCount := 0
	var maxWeightedDecline float64
	var dominantDriver *models.MHRIDomain

	for domain, extractor := range domainExtractors {
		scores := extractScores(sorted, extractor)
		slope, r2 := computeOLSWithR2(sorted, scores)

		ds := models.DomainSlope{
			Domain:      domain,
			SlopePerDay: roundTo3(slope),
			Trend:       classifyDomainTrend(slope),
			StartScore:  scores[0],
			EndScore:    scores[len(scores)-1],
			DeltaScore:  roundTo1(scores[len(scores)-1] - scores[0]),
			R2:          roundTo3(r2),
			Confidence:  classifyR2Confidence(r2),
		}
		result.DomainSlopes[domain] = ds

		if slope < decliningThreshold {
			decliningCount++
		}

		weightedDecline := math.Abs(slope) * domainWeights[domain]
		if slope < 0 && weightedDecline > maxWeightedDecline {
			maxWeightedDecline = weightedDecline
			d := domain
			dominantDriver = &d
		}
	}

	result.DomainsDeteriorating = decliningCount
	result.ConcordantDeterioration = decliningCount >= 2

	// Dominant driver calculation.
	if dominantDriver != nil && result.CompositeSlope < 0 {
		result.DominantDriver = dominantDriver
		totalWeightedDecline := 0.0
		for domain, ds := range result.DomainSlopes {
			if ds.SlopePerDay < 0 {
				totalWeightedDecline += math.Abs(ds.SlopePerDay) * domainWeights[domain]
			}
		}
		if totalWeightedDecline > 0 {
			result.DriverContribution = roundTo1((maxWeightedDecline / totalWeightedDecline) * 100)
		}
	}

	// Detect divergence patterns.
	result.Divergences = detectDivergences(result.DomainSlopes)
	result.HasDiscordantTrend = len(result.Divergences) > 0

	// Detect domain category crossings.
	result.DomainCrossings = detectDomainCrossings(sorted, domainExtractors)

	// Detect behavioral leading indicator.
	result.LeadingIndicators = detectLeadingIndicators(sorted, result.DomainSlopes)

	return result
}

// computeOLSWithR2 runs OLS linear regression returning slope (per day) and R².
func computeOLSWithR2(points []models.DomainTrajectoryPoint, scores []float64) (float64, float64) {
	if len(points) < 2 {
		return 0, 0
	}

	baseTime := points[0].Timestamp
	n := float64(len(points))
	var sumX, sumY, sumXY, sumX2, sumY2 float64

	for i, pt := range points {
		x := pt.Timestamp.Sub(baseTime).Hours() / 24.0 // days from first point
		y := scores[i]
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
		sumY2 += y * y
	}

	denom := n*sumX2 - sumX*sumX
	if math.Abs(denom) < 1e-10 {
		return 0, 0
	}

	slope := (n*sumXY - sumX*sumY) / denom
	intercept := (sumY - slope*sumX) / n

	// R² calculation.
	meanY := sumY / n
	ssTot := sumY2 - n*meanY*meanY
	ssRes := 0.0
	for i, pt := range points {
		x := pt.Timestamp.Sub(baseTime).Hours() / 24.0
		predicted := intercept + slope*x
		residual := scores[i] - predicted
		ssRes += residual * residual
	}

	r2 := 0.0
	if ssTot > 1e-10 {
		r2 = 1 - (ssRes / ssTot)
		if r2 < 0 {
			r2 = 0
		}
	}

	return slope, r2
}

func classifyDomainTrend(slopePerDay float64) string {
	switch {
	case slopePerDay > rapidImprovingThreshold:
		return "RAPID_IMPROVING"
	case slopePerDay > improvingThreshold:
		return "IMPROVING"
	case slopePerDay >= decliningThreshold:
		return "STABLE"
	case slopePerDay >= rapidDecliningThreshold:
		return "DECLINING"
	default:
		return "RAPID_DECLINING"
	}
}

func classifyR2Confidence(r2 float64) string {
	if r2 >= 0.5 {
		return "HIGH"
	}
	if r2 >= 0.25 {
		return "MODERATE"
	}
	return "LOW"
}

func detectDomainCrossings(points []models.DomainTrajectoryPoint, extractors map[models.MHRIDomain]func(models.DomainTrajectoryPoint) float64) []models.DomainCategoryCrossing {
	if len(points) < 2 {
		return nil
	}

	first := points[0]
	last := points[len(points)-1]
	var crossings []models.DomainCategoryCrossing

	for domain, extractor := range extractors {
		startScore := extractor(first)
		endScore := extractor(last)
		startCat := categorizeDomainScore(startScore)
		endCat := categorizeDomainScore(endScore)

		if startCat != endCat {
			direction := "IMPROVED"
			if endScore < startScore {
				direction = "WORSENED"
			}
			crossings = append(crossings, models.DomainCategoryCrossing{
				Domain:       domain,
				PrevCategory: startCat,
				CurrCategory: endCat,
				Direction:    direction,
				CrossingDate: last.Timestamp,
			})
		}
	}

	return crossings
}

func detectLeadingIndicators(points []models.DomainTrajectoryPoint, slopes map[models.MHRIDomain]models.DomainSlope) []models.LeadingIndicator {
	if len(points) < 5 {
		return nil // need enough data for lead-lag
	}

	behSlope := slopes[models.DomainBehavioral]
	if behSlope.SlopePerDay >= -0.5 {
		return nil // behavioral not declining meaningfully
	}

	// Check if behavioral started declining before clinical domains.
	var lagging []models.MHRIDomain
	for _, domain := range []models.MHRIDomain{models.DomainGlucose, models.DomainCardio} {
		ds := slopes[domain]
		if ds.SlopePerDay < decliningThreshold {
			// Clinical domain also declining — check if behavioral dropped more.
			if behSlope.DeltaScore < ds.DeltaScore {
				lagging = append(lagging, domain)
			}
		}
	}

	if len(lagging) == 0 {
		return nil
	}

	return []models.LeadingIndicator{{
		LeadingDomain:  models.DomainBehavioral,
		LaggingDomains: lagging,
		Confidence:     "MODERATE",
		Interpretation: "Behavioral domain decline preceded clinical domain deterioration — engagement collapse may be driving worsening outcomes",
	}}
}

func categorizeDomainScore(score float64) string {
	if score >= categoryOptimal {
		return "OPTIMAL"
	}
	if score >= categoryMild {
		return "MILD"
	}
	if score >= categoryModerate {
		return "MODERATE"
	}
	return "HIGH"
}

func extractScores(points []models.DomainTrajectoryPoint, extractor func(models.DomainTrajectoryPoint) float64) []float64 {
	scores := make([]float64, len(points))
	for i, p := range points {
		scores[i] = extractor(p)
	}
	return scores
}

func sortTrajectoryPoints(pts []models.DomainTrajectoryPoint) {
	for i := 1; i < len(pts); i++ {
		for j := i; j > 0 && pts[j].Timestamp.Before(pts[j-1].Timestamp); j-- {
			pts[j], pts[j-1] = pts[j-1], pts[j]
		}
	}
}

func roundTo3(v float64) float64 { return math.Round(v*1000) / 1000 }
func roundTo1(v float64) float64 { return math.Round(v*10) / 10 }
```

- [ ] **Step 2: Run tests to verify they pass**

Run:
```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/services/ -run "TestDomainTrajectory|TestDomainCategory" -v
```

Expected: All 8 tests PASS.

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/mri_domain_trajectory.go
git commit -m "feat(kb26): implement domain trajectory decomposition engine

Per-domain OLS with R² confidence (HIGH/MODERATE/LOW). 5-tier trend
classification. Dominant driver with weighted contribution %. Concordant
deterioration (>=2 domains). Category crossing detection. Behavioral
leading indicator. All 8 tests pass."
```

---

## Task 5: Divergence Detection — Failing Tests

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/domain_divergence_test.go`

- [ ] **Step 1: Write the divergence test file with 4 test cases**

```go
package services

import (
	"testing"

	"kb-26-metabolic-digital-twin/internal/models"
)

// ---------------------------------------------------------------------------
// TestDetectDivergence_GlucoseImproving_CardioDeclining
// ---------------------------------------------------------------------------

func TestDetectDivergence_GlucoseImproving_CardioDeclining(t *testing.T) {
	slopes := map[models.MHRIDomain]models.DomainSlope{
		models.DomainGlucose:    {Domain: models.DomainGlucose, SlopePerDay: 0.8, Trend: "IMPROVING"},
		models.DomainCardio:     {Domain: models.DomainCardio, SlopePerDay: -1.2, Trend: "RAPID_DECLINING"},
		models.DomainBodyComp:   {Domain: models.DomainBodyComp, SlopePerDay: 0.1, Trend: "STABLE"},
		models.DomainBehavioral: {Domain: models.DomainBehavioral, SlopePerDay: 0.0, Trend: "STABLE"},
	}

	divergences := detectDivergences(slopes)
	if len(divergences) != 1 {
		t.Fatalf("expected 1 divergence, got %d", len(divergences))
	}
	if divergences[0].ImprovingDomain != models.DomainGlucose {
		t.Errorf("expected improving domain GLUCOSE, got %s", divergences[0].ImprovingDomain)
	}
	if divergences[0].DecliningDomain != models.DomainCardio {
		t.Errorf("expected declining domain CARDIO, got %s", divergences[0].DecliningDomain)
	}
	// |0.8| + |-1.2| = 2.0
	if divergences[0].DivergenceRate < 1.5 {
		t.Errorf("expected divergence rate >= 1.5, got %.3f", divergences[0].DivergenceRate)
	}
	if divergences[0].ClinicalConcern == "" {
		t.Error("expected non-empty ClinicalConcern")
	}
}

// ---------------------------------------------------------------------------
// TestDetectDivergence_NoDivergence_AllStable
// ---------------------------------------------------------------------------

func TestDetectDivergence_NoDivergence_AllStable(t *testing.T) {
	slopes := map[models.MHRIDomain]models.DomainSlope{
		models.DomainGlucose:    {SlopePerDay: 0.1},
		models.DomainCardio:     {SlopePerDay: -0.1},
		models.DomainBodyComp:   {SlopePerDay: 0.05},
		models.DomainBehavioral: {SlopePerDay: -0.05},
	}

	divergences := detectDivergences(slopes)
	if len(divergences) != 0 {
		t.Errorf("expected 0 divergences for all-stable, got %d", len(divergences))
	}
}

// ---------------------------------------------------------------------------
// TestDetectDivergence_MultiplePairs
// ---------------------------------------------------------------------------

func TestDetectDivergence_MultiplePairs(t *testing.T) {
	slopes := map[models.MHRIDomain]models.DomainSlope{
		models.DomainGlucose:    {Domain: models.DomainGlucose, SlopePerDay: 1.0, Trend: "RAPID_IMPROVING"},
		models.DomainCardio:     {Domain: models.DomainCardio, SlopePerDay: -0.8, Trend: "DECLINING"},
		models.DomainBodyComp:   {Domain: models.DomainBodyComp, SlopePerDay: 0.5, Trend: "IMPROVING"},
		models.DomainBehavioral: {Domain: models.DomainBehavioral, SlopePerDay: -0.6, Trend: "DECLINING"},
	}

	divergences := detectDivergences(slopes)
	if len(divergences) < 2 {
		t.Errorf("expected >= 2 divergences (glucose/cardio + bodycomp/behavioral), got %d", len(divergences))
	}
}

// ---------------------------------------------------------------------------
// TestDivergence_ClinicalConcernText
// ---------------------------------------------------------------------------

func TestDivergence_ClinicalConcernText(t *testing.T) {
	slopes := map[models.MHRIDomain]models.DomainSlope{
		models.DomainGlucose:    {Domain: models.DomainGlucose, SlopePerDay: 0.8, Trend: "IMPROVING"},
		models.DomainCardio:     {Domain: models.DomainCardio, SlopePerDay: -0.9, Trend: "DECLINING"},
		models.DomainBodyComp:   {Domain: models.DomainBodyComp, SlopePerDay: 0.0, Trend: "STABLE"},
		models.DomainBehavioral: {Domain: models.DomainBehavioral, SlopePerDay: 0.0, Trend: "STABLE"},
	}

	divergences := detectDivergences(slopes)
	if len(divergences) != 1 {
		t.Fatalf("expected 1 divergence, got %d", len(divergences))
	}

	concern := divergences[0].ClinicalConcern
	if concern == "" {
		t.Error("expected non-empty ClinicalConcern")
	}

	mechanism := divergences[0].PossibleMechanism
	if mechanism == "" {
		t.Error("expected non-empty PossibleMechanism")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:
```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/services/ -run "TestDetectDivergence|TestDivergence_" -v 2>&1 | head -20
```

Expected: FAIL — `detectDivergences` undefined.

- [ ] **Step 3: Commit failing tests**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/domain_divergence_test.go
git commit -m "test(kb26): add failing tests for divergence detection

4 tests: single divergence pair, no divergence (all stable), multiple
pairs, clinical concern text populated."
```

---

## Task 6: Divergence Detection — Implementation

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/domain_divergence.go`

- [ ] **Step 1: Create the divergence detection engine**

```go
package services

import (
	"fmt"
	"math"

	"kb-26-metabolic-digital-twin/internal/models"
)

const minDivergenceRate = 0.5

// detectDivergences finds pairs of domains moving in opposite directions.
func detectDivergences(slopes map[models.MHRIDomain]models.DomainSlope) []models.DivergencePattern {
	var divergences []models.DivergencePattern
	domains := models.AllMHRIDomains

	for i := 0; i < len(domains); i++ {
		for j := i + 1; j < len(domains); j++ {
			slopeA := slopes[domains[i]]
			slopeB := slopes[domains[j]]

			// One must be genuinely improving, the other genuinely declining.
			var improving, declining models.DomainSlope
			if slopeA.SlopePerDay > improvingThreshold && slopeB.SlopePerDay < decliningThreshold {
				improving = slopeA
				declining = slopeB
			} else if slopeB.SlopePerDay > improvingThreshold && slopeA.SlopePerDay < decliningThreshold {
				improving = slopeB
				declining = slopeA
			} else {
				continue
			}

			divergenceRate := math.Abs(improving.SlopePerDay) + math.Abs(declining.SlopePerDay)
			if divergenceRate < minDivergenceRate {
				continue
			}

			divergences = append(divergences, models.DivergencePattern{
				ImprovingDomain: improving.Domain,
				DecliningDomain: declining.Domain,
				ImprovingSlope:  improving.SlopePerDay,
				DecliningSlope:  declining.SlopePerDay,
				DivergenceRate:  roundTo3(divergenceRate),
				ClinicalConcern: fmt.Sprintf("%s improving while %s declining — therapeutic attention may be misdirected",
					improving.Domain, declining.Domain),
				PossibleMechanism: inferDivergenceMechanism(improving.Domain, declining.Domain),
			})
		}
	}

	return divergences
}

// inferDivergenceMechanism provides clinical hypotheses for specific divergence pairs.
func inferDivergenceMechanism(improving, declining models.MHRIDomain) string {
	key := string(improving) + "_" + string(declining)
	mechanisms := map[string]string{
		"GLUCOSE_CARDIO": "Glycaemic therapy may lack hemodynamic benefit, or antihypertensive review needed. " +
			"Consider SGLT2i (dual glucose + BP benefit) or adding dedicated antihypertensive.",
		"CARDIO_GLUCOSE": "BP medications may be worsening glycaemic control (e.g., thiazide raising glucose, " +
			"beta-blocker masking hypoglycaemia). Review cross-domain drug effects.",
		"GLUCOSE_BEHAVIORAL": "Glycaemic markers improving on medication but patient disengaging from self-management. " +
			"Improvement may not sustain without behavioral re-engagement.",
		"BEHAVIORAL_GLUCOSE": "Patient engaged and self-managing but glycaemic control worsening — suggests " +
			"medication inadequacy rather than adherence problem. Intensify pharmacotherapy.",
		"CARDIO_BEHAVIORAL": "BP improving but engagement declining — may indicate medication working but patient " +
			"developing complacency. Monitor for future adherence-related BP rebound.",
		"BEHAVIORAL_CARDIO": "Patient engaged but cardiovascular metrics worsening despite adherence — " +
			"suggests medication resistance, secondary hypertension workup, or emerging cardiac pathology.",
		"GLUCOSE_BODY_COMP": "Glycaemic control improving while body composition worsening — " +
			"check for insulin-driven weight gain or thiazolidinedione fluid retention.",
		"BODY_COMP_GLUCOSE": "Body composition improving (weight loss) but glucose worsening — " +
			"paradoxical in T2DM. Investigate: stress hyperglycaemia, steroid use, pancreatic pathology.",
		"CARDIO_BODY_COMP": "Cardiovascular metrics improving while body composition declining — " +
			"may indicate effective medication but dietary non-adherence.",
		"BODY_COMP_CARDIO": "Weight management improving but CV worsening — " +
			"consider sleep apnea, endocrine causes of hypertension, or medication interaction.",
	}

	if m, ok := mechanisms[key]; ok {
		return m
	}
	return "Domain divergence detected — clinical review recommended to identify cause and adjust therapy."
}
```

- [ ] **Step 2: Run all KB-26 service tests**

Run:
```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/services/ -run "TestDomainTrajectory|TestDomainCategory|TestDetectDivergence|TestDivergence_" -v
```

Expected: All 12 tests PASS (8 domain trajectory + 4 divergence).

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/domain_divergence.go
git commit -m "feat(kb26): domain divergence detection with clinical mechanism inference

Pairwise divergence across 4 MHRI domains. Min divergence rate 0.5.
Clinical mechanism hypotheses for 10 domain pairs: glucose/cardio
(SGLT2i), cardio/glucose (cross-domain drug effects), behavioral
leading patterns, body comp interactions. All 12 tests pass."
```

---

## Task 7: Trajectory Card Rules — Failing Tests (KB-23)

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/trajectory_card_rules_test.go`

**Important:** KB-23 needs to import KB-26 models. Before writing tests, we need to set up the module dependency.

- [ ] **Step 1: Add KB-26 as a dependency in KB-23 go.mod**

Run:
```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards
```

Add to `go.mod` — append a `replace` directive and a `require` for KB-26:

At the end of `go.mod`, add:

```
replace kb-26-metabolic-digital-twin => ../kb-26-metabolic-digital-twin
```

Then add to the `require` block:

```
kb-26-metabolic-digital-twin v0.0.0
```

Run:
```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go mod tidy
```

Expected: No errors. The `replace` directive points to the sibling directory.

- [ ] **Step 2: Write the trajectory card test file with 5 test cases**

```go
package services

import (
	"testing"

	dtModels "kb-26-metabolic-digital-twin/internal/models"
)

// ---------------------------------------------------------------------------
// TestTrajectoryCards_ConcordantDeterioration
// ---------------------------------------------------------------------------

func TestTrajectoryCards_ConcordantDeterioration(t *testing.T) {
	traj := &dtModels.DecomposedTrajectory{
		CompositeTrend:          "DECLINING",
		CompositeSlope:          -1.5,
		ConcordantDeterioration: true,
		DomainsDeteriorating:    3,
		DomainSlopes: map[dtModels.MHRIDomain]dtModels.DomainSlope{
			dtModels.DomainGlucose:    {Trend: "DECLINING", SlopePerDay: -0.8},
			dtModels.DomainCardio:     {Trend: "RAPID_DECLINING", SlopePerDay: -1.5},
			dtModels.DomainBehavioral: {Trend: "DECLINING", SlopePerDay: -0.6},
			dtModels.DomainBodyComp:   {Trend: "STABLE", SlopePerDay: -0.1},
		},
	}

	cards := EvaluateTrajectoryCards(traj)
	found := false
	for _, c := range cards {
		if c.CardType == "CONCORDANT_DETERIORATION" {
			found = true
			if c.Urgency != "IMMEDIATE" {
				t.Errorf("expected IMMEDIATE urgency for 3-domain concordant, got %s", c.Urgency)
			}
		}
	}
	if !found {
		t.Error("expected CONCORDANT_DETERIORATION card")
	}
}

// ---------------------------------------------------------------------------
// TestTrajectoryCards_DivergenceAlert
// ---------------------------------------------------------------------------

func TestTrajectoryCards_DivergenceAlert(t *testing.T) {
	traj := &dtModels.DecomposedTrajectory{
		CompositeTrend:     "STABLE",
		HasDiscordantTrend: true,
		Divergences: []dtModels.DivergencePattern{
			{
				ImprovingDomain:   dtModels.DomainGlucose,
				DecliningDomain:   dtModels.DomainCardio,
				ImprovingSlope:    0.8,
				DecliningSlope:    -1.2,
				DivergenceRate:    2.0,
				ClinicalConcern:   "GLUCOSE improving while CARDIO declining",
				PossibleMechanism: "Consider SGLT2i for dual benefit",
			},
		},
		DomainSlopes: map[dtModels.MHRIDomain]dtModels.DomainSlope{
			dtModels.DomainGlucose:    {Trend: "IMPROVING"},
			dtModels.DomainCardio:     {Trend: "DECLINING"},
			dtModels.DomainBodyComp:   {Trend: "STABLE"},
			dtModels.DomainBehavioral: {Trend: "STABLE"},
		},
	}

	cards := EvaluateTrajectoryCards(traj)
	found := false
	for _, c := range cards {
		if c.CardType == "DOMAIN_DIVERGENCE" {
			found = true
			if c.Urgency != "URGENT" {
				t.Errorf("expected URGENT urgency for divergence, got %s", c.Urgency)
			}
		}
	}
	if !found {
		t.Error("expected DOMAIN_DIVERGENCE card")
	}
}

// ---------------------------------------------------------------------------
// TestTrajectoryCards_BehavioralLeadingIndicator
// ---------------------------------------------------------------------------

func TestTrajectoryCards_BehavioralLeadingIndicator(t *testing.T) {
	traj := &dtModels.DecomposedTrajectory{
		CompositeTrend: "DECLINING",
		LeadingIndicators: []dtModels.LeadingIndicator{
			{
				LeadingDomain:  dtModels.DomainBehavioral,
				LaggingDomains: []dtModels.MHRIDomain{dtModels.DomainGlucose, dtModels.DomainCardio},
				Interpretation: "Behavioral decline preceded clinical deterioration",
			},
		},
		DomainSlopes: map[dtModels.MHRIDomain]dtModels.DomainSlope{
			dtModels.DomainBehavioral: {Trend: "RAPID_DECLINING", SlopePerDay: -2.5},
			dtModels.DomainGlucose:    {Trend: "DECLINING", SlopePerDay: -0.5},
			dtModels.DomainCardio:     {Trend: "DECLINING", SlopePerDay: -0.4},
			dtModels.DomainBodyComp:   {Trend: "STABLE", SlopePerDay: 0.0},
		},
	}

	cards := EvaluateTrajectoryCards(traj)
	found := false
	for _, c := range cards {
		if c.CardType == "BEHAVIORAL_LEADING_INDICATOR" {
			found = true
			if c.Urgency != "URGENT" {
				t.Errorf("expected URGENT urgency, got %s", c.Urgency)
			}
		}
	}
	if !found {
		t.Error("expected BEHAVIORAL_LEADING_INDICATOR card")
	}
}

// ---------------------------------------------------------------------------
// TestTrajectoryCards_SingleDomainRapidDecline
// ---------------------------------------------------------------------------

func TestTrajectoryCards_SingleDomainRapidDecline(t *testing.T) {
	cardio := dtModels.DomainCardio
	traj := &dtModels.DecomposedTrajectory{
		CompositeTrend: "DECLINING",
		DomainSlopes: map[dtModels.MHRIDomain]dtModels.DomainSlope{
			dtModels.DomainCardio:     {Domain: dtModels.DomainCardio, Trend: "RAPID_DECLINING", SlopePerDay: -2.0, Confidence: "HIGH", R2: 0.85, StartScore: 70, EndScore: 42},
			dtModels.DomainGlucose:    {Trend: "STABLE"},
			dtModels.DomainBodyComp:   {Trend: "STABLE"},
			dtModels.DomainBehavioral: {Trend: "STABLE"},
		},
		DominantDriver: &cardio,
	}

	cards := EvaluateTrajectoryCards(traj)
	found := false
	for _, c := range cards {
		if c.CardType == "DOMAIN_RAPID_DECLINE" {
			found = true
		}
	}
	if !found {
		t.Error("expected DOMAIN_RAPID_DECLINE card")
	}
}

// ---------------------------------------------------------------------------
// TestTrajectoryCards_AllStable_NoUrgentCards
// ---------------------------------------------------------------------------

func TestTrajectoryCards_AllStable_NoUrgentCards(t *testing.T) {
	traj := &dtModels.DecomposedTrajectory{
		CompositeTrend: "STABLE",
		DomainSlopes: map[dtModels.MHRIDomain]dtModels.DomainSlope{
			dtModels.DomainGlucose:    {Trend: "STABLE"},
			dtModels.DomainCardio:     {Trend: "STABLE"},
			dtModels.DomainBodyComp:   {Trend: "STABLE"},
			dtModels.DomainBehavioral: {Trend: "STABLE"},
		},
	}

	cards := EvaluateTrajectoryCards(traj)
	for _, c := range cards {
		if c.Urgency == "IMMEDIATE" || c.Urgency == "URGENT" {
			t.Errorf("expected no urgent/immediate cards for all-stable, got %s (%s)", c.Urgency, c.CardType)
		}
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run:
```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go test ./internal/services/ -run "TestTrajectoryCards" -v 2>&1 | head -20
```

Expected: FAIL — `EvaluateTrajectoryCards` undefined.

- [ ] **Step 4: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/trajectory_card_rules_test.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/go.mod
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/go.sum
git commit -m "test(kb23): add failing tests for trajectory card rules

5 tests: concordant deterioration (IMMEDIATE), divergence alert (URGENT),
behavioral leading indicator, single domain rapid decline, all-stable
(no urgent cards). KB-26 models dependency added via replace directive."
```

---

## Task 8: Trajectory Card Rules — Implementation

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/trajectory_card_rules.go`

- [ ] **Step 1: Create the trajectory card rules engine**

```go
package services

import (
	"fmt"
	"strings"

	dtModels "kb-26-metabolic-digital-twin/internal/models"
)

// TrajectoryCard represents a decision card generated from domain trajectory analysis.
type TrajectoryCard struct {
	CardType  string   `json:"card_type"`
	Urgency   string   `json:"urgency"`
	Title     string   `json:"title"`
	Rationale string   `json:"rationale"`
	Actions   []string `json:"actions"`
	Domain    string   `json:"domain,omitempty"`
}

// EvaluateTrajectoryCards generates decision cards from decomposed domain trajectories.
func EvaluateTrajectoryCards(traj *dtModels.DecomposedTrajectory) []TrajectoryCard {
	if traj == nil {
		return nil
	}

	var cards []TrajectoryCard

	// 1. Concordant deterioration — highest priority.
	if traj.ConcordantDeterioration {
		var decliningDomains []string
		for domain, ds := range traj.DomainSlopes {
			if ds.SlopePerDay < -0.3 {
				decliningDomains = append(decliningDomains, string(domain))
			}
		}

		urgency := "URGENT"
		if traj.DomainsDeteriorating >= 3 {
			urgency = "IMMEDIATE"
		}

		cards = append(cards, TrajectoryCard{
			CardType: "CONCORDANT_DETERIORATION",
			Urgency:  urgency,
			Title:    fmt.Sprintf("Multi-Domain Deterioration — %d Domains Declining", traj.DomainsDeteriorating),
			Rationale: fmt.Sprintf("Simultaneous decline across %s. Concordant multi-domain "+
				"deterioration indicates systemic worsening — risk is multiplicative, not additive "+
				"(AHA CKM Framework). Composite MHRI slope: %.2f/day.",
				strings.Join(decliningDomains, ", "), traj.CompositeSlope),
			Actions: []string{
				"Comprehensive multi-domain medication review",
				"Identify root cause: medication non-adherence, intercurrent illness, lifestyle change",
				"Consider dual-benefit agents (SGLT2i for glucose + BP + renal)",
				"Schedule urgent clinical review within 1 week",
			},
		})
	}

	// 2. Domain divergence.
	for _, div := range traj.Divergences {
		cards = append(cards, TrajectoryCard{
			CardType: "DOMAIN_DIVERGENCE",
			Urgency:  "URGENT",
			Title: fmt.Sprintf("Discordant Trajectory — %s Improving, %s Declining",
				div.ImprovingDomain, div.DecliningDomain),
			Rationale: fmt.Sprintf("%s (slope +%.2f/day) while %s (slope %.2f/day). %s",
				div.ImprovingDomain, div.ImprovingSlope,
				div.DecliningDomain, div.DecliningSlope,
				div.ClinicalConcern),
			Actions: []string{
				div.PossibleMechanism,
				"Review whether improvement in one domain is masking deterioration in another",
				"Consider medication with cross-domain benefit",
			},
			Domain: string(div.DecliningDomain),
		})
	}

	// 3. Behavioral leading indicator.
	for _, lead := range traj.LeadingIndicators {
		laggingNames := make([]string, len(lead.LaggingDomains))
		for i, d := range lead.LaggingDomains {
			laggingNames[i] = string(d)
		}

		cards = append(cards, TrajectoryCard{
			CardType: "BEHAVIORAL_LEADING_INDICATOR",
			Urgency:  "URGENT",
			Title:    "Engagement Collapse Preceding Clinical Deterioration",
			Rationale: fmt.Sprintf("Behavioral domain declining before %s. %s "+
				"Clinical evidence shows behavioral disengagement predicts clinical "+
				"deterioration by 2-4 weeks. Intervene now to prevent further clinical decline.",
				strings.Join(laggingNames, " and "), lead.Interpretation),
			Actions: []string{
				"Clinical outreach — phone call preferred over digital notification",
				"Assess barriers to engagement: health anxiety, cost, side effects, access",
				"Do NOT default to app engagement nudge — this is a clinical signal, not a UX problem",
			},
		})
	}

	// 4. Single domain rapid decline (only if not already covered by concordant).
	if !traj.ConcordantDeterioration {
		for domain, ds := range traj.DomainSlopes {
			if ds.Trend == "RAPID_DECLINING" && ds.Confidence != "LOW" {
				cards = append(cards, TrajectoryCard{
					CardType: "DOMAIN_RAPID_DECLINE",
					Urgency:  "URGENT",
					Title:    fmt.Sprintf("%s Domain Rapid Decline", domain),
					Rationale: fmt.Sprintf("%s domain declining at %.2f/day (R²=%.2f, %s confidence). "+
						"Score dropped from %.0f to %.0f over the observation window.",
						domain, ds.SlopePerDay, ds.R2, ds.Confidence, ds.StartScore, ds.EndScore),
					Actions: []string{
						fmt.Sprintf("Review %s domain clinical data and recent changes", domain),
						"Investigate cause of rapid decline",
					},
					Domain: string(domain),
				})
			}
		}
	}

	// 5. Domain category crossing.
	for _, crossing := range traj.DomainCrossings {
		if crossing.Direction == "WORSENED" {
			cards = append(cards, TrajectoryCard{
				CardType: "DOMAIN_CATEGORY_CROSSING",
				Urgency:  "ROUTINE",
				Title: fmt.Sprintf("%s Domain: %s → %s",
					crossing.Domain, crossing.PrevCategory, crossing.CurrCategory),
				Rationale: fmt.Sprintf("%s domain crossed from %s to %s status. "+
					"This threshold crossing may indicate need for therapy adjustment.",
					crossing.Domain, crossing.PrevCategory, crossing.CurrCategory),
				Domain: string(crossing.Domain),
			})
		}
	}

	return cards
}
```

- [ ] **Step 2: Run tests to verify they pass**

Run:
```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go test ./internal/services/ -run "TestTrajectoryCards" -v
```

Expected: All 5 tests PASS.

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/trajectory_card_rules.go
git commit -m "feat(kb23): trajectory card rules — 5 card types from decomposed trajectory

CONCORDANT_DETERIORATION (IMMEDIATE for >=3 domains), DOMAIN_DIVERGENCE
(misdirected therapy), BEHAVIORAL_LEADING_INDICATOR (clinical outreach,
not app nudge), DOMAIN_RAPID_DECLINE (single domain, non-LOW confidence),
DOMAIN_CATEGORY_CROSSING (ROUTINE). All 5 tests pass."
```

---

## Task 9: Four-Pillar Evaluator Integration

**Files:**
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/four_pillar_evaluator.go`

- [ ] **Step 1: Add the import for KB-26 models**

At the top of `four_pillar_evaluator.go`, change the import block from:

```go
import "kb-23-decision-cards/internal/models"
```

to:

```go
import (
	"fmt"

	"kb-23-decision-cards/internal/models"
	dtModels "kb-26-metabolic-digital-twin/internal/models"
)
```

- [ ] **Step 2: Add DecomposedTrajectory field to FourPillarInput**

In the `FourPillarInput` struct, after the `InertiaReport` field, add:

```go
	DecomposedTrajectory *dtModels.DecomposedTrajectory `json:"decomposed_trajectory,omitempty"`
```

So the struct becomes:

```go
type FourPillarInput struct {
	PatientID            string                         `json:"patient_id"`
	DualDomainState      string                         `json:"dual_domain_state"`
	Medication           MedicationPillarInput          `json:"medication"`
	Monitoring           MonitoringPillarInput          `json:"monitoring"`
	Lifestyle            LifestylePillarInput           `json:"lifestyle"`
	Education            EducationPillarInput           `json:"education"`
	RenalGating          *models.PatientGatingReport    `json:"renal_gating,omitempty"`
	InertiaReport        *models.PatientInertiaReport   `json:"inertia_report,omitempty"`
	DecomposedTrajectory *dtModels.DecomposedTrajectory `json:"decomposed_trajectory,omitempty"`
}
```

- [ ] **Step 3: Add trajectory-based recommendations to evaluateMonitoringPillar**

In the `evaluateMonitoringPillar` function, before the final `p.Status = PillarOnTrack` return, add trajectory-based monitoring recommendations:

```go
	// Trajectory-based monitoring recommendations.
	if input.DecomposedTrajectory != nil {
		dt := input.DecomposedTrajectory
		if dt.ConcordantDeterioration {
			p.Status = PillarUrgentGap
			p.Reason = fmt.Sprintf("concordant deterioration: %d domains declining — increase monitoring frequency", dt.DomainsDeteriorating)
			p.Actions = append(p.Actions, "increase monitoring frequency across all domains")
			return p
		}
		if dt.HasDiscordantTrend {
			p.Status = PillarGap
			p.Reason = "discordant trajectory: domains moving in opposite directions"
			p.Actions = append(p.Actions, "investigate cross-domain medication effects")
			return p
		}
		for _, lead := range dt.LeadingIndicators {
			if lead.LeadingDomain == dtModels.DomainBehavioral {
				p.Status = PillarGap
				p.Reason = "behavioral leading indicator: engagement collapse detected"
				p.Actions = append(p.Actions, "clinical outreach recommended before clinical domains deteriorate further")
				return p
			}
		}
	}
```

This should be inserted after the stale eGFR check block and before the final ON_TRACK return. The full `evaluateMonitoringPillar` function becomes:

```go
func evaluateMonitoringPillar(input FourPillarInput) PillarResult {
	p := PillarResult{Pillar: "MONITORING"}

	if input.Monitoring.StaleEGFR != nil && input.Monitoring.StaleEGFR.IsStale {
		if input.Monitoring.StaleEGFR.Severity == "CRITICAL" {
			p.Status = PillarUrgentGap
			p.Reason = "eGFR critically overdue — data insufficient for safe prescribing"
			p.Actions = []string{"order urgent renal function panel"}
			return p
		}
		p.Status = PillarGap
		p.Reason = "eGFR measurement overdue"
		p.Actions = []string{"schedule renal function panel"}
		return p
	}

	// Trajectory-based monitoring recommendations.
	if input.DecomposedTrajectory != nil {
		dt := input.DecomposedTrajectory
		if dt.ConcordantDeterioration {
			p.Status = PillarUrgentGap
			p.Reason = fmt.Sprintf("concordant deterioration: %d domains declining — increase monitoring frequency", dt.DomainsDeteriorating)
			p.Actions = append(p.Actions, "increase monitoring frequency across all domains")
			return p
		}
		if dt.HasDiscordantTrend {
			p.Status = PillarGap
			p.Reason = "discordant trajectory: domains moving in opposite directions"
			p.Actions = append(p.Actions, "investigate cross-domain medication effects")
			return p
		}
		for _, lead := range dt.LeadingIndicators {
			if lead.LeadingDomain == dtModels.DomainBehavioral {
				p.Status = PillarGap
				p.Reason = "behavioral leading indicator: engagement collapse detected"
				p.Actions = append(p.Actions, "clinical outreach recommended before clinical domains deteriorate further")
				return p
			}
		}
	}

	p.Status = PillarOnTrack
	p.Reason = "monitoring up to date"
	return p
}
```

- [ ] **Step 4: Verify KB-23 compiles and existing tests still pass**

Run:
```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go build ./... && go test ./internal/services/ -v
```

Expected: Build succeeds. All existing tests PASS (existing tests don't set `DecomposedTrajectory`, so the nil check skips the new logic).

- [ ] **Step 5: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/four_pillar_evaluator.go
git commit -m "feat(kb23): integrate decomposed trajectory into four-pillar evaluator

Add DecomposedTrajectory field to FourPillarInput. Monitoring pillar now
checks: concordant deterioration -> URGENT_GAP, discordant trajectory ->
GAP, behavioral leading indicator -> GAP. Stale eGFR still takes priority.
Existing tests unaffected (nil trajectory skipped)."
```

---

## Task 10: Database Migration

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/migrations/006_domain_trajectory.sql`

- [ ] **Step 1: Create the migration SQL**

```sql
-- 006_domain_trajectory.sql
-- Domain trajectory history for trend-over-time analysis.

CREATE TABLE IF NOT EXISTS domain_trajectory_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id VARCHAR(100) NOT NULL,
    snapshot_date DATE NOT NULL,
    window_days INT,
    composite_slope DECIMAL(6,3),
    glucose_slope DECIMAL(6,3),
    cardio_slope DECIMAL(6,3),
    body_comp_slope DECIMAL(6,3),
    behavioral_slope DECIMAL(6,3),
    has_discordance BOOLEAN DEFAULT FALSE,
    dominant_driver VARCHAR(20),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(patient_id, snapshot_date)
);

CREATE INDEX idx_dth_patient ON domain_trajectory_history(patient_id, snapshot_date DESC);
```

- [ ] **Step 2: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/migrations/006_domain_trajectory.sql
git commit -m "migration(kb26): add domain_trajectory_history table

UUID PK, patient_id + snapshot_date unique, per-domain slope columns,
discordance flag, dominant driver. Index on (patient_id, snapshot_date DESC)
for efficient trend queries."
```

---

## Task 11: API Handler + Route Registration

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/domain_trajectory_handlers.go`
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/routes.go`

- [ ] **Step 1: Create the API handler**

```go
package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"kb-26-metabolic-digital-twin/internal/models"
	"kb-26-metabolic-digital-twin/internal/services"
)

// getDomainTrajectory computes and returns the decomposed MRI trajectory for a patient.
func (s *Server) getDomainTrajectory(c *gin.Context) {
	patientID := c.Param("patientId")
	if patientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "patientId is required"})
		return
	}

	// Fetch recent MRI scores for this patient (last 14 days by default).
	var mriScores []models.MRIScore
	cutoff := time.Now().AddDate(0, 0, -14)
	result := s.DB.Where("patient_id = ? AND computed_at >= ?", patientID, cutoff).
		Order("computed_at ASC").
		Find(&mriScores)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch MRI scores"})
		return
	}

	if len(mriScores) < 2 {
		c.JSON(http.StatusOK, gin.H{
			"patient_id": patientID,
			"status":     "INSUFFICIENT_DATA",
			"message":    "need at least 2 MRI scores for trajectory computation",
			"data_points": len(mriScores),
		})
		return
	}

	// Convert MRIScore records to DomainTrajectoryPoints.
	points := make([]models.DomainTrajectoryPoint, len(mriScores))
	for i, score := range mriScores {
		points[i] = models.DomainTrajectoryPoint{
			Timestamp:       score.ComputedAt,
			CompositeScore:  score.Score,
			GlucoseScore:    score.GlucoseDomain,
			CardioScore:     score.CardioDomain,
			BodyCompScore:   score.BodyCompDomain,
			BehavioralScore: score.BehavioralDomain,
		}
	}

	// Compute decomposed trajectory.
	trajectory := services.ComputeDecomposedTrajectory(patientID, points)

	// Persist snapshot to history table.
	history := models.DomainTrajectoryHistory{
		ID:              uuid.New().String(),
		PatientID:       patientID,
		SnapshotDate:    time.Now().Truncate(24 * time.Hour),
		WindowDays:      trajectory.WindowDays,
		CompositeSlope:  trajectory.CompositeSlope,
		GlucoseSlope:    trajectory.DomainSlopes[models.DomainGlucose].SlopePerDay,
		CardioSlope:     trajectory.DomainSlopes[models.DomainCardio].SlopePerDay,
		BodyCompSlope:   trajectory.DomainSlopes[models.DomainBodyComp].SlopePerDay,
		BehavioralSlope: trajectory.DomainSlopes[models.DomainBehavioral].SlopePerDay,
		HasDiscordance:  trajectory.HasDiscordantTrend,
		CreatedAt:       time.Now(),
	}
	if trajectory.DominantDriver != nil {
		history.DominantDriver = string(*trajectory.DominantDriver)
	}

	// Upsert — one snapshot per patient per day.
	s.DB.Where("patient_id = ? AND snapshot_date = ?", history.PatientID, history.SnapshotDate).
		Assign(history).
		FirstOrCreate(&history)

	c.JSON(http.StatusOK, trajectory)
}
```

- [ ] **Step 2: Register the route in routes.go**

In `routes.go`, inside the `v1` group block, after the existing MRI endpoints, add:

```go
		// Domain trajectory
		v1.GET("/mri/:patientId/domain-trajectory", s.getDomainTrajectory)
```

The MRI section in routes.go should now look like:

```go
		// MRI (Metabolic Risk Index)
		v1.GET("/mri/:patientId", s.getMRI)
		v1.GET("/mri/:patientId/history", s.getMRIHistory)
		v1.GET("/mri/:patientId/decomposition", s.getMRIDecomposition)
		v1.POST("/mri/simulate", s.simulateMRI)

		// Domain trajectory
		v1.GET("/mri/:patientId/domain-trajectory", s.getDomainTrajectory)
```

- [ ] **Step 3: Verify KB-26 compiles**

Run:
```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go build ./...
```

Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/domain_trajectory_handlers.go
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/routes.go
git commit -m "feat(kb26): API endpoint for domain-decomposed trajectory

GET /api/v1/kb26/mri/:patientId/domain-trajectory — fetches recent MRI
scores, computes decomposed trajectory, persists snapshot to history
table, returns full DecomposedTrajectory JSON."
```

---

## Task 12: Full Regression Test

- [ ] **Step 1: Run all KB-26 tests**

Run:
```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/services/ -v -count=1
```

Expected: All tests PASS, including 12 new tests (8 domain trajectory + 4 divergence) and all existing tests.

- [ ] **Step 2: Run all KB-23 tests**

Run:
```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go test ./internal/services/ -v -count=1
```

Expected: All tests PASS, including 5 new trajectory card tests and all existing tests.

- [ ] **Step 3: Build both services**

Run:
```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go build ./...
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go build ./...
```

Expected: Both services compile without errors.

- [ ] **Step 4: Final commit**

```bash
git add -A
git commit -m "feat: MHRI domain-decomposed trajectory — complete system

KB-26: per-domain OLS trajectories with R² confidence. Dominant driver
identification. Concordant deterioration (>=2 domains). Divergence
detection with clinical mechanism inference for 10 domain pairs.
Behavioral leading indicator. Domain category crossing alerts.
API endpoint + history table.

KB-23: 5 trajectory card types — CONCORDANT_DETERIORATION (IMMEDIATE
for >=3), DOMAIN_DIVERGENCE, BEHAVIORAL_LEADING_INDICATOR,
DOMAIN_RAPID_DECLINE, DOMAIN_CATEGORY_CROSSING. Four-pillar monitoring
pillar integration.

17 new tests. 12 files (9 create, 3 modify)."
```

---

## Plan Summary

| Task | Component | Files | Tests |
|------|-----------|-------|-------|
| 1 | Data Models | 1 create | compile check |
| 2 | Thresholds YAML | 1 create | — |
| 3 | Core Engine Tests | 1 create | 8 failing |
| 4 | Core Engine Impl | 1 create | 8 passing |
| 5 | Divergence Tests | 1 create | 4 failing |
| 6 | Divergence Impl | 1 create | 4 passing |
| 7 | Card Tests + go.mod | 1 create + 1 modify | 5 failing |
| 8 | Card Impl | 1 create | 5 passing |
| 9 | Four-Pillar Integration | 1 modify | existing pass |
| 10 | Migration SQL | 1 create | — |
| 11 | API Handler + Route | 1 create + 1 modify | compile check |
| 12 | Full Regression | — | all 17 pass |

**Total: 12 tasks, 9 created files, 3 modified files, 18 new tests (incl. integration test added in final review)**

---

## Known Gap (Phase 1 Follow-up)

**Runtime wiring of trajectory cards and four-pillar trajectory input is not connected to any HTTP handler.**

- `EvaluateTrajectoryCards()` is reachable only from tests.
- `FourPillarInput.DecomposedTrajectory` has no populator — callers constructing `FourPillarInput` today do not fetch the trajectory.
- The KB-26 endpoint `/api/v1/kb26/mri/:patientId/domain-trajectory` works standalone, so dashboards and shadow-mode analytics can consume it directly.

This matches the same gap in the parallel Masked HTN feature:
- `EvaluateMaskedHTNCards()` is also only called from tests.
- `FourPillarInput.BPContext` also has no populator in KB-23 handlers.

**Both features share the same architectural gap by design** — they ship as computable components and defer the "card orchestration" layer to a Phase 1 task that wires BOTH features into a unified card-generation pipeline (likely via `CompositeCardService`). Merging trajectory-only wiring now would create inconsistency with HTN.

**Phase 1 task (out of scope for this plan):** Create a KB-23 handler that (a) fetches both `DecomposedTrajectory` from KB-26 and `BPContextClassification` from KB-26, (b) populates `FourPillarInput` with both, (c) calls `EvaluateTrajectoryCards` + `EvaluateMaskedHTNCards`, and (d) feeds the combined output into the existing `CompositeCardService` 72-hour aggregation window.
