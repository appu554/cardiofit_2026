# MHRI Domain-Decomposed Trajectory — Design Spec

**Date**: 2026-04-11
**Status**: Approved
**Scope**: KB-26 (Metabolic Digital Twin) + KB-23 (Decision Cards)

## Problem

The current MHRI system computes per-domain scores (glucose 35%, cardio 25%, body_comp 25%, behavioral 15%) but has no trajectory computation over these scores. There is no composite trajectory and no per-domain trajectory. Clinicians cannot see which domain is driving deterioration, whether domains are diverging, or whether behavioral disengagement is a leading indicator of clinical decline.

**Clinical impact**: Two patients whose composite MHRI drops from 72 to 58 may need completely different interventions — one needs glycaemic intensification (glucose domain driving), the other needs urgent BP review + clinical outreach (cardio + behavioral driving). Without decomposition, the clinician must independently review raw data to figure out what a trajectory system should already compute.

**Evidence base**:
- Dagliati et al. 2018 (J Biomed Inform): discordant domain trajectories predict complications 2-4 years earlier than composites
- Ahlqvist et al. 2018 (Lancet D&E): trajectory pattern clustering predicts outcomes better than single metrics
- Ndumele et al. 2023 (Circulation): AHA CKM framework calls for longitudinal multi-domain risk assessment

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Trajectory system architecture | Decomposed-first — no separate composite-only trajectory | No composite trajectory exists today. `ComputeDecomposedTrajectory` produces composite slope as a by-product. No throwaway intermediate code. |
| KB-23 → KB-26 dependency | Direct import of KB-26 models into KB-23 | Same monorepo under `shared-infrastructure/`. Natural dependency direction: downstream consumer imports upstream producer. |
| Implementation approach | Monolithic compute function with helper extraction | Matches existing `egfr_trajectory.go` pattern. Single call site. Helpers (`detectDivergences`, `detectLeadingIndicators`, `detectDomainCrossings`) provide natural test boundaries. |
| Scope | Core engine only (18 steps). Market-specific adjustments as Future Work. | Core decomposition is substantial (13 files, 4 phases). Seasonal/market logic depends on it working first. |
| Threshold source | Go constants matching YAML reference | Matches existing `egfr_trajectory.go` pattern. YAML is canonical reference for future config-driven overrides. |

## Architecture

```
MRIScore (existing) ──→ DomainTrajectoryPoint[] ──→ ComputeDecomposedTrajectory()
                                                          │
                                                          ├─ Composite OLS (slope, R², trend)
                                                          ├─ Per-domain OLS ×4 (slope, R², trend, confidence)
                                                          ├─ Dominant driver (weighted contribution %)
                                                          ├─ Concordant deterioration (≥2 domains)
                                                          ├─ detectDivergences() → DivergencePattern[]
                                                          ├─ detectDomainCrossings() → DomainCategoryCrossing[]
                                                          └─ detectLeadingIndicators() → LeadingIndicator[]
                                                          │
                                                          ▼
                                                   DecomposedTrajectory
                                                          │
                                          ┌───────────────┼───────────────┐
                                          ▼               ▼               ▼
                                   KB-23 Cards    Four-Pillar      API Response
                                   (5 types)      Integration      + History Table
```

## Data Models (KB-26)

**File**: `kb-26-metabolic-digital-twin/internal/models/domain_trajectory.go`

### MHRIDomain (string enum)
`GLUCOSE` | `CARDIO` | `BODY_COMP` | `BEHAVIORAL`

### DomainTrajectoryPoint (input)
Timestamped snapshot: composite score + 4 domain scores.

### DomainSlope (per-domain OLS result)
- `SlopePerDay float64` — OLS regression slope in score-units/day
- `Trend string` — 5-tier: `RAPID_IMPROVING` | `IMPROVING` | `STABLE` | `DECLINING` | `RAPID_DECLINING`
- `StartScore`, `EndScore`, `DeltaScore` — window boundary values
- `R2 float64` — goodness of fit
- `Confidence string` — `HIGH` (R² >= 0.5) | `MODERATE` (0.25-0.5) | `LOW` (<0.25)

