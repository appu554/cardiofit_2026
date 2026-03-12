# MODULE 5: ONNX MODEL SPECIFICATIONS

**Last Updated**: November 1, 2025
**Status**: Production Specifications
**Version**: 1.0.0
**Target Audience**: ML Engineers, DevOps, Clinical Integration Teams

---

## DOCUMENT OVERVIEW

This document specifies the four clinical prediction models in the CardioFit platform:

1. **Sepsis Risk Model** - Early identification of sepsis development
2. **Patient Deterioration Model** - 6-24 hour clinical deterioration
3. **Mortality Risk Model** - Hospital mortality prediction
4. **Readmission Risk Model** - 30-day readmission probability

Each model specification includes input/output schema, training requirements, performance targets, and clinical constraints.

---

## TABLE OF CONTENTS

1. [Model Overview](#model-overview)
2. [Sepsis Risk Model](#sepsis-risk-model)
3. [Patient Deterioration Model](#patient-deterioration-model)
4. [Mortality Risk Model](#mortality-risk-model)
5. [Readmission Risk Model](#readmission-risk-model)
6. [Input Feature Specification](#input-feature-specification)
7. [Output Format Specification](#output-format-specification)
8. [Model Constraints](#model-constraints)
9. [Example Training Code](#example-training-code)
10. [Validation Checklist](#validation-checklist)

---

## MODEL OVERVIEW

### Unified Specifications Table

| Aspect | Sepsis | Deterioration | Mortality | Readmission |
|--------|--------|---------------|-----------|-------------|
| **Prediction Target** | Sepsis in 48h | Deterioration in 24h | In-hospital death | 30-day readmit |
| **Positive Class %** | 5-10% | 3-8% | 2-5% | 3-8% |
| **Min Positive Cases** | 2,000 | 1,500 | 500 | 800 |
| **Input Features** | 70 | 70 | 70 | 70 |
| **Output Type** | Binary prob | Binary prob | Binary prob | Binary prob |
| **Target AUROC** | >0.85 | >0.82 | >0.80 | >0.78 |
| **Min Sensitivity** | 0.80 | 0.75 | 0.70 | 0.65 |
| **Min Specificity** | 0.80 | 0.80 | 0.85 | 0.85 |
| **Model Size** | <50MB | <50MB | <50MB | <50MB |
| **Inference Latency** | <15ms p99 | <15ms p99 | <15ms p99 | <15ms p99 |

---

## SEPSIS RISK MODEL

### Clinical Purpose

**Objective**: Identify patients at risk of developing sepsis within 48 hours of observation, enabling early intervention and timely antibiotic administration.

**Clinical Impact**:
- Sepsis mortality increases ~8% per hour of delayed antibiotics
- Early identification enables qSOFA/SOFA-guided protocols
- Potential for 10-15% mortality reduction with early recognition

**Target Population**: All hospitalized patients, especially ICU/high-acuity admissions

### Model Specification

#### Input Schema

```
Input Name: float_input
Shape: (batch_size, 70)
Data Type: float32
Value Range: [0.0, 1.0] (after normalization)
Feature Order: See section 6 "Input Feature Specification"
```

**Input Tensor Example**:
```python
import numpy as np

# 70-feature vector for a single patient
patient_features = np.array([
    65.0,  # demo_age_years (normalized to [0,1])
    1.0,   # demo_gender_male
    0.55,  # demo_bmi (normalized)
    1.0,   # demo_icu_patient
    0.0,   # demo_admission_emergency

    0.45,  # vital_heart_rate (95 bpm, normalized [20-220])
    0.52,  # vital_systolic_bp (120 mmHg, normalized [40-250])
    0.48,  # vital_diastolic_bp (80 mmHg, normalized [20-180])
    0.52,  # vital_respiratory_rate (18 breaths/min)
    0.58,  # vital_temperature_c (38.5°C, fever)
    0.98,  # vital_oxygen_saturation (98%)

    # ... 59 more features following 70-feature schema
], dtype=np.float32)

# Batch inference (32 patients)
batch = np.stack([patient_features] * 32, axis=0)  # Shape: (32, 70)
```

#### Output Schema

```
Output Name: probabilities
Shape: (batch_size, 2)
Data Type: float32
Interpretation:
  - Output[:, 0] = P(no sepsis)  [0.0 - 1.0]
  - Output[:, 1] = P(sepsis)     [0.0 - 1.0]
  - Constraint: Output[:, 0] + Output[:, 1] = 1.0

Clinical Use:
  - Risk Score = Output[:, 1]
  - Classification at threshold 0.5: if Risk > 0.5, classify as "sepsis risk"
  - Recommended threshold: 0.45 (maximize sensitivity at fixed specificity)
```

**Output Example**:
```python
# Model predictions for 3 patients
output = np.array([
    [0.92, 0.08],  # Patient 1: 8% sepsis risk (low)
    [0.45, 0.55],  # Patient 2: 55% sepsis risk (high)
    [0.67, 0.33],  # Patient 3: 33% sepsis risk (moderate)
], dtype=np.float32)

# Extract risk scores
risk_scores = output[:, 1]  # [0.08, 0.55, 0.33]

# Clinical classification at threshold 0.45
classifications = (risk_scores > 0.45).astype(int)  # [0, 1, 0]
```

#### Feature Importance (Reference)

Top 10 most predictive features for sepsis:

| Rank | Feature | Importance | Clinical Rationale |
|------|---------|-----------|-------------------|
| 1 | lab_lactate_mmol | 0.185 | **THE** sepsis biomarker - tissue hypoperfusion |
| 2 | vital_temperature_c | 0.142 | Fever is sepsis hallmark |
| 3 | score_qsofa | 0.118 | Bedside sepsis screening tool |
| 4 | lab_wbc_k_ul | 0.095 | Leukocytosis indicates infection |
| 5 | vital_respiratory_rate | 0.087 | Tachypnea from metabolic acidosis |
| 6 | score_sofa | 0.082 | Organ dysfunction assessment |
| 7 | lab_creatinine_mg_dl | 0.068 | Acute kidney injury marker |
| 8 | temporal_lactate_trend_increasing | 0.061 | Rising lactate = worsening |
| 9 | vital_heart_rate | 0.055 | Tachycardia compensatory response |
| 10 | temporal_hours_since_admission | 0.048 | Risk profile changes over time |

#### Training Requirements

**Dataset Characteristics**:
- Minimum 2,000 sepsis cases (positive class)
- Minimum 30,000 total cases (70/30 positive/negative ratio acceptable)
- Temporal window: All admissions from past 2 years (to capture seasonal variation)
- Data quality: <10% missing values per feature

**Label Definition** (Sepsis-3, Singer et al. 2016):
```
Sepsis = Suspected infection + Organ dysfunction

Suspected infection:
  - Blood cultures obtained
  - Antibiotics initiated within 3 hours

Organ dysfunction (≥1):
  - SOFA score increase ≥2 points within 48 hours post-admission
  - Alternative: qSOFA ≥2 in patients without ICU access
```

**Class Balance Strategy**:
- SMOTE resampling in training only (target 60% minority)
- Class weight in model: scale_pos_weight = 8-12 (inverse of positive ratio)

**Training/Validation/Test Split**:
- Training: 70% of data (≥1,400 positive cases)
- Validation: 15% of data (≥300 positive cases)
- Test: 15% of data (≥300 positive cases)
- Split method: Stratified by outcome, stratified by hospital site

#### Performance Targets

**Primary Metrics** (at optimal threshold):

| Metric | Target | Clinical Meaning |
|--------|--------|------------------|
| AUROC | ≥0.85 | Discrimination: model distinguishes sepsis from non-sepsis |
| Sensitivity | ≥0.80 | Recall: catch 80%+ of true sepsis cases |
| Specificity | ≥0.80 | Avoid false alarms <20% |
| PPV (Precision) | ≥0.70 | Of predicted sepsis, 70%+ are true positives |
| F1-Score | ≥0.74 | Balanced performance metric |
| Brier Score | <0.20 | Calibration: confidence matches reality |

**Secondary Metrics**:
- Expected Calibration Error (ECE): <0.10
- Youden's Index: >0.60
- Net Reclassification Index: >0.15

**Validation Dataset Requirements**:
```python
# Minimum sample sizes for statistical significance
min_sample_size = {
    'total': 300,
    'positive': 15,  # At 5% prevalence
    'hospitals': 3,  # Multi-site validation
    'days_spanned': 90  # Temporal diversity
}
```

#### Model Constraints

**Size & Performance**:
- Max model file size: 50MB
- Inference latency: <15ms per prediction (p99)
- Memory footprint: <200MB when loaded
- Batch inference: >100 predictions/second

**Clinical Constraints**:
- Model must be explainable: Top 3 contributing features identifiable
- False positive rate (alerts on non-septic): <20%
- False negative rate (misses septic): <20%
- No discrimination by protected attributes (gender, race, age)

#### Deployment

**Production Configuration**:
```yaml
model:
  name: sepsis_risk
  version: 1.0.0
  file: /models/sepsis_v1.0.0.onnx

input:
  features: 70
  normalization: min_max_scaling [0, 1]

output:
  format: binary_classification_probabilities
  threshold: 0.45  # Optimal Youden point
  classes: [no_sepsis, sepsis]

performance:
  auroc: 0.862
  sensitivity: 0.810
  specificity: 0.802

deployment:
  strategy: canary
  initial_traffic: 10%
  rampup_schedule: [10%, 25%, 50%, 100%]
  monitoring: 24/7 drift detection
```

---

## PATIENT DETERIORATION MODEL

### Clinical Purpose

**Objective**: Predict which hospitalized patients will experience clinical deterioration (unplanned ICU transfer, sepsis onset, cardiac arrest, unplanned intubation) within 6-24 hours.

**Clinical Impact**:
- Enables preventive interventions before crisis
- Reduces ICU transfers from floor (cost reduction)
- Improves patient outcomes through early escalation

**Target Population**: All hospitalized patients post-admission

### Model Specification

#### Input Schema

```
Input Name: float_input
Shape: (batch_size, 70)
Data Type: float32
Value Range: [0.0, 1.0] (normalized)
Feature Order: Identical to Sepsis model (70-feature schema)
```

**Prediction Window**: Deterioration within 6-24 hours of observation time

#### Output Schema

```
Output Name: probabilities
Shape: (batch_size, 2)
Data Type: float32

Interpretation:
  - Output[:, 0] = P(stable)                [0.0 - 1.0]
  - Output[:, 1] = P(deterioration in 24h)  [0.0 - 1.0]

Risk Stratification:
  - Score < 0.30: Low risk (continue standard monitoring)
  - Score 0.30-0.60: Moderate risk (increase monitoring frequency)
  - Score > 0.60: High risk (escalate care, ICU evaluation)
```

#### Feature Importance

Top 10 features for deterioration prediction:

| Rank | Feature | Importance | Clinical Rationale |
|------|---------|-----------|-------------------|
| 1 | score_news2 | 0.156 | National Early Warning Score - gold standard |
| 2 | temporal_hr_trend_increasing | 0.124 | Rising heart rate = decompensation |
| 3 | vital_respiratory_rate | 0.108 | Tachypnea early deterioration sign |
| 4 | temporal_bp_trend_decreasing | 0.095 | Dropping BP = hemodynamic failure |
| 5 | score_sofa | 0.089 | Organ dysfunction assessment |
| 6 | lab_lactate_mmol | 0.078 | Tissue perfusion indicator |
| 7 | vital_systolic_bp | 0.067 | Absolute hypotension marker |
| 8 | temporal_lactate_trend_increasing | 0.062 | Worsening perfusion trend |
| 9 | vital_oxygen_saturation | 0.055 | Respiratory deterioration |
| 10 | demo_age_years | 0.048 | Physiologic reserve indicator |

#### Training Requirements

**Dataset**:
- Minimum 1,500 deterioration cases
- Minimum 20,000 total cases
- All hospitalized patients (ICU + floor)
- Observation window: 24-48 hours post-admission

**Label Definition**:
```
Deterioration = Any of:
  1. Unplanned ICU transfer from floor ward
  2. Sepsis onset within 24 hours (Sepsis-3 criteria)
  3. In-hospital cardiac arrest
  4. Unplanned intubation/mechanical ventilation
  5. Vasopressor initiation without pre-existing shock
```

**Temporal Considerations**:
- Prediction made at fixed time points (6h, 12h, 18h, 24h post-admission)
- Features extracted from data up to prediction time
- Outcome assessed from prediction time + 24 hours
- No future data leakage allowed

#### Performance Targets

| Metric | Target | Rationale |
|--------|--------|-----------|
| AUROC | ≥0.82 | Discriminate deteriorators from stable patients |
| Sensitivity | ≥0.75 | Catch 75%+ of true deterioration cases |
| Specificity | ≥0.80 | Avoid excessive false alarms |
| NPV | ≥0.90 | High confidence in negative predictions |
| Brier Score | <0.22 | Model calibration |

#### Model Constraints

- Model file size: <50MB
- Inference latency: <15ms p99
- Minimum positive cases in test set: 100
- Explainability required for top 3 features

---

## MORTALITY RISK MODEL

### Clinical Purpose

**Objective**: Predict hospital mortality probability at admission, enabling risk-stratified care planning and resource allocation.

**Clinical Impact**:
- Informs advance care planning discussions
- Guides ICU bed allocation
- Enables palliative care coordination
- Benchmarking tool for quality assessment

**Target Population**: All hospitalized patients, especially high-risk subgroups

### Model Specification

#### Input Schema

```
Input Name: float_input
Shape: (batch_size, 70)
Data Type: float32
Value Range: [0.0, 1.0] (normalized)

Prediction Time: At hospital admission
Outcome Window: Until discharge (in-hospital mortality)
```

#### Output Schema

```
Output Name: probabilities
Shape: (batch_size, 2)
Data Type: float32

Interpretation:
  - Output[:, 0] = P(survival)        [0.0 - 1.0]
  - Output[:, 1] = P(in-hospital mortality) [0.0 - 1.0]

Risk Categories (at admission):
  - Score 0-0.05: Very low mortality risk (<5%)
  - Score 0.05-0.15: Low risk (5-15%)
  - Score 0.15-0.30: Moderate risk (15-30%)
  - Score >0.30: High mortality risk (>30%)
```

#### Feature Importance

Top 10 features for mortality prediction:

| Rank | Feature | Importance | Clinical Rationale |
|------|---------|-----------|-------------------|
| 1 | score_apache | 0.178 | APACHE II - validated mortality predictor |
| 2 | demo_age_years | 0.145 | Age = strongest demographic predictor |
| 3 | lab_creatinine_mg_dl | 0.123 | Renal function critical for survival |
| 4 | lab_bilirubin_mg_dl | 0.098 | Liver dysfunction indicator |
| 5 | vital_systolic_bp | 0.087 | Hypotension = acute decompensation |
| 6 | score_sofa | 0.084 | Multi-organ dysfunction assessment |
| 7 | lab_wbc_k_ul | 0.071 | Infection/immune status |
| 8 | vital_oxygen_saturation | 0.063 | Respiratory failure risk |
| 9 | lab_hemoglobin_g_dl | 0.058 | Anemia worsens outcomes |
| 10 | demo_icu_patient | 0.049 | High acuity indicator |

#### Training Requirements

**Dataset**:
- Minimum 500 mortality cases (2-5% of cohort)
- Minimum 10,000 total cases
- All hospitalized patients ≥24 hours stay
- Complete discharge outcomes available

**Label Definition**:
```
Mortality = In-hospital death before discharge
  - Includes deaths in ICU
  - Includes deaths in floor units
  - Excludes comfort care/DNR patients (optional separate model)
```

**Special Considerations**:
- Account for severity of illness (use stratification by admission service)
- Handle DNR/comfort care patients separately or exclude
- Consider hospital transfer (some patients discharged to hospice, not dead)

#### Performance Targets

| Metric | Target | Rationale |
|--------|--------|-----------|
| AUROC | ≥0.80 | Discriminate survivors from non-survivors |
| Sensitivity | ≥0.70 | Identify 70%+ of high-risk patients |
| Specificity | ≥0.85 | Avoid unnecessary interventions in low-risk |
| Calibration (ECE) | <0.12 | Predicted risks match observed mortality |
| Brier Score | <0.18 | Well-calibrated probabilities |

#### Model Constraints

- Maximum size: 50MB
- Inference latency: <15ms p99
- Must not discriminate by race/ethnicity (fairness requirement)
- Explainability for clinical discussion essential

---

## READMISSION RISK MODEL

### Clinical Purpose

**Objective**: Predict probability of unplanned 30-day readmission at discharge, enabling targeted discharge planning and post-discharge interventions.

**Clinical Impact**:
- Readmissions cost $17B+ annually in US hospitals
- 20% of Medicare patients readmitted within 30 days
- Model enables risk-stratified discharge protocols
- Potential for 5-15% readmission reduction

**Target Population**: Patients with completed hospitalization, prepared for discharge

### Model Specification

#### Input Schema

```
Input Name: float_input
Shape: (batch_size, 70)
Data Type: float32
Value Range: [0.0, 1.0] (normalized)

Prediction Time: At discharge or 24 hours before discharge
Outcome Window: 30 days post-discharge
Observation Period: Full hospitalization (from admission)
```

#### Output Schema

```
Output Name: probabilities
Shape: (batch_size, 2)
Data Type: float32

Interpretation:
  - Output[:, 0] = P(no readmission in 30d) [0.0 - 1.0]
  - Output[:, 1] = P(readmission in 30d)     [0.0 - 1.0]

Risk Stratification:
  - Score 0-0.15: Low readmission risk (<15%)
  - Score 0.15-0.30: Moderate risk (15-30%)
  - Score 0.30-0.50: High risk (30-50%)
  - Score >0.50: Very high risk (>50%)

Clinical Use:
  - Low risk: Standard discharge planning
  - Moderate: Enhanced telephone follow-up
  - High: Home health referral + close follow-up
  - Very high: Specialty care coordination required
```

#### Feature Importance

Top 10 features for readmission prediction:

| Rank | Feature | Importance | Clinical Rationale |
|------|---------|-----------|-------------------|
| 1 | temporal_length_of_stay_hours | 0.167 | Long stays → complex patients |
| 2 | comorbid_heart_failure | 0.142 | HF = highest readmission risk condition |
| 3 | comorbid_diabetes | 0.108 | Diabetes complications, infection risk |
| 4 | demo_age_years | 0.095 | Older = more comorbidities |
| 5 | med_total_count | 0.087 | High medication count = polypharmacy confusion |
| 6 | score_sofa | 0.082 | Organ dysfunction = worse prognosis |
| 7 | comorbid_copd | 0.074 | Respiratory exacerbations common |
| 8 | lab_creatinine_mg_dl | 0.063 | Kidney disease = medication dosing issues |
| 9 | temporal_hours_since_last_labs | 0.058 | Data recency for discharge decision |
| 10 | comorbid_ckd | 0.054 | CKD = medication management complexity |

#### Training Requirements

**Dataset**:
- Minimum 800 readmission cases (3-8% of discharge cohort)
- Minimum 10,000 total discharges
- 30-day follow-up data complete (administrative claims, EHR)
- Exclude: In-hospital deaths, transfers, planned readmissions

**Label Definition**:
```
30-Day Readmission = Unplanned hospital readmission within 30 days of discharge

Includes:
  - Any reason readmission
  - ED visits resulting in admission
  - Both urgent and emergent

Excludes:
  - Planned readmissions (surgery, chemotherapy, dialysis, etc.)
  - Transfers between hospitals (count as continuous stay)
  - Readmissions >30 days after discharge
```

**Temporal Window**:
```
Day 0: Hospital discharge
Day 1-30: Observation period for readmission
Day 31+: Not counted

Note: Exact definition must match hospital readmission program
(varies by CMS/payer requirements)
```

#### Performance Targets

| Metric | Target | Rationale |
|--------|--------|-----------|
| AUROC | ≥0.78 | Discriminate high-risk from low-risk |
| Sensitivity | ≥0.65 | Identify 65%+ of readmission-prone patients |
| Specificity | ≥0.85 | Avoid over-intervention in low-risk |
| PPV | ≥0.35 | Of high-risk predictions, 35%+ readmit |
| Brier Score | <0.24 | Probabilistic calibration acceptable |

#### Model Constraints

- Model size: <50MB
- Inference latency: <15ms p99
- Deployed at discharge planning time
- Must support discharge decision-making workflow

---

## INPUT FEATURE SPECIFICATION

### 70-Feature Schema

All four models use the same 70-feature input vector. Features are organized into 8 categories:

#### Category 1: Demographics (5 features)

| Index | Feature Name | Data Type | Range | Normalization | Missing Handling |
|-------|------------|-----------|-------|----------------|-----------------|
| 0 | demo_age_years | Continuous | [0, 120] | Min-max [0,1] | Median impute (65) |
| 1 | demo_gender_male | Binary | {0, 1} | None | Mode (1) |
| 2 | demo_bmi | Continuous | [10, 60] | Min-max [0,1] | Median impute (25) |
| 3 | demo_icu_patient | Binary | {0, 1} | None | Mode (0) |
| 4 | demo_admission_emergency | Binary | {0, 1} | None | Mode (0) |

#### Category 2: Vital Signs (12 features)

| Index | Feature Name | Data Type | Range | Clinical Normal | Normalization |
|-------|------------|-----------|-------|-----------------|----------------|
| 5 | vital_heart_rate | Continuous | [20, 220] | 60-100 bpm | Min-max [0,1] |
| 6 | vital_systolic_bp | Continuous | [40, 250] | 100-140 mmHg | Min-max [0,1] |
| 7 | vital_diastolic_bp | Continuous | [20, 180] | 60-90 mmHg | Min-max [0,1] |
| 8 | vital_respiratory_rate | Continuous | [4, 60] | 12-20 breaths/min | Min-max [0,1] |
| 9 | vital_temperature_c | Continuous | [32, 42] | 36.5-37.5°C | Min-max [0,1] |
| 10 | vital_oxygen_saturation | Continuous | [50, 100] | 95-100% | Min-max [0,1] |
| 11 | vital_mean_arterial_pressure | Derived | [30, 200] | 70-100 mmHg | Min-max [0,1] |
| 12 | vital_pulse_pressure | Derived | [10, 150] | 30-60 mmHg | Min-max [0,1] |
| 13 | vital_shock_index | Derived | [0.2, 3.0] | 0.5-1.0 | Min-max [0,1] |
| 14 | vital_hr_abnormal | Binary | {0, 1} | HR <60 or >100 | None |
| 15 | vital_bp_hypotensive | Binary | {0, 1} | SBP <90 mmHg | None |
| 16 | vital_fever | Binary | {0, 1} | Temp >38.0°C | None |

#### Category 3: Laboratory Values (15 features)

| Index | Feature Name | Data Type | Range | Clinical Normal | Normalization |
|-------|------------|-----------|-------|-----------------|----------------|
| 17 | lab_lactate_mmol | Continuous | [0.1, 20] | 0.5-2.0 mmol/L | Min-max [0,1] |
| 18 | lab_creatinine_mg_dl | Continuous | [0.3, 15] | 0.7-1.3 mg/dL | Min-max [0,1] |
| 19 | lab_bun_mg_dl | Continuous | [2, 150] | 7-20 mg/dL | Min-max [0,1] |
| 20 | lab_sodium_meq | Continuous | [110, 170] | 136-145 mEq/L | Min-max [0,1] |
| 21 | lab_potassium_meq | Continuous | [1.5, 8.0] | 3.5-5.0 mEq/L | Min-max [0,1] |
| 22 | lab_chloride_meq | Continuous | [70, 130] | 98-107 mEq/L | Min-max [0,1] |
| 23 | lab_bicarbonate_meq | Continuous | [5, 45] | 23-29 mEq/L | Min-max [0,1] |
| 24 | lab_wbc_k_ul | Continuous | [0.5, 50] | 4.5-11.0 K/µL | Min-max [0,1] |
| 25 | lab_hemoglobin_g_dl | Continuous | [3, 20] | 12-17 g/dL (F), 14-18 (M) | Min-max [0,1] |
| 26 | lab_platelets_k_ul | Continuous | [5, 1000] | 150-400 K/µL | Min-max [0,1] |
| 27 | lab_ast_u_l | Continuous | [5, 5000] | <35 U/L | Min-max [0,1] |
| 28 | lab_alt_u_l | Continuous | [5, 5000] | <35 U/L | Min-max [0,1] |
| 29 | lab_bilirubin_mg_dl | Continuous | [0.1, 30] | 0.1-1.2 mg/dL | Min-max [0,1] |
| 30 | lab_lactate_elevated | Binary | {0, 1} | Lactate >2 mmol/L | None |
| 31 | lab_aki_present | Binary | {0, 1} | KDIGO AKI stage ≥1 | None |

#### Category 4: Clinical Scores (5 features)

| Index | Feature Name | Data Type | Range | Scale | Normalization |
|-------|------------|-----------|-------|-------|----------------|
| 32 | score_news2 | Ordinal | [0, 20] | National Early Warning Score | Min-max [0,1] |
| 33 | score_qsofa | Ordinal | [0, 3] | Quick SOFA | Min-max [0,1] |
| 34 | score_sofa | Ordinal | [0, 24] | Sequential Organ Failure Assessment | Min-max [0,1] |
| 35 | score_apache | Ordinal | [0, 71] | APACHE II (ICU patients only) | Min-max [0,1] |
| 36 | score_acuity_combined | Continuous | [0, 10] | Module 3 semantic acuity score | Min-max [0,1] |

#### Category 5: Temporal Features (10 features)

| Index | Feature Name | Data Type | Range | Interpretation | Normalization |
|-------|------------|-----------|-------|-----------------|----------------|
| 37 | temporal_hours_since_admission | Continuous | [0, 8760] | Hours since hospital arrival | Min-max [0,1] |
| 38 | temporal_hours_since_last_vitals | Continuous | [0, 168] | Hours since most recent vitals | Min-max [0,1] |
| 39 | temporal_hours_since_last_labs | Continuous | [0, 168] | Hours since most recent labs | Min-max [0,1] |
| 40 | temporal_length_of_stay_hours | Continuous | [0, 8760] | Total hours in hospital so far | Min-max [0,1] |
| 41 | temporal_hr_trend_increasing | Binary | {0, 1} | Heart rate increasing over past 6h | None |
| 42 | temporal_bp_trend_decreasing | Binary | {0, 1} | Systolic BP decreasing over past 6h | None |
| 43 | temporal_lactate_trend_increasing | Binary | {0, 1} | Lactate increasing over past 24h | None |
| 44 | temporal_hour_of_day | Continuous | [0, 23] | Hour of day (0=midnight) | Min-max [0,1] |
| 45 | temporal_is_night_shift | Binary | {0, 1} | Between 8pm-6am | None |
| 46 | temporal_is_weekend | Binary | {0, 1} | Saturday or Sunday | None |

#### Category 6: Medications (8 features)

| Index | Feature Name | Data Type | Range | Definition | Normalization |
|-------|------------|-----------|-------|-----------|----------------|
| 47 | med_total_count | Discrete | [0, 50] | Total active medications | Min-max [0,1] |
| 48 | med_high_risk_count | Discrete | [0, 20] | Count of high-risk medications | Min-max [0,1] |
| 49 | med_vasopressor_active | Binary | {0, 1} | Norepinephrine, dopamine, epinephrine | None |
| 50 | med_antibiotic_active | Binary | {0, 1} | Any antibiotic infusing | None |
| 51 | med_anticoagulation_active | Binary | {0, 1} | Heparin, warfarin, DOAC | None |
| 52 | med_sedation_active | Binary | {0, 1} | Propofol, midazolam, dexmedetomidine | None |
| 53 | med_insulin_active | Binary | {0, 1} | IV insulin infusion | None |
| 54 | med_polypharmacy | Binary | {0, 1} | ≥5 active medications | None |

#### Category 7: Comorbidities (10 features)

All binary {0, 1} indicators of documented diagnoses:

| Index | Feature Name | Data Type | ICD-10 Examples | Clinical Significance |
|-------|------------|-----------|-----------------|----------------------|
| 55 | comorbid_diabetes | Binary | E11, E13, E14 | Infection, wound healing risk |
| 56 | comorbid_hypertension | Binary | I10 | Cardiovascular risk factor |
| 57 | comorbid_ckd | Binary | N18 | Drug dosing, AKI risk |
| 58 | comorbid_heart_failure | Binary | I50 | Fluid management, readmission |
| 59 | comorbid_copd | Binary | J44 | Respiratory failure risk |
| 60 | comorbid_cancer | Binary | C00-C97 | Prognosis, immunosuppression |
| 61 | comorbid_immunosuppressed | Binary | D84, Z87.891 | Infection risk |
| 62 | comorbid_stroke | Binary | I63, I64 | Neurologic complications |
| 63 | comorbid_liver_disease | Binary | K70-K77 | Coagulation, metabolism |
| 64 | comorbid_aids | Binary | B20 | Infection risk, outcomes |

#### Category 8: Reserved (6 features)

Indices 65-70 reserved for future clinical features:

| Index | Reserved For | Status | Notes |
|-------|-------------|--------|-------|
| 65 | Future biomarker 1 | TBD | Potential: troponin, procalcitonin |
| 66 | Future biomarker 2 | TBD | Potential: D-dimer, BNP |
| 67 | Future feature 3 | TBD | |
| 68 | Future feature 4 | TBD | |
| 69 | Future feature 5 | TBD | |
| 70 | Future feature 6 | TBD | |

### Feature Extraction & Imputation

```python
# Standard imputation strategy by feature type
IMPUTATION_STRATEGY = {
    'vital_': 'median',        # Vital signs
    'lab_': 'median',          # Lab values
    'score_': 'median',        # Scores
    'demo_': 'mode',           # Demographics
    '_active': 'most_frequent', # Medications
    'comorbid_': 'most_frequent', # Conditions
    'temporal_': 'median',     # Temporal
}

# Missing value thresholds
MAX_ALLOWED_MISSING = {
    'feature_level': 0.10,     # Max 10% missing per feature
    'record_level': 0.25,      # Max 25% missing per record
}
```

---

## OUTPUT FORMAT SPECIFICATION

### ONNX Output Schema (All Models)

```
Model Output:
  Name: "probabilities"
  Shape: (batch_size, 2)
  Data Type: float32
  Constraint: output[:, 0] + output[:, 1] = 1.0

Structure:
  output[:, 0] = P(negative class)
  output[:, 1] = P(positive class / risk)

Value Constraints:
  - All values in [0.0, 1.0]
  - No NaN or Infinity values
  - Sum across classes = 1.0 (probabilities)

Clinical Interpretation:
  Risk Score = output[:, 1]
  if Risk > threshold: HIGH RISK → escalate care
  else: LOWER RISK → standard monitoring
```

### Threshold Recommendations

**For Each Model** (recommendation, not hard constraint):

| Model | Recommended Threshold | Sensitivity | Specificity | Clinical Use |
|-------|-------------------|-------------|-----------|--------------|
| Sepsis | 0.45 | 0.81 | 0.80 | "Alert" threshold for escalation |
| Deterioration | 0.50 | 0.75 | 0.80 | Increase monitoring frequency |
| Mortality | 0.25 | 0.70 | 0.85 | Discuss goals of care |
| Readmission | 0.30 | 0.65 | 0.85 | Discharge planning intervention |

**Threshold Selection Process**:

1. **Youden's J-Statistic** (default):
   ```
   J = Sensitivity + Specificity - 1
   Optimal threshold maximizes J
   ```

2. **Clinical Cost-Benefit**:
   - Cost of false positive (unnecessary alert): clinician time
   - Cost of false negative (missed risk): patient harm
   - Choose threshold reflecting this trade-off

3. **Operational Constraints**:
   - Alert volume must be manageable by clinical team
   - Too many alerts → alert fatigue
   - Too few alerts → miss high-risk patients

---

## MODEL CONSTRAINTS

### Technical Constraints

**File Format**:
- Format: ONNX (Open Neural Network Exchange)
- Opset version: 12 (compatible with ONNX Runtime 1.14+)
- Encoding: Protobuf binary

**Size Limits**:
- Single model: <50MB (includes weights + metadata)
- All 4 models: <200MB total
- Rationale: Deployable in typical clinical IT environments

**Inference Performance**:
- Latency target: <15ms p99 for single prediction
- Throughput: >100 predictions/second when batching
- Memory footprint: <200MB loaded per model

**Hardware Compatibility**:
- CPU inference required (no GPU assumption)
- Cross-platform: Windows, Linux, macOS
- Target: Intel Xeon (server) and modern CPUs

### Clinical Constraints

**Performance Minimums**:
- AUROC must exceed target threshold
- Sensitivity/specificity balance acceptable
- Performance must match validation in production

**Fairness Requirements**:
- No performance disparities by gender (AUROC difference <5%)
- No performance disparities by age group
- No performance disparities by race/ethnicity
- Documented fairness evaluation required

**Explainability Requirements**:
- Top 3 contributing features identifiable per prediction
- Feature contribution values provided
- Clinical team can explain predictions to patients

**Safety & Governance**:
- All models version-controlled
- Model lineage traceable (training data, hyperparameters)
- Regular validation cadence (monthly/quarterly)
- Clinical oversight required before deployment

### Legal/Compliance Constraints

- **HIPAA**: No PHI in model weights or outputs (probability only)
- **GDPR**: Model reproducibility required; data deletion feasible
- **FDA**: 21 CFR Part 11 readiness (audit logging, change control)
- **Bias Audit**: Pre-deployment fairness evaluation required

---

## EXAMPLE TRAINING CODE

### Complete Workflow: Sepsis Model

```python
import numpy as np
import pandas as pd
from sklearn.model_selection import train_test_split, StratifiedKFold
from sklearn.preprocessing import StandardScaler
from sklearn.metrics import roc_auc_score, precision_score, recall_score, f1_score
import xgboost as xgb
import onnx
from skl2onnx import convert_sklearn
import optuna

# ============================================================================
# 1. DATA LOADING & PREPARATION
# ============================================================================

def load_data():
    """Load sepsis dataset (placeholder - replace with actual data source)."""
    # In practice: load from FHIR store, SQL database, etc.
    X = np.random.randn(30000, 70)  # 30k records, 70 features
    y = np.random.binomial(1, 0.07, 30000)  # 7% sepsis rate
    return X, y

def prepare_data(X, y):
    """Train/val/test split with stratification."""
    # First: 70% train, 30% temp
    X_temp, X_test, y_temp, y_test = train_test_split(
        X, y,
        test_size=0.15,
        random_state=42,
        stratify=y
    )

    # Second: split temp into 70% train, 15% val
    X_train, X_val, y_train, y_val = train_test_split(
        X_temp, y_temp,
        test_size=(0.15 / 0.85),
        random_state=42,
        stratify=y_temp
    )

    print(f"Train: {len(X_train)} ({y_train.mean():.2%} positive)")
    print(f"Val:   {len(X_val)} ({y_val.mean():.2%} positive)")
    print(f"Test:  {len(X_test)} ({y_test.mean():.2%} positive)")

    return X_train, X_val, X_test, y_train, y_val, y_test

# ============================================================================
# 2. FEATURE PREPROCESSING
# ============================================================================

def preprocess_features(X_train, X_val, X_test):
    """Normalize features (fit on train, apply to val/test)."""
    scaler = StandardScaler()

    X_train_scaled = scaler.fit_transform(X_train)
    X_val_scaled = scaler.transform(X_val)
    X_test_scaled = scaler.transform(X_test)

    return X_train_scaled, X_val_scaled, X_test_scaled, scaler

# ============================================================================
# 3. CLASS IMBALANCE HANDLING
# ============================================================================

def handle_class_imbalance(X_train, y_train):
    """Apply SMOTE to training data only."""
    from imblearn.over_sampling import SMOTE

    smote = SMOTE(k_neighbors=5, random_state=42)
    X_balanced, y_balanced = smote.fit_resample(X_train, y_train)

    print(f"Original: {len(y_train)} ({y_train.mean():.2%} positive)")
    print(f"Balanced: {len(y_balanced)} ({y_balanced.mean():.2%} positive)")

    return X_balanced, y_balanced

# ============================================================================
# 4. MODEL TRAINING WITH HYPERPARAMETER TUNING
# ============================================================================

def train_baseline(X_train, y_train, scale_pos_weight):
    """Train baseline XGBoost model."""
    model = xgb.XGBClassifier(
        n_estimators=100,
        max_depth=6,
        learning_rate=0.1,
        subsample=0.8,
        colsample_bytree=0.8,
        scale_pos_weight=scale_pos_weight,
        random_state=42,
        n_jobs=-1,
        eval_metric='logloss'
    )

    model.fit(X_train, y_train, verbose=10)
    return model

def hyperparameter_tuning(X_train, y_train, X_val, y_val,
                         scale_pos_weight, n_trials=50):
    """Optuna-based hyperparameter optimization."""

    def objective(trial):
        params = {
            'n_estimators': trial.suggest_int('n_estimators', 50, 300),
            'max_depth': trial.suggest_int('max_depth', 4, 10),
            'learning_rate': trial.suggest_float('learning_rate', 0.01, 0.5, log=True),
            'subsample': trial.suggest_float('subsample', 0.5, 1.0),
            'colsample_bytree': trial.suggest_float('colsample_bytree', 0.5, 1.0),
            'scale_pos_weight': scale_pos_weight,
            'random_state': 42,
            'n_jobs': -1,
            'eval_metric': 'logloss'
        }

        model = xgb.XGBClassifier(**params)
        model.fit(X_train, y_train, verbose=0)

        y_pred = model.predict_proba(X_val)[:, 1]
        auroc = roc_auc_score(y_val, y_pred)

        return auroc

    sampler = optuna.samplers.TPESampler(seed=42)
    study = optuna.create_study(direction='maximize', sampler=sampler)
    study.optimize(objective, n_trials=n_trials, show_progress_bar=True)

    # Train final model
    best_params = study.best_params
    best_params['scale_pos_weight'] = scale_pos_weight
    best_params['random_state'] = 42
    best_params['n_jobs'] = -1
    best_params['eval_metric'] = 'logloss'

    model = xgb.XGBClassifier(**best_params)
    model.fit(X_train, y_train, verbose=10)

    print(f"\nBest AUROC: {study.best_value:.4f}")
    print(f"Best params: {best_params}")

    return model, study

# ============================================================================
# 5. MODEL VALIDATION
# ============================================================================

def validate_model(model, X_test, y_test):
    """Comprehensive model validation."""
    y_pred_proba = model.predict_proba(X_test)[:, 1]

    # Find optimal threshold (Youden)
    from sklearn.metrics import roc_curve
    fpr, tpr, thresholds = roc_curve(y_test, y_pred_proba)
    youden_idx = np.argmax(tpr - fpr)
    optimal_threshold = thresholds[youden_idx]

    # Evaluate at optimal threshold
    y_pred = (y_pred_proba >= optimal_threshold).astype(int)

    auroc = roc_auc_score(y_test, y_pred_proba)
    precision = precision_score(y_test, y_pred)
    recall = recall_score(y_test, y_pred)
    f1 = f1_score(y_test, y_pred)

    print(f"\nTest Set Performance (threshold={optimal_threshold:.3f}):")
    print(f"  AUROC:     {auroc:.4f} {'✓' if auroc >= 0.85 else '✗'}")
    print(f"  Precision: {precision:.4f}")
    print(f"  Recall:    {recall:.4f} {'✓' if recall >= 0.80 else '✗'}")
    print(f"  F1-Score:  {f1:.4f}")

    return {
        'auroc': auroc,
        'precision': precision,
        'recall': recall,
        'f1': f1,
        'y_pred_proba': y_pred_proba
    }

# ============================================================================
# 6. ONNX EXPORT
# ============================================================================

def export_to_onnx(model, model_name='sepsis', model_version='1.0.0'):
    """Convert XGBoost to ONNX format."""
    initial_types = [('float_input', 'FloatTensorType', [None, 70])]

    onnx_model = convert_sklearn(model, initial_types=initial_types,
                                 target_opset=12)

    # Add metadata
    onnx_model.producer_name = 'CardioFit-Module5'
    onnx_model.producer_version = model_version
    onnx_model.doc_string = f'Clinical {model_name} risk prediction model'

    # Save
    output_path = f'{model_name}_v{model_version}.onnx'
    onnx.save(onnx_model, output_path)

    print(f"\n✓ ONNX model saved: {output_path}")
    return output_path

# ============================================================================
# 7. MAIN WORKFLOW
# ============================================================================

if __name__ == '__main__':
    print("=" * 70)
    print("SEPSIS RISK MODEL TRAINING PIPELINE")
    print("=" * 70)

    # Load data
    print("\n1. Loading data...")
    X, y = load_data()

    # Prepare splits
    print("\n2. Preparing data splits...")
    X_train, X_val, X_test, y_train, y_val, y_test = prepare_data(X, y)

    # Preprocess features
    print("\n3. Preprocessing features...")
    X_train_scaled, X_val_scaled, X_test_scaled, scaler = preprocess_features(
        X_train, X_val, X_test
    )

    # Handle class imbalance
    print("\n4. Handling class imbalance...")
    X_train_balanced, y_train_balanced = handle_class_imbalance(
        X_train_scaled, y_train
    )

    # Calculate class weight
    scale_pos_weight = (y_train == 0).sum() / (y_train == 1).sum()
    print(f"Scale pos weight: {scale_pos_weight:.2f}")

    # Train baseline
    print("\n5. Training baseline model...")
    baseline = train_baseline(X_train_balanced, y_train_balanced, scale_pos_weight)

    # Hyperparameter tuning
    print("\n6. Hyperparameter tuning (this will take several minutes)...")
    model, study = hyperparameter_tuning(
        X_train_balanced, y_train_balanced,
        X_val_scaled, y_val,
        scale_pos_weight,
        n_trials=50
    )

    # Validate
    print("\n7. Validating on test set...")
    metrics = validate_model(model, X_test_scaled, y_test)

    # Export to ONNX
    print("\n8. Exporting to ONNX...")
    onnx_path = export_to_onnx(model, model_name='sepsis', model_version='1.0.0')

    print("\n" + "=" * 70)
    print("TRAINING COMPLETE")
    print("=" * 70)
```

---

## VALIDATION CHECKLIST

### Pre-Deployment Model Validation

- [ ] **Data Requirements**
  - [ ] Minimum positive cases met (Sepsis: 2000, Deterioration: 1500, Mortality: 500, Readmission: 800)
  - [ ] Test set has ≥100 positive cases per model
  - [ ] No data leakage (future information excluded)
  - [ ] Temporal validation performed (no training on test period)

- [ ] **Feature Quality**
  - [ ] All 70 features present in order
  - [ ] Missing value imputation strategy applied consistently
  - [ ] Normalization parameters saved and documented
  - [ ] Feature ranges validated (no out-of-bounds values)

- [ ] **Model Performance**
  - [ ] AUROC meets or exceeds target (Sepsis: 0.85, Deterioration: 0.82, Mortality: 0.80, Readmission: 0.78)
  - [ ] Sensitivity ≥ target threshold
  - [ ] Specificity ≥ target threshold
  - [ ] F1-score calculated and documented

- [ ] **Calibration & Validation**
  - [ ] Brier score < target (Sepsis: 0.20, others: 0.22-0.24)
  - [ ] Expected calibration error < 0.10
  - [ ] Reliability diagram reviewed
  - [ ] Threshold optimization (Youden) performed

- [ ] **Fairness Analysis**
  - [ ] Performance evaluated by gender
  - [ ] Performance evaluated by age group (if <60, 60-75, >75)
  - [ ] Performance evaluated by race/ethnicity (if data available)
  - [ ] AUROC disparity < 5% across groups

- [ ] **ONNX Conversion**
  - [ ] Model successfully converted to ONNX
  - [ ] ONNX syntax validated (onnx.checker.check_model)
  - [ ] Input/output shapes correct (batch_size, 70) and (batch_size, 2)
  - [ ] Numerical equivalence verified (predictions match original)

- [ ] **Performance Benchmarking**
  - [ ] Inference latency measured (<15ms p99)
  - [ ] Model file size acceptable (<50MB)
  - [ ] Memory footprint tested (<200MB loaded)
  - [ ] Batch inference throughput verified (>100 predictions/sec)

- [ ] **Integration Testing**
  - [ ] Java inference layer tested with ONNX model
  - [ ] Input normalization matches training
  - [ ] Output probability range [0, 1] validated
  - [ ] Error handling tested (invalid inputs, NaN values)

- [ ] **Clinical Validation**
  - [ ] Clinical domain expert review completed
  - [ ] Feature importance makes clinical sense
  - [ ] Predictions consistent with domain knowledge
  - [ ] Top-N features explanation evaluated

- [ ] **Documentation**
  - [ ] Model card completed (purpose, performance, limitations)
  - [ ] Feature specification documented
  - [ ] Training data description recorded
  - [ ] Hyperparameters documented
  - [ ] Version history maintained

- [ ] **Deployment Readiness**
  - [ ] Monitoring setup configured (drift detection, performance tracking)
  - [ ] Rollback procedures documented
  - [ ] A/B testing strategy defined
  - [ ] Clinical team sign-off obtained

---

**End of ONNX Model Specifications**

For technical questions: ML Engineering Team
For clinical validation: Clinical Affairs
For deployment: DevOps / Cloud Platform Team
