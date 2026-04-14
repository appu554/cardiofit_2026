# MHRI Domain-Decomposed Trajectory — Phase 2 Design Spec

**Date**: 2026-04-14
**Status**: Approved
**Scope**: KB-26 (Metabolic Digital Twin) + KB-23 (Decision Cards) + market-configs
**Depends on**: Phase 0+1 completed in `2026-04-11-mhri-domain-decomposed-trajectory-design.md`

## Problem

Phase 0 shipped the MHRI domain-decomposed trajectory engine end-to-end. Phase 1 wired it into the composite synthesis HTTP handler alongside masked HTN. The system is runtime-reachable but not production-hardened.

Phase 2 closes five gaps identified during the Phase 0+1 code review cycles and the clinical review of the implementation audit:

1. **No observability** — the engine computes trajectories but emits no metrics. Concordant deterioration count, divergence detection rate, leading indicator fire rate, compute latency, and KB-26→KB-23 fetch success rate are all invisible. Shadow deployment cannot be evaluated without these.

2. **Thresholds hardcoded in Go** — the canonical `domain_trajectory_thresholds.yaml` is a reference document only. Trend classification, divergence rate, confidence thresholds, and domain weights are all package-level constants in `mri_domain_trajectory.go`. Market-specific tuning requires a code change, making India seasonal adjustment and Australia Indigenous benchmarking impossible to iterate without redeployment.

3. **Numerical stability bug in OLS R² computation** — `ssTot = sumY2 - n*meanY*meanY` suffers catastrophic cancellation when scores cluster tightly (e.g., a stable domain with scores [70.1, 70.3, 69.8, 70.2, 70.0]). Subtraction of two nearly-equal large numbers produces a result dominated by floating-point error, making R² unreliable. Unreliable R² propagates to unreliable confidence classification (HIGH/MODERATE/LOW), which propagates to unreliable clinical action. The same bug exists in the sibling `egfr_trajectory.go`.

4. **Seasonal patterns generate false-positive clinical alerts** — during Diwali in India, patients' body composition and glucose domains worsen predictably due to festival eating patterns. The current engine flags this as concordant deterioration and emits IMMEDIATE urgency cards. Similar issues exist for Ramadan (altered eating), Indian heat season (cardio domain from dehydration), and Australian harvest-season rural engagement drops. Without seasonal suppression, physician trust erodes during the periods when card volume is highest.

5. **Module 13 Flink state sync has no event source** — the original E2E trace with patient Rajesh Kumar showed Module 13's `domain_velocities` map permanently empty because no service publishes per-domain velocity events. Phase 0 built the data (per-domain slopes with trend classification) but never wired the event publication. Module 4's cross-domain CEP patterns cannot fire without this data flowing through Flink.

**Evidence base**:
- Phase 0 code review flagged items #3 and #2 as "Important" and deferred them citing scope
- Phase 0+1 audit flagged item #1 as operational-readiness gap before shadow deployment
- Spec's "Future Work" section explicitly called out item #4 (India seasonal) as a known false-positive source
- Original E2E clinical debugging of patient Rajesh Kumar surfaced item #5 as the root cause of Module 13's missing cross-domain CEP signals

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Config-driven threshold architecture | Convert `ComputeDecomposedTrajectory` from free function to `TrajectoryEngine` struct method holding config | Idiomatic Go; few existing callers (one handler + tests); testing cleaner than package-level vars set at startup |
| Config loading | Parse `domain_trajectory_thresholds.yaml` in KB-26 config package alongside existing config, expose typed `TrajectoryThresholds` struct | Matches existing KB-26 config pattern (viper + struct); keeps YAML as single source of truth |
| Numerical stability fix | Two-pass ssTot: compute mean first, then `Σ(yᵢ − ȳ)²` | Eliminates catastrophic cancellation; 20-line fix applied to both `mri_domain_trajectory.go` and `egfr_trajectory.go` in one commit |
| India seasonal mechanism | New `seasonal_calendar.yaml` per market + `SeasonalContext` checker + suppression flag on `ConcordantDeterioration` during known windows | Mechanism is general (works for Diwali, Ramadan, heat season, harvest); calendar is data-driven so clinical review can tune dates without code changes |
| Module 13 Kafka event | KB-26 publishes `DomainTrajectoryComputed` event to Kafka topic `kb26.domain_trajectory.v1` after each successful compute | Producer-only scope; Module 13 Flink consumer implementation is a separate task in a different tech stack (Java) |
| Observability metric library | Prometheus client already in both services via existing `metrics` packages; add new `TrajectoryMetrics` collector | Matches masked HTN observability pattern; no new dependencies |
| Phase 2 migration approach | Each item is its own committable slice — ship incrementally behind feature flags where needed | Reduces risk; item 1 (observability) has no behavior change; items 2-5 can land independently |

