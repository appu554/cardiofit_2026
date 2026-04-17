# Patient Acuity Index (PAI) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a velocity-weighted Patient Acuity Index that answers "who needs me first?" by combining 5 dimensions (velocity, proximity, behavioral, clinical context, attention gap) into a 0-100 urgency score — enabling clinical triage, worklist sort, and escalation in both Indian primary care and Australian GP settings.

**Architecture:** PAI is a higher-order computation in KB-26 that consumes outputs from V4-1 through V4-8 (domain trajectories, eGFR, CGM, engagement, safety events, confounder calendar). Event-driven: recalculates on new clinical data with 15-minute rate limiting. Publishes PAI change events to KB-19 for downstream escalation. YAML-driven dimension weights and thresholds with market-specific overrides for India (ASHA caseloads, festival dampening) and Australia (indigenous remote communities, GPMP integration).

**Tech Stack:** Go 1.21 (Gin, GORM) for KB-26 PAI engine. PostgreSQL 15 for PAI history. Kafka for PAI change events. Existing V4-1 through V4-8 outputs as input sources. YAML market configs.

---

## Scope Check

This is one cohesive subsystem: the PAI compute engine. It lives primarily in KB-26, with a small card-prioritizer extension in KB-23. One plan is appropriate.

## Existing Infrastructure

| What exists | Where | Relevance |
|---|---|---|
| `MRIScorer` (composite health score) | KB-26 `internal/services/` | PAI is the urgency counterpart to MRI's health state |
| `TrajectoryEngine` (domain decomposition) | KB-26 `internal/services/trajectory_engine.go` | Velocity dimension consumes domain slopes + second derivative |
| `KafkaTrajectoryPublisher` | KB-26 `internal/services/trajectory_publisher.go` | PAI will create a parallel `KafkaPAIPublisher` using the same pattern |
| `SummaryContext` wire contract | KB-20 `internal/services/summary_context_service.go` | Provides engagement, CKM stage, medications, CGM status |
| `SafetyEvent` audit trail | KB-20 `internal/models/safety_event.go` | Clinical context dimension sources |
| `ConfounderCalendar` (V4-8) | KB-20 `internal/services/confounder_calendar.go` | Seasonal dampening for velocity dimension |
| `ClusterTransitionRecord` (V4-7) | KB-20 `internal/models/cluster_history.go` | Phenotype stability feeds into context |
| KB-26 AutoMigrate pattern | KB-26 `main.go` lines 62-77 | PAI models added here |
| KB-26 Server struct (13 deps) | KB-26 `internal/api/server.go` | PAI engine injected here |

## File Inventory

