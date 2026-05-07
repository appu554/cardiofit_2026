"""LightOnOCR-2-1B-bbox FastAPI server.

KNOWN LIMITATION (verified 2026-05-06 via /debug/ocr probe):
The default ``apply_chat_template`` invocation for this model returns clean
markdown only — *no* ``<|box_start|>``/``<|box_end|>`` tokens are emitted,
even though the tokenizer defines them (IDs 151648/151649). The bbox-mode
prompt template is not documented in the model card; reaching it likely
requires either:

  1. A specific instruction prefix (e.g. ``"With bounding boxes:"``) that
     the bbox-finetune was trained on.
  2. The ``apply_chat_template`` ``bbox=True`` extra-context flag if such
     a thing exists in transformers' chat_template DSL.
  3. An undocumented system-prompt pattern reverse-engineered from the
     LightOnOCR training data.

Until that's resolved, this sidecar serves clean OCR markdown only —
which is still useful as an alternative L1 source the orchestrator can
diff against MonkeyOCR. The captured probe sample at
``tests/v5/golden/sidecar_samples/lightonocr_p3_*.json`` records this
state so future re-evaluation has a reference.

The parser DOES correctly handle bbox tokens when present (see
``_parse_bbox_segments``); the integration is wired and tested. The gap
is purely on the input-prompt side.


Implements the contract documented in
``extraction/v4/specialists/lightonocr.py`` — accepts a base64 PNG and
returns ``{markdown, blocks: [{text, bbox, confidence, word_spans}], ...}``
with bboxes in **PDF points** (page-relative, top-left origin).

Lifecycle: model weights load lazily on first request and are pinned for
the lifetime of the process. The pinned model object lets request handlers
share a single GPU allocation without re-paging weights between calls.

Bbox conversion contract
------------------------
The model returns coordinates in *image pixel space* (relative to the
rendered PNG). The client tells us the page DPI it used, so we convert
back to PDF points: ``pt = px * 72 / dpi``. We also emit page_number on
each block so client-side coordinate joining is unambiguous.
"""
from __future__ import annotations

import base64
import io
import logging
import os
from threading import Lock
from typing import Optional

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field
from PIL import Image

log = logging.getLogger("lightonocr_server")
logging.basicConfig(level=logging.INFO)

app = FastAPI(title="LightOnOCR-2-1B-bbox sidecar", version="0.1.0")

# Model registry — populated on first request. Lock guards the populate so
# two concurrent first-requests don't both try to load weights.
_MODEL = None
_PROCESSOR = None
_MODEL_LOCK = Lock()
_MODEL_ID = os.environ.get("LIGHTONOCR_MODEL", "lightonai/LightOnOCR-2-1B-bbox")
_DEVICE = os.environ.get("LIGHTONOCR_DEVICE", "auto")  # auto | cuda | cpu


class OCRRequest(BaseModel):
    image_b64: str = Field(..., description="Base64-encoded PNG of one PDF page")
    page_number: int = Field(..., ge=1, description="1-indexed PDF page number")
    bbox: bool = Field(True, description="When true, request the bbox variant's word-level coords")
    # Render DPI used by the client — needed to convert px → PDF pt.
    render_dpi: float = Field(200.0, gt=0, description="Pixels-per-PDF-inch the client used to render")


class OCRBlock(BaseModel):
    text: str
    bbox: Optional[list[float]] = None
    confidence: float = 1.0
    page_number: int
    word_spans: list[dict] = []


class OCRResponse(BaseModel):
    page_number: int
    markdown: str
    blocks: list[OCRBlock]
    model_version: str
    metadata: dict = {}


# ──────────────────────────────────────────────────────────────────────────
# Model loading
# ──────────────────────────────────────────────────────────────────────────

