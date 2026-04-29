"""
PBS API CSV bundle relational loader (Wave 2 completion).

Loads the 12 relational CSVs from the monthly PBS API bundle into the
kb6_pbs_rel_* tables. Target tables match CSV grain 1:1 (no joins, no
denormalization at load time). Idempotent: TRUNCATE + COPY each table.

Why a separate loader:
  - load_pbs.py is item-centric (one row per PBS item, with derived
    summary flags into kb6_pbs_authorities/restrictions/etc.).
  - This loader is graph-centric: full restriction text, prescribing
    text bodies, criteria/parameter trees, ATC dictionary.

Usage:
    cd kb-6-formulary
    python3 scripts/load_pbs_relational.py
    python3 scripts/load_pbs_relational.py --csv-dir data/pbs/extracted/tables_as_csv
    python3 scripts/load_pbs_relational.py --apply-migrations
    python3 scripts/load_pbs_relational.py --only restrictions,prescribing-texts

The default CSV directory is data/pbs/extracted/tables_as_csv (where the
April 2026 PBS-API-CSV-files.zip is unpacked).
"""

from __future__ import annotations

import argparse
import csv
import logging
import sys
from io import StringIO
from pathlib import Path

import psycopg2

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)-7s %(message)s")
log = logging.getLogger(__name__)

KB6 = Path(__file__).resolve().parent.parent
DEFAULT_CSV_DIR = KB6 / "data" / "pbs" / "extracted" / "tables_as_csv"

KB6_DSN = dict(
    host="localhost", port=5447, user="kb6_admin",
    password="kb6_secure_password", dbname="kb_formulary",
)

