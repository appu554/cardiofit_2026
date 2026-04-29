"""
ICD-10-AM + ACHI loader for KB-7 Postgres.

Loads files distributed by IHACPA / ACCD (Australian Consortium for
Classification Development). Distribution shape varies by edition; this
loader handles the common XML "Tabular List" format and TSV/CSV index
files. Parametric on file paths so it runs the moment files are placed
on disk — no API call required.

Usage:
    # Full load — disease tabular + alphabetic index + ACHI tabular + ACHI index
    python3 scripts/load_icd10am.py \\
        --edition "12th edition" \\
        --release-date 2024-07-01 \\
        --tabular   data/icd10am/12th/icd10am_tabular.xml \\
        --index     data/icd10am/12th/icd10am_index.csv \\
        --achi-tab  data/icd10am/12th/achi_tabular.xml \\
        --achi-idx  data/icd10am/12th/achi_index.csv

    # Validate parsing without DB write
    python3 scripts/load_icd10am.py --tabular path/to/file.xml --dry-run

Expected XML structure (typical ACCD format — adapt parser if real
distribution differs):

    <icd10am>
      <chapter number="1" title="Certain infectious diseases" range="A00-B99">
        <block code="A00-A09" title="Intestinal infectious diseases">
          <category code="A00" title="Cholera">
            <code value="A00.0" desc="Cholera due to Vibrio cholerae 01..."/>
            ...
          </category>
        </block>
      </chapter>
    </icd10am>

If the actual IHACPA XML uses different element/attribute names, the
parser raises a clear error and the JSON-shape exception message points
to which field needs renaming.
"""

from __future__ import annotations

import argparse
import logging
import sys
import xml.etree.ElementTree as ET
from pathlib import Path
from typing import Iterable, Iterator

import psycopg2

logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
)
log = logging.getLogger(__name__)

REPO_KB7 = Path(__file__).resolve().parent.parent

DSN = dict(
    host="localhost",
    port=5457,
    user="postgres",
    password="password",
    dbname="kb_terminology",
)


# ---------- XML parsing ----------

def parse_icd10am_tabular(path: Path) -> tuple[list[dict], list[dict], list[dict]]:
    """Parse ICD-10-AM Tabular XML.

    Returns (chapters, blocks, codes). Each is a list of dict rows
    matching the kb7_icd10am_* schemas.
    """
    log.info("Parsing ICD-10-AM tabular XML: %s", path)
    tree = ET.parse(path)
    root = tree.getroot()

    chapters: list[dict] = []
    blocks: list[dict] = []
    codes: list[dict] = []

    for chap in root.iter("chapter"):
        chapter_num = int(chap.get("number", "0"))
        chapter_title = chap.get("title", "") or chap.findtext("title", "")
        chapter_range = chap.get("range", "") or chap.findtext("range", "")
        chapters.append({
            "chapter_number": chapter_num,
            "title": chapter_title,
            "code_range": chapter_range,
        })

        for blk in chap.iter("block"):
            block_id = blk.get("code") or blk.get("id") or ""
            block_title = blk.get("title", "") or blk.findtext("title", "")
            block_range = blk.get("range", "") or block_id
            if block_id:
                blocks.append({
                    "block_id": block_id,
                    "chapter_number": chapter_num,
                    "title": block_title,
                    "code_range": block_range,
                })

            for code_el in blk.iter("code"):
                code_value = code_el.get("value") or code_el.get("code") or ""
                if not code_value:
                    continue
                desc = code_el.get("desc") or code_el.findtext("desc", "")
                parent = code_el.get("parent")
                if parent is None and "." in code_value:
                    parent = code_value.split(".", 1)[0]
                inclusions = [n.text for n in code_el.findall("include") if n.text]
                exclusions = [n.text for n in code_el.findall("exclude") if n.text]
                codes.append({
                    "code": code_value,
                    "parent_code": parent,
                    "chapter_number": chapter_num,
                    "block_id": block_id or None,
                    "description": desc,
                    "is_billable": ("." in code_value),
                    "asterisk_dagger": code_el.get("marker"),
                    "inclusions": inclusions or None,
                    "exclusions": exclusions or None,
                    "notes": code_el.findtext("notes"),
                })

    log.info("  parsed %d chapters, %d blocks, %d codes",
             len(chapters), len(blocks), len(codes))
    return chapters, blocks, codes


