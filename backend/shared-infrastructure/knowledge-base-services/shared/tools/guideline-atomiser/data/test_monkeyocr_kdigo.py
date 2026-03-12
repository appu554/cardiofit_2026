#!/usr/bin/env python3
"""
MonkeyOCR Test: KDIGO 2022 Page 20 (algorithm figure) + Page 34 (rotated table)

Two-phase script:
  Phase 1 (no GPU needed): Extract pages 20 & 34 as single-page PDFs using PyMuPDF
  Phase 2 (GPU needed):    Run MonkeyOCR on extracted pages, dump structured output

Usage:
  # Phase 1 — extract pages (runs anywhere)
  python test_monkeyocr_kdigo.py --extract

  # Phase 2 — run MonkeyOCR (requires GPU + monkeyocr installed)
  python test_monkeyocr_kdigo.py --parse

  # Both phases
  python test_monkeyocr_kdigo.py --extract --parse

Install MonkeyOCR first:
  git clone https://github.com/Yuliang-Liu/MonkeyOCR.git
  cd MonkeyOCR
  pip install -e .
  python tools/download_model.py -n MonkeyOCR-pro-3B
"""

import argparse
import json
import os
import sys
from pathlib import Path

# Paths
SCRIPT_DIR = Path(__file__).resolve().parent
PDF_PATH = SCRIPT_DIR.parent / "shared" / "datasources" / "kdigo" / "pdfs" / \
    "KDIGO-2022-Clinical-Practice-Guideline-for-Diabetes-Management-in-CKD.pdf"
# Fallback: the copy in guideline-atomiser/data/pdfs
PDF_PATH_ALT = SCRIPT_DIR / "pdfs" / \
    "KDIGO-2022-Clinical-Practice-Guideline-for-Diabetes-Management-in-CKD.pdf"

OUTPUT_DIR = SCRIPT_DIR / "monkeyocr_test_output"

# Pages to test (1-indexed PDF page numbers)
TEST_PAGES = [20, 34]


# ============================================================================
# Phase 1: Extract pages using PyMuPDF
# ============================================================================

def extract_pages():
    """Extract specific pages from the KDIGO PDF into single-page PDFs."""
    import fitz  # PyMuPDF

    pdf_path = PDF_PATH if PDF_PATH.exists() else PDF_PATH_ALT
    if not pdf_path.exists():
        print(f"ERROR: KDIGO PDF not found at:\n  {PDF_PATH}\n  {PDF_PATH_ALT}")
        sys.exit(1)

    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)

    doc = fitz.open(str(pdf_path))
    print(f"Opened: {pdf_path.name} ({len(doc)} pages)")

    extracted_paths = []
    for page_num in TEST_PAGES:
        page_idx = page_num - 1  # fitz uses 0-indexed
        if page_idx >= len(doc):
            print(f"  SKIP: Page {page_num} exceeds document length ({len(doc)})")
            continue

        page = doc[page_idx]
        out_pdf = OUTPUT_DIR / f"kdigo_page_{page_num}.pdf"

        # Create single-page PDF
        new_doc = fitz.open()
        new_doc.insert_pdf(doc, from_page=page_idx, to_page=page_idx)
        new_doc.save(str(out_pdf))
        new_doc.close()

        # Also render as PNG for visual reference
        out_png = OUTPUT_DIR / f"kdigo_page_{page_num}.png"
        pix = page.get_pixmap(dpi=200)
        pix.save(str(out_png))

        # Dump PyMuPDF rawdict for comparison (this is what our pipeline uses)
        rawdict = page.get_text("rawdict")
        out_rawdict = OUTPUT_DIR / f"kdigo_page_{page_num}_pymupdf_rawdict.json"
        with open(out_rawdict, "w") as f:
            json.dump(rawdict, f, indent=2, default=str)

        # Also dump search_for results for L1_RECOVERY text on these pages
        l1_texts = {
            20: "Metformin (if eGFR ≥30) SGLT2i",
            34: "CrCl 10–30 ml/min",
        }
        if page_num in l1_texts:
            search_text = l1_texts[page_num]
            rects = page.search_for(search_text)
            search_result = {
                "search_text": search_text,
                "page_number": page_num,
                "results": [
                    {"x0": r.x0, "y0": r.y0, "x1": r.x1, "y1": r.y1,
                     "width": r.width, "height": r.height}
                    for r in rects
                ],
                "num_results": len(rects),
            }
            out_search = OUTPUT_DIR / f"kdigo_page_{page_num}_search_for.json"
            with open(out_search, "w") as f:
                json.dump(search_result, f, indent=2)
            print(f"  Page {page_num}: search_for('{search_text}') → {len(rects)} results")

        print(f"  Page {page_num}: {out_pdf.name}, {out_png.name}, rawdict dumped")
        extracted_paths.append(out_pdf)

    doc.close()
    print(f"\nExtracted {len(extracted_paths)} pages to: {OUTPUT_DIR}")
    return extracted_paths


