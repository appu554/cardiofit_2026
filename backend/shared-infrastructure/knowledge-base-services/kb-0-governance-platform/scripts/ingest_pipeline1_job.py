#!/usr/bin/env python3
"""
Pipeline 1 Ingestion Script — Load V4.2.1 extraction artifacts into GCP KB0 DB.

Reads job output from Pipeline 1 Docker volume and inserts into the l2_* tables
on the GCP PostgreSQL database (canonical_facts).

Usage:
    # Single job directory
    python ingest_pipeline1_job.py /data/output/v4/job_d6108962/

    # All job directories under a parent
    python ingest_pipeline1_job.py --all /data/output/v4/

    # Override database connection
    DB_HOST=34.46.243.149 DB_PORT=5433 DB_NAME=canonical_facts \
        python ingest_pipeline1_job.py /data/output/v4/job_abc123/

Environment Variables:
    DB_HOST     PostgreSQL host    (default: 34.46.243.149)
    DB_PORT     PostgreSQL port    (default: 5433)
    DB_NAME     Database name      (default: canonical_facts)
    DB_USER     Database user      (default: kb0_user)
    DB_PASSWORD Database password  (default: empty)
"""

import argparse
import json
import os
import sys
from datetime import datetime, timezone
from pathlib import Path
from uuid import UUID

try:
    import psycopg2
    import psycopg2.extras
except ImportError:
    print("ERROR: psycopg2 not installed. Run: pip install psycopg2-binary")
    sys.exit(1)


# ─────────────────────────────────────────────────────────────────────────────
# Database connection
# ─────────────────────────────────────────────────────────────────────────────

def get_connection():
    """Connect to GCP PostgreSQL (canonical_facts database)."""
    return psycopg2.connect(
        host=os.getenv("DB_HOST", "34.46.243.149"),
        port=os.getenv("DB_PORT", "5433"),
        dbname=os.getenv("DB_NAME", "canonical_facts"),
        user=os.getenv("DB_USER", "kb0_user"),
        password=os.getenv("DB_PASSWORD", ""),
    )


# ─────────────────────────────────────────────────────────────────────────────
# File loaders
# ─────────────────────────────────────────────────────────────────────────────

def load_json(path: Path):
    with open(path, "r", encoding="utf-8") as f:
        return json.load(f)


def load_text(path: Path) -> str:
    with open(path, "r", encoding="utf-8") as f:
        return f.read()


# ─────────────────────────────────────────────────────────────────────────────
# Section flattening — derive l2_section_passages from guideline_tree
# ─────────────────────────────────────────────────────────────────────────────

def flatten_sections(sections: list[dict], normalized_text: str, spans: list[dict]) -> list[dict]:
    """
    Recursively flatten the guideline tree into passage rows for l2_section_passages.

    Each section gets:
    - prose_text: slice of normalized_text between its offsets
    - span_ids: UUIDs of merged spans whose start falls within this section
    - child_section_ids: IDs of direct children
    """
    results = []

    def _walk(section_list: list[dict]):
        for sec in section_list:
            start = sec.get("start_offset", 0)
            end = sec.get("end_offset", 0)
            children = sec.get("children", [])

            # Extract prose text from normalized text
            prose = ""
            if start >= 0 and end > start and end <= len(normalized_text):
                prose = normalized_text[start:end]

            # Find spans belonging to this section
            section_span_ids = [
                s["id"] for s in spans
                if s.get("section_id") == sec["section_id"]
            ]

            results.append({
                "section_id": sec["section_id"],
                "heading": sec.get("heading", ""),
                "page_number": sec.get("page_number"),
                "prose_text": prose,
                "span_ids": section_span_ids,
                "span_count": len(section_span_ids),
                "child_section_ids": [c["section_id"] for c in children],
                "start_offset": start,
                "end_offset": end,
            })

            if children:
                _walk(children)

    _walk(sections)
    return results


# ─────────────────────────────────────────────────────────────────────────────
# Ingestion logic
# ─────────────────────────────────────────────────────────────────────────────

