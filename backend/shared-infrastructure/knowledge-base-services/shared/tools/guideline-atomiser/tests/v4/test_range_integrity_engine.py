"""
Tests for Range Integrity Engine (Phase 4).

Validates:
1. Interval extraction from threshold patterns (eGFR <, >=, range, "to")
2. Continuity checking (gaps and overlaps detected)
3. Monotonic severity validation
4. Cross-system threshold consistency (P2 comparison)
5. Empty input → empty report
6. Report structure and convenience properties
"""

import sys
from pathlib import Path
from uuid import uuid4

import pytest

SHARED_DIR = Path(__file__).resolve().parents[4]
sys.path.insert(0, str(SHARED_DIR))

from extraction.v4.range_integrity_engine import (
    RangeIntegrityEngine,
    RangeIntegrityReport,
    RangeIssue,
    ThresholdDiff,
    ThresholdInterval,
)


# ── Helpers ────────────────────────────────────────────────────────────

class _FakeSpan:
    """Minimal span-like object for RIE testing."""
    def __init__(self, text: str, start: int = 0, page_number: int = 1):
        self.text = text
        self.start = start
        self.end = start + len(text)
        self.page_number = page_number
        self.channel_metadata = {}


def _make_spans_and_text(*span_texts: str) -> tuple[list[_FakeSpan], str]:
    """Build fake spans and a normalized text string."""
    parts = []
    spans = []
    offset = 0
    for text in span_texts:
        spans.append(_FakeSpan(text, start=offset))
        parts.append(text)
        offset += len(text) + 2  # gap between spans
        parts.append("  ")
    full_text = "".join(parts)
    return spans, full_text


# ── Report Structure ──────────────────────────────────────────────────

class TestReportStructure:
    """Verify RangeIntegrityReport properties."""

    def test_empty_report(self):
        report = RangeIntegrityReport()
        assert report.total_intervals == 0
        assert report.drugs_analyzed == 0
        assert report.total_warnings == 0
        assert report.total_errors == 0
        assert not report.has_issues

    def test_report_with_issues(self):
        issue = RangeIssue(
            check="continuity", severity="WARNING",
            drug_name="metformin", parameter="eGFR",
            description="test gap",
        )
        report = RangeIntegrityReport(
            total_intervals=3, drugs_analyzed=1,
            issues=[issue],
        )
        assert report.total_warnings == 1
        assert report.total_errors == 0
        assert report.has_issues

    def test_report_error_count(self):
        issues = [
            RangeIssue(check="monotonic", severity="ERROR",
                       drug_name="metformin", parameter="eGFR",
                       description="non-monotonic"),
            RangeIssue(check="continuity", severity="WARNING",
                       drug_name="metformin", parameter="eGFR",
                       description="gap"),
        ]
        report = RangeIntegrityReport(issues=issues)
        assert report.total_warnings == 1
        assert report.total_errors == 1


# ── Empty Input ───────────────────────────────────────────────────────

class TestEmptyInput:
    """RIE with no spans or no thresholds."""

    def test_no_spans(self):
        rie = RangeIntegrityEngine()
        report = rie.validate([], "")
        assert report.total_intervals == 0
        assert not report.has_issues

    def test_spans_without_thresholds(self):
        spans = [_FakeSpan("metformin 500 mg daily")]
        text = "metformin 500 mg daily"
        rie = RangeIntegrityEngine()
        report = rie.validate(spans, text)
        assert report.total_intervals == 0


# ── Interval Extraction ───────────────────────────────────────────────

class TestIntervalExtraction:
    """Verify threshold pattern recognition."""

    def test_egfr_less_than(self):
        text = "metformin contraindicated when eGFR < 30"
        spans = [_FakeSpan(text)]
        rie = RangeIntegrityEngine()
        report = rie.validate(spans, text)
        assert report.total_intervals == 1

    def test_egfr_greater_equal(self):
        text = "initiate dapagliflozin when eGFR >= 25"
        spans = [_FakeSpan(text)]
        rie = RangeIntegrityEngine()
        report = rie.validate(spans, text)
        assert report.total_intervals == 1

    def test_egfr_range_dash(self):
        text = "reduce metformin dose eGFR 30-45"
        spans = [_FakeSpan(text)]
        rie = RangeIntegrityEngine()
        report = rie.validate(spans, text)
        assert report.total_intervals == 1

    def test_egfr_range_to(self):
        text = "monitor finerenone eGFR 15 to 30"
        spans = [_FakeSpan(text)]
        rie = RangeIntegrityEngine()
        report = rie.validate(spans, text)
        assert report.total_intervals == 1

    def test_crcl_pattern(self):
        text = "CrCl < 30 mL/min contraindicated"
        spans = [_FakeSpan(text)]
        rie = RangeIntegrityEngine()
        report = rie.validate(spans, text)
        assert report.total_intervals == 1

    def test_multiple_thresholds(self):
        text = "metformin: eGFR < 30 contraindicated, eGFR 30-45 reduce dose"
        spans = [_FakeSpan(text)]
        rie = RangeIntegrityEngine()
        report = rie.validate(spans, text)
        assert report.total_intervals == 2


