"""Nemotron Nano VL 8B FastAPI server.

Implements the contract documented in
``extraction/v4/specialists/nemotron_nano_vl.py`` — accepts a base64 PNG of
a figure region and returns ``{description, facts, confidence, model_version}``.

The ``trust_remote_code=True`` model uses NVIDIA's custom CRADIOv2 vision
encoder; the model card recipe is reproduced here verbatim (including the
custom ``model.chat()`` interface, which is NOT the standard
``model.generate()``).
"""
from __future__ import annotations

import base64
import io
import json
import logging
import os
import re
from threading import Lock

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field
from PIL import Image

log = logging.getLogger("nemotron_vl_server")
logging.basicConfig(level=logging.INFO)

app = FastAPI(title="Nemotron Nano VL 8B sidecar", version="0.1.0")

_MODEL = None
_TOKENIZER = None
_IMAGE_PROCESSOR = None
_MODEL_LOCK = Lock()
_MODEL_ID = os.environ.get(
    "NEMOTRON_VL_MODEL", "nvidia/Llama-3.1-Nemotron-Nano-VL-8B-V1"
)
_DEVICE = os.environ.get("NEMOTRON_VL_DEVICE", "auto")


# ──────────────────────────────────────────────────────────────────────────
# Prompt templates
# ──────────────────────────────────────────────────────────────────────────

# extract_facts: structured JSON response designed for ingestion into the
# atomiser. Asks the model for a confidence so the caller can gate
# escalation to Gemini 2.5 Pro at confidence < threshold.
_PROMPT_EXTRACT_FACTS = (
    "This figure is from a clinical practice guideline. Extract every "
    "discrete clinical fact / recommendation it conveys. Return JSON only:\n"
    "{\n"
    '  "description": "<one-sentence description of the figure>",\n'
    '  "facts": ["<fact 1>", "<fact 2>", ...],\n'
    '  "confidence": <float 0-1 reflecting how confident you are the '
    "extraction is complete and correct>\n"
    "}\n"
    "Rules:\n"
    "- Each fact must be a single decision rule, threshold, or "
    "recommendation — split compound statements.\n"
    "- Preserve numbers verbatim including units.\n"
    "- Confidence < 0.7 means the figure is too complex / ambiguous; "
    "say so and lower confidence accordingly.\n"
    "- Output JSON only — no prose, no markdown fence."
)

_PROMPT_DESCRIBE = (
    "Describe this image from a clinical guideline in 1–3 sentences. "
    "Mention drug names, thresholds and decision rules verbatim if any. "
    "Plain text only."
)


class FigureRequest(BaseModel):
    image_b64: str = Field(..., description="Base64-encoded PNG of the figure region")
    task: str = Field("extract_facts", description="extract_facts | describe")
    max_tokens: int = Field(1024, gt=0, le=4096)


class FigureResponse(BaseModel):
    description: str
    facts: list[str] = []
    confidence: float = 0.0
    model_version: str


# ──────────────────────────────────────────────────────────────────────────
# Model loading
# ──────────────────────────────────────────────────────────────────────────

def _load_model():
    """Load Nemotron Nano VL once and cache it.

    Note: Nano VL uses ``model.chat()``, NOT ``model.generate()`` — the
    chat method is provided by the trust_remote_code modelling file.
    """
    global _MODEL, _TOKENIZER, _IMAGE_PROCESSOR
    with _MODEL_LOCK:
        if _MODEL is not None:
            return _MODEL, _TOKENIZER, _IMAGE_PROCESSOR

        log.info("Loading %s on device=%s", _MODEL_ID, _DEVICE)
        import torch
        from transformers import AutoImageProcessor, AutoModel, AutoTokenizer

        device = _DEVICE
        if device == "auto":
            device = "cuda" if torch.cuda.is_available() else "cpu"

        kw = dict(trust_remote_code=True)
        if device == "cuda":
            kw["device_map"] = "cuda"
            kw["torch_dtype"] = torch.bfloat16
        else:
            kw["torch_dtype"] = torch.float32

        _MODEL = AutoModel.from_pretrained(_MODEL_ID, **kw).eval()
        _TOKENIZER = AutoTokenizer.from_pretrained(_MODEL_ID)
        _IMAGE_PROCESSOR = AutoImageProcessor.from_pretrained(
            _MODEL_ID, trust_remote_code=True, device=device,
        )
        log.info("Model loaded: %s on %s", _MODEL_ID, device)
        return _MODEL, _TOKENIZER, _IMAGE_PROCESSOR