## Architecture

```
                             ┌─ metrics/trajectory_metrics.go (new)
                             │    - compute_duration_ms
                             │    - concordant_deterioration_total
                             │    - divergence_detected_total{pair}
                             │    - leading_indicator_total{lagging_domain}
                             │    - category_crossing_total{from,to}
                             │
KB-26 TrajectoryEngine ──────┼─ config/trajectory_config.go (new)
                             │    - TrajectoryThresholds struct
                             │    - loaded from domain_trajectory_thresholds.yaml
                             │
                             ├─ services/trajectory_engine.go (refactor of mri_domain_trajectory.go)
                             │    - Compute(patientID, points) method
                             │    - Config injected at NewTrajectoryEngine()
                             │    - numerical stability fix in computeOLSWithR2
                             │
                             ├─ services/seasonal_context.go (new)
                             │    - IsInSeasonalWindow(patientMarket, ts) (bool, *SeasonalWindow)
                             │    - loaded from seasonal_calendar.yaml per market
                             │
                             └─ services/trajectory_publisher.go (new)
                                  - Publishes DomainTrajectoryComputed event
                                  - Kafka topic: kb26.domain_trajectory.v1


KB-23 v4 wiring (existing) ──┬─ metrics/trajectory_card_metrics.go (new)
                             │    - card_evaluated_total{card_type,urgency}
                             │    - kb26_trajectory_fetch_duration_ms
                             │    - kb26_trajectory_fetch_total{status}
                             │
                             └─ composite_handlers.go (instrumented)


market-configs/
  shared/
    domain_trajectory_thresholds.yaml (existing — now actually loaded)
  india/
    seasonal_calendar.yaml (new)
  australia/
    seasonal_calendar.yaml (new — empty seed)


Kafka: kb26.domain_trajectory.v1 topic
        consumer: Module 13 Flink state-sync (out of scope, documented contract only)
```

## Item 1: Observability

**Goal**: Expose 10 Prometheus metrics across KB-26 and KB-23 so Phase 0 shadow deployment can be evaluated.

### KB-26 metrics (in `internal/metrics/trajectory_metrics.go`, new file)

| Metric | Type | Labels | Purpose |
|--------|------|--------|---------|
| `kb26_trajectory_compute_duration_ms` | Histogram | — | Trajectory compute latency per request |
| `kb26_trajectory_concordant_deterioration_total` | Counter | `domains_count` | Concordant multi-domain decline fires |
| `kb26_trajectory_divergence_total` | Counter | `improving_domain`, `declining_domain` | Divergence pair detection |
| `kb26_trajectory_leading_indicator_total` | Counter | `lagging_domain` | Behavioral leading indicator fires |
| `kb26_trajectory_domain_crossing_total` | Counter | `domain`, `direction` | Domain category crossing events |
| `kb26_trajectory_insufficient_data_total` | Counter | — | Requests blocked by <2 data points |
| `kb26_trajectory_persist_total` | Counter | `result` (ok/fail) | History table persistence outcome |

### KB-23 metrics (in `internal/metrics/trajectory_card_metrics.go`, new file)

| Metric | Type | Labels | Purpose |
|--------|------|--------|---------|
| `kb23_trajectory_card_evaluated_total` | Counter | `card_type`, `urgency` | Per-type card generation |
| `kb23_kb26_trajectory_fetch_duration_ms` | Histogram | — | KB-26 fetch latency from KB-23 side |
| `kb23_kb26_trajectory_fetch_total` | Counter | `status` (ok/404/error) | Fetch outcome distribution |

### Grafana dashboard

A single new dashboard JSON file at `observability/dashboards/mhri_trajectory.json` with 6 panels:
1. Compute latency p50/p95/p99
2. Concordant deterioration rate by domain count
3. Divergence pair heatmap
4. Leading indicator fires over time
5. KB-23 fetch success rate
6. Persistence failure rate

