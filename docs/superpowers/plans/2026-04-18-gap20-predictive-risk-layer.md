# Gap 20: Predictive Risk Layer — Implementation Plan (Sprint 1: Rule-Based Predictor)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a rule-based risk predictor that identifies stable-looking patients likely to deteriorate in the next 30 days — enabling proactive outreach before acute detection triggers — using clinical heuristics from existing PAI trajectories, domain slopes, and confounder context as a precursor to ML-based prediction in Sprint 2.

**Architecture:** The risk predictor lives in KB-26 (alongside PAI and domain trajectories) as a pure scoring function that produces a `PredictedRisk` score (0-100) with risk tier (HIGH/MODERATE/LOW) and contributing factors. A proactive outreach selector in KB-23 filters to patients who are stable-PAI but high-predicted-risk and surfaces them in a separate worklist view. The predictor uses 6 heuristic signals: declining trajectory slope, increasing PAI trend over 30 days, declining engagement trajectory, post-discharge window active, recent medication complexity increase, and elevated confounder burden. When ML models are trained (Sprint 2+), they replace the heuristic scoring while the rest of the infrastructure remains unchanged.

**Tech Stack:** Go 1.21 for KB-26 risk predictor + KB-23 proactive outreach. Existing PAI, domain trajectory, and confounder infrastructure as inputs. YAML config for heuristic weights. Same API contract that future ML predictions will implement.

---

## Existing Infrastructure

| What exists | Where | Relevance |
|---|---|---|
| PAI scores + history | KB-26 PAI engine | PAI trajectory trend (30-day slope) is a predictive signal |
| Domain decomposition slopes | KB-26 trajectory engine | Per-domain 30/90-day slopes indicate chronic trajectory |
| Second derivative (acceleration) | KB-26 trajectory engine | Decelerating improvement → risk of reversal |
| Engagement composite | KB-20 SummaryContext | Declining engagement precedes deterioration |
| Care transition active | KB-20 CareTransition | Post-discharge patients are inherently higher risk |
| Confounder burden | V4-8 confounder scorer | Active confounders increase outcome uncertainty |
| Medication changes | KB-20 intervention timeline | Recent changes correlate with instability |
| Worklist aggregator | KB-23 Gap 18 | Integration point for proactive outreach view |

## File Inventory

### KB-26 — Risk Predictor
| Action | File | Responsibility |
|---|---|---|
| Create | `internal/models/predicted_risk.go` | PredictedRisk, RiskFactor, PredictedRiskInput |
| Create | `internal/services/risk_predictor.go` | ComputePredictedRisk — 6-signal heuristic scorer |
| Create | `internal/services/risk_predictor_test.go` | 6 tests |
| Create | `internal/api/risk_handlers.go` | GET /risk/:patientId, POST /risk/batch |
| Modify | `internal/api/routes.go` | Add risk route group |
| Modify | `main.go` | AutoMigrate PredictedRisk |

### KB-23 — Proactive Outreach
| Action | File | Responsibility |
|---|---|---|
| Create | `internal/services/proactive_outreach_selector.go` | SelectProactiveOutreach — filters stable+high-risk patients |
| Create | `internal/services/proactive_outreach_selector_test.go` | 4 tests |
| Modify | `internal/api/worklist_handlers.go` | GET /worklist/proactive endpoint |

### Market Configs
| Action | File | Responsibility |
|---|---|---|
| Create | `market-configs/shared/predictive_risk_parameters.yaml` | Heuristic weights, risk tier thresholds, outreach caps |

**Total: 11 files (8 create, 3 modify), ~10 tests**

---

### Task 1: Predicted risk models + config YAML

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/models/predicted_risk.go`
- Create: `market-configs/shared/predictive_risk_parameters.yaml`

- [ ] **Step 1:** Create `predicted_risk.go` with:

```go
package models

import (
    "time"
    "github.com/google/uuid"
)

// RiskTier classifies predicted risk level.
type RiskTier string

const (
    RiskTierHigh     RiskTier = "HIGH"
    RiskTierModerate RiskTier = "MODERATE"
    RiskTierLow      RiskTier = "LOW"
)

// PredictedRisk is the output of the risk predictor for one patient.
type PredictedRisk struct {
    ID                  uuid.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
    PatientID           string       `gorm:"size:100;index;not null" json:"patient_id"`
    PredictionType      string       `gorm:"size:30;not null" json:"prediction_type"` // DETERIORATION_30D
    RiskScore           float64      `json:"risk_score"`             // 0-100
    RiskTier            string       `gorm:"size:10" json:"risk_tier"`
    PrimaryDrivers      []RiskFactor `gorm:"-" json:"primary_drivers"`
    ModifiableDrivers   []RiskFactor `gorm:"-" json:"modifiable_drivers"`
    RiskSummary         string       `gorm:"type:text" json:"risk_summary"`
    RecommendedAction   string       `gorm:"type:text" json:"recommended_action"`
    CounterfactualReduction float64  `json:"counterfactual_reduction"` // estimated risk reduction with intervention
    PredictionWindowDays int         `gorm:"default:30" json:"prediction_window_days"`
    ModelType           string       `gorm:"size:20;default:'HEURISTIC'" json:"model_type"` // HEURISTIC or ML
    ComputedAt          time.Time    `gorm:"not null" json:"computed_at"`
    ExpiresAt           time.Time    `json:"expires_at"`
    CreatedAt           time.Time    `gorm:"autoCreateTime" json:"created_at"`
}

