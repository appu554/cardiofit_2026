"""LightOnOCR-2-1B (bbox variant) — body-OCR sidecar HTTP client.

Why this lane exists
--------------------
``lightonai/LightOnOCR-2-1B-bbox`` does two things our pipeline currently has
no clean source for:

1. **Fast body-text OCR** — 1B params, ~5.7 pages/s on H100, SOTA-at-its-size
   on OlmOCR-Bench. Faster than MonkeyOCR's full Qwen2.5-VL pass for the
   pure-text body of long documents.

2. **Per-word bounding boxes** — the ``-bbox`` variant emits coords for each
   word, which is exactly the missing input for Feature #2 (``FieldProvenance``
   on NER channels E/F). When GLiNER extracts "30 mL/min" from "eGFR ≥30
   mL/min, continue Metformin", the per-word bboxes let us rebuild a
   character-offset → page-coord map and populate sub-span geometry without
   the architecture refactor we'd need to give Channels E/F access to blocks.

Why it's a sidecar
------------------
``LightOnOcrForConditionalGeneration`` lands in ``transformers≥4.55``, but
the main pipeline force-pins to 4.51.0 (magic-pdf / docling / gliner all
fight over that exact version). Same pattern as Nano VL — separate
container, thin HTTP boundary.

Server contract
---------------
::

    POST {base_url}/ocr
    body: {"image_b64": str, "page_number": int, "bbox": bool}
    returns: {
        "page_number": int,
        "markdown": str,                    # reading-order body text
        "blocks": [
            {
                "text": str,
                "bbox": [x0, y0, x1, y1],   # in PDF points (page-relative)
                "confidence": float,
                "word_spans": [             # only present when bbox=true
                    {"text": str, "bbox": [...], "confidence": float},
                    ...
                ],
            }
        ],
        "model_version": str,
    }

Rendering: callers send the *whole page* image, not a region. The page-DPI
used at render time is implied (the server knows the model's input
resolution and rescales on its end). Bbox coordinates come back in PDF
points (page-relative), already converted by the server — this matches
``MergedSpan.bbox`` so no further transform is needed downstream.
"""
from __future__ import annotations

import base64
import logging
import os
from pathlib import Path
from typing import Optional

from .base import (
    OCRBlockResult,
    OCRSpecialistResult,
    SpecialistError,
    SpecialistTimeoutError,
    SpecialistUnavailableError,
)

log = logging.getLogger(__name__)


DEFAULT_TIMEOUT_S = 30.0
DEFAULT_RENDER_SCALE = 2.77  # 200 DPI per LightOnOCR docs