## Item 2: Config-Driven Thresholds

**Goal**: Convert all trajectory thresholds from Go constants to values loaded from `domain_trajectory_thresholds.yaml` at service startup, so market-specific tuning requires only a config change.

### New `TrajectoryThresholds` struct (in `internal/config/trajectory_config.go`)

```go
type TrajectoryThresholds struct {
	Trend             TrendThresholds
	Divergence        DivergenceThresholds
	LeadingIndicator  LeadingIndicatorThresholds
	Concordant        ConcordantThresholds
	Driver            DriverThresholds
	RSquared          R2Thresholds
	CategoryBoundaries CategoryBoundaries
}

type TrendThresholds struct {
	RapidImproving float64 // default 1.0
	Improving      float64 // default 0.3
	Declining      float64 // default -0.3
	RapidDeclining float64 // default -1.0
}

type DivergenceThresholds struct {
	MinDivergenceRate float64 // default 0.5
	MinImprovingSlope float64 // default 0.3
	MinDecliningSlope float64 // default -0.3
}

type LeadingIndicatorThresholds struct {
	MinDataPoints            int     // default 5
	MinBehavioralDeclineSlope float64 // default -0.5
}

type ConcordantThresholds struct {
	MinDomainsDeclining int     // default 2
	MinSlopePerDomain   float64 // default -0.3
}

type DriverThresholds struct {
	MinContributionPct float64                     // default 40.0
	WeightMap          map[MHRIDomain]float64      // default {G:0.35, C:0.25, BC:0.25, B:0.15}
}

type R2Thresholds struct {
	High     float64 // default 0.5
	Moderate float64 // default 0.25
}

type CategoryBoundaries struct {
	Optimal  float64 // default 70
	Mild     float64 // default 55
	Moderate float64 // default 40
}
```

### Refactor: `ComputeDecomposedTrajectory` → `TrajectoryEngine.Compute`

Current shape (free function):
```go
func ComputeDecomposedTrajectory(patientID string, points []models.DomainTrajectoryPoint) models.DecomposedTrajectory
```

New shape (struct method):
```go
type TrajectoryEngine struct {
	thresholds TrajectoryThresholds
	metrics    *metrics.TrajectoryMetrics // optional, nil in tests
	publisher  TrajectoryPublisher        // optional, nil in tests
}

func NewTrajectoryEngine(thresholds TrajectoryThresholds, opts ...EngineOption) *TrajectoryEngine
func (e *TrajectoryEngine) Compute(patientID string, points []models.DomainTrajectoryPoint) models.DecomposedTrajectory
```

### Callers to update
- `internal/api/domain_trajectory_handlers.go` — construct engine at Server init, call `s.trajectoryEngine.Compute(...)`
- `internal/api/server.go` — add `trajectoryEngine` field, initialize in `InitServices`
- `internal/services/mri_domain_trajectory_test.go` — construct engine in test setup with default thresholds
- `pkg/trajectory/models.go` — `Compute` function variable still exists but now wraps `NewTrajectoryEngine(DefaultThresholds).Compute` for backward compat with the KB-23 integration test

### Default thresholds helper
`DefaultTrajectoryThresholds()` returns a `TrajectoryThresholds` with the same values currently hardcoded, so tests and code that doesn't need config overrides can use defaults.

### Loading YAML at startup
`internal/config/trajectory_config.go` exposes `LoadTrajectoryThresholds(path string) (TrajectoryThresholds, error)` which the main config loader calls and threads through to the server struct.

## Item 3: Numerical Stability Fix

**Goal**: Replace the single-pass `ssTot = sumY2 − n·meanY²` formula with the two-pass `Σ(yᵢ − ȳ)²` form in both `mri_domain_trajectory.go` and `egfr_trajectory.go`.

### The bug (illustrated)

For scores [70.1, 70.3, 69.8, 70.2, 70.0]:
- `meanY = 70.08`
- `sumY2 = 24556.18`
- `n*meanY*meanY = 5 × 4911.2064 = 24556.032`
- `ssTot = 24556.18 − 24556.032 = 0.148`

True value (two-pass): `Σ(yᵢ − ȳ)² = 0.148` — happens to match here because the numbers aren't *that* close.

