# V4 Remaining Tasks — Design Specification

**Date**: 2026-04-05
**Scope**: 11 remaining tasks from the V4 North Star Implementation Plan
**Execution model**: Strict sequential — C0 → C2 → C3 → C4 → C5 → C6
**Master plan reference**: `docs/superpowers/plans/2026-03-26-v4-north-star-implementation.md`

---

## Context

The V4 North Star plan defined 15 tasks across 6 phases (C0-C6). As of 2026-04-05, the Flink processing layer is essentially complete — all 7 new modules (M7-M13) are implemented with TDD tests and wired into `FlinkJobOrchestrator`. The remaining work spans Go KB service extensions, Flink V3 dual-domain logic, PostgreSQL persistence layers, a Python ML batch pipeline, and market configuration infrastructure.

### Already Complete (skip in plan)
- Task 1: KB-22 Stratum Hierarchy Resolver — `stratum_hierarchy.go` exists with 58 tests, wired into `CreateSession()` and `ResumeSession()`
- Task 2: V4 Kafka Topics — wired into `KafkaTopics.java` via module-specific commits
- Task 3: V4 Signal Types in Ingestion — 7 new `ObservationType` enums + 13 V4 fields in `CanonicalObservation` with 8 tests
- Task 4: Module8 CID (17 rules) — fully implemented with HALT/PAUSE/SOFT_FLAG evaluators
- Task 5: Module7 BP Variability Engine — ARV, surge, dip classification implemented
- Task 9: Module9 Engagement Monitor — 5 sub-operators implemented
- Task 10: Module11 InterventionWindowMonitor (renamed Module12) — 5 sub-operators
- Task 10b: Module10 MealResponseCorrelator — 4 sub-operators with session windows
- Task 10c: Module10b MealPatternAggregator — weekly aggregation with OLS salt sensitivity

### Remaining (11 tasks, this spec)
| # | Task | Phase | Tech Stack |
|---|------|-------|------------|
| 3b | V3 Schema Extensions for V4 Fields (Flink side) | C0 | Java (Flink) |
| 3c | V3 Contract Wiring Fixes | C0 | Go (KB-20, KB-22) |
| 6 | KB-20 V4 State Fields | C2 | Go (KB-20) |
| 6b | V3 Flink Job Dual-Domain Extensions | C2 | Java (Flink) |
| 7 | MHRI Score in KB-26 (audit + gaps) | C3 | Go (KB-26) |
| 8 | IOR System | C3 | Go + PostgreSQL |
| 11 | KB-23 Dual-Domain Decision Cards | C4 | Go (KB-23) |
| 12 | Physician Feedback Capture | C4 | Go + PostgreSQL |
| 13 | Phenotype Clustering Pipeline | C5 | Python (UMAP + HDBSCAN) |
| 14 | Feedback Analysis Pipelines | C5 | Go (integrated into KB-23) |
| 15 | Market Shim Configuration | C6 | Go + YAML |

---

## Phase C0: Foundation Wiring

### Task 3b: Flink-Side V4 Field Propagation

**Problem**: Ingestion service emits V4 fields in `CanonicalObservation` JSON, but Flink Module 1b (`IngestionCanonicalizer`) may not deserialize or propagate all 13 V4 fields through the pipeline.

**Approach**:
1. Audit `CanonicalEvent.java` and payload models for V4 field presence
2. Add missing V4 fields to `CanonicalEvent` or appropriate payload models:
   - `dataTier` (String) — already present in `EnrichedPatientContext`
   - `linkedMealId` (String) — needed for Module 10 meal correlation
   - `sodiumEstimatedMg` (Double) — needed for Module 10b salt sensitivity
   - `preparationMethod` (String) — food metadata
   - `foodNameLocal` (String) — regional food name
   - `symptomAwareness` (Boolean) — CID-03 masking detection
   - `bpDeviceType` (String) — device classification
   - `clinicalGrade` (Boolean) — validated device flag
   - `measurementMethod` (String) — BP measurement method
   - `linkedSeatedReadingId` (String) — orthostatic delta linkage
   - `wakingTime` (String) — surge window timing
   - `sleepTime` (String) — nocturnal window timing
   - `sourceProtocol` (String) — meal protocol identifier
3. Update Module 1b canonicalizer to extract and forward these fields
4. Write tests verifying V4 field round-trip through Module 1b

**Files to modify**:
- `flink-processing/src/main/java/com/cardiofit/flink/models/CanonicalEvent.java`
- `flink-processing/src/main/java/com/cardiofit/flink/operators/Module1b_IngestionCanonicalizer.java`
- `flink-processing/src/test/java/com/cardiofit/flink/operators/Module1bV4FieldTest.java` (new)

