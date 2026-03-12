# Dataset Strategy - Clarified Approach

**Date**: November 5, 2025
**Status**: ✅ **STRATEGY CLARIFIED - TWO SEPARATE MODEL TRACKS**

---

## Executive Summary

### Critical Clarification
The datasets provided serve **different clinical use cases** and should drive **separate model development tracks**:

1. **ICU Risk Models** (Module 5 - Flink Real-Time): Needs Indian ICU patient-level data
2. **Maternal Care Models** (New Module): Can use Mendeley + NidaanKosha datasets

**Key Insight**: We should NOT try to force maternal health data into ICU models - they serve completely different clinical contexts.

---

## Two-Track Strategy

### Track 1: ICU Deterioration Models (Module 5)
**Current Status**: ✅ **READY FOR DEPLOYMENT WITH MIMIC-IV**

#### Models
- Sepsis Risk (AUROC 98.55%)
- Clinical Deterioration (AUROC 78.96%)
- Mortality Risk (AUROC 95.70%)

#### Current Dataset
- **MIMIC-IV** (U.S. tertiary ICU): 23,000 patients
- **Feature Scaling**: Implemented and validated
- **Integration Tests**: 5/5 passing

#### Deployment Strategy
**Option B: Shadow Mode + Local Calibration** (RECOMMENDED)

1. **Week 1**: Deploy MIMIC-IV models in shadow mode
   - No clinical alerts
   - Log all predictions + outcomes
   - Monitor prediction distributions

2. **Weeks 2-8**: Data collection phase
   - Target: 500-1000 patient predictions
   - Track outcomes: sepsis confirmed, ICU transfer, mortality
   - Generate calibration dataset

3. **Weeks 9-10**: Calibration analysis
   - Reliability curves (predicted vs actual)
   - Platt scaling / isotonic regression
   - Threshold tuning for Indian population

4. **Week 11+**: Advisory mode pilot
   - Show predictions to clinicians
   - Require confirmation for actions
   - Track override rates

#### Future Enhancement: Indian ICU Dataset
**Target Datasets** (when available):

1. **ICMR COVID Clinical Registry**
   - 40,000+ ICU patients
   - Full vitals + labs + outcomes
   - Access: Research MoU required

2. **ISCCM Indian Critical Care Data Registry (INCIID)**
   - Indian Society of Critical Care Medicine
   - AIIMS + PGIMER network hospitals
   - Vitals, labs, ventilator settings, SOFA scores

3. **JIPMER ICU Patient Cohort**
   - High-resolution ICU time-series
   - Research approval needed

**Timeline**: 3-6 months (requires institutional partnerships)

---

### Track 2: Maternal Care Models (New Module)
**Current Status**: 📋 **DESIGN PHASE - NEW MODEL DEVELOPMENT**

#### Datasets Available

**1. Mendeley Maternal Health Service Coverage (ngtpyv4zws)**
- **Type**: Population-level maternal health statistics
- **Source**: HMIS (Health Management Information System), National Health Mission
- **Coverage**: State/Union Territory level, 2017-2020
- **Domains**:
  - Antenatal care (ANC) coverage
  - Intranatal care (INC) coverage
  - Postnatal care (PNC) coverage
- **Use Cases**:
  - ✅ Maternal care service coverage prediction
  - ✅ ANC adherence risk scoring
  - ✅ Care pathway compliance modeling
  - ✅ Community health worker (CHW) follow-up prioritization

**2. NidaanKosha-100k (HuggingFace)**
- **Type**: Lab test records (100k+ tests)
- **Features**: Age, gender, test results with LOINC codes
- **Use Cases**:
  - ✅ Anemia risk during pregnancy (Hgb trends)
  - ✅ Gestational diabetes screening (glucose patterns)
  - ✅ Preeclampsia markers (blood pressure, protein)

**3. NFHS-5 (National Family Health Survey)**
- **Type**: Population health survey
- **Coverage**: District-level maternal health indicators
- **Use Cases**:
  - ✅ Risk factor prevalence (anemia, undernutrition)
  - ✅ Geographic risk stratification

