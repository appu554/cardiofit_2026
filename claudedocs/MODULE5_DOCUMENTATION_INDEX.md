# MODULE 5: ONNX MODEL TRAINING & DEPLOYMENT DOCUMENTATION INDEX

**Last Updated**: November 1, 2025
**Status**: Complete Documentation Package
**Total Documents**: 3 comprehensive guides
**Total Lines**: 4,653 lines of documentation
**Total Size**: 148KB

---

## DOCUMENTATION OVERVIEW

This index provides a roadmap through the complete MODULE 5 documentation package for ONNX model training, export, and deployment. Use this guide to navigate to the specific information you need.

---

## DOCUMENT STRUCTURE

### Document 1: MODEL TRAINING PIPELINE
**File**: `/Users/apoorvabk/Downloads/cardiofit/claudedocs/MODULE5_MODEL_TRAINING_PIPELINE.md`
**Size**: 61 KB | **Lines**: 1,870 | **Reading Time**: 60 minutes
**Audience**: ML Engineers, Data Scientists

**Purpose**: Complete guide for training clinical risk prediction models from data preparation through ONNX export

**Key Sections**:
- Overview & Key Characteristics
- End-to-End Pipeline Architecture
- Data Preparation (cohort definition, labeling, deduplication, splitting)
- Feature Engineering (70-feature specification & extraction code)
- Model Training (XGBoost, hyperparameter tuning, cross-validation)
- Model Validation (performance metrics, calibration, fairness)
- ONNX Export & Optimization (conversion, quantization, validation)
- Deployment Workflow (versioning, A/B testing, canary deployment)
- Monitoring & Retraining (drift detection, performance monitoring, retraining triggers)
- Troubleshooting Guide (10 common issues with solutions)

**Code Examples Included**:
- Data preparation & splitting (stratified, temporal)
- SMOTE & class weight implementation
- XGBoost baseline & Optuna hyperparameter tuning
- 5-fold cross-validation
- Performance metric calculation
- ONNX conversion & validation
- Quantization (INT8)
- Production monitoring (drift detection, PSI)

**Use When**:
- Training new models from scratch
- Optimizing existing model hyperparameters
- Converting models to ONNX format
- Setting up production monitoring
- Troubleshooting model training issues

---

### Document 2: ONNX MODEL SPECIFICATIONS
**File**: `/Users/apoorvabk/Downloads/cardiofit/claudedocs/MODULE5_ONNX_MODEL_SPECIFICATIONS.md`
**Size**: 41 KB | **Lines**: 1,193 | **Reading Time**: 45 minutes
**Audience**: ML Engineers, DevOps, Clinical Teams

**Purpose**: Technical specifications for all four clinical prediction models (Sepsis, Deterioration, Mortality, Readmission)

**Key Sections**:
- Model Overview (comparison table for all 4 models)
- Sepsis Risk Model (input/output, features, targets, constraints)
- Patient Deterioration Model (6-24h prediction specification)
- Mortality Risk Model (in-hospital mortality specification)
- Readmission Risk Model (30-day readmission specification)
- Input Feature Specification (complete 70-feature documentation)
- Output Format Specification (probability tensors, thresholds)
- Model Constraints (technical, clinical, compliance)
- Example Training Code (complete Python workflow)
- Validation Checklist (pre-deployment items)

**Features Documented**:
All 70 features organized by 8 categories:
- Demographics (5 features)
- Vital Signs (12 features)
- Laboratory Values (15 features)
- Clinical Scores (5 features)
- Temporal Features (10 features)
- Medications (8 features)
- Comorbidities (10 features)
- Reserved for Future (6 features)

**Input/Output Specification**:
```
INPUT:  float_input, shape (batch_size, 70), float32, range [0.0, 1.0]
OUTPUT: probabilities, shape (batch_size, 2), float32, sum = 1.0
```

**Threshold Recommendations**:
- Sepsis: 0.45 (sensitivity 0.81, specificity 0.80)
- Deterioration: 0.50 (sensitivity 0.75, specificity 0.80)
- Mortality: 0.25 (sensitivity 0.70, specificity 0.85)
- Readmission: 0.30 (sensitivity 0.65, specificity 0.85)