# CSV → (target_table, [(csv_col, db_col, transform)])
# transform: "text" | "int" | "bool_yn" | "date" | "numeric" | None (passthrough)
LOADERS: dict[str, tuple[str, list[tuple[str, str, str | None]]]] = {
    "atc-codes.csv": ("kb6_pbs_rel_atc_codes", [
        ("atc_code",        "atc_code",        "text"),
        ("atc_description", "atc_description", "text"),
        ("atc_level",       "atc_level",       "int"),
        ("atc_parent_code", "atc_parent_code", "text"),
        ("schedule_code",   "schedule_code",   "int"),
    ]),
    "item-atc-relationships.csv": ("kb6_pbs_rel_item_atc", [
        ("atc_code",         "atc_code",         "text"),
        ("pbs_code",         "pbs_code",         "text"),
        ("atc_priority_pct", "atc_priority_pct", "numeric"),
        ("schedule_code",    "schedule_code",    "int"),
    ]),
    "prescribers.csv": ("kb6_pbs_rel_prescribers", [
        ("pbs_code",        "pbs_code",        "text"),
        ("prescriber_code", "prescriber_code", "text"),
        ("prescriber_type", "prescriber_type", "text"),
        ("schedule_code",   "schedule_code",   "int"),
    ]),
    "indications.csv": ("kb6_pbs_rel_indications", [
        ("indication_prescribing_txt_id", "indication_prescribing_txt_id", "int"),
        ("condition",                     "condition",                     "text"),
        ("episodicity",                   "episodicity",                   "text"),
        ("severity",                      "severity",                      "text"),
        ("schedule_code",                 "schedule_code",                 "int"),
    ]),
    "restrictions.csv": ("kb6_pbs_rel_restrictions", [
        ("res_code",                   "res_code",                   "text"),
        ("treatment_phase",            "treatment_phase",            "text"),
        ("authority_method",           "authority_method",           "text"),
        ("treatment_of_code",          "treatment_of_code",          "text"),
        ("restriction_number",         "restriction_number",         "text"),
        ("li_html_text",               "li_html_text",               "text"),
        ("schedule_html_text",         "schedule_html_text",         "text"),
        ("note_indicator",             "note_indicator",             "bool_yn"),
        ("caution_indicator",          "caution_indicator",          "bool_yn"),
        ("complex_authority_rqrd_ind", "complex_authority_rqrd_ind", "bool_yn"),
        ("assessment_type_code",       "assessment_type_code",       "text"),
        ("criteria_relationship",      "criteria_relationship",      "text"),
        ("variation_rule_applied",     "variation_rule_applied",     "text"),
        ("first_listing_date",         "first_listing_date",         "date"),
        ("written_authority_required", "written_authority_required", "bool_yn"),
        ("schedule_code",              "schedule_code",              "int"),
    ]),
    "prescribing-texts.csv": ("kb6_pbs_rel_prescribing_texts", [
        ("prescribing_txt_id",         "prescribing_txt_id",         "int"),
        ("prescribing_type",           "prescribing_type",           "text"),
        ("prescribing_txt",            "prescribing_txt",            "text"),
        ("prscrbg_txt_html",           "prscrbg_txt_html",           "text"),
        ("complex_authority_rqrd_ind", "complex_authority_rqrd_ind", "bool_yn"),
        ("assessment_type_code",       "assessment_type_code",       "text"),
        ("apply_to_increase_mq_flag",  "apply_to_increase_mq_flag",  "bool_yn"),
        ("apply_to_increase_nr_flag",  "apply_to_increase_nr_flag",  "bool_yn"),
        ("schedule_code",              "schedule_code",              "int"),
    ]),
    "item-restriction-relationships.csv": ("kb6_pbs_rel_item_restrictions", [
        ("res_code",              "res_code",              "text"),
        ("pbs_code",              "pbs_code",              "text"),
        ("benefit_type_code",     "benefit_type_code",     "text"),
        ("restriction_indicator", "restriction_indicator", "text"),
        ("res_position",          "res_position",          "int"),
        ("schedule_code",         "schedule_code",         "int"),
    ]),
    "item-prescribing-text-relationships.csv": ("kb6_pbs_rel_item_prescribing_texts", [
        ("pbs_code",           "pbs_code",           "text"),
        ("prescribing_txt_id", "prescribing_txt_id", "int"),
        ("pt_position",        "pt_position",        "int"),
        ("schedule_code",      "schedule_code",      "int"),
    ]),
    "criteria.csv": ("kb6_pbs_rel_criteria", [
        ("criteria_prescribing_txt_id", "criteria_prescribing_txt_id", "int"),
        ("criteria_type",               "criteria_type",               "text"),
        ("parameter_relationship",      "parameter_relationship",      "text"),
        ("schedule_code",               "schedule_code",               "int"),
    ]),
    "parameters.csv": ("kb6_pbs_rel_parameters", [
        ("assessment_type",              "assessment_type",              "text"),
        ("parameter_prescribing_txt_id", "parameter_prescribing_txt_id", "int"),
        ("parameter_type",               "parameter_type",               "text"),
        ("schedule_code",                "schedule_code",                "int"),
    ]),
    "criteria-parameter-relationships.csv": ("kb6_pbs_rel_criteria_parameters", [
        ("criteria_prescribing_txt_id",  "criteria_prescribing_txt_id",  "int"),
        ("parameter_prescribing_txt_id", "parameter_prescribing_txt_id", "int"),
        ("pt_position",                  "pt_position",                  "int"),
        ("schedule_code",                "schedule_code",                "int"),
    ]),
    "programs.csv": ("kb6_pbs_rel_programs", [
        ("program_code",  "program_code",  "text"),
        ("program_title", "program_title", "text"),
        ("schedule_code", "schedule_code", "int"),
    ]),
}


def _coerce(val: str, kind: str | None) -> str:
    """Coerce a CSV cell value into the COPY-compatible form for the target type.

    COPY accepts \\N for NULL. We emit empty fields as \\N so PG casts cleanly
    into INT / DATE / BOOLEAN columns.
    """
    if val is None or val == "":
        return r"\N"
    if kind == "int":
        v = val.strip()
        return v if v.lstrip("-").isdigit() else r"\N"
    if kind == "numeric":
        v = val.strip()
        try:
            float(v)
            return v
        except ValueError:
            return r"\N"
    if kind == "bool_yn":
        v = val.strip().upper()
        if v in ("Y", "YES", "TRUE", "T", "1"):  return "true"
        if v in ("N", "NO", "FALSE", "F", "0"):  return "false"
        return r"\N"
    if kind == "date":
        v = val.strip()
        # PBS uses YYYY-MM-DD; pass through.
        return v if len(v) >= 10 else r"\N"
    return val


