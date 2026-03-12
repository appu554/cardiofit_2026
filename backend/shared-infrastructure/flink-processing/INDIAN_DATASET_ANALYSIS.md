# Indian Dataset Analysis for MIMIC-IV Fine-tuning

**Date**: November 5, 2025
**Status**: ⚠️ **DATASET SUITABILITY ASSESSMENT REQUIRED**
**Purpose**: Evaluate Indian datasets for fine-tuning MIMIC-IV sepsis/deterioration/mortality models

---

## Executive Summary

### Critical Finding
Neither of the initially identified datasets is suitable for fine-tuning ICU risk prediction models:

1. **NidaanKosha-100k** ❌ Lab test records (not patient-level ICU data)
2. **Mendeley ngtpyv4zws** ❌ Maternal health statistics (population-level aggregates)

### What We Need
**Indian ICU patient-level dataset** with:
- Demographics (age, gender)
- Vital signs time-series (HR, BP, RR, Temp, SpO2)
- Lab values (WBC, Hgb, Creatinine, Lactate, etc.)
- Clinical outcomes (sepsis, deterioration, mortality)
- Minimum 500-1000 patients for effective fine-tuning

---

## Dataset Analysis Results

### 1. NidaanKosha-100k (HuggingFace)

**Source**: https://huggingface.co/datasets/ekacare/NidaanKosha-100k-V1.0

**Description**: 100,000+ clinical lab test records from EkaCare platform

#### Dataset Structure
```
Columns (9):
- document_id: Patient document identifier
- age: Patient age
- gender: Patient gender
- test_name: Lab test name (e.g., "% TRANSFERRIN SATURATION")
- display_ranges: Normal ranges
- value: Test result value
- unit: Unit of measurement
- specimen: Sample type (blood, urine)
- loinc: LOINC code for test
```

#### Sample Record
```json
{
  "document_id": "a1ce7f03fca11a7351927f5d6d635552",
  "age": 72,
  "gender": "female",
  "test_name": "% TRANSFERRIN SATURATION",
  "value": "25.4100",
  "unit": "%",
  "specimen": "blood",
  "loinc": "2502-3"
}
```

#### Suitability Assessment

**❌ UNSUITABLE for ICU Risk Model Fine-tuning**

**Reasons**:
1. **Lab-test-level data**: Each row is a single lab test, not a patient encounter
2. **No vital signs**: Missing HR, BP, RR, Temp, SpO2 (15/37 features = 40% of MIMIC features)
3. **No clinical scores**: Missing SOFA, GCS, NEWS2, qSOFA (8/37 features = 22%)
4. **No temporal context**: No time-series data for tracking deterioration
5. **No outcomes**: No sepsis diagnosis, mortality, or deterioration labels
6. **Feature coverage**: Only 2/37 MIMIC-IV features (5.4%)

**Data Model Mismatch**:
- **NidaanKosha**: One row per lab test → Multiple rows per patient
- **MIMIC-IV**: One row per patient encounter with aggregated features

**Use Case**:
- ✅ **Could be useful for**: OPD progression models, chronic disease monitoring
- ❌ **NOT suitable for**: ICU severity prediction, acute deterioration detection

---

### 2. Mendeley Dataset (ngtpyv4zws)

**Source**: https://data.mendeley.com/datasets/ngtpyv4zws/2

**Title**: "Maternal Health Service Coverage in India"

**Description**: Population-level maternal health statistics at sub-national level

#### Dataset Structure
- **Level**: State/Union Territory aggregates (NOT individual patients)
- **Metrics**: ANC coverage %, INC coverage %, PNC coverage %
- **Time period**: 2017-2020 (3 financial years)
- **Source**: Health Management Information System (HMIS), National Health Mission

#### Suitability Assessment

**❌ COMPLETELY UNSUITABLE for ICU Risk Model Fine-tuning**

**Reasons**:
1. **Population-level aggregates**: No individual patient records
2. **Maternal health focus**: Pregnancy/childbirth outcomes, not ICU severity
3. **Coverage statistics**: Service utilization rates, not clinical measurements
4. **No ICU data**: No vitals, labs, or critical care metrics

**Use Case**:
- ✅ **Suitable for**: Public health policy, maternal care coverage analysis
- ❌ **NOT suitable for**: Any clinical ML model at patient level

---

## Required Dataset Characteristics

### For ICU Risk Model Fine-tuning (Sepsis/Deterioration/Mortality)