### DivergencePattern
- Improving/declining domain pair with slopes
- `DivergenceRate` = |improving_slope| + |declining_slope|
- `ClinicalConcern` — human-readable divergence description
- `PossibleMechanism` — clinical hypothesis for the specific pair

### LeadingIndicator
- Behavioral domain declining before clinical domains
- `LaggingDomains []MHRIDomain` — which clinical domains are following

### DomainCategoryCrossing
- Domain crossing MHRI category boundary (OPTIMAL >=70, MILD >=55, MODERATE >=40, HIGH <40)
- Direction: `WORSENED` | `IMPROVED`

### DecomposedTrajectory (full output)
- Composite trajectory fields (backward-compat)
- `DomainSlopes map[MHRIDomain]DomainSlope` — O(1) lookup
- `DominantDriver *MHRIDomain` — nil when composite not declining
- `DriverContribution float64` — % of composite change from dominant domain
- `Divergences`, `LeadingIndicators`, `DomainCrossings` — derived analytics
- `HasDiscordantTrend`, `ConcordantDeterioration`, `DomainsDeterioration` — summary flags

### DomainTrajectoryHistory (GORM persistence)
UUID PK, patient_id + snapshot_date (unique), flattened per-domain slopes, discordance flag.

## Configuration

**File**: `backend/shared-infrastructure/market-configs/shared/domain_trajectory_thresholds.yaml`

| Group | Key Thresholds |
|-------|---------------|
| Trend classification | RAPID_IMPROVING >1.0, IMPROVING >0.3, STABLE +/-0.3, DECLINING <-0.3, RAPID_DECLINING <-1.0 (score/day) |
| Divergence | min_divergence_rate: 0.5, min slopes +/-0.3 |
| Leading indicator | behavioral lead >=7 days, min slope -0.5, min 10 data points |
| Concordant deterioration | >=2 domains declining at >=0.3/day |
| Dominant driver | >=40% weighted contribution using MHRI weights (G:35%, C:25%, BC:25%, B:15%) |
| R-squared confidence | HIGH >=0.5, MODERATE >=0.25, LOW <0.25 |
| Category boundaries | OPTIMAL >=70, MILD >=55, MODERATE >=40, HIGH <40 |

## Core Decomposition Engine (KB-26)

**File**: `kb-26-metabolic-digital-twin/internal/services/mri_domain_trajectory.go`

### ComputeDecomposedTrajectory(patientID, points) -> DecomposedTrajectory

1. Guard: <2 points -> INSUFFICIENT_DATA for all domains
2. Sort points by timestamp, compute window days
3. Composite OLS -> slope, R², trend
4. Per-domain OLS x4 using domain-specific score extractors -> slope, R², trend, confidence
5. Count declining domains -> concordant deterioration flag (>=2)
6. Dominant driver: max(|slope| x weight) among declining domains, contribution = weighted%
7. detectDivergences() -> pairwise opposite-direction detection
8. detectDomainCrossings() -> first vs last point category comparison per domain
9. detectLeadingIndicators() -> behavioral slope < -0.5 AND leading clinical domains

### Helper functions (unexported)
- `computeOLSWithR2()` — OLS linear regression with R-squared
- `classifyDomainTrend()` — slope -> 5-tier trend
- `classifyR2Confidence()` — R² -> HIGH/MODERATE/LOW
- `detectDomainCrossings()` — category boundary comparison
- `detectLeadingIndicators()` — behavioral lead detection
- `extractScores()` — generic extractor with function parameter
- `sortTrajectoryPoints()` — insertion sort by timestamp
- `categorizeDomainScore()` — score -> OPTIMAL/MILD/MODERATE/HIGH

