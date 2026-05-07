"""Nemotron Parse v1.1-TC FastAPI server.

Runs ``nvidia/NVIDIA-Nemotron-Parse-v1.1-TC`` locally and exposes a thin
HTTP boundary that returns table cells in the shape Channel D's specialist
client expects::

    POST /parse
    body: {"image_b64": str, "page_number": int}
    returns: {
        "page_number": int,
        "cells": [
            {"row_idx": int, "col_idx": int, "text": str,
             "bbox_norm": [x0,y0,x1,y1] in [0,1] image-space,
             "is_header": bool, "confidence": float}
        ],
        "model_version": str,
    }

The model produces a *structured token sequence* with class tags
(``<table>``, ``<text>``, …), bounding-box tokens
(``<bbox_x0_y0_x1_y1>``), and inline content. We post-process to:

1. Filter to ``table``-classed regions (we only care about table cells here —
   text/figure regions go through the OCR / figure lanes).
2. Convert the LaTeX/HTML/markdown table content into ``(row_idx, col_idx, text)``
   tuples. The model's own ``postprocessing.py`` handles bbox extraction and
   format normalisation; we let it do the work and then layer a small
   table-row parser on top.

Why we lean on the model's ``postprocessing`` module
----------------------------------------------------
The bbox token format is undocumented in the readable parts of the model
card (it's mentioned but the exact lex grammar isn't given). The
``postprocessing.py`` shipped with the model is the canonical parser —
better to depend on it than to reimplement and silently drift when NVIDIA
adjusts the format in v1.2.
"""
from __future__ import annotations

import base64
import io
import logging
import os
import re
import sys
from threading import Lock

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field
from PIL import Image

log = logging.getLogger("nemotron_parse_server")
logging.basicConfig(level=logging.INFO)

app = FastAPI(title="Nemotron Parse v1.1-TC sidecar", version="0.1.0")

_MODEL = None
_TOKENIZER = None
_PROCESSOR = None
_GEN_CONFIG = None
_POSTPROCESSING = None  # The model's own postprocessing module, lazy-imported
_MODEL_LOCK = Lock()

_MODEL_ID = os.environ.get(
    "NEMOTRON_PARSE_MODEL", "nvidia/NVIDIA-Nemotron-Parse-v1.1-TC"
)
_DEVICE = os.environ.get("NEMOTRON_PARSE_DEVICE", "auto")  # auto | cuda | cpu

# The native prompt for full extraction (bboxes + classes + markdown).
_FULL_PROMPT = "</s><s><predict_bbox><predict_classes><output_markdown>"


# ──────────────────────────────────────────────────────────────────────────
# Schemas
# ──────────────────────────────────────────────────────────────────────────

class ParseRequest(BaseModel):
    image_b64: str = Field(..., description="Base64-encoded PNG of the table region")
    page_number: int = Field(..., ge=1)


class CellPayload(BaseModel):
    row_idx: int
    col_idx: int
    text: str
    bbox_norm: list[float] | None = None
    is_header: bool = False
    confidence: float = 0.99


class ParseResponse(BaseModel):
    page_number: int
    cells: list[CellPayload]
    model_version: str
    metadata: dict = {}


# ──────────────────────────────────────────────────────────────────────────
# Model loading
# ──────────────────────────────────────────────────────────────────────────

