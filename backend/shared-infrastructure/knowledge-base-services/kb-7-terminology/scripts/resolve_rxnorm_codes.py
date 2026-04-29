"""
RxNorm code resolver for L3 staging rows.

Many L3 fact JSONs have rxnormCode = "<LOOKUP_REQUIRED>" because the
guideline-atomiser LLM couldn't resolve the drug name to an RxCUI from
guideline prose alone. This script bridges that gap:

  1. ALTER each kb_l3_staging table (in 4 KB DBs) to add resolved_rxcui
     and resolved_rxcui_source columns (idempotent).
  2. Read distinct drug_name values from each staging table.
  3. For each drug_name, resolve via RxNav-in-a-Box (localhost:4000):
       - try exact match (/REST/rxcui.json?name=<name>)
       - on miss, try approximate (/REST/approximateTerm.json)
       - cache results so we never call RxNav twice per drug
  4. UPDATE kb_l3_staging rows with the resolved value.

The original raw_json is preserved untouched. resolved_rxcui /
resolved_rxcui_source are new columns the typed extractors read instead.

resolved_rxcui_source values:
  EXACT       — RxNav matched a drug name directly
  APPROXIMATE — RxNav fuzzy match (typically class names like "ACE inhibitor")
  NONE        — no match found, even with fuzzy

Usage
    python3 scripts/resolve_rxnorm_codes.py
    python3 scripts/resolve_rxnorm_codes.py --dry-run
"""

from __future__ import annotations

import argparse
import json
import logging
import sys
import time
from dataclasses import dataclass

import psycopg2
import requests

logging.basicConfig(level=logging.INFO, format="%(message)s")
log = logging.getLogger(__name__)

RXNAV_BASE = "http://localhost:4000/REST"

TARGETS = {
    "contextual": dict(host="localhost", port=5436, user="kb20_user",
                       password="kb20_password", dbname="kb_service_20"),
    "dosing":     dict(host="localhost", port=5481, user="kb1_user",
                       password="kb1_password", dbname="kb1_drug_rules"),
    "monitoring": dict(host="localhost", port=5446, user="kb16_user",
                       password="kb_password", dbname="kb_lab_interpretation"),
    "safety":     dict(host="localhost", port=5440, user="kb4_safety_user",
                       password="kb4_safety_password", dbname="kb4_patient_safety"),
}

DDL = """
ALTER TABLE kb_l3_staging
    ADD COLUMN IF NOT EXISTS resolved_rxcui VARCHAR(50),
    ADD COLUMN IF NOT EXISTS resolved_rxcui_source VARCHAR(20),
    ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMPTZ;
"""


@dataclass
class Resolution:
    drug_name: str
    rxcui: str | None
    source: str  # EXACT | APPROXIMATE | NONE


def normalize(name: str) -> str:
    """Lowercase, trim, drop trailing 's' for plurals like 'ACE inhibitors' -> 'ACE inhibitor'."""
    n = (name or "").strip()
    return n


def rxnav_exact(name: str) -> str | None:
    try:
        r = requests.get(f"{RXNAV_BASE}/rxcui.json", params={"name": name}, timeout=10)
        if r.status_code != 200:
            return None
        ids = (r.json().get("idGroup") or {}).get("rxnormId") or []
        return ids[0] if ids else None
    except requests.RequestException:
        return None


def rxnav_approximate(name: str) -> str | None:
    try:
        r = requests.get(
            f"{RXNAV_BASE}/approximateTerm.json",
            params={"term": name, "maxEntries": 1},
            timeout=10,
        )
        if r.status_code != 200:
            return None
        cands = (r.json().get("approximateGroup") or {}).get("candidate") or []
        return cands[0]["rxcui"] if cands else None
    except (requests.RequestException, KeyError):
        return None


def resolve(name: str, cache: dict[str, Resolution]) -> Resolution:
    n = normalize(name)
    if not n:
        return Resolution(name, None, "NONE")
    if n in cache:
        return cache[n]

    rxcui = rxnav_exact(n)
    if rxcui:
        res = Resolution(n, rxcui, "EXACT")
    else:
        # Try without trailing 's' for plurals
        n_singular = n.rstrip("s") if n.endswith("s") and len(n) > 4 else None
        rxcui_s = rxnav_exact(n_singular) if n_singular else None
        if rxcui_s:
            res = Resolution(n, rxcui_s, "EXACT")
        else:
            # Try approximate / fuzzy
            rxcui_a = rxnav_approximate(n)
            if rxcui_a:
                res = Resolution(n, rxcui_a, "APPROXIMATE")
            else:
                res = Resolution(n, None, "NONE")

    cache[n] = res
    # Be polite to RxNav
    time.sleep(0.05)
    return res


