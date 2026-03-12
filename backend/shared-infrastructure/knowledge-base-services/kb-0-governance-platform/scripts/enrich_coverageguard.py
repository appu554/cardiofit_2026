#!/usr/bin/env python3
"""
Enrich l2_merged_spans with CoverageGuard data from coverage_guard_report.json.

Populates three columns that were added in migration 006 but never loaded:
  1. tier              — SMALLINT (1 or 2) from corroboration_details[].tier
  2. coverage_guard_alert — JSONB alert payload assembled from CoverageGuard findings
  3. semantic_tokens    — JSONB {numerics, conditions, negations} extracted from span text

Usage:
    # Dry run — show what would change without writing
    python enrich_coverageguard.py <data_dir> --dry-run

    # Apply to production
    python enrich_coverageguard.py <data_dir> --db-url postgresql://kb_admin:kb_secure_password_2024@localhost:5432/canonical_facts

    # Apply via SSH tunnel (GCP)
    python enrich_coverageguard.py <data_dir> --db-url postgresql://kb_admin:kb_secure_password_2024@34.46.243.149:5432/canonical_facts
"""

import argparse
import json
import re
import sys
from pathlib import Path
from collections import defaultdict
from typing import Optional

try:
    import psycopg2
    from psycopg2.extras import execute_values, Json
except ImportError:
    print("Installing psycopg2-binary...")
    import os
    os.system(f"{sys.executable} -m pip install psycopg2-binary")
    import psycopg2
    from psycopg2.extras import execute_values, Json


# =============================================================================
# TIER MAPPING
# =============================================================================

TIER_MAP = {
    "TIER_1": 1,
    "TIER_2": 2,
    "TIER_3": 3,
}


# =============================================================================
# SEMANTIC TOKEN EXTRACTION
# =============================================================================

# Numeric patterns: clinical values like ≥30, <6.5%, eGFR 45, HbA1c 7.0%
NUMERIC_RE = re.compile(
    r'(?:'
    r'[≥≤><]=?\s*\d+[\d.,%]*'        # ≥30, <6.5%, >=45
    r'|\d+[\d.]*\s*(?:%|mg|ml|mmol|g/d[lL]|mL/min|kg|mm\s*Hg|µg|ng)'  # 30%, 1.73 m², 140 mm Hg
    r'|\d+\.?\d*\s*(?:mg/d[lL]|mmol/L|mEq/L|U/L|IU/L)'               # 5.0 mg/dL
    r'|\d+\.?\d*\s*(?:months?|years?|weeks?|days?|hours?)'              # 6 months
    r'|\beGFR\s*[<>≥≤]?\s*\d+'        # eGFR ≥20, eGFR<45
    r'|\bHbA1c\s*[<>≥≤]?\s*\d+[.\d]*' # HbA1c <7.0
    r'|\bBMI\s*[<>≥≤]?\s*\d+'         # BMI ≥30
    r'|\bSBP\s*[<>≥≤]?\s*\d+'         # SBP <130
    r'|\bDBP\s*[<>≥≤]?\s*\d+'         # DBP <80
    r'|\b\d+\.?\d*\s*/\s*\d+\.?\d*\s*m[²2]'  # 1.73 m²
    r')',
    re.IGNORECASE
)

# Condition patterns: clinical branching logic
CONDITION_WORDS = [
    # Multi-word patterns first (longest match)
    "provided that", "as long as", "in the setting of", "in the context of",
    "in patients with", "in people with", "for patients with",
    "should be considered", "may be considered", "is recommended",
    "is suggested", "should not", "should be",
    # Single-word patterns
    "if", "when", "unless", "while", "whereas", "although",
    "provided", "given", "during", "after", "before", "until",
    "whether", "where",
]

# Negation patterns: clinical safety-critical negations
NEGATION_PATTERNS = [
    # Multi-word negation phrases (longest first)
    "not recommended", "not suggested", "not be used", "not treated with",
    "not be initiated", "should not", "do not", "does not",
    "is not", "are not", "was not", "were not",
    "no longer", "no evidence", "no benefit",
    "contraindicated", "avoid", "discontinue", "withhold",
    "not", "no",  # single words last
]


