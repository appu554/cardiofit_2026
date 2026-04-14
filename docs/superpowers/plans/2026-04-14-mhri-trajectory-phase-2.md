# MHRI Domain-Decomposed Trajectory Phase 2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Production-harden the MHRI domain-decomposed trajectory system shipped in Phase 0+1 by adding observability, config-driven thresholds, fixing the OLS R² numerical stability bug, suppressing seasonal false positives, and publishing trajectory events to Kafka for Module 13 Flink integration.

**Architecture:** Five independent slices building on the Phase 0+1 foundation. The config-driven threshold refactor (Item 2) converts `ComputeDecomposedTrajectory` from a free function to a `TrajectoryEngine` struct method, which is the sequencing bottleneck for the other items. Items 1, 3, 4, 5 depend on this refactor and land in any order after it.

**Tech Stack:** Go 1.25 (KB-26), Go 1.25 (KB-23), Prometheus, Kafka (segmentio/kafka-go), YAML config, stdlib testing

**Spec:** `docs/superpowers/specs/2026-04-14-mhri-trajectory-phase-2-design.md`

---

## File Structure

### KB-26 (Metabolic Digital Twin)
| Action | File | Responsibility |
|--------|------|---------------|
| Create | `kb-26-metabolic-digital-twin/internal/metrics/trajectory_metrics.go` | Prometheus metrics for trajectory engine |
| Create | `kb-26-metabolic-digital-twin/internal/config/trajectory_config.go` | TrajectoryThresholds struct + YAML loader |
| Modify | `kb-26-metabolic-digital-twin/internal/config/config.go` | Wire trajectory thresholds into root config |
| Create | `kb-26-metabolic-digital-twin/internal/services/trajectory_engine.go` | Engine struct holding config; Compute method |
| Delete | `kb-26-metabolic-digital-twin/internal/services/mri_domain_trajectory.go` | Contents merged into trajectory_engine.go |
| Modify | `kb-26-metabolic-digital-twin/internal/services/mri_domain_trajectory_test.go` | Tests target the new struct API |
| Modify | `kb-26-metabolic-digital-twin/internal/services/egfr_trajectory.go` | Two-pass ssTot numerical stability fix |
| Modify | `kb-26-metabolic-digital-twin/internal/services/egfr_trajectory_test.go` | Add ClusteredScores regression test |
| Create | `kb-26-metabolic-digital-twin/internal/services/trajectory_publisher.go` | Kafka producer for DomainTrajectoryComputed events |
| Create | `kb-26-metabolic-digital-twin/internal/services/trajectory_publisher_test.go` | Publisher tests with mock writer |
| Modify | `kb-26-metabolic-digital-twin/internal/api/domain_trajectory_handlers.go` | Use engine struct, increment metrics, publish event |
| Modify | `kb-26-metabolic-digital-twin/internal/api/server.go` | Init engine, metrics, publisher |
| Modify | `kb-26-metabolic-digital-twin/pkg/trajectory/models.go` | Compute wraps default engine for backward compat |

### KB-23 (Decision Cards)
| Action | File | Responsibility |
|--------|------|---------------|
| Create | `kb-23-decision-cards/internal/metrics/trajectory_card_metrics.go` | Card eval + KB-26 fetch metrics |
| Create | `kb-23-decision-cards/internal/services/seasonal_context.go` | Seasonal calendar checker |
| Create | `kb-23-decision-cards/internal/services/seasonal_context_test.go` | Seasonal context unit tests |
| Modify | `kb-23-decision-cards/internal/services/trajectory_card_rules.go` | Optional SeasonalContext for suppression/downgrade |
| Modify | `kb-23-decision-cards/internal/services/trajectory_card_rules_test.go` | Seasonal suppression tests |
| Modify | `kb-23-decision-cards/internal/services/kb26_trajectory_client.go` | Fetch metric instrumentation |
| Modify | `kb-23-decision-cards/internal/api/composite_handlers.go` | Wire seasonal context + metrics |
| Modify | `kb-23-decision-cards/internal/api/server.go` | Init metrics + seasonal context |

### Market Configs
| Action | File | Responsibility |
|--------|------|---------------|
| Create | `market-configs/india/seasonal_calendar.yaml` | India festival/season windows |
| Create | `market-configs/australia/seasonal_calendar.yaml` | Empty seed for AU contract |

### Observability
| Action | File | Responsibility |
|--------|------|---------------|
| Create | `observability/dashboards/mhri_trajectory.json` | Grafana dashboard with 6 panels |

### Stream Services Docs
| Action | File | Responsibility |
|--------|------|---------------|
| Create | `backend/stream-services/docs/kb26_domain_trajectory_consumer.md` | Module 13 Flink consumer contract |

---

## Sequencing

Item 2 (config refactor) is the bottleneck — it changes the engine API surface that Items 1, 3, 5 instrument or extend. Recommended order:

1. **Slice P2.2** (Tasks 1-6): Config refactor — engine becomes a struct
2. **Slice P2.3** (Tasks 7-9): Numerical stability — applied to the new engine + sibling egfr file
3. **Slice P2.1** (Tasks 10-13): KB-26 observability — instruments the engine
4. **Slice P2.5** (Tasks 14-17): Kafka publisher — wires into the engine
5. **Slice P2.4** (Tasks 18-22): Seasonal adjustment — extends KB-23 card rules
6. **Slice P2.6** (Tasks 23-26): KB-23 observability + Grafana dashboard

---

## Slice P2.2: Config Refactor (Tasks 1-6)

### Task 1: Define TrajectoryThresholds config struct

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/config/trajectory_config.go`

- [ ] **Step 1: Create the config file**

```go
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"kb-26-metabolic-digital-twin/internal/models"
)

// TrajectoryThresholds holds all tunable thresholds for the trajectory engine.
type TrajectoryThresholds struct {
	Trend              TrendThresholds              `yaml:"trend_thresholds"`
	Divergence         DivergenceThresholds         `yaml:"divergence"`
	LeadingIndicator   LeadingIndicatorThresholds   `yaml:"leading_indicator"`
	Concordant         ConcordantThresholds         `yaml:"concordant"`
	Driver             DriverThresholds             `yaml:"driver"`
	RSquared           R2Thresholds                 `yaml:"r_squared"`
	CategoryBoundaries CategoryBoundaries           `yaml:"category_boundaries"`
}

type TrendThresholds struct {
	RapidImproving float64 `yaml:"rapid_improving"`
	Improving      float64 `yaml:"improving"`
	Declining      float64 `yaml:"declining"`
	RapidDeclining float64 `yaml:"rapid_declining"`
}

type DivergenceThresholds struct {
	MinDivergenceRate float64 `yaml:"min_divergence_rate"`
	MinImprovingSlope float64 `yaml:"min_improving_slope"`
	MinDecliningSlope float64 `yaml:"min_declining_slope"`
}

type LeadingIndicatorThresholds struct {
	MinDataPoints             int     `yaml:"min_data_points"`
	MinBehavioralDeclineSlope float64 `yaml:"min_behavioral_decline_slope"`
}

type ConcordantThresholds struct {
	MinDomainsDeclining int     `yaml:"min_domains_declining"`
	MinSlopePerDomain   float64 `yaml:"min_slope_per_domain"`
}

type DriverThresholds struct {
	MinContributionPct float64                       `yaml:"min_contribution_pct"`
	WeightMap          map[models.MHRIDomain]float64 `yaml:"weight_map"`
}

type R2Thresholds struct {
	High     float64 `yaml:"high"`
	Moderate float64 `yaml:"moderate"`
}

type CategoryBoundaries struct {
	Optimal  float64 `yaml:"optimal"`
	Mild     float64 `yaml:"mild"`
	Moderate float64 `yaml:"moderate"`
}

// DefaultTrajectoryThresholds returns the canonical defaults matching the
// values that were previously hardcoded in mri_domain_trajectory.go.
func DefaultTrajectoryThresholds() TrajectoryThresholds {
	return TrajectoryThresholds{
		Trend: TrendThresholds{
			RapidImproving: 1.0,
			Improving:      0.3,
			Declining:      -0.3,
			RapidDeclining: -1.0,
		},
		Divergence: DivergenceThresholds{
			MinDivergenceRate: 0.5,
			MinImprovingSlope: 0.3,
			MinDecliningSlope: -0.3,
		},
		LeadingIndicator: LeadingIndicatorThresholds{
			MinDataPoints:             5,
			MinBehavioralDeclineSlope: -0.5,
		},
		Concordant: ConcordantThresholds{
			MinDomainsDeclining: 2,
			MinSlopePerDomain:   -0.3,
		},
		Driver: DriverThresholds{
			MinContributionPct: 40.0,
			WeightMap: map[models.MHRIDomain]float64{
				models.DomainGlucose:    0.35,
				models.DomainCardio:     0.25,
				models.DomainBodyComp:   0.25,
				models.DomainBehavioral: 0.15,
			},
		},
		RSquared: R2Thresholds{
			High:     0.5,
			Moderate: 0.25,
		},
		CategoryBoundaries: CategoryBoundaries{
			Optimal:  70.0,
			Mild:     55.0,
			Moderate: 40.0,
		},
	}
}

// LoadTrajectoryThresholds parses a YAML file and returns thresholds.
// Returns DefaultTrajectoryThresholds and a warning error if the file
// is missing — startup should not fail on missing config.
func LoadTrajectoryThresholds(path string) (TrajectoryThresholds, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultTrajectoryThresholds(), nil
		}
		return TrajectoryThresholds{}, fmt.Errorf("read trajectory thresholds: %w", err)
	}

	var t TrajectoryThresholds
	if err := yaml.Unmarshal(data, &t); err != nil {
		return TrajectoryThresholds{}, fmt.Errorf("parse trajectory thresholds: %w", err)
	}

	return t, nil
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go build ./internal/config/
```

Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/config/trajectory_config.go
git commit -m "feat(kb26): add TrajectoryThresholds config struct + YAML loader

Defines the threshold types previously hardcoded as Go constants. Provides
DefaultTrajectoryThresholds() returning canonical Phase 0 values for tests
and fallback. LoadTrajectoryThresholds() parses YAML and falls back to
defaults on missing file (startup must not fail on missing config)."
```

---

### Task 2: Test config loader against existing YAML

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/config/trajectory_config_test.go`

- [ ] **Step 1: Write the test**

```go
package config

import (
	"os"
	"path/filepath"
	"testing"

	"kb-26-metabolic-digital-twin/internal/models"
)

func TestLoadTrajectoryThresholds_DefaultsOnMissingFile(t *testing.T) {
	tmp := t.TempDir()
	missing := filepath.Join(tmp, "nope.yaml")

	got, err := LoadTrajectoryThresholds(missing)
	if err != nil {
		t.Fatalf("expected nil error on missing file, got %v", err)
	}

	defaults := DefaultTrajectoryThresholds()
	if got.Trend.RapidImproving != defaults.Trend.RapidImproving {
		t.Errorf("expected default RapidImproving %v, got %v", defaults.Trend.RapidImproving, got.Trend.RapidImproving)
	}
	if got.Driver.WeightMap[models.DomainGlucose] != 0.35 {
		t.Errorf("expected default glucose weight 0.35, got %v", got.Driver.WeightMap[models.DomainGlucose])
	}
}

func TestLoadTrajectoryThresholds_ParsesValidYAML(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "thresholds.yaml")

	yaml := `
trend_thresholds:
  rapid_improving: 1.5
  improving: 0.4
  declining: -0.4
  rapid_declining: -1.2

divergence:
  min_divergence_rate: 0.6
  min_improving_slope: 0.4
  min_declining_slope: -0.4

leading_indicator:
  min_data_points: 6
  min_behavioral_decline_slope: -0.6

concordant:
  min_domains_declining: 3
  min_slope_per_domain: -0.4

driver:
  min_contribution_pct: 50.0
  weight_map:
    GLUCOSE: 0.40
    CARDIO: 0.30
    BODY_COMP: 0.20
    BEHAVIORAL: 0.10

r_squared:
  high: 0.6
  moderate: 0.3

category_boundaries:
  optimal: 75
  mild: 60
  moderate: 45
`
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatalf("write test yaml: %v", err)
	}

	got, err := LoadTrajectoryThresholds(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if got.Trend.RapidImproving != 1.5 {
		t.Errorf("expected RapidImproving 1.5, got %v", got.Trend.RapidImproving)
	}
	if got.Driver.WeightMap[models.DomainGlucose] != 0.40 {
		t.Errorf("expected glucose weight 0.40, got %v", got.Driver.WeightMap[models.DomainGlucose])
	}
	if got.CategoryBoundaries.Optimal != 75 {
		t.Errorf("expected optimal boundary 75, got %v", got.CategoryBoundaries.Optimal)
	}
}
```

- [ ] **Step 2: Run tests**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/config/ -run "TestLoadTrajectoryThresholds" -v
```

