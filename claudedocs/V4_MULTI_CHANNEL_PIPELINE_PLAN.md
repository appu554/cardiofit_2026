# Clinical Guideline Curation Pipeline V3: L1+L2 Upgrade — Multi-Channel Extraction

> **This document is an UPDATE to [twinkly-baking-starlight.md](V3 Plan).**
> It replaces only **L1** (Marker → Docling) and **L2** (single NER → multi-channel + reviewer).
> **Everything else in V3 stays exactly the same**: L3 KB-specific schemas, L4 terminology, L5 CQL validation, L6 provenance, L7 orchestration, KB storage, database schema.

---

## What Changes vs V3

| Layer | V3 (Current) | Updated | Status |
|-------|-------------|---------|--------|
| **L1** | Marker v1.10 PDF→JSON | **Docling** (IBM, AAAI 2025) PDF→structured markdown | **REPLACED** |
| **L2** | Single NER (GLiNER/OpenMed) + regex fallback on Exception only | **Multi-channel extraction** (Channels 0, A-F) + Signal Merger + Reviewer UI | **REPLACED** |
| L2.5 | KB-7 RxNorm pre-lookup | KB-7 pre-lookup — **unchanged** (runs after reviewer, before L3) | UNCHANGED |
| L3 | Claude structured output → KB schemas | Same — minor prompt update (verified spans instead of gliner_entities) | **MINOR UPDATE** |
| L4 | Snow Owl / KB-7 terminology | **unchanged** | UNCHANGED |
| L5 | CQL validation (NOT generation) | **unchanged** | UNCHANGED |
| L6 | Git + FHIR Provenance | **unchanged** | UNCHANGED |
| L7 | MCP Orchestration | **unchanged** (workflow updated to call L2 channels) | UNCHANGED |
| KB schemas | KB-1/KB-4/KB-16 Pydantic → Go structs | **unchanged** | UNCHANGED |
| DB schema | source_documents, source_sections, derived_facts | **unchanged** + new L2 tables for spans/reviewer | EXTENDED |
| Fact store | Three-table pattern with governance | **unchanged** | UNCHANGED |

---

## The Problem with Current L2

The V3 L2 has a **single point of failure**:

1. **GLiNER silently misses** drug names, dosing thresholds, lab parameters, and recommendation IDs in 300+ page PDFs
2. **Regex fallback only fires on `except Exception`** (line 91 of `run_pipeline.py`), meaning partial GLiNER misses are never caught
3. **`markdown_text[:5000]`** truncation means most of a 300-page PDF is never processed by NER
4. **Docling output reveals** systematic ligature corruption (`/uniFB01` = fi, `/uniFB02` = fl) breaking finerenone, empagliflozin, dapagliflozin, canagliflozin — ALL key SGLT2 inhibitor names

---

## Two-Pipeline Architecture