#### Minimum Requirements
| Category | Required Features | MIMIC-IV Equivalent |
|----------|------------------|---------------------|
| **Demographics** | Age, Gender | 2 features |
| **Vital Signs** | HR, BP, RR, Temp, SpO2 (time-series) | 15 features (mean/min/max/std) |
| **Lab Values** | WBC, Hgb, Platelets, Creatinine, Lactate, BUN, Glucose, Electrolytes | 12 features |
| **Clinical Scores** | SOFA, GCS (or components) | 8 features |
| **Outcomes** | Sepsis diagnosis, ICU mortality, deterioration events | Labels for supervised learning |

#### Data Volume
- **Minimum**: 500 patients (for calibration layer only)
- **Recommended**: 2,000+ patients (for effective fine-tuning)
- **Ideal**: 5,000+ patients (for robust fine-tuning)

#### Data Quality
- **Completeness**: ≥80% of vital signs and labs available
- **Temporal resolution**: Hourly vitals, 6-24 hour lab windows
- **Outcome labels**: Clear sepsis/mortality/deterioration outcomes
- **Population**: Indian ICU/inpatient population (secondary/tertiary care)

---

## Alternative Approaches

Given the dataset limitations, here are recommended alternatives:

### Option A: MIMIC-IV Only Deployment (Current Status)
**Status**: ✅ **READY NOW**

**Approach**:
- Deploy MIMIC-IV models as-is with feature scaling
- Use shadow mode for 4-8 weeks to collect local data
- Apply calibration layer based on observed outcomes

**Advantages**:
- ✅ Immediate deployment possible
- ✅ Models already validated (AUROC: Sepsis 98.55%, Mortality 95.70%)
- ✅ Integration tests passing (5/5)

**Disadvantages**:
- ⚠️ Distribution shift from U.S. ICU to Indian patients
- ⚠️ May need threshold tuning for Indian population
- ⚠️ Conservative predictions for moderate-risk patients

**Timeline**: Ready for deployment today

---

### Option B: Shadow Mode + Local Calibration
**Status**: ⏳ **RECOMMENDED APPROACH**

**Approach**:
1. **Deploy MIMIC-IV models in shadow mode** (no clinical alerts)
2. **Collect local data** for 4-8 weeks:
   - Patient demographics
   - Vital signs and labs
   - Clinical outcomes (sepsis, mortality, deterioration)
3. **Generate calibration curves**: Predicted vs actual outcomes
4. **Apply calibration layer**: Platt scaling or isotonic regression
5. **Tune alert thresholds** for Indian population

**Advantages**:
- ✅ Uses real deployment environment data
- ✅ Adapts to local population characteristics
- ✅ Lower risk than immediate fine-tuning
- ✅ Faster than waiting for large Indian dataset

**Timeline**: 1-2 months total
- Week 1-8: Shadow mode data collection (500-1000 predictions)
- Week 9-10: Calibration analysis and threshold tuning

---

### Option C: Find Validated Indian ICU Dataset
**Status**: ⏳ **PENDING DATASET IDENTIFICATION**

**Approach**:
- Identify Indian ICU dataset with patient-level data
- Validate dataset has required features (vitals, labs, outcomes)
- Perform transfer learning: MIMIC-IV → Indian data
- Generate fine-tuned models with Indian population statistics

**Required Dataset Sources**:
1. **Indian Medical Research Councils**: ICMR, DBT datasets
2. **Academic Hospitals**: AIIMS, PGIMER, CMC Vellore, JIPMER
3. **Industry Partners**: PharmEasy, Practo, Netmeds (if aggregated ICU data available)
4. **Public Health Datasets**: National Health Data Repository (if patient-level)

**Timeline**: 3-5 days (if dataset found) + 2-3 weeks fine-tuning

---

## Recommendations

### Immediate Action (Week 1)
1. **Deploy MIMIC-IV models (Option A)** in shadow mode
   - Models are validated and ready
   - Start collecting real deployment data
   - No clinical alerts yet (risk-free)

2. **Parallel dataset search (Option C)**
   - Contact Indian academic medical centers
   - Search ICMR/DBT public health repositories
   - Check industry partners for anonymized ICU data

3. **Set up data collection infrastructure**
   - Log all predictions with patient demographics
   - Track clinical outcomes (sepsis confirmed, ICU transfer, mortality)
   - Store for calibration analysis

### Short-term (Weeks 2-8)
1. **Continue shadow mode deployment**
   - Target: 500-1000 patient predictions
   - Monitor prediction distributions
   - Flag any obvious miscalibrations

2. **If Indian ICU dataset found**:
   - Validate dataset quality and coverage
   - Perform feature mapping (to 37 MIMIC-IV features)
   - Fine-tune models using transfer learning

