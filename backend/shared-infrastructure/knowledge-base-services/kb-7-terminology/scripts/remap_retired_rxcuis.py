"""
Retired-RxCUI remap manifest generator (Tier 1 — Quality / Integrity).

For every unresolved RxCUI in KB-4 (per the cross-KB validator), classify
against RxNav-in-a-Box and produce a remediation manifest.

Output is a markdown + JSON manifest. THIS SCRIPT DOES NOT MUTATE YAMLs.
Apply step is deliberately separate — RxCUIs are clinical safety values,
mutations require human review of the manifest first.

Three remap strategies, in order of confidence:
  1. RxNav historystatus.derivedConcepts.remappedConcept — direct target
  2. RxNav drugs.json?name=<name> — name-based lookup when name is known
  3. Manual review needed — no name, no target

Usage:
    cd kb-7-terminology
    python3 scripts/remap_retired_rxcuis.py
    python3 scripts/remap_retired_rxcuis.py --output /tmp/remap.md

Reads from KB-4 (kb4_explicit_criteria) directly via psycopg2; queries
RxNav-in-a-Box at localhost:4000.
"""

from __future__ import annotations

import argparse
import json
import logging
import sys
from collections import Counter
from datetime import datetime, timezone
from pathlib import Path
from urllib.error import URLError
from urllib.request import urlopen

import psycopg2

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


def _fetch_json(url: str, timeout: float = 3.0) -> dict | None:
    try:
        with urlopen(url, timeout=timeout) as resp:
            return json.load(resp)
    except (URLError, json.JSONDecodeError, OSError):
        return None


def get_unresolved_rxcuis() -> tuple[set[str], set[str]]:
    """Pull KB-4 RxCUIs and KB-7 RxNorm reference, return their set difference.

    Returns (primary_unresolved, array_unresolved).
    """
    with psycopg2.connect(**KB7_DSN) as kc, kc.cursor() as cur:
        cur.execute("SELECT code FROM concepts_rxnorm")
        ref = {r[0] for r in cur.fetchall()}
    log.info("KB-7 RxNorm reference: %d codes", len(ref))

    with psycopg2.connect(**KB4_DSN) as kc4, kc4.cursor() as cur:
        cur.execute(
            "SELECT DISTINCT rxnorm_code_primary FROM kb4_explicit_criteria "
            "WHERE rxnorm_code_primary IS NOT NULL AND rxnorm_code_primary <> ''"
        )
        primary = {r[0] for r in cur.fetchall()}
        cur.execute(
            "SELECT DISTINCT unnest(rxnorm_codes)::text "
            "FROM kb4_explicit_criteria "
            "WHERE rxnorm_codes IS NOT NULL AND array_length(rxnorm_codes,1)>0"
        )
        array = {r[0] for r in cur.fetchall() if r[0]}

    log.info("KB-4 distinct RxCUIs: primary=%d, array=%d", len(primary), len(array))
    return primary - ref, array - ref


def classify_rxcui(rxcui: str) -> dict:
    """Query RxNav historystatus + (if name known) drugs.json by name."""
    hs = _fetch_json(f"{RXNAV_BASE}/REST/rxcui/{rxcui}/historystatus.json") or {}
    history = hs.get("rxcuiStatusHistory") or {}
    meta = history.get("metaData") or {}
    attrs = history.get("attributes") or {}
    derived = history.get("derivedConcepts") or {}

    status = meta.get("status") or "TruePhantom"
    name = (attrs.get("name") or "").strip()
    tty = (attrs.get("tty") or "").strip()

    # Strategy 1: derivedConcepts remap targets
    remap_targets: list[str] = []
    for bucket in ("remappedConcept", "quantifiedConcept"):
        for entry in derived.get(bucket) or []:
            target = entry.get("remappedRxCui") or entry.get("quantifiedRxCui")
            if target:
                remap_targets.append(str(target))

    # Strategy 2: drugs.json by name (only when we have a usable name)
    name_candidates: list[dict] = []
    if name and name not in ("PROPRIETARY", "Gas") and len(name) > 3:
        # Use cleaned ingredient if name has dose/form; just use first token
        # for ingredient-like search. Conservative — let user verify in manifest.
        nq = name.replace(" ", "+")
        drugs = _fetch_json(
            f"{RXNAV_BASE}/REST/drugs.json?name={nq}", timeout=3.0
        ) or {}
        groups = (drugs.get("drugGroup") or {}).get("conceptGroup") or []
        # Prefer IN (ingredient) tty for ingredient-name lookups
        for grp in groups:
            if grp.get("tty") in ("IN", "MIN", "PIN"):
                for cp in grp.get("conceptProperties") or []:
                    name_candidates.append({
                        "rxcui": cp.get("rxcui"),
                        "name": cp.get("name"),
                        "tty": cp.get("tty"),
                    })
                break

    # Best-confidence suggestion
    if remap_targets:
        suggestion = remap_targets[0]
        confidence = "high"
        reason = f"RxNav derivedConcepts.remappedConcept ({status})"
    elif len(name_candidates) == 1:
        suggestion = name_candidates[0]["rxcui"]
        confidence = "medium"
        reason = f"single by-name match for '{name}'"
    elif len(name_candidates) > 1:
        suggestion = None
        confidence = "review"
        reason = f"{len(name_candidates)} by-name candidates for '{name}'"
    else:
        suggestion = None
        confidence = "manual"
        reason = "no remap target, no by-name candidates"

    return {
        "rxcui": rxcui,
        "status": status,
        "name": name,
        "tty": tty,
        "remap_targets": remap_targets,
        "name_candidates": name_candidates,
        "suggested_replacement": suggestion,
        "confidence": confidence,
        "reason": reason,
    }