But for scores [70.0001, 70.0002, 70.0000, 70.0003, 70.0000]:
- True `ssTot` ≈ 6.8e-8
- Shortcut `ssTot` after catastrophic cancellation ≈ −1.2e-11 (can even go negative due to rounding!)
- Code path: `ssTot > 1e-10` is false → R² stays at 0 → confidence classified as LOW
- Reality: the scores are rock-stable at 70.00 ± 0.0003; a stable trend should have R² near 1 (not 0)

A stable domain falsely reported as LOW confidence means the monitoring pillar skips it. A real noisy trend falsely reported as HIGH confidence means the system acts on noise. Both are clinical failure modes.

### The fix

```go
// Before (catastrophic cancellation risk)
meanY := sumY / n
ssTot := sumY2 - n*meanY*meanY

// After (numerically stable two-pass)
meanY := sumY / n
ssTot := 0.0
for _, y := range scores {
	delta := y - meanY
	ssTot += delta * delta
}
```

This adds one pass over the scores slice — O(n) extra work for typically n ≤ 10 points. Cost: negligible. Benefit: R² correctness.

### Apply to both files

Both `computeOLSWithR2` in `mri_domain_trajectory.go` and `ComputeEGFRTrajectory` in `egfr_trajectory.go` have the same formula. Fix both in one commit for consistency.

### Regression test

New test `TestComputeOLSWithR2_ClusteredScores` feeds scores that would trigger catastrophic cancellation under the old formula and asserts R² ≥ 0.9 (real value is near 1).

## Item 4: India Seasonal Adjustment

**Goal**: During known seasonal windows (festivals, Ramadan, heat season), suppress or de-urgency trajectory cards for domains the season is expected to affect, preventing false-positive clinical alerts.

### Seasonal calendar YAML schema

New file: `market-configs/india/seasonal_calendar.yaml`

```yaml
# India seasonal calendar for trajectory alert suppression.
# Each window names the season, its affected domains, the suppression mode,
# and a date range. Dates are ISO-8601 day-of-year ranges (year-agnostic)
# or explicit date ranges for year-specific festivals.

windows:
  - name: diwali
    start: "2026-11-04"  # year-specific; clinical review updates annually
    end: "2026-11-14"
    affected_domains: [GLUCOSE, BODY_COMP]
    mode: DOWNGRADE_URGENCY   # DOWNGRADE_URGENCY | SUPPRESS
    rationale: "Festival eating patterns predictably worsen glucose and body composition for 7-10 days"

  - name: ramadan
    start: "2026-03-17"
    end: "2026-04-15"
    affected_domains: [GLUCOSE, BEHAVIORAL]
    mode: DOWNGRADE_URGENCY
    rationale: "Altered eating windows affect glycaemic control and monitoring patterns"

  - name: summer_heat
    start_doy: 121  # May 1
    end_doy: 181    # June 30
    affected_domains: [CARDIO]
    mode: DOWNGRADE_URGENCY
    rationale: "Extreme heat causes dehydration and volume depletion affecting BP readings"
```

### Suppression mechanism

Two modes:
- **DOWNGRADE_URGENCY**: IMMEDIATE → URGENT → ROUTINE (one step softer), card still fires, rationale includes seasonal context ("during Diwali window")
- **SUPPRESS**: card is not generated at all, but the trajectory is still computed and persisted for history

### `SeasonalContext` service (new)

```go
type SeasonalContext struct {
	market  string // "india", "australia", ...
	windows []SeasonalWindow
}

type SeasonalWindow struct {
	Name             string
	Start, End       time.Time // absolute dates for year-specific windows
	StartDOY, EndDOY int       // day-of-year for recurring windows (0 if not set)
	AffectedDomains  []models.MHRIDomain
	Mode             string // DOWNGRADE_URGENCY | SUPPRESS
	Rationale        string
}

func NewSeasonalContext(market string, calendarPath string) (*SeasonalContext, error)

// ActiveWindows returns all seasonal windows active at the given timestamp.
func (s *SeasonalContext) ActiveWindows(ts time.Time) []SeasonalWindow

// ShouldSuppress returns (suppress, downgrade, rationale) for a given domain at a given time.
func (s *SeasonalContext) ShouldSuppress(domain models.MHRIDomain, ts time.Time) (suppress bool, downgrade bool, rationale string)
```