def ingest_job(job_dir: Path, conn) -> bool:
    """
    Ingest a single Pipeline 1 job directory into the database.

    Expected directory contents:
        job_metadata.json
        merged_spans.json
        guideline_tree.json
        normalized_text.txt

    Returns True if ingested, False if skipped (already exists).
    """
    # ── Validate required files ──────────────────────────────────────────
    metadata_path = job_dir / "job_metadata.json"
    spans_path = job_dir / "merged_spans.json"
    tree_path = job_dir / "guideline_tree.json"
    text_path = job_dir / "normalized_text.txt"

    if not metadata_path.exists():
        print(f"  SKIP: No job_metadata.json in {job_dir}")
        return False

    # ── Load artifacts ───────────────────────────────────────────────────
    metadata = load_json(metadata_path)
    job_id = metadata["job_id"]

    print(f"  Loading job {job_id} from {job_dir.name}...")

    # ── Idempotency check ────────────────────────────────────────────────
    with conn.cursor() as cur:
        cur.execute("SELECT 1 FROM l2_extraction_jobs WHERE job_id = %s", (job_id,))
        if cur.fetchone():
            print(f"  SKIP: Job {job_id} already exists in database")
            return False

    # Load remaining files
    spans = load_json(spans_path) if spans_path.exists() else []
    tree_data = load_json(tree_path) if tree_path.exists() else {"sections": [], "tables": [], "total_pages": 0}
    normalized_text = load_text(text_path) if text_path.exists() else ""

    # Load section passages — prefer pre-assembled file, fall back to guideline_tree
    passages_path = job_dir / "section_passages.json"
    if passages_path.exists():
        passages = load_json(passages_path)
        print(f"    Loaded {len(passages)} passages from section_passages.json")
    else:
        sections = tree_data.get("sections", [])
        passages = flatten_sections(sections, normalized_text, spans)
        print(f"    Derived {len(passages)} passages from guideline_tree.json")

    # ── Deduplicate section_ids (Option A: drop Quick Reference) ─────
    # KDIGO guidelines have a Quick Reference summary (pages 2-29) that
    # mirrors chapter headings, producing duplicate section_ids.
    # Keep only the richest entry (most children, then most spans) per
    # section_id. The raw section_passages.json artifact preserves all.
    from collections import defaultdict
    sid_groups = defaultdict(list)
    for i, p in enumerate(passages):
        sid_groups[p["section_id"]].append(i)

    drop_indices = set()
    for sid, indices in sid_groups.items():
        if len(indices) <= 1:
            continue
        # Pick canonical: most child_section_ids, then most spans, then highest page
        def sort_key(idx):
            p = passages[idx]
            return (
                len(p.get("child_section_ids", [])),
                p.get("span_count", 0),
                p.get("page_number", 0) or 0,
            )
        indices_sorted = sorted(indices, key=sort_key, reverse=True)
        canonical = passages[indices_sorted[0]]
        for idx in indices_sorted[1:]:
            dropped = passages[idx]
            print(f"    DROP duplicate '{sid}' page {dropped.get('page_number')} "
                  f"({dropped.get('span_count')} spans) — "
                  f"keeping page {canonical.get('page_number')} "
                  f"({canonical.get('span_count')} spans, "
                  f"{len(canonical.get('child_section_ids', []))} children)")
            drop_indices.add(idx)

    if drop_indices:
        passages = [p for i, p in enumerate(passages) if i not in drop_indices]
        print(f"    Dropped {len(drop_indices)} Quick Reference duplicates → "
              f"{len(passages)} passages")

    # Count sections recursively
    def count_sections(sec_list):
        total = len(sec_list)
        for s in sec_list:
            total += count_sections(s.get("children", []))
        return total

    sections = tree_data.get("sections", [])
    total_sections = count_sections(sections)

    # ── Resolve source PDF path for the KB0 server ──────────────────────
    # Convention: KB0 serves PDFs from /app/pdfs/<filename> inside the
    # container.  Store this as source_pdf_path so the frontend can load
    # the PDF via GET /api/v2/pipeline1/jobs/{job_id}/source-pdf.
    source_pdf_name = metadata.get("source_pdf", "unknown.pdf")
    source_pdf_path = f"/app/pdfs/{source_pdf_name}"

    # ── Insert everything in a single transaction ────────────────────────
    try:
        with conn.cursor() as cur:
            # 1. Insert extraction job
            cur.execute("""
                INSERT INTO l2_extraction_jobs (
                    job_id, source_pdf, page_range, pipeline_version, l1_tag,
                    total_merged_spans, total_sections, total_pages,
                    alignment_confidence, l1_oracle_stats,
                    spans_pending, status, created_at, source_pdf_path
                ) VALUES (
                    %s, %s, %s, %s, %s,
                    %s, %s, %s,
                    %s, %s,
                    %s, 'PENDING_REVIEW', %s, %s
                )
            """, (
                job_id,
                source_pdf_name,
                json.dumps(metadata["page_range"]) if metadata.get("page_range") else None,
                metadata.get("pipeline_version", "V4.2.1"),
                metadata.get("l1_backend"),
                metadata.get("total_merged_spans", len(spans)),
                total_sections,
                tree_data.get("total_pages", 0),
                tree_data.get("alignment_confidence"),
                json.dumps(metadata.get("l1_oracle_stats", {})),
                len(spans),  # All spans start as pending
                metadata.get("created_at", datetime.now(timezone.utc).isoformat()),
                source_pdf_path,
            ))

            # 2. Batch insert merged spans
            if spans:
                span_values = []
                for s in spans:
                    # bbox: list[float] or None → JSONB
                    bbox_val = json.dumps(s["bbox"]) if s.get("bbox") else None
                    span_values.append((
                        s["id"],
                        job_id,
                        s["text"],
                        s["start"],                     # → start_offset
                        s["end"],                       # → end_offset
                        s["contributing_channels"],      # → TEXT[]
                        json.dumps(s.get("channel_confidences", {})),
                        s.get("merged_confidence", 0.0),
                        s.get("has_disagreement", False),
                        s.get("disagreement_detail"),
                        s.get("page_number"),
                        s.get("section_id"),
                        s.get("table_id"),
                        bbox_val,
                        s.get("surrounding_context"),
                        s.get("review_status", "PENDING"),
                    ))

                psycopg2.extras.execute_values(
                    cur,
                    """
                    INSERT INTO l2_merged_spans (
                        id, job_id, text, start_offset, end_offset,
                        contributing_channels, channel_confidences,
                        merged_confidence, has_disagreement, disagreement_detail,
                        page_number, section_id, table_id,
                        bbox, surrounding_context,
                        review_status
                    ) VALUES %s
                    """,
                    span_values,
                    template="(%s, %s, %s, %s, %s, %s::text[], %s::jsonb, %s, %s, %s, %s, %s, %s, %s::jsonb, %s, %s)",
                )

            # 3. Batch insert section passages
            if passages:
                passage_values = []
                for p in passages:
                    passage_values.append((
                        job_id,
                        p["section_id"],
                        p["heading"],
                        p.get("page_number"),
                        p.get("prose_text"),
                        p["span_ids"],          # → UUID[]
                        p["span_count"],
                        p["child_section_ids"],  # → TEXT[]
                        p.get("start_offset"),
                        p.get("end_offset"),
                    ))

                psycopg2.extras.execute_values(
                    cur,
                    """
                    INSERT INTO l2_section_passages (
                        job_id, section_id, heading, page_number,
                        prose_text, span_ids, span_count,
                        child_section_ids, start_offset, end_offset
                    ) VALUES %s
                    """,
                    passage_values,
                    template="(%s, %s, %s, %s, %s, %s::uuid[], %s, %s::text[], %s, %s)",
                )

            # 4. Insert guideline tree + normalized text
            cur.execute("""
                INSERT INTO l2_guideline_tree (job_id, tree_json, normalized_text)
                VALUES (%s, %s::jsonb, %s)
            """, (
                job_id,
                json.dumps(tree_data),
                normalized_text,
            ))

        conn.commit()

        print(f"  OK: Ingested job {job_id}")
        print(f"      {len(spans)} spans, {len(passages)} passages, "
              f"{tree_data.get('total_pages', 0)} pages")
        return True

    except Exception as e:
        conn.rollback()
        print(f"  ERROR: Failed to ingest job {job_id}: {e}")
        raise