# ── Continuity Check ──────────────────────────────────────────────────

class TestContinuityCheck:
    """Verify gap and overlap detection."""

    def test_contiguous_no_issues(self):
        """Contiguous intervals [0,30) [30,45) [45,inf) → no issues."""
        text = (
            "metformin: avoid when eGFR < 30. "
            "metformin: reduce dose eGFR 30-45. "
            "metformin: standard dose eGFR >= 45."
        )
        spans = [_FakeSpan(text)]
        rie = RangeIntegrityEngine()
        report = rie.validate(spans, text)
        # The RIE should extract 3 intervals and find them contiguous
        # (with tolerance for boundary precision)
        continuity_issues = [i for i in report.issues if i.check == "continuity"]
        # Contiguous → no gaps
        assert len(continuity_issues) == 0

    def test_gap_detected(self):
        """Gap between [0,25) and [35,inf) → warning."""
        text = "metformin: avoid when eGFR < 25. metformin: standard dose eGFR >= 35."
        spans = [_FakeSpan(text)]
        rie = RangeIntegrityEngine()
        report = rie.validate(spans, text)
        gap_issues = [i for i in report.issues if i.check == "continuity"]
        assert len(gap_issues) >= 1
        assert gap_issues[0].severity == "WARNING"
        assert "gap" in gap_issues[0].description.lower()

    def test_overlap_detected(self):
        """Overlapping [0,35) and [25,inf) → warning."""
        # eGFR < 35 means [0,35) and eGFR >= 25 means [25,inf)
        text = "metformin: avoid when eGFR < 35. metformin: continue eGFR >= 25."
        spans = [_FakeSpan(text)]
        rie = RangeIntegrityEngine()
        report = rie.validate(spans, text)
        overlap_issues = [i for i in report.issues if i.check == "overlap"]
        assert len(overlap_issues) >= 1
        assert "overlap" in overlap_issues[0].description.lower()


# ── Monotonic Severity ────────────────────────────────────────────────

class TestMonotonicSeverity:
    """Verify severity ordering validation."""

    def test_correct_monotonic_order(self):
        """Higher eGFR → less severe action → no issue."""
        text = (
            "metformin: eGFR >= 45 continue. "
            "metformin: eGFR 30-45 reduce dose. "
            "metformin: eGFR < 30 avoid."
        )
        spans = [_FakeSpan(text)]
        rie = RangeIntegrityEngine()
        report = rie.validate(spans, text)
        monotonic_issues = [i for i in report.issues if i.check == "monotonic"]
        assert len(monotonic_issues) == 0

    def test_non_monotonic_severity_flagged(self):
        """eGFR ≥ 45 → CONTRAINDICATED but eGFR < 30 → NO_CHANGE is wrong."""
        # Separate spans by >100 chars so _infer_action context windows don't overlap
        padding = " " * 120
        text = f"metformin: avoid eGFR >= 45.{padding}metformin: continue eGFR < 30."
        span1_text = "metformin: avoid eGFR >= 45"
        span2_start = len(f"metformin: avoid eGFR >= 45.{padding}")
        span2_text = "metformin: continue eGFR < 30"
        span1 = _FakeSpan(span1_text, start=0)
        span2 = _FakeSpan(span2_text, start=span2_start)
        rie = RangeIntegrityEngine()
        report = rie.validate([span1, span2], text)
        monotonic_issues = [i for i in report.issues if i.check == "monotonic"]
        assert len(monotonic_issues) >= 1
        assert monotonic_issues[0].severity == "ERROR"


# ── Cross-System Consistency ──────────────────────────────────────────