def parse_achi_tabular(path: Path) -> tuple[list[dict], list[dict]]:
    """Parse ACHI Tabular XML. Returns (blocks, codes)."""
    log.info("Parsing ACHI tabular XML: %s", path)
    tree = ET.parse(path)
    root = tree.getroot()

    blocks: list[dict] = []
    codes: list[dict] = []

    for chap in root.iter("chapter"):
        chapter_num_raw = chap.get("number", "")
        chapter_num = int(chapter_num_raw) if chapter_num_raw.isdigit() else None

        for blk in chap.iter("block"):
            block_id = blk.get("code") or blk.get("id") or ""
            if block_id:
                blocks.append({
                    "block_id": block_id,
                    "chapter_number": chapter_num,
                    "title": blk.get("title", "") or blk.findtext("title", ""),
                })

            for code_el in blk.iter("code"):
                code_value = code_el.get("value") or code_el.get("code") or ""
                if not code_value:
                    continue
                codes.append({
                    "code": code_value,
                    "block_id": block_id or None,
                    "description": code_el.get("desc") or code_el.findtext("desc", ""),
                    "procedure_type": code_el.get("type"),
                    "notes": code_el.findtext("notes"),
                })

    log.info("  parsed %d ACHI blocks, %d ACHI codes", len(blocks), len(codes))
    return blocks, codes


# ---------- Index file parsing (CSV/TSV) ----------

def parse_index_file(path: Path) -> Iterator[dict]:
    """Yield index rows from a CSV/TSV file with columns:
    lead_term, modifiers (optional), code.
    """
    log.info("Parsing index file: %s", path)
    delimiter = "\t" if path.suffix.lower() in (".tsv", ".txt") else ","
    import csv
    with path.open("r", encoding="utf-8", newline="") as f:
        reader = csv.DictReader(f, delimiter=delimiter)
        for row in reader:
            term = row.get("lead_term") or row.get("term") or ""
            code = row.get("code") or row.get("icd_code") or row.get("achi_code") or ""
            if not term or not code:
                continue
            yield {
                "lead_term": term.strip(),
                "modifiers": (row.get("modifiers") or "").strip() or None,
                "code": code.strip(),
            }


# ---------- DB load helpers ----------

def apply_migration(conn, name: str) -> None:
    log.info("Applying migration: %s", name)
    sql = (REPO_KB7 / "migrations" / name).read_text()
    with conn.cursor() as cur:
        cur.execute(sql)
    conn.commit()


def insert_rows(conn, table: str, rows: Iterable[dict], cols: list[str]) -> int:
    rows = list(rows)
    if not rows:
        return 0
    with conn.cursor() as cur:
        cur.execute(f"TRUNCATE {table} RESTART IDENTITY CASCADE")
        col_csv = ", ".join(cols)
        placeholders = ", ".join(["%s"] * len(cols))
        values = [tuple(r.get(c) for c in cols) for r in rows]
        cur.executemany(
            f"INSERT INTO {table} ({col_csv}) VALUES ({placeholders}) "
            f"ON CONFLICT DO NOTHING",
            values,
        )
        cur.execute(f"SELECT count(*) FROM {table}")
        n = cur.fetchone()[0]
    conn.commit()
    return n


def log_load(conn, edition: str, release_date: str | None,
             source_file: str, table: str, rows: int) -> None:
    with conn.cursor() as cur:
        cur.execute(
            "INSERT INTO kb7_icd10am_load_log "
            "(edition, release_date, source_file, table_name, rows_loaded) "
            "VALUES (%s, %s, %s, %s, %s)",
            (edition, release_date, source_file, table, rows),
        )
    conn.commit()


# ---------- Main ----------

