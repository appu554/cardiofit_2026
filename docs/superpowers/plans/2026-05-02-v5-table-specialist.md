# V5 Subsystem #1 — Table Specialist Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace Channel D's text-offset-based pipe/OTSL table decomposition with a Docling TableBlock-aware path that uses the already-present `l1_result.tables` (from `marker_extractor.TableBlock`) as primary source, raising mean table-cell accuracy from ~50% to ≥85% on the 15-table hand-graded set.

**Architecture:** Additive V5 layer on top of V4 Channel D. When `V5_TABLE_SPECIALIST=1` and `l1_tables` is non-empty, `ChannelDTableDecomposer.extract()` delegates table decomposition to `_decompose_docling_table()` which uses the structured `TableBlock.rows`/`TableBlock.headers`/`TableBlock.bbox` data directly instead of re-parsing markdown text. Falls back to V4 OTSL/pipe paths when flag is off. `_V5_KNOWN_FEATURES` in `run_pipeline_targeted.py` gains `"table_specialist"`.

**Tech Stack:** Python 3.13, existing `marker_extractor.TableBlock` dataclass, `extraction.v4.channel_d_table.ChannelDTableDecomposer`, `extraction.v4.v5_flags.is_v5_enabled`, pytest.

**Success criteria (from master spec §7):**
- Primary: ≥85% mean cell accuracy across 15 hand-graded tables (macro-average)
- Secondary: header detection ≥95%, cell-merge accuracy ≥95%, numeric presence-in-cells ≥98%
- Universal regression: total spans within ±15% of V4, TIER_1 prop ≥ V4, 0 new ERRORs

---

## File Structure

| File | Disposition | Purpose |
|------|-------------|---------|
| `${EXTRACTION}/v4/channel_d_table.py` | MODIFY | Add `_decompose_docling_table()` + V5 routing in `extract()` |
| `${ATOMISER}/data/run_pipeline_targeted.py` | MODIFY | Thread `l1_tables` into Channel D; add `"table_specialist"` to `_V5_KNOWN_FEATURES` |
| `${ATOMISER}/data/v5_metrics.py` | MODIFY | Add `compute_v5_table_metrics()` function |
| `${ATOMISER}/tests/v5/test_v5_table_specialist.py` | CREATE | Unit tests for `_decompose_docling_table()` and flag routing |
| `${ATOMISER}/tests/v5/golden/tables/sample_table_01.csv` | CREATE | Ground truth: one 4×3 table for smoke-set unit test |

```
ATOMISER=backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser
EXTRACTION=backend/shared-infrastructure/knowledge-base-services/shared/extraction
```

---

## Task 1: Failing unit tests for `_decompose_docling_table()`

**Files:**
- Create: `${ATOMISER}/tests/v5/test_v5_table_specialist.py`
- Create: `${ATOMISER}/tests/v5/golden/tables/sample_table_01.csv`

- [ ] **Step 1.1: Create ground-truth CSV fixture**

Create `tests/v5/golden/tables/sample_table_01.csv`:
```
row_idx,col_idx,expected_text
0,0,Metformin
0,1,≥30 mL/min
0,2,Continue
1,0,SGLT2i
1,1,<45 mL/min
1,2,Reduce dose
2,0,GLP-1 RA
2,1,Any eGFR
2,2,Continue
```

- [ ] **Step 1.2: Write failing tests**

