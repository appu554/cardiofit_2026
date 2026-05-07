"""Nemotron Nano VL 8B — figure-lane specialist (sidecar HTTP client).

Why this lane exists
--------------------
Figures, therapeutic algorithms and decision-tree flowcharts are visual
artifacts that the OCR + table lanes can't decompose into clinical facts on
their own. KDIGO §3.1.4 "drug class hierarchy" is a classic example: a
flowchart node like "if eGFR <30 → reduce dose" is a discrete clinical fact
but lives only in pixels.

Nemotron Nano VL 8B (``nvidia/Llama-3.1-Nemotron-Nano-VL-8B-v1``) tops
OCRBench v2 on diagram and infographic reasoning. It's open-weight and runs
on a single GPU, which is exactly the right operating point: we need
deterministic, on-prem inference for clinical-grade auditability.

Why it's a sidecar, not in-process
----------------------------------
The pipeline image force-pins ``transformers==4.51.0`` for magic-pdf /
docling / gliner compat. Nemotron Nano VL works with that pin, but its
``trust_remote_code=True`` modelling code pulls heavyweight extras
(``timm``, ``open_clip``, custom CRADIOv2 vision encoder) that we don't want
bloating the main image. A separate sidecar — same pattern as Ollama for
Channel F — keeps the main image lean and lets the figure lane scale on
GPU nodes independently.

This module is the *client* — the server is in
``tools/guideline-atomiser/servers/nemotron_vl_server.py``.

Escalation
----------
For the small slice of figures Nano VL can't handle (complex multi-panel
decision trees), the protocol routes to Gemini 2.5 Pro. That escalation
is gated on ``confidence < FIGURE_ESCALATION_THRESHOLD`` and lives outside
this module — see ``extraction.v4.specialists.figure_router``. (Not yet
implemented; tracked as a follow-up.)
"""
from __future__ import annotations

import base64
import io
import logging
import os
from pathlib import Path
from typing import Optional

from .base import (
    SpecialistError,
    SpecialistTimeoutError,
    SpecialistUnavailableError,
)

log = logging.getLogger(__name__)


DEFAULT_TIMEOUT_S = 60.0
DEFAULT_RENDER_SCALE = 2.083  # ~150 DPI for Nano VL's 512x512-tile encoder


class NemotronNanoVLFigureSpecialist:
    """HTTP client for the Nano VL sidecar.

    The sidecar exposes a single endpoint::

        POST {base_url}/figure
        body: {"image_b64": str, "task": "extract_facts" | "describe", "max_tokens": int}
        returns: {"description": str, "facts": list[str], "confidence": float, "model_version": str}

    The server is responsible for prompt templating + structured-output
    coaxing; this client just transports the image and parses the response.
    Keeping the sidecar in charge of the prompt means we can re-tune
    extraction behaviour without redeploying the pipeline.
    """

    DEFAULT_BASE_URL_ENV = "NEMOTRON_VL_URL"

    def __init__(
        self,
        base_url: Optional[str] = None,
        timeout_s: float = DEFAULT_TIMEOUT_S,
    ):
        self._base_url = (
            base_url
            or os.environ.get(self.DEFAULT_BASE_URL_ENV, "")
        ).rstrip("/")
        self._timeout_s = timeout_s
        self._session = None

    @property
    def is_available(self) -> bool:
        """True iff a sidecar URL is configured. We don't ping at import time —
        an unreachable URL becomes ``SpecialistUnavailableError`` on first call.
        """
        return bool(self._base_url)

    def describe_figure(
        self,
        pdf_path: str | Path,
        page_number: int,
        figure_bbox_pts: tuple[float, float, float, float],
        task: str = "extract_facts",
        render_scale: float = DEFAULT_RENDER_SCALE,
    ) -> dict:
        """Describe / extract structured facts from a figure region.

        Returns the sidecar's raw response dict — schema is defined by the
        server contract above. Callers convert to RawSpan downstream.
        """
        if not self._base_url:
            raise SpecialistUnavailableError(
                f"{self.DEFAULT_BASE_URL_ENV} not set; figure lane unavailable"
            )

        image_b64 = _render_pdf_crop(pdf_path, page_number, figure_bbox_pts, render_scale)
        return self._call_sidecar(image_b64, task=task)

    def _call_sidecar(self, image_b64: str, task: str) -> dict:
        try:
            import requests
        except ImportError as e:
            raise SpecialistUnavailableError(f"requests required: {e}")

        if self._session is None:
            self._session = requests.Session()

        url = f"{self._base_url}/figure"
        try:
            resp = self._session.post(
                url,
                json={
                    "image_b64": image_b64,
                    "task": task,
                    "max_tokens": 1024,
                },
                timeout=self._timeout_s,
            )
        except requests.exceptions.Timeout as e:
            raise SpecialistTimeoutError(
                f"Nemotron Nano VL exceeded {self._timeout_s}s budget"
            ) from e
        except requests.exceptions.ConnectionError as e:
            # Sidecar down or network blip → treat as unavailable so the
            # pipeline can fall through. Distinct from a transient timeout.
            raise SpecialistUnavailableError(
                f"Nano VL sidecar unreachable at {self._base_url}: {e}"
            ) from e
        except requests.exceptions.RequestException as e:
            raise SpecialistError(f"Nano VL request failed: {e}") from e

        if resp.status_code >= 500:
            # 5xx: server bug; surface so alerts fire.
            raise SpecialistError(
                f"Nano VL HTTP {resp.status_code}: {resp.text[:500]}"
            )
        if resp.status_code >= 400:
            raise SpecialistError(
                f"Nano VL HTTP {resp.status_code}: {resp.text[:500]}"
            )

        try:
            return resp.json()
        except ValueError as e:
            raise SpecialistError(f"Nano VL returned non-JSON: {e}") from e


def _render_pdf_crop(
    pdf_path: str | Path,
    page_number: int,
    bbox_pts: tuple[float, float, float, float],
    render_scale: float,
) -> str:
    """Render a PDF region to a base64-encoded PNG.

    Shared with ``nemotron_parse._render_table_crop`` in spirit — kept
    separate to avoid a cross-module dependency that would force
    PyMuPDF to be imported at package-init time.
    """
    try:
        import fitz
    except ImportError as e:
        raise SpecialistUnavailableError(
            f"PyMuPDF (fitz) required for figure rendering: {e}"
        )

    doc = fitz.open(str(pdf_path))
    try:
        if page_number < 1 or page_number > doc.page_count:
            raise SpecialistError(
                f"page_number {page_number} out of range (PDF has {doc.page_count})"
            )
        page = doc[page_number - 1]
        x0, y0, x1, y1 = bbox_pts
        clip = fitz.Rect(x0, y0, x1, y1)
        mat = fitz.Matrix(render_scale, render_scale)
        pix = page.get_pixmap(matrix=mat, clip=clip, alpha=False)
        png = pix.tobytes("png")
    finally:
        doc.close()
    return base64.b64encode(png).decode("ascii")
