# MIMIC-IV Model Safety Verification Report

**Date**: November 5, 2025
**Status**: ⚠️ **VERIFICATION REQUIRED BEFORE CLINICAL USE**
**Risk Level**: HIGH - Models returning 99% confidence requires validation

---

## Executive Summary

✅ **Technical Integration**: ONNX models load and execute successfully
⚠️ **Clinical Safety**: Extremely high probabilities (~99%) require verification
🚫 **NOT APPROVED**: Shadow mode only until calibration validated

---

## Test Results

### Test 1: Technical Integration ✅ PASSED
- ONNX Runtime: Successfully loads 3 models (158-205 KB)
- Feature Extraction: 37 MIMIC-IV features extracted
- Inference Pipeline: EnrichedPatientContext → Features → ONNX → MLPrediction
- Data Types: Correct API usage (MLPrediction, confidence_score)

### Test 2: Clinical Plausibility ⚠️ NEEDS VALIDATION

**Patient Profile**: PAT-ROHAN-001 (Septic Shock)
```
Age: 42, HR: 108, BP: 100/60, RR: 23, Temp: 38.8°C, SpO2: 92%
Labs: WBC 15, Hgb 10, Platelets 100, Creatinine 2.5, Lactate 4.5
Scores: NEWS2=8, qSOFA=2
```

**Model Predictions**:
- Sepsis Risk: **99.56%** (expected: high, but 99% is extreme)
- Deterioration Risk: **99.81%** (expected: high, but 99% is extreme)
- Mortality Risk: **97.86%** (expected: moderate-high, but 98% is extreme)

**Clinical Assessment**:
- Profile DOES suggest severe septic shock (hypotension, elevated lactate, renal dysfunction)
- ICU transfer IS appropriate
- BUT 99% confidence suggests potential calibration issues

---

## Critical Safety Concerns

### 1. Feature Ordering & Scaling (HIGHEST PRIORITY)

**Risk**: If feature order or normalization differs from training, outputs are unpredictable.

**Verification Needed**:
- [ ] Confirm `MIMICFeatureExtractor.getFeatureNames()` order matches training input order EXACTLY
- [ ] Verify numeric scaling (standardization: (x - mean) / std OR min-max: (x - min) / (max - min))
- [ ] Check missingness handling (0-fill? mean-fill? indicator flags?)
- [ ] Test with known MIMIC samples with ground truth labels

**Action**: Run feature parity verification test (see Test Plan below)

### 2. Model Calibration

**Risk**: XGBoost often outputs poorly calibrated probabilities. High AUROC ≠ calibrated probabilities.

**Evidence**:
- All three models return 97-99% for moderate-severe case
- Low-risk patient returned 1.74% (from MIMICModelTest) - shows discrimination works
- BUT confidence levels may not reflect true probabilities

**Verification Needed**:
- [ ] Generate calibration curves (reliability diagrams)
- [ ] Calculate Brier score on validation set
- [ ] Compare with original Python model outputs on same samples
- [ ] Apply Platt scaling or isotonic regression if miscalibrated

### 3. Population Shift

**Risk**: MIMIC-IV is ICU population. Real-world CHW population differs.

**Verification Needed**:
- [ ] Compare feature distributions: MIMIC training vs local patient population
- [ ] Assess prevalence differences (ICU vs community health workers)
- [ ] Recalibrate thresholds for local population

### 4. ONNX Parity

**Risk**: ONNX export/import can introduce numerical differences.

**Verification Needed**:
- [ ] Compare Python XGBoost outputs vs Java ONNX outputs on same input
- [ ] Assert differences < 0.001 (tolerance for floating point)
- [ ] Test with 100+ diverse samples

---

## Immediate Actions Required (Before Clinical Use)

### Phase 1: Technical Verification (1-2 days)

#### Test 1: Feature Order & Scaling Verification
```java
// Create test with KNOWN training sample
@Test
public void testFeatureParityWithTrainingData() {
    // Use exact sample from MIMIC training set with known label
    EnrichedPatientContext trainingFixture = loadMIMICTrainingSample("sample_id_12345");

    float[] extractedFeatures = featureExtractor.extractFeatures(adapter.adapt(trainingFixture));
    float[] expectedFeatures = loadExpectedFeatureVector("sample_id_12345"); // From training

    // Assert exact match (or very close for floating point)
    assertArrayEquals(expectedFeatures, extractedFeatures, 0.0001);
}
```