Expected: 2/2 PASS.

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/config/trajectory_config_test.go
git commit -m "test(kb26): config loader tests for TrajectoryThresholds

Verifies fallback to defaults on missing file and YAML parsing for
all threshold groups including the WeightMap map[MHRIDomain]float64."
```

---

### Task 3: Refactor engine to TrajectoryEngine struct

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/trajectory_engine.go`
- Delete: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/mri_domain_trajectory.go`

- [ ] **Step 1: Create the new engine file**

This file replaces `mri_domain_trajectory.go` entirely. The free function `ComputeDecomposedTrajectory` becomes the `Compute` method on `TrajectoryEngine`. All hardcoded constants are replaced with reads from the engine's `thresholds` field.

```go
package services

import (
	"math"
	"sort"
	"time"

	"kb-26-metabolic-digital-twin/internal/config"
	"kb-26-metabolic-digital-twin/internal/models"
)

// TrajectoryEngine computes per-domain MHRI trajectories. Constructed with
// a TrajectoryThresholds value; all classification logic reads from this
// config so callers can override per market.
type TrajectoryEngine struct {
	thresholds config.TrajectoryThresholds
}

// NewTrajectoryEngine constructs an engine with the given thresholds.
// Pass config.DefaultTrajectoryThresholds() for the canonical Phase 0 values.
func NewTrajectoryEngine(thresholds config.TrajectoryThresholds) *TrajectoryEngine {
	return &TrajectoryEngine{thresholds: thresholds}
}

// Compute computes per-domain OLS trajectories and derived analytics.
func (e *TrajectoryEngine) Compute(patientID string, points []models.DomainTrajectoryPoint) models.DecomposedTrajectory {
	result := models.DecomposedTrajectory{
		PatientID:    patientID,
		DataPoints:   len(points),
		ComputedAt:   time.Now(),
		DomainSlopes: make(map[models.MHRIDomain]models.DomainSlope),
	}

	if len(points) < 2 {
		result.CompositeTrend = models.TrendInsufficient
		for _, d := range models.AllMHRIDomains {
			result.DomainSlopes[d] = models.DomainSlope{Domain: d, Trend: models.TrendInsufficient}
		}
		return result
	}

	// Sort by timestamp.
	sorted := make([]models.DomainTrajectoryPoint, len(points))
	copy(sorted, points)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.Before(sorted[j].Timestamp)
	})

	first, last := sorted[0], sorted[len(sorted)-1]
	result.WindowDays = int(math.Round(last.Timestamp.Sub(first.Timestamp).Hours() / 24))

	// Composite trajectory.
	compositeScores := extractScores(sorted, func(p models.DomainTrajectoryPoint) float64 { return p.CompositeScore })
	compSlope, _ := e.computeOLSWithR2(sorted, compositeScores)
	result.CompositeSlope = roundTo3(compSlope)
	result.CompositeTrend = e.classifyTrend(compSlope)
	result.CompositeStartScore = first.CompositeScore
	result.CompositeEndScore = last.CompositeScore

	// Per-domain trajectories.
	domainExtractors := map[models.MHRIDomain]func(models.DomainTrajectoryPoint) float64{
		models.DomainGlucose:    func(p models.DomainTrajectoryPoint) float64 { return p.GlucoseScore },
		models.DomainCardio:     func(p models.DomainTrajectoryPoint) float64 { return p.CardioScore },
		models.DomainBodyComp:   func(p models.DomainTrajectoryPoint) float64 { return p.BodyCompScore },
		models.DomainBehavioral: func(p models.DomainTrajectoryPoint) float64 { return p.BehavioralScore },
	}

	decliningCount := 0
	var maxWeightedDecline float64
	var dominantDriver *models.MHRIDomain

	for _, domain := range models.AllMHRIDomains {
		extractor := domainExtractors[domain]
		scores := extractScores(sorted, extractor)
		slope, r2 := e.computeOLSWithR2(sorted, scores)

		ds := models.DomainSlope{
			Domain:      domain,
			SlopePerDay: roundTo3(slope),
			Trend:       e.classifyTrend(slope),
			StartScore:  scores[0],
			EndScore:    scores[len(scores)-1],
			DeltaScore:  roundTo1(scores[len(scores)-1] - scores[0]),
			R2:          roundTo3(r2),
			Confidence:  e.classifyConfidence(r2),
		}
		result.DomainSlopes[domain] = ds

		if slope < e.thresholds.Concordant.MinSlopePerDomain {
			decliningCount++
		}

		weight := e.thresholds.Driver.WeightMap[domain]
		weightedDecline := math.Abs(slope) * weight
		if slope < e.thresholds.Trend.Declining && weightedDecline > maxWeightedDecline {
			maxWeightedDecline = weightedDecline
			d := domain
			dominantDriver = &d
		}
	}

	result.DomainsDeteriorating = decliningCount
	result.ConcordantDeterioration = decliningCount >= e.thresholds.Concordant.MinDomainsDeclining

	// Dominant driver calculation.
	if dominantDriver != nil && result.CompositeSlope < 0 {
		result.DominantDriver = dominantDriver
		totalWeightedDecline := 0.0
		for d, ds := range result.DomainSlopes {
			if ds.SlopePerDay < e.thresholds.Trend.Declining {
				totalWeightedDecline += math.Abs(ds.SlopePerDay) * e.thresholds.Driver.WeightMap[d]
			}
		}
		if totalWeightedDecline > 0 {
			result.DriverContribution = roundTo1((maxWeightedDecline / totalWeightedDecline) * 100)
		}
	}

	// Divergence (uses engine thresholds via package-level helper updated in the same task).
	result.Divergences = e.detectDivergences(result.DomainSlopes)
	result.HasDiscordantTrend = len(result.Divergences) > 0

	// Domain category crossings.
	result.DomainCrossings = e.detectDomainCrossings(sorted, domainExtractors)

	// Behavioral leading indicator.
	result.LeadingIndicators = e.detectLeadingIndicators(sorted, result.DomainSlopes)

	return result
}

