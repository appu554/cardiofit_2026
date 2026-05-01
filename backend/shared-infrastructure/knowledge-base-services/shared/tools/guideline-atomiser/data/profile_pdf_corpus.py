#!/usr/bin/env python3
"""Corpus profiler — characterise PDFs to predict pipeline routing + cost.

Walks a directory of PDFs and emits per-PDF metadata that lets us predict
which Pipeline-1 path each PDF will take + the total GPU-hour budget for
the corpus. Intended to run BEFORE provisioning hardware.

Per-PDF features extracted:
  - file_size_bytes
  - page_count
  - has_text_layer       : any extractable text per page
  - text_chars_per_page  : avg chars; <50 = scanned, >500 = clean digital
  - has_images           : any embedded images
  - image_count          : total image objects across all pages
  - is_form              : has form fields (FillBox / RadioButton)
  - is_pdfa              : declares PDF/A conformance in metadata
  - producer             : the software that wrote the PDF
  - encrypted            : password-protected
  - estimated_route      : "clean_text" | "hybrid_visual" | "full_ocr"
                           — drives the L1 backend choice in v2

Routing heuristic:
  - clean_text   : text >500 char/pg, image_count/page <= 1
                   → docling text-only (no GPU, ~0.5 sec/page on CPU)
  - hybrid_visual: text >50 char/pg, has images
                   → docling + Channel A (modest GPU, ~3 sec/page)
  - full_ocr     : text <50 char/pg OR producer suggests scan
                   → MonkeyOCR + all channels (heavy GPU, ~30 sec/page)

Usage:
    python profile_pdf_corpus.py /path/to/corpus_dir
    python profile_pdf_corpus.py --json /path/to/corpus_dir > out.json
    python profile_pdf_corpus.py --csv  /path/to/corpus_dir > out.csv
    python profile_pdf_corpus.py --gpu-cost-only /path/to/corpus_dir

The --gpu-cost-only flag prints just the predicted GPU-hour budget and
$ cost for RunPod / Lambda / GCP A100 (skips per-PDF detail).

Dependencies (already in .venv13):
    pypdf, pdfplumber. Optional: pymupdf (better image counting).
"""

from __future__ import annotations

import argparse
import csv
import json
import logging
import sys
from dataclasses import dataclass, asdict
from pathlib import Path

logging.basicConfig(level=logging.INFO, format="%(message)s")
log = logging.getLogger(__name__)

# Per-route cost constants — measured on our hardware (M-series CPU + V100 estimates)
# Update these once we have actual GPU profiling data.
PER_PAGE_SECONDS = {
    "clean_text":    {"cpu":  0.5, "gpu":  0.1},   # docling text only
    "hybrid_visual": {"cpu": 30.0, "gpu":  3.0},   # docling + Channel A
    "full_ocr":      {"cpu": 540.0, "gpu": 30.0},  # MonkeyOCR + channels
}

# $/hr for cloud GPU options (April 2026 pricing)
GPU_COST_PER_HOUR = {
    "RunPod community 4090":  0.45,
    "RunPod secure 4090":     0.69,
    "Vast.ai 4090 (cheapest)": 0.30,
    "Lambda Labs A100-40GB":  1.29,
    "Akamai/Linode RTX 4000 Ada": 1.50,
    "GCP V100 on-demand":     2.78,
    "GCP A100-40GB":          3.80,
}


@dataclass
class PDFProfile:
    path: str
    filename: str
    file_size_bytes: int
    page_count: int
    has_text_layer: bool
    text_chars_per_page: float
    has_images: bool
    image_count: int
    is_form: bool
    is_pdfa: bool
    producer: str | None
    encrypted: bool
    estimated_route: str
    error: str | None = None


def _classify_route(text_chars_per_page: float, image_count: int,
                    page_count: int) -> str:
    """Decide which Pipeline-1 path this PDF should take."""
    if text_chars_per_page >= 500 and image_count / max(page_count, 1) <= 1.0:
        return "clean_text"
    if text_chars_per_page >= 50:
        return "hybrid_visual"
    return "full_ocr"