**Acceptance criteria**:
- All 13 V4 fields deserialize from ingestion JSON
- Fields propagate through Module 1b output to downstream consumers
- Module 10/10b can read `linkedMealId` and `sodiumEstimatedMg` from upstream events
- Tests cover null/missing field handling (V3 events without V4 fields)

---

### Task 3c: V3 Contract Wiring Fixes

**Problem**: Four inter-service contract issues identified in the north star plan. Some may already be fixed (KB-22 and KB-24 show modifications in git status). Need audit + completion.

**Contract items**:

#### P12: KB-20 Event Publication
KB-20 must publish `LAB_RESULT` and `MEDICATION_CHANGE` events to Kafka so downstream consumers (Module 8 CID, Module 13) receive them.

- **Check**: Does `kb-20-patient-profile/internal/services/kafka_outbox_relay.go` publish these event types?
- **If missing**: Add event publication in lab_service.go and medication_service.go after successful writes
- **Test**: Verify events appear on `clinical-events` topic

#### P14: KB-22 Medication Safety Provider (KB-9 → KB-5)
KB-22's medication safety checks must call KB-5 (Drug Interactions, port 8089) not the deprecated KB-9.

- **Check**: `grep -r "kb-9\|KB-9\|8089" kb-22-hpi-engine/` — should find KB-5 references only
- **If stale**: Rewrite `medication_safety_provider.go` to target KB-5 gRPC endpoint
- **Test**: Verify drug interaction queries route to KB-5

#### P15-a: KB-22 Outcome Publisher Endpoint
KB-22's outcome publisher must POST to `/execute` not `/events` on KB-19.

- **Check**: `grep -r "/events" kb-22-hpi-engine/internal/services/outcome_publisher.go`
- **If stale**: Change endpoint to `/execute`
- **Test**: Verify outcome publication succeeds

#### P15-b: KB-22 Minimum Inclusion Guard
`CompleteSession()` must enforce a minimum number of questions answered before allowing completion, preventing empty sessions from generating decision cards.

- **Check**: Does `session_service.go` `CompleteSession()` have a guard like `if session.QuestionsAnswered < minInclusion`?
- **If missing**: Add guard with configurable minimum (default: 3 questions)
- **Test**: Verify early completion is rejected with appropriate error

**Files to modify**:
- `kb-20-patient-profile/internal/services/kafka_outbox_relay.go` (P12)
- `kb-20-patient-profile/internal/services/lab_service.go` (P12)
- `kb-20-patient-profile/internal/services/medication_service.go` (P12)
- `kb-22-hpi-engine/internal/services/medication_safety_provider.go` (P14)
- `kb-22-hpi-engine/internal/services/outcome_publisher.go` (P15-a)
- `kb-22-hpi-engine/internal/services/session_service.go` (P15-b)

**Acceptance criteria**:
- `grep -r "kb-9\|KB-9" kb-22-hpi-engine/` returns no matches
- `grep -r "/events" kb-22-hpi-engine/internal/services/outcome_publisher.go` returns no matches
- KB-20 publishes LAB_RESULT and MEDICATION_CHANGE events
- CompleteSession rejects sessions with < minimum questions answered
- All existing KB-22 tests still pass

---

## Phase C2: KB-20 Extensions + Dual-Domain Flink

### Task 6: KB-20 V4 State Fields

**Problem**: KB-20 patient profile needs cached V4 state fields that Module 13 (Clinical State Synchroniser) writes to via `KB20AsyncSinkFunction`. These fields enable KB-23 dual-domain cards and KB-26 MHRI computation.

**New fields on patient profile model**:

```go
// V4 State Cache (written by Module 13)
CKMStage              int        // AHA CKM 0-4
CKMStageUpdatedAt     time.Time
MHRIScore             float64    // 0-100 composite from KB-26
MHRICategory          string     // OPTIMAL/MILD/MODERATE/HIGH
MHRIUpdatedAt         time.Time
EngagementScore       float64    // 0-100 from Module 9
EngagementLevel       string     // HIGH/MODERATE/LOW/DISENGAGED
EngagementUpdatedAt   time.Time
ARVValue              float64    // Average Real Variability from Module 7
SurgeClassification   string     // NORMAL/ELEVATED/HIGH
DipClassification     string     // DIPPER/NON_DIPPER/REVERSE_DIPPER/EXTREME_DIPPER
BPVariabilityUpdatedAt time.Time
ActiveCIDAlerts       JSONB      // Current CID alert IDs from Module 8
CIDUpdatedAt          time.Time
PhenotypeClusterID    *int       // From batch pipeline (Task 13)
PhenotypeUpdatedAt    *time.Time
```

**CKM Stage computation function**:
AHA Cardiovascular-Kidney-Metabolic staging (0-4):
- Stage 0: No risk factors
- Stage 1: Excess adiposity (BMI ≥ 25 or waist > threshold)
- Stage 2: Metabolic risk (pre-diabetes, hypertension, moderate CKD, high triglycerides)
- Stage 3: Subclinical CVD or high predicted risk (PREVENT >10%)
- Stage 4: Clinical CVD or advanced CKD (eGFR <30) or HF