def _load_model():
    """Load LightOnOCR-2-1B-bbox once and cache it.

    Raises 503 (via the caller's HTTPException) if loading fails — better to
    surface the error than silently fall back to a stub that returns empty
    results.
    """
    global _MODEL, _PROCESSOR
    with _MODEL_LOCK:
        if _MODEL is not None:
            return _MODEL, _PROCESSOR
        log.info("Loading %s on device=%s", _MODEL_ID, _DEVICE)
        import torch
        from transformers import (
            LightOnOcrForConditionalGeneration,
            LightOnOcrProcessor,
        )

        device = _DEVICE
        if device == "auto":
            if torch.cuda.is_available():
                device = "cuda"
            elif torch.backends.mps.is_available():
                device = "mps"
            else:
                device = "cpu"

        # bfloat16 on GPU; fp32 on CPU/MPS where bfloat16 is iffy.
        dtype = torch.bfloat16 if device == "cuda" else torch.float32

        _MODEL = LightOnOcrForConditionalGeneration.from_pretrained(
            _MODEL_ID, torch_dtype=dtype,
        ).to(device)
        _MODEL.eval()
        _PROCESSOR = LightOnOcrProcessor.from_pretrained(_MODEL_ID)
        log.info("Model loaded: %s on %s (%s)", _MODEL_ID, device, dtype)
        return _MODEL, _PROCESSOR


# ──────────────────────────────────────────────────────────────────────────
# Endpoints
# ──────────────────────────────────────────────────────────────────────────

@app.get("/healthz")
def healthz():
    """Readiness probe. ``loaded=true`` once weights are in memory."""
    return {"status": "ok", "model": _MODEL_ID, "loaded": _MODEL is not None}


@app.post("/debug/ocr")
def debug_ocr(req: OCRRequest):
    """Diagnostic endpoint — returns raw decoded text WITH special tokens
    visible. Lets us see whether ``<|box_start|>`` / ``<|box_end|>`` tokens
    are being emitted and capture sample output to drive parser writing.

    Includes ``text_clean`` (skip_special_tokens=True) as a side-by-side
    comparison so we can confirm tokens are stripped vs simply absent.
    """
    try:
        model, processor = _load_model()
    except Exception as e:  # noqa: BLE001
        raise HTTPException(status_code=503, detail=f"model load failed: {e}")
    try:
        image_bytes = base64.b64decode(req.image_b64)
        image = Image.open(io.BytesIO(image_bytes)).convert("RGB")
    except Exception as e:  # noqa: BLE001
        raise HTTPException(status_code=400, detail=f"bad image: {e}")

    import torch
    device = next(model.parameters()).device
    dtype = next(model.parameters()).dtype
    conversation = [{"role": "user", "content": [{"type": "image", "image": image}]}]
    inputs = processor.apply_chat_template(
        conversation, add_generation_prompt=True, tokenize=True,
        return_dict=True, return_tensors="pt",
    )
    inputs = {
        k: (v.to(device=device, dtype=dtype) if v.is_floating_point() else v.to(device))
        for k, v in inputs.items()
    }
    with torch.inference_mode():
        out = model.generate(**inputs, max_new_tokens=2048, do_sample=False)
    gen = out[0, inputs["input_ids"].shape[1]:]

    text_with_boxes = processor.decode(gen, skip_special_tokens=False)
    text_clean = processor.decode(gen, skip_special_tokens=True)

    return {
        "page_number": req.page_number,
        "image_width": image.width,
        "image_height": image.height,
        "raw_token_count": int(gen.shape[0]),
        "text_with_boxes_chars": len(text_with_boxes),
        "text_clean_chars": len(text_clean),
        "has_box_start_token": "<|box_start|>" in text_with_boxes,
        "has_box_end_token": "<|box_end|>" in text_with_boxes,
        # Truncate to keep response reasonable but include enough to identify pattern
        "text_with_boxes": text_with_boxes[:8000],
        "text_clean": text_clean[:2000],
        "model": _MODEL_ID,
    }


