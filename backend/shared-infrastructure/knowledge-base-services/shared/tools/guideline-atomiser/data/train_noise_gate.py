#!/usr/bin/env python3
"""Train the noise gate classifier (binary XGBoost).

Reads the enriched golden dataset, trains a binary noise/signal classifier,
evaluates on the held-out test set, and saves the model.

Usage:
    python train_noise_gate.py \
        --input golden_dataset_enriched.parquet \
        --splits golden_dataset_splits.json \
        --output models/noise_gate.joblib

Requires:
    pip install pandas xgboost scikit-learn joblib
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

import joblib
import numpy as np
import pandas as pd
from sklearn.metrics import classification_report, precision_recall_fscore_support
from xgboost import XGBClassifier

# Add project root to path
_PROJECT_ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(_PROJECT_ROOT))

from extraction.v4.classifiers.noise_gate import NOISE_GATE_FEATURES


def main():
    parser = argparse.ArgumentParser(description="Train noise gate classifier")
    parser.add_argument("--input", "-i", default="golden_dataset_enriched.parquet")
    parser.add_argument("--splits", "-s", default="golden_dataset_splits.json")
    parser.add_argument("--output", "-o", default="models/noise_gate.joblib")
    args = parser.parse_args()

    # Load data
    df = pd.read_parquet(args.input)
    with open(args.splits) as f:
        splits = json.load(f)

    print(f"Loaded {len(df)} spans from {args.input}")

    # Prepare features and target
    X = df[NOISE_GATE_FEATURES].fillna(0).values
    y = df["is_noise"].astype(int).values

    # Split
    train_idx = splits["train_indices"]
    val_idx = splits["val_indices"]
    test_idx = splits["test_indices"]

    X_train, y_train = X[train_idx], y[train_idx]
    X_val, y_val = X[val_idx], y[val_idx]
    X_test, y_test = X[test_idx], y[test_idx]

    print(f"\nTrain: {len(X_train)} | Val: {len(X_val)} | Test: {len(X_test)}")
    print(f"Train noise rate: {y_train.mean():.3f}")

    # Train XGBoost
    model = XGBClassifier(
        n_estimators=200,
        max_depth=6,
        learning_rate=0.1,
        min_child_weight=5,
        scale_pos_weight=(1 - y_train.mean()) / max(y_train.mean(), 0.01),
        eval_metric="logloss",
        early_stopping_rounds=20,
        random_state=42,
    )

    model.fit(
        X_train, y_train,
        eval_set=[(X_val, y_val)],
        verbose=False,
    )

    # Evaluate on test set
    y_pred = model.predict(X_test)
    y_proba = model.predict_proba(X_test)[:, 1]

    print("\n── Test Set Performance ────────────────────────")
    print(classification_report(y_test, y_pred, target_names=["Signal", "Noise"]))

    precision, recall, f1, _ = precision_recall_fscore_support(
        y_test, y_pred, average="binary"
    )
    print(f"Noise Gate — Precision: {precision:.3f}, Recall: {recall:.3f}, F1: {f1:.3f}")

    # Feature importance
    print("\n── Top 10 Feature Importances ──────────────────")
    importances = model.feature_importances_
    indices = np.argsort(importances)[::-1][:10]
    for rank, idx in enumerate(indices, 1):
        print(f"  {rank:2d}. {NOISE_GATE_FEATURES[idx]:30s} {importances[idx]:.4f}")

    # Save model
    output_path = Path(args.output)
    output_path.parent.mkdir(parents=True, exist_ok=True)
    joblib.dump(model, output_path)
    print(f"\nModel saved to {output_path}")

    # Save metrics for comparison
    metrics_path = output_path.with_suffix(".metrics.json")
    with open(metrics_path, "w") as f:
        json.dump({
            "precision": float(precision),
            "recall": float(recall),
            "f1": float(f1),
            "train_size": len(X_train),
            "test_size": len(X_test),
            "noise_rate_train": float(y_train.mean()),
        }, f, indent=2)
    print(f"Metrics saved to {metrics_path}")


if __name__ == "__main__":
    main()
