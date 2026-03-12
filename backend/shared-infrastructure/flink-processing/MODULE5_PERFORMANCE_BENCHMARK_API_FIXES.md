# Module 5 Performance Benchmark API Fixes - Complete

**File**: `/Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/ml/Module5PerformanceBenchmark.java`

**Status**: ✅ ALL COMPILATION ERRORS FIXED - BUILD SUCCESS

---

## Summary of Changes

Applied the same API fix patterns from `Module5IntegrationTest.java` to `Module5PerformanceBenchmark.java` to ensure consistency across the test suite.

### 1. Import Updates

**Added missing imports**:
```java
import com.cardiofit.flink.ml.ModelConfig;
import com.cardiofit.flink.ml.util.TestDataFactory;
import com.cardiofit.flink.ml.features.ClinicalFeatureVector;
import java.time.Instant;
```

**Fixed package location**:
```java
// BEFORE
import com.cardiofit.flink.ml.ClinicalFeatureVector;

// AFTER
import com.cardiofit.flink.ml.features.ClinicalFeatureVector;
```

---

## 2. ONNXModelContainer Builder Pattern (Lines 70-145)

Fixed all 4 model initializations to use proper builder pattern with ModelConfig.

### Pattern Applied (4 models: sepsis, deterioration, mortality, readmission):

```java
// BEFORE (Constructor - WRONG)
sepsisModel = new ONNXModelContainer(
    MODELS_DIR + "/sepsis_risk_v1.0.0.onnx",
    "sepsis_risk"
);

// AFTER (Builder pattern - CORRECT)
ModelConfig sepsisConfig = ModelConfig.builder()
    .modelPath(MODELS_DIR + "/sepsis_risk_v1.0.0.onnx")
    .inputDimension(70)
    .outputDimension(2)
    .predictionThreshold(0.7)
    .build();

sepsisModel = ONNXModelContainer.builder()
    .modelId("sepsis_risk_v1")
    .modelName("Sepsis Risk Predictor")
    .modelType(ONNXModelContainer.ModelType.SEPSIS_ONSET)
    .modelVersion("1.0.0")
    .inputFeatureNames(TestDataFactory.createFeatureNames())
    .outputNames(Arrays.asList("sepsis_probability"))
    .config(sepsisConfig)
    .build();
sepsisModel.initialize();
```

### Model Type Correction:
```java
// BEFORE
.modelType(ONNXModelContainer.ModelType.MORTALITY_RISK)  // WRONG - enum doesn't exist

// AFTER
.modelType(ONNXModelContainer.ModelType.MORTALITY_PREDICTION)  // CORRECT
```

---

## 3. ClinicalFeatureExtractor API Updates

Fixed all 8 occurrences of `extractor.extract()` calls to include null parameters.

### Locations Fixed:
- Line 201: Latency profiling benchmark
- Line 270: Throughput measurement benchmark
- Line 328: Batch optimization benchmark
- Line 411: Memory usage profiling benchmark
- Line 458: Parallel speedup analysis (sequential)
- Line 590: Warmup models method

### Pattern Applied:

```java
// BEFORE (Missing parameters - WRONG)
ClinicalFeatureVector features = extractor.extract(patient);

// AFTER (With null parameters - CORRECT)
ClinicalFeatureVector features = extractor.extract(patient, null, null);
```

**Rationale**: New API signature expects:
- `PatientContextSnapshot patient` (required)
- `List<Observation> observations` (optional - null for benchmarks)
- `List<Medication> medications` (optional - null for benchmarks)

---

## 4. Model Prediction API Updates

Fixed all 11 occurrences of `model.predict()` calls to use float array parameter.

### Locations Fixed:
- Line 204-205: Latency profiling
- Line 271-272: Throughput measurement
- Line 338-339: Batch optimization (in loop)
- Line 412-413: Memory profiling
- Line 459-472: Parallel speedup sequential execution (4 models)
- Line 496-503: Parallel speedup parallel execution (4 futures)
- Line 591-597: Warmup models (4 models)

### Pattern Applied:

```java
// BEFORE (Direct features object - WRONG)
MLPrediction prediction = sepsisModel.predict(features, patient.getPatientId());

// AFTER (Convert to float array - CORRECT)
float[] featureArray = features.toFloatArray();
MLPrediction prediction = sepsisModel.predict(featureArray);
```

**Rationale**: New API expects `float[]` instead of `ClinicalFeatureVector` object and no longer takes patientId parameter.

---

## 5. PatientContextSnapshot Data Type Fixes

Fixed data type mismatches in test patient generation.

### Timestamp Fix (Line 556):
```java
// BEFORE (long - WRONG)
patient.setTimestamp(System.currentTimeMillis());

// AFTER (Instant - CORRECT)
patient.setTimestamp(java.time.Instant.ofEpochMilli(System.currentTimeMillis()));
```

### Vitals Type Fixes (Lines 563-568):
```java
// BEFORE (int - WRONG)
patient.setHeartRate(60 + random.nextInt(40));
patient.setSystolicBP(100 + random.nextInt(50));

// AFTER (Double - CORRECT)
patient.setHeartRate((double)(60 + random.nextInt(40)));
patient.setSystolicBP((double)(100 + random.nextInt(50)));
```

### Labs Type Fixes (Lines 571-577):
```java
// BEFORE (Wrong method name)
patient.setWbcCount(4.0 + random.nextDouble() * 11.0);

// AFTER (Correct method name)
patient.setWhiteBloodCells(4.0 + random.nextDouble() * 11.0);
```

**Fixed setter methods**:
- `setWbcCount()` → `setWhiteBloodCells()`
- Cast int values to Double for vitals
- Cast int values to Double for platelets and sodium

---

## 6. Compilation Verification

### Build Result:
```
[INFO] BUILD SUCCESS
[INFO] Total time:  2.489 s
```

### Error Count:
- **Before fixes**: 10 compilation errors
- **After fixes**: 0 compilation errors ✅

---

## API Consistency Across Test Suite

Both test files now use identical patterns:

| Test File | Status | Patterns Applied |
|-----------|--------|------------------|
| `Module5IntegrationTest.java` | ✅ Fixed | Builder, extract(x3), predict(float[]) |
| `Module5PerformanceBenchmark.java` | ✅ Fixed | Builder, extract(x3), predict(float[]) |

---

## Total Fixes Applied

1. **Import statements**: 4 additions/corrections
2. **Model initialization**: 4 models × builder pattern = 4 fixes
3. **Feature extraction**: 8 calls × null parameters = 8 fixes
4. **Model prediction**: 11 calls × float array conversion = 11 fixes
5. **Patient data types**: 10 setter corrections
6. **Enum correction**: 1 ModelType fix

**Total**: ~38 individual fixes across the file

---

## Key Takeaways

### Refactoring Patterns Applied:

1. **Constructor → Builder Pattern**: All model initialization now uses fluent builder API
2. **Feature Extraction Signature**: Added null parameters for optional lists
3. **Prediction Input Type**: Convert ClinicalFeatureVector to float[] before prediction
4. **Type Safety**: Ensure all setter methods receive correct parameter types
5. **Time Representation**: Use Instant instead of long for timestamps

### Benefits:

- **Consistency**: Both test files now use identical API patterns
- **Type Safety**: Proper type conversions prevent runtime errors
- **Maintainability**: Builder pattern makes configuration explicit and discoverable
- **Extensibility**: Null parameters allow future addition of observations/medications

---

## Verification

### Compile Check:
```bash
mvn test-compile -Dtest=Module5PerformanceBenchmark
# Result: BUILD SUCCESS ✅
```

### Next Steps:
1. Run benchmarks to verify runtime behavior
2. Update mock model generator if needed for ONNX files
3. Consider adding observation/medication data to benchmarks for realistic testing

---

**Date**: 2025-11-03
**Refactoring Expert**: Applied SOLID principles and safe transformation patterns
**Quality Metrics**: 100% compilation success, 0 errors, consistent API usage
