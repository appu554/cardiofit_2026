"""
Range Integrity Engine (RIE): Post-merge numeric validation.

Runs between Signal Merger output and CoverageGuard. Performs four checks
on eGFR/CrCl threshold spans extracted from clinical guidelines:

1. **Interval Canonicalization**: Normalize "eGFR < 30" + "eGFR 30-45" into
   canonical ``[lower, upper)`` intervals per drug.
2. **Continuity Checking**: Detect gaps or overlaps in the interval set.
3. **Monotonic Severity**: Verify that clinical severity increases as eGFR
   decreases (NO_CHANGE < MONITOR < REDUCE_DOSE < CONTRAINDICATED).
4. **Cross-System Threshold Consistency** (optional): Compare pipeline-extracted
   thresholds against P2's stratum_activation_rules YAML.

Produces a WARNING-only ``RangeIntegrityReport``. Does NOT block the pipeline —
CoverageGuard remains the sole gate.

Pipeline Position:
    Signal Merger -> RIE (THIS) -> L1 Recovery -> CoverageGuard
"""

from __future__ import annotations

import json
import re
from dataclasses import dataclass, field
from pathlib import Path
from typing import Optional


# ══════════════════════════════════════════════════════════════════════════
# Data Models
# ══════════════════════════════════════════════════════════════════════════

@dataclass
class ThresholdInterval:
    """A canonicalized numeric threshold interval for one drug/parameter."""
    drug_name: str
    parameter: str         # "eGFR" or "CrCl"
    lower: float           # inclusive lower bound (0 if unbounded)
    upper: float           # exclusive upper bound (inf if unbounded)
    operator: str          # original operator: "<", ">=", "range", etc.
    action: str            # "CONTRAINDICATED", "REDUCE_DOSE", "MONITOR", "NO_CHANGE"
    source_text: str       # original span text
    page_number: Optional[int] = None


@dataclass
class RangeIssue:
    """A single issue found by the Range Integrity Engine."""
    check: str             # "continuity", "overlap", "monotonic", "cross_system"
    severity: str          # "WARNING" or "ERROR"
    drug_name: str
    parameter: str
    description: str
    intervals: list[str] = field(default_factory=list)  # human-readable intervals


@dataclass
class ThresholdDiff:
    """Difference between pipeline-extracted and P2 threshold values."""
    parameter: str
    drug_name: str
    pipeline_value: float
    p2_value: float
    delta_pct: float
    severity: str          # "WARNING" if delta > 5%, "INFO" otherwise


@dataclass
class RangeIntegrityReport:
    """Output report from the Range Integrity Engine."""
    total_intervals: int = 0
    drugs_analyzed: int = 0
    issues: list[RangeIssue] = field(default_factory=list)
    cross_system_diffs: list[ThresholdDiff] = field(default_factory=list)
    elapsed_ms: float = 0.0

    @property
    def total_warnings(self) -> int:
        return sum(1 for i in self.issues if i.severity == "WARNING")

    @property
    def total_errors(self) -> int:
        return sum(1 for i in self.issues if i.severity == "ERROR")

    @property
    def has_issues(self) -> bool:
        return len(self.issues) > 0


# ══════════════════════════════════════════════════════════════════════════
# Threshold Regex Patterns
# ══════════════════════════════════════════════════════════════════════════

# Matches patterns like: "eGFR < 30", "eGFR >= 45", "CrCl 30-45", "eGFR 15 to 30"
# Also handles HTML entities (&gt;, &lt;, &ge;, &le;) found in 4 confirmed golden
# dataset spans, and en-dash ranges (25–59) found in 6 confirmed spans.
_THRESHOLD_RE = re.compile(
    r"(?P<param>eGFR|CrCl|GFR)"
    r"\s*"
    r"(?:"
    r"(?P<op>[<>]=?|≥|≤|&gt;=?|&lt;=?|&ge;|&le;)\s*(?P<val>\d+(?:\.\d+)?)"  # eGFR < 30, eGFR &gt; 45
    r"|"
    r"(?P<lo>\d+(?:\.\d+)?)\s*[-–—]\s*(?P<hi>\d+(?:\.\d+)?)"  # eGFR 30-45 (hyphen, en-dash, em-dash)
    r"|"
    r"(?P<lo2>\d+(?:\.\d+)?)\s+to\s+(?P<hi2>\d+(?:\.\d+)?)"   # eGFR 15 to 30
    r")",
    re.IGNORECASE,
)

