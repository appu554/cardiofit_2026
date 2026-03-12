# MODULE 5: MODEL DEPLOYMENT CHECKLIST

**Last Updated**: November 1, 2025
**Status**: Operational Checklist
**Version**: 1.0.0
**Target Audience**: DevOps, ML Engineers, Clinical Integration Teams, Project Managers

---

## DOCUMENT PURPOSE

This document provides a comprehensive pre-deployment checklist for the four clinical prediction models (Sepsis, Deterioration, Mortality, Readmission). Use this checklist to verify all requirements are met before production deployment.

**Duration**: ~2 weeks per model (validate → A/B test → full rollout)

**Sign-Off Required**: ML Engineering, Clinical Affairs, IT Security, Medical Records

---

## TABLE OF CONTENTS

1. [Phase 1: Pre-Deployment Validation](#phase-1-pre-deployment-validation)
2. [Phase 2: Performance Benchmarking](#phase-2-performance-benchmarking)
3. [Phase 3: Integration Testing](#phase-3-integration-testing)
4. [Phase 4: Security & Compliance](#phase-4-security--compliance)
5. [Phase 5: A/B Testing Setup](#phase-5-ab-testing-setup)
6. [Phase 6: Monitoring Configuration](#phase-6-monitoring-configuration)
7. [Phase 7: Rollback Planning](#phase-7-rollback-planning)
8. [Phase 8: Sign-Off & Deployment](#phase-8-sign-off--deployment)
9. [Post-Deployment Verification](#post-deployment-verification)

---

## PHASE 1: PRE-DEPLOYMENT VALIDATION

### 1.1 Model Artifact Verification

**Objective**: Confirm model file integrity and metadata correctness

#### Checklist

- [ ] **Model File Integrity**
  - [ ] ONNX model file exists at expected path
  - [ ] File size < 50MB (target: 20-40MB with quantization)
  - [ ] File is readable (no corruption)
  - [ ] SHA256 checksum matches expected value

  ```bash
  # Verify file size
  ls -lh sepsis_v1.0.0.onnx
  # Expected output: -rw-r--r-- 1 user group 35M Nov 1 2025 sepsis_v1.0.0.onnx

  # Verify checksum
  sha256sum sepsis_v1.0.0.onnx > sepsis_v1.0.0.onnx.sha256
  # Compare with expected value
  ```

- [ ] **ONNX Syntax Validation**
  - [ ] ONNX model passes syntax check
  - [ ] Opset version appropriate (12+)
  - [ ] All node types supported by ONNX Runtime

  ```python
  import onnx
  model = onnx.load('sepsis_v1.0.0.onnx')
  onnx.checker.check_model(model)
  print("✓ Model syntax valid")
  ```

- [ ] **Metadata Completeness**
  - [ ] Model name specified (e.g., "sepsis")
  - [ ] Version number specified (e.g., "1.0.0")
  - [ ] Producer name set to "CardioFit-Module5"
  - [ ] Documentation string present
  - [ ] Training date recorded

  ```python
  import onnx
  model = onnx.load('sepsis_v1.0.0.onnx')
  print(f"Producer: {model.producer_name}")
  print(f"Version: {model.producer_version}")
  for prop in model.metadata_props:
      print(f"{prop.key}: {prop.value}")
  ```

### 1.2 Input/Output Schema Validation

**Objective**: Verify tensor shapes and data types match specification

#### Checklist

- [ ] **Input Tensor Specification**
  - [ ] Input name: "float_input" (or documented alternative)
  - [ ] Shape: (batch_size, 70) - dynamic batch, 70 features
  - [ ] Data type: float32
  - [ ] Value range: [0.0, 1.0] after normalization

  ```python
  from onnxruntime import InferenceSession
  session = InferenceSession('sepsis_v1.0.0.onnx')

  input_info = session.get_inputs()[0]
  print(f"Input name: {input_info.name}")
  print(f"Input shape: {input_info.shape}")
  print(f"Input type: {input_info.type}")

  # Expected output:
  # Input name: float_input
  # Input shape: ['batch_size', 70]
  # Input type: float32
  ```

- [ ] **Output Tensor Specification**
  - [ ] Output name documented (typically "probabilities")
  - [ ] Shape: (batch_size, 2)
  - [ ] Data type: float32
  - [ ] Values constrained to [0.0, 1.0]
  - [ ] Probability sum per sample = 1.0

  ```python
  output_info = session.get_outputs()[0]
  print(f"Output name: {output_info.name}")
  print(f"Output shape: {output_info.shape}")
  print(f"Output type: {output_info.type}")
  ```

- [ ] **Tensor Validation Test**
  - [ ] Test with batch size 1
  - [ ] Test with batch size 32 (typical batch)
  - [ ] Test with valid feature ranges
  - [ ] Verify output shapes are correct

  ```python
  import numpy as np
  from onnxruntime import InferenceSession

  session = InferenceSession('sepsis_v1.0.0.onnx')
  input_name = session.get_inputs()[0].name
  output_name = session.get_outputs()[0].name

  # Test 1: Single prediction
  test_input = np.random.rand(1, 70).astype(np.float32)
  output = session.run([output_name], {input_name: test_input})
  assert output[0].shape == (1, 2), f"Expected (1, 2), got {output[0].shape}"

  # Test 2: Batch prediction
  test_batch = np.random.rand(32, 70).astype(np.float32)
  output = session.run([output_name], {input_name: test_batch})
  assert output[0].shape == (32, 2), f"Expected (32, 2), got {output[0].shape}"

  # Test 3: Probability constraints
  assert np.all(output[0] >= 0.0) and np.all(output[0] <= 1.0), \
      "Probabilities outside [0, 1] range"
  assert np.allclose(output[0].sum(axis=1), 1.0, atol=1e-5), \
      "Probabilities don't sum to 1.0"

  print("✓ All tensor validation tests passed")
  ```

### 1.3 Numerical Equivalence Verification

**Objective**: Confirm ONNX model produces same predictions as original

#### Checklist

- [ ] **Compare to Training Model**
  - [ ] Original model (XGBoost/LightGBM) loaded successfully
  - [ ] Test data set prepared (≥100 samples)
  - [ ] Predictions generated from both models
  - [ ] Difference in probabilities ≤ 0.01 (acceptable tolerance)

  ```python
  import numpy as np
  from onnxruntime import InferenceSession

  # Load original model
  original_model = xgb.Booster(model_file='sepsis_original.xgb')

  # Load ONNX model
  session = InferenceSession('sepsis_v1.0.0.onnx')
  input_name = session.get_inputs()[0].name
  output_name = session.get_outputs()[0].name

  # Test on sample data
  test_data = np.random.rand(100, 70).astype(np.float32)

  # Original predictions
  original_pred = original_model.predict(
      xgb.DMatrix(test_data)
  )

  # ONNX predictions
  onnx_output = session.run([output_name], {input_name: test_data})[0]
  onnx_pred = onnx_output[:, 1]  # Risk probabilities

  # Compare
  max_diff = np.max(np.abs(original_pred - onnx_pred))
  print(f"Max difference: {max_diff:.6f}")

  assert max_diff < 0.01, f"Difference too large: {max_diff}"
  print("✓ Numerical equivalence verified")
  ```

- [ ] **Cross-Platform Testing**
  - [ ] Test on Windows (if deployment target)
  - [ ] Test on Linux (production platform)
  - [ ] Test on macOS (if development platform)
  - [ ] Verify results identical across platforms

### 1.4 Data Consistency Verification

**Objective**: Confirm preprocessing pipeline matches training

#### Checklist

- [ ] **Feature Order Validation**
  - [ ] Feature extraction order documented
  - [ ] Feature names match specification exactly
  - [ ] Normalization method matches training
  - [ ] Imputation strategy documented

  ```python
  # Feature order must match exactly
  FEATURE_ORDER = [
      # Demographics (5)
      'demo_age_years', 'demo_gender_male', 'demo_bmi', 'demo_icu_patient', 'demo_admission_emergency',
      # Vitals (12)
      'vital_heart_rate', 'vital_systolic_bp', ...
      # ... all 70 features in exact order
  ]

  # Verify in production code
  assert len(features_dict) == 70, "Incorrect number of features"
  feature_vector = np.array([features_dict[f] for f in FEATURE_ORDER])
  assert feature_vector.shape == (70,), "Incorrect feature shape"
  ```

- [ ] **Normalization Consistency**
  - [ ] Scaler object saved from training
  - [ ] Scaler parameters (mean, std) documented
  - [ ] Scaler applied in production pipeline
  - [ ] Validation: scaled values in [0, 1] (or approximately [-3, 3] for z-score)

  ```python
  import joblib
  from sklearn.preprocessing import StandardScaler

  # Load training scaler
  scaler = joblib.load('normalizer_sepsis_v1.0.0.pkl')

  # Apply to production data
  raw_features = extract_features(patient_data)  # Shape: (70,)
  normalized_features = scaler.transform(raw_features.reshape(1, -1))[0]

  # Verify range
  assert np.all(normalized_features >= -3) and np.all(normalized_features <= 3), \
      "Normalized features out of expected range"
  ```

- [ ] **Missing Data Imputation**
  - [ ] Imputation strategy for each feature documented
  - [ ] Imputation parameters (medians, modes) saved from training
  - [ ] Applied consistently in production
  - [ ] Validation: no NaN values in final feature vector

---

## PHASE 2: PERFORMANCE BENCHMARKING

### 2.1 Inference Latency Measurement

**Objective**: Verify model meets latency requirement (<15ms p99)

#### Checklist

- [ ] **Latency Test Setup**
  - [ ] Test data prepared (1000 realistic samples)
  - [ ] Warm-up runs completed (2-3 iterations)
  - [ ] CPU affinity set if needed (performance consistency)
  - [ ] Memory pre-allocated

  ```python
  import numpy as np
  import time
  from onnxruntime import InferenceSession

  session = InferenceSession('sepsis_v1.0.0.onnx')
  input_name = session.get_inputs()[0].name
  output_name = session.get_outputs()[0].name

  # Prepare test data
  test_data = np.random.rand(1000, 70).astype(np.float32)

  # Warm-up runs
  for _ in range(3):
      session.run([output_name], {input_name: test_data[0:1]})
  ```

- [ ] **Single Prediction Latency**
  - [ ] Measure time for 100 individual predictions
  - [ ] Calculate mean, p50, p95, p99 latency
  - [ ] P99 latency < 15ms
  - [ ] Mean latency < 5ms

  ```python
  latencies = []
  for i in range(100):
      start = time.perf_counter()
      session.run([output_name], {input_name: test_data[i:i+1]})
      elapsed = (time.perf_counter() - start) * 1000  # Convert to ms
      latencies.append(elapsed)

  latencies = np.array(latencies)
  print(f"Mean latency:  {latencies.mean():.2f}ms")
  print(f"P50 latency:   {np.percentile(latencies, 50):.2f}ms")
  print(f"P95 latency:   {np.percentile(latencies, 95):.2f}ms")
  print(f"P99 latency:   {np.percentile(latencies, 99):.2f}ms")

  assert np.percentile(latencies, 99) < 15, "P99 latency exceeds 15ms limit"
  print("✓ Latency requirement met")
  ```

- [ ] **Batch Inference Latency**
  - [ ] Measure time for batch of 32 predictions
  - [ ] Calculate throughput (predictions/second)
  - [ ] Throughput > 100 predictions/second
  - [ ] Amortized latency per prediction < 10ms

  ```python
  batch_size = 32
  num_batches = 100

  start = time.perf_counter()
  for i in range(num_batches):
      batch_idx = (i * batch_size) % len(test_data)
      session.run([output_name], {input_name: test_data[batch_idx:batch_idx+batch_size]})
  elapsed = time.perf_counter() - start

  total_predictions = num_batches * batch_size
  throughput = total_predictions / elapsed
  latency_per_pred = elapsed / total_predictions * 1000

  print(f"Throughput: {throughput:.0f} predictions/second")
  print(f"Latency per prediction: {latency_per_pred:.2f}ms")

  assert throughput > 100, f"Throughput too low: {throughput}"
  print("✓ Batch throughput requirement met")
  ```

### 2.2 Memory Footprint Analysis

**Objective**: Verify memory requirements acceptable

#### Checklist

- [ ] **Model Load Memory**
  - [ ] Model loaded into memory
  - [ ] Peak memory usage measured
  - [ ] Memory < 200MB
  - [ ] Memory stable (no leaks)

  ```python
  import psutil
  import os

  process = psutil.Process(os.getpid())

  # Get baseline memory
  baseline_memory = process.memory_info().rss / 1e6  # MB

  # Load model
  from onnxruntime import InferenceSession
  session = InferenceSession('sepsis_v1.0.0.onnx')

  # Get peak memory
  peak_memory = process.memory_info().rss / 1e6  # MB
  model_memory = peak_memory - baseline_memory

  print(f"Model memory footprint: {model_memory:.1f}MB")
  assert model_memory < 200, f"Memory footprint too large: {model_memory}MB"
  ```

- [ ] **Inference Memory Usage**
  - [ ] Run 100 inferences
  - [ ] Monitor memory during inference
  - [ ] No unbounded memory growth
  - [ ] Garbage collection behaves normally

### 2.3 Model File Size Verification

**Objective**: Confirm model size acceptable for deployment

#### Checklist

- [ ] **File Size Check**
  - [ ] Uncompressed size < 50MB
  - [ ] Compressed size (gzip) < 30MB
  - [ ] Size acceptable for clinical systems

  ```bash
  # Check uncompressed size
  du -h sepsis_v1.0.0.onnx
  # Expected: 20-40MB

  # Check compressed size
  gzip -k sepsis_v1.0.0.onnx
  du -h sepsis_v1.0.0.onnx.gz
  # Expected: 10-20MB
  ```

- [ ] **Total Footprint (4 Models)**
  - [ ] All 4 models combined < 200MB
  - [ ] Deployable in typical clinical IT environment
  - [ ] Fits in container image size limits (if containerized)

---

## PHASE 3: INTEGRATION TESTING

### 3.1 Java Integration Testing

**Objective**: Verify ONNX model works with Java inference layer (ONNXModelContainer)

#### Checklist

- [ ] **Model Loading in Java**
  - [ ] ONNXModelContainer.java compiled successfully
  - [ ] Model file path configured correctly
  - [ ] Model loads without errors
  - [ ] OrtEnvironment initialized
  - [ ] OrtSession created

  ```java
  // Test code
  ModelConfig config = ModelConfig.builder()
      .modelName("sepsis")
      .modelVersion("1.0.0")
      .modelPath("/models/sepsis_v1.0.0.onnx")
      .inputDimensions(70)
      .outputDimensions(2)
      .build();

  ONNXModelContainer container = new ONNXModelContainer(config);

  // Verify model loaded
  assert container.isModelLoaded();
  System.out.println("✓ Model loaded successfully in Java");
  ```

- [ ] **Single Prediction Inference**
  - [ ] Test prediction with valid 70-feature input
  - [ ] Output is valid probability [0, 1]
  - [ ] Latency < 15ms
  - [ ] No exceptions thrown

  ```java
  float[] features = new float[70];
  // Fill with normalized values [0, 1]
  for (int i = 0; i < 70; i++) {
      features[i] = (float) Math.random();
  }

  MLPrediction prediction = container.predict(features);

  // Verify output
  assert prediction.getRiskScore() >= 0.0f && prediction.getRiskScore() <= 1.0f;
  assert prediction.getInferenceTimeMs() < 15;
  System.out.println(f"Risk score: {prediction.getRiskScore()}");
  System.out.println(f"Latency: {prediction.getInferenceTimeMs()}ms");
  ```

- [ ] **Batch Prediction Inference**
  - [ ] Test batch of 32 predictions
  - [ ] All predictions valid
  - [ ] Throughput > 100 predictions/second
  - [ ] No memory leaks

  ```java
  List<float[]> batch = new ArrayList<>();
  for (int i = 0; i < 32; i++) {
      float[] features = new float[70];
      for (int j = 0; j < 70; j++) {
          features[j] = (float) Math.random();
      }
      batch.add(features);
  }

  List<MLPrediction> predictions = container.predictBatch(batch);

  assert predictions.size() == 32;
  for (MLPrediction pred : predictions) {
      assert pred.getRiskScore() >= 0.0f && pred.getRiskScore() <= 1.0f;
  }
  ```

### 3.2 Feature Extraction Pipeline Testing

**Objective**: Verify 70-feature extraction produces correct input

#### Checklist

- [ ] **Feature Extraction Correctness**
  - [ ] All 70 features extracted
  - [ ] Feature order matches specification
  - [ ] Feature values within expected ranges
  - [ ] No NaN or Infinity values

  ```java
  // Test feature extraction
  PatientState patient = loadTestPatient();
  ClinicalFeatureVector features = ClinicalFeatureExtractor.extract(patient);

  float[] featureArray = features.toArray();
  assert featureArray.length == 70;

  for (int i = 0; i < 70; i++) {
      assert !Float.isNaN(featureArray[i]);
      assert !Float.isInfinite(featureArray[i]);
      assert featureArray[i] >= 0.0f && featureArray[i] <= 1.0f;
  }
  ```

- [ ] **Missing Data Handling**
  - [ ] Test with missing vitals (should impute median)
  - [ ] Test with missing labs (should impute median)
  - [ ] Test with missing medications (should impute 0)
  - [ ] No crashes on missing data

  ```java
  PatientState patientWithMissing = createPatientWithMissingData();
  ClinicalFeatureVector features = ClinicalFeatureExtractor.extract(patientWithMissing);

  // All features should be present (imputed if necessary)
  assert features.toArray().length == 70;
  ```

- [ ] **Normalization Consistency**
  - [ ] Features normalized to [0, 1]
  - [ ] Normalization parameters match training
  - [ ] Verification: sample data normalized correctly

  ```java
  // Verify normalization
  float[] features = features.toArray();

  for (int i = 0; i < 70; i++) {
      assert features[i] >= 0.0f;
      assert features[i] <= 1.0f;
  }
  ```

### 3.3 End-to-End Integration Test

**Objective**: Test complete pipeline from patient data to clinical decision

#### Checklist

- [ ] **Complete Workflow Test**
  - [ ] Load patient from EHR (test data)
  - [ ] Extract 70 features
  - [ ] Run inference with all 4 models
  - [ ] Generate risk scores
  - [ ] Output clinical decision (alert threshold)

  ```java
  // End-to-end test
  PatientData patient = ehrSystem.getPatient("TEST_PATIENT_001");

  // Extract features
  ClinicalFeatureVector features = featureExtractor.extract(patient);
  float[] featureArray = features.toArray();

  // Run inference for all 4 models
  MLPrediction sepsisRisk = sepsisModel.predict(featureArray);
  MLPrediction deteriorationRisk = deteriorationModel.predict(featureArray);
  MLPrediction mortalityRisk = mortalityModel.predict(featureArray);
  MLPrediction readmissionRisk = readmissionModel.predict(featureArray);

  // Generate alerts
  boolean sepsisAlert = sepsisRisk.getRiskScore() > 0.45;
  boolean deteriorationAlert = deteriorationRisk.getRiskScore() > 0.50;
  boolean mortalityAlert = mortalityRisk.getRiskScore() > 0.25;
  boolean readmissionAlert = readmissionRisk.getRiskScore() > 0.30;

  // Log results
  System.out.println("Patient: " + patient.getId());
  System.out.println("Sepsis Risk: " + sepsisRisk.getRiskScore());
  System.out.println("Sepsis Alert: " + sepsisAlert);
  ```

- [ ] **Performance Validation**
  - [ ] Complete inference < 100ms for all 4 models
  - [ ] Individual model latency < 15ms
  - [ ] No errors or exceptions
  - [ ] Output logged correctly

---

## PHASE 4: SECURITY & COMPLIANCE

### 4.1 Security Verification

**Objective**: Ensure model meets security requirements

#### Checklist

- [ ] **Model File Security**
  - [ ] ONNX file stored with appropriate file permissions (600)
  - [ ] Read access restricted to model serving process
  - [ ] Write access restricted to deployment team
  - [ ] Audit logging enabled for file access

  ```bash
  # Set appropriate permissions
  chmod 600 /models/sepsis_v1.0.0.onnx

  # Verify permissions
  ls -l /models/sepsis_v1.0.0.onnx
  # Expected: -rw------- 1 modelserver modelserver ...

  # Verify no execution bit
  test ! -x /models/sepsis_v1.0.0.onnx && echo "✓ Not executable"
  ```

- [ ] **Input Validation**
  - [ ] Model receives only normalized [0, 1] input
  - [ ] Invalid input doesn't crash model
  - [ ] Out-of-range values handled gracefully
  - [ ] NaN/Infinity values caught before inference

  ```java
  // Input validation test
  float[] invalidInput = new float[70];
  for (int i = 0; i < 70; i++) {
      invalidInput[i] = Float.NaN;
  }

  // Should not crash
  try {
      container.predict(invalidInput);
      System.out.println("✓ Handles NaN input gracefully");
  } catch (Exception e) {
      System.out.println("✓ Caught invalid input: " + e.getMessage());
  }
  ```

- [ ] **Output Validation**
  - [ ] Model output validated for correctness
  - [ ] Probability values in [0, 1]
  - [ ] Probabilities constrained by application logic
  - [ ] No PHI in model predictions

- [ ] **Audit Logging**
  - [ ] All model inferences logged with timestamp
  - [ ] Patient ID (de-identified if needed) logged
  - [ ] Risk score logged
  - [ ] Any errors logged
  - [ ] Logs retained per compliance requirements

### 4.2 Compliance Verification

**Objective**: Ensure model deployment complies with regulations

#### Checklist

- [ ] **HIPAA Compliance**
  - [ ] No Protected Health Information in model weights
  - [ ] Predictions don't contain PHI (just probabilities)
  - [ ] Audit logging configured
  - [ ] Access controls implemented
  - [ ] Encryption in transit enabled

  ```
  ✓ Model contains no patient-identifying information
  ✓ Predictions are de-identified probabilities
  ✓ Audit trail: timestamp, encrypted inference log
  ```

- [ ] **FDA Validation** (if applicable)
  - [ ] 21 CFR Part 11 readiness assessed
  - [ ] Change control procedure documented
  - [ ] Version control implemented
  - [ ] Traceability maintained (training data → model → deployment)

  ```
  ✓ Model versioning: v1.0.0 (semantic versioning)
  ✓ Training data documented: 30,000 records from 2023-2024
  ✓ Hyperparameters: documented in model card
  ✓ Performance: AUROC 0.862 on hold-out test set
  ```

- [ ] **Bias & Fairness Audit**
  - [ ] Performance validated by gender
  - [ ] Performance validated by age groups
  - [ ] Performance validated by race/ethnicity
  - [ ] AUROC disparity < 5% across groups documented

  ```
  Performance by Gender:
    Female (n=2500): AUROC 0.859
    Male (n=2500):   AUROC 0.865
    Disparity: 0.6% ✓

  Performance by Age Group:
    <60 (n=1667): AUROC 0.856
    60-75 (n=1667): AUROC 0.864
    >75 (n=1666): AUROC 0.860
    Disparity: 0.8% ✓
  ```

- [ ] **Documentation Package**
  - [ ] Model Card completed (model purpose, performance, limitations)
  - [ ] Training Data Sheet documented
  - [ ] Feature documentation (70-feature specification)
  - [ ] Limitations and failure modes documented

---

## PHASE 5: A/B TESTING SETUP

### 5.1 Canary Deployment Configuration

**Objective**: Set up safe, gradual rollout with continuous monitoring

#### Checklist

- [ ] **Traffic Routing Configuration**
  - [ ] A/B testing framework deployed
  - [ ] Initial traffic split: 10% to new model
  - [ ] 90% traffic continues to baseline
  - [ ] Traffic routing random (not stratified by patient type)
  - [ ] Can adjust split without downtime

  ```yaml
  # A/B Testing Configuration
  model_routing:
    baseline_model: sepsis_v0.9.5
    baseline_traffic: 90%
    new_model: sepsis_v1.0.0
    new_traffic: 10%

  rolling_deployment:
    phase_1: 10% (Day 1-7)
    phase_2: 25% (Day 8-14)
    phase_3: 50% (Day 15-21)
    phase_4: 100% (Day 22+)

  metric_monitoring:
    - auroc_degradation_threshold: -5%
    - inference_error_rate_threshold: 1%
    - latency_p99_threshold: 20ms
    auto_rollback: true
  ```

- [ ] **Monitoring Dashboard Setup**
  - [ ] Real-time metrics displayed
  - [ ] Comparison: baseline vs new model
  - [ ] Key metrics tracked:
    - AUROC (on recent outcomes)
    - Precision/Recall
    - Inference latency (p50, p95, p99)
    - Error rate
    - Prediction drift
  - [ ] Alert thresholds configured

  ```
  Dashboard: Sepsis Risk Model A/B Test

  Model Comparison:
  ┌─────────────────────┬──────────────┬──────────────┐
  │ Metric              │ Baseline     │ New Model    │
  ├─────────────────────┼──────────────┼──────────────┤
  │ AUROC (7-day)       │ 0.862 ±0.008 │ 0.859 ±0.009 │
  │ Precision @ 0.45    │ 0.71 ±0.03   │ 0.70 ±0.03   │
  │ Recall @ 0.45       │ 0.81 ±0.02   │ 0.82 ±0.02   │
  │ P99 Latency         │ 12.3ms       │ 11.8ms       │
  │ Error Rate          │ 0.02%        │ 0.01%        │
  └─────────────────────┴──────────────┴──────────────┘

  Status: ✓ No significant differences, proceeding to 25%
  ```

- [ ] **Automated Monitoring**
  - [ ] Drift detection active (PSI monitoring)
  - [ ] Performance degradation detection active
  - [ ] Error rate monitoring active
  - [ ] Latency monitoring active
  - [ ] Auto-alerts on thresholds exceeded

### 5.2 A/B Test Validation Criteria

**Objective**: Define objective criteria for phase progression

#### Checklist

- [ ] **Performance Non-Inferiority**
  - [ ] New model AUROC not significantly worse than baseline
  - [ ] Margin: -5% acceptable
  - [ ] Statistical test: calculate confidence interval
  - [ ] Duration: 7 days minimum per phase

  ```python
  # Non-inferiority test
  from scipy.stats import norm

  baseline_auroc = 0.862
  new_auroc = 0.859
  margin = 0.05

  # Calculate z-statistic for non-inferiority
  z = (new_auroc - baseline_auroc + margin) / sqrt(var)
  p_value = 1 - norm.cdf(z)

  if p_value < 0.05:
      print("✓ Non-inferiority demonstrated")
  else:
      print("✗ Non-inferiority test failed, remain at current split")
  ```

- [ ] **Stability Validation**
  - [ ] Metrics stable over 7-day window
  - [ ] No unexplained variance
  - [ ] Outlier rate < 1%
  - [ ] No concerning trends

  ```
  Metrics Stability Check (7-day rolling)
  ├─ AUROC: 0.859 ± 0.009 (trend: flat ✓)
  ├─ Precision: 0.70 ± 0.03 (trend: flat ✓)
  ├─ Recall: 0.82 ± 0.02 (trend: slightly increasing ✓)
  └─ Latency P99: 11.8 ± 0.2ms (trend: flat ✓)
  ```

- [ ] **No Safety Signals**
  - [ ] No increase in false negative rate (missed high-risk)
  - [ ] No increase in inference errors
  - [ ] No increase in response time
  - [ ] No patient safety incidents reported

  ```
  Safety Assessment:
  ├─ False Negative Rate: 0.19 vs 0.20 (no increase ✓)
  ├─ Inference Error Rate: 0.01% vs 0.02% (improvement ✓)
  ├─ Response Time P99: 11.8ms vs 12.3ms (improvement ✓)
  └─ Safety Incidents: 0 vs 0 (✓)
  ```

### 5.3 Progressive Rollout Plan

**Objective**: Document phase progression schedule

#### Checklist

- [ ] **Phase 1: Canary (10% traffic, 1 week)**
  - [ ] Start date: [YYYY-MM-DD]
  - [ ] Expected end date: [YYYY-MM-DD]
  - [ ] Success criteria defined (see above)
  - [ ] Monitoring active
  - [ ] Rollback plan ready

  ```
  Phase 1: Canary (10% traffic)
  Start: 2025-11-08 @ 14:00 UTC
  Expected End: 2025-11-14 @ 14:00 UTC
  Status: ✓ MONITORING ACTIVE

  Monitoring Metrics:
  ├─ AUROC comparison
  ├─ Latency monitoring
  ├─ Error rate tracking
  └─ Drift detection
  ```

- [ ] **Phase 2: Graduated (25% traffic, 1 week)**
  - [ ] Prerequisites met (Phase 1 criteria)
  - [ ] Start date: [YYYY-MM-DD]
  - [ ] Expanded monitoring
  - [ ] Clinical team notified

  ```
  Phase 2: Graduated (25% traffic)
  Start: 2025-11-15 @ 14:00 UTC
  End: 2025-11-21 @ 14:00 UTC
  Prerequisites:
    ✓ Phase 1 success criteria met
    ✓ No safety signals
    ✓ Performance acceptable
  Monitoring: EXPANDED
  ```

- [ ] **Phase 3: Majority (50% traffic, 1 week)**
  - [ ] Prerequisites met (Phase 2 criteria)
  - [ ] High clinical visibility
  - [ ] Ready for full rollout

- [ ] **Phase 4: Production (100% traffic)**
  - [ ] Prerequisites met (Phase 3 criteria)
  - [ ] Baseline model archived
  - [ ] Rollback plan retained for 90 days

---

## PHASE 6: MONITORING CONFIGURATION

### 6.1 Real-Time Metrics Collection

**Objective**: Set up production monitoring for ongoing performance assessment

#### Checklist

- [ ] **Metrics Infrastructure**
  - [ ] Time-series database ready (Prometheus, InfluxDB, etc.)
  - [ ] Metrics export configured
  - [ ] Retention policy defined (minimum 1 year)
  - [ ] Backup configured

  ```
  Metrics Retention Policy:
  ├─ Raw metrics: 30 days
  ├─ Hourly aggregates: 1 year
  ├─ Daily aggregates: 5 years
  └─ Backups: Daily, retained 90 days
  ```

- [ ] **Prediction Logging**
  - [ ] All predictions logged (timestamp, patient_id, risk_score, model_version)
  - [ ] Log location: [database/path]
  - [ ] Log format: JSON/Parquet
  - [ ] Logs encrypted at rest
  - [ ] Access audited

  ```python
  # Prediction log structure
  {
      "timestamp": "2025-11-08T14:23:45Z",
      "patient_id_hash": "abc123def456",
      "model_name": "sepsis",
      "model_version": "1.0.0",
      "risk_score": 0.68,
      "classification": "HIGH_RISK",
      "latency_ms": 12.3,
      "inference_success": true
  }
  ```

- [ ] **Outcome Tracking**
  - [ ] Actual outcomes recorded (sepsis onset yes/no within 48h)
  - [ ] Outcome timing synchronized with predictions
  - [ ] Matching logic: patient_id, time window
  - [ ] Audit trail for outcome data

### 6.2 Drift Detection

**Objective**: Identify when model performance degrades due to data drift

#### Checklist

- [ ] **Population Stability Index (PSI) Monitoring**
  - [ ] Calculate weekly PSI for prediction distribution
  - [ ] Alert if PSI > 0.25 (significant drift)
  - [ ] Manual investigation triggered
  - [ ] Threshold for retraining: PSI > 0.25

  ```python
  # PSI calculation (weekly)
  def calculate_psi(current_predictions, baseline_predictions, n_bins=10):
      baseline_hist, bin_edges = np.histogram(baseline_predictions, bins=n_bins)
      current_hist, _ = np.histogram(current_predictions, bins=bin_edges)

      baseline_prop = (baseline_hist + 1) / (baseline_hist.sum() + n_bins)
      current_prop = (current_hist + 1) / (current_hist.sum() + n_bins)

      psi = np.sum((current_prop - baseline_prop) * np.log(current_prop / baseline_prop))
      return psi

  # Weekly monitoring
  psi = calculate_psi(
      predictions_this_week,
      predictions_training_period
  )

  if psi > 0.25:
      alert("DRIFT DETECTED", f"PSI={psi:.3f}, consider retraining")
  ```

- [ ] **Feature Distribution Monitoring**
  - [ ] Key features tracked for distribution changes
  - [ ] Top 5 features by importance monitored
  - [ ] Alert if distribution significantly changes
  - [ ] KL divergence > threshold triggers alert

  ```
  Feature Drift Monitoring (Weekly):

  Feature                Current Median    Training Median    Status
  ─────────────────────────────────────────────────────────────────
  lab_lactate_mmol       2.1 mmol/L        2.0 mmol/L        ✓ Normal
  vital_temperature_c    37.2°C            37.1°C            ✓ Normal
  score_qsofa            1.2               1.1               ⚠ Watch
  lab_wbc_k_ul           9.8 K/µL          9.5 K/µL          ✓ Normal
  score_sofa             5.3               5.1               ✓ Normal
  ```

### 6.3 Performance Monitoring

**Objective**: Track model performance against validation baseline

#### Checklist

- [ ] **AUROC Monitoring** (requires outcome labels)
  - [ ] Calculate weekly AUROC on recent outcomes
  - [ ] Compare to baseline (0.862 for sepsis model)
  - [ ] Alert if drop > 5% (critical)
  - [ ] Alert if drop > 3% (warning)

  ```
  Performance Trending (Weekly):

  Week    AUROC    Baseline    Diff      Status
  ──────────────────────────────────────────────
  2025-42 0.859    0.862      -0.3%     ✓ Nominal
  2025-43 0.858    0.862      -0.4%     ✓ Nominal
  2025-44 0.851    0.862      -1.3%     ✓ Nominal
  2025-45 0.842    0.862      -2.3%     ⚠ Warning (>3% drop)

  Action: Investigate recent data changes, consider retraining
  ```

- [ ] **Calibration Monitoring**
  - [ ] Brier score calculated weekly
  - [ ] Expected Calibration Error (ECE) calculated
  - [ ] Alert if Brier score > 0.25 (poor calibration)
  - [ ] Reliability diagrams generated monthly

- [ ] **Latency Monitoring**
  - [ ] P99 latency tracked daily
  - [ ] Alert if P99 > 20ms (warning)
  - [ ] Alert if P99 > 25ms (critical)
  - [ ] Throughput tracked (predictions/second)

### 6.4 Alerting Rules

**Objective**: Define automated alert conditions

#### Checklist

- [ ] **Critical Alerts** (immediate escalation)

  ```
  Alert: MODEL_INFERENCE_ERROR
  Condition: Error rate > 1% over 1-hour window
  Action: Page on-call ML engineer, investigate, consider rollback

  Alert: PERFORMANCE_CRITICAL
  Condition: AUROC drop > 10% or precision drop > 20%
  Action: Immediate investigation, likely rollback

  Alert: LATENCY_CRITICAL
  Condition: P99 latency > 25ms
  Action: Investigate resource constraints, scale if needed
  ```

- [ ] **Warning Alerts** (daily review)

  ```
  Alert: DRIFT_DETECTED
  Condition: PSI > 0.25
  Action: Schedule retraining discussion, monitor closely

  Alert: PERFORMANCE_WARNING
  Condition: AUROC drop 3-5% or precision drop 5-10%
  Action: Daily monitoring, plan retraining if continues

  Alert: LATENCY_WARNING
  Condition: P99 latency 15-25ms
  Action: Monitor, investigate resource usage
  ```

- [ ] **Info Alerts** (logged, weekly review)

  ```
  Alert: METRICS_STABLE
  Condition: All metrics within expected range (daily)
  Action: Log, no action required

  Alert: WEEKLY_PERFORMANCE_REPORT
  Condition: Generated every Monday
  Action: Review, document in monitoring log
  ```

---

## PHASE 7: ROLLBACK PLANNING

### 7.1 Rollback Decision Criteria

**Objective**: Define conditions triggering immediate rollback

#### Checklist

- [ ] **Automatic Rollback Triggers**
  - [ ] AUROC drops > 10% (automatic rollback)
  - [ ] Error rate > 2% (automatic rollback)
  - [ ] Latency P99 > 30ms (automatic rollback)
  - [ ] System executes rollback automatically

  ```
  Automatic Rollback Triggers:

  if auroc_drop > 10% OR error_rate > 2% OR latency_p99 > 30ms:
      # Immediate action
      traffic_split = {baseline: 100%, new_model: 0%}
      send_alert("CRITICAL: Automatic rollback executed")
      log_incident()
      notify_on_call_engineer()
  ```

- [ ] **Manual Rollback Triggers**
  - [ ] AUROC drops > 5% for >3 days
  - [ ] Calibration significantly degrades
  - [ ] Clinical team requests rollback
  - [ ] Safety concerns raised
  - [ ] Infrastructure issues prevent proper monitoring

  ```
  Manual Rollback Decision Process:

  1. Detect concerning metric (e.g., AUROC -5%)
  2. Alert on-call engineer and clinical contact
  3. Investigate root cause (data drift? model issue? infrastructure?)
  4. If no clear cause or concerning trend → initiate rollback
  5. Execute rollback (see procedure below)
  6. Post-incident review within 24 hours
  ```

### 7.2 Rollback Execution Procedure

**Objective**: Document step-by-step rollback process

#### Checklist

- [ ] **Pre-Rollback Verification**
  - [ ] Baseline model health confirmed (load test passed)
  - [ ] Routing configuration accessible
  - [ ] Rollback procedure communicated to team
  - [ ] Incident tracking created

  ```bash
  # Pre-rollback checklist

  # 1. Verify baseline model is healthy
  curl http://sepsis-model-baseline:8080/health
  # Expected: {"status": "healthy", "auroc": 0.862}

  # 2. Verify routing configuration
  kubectl get configmap model-routing
  # Check baseline model is configured and ready

  # 3. Create incident ticket
  incident_id=$(create_incident.sh "Sepsis Model Rollback")
  echo $incident_id  # e.g., INC-2025-1847
  ```

- [ ] **Execute Rollback**
  - [ ] Set traffic to baseline: 100%
  - [ ] Set traffic to new model: 0%
  - [ ] Verify routing change applied
  - [ ] Monitor metrics for recovery

  ```bash
  # Rollback execution

  kubectl patch configmap model-routing -p \
    '{
      "data": {
        "sepsis_baseline": "100",
        "sepsis_new": "0"
      }
    }'

  # Verify change
  kubectl get configmap model-routing -o jsonpath='{.data}'

  # Monitor for traffic change
  watch -n 1 'curl http://monitoring:9090/graph?q=model_traffic'
  ```

- [ ] **Post-Rollback Verification**
  - [ ] All traffic routed to baseline confirmed
  - [ ] Metrics return to baseline levels
  - [ ] No increase in errors
  - [ ] Latency back to normal
  - [ ] Clinical team notified

  ```
  Post-Rollback Verification:

  Time    Baseline Traffic    New Model Traffic    Status
  ─────────────────────────────────────────────────────────
  14:23   45%                 55%                  (Pre-rollback)
  14:24   92%                 8%                   (Transitioning)
  14:25   99%                 1%                   (Transitioning)
  14:26   100%                0%                   ✓ Rollback complete

  Metrics:
  ├─ AUROC: 0.859 → 0.862 ✓ (recovering)
  ├─ Error Rate: 2.1% → 0.05% ✓ (improving)
  └─ P99 Latency: 28ms → 12ms ✓ (improving)
  ```

### 7.3 Post-Rollback Analysis

**Objective**: Understand root cause and prevent recurrence

#### Checklist

- [ ] **Root Cause Analysis**
  - [ ] Timeline documented (when did metric first degrade?)
  - [ ] Data investigated (any distribution changes?)
  - [ ] Model investigated (any weight issues?)
  - [ ] Infrastructure investigated (resource constraints?)
  - [ ] Root cause identified and documented

  ```
  Root Cause Analysis: Sepsis Model Rollback (2025-11-15)

  Timeline:
  2025-11-15 14:00  New model deployed (10% traffic)
  2025-11-17 08:30  AUROC drop detected (-3.2%)
  2025-11-17 09:15  AUROC drop critical (-8.1%)
  2025-11-17 09:18  Automatic rollback executed

  Investigation:
  ├─ Data drift check: PSI = 0.28 (SIGNIFICANT DRIFT)
  ├─ Root cause: Hospital A admission policy changed
  │   - More palliative care admissions (lower sepsis rate)
  │   - Training data: 7% sepsis prevalence
  │   - Current data: 3% sepsis prevalence
  └─ Impact: Model miscalibrated for new patient population

  Lessons Learned:
  1. Need cohort-specific drift detection
  2. Consider retraining on recent data before production
  3. Implement stratified monitoring by hospital site
  ```

- [ ] **Corrective Actions**
  - [ ] Root cause addressed
  - [ ] Model retraining plan created
  - [ ] Drift detection improved
  - [ ] Deployment procedure updated (if needed)
  - [ ] Team training updated

- [ ] **Communication & Documentation**
  - [ ] Incident report completed
  - [ ] Lessons learned shared with team
  - [ ] Clinical team briefed on what happened
  - [ ] Prevention steps communicated
  - [ ] Documentation updated

---

## PHASE 8: SIGN-OFF & DEPLOYMENT

### 8.1 Sign-Off Requirements

**Objective**: Obtain required approvals before production deployment

#### Checklist

- [ ] **ML Engineering Sign-Off**
  - [ ] All technical requirements met
  - [ ] Performance targets achieved
  - [ ] Testing completed
  - [ ] Documentation complete
  - [ ] ML Lead approves deployment

  ```
  ML Engineering Sign-Off Form

  Reviewed by: Dr. Jane Smith (ML Lead)
  Date: 2025-11-01

  ✓ Technical requirements: MET
    ├─ Input/output schema validated
    ├─ Numerical equivalence verified
    ├─ Latency <15ms p99: 12.3ms
    ├─ Memory footprint <200MB: 145MB
    └─ Inference throughput >100/sec: 220/sec

  ✓ Performance targets: MET
    ├─ AUROC ≥0.85: 0.862
    ├─ Sensitivity ≥0.80: 0.810
    ├─ Specificity ≥0.80: 0.802
    └─ Brier score <0.20: 0.158

  ✓ Testing: COMPLETE
    ├─ Unit tests: 47/47 passed
    ├─ Integration tests: 23/23 passed
    └─ End-to-end tests: 15/15 passed

  ✓ Documentation: COMPLETE
    ├─ Model card: Approved
    ├─ Feature specification: Reviewed
    ├─ Training procedure: Documented
    └─ Deployment guide: Ready

  Approval: SIGNED
  Signature: ___________________
  ```

- [ ] **Clinical Affairs Sign-Off**
  - [ ] Clinical validation completed
  - [ ] Fairness audit approved
  - [ ] Clinical safety assessment passed
  - [ ] Clinical Affairs Director approves

  ```
  Clinical Affairs Sign-Off Form

  Reviewed by: Dr. Robert Johnson (Clinical Affairs)
  Date: 2025-11-01

  ✓ Clinical Validation: APPROVED
    ├─ Feature importance: Clinically sound
    ├─ Predictions: Align with clinical expectations
    ├─ Thresholds: Appropriate for clinical use
    └─ False negative rate: Acceptable (<20%)

  ✓ Fairness Audit: APPROVED
    ├─ Gender disparity: <5% ✓
    ├─ Age disparity: <5% ✓
    ├─ Race disparity: <5% ✓
    └─ No discriminatory patterns detected

  ✓ Safety Assessment: APPROVED
    ├─ Risk-benefit analysis: Favorable
    ├─ Mitigation strategies: In place
    ├─ Alert thresholds: Clinically appropriate
    └─ Workflow integration: Acceptable

  Approval: SIGNED
  Signature: ___________________
  ```

- [ ] **IT Security Sign-Off**
  - [ ] Security assessment completed
  - [ ] HIPAA compliance verified
  - [ ] Access controls validated
  - [ ] Audit logging enabled
  - [ ] Security Officer approves

  ```
  IT Security Sign-Off Form

  Reviewed by: Sarah Lee (Security Officer)
  Date: 2025-11-01

  ✓ Security Assessment: APPROVED
    ├─ Input validation: Implemented
    ├─ File permissions: 600 (restricted)
    ├─ Encryption at rest: Enabled
    ├─ Encryption in transit: TLS 1.2+
    └─ Access audit logging: Configured

  ✓ HIPAA Compliance: VERIFIED
    ├─ No PHI in model weights
    ├─ Predictions contain no identifiers
    ├─ Audit trail: Complete
    └─ Data retention: <365 days

  ✓ Infrastructure Security: APPROVED
    ├─ Container image: Scanned (no vulnerabilities)
    ├─ Dependencies: Up-to-date
    ├─ Network policies: Restrictive
    └─ Monitoring: Active

  Approval: SIGNED
  Signature: ___________________
  ```

### 8.2 Deployment Execution

**Objective**: Execute production deployment safely

#### Checklist

- [ ] **Pre-Deployment Steps**
  - [ ] All sign-offs obtained
  - [ ] Deployment window scheduled
  - [ ] Team briefed
  - [ ] Incident management setup
  - [ ] Monitoring prepared

  ```
  Deployment Window: 2025-11-08, 14:00-18:00 UTC

  Participants:
  ├─ ML Engineer (on-call): jane.smith@cardiofit.com
  ├─ DevOps Lead (on-call): alex.chen@cardiofit.com
  ├─ Clinical Contact (on-call): dr.johnson@cardiofit.com
  └─ On-Call Manager: incident@cardiofit.com

  Communication Channels:
  ├─ Slack: #sepsis-model-deployment
  ├─ Conference bridge: (details in incident ticket)
  └─ Escalation: +1-555-0147 (security on-call)
  ```

- [ ] **Phase 1 Deployment (10% Traffic)**
  - [ ] Traffic split configured (90% baseline, 10% new)
  - [ ] New model deployed
  - [ ] Routing verified
  - [ ] Monitoring active
  - [ ] Team observing for 1 week

  ```bash
  # Deployment commands

  # 1. Deploy model to production cluster
  kubectl apply -f deployment-sepsis-v1.0.0.yaml

  # 2. Verify deployment
  kubectl get deployment sepsis-v1.0.0
  kubectl get pods -l app=sepsis-v1.0.0

  # 3. Configure traffic routing (10%)
  kubectl patch configmap model-routing -p \
    '{"data": {"sepsis_baseline": "90", "sepsis_new": "10"}}'

  # 4. Verify routing
  kubectl logs -l app=routing-controller | grep sepsis

  # 5. Monitor metrics
  watch -n 5 'curl http://monitoring:9090/api/v1/query?query=model_accuracy'
  ```

- [ ] **Phase 2+ Deployment (25%, 50%, 100%)**
  - [ ] Success criteria met for previous phase
  - [ ] Traffic increased to next level
  - [ ] Monitoring continues
  - [ ] Rollback plan always ready

### 8.3 Post-Deployment Verification

**Objective**: Confirm successful deployment

#### Checklist

- [ ] **Immediate Verification** (within 1 hour)
  - [ ] Model serving without errors
  - [ ] Predictions generated successfully
  - [ ] Inference latency acceptable
  - [ ] No increase in error rate

  ```
  Post-Deployment Verification (2025-11-08 15:00 UTC):

  Service Health:
  ├─ Model serving: ✓ HEALTHY
  ├─ Request latency: 12.2ms (target <15ms) ✓
  ├─ Error rate: 0.01% (threshold 1.0%) ✓
  └─ Prediction volume: 450/sec (normal) ✓

  Prediction Quality:
  ├─ Risk score range: [0.0, 1.0] ✓
  ├─ Probability sum: 1.0 ±0.0001 ✓
  └─ Sample predictions logged ✓
  ```

- [ ] **24-Hour Monitoring**
  - [ ] No concerning trends
  - [ ] Performance stable
  - [ ] No alert escalations
  - [ ] Team satisfied with deployment

  ```
  24-Hour Post-Deployment Report (2025-11-09 14:00 UTC):

  Period: 2025-11-08 14:00 - 2025-11-09 14:00

  Traffic Distribution:
  ├─ Baseline model: 90%
  ├─ New model: 10%
  └─ Total predictions: 38.8M

  Performance (Baseline vs New):
  ├─ AUROC: 0.862 vs 0.859 (✓ non-inferior)
  ├─ Latency P99: 12.3ms vs 12.0ms (✓ improved)
  ├─ Error rate: 0.02% vs 0.01% (✓ improved)
  └─ Throughput: 450/sec vs 450/sec (✓ equal)

  Clinical Validation:
  ├─ No patient safety incidents reported ✓
  ├─ Clinical team satisfied ✓
  └─ No alerts triggered ✓

  Recommendation: PROCEED TO PHASE 2 (25% traffic)
  ```

---

## POST-DEPLOYMENT VERIFICATION

### Ongoing Monitoring Schedule

**Objective**: Maintain production monitoring long-term

#### Checklist

- [ ] **Daily Monitoring** (automated)
  - [ ] AUROC calculated (with recent outcomes)
  - [ ] Drift metrics calculated
  - [ ] Error rate monitored
  - [ ] Latency monitored
  - [ ] Alerts sent if thresholds exceeded

- [ ] **Weekly Review** (manual)
  - [ ] Performance report generated
  - [ ] Metrics reviewed by ML team
  - [ ] Clinical feedback gathered
  - [ ] Issues logged and prioritized
  - [ ] Retraining needs assessed

- [ ] **Monthly Review** (comprehensive)
  - [ ] Full performance evaluation
  - [ ] Fairness metrics recalculated
  - [ ] Calibration assessment
  - [ ] Drift analysis detailed
  - [ ] Retraining decision made

- [ ] **Quarterly Review** (strategic)
  - [ ] Model refresh baseline (schedule retraining)
  - [ ] Architecture improvements planned
  - [ ] Feature engineering opportunities assessed
  - [ ] Clinical feedback incorporated
  - [ ] Annual deployment plan updated

### Maintenance & Retraining

- [ ] **Retraining Triggers**
  - [ ] Quarterly baseline retraining (every 90 days)
  - [ ] Drift-triggered retraining (PSI > 0.25)
  - [ ] Performance-triggered retraining (AUROC drop > 5%)
  - [ ] Ad-hoc retraining based on clinical feedback

- [ ] **Model Versioning**
  - [ ] New version number assigned (e.g., v1.1.0)
  - [ ] Training completed with new data
  - [ ] Validation on hold-out test set
  - [ ] Comparison to current production model
  - [ ] If improved: schedule deployment
  - [ ] If not improved: investigate root cause

---

## APPENDIX: CHECKLIST SUMMARY

### Quick Reference

**Pre-Deployment** (1 week)
- [ ] Model artifact validation
- [ ] Input/output schema verification
- [ ] Numerical equivalence testing
- [ ] Data consistency checks
- [ ] Latency benchmarking (<15ms p99)
- [ ] Memory testing (<200MB)
- [ ] Security review
- [ ] HIPAA compliance verification
- [ ] Sign-offs obtained

**Deployment** (2 weeks)
- [ ] Phase 1: 10% traffic (1 week)
- [ ] Phase 2: 25% traffic (1 week)
- [ ] Phase 3: 50% traffic (optional)
- [ ] Phase 4: 100% traffic (if needed)

**Post-Deployment** (ongoing)
- [ ] Daily monitoring
- [ ] Weekly reviews
- [ ] Monthly evaluations
- [ ] Quarterly retraining assessments

---

**Document End**

For deployment questions: DevOps Team (devops@cardiofit.com)
For clinical questions: Clinical Affairs (clinical@cardiofit.com)
For technical issues: ML Engineering (mleng@cardiofit.com)
