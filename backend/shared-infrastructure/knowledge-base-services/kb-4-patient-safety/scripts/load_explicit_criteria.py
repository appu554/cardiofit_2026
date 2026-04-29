"""
Explicit-criteria loader for KB-4 (Wave 3).

Reads the structured YAML files at:
  knowledge/global/stopp_start/stopp_v3.yaml   (80 STOPP entries)
  knowledge/global/stopp_start/start_v3.yaml   (40 START entries)
  knowledge/beers/beers_criteria_2023.yaml     (57 Beers entries)

Inserts/upserts into kb4_explicit_criteria (one unified table) with a
criterion_set discriminator (STOPP_V3 / START_V3 / BEERS_2023).

These three sources together form the Wave 3 explicit-criteria backbone:
the geriatric prescribing rules ACOP rule engines actually evaluate. AU
PIMs 2024 (Wang, IMJ) is a separate source not yet in the repo and would
be added here as a fourth criterion_set when source data lands.

Usage
    python3 scripts/load_explicit_criteria.py
    python3 scripts/load_explicit_criteria.py --dry-run
    python3 scripts/load_explicit_criteria.py --set STOPP_V3   # only one set

Run from kb-4-patient-safety directory:
    cd backend/shared-infrastructure/knowledge-base-services/kb-4-patient-safety
    python3 scripts/load_explicit_criteria.py
"""

from __future__ import annotations

import argparse
import hashlib
import json
import logging
import sys
from datetime import date, datetime
from pathlib import Path

import psycopg2
import psycopg2.extras
import yaml

logging.basicConfig(level=logging.INFO, format="%(message)s")
log = logging.getLogger(__name__)

KB4 = Path(__file__).resolve().parent.parent
DSN = dict(
    host="localhost", port=5440, user="kb4_safety_user",
    password="kb4_safety_password", dbname="kb4_patient_safety",
)

SOURCES = {
    "STOPP_V3":      ("knowledge/global/stopp_start/stopp_v3.yaml",     "stopp_entries"),
    "START_V3":      ("knowledge/global/stopp_start/start_v3.yaml",     "start_entries"),
    "BEERS_2023":    ("knowledge/beers/beers_criteria_2023.yaml",       "beers_entries"),
    # Wave 3 follow-up + Wave 4 partial — additional AU / global safety lists
    "AU_APINCHS":    ("knowledge/au/high-alert/apinchs.yaml",           "entries"),
    "TGA_BLACKBOX":  ("knowledge/au/blackbox/tga_blackbox.yaml",        "entries"),
    "TGA_PREGNANCY": ("knowledge/au/pregnancy/tga_pregnancy.yaml",      "entries"),
    "ACB_SCALE":     ("knowledge/global/anticholinergic/acb_scale.yaml", "entries"),
}


def _to_date(s) -> date | None:
    if not s:
        return None
    if isinstance(s, date):
        return s
    if isinstance(s, datetime):
        return s.date()
    try:
        return datetime.strptime(str(s)[:10], "%Y-%m-%d").date()
    except ValueError:
        return None


def _sha256(obj) -> str:
    return hashlib.sha256(json.dumps(obj, sort_keys=True, default=str).encode()).hexdigest()


def _build_criteria_text(criterion_set: str, entry: dict) -> str:
    """Construct criteria_text for sources that don't expose it directly.

    Different YAMLs use different field names for the rule sentence; this
    function maps each criterion_set to its natural canonical phrasing.
    """
    explicit = entry.get("criteria") or entry.get("criteriaText")
    if explicit:
        return explicit

    drug = entry.get("drugName") or "Drug"

    if criterion_set == "AU_APINCHS":
        cat = entry.get("categoryName") or entry.get("category") or "high-alert"
        risk = entry.get("riskLevel") or "high"
        factors = entry.get("riskFactors") or []
        risk_summary = factors[0] if factors else "specific safety requirements apply"
        return (
            f"{drug} is a high-alert medication ({cat}, risk: {risk}). "
            f"Key risk: {risk_summary}."
        )
    if criterion_set == "TGA_BLACKBOX":
        title = entry.get("warningTitle") or "Boxed warning"
        text = entry.get("warningText") or ""
        return f"TGA black-box warning ({title}) for {drug}. {text}".strip()
    if criterion_set == "TGA_PREGNANCY":
        cat = entry.get("category") or "?"
        desc = entry.get("categoryDescription") or ""
        return f"TGA pregnancy category {cat} for {drug}. {desc}".strip()
    if criterion_set == "ACB_SCALE":
        score = entry.get("acbScore")
        risk = entry.get("riskLevel") or ""
        cog = entry.get("cognitiveRisk") or ""
        return f"{drug} carries ACB score {score} ({risk} anticholinergic burden). {cog}".strip()
    return f"{drug}: {criterion_set} entry"