Inputs: BMI, waist circumference, HbA1c, FBG, SBP, DBP, eGFR, ACR, triglycerides, PREVENT score, CVD history, HF diagnosis.

**API endpoints**:
- `GET /patients/{id}/v4-state` — returns all V4 cached fields
- `PATCH /patients/{id}/v4-state` — Module 13 sink writes (internal only)

**Files to create/modify**:
- `kb-20-patient-profile/internal/models/v4_state.go` (new — V4 field definitions)
- `kb-20-patient-profile/internal/services/ckm_stage.go` (new — CKM computation)
- `kb-20-patient-profile/internal/services/ckm_stage_test.go` (new)
- `kb-20-patient-profile/internal/api/v4_state_handlers.go` (new)
- `kb-20-patient-profile/internal/models/patient.go` (modify — add V4 fields)
- Database migration for new columns

**Acceptance criteria**:
- CKM stage correctly classifies across all 5 stages with edge cases
- V4 state PATCH endpoint accepts Module 13 sink writes
- GET endpoint returns all cached V4 fields with timestamps
- Null/zero handling for patients without V4 data (graceful defaults)
- Migration runs cleanly on existing data

---

### Task 6b: V3 Flink Job Dual-Domain Extensions

**Problem**: V3 Flink jobs (Module 1b, 3, 4) have V4 field placeholders but lack the actual dual-domain business logic.

**Module 1b extensions** (IngestionCanonicalizer):
- Set `clinicalDomain` on output events based on observation type:
  - FBG, PPBG, HbA1c, CGM → `GLYCAEMIC`
  - BP (seated/standing/morning/evening) → `HEMODYNAMIC`
  - Creatinine, ACR, eGFR → `RENAL`
  - Weight, waist, lipids → `METABOLIC`
- Propagate `data_tier` from ingestion payload to enriched context

**Module 3 extensions** (ComprehensiveCDS):
- Add BP trajectory slope computation (SBP slope over 14-day window) alongside existing glucose trajectory
- Set `trajectoryClass` for hemodynamic domain events
- Emit dual-domain flag when both glycaemic AND hemodynamic trajectories are deteriorating simultaneously

**Module 4 extensions** (PatternDetection):
- Add cross-domain CEP patterns:
  - `GLYCAEMIC_HEMODYNAMIC_CONCORDANT_DECLINE`: both FBG rising + SBP rising in same 7-day window
  - `RENAL_METABOLIC_RISK`: eGFR declining + weight increasing
  - `MEDICATION_CROSS_EFFECT`: thiazide started (BP domain) + glucose rising (glycaemic domain)
- Route dual-domain alerts to `v4-cross-domain-alerts` topic

**Files to modify**:
- `Module1b_IngestionCanonicalizer.java` — domain classification logic
- `Module3_ComprehensiveCDS.java` — BP trajectory + dual-domain flag
- `Module4_PatternDetection.java` — cross-domain CEP patterns
- New test files for each extension

**Acceptance criteria**:
- Every canonical event carries a `clinicalDomain` tag after Module 1b
- Module 3 computes BP trajectory slope with same algorithm as glucose trajectory
- Module 4 fires cross-domain patterns when concordant deterioration occurs
- Existing V3 test suites still pass (no regression)
- New tests cover domain classification for all observation types

---

## Phase C3: Intelligence Core

### Task 7: MHRI Score Audit + Gap Closure in KB-26

**Problem**: KB-26 already has `mri_scorer.go` with 4-domain composite scoring. The north star plan specified additional components that may or may not exist. Need audit and gap closure.

**Audit checklist**:
1. 5 piecewise-linear normalization functions (glucose, body_comp, cardio, renal, behavioral) — check `mri_normalizer.go`
2. 14-day trajectory engine with trend detection — check for sliding window logic
3. MHRI API endpoint — check `mri_handlers.go`
4. CKM Stage computation (Gap G1) — may overlap with Task 6, need to determine ownership

**Known gap: CKM Stage ownership**
The north star plan places CKM in both Task 6 (KB-20) and Task 7 (KB-26 G1). Resolution: KB-20 owns the CKM stage computation (Task 6) since it has all input signals. KB-26 reads CKM stage from KB-20 for MHRI domain weighting. No duplication.

**Likely gaps to implement**:
- 14-day MHRI trajectory with slope computation and trend classification (IMPROVING/STABLE/DECLINING)
- Per-domain trajectory breakdown (which domain is driving change)
- MHRI change event publication to Kafka when score crosses category thresholds

