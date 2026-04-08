#!/usr/bin/env python3
"""
Train v3.0.0 ONNX models on MIMIC-IV 55-feature data.

Reads 55-feature CSVs from extract_mimic_features_v3.py and trains
XGBoost binary classifiers for each clinical prediction category.
Exports to ONNX format compatible with ONNXModelContainer.java.

Output:
    models/{category}/model.onnx     — ONNX model (replaces mock models)
    results/mimic_iv/v3_{category}_metrics.json  — Performance metrics
    results/mimic_iv/figures/v3_{category}_performance.png — ROC/PR/calibration plots

Usage:
    python scripts/train_mimic_models_v3.py
    python scripts/train_mimic_models_v3.py --cohorts sepsis mortality
    python scripts/train_mimic_models_v3.py --skip-plots
"""

import os
import sys
import json
import argparse
from pathlib import Path
from datetime import datetime

import numpy as np
import pandas as pd
import xgboost as xgb
from sklearn.model_selection import train_test_split
from sklearn.metrics import (
    roc_auc_score, roc_curve,
    precision_recall_curve, average_precision_score,
    confusion_matrix, classification_report,
)
from sklearn.calibration import calibration_curve
import onnx
import onnxmltools
from onnxmltools.convert import convert_xgboost
from onnxmltools.convert.common.data_types import FloatTensorType
import onnxruntime as ort

try:
    import matplotlib
    matplotlib.use("Agg")
    import matplotlib.pyplot as plt
    HAS_MATPLOTLIB = True
except ImportError:
    HAS_MATPLOTLIB = False

FEATURE_COUNT = 55
DATA_DIR = Path("data/mimic_iv")
MODELS_DIR = Path("models")
RESULTS_DIR = Path("results/mimic_iv")

# ═══════════════════════════════════════════════════════════════
# Cohort configuration
# ═══════════════════════════════════════════════════════════════
COHORTS = {
    "sepsis": {
        "data_file": "v3_sepsis_55f.csv",
        "label_col": "sepsis_label",
        "description": "Sepsis onset prediction (6-12h horizon)",
        "positive_rate_expected": 0.50,
    },
    "deterioration": {
        "data_file": "v3_deterioration_55f.csv",
        "label_col": "deterioration_label",
        "description": "Clinical deterioration prediction (6-24h window)",
        "positive_rate_expected": 0.50,
    },
    "mortality": {
        "data_file": "v3_mortality_55f.csv",
        "label_col": "mortality_label",
        "description": "In-hospital mortality prediction",
        "positive_rate_expected": 0.50,
    },
}

# XGBoost hyperparameters
XGBOOST_PARAMS = {
    "n_estimators": 200,
    "max_depth": 6,
    "learning_rate": 0.1,
    "subsample": 0.8,
    "colsample_bytree": 0.8,
    "min_child_weight": 1,
    "gamma": 0,
    "reg_alpha": 0,
    "reg_lambda": 1,
    "random_state": 42,
    "eval_metric": "logloss",
    "n_jobs": -1,
}

# Performance thresholds
MIN_AUROC = 0.70       # Lower than v2 (0.85) because fallback mode has zero-filled features
MIN_SENSITIVITY = 0.65
MIN_SPECIFICITY = 0.60

# Feature names (must match extract_mimic_features_v3.py)
FEATURE_NAMES = [
    "hr_norm", "sbp_norm", "dbp_norm", "rr_norm", "spo2_norm", "temp_norm",
    "news2_norm", "qsofa_norm", "acuity_norm",
    "log_event_count", "hours_since_admission_norm",
    *[f"news2_hist_{i}" for i in range(10)],
    *[f"acuity_hist_{i}" for i in range(10)],
    "pattern_count", "deterioration_pattern_count", "max_severity_index", "severity_escalation",
    "tachycardia", "hypotension", "fever", "hypoxia",
    "elevated_lactate", "elevated_creatinine", "hyperkalemia",
    "thrombocytopenia", "on_anticoagulation", "sepsis_risk",
    "lab_lactate", "lab_creatinine", "lab_potassium", "lab_wbc", "lab_platelets",
    "alert_sepsis", "alert_aki", "alert_anticoag", "alert_bleeding", "alert_max_severity",
]

assert len(FEATURE_NAMES) == FEATURE_COUNT


