#!/usr/bin/env python3
"""Weekly retraining pipeline: retrain all classifiers and compare metrics.

1. Loads latest golden_dataset_enriched.parquet
2. Re-splits with golden_dataset_splits.json strategy
3. Retrains noise gate, tier assigner, safety criticality classifiers
4. Evaluates on held-out test set
5. Compares metrics to current production models
6. Runs feature drift analysis
7. Outputs retrain_report.json

Usage:
    python weekly_retrain.py
    python weekly_retrain.py --input golden_dataset_enriched.parquet --models-dir models/

Cron example (run weekly on Sunday at 3am):
    0 3 * * 0 cd /path/to/guideline-atomiser/data && python weekly_retrain.py
"""

from __future__ import annotations

import argparse
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

import joblib
import numpy as np
import pandas as pd
from sklearn.metrics import precision_recall_fscore_support, classification_report
from sklearn.model_selection import StratifiedShuffleSplit
from xgboost import XGBClassifier
from sklearn.linear_model import LogisticRegression
from sklearn.preprocessing import LabelEncoder

_PROJECT_ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(_PROJECT_ROOT))

from extraction.v4.classifiers.noise_gate import NOISE_GATE_FEATURES
from extraction.v4.classifiers.tier_assigner import TIER_ASSIGNER_FEATURES
from extraction.v4.classifiers.safety_criticality import SAFETY_FEATURES
from extraction.v4.classifiers.drift_monitor import DriftMonitor


def _derive_tier_label(row) -> str:
    if row["reviewer_action"] == "REJECT":
        return "NOISE"
    tier = row.get("tier", "TIER_2")
    return tier if tier in ("TIER_1", "TIER_2") else "TIER_2"


def retrain_noise_gate(X_train, y_train, X_val, y_val, X_test, y_test) -> dict:
    """Retrain noise gate and return metrics."""
    model = XGBClassifier(
        n_estimators=200, max_depth=6, learning_rate=0.1, min_child_weight=5,
        scale_pos_weight=(1 - y_train.mean()) / max(y_train.mean(), 0.01),
        eval_metric="logloss", early_stopping_rounds=20, random_state=42,
    )
    model.fit(X_train, y_train, eval_set=[(X_val, y_val)], verbose=False)
    y_pred = model.predict(X_test)
    p, r, f1, _ = precision_recall_fscore_support(y_test, y_pred, average="binary")
    return {"model": model, "precision": float(p), "recall": float(r), "f1": float(f1)}


def retrain_tier_assigner(X_train, y_train, X_val, y_val, X_test, y_test, le) -> dict:
    """Retrain tier assigner and return metrics."""
    model = XGBClassifier(
        n_estimators=300, max_depth=6, learning_rate=0.1, min_child_weight=3,
        objective="multi:softprob", num_class=len(le.classes_),
        eval_metric="mlogloss", early_stopping_rounds=20, random_state=42,
    )
    model.fit(X_train, y_train, eval_set=[(X_val, y_val)], verbose=False)
    y_pred = model.predict(X_test)
    p, r, f1, _ = precision_recall_fscore_support(y_test, y_pred, average="weighted")
    return {"model": model, "precision": float(p), "recall": float(r), "f1": float(f1)}


def retrain_safety(X_train, y_train, X_test, y_test) -> dict:
    """Retrain safety criticality detector and return metrics."""
    model = LogisticRegression(max_iter=1000, class_weight="balanced", random_state=42)
    model.fit(X_train, y_train)
    y_pred = model.predict(X_test)
    p, r, f1, _ = precision_recall_fscore_support(y_test, y_pred, average="binary")
    return {"model": model, "precision": float(p), "recall": float(r), "f1": float(f1)}


