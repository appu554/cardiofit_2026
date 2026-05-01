"""
Tier 2 RxCUI remapper — YAML-driven name lookup + apply.

Strategy: each KB-4 explicit-criteria YAML carries drug names alongside
RxCUIs. When an RxCUI in the YAML doesn't resolve in KB-7 RxNorm, look up
the corresponding drug name(s) in RxNav-in-a-Box's name index, and verify
the candidate exists in KB-7. Confidence is HIGH when (a) the entry has
exactly as many drug names as RxCUIs, AND (b) every other RxCUI in the
entry already matches a name's lookup, AND (c) the candidate is in KB-7.

This is a strict version of the v1 dry-run script: it actually mutates
YAMLs when run with --apply. Without --apply, it produces a manifest only.

Reads:
    knowledge/global/stopp_start/{stopp_v3.yaml, start_v3.yaml}
    knowledge/beers/beers_criteria_2023.yaml
    knowledge/au/{high-alert/apinchs.yaml, blackbox/tga_blackbox.yaml,
                   pregnancy/tga_pregnancy.yaml,
                   pims_wang_2024/wang_2024_pims.yaml}
    knowledge/global/anticholinergic/acb_scale.yaml
    KB-4 DB for unresolved RxCUI list
    KB-7 DB for current-RxNorm reference

Writes:
    claudedocs/audits/2026-04-30_retired_rxcui_remap_manifest_v2.{md,json}
    With --apply: mutates the 8 YAMLs in place + writes a backup as
    <yaml>.pre-remap-2026-04-30.bak

Usage:
    python3 scripts/remap_retired_rxcuis_v2.py            # dry-run (default)
    python3 scripts/remap_retired_rxcuis_v2.py --apply    # mutate YAMLs
"""

from __future__ import annotations

import argparse
import json
import logging
import re
import shutil
import sys
from collections import Counter, defaultdict
from datetime import datetime, timezone
from pathlib import Path
from urllib.error import URLError
from urllib.request import urlopen

import psycopg2
import yaml

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)-7s %(message)s")
log = logging.getLogger(__name__)

KB4_DSN = dict(
    host="localhost", port=5440, user="kb4_safety_user",
    password="kb4_safety_password", dbname="kb4_patient_safety",
)
KB7_DSN = dict(
    host="localhost", port=5457, user="postgres",
    password="password", dbname="kb_terminology",
)
RXNAV_BASE = "http://localhost:4000"

KB4_DIR = (Path(__file__).resolve().parent.parent.parent
           / "kb-4-patient-safety" / "knowledge")

YAML_FILES = [
    KB4_DIR / "global" / "stopp_start" / "stopp_v3.yaml",
    KB4_DIR / "global" / "stopp_start" / "start_v3.yaml",
    KB4_DIR / "beers" / "beers_criteria_2023.yaml",
    KB4_DIR / "au" / "high-alert" / "apinchs.yaml",
    KB4_DIR / "au" / "blackbox" / "tga_blackbox.yaml",
    KB4_DIR / "au" / "pregnancy" / "tga_pregnancy.yaml",
    KB4_DIR / "global" / "anticholinergic" / "acb_scale.yaml",
    KB4_DIR / "au" / "pims_wang_2024" / "wang_2024_pims.yaml",
]


def _fetch_json(url: str, timeout: float = 3.0) -> dict | None:
    try:
        with urlopen(url, timeout=timeout) as resp:
            return json.load(resp)
    except (URLError, json.JSONDecodeError, OSError):
        return None


def _split_drug_names(raw: str | list | None) -> list[str]:
    """Split a multi-drug name field into individual ingredient names.

    'Celecoxib, Etoricoxib' -> ['Celecoxib', 'Etoricoxib']
    ['L-DOPA', 'Dopamine agonist'] -> as-is
    'Insulin Regular (Human)' -> ['Insulin Regular (Human)']
    """
    if not raw:
        return []
    if isinstance(raw, list):
        names = raw
    else:
        names = re.split(r"[,/]|\\sand\\s|\\s&\\s", raw)
    cleaned = [n.strip() for n in names if n and n.strip()]
    return cleaned


def _entry_name_candidates(entry: dict) -> list[str]:
    """Pull all possible drug-name strings from a YAML entry, in priority order."""
    out: list[str] = []
    for k in ("drugName", "drugClass", "recommendedDrugs"):
        v = entry.get(k)
        if v:
            out.extend(_split_drug_names(v))
    # Deduplicate while preserving order
    seen = set()
    return [n for n in out if not (n in seen or seen.add(n))]


