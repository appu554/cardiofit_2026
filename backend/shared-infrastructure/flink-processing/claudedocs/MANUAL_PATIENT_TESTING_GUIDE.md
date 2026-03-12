# Manual Patient Data Testing Guide - MIMIC-IV ML Models

**Purpose**: Test MIMIC-IV machine learning models with your own patient data and see risk predictions

**Date**: November 5, 2025
**Status**: ✅ Ready to use

---

## Quick Start

### Option 1: Interactive Script (Recommended)

Run the interactive script that will prompt you for patient data:

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing
./test-manual-patient.sh
```

The script will ask you for:
- **Patient Demographics**: ID, age, gender, weight
- **Vital Signs**: Heart rate, blood pressure, respiratory rate, temperature, SpO2
- **Lab Values**: WBC, hemoglobin, platelets, creatinine, BUN, glucose, sodium, potassium, lactate
- **Clinical Scores**: NEWS2, qSOFA

Then it will show you:
- ✅ Patient profile summary
- ✅ Feature extraction status (37 MIMIC-IV features)
- ✅ Risk predictions for:
  - **Sepsis** (probability %)
  - **Clinical Deterioration** (probability %)
  - **Mortality** (probability %)
- ✅ Risk classification (LOW / MODERATE / HIGH)
- ✅ Clinical recommendations

### Option 2: Use Example Data

Check the example file for reference values:

```bash
cat example-patient-data.txt
```

This file contains:
- Normal reference ranges for all parameters
- Three example scenarios (low-risk, moderate-risk, high-risk)
- Guidance on what values to enter

---

## Understanding the Output

### Sample Output Format

```
╔════════════════════════════════════════════════════════════════╗
║     MIMIC-IV Model Predictions                                 ║
╚════════════════════════════════════════════════════════════════╝

📋 PATIENT PROFILE:
   ID: PT-12345
   Age: 65 years
   Gender: M
   Weight: 75 kg

❤️ VITAL SIGNS:
   Heart Rate: 85 bpm
   Blood Pressure: 130/85 mmHg
   Respiratory Rate: 18 breaths/min
   Temperature: 37.2 °C
   SpO2: 96%

🧪 KEY LAB VALUES:
   WBC: 9.5 10^3/uL
   Hemoglobin: 13.5 g/dL
   Creatinine: 1.1 mg/dL
   Lactate: 1.5 mmol/L

📊 CLINICAL SCORES:
   NEWS2: 3
   qSOFA: 0

🔬 FEATURE EXTRACTION:
   ✅ Extracted 37 MIMIC-IV features

🤖 LOADING ML MODELS...
═══════════════════════════════════════════════════════════════
✅ All 3 MIMIC-IV models loaded successfully!

🔮 RISK PREDICTIONS:
═══════════════════════════════════════════════════════════════
   Sepsis Risk         : 0.0234 (2.34%) - ✅ LOW RISK
   Deterioration Risk  : 0.1567 (15.67%) - ✅ LOW RISK
   Mortality Risk      : 0.0189 (1.89%) - ✅ LOW RISK

═══════════════════════════════════════════════════════════════

💡 CLINICAL RECOMMENDATIONS:
   ✅ LOW RISK - Continue standard care
   • Maintain routine monitoring
   • Follow standard protocols

✅ Analysis Complete!
```

### Risk Level Classification

| Risk Score | Classification | Action |
|------------|----------------|--------|
| **< 30%** | ✅ LOW RISK | Standard care, routine monitoring |
| **30-50%** | ⚡ LOW-MODERATE RISK | Increased monitoring consideration |
| **50-80%** | ⚠️ MODERATE RISK | Enhanced monitoring, additional labs |
| **> 80%** | 🚨 HIGH RISK | Immediate intervention, ICU consultation |

---

## Example Test Scenarios

### Scenario 1: Normal Healthy Patient (Expected: LOW RISK)

```
Age: 45
HR: 72 bpm | BP: 118/76 mmHg | RR: 14 | Temp: 36.8°C | SpO2: 99%
WBC: 7.5 | Hgb: 14.0 | Creatinine: 0.9 | Lactate: 1.0
NEWS2: 0 | qSOFA: 0

Expected Results:
- Sepsis: ~2-5%
- Deterioration: ~10-20%
- Mortality: ~1-3%
```

### Scenario 2: Moderate Risk Patient (Expected: MODERATE RISK)

```
Age: 70
HR: 105 bpm | BP: 145/92 mmHg | RR: 22 | Temp: 38.1°C | SpO2: 93%
WBC: 13.5 | Hgb: 11.0 | Creatinine: 1.8 | Lactate: 2.8
NEWS2: 6 | qSOFA: 1

Expected Results:
- Sepsis: ~40-70%
- Deterioration: ~50-80%
- Mortality: ~30-60%
```

### Scenario 3: High Risk Septic Patient (Expected: HIGH RISK)

```
Age: 78
HR: 125 bpm | BP: 88/55 mmHg | RR: 28 | Temp: 39.2°C | SpO2: 89%
WBC: 18.5 | Hgb: 9.5 | Creatinine: 2.8 | Lactate: 4.5
NEWS2: 12 | qSOFA: 3

