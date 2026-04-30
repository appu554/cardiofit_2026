#!/usr/bin/env python3
"""Push Pipeline 1 V4 reviewer-queue artefacts to KB-0 governance platform.

Reads a local job dir (data/output/v4/job_monkeyocr_<uuid>/) and inserts
its rows into the GCP Cloud SQL canonical_facts database — the same
database the KB-0 governance platform reads from to surface spans for
human review.

Tables written:
  l2_extraction_jobs   (1 row per job, ON CONFLICT (job_id) DO UPDATE)
  l2_merged_spans      (N rows; DELETE-then-INSERT for full replace)
  l2_section_passages  (N rows; DELETE-then-INSERT)
  l2_guideline_tree    (1 row, ON CONFLICT (job_id) DO UPDATE)

Usage:
    # Single job
    python push_to_kb0_gcp.py output/v4/job_monkeyocr_<uuid>

    # All HF queue jobs at once
    python push_to_kb0_gcp.py output/v4/job_monkeyocr_*/

    # Dry-run to preview SQL only (no writes)
    python push_to_kb0_gcp.py --dry-run output/v4/job_monkeyocr_<uuid>

After push, the spans show up in the KB-0 reviewer UI under the new job_id.
The companion script export_gcp_to_merged_spans.py round-trips the
reviewed spans back to local for Pipeline 2 ingestion.
"""

from __future__ import annotations

import argparse
import json
import logging
import sys
from datetime import datetime, timezone
from pathlib import Path

import psycopg2
import psycopg2.extras

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)-7s %(message)s")
log = logging.getLogger(__name__)

# GCP Cloud SQL — same credentials the KB-0 server reads from .env
DB_CONFIG = dict(
    host="34.46.243.149",
    port=5433,
    dbname="canonical_facts",
    user="kb_admin",
    password="kb_secure_password_2024",
    connect_timeout=10,
)

# Tier string → smallint (matches export_gcp_to_merged_spans.py reverse map)
TIER_REVERSE = {"NOISE": 0, "TIER_1": 1, "TIER_2": 2, None: None}

# Default guideline_tier (smallint) when not set in metadata. AU sources
# default to 1 (Tier 1 = primary peak-body guideline) until reviewers
# downgrade individual jobs.
DEFAULT_GUIDELINE_TIER = 1