// computeOLSWithR2 runs OLS linear regression returning slope (per day) and R².
// Uses the numerically stable two-pass form for ssTot. (Item 3 fix.)
func (e *TrajectoryEngine) computeOLSWithR2(points []models.DomainTrajectoryPoint, scores []float64) (float64, float64) {
	if len(points) < 2 {
		return 0, 0
	}

	baseTime := points[0].Timestamp
	n := float64(len(points))
	var sumX, sumY, sumXY, sumX2 float64

	for i, pt := range points {
		x := pt.Timestamp.Sub(baseTime).Hours() / 24.0
		y := scores[i]
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	denom := n*sumX2 - sumX*sumX
	if math.Abs(denom) < 1e-10 {
		return 0, 0
	}

	slope := (n*sumXY - sumX*sumY) / denom
	intercept := (sumY - slope*sumX) / n

	// Two-pass ssTot (numerically stable — Item 3 fix).
	meanY := sumY / n
	ssTot := 0.0
	ssRes := 0.0
	for i, pt := range points {
		x := pt.Timestamp.Sub(baseTime).Hours() / 24.0
		predicted := intercept + slope*x
		residual := scores[i] - predicted
		ssRes += residual * residual

		delta := scores[i] - meanY
		ssTot += delta * delta
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

func (e *TrajectoryEngine) classifyTrend(slopePerDay float64) string {
	switch {
	case slopePerDay > e.thresholds.Trend.RapidImproving:
		return models.TrendRapidImproving
	case slopePerDay > e.thresholds.Trend.Improving:
		return models.TrendImproving
	case slopePerDay >= e.thresholds.Trend.Declining:
		return models.TrendStable
	case slopePerDay >= e.thresholds.Trend.RapidDeclining:
		return models.TrendDeclining
	default:
		return models.TrendRapidDeclining
	}
}

func (e *TrajectoryEngine) classifyConfidence(r2 float64) string {
	if r2 >= e.thresholds.RSquared.High {
		return models.ConfidenceHigh
	}
	if r2 >= e.thresholds.RSquared.Moderate {
		return models.ConfidenceModerate
	}
	return models.ConfidenceLow
}

func (e *TrajectoryEngine) categorizeDomainScore(score float64) string {
	if score >= e.thresholds.CategoryBoundaries.Optimal {
		return "OPTIMAL"
	}
	if score >= e.thresholds.CategoryBoundaries.Mild {
		return "MILD"
	}
	if score >= e.thresholds.CategoryBoundaries.Moderate {
		return "MODERATE"
	}
	return "HIGH"
}

func (e *TrajectoryEngine) detectDomainCrossings(points []models.DomainTrajectoryPoint, extractors map[models.MHRIDomain]func(models.DomainTrajectoryPoint) float64) []models.DomainCategoryCrossing {
	if len(points) < 2 {
		return nil
	}

	first := points[0]
	last := points[len(points)-1]
	var crossings []models.DomainCategoryCrossing

	for _, domain := range models.AllMHRIDomains {
		extractor, ok := extractors[domain]
		if !ok {
			continue
		}
		startScore := extractor(first)
		endScore := extractor(last)
		startCat := e.categorizeDomainScore(startScore)
		endCat := e.categorizeDomainScore(endScore)

		if startCat != endCat {
			direction := models.DirectionImproved
			if endScore < startScore {
				direction = models.DirectionWorsened
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

func (e *TrajectoryEngine) detectLeadingIndicators(points []models.DomainTrajectoryPoint, slopes map[models.MHRIDomain]models.DomainSlope) []models.LeadingIndicator {
	if len(points) < e.thresholds.LeadingIndicator.MinDataPoints {
		return nil
	}

	behSlope := slopes[models.DomainBehavioral]
	if behSlope.SlopePerDay >= e.thresholds.LeadingIndicator.MinBehavioralDeclineSlope {
		return nil
	}

	var lagging []models.MHRIDomain
	for _, domain := range []models.MHRIDomain{models.DomainGlucose, models.DomainCardio} {
		ds := slopes[domain]
		if ds.SlopePerDay < e.thresholds.Trend.Declining {
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
		Confidence:     models.ConfidenceModerate,
		Interpretation: "Behavioral domain decline preceded clinical domain deterioration — engagement collapse may be driving worsening outcomes",
	}}
}

func extractScores(points []models.DomainTrajectoryPoint, extractor func(models.DomainTrajectoryPoint) float64) []float64 {
	scores := make([]float64, len(points))
	for i, p := range points {
		scores[i] = extractor(p)
	}
	return scores
}

func roundTo3(v float64) float64 { return math.Round(v*1000) / 1000 }
func roundTo1(v float64) float64 { return math.Round(v*10) / 10 }
```

- [ ] **Step 2: Delete the old file**

```bash
rm backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/mri_domain_trajectory.go
```

- [ ] **Step 3: Verify build (will fail until divergence + tests are updated)**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go build ./internal/services/ 2>&1 | head -10
```

Expected: build errors referencing `detectDivergences` (still package-level free function) and `ComputeDecomposedTrajectory` (no longer exists). These get fixed in Tasks 4 and 5.

- [ ] **Step 4: DO NOT commit yet** — wait until Tasks 4 and 5 land so the package compiles.

---

### Task 4: Move detectDivergences onto the engine

**Files:**
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/domain_divergence.go`

- [ ] **Step 1: Convert detectDivergences to a method on TrajectoryEngine**

Replace the existing `detectDivergences` free function and `inferDivergenceMechanism` free function. The mechanism inference stays a free function (no config dependency). Only `detectDivergences` becomes a method:

```go
package services

import (
	"fmt"
	"math"

	"kb-26-metabolic-digital-twin/internal/models"
)

// detectDivergences finds pairs of domains moving in opposite directions.
// Method on TrajectoryEngine so it can read divergence thresholds from config.
func (e *TrajectoryEngine) detectDivergences(slopes map[models.MHRIDomain]models.DomainSlope) []models.DivergencePattern {
	var divergences []models.DivergencePattern
	domains := models.AllMHRIDomains

	improvingThreshold := e.thresholds.Divergence.MinImprovingSlope
	decliningThreshold := e.thresholds.Divergence.MinDecliningSlope
	minRate := e.thresholds.Divergence.MinDivergenceRate

	for i := 0; i < len(domains); i++ {
		for j := i + 1; j < len(domains); j++ {
			slopeA := slopes[domains[i]]
			slopeB := slopes[domains[j]]

			var improving, declining models.DomainSlope
			if slopeA.SlopePerDay > improvingThreshold && slopeB.SlopePerDay < decliningThreshold {
				improving = slopeA
				improving.Domain = domains[i]
				declining = slopeB
				declining.Domain = domains[j]
			} else if slopeB.SlopePerDay > improvingThreshold && slopeA.SlopePerDay < decliningThreshold {
				improving = slopeB
				improving.Domain = domains[j]
				declining = slopeA
				declining.Domain = domains[i]
			} else {
				continue
			}

			divergenceRate := math.Abs(improving.SlopePerDay) + math.Abs(declining.SlopePerDay)
			if divergenceRate < minRate {
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

// inferDivergenceMechanism remains a free function — no config dependency.
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

- [ ] **Step 2: DO NOT commit yet** — Task 5 fixes the test files so the package compiles.

---

### Task 5: Update tests to use the engine struct

**Files:**
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/mri_domain_trajectory_test.go`
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/domain_divergence_test.go`

- [ ] **Step 1: Add a test helper at the top of mri_domain_trajectory_test.go**

```go
package services

import (
	"testing"
	"time"

	"kb-26-metabolic-digital-twin/internal/config"
	"kb-26-metabolic-digital-twin/internal/models"
)

// defaultEngine returns a trajectory engine with canonical Phase 0 thresholds.
// All tests in this file use it unless they specifically test config overrides.
func defaultEngine() *TrajectoryEngine {
	return NewTrajectoryEngine(config.DefaultTrajectoryThresholds())
}
```

- [ ] **Step 2: Replace every call to ComputeDecomposedTrajectory with engine.Compute**

Find every occurrence of `ComputeDecomposedTrajectory(` in the test file and replace with `defaultEngine().Compute(`.

For example:
```go
result := ComputeDecomposedTrajectory("PAT-001", points)
```
becomes:
```go
result := defaultEngine().Compute("PAT-001", points)
```

Apply this replacement to all 8 tests in the file.

- [ ] **Step 3: Update domain_divergence_test.go**

The existing tests call the package-level `detectDivergences(slopes)`. After the refactor this is now `(*TrajectoryEngine).detectDivergences(slopes)`. Update the calls:

```go
divergences := detectDivergences(slopes)
```
becomes:
```go
divergences := defaultEngine().detectDivergences(slopes)
```

Apply to all 4 tests.

- [ ] **Step 4: Run tests**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/services/ -run "TestDomainTrajectory|TestDomainCategory|TestDetectDivergence|TestDivergence_" -v
```

Expected: All 12 tests PASS.

- [ ] **Step 5: Commit Tasks 3-5 together**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/trajectory_engine.go backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/mri_domain_trajectory.go backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/mri_domain_trajectory_test.go backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/domain_divergence.go backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/domain_divergence_test.go
git commit -m "refactor(kb26): trajectory engine becomes config-driven struct

ComputeDecomposedTrajectory free function → TrajectoryEngine.Compute method.
All hardcoded thresholds replaced with reads from config.TrajectoryThresholds.
detectDivergences also becomes a method so it reads divergence thresholds
from the same config. Existing 12 tests updated to construct an engine
via defaultEngine() helper using DefaultTrajectoryThresholds(). Includes
the numerical stability fix (two-pass ssTot) — see Item 3 docs."
```

---

### Task 6: Update API handler and pkg/trajectory facade

**Files:**
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/domain_trajectory_handlers.go`
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/server.go`
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/pkg/trajectory/models.go`

- [ ] **Step 1: Update Server struct to hold a TrajectoryEngine**

In `server.go`, add to the Server struct fields:
```go
trajectoryEngine *services.TrajectoryEngine
```

In `InitServices()` (or wherever the server initializes services), add:
```go
trajectoryThresholds, err := config.LoadTrajectoryThresholds(s.cfg.TrajectoryThresholdsPath)
if err != nil {
	s.logger.Warn("failed to load trajectory thresholds, using defaults", zap.Error(err))
	trajectoryThresholds = config.DefaultTrajectoryThresholds()
}
s.trajectoryEngine = services.NewTrajectoryEngine(trajectoryThresholds)
```

The path comes from `cfg.TrajectoryThresholdsPath` — add this field to the root Config struct in `config.go` with default `"market-configs/shared/domain_trajectory_thresholds.yaml"`.

- [ ] **Step 2: Update handler to use the engine**

In `domain_trajectory_handlers.go`, find this line:
```go
trajectory := services.ComputeDecomposedTrajectory(patientID.String(), points)
```

Replace with:
```go
trajectory := s.trajectoryEngine.Compute(patientID.String(), points)
```

- [ ] **Step 3: Update pkg/trajectory facade**

In `pkg/trajectory/models.go`, the existing `Compute` var was defined as:
```go
var Compute = services.ComputeDecomposedTrajectory
```

This no longer compiles. Replace with a wrapper that constructs a default engine:
```go
import (
	"kb-26-metabolic-digital-twin/internal/config"
	"kb-26-metabolic-digital-twin/internal/services"
)

// Compute computes the domain-decomposed trajectory using default thresholds.
// Cross-module consumers (e.g., KB-23 integration tests) call this for the
// canonical Phase 0 behavior. For per-market customization, use a
// TrajectoryEngine constructed with custom thresholds inside KB-26.
func Compute(patientID string, points []DomainTrajectoryPoint) DecomposedTrajectory {
	engine := services.NewTrajectoryEngine(config.DefaultTrajectoryThresholds())
	return engine.Compute(patientID, points)
}
```

(Note: changing `Compute` from `var` to `func` is a source-compatible change for callers because both are called the same way: `trajectory.Compute(id, points)`.)

- [ ] **Step 4: Build and test**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go build ./... && go test ./internal/services/ -count=1 2>&1 | tail -5
```

Then KB-23:
```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go build ./... && go test ./internal/services/ -run "TestIntegration_ComputeAndEvaluate" -v
```

Expected: Both build clean, all tests pass including the cross-module integration test.

- [ ] **Step 5: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/domain_trajectory_handlers.go backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/server.go backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/config/config.go backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/pkg/trajectory/models.go
git commit -m "feat(kb26): wire TrajectoryEngine into Server + facade

Server initializes engine with thresholds loaded from the configured YAML
path (falls back to defaults on missing file). Handler calls
s.trajectoryEngine.Compute(...). pkg/trajectory.Compute becomes a thin
wrapper constructing a default engine for cross-module consumers."
```

---

## Slice P2.3: Numerical Stability (Tasks 7-9)

The fix to `mri_domain_trajectory.go` was already included in Task 3 (the new `trajectory_engine.go` file). This slice covers the sibling `egfr_trajectory.go` file plus a regression test.

### Task 7: Write failing test exposing the egfr stability bug

**Files:**
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/egfr_trajectory_test.go`

- [ ] **Step 1: Add a test for clustered scores**

Append to the existing test file:

```go
// TestComputeEGFRTrajectory_ClusteredScores exposes the catastrophic
// cancellation bug in the single-pass ssTot formula. With scores tightly
// clustered around 70, the shortcut ssTot computation produces a value
// dominated by floating-point error, leading to an unreliable R².
func TestComputeEGFRTrajectory_ClusteredScores(t *testing.T) {
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	// 5 readings tightly clustered around 70 with a clear linear trend.
	readings := []EGFRReading{
		{Value: 70.0001, MeasuredAt: base},
		{Value: 70.0002, MeasuredAt: base.AddDate(0, 1, 0)},
		{Value: 70.0003, MeasuredAt: base.AddDate(0, 2, 0)},
		{Value: 70.0004, MeasuredAt: base.AddDate(0, 3, 0)},
		{Value: 70.0005, MeasuredAt: base.AddDate(0, 4, 0)},
	}

	result, err := ComputeEGFRTrajectory(readings)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With a perfect linear progression, R² should be near 1.0.
	// The single-pass formula returns ~0 due to catastrophic cancellation.
	if result.RSquared < 0.9 {
		t.Errorf("expected R² >= 0.9 for perfect linear trend, got %.6f", result.RSquared)
	}
}
```

- [ ] **Step 2: Run test to verify it fails (before the fix)**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/services/ -run "TestComputeEGFRTrajectory_ClusteredScores" -v
```

Expected: FAIL — R² is near 0 due to the bug.

- [ ] **Step 3: Commit failing test**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/egfr_trajectory_test.go
git commit -m "test(kb26): failing test for egfr OLS catastrophic cancellation

5 readings clustered around 70.0 with a perfect linear trend should
produce R² near 1.0. The single-pass ssTot formula returns ~0 due to
floating-point cancellation. This test pins the bug for the fix in Task 8."
```

---

### Task 8: Apply two-pass ssTot fix to egfr_trajectory.go

**Files:**
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/egfr_trajectory.go`

- [ ] **Step 1: Replace the ssTot computation**

In `ComputeEGFRTrajectory`, find the existing R² calculation block:

```go
// R² calculation
var ssTot, ssRes float64
intercept := meanY - slope*meanX
for _, r := range readings {
	x := r.MeasuredAt.Sub(earliest).Hours() / hoursPerYear
	predicted := intercept + slope*x
	ssRes += (r.Value - predicted) * (r.Value - predicted)
	ssTot += (r.Value - meanY) * (r.Value - meanY)
}
```

This is actually already two-pass for ssTot (it computes deviations from meanY in the loop). Looking more carefully — wait, let me check if egfr is actually using the buggy form or not. Let me re-read.

Looking again: `ssTot += (r.Value - meanY) * (r.Value - meanY)` is the two-pass form (computes deviation from mean inside the loop). So `egfr_trajectory.go` is already correct!

The bug only existed in `mri_domain_trajectory.go` which used `ssTot := sumY2 - n*meanY*meanY`. The new `trajectory_engine.go` from Task 3 already has the two-pass fix.

**Revised Task 8**: Verify the egfr regression test now passes (it should already pass against the existing egfr code). If it does pass, the test serves as a regression guard for the future.

- [ ] **Step 1 (revised): Run the new test**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/services/ -run "TestComputeEGFRTrajectory_ClusteredScores" -v
```

Expected: PASS — `egfr_trajectory.go` already uses the two-pass form. The test now serves as a regression guard.

- [ ] **Step 2: Commit (test-only commit since no fix was needed)**

If the test passes against existing code, amend the previous commit message to clarify:

```bash
git commit --amend -m "test(kb26): regression guard for egfr OLS numerical stability

5 readings clustered around 70.0 with a perfect linear trend produce
R² near 1.0. Test confirms the existing two-pass ssTot in egfr_trajectory.go
is correct and guards against future regression to the buggy single-pass
form. (The buggy form was only in mri_domain_trajectory.go and was fixed
during the trajectory_engine.go refactor in Task 3.)"
```

---

### Task 9: Add ClusteredScores test to trajectory engine

**Files:**
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/mri_domain_trajectory_test.go`

- [ ] **Step 1: Add the test**

Append to the existing test file:

```go
// TestDomainTrajectory_ClusteredScores_HighConfidence verifies the two-pass
// ssTot formula in the new trajectory engine. With a domain whose scores
// cluster tightly around a stable value but show a clear linear trend,
// R² should be near 1.0 (HIGH confidence). The previous single-pass formula
// would have returned R² near 0 (LOW confidence) due to catastrophic cancellation.
func TestDomainTrajectory_ClusteredScores_HighConfidence(t *testing.T) {
	now := time.Now()
	// All scores within 0.0005 of 70, but with a clear linear improvement.
	points := []models.DomainTrajectoryPoint{
		{Timestamp: now.Add(-13 * 24 * time.Hour), GlucoseScore: 70.0001, CardioScore: 65, BodyCompScore: 60, BehavioralScore: 65, CompositeScore: 65},
		{Timestamp: now.Add(-10 * 24 * time.Hour), GlucoseScore: 70.0002, CardioScore: 65, BodyCompScore: 60, BehavioralScore: 65, CompositeScore: 65},
		{Timestamp: now.Add(-7 * 24 * time.Hour),  GlucoseScore: 70.0003, CardioScore: 65, BodyCompScore: 60, BehavioralScore: 65, CompositeScore: 65},
		{Timestamp: now.Add(-4 * 24 * time.Hour),  GlucoseScore: 70.0004, CardioScore: 65, BodyCompScore: 60, BehavioralScore: 65, CompositeScore: 65},
		{Timestamp: now.Add(-1 * 24 * time.Hour),  GlucoseScore: 70.0005, CardioScore: 65, BodyCompScore: 60, BehavioralScore: 65, CompositeScore: 65},
	}

	result := defaultEngine().Compute("PAT-stable", points)
	glucose := result.DomainSlopes[models.DomainGlucose]

	// Perfect linear trend should yield R² near 1.0.
	if glucose.R2 < 0.9 {
		t.Errorf("expected glucose R² >= 0.9 for perfect linear trend, got %.6f (Confidence=%s)", glucose.R2, glucose.Confidence)
	}
	if glucose.Confidence != models.ConfidenceHigh {
		t.Errorf("expected HIGH confidence, got %s (R²=%.6f)", glucose.Confidence, glucose.R2)
	}
}
```

- [ ] **Step 2: Run test**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/services/ -run "TestDomainTrajectory_ClusteredScores_HighConfidence" -v
```

Expected: PASS (the new engine already has the two-pass fix from Task 3).

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/mri_domain_trajectory_test.go
git commit -m "test(kb26): clustered-scores regression guard for trajectory engine

Verifies the two-pass ssTot in trajectory_engine.go produces correct R²
for a domain whose scores cluster tightly but show a perfect linear trend.
Catches future regression if anyone reverts to the single-pass formula."
```

---

## Slice P2.1: KB-26 Observability (Tasks 10-13)

### Task 10: Define TrajectoryMetrics collector

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/metrics/trajectory_metrics.go`

- [ ] **Step 1: Create the metrics file**

```go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// TrajectoryMetrics groups all Prometheus metrics for the MHRI trajectory engine.
type TrajectoryMetrics struct {
	ComputeDuration         prometheus.Histogram
	ConcordantDeterioration *prometheus.CounterVec
	DivergenceTotal         *prometheus.CounterVec
	LeadingIndicatorTotal   *prometheus.CounterVec
	DomainCrossingTotal     *prometheus.CounterVec
	InsufficientData        prometheus.Counter
	PersistTotal            *prometheus.CounterVec
}

// NewTrajectoryMetrics registers all metrics with the global registry and
// returns the collector. Call once at server init.
func NewTrajectoryMetrics() *TrajectoryMetrics {
	return &TrajectoryMetrics{
		ComputeDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "kb26_trajectory_compute_duration_ms",
			Help:    "Latency of ComputeDecomposedTrajectory in milliseconds",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
		}),
		ConcordantDeterioration: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "kb26_trajectory_concordant_deterioration_total",
			Help: "Number of patients flagged with concordant multi-domain deterioration",
		}, []string{"domains_count"}),
		DivergenceTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "kb26_trajectory_divergence_total",
			Help: "Number of divergence pairs detected",
		}, []string{"improving_domain", "declining_domain"}),
		LeadingIndicatorTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "kb26_trajectory_leading_indicator_total",
			Help: "Behavioral leading indicator fires by lagging domain",
		}, []string{"lagging_domain"}),
		DomainCrossingTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "kb26_trajectory_domain_crossing_total",
			Help: "Domain category crossings by domain and direction",
		}, []string{"domain", "direction"}),
		InsufficientData: promauto.NewCounter(prometheus.CounterOpts{
			Name: "kb26_trajectory_insufficient_data_total",
			Help: "Trajectory requests blocked by <2 data points",
		}),
		PersistTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "kb26_trajectory_persist_total",
			Help: "Trajectory history persistence outcomes",
		}, []string{"result"}),
	}
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go build ./internal/metrics/
```

Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/metrics/trajectory_metrics.go
git commit -m "feat(kb26): TrajectoryMetrics Prometheus collector

7 metrics covering compute latency (histogram), concordant deterioration
counts, divergence pair detection, leading indicator fires, domain
crossings, insufficient-data blocks, and persistence outcomes."
```

---

### Task 11: Inject metrics into TrajectoryEngine

**Files:**
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/trajectory_engine.go`

- [ ] **Step 1: Add metrics field to engine**

Update the engine struct and constructor:

```go
type TrajectoryEngine struct {
	thresholds config.TrajectoryThresholds
	metrics    *metrics.TrajectoryMetrics // optional, nil-safe
}

// NewTrajectoryEngine constructs an engine with the given thresholds.
// Metrics are nil-safe; pass nil for tests that don't care about telemetry.
func NewTrajectoryEngine(thresholds config.TrajectoryThresholds, metrics *metrics.TrajectoryMetrics) *TrajectoryEngine {
	return &TrajectoryEngine{thresholds: thresholds, metrics: metrics}
}
```

Add the metrics import:
```go
import (
	"math"
	"sort"
	"strconv"
	"time"

	"kb-26-metabolic-digital-twin/internal/config"
	"kb-26-metabolic-digital-twin/internal/metrics"
	"kb-26-metabolic-digital-twin/internal/models"
)
```

- [ ] **Step 2: Instrument Compute method**

At the top of `Compute`, add a deferred latency measurement:

```go
func (e *TrajectoryEngine) Compute(patientID string, points []models.DomainTrajectoryPoint) models.DecomposedTrajectory {
	start := time.Now()
	defer func() {
		if e.metrics != nil {
			e.metrics.ComputeDuration.Observe(float64(time.Since(start).Milliseconds()))
		}
	}()

	// ... existing logic ...
}
```

After the `len(points) < 2` guard:
```go
if len(points) < 2 {
	if e.metrics != nil {
		e.metrics.InsufficientData.Inc()
	}
	result.CompositeTrend = models.TrendInsufficient
	// ...
}
```

After computing concordant deterioration:
```go
result.ConcordantDeterioration = decliningCount >= e.thresholds.Concordant.MinDomainsDeclining
if result.ConcordantDeterioration && e.metrics != nil {
	e.metrics.ConcordantDeterioration.WithLabelValues(strconv.Itoa(decliningCount)).Inc()
}
```

After detecting divergences:
```go
result.Divergences = e.detectDivergences(result.DomainSlopes)
result.HasDiscordantTrend = len(result.Divergences) > 0
if e.metrics != nil {
	for _, div := range result.Divergences {
		e.metrics.DivergenceTotal.WithLabelValues(string(div.ImprovingDomain), string(div.DecliningDomain)).Inc()
	}
}
```

After detecting domain crossings:
```go
result.DomainCrossings = e.detectDomainCrossings(sorted, domainExtractors)
if e.metrics != nil {
	for _, c := range result.DomainCrossings {
		e.metrics.DomainCrossingTotal.WithLabelValues(string(c.Domain), c.Direction).Inc()
	}
}
```

After detecting leading indicators:
```go
result.LeadingIndicators = e.detectLeadingIndicators(sorted, result.DomainSlopes)
if e.metrics != nil {
	for _, lead := range result.LeadingIndicators {
		for _, lagging := range lead.LaggingDomains {
			e.metrics.LeadingIndicatorTotal.WithLabelValues(string(lagging)).Inc()
		}
	}
}
```

- [ ] **Step 3: Update all NewTrajectoryEngine call sites**

The constructor signature changed from `NewTrajectoryEngine(thresholds)` to `NewTrajectoryEngine(thresholds, metrics)`. Update:

- `internal/api/server.go`: pass `s.metrics.Trajectory` (after Task 12 wires it up; for now pass nil and revisit)
- `internal/services/mri_domain_trajectory_test.go`: `defaultEngine()` helper passes nil for metrics
- `pkg/trajectory/models.go`: `Compute` wrapper passes nil for metrics

```go
// In test helper:
func defaultEngine() *TrajectoryEngine {
	return NewTrajectoryEngine(config.DefaultTrajectoryThresholds(), nil)
}

// In pkg/trajectory:
func Compute(patientID string, points []DomainTrajectoryPoint) DecomposedTrajectory {
	engine := services.NewTrajectoryEngine(config.DefaultTrajectoryThresholds(), nil)
	return engine.Compute(patientID, points)
}
```

- [ ] **Step 4: Build and test**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go build ./... && go test ./internal/services/ -count=1 2>&1 | tail -5
```

Expected: All tests still pass (metrics are nil-safe).

- [ ] **Step 5: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/trajectory_engine.go backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/mri_domain_trajectory_test.go backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/pkg/trajectory/models.go
git commit -m "feat(kb26): instrument TrajectoryEngine with Prometheus metrics

ComputeDuration histogram, concordant/divergence/leading indicator/crossing
counters, insufficient-data and persist-result counters. Metrics are
nil-safe so tests pass nil and the production handler passes a real
collector. All existing tests pass unchanged."
```

---

### Task 12: Wire metrics collector into Server init

**Files:**
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/server.go`

- [ ] **Step 1: Add trajectory metrics to Server**

In `Server` struct, add:
```go
trajectoryMetrics *metrics.TrajectoryMetrics
```

In `InitServices` (or wherever services initialize):
```go
s.trajectoryMetrics = metrics.NewTrajectoryMetrics()

trajectoryThresholds, err := config.LoadTrajectoryThresholds(s.cfg.TrajectoryThresholdsPath)
if err != nil {
	s.logger.Warn("failed to load trajectory thresholds, using defaults", zap.Error(err))
	trajectoryThresholds = config.DefaultTrajectoryThresholds()
}
s.trajectoryEngine = services.NewTrajectoryEngine(trajectoryThresholds, s.trajectoryMetrics)
```

- [ ] **Step 2: Build and test**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go build ./... && go test ./internal/services/ -count=1 2>&1 | tail -5
```

Expected: Build clean, all tests pass.

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/server.go
git commit -m "feat(kb26): wire TrajectoryMetrics into Server init

Server now constructs a TrajectoryMetrics collector and passes it into
NewTrajectoryEngine alongside the loaded thresholds. Metrics flow to
the existing /metrics Prometheus scrape endpoint via promauto."
```

---

### Task 13: Instrument persist outcome and verify metrics endpoint

**Files:**
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/domain_trajectory_handlers.go`

- [ ] **Step 1: Increment persist counter**

In `persistDomainTrajectorySnapshot` (or wherever the upsert happens), wrap the GORM call:

```go
err := gormDB.Where("patient_id = ? AND snapshot_date = ?", history.PatientID, history.SnapshotDate).
	Assign(history).
	FirstOrCreate(&history).Error

if s.trajectoryMetrics != nil {
	if err != nil {
		s.trajectoryMetrics.PersistTotal.WithLabelValues("fail").Inc()
	} else {
		s.trajectoryMetrics.PersistTotal.WithLabelValues("ok").Inc()
	}
}

return err
```

- [ ] **Step 2: Build and run**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go build ./... && go vet ./...
```

Expected: Clean.

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/domain_trajectory_handlers.go
git commit -m "feat(kb26): instrument trajectory history persist outcome

Increments kb26_trajectory_persist_total{result=ok|fail} on each
upsert attempt. Failures remain non-fatal — metric just records the
outcome for shadow-deployment monitoring."
```

---

## Slice P2.5: Kafka Publisher (Tasks 14-17)

### Task 14: Define event schema and publisher interface

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/trajectory_publisher.go`

- [ ] **Step 1: Create the publisher file**

```go
package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"kb-26-metabolic-digital-twin/internal/models"
)

// DomainTrajectoryComputedEvent is the Kafka event published after each
// successful trajectory computation. Module 13 Flink state-sync consumes
// this to populate its domain_velocities map.
type DomainTrajectoryComputedEvent struct {
	EventType    string    `json:"event_type"`
	EventVersion string    `json:"event_version"`
	EventID      string    `json:"event_id"`
	EmittedAt    time.Time `json:"emitted_at"`
	PatientID    string    `json:"patient_id"`
	WindowDays   int       `json:"window_days"`
	DataPoints   int       `json:"data_points"`

	Composite CompositeSummary `json:"composite"`
	Domains   map[string]DomainSummary `json:"domains"`

	DominantDriver         *string `json:"dominant_driver,omitempty"`
	DriverContributionPct  float64 `json:"driver_contribution_pct"`
	HasDiscordantTrend     bool    `json:"has_discordant_trend"`
	ConcordantDeterioration bool   `json:"concordant_deterioration"`
	DomainsDeteriorating   int     `json:"domains_deteriorating"`
}

type CompositeSummary struct {
	SlopePerDay float64 `json:"slope_per_day"`
	Trend       string  `json:"trend"`
	StartScore  float64 `json:"start_score"`
	EndScore    float64 `json:"end_score"`
}

type DomainSummary struct {
	SlopePerDay float64 `json:"slope_per_day"`
	Trend       string  `json:"trend"`
	Confidence  string  `json:"confidence"`
	RSquared    float64 `json:"r_squared"`
}

// NewDomainTrajectoryComputedEvent builds an event from a DecomposedTrajectory.
func NewDomainTrajectoryComputedEvent(traj *models.DecomposedTrajectory) DomainTrajectoryComputedEvent {
	domains := make(map[string]DomainSummary, len(traj.DomainSlopes))
	for d, slope := range traj.DomainSlopes {
		domains[string(d)] = DomainSummary{
			SlopePerDay: slope.SlopePerDay,
			Trend:       slope.Trend,
			Confidence:  slope.Confidence,
			RSquared:    slope.R2,
		}
	}

	var dominant *string
	if traj.DominantDriver != nil {
		s := string(*traj.DominantDriver)
		dominant = &s
	}

	return DomainTrajectoryComputedEvent{
		EventType:    "DomainTrajectoryComputed",
		EventVersion: "v1",
		EventID:      uuid.New().String(),
		EmittedAt:    time.Now().UTC(),
		PatientID:    traj.PatientID,
		WindowDays:   traj.WindowDays,
		DataPoints:   traj.DataPoints,
		Composite: CompositeSummary{
			SlopePerDay: traj.CompositeSlope,
			Trend:       traj.CompositeTrend,
			StartScore:  traj.CompositeStartScore,
			EndScore:    traj.CompositeEndScore,
		},
		Domains:                domains,
		DominantDriver:         dominant,
		DriverContributionPct:  traj.DriverContribution,
		HasDiscordantTrend:     traj.HasDiscordantTrend,
		ConcordantDeterioration: traj.ConcordantDeterioration,
		DomainsDeteriorating:   traj.DomainsDeteriorating,
	}
}

// TrajectoryPublisher publishes DomainTrajectoryComputed events.
type TrajectoryPublisher interface {
	Publish(ctx context.Context, event DomainTrajectoryComputedEvent) error
}

// KafkaTrajectoryPublisher writes events to a Kafka topic.
type KafkaTrajectoryPublisher struct {
	writer *kafka.Writer
	topic  string
	logger *zap.Logger
}

// NewKafkaTrajectoryPublisher constructs a Kafka producer for trajectory events.
func NewKafkaTrajectoryPublisher(brokers []string, topic string, logger *zap.Logger) *KafkaTrajectoryPublisher {
	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireOne,
	}
	return &KafkaTrajectoryPublisher{writer: w, topic: topic, logger: logger}
}

// Publish writes an event to Kafka. Errors are returned to the caller, which
// is expected to log and continue (publish failure is non-fatal).
func (p *KafkaTrajectoryPublisher) Publish(ctx context.Context, event DomainTrajectoryComputedEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.PatientID),
		Value: body,
	})
}

// NoopTrajectoryPublisher is a publisher that drops all events. Used in tests
// and in environments where Kafka is not configured.
type NoopTrajectoryPublisher struct{}

func (NoopTrajectoryPublisher) Publish(ctx context.Context, event DomainTrajectoryComputedEvent) error {
	return nil
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go build ./internal/services/
```

Expected: No errors. (`segmentio/kafka-go` and `go.uber.org/zap` already in go.mod.)

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/trajectory_publisher.go
git commit -m "feat(kb26): trajectory publisher for Kafka

DomainTrajectoryComputedEvent v1 schema covering composite + per-domain
slopes/trends/R²/confidence + dominant driver + flags. KafkaTrajectoryPublisher
writes to kb26.domain_trajectory.v1 keyed by patient_id. NoopTrajectoryPublisher
for tests and Kafka-disabled environments."
```

---

### Task 15: Test publisher with mock writer

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/trajectory_publisher_test.go`

- [ ] **Step 1: Write tests**

```go
package services

import (
	"context"
	"encoding/json"
	"testing"

	"kb-26-metabolic-digital-twin/internal/models"
)

func TestNewDomainTrajectoryComputedEvent_PopulatesAllFields(t *testing.T) {
	driver := models.DomainGlucose
	traj := &models.DecomposedTrajectory{
		PatientID:               "pat-001",
		WindowDays:              13,
		DataPoints:              5,
		CompositeSlope:          -1.42,
		CompositeTrend:          models.TrendDeclining,
		CompositeStartScore:     62.0,
		CompositeEndScore:       42.0,
		DominantDriver:          &driver,
		DriverContribution:      45.3,
		HasDiscordantTrend:      false,
		ConcordantDeterioration: true,
		DomainsDeteriorating:    3,
		DomainSlopes: map[models.MHRIDomain]models.DomainSlope{
			models.DomainGlucose: {
				Domain:      models.DomainGlucose,
				SlopePerDay: -1.67,
				Trend:       models.TrendRapidDeclining,
				Confidence:  models.ConfidenceHigh,
				R2:          0.98,
			},
		},
	}

	event := NewDomainTrajectoryComputedEvent(traj)

	if event.EventType != "DomainTrajectoryComputed" {
		t.Errorf("expected event_type DomainTrajectoryComputed, got %s", event.EventType)
	}
	if event.EventVersion != "v1" {
		t.Errorf("expected v1, got %s", event.EventVersion)
	}
	if event.PatientID != "pat-001" {
		t.Errorf("expected patient_id pat-001, got %s", event.PatientID)
	}
	if !event.ConcordantDeterioration {
		t.Error("expected ConcordantDeterioration true")
	}
	if event.DomainsDeteriorating != 3 {
		t.Errorf("expected DomainsDeteriorating 3, got %d", event.DomainsDeteriorating)
	}

	glucose, ok := event.Domains["GLUCOSE"]
	if !ok {
		t.Fatal("expected GLUCOSE domain in event")
	}
	if glucose.SlopePerDay != -1.67 {
		t.Errorf("expected glucose slope -1.67, got %.2f", glucose.SlopePerDay)
	}
	if glucose.Confidence != models.ConfidenceHigh {
		t.Errorf("expected glucose confidence HIGH, got %s", glucose.Confidence)
	}

	if event.DominantDriver == nil || *event.DominantDriver != "GLUCOSE" {
		t.Errorf("expected dominant_driver GLUCOSE, got %v", event.DominantDriver)
	}
}

func TestDomainTrajectoryComputedEvent_JSONRoundTrip(t *testing.T) {
	traj := &models.DecomposedTrajectory{
		PatientID:    "pat-002",
		WindowDays:   7,
		DataPoints:   3,
		CompositeSlope: 0.5,
		CompositeTrend: models.TrendImproving,
		DomainSlopes: map[models.MHRIDomain]models.DomainSlope{
			models.DomainCardio: {Domain: models.DomainCardio, SlopePerDay: 0.6, Trend: models.TrendImproving, Confidence: models.ConfidenceHigh, R2: 0.92},
		},
	}

	event := NewDomainTrajectoryComputedEvent(traj)
	body, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var roundtrip DomainTrajectoryComputedEvent
	if err := json.Unmarshal(body, &roundtrip); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if roundtrip.PatientID != "pat-002" {
		t.Errorf("expected pat-002, got %s", roundtrip.PatientID)
	}
	if roundtrip.Domains["CARDIO"].SlopePerDay != 0.6 {
		t.Errorf("expected cardio slope 0.6, got %.2f", roundtrip.Domains["CARDIO"].SlopePerDay)
	}
}

func TestNoopTrajectoryPublisher_NeverErrors(t *testing.T) {
	noop := NoopTrajectoryPublisher{}
	event := DomainTrajectoryComputedEvent{PatientID: "pat-003"}

	if err := noop.Publish(context.Background(), event); err != nil {
		t.Errorf("noop publisher should not error, got %v", err)
	}
}
```

- [ ] **Step 2: Run tests**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/services/ -run "TestNewDomainTrajectoryComputedEvent|TestDomainTrajectoryComputedEvent_JSONRoundTrip|TestNoopTrajectoryPublisher" -v
```

Expected: 3/3 PASS.

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/trajectory_publisher_test.go
git commit -m "test(kb26): trajectory publisher event construction + JSON roundtrip

3 tests: NewDomainTrajectoryComputedEvent populates all fields,
JSON marshal/unmarshal preserves shape, NoopTrajectoryPublisher
never errors. Kafka writer integration is tested at the broker level
in the integration suite (out of scope for this unit test)."
```

---

### Task 16: Wire publisher into TrajectoryEngine and Server

**Files:**
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/trajectory_engine.go`
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/server.go`

- [ ] **Step 1: Add publisher field to engine**

```go
type TrajectoryEngine struct {
	thresholds config.TrajectoryThresholds
	metrics    *metrics.TrajectoryMetrics
	publisher  TrajectoryPublisher
	logger     *zap.Logger
}

func NewTrajectoryEngine(
	thresholds config.TrajectoryThresholds,
	m *metrics.TrajectoryMetrics,
	publisher TrajectoryPublisher,
	logger *zap.Logger,
) *TrajectoryEngine {
	if publisher == nil {
		publisher = NoopTrajectoryPublisher{}
	}
	return &TrajectoryEngine{thresholds: thresholds, metrics: m, publisher: publisher, logger: logger}
}
```

- [ ] **Step 2: Publish at the end of Compute**

Just before `return result`:

```go
// Publish event for downstream consumers (Module 13 Flink state-sync).
// Publish failure is non-fatal — log and continue.
if e.publisher != nil {
	event := NewDomainTrajectoryComputedEvent(&result)
	if err := e.publisher.Publish(context.Background(), event); err != nil && e.logger != nil {
		e.logger.Warn("failed to publish trajectory event", zap.Error(err))
	}
}

return result
```

Add `"context"` and `"go.uber.org/zap"` imports.

- [ ] **Step 3: Update all NewTrajectoryEngine call sites**

The constructor now takes 4 args. Update:
- `internal/api/server.go`: pass `s.trajectoryMetrics, s.trajectoryPublisher, s.logger`
- `internal/services/mri_domain_trajectory_test.go`: `defaultEngine()` passes nil for metrics/publisher and a noop logger
- `pkg/trajectory/models.go`: passes nil/nil/nil

```go
// Test helper
func defaultEngine() *TrajectoryEngine {
	return NewTrajectoryEngine(config.DefaultTrajectoryThresholds(), nil, nil, zap.NewNop())
}

// Server init (add new field for publisher)
s.trajectoryPublisher = services.NewKafkaTrajectoryPublisher(
	s.cfg.KafkaBrokers,
	"kb26.domain_trajectory.v1",
	s.logger,
)
s.trajectoryEngine = services.NewTrajectoryEngine(
	trajectoryThresholds,
	s.trajectoryMetrics,
	s.trajectoryPublisher,
	s.logger,
)

// Facade
func Compute(patientID string, points []DomainTrajectoryPoint) DecomposedTrajectory {
	engine := services.NewTrajectoryEngine(config.DefaultTrajectoryThresholds(), nil, nil, nil)
	return engine.Compute(patientID, points)
}
```

- [ ] **Step 4: Build and test**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go build ./... && go test ./internal/services/ -count=1 2>&1 | tail -5
```

Expected: Build clean, all tests pass (publisher is noop in tests).

- [ ] **Step 5: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/trajectory_engine.go backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/server.go backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/mri_domain_trajectory_test.go backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/pkg/trajectory/models.go
git commit -m "feat(kb26): publish DomainTrajectoryComputed events from engine

TrajectoryEngine now takes a TrajectoryPublisher. After each successful
Compute, builds an event and publishes via Kafka (or noop in tests).
Publish failure is logged and non-fatal — matches the existing
'persistence failure is non-fatal' pattern in the API handler."
```

---

### Task 17: Document the consumer contract for Module 13

**Files:**
- Create: `backend/stream-services/docs/kb26_domain_trajectory_consumer.md`

- [ ] **Step 1: Write the consumer doc**

```markdown
# KB-26 Domain Trajectory Consumer Contract

**Topic**: `kb26.domain_trajectory.v1`
**Producer**: `kb-26-metabolic-digital-twin` (Go)
**Consumers**: Module 13 Flink state-sync (Java/Flink); future analytics consumers may also subscribe

## Event Schema

```json
{
  "event_type": "DomainTrajectoryComputed",
  "event_version": "v1",
  "event_id": "uuid",
  "emitted_at": "2026-04-14T10:23:45Z",
  "patient_id": "uuid",
  "window_days": 13,
  "data_points": 5,
  "composite": {
    "slope_per_day": -1.42,
    "trend": "DECLINING",
    "start_score": 62.0,
    "end_score": 42.0
  },
  "domains": {
    "GLUCOSE":    { "slope_per_day": -1.67, "trend": "RAPID_DECLINING", "confidence": "HIGH", "r_squared": 0.98 },
    "CARDIO":     { "slope_per_day": -1.65, "trend": "RAPID_DECLINING", "confidence": "HIGH", "r_squared": 0.97 },
    "BODY_COMP":  { "slope_per_day": -0.17, "trend": "STABLE",          "confidence": "HIGH", "r_squared": 0.94 },
    "BEHAVIORAL": { "slope_per_day": -3.50, "trend": "RAPID_DECLINING", "confidence": "HIGH", "r_squared": 0.99 }
  },
  "dominant_driver": "GLUCOSE",
  "driver_contribution_pct": 45.3,
  "has_discordant_trend": false,
  "concordant_deterioration": true,
  "domains_deteriorating": 3
}
```

## Trend values
`RAPID_IMPROVING` | `IMPROVING` | `STABLE` | `DECLINING` | `RAPID_DECLINING` | `INSUFFICIENT_DATA`

## Confidence values
`HIGH` (R² >= 0.5) | `MODERATE` (0.25-0.5) | `LOW` (<0.25)

## Domain keys
`GLUCOSE`, `CARDIO`, `BODY_COMP`, `BEHAVIORAL`

## Partitioning
Events are keyed by `patient_id` so all events for a given patient land on the same partition. Consumers can rely on per-patient ordering.

## Module 13 Integration Requirements

Module 13's Flink state-sync should:

1. Subscribe to `kb26.domain_trajectory.v1`
2. Deserialize the JSON payload
3. Update Flink's `domain_velocities` map keyed by `patient_id`:
   ```
   domain_velocities[patient_id] = {
     "glucose":    domains.GLUCOSE.slope_per_day,
     "cardio":     domains.CARDIO.slope_per_day,
     "body_comp":  domains.BODY_COMP.slope_per_day,
     "behavioral": domains.BEHAVIORAL.slope_per_day,
   }
   ```
4. Ignore events older than 48 hours (`emitted_at` < now - 48h) to prevent replay from populating stale state
5. Acknowledge offsets only after state update is committed

## Failure Modes

- **Producer down**: events are not emitted; Module 13 state stays at last known values. The KB-26 API endpoint `GET /api/v1/kb26/mri/:patientId/domain-trajectory` still works as a synchronous fallback.
- **Consumer down**: events accumulate in Kafka topic (default retention applies). Module 13 catches up on restart; events older than 48h are dropped.
- **Schema evolution**: `event_version` field will increment on breaking changes. Consumers should reject unknown versions (not silently ignore).

## Versioning

This is `v1`. Future versions will be published to new topics (`kb26.domain_trajectory.v2`) for safe rollout. Producers do not dual-write across versions.

## Related

- KB-26 producer: `internal/services/trajectory_publisher.go`
- KB-26 producer init: `internal/api/server.go::InitServices`
- Original E2E gap that motivated this event: patient Rajesh Kumar trace, Module 13 `domain_velocities` empty
```

- [ ] **Step 2: Commit**

```bash
git add backend/stream-services/docs/kb26_domain_trajectory_consumer.md
git commit -m "docs: Module 13 Flink consumer contract for trajectory events

Schema, partitioning, integration steps, failure modes, and versioning
strategy for kb26.domain_trajectory.v1. Module 13 Java/Flink consumer
implementation tracked separately."
```

---

## Slice P2.4: Seasonal Adjustment (Tasks 18-22)

### Task 18: Create India seasonal calendar YAML

**Files:**
- Create: `backend/shared-infrastructure/market-configs/india/seasonal_calendar.yaml`
- Create: `backend/shared-infrastructure/market-configs/australia/seasonal_calendar.yaml`

- [ ] **Step 1: Create the India calendar**

```yaml
# India seasonal calendar for trajectory alert suppression.
# Each window names the season, its affected MHRI domains, the suppression
# mode, and a date range. Year-specific dates (festivals) use ISO-8601
# absolute dates; recurring seasons (heat, monsoon) use day-of-year.

windows:
  - name: diwali
    start: "2026-11-04"
    end: "2026-11-14"
    affected_domains: [GLUCOSE, BODY_COMP]
    mode: DOWNGRADE_URGENCY
    rationale: "Festival eating patterns predictably worsen glucose and body composition for 7-10 days"

  - name: ramadan
    start: "2026-03-17"
    end: "2026-04-15"
    affected_domains: [GLUCOSE, BEHAVIORAL]
    mode: DOWNGRADE_URGENCY
    rationale: "Altered eating windows affect glycaemic control and monitoring patterns"

  - name: pongal
    start: "2026-01-14"
    end: "2026-01-17"
    affected_domains: [GLUCOSE, BODY_COMP]
    mode: DOWNGRADE_URGENCY
    rationale: "South Indian harvest festival with traditional sweet/rice-based meals"

  - name: summer_heat
    start_doy: 121  # May 1
    end_doy: 181    # June 30
    affected_domains: [CARDIO]
    mode: DOWNGRADE_URGENCY
    rationale: "Extreme heat causes dehydration and volume depletion affecting BP readings"
```

- [ ] **Step 2: Create empty Australia seed**

```yaml
# Australia seasonal calendar — Phase 2 seed (empty).
# Clinical review will populate windows for: harvest season (rural Indigenous),
# QLD/WA summer heat, and remote area connectivity drops.

windows: []
```

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/market-configs/india/seasonal_calendar.yaml backend/shared-infrastructure/market-configs/australia/seasonal_calendar.yaml
git commit -m "config: seed seasonal calendars for India + Australia

India: Diwali, Ramadan, Pongal (year-specific 2026 dates), summer heat
(day-of-year recurring). All in DOWNGRADE_URGENCY mode pending clinical
review of false-positive rates from Phase 0 shadow data.
Australia: empty seed to establish the contract; clinical review to populate."
```

---

### Task 19: SeasonalContext service + tests

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/seasonal_context.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/seasonal_context_test.go`

- [ ] **Step 1: Create the service**

```go
package services

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	dtModels "kb-26-metabolic-digital-twin/pkg/trajectory"
)

// SeasonalWindow describes a date range during which trajectory cards for
// specific domains should be suppressed or downgraded.
type SeasonalWindow struct {
	Name             string             `yaml:"name"`
	Start            string             `yaml:"start,omitempty"`     // ISO date or empty
	End              string             `yaml:"end,omitempty"`
	StartDOY         int                `yaml:"start_doy,omitempty"` // 1-366 or 0
	EndDOY           int                `yaml:"end_doy,omitempty"`
	AffectedDomains  []dtModels.MHRIDomain `yaml:"affected_domains"`
	Mode             string             `yaml:"mode"` // DOWNGRADE_URGENCY | SUPPRESS
	Rationale        string             `yaml:"rationale"`
}

type seasonalCalendarFile struct {
	Windows []SeasonalWindow `yaml:"windows"`
}

// SeasonalContext holds the loaded calendar for a single market.
type SeasonalContext struct {
	market  string
	windows []SeasonalWindow
}

// NewSeasonalContext loads a seasonal calendar from a YAML file.
// Returns an empty context (no suppression) if the file is missing.
func NewSeasonalContext(market, calendarPath string) (*SeasonalContext, error) {
	data, err := os.ReadFile(calendarPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &SeasonalContext{market: market, windows: nil}, nil
		}
		return nil, fmt.Errorf("read seasonal calendar: %w", err)
	}

	var file seasonalCalendarFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse seasonal calendar: %w", err)
	}

	return &SeasonalContext{market: market, windows: file.Windows}, nil
}

// ActiveWindows returns the seasonal windows active at the given timestamp.
func (s *SeasonalContext) ActiveWindows(ts time.Time) []SeasonalWindow {
	var active []SeasonalWindow
	for _, w := range s.windows {
		if windowActiveAt(w, ts) {
			active = append(active, w)
		}
	}
	return active
}

// ShouldSuppress returns (suppress, downgrade, rationale) for a domain at a given time.
// If multiple windows apply, SUPPRESS wins over DOWNGRADE_URGENCY.
func (s *SeasonalContext) ShouldSuppress(domain dtModels.MHRIDomain, ts time.Time) (suppress bool, downgrade bool, rationale string) {
	for _, w := range s.ActiveWindows(ts) {
		if !containsDomain(w.AffectedDomains, domain) {
			continue
		}
		switch w.Mode {
		case "SUPPRESS":
			return true, false, w.Rationale
		case "DOWNGRADE_URGENCY":
			downgrade = true
			rationale = w.Rationale
			// keep scanning for SUPPRESS which would override
		}
	}
	return false, downgrade, rationale
}

func windowActiveAt(w SeasonalWindow, ts time.Time) bool {
	// Year-specific window
	if w.Start != "" && w.End != "" {
		start, err1 := time.Parse("2006-01-02", w.Start)
		end, err2 := time.Parse("2006-01-02", w.End)
		if err1 != nil || err2 != nil {
			return false
		}
		return !ts.Before(start) && !ts.After(end.Add(24*time.Hour))
	}
	// Recurring day-of-year window
	if w.StartDOY > 0 && w.EndDOY > 0 {
		doy := ts.YearDay()
		return doy >= w.StartDOY && doy <= w.EndDOY
	}
	return false
}

func containsDomain(domains []dtModels.MHRIDomain, target dtModels.MHRIDomain) bool {
	for _, d := range domains {
		if d == target {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Write tests**

```go
package services

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	dtModels "kb-26-metabolic-digital-twin/pkg/trajectory"
)

func TestSeasonalContext_EmptyOnMissingFile(t *testing.T) {
	tmp := t.TempDir()
	ctx, err := NewSeasonalContext("india", filepath.Join(tmp, "missing.yaml"))
	if err != nil {
		t.Fatalf("expected nil error on missing file, got %v", err)
	}
	if len(ctx.windows) != 0 {
		t.Errorf("expected no windows, got %d", len(ctx.windows))
	}
}

func TestSeasonalContext_DiwaliWindow_DowngradesGlucose(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "calendar.yaml")
	yaml := `
windows:
  - name: diwali
    start: "2026-11-04"
    end: "2026-11-14"
    affected_domains: [GLUCOSE, BODY_COMP]
    mode: DOWNGRADE_URGENCY
    rationale: "festival eating"
`
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	ctx, err := NewSeasonalContext("india", path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	// Inside Diwali window
	tsDiwali := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	suppress, downgrade, rationale := ctx.ShouldSuppress(dtModels.DomainGlucose, tsDiwali)
	if suppress {
		t.Error("expected no suppress for DOWNGRADE_URGENCY mode")
	}
	if !downgrade {
		t.Error("expected downgrade=true during Diwali for GLUCOSE")
	}
	if rationale == "" {
		t.Error("expected non-empty rationale")
	}

	// CARDIO is not affected
	suppress, downgrade, _ = ctx.ShouldSuppress(dtModels.DomainCardio, tsDiwali)
	if suppress || downgrade {
		t.Error("expected no suppression for CARDIO during Diwali")
	}

	// Outside Diwali window
	tsAfter := time.Date(2026, 12, 1, 12, 0, 0, 0, time.UTC)
	suppress, downgrade, _ = ctx.ShouldSuppress(dtModels.DomainGlucose, tsAfter)
	if suppress || downgrade {
		t.Error("expected no suppression outside Diwali")
	}
}

func TestSeasonalContext_DOYWindow_HeatSeason(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "calendar.yaml")
	yaml := `
windows:
  - name: summer_heat
    start_doy: 121
    end_doy: 181
    affected_domains: [CARDIO]
    mode: DOWNGRADE_URGENCY
    rationale: "extreme heat"
`
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	ctx, err := NewSeasonalContext("india", path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	// May 15 = doy 135 (within window)
	tsMay := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)
	_, downgrade, _ := ctx.ShouldSuppress(dtModels.DomainCardio, tsMay)
	if !downgrade {
		t.Errorf("expected CARDIO downgrade on May 15 (doy %d)", tsMay.YearDay())
	}

	// April 1 = doy 91 (before window)
	tsApr := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	_, downgrade, _ = ctx.ShouldSuppress(dtModels.DomainCardio, tsApr)
	if downgrade {
		t.Error("expected no CARDIO downgrade on April 1")
	}
}

func TestSeasonalContext_SuppressOverridesDowngrade(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "calendar.yaml")
	yaml := `
windows:
  - name: window1
    start: "2026-06-01"
    end: "2026-06-30"
    affected_domains: [GLUCOSE]
    mode: DOWNGRADE_URGENCY
    rationale: "downgrade"
  - name: window2
    start: "2026-06-15"
    end: "2026-06-20"
    affected_domains: [GLUCOSE]
    mode: SUPPRESS
    rationale: "full suppress"
`
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	ctx, err := NewSeasonalContext("test", path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	// June 17 — both windows active
	ts := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	suppress, _, _ := ctx.ShouldSuppress(dtModels.DomainGlucose, ts)
	if !suppress {
		t.Error("expected SUPPRESS to win when both modes active")
	}
}
```

- [ ] **Step 3: Run tests**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go test ./internal/services/ -run "TestSeasonalContext" -v
```

Expected: 4/4 PASS.

- [ ] **Step 4: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/seasonal_context.go backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/seasonal_context_test.go
git commit -m "feat(kb23): seasonal context service for trajectory alert suppression

Loads market-specific seasonal_calendar.yaml from market-configs.
Two suppression modes: DOWNGRADE_URGENCY (one urgency level softer)
and SUPPRESS (no card emitted). Supports both year-specific dates
(festivals) and day-of-year recurring windows (heat seasons).
SUPPRESS overrides DOWNGRADE_URGENCY when multiple windows overlap.
4 unit tests covering empty calendar, year-specific window, DOY window,
and override priority."
```

---

### Task 20: Wire SeasonalContext into trajectory card rules

**Files:**
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/trajectory_card_rules.go`

- [ ] **Step 1: Add an optional SeasonalContext parameter (backward compat)**

The existing `EvaluateTrajectoryCards(traj)` should keep working for callers that don't pass a seasonal context. Add a new variant:

```go
// EvaluateTrajectoryCardsWithSeasonalContext is the season-aware variant.
// Callers without a SeasonalContext should use EvaluateTrajectoryCards (no suppression).
func EvaluateTrajectoryCardsWithSeasonalContext(
	traj *dtModels.DecomposedTrajectory,
	seasonalCtx *SeasonalContext,
	now time.Time,
) []TrajectoryCard {
	if traj == nil {
		return nil
	}

	cards := EvaluateTrajectoryCards(traj)
	if seasonalCtx == nil {
		return cards
	}

	filtered := make([]TrajectoryCard, 0, len(cards))
	for _, card := range cards {
		// Determine which domain(s) the card is about.
		domain := cardDomain(card, traj)
		if domain == "" {
			filtered = append(filtered, card) // no domain — pass through
			continue
		}

		suppress, downgrade, rationale := seasonalCtx.ShouldSuppress(domain, now)
		if suppress {
			continue // do not emit
		}
		if downgrade {
			card.Urgency = downgradeUrgency(card.Urgency)
			card.Rationale += " (seasonal context: " + rationale + ")"
		}
		filtered = append(filtered, card)
	}

	return filtered
}

// cardDomain extracts the primary domain a card is about, or "" if multi-domain.
func cardDomain(card TrajectoryCard, traj *dtModels.DecomposedTrajectory) dtModels.MHRIDomain {
	if card.Domain != "" {
		return dtModels.MHRIDomain(card.Domain)
	}
	// CONCORDANT_DETERIORATION cards are multi-domain — only suppress if ALL declining
	// domains are in seasonal windows. Conservative: don't suppress concordant cards
	// from a single seasonal window. Return empty so they pass through.
	return ""
}

// downgradeUrgency returns the next less-urgent urgency level.
func downgradeUrgency(urgency string) string {
	switch urgency {
	case "IMMEDIATE":
		return "URGENT"
	case "URGENT":
		return "ROUTINE"
	default:
		return urgency // ROUTINE stays ROUTINE
	}
}
```

Add `"time"` import.

- [ ] **Step 2: Test the seasonal variant**

Add to `trajectory_card_rules_test.go`:

```go
func TestTrajectoryCardsWithSeasonalContext_GlucoseDowngraded(t *testing.T) {
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)

	// Build a seasonal context with Diwali active.
	tmp := t.TempDir()
	path := filepath.Join(tmp, "cal.yaml")
	yaml := `
windows:
  - name: diwali
    start: "2026-11-04"
    end: "2026-11-14"
    affected_domains: [GLUCOSE]
    mode: DOWNGRADE_URGENCY
    rationale: "festival eating"
`
	os.WriteFile(path, []byte(yaml), 0644)
	seasonalCtx, _ := NewSeasonalContext("india", path)

	// A glucose RAPID_DECLINE card should be downgraded URGENT → ROUTINE.
	traj := &dtModels.DecomposedTrajectory{
		CompositeTrend: dtModels.TrendDeclining,
		DomainSlopes: map[dtModels.MHRIDomain]dtModels.DomainSlope{
			dtModels.DomainGlucose:    {Domain: dtModels.DomainGlucose, Trend: dtModels.TrendRapidDeclining, SlopePerDay: -1.5, Confidence: dtModels.ConfidenceHigh, R2: 0.9, StartScore: 70, EndScore: 50},
			dtModels.DomainCardio:     {Trend: dtModels.TrendStable},
			dtModels.DomainBodyComp:   {Trend: dtModels.TrendStable},
			dtModels.DomainBehavioral: {Trend: dtModels.TrendStable},
		},
	}

	cards := EvaluateTrajectoryCardsWithSeasonalContext(traj, seasonalCtx, now)

	found := false
	for _, c := range cards {
		if c.CardType == "DOMAIN_RAPID_DECLINE" && c.Domain == "GLUCOSE" {
			found = true
			if c.Urgency != "ROUTINE" {
				t.Errorf("expected glucose card downgraded to ROUTINE, got %s", c.Urgency)
			}
			if !contains(c.Rationale, "seasonal context") {
				t.Errorf("expected rationale to mention seasonal context, got: %s", c.Rationale)
			}
		}
	}
	if !found {
		t.Error("expected glucose rapid decline card to still be present (downgraded)")
	}
}

func TestTrajectoryCardsWithSeasonalContext_NilContextPassthrough(t *testing.T) {
	traj := &dtModels.DecomposedTrajectory{
		DomainSlopes: map[dtModels.MHRIDomain]dtModels.DomainSlope{
			dtModels.DomainGlucose:    {Domain: dtModels.DomainGlucose, Trend: dtModels.TrendRapidDeclining, SlopePerDay: -1.5, Confidence: dtModels.ConfidenceHigh},
			dtModels.DomainCardio:     {Trend: dtModels.TrendStable},
			dtModels.DomainBodyComp:   {Trend: dtModels.TrendStable},
			dtModels.DomainBehavioral: {Trend: dtModels.TrendStable},
		},
	}

	without := EvaluateTrajectoryCards(traj)
	with := EvaluateTrajectoryCardsWithSeasonalContext(traj, nil, time.Now())

	if len(without) != len(with) {
		t.Errorf("expected nil context to produce identical output: %d vs %d cards", len(without), len(with))
	}
}

// helper for substring check (already defined in domain_divergence_test.go as containsSubstring)
func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
```

(Note: if `contains` is already defined in the package, omit the helper definition here.)

- [ ] **Step 3: Run tests**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go test ./internal/services/ -run "TestTrajectoryCards" -v
```

Expected: All trajectory card tests pass including the 2 new seasonal tests.

- [ ] **Step 4: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/trajectory_card_rules.go backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/trajectory_card_rules_test.go
git commit -m "feat(kb23): seasonal-aware trajectory card evaluation

EvaluateTrajectoryCardsWithSeasonalContext takes a SeasonalContext and
filters/downgrades cards based on active seasonal windows. Single-domain
cards (DOMAIN_RAPID_DECLINE, DOMAIN_DIVERGENCE, DOMAIN_CATEGORY_CROSSING)
can be downgraded or suppressed. CONCORDANT_DETERIORATION is conservative
and not seasonally suppressed (multi-domain risk). Backward compat:
EvaluateTrajectoryCards (no context) still works unchanged."
```

---

### Task 21: Wire SeasonalContext into composite handler

**Files:**
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/api/composite_handlers.go`
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/api/server.go`

- [ ] **Step 1: Add SeasonalContext to Server**

In `server.go`:
```go
seasonalContext *services.SeasonalContext
```

In `InitServices`:
```go
seasonalCalendarPath := s.cfg.SeasonalCalendarPath // e.g. "market-configs/india/seasonal_calendar.yaml"
ctx, err := services.NewSeasonalContext(s.cfg.Market, seasonalCalendarPath)
if err != nil {
	s.logger.Warn("failed to load seasonal calendar, no suppression will apply", zap.Error(err))
	ctx, _ = services.NewSeasonalContext(s.cfg.Market, "") // empty context
}
s.seasonalContext = ctx
```

Add `Market` and `SeasonalCalendarPath` fields to the Config struct.

- [ ] **Step 2: Update composite handler to pass seasonal context**

In `composite_handlers.go`, find the trajectory card evaluation:
```go
for _, card := range services.EvaluateTrajectoryCards(traj) {
```

Replace with:
```go
for _, card := range services.EvaluateTrajectoryCardsWithSeasonalContext(traj, s.seasonalContext, time.Now()) {
```

Add `"time"` import if not already present.

- [ ] **Step 3: Build and test**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go build ./... && go test ./internal/services/ -count=1 2>&1 | tail -5
```

Expected: Build clean, all tests pass.

- [ ] **Step 4: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/api/composite_handlers.go backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/api/server.go
git commit -m "feat(kb23): wire seasonal context into composite synthesis

Server initializes SeasonalContext from market-configs/<market>/seasonal_calendar.yaml.
Composite handler passes the context to EvaluateTrajectoryCardsWithSeasonalContext
so trajectory cards are downgraded/suppressed during known seasonal windows
(Diwali, Ramadan, etc). Missing calendar file is non-fatal — empty context
means no suppression."
```

---

### Task 22: Verify end-to-end seasonal suppression

**Files:**
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/trajectory_integration_test.go`

- [ ] **Step 1: Add a seasonal integration test**

Append to the existing integration test file:

```go
func TestIntegration_SeasonalSuppression_Diwali(t *testing.T) {
	// Compute a trajectory with glucose declining
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	points := []dtModels.DomainTrajectoryPoint{
		{Timestamp: now.Add(-13 * 24 * time.Hour), CompositeScore: 70, GlucoseScore: 70, CardioScore: 65, BodyCompScore: 65, BehavioralScore: 70},
		{Timestamp: now.Add(-7 * 24 * time.Hour),  CompositeScore: 60, GlucoseScore: 55, CardioScore: 65, BodyCompScore: 65, BehavioralScore: 70},
		{Timestamp: now.Add(-1 * 24 * time.Hour),  CompositeScore: 50, GlucoseScore: 40, CardioScore: 65, BodyCompScore: 65, BehavioralScore: 70},
	}
	trajectory := dtModels.Compute("e2e-diwali", points)

	// Build a Diwali seasonal context
	tmp := t.TempDir()
	path := filepath.Join(tmp, "cal.yaml")
	yaml := `
windows:
  - name: diwali
    start: "2026-11-04"
    end: "2026-11-14"
    affected_domains: [GLUCOSE]
    mode: DOWNGRADE_URGENCY
    rationale: "festival eating"
`
	os.WriteFile(path, []byte(yaml), 0644)
	seasonalCtx, _ := NewSeasonalContext("india", path)

	// Without seasonal context → glucose rapid decline card at URGENT
	cardsWithout := EvaluateTrajectoryCardsWithSeasonalContext(&trajectory, nil, now)
	hadUrgentGlucose := false
	for _, c := range cardsWithout {
		if c.CardType == "DOMAIN_RAPID_DECLINE" && c.Domain == "GLUCOSE" && c.Urgency == "URGENT" {
			hadUrgentGlucose = true
		}
	}
	if !hadUrgentGlucose {
		t.Fatal("baseline check failed: expected URGENT glucose card without seasonal context")
	}

	// With Diwali context → glucose rapid decline card downgraded to ROUTINE
	cardsWith := EvaluateTrajectoryCardsWithSeasonalContext(&trajectory, seasonalCtx, now)
	hadDowngradedGlucose := false
	for _, c := range cardsWith {
		if c.CardType == "DOMAIN_RAPID_DECLINE" && c.Domain == "GLUCOSE" && c.Urgency == "ROUTINE" {
			hadDowngradedGlucose = true
		}
	}
	if !hadDowngradedGlucose {
		t.Error("expected glucose card downgraded to ROUTINE during Diwali")
	}
}
```

Add `"os"`, `"path/filepath"`, and `"time"` imports if not present.

- [ ] **Step 2: Run tests**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go test ./internal/services/ -run "TestIntegration_SeasonalSuppression" -v
```

Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/trajectory_integration_test.go
git commit -m "test(kb23): integration test for Diwali seasonal suppression

Compute a glucose-declining trajectory, evaluate cards with and without
a Diwali seasonal context, verify the URGENT card is downgraded to
ROUTINE during the Diwali window. Validates the full Compute → Evaluate
→ Suppress pipeline through the public facade."
```

---

## Slice P2.6: KB-23 Observability + Grafana (Tasks 23-26)

### Task 23: KB-23 trajectory card metrics collector

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/metrics/trajectory_card_metrics.go`

- [ ] **Step 1: Create the metrics file**

```go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// TrajectoryCardMetrics groups Prometheus metrics for KB-23 trajectory card
// evaluation and KB-26 fetch outcomes.
type TrajectoryCardMetrics struct {
	CardEvaluatedTotal       *prometheus.CounterVec
	KB26FetchDuration        prometheus.Histogram
	KB26FetchTotal           *prometheus.CounterVec
}

// NewTrajectoryCardMetrics registers all metrics with the global registry.
func NewTrajectoryCardMetrics() *TrajectoryCardMetrics {
	return &TrajectoryCardMetrics{
		CardEvaluatedTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "kb23_trajectory_card_evaluated_total",
			Help: "Trajectory cards generated by type and urgency",
		}, []string{"card_type", "urgency"}),
		KB26FetchDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "kb23_kb26_trajectory_fetch_duration_ms",
			Help:    "Latency of KB-26 trajectory fetch in milliseconds",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500},
		}),
		KB26FetchTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "kb23_kb26_trajectory_fetch_total",
			Help: "KB-26 trajectory fetch outcomes from KB-23",
		}, []string{"status"}), // ok | not_found | error
	}
}
```

- [ ] **Step 2: Build and commit**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go build ./internal/metrics/
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/metrics/trajectory_card_metrics.go
git commit -m "feat(kb23): TrajectoryCardMetrics Prometheus collector

3 metrics: card_evaluated_total{card_type,urgency},
kb26_trajectory_fetch_duration_ms histogram, kb26_trajectory_fetch_total{status}."
```

---

### Task 24: Instrument KB26TrajectoryClient

**Files:**
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/kb26_trajectory_client.go`

- [ ] **Step 1: Add metrics field to client**

```go
type KB26TrajectoryClient struct {
	baseURL string
	http    *http.Client
	logger  *zap.Logger
	metrics *metrics.TrajectoryCardMetrics // optional, nil-safe
}

func NewKB26TrajectoryClient(baseURL string, timeout time.Duration, logger *zap.Logger, metrics *metrics.TrajectoryCardMetrics) *KB26TrajectoryClient {
	return &KB26TrajectoryClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: timeout},
		logger:  logger,
		metrics: metrics,
	}
}
```

- [ ] **Step 2: Instrument GetTrajectory**

```go
func (c *KB26TrajectoryClient) GetTrajectory(ctx context.Context, patientID string) (*dtModels.DecomposedTrajectory, error) {
	start := time.Now()
	defer func() {
		if c.metrics != nil {
			c.metrics.KB26FetchDuration.Observe(float64(time.Since(start).Milliseconds()))
		}
	}()

	// ... existing fetch logic ...

	// On status==404:
	if c.metrics != nil {
		c.metrics.KB26FetchTotal.WithLabelValues("not_found").Inc()
	}
	return nil, nil

	// On INSUFFICIENT_DATA in body:
	if c.metrics != nil {
		c.metrics.KB26FetchTotal.WithLabelValues("not_found").Inc()
	}
	return nil, nil

	// On success:
	if c.metrics != nil {
		c.metrics.KB26FetchTotal.WithLabelValues("ok").Inc()
	}
	return &result, nil

	// On transport error:
	if c.metrics != nil {
		c.metrics.KB26FetchTotal.WithLabelValues("error").Inc()
	}
	return nil, err
}
```

- [ ] **Step 3: Update server init**

In `server.go`, the existing `NewKB26TrajectoryClient` call needs the metrics arg:
```go
s.trajectoryCardMetrics = metrics.NewTrajectoryCardMetrics()
s.kb26TrajectoryClient = services.NewKB26TrajectoryClient(
	s.cfg.KB26URL,
	s.cfg.KB26Timeout(),
	s.log,
	s.trajectoryCardMetrics,
)
```

- [ ] **Step 4: Build and test**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go build ./... && go test ./internal/services/ -count=1 2>&1 | tail -5
```

