# V4.1 Hybrid Architecture: Marker L1 + Granite Channel A + Ollama Channel F + L1 Completeness Checker

> **This document is an ARCHITECTURAL AMENDMENT to [V4_MULTI_CHANNEL_PIPELINE_PLAN.md](V4_MULTI_CHANNEL_PIPELINE_PLAN.md).**
> It changes **L1 strategy**, **Channel A internals**, **Channel D table parsing**, **Channel F deployment model**, and adds an **L1 Completeness Checker** safety layer.
> **Everything else in V4 stays exactly the same**: Channels B/C/E, Dossier Assembly, Pipeline 2 (L3→L7), KB schemas.

---

## Why This Amendment Exists

Four problems emerged during V4 empirical testing and architectural review on KDIGO 2022 Diabetes-CKD:

| Problem | Evidence | Root Cause |
|---------|----------|------------|
| **0 tables detected** by Docling StandardPipeline | Pipeline 1 run: Docling found 0 tables vs Marker's 1 | TableFormer + Channel A regex mismatch on Docling's table output format |
| **Channel F OOM on CPU** | NuExtract 1.5 (3.8B): 19GB/21.7GB RAM (87.5%), 30+ min for 2 pages, Docker daemon died | Loading a 3.8B model inside Docker on CPU is fundamentally non-viable |
| **NuExtract 2.0 incompatible** | `ValueError: Qwen2_5_VLConfig` — NuExtract 2.0 series migrated to Qwen2.5-VL architecture | `AutoModelForCausalLM` cannot load vision-language models |
| **Irrecoverable L1 miss class** | If Marker drops a paragraph, no channel sees it, no span is created, no reviewer can review it — silently lost | Single L1 extraction = single point of failure for text completeness |

The V4.1 hybrid architecture solves all four with three design decisions:
1. **Marker L1 + Granite-Docling Channel A** — best text quality + richest structural understanding
2. **Ollama sidecar for Channel F** — NuExtract 2.0-8B on Apple Silicon Metal GPU, not Docker CPU
3. **L1 Completeness Checker** — cross-reference Marker text against Granite-Docling DocTags to detect and recover missed content

---

## The Three Design Decisions

### Decision 1: Separation of Text and Structure

**Principle**: Text extraction quality and structural understanding are different concerns. Use each model for what it's best at.

| Concern | Best Model | Why |
|---------|-----------|-----|
| **Text quality** (clean markdown, ligature-free, proven) | Marker | 232 spans on KDIGO, Channel 0 handles ligatures, Channels B/C/E/F already work on it |
| **Structural understanding** (sections, tables, footnotes, captions, hierarchy) | Granite-Docling 258M VLM | 15+ DocTags semantic types, OTSL table format, native footnote/caption detection |

**Why not use one model for both?**
- **Marker-only L1**: Good text but Channel A can only extract 4 structural types via regex (heading, table, list, recommendation). Found 0 tables via Docling StandardPipeline.
- **Granite-Docling-only L1**: Rich structure (15+ DocTags types, OTSL tables), but its markdown text export is untested against our 7 channels. Channels B-F are validated against Marker's markdown.
- **Hybrid**: Marker provides the text-of-record. Granite-Docling provides the structural oracle. Each operates on the original PDF independently. Combined at the GuidelineTree abstraction layer.

### Decision 2: Ollama Sidecar for Heavy Models

**Principle**: Heavy ML models (>500M params) run as native services on Apple Silicon, not inside Docker on CPU.

| Approach | Model | Memory | Speed | Outcome |
|----------|-------|--------|-------|---------|
| Docker + transformers (tried) | NuExtract 1.5 (3.8B) | 19GB/21.7GB | 30+ min/2 pages | Docker daemon died |
| Docker + transformers (wanted) | NuExtract 2.0-8B | N/A | N/A | `Qwen2_5_VLConfig` error |
| **Ollama on M4 Mac Mini** | **NuExtract 2.0-8B Q4_K_M** | **~6GB VRAM** | **3-4 t/s, ~2.5 min/page** | **Viable** |

Why Ollama works:
- Apple Silicon unified memory → Metal GPU acceleration (not CPU brute-force)
- GGUF quantization → 8B model in ~6GB (Q4_K_M)
- llama.cpp supports Qwen2.5 architecture natively (no `AutoModelForCausalLM` issue)
- Process isolation → no RAM competition with Docker pipeline
- NuExtract 2.0-8B beats GPT-4.1 on proposition extraction (vs 1.5 which is Phi-3-mini)

---

## V4.1 Pipeline Architecture