### Tests (8 tests)
- `GlucoseDeclining_CardioStable` — per-domain independence, dominant driver
- `AllDomainsImproving` — no discordance, no deterioration
- `ConcordantDeterioration` — >=2 domains flag
- `InsufficientData` — single point guard
- `NoisyData_LowConfidence` — oscillating data -> LOW R²
- `GlucoseOptimalToMild` — category crossing detection
- `RajeshKumar` — E2E scenario: 3 domains declining, behavioral RAPID_DECLINING

## Divergence Detection (KB-26)

**File**: `kb-26-metabolic-digital-twin/internal/services/domain_divergence.go`

### detectDivergences(slopes) -> []DivergencePattern

Pairwise comparison across 4 domains (6 pairs). Flags when:
1. One domain slope > +0.3 (improving)
2. Other domain slope < -0.3 (declining)
3. Combined divergence rate >= 0.5

### inferDivergenceMechanism() — 10 domain pair hypotheses

| Improving | Declining | Hypothesis |
|-----------|-----------|------------|
| GLUCOSE | CARDIO | Glycaemic therapy lacks hemodynamic benefit -> consider SGLT2i |
| CARDIO | GLUCOSE | BP meds worsening glycaemia (thiazide, beta-blocker) |
| GLUCOSE | BEHAVIORAL | Meds working, patient disengaging -> unsustainable |
| BEHAVIORAL | GLUCOSE | Engaged but glucose worsening -> intensify pharmacotherapy |
| CARDIO | BEHAVIORAL | BP improving, engagement declining -> adherence rebound risk |
| BEHAVIORAL | CARDIO | Engaged but CV worsening -> secondary HTN workup |
| GLUCOSE | BODY_COMP | Insulin-driven weight gain or TZD fluid retention |
| BODY_COMP | GLUCOSE | Paradoxical -> stress hyperglycaemia, steroids |
| CARDIO | BODY_COMP | Effective meds, dietary non-adherence |
| BODY_COMP | CARDIO | Weight improving, CV worsening -> sleep apnea, endocrine |

Fallback: generic "domain divergence detected" for unmatched pairs.

### Tests (4 tests)
- `GlucoseImproving_RenalDeclining` — single divergence, rate >= 1.5
- `NoDivergence_AllStable` — all within +/-0.3 -> empty
- `MultiplePairs` — 2 improving + 2 declining -> >=2 divergences
- `ClinicalConcernText` — concern string contains domain names

## Trajectory Cards (KB-23)

**File**: `kb-23-decision-cards/internal/services/trajectory_card_rules.go`

### EvaluateTrajectoryCards(traj) -> []TrajectoryCard

5 card types in priority order:

| # | Card Type | Urgency | Trigger |
|---|-----------|---------|---------|
| 1 | `CONCORDANT_DETERIORATION` | IMMEDIATE (>=3) / URGENT (2) | ConcordantDeterioration flag |
| 2 | `DOMAIN_DIVERGENCE` | URGENT | HasDiscordantTrend flag |
| 3 | `BEHAVIORAL_LEADING_INDICATOR` | URGENT | LeadingIndicators present |
| 4 | `DOMAIN_RAPID_DECLINE` | URGENT | Single domain RAPID_DECLINING, non-LOW confidence, NOT covered by concordant |
| 5 | `DOMAIN_CATEGORY_CROSSING` | ROUTINE | Domain crossed boundary in WORSENED direction |

### Four-Pillar Integration

**Modified file**: `kb-23-decision-cards/internal/services/four_pillar_evaluator.go`

Add `DecomposedTrajectory *dtModels.DecomposedTrajectory` to `FourPillarInput`.

In `evaluateMonitoringPillar`, append recommendations:
- Concordant deterioration -> increase monitoring frequency
- Discordant trajectory -> investigate cross-domain medication effects
- Behavioral leading indicator -> clinical outreach recommended

### Tests (5 tests)
- `ConcordantDeterioration` — 3 domains -> IMMEDIATE
- `DivergenceAlert` — glucose up/cardio down -> URGENT with rationale
- `BehavioralLeadingIndicator` — URGENT, rationale contains "behavioral"
- `SingleDomainRapidDecline` — cardio alone -> card with domain in title
- `AllStable_NoCards` — zero urgent/immediate cards