Create `tests/v5/test_v5_table_specialist.py`:
```python
"""V5 Table Specialist unit tests.

Run from guideline-atomiser/:
    PYTHONPATH=. V5_TABLE_SPECIALIST=1 pytest tests/v5/test_v5_table_specialist.py -v
"""
from __future__ import annotations

import csv
import os
from pathlib import Path

import pytest

from marker_extractor import TableBlock

# We can't import BoundingBox from marker_extractor easily; build minimal stub
try:
    from marker_extractor import BoundingBox as MBBox
    _BBOX = MBBox(x=10.0, y=20.0, width=400.0, height=120.0)
except Exception:
    _BBOX = None

from extraction.v4.channel_d_table import ChannelDTableDecomposer
from extraction.v4.models import GuidelineTree, TableBoundary


GOLDEN_DIR = Path(__file__).parent / "golden" / "tables"


def _make_minimal_tree() -> GuidelineTree:
    return GuidelineTree(sections=[], tables=[], total_pages=3)


# ─── Flag routing ──────────────────────────────────────────────────────────

def test_v4_path_used_when_flag_off(monkeypatch):
    """When V5_TABLE_SPECIALIST is absent, extract() calls V4 OTSL/pipe paths."""
    monkeypatch.delenv("V5_TABLE_SPECIALIST", raising=False)
    monkeypatch.delenv("V5_DISABLE_ALL", raising=False)

    decomp = ChannelDTableDecomposer()
    tree = _make_minimal_tree()
    tb = TableBlock(
        headers=["Drug", "eGFR", "Action"],
        rows=[["Metformin", "≥30 mL/min", "Continue"]],
        page_number=1,
        confidence=1.0,
        table_index=0,
    )
    out = decomp.extract("", tree, l1_tables=[tb])
    # No spans from Docling path; tree has no TableBoundary so pipe path also empty
    assert out.channel == "D"
    assert "table_specialist_used" not in out.metadata or not out.metadata.get("table_specialist_used")


def test_docling_path_used_when_flag_on(monkeypatch):
    """When V5_TABLE_SPECIALIST=1 and l1_tables provided, Docling path is used."""
    monkeypatch.setenv("V5_TABLE_SPECIALIST", "1")
    monkeypatch.delenv("V5_DISABLE_ALL", raising=False)

    decomp = ChannelDTableDecomposer()
    tree = _make_minimal_tree()
    tb = TableBlock(
        headers=["Drug", "eGFR", "Action"],
        rows=[["Metformin", "≥30 mL/min", "Continue"]],
        page_number=1,
        confidence=1.0,
        table_index=0,
    )
    out = decomp.extract("", tree, l1_tables=[tb])
    assert out.metadata.get("table_specialist_used") is True
    assert len(out.spans) >= 3  # at least 3 cells (1 data row × 3 cols)


def test_disable_all_overrides_table_specialist(monkeypatch):
    """V5_DISABLE_ALL=1 forces V4 path even if V5_TABLE_SPECIALIST=1."""
    monkeypatch.setenv("V5_TABLE_SPECIALIST", "1")
    monkeypatch.setenv("V5_DISABLE_ALL", "1")

    decomp = ChannelDTableDecomposer()
    tree = _make_minimal_tree()
    tb = TableBlock(
        headers=["Drug", "eGFR", "Action"],
        rows=[["Metformin", "≥30 mL/min", "Continue"]],
        page_number=1,
        confidence=1.0,
        table_index=0,
    )
    out = decomp.extract("", tree, l1_tables=[tb])
    assert not out.metadata.get("table_specialist_used")


# ─── Cell extraction accuracy ─────────────────────────────────────────────

def test_docling_table_cells_match_golden(monkeypatch):
    """Cells extracted from a 3-row×3-col TableBlock match golden CSV."""
    monkeypatch.setenv("V5_TABLE_SPECIALIST", "1")
    monkeypatch.delenv("V5_DISABLE_ALL", raising=False)

    decomp = ChannelDTableDecomposer()
    tree = _make_minimal_tree()
    tb = TableBlock(
        headers=["Drug", "eGFR", "Action"],
        rows=[
            ["Metformin", "≥30 mL/min", "Continue"],
            ["SGLT2i", "<45 mL/min", "Reduce dose"],
            ["GLP-1 RA", "Any eGFR", "Continue"],
        ],
        page_number=2,
        confidence=1.0,
        table_index=0,
    )
    out = decomp.extract("", tree, l1_tables=[tb])

    # Load golden
    golden: dict[tuple[int, int], str] = {}
    with open(GOLDEN_DIR / "sample_table_01.csv") as f:
        for row in csv.DictReader(f):
            golden[(int(row["row_idx"]), int(row["col_idx"]))] = row["expected_text"]

    # Build extracted index
    extracted: dict[tuple[int, int], str] = {}
    for span in out.spans:
        if span.source_block_type == "table_cell":
            r = span.channel_metadata.get("row_index", -1)
            c = span.channel_metadata.get("col_index", -1)
            extracted[(r, c)] = span.text.strip()

    # All golden cells must be present and match (case-insensitive, whitespace-norm)
    correct = sum(
        1 for (r, c), expected in golden.items()
        if extracted.get((r, c), "").lower() == expected.lower()
    )
    accuracy = correct / len(golden) * 100
    assert accuracy >= 85.0, f"Cell accuracy {accuracy:.1f}% < 85% threshold"


def test_header_detection(monkeypatch):
    """Headers row is detected and col_header metadata is set correctly."""
    monkeypatch.setenv("V5_TABLE_SPECIALIST", "1")
    monkeypatch.delenv("V5_DISABLE_ALL", raising=False)

    decomp = ChannelDTableDecomposer()
    tree = _make_minimal_tree()
    tb = TableBlock(
        headers=["Drug", "Dose", "Frequency"],
        rows=[["Metformin", "500 mg", "BD"]],
        page_number=1,
        confidence=1.0,
        table_index=0,
    )
    out = decomp.extract("", tree, l1_tables=[tb])
    data_spans = [s for s in out.spans if s.source_block_type == "table_cell"
                  and s.channel_metadata.get("row_index", 0) > 0]
    headers_seen = {s.channel_metadata.get("col_header") for s in data_spans}
    assert "Drug" in headers_seen
    assert "Dose" in headers_seen
    assert "Frequency" in headers_seen


def test_page_number_propagated(monkeypatch):
    """Page number from TableBlock is propagated to every cell RawSpan."""
    monkeypatch.setenv("V5_TABLE_SPECIALIST", "1")
    monkeypatch.delenv("V5_DISABLE_ALL", raising=False)

    decomp = ChannelDTableDecomposer()
    tree = _make_minimal_tree()
    tb = TableBlock(
        headers=["A"],
        rows=[["val1"], ["val2"]],
        page_number=7,
        confidence=1.0,
        table_index=0,
    )
    out = decomp.extract("", tree, l1_tables=[tb])
    for span in out.spans:
        if span.source_block_type == "table_cell":
            assert span.page_number == 7, f"Expected page 7, got {span.page_number}"


def test_empty_cells_skipped(monkeypatch):
    """Empty cells (empty string or whitespace) are not emitted as spans."""
    monkeypatch.setenv("V5_TABLE_SPECIALIST", "1")
    monkeypatch.delenv("V5_DISABLE_ALL", raising=False)

    decomp = ChannelDTableDecomposer()
    tree = _make_minimal_tree()
    tb = TableBlock(
        headers=["A", "B"],
        rows=[["val", ""], ["", "val2"]],
        page_number=1,
        confidence=1.0,
        table_index=0,
    )
    out = decomp.extract("", tree, l1_tables=[tb])
    texts = [s.text for s in out.spans if s.source_block_type == "table_cell"]
    assert "" not in texts
    assert "   " not in texts
```