The upgraded pipeline splits into **two pipelines** with a human approval gate in between. This uses the existing factstore DB in GCP for KB-0, adding a new reviewer table for the approval gate.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    PIPELINE 1: EXTRACTION + REVIEW                          │
│                    (automated channels → human text QA)                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  GUIDELINES (SOURCE)                                                        │
│  KDIGO PDFs, FDA Labels, ADA, etc.                                         │
│       │                                                                      │
│       ▼                                                                      │
│  L1: Docling (REPLACES Marker)                                              │
│  PDF → structured markdown with chapters, sections, tables, recs            │
│       │                                                                      │
│       ▼                                                                      │
│  L2: Multi-Channel Extraction (REPLACES single GLiNER)                      │
│  ┌────────────────────────────────────────────────────────────────────────┐ │
│  │  Channel 0: Text Normalizer (ligature/symbol/OCR fix)                  │ │
│  │       ↓                                                                 │ │
│  │  Channel A: Docling structure parse (RUNS FIRST — prerequisite)        │ │
│  │       ↓                                                                 │ │
│  │  Channels B-F run in parallel (consume A's structure metadata):        │ │
│  │    B: Aho-Corasick drug dictionary (word-boundary enforced)            │ │
│  │    C: Grammar/regex (eGFR, labs, doses, rec IDs) — drug-agnostic       │ │
│  │    D: Table decomposer (one RawSpan per cell, NOT synthetic concat)    │ │
│  │    E: GLiNER residual booster (full text, NO truncation)               │ │
│  │    F: NuExtract 2.0-4B (prose only, >15 words, temperature=0)         │ │
│  │       ↓                                                                 │ │
│  │  Signal Merger → per-section dossier                                    │ │
│  └────────────────────────────────────────────────────────────────────────┘ │
│       │                                                                      │
│       ▼                                                                      │
│  DB: l2_* tables (existing factstore GCP) — merged spans stored             │
│       │                                                                      │
│       ▼                                                                      │
│  REVIEWER UI (text QA ONLY: confirm/fix/add)                                │
│  No classification, no entity typing, no KB routing                          │
│       │                                                                      │
│       ▼                                                                      │
│  DB: l2_reviewer_decisions table — approval stored                           │
│                                                                              │
│  ════════════════════ APPROVAL GATE ═══════════════════════                  │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘

                        ↓ (after reviewer approves)

┌─────────────────────────────────────────────────────────────────────────────┐
│                    PIPELINE 2: CLASSIFICATION + GOVERNANCE                   │
│                    (L3 clinical intelligence → KB storage)                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Verified text spans (from Pipeline 1 approval)                             │
│       │                                                                      │
│       ▼                                                                      │
│  Dossier Assembly (NEW — groups spans into per-drug dossiers)               │
│  Drug anchors (Channel B) + associated signals (C-F) via section co-location│
│       │                                                                      │
│       ▼                                                                      │
│  L2.5: KB-7 RxNorm Pre-Lookup (per-drug) ◄── UNCHANGED                     │
│       │                                                                      │
│       ▼                                                                      │
│  L3: Claude Structured Output (per-drug dossier) ◄── MINOR UPDATE           │
│  KB-1: RenalAdjustment facts                                                │
│  KB-4: Contraindication facts                                                │
│  KB-16: LabRequirement facts                                                 │
│       │                                                                      │
│       ▼                                                                      │
│  L4: KB-7 Terminology Validation ◄── UNCHANGED                              │
│  L5: CQL Compatibility Check ◄── UNCHANGED                                  │
│       │                                                                      │
│       ▼                                                                      │
│  DB: derived_facts table (existing factstore) ◄── UNCHANGED                 │
│       │                                                                      │
│       ▼                                                                      │
│  KB-0 Governance (DRAFT → PENDING_REVIEW → APPROVED) ◄── UNCHANGED         │
│       │                                                                      │
│       ▼                                                                      │
│  KB-1/KB-4/KB-16 Storage ◄── UNCHANGED                                      │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Why Two Pipelines

1. **Clear human gate**: Pipeline 1 is automated extraction ending in DB storage. Pipeline 2 only starts after human approval. No accidental L3 invocation on unreviewed spans.
2. **Uses existing DB**: The factstore DB in GCP already has KB-0 governance tables. We add `l2_*` tables alongside — no new database, just new tables in the existing factstore.
3. **Async workflow**: Pipeline 1 can run batch-mode overnight on new guidelines. Reviewer works the queue during business hours. Pipeline 2 triggers automatically on approval.
4. **Cost control**: L3 Claude API calls only happen on reviewer-approved spans, not on all raw channel output.

---

## L1 Replacement: Marker → Docling

### Why Replace Marker

| Issue | Marker v1.10 | Docling |
|-------|-------------|---------|
| Structure preservation | Flat markdown, no hierarchy | Chapter/section/recommendation hierarchy |
| Table handling | Basic markdown tables | Structured table objects with cell metadata |
| Document grammar | None — treats PDF as raw text | Hierarchical PDF grammar parser |
| Recommendation IDs | Lost in flat text | Preserved as structural elements |
| Page provenance | Limited bbox tracking | Full page/section/block provenance |

### Docling Output Already Exists

The Docling output is already at project root: `KDIGO-2022-Diabetes-CKD-Docling-Output.md`

Verified observations from this output:
- Document structure (chapters 1-6, sections, recommendations) comes through perfectly
- Tables render as proper markdown (GLP-1 RA dosing table at lines 603-611)
- **BUT** ligature corruption: `/uniFB01` → fi, `/uniFB02` → fl (Channel 0 fixes this)
- **BUT** symbol errors: `‡` should be `≥` (Channel 0 fixes this)
- **BUT** some table cells have incomplete drug names (Channel D + B cross-reference fixes this)

### Files to Change for L1

| Action | File | Description |
|--------|------|-------------|
| MODIFY | `shared/tools/guideline-atomiser/marker_extractor.py` | Replace Marker invocation with Docling; keep `TableBlock` structure and OCR post-processor (reused by Channel 0) |
| MODIFY | `shared/tools/guideline-atomiser/requirements.txt` | Replace `marker>=1.10.0` with `docling` |
| KEEP | All L1 test infrastructure | Update test expectations for Docling output format |

### L1 Output Format (Same Contract, Different Source)

L1 continues to output structured markdown with provenance metadata — same contract that L2 consumes. The internal implementation changes from Marker to Docling, but the interface to L2 stays the same.

---

## L2 Replacement: Single NER → Multi-Channel Extraction

### Design Philosophy

**Every clinically relevant text span should be discovered by at least two independent channels.** Where channels disagree, a human reviewer (text QA, not clinical expert) confirms, rejects, or adds spans. L3 Claude then receives reviewer-verified spans and does ALL clinical intelligence.

### Three Key Design Decisions

1. **Reviewer is NOT a clinical expert** — they do text QA only (confirm/fix/add spans). NO classification, NO entity typing, NO KB routing.

2. **L3 Claude does ALL clinical intelligence** — entity classification, KB routing (KB-1/KB-4/KB-16), structured fact extraction. Same as V3, just receives better input.

3. **NuExtract 2.0-4B** for Channel F — small deterministic extraction model, temperature=0, purely extractive (no hallucination by design).

### L2 Sub-Components (CORRECTED Ordering)

```
L1 output (Docling markdown)
    │
    ▼
Channel 0: Text Normalizer (sequential — runs first)
    │
    ▼
Channel A: Docling structure parse (sequential — runs second, prerequisite for D+F)
    │
    ├──► Channel B: Aho-Corasick drug dictionary
    ├──► Channel C: Grammar/regex patterns (drug-agnostic)
    ├──► Channel D: Table decomposer (consumes A's table boundaries)     ← parallel
    ├──► Channel E: GLiNER residual booster (depends on B+C output)
    └──► Channel F: NuExtract 2.0-4B (consumes A's block types, prose only)
            │
            ▼
    Signal Merger (union + boost + disagreement flags)
            │
            ▼
    DB: l2_merged_spans table (Pipeline 1 ends here)
            │
            ▼
    ════ REVIEWER UI (text QA) → APPROVAL GATE ════
            │
            ▼
    Dossier Assembly (groups verified spans into per-drug dossiers)
            │
            ▼
    Verified dossiers → L2.5 → L3 (Pipeline 2 starts here)
```

> **CORRECTED from prior version**: Channel A runs BEFORE B-F (not parallel). D needs A's table boundaries. F needs A's block types to know what's prose. E depends on B+C output for novel-only filtering.

---

### Channel 0: Text Normalizer

Runs before all channels. Fixes Docling output corruption.

**Source code to harvest**:
- `marker_extractor.py` lines 215-524: `ClinicalOCRPostProcessor` with all correction dictionaries
- `gliner/extractor.py` lines 163-212: `OCR_CORRECTIONS` dict

```python
class Channel0Normalizer:
    """Normalize Docling output text before extraction channels run.

    Fixes:
    1. Unicode ligature corruption (fi → fi, fl → fl)
    2. Symbol errors (double-dagger → >=)
    3. OCR letter substitutions (rn → m, l → 1)
    4. Unit normalization (mL/min/1.73m2 variants)
    5. Whitespace normalization

    IMPORTANT: Must also handle Docling-specific artifacts (not just Marker-era patterns).
    Validate against real Docling output to confirm all corruption is caught.
    """

    LIGATURE_MAP = {
        "\ufb01": "fi",   # fi ligature
        "\ufb02": "fl",   # fl ligature
        "/uniFB01": "fi",
        "/uniFB02": "fl",
    }

    SYMBOL_MAP = {
        "\u2021": ">=",   # double-dagger → >=
        "\u2020": "+",    # dagger → +
    }

    def normalize(self, text: str) -> tuple[str, dict]:
        """Returns (normalized_text, metadata with fix counts)."""
```

**File**: `shared/extraction/v4/channel_0_normalizer.py` (NEW)

**Required test** (validates against real Docling output, not just synthetic):
```python
def test_channel_0_on_real_docling_output():
    """Run normalizer on actual Docling output, assert no residual corruption."""
    text = open("KDIGO-2022-Diabetes-CKD-Docling-Output.md").read()
    normalized, meta = Channel0Normalizer().normalize(text)
    assert "/uniFB01" not in normalized
    assert "/uniFB02" not in normalized
    assert "‡" not in normalized  # should be ≥
    assert meta["fix_count"] > 0  # confirms it actually ran
```

---

### Channel A: Docling Structure Parser (RUNS FIRST — Sequential)

Parses the Docling markdown output to extract structural boundaries: chapter breaks, section headings, recommendation blocks, table boundaries.

Does NOT extract entities — produces a **`GuidelineTree`** structure that other channels consume for:
- `section_id` assignment (all channels use this for provenance)
- `source_block_type` classification (heading, paragraph, table_cell, list_item, recommendation)
- Table boundary identification (Channel D needs this)
- Prose block identification (Channel F needs this — only processes blocks where `source_block_type = "paragraph"` or `"list_item"`)

**Output contract**: Channel A does NOT produce `RawSpan` objects. It produces:
```python
@dataclass
class GuidelineTree:
    """Structure map of the guideline document."""
    sections: list[GuidelineSection]
    tables: list[TableBoundary]
    total_pages: int

@dataclass
class GuidelineSection:
    section_id: str           # e.g., "4.1.1"
    heading: str              # e.g., "Recommendation 4.1.1"
    start_offset: int
    end_offset: int
    page_number: int
    block_type: str           # heading, paragraph, table, list_item, recommendation
    children: list["GuidelineSection"]

@dataclass
class TableBoundary:
    table_id: str             # e.g., "table_3"
    section_id: str           # parent section
    start_offset: int
    end_offset: int
    headers: list[str]        # column headers
    row_count: int
    page_number: int
```

**Span-to-section association**: Uses **offset ranges**, not element IDs. A span belongs to a section if `section.start_offset <= span.start < section.end_offset`. This is simpler and avoids needing a separate element_ids set per section.

**Why sequential**: Channel A must complete before B-F start because:
- Channel D needs `TableBoundary` objects to know which text blocks are tables
- Channel F needs `block_type` to only process prose (paragraph/list_item), skipping headings and table cells
- All channels need `section_id` for provenance assignment

**File**: `shared/extraction/v4/channel_a_docling.py` (NEW)

---

### Channel B: Drug Dictionary (Aho-Corasick)

O(n) multi-pattern string matching using a pre-built dictionary with **word boundary enforcement**.

**Dictionary sources**:
1. `KNOWN_DRUG_INGREDIENTS` set (extractor.py lines 192-205): metformin, dapagliflozin, empagliflozin, finerenone, canagliflozin, lisinopril, etc.
2. `DRUG_CLASSES` set (extractor.py lines 257-278): "SGLT2 inhibitors", "ACE inhibitors", etc.
3. `KNOWN_CLASS_ABBREVIATIONS` (extractor.py lines 208-212): sglt2i, acei, arb, mra, etc.
4. CQL valuesets from `RenalCommon.cql`: Nephrotoxic drug lists (NSAIDs, Aminoglycosides, Vancomycin, ACEi, ARB, Diuretics)
5. CQL valuesets from `T2DMGuidelines.cql`: Metformin, SGLT2i, GLP1-RA, DPP4i, Sulfonylurea, TZD, Insulin variants

**Implementation**: Use the `ahocorasick` Python library. Build automaton from all drug names (lowercased). On match, emit a RawSpan with `channel_metadata = {"match_type": "exact"|"class"|"abbreviation", "rxnorm_candidate": "..."}`.

**Word boundary enforcement** (CRITICAL — prevents "ARB" matching inside "garbanzo"):
```python
def _is_word_boundary(self, text: str, start: int, end: int) -> bool:
    """Reject matches that aren't at word boundaries."""
    if start > 0 and text[start - 1].isalnum():
        return False
    if end < len(text) and text[end].isalnum():
        return False
    return True
```

**Dictionary deduplication**: The `build_dictionary.py` script normalizes all drug names to lowercase and deduplicates across sources. Each drug has one canonical entry with a `sources` list indicating provenance (CQL, extractor.py, etc.). The automaton does NOT emit 3 hits for "metformin" appearing in 3 source lists.

**Files**:
- `shared/extraction/v4/channel_b_drug_dict.py` (NEW)
- `shared/extraction/v4/dictionaries/drug_dictionary.json` (NEW — generated)
- `shared/extraction/v4/dictionaries/build_dictionary.py` (NEW — harvests from CQL + extractor.py)

---

### Channel C: Grammar/Regex Patterns (Drug-Agnostic)

Deterministic pattern matching for clinical signals. **Channel C outputs are drug-agnostic** — a dose "10 mg daily" or a threshold "eGFR < 30" is extracted without knowing which drug it belongs to. Drug-signal association happens later in the Dossier Assembly step via section co-location.

**Pattern sources** (harvested from `_apply_clinical_rules()` at extractor.py lines 576-659):
- eGFR threshold patterns (3 regex patterns)
- Monitoring frequency patterns (5 regex patterns)
- Lab test patterns (1 comprehensive pattern)
- Contraindication marker patterns (1 pattern)

**Additional patterns to add**:
- Recommendation ID: `r'Recommendation\s+\d+\.\d+(?:\.\d+)?'`
- Dose values: `r'\d+(?:\.\d+)?\s*(?:mg|mcg|g|mL|units?)(?:\s*/\s*(?:day|dose|kg))?'`
- eGFR range: `r'eGFR\s+\d+\s*[-–]\s*\d+\s*mL/min/1\.73\s*m[2²]'`
- Potassium thresholds: `r'(?:potassium|K\+)\s*[>≥]\s*\d+(?:\.\d+)?\s*(?:mEq/L|mmol/L)'`
- LOINC code references: `r'\b\d{4,5}-\d\b'`

> **Note on miss risk**: Regex has zero miss risk only for patterns in the set. If KDIGO uses uncovered phrasing (e.g., "hold if potassium exceeds 5.5" where "exceeds" isn't in the pattern), Channel C misses it. Channel F (NuExtract) provides the safety net for non-standard phrasing.

**File**: `shared/extraction/v4/channel_c_grammar.py` (NEW)

---

### Channel D: Table Decomposer (One RawSpan Per Cell)

Consumes `TableBoundary` objects from Channel A (Docling). Decomposes each table into **one `RawSpan` per cell**, preserving verbatim provenance.

> **CORRECTED from prior version**: The prior spec concatenated cells into synthetic phrases ("Metformin eGFR Required") that don't exist in the source text, breaking `start`/`end` offset provenance. The corrected design emits one RawSpan per cell with table-aware metadata.

**Output per cell**:
```python
RawSpan(
    channel="D",
    text="Every 3-6 months",          # verbatim cell text
    start=12345,                        # real offset in normalized text
    end=12362,
    confidence=0.95,                       # 0.95 not 1.0: Docling TableFormer is very good but not infallible on complex merged cells
    table_id="table_3",
    section_id="4.1.1",
    source_block_type="table_cell",
    channel_metadata={
        "row_index": 0,
        "col_index": 3,
        "col_header": "Monitoring Frequency",
        "row_drug": "Metformin",         # from column 0 of same row
    }
)
```

The drug-to-signal association (which drug does "Every 3-6 months" belong to?) is captured in `channel_metadata.row_drug` from the same row's drug column. The **Signal Merger** then clusters these per-row into groups, and the **Dossier Assembly** step creates the per-drug dossier.

**File**: `shared/extraction/v4/channel_d_table.py` (NEW)

---

### Channel E: GLiNER Residual Booster (Full Text — NO Truncation)

Wraps existing `ClinicalNERExtractor` from `extraction/gliner/extractor.py` but with two critical changes:

1. **No truncation**: Runs on the FULL normalized text, not `markdown_text[:5000]`. The V3 truncation (Problem #3) was a primary cause of missed entities.
2. **Novel-only filtering**: Only emits spans NOT already found by Channels B + C. GLiNER becomes a safety net for novel entities, not the primary NER.

```python
class ChannelEGLiNERResidual:
    def extract(self, text: str, existing_spans: list[RawSpan]) -> ChannelOutput:
        """Run GLiNER on FULL text, then subtract spans already found by B+C.

        CRITICAL: Do NOT truncate text. The old pipeline had markdown_text[:5000]
        which is why GLiNER missed entities beyond the first few pages.
        """
        gliner_result = self.ner.extract_for_kb(text, "all")  # FULL text
        # Filter: only keep spans where no existing span overlaps by >50%
        novel_spans = self._filter_novel(gliner_result.entities, existing_spans)
        return ChannelOutput(channel="E", spans=novel_spans, ...)
```

**File**: `shared/extraction/v4/channel_e_gliner.py` (NEW — wraps existing extractor.py)

---

### Channel F: NuExtract 2.0-4B Propositions (Detailed Spec)

Uses NuExtract 2.0-4B (quantized) to decompose complex prose into atomic propositions. This is a **purely extractive model** — all output text must exist in the input. This is the primary reason it was chosen over Qwen3 or Phi-4.

**Operational contract**:

| Parameter | Value | Rationale |
|-----------|-------|-----------|
| Model | `numind/NuExtract-2.0-4B` (GGUF Q4_K_M) | 4B params, fits in 4GB VRAM |
| Temperature | **0** (NOT 0.1) | Extraction model, not generative |
| Invocation threshold | Prose blocks only, **>15 words** | Short spans are already atomic — skip LLM |
| Block types processed | `paragraph`, `list_item` only | NOT `heading`, NOT `table_cell` (tables handled by Channel D) |
| Input | One prose block at a time | From Channel A's `GuidelineSection` where `block_type` is prose |

**Extraction template** (JSON template that defines the output schema):
```json
{
    "atomic_facts": [
        {
            "statement": "verbatim-string",
            "drug": "verbatim-string",
            "threshold": "verbatim-string",
            "action": "verbatim-string",
            "condition": "verbatim-string"
        }
    ]
}
```

**Example**:

Input: "Metformin is contraindicated when eGFR falls below 30 mL/min/1.73m2. When eGFR is 30-45, reduce maximum daily dose by 50%."

Output:
```json
{
    "atomic_facts": [
        {
            "statement": "Metformin is contraindicated when eGFR falls below 30 mL/min/1.73m2",
            "drug": "Metformin",
            "threshold": "eGFR falls below 30 mL/min/1.73m2",
            "action": "contraindicated",
            "condition": "eGFR falls below 30"
        },
        {
            "statement": "When eGFR is 30-45, reduce maximum daily dose by 50%",
            "drug": "Metformin",
            "threshold": "eGFR is 30-45",
            "action": "reduce maximum daily dose by 50%",
            "condition": "eGFR is 30-45"
        }
    ]
}
```

Each atomic fact becomes a `RawSpan` with `channel_metadata = {"proposition": statement, "drug": drug, "threshold": threshold}`.

**Passthrough logic**: Elements under 15 words pass through WITHOUT LLM processing — they're already atomic. This saves compute and avoids unnecessary model invocations on short spans.

**Error handling**: NuExtract sometimes returns truncated JSON. Channel F must handle malformed output gracefully (log warning, skip element, not crash pipeline).

**File**: `shared/extraction/v4/channel_f_nuextract.py` (NEW)

---

### Signal Merger

Combines output from all channels into per-section dossiers and writes to `l2_merged_spans` table.

**Algorithm**:
```
Input: ChannelOutput from channels B, C, D, E, F
       GuidelineTree from Channel A (for section assignment)
       Channel 0 already applied to text

Step 1: UNION — Collect all RawSpans from all channels, sort by start offset

Step 2: CLUSTER — Overlapping spans (≥50% character overlap) → one cluster
  - Each cluster becomes one MergedSpan

Step 3: CONFIDENCE BOOST
    1 channel: +0.00
    2 channels: +0.05
    3 channels: +0.10
    4+ channels: +0.15 (cap at 1.0)

Step 4: TEXT — Use longest span in cluster as merged text
  - Record all contributing channel texts in channel_confidences

Step 5: DISAGREEMENT — Flag if contributing spans have different text (beyond whitespace)
  - e.g., Channel B: "dapagliflozin" vs Channel E: "dapagliflozin 10mg"
  - Set has_disagreement = True with detail string

Step 6: SECTION ASSIGNMENT — Use Channel A's GuidelineTree to assign section_id
  - Each MergedSpan gets the section_id of its enclosing section

Step 7: OUTPUT — Write MergedSpans to l2_merged_spans table
  - Update l2_extraction_jobs with total_merged_spans count
  - Set review_status = 'IN_REVIEW' on the job
  - Pipeline 1 ends here — waits for reviewer
```

**File**: `shared/extraction/v4/signal_merger.py` (NEW)

---

### Reviewer UI (Pipeline 1 → Approval Gate)

**Key Constraint**: Reviewer does text QA ONLY. NO entity type dropdown, NO KB routing selector, NO classification.

The reviewer's four actions:
1. **CONFIRM** — this text span is correct
2. **REJECT** — this is OCR garbage / not clinically relevant
3. **EDIT** — fix the text (spelling, truncation, etc.)
4. **ADD** — pipeline missed a span; select it from the text

All entity typing, KB routing, and clinical classification happens in **L3 Claude** (Pipeline 2) after the reviewer is done.

**UI Layout**:
- **Left panel**: Full normalized guideline text with highlighted spans (green = 3+ channels, yellow = 2, orange = 1, red outline = disagreement)
- **Right panel**: Span list with contributing channels, confidence, actions
- **Keyboard shortcuts**: `C` confirm, `R` reject, `E` edit, `N` next, `P` previous

**Files**:
- `shared/tools/guideline-atomiser/reviewer_api.py` (NEW — FastAPI backend)
- `shared/tools/guideline-atomiser/reviewer_api_models.py` (NEW)
- `shared/tools/guideline-atomiser/static/reviewer/` (NEW — Angular SPA)

**FastAPI Endpoints**:
```
GET  /api/v4/jobs                          # List extraction jobs
GET  /api/v4/jobs/{job_id}/spans           # Get merged spans for review
     ?status=PENDING                       # Filter by review status
     ?disagreement=true                    # Filter disagreement spans only
     ?page=3                               # Filter by page number
GET  /api/v4/jobs/{job_id}/text            # Get full normalized text
POST /api/v4/spans/{span_id}/decide        # Submit reviewer decision
POST /api/v4/jobs/{job_id}/add-span        # Reviewer adds missed span
POST /api/v4/jobs/{job_id}/complete-review  # Mark review done → triggers Pipeline 2
GET  /api/v4/jobs/{job_id}/decisions        # Audit trail
```

**On `complete-review`**: The backend:
1. Sets `review_status = 'COMPLETED'` on the `l2_extraction_jobs` row
2. Computes verified spans (CONFIRMED + EDITED + ADDED)
3. Triggers Pipeline 2 (dossier assembly → L2.5 → L3 → L4 → L5 → KB-0)

---

### Dossier Assembly (NEW — Bridge Between Pipeline 1 and Pipeline 2)

> **This component was missing from the prior version.** Without it, L3 receives a flat bag of verified spans with no drug association. Claude would have to figure out which spans belong to which drug — the same "find the needle" task we're trying to eliminate.

After the reviewer approves spans, the **Dossier Assembler** groups them into per-drug dossiers before passing to L3.

**Algorithm**:
```
Input: list[VerifiedSpan] (reviewer-approved)
       GuidelineTree (from Channel A)

Step 1: IDENTIFY DRUG ANCHORS
  - Find all VerifiedSpans from Channel B (drug dictionary matches)
  - These are the drug anchors: "Metformin", "Dapagliflozin", "Finerenone", etc.
  - NOTE: If the reviewer REJECTED a drug anchor span, that drug gets NO dossier.

Step 2: ASSOCIATE SIGNALS WITH DRUGS
  - For each non-drug span (Channel C thresholds, Channel D table cells, Channel F propositions):
    - Table cells: Use channel_metadata.row_drug for DIRECT association (highest priority)
    - Prose spans: Find the nearest drug anchor in the same section_id by character offset
  - Tie-breaking for multi-drug sections:
    - ONE drug in section → associate all signals with that drug
    - MULTIPLE drugs, span within 200 chars of one drug → associate with that drug
    - MULTIPLE drugs, span NOT within 200 chars of any single drug → associate with
      ALL drugs in the section. L3 handles deduplication across dossiers.
    - Drug CLASS anchors (e.g., "SGLT2i") → associate with the class dossier.
      L3 decides whether to expand to individual drugs or keep as class-level fact.

Step 3: BUILD PER-DRUG DOSSIER
  - For each drug anchor, collect:
    - Drug name + RxNorm candidate (from Channel B hint)
    - All associated threshold spans (from Channel C)
    - All associated table cell spans (from Channel D)
    - All associated propositions (from Channel F)
    - Source section_ids and page_numbers
    - Full source text for the enclosing sections

Step 4: OUTPUT
  - list[DrugDossier] — one per drug found in the guideline
  - Each dossier is a self-contained package for L3
```

```python
@dataclass
class DrugDossier:
    """A self-contained extraction package for one drug, ready for L3."""
    drug_name: str
    rxnorm_candidate: Optional[str]        # from Channel B hint, NOT authoritative
    verified_spans: list[VerifiedSpan]       # all spans associated with this drug
    source_sections: list[str]               # section_ids this drug appears in
    source_pages: list[int]                  # page numbers
    source_text: str                         # full text of enclosing sections
    signal_summary: dict                     # {"thresholds": 3, "doses": 2, "monitoring": 1}
```

L3 then processes one `DrugDossier` at a time, not the entire guideline. This is exactly what V3 was supposed to do, but with verified high-quality input instead of raw GLiNER entities.

**File**: `shared/extraction/v4/dossier_assembler.py` (NEW)

---

## L2 Data Models

These are **new models for the L2 multi-channel internals only**. They do NOT replace any existing V3 models.

### RawSpan (Output of each Channel B-F)

```python
from pydantic import BaseModel, Field
from typing import Optional, Literal
from datetime import datetime
from uuid import UUID, uuid4

class RawSpan(BaseModel):
    """A single text span discovered by one channel."""
    id: UUID = Field(default_factory=uuid4)
    channel: Literal["B", "C", "D", "E", "F"]  # A produces GuidelineTree, not RawSpan
    text: str
    start: int
    end: int
    confidence: float = Field(ge=0.0, le=1.0)
    page_number: Optional[int] = None
    section_id: Optional[str] = None
    table_id: Optional[str] = None
    source_block_type: Optional[Literal[
        "heading", "paragraph", "table_cell", "list_item", "recommendation"
    ]] = None
    channel_metadata: dict = Field(default_factory=dict)
```

### MergedSpan (Signal Merger output → stored in DB → Reviewer queue item)

```python
class MergedSpan(BaseModel):
    """A span after multi-channel merging. Stored in l2_merged_spans table.
    This is what the reviewer sees."""
    id: UUID = Field(default_factory=uuid4)
    job_id: UUID
    text: str
    start: int
    end: int
    contributing_channels: list[Literal["B", "C", "D", "E", "F"]]
    channel_confidences: dict[str, float]
    merged_confidence: float
    has_disagreement: bool = False
    disagreement_detail: Optional[str] = None
    page_number: Optional[int] = None
    section_id: Optional[str] = None
    table_id: Optional[str] = None
    review_status: Literal["PENDING", "CONFIRMED", "REJECTED", "EDITED", "ADDED"] = "PENDING"
    reviewer_text: Optional[str] = None
    reviewed_by: Optional[str] = None
    reviewed_at: Optional[datetime] = None
```

### ReviewerDecision (Audit trail for reviewer actions)

```python
class ReviewerDecision(BaseModel):
    """Record of a reviewer's action on a merged span. Stored in l2_reviewer_decisions."""
    id: UUID = Field(default_factory=uuid4)
    merged_span_id: UUID
    job_id: UUID
    action: Literal["CONFIRM", "REJECT", "EDIT", "ADD"]
    original_text: Optional[str] = None
    edited_text: Optional[str] = None
    reviewer_id: str
    decided_at: datetime = Field(default_factory=datetime.utcnow)
    note: Optional[str] = None  # e.g., "OCR garbled this span, fixed spelling"
```

### VerifiedSpan (L2 output → Dossier Assembly input → eventually L3 input)

```python
class VerifiedSpan(BaseModel):
    """A reviewer-approved span ready for dossier assembly and then L3 Claude.

    This replaces the old gliner_entities list[dict].
    The reviewer confirmed the TEXT is correct.
    L3 Claude does ALL classification, entity typing, and KB routing.
    """
    text: str
    start: int
    end: int
    confidence: float
    contributing_channels: list[str]
    page_number: Optional[int] = None
    section_id: Optional[str] = None
    table_id: Optional[str] = None

    # Machine-generated hints for L3. NOT shown to reviewer. May be incorrect.
    extraction_context: dict = Field(default_factory=dict)
    # e.g., {"channel_B_rxnorm_candidate": "860975", "channel_C_pattern": "egfr_threshold"}
```

> **RENAMED from prior version**: `channel_hints` → `extraction_context`. Clarifies these are machine-generated context for L3, not reviewer-visible classification labels.

**File**: `shared/extraction/v4/models.py` (NEW)

---

## L2 Database Tables (New — extends existing factstore in GCP)

These tables are added to the **existing factstore DB in GCP** alongside KB-0 governance tables. They do NOT modify existing tables.

```sql
-- Migration: 02-l2-multichannel-schema.sql
-- Extends existing factstore schema with L2 multi-channel extraction tables

-- Extraction jobs for multi-channel runs
CREATE TABLE IF NOT EXISTS l2_extraction_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_pdf VARCHAR(500) NOT NULL,
    source_hash VARCHAR(64) NOT NULL,
    guideline_authority VARCHAR(100) NOT NULL,
    guideline_document VARCHAR(500) NOT NULL,
    normalized_text TEXT,

    -- Channel statuses
    channel_0_status VARCHAR(20) DEFAULT 'PENDING',
    channel_a_status VARCHAR(20) DEFAULT 'PENDING',
    channel_b_status VARCHAR(20) DEFAULT 'PENDING',
    channel_c_status VARCHAR(20) DEFAULT 'PENDING',
    channel_d_status VARCHAR(20) DEFAULT 'PENDING',
    channel_e_status VARCHAR(20) DEFAULT 'PENDING',
    channel_f_status VARCHAR(20) DEFAULT 'PENDING',
    merger_status VARCHAR(20) DEFAULT 'PENDING',
    review_status VARCHAR(20) DEFAULT 'PENDING',   -- PENDING → IN_REVIEW → COMPLETED

    -- Pipeline 2 statuses (triggered after review approval)
    dossier_status VARCHAR(20) DEFAULT 'PENDING',
    l3_status VARCHAR(20) DEFAULT 'PENDING',
    l4_status VARCHAR(20) DEFAULT 'PENDING',
    l5_status VARCHAR(20) DEFAULT 'PENDING',

    -- Metrics
    total_raw_spans INT DEFAULT 0,
    total_merged_spans INT DEFAULT 0,
    spans_confirmed INT DEFAULT 0,
    spans_rejected INT DEFAULT 0,
    spans_edited INT DEFAULT 0,
    spans_added INT DEFAULT 0,
    dossiers_created INT DEFAULT 0,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    review_started_at TIMESTAMPTZ,
    review_completed_at TIMESTAMPTZ,
    pipeline2_started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_l2_jobs_review ON l2_extraction_jobs(review_status);

-- Per-channel raw spans
CREATE TABLE IF NOT EXISTS l2_raw_spans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES l2_extraction_jobs(id) ON DELETE CASCADE,
    channel VARCHAR(2) NOT NULL,          -- 'B','C','D','E','F'
    text TEXT NOT NULL,
    start_offset INT NOT NULL,
    end_offset INT NOT NULL,
    confidence DECIMAL(4,3) NOT NULL,
    page_number INT,
    section_id VARCHAR(50),
    table_id VARCHAR(50),
    source_block_type VARCHAR(30),
    channel_metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_l2_raw_spans_job ON l2_raw_spans(job_id);
CREATE INDEX idx_l2_raw_spans_channel ON l2_raw_spans(job_id, channel);
CREATE INDEX idx_l2_raw_spans_offset ON l2_raw_spans(job_id, start_offset);

-- Merged spans (signal merger output = reviewer queue)
CREATE TABLE IF NOT EXISTS l2_merged_spans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES l2_extraction_jobs(id) ON DELETE CASCADE,
    text TEXT NOT NULL,
    start_offset INT NOT NULL,
    end_offset INT NOT NULL,
    contributing_channels TEXT[] NOT NULL,
    channel_confidences JSONB NOT NULL,
    merged_confidence DECIMAL(4,3) NOT NULL,
    has_disagreement BOOLEAN DEFAULT FALSE,
    disagreement_detail TEXT,
    page_number INT,
    section_id VARCHAR(50),
    table_id VARCHAR(50),
    review_status VARCHAR(20) DEFAULT 'PENDING',
    reviewer_text TEXT,
    reviewed_by VARCHAR(100),
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_l2_merged_review ON l2_merged_spans(job_id, review_status);
CREATE INDEX idx_l2_merged_disagreement ON l2_merged_spans(job_id) WHERE has_disagreement = TRUE;

-- Reviewer decisions (audit trail)
CREATE TABLE IF NOT EXISTS l2_reviewer_decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merged_span_id UUID NOT NULL REFERENCES l2_merged_spans(id) ON DELETE CASCADE,
    job_id UUID NOT NULL REFERENCES l2_extraction_jobs(id) ON DELETE CASCADE,
    action VARCHAR(20) NOT NULL,          -- CONFIRM, REJECT, EDIT, ADD
    original_text TEXT,
    edited_text TEXT,
    reviewer_id VARCHAR(100) NOT NULL,
    decided_at TIMESTAMPTZ DEFAULT NOW(),
    note TEXT
);

CREATE INDEX idx_l2_decisions_job ON l2_reviewer_decisions(job_id);

-- Per-drug dossier results (Pipeline 2 per-drug tracking)
-- Allows retrying L3 for a single drug without re-running Pipeline 1
CREATE TABLE IF NOT EXISTS l2_dossier_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES l2_extraction_jobs(id) ON DELETE CASCADE,
    drug_name VARCHAR(200) NOT NULL,
    rxnorm_candidate VARCHAR(20),
    span_count INT NOT NULL,
    l3_status VARCHAR(20) DEFAULT 'PENDING',  -- PENDING → RUNNING → COMPLETED → FAILED
    l3_result JSONB,                           -- KB-specific extraction result
    l3_error TEXT,                              -- error message if FAILED
    l4_status VARCHAR(20) DEFAULT 'PENDING',
    l5_status VARCHAR(20) DEFAULT 'PENDING',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_l2_dossier_job ON l2_dossier_results(job_id);
CREATE INDEX idx_l2_dossier_status ON l2_dossier_results(job_id, l3_status);
```

> **Per-drug tracking**: If Metformin's dossier extracts successfully but Finerenone's L3 call fails, you can retry just Finerenone without re-running Pipeline 1 or re-processing other drugs. The `l2_extraction_jobs` Pipeline 2 status columns (`l3_status`, `l4_status`, `l5_status`) track overall job progress; `l2_dossier_results` tracks per-drug progress.

**File**: `shared/tools/guideline-atomiser/init-db/02-l2-multichannel-schema.sql` (NEW)

---

## L3 Prompt Change (Minor — Only Input Format Changes)

### What Changes

The L3 `fact_extractor.py` gains ONE new method. The existing `extract_facts()` stays for backward compatibility.

**Current V3 signature** (STAYS):
```python
def extract_facts(self, markdown_text, gliner_entities, target_kb, guideline_context)
```

**New method added** (for multi-channel L2 — receives per-drug dossier):
```python
def extract_facts_from_dossier(self, dossier: DrugDossier, target_kb, guideline_context)
```

### Prompt Section Replacement

The only prompt change is replacing:
```
## Pre-tagged Clinical Entities (from GLiNER):
{json.dumps(entities)}
```

with:
```
## Reviewer-Verified Text Spans:

The following text spans have been confirmed by a human text reviewer.
The text is VERIFIED CORRECT.
Machine-generated extraction context is included but may be incorrect.

YOUR TASK: Classify each span and extract structured facts for {drug_name}.

Verified Spans:
1. "Metformin" (confidence: 0.98, channels: [B, C, E] [page 12] [section 4.1.1])
2. "eGFR falls below 30" (confidence: 0.97, channels: [C, E] [page 12] [section 4.1.1])
...
```

### Everything Else in L3 Stays

- Same KB-specific Pydantic schemas (KB1ExtractionResult, KB4ExtractionResult, KB16ExtractionResult)
- Same Claude model (`claude-sonnet-4-5-20250514`)
- Same tool_choice pattern
- Same KB routing logic
- Same "FACTS not RULES" principle
- Same provenance fields

---

## What Stays EXACTLY the Same from V3 Plan

### KB Schemas (UNCHANGED)
- `schemas/kb1_dosing.py` — RenalAdjustment, DrugRenalFacts, KB1ExtractionResult
- `schemas/kb4_safety.py` — ContraindicationFact, KB4ExtractionResult
- `schemas/kb16_labs.py` — LabMonitoringEntry, LabRequirementFact, KB16ExtractionResult

### Target Extraction Tables (UNCHANGED)
Same extraction targets from V3 plan:

| Drug | eGFR Range | Adjustment | Max Dose | Action |
|------|------------|------------|----------|--------|
| Metformin | < 30 | - | - | **CONTRAINDICATED** |
| Metformin | 30-44 | 0.5 | 1000mg | REDUCE_DOSE |
| Dapagliflozin | < 20 | - | - | **CONTRAINDICATED** |
| Dapagliflozin | ≥ 20 | 1.0 | 10mg | CONTINUE |
| Finerenone | - | - | - | MONITOR (K+ > 5.5 → HOLD) |

### L4: Snow Owl / KB-7 Terminology (UNCHANGED)
Same as V3: Validate and enrich facts with RxNorm, LOINC, SNOMED-CT codes.

### L5: CQL Validation (UNCHANGED)
Same as V3: Validate that extracted facts can be consumed by existing CQL in vaidshala/. NOT generating CQL.

### L6: Provenance (UNCHANGED)
Same as V3: Git + FHIR Provenance audit trail.

### L7: MCP Orchestration (UNCHANGED — workflow updated)
Same L7, but the two-pipeline split means:
```
Pipeline 1 (automated + review):
  1. L7 calls L1 (Docling) → structured markdown           ← CHANGED from Marker
  2. L7 calls L2 (multi-channel) → raw spans → DB          ← CHANGED from single NER
  3. L7 triggers Reviewer UI → waits for approval           ← NEW
     L7 invocation ENDS here. Pipeline 1 is complete.

Pipeline 2 (SEPARATE L7 invocation, triggered by complete-review webhook):
  4. Dossier Assembly → per-drug packages                   ← NEW
  5. L7 calls L2.5 (KB-7 pre-lookup) per drug              ← UNCHANGED
  6. L7 calls L3 (Claude) per drug dossier → KB facts       ← UNCHANGED
  7. L7 calls L4 (terminology) → validated facts            ← UNCHANGED
  8. L7 calls L5 (CQL check) → compatibility report         ← UNCHANGED
  9. L7 calls L6 (Git commit) → provenance                  ← UNCHANGED
 10. KB-0 Governance (DRAFT → PENDING_REVIEW → APPROVED)    ← UNCHANGED
```

**Orchestration boundary**: Pipeline 1 and Pipeline 2 are **separate L7 invocations**. Pipeline 1 runs to completion (L1 → L2 → DB → reviewer queue) and exits. Pipeline 2 is triggered by the `complete-review` API endpoint, which sets `review_status = 'COMPLETED'` on `l2_extraction_jobs` and enqueues a new L7 Pipeline 2 job. The `l2_extraction_jobs` table is the coordination point between the two invocations — Pipeline 2 reads the job row and its verified spans to begin dossier assembly.

### Storage Destinations (UNCHANGED)
- KB-1: `kb-1-drug-rules/` — drug_rules.renal_adjustments
- KB-4: `kb-4-patient-safety/` — pkg/safety/data/contraindications/*.yaml
- KB-16: `kb-16-lab-interpretation/` — lab_monitoring_requirements
- Vaidshala CQL: NOT modified by pipeline

### Database (UNCHANGED — V3 tables + new L2 tables)
- `source_documents`, `source_sections`, `derived_facts` — all unchanged
- `extraction_jobs` — unchanged
- New `l2_*` tables added alongside (not replacing) in existing factstore DB

---

## File Summary: What's New vs What's Reused

### New Files (L2 multi-channel internals only)

| File | Purpose |
|------|---------|
| `extraction/v4/__init__.py` | Package init |
| `extraction/v4/models.py` | RawSpan, MergedSpan, VerifiedSpan, ReviewerDecision, DrugDossier, DossierResult |
| `extraction/v4/channel_0_normalizer.py` | Ligature/symbol/OCR fix |
| `extraction/v4/channel_a_docling.py` | Docling structure parser → GuidelineTree |
| `extraction/v4/channel_b_drug_dict.py` | Aho-Corasick drug dictionary (word-boundary enforced) |
| `extraction/v4/channel_c_grammar.py` | Regex/grammar patterns (drug-agnostic) |
| `extraction/v4/channel_d_table.py` | Table cell decomposer (one RawSpan per cell) |
| `extraction/v4/channel_e_gliner.py` | GLiNER residual (full text, no truncation) |
| `extraction/v4/channel_f_nuextract.py` | NuExtract 2.0-4B (temperature=0, prose only, >15 words) |
| `extraction/v4/signal_merger.py` | Span union + confidence boost + section assignment |
| `extraction/v4/dossier_assembler.py` | Groups verified spans into per-drug dossiers for L3 |
| `extraction/v4/dictionaries/drug_dictionary.json` | Pre-built drug dictionary (deduplicated) |
| `extraction/v4/dictionaries/build_dictionary.py` | Dictionary builder from CQL + extractor.py |
| `tools/guideline-atomiser/reviewer_api.py` | FastAPI reviewer backend |
| `tools/guideline-atomiser/reviewer_api_models.py` | API models |
| `tools/guideline-atomiser/static/reviewer/` | Angular reviewer SPA |
| `tools/guideline-atomiser/init-db/02-l2-multichannel-schema.sql` | L2 tables in existing factstore DB |

### Modified Files (Minimal changes)

| File | Change |
|------|--------|
| `tools/guideline-atomiser/marker_extractor.py` | Replace Marker with Docling invocation |
| `tools/guideline-atomiser/fact_extractor.py` | Add `extract_facts_from_dossier()` method |
| `tools/guideline-atomiser/data/run_pipeline.py` | Update to two-pipeline flow |
| `tools/guideline-atomiser/requirements.txt` | Add docling, ahocorasick, nuextract deps |

### Untouched Files (Everything else from V3)

All KB schemas, L3 prompts (except input section), L4 client, L5 checker, L6 provenance, L7 orchestrator, KB storage, existing DB schema, CQL registry — ALL unchanged.

---

## Testing Strategy

Tests are for the **new L2 components only**. All existing V3 tests remain and must continue passing.

| Test File | What It Tests |
|-----------|--------------|
| `tests/v4/test_channel_0.py` | Ligature repair, symbol fix, OCR cleanup, idempotency, **real Docling output validation** |
| `tests/v4/test_channel_a.py` | Section boundary detection, table detection, heading hierarchy, GuidelineTree contract |
| `tests/v4/test_channel_b.py` | Dictionary completeness, Aho-Corasick match speed, case-insensitive, **word boundary enforcement** (no "ARB" in "garbanzo"), abbreviation matching |
| `tests/v4/test_channel_c.py` | eGFR patterns, monitoring frequency, rec IDs, no overlapping spans, **drug-agnostic output** |
| `tests/v4/test_channel_d.py` | **One RawSpan per cell** (not synthetic concatenation), row_drug metadata, column header propagation |
| `tests/v4/test_channel_e.py` | **Full text processing** (no truncation), novel-only filtering, confidence in valid range |
| `tests/v4/test_channel_f.py` | NuExtract template invocation, temperature=0, **>15 word threshold**, prose-only filtering, **malformed JSON handling** |
| `tests/v4/test_signal_merger.py` | Overlap detection, confidence boost, disagreement flagging, section assignment |
| `tests/v4/test_dossier_assembler.py` | **Drug anchor identification**, signal-to-drug association via section co-location, table row_drug association, per-drug dossier completeness |
| `tests/v4/test_reviewer_api.py` | FastAPI CRUD, status transitions, audit trail, Pipeline 2 trigger on complete-review |
| `tests/v4/test_l3_dossier.py` | `extract_facts_from_dossier()` accepts DrugDossier, prompt correct, extraction_context hints present |

**Golden test**: Run all channels on KDIGO sample text → merge → mock-confirm all → dossier assembly → L3 extract → assert Metformin facts match V3 expected output.

**Rejection path tests** (ensures rejected spans propagate correctly through dossier assembly):
```python
def test_rejected_drug_anchor_removes_dossier():
    """If reviewer rejects the 'Finerenone' drug anchor span,
    no Finerenone dossier should be created by the assembler."""

def test_rejected_threshold_excluded_from_dossier():
    """If reviewer rejects 'eGFR ≥ 20' but confirms 'Metformin',
    Metformin dossier should still exist but without that threshold span."""

def test_edited_span_uses_reviewer_text():
    """If reviewer edits 'dapa gliflozin' → 'dapagliflozin',
    the dossier should contain the corrected reviewer text."""

def test_added_span_included_in_dossier():
    """If reviewer adds a missed span 'canagliflozin' in section 4.2,
    a new Canagliflozin dossier should be created."""
```

**V3 regression**: All existing V3 tests must continue to pass unchanged.

### Performance Benchmarks

| Component | Target | Notes |
|-----------|--------|-------|
| Channel 0 normalization | < 100ms for 100K chars | String replacement |
| Channel A structure parse | < 2s for 350-page PDF | Markdown parsing |
| Channel B dictionary | < 5ms for 100K chars | Aho-Corasick O(n+m) |
| Channel C regex | < 50ms for 100K chars | Compiled regex |
| Channel D table decomposition | < 1s per table | One pass per table |
| Channel E GLiNER (full text) | < 60s for 350-page PDF | GPU-dependent |
| **Channel F NuExtract** | **< 25 min for 350-page PDF** | **~50-80 tokens/sec on RTX 4090, ~500 prose elements** |
| Channel F per-element | < 5s for 200-word paragraph | Single invocation |
| All channels parallel | < 30 min for 350-page PDF | Channel F is bottleneck |
| Signal merger | < 500ms for 5000 raw spans | DB write |
| Dossier assembly | < 1s for 50 drugs | In-memory grouping |

---

## Implementation Sequence

```
Pipeline 1 components:
  L1 Docling setup (replaces Marker)
      │
      ▼
  Channel 0 + L2 data models + DB migration (foundation)
      │
      ▼
  Channel A structure parser (sequential — must complete first)
      │
      ├──► Channels B+C (deterministic — can be tested standalone)
      ├──► Channel D (structural — consumes A's table boundaries)        ← parallel
      ├──► Channel E (ML — depends on B+C for novel-only filter)
      └──► Channel F (ML — consumes A's block types for prose-only)
              │
              ▼
          Signal Merger
              │
              ▼
          Reviewer UI (backend + frontend)

Pipeline 2 components (triggered after reviewer approval):
          Dossier Assembler
              │
              ▼
          L3 prompt update (extract_facts_from_dossier)
              │
              ▼
          Integration test (full Pipeline 1 → approval → Pipeline 2 → L4 → L5)
```

Everything from L3 onward is the same V3 pipeline running as before — just with better, reviewer-verified, per-drug-dossier input.