### Integration point: KB-23 trajectory card rules

`EvaluateTrajectoryCards` takes an optional `*SeasonalContext` (via a new `EvaluateTrajectoryCardsWithContext` variant — keep old signature for backward compat). During card generation:

```go
for _, ds := range traj.DomainSlopes {
    if ds.Trend == TrendRapidDeclining && ds.Confidence != ConfidenceLow {
        suppress, downgrade, rationale := seasonalCtx.ShouldSuppress(ds.Domain, time.Now())
        if suppress {
            continue // do not generate card
        }
        card := buildDomainRapidDeclineCard(ds, traj.CompositeSlope)
        if downgrade {
            card.Urgency = downgradeUrgency(card.Urgency)
            card.Rationale += " (seasonal context: " + rationale + ")"
        }
        cards = append(cards, card)
    }
}
```

### Scope boundaries
- Only Indian festivals and seasons are populated in Phase 2. Australia's `seasonal_calendar.yaml` is seeded with an empty `windows: []` list to establish the contract.
- Patient-segment-aware behavioral weighting (rural/urban harvest season) is Phase 3 — requires KB-20 patient profile integration and clinical review.

## Item 5: Module 13 Flink Integration (Kafka Publisher)

**Goal**: KB-26 publishes a `DomainTrajectoryComputed` event to Kafka topic `kb26.domain_trajectory.v1` after each successful trajectory compute. Module 13's Flink state-sync consumes this (consumer side is out of scope for Phase 2).

### Event schema

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

### Kafka producer (KB-26)

New file `internal/services/trajectory_publisher.go`:

```go
type TrajectoryPublisher interface {
	Publish(ctx context.Context, event DomainTrajectoryComputedEvent) error
}

type KafkaTrajectoryPublisher struct {
	writer *kafka.Writer
	topic  string
	logger *zap.Logger
}

func NewKafkaTrajectoryPublisher(brokers []string, topic string, logger *zap.Logger) *KafkaTrajectoryPublisher

func (p *KafkaTrajectoryPublisher) Publish(ctx context.Context, event DomainTrajectoryComputedEvent) error
```

### No-op publisher for tests

```go
type NoopTrajectoryPublisher struct{}
func (NoopTrajectoryPublisher) Publish(ctx context.Context, event DomainTrajectoryComputedEvent) error { return nil }
```

### Wire into TrajectoryEngine

The engine (from Item 2) takes an optional `TrajectoryPublisher`. After a successful compute, engine calls `publisher.Publish(ctx, event)`. Publish failure is logged and non-fatal (matching the existing "persistence failure is non-fatal" pattern in the API handler).

### Consumer contract (documented, out of scope)

Module 13 consumers should:
- Subscribe to `kb26.domain_trajectory.v1`
- Deserialize the JSON event
- Update Flink state's `domain_velocities` map keyed by patient_id
- Ignore events older than 48 hours (prevent replay from populating stale state)

A consumer README is added at `backend/stream-services/docs/kb26_domain_trajectory_consumer.md` with the schema and integration instructions. Module 13 Java/Flink implementation is tracked as a separate task.

## File Inventory

### KB-26 (Metabolic Digital Twin)
| Action | File | Item |
|--------|------|------|
| Create | `internal/metrics/trajectory_metrics.go` | 1 |
| Create | `internal/config/trajectory_config.go` | 2 |
| Modify | `internal/config/config.go` (load TrajectoryThresholds) | 2 |
| Create | `internal/services/trajectory_engine.go` (refactored from mri_domain_trajectory.go) | 2 |
| Delete/Merge | `internal/services/mri_domain_trajectory.go` (contents merged into trajectory_engine.go) | 2 |
| Modify | `internal/services/mri_domain_trajectory_test.go` (use engine struct) | 2 |
| Modify | `internal/services/egfr_trajectory.go` (numerical stability fix) | 3 |
| Modify | `internal/services/egfr_trajectory_test.go` (add clustered scores test) | 3 |
| Create | `internal/services/trajectory_engine_test.go` (new ClusteredScores test; all existing tests now target the struct) | 3 |
| Create | `internal/services/trajectory_publisher.go` | 5 |
| Create | `internal/services/trajectory_publisher_test.go` | 5 |
| Modify | `internal/api/domain_trajectory_handlers.go` (use engine + publish event + increment metrics) | 1, 2, 5 |
| Modify | `internal/api/server.go` (init engine, metrics, publisher) | 1, 2, 5 |
| Modify | `pkg/trajectory/models.go` (Compute wraps default engine) | 2 |