@app.post("/ocr", response_model=OCRResponse)
def ocr(req: OCRRequest):
    try:
        model, processor = _load_model()
    except Exception as e:  # noqa: BLE001 — surface any load error to client
        log.exception("Model load failed")
        raise HTTPException(status_code=503, detail=f"model load failed: {e}")

    try:
        image_bytes = base64.b64decode(req.image_b64)
        image = Image.open(io.BytesIO(image_bytes)).convert("RGB")
    except Exception as e:  # noqa: BLE001
        raise HTTPException(status_code=400, detail=f"bad image: {e}")

    try:
        markdown, blocks = _run_inference(model, processor, image, req)
    except Exception as e:  # noqa: BLE001
        log.exception("Inference failed")
        raise HTTPException(status_code=500, detail=f"inference failed: {e}")

    return OCRResponse(
        page_number=req.page_number,
        markdown=markdown,
        blocks=blocks,
        model_version=_MODEL_ID,
        metadata={"render_dpi": req.render_dpi, "device": _DEVICE},
    )


# ──────────────────────────────────────────────────────────────────────────
# Inference
# ──────────────────────────────────────────────────────────────────────────

def _run_inference(model, processor, image: Image.Image, req: OCRRequest):
    """Run LightOnOCR and convert the structured output into our response shape.

    Returns ``(markdown, [OCRBlock, ...])``.

    Pixel-to-point conversion: the bbox variant emits coords in pixel space
    of the input image. We divide by ``req.render_dpi / 72`` to land back in
    PDF points (page-relative, top-left origin). When the client renders at
    200 DPI, that's a divisor of ~2.78.
    """
    import torch

    device = next(model.parameters()).device
    dtype = next(model.parameters()).dtype

    conversation = [{
        "role": "user",
        "content": [{"type": "image", "image": image}],
    }]
    inputs = processor.apply_chat_template(
        conversation,
        add_generation_prompt=True,
        tokenize=True,
        return_dict=True,
        return_tensors="pt",
    )
    inputs = {
        k: (v.to(device=device, dtype=dtype) if v.is_floating_point() else v.to(device))
        for k, v in inputs.items()
    }

    with torch.inference_mode():
        output_ids = model.generate(
            **inputs,
            max_new_tokens=2048,
            do_sample=False,
        )

    generated = output_ids[0, inputs["input_ids"].shape[1]:]
    # Keep the <|box_start|>/<|box_end|> special tokens visible in the
    # decoded text so we can parse per-segment bboxes. The "bbox" variant
    # of LightOnOCR-2-1B emits these around each grounded text segment;
    # decoding with skip_special_tokens=True (the original code) stripped
    # them, leaving only flat markdown — which is why the previous smoke
    # produced 0 word_spans.
    text_with_boxes = processor.decode(generated, skip_special_tokens=False)

    # Two output streams from the same generation:
    #   1. ``markdown_clean``: the model's text without box tokens — the
    #      reading-order body text we feed to ``normalized_text``.
    #   2. ``blocks_with_geometry``: per-segment text + bbox tuples
    #      derived from the box tokens. Used to populate FieldProvenance.
    px_to_pt = 72.0 / req.render_dpi
    image_w_px = image.width
    image_h_px = image.height
    markdown_clean = processor.decode(generated, skip_special_tokens=True)

    # Parse the bbox-tagged text into (text, bbox_pixels) tuples.
    segments = _parse_bbox_segments(text_with_boxes, image_w_px, image_h_px)

    if segments:
        # Each segment becomes one OCRBlock with a single word_span (the
        # segment text itself). The model emits per-line / per-region
        # bboxes, not per-word — so word_spans is really "segment_spans".
        # We keep the field name for compatibility with the orchestrator's
        # eventual char-offset → bbox alignment work.
        blocks = []
        for seg_text, bbox_px in segments:
            seg_text = seg_text.strip()
            if not seg_text:
                continue
            bbox_pt = _scale_bbox(list(bbox_px), px_to_pt) if bbox_px else None
            blocks.append(OCRBlock(
                text=seg_text,
                bbox=bbox_pt,
                confidence=1.0,
                page_number=req.page_number,
                word_spans=[{
                    "text": seg_text,
                    "bbox": bbox_pt,
                    "confidence": 1.0,
                }] if bbox_pt else [],
            ))
        return markdown_clean, blocks

    # Fallback: model returned no bbox tokens (perhaps the prompt didn't
    # request them, or this is the non-bbox variant). Return the markdown
    # as a single block without geometry — same as the original behaviour.
    log.warning(
        "lightonocr: no <|box_start|>...<|box_end|> tokens found in output "
        "(len=%d). Returning markdown without word_spans.",
        len(text_with_boxes),
    )
    return markdown_clean, [OCRBlock(
        text=markdown_clean,
        bbox=None,
        confidence=1.0,
        page_number=req.page_number,
        word_spans=[],
    )]


