"""
Channel E: GLiNER Residual Booster.

Wraps existing ClinicalNERExtractor from extraction/gliner/extractor.py with
two critical changes from V3:

1. No truncation: Runs on the FULL normalized text, not markdown_text[:5000].
   The V3 truncation was a primary cause of missed entities.
2. Novel-only filtering: Only emits spans NOT already found by Channels B + C.
   GLiNER becomes a safety net for novel entities, not the primary NER.

Pipeline Position:
    Channel A (GuidelineTree) -> Channels B+C (first) -> Channel E (THIS)
    Requires: Channel B + C outputs (for novel-only filtering)
"""

from __future__ import annotations

import time
from typing import Optional

from .models import ChannelOutput, GuidelineTree, RawSpan


class ChannelEGLiNERResidual:
    """GLiNER residual channel — catches entities missed by B+C.

    Also exported as ``ChannelE`` for pipeline convenience.

    Uses the existing ClinicalNERExtractor but filters to novel spans only.
    Falls back gracefully if GLiNER model is not available.
    """

    VERSION = "4.0.0"

    def __init__(self) -> None:
        """Initialize GLiNER wrapper.

        Imports the existing ClinicalNERExtractor lazily to avoid
        hard dependency on the GLiNER model at import time.
        """
        self._extractor = None
        self._available = False
        self._init_error: Optional[str] = None

        try:
            from extraction.gliner.extractor import ClinicalNERExtractor
            self._extractor = ClinicalNERExtractor()
            self._available = True
        except Exception as e:
            self._init_error = str(e)

    @property
    def available(self) -> bool:
        """Whether the GLiNER model loaded successfully."""
        return self._available

    def extract(
        self,
        text: str,
        tree: GuidelineTree,
        existing_spans: list[RawSpan],
    ) -> ChannelOutput:
        """Run GLiNER on FULL text, then subtract spans already found by B+C.

        CRITICAL: Do NOT truncate text. The old pipeline had
        markdown_text[:5000] which is why GLiNER missed entities
        beyond the first few pages.

        Args:
            text: Full normalized text (Channel 0 output) — NO truncation
            tree: GuidelineTree from Channel A
            existing_spans: RawSpans from Channels B + C (for deduplication)

        Returns:
            ChannelOutput with novel-only GLiNER spans
        """
        start_time = time.monotonic()

        if not self._available:
            return ChannelOutput(
                channel="E",
                spans=[],
                error=f"GLiNER not available: {self._init_error}",
                elapsed_ms=0.0,
            )

        # Run GLiNER on FULL text — no truncation
        # When target_kb is "all" or unspecified, use extract_entities()
        # which runs ALL clinical labels. extract_for_kb() only accepts
        # "dosing", "safety", "monitoring" — NOT "all".
        try:
            gliner_result = self._extractor.extract_entities(text)
        except Exception as e:
            elapsed_ms = (time.monotonic() - start_time) * 1000
            return ChannelOutput(
                channel="E",
                spans=[],
                error=f"GLiNER extraction failed: {e}",
                elapsed_ms=elapsed_ms,
            )

        # Convert GLiNER entities to RawSpans
        raw_spans = self._convert_to_raw_spans(gliner_result, tree, text)

        # Filter to novel-only spans (not already covered by B+C)
        novel_spans = self._filter_novel(raw_spans, existing_spans)

        elapsed_ms = (time.monotonic() - start_time) * 1000

        return ChannelOutput(
            channel="E",
            spans=novel_spans,
            metadata={
                "gliner_total": len(raw_spans),
                "novel_after_filter": len(novel_spans),
                "existing_spans_checked": len(existing_spans),
            },
            elapsed_ms=elapsed_ms,
        )

    def _convert_to_raw_spans(
        self, gliner_result, tree: GuidelineTree, text: str
    ) -> list[RawSpan]:
        """Convert GLiNER extraction result to V4 RawSpan objects."""
        spans: list[RawSpan] = []

        entities = getattr(gliner_result, 'entities', [])
        if isinstance(entities, list):
            for entity in entities:
                # Handle both dict and object entity formats
                if isinstance(entity, dict):
                    entity_text = entity.get('text', '')
                    start = entity.get('start', 0)
                    end = entity.get('end', 0)
                    label = entity.get('label', '')
                    score = entity.get('score', 0.5)
                else:
                    entity_text = getattr(entity, 'text', '')
                    start = getattr(entity, 'start', 0)
                    end = getattr(entity, 'end', 0)
                    label = getattr(entity, 'label', '')
                    score = getattr(entity, 'score', 0.5)

                if not entity_text:
                    continue

                # Clamp confidence to [0, 1]
                confidence = max(0.0, min(1.0, float(score)))

                # Find section
                # V4.2.2: direct offset lookup for correct page
                section = tree.find_section_for_offset(start)
                section_id = section.section_id if section else None
                page = tree.get_page_for_offset(start)

                # Determine block type
                table = tree.find_table_for_offset(start)
                block_type = "table_cell" if table else "paragraph"

                spans.append(RawSpan(
                    channel="E",
                    text=entity_text,
                    start=start,
                    end=end,
                    confidence=confidence,
                    page_number=page,
                    section_id=section_id,
                    source_block_type=block_type,
                    channel_metadata={
                        "gliner_label": label,
                        "gliner_score": score,
                    },
                ))

        return spans

    def _filter_novel(
        self,
        gliner_spans: list[RawSpan],
        existing_spans: list[RawSpan],
    ) -> list[RawSpan]:
        """Keep only GLiNER spans not already covered by B+C.

        A GLiNER span is "covered" if an existing span overlaps by >50%
        of the GLiNER span's length.
        """
        novel: list[RawSpan] = []

        for g_span in gliner_spans:
            is_covered = False
            g_len = g_span.end - g_span.start

            if g_len <= 0:
                continue

            for existing in existing_spans:
                overlap_start = max(g_span.start, existing.start)
                overlap_end = min(g_span.end, existing.end)

                if overlap_start < overlap_end:
                    overlap_len = overlap_end - overlap_start
                    if overlap_len / g_len > 0.5:
                        is_covered = True
                        break

            if not is_covered:
                novel.append(g_span)

        return novel


# Short alias for pipeline imports
ChannelE = ChannelEGLiNERResidual
