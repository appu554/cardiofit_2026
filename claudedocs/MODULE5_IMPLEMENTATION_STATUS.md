# MODULE 5: ML INFERENCE & REAL-TIME RISK SCORING - IMPLEMENTATION STATUS

**Date**: 2025-11-01
**Current Status**: 40% Complete (Framework Stub → Production Implementation)
**Priority**: HIGH - Critical for Clinical Decision Support

---

## 📋 EXECUTIVE SUMMARY

Module 5 currently has a solid architectural foundation but lacks the core ML infrastructure required for production deployment. This document outlines the current state, gaps, and implementation roadmap to achieve production-ready ML inference capabilities.

### Current State
- ✅ **Architecture**: Well-designed multi-stream ML pipeline
- ✅ **Data Models**: Comprehensive MLPrediction with nested metadata
- ✅ **Kafka Integration**: Proper topic routing with side outputs
- ❌ **ONNX Runtime**: Not integrated - using simulated inference
- ❌ **Clinical Features**: 30 semantic features vs 70 clinical features required
- ❌ **Explainability**: Data structures exist but not populated
- ❌ **Alert Integration**: Missing CEP+ML alert enhancement layer

### Implementation Metrics
| Metric | Target | Current | Gap |
|--------|--------|---------|-----|
| Production Code | 5,500 lines | 945 lines | -83% |
| Clinical Features | 70 features | 30 features | -57% |
| ONNX Models | 4 models (97MB) | 0 models | -100% |
| Test Coverage | 191 tests | 0 tests | -100% |
| Components | 7 major | 3 major | -57% |

---

## 🏗️ ARCHITECTURE OVERVIEW

### Current Implementation Architecture

```
┌─────────────────────────────────────────────────────────────┐
│         MODULE 5: CURRENT IMPLEMENTATION (v0.4)              │
└─────────────────────────────────────────────────────────────┘

Input Streams:
┌────────────────────┐    ┌────────────────────┐
│ Semantic Events    │    │ Pattern Events     │
│ (Module 3)         │    │ (Module 4)         │
└─────────┬──────────┘    └─────────┬──────────┘
          │                         │
          ▼                         ▼
┌────────────────────┐    ┌────────────────────┐
│ Semantic Feature   │    │ Pattern Feature    │
│ Extractor          │    │ Extractor          │
│ (~17 features)     │    │ (~11 features)     │
└─────────┬──────────┘    └─────────┬──────────┘
          │                         │
          └────────┬────────────────┘
                   │
                   ▼
          ┌────────────────┐
          │ Feature        │
          │ Combiner       │
          │ (~30 features) │
          └────────┬───────┘
                   │
                   ▼
          ┌────────────────────────────┐
          │ ML Inference Processor     │
          │ (5 Simulated Models)       │
          │ - Readmission Risk         │
          │ - Sepsis Prediction        │
          │ - Deterioration Risk       │
          │ - Fall Risk                │
          │ - Mortality Risk           │
          └────────┬───────────────────┘
                   │
                   ▼
          ┌────────────────┐
          │ Ensemble       │
          │ Processor      │
          │ (Averaging)    │
          └────────┬───────┘
                   │
          ┌────────┴────────┬────────────┬──────────┐
          ▼                 ▼            ▼          ▼
   ┌──────────┐    ┌─────────────┐  ┌────────┐  ┌──────┐
   │inference-│    │clinical-    │  │alert-  │  │safety│
   │results   │    │reasoning    │  │mgmt    │  │events│
   └──────────┘    └─────────────┘  └────────┘  └──────┘
```

### Target Production Architecture