**Files to audit/modify**:
- `kb-26-metabolic-digital-twin/internal/services/mri_scorer.go` (audit)
- `kb-26-metabolic-digital-twin/internal/services/mri_normalizer.go` (audit)
- `kb-26-metabolic-digital-twin/internal/services/mri_trajectory.go` (new if missing)
- `kb-26-metabolic-digital-twin/internal/api/mri_handlers.go` (audit)
- Tests for any new/modified logic

**Acceptance criteria**:
- MHRI score computes correctly for all 4 domains
- 14-day trajectory with trend classification exists and is tested
- Category threshold crossing triggers Kafka event
- KB-26 reads CKM stage from KB-20 (no local CKM computation)
- All normalization functions handle edge cases (missing signals, zero values)

---

### Task 8: IOR System (Intervention-Outcome Records)

**Problem**: No system exists to track what clinical interventions were applied and what outcomes they produced. The IOR system enables evidence-based card enrichment (Task 11) and physician feedback analysis (Task 14).

**Architecture**: New Go service module within KB-20 (same database, separate schema) or standalone micro-package. Given KB-20 already tracks medications, protocols, and labs, integrating IOR into KB-20 minimizes cross-service calls.

**Data models**:

```go
// InterventionRecord: tracks a clinical intervention event
type InterventionRecord struct {
    ID                uuid.UUID
    PatientID         uuid.UUID
    InterventionType  string     // MEDICATION_START, DOSE_CHANGE, LIFESTYLE_RX, REFERRAL
    DrugClass         *string    // e.g., "SGLT2i", "ACEi" (nil for non-medication)
    DrugName          *string
    DoseChangeMg      *float64   // positive = increase, negative = decrease
    PrescribedBy      *uuid.UUID // physician ID
    CardID            *uuid.UUID // decision card that triggered this
    ProtocolID        *string    // linked protocol (GLYC-1, HTN-1, etc.)
    ProtocolPhase     *string
    StartDate         time.Time
    EndDate           *time.Time // nil = ongoing
    Status            string     // ACTIVE, COMPLETED, DISCONTINUED, SUPERSEDED
    // Gap G3: Confounder capture
    ConcurrentMedChanges  JSONB  // other med changes in same window
    AdherenceAtStart      *float64
    AdherenceAtEnd        *float64
    LifestyleChanges      JSONB  // concurrent lifestyle modifications
    SeasonAtStart         *string
    StressLevelAtStart    *string
    CreatedAt         time.Time
    UpdatedAt         time.Time
}

// OutcomeRecord: measured outcome linked to an intervention
type OutcomeRecord struct {
    ID                uuid.UUID
    InterventionID    uuid.UUID  // FK to InterventionRecord
    PatientID         uuid.UUID
    OutcomeType       string     // DELTA_HBA1C, DELTA_SBP, DELTA_EGFR, DELTA_WEIGHT, etc.
    BaselineValue     float64    // value at intervention start
    OutcomeValue      float64    // value at measurement time
    DeltaValue        float64    // outcome - baseline
    DeltaPercent      float64    // (delta / baseline) * 100
    MeasurementDate   time.Time
    WindowWeeks       int        // weeks since intervention start
    ConfidenceLevel   string     // HIGH (controlled), MODERATE (some confounders), LOW (many confounders)
    ConfounderScore   float64    // 0-1, higher = more confounded
    CreatedAt         time.Time
}
```

**IOR store service**:
- `CreateIntervention(ctx, record)` — insert with validation
- `CreateOutcome(ctx, record)` — insert with intervention FK check
- `GetInterventionsByPatient(ctx, patientID, filters)` — list with status/type/date filters
- `QuerySimilarPatientOutcomes(ctx, query)` — find outcomes for similar interventions across patients, filtered by stratum, CKM stage, phenotype cluster. Returns aggregate statistics (median delta, IQR, n).

**IOR generator batch job**:
- Runs periodically (daily or on-demand)
- Scans active interventions where `StartDate + windowWeeks` has elapsed
- Fetches current lab/vital values from KB-20
- Computes delta from baseline
- Creates OutcomeRecord with confounder scoring
- Window checkpoints: 4 weeks, 12 weeks, 26 weeks, 52 weeks

**API endpoints**:
- `POST /ior/interventions` — create intervention record
- `GET /ior/interventions?patient_id=X` — list by patient
- `GET /ior/outcomes?intervention_id=X` — outcomes for intervention
- `GET /ior/similar-outcomes` — similar-patient aggregate query
- `POST /ior/generate` — trigger batch outcome generation

**Files to create**:
- `kb-20-patient-profile/internal/models/ior.go` — InterventionRecord + OutcomeRecord
- `kb-20-patient-profile/internal/services/ior_store.go` — CRUD + similar-patient queries
- `kb-20-patient-profile/internal/services/ior_generator.go` — batch outcome computation
- `kb-20-patient-profile/internal/services/ior_confounder.go` — confounder scoring logic
- `kb-20-patient-profile/internal/api/ior_handlers.go` — REST endpoints
- `kb-20-patient-profile/internal/services/ior_store_test.go`
- `kb-20-patient-profile/internal/services/ior_generator_test.go`
- Database migration for IOR tables