def train_cohort(cohort_name, config, skip_plots=False):
    """Train a single cohort model and export to ONNX."""
    print(f"\n{'='*60}")
    print(f"  TRAINING: {cohort_name.upper()}")
    print(f"  {config['description']}")
    print(f"{'='*60}")

    # ── Load data ──
    data_path = DATA_DIR / config["data_file"]
    if not data_path.exists():
        print(f"  ERROR: {data_path} not found. Run extract_mimic_features_v3.py first.")
        return None

    df = pd.read_csv(data_path)
    label_col = config["label_col"]

    X = df[FEATURE_NAMES].values.astype(np.float32)
    y = df[label_col].values.astype(np.int32)

    print(f"  Data: {len(X):,} patients, {X.shape[1]} features")
    print(f"  Label distribution: {(y==1).sum():,} positive ({y.mean()*100:.1f}%), "
          f"{(y==0).sum():,} negative ({(1-y.mean())*100:.1f}%)")

    # ── Train/val/test split (70/15/15) ──
    X_trainval, X_test, y_trainval, y_test = train_test_split(
        X, y, test_size=0.15, stratify=y, random_state=42
    )
    X_train, X_val, y_train, y_val = train_test_split(
        X_trainval, y_trainval, test_size=0.176, stratify=y_trainval, random_state=42
    )

    print(f"  Split: train={len(X_train):,}, val={len(X_val):,}, test={len(X_test):,}")

    # ── Train XGBoost ──
    pos_weight = (y_train == 0).sum() / max((y_train == 1).sum(), 1)
    params = XGBOOST_PARAMS.copy()
    params["scale_pos_weight"] = pos_weight

    model = xgb.XGBClassifier(**params)
    model.fit(
        X_train, y_train,
        eval_set=[(X_train, y_train), (X_val, y_val)],
        verbose=False,
    )
    print(f"  Trained: {model.n_estimators} trees, depth={model.max_depth}")

    # ── Evaluate ──
    metrics = {}
    for split_name, X_split, y_split in [
        ("val", X_val, y_val),
        ("test", X_test, y_test),
    ]:
        y_proba = model.predict_proba(X_split)[:, 1]
        y_pred = (y_proba >= 0.5).astype(int)

        auroc = roc_auc_score(y_split, y_proba)
        auprc = average_precision_score(y_split, y_proba)
        tn, fp, fn, tp = confusion_matrix(y_split, y_pred).ravel()
        sensitivity = tp / (tp + fn) if (tp + fn) > 0 else 0
        specificity = tn / (tn + fp) if (tn + fp) > 0 else 0
        ppv = tp / (tp + fp) if (tp + fp) > 0 else 0
        npv = tn / (tn + fn) if (tn + fn) > 0 else 0

        metrics[split_name] = {
            "auroc": round(auroc, 4),
            "auprc": round(auprc, 4),
            "sensitivity": round(sensitivity, 4),
            "specificity": round(specificity, 4),
            "ppv": round(ppv, 4),
            "npv": round(npv, 4),
            "tp": int(tp), "fp": int(fp), "fn": int(fn), "tn": int(tn),
        }

        print(f"\n  {split_name.upper()} SET:")
        print(f"    AUROC: {auroc:.4f}  AUPRC: {auprc:.4f}")
        print(f"    Sensitivity: {sensitivity:.4f}  Specificity: {specificity:.4f}")
        print(f"    PPV: {ppv:.4f}  NPV: {npv:.4f}")

        # Check thresholds
        if split_name == "test":
            warnings = []
            if auroc < MIN_AUROC:
                warnings.append(f"AUROC {auroc:.4f} < {MIN_AUROC}")
            if sensitivity < MIN_SENSITIVITY:
                warnings.append(f"Sensitivity {sensitivity:.4f} < {MIN_SENSITIVITY}")
            if specificity < MIN_SPECIFICITY:
                warnings.append(f"Specificity {specificity:.4f} < {MIN_SPECIFICITY}")
            if warnings:
                print(f"    WARNINGS: {', '.join(warnings)}")
            else:
                print(f"    All thresholds passed")

    # ── Feature importance ──
    importance = model.feature_importances_
    top_indices = np.argsort(importance)[-10:][::-1]
    print(f"\n  Top 10 features:")
    for idx in top_indices:
        print(f"    [{idx:2d}] {FEATURE_NAMES[idx]:35s} importance={importance[idx]:.4f}")

    # ── Platt scaling parameters ──
    # Fit sigmoid calibration on validation set
    y_val_proba = model.predict_proba(X_val)[:, 1]
    from scipy.optimize import minimize

    def platt_loss(params, y_true, y_pred):
        A, B = params
        p = 1.0 / (1.0 + np.exp(A * y_pred + B))
        p = np.clip(p, 1e-10, 1 - 1e-10)
        return -np.mean(y_true * np.log(p) + (1 - y_true) * np.log(1 - p))

    result = minimize(platt_loss, x0=[-1.0, 0.0], args=(y_val, y_val_proba),
                      method="Nelder-Mead")
    platt_A, platt_B = result.x
    print(f"\n  Platt scaling: A={platt_A:.4f}, B={platt_B:.4f}")
    metrics["platt_params"] = {"A": round(platt_A, 4), "B": round(platt_B, 4)}

    # ── Export to ONNX ──
    print(f"\n  Exporting to ONNX...")
    initial_types = [("float_input", FloatTensorType([None, FEATURE_COUNT]))]
    onnx_model = convert_xgboost(model, initial_types=initial_types, target_opset=12)

    # Metadata
    onnx_model.producer_name = "CardioFit-Module5-v3-MIMIC"
    onnx_model.producer_version = "3.0.0"
    onnx_model.doc_string = config["description"]

    for key, value in {
        "model_name": cohort_name,
        "version": "3.0.0",
        "input_features": str(FEATURE_COUNT),
        "feature_layout": "Module5FeatureExtractor-55",
        "output_type": "xgboost_binary_classification",
        "training_data": "mimic_iv_v3.1",
        "training_mode": "fallback_37_to_55_mapping",
        "training_samples": str(len(X_train)),
        "test_auroc": str(metrics["test"]["auroc"]),
        "test_sensitivity": str(metrics["test"]["sensitivity"]),
        "platt_A": str(platt_A),
        "platt_B": str(platt_B),
        "created_date": datetime.now().strftime("%Y-%m-%d"),
        "is_mock_model": "false",
    }.items():
        meta = onnx_model.metadata_props.add()
        meta.key = key
        meta.value = value

    onnx.checker.check_model(onnx_model)

    # Save
    model_dir = MODELS_DIR / cohort_name
    model_dir.mkdir(parents=True, exist_ok=True)
    output_path = model_dir / "model.onnx"
    onnx.save(onnx_model, str(output_path))

    # Validate with ONNX Runtime
    session = ort.InferenceSession(str(output_path))
    inp = session.get_inputs()[0]
    outs = session.get_outputs()

    assert inp.shape == [None, FEATURE_COUNT], f"Input shape mismatch: {inp.shape}"
    assert len(outs) == 2, f"Expected 2 outputs, got {len(outs)}"

    # Test inference
    test_input = X_test[:1]
    results = session.run(None, {inp.name: test_input})
    probs = results[1]
    assert probs.shape == (1, 2), f"Prob shape mismatch: {probs.shape}"
    assert np.allclose(probs.sum(axis=1), 1.0, atol=0.01), "Probs don't sum to 1"

    file_size_kb = output_path.stat().st_size / 1024
    print(f"  Saved: {output_path} ({file_size_kb:.0f} KB)")
    print(f"  Input: {inp.name} shape={inp.shape}")
    print(f"  Output[0]: labels, Output[1]: probabilities")
    print(f"  ONNX Runtime validation: PASSED")

    # ── Save metrics ──
    RESULTS_DIR.mkdir(parents=True, exist_ok=True)
    metrics_path = RESULTS_DIR / f"v3_{cohort_name}_metrics.json"
    metrics["model_info"] = {
        "cohort": cohort_name,
        "version": "3.0.0",
        "feature_count": FEATURE_COUNT,
        "training_samples": len(X_train),
        "validation_samples": len(X_val),
        "test_samples": len(X_test),
        "n_estimators": model.n_estimators,
        "max_depth": model.max_depth,
        "model_file": str(output_path),
        "model_size_kb": round(file_size_kb, 1),
        "created": datetime.now().isoformat(),
    }
    with open(metrics_path, "w") as f:
        json.dump(metrics, f, indent=2)
    print(f"  Metrics: {metrics_path}")

    # ── Plot performance ──
    if not skip_plots and HAS_MATPLOTLIB:
        plot_performance(cohort_name, model, X_test, y_test, X_val, y_val, metrics)

    return metrics


