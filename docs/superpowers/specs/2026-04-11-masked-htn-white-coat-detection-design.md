# Masked Hypertension & White-Coat Detection — Design Spec

**Date:** 2026-04-11
**Branch:** `feature/v4-clinical-gaps`
**Scope:** KB-26 classifier + KB-23 cards + market configs + migration (no Flink/Java)

---

## 1. Problem Statement

A platform that collects both home and clinic BP readings and doesn't compare them is leaving its most distinctive clinical signal on the table. Masked hypertension (MH) affects 10-30% of hypertensives (Banegas 2018), rises to 30-40% in diabetics (Franklin 2020), and 30-50% in CKD patients (Agarwal 2022). MH carries 2.09x CV event risk vs normotension (Pierdomenico 2017) and is invisible to clinic-only measurement.

The cross-domain innovation: this isn't a single-domain BP feature. It integrates diabetes status, CKD stage, engagement phenotype, morning surge, medication timing, and measurement behavior to produce clinically actionable insight invisible to any single-domain CDS.

## 2. Clinical Phenotypes

Six phenotypes from the 2x2 clinic-home threshold matrix, split by treatment status:

| Clinic >= threshold | Home >= threshold | Untreated | Treated |
|---|---|---|---|
| Yes | Yes | SUSTAINED_HTN | SUSTAINED_HTN |
| Yes | No | WHITE_COAT_HTN | WHITE_COAT_UNCONTROLLED |
| No | Yes | MASKED_HTN | MASKED_UNCONTROLLED |
| No | No | SUSTAINED_NORMOTENSION | SUSTAINED_NORMOTENSION |

Plus `INSUFFICIENT_DATA` when minimum reading requirements are not met.

### Thresholds (ESH 2023, ISH 2020)

| Context | SBP | DBP |
|---------|-----|-----|
| Clinic (general) | >= 140 | >= 90 |
| Clinic (diabetic) | >= 130 | >= 80 |
| Home (mean) | >= 135 | >= 85 |

### Minimum Data Requirements

- **Clinic:** >= 2 readings within last 90 days (separate visits preferred)
- **Home:** >= 12 readings over >= 4 distinct days within last 14 days (ESH: discard day 1)

## 3. Architecture

### Data Flow (Go-only)

```
KB-20 (patient profile) provides clinic + home readings
  -> KB-26 ClassifyBPContext() -> BPContextClassification
    -> KB-23 EvaluateMaskedHTNCards() -> []MaskedHTNCard
    -> KB-23 FourPillarEvaluator -> masked HTN medication pillar gap
```

### Components

1. **KB-26 (Metabolic Digital Twin):** Core classification engine
   - `internal/models/bp_context.go` — BPContextPhenotype enum + BPContextClassification struct
   - `internal/services/bp_context_classifier.go` — ClassifyBPContext() + helpers
   - `internal/services/bp_context_classifier_test.go` — 14 test cases
   - `migrations/006_bp_context.sql` — bp_context_history table

2. **KB-23 (Decision Cards):** Card generation + four-pillar integration
   - `internal/models/bp_context.go` — Local mirror of BPContextClassification (KB-23 owns its models)
   - `internal/services/masked_htn_cards.go` — EvaluateMaskedHTNCards() producing 8 card types
   - `internal/services/masked_htn_cards_test.go` — 7 test cases
   - `internal/services/four_pillar_evaluator.go` — MODIFY: add BPContext to FourPillarInput

3. **Market Configs:**
   - `market-configs/shared/bp_context_thresholds.yaml` — Base thresholds + amplification rules
   - `market-configs/india/bp_context_overrides.yaml` — Wider WCE (20mmHg), sodium correlation, device validation
   - `market-configs/australia/bp_context_overrides.yaml` — ABPM MBS 11607, indigenous overrides

## 4. Cross-Domain Amplification

These patterns are invisible to single-domain CDS:

| Signal Combination | Clinical Impact | Urgency Override |
|---|---|---|
| MH + Diabetes | 3.2x target organ damage (Leitao 2015) | IMMEDIATE |
| MH + CKD | 15% faster eGFR decline per 10mmHg (Agarwal 2017) | URGENT |
| MH + Morning Surge >20mmHg | Multiplicative stroke risk (Kario 2019) | IMMEDIATE |
| MH + MEASUREMENT_AVOIDANT | Possible selection bias in home readings | Flag LOW confidence |
| Treated + Morning>Evening by >15mmHg | Medication wearing off overnight | Timing hypothesis card |

## 5. Decision Cards (8 Types)

| Card Type | Trigger | Urgency |
|---|---|---|
| MASKED_HYPERTENSION | MH detected | URGENT (IMMEDIATE if DM) |
| MASKED_UNCONTROLLED | MUCH in treated patient | URGENT |
| WHITE_COAT_HYPERTENSION | WCH detected | ROUTINE |
| WHITE_COAT_UNCONTROLLED | WCUH in treated patient | ROUTINE |
| MASKED_HTN_MORNING_SURGE_COMPOUND | MH + surge >20 | IMMEDIATE |
| SUSTAINED_HTN_MORNING_SURGE | SH + surge >20 | URGENT |
| SELECTION_BIAS_WARNING | Bias-prone engagement phenotype | ROUTINE |
| MEDICATION_TIMING | Morning-evening differential >15mmHg | ROUTINE |

## 6. Four-Pillar Integration

Add `BPContext *models.BPContextClassification` to `FourPillarInput`. In medication pillar evaluation:

- MH/MUCH -> PillarGap (PillarUrgentGap if DM amplification or morning surge compound)
- WCH/WCUH -> PillarGap (avoid overtreatment rationale)

## 7. Market-Specific Behavior

### India (ISH 2020, RSSDI 2023)
- WCE clinically_significant threshold: **20 mmHg** (vs 15 standard) due to higher clinic anxiety
- Sodium correlation: correlate evening home BP with same-day sodium intake (Module 10 data)
- Device validation warnings for wrist/unvalidated cuffs
- Recommended devices: Omron HEM-7120, HEM-7130, Dr. Morepen BP-02

### Australia (Heart Foundation 2022, RACGP 2024)
- ABPM recommendation with MBS item code 11607 when MH suspected
- Community health worker readings treated as home-equivalent
- Indigenous screening interval: 3 months (vs 6 general)
- Heart Foundation threshold alignment with ISH 2020

## 8. Database Schema

```sql
CREATE TABLE bp_context_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id VARCHAR(100) NOT NULL,
    snapshot_date DATE NOT NULL,
    phenotype VARCHAR(30) NOT NULL,
    clinic_sbp_mean DECIMAL(5,1),
    home_sbp_mean DECIMAL(5,1),
    gap_sbp DECIMAL(5,1),
    confidence VARCHAR(10),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(patient_id, snapshot_date)
);

CREATE INDEX idx_bpc_patient ON bp_context_history(patient_id, snapshot_date DESC);
```

## 9. File Inventory

| Phase | Action | File |
|-------|--------|------|
| M1 | Create | `kb-26-metabolic-digital-twin/internal/models/bp_context.go` |
| M1 | Create | `market-configs/shared/bp_context_thresholds.yaml` |
| M1 | Create | `market-configs/india/bp_context_overrides.yaml` |
| M1 | Create | `market-configs/australia/bp_context_overrides.yaml` |
| M2 | Create | `kb-26-metabolic-digital-twin/internal/services/bp_context_classifier.go` |
| M2 | Create | `kb-26-metabolic-digital-twin/internal/services/bp_context_classifier_test.go` |
| M2 | Create | `kb-26-metabolic-digital-twin/migrations/006_bp_context.sql` |
| M3 | Create | `kb-23-decision-cards/internal/models/bp_context.go` |
| M3 | Create | `kb-23-decision-cards/internal/services/masked_htn_cards.go` |
| M3 | Create | `kb-23-decision-cards/internal/services/masked_htn_cards_test.go` |
| M3 | Modify | `kb-23-decision-cards/internal/services/four_pillar_evaluator.go` |

**Total: 11 files (10 create, 1 modify)**

## 10. Adjustments from Original Requirements

| Item | Original Spec | Adjusted |
|------|--------------|----------|
| Migration number | 007 | **006** (next after 005_cgm_tables.sql) |
| KB-23 model imports | `import "kb-metabolic-digital-twin/..."` | Local `bp_context.go` in KB-23 models (follows codebase pattern) |
| `ClinicReadings_` field | Underscore suffix | `ClinicReadingCount` (cleaner naming) |
| `HomeReadings_` field | Underscore suffix | `HomeReadingCount` |
| Flink Module 7 operator | Listed in file inventory | **Excluded** (user scoped to Go-only) |
| Module 7 Java model | `BPContextClassification.java` | **Excluded** |
| Module 7 test | `Module7_MaskedHTNDetectorTest.java` | **Excluded** |

## 11. Clinical Evidence Base

- Banegas et al. 2018 (European Heart Journal) — MH prevalence 10-30%, MUCH 30-50%
- Pierdomenico et al. 2017 (Hypertension) — MH 2.09x CV event risk
- Leitao et al. 2015 (Diabetologia) — DM + MH 3.2x target organ damage
- Agarwal et al. 2017 (CJASN) — MH strongest predictor of CKD progression
- Agarwal et al. 2022 (Kidney International) — MH prevalence 30-50% in CKD
- Franklin et al. 2020 (Hypertension) — MH prevalence 30-40% in diabetes
- Kario et al. 2019 (JACC) — MH + morning surge compound stroke risk
- Ohkubo et al. 2004 (JAMA) — home BP prognostic superiority over clinic
- Mancia et al. 2006 (PAMELA) — WCH progression rate ~3%/year
- ESH 2023, ISH 2020, AHA/ACC 2017, Heart Foundation 2022

## 12. Test Coverage

### KB-26 Classifier Tests (14 cases)
1. Sustained HTN classification
2. Masked HTN classification
3. White-coat HTN classification
4. Sustained normotension classification
5. Masked uncontrolled (treated)
6. White-coat uncontrolled (treated)
7. Diabetic amplification (MH + DM)
8. CKD amplification (MH + CKD)
9. Morning surge compound (MH + surge >20)
10. Selection bias — MEASUREMENT_AVOIDANT
11. Insufficient home data
12. Insufficient clinic data
13. Medication timing hypothesis (morning-evening differential)
14. Rajesh Kumar integration (sustained HTN + DM + CKD + surge)

### KB-23 Card Tests (7 cases)
1. Masked HTN + diabetic -> IMMEDIATE urgency card
2. White-coat HTN -> ROUTINE + overtreatment warning
3. MUCH -> URGENT + false therapeutic confidence
4. Compound risk MH + morning surge -> IMMEDIATE
5. Selection bias -> uncertainty flag card
6. Medication timing -> timing hypothesis card
7. Normotensive -> no urgent cards