**Acceptance criteria**:
- Intervention CRUD with proper validation and status transitions
- Outcome records link to interventions with delta computation
- Similar-patient query filters by stratum, CKM stage, phenotype cluster
- Confounder scoring accounts for concurrent meds, adherence, lifestyle, season
- Batch generator creates outcomes at 4/12/26/52 week checkpoints
- All endpoints tested with edge cases (no baseline value, discontinued intervention, etc.)

---

## Phase C4: Decision Layer

### Task 11: KB-23 Dual-Domain Decision Card Generator

**Problem**: KB-23 currently generates single-domain cards (glycaemic OR hemodynamic). V4 requires integrated dual-domain cards that consider both domains simultaneously, detect conflicts, and incorporate IOR evidence.

**New components**:

#### Dual-Domain State Classifier
Classifies patient into one of 9 combined states:
```
           Glycaemic: CONTROLLED | AT_TARGET | UNCONTROLLED
Hemodynamic:
  CONTROLLED          GC-HC        GC-HT        GC-HU
  AT_TARGET           GT-HC        GT-HT        GT-HU
  UNCONTROLLED        GU-HC        GU-HT        GU-HU
```
Inputs: latest HbA1c, FBG trajectory, SBP average, BP variability (ARV), CKM stage.

#### Four-Pillar Evaluator (DD#5)
For each dual-domain state, evaluates 4 pillars:
1. **Medication pillar**: Are current medications optimal for both domains? Detect under-treatment.
2. **Lifestyle pillar**: Are lifestyle interventions adequate? (exercise dose, sodium intake, weight trajectory)
3. **Monitoring pillar**: Are monitoring frequencies appropriate? (lab schedules, BP measurement cadence)
4. **Referral pillar**: Does the patient need specialist referral? (nephrology, cardiology, endocrinology)

Each pillar outputs: ADEQUATE / GAP_DETECTED / URGENT_GAP with specific recommendations.

#### Conflict Detector
Detects when glycaemic and hemodynamic recommendations conflict:
- Thiazide diuretic (helps BP) may raise glucose
- Beta-blocker (helps BP/HR) may mask hypoglycaemia awareness
- SGLT2i (helps glucose + BP + renal) — no conflict, flag as synergistic
- Corticosteroid (needed for comorbidity) raises both glucose and BP

Output: list of `ConflictRecord{drug_class, glycaemic_effect, hemodynamic_effect, severity, resolution_suggestion}`.

#### Urgency Calculator
Combines domain urgencies into a single priority:
- If either domain is URGENT → card priority = URGENT
- If both domains have GAP_DETECTED → escalate to URGENT (concordant deterioration)
- Otherwise → highest individual domain urgency

#### IOR Insight Provider
Enriches card recommendations with evidence from Task 8:
- "Similar patients (n=47) on SGLT2i showed median HbA1c reduction of 0.8% and SBP reduction of 5 mmHg at 12 weeks"
- Filters by stratum, CKM stage, phenotype cluster for relevance

#### New card types
- `INTEGRATED_DUAL_DOMAIN`: combined glycaemic+hemodynamic assessment
- `FOUR_PILLAR_GAP`: specific pillar gap with actionable recommendation

#### Card content template engine
Generates `ClinicianSummary`, `PatientSummaryEn`, `PatientSummaryHi` from structured data using template strings with variable interpolation.

**Files to create/modify**:
- `kb-23-decision-cards/internal/services/dual_domain_classifier.go` (new)
- `kb-23-decision-cards/internal/services/four_pillar_evaluator.go` (new)
- `kb-23-decision-cards/internal/services/conflict_detector.go` (new)
- `kb-23-decision-cards/internal/services/urgency_calculator.go` (new)
- `kb-23-decision-cards/internal/services/ior_insight_provider.go` (new)
- `kb-23-decision-cards/internal/services/card_template_engine.go` (new)
- `kb-23-decision-cards/internal/models/dual_domain.go` (new — classifier + conflict models)
- `kb-23-decision-cards/internal/models/four_pillar.go` (new — pillar evaluation models)
- `kb-23-decision-cards/internal/models/enums.go` (modify — add new card types)
- `kb-23-decision-cards/internal/services/card_builder.go` (modify — integrate dual-domain path)
- Tests for each new component