### KB-26 (Metabolic Digital Twin) — PAI Compute Engine
| Action | File | Responsibility |
|---|---|---|
| Create | `internal/models/patient_acuity.go` | `PAIScore`, `PAIDimensionInput`, `PAITier` constants, `PAIHistory`, `PAIChangeEvent` |
| Create | `internal/services/pai_velocity.go` | `ComputeVelocityScore(input, config) float64` — slope mapping, 2nd derivative amplification, concordant bonus, seasonal dampening |
| Create | `internal/services/pai_velocity_test.go` | 6 tests: severe decline, accelerating amplified, improving low, stable moderate, no data zero, seasonal dampening |
| Create | `internal/services/pai_proximity.go` | `ComputeProximityScore(input, config) float64` — exponential scaling to danger thresholds, multi-metric compounding, HF-only acute weight gain |
| Create | `internal/services/pai_proximity_test.go` | 6 tests: eGFR near threshold, eGFR far, multiple metrics, exponential scaling, acute weight HF-only, no data |
| Create | `internal/services/pai_behavioral.go` | `ComputeBehavioralScore(input, config) float64` — engagement composite, measurement frequency drop, compound cessation+disengagement |
| Create | `internal/services/pai_behavioral_test.go` | 5 tests: disengaged high, active low, measurement cessation, frequency drop, compound both=95 |
| Create | `internal/services/pai_context.go` | `ComputeContextScore(input, config) float64` — CKM stage base, post-discharge/illness/hypo/steroid/polypharmacy modifiers, NYHA amplifier |
| Create | `internal/services/pai_context_test.go` | 5 tests: CKM 4c-HFrEF max, CKM 2 low, post-discharge bonus, polypharmacy-elderly, no modifiers |
| Create | `internal/services/pai_attention.go` | `ComputeAttentionScore(input, config) float64` — days since clinician, unacknowledged card accumulation |
| Create | `internal/services/pai_attention_test.go` | 5 tests: 90d no clinician critical, recent review low, unacknowledged cards, combined high, no data |
| Create | `internal/services/pai_engine.go` | `ComputePAI(input, config) PAIScore` — weights 5 dimensions, determines tier/trend/dominant/reason/action, `PAIConfig` struct + YAML loader |
| Create | `internal/services/pai_engine_test.go` | 6 tests: Rajesh high acuity, stable-well-managed low, acute-on-chronic critical, dominant dimension, no data minimal, significant change |
| Create | `internal/services/pai_repository.go` | `PAIRepository` — persist PAIHistory, fetch latest, fetch trend (last N scores) |
| Create | `internal/services/pai_event_trigger.go` | `PAIEventTrigger` — rate limiting (15min), significant change detection, Kafka publish |
| Create | `internal/services/pai_event_trigger_test.go` | 4 tests: rate limit blocks rapid recompute, significant change publishes, tier change publishes, no change suppresses |
| Create | `internal/api/pai_handlers.go` | GET `/pai/:patientId`, GET `/pai/:patientId/history`, POST `/pai/:patientId/compute` |
| Modify | `internal/api/routes.go` | Add PAI route group |
| Modify | `internal/database/connection.go` or `main.go` | AutoMigrate `PAIScore` + `PAIHistory` |
| Modify | `main.go` | Wire PAI engine + repository + event trigger into Server |

### Market Configs
| Action | File | Responsibility |
|---|---|---|
| Create | `market-configs/shared/pai_dimensions.yaml` | 5 dimension weights, velocity thresholds, proximity metrics, behavioral thresholds, context stage map, attention gaps, tier boundaries, rate limiting, confounder dampening |
| Create | `market-configs/india/pai_overrides.yaml` | Weight adjustments (velocity 0.32, behavioral 0.18), ASHA attention gaps (120d critical), festival velocity dampening, telehealth thresholds |
| Create | `market-configs/australia/pai_overrides.yaml` | Indigenous remote attention gaps (56d critical), GPMP integration flag, aged care aggregation flag |

### KB-23 (Decision Cards) — PAI-Aware Card Prioritization
| Action | File | Responsibility |
|---|---|---|
| Create | `internal/services/pai_card_prioritizer.go` | `PrioritizeCards(cards, paiScores) → sorted` — sort pending cards by patient PAI, annotate with urgency tier |
| Create | `internal/services/pai_card_prioritizer_test.go` | 4 tests: critical-first sort, equal-PAI stable order, missing-PAI-last, empty input |

**Total: 26 files (23 create, 3 modify), ~50 tests**

---

