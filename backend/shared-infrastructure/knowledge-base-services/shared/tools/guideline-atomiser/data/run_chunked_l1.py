#!/usr/bin/env python3
"""
Chunked L1 runner for MonkeyOCR — subprocess-isolated, OOM-safe.

Splits a large PDF into section-aligned chunks, runs MonkeyOCR L1 on each
chunk in a SEPARATE SUBPROCESS (so the OS fully reclaims ~3 GB model memory
between chunks), then merges the results into a single L1 cache file keyed
to the full PDF hash.

Two modes:
  --mode orchestrator (default): splits PDF, spawns workers, merges results
  --mode worker:                 processes a single chunk PDF, saves intermediate JSON

After this script completes, run the normal pipeline with --source delta-pages
and it will get an L1 CACHE HIT, skipping straight to L2+ with all sections
together so cross-section relationships are preserved.

Usage:
    python run_chunked_l1.py --guideline kdigo_2024_ckd.yaml --source delta-pages
"""

import argparse
import hashlib
import json
import os
import subprocess
import sys
import tempfile
import time
from datetime import datetime, timezone
from pathlib import Path

_script_dir = Path(__file__).resolve().parent
_project_root = _script_dir.parent  # guideline-atomiser/
_shared_dir = str(_project_root.parent.parent)  # shared/ directory (for extraction.v4.*)
sys.path.insert(0, str(_project_root))
sys.path.insert(0, _shared_dir)
sys.path.insert(0, "/app/guideline-atomiser")  # Docker path
sys.path.insert(0, "/app")                      # Docker: extraction.v4.* resolves from /app

# Worker code version — bump when worker output format or page mapping changes.
# Intermediate chunk JSONs include this in their filename so stale cached
# results from older code versions are automatically invalidated.
_WORKER_VERSION = "w3"  # w1: original, w2: PAGE marker remapping fix, w3: PageBoundaryOracle fix
# NOTE: w4 backfill runs in orchestrator AFTER merge — no worker change needed.

# ── Parse args ──────────────────────────────────────────────────────────────

parser = argparse.ArgumentParser(description="Chunked L1 MonkeyOCR runner")
parser.add_argument("--guideline", required=True, help="Profile YAML filename")
parser.add_argument("--source", required=True, help="PDF source key from profile")
parser.add_argument("--mode", default="orchestrator", choices=["orchestrator", "worker"],
                    help="orchestrator (default) splits+merges; worker processes one chunk")
# Worker-only args
parser.add_argument("--chunk-pdf", help="(worker) Path to chunk PDF")
parser.add_argument("--page-offset", type=int, help="(worker) Page offset for remapping")
parser.add_argument("--output-json", help="(worker) Path to write intermediate JSON")
parser.add_argument("--extractor", default="monkeyocr", choices=["monkeyocr", "pymupdf"],
                    help="(worker) Which extractor to use — pymupdf for dense table pages")
args = parser.parse_args()


# ═══════════════════════════════════════════════════════════════════════════
# WORKER MODE — process a single chunk, save JSON, exit
# ═══════════════════════════════════════════════════════════════════════════

