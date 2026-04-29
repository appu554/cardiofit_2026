"""
SNOMED CT-AU RF2 loader for KB-7 Postgres.

Strategy:
  1. Apply migration 013 (schema only, no indexes).
  2. COPY each RF2 Snapshot TSV into a TEMP staging table (all TEXT columns).
  3. INSERT-SELECT with type casts (effectiveTime YYYYMMDD -> DATE) into typed targets.
  4. Apply migration 014 (indexes, after rows are loaded).
  5. Record load metrics in kb7_snomed_load_log.

Run from kb-7-terminology directory:
    python3 scripts/load_snomed_au_rf2.py
"""

import logging
import sys
from pathlib import Path

import psycopg2

logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
)
log = logging.getLogger(__name__)

REPO_KB7 = Path(__file__).resolve().parent.parent
EXTRACT_ROOT = REPO_KB7 / "data/snomed/extracted/AU_20260430/SnomedCT_Release_AU1000036_20260430/Snapshot"
RELEASE_DATE = "2026-04-30"

DSN = dict(
    host="localhost",
    port=5457,
    user="postgres",
    password="password",
    dbname="kb_terminology",
)

# (file_relative_path, target_table, RF2 columns -> staging columns, INSERT-SELECT cast SQL)
TABLES = [
    {
        "file": "Terminology/sct2_Concept_Snapshot_AU1000036_20260430.txt",
        "target": "kb7_snomed_concept",
        "stage": "_stg_concept",
        "columns": ["id", "effective_time", "active", "module_id", "definition_status_id"],
        "select": """
            id::BIGINT,
            to_date(effective_time, 'YYYYMMDD'),
            active::SMALLINT,
            module_id::BIGINT,
            definition_status_id::BIGINT
        """,
    },
    {
        "file": "Terminology/sct2_Description_Snapshot-en-au_AU1000036_20260430.txt",
        "target": "kb7_snomed_description",
        "stage": "_stg_description",
        "columns": [
            "id", "effective_time", "active", "module_id", "concept_id",
            "language_code", "type_id", "term", "case_significance_id",
        ],
        "select": """
            id::BIGINT,
            to_date(effective_time, 'YYYYMMDD'),
            active::SMALLINT,
            module_id::BIGINT,
            concept_id::BIGINT,
            language_code,
            type_id::BIGINT,
            term,
            case_significance_id::BIGINT
        """,
    },
    {
        "file": "Terminology/sct2_Relationship_Snapshot_AU1000036_20260430.txt",
        "target": "kb7_snomed_relationship",
        "stage": "_stg_relationship",
        "columns": [
            "id", "effective_time", "active", "module_id", "source_id",
            "destination_id", "relationship_group", "type_id",
            "characteristic_type_id", "modifier_id",
        ],
        "select": """
            id::BIGINT,
            to_date(effective_time, 'YYYYMMDD'),
            active::SMALLINT,
            module_id::BIGINT,
            source_id::BIGINT,
            destination_id::BIGINT,
            relationship_group::INTEGER,
            type_id::BIGINT,
            characteristic_type_id::BIGINT,
            modifier_id::BIGINT
        """,
    },
    {
        "file": "Refset/Content/der2_Refset_SimpleSnapshot_AU1000036_20260430.txt",
        "target": "kb7_snomed_refset_simple",
        "stage": "_stg_refset_simple",
        "columns": [
            "id", "effective_time", "active", "module_id",
            "refset_id", "referenced_component_id",
        ],
        "select": """
            id::UUID,
            to_date(effective_time, 'YYYYMMDD'),
            active::SMALLINT,
            module_id::BIGINT,
            refset_id::BIGINT,
            referenced_component_id::BIGINT
        """,
    },
]


def apply_migration(conn, path: Path) -> None:
    log.info("Applying migration: %s", path.name)
    with conn.cursor() as cur:
        cur.execute(path.read_text())
    conn.commit()


def load_table(conn, spec: dict) -> int:
    src = EXTRACT_ROOT / spec["file"]
    if not src.exists():
        raise FileNotFoundError(src)

    target = spec["target"]
    stage = spec["stage"]
    cols = spec["columns"]
    select_sql = spec["select"]

    log.info("Loading %s -> %s (file: %s, %.0f MB)",
             src.name, target, src.name, src.stat().st_size / (1024 * 1024))

    with conn.cursor() as cur:
        cur.execute(f"DROP TABLE IF EXISTS {stage}")
        col_defs = ", ".join(f"{c} TEXT" for c in cols)
        cur.execute(f"CREATE UNLOGGED TABLE {stage} ({col_defs})")

        with src.open("r", encoding="utf-8") as f:
            f.readline()  # discard RF2 header row
            cur.copy_expert(
                f"COPY {stage} ({', '.join(cols)}) FROM STDIN "
                f"WITH (FORMAT text, DELIMITER E'\\t', NULL '')",
                f,
            )
        cur.execute(f"SELECT count(*) FROM {stage}")
        staged = cur.fetchone()[0]
        log.info("  staged %s rows", f"{staged:,}")

        cur.execute(f"TRUNCATE {target} CASCADE")
        cur.execute(
            f"INSERT INTO {target} ({', '.join(cols)}) "
            f"SELECT {select_sql} FROM {stage}"
        )
        cur.execute(f"SELECT count(*) FROM {target}")
        loaded = cur.fetchone()[0]
        cur.execute(f"DROP TABLE {stage}")

        cur.execute(
            "INSERT INTO kb7_snomed_load_log "
            "(release_date, source_file, table_name, rows_loaded) "
            "VALUES (%s, %s, %s, %s)",
            (RELEASE_DATE, src.name, target, loaded),
        )

    conn.commit()
    log.info("  loaded %s rows into %s", f"{loaded:,}", target)
    return loaded


def main() -> int:
    log.info("Connecting to kb7-postgres at %s:%s", DSN["host"], DSN["port"])
    conn = psycopg2.connect(**DSN)

    try:
        apply_migration(conn, REPO_KB7 / "migrations/013_snomed_au_rf2_schema.sql")

        total = 0
        for spec in TABLES:
            total += load_table(conn, spec)

        apply_migration(conn, REPO_KB7 / "migrations/014_snomed_au_rf2_indexes.sql")

        log.info("=" * 60)
        log.info("LOAD COMPLETE: %s rows across %d tables", f"{total:,}", len(TABLES))
        log.info("=" * 60)

        with conn.cursor() as cur:
            cur.execute("""
                SELECT module_id, count(*) AS active_concepts
                FROM kb7_snomed_concept
                WHERE active = 1
                GROUP BY module_id
                ORDER BY active_concepts DESC
            """)
            log.info("Active concept counts by module:")
            for module_id, n in cur.fetchall():
                label = {
                    900000000000207008: "SNOMED Core (International)",
                    32506021000036107:  "SNOMED CT-AU",
                    900000000000012004: "SNOMED Model Component",
                }.get(module_id, "")
                log.info("  module=%s  active=%s  %s", module_id, f"{n:,}", label)

    except Exception:
        conn.rollback()
        raise
    finally:
        conn.close()
    return 0


if __name__ == "__main__":
    sys.exit(main())
