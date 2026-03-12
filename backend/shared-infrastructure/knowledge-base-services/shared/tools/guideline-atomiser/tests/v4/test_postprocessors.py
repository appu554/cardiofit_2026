"""
Tests for V4 Post-processors: Parenthetical Extension.

Validates:
1. No-op when span has balanced brackets
2. Extension to capture unmatched closing parenthesis
3. Extension to capture unmatched closing bracket
4. Mixed parentheses and brackets
5. Newline boundary stops extension
6. max_extension cap
7. Multiple unmatched openers
8. Extension near end of text
"""

import sys
from pathlib import Path

import pytest

SHARED_DIR = Path(__file__).resolve().parents[4]
sys.path.insert(0, str(SHARED_DIR))

from extraction.v4.postprocessors import extend_parenthetical


class TestExtendParenthetical:
    """Tests for extend_parenthetical()."""

    def test_no_extension_when_balanced(self):
        """Balanced brackets should return unchanged span."""
        text = "metformin (500 mg) is recommended"
        start = 0
        end = len("metformin (500 mg)")  # already balanced
        new_start, new_end = extend_parenthetical(text, start, end)
        assert (new_start, new_end) == (start, end)

    def test_no_extension_when_no_brackets(self):
        """Span without any brackets should return unchanged."""
        text = "eGFR >= 30 mL/min"
        new_start, new_end = extend_parenthetical(text, 0, len(text))
        assert (new_start, new_end) == (0, len(text))

    def test_extends_to_closing_paren(self):
        """Unmatched '(' should extend to find closing ')'."""
        text = "eGFR < 30 mL/min/1.73m2 (contraindicated) extra text"
        # Span captures up to "(contraindicated" but misses the ")"
        span_text = "eGFR < 30 mL/min/1.73m2 (contraindicated"
        start = 0
        end = len(span_text)
        new_start, new_end = extend_parenthetical(text, start, end)
        assert new_start == 0
        assert text[new_start:new_end] == "eGFR < 30 mL/min/1.73m2 (contraindicated)"

    def test_extends_to_closing_bracket(self):
        """Unmatched '[' should extend to find closing ']'."""
        text = "metformin [Grade 1A] is first-line"
        span_text = "metformin [Grade 1A"
        start = 0
        end = len(span_text)
        new_start, new_end = extend_parenthetical(text, start, end)
        assert text[new_start:new_end] == "metformin [Grade 1A]"

    def test_mixed_parens_and_brackets(self):
        """Both unmatched '(' and '[' should be resolved."""
        text = "dose (10 mg [adjusted] for CKD) more text"
        # Span has "dose (10 mg [adjusted" — missing "]" and ")"
        span_text = "dose (10 mg [adjusted"
        start = 0
        end = len(span_text)
        new_start, new_end = extend_parenthetical(text, start, end)
        # Should extend to capture both "]" and ")"
        assert text[new_start:new_end] == "dose (10 mg [adjusted] for CKD)"

    def test_stops_at_newline(self):
        """Extension should stop at newline boundary."""
        text = "eGFR (contraindicated\nnew paragraph here)"
        span_text = "eGFR (contraindicated"
        start = 0
        end = len(span_text)
        new_start, new_end = extend_parenthetical(text, start, end)
        # The ")" is after newline, so extension stops
        assert new_end == end  # no extension

    def test_max_extension_cap(self):
        """Extension should not exceed max_extension characters."""
        filler = "x" * 200
        text = f"dose (10 mg{filler})"
        span_text = "dose (10 mg"
        start = 0
        end = len(span_text)
        # Default max_extension=100, so ")" at 211 chars away is unreachable
        new_start, new_end = extend_parenthetical(text, start, end)
        assert new_end == end  # can't reach closing paren

        # With larger max_extension, it should work
        new_start2, new_end2 = extend_parenthetical(text, start, end, max_extension=250)
        assert text[new_start2:new_end2] == f"dose (10 mg{filler})"

    def test_multiple_unmatched_openers(self):
        """Multiple unmatched openers should find multiple closers."""
        text = "dose ((adjusted)) text"
        # Span has "dose ((adjusted" — two unmatched "("
        span_text = "dose ((adjusted"
        start = 0
        end = len(span_text)
        new_start, new_end = extend_parenthetical(text, start, end)
        assert text[new_start:new_end] == "dose ((adjusted))"

    def test_extension_at_end_of_text(self):
        """Extension near end of text should not go out of bounds."""
        text = "metformin (500 mg"
        start = 0
        end = len(text)
        # No closing paren exists — should return original
        new_start, new_end = extend_parenthetical(text, start, end)
        assert new_end == end

    def test_span_in_middle_of_text(self):
        """Extension works correctly for spans not starting at offset 0."""
        text = "Prefix text. eGFR (below 30 mL/min) is the threshold."
        # Span starts at "eGFR" and ends before ")"
        prefix = "Prefix text. "
        start = len(prefix)
        span_text = "eGFR (below 30 mL/min"
        end = start + len(span_text)
        new_start, new_end = extend_parenthetical(text, start, end)
        assert text[new_start:new_end] == "eGFR (below 30 mL/min)"

    def test_excess_closers_no_extension(self):
        """More closers than openers should not trigger extension."""
        text = "dose 500 mg) already closed"
        start = 0
        end = len("dose 500 mg)")
        new_start, new_end = extend_parenthetical(text, start, end)
        assert (new_start, new_end) == (start, end)