Expected: Build clean, tests pass.

- [ ] **Step 5: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/kb26_trajectory_client.go backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/api/server.go
git commit -m "feat(kb23): instrument KB26TrajectoryClient with Prometheus metrics

GetTrajectory now records fetch duration histogram and emits
kb23_kb26_trajectory_fetch_total{status} for each outcome (ok, not_found, error).
Metrics are nil-safe so tests pass nil and the production server passes
a real collector."
```

---

### Task 25: Instrument card evaluation in composite handler

**Files:**
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/api/composite_handlers.go`

- [ ] **Step 1: Increment card_evaluated_total per card built**

Inside the trajectory card loop in `handleCompositeSynthesize`:

```go
for _, card := range services.EvaluateTrajectoryCardsWithSeasonalContext(traj, s.seasonalContext, time.Now()) {
	if s.trajectoryCardMetrics != nil {
		s.trajectoryCardMetrics.CardEvaluatedTotal.WithLabelValues(card.CardType, card.Urgency).Inc()
	}
	dc := services.BuildTrajectoryDecisionCard(card, patientID)
	if err := s.db.DB.Create(dc).Error; err != nil {
		s.logger.Warn("failed to persist trajectory card", zap.Error(err))
	}
}
```

- [ ] **Step 2: Build and commit**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go build ./...
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/api/composite_handlers.go
git commit -m "feat(kb23): instrument trajectory card evaluation count