if args.mode == "worker":
    import fitz as worker_fitz

    chunk_path = Path(args.chunk_pdf)
    page_offset = args.page_offset
    output_path = Path(args.output_json)
    use_pymupdf = args.extractor == "pymupdf"

    print(f"[WORKER] Processing: {chunk_path.name}, page_offset={page_offset}, "
          f"extractor={'PyMuPDF (no OCR)' if use_pymupdf else 'MonkeyOCR'}")

    t0 = time.time()
    blocks_data = []
    tables_data = []
    markdown_parts = []

    if use_pymupdf:
        # ── PyMuPDF fallback — born-digital text extraction, zero GPU/model memory ──
        doc = worker_fitz.open(str(chunk_path))
        for page_idx in range(len(doc)):
            page = doc[page_idx]
            page_num = page_idx + 1 + page_offset  # 1-based, remapped
            page_dict = page.get_text("dict")
            # FIX (w3): PAGE marker BEFORE text, not after.
            # Previous placement (after text) caused systematic -1 page shift
            # because Channel A's _get_page_for_offset() assigns text between
            # markers to the PRECEDING marker's page number.
            markdown_parts.append(f"\n<!-- PAGE {page_num} -->\n")
            for block in page_dict.get("blocks", []):
                if "lines" in block:
                    text = " ".join(
                        span["text"]
                        for line in block["lines"]
                        for span in line["spans"]
                    ).strip()
                    if not text:
                        continue
                    bbox = block.get("bbox", (0, 0, 0, 0))
                    blocks_data.append({
                        "text": text,
                        "block_type": "text",
                        "page_number": page_num,
                        "heading_level": None,
                        "confidence": 1.0,
                        "bbox": {"x0": bbox[0], "y0": bbox[1], "x1": bbox[2], "y1": bbox[3]},
                    })
                    markdown_parts.append(text)
        doc.close()
    else:
        # ── MonkeyOCR (VLM-based OCR) ──
        from monkeyocr_extractor import MonkeyOCRExtractor
        extractor = MonkeyOCRExtractor()
        extractor._ensure_model()
        result = extractor._extract_full(chunk_path, page_range=None)

        for b in result.blocks:
            blocks_data.append({
                "text": b.text,
                "block_type": b.block_type,
                "page_number": b.page_number + page_offset,
                "heading_level": getattr(b, "heading_level", None),
                "confidence": getattr(b, "confidence", None),
                "bbox": (
                    {"x0": b.bbox.x0, "y0": b.bbox.y0, "x1": b.bbox.x1, "y1": b.bbox.y1}
                    if b.bbox else None
                ),
            })
        for t in result.tables:
            tables_data.append({
                "html": t.html,
                "markdown": t.markdown,
                "page_number": t.page_number + page_offset,
                "caption": t.caption,
                "bbox": (
                    {"x0": t.bbox.x0, "y0": t.bbox.y0, "x1": t.bbox.x1, "y1": t.bbox.y1}
                    if t.bbox else None
                ),
            })
        # Rewrite PAGE markers in markdown: MonkeyOCR uses chunk-local
        # page numbers (1-based within the chunk PDF). Add page_offset
        # to produce global PDF page numbers matching blocks_data.
        import re as _re
        def _remap_page(m):
            local_page = int(m.group(1))
            global_page = local_page + page_offset
            return f"<!-- PAGE {global_page} -->"
        remapped_md = _re.sub(
            r'<!--\s*PAGE\s+(\d+)\s*-->',
            _remap_page,
            result.markdown,
            flags=_re.IGNORECASE,
        )

        # FIX (w3): Use PageBoundaryOracle to correct PAGE markers.
        # MonkeyOCR's _insert_page_markers() uses str.index() to find split
        # points, which matches the FIRST occurrence of repeating headers
        # (e.g. "www.kidney-international.org") causing 23% page misattribution.
        # The oracle uses PyMuPDF per-page text with word-sequence regex to
        # find the correct split positions — formatting-tolerant, unique-block
        # preferring, monotonically ordered.
        try:
            # shared/ dir contains extraction/v4/ — two levels up from guideline-atomiser/
            _shared_dir = str(Path(__file__).resolve().parent.parent.parent.parent)
            if _shared_dir not in sys.path:
                sys.path.insert(0, _shared_dir)
            from extraction.v4.page_boundary_oracle import PageBoundaryOracle
            oracle = PageBoundaryOracle(str(chunk_path), page_offset=page_offset)
            corrected_md = oracle.correct_page_markers(remapped_md)
            print(f"[WORKER] PageBoundaryOracle: corrected PAGE markers for {chunk_path.name}")
            markdown_parts.append(corrected_md)
        except Exception as e:
            print(f"[WORKER] PageBoundaryOracle failed ({e}), using MonkeyOCR markers")
            markdown_parts.append(remapped_md)

    elapsed = time.time() - t0

    intermediate = {
        "blocks": blocks_data,
        "tables": tables_data,
        "markdown": "\n".join(markdown_parts) if use_pymupdf else markdown_parts[0] if markdown_parts else "",
        "n_blocks": len(blocks_data),
        "n_tables": len(tables_data),
        "elapsed_sec": elapsed,
        "extractor": args.extractor,
    }

    output_path.write_text(json.dumps(intermediate), encoding="utf-8")
    print(f"[WORKER] Done: {len(blocks_data)} blocks, {len(tables_data)} tables, "
          f"{elapsed/60:.1f} min → {output_path.name}")
    sys.exit(0)