# ============================================================================
# Phase 2: Run MonkeyOCR
# ============================================================================

def run_monkeyocr():
    """Run MonkeyOCR on the extracted single-page PDFs."""
    try:
        from magic_pdf.data.data_reader_writer import FileBasedDataWriter, FileBasedDataReader
        from magic_pdf.data.dataset import PymuDocDataset
        from magic_pdf.model.doc_analyze_by_custom_model_llm import doc_analyze_llm
        from magic_pdf.model.custom_model import MonkeyOCR
    except ImportError:
        print("ERROR: MonkeyOCR not installed.")
        print("Install with:")
        print("  git clone https://github.com/Yuliang-Liu/MonkeyOCR.git")
        print("  cd MonkeyOCR && pip install -e .")
        print("  python tools/download_model.py -n MonkeyOCR-pro-3B")
        sys.exit(1)

    # Find config — default location in MonkeyOCR repo
    config_candidates = [
        Path("model_configs.yaml"),
        Path.home() / "MonkeyOCR" / "model_configs.yaml",
        Path("/opt/MonkeyOCR/model_configs.yaml"),
    ]
    config_path = None
    for c in config_candidates:
        if c.exists():
            config_path = str(c)
            break

    if not config_path:
        config_path = os.environ.get("MONKEYOCR_CONFIG", "model_configs.yaml")
        print(f"WARNING: Using config path: {config_path}")
        print("Set MONKEYOCR_CONFIG env var if model config is elsewhere.")

    print(f"Loading MonkeyOCR model (config: {config_path})...")
    model = MonkeyOCR(config_path)

    for page_num in TEST_PAGES:
        pdf_path = OUTPUT_DIR / f"kdigo_page_{page_num}.pdf"
        if not pdf_path.exists():
            print(f"  SKIP: {pdf_path} not found. Run --extract first.")
            continue

        page_output_dir = OUTPUT_DIR / f"monkeyocr_page_{page_num}"
        page_output_dir.mkdir(parents=True, exist_ok=True)
        image_dir = page_output_dir / "images"
        image_dir.mkdir(exist_ok=True)

        print(f"\n{'='*60}")
        print(f"Processing Page {page_num}: {pdf_path.name}")
        print(f"{'='*60}")

        reader = FileBasedDataReader()
        file_bytes = reader.read(str(pdf_path))
        ds = PymuDocDataset(file_bytes)

        # Run document analysis
        infer_result = ds.apply(
            doc_analyze_llm,
            MonkeyOCR_model=model,
            split_pages=False,
            pred_abandon=False,
        )

        # OCR pipeline
        image_writer = FileBasedDataWriter(str(image_dir))
        md_writer = FileBasedDataWriter(str(page_output_dir))

        pipe_result = infer_result.pipe_ocr_mode(image_writer, MonkeyOCR_model=model)

        # Save all outputs
        name = f"kdigo_page_{page_num}"

        # Layout visualization (overlaid on PDF)
        infer_result.draw_model(str(page_output_dir / f"{name}_model.pdf"))
        pipe_result.draw_layout(str(page_output_dir / f"{name}_layout.pdf"))
        pipe_result.draw_span(str(page_output_dir / f"{name}_spans.pdf"))

        # Markdown output
        pipe_result.dump_md(md_writer, f"{name}.md", "images")

        # Content list (structured JSON)
        pipe_result.dump_content_list(md_writer, f"{name}_content_list.json", "images")

        # Middle JSON — block-level coordinates and OCR results
        pipe_result.dump_middle_json(md_writer, f"{name}_middle.json")

        print(f"  Output saved to: {page_output_dir}")
        print(f"  Files: {name}.md, {name}_middle.json, {name}_content_list.json")
        print(f"         {name}_layout.pdf, {name}_spans.pdf, {name}_model.pdf")

    print(f"\nDone. All output in: {OUTPUT_DIR}")


# ============================================================================
# Summary: compare PyMuPDF rawdict vs MonkeyOCR output
# ============================================================================

