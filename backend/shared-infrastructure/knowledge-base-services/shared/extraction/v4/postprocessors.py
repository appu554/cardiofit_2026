"""
V4 Post-processors for span refinement.

Utilities applied after initial channel extraction to improve span quality.
Currently contains:
- extend_parenthetical: Extends spans to include unmatched closing brackets

Pipeline Position:
    Channel B/C raw span -> postprocessors -> final RawSpan
"""

from __future__ import annotations


def extend_parenthetical(
    text: str,
    start: int,
    end: int,
    max_extension: int = 100,
) -> tuple[int, int]:
    """Extend a span to include unmatched closing parentheses or brackets.

    Clinical text frequently has patterns like:
        "eGFR < 30 mL/min/1.73m2 (contraindicated)"
    where a channel captures "eGFR < 30 mL/min/1.73m2" but misses the
    parenthetical qualifier. This function extends the span rightward
    to capture the closing delimiter.

    Args:
        text: Full normalized text.
        start: Current span start offset.
        end: Current span end offset.
        max_extension: Maximum characters to scan forward (caps at EOL).

    Returns:
        (new_start, new_end) — unchanged if no extension needed.
    """
    span_text = text[start:end]

    # Count unmatched openers in the current span
    open_parens = span_text.count("(") - span_text.count(")")
    open_brackets = span_text.count("[") - span_text.count("]")

    if open_parens <= 0 and open_brackets <= 0:
        return start, end

    # Scan forward from end, up to max_extension or end of line
    scan_limit = min(len(text), end + max_extension)
    new_end = end

    parens_needed = max(open_parens, 0)
    brackets_needed = max(open_brackets, 0)

    for i in range(end, scan_limit):
        ch = text[i]

        # Stop at line boundaries
        if ch == "\n":
            break

        if ch == ")" and parens_needed > 0:
            parens_needed -= 1
            new_end = i + 1
            if parens_needed == 0 and brackets_needed == 0:
                break
        elif ch == "]" and brackets_needed > 0:
            brackets_needed -= 1
            new_end = i + 1
            if parens_needed == 0 and brackets_needed == 0:
                break

    return start, new_end
