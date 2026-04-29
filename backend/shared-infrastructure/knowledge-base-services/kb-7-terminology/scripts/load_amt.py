"""
AMT (Australian Medicines Terminology) loader for KB-7 Postgres.

AMT TSV is a denormalized flat file: one row per CTPP carrying all 6
hierarchy levels (CTPP, TPP, TPUU, TP, MPP, MPUU, MP) inline.

Strategy:
  1. Apply migration 015 (schema only, no indexes).
  2. COPY TSV into a TEMP staging table (all TEXT columns).
  3. INSERT-SELECT with type casts into kb7_amt_pack
     (handling empty ARTG_ID values as NULL).
  4. Apply migration 016 (indexes).
  5. Record load metrics.

Run from kb-7-terminology directory:
    python3 scripts/load_amt.py
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
AMT_FILE = REPO_KB7 / "data/amt/amt-20260430.tsv"
RELEASE_DATE = "2026-04-30"

DSN = dict(
    host="localhost",
    port=5457,
    user="postgres",
    password="password",
    dbname="kb_terminology",
)

# Header line in the AMT TSV (verified against the file)
HEADER_FIELDS = [
    "CTPP SCTID", "CTPP PT", "ARTG_ID",
    "TPP SCTID", "TPP PT",
    "TPUU SCTID", "TPUU PT",
    "TPP TP SCTID", "TPP TP PT",
    "TPUU TP SCTID", "TPUU TP PT",
    "MPP SCTID", "MPP PT",
    "MPUU SCTID", "MPUU PT",
    "MP SCTID", "MP PT",
]

# Staging columns (one TEXT per TSV column, in same order)
STAGE_COLS = [
    "ctpp_sctid", "ctpp_pt", "artg_id",
    "tpp_sctid", "tpp_pt",
    "tpuu_sctid", "tpuu_pt",
    "tpp_tp_sctid", "tpp_tp_pt",
    "tpuu_tp_sctid", "tpuu_tp_pt",
    "mpp_sctid", "mpp_pt",
    "mpuu_sctid", "mpuu_pt",
    "mp_sctid", "mp_pt",
]


def apply_migration(conn, path: Path) -> None:
    log.info("Applying migration: %s", path.name)
    with conn.cursor() as cur:
        cur.execute(path.read_text())
    conn.commit()


def load(conn) -> int:
    if not AMT_FILE.exists():
        raise FileNotFoundError(AMT_FILE)

    log.info("Loading %s (%.1f MB)", AMT_FILE.name, AMT_FILE.stat().st_size / (1024 * 1024))

    with conn.cursor() as cur:
        cur.execute("DROP TABLE IF EXISTS _stg_amt")
        col_defs = ", ".join(f"{c} TEXT" for c in STAGE_COLS)
        cur.execute(f"CREATE UNLOGGED TABLE _stg_amt ({col_defs})")

        with AMT_FILE.open("r", encoding="utf-8") as f:
            header = f.readline().rstrip("\n").split("\t")
            if header != HEADER_FIELDS:
                raise RuntimeError(
                    f"Unexpected AMT TSV header. "
                    f"Expected {HEADER_FIELDS}, got {header}"
                )
            cur.copy_expert(
                f"COPY _stg_amt ({', '.join(STAGE_COLS)}) FROM STDIN "
                f"WITH (FORMAT text, DELIMITER E'\\t', NULL '')",
                f,
            )
        cur.execute("SELECT count(*) FROM _stg_amt")
        staged = cur.fetchone()[0]
        log.info("  staged %s rows", f"{staged:,}")

        cur.execute("TRUNCATE kb7_amt_pack")
        cur.execute(
            """
            INSERT INTO kb7_amt_pack (
                ctpp_sctid, ctpp_pt, artg_id,
                tpp_sctid, tpp_pt,
                tpuu_sctid, tpuu_pt,
                tpp_tp_sctid, tpp_tp_pt,
                tpuu_tp_sctid, tpuu_tp_pt,
                mpp_sctid, mpp_pt,
                mpuu_sctid, mpuu_pt,
                mp_sctid, mp_pt
            )
            SELECT
                ctpp_sctid::BIGINT,    ctpp_pt,
                NULLIF(artg_id, '')::BIGINT,
                tpp_sctid::BIGINT,     tpp_pt,
                tpuu_sctid::BIGINT,    tpuu_pt,
                tpp_tp_sctid::BIGINT,  tpp_tp_pt,
                tpuu_tp_sctid::BIGINT, tpuu_tp_pt,
                mpp_sctid::BIGINT,     mpp_pt,
                mpuu_sctid::BIGINT,    mpuu_pt,
                mp_sctid::BIGINT,      mp_pt
            FROM _stg_amt
            """
        )
        cur.execute("SELECT count(*) FROM kb7_amt_pack")
        loaded = cur.fetchone()[0]
        cur.execute("DROP TABLE _stg_amt")

        cur.execute(
            "INSERT INTO kb7_amt_load_log (release_date, source_file, rows_loaded) "
            "VALUES (%s, %s, %s)",
            (RELEASE_DATE, AMT_FILE.name, loaded),
        )

    conn.commit()
    log.info("  loaded %s rows into kb7_amt_pack", f"{loaded:,}")
    return loaded


def main() -> int:
    log.info("Connecting to kb7-postgres at %s:%s", DSN["host"], DSN["port"])
    conn = psycopg2.connect(**DSN)

    try:
        apply_migration(conn, REPO_KB7 / "migrations/015_amt_schema.sql")
        loaded = load(conn)
        apply_migration(conn, REPO_KB7 / "migrations/016_amt_indexes.sql")

        log.info("=" * 60)
        log.info("AMT LOAD COMPLETE: %s rows", f"{loaded:,}")
        log.info("=" * 60)

        with conn.cursor() as cur:
            cur.execute("""
                SELECT count(DISTINCT mp_sctid)   AS unique_substances,
                       count(DISTINCT tpp_tp_sctid) AS unique_brands,
                       count(DISTINCT mpuu_sctid) AS unique_generic_uous,
                       count(*)                   AS total_packs,
                       count(artg_id)             AS packs_with_artg_id
                FROM kb7_amt_pack
            """)
            row = cur.fetchone()
            log.info("AMT inventory:")
            log.info("  unique substances (MP):       %s", f"{row[0]:,}")
            log.info("  unique brands (TP):           %s", f"{row[1]:,}")
            log.info("  unique generic UoUs (MPUU):   %s", f"{row[2]:,}")
            log.info("  total packs (CTPP):           %s", f"{row[3]:,}")
            log.info("  packs with ARTG_ID:           %s", f"{row[4]:,}")
    except Exception:
        conn.rollback()
        raise
    finally:
        conn.close()
    return 0


if __name__ == "__main__":
    sys.exit(main())