def summarize():
    """Print a comparison summary of what each tool extracted."""
    print(f"\n{'='*60}")
    print("SUMMARY: PyMuPDF rawdict vs MonkeyOCR")
    print(f"{'='*60}\n")

    for page_num in TEST_PAGES:
        print(f"--- Page {page_num} ---")

        # PyMuPDF rawdict summary
        rawdict_path = OUTPUT_DIR / f"kdigo_page_{page_num}_pymupdf_rawdict.json"
        if rawdict_path.exists():
            with open(rawdict_path) as f:
                rd = json.load(f)
            blocks = rd.get("blocks", [])
            text_blocks = [b for b in blocks if b.get("type") == 0]
            image_blocks = [b for b in blocks if b.get("type") == 1]
            total_chars = sum(
                len(span.get("text", ""))
                for b in text_blocks
                for line in b.get("lines", [])
                for span in line.get("spans", [])
            )
            print(f"  PyMuPDF rawdict: {len(text_blocks)} text blocks, "
                  f"{len(image_blocks)} image blocks, ~{total_chars} chars")
        else:
            print(f"  PyMuPDF rawdict: not found (run --extract)")

        # search_for results
        search_path = OUTPUT_DIR / f"kdigo_page_{page_num}_search_for.json"
        if search_path.exists():
            with open(search_path) as f:
                sr = json.load(f)
            print(f"  search_for('{sr['search_text']}'): {sr['num_results']} rects")
            for r in sr["results"]:
                print(f"    [{r['x0']:.1f}, {r['y0']:.1f}, {r['x1']:.1f}, {r['y1']:.1f}] "
                      f"({r['width']:.1f} x {r['height']:.1f} pt)")

        # MonkeyOCR middle.json summary
        middle_path = OUTPUT_DIR / f"monkeyocr_page_{page_num}" / f"kdigo_page_{page_num}_middle.json"
        if middle_path.exists():
            with open(middle_path) as f:
                mj = json.load(f)
            # middle.json structure varies — try common patterns
            if isinstance(mj, list):
                print(f"  MonkeyOCR middle.json: {len(mj)} blocks")
                for i, block in enumerate(mj[:5]):
                    btype = block.get("type", "?")
                    bbox = block.get("bbox", block.get("poly", "?"))
                    text = str(block.get("text", block.get("content", "")))[:60]
                    print(f"    [{i}] type={btype} bbox={bbox} text='{text}...'")
            elif isinstance(mj, dict):
                pages = mj.get("pdf_info", [])
                if pages:
                    page_data = pages[0] if isinstance(pages, list) else pages
                    blocks = page_data.get("preproc_blocks", page_data.get("blocks", []))
                    print(f"  MonkeyOCR middle.json: {len(blocks)} blocks in page")
                    for i, block in enumerate(blocks[:8]):
                        btype = block.get("type", "?")
                        bbox = block.get("bbox", "?")
                        lines = block.get("lines", [])
                        text_preview = ""
                        if lines:
                            spans = lines[0].get("spans", [])
                            if spans:
                                text_preview = spans[0].get("content", "")[:60]
                        print(f"    [{i}] type={btype} bbox={bbox} text='{text_preview}'")
                else:
                    print(f"  MonkeyOCR middle.json: {list(mj.keys())}")
        else:
            print(f"  MonkeyOCR: not found (run --parse with GPU)")

        # MonkeyOCR markdown preview
        md_path = OUTPUT_DIR / f"monkeyocr_page_{page_num}" / f"kdigo_page_{page_num}.md"
        if md_path.exists():
            with open(md_path) as f:
                md_text = f.read()
            lines = md_text.strip().split("\n")
            print(f"  MonkeyOCR markdown: {len(lines)} lines, {len(md_text)} chars")
            for line in lines[:5]:
                print(f"    | {line[:80]}")
            if len(lines) > 5:
                print(f"    | ... ({len(lines) - 5} more lines)")

        print()


# ============================================================================
# Main
# ============================================================================

def main():
    parser = argparse.ArgumentParser(
        description="Test MonkeyOCR on KDIGO 2022 pages 20 & 34",
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument("--extract", action="store_true",
                        help="Phase 1: Extract pages as single-page PDFs + PyMuPDF rawdict")
    parser.add_argument("--parse", action="store_true",
                        help="Phase 2: Run MonkeyOCR on extracted pages (requires GPU)")
    parser.add_argument("--summary", action="store_true",
                        help="Print comparison summary of PyMuPDF vs MonkeyOCR output")
    parser.add_argument("--all", action="store_true",
                        help="Run all phases")

    args = parser.parse_args()

    if not any([args.extract, args.parse, args.summary, args.all]):
        # Default: just extract (no GPU needed)
        args.extract = True
        args.summary = True

    if args.all:
        args.extract = True
        args.parse = True
        args.summary = True

    if args.extract:
        print("Phase 1: Extracting pages with PyMuPDF...\n")
        extract_pages()

    if args.parse:
        print("\nPhase 2: Running MonkeyOCR...\n")
        run_monkeyocr()

    if args.summary:
        summarize()


if __name__ == "__main__":
    main()