func (PredictedRisk) TableName() string { return "predicted_risks" }

// RiskFactor is one contributing factor to a predicted risk score.
type RiskFactor struct {
    FactorName       string  `json:"factor_name"`
    FactorValue      float64 `json:"factor_value"`
    Contribution     float64 `json:"contribution"`      // points contributed to risk score
    Direction        string  `json:"direction"`          // INCREASES_RISK, DECREASES_RISK
    Modifiable       bool    `json:"modifiable"`
    Interpretation   string  `json:"interpretation"`     // human-readable
    RecommendedAction string `json:"recommended_action,omitempty"`
}

// PredictedRiskInput carries all inputs for risk prediction.
type PredictedRiskInput struct {
    PatientID string

    // Trajectory signals (from domain decomposition)
    CompositeSlope30d    *float64 // MHRI 30-day slope
    WorstDomainSlope30d  *float64 // most declining domain
    SecondDerivative     *string  // ACCELERATING_DECLINE, DECELERATING_DECLINE, etc.
    DomainsDeterioring   int

    // PAI signals
    PAIScore             float64
    PAITrend30d          *float64 // PAI slope over last 30 days (positive = rising = worsening)
    PAICriticalCount90d  int      // count of CRITICAL transitions in 90 days

    // Engagement signals
    EngagementComposite  *float64
    EngagementTrend30d   *float64 // negative = declining engagement
    MeasurementFreqDrop  float64  // 0-1, percentage drop from average

    // Clinical context
    IsPostDischarge      bool
    DaysSinceDischarge   int
    MedicationChanges30d int
    PolypharmacyCount    int
    CKMStage             string
    Age                  int

    // Confounder context
    ActiveConfounderScore float64
    SeasonalWindow        bool
}
```

- [ ] **Step 2:** Create `predictive_risk_parameters.yaml`:
```yaml
# Predictive Risk Layer — Sprint 1 heuristic weights.
# These weights approximate ML model behavior using clinical heuristics.
# Replaced by trained model weights in Sprint 2+ when outcome data is available.

heuristic_weights:
  trajectory_declining:
    weight: 25
    threshold_slope: -0.5   # composite slope below this contributes
    max_contribution: 25
    modifiable: false
    interpretation: "Clinical trajectory has been declining"

  pai_trend_rising:
    weight: 20
    threshold_slope: 0.3    # PAI 30d slope above this contributes
    max_contribution: 20
    modifiable: false
    interpretation: "Urgency score has been rising over 30 days"

  engagement_declining:
    weight: 20
    threshold_drop: 0.20    # engagement drop >20% contributes
    max_contribution: 20
    modifiable: true
    interpretation: "Patient engagement declining — proactive outreach may help"
    action: "Schedule engagement re-establishment outreach within 7 days"

  post_discharge_window:
    weight: 15
    max_days: 30
    max_contribution: 15
    modifiable: false
    interpretation: "Patient is in post-discharge high-risk window"

  medication_complexity:
    weight: 10
    threshold_changes: 2    # >2 medication changes in 30d
    threshold_polypharmacy: 8
    max_contribution: 10
    modifiable: true
    interpretation: "Recent medication changes or high polypharmacy"
    action: "Schedule medication reconciliation review"

  confounder_burden:
    weight: 10
    threshold_score: 0.3
    max_contribution: 10
    modifiable: false
    interpretation: "Active confounders may be affecting clinical trajectory"

risk_tiers:
  high: 50       # score >= 50 = HIGH
  moderate: 25   # score >= 25 = MODERATE
  low: 0         # below 25 = LOW

counterfactual_estimate:
  engagement_intervention_reduction: 8   # estimated risk points reduced by re-engagement
  medication_review_reduction: 5         # estimated reduction by medication review

outreach:
  max_per_clinician_per_day: 8
  min_risk_score: 25            # only surface patients with score >= 25
  exclude_pai_tiers: [CRITICAL, HIGH]  # already in urgent worklist
  cooldown_days: 14             # don't re-surface same patient within 14 days