# ──────────────────────────────────────────────────────────────────────────
# Endpoints
# ──────────────────────────────────────────────────────────────────────────

@app.get("/healthz")
def healthz():
    return {"status": "ok", "model": _MODEL_ID, "loaded": _MODEL is not None}


@app.post("/figure", response_model=FigureResponse)
def figure(req: FigureRequest):
    try:
        model, tokenizer, image_processor = _load_model()
    except Exception as e:  # noqa: BLE001
        log.exception("Model load failed")
        raise HTTPException(status_code=503, detail=f"model load failed: {e}")

    try:
        image_bytes = base64.b64decode(req.image_b64)
        image = Image.open(io.BytesIO(image_bytes)).convert("RGB")
    except Exception as e:  # noqa: BLE001
        raise HTTPException(status_code=400, detail=f"bad image: {e}")

    if req.task == "extract_facts":
        prompt = _PROMPT_EXTRACT_FACTS
    elif req.task == "describe":
        prompt = _PROMPT_DESCRIBE
    else:
        raise HTTPException(status_code=400, detail=f"unknown task: {req.task}")

    try:
        raw = _run_chat(model, tokenizer, image_processor, image, prompt, req.max_tokens)
    except Exception as e:  # noqa: BLE001
        log.exception("Inference failed")
        raise HTTPException(status_code=500, detail=f"inference failed: {e}")

    description, facts, confidence = _parse_response(raw, req.task)
    return FigureResponse(
        description=description,
        facts=facts,
        confidence=confidence,
        model_version=_MODEL_ID,
    )


# ──────────────────────────────────────────────────────────────────────────
# Inference helpers
# ──────────────────────────────────────────────────────────────────────────

def _run_chat(model, tokenizer, image_processor, image, question: str, max_tokens: int) -> str:
    """Run Nemotron Nano VL using its custom ``model.chat()`` interface."""
    image_features = image_processor([image])
    generation_config = dict(
        max_new_tokens=max_tokens,
        do_sample=False,
        eos_token_id=tokenizer.eos_token_id,
    )
    return model.chat(
        tokenizer=tokenizer,
        question=question,
        generation_config=generation_config,
        **image_features,
    )


_FENCE_RE = re.compile(r"^```(?:json|JSON)?\s*\n?|\n?```\s*$", re.MULTILINE)


def _parse_response(raw: str, task: str):
    """Convert model output to ``(description, facts, confidence)``.

    For ``extract_facts`` we expect JSON; for ``describe`` we expect plain text.
    Defensive: any parse failure returns the raw string as the description
    with empty facts and confidence=0.5 — better than a 500.
    """
    if task == "describe":
        return raw.strip(), [], 0.9

    # extract_facts → JSON
    cleaned = _FENCE_RE.sub("", raw).strip()
    try:
        obj = json.loads(cleaned)
    except json.JSONDecodeError:
        # Try extracting the first {...} block.
        start = cleaned.find("{")
        end = cleaned.rfind("}")
        if start >= 0 and end > start:
            try:
                obj = json.loads(cleaned[start: end + 1])
            except json.JSONDecodeError:
                obj = None
        else:
            obj = None

    if not isinstance(obj, dict):
        return cleaned, [], 0.5

    description = str(obj.get("description", ""))
    facts = obj.get("facts") or []
    if not isinstance(facts, list):
        facts = []
    facts = [str(f) for f in facts if isinstance(f, (str, int, float))]
    try:
        conf = float(obj.get("confidence", 0.5))
    except (TypeError, ValueError):
        conf = 0.5
    conf = max(0.0, min(1.0, conf))

    return description, facts, conf
