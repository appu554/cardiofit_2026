"""
PBS (Australian Pharmaceutical Benefits Scheme) loader for KB-6.

Parses a PBS Schedule extract (XML or CSV) and inserts into the
kb6_pbs_* tables created by migrations 005 + 006.

PBS distributes monthly extracts at https://www.pbs.gov.au/info/browse/downloads
(and also via https://data.gov.au search for "pharmaceutical benefits scheme").
The exact URL/format moves between releases — this loader is file-based so
it works against any extract you've downloaded.

Usage
    python3 scripts/load_pbs.py --xml /path/to/pbs-items.xml
    python3 scripts/load_pbs.py --csv /path/to/PBS_LIMITS.csv --schedule-date 2026-04-01
    python3 scripts/load_pbs.py --xml /path/to/pbs-items.xml --dry-run

The XML parser is tolerant of common element/attribute name variants
(handles both legacy and current PBS schemas). For unfamiliar shapes,
run with --dry-run first to see what was parsed.

Run from kb-6-formulary directory:
    cd backend/shared-infrastructure/knowledge-base-services/kb-6-formulary
    python3 scripts/load_pbs.py --xml ...
"""

from __future__ import annotations

import argparse
import csv
import logging
import re
import sys
import xml.etree.ElementTree as ET
from dataclasses import dataclass, field
from datetime import date, datetime
from pathlib import Path

import psycopg2
import psycopg2.extras

logging.basicConfig(level=logging.INFO, format="%(message)s")
log = logging.getLogger(__name__)

KB6_DSN = dict(
    host="localhost", port=5447, user="kb6_admin",
    password="kb6_secure_password", dbname="kb_formulary",
)


@dataclass
class ParsedItem:
    pbs_code: str
    drug_name: str
    drug_class: str | None = None
    form: str | None = None
    strength: str | None = None
    manner_of_administration: str | None = None
    max_quantity: int | None = None
    max_repeats: int | None = None
    pack_size: int | None = None
    pack_quantity: int | None = None
    schedule_section: str | None = None
    is_authority_required: bool = False
    is_streamlined: bool = False
    is_restricted: bool = False
    is_section_100: bool = False
    is_palliative_care: bool = False
    is_chemotherapy: bool = False
    amt_mp_sctid: int | None = None
    amt_mpuu_sctid: int | None = None
    amt_tpp_sctid: int | None = None
    amt_ctpp_sctid: int | None = None
    rxnorm_code: str | None = None
    effective_date: date | None = None
    end_date: date | None = None
    raw: dict = field(default_factory=dict)
    authorities: list[dict] = field(default_factory=list)
    restrictions: list[dict] = field(default_factory=list)
    prescriber_types: list[str] = field(default_factory=list)
    section_100_type: str | None = None
    indications: list[dict] = field(default_factory=list)


# ---------- Helpers ----------

def _attr_or_text(elem: ET.Element, *names: str) -> str | None:
    """Return first non-empty value among attributes or child text matching any of `names`."""
    for n in names:
        v = elem.get(n)
        if v not in (None, "", "null"):
            return v
    for n in names:
        child = elem.find(n)
        if child is not None and child.text and child.text.strip():
            return child.text.strip()
        # Try with any namespace
        for c in elem:
            if _localname(c.tag).lower() == n.lower() and c.text and c.text.strip():
                return c.text.strip()
    return None


def _localname(tag: str) -> str:
    return tag.split("}", 1)[-1] if "}" in tag else tag


def _to_int(s: str | None) -> int | None:
    if s is None:
        return None
    try:
        return int(re.sub(r"[^\d-]", "", s) or "0") or None
    except ValueError:
        return None


def _to_bool(s: str | None) -> bool:
    if s is None:
        return False
    return s.strip().lower() in ("true", "1", "yes", "y")


def _to_date(s: str | None) -> date | None:
    if not s:
        return None
    s = s.strip()
    for fmt in ("%Y-%m-%d", "%d/%m/%Y", "%Y%m%d", "%d-%m-%Y"):
        try:
            return datetime.strptime(s, fmt).date()
        except ValueError:
            continue
    return None


