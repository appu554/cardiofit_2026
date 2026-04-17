# V4-8 IOR Confounder Scoring Enhancement — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the binary confounder flags (isAcuteIll / hasRecentTransfusion / hasRecentHypoglycaemia) with a multi-dimensional confounder scoring engine that accounts for seasonal variation, religious fasting, festival dietary changes, acute illness, iatrogenic confounders (steroids), and lifestyle disruption — producing a 0-1 composite quality-of-evidence score with human-readable narrative for every IOR outcome record.

**Architecture:** Three new components in KB-20: (1) ConfounderCalendar loads market-specific YAML calendars and returns active confounders for any date window with religious/regional filtering, overlap-proportional weighting, and washout periods; (2) ClinicalEventDetector scans patient safety events and medication records for steroids, hospitalization, infection, and AKI; (3) EnhancedConfounderScorer integrates all factor sources into a composite score with per-category subscores, outcome-type relevance filtering, deferral recommendations, and narrative generation. KB-23 gains an evidence quality filter that annotates card evidence chains with confounder context.

**Tech Stack:** Go 1.21 (Gin, GORM) for KB-20 services. PostgreSQL 15 for clinical event queries. YAML market configs for confounder calendars and weights. Existing SafetyEvent audit trail + MedicationState for clinical event sourcing.

---

## Scope Check

This spec covers one cohesive subsystem: the enhanced confounder scoring engine. It touches two services (KB-20 for the scoring engine, KB-23 for the evidence filter) and market-configs for calendar data. One plan is appropriate.

**Important codebase note:** The spec references modifying an existing `ior_generator.go` with `ComputeConfounderScore`. This file does **not exist** in the codebase. The current confounder infrastructure is limited to `SafetyEventRecorder.ConfounderFlags()` which produces three boolean flags. This plan builds the confounder scoring engine as a standalone service that can be integrated into a future IOR generator. Task 5 provides the integration bridge via an `IORConfounderAssessor` that combines all three data sources.

## Existing Infrastructure

| What exists | Where | Relevance |
|---|---|---|
| `SafetyEvent` model (6 event types) | KB-20 `internal/models/safety_event.go` | Clinical events (ACUTE_ILLNESS, BLOOD_TRANSFUSION, HYPO_EVENT, EGFR_CRITICAL, POTASSIUM_HIGH, HYPERTENSIVE_CRISIS) — confounder signal source |
| `SafetyEventRecorder.ConfounderFlags()` | KB-20 `internal/services/safety_event_recorder.go` | Binary confounder flags with sliding windows (7d/90d/30d). V4-8 produces richer 0-1 scores but does NOT replace this — it adds a parallel scoring path |
| `MedicationState` model | KB-20 `internal/models/medication_state.go` | DrugName, DrugClass, StartDate, IsActive — source for steroid detection and concurrent medication counting |
| `InterventionTimelineService` | KB-20 `internal/services/intervention_timeline.go` | ByDomain medication actions — source for concurrent intervention counting |
| `seasonal_calendar.yaml` | `market-configs/india/seasonal_calendar.yaml` | Existing seasonal windows for trajectory suppression (Diwali, summer_heat). V4-8 adds a separate confounder calendar with different schema (weights, washouts, religious affiliation) |
| `SummaryContext` wire contract | KB-20 `internal/services/summary_context_service.go` | Already carries confounder flags. V4-8 can extend this with the enhanced score when the IOR generator is built |
| `PatientProfile.PhenotypeCluster` | KB-20 `internal/models/patient_profile.go:61` | V4-7 stable cluster — the spec mentions cluster stability as IOR evidence quality signal |

## File Inventory