```
Mac Mini M4 (24GB unified memory)
│
├── Ollama (native, Metal GPU)                     ◄── ~6GB persistent
│   └── NuExtract 2.0-8B Q4_K_M
│       Endpoint: localhost:11434
│       Temperature: 0 (clinical extraction)
│       Context: 8192 tokens
│
└── Docker Container (V4.1 Pipeline)               ◄── ~4-5GB peak
    │
    ├── INPUTS
    │   └── PDF (e.g., KDIGO-2022-Diabetes-CKD.pdf)
    │
    ├── L1: Marker                                 ◄── TEXT OF RECORD
    │   └── PDF → markdown (consumed by Ch.0 → all channels)
    │
    ├── Channel 0: Text Normalizer                 ◄── UNCHANGED
    │   └── Ligatures, unicode, OCR fixes on Marker markdown
    │
    ├── Channel A: Granite-Docling Structural Oracle ◄── REWRITTEN (was regex-only)
    │   ├── Runs Granite-Docling VlmPipeline on ORIGINAL PDF → DocTags
    │   ├── Extracts: sections, headings, footnotes, captions, tables (OTSL)
    │   ├── Aligns structural elements to Marker markdown offsets
    │   ├── Fallback: regex parsing if alignment confidence < 80%
    │   └── Produces: GuidelineTree (richer than V4 regex-only tree)
    │
    │   ┌─── Parallel (consume normalized_text + GuidelineTree) ───┐
    │   │                                                           │
    │   ├── Channel B: Drug Dictionary (Aho-Corasick)   UNCHANGED  │
    │   ├── Channel C: Grammar/Regex Patterns            UNCHANGED  │
    │   ├── Channel D: Table Decomposer                  UPDATED    │
    │   │   ├── source="marker_pipe" → pipe table path (current)    │
    │   │   └── source="granite_otsl" → OTSL path (NEW)            │
    │   ├── Channel E: GLiNER Residual                   UNCHANGED  │
    │   └── Channel F: NuExtract via Ollama              REWRITTEN  │
    │       └── HTTP client → host.docker.internal:11434            │
    │                                                               │
    │   └───────────────────────────────────────────────────────────┘
    │
    ├── Signal Merger                               ◄── UNCHANGED
    ├── DB: l2_* tables                             ◄── UNCHANGED
    ├── Reviewer UI                                 ◄── UNCHANGED
    ├── Dossier Assembly                            ◄── UNCHANGED
    └── Pipeline 2 (L3→L7)                          ◄── UNCHANGED
```

### Data Flow Diagram

```
                    ┌────────────────┐
                    │  PDF (KDIGO)   │
                    └───────┬────────┘
                            │
              ┌─────────────┼─────────────┐
              │             │             │
              ▼             │             ▼
    ┌─────────────────┐     │   ┌──────────────────────┐
    │  Marker (L1)    │     │   │  Granite-Docling      │
    │  PDF → markdown │     │   │  PDF → DocTags        │
    │  (text quality) │     │   │  (structural oracle)  │
    └────────┬────────┘     │   └──────────┬───────────┘
             │              │              │
             ▼              │              │
    ┌─────────────────┐     │              │
    │  Channel 0      │     │              │
    │  (normalize)    │     │              │
    └────────┬────────┘     │              │
             │              │              │
             ▼              ▼              ▼
    ┌──────────────────────────────────────────────┐
    │  Channel A: Structural Oracle                 │
    │  ┌─────────────────────────────────────────┐ │
    │  │ DocTags structure + Marker text offsets  │ │
    │  │ → GuidelineTree (sections, tables,      │ │
    │  │   footnotes, captions, OTSL tables)     │ │
    │  └─────────────────────────────────────────┘ │
    └─────────────┬────────────────────────────────┘
                  │
    ┌─────────────┼─────────────────────────────┐
    │             │             │         │      │
    ▼             ▼             ▼         ▼      ▼
  Ch.B          Ch.C          Ch.D     Ch.E    Ch.F
  (drugs)       (regex)       (tables)  (NER)  (Ollama)
    │             │             │         │      │
    └─────────────┴──────┬──────┴─────────┘      │
                         │                        │
                         ▼                        │
                  Signal Merger  ◄────────────────┘
                         │
                         ▼
                  Reviewer Queue
```

---

## What Changes vs V4 Original Plan

| Component | V4 (Original) | V4.1 (This Plan) | Change Level |
|-----------|--------------|-------------------|--------------|
| **L1** | Docling StandardPipeline (sole L1) | **Marker** (text) + **Granite-Docling** (structure, in Channel A) | **MAJOR** |
| **Channel 0** | Normalizer on Docling markdown | Normalizer on Marker markdown | NONE (same logic) |
| **Channel A** | Regex-only: ATX headings + pipe tables + page markers | **Granite-Docling structural oracle** + Marker offset alignment + regex fallback | **MAJOR REWRITE** |
| **Channel B** | Aho-Corasick on normalized text | Same | NONE |
| **Channel C** | Regex patterns on normalized text | Same | NONE |
| **Channel D** | Pipe table decomposer only | Pipe table + **OTSL decomposer** (dual-source) | **MODERATE** |
| **Channel E** | GLiNER on normalized text | Same | NONE |
| **Channel F** | NuExtract 2.0-4B in-process (Docker/CPU) | **NuExtract 2.0-8B via Ollama** (native/Metal GPU) | **MAJOR REWRITE** |
| Signal Merger | Overlap clustering + confidence boost | Same | NONE |
| Reviewer UI | FastAPI + Angular SPA | Same | NONE |
| Dossier Assembly | Per-drug grouping from verified spans | Same | NONE |
| Pipeline 2 | L3→L7 on per-drug dossiers | Same | NONE |
| DB Schema | l2_* tables in factstore | Same | NONE |
| Docker image | Single container with all models | **Lighter** (no NuExtract weights, Granite-Docling 258M only) | SMALLER |