def build_manifest(primary: set[str], array: set[str]) -> dict:
    log.info("Classifying %d unresolved RxCUIs via RxNav...",
             len(primary | array))
    all_unresolved = sorted(primary | array)
    classified: list[dict] = []
    for i, rxcui in enumerate(all_unresolved):
        rec = classify_rxcui(rxcui)
        rec["columns"] = []
        if rxcui in primary:
            rec["columns"].append("rxnorm_code_primary")
        if rxcui in array:
            rec["columns"].append("rxnorm_codes")
        classified.append(rec)
        if (i + 1) % 25 == 0:
            log.info("  ... %d/%d", i + 1, len(all_unresolved))

    counts = Counter(r["confidence"] for r in classified)
    return {
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "total_unresolved": len(all_unresolved),
        "primary_only": len(primary - array),
        "array_only": len(array - primary),
        "both": len(primary & array),
        "confidence_summary": dict(counts),
        "entries": classified,
    }


def render_markdown(manifest: dict) -> str:
    out = [
        "# Retired-RxCUI Remediation Manifest",
        "",
        f"**Generated:** {manifest['generated_at']}",
        f"**Total unresolved:** {manifest['total_unresolved']} "
        f"(primary-only {manifest['primary_only']}, array-only "
        f"{manifest['array_only']}, both {manifest['both']})",
        "",
        "## Confidence summary",
        "",
        "| Confidence | Count | Action |",
        "|------------|------:|--------|",
    ]
    actions = {
        "high":   "✅ auto-remap candidate (RxNav derivedConcepts target)",
        "medium": "⚠️ verify single by-name match before applying",
        "review": "👀 multiple by-name candidates — pick one",
        "manual": "🔍 no automated suggestion — open YAML, find ingredient",
    }
    for k, v in sorted(manifest["confidence_summary"].items()):
        out.append(f"| {k} | {v} | {actions.get(k, '')} |")

    out += [
        "",
        "## Per-RxCUI detail",
        "",
        "| RxCUI | Columns | Status | Name (RxNav) | Suggested → | Confidence | Reason |",
        "|-------|---------|--------|--------------|-------------|------------|--------|",
    ]
    for r in manifest["entries"]:
        cols = ",".join(c.replace("rxnorm_", "rx") for c in r["columns"])
        name = (r["name"] or "—")[:40]
        suggestion = r["suggested_replacement"] or "—"
        out.append(
            f"| {r['rxcui']} | {cols} | {r['status']} | {name} | "
            f"{suggestion} | {r['confidence']} | {r['reason']} |"
        )

    out += [
        "",
        "## Apply steps (NOT performed automatically)",
        "",
        "After human review of this manifest:",
        "1. For `confidence=high` rows: update the corresponding YAML",
        "   (`stopp_v3.yaml`, `start_v3.yaml`, `beers_criteria_2023.yaml`,",
        "   `apinchs.yaml`, `tga_blackbox.yaml`, `tga_pregnancy.yaml`,",
        "   `acb_scale.yaml`, `wang_2024_pims.yaml`) replacing old RxCUI with",
        "   `suggested_replacement`. Add comment trail: `# remapped 2026-04-30:",
        "   <old> -> <new> per RxNav historystatus`.",
        "2. For `confidence=medium` rows: spot-check the single candidate is",
        "   the same active ingredient before applying.",
        "3. For `confidence=review` and `manual`: open YAML, find which",
        "   criterion the RxCUI belongs to, decide whether to keep retired or",
        "   pick a current code based on clinical intent.",
        "4. Re-run loaders: `python3 scripts/load_explicit_criteria.py`",
        "5. Re-run validator: `python3 scripts/validate_kb_codes.py --rxnav`",
        "",
        "Expected outcome: KB-4 RxNorm coverage 80% → ~95%+ after high+medium",
        "confidence remaps applied.",
    ]
    return "\n".join(out)


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__,
                                formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("--output-md", type=Path,
                   default=Path("/Volumes/Vaidshala/cardiofit/claudedocs/audits/"
                                "2026-04-30_retired_rxcui_remap_manifest.md"))
    p.add_argument("--output-json", type=Path,
                   default=Path("/Volumes/Vaidshala/cardiofit/claudedocs/audits/"
                                "2026-04-30_retired_rxcui_remap_manifest.json"))
    args = p.parse_args()

    primary, array = get_unresolved_rxcuis()
    manifest = build_manifest(primary, array)

    args.output_json.parent.mkdir(parents=True, exist_ok=True)
    args.output_json.write_text(json.dumps(manifest, indent=2))
    args.output_md.write_text(render_markdown(manifest))

    log.info("=" * 70)
    log.info("REMAP MANIFEST GENERATED")
    log.info("=" * 70)
    log.info("  Markdown: %s", args.output_md)
    log.info("  JSON:     %s", args.output_json)
    log.info("  Confidence breakdown: %s", manifest["confidence_summary"])
    return 0


if __name__ == "__main__":
    sys.exit(main())
