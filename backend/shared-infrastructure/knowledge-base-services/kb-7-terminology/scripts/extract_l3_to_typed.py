"""
Per-KB typed extractors: kb_l3_staging -> each KB's clinical tables.

Reads the L3 staging rows already loaded into each KB's Postgres
(see load_l3_facts_staging.py), uses the resolved_rxcui populated by
resolve_rxnorm_codes.py, and INSERTs into each KB's typed clinical
tables so the KB service endpoints can query real data.

Per-KB target:
  KB-1   contextual? no — DOSING -> drug_rules (existing)
  KB-4   SAFETY -> kb4_l3_safety_facts (new)
  KB-16  MONITORING -> kb16_l3_lab_requirements (new)
  KB-20  CONTEXTUAL -> adverse_reaction_profiles + context_modifiers (existing)

Each extractor is idempotent — re-running replaces existing rows that
came from the same source_file. Source provenance (source_file,
source_authority) is preserved on every typed row.

Usage
    python3 scripts/extract_l3_to_typed.py             # all 4 KBs
    python3 scripts/extract_l3_to_typed.py --kb 1      # just KB-1
    python3 scripts/extract_l3_to_typed.py --dry-run   # report only
"""

from __future__ import annotations

import argparse
import hashlib
import json
import logging
import sys
from dataclasses import dataclass

import psycopg2
import psycopg2.extras

logging.basicConfig(level=logging.INFO, format="%(message)s")
log = logging.getLogger(__name__)

DSN = {
    "kb1":  dict(host="localhost", port=5481, user="kb1_user",
                 password="kb1_password", dbname="kb1_drug_rules"),
    "kb4":  dict(host="localhost", port=5440, user="kb4_safety_user",
                 password="kb4_safety_password", dbname="kb4_patient_safety"),
    "kb16": dict(host="localhost", port=5446, user="kb16_user",
                 password="kb_password", dbname="kb_lab_interpretation"),
    "kb20": dict(host="localhost", port=5436, user="kb20_user",
                 password="kb20_password", dbname="kb_service_20"),
}


# --------- DDL for new typed tables ----------

DDL_KB4 = """
CREATE TABLE IF NOT EXISTS kb4_l3_safety_facts (
    id                 BIGSERIAL    PRIMARY KEY,
    source_file        TEXT         NOT NULL,
    rxnorm_code        VARCHAR(50),
    drug_name          VARCHAR(200) NOT NULL,
    drug_class         VARCHAR(200),
    fact_type          VARCHAR(20)  NOT NULL,    -- 'CONTRAINDICATION' | 'WARNING'
    severity           VARCHAR(20),              -- CRITICAL | HIGH | MODERATE | LOW
    type               VARCHAR(20),              -- absolute | relative
    condition_codes    TEXT[],                   -- ICD codes
    condition_descs    TEXT[],
    snomed_codes       TEXT[],
    lab_parameter      VARCHAR(100),
    lab_loinc          VARCHAR(50),
    lab_threshold      NUMERIC(12,4),
    lab_operator       VARCHAR(5),
    lab_unit           VARCHAR(20),
    clinical_rationale TEXT,
    risk_description   TEXT,
    source_authority   VARCHAR(50),
    source_document    VARCHAR(200),
    source_section     VARCHAR(100),
    evidence_level     VARCHAR(10),
    raw_fact           JSONB        NOT NULL,
    loaded_at          TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_kb4_l3_drug ON kb4_l3_safety_facts (drug_name);
CREATE INDEX IF NOT EXISTS idx_kb4_l3_rxcui ON kb4_l3_safety_facts (rxnorm_code) WHERE rxnorm_code IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_kb4_l3_type ON kb4_l3_safety_facts (fact_type, severity);
"""

DDL_KB16 = """
CREATE TABLE IF NOT EXISTS kb16_l3_lab_requirements (
    id                 BIGSERIAL    PRIMARY KEY,
    source_file        TEXT         NOT NULL,
    rxnorm_code        VARCHAR(50),
    drug_name          VARCHAR(200) NOT NULL,
    drug_class         VARCHAR(200),
    lab_name           VARCHAR(200) NOT NULL,
    loinc_code         VARCHAR(50),              -- the canonical lab code
    frequency          TEXT,
    baseline_timing    TEXT,
    initial_monitoring TEXT,
    maintenance_freq   TEXT,
    target_low         NUMERIC(12,4),
    target_high        NUMERIC(12,4),
    target_unit        VARCHAR(20),
    target_population  TEXT,
    critical_high_op   VARCHAR(5),
    critical_high_val  NUMERIC(12,4),
    critical_high_unit VARCHAR(20),
    critical_high_act  TEXT,
    critical_low_op    VARCHAR(5),
    critical_low_val   NUMERIC(12,4),
    action_required    TEXT,
    clinical_context   TEXT,
    source_authority   VARCHAR(50),
    source_document    VARCHAR(200),
    raw_fact           JSONB        NOT NULL,
    loaded_at          TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_kb16_l3_drug ON kb16_l3_lab_requirements (drug_name);
CREATE INDEX IF NOT EXISTS idx_kb16_l3_loinc ON kb16_l3_lab_requirements (loinc_code) WHERE loinc_code IS NOT NULL AND loinc_code <> '<UNKNOWN>';
CREATE INDEX IF NOT EXISTS idx_kb16_l3_rxcui ON kb16_l3_lab_requirements (rxnorm_code) WHERE rxnorm_code IS NOT NULL;
"""