# ── Severity Keywords (loadable from JSON) ──────────────────────────────
# Defaults are used when no JSON file is provided.  The JSON file can be
# regenerated from golden dataset analysis using scripts/derive_severity_keywords.py.

_DEFAULT_SEVERITY_KEYWORDS: dict[str, str] = {
    "contraindicated": "CONTRAINDICATED",
    "contraindication": "CONTRAINDICATED",
    "do not use": "CONTRAINDICATED",
    "avoid": "CONTRAINDICATED",
    "not recommended": "CONTRAINDICATED",
    "reduce dose": "REDUCE_DOSE",
    "dose reduction": "REDUCE_DOSE",
    "dose adjust": "REDUCE_DOSE",
    "reduce": "REDUCE_DOSE",
    "half dose": "REDUCE_DOSE",
    "monitor": "MONITOR",
    "monitoring": "MONITOR",
    "check": "MONITOR",
    "no adjustment": "NO_CHANGE",
    "no change": "NO_CHANGE",
    "standard dose": "NO_CHANGE",
    "initiate": "NO_CHANGE",
    "continue": "NO_CHANGE",
}

_DEFAULT_SEVERITY_ORDER: list[str] = [
    "NO_CHANGE", "MONITOR", "REDUCE_DOSE", "CONTRAINDICATED", "UNKNOWN",
]

# Default JSON path (sibling data/ directory)
_DEFAULT_KEYWORDS_PATH = Path(__file__).parent / "data" / "severity_keywords.json"


def _load_severity_keywords(
    path: Optional[str | Path] = None,
) -> tuple[dict[str, str], list[str]]:
    """Load severity keywords from JSON file, falling back to defaults.

    Args:
        path: Path to severity_keywords.json.  If None, tries the default
              path (extraction/v4/data/severity_keywords.json).  If the file
              does not exist, returns the hardcoded defaults.

    Returns:
        (keywords_dict, severity_order_list)
    """
    target = Path(path) if path else _DEFAULT_KEYWORDS_PATH
    if target.exists():
        try:
            with open(target) as f:
                data = json.load(f)
            keywords = data.get("keywords", _DEFAULT_SEVERITY_KEYWORDS)
            order = data.get("severity_order", _DEFAULT_SEVERITY_ORDER)
            return keywords, order
        except (json.JSONDecodeError, KeyError):
            pass  # fall through to defaults
    return _DEFAULT_SEVERITY_KEYWORDS, _DEFAULT_SEVERITY_ORDER


# Module-level defaults (loaded once at import time from JSON if available)
_SEVERITY_KEYWORDS, _SEVERITY_ORDER = _load_severity_keywords()


# ══════════════════════════════════════════════════════════════════════════
# Range Integrity Engine
# ══════════════════════════════════════════════════════════════════════════