### Task 1: PAI data models + market configuration YAML

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/models/patient_acuity.go`
- Create: `market-configs/shared/pai_dimensions.yaml`
- Create: `market-configs/india/pai_overrides.yaml`
- Create: `market-configs/australia/pai_overrides.yaml`

- [ ] **Step 1:** Create `patient_acuity.go` with 5 types from the spec: `PAIScore` (GORM model with composite score, 5 dimension scores, tier, dominant dimension, actionability context, change detection, metadata), `PAIDimensionInput` (all inputs across 5 dimensions — velocity slopes, proximity values, behavioral signals, clinical context, attention gaps, confounder context), `PAITier` constants (CRITICAL >80, HIGH 60-80, MODERATE 40-60, LOW 20-40, MINIMAL <20), `PAIHistory` (snapshot for trend analysis), `PAIChangeEvent` (published to KB-19 on significant change). Copy the exact type definitions from the spec's Phase A1 Step 1.

- [ ] **Step 2:** Create `pai_dimensions.yaml` with the complete configuration from the spec: dimension weights (velocity 0.30, proximity 0.25, behavioral 0.20, context 0.15, attention 0.10), velocity thresholds (severe -2.0, moderate -1.0, mild -0.3, stable 0.3) with second derivative multipliers and concordant bonus, proximity metrics (eGFR/HbA1c/SBP/potassium/TBR/acute weight gain with exponential scaling exponent 2.0), behavioral thresholds (engagement levels + measurement frequency), context CKM stage base scores with modifiers and NYHA amplifier, attention gap days, tier boundaries, rate limiting (15min/8 per hour), and confounder dampening (max velocity 60 during season). Copy from the spec.

- [ ] **Step 3:** Create `india/pai_overrides.yaml` with weight adjustments (velocity 0.32, behavioral 0.18), ASHA attention overrides (critical 120d), festival-season velocity dampening (Diwali max 50, Ramadan max 55), and telehealth triage thresholds (PAI ≥60 priority slot, PAI ≥40 ASHA visit).

- [ ] **Step 4:** Create `australia/pai_overrides.yaml` with indigenous remote attention overrides (critical 56d, high 42d, adequate 21d), GPMP integration flag, and aged care aggregation settings.

- [ ] **Step 5:** Verify models compile: `go build ./...` in KB-26.

- [ ] **Step 6:** Verify all YAML parses correctly.

- [ ] **Step 7:** Commit: `feat(kb26): PAI data models + market config (PAI Task 1)`

---

### Task 2: Velocity dimension engine

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/services/pai_velocity.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/pai_velocity_test.go`

- [ ] **Step 1:** Write 6 failing tests from the spec:
1. `TestVelocity_SevereDecline` — slope -2.5, concordant 3 domains → score ≥85
2. `TestVelocity_AcceleratingDecline_Amplified` — slope -1.5 + ACCELERATING_DECLINE > same slope + STABLE
3. `TestVelocity_Improving_LowScore` — slope +1.5 → score <15
4. `TestVelocity_Stable_ModerateScore` — slope -0.1 → score 0-30
5. `TestVelocity_NoData_Zero` — nil slopes → score 0
6. `TestVelocity_SeasonalDampening` — slope -1.8 + seasonal window + dampening enabled → score ≤60

Include `testPAIConfig()` helper returning a `PAIConfig` with the standard values from pai_dimensions.yaml, and `floatPtr`/`stringPtr`/`intPtr` helpers.

- [ ] **Step 2:** Implement `ComputeVelocityScore(input PAIDimensionInput, cfg *PAIConfig) float64` with: piecewise linear slope-to-score mapping, second derivative amplification (1.5× for accelerating, 0.7× for decelerating), concordant deterioration bonus (+15 base + 5 per extra domain), seasonal confounder dampening (cap at MaxVelocityDuringSeason when active). Include `scaleLinear(value, fromMin, fromMax, toMin, toMax)` helper.

- [ ] **Step 3:** Run tests — all 6 pass.

- [ ] **Step 4:** Commit: `feat(kb26): PAI velocity dimension — slope mapping + acceleration + seasonal dampening (PAI Task 2)`

---

### Task 3: Proximity dimension engine

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/services/pai_proximity.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/pai_proximity_test.go`

- [ ] **Step 1:** Write 6 failing tests from the spec:
1. `TestProximity_EGFRNearThreshold` — eGFR 32 (2 from threshold 30) → score ≥80
2. `TestProximity_EGFRFarFromThreshold` — eGFR 75 → score <15
3. `TestProximity_MultipleMetricsNearThreshold` — eGFR 33 + SBP 172 + potassium 5.4 → compound score ≥70
4. `TestProximity_ExponentialScaling` — eGFR 31.5 (90% proximity) vs 37.5 (50%) → ratio >2×
5. `TestProximity_AcuteWeightGain_HFOnly` — 2.5kg gain for CKM 4c >> same gain for CKM 2
6. `TestProximity_NoData_Zero` — empty input → 0

- [ ] **Step 2:** Implement `ComputeProximityScore(input, cfg) float64` with: 6 metric definitions (eGFR below 30, HbA1c above 10, SBP above 180, potassium above 6, TBR L2 above 5%, acute weight gain above 3kg for CKM 4c only), exponential scaling (`fraction^exponent` where exponent=2.0), multi-metric compounding (max + 20% of secondary contributions). Include `computeSingleProximity`, `buildProximityMetrics`, `matchesContext` helpers.

- [ ] **Step 3:** Run tests — all 6 pass.

- [ ] **Step 4:** Commit: `feat(kb26): PAI proximity dimension — exponential danger boundary scoring (PAI Task 3)`

---

### Task 4: Behavioral dimension engine

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/services/pai_behavioral.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/pai_behavioral_test.go`

