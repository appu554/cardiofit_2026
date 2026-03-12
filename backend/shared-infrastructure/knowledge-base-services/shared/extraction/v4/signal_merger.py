"""
Signal Merger: Combines multi-channel extraction outputs.

Takes the union of all RawSpans from Channels B-F, clusters overlapping
spans, boosts confidence for multi-channel agreement, and produces
MergedSpans ready for the reviewer queue.

Algorithm:
1. UNION — Collect all RawSpans from all channels, sort by start offset
2. CLUSTER — Overlapping spans (≥50% overlap) → one cluster
3. CONFIDENCE BOOST — 1 channel: +0.00, 2: +0.05, 3: +0.10, 4+: +0.15
4. TEXT — Use longest span in cluster as merged text
5. DISAGREEMENT — Flag if contributing spans have different text
6. SECTION ASSIGNMENT — Use Channel A's tree for section_id
7. OUTPUT — Produce list[MergedSpan]

Pipeline Position:
    Channels B-F (all outputs) -> Signal Merger (THIS) -> DB -> Reviewer
"""

from __future__ import annotations

import time
from collections import defaultdict
from typing import Optional
from uuid import UUID

from .models import (
    ChannelOutput,
    GuidelineSection,
    GuidelineTree,
    MergedSpan,
    RawSpan,
    SectionPassage,
)
from .tiering_classifier import TieringClassifier


# Confidence boost by number of contributing channels
CONFIDENCE_BOOST = {
    1: 0.00,
    2: 0.05,
    3: 0.10,
}
CONFIDENCE_BOOST_MAX = 0.15  # for 4+ channels


