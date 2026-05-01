"""
Channel G: Sentence-Level Context Extraction.

Takes the union of B-F span offsets, finds the enclosing sentence
boundaries, and produces sentence-level context spans. These derived
spans carry lower confidence (0.70) since they're not primary matches
but provide richer context for downstream dossier assembly.

Algorithm:
1. Collect all unique span offsets from Channels B-F
2. For each offset, expand to sentence boundaries (`. ` or `\\n\\n`)
3. Deduplicate overlapping sentence spans
4. Tag with source channel metadata

Pipeline Position:
    Channels B-F -> Channel G (THIS, post-B-F) -> Signal Merger
"""

from __future__ import annotations

import time
from typing import Optional
from uuid import uuid4

from .models import ChannelOutput, GuidelineTree, RawSpan
from .provenance import (
    ChannelProvenance,
    _normalise_bbox,
    _normalise_confidence,
    _normalise_page_number,
)
from .v5_flags import is_v5_enabled


def _channel_g_model_version() -> str:
    """Channel G model version (deterministic sentence boundary expansion)."""
    return "sentence@v1.0"


def _channel_g_provenance(
    bbox,
    page_number,
    confidence,
    profile,
    notes: Optional[str] = None,
) -> Optional[ChannelProvenance]:
    """Build a ChannelProvenance entry for Channel G (sentence-level context).

    Returns None when V5_BBOX_PROVENANCE is off or bbox is missing. Bbox is
    typically inherited from the originating B-F span's parent block.
    """
    if not is_v5_enabled("bbox_provenance", profile):
        return None
    bb = _normalise_bbox(bbox)
    if bb is None:
        return None
    return ChannelProvenance(
        channel_id="G",
        bbox=bb,
        page_number=_normalise_page_number(page_number),
        confidence=_normalise_confidence(confidence),
        model_version=_channel_g_model_version(),
        notes=notes,
    )


class ChannelGSentence:
    """Sentence-level context extractor derived from B-F span positions."""

    VERSION = "4.3.0"
    CONFIDENCE = 0.70  # Derived, not primary
    CHANNEL = "G"

    # Sentence boundary characters
    _SENTENCE_TERMINATORS = frozenset({
        ".", "!", "?", "\n\n",
    })

    # Maximum sentence length (prevents runaway expansion)
    MAX_SENTENCE_LENGTH = 500

    def extract(
        self,
        text: str,
        tree: GuidelineTree,
        prior_channel_outputs: list[ChannelOutput],
    ) -> ChannelOutput:
        """Extract sentence-level context spans around B-F matches.

        Args:
            text: Normalized text (Channel 0 output).
            tree: GuidelineTree from Channel A.
            prior_channel_outputs: Outputs from Channels B-F.

        Returns:
            ChannelOutput with sentence-level spans.
        """
        start_time = time.monotonic()

        # Step 1: Collect unique span midpoints from B-F
        anchor_positions: list[tuple[int, int, list[str]]] = []
        for output in prior_channel_outputs:
            if not output.success:
                continue
            for span in output.spans:
                anchor_positions.append(
                    (span.start, span.end, [span.channel])
                )

        if not anchor_positions:
            return ChannelOutput(
                channel=self.CHANNEL,
                spans=[],
                metadata={"source_span_count": 0},
                elapsed_ms=0,
            )

        # Step 2: Expand each anchor to sentence boundaries
        raw_sentences: list[tuple[int, int, list[str]]] = []
        for start, end, channels in anchor_positions:
            sent_start = self._find_sentence_start(text, start)
            sent_end = self._find_sentence_end(text, end)

            # Clamp to max length
            if sent_end - sent_start > self.MAX_SENTENCE_LENGTH:
                mid = (start + end) // 2
                sent_start = max(sent_start, mid - self.MAX_SENTENCE_LENGTH // 2)
                sent_end = min(sent_end, mid + self.MAX_SENTENCE_LENGTH // 2)

            raw_sentences.append((sent_start, sent_end, channels))

        # Step 3: Merge overlapping sentence spans
        merged = self._merge_overlapping(raw_sentences)

        # Step 4: Build RawSpan objects
        spans: list[RawSpan] = []
        for sent_start, sent_end, source_channels in merged:
            sent_text = text[sent_start:sent_end].strip()
            if not sent_text or len(sent_text) < 10:
                continue

            page_number = tree.get_page_for_offset(sent_start)
            section = tree.find_section_for_offset(sent_start)
            section_id = section.section_id if section else None

            spans.append(RawSpan(
                channel=self.CHANNEL,
                text=sent_text,
                start=sent_start,
                end=sent_end,
                confidence=self.CONFIDENCE,
                page_number=page_number,
                section_id=section_id,
                source_block_type="paragraph",
                channel_metadata={
                    "source_channels": sorted(set(source_channels)),
                    "source_span_count": len(source_channels),
                },
            ))

        elapsed_ms = (time.monotonic() - start_time) * 1000

        return ChannelOutput(
            channel=self.CHANNEL,
            spans=spans,
            metadata={
                "source_span_count": len(anchor_positions),
                "sentence_count": len(spans),
            },
            elapsed_ms=elapsed_ms,
        )

    def _find_sentence_start(self, text: str, pos: int) -> int:
        """Find the start of the sentence containing `pos`."""
        # Walk backward to find ". " or "\n\n" or start of text
        search_start = max(0, pos - self.MAX_SENTENCE_LENGTH)
        region = text[search_start:pos]

        # Look for the last sentence terminator
        best = 0  # relative to search_start
        for i in range(len(region) - 1, -1, -1):
            ch = region[i]
            if ch in (".", "!", "?"):
                # Check it's followed by whitespace (sentence boundary, not decimal)
                abs_pos = search_start + i + 1
                if abs_pos < len(text) and text[abs_pos] in (" ", "\n", "\t"):
                    best = i + 1  # start after the terminator
                    break
            elif ch == "\n" and i > 0 and region[i - 1] == "\n":
                best = i + 1
                break

        return search_start + best

    def _find_sentence_end(self, text: str, pos: int) -> int:
        """Find the end of the sentence containing `pos`."""
        search_end = min(len(text), pos + self.MAX_SENTENCE_LENGTH)
        region = text[pos:search_end]

        for i, ch in enumerate(region):
            if ch in (".", "!", "?"):
                abs_pos = pos + i + 1
                if abs_pos < len(text) and text[abs_pos] in (" ", "\n", "\t"):
                    return abs_pos
            elif ch == "\n" and i + 1 < len(region) and region[i + 1] == "\n":
                return pos + i

        return search_end

    @staticmethod
    def _merge_overlapping(
        spans: list[tuple[int, int, list[str]]],
    ) -> list[tuple[int, int, list[str]]]:
        """Merge overlapping sentence spans, combining source channels."""
        if not spans:
            return []

        # Sort by start position
        sorted_spans = sorted(spans, key=lambda x: x[0])
        merged: list[tuple[int, int, list[str]]] = []

        curr_start, curr_end, curr_channels = sorted_spans[0]
        for start, end, channels in sorted_spans[1:]:
            if start <= curr_end:
                # Overlapping — merge
                curr_end = max(curr_end, end)
                curr_channels = curr_channels + channels
            else:
                merged.append((curr_start, curr_end, curr_channels))
                curr_start, curr_end, curr_channels = start, end, channels

        merged.append((curr_start, curr_end, curr_channels))
        return merged


# Convenience alias
ChannelG = ChannelGSentence