def lookup_name_in_rxnav(name: str) -> list[dict]:
    """Query RxNav for all RxCUIs matching this drug name (ingredient-level)."""
    if not name or len(name) < 3:
        return []
    nq = name.replace(" ", "+")
    drugs = _fetch_json(f"{RXNAV_BASE}/REST/drugs.json?name={nq}") or {}
    groups = (drugs.get("drugGroup") or {}).get("conceptGroup") or []
    results: list[dict] = []
    for grp in groups:
        if grp.get("tty") in ("IN", "MIN", "PIN"):
            for cp in grp.get("conceptProperties") or []:
                results.append({
                    "rxcui": cp.get("rxcui"),
                    "name": cp.get("name"),
                    "tty": cp.get("tty"),
                })
    # Also try the simpler /rxcui.json?name= lookup (returns single ID)
    res = _fetch_json(f"{RXNAV_BASE}/REST/rxcui.json?name={nq}") or {}
    rxcui_id = ((res.get("idGroup") or {}).get("rxnormId") or [None])[0]
    if rxcui_id and not any(r["rxcui"] == rxcui_id for r in results):
        results.insert(0, {"rxcui": rxcui_id, "name": name, "tty": "IN"})
    return results


def get_kb7_rxnorm() -> set[str]:
    with psycopg2.connect(**KB7_DSN) as conn, conn.cursor() as cur:
        cur.execute("SELECT code FROM concepts_rxnorm")
        return {r[0] for r in cur.fetchall()}


def get_kb4_unresolved(ref: set[str]) -> tuple[set[str], set[str]]:
    with psycopg2.connect(**KB4_DSN) as conn, conn.cursor() as cur:
        cur.execute(
            "SELECT DISTINCT rxnorm_code_primary FROM kb4_explicit_criteria "
            "WHERE rxnorm_code_primary IS NOT NULL AND rxnorm_code_primary <> ''"
        )
        primary = {r[0] for r in cur.fetchall()}
        cur.execute(
            "SELECT DISTINCT unnest(rxnorm_codes)::text FROM kb4_explicit_criteria "
            "WHERE rxnorm_codes IS NOT NULL AND array_length(rxnorm_codes,1)>0"
        )
        array = {r[0] for r in cur.fetchall() if r[0]}
    return primary - ref, array - ref


def build_yaml_index() -> dict[str, list[dict]]:
    """Map rxcui -> list of {yaml, entry_id, names, all_rxcuis_in_entry}."""
    index: dict[str, list[dict]] = defaultdict(list)
    for path in YAML_FILES:
        if not path.exists():
            log.warning("missing YAML: %s", path)
            continue
        data = yaml.safe_load(path.read_text())
        # Different YAMLs use different list keys
        entries = (data.get("entries") or data.get("stopp_entries")
                   or data.get("start_entries") or data.get("beers_entries") or [])
        for entry in entries:
            entry_id = (entry.get("id") or entry.get("rxnorm")
                        or entry.get("drugName") or "?")
            names = _entry_name_candidates(entry)
            rxcuis_array = [str(r) for r in (entry.get("rxnormCodes") or [])]
            rxcui_single = str(entry["rxnorm"]) if entry.get("rxnorm") else None
            entry_rxcuis = list(rxcuis_array)
            if rxcui_single:
                entry_rxcuis.append(rxcui_single)
            record = {
                "yaml": str(path),
                "entry_id": entry_id,
                "names": names,
                "all_rxcuis_in_entry": entry_rxcuis,
                "rxnorm_field": "rxnorm" if rxcui_single else "rxnormCodes",
            }
            for rx in entry_rxcuis:
                index[rx].append(record)
    return index