**5 of 7 channels completely untouched. 2 channels rewritten. 1 channel updated.**

---

## Channel A Rewrite: Granite-Docling Structural Oracle

### Why Rewrite

The V4 Channel A ([channel_a_docling.py](backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/channel_a_docling.py)) parses markdown with 4 regex patterns:
- `HEADING_RE`: ATX headings (`# through ######`)
- `TABLE_ROW_RE`: Pipe table rows (`|...|`)
- `TABLE_SEP_RE`: Pipe table separators (`|---|---|`)
- `PAGE_MARKER_RE`: `<!-- PAGE N -->`

This produces a GuidelineTree with ~4 block types. **Empirically, it found 0 tables on KDIGO via Docling StandardPipeline** because the regex patterns didn't match Docling's table output format.

The V4.1 Channel A runs Granite-Docling VlmPipeline on the original PDF, getting DocTags with 15+ semantic types including `<footnote>`, `<caption>`, `<ordered_list>`, `<unordered_list>`, `<otsl>`, `<section_header>`, and per-element bounding boxes.

### Architecture

Channel A now has **TWO inputs**:
1. `normalized_text` — from Marker → Channel 0 (for offset computation)
2. `pdf_path` — original PDF (for Granite-Docling VlmPipeline)

```python
class ChannelAStructuralOracle:
    """Granite-Docling structural oracle with Marker text alignment.

    Uses Granite-Docling VlmPipeline (258M) for STRUCTURAL understanding:
    - Section hierarchy from DocTags nesting
    - Table detection via OTSL format
    - Footnote, caption, list type discrimination

    Uses Marker's normalized markdown as TEXT OF RECORD:
    - All offsets reference positions in Marker's markdown
    - Channels B-F operate on Marker's text

    Falls back to regex-only parsing if:
    - Granite-Docling is unavailable
    - Alignment confidence drops below 80%
    """

    VERSION = "4.1.0"
    ALIGNMENT_THRESHOLD = 0.80  # minimum heading match ratio

    def parse(self, normalized_text: str, pdf_path: str) -> GuidelineTree:
        """Parse structure using Granite-Docling, aligned to Marker text.

        Args:
            normalized_text: Channel 0 output (Marker markdown, normalized)
            pdf_path: Original PDF path (for Granite-Docling processing)

        Returns:
            GuidelineTree with rich structure and Marker-space offsets
        """
        try:
            doctags = self._run_granite_docling(pdf_path)
            tree = self._align_doctags_to_text(doctags, normalized_text)

            if tree.alignment_confidence >= self.ALIGNMENT_THRESHOLD:
                return tree
            else:
                # Alignment too low — fall back to regex
                return self._parse_markdown_regex(normalized_text)
        except Exception:
            # Granite-Docling unavailable — fall back to regex
            return self._parse_markdown_regex(normalized_text)

    def _run_granite_docling(self, pdf_path: str) -> DocTagsResult:
        """Run Granite-Docling VlmPipeline on PDF → DocTags output."""
        from docling.document_converter import DocumentConverter, PdfFormatOption
        from docling.pipeline.vlm_pipeline import VlmPipeline
        from docling.datamodel.pipeline_options import VlmPipelineOptions

        converter = DocumentConverter(
            format_options={
                InputFormat.PDF: PdfFormatOption(
                    pipeline_cls=VlmPipeline,
                    pipeline_options=VlmPipelineOptions(),
                )
            }
        )
        doc = converter.convert(pdf_path).document
        return self._extract_doctags(doc)

    def _parse_markdown_regex(self, normalized_text: str) -> GuidelineTree:
        """V4 fallback: regex-only parsing of Marker markdown.
        Same logic as current channel_a_docling.py (ATX headings + pipe tables).
        """
        # ... existing regex implementation preserved as fallback ...
```

### Three Alignment Strategies

| Element Type | Strategy | Confidence |
|-------------|----------|------------|
| **Headings** | Text-match DocTags heading text against Marker ATX headings (strip `#` prefix) | HIGH — heading text is unique per document |
| **Footnotes/Captions/Lists** | Metadata enrichment only — add `block_type` tag to existing sections, no offset alignment needed | HIGH — no alignment required |
| **Tables** | Match OTSL header cells against Marker pipe table headers. If Marker has table → use Marker offsets. If Marker missed table → store OTSL text separately. | MEDIUM — header text matching |

#### Heading Alignment (High Confidence)

```python
def _align_heading(self, doctag_heading: str, normalized_text: str) -> Optional[int]:
    """Find DocTags heading text in Marker markdown.

    DocTags:  <section_header>Chapter 4: Glucose-Lowering Therapies</section_header>
    Marker:   ## Chapter 4: Glucose-Lowering Therapies

    Strategy: Find heading text in normalized_text, compute offset.
    """
    # Exact match first
    idx = normalized_text.find(doctag_heading)
    if idx >= 0:
        return idx

    # Fuzzy match (handle minor OCR differences)
    # Use difflib.SequenceMatcher with ratio > 0.85
    ...
```

#### Footnote/Caption/List Enrichment (No Alignment Needed)