@dataclass
class Counts:
    inserted: int = 0
    skipped: int = 0
    errors: int = 0


def sha256_of(obj) -> str:
    return hashlib.sha256(json.dumps(obj, sort_keys=True, default=str).encode()).hexdigest()


def fetch_staging(dsn: dict) -> list[dict]:
    with psycopg2.connect(**dsn) as conn:
        conn.set_session(readonly=False)
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            cur.execute("""
                SELECT id, source_file, drug_name, domain, source_guideline,
                       extractor_version, extraction_date, quality_grade,
                       resolved_rxcui, resolved_rxcui_source, raw_json
                FROM kb_l3_staging
                WHERE fact_count > 0
                ORDER BY id
            """)
            return [dict(r) for r in cur.fetchall()]


# ---------- KB-1 dosing -> drug_rules ----------

def extract_kb1(dry_run: bool) -> Counts:
    c = Counts()
    rows = fetch_staging(DSN["kb1"])
    if not rows:
        log.info("  no fact-bearing rows in KB-1 staging")
        return c

    if dry_run:
        c.skipped = len(rows)
        return c

    conn = psycopg2.connect(**DSN["kb1"])
    conn.autocommit = True
    try:
        with conn.cursor() as cur:
            cur.execute("DELETE FROM drug_rules WHERE source_set_id LIKE 'L3:%'")
        for r in rows:
            try:
                with conn.cursor() as cur:
                    rxcui = r["resolved_rxcui"] or f"UNRESOLVED:{r['drug_name']}"
                    rule_data = r["raw_json"]
                    src_hash = sha256_of(rule_data)
                    cur.execute("""
                        INSERT INTO drug_rules
                            (rxnorm_code, jurisdiction, drug_name, rule_data,
                             authority, document_name, document_section, source_set_id,
                             source_hash, version, evidence_level)
                        VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                        ON CONFLICT (rxnorm_code, jurisdiction) DO UPDATE SET
                            drug_name        = EXCLUDED.drug_name,
                            rule_data        = EXCLUDED.rule_data,
                            authority        = EXCLUDED.authority,
                            document_name    = EXCLUDED.document_name,
                            source_set_id    = EXCLUDED.source_set_id,
                            source_hash      = EXCLUDED.source_hash,
                            version          = EXCLUDED.version,
                            ingested_at      = now()
                    """, (
                        rxcui, "GLOBAL", r["drug_name"],
                        psycopg2.extras.Json(rule_data),
                        rule_data.get("sourceGuideline", "KDIGO 2022"),
                        rule_data.get("sourceGuideline", "KDIGO 2022 Diabetes in CKD"),
                        None, f"L3:{r['source_file']}",
                        src_hash, r["extractor_version"] or "v3.0.0-facts", None,
                    ))
                c.inserted += 1
            except Exception as e:
                log.warning("  KB-1 INSERT failed for %s: %s", r["source_file"], e)
                c.errors += 1
    finally:
        conn.close()
    return c


# ---------- KB-20 contextual -> adverse_reaction_profiles + context_modifiers ----------

