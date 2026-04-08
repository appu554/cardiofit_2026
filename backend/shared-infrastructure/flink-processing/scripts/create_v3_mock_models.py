#!/usr/bin/env python3
"""
Generate v3.0.0 mock ONNX models matching Module5FeatureExtractor's 55-feature layout.

These models:
- Accept exactly 55 float32 features (matching production Module5FeatureExtractor)
- Output XGBoost ONNX format: [labels(INT64), probabilities(FLOAT, [batch,2])]
- Are trained on synthetic data with clinically-weighted features (not random noise)
- Validate end-to-end ONNX integration: load → tensor → session.run() → output

They are NOT clinically useful — Phase 2 (MIMIC-IV extraction + training) produces real models.

Usage:
    python scripts/create_v3_mock_models.py

Output:
    models/sepsis/model.onnx
    models/deterioration/model.onnx
    models/mortality/model.onnx
    models/readmission/model.onnx
    models/fall/model.onnx
"""

import numpy as np
import xgboost as xgb
import onnx
import onnxmltools
from onnxmltools.convert import convert_xgboost
from onnxmltools.convert.common.data_types import FloatTensorType
import onnxruntime as ort
from datetime import datetime
import os
import sys

# ═══════════════════════════════════════════════════
# Feature layout matching Module5FeatureExtractor.java
# ═══════════════════════════════════════════════════
#
# [0-5]   Vital signs (normalized 0-1): HR, SBP, DBP, RR, SpO2, Temp
# [6-8]   Clinical scores (normalized 0-1): NEWS2, qSOFA, acuity
# [9-10]  Event context: log1p(eventCount), hoursSinceAdmission
# [11-20] NEWS2 history ring buffer (10 slots, normalized 0-1)
# [21-30] Acuity history ring buffer (10 slots, normalized 0-1)
# [31-34] Pattern features: patternCount, deteriorationCount, maxSeverity, escalation
# [35-44] Risk indicator flags (0/1): tachy, hypo, fever, hypoxia, lactate, creat, K, plt, anticoag, sepsis
# [45-49] Lab-derived features (normalized, -1=missing): lactate, creatinine, K, WBC, plt
# [50-54] Active alert features: sepsis_pattern, AKI_risk, anticoag_risk, bleeding_risk, maxAlertSeverity

FEATURE_COUNT = 55

FEATURE_NAMES = [
    # Vitals [0-5]
    "hr_norm", "sbp_norm", "dbp_norm", "rr_norm", "spo2_norm", "temp_norm",
    # Clinical scores [6-8]
    "news2_norm", "qsofa_norm", "acuity_norm",
    # Event context [9-10]
    "log_event_count", "hours_since_admission_norm",
    # NEWS2 history [11-20]
    *[f"news2_hist_{i}" for i in range(10)],
    # Acuity history [21-30]
    *[f"acuity_hist_{i}" for i in range(10)],
    # Pattern features [31-34]
    "pattern_count", "deterioration_pattern_count", "max_severity_index", "severity_escalation",
    # Risk flags [35-44]
    "tachycardia", "hypotension", "fever", "hypoxia",
    "elevated_lactate", "elevated_creatinine", "hyperkalemia",
    "thrombocytopenia", "on_anticoagulation", "sepsis_risk",
    # Labs [45-49]
    "lab_lactate", "lab_creatinine", "lab_potassium", "lab_wbc", "lab_platelets",
    # Alerts [50-54]
    "alert_sepsis", "alert_aki", "alert_anticoag", "alert_bleeding", "alert_max_severity",
]

assert len(FEATURE_NAMES) == FEATURE_COUNT, f"Expected {FEATURE_COUNT} features, got {len(FEATURE_NAMES)}"