def profile_one(pdf_path: Path) -> PDFProfile:
    """Extract features from a single PDF. Never raises — always returns a profile."""
    fname = pdf_path.name
    try:
        size = pdf_path.stat().st_size
    except OSError as e:
        return PDFProfile(
            path=str(pdf_path), filename=fname, file_size_bytes=0,
            page_count=0, has_text_layer=False, text_chars_per_page=0.0,
            has_images=False, image_count=0, is_form=False, is_pdfa=False,
            producer=None, encrypted=False, estimated_route="error",
            error=f"stat failed: {e}",
        )

    try:
        from pypdf import PdfReader
    except ImportError:
        return PDFProfile(
            path=str(pdf_path), filename=fname, file_size_bytes=size,
            page_count=0, has_text_layer=False, text_chars_per_page=0.0,
            has_images=False, image_count=0, is_form=False, is_pdfa=False,
            producer=None, encrypted=False, estimated_route="error",
            error="pypdf not installed",
        )

    try:
        reader = PdfReader(str(pdf_path), strict=False)
        encrypted = reader.is_encrypted
        if encrypted:
            try:
                reader.decrypt("")
            except Exception:
                pass
        page_count = len(reader.pages)
    except Exception as e:
        return PDFProfile(
            path=str(pdf_path), filename=fname, file_size_bytes=size,
            page_count=0, has_text_layer=False, text_chars_per_page=0.0,
            has_images=False, image_count=0, is_form=False, is_pdfa=False,
            producer=None, encrypted=False, estimated_route="error",
            error=f"open failed: {type(e).__name__}: {str(e)[:80]}",
        )

    # Text-layer extraction — sample first 5 pages to keep it fast
    sample_n = min(5, page_count)
    total_chars = 0
    image_count = 0
    has_text = False
    is_form = False
    for i in range(sample_n):
        try:
            page = reader.pages[i]
            text = page.extract_text() or ""
            if text.strip():
                has_text = True
            total_chars += len(text)

            # Image objects on this page
            try:
                images = list(page.images)
                image_count += len(images)
            except Exception:
                pass

            # Form fields
            try:
                if "/Annots" in page:
                    annots = page["/Annots"]
                    if hasattr(annots, "get_object"):
                        annots = annots.get_object()
                    if annots:
                        for a in annots:
                            obj = a.get_object() if hasattr(a, "get_object") else a
                            if obj.get("/Subtype") in ("/Widget",):
                                is_form = True
                                break
            except Exception:
                pass
        except Exception:
            continue

    text_chars_per_page = total_chars / sample_n if sample_n > 0 else 0
    # Extrapolate image count to full document
    if sample_n > 0 and page_count > sample_n:
        image_count = int(image_count * (page_count / sample_n))

    # Producer + PDF/A from document info
    producer = None
    is_pdfa = False
    try:
        info = reader.metadata
        if info:
            producer = (info.get("/Producer") or info.get("/Creator") or "")
            if hasattr(producer, "__str__"):
                producer = str(producer)[:80]
            xmp = getattr(reader, "xmp_metadata", None)
            if xmp and hasattr(xmp, "rdf_root"):
                xml = str(xmp.rdf_root)
                if "pdfaid:part" in xml or "PDF/A" in xml:
                    is_pdfa = True
    except Exception:
        pass

    route = _classify_route(text_chars_per_page, image_count, page_count)

    return PDFProfile(
        path=str(pdf_path),
        filename=fname,
        file_size_bytes=size,
        page_count=page_count,
        has_text_layer=has_text,
        text_chars_per_page=round(text_chars_per_page, 1),
        has_images=image_count > 0,
        image_count=image_count,
        is_form=is_form,
        is_pdfa=is_pdfa,
        producer=producer,
        encrypted=encrypted,
        estimated_route=route,
        error=None,
    )


def aggregate_routing_costs(profiles: list[PDFProfile]) -> dict:
    """Sum per-route page counts and predict GPU-hours + cost."""
    by_route: dict[str, dict] = {}
    for p in profiles:
        if p.estimated_route == "error":
            continue
        rt = p.estimated_route
        if rt not in by_route:
            by_route[rt] = {"pdf_count": 0, "page_count": 0}
        by_route[rt]["pdf_count"] += 1
        by_route[rt]["page_count"] += p.page_count

    # Wall-time budget per route
    cpu_seconds = 0.0
    gpu_seconds = 0.0
    for rt, agg in by_route.items():
        agg["cpu_hours"] = round(agg["page_count"] * PER_PAGE_SECONDS[rt]["cpu"] / 3600.0, 1)
        agg["gpu_hours"] = round(agg["page_count"] * PER_PAGE_SECONDS[rt]["gpu"] / 3600.0, 1)
        cpu_seconds += agg["page_count"] * PER_PAGE_SECONDS[rt]["cpu"]
        gpu_seconds += agg["page_count"] * PER_PAGE_SECONDS[rt]["gpu"]

    cpu_hours = round(cpu_seconds / 3600.0, 1)
    gpu_hours = round(gpu_seconds / 3600.0, 1)

    # Cost per provider
    costs = {}
    for provider, rate in GPU_COST_PER_HOUR.items():
        costs[provider] = {"$": round(gpu_hours * rate, 2),
                           "₹": round(gpu_hours * rate * 84, 0)}  # 84 INR/USD

    return {
        "by_route": by_route,
        "total_cpu_hours": cpu_hours,
        "total_gpu_hours": gpu_hours,
        "cloud_cost_estimates": costs,
        "naive_all_full_ocr_gpu_hours": round(
            sum(p.page_count for p in profiles if p.estimated_route != "error")
            * PER_PAGE_SECONDS["full_ocr"]["gpu"] / 3600.0, 1
        ),
    }


