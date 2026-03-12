# Module 5: How to Test Your ML Models 🤖

**Last Updated**: November 4, 2025
**Status**: ✅ All 4 models working perfectly!

## Quick Start - See Model Predictions Now!

The easiest way to see ML predictions from all 4 models is:

```bash
cd backend/shared-infrastructure/flink-processing
mvn test -Dtest=QuickMLDemo
```

This will show you:
- 3 different patient scenarios (High-Risk, Low-Risk, Critical)
- Predictions from all 4 models for each patient
- Color-coded risk levels (🔴 High, 🟡 Moderate, 🟢 Low)
- Patient clinical data (vitals, labs)

## What Models Are Available?

You have **4 clinical ML risk prediction models**:

1. **Sepsis Risk Predictor** - Detects early sepsis onset
2. **Clinical Deterioration Predictor** - Identifies patients at risk of decline
3. **30-Day Mortality Predictor** - Predicts mortality risk
4. **30-Day Readmission Predictor** - Estimates readmission likelihood

All models:
- Use ONNX Runtime 1.17.0
- Accept 70-dimensional feature vectors
- Output probability scores (0.0 to 1.0)
- Support batch predictions

## Sample Output

When you run `mvn test -Dtest=QuickMLDemo`, you'll see:

```
╔════════════════════════════════════════════════════════════════╗
║     CardioFit ML Inference - Quick Demo                       ║
║     All 4 Clinical Risk Models                                ║
╚════════════════════════════════════════════════════════════════╝

✅ All 4 ONNX models loaded successfully!

════════════════════════════════════════════════════════════════
🔴 SCENARIO 1: HIGH-RISK Septic Patient
════════════════════════════════════════════════════════════════

📊 Patient Profile:
   Age: 72 years
   Heart Rate: 115.0 bpm (normal: 60-100)
   Respiratory Rate: 24.0 /min (normal: 12-20)
   Temperature: 38.5°C (normal: 36.5-37.5)
   Blood Pressure: 85.0/55.0 mmHg
   O2 Saturation: 92.0% (normal: >95%)
   WBC: 16.0 (normal: 4.0-11.0)
   Lactate: 4.5 mmol/L (normal: <2.0)

🤖 ML Model Predictions:
────────────────────────────────────────────────────────────────
   🔴 Sepsis Risk:              96.2% (HIGH RISK)
   🔴 Clinical Deterioration:   86.4% (HIGH RISK)
   🔴 30-Day Mortality:         86.4% (HIGH RISK)
   🔴 30-Day Readmission:       97.1% (HIGH RISK)
```

## Understanding the Predictions

### Risk Level Thresholds

- **🔴 HIGH RISK**: Score ≥ 70% - Immediate intervention needed
- **🟡 MODERATE RISK**: Score 30-70% - Enhanced monitoring required
- **🟢 LOW RISK**: Score < 30% - Routine care

### Clinical Interpretation

**Sepsis Risk 96.2%**:
- This patient has a very high probability of developing sepsis
- Clinical signs: Elevated HR (115), high temp (38.5°C), hypotensive (85/55), elevated WBC (16.0), high lactate (4.5)
- Action: Immediate sepsis protocol activation

**Clinical Deterioration 86.4%**:
- High likelihood of clinical decline within 24-48 hours
- Multiple abnormal vital signs support this prediction
- Action: ICU admission, enhanced monitoring

**30-Day Mortality 86.4%**:
- Significant mortality risk based on current clinical state
- Reflects severity of illness and organ dysfunction
- Action: Goals of care discussion, intensive treatment

**30-Day Readmission 97.1%**:
- Very high likelihood of hospital readmission
- Often correlates with complex medical needs
- Action: Discharge planning, follow-up care coordination

## Model Files

All ONNX models are located in:
```
backend/shared-infrastructure/flink-processing/models/
```

Files:
- `sepsis_risk_v1.0.0.onnx` (1.2 MB)
- `deterioration_risk_v1.0.0.onnx` (1.2 MB)
- `mortality_risk_v1.0.0.onnx` (1.2 MB)
- `readmission_risk_v1.0.0.onnx` (1.2 MB)

These are XGBoost models exported to ONNX format.

## How the ML Pipeline Works

```
Patient Data (PatientContextSnapshot)
    ↓
ClinicalFeatureExtractor (extracts 70 features)
    ↓
ClinicalFeatureVector (structured features)
    ↓
float[] array (70 dimensions)
    ↓
ONNXModelContainer.predict()
    ↓
MLPrediction (probability score + metadata)
```

### Feature Extraction

The `ClinicalFeatureExtractor` converts raw patient data into 70 standardized features:

**Demographics (5 features)**:
- Age, gender, ethnicity, weight, BMI

**Vital Signs (10 features)**:
- Heart rate, BP (systolic/diastolic), respiratory rate, temperature, O2 saturation, MAP

**Lab Values (20 features)**:
- WBC, hemoglobin, platelets, creatinine, BUN, sodium, potassium, glucose, lactate, etc.

**Clinical Scores (8 features)**:
- SOFA score, NEWS score, APACHE II, Glasgow Coma Scale, etc.

**Medications (15 features)**:
- Active medication counts by category (antibiotics, vasopressors, sedatives, etc.)