# Model configurations with clinically meaningful synthetic data generation
MODEL_CONFIGS = {
    "sepsis": {
        "description": "Sepsis onset prediction (6-12h horizon)",
        "positive_rate": 0.08,
        # Features most informative for sepsis (higher weight in synthetic generation)
        "informative_indices": [0, 3, 5, 6, 7, 39, 44, 45, 48, 50],
        "n_estimators": 80,
        "max_depth": 5,
    },
    "deterioration": {
        "description": "Clinical deterioration prediction (6-24h window)",
        "positive_rate": 0.06,
        "informative_indices": [0, 1, 3, 4, 6, 7, 8, 32, 34, 35, 36],
        "n_estimators": 80,
        "max_depth": 5,
    },
    "mortality": {
        "description": "In-hospital mortality prediction",
        "positive_rate": 0.04,
        "informative_indices": [6, 7, 8, 33, 36, 38, 40, 45, 46, 54],
        "n_estimators": 80,
        "max_depth": 5,
    },
    "readmission": {
        "description": "30-day readmission prediction",
        "positive_rate": 0.10,
        "informative_indices": [8, 10, 31, 33, 40, 42, 46, 49, 51, 54],
        "n_estimators": 80,
        "max_depth": 5,
    },
    "fall": {
        "description": "Inpatient fall risk prediction",
        "positive_rate": 0.05,
        "informative_indices": [0, 1, 2, 6, 10, 35, 36, 43, 46, 54],
        "n_estimators": 60,
        "max_depth": 4,
    },
}


def generate_synthetic_data(config, n_samples=5000, seed=42):
    """
    Generate synthetic training data with clinically-weighted features.

    Not random noise — the informative_indices features are correlated
    with the label, so the model learns which features matter.
    """
    rng = np.random.RandomState(seed)

    positive_rate = config["positive_rate"]
    informative = config["informative_indices"]

    n_positive = int(n_samples * positive_rate)
    n_negative = n_samples - n_positive

    # Generate base features (uniform 0-1 for normalized features)
    X = rng.rand(n_samples, FEATURE_COUNT).astype(np.float32)

    # Binary flags [35-44] and [50-54] should be 0/1
    for i in list(range(35, 45)) + list(range(50, 54)):
        X[:, i] = (rng.rand(n_samples) > 0.7).astype(np.float32)

    # Labs [45-49] can be -1 (missing) ~20% of the time
    for i in range(45, 50):
        missing_mask = rng.rand(n_samples) < 0.2
        X[missing_mask, i] = -1.0

    # Create labels
    y = np.zeros(n_samples, dtype=np.int32)
    y[:n_positive] = 1

    # Shift informative features for positive cases to create signal
    for idx in informative:
        if idx < 35 or (45 <= idx <= 49):
            # Continuous features: shift positive cases higher
            X[:n_positive, idx] += rng.rand(n_positive) * 0.3 + 0.2
            X[:n_positive, idx] = np.clip(X[:n_positive, idx], 0, 1)
        else:
            # Binary features: higher probability for positive cases
            X[:n_positive, idx] = (rng.rand(n_positive) > 0.3).astype(np.float32)

    # Shuffle
    perm = rng.permutation(n_samples)
    X = X[perm]
    y = y[perm]

    return X, y


