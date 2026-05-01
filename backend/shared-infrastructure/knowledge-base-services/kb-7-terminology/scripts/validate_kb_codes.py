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
from urllib.error import URLError
from urllib.request import urlopen

import psycopg2

# RxNav-in-a-Box base URL — used by --rxnav to classify unresolved RxCUIs
# as Active / NotCurrent (retired) / Obsolete (remapped) / TruePhantom.
# The container ships a complete NLM RxNorm release with full history, so
# it's the authoritative oracle for "is this RxCUI legitimate?"
RXNAV_BASE = "http://localhost:4000"

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
KB4_DSN = dict(
    host="localhost", port=5440, user="kb4_safety_user",
    password="kb4_safety_password", dbname="kb4_patient_safety",
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
    is_array: bool = False    # True when column is a TEXT[] / VARCHAR[] needing UNNEST


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
    # Optional RxNav classification (only populated for RxNorm checks when
    # --rxnav is passed). Counts how unresolved codes break down per status.
    rxnav_classification: dict = field(default_factory=dict)
    rxnav_remap_samples: list[dict] = field(default_factory=list)


def _rxnav_status(rxcui: str, timeout: float = 2.0) -> dict:
    """Query RxNav-in-a-Box historystatus for one RxCUI.

    Returns a dict with keys:
        status: 'Active' | 'NotCurrent' | 'Obsolete' | 'Remapped'
                | 'Quantified' | 'Alien' | 'TruePhantom' | 'RxNavError'
        name:   current/historical name if known
        tty:    term type if known
        remap_target: list of remap target RxCUI strings (for Obsolete/Remapped)

    A status of 'TruePhantom' means RxNav also doesn't recognize the code —
    it's neither current nor historical, suggesting a typo or invented code
    in the source YAML.
    """
    try:
        with urlopen(
            f"{RXNAV_BASE}/REST/rxcui/{rxcui}/historystatus.json",
            timeout=timeout,
        ) as resp:
            payload = json.load(resp)
    except (URLError, json.JSONDecodeError, OSError):
        return {"status": "RxNavError", "name": "", "tty": "",
                "remap_target": []}

    history = payload.get("rxcuiStatusHistory") or {}
    meta = history.get("metaData") or {}
    attrs = history.get("attributes") or {}
    status = meta.get("status") or "TruePhantom"

    # Find any remap targets (RxNav lists these under derivedConcepts /
    # remappedConcept depending on transition type).
    remap_target: list[str] = []
    derived = history.get("derivedConcepts") or {}
    for bucket in ("remappedConcept", "quantifiedConcept"):
        for entry in derived.get(bucket, []) or []:
            target = entry.get("remappedRxCui") or entry.get("quantifiedRxCui")
            if target:
                remap_target.append(str(target))

    return {
        "status": status,
        "name": attrs.get("name", ""),
        "tty": attrs.get("tty", ""),
        "remap_target": remap_target,
    }


def _classify_unresolved_rxcuis(
    unresolved: set[str],
    sample_n: int,
    log_progress: bool = False,
) -> tuple[dict, list[dict]]:
    """For a set of unresolved RxCUIs, classify each via RxNav and return:
        - status_counts: {Active: N, NotCurrent: N, Obsolete: N, ...}
        - sample_remaps: up to sample_n {rxcui, status, name, remap_target}
          biased toward Obsolete/Remapped (the actionable cases)
    """
    counts: dict[str, int] = {}
    actionable: list[dict] = []
    other: list[dict] = []

    for i, rxcui in enumerate(sorted(unresolved)):
        info = _rxnav_status(rxcui)
        s = info["status"]
        counts[s] = counts.get(s, 0) + 1

        record = {"rxcui": rxcui, **info}
        # Bias the sample toward fixable cases (have a remap target or are
        # marked NotCurrent — both repairable by RxNav lookup).
        if info["remap_target"] or s in ("NotCurrent", "Obsolete", "Remapped"):
            if len(actionable) < sample_n:
                actionable.append(record)
        else:
            if len(other) < sample_n:
                other.append(record)

        if log_progress and (i + 1) % 25 == 0:
            log.info("    RxNav classified %d/%d unresolved codes",
                     i + 1, len(unresolved))

    samples = actionable + other[: max(0, sample_n - len(actionable))]
    return counts, samples[:sample_n]


def _resolution_status(pct: float) -> str:
    if pct >= 99.5:
        return "PASS"
    if pct >= 95.0:
        return "WARN"
    return "FAIL"


def _normalize_icd10(code: str) -> str:
    """Strip dots and uppercase an ICD-10 code.

    'E10.10' -> 'E1010', 'E10' -> 'E10', ' D50.0 ' -> 'D500'.
    Used to bridge WHO ICD-10 (dotted) vs ICD-10-CM (dotless billable) — see
    claudedocs/audits/2026-04-29_kb_cross_reference_gap_report.md §"Format mismatch".
    """
    return (code or "").replace(".", "").strip().upper()


def _resolve_icd10(consumer_codes: set[str], ref_codes: set[str]) -> tuple[set[str], set[str]]:
    """Match WHO ICD-10 consumer codes against ICD-10-CM reference codes.

    KB-7 holds ICD-10-CM (dotless, billable 5-char like 'E1010').
    KB-4 START_V3 holds WHO ICD-10 (dotted, often 3-char rollup like 'E10').

    A consumer code is resolved if:
      (a) its dotless form is an exact match in the reference set, OR
      (b) it's a 3- or 4-character rollup AND any reference code starts
          with that dotless prefix (e.g. 'E10' resolves because 'E1010',
          'E1011', etc. exist in the reference set).

    Returns (resolved_set, unresolved_set) using ORIGINAL consumer-code form
    so the report retains the source-faithful spelling.
    """
    import bisect
    ref_norm = {_normalize_icd10(c) for c in ref_codes if c}
    ref_sorted = sorted(ref_norm)
    resolved: set[str] = set()
    for code in consumer_codes:
        norm = _normalize_icd10(code)
        if not norm:
            continue
        if norm in ref_norm:
            resolved.add(code)
            continue
        # Rollup match: only for short codes (3-4 dotless chars)
        if 3 <= len(norm) <= 4:
            i = bisect.bisect_left(ref_sorted, norm)
            if i < len(ref_sorted) and ref_sorted[i].startswith(norm):
                resolved.add(code)
    return resolved, consumer_codes - resolved


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


def validate_consumer(check: ConsumerCheck, sample_n: int,
                      rxnav: bool = False) -> CheckResult:
    """Run one validation check.

    Strategy:
      1. Pull all distinct non-NULL codes from consumer KB into Python.
      2. Pull the full set of valid codes from KB-7 reference query.
      3. Compute set difference (consumer - reference).
    """
    # 1. Pull consumer codes
    consumer_dsn = {
        "kb1_drug_rules":     KB1_DSN,
        "kb4_patient_safety": KB4_DSN,
    }.get(check.consumer_kb, KB7_DSN)

    with psycopg2.connect(**consumer_dsn) as cc, cc.cursor() as cur:
        if check.is_array:
            # Array column — count rows-with-any-non-empty-array, then unnest distinct
            cur.execute(
                f"SELECT count(*) FROM {check.table} "
                f"WHERE {check.column} IS NOT NULL "
                f"AND array_length({check.column}, 1) > 0"
            )
            total_rows = cur.fetchone()[0]
            cur.execute(
                f"SELECT DISTINCT unnest({check.column})::text "
                f"FROM {check.table} "
                f"WHERE {check.column} IS NOT NULL "
                f"AND array_length({check.column}, 1) > 0"
            )
            consumer_codes = {r[0] for r in cur.fetchall() if r[0]}
        else:
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
    # ICD-10 needs normalized matching — KB-7 holds ICD-10-CM (dotless,
    # 5-char billable) while consumer KBs typically hold WHO ICD-10 (dotted,
    # often 3-char rollups). Plain set difference reports ~2.3% resolution
    # which is a format mismatch, not a data-quality failure.
    if "ICD-10" in check.target_system:
        resolved_set, unresolved_set = _resolve_icd10(consumer_codes, ref_codes)
    else:
        resolved_set = consumer_codes & ref_codes
        unresolved_set = consumer_codes - ref_codes
    resolved = len(resolved_set)
    unresolved = len(unresolved_set)
    pct = (resolved / distinct * 100.0) if distinct else 100.0

    # 4. Optional: classify unresolved RxCUIs via RxNav-in-a-Box.
    # Only meaningful for RxNorm checks; skip for SNOMED/ICD-10/etc.
    rxnav_counts: dict = {}
    rxnav_samples: list[dict] = []
    if rxnav and unresolved_set and "RxNorm" in check.target_system:
        log.info("    [%s] classifying %d unresolved RxCUIs via RxNav...",
                 check.column, len(unresolved_set))
        rxnav_counts, rxnav_samples = _classify_unresolved_rxcuis(
            unresolved_set, sample_n=sample_n, log_progress=True,
        )

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
        rxnav_classification=rxnav_counts,
        rxnav_remap_samples=rxnav_samples,
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
        # KB-4 patient-safety — explicit-criteria (STOPP/START/Beers/APINCHs/
        # TGA blackbox/TGA pregnancy/ACB/Wang 2024 = 392 rules as of 2026-04-29)
        ConsumerCheck(
            consumer_kb="kb4_patient_safety",
            table="kb4_explicit_criteria",
            column="rxnorm_code_primary",
            target_system="RxNorm (single-drug rules — Beers/APINCHs/TGA/ACB)",
            reference_query="SELECT code FROM concepts_rxnorm",
        ),
        ConsumerCheck(
            consumer_kb="kb4_patient_safety",
            table="kb4_explicit_criteria",
            column="rxnorm_codes",
            target_system="RxNorm (multi-drug rules — STOPP/START/Wang)",
            reference_query="SELECT code FROM concepts_rxnorm",
            is_array=True,
        ),
        ConsumerCheck(
            consumer_kb="kb4_patient_safety",
            table="kb4_explicit_criteria",
            column="condition_icd10",
            target_system="ICD-10 (START condition codes)",
            # Union of ICD-10-CM (dotless billable) + WHO ICD-10 (dotted).
            # _resolve_icd10 normalizes both sides for comparison; WHO codes
            # bridge the gap for residual taxonomic divergences (F00 dementia
            # series, H40.10-13 glaucoma subcodes, J46 status asthmaticus,
            # M82 osteoporosis in diseases — see migration 019).
            reference_query=("SELECT code FROM concepts_icd10 "
                             "UNION ALL SELECT code FROM concepts_icd10_who"),
            is_array=True,
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

    rxnav_issues = [r for r in results if r.rxnav_classification]
    if rxnav_issues:
        out.append("")
        out.append("RXNAV CLASSIFICATION (unresolved RxCUIs)")
        out.append("-" * 78)
        for r in rxnav_issues:
            out.append(f"  {r.consumer_kb}.{r.table}.{r.column}")
            total = sum(r.rxnav_classification.values())
            for status, count in sorted(r.rxnav_classification.items(),
                                        key=lambda x: -x[1]):
                pct = (count / total * 100.0) if total else 0.0
                hint = {
                    "Active":      "→ stale KB-7; reload RxNorm",
                    "NotCurrent":  "→ retired in current RxNorm; remap or remove",
                    "Obsolete":    "→ replaced by newer RxCUI; remap via RxNav",
                    "Remapped":    "→ already remapped; use new RxCUI",
                    "Quantified":  "→ dose-quantified concept; check parent",
                    "Alien":       "→ from another vocabulary; map to RxNorm",
                    "UNKNOWN":     "→ RxNav has metadata but cannot classify; investigate YAML",
                    "TruePhantom": "→ NOT in RxNav either; YAML typo",
                    "RxNavError":  "→ RxNav unreachable",
                }.get(status, "")
                out.append(f"    {status:<12} {count:>4} ({pct:>5.1f}%)  {hint}")

            if r.rxnav_remap_samples:
                out.append("    Sample remap targets:")
                for s in r.rxnav_remap_samples[:5]:
                    target = ",".join(s.get("remap_target") or []) or "—"
                    out.append(f"      {s['rxcui']:<10} {s['status']:<12} "
                               f"→ {target:<10}  {s.get('name', '')[:40]}")
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
    p.add_argument("--rxnav", action="store_true",
                   help=("For RxNorm checks, classify unresolved codes against "
                         "RxNav-in-a-Box (localhost:4000) to distinguish "
                         "Active/NotCurrent/Obsolete/TruePhantom"))
    args = p.parse_args()

    refs = reference_summary()

    results = []
    for check in build_checks():
        try:
            res = validate_consumer(check, args.sample, rxnav=args.rxnav)
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
