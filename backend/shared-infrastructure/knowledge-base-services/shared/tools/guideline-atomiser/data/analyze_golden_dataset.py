#!/usr/bin/env python3
"""Analyze enriched golden dataset: label distribution, splits, and diagnostics.

Reads the enriched Parquet from enrich_golden_dataset.py, reports class
distributions, creates stratified train/val/test splits (70/15/15), and
identifies underrepresented archetypes needing more reviewer labels.

Usage:
    python analyze_golden_dataset.py --input golden_dataset_enriched.parquet

Requires:
    pip install pandas scikit-learn
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

import pandas as pd
from sklearn.model_selection import StratifiedShuffleSplit


def analyze_distribution(df: pd.DataFrame) -> None:
    """Print detailed label distribution analysis."""
    print("═══════════════════════════════════════════════════")
    print("  GOLDEN DATASET LABEL DISTRIBUTION ANALYSIS")
    print("═══════════════════════════════════════════════════")

    print(f"\nTotal spans: {len(df)}")

    # By reviewer action
    print("\n── By Reviewer Action ──────────────────────────")
    for action, group in df.groupby("reviewer_action"):
        count = len(group)
        pct = count / len(df) * 100
        print(f"  {action:12s}: {count:5d} ({pct:5.1f}%)")

    # By tier
    print("\n── By Tier ────────────────────────────────────")
    for tier, group in df.groupby("tier"):
        count = len(group)
        pct = count / len(df) * 100
        # Precision: proportion of CONFIRM+EDIT in this tier
        confirmed = len(group[group["reviewer_action"].isin(["CONFIRM", "EDIT"])])
        precision = confirmed / count * 100 if count > 0 else 0
        print(f"  {tier:8s}: {count:5d} ({pct:5.1f}%) — precision {precision:.1f}%")

    # By channel count
    print("\n── By Channel Count ───────────────────────────")
    for n_ch, group in df.groupby("n_channels"):
        count = len(group)
        confirmed = len(group[group["reviewer_action"].isin(["CONFIRM", "EDIT"])])
        precision = confirmed / count * 100 if count > 0 else 0
        print(f"  {n_ch}-channel: {count:5d} — precision {precision:.1f}%")

    # By noise archetype (non-null only)
    archetypes = df[df["noise_archetype"].notna()]
    if len(archetypes) > 0:
        print("\n── By Noise Archetype ─────────────────────────")
        for arch, group in archetypes.groupby("noise_archetype"):
            count = len(group)
            confirmed = len(group[group["reviewer_action"].isin(["CONFIRM", "EDIT"])])
            precision = confirmed / count * 100 if count > 0 else 0
            print(f"  {arch:35s}: {count:4d} — precision {precision:.1f}%")

    # Per-channel-type (single-channel)
    single_ch = df[df["n_channels"] == 1]
    if len(single_ch) > 0:
        print("\n── Single-Channel Precision ────────────────────")
        channel_cols = [c for c in df.columns if c.startswith("has_channel_")]
        for col in channel_cols:
            ch_name = col.replace("has_channel_", "")
            ch_spans = single_ch[single_ch[col] == True]  # noqa: E712
            if len(ch_spans) == 0:
                continue
            confirmed = len(ch_spans[ch_spans["reviewer_action"].isin(["CONFIRM", "EDIT"])])
            precision = confirmed / len(ch_spans) * 100
            print(f"  Channel {ch_name} alone: {len(ch_spans):4d} spans — precision {precision:.1f}%")

    # Disagreement signal
    print("\n── Disagreement Signal ────────────────────────")
    for disagree in [True, False]:
        group = df[df["has_disagreement"] == disagree]
        if len(group) == 0:
            continue
        confirmed = len(group[group["reviewer_action"].isin(["CONFIRM", "EDIT"])])
        precision = confirmed / len(group) * 100
        label = "has_disagreement" if disagree else "no_disagreement"
        print(f"  {label:20s}: {len(group):5d} — precision {precision:.1f}%")

    # Underrepresented archetypes
    print("\n── Underrepresented Archetypes (< 30 samples) ─")
    if len(archetypes) > 0:
        arch_counts = archetypes["noise_archetype"].value_counts()
        for arch, count in arch_counts.items():
            if count < 30:
                print(f"  ⚠️  {arch}: only {count} samples — needs more reviewer labels")
    else:
        print("  No noise archetypes detected in dataset")


def create_splits(
    df: pd.DataFrame,
    output_path: Path,
    train_ratio: float = 0.70,
    val_ratio: float = 0.15,
) -> None:
    """Create stratified train/val/test splits preserving tier and archetype distributions.

    Strategy: Use reviewer_action as stratification key (CONFIRM vs REJECT).
    For archetypes, we use a combined stratification key to preserve distributions.
    """
    # Create stratification key combining is_noise + tier
    df = df.copy()
    df["strat_key"] = df["is_noise"].astype(str) + "_" + df["tier"].fillna("UNKNOWN")

    # Handle rare classes by merging into "OTHER" if < 3 samples
    strat_counts = df["strat_key"].value_counts()
    rare_keys = strat_counts[strat_counts < 3].index
    df.loc[df["strat_key"].isin(rare_keys), "strat_key"] = "OTHER"

    test_ratio = 1.0 - train_ratio - val_ratio

    # First split: train+val vs test
    sss1 = StratifiedShuffleSplit(n_splits=1, test_size=test_ratio, random_state=42)
    train_val_idx, test_idx = next(sss1.split(df, df["strat_key"]))

    # Second split: train vs val (from train+val)
    df_train_val = df.iloc[train_val_idx]
    val_from_trainval = val_ratio / (train_ratio + val_ratio)
    sss2 = StratifiedShuffleSplit(n_splits=1, test_size=val_from_trainval, random_state=42)
    train_idx_rel, val_idx_rel = next(sss2.split(df_train_val, df_train_val["strat_key"]))

    # Map back to original indices
    train_idx = train_val_idx[train_idx_rel]
    val_idx = train_val_idx[val_idx_rel]

    splits = {
        "train_indices": train_idx.tolist(),
        "val_indices": val_idx.tolist(),
        "test_indices": test_idx.tolist(),
        "train_span_ids": df.iloc[train_idx]["span_id"].tolist(),
        "val_span_ids": df.iloc[val_idx]["span_id"].tolist(),
        "test_span_ids": df.iloc[test_idx]["span_id"].tolist(),
        "split_ratios": {
            "train": train_ratio,
            "val": val_ratio,
            "test": test_ratio,
        },
        "total_samples": len(df),
        "train_samples": len(train_idx),
        "val_samples": len(val_idx),
        "test_samples": len(test_idx),
    }

    with open(output_path, "w") as f:
        json.dump(splits, f, indent=2)

    print(f"\n── Dataset Splits ─────────────────────────────")
    print(f"  Train: {len(train_idx):5d} ({len(train_idx)/len(df)*100:.1f}%)")
    print(f"  Val:   {len(val_idx):5d} ({len(val_idx)/len(df)*100:.1f}%)")
    print(f"  Test:  {len(test_idx):5d} ({len(test_idx)/len(df)*100:.1f}%)")
    print(f"  Saved splits to {output_path}")


def main():
    parser = argparse.ArgumentParser(description="Analyze enriched golden dataset")
    parser.add_argument(
        "--input", "-i",
        default="golden_dataset_enriched.parquet",
        help="Input enriched Parquet file",
    )
    parser.add_argument(
        "--splits-output",
        default="golden_dataset_splits.json",
        help="Output splits JSON file",
    )
    args = parser.parse_args()

    input_path = Path(args.input)
    if not input_path.exists():
        print(f"ERROR: Input file not found: {input_path}")
        print("Run enrich_golden_dataset.py first.")
        sys.exit(1)

    df = pd.read_parquet(input_path)
    print(f"Loaded {len(df)} rows from {input_path}")

    analyze_distribution(df)
    create_splits(df, Path(args.splits_output))


if __name__ == "__main__":
    main()