```python
def _enrich_section_types(self, tree: GuidelineTree, doctags: DocTagsResult):
    """Add DocTags block types to existing sections.

    DocTags provide <footnote>, <caption>, <ordered_list>, <unordered_list>
    that Marker's markdown doesn't distinguish.

    This is METADATA ENRICHMENT — no offset recalculation needed.
    """
    for section in tree.all_sections():
        doctag_type = self._find_matching_doctag_type(section.heading, doctags)
        if doctag_type:
            section.block_type = doctag_type  # "footnote", "caption", etc.
```

#### Table Alignment (Medium Confidence — The Critical Case)

```python
def _align_tables(self, doctags: DocTagsResult, normalized_text: str) -> list[TableBoundary]:
    """Align Granite-Docling tables to Marker markdown.

    Three outcomes per table:
    1. Marker has matching pipe table → use Marker offsets (source="marker_pipe")
    2. Marker missed the table → store OTSL text (source="granite_otsl")
    3. No match in either → skip (logged as warning)
    """
    tables = []

    for otsl_table in doctags.tables:
        otsl_headers = otsl_table.column_headers  # from <ched> tags

        # Try to find matching pipe table in Marker markdown
        marker_table = self._find_pipe_table_by_headers(otsl_headers, normalized_text)

        if marker_table:
            # Marker has this table — use Marker offsets
            tables.append(TableBoundary(
                start_offset=marker_table.start,
                end_offset=marker_table.end,
                headers=otsl_headers,
                source="marker_pipe",
                page_number=otsl_table.page,
            ))
        else:
            # Marker missed this table — store OTSL text separately
            tables.append(TableBoundary(
                start_offset=-1,  # not in Marker text
                end_offset=-1,
                headers=otsl_headers,
                source="granite_otsl",
                otsl_text=otsl_table.raw_otsl,  # NEW field
                page_number=otsl_table.page,
            ))

    return tables
```

### Updated GuidelineTree Model

```python
@dataclass
class TableBoundary:
    """A table detected in the guideline document.

    V4.1: Tables can come from two sources:
    - "marker_pipe": Marker markdown pipe tables (offsets in normalized_text)
    - "granite_otsl": Granite-Docling OTSL tables (offsets are -1, text in otsl_text)
    """
    table_id: str
    section_id: str
    start_offset: int         # -1 if source="granite_otsl"
    end_offset: int           # -1 if source="granite_otsl"
    headers: list[str]
    row_count: int
    page_number: int
    source: str = "marker_pipe"    # NEW: "marker_pipe" | "granite_otsl"
    otsl_text: Optional[str] = None  # NEW: raw OTSL text if source="granite_otsl"

@dataclass
class GuidelineSection:
    """A section in the guideline document.

    V4.1: block_type is now enriched by DocTags — may include:
    "footnote", "caption", "ordered_list", "unordered_list"
    in addition to V4 types: "heading", "paragraph", "table", "list_item"
    """
    section_id: str
    heading: str
    start_offset: int
    end_offset: int
    page_number: int
    block_type: str  # V4.1: expanded type set from DocTags
    children: list["GuidelineSection"]

@dataclass
class GuidelineTree:
    sections: list[GuidelineSection]
    tables: list[TableBoundary]
    total_pages: int
    alignment_confidence: float = 1.0  # NEW: ratio of headings successfully aligned
    structural_source: str = "regex"   # NEW: "granite_doctags" | "regex" (fallback)
```

### File Changes for Channel A

| Action | File | Description |
|--------|------|-------------|
| **REWRITE** | `extraction/v4/channel_a_docling.py` | Granite-Docling structural oracle + alignment logic + regex fallback |
| **UPDATE** | `extraction/v4/models.py` | Add `source`, `otsl_text` to TableBoundary; add `alignment_confidence`, `structural_source` to GuidelineTree |
| **NEW** | `extraction/v4/granite_docling_extractor.py` | Granite-Docling VlmPipeline wrapper (DocTags extraction) |
| UPDATE | `tools/guideline-atomiser/requirements.txt` | Ensure `docling` with VlmPipeline support |

---

## Channel D Update: Dual Table Source

### Why Update

Channel D ([channel_d_table.py](backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/channel_d_table.py)) currently only decomposes markdown pipe tables. With V4.1, tables may come from two sources:

1. **marker_pipe**: Marker found a pipe table → Channel D decomposes it from `text[start:end]` (current behavior)
2. **granite_otsl**: Granite-Docling found a table that Marker missed → Channel D decomposes OTSL text from `table.otsl_text`

### Updated Channel D