Increments kb23_trajectory_card_evaluated_total{card_type,urgency} for
each card generated by EvaluateTrajectoryCardsWithSeasonalContext.
Lets shadow deployment monitor card volume by type and urgency over time."
```

---

### Task 26: Grafana dashboard JSON

**Files:**
- Create: `observability/dashboards/mhri_trajectory.json`

- [ ] **Step 1: Create the dashboard file**

```json
{
  "title": "MHRI Domain-Decomposed Trajectory",
  "uid": "mhri-trajectory",
  "schemaVersion": 38,
  "version": 1,
  "panels": [
    {
      "id": 1,
      "title": "Compute Latency p50/p95/p99",
      "type": "timeseries",
      "targets": [
        {"expr": "histogram_quantile(0.50, rate(kb26_trajectory_compute_duration_ms_bucket[5m]))", "legendFormat": "p50"},
        {"expr": "histogram_quantile(0.95, rate(kb26_trajectory_compute_duration_ms_bucket[5m]))", "legendFormat": "p95"},
        {"expr": "histogram_quantile(0.99, rate(kb26_trajectory_compute_duration_ms_bucket[5m]))", "legendFormat": "p99"}
      ],
      "gridPos": {"x": 0, "y": 0, "w": 12, "h": 8}
    },
    {
      "id": 2,
      "title": "Concordant Deterioration Rate by Domain Count",
      "type": "timeseries",
      "targets": [
        {"expr": "rate(kb26_trajectory_concordant_deterioration_total[5m])", "legendFormat": "{{domains_count}} domains"}
      ],
      "gridPos": {"x": 12, "y": 0, "w": 12, "h": 8}
    },
    {
      "id": 3,
      "title": "Divergence Pair Heatmap",
      "type": "heatmap",
      "targets": [
        {"expr": "rate(kb26_trajectory_divergence_total[15m])", "legendFormat": "{{improving_domain}} ↑ / {{declining_domain}} ↓"}
      ],
      "gridPos": {"x": 0, "y": 8, "w": 12, "h": 8}
    },
    {
      "id": 4,
      "title": "Behavioral Leading Indicator Fires",
      "type": "timeseries",
      "targets": [
        {"expr": "rate(kb26_trajectory_leading_indicator_total[5m])", "legendFormat": "lagging: {{lagging_domain}}"}
      ],
      "gridPos": {"x": 12, "y": 8, "w": 12, "h": 8}
    },
    {
      "id": 5,
      "title": "KB-23 → KB-26 Fetch Success Rate",
      "type": "stat",
      "targets": [
        {"expr": "sum(rate(kb23_kb26_trajectory_fetch_total{status=\"ok\"}[5m])) / sum(rate(kb23_kb26_trajectory_fetch_total[5m]))"}
      ],
      "gridPos": {"x": 0, "y": 16, "w": 12, "h": 8}
    },
    {
      "id": 6,
      "title": "Persistence Failure Rate",
      "type": "stat",
      "targets": [
        {"expr": "sum(rate(kb26_trajectory_persist_total{result=\"fail\"}[5m])) / sum(rate(kb26_trajectory_persist_total[5m]))"}
      ],
      "gridPos": {"x": 12, "y": 16, "w": 12, "h": 8}
    }
  ]
}
```

- [ ] **Step 2: Commit**

```bash
git add observability/dashboards/mhri_trajectory.json
git commit -m "dashboard: Grafana JSON for MHRI trajectory observability