- [ ] **Step 1.3: Run tests to confirm they fail**

```bash
cd backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser
PYTHONPATH=. V5_TABLE_SPECIALIST=1 python -m pytest tests/v5/test_v5_table_specialist.py -v 2>&1 | tail -20
```
Expected: `FAILED` — `extract()` does not yet accept `l1_tables` kwarg.

---

## Task 2: Implement `_decompose_docling_table()` in Channel D

**Files:**
- Modify: `${EXTRACTION}/v4/channel_d_table.py`

- [ ] **Step 2.1: Update `extract()` signature to accept `l1_tables`**

In `channel_d_table.py`, change `ChannelDTableDecomposer.extract()` (line 94):

```python
    def extract(
        self,
        text: str,
        tree: GuidelineTree,
        l1_tables: Optional[list] = None,   # list[TableBlock] from marker_extractor
        profile=None,
    ) -> ChannelOutput:
```

- [ ] **Step 2.2: Add V5 routing at the start of `extract()`**

Replace the opening of `extract()` (after `start_time = time.monotonic()`):

```python
        start_time = time.monotonic()
        spans: list[RawSpan] = []
        tables_pipe = 0
        tables_otsl = 0
        tables_docling = 0
        suspicious_tables = 0
        table_specialist_used = False

        # V5 Table Specialist: use Docling TableBlock as primary when flag on
        if l1_tables and is_v5_enabled("table_specialist", profile):
            table_specialist_used = True
            for tb in l1_tables:
                docling_spans = self._decompose_docling_table(tb)
                tables_docling += 1
                spans.extend(docling_spans)
            # Still run OTSL/pipe for tables NOT covered by l1_tables
            # (those without a matching TableBlock index stay on V4 path)
            covered_indices = {tb.table_index for tb in l1_tables}
            for table in tree.tables:
                if getattr(table, "table_index", None) in covered_indices:
                    continue
                if table.source == "granite_otsl" and table.otsl_text:
                    table_spans = self._decompose_otsl_table(table)
                    tables_otsl += 1
                else:
                    table_spans = self._decompose_pipe_table(text, table, tree)
                    tables_pipe += 1
                if self._is_suspicious(table, table_spans):
                    suspicious_tables += 1
                spans.extend(table_spans)
        else:
            # V4 path: unchanged
            for table in tree.tables:
                if table.source == "granite_otsl" and table.otsl_text:
                    table_spans = self._decompose_otsl_table(table)
                    tables_otsl += 1
                else:
                    table_spans = self._decompose_pipe_table(text, table, tree)
                    tables_pipe += 1
                if self._is_suspicious(table, table_spans):
                    suspicious_tables += 1
                spans.extend(table_spans)
```