#### Test 2: ONNX Parity Verification
```python
# Python script to compare outputs
import onnxruntime as ort
import xgboost as xgb

# Load original XGBoost model
original_model = xgb.Booster()
original_model.load_model('sepsis_risk_v2.0.0.json')

# Load ONNX model
onnx_session = ort.InferenceSession('sepsis_risk_v2.0.0_mimic.onnx')

# Test with 100 samples
for sample in validation_samples:
    original_pred = original_model.predict(xgb.DMatrix(sample))
    onnx_pred = onnx_session.run(None, {'input': sample})[0]

    assert abs(original_pred - onnx_pred) < 0.001, f"Mismatch: {original_pred} vs {onnx_pred}"
```

#### Test 3: Known Ground Truth Validation
- Use 20-50 labeled MIMIC samples (5 low-risk, 10 moderate, 5 high-risk with known outcomes)
- Compare predictions with expected ranges
- Calculate AUROC, calibration, Brier score

### Phase 2: Calibration & Thresholding (1-2 weeks)

1. **Collect Local Validation Set** (200-1000 cases)
   - Retrospective cases with outcomes (sepsis confirmed, ICU transfer, etc.)
   - Representative of actual CHW patient population

2. **Generate Calibration Curves**
   ```python
   from sklearn.calibration import calibration_curve

   # For each model
   fraction_of_positives, mean_predicted_value = calibration_curve(
       y_true, y_pred, n_bins=10, strategy='quantile'
   )
   ```

3. **Apply Calibration Correction** (if needed)
   ```python
   from sklearn.calibration import CalibratedClassifierCV

   # Platt scaling or isotonic regression
   calibrated_model = CalibratedClassifierCV(base_model, method='sigmoid', cv='prefit')
   calibrated_model.fit(X_val, y_val)
   ```

4. **Define Action Thresholds**
   - Optimize for **sensitivity ≥ 90%** for sepsis detection
   - Balance with alert burden (false positive rate)
   - Example thresholds (AFTER calibration):
     - HIGH (Red): p ≥ 0.60 → ICU referral recommended
     - MODERATE (Amber): 0.30 ≤ p < 0.60 → Increase monitoring
     - LOW (Green): p < 0.30 → Standard care

### Phase 3: Shadow Deployment (2-4 weeks)

1. **Shadow Mode Configuration**
   - Run models in production BUT don't trigger clinical alerts
   - Log all predictions, feature vectors, patient IDs
   - Track outcomes (sepsis confirmed, ICU transfer, 30-day mortality)

2. **Monitoring Dashboards**
   - Prediction distribution (histogram of risk scores)
   - % patients by risk tier (Low/Moderate/High)
   - Feature distribution drift detection
   - Inference latency and error rates

3. **Outcome Linking**
   - Link predictions to clinical outcomes (30-day window)
   - Calculate prospective AUROC, sensitivity, specificity
   - Identify miscalibration patterns

---

## Safety Guardrails (MUST IMPLEMENT)

### 1. Advisory Only (No Automated Actions)
- ✅ Show predictions to clinicians with HIGH-PRIORITY banner
- ✅ Provide recommended actions (ICU referral, increase monitoring)
- ❌ NO automatic orders, referrals, or medication changes
- ✅ REQUIRE clinician confirmation for any action

### 2. Explainability Output
```java
// Add to MLPrediction
Map<String, Double> getTopContributingFeatures(int topN) {
    // Return top-N features by SHAP value or importance
    // Example: {"lactate": 0.45, "systolic_bp": -0.32, "heart_rate": 0.28}
}
```

### 3. Audit Trail (Every Prediction)
```json
{
  "prediction_id": "uuid",
  "patient_id": "PAT-ROHAN-001",
  "timestamp": "2025-11-05T10:54:32Z",
  "model_version": "sepsis_risk_v2.0.0_mimic",
  "onnx_checksum": "sha256:abc123...",
  "input_features": [42.0, 1.0, 108.0, ...],
  "feature_names": ["age", "gender_male", "heart_rate_mean", ...],
  "prediction": {
    "sepsis_risk": 0.9956,
    "deterioration_risk": 0.9981,
    "mortality_risk": 0.9786
  },
  "top_features": {
    "lactate_mean": 0.45,
    "systolic_bp_mean": -0.32,
    "heart_rate_mean": 0.28
  },
  "clinician_action": "referred_to_icu",
  "clinician_override": false,
  "outcome_confirmed": null  // Link later
}
```