def _normalize_entry(criterion_set: str, entry: dict) -> dict:
    """Map a YAML entry to kb4_explicit_criteria row fields."""
    g = entry.get("governance") or {}

    # criterion_id: STOPP/START use 'id'; Beers/APINCHs/TGA/ACB use rxnorm
    crit_id = entry.get("id")
    if not crit_id:
        crit_id = entry.get("rxnorm") or f"{entry.get('drugName', 'unk')}-{_sha256(entry)[:8]}"

    # rxnorm handling — sources with single rxnorm use rxnorm_code_primary
    if criterion_set in ("BEERS_2023", "AU_APINCHS", "TGA_BLACKBOX",
                         "TGA_PREGNANCY", "ACB_SCALE"):
        rxnorm_primary = entry.get("rxnorm")
        rxnorm_codes = None
    else:
        rxnorm_primary = None
        rxnorm_codes = entry.get("rxnormCodes") or []

    # condition handling
    if criterion_set == "START_V3":
        condition_text = entry.get("condition")
        condition_icd10 = entry.get("conditionICD10") or []
    elif criterion_set == "BEERS_2023":
        cta = entry.get("conditionsToAvoid") or []
        if isinstance(cta, list) and cta:
            condition_text = "; ".join(
                (c.get("display") or c.get("code") or "") if isinstance(c, dict) else str(c)
                for c in cta
            )
        else:
            condition_text = None
        condition_icd10 = []
    elif criterion_set == "TGA_PREGNANCY":
        condition_text = "Pregnancy (any trimester)"
        condition_icd10 = []
    else:
        condition_text = None
        condition_icd10 = []

    return dict(
        criterion_set              = criterion_set,
        criterion_id               = str(crit_id),
        section                    = entry.get("section"),
        section_name               = entry.get("sectionName"),
        drug_class                 = entry.get("drugClass"),
        drug_name                  = entry.get("drugName"),
        rxnorm_codes               = rxnorm_codes,
        rxnorm_code_primary        = rxnorm_primary,
        atc_code                   = entry.get("atcCode"),
        condition_text             = condition_text,
        condition_icd10            = condition_icd10,
        conditions_to_avoid        = entry.get("conditionsToAvoid"),
        recommended_drugs          = entry.get("recommendedDrugs"),
        recommendation             = entry.get("recommendation"),
        criteria_text              = _build_criteria_text(criterion_set, entry),
        rationale                  = (entry.get("rationale")
                                      or entry.get("clinicalGuidance")  # TGA pregnancy
                                      or entry.get("warningText")        # TGA blackbox
                                      or entry.get("dementiaRisk")       # ACB
                                      or entry.get("geriatricRisk")),    # ACB
        exceptions                 = entry.get("exceptions"),
        evidence_level             = entry.get("evidenceLevel") or g.get("evidenceLevel"),
        quality_of_evidence        = entry.get("qualityOfEvidence"),
        strength_of_recommendation = entry.get("strengthOfRecommendation")
                                      or entry.get("severity"),          # TGA blackbox
        acb_score                  = entry.get("acbScore"),
        alternatives               = entry.get("alternatives"),
        source_authority           = g.get("sourceAuthority"),
        source_document            = g.get("sourceDocument"),
        source_url                 = g.get("sourceUrl"),
        source_section             = g.get("sourceSection"),
        jurisdiction               = g.get("jurisdiction"),
        knowledge_version          = g.get("knowledgeVersion"),
        effective_date             = _to_date(g.get("effectiveDate")),
        review_date                = _to_date(g.get("reviewDate")),
        approval_status            = g.get("approvalStatus"),
        governance                 = g,
        raw_yaml                   = entry,
    )


def apply_migration(conn, name: str) -> None:
    log.info("Applying migration: %s", name)
    sql = (KB4 / "migrations" / name).read_text()
    with conn.cursor() as cur:
        cur.execute(sql)