def _load_model():
    """Load Parse v1.1-TC + the model's own postprocessing module.

    The postprocessing import requires us to add the snapshot dir to
    ``sys.path`` because HF doesn't auto-import non-modeling files when
    ``trust_remote_code=True``. We do the path tweak under the lock to
    avoid races on first call.
    """
    global _MODEL, _TOKENIZER, _PROCESSOR, _GEN_CONFIG, _POSTPROCESSING

    with _MODEL_LOCK:
        if _MODEL is not None:
            return _MODEL, _TOKENIZER, _PROCESSOR, _GEN_CONFIG, _POSTPROCESSING

        log.info("Loading %s on device=%s", _MODEL_ID, _DEVICE)
        import torch
        from transformers import (
            AutoModel,
            AutoProcessor,
            AutoTokenizer,
            GenerationConfig,
        )

        device = _DEVICE
        if device == "auto":
            device = "cuda" if torch.cuda.is_available() else "cpu"

        dtype = torch.bfloat16 if device == "cuda" else torch.float32

        _MODEL = AutoModel.from_pretrained(
            _MODEL_ID,
            trust_remote_code=True,
            torch_dtype=dtype,
        ).to(device).eval()
        _TOKENIZER = AutoTokenizer.from_pretrained(_MODEL_ID)
        _PROCESSOR = AutoProcessor.from_pretrained(_MODEL_ID, trust_remote_code=True)
        _GEN_CONFIG = GenerationConfig.from_pretrained(_MODEL_ID, trust_remote_code=True)

        _POSTPROCESSING = _try_import_postprocessing(_MODEL_ID)

        log.info("Loaded %s on %s (%s); postprocessing=%s",
                 _MODEL_ID, device, dtype, "available" if _POSTPROCESSING else "fallback-regex")
        return _MODEL, _TOKENIZER, _PROCESSOR, _GEN_CONFIG, _POSTPROCESSING


def _try_import_postprocessing(model_id: str):
    """Attempt to import the model's ``postprocessing`` module from the HF cache.

    Returns the module on success, ``None`` on failure (caller falls back
    to a hand-rolled regex parser). The cache layout is::

        $HF_HOME/hub/models--<org>--<name>/snapshots/<rev>/postprocessing.py

    We use ``huggingface_hub.snapshot_download`` purely as a cache lookup
    (it's idempotent — already-downloaded files don't re-fetch) so that
    the snapshot dir is locatable without parsing the cache layout
    ourselves.
    """
    try:
        from huggingface_hub import snapshot_download

        snap_dir = snapshot_download(model_id)
        if snap_dir not in sys.path:
            sys.path.insert(0, snap_dir)
        import importlib
        return importlib.import_module("postprocessing")
    except Exception as e:  # noqa: BLE001
        log.warning("Could not import model's postprocessing module: %s", e)
        return None


# ──────────────────────────────────────────────────────────────────────────
# Endpoints
# ──────────────────────────────────────────────────────────────────────────

@app.get("/healthz")
def healthz():
    return {"status": "ok", "model": _MODEL_ID, "loaded": _MODEL is not None}


@app.post("/debug/parse")
def debug_parse(req: ParseRequest):
    """Diagnostic endpoint — runs inference and returns the raw decoded text
    BEFORE postprocessing. Lets us write the cell parser against real model
    output instead of guessing the format from the model card.

    The full ``raw_output`` is included in the response so callers can save
    it as a fixture for parser unit tests.
    """
    try:
        model, tokenizer, processor, gen_config, postprocessing = _load_model()
    except Exception as e:  # noqa: BLE001
        raise HTTPException(status_code=503, detail=f"model load failed: {e}")
    try:
        image = _decode_image(req.image_b64)
        raw_output = _generate(model, processor, image, gen_config)
    except Exception as e:  # noqa: BLE001
        raise HTTPException(status_code=500, detail=f"inference failed: {e}")

    return {
        "page_number": req.page_number,
        "image_width": image.width,
        "image_height": image.height,
        "raw_output_chars": len(raw_output),
        "raw_output": raw_output,
        "postprocessing_available": postprocessing is not None,
        "model": _MODEL_ID,
    }


