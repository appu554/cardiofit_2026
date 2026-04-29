"""
Pipeline 2 multi-layer loader (L1 / L2 / L2.5 / L4).

Loads the rest of the Pipeline 2 outputs that the L3 staging loader
didn't cover:

  L1   raw_spans.json          -> kb3_pipeline2_raw_spans
  L2   merged_spans.json       -> kb3_pipeline2_merged_spans
  L2.5 section_passages.json   -> kb3_pipeline2_sections
  L2.5 guideline_tree.json     -> kb3_pipeline2_tree
  L4   l4_rxnorm_corrections   -> kb3_pipeline2_rxnorm_corrections
  L4   l4_validation_report    -> kb3_pipeline2_validation_reports

Lands everything in KB-3 (kb3-postgres :5435) — the Guidelines KB —
which owns source-of-truth guideline content. Each row carries
job_id provenance back to the originating Pipeline 2 run so multiple
guidelines can coexist.

Additionally, applies L4 RxNorm corrections back to the 4 consumer
KB staging tables (kb1/kb4/kb16/kb20) to fix resolved_rxcui values
where the LLM was wrong but L4 found the correct code.

Usage
    python3 scripts/load_pipeline2_layers.py
    python3 scripts/load_pipeline2_layers.py --dry-run
"""

from __future__ import annotations

import argparse
import json
import logging
import sys
from pathlib import Path

import psycopg2
import psycopg2.extras

logging.basicConfig(level=logging.INFO, format="%(message)s")
log = logging.getLogger(__name__)

REPO_KB7 = Path(__file__).resolve().parent.parent
PIPELINE_OUTPUT = REPO_KB7.parent / "shared/tools/guideline-atomiser/data/output"
JOB_DIR = PIPELINE_OUTPUT / "v4/job_monkeyocr_df538e50-0170-4ef8-862d-5b0a7c48e4ff"

KB3_DSN = dict(host="localhost", port=5435, user="postgres",
               password="password", dbname="kb3_guidelines")

# Consumer KB staging targets for L4 correction propagation
STAGING_DSN = {
    "contextual": dict(host="localhost", port=5436, user="kb20_user",
                       password="kb20_password", dbname="kb_service_20"),
    "dosing":     dict(host="localhost", port=5481, user="kb1_user",
                       password="kb1_password", dbname="kb1_drug_rules"),
    "monitoring": dict(host="localhost", port=5446, user="kb16_user",
                       password="kb_password", dbname="kb_lab_interpretation"),
    "safety":     dict(host="localhost", port=5440, user="kb4_safety_user",
                       password="kb4_safety_password", dbname="kb4_patient_safety"),
}


# ---------- DDL ----------

DDL = """
CREATE TABLE IF NOT EXISTS kb3_pipeline2_jobs (
    job_id            TEXT          PRIMARY KEY,
    source_guideline  TEXT,
    pipeline_version  TEXT,
    extraction_method TEXT,
    metadata          JSONB         NOT NULL,
    loaded_at         TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS kb3_pipeline2_raw_spans (
    id                BIGSERIAL     PRIMARY KEY,
    job_id            TEXT          NOT NULL,
    channel           VARCHAR(2)    NOT NULL,
    span_id           TEXT,
    page_number       INTEGER,
    text              TEXT,
    span_data         JSONB         NOT NULL,
    loaded_at         TIMESTAMPTZ   NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_kb3_raw_spans_job ON kb3_pipeline2_raw_spans (job_id);
CREATE INDEX IF NOT EXISTS idx_kb3_raw_spans_channel ON kb3_pipeline2_raw_spans (job_id, channel);

CREATE TABLE IF NOT EXISTS kb3_pipeline2_merged_spans (
    id                BIGSERIAL     PRIMARY KEY,
    job_id            TEXT          NOT NULL,
    span_id           TEXT,
    page_number       INTEGER,
    text              TEXT,
    span_type         TEXT,
    span_data         JSONB         NOT NULL,
    loaded_at         TIMESTAMPTZ   NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_kb3_merged_spans_job ON kb3_pipeline2_merged_spans (job_id);

CREATE TABLE IF NOT EXISTS kb3_pipeline2_sections (
    id                BIGSERIAL     PRIMARY KEY,
    job_id            TEXT          NOT NULL,
    section_id        TEXT          NOT NULL,
    heading           TEXT,
    page_number       INTEGER,
    prose_text        TEXT,
    span_ids          TEXT[],
    span_count        INTEGER,
    child_section_ids TEXT[],
    start_offset      INTEGER,
    section_data      JSONB         NOT NULL,
    loaded_at         TIMESTAMPTZ   NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_kb3_sections_job ON kb3_pipeline2_sections (job_id);

CREATE TABLE IF NOT EXISTS kb3_pipeline2_tree (
    id                BIGSERIAL     PRIMARY KEY,
    job_id            TEXT          NOT NULL,
    tree_data         JSONB         NOT NULL,
    loaded_at         TIMESTAMPTZ   NOT NULL DEFAULT now(),
    UNIQUE (job_id)
);

CREATE TABLE IF NOT EXISTS kb3_pipeline2_rxnorm_corrections (
    id                BIGSERIAL     PRIMARY KEY,
    job_id            TEXT          NOT NULL,
    correction_type   VARCHAR(20)   NOT NULL,   -- CORRECTION | VALID | NOT_IN_KB7
    drug_name         TEXT,
    original_rxcui    VARCHAR(50),
    corrected_rxcui   VARCHAR(50),
    rationale         TEXT,
    correction_data   JSONB         NOT NULL,
    loaded_at         TIMESTAMPTZ   NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_kb3_rxnorm_corr_drug ON kb3_pipeline2_rxnorm_corrections (lower(drug_name));

CREATE TABLE IF NOT EXISTS kb3_pipeline2_validation_reports (
    id                BIGSERIAL     PRIMARY KEY,
    job_id            TEXT          NOT NULL,
    report_data       JSONB         NOT NULL,
    loaded_at         TIMESTAMPTZ   NOT NULL DEFAULT now()
);
"""


