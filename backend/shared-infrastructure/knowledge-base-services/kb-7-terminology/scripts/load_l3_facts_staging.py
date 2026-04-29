"""
L3 fact loader (staging-table approach).

Reads ADA-extracted L3 fact JSONs from the guideline-atomiser output
and inserts them into a kb_l3_staging table in each consumer KB's
own Postgres DB. Each row carries:

  - source_file, drug_name, domain (contextual/dosing/monitoring/safety)
  - quality_grade (FULL / PARTIAL / STUB / EMPTY)
  - has_placeholders flag + placeholder_count
  - raw JSON for downstream domain-specific mapping

Domain -> KB mapping
  contextual -> KB-20 (kb20-postgres:5436 / kb_service_20)
  dosing     -> KB-1  (kb1-postgres:5481 / kb1_drug_rules)
  monitoring -> KB-16 (kb16-postgres:5446 / kb_lab_interpretation)
  safety     -> KB-4  (kb4-patient-safety-postgres:5440 / kb4_patient_safety)

Why staging instead of mapping straight to final schemas
- L3 data is 26% clean, 65% partial (placeholder markers), 9% empty.
  Direct loading into typed final tables silently loses partial rows
  or violates NOT NULL constraints. Staging preserves everything,
  flagged for downstream curation.

Usage
    python3 scripts/load_l3_facts_staging.py
    python3 scripts/load_l3_facts_staging.py --dry-run     # report only, no DB writes
    python3 scripts/load_l3_facts_staging.py --json        # JSON output
"""

from __future__ import annotations

import argparse
import json
import logging
import re
import sys
from dataclasses import asdict, dataclass, field
from datetime import datetime, timezone
from pathlib import Path

import psycopg2
import psycopg2.extras

logging.basicConfig(level=logging.INFO, format="%(message)s")
log = logging.getLogger(__name__)

REPO_KB7 = Path(__file__).resolve().parent.parent
L3_DIR = (
    REPO_KB7.parent
    / "shared/tools/guideline-atomiser/data/output/v4"
    / "job_monkeyocr_df538e50-0170-4ef8-862d-5b0a7c48e4ff/l3_output"
)

# Domain -> (label, target DSN)
TARGET_DSN = {
    "contextual": dict(host="localhost", port=5436, user="kb20_user",
                       password="kb20_password", dbname="kb_service_20"),
    "dosing":     dict(host="localhost", port=5481, user="kb1_user",
                       password="kb1_password", dbname="kb1_drug_rules"),
    "monitoring": dict(host="localhost", port=5446, user="kb16_user",
                       password="kb_password", dbname="kb_lab_interpretation"),
    "safety":     dict(host="localhost", port=5440, user="kb4_safety_user",
                       password="kb4_safety_password", dbname="kb4_patient_safety"),
}

KB_LABEL = {
    "contextual": "KB-20 patient-profile",
    "dosing":     "KB-1 drug-rules",
    "monitoring": "KB-16 lab-interpretation",
    "safety":     "KB-4 patient-safety",
}

DDL = """
CREATE TABLE IF NOT EXISTS kb_l3_staging (
    id                BIGSERIAL   PRIMARY KEY,
    source_file       TEXT        NOT NULL,
    drug_name         TEXT        NOT NULL,
    domain            TEXT        NOT NULL,
    source_guideline  TEXT,
    extractor_version TEXT,
    extraction_date   DATE,
    quality_grade     TEXT        NOT NULL,   -- FULL | PARTIAL | STUB | EMPTY
    has_placeholders  BOOLEAN     NOT NULL,
    placeholder_count INT         NOT NULL DEFAULT 0,
    fact_count        INT         NOT NULL DEFAULT 0,
    raw_json          JSONB       NOT NULL,
    loaded_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (source_file)
);

CREATE INDEX IF NOT EXISTS idx_kb_l3_staging_drug ON kb_l3_staging (drug_name);
CREATE INDEX IF NOT EXISTS idx_kb_l3_staging_domain_quality
    ON kb_l3_staging (domain, quality_grade);
"""