prediction_window_days: 30
prediction_expiry_hours: 24
```

- [ ] **Step 3:** Verify compile + YAML parse. Commit: `feat(kb26): predicted risk models + heuristic config (Gap 20 Task 1)`

---

### Task 2: Rule-based risk predictor — 6-signal heuristic

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/services/risk_predictor.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/risk_predictor_test.go`

- [ ] **Step 1:** Write 6 tests:
1. `TestRisk_DecliningTrajectory_HighContribution` — composite slope -1.2 → trajectory factor contributes ~25 points
2. `TestRisk_RisingPAITrend_Contribution` — PAI 30d slope +0.8 → PAI trend contributes ~20 points
3. `TestRisk_DecliningEngagement_ModifiableDriver` — engagement drop 40% → engagement factor contributes ~20, flagged as modifiable with recommended action
4. `TestRisk_PostDischarge_Contribution` — within 30d of discharge → post-discharge contributes ~15
5. `TestRisk_StablePatient_LowRisk` — all signals normal → risk score <15, tier LOW
6. `TestRisk_CompoundRisk_HighTier` — declining trajectory + rising PAI + declining engagement → score ≥50, tier HIGH, multiple drivers listed

- [ ] **Step 2:** Implement:

```go
// ComputePredictedRisk produces a heuristic-based 30-day deterioration risk score.
// In Sprint 2+, this is replaced by ML model inference with the same API contract.
func ComputePredictedRisk(input PredictedRiskInput) models.PredictedRisk
```

**6 scoring signals (each produces 0-max_contribution points):**

1. **Trajectory declining** (max 25): if CompositeSlope30d < -0.5, score = min(abs(slope) / 2.0 * 25, 25). Worse slope = higher score.

2. **PAI trend rising** (max 20): if PAITrend30d > 0.3, score = min(trend / 1.0 * 20, 20). PAI rising = urgency increasing over time.

3. **Engagement declining** (max 20, modifiable): if EngagementTrend30d < -0.20 or MeasurementFreqDrop > 0.20, score = min(drop * 100 / 50 * 20, 20). Modifiable=true, action="Schedule engagement re-establishment outreach."

4. **Post-discharge window** (max 15): if IsPostDischarge and DaysSinceDischarge ≤ 30, score = 15 * (1 - DaysSinceDischarge/30). Higher earlier in window.

5. **Medication complexity** (max 10, modifiable): if MedicationChanges30d > 2 or PolypharmacyCount > 8, score = min((changes + polypharmacy/4) * 2, 10). Modifiable=true, action="Schedule medication reconciliation review."

6. **Confounder burden** (max 10): if ActiveConfounderScore > 0.3, score = min(confScore * 20, 10).

**Composite:** sum of 6 signals, cap at 100.
**Tier:** ≥50 → HIGH, ≥25 → MODERATE, else LOW.
**Counterfactual:** sum of modifiable factor reductions (engagement 8pts, medication 5pts if applicable).
**RiskSummary:** "Moderate risk of clinical deterioration in 30 days. Primary drivers: declining trajectory, declining engagement."
**RecommendedAction:** from top modifiable driver.

- [ ] **Step 3:** Run tests — all 6 pass. Commit: `feat(kb26): rule-based risk predictor — 6-signal heuristic (Gap 20 Task 2)`

---

### Task 3: Risk API handlers + KB-26 wiring

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/api/risk_handlers.go`
- Modify: `kb-26-metabolic-digital-twin/internal/api/routes.go`
- Modify: `kb-26-metabolic-digital-twin/main.go`

- [ ] **Step 1:** Create 2 handlers:
```go
// GET /api/v1/kb26/risk/:patientId — current predicted risk for a patient
func (s *Server) getPredictedRisk(c *gin.Context)

// POST /api/v1/kb26/risk/batch — batch prediction for multiple patients
// Body: { "patient_ids": ["P1", "P2", ...] }
func (s *Server) batchPredictRisk(c *gin.Context)
```

The handlers construct `PredictedRiskInput` from available data (PAI score from PAI repo, summary context from KB-20 if available, domain slopes if available — graceful degradation for missing data).

- [ ] **Step 2:** Add routes + AutoMigrate PredictedRisk.

- [ ] **Step 3:** Build + test. Commit: `feat(kb26): predicted risk API — single + batch endpoints (Gap 20 Task 3)`

---

### Task 4: Proactive outreach selector in KB-23

**Files:**
- Create: `kb-23-decision-cards/internal/services/proactive_outreach_selector.go`
- Create: `kb-23-decision-cards/internal/services/proactive_outreach_selector_test.go`

- [ ] **Step 1:** Write 4 tests:
1. `TestOutreach_HighRiskStable_Included` — PAI LOW + predicted risk HIGH → included in outreach
2. `TestOutreach_PAICritical_Excluded` — PAI CRITICAL + predicted risk HIGH → excluded (already urgent)
3. `TestOutreach_RecentlyContacted_Excluded` — contacted 5 days ago for same driver → excluded (14-day cooldown)
4. `TestOutreach_DailyCap_Enforced` — 15 eligible patients, cap=8 → only top 8 by risk score returned

- [ ] **Step 2:** Implement:
```go
// ProactiveOutreachItem mirrors WorklistItem for the proactive view.
type ProactiveOutreachItem struct {
    PatientID          string  `json:"patient_id"`
    PatientName        string  `json:"patient_name"`
    RiskScore          float64 `json:"risk_score"`
    RiskTier           string  `json:"risk_tier"`
    PrimaryReason      string  `json:"primary_reason"`
    RecommendedAction  string  `json:"recommended_action"`
    CounterfactualReduction float64 `json:"counterfactual_reduction"`
    ModifiableDrivers  []string `json:"modifiable_drivers"`
}

