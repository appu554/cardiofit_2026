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

Note for migrators: V4's MergedSpan.bbox is `Optional[list[float]]` (flat 4-tuple).
Translate to the V5 BoundingBox via:
    BoundingBox(x0=b[0], y0=b[1], x1=b[2], y1=b[3])
"""
from __future__ import annotations

from typing import Iterable, Literal

from pydantic import BaseModel, ConfigDict, Field, model_validator

# Channel IDs — the union of all channels that may observe spans across V4 and V5:
#   "0"           — text normaliser (pre-merger stage)
#   "A"           — docling structure parser (pre-merger stage)
#   "B"-"H"       — V4 main extraction channels (drug dict, grammar, table,
#                   gliner, nuextract, sentence, recovery)
#   "L1_RECOVERY" — L1 Completeness Oracle injection
#   "REVIEWER"    — human-edited spans from KB-0 review
# Mirrors MergedSpan.contributing_channels in extraction/v4/models.py and adds
# the pre-merger stages (0, A) that produce structure consumed by B-H.
ChannelId = Literal[
    "0", "A", "B", "C", "D", "E", "F", "G", "H",
    "L1_RECOVERY", "REVIEWER",
]


# Sanity ceiling for bbox coords — ~7× the largest realistic PDF page (14400 pt).
# Catches garbage upstream (e.g. float-overflow producing 1e308) without
# rejecting any genuine document.
_MAX_PT = 100_000.0


class BoundingBox(BaseModel):
    """Page-coordinate bounding box. (x0, y0) = top-left, (x1, y1) = bottom-right.

    Coordinates are in PDF points (typographic), origin at top-left.
    """
    model_config = ConfigDict(extra="forbid")
    x0: float = Field(ge=0, le=_MAX_PT)
    y0: float = Field(ge=0, le=_MAX_PT)
    x1: float = Field(ge=0, le=_MAX_PT)
    y1: float = Field(ge=0, le=_MAX_PT)

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
    notes: str | None = Field(default=None, max_length=500)


# Bbox-coordinate rounding precision for dedup keys. PDF audit only needs
# ~0.01 pt resolution, and rounding here makes dedup robust to ULP-level
# differences from upstream `max()` arithmetic in table_boundary_oracle and
# similar bbox-merging code paths.
_BBOX_DEDUP_NDIGITS = 2


def _bbox_dedup_key(b: BoundingBox) -> tuple[float, float, float, float]:
    """Bbox key for dedup that tolerates ULP-level float differences."""
    return (
        round(b.x0, _BBOX_DEDUP_NDIGITS),
        round(b.y0, _BBOX_DEDUP_NDIGITS),
        round(b.x1, _BBOX_DEDUP_NDIGITS),
        round(b.y1, _BBOX_DEDUP_NDIGITS),
    )


class FieldProvenance(BaseModel):
    """Per-extracted-value bounding box within a span.

    Populated when the extraction model returns grounded per-cell geometry
    (e.g. MonkeyOCR Qwen2.5-VL OCR spans). Enables sub-span traceability:
    a table cell span carrying "eGFR ≥30 mL/min" can record WHERE on the page
    that value appeared, separate from the span's overall bbox.

    Schema is additive — RawSpan.field_provenance defaults to an empty list
    so existing V4 spans remain valid.
    """
    model_config = ConfigDict(extra="forbid")

    field_name: str = Field(min_length=1, max_length=200)
    value: str = Field(default="", max_length=2000)
    bbox: BoundingBox
    page_number: int = Field(ge=1)
    confidence: float = Field(ge=0.0, le=1.0)
    channel_id: ChannelId


def merge_field_provenance_lists(
    lists: Iterable[Iterable[FieldProvenance]],
) -> list[FieldProvenance]:
    """Merge per-fact provenance from a span cluster, deduplicating by (field_name, page, bbox).

    Dedup rule: same (field_name, page_number, rounded-bbox) → keep highest confidence.
    """
    out: dict[tuple, FieldProvenance] = {}
    for lst in lists:
        for fp in lst:
            key = (fp.field_name, fp.page_number) + _bbox_dedup_key(fp.bbox)
            existing = out.get(key)
            if existing is None or fp.confidence > existing.confidence:
                out[key] = fp
    return list(out.values())


def merge_provenance_lists(
    lists: Iterable[Iterable[ChannelProvenance]],
) -> list[ChannelProvenance]:
    """Concat multiple per-channel provenance lists into one for a merged span.

    Dedup rule: when two entries share the same (channel_id, page_number,
    rounded-bbox), keep the one with the higher confidence. This matches the
    signal_merger semantics of "channels can re-fire on the same region;
    the strongest wins". Bbox coords are rounded to ~0.01 pt for dedup so
    that ULP-level differences from upstream bbox arithmetic do not produce
    spurious duplicates.
    """
    out: dict[tuple, ChannelProvenance] = {}
    for lst in lists:
        for p in lst:
            key = (p.channel_id, p.page_number) + _bbox_dedup_key(p.bbox)
            existing = out.get(key)
            if existing is None or p.confidence > existing.confidence:
                out[key] = p
    return list(out.values())


def serialise_provenance_list(
    items: Iterable[ChannelProvenance],
) -> list[dict[str, object]]:
    """Render a list to JSON-compatible dicts for jsonb storage."""
    return [item.model_dump() for item in items]


def get_channel_provenance_builder(channel_id: str):
    """Return the per-channel provenance builder for a given channel_id.

    Returns None if no builder is registered for that channel (e.g., REVIEWER
    spans, which signal_merger handles differently — there is no extraction
    model attached to a human-edited span).

    Lazy imports inside the function avoid the circular-import problem: each
    channel module imports ChannelProvenance from this module, so a top-level
    import here would cycle.
    """
    if channel_id == "0":
        from .channel_0_normalizer import _channel_0_provenance
        return _channel_0_provenance
    if channel_id == "A":
        from .channel_a_docling import _channel_a_provenance
        return _channel_a_provenance
    if channel_id == "B":
        from .channel_b_drug_dict import _channel_b_provenance
        return _channel_b_provenance
    if channel_id == "C":
        from .channel_c_grammar import _channel_c_provenance
        return _channel_c_provenance
    if channel_id == "D":
        from .channel_d_table import _channel_d_provenance
        return _channel_d_provenance
    if channel_id == "E":
        from .channel_e_gliner import _channel_e_provenance
        return _channel_e_provenance
    if channel_id == "F":
        from .channel_f_nuextract import _channel_f_provenance
        return _channel_f_provenance
    if channel_id == "G":
        from .channel_g_sentence import _channel_g_provenance
        return _channel_g_provenance
    if channel_id == "H":
        from .channel_h_recovery import _channel_h_provenance
        return _channel_h_provenance
    if channel_id == "L1_RECOVERY":
        # L1_RECOVERY uses Channel H semantics (recovery-grade extraction).
        from .channel_h_recovery import _channel_h_provenance
        return _channel_h_provenance
    # REVIEWER, "" and arbitrary unknowns: no builder. signal_merger handles
    # human-edited spans separately when KB-0 review edits land.
    return None


def _normalise_bbox(
    bbox: tuple[float, float, float, float] | list[float] | None,
) -> "BoundingBox | None":
    """Build a BoundingBox from raw upstream coords, with defensive clamping.

    Returns None when bbox is missing or malformed (so callers can skip without
    a behaviour change). When bbox is a 4-tuple/list, clamps negatives to 0 and
    enforces x1>=x0, y1>=y0. Coordinates above the BoundingBox sanity ceiling
    (100_000 pt) cause Pydantic validation to raise; this is intentional —
    garbage upstream should fail loudly, not be silently truncated.
    """
    if bbox is None:
        return None
    if len(bbox) != 4:
        return None
    x0, y0, x1, y1 = bbox
    x0 = max(0.0, float(x0))
    y0 = max(0.0, float(y0))
    x1 = max(x0, float(x1))
    y1 = max(y0, float(y1))
    return BoundingBox(x0=x0, y0=y0, x1=x1, y1=y1)


def _normalise_page_number(page_number: int | float | None) -> int:
    """Clamp page_number to ≥1. Defaults to 1 if None or non-numeric.

    Page-number is intentionally lenient (clamp-not-reject) because upstream
    parsers occasionally emit page=0 for "document header" / non-page content;
    the audit trail needs a valid integer regardless.
    """
    if page_number is None:
        return 1
    try:
        return max(1, int(page_number))
    except (TypeError, ValueError):
        return 1


def _normalise_confidence(confidence: float | None) -> float:
    """Clamp confidence to [0.0, 1.0]. Defaults to 0.0 if None or non-numeric."""
    if confidence is None:
        return 0.0
    try:
        return max(0.0, min(1.0, float(confidence)))
    except (TypeError, ValueError):
        return 0.0