3. **If no dataset found**:
   - Proceed with Option B (local calibration)
   - Use shadow mode data for calibration curves

### Medium-term (Weeks 9-12)
1. **Calibration analysis**
   - Generate reliability curves (predicted vs actual)
   - Calculate Brier score, calibration error
   - Compare with MIMIC-IV performance

2. **Threshold tuning**
   - Optimize for local population (sensitivity vs FPR)
   - Set alert thresholds:
     - RED (Critical): Sepsis ≥ 70%, Mortality ≥ 65%
     - AMBER (High): Sepsis 50-70%, Mortality 40-65%
     - GREEN (Low): Below thresholds

3. **Advisory mode pilot**
   - Show predictions to clinicians (no auto-actions)
   - Require confirmation for interventions
   - Track override rates and reasons

---

## Dataset Search Checklist

### Indian ICU Datasets to Investigate

- [ ] **ICMR Data Portal**: https://main.icmr.nic.in/data-portal
- [ ] **DBT eGOV Life Sciences**: https://dbtindia.gov.in/
- [ ] **AIIMS Delhi Research**: Contact clinical research department
- [ ] **PGIMER Chandigarh**: Contact hospital IT/research
- [ ] **CMC Vellore**: Check clinical data repository
- [ ] **JIPMER**: Academic hospital research data
- [ ] **Indian Intensive Care Database**: Check if exists (similar to eICU)
- [ ] **PharmEasy/Practo**: Industry partners (anonymized data)
- [ ] **Google Dataset Search**: Search "Indian ICU patient data"
- [ ] **Kaggle**: Search for Indian healthcare datasets
- [ ] **OpenICU**: Check for Indian hospital participation

### Dataset Validation Criteria
When evaluating potential datasets:

1. **✅ Data Level**: Patient-level (not lab-test-level or population-level)
2. **✅ Care Setting**: ICU, inpatient, or emergency department
3. **✅ Features**: Vitals + Labs + Demographics
4. **✅ Outcomes**: Sepsis diagnosis, mortality, or deterioration
5. **✅ Volume**: ≥500 patients minimum
6. **✅ Quality**: ≥80% feature completeness
7. **✅ Access**: Publicly available or obtainable with permissions

---

## Technical Notes

### NidaanKosha Data Model
If NidaanKosha is needed in future for OPD models:

**Aggregation Strategy**:
```python
# Group by document_id to create patient-level records
patient_data = nidaan_kosha.groupby('document_id').agg({
    'age': 'first',
    'gender': 'first',
    'test_name': list,  # All tests ordered
    'value': list,      # All test values
    'loinc': list       # All LOINC codes
})

# Extract specific labs by LOINC code
# Example: WBC (26464-8), Hemoglobin (718-7), Creatinine (2160-0)
```

**Limitations**:
- No vital signs (would need separate data source)
- No temporal ordering (can't track deterioration)
- Still missing clinical scores

---

## Files Generated

### Analysis Files
- [nidaan_kosha_sample.csv](scripts/indian_datasets/nidaan_kosha_sample.csv) - 100 sample records
- [feature_mapping.json](scripts/indian_datasets/feature_mapping.json) - MIMIC-IV feature mapping (2/37 features)
- [nidaan_kosha_analysis.md](scripts/indian_datasets/nidaan_kosha_analysis.md) - Detailed analysis report

### Scripts Created
- [download_nidaan_kosha.py](scripts/download_nidaan_kosha.py) - Dataset download and analysis script

---

## Summary

### Current Status
**Module 5 MIMIC-IV Integration**: ✅ **COMPLETE AND VALIDATED**
- Feature scaling implemented
- Integration tests passing (5/5)
- Clinically appropriate predictions
- Ready for deployment

**Indian Dataset Integration**: ⚠️ **AWAITING SUITABLE DATASET**
- NidaanKosha: Unsuitable (lab tests only)
- Mendeley: Unsuitable (population statistics)
- Need: Indian ICU patient-level dataset

### Recommended Path Forward
1. **Deploy MIMIC-IV models now** (shadow mode)
2. **Collect local data** (4-8 weeks)
3. **Apply calibration layer** based on real outcomes
4. **Continue search** for Indian ICU dataset for future fine-tuning

### Key Decision
**Can you provide access to a validated Indian ICU dataset**, or should we proceed with **Option B (shadow mode + local calibration)**?

---

**Document Version**: 1.0
**Last Updated**: November 5, 2025
**Status**: ⚠️ **AWAITING INDIAN ICU DATASET OR DEPLOYMENT DECISION**