#### Model Objectives

**Primary Models**:
1. **Pregnancy Progression Risk** (anemia, GDM, preeclampsia)
2. **ANC Adherence Prediction** (likelihood of completing ANC visits)
3. **Maternal Complication Risk** (hemorrhage, sepsis, eclampsia)
4. **CHW Follow-up Prioritization** (triage high-risk pregnancies)

**Target Features**:
- Maternal demographics (age, parity, BMI)
- ANC visit timeline and completeness
- Lab markers (Hgb, glucose, blood pressure trends)
- Geographic/socioeconomic factors
- Previous pregnancy outcomes

**Model Architecture**:
- **Type**: Time-series risk progression models
- **Framework**: LSTM or Transformer for temporal patterns
- **Output**: Risk scores at each ANC visit (ANC1, ANC2, ANC3, ANC4)

#### Development Timeline
- **Phase 1** (2 weeks): Dataset integration and feature engineering
- **Phase 2** (3 weeks): Model development and training
- **Phase 3** (2 weeks): Validation with clinical experts
- **Phase 4** (2 weeks): Integration with CHW workflow system

---

## Critical Differences Between Tracks

| Dimension | Track 1: ICU Models | Track 2: Maternal Models |
|-----------|-------------------|------------------------|
| **Care Setting** | Tertiary ICU, Emergency | Primary care, Community health centers |
| **Time Scale** | Hours to days (acute) | Weeks to months (pregnancy) |
| **Data Type** | High-frequency vitals + labs | Periodic ANC visits + lab panels |
| **Outcomes** | Sepsis, deterioration, mortality | Pregnancy complications, maternal outcomes |
| **Decision Support** | Real-time alerts for clinicians | Longitudinal risk stratification for CHWs |
| **Population** | Critically ill inpatients | Pregnant women in community |
| **Feature Frequency** | Hourly vitals, 6-24hr labs | Monthly ANC visits, quarterly labs |

---

## Why Separation is Critical

### Clinical Context Mismatch
- **ICU models** predict **acute physiological deterioration** (shock, organ failure)
- **Maternal models** predict **chronic risk progression** (anemia, hypertension, diabetes)

### Data Structure Incompatibility
- **ICU data**: Dense time-series (hourly vitals, frequent labs)
- **Maternal data**: Sparse longitudinal (monthly visits, quarterly labs)

### Model Architecture Requirements
- **ICU models**: Real-time streaming inference (Flink), <10ms latency
- **Maternal models**: Batch risk scoring, weekly updates acceptable

### Training Data Characteristics
- **ICU training**: MIMIC-IV (U.S. tertiary ICU, age 65+, organ failure)
- **Maternal training**: Indian population (age 18-35, pregnancy physiology)

---

## Recommended Action Plan

### Immediate (Week 1)
1. ✅ **Deploy Module 5 (ICU models) with MIMIC-IV** in shadow mode
   - Models are validated and ready
   - Begin collecting local calibration data
   - No risk to patients (shadow mode only)

2. 📋 **Design Maternal Care Module** (Track 2)
   - Architect new model pipeline
   - Define feature engineering for maternal data
   - Plan CHW workflow integration

### Short-term (Weeks 2-8)
1. **Track 1 (ICU)**:
   - Shadow mode data collection
   - Monitor prediction distributions
   - Prepare for calibration phase

2. **Track 2 (Maternal)**:
   - Integrate Mendeley + NidaanKosha datasets
   - Develop maternal risk progression models
   - Validate with obstetric clinicians

### Medium-term (Weeks 9-16)
1. **Track 1 (ICU)**:
   - Apply calibration layer to MIMIC-IV models
   - Tune thresholds for Indian population
   - Begin advisory mode pilot

2. **Track 2 (Maternal)**:
   - Deploy maternal models in CHW workflow
   - Shadow mode for maternal risk scoring
   - Collect outcomes for validation

### Long-term (3-6 months)
1. **Track 1 (ICU)**:
   - Pursue Indian ICU dataset partnerships (ICMR, ISCCM)
   - Fine-tune models when dataset available
   - Full clinical deployment with alerts