def _classify_schedule(text: str | None) -> tuple[str, dict]:
    """Map PBS schedule_section text to canonical enum + flags."""
    if not text:
        return "GENERAL", {}
    t = text.lower()
    flags = {
        "is_authority_required": "authority" in t and "streamlin" not in t,
        "is_streamlined":        "streamlin" in t,
        "is_restricted":         "restrict" in t,
        "is_section_100":        "section 100" in t or "s100" in t or "hsd" in t or "raahs" in t,
        "is_palliative_care":    "palliative" in t,
        "is_chemotherapy":       "chemotherapy" in t or "chemo" in t,
    }
    if flags["is_section_100"]:
        if "raahs" in t: section = "S100_RAAHS"
        else:            section = "S100_HSD"
    elif flags["is_chemotherapy"]: section = "CHEMO"
    elif flags["is_palliative_care"]: section = "PALLIATIVE"
    elif flags["is_authority_required"]: section = "AUTHORITY"
    elif flags["is_streamlined"]:        section = "STREAMLINED"
    elif flags["is_restricted"]:         section = "RESTRICTED"
    else: section = "GENERAL"
    return section, flags


# ---------- XML parsing ----------

def parse_xml(path: Path) -> list[ParsedItem]:
    log.info("Parsing PBS XML: %s (%.1f MB)", path, path.stat().st_size / (1024 * 1024))
    tree = ET.parse(path)
    root = tree.getroot()

    items: list[ParsedItem] = []
    for elem in root.iter():
        if _localname(elem.tag).lower() not in ("item", "drug", "pbsitem", "drug-item"):
            continue
        code = _attr_or_text(elem, "code", "pbs_code", "ItemCode", "PBSCode", "pbsCode") or ""
        if not code:
            continue
        drug_name = _attr_or_text(elem, "drug_name", "DrugName", "name", "Name", "GenericName", "li_drug_name") or ""
        if not drug_name:
            continue

        sched_text = _attr_or_text(elem, "schedule_section", "ScheduleSection", "schedule",
                                   "Schedule", "Section", "ScheduleSubsection", "li_schedule_section")
        section, flags = _classify_schedule(sched_text)

        item = ParsedItem(
            pbs_code=code,
            drug_name=drug_name,
            drug_class=_attr_or_text(elem, "drug_class", "DrugClass", "class"),
            form=_attr_or_text(elem, "form", "Form", "DoseForm", "li_form"),
            strength=_attr_or_text(elem, "strength", "Strength", "PackQty", "li_strength"),
            manner_of_administration=_attr_or_text(
                elem, "manner_of_administration", "MannerOfAdministration", "Route"),
            max_quantity=_to_int(_attr_or_text(elem, "max_quantity", "MaxQuantity", "MaxQty", "li_max_qty")),
            max_repeats=_to_int(_attr_or_text(elem, "max_repeats", "MaxRepeats", "MaxRpts", "li_max_repeats")),
            pack_size=_to_int(_attr_or_text(elem, "pack_size", "PackSize", "li_pack_size")),
            pack_quantity=_to_int(_attr_or_text(elem, "pack_quantity", "PackQuantity", "li_pack_qty")),
            schedule_section=section,
            is_authority_required=flags.get("is_authority_required", False),
            is_streamlined=flags.get("is_streamlined", False),
            is_restricted=flags.get("is_restricted", False),
            is_section_100=flags.get("is_section_100", False),
            is_palliative_care=flags.get("is_palliative_care", False),
            is_chemotherapy=flags.get("is_chemotherapy", False),
            amt_mp_sctid=_to_int(_attr_or_text(elem, "amt_mp_sctid", "AMTMpId", "MP_SCTID")),
            amt_mpuu_sctid=_to_int(_attr_or_text(elem, "amt_mpuu_sctid", "AMTMpuuId", "MPUU_SCTID")),
            amt_tpp_sctid=_to_int(_attr_or_text(elem, "amt_tpp_sctid", "AMTTppId", "TPP_SCTID")),
            amt_ctpp_sctid=_to_int(_attr_or_text(elem, "amt_ctpp_sctid", "AMTCtppId", "CTPP_SCTID")),
            effective_date=_to_date(_attr_or_text(elem, "effective_date", "EffectiveDate", "StartDate")),
            end_date=_to_date(_attr_or_text(elem, "end_date", "EndDate", "ExpiryDate")),
            raw={"source_tag": _localname(elem.tag), "attrs": dict(elem.attrib)},
        )

        # Authorities — child <Authority> elements
        for child in elem:
            if _localname(child.tag).lower() in ("authority", "authorityrequirement"):
                item.authorities.append({
                    "authority_type": _attr_or_text(child, "type", "AuthorityType")
                                      or ("STREAMLINED" if item.is_streamlined else "AUTHORITY_REQUIRED"),
                    "authority_code": _attr_or_text(child, "code", "AuthorityCode"),
                    "description":    _attr_or_text(child, "description", "Description") or (child.text or "").strip(),
                    "requires_specialist": _to_bool(_attr_or_text(child, "specialist", "RequiresSpecialist")),
                    "requires_consultant": _to_bool(_attr_or_text(child, "consultant", "RequiresConsultant")),
                })
            elif _localname(child.tag).lower() in ("restriction", "clinicalcriteria"):
                rt = _attr_or_text(child, "text", "Text", "Description") or (child.text or "").strip()
                if rt:
                    item.restrictions.append({
                        "restriction_text": rt,
                        "indication_code": _attr_or_text(child, "indication_code", "IndicationCode"),
                        "is_initial":     _to_bool(_attr_or_text(child, "initial", "IsInitial")),
                        "is_continuing":  _to_bool(_attr_or_text(child, "continuing", "IsContinuing")),
                    })
            elif _localname(child.tag).lower() in ("prescribertype", "prescriber"):
                pt = _attr_or_text(child, "type", "Type") or (child.text or "").strip()
                if pt:
                    item.prescriber_types.append(pt.upper())
            elif _localname(child.tag).lower() in ("indication", "approvedindication"):
                it = _attr_or_text(child, "text", "Text", "IndicationText") or (child.text or "").strip()
                if it:
                    item.indications.append({
                        "indication_text": it,
                        "icd10am_codes":   [], "snomed_codes": [],
                    })

        if item.is_section_100:
            sec_text = (sched_text or "").lower()
            if "raahs" in sec_text:                 item.section_100_type = "RAAHS"
            elif "methadone" in sec_text:           item.section_100_type = "METHADONE"
            elif "growth hormone" in sec_text:      item.section_100_type = "GROWTH_HORMONE"
            elif "chemotherapy" in sec_text:        item.section_100_type = "CHEMOTHERAPY"
            elif "ivf" in sec_text:                 item.section_100_type = "IVF"
            elif "botulinum" in sec_text:           item.section_100_type = "BOTULINUM_TOXIN"
            else:                                   item.section_100_type = "HSD"

        items.append(item)

    log.info("  parsed %d items", len(items))
    return items


