#!/usr/bin/env python3
"""Train the tier assigner classifier (multiclass XGBoost).

Reads the enriched golden dataset, trains a TIER_1/TIER_2/NOISE classifier,
evaluates on held-out test set, and saves the model.

Usage:
    python train_tier_assigner.py \
        --input golden_dataset_enriched.parquet \
        --splits golden_dataset_splits.json \
        --output models/tier_assigner.joblib
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

import joblib
import numpy as np
import pandas as pd
from sklearn.metrics import classification_report
from sklearn.preprocessing import LabelEncoder
from xgboost import XGBClassifier

_PROJECT_ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(_PROJECT_ROOT))

from extraction.v4.classifiers.tier_assigner import TIER_ASSIGNER_FEATURES

# Target mapping: reviewer_action → tier label for training
# CONFIRM/EDIT → keep original tier, REJECT → NOISE
def _derive_training_label(row) -> str:
    if row["reviewer_action"] == "REJECT":
        return "NOISE"
    # Use the pipeline's original tier for CONFIRM/EDIT
    tier = row.get("tier", "TIER_2")
    if tier in ("TIER_1", "TIER_2"):
        return tier
    return "TIER_2"


def main():
    parser = argparse.ArgumentParser(description="Train tier assigner classifier")
    parser.add_argument("--input", "-i", default="golden_dataset_enriched.parquet")
    parser.add_argument("--splits", "-s", default="golden_dataset_splits.json")
    parser.add_argument("--output", "-o", default="models/tier_assigner.joblib")
    args = parser.parse_args()

    df = pd.read_parquet(args.input)
    with open(args.splits) as f:
        splits = json.load(f)

    # Derive training labels
    df["training_label"] = df.apply(_derive_training_label, axis=1)
    label_encoder = LabelEncoder()
    df["label_encoded"] = label_encoder.fit_transform(df["training_label"])

    print(f"Loaded {len(df)} spans")
    print(f"Label distribution:")
    for label, count in df["training_label"].value_counts().items():
        print(f"  {label}: {count} ({count/len(df)*100:.1f}%)")

    X = df[TIER_ASSIGNER_FEATURES].fillna(0).values
    y = df["label_encoded"].values

    train_idx = splits["train_indices"]
    val_idx = splits["val_indices"]
    test_idx = splits["test_indices"]

    X_train, y_train = X[train_idx], y[train_idx]
    X_val, y_val = X[val_idx], y[val_idx]
    X_test, y_test = X[test_idx], y[test_idx]

    print(f"\nTrain: {len(X_train)} | Val: {len(X_val)} | Test: {len(X_test)}")

    model = XGBClassifier(
        n_estimators=300,
        max_depth=6,
        learning_rate=0.1,
        min_child_weight=3,
        objective="multi:softprob",
        num_class=len(label_encoder.classes_),
        eval_metric="mlogloss",
        early_stopping_rounds=20,
        random_state=42,
    )

    model.fit(
        X_train, y_train,
        eval_set=[(X_val, y_val)],
        verbose=False,
    )

    y_pred = model.predict(X_test)
    target_names = label_encoder.classes_.tolist()

    print("\n── Test Set Performance ────────────────────────")
    print(classification_report(y_test, y_pred, target_names=target_names))

    # Feature importance
    print("── Top 10 Feature Importances ──────────────────")
    importances = model.feature_importances_
    indices = np.argsort(importances)[::-1][:10]
    for rank, idx in enumerate(indices, 1):
        print(f"  {rank:2d}. {TIER_ASSIGNER_FEATURES[idx]:30s} {importances[idx]:.4f}")

    # Save model + label encoder
    output_path = Path(args.output)
    output_path.parent.mkdir(parents=True, exist_ok=True)
    joblib.dump(model, output_path)

    # Save label encoder alongside model
    encoder_path = output_path.with_name("tier_assigner_encoder.joblib")
    joblib.dump(label_encoder, encoder_path)
    print(f"\nModel saved to {output_path}")
    print(f"Label encoder saved to {encoder_path}")


if __name__ == "__main__":
    main()