### KB-20 (Patient Profile) — Confounder Scoring Engine
| Action | File | Responsibility |
|---|---|---|
| Create | `internal/models/confounder.go` | Types: `ConfounderCategory` (9 constants), `ConfounderFactor` (12 fields), `EnhancedConfounderResult` (composite + subscores + narrative + deferral), `CalendarEvent`, `ClinicalEventConfounder`, `OutcomeConfounderAnnotation` |
| Create | `internal/services/confounder_calendar.go` | `ConfounderCalendar.FindActiveConfounders(start, end, religion, region) → []ConfounderFactor`, `GetSeasonalHbA1cAdjustment(month) → float64`, YAML loader |
| Create | `internal/services/confounder_calendar_test.go` | 8 tests: Ramadan overlap, Ramadan washout, Diwali season, monsoon Mumbai regional, non-Muslim skips Ramadan, Australia winter, no overlap empty, seasonal HbA1c adjustment |
| Create | `internal/services/clinical_event_confounder.go` | `ClinicalEventDetector.DetectConfounders(events, start, end) → []ConfounderFactor` — steroid courses, hospitalization, antibiotic→infection, AKI from creatinine spike |
| Create | `internal/services/clinical_event_confounder_test.go` | 6 tests: steroid course, steroid washout, hospitalization, antibiotic infection, AKI creatinine spike, no events empty |
| Create | `internal/services/enhanced_confounder_scorer.go` | `EnhancedConfounderScorer.Compute(input) → EnhancedConfounderResult` — 4 subscores (medication, calendar, clinical event, lifestyle), category caps, outcome-type filtering, deferral logic, narrative |
| Create | `internal/services/enhanced_confounder_scorer_test.go` | 8 tests: no confounders, medication-only backward compatible, Ramadan during window, steroid high confounder, multiple factors compound, irrelevant confounder not counted, defer during Ramadan, narrative contains all factors |
| Create | `internal/services/ior_confounder_assessor.go` | `IORConfounderAssessor.Assess(patientID, windowStart, windowEnd, outcomeType) → EnhancedConfounderResult` — integration bridge combining calendar + clinical events + medication count + engagement |

### Market Configs — Confounder Calendars
| Action | File | Responsibility |
|---|---|---|
| Create | `market-configs/shared/confounder_weights.yaml` | Category caps (9 categories), medication detail (per-change weight, dose vs class), adherence scaling, clinical event weights (steroid/hosp/infection/surgery/AKI with washouts), lifestyle weights, deferral thresholds |
| Create | `market-configs/india/confounder_calendar.yaml` | Religious: Ramadan (30d, IDF-DAR 2021), Navratri Chaitra + Sharad. Festival: Diwali season (21d), wedding season (120d), Pongal. Seasonal: monsoon (regional), extreme summer (regional). Seasonal HbA1c tropical pattern. |
| Create | `market-configs/australia/confounder_calendar.yaml` | Festival: Christmas/NYE (21d), Easter holidays. Seasonal: winter temperate (90d, Tseng 2005), bushfire season (120d, McArthur 2021). Indigenous: Sorry Business, ceremony season. Seasonal HbA1c temperate southern hemisphere. |

### KB-23 (Decision Cards) — Evidence Quality Filter
| Action | File | Responsibility |
|---|---|---|
| Create | `internal/services/confounder_evidence_filter.go` | `FilterIORByConfidence(outcomes, minConfidence) → filtered`, `AnnotateEvidenceWithConfounderContext(summary, counts) → annotated` |
| Create | `internal/services/confounder_evidence_filter_test.go` | 4 tests: high-confidence-only filter, moderate-and-above filter, high quality annotation, low quality annotation |

**Total: 14 files (14 create, 0 modify)**

---

### Task 1: Create confounder models + YAML configs

**Files:**
- Create: `kb-20-patient-profile/internal/models/confounder.go`
- Create: `market-configs/shared/confounder_weights.yaml`
- Create: `market-configs/india/confounder_calendar.yaml`
- Create: `market-configs/australia/confounder_calendar.yaml`

- [ ] **Step 1:** Create `confounder.go` with these types:

```go
package models

import "time"

// ConfounderCategory classifies the type of confounder.
type ConfounderCategory string

const (
    ConfounderMedication    ConfounderCategory = "MEDICATION"
    ConfounderAdherence     ConfounderCategory = "ADHERENCE"
    ConfounderSeasonal      ConfounderCategory = "SEASONAL"
    ConfounderReligiousFast ConfounderCategory = "RELIGIOUS_FASTING"
    ConfounderFestivalDiet  ConfounderCategory = "FESTIVAL_DIETARY"
    ConfounderAcuteIllness  ConfounderCategory = "ACUTE_ILLNESS"
    ConfounderIatrogenic    ConfounderCategory = "IATROGENIC"
    ConfounderLifestyle     ConfounderCategory = "LIFESTYLE"
    ConfounderEnvironmental ConfounderCategory = "ENVIRONMENTAL"
)

// ConfounderFactor represents a single active confounder during an outcome window.
type ConfounderFactor struct {
    Category          ConfounderCategory `json:"category"`
    Name              string             `json:"name"`
    Weight            float64            `json:"weight"`
    AffectedOutcomes  []string           `json:"affected_outcomes"`
    ExpectedDirection string             `json:"expected_direction"`
    ExpectedMagnitude string             `json:"expected_magnitude"`
    WindowStart       time.Time          `json:"window_start"`
    WindowEnd         time.Time          `json:"window_end"`
    OverlapDays       int                `json:"overlap_days"`
    OverlapPct        float64            `json:"overlap_pct"`
    Source            string             `json:"source"`
    Confidence        string             `json:"confidence"`
}

// EnhancedConfounderResult is the full output of the enhanced confounder scorer.
type EnhancedConfounderResult struct {
    CompositeScore       float64            `json:"composite_score"`
    ConfidenceLevel      string             `json:"confidence_level"`
    MedicationScore      float64            `json:"medication_score"`
    CalendarScore        float64            `json:"calendar_score"`
    ClinicalEventScore   float64            `json:"clinical_event_score"`
    LifestyleScore       float64            `json:"lifestyle_score"`
    ActiveFactors        []ConfounderFactor `json:"active_factors"`
    FactorCount          int                `json:"factor_count"`
    Narrative            string             `json:"narrative"`
    ShouldDefer          bool               `json:"should_defer"`
    DeferReasonCode      string             `json:"defer_reason_code,omitempty"`
    SuggestedRecheckWeeks int               `json:"suggested_recheck_weeks,omitempty"`
}

// CalendarEvent represents a seasonal/religious/cultural event in the confounder calendar.
type CalendarEvent struct {
    Name              string   `yaml:"name" json:"name"`
    Category          string   `yaml:"category" json:"category"`
    RecurrenceType    string   `yaml:"recurrence_type" json:"recurrence_type"`
    GregorianApprox   []int    `yaml:"gregorian_approx_month" json:"gregorian_approx_month,omitempty"`
    DurationDays      int      `yaml:"duration_days" json:"duration_days"`
    AffectedOutcomes  []string `yaml:"affected_outcomes" json:"affected_outcomes"`
    ExpectedDirection string   `yaml:"expected_direction" json:"expected_direction"`
    ExpectedMagnitude string   `yaml:"expected_magnitude" json:"expected_magnitude"`
    BaseWeight        float64  `yaml:"base_weight" json:"base_weight"`
    PostEventWashout  int      `yaml:"post_event_washout_days" json:"post_event_washout_days"`
    AppliesTo         string   `yaml:"applies_to" json:"applies_to"`
    Notes             string   `yaml:"notes" json:"notes,omitempty"`
}

// ClinicalEventConfounder represents an acute clinical event that confounds outcomes.
type ClinicalEventConfounder struct {
    EventType         string   `json:"event_type"`
    DetectionMethod   string   `json:"detection_method"`
    AffectedOutcomes  []string `json:"affected_outcomes"`
    ExpectedDirection string   `json:"expected_direction"`
    Weight            float64  `json:"weight"`
    WashoutDays       int      `json:"washout_days"`
}

// OutcomeConfounderAnnotation extends an outcome record with confounder detail.
type OutcomeConfounderAnnotation struct {
    OutcomeID          string                  `json:"outcome_id"`
    ConfounderResult   EnhancedConfounderResult `json:"confounder_result"`
    AdjustedConfidence string                  `json:"adjusted_confidence"`
    OriginalDelta      float64                 `json:"original_delta"`
}
```

- [ ] **Step 2:** Create `confounder_weights.yaml` with category caps (MEDICATION: 0.40, ADHERENCE: 0.25, SEASONAL: 0.15, RELIGIOUS_FASTING: 0.25, FESTIVAL_DIETARY: 0.15, ACUTE_ILLNESS: 0.35, IATROGENIC: 0.40, LIFESTYLE: 0.20, ENVIRONMENTAL: 0.10), medication detail (per_concurrent_change: 0.12, max: 0.40, dose_change: 0.08, class_change: 0.15), adherence (per_10pct_drop: 0.10, max: 0.25), clinical event weights (steroid: 0.35/28d, hospitalization: 0.30/42d, infection: 0.20/21d, surgery: 0.30/56d, AKI: 0.40/90d), lifestyle (engagement_collapse: 0.15, exercise_cessation: 0.10, dietary_shift: 0.10), and deferral thresholds (composite: 0.70, steroid_active: true, ramadan_glucose: true, hospitalization_washout: true). Copy the exact YAML from the spec lines 257-349.