6 panels: compute latency p50/p95/p99, concordant deterioration rate
by domain count, divergence pair heatmap, leading indicator fires,
KB-23→KB-26 fetch success rate, persistence failure rate.
Imports against any Prometheus datasource."
```

---

## Final Regression

### Task 27: Full regression across both services

- [ ] **Step 1: KB-26 build + test + vet**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go build ./... && go test ./... -count=1 && go vet ./...
```

Expected: All clean.

- [ ] **Step 2: KB-23 build + test + vet**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go build ./... && go test ./... -count=1 && go vet ./...
```

Expected: All clean.

- [ ] **Step 3: Verify Phase 0+1 tests still pass**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/services/ -run "TestDomainTrajectory|TestDomainCategory|TestDetectDivergence|TestDivergence_" -v 2>&1 | grep -E "^(--- PASS|--- FAIL|PASS|FAIL|ok)"
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go test ./internal/services/ -run "TestTrajectoryCards|TestBuildTrajectoryDecisionCard|TestBuildMaskedHTNDecisionCard|TestIntegration|TestSeasonalContext" -v 2>&1 | grep -E "^(--- PASS|--- FAIL|PASS|FAIL|ok)"
```

Expected: All tests PASS — Phase 0+1 (12 + 6 = 18) plus Phase 2 additions (4 seasonal + 1 cluster regression + 3 publisher + 2 config = 10 new tests).

