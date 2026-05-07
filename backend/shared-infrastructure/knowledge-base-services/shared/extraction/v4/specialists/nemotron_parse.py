"""Nemotron Parse v1.1 — table-lane specialist (dual-backend).

Two ways to reach the model — both honour the same outward contract:

1. **Sidecar HTTP** (preferred): ``NEMOTRON_PARSE_URL`` points at our own
   ``nemotron_parse_server`` container running ``nvidia/NVIDIA-Nemotron-Parse-v1.1-TC``.
   No per-call cost, deterministic version, no network egress for clinical
   content. The TC variant gives ~20% throughput vs the base v1.1 and
   preserves page order on unordered elements.

2. **NIM cloud** (fallback): ``NVIDIA_API_KEY`` is set and we call
   ``https://integrate.api.nvidia.com/v1/chat/completions`` directly. Useful
   when no GPU is available or the deployment is cloud-only.

If neither env var is set ``extract_table()`` raises ``SpecialistUnavailableError``
and Channel D's priority chain falls through to the next lane silently.

Why dual-backend
----------------
NVIDIA published Parse v1.1 weights publicly under the Open Model License
(<1B params, runs on a single A10/H100), so self-hosting is now an option.
Cloud-only would force every deployment to budget per-call costs, ship table
pixels off-prem, and accept whatever cadence NVIDIA chooses for upgrades.
Sidecar gives us the opposite trade. We let users pick by env var rather
than baking the decision into the code — staging may cloud, prod may
self-host, and the rest of the pipeline doesn't notice.

Why callers don't care which backend ran
----------------------------------------
``extract_table()`` returns a ``TableSpecialistResult`` either way. The
``model_version`` field discloses which path was taken (``...@nim`` vs
``...@sidecar``) so audits can tell, but the cell shape is identical.
"""
from __future__ import annotations

import base64
import json
import logging
import os
import re
from pathlib import Path
from typing import Optional

from .base import (
    SpecialistError,
    SpecialistTimeoutError,
    SpecialistUnavailableError,
    TableCellResult,
    TableSpecialistResult,
)

log = logging.getLogger(__name__)


# ──────────────────────────────────────────────────────────────────────────
# Constants
# ──────────────────────────────────────────────────────────────────────────

# NIM cloud endpoint. Production deployments may override via NVIDIA_NIM_BASE_URL
# (e.g. self-hosted NIM cluster behind a private network).
DEFAULT_NIM_BASE_URL = "https://integrate.api.nvidia.com/v1"
# The cloud-NIM endpoint slug for Parse 1.1 — kept distinct from the local
# sidecar's HF model ID since NVIDIA's NIM catalog uses a flatter naming.
DEFAULT_NIM_MODEL = "nvidia/nemotron-parse-1.1"
# The HF model ID we self-host. The TC variant is the default — token-compression
# wins on every dimension that matters for our workload.
DEFAULT_HF_MODEL = "nvidia/NVIDIA-Nemotron-Parse-v1.1-TC"

# Native PDF render scale for table crops. Parse v1.1 wants images in the
# 1024x1280..1648x2048 range; for typical guideline tables (~6"x4") at PDF
# 72 DPI, a 2.083x scale lands inside that range.
DEFAULT_RENDER_SCALE = 2.083

# Default per-call budget. Sidecar inference for Parse v1.1-TC is ~1-2s on
# H100 / ~5s on A10 / ~25 min on Mac CPU (page-fallback bbox, no flash-attn).
# Bumped from 30s → 1800s after Mac CPU smoke testing showed the original
# 30s budget aborted the call ~50× before completion. The matching server
# never sees the cancellation — it keeps computing and the result is
# discarded on socket close. 1800s is safe headroom on commodity CPU and
# trivially fast on real GPU hardware (1800s ÷ 5s = 360× margin).
#
# Override per-deployment via ``NEMOTRON_PARSE_TIMEOUT_S`` env var.
DEFAULT_TIMEOUT_S = float(os.environ.get("NEMOTRON_PARSE_TIMEOUT_S", "1800"))