## Database Migration (KB-26)

**File**: `kb-26-metabolic-digital-twin/migrations/006_domain_trajectory.sql`

```sql
CREATE TABLE domain_trajectory_history (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id      VARCHAR(100) NOT NULL,
    snapshot_date   DATE NOT NULL,
    window_days     INT,
    composite_slope DECIMAL(6,3),
    glucose_slope   DECIMAL(6,3),
    cardio_slope    DECIMAL(6,3),
    body_comp_slope DECIMAL(6,3),
    behavioral_slope DECIMAL(6,3),
    has_discordance BOOLEAN DEFAULT FALSE,
    dominant_driver VARCHAR(20),
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(patient_id, snapshot_date)
);
CREATE INDEX idx_dth_patient ON domain_trajectory_history(patient_id, snapshot_date DESC);
```

Flattened slopes (not JSON) for queryability.

## API Endpoint (KB-26)

**File**: `kb-26-metabolic-digital-twin/internal/api/domain_trajectory_handlers.go`
**Modified**: `kb-26-metabolic-digital-twin/internal/api/routes.go`

`GET /api/v1/patients/:id/domain-trajectory` — fetches recent MRIScore records, maps to DomainTrajectoryPoint[], calls ComputeDecomposedTrajectory(), optionally persists snapshot, returns DecomposedTrajectory JSON.

## File Inventory

| Phase | File | Action | Service |
|-------|------|--------|---------|
| D1 | `internal/models/domain_trajectory.go` | Create | KB-26 |
| D1 | `market-configs/shared/domain_trajectory_thresholds.yaml` | Create | Shared |
| D2 | `internal/services/mri_domain_trajectory.go` | Create | KB-26 |
| D2 | `internal/services/mri_domain_trajectory_test.go` | Create | KB-26 |
| D3 | `internal/services/domain_divergence.go` | Create | KB-26 |
| D3 | `internal/services/domain_divergence_test.go` | Create | KB-26 |
| D4 | `internal/services/trajectory_card_rules.go` | Create | KB-23 |
| D4 | `internal/services/trajectory_card_rules_test.go` | Create | KB-23 |
| D4 | `internal/services/four_pillar_evaluator.go` | Modify | KB-23 |
| D4 | `migrations/006_domain_trajectory.sql` | Create | KB-26 |
| D4 | `internal/api/domain_trajectory_handlers.go` | Create | KB-26 |
| D4 | `internal/api/routes.go` | Modify | KB-26 |

**Total: 12 files (9 create, 3 modify)**

## Phases

| Phase | Steps | Deliverables |
|-------|-------|-------------|
| D1: Models + Config | 1-2 | Data models, thresholds YAML |
| D2: Core Engine | 3-7 | `ComputeDecomposedTrajectory()`, 8 tests |
| D3: Divergence | 8-11 | `detectDivergences()`, mechanism inference, 4 tests |
| D4: Cards + Integration | 12-18 | 5 card types, four-pillar integration, migration, API, 5 tests |

## Future Work (out of scope)

- **India seasonal adjustment**: Festival season (Diwali, Pongal) body comp/glucose patterns, Ramadan altered eating, extreme heat cardio impact. Domain decomposition must not flag seasonal patterns as clinical deterioration.
- **India behavioral weighting**: Rural/semi-urban "engagement collapse" driven by harvest season, family obligations, telecom connectivity — not clinical disengagement. Behavioral leading indicator weighted differently by patient segment.
- **Australia Indigenous benchmarking**: Aboriginal and Torres Strait Islander trajectory patterns — earlier/faster renal domain decline, more labile glucose. Population-specific trajectory benchmarking.
- **Australia GPMP alignment**: 6-month GPMP review cycle creates natural trajectory reporting periods. Per-domain trend summary for chronic disease management plan.