# ═══════════════════════════════════════════════════════════════════════════
# ORCHESTRATOR MODE — split PDF, spawn workers, merge results
# ═══════════════════════════════════════════════════════════════════════════

import fitz  # PyMuPDF

from extraction.v4.guideline_profile import GuidelineProfile

profiles_dir = _script_dir / "profiles"
profile_path = profiles_dir / args.guideline
if not profile_path.exists() and not args.guideline.endswith(".yaml"):
    profile_path = profiles_dir / f"{args.guideline}.yaml"
profile = GuidelineProfile.from_yaml(profile_path)

pdfs_dir = Path(os.environ.get("PDFS_DIR", "/data/pdfs"))
if not pdfs_dir.exists():
    pdfs_dir = _script_dir / "pdfs"

pdf_filename = profile.pdf_sources.get(args.source)
if not pdf_filename:
    sys.exit(f"Source '{args.source}' not in profile. Available: {list(profile.pdf_sources.keys())}")

full_pdf_path = pdfs_dir / pdf_filename
if not full_pdf_path.exists():
    sys.exit(f"PDF not found: {full_pdf_path}")

print(f"{'=' * 70}")
print(f"CHUNKED L1 RUNNER — MonkeyOCR OOM-safe (subprocess isolation)")
print(f"{'=' * 70}")
print(f"PDF:     {full_pdf_path.name}")
print(f"Source:  {args.source}")
print(f"Profile: {profile.profile_id}")

# ── Early-exit: skip if final merged L1 cache already exists ──────────────
cache_dir = _project_root / "data" / "l1_cache"
with open(full_pdf_path, "rb") as f:
    _early_hash = hashlib.sha256(f.read()).hexdigest()
_early_cache = cache_dir / f"{full_pdf_path.stem}_{_early_hash[:12]}_l1.json"
if _early_cache.exists():
    size_kb = _early_cache.stat().st_size / 1024
    print(f"\nL1 CACHE HIT: {_early_cache.name} ({size_kb:.0f} KB) — skipping chunked L1.")
    sys.exit(0)

# ── Define chunk boundaries ───────────────────────────────────────────────
# Three sources, tried in priority order:
#   1. Profile YAML chunk_defs (manual, exact control)
#   2. Auto-chunker (deterministic, works for any guideline)

doc = fitz.open(str(full_pdf_path))
total_pages = len(doc)


def auto_chunk(doc, max_pages: int, char_threshold: int):
    """Deterministic auto-chunker: split PDF into sized chunks with extractor selection.

    Algorithm:
    1. Classify each page as born-digital or scanned by extracting text
       with PyMuPDF and checking if char count exceeds char_threshold.
    2. Group consecutive pages with the same extractor type.
    3. Split any group larger than max_pages into equal sub-chunks.
    4. Name chunks sequentially: chunk-001, chunk-002, ...

    Born-digital pages (high text yield) use PyMuPDF — fast, no GPU.
    Scanned/image-heavy pages (low text yield) use MonkeyOCR — VLM OCR.
    """
    # Step 1: Classify pages
    page_types = []  # "pymupdf" or "monkeyocr" per page
    for page_idx in range(len(doc)):
        page = doc[page_idx]
        text = page.get_text("text")
        char_count = len(text.strip())
        ext = "pymupdf" if char_count >= char_threshold else "monkeyocr"
        page_types.append(ext)

    # Step 2: Group consecutive same-extractor pages
    groups = []  # (start, end, extractor)
    group_start = 0
    for i in range(1, len(page_types)):
        if page_types[i] != page_types[group_start]:
            groups.append((group_start, i, page_types[group_start]))
            group_start = i
    groups.append((group_start, len(page_types), page_types[group_start]))

    # Step 3: Split oversized groups
    chunk_defs = []
    chunk_idx = 1
    for g_start, g_end, ext in groups:
        g_size = g_end - g_start
        if g_size <= max_pages:
            name = f"chunk-{chunk_idx:03d}"
            desc = f"pp.{g_start+1}-{g_end} ({ext})"
            chunk_defs.append((g_start, g_end, name, desc, ext))
            chunk_idx += 1
        else:
            # Split into approximately equal sub-chunks
            n_sub = (g_size + max_pages - 1) // max_pages
            sub_size = g_size // n_sub
            remainder = g_size % n_sub
            pos = g_start
            for s in range(n_sub):
                size = sub_size + (1 if s < remainder else 0)
                name = f"chunk-{chunk_idx:03d}"
                desc = f"pp.{pos+1}-{pos+size} ({ext})"
                chunk_defs.append((pos, pos + size, name, desc, ext))
                chunk_idx += 1
                pos += size

    return chunk_defs