@dataclass
class L3File:
    path: Path
    drug_name: str
    domain: str           # contextual / dosing / monitoring / safety
    body: dict

    @property
    def fact_count(self) -> int:
        # Each domain stores facts under different keys; count whichever is present.
        for k in ("drugs", "labRequirements", "adrProfiles",
                  "contraindications", "warnings", "standaloneModifiers"):
            v = self.body.get(k)
            if isinstance(v, list):
                return len(v)
        return 0

    @property
    def placeholder_count(self) -> int:
        s = json.dumps(self.body)
        return s.count("<LOOKUP_REQUIRED>") + s.count("<UNKNOWN>")

    @property
    def has_placeholders(self) -> bool:
        return self.placeholder_count > 0

    @property
    def quality_grade(self) -> str:
        if self.fact_count == 0:
            return "EMPTY"

        cs = self.body.get("completenessSummary") or {}
        if cs:
            full = cs.get("FULL", 0)
            stub = cs.get("STUB", 0)
            partial = cs.get("PARTIAL", 0)
            total = full + stub + partial
            if total == 0:
                return "STUB"
            if stub == total:
                return "STUB"
            if full == total and not self.has_placeholders:
                return "FULL"
            return "PARTIAL"

        # No completenessSummary — grade by placeholders.
        return "FULL" if not self.has_placeholders else "PARTIAL"


@dataclass
class LoadResult:
    domain: str
    target_kb: str
    inserted: int = 0
    skipped: int = 0
    errors: int = 0
    by_grade: dict[str, int] = field(default_factory=lambda: {
        "FULL": 0, "PARTIAL": 0, "STUB": 0, "EMPTY": 0
    })


def parse_filename(p: Path) -> tuple[str, str] | None:
    m = re.match(r"^(.+)_([a-z]+)_targeted\.json$", p.name)
    if not m:
        return None
    return m.group(1), m.group(2)


def discover_l3() -> list[L3File]:
    if not L3_DIR.is_dir():
        raise FileNotFoundError(f"L3 dir not found: {L3_DIR}")
    out: list[L3File] = []
    for p in sorted(L3_DIR.glob("*.json")):
        parsed = parse_filename(p)
        if not parsed:
            continue
        drug, domain = parsed
        if domain not in TARGET_DSN:
            continue
        try:
            body = json.loads(p.read_text())
        except json.JSONDecodeError:
            log.warning("  skipping unparseable: %s", p.name)
            continue
        out.append(L3File(path=p, drug_name=drug, domain=domain, body=body))
    return out


def ensure_schema(dsn: dict) -> None:
    with psycopg2.connect(**dsn) as conn, conn.cursor() as cur:
        cur.execute(DDL)
    # connect() context-manager doesn't auto-commit DDL across PG versions; explicit:
    # (psycopg2 connect-cm only rolls back on exception; commit on normal exit)


def load_domain(domain: str, files: list[L3File], dry_run: bool) -> LoadResult:
    dsn = TARGET_DSN[domain]
    res = LoadResult(domain=domain, target_kb=KB_LABEL[domain])

    if not dry_run:
        ensure_schema(dsn)

    if not dry_run:
        conn = psycopg2.connect(**dsn)
    else:
        conn = None

    try:
        for f in files:
            res.by_grade[f.quality_grade] += 1
            if dry_run:
                res.skipped += 1
                continue
            try:
                with conn.cursor() as cur:
                    cur.execute(
                        """
                        INSERT INTO kb_l3_staging
                            (source_file, drug_name, domain, source_guideline,
                             extractor_version, extraction_date, quality_grade,
                             has_placeholders, placeholder_count, fact_count, raw_json)
                        VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                        ON CONFLICT (source_file) DO UPDATE SET
                            quality_grade     = EXCLUDED.quality_grade,
                            has_placeholders  = EXCLUDED.has_placeholders,
                            placeholder_count = EXCLUDED.placeholder_count,
                            fact_count        = EXCLUDED.fact_count,
                            raw_json          = EXCLUDED.raw_json,
                            loaded_at         = now()
                        """,
                        (
                            f.path.name,
                            f.drug_name,
                            f.domain,
                            f.body.get("sourceGuideline"),
                            f.body.get("extractorVersion"),
                            f.body.get("extractionDate"),
                            f.quality_grade,
                            f.has_placeholders,
                            f.placeholder_count,
                            f.fact_count,
                            psycopg2.extras.Json(f.body),
                        ),
                    )
                res.inserted += 1
            except Exception as e:
                log.warning("  INSERT failed for %s: %s", f.path.name, e)
                res.errors += 1
    finally:
        if conn:
            conn.commit()
            conn.close()
    return res