- [ ] **Step 4: Final summary commit (no file changes)**

```bash
git commit --allow-empty -m "feat: MHRI domain-decomposed trajectory Phase 2 complete

Slice P2.1: KB-26 observability — 7 Prometheus metrics + nil-safe injection
Slice P2.2: Config refactor — TrajectoryEngine struct, YAML loader, all
            tests target the new struct API
Slice P2.3: Numerical stability — two-pass ssTot in trajectory engine,
            regression test for clustered scores
Slice P2.4: Seasonal adjustment — SeasonalContext service, India calendar,
            card rule integration with DOWNGRADE_URGENCY/SUPPRESS modes
Slice P2.5: Kafka publisher — DomainTrajectoryComputed v1 events emitted
            to kb26.domain_trajectory.v1, Module 13 consumer contract docs
Slice P2.6: KB-23 observability — 3 Prometheus metrics, Grafana dashboard

Total: 12 created files, 9 modified files, 10 new tests, 0 regressions.
Phase 0+1 tests (18) all still pass."
```

---

## Plan Summary

| Slice | Tasks | Key Deliverables |
|-------|-------|-----------------|
| P2.2: Config refactor | 1-6 | TrajectoryEngine struct, YAML config, all tests use struct |
| P2.3: Numerical stability | 7-9 | Two-pass ssTot, regression tests for clustered scores |
| P2.1: KB-26 observability | 10-13 | 7 Prometheus metrics, instrumentation, persist counter |
| P2.5: Kafka publisher | 14-17 | DomainTrajectoryComputed events, consumer contract docs |
| P2.4: Seasonal adjustment | 18-22 | SeasonalContext, India calendar, card rule integration |
| P2.6: KB-23 observability + dashboard | 23-26 | 3 Prometheus metrics, Grafana JSON |
| Final regression | 27 | Full build + test + vet across both services |