def main():
    parser = argparse.ArgumentParser(description="Weekly classifier retraining")
    parser.add_argument("--input", "-i", default="golden_dataset_enriched.parquet")
    parser.add_argument("--splits", "-s", default="golden_dataset_splits.json")
    parser.add_argument("--models-dir", "-m", default="models/")
    parser.add_argument("--report", "-r", default="retrain_report.json")
    args = parser.parse_args()

    models_dir = Path(args.models_dir)
    models_dir.mkdir(parents=True, exist_ok=True)

    df = pd.read_parquet(args.input)
    with open(args.splits) as f:
        splits = json.load(f)

    print(f"[{datetime.now(timezone.utc).isoformat()}] Weekly retraining: {len(df)} spans")

    train_idx = splits["train_indices"]
    val_idx = splits["val_indices"]
    test_idx = splits["test_indices"]

    report = {"timestamp": datetime.now(timezone.utc).isoformat(), "total_spans": len(df)}

    # ── Noise Gate ──────────────────────────────────────────────
    print("\n1. Retraining noise gate...")
    X_ng = df[NOISE_GATE_FEATURES].fillna(0).values
    y_ng = df["is_noise"].astype(int).values
    ng_result = retrain_noise_gate(
        X_ng[train_idx], y_ng[train_idx],
        X_ng[val_idx], y_ng[val_idx],
        X_ng[test_idx], y_ng[test_idx],
    )
    joblib.dump(ng_result["model"], models_dir / "noise_gate.joblib")
    report["noise_gate"] = {k: v for k, v in ng_result.items() if k != "model"}
    print(f"   Noise gate F1: {ng_result['f1']:.3f}")

    # ── Tier Assigner ───────────────────────────────────────────
    print("2. Retraining tier assigner...")
    df["training_label"] = df.apply(_derive_tier_label, axis=1)
    le = LabelEncoder()
    df["label_encoded"] = le.fit_transform(df["training_label"])
    X_ta = df[TIER_ASSIGNER_FEATURES].fillna(0).values
    y_ta = df["label_encoded"].values
    ta_result = retrain_tier_assigner(
        X_ta[train_idx], y_ta[train_idx],
        X_ta[val_idx], y_ta[val_idx],
        X_ta[test_idx], y_ta[test_idx],
        le,
    )
    joblib.dump(ta_result["model"], models_dir / "tier_assigner.joblib")
    joblib.dump(le, models_dir / "tier_assigner_encoder.joblib")
    report["tier_assigner"] = {k: v for k, v in ta_result.items() if k != "model"}
    print(f"   Tier assigner weighted F1: {ta_result['f1']:.3f}")

    # ── Safety Criticality ──────────────────────────────────────
    print("3. Retraining safety criticality detector...")
    X_sc = df[SAFETY_FEATURES].fillna(0).values
    y_sc = df.get("is_safety_critical", df["has_safety_keyword"]).astype(int).values
    from sklearn.model_selection import train_test_split
    X_sc_train, X_sc_test, y_sc_train, y_sc_test = train_test_split(
        X_sc, y_sc, test_size=0.2, random_state=42, stratify=y_sc
    )
    sc_result = retrain_safety(X_sc_train, y_sc_train, X_sc_test, y_sc_test)
    joblib.dump(sc_result["model"], models_dir / "safety_criticality.joblib")
    report["safety_criticality"] = {k: v for k, v in sc_result.items() if k != "model"}
    print(f"   Safety criticality F1: {sc_result['f1']:.3f}")

    # ── Drift Analysis ──────────────────────────────────────────
    print("4. Running drift analysis...")
    monitor = DriftMonitor()
    # Compare train vs test feature distributions as a basic check
    training_features = {f: X_ng[train_idx, i] for i, f in enumerate(NOISE_GATE_FEATURES)}
    recent_features = {f: X_ng[test_idx, i] for i, f in enumerate(NOISE_GATE_FEATURES)}
    drift_report = monitor.analyze(training_features, recent_features)
    report["drift"] = drift_report.to_dict()
    print(f"   Drift: max PSI={drift_report.overall_max_psi:.4f}, "
          f"recommendation={drift_report.recommendation}")

    # ── Compare with production models ──────────────────────────
    print("5. Comparing with production models...")
    for model_name in ["noise_gate", "tier_assigner", "safety_criticality"]:
        prod_metrics_path = models_dir / f"{model_name}.metrics.json"
        if prod_metrics_path.exists():
            with open(prod_metrics_path) as f:
                prod_metrics = json.load(f)
            new_f1 = report[model_name]["f1"]
            prod_f1 = prod_metrics.get("f1", 0)
            delta = new_f1 - prod_f1
            report[model_name]["production_f1"] = prod_f1
            report[model_name]["f1_delta"] = delta
            print(f"   {model_name}: F1 {prod_f1:.3f} → {new_f1:.3f} (Δ={delta:+.3f})")

    # Save report
    with open(args.report, "w") as f:
        json.dump(report, f, indent=2)
    print(f"\nReport saved to {args.report}")


if __name__ == "__main__":
    main()