if profile.chunk_defs:
    # Manual chunk boundaries from profile YAML
    CHUNK_DEFS = []
    for cd in profile.chunk_defs:
        CHUNK_DEFS.append((
            cd["start"], cd["end"], cd["name"],
            cd.get("description", f"pp.{cd['start']+1}-{cd['end']}"),
            cd.get("extractor", "monkeyocr"),
        ))
    print(f"Chunks:  {len(CHUNK_DEFS)} (from profile YAML — manual)")
else:
    # Deterministic auto-chunking
    CHUNK_DEFS = auto_chunk(
        doc,
        max_pages=profile.max_pages_per_chunk,
        char_threshold=profile.born_digital_char_threshold,
    )
    print(f"Chunks:  {len(CHUNK_DEFS)} (auto-chunked, max {profile.max_pages_per_chunk} pages)")

doc.close()

print(f"Pages:   {total_pages}")
print(f"Chunks:  {len(CHUNK_DEFS)} (subprocess-isolated)")
for start, end, name, desc, ext in CHUNK_DEFS:
    tag = " [pymupdf]" if ext == "pymupdf" else ""
    print(f"  {name}: pages {start+1}-{end} ({end-start} pages) — {desc}{tag}")
print()

# ── Split PDF into chunk files ──────────────────────────────────────────────

# Use persistent cache dir for intermediate JSONs (survives container crash/restart)
cache_dir = _project_root / "data" / "l1_cache"
cache_dir.mkdir(parents=True, exist_ok=True)

# Temp dir for chunk PDFs only (ephemeral)
work_dir = Path(tempfile.mkdtemp(prefix="chunked_l1_"))
chunk_infos = []

doc = fitz.open(str(full_pdf_path))
for start, end, name, desc, ext in CHUNK_DEFS:
    chunk_path = work_dir / f"{name}.pdf"
    out = fitz.open()
    out.insert_pdf(doc, from_page=start, to_page=end - 1)
    out.save(str(chunk_path), garbage=4, deflate=True)
    out.close()
    # Intermediate JSONs go to persistent cache dir — survive container crash.
    # Include worker version + page_offset in filename to auto-invalidate
    # stale results when the worker code or chunk boundaries change.
    output_json = cache_dir / f"_chunk_{name}_p{start}_{_WORKER_VERSION}_result.json"
    chunk_infos.append((chunk_path, output_json, start, end, name, desc, ext))
    print(f"Created: {chunk_path.name} ({end - start} pages)")
doc.close()
print()

# ── Run each chunk as a separate subprocess ─────────────────────────────────

this_script = str(Path(__file__).resolve())
chunk_timings = []