def plot_performance(cohort_name, model, X_test, y_test, X_val, y_val, metrics):
    """Generate ROC, PR, calibration, and feature importance plots."""
    fig_dir = RESULTS_DIR / "figures"
    fig_dir.mkdir(parents=True, exist_ok=True)

    fig, axes = plt.subplots(2, 2, figsize=(12, 10))
    fig.suptitle(f"{cohort_name.capitalize()} v3.0.0 — MIMIC-IV Trained", fontsize=14)

    for split_name, X_s, y_s, color in [
        ("val", X_val, y_val, "blue"),
        ("test", X_test, y_test, "red"),
    ]:
        y_proba = model.predict_proba(X_s)[:, 1]

        # ROC
        fpr, tpr, _ = roc_curve(y_s, y_proba)
        auroc = metrics[split_name]["auroc"]
        axes[0, 0].plot(fpr, tpr, color=color, label=f"{split_name} (AUC={auroc:.3f})")

        # PR
        precision, recall, _ = precision_recall_curve(y_s, y_proba)
        auprc = metrics[split_name]["auprc"]
        axes[0, 1].plot(recall, precision, color=color, label=f"{split_name} (AP={auprc:.3f})")

    axes[0, 0].plot([0, 1], [0, 1], "k--", alpha=0.3)
    axes[0, 0].set_xlabel("FPR")
    axes[0, 0].set_ylabel("TPR")
    axes[0, 0].set_title("ROC Curve")
    axes[0, 0].legend()
    axes[0, 0].grid(True, alpha=0.3)

    axes[0, 1].set_xlabel("Recall")
    axes[0, 1].set_ylabel("Precision")
    axes[0, 1].set_title("Precision-Recall Curve")
    axes[0, 1].legend()
    axes[0, 1].grid(True, alpha=0.3)

    # Calibration
    y_test_proba = model.predict_proba(X_test)[:, 1]
    prob_true, prob_pred = calibration_curve(y_test, y_test_proba, n_bins=10)
    axes[1, 0].plot(prob_pred, prob_true, "o-", label="Model")
    axes[1, 0].plot([0, 1], [0, 1], "k--", alpha=0.3, label="Perfect")
    axes[1, 0].set_xlabel("Mean Predicted Probability")
    axes[1, 0].set_ylabel("Fraction of Positives")
    axes[1, 0].set_title("Calibration Plot")
    axes[1, 0].legend()
    axes[1, 0].grid(True, alpha=0.3)

    # Feature importance (top 15)
    importance = model.feature_importances_
    top_idx = np.argsort(importance)[-15:]
    axes[1, 1].barh(range(len(top_idx)), importance[top_idx])
    axes[1, 1].set_yticks(range(len(top_idx)))
    axes[1, 1].set_yticklabels([FEATURE_NAMES[i] for i in top_idx], fontsize=7)
    axes[1, 1].set_xlabel("Importance")
    axes[1, 1].set_title("Top 15 Features")
    axes[1, 1].grid(True, alpha=0.3, axis="x")

    plt.tight_layout()
    fig_path = fig_dir / f"v3_{cohort_name}_performance.png"
    plt.savefig(fig_path, dpi=150, bbox_inches="tight")
    plt.close()
    print(f"  Plot: {fig_path}")