### 4. Drift Detection Alerts
```java
// Monitor feature distributions
if (abs(current_mean - training_mean) > 2 * training_std) {
    LOG.warn("Feature drift detected: {} - current: {}, training: {}",
        featureName, current_mean, training_mean);
    // Send alert to MLOps team
}
```

### 5. Override Tracking
- Log every clinician override with reason
- Analyze patterns monthly
- Retrain if override rate > 30%

---

## Monitoring Metrics (Production)

### Model Performance
- AUROC (track weekly, alert if drops > 5%)
- Calibration curves (monthly)
- Brier score (track weekly)
- Alert rate (% high-risk predictions per day)

### Operational
- Inference latency (p50, p95, p99)
- Error rate (failed predictions / total)
- Feature missingness rate
- ONNX runtime exceptions

### Clinical Outcomes
- Sensitivity for sepsis detection (confirmed cases)
- Specificity (false positive rate)
- Positive predictive value
- Time to clinician action
- Override rate and reasons

### Drift Detection
- Feature mean/std by week
- Population characteristics shift
- Alert rate trends
- Outcome rate trends

---

## Deployment Roadmap

### Stage 1: SHADOW MODE (4-8 weeks)
- Models run but NO clinician alerts
- Collect predictions + outcomes
- Validate calibration
- Tune thresholds

**Exit Criteria**:
- Calibration Brier score < 0.15
- Prospective AUROC > 0.75
- Feature distributions stable (no drift alerts)
- < 1% ONNX runtime errors

### Stage 2: ADVISORY MODE (8-12 weeks)
- Show predictions to clinicians (HIGH-PRIORITY banner)
- Require confirmation for actions
- Track override rates
- Refine thresholds based on feedback

**Exit Criteria**:
- Clinician override rate < 30%
- Sensitivity ≥ 90% for sepsis detection
- False positive rate acceptable (< 20%)
- Clinician satisfaction > 3.5/5

### Stage 3: AUTOMATED ALERTS (After Stage 2 success)
- Trigger automated workflows (WITH clinician review)
- Example: HIGH risk → auto-schedule ICU consult
- Maintain human-in-loop for critical decisions

---

## Current Status: HOLD

### ✅ Ready for Shadow Mode:
- Technical integration complete
- Inference pipeline working
- Audit logging ready
- Monitoring dashboards (need deployment)

### ⚠️ Blockers for Shadow Mode:
1. **Feature parity verification test** (MUST COMPLETE)
2. **ONNX parity validation** (MUST COMPLETE)
3. **Calibration analysis** (MUST COMPLETE)
4. **Local validation set** (200-1000 cases with outcomes)

### 🚫 NOT READY FOR:
- Clinical alerts or automated actions
- Field pilot without shadow mode
- Any production use affecting care decisions

---

## Recommendations

1. **Complete verification tests FIRST** (1-2 days)
   - Feature parity test with known MIMIC samples
   - ONNX parity test comparing Python vs Java outputs
   - Ground truth validation with 20-50 labeled cases

2. **Deploy to shadow mode** (after verification passes)
   - 1 facility, 500-1000 patients
   - 4-8 weeks collection period
   - Full audit logging enabled

3. **Analyze shadow mode results**
   - Calculate calibration metrics
   - Tune thresholds for local population
   - Identify any distribution shift

4. **Advisory mode pilot** (after calibration validated)
   - Show predictions to clinicians
   - Collect override reasons
   - Refine based on feedback

5. **Production rollout** (after pilot success)
   - Gradual expansion to more facilities
   - Continuous monitoring
   - Quarterly model retraining

---

## Next Steps for You

**Immediate** (Before Module 5 full pipeline test):
1. Run feature parity verification test
2. Check ONNX parity with Python model
3. Document any discrepancies found

**Short-term** (1-2 weeks):
1. Collect local validation set (if available)
2. Generate calibration curves
3. Define action thresholds

**Medium-term** (1-2 months):
1. Deploy shadow mode
2. Collect predictions + outcomes
3. Analyze results and tune

---

**Document Version**: 1.0
**Last Updated**: November 5, 2025
**Status**: ⚠️ VERIFICATION IN PROGRESS
**Next Review**: After feature parity verification complete