def alter_schema(dsn: dict) -> None:
    with psycopg2.connect(**dsn) as conn, conn.cursor() as cur:
        cur.execute(DDL)


def distinct_drugs(dsn: dict) -> list[str]:
    with psycopg2.connect(**dsn) as conn, conn.cursor() as cur:
        cur.execute(
            """
            SELECT DISTINCT
                COALESCE(raw_json ->> 'drugName',
                         raw_json -> 'drugs' -> 0 ->> 'drugName',
                         raw_json -> 'contraindications' -> 0 ->> 'drugName',
                         raw_json -> 'labRequirements' -> 0 ->> 'drugName',
                         raw_json -> 'adrProfiles' -> 0 ->> 'drugName',
                         drug_name)
            FROM kb_l3_staging
            WHERE fact_count > 0
            """
        )
        return [r[0] for r in cur.fetchall() if r[0]]


def update_resolutions(dsn: dict, resolutions_by_drug: dict[str, Resolution]) -> int:
    """For each row whose JSON drug name maps to a resolution, set
    resolved_rxcui + resolved_rxcui_source. Returns number updated."""
    updated = 0
    with psycopg2.connect(**dsn) as conn, conn.cursor() as cur:
        cur.execute(
            """
            SELECT id, drug_name,
                   COALESCE(raw_json ->> 'drugName',
                            raw_json -> 'drugs' -> 0 ->> 'drugName',
                            raw_json -> 'contraindications' -> 0 ->> 'drugName',
                            raw_json -> 'labRequirements' -> 0 ->> 'drugName',
                            raw_json -> 'adrProfiles' -> 0 ->> 'drugName')
            FROM kb_l3_staging
            WHERE fact_count > 0
            """
        )
        rows = cur.fetchall()
        for row_id, drug_slug, json_drug_name in rows:
            key = normalize(json_drug_name) if json_drug_name else ""
            if not key or key not in resolutions_by_drug:
                continue
            res = resolutions_by_drug[key]
            cur.execute(
                """
                UPDATE kb_l3_staging
                SET resolved_rxcui = %s,
                    resolved_rxcui_source = %s,
                    resolved_at = now()
                WHERE id = %s
                """,
                (res.rxcui, res.source, row_id),
            )
            updated += 1
    return updated


def main() -> int:
    p = argparse.ArgumentParser(
        description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter
    )
    p.add_argument("--dry-run", action="store_true",
                   help="Resolve only; do not modify DB")
    args = p.parse_args()

    # Connectivity probe
    try:
        r = requests.get(f"{RXNAV_BASE}/version.json", timeout=5)
        log.info("RxNav-in-a-Box reachable: %s", r.json())
    except Exception as e:
        log.error("RxNav-in-a-Box not reachable on %s: %s", RXNAV_BASE, e)
        return 2

    cache: dict[str, Resolution] = {}
    summary: dict[str, dict] = {}

    for domain, dsn in TARGETS.items():
        log.info("--- %s -> %s ---", domain, dsn["dbname"])
        if not args.dry_run:
            alter_schema(dsn)

        drugs = distinct_drugs(dsn)
        log.info("  distinct drug names: %d", len(drugs))

        # Resolve each, accumulating into shared cache
        for d in drugs:
            res = resolve(d, cache)
            log.info("    %s -> rxcui=%s (%s)", d[:50], res.rxcui, res.source)

        if args.dry_run:
            continue

        # Build per-drug map for this DSN's UPDATE
        per_drug = {normalize(d): cache[normalize(d)] for d in drugs if normalize(d) in cache}
        n_updated = update_resolutions(dsn, per_drug)

        # Summary
        exact = sum(1 for d in drugs if cache[normalize(d)].source == "EXACT")
        approx = sum(1 for d in drugs if cache[normalize(d)].source == "APPROXIMATE")
        none_ = sum(1 for d in drugs if cache[normalize(d)].source == "NONE")
        summary[domain] = dict(
            distinct_drugs=len(drugs),
            exact=exact,
            approximate=approx,
            none=none_,
            rows_updated=n_updated,
        )
        log.info(
            "  resolutions: %d EXACT, %d APPROXIMATE, %d NONE  | rows updated: %d",
            exact, approx, none_, n_updated
        )

    if not args.dry_run:
        log.info("=" * 70)
        log.info("RESOLUTION SUMMARY")
        log.info("=" * 70)
        log.info(
            "  %-12s %8s %7s %12s %5s %14s",
            "domain", "distinct", "EXACT", "APPROXIMATE", "NONE", "rows_updated"
        )
        for d, s in summary.items():
            log.info(
                "  %-12s %8d %7d %12d %5d %14d",
                d, s["distinct_drugs"], s["exact"], s["approximate"],
                s["none"], s["rows_updated"]
            )
    return 0


if __name__ == "__main__":
    sys.exit(main())