# Strict cell-shape JSON prompt for the NIM cloud backend. The local sidecar
# server uses Parse v1.1's native prompt (``</s><s><predict_bbox>...``) and
# the postprocessing module — we don't override it via this client.
_NIM_TABLE_PROMPT = (
    "Extract every cell in this table as JSON. Return ONLY a JSON array — no "
    "prose, no markdown fence, no explanation. Each element MUST have these "
    "fields:\n"
    '  "row_idx": int (0-indexed; 0 = header row if present)\n'
    '  "col_idx": int (0-indexed)\n'
    '  "text":    string (verbatim cell content; empty string if cell is blank)\n'
    '  "bbox":    [x0, y0, x1, y1] floats in [0,1] normalised to image extents\n'
    '  "is_header": bool (true for the header row)\n'
    "Include EVERY cell — including header cells and blank cells. Preserve "
    "leading/trailing whitespace inside text. Do not invent rows or columns "
    "that are not visually present."
)


# ──────────────────────────────────────────────────────────────────────────
# Backend selection
# ──────────────────────────────────────────────────────────────────────────

class _Backend:
    """Lightweight enum-style sentinel for which backend was selected.

    Not an Enum because Python's Enum makes mocking awkward in tests and
    the values are inert — never serialised, never compared with anything
    other than ``is``.
    """
    SIDECAR = "sidecar"
    NIM = "nim"
    UNAVAILABLE = "unavailable"


def _select_backend(sidecar_url: str, api_key: str) -> str:
    """Pick the backend by precedence: sidecar > nim > unavailable.

    Self-host wins because it's cheaper, deterministic, and keeps clinical
    images on-prem. The choice is made per-request rather than at construction
    so that env-var changes between calls (e.g. failover scripts) take effect
    without process restart.
    """
    if sidecar_url:
        return _Backend.SIDECAR
    if api_key:
        return _Backend.NIM
    return _Backend.UNAVAILABLE


# ──────────────────────────────────────────────────────────────────────────
# Specialist class
# ──────────────────────────────────────────────────────────────────────────