def extract_semantic_tokens(text: str) -> dict:
    """Extract semantic tokens from span text for UI highlighting."""
    if not text or len(text) < 3:
        return {"numerics": [], "conditions": [], "negations": []}

    # 1. Numerics — regex-based extraction
    numerics = []
    seen_nums = set()
    for m in NUMERIC_RE.finditer(text):
        val = m.group().strip()
        if val not in seen_nums:
            seen_nums.add(val)
            numerics.append(val)

    # 2. Conditions — exact substring match (case-insensitive)
    conditions = []
    text_lower = text.lower()
    seen_conds = set()
    for pattern in CONDITION_WORDS:
        if pattern.lower() in text_lower and pattern.lower() not in seen_conds:
            # Find the exact case from the original text
            idx = text_lower.find(pattern.lower())
            if idx >= 0:
                original = text[idx:idx + len(pattern)]
                seen_conds.add(pattern.lower())
                conditions.append(original)

    # 3. Negations — exact substring match (case-insensitive, longest first)
    negations = []
    seen_neg_ranges = []  # Track character ranges to avoid overlaps
    for pattern in NEGATION_PATTERNS:
        start = 0
        pat_lower = pattern.lower()
        while True:
            idx = text_lower.find(pat_lower, start)
            if idx < 0:
                break
            end = idx + len(pattern)
            # Check overlap with existing negation ranges
            overlaps = any(
                not (end <= r_start or idx >= r_end)
                for r_start, r_end in seen_neg_ranges
            )
            if not overlaps:
                original = text[idx:end]
                negations.append(original)
                seen_neg_ranges.append((idx, end))
            start = end

    return {
        "numerics": numerics[:15],    # Cap at 15 to avoid huge payloads
        "conditions": conditions[:10],
        "negations": negations[:10],
    }


# =============================================================================
# ALERT ASSEMBLY
# =============================================================================

def build_coverage_guard_alert(
    span_id: str,
    corr: dict,
    l1_recovery_ids: set,
    residual_lookup: dict,
    branch_lookup: dict,
) -> Optional[dict]:
    """
    Build a CoverageGuardAlert JSONB for a span based on CoverageGuard findings.

    Alert types (from types/pipeline1.ts):
      - numeric_mismatch: numeric value differs from source
      - branch_loss: threshold/branch logic incomplete
      - llm_only: single-channel LLM extraction (low corroboration)
      - negation_flip: negation changed or lost

    Returns None if no alert is warranted.
    """
    tier_str = corr.get("tier", "TIER_2")
    action = corr.get("action", "PASS")
    score = corr.get("corroboration_score", 1.0)
    channels = corr.get("contributing_channels", [])

    # Priority 1: L1_RECOVERY spans with BLOCK action
    if span_id in l1_recovery_ids and action == "BLOCK":
        return {
            "type": "llm_only",
            "label": "L1 Recovery — Low Corroboration",
            "detail": f"This span was recovered from OCR (L1_RECOVERY) with corroboration score {score:.1f}. "
                      f"CoverageGuard verdict: BLOCK. Verify against source PDF.",
            "alertSeverity": "critical",
        }

    # Priority 2: Single-channel TIER_1 with low corroboration
    if tier_str == "TIER_1" and len(channels) == 1 and score < 0.5:
        channel = channels[0]
        return {
            "type": "llm_only",
            "label": f"Single-Channel Extraction ({channel})",
            "detail": f"Only channel {channel} extracted this text (score {score:.2f}). "
                      f"No corroboration from other extraction methods.",
            "alertSeverity": "warning",
        }

    # Priority 3: Branch comparison issues (by section)
    # branch_lookup maps section_id → branch comparison data
    # We can't directly link span to branch, but if span's section has issues, flag it
    # (This is a best-effort enrichment — full span-level branch analysis would need
    #  the pipeline to produce per-span branch alerts)

    # Priority 4: TIER_1 spans with action=BLOCK (non-L1_RECOVERY)
    if action == "BLOCK" and span_id not in l1_recovery_ids:
        return {
            "type": "llm_only",
            "label": "CoverageGuard BLOCK",
            "detail": f"CoverageGuard flagged this span for review (score {score:.2f}, "
                      f"channels: {', '.join(channels)}). Gate verdict: BLOCK.",
            "alertSeverity": "critical",
        }

    # No alert for well-corroborated spans
    return None


# =============================================================================
# MAIN ENRICHMENT LOGIC
# =============================================================================

def load_json(path: Path):
    with open(path, "r", encoding="utf-8") as f:
        return json.load(f)


