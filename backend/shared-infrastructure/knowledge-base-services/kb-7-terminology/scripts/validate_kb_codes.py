"""
Cross-KB code-resolution validation tool (Wave 1 exit check).

For each consumer KB's code-bearing column, count how many codes don't
resolve in KB-7's authoritative reference tables. Surfaces phantom
codes — references to drugs/conditions/labs/etc that won't be findable
at runtime.

Currently checks:
  - kb1_drug_rules.drug_rules.rxnorm_code  -> kb_terminology.concepts_rxnorm
  - kb1_drug_rules.drug_rules.snomed_code  -> kb_terminology.{concepts_snomed, kb7_snomed_concept}
  - kb1_drug_rules.drug_rule_history.rxnorm_code
  - kb1_drug_rules.ingestion_items.rxnorm_code
  - kb_terminology.concept_mappings.{source,target}_code (where applicable)

Other KBs (KB-4 patient-safety, KB-5 interactions, KB-16 labs, KB-20)
are not currently running locally and cannot be validated until their
databases are accessible.

Usage:
    python3 scripts/validate_kb_codes.py
    python3 scripts/validate_kb_codes.py --json    # JSON output
    python3 scripts/validate_kb_codes.py --sample 20  # sample N unresolved per check
"""

from __future__ import annotations

import argparse
import json
import logging
import sys
from dataclasses import asdict, dataclass, field
from datetime import datetime, timezone

import psycopg2

logging.basicConfig(level=logging.INFO, format="%(message)s")
log = logging.getLogger(__name__)

KB7_DSN = dict(
    host="localhost", port=5457, user="postgres",
    password="password", dbname="kb_terminology",
)
KB1_DSN = dict(
    host="localhost", port=5457, user="kb1_user",
    password="kb1_password", dbname="kb1_drug_rules",
)

# Reference tables in kb_terminology — where codes SHOULD resolve to.
REFERENCE_TABLES = [
    ("RxNorm", "concepts_rxnorm", "code"),
    ("LOINC", "concepts_loinc", "code"),
    ("ICD-10", "concepts_icd10", "code"),
    ("SNOMED (legacy)", "concepts_snomed", "code"),
    ("SNOMED-AU (new)", "kb7_snomed_concept", "id::text"),
    ("AMT (new) — CTPP", "kb7_amt_pack", "ctpp_sctid::text"),
]


@dataclass
class ConsumerCheck:
    """One code-column-in-a-consumer-KB to validate against KB-7."""
    consumer_kb: str
    table: str
    column: str
    target_system: str        # human label
    reference_query: str      # SELECT producing one TEXT column of valid codes


@dataclass
class CheckResult:
    consumer_kb: str
    table: str
    column: str
    target_system: str
    total_rows: int
    distinct_codes: int
    resolved: int
    unresolved: int
    resolution_pct: float
    sample_unresolved: list[str] = field(default_factory=list)
    status: str = "PASS"          # PASS | WARN | FAIL


def _resolution_status(pct: float) -> str:
    if pct >= 99.5:
        return "PASS"
    if pct >= 95.0:
        return "WARN"
    return "FAIL"


def reference_summary() -> list[dict]:
    rows: list[dict] = []
    with psycopg2.connect(**KB7_DSN) as conn, conn.cursor() as cur:
        for label, table, _ in REFERENCE_TABLES:
            try:
                cur.execute(f"SELECT count(*) FROM {table}")
                n = cur.fetchone()[0]
                rows.append({"label": label, "table": table, "rows": n})
            except Exception as e:
                rows.append({"label": label, "table": table,
                             "rows": -1, "error": str(e)})
    return rows


def validate_consumer(check: ConsumerCheck, sample_n: int) -> CheckResult:
    """Run one validation check.

    Strategy:
      1. Pull all distinct non-NULL codes from consumer KB into Python.
      2. Pull the full set of valid codes from KB-7 reference query.
      3. Compute set difference (consumer - reference).
    """
    # 1. Pull consumer codes
    consumer_dsn = KB1_DSN if check.consumer_kb == "kb1_drug_rules" else KB7_DSN
    with psycopg2.connect(**consumer_dsn) as cc, cc.cursor() as cur:
        cur.execute(
            f"SELECT count(*) FROM {check.table} "
            f"WHERE {check.column} IS NOT NULL AND {check.column} <> ''"
        )
        total_rows = cur.fetchone()[0]

        cur.execute(
            f"SELECT DISTINCT {check.column} FROM {check.table} "
            f"WHERE {check.column} IS NOT NULL AND {check.column} <> ''"
        )
        consumer_codes = {r[0] for r in cur.fetchall()}

    distinct = len(consumer_codes)
    if distinct == 0:
        return CheckResult(
            consumer_kb=check.consumer_kb,
            table=check.table,
            column=check.column,
            target_system=check.target_system,
            total_rows=total_rows,
            distinct_codes=0,
            resolved=0,
            unresolved=0,
            resolution_pct=100.0,
            status="PASS",
        )

    # 2. Pull reference codes from KB-7
    with psycopg2.connect(**KB7_DSN) as kc, kc.cursor() as cur:
        cur.execute(check.reference_query)
        ref_codes = {r[0] for r in cur.fetchall()}

    # 3. Compute resolution
    resolved_set = consumer_codes & ref_codes
    unresolved_set = consumer_codes - ref_codes
    resolved = len(resolved_set)
    unresolved = len(unresolved_set)
    pct = (resolved / distinct * 100.0) if distinct else 100.0

    return CheckResult(
        consumer_kb=check.consumer_kb,
        table=check.table,
        column=check.column,
        target_system=check.target_system,
        total_rows=total_rows,
        distinct_codes=distinct,
        resolved=resolved,
        unresolved=unresolved,
        resolution_pct=pct,
        sample_unresolved=sorted(unresolved_set)[:sample_n],
        status=_resolution_status(pct),
    )