```python
class ChannelDTableDecomposer:
    """Decompose tables from either pipe format or OTSL format.

    V4.1: Dual-source table decomposition.
    - marker_pipe: text[table.start:table.end] → split by | delimiters
    - granite_otsl: table.otsl_text → parse <ched>/<rhed>/<fcel>/<ecel>/<lcel> tags
    """

    VERSION = "4.1.0"
    CONFIDENCE_PIPE = 0.95   # Marker pipe tables
    CONFIDENCE_OTSL = 0.92   # Granite OTSL tables (slightly lower — alignment uncertainty)

    def extract(self, text: str, tree: GuidelineTree) -> ChannelOutput:
        spans = []
        for table in tree.tables:
            if table.source == "marker_pipe":
                table_spans = self._decompose_pipe_table(text, table)
            elif table.source == "granite_otsl":
                table_spans = self._decompose_otsl_table(table)
            else:
                continue
            spans.extend(table_spans)
        return ChannelOutput(channel="D", spans=spans, ...)

    def _decompose_pipe_table(self, text: str, table: TableBoundary) -> list[RawSpan]:
        """Current V4 implementation — decompose markdown pipe table."""
        table_text = text[table.start_offset:table.end_offset]
        # ... existing pipe table logic ...

    def _decompose_otsl_table(self, table: TableBoundary) -> list[RawSpan]:
        """NEW: Decompose OTSL table from Granite-Docling DocTags.

        OTSL tokens:
        - <ched>: column header cell
        - <rhed>: row header cell
        - <fcel>: filled cell (has content)
        - <ecel>: empty cell
        - <lcel>: left-merged cell (spans from previous column)
        - <nl>: row delimiter

        Each non-empty cell becomes one RawSpan.
        """
        otsl_text = table.otsl_text
        rows = otsl_text.split("<nl>")
        headers = []
        spans = []

        for row_idx, row in enumerate(rows):
            cells = re.findall(r'<(ched|rhed|fcel|ecel|lcel)>(.*?)</\1>', row)

            if row_idx == 0:
                # Header row
                headers = [cell_text.strip() for cell_type, cell_text in cells
                          if cell_type == "ched"]

            for col_idx, (cell_type, cell_text) in enumerate(cells):
                if cell_type in ("ecel", "lcel"):
                    continue  # skip empty and merged cells

                cell_text = cell_text.strip()
                if not cell_text:
                    continue

                col_header = headers[col_idx] if col_idx < len(headers) else ""
                row_drug = self._get_row_drug(cells)  # column 0 text

                spans.append(RawSpan(
                    channel="D",
                    text=cell_text,
                    start=-1,              # not in Marker text space
                    end=-1,
                    confidence=self.CONFIDENCE_OTSL,
                    page_number=table.page_number,
                    section_id=table.section_id,
                    table_id=table.table_id,
                    source_block_type="table_cell",
                    channel_metadata={
                        "row_index": row_idx,
                        "col_index": col_idx,
                        "col_header": col_header,
                        "row_drug": row_drug,
                        "table_source": "granite_otsl",
                        "cell_type": cell_type,
                    },
                ))

        return spans

    def _is_suspicious(self, table: TableBoundary, spans: list[RawSpan]) -> bool:
        """Flag tables where decomposition might have failed.

        Suspicion heuristics:
        - Inconsistent column counts across rows
        - Zero cells extracted from a non-empty table
        - More than 50% empty/merged cells (complex table structure)
        """
        if len(spans) == 0:
            return True
        col_counts = set()
        for span in spans:
            col_counts.add(span.channel_metadata.get("col_index", 0))
        if len(col_counts) == 1 and table.row_count > 2:
            return True  # all cells in one column — likely parsing failure
        return False
```

### OTSL Table Spans and Signal Merger

OTSL-sourced spans have `start=-1, end=-1` because they don't exist in Marker's markdown text. The Signal Merger handles this:

- **Overlap clustering**: OTSL spans cluster by `table_id + row_index` instead of character overlap
- **Section assignment**: Uses `table.section_id` (set by Channel A during alignment) instead of offset ranges
- **Confidence boost**: OTSL table cells can still participate in multi-channel agreement (e.g., Channel B finds "Metformin" in prose AND Channel D finds "Metformin" in an OTSL table cell)

### File Changes for Channel D

| Action | File | Description |
|--------|------|-------------|
| **UPDATE** | `extraction/v4/channel_d_table.py` | Add `_decompose_otsl_table()`, `_is_suspicious()`, dual-source routing |
| **UPDATE** | `extraction/v4/signal_merger.py` | Handle OTSL spans with `start=-1` (cluster by table_id+row_index) |

---

## Channel F Rewrite: Ollama Sidecar

### Why Rewrite

The V4 Channel F ([channel_f_nuextract.py](backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/channel_f_nuextract.py)) loads NuExtract in-process:

```python
# V4 (BROKEN): 60 lines of model loading that fails on Docker CPU
def _load_model(self, model_path):
    # Path 1: llama-cpp-python (GGUF) — works but no GGUF available for 2.0
    # Path 2: transformers (AutoModelForCausalLM) — Qwen2_5_VLConfig error for 2.0
    # Path 3: NuExtract 1.5 works but OOMs at 3.8B on Docker CPU
```

V4.1 replaces this with a lightweight HTTP client calling Ollama:

```python
# V4.1: Channel F is just an HTTP client
def _check_ollama(self):
    response = requests.get(f"{self.ollama_url}/api/tags")
    # If Ollama is running and model is loaded → available = True

def _run_inference(self, prompt):
    response = requests.post(f"{self.ollama_url}/api/generate", json={...})
    return response.json()["response"]
```

### Updated Channel F

