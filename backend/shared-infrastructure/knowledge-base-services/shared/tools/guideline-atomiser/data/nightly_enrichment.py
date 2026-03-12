#!/usr/bin/env python3
"""Nightly batch enrichment: compute features for newly reviewed spans.

Cron-ready script that:
1. Queries l2_reviewer_decisions for decisions made since last run
2. Joins with l2_merged_spans to get span data
3. Calls enrich_span_features() on each
4. Upserts enrichment_features JSONB column in l2_merged_spans
5. Appends to golden_dataset_enriched.parquet

Usage:
    python nightly_enrichment.py  # processes all un-enriched spans
    python nightly_enrichment.py --since 2026-03-01  # from a specific date

Cron example (run nightly at 2am):
    0 2 * * * cd /path/to/guideline-atomiser/data && python nightly_enrichment.py
"""

from __future__ import annotations

import argparse
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

import pandas as pd
import psycopg2
import psycopg2.extras

_PROJECT_ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(_PROJECT_ROOT))

from extraction.v4.feature_enrichment import enrich_span_features

DB_CONFIG = {
    "host": "34.46.243.149",
    "port": 5433,
    "dbname": "canonical_facts",
    "user": "kb_admin",
    "password": "kb_secure_password_2024",
}


def fetch_unenriched_spans(since: str | None = None) -> list[dict]:
    """Fetch spans that have reviewer decisions but no enrichment features."""
    conn = psycopg2.connect(**DB_CONFIG)
    cur = conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor)

    query = """
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
        WHERE ms.enrichment_features IS NULL
    """
    params = []
    if since:
        query += " AND rd.decided_at >= %s"
        params.append(since)

    query += " ORDER BY rd.decided_at"
    cur.execute(query, params)
    rows = cur.fetchall()
    cur.close()
    conn.close()
    return [dict(r) for r in rows]


def upsert_enrichment(span_id: str, features: dict) -> None:
    """Upsert enrichment_features JSONB for a span."""
    conn = psycopg2.connect(**DB_CONFIG)
    cur = conn.cursor()
    cur.execute(
        """
        UPDATE l2_merged_spans
        SET enrichment_features = %s
        WHERE id = %s
        """,
        (json.dumps(features), span_id),
    )
    conn.commit()
    cur.close()
    conn.close()


def main():
    parser = argparse.ArgumentParser(description="Nightly enrichment batch")
    parser.add_argument("--since", help="Only process decisions after this date (YYYY-MM-DD)")
    parser.add_argument(
        "--parquet", default="golden_dataset_enriched.parquet",
        help="Parquet file to append to",
    )
    args = parser.parse_args()

    print(f"[{datetime.now(timezone.utc).isoformat()}] Nightly enrichment starting...")

    rows = fetch_unenriched_spans(args.since)
    print(f"  Found {len(rows)} un-enriched reviewed spans")

    if not rows:
        print("  Nothing to do.")
        return

    new_records = []
    for i, row in enumerate(rows):
        channels = row["contributing_channels"] or []
        channel_conf = row["channel_confidences"] or {}

        features = enrich_span_features(
            text=row["text"] or "",
            start=row["start"] or 0,
            end=row["end"] or 0,
            contributing_channels=channels,
            channel_confidences=channel_conf,
            merged_confidence=row["merged_confidence"] or 0.0,
            has_disagreement=row["has_disagreement"] or False,
            full_text=row["text"] or "",
            section_id=row["section_id"],
            page_number=row["page_number"],
        )

        # Upsert to DB
        upsert_enrichment(str(row["span_id"]), features)

        # Add metadata for Parquet
        features["span_id"] = str(row["span_id"])
        features["job_id"] = str(row["job_id"])
        features["reviewer_action"] = row["reviewer_action"]
        features["tier"] = {0: "NOISE", 1: "TIER_1", 2: "TIER_2"}.get(row["tier"], "UNKNOWN")
        features["decided_at"] = row["decided_at"]
        features["is_noise"] = row["reviewer_action"] == "REJECT"
        new_records.append(features)

        if (i + 1) % 100 == 0:
            print(f"  Enriched {i + 1}/{len(rows)} spans...")

    # Append to Parquet
    new_df = pd.DataFrame(new_records)
    parquet_path = Path(args.parquet)
    if parquet_path.exists():
        existing_df = pd.read_parquet(parquet_path)
        combined_df = pd.concat([existing_df, new_df], ignore_index=True)
        # Deduplicate by span_id (keep latest)
        combined_df = combined_df.drop_duplicates(subset=["span_id"], keep="last")
        combined_df.to_parquet(parquet_path, index=False, engine="pyarrow")
        print(f"  Appended {len(new_df)} rows to {parquet_path} (total: {len(combined_df)})")
    else:
        new_df.to_parquet(parquet_path, index=False, engine="pyarrow")
        print(f"  Created {parquet_path} with {len(new_df)} rows")

    # Report label shift
    print(f"\n  New decisions: CONFIRM={sum(1 for r in new_records if r['reviewer_action']=='CONFIRM')}, "
          f"REJECT={sum(1 for r in new_records if r['reviewer_action']=='REJECT')}, "
          f"EDIT={sum(1 for r in new_records if r['reviewer_action']=='EDIT')}")


if __name__ == "__main__":
    main()