**Total: 27 tasks, 12 created files, 9 modified files, 10 new tests on top of Phase 0+1's 22 tests**

## Notes on Phase 2 dependencies

- **Tasks 1-6 must land first** because they change the engine API surface that all other tasks instrument or extend.
- **Tasks 7-9 (numerical stability)** are validated by Tasks 3-6 (the new engine has the fix); the explicit tests in this slice are regression guards.
- **Tasks 10-13 (KB-26 observability)** require Task 6 complete because they extend the engine constructor signature.
- **Tasks 14-17 (Kafka publisher)** also extend the engine constructor; should land after KB-26 observability to avoid double-touching the constructor.
- **Tasks 18-22 (seasonal)** are independent of KB-26 changes — can land in parallel with Tasks 14-17 if there are multiple developers.
- **Tasks 23-26 (KB-23 observability)** depend on Phase 1 wiring (composite handler) being intact and on Task 24 extending the trajectory client constructor.

## Ship strategy

Each slice is independently committable and reversible. For shadow-deployment safety:
- Slices P2.1, P2.3, P2.6 are pure additions (metrics, regression tests, dashboard) — can deploy immediately.
- Slice P2.2 (config refactor) is a behavior-preserving refactor — should deploy with a canary to validate threshold loading.
- Slice P2.4 (seasonal) changes card output for Indian markets — should deploy with a feature flag and validate against shadow data first.
- Slice P2.5 (Kafka publisher) requires the Kafka broker to be reachable; deploy after broker provisioning is confirmed.