def render_text(results: list[LoadResult]) -> str:
    out: list[str] = []
    out.append("=" * 78)
    out.append(f"L3 Fact Staging Load Report")
    out.append(f"Generated: {datetime.now(timezone.utc).isoformat()}")
    out.append(f"Source: {L3_DIR}")
    out.append("=" * 78)
    out.append(f"  {'DOMAIN':<12} {'TARGET KB':<25} {'IN':>5} {'ERR':>4} "
               f"{'FULL':>5} {'PART':>5} {'STUB':>5} {'EMPT':>5}")
    totals = {"in": 0, "err": 0, "FULL": 0, "PARTIAL": 0, "STUB": 0, "EMPTY": 0}
    for r in results:
        totals["in"] += r.inserted
        totals["err"] += r.errors
        for g in ("FULL", "PARTIAL", "STUB", "EMPTY"):
            totals[g] += r.by_grade[g]
        out.append(
            f"  {r.domain:<12} {r.target_kb:<25} {r.inserted:>5} {r.errors:>4} "
            f"{r.by_grade['FULL']:>5} {r.by_grade['PARTIAL']:>5} "
            f"{r.by_grade['STUB']:>5} {r.by_grade['EMPTY']:>5}"
        )
    out.append("-" * 78)
    out.append(f"  {'TOTAL':<12} {'':<25} {totals['in']:>5} {totals['err']:>4} "
               f"{totals['FULL']:>5} {totals['PARTIAL']:>5} "
               f"{totals['STUB']:>5} {totals['EMPTY']:>5}")
    out.append("=" * 78)
    out.append("")
    out.append("FULL    = no placeholders, all completeness=FULL  (load directly)")
    out.append("PARTIAL = some placeholders or mixed completeness (curate before use)")
    out.append("STUB    = mostly empty (review)")
    out.append("EMPTY   = no facts extracted (legitimate empty result)")
    return "\n".join(out)


def main() -> int:
    p = argparse.ArgumentParser(
        description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter
    )
    p.add_argument("--dry-run", action="store_true",
                   help="Discover + grade only; no DB writes")
    p.add_argument("--json", action="store_true",
                   help="Emit JSON instead of text report")
    p.add_argument("--domain", choices=list(TARGET_DSN.keys()),
                   help="Restrict to one domain (default: all)")
    args = p.parse_args()

    log.info("Discovering L3 files in %s ...", L3_DIR)
    files = discover_l3()
    log.info("  %d files", len(files))

    if args.domain:
        files = [f for f in files if f.domain == args.domain]

    by_domain: dict[str, list[L3File]] = {}
    for f in files:
        by_domain.setdefault(f.domain, []).append(f)

    results: list[LoadResult] = []
    for domain in sorted(by_domain):
        log.info("--- domain: %s (%d files) -> %s ---",
                 domain, len(by_domain[domain]), KB_LABEL[domain])
        try:
            r = load_domain(domain, by_domain[domain], args.dry_run)
        except psycopg2.Error as e:
            log.error("  DB error: %s", e)
            r = LoadResult(domain=domain, target_kb=KB_LABEL[domain],
                           errors=len(by_domain[domain]))
        results.append(r)
        log.info("  inserted=%d errors=%d", r.inserted, r.errors)

    if args.json:
        print(json.dumps({
            "generated_at": datetime.now(timezone.utc).isoformat(),
            "source_dir": str(L3_DIR),
            "results": [asdict(r) for r in results],
        }, indent=2, default=str))
    else:
        print()
        print(render_text(results))

    return 0 if all(r.errors == 0 for r in results) else 1


if __name__ == "__main__":
    sys.exit(main())