- [ ] **Step 3:** Create `india/confounder_calendar.yaml` with religious events (Ramadan 30d with 2026/2027 dates + 28d washout, Navratri Chaitra 9d, Navratri Sharad 9d), festival events (Diwali season 21d, wedding season 120d Oct-Jan, Pongal/Makar Sankranti 4d), seasonal events (monsoon 120d Jun-Sep regional [MUMBAI, KOLKATA, CHENNAI, KERALA, GOA], extreme summer 90d Apr-Jun regional [RAJASTHAN, MADHYA_PRADESH, DELHI, UTTAR_PRADESH]), and seasonal HbA1c (tropical_mild, peak months [7,8,9], trough [1,2,3], magnitude 0.15%). Copy the exact YAML from the spec lines 353-495.

- [ ] **Step 4:** Create `australia/confounder_calendar.yaml` with festival events (Christmas/NYE 21d, Easter 14d), seasonal events (winter temperate 90d Jun-Aug regional [VIC, NSW, TAS, SA, ACT], bushfire season 120d Nov-Feb regional [NSW, VIC, SA, QLD]), indigenous events (Sorry Business 14d patient-specific, ceremony season 21d patient-specific), and seasonal HbA1c (temperate southern hemisphere, peak [7,8,9], trough [1,2,3], magnitude 0.30%). Copy the exact YAML from the spec lines 498-599.

- [ ] **Step 5:** Verify models compile: `go build ./...` in KB-20.

- [ ] **Step 6:** Verify all YAML parses: `python3 -c "import yaml; [yaml.safe_load(open(f)) for f in ['market-configs/shared/confounder_weights.yaml', 'market-configs/india/confounder_calendar.yaml', 'market-configs/australia/confounder_calendar.yaml']]"`

- [ ] **Step 7:** Commit: `feat(kb20): confounder models + market config calendars (V4-8 Task 1)`

---

### Task 2: Build ConfounderCalendar service

**Files:**
- Create: `kb-20-patient-profile/internal/services/confounder_calendar.go`
- Create: `kb-20-patient-profile/internal/services/confounder_calendar_test.go`

- [ ] **Step 1:** Write 8 failing tests from the spec (lines 607-801):

1. `TestCalendar_RamadanOverlap` — window Feb 1–Apr 30 2026 + MUSLIM patient → finds RAMADAN with overlap ≥28 days, weight >0.15, affects DELTA_HBA1C
2. `TestCalendar_RamadanWashout` — window Mar 20–Apr 20 2026 (Ramadan ended Mar 19, washout 28 days → Apr 16) → still detects Ramadan washout
3. `TestCalendar_DiwaliSeason` — window Oct 1–Dec 31 2026 + HINDU patient → finds both DIWALI_SEASON and NAVRATRI_SHARAD
4. `TestCalendar_MonsoonMumbai` — window Jun–Sep 2026 + MUMBAI → finds MONSOON_SEASON. Same window + DELHI → does NOT find MONSOON_SEASON (not in region list)
5. `TestCalendar_NonMuslimSkipsRamadan` — window Feb–Apr 2026 + HINDU → no RAMADAN factor
6. `TestCalendar_AustraliaWinter` — window Jun–Aug 2026 + VIC → finds WINTER_TEMPERATE with weight ≥0.10
7. `TestCalendar_NoOverlap_EmptyFactors` — window Mar 1–15 2026 + HINDU + DELHI → no major confounders (all weights <0.15)
8. `TestCalendar_SeasonalHbA1cAdjustment` — Australia calendar: July (winter) → positive adjustment; January (summer) → negative adjustment

Include `loadTestCalendar(t, market)` helper that calls `LoadConfounderCalendar("../../market-configs", market)` with a `require.NoError`.

- [ ] **Step 2:** Run tests to verify they all fail.

- [ ] **Step 3:** Implement `ConfounderCalendar` with:
- `LoadConfounderCalendar(configDir, market) → (*ConfounderCalendar, error)` — reads and unmarshals YAML
- `FindActiveConfounders(windowStart, windowEnd, religiousAffiliation, region) → []ConfounderFactor` — iterates all events, checks religious/regional applicability, resolves event dates (exact dates for 2026/2027, fallback to GregorianMonths), extends by washout, checks overlap, weight scales with overlap percentage
- `GetSeasonalHbA1cAdjustment(month) → float64` — returns ±magnitude for peak/trough months
- `resolveEventDates(event, year)` — tries exact dates_2026/dates_2027 first, falls back to gregorian_approx_month
- Helpers: `maxTime`, `minTime`, `containsString`