**Acceptance criteria**:
- Dual-domain classifier correctly maps to all 9 states
- Four-pillar evaluator identifies gaps with specific recommendations
- Conflict detector catches all known drug-domain conflicts (thiazide, beta-blocker, corticosteroid)
- Synergistic medications (SGLT2i, GLP-1 RA) flagged positively
- Urgency calculator escalates concordant deterioration
- IOR insights query returns relevant similar-patient evidence
- Card templates produce readable clinician and patient summaries in English and Hindi
- Existing single-domain card generation continues to work (backward compatible)

---

### Task 12: Physician Feedback Capture

**Problem**: No mechanism to capture physician responses to decision cards (accept/modify/reject). This data feeds the learning pipelines in Task 14.

**Feedback model** (21 fields per north star):

```go
type PhysicianFeedback struct {
    ID                  uuid.UUID
    CardID              uuid.UUID   // FK to DecisionCard
    PatientID           uuid.UUID
    PhysicianID         uuid.UUID
    SessionID           *uuid.UUID  // KB-22 session if applicable
    ActionTaken         string      // ACCEPT, MODIFY, REJECT, DEFER
    // Modification details (populated when MODIFY)
    ModifiedDrugClass   *string
    ModifiedDose        *float64
    ModifiedFrequency   *string
    ModificationReason  *string
    // Rejection details (populated when REJECT)
    RejectionReason     *string     // CLINICALLY_INAPPROPRIATE, PATIENT_PREFERENCE, COST, CONTRAINDICATED, OTHER
    RejectionFreeText   *string
    // Deferral details
    DeferralReason      *string
    DeferUntilDate      *time.Time
    // Context
    TimeToDecisionSec   *int        // seconds from card display to action
    ViewedSections      JSONB       // which card sections physician viewed
    IORInsightViewed    bool        // did physician view similar-patient data
    ConfidenceInAction  *int        // 1-5 self-reported confidence
    // Metadata
    Platform            string      // WEB_DASHBOARD, MOBILE_APP
    CreatedAt           time.Time
    UpdatedAt           time.Time
}
```

**Store service**: PostgreSQL CRUD with query methods:
- `CreateFeedback(ctx, feedback)` — insert with card FK validation
- `GetFeedbackByCard(ctx, cardID)` — all feedback for a card
- `GetFeedbackByPhysician(ctx, physicianID, filters)` — physician's history
- `GetRejectionPatterns(ctx, filters)` — aggregate rejection reasons by card type

**API endpoint**:
- `POST /feedback` — capture physician feedback
- `GET /feedback?card_id=X` — retrieve feedback for card
- `GET /feedback/stats?physician_id=X` — acceptance rate, common modifications

**Files to create**:
- `kb-23-decision-cards/internal/models/feedback.go` (new)
- `kb-23-decision-cards/internal/services/feedback_store.go` (new)
- `kb-23-decision-cards/internal/api/feedback_handlers.go` (new)
- `kb-23-decision-cards/internal/services/feedback_store_test.go` (new)
- Database migration for feedback table