class TestCrossSystemCheck:
    """Verify P2 threshold comparison."""

    def test_matching_thresholds(self):
        """Pipeline and P2 agree → INFO (no warning)."""
        text = "metformin: avoid when eGFR < 30"
        spans = [_FakeSpan(text)]
        rie = RangeIntegrityEngine()
        p2 = {"metformin": {"eGFR": 30}}
        report = rie.validate(spans, text, p2_thresholds=p2)
        cross_issues = [i for i in report.issues if i.check == "cross_system"]
        assert len(cross_issues) == 0  # delta ≤ 5% → INFO, not WARNING

    def test_mismatched_thresholds(self):
        """Pipeline says 30, P2 says 20 → >5% delta → WARNING."""
        text = "metformin: avoid when eGFR < 30"
        spans = [_FakeSpan(text)]
        rie = RangeIntegrityEngine()
        p2 = {"metformin": {"eGFR": 20}}  # 50% delta
        report = rie.validate(spans, text, p2_thresholds=p2)
        assert len(report.cross_system_diffs) >= 1
        warning_diffs = [d for d in report.cross_system_diffs if d.severity == "WARNING"]
        assert len(warning_diffs) >= 1

    def test_no_p2_thresholds(self):
        """When p2_thresholds is None, cross-system check is skipped."""
        text = "metformin: avoid when eGFR < 30"
        spans = [_FakeSpan(text)]
        rie = RangeIntegrityEngine()
        report = rie.validate(spans, text)
        assert len(report.cross_system_diffs) == 0


# ── Operator to Bounds ────────────────────────────────────────────────

class TestOperatorBounds:
    """Test interval boundary conversion."""

    def test_less_than(self):
        lower, upper = RangeIntegrityEngine._operator_to_bounds("<", 30)
        assert lower == 0
        assert upper == 30

    def test_greater_equal(self):
        lower, upper = RangeIntegrityEngine._operator_to_bounds(">=", 45)
        assert lower == 45
        assert upper == float("inf")

    def test_less_equal(self):
        lower, upper = RangeIntegrityEngine._operator_to_bounds("<=", 30)
        assert lower == 0
        assert upper == 30.1  # inclusive approximation

    def test_unicode_greater_equal(self):
        lower, upper = RangeIntegrityEngine._operator_to_bounds("≥", 25)
        assert lower == 25
        assert upper == float("inf")

    def test_html_gt(self):
        """HTML entity &gt; normalizes to > operator."""
        lower, upper = RangeIntegrityEngine._operator_to_bounds("&gt;", 30)
        assert lower == 30
        assert upper == float("inf")

    def test_html_lt(self):
        """HTML entity &lt; normalizes to < operator."""
        lower, upper = RangeIntegrityEngine._operator_to_bounds("&lt;", 30)
        assert lower == 0
        assert upper == 30

    def test_html_ge(self):
        """HTML entity &ge; normalizes to >= operator."""
        lower, upper = RangeIntegrityEngine._operator_to_bounds("&ge;", 45)
        assert lower == 45
        assert upper == float("inf")

    def test_html_le(self):
        """HTML entity &le; normalizes to <= operator."""
        lower, upper = RangeIntegrityEngine._operator_to_bounds("&le;", 30)
        assert lower == 0
        assert upper == 30.1

    def test_html_gte(self):
        """HTML entity &gt;= normalizes to >= operator."""
        lower, upper = RangeIntegrityEngine._operator_to_bounds("&gt;=", 25)
        assert lower == 25
        assert upper == float("inf")


# ── HTML Entity Pattern Matching ─────────────────────────────────────

class TestHTMLEntityExtraction:
    """Calibration 2: HTML entities in golden dataset spans.

    4 confirmed spans used &gt;/&lt; notation, 6 used en-dash ranges.
    """

    def test_html_gt_in_span_text(self):
        """eGFR &gt; 45 should be recognized as a threshold."""
        text = "initiate dapagliflozin when eGFR &gt; 45"
        spans = [_FakeSpan(text)]
        rie = RangeIntegrityEngine()
        report = rie.validate(spans, text)
        assert report.total_intervals == 1

    def test_html_lt_in_span_text(self):
        """eGFR &lt; 30 should be recognized as a threshold."""
        text = "metformin contraindicated when eGFR &lt; 30"
        spans = [_FakeSpan(text)]
        rie = RangeIntegrityEngine()
        report = rie.validate(spans, text)
        assert report.total_intervals == 1

    def test_html_ge_in_span_text(self):
        """eGFR &ge; 25 should be recognized (golden dataset notation)."""
        text = "dapagliflozin when eGFR &ge; 25"
        spans = [_FakeSpan(text)]
        rie = RangeIntegrityEngine()
        report = rie.validate(spans, text)
        assert report.total_intervals == 1

    def test_en_dash_range(self):
        """eGFR 25\u201359 (en-dash range) from golden dataset."""
        text = "metformin: reduce dose eGFR 25\u201359"  # \u2013 = en-dash
        spans = [_FakeSpan(text)]
        rie = RangeIntegrityEngine()
        report = rie.validate(spans, text)
        assert report.total_intervals == 1