# ---------- CSV parsing ----------

def _classify_csv_row(row: dict) -> tuple[str, dict]:
    """Map official PBS API CSV columns to schedule classification.

    Uses benefit_type_code (A/S/R/U), section100_only_indicator (Y/N),
    and program_code. Falls back to text-based classification of any
    schedule_section column for non-PBS-API CSV formats.
    """
    btc = (row.get("benefit_type_code") or "").strip().upper()
    sec100 = (row.get("section100_only_indicator") or "").strip().upper() == "Y"
    program = (row.get("program_code") or "").strip().upper()

    if not btc:
        # Fallback to legacy text-based classification
        sched_text = (row.get("schedule_section") or row.get("ScheduleSection")
                      or row.get("li_schedule_section") or "")
        return _classify_schedule(sched_text)

    flags = {
        "is_authority_required": btc == "A",
        "is_streamlined":        btc == "S",
        "is_restricted":         btc == "R",
        "is_section_100":        sec100,
        "is_palliative_care":    program in ("PL", "EP"),
        "is_chemotherapy":       program in ("CT", "MF"),
    }
    if sec100:
        if program == "HS":   section = "S100_HSD"
        else:                 section = "S100_HSD"
    elif btc == "A": section = "AUTHORITY"
    elif btc == "S": section = "STREAMLINED"
    elif btc == "R": section = "RESTRICTED"
    elif program in ("PL", "EP"): section = "PALLIATIVE"
    elif program in ("CT", "MF"): section = "CHEMO"
    else: section = "GENERAL"
    return section, flags