def load_set(conn, criterion_set: str, file_rel: str, list_key: str, dry_run: bool) -> int:
    src = KB4 / file_rel
    if not src.exists():
        log.warning("  source not found: %s", src)
        return 0
    doc = yaml.safe_load(src.read_text())
    entries = doc.get(list_key) or []
    log.info("  %s: parsed %d entries from %s", criterion_set, len(entries), src.name)

    if dry_run:
        return len(entries)

    cols = [
        "criterion_set", "criterion_id", "section", "section_name", "drug_class", "drug_name",
        "rxnorm_codes", "rxnorm_code_primary", "atc_code",
        "condition_text", "condition_icd10", "conditions_to_avoid",
        "recommended_drugs", "recommendation", "criteria_text", "rationale", "exceptions",
        "evidence_level", "quality_of_evidence", "strength_of_recommendation",
        "acb_score", "alternatives",
        "source_authority", "source_document", "source_url", "source_section",
        "jurisdiction", "knowledge_version", "effective_date", "review_date",
        "approval_status", "governance", "raw_yaml",
    ]
    placeholders = ", ".join(["%s"] * len(cols))
    col_csv = ", ".join(cols)

    n = 0
    with conn.cursor() as cur:
        cur.execute(
            "DELETE FROM kb4_explicit_criteria WHERE criterion_set = %s",
            (criterion_set,),
        )
        for e in entries:
            row = _normalize_entry(criterion_set, e)
            try:
                cur.execute(
                    f"""
                    INSERT INTO kb4_explicit_criteria ({col_csv}) VALUES ({placeholders})
                    ON CONFLICT (criterion_set, criterion_id) DO UPDATE SET
                        section = EXCLUDED.section, criteria_text = EXCLUDED.criteria_text,
                        rationale = EXCLUDED.rationale, raw_yaml = EXCLUDED.raw_yaml,
                        loaded_at = now()
                    """,
                    [
                        row["criterion_set"], row["criterion_id"], row["section"],
                        row["section_name"], row["drug_class"], row["drug_name"],
                        row["rxnorm_codes"], row["rxnorm_code_primary"], row["atc_code"],
                        row["condition_text"], row["condition_icd10"],
                        psycopg2.extras.Json(row["conditions_to_avoid"]) if row["conditions_to_avoid"] is not None else None,
                        row["recommended_drugs"], row["recommendation"],
                        row["criteria_text"], row["rationale"], row["exceptions"],
                        row["evidence_level"], row["quality_of_evidence"],
                        row["strength_of_recommendation"],
                        row["acb_score"], row["alternatives"],
                        row["source_authority"], row["source_document"], row["source_url"],
                        row["source_section"],
                        row["jurisdiction"], row["knowledge_version"],
                        row["effective_date"], row["review_date"],
                        row["approval_status"],
                        psycopg2.extras.Json(row["governance"]) if row["governance"] else None,
                        psycopg2.extras.Json(row["raw_yaml"]),
                    ],
                )
                n += 1
            except Exception as ex:
                log.warning("  insert failed for %s/%s: %s",
                            criterion_set, row["criterion_id"], str(ex)[:200])

        cur.execute(
            """
            INSERT INTO kb4_explicit_criteria_load_log (criterion_set, source_file, rows_loaded)
            VALUES (%s, %s, %s)
            """,
            (criterion_set, file_rel, n),
        )
    return n


def main() -> int:
    p = argparse.ArgumentParser(
        description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter
    )
    p.add_argument("--set", choices=list(SOURCES.keys()),
                   help="Load only one criterion set (default: all)")
    p.add_argument("--dry-run", action="store_true",
                   help="Parse only; no DB writes")
    args = p.parse_args()

    targets = {args.set: SOURCES[args.set]} if args.set else SOURCES

    log.info("Connecting to KB-4 patient_safety (kb4-patient-safety-postgres:5440)")
    conn = psycopg2.connect(**DSN)
    conn.autocommit = True
    try:
        if not args.dry_run:
            apply_migration(conn, "005_explicit_criteria.sql")

        totals: dict[str, int] = {}
        for cset, (file_rel, list_key) in targets.items():
            n = load_set(conn, cset, file_rel, list_key, args.dry_run)
            totals[cset] = n

        if not args.dry_run:
            apply_migration(conn, "006_explicit_criteria_indexes.sql")

        log.info("=" * 60)
        log.info("WAVE 3 LOAD COMPLETE")
        log.info("=" * 60)
        for cset, n in totals.items():
            log.info("  %-12s %d", cset, n)
        log.info("  %-12s %d", "TOTAL", sum(totals.values()))
    finally:
        conn.close()

    return 0


if __name__ == "__main__":
    sys.exit(main())