def enrich(data_dir: Path, db_url: str, dry_run: bool = False):
    """Read CoverageGuard report and enrich l2_merged_spans."""

    # ── Load pipeline outputs ─────────────────────────────────────────────
    cg_path = data_dir / "coverage_guard_report.json"
    spans_path = data_dir / "merged_spans.json"
    meta_path = data_dir / "job_metadata.json"

    if not cg_path.exists():
        print(f"ERROR: {cg_path} not found")
        sys.exit(1)

    metadata = load_json(meta_path)
    job_id = metadata["job_id"]
    print(f"Job: {job_id}")
    print(f"Source: {metadata.get('source_pdf', 'unknown')}")

    cg_report = load_json(cg_path)
    spans = load_json(spans_path) if spans_path.exists() else []

    # Build span text lookup for semantic token extraction
    span_texts = {s["id"]: s["text"] for s in spans}
    print(f"Loaded {len(span_texts)} span texts")

    # ── Parse CoverageGuard report ────────────────────────────────────────
    corr_details = cg_report.get("corroboration_details", [])
    l1_recovery_ids = set(cg_report.get("l1_recovery_escalations", []))
    residual_fragments = cg_report.get("residual_fragments", [])
    branch_comparisons = cg_report.get("branch_comparisons", [])

    print(f"Corroboration details: {len(corr_details)} items")
    print(f"L1 recovery escalations: {len(l1_recovery_ids)} span IDs")
    print(f"Residual fragments: {len(residual_fragments)}")
    print(f"Branch comparisons: {len(branch_comparisons)}")

    # Build lookup by span_id
    corr_by_span = {c["span_id"]: c for c in corr_details}

    # Build branch lookup by section_id
    branch_by_section = defaultdict(list)
    for bc in branch_comparisons:
        branch_by_section[bc.get("section_id", "")].append(bc)

    # Build residual lookup — residuals don't have span_ids, skip for now
    residual_lookup = {}

    # ── Compute enrichment for each span ──────────────────────────────────
    enrichments = []  # (span_id, tier, alert_json, tokens_json)

    tier_counts = defaultdict(int)
    alert_counts = defaultdict(int)
    token_counts = {"with_numerics": 0, "with_conditions": 0, "with_negations": 0}

    for span_id, text in span_texts.items():
        corr = corr_by_span.get(span_id)

        # 1. Tier
        tier = None
        if corr:
            tier = TIER_MAP.get(corr.get("tier"), None)
            if tier:
                tier_counts[tier] += 1

        # 2. Coverage Guard Alert
        alert = None
        if corr:
            alert = build_coverage_guard_alert(
                span_id, corr, l1_recovery_ids,
                residual_lookup, branch_by_section,
            )
            if alert:
                alert_counts[alert["type"]] += 1

        # 3. Semantic Tokens
        tokens = extract_semantic_tokens(text)
        if tokens["numerics"]:
            token_counts["with_numerics"] += 1
        if tokens["conditions"]:
            token_counts["with_conditions"] += 1
        if tokens["negations"]:
            token_counts["with_negations"] += 1

        # Only include if we have at least one non-empty field
        has_tokens = bool(tokens["numerics"] or tokens["conditions"] or tokens["negations"])

        enrichments.append((
            span_id,
            tier,
            json.dumps(alert) if alert else None,
            json.dumps(tokens) if has_tokens else None,
        ))

    # ── Report ────────────────────────────────────────────────────────────
    print(f"\n{'='*60}")
    print(f"ENRICHMENT SUMMARY for job {job_id}")
    print(f"{'='*60}")
    print(f"Total spans: {len(enrichments)}")
    print(f"\nTier distribution:")
    for tier, count in sorted(tier_counts.items()):
        print(f"  Tier {tier}: {count} spans")
    untiered = sum(1 for e in enrichments if e[1] is None)
    print(f"  No tier: {untiered} spans (not in corroboration_details)")

    print(f"\nCoverageGuard alerts:")
    for alert_type, count in sorted(alert_counts.items()):
        print(f"  {alert_type}: {count} spans")
    no_alert = sum(1 for e in enrichments if e[2] is None)
    print(f"  No alert: {no_alert} spans (well-corroborated)")

    print(f"\nSemantic tokens:")
    print(f"  With numerics:   {token_counts['with_numerics']} spans")
    print(f"  With conditions: {token_counts['with_conditions']} spans")
    print(f"  With negations:  {token_counts['with_negations']} spans")
    no_tokens = sum(1 for e in enrichments if e[3] is None)
    print(f"  No tokens: {no_tokens} spans")

    if dry_run:
        print(f"\n--- DRY RUN: No database changes made ---")

        # Show a few examples
        print(f"\nSample enrichments (first 5 with alerts):")
        shown = 0
        for span_id, tier, alert_json, tokens_json in enrichments:
            if alert_json and shown < 5:
                alert = json.loads(alert_json)
                tokens = json.loads(tokens_json) if tokens_json else {}
                print(f"\n  Span: {span_id}")
                print(f"  Tier: {tier}")
                print(f"  Alert: {alert['type']} ({alert['alertSeverity']})")
                print(f"  Detail: {alert['detail'][:100]}...")
                print(f"  Tokens: {len(tokens.get('numerics', []))} numerics, "
                      f"{len(tokens.get('conditions', []))} conditions, "
                      f"{len(tokens.get('negations', []))} negations")
                shown += 1

        print(f"\nSample semantic tokens (first 3 with rich tokens):")
        shown = 0
        for span_id, tier, alert_json, tokens_json in enrichments:
            if tokens_json and shown < 3:
                tokens = json.loads(tokens_json)
                if len(tokens.get("numerics", [])) >= 2:
                    text = span_texts.get(span_id, "")[:80]
                    print(f"\n  Span: {span_id}")
                    print(f"  Text: {text}...")
                    print(f"  Numerics: {tokens['numerics'][:5]}")
                    print(f"  Conditions: {tokens['conditions'][:3]}")
                    print(f"  Negations: {tokens['negations'][:3]}")
                    shown += 1
        return

    # ── Apply to database ──────────────────────────────────────────────────
    print(f"\nConnecting to: {db_url.split('@')[1] if '@' in db_url else db_url}")
    conn = psycopg2.connect(db_url)
    conn.autocommit = False

    try:
        cur = conn.cursor()

        # Verify job exists
        cur.execute("SELECT COUNT(*) FROM l2_merged_spans WHERE job_id = %s", (job_id,))
        db_count = cur.fetchone()[0]
        print(f"DB has {db_count} spans for this job")

        if db_count == 0:
            print("ERROR: No spans found in DB for this job. Run the loader first.")
            sys.exit(1)

        # Batch UPDATE using a temp table approach for speed
        print("Creating temp table for batch update...")
        cur.execute("""
            CREATE TEMP TABLE _enrichment (
                span_id UUID PRIMARY KEY,
                tier SMALLINT,
                coverage_guard_alert JSONB,
                semantic_tokens JSONB
            ) ON COMMIT DROP
        """)

        # Insert enrichments into temp table
        values = [
            (e[0], e[1], e[2], e[3])
            for e in enrichments
        ]
        execute_values(cur, """
            INSERT INTO _enrichment (span_id, tier, coverage_guard_alert, semantic_tokens)
            VALUES %s
        """, values, template="(%s::uuid, %s, %s::jsonb, %s::jsonb)")
        print(f"  Loaded {len(values)} enrichment rows into temp table")

        # Single UPDATE join
        cur.execute("""
            UPDATE l2_merged_spans s
            SET
                tier = e.tier,
                coverage_guard_alert = e.coverage_guard_alert,
                semantic_tokens = e.semantic_tokens
            FROM _enrichment e
            WHERE s.id = e.span_id
              AND s.job_id = %s
        """, (job_id,))

        updated = cur.rowcount
        print(f"  Updated {updated} spans")

        conn.commit()
        print(f"\n=== ENRICHMENT COMPLETE ===")

        # Verify
        cur.execute("""
            SELECT
                COUNT(*) FILTER (WHERE tier IS NOT NULL) as with_tier,
                COUNT(*) FILTER (WHERE coverage_guard_alert IS NOT NULL) as with_alert,
                COUNT(*) FILTER (WHERE semantic_tokens IS NOT NULL) as with_tokens,
                COUNT(*) as total
            FROM l2_merged_spans
            WHERE job_id = %s
        """, (job_id,))
        row = cur.fetchone()
        print(f"\nVerification:")
        print(f"  tier populated:    {row[0]}/{row[3]}")
        print(f"  alert populated:   {row[1]}/{row[3]}")
        print(f"  tokens populated:  {row[2]}/{row[3]}")

    except Exception as e:
        conn.rollback()
        print(f"\nERROR: {e}")
        raise
    finally:
        conn.close()


# =============================================================================
# CLI
# =============================================================================

def main():
    parser = argparse.ArgumentParser(
        description="Enrich l2_merged_spans with CoverageGuard data"
    )
    parser.add_argument(
        "data_dir",
        type=Path,
        help="Path to pipeline output directory containing coverage_guard_report.json",
    )
    parser.add_argument(
        "--db-url",
        default="postgresql://kb_admin:kb_secure_password_2024@34.46.243.149:5432/canonical_facts",
        help="PostgreSQL connection URL",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Show enrichment plan without writing to database",
    )
    args = parser.parse_args()

    enrich(args.data_dir, args.db_url, args.dry_run)


if __name__ == "__main__":
    main()