class NemotronParseTableSpecialist:
    """Dual-backend client for Nemotron Parse v1.1.

    Stateless w.r.t. the model — backend selection happens per-call, sessions
    are lazy-init and cached. Safe to share across threads.
    """

    def __init__(
        self,
        api_key: Optional[str] = None,
        sidecar_url: Optional[str] = None,
        base_url: str = DEFAULT_NIM_BASE_URL,
        model: str = DEFAULT_NIM_MODEL,
        timeout_s: float = DEFAULT_TIMEOUT_S,
    ):
        self._api_key = api_key or os.environ.get("NVIDIA_API_KEY", "")
        self._sidecar_url = (
            sidecar_url
            or os.environ.get("NEMOTRON_PARSE_URL", "")
        ).rstrip("/")
        self._nim_base_url = base_url.rstrip("/")
        self._nim_model = model
        self._timeout_s = timeout_s
        self._session = None  # lazy

    @property
    def model_version(self) -> str:
        """Disclose which path will be used at next call.

        The audit trail on RawSpan picks this up via ``channel_metadata``;
        a span tagged ``...@sidecar`` is reproducible from local weights, a
        span tagged ``...@nim`` depends on whatever NIM happened to be
        serving that day.
        """
        backend = _select_backend(self._sidecar_url, self._api_key)
        if backend == _Backend.SIDECAR:
            return f"{DEFAULT_HF_MODEL}@sidecar"
        if backend == _Backend.NIM:
            return f"{self._nim_model}@nim"
        return f"{DEFAULT_HF_MODEL}@unavailable"

    @property
    def is_available(self) -> bool:
        """True iff some backend can be reached."""
        return _select_backend(self._sidecar_url, self._api_key) != _Backend.UNAVAILABLE

    @property
    def backend(self) -> str:
        """Expose the resolved backend for tests / metrics."""
        return _select_backend(self._sidecar_url, self._api_key)

    # ──────────────────────────────────────────────────────────────────
    # Public entry point
    # ──────────────────────────────────────────────────────────────────

    def extract_table(
        self,
        pdf_path: str | Path,
        page_number: int,
        table_bbox_pts: tuple[float, float, float, float],
        render_scale: float = DEFAULT_RENDER_SCALE,
    ) -> TableSpecialistResult:
        """Crop the table region and run Nemotron Parse on it.

        Backend selection happens here, not at construction time, so a
        deployment can flip env vars (e.g. cloud failover) without
        restarting the pipeline process.

        Raises:
            SpecialistUnavailableError: No backend reachable.
            SpecialistTimeoutError: Request exceeded ``timeout_s``.
            SpecialistError: Anything else (HTTP error, malformed JSON).
        """
        backend = _select_backend(self._sidecar_url, self._api_key)
        if backend == _Backend.UNAVAILABLE:
            raise SpecialistUnavailableError(
                "Neither NEMOTRON_PARSE_URL nor NVIDIA_API_KEY set; "
                "Nemotron Parse lane unavailable"
            )

        image_b64 = self._render_table_crop(
            pdf_path, page_number, table_bbox_pts, render_scale
        )

        if backend == _Backend.SIDECAR:
            cells = self._call_sidecar(image_b64, page_number, table_bbox_pts)
            model_version = f"{DEFAULT_HF_MODEL}@sidecar"
        else:  # NIM
            raw = self._call_nim_cloud(image_b64)
            cells = self._parse_nim_response(raw, table_bbox_pts)
            model_version = f"{self._nim_model}@nim"

        return TableSpecialistResult(
            page_number=page_number,
            cells=cells,
            model_version=model_version,
            table_bbox=table_bbox_pts,
            metadata={
                "render_scale": render_scale,
                "raw_cell_count": len(cells),
                "backend": backend,
            },
        )

    # ──────────────────────────────────────────────────────────────────
    # Render
    # ──────────────────────────────────────────────────────────────────

    @staticmethod
    def _render_table_crop(
        pdf_path: str | Path,
        page_number: int,
        bbox_pts: tuple[float, float, float, float],
        render_scale: float,
    ) -> str:
        """Render the table region to a base64-encoded PNG.

        Lazy import of PyMuPDF — only this codepath needs it; tests mock the
        method directly so they don't pay for the import.
        """
        try:
            import fitz  # PyMuPDF
        except ImportError as e:
            raise SpecialistUnavailableError(
                f"PyMuPDF (fitz) required for crop rendering: {e}"
            )

        doc = fitz.open(str(pdf_path))
        try:
            if page_number < 1 or page_number > doc.page_count:
                raise SpecialistError(
                    f"page_number {page_number} out of range for PDF with "
                    f"{doc.page_count} pages"
                )
            page = doc[page_number - 1]
            x0, y0, x1, y1 = bbox_pts
            clip = fitz.Rect(x0, y0, x1, y1)
            mat = fitz.Matrix(render_scale, render_scale)
            pix = page.get_pixmap(matrix=mat, clip=clip, alpha=False)
            png_bytes = pix.tobytes("png")
        finally:
            doc.close()

        return base64.b64encode(png_bytes).decode("ascii")

    # ──────────────────────────────────────────────────────────────────
    # SIDECAR backend
    # ──────────────────────────────────────────────────────────────────

    def _call_sidecar(
        self,
        image_b64: str,
        page_number: int,
        table_bbox_pts: tuple[float, float, float, float],
    ) -> list[TableCellResult]:
        """POST to the local sidecar and parse its structured response.

        The sidecar already returns cells in the response format we want
        (the server does the prompt + postprocessing internally), so
        parsing is just dict → dataclass mapping plus bbox conversion.

        Sidecar contract::

            POST {url}/parse
            body: {"image_b64": str, "page_number": int, "render_scale": float}
            returns: {
                "cells": [
                    {"row_idx": int, "col_idx": int, "text": str,
                     "bbox_norm": [x0,y0,x1,y1] in [0,1], "is_header": bool,
                     "confidence": float}
                ],
                "model_version": str
            }
        """
        try:
            import requests
        except ImportError as e:
            raise SpecialistUnavailableError(f"requests library required: {e}")

        if self._session is None:
            self._session = requests.Session()

        url = f"{self._sidecar_url}/parse"
        try:
            resp = self._session.post(
                url,
                json={
                    "image_b64": image_b64,
                    "page_number": page_number,
                },
                timeout=self._timeout_s,
            )
        except requests.exceptions.Timeout as e:
            raise SpecialistTimeoutError(
                f"Nemotron Parse sidecar exceeded {self._timeout_s}s budget"
            ) from e
        except requests.exceptions.ConnectionError as e:
            raise SpecialistUnavailableError(
                f"Nemotron Parse sidecar unreachable at {self._sidecar_url}: {e}"
            ) from e
        except requests.exceptions.RequestException as e:
            raise SpecialistError(f"Nemotron Parse sidecar request failed: {e}") from e

        if resp.status_code >= 400:
            raise SpecialistError(
                f"Nemotron Parse sidecar HTTP {resp.status_code}: {resp.text[:500]}"
            )

        try:
            data = resp.json()
        except ValueError as e:
            raise SpecialistError(f"sidecar returned non-JSON body: {e}") from e

        return _cells_from_sidecar_payload(data, table_bbox_pts)

    # ──────────────────────────────────────────────────────────────────
    # NIM cloud backend
    # ──────────────────────────────────────────────────────────────────

    def _call_nim_cloud(self, image_b64: str) -> str:
        """POST to NIM ``/chat/completions`` and return the assistant message."""
        try:
            import requests
        except ImportError as e:
            raise SpecialistUnavailableError(f"requests library required: {e}")

        if self._session is None:
            self._session = requests.Session()
            self._session.headers.update({
                "Authorization": f"Bearer {self._api_key}",
                "Content-Type": "application/json",
                "Accept": "application/json",
            })

        url = f"{self._nim_base_url}/chat/completions"
        payload = {
            "model": self._nim_model,
            "messages": [{
                "role": "user",
                "content": [
                    {"type": "image_url",
                     "image_url": {"url": f"data:image/png;base64,{image_b64}"}},
                    {"type": "text", "text": _NIM_TABLE_PROMPT},
                ],
            }],
            "temperature": 0.0,
            "max_tokens": 4096,
        }

        try:
            resp = self._session.post(url, json=payload, timeout=self._timeout_s)
        except requests.exceptions.Timeout as e:
            raise SpecialistTimeoutError(
                f"Nemotron Parse NIM exceeded {self._timeout_s}s budget"
            ) from e
        except requests.exceptions.RequestException as e:
            raise SpecialistError(f"Nemotron Parse NIM request failed: {e}") from e

        if resp.status_code in (401, 403):
            raise SpecialistUnavailableError(
                f"NIM auth failed (HTTP {resp.status_code}); check NVIDIA_API_KEY"
            )
        if resp.status_code >= 400:
            raise SpecialistError(
                f"Nemotron Parse NIM HTTP {resp.status_code}: {resp.text[:500]}"
            )

        try:
            data = resp.json()
        except ValueError as e:
            raise SpecialistError(f"NIM returned non-JSON body: {e}") from e

        choices = data.get("choices") or []
        if not choices:
            raise SpecialistError(
                f"NIM response had no choices: {json.dumps(data)[:500]}"
            )
        return choices[0].get("message", {}).get("content") or ""

    # ──────────────────────────────────────────────────────────────────
    # NIM response parser (sidecar uses _cells_from_sidecar_payload)
    # ──────────────────────────────────────────────────────────────────

    @staticmethod
    def _parse_nim_response(
        raw: str,
        table_bbox_pts: tuple[float, float, float, float],
    ) -> list[TableCellResult]:
        """Convert NIM cloud's raw assistant text into TableCellResult.

        VLMs occasionally wrap JSON in code fences or prepend prose; we
        strip / recover defensively so cosmetic differences don't reject
        an otherwise-valid table.
        """
        cleaned = _strip_code_fences(raw).strip()
        if not cleaned:
            return []

        try:
            arr = json.loads(cleaned)
        except json.JSONDecodeError as e:
            arr = _recover_json_array(cleaned)
            if arr is None:
                raise SpecialistError(
                    f"Nemotron Parse NIM returned unparseable JSON: {e} | "
                    f"raw[:200]={cleaned[:200]!r}"
                ) from e

        if not isinstance(arr, list):
            raise SpecialistError(
                f"Nemotron Parse NIM returned non-list JSON: {type(arr).__name__}"
            )

        return _cells_from_normalized_array(arr, table_bbox_pts)

    # Backward compat: tests patched ``_call_nim`` and ``_parse_response`` on
    # the previous interface — keep aliases so they keep working without
    # rewriting the test fixtures that reach into private methods.
    def _call_nim(self, image_b64: str) -> str:  # pragma: no cover (compat alias)
        return self._call_nim_cloud(image_b64)

    @staticmethod
    def _parse_response(  # pragma: no cover (compat alias)
        raw: str,
        table_bbox_pts: tuple[float, float, float, float],
    ) -> list[TableCellResult]:
        return NemotronParseTableSpecialist._parse_nim_response(raw, table_bbox_pts)