```python
class ChannelFNuExtract:
    """NuExtract 2.0-8B proposition extractor via Ollama sidecar.

    V4.1: Model runs as a native Ollama service on Apple Silicon,
    NOT inside the Docker container. Channel F is an HTTP client.

    Ollama provides:
    - Metal GPU acceleration on Apple Silicon (3-4 tokens/sec)
    - GGUF Q4_K_M quantization (8B model in ~6GB VRAM)
    - Native Qwen2.5 architecture support (no transformers compatibility issue)
    - Process isolation (no RAM competition with pipeline)

    Operational contract (UNCHANGED from V4):
    - Temperature: 0 (extraction, not generative)
    - Invocation threshold: Prose blocks only, >15 words
    - Block types processed: paragraph, list_item only
    - Passthrough: Elements under 15 words skip LLM

    Deployment requirement:
    - Mac Mini M4 with 24GB unified memory (MINIMUM)
    - Ollama installed and running: `ollama serve`
    - Model pulled: `ollama pull nuextract` (or custom GGUF import)
    """

    VERSION = "4.1.0"
    WORD_THRESHOLD = 15
    PROSE_BLOCK_TYPES = {"paragraph", "list_item"}

    # Configurable via environment variables
    DEFAULT_OLLAMA_URL = "http://host.docker.internal:11434"  # Docker Desktop macOS
    DEFAULT_MODEL_NAME = "nuextract"

    def __init__(
        self,
        ollama_url: Optional[str] = None,
        model_name: Optional[str] = None,
    ) -> None:
        """Initialize Ollama connection (NOT model loading).

        Args:
            ollama_url: Ollama API endpoint. Default: host.docker.internal:11434
            model_name: Ollama model name. Default: "nuextract"
        """
        import os
        self.ollama_url = ollama_url or os.environ.get(
            "OLLAMA_URL", self.DEFAULT_OLLAMA_URL
        )
        self.model_name = model_name or os.environ.get(
            "NUEXTRACT_MODEL", self.DEFAULT_MODEL_NAME
        )
        self._available = False
        self._init_error: Optional[str] = None

        try:
            self._check_ollama()
            self._available = True
        except Exception as e:
            self._init_error = str(e)

    def _check_ollama(self) -> None:
        """Verify Ollama service is running and model is loaded.

        Checks:
        1. Ollama API is reachable (GET /api/tags)
        2. Target model is available in Ollama's model list
        """
        import requests

        try:
            response = requests.get(
                f"{self.ollama_url}/api/tags",
                timeout=5,
            )
            response.raise_for_status()

            models = response.json().get("models", [])
            model_names = [m.get("name", "").split(":")[0] for m in models]

            if self.model_name not in model_names:
                available = ", ".join(model_names) or "(none)"
                raise ConnectionError(
                    f"Model '{self.model_name}' not found in Ollama. "
                    f"Available: {available}. "
                    f"Pull with: ollama pull {self.model_name}"
                )
        except requests.ConnectionError:
            raise ConnectionError(
                f"Cannot reach Ollama at {self.ollama_url}. "
                "Ensure Ollama is running: `ollama serve`"
            )

    def _run_inference(self, prompt: str) -> str:
        """Run NuExtract inference via Ollama HTTP API.

        Uses temperature=0 for deterministic clinical extraction.
        Context size 8192 for long KDIGO prose blocks.
        Stream disabled for simpler response handling.
        """
        import requests

        response = requests.post(
            f"{self.ollama_url}/api/generate",
            json={
                "model": self.model_name,
                "prompt": prompt,
                "temperature": 0,
                "stream": False,
                "options": {
                    "num_ctx": 8192,
                    "num_predict": 1024,
                },
            },
            timeout=300,  # 5 min timeout per chunk (generous for CPU fallback)
        )
        response.raise_for_status()
        return response.json().get("response", "")

    # ═══════════════════════════════════════════════════════════
    # Everything below is UNCHANGED from V4 Channel F:
    # - extract() method (prose block iteration, passthrough logic)
    # - _extract_prose_blocks() (skip headings and tables)
    # - _extract_propositions() (build prompt → inference → parse)
    # - _build_prompt() (NuExtract <|input|> <|template|> <|output|> format)
    # - _parse_response() (JSON extraction with truncation recovery)
    # - _find_span_offset() (locate proposition text in document)
    # - _collect_all_sections() (leaf section traversal)
    # - EXTRACTION_TEMPLATE (atomic_facts JSON schema)
    # ═══════════════════════════════════════════════════════════
```

### Ollama Setup Commands

```bash
# Install Ollama (macOS)
brew install ollama

# Start Ollama service
ollama serve

# Option A: Pull from Ollama registry (if numind published)
ollama pull nuextract

# Option B: Import GGUF from HuggingFace
# 1. Download GGUF file
huggingface-cli download numind/NuExtract-2.0-8B-GGUF \
    NuExtract-2.0-8B-Q4_K_M.gguf --local-dir ./models

# 2. Create Ollama Modelfile
cat > Modelfile << 'EOF'
FROM ./models/NuExtract-2.0-8B-Q4_K_M.gguf
PARAMETER temperature 0
PARAMETER num_ctx 8192
TEMPLATE """<|input|>
{{ .Prompt }}
<|output|>
"""
EOF

# 3. Create Ollama model
ollama create nuextract -f Modelfile

# Verify
ollama list  # should show "nuextract"
```

### Docker Networking