- [ ] **Step 1:** Write 5 failing tests:
1. `TestBehavioral_Disengaged_High` — engagement composite 0.25, status DISENGAGED → score ≥80
2. `TestBehavioral_Active_Low` — engagement composite 0.85, status ACTIVE, good frequency → score <20
3. `TestBehavioral_MeasurementCessation` — 0 readings for 5+ days → score ≥70
4. `TestBehavioral_FrequencyDrop` — 50% drop from average (was 7/week, now 3.5/week) → score ~50
5. `TestBehavioral_CompoundBoth` — disengaged AND stopped measuring → score ≥95

- [ ] **Step 2:** Implement `ComputeBehavioralScore(input, cfg) float64` with: engagement composite threshold mapping (disengaged <0.3 → 80, declining 0.3-0.5 → 50-80, active 0.5-0.7 → 20-50, engaged >0.7 → 0-20), measurement frequency change detection (cessation 5+ days → 70, >50% drop → 50, 25-50% drop → 25), compound rule (both disengaged + ceased → 95).

- [ ] **Step 3:** Run tests — all 5 pass.

- [ ] **Step 4:** Commit: `feat(kb26): PAI behavioral dimension — engagement + measurement frequency (PAI Task 4)`

---

### Task 5: Clinical context dimension engine

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/services/pai_context.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/pai_context_test.go`

- [ ] **Step 1:** Write 5 failing tests:
1. `TestContext_CKM4c_HFrEF_NYHA3_Max` — CKM 4c (base 65) + NYHA III (×1.3) + post-discharge 30d (+25) + polypharmacy-elderly (+15) → near 100
2. `TestContext_CKM2_Healthy_Low` — CKM 2 (base 10), no modifiers → score ~10
3. `TestContext_PostDischargeBonus` — CKM 3 + post-discharge within 30d → score includes +25
4. `TestContext_PolypharmacyElderly` — age 78, 7 meds → adds 15
5. `TestContext_NoModifiers` — CKM 1, young, no events → score ~5

- [ ] **Step 2:** Implement `ComputeContextScore(input, cfg) float64` with: CKM stage base score map ("0"→0, "1"→5, "2"→10, "3"→20, "4a"→35, "4b"→50, "4c"→65), modifier additions (post_discharge_30d +25, acute_illness +20, recent_hypo +15, active_steroid +10, polypharmacy_elderly +15), NYHA amplifier (IV ×1.5, III ×1.3, II ×1.1), cap at 100.

- [ ] **Step 3:** Run tests — all 5 pass.

- [ ] **Step 4:** Commit: `feat(kb26): PAI context dimension — CKM stage + clinical modifiers + NYHA (PAI Task 5)`

---

### Task 6: Attention gap dimension engine

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/services/pai_attention.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/pai_attention_test.go`

- [ ] **Step 1:** Write 5 failing tests:
1. `TestAttention_90DaysNoClinician_Critical` — 90 days since clinician → score ≥90
2. `TestAttention_RecentReview_Low` — clinician 5 days ago, no pending cards → score <10
3. `TestAttention_UnacknowledgedCards` — 3 unacked cards, oldest 10 days → adds 10×3 + 3×10 = 60 (capped at 50 for cards component)
4. `TestAttention_Combined_High` — 45 days no clinician + 2 unacked cards → elevated score
5. `TestAttention_NoData_Zero` — zero days → score 0

- [ ] **Step 2:** Implement `ComputeAttentionScore(input, cfg) float64` with: days-since-clinician piecewise scoring (>90d → 100, 60-90d → 60-100, 30-60d → 30-60, 14-30d → 10-30, <14d → 0-10), unacknowledged card scoring (10 per card + 3 per day oldest, capped at 50), combine both (max(clinician_score, card_score) + 20% of secondary).

- [ ] **Step 3:** Run tests — all 5 pass.

