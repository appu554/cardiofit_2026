"""V5 lane-specialist models.

Each lane consumes a *page region* (table / figure / body text) and returns a
structured result that the surrounding V4 channels can stitch into RawSpans.

Lanes
-----
- table  : nvidia/nemo-document-parse-1.1   (NVIDIA NIM API; cloud)
- figure : nvidia/Llama-3.1-Nemotron-Nano-VL-8B-v1   (HF; sidecar HTTP)
- ocr    : lightonai/LightOnOCR-2-1B-bbox   (HF; sidecar HTTP)

Why sidecars for figure + OCR
-----------------------------
The pipeline image force-pins ``transformers==4.51.0`` because magic-pdf,
docling-ibm-models and gliner all argue over that exact version. The newer
HuggingFace specialists (especially LightOnOCR's ``LightOnOcrForConditionalGeneration``,
which lands in transformers ≥4.55) cannot share that environment, so they run
as separate containers behind a thin HTTP boundary — same pattern used for
Ollama / NuExtract (Channel F).

The Nemotron Parse table lane is cloud-only (NVIDIA NIM has no public
weights), so it makes a direct HTTPS call from the main pipeline image and
needs no sidecar.
"""
from .base import (
    OCRBlockResult,
    OCRSpecialistResult,
    SpecialistError,
    SpecialistTimeoutError,
    SpecialistUnavailableError,
    TableCellResult,
    TableSpecialistResult,
)

__all__ = [
    "OCRBlockResult",
    "OCRSpecialistResult",
    "SpecialistError",
    "SpecialistTimeoutError",
    "SpecialistUnavailableError",
    "TableCellResult",
    "TableSpecialistResult",
]