def load_one_csv(conn, csv_path: Path, target_table: str,
                 columns: list[tuple[str, str, str | None]]) -> tuple[int, int | None]:
    """TRUNCATE the target table, then COPY rows from the CSV.

    Returns (rows_loaded, schedule_code) where schedule_code is the most
    recent schedule_code seen in the file (used by the load log).
    """
    csv_cols = [c for c, _, _ in columns]
    db_cols = [d for _, d, _ in columns]
    kinds = [k for _, _, k in columns]

    # Stream: read CSV → emit COPY-compatible TSV lines
    buf = StringIO()
    n = 0
    schedule_code: int | None = None
    with csv_path.open("r", encoding="utf-8", newline="") as f:
        reader = csv.DictReader(f)
        # Sanity-check header
        missing = [c for c in csv_cols if c not in (reader.fieldnames or [])]
        if missing:
            raise RuntimeError(
                f"{csv_path.name}: missing expected columns {missing}; "
                f"got {reader.fieldnames}"
            )
        for row in reader:
            cells = [_coerce(row.get(c, ""), k) for c, k in zip(csv_cols, kinds)]
            # TSV-encode; tabs/newlines/backslashes inside text bodies must
            # be escaped to survive COPY's tab-delimited parser.
            cells = [
                c.replace("\\", "\\\\").replace("\t", "\\t").replace("\n", "\\n")
                  .replace("\r", "\\r")
                if c != r"\N" else c
                for c in cells
            ]
            buf.write("\t".join(cells))
            buf.write("\n")
            n += 1
            sched_raw = row.get("schedule_code") or ""
            if sched_raw.strip().isdigit():
                schedule_code = int(sched_raw.strip())

    buf.seek(0)
    with conn.cursor() as cur:
        cur.execute(f"TRUNCATE TABLE {target_table} RESTART IDENTITY")
        cur.copy_expert(
            f"COPY {target_table} ({', '.join(db_cols)}) FROM STDIN "
            f"WITH (FORMAT text, NULL '\\N')",
            buf,
        )
    return n, schedule_code


def apply_migrations(conn) -> None:
    for name in ("007_pbs_relational_schema.sql", "008_pbs_relational_indexes.sql"):
        path = KB6 / "migrations" / name
        if not path.exists():
            log.warning("migration %s not found, skipping", name)
            continue
        log.info("applying migration: %s", name)
        with conn.cursor() as cur:
            cur.execute(path.read_text())


def write_load_log(conn, csv_name: str, target_table: str, rows: int,
                   schedule_code: int | None, notes: str = "") -> None:
    with conn.cursor() as cur:
        cur.execute(
            "INSERT INTO kb6_pbs_rel_load_log "
            "(csv_name, target_table, rows_loaded, schedule_code, notes) "
            "VALUES (%s, %s, %s, %s, %s)",
            (csv_name, target_table, rows, schedule_code, notes),
        )


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__,
                                formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("--csv-dir", type=Path, default=DEFAULT_CSV_DIR,
                   help=f"PBS API CSV bundle directory (default: {DEFAULT_CSV_DIR})")
    p.add_argument("--apply-migrations", action="store_true",
                   help="Apply migrations 007/008 before loading")
    p.add_argument("--only", default="",
                   help="Comma-separated list of CSV stems to load (default: all). "
                        "Use stems without .csv: 'restrictions,prescribing-texts'")
    args = p.parse_args()

    if not args.csv_dir.exists():
        log.error("CSV directory does not exist: %s", args.csv_dir)
        return 1

    only = {s.strip() for s in args.only.split(",") if s.strip()}

    conn = psycopg2.connect(**KB6_DSN)
    conn.autocommit = True
    try:
        if args.apply_migrations:
            apply_migrations(conn)

        totals: dict[str, int] = {}
        for csv_name, (target_table, cols) in LOADERS.items():
            stem = csv_name.replace(".csv", "")
            if only and stem not in only:
                continue
            csv_path = args.csv_dir / csv_name
            if not csv_path.exists():
                log.warning("missing %s, skipping", csv_path)
                continue
            log.info("loading %s -> %s", csv_name, target_table)
            try:
                n, schedule_code = load_one_csv(conn, csv_path, target_table, cols)
                write_load_log(conn, csv_name, target_table, n, schedule_code)
                totals[target_table] = n
                log.info("  loaded %d rows into %s (schedule_code=%s)",
                         n, target_table, schedule_code)
            except Exception as e:
                log.error("  failed loading %s: %s", csv_name, e)
                write_load_log(conn, csv_name, target_table, 0, None,
                               notes=f"ERROR: {str(e)[:200]}")
                raise

        log.info("=" * 70)
        log.info("PBS RELATIONAL LOAD COMPLETE")
        log.info("=" * 70)
        for tbl, n in sorted(totals.items()):
            log.info("  %-45s %10d", tbl, n)
        log.info("  %-45s %10d", "TOTAL", sum(totals.values()))
    finally:
        conn.close()
    return 0


if __name__ == "__main__":
    sys.exit(main())