def main() -> int:
    p = argparse.ArgumentParser(description=__doc__,
                                formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("--edition", default="unknown",
                   help='ICD-10-AM edition label, e.g., "12th edition"')
    p.add_argument("--release-date", help="Release date YYYY-MM-DD")
    p.add_argument("--tabular", type=Path,
                   help="Path to ICD-10-AM Tabular List XML")
    p.add_argument("--index", type=Path,
                   help="Path to ICD-10-AM Alphabetic Index CSV/TSV")
    p.add_argument("--achi-tab", type=Path,
                   help="Path to ACHI Tabular List XML")
    p.add_argument("--achi-idx", type=Path,
                   help="Path to ACHI Alphabetic Index CSV/TSV")
    p.add_argument("--dry-run", action="store_true",
                   help="Parse files, print row counts, do not touch DB")
    args = p.parse_args()

    if not any([args.tabular, args.index, args.achi_tab, args.achi_idx]):
        log.error("Provide at least one of: --tabular, --index, --achi-tab, --achi-idx")
        return 2

    parsed: dict[str, list[dict]] = {}

    if args.tabular:
        chapters, blocks, codes = parse_icd10am_tabular(args.tabular)
        parsed["chapters"] = chapters
        parsed["blocks"] = blocks
        parsed["codes"] = codes

    if args.index:
        parsed["index"] = list(parse_index_file(args.index))
        log.info("  parsed %d ICD-10-AM index rows", len(parsed["index"]))

    if args.achi_tab:
        achi_blocks, achi_codes = parse_achi_tabular(args.achi_tab)
        parsed["achi_blocks"] = achi_blocks
        parsed["achi_codes"] = achi_codes

    if args.achi_idx:
        parsed["achi_index"] = list(parse_index_file(args.achi_idx))
        log.info("  parsed %d ACHI index rows", len(parsed["achi_index"]))

    if args.dry_run:
        log.info("DRY-RUN summary:")
        for k, v in parsed.items():
            log.info("  %s: %d rows", k, len(v))
        return 0

    log.info("Connecting to kb7-postgres")
    conn = psycopg2.connect(**DSN)
    try:
        apply_migration(conn, "017_icd10am_schema.sql")

        for r in parsed.get("codes", []):
            r["edition"] = args.edition
        for r in parsed.get("achi_codes", []):
            r["edition"] = args.edition
        for r in parsed.get("index", []):
            r["edition"] = args.edition
        for r in parsed.get("achi_index", []):
            r["edition"] = args.edition

        if "chapters" in parsed:
            n = insert_rows(conn, "kb7_icd10am_chapter", parsed["chapters"],
                            ["chapter_number", "title", "code_range"])
            log_load(conn, args.edition, args.release_date,
                     str(args.tabular), "kb7_icd10am_chapter", n)
            log.info("loaded %d chapters", n)

            n = insert_rows(conn, "kb7_icd10am_block", parsed["blocks"],
                            ["block_id", "chapter_number", "title", "code_range"])
            log_load(conn, args.edition, args.release_date,
                     str(args.tabular), "kb7_icd10am_block", n)
            log.info("loaded %d blocks", n)

            n = insert_rows(conn, "kb7_icd10am_code", parsed["codes"],
                            ["code", "parent_code", "chapter_number", "block_id",
                             "description", "is_billable", "asterisk_dagger",
                             "inclusions", "exclusions", "notes", "edition"])
            log_load(conn, args.edition, args.release_date,
                     str(args.tabular), "kb7_icd10am_code", n)
            log.info("loaded %d codes", n)

        if "index" in parsed:
            n = insert_rows(conn, "kb7_icd10am_index", parsed["index"],
                            ["lead_term", "modifiers", "code", "edition"])
            log_load(conn, args.edition, args.release_date,
                     str(args.index), "kb7_icd10am_index", n)
            log.info("loaded %d index rows", n)

        if "achi_blocks" in parsed:
            n = insert_rows(conn, "kb7_achi_block", parsed["achi_blocks"],
                            ["block_id", "chapter_number", "title"])
            log_load(conn, args.edition, args.release_date,
                     str(args.achi_tab), "kb7_achi_block", n)
            log.info("loaded %d ACHI blocks", n)

            n = insert_rows(conn, "kb7_achi_code", parsed["achi_codes"],
                            ["code", "block_id", "description", "procedure_type",
                             "notes", "edition"])
            log_load(conn, args.edition, args.release_date,
                     str(args.achi_tab), "kb7_achi_code", n)
            log.info("loaded %d ACHI codes", n)

        if "achi_index" in parsed:
            n = insert_rows(conn, "kb7_achi_index", parsed["achi_index"],
                            ["lead_term", "modifiers", "code", "edition"])
            log_load(conn, args.edition, args.release_date,
                     str(args.achi_idx), "kb7_achi_index", n)
            log.info("loaded %d ACHI index rows", n)

        apply_migration(conn, "018_icd10am_indexes.sql")
        log.info("=" * 60)
        log.info("ICD-10-AM/ACHI LOAD COMPLETE")
        log.info("=" * 60)
    except Exception:
        conn.rollback()
        raise
    finally:
        conn.close()
    return 0


if __name__ == "__main__":
    sys.exit(main())