class RangeIntegrityEngine:
    """Post-merge numeric validation for clinical threshold spans."""

    def __init__(self, severity_keywords_path: Optional[str | Path] = None) -> None:
        """Initialize with optional custom severity keywords.

        Args:
            severity_keywords_path: Path to a JSON file with severity keywords.
                If None, uses the module-level defaults (loaded from
                data/severity_keywords.json or hardcoded fallback).
                Provided by GuidelineProfile.severity_keywords_path.
        """
        if severity_keywords_path:
            self._severity_keywords, self._severity_order = _load_severity_keywords(
                severity_keywords_path
            )
        else:
            self._severity_keywords = _SEVERITY_KEYWORDS
            self._severity_order = _SEVERITY_ORDER

    def validate(
        self,
        merged_spans: list,
        normalized_text: str,
        p2_thresholds: Optional[dict] = None,
    ) -> RangeIntegrityReport:
        """Run all four RIE checks on merged spans.

        Args:
            merged_spans: List of MergedSpan objects from Signal Merger.
            normalized_text: Full normalized text for context window extraction.
            p2_thresholds: Optional dict of P2 stratum thresholds for
                cross-system consistency check. Format:
                ``{"metformin": {"eGFR": 30}, "dapagliflozin": {"eGFR": 25}}``.

        Returns:
            RangeIntegrityReport with all issues found.
        """
        import time
        start_time = time.monotonic()

        # Step 1: Extract threshold intervals from spans
        intervals = self._extract_intervals(merged_spans, normalized_text)

        # Group by drug
        drug_intervals: dict[str, list[ThresholdInterval]] = {}
        for iv in intervals:
            drug_intervals.setdefault(iv.drug_name, []).append(iv)

        issues: list[RangeIssue] = []

        for drug_name, ivs in drug_intervals.items():
            # Sort by lower bound
            ivs.sort(key=lambda x: x.lower)

            # Step 2: Continuity check
            issues.extend(self._check_continuity(drug_name, ivs))

            # Step 3: Monotonic severity check
            issues.extend(self._check_monotonic_severity(drug_name, ivs))

        # Step 4: Cross-system threshold consistency
        cross_diffs: list[ThresholdDiff] = []
        if p2_thresholds:
            cross_diffs = self._check_cross_system(intervals, p2_thresholds)
            for diff in cross_diffs:
                if diff.severity == "WARNING":
                    issues.append(RangeIssue(
                        check="cross_system",
                        severity="WARNING",
                        drug_name=diff.drug_name,
                        parameter=diff.parameter,
                        description=(
                            f"Pipeline threshold ({diff.pipeline_value}) differs from "
                            f"P2 threshold ({diff.p2_value}) by {diff.delta_pct:.1f}%"
                        ),
                    ))

        elapsed = (time.monotonic() - start_time) * 1000

        return RangeIntegrityReport(
            total_intervals=len(intervals),
            drugs_analyzed=len(drug_intervals),
            issues=issues,
            cross_system_diffs=cross_diffs,
            elapsed_ms=elapsed,
        )

    # ── Step 1: Interval Extraction ──────────────────────────────────────

    def _extract_intervals(
        self,
        merged_spans: list,
        normalized_text: str,
    ) -> list[ThresholdInterval]:
        """Extract ThresholdInterval objects from merged spans."""
        intervals: list[ThresholdInterval] = []

        for span in merged_spans:
            text = span.text
            # Try matching threshold patterns in the span text
            for m in _THRESHOLD_RE.finditer(text):
                interval = self._match_to_interval(m, span, normalized_text)
                if interval:
                    intervals.append(interval)

        return intervals

    def _match_to_interval(
        self,
        match: re.Match,
        span,
        normalized_text: str,
    ) -> Optional[ThresholdInterval]:
        """Convert a regex match to a ThresholdInterval."""
        param = match.group("param")

        # Determine bounds from match groups
        if match.group("op"):
            # Single operator: eGFR < 30, eGFR >= 45
            op = match.group("op")
            val = float(match.group("val"))
            lower, upper = self._operator_to_bounds(op, val)
        elif match.group("lo"):
            # Range: eGFR 30-45
            op = "range"
            lower = float(match.group("lo"))
            upper = float(match.group("hi"))
        elif match.group("lo2"):
            # Range with "to": eGFR 15 to 30
            op = "range"
            lower = float(match.group("lo2"))
            upper = float(match.group("hi2"))
        else:
            return None

        # Determine drug name from surrounding context
        drug_name = self._infer_drug_name(span, normalized_text)
        if not drug_name:
            drug_name = "UNKNOWN"

        # Determine action/severity from surrounding context
        action = self._infer_action(span, normalized_text)

        return ThresholdInterval(
            drug_name=drug_name,
            parameter=param.upper().replace("GFR", "eGFR") if param.upper() == "GFR" else param,
            lower=lower,
            upper=upper,
            operator=op,
            action=action,
            source_text=span.text,
            page_number=getattr(span, "page_number", None),
        )

    @staticmethod
    def _operator_to_bounds(op: str, val: float) -> tuple[float, float]:
        """Convert operator + value to [lower, upper) interval.

        Handles standard operators (<, >, <=, >=), Unicode (≥, ≤),
        and HTML entities (&gt;, &lt;, &ge;, &le;) found in golden dataset.
        """
        # Normalize HTML entities to standard operators
        _html_map = {
            "&gt;=": ">=", "&gt;": ">",
            "&lt;=": "<=", "&lt;": "<",
            "&ge;": ">=", "&le;": "<=",
        }
        op = _html_map.get(op, op)

        if op in ("<", ""):
            return (0, val)
        elif op in ("<=", "≤"):
            return (0, val + 0.1)  # inclusive upper approximation
        elif op == ">":
            return (val, float("inf"))
        elif op in (">=", "≥"):
            return (val, float("inf"))
        return (0, float("inf"))

    @staticmethod
    def _infer_drug_name(span, normalized_text: str) -> Optional[str]:
        """Infer drug name from span metadata or surrounding text."""
        # Check channel_metadata for drug info
        meta = getattr(span, "channel_metadata", {}) or {}
        if meta:
            # Channel B stores rxnorm_candidate
            if "rxnorm_candidate" in meta:
                return span.text.split()[0].lower() if span.text else None

        # Check surrounding text (50 chars before span)
        start = max(0, span.start - 80)
        context = normalized_text[start:span.start + len(span.text)].lower()

        # Common drug names in renal guidelines
        _DRUG_PATTERNS = [
            "metformin", "dapagliflozin", "empagliflozin", "canagliflozin",
            "ertugliflozin", "finerenone", "spironolactone", "lisinopril",
            "enalapril", "losartan", "valsartan",
        ]
        for drug in _DRUG_PATTERNS:
            if drug in context:
                return drug

        return None

    def _infer_action(self, span, normalized_text: str) -> str:
        """Infer the clinical action from surrounding text."""
        start = max(0, span.start - 50)
        end = min(len(normalized_text), span.end + 50)
        context = normalized_text[start:end].lower()

        for keyword, action in self._severity_keywords.items():
            if keyword in context:
                return action

        return "UNKNOWN"

    # ── Step 2: Continuity Check ─────────────────────────────────────────

    def _check_continuity(
        self,
        drug_name: str,
        intervals: list[ThresholdInterval],
    ) -> list[RangeIssue]:
        """Check for gaps and overlaps in sorted intervals for one drug."""
        issues: list[RangeIssue] = []

        if len(intervals) < 2:
            return issues

        for i in range(len(intervals) - 1):
            curr = intervals[i]
            next_iv = intervals[i + 1]

            # Skip if different parameters
            if curr.parameter != next_iv.parameter:
                continue

            # Gap detection: curr.upper < next.lower
            if curr.upper < next_iv.lower:
                gap_size = next_iv.lower - curr.upper
                issues.append(RangeIssue(
                    check="continuity",
                    severity="WARNING",
                    drug_name=drug_name,
                    parameter=curr.parameter,
                    description=(
                        f"Gap of {gap_size} between intervals: "
                        f"[{curr.lower}, {curr.upper}) and [{next_iv.lower}, {next_iv.upper})"
                    ),
                    intervals=[
                        f"[{curr.lower}, {curr.upper})",
                        f"[{next_iv.lower}, {next_iv.upper})",
                    ],
                ))

            # Overlap detection: curr.upper > next.lower (with tolerance)
            elif curr.upper > next_iv.lower + 0.5:
                overlap = curr.upper - next_iv.lower
                issues.append(RangeIssue(
                    check="overlap",
                    severity="WARNING",
                    drug_name=drug_name,
                    parameter=curr.parameter,
                    description=(
                        f"Overlap of {overlap} between intervals: "
                        f"[{curr.lower}, {curr.upper}) and [{next_iv.lower}, {next_iv.upper})"
                    ),
                    intervals=[
                        f"[{curr.lower}, {curr.upper})",
                        f"[{next_iv.lower}, {next_iv.upper})",
                    ],
                ))

        return issues

    # ── Step 3: Monotonic Severity ───────────────────────────────────────

    def _check_monotonic_severity(
        self,
        drug_name: str,
        intervals: list[ThresholdInterval],
    ) -> list[RangeIssue]:
        """Verify severity increases as eGFR decreases (lower bound decreases)."""
        issues: list[RangeIssue] = []

        if len(intervals) < 2:
            return issues

        # Filter to same parameter and known actions
        for param in ("eGFR", "CrCl"):
            param_ivs = [
                iv for iv in intervals
                if iv.parameter == param and iv.action != "UNKNOWN"
            ]
            if len(param_ivs) < 2:
                continue

            # Sort by lower bound descending (highest eGFR first)
            param_ivs.sort(key=lambda x: -x.lower)

            for i in range(len(param_ivs) - 1):
                higher_gfr = param_ivs[i]
                lower_gfr = param_ivs[i + 1]

                higher_sev = self._severity_order.index(higher_gfr.action)
                lower_sev = self._severity_order.index(lower_gfr.action)

                # Severity should increase (or stay same) as eGFR decreases
                if lower_sev < higher_sev:
                    issues.append(RangeIssue(
                        check="monotonic",
                        severity="ERROR",
                        drug_name=drug_name,
                        parameter=param,
                        description=(
                            f"Non-monotonic severity: {param} [{higher_gfr.lower}, {higher_gfr.upper}) "
                            f"→ {higher_gfr.action} but [{lower_gfr.lower}, {lower_gfr.upper}) "
                            f"→ {lower_gfr.action} (should be same or more severe)"
                        ),
                    ))

        return issues

    # ── Step 4: Cross-System Consistency ─────────────────────────────────

    def _check_cross_system(
        self,
        intervals: list[ThresholdInterval],
        p2_thresholds: dict,
    ) -> list[ThresholdDiff]:
        """Compare pipeline thresholds against P2 stratum activation rules."""
        diffs: list[ThresholdDiff] = []

        for iv in intervals:
            drug = iv.drug_name.lower()
            if drug not in p2_thresholds:
                continue

            drug_thresholds = p2_thresholds[drug]
            param = iv.parameter
            if param not in drug_thresholds:
                continue

            p2_val = float(drug_thresholds[param])

            # Compare the interval boundary closest to the P2 threshold
            # Use lower bound for "< X" patterns, upper bound for ">= X"
            if iv.operator in ("<", "<=", "≤"):
                pipeline_val = iv.upper
            elif iv.operator in (">=", ">", "≥"):
                pipeline_val = iv.lower
            else:
                pipeline_val = iv.lower  # range: use lower

            if p2_val == 0:
                continue

            delta_pct = abs(pipeline_val - p2_val) / p2_val * 100

            diffs.append(ThresholdDiff(
                parameter=param,
                drug_name=iv.drug_name,
                pipeline_value=pipeline_val,
                p2_value=p2_val,
                delta_pct=delta_pct,
                severity="WARNING" if delta_pct > 5 else "INFO",
            ))

        return diffs
