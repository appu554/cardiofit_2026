"""
L1: PDF Parsing with MonkeyOCR — VLM-based Structure-Recognition-Relation parser.

Replaces Marker v1.10 as the L1 PDF parser in Pipeline 1.  Produces the same
ExtractionResult contract (blocks, tables, markdown, provenance) consumed by
all downstream channels (Ch0, ChA, B–F) and the L1 Completeness Oracle.

Key advantages over Marker:
  - Per-block bounding boxes from middle.json (Marker only provided bboxes
    in the PyMuPDF fallback path, not from its native ML pipeline)
  - VLM-based OCR (Qwen2.5-VL) produces better results on complex layouts
  - Unified layout+OCR in one model call (no separate table detection pass)

Architecture position:  **MonkeyOCR** → L1 Oracle → Channel 0 → Channels A–F

Usage:
    from monkeyocr_extractor import MonkeyOCRExtractor

    extractor = MonkeyOCRExtractor()
    result = extractor.extract(pdf_path="kdigo_2022_diabetes.pdf")

    for block in result.blocks:
        print(f"Page {block.page_number}: {block.text[:50]}...")
        print(f"  Bbox: {block.bbox}")  # Always populated from middle.json
"""

import io
import json
import hashlib
import os
import re
import tempfile
from datetime import datetime, timezone
from pathlib import Path
from typing import Literal, Optional, Union

from marker_extractor import (
    BoundingBox,
    ClinicalOCRPostProcessor,
    ExtractionProvenance,
    ExtractionResult,
    TableBlock,
    TextBlock,
)