class SignalMerger:
    """Merge multi-channel extraction outputs into reviewer-ready MergedSpans.

    Produces one MergedSpan per cluster of overlapping RawSpans.
    Flags disagreements where channels produce different text.
    """

    VERSION = "4.2.4"
    OVERLAP_THRESHOLD = 0.5  # ≥50% character overlap to cluster

    def merge(
        self,
        job_id: UUID,
        channel_outputs: list[ChannelOutput],
        tree: GuidelineTree,
        classifier: Optional[TieringClassifier] = None,
    ) -> list[MergedSpan]:
        """Merge all channel outputs into MergedSpans.

        Args:
            job_id: The extraction job UUID
            channel_outputs: List of ChannelOutput from channels B-F
            tree: GuidelineTree from Channel A (for section assignment)
            classifier: Optional TieringClassifier to assign tier labels.
                If None, MergedSpan.tier remains None (backward compat).

        Returns:
            List of MergedSpan objects ready for reviewer queue
        """
        # Step 1: UNION — collect and sort all spans
        all_spans = self._collect_all_spans(channel_outputs)

        if not all_spans:
            return []

        # Step 2: CLUSTER — group overlapping spans
        clusters = self._cluster_overlapping(all_spans)

        # Steps 3-6: Build MergedSpans from clusters
        merged_spans: list[MergedSpan] = []
        for cluster in clusters:
            merged = self._build_merged_span(job_id, cluster, tree, classifier)
            merged_spans.append(merged)

        # Step 7: Page coverage assertion — detect page_map poisoning
        self._validate_page_coverage(merged_spans, tree)

        return merged_spans

    def _validate_page_coverage(
        self, spans: list[MergedSpan], tree: GuidelineTree
    ) -> None:
        """Warn if page distribution is suspiciously skewed.

        Catches stale page_map data where many spans incorrectly map to
        page 1 (the default when offset lookup fails on a broken page_map).
        """
        if not spans or tree.total_pages <= 1:
            return

        from collections import Counter
        page_counts = Counter(s.page_number for s in spans)
        total = len(spans)

        # Check 1: Page 1 concentration — should not exceed 20% for multi-page docs
        page1_count = page_counts.get(1, 0)
        page1_pct = page1_count / total * 100
        if tree.total_pages > 10 and page1_pct > 20:
            import warnings
            warnings.warn(
                f"SignalMerger page_map warning: {page1_pct:.0f}% of spans "
                f"({page1_count}/{total}) mapped to page 1. "
                f"This may indicate a broken page_map in the guideline tree. "
                f"Expected <20% for a {tree.total_pages}-page document.",
                stacklevel=2,
            )

        # Check 2: page_map coverage — should have entries for most pages
        if tree.page_map:
            mapped_pages = len(set(tree.page_map.values()))
            coverage = mapped_pages / tree.total_pages * 100
            if coverage < 80:
                import warnings
                warnings.warn(
                    f"SignalMerger page_map coverage: {coverage:.0f}% "
                    f"({mapped_pages}/{tree.total_pages} pages). "
                    f"Rebuild guideline_tree from fixed L1 cache.",
                    stacklevel=2,
                )

    def _collect_all_spans(
        self, channel_outputs: list[ChannelOutput]
    ) -> list[RawSpan]:
        """Collect all spans from all channels, sorted by start offset."""
        all_spans: list[RawSpan] = []
        for output in channel_outputs:
            if output.success:
                all_spans.extend(output.spans)

        all_spans.sort(key=lambda s: (s.start, -len(s.text)))
        return all_spans

    def _cluster_overlapping(
        self, spans: list[RawSpan]
    ) -> list[list[RawSpan]]:
        """Group overlapping spans into clusters.

        Two spans are in the same cluster if they share ≥50% character
        overlap (relative to the shorter span).
        """
        if not spans:
            return []

        clusters: list[list[RawSpan]] = []
        used: set[int] = set()

        for i, span_i in enumerate(spans):
            if i in used:
                continue

            cluster = [span_i]
            used.add(i)

            # Find the current cluster's effective range
            cluster_start = span_i.start
            cluster_end = span_i.end

            for j, span_j in enumerate(spans):
                if j in used:
                    continue
                if j <= i:
                    continue

                # Check overlap with the cluster's current range
                if self._has_significant_overlap(
                    cluster_start, cluster_end, span_j.start, span_j.end
                ):
                    cluster.append(span_j)
                    used.add(j)
                    # Expand cluster range
                    cluster_start = min(cluster_start, span_j.start)
                    cluster_end = max(cluster_end, span_j.end)

            clusters.append(cluster)

        return clusters

    def _has_significant_overlap(
        self,
        start1: int, end1: int,
        start2: int, end2: int,
    ) -> bool:
        """Check if two spans overlap by ≥50% of the shorter span."""
        overlap_start = max(start1, start2)
        overlap_end = min(end1, end2)

        if overlap_start >= overlap_end:
            return False

        overlap_len = overlap_end - overlap_start
        shorter_len = min(end1 - start1, end2 - start2)

        if shorter_len <= 0:
            return False

        return overlap_len / shorter_len >= self.OVERLAP_THRESHOLD

    def _build_merged_span(
        self,
        job_id: UUID,
        cluster: list[RawSpan],
        tree: GuidelineTree,
        classifier: Optional[TieringClassifier] = None,
    ) -> MergedSpan:
        """Build a MergedSpan from a cluster of overlapping RawSpans."""
        # Step 3: CONFIDENCE BOOST
        channels = list({s.channel for s in cluster})
        channel_count = len(channels)
        boost = CONFIDENCE_BOOST.get(channel_count, CONFIDENCE_BOOST_MAX)

        # Channel confidences
        channel_confidences: dict[str, float] = {}
        for span in cluster:
            if span.channel not in channel_confidences:
                channel_confidences[span.channel] = span.confidence
            else:
                # Keep the higher confidence for same channel
                channel_confidences[span.channel] = max(
                    channel_confidences[span.channel], span.confidence
                )

        # Base confidence = weighted average of channel confidences
        if channel_confidences:
            base_confidence = sum(channel_confidences.values()) / len(channel_confidences)
        else:
            base_confidence = 0.0

        merged_confidence = min(1.0, base_confidence + boost)

        # Step 4: TEXT — use longest span text
        longest_span = max(cluster, key=lambda s: len(s.text))
        merged_text = longest_span.text
        merged_start = longest_span.start
        merged_end = longest_span.end

        # Step 5: DISAGREEMENT — check if texts differ
        unique_texts = {s.text.strip().lower() for s in cluster}
        has_disagreement = len(unique_texts) > 1
        disagreement_detail = None
        if has_disagreement:
            # V4.2.3: For each channel, keep the longest span text.
            # Previous code used a dict comprehension that kept the LAST
            # entry per channel, which could differ from merged_text
            # (selected by longest span globally). Using longest-per-channel
            # ensures disagreement_detail is consistent with merged text.
            text_by_channel: dict[str, str] = {}
            for span in cluster:
                if (span.channel not in text_by_channel
                        or len(span.text) > len(text_by_channel[span.channel])):
                    text_by_channel[span.channel] = span.text
            disagreement_detail = " vs ".join(
                f"{ch}:'{txt}'" for ch, txt in sorted(text_by_channel.items())
            )

        # Step 6: SECTION + PAGE ASSIGNMENT
        # V4.2.2: page_number from direct offset lookup, NOT section.page_number.
        # section.page_number reflects the heading's page — wrong for spans
        # deep within multi-page sections (e.g., heading on p.92 but span on p.96).
        #
        # V4.2.4: When merged_start < 0 (Channel D table cells, Channel H
        # recovery, L1_RECOVERY spans that lack text offsets), offset-based
        # lookup returns a wrong default (page 1). Fall back to the most
        # common page_number carried by the contributing RawSpans.
        if merged_start >= 0:
            section = tree.find_section_for_offset(merged_start)
            section_id = section.section_id if section else None
            page_number = tree.get_page_for_offset(merged_start)
            table = tree.find_table_for_offset(merged_start)
            table_id = table.table_id if table else None
        else:
            # Fallback for spans without text offsets (Channel D table
            # cells, Channel H recovery, L1_RECOVERY): use metadata
            # carried by the contributing RawSpans.
            span_sections = [s.section_id for s in cluster if s.section_id]
            section_id = (max(set(span_sections), key=span_sections.count)
                          if span_sections else None)
            span_pages = [s.page_number for s in cluster if s.page_number is not None]
            page_number = (max(set(span_pages), key=span_pages.count)
                           if span_pages else None)
            span_tables = [s.table_id for s in cluster if s.table_id]
            table_id = (max(set(span_tables), key=span_tables.count)
                        if span_tables else None)

        # Step 7: TIERING — classify span if classifier provided
        tier = None
        tier_reason = None
        if classifier is not None:
            tiering_result = classifier.classify(
                text=merged_text,
                merged_confidence=merged_confidence,
                contributing_channels=sorted(channels),
                channel_confidences=channel_confidences,
                has_disagreement=has_disagreement,
                section_id=section_id,
                page_number=page_number,
            )
            tier = tiering_result.tier
            tier_reason = tiering_result.reason

        return MergedSpan(
            job_id=job_id,
            text=merged_text,
            start=merged_start,
            end=merged_end,
            contributing_channels=sorted(channels),
            channel_confidences=channel_confidences,
            merged_confidence=merged_confidence,
            has_disagreement=has_disagreement,
            disagreement_detail=disagreement_detail,
            tier=tier,
            tier_reason=tier_reason,
            page_number=page_number,
            section_id=section_id,
            table_id=table_id,
        )

    # ── V4.2.1: Section Passage Assembly ─────────────────────────────────

    def assemble_section_passages(
        self,
        merged_spans: list[MergedSpan],
        tree: GuidelineTree,
        text: str,
    ) -> list[SectionPassage]:
        """Assemble prose passages per section with MergedSpan provenance.

        Uses the tree's expanded offset ranges (post-reparenting) and the
        tree's TableBoundary ranges (from Marker) to excise table content.

        Args:
            merged_spans: MergedSpans from merge()
            tree: GuidelineTree (post-reparenting, post-offset-expansion)
            text: Normalized text (Channel 0 output)

        Returns:
            List of SectionPassage, one per section in DFS order
        """
        # Build section_id → spans index
        section_spans: dict[str, list[MergedSpan]] = defaultdict(list)
        for span in merged_spans:
            if span.section_id:
                section_spans[span.section_id].append(span)

        # Build table exclusion ranges from tree.tables (Marker boundaries)
        table_ranges: list[tuple[int, int]] = sorted(
            (t.start_offset, t.end_offset)
            for t in tree.tables
            if t.start_offset >= 0 and t.end_offset >= 0
        )

        # Flatten tree to all sections in DFS order
        all_sections = self._flatten_tree(tree.sections)

        passages: list[SectionPassage] = []
        for section in all_sections:
            # Use expanded offset range (covers reparented children)
            prose = self._extract_prose_text(
                text, section.start_offset, section.end_offset, table_ranges
            )

            # Collect overlapping MergedSpan UUIDs
            overlapping_ids: list[UUID] = []
            for span in merged_spans:
                if span.start < section.end_offset and span.end > section.start_offset:
                    overlapping_ids.append(span.id)

            # Also include spans indexed by section_id
            for span in section_spans.get(section.section_id, []):
                if span.id not in overlapping_ids:
                    overlapping_ids.append(span.id)

            passages.append(SectionPassage(
                section_id=section.section_id,
                heading=section.heading,
                page_number=section.page_number,
                prose_text=prose,
                span_ids=overlapping_ids,
                span_count=len(overlapping_ids),
                child_section_ids=[c.section_id for c in section.children],
                start_offset=section.start_offset,
                end_offset=section.end_offset,
            ))

        return passages

    @staticmethod
    def _flatten_tree(
        sections: list[GuidelineSection],
    ) -> list[GuidelineSection]:
        """Recursively collect all sections in DFS order."""
        result: list[GuidelineSection] = []
        for section in sections:
            result.append(section)
            if section.children:
                result.extend(SignalMerger._flatten_tree(section.children))
        return result

    @staticmethod
    def _extract_prose_text(
        text: str,
        start: int,
        end: int,
        table_ranges: list[tuple[int, int]],
    ) -> str:
        """Extract section text with table regions excised.

        Uses sorted table boundary ranges from the GuidelineTree (Marker's
        table detection), not Channel D span offsets.
        """
        if start >= end or start >= len(text):
            return ""

        # Clamp to text bounds
        start = max(0, start)
        end = min(len(text), end)

        # Find table ranges that overlap [start, end)
        excisions: list[tuple[int, int]] = []
        for tbl_start, tbl_end in table_ranges:
            # Check overlap
            ov_start = max(start, tbl_start)
            ov_end = min(end, tbl_end)
            if ov_start < ov_end:
                excisions.append((ov_start, ov_end))

        if not excisions:
            return text[start:end]

        # Build prose by skipping excised regions
        parts: list[str] = []
        cursor = start
        for exc_start, exc_end in excisions:
            if cursor < exc_start:
                parts.append(text[cursor:exc_start])
            cursor = max(cursor, exc_end)
        if cursor < end:
            parts.append(text[cursor:end])

        return "".join(parts)