```
┌─────────────────────────────────────────────────────────────┐
│         MODULE 5: TARGET IMPLEMENTATION (v1.0)               │
└─────────────────────────────────────────────────────────────┘

Input Streams:
┌────────────────────┐    ┌────────────────────┐    ┌──────────────┐
│ Enriched Events    │    │ Pattern Events     │    │ Patient State│
│ (Module 4 CEP)     │    │ (Module 4 CEP)     │    │ (Module 2)   │
└─────────┬──────────┘    └─────────┬──────────┘    └──────┬───────┘
          │                         │                      │
          └────────────┬────────────┴──────────────────────┘
                       │
                       ▼
          ┌─────────────────────────┐
          │ Clinical Feature        │
          │ Extraction Pipeline     │
          │                         │
          │ Demographics (5)        │
          │ Vitals (12)            │
          │ Labs (15)              │
          │ Clinical Scores (5)    │
          │ Temporal (10)          │
          │ Medications (8)        │
          │ Comorbidities (10)     │
          │ CEP Patterns (5)       │
          │ TOTAL: 70 features     │
          └────────┬────────────────┘
                   │
                   ▼
          ┌────────────────────────────┐
          │ Feature Validation &       │
          │ Normalization              │
          └────────┬───────────────────┘
                   │
                   ▼
          ┌────────────────────────────┐
          │ ONNX Runtime Inference     │
          │ (5 Production Models)      │
          │                            │
          │ ┌────────────────────┐    │
          │ │ Mortality Model    │    │
          │ │ (APACHE IV-based)  │    │
          │ │ AUROC: 0.87        │    │
          │ └────────────────────┘    │
          │                            │
          │ ┌────────────────────┐    │
          │ │ Sepsis Model       │    │
          │ │ (InSight LSTM)     │    │
          │ │ AUROC: 0.83        │    │
          │ │ Lead: 2-6 hours    │    │
          │ └────────────────────┘    │
          │                            │
          │ ┌────────────────────┐    │
          │ │ Readmission Model  │    │
          │ │ (HOSPITAL+ XGBoost)│    │
          │ │ AUROC: 0.79        │    │
          │ └────────────────────┘    │
          │                            │
          │ ┌────────────────────┐    │
          │ │ AKI Model          │    │
          │ │ (KDIGO-ML)         │    │
          │ │ AUROC: 0.80        │    │
          │ └────────────────────┘    │
          │                            │
          │ ┌────────────────────┐    │
          │ │ Deterioration Model│    │
          │ │ (LSTM-based)       │    │
          │ └────────────────────┘    │
          └────────┬───────────────────┘
                   │
                   ▼
          ┌────────────────────────────┐
          │ SHAP Explainability        │
          │ (Feature Importance)       │
          └────────┬───────────────────┘
                   │
                   ▼
          ┌────────────────────────────┐
          │ Model Ensemble &           │
          │ Calibration                │
          │ (Weighted Averaging)       │
          └────────┬───────────────────┘
                   │
                   ▼
          ┌────────────────────────────┐
          │ ML Alert Generation        │
          │ (Threshold-based)          │
          └────────┬───────────────────┘
                   │
                   ▼
          ┌────────────────────────────┐
          │ Alert Enhancement Function │
          │ (CEP + ML Integration)     │
          └────────┬───────────────────┘
                   │
          ┌────────┴────────┬──────────┬──────────┐
          ▼                 ▼          ▼          ▼
   ┌──────────┐    ┌─────────────┐  ┌────────┐  ┌──────────┐
   │ml-       │    │enhanced-    │  │model-  │  │prediction│
   │predictions│   │alerts       │  │metrics │  │quality   │
   └──────────┘    └─────────────┘  └────────┘  └──────────┘
```

---

## 📊 COMPONENT IMPLEMENTATION STATUS

### ✅ Component 5A: ML Model Infrastructure (40% Complete)

#### Implemented
- ✅ Basic MLModel abstraction class
- ✅ Model applicability checking
- ✅ Simple threshold-based risk categorization
- ✅ Recommendation generation by model type
- ✅ 5 model definitions (readmission, sepsis, deterioration, fall, mortality)

#### Missing (HIGH PRIORITY)
- ❌ **ONNX Runtime Integration** (Critical)
  - Location: `com.cardiofit.flink.ml.ONNXModelContainer`
  - Dependencies: `ai.onnxruntime:onnxruntime:1.16.0`
  - Effort: 2-3 days
  - LOC: ~500 lines

- ❌ **Model Loading Infrastructure** (Critical)
  - Load ONNX models from resources/S3
  - Model version management
  - Hot-swapping capability
  - Effort: 1-2 days
  - LOC: ~200 lines

- ❌ **Batch Inference Optimization** (Medium)
  - `predictBatch()` implementation
  - Async inference with thread pools
  - Effort: 1 day
  - LOC: ~150 lines

- ❌ **Model Performance Tracking** (Medium)
  - Inference latency metrics
  - Throughput monitoring
  - Model accuracy tracking
  - Effort: 1 day
  - LOC: ~150 lines