class MonkeyOCRExtractor:
    """
    L1 PDF Extractor using MonkeyOCR (magic_pdf).

    MonkeyOCR uses a Structure-Recognition-Relation paradigm:
      1. DoclayoutYOLO detects document layout regions
      2. Qwen2.5-VL performs OCR on each region
      3. LayoutReader determines reading order

    The model is loaded once in __init__ (~10s) and reused for all pages.
    """

    VERSION = "1.0.0"

    def __init__(
        self,
        config_path: Optional[str] = None,
        enable_ocr_postprocessing: bool = True,
        ocr_postprocessor: Optional[ClinicalOCRPostProcessor] = None,
        seed: int = 42,
    ):
        """
        Initialize the MonkeyOCR extractor.

        Args:
            config_path: Path to MonkeyOCR model_configs.yaml.
                         Resolution order:
                           1. Explicit config_path argument
                           2. MONKEYOCR_CONFIG environment variable
                           3. Standard paths: ./model_configs.yaml,
                              ~/MonkeyOCR/model_configs.yaml,
                              /opt/MonkeyOCR/model_configs.yaml,
                              /tmp/MonkeyOCR/model_configs.yaml
            enable_ocr_postprocessing: Apply ClinicalOCRPostProcessor + LaTeX stripping
            ocr_postprocessor: Custom post-processor (uses default if None)
            seed: Random seed for reproducibility
        """
        self.seed = seed
        self.enable_ocr_postprocessing = enable_ocr_postprocessing

        if enable_ocr_postprocessing:
            self.ocr_postprocessor = ocr_postprocessor or ClinicalOCRPostProcessor()
        else:
            self.ocr_postprocessor = None

        # Resolve config path
        self._config_path = self._resolve_config(config_path)

        # Lazy-load model on first use (avoids import errors if magic_pdf not installed)
        self._model = None
        self._magic_pdf_loaded = False

    def _resolve_config(self, explicit_path: Optional[str]) -> str:
        """Resolve MonkeyOCR config path using priority chain."""
        if explicit_path and Path(explicit_path).exists():
            return explicit_path

        env_path = os.environ.get("MONKEYOCR_CONFIG")
        if env_path and Path(env_path).exists():
            return env_path

        standard_paths = [
            Path("model_configs.yaml"),
            Path.home() / "MonkeyOCR" / "model_configs.yaml",
            Path("/opt/MonkeyOCR/model_configs.yaml"),
            Path("/tmp/MonkeyOCR/model_configs.yaml"),
            Path("/tmp/MonkeyOCR/model_configs_mps.yaml"),
        ]
        for p in standard_paths:
            if p.exists():
                return str(p)

        # Fallback: return whatever was given or a default
        return explicit_path or "model_configs.yaml"

    def _ensure_model(self):
        """Lazy-load MonkeyOCR model on first use.

        Resolves HuggingFace cache paths at runtime so the same code works
        locally (macOS Metal), in Docker (CPU), and on GPU servers.  Models
        are downloaded on first call (~3 GB for echo840/MonkeyOCR).
        """
        if self._model is not None:
            return

        runtime_config = self._build_runtime_config()

        from magic_pdf.model.custom_model import MonkeyOCR

        print(f"Loading MonkeyOCR model (runtime config: {runtime_config})...")
        self._model = MonkeyOCR(runtime_config)
        self._magic_pdf_loaded = True
        print("MonkeyOCR model loaded.")

    def _build_runtime_config(self) -> str:
        """Generate a runtime model_configs.yaml with resolved HF cache paths.

        MonkeyOCR's custom_model.py constructs model paths as:
            os.path.join(models_dir, weights[model_name])
        HuggingFace stores snapshots under:
            ~/.cache/huggingface/hub/models--org--repo/snapshots/{hash}/

        This method bridges the gap by setting models_dir to the MonkeyOCR
        snapshot directory so all relative weight paths resolve correctly.
        """
        import yaml
        from huggingface_hub import snapshot_download

        # Download / resolve MonkeyOCR snapshot (no-op if cached)
        print("Resolving MonkeyOCR model paths from HuggingFace cache...")
        monkeyocr_snap = snapshot_download("echo840/MonkeyOCR")
        print(f"  MonkeyOCR snapshot: {monkeyocr_snap}")

        # Detect device: MONKEYOCR_DEVICE env override → CUDA → CPU.
        # MPS is explicitly skipped: Qwen2.5-VL exhausts Apple Silicon's
        # ~30 GB unified memory on full guideline runs, causing macOS OOM
        # killer (SIGKILL 137).  CPU is slower (~30s/batch vs ~10s) but
        # completes reliably.  Use MONKEYOCR_DEVICE=mps to force MPS for
        # short test runs (≤20 pages).
        import torch
        device_override = os.environ.get("MONKEYOCR_DEVICE", "").lower()
        if device_override in ("cuda", "mps", "cpu"):
            device = device_override
        elif torch.cuda.is_available():
            device = "cuda"
        else:
            device = "cpu"
        print(f"  Device: {device}")

        # Layout model: bundled inside MonkeyOCR snapshot
        layout_file = "Structure/doclayout_yolo_docstructbench_imgsz1280_2501.pt"
        layout_path = os.path.join(monkeyocr_snap, layout_file)
        if not os.path.exists(layout_path):
            raise FileNotFoundError(
                f"Layout model not found at {layout_path}. "
                "The echo840/MonkeyOCR snapshot may be incomplete."
            )

        # Layoutreader: try HF cache first, download if missing
        try:
            layoutreader_snap = snapshot_download("hantian/layoutreader")
            layoutreader_rel = os.path.relpath(layoutreader_snap, monkeyocr_snap)
            print(f"  LayoutReader: {layoutreader_snap}")
        except Exception as e:
            print(f"  LayoutReader download: {e}")
            # Create a dummy path — MonkeyOCR will raise if truly needed
            layoutreader_rel = "layoutreader_not_available"

        config = {
            "device": device,
            "models_dir": monkeyocr_snap,
            "weights": {
                "doclayout_yolo": layout_file,
                "layoutreader": layoutreader_rel,
            },
            "layout_config": {
                "model": "doclayout_yolo",
                "reader": {
                    "name": "layoutreader",
                },
            },
            "chat_config": {
                "weight_path": "Recognition",
                "backend": "transformers",
                # batch_size: CUDA=4 (fast, stable bf16), CPU/MPS=1.
                # MPS bf16 matmul produces inf/nan on batched attention;
                # CPU batch=1 keeps peak RAM manageable (~15GB vs ~25GB).
                "batch_size": 4 if device == "cuda" else 1,
            },
        }

        # Write to temp file
        runtime_path = os.path.join(
            tempfile.gettempdir(), "monkeyocr_runtime_config.yaml"
        )
        with open(runtime_path, "w") as f:
            yaml.dump(config, f, default_flow_style=False)
        print(f"  Runtime config written: {runtime_path}")

        return runtime_path

    def _compute_file_hash(self, file_path: Path) -> str:
        """Compute SHA-256 hash of the source file."""
        sha256 = hashlib.sha256()
        with open(file_path, "rb") as f:
            for chunk in iter(lambda: f.read(8192), b""):
                sha256.update(chunk)
        return sha256.hexdigest()

    # ═══════════════════════════════════════════════════════════════════════════
    # Public API — same signature as MarkerExtractor.extract()
    # ═══════════════════════════════════════════════════════════════════════════

    # Persistent cache directory for L1 extraction results (NOT /tmp — survives reboots)
    L1_CACHE_DIR = Path(__file__).resolve().parent / "data" / "l1_cache"

    def extract(
        self,
        pdf_path: Union[str, Path],
        page_range: Optional[tuple[int, int]] = None,
    ) -> ExtractionResult:
        """
        Extract content from a PDF with full provenance.

        Checks for a cached L1 result first (keyed by PDF SHA-256 hash).
        Cache saves ~18 hours on full guideline re-runs.

        Args:
            pdf_path: Path to the PDF file
            page_range: Optional (start, end) page range (1-indexed, inclusive).
                        If given, extracts only those pages into a temp PDF first.

        Returns:
            ExtractionResult with blocks, tables, markdown, and provenance
        """
        pdf_path = Path(pdf_path)
        if not pdf_path.exists():
            raise FileNotFoundError(f"PDF not found: {pdf_path}")

        # Check L1 cache (skip for page_range — those are fast subset runs)
        if page_range is None:
            cached = self._load_from_cache(pdf_path)
            if cached is not None:
                return cached

        self._ensure_model()

        # If page_range specified, extract subset into temp PDF
        if page_range:
            result = self._extract_page_range(pdf_path, page_range)
        else:
            result = self._extract_full(pdf_path, page_range=None)
            # Cache full-document results for future runs
            self._save_to_cache(pdf_path, result)

        return result

    def _cache_path_for(self, pdf_path: Path) -> Path:
        """Return the cache file path for a given PDF."""
        file_hash = self._compute_file_hash(pdf_path)
        self.L1_CACHE_DIR.mkdir(parents=True, exist_ok=True)
        return self.L1_CACHE_DIR / f"{pdf_path.stem}_{file_hash[:12]}_l1.json"

    def _save_to_cache(self, pdf_path: Path, result: ExtractionResult) -> None:
        """Serialize ExtractionResult to JSON cache."""
        cache_file = self._cache_path_for(pdf_path)
        data = {
            "blocks": [
                {
                    "text": b.text,
                    "block_type": b.block_type,
                    "page_number": b.page_number,
                    "heading_level": b.heading_level,
                    "confidence": b.confidence,
                    "bbox": (
                        {"x0": b.bbox.x0, "y0": b.bbox.y0, "x1": b.bbox.x1, "y1": b.bbox.y1}
                        if b.bbox else None
                    ),
                }
                for b in result.blocks
            ],
            "tables": [
                {
                    "headers": t.headers,
                    "rows": t.rows,
                    "page_number": t.page_number,
                    "caption": t.caption,
                    "table_index": t.table_index,
                    "region_type": t.region_type,
                    "cell_data": t.cell_data,
                    "bbox": (
                        {"x0": t.bbox.x0, "y0": t.bbox.y0, "x1": t.bbox.x1, "y1": t.bbox.y1}
                        if t.bbox else None
                    ),
                }
                for t in result.tables
            ],
            "markdown": result.markdown,
            "provenance": {
                "source_file": result.provenance.source_file,
                "source_hash": result.provenance.source_hash,
                "extraction_timestamp": result.provenance.extraction_timestamp,
                "extractor_version": result.provenance.extractor_version,
                "marker_version": result.provenance.marker_version,
                "seed": result.provenance.seed,
                "total_pages": result.provenance.total_pages,
                "extraction_params": result.provenance.extraction_params,
            },
        }
        cache_file.write_text(json.dumps(data), encoding="utf-8")
        print(f"   L1 cache saved: {cache_file} ({cache_file.stat().st_size / 1024:.0f} KB)")

    def _load_from_cache(self, pdf_path: Path) -> Optional[ExtractionResult]:
        """Load ExtractionResult from JSON cache if it exists."""
        cache_file = self._cache_path_for(pdf_path)
        if not cache_file.exists():
            return None

        print(f"   L1 CACHE HIT: {cache_file.name}")
        data = json.loads(cache_file.read_text(encoding="utf-8"))

        blocks = [
            TextBlock(
                text=b["text"],
                block_type=b["block_type"],
                page_number=b["page_number"],
                heading_level=b.get("heading_level"),
                confidence=b.get("confidence"),
                bbox=(
                    BoundingBox(x0=b["bbox"]["x0"], y0=b["bbox"]["y0"],
                                x1=b["bbox"]["x1"], y1=b["bbox"]["y1"])
                    if b.get("bbox") else None
                ),
            )
            for b in data["blocks"]
        ]
        tables = [
            TableBlock(
                headers=t.get("headers", []),
                rows=t.get("rows", []),
                page_number=t["page_number"],
                caption=t.get("caption"),
                table_index=t.get("table_index", 0),
                region_type=t.get("region_type", "table"),
                cell_data=t.get("cell_data"),
                bbox=(
                    BoundingBox(x0=t["bbox"]["x0"], y0=t["bbox"]["y0"],
                                x1=t["bbox"]["x1"], y1=t["bbox"]["y1"])
                    if t.get("bbox") else None
                ),
            )
            for t in data["tables"]
        ]
        prov = data["provenance"]
        provenance = ExtractionProvenance(
            source_file=prov["source_file"],
            source_hash=prov["source_hash"],
            extraction_timestamp=prov["extraction_timestamp"],
            extractor_version=prov["extractor_version"],
            marker_version=prov["marker_version"],
            seed=prov["seed"],
            total_pages=prov["total_pages"],
            extraction_params=prov.get("extraction_params", {}),
        )

        print(f"   Loaded from cache: {len(blocks)} blocks, {len(tables)} tables, "
              f"{len(data['markdown'])} chars markdown")
        return ExtractionResult(
            blocks=blocks, tables=tables, markdown=data["markdown"], provenance=provenance
        )

    def _extract_page_range(
        self,
        pdf_path: Path,
        page_range: tuple[int, int],
    ) -> ExtractionResult:
        """Extract a page range by creating a temp PDF with just those pages."""
        import fitz

        doc = fitz.open(pdf_path)
        total_pages = len(doc)
        start_page = max(0, page_range[0] - 1)  # 0-indexed
        end_page = min(total_pages, page_range[1])  # inclusive, 1-indexed

        # Create temp PDF with just the requested pages
        with tempfile.NamedTemporaryFile(suffix=".pdf", delete=False) as tmp:
            tmp_path = Path(tmp.name)

        new_doc = fitz.open()
        new_doc.insert_pdf(doc, from_page=start_page, to_page=end_page - 1)
        new_doc.save(str(tmp_path))
        new_doc.close()
        doc.close()

        try:
            result = self._extract_full(
                tmp_path,
                page_range=page_range,
                original_pdf_path=pdf_path,
            )
            # Remap page numbers to original document page numbers
            page_offset = start_page  # 0-indexed offset
            for block in result.blocks:
                block.page_number += page_offset
            for table in result.tables:
                table.page_number += page_offset
            return result
        finally:
            tmp_path.unlink(missing_ok=True)

    def _extract_full(
        self,
        pdf_path: Path,
        page_range: Optional[tuple[int, int]] = None,
        original_pdf_path: Optional[Path] = None,
    ) -> ExtractionResult:
        """
        Core extraction using MonkeyOCR magic_pdf API.

        Steps:
          1. Read PDF bytes → PymuDocDataset
          2. Layout analysis via DoclayoutYOLO
          3. OCR via pipe_ocr_mode (Qwen2.5-VL)
          4. Capture markdown + middle.json
          5. Convert middle.json blocks → TextBlock[] with bboxes
          6. Apply LaTeX stripping + ClinicalOCRPostProcessor
          7. Insert <!-- PAGE N --> markers
          8. Return ExtractionResult
        """
        from magic_pdf.data.data_reader_writer import (
            FileBasedDataReader,
            FileBasedDataWriter,
        )
        from magic_pdf.data.dataset import PymuDocDataset
        from magic_pdf.model.doc_analyze_by_custom_model_llm import doc_analyze_llm

        # Step 1: Read PDF
        reader = FileBasedDataReader()
        file_bytes = reader.read(str(pdf_path))
        ds = PymuDocDataset(file_bytes)

        # Step 2: Layout analysis
        infer_result = ds.apply(
            doc_analyze_llm,
            MonkeyOCR_model=self._model,
            split_pages=False,
            pred_abandon=False,
        )

        # Step 3: OCR pipeline — write images to temp dir
        with tempfile.TemporaryDirectory() as tmp_dir:
            tmp_path = Path(tmp_dir)
            image_dir = tmp_path / "images"
            image_dir.mkdir()

            image_writer = FileBasedDataWriter(str(image_dir))
            md_writer = FileBasedDataWriter(str(tmp_path))

            pipe_result = infer_result.pipe_ocr_mode(
                image_writer, MonkeyOCR_model=self._model
            )

            # Step 4: Capture markdown
            pipe_result.dump_md(md_writer, "output.md", "images")
            md_file = tmp_path / "output.md"
            markdown_text = md_file.read_text(encoding="utf-8") if md_file.exists() else ""

            # Step 5: Capture middle.json for structured blocks
            pipe_result.dump_middle_json(md_writer, "output_middle.json")
            middle_file = tmp_path / "output_middle.json"
            if middle_file.exists():
                middle_data = json.loads(middle_file.read_text(encoding="utf-8"))
            else:
                middle_data = {}

        # Step 6: Convert middle.json → TextBlock[] + TableBlock[]
        blocks, tables = self._middle_json_to_blocks(middle_data)

        # Step 7: Apply LaTeX stripping BEFORE clinical OCR post-processing
        markdown_text = self._strip_latex(markdown_text)
        for block in blocks:
            block.text = self._strip_latex(block.text)

        # Step 8: Apply ClinicalOCRPostProcessor
        ocr_correction_summary = None
        if self.ocr_postprocessor:
            blocks, block_summary = self.ocr_postprocessor.process_blocks(blocks)
            markdown_text, md_summary = self.ocr_postprocessor.process_markdown(
                markdown_text
            )
            ocr_correction_summary = {
                "blocks": block_summary,
                "markdown": md_summary,
                "total_corrections": (
                    block_summary["total_corrections"]
                    + md_summary["total_corrections"]
                ),
            }

        # Step 9: Insert <!-- PAGE N --> markers into markdown
        markdown_text = self._insert_page_markers(markdown_text, middle_data)

        # Step 10: Build provenance
        source_path = original_pdf_path or pdf_path
        total_pages = len(middle_data.get("pdf_info", []))
        if total_pages == 0:
            total_pages = max((b.page_number for b in blocks), default=1)

        extraction_params = {
            "parser": "monkeyocr",
            "config": self._config_path,
            "page_range": list(page_range) if page_range else None,
            "ocr_postprocessing_enabled": self.enable_ocr_postprocessing,
            "latex_stripping_enabled": True,
        }
        if ocr_correction_summary:
            extraction_params["ocr_corrections"] = {
                "total": ocr_correction_summary["total_corrections"],
                "by_type": ocr_correction_summary["blocks"].get(
                    "correction_types", {}
                ),
            }

        provenance = ExtractionProvenance(
            source_file=str(source_path),
            source_hash=self._compute_file_hash(source_path),
            extraction_timestamp=datetime.now(timezone.utc).isoformat(),
            extractor_version=self.VERSION,
            marker_version="monkeyocr",
            seed=self.seed,
            total_pages=total_pages,
            extraction_params=extraction_params,
        )

        return ExtractionResult(
            blocks=blocks,
            tables=tables,
            markdown=markdown_text,
            provenance=provenance,
        )

    # ═══════════════════════════════════════════════════════════════════════════
    # middle.json → TextBlock/TableBlock conversion
    # ═══════════════════════════════════════════════════════════════════════════

    # MonkeyOCR type → TextBlock.block_type mapping
    _TYPE_MAP: dict[str, Literal["text", "table", "heading", "list", "code", "image_caption"]] = {
        "title": "heading",
        "text": "text",
        "table": "table",
        "image": "image_caption",
        "image_body": "image_caption",
        "image_caption": "image_caption",
        "table_caption": "text",
        "table_footnote": "text",
        "header": "text",
        "footer": "text",
        "reference": "text",
        "equation": "text",
        "abandoned": "text",
    }

    def _middle_json_to_blocks(
        self, middle_data: dict
    ) -> tuple[list[TextBlock], list[TableBlock]]:
        """
        Convert MonkeyOCR middle.json structure to TextBlock[] and TableBlock[].

        middle.json structure:
            pdf_info: [
                {  # page 0
                    preproc_blocks: [
                        {
                            type: "title"|"text"|"table"|"image"|...,
                            bbox: [x0, y0, x1, y1],
                            lines: [{
                                bbox: [...],
                                spans: [{
                                    bbox: [...],
                                    score: 0.95,
                                    content: "text content",
                                    type: "text"
                                }]
                            }],
                            index: 0
                        }
                    ]
                }
            ]
        """
        blocks: list[TextBlock] = []
        tables: list[TableBlock] = []
        table_index = 0

        pdf_info = middle_data.get("pdf_info", [])
        for page_idx, page_data in enumerate(pdf_info):
            page_number = page_idx + 1  # 1-indexed

            for preproc_block in page_data.get("preproc_blocks", []):
                block_type_raw = preproc_block.get("type", "text")
                bbox_raw = preproc_block.get("bbox", [0, 0, 0, 0])

                # Build bounding box
                bbox = BoundingBox(
                    x0=float(bbox_raw[0]),
                    y0=float(bbox_raw[1]),
                    x1=float(bbox_raw[2]),
                    y1=float(bbox_raw[3]),
                )

                # Extract text from lines → spans → content
                text_parts: list[str] = []
                min_confidence = 1.0

                for line in preproc_block.get("lines", []):
                    line_parts: list[str] = []
                    for span in line.get("spans", []):
                        content = span.get("content", "")
                        if content:
                            line_parts.append(content)
                        score = span.get("score", 1.0)
                        if score < min_confidence:
                            min_confidence = score
                    if line_parts:
                        text_parts.append(" ".join(line_parts))

                text = "\n".join(text_parts).strip()
                if not text:
                    continue

                # Map type
                mapped_type = self._TYPE_MAP.get(block_type_raw, "text")

                # Heading level detection
                heading_level = None
                if mapped_type == "heading":
                    heading_level = 1  # MonkeyOCR "title" = heading level 1

                # Handle table blocks — capture per-cell data with span-level bboxes
                if mapped_type == "table":
                    table_data = self._parse_table_text(text)
                    if table_data and len(table_data) > 1:
                        headers = table_data[0]
                        rows = table_data[1:]
                        cell_data = self._extract_table_cell_data(preproc_block)
                        tables.append(
                            TableBlock(
                                headers=headers,
                                rows=rows,
                                page_number=page_number,
                                bbox=bbox,
                                confidence=min_confidence,
                                table_index=table_index,
                                cell_data=cell_data,
                                region_type=block_type_raw,
                            )
                        )
                        table_index += 1

                blocks.append(
                    TextBlock(
                        text=text,
                        page_number=page_number,
                        block_type=mapped_type,
                        bbox=bbox,
                        confidence=min_confidence,
                        heading_level=heading_level,
                        region_type=block_type_raw,
                        seed=self.seed,
                    )
                )

        return blocks, tables

    def _extract_table_cell_data(self, preproc_block: dict) -> list[dict]:
        """Extract per-cell data with span-level bboxes from a MonkeyOCR table block.

        MonkeyOCR's middle.json stores each recognized text region as a span with
        an individual bbox from Qwen2.5-VL's grounded OCR output. For table blocks:
          lines[row_idx].spans[col_idx].bbox  = the cell's page-coordinate bbox
          lines[row_idx].spans[col_idx].content = the cell text
          lines[row_idx].spans[col_idx].score   = OCR confidence

        These per-cell bboxes are the per-fact geometry that Feature 2 needs.
        """
        cells: list[dict] = []
        for row_idx, line in enumerate(preproc_block.get("lines", [])):
            for col_idx, span in enumerate(line.get("spans", [])):
                content = span.get("content", "").strip()
                if not content:
                    continue
                cells.append({
                    "text": content,
                    "row_idx": row_idx,
                    "col_idx": col_idx,
                    "bbox": span.get("bbox", [0.0, 0.0, 0.0, 0.0]),
                    "confidence": float(span.get("score", 1.0)),
                })
        return cells

    def _parse_table_text(self, text: str) -> Optional[list[list[str]]]:
        """Parse markdown table text into rows of cells."""
        lines = text.strip().split("\n")
        rows: list[list[str]] = []
        for line in lines:
            line = line.strip()
            if not line:
                continue
            # Skip separator lines (|---|---|)
            if line.startswith("|") and all(
                c in "-| " for c in line
            ):
                continue
            if "|" in line:
                cells = [cell.strip() for cell in line.split("|")]
                # Remove empty first/last from leading/trailing |
                if cells and cells[0] == "":
                    cells = cells[1:]
                if cells and cells[-1] == "":
                    cells = cells[:-1]
                if cells:
                    rows.append(cells)
        return rows if rows else None

    # ═══════════════════════════════════════════════════════════════════════════
    # LaTeX stripping — specific to MonkeyOCR output
    # ═══════════════════════════════════════════════════════════════════════════

    # Compiled LaTeX patterns for efficiency
    _LATEX_SUPERSCRIPT_RE = re.compile(r'\$\^\{([^}]*)\}\$')
    _LATEX_SUBSCRIPT_RE = re.compile(r'\$_\{([^}]*)\}\$')
    _LATEX_GEQ_RE = re.compile(r'\$\\geq\$')
    _LATEX_LEQ_RE = re.compile(r'\$\\leq\$')
    _LATEX_TIMES_RE = re.compile(r'\$\\times\$')
    _LATEX_PM_RE = re.compile(r'\$\\pm\$')
    _LATEX_APPROX_RE = re.compile(r'\$\\approx\$')
    _LATEX_INLINE_RE = re.compile(r'\$([^$]+)\$')

    # Subscript Unicode mapping
    _SUBSCRIPT_MAP = {
        "0": "\u2080", "1": "\u2081", "2": "\u2082", "3": "\u2083",
        "4": "\u2084", "5": "\u2085", "6": "\u2086", "7": "\u2087",
        "8": "\u2088", "9": "\u2089",
    }

    # Superscript Unicode mapping
    _SUPERSCRIPT_MAP = {
        "0": "\u2070", "1": "\u00B9", "2": "\u00B2", "3": "\u00B3",
        "4": "\u2074", "5": "\u2075", "6": "\u2076", "7": "\u2077",
        "8": "\u2078", "9": "\u2079",
    }

    @classmethod
    def _strip_latex(cls, text: str) -> str:
        """
        Strip LaTeX notation from MonkeyOCR output and convert to Unicode.

        Conversions:
          $^{N}$    → superscript Unicode or plain N (footnote refs: $^{57}$ → 57)
          $_{N}$    → subscript Unicode ($_{2}$ → ₂)
          $\\geq$   → ≥
          $\\leq$   → ≤
          $\\times$  → ×
          $\\pm$     → ±
          $\\approx$ → ≈
          1.73m$^{2}$ → 1.73m²
        """
        if "$" not in text:
            return text

        # Comparators first (most common in clinical text)
        text = cls._LATEX_GEQ_RE.sub("≥", text)
        text = cls._LATEX_LEQ_RE.sub("≤", text)
        text = cls._LATEX_TIMES_RE.sub("×", text)
        text = cls._LATEX_PM_RE.sub("±", text)
        text = cls._LATEX_APPROX_RE.sub("≈", text)

        # Superscripts: $^{2}$ → ² (single digit), $^{57}$ → 57 (multi-digit footnote refs)
        def _replace_superscript(m):
            inner = m.group(1)
            if len(inner) == 1 and inner in cls._SUPERSCRIPT_MAP:
                return cls._SUPERSCRIPT_MAP[inner]
            # Multi-character (footnote references like $^{57}$) → plain text
            return inner

        text = cls._LATEX_SUPERSCRIPT_RE.sub(_replace_superscript, text)

        # Subscripts: $_{2}$ → ₂
        def _replace_subscript(m):
            inner = m.group(1)
            result = []
            for ch in inner:
                if ch in cls._SUBSCRIPT_MAP:
                    result.append(cls._SUBSCRIPT_MAP[ch])
                else:
                    result.append(ch)
            return "".join(result)

        text = cls._LATEX_SUBSCRIPT_RE.sub(_replace_subscript, text)

        # Remaining inline LaTeX: $...$ → strip delimiters, keep content
        # This catches edge cases like $\alpha$ → \alpha (raw, but no $ wrappers)
        text = cls._LATEX_INLINE_RE.sub(lambda m: m.group(1), text)

        return text

    # ═══════════════════════════════════════════════════════════════════════════
    # Page marker insertion
    # ═══════════════════════════════════════════════════════════════════════════

    def _insert_page_markers(self, markdown: str, middle_data: dict) -> str:
        """
        Insert <!-- PAGE N --> markers into markdown based on middle.json page info.

        MonkeyOCR markdown doesn't include page markers by default.
        We insert them using the page structure from pdf_info[].
        """
        pdf_info = middle_data.get("pdf_info", [])
        num_pages = len(pdf_info)

        if num_pages <= 1:
            # Single page or no page info: prepend a single marker
            return f"<!-- PAGE 1 -->\n{markdown}"

        # For multi-page documents, we need to split the markdown at page boundaries.
        # Strategy: use the text from each page's preproc_blocks to find split points.
        # The markdown is a concatenation of all pages' text in order.
        result_parts: list[str] = []
        remaining_md = markdown

        for page_idx in range(num_pages):
            page_number = page_idx + 1
            result_parts.append(f"\n<!-- PAGE {page_number} -->\n")

            if page_idx < num_pages - 1:
                # Find the first unique text of the NEXT page to split on
                next_page = pdf_info[page_idx + 1]
                split_text = self._get_page_split_anchor(next_page)

                if split_text and split_text in remaining_md:
                    split_idx = remaining_md.index(split_text)
                    result_parts.append(remaining_md[:split_idx])
                    remaining_md = remaining_md[split_idx:]
                else:
                    # Can't find split point — dump everything here
                    # (happens with single-page extraction or short pages)
                    pass
            else:
                # Last page: append all remaining
                result_parts.append(remaining_md)
                remaining_md = ""

        # If any remaining text wasn't consumed
        if remaining_md:
            result_parts.append(remaining_md)

        return "".join(result_parts)

    def _get_page_split_anchor(self, page_data: dict) -> Optional[str]:
        """Get the first non-trivial text from a page to use as a split anchor."""
        for block in page_data.get("preproc_blocks", []):
            for line in block.get("lines", []):
                for span in line.get("spans", []):
                    content = span.get("content", "").strip()
                    # Skip very short anchors (risk of false matches)
                    if len(content) >= 20:
                        return content
        return None