# ──────────────────────────────────────────────────────────────────────────
# Cell-conversion helpers (used by both backends)
# ──────────────────────────────────────────────────────────────────────────

def _cells_from_normalized_array(
    arr: list,
    table_bbox_pts: tuple[float, float, float, float],
) -> list[TableCellResult]:
    """Build TableCellResult list from a NIM-style array (``bbox`` in [0,1])."""
    x0, y0, x1, y1 = table_bbox_pts
    w = max(0.0, x1 - x0)
    h = max(0.0, y1 - y0)

    cells: list[TableCellResult] = []
    for item in arr:
        if not isinstance(item, dict):
            continue
        try:
            row_idx = int(item.get("row_idx", 0))
            col_idx = int(item.get("col_idx", 0))
        except (TypeError, ValueError):
            log.warning("nemotron_parse: skipping cell with bad indices: %r", item)
            continue

        text = str(item.get("text", ""))
        is_header = bool(item.get("is_header", row_idx == 0))

        bbox_norm = item.get("bbox")
        bbox_pts = _norm_bbox_to_pts(bbox_norm, x0, y0, w, h)

        confidence = _clamp_unit(item.get("confidence", 0.95))

        cells.append(TableCellResult(
            row_idx=row_idx,
            col_idx=col_idx,
            text=text,
            bbox=bbox_pts,
            confidence=confidence,
            is_header=is_header,
        ))

    return cells


