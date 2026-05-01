"""V5 channel provenance model.

A ChannelProvenance records WHO observed a span (which channel), WHERE on
the page (bbox + page_number), HOW confident they were, and WHICH model
version was used. Lists of these go into the merged_spans.json
`channel_provenance` field and the KB-0 `l2_merged_spans.provenance_v5`
jsonb column.

Used by:
  - extraction.v4.signal_merger to thread per-channel observations into
    the final merged span
  - tools/guideline-atomiser/data/push_to_kb0_gcp.py for serialisation to
    the canonical_facts DB
  - tests/v5/* acceptance tests for coverage assertions

Schema is backward-compatible: spans without channel_provenance default to
an empty list, so existing V4 jobs continue to validate.
"""
from __future__ import annotations

from typing import Annotated, Iterable

from pydantic import BaseModel, ConfigDict, Field, model_validator

# Channel IDs are stable across V4 and V5: 0 (normaliser), A (docling),
# B (drug dict), C (grammar), D (table), E (gliner), F (nuextract),
# G (sentence), H (recovery).
ChannelId = Annotated[str, Field(pattern=r"^[0A-H]$")]


class BoundingBox(BaseModel):
    """Page-coordinate bounding box. (x0, y0) = top-left, (x1, y1) = bottom-right.

    Coordinates are in PDF points (typographic), origin at top-left.
    """
    model_config = ConfigDict(extra="forbid")
    x0: float = Field(ge=0)
    y0: float = Field(ge=0)
    x1: float
    y1: float

    @model_validator(mode="after")
    def _check_ordered(self) -> "BoundingBox":
        if self.x1 < self.x0:
            raise ValueError(f"bbox x1 ({self.x1}) < x0 ({self.x0})")
        if self.y1 < self.y0:
            raise ValueError(f"bbox y1 ({self.y1}) < y0 ({self.y0})")
        return self


class ChannelProvenance(BaseModel):
    """Per-channel evidence for a merged span.

    Lists of ChannelProvenance form the audit trail enabling V5 subsystems
    #1, #3, #4, #5 to compare and aggregate channel signals.
    """
    model_config = ConfigDict(extra="forbid")

    channel_id: ChannelId
    bbox: BoundingBox
    page_number: int = Field(ge=1)
    confidence: float = Field(ge=0.0, le=1.0)
    model_version: str = Field(min_length=1, max_length=200)
    notes: str | None = None  # optional free-text per-observation note


def merge_provenance_lists(
    lists: Iterable[list[ChannelProvenance]],
) -> list[ChannelProvenance]:
    """Concat multiple per-channel provenance lists into one for a merged span.

    Dedup rule: when two entries share the same (channel_id, bbox, page_number),
    keep the one with the higher confidence. This matches the signal_merger
    semantics of "channels can re-fire on the same region; the strongest wins".
    """
    out: dict[tuple[str, int, float, float, float, float], ChannelProvenance] = {}
    for lst in lists:
        for p in lst:
            key = (p.channel_id, p.page_number,
                   p.bbox.x0, p.bbox.y0, p.bbox.x1, p.bbox.y1)
            existing = out.get(key)
            if existing is None or p.confidence > existing.confidence:
                out[key] = p
    return list(out.values())


def serialise_provenance_list(
    items: Iterable[ChannelProvenance],
) -> list[dict]:
    """Render a list to JSON-compatible dicts for jsonb storage."""
    return [item.model_dump() for item in items]
