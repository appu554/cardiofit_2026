#!/usr/bin/env python3
"""Export reviewed spans from GCP Cloud SQL → merged_spans.json for Pipeline 2.

Reads all l2_merged_spans for a given job_id from the canonical_facts database,
maps GCP columns to MergedSpan JSON schema, and writes merged_spans.json.

Handles:
- Column renames: start_offset→start, end_offset→end
- PostgreSQL ARRAY→list, jsonb→dict
- tier smallint→string mapping (0→NOISE, 1→TIER_1, 2→TIER_2)
- Auto-rejects PENDING spans (Pipeline 2 fatally exits on any PENDING)
- Preserves ADDED spans from reviewer (start=-1, end=-1 for manually added)

Usage:
    python export_gcp_to_merged_spans.py --job-id df538e50-0170-4ef8-862d-5b0a7c48e4ff
"""

from __future__ import annotations

import argparse
import json
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Optional

import psycopg2
import psycopg2.extras

# GCP Cloud SQL credentials (from kb-0-governance-platform/.env)
DB_CONFIG = {
    "host": "34.46.243.149",
    "port": 5433,
    "dbname": "canonical_facts",
    "user": "kb_admin",
    "password": "kb_secure_password_2024",
}

# Tier smallint → string mapping
TIER_MAP = {0: "NOISE", 1: "TIER_1", 2: "TIER_2", None: None}


def export_spans(job_id: str, output_dir: str | None = None, auto_reject_pending: bool = True) -> Path:
    """Export reviewed spans from GCP to merged_spans.json.

    Args:
        job_id: Pipeline job UUID.
        output_dir: Output directory. If None, uses standard job directory.
        auto_reject_pending: If True, set PENDING spans to REJECTED.

    Returns:
        Path to written merged_spans.json.
    """
    conn = psycopg2.connect(**DB_CONFIG)
    cur = conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor)

    # Fetch all spans for this job
    cur.execute(
        """
        SELECT id, job_id, text, start_offset, end_offset,
               contributing_channels, channel_confidences,
               merged_confidence, has_disagreement, disagreement_detail,
               page_number, section_id, table_id,
               review_status, reviewer_text, reviewed_by, reviewed_at,
               bbox, surrounding_context, tier
        FROM l2_merged_spans
        WHERE job_id = %s
        ORDER BY page_number NULLS LAST, start_offset
        """,
        (job_id,),
    )
    rows = cur.fetchall()
    cur.close()
    conn.close()

    if not rows:
        print(f"ERROR: No spans found for job_id={job_id}")
        sys.exit(1)

    # Status counts
    status_counts = {}
    for row in rows:
        s = row["review_status"]
        status_counts[s] = status_counts.get(s, 0) + 1

    print(f"Found {len(rows)} spans in GCP for job {job_id}")
    for status, count in sorted(status_counts.items(), key=lambda x: -x[1]):
        print(f"  {status:12s} {count:5d}")

    pending_count = status_counts.get("PENDING", 0)
    if pending_count > 0:
        if auto_reject_pending:
            print(f"\n⚠️  Auto-rejecting {pending_count} PENDING spans (Pipeline 2 requires zero PENDING)")
        else:
            print(f"\n❌ {pending_count} PENDING spans found. Pipeline 2 will refuse to run.")
            print("   Use --auto-reject-pending to auto-reject them.")
            sys.exit(1)

    # Convert to MergedSpan JSON format
    spans_json = []
    for row in rows:
        review_status = row["review_status"]
        if review_status == "PENDING" and auto_reject_pending:
            review_status = "REJECTED"

        # Map tier smallint → string
        tier_val = TIER_MAP.get(row["tier"])

        # Handle reviewed_at timestamp
        reviewed_at = None
        if row["reviewed_at"] is not None:
            if isinstance(row["reviewed_at"], datetime):
                reviewed_at = row["reviewed_at"].isoformat()
            else:
                reviewed_at = str(row["reviewed_at"])

        span = {
            "id": str(row["id"]),
            "job_id": str(row["job_id"]),
            "text": row["text"],
            "start": row["start_offset"],       # GCP start_offset → JSON start
            "end": row["end_offset"],            # GCP end_offset → JSON end
            "contributing_channels": list(row["contributing_channels"]) if row["contributing_channels"] else [],
            "channel_confidences": row["channel_confidences"] if row["channel_confidences"] else {},
            "merged_confidence": float(row["merged_confidence"]) if row["merged_confidence"] else 0.0,
            "has_disagreement": bool(row["has_disagreement"]),
            "disagreement_detail": row["disagreement_detail"],
            "page_number": row["page_number"],
            "section_id": row["section_id"],
            "table_id": row["table_id"],
            "bbox": row["bbox"],
            "surrounding_context": row["surrounding_context"],
            "review_status": review_status,
            "reviewer_text": row["reviewer_text"],
            "reviewed_by": row["reviewed_by"],
            "reviewed_at": reviewed_at,
        }

        # Add tier fields if present (Phase 6 addition)
        if tier_val is not None:
            span["tier"] = tier_val
            span["tier_reason"] = None  # GCP doesn't store tier_reason yet

        spans_json.append(span)

    # Determine output path
    if output_dir is None:
        script_dir = Path(__file__).resolve().parent
        output_dir = script_dir / "output" / "v4" / f"job_monkeyocr_{job_id}"

    output_path = Path(output_dir)
    output_path.mkdir(parents=True, exist_ok=True)

    # Back up existing merged_spans.json
    target = output_path / "merged_spans.json"
    if target.exists():
        backup = output_path / "merged_spans_pre_review_backup.json"
        if not backup.exists():
            target.rename(backup)
            print(f"\n📋 Backed up original to {backup.name}")
        else:
            print(f"\n📋 Backup already exists, overwriting merged_spans.json")

    # Write
    with open(target, "w") as f:
        json.dump(spans_json, f, indent=2, default=str)

    # Summary
    final_statuses = {}
    for s in spans_json:
        st = s["review_status"]
        final_statuses[st] = final_statuses.get(st, 0) + 1

    print(f"\n✅ Exported {len(spans_json)} spans to {target}")
    print("   Final status distribution:")
    for status, count in sorted(final_statuses.items(), key=lambda x: -x[1]):
        marker = "✅" if status in ("CONFIRMED", "EDITED", "ADDED") else "❌"
        print(f"   {marker} {status:12s} {count:5d}")

    verified = sum(
        1 for s in spans_json if s["review_status"] in ("CONFIRMED", "EDITED", "ADDED")
    )
    print(f"\n   Pipeline 2 will use {verified} verified spans for dossier assembly")

    return target


def main():
    parser = argparse.ArgumentParser(description="Export GCP reviewed spans to merged_spans.json")
    parser.add_argument("--job-id", required=True, help="Pipeline job UUID")
    parser.add_argument("--output-dir", help="Output directory (default: standard job dir)")
    parser.add_argument(
        "--no-auto-reject",
        action="store_true",
        help="Don't auto-reject PENDING spans (will exit if any found)",
    )
    args = parser.parse_args()

    export_spans(
        job_id=args.job_id,
        output_dir=args.output_dir,
        auto_reject_pending=not args.no_auto_reject,
    )


if __name__ == "__main__":
    main()