**Acceptance criteria**:
- All 4 action types (ACCEPT/MODIFY/REJECT/DEFER) handled with appropriate required fields
- Card FK validation (can't submit feedback for non-existent card)
- Rejection patterns queryable by card type, time range, physician
- Stats endpoint returns acceptance rate and common modifications
- Tests cover all action types and validation edge cases

---

## Phase C5: Learning Layer

### Task 13: Phenotype Clustering Pipeline (Python)

**Problem**: Need to group patients into clinically meaningful phenotype clusters for personalized therapy mapping and IOR relevance filtering.

**Architecture**: Standalone Python batch pipeline. Runs weekly or on-demand. Reads patient features from KB-20, clusters using UMAP + HDBSCAN, writes cluster assignments back to KB-20.

**Project structure**:
```
backend/shared-infrastructure/phenotype-clustering/
├── requirements.txt
├── src/
│   ├── feature_extractor.py    — 21-feature extraction from KB-20 API
│   ├── clustering_pipeline.py  — UMAP + HDBSCAN with validation
│   ├── centroid_exporter.py    — export cluster centroids for runtime use
│   ├── therapy_mapper.py       — map clusters to therapy pathways (G6)
│   └── kb20_updater.py         — write cluster assignments to KB-20
├── tests/
│   ├── test_feature_extractor.py
│   ├── test_clustering.py
│   └── test_therapy_mapper.py
└── configs/
    └── clustering_config.yaml
```

**21 features** (from north star DD#9):
1-5: HbA1c, FBG mean, FBG variability, PPBG mean, glucose trajectory slope
6-10: SBP mean, SBP variability (ARV), DBP mean, BP trajectory slope, dipping pattern (encoded)
11-15: eGFR, eGFR slope, ACR, BMI, waist circumference
16-18: Total cholesterol, LDL, triglycerides
19-21: Engagement score, medication adherence rate, age

**Pipeline steps**:
1. Feature extraction: query KB-20 for all active patients, extract 21 features, handle missing values (imputation or exclusion)
2. Preprocessing: standardize features (z-score), handle outliers (winsorize at 3 SD)
3. UMAP reduction: 21D → 5D (preserving local structure for clinical similarity)
4. HDBSCAN clustering: min_cluster_size=20, min_samples=5 (tunable via config)
5. Validation: silhouette score, DBCV, clinical coherence check (do clusters have clinically distinct profiles?)
6. Centroid export: compute cluster centroids in original feature space for runtime assignment
7. Therapy mapping: assign each cluster a recommended therapy pathway based on centroid profile
8. KB-20 update: PATCH each patient's `phenotype_cluster_id`

**Therapy mapper (Gap G6)**:
Maps cluster centroids to therapy pathways:
- High glucose + high BP + low adherence → Intensive dual-domain + adherence support
- Controlled glucose + uncontrolled BP + high variability → BP-focused with variability monitoring
- Young + metabolic syndrome + high engagement → Aggressive lifestyle-first
- Elderly + CKD + polypharmacy → Conservative with deprescribing review
Uses decision tree logic, not ML, for interpretability.

**Acceptance criteria**:
- Feature extractor handles missing values gracefully (impute or exclude with documentation)
- Clustering produces 4-12 clinically distinct clusters (configurable)
- Silhouette score > 0.3 on test data
- Therapy mapper assigns pathway to every cluster with rationale
- KB-20 updater writes cluster IDs without affecting other patient fields
- Full pipeline runs end-to-end with test fixture data

---

### Task 14: Feedback Analysis Pipelines

**Problem**: Physician feedback data (from Task 12) needs automated analysis to improve decision card quality and safety rules.

**Four pipelines**:

#### Pipeline 1: Rejection Pattern Detection
- Input: feedback records where `action_taken = REJECT`
- Analysis: group by `rejection_reason × card_type × stratum`, compute rejection rates
- Output: `RejectionPattern{card_type, stratum, rejection_rate, top_reasons, sample_size, trend}`
- Trigger: flag patterns where rejection rate > 30% for review

#### Pipeline 3: Safety Rule Enrichment
- Input: feedback records + corresponding card safety checks
- Analysis: identify cases where physicians rejected cards that passed safety checks (false negatives) or accepted cards flagged by safety (false positives)
- Output: `SafetyRuleProposal{rule_id, proposed_change, evidence_count, confidence}`
- Integration: feeds KB-24 safety constraint engine with proposed rule modifications

#### Pipeline 4: Acceptance Rate Metrics
- Input: all feedback records
- Analysis: compute per-physician, per-card-type, per-time-period acceptance rates
- Output: dashboards/metrics — acceptance rate, median time-to-decision, modification patterns
- Trigger: flag physicians with < 50% acceptance rate for training review

#### Governance Lifecycle (Gap G4)
Rule change proposals follow a governance lifecycle:
```
PROPOSED → UNDER_REVIEW → APPROVED → DEPLOYED
                       → REJECTED (with reason)
```

**RuleChangeProposal model**:
```go
type RuleChangeProposal struct {
    ID                uuid.UUID
    ProposalType      string     // SAFETY_RULE_MODIFY, SAFETY_RULE_ADD, CARD_TEMPLATE_MODIFY
    SourcePipeline    string     // REJECTION_PATTERN, SAFETY_ENRICHMENT, ACCEPTANCE_METRIC
    CurrentRule       JSONB      // snapshot of current rule/template
    ProposedChange    JSONB      // proposed modification
    EvidenceCount     int        // number of feedback records supporting this
    ConfidenceScore   float64    // statistical confidence
    Status            string     // PROPOSED, UNDER_REVIEW, APPROVED, REJECTED, DEPLOYED
    ReviewerID        *uuid.UUID
    ReviewNotes       *string
    ApprovedAt        *time.Time
    DeployedAt        *time.Time
    CreatedAt         time.Time
}
```

**Files to create**:
- `kb-23-decision-cards/internal/services/feedback_analyzer.go` — Pipeline 1 + 4
- `kb-23-decision-cards/internal/services/safety_enrichment.go` — Pipeline 3
- `kb-23-decision-cards/internal/models/rule_change_proposal.go` — governance model
- `kb-23-decision-cards/internal/services/governance_service.go` — lifecycle management
- `kb-23-decision-cards/internal/api/governance_handlers.go` — governance API
- Tests for each pipeline and governance transitions

**Acceptance criteria**:
- Pipeline 1 detects rejection patterns with > 30% rate and generates alerts
- Pipeline 3 identifies safety rule gaps from physician feedback
- Pipeline 4 computes per-physician acceptance metrics
- Governance lifecycle enforces valid state transitions
- Proposals cannot be deployed without review approval
- All pipelines handle empty/insufficient data gracefully

---

## Phase C6: Market Shim

### Task 15: Market Configuration Infrastructure

**Problem**: Clinical parameters, communication channels, pharmacy databases, food databases, and compliance rules differ between India and Australia. A market-specific configuration layer allows the same codebase to serve both markets.

**Configuration files** (per market):

```
backend/shared-infrastructure/market-configs/
├── india/
│   ├── clinical_params.yaml     — thresholds, targets, units
│   ├── channels.yaml            — WhatsApp, SMS, push notification config
│   ├── pharma_shim.yaml         — Indian drug names, formulations, pricing
│   ├── food_db_config.yaml      — IFCT 2017 database connection
│   └── compliance.yaml          — ABDM, DISHA Act requirements
├── australia/
│   ├── clinical_params.yaml     — thresholds (may differ), Medicare rules
│   ├── channels.yaml            — SMS, email, My Health Record integration
│   ├── pharma_shim.yaml         — PBS formulary, Australian drug names
│   ├── food_db_config.yaml      — AUSNUT 2011-13 database connection
│   ├── compliance.yaml          — Privacy Act, My Health Records Act
│   └── indigenous_overrides.yaml — NACCHO guidelines, remote health adjustments
└── shared/
    └── base_clinical_params.yaml — shared defaults

```

**Shared config loader Go package**:

```go
package marketconfig

// MarketConfig holds all market-specific configuration
type MarketConfig struct {
    Market           string
    ClinicalParams   ClinicalParams
    Channels         ChannelConfig
    PharmaShim       PharmaConfig
    FoodDB           FoodDBConfig
    Compliance       ComplianceConfig
}

// Load reads and merges market-specific YAML over shared base
func Load(market string, configDir string) (*MarketConfig, error)
```

Key clinical parameter differences:
- India: HbA1c target 7.0% (RSSDI), SBP target <130 (ISH), uses IFCT sodium database
- Australia: HbA1c target 7.0% (RACGP) with 8.0% for elderly/frail, SBP target <140 (Heart Foundation), indigenous populations have adjusted targets per NACCHO

**Wire into services**:
- KB-20: uses `clinical_params.yaml` for threshold checks and `compliance.yaml` for data handling
- KB-23: uses `clinical_params.yaml` for card recommendations and `channels.yaml` for delivery routing
- KB-26: uses `clinical_params.yaml` for MHRI normalization ranges and `food_db_config.yaml` for sodium estimation

**Files to create**:
- `backend/shared-infrastructure/market-configs/` directory with all YAML files
- `backend/shared-infrastructure/market-configs/loader/` Go package
- `backend/shared-infrastructure/market-configs/loader/config.go` — types + Load()
- `backend/shared-infrastructure/market-configs/loader/config_test.go`
- Modify KB-20, KB-23, KB-26 `main.go` to inject market config at startup

**Acceptance criteria**:
- Config loader reads market YAML and merges over shared base
- India and Australia configs are complete with all required fields
- Indigenous overrides correctly merge over Australia base
- KB-20, KB-23, KB-26 use market config for clinical thresholds (not hardcoded)
- Invalid market name returns clear error
- Tests verify config loading, merging, and override precedence

---

## Dependency Graph

```
C0: Task 3b ──→ C2: Task 6b (Flink needs V4 fields propagated)
C0: Task 3c ──→ C2: Task 6 (KB-20 contracts must be clean)
C2: Task 6  ──→ C3: Task 7 (KB-26 reads CKM from KB-20)
C2: Task 6  ──→ C3: Task 8 (IOR queries KB-20 for baselines)
C3: Task 7  ──→ C4: Task 11 (cards use MHRI scores)
C3: Task 8  ──→ C4: Task 11 (cards use IOR evidence)
C4: Task 11 ──→ C4: Task 12 (feedback captures responses to cards)
C4: Task 12 ──→ C5: Task 14 (feedback pipelines analyze feedback data)
C2: Task 6  ──→ C5: Task 13 (clustering reads KB-20 features)
C5: Task 13 ──→ C3: Task 8 (IOR uses cluster ID for similarity — soft dependency)
C6: Task 15 is independent (can be done any time, placed last per sequential requirement)
```

---

## Risk Factors

1. **Task 11 (KB-23 Dual-Domain Cards) is the largest task** — 7 new components. May need decomposition into sub-tasks during implementation planning.
2. **Task 8 (IOR) similar-patient query performance** — aggregating outcomes across patients with filtering could be slow. May need materialized views or pre-computed aggregates.
3. **Task 13 (Phenotype Clustering) reproducibility** — UMAP + HDBSCAN are stochastic. Need fixed random seeds and validation metrics.
4. **Task 6b (Dual-Domain Flink)** — modifying production V3 operators carries regression risk. Extensive test coverage needed.
5. **Task 15 (Market Shim)** — clinical parameter accuracy is critical. India and Australia medical guidelines must be sourced from authoritative references.