@app.post("/parse", response_model=ParseResponse)
def parse(req: ParseRequest):
    try:
        model, tokenizer, processor, gen_config, postprocessing = _load_model()
    except Exception as e:  # noqa: BLE001
        log.exception("Model load failed")
        raise HTTPException(status_code=503, detail=f"model load failed: {e}")

    try:
        image = _decode_image(req.image_b64)
    except Exception as e:  # noqa: BLE001
        raise HTTPException(status_code=400, detail=f"bad image: {e}")

    try:
        raw_output = _generate(model, processor, image, gen_config)
    except Exception as e:  # noqa: BLE001
        log.exception("Inference failed")
        raise HTTPException(status_code=500, detail=f"inference failed: {e}")

    cells = _extract_table_cells(
        raw_output, image.width, image.height, postprocessing,
    )

    return ParseResponse(
        page_number=req.page_number,
        cells=cells,
        model_version=_MODEL_ID,
        metadata={
            "device": _DEVICE,
            "raw_output_chars": len(raw_output),
            "postprocessing": "module" if postprocessing else "fallback",
        },
    )


# ──────────────────────────────────────────────────────────────────────────
# Inference
# ──────────────────────────────────────────────────────────────────────────

def _decode_image(image_b64: str) -> Image.Image:
    return Image.open(io.BytesIO(base64.b64decode(image_b64))).convert("RGB")


def _generate(model, processor, image: Image.Image, gen_config) -> str:
    """Run model.generate with the native prompt and return the decoded text."""
    import torch

    device = next(model.parameters()).device
    inputs = processor(
        images=[image],
        text=_FULL_PROMPT,
        return_tensors="pt",
        add_special_tokens=False,
    ).to(device)

    with torch.inference_mode():
        outputs = model.generate(**inputs, generation_config=gen_config)

    decoded = processor.batch_decode(outputs, skip_special_tokens=True)
    return decoded[0] if decoded else ""


# ──────────────────────────────────────────────────────────────────────────
# Output → cells
# ──────────────────────────────────────────────────────────────────────────

def _extract_table_cells(
    raw_output: str,
    image_width: int,
    image_height: int,
    postprocessing,
) -> list[CellPayload]:
    """Convert structured Parse output into table cell payloads.

    Two-step pipeline:
      1. Use the model's ``extract_classes_bboxes`` to peel off
         ``(class, bbox, text)`` triples (one per detected region).
      2. For every triple whose class is ``table``, parse the inner
         content (markdown / LaTeX / HTML — depends on prompt) into rows
         and columns.

    When the postprocessing module isn't available (offline tests) we use a
    fallback regex parser that recognises the most common token shape
    documented in the model card.
    """
    # The fallback parser is now the *primary* path because the model's
    # bundled postprocessing.py shipped with v1.1-TC operates on a different
    # output convention than what we observe in production (verified via
    # /debug/parse probe). Our regex-driven extractor matches the actual
    # ``<x_F><y_F>{body}<x_F><y_F><class_NAME>`` stream the model emits.
    if postprocessing is not None:
        try:
            classes, bboxes, texts = postprocessing.extract_classes_bboxes(raw_output)
            # postprocessing returns pixel-space bboxes; our fallback
            # returns normalised. Normalise pp output for unified handling.
            triples_pp = []
            for cls, bbox_px, text in zip(classes, bboxes, texts):
                if isinstance(bbox_px, (list, tuple)) and len(bbox_px) == 4:
                    bbox_norm = [
                        bbox_px[0] / max(image_width, 1),
                        bbox_px[1] / max(image_height, 1),
                        bbox_px[2] / max(image_width, 1),
                        bbox_px[3] / max(image_height, 1),
                    ]
                else:
                    bbox_norm = None
                triples_pp.append((cls, bbox_norm, text))
            triples = triples_pp
        except Exception as e:  # noqa: BLE001
            log.warning("postprocessing.extract_classes_bboxes failed: %s; using fallback", e)
            triples = _fallback_extract_triples(raw_output)
    else:
        triples = _fallback_extract_triples(raw_output)

    cells: list[CellPayload] = []
    for cls, bbox_norm, text in triples:
        # Class names in real Nemotron output are ``Table``, ``Caption``,
        # ``Text``, ``Section-header``, ``List-item``, ``Page-header``,
        # ``Page-footer`` (case-sensitive). We normalize-and-match to be
        # forgiving across model versions.
        if (cls or "").lower().replace("-", "_") != "table":
            continue

        # bbox_norm here is already in [0,1] image-space (the canonical
        # form sidecar consumers expect). Clamp + sanity-check.
        if bbox_norm is not None:
            bbox_norm = [max(0.0, min(1.0, float(v))) for v in bbox_norm]

        rows = _parse_table_content(text)
        if not rows:
            continue

        for r_idx, row in enumerate(rows):
            is_header = (r_idx == 0)
            for c_idx, cell_text in enumerate(row):
                cleaned = (cell_text or "").strip()
                if not cleaned and not is_header:
                    continue
                cells.append(CellPayload(
                    row_idx=r_idx,
                    col_idx=c_idx,
                    text=cleaned,
                    # Per-cell bbox isn't returned by Parse v1.1-TC — the
                    # model emits one bbox per Table region, not per cell.
                    # Each cell inherits the table-region bbox so downstream
                    # FieldProvenance still has page geometry.
                    bbox_norm=bbox_norm,
                    is_header=is_header,
                    confidence=0.99,
                ))

    return cells