def extract_kb20(dry_run: bool) -> Counts:
    c = Counts()
    rows = fetch_staging(DSN["kb20"])

    if dry_run:
        c.skipped = sum(
            len(r["raw_json"].get("adrProfiles", []) or []) +
            len(r["raw_json"].get("standaloneModifiers", []) or [])
            for r in rows
        )
        return c

    # KB-20 strategy: ADR profiles -> adverse_reaction_profiles. Skip
    # context_modifiers because production constraint chk_context_modifiers_effect
    # requires Bayesian directions (INCREASE_PRIOR/DECREASE_PRIOR) that L3
    # vocabulary doesn't produce. Modifiers stay in staging for human curation.
    conn = psycopg2.connect(**DSN["kb20"])
    conn.autocommit = True
    try:
        with conn.cursor() as cur:
            cur.execute("DELETE FROM adverse_reaction_profiles WHERE source = 'L3_KDIGO'")
        for r in rows:
            raw = r["raw_json"] or {}
            rxcui = r["resolved_rxcui"]
            for adr in raw.get("adrProfiles") or []:
                try:
                    gov = adr.get("governance") or {}
                    with conn.cursor() as cur:
                        cur.execute("""
                            INSERT INTO adverse_reaction_profiles
                                (rx_norm_code, drug_name, drug_class, reaction,
                                 reaction_snomed, mechanism, symptom, onset_window,
                                 onset_category, frequency, severity, risk_factors,
                                 source, completeness_grade, source_snippet,
                                 source_authority, source_document, source_section,
                                 evidence_level)
                            VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s)
                        """, (
                            (rxcui or "")[:50],
                            (adr.get("drugName") or r["drug_name"] or "unspecified")[:200],
                            (adr.get("drugClass") or "unspecified")[:50],
                            adr.get("reaction") or "unspecified",
                            (adr.get("reactionSnomed") or "")[:50] or None,
                            adr.get("mechanism"),
                            (adr.get("symptom") or "")[:100] or None,
                            (adr.get("onsetWindow") or "")[:50] or None,
                            (adr.get("onsetCategory") or "")[:20] or None,
                            (adr.get("frequency") or "")[:20] or None,
                            (adr.get("severity") or "")[:20] or None,
                            adr.get("riskFactors") or [],
                            "L3_KDIGO",
                            (adr.get("completenessGrade") or "STUB")[:10],
                            adr.get("sourceSnippet"),
                            (gov.get("sourceAuthority") or "")[:50] or None,
                            (gov.get("sourceDocument") or "")[:200] or None,
                            (gov.get("sourceSection") or "")[:100] or None,
                            (gov.get("evidenceLevel") or "")[:10] or None,
                        ))
                    c.inserted += 1
                except Exception as e:
                    log.warning("  KB-20 ADR INSERT fail (%s): %s",
                                r["source_file"], str(e)[:200])
                    c.errors += 1
                    continue
    finally:
        conn.close()
    return c


# ---------- KB-4 safety -> kb4_l3_safety_facts (new) ----------

def extract_kb4(dry_run: bool) -> Counts:
    c = Counts()
    rows = fetch_staging(DSN["kb4"])

    if dry_run:
        n = sum(
            len(r["raw_json"].get("contraindications", []) or []) +
            len(r["raw_json"].get("warnings", []) or [])
            for r in rows
        )
        c.skipped = n
        return c

    conn = psycopg2.connect(**DSN["kb4"])
    conn.autocommit = True
    try:
        with conn.cursor() as cur:
            cur.execute(DDL_KB4)
            cur.execute("DELETE FROM kb4_l3_safety_facts")
        for r in rows:
            raw = r["raw_json"] or {}
            rxcui = r["resolved_rxcui"]
            for kind, fact_list in (("CONTRAINDICATION", raw.get("contraindications") or []),
                                     ("WARNING",         raw.get("warnings") or [])):
                for f in fact_list:
                    try:
                        with conn.cursor() as cur:
                            cur.execute("""
                                INSERT INTO kb4_l3_safety_facts
                                    (source_file, rxnorm_code, drug_name, drug_class,
                                     fact_type, severity, type,
                                     condition_codes, condition_descs, snomed_codes,
                                     lab_parameter, lab_loinc, lab_threshold,
                                     lab_operator, lab_unit,
                                     clinical_rationale, risk_description,
                                     source_authority, source_document, source_section,
                                     evidence_level, raw_fact)
                                VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,
                                        %s,%s,%s,%s,%s,%s,%s)
                            """, (
                                r["source_file"], rxcui,
                                f.get("drugName") or r["drug_name"],
                                f.get("drugClass"),
                                kind, f.get("severity"), f.get("type"),
                                f.get("conditionCodes") or [],
                                f.get("conditionDescriptions") or [],
                                f.get("snomedCodes") or [],
                                f.get("labParameter"),
                                f.get("labLoinc"),
                                f.get("labThreshold"),
                                f.get("labOperator"),
                                f.get("labUnit"),
                                f.get("clinicalRationale"),
                                f.get("riskDescription"),
                                r["source_guideline"] or "KDIGO",
                                r["source_guideline"],
                                None, None,
                                psycopg2.extras.Json(f),
                            ))
                        c.inserted += 1
                    except Exception as e:
                        log.warning("  KB-4 INSERT fail (%s): %s",
                                    r["source_file"], str(e)[:200])
                        c.errors += 1
                        continue
    finally:
        conn.close()
    return c


# ---------- KB-16 monitoring -> kb16_l3_lab_requirements (new) ----------