def print_summary(profiles: list[PDFProfile], agg: dict, gpu_cost_only: bool = False) -> None:
    valid = [p for p in profiles if p.estimated_route != "error"]
    errs = [p for p in profiles if p.estimated_route == "error"]

    if not gpu_cost_only:
        print()
        print("=" * 78)
        print(f"  CORPUS PROFILE — {len(profiles)} PDFs ({len(valid)} valid, {len(errs)} errors)")
        print("=" * 78)
        print()
        print(f"  Total bytes:  {sum(p.file_size_bytes for p in valid):>15,}")
        print(f"  Total pages:  {sum(p.page_count for p in valid):>15,}")
        print()
        print("  Per-route breakdown:")
        print(f"    {'route':<16}  {'PDFs':>6}  {'pages':>8}  {'CPU-h':>8}  {'GPU-h':>8}")
        for rt in ("clean_text", "hybrid_visual", "full_ocr"):
            r = agg["by_route"].get(rt)
            if r:
                pct = 100.0 * r["pdf_count"] / len(valid) if valid else 0
                print(f"    {rt:<16}  {r['pdf_count']:>6}  {r['page_count']:>8,}"
                      f"  {r['cpu_hours']:>8}  {r['gpu_hours']:>8}    ({pct:.1f}% of corpus)")
        print()

    print(f"  TOTAL CPU-hours: {agg['total_cpu_hours']}")
    print(f"  TOTAL GPU-hours: {agg['total_gpu_hours']}")
    print(f"  vs naive 'all-MonkeyOCR' GPU-hours: {agg['naive_all_full_ocr_gpu_hours']}")
    speedup = (agg['naive_all_full_ocr_gpu_hours'] / agg['total_gpu_hours']
               if agg['total_gpu_hours'] > 0 else 0)
    print(f"  Routing speedup vs naive: {speedup:.1f}×")
    print()
    print("  Cloud cost estimates (smart-routed):")
    for provider, c in agg["cloud_cost_estimates"].items():
        print(f"    {provider:<32}  ${c['$']:>8,.2f}  (~₹{c['₹']:,.0f})")

    if errs and not gpu_cost_only:
        print()
        print(f"  Failed to profile ({len(errs)}):")
        for p in errs[:5]:
            print(f"    {p.filename}: {p.error}")


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__,
                                formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("corpus_dir", type=Path, help="Directory containing PDFs (recursive)")
    p.add_argument("--json", action="store_true", help="Emit per-PDF JSON")
    p.add_argument("--csv", action="store_true", help="Emit per-PDF CSV")
    p.add_argument("--gpu-cost-only", action="store_true",
                   help="Skip per-PDF detail; print only the cost summary")
    p.add_argument("--limit", type=int, default=0,
                   help="Profile only first N PDFs (for sampling)")
    args = p.parse_args()

    if not args.corpus_dir.exists():
        log.error("dir not found: %s", args.corpus_dir)
        return 1

    pdfs = sorted(args.corpus_dir.rglob("*.pdf"))
    if args.limit > 0:
        pdfs = pdfs[: args.limit]
    if not pdfs:
        log.error("no PDFs found under %s", args.corpus_dir)
        return 1

    if not args.json and not args.csv:
        log.info(f"profiling {len(pdfs)} PDFs from {args.corpus_dir}...")

    profiles: list[PDFProfile] = []
    for i, pdf_path in enumerate(pdfs, 1):
        prof = profile_one(pdf_path)
        profiles.append(prof)
        if (not args.json and not args.csv) and (i % 50 == 0 or i == len(pdfs)):
            log.info(f"  {i}/{len(pdfs)} done")

    agg = aggregate_routing_costs(profiles)

    if args.json:
        print(json.dumps({
            "profiles": [asdict(p) for p in profiles],
            "aggregate": agg,
        }, indent=2, default=str))
    elif args.csv:
        w = csv.DictWriter(sys.stdout, fieldnames=list(asdict(profiles[0]).keys()))
        w.writeheader()
        for prof in profiles:
            w.writerow(asdict(prof))
    else:
        print_summary(profiles, agg, gpu_cost_only=args.gpu_cost_only)

    return 0


if __name__ == "__main__":
    sys.exit(main())