for chunk_path, output_json, page_start, page_end, name, desc, ext in chunk_infos:
    print(f"{'─' * 70}")

    # RESUME SUPPORT: skip chunks that already have a result JSON
    if output_json.exists():
        data = json.loads(output_json.read_text(encoding="utf-8"))
        chunk_timings.append((name, data["n_blocks"], data["n_tables"], 0))
        print(f"SKIP (cached): {name} — {data['n_blocks']} blocks, {data['n_tables']} tables")
        print(f"{'─' * 70}")
        print()
        continue

    ext_label = " [pymupdf]" if ext == "pymupdf" else ""
    print(f"SPAWNING WORKER: {name} — {desc} (pages {page_start+1}-{page_end}){ext_label}")
    print(f"{'─' * 70}")

    t0 = time.time()

    cmd = [
        sys.executable, "-u", this_script,
        "--guideline", args.guideline,
        "--source", args.source,
        "--mode", "worker",
        "--chunk-pdf", str(chunk_path),
        "--page-offset", str(page_start),
        "--output-json", str(output_json),
        "--extractor", ext,
    ]

    proc = subprocess.run(cmd, timeout=None)

    elapsed = time.time() - t0

    if proc.returncode != 0:
        print(f"  WORKER FAILED (exit {proc.returncode}) for {name}")
        print(f"  Aborting — check logs above for error details.")
        sys.exit(proc.returncode)

    # Read back results summary
    if output_json.exists():
        data = json.loads(output_json.read_text(encoding="utf-8"))
        chunk_timings.append((name, data["n_blocks"], data["n_tables"], elapsed))
        print(f"  Blocks: {data['n_blocks']}, Tables: {data['n_tables']}, "
              f"Time: {elapsed/60:.1f} min")
    else:
        print(f"  WARNING: No output JSON for {name}")
        chunk_timings.append((name, 0, 0, elapsed))

    # Memory is fully reclaimed — subprocess exited, OS freed all pages
    print(f"  Memory reclaimed (subprocess exited).")
    print()

# ── Merge all chunk results into single L1 cache ───────────────────────────

print(f"{'=' * 70}")
print(f"MERGING {len(CHUNK_DEFS)} chunks → single L1 cache")
print(f"{'=' * 70}")

all_blocks = []
all_tables = []
all_markdown_parts = []

for chunk_path, output_json, page_start, page_end, name, desc, ext in chunk_infos:
    if not output_json.exists():
        print(f"  SKIP {name} — no output JSON")
        continue
    data = json.loads(output_json.read_text(encoding="utf-8"))
    all_blocks.extend(data["blocks"])
    all_tables.extend(data["tables"])
    all_markdown_parts.append(f"\n<!-- Chunk {name}: pages {page_start+1}-{page_end} -->\n")
    all_markdown_parts.append(data["markdown"])
    print(f"  {name}: {data['n_blocks']} blocks, {data['n_tables']} tables")

# Compute the hash of the FULL PDF (not chunks) — this is the cache key
with open(full_pdf_path, "rb") as f:
    full_hash = hashlib.sha256(f.read()).hexdigest()

merged_data = {
    "blocks": all_blocks,
    "tables": all_tables,
    "markdown": "\n".join(all_markdown_parts),
    "provenance": {
        "source_file": str(full_pdf_path),
        "source_hash": full_hash,
        "extraction_timestamp": datetime.now(timezone.utc).isoformat(),
        "extractor_version": "1.0.0",
        "worker_version": _WORKER_VERSION,
        "marker_version": "monkeyocr",
        "seed": 42,
        "total_pages": total_pages,
        "extraction_params": {
            "parser": "monkeyocr",
            "chunked_l1": True,
            "subprocess_isolated": True,
            "chunks": len(CHUNK_DEFS),
            "chunk_boundaries": [(s, e, n, x) for s, e, n, _, x in CHUNK_DEFS],
        },
    },
}

# ── Page coverage validation gate ─────────────────────────────────────────
# Verify that the merged markdown has PAGE markers for all expected pages.
# This catches the exact bug class where stale intermediate JSONs carry
# chunk-local page numbers instead of globally remapped ones.
import re as _re_merge
merged_markdown = merged_data["markdown"]
found_pages = sorted(set(int(p) for p in _re_merge.findall(
    r'<!--\s*PAGE\s+(\d+)\s*-->', merged_markdown
)))
expected_pages = list(range(1, total_pages + 1))
missing_pages = [p for p in expected_pages if p not in found_pages]
duplicate_pages = [p for p in found_pages
                   if merged_markdown.count(f"PAGE {p} -->") > 1]

print(f"\nPage coverage gate:")
print(f"  Expected: {total_pages} pages, Found: {len(found_pages)} unique PAGE markers")
if missing_pages:
    print(f"  MISSING pages: {missing_pages}")