# ---------- Loaders ----------

def load_job_metadata(conn, job_id: str, metadata: dict) -> int:
    with conn.cursor() as cur:
        cur.execute(
            """
            INSERT INTO kb3_pipeline2_jobs
                (job_id, source_guideline, pipeline_version, extraction_method, metadata)
            VALUES (%s, %s, %s, %s, %s)
            ON CONFLICT (job_id) DO UPDATE SET
                metadata = EXCLUDED.metadata,
                loaded_at = now()
            """,
            (
                job_id,
                metadata.get("source_guideline") or metadata.get("guideline_name"),
                metadata.get("pipeline_version") or metadata.get("version"),
                metadata.get("extraction_method") or metadata.get("ocr_method"),
                psycopg2.extras.Json(metadata),
            ),
        )
    return 1


def load_raw_spans(conn, job_id: str, raw: dict) -> int:
    """L1 raw_spans is a dict keyed by channel; each channel is
    {count, error, spans} where spans is the actual span list."""
    n = 0
    with conn.cursor() as cur:
        cur.execute("DELETE FROM kb3_pipeline2_raw_spans WHERE job_id = %s", (job_id,))
        for channel, ch_data in raw.items():
            if isinstance(ch_data, dict):
                spans = ch_data.get("spans") or []
            elif isinstance(ch_data, list):
                spans = ch_data
            else:
                continue
            for s in spans:
                if not isinstance(s, dict):
                    continue
                cur.execute(
                    """
                    INSERT INTO kb3_pipeline2_raw_spans
                        (job_id, channel, span_id, page_number, text, span_data)
                    VALUES (%s, %s, %s, %s, %s, %s)
                    """,
                    (
                        job_id, channel,
                        s.get("span_id") or s.get("id"),
                        s.get("page_number") or s.get("page"),
                        s.get("text") or s.get("content"),
                        psycopg2.extras.Json(s),
                    ),
                )
                n += 1
    return n


def load_merged_spans(conn, job_id: str, merged: list) -> int:
    n = 0
    with conn.cursor() as cur:
        cur.execute("DELETE FROM kb3_pipeline2_merged_spans WHERE job_id = %s", (job_id,))
        psycopg2.extras.execute_values(
            cur,
            """
            INSERT INTO kb3_pipeline2_merged_spans
                (job_id, span_id, page_number, text, span_type, span_data)
            VALUES %s
            """,
            [
                (
                    job_id,
                    s.get("span_id") or s.get("id"),
                    s.get("page_number") or s.get("page"),
                    s.get("text") or s.get("content"),
                    s.get("span_type") or s.get("type"),
                    psycopg2.extras.Json(s),
                )
                for s in merged
                if isinstance(s, dict)
            ],
            page_size=500,
        )
        n = cur.rowcount
    return n