Inside Docker Desktop on macOS, the Ollama service on the host is reachable at `host.docker.internal:11434`. No extra networking flags needed:

```bash
# Just works — Docker Desktop resolves host.docker.internal automatically
docker run -e OLLAMA_URL=http://host.docker.internal:11434 v4-pipeline:4.2.0
```

For Linux Docker (where `host.docker.internal` may not resolve):
```bash
docker run --add-host=host.docker.internal:host-gateway \
    -e OLLAMA_URL=http://host.docker.internal:11434 \
    v4-pipeline:4.2.0
```

### File Changes for Channel F

| Action | File | Description |
|--------|------|-------------|
| **REWRITE** | `extraction/v4/channel_f_nuextract.py` | Replace model loading with Ollama HTTP client. Keep extract logic. |
| **UPDATE** | `tools/guideline-atomiser/requirements.txt` | Remove `transformers` if no other channel needs it. Keep `requests`. |
| **UPDATE** | `tools/guideline-atomiser/Dockerfile` | Remove NuExtract model pre-download (model lives in Ollama, not Docker) |
| **NEW** | `tools/guideline-atomiser/ollama-setup.sh` | Ollama model setup script |

---

## Deployment: Mac Mini M4 (24GB)

### Memory Budget

| Component | Memory | Location | Notes |
|-----------|--------|----------|-------|
| Ollama NuExtract 2.0-8B Q4_K_M | ~6GB | Native (Metal GPU) | Persistent while Ollama runs |
| Marker L1 | ~2GB peak | Docker | During PDF processing only |
| Granite-Docling 258M | ~1-2GB | Docker | During Channel A only |
| GLiNER (Channel E) | ~500MB | Docker | During Channel E only |
| Python + Docker overhead | ~1-2GB | Docker | Persistent |
| macOS + services | ~4GB | Native | Persistent |
| **Total** | **~15-17GB** | | |
| **Headroom** | **~7-9GB** | | Comfortable |

### Performance Budget (Full KDIGO ~40 pages)

| Component | Time | Blocking? | Notes |
|-----------|------|-----------|-------|
| Marker L1 | ~30s | Sequential (L1 runs first) | PDF → markdown |
| Channel 0 | <100ms | Sequential | String replacement |
| Granite-Docling (Ch.A) | ~60-90s | Sequential (structure runs before B-F) | 258M VLM on CPU/Metal |
| Channels B, C, E | ~5-10s total | Parallel | Dictionary + regex + NER |
| Channel D | ~1s | Parallel | Table decomposition |
| **Channel F (Ollama)** | **~60 min** | **Parallel with B-E** | ~24 prose pages × 2.5 min |
| Signal Merger | <1s | Sequential (after all channels) | Overlap clustering |
| **Pipeline 1 Total** | **~62 min** | | Dominated by Channel F |

### Performance Tiers

| Tier | Channels | Time | Use Case |
|------|----------|------|----------|
| **Fast** (no Channel F) | 0, A, B, C, D, E | ~2 min | Quick extraction, review cycle |
| **Full** (with Channel F) | 0, A, B, C, D, E, F | ~62 min | Maximum proposition coverage |

CLI flag: `--channels BCDE` (fast) vs `--channels BCDEF` (full, default)

### 16GB Mac Mini Fallback

| Option | Model | VRAM | Speed | F1 Impact |
|--------|-------|------|-------|-----------|
| NuExtract 2.0-**2B** Q4 | 2B params | ~2GB | 8-10 t/s | -5% vs 8B |
| Skip Channel F | — | 0GB | instant | Channels B+C+D+E still cover core extraction |

---

## Docker Image Changes

### V4 Image (Current)
```
v4-pipeline:4.2.0-nuextract (27.1GB)
├── Marker models (~2GB)
├── Docling StandardPipeline models (~2GB): RT-DETR, TableFormer
├── GLiNER models (~500MB)
├── NuExtract-1.5 weights (~7.6GB)         ← REMOVE
└── Python + dependencies (~1GB)
```

### V4.1 Image (Target)
```
v4-pipeline:4.3.0-hybrid (~16GB estimate)
├── Marker models (~2GB)                    ← KEEP (L1 text)
├── Granite-Docling 258M (~500MB-1GB)       ← NEW (structural oracle)
├── GLiNER models (~500MB)                  ← KEEP (Channel E)
├── Python + dependencies (~1GB)            ← KEEP
└── NO NuExtract weights                    ← MOVED to Ollama
```