The YAML struct types: `calendarEventConfig` (with Dates2026/Dates2027 sub-structs), `seasonalHbA1cConfig`, `calendarYAML` (top-level with religious_events, festival_events, seasonal_events, indigenous_events, seasonal_hba1c).

Copy the exact implementation from the spec lines 805-1043.

- [ ] **Step 4:** Run tests — all 8 should pass.

- [ ] **Step 5:** Commit: `feat(kb20): confounder calendar service — seasonal/religious/cultural events (V4-8 Task 2)`

---

### Task 3: Build ClinicalEventDetector

**Files:**
- Create: `kb-20-patient-profile/internal/services/clinical_event_confounder.go`
- Create: `kb-20-patient-profile/internal/services/clinical_event_confounder_test.go`

- [ ] **Step 1:** Write 6 failing tests from the spec (lines 1076-1224):

1. `TestClinicalEvent_SteroidCourse` — Prednisolone start -20d / stop -6d within 30d window → STEROID_COURSE factor with category IATROGENIC, weight ≥0.30, affects DELTA_HBA1C
2. `TestClinicalEvent_SteroidWashout` — Prednisolone ended 10d ago, window is last 5d → still detects STEROID_COURSE (within 28d washout)
3. `TestClinicalEvent_Hospitalization` — HOSPITALIZATION event 15d ago within 30d window → weight ≥0.25
4. `TestClinicalEvent_AcuteInfection_DetectedByAntibiotic` — Amoxicillin start 12d ago → ACUTE_INFECTION factor
5. `TestClinicalEvent_AKI_DetectedByCreatinineSpike` — Creatinine 2.8 (10d ago) vs baseline 1.2 (60d ago) → ACUTE_KIDNEY_INJURY with weight ≥0.35, affects DELTA_EGFR
6. `TestClinicalEvent_NoEvents_EmptyFactors` — nil events → empty factors

Include `testConfounderWeights()` helper that returns `&ConfounderWeights{SteroidWeight: 0.35, SteroidWashoutDays: 28, HospWeight: 0.30, HospWashoutDays: 42, InfectionWeight: 0.20, InfectionWashoutDays: 21, AKIWeight: 0.40, AKIWashoutDays: 90, SurgeryWeight: 0.30, SurgeryWashoutDays: 56}`.

- [ ] **Step 2:** Run tests to verify they all fail.

- [ ] **Step 3:** Implement `ClinicalEventDetector` with:
- `PatientClinicalEvent` struct (Type, DrugName, LabType, Value, Date, Duration)
- `ConfounderWeights` struct (per-event-type weight + washout days)
- `NewClinicalEventDetector(weights) → *ClinicalEventDetector`
- `DetectConfounders(events, windowStart, windowEnd) → []ConfounderFactor`
- Drug pattern lists: `steroidPatterns` (prednisolone, prednisone, dexamethasone, methylprednisolone, hydrocortisone, cortisone, betamethasone, triamcinolone), `antibioticPatterns` (amoxicillin, azithromycin, ciprofloxacin, levofloxacin, doxycycline, cephalexin, trimethoprim, nitrofurantoin, metronidazole, clindamycin, augmentin, cefuroxime)
- AKI detection: creatinine ≥1.5× baseline (KDIGO definition)
- Helpers: `matchesAny`, `overlaps`, `computeOverlapDays`

Copy the exact implementation from the spec lines 1227-1493.

- [ ] **Step 4:** Run tests — all 6 should pass.

- [ ] **Step 5:** Commit: `feat(kb20): clinical event confounder detector — steroid/hospitalization/AKI (V4-8 Task 3)`

---

### Task 4: Build EnhancedConfounderScorer

**Files:**
- Create: `kb-20-patient-profile/internal/services/enhanced_confounder_scorer.go`
- Create: `kb-20-patient-profile/internal/services/enhanced_confounder_scorer_test.go`

- [ ] **Step 1:** Write 8 failing tests from the spec (lines 1519-1695):