def load_sections(conn, job_id: str, sections: list) -> int:
    n = 0
    with conn.cursor() as cur:
        cur.execute("DELETE FROM kb3_pipeline2_sections WHERE job_id = %s", (job_id,))
        for s in sections:
            if not isinstance(s, dict):
                continue
            cur.execute(
                """
                INSERT INTO kb3_pipeline2_sections
                    (job_id, section_id, heading, page_number, prose_text,
                     span_ids, span_count, child_section_ids, start_offset, section_data)
                VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s,%s)
                """,
                (
                    job_id,
                    s.get("section_id") or "",
                    s.get("heading"),
                    s.get("page_number"),
                    s.get("prose_text"),
                    s.get("span_ids") or [],
                    s.get("span_count"),
                    s.get("child_section_ids") or [],
                    s.get("start_offset"),
                    psycopg2.extras.Json(s),
                ),
            )
            n += 1
    return n


def load_tree(conn, job_id: str, tree: dict) -> int:
    with conn.cursor() as cur:
        cur.execute(
            """
            INSERT INTO kb3_pipeline2_tree (job_id, tree_data)
            VALUES (%s, %s)
            ON CONFLICT (job_id) DO UPDATE SET
                tree_data = EXCLUDED.tree_data, loaded_at = now()
            """,
            (job_id, psycopg2.extras.Json(tree)),
        )
    return 1


def load_rxnorm_corrections(conn, job_id: str, corrections_doc: dict) -> int:
    """L4 file structure: {validation_summary, corrections[], valid_codes[], not_in_kb7[], root_cause_analysis}."""
    n = 0
    with conn.cursor() as cur:
        cur.execute("DELETE FROM kb3_pipeline2_rxnorm_corrections WHERE job_id = %s", (job_id,))
        for kind, key in (("CORRECTION", "corrections"),
                          ("VALID", "valid_codes"),
                          ("NOT_IN_KB7", "not_in_kb7")):
            for entry in corrections_doc.get(key, []) or []:
                if not isinstance(entry, dict):
                    continue
                cur.execute(
                    """
                    INSERT INTO kb3_pipeline2_rxnorm_corrections
                        (job_id, correction_type, drug_name,
                         original_rxcui, corrected_rxcui, rationale, correction_data)
                    VALUES (%s,%s,%s,%s,%s,%s,%s)
                    """,
                    (
                        job_id, kind,
                        entry.get("drug_name") or entry.get("drug"),
                        entry.get("original_rxcui") or entry.get("provided_rxcui") or entry.get("rxcui"),
                        entry.get("corrected_rxcui") or entry.get("correct_rxcui"),
                        entry.get("rationale") or entry.get("reason"),
                        psycopg2.extras.Json(entry),
                    ),
                )
                n += 1
    return n


def load_validation_report(conn, job_id: str, report: dict) -> int:
    with conn.cursor() as cur:
        cur.execute(
            "INSERT INTO kb3_pipeline2_validation_reports (job_id, report_data) VALUES (%s, %s)",
            (job_id, psycopg2.extras.Json(report)),
        )
    return 1


def apply_corrections_to_staging(corrections: list[dict]) -> dict[str, int]:
    """For each correction (drug_name -> corrected_rxcui), UPDATE the
    resolved_rxcui in every staging table where the drug matches."""
    if not corrections:
        return {}
    updates: dict[str, int] = {}
    for domain, dsn in STAGING_DSN.items():
        n = 0
        try:
            with psycopg2.connect(**dsn) as conn:
                conn.autocommit = True
                with conn.cursor() as cur:
                    for c in corrections:
                        drug = (c.get("drug_name") or c.get("drug") or "").strip()
                        new_rxcui = c.get("corrected_rxcui") or c.get("correct_rxcui")
                        if not drug or not new_rxcui:
                            continue
                        cur.execute(
                            """
                            UPDATE kb_l3_staging
                            SET resolved_rxcui = %s,
                                resolved_rxcui_source = 'L4_CORRECTION',
                                resolved_at = now()
                            WHERE lower(drug_name) = lower(%s)
                               OR lower(raw_json ->> 'drugName') = lower(%s)
                            """,
                            (new_rxcui, drug, drug),
                        )
                        n += cur.rowcount
        except Exception as e:
            log.warning("  L4 correction apply failed for %s: %s", domain, e)
        updates[domain] = n
    return updates


# ---------- Main ----------

