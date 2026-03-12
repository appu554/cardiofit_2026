"""
Channel H: Cross-Channel Recovery.

Analyzes MergedSpans where only a single channel contributed
(``len(contributing_channels) == 1``) and checks if the text pattern
SHOULD have been caught by another channel. Produces recovery spans
with metadata indicating which channel missed the match and why.

This is a post-merge analysis channel — it takes MergedSpans (not raw
text) and produces additional RawSpans that feed back into the merger
for the next iteration, or are reported as recovery diagnostics.

Recovery Patterns:
- Drug name in text but Channel B didn't fire → likely word boundary issue
- Numeric threshold in text but Channel C didn't fire → pattern gap
- Drug class mentioned but not matched → dictionary gap

Pipeline Position:
    Signal Merger -> Channel H (THIS, post-merge) -> Recovery Report
"""

from __future__ import annotations

import re
import time
from typing import Optional
from uuid import uuid4

from .models import ChannelOutput, GuidelineTree, MergedSpan, RawSpan


# Common drug names that Channel B should catch
_DRUG_NAMES = frozenset({
    "metformin", "dapagliflozin", "empagliflozin", "canagliflozin",
    "ertugliflozin", "sotagliflozin", "finerenone", "spironolactone",
    "eplerenone", "lisinopril", "enalapril", "ramipril", "losartan",
    "valsartan", "irbesartan", "semaglutide", "liraglutide", "dulaglutide",
    "sitagliptin", "linagliptin", "pioglitazone", "insulin", "furosemide",
    "amlodipine", "atorvastatin", "rosuvastatin",
})

# Numeric threshold patterns that Channel C should catch
_THRESHOLD_RE = re.compile(
    r"(?:eGFR|CrCl|GFR|HbA1c|A1c)\s*[<>=≥≤]+\s*\d+",
    re.IGNORECASE,
)

# Drug class patterns
_DRUG_CLASS_RE = re.compile(
    r"\b(?:SGLT2|GLP-1|ACE|ARB|MRA|DPP-4|TZD|NSAID|statin)\b",
    re.IGNORECASE,
)


class ChannelHRecovery:
    """Cross-channel recovery analysis for single-channel spans."""

    VERSION = "4.3.0"
    CONFIDENCE = 0.60  # Recovery spans are lower confidence
    CHANNEL = "H"

    def extract(
        self,
        merged_spans: list[MergedSpan],
        text: str,
        tree: GuidelineTree,
    ) -> ChannelOutput:
        """Analyze single-channel spans for missed cross-channel matches.

        Args:
            merged_spans: MergedSpans from Signal Merger.
            text: Normalized text (Channel 0 output).
            tree: GuidelineTree from Channel A.

        Returns:
            ChannelOutput with recovery spans and diagnostic metadata.
        """
        start_time = time.monotonic()

        single_channel_spans = [
            s for s in merged_spans
            if len(s.contributing_channels) == 1
        ]

        recovery_spans: list[RawSpan] = []
        recovery_reasons: dict[str, int] = {
            "drug_missed_by_b": 0,
            "threshold_missed_by_c": 0,
            "class_missed_by_b": 0,
        }

        for span in single_channel_spans:
            channel = span.contributing_channels[0]
            span_text = span.text.lower()

            # Check 1: Drug name in text but Channel B didn't fire
            if channel != "B":
                for drug in _DRUG_NAMES:
                    if drug in span_text:
                        recovery_spans.append(self._build_recovery_span(
                            span, text, tree,
                            reason="drug_missed_by_b",
                            detail=f"'{drug}' present but Channel B didn't match",
                            original_channel=channel,
                        ))
                        recovery_reasons["drug_missed_by_b"] += 1
                        break  # one recovery per span

            # Check 2: Threshold pattern but Channel C didn't fire
            if channel != "C":
                if _THRESHOLD_RE.search(span.text):
                    recovery_spans.append(self._build_recovery_span(
                        span, text, tree,
                        reason="threshold_missed_by_c",
                        detail="Threshold pattern present but Channel C didn't match",
                        original_channel=channel,
                    ))
                    recovery_reasons["threshold_missed_by_c"] += 1

            # Check 3: Drug class but Channel B didn't fire
            if channel != "B":
                if _DRUG_CLASS_RE.search(span.text):
                    # Don't double-count if already caught by drug name check
                    if not any(drug in span_text for drug in _DRUG_NAMES):
                        recovery_spans.append(self._build_recovery_span(
                            span, text, tree,
                            reason="class_missed_by_b",
                            detail="Drug class pattern present but Channel B didn't match",
                            original_channel=channel,
                        ))
                        recovery_reasons["class_missed_by_b"] += 1

        elapsed_ms = (time.monotonic() - start_time) * 1000

        return ChannelOutput(
            channel=self.CHANNEL,
            spans=recovery_spans,
            metadata={
                "single_channel_spans_analyzed": len(single_channel_spans),
                "total_merged_spans": len(merged_spans),
                "recovery_reasons": recovery_reasons,
                "recovery_rate_pct": round(
                    len(recovery_spans) / max(len(merged_spans), 1) * 100, 1
                ),
            },
            elapsed_ms=elapsed_ms,
        )

    def _build_recovery_span(
        self,
        source_span: MergedSpan,
        text: str,
        tree: GuidelineTree,
        reason: str,
        detail: str,
        original_channel: str,
    ) -> RawSpan:
        """Build a recovery RawSpan from a MergedSpan."""
        return RawSpan(
            channel=self.CHANNEL,
            text=source_span.text,
            start=source_span.start,
            end=source_span.end,
            confidence=self.CONFIDENCE,
            page_number=source_span.page_number,
            section_id=source_span.section_id,
            source_block_type="paragraph",
            channel_metadata={
                "recovery_reason": reason,
                "recovery_detail": detail,
                "original_channel": original_channel,
                "original_confidence": source_span.merged_confidence,
            },
        )


# Convenience alias
ChannelH = ChannelHRecovery