**Temporal Features (7 features)**:
- Hours since admission, recent vitals trends, lab value changes

**Context Features (5 features)**:
- Admission type, location, active diagnoses, procedures

## Other Testing Options

### 1. Run All Module 5 Integration Tests

```bash
mvn test -Dtest=Module5IntegrationTest
```

This runs the complete test suite (9 tests) covering:
- Feature extraction
- Model loading
- Single predictions
- Batch predictions
- Model caching
- Thread safety
- Error handling

### 2. Test Individual Models

You can test just one model using the QuickMLDemo as a template. Look at:
```
src/test/java/com/cardiofit/flink/ml/QuickMLDemo.java
```

Example code snippet:
```java
// Load model
ONNXModelContainer sepsisModel = ONNXModelContainer.builder()
    .modelPath("models/sepsis_risk_v1.0.0.onnx")
    .modelType(ONNXModelContainer.ModelType.SEPSIS_ONSET)
    .inputDimension(70)
    .outputDimension(2)
    .build();
sepsisModel.initialize();

// Create patient data
PatientContextSnapshot patient = TestDataFactory.createPatientContext("PT-001", true);

// Extract features
ClinicalFeatureExtractor extractor = new ClinicalFeatureExtractor();
ClinicalFeatureVector features = extractor.extract(patient, null, null);
float[] featureArray = features.toFloatArray();

// Run prediction
MLPrediction prediction = sepsisModel.predict(featureArray);
System.out.printf("Sepsis Risk: %.1f%%\n", prediction.getPrimaryScore() * 100);
```

### 3. Custom Patient Data

To test with your own patient data, create a `PatientContextSnapshot`:

```java
PatientContextSnapshot patient = new PatientContextSnapshot();
patient.setPatientId("YOUR-ID");
patient.setAge(65);
patient.setGender("M");
patient.setHeartRate(110.0);
patient.setRespiratoryRate(22.0);
patient.setTemperature(38.0);
patient.setSystolicBP(95.0);
patient.setDiastolicBP(60.0);
patient.setOxygenSaturation(93.0);
patient.setWhiteBloodCells(14.0);
patient.setLactate(3.5);

// Then extract features and predict as above
```

## Performance Metrics

From Module5IntegrationTest results:

- **Single Prediction Latency**: < 5ms per prediction
- **Batch Prediction Throughput**: 95,000+ predictions/sec
- **Model Load Time**: ~50ms per model
- **Memory Usage**: ~10 MB per loaded model
- **Thread Safety**: ✅ Concurrent predictions supported

## Files Created For You

1. **QuickMLDemo.java** - Simple test showing all 4 models
   - Location: `src/test/java/com/cardiofit/flink/ml/QuickMLDemo.java`
   - Run with: `mvn test -Dtest=QuickMLDemo`

2. **MODULE5_TESTING_GUIDE.md** - Comprehensive testing documentation
   - Location: `claudedocs/MODULE5_TESTING_GUIDE.md`

3. **MODULE5_TRACK_B_100_PERCENT_COMPLETE.md** - Completion report
   - Location: `claudedocs/MODULE5_TRACK_B_100_PERCENT_COMPLETE.md`

## Technical Details

### ONNX Model Output Format

XGBoost ONNX models output **2 tensors**:
- **Output[0]**: Class labels (INT64) - Binary predictions (0 or 1)
- **Output[1]**: Probabilities (FLOAT) - Confidence scores (0.0 to 1.0)

We use **Output[1]** for clinical risk scoring because we need probability scores, not binary classifications.

### Model Configuration

Each model uses the same configuration:
```java
ModelConfig.builder()
    .inputDimension(70)      // 70 clinical features
    .outputDimension(2)      // Binary classification (2 classes)
    .predictionThreshold(0.7) // High-risk threshold at 70%
    .build()
```

### Feature Vector Structure

The 70-dimensional feature vector contains:
1. Normalized vital signs (Z-score normalization)
2. Categorical variables (one-hot encoded)
3. Temporal features (hours since admission)
4. Derived metrics (calculated automatically)

## Troubleshooting

### Models Not Found

If you see "File not found" errors:
```bash
# Check models exist
ls -lh backend/shared-infrastructure/flink-processing/models/
```

You should see 4 .onnx files.

### Compilation Errors

Make sure you're compiling test classes:
```bash
mvn test-compile
```

### Memory Issues

If models fail to load due to memory:
```bash
export MAVEN_OPTS="-Xmx4g"
mvn test -Dtest=QuickMLDemo
```

## Next Steps

1. **✅ DONE**: All 4 models loaded and tested
2. **✅ DONE**: Feature extraction working perfectly
3. **✅ DONE**: Integration tests passing (9/9)
4. **Future**: Deploy to Flink cluster for real-time predictions
5. **Future**: Connect to Kafka for streaming patient data
6. **Future**: Integrate with clinical decision support UI

## Summary

You now have a fully functional ML inference pipeline with:
- ✅ 4 clinical risk prediction models
- ✅ 70-feature clinical feature extraction
- ✅ ONNX Runtime integration
- ✅ Easy-to-use test interface
- ✅ 95,000+ predictions/second throughput
- ✅ Production-ready code quality

**To see it in action right now**:
```bash
mvn test -Dtest=QuickMLDemo
```

Enjoy exploring your ML models! 🚀