- [ ] **Step 4:** Commit: `feat(kb26): PAI attention dimension — clinician gap + unacknowledged cards (PAI Task 6)`

---

### Task 7: PAI composite engine + config loader

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/services/pai_engine.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/pai_engine_test.go`

- [ ] **Step 1:** Write 6 failing tests from the spec:
1. `TestPAI_RajeshKumar_HighAcuity` — full scenario: slope -1.4, concordant 3 domains, eGFR 42, HbA1c 8.2, SBP 170, engagement declining, 35 days no clinician → score ≥65, tier HIGH
2. `TestPAI_StableWellManaged_LowAcuity` — slope +0.1, eGFR 72, HbA1c 6.8, SBP 128, engagement 0.85, clinician 12d ago → score <25, tier LOW/MINIMAL
3. `TestPAI_AcuteOnChronic_CriticalAcuity` — CKM 4c-HFrEF, post-discharge 10d, slope -2.8, accelerating, eGFR 28, weight +2.5kg → score ≥85, tier CRITICAL, escalation SAFETY
4. `TestPAI_DominantDimension_Identified` — high velocity, everything else low → dominant = VELOCITY, contribution ≥50%
5. `TestPAI_NoData_MinimalAcuity` — empty input → score <15, data freshness STALE
6. `TestPAI_SignificantChange_Detected` — PAI jump from 30 to 65+ → SignificantChange=true

- [ ] **Step 2:** Implement `PAIConfig` struct with all dimension thresholds (from YAML), `LoadPAIConfig(path) (PAIConfig, error)` YAML loader with `DefaultPAIConfig()` fallback, and `ComputePAI(input PAIDimensionInput, cfg *PAIConfig) PAIScore` that: calls all 5 dimension scorers, applies dimension weights, computes composite, determines tier from boundaries, identifies dominant dimension and contribution %, generates primary reason + suggested action + suggested timeframe + escalation tier from the dominant dimension, computes data freshness.

- [ ] **Step 3:** Run tests — all 6 pass.

- [ ] **Step 4:** Commit: `feat(kb26): PAI composite engine — 5-dimension weighted scoring (PAI Task 7)`

---

### Task 8: PAI repository + event trigger + rate limiting

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/services/pai_repository.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/pai_event_trigger.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/pai_event_trigger_test.go`

- [ ] **Step 1:** Create `PAIRepository` with: `SaveScore(score PAIScore) error` (persists to pai_scores + pai_history), `FetchLatest(patientID) (*PAIScore, error)`, `FetchTrend(patientID, limit int) ([]PAIHistory, error)`.

- [ ] **Step 2:** Write 4 failing tests for `PAIEventTrigger`:
1. `TestTrigger_RateLimitBlocks` — two computes within 15 minutes → second blocked
2. `TestTrigger_SignificantChange_Publishes` — score delta ≥10 → event published
3. `TestTrigger_TierChange_Publishes` — tier MODERATE→HIGH → event published
4. `TestTrigger_NoChange_Suppresses` — score delta <10, same tier → no event

- [ ] **Step 3:** Implement `PAIEventTrigger` with: in-memory rate limiter (`map[string]time.Time` with 15-minute TTL), `ShouldRecompute(patientID) bool`, `ProcessResult(current, previous PAIScore) *PAIChangeEvent` (returns event if significant, nil otherwise). The event trigger does NOT do the Kafka publish directly — it produces a `PAIChangeEvent` that the caller can publish. This keeps the trigger testable without Kafka.

- [ ] **Step 4:** Run tests — all 4 pass.

- [ ] **Step 5:** Commit: `feat(kb26): PAI repository + event trigger with rate limiting (PAI Task 8)`

---