**Use When**:
- Need to understand model specifications
- Implementing feature extraction
- Setting up model input validation
- Determining classification thresholds
- Pre-deployment validation

---

### Document 3: MODEL DEPLOYMENT CHECKLIST
**File**: `/Users/apoorvabk/Downloads/cardiofit/claudedocs/MODULE5_MODEL_DEPLOYMENT_CHECKLIST.md`
**Size**: 46 KB | **Lines**: 1,590 | **Reading Time**: 50 minutes
**Audience**: DevOps, ML Engineers, Project Managers, Clinical Integration

**Purpose**: Step-by-step operational checklist for safe model deployment to production

**Key Phases**:
1. **Phase 1**: Pre-Deployment Validation (1 week)
   - Model artifact verification
   - Input/output schema validation
   - Numerical equivalence testing
   - Data consistency checks

2. **Phase 2**: Performance Benchmarking (1 week)
   - Inference latency (<15ms p99)
   - Memory footprint (<200MB)
   - Model file size verification

3. **Phase 3**: Integration Testing (1 week)
   - Java integration (ONNXModelContainer)
   - Feature extraction pipeline
   - End-to-end workflow

4. **Phase 4**: Security & Compliance (1 week)
   - Security verification (HIPAA, file permissions)
   - Compliance verification (FDA, bias audit)
   - Documentation package

5. **Phase 5**: A/B Testing Setup (1-4 weeks)
   - Canary deployment configuration
   - Validation criteria
   - Progressive rollout plan (10% → 25% → 50% → 100%)

6. **Phase 6**: Monitoring Configuration (ongoing)
   - Real-time metrics collection
   - Drift detection (PSI > 0.25)
   - Performance monitoring
   - Alerting rules

7. **Phase 7**: Rollback Planning (pre-deployment)
   - Rollback decision criteria
   - Execution procedure
   - Post-rollback analysis

8. **Phase 8**: Sign-Off & Deployment
   - ML Engineering sign-off
   - Clinical Affairs sign-off
   - IT Security sign-off

**Checklists Included**: 50+ detailed checkboxes with verification commands

**Timeline**: 2-4 weeks total (1 week pre-deployment, 1-4 weeks A/B testing, ongoing monitoring)

**Use When**:
- Deploying model to production
- Setting up A/B testing
- Configuring production monitoring
- Planning deployment phases
- Preparing rollback procedures

---

## QUICK REFERENCE: CHOOSING YOUR DOCUMENT

**Q: I need to train a new model from scratch**
→ **MODULE5_MODEL_TRAINING_PIPELINE.md**
- Sections 3-7: Complete workflow
- Use code examples for data prep through ONNX export
- Time: 1-2 weeks training, 1-2 days export

**Q: I need to understand model specifications**
→ **MODULE5_ONNX_MODEL_SPECIFICATIONS.md**
- Sections 1-5: Model specs
- Section 6: 70-feature documentation
- Section 7: Input/output details
- Time: 45 minutes reading

**Q: I'm deploying a model to production**
→ **MODULE5_MODEL_DEPLOYMENT_CHECKLIST.md**
- Follow phases 1-8 sequentially
- Use checklists for verification
- Time: 2-4 weeks (1-2 weeks pre-deployment, 1-4 weeks deployment)

**Q: I need to understand the 70 features**
→ **MODULE5_ONNX_MODEL_SPECIFICATIONS.md**, Section 6
- Complete specification for all 70 features
- Data types, ranges, clinical significance
- Imputation and normalization strategies

**Q: I need to troubleshoot a training issue**
→ **MODULE5_MODEL_TRAINING_PIPELINE.md**, Section 10
- Class imbalance, calibration, latency, ONNX validation, feature extraction
- Root cause analysis and solutions for each

**Q: I need to deploy safely with A/B testing**
→ **MODULE5_MODEL_DEPLOYMENT_CHECKLIST.md**, Phase 5
- Canary deployment (10% traffic initially)
- Validation criteria for phase progression
- Progressive rollout to 100%

