"""Shared types for the V5 lane specialists.

All specialists return *structured* dataclasses (not raw dicts) so that the
contract is checked at the boundary, not deep inside Channel D / the OCR
extractor. The dataclasses are intentionally narrow — anything richer goes in
``channel_metadata`` on the resulting RawSpan.

Bbox convention
---------------
Specialists emit bboxes in **PDF page-points** (origin top-left), matching the
rest of the V4 pipeline. Sidecar servers that work in pixel coordinates must
convert before responding (the conversion uses the page DPI used at render
time, which the caller supplies and the server echoes back).
"""
from __future__ import annotations

from dataclasses import dataclass, field
from typing import Optional


class SpecialistError(Exception):
    """Base class for all specialist failures.

    Channel D / the orchestrator catches this and falls through to the next
    lane (e.g. nemotron_parse → vlm_table_specialist → docling). Subclasses
    let the caller distinguish "infrastructure missing" from "model returned
    garbage", which matter for different rollback decisions.
    """


class SpecialistUnavailableError(SpecialistError):
    """The specialist could not be reached at all.

    Examples: ``NVIDIA_API_KEY`` unset, sidecar URL not reachable, model
    weights missing on disk. The caller should fall through silently — this is
    expected when a lane is gated behind opt-in infrastructure.
    """


class SpecialistTimeoutError(SpecialistError):
    """The specialist was reachable but exceeded the request budget.

    Distinguished from Unavailable so we can warn loudly: a slow model is a
    different problem from a missing one.
    """


@dataclass(frozen=True)
class TableCellResult:
    """One cell extracted by a table specialist.

    ``row_idx`` and ``col_idx`` are 0-indexed; row 0 is the header row when
    ``is_header=True`` for any cell in that row. When the specialist did not
    distinguish header vs body, callers should treat ``row_idx==0`` as the
    header row by convention.

    ``bbox`` is in PDF page-points; ``None`` if the model did not ground the
    cell. ``confidence`` is in [0,1].
    """
    row_idx: int
    col_idx: int
    text: str
    bbox: Optional[tuple[float, float, float, float]] = None
    confidence: float = 1.0
    is_header: bool = False


@dataclass(frozen=True)
class TableSpecialistResult:
    """Output of a table-lane specialist for one table region.

    ``page_number`` is 1-indexed (matches ``MergedSpan.page_number``).
    ``model_version`` is folded into RawSpan provenance so audits can tell
    which lane produced the cells.
    """
    page_number: int
    cells: list[TableCellResult]
    model_version: str
    table_bbox: Optional[tuple[float, float, float, float]] = None
    metadata: dict = field(default_factory=dict)

    @property
    def is_empty(self) -> bool:
        return not any(c.text.strip() for c in self.cells)


@dataclass(frozen=True)
class OCRBlockResult:
    """One text block from a body-OCR specialist.

    The ``bbox`` here is the *block* bbox (paragraph / line). Per-word bboxes
    live in ``word_spans`` when the specialist supports them
    (``LightOnOCR-2-1B-bbox`` does). Each word_spans entry is a dict with
    keys: ``text``, ``bbox``, ``confidence``.
    """
    text: str
    page_number: int
    bbox: Optional[tuple[float, float, float, float]] = None
    confidence: float = 1.0
    word_spans: list[dict] = field(default_factory=list)


@dataclass(frozen=True)
class OCRSpecialistResult:
    """Output of a body-OCR specialist for one page.

    ``markdown`` is the full reading-order text (what feeds normalized_text).
    ``blocks`` is the structured per-block breakdown that the FieldProvenance
    builder consumes.
    """
    page_number: int
    markdown: str
    blocks: list[OCRBlockResult]
    model_version: str
    metadata: dict = field(default_factory=dict)
