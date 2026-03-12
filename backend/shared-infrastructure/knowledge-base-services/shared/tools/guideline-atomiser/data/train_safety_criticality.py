#!/usr/bin/env python3
"""Train the safety criticality detector (binary logistic regression).

Reads the enriched golden dataset and trains on manually labelled safety-critical
spans. Requires a 'is_safety_critical' column in the Parquet (manual labelling).

If the column doesn't exist, creates heuristic labels from safety keyword patterns
as a bootstrap — these should be validated by clinical reviewers before production.

Usage:
    python train_safety_criticality.py \
        --input golden_dataset_enriched.parquet \
        --output models/safety_criticality.joblib
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path

import joblib
import pandas as pd
from sklearn.linear_model import LogisticRegression
from sklearn.metrics import classification_report

_PROJECT_ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(_PROJECT_ROOT))

from extraction.v4.classifiers.safety_criticality import SAFETY_FEATURES

_SAFETY_RE = re.compile(
    r'contraindicated|avoid\b|black\s*box|maximum\s+dose|do\s+not\s+use|'
    r'not\s+recommended|discontinue\s+if|withhold|'
    r'life[\s-]threatening|fatal|anaphylaxis|renal\s+failure|hepatotoxic',
    re.IGNORECASE,
)


def main():
    parser = argparse.ArgumentParser(description="Train safety criticality detector")
    parser.add_argument("--input", "-i", default="golden_dataset_enriched.parquet")
    parser.add_argument("--output", "-o", default="models/safety_criticality.joblib")
    args = parser.parse_args()

    df = pd.read_parquet(args.input)
    print(f"Loaded {len(df)} spans")

    # Check for manual labels, create heuristic labels if absent
    if "is_safety_critical" not in df.columns:
        print("WARNING: No manual safety labels found. Using heuristic bootstrap.")
        print("These labels should be validated by clinical reviewers before production.")
        # Use span text (stored as 'text' or reconstructable from features)
        # Since we don't have text in features, use has_safety_keyword as proxy
        df["is_safety_critical"] = df["has_safety_keyword"].astype(bool)

    y = df["is_safety_critical"].astype(int).values
    X = df[SAFETY_FEATURES].fillna(0).values

    print(f"Safety-critical spans: {y.sum()} ({y.mean()*100:.1f}%)")

    # Simple train/test split (80/20)
    from sklearn.model_selection import train_test_split
    X_train, X_test, y_train, y_test = train_test_split(
        X, y, test_size=0.2, random_state=42, stratify=y
    )

    model = LogisticRegression(
        max_iter=1000,
        class_weight="balanced",
        random_state=42,
    )
    model.fit(X_train, y_train)

    y_pred = model.predict(X_test)
    print("\n── Test Set Performance ────────────────────────")
    print(classification_report(y_test, y_pred, target_names=["Non-safety", "Safety-critical"]))

    # Feature coefficients
    print("── Feature Coefficients ────────────────────────")
    for name, coef in sorted(
        zip(SAFETY_FEATURES, model.coef_[0]),
        key=lambda x: abs(x[1]),
        reverse=True,
    ):
        print(f"  {name:30s} {coef:+.4f}")

    output_path = Path(args.output)
    output_path.parent.mkdir(parents=True, exist_ok=True)
    joblib.dump(model, output_path)
    print(f"\nModel saved to {output_path}")


if __name__ == "__main__":
    main()