def extract_kb16(dry_run: bool) -> Counts:
    c = Counts()
    rows = fetch_staging(DSN["kb16"])

    if dry_run:
        n = 0
        for r in rows:
            for d in r["raw_json"].get("labRequirements", []) or []:
                n += len(d.get("labs") or [])
        c.skipped = n
        return c

    conn = psycopg2.connect(**DSN["kb16"])
    conn.autocommit = True
    try:
        with conn.cursor() as cur:
            cur.execute(DDL_KB16)
            cur.execute("DELETE FROM kb16_l3_lab_requirements")
        for r in rows:
            raw = r["raw_json"] or {}
            rxcui = r["resolved_rxcui"]
            for req in raw.get("labRequirements") or []:
                drug_name = req.get("drugName") or r["drug_name"]
                drug_class = req.get("drugClass")
                for lab in req.get("labs") or []:
                    try:
                        tr = lab.get("targetRange") or {}
                        ch = lab.get("criticalHigh") or {}
                        cl = lab.get("criticalLow") or {}
                        loinc = lab.get("loincCode")
                        if loinc == "<UNKNOWN>":
                            loinc = None
                        with conn.cursor() as cur:
                            cur.execute("""
                                INSERT INTO kb16_l3_lab_requirements
                                    (source_file, rxnorm_code, drug_name, drug_class,
                                     lab_name, loinc_code, frequency, baseline_timing,
                                     initial_monitoring, maintenance_freq,
                                     target_low, target_high, target_unit, target_population,
                                     critical_high_op, critical_high_val, critical_high_unit,
                                     critical_high_act,
                                     critical_low_op, critical_low_val,
                                     action_required, clinical_context,
                                     source_authority, source_document, raw_fact)
                                VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,
                                        %s,%s,%s,%s,
                                        %s,%s,%s,%s,
                                        %s,%s,
                                        %s,%s,%s,%s,%s)
                            """, (
                                r["source_file"], rxcui, drug_name, drug_class,
                                lab.get("labName") or "unspecified",
                                loinc,
                                lab.get("frequency"),
                                lab.get("baselineTiming"),
                                lab.get("initialMonitoring"),
                                lab.get("maintenanceFrequency"),
                                tr.get("low"), tr.get("high"),
                                tr.get("unit"), tr.get("populationContext"),
                                ch.get("operator"), ch.get("value"),
                                ch.get("unit"), ch.get("action"),
                                cl.get("operator"), cl.get("value"),
                                lab.get("actionRequired"),
                                lab.get("clinicalContext"),
                                "KDIGO", r["source_guideline"],
                                psycopg2.extras.Json(lab),
                            ))
                        c.inserted += 1
                    except Exception as e:
                        log.warning("  KB-16 INSERT fail (%s): %s",
                                    r["source_file"], str(e)[:200])
                        c.errors += 1
                        continue
    finally:
        conn.close()
    return c


# ---------- main ----------

EXTRACTORS = {
    "1":  ("KB-1 drug-rules",           extract_kb1),
    "4":  ("KB-4 patient-safety",       extract_kb4),
    "16": ("KB-16 lab-interpretation",  extract_kb16),
    "20": ("KB-20 patient-profile",     extract_kb20),
}


def main() -> int:
    p = argparse.ArgumentParser(
        description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter
    )
    p.add_argument("--kb", choices=list(EXTRACTORS.keys()),
                   help="Extract only one KB (default: all)")
    p.add_argument("--dry-run", action="store_true",
                   help="Count only, no DB writes")
    args = p.parse_args()

    keys = [args.kb] if args.kb else list(EXTRACTORS.keys())
    log.info("=" * 70)
    log.info("L3 -> typed clinical tables")
    log.info("=" * 70)
    totals: dict[str, Counts] = {}
    for k in keys:
        label, fn = EXTRACTORS[k]
        log.info("--- %s ---", label)
        try:
            c = fn(args.dry_run)
        except Exception as e:
            log.error("  fatal: %s", e)
            c = Counts(errors=1)
        log.info("  inserted=%d skipped=%d errors=%d",
                 c.inserted, c.skipped, c.errors)
        totals[label] = c

    log.info("=" * 70)
    log.info("SUMMARY")
    log.info("=" * 70)
    grand = sum(c.inserted for c in totals.values())
    grand_err = sum(c.errors for c in totals.values())
    for label, c in totals.items():
        log.info("  %-40s inserted=%4d errors=%d", label, c.inserted, c.errors)
    log.info("  %-40s inserted=%4d errors=%d", "TOTAL", grand, grand_err)
    return 0 if grand_err == 0 else 1


if __name__ == "__main__":
    sys.exit(main())