**Q: I need to set up production monitoring**
→ **MODULE5_MODEL_DEPLOYMENT_CHECKLIST.md**, Phase 6
- Real-time metrics collection
- Drift detection (PSI > 0.25)
- Performance monitoring (AUROC, precision, recall)
- Alerting rules

---

## KEY METRICS REFERENCE

### Performance Targets by Model

| Metric | Sepsis | Deterioration | Mortality | Readmission |
|--------|--------|---------------|-----------|------------|
| AUROC Target | >0.85 | >0.82 | >0.80 | >0.78 |
| Sensitivity Target | >0.80 | >0.75 | >0.70 | >0.65 |
| Specificity Target | >0.80 | >0.80 | >0.85 | >0.85 |
| Min Positive Cases | 2,000 | 1,500 | 500 | 800 |

### Technical Requirements

| Requirement | Target | Notes |
|-----------|--------|-------|
| Input Features | 70 | float32, [0, 1] normalized |
| Output Shape | (batch_size, 2) | Probabilities [0.0, 1.0] |
| Inference Latency P99 | <15ms | Single prediction |
| Throughput | >100/sec | Batch inference |
| Model Size | <50MB per model | <200MB for all 4 |
| Memory Footprint | <200MB | Loaded in memory |

### Retraining Triggers

| Trigger | Condition | Action | Frequency |
|---------|----------|--------|-----------|
| Scheduled | Every 90 days | Retrain with new data | Quarterly |
| Drift | PSI > 0.25 | Investigate + retrain | Monthly if triggered |
| Performance | AUROC drop >5% | Urgent retraining | Immediate |
| Data | >10,000 new cases | Evaluate & consider retrain | Ongoing |

---

## IMPLEMENTATION ROADMAP

### Training Phase (Week 1-2)
```
Day 1-2:   Data preparation & cohort definition
Day 3-4:   Feature extraction & engineering
Day 5:     Model training (baseline + tuning)
Day 6:     Model validation & threshold optimization
Day 7:     ONNX export & verification
Day 8-14:  Integration testing & refinement
```

### Pre-Deployment Phase (Week 3-4)
```
Day 1-2:   Security & compliance verification
Day 3:     Integration testing with Java layer
Day 4:     Fairness audit & bias evaluation
Day 5:     Performance benchmarking
Day 6:     Final sign-off preparation
```

### Deployment Phase (Week 5-8+)
```
Week 1:    Phase 1 (10% traffic canary)
Week 2:    Phase 2 (25% traffic graduated)
Week 3:    Phase 3 (50% traffic majority) [optional]
Week 4:    Phase 4 (100% traffic production)
Ongoing:   Production monitoring & maintenance
```

---

## DOCUMENTATION STATISTICS

| Document | Size | Lines | Sections | Code Examples |
|----------|------|-------|----------|----------------|
| MODEL_TRAINING_PIPELINE.md | 61 KB | 1,870 | 11 | 15+ |
| ONNX_MODEL_SPECIFICATIONS.md | 41 KB | 1,193 | 10 | 3 |
| MODEL_DEPLOYMENT_CHECKLIST.md | 46 KB | 1,590 | 9 | 10+ |
| **TOTAL** | **148 KB** | **4,653** | **30** | **28+** |

---

## RELATED MODULE 5 DOCUMENTATION

For additional context, see:

1. **MODULE5_ONNX_IMPLEMENTATION_COMPLETE.md** - Java ONNXModelContainer implementation details
2. **MODULE5_PHASE2_FEATURE_ENGINEERING_COMPLETE.md** - Detailed 70-feature extraction
3. **MODULE5_PHASE3_EXPLAINABILITY_ALERTS_COMPLETE.md** - SHAP explainability integration
4. **MODULE5_CURRENT_STATUS_REPORT.md** - Current implementation status

---

## CONTACT & SUPPORT

**Technical Questions** (training, ONNX, integration)
- ML Engineering: mleng@cardiofit.com

**Deployment Questions** (A/B testing, monitoring)
- DevOps Team: devops@cardiofit.com

**Clinical Questions** (fairness, safety, validation)
- Clinical Affairs: clinical@cardiofit.com

---

**Document Last Updated**: November 1, 2025
**Next Review**: February 1, 2026
**Maintainer**: ML Engineering Team