Expected Results:
- Sepsis: >95%
- Deterioration: >95%
- Mortality: >90%
```

---

## Clinical Parameter Reference

### Normal Ranges

#### Vital Signs
- **Heart Rate**: 60-100 bpm
- **Blood Pressure**: 90-140 / 60-90 mmHg
- **Respiratory Rate**: 12-20 breaths/min
- **Temperature**: 36.1-37.2 °C (96.8-99.0 °F)
- **SpO2**: ≥95%

#### Laboratory Values
- **WBC**: 4.5-11.0 × 10³/μL
- **Hemoglobin**: 13.5-17.5 g/dL (male), 12.0-15.5 g/dL (female)
- **Platelets**: 150-400 × 10³/μL
- **Creatinine**: 0.7-1.3 mg/dL (male), 0.6-1.1 mg/dL (female)
- **BUN**: 7-20 mg/dL
- **Glucose**: 70-100 mg/dL (fasting)
- **Sodium**: 136-145 mmol/L
- **Potassium**: 3.5-5.0 mmol/L
- **Lactate**: 0.5-2.0 mmol/L

#### Clinical Scores
- **NEWS2**: 0-4 (low), 5-6 (medium), 7+ (high risk)
- **qSOFA**: 0-1 (low risk), 2+ (sepsis suspected)

---

## Technical Details

### Data Flow

```
Your Patient Data
    ↓
EnrichedPatientContext (created from input)
    ↓
PatientContextAdapter
    ↓
PatientContextSnapshot (37 clinical features)
    ↓
MIMICFeatureExtractor
    ↓
ONNX Models (Sepsis, Deterioration, Mortality)
    ↓
Risk Predictions + Recommendations
```

### MIMIC-IV Models Used

1. **Sepsis Risk Model v2.0.0**
   - Training: MIMIC-IV v3.1 dataset
   - Performance: AUROC 98.55%
   - Input: 37 clinical features
   - Output: Sepsis onset probability

2. **Clinical Deterioration Model v2.0.0**
   - Training: MIMIC-IV v3.1 dataset
   - Performance: AUROC 78.96%
   - Input: 37 clinical features
   - Output: Deterioration probability

3. **Mortality Risk Model v2.0.0**
   - Training: MIMIC-IV v3.1 dataset
   - Performance: AUROC 95.70%
   - Input: 37 clinical features
   - Output: In-hospital mortality probability

### 37 MIMIC-IV Features

**Demographics (2)**:
- Age
- Gender

**Vital Signs (16)**:
- Heart rate (current, min, max, mean)
- Respiratory rate (current, min, max, mean)
- Temperature (current, min, max, mean)
- Blood pressure - systolic/diastolic (min, max)
- Mean arterial pressure (MAP)
- Oxygen saturation (SpO2)

**Laboratory Values (13)**:
- White blood cells (WBC)
- Hemoglobin
- Platelets
- Creatinine
- Blood urea nitrogen (BUN)
- Glucose
- Sodium
- Potassium
- Lactate
- Bilirubin

**Clinical Scores (6)**:
- NEWS2 score
- qSOFA score
- SOFA total
- SOFA components (cardiovascular, respiratory, renal)
- Glasgow Coma Scale (GCS)

**Note**: Current implementation uses latest values for mean approximations. Full Module 5 pipeline will aggregate from time windows for accurate means/mins/maxes.

---

## Troubleshooting

### Models Not Found

**Error**: "❌ Error loading ONNX models"

**Solution**: Verify models exist:
```bash
ls -lh models/*.onnx
```

Should show:
- sepsis_risk_v2.0.0_mimic.onnx (158KB)
- deterioration_risk_v2.0.0_mimic.onnx (205KB)
- mortality_risk_v2.0.0_mimic.onnx (187KB)

### Compilation Errors

**Error**: Compilation failures

**Solution**: Recompile the project:
```bash
mvn clean compile
```

### Invalid Patient Data

**Error**: "incompatible types" or parsing errors

**Solution**: Check input format:
- Age: Integer (18-100)
- Vitals/Labs: Decimal numbers (use . not ,)
- Gender: M or F
- Scores: Integers (NEWS2: 0-20, qSOFA: 0-3)

---

## Next Steps

After validating models with your patient data:

1. **Module 5 Full Pipeline Test**:
   - Start Module 2 (produces EnrichedPatientContext)
   - Start Module 5 with MIMIC-IV integration
   - Verify predictions in Kafka topics

2. **Production Deployment**:
   - Deploy Module 5 with MIMIC-IV operator
   - Configure monitoring dashboards
   - Set up alerting rules

3. **Performance Validation**:
   - Measure inference latency (<10ms expected)
   - Monitor throughput (>1000 predictions/sec)
   - Track prediction quality metrics

---

## Files

### Testing Scripts
- [test-manual-patient.sh](../test-manual-patient.sh) - Interactive testing script
- [example-patient-data.txt](../example-patient-data.txt) - Reference values and examples

### Test Classes
- [RealPatientDataMLTest.java](../src/test/java/com/cardiofit/flink/ml/RealPatientDataMLTest.java) - JUnit tests for adapter and feature extraction
- [MIMICModelTest.java](../src/test/java/MIMICModelTest.java) - Direct ONNX model testing

### Implementation Files
- [PatientContextAdapter.java](../src/main/java/com/cardiofit/flink/adapters/PatientContextAdapter.java) - Data type converter
- [MIMICFeatureExtractor.java](../src/main/java/com/cardiofit/flink/ml/features/MIMICFeatureExtractor.java) - Feature extraction
- [MIMICMLInferenceOperator.java](../src/main/java/com/cardiofit/flink/operators/MIMICMLInferenceOperator.java) - Flink operator

---

**Document Version**: 1.0
**Last Updated**: November 5, 2025
**Status**: ✅ Ready for manual patient testing
