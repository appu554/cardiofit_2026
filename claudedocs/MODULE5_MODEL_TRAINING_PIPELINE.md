# MODULE 5: ONNX MODEL TRAINING AND DEPLOYMENT PIPELINE

**Last Updated**: November 1, 2025
**Status**: Production Documentation
**Version**: 1.0.0
**Target Audience**: ML Engineers, Data Scientists, DevOps Teams

---

## TABLE OF CONTENTS

1. [Overview](#overview)
2. [Architecture & Workflow](#architecture--workflow)
3. [Data Preparation](#data-preparation)
4. [Feature Engineering](#feature-engineering)
5. [Model Training](#model-training)
6. [Model Validation](#model-validation)
7. [ONNX Export & Optimization](#onnx-export--optimization)
8. [Deployment Workflow](#deployment-workflow)
9. [Monitoring & Retraining](#monitoring--retraining)
10. [Troubleshooting Guide](#troubleshooting-guide)

---

## OVERVIEW

### Purpose

The Module 5 ML Pipeline trains, validates, and deploys clinical risk prediction models to the CardioFit platform. The pipeline produces ONNX-format models for four critical clinical prediction tasks:

1. **Sepsis Risk** - Early identification of sepsis development
2. **Patient Deterioration** - 6-24 hour clinical deterioration prediction
3. **Mortality Risk** - Hospital mortality probability
4. **Readmission Risk** - 30-day unplanned readmission

### Key Characteristics

- **Input Features**: 70 clinical features extracted from patient state
- **Output Format**: Binary classification (risk probability [0.0, 1.0])
- **Model Types**: XGBoost, LightGBM, CatBoost (gradient boosting recommended)
- **Deployment Format**: ONNX Runtime (cross-platform, optimized)
- **Inference Target**: <15ms per prediction (Java inference layer)
- **HIPAA Compliance**: Audit logging, data anonymization, secure storage

### Training Frequency

- **Baseline**: Quarterly (every 90 days)
- **Drift Trigger**: Monthly (if PSI > 0.25)
- **Performance Trigger**: Immediate (if AUROC drops >5%)
- **Data Requirements**: Minimum 10,000 positive cases per model

---

## ARCHITECTURE & WORKFLOW

### End-to-End Pipeline

```
┌─────────────────┐
│  Data Extract   │
│ (FHIR Store)    │
└────────┬────────┘
         │
         ▼
┌─────────────────────────┐
│  Data Preparation       │
│ - Cohort definition     │
│ - Label assignment      │
│ - Deduplication         │
└────────┬────────────────┘
         │
         ▼
┌─────────────────────────┐
│  Feature Engineering    │
│ - 70 features extracted │
│ - Imputation strategy   │
│ - Quality validation    │
└────────┬────────────────┘
         │
         ▼
┌─────────────────────────┐
│  Data Splitting         │
│ - Train: 70% (≥10K)     │
│ - Val: 15% (≥1K)        │
│ - Test: 15% (≥1K)       │
└────────┬────────────────┘
         │
         ▼
┌─────────────────────────┐
│  Class Imbalance        │
│ - SMOTE (train only)    │
│ - Class weights (model) │
│ - Threshold optimization│
└────────┬────────────────┘
         │
         ▼
┌─────────────────────────┐
│  Model Training         │
│ - 5-fold CV stratified  │
│ - Hyperparameter tuning │
│ - Baseline comparison   │
└────────┬────────────────┘
         │
         ▼
┌─────────────────────────┐
│  Model Validation       │
│ - Performance metrics   │
│ - Calibration check     │
│ - Fairness evaluation   │
└────────┬────────────────┘
         │
         ▼
┌─────────────────────────┐
│  ONNX Export            │
│ - Model conversion      │
│ - Input/output schema   │
│ - Quantization (opt)    │
└────────┬────────────────┘
         │
         ▼
┌─────────────────────────┐
│  Deployment             │
│ - Versioning (v1.0.0)   │
│ - A/B testing (10%)     │
│ - Canary deployment     │
└────────┬────────────────┘
         │
         ▼
┌─────────────────────────┐
│  Production Monitoring  │
│ - Drift detection       │
│ - Performance tracking  │
│ - Retraining triggers   │
└─────────────────────────┘
```

---

## DATA PREPARATION

### Cohort Definition

#### Sepsis Risk Model

**Inclusion Criteria**:
- Adult patients (≥18 years)
- ICU or high-acuity ward admission
- At least 24 hours of observation
- Vital signs within first 6 hours of admission

**Exclusion Criteria**:
- Immunocompromised status (transient exclusion only)
- Missing lactate or WBC on admission
- Admission for palliative care

**Positive Cases** (Sepsis-3 Definition):
- Suspected infection (cultures, antibiotics)
- SOFA score increase ≥2 points within 48 hours

**Sample Size**: Minimum 2,000 positive cases (≥5% of total cohort)

#### Patient Deterioration Model

**Inclusion Criteria**:
- All hospitalized patients
- At least 12 hours post-admission monitoring
- Valid vital signs available

**Positive Cases**: Clinical deterioration defined as:
- Unplanned ICU transfer from floor
- Sepsis onset within 6-24 hours
- In-hospital cardiac arrest
- Unplanned intubation

**Sample Size**: Minimum 1,500 positive cases

#### Mortality Risk Model

**Inclusion Criteria**:
- All hospitalized patients ≥24 hours stay
- Complete admission demographics
- At least one full vital signs set

**Positive Cases**:
- In-hospital mortality (death before discharge)

**Sample Size**: Minimum 500 positive cases (2-5% baseline)

#### Readmission Risk Model

**Inclusion Criteria**:
- Patients with completed hospital discharge
- Initial stay ≥24 hours
- Valid discharge diagnosis codes

**Positive Cases**:
- Unplanned readmission within 30 days

**Sample Size**: Minimum 800 positive cases (3-8% baseline)

### Label Assignment Strategy

**Outcome Window**:
```python
# Sepsis Risk
outcome_window = (admission_time, admission_time + timedelta(hours=48))

# Deterioration Risk
outcome_window = (prediction_time, prediction_time + timedelta(hours=24))

# Mortality Risk
outcome_window = (admission_time, discharge_time)

# Readmission Risk
outcome_window = (discharge_time, discharge_time + timedelta(days=30))
```

**Label Quality Assurance**:
- Validate outcome definitions against clinical guidelines
- Cross-check with structured and unstructured (NLP) data
- Manual review of 100 random positive cases (80% agreement threshold)

### Deduplication & Data Cleaning

```python
import pandas as pd
import numpy as np

# Step 1: Remove exact duplicates
df = df.drop_duplicates(subset=['patient_id', 'encounter_id', 'admission_time'])

# Step 2: Handle duplicate patient entries (keep most recent)
df = df.sort_values('data_collection_date').groupby('encounter_id').tail(1)

# Step 3: Remove encounters with critical missing data
critical_features = [
    'vital_heart_rate', 'vital_systolic_bp', 'vital_temperature_c',
    'lab_creatinine_mg_dl', 'lab_wbc_k_ul', 'lab_lactate_mmol'
]
df = df.dropna(subset=critical_features)

# Step 4: Clinical validity checks
df = df[df['demo_age_years'].between(18, 120)]
df = df[df['vital_systolic_bp'].between(40, 250)]
df = df[df['vital_heart_rate'].between(20, 220)]

print(f"Final dataset: {len(df)} records, {df['label'].sum()} positive cases")
print(f"Positive class ratio: {df['label'].mean():.2%}")
```

### Data Split Strategy

**Temporal Split** (Recommended for time-series stability):

```python
from datetime import datetime, timedelta

# Sort by admission date
df_sorted = df.sort_values('admission_date')
total_records = len(df_sorted)

# Calculate split indices (chronological)
train_end = int(total_records * 0.70)
val_end = int(total_records * 0.85)

X_train = df_sorted.iloc[:train_end]
X_val = df_sorted.iloc[train_end:val_end]
X_test = df_sorted.iloc[val_end:]

y_train = X_train['label'].values
y_val = X_val['label'].values
y_test = X_test['label'].values

print(f"Train set: {len(X_train)} records ({y_train.mean():.2%} positive)")
print(f"Val set: {len(X_val)} records ({y_val.mean():.2%} positive)")
print(f"Test set: {len(X_test)} records ({y_test.mean():.2%} positive)")
```

**Stratified Split** (Ensures balanced class representation):

```python
from sklearn.model_selection import train_test_split

# First split: 70% train, 30% temp
X_temp, X_test, y_temp, y_test = train_test_split(
    df, df['label'],
    test_size=0.15,
    random_state=42,
    stratify=df['label']
)

# Second split: from temp, create 70% train and 15% val
X_train, X_val, y_train, y_val = train_test_split(
    X_temp, y_temp,
    test_size=(0.15 / 0.85),  # Adjust ratio
    random_state=42,
    stratify=y_temp
)
```

### Handling Class Imbalance

#### SMOTE (Training Data Only)

```python
from imblearn.over_sampling import SMOTE
from imblearn.under_sampling import RandomUnderSampler
from imblearn.pipeline import Pipeline

# Balanced sampling pipeline (train only)
resampler = Pipeline([
    ('under', RandomUnderSampler(sampling_strategy=0.5, random_state=42)),
    ('over', SMOTE(k_neighbors=5, random_state=42))
])

X_train_balanced, y_train_balanced = resampler.fit_resample(X_train, y_train)

print(f"Original: {y_train.sum()}/{len(y_train)} positive ({y_train.mean():.2%})")
print(f"Balanced: {y_train_balanced.sum()}/{len(y_train_balanced)} positive ({y_train_balanced.mean():.2%})")
```

#### Class Weight Strategy (XGBoost)

```python
from sklearn.utils.class_weight import compute_class_weight

# Calculate class weights
class_weights = compute_class_weight(
    'balanced',
    classes=np.unique(y_train),
    y=y_train
)

# Convert to XGBoost scale_pos_weight
scale_pos_weight = class_weights[0] / class_weights[1]

print(f"Class weights: {dict(zip(['negative', 'positive'], class_weights))}")
print(f"XGBoost scale_pos_weight: {scale_pos_weight:.2f}")
```

---

## FEATURE ENGINEERING

### 70-Feature Overview

The clinical feature set is organized into 8 categories:

| Category | Count | Purpose |
|----------|-------|---------|
| Demographics | 5 | Age, gender, BMI, acuity |
| Vitals (Real-time) | 12 | Heart rate, BP, O2 sat, etc. |
| Labs | 15 | Lactate, creatinine, CBC, CMP |
| Clinical Scores | 5 | SOFA, qSOFA, APACHE, NEWS2 |
| Temporal | 10 | Trends, time since measurement |
| Medications | 8 | Active medications, counts |
| Comorbidities | 10 | Chronic conditions |
| **Total** | **70** | **Complete clinical picture** |

### Feature Extraction Code

```python
from typing import Dict, List
import pandas as pd
import numpy as np
from datetime import datetime, timedelta

class ClinicalFeatureExtractor:
    """Extract 70-feature clinical vector from patient state."""

    def __init__(self, patient_data: Dict, reference_time: datetime):
        self.patient = patient_data
        self.reference_time = reference_time
        self.features = {}

    def extract_all(self) -> np.ndarray:
        """Extract complete 70-feature vector."""
        self._extract_demographics()
        self._extract_vitals()
        self._extract_labs()
        self._extract_scores()
        self._extract_temporal()
        self._extract_medications()
        self._extract_comorbidities()
        return self._to_array()

    def _extract_demographics(self):
        """5 demographic features."""
        self.features['demo_age_years'] = self.patient.get('age', 65)
        self.features['demo_gender_male'] = 1 if self.patient.get('gender') == 'M' else 0
        self.features['demo_bmi'] = self.patient.get('bmi', 25)
        self.features['demo_icu_patient'] = 1 if self.patient.get('location') == 'ICU' else 0
        self.features['demo_admission_emergency'] = 1 if self.patient.get('admission_type') == 'EMERGENCY' else 0

    def _extract_vitals(self):
        """12 vital sign features."""
        vitals = self.patient.get('latest_vitals', {})

        # Raw measurements
        self.features['vital_heart_rate'] = vitals.get('heart_rate', 80)
        self.features['vital_systolic_bp'] = vitals.get('systolic_bp', 120)
        self.features['vital_diastolic_bp'] = vitals.get('diastolic_bp', 80)
        self.features['vital_respiratory_rate'] = vitals.get('respiratory_rate', 16)
        self.features['vital_temperature_c'] = vitals.get('temperature_c', 37.0)
        self.features['vital_oxygen_saturation'] = vitals.get('oxygen_saturation', 98)

        # Derived metrics
        sys = self.features['vital_systolic_bp']
        dia = self.features['vital_diastolic_bp']
        hr = self.features['vital_heart_rate']

        self.features['vital_mean_arterial_pressure'] = (sys + 2*dia) / 3
        self.features['vital_pulse_pressure'] = sys - dia
        self.features['vital_shock_index'] = hr / self.features['vital_mean_arterial_pressure']

        # Flag features
        self.features['vital_hr_abnormal'] = 1 if hr < 60 or hr > 100 else 0
        self.features['vital_bp_hypotensive'] = 1 if sys < 90 else 0
        self.features['vital_fever'] = 1 if self.features['vital_temperature_c'] > 38.0 else 0

    def _extract_labs(self):
        """15 lab features."""
        labs = self.patient.get('latest_labs', {})

        # Core labs
        self.features['lab_lactate_mmol'] = labs.get('lactate', 1.0)
        self.features['lab_creatinine_mg_dl'] = labs.get('creatinine', 1.0)
        self.features['lab_bun_mg_dl'] = labs.get('bun', 18)
        self.features['lab_sodium_meq'] = labs.get('sodium', 140)
        self.features['lab_potassium_meq'] = labs.get('potassium', 4.0)
        self.features['lab_chloride_meq'] = labs.get('chloride', 105)
        self.features['lab_bicarbonate_meq'] = labs.get('bicarbonate', 24)
        self.features['lab_wbc_k_ul'] = labs.get('wbc', 10)
        self.features['lab_hemoglobin_g_dl'] = labs.get('hemoglobin', 14)
        self.features['lab_platelets_k_ul'] = labs.get('platelets', 250)
        self.features['lab_ast_u_l'] = labs.get('ast', 30)
        self.features['lab_alt_u_l'] = labs.get('alt', 30)
        self.features['lab_bilirubin_mg_dl'] = labs.get('bilirubin', 0.5)

        # Flags
        self.features['lab_lactate_elevated'] = 1 if self.features['lab_lactate_mmol'] > 2.0 else 0
        self.features['lab_aki_present'] = 1 if self._compute_aki_stage() > 0 else 0

    def _extract_scores(self):
        """5 clinical score features."""
        scores = self.patient.get('clinical_scores', {})

        self.features['score_news2'] = scores.get('news2', 2)
        self.features['score_qsofa'] = scores.get('qsofa', 0)
        self.features['score_sofa'] = scores.get('sofa', 0)
        self.features['score_apache'] = scores.get('apache', 10)
        self.features['score_acuity_combined'] = scores.get('acuity', 2)

    def _extract_temporal(self):
        """10 temporal features."""
        admission_time = self.patient.get('admission_time')

        hours_since_admission = (self.reference_time - admission_time).total_seconds() / 3600
        self.features['temporal_hours_since_admission'] = max(0, hours_since_admission)
        self.features['temporal_hours_since_last_vitals'] = self.patient.get('vitals_age_hours', 1)
        self.features['temporal_hours_since_last_labs'] = self.patient.get('labs_age_hours', 6)
        self.features['temporal_length_of_stay_hours'] = hours_since_admission

        # Trends (1 = increasing/worsening, 0 = stable/improving)
        self.features['temporal_hr_trend_increasing'] = 1 if self.patient.get('hr_trend') == 'increasing' else 0
        self.features['temporal_bp_trend_decreasing'] = 1 if self.patient.get('bp_trend') == 'decreasing' else 0
        self.features['temporal_lactate_trend_increasing'] = 1 if self.patient.get('lactate_trend') == 'increasing' else 0

        # Circadian features
        hour = self.reference_time.hour
        self.features['temporal_hour_of_day'] = hour
        self.features['temporal_is_night_shift'] = 1 if hour >= 20 or hour < 6 else 0
        self.features['temporal_is_weekend'] = 1 if self.reference_time.weekday() >= 5 else 0

    def _extract_medications(self):
        """8 medication features."""
        meds = self.patient.get('active_medications', [])

        self.features['med_total_count'] = len(meds)
        self.features['med_high_risk_count'] = sum(1 for m in meds if m.get('risk_level') == 'HIGH')
        self.features['med_vasopressor_active'] = 1 if any(m.get('class') == 'VASOPRESSOR' for m in meds) else 0
        self.features['med_antibiotic_active'] = 1 if any(m.get('class') == 'ANTIBIOTIC' for m in meds) else 0
        self.features['med_anticoagulation_active'] = 1 if any(m.get('class') == 'ANTICOAGULANT' for m in meds) else 0
        self.features['med_sedation_active'] = 1 if any(m.get('class') == 'SEDATIVE' for m in meds) else 0
        self.features['med_insulin_active'] = 1 if any(m.get('class') == 'INSULIN' for m in meds) else 0
        self.features['med_polypharmacy'] = 1 if len(meds) >= 5 else 0

    def _extract_comorbidities(self):
        """10 comorbidity features (binary flags)."""
        comorbid = self.patient.get('comorbidities', {})

        comorbid_list = [
            'diabetes', 'hypertension', 'ckd', 'heart_failure',
            'copd', 'cancer', 'immunosuppressed', 'stroke',
            'liver_disease', 'aids'
        ]

        for i, condition in enumerate(comorbid_list):
            self.features[f'comorbid_{condition}'] = 1 if comorbid.get(condition) else 0

    def _to_array(self) -> np.ndarray:
        """Convert feature dict to 70-element array."""
        # Maintain consistent feature order
        feature_order = [
            # Demographics (5)
            'demo_age_years', 'demo_gender_male', 'demo_bmi', 'demo_icu_patient', 'demo_admission_emergency',
            # Vitals (12)
            'vital_heart_rate', 'vital_systolic_bp', 'vital_diastolic_bp', 'vital_respiratory_rate',
            'vital_temperature_c', 'vital_oxygen_saturation', 'vital_mean_arterial_pressure',
            'vital_pulse_pressure', 'vital_shock_index', 'vital_hr_abnormal', 'vital_bp_hypotensive', 'vital_fever',
            # Labs (15)
            'lab_lactate_mmol', 'lab_creatinine_mg_dl', 'lab_bun_mg_dl', 'lab_sodium_meq',
            'lab_potassium_meq', 'lab_chloride_meq', 'lab_bicarbonate_meq', 'lab_wbc_k_ul',
            'lab_hemoglobin_g_dl', 'lab_platelets_k_ul', 'lab_ast_u_l', 'lab_alt_u_l', 'lab_bilirubin_mg_dl',
            'lab_lactate_elevated', 'lab_aki_present',
            # Scores (5)
            'score_news2', 'score_qsofa', 'score_sofa', 'score_apache', 'score_acuity_combined',
            # Temporal (10)
            'temporal_hours_since_admission', 'temporal_hours_since_last_vitals', 'temporal_hours_since_last_labs',
            'temporal_length_of_stay_hours', 'temporal_hr_trend_increasing', 'temporal_bp_trend_decreasing',
            'temporal_lactate_trend_increasing', 'temporal_hour_of_day', 'temporal_is_night_shift', 'temporal_is_weekend',
            # Medications (8)
            'med_total_count', 'med_high_risk_count', 'med_vasopressor_active', 'med_antibiotic_active',
            'med_anticoagulation_active', 'med_sedation_active', 'med_insulin_active', 'med_polypharmacy',
            # Comorbidities (10)
            'comorbid_diabetes', 'comorbid_hypertension', 'comorbid_ckd', 'comorbid_heart_failure',
            'comorbid_copd', 'comorbid_cancer', 'comorbid_immunosuppressed', 'comorbid_stroke',
            'comorbid_liver_disease', 'comorbid_aids'
        ]

        return np.array([self.features.get(f, 0.0) for f in feature_order], dtype=np.float32)
```

### Missing Data Imputation

```python
from sklearn.impute import SimpleImputer, KNNImputer
import numpy as np

class FeatureImputer:
    """Impute missing values using clinical domain knowledge."""

    IMPUTATION_STRATEGY = {
        # Vitals: use median
        'vital_': 'median',
        # Labs: use median (distribution often right-skewed)
        'lab_': 'median',
        # Scores: use median
        'score_': 'median',
        # Binary features: use most_frequent (0 if absent)
        'demo_gender_male': 'most_frequent',
        '_active': 'most_frequent',
        'comorbid_': 'most_frequent',
    }

    @staticmethod
    def impute_features(X: np.ndarray, feature_names: List[str]) -> np.ndarray:
        """Impute missing values with strategy-specific handling."""
        X_imputed = X.copy()

        for i, feature in enumerate(feature_names):
            strategy = 'median'  # default

            # Determine strategy by feature name
            for pattern, strat in FeatureImputer.IMPUTATION_STRATEGY.items():
                if pattern in feature:
                    strategy = strat
                    break

            # Apply imputation
            imputer = SimpleImputer(strategy=strategy)
            X_imputed[:, i:i+1] = imputer.fit_transform(X[:, i:i+1])

        return X_imputed
```

### Feature Normalization

```python
from sklearn.preprocessing import StandardScaler, MinMaxScaler, RobustScaler
import numpy as np

class FeatureNormalizer:
    """Normalize features for model training."""

    def __init__(self, method: str = 'standard'):
        """
        Initialize normalizer.
        Methods: 'standard' (z-score), 'minmax', 'robust'
        """
        self.method = method
        self.scaler = None
        self._init_scaler()

    def _init_scaler(self):
        if self.method == 'standard':
            self.scaler = StandardScaler()
        elif self.method == 'minmax':
            self.scaler = MinMaxScaler(feature_range=(0, 1))
        elif self.method == 'robust':
            self.scaler = RobustScaler()
        else:
            raise ValueError(f"Unknown normalization method: {self.method}")

    def fit_and_transform(self, X_train: np.ndarray) -> np.ndarray:
        """Fit on training data and transform."""
        return self.scaler.fit_transform(X_train)

    def transform(self, X: np.ndarray) -> np.ndarray:
        """Transform using fitted parameters."""
        return self.scaler.transform(X)

    def inverse_transform(self, X: np.ndarray) -> np.ndarray:
        """Reverse normalization (for interpretation)."""
        return self.scaler.inverse_transform(X)

# Usage
normalizer = FeatureNormalizer(method='standard')
X_train_norm = normalizer.fit_and_transform(X_train)
X_val_norm = normalizer.transform(X_val)
X_test_norm = normalizer.transform(X_test)
```

---

## MODEL TRAINING

### Algorithm Selection

#### Recommended: Gradient Boosting Ensemble

**Why Gradient Boosting**:
- Superior performance on tabular clinical data
- Excellent handling of mixed feature types (continuous + binary)
- Built-in feature importance (for explainability)
- Handles non-linear relationships in vital signs/labs
- Production-proven in healthcare ML applications

#### Algorithm Comparison

| Algorithm | AUROC | Latency | Interpretability | ONNX Support |
|-----------|-------|---------|-----------------|--------------|
| **XGBoost** | 0.82-0.88 | 2-5ms | Excellent | Yes |
| **LightGBM** | 0.81-0.87 | 1-3ms | Good | Yes |
| **CatBoost** | 0.81-0.87 | 3-6ms | Excellent | Yes |
| Logistic Regression | 0.75-0.80 | <1ms | Excellent | Yes |
| Random Forest | 0.80-0.85 | 5-10ms | Good | Yes |

### XGBoost Training Example

```python
import xgboost as xgb
from sklearn.metrics import roc_auc_score, precision_score, recall_score, f1_score
import optuna
import numpy as np

class XGBoostModelTrainer:
    """Train XGBoost model for clinical prediction."""

    def __init__(self, model_name: str = 'sepsis', random_state: int = 42):
        self.model_name = model_name
        self.random_state = random_state
        self.model = None
        self.best_params = None
        self.feature_importance = None

    def train_baseline(self, X_train: np.ndarray, y_train: np.ndarray,
                       scale_pos_weight: float = 1.0):
        """Train baseline model with default hyperparameters."""

        self.model = xgb.XGBClassifier(
            n_estimators=100,
            max_depth=6,
            learning_rate=0.1,
            subsample=0.8,
            colsample_bytree=0.8,
            scale_pos_weight=scale_pos_weight,
            random_state=self.random_state,
            n_jobs=-1,
            eval_metric='logloss'
        )

        self.model.fit(X_train, y_train, verbose=10)
        print(f"Baseline model trained on {len(X_train)} samples")
        return self.model

    def hyperparameter_tuning(self, X_train: np.ndarray, y_train: np.ndarray,
                             X_val: np.ndarray, y_val: np.ndarray,
                             scale_pos_weight: float = 1.0, n_trials: int = 50):
        """Optimize hyperparameters using Optuna."""

        def objective(trial: optuna.Trial) -> float:
            # Suggest hyperparameters
            params = {
                'n_estimators': trial.suggest_int('n_estimators', 50, 300),
                'max_depth': trial.suggest_int('max_depth', 4, 10),
                'learning_rate': trial.suggest_float('learning_rate', 0.01, 0.5, log=True),
                'subsample': trial.suggest_float('subsample', 0.5, 1.0),
                'colsample_bytree': trial.suggest_float('colsample_bytree', 0.5, 1.0),
                'lambda': trial.suggest_float('lambda', 0.0, 5.0),  # L2 regularization
                'alpha': trial.suggest_float('alpha', 0.0, 5.0),   # L1 regularization
                'scale_pos_weight': scale_pos_weight,
                'random_state': self.random_state,
                'n_jobs': -1,
                'eval_metric': 'logloss'
            }

            # Train model
            model = xgb.XGBClassifier(**params)
            model.fit(X_train, y_train, verbose=0)

            # Evaluate on validation set
            y_pred_proba = model.predict_proba(X_val)[:, 1]
            auroc = roc_auc_score(y_val, y_pred_proba)

            return auroc

        # Run optimization
        sampler = optuna.samplers.TPESampler(seed=self.random_state)
        study = optuna.create_study(direction='maximize', sampler=sampler)
        study.optimize(objective, n_trials=n_trials, show_progress_bar=True)

        # Train final model with best parameters
        self.best_params = study.best_params
        self.best_params['scale_pos_weight'] = scale_pos_weight
        self.best_params['random_state'] = self.random_state
        self.best_params['n_jobs'] = -1
        self.best_params['eval_metric'] = 'logloss'

        self.model = xgb.XGBClassifier(**self.best_params)
        self.model.fit(X_train, y_train, verbose=10)

        print(f"Best AUROC: {study.best_value:.4f}")
        print(f"Best parameters: {self.best_params}")
        return study

    def cross_validate(self, X: np.ndarray, y: np.ndarray,
                       k_folds: int = 5, scale_pos_weight: float = 1.0):
        """5-fold stratified cross-validation."""
        from sklearn.model_selection import StratifiedKFold, cross_validate

        skf = StratifiedKFold(n_splits=k_folds, shuffle=True,
                              random_state=self.random_state)

        model = xgb.XGBClassifier(
            n_estimators=150,
            max_depth=6,
            learning_rate=0.1,
            subsample=0.8,
            colsample_bytree=0.8,
            scale_pos_weight=scale_pos_weight,
            random_state=self.random_state,
            n_jobs=-1,
            eval_metric='logloss'
        )

        scoring = {
            'auroc': 'roc_auc',
            'precision': 'precision',
            'recall': 'recall',
            'f1': 'f1'
        }

        cv_results = cross_validate(
            model, X, y,
            cv=skf,
            scoring=scoring,
            return_train_score=True,
            n_jobs=-1
        )

        # Print results
        print(f"\n5-Fold Cross-Validation Results ({self.model_name}):")
        print("=" * 60)
        for metric in ['auroc', 'precision', 'recall', 'f1']:
            test_scores = cv_results[f'test_{metric}']
            train_scores = cv_results[f'train_{metric}']
            print(f"{metric.upper():12} | Train: {train_scores.mean():.4f}±{train_scores.std():.4f} | "
                  f"Test: {test_scores.mean():.4f}±{test_scores.std():.4f}")

        return cv_results

    def get_feature_importance(self, top_n: int = 20) -> dict:
        """Get top N important features."""
        if self.model is None:
            raise ValueError("Model not trained yet")

        importance = self.model.feature_importances_
        feature_names = [f"feature_{i}" for i in range(len(importance))]

        # Sort by importance
        sorted_idx = np.argsort(importance)[::-1][:top_n]

        importance_dict = {
            feature_names[i]: importance[i]
            for i in sorted_idx
        }

        return importance_dict

    def evaluate_on_test_set(self, X_test: np.ndarray, y_test: np.ndarray,
                             threshold: float = 0.5):
        """Evaluate model on held-out test set."""

        if self.model is None:
            raise ValueError("Model not trained yet")

        y_pred_proba = self.model.predict_proba(X_test)[:, 1]
        y_pred = (y_pred_proba >= threshold).astype(int)

        # Calculate metrics
        auroc = roc_auc_score(y_test, y_pred_proba)
        precision = precision_score(y_test, y_pred)
        recall = recall_score(y_test, y_pred)
        f1 = f1_score(y_test, y_pred)

        # Sensitivity at different thresholds
        from sklearn.metrics import roc_curve
        fpr, tpr, thresholds = roc_curve(y_test, y_pred_proba)

        print(f"\n{self.model_name.upper()} Model - Test Set Evaluation:")
        print("=" * 60)
        print(f"AUROC:     {auroc:.4f}")
        print(f"Precision: {precision:.4f}")
        print(f"Recall:    {recall:.4f}")
        print(f"F1-Score:  {f1:.4f}")

        return {
            'auroc': auroc,
            'precision': precision,
            'recall': recall,
            'f1': f1,
            'roc_curve': (fpr, tpr, thresholds),
            'y_pred_proba': y_pred_proba
        }

# Usage Example
trainer = XGBoostModelTrainer(model_name='sepsis')

# Step 1: Train baseline
trainer.train_baseline(X_train_norm, y_train, scale_pos_weight=2.5)

# Step 2: Hyperparameter tuning
study = trainer.hyperparameter_tuning(X_train_norm, y_train, X_val_norm, y_val,
                                      scale_pos_weight=2.5, n_trials=50)

# Step 3: Cross-validation
cv_results = trainer.cross_validate(X_train_norm, y_train, k_folds=5,
                                    scale_pos_weight=2.5)

# Step 4: Test set evaluation
test_metrics = trainer.evaluate_on_test_set(X_test_norm, y_test, threshold=0.5)
```

### LightGBM Alternative

```python
import lightgbm as lgb

class LightGBMModelTrainer:
    """Train LightGBM model (faster alternative to XGBoost)."""

    def __init__(self, model_name: str = 'sepsis', random_state: int = 42):
        self.model_name = model_name
        self.random_state = random_state
        self.model = None

    def train(self, X_train: np.ndarray, y_train: np.ndarray,
              X_val: np.ndarray, y_val: np.ndarray,
              scale_pos_weight: float = 1.0):
        """Train LightGBM with early stopping."""

        # Create LightGBM datasets
        train_data = lgb.Dataset(X_train, label=y_train)
        val_data = lgb.Dataset(X_val, label=y_val, reference=train_data)

        # Parameters
        params = {
            'objective': 'binary',
            'metric': 'auc',
            'num_leaves': 31,
            'learning_rate': 0.05,
            'feature_fraction': 0.8,
            'bagging_fraction': 0.8,
            'bagging_freq': 5,
            'scale_pos_weight': scale_pos_weight,
            'num_threads': -1,
            'verbose': -1
        }

        # Train with early stopping
        self.model = lgb.train(
            params,
            train_data,
            num_boost_round=200,
            valid_sets=[val_data],
            early_stopping_rounds=20,
            verbose_eval=10
        )

        print(f"LightGBM model trained with {self.model.num_trees()} trees")
        return self.model
```

---

## MODEL VALIDATION

### Performance Metrics Framework

```python
from sklearn.metrics import (
    roc_auc_score, precision_score, recall_score, f1_score,
    confusion_matrix, classification_report, roc_curve, precision_recall_curve
)
import numpy as np

class ModelValidator:
    """Comprehensive model validation with clinical metrics."""

    @staticmethod
    def compute_metrics(y_true: np.ndarray, y_pred_proba: np.ndarray,
                       threshold: float = 0.5) -> dict:
        """
        Compute comprehensive performance metrics.

        Args:
            y_true: True binary labels
            y_pred_proba: Predicted probabilities [0, 1]
            threshold: Classification threshold

        Returns:
            Dictionary of metrics
        """
        y_pred = (y_pred_proba >= threshold).astype(int)

        # Basic metrics
        auroc = roc_auc_score(y_true, y_pred_proba)
        precision = precision_score(y_true, y_pred)
        recall = recall_score(y_true, y_pred)
        f1 = f1_score(y_true, y_pred)

        # Confusion matrix
        tn, fp, fn, tp = confusion_matrix(y_true, y_pred).ravel()

        # Clinical metrics
        specificity = tn / (tn + fp)  # True negative rate
        npv = tn / (tn + fn)  # Negative predictive value
        sensitivity = recall  # True positive rate
        ppv = precision  # Positive predictive value

        # Brier score (calibration)
        brier = np.mean((y_pred_proba - y_true) ** 2)

        return {
            # Discrimination
            'auroc': auroc,
            'f1': f1,

            # Clinical interpretation
            'sensitivity': sensitivity,  # True positive rate
            'specificity': specificity,  # True negative rate
            'ppv': ppv,                  # Precision
            'npv': npv,

            # Additional
            'precision': precision,
            'recall': recall,
            'brier_score': brier,  # Calibration

            # Confusion matrix
            'true_positives': int(tp),
            'true_negatives': int(tn),
            'false_positives': int(fp),
            'false_negatives': int(fn),
        }

    @staticmethod
    def find_optimal_threshold(y_true: np.ndarray, y_pred_proba: np.ndarray,
                               metric: str = 'f1') -> float:
        """
        Find optimal classification threshold using Youden's J statistic
        or F1 score maximization.
        """
        fpr, tpr, thresholds = roc_curve(y_true, y_pred_proba)

        if metric == 'youden':
            # Youden's J = TPR - FPR (maximize sensitivity and specificity)
            j_scores = tpr - fpr
            optimal_idx = np.argmax(j_scores)
            return thresholds[optimal_idx]

        elif metric == 'f1':
            # F1 score maximization
            best_f1 = 0
            best_threshold = 0.5

            for threshold in np.arange(0.1, 0.9, 0.01):
                y_pred = (y_pred_proba >= threshold).astype(int)
                f1 = f1_score(y_true, y_pred)
                if f1 > best_f1:
                    best_f1 = f1
                    best_threshold = threshold

            return best_threshold

    @staticmethod
    def compute_calibration(y_true: np.ndarray, y_pred_proba: np.ndarray,
                           n_bins: int = 10) -> dict:
        """
        Assess calibration using reliability diagram.

        Perfect calibration: predicted probability = observed frequency
        """
        from sklearn.calibration import calibration_curve

        prob_true, prob_pred = calibration_curve(y_true, y_pred_proba, n_bins=n_bins)

        # Expected calibration error
        ece = np.mean(np.abs(prob_true - prob_pred))

        return {
            'prob_true': prob_true,
            'prob_pred': prob_pred,
            'expected_calibration_error': ece
        }

    @staticmethod
    def fairness_evaluation(y_true: np.ndarray, y_pred_proba: np.ndarray,
                           groups: np.ndarray, threshold: float = 0.5) -> dict:
        """
        Evaluate model fairness across demographic groups.

        Args:
            y_true: True labels
            y_pred_proba: Predicted probabilities
            groups: Group membership (e.g., 0=female, 1=male)
            threshold: Classification threshold
        """
        results = {}

        for group in np.unique(groups):
            mask = groups == group
            y_true_group = y_true[mask]
            y_pred_proba_group = y_pred_proba[mask]

            metrics = ModelValidator.compute_metrics(
                y_true_group, y_pred_proba_group, threshold
            )
            results[f'group_{group}'] = metrics

        # Check for disparity
        aurocs = [results[g]['auroc'] for g in results]
        auroc_disparity = max(aurocs) - min(aurocs)

        return {
            'by_group': results,
            'auroc_disparity': auroc_disparity,
            'acceptable_fairness': auroc_disparity < 0.05  # <5% difference
        }
```

### Validation Workflow

```python
# Complete validation pipeline
def validate_model_comprehensive(model, X_val, y_val, model_name='sepsis'):
    """Run complete model validation."""

    print(f"\n{'='*70}")
    print(f"MODEL VALIDATION: {model_name.upper()}")
    print(f"{'='*70}\n")

    # 1. Get predictions
    y_pred_proba = model.predict_proba(X_val)[:, 1]

    # 2. Find optimal threshold
    optimal_threshold = ModelValidator.find_optimal_threshold(
        y_val, y_pred_proba, metric='youden'
    )
    print(f"Optimal threshold (Youden): {optimal_threshold:.3f}")

    # 3. Evaluate at multiple thresholds
    print("\nPerformance at Different Thresholds:")
    print("-" * 70)
    print(f"{'Threshold':<12} {'Sensitivity':<15} {'Specificity':<15} {'F1-Score':<15}")
    print("-" * 70)

    for threshold in [0.3, 0.4, 0.5, 0.6, 0.7]:
        metrics = ModelValidator.compute_metrics(y_val, y_pred_proba, threshold)
        print(f"{threshold:<12.2f} {metrics['sensitivity']:<15.4f} "
              f"{metrics['specificity']:<15.4f} {metrics['f1']:<15.4f}")

    # 4. Primary metrics at optimal threshold
    primary_metrics = ModelValidator.compute_metrics(
        y_val, y_pred_proba, optimal_threshold
    )

    print(f"\n{'Primary Performance Metrics':<35} (threshold={optimal_threshold:.3f})")
    print("-" * 70)
    print(f"AUROC:            {primary_metrics['auroc']:.4f}")
    print(f"Sensitivity:      {primary_metrics['sensitivity']:.4f}")
    print(f"Specificity:      {primary_metrics['specificity']:.4f}")
    print(f"Precision (PPV):  {primary_metrics['ppv']:.4f}")
    print(f"F1-Score:         {primary_metrics['f1']:.4f}")
    print(f"Brier Score:      {primary_metrics['brier_score']:.4f}")

    # 5. Calibration assessment
    calibration = ModelValidator.compute_calibration(y_val, y_pred_proba)
    print(f"\nCalibration:")
    print(f"Expected Calibration Error: {calibration['expected_calibration_error']:.4f}")

    # 6. Production readiness check
    print(f"\n{'Production Readiness Assessment':<35}")
    print("-" * 70)
    checks = {
        'AUROC > 0.80': primary_metrics['auroc'] > 0.80,
        'Sensitivity > 0.75': primary_metrics['sensitivity'] > 0.75,
        'Specificity > 0.75': primary_metrics['specificity'] > 0.75,
        'Brier Score < 0.20': primary_metrics['brier_score'] < 0.20,
        'ECE < 0.10': calibration['expected_calibration_error'] < 0.10
    }

    for check, passed in checks.items():
        status = "✓ PASS" if passed else "✗ FAIL"
        print(f"{check:<40} {status}")

    all_passed = all(checks.values())
    print(f"\nOverall Status: {'✓ READY FOR PRODUCTION' if all_passed else '✗ NEEDS IMPROVEMENT'}")

    return primary_metrics, calibration, checks
```

---

## ONNX EXPORT & OPTIMIZATION

### Model Conversion to ONNX

```python
import onnx
from skl2onnx import convert_sklearn
from onnxruntime import InferenceSession
import xgboost as xgb
import numpy as np

class ONNXExporter:
    """Convert trained models to ONNX format."""

    @staticmethod
    def convert_xgboost_to_onnx(xgb_model, model_name: str = 'sepsis',
                                model_version: str = '1.0.0',
                                input_dim: int = 70):
        """
        Convert XGBoost model to ONNX.

        Args:
            xgb_model: Trained XGBoost classifier
            model_name: Model identifier (sepsis, deterioration, mortality, readmission)
            model_version: Semantic version (e.g., 1.0.0)
            input_dim: Number of input features (70 for CardioFit)

        Returns:
            ONNX model and file path
        """
        try:
            # Define input type (batch of 70 float features)
            initial_types = [('float_input', 'FloatTensorType', [None, input_dim])]

            # Convert to ONNX
            onnx_model = convert_sklearn(xgb_model, initial_types=initial_types,
                                        target_opset=12)

            # Set metadata
            onnx_model.ir_version = onnx.IR_VERSION
            onnx_model.producer_name = 'CardioFit-Module5'
            onnx_model.producer_version = model_version
            onnx_model.doc_string = f'Clinical {model_name} risk prediction model (v{model_version})'

            # Add model metadata
            meta = onnx_model.metadata_props.add()
            meta.key = 'model_name'
            meta.value = model_name

            meta = onnx_model.metadata_props.add()
            meta.key = 'input_features'
            meta.value = str(input_dim)

            meta = onnx_model.metadata_props.add()
            meta.key = 'output_type'
            meta.value = 'binary_classification_probability'

            # Verify model
            onnx.checker.check_model(onnx_model)

            # Save to file
            output_path = f'/models/{model_name}_v{model_version}.onnx'
            onnx.save(onnx_model, output_path)

            print(f"✓ ONNX model saved: {output_path}")
            print(f"  Model name: {model_name}")
            print(f"  Version: {model_version}")
            print(f"  Input shape: (batch_size, {input_dim})")
            print(f"  Output: Binary classification probability")

            return onnx_model, output_path

        except Exception as e:
            print(f"✗ Error converting model to ONNX: {e}")
            raise

    @staticmethod
    def validate_onnx_model(onnx_model_path: str, X_test: np.ndarray,
                           y_test: np.ndarray):
        """
        Validate ONNX model against original model.

        Ensures:
        - Input/output shapes are correct
        - Numerical equivalence (within tolerance)
        - Inference performance is acceptable
        """
        try:
            # Load ONNX model
            session = InferenceSession(onnx_model_path)

            # Get input/output names
            input_name = session.get_inputs()[0].name
            output_name = session.get_outputs()[0].name

            print(f"\nONNX Model Validation:")
            print(f"-" * 60)
            print(f"Input name: {input_name}")
            print(f"Output name: {output_name}")

            # Test inference on small batch
            test_batch = X_test[:100].astype(np.float32)

            # Predict
            onnx_output = session.run([output_name], {input_name: test_batch})[0]

            print(f"Input shape: {test_batch.shape}")
            print(f"Output shape: {onnx_output.shape}")
            print(f"Output sample: {onnx_output[:5].flatten()}")

            # Validate output range
            assert np.all(onnx_output >= 0) and np.all(onnx_output <= 1), \
                "Probabilities outside [0, 1] range"
            print(f"✓ Output range valid: [0, 1]")

            # Benchmark inference speed
            import time
            start = time.time()
            for _ in range(100):
                session.run([output_name], {input_name: test_batch[:1]})
            elapsed = (time.time() - start) / 100

            print(f"✓ Inference latency: {elapsed*1000:.2f}ms (target: <15ms)")

            return session

        except Exception as e:
            print(f"✗ Validation failed: {e}")
            raise

# Usage Example
# 1. Export to ONNX
onnx_model, path = ONNXExporter.convert_xgboost_to_onnx(
    xgb_model=trained_model,
    model_name='sepsis',
    model_version='1.0.0',
    input_dim=70
)

# 2. Validate ONNX model
session = ONNXExporter.validate_onnx_model(path, X_test, y_test)
```

### Quantization for Size Optimization

```python
from onnx import quantization
import onnx

class ONNXQuantizer:
    """Quantize ONNX models for reduced size and faster inference."""

    @staticmethod
    def quantize_int8(model_path: str, output_path: str = None):
        """
        Quantize model to INT8 (8-bit integers).

        Benefits:
        - 4x smaller model size
        - Faster inference on some devices
        - Minimal accuracy loss (<1% typically)

        Trade-offs:
        - Slight accuracy loss
        - Not supported on all devices
        """
        if output_path is None:
            output_path = model_path.replace('.onnx', '_int8.onnx')

        try:
            # Load model
            model = onnx.load(model_path)

            # Quantize to INT8
            quantized = quantization.quantize_dynamic(
                model_path,
                output_path,
                weight_type=quantization.QuantType.QUInt8
            )

            # Check file sizes
            import os
            original_size = os.path.getsize(model_path) / 1e6
            quantized_size = os.path.getsize(output_path) / 1e6
            compression_ratio = (1 - quantized_size/original_size) * 100

            print(f"Quantization Complete (INT8):")
            print(f"Original size:   {original_size:.2f}MB")
            print(f"Quantized size:  {quantized_size:.2f}MB")
            print(f"Compression:     {compression_ratio:.1f}%")

            return quantized

        except Exception as e:
            print(f"Quantization failed: {e}")
            raise

    @staticmethod
    def compare_quantized_accuracy(original_model_path: str,
                                   quantized_model_path: str,
                                   X_test: np.ndarray, y_test: np.ndarray):
        """
        Compare accuracy of original vs quantized model.
        Ensures accuracy loss is acceptable (<1%).
        """
        from onnxruntime import InferenceSession
        from sklearn.metrics import roc_auc_score

        # Load both models
        session_original = InferenceSession(original_model_path)
        session_quantized = InferenceSession(quantized_model_path)

        # Get input/output names
        input_name = session_original.get_inputs()[0].name
        output_name = session_original.get_outputs()[0].name

        # Get predictions
        X_test_float = X_test.astype(np.float32)

        y_pred_original = session_original.run(
            [output_name], {input_name: X_test_float}
        )[0].flatten()

        y_pred_quantized = session_quantized.run(
            [output_name], {input_name: X_test_float}
        )[0].flatten()

        # Compare
        auroc_original = roc_auc_score(y_test, y_pred_original)
        auroc_quantized = roc_auc_score(y_test, y_pred_quantized)
        accuracy_loss = (auroc_original - auroc_quantized) * 100

        print(f"\nQuantization Impact Analysis:")
        print(f"-" * 60)
        print(f"Original AUROC:  {auroc_original:.4f}")
        print(f"Quantized AUROC: {auroc_quantized:.4f}")
        print(f"Accuracy Loss:   {accuracy_loss:.2f}%")

        if accuracy_loss < 1.0:
            print(f"✓ Accuracy loss acceptable (<1%)")
            return True
        else:
            print(f"✗ Accuracy loss too high (>1%)")
            return False
```

---

## DEPLOYMENT WORKFLOW

### Model Versioning

```
Model Naming: {model_type}_{version}.onnx
Examples:
  - sepsis_v1.0.0.onnx
  - deterioration_v1.2.1.onnx
  - mortality_v2.0.0.onnx
  - readmission_v1.1.0.onnx

Version Scheme (Semantic Versioning):
  MAJOR: Significant architecture change (retrain entire model)
  MINOR: Hyperparameter tuning (same features/architecture)
  PATCH: Bug fixes or data cleaning (no model retraining)

Version History:
  v1.0.0 (2025-11-01): Initial production release
  v1.1.0 (2025-12-01): Hyperparameter optimization
  v1.1.1 (2025-12-15): Data quality fix
  v2.0.0 (2026-02-01): New features added (new deployment cycle)
```

### A/B Testing Strategy

```
Phase 1: Shadow Deployment (0% production traffic)
  - Deploy model alongside production
  - Log predictions without using them
  - Duration: 1 week
  - Success criteria: No errors, consistent performance

Phase 2: Canary Deployment (10% traffic)
  - Route 10% of predictions to new model
  - 90% continue using existing model
  - Duration: 1 week
  - Success criteria: AUROC ≥ existing model

Phase 3: Staged Rollout (50% traffic)
  - Route 50% of predictions to new model
  - Duration: 1 week
  - Success criteria: Clinical team approval

Phase 4: Full Production (100% traffic)
  - All predictions use new model
  - Maintain old model as rollback
  - Duration: 30 days monitoring

Phase 5: Sunset (Remove old model)
  - Archive old model version
  - Update documentation
```

### Deployment Configuration (Java)

```java
// ModelConfig.java - Production deployment settings
public class ModelDeploymentConfig {

    public static ModelConfig productionConfig() {
        return ModelConfig.builder()
            // Model identification
            .modelName("sepsis")
            .modelVersion("1.0.0")
            .modelPath("/models/sepsis_v1.0.0.onnx")

            // Runtime optimization
            .intraOpThreads(4)
            .interOpThreads(2)
            .enableMemoryPattern(true)
            .enableCpuMemArena(true)

            // Input/output specification
            .inputDimensions(70)
            .outputDimensions(2)
            .predictionThreshold(0.5f)

            // Batch processing
            .enableBatching(true)
            .batchSize(32)
            .batchTimeoutMs(1000)

            // Performance monitoring
            .enableMetricsCollection(true)
            .metricsReportingIntervalMs(60000)

            // Health checks
            .enableHealthCheck(true)
            .healthCheckIntervalMs(30000)

            .build();
    }

    public static ModelConfig canaryConfig() {
        // A/B testing configuration (10% traffic)
        return productionConfig()
            .toBuilder()
            .enableTrafficSampling(true)
            .trafficSamplePercentage(10)
            .build();
    }
}
```

### Rollback Procedure

```
Rollback Triggers:
  1. AUROC drops > 5% in production (automated)
  2. Inference error rate > 1% (automated)
  3. Latency > 50ms p99 (automated)
  4. Clinical team requests rollback (manual)

Rollback Steps:
  1. Stop traffic to new model immediately
  2. Verify old model is healthy
  3. Switch all traffic to old model
  4. Notify clinical team and engineering
  5. Root cause analysis
  6. Fix issues before re-attempting deployment
```

---

## MONITORING & RETRAINING

### Production Monitoring

```python
from datetime import datetime, timedelta
import numpy as np
from typing import Dict, List

class ProductionMonitor:
    """Monitor deployed models for performance degradation."""

    def __init__(self, model_name: str, baseline_auroc: float):
        self.model_name = model_name
        self.baseline_auroc = baseline_auroc
        self.predictions_log = []

    def log_prediction(self, patient_id: str, predicted_probability: float,
                      actual_outcome: bool = None):
        """Log each prediction for monitoring."""
        self.predictions_log.append({
            'timestamp': datetime.now(),
            'patient_id': patient_id,
            'predicted_probability': predicted_probability,
            'actual_outcome': actual_outcome
        })

    def detect_drift(self, window_days: int = 7) -> Dict:
        """
        Detect data drift using Population Stability Index (PSI).

        PSI = Σ (% current - % baseline) * ln(% current / % baseline)

        PSI < 0.1:   No significant drift
        PSI 0.1-0.25: Small drift, monitor
        PSI > 0.25:   Significant drift, retrain recommended
        """
        # Get baseline distribution (first week)
        baseline_predictions = [p['predicted_probability']
                               for p in self.predictions_log[:len(self.predictions_log)//4]]

        # Get recent distribution
        cutoff_date = datetime.now() - timedelta(days=window_days)
        recent_predictions = [p['predicted_probability']
                             for p in self.predictions_log
                             if p['timestamp'] > cutoff_date]

        # Compute PSI
        psi = self._compute_psi(
            np.array(baseline_predictions),
            np.array(recent_predictions)
        )

        return {
            'psi': psi,
            'drift_detected': psi > 0.25,
            'severity': 'none' if psi < 0.1 else 'small' if psi < 0.25 else 'significant'
        }

    @staticmethod
    def _compute_psi(baseline: np.ndarray, current: np.ndarray,
                     n_bins: int = 10) -> float:
        """Calculate Population Stability Index."""
        baseline_hist, bin_edges = np.histogram(baseline, bins=n_bins, range=(0, 1))
        current_hist, _ = np.histogram(current, bins=bin_edges)

        # Normalize to proportions
        baseline_prop = (baseline_hist + 1) / (baseline_hist.sum() + n_bins)
        current_prop = (current_hist + 1) / (current_hist.sum() + n_bins)

        # Calculate PSI
        psi = np.sum((current_prop - baseline_prop) * np.log(current_prop / baseline_prop))

        return psi

    def evaluate_performance(self) -> Dict:
        """
        Evaluate recent model performance.
        Requires ground truth outcomes (labels).
        """
        recent_cutoff = datetime.now() - timedelta(days=30)
        recent_predictions = [p for p in self.predictions_log
                            if p['timestamp'] > recent_cutoff
                            and p['actual_outcome'] is not None]

        if len(recent_predictions) < 100:
            return {'insufficient_data': True}

        y_true = np.array([p['actual_outcome'] for p in recent_predictions])
        y_pred_proba = np.array([p['predicted_probability'] for p in recent_predictions])

        from sklearn.metrics import roc_auc_score

        auroc = roc_auc_score(y_true, y_pred_proba)
        auroc_drop = (self.baseline_auroc - auroc) / self.baseline_auroc * 100

        return {
            'auroc': auroc,
            'auroc_drop_percent': auroc_drop,
            'performance_degradation': auroc_drop > 5.0,
            'sample_count': len(recent_predictions)
        }

    def check_retraining_triggers(self) -> List[str]:
        """Check all retraining trigger conditions."""
        triggers = []

        # Drift detection
        drift_info = self.detect_drift()
        if drift_info['drift_detected']:
            triggers.append(f"Data drift detected (PSI={drift_info['psi']:.3f})")

        # Performance degradation
        perf_info = self.evaluate_performance()
        if not perf_info.get('insufficient_data') and perf_info.get('performance_degradation'):
            triggers.append(f"Performance degradation: AUROC dropped {perf_info['auroc_drop_percent']:.1f}%")

        # Time-based (quarterly)
        # (check against model creation timestamp)
        triggers.append("Quarterly retraining cycle (90 days)")

        return triggers
```

### Retraining Pipeline

```
Retraining Triggers:
  1. Scheduled: Quarterly (90 days)
  2. Drift-based: PSI > 0.25
  3. Performance-based: AUROC drop > 5%
  4. Data-based: >10,000 new positive cases

Retraining Steps:
  1. Extract new data (since last training)
  2. Run complete data preparation pipeline
  3. Extract 70 features
  4. Handle class imbalance
  5. Train new models (baseline + tuned)
  6. Validate on held-out test set
  7. Compare to current production model
  8. If improved: proceed to deployment
  9. If not improved: investigate root cause

Retraining Frequency:
  - Baseline: Quarterly (every 90 days)
  - Drift-triggered: Monthly (if PSI > 0.25)
  - Emergency: Immediate (if AUROC drop > 10%)
```

---

## TROUBLESHOOTING GUIDE

### Common Issues and Solutions

#### Issue 1: Class Imbalance Problems

**Symptom**: Low precision or poor positive case identification

**Root Causes**:
- Insufficient positive cases (< 5% of dataset)
- Improper SMOTE application (applied to entire dataset)
- Incorrect class weight calculation

**Solutions**:
```python
# Check class distribution
print(f"Positive ratio: {y_train.mean():.2%}")

# If <5%, increase data collection window
# Or use SMOTE more aggressively:
smote = SMOTE(k_neighbors=3, sampling_strategy=0.6)  # Target 60% minority
X_resampled, y_resampled = smote.fit_resample(X_train, y_train)

# Verify SMOTE only applied to training set
assert len(X_val) == original_val_length  # Validation unchanged
```

#### Issue 2: Poor Model Calibration

**Symptom**: Predicted probabilities don't match actual rates

**Root Causes**:
- Class imbalance not addressed
- Improper threshold selection
- Model overconfident

**Solutions**:
```python
from sklearn.calibration import CalibratedClassifierCV

# Post-hoc calibration
calibrated_model = CalibratedClassifierCV(
    model,
    method='sigmoid',  # or 'isotonic'
    cv=5
)
calibrated_model.fit(X_val, y_val)

# Check improvement
brier_before = np.mean((y_val - model.predict_proba(X_val)[:, 1])**2)
brier_after = np.mean((y_val - calibrated_model.predict_proba(X_val)[:, 1])**2)
print(f"Brier score improvement: {brier_before:.3f} → {brier_after:.3f}")
```

#### Issue 3: High Inference Latency

**Symptom**: Inference takes >15ms per prediction

**Root Causes**:
- Model too complex (deep trees, many trees)
- Low thread count
- Batch size too small

**Solutions**:
```python
# Reduce model complexity
model = xgb.XGBClassifier(
    n_estimators=100,  # Reduce from 300
    max_depth=6,       # Reduce from 10
    n_jobs=-1
)

# Increase ONNX parallelism
config = ModelConfig.builder()
    .intraOpThreads(8)  # Increase parallelism
    .interOpThreads(4)
    .build()

# Use batch inference
predictions = model.predictBatch(patient_batch)  # Not individual calls
```

#### Issue 4: ONNX Model Validation Failure

**Symptom**: ONNX model doesn't match original model predictions

**Root Causes**:
- Incompatible XGBoost version
- Missing input preprocessing in ONNX
- Opset version mismatch

**Solutions**:
```python
# Check XGBoost version compatibility
print(xgb.__version__)  # Should be 1.5+

# Verify preprocessing matches
# Original: normalize features
# ONNX: must receive already-normalized input

# Debug prediction differences
original_pred = model.predict_proba(X_test[0:1])
onnx_pred = session.run([output_name], {input_name: X_test[0:1]})[0]

print(f"Original: {original_pred}")
print(f"ONNX:     {onnx_pred}")
print(f"Difference: {abs(original_pred - onnx_pred)}")
```

#### Issue 5: Feature Extraction Inconsistency

**Symptom**: Model performs well in training but poorly in production

**Root Causes**:
- Feature extraction logic differs between training and production
- Missing value imputation strategy not documented
- Normalization parameters not saved

**Solutions**:
```python
# Save normalization parameters
import joblib

# After training, save the normalizer
joblib.dump(normalizer, 'normalizer.pkl')

# In production, load and apply
normalizer = joblib.load('normalizer.pkl')
X_production = normalizer.transform(X_raw)

# Document feature order explicitly
FEATURE_ORDER = [
    'demo_age_years', 'demo_gender_male', ...  # All 70 features
]

# Validate feature completeness
for feature in FEATURE_ORDER:
    assert feature in extracted_dict, f"Missing feature: {feature}"
```

---

## APPENDIX

### A. Performance Targets Summary

| Model | Target AUROC | Min Sensitivity | Min Specificity | Max Latency |
|-------|--------------|-----------------|-----------------|-------------|
| Sepsis | 0.85+ | 0.80 | 0.80 | 15ms |
| Deterioration | 0.82+ | 0.75 | 0.80 | 15ms |
| Mortality | 0.80+ | 0.70 | 0.85 | 15ms |
| Readmission | 0.78+ | 0.65 | 0.85 | 15ms |

### B. Feature Requirements Checklist

- [ ] All 70 features extracted correctly
- [ ] Missing values imputed appropriately
- [ ] Features normalized consistently
- [ ] No data leakage (future information included)
- [ ] Categorical features encoded properly
- [ ] Temporal features computed from correct reference time
- [ ] Outliers handled (Winsorization or removal)
- [ ] Feature order matches documented specification

### C. Model Export Checklist

- [ ] Model trained with optimal hyperparameters
- [ ] Validated on hold-out test set
- [ ] AUROC ≥ target threshold
- [ ] Calibration checked (ECE < 0.10)
- [ ] Converted to ONNX successfully
- [ ] ONNX model validates with test data
- [ ] Inference latency <15ms p99
- [ ] Model versioned (v1.0.0, v1.1.0, etc.)

### D. Deployment Checklist

- [ ] A/B testing strategy defined
- [ ] Canary deployment parameters set (10% traffic)
- [ ] Rollback procedure documented
- [ ] Monitoring metrics configured
- [ ] Performance baseline established
- [ ] Clinical team sign-off obtained
- [ ] Production readiness confirmed
- [ ] Deployment time windows identified

### E. Useful Commands

```bash
# Check model file size
du -h sepsis_v1.0.0.onnx

# Validate ONNX syntax
python -c "import onnx; onnx.checker.check_model('sepsis_v1.0.0.onnx')"

# Extract model information
python scripts/onnx_info.py sepsis_v1.0.0.onnx

# Test inference latency
python scripts/benchmark_inference.py sepsis_v1.0.0.onnx --samples 1000

# Compare model versions
python scripts/compare_models.py sepsis_v1.0.0.onnx sepsis_v1.1.0.onnx
```

---

**Document End**

For questions or updates, contact the ML Engineering team.