2. **Track 2 (Maternal)**:
   - Expand to preeclampsia, GDM, hemorrhage models
   - Integrate with mobile CHW apps
   - State-level deployment

---

## Dataset Access Strategy

### For Track 1 (Indian ICU Data)
**Institutional Partnerships Required**:

1. **ICMR (Indian Council of Medical Research)**
   - COVID Clinical Registry (40k+ ICU patients)
   - Requires: Research proposal + data use agreement
   - Timeline: 3-4 months

2. **ISCCM (Indian Society of Critical Care Medicine)**
   - INCIID database
   - Requires: ISCCM membership + MoU
   - Timeline: 2-3 months

3. **Academic Hospitals** (AIIMS, PGIMER, JIPMER, CMC Vellore)
   - Direct hospital partnerships
   - Requires: IRB approval + data sharing agreement
   - Timeline: 4-6 months

**Alternative**: Use shadow mode data as proxy Indian ICU dataset

### For Track 2 (Maternal Data)
**Already Available**:

1. ✅ Mendeley Maternal Health Dataset (public)
2. ✅ NidaanKosha-100k (public via HuggingFace)
3. ✅ NFHS-5 (public via Government of India)

**No institutional barriers - can start immediately**

---

## Technical Architecture

### Track 1: ICU Models (Module 5)
```
Kafka (Device Data)
    ↓
Flink Stream Processing
    ↓
MIMICFeatureExtractor (37 features + z-score standardization)
    ↓
ONNX Models (Sepsis, Deterioration, Mortality)
    ↓
Calibration Layer (Platt scaling - added post-shadow mode)
    ↓
Alert System (Kafka topic: ml-predictions-v1)
```

**Language**: Java
**Runtime**: Apache Flink 2.1.0
**Inference**: ONNX Runtime (CPU, <10ms)

### Track 2: Maternal Models (New Module)
```
HMIS Data (ANC visits)
    ↓
Feature Engineering (Maternal demographics, lab trends, ANC timeline)
    ↓
LSTM / Transformer Model (Pregnancy progression)
    ↓
Risk Scores (Anemia, GDM, Preeclampsia, Complications)
    ↓
CHW Workflow Integration (High-risk triaging)
```

**Language**: Python
**Framework**: PyTorch / TensorFlow
**Deployment**: Batch scoring (weekly updates)

---

## Success Metrics

### Track 1 (ICU Models)
- **Shadow Mode**: 500-1000 predictions collected
- **Calibration**: Brier score <0.15, calibration error <0.05
- **Discrimination**: Maintain AUROC >0.90 for sepsis, >0.75 for deterioration
- **Clinical Utility**: False positive rate <20% for high-risk alerts

### Track 2 (Maternal Models)
- **Coverage**: Predict risk for 1,000+ pregnancies
- **Accuracy**: AUROC >0.80 for anemia, >0.75 for GDM
- **Clinical Impact**: Identify 90% of high-risk pregnancies for CHW follow-up
- **Workflow Integration**: <5 min to generate risk report per patient

---

## Summary

### Key Decisions
1. ✅ **Deploy Module 5 (ICU models) with MIMIC-IV now** (shadow mode + calibration)
2. ✅ **Develop new Maternal Care Module** using Mendeley + NidaanKosha
3. ✅ **Pursue Indian ICU dataset** in parallel (3-6 month timeline)

### Why This Approach Works
- **Track 1**: Immediate deployment possible, low risk, proven models
- **Track 2**: New models designed for maternal care context, datasets available
- **Separation**: Respects clinical context differences, optimizes for each use case

### Next Steps
1. **Approve shadow mode deployment** for Module 5
2. **Kickoff maternal model design** (Track 2)
3. **Identify institutional partners** for Indian ICU dataset access

---

**Document Version**: 1.0
**Last Updated**: November 5, 2025
**Status**: ✅ **STRATEGY CLARIFIED - READY TO PROCEED WITH BOTH TRACKS**