// SelectProactiveOutreach returns patients eligible for proactive outreach.
func SelectProactiveOutreach(
    predictions []models.PredictedRisk,
    currentPAITiers map[string]string,  // patientID → PAI tier
    lastContactDays map[string]int,     // patientID → days since last proactive contact
    maxItems int,
    cooldownDays int,
) []ProactiveOutreachItem
```

Eligibility: RiskScore ≥ 25 AND PAI tier not CRITICAL/HIGH AND lastContact ≥ cooldownDays (or no prior contact). Sort by RiskScore descending. Truncate to maxItems.

- [ ] **Step 3:** Run tests — all 4 pass. Commit: `feat(kb23): proactive outreach selector (Gap 20 Task 4)`

---

### Task 5: Worklist proactive outreach endpoint + integration

**Files:**
- Modify: `kb-23-decision-cards/internal/api/worklist_handlers.go`
- Modify: `kb-23-decision-cards/internal/api/routes.go`

- [ ] **Step 1:** Add handler:
```go
// GET /api/v1/worklist/proactive?clinician_id=X&patient_ids=P1,P2,...
// Returns proactive outreach candidates from the predictive risk layer.
func (s *Server) getProactiveWorklist(c *gin.Context)
```

The handler: parse patient IDs → for each patient, call KB-26 risk API (or compute inline with available data) → call SelectProactiveOutreach → return sorted list.

- [ ] **Step 2:** Add route: `worklist.GET("/proactive", s.getProactiveWorklist)`

- [ ] **Step 3:** Build + full test sweep KB-23 + KB-26.

- [ ] **Step 4:** Commit: `feat: complete Gap 20 predictive risk layer (Sprint 1 — rule-based)`

- [ ] **Step 5:** Push to origin.

---

## Verification Questions

1. Does a declining trajectory (slope -1.2) contribute ~25 points? (yes / test)
2. Does a rising PAI trend contribute ~20 points? (yes / test)
3. Does declining engagement surface as a modifiable driver? (yes / test)
4. Does a stable patient with all normal signals score LOW? (yes / test)
5. Does a compound-risk patient (3+ signals) score HIGH? (yes / test)
6. Does proactive outreach exclude PAI CRITICAL patients? (yes / test)
7. Does the 14-day cooldown prevent re-surfacing? (yes / test)
8. Does the daily cap enforce maximum items? (yes / test)

## Effort Estimate

| Task | Scope | Expected |
|---|---|---|
| Task 1: Models + YAML | 2 files | 1-2 hours |
| Task 2: Risk predictor (6 tests) | 2 files | 2-3 hours |
| Task 3: Risk API + wiring | 3 files | 1-2 hours |
| Task 4: Proactive outreach (4 tests) | 2 files | 1-2 hours |
| Task 5: Worklist integration | 2 files modified | 1 hour |
| **Total** | **~11 files, ~10 tests** | **~7-10 hours** |

---

## Sprint 2 Deferred Items (Require Outcome Data)

| Component | Reason | When |
|-----------|--------|------|
| KB-28 Python ML service | Needs training data | After 6 months of pilot |
| Training dataset builder | Needs outcome labels (T4 from Gap 19) | After 6 months |
| XGBoost/LightGBM models | Needs training data | After 6 months |
| SHAP feature explanations | Requires trained model | With KB-28 |
| Isotonic calibration | Requires trained model | With KB-28 |
| Model registry + A/B testing | Requires multiple model versions | With KB-28 |
| Fairness audit across cohorts | Requires sufficient per-cohort data | After 9 months |
| Confidence intervals | Requires bootstrap on training distribution | With KB-28 |
| PAI velocity integration (predictive boost) | Needs validated predictions first | After model validation |

**Transition plan:** When ML models are trained, they implement the same `PredictedRisk` output struct. The `ComputePredictedRisk` function is replaced by `MLPredict` which calls the KB-28 inference API. The proactive outreach selector, worklist integration, and API endpoints remain unchanged — same contract, better predictions.
