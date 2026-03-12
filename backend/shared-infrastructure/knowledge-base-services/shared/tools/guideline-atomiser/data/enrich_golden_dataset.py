#!/usr/bin/env python3
"""Enrich golden dataset: compute ML features for all reviewed spans in GCP.

Connects to canonical_facts GCP Cloud SQL, JOINs l2_merged_spans with
l2_reviewer_decisions, computes feature vectors via enrich_span_features(),
and exports to Parquet.

Usage:
    python enrich_golden_dataset.py --output golden_dataset_enriched.parquet

Requires:
    pip install psycopg2-binary pandas pyarrow
"""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

import pandas as pd
import psycopg2
import psycopg2.extras

# Add project root to path for extraction imports
_PROJECT_ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(_PROJECT_ROOT))

from extraction.v4.feature_enrichment import enrich_span_features

# GCP Cloud SQL credentials (reuse from export_gcp_to_merged_spans.py)
DB_CONFIG = {
    "host": "34.46.243.149",
    "port": 5433,
    "dbname": "canonical_facts",
    "user": "kb_admin",
    "password": "kb_secure_password_2024",
}

TIER_MAP = {0: "NOISE", 1: "TIER_1", 2: "TIER_2", None: "UNKNOWN"}


def fetch_reviewed_spans() -> list[dict]:
    """Fetch all reviewed spans with their reviewer decisions."""
    conn = psycopg2.connect(**DB_CONFIG)
    cur = conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor)

    cur.execute("""
        SELECT
            ms.id AS span_id,
            ms.job_id,
            ms.text,
            ms.start_offset AS start,
            ms.end_offset AS "end",
            ms.contributing_channels,
            ms.channel_confidences,
            ms.merged_confidence,
            ms.has_disagreement,
            ms.page_number,
            ms.section_id,
            ms.tier,
            rd.action AS reviewer_action,
            rd.decided_at
        FROM l2_merged_spans ms
        JOIN l2_reviewer_decisions rd ON rd.merged_span_id = ms.id
        ORDER BY ms.job_id, ms.page_number NULLS LAST, ms.start_offset
    """)
    rows = cur.fetchall()
    cur.close()
    conn.close()
    return [dict(r) for r in rows]


def enrich_rows(rows: list[dict]) -> pd.DataFrame:
    """Compute features for each reviewed span and build a DataFrame."""
    records = []

    for row in rows:
        channels = row["contributing_channels"] or []
        channel_conf = row["channel_confidences"] or {}

        # Compute features (full_text is unavailable offline — use span text
        # with empty context window; context features will be approximate)
        features = enrich_span_features(
            text=row["text"] or "",
            start=row["start"] or 0,
            end=row["end"] or 0,
            contributing_channels=channels,
            channel_confidences=channel_conf,
            merged_confidence=row["merged_confidence"] or 0.0,
            has_disagreement=row["has_disagreement"] or False,
            full_text=row["text"] or "",  # no full text available offline
            section_id=row["section_id"],
            page_number=row["page_number"],
        )

        # Add target labels and identifiers
        features["span_id"] = str(row["span_id"])
        features["job_id"] = str(row["job_id"])
        features["reviewer_action"] = row["reviewer_action"]
        features["tier"] = TIER_MAP.get(row["tier"], "UNKNOWN")
        features["decided_at"] = row["decided_at"]

        # Derived target: binary noise label (REJECT → noise, CONFIRM/EDIT → signal)
        features["is_noise"] = row["reviewer_action"] == "REJECT"

        records.append(features)

    return pd.DataFrame(records)


def main():
    parser = argparse.ArgumentParser(description="Enrich golden dataset with ML features")
    parser.add_argument(
        "--output", "-o",
        default="golden_dataset_enriched.parquet",
        help="Output Parquet file path",
    )
    args = parser.parse_args()

    print("Fetching reviewed spans from GCP Cloud SQL...")
    rows = fetch_reviewed_spans()
    print(f"  Fetched {len(rows)} reviewed spans")

    if not rows:
        print("ERROR: No reviewed spans found")
        sys.exit(1)

    print("Computing features...")
    df = enrich_rows(rows)

    # Report label distribution
    print("\n─── Label Distribution ─────────────────────")
    print(f"  Total spans: {len(df)}")
    print(f"\n  By reviewer action:")
    for action, count in df["reviewer_action"].value_counts().items():
        pct = count / len(df) * 100
        print(f"    {action}: {count} ({pct:.1f}%)")
    print(f"\n  By tier:")
    for tier, count in df["tier"].value_counts().items():
        pct = count / len(df) * 100
        print(f"    {tier}: {count} ({pct:.1f}%)")
    print(f"\n  By noise archetype (non-null):")
    archetype_counts = df[df["noise_archetype"].notna()]["noise_archetype"].value_counts()
    for arch, count in archetype_counts.items():
        print(f"    {arch}: {count}")
    print("─────────────────────────────────────────────")

    # Save to Parquet
    output_path = Path(args.output)
    df.to_parquet(output_path, index=False, engine="pyarrow")
    print(f"\nSaved enriched dataset to {output_path} ({len(df)} rows, {len(df.columns)} columns)")


if __name__ == "__main__":
    main()