def train_and_export(category, config, output_dir="models"):
    """Train XGBoost model and export to ONNX."""
    print(f"\n{'─'*60}")
    print(f"  {category.upper()}: {config['description']}")
    print(f"{'─'*60}")

    # Generate data
    seed = hash(category) % 10000
    X, y = generate_synthetic_data(config, n_samples=5000, seed=seed)

    pos_weight = (y == 0).sum() / max((y == 1).sum(), 1)
    print(f"  Samples: {len(X):,} ({y.sum():,} positive, {pos_weight:.1f}x weight)")

    # Train XGBoost
    model = xgb.XGBClassifier(
        n_estimators=config["n_estimators"],
        max_depth=config["max_depth"],
        learning_rate=0.1,
        subsample=0.8,
        colsample_bytree=0.8,
        scale_pos_weight=pos_weight,
        random_state=42,
        eval_metric="logloss",
        n_jobs=-1,
    )
    model.fit(X, y, verbose=False)

    # Convert to ONNX
    initial_types = [("float_input", FloatTensorType([None, FEATURE_COUNT]))]
    onnx_model = convert_xgboost(model, initial_types=initial_types, target_opset=12)

    # Metadata
    onnx_model.producer_name = "CardioFit-Module5-v3"
    onnx_model.producer_version = "3.0.0"
    onnx_model.doc_string = config["description"]

    for key, value in {
        "model_name": category,
        "version": "3.0.0",
        "input_features": str(FEATURE_COUNT),
        "feature_layout": "Module5FeatureExtractor-55",
        "output_type": "xgboost_binary_classification",
        "is_mock_model": "true",
        "training_data": "synthetic_clinically_weighted",
        "created_date": datetime.now().strftime("%Y-%m-%d"),
    }.items():
        meta = onnx_model.metadata_props.add()
        meta.key = key
        meta.value = value

    # Validate ONNX structure
    onnx.checker.check_model(onnx_model)

    # Save to {category}/model.onnx (matching Module5_MLInferenceEngine.open() path)
    model_dir = os.path.join(output_dir, category)
    os.makedirs(model_dir, exist_ok=True)
    output_path = os.path.join(model_dir, "model.onnx")
    onnx.save(onnx_model, output_path)

    # Validate with ONNX Runtime
    session = ort.InferenceSession(output_path)
    inp = session.get_inputs()[0]
    outs = session.get_outputs()

    assert inp.shape == [None, FEATURE_COUNT], f"Input shape mismatch: {inp.shape}"
    assert len(outs) == 2, f"Expected 2 outputs (labels, probs), got {len(outs)}"

    # Test inference
    test_input = np.random.randn(1, FEATURE_COUNT).astype(np.float32)
    results = session.run(None, {inp.name: test_input})
    labels = results[0]
    probs = results[1]

    assert probs.shape == (1, 2), f"Probability shape mismatch: {probs.shape}"
    assert np.allclose(probs.sum(axis=1), 1.0, atol=0.01), "Probabilities don't sum to 1"

    file_size_kb = os.path.getsize(output_path) / 1024
    print(f"  Output: {output_path} ({file_size_kb:.0f} KB)")
    print(f"  Input:  {inp.name} shape={inp.shape} ({inp.type})")
    print(f"  Out[0]: {outs[0].name} shape={outs[0].shape} ({outs[0].type}) — labels")
    print(f"  Out[1]: {outs[1].name} shape={outs[1].shape} ({outs[1].type}) — probabilities")
    print(f"  Test:   label={labels[0]}, prob=[{probs[0][0]:.4f}, {probs[0][1]:.4f}]")
    print(f"  ✅ ONNX Runtime validation PASSED")

    return output_path


def main():
    print("=" * 60)
    print("MODULE 5 v3.0.0 MOCK MODEL GENERATOR")
    print("55-feature layout matching Module5FeatureExtractor")
    print("=" * 60)

    output_dir = "models"
    generated = []

    for category, config in MODEL_CONFIGS.items():
        try:
            path = train_and_export(category, config, output_dir)
            generated.append((category, path))
        except Exception as e:
            print(f"  ❌ {category}: {e}")
            import traceback
            traceback.print_exc()

    print(f"\n{'=' * 60}")
    print(f"✅ Generated {len(generated)}/{len(MODEL_CONFIGS)} models")
    print("=" * 60)

    for cat, path in generated:
        print(f"  {cat:20s} → {path}")

    print(f"\nDirectory layout (Module5_MLInferenceEngine.open() compatible):")
    print(f"  ML_MODEL_PATH={os.path.abspath(output_dir)}")
    print(f"  {{ML_MODEL_PATH}}/{{category}}/model.onnx")
    print()
    print("Next: mvn test -Dtest=Module5IntegrationTest")


if __name__ == "__main__":
    main()