def main():
    parser = argparse.ArgumentParser(description="Train v3.0.0 ONNX models on MIMIC-IV 55-feature data")
    parser.add_argument("--cohorts", nargs="+", choices=list(COHORTS.keys()),
                        default=list(COHORTS.keys()), help="Which cohorts to train")
    parser.add_argument("--skip-plots", action="store_true", help="Skip performance plots")
    args = parser.parse_args()

    print("=" * 60)
    print("MODULE 5 v3.0.0 MODEL TRAINING — MIMIC-IV")
    print(f"Cohorts: {', '.join(args.cohorts)}")
    print(f"Features: {FEATURE_COUNT}")
    print(f"Output: {MODELS_DIR}/{{category}}/model.onnx")
    print("=" * 60)

    trained = []

    for cohort_name in args.cohorts:
        config = COHORTS[cohort_name]
        metrics = train_cohort(cohort_name, config, skip_plots=args.skip_plots)
        if metrics:
            trained.append((cohort_name, metrics))

    # ── Summary ──
    print(f"\n{'='*60}")
    print(f"TRAINING COMPLETE: {len(trained)}/{len(args.cohorts)} models")
    print("=" * 60)

    print(f"\n{'Model':<20s} {'AUROC':>8s} {'Sens':>8s} {'Spec':>8s} {'AUPRC':>8s} {'Size':>8s}")
    print("-" * 56)
    for name, m in trained:
        t = m["test"]
        size_kb = m["model_info"]["model_size_kb"]
        print(f"  {name:<18s} {t['auroc']:>7.4f} {t['sensitivity']:>7.4f} "
              f"{t['specificity']:>7.4f} {t['auprc']:>7.4f} {size_kb:>6.0f}KB")

    print(f"\nPlatt calibration parameters (update Module5ClinicalScoring.java):")
    for name, m in trained:
        p = m["platt_params"]
        print(f'  "{name}": new double[]{{ {p["A"]}, {p["B"]} }},')

    print(f"\nNext steps:")
    print(f"  1. Run ONNX integration test: mvn test -Dtest=Module5OnnxIntegrationTest")
    print(f"  2. Update Platt params in Module5ClinicalScoring.java")
    print(f"  3. When PhysioNet download works, re-extract with --mode raw for full 55 features")


if __name__ == "__main__":
    main()