**Status Files**:
- Current: [Module5_MLInference.java:386-661](../../backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module5_MLInference.java#L386-L661)
- Target: Create `com.cardiofit.flink.ml.ONNXModelContainer`

---

### 🟡 Component 5B: Feature Engineering Pipeline (43% Complete)

#### Implemented
- ✅ SemanticFeatureExtractor (~17 features from Module 3)
- ✅ PatternFeatureExtractor (~11 features from Module 4)
- ✅ FeatureCombiner with stateful merging
- ✅ Temporal relevance features (length_of_stay_hours)
- ✅ Clinical context features (acuity_score, medication_count)

#### Missing (HIGH PRIORITY)
- ❌ **70-Feature Clinical Extraction** (Critical)
  - Demographics: age, gender, BMI, admission_source
  - Vitals: HR, BP, RR, temp, O2, derived metrics (MAP, shock_index)
  - Labs: lactate, creatinine, BUN, electrolytes, CBC
  - Scores: NEWS2, qSOFA, SOFA, APACHE
  - Effort: 2-3 days
  - LOC: ~400 lines

- ❌ **Feature Validation & Normalization** (High)
  - Missing value imputation
  - Outlier detection (Winsorization)
  - Standard scaling
  - Feature completeness checking
  - Effort: 1-2 days
  - LOC: ~200 lines

- ❌ **Feature Store Integration** (Medium)
  - Redis/DynamoDB caching
  - Feature versioning
  - Historical feature retrieval
  - Effort: 2-3 days
  - LOC: ~250 lines

**Status Files**:
- Current: [Module5_MLInference.java:201-379](../../backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module5_MLInference.java#L201-L379)
- Target: Create `com.cardiofit.flink.ml.features.ClinicalFeatureExtractor`

---

### ❌ Component 5C: SHAP Explainability (10% Complete)

#### Implemented
- ✅ ExplainabilityData data structure in MLPrediction
- ✅ Basic feature importance placeholder

#### Missing (HIGH PRIORITY)
- ❌ **SHAP Integration** (Critical for Clinical Adoption)
  - SHAP library integration (`ai.djl:djl-shap`)
  - TreeSHAP for XGBoost models
  - DeepSHAP for neural networks
  - Feature contribution visualization data
  - Effort: 2-3 days
  - LOC: ~300 lines

- ❌ **Explanation Generation** (High)
  - Natural language explanations
  - "Why this prediction?" summaries
  - Top contributing factors
  - Effort: 1 day
  - LOC: ~150 lines

**Status Files**:
- Current: [MLPrediction.java:462-486](../../backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/MLPrediction.java#L462-L486)
- Target: Create `com.cardiofit.flink.ml.explainability.SHAPCalculator`

---

### ❌ Component 5D: Alert Enhancement Layer (0% Complete)

#### Missing (CRITICAL)
- ❌ **AlertEnhancementFunction** (Critical for Integration)
  - Merge CEP alerts with ML predictions
  - Agreement scoring (CEP confidence × ML confidence)
  - Combined interpretation generation
  - Enhanced recommendations
  - Effort: 2-3 days
  - LOC: ~350 lines

- ❌ **ML Alert Generator** (High)
  - Threshold-based alert triggering
  - ML-only alerts (high confidence predictions)
  - Alert prioritization logic
  - Effort: 1-2 days
  - LOC: ~250 lines

**Status**: Not started
**Target**: Create `com.cardiofit.flink.ml.alerts.AlertEnhancementFunction`

---

### 🟡 Component 5E: Model Monitoring (5% Complete)

#### Implemented
- ✅ Basic model metadata in MLPrediction

#### Missing (MEDIUM PRIORITY)
- ❌ **Model Performance Monitoring**
  - Real-time AUROC tracking
  - Prediction distribution monitoring
  - Latency percentiles (p50, p95, p99)
  - Effort: 1-2 days
  - LOC: ~200 lines

- ❌ **Model Drift Detection**
  - Feature drift detection
  - Prediction drift detection
  - Alert on drift threshold
  - Effort: 2-3 days
  - LOC: ~250 lines

- ❌ **Model Registry**
  - Model versioning
  - A/B testing framework
  - Shadow mode deployment
  - Effort: 2-3 days
  - LOC: ~300 lines

**Target**: Create `com.cardiofit.flink.ml.monitoring.ModelMonitoringService`

---

### ✅ Component 5F: Data Models (90% Complete)

#### Implemented
- ✅ MLPrediction with comprehensive metadata (552 lines)
- ✅ ConfidenceInterval nested class
- ✅ ClinicalInterpretation nested class
- ✅ PredictionQuality nested class
- ✅ EnsembleMetadata nested class
- ✅ ExplainabilityData nested class
- ✅ Model enums (ModelType, RiskLevel, ValidationStatus)
- ✅ Utility methods (isHighRisk, requiresImmediateAttention, etc.)

#### Minor Enhancements Needed
- 🟡 Add model performance metadata fields
- 🟡 Add SHAP-specific fields to ExplainabilityData

**Status**: Excellent - requires minimal changes

---

## 🎯 IMPLEMENTATION ROADMAP

### Phase 1: ONNX Runtime Foundation (Week 1)
**Goal**: Enable real ML model inference

#### Day 1: ONNX Dependencies & Setup
- [ ] Add ONNX Runtime to pom.xml (onnxruntime:1.16.0)
- [ ] Create ONNXModelContainer.java skeleton
- [ ] Test basic ONNX model loading with sample model
- [ ] Write 5 unit tests for model container

**Deliverable**: ONNX Runtime operational, can load and run basic model

#### Day 2-3: Model Container Implementation
- [ ] Implement full ONNXModelContainer (250 lines)
  - Model initialization with OrtEnvironment
  - Inference with input/output tensor handling
  - Performance metrics tracking
  - Resource cleanup
- [ ] Create model loading from resources
- [ ] Implement batch inference optimization
- [ ] Write 15 unit tests

**Deliverable**: Production-ready ONNX model container

#### Day 4-5: Model Integration
- [ ] Integrate ONNXModelContainer with MLInferenceProcessor
- [ ] Replace simulated inference with ONNX calls
- [ ] Add model performance logging
- [ ] Create sample ONNX model for sepsis prediction
- [ ] Write 10 integration tests

**Deliverable**: At least 1 real ONNX model running in pipeline

---

### Phase 2: Clinical Feature Engineering (Week 2)
**Goal**: Extract 70 clinical features from Module 2 state

#### Day 6-7: Clinical Feature Extraction
- [ ] Create ClinicalFeatureExtractor.java (400 lines)
  - Demographics (5 features)
  - Vitals (12 features)
  - Labs (15 features)
  - Clinical scores (5 features)
  - Temporal (10 features)
  - Medications (8 features)
  - Comorbidities (10 features)
  - CEP patterns (5 features)
- [ ] Integrate with Module 2 patient state
- [ ] Write 20 unit tests

**Deliverable**: 70-feature extraction working

#### Day 8: Feature Validation & Normalization
- [ ] Create FeatureValidator.java (150 lines)
- [ ] Implement missing value imputation
- [ ] Add outlier detection (Winsorization)
- [ ] Create standard scaler
- [ ] Write 12 unit tests

**Deliverable**: Robust feature preprocessing pipeline

#### Day 9-10: Feature Schema & Documentation
- [ ] Create feature-schema-v1.yaml
- [ ] Document all 70 features
- [ ] Add feature importance documentation
- [ ] Create feature engineering guide
- [ ] Write 8 integration tests

**Deliverable**: Complete feature engineering documentation

---

### Phase 3: Explainability & Alert Integration (Week 3)
**Goal**: Add SHAP explainability and CEP+ML alert integration

#### Day 11-12: SHAP Integration
- [ ] Add SHAP library dependency
- [ ] Create SHAPCalculator.java (250 lines)
- [ ] Implement TreeSHAP for XGBoost models
- [ ] Implement DeepSHAP for neural networks
- [ ] Generate feature importance for predictions
- [ ] Write 10 unit tests

**Deliverable**: SHAP explainability working for all models

#### Day 13-14: Alert Enhancement Layer
- [ ] Create AlertEnhancementFunction.java (350 lines)
- [ ] Implement CEP + ML alert merging
- [ ] Add agreement scoring logic
- [ ] Generate combined interpretations
- [ ] Create enhanced recommendations
- [ ] Write 15 integration tests

**Deliverable**: Unified alert system with CEP+ML

#### Day 15: ML Alert Generation
- [ ] Create MLAlertGenerator.java (250 lines)
- [ ] Implement threshold-based triggering
- [ ] Add ML-only alert generation
- [ ] Create alert prioritization logic
- [ ] Write 10 unit tests

**Deliverable**: ML alerts publishing to Kafka

---

### Phase 4: Monitoring & Production Readiness (Week 4)
**Goal**: Add monitoring, testing, and production deployment

#### Day 16-17: Model Monitoring
- [ ] Create ModelMonitoringService.java (200 lines)
- [ ] Implement real-time AUROC tracking
- [ ] Add prediction distribution monitoring
- [ ] Create drift detection (250 lines)
- [ ] Add Prometheus metrics export
- [ ] Write 12 unit tests

**Deliverable**: Complete model monitoring infrastructure

#### Day 18: Model Registry & Versioning
- [ ] Create ModelRegistry.java (220 lines)
- [ ] Implement model versioning
- [ ] Add A/B testing framework
- [ ] Create shadow mode deployment
- [ ] Write 8 integration tests

**Deliverable**: Model deployment infrastructure

#### Day 19: Comprehensive Testing
- [ ] Write 50+ unit tests across all components
- [ ] Create 25+ integration tests
- [ ] Add performance tests (10K events/sec)
- [ ] Create load tests with Flink JobManager
- [ ] Document test coverage report

**Deliverable**: >80% test coverage

#### Day 20: Documentation & Deployment
- [ ] Complete technical documentation
- [ ] Create operational runbooks
- [ ] Write deployment guide
- [ ] Create clinician training materials
- [ ] Prepare production deployment checklist

**Deliverable**: Production-ready Module 5

---

## 📁 FILE STRUCTURE (Target)

```
src/main/java/com/cardiofit/flink/
├── ml/
│   ├── ONNXModelContainer.java              (250 lines) ← NEW
│   ├── ModelConfig.java                     (100 lines) ← NEW
│   ├── ModelMetrics.java                    (80 lines)  ← NEW
│   │
│   ├── features/
│   │   ├── ClinicalFeatureExtractor.java    (400 lines) ← NEW
│   │   ├── FeatureVector.java               (150 lines) ← EXPAND
│   │   ├── FeatureValidator.java            (150 lines) ← NEW
│   │   ├── FeatureNormalizer.java           (120 lines) ← NEW
│   │   └── FeatureDefinition.java           (120 lines) ← NEW
│   │
│   ├── models/
│   │   ├── MortalityPredictionModel.java    (300 lines) ← NEW
│   │   ├── SepsisOnsetModel.java            (350 lines) ← NEW
│   │   ├── ReadmissionRiskModel.java        (250 lines) ← NEW
│   │   ├── AKIProgressionModel.java         (280 lines) ← NEW
│   │   └── DeteriorationModel.java          (290 lines) ← NEW
│   │
│   ├── explainability/
│   │   ├── SHAPCalculator.java              (250 lines) ← NEW
│   │   ├── FeatureImportanceService.java    (200 lines) ← NEW
│   │   └── ExplanationGenerator.java        (150 lines) ← NEW
│   │
│   ├── alerts/
│   │   ├── AlertEnhancementFunction.java    (350 lines) ← NEW
│   │   ├── MLAlertGenerator.java            (250 lines) ← NEW
│   │   └── MLAlert.java                     (120 lines) ← NEW
│   │
│   └── monitoring/
│       ├── ModelMonitoringService.java      (200 lines) ← NEW
│       ├── DriftDetector.java               (250 lines) ← NEW
│       ├── PerformanceTracker.java          (150 lines) ← NEW
│       └── ModelRegistry.java               (220 lines) ← NEW
│
├── operators/
│   └── Module5_MLInference.java             (500 lines) ← REFACTOR
│
└── models/
    └── MLPrediction.java                    (552 lines) ✅ COMPLETE

src/main/resources/
└── models/
    ├── mortality_prediction_v1.onnx         (25 MB)     ← NEW
    ├── sepsis_onset_v1.onnx                 (32 MB)     ← NEW
    ├── readmission_risk_v1.onnx             (18 MB)     ← NEW
    ├── aki_progression_v1.onnx              (22 MB)     ← NEW
    └── deterioration_risk_v1.onnx           (28 MB)     ← NEW

src/main/resources/config/
├── feature-schema-v1.yaml                   (500 lines) ← NEW
├── model-registry.yaml                      (200 lines) ← NEW
└── ml-config.properties                     (100 lines) ← NEW

src/test/java/com/cardiofit/flink/ml/
├── ONNXModelContainerTest.java              (15 tests)  ← NEW
├── ClinicalFeatureExtractorTest.java        (20 tests)  ← NEW
├── SHAPCalculatorTest.java                  (10 tests)  ← NEW
├── AlertEnhancementFunctionTest.java        (15 tests)  ← NEW
├── ModelMonitoringServiceTest.java          (12 tests)  ← NEW
└── Module5IntegrationTest.java              (25 tests)  ← NEW

TOTAL NEW CODE: ~4,500 lines production code
TOTAL NEW TESTS: ~100+ tests
TOTAL MODEL FILES: ~125 MB
```

---

## 🚨 CRITICAL DEPENDENCIES

### Maven Dependencies to Add

```xml
<!-- ONNX Runtime -->
<dependency>
    <groupId>com.microsoft.onnxruntime</groupId>
    <artifactId>onnxruntime</artifactId>
    <version>1.16.0</version>
</dependency>

<!-- SHAP for Explainability (via DJL) -->
<dependency>
    <groupId>ai.djl</groupId>
    <artifactId>api</artifactId>
    <version>0.24.0</version>
</dependency>

<!-- For Feature Store (Optional) -->
<dependency>
    <groupId>org.redisson</groupId>
    <artifactId>redisson</artifactId>
    <version>3.24.3</version>
</dependency>
```

### Python Model Training Requirements
```python
# For ONNX model export
onnx==1.15.0
onnxruntime==1.16.0
tf2onnx==1.15.1  # For TensorFlow models
skl2onnx==1.16.0  # For scikit-learn models

# For SHAP
shap==0.44.0

# For model training
xgboost==2.0.2
tensorflow==2.15.0
scikit-learn==1.3.2
```

---

## 📊 SUCCESS METRICS

### Technical Performance
- [ ] ONNX inference latency: <15ms (p99) per model
- [ ] Total pipeline latency: <50ms (p99) for all 5 models
- [ ] Feature extraction: <10ms (p99)
- [ ] Throughput: >10,000 events/sec sustained
- [ ] Test coverage: >80%

### Clinical Accuracy (Post-Training)
- [ ] Mortality model AUROC: ≥0.85
- [ ] Sepsis model AUROC: ≥0.83 with 2-6 hour lead time
- [ ] Readmission model AUROC: ≥0.79
- [ ] AKI model AUROC: ≥0.80
- [ ] False positive rate: <10%
- [ ] Sensitivity: >90% for critical events

### Production Readiness
- [ ] All 70 features extracted correctly
- [ ] SHAP explanations for all predictions
- [ ] Model monitoring operational
- [ ] Alert enhancement integrated with Module 4
- [ ] Comprehensive test suite passing
- [ ] Documentation complete

---

## 🎯 NEXT ACTIONS

### Immediate (This Week)
1. ✅ Complete this implementation status document
2. ⏳ Add ONNX Runtime dependencies to pom.xml
3. ⏳ Create ONNXModelContainer skeleton
4. ⏳ Test basic ONNX model loading

### Short Term (Week 2-3)
5. Implement 70-feature clinical extraction
6. Integrate SHAP explainability
7. Build alert enhancement layer
8. Create comprehensive test suite

### Medium Term (Week 4)
9. Add model monitoring and drift detection
10. Implement model registry and versioning
11. Performance testing and optimization
12. Production deployment

---

## 📚 REFERENCES

- **Documentation**: [Module_5_ML_Inference_&_Real-Time_Risk_Scoring.txt](../../backend/shared-infrastructure/flink-processing/src/docs/module_5/Module_5_ ML_Inference_&_Real-Time_Risk_Scoring.txt)
- **Current Code**: [Module5_MLInference.java](../../backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module5_MLInference.java)
- **Data Model**: [MLPrediction.java](../../backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/MLPrediction.java)
- **ONNX Runtime Docs**: https://onnxruntime.ai/docs/
- **SHAP Library**: https://shap.readthedocs.io/

---

**Status**: Ready for implementation - All gaps identified, roadmap defined, priorities set.