def parse_csv(path: Path) -> list[ParsedItem]:
    log.info("Parsing PBS CSV: %s (%.1f MB)", path, path.stat().st_size / (1024 * 1024))
    items: list[ParsedItem] = []
    with path.open("r", encoding="utf-8", newline="") as f:
        reader = csv.DictReader(f)
        for row in reader:
            code = (row.get("pbs_code") or row.get("PBS_Code") or row.get("ItemCode")
                    or row.get("li_item_id") or "").strip()
            drug = (row.get("drug_name") or row.get("DrugName") or row.get("li_drug_name") or "").strip()
            if not code or not drug:
                continue
            section, flags = _classify_csv_row(row)
            item = ParsedItem(
                pbs_code=code, drug_name=drug,
                drug_class=row.get("drug_class") or row.get("DrugClass"),
                form=row.get("form") or row.get("li_form") or row.get("schedule_form"),
                strength=row.get("strength") or row.get("li_strength"),
                manner_of_administration=(row.get("manner_of_administration")
                                          or row.get("moa_preferred_term")
                                          or row.get("Route")),
                max_quantity=_to_int(row.get("max_quantity")
                                     or row.get("maximum_quantity_units")
                                     or row.get("li_max_qty")),
                max_repeats=_to_int(row.get("max_repeats")
                                    or row.get("number_of_repeats")
                                    or row.get("li_max_repeats")),
                pack_size=_to_int(row.get("pack_size") or row.get("li_pack_size")),
                pack_quantity=_to_int(row.get("pack_quantity")
                                      or row.get("pricing_quantity")
                                      or row.get("li_pack_qty")),
                schedule_section=section,
                is_authority_required=flags.get("is_authority_required", False),
                is_streamlined=flags.get("is_streamlined", False),
                is_restricted=flags.get("is_restricted", False),
                is_section_100=flags.get("is_section_100", False),
                is_palliative_care=flags.get("is_palliative_care", False),
                is_chemotherapy=flags.get("is_chemotherapy", False),
                amt_mpuu_sctid=_to_int(row.get("amt_mpuu_sctid") or row.get("MPUU_SCTID")),
                amt_ctpp_sctid=_to_int(row.get("amt_ctpp_sctid") or row.get("CTPP_SCTID")),
                effective_date=_to_date(row.get("effective_date")
                                        or row.get("first_listed_date")
                                        or row.get("EffectiveDate")),
                end_date=_to_date(row.get("end_date") or row.get("non_effective_date")),
                raw=dict(row),
            )
            items.append(item)
    log.info("  parsed %d items", len(items))
    return items


# ---------- DB load ----------

def load_to_db(items: list[ParsedItem], schedule_date: date, source_file: str) -> dict:
    counts = {"items": 0, "authorities": 0, "restrictions": 0,
              "prescriber_types": 0, "section_100": 0, "indications": 0,
              "errors": 0}

    conn = psycopg2.connect(**KB6_DSN)
    conn.autocommit = True
    try:
        for item in items:
            try:
                with conn.cursor() as cur:
                    cur.execute("""
                        INSERT INTO kb6_pbs_items
                            (pbs_code, drug_name, drug_class, form, strength,
                             manner_of_administration, max_quantity, max_repeats,
                             pack_size, pack_quantity, schedule_section,
                             is_authority_required, is_streamlined, is_restricted,
                             is_section_100, is_palliative_care, is_chemotherapy,
                             amt_mp_sctid, amt_mpuu_sctid, amt_tpp_sctid, amt_ctpp_sctid,
                             rxnorm_code, effective_date, end_date,
                             schedule_publish_date, raw_xml)
                        VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,
                                %s,%s,%s,%s,%s,%s,%s,%s,%s)
                        ON CONFLICT (pbs_code) DO UPDATE SET
                            drug_name             = EXCLUDED.drug_name,
                            schedule_section      = EXCLUDED.schedule_section,
                            is_authority_required = EXCLUDED.is_authority_required,
                            is_streamlined        = EXCLUDED.is_streamlined,
                            is_section_100        = EXCLUDED.is_section_100,
                            schedule_publish_date = EXCLUDED.schedule_publish_date,
                            raw_xml               = EXCLUDED.raw_xml,
                            loaded_at             = now()
                    """, (
                        item.pbs_code, item.drug_name, item.drug_class, item.form,
                        item.strength, item.manner_of_administration,
                        item.max_quantity, item.max_repeats,
                        item.pack_size, item.pack_quantity, item.schedule_section,
                        item.is_authority_required, item.is_streamlined, item.is_restricted,
                        item.is_section_100, item.is_palliative_care, item.is_chemotherapy,
                        item.amt_mp_sctid, item.amt_mpuu_sctid, item.amt_tpp_sctid, item.amt_ctpp_sctid,
                        item.rxnorm_code, item.effective_date, item.end_date,
                        schedule_date, psycopg2.extras.Json(item.raw),
                    ))
                counts["items"] += 1

                # Children
                with conn.cursor() as cur:
                    cur.execute("DELETE FROM kb6_pbs_authorities      WHERE pbs_code = %s", (item.pbs_code,))
                    cur.execute("DELETE FROM kb6_pbs_restrictions     WHERE pbs_code = %s", (item.pbs_code,))
                    cur.execute("DELETE FROM kb6_pbs_prescriber_types WHERE pbs_code = %s", (item.pbs_code,))
                    cur.execute("DELETE FROM kb6_pbs_section_100      WHERE pbs_code = %s", (item.pbs_code,))
                    cur.execute("DELETE FROM kb6_pbs_indications      WHERE pbs_code = %s", (item.pbs_code,))

                for a in item.authorities:
                    with conn.cursor() as cur:
                        cur.execute("""
                            INSERT INTO kb6_pbs_authorities
                                (pbs_code, authority_type, authority_code, description,
                                 requires_specialist, requires_consultant)
                            VALUES (%s,%s,%s,%s,%s,%s)
                        """, (item.pbs_code, a.get("authority_type"), a.get("authority_code"),
                              a.get("description"),
                              a.get("requires_specialist", False),
                              a.get("requires_consultant", False)))
                    counts["authorities"] += 1

                for r in item.restrictions:
                    with conn.cursor() as cur:
                        cur.execute("""
                            INSERT INTO kb6_pbs_restrictions
                                (pbs_code, restriction_text, indication_code, is_initial, is_continuing)
                            VALUES (%s,%s,%s,%s,%s)
                        """, (item.pbs_code, r["restriction_text"], r.get("indication_code"),
                              r.get("is_initial", False), r.get("is_continuing", False)))
                    counts["restrictions"] += 1

                for pt in item.prescriber_types:
                    with conn.cursor() as cur:
                        cur.execute("""
                            INSERT INTO kb6_pbs_prescriber_types (pbs_code, prescriber_type)
                            VALUES (%s,%s)
                        """, (item.pbs_code, pt))
                    counts["prescriber_types"] += 1

                if item.is_section_100 and item.section_100_type:
                    with conn.cursor() as cur:
                        cur.execute("""
                            INSERT INTO kb6_pbs_section_100 (pbs_code, section_100_type)
                            VALUES (%s,%s)
                        """, (item.pbs_code, item.section_100_type))
                    counts["section_100"] += 1

                for ind in item.indications:
                    with conn.cursor() as cur:
                        cur.execute("""
                            INSERT INTO kb6_pbs_indications (pbs_code, indication_text, icd10am_codes, snomed_codes)
                            VALUES (%s,%s,%s,%s)
                        """, (item.pbs_code, ind["indication_text"],
                              ind.get("icd10am_codes") or [], ind.get("snomed_codes") or []))
                    counts["indications"] += 1
            except Exception as e:
                log.warning("  insert failed for item=%s: %s", item.pbs_code, str(e)[:200])
                counts["errors"] += 1

        # Audit
        with conn.cursor() as cur:
            for tbl, n in counts.items():
                if tbl in ("errors",):
                    continue
                cur.execute(
                    """
                    INSERT INTO kb6_pbs_load_log
                        (schedule_date, source_file, table_name, rows_loaded)
                    VALUES (%s, %s, %s, %s)
                    """,
                    (schedule_date, source_file, f"kb6_pbs_{tbl}", n),
                )
    finally:
        conn.close()

    return counts