def push_job(conn, job_dir: Path, dry_run: bool = False) -> dict:
    """Push one Pipeline 1 V4 job dir to KB-0 GCP.

    Returns a dict with per-table row counts.
    """
    if not job_dir.is_dir():
        raise FileNotFoundError(f"Job dir not found: {job_dir}")

    log.info("--- Pushing %s ---", job_dir.name)

    metadata = json.loads((job_dir / "job_metadata.json").read_text())
    spans = json.loads((job_dir / "merged_spans.json").read_text())
    passages = json.loads((job_dir / "section_passages.json").read_text())
    tree = json.loads((job_dir / "guideline_tree.json").read_text())
    norm_text = (job_dir / "normalized_text.txt").read_text() if (
        job_dir / "normalized_text.txt"
    ).exists() else ""

    job_id = metadata["job_id"]
    log.info("  job_id=%s  spans=%d  passages=%d  tree_keys=%s",
             job_id, len(spans), len(passages),
             list(tree.keys())[:6] if isinstance(tree, dict) else "?")

    counts = {}

    if dry_run:
        log.info("  [DRY RUN] would write to GCP — skipping actual SQL")
        counts["jobs"] = 1
        counts["spans"] = len(spans)
        counts["passages"] = len(passages)
        counts["tree"] = 1
        return counts

    cur = conn.cursor()

    # --- l2_extraction_jobs ---
    job_row = (
        job_id,
        metadata.get("source_pdf"),
        metadata.get("page_range"),
        metadata.get("pipeline_version", "4.2.2"),
        metadata.get("l1_backend") or metadata.get("l1_tag"),
        int(metadata.get("total_merged_spans", len(spans))),
        int(metadata.get("section_passages", len(passages))),
        int(tree.get("total_pages", 0)) if isinstance(tree, dict) else 0,
        float(metadata.get("alignment_confidence", 0.0)),
        psycopg2.extras.Json(metadata.get("l1_oracle") or {}),
        0, 0, 0, 0,                    # confirmed/rejected/edited/added counters
        len(spans),                    # spans_pending = total since just ingested
        # l2_extraction_jobs_status_check allows: PENDING_REVIEW, IN_PROGRESS,
        # COMPLETED, ARCHIVED — local job_metadata uses 'PENDING' which we
        # remap here to 'PENDING_REVIEW'.
        "PENDING_REVIEW",
        metadata.get("created_at") or datetime.now(timezone.utc).isoformat(),
        datetime.now(timezone.utc).isoformat(),  # updated_at
        None,                          # completed_at
        f"data/pdfs/{metadata.get('source_pdf')}",
        0,                             # pdf_page_offset
        None,                          # completed_by
        DEFAULT_GUIDELINE_TIER,
    )
    cur.execute("""
        INSERT INTO l2_extraction_jobs (
            job_id, source_pdf, page_range, pipeline_version, l1_tag,
            total_merged_spans, total_sections, total_pages, alignment_confidence,
            l1_oracle_stats, spans_confirmed, spans_rejected, spans_edited,
            spans_added, spans_pending, status, created_at, updated_at,
            completed_at, source_pdf_path, pdf_page_offset, completed_by,
            guideline_tier
        ) VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s)
        ON CONFLICT (job_id) DO UPDATE SET
            total_merged_spans = EXCLUDED.total_merged_spans,
            total_sections = EXCLUDED.total_sections,
            total_pages = EXCLUDED.total_pages,
            alignment_confidence = EXCLUDED.alignment_confidence,
            l1_oracle_stats = EXCLUDED.l1_oracle_stats,
            spans_pending = EXCLUDED.spans_pending,
            status = EXCLUDED.status,
            updated_at = EXCLUDED.updated_at
    """, job_row)
    counts["jobs"] = 1

    # --- l2_merged_spans (DELETE then INSERT for full replace) ---
    cur.execute("DELETE FROM l2_merged_spans WHERE job_id = %s", (job_id,))
    cur.execute("DELETE FROM l2_section_passages WHERE job_id = %s", (job_id,))
    cur.execute("DELETE FROM l2_guideline_tree WHERE job_id = %s", (job_id,))

    span_rows = []
    for s in spans:
        span_rows.append((
            s["id"],
            s["job_id"],
            s["text"],
            int(s["start"]),
            int(s["end"]),
            s.get("contributing_channels") or [],
            psycopg2.extras.Json(s.get("channel_confidences") or {}),
            float(s.get("merged_confidence") or 0.0),
            bool(s.get("has_disagreement")),
            s.get("disagreement_detail"),
            s.get("page_number"),
            s.get("section_id"),
            s.get("table_id"),
            s.get("review_status", "PENDING"),
            s.get("reviewer_text"),
            s.get("reviewed_by"),
            s.get("reviewed_at"),
            s.get("created_at") or datetime.now(timezone.utc).isoformat(),
            psycopg2.extras.Json(s.get("bbox")) if s.get("bbox") else None,
            s.get("surrounding_context"),
            TIER_REVERSE.get(s.get("tier")),
            psycopg2.extras.Json(s.get("coverage_guard_alert")) if s.get("coverage_guard_alert") else None,
            psycopg2.extras.Json(s.get("semantic_tokens")) if s.get("semantic_tokens") else None,
        ))
    if span_rows:
        psycopg2.extras.execute_batch(cur, """
            INSERT INTO l2_merged_spans (
                id, job_id, text, start_offset, end_offset, contributing_channels,
                channel_confidences, merged_confidence, has_disagreement,
                disagreement_detail, page_number, section_id, table_id,
                review_status, reviewer_text, reviewed_by, reviewed_at,
                created_at, bbox, surrounding_context, tier, coverage_guard_alert,
                semantic_tokens
            ) VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s)
        """, span_rows, page_size=200)
    counts["spans"] = len(span_rows)

    # --- l2_section_passages ---
    # Local section_ids occasionally collide when two short headings share
    # the same slug (e.g. headings like "2" and "2. Hospital care" both get
    # section_id="2"). l2_section_passages PK is (job_id, section_id), so
    # we disambiguate collisions with a __dN suffix and log the rename.
    pass_rows = []
    seen: dict[str, int] = {}
    for p in passages:
        sid = p["section_id"]
        if sid in seen:
            seen[sid] += 1
            new_sid = f"{sid}__d{seen[sid]}"
            log.warning("    duplicate section_id %r → renamed to %r (heading=%r)",
                        sid, new_sid, p.get("heading", "")[:60])
            sid = new_sid
        else:
            seen[sid] = 0
        pass_rows.append((
            job_id,
            sid,
            p["heading"],
            p.get("page_number"),
            p.get("prose_text"),
            p.get("span_ids") or [],
            int(p.get("span_count", 0)),
            p.get("child_section_ids") or [],
            p.get("start_offset"),
            p.get("end_offset"),
        ))
    if pass_rows:
        # span_ids is uuid[]; child_section_ids is text[]. psycopg2 sends
        # text[] by default, so we explicitly cast span_ids to uuid[].
        psycopg2.extras.execute_batch(cur, """
            INSERT INTO l2_section_passages (
                job_id, section_id, heading, page_number, prose_text,
                span_ids, span_count, child_section_ids, start_offset, end_offset
            ) VALUES (%s,%s,%s,%s,%s,%s::uuid[],%s,%s,%s,%s)
        """, pass_rows, page_size=200)
    counts["passages"] = len(pass_rows)

    # --- l2_guideline_tree ---
    cur.execute("""
        INSERT INTO l2_guideline_tree (job_id, tree_json, normalized_text)
        VALUES (%s, %s, %s)
        ON CONFLICT (job_id) DO UPDATE SET
            tree_json = EXCLUDED.tree_json,
            normalized_text = EXCLUDED.normalized_text
    """, (job_id, psycopg2.extras.Json(tree), norm_text))
    counts["tree"] = 1

    conn.commit()
    log.info("  ✅ pushed: jobs=%d  spans=%d  passages=%d  tree=%d",
             counts["jobs"], counts["spans"], counts["passages"], counts["tree"])
    return counts


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__,
                                formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("job_dirs", nargs="+", help="One or more job dirs")
    p.add_argument("--dry-run", action="store_true",
                   help="Read + validate locally; don't write to GCP")
    args = p.parse_args()

    job_dirs = [Path(d) for d in args.job_dirs if Path(d).is_dir()]
    if not job_dirs:
        log.error("no valid job dirs given")
        return 1

    log.info("=" * 70)
    log.info("KB-0 GCP push: %d job(s)", len(job_dirs))
    log.info("=" * 70)

    if args.dry_run:
        conn = None
    else:
        conn = psycopg2.connect(**DB_CONFIG)

    totals = {"jobs": 0, "spans": 0, "passages": 0, "tree": 0}
    try:
        for d in job_dirs:
            try:
                c = push_job(conn, d, dry_run=args.dry_run)
                for k, v in c.items():
                    totals[k] = totals.get(k, 0) + v
            except Exception as e:
                log.error("  FAILED on %s: %s", d.name, e)
                if conn:
                    conn.rollback()
                # Continue to the next job dir.
    finally:
        if conn:
            conn.close()

    log.info("=" * 70)
    log.info("TOTALS  jobs=%(jobs)d  spans=%(spans)d  passages=%(passages)d  tree=%(tree)d", totals)
    log.info("=" * 70)
    return 0


if __name__ == "__main__":
    sys.exit(main())