**Net image reduction**: ~27GB → ~16GB (NuExtract removed, Granite-Docling smaller than StandardPipeline's RT-DETR+TableFormer stack)

### Dockerfile Changes

```dockerfile
# V4.1: Remove NuExtract pre-download (now in Ollama)
# REMOVED: RUN python -c "from transformers import AutoTokenizer, AutoModelForCausalLM; ..."

# V4.1: Add Granite-Docling model cache
RUN python -c "
from docling.document_converter import DocumentConverter
from docling.pipeline.vlm_pipeline import VlmPipeline
from docling.datamodel.pipeline_options import VlmPipelineOptions
# Trigger model download during build
print('Granite-Docling VlmPipeline cached!')
"

# V4.1: Environment variables for Ollama connection
ENV OLLAMA_URL=http://host.docker.internal:11434
ENV NUEXTRACT_MODEL=nuextract
```

---

## Implementation Sequence

```
Phase 1: Foundation (Channel A + models update)
    │
    ├── 1a. granite_docling_extractor.py (VlmPipeline wrapper)
    ├── 1b. Update models.py (TableBoundary.source, TableBoundary.otsl_text,
    │       GuidelineTree.alignment_confidence, GuidelineTree.structural_source)
    └── 1c. Rewrite channel_a_docling.py (structural oracle + alignment + fallback)
    │
    ▼
Phase 2: Channel D update
    │
    ├── 2a. Add _decompose_otsl_table() to channel_d_table.py
    ├── 2b. Add _is_suspicious() heuristic
    └── 2c. Update signal_merger.py for OTSL spans (start=-1 handling)
    │
    ▼
Phase 3: Channel F rewrite
    │
    ├── 3a. Rewrite channel_f_nuextract.py (Ollama HTTP client)
    ├── 3b. Create ollama-setup.sh
    ├── 3c. Update Dockerfile (remove NuExtract, add Granite-Docling)
    └── 3d. Update run_pipeline_targeted.py (add --channels flag)
    │
    ▼
Phase 4: Integration testing
    │
    ├── 4a. Run Marker L1 on KDIGO full PDF → verify markdown quality
    ├── 4b. Run Granite-Docling on same PDF → verify DocTags structure
    ├── 4c. Run Channel A alignment → verify heading match ratio > 80%
    ├── 4d. Run Channel D on OTSL tables → verify cell extraction
    ├── 4e. Run Channel F via Ollama → verify proposition extraction
    └── 4f. Full Pipeline 1 run → compare span counts vs V4 baseline
    │
    ▼
Phase 5: Docker image build
    │
    ├── 5a. Build v4-pipeline:4.3.0-hybrid image
    ├── 5b. Verify Granite-Docling model cached in image
    ├── 5c. Verify Ollama connectivity from inside Docker
    └── 5d. End-to-end Pipeline 1 test in Docker
```

---

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Granite-Docling heading alignment < 80% on some guidelines | Medium | Channel A falls back to regex (V4 behavior) | Fallback is built-in. Log alignment scores for monitoring. |
| OTSL parsing edge cases (nested tables, complex merges) | Medium | Some table cells missed | `_is_suspicious()` flags these for human review |
| Ollama not installed/running | Low | Channel F returns 0 spans (graceful degradation) | Channel F is supplementary — B+C+D+E cover core extraction |
| Granite-Docling 258M too slow on CPU | Low | Channel A takes >5 min per document | Still viable for 10-20 docs/year. Can use Metal acceleration. |
| Marker and Granite-Docling produce conflicting structure | Low | Tree has wrong section assignments | Alignment confidence threshold catches this → falls back to regex |

---

## What Is Explicitly NOT Changing

Everything in the V4 plan below remains EXACTLY the same:

- **Channel 0**: Text normalizer (ligatures, unicode, OCR) — same logic, now on Marker markdown
- **Channel B**: Aho-Corasick drug dictionary — unchanged (text-based, L1-agnostic)
- **Channel C**: Grammar/regex patterns — unchanged (text-based, L1-agnostic)
- **Channel E**: GLiNER residual booster — unchanged (text-based, L1-agnostic)
- **Signal Merger**: Overlap clustering + confidence boost (minor update for OTSL span handling)
- **Reviewer UI**: FastAPI backend + Angular SPA — unchanged
- **Dossier Assembly**: Per-drug grouping from verified spans — unchanged
- **L2 Data Models**: RawSpan, MergedSpan, VerifiedSpan, ReviewerDecision, DrugDossier — unchanged (TableBoundary extended)
- **L2 Database Tables**: l2_extraction_jobs, l2_raw_spans, l2_merged_spans, l2_reviewer_decisions, l2_dossier_results — unchanged
- **L3 Prompt**: extract_facts_from_dossier() — unchanged
- **L4/L5/L6/L7**: Terminology, CQL validation, provenance, orchestration — unchanged
- **KB Schemas**: KB-1, KB-4, KB-16 — unchanged
- **Pipeline 2**: Full L3→L7 flow — unchanged
- **Testing Strategy**: All V4 tests remain, new tests added for Channel A alignment and Channel D OTSL

---

## Summary: Three Problems → Three Solutions

| Problem | V4 (Broken) | V4.1 (Fixed) |
|---------|------------|--------------|
| 0 tables from Docling StandardPipeline | Channel A regex couldn't parse Docling's table format | Granite-Docling OTSL tables parsed directly in Channel D |
| NuExtract OOM in Docker CPU | 3.8B model loaded in-process, 87.5% RAM, daemon died | 8B model runs in Ollama (native Metal GPU, ~6GB VRAM) |
| NuExtract 2.0 incompatible | Qwen2.5-VL architecture, `AutoModelForCausalLM` error | Ollama/llama.cpp supports Qwen2.5 natively via GGUF |

**Net result**: A pipeline that uses each model for its strength — Marker for text, Granite-Docling for structure, NuExtract via Ollama for propositions — running on a single Mac Mini M4 (24GB) with 7-9GB headroom.