def _bbox_px_to_norm(bbox, image_w: int, image_h: int) -> list[float] | None:
    """Convert a pixel-space bbox to normalised [0,1] image-space."""
    if not isinstance(bbox, (list, tuple)) or len(bbox) != 4 or image_w <= 0 or image_h <= 0:
        return None
    try:
        x0, y0, x1, y1 = (float(v) for v in bbox)
    except (TypeError, ValueError):
        return None
    return [
        max(0.0, min(1.0, x0 / image_w)),
        max(0.0, min(1.0, y0 / image_h)),
        max(0.0, min(1.0, x1 / image_w)),
        max(0.0, min(1.0, y1 / image_h)),
    ]


# ──────────────────────────────────────────────────────────────────────────
# Fallback parsers (used when the model's postprocessing module is absent)
# ──────────────────────────────────────────────────────────────────────────

# Real Nemotron Parse v1.1-TC output format (verified by /debug/parse probe
# on KDIGO page 3, captured to tests/v5/golden/sidecar_samples/nemotron_*.json).
#
# Each element is emitted as:
#     <x_F1><y_F1>{content}<x_F2><y_F2><class_NAME>
#
# Where (x,y) coords are NORMALISED to [0,1] image-space (NOT pixel coords),
# the first pair is the top-left corner and the second pair is the
# bottom-right corner. ``content`` is the rendered text — for tables it's
# inline LaTeX (``\begin{tabular}...\end{tabular}``), for figures it's a
# caption, for body text it's plain markdown. Class names observed:
# ``Table``, ``Caption``, ``Text``, ``Section-header``, ``List-item``,
# ``Page-header``, ``Page-footer``, ``Title``, ``Picture`` — case-sensitive.
#
# The previous regex (``<bbox_DDD_DDD>`` style) was incorrect — that format
# was a guess from the model card's example, not the real output. After
# capturing real output via the probe, we rewrote against actual data.
_NEMOTRON_ELEMENT_RE = re.compile(
    r"<x_(?P<x0>\d*\.?\d+)>"
    r"<y_(?P<y0>\d*\.?\d+)>"
    r"(?P<body>.*?)"
    r"<x_(?P<x1>\d*\.?\d+)>"
    r"<y_(?P<y1>\d*\.?\d+)>"
    r"<class_(?P<cls>[A-Za-z][A-Za-z0-9_-]*)>",
    re.DOTALL,
)