if duplicate_pages:
    print(f"  DUPLICATE pages: {duplicate_pages}")

coverage_pct = len(found_pages) / total_pages * 100 if total_pages else 0
if coverage_pct < 80:
    print(f"\n  GATE BLOCKED: Page coverage {coverage_pct:.0f}% < 80% threshold.")
    print(f"  This likely indicates stale intermediate chunk JSONs with wrong page numbers.")
    print(f"  Delete l1_cache/_chunk_*_result.json files and re-run.")
    sys.exit(1)
elif missing_pages or duplicate_pages:
    print(f"  WARNING: {len(missing_pages)} missing, {len(duplicate_pages)} duplicates "
          f"(coverage {coverage_pct:.0f}%)")
else:
    print(f"  PASS: All {total_pages} pages present, no duplicates.")

# ── Page content validation gate (w3 NEW) ─────────────────────────────────
# Check that each page has actual content between its markers.
# This catches "absorbed pages" where _insert_page_markers() consumed
# zero text for a page (the `pass` fallback at monkeyocr_extractor.py:837),
# leaving PAGE N and PAGE N+1 markers adjacent with nothing between them.
_page_marker_positions = [(m.start(), int(m.group(1)))
                          for m in _re_merge.finditer(r'<!--\s*PAGE\s+(\d+)\s*-->', merged_markdown)]
_page_marker_positions.sort(key=lambda x: x[0])

empty_pages = []
for i, (pos, page_num) in enumerate(_page_marker_positions):
    if i + 1 < len(_page_marker_positions):
        next_pos = _page_marker_positions[i + 1][0]
    else:
        next_pos = len(merged_markdown)

    # Extract text between this marker and the next, strip whitespace + markers
    between = merged_markdown[pos:next_pos]
    # Remove the marker itself and chunk comment markers
    content = _re_merge.sub(r'<!--.*?-->', '', between).strip()
    if len(content) < 50:
        empty_pages.append((page_num, len(content)))