# ---------- Main ----------

def main() -> int:
    p = argparse.ArgumentParser(description=__doc__,
                                formatter_class=argparse.RawDescriptionHelpFormatter)
    src = p.add_mutually_exclusive_group(required=True)
    src.add_argument("--xml", type=Path, help="Path to PBS XML extract")
    src.add_argument("--csv", type=Path, help="Path to PBS CSV extract")
    p.add_argument("--schedule-date", default=None,
                   help="Schedule effective date YYYY-MM-DD (default: today)")
    p.add_argument("--dry-run", action="store_true",
                   help="Parse only, no DB writes")
    args = p.parse_args()

    schedule_date = _to_date(args.schedule_date) or date.today()

    if args.xml:
        items = parse_xml(args.xml)
        source_file = str(args.xml)
    else:
        items = parse_csv(args.csv)
        source_file = str(args.csv)

    if args.dry_run:
        log.info("DRY RUN — would load:")
        log.info("  items: %d", len(items))
        for tag, getter in (
            ("authorities", lambda i: len(i.authorities)),
            ("restrictions", lambda i: len(i.restrictions)),
            ("prescriber_types", lambda i: len(i.prescriber_types)),
            ("indications", lambda i: len(i.indications)),
        ):
            log.info("  %s (sum across items): %d", tag, sum(getter(i) for i in items))
        return 0

    counts = load_to_db(items, schedule_date, source_file)
    log.info("=" * 60)
    log.info("PBS LOAD COMPLETE")
    log.info("=" * 60)
    for k, v in counts.items():
        log.info("  %-18s %d", k, v)
    return 0 if counts["errors"] == 0 else 1


if __name__ == "__main__":
    sys.exit(main())