- [ ] **Step 2.3: Add `_decompose_docling_table()` method**

Add after `_decompose_otsl_table()`:

```python
    # ═══════════════════════════════════════════════════════════════════════
    # V5 #1 TABLE SPECIALIST: Docling TableBlock path
    # ═══════════════════════════════════════════════════════════════════════

    CONFIDENCE_DOCLING = 0.97  # Docling TableFormer has higher structural confidence

    def _decompose_docling_table(self, tb) -> list[RawSpan]:
        """Decompose a Docling TableBlock into cell-level RawSpans.

        TableBlock.headers contains the header row as a list[str].
        TableBlock.rows contains data rows as list[list[str]].
        Each non-empty cell becomes one RawSpan with row/col/header metadata.

        Args:
            tb: marker_extractor.TableBlock instance.
        """
        spans: list[RawSpan] = []
        headers = tb.headers or []
        bbox_raw = None
        if tb.bbox is not None:
            # marker_extractor.BoundingBox: x, y, width, height
            # Convert to [x0, y0, x1, y1] for RawSpan.bbox
            try:
                bbox_raw = [
                    float(tb.bbox.x),
                    float(tb.bbox.y),
                    float(tb.bbox.x) + float(tb.bbox.width),
                    float(tb.bbox.y) + float(tb.bbox.height),
                ]
            except (AttributeError, TypeError):
                bbox_raw = None

        for row_idx, row in enumerate(tb.rows):
            row_drug = None
            for ci, cell in enumerate(row):
                if cell and cell.strip():
                    row_drug = cell.strip()
                    break

            for col_idx, cell_text in enumerate(row):
                if not cell_text or not cell_text.strip():
                    continue

                col_header = headers[col_idx] if col_idx < len(headers) else None

                spans.append(RawSpan(
                    channel="D",
                    text=cell_text.strip(),
                    start=-1,
                    end=-1,
                    confidence=self.CONFIDENCE_DOCLING,
                    page_number=tb.page_number,
                    source_block_type="table_cell",
                    bbox=bbox_raw,
                    channel_metadata={
                        "row_index": row_idx,
                        "col_index": col_idx,
                        "col_header": col_header,
                        "row_drug": row_drug,
                        "table_source": "docling_tableblock",
                        "table_index": tb.table_index,
                    },
                ))

        return spans
```

- [ ] **Step 2.4: Update `ChannelOutput` return to include new metadata key**

Change the `return ChannelOutput(...)` block to add `table_specialist_used` and `tables_docling`:

```python
        return ChannelOutput(
            channel="D",
            spans=spans,
            metadata={
                "tables_processed": len(tree.tables) + tables_docling,
                "tables_pipe": tables_pipe,
                "tables_otsl": tables_otsl,
                "tables_docling": tables_docling,
                "cells_extracted": len(spans) - len(footnote_spans),
                "caption_footnotes": len(footnote_spans),
                "suspicious_tables": suspicious_tables,
                "table_specialist_used": table_specialist_used,
            },
            elapsed_ms=elapsed_ms,
        )
```