def _scale_bbox(bbox, scale: float):
    """Scale a 4-tuple bbox; returns None on missing/malformed input."""
    if not isinstance(bbox, (list, tuple)) or len(bbox) != 4:
        return None
    try:
        return [float(v) * scale for v in bbox]
    except (TypeError, ValueError):
        return None


# LightOnOCR-2-1B-bbox emits each grounded text region wrapped in
# ``<|box_start|>...<|box_end|>`` tokens. Inside the wrapper is a
# space-or-comma-separated 4-tuple of integers — pixel coords on the input
# image at the resolution the processor saw. The text *preceding* a
# box-start (back to the prior box-end or the start of stream) is the
# content of that segment. We use a non-greedy regex that captures both
# halves in one sweep.
import re as _re
_BOX_RE = _re.compile(
    r"<\|box_start\|>"
    r"\s*(?P<coords>[-\d.,\s()]+?)"
    r"\s*<\|box_end\|>",
    _re.DOTALL,
)


def _parse_bbox_segments(text_with_boxes: str, image_w: int, image_h: int):
    """Walk the decoded output, pairing each preceding text run with the
    box that follows it.

    Returns a list of ``(text, bbox_pixels)`` tuples. ``bbox_pixels`` is a
    4-tuple of ``int`` (x0, y0, x1, y1) on the *image* coordinate frame —
    the caller scales to PDF points using the render DPI it knows.

    The model emits coords in several plausible formats — Qwen-style
    ``(x1,y1),(x2,y2)``, plain ``x1 y1 x2 y2``, or ``x1,y1,x2,y2``. We
    extract all integer-like numbers and take the first four; this is
    forgiving but safe because the special tokens already bracket the
    coord block, so spurious numbers from the surrounding text can't
    sneak in.

    Returns ``[]`` when no box tokens are found — caller falls back to
    "whole page as one block".
    """
    segments: list[tuple[str, tuple[int, int, int, int] | None]] = []
    cursor = 0
    for m in _BOX_RE.finditer(text_with_boxes):
        # Text BEFORE this box → segment content
        seg_text = text_with_boxes[cursor: m.start()]
        coords_raw = m.group("coords")
        nums = _re.findall(r"-?\d+(?:\.\d+)?", coords_raw)
        bbox: tuple[int, int, int, int] | None = None
        if len(nums) >= 4:
            try:
                x0, y0, x1, y1 = (int(float(v)) for v in nums[:4])
                # Some variants emit normalised 0-1 coords; detect that
                # and rescale by the input image dimensions. Threshold of
                # 1.5 is safe — pixel coords are always ≥ 0 and bigger
                # than that on any realistic page.
                if max(x0, y0, x1, y1) <= 1:
                    x0 = int(x0 * image_w)
                    y0 = int(y0 * image_h)
                    x1 = int(x1 * image_w)
                    y1 = int(y1 * image_h)
                # Clamp to image bounds; the model occasionally drifts a
                # pixel or two off the edge.
                x0 = max(0, min(image_w, x0))
                y0 = max(0, min(image_h, y0))
                x1 = max(0, min(image_w, x1))
                y1 = max(0, min(image_h, y1))
                if x1 > x0 and y1 > y0:
                    bbox = (x0, y0, x1, y1)
            except (TypeError, ValueError):
                bbox = None
        if seg_text.strip():
            segments.append((seg_text, bbox))
        cursor = m.end()

    # Trailing text after the last box (no geometry).
    tail = text_with_boxes[cursor:]
    if tail.strip():
        segments.append((tail, None))

    return segments