def _cells_from_sidecar_payload(
    data: dict,
    table_bbox_pts: tuple[float, float, float, float],
) -> list[TableCellResult]:
    """Build TableCellResult list from the sidecar's structured response.

    The sidecar emits ``bbox_norm`` in [0,1] (image-space, not
    page-space) so we apply the same crop→page conversion as the NIM path.
    """
    raw_cells = data.get("cells") or []
    if not isinstance(raw_cells, list):
        raise SpecialistError(
            f"Nemotron Parse sidecar returned non-list cells: {type(raw_cells).__name__}"
        )

    x0, y0, x1, y1 = table_bbox_pts
    w = max(0.0, x1 - x0)
    h = max(0.0, y1 - y0)

    cells: list[TableCellResult] = []
    for item in raw_cells:
        if not isinstance(item, dict):
            continue
        try:
            row_idx = int(item.get("row_idx", 0))
            col_idx = int(item.get("col_idx", 0))
        except (TypeError, ValueError):
            log.warning("nemotron_parse sidecar: skipping cell with bad indices: %r", item)
            continue

        text = str(item.get("text", ""))
        is_header = bool(item.get("is_header", row_idx == 0))
        bbox_pts = _norm_bbox_to_pts(item.get("bbox_norm"), x0, y0, w, h)
        confidence = _clamp_unit(item.get("confidence", 0.99))

        cells.append(TableCellResult(
            row_idx=row_idx,
            col_idx=col_idx,
            text=text,
            bbox=bbox_pts,
            confidence=confidence,
            is_header=is_header,
        ))
    return cells


def _norm_bbox_to_pts(
    bbox_norm,
    x0: float,
    y0: float,
    w: float,
    h: float,
) -> Optional[tuple[float, float, float, float]]:
    """Map a normalised [0,1] bbox into PDF points using the table extents.

    Returns ``None`` for any malformed input — callers treat that as
    "cell text without geometry", which is still a useful span.
    """
    if not isinstance(bbox_norm, (list, tuple)) or len(bbox_norm) != 4:
        return None
    if w <= 0 or h <= 0:
        return None
    try:
        nx0, ny0, nx1, ny1 = (float(v) for v in bbox_norm)
    except (TypeError, ValueError):
        return None
    return (
        x0 + nx0 * w,
        y0 + ny0 * h,
        x0 + nx1 * w,
        y0 + ny1 * h,
    )


def _clamp_unit(v) -> float:
    """Coerce to float and clamp to [0,1]; defaults to 0.0 on bad input."""
    try:
        return max(0.0, min(1.0, float(v)))
    except (TypeError, ValueError):
        return 0.0


# ──────────────────────────────────────────────────────────────────────────
# JSON cleanup helpers
# ──────────────────────────────────────────────────────────────────────────

_FENCE_RE = re.compile(r"^```(?:json|JSON)?\s*\n?|\n?```\s*$", re.MULTILINE)


def _strip_code_fences(text: str) -> str:
    """Remove ```json ... ``` wrappers if present. Idempotent."""
    return _FENCE_RE.sub("", text)


def _recover_json_array(text: str) -> Optional[list]:
    """Last-ditch recovery: find first '[' through the matching ']'.

    Handles the "Here is the JSON:\n[...]" and "Here you go: [...]" cases.
    Returns None when recovery fails.
    """
    start = text.find("[")
    end = text.rfind("]")
    if start < 0 or end <= start:
        return None
    candidate = text[start: end + 1]
    try:
        result = json.loads(candidate)
    except json.JSONDecodeError:
        return None
    return result if isinstance(result, list) else None