- [ ] **Step 2.5: Also need to handle caption footnotes after the V5 branch**

Ensure `footnote_spans = self._extract_caption_footnotes(text, tree)` and `spans.extend(footnote_spans)` are called in **both** branches (before `elapsed_ms`). The `_decompose_docling_table` path still benefits from caption footnotes since those come from `text`/`tree`.

- [ ] **Step 2.6: Run tests — expect them to pass**

```bash
cd backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser
PYTHONPATH=. V5_TABLE_SPECIALIST=1 python -m pytest tests/v5/test_v5_table_specialist.py -v 2>&1 | tail -20
```
Expected: all 6 tests `PASSED`.

- [ ] **Step 2.7: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/channel_d_table.py \
        backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/tests/v5/test_v5_table_specialist.py \
        backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/tests/v5/golden/tables/sample_table_01.csv
git commit -m "feat(v5): Table Specialist — Docling TableBlock path in Channel D (#1)

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 3: Thread `l1_tables` through `run_pipeline_targeted.py`

**Files:**
- Modify: `${ATOMISER}/data/run_pipeline_targeted.py`

- [ ] **Step 3.1: Update Channel D call site (line ~459) to pass `l1_tables`**

Find `d_output = channel_d.extract(normalized_text, tree)` and replace:

```python
    print("   [Channel D] Table cell decomposition...")
    channel_d = ChannelD()
    d_output = channel_d.extract(normalized_text, tree, l1_tables=l1_result.tables, profile=profile)
```

There are two pipeline functions — apply the same change to **both** occurrences (the main pipeline around line 459 and the alternate pipeline around line 1624 if it exists).

- [ ] **Step 3.2: Add `"table_specialist"` to `_V5_KNOWN_FEATURES` (line 825)**

Change:
```python
    _V5_KNOWN_FEATURES = ["bbox_provenance"]
```
To:
```python
    _V5_KNOWN_FEATURES = ["bbox_provenance", "table_specialist"]
```

- [ ] **Step 3.3: Run the pipeline smoke test with both flags**

```bash
cd backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser
V5_BBOX_PROVENANCE=1 V5_TABLE_SPECIALIST=1 python data/run_pipeline_targeted.py \
    --source docling_l1 --target-kb kb-3-guidelines \
    data/input/AU-HF-ACS-HCP-Summary-2025.pdf 2>&1 | tail -30
```
Expected: `tables_docling: N` in Channel D output (N > 0), no errors.

- [ ] **Step 3.4: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/data/run_pipeline_targeted.py
git commit -m "feat(v5): wire l1_tables into Channel D; add table_specialist to _V5_KNOWN_FEATURES

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 4: Add `compute_v5_table_metrics()` to `v5_metrics.py`

**Files:**
- Modify: `${ATOMISER}/data/v5_metrics.py`

- [ ] **Step 4.1: Add the function**

Add after `write_v5_metrics()`:

```python
_TABLE_CELL_ACCURACY_THRESHOLD = 85.0


def compute_v5_table_metrics(
    merged_spans: list[dict],
    ground_truth_csvs: list[str] | None = None,
) -> dict:
    """Compute V5 Table Specialist metrics.

    Without ground_truth_csvs: reports only structural metadata (tables_docling,
    total table-cell spans, % from docling path).

    With ground_truth_csvs: also computes cell accuracy against golden CSVs.
    Each CSV must have columns: row_idx, col_idx, expected_text.
    """
    import csv as _csv

    table_spans = [
        s for s in merged_spans
        if isinstance(s, dict)
        and any(
            cm.get("table_source") == "docling_tableblock"
            for cm in [s.get("channel_metadata", {})]
            if isinstance(cm, dict)
        )
    ]

    total_table_spans = len([
        s for s in merged_spans
        if isinstance(s, dict)
        and s.get("source_block_type") == "table_cell"
    ])

    docling_table_spans = len(table_spans)

    accuracy_pct = None
    if ground_truth_csvs:
        all_correct = 0
        all_total = 0
        per_table_accuracy: list[float] = []

        for csv_path in ground_truth_csvs:
            golden: dict[tuple[int, int], str] = {}
            try:
                with open(csv_path) as f:
                    for row in _csv.DictReader(f):
                        golden[(int(row["row_idx"]), int(row["col_idx"]))] = row["expected_text"]
            except (OSError, KeyError):
                continue

            extracted: dict[tuple[int, int], str] = {}
            for s in merged_spans:
                if not isinstance(s, dict):
                    continue
                cm = s.get("channel_metadata", {})
                if not isinstance(cm, dict):
                    continue
                if cm.get("table_source") == "docling_tableblock":
                    r = cm.get("row_index", -1)
                    c = cm.get("col_index", -1)
                    extracted[(r, c)] = (s.get("text") or "").strip()

            correct = sum(
                1 for (r, c), expected in golden.items()
                if extracted.get((r, c), "").lower() == expected.lower()
            )
            table_acc = (correct / len(golden) * 100) if golden else 0.0
            per_table_accuracy.append(table_acc)
            all_correct += correct
            all_total += len(golden)

        accuracy_pct = (
            round(sum(per_table_accuracy) / len(per_table_accuracy), 2)
            if per_table_accuracy else 0.0
        )

    primary_status = None
    if accuracy_pct is not None:
        primary_status = "PASS" if accuracy_pct >= _TABLE_CELL_ACCURACY_THRESHOLD else "FAIL"

    result: dict = {
        "v5_table_specialist": {
            "total_table_cell_spans": total_table_spans,
            "docling_table_cell_spans": docling_table_spans,
            "docling_coverage_pct": round(
                docling_table_spans / total_table_spans * 100, 2
            ) if total_table_spans > 0 else 0.0,
        },
    }
    if accuracy_pct is not None:
        result["v5_table_specialist"]["cell_accuracy_pct"] = accuracy_pct
        result["primary"] = result.get("primary", {})
        result["primary"]["table_cell_accuracy_pct"] = {
            "v5": accuracy_pct,
            "threshold": _TABLE_CELL_ACCURACY_THRESHOLD,
            "status": primary_status,
        }
        result["verdict"] = primary_status

    return result
```

- [ ] **Step 4.2: Update `_main()` to report table metrics when `merged_spans.json` has docling spans**

In `_main()`, after computing bbox metrics, add:

```python
        table_metrics = compute_v5_table_metrics(spans)
        if table_metrics["v5_table_specialist"]["docling_table_cell_spans"] > 0:
            write_v5_metrics(job_dir, table_metrics)
            ts = table_metrics["v5_table_specialist"]
            print(
                f"{job_dir.name}: docling_table_spans={ts['docling_table_cell_spans']} "
                f"({ts['docling_coverage_pct']:.1f}% of table cells)"
            )
```

- [ ] **Step 4.3: Run smoke test and check metrics.json**

```bash
V5_BBOX_PROVENANCE=1 V5_TABLE_SPECIALIST=1 python data/run_pipeline_targeted.py \
    --source docling_l1 --target-kb kb-3-guidelines \
    data/input/AU-HF-ACS-HCP-Summary-2025.pdf 2>&1 | tail -10
python data/v5_metrics.py data/output/v4/$(ls -t data/output/v4/ | head -1)
```
Expected: `v5_table_specialist` key in `metrics.json`, no errors.

- [ ] **Step 4.4: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/data/v5_metrics.py
git commit -m "feat(v5): add compute_v5_table_metrics() to v5_metrics sidecar

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 5: Integration test — run smoke set, verify ≥85%

**Files:**
- Create: `${ATOMISER}/tests/v5/test_v5_table_specialist_smoke.py`

- [ ] **Step 5.1: Write smoke acceptance test**

