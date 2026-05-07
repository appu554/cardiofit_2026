#!/usr/bin/env python3
"""Probe each V5 sidecar with a real PDF page → dump raw model output.

Why this script exists
----------------------
The earlier smoke runs proved both sidecars *work* end-to-end (load weights,
accept POSTs, return 200 OK), but their parsers were guessing the model
output format from the model cards. This script captures the *real* output
strings from each model so we can write parsers against actual data.

Usage::

    python probe.py [pdf-path] [page-number]

Without args, defaults to KDIGO page 3 (richest content — Figure 13 drug
table + Practice Point 2.1.5 + Recommendation 2.2.1).

Output:
    sidecar_samples/nemotron_parse_p<N>_<ts>.json
    sidecar_samples/lightonocr_p<N>_<ts>.json
"""
from __future__ import annotations

import base64
import json
import sys
import time
from pathlib import Path

import fitz  # PyMuPDF (host)
import requests

DEFAULT_PDF = Path(
    "/Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/"
    "knowledge-base-services/shared/tools/guideline-atomiser/data/pdfs/"
    "KDIGO-2022-full-guide-pages-58-61.pdf"
)
DEFAULT_PAGE = 3   # 1-indexed; the drug-vs-hypoglycemia matrix page

OUT_DIR = Path(__file__).parent
NEMOTRON_URL = "http://localhost:8503"
LIGHTONOCR_URL = "http://localhost:8501"

# Render at 200 DPI (LightOnOCR's recommended) — Nemotron also handles this size.
RENDER_DPI = 200


def render_page_b64(pdf_path: Path, page_no: int) -> tuple[str, int, int]:
    """Render one page to PNG, base64-encode, return (b64, width_px, height_px)."""
    doc = fitz.open(str(pdf_path))
    page = doc[page_no - 1]
    scale = RENDER_DPI / 72.0
    mat = fitz.Matrix(scale, scale)
    pix = page.get_pixmap(matrix=mat, alpha=False)
    png_bytes = pix.tobytes("png")
    doc.close()
    return base64.b64encode(png_bytes).decode("ascii"), pix.width, pix.height


def probe_nemotron(image_b64: str, page_no: int):
    print(f"\n=== Nemotron Parse v1.1-TC — page {page_no} ===")
    t0 = time.monotonic()
    resp = requests.post(
        f"{NEMOTRON_URL}/debug/parse",
        json={"image_b64": image_b64, "page_number": page_no},
        timeout=3600,
    )
    elapsed = time.monotonic() - t0
    print(f"  HTTP {resp.status_code}, elapsed={elapsed:.1f}s")
    if resp.status_code != 200:
        print(f"  body: {resp.text[:500]}")
        return None
    data = resp.json()
    print(f"  raw_output_chars: {data.get('raw_output_chars')}")
    print(f"  postprocessing_available: {data.get('postprocessing_available')}")
    print(f"  image WxH: {data.get('image_width')}x{data.get('image_height')}")
    raw = data.get("raw_output", "")
    print(f"  first 600 chars: {raw[:600]!r}")
    print(f"  last  300 chars: {raw[-300:]!r}")
    return data


def probe_lightonocr(image_b64: str, page_no: int):
    print(f"\n=== LightOnOCR-2-1B-bbox — page {page_no} ===")
    t0 = time.monotonic()
    resp = requests.post(
        f"{LIGHTONOCR_URL}/debug/ocr",
        json={
            "image_b64": image_b64,
            "page_number": page_no,
            "bbox": True,
            "render_dpi": RENDER_DPI,
        },
        timeout=3600,
    )
    elapsed = time.monotonic() - t0
    print(f"  HTTP {resp.status_code}, elapsed={elapsed:.1f}s")
    if resp.status_code != 200:
        print(f"  body: {resp.text[:500]}")
        return None
    data = resp.json()
    print(f"  raw_token_count: {data.get('raw_token_count')}")
    print(f"  has_box_start_token: {data.get('has_box_start_token')}")
    print(f"  has_box_end_token:   {data.get('has_box_end_token')}")
    print(f"  text_with_boxes_chars: {data.get('text_with_boxes_chars')}")
    print(f"  text_clean_chars:      {data.get('text_clean_chars')}")
    twb = data.get("text_with_boxes", "")
    print(f"  text_with_boxes first 600: {twb[:600]!r}")
    print(f"  text_with_boxes last  300: {twb[-300:]!r}")
    return data


def main(argv):
    pdf_path = Path(argv[1]) if len(argv) > 1 else DEFAULT_PDF
    page_no = int(argv[2]) if len(argv) > 2 else DEFAULT_PAGE

    print(f"Source PDF: {pdf_path.name} (page {page_no})")
    print(f"Rendering at {RENDER_DPI} DPI…")
    image_b64, w, h = render_page_b64(pdf_path, page_no)
    print(f"  image: {w}x{h} px, {len(image_b64)/1024:.1f} KB base64")

    ts = int(time.time())

    nemo = probe_nemotron(image_b64, page_no)
    if nemo is not None:
        out = OUT_DIR / f"nemotron_parse_p{page_no}_{ts}.json"
        # Drop the giant base64 from the saved sample to keep file size sane;
        # we save the PROBE INPUT separately.
        out.write_text(json.dumps(nemo, indent=2))
        print(f"  saved: {out.relative_to(OUT_DIR.parent.parent.parent.parent)}")

    light = probe_lightonocr(image_b64, page_no)
    if light is not None:
        out = OUT_DIR / f"lightonocr_p{page_no}_{ts}.json"
        out.write_text(json.dumps(light, indent=2))
        print(f"  saved: {out.relative_to(OUT_DIR.parent.parent.parent.parent)}")

    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