if empty_pages:
    print(f"\n  Page CONTENT gate:")
    print(f"  {len(empty_pages)} pages have < 50 chars of content between markers:")
    for page_num, char_count in empty_pages[:10]:
        print(f"    Page {page_num}: {char_count} chars")
    if len(empty_pages) > 10:
        print(f"    ... and {len(empty_pages) - 10} more")

    # ── PyMuPDF Backfill for MonkeyOCR-dropped pages ─────────────────
    # MonkeyOCR (VLM-based) sometimes produces empty output for born-digital
    # pages. For these pages, PyMuPDF can reliably extract the embedded text
    # layer. We check each empty page: if the PDF has ≥200 chars of text,
    # it's born-digital content that MonkeyOCR dropped — backfill it.
    print(f"\n  PyMuPDF backfill: checking {len(empty_pages)} empty pages...")
    backfill_doc = fitz.open(str(full_pdf_path))
    backfilled_pages = []

    for page_num, char_count in empty_pages:
        pdf_page_idx = page_num - 1  # 0-based index into full PDF
        if pdf_page_idx < 0 or pdf_page_idx >= len(backfill_doc):
            continue

        page = backfill_doc[pdf_page_idx]
        pymupdf_text = page.get_text("text").strip()

        if len(pymupdf_text) < 200:
            # Genuinely sparse page (blank separator, image-only, etc.)
            continue

        # Build replacement markdown: PAGE marker + block-level text
        page_dict = page.get_text("dict")
        replacement_parts = []
        for block in page_dict.get("blocks", []):
            if block.get("type") != 0:
                continue
            block_text = " ".join(
                span.get("text", "")
                for line in block.get("lines", [])
                for span in line.get("spans", [])
            ).strip()
            if block_text:
                replacement_parts.append(block_text)

        if not replacement_parts:
            continue

        replacement_md = "\n".join(replacement_parts)

        # Find the PAGE marker for this page and replace the empty gap
        # with the PyMuPDF-extracted content.
        marker_pattern = _re_merge.compile(
            rf'(<!--\s*PAGE\s+{page_num}\s*-->)\s*'
        )
        match = marker_pattern.search(merged_markdown)
        if match:
            insert_pos = match.end()
            merged_markdown = (
                merged_markdown[:insert_pos]
                + "\n" + replacement_md + "\n"
                + merged_markdown[insert_pos:]
            )
            backfilled_pages.append((page_num, len(pymupdf_text)))

    backfill_doc.close()

    if backfilled_pages:
        # Update merged_data with backfilled markdown
        merged_data["markdown"] = merged_markdown

        # Also add PyMuPDF blocks to the blocks list for downstream channels
        backfill_doc2 = fitz.open(str(full_pdf_path))
        backfill_block_count = 0
        for page_num, _ in backfilled_pages:
            pdf_page_idx = page_num - 1
            page = backfill_doc2[pdf_page_idx]
            page_dict = page.get_text("dict")
            for block in page_dict.get("blocks", []):
                if block.get("type") != 0:
                    continue
                block_text = " ".join(
                    span.get("text", "")
                    for line in block.get("lines", [])
                    for span in line.get("spans", [])
                ).strip()
                if not block_text:
                    continue
                bbox = block.get("bbox", (0, 0, 0, 0))
                all_blocks.append({
                    "text": block_text,
                    "block_type": "text",
                    "page_number": page_num,
                    "heading_level": None,
                    "confidence": 1.0,
                    "bbox": {"x0": bbox[0], "y0": bbox[1],
                             "x1": bbox[2], "y1": bbox[3]},
                })
                backfill_block_count += 1
        backfill_doc2.close()
        merged_data["blocks"] = all_blocks

        print(f"  ✅ Backfilled {len(backfilled_pages)} pages with PyMuPDF text:")
        for pg, chars in backfilled_pages:
            print(f"    Page {pg}: {chars} chars from PDF text layer")
        print(f"    Added {backfill_block_count} blocks to L1 cache")

        # Record backfill provenance
        merged_data["provenance"]["backfilled_pages"] = [
            {"page": pg, "chars": ch, "source": "pymupdf_born_digital"}
            for pg, ch in backfilled_pages
        ]
    else:
        print(f"  ⚠️  No born-digital content found for empty pages (image-only?)")
        print(f"  These pages may have been 'absorbed' by adjacent pages.")
        print(f"  PageBoundaryOracle should have corrected this — check oracle logs.")
else:
    print(f"  Content gate: PASS — all pages have >= 50 chars of content.")

# Save to L1 cache with the full PDF's hash (same key the pipeline will look up)
# cache_dir already set above (persistent volume)
cache_file = cache_dir / f"{full_pdf_path.stem}_{full_hash[:12]}_l1.json"
cache_file.write_text(json.dumps(merged_data), encoding="utf-8")

print()
print(f"Total blocks:  {len(all_blocks)}")
print(f"Total tables:  {len(all_tables)}")
print(f"Cache saved:   {cache_file.name} ({cache_file.stat().st_size / 1024:.0f} KB)")
print()

# ── Summary ─────────────────────────────────────────────────────────────────

print(f"{'=' * 70}")
print(f"CHUNKED L1 COMPLETE")
print(f"{'=' * 70}")
total_time = sum(t for _, _, _, t in chunk_timings)
for name, nblocks, ntables, elapsed in chunk_timings:
    print(f"  {name}: {nblocks:4d} blocks, {ntables:2d} tables, {elapsed/60:5.1f} min")
print(f"  {'─' * 50}")
print(f"  Total: {len(all_blocks):4d} blocks, {len(all_tables):2d} tables, {total_time/60:5.1f} min")
print()

# Cleanup temp chunk PDFs + intermediate JSONs
import shutil
shutil.rmtree(work_dir, ignore_errors=True)
for _, output_json, _, _, name, _, _ in chunk_infos:
    if output_json.exists():
        output_json.unlink()
print(f"Cleaned up temp dir + intermediate JSONs")

print(f"Next step: run the full pipeline (L2-L3) — it will get an L1 CACHE HIT:")
print(f"  python run_pipeline_targeted.py --pipeline 1 --source {args.source} \\")
print(f"    --l1 monkeyocr --guideline {args.guideline}")
