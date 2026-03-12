#!/usr/bin/env python3
"""
Load Pipeline 1 V4.2.1 JSON output into kb0_local database.

Usage:
    python load_pipeline_output.py <data_dir> [--db-url <url>]

Default DB URL: postgresql://v4user:v4pass@localhost:5434/kb0_local
"""

import json
import sys
import os
import argparse
from pathlib import Path

try:
    import psycopg2
    from psycopg2.extras import execute_values, Json
except ImportError:
    print("Installing psycopg2-binary...")
    os.system(f"{sys.executable} -m pip install psycopg2-binary")
    import psycopg2
    from psycopg2.extras import execute_values, Json


DEFAULT_DB_URL = "postgresql://v4user:v4pass@localhost:5434/kb0_local"


def load_json(path: Path):
    with open(path, "r") as f:
        return json.load(f)


def load_text(path: Path) -> str:
    with open(path, "r") as f:
        return f.read()


def insert_job(cur, data_dir: Path, pdfs_dir: Path = None):
    """Insert l2_extraction_jobs from job_metadata.json."""
    meta = load_json(data_dir / "job_metadata.json")

    # Count unique pages from section_passages for total_pages
    passages = load_json(data_dir / "section_passages.json")
    page_numbers = set()
    for p in passages:
        if p.get("page_number") is not None:
            page_numbers.add(p["page_number"])
    total_pages = len(page_numbers)

    # Count sections
    tree = load_json(data_dir / "guideline_tree.json")
    total_sections = len(tree.get("sections", []))

    # Resolve source PDF path (look in data/pdfs/ sibling directory)
    source_pdf_path = None
    if pdfs_dir is None:
        pdfs_dir = data_dir.parent.parent / "pdfs"
    page_range = meta.get("page_range", "")
    # Try page-range-specific PDF first, then full PDF
    for pattern in [f"*pages*{page_range}*.pdf", f"*{meta['source_pdf'].replace('.pdf', '')}*pages*.pdf", meta["source_pdf"]]:
        matches = list(pdfs_dir.glob(pattern))
        if matches:
            source_pdf_path = str(matches[0].resolve())
            break
    if source_pdf_path:
        print(f"  Source PDF: {source_pdf_path}")
    else:
        print(f"  WARNING: No source PDF found in {pdfs_dir}")

    cur.execute("""
        INSERT INTO l2_extraction_jobs (
            job_id, source_pdf, page_range, pipeline_version,
            l1_tag, total_merged_spans, total_sections, total_pages,
            alignment_confidence, l1_oracle_stats,
            spans_pending, status, created_at, source_pdf_path
        ) VALUES (
            %s, %s, %s, %s,
            %s, %s, %s, %s,
            %s, %s,
            %s, %s, %s, %s
        )
        ON CONFLICT (job_id) DO UPDATE SET
            total_merged_spans = EXCLUDED.total_merged_spans,
            total_sections = EXCLUDED.total_sections,
            total_pages = EXCLUDED.total_pages,
            source_pdf_path = EXCLUDED.source_pdf_path,
            updated_at = NOW()
    """, (
        meta["job_id"],
        meta["source_pdf"],
        meta.get("page_range"),
        meta.get("pipeline_version", "4.2.1"),
        meta.get("l1_backend"),  # l1_backend -> l1_tag
        meta.get("total_merged_spans", 0),
        total_sections,
        total_pages,
        meta.get("alignment_confidence"),
        Json(meta.get("l1_oracle", {})),
        meta.get("total_merged_spans", 0),  # all pending initially
        "PENDING_REVIEW",
        meta.get("created_at"),
        source_pdf_path,
    ))
    print(f"  Job: {meta['job_id']} ({meta['source_pdf']})")
    return meta["job_id"]


def insert_merged_spans(cur, data_dir: Path, job_id: str):
    """Insert l2_merged_spans from merged_spans.json."""
    spans = load_json(data_dir / "merged_spans.json")
    print(f"  Loading {len(spans)} merged spans...")

    # Batch insert using execute_values for speed
    values = []
    for s in spans:
        values.append((
            s["id"],
            job_id,
            s["text"],
            s.get("start", s.get("start_offset", -1)),  # JSON uses "start"
            s.get("end", s.get("end_offset", -1)),       # JSON uses "end"
            s.get("contributing_channels", []),
            Json(s.get("channel_confidences", {})),
            s.get("merged_confidence", 0.0),
            s.get("has_disagreement", False),
            s.get("disagreement_detail"),
            s.get("page_number"),
            s.get("section_id"),
            s.get("table_id"),
            s.get("review_status", "PENDING"),
            s.get("reviewer_text"),
            s.get("reviewed_by"),
            s.get("reviewed_at"),
        ))

    execute_values(cur, """
        INSERT INTO l2_merged_spans (
            id, job_id, text, start_offset, end_offset,
            contributing_channels, channel_confidences, merged_confidence,
            has_disagreement, disagreement_detail,
            page_number, section_id, table_id,
            review_status, reviewer_text, reviewed_by, reviewed_at
        ) VALUES %s
        ON CONFLICT (id) DO NOTHING
    """, values, template="""(
        %s::uuid, %s::uuid, %s, %s, %s,
        %s::text[], %s::jsonb, %s,
        %s, %s,
        %s, %s, %s,
        %s, %s, %s, %s::timestamptz
    )""")
    print(f"  Inserted {len(spans)} spans")