def classify_unresolved(rxcui: str,
                        yaml_records: list[dict],
                        kb7_ref: set[str]) -> dict:
    """Pick a HIGH-confidence remap target for one unresolved RxCUI by:
       (a) iterating all entries containing this RxCUI
       (b) for each entry, identifying which drug name in the entry is unmatched
           (i.e. all OTHER rxcuis in the entry already match a name lookup)
       (c) looking up that name in RxNav and verifying in KB-7"""
    if not yaml_records:
        return {"rxcui": rxcui, "confidence": "manual",
                "reason": "no YAML entry references this RxCUI",
                "candidates": []}

    suggestions: list[dict] = []
    for rec in yaml_records:
        names = rec["names"]
        all_rxcuis = rec["all_rxcuis_in_entry"]
        if not names:
            continue

        # Look up every name; build name->kb7_resolving_rxcui map
        name_lookups: dict[str, list[dict]] = {}
        for name in names:
            cands = lookup_name_in_rxnav(name)
            kb7_cands = [c for c in cands if c["rxcui"] in kb7_ref]
            name_lookups[name] = kb7_cands

        # Match name_lookups against all_rxcuis: which names already covered?
        covered_names: set[str] = set()
        for name, cands in name_lookups.items():
            for c in cands:
                if c["rxcui"] in all_rxcuis:
                    covered_names.add(name)
                    break

        # The unmatched (uncovered) names are the candidates for THIS rxcui
        unmatched_names = [n for n in names if n not in covered_names]

        for name in unmatched_names:
            cands = name_lookups[name]
            if len(cands) == 1:
                suggestions.append({
                    "yaml": rec["yaml"],
                    "entry_id": rec["entry_id"],
                    "drug_name": name,
                    "candidate": cands[0]["rxcui"],
                    "candidate_name": cands[0]["name"],
                    "candidate_tty": cands[0]["tty"],
                    "confidence": "high",
                    "reason": (f"Single KB-7-verified ingredient match for "
                               f"'{name}' in entry {rec['entry_id']}"),
                })
            elif len(cands) > 1:
                suggestions.append({
                    "yaml": rec["yaml"],
                    "entry_id": rec["entry_id"],
                    "drug_name": name,
                    "candidate": None,
                    "all_candidates": cands,
                    "confidence": "review",
                    "reason": (f"{len(cands)} KB-7-verified candidates for "
                               f"'{name}' — pick one"),
                })

    if not suggestions:
        return {"rxcui": rxcui, "confidence": "manual",
                "reason": "YAML has names but no candidates resolve in KB-7",
                "candidates": []}

    # Best confidence wins
    high = [s for s in suggestions if s["confidence"] == "high"]
    if high:
        return {"rxcui": rxcui, "confidence": "high",
                "reason": high[0]["reason"], "candidates": high}
    return {"rxcui": rxcui, "confidence": "review",
            "reason": suggestions[0]["reason"], "candidates": suggestions}


def render_markdown(manifest: dict) -> str:
    out = [
        "# Retired-RxCUI Remediation Manifest v2 (YAML-name-lookup)",
        "",
        f"**Generated:** {manifest['generated_at']}",
        f"**Total unresolved:** {manifest['total_unresolved']}",
        "",
        "## Confidence summary",
        "",
        "| Confidence | Count | Action |",
        "|------------|------:|--------|",
    ]
    actions = {
        "high":   "✅ auto-apply candidate (single KB-7-verified ingredient match)",
        "review": "👀 multiple KB-7-verified candidates — human picks one",
        "manual": "🔍 no candidate from YAML name lookup — investigate",
    }
    for k in ("high", "review", "manual"):
        n = manifest["confidence_summary"].get(k, 0)
        if n:
            out.append(f"| {k} | {n} | {actions[k]} |")

    out += ["", "## Per-RxCUI proposed remaps (HIGH confidence)", "",
            "| Old RxCUI | YAML | Entry | Drug name | New RxCUI | Reason |",
            "|-----------|------|-------|-----------|-----------|--------|"]
    for entry in manifest["entries"]:
        if entry["confidence"] != "high":
            continue
        c = entry["candidates"][0]
        yaml_short = c["yaml"].split("/knowledge/", 1)[-1]
        out.append(
            f"| {entry['rxcui']} | {yaml_short} | {c['entry_id']} | "
            f"{c['drug_name']} | {c['candidate']} ({c['candidate_name']}) | "
            f"{c['reason']} |"
        )

    review_entries = [e for e in manifest["entries"] if e["confidence"] == "review"]
    if review_entries:
        out += ["", "## Per-RxCUI candidates needing review", "",
                "| Old RxCUI | YAML | Entry | Drug name | Candidate count |",
                "|-----------|------|-------|-----------|----------------:|"]
        for entry in review_entries:
            for c in entry["candidates"]:
                yaml_short = c["yaml"].split("/knowledge/", 1)[-1]
                out.append(
                    f"| {entry['rxcui']} | {yaml_short} | {c['entry_id']} | "
                    f"{c['drug_name']} | {len(c.get('all_candidates') or [])} |"
                )

    manual_entries = [e for e in manifest["entries"] if e["confidence"] == "manual"]
    if manual_entries:
        out += ["", "## Manual-review needed", "",
                "| Old RxCUI | Reason |",
                "|-----------|--------|"]
        for entry in manual_entries:
            out.append(f"| {entry['rxcui']} | {entry['reason']} |")

    return "\n".join(out)