### KB-23 (Decision Cards)
| Action | File | Item |
|--------|------|------|
| Create | `internal/metrics/trajectory_card_metrics.go` | 1 |
| Create | `internal/services/seasonal_context.go` | 4 |
| Create | `internal/services/seasonal_context_test.go` | 4 |
| Modify | `internal/services/trajectory_card_rules.go` (accept optional SeasonalContext) | 4 |
| Modify | `internal/services/trajectory_card_rules_test.go` (add seasonal suppression tests) | 4 |
| Modify | `internal/services/kb26_trajectory_client.go` (metrics instrumentation) | 1 |
| Modify | `internal/api/composite_handlers.go` (metrics + seasonal context) | 1, 4 |
| Modify | `internal/api/server.go` (init metrics + seasonal context) | 1, 4 |

### Market Configs
| Action | File | Item |
|--------|------|------|
| Create | `backend/shared-infrastructure/market-configs/india/seasonal_calendar.yaml` | 4 |
| Create | `backend/shared-infrastructure/market-configs/australia/seasonal_calendar.yaml` | 4 |

### Observability
| Action | File | Item |
|--------|------|------|
| Create | `observability/dashboards/mhri_trajectory.json` | 1 |

### Stream Services Docs
| Action | File | Item |
|--------|------|------|
| Create | `backend/stream-services/docs/kb26_domain_trajectory_consumer.md` | 5 |

**Total: 12 create, 9 modify (21 files)**

## Phases

Phase 2 is a single delivery with 5 item slices that can land in any order after Item 2 (config refactor). Item 2 is the sequencing bottleneck because Items 1, 3, 4, and 5 all build on the engine struct.

| Slice | Items | Deliverables |
|-------|-------|-------------|
| P2.1: Observability foundation | 1 (KB-26 only) | Metrics package, engine instrumentation, no behavior change |
| P2.2: Config refactor | 2 | Engine struct, YAML loading, all tests target new shape |
| P2.3: Numerical stability | 3 | Two-pass ssTot in both trajectory engines, regression test |
| P2.4: Seasonal adjustment | 4 | SeasonalContext service, India calendar, card rule integration |
| P2.5: Kafka publisher | 5 | Publisher, wiring, consumer contract docs |
| P2.6: KB-23 observability + dashboard | 1 (KB-23), 1 (Grafana) | Fetch metrics, card eval metrics, dashboard JSON |

## Future Work (out of scope)

- **Module 13 Flink consumer implementation** — Java/Flink code that subscribes to `kb26.domain_trajectory.v1` and populates the `domain_velocities` state map. Different tech stack, different team. The Kafka schema and consumer README establish the contract.
- **Australia seasonal calendar population** — empty YAML seeded in Phase 2; clinical review to add Australian windows (e.g., harvest season for rural Indigenous populations, summer heat for QLD/WA) is Phase 3.
- **Patient-segment-aware behavioral weighting** — rural/urban/semi-urban distinction for behavioral leading indicator requires KB-20 patient profile integration.
- **LeadDays field repopulation via cross-correlation** — proper time-series lead-lag analysis between behavioral and clinical domains requires statistical validation. Phase 3.
- **Trajectory-of-the-trajectory (second-derivative) analysis** — detecting patients whose per-domain slopes are themselves changing over successive snapshots. Reads `domain_trajectory_history` (already populated). Phase 3 research item.
- **Cross-feature aggregation rules** — compound patterns like "masked HTN + concordant cardio decline → escalate urgency". Requires production data from Phase 2 observability before rule design. Phase 3.
- **Frontend dashboard** — clinician-facing UI consuming `GET /api/v1/kb26/mri/:patientId/domain-trajectory`. API already exists; UI is a different workstream.
- **Config hot-reload** — currently thresholds are loaded at startup only; hot-reload on YAML change is deferred until production tuning actually needs it.
- **Threshold override per-patient** — per-patient trajectory threshold customization (e.g., elite athletes with different baselines) is not currently scoped.