def _fallback_extract_triples(raw_output: str) -> list[tuple[str, list[float] | None, str]]:
    """Parse Nemotron Parse v1.1-TC's element stream into (class, bbox, body) triples.

    bbox is returned in NORMALISED [0,1] image coords — caller scales to
    pixel coords via image_width / image_height. We keep the float
    representation rather than rounding to ints because table-cell bboxes
    use 3-4 decimal places of precision.
    """
    triples: list[tuple[str, list[float] | None, str]] = []
    for m in _NEMOTRON_ELEMENT_RE.finditer(raw_output):
        try:
            bbox_norm = [
                float(m.group("x0")),
                float(m.group("y0")),
                float(m.group("x1")),
                float(m.group("y1")),
            ]
        except (TypeError, ValueError):
            bbox_norm = None
        triples.append((m.group("cls"), bbox_norm, m.group("body")))
    return triples


def _parse_table_content(content: str) -> list[list[str]]:
    """Convert Nemotron Parse v1.1-TC table content into rows.

    The probe (tests/v5/golden/sidecar_samples/nemotron_parse_p3_*.json)
    confirmed that v1.1-TC tables come back as inline LaTeX tabular
    blocks like::

        \\begin{tabular}{ccc}
        Antihyperglycemic agents & Risk of hypoglycemia & Rationale for CGM or SMBG \\\\
        • Insulin · Sulfonylureas · Meglitinides & Higher & Higher \\\\
        • Metformin · SGLT2 inhibitors · GLP-1 receptor agonists & Lower & Lower \\\\
        \\end{tabular}

    Each row has columns separated by ``&``, rows by ``\\\\``. Cells can
    contain bullet lists (``·`` or ``\\bullet``) without breaking the
    tabular structure. We try LaTeX first because it's the verified
    output format; markdown-pipe is a fallback in case v1.2+ changes.

    Returns ``list[list[str]]`` — caller treats row 0 as header.
    """
    if not content:
        return []

    tex_rows = _parse_latex_tabular(content)
    if tex_rows:
        return tex_rows

    md_rows = _parse_markdown_pipe_table(content)
    if md_rows:
        return md_rows

    return []


def _parse_markdown_pipe_table(content: str) -> list[list[str]]:
    """Parse a pipe-table body. Skips alignment rows like ``|---|---|``."""
    rows: list[list[str]] = []
    for raw_line in content.split("\n"):
        line = raw_line.strip()
        if not line.startswith("|") and not line.endswith("|"):
            continue
        # Strip leading / trailing pipe so split doesn't produce empty edges.
        line = line.strip("|")
        # Alignment row check: cells contain only --- and :
        cells = [c.strip() for c in line.split("|")]
        if all(set(c) <= set("-:") and c for c in cells):
            continue
        rows.append(cells)
    # Need at least one row with content to count as a table.
    return rows if any(any(c for c in row) for row in rows) else []


_TABULAR_RE = re.compile(
    r"\\begin\{tabular\}.*?\}(.*?)\\end\{tabular\}",
    re.DOTALL,
)


def _parse_latex_tabular(content: str) -> list[list[str]]:
    """Parse a ``tabular`` environment. Splits rows on ``\\\\``, cells on ``&``.

    Strips ``\\hline`` and ``\\multirow``/``\\multicolumn`` wrappers but
    preserves the cell text inside.
    """
    m = _TABULAR_RE.search(content)
    body = m.group(1) if m else content

    rows: list[list[str]] = []
    for row_str in re.split(r"\\\\", body):
        row_clean = row_str.replace(r"\hline", "").strip()
        if not row_clean:
            continue
        cells = [_strip_latex(cell).strip() for cell in row_clean.split("&")]
        rows.append(cells)
    return rows if any(any(c for c in row) for row in rows) else []


_LATEX_WRAPPERS = re.compile(r"\\(multirow|multicolumn|cell)\s*\{[^}]*\}\s*\{[^}]*\}\s*\{?")


def _strip_latex(text: str) -> str:
    """Remove the most common LaTeX wrappers; preserve inner text."""
    text = _LATEX_WRAPPERS.sub("", text)
    text = text.replace("{", "").replace("}", "")
    return text.strip()