### Task 9: API handlers + wiring into KB-26 server

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/api/pai_handlers.go`
- Modify: `kb-26-metabolic-digital-twin/internal/api/routes.go`
- Modify: `kb-26-metabolic-digital-twin/main.go`

- [ ] **Step 1:** Create `pai_handlers.go` with 3 handlers:
- `GET /api/v1/kb26/pai/:patientId` — returns latest PAI score from repository
- `GET /api/v1/kb26/pai/:patientId/history` — returns PAI trend (last 30 scores)
- `POST /api/v1/kb26/pai/:patientId/compute` — triggers recompute (accepts `PAIDimensionInput` as JSON body), returns computed `PAIScore`

- [ ] **Step 2:** Add PAI route group to `routes.go` inside the existing v1 group.

- [ ] **Step 3:** Add `PAIScore` + `PAIHistory` to AutoMigrate in `main.go`.

- [ ] **Step 4:** Wire `PAIEngine` + `PAIRepository` + `PAIEventTrigger` into Server struct with setter injection (same pattern as other KB-26 services).

- [ ] **Step 5:** Build: `go build ./...`

- [ ] **Step 6:** Commit: `feat(kb26): PAI API handlers + server wiring (PAI Task 9)`

---

### Task 10: KB-23 card prioritizer + integration test + final commit

**Files:**
- Create: `kb-23-decision-cards/internal/services/pai_card_prioritizer.go`
- Create: `kb-23-decision-cards/internal/services/pai_card_prioritizer_test.go`

- [ ] **Step 1:** Write 4 tests:
1. `TestPrioritize_CriticalFirst` — 3 patients with PAI 85/45/72 → sorted 85, 72, 45
2. `TestPrioritize_EqualPAI_StableOrder` — patients with same PAI maintain original order
3. `TestPrioritize_MissingPAI_Last` — patients without PAI score sorted to end
4. `TestPrioritize_Empty_NoError` — empty input → empty output

- [ ] **Step 2:** Implement `PrioritizeCards` and `AnnotateCardWithPAI` functions.

- [ ] **Step 3:** Full test sweep across KB-26, KB-23.

- [ ] **Step 4:** Verify all YAML parses correctly.

- [ ] **Step 5:** Commit: `feat: complete PAI patient acuity index`

- [ ] **Step 6:** Push to origin.

---

## Verification Questions

1. Does a severely declining patient (slope -2.5, concordant) score ≥85 on velocity? (yes / test)
2. Does an eGFR of 32 (2 from threshold) score ≥80 on proximity? (yes / test)
3. Does exponential scaling make 90% proximity >2× more urgent than 50%? (yes / test)
4. Does a disengaged patient who stopped measuring score ≥95 on behavioral? (yes / test)
5. Does a CKM 4c-HFrEF post-discharge patient with NYHA III approach context score 100? (yes / test)
6. Does 90 days without clinician contact produce attention score ≥90? (yes / test)
7. Does Rajesh Kumar's full scenario produce PAI ≥65 (tier HIGH)? (yes / test)
8. Does the acute-on-chronic HFrEF scenario produce PAI ≥85 (tier CRITICAL)? (yes / test)
9. Does seasonal dampening cap velocity at ≤60 during confounder windows? (yes / test)
10. Does the rate limiter block recomputes within 15 minutes? (yes / test)
11. Does the card prioritizer sort by PAI descending? (yes / test)
12. Are all KB-26 + KB-23 test suites green? (yes / sweep)

---

## Effort Estimate

| Task | Scope | Expected |
|---|---|---|
| Task 1: Models + YAML configs | 4 files, ~400 LOC models + ~300 LOC YAML | 1-2 hours |
| Task 2: Velocity dimension | 2 files, ~150 LOC + 6 tests | 1-2 hours |
| Task 3: Proximity dimension | 2 files, ~200 LOC + 6 tests | 1-2 hours |
| Task 4: Behavioral dimension | 2 files, ~120 LOC + 5 tests | 1 hour |
| Task 5: Context dimension | 2 files, ~120 LOC + 5 tests | 1 hour |
| Task 6: Attention dimension | 2 files, ~120 LOC + 5 tests | 1 hour |
| Task 7: PAI composite engine | 2 files, ~250 LOC + 6 tests | 2-3 hours |
| Task 8: Repository + trigger | 3 files, ~200 LOC + 4 tests | 1-2 hours |
| Task 9: API handlers + wiring | 3 files modified/created, ~200 LOC | 1-2 hours |
| Task 10: KB-23 prioritizer + integration | 2 files + sweep, ~100 LOC + 4 tests | 1-2 hours |
| **Total** | **~26 files, ~2200 LOC, ~52 tests** | **~10-16 hours** |