# ─────────────────────────────────────────────────────────────────────────────
# CLI
# ─────────────────────────────────────────────────────────────────────────────

def find_job_dirs(parent: Path) -> list[Path]:
    """Find all directories containing job_metadata.json."""
    return sorted([
        d for d in parent.iterdir()
        if d.is_dir() and (d / "job_metadata.json").exists()
    ])


def main():
    parser = argparse.ArgumentParser(
        description="Ingest Pipeline 1 extraction artifacts into GCP KB0 database"
    )
    parser.add_argument(
        "path",
        type=Path,
        help="Path to a single job directory or parent directory (with --all)",
    )
    parser.add_argument(
        "--all",
        action="store_true",
        help="Ingest all job directories under the given parent path",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Show what would be ingested without writing to database",
    )

    args = parser.parse_args()

    # Determine which directories to process
    if args.all:
        job_dirs = find_job_dirs(args.path)
        if not job_dirs:
            print(f"No job directories found under {args.path}")
            sys.exit(1)
        print(f"Found {len(job_dirs)} job directories under {args.path}")
    else:
        if not args.path.is_dir():
            print(f"ERROR: {args.path} is not a directory")
            sys.exit(1)
        job_dirs = [args.path]

    if args.dry_run:
        print("\n--- DRY RUN (no database writes) ---")
        for d in job_dirs:
            meta = load_json(d / "job_metadata.json")
            spans_path = d / "merged_spans.json"
            span_count = len(load_json(spans_path)) if spans_path.exists() else 0
            print(f"  Would ingest: {meta['job_id']}  "
                  f"({meta.get('source_pdf', '?')}, {span_count} spans)")
        print(f"\nTotal: {len(job_dirs)} jobs")
        return

    # Connect and ingest
    conn = get_connection()
    print(f"Connected to {os.getenv('DB_HOST', '34.46.243.149')}:"
          f"{os.getenv('DB_PORT', '5433')}/{os.getenv('DB_NAME', 'canonical_facts')}")

    ingested = 0
    skipped = 0
    failed = 0

    try:
        for d in job_dirs:
            try:
                if ingest_job(d, conn):
                    ingested += 1
                else:
                    skipped += 1
            except Exception as e:
                failed += 1
                print(f"  FAILED: {d.name} — {e}")
    finally:
        conn.close()

    # Summary
    print(f"\n{'='*50}")
    print(f"Ingestion complete: {ingested} ingested, {skipped} skipped, {failed} failed")
    print(f"{'='*50}")

    if failed > 0:
        sys.exit(1)


if __name__ == "__main__":
    main()