1. `TestEnhancedScorer_NoConfounders` — empty input → score 0.0, confidence HIGH, no factors, no defer
2. `TestEnhancedScorer_MedicationOnly_BackwardCompatible` — 2 concurrent meds + 0.15 adherence drop → score >0 and <0.5, confidence MODERATE
3. `TestEnhancedScorer_RamadanDuringWindow` — RAMADAN factor (weight 0.25, overlap 80%, DELTA_HBA1C) → calendar score >0, composite ≥0.20, narrative contains "RAMADAN"
4. `TestEnhancedScorer_SteroidCourse_HighConfounder` — STEROID_COURSE factor (weight 0.35, DELTA_HBA1C) → clinical event score ≥0.30, confidence LOW, should defer
5. `TestEnhancedScorer_MultipleFactors_Compound` — 1 med + Diwali + engagement collapse → composite >0.30, factor count ≥3, narrative contains all factor names
6. `TestEnhancedScorer_IrrelevantConfounder_NotCounted` — monsoon (affects DELTA_WEIGHT/DELTA_SBP only) with outcomeType=DELTA_HBA1C → calendar score 0.0
7. `TestEnhancedScorer_DeferDuringActiveRamadan` — RAMADAN overlap 90% + DeferOnRamadan=true → ShouldDefer=true, DeferReasonCode="RAMADAN_ACTIVE", SuggestedRecheckWeeks>0
8. `TestEnhancedScorer_NarrativeContainsAllFactors` — 2 meds + RAMADAN + ACUTE_INFECTION → narrative contains "concurrent medication", "RAMADAN", "ACUTE_INFECTION"

- [ ] **Step 2:** Run tests to verify they all fail.

- [ ] **Step 3:** Implement `EnhancedConfounderScorer` with:
- `EnhancedConfounderInput` struct (ConcurrentMedCount, AdherenceDrop, CalendarFactors, ClinicalEventFactors, LifestyleFactors, OutcomeType, DeferOnRamadan, DeferOnSteroid)
- `NewEnhancedConfounderScorer() → *EnhancedConfounderScorer`
- `Compute(input) → EnhancedConfounderResult` with 4 subscores:
  1. Medication: `min(medCount × 0.12, 0.40) + min(adherenceDrop × 1.0, 0.25)`, capped at 0.45
  2. Calendar: sum of relevant factor weights (filtered by `affectsOutcome`), capped at 0.30
  3. Clinical events: sum of relevant factor weights, capped at 0.45. ClinicalEventScore ≥0.35 forces deferral
  4. Lifestyle: sum of factor weights, capped at 0.20
- Composite: `min(sum of subscores, 1.0)`, rounded to 2 decimals
- Confidence: <0.20 → HIGH, 0.20-0.50 → MODERATE, >0.50 → LOW
- Deferral: Ramadan overlap >50% + DeferOnRamadan, steroid active + DeferOnSteroid, clinical event score ≥0.35
- Narrative: "Outcome confidence {level} (score {score}). Active confounders: {names}."
- `affectsOutcome(affectedOutcomes, outcomeType) → bool`

Copy the exact implementation from the spec lines 1698-1878.

- [ ] **Step 4:** Run tests — all 8 should pass.

- [ ] **Step 5:** Commit: `feat(kb20): enhanced confounder scorer — 4-subscore composite with deferral (V4-8 Task 4)`

---

### Task 5: Build IOR confounder assessor + KB-23 evidence filter

**Files:**
- Create: `kb-20-patient-profile/internal/services/ior_confounder_assessor.go`
- Create: `kb-23-decision-cards/internal/services/confounder_evidence_filter.go`
- Create: `kb-23-decision-cards/internal/services/confounder_evidence_filter_test.go`

- [ ] **Step 1:** Create `ior_confounder_assessor.go` — the integration bridge that combines all three confounder data sources for a given patient + outcome window:

```go
package services

import (
    "time"

    "go.uber.org/zap"
    "gorm.io/gorm"

    "kb-patient-profile/internal/models"
)

// IORConfounderAssessor combines the confounder calendar, clinical event
// detector, and enhanced scorer into a single Assess call. This is the
// integration point that a future IOR generator will call to annotate
// each outcome record with a confounder quality-of-evidence score.
type IORConfounderAssessor struct {
    calendar        *ConfounderCalendar
    eventDetector   *ClinicalEventDetector
    scorer          *EnhancedConfounderScorer
    db              *gorm.DB
    log             *zap.Logger
}

func NewIORConfounderAssessor(
    calendar *ConfounderCalendar,
    eventDetector *ClinicalEventDetector,
    db *gorm.DB,
    log *zap.Logger,
) *IORConfounderAssessor {
    return &IORConfounderAssessor{
        calendar:      calendar,
        eventDetector: eventDetector,
        scorer:        NewEnhancedConfounderScorer(),
        db:            db,
        log:           log,
    }
}

// Assess computes the full confounder result for a patient's outcome window.
func (a *IORConfounderAssessor) Assess(
    patientID string,
    windowStart, windowEnd time.Time,
    outcomeType string,
    religiousAffiliation string,
    region string,
    concurrentMedCount int,
    adherenceDrop float64,
) models.EnhancedConfounderResult {
    // 1. Calendar confounders
    var calendarFactors []models.ConfounderFactor
    if a.calendar != nil {
        calendarFactors = a.calendar.FindActiveConfounders(
            windowStart, windowEnd, religiousAffiliation, region)
    }

    // 2. Clinical event confounders from safety_events + medication records
    var clinicalFactors []models.ConfounderFactor
    if a.eventDetector != nil && a.db != nil {
        events := a.fetchPatientClinicalEvents(patientID, windowStart, windowEnd)
        clinicalFactors = a.eventDetector.DetectConfounders(events, windowStart, windowEnd)
    }

    // 3. Compute enhanced score
    return a.scorer.Compute(EnhancedConfounderInput{
        ConcurrentMedCount:   concurrentMedCount,
        AdherenceDrop:        adherenceDrop,
        CalendarFactors:      calendarFactors,
        ClinicalEventFactors: clinicalFactors,
        OutcomeType:          outcomeType,
        DeferOnRamadan:       true,
        DeferOnSteroid:       true,
    })
}

// fetchPatientClinicalEvents queries safety_events and medication_states
// to build the clinical event list for the detector.
func (a *IORConfounderAssessor) fetchPatientClinicalEvents(
    patientID string,
    windowStart, windowEnd time.Time,
) []PatientClinicalEvent {
    var events []PatientClinicalEvent
    lookback := windowStart.AddDate(0, 0, -90) // extend lookback for baselines

    // Safety events (hospitalization, acute illness)
    var safetyEvents []models.SafetyEvent
    a.db.Where("patient_id = ? AND observed_at BETWEEN ? AND ?",
        patientID, lookback, windowEnd).Find(&safetyEvents)
    for _, se := range safetyEvents {
        events = append(events, PatientClinicalEvent{
            Type: se.EventType,
            Date: se.ObservedAt,
        })
    }

    // Medication records (steroid courses, antibiotics)
    var meds []models.MedicationState
    a.db.Where("patient_id = ? AND start_date BETWEEN ? AND ?",
        patientID, lookback, windowEnd).Find(&meds)
    for _, m := range meds {
        events = append(events, PatientClinicalEvent{
            Type:     "MEDICATION_START",
            DrugName: m.DrugName,
            Date:     m.StartDate,
        })
    }

    // Lab results (creatinine for AKI detection)
    var labs []models.LabEntry
    a.db.Where("patient_id = ? AND lab_type = 'CREATININE' AND observed_at BETWEEN ? AND ?",
        patientID, lookback, windowEnd).Find(&labs)
    for _, l := range labs {
        events = append(events, PatientClinicalEvent{
            Type:    "LAB_RESULT",
            LabType: "CREATININE",
            Value:   l.NumericValue,
            Date:    l.ObservedAt,
        })
    }

    return events
}
```

- [ ] **Step 2:** Create KB-23 `confounder_evidence_filter.go`:

```go
package services

import "fmt"

// IOROutcomeResult is the outcome data consumed by KB-23 for evidence display.
type IOROutcomeResult struct {
    DeltaValue      float64 `json:"delta_value"`
    ConfidenceLevel string  `json:"confidence_level"`
    ConfounderScore float64 `json:"confounder_score"`
    Narrative       string  `json:"narrative,omitempty"`
}

// FilterIORByConfidence filters IOR outcomes based on confounder confidence.
func FilterIORByConfidence(outcomes []IOROutcomeResult, minConfidence string) []IOROutcomeResult {
    confidenceRank := map[string]int{"HIGH": 3, "MODERATE": 2, "LOW": 1}
    minRank := confidenceRank[minConfidence]
    if minRank == 0 {
        minRank = 1
    }
    var filtered []IOROutcomeResult
    for _, o := range outcomes {
        rank := confidenceRank[o.ConfidenceLevel]
        if rank >= minRank {
            filtered = append(filtered, o)
        }
    }
    return filtered
}

// AnnotateEvidenceWithConfounderContext adds confounder context to card evidence chains.
func AnnotateEvidenceWithConfounderContext(
    evidenceSummary string,
    totalOutcomes int,
    highConfidenceCount int,
    moderateCount int,
    lowCount int,
) string {
    if totalOutcomes == 0 {
        return evidenceSummary
    }
    highPct := float64(highConfidenceCount) / float64(totalOutcomes) * 100
    if highPct >= 70 {
        return fmt.Sprintf("%s (evidence quality: HIGH — majority of outcomes have minimal confounding)", evidenceSummary)
    }
    if highPct >= 40 {
        return fmt.Sprintf("%s (evidence quality: MODERATE — some outcomes affected by confounders)", evidenceSummary)
    }
    return fmt.Sprintf("%s (evidence quality: LOW — significant confounding in outcome data; interpret with caution)", evidenceSummary)
}
```

- [ ] **Step 3:** Write 4 tests for the evidence filter from the spec (lines 2035-2084):

1. `TestFilterIOR_HighConfidenceOnly` — 4 outcomes (2 HIGH, 1 MODERATE, 1 LOW), filter "HIGH" → 2 results
2. `TestFilterIOR_ModerateAndAbove` — 3 outcomes, filter "MODERATE" → 2 results (HIGH + MODERATE)
3. `TestAnnotateEvidence_HighQuality` — 10 outcomes, 8 HIGH → contains "HIGH"
4. `TestAnnotateEvidence_LowQuality` — 10 outcomes, 2 HIGH, 3 MOD, 5 LOW → contains "LOW" and "caution"

- [ ] **Step 4:** Run tests: KB-20 `go build ./...` + KB-23 `go test ./internal/services/ -run "TestFilterIOR|TestAnnotateEvidence" -v` — all pass.

- [ ] **Step 5:** Commit: `feat: IOR confounder assessor + KB-23 evidence filter (V4-8 Task 5)`

---

### Task 6: Full integration test + commit

- [ ] **Step 1:** Full test sweep across KB-20, KB-23.

- [ ] **Step 2:** Verify all new YAML files parse correctly.

- [ ] **Step 3:** Final commit: `feat: complete V4-8 IOR confounder scoring enhancement`

- [ ] **Step 4:** Push to origin.

---

## Verification Questions

1. Does the India calendar detect Ramadan for Muslim patients but not Hindu patients? (yes / test)
2. Does Ramadan washout (28 days post-event) trigger a confounder factor? (yes / test)
3. Does monsoon only apply to Mumbai/Kolkata/Chennai but not Delhi? (yes / test)
4. Does steroid detection work from medication drug name matching? (yes / test)
5. Does AKI detection use creatinine ≥1.5× baseline per KDIGO? (yes / test)
6. Does the enhanced scorer filter irrelevant confounders by outcome type? (yes / test)
7. Does the scorer defer glucose outcomes during active Ramadan? (yes / test)
8. Does the KB-23 evidence filter correctly partition HIGH/MODERATE/LOW? (yes / test)
9. Are all KB-20 + KB-23 test suites green? (yes / sweep)
10. Do all 3 market-config YAML files parse correctly? (yes / sweep)

---

## Effort Estimate

| Task | Scope | Expected |
|---|---|---|
| Task 1: Models + YAML configs | 4 files, ~400 LOC models + ~200 LOC YAML | 1-2 hours |
| Task 2: Calendar service | 2 files, ~250 LOC + 8 tests | 2-3 hours |
| Task 3: Clinical event detector | 2 files, ~250 LOC + 6 tests | 1-2 hours |
| Task 4: Enhanced scorer | 2 files, ~200 LOC + 8 tests | 1-2 hours |
| Task 5: Assessor + KB-23 filter | 3 files, ~200 LOC + 4 tests | 1-2 hours |
| Task 6: Integration test | 0 new files | 30 min |
| **Total** | **~13 files, ~1500 LOC, ~26 tests** | **~7-10 hours** |