class LightOnOCRBodySpecialist:
    """HTTP client for the LightOnOCR-2-1B-bbox sidecar.

    Single-page granularity: the caller drives a loop over pages and merges
    the markdown / block list. We don't batch in this client because the
    sidecar already serves concurrent requests via vLLM — batching here
    would just add buffer-and-flush latency for no gain.
    """

    DEFAULT_BASE_URL_ENV = "LIGHTONOCR_URL"

    def __init__(
        self,
        base_url: Optional[str] = None,
        timeout_s: float = DEFAULT_TIMEOUT_S,
        bbox: bool = True,
    ):
        self._base_url = (
            base_url
            or os.environ.get(self.DEFAULT_BASE_URL_ENV, "")
        ).rstrip("/")
        self._timeout_s = timeout_s
        self._bbox = bbox
        self._session = None

    @property
    def is_available(self) -> bool:
        return bool(self._base_url)

    def ocr_page(
        self,
        pdf_path: str | Path,
        page_number: int,
        render_scale: float = DEFAULT_RENDER_SCALE,
    ) -> OCRSpecialistResult:
        """Render a page to PNG and OCR it via the sidecar.

        Returns the parsed structured result. ``markdown`` feeds
        ``normalized_text``; ``blocks[*].word_spans`` feeds the
        char-offset → bbox index used by Feature #2.
        """
        if not self._base_url:
            raise SpecialistUnavailableError(
                f"{self.DEFAULT_BASE_URL_ENV} not set; LightOnOCR lane unavailable"
            )

        image_b64 = _render_full_page(pdf_path, page_number, render_scale)
        raw = self._call_sidecar(image_b64, page_number)
        return _parse_ocr_response(raw, page_number)

    def _call_sidecar(self, image_b64: str, page_number: int) -> dict:
        try:
            import requests
        except ImportError as e:
            raise SpecialistUnavailableError(f"requests required: {e}")

        if self._session is None:
            self._session = requests.Session()

        url = f"{self._base_url}/ocr"
        try:
            resp = self._session.post(
                url,
                json={
                    "image_b64": image_b64,
                    "page_number": page_number,
                    "bbox": self._bbox,
                },
                timeout=self._timeout_s,
            )
        except requests.exceptions.Timeout as e:
            raise SpecialistTimeoutError(
                f"LightOnOCR exceeded {self._timeout_s}s budget"
            ) from e
        except requests.exceptions.ConnectionError as e:
            raise SpecialistUnavailableError(
                f"LightOnOCR sidecar unreachable at {self._base_url}: {e}"
            ) from e
        except requests.exceptions.RequestException as e:
            raise SpecialistError(f"LightOnOCR request failed: {e}") from e

        if resp.status_code >= 400:
            raise SpecialistError(
                f"LightOnOCR HTTP {resp.status_code}: {resp.text[:500]}"
            )
        try:
            return resp.json()
        except ValueError as e:
            raise SpecialistError(f"LightOnOCR returned non-JSON: {e}") from e


# ──────────────────────────────────────────────────────────────────────────
# Helpers
# ──────────────────────────────────────────────────────────────────────────

def _render_full_page(
    pdf_path: str | Path,
    page_number: int,
    render_scale: float,
) -> str:
    try:
        import fitz
    except ImportError as e:
        raise SpecialistUnavailableError(f"PyMuPDF required: {e}")

    doc = fitz.open(str(pdf_path))
    try:
        if page_number < 1 or page_number > doc.page_count:
            raise SpecialistError(
                f"page_number {page_number} out of range (PDF has {doc.page_count})"
            )
        page = doc[page_number - 1]
        mat = fitz.Matrix(render_scale, render_scale)
        pix = page.get_pixmap(matrix=mat, alpha=False)
        png = pix.tobytes("png")
    finally:
        doc.close()
    return base64.b64encode(png).decode("ascii")


def _parse_ocr_response(raw: dict, expected_page: int) -> OCRSpecialistResult:
    """Convert the sidecar's JSON to a typed ``OCRSpecialistResult``."""
    blocks_raw = raw.get("blocks") or []
    blocks: list[OCRBlockResult] = []
    for b in blocks_raw:
        if not isinstance(b, dict):
            continue
        bbox = b.get("bbox")
        bbox_tuple = None
        if isinstance(bbox, (list, tuple)) and len(bbox) == 4:
            try:
                bbox_tuple = tuple(float(v) for v in bbox)
            except (TypeError, ValueError):
                bbox_tuple = None

        word_spans = b.get("word_spans") or []
        # Defensive: we keep word_spans as raw dicts (already structured) so
        # downstream code can iterate without re-typing.
        word_spans = [w for w in word_spans if isinstance(w, dict)]

        blocks.append(OCRBlockResult(
            text=str(b.get("text", "")),
            page_number=int(b.get("page_number", expected_page)),
            bbox=bbox_tuple,
            confidence=float(b.get("confidence", 1.0)),
            word_spans=word_spans,
        ))

    return OCRSpecialistResult(
        page_number=int(raw.get("page_number", expected_page)),
        markdown=str(raw.get("markdown", "")),
        blocks=blocks,
        model_version=str(raw.get("model_version", "lightonocr-2-1b-bbox")),
        metadata={"raw_block_count": len(blocks_raw)},
    )