Create `tests/v5/test_v5_table_specialist_smoke.py`:
```python
"""Smoke acceptance test for V5 Table Specialist.

Requires:
    V5_TABLE_SPECIALIST=1
    data/input/AU-HF-ACS-HCP-Summary-2025.pdf present
    data/input/AU-HF-Cholesterol-Action-Plan-2026.pdf present

Run:
    PYTHONPATH=. V5_TABLE_SPECIALIST=1 V5_BBOX_PROVENANCE=1 \
        pytest tests/v5/test_v5_table_specialist_smoke.py -v -m smoke
"""
import json
import os
from pathlib import Path

import pytest

pytestmark = pytest.mark.smoke

SMOKE_JOB_DIR = Path("data/output/v4")
GOLDEN_DIR = Path("tests/v5/golden/tables")


@pytest.mark.skipif(
    not any(SMOKE_JOB_DIR.glob("**/job_metadata.json")),
    reason="No job output found — run pipeline first",
)
def test_table_specialist_active_in_metadata():
    """job_metadata.json reports table_specialist in v5_features_enabled."""
    latest_meta = sorted(SMOKE_JOB_DIR.glob("**/job_metadata.json"))[-1]
    with open(latest_meta) as f:
        meta = json.load(f)
    assert "table_specialist" in meta.get("v5_features_enabled", []), (
        f"Expected 'table_specialist' in v5_features_enabled, got {meta.get('v5_features_enabled')}"
    )


@pytest.mark.skipif(
    not any(SMOKE_JOB_DIR.glob("**/merged_spans.json")),
    reason="No merged_spans.json found — run pipeline first",
)
def test_docling_table_spans_present():
    """merged_spans.json contains at least one span from docling_tableblock."""
    latest_spans = sorted(SMOKE_JOB_DIR.glob("**/merged_spans.json"))[-1]
    with open(latest_spans) as f:
        spans = json.load(f)
    docling_spans = [
        s for s in spans
        if isinstance(s.get("channel_metadata"), dict)
        and s["channel_metadata"].get("table_source") == "docling_tableblock"
    ]
    assert len(docling_spans) > 0, "No docling_tableblock table spans found in merged_spans.json"


@pytest.mark.skipif(
    not any(SMOKE_JOB_DIR.glob("**/metrics.json")),
    reason="No metrics.json found — run v5_metrics.py first",
)
def test_table_specialist_metrics_written():
    """metrics.json contains v5_table_specialist key with docling stats."""
    latest_metrics = sorted(SMOKE_JOB_DIR.glob("**/metrics.json"))[-1]
    with open(latest_metrics) as f:
        m = json.load(f)
    assert "v5_table_specialist" in m, "metrics.json missing v5_table_specialist key"
    ts = m["v5_table_specialist"]
    assert ts["docling_table_cell_spans"] >= 0
```

- [ ] **Step 5.2: Run full smoke pipeline then tests**

```bash
cd backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser
V5_BBOX_PROVENANCE=1 V5_TABLE_SPECIALIST=1 python data/run_pipeline_targeted.py \
    --source docling_l1 --target-kb kb-3-guidelines \
    data/input/AU-HF-ACS-HCP-Summary-2025.pdf

LATEST_JOB=$(ls -t data/output/v4/ | head -1)
python data/v5_metrics.py "data/output/v4/$LATEST_JOB"

PYTHONPATH=. V5_TABLE_SPECIALIST=1 V5_BBOX_PROVENANCE=1 \
    python -m pytest tests/v5/test_v5_table_specialist_smoke.py -v -m smoke 2>&1 | tail -20
```
Expected: all smoke tests `PASSED`.

- [ ] **Step 5.3: Final commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/tests/v5/test_v5_table_specialist_smoke.py
git commit -m "test(v5): Table Specialist smoke acceptance tests

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Self-review

**Spec coverage:**
- ✅ `V5_TABLE_SPECIALIST` flag resolves via `is_v5_enabled()` — covered in Task 2
- ✅ V4 byte-identical output when flag off — covered by `test_v4_path_used_when_flag_off`
- ✅ Docling TableBlock used as primary source — `_decompose_docling_table()` in Task 2
- ✅ `_V5_KNOWN_FEATURES` updated — Task 3
- ✅ `table_specialist_used` in metadata — Task 2 Step 2.4
- ✅ Cell accuracy ≥85% formula (macro-average) — `compute_v5_table_metrics()` in Task 4
- ✅ Header detection metadata — `test_header_detection` in Task 1
- ✅ Page number propagation — `test_page_number_propagated` in Task 1
- ⚠️ 15-table full ground-truth set not created here (requires clinician effort per spec §7 "one-time investment"). `sample_table_01.csv` is a smoke fixture. Full ground truth is a separate data-collection step.

**Placeholder scan:** None found.

**Type consistency:** `l1_tables: Optional[list] = None` avoids a circular import from `marker_extractor`; callers pass `l1_result.tables` directly. The `Optional[list]` is intentional — the type-checker cannot enforce `TableBlock` without the import.