def apply_high_confidence_remaps(manifest: dict) -> dict:
    """Mutate YAMLs in place; backup originals first.

    Replaces the old RxCUI text with the new in the YAML files, only for
    HIGH-confidence entries. Returns dict counting per-file changes.
    """
    changes: dict[str, int] = defaultdict(int)
    file_remaps: dict[str, list[tuple[str, str]]] = defaultdict(list)
    for entry in manifest["entries"]:
        if entry["confidence"] != "high":
            continue
        c = entry["candidates"][0]
        file_remaps[c["yaml"]].append((entry["rxcui"], c["candidate"]))

    today = datetime.now(timezone.utc).strftime("%Y-%m-%d")
    for path, remaps in file_remaps.items():
        p = Path(path)
        backup = p.with_suffix(p.suffix + f".pre-remap-{today}.bak")
        shutil.copyfile(p, backup)
        text = p.read_text()
        original = text
        for old, new in remaps:
            # Replace occurrences of "<old>" (quoted) — the YAML format
            text = text.replace(f'"{old}"', f'"{new}"')
        if text != original:
            p.write_text(text)
            changes[str(p)] = sum(1 for old, _ in remaps if f'"{old}"' in original)
        log.info("  %s: %d remap(s), backup at %s",
                 p.name, changes[str(p)], backup.name)
    return dict(changes)


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__,
                                formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("--apply", action="store_true",
                   help="Mutate YAMLs for HIGH confidence entries (default: dry-run)")
    p.add_argument("--output-md", type=Path,
                   default=Path("/Volumes/Vaidshala/cardiofit/claudedocs/audits/"
                                "2026-04-30_retired_rxcui_remap_manifest_v2.md"))
    p.add_argument("--output-json", type=Path,
                   default=Path("/Volumes/Vaidshala/cardiofit/claudedocs/audits/"
                                "2026-04-30_retired_rxcui_remap_manifest_v2.json"))
    args = p.parse_args()

    log.info("Loading KB-7 RxNorm reference...")
    kb7_ref = get_kb7_rxnorm()
    log.info("  %d codes", len(kb7_ref))

    log.info("Computing KB-4 unresolved RxCUIs...")
    primary, array = get_kb4_unresolved(kb7_ref)
    all_unresolved = sorted(primary | array)
    log.info("  unresolved: %d (primary %d, array %d, both %d)",
             len(all_unresolved),
             len(primary - array), len(array - primary), len(primary & array))

    log.info("Building YAML index...")
    yaml_index = build_yaml_index()
    log.info("  YAML index covers %d distinct RxCUIs", len(yaml_index))

    log.info("Classifying via name lookup...")
    classified: list[dict] = []
    for i, rxcui in enumerate(all_unresolved):
        records = yaml_index.get(rxcui, [])
        c = classify_unresolved(rxcui, records, kb7_ref)
        c["columns"] = ["primary"] if rxcui in primary else []
        if rxcui in array:
            c["columns"].append("array")
        classified.append(c)
        if (i + 1) % 20 == 0:
            log.info("  ... %d/%d", i + 1, len(all_unresolved))

    counts = Counter(c["confidence"] for c in classified)
    manifest = {
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "total_unresolved": len(all_unresolved),
        "confidence_summary": dict(counts),
        "entries": classified,
    }

    args.output_json.parent.mkdir(parents=True, exist_ok=True)
    args.output_json.write_text(json.dumps(manifest, indent=2))
    args.output_md.write_text(render_markdown(manifest))

    log.info("=" * 70)
    log.info("MANIFEST GENERATED")
    log.info("  Confidence: %s", dict(counts))
    log.info("  Markdown:   %s", args.output_md)
    log.info("  JSON:       %s", args.output_json)

    if args.apply:
        log.info("=" * 70)
        log.info("APPLYING HIGH-confidence remaps to YAMLs...")
        changes = apply_high_confidence_remaps(manifest)
        log.info("APPLY COMPLETE: %d remaps across %d files",
                 sum(changes.values()), len(changes))
    else:
        log.info("(dry-run; pass --apply to mutate YAMLs)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