def main() -> int:
    p = argparse.ArgumentParser(
        description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter
    )
    p.add_argument("--dry-run", action="store_true",
                   help="Discover + count only, no DB writes")
    args = p.parse_args()

    if not JOB_DIR.is_dir():
        log.error("Job dir not found: %s", JOB_DIR); return 2

    log.info("Job dir: %s", JOB_DIR.name)

    # Load file contents
    files = {
        "metadata":     JOB_DIR / "job_metadata.json",
        "raw_spans":    JOB_DIR / "raw_spans.json",
        "merged_spans": JOB_DIR / "merged_spans.json",
        "sections":     JOB_DIR / "section_passages.json",
        "tree":         JOB_DIR / "guideline_tree.json",
        "l4_corr":      PIPELINE_OUTPUT / "l4_rxnorm_corrections.json",
        "l4_valid":     PIPELINE_OUTPUT / "l4_validation_report.json",
    }
    docs = {}
    for k, p in files.items():
        if p.exists():
            try:
                docs[k] = json.loads(p.read_text())
                size = p.stat().st_size
                log.info("  loaded %-12s %.1f MB", k, size / (1024 * 1024))
            except Exception as e:
                log.warning("  %s parse error: %s", k, e)
        else:
            log.warning("  %s NOT FOUND at %s", k, p)

    job_id = (docs.get("metadata") or {}).get("job_id") or JOB_DIR.name
    log.info("  job_id: %s", job_id)

    if args.dry_run:
        log.info("DRY-RUN — would load:")
        if "raw_spans" in docs:
            total = 0
            for ch, v in docs["raw_spans"].items():
                if isinstance(v, dict):
                    total += len(v.get("spans") or [])
                elif isinstance(v, list):
                    total += len(v)
            log.info("  raw_spans: %d spans across %d channels", total, len(docs["raw_spans"]))
        if "merged_spans" in docs:
            log.info("  merged_spans: %d", len(docs["merged_spans"]) if isinstance(docs["merged_spans"], list) else 0)
        if "sections" in docs:
            log.info("  sections: %d", len(docs["sections"]) if isinstance(docs["sections"], list) else 0)
        if "tree" in docs:
            log.info("  tree: 1 nested JSON")
        if "l4_corr" in docs:
            n = sum(len(docs["l4_corr"].get(k, []) or []) for k in ("corrections","valid_codes","not_in_kb7"))
            log.info("  l4_rxnorm_corrections: %d entries", n)
        return 0

    # Load into KB-3
    log.info("Connecting to KB-3 (%s) ...", KB3_DSN["dbname"])
    with psycopg2.connect(**KB3_DSN) as conn, conn.cursor() as cur:
        cur.execute(DDL)
        conn.commit()

    with psycopg2.connect(**KB3_DSN) as conn:
        conn.autocommit = True
        if "metadata" in docs:
            n = load_job_metadata(conn, job_id, docs["metadata"])
            log.info("  job_metadata: %d", n)
        if "raw_spans" in docs:
            n = load_raw_spans(conn, job_id, docs["raw_spans"])
            log.info("  raw_spans: %d rows", n)
        if "merged_spans" in docs:
            n = load_merged_spans(conn, job_id, docs["merged_spans"])
            log.info("  merged_spans: %d rows", n)
        if "sections" in docs:
            n = load_sections(conn, job_id, docs["sections"])
            log.info("  sections: %d rows", n)
        if "tree" in docs:
            n = load_tree(conn, job_id, docs["tree"])
            log.info("  tree: %d row", n)
        if "l4_corr" in docs:
            n = load_rxnorm_corrections(conn, job_id, docs["l4_corr"])
            log.info("  rxnorm_corrections: %d rows", n)
        if "l4_valid" in docs:
            n = load_validation_report(conn, job_id, docs["l4_valid"])
            log.info("  validation_reports: %d row", n)

    # Apply L4 corrections back to consumer KBs' staging
    if "l4_corr" in docs:
        applied = apply_corrections_to_staging(docs["l4_corr"].get("corrections", []) or [])
        log.info("=" * 60)
        log.info("L4 CORRECTIONS APPLIED TO STAGING TABLES")
        log.info("=" * 60)
        for domain, n in applied.items():
            log.info("  %-12s rows updated: %d", domain, n)

    log.info("=" * 60)
    log.info("Pipeline 2 layer load complete for job %s", job_id)
    log.info("=" * 60)
    return 0


if __name__ == "__main__":
    sys.exit(main())