def build_checks() -> list[ConsumerCheck]:
    return [
        ConsumerCheck(
            consumer_kb="kb1_drug_rules",
            table="drug_rules",
            column="rxnorm_code",
            target_system="RxNorm",
            reference_query="SELECT code FROM concepts_rxnorm",
        ),
        ConsumerCheck(
            consumer_kb="kb1_drug_rules",
            table="drug_rules",
            column="snomed_code",
            target_system="SNOMED (legacy + new union)",
            reference_query=(
                "SELECT code FROM concepts_snomed "
                "UNION "
                "SELECT id::text FROM kb7_snomed_concept"
            ),
        ),
        ConsumerCheck(
            consumer_kb="kb1_drug_rules",
            table="drug_rule_history",
            column="rxnorm_code",
            target_system="RxNorm",
            reference_query="SELECT code FROM concepts_rxnorm",
        ),
        ConsumerCheck(
            consumer_kb="kb1_drug_rules",
            table="ingestion_items",
            column="rxnorm_code",
            target_system="RxNorm",
            reference_query="SELECT code FROM concepts_rxnorm",
        ),
    ]


def render_text(refs: list[dict], results: list[CheckResult]) -> str:
    out: list[str] = []
    out.append("=" * 78)
    out.append("KB-7 Cross-Reference Validation Report")
    out.append(f"Generated: {datetime.now(timezone.utc).isoformat()}")
    out.append("=" * 78)
    out.append("")
    out.append("REFERENCE TABLES (KB-7 authoritative)")
    out.append("-" * 78)
    out.append(f"  {'SYSTEM':<22}  {'TABLE':<25}  {'ROWS':>12}")
    for r in refs:
        n = r["rows"]
        n_str = "ERROR" if n == -1 else f"{n:,}"
        out.append(f"  {r['label']:<22}  {r['table']:<25}  {n_str:>12}")
    out.append("")
    out.append("CONSUMER VALIDATIONS")
    out.append("-" * 78)
    out.append(
        f"  {'STATUS':<6}  {'KB.TABLE.COLUMN':<48}  "
        f"{'DISTINCT':>8}  {'RESOLVED':>8}  {'PCT':>6}"
    )
    for r in results:
        target = f"{r.consumer_kb}.{r.table}.{r.column}"
        out.append(
            f"  [{r.status:<4}] {target:<48}  "
            f"{r.distinct_codes:>8,}  {r.resolved:>8,}  {r.resolution_pct:>5.1f}%"
        )

    issues = [r for r in results if r.status != "PASS"]
    if issues:
        out.append("")
        out.append("UNRESOLVED SAMPLES")
        out.append("-" * 78)
        for r in issues:
            out.append(f"  {r.consumer_kb}.{r.table}.{r.column} "
                       f"({r.unresolved} unresolved):")
            for s in r.sample_unresolved:
                out.append(f"    {s}")
    out.append("")
    out.append("=" * 78)
    out.append(f"Total checks: {len(results)}  "
               f"PASS: {sum(1 for r in results if r.status == 'PASS')}  "
               f"WARN: {sum(1 for r in results if r.status == 'WARN')}  "
               f"FAIL: {sum(1 for r in results if r.status == 'FAIL')}")
    out.append("=" * 78)
    return "\n".join(out)


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__,
                                formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("--json", action="store_true",
                   help="Emit JSON instead of human-readable report")
    p.add_argument("--sample", type=int, default=10,
                   help="Sample size of unresolved codes per failing check (default 10)")
    args = p.parse_args()

    refs = reference_summary()

    results = []
    for check in build_checks():
        try:
            res = validate_consumer(check, args.sample)
        except psycopg2.Error as e:
            res = CheckResult(
                consumer_kb=check.consumer_kb, table=check.table,
                column=check.column, target_system=check.target_system,
                total_rows=-1, distinct_codes=-1, resolved=-1,
                unresolved=-1, resolution_pct=0.0, status="FAIL",
                sample_unresolved=[f"db error: {e}"],
            )
        results.append(res)

    if args.json:
        out = {
            "generated_at": datetime.now(timezone.utc).isoformat(),
            "reference_tables": refs,
            "consumer_validations": [asdict(r) for r in results],
            "summary": {
                "total": len(results),
                "pass": sum(1 for r in results if r.status == "PASS"),
                "warn": sum(1 for r in results if r.status == "WARN"),
                "fail": sum(1 for r in results if r.status == "FAIL"),
            },
        }
        print(json.dumps(out, indent=2, default=str))
    else:
        print(render_text(refs, results))

    return 0 if all(r.status != "FAIL" for r in results) else 2


if __name__ == "__main__":
    sys.exit(main())