def insert_section_passages(cur, data_dir: Path, job_id: str):
    """Insert l2_section_passages from section_passages.json."""
    passages = load_json(data_dir / "section_passages.json")
    print(f"  Loading {len(passages)} section passages...")

    values = []
    for p in passages:
        values.append((
            job_id,
            p["section_id"],
            p["heading"],
            p.get("page_number"),
            p.get("prose_text"),
            p.get("span_ids", []),
            p.get("span_count", len(p.get("span_ids", []))),
            p.get("child_section_ids", []),
            p.get("start_offset"),
            p.get("end_offset"),
        ))

    execute_values(cur, """
        INSERT INTO l2_section_passages (
            job_id, section_id, heading, page_number,
            prose_text, span_ids, span_count,
            child_section_ids, start_offset, end_offset
        ) VALUES %s
        ON CONFLICT (job_id, section_id) DO NOTHING
    """, values, template="""(
        %s::uuid, %s, %s, %s,
        %s, %s::uuid[], %s,
        %s::text[], %s, %s
    )""")
    print(f"  Inserted {len(passages)} passages")


def insert_guideline_tree(cur, data_dir: Path, job_id: str):
    """Insert l2_guideline_tree from guideline_tree.json + normalized_text.txt + highlight HTML."""
    tree_json = load_json(data_dir / "guideline_tree.json")

    normalized_text = None
    norm_path = data_dir / "normalized_text.txt"
    if norm_path.exists():
        normalized_text = load_text(norm_path)
        print(f"  Normalized text: {len(normalized_text)} chars")

    # Load pipeline-generated highlight HTML (if present)
    highlight_html = None
    html_files = list(data_dir.glob("highlighted_text_pages_*.html"))
    if html_files:
        highlight_html = load_text(html_files[0])
        print(f"  Highlight HTML: {len(highlight_html)} chars ({html_files[0].name})")

    cur.execute("""
        INSERT INTO l2_guideline_tree (job_id, tree_json, normalized_text, highlight_html)
        VALUES (%s, %s, %s, %s)
        ON CONFLICT (job_id) DO UPDATE SET
            tree_json = EXCLUDED.tree_json,
            normalized_text = EXCLUDED.normalized_text,
            highlight_html = EXCLUDED.highlight_html
    """, (
        job_id,
        Json(tree_json),
        normalized_text,
        highlight_html,
    ))
    print(f"  Guideline tree: {len(tree_json.get('sections', []))} sections")


def main():
    parser = argparse.ArgumentParser(description="Load pipeline output into kb0_local")
    parser.add_argument("data_dir", help="Path to pipeline output directory")
    parser.add_argument("--db-url", default=DEFAULT_DB_URL, help="Database URL")
    args = parser.parse_args()

    data_dir = Path(args.data_dir)
    if not (data_dir / "job_metadata.json").exists():
        print(f"ERROR: {data_dir / 'job_metadata.json'} not found")
        sys.exit(1)

    print(f"Connecting to: {args.db_url}")
    conn = psycopg2.connect(args.db_url)
    conn.autocommit = False

    try:
        cur = conn.cursor()

        print("\n[1/4] Inserting extraction job...")
        job_id = insert_job(cur, data_dir)

        print("\n[2/4] Inserting merged spans...")
        insert_merged_spans(cur, data_dir, job_id)

        print("\n[3/4] Inserting section passages...")
        insert_section_passages(cur, data_dir, job_id)

        print("\n[4/4] Inserting guideline tree...")
        insert_guideline_tree(cur, data_dir, job_id)

        conn.commit()
        print("\n=== ALL DATA LOADED SUCCESSFULLY ===")

        # Verify counts
        cur.execute("SELECT COUNT(*) FROM l2_merged_spans WHERE job_id = %s", (job_id,))
        span_count = cur.fetchone()[0]
        cur.execute("SELECT COUNT(*) FROM l2_section_passages WHERE job_id = %s", (job_id,))
        passage_count = cur.fetchone()[0]
        cur.execute("SELECT COUNT(*) FROM l2_guideline_tree WHERE job_id = %s", (job_id,))
        tree_count = cur.fetchone()[0]

        print(f"\nVerification:")
        print(f"  Spans:    {span_count}")
        print(f"  Passages: {passage_count}")
        print(f"  Tree:     {tree_count}")

        # Test the L3 view
        cur.execute("SELECT COUNT(*) FROM l2_passages_for_l3 WHERE job_id = %s", (job_id,))
        l3_count = cur.fetchone()[0]
        print(f"  L3 View:  {l3_count} passages")

    except Exception as e:
        conn.rollback()
        print(f"\nERROR: {e}")
        raise
    finally:
        conn.close()


if __name__ == "__main__":
    main()
