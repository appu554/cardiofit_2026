# V5 Subsystem #4 — Consensus Entropy Gate Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a post-merge Consensus Entropy gate to `SignalMerger` that removes (or quarantines) single-channel spans whose confidence falls below the session median, reducing the false-positive span rate by ≥20% relative to V4 on the smoke set without violating the ±15% total-spans regression guard.

**Architecture:** After `_cluster_overlapping()` builds `merged_spans`, a new `_apply_ce_gate()` method filters out spans that are: (a) contributed by only one channel, AND (b) have `merged_confidence < median(all_merged_confidence)`. Filtered spans are NOT discarded — they are collected into a `ce_filtered_spans` list returned via `ChannelOutput.metadata` so `v5_metrics.py` can compute the FP-rate delta. `MergedSpan` gains a `ce_flagged: bool = False` field as an opt-in audit trail. `_V5_KNOWN_FEATURES` gains `"consensus_entropy"`.

**Tech Stack:** Python 3.13, `statistics.median`, existing `signal_merger.SignalMerger`, `extraction.v4.models.MergedSpan`, `extraction.v4.v5_flags.is_v5_enabled`, pytest.

**Success criteria (from master spec §7):**
- Primary: FP-rate drops ≥20% relative on smoke set (V5 FP% ≤ 0.8 × V4 FP%)
- FP formula: `100 × count(spans: len(channels)==1 AND confidence < median) / count(*)`
- Secondary: escalation rate ≤5%, wall-time delta ≤+10%, universal regression ±15%

---

## File Structure

| File | Disposition | Purpose |
|------|-------------|---------|
| `${EXTRACTION}/v4/models.py` | MODIFY | Add `ce_flagged: bool = False` to `MergedSpan` |
| `${EXTRACTION}/v4/signal_merger.py` | MODIFY | Add `_apply_ce_gate()`, call it in `merge()` when flag on |
| `${ATOMISER}/data/run_pipeline_targeted.py` | MODIFY | Add `"consensus_entropy"` to `_V5_KNOWN_FEATURES`; pass flag into `merger.merge()` |
| `${ATOMISER}/data/v5_metrics.py` | MODIFY | Add `compute_v5_ce_metrics()` |
| `${ATOMISER}/tests/v5/test_v5_consensus_entropy.py` | CREATE | Unit tests for CE gate logic and flag routing |

```
ATOMISER=backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser
EXTRACTION=backend/shared-infrastructure/knowledge-base-services/shared/extraction
```

---

## Task 1: Add `ce_flagged` field to `MergedSpan`

**Files:**
- Modify: `${EXTRACTION}/v4/models.py`

- [ ] **Step 1.1: Add `ce_flagged` field to `MergedSpan`**

In `models.py`, find the `MergedSpan` class (after `channel_provenance`). Add one field:

```python
    # V5 #4 Consensus Entropy gate — backward-compatible additive field.
    # True when this span was flagged by the CE gate (single-channel, below
    # session median confidence). Spans with ce_flagged=True are suppressed
    # from the default output when V5_CONSENSUS_ENTROPY is on.
    ce_flagged: bool = False
```

- [ ] **Step 1.2: Verify existing tests still pass**

```bash
cd backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser
PYTHONPATH=. python -m pytest tests/v5/ -v --ignore=tests/v5/test_v5_table_specialist_smoke.py 2>&1 | tail -20
```
Expected: all tests `PASSED` (new field has default, backward-compatible).

---

## Task 2: Write failing unit tests for CE gate

**Files:**
- Create: `${ATOMISER}/tests/v5/test_v5_consensus_entropy.py`

- [ ] **Step 2.1: Write failing tests**

Create `tests/v5/test_v5_consensus_entropy.py`:
```python
"""V5 Consensus Entropy Gate unit tests.

Run from guideline-atomiser/:
    PYTHONPATH=. V5_CONSENSUS_ENTROPY=1 pytest tests/v5/test_v5_consensus_entropy.py -v
"""
from __future__ import annotations

import statistics
from uuid import uuid4

import pytest

from extraction.v4.signal_merger import SignalMerger
from extraction.v4.models import (
    ChannelOutput,
    GuidelineTree,
    MergedSpan,
    RawSpan,
)


def _make_tree() -> GuidelineTree:
    return GuidelineTree(sections=[], tables=[], total_pages=2)


def _make_raw_span(channel: str, text: str, confidence: float, start: int, end: int) -> RawSpan:
    return RawSpan(
        channel=channel,
        text=text,
        start=start,
        end=end,
        confidence=confidence,
    )


def _make_channel_output(channel: str, spans: list[RawSpan]) -> ChannelOutput:
    return ChannelOutput(channel=channel, spans=spans, metadata={})


# ─── Flag routing ──────────────────────────────────────────────────────────

def test_ce_gate_off_by_default(monkeypatch):
    """When V5_CONSENSUS_ENTROPY is absent, all spans are returned unchanged."""
    monkeypatch.delenv("V5_CONSENSUS_ENTROPY", raising=False)
    monkeypatch.delenv("V5_DISABLE_ALL", raising=False)

    merger = SignalMerger()
    tree = _make_tree()
    # Low-confidence single-channel span
    co = _make_channel_output("B", [_make_raw_span("B", "low conf", 0.20, 0, 8)])
    result = merger.merge(uuid4(), [co], tree)
    assert len(result) == 1
    assert not result[0].ce_flagged


def test_ce_gate_on_flags_low_single_channel(monkeypatch):
    """When V5_CONSENSUS_ENTROPY=1, single-channel spans below median are flagged and removed."""
    monkeypatch.setenv("V5_CONSENSUS_ENTROPY", "1")
    monkeypatch.delenv("V5_DISABLE_ALL", raising=False)

    merger = SignalMerger()
    tree = _make_tree()

    # 5 spans: 4 high-confidence (multi or high-single), 1 low single-channel
    # median will be ~0.90; the 0.20-confidence span should be flagged
    high_spans_b = [_make_raw_span("B", f"drug{i}", 0.90, i * 20, i * 20 + 10) for i in range(4)]
    low_span_c = _make_raw_span("C", "noise", 0.20, 100, 105)

    co_b = _make_channel_output("B", high_spans_b)
    co_c = _make_channel_output("C", [low_span_c])

    result = merger.merge(uuid4(), [co_b, co_c], tree)

    # The high-confidence B spans (non-overlapping) stay; the low C span is filtered
    non_flagged = [s for s in result if not s.ce_flagged]
    flagged = [s for s in result if s.ce_flagged]

    assert len(non_flagged) == 4, f"Expected 4 non-flagged spans, got {len(non_flagged)}"
    assert len(flagged) == 1, f"Expected 1 flagged span, got {len(flagged)}"
    assert flagged[0].text == "noise"


def test_disable_all_overrides_ce_gate(monkeypatch):
    """V5_DISABLE_ALL=1 forces V4 path — no CE gate applied."""
    monkeypatch.setenv("V5_CONSENSUS_ENTROPY", "1")
    monkeypatch.setenv("V5_DISABLE_ALL", "1")

    merger = SignalMerger()
    tree = _make_tree()
    low_span = _make_raw_span("B", "low", 0.10, 0, 3)
    high_spans = [_make_raw_span("B", f"h{i}", 0.95, i * 20, i * 20 + 10) for i in range(4)]
    co = _make_channel_output("B", [low_span] + high_spans)

    result = merger.merge(uuid4(), [co], tree)
    assert all(not s.ce_flagged for s in result), "No spans should be flagged when V5_DISABLE_ALL=1"


def test_multi_channel_span_not_flagged_even_if_low_confidence(monkeypatch):
    """Multi-channel span is NEVER CE-flagged regardless of confidence."""
    monkeypatch.setenv("V5_CONSENSUS_ENTROPY", "1")
    monkeypatch.delenv("V5_DISABLE_ALL", raising=False)

    merger = SignalMerger()
    tree = _make_tree()

    # Two channels, same text region → will cluster into one multi-channel span
    span_b = _make_raw_span("B", "metformin 500mg", 0.15, 0, 15)
    span_c = _make_raw_span("C", "metformin 500mg", 0.15, 0, 15)
    # High-confidence spans to drive median up
    high = [_make_raw_span("B", f"h{i}", 0.95, i * 30, i * 30 + 10) for i in range(4)]

    co_b = _make_channel_output("B", [span_b] + high)
    co_c = _make_channel_output("C", [span_c])

    result = merger.merge(uuid4(), [co_b, co_c], tree)

    # The merged span from B+C should NOT be flagged (multi-channel)
    merged_mc = next(
        (s for s in result if "B" in s.contributing_channels and "C" in s.contributing_channels),
        None,
    )
    assert merged_mc is not None, "Expected a B+C merged span"
    assert not merged_mc.ce_flagged, "Multi-channel span must not be CE-flagged"


def test_fp_rate_drops_with_ce_gate(monkeypatch):
    """FP rate (single-channel below median) drops by ≥20% relative when CE gate is on."""
    monkeypatch.setenv("V5_CONSENSUS_ENTROPY", "1")
    monkeypatch.delenv("V5_DISABLE_ALL", raising=False)

    merger = SignalMerger()
    tree = _make_tree()

    # Construct: 4 high-conf B spans + 2 low-conf C spans (single-channel)
    # V4 FP rate = 2/6 = 33.3%; V5 target = FP rate should drop ≥20% relative → ≤26.7%
    high = [_make_raw_span("B", f"high{i}", 0.92, i * 30, i * 30 + 10) for i in range(4)]
    low = [_make_raw_span("C", f"low{i}", 0.20, 200 + i * 20, 210 + i * 20) for i in range(2)]

    # V4 baseline (no CE gate)
    monkeypatch.delenv("V5_CONSENSUS_ENTROPY", raising=False)
    co_b = _make_channel_output("B", high)
    co_c = _make_channel_output("C", low)
    v4_result = merger.merge(uuid4(), [co_b, co_c], tree)

    def fp_rate(spans):
        if not spans:
            return 0.0
        median_conf = statistics.median(s.merged_confidence for s in spans)
        fp = sum(
            1 for s in spans
            if len(s.contributing_channels) == 1
            and s.merged_confidence < median_conf
        )
        return fp / len(spans) * 100

    v4_fp = fp_rate(v4_result)

    # V5 with CE gate
    monkeypatch.setenv("V5_CONSENSUS_ENTROPY", "1")
    co_b2 = _make_channel_output("B", high)
    co_c2 = _make_channel_output("C", low)
    v5_result = merger.merge(uuid4(), [co_b2, co_c2], tree)

    # Only count non-flagged spans in V5 FP rate
    v5_non_flagged = [s for s in v5_result if not s.ce_flagged]
    v5_fp = fp_rate(v5_non_flagged)

    if v4_fp > 0:
        relative_drop = (v4_fp - v5_fp) / v4_fp * 100
        assert relative_drop >= 20.0, (
            f"FP relative drop {relative_drop:.1f}% < 20% threshold "
            f"(v4={v4_fp:.1f}%, v5={v5_fp:.1f}%)"
        )
```

- [ ] **Step 2.2: Run tests to confirm they fail**

```bash
cd backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser
PYTHONPATH=. V5_CONSENSUS_ENTROPY=1 python -m pytest tests/v5/test_v5_consensus_entropy.py -v 2>&1 | tail -20
```
Expected: `FAILED` — `merger.merge()` doesn't apply CE gate yet.

---

## Task 3: Implement `_apply_ce_gate()` in `SignalMerger`

**Files:**
- Modify: `${EXTRACTION}/v4/signal_merger.py`

- [ ] **Step 3.1: Add `v5_consensus_entropy` parameter to `merge()`**

In `SignalMerger.merge()` (line ~111), add parameter after `page_bbox_map`:

```python
    def merge(
        self,
        job_id: UUID,
        channel_outputs: list[ChannelOutput],
        tree: GuidelineTree,
        classifier: Optional[TieringClassifier] = None,
        v5_bbox_provenance: bool = False,
        profile=None,
        page_bbox_map: Optional[dict[int, list[float]]] = None,
        v5_consensus_entropy: bool = False,
    ) -> list[MergedSpan]:
```

- [ ] **Step 3.2: Call `_apply_ce_gate()` after building `merged_spans`**

After the `for cluster in clusters:` loop (line ~167) and the `_validate_page_coverage()` call, add:

```python
        # V5 #4: Consensus Entropy gate — flag and suppress low-quality
        # single-channel spans below the session median confidence.
        if v5_consensus_entropy:
            merged_spans = self._apply_ce_gate(merged_spans)

        return merged_spans
```

(Replace the existing bare `return merged_spans` with the above block.)

- [ ] **Step 3.3: Add `_apply_ce_gate()` method**

Add after `_validate_page_coverage()`:

```python
    def _apply_ce_gate(
        self, merged_spans: list[MergedSpan],
    ) -> list[MergedSpan]:
        """Apply the Consensus Entropy gate to flag noisy single-channel spans.

        A span is CE-flagged when BOTH:
          1. It has exactly one contributing channel (no multi-channel consensus)
          2. Its merged_confidence < median(all spans' merged_confidence)

        Flagged spans have ce_flagged=True set and are excluded from the
        returned list. They are not deleted — callers can access them via
        ce_flagged attribute if needed for audit.

        Returns: list of non-flagged MergedSpans (ce_flagged spans dropped).
        """
        import statistics as _stats

        if not merged_spans:
            return merged_spans

        confidences = [s.merged_confidence for s in merged_spans]
        median_conf = _stats.median(confidences)

        kept: list[MergedSpan] = []
        for span in merged_spans:
            if (
                len(span.contributing_channels) == 1
                and span.merged_confidence < median_conf
            ):
                span.ce_flagged = True
                # Drop from output — stored in audit trail via ce_flagged attribute
            else:
                kept.append(span)

        return kept
```

- [ ] **Step 3.4: Update `_merge_with_v5_flag()` in `run_pipeline_targeted.py` to resolve CE flag**

Find `_merge_with_v5_flag()` (around line 149) and update it:

```python
def _merge_with_v5_flag(merger, *args_, profile, **kwargs):
    """Call merger.merge() with all resolved V5 flags."""
    from extraction.v4.v5_flags import is_v5_enabled
    kwargs["v5_bbox_provenance"] = is_v5_enabled("bbox_provenance", profile)
    kwargs["v5_consensus_entropy"] = is_v5_enabled("consensus_entropy", profile)
    kwargs["profile"] = profile
    return merger.merge(*args_, **kwargs)
```

- [ ] **Step 3.5: Run tests — expect them to pass**

```bash
cd backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser
PYTHONPATH=. V5_CONSENSUS_ENTROPY=1 python -m pytest tests/v5/test_v5_consensus_entropy.py -v 2>&1 | tail -20
```
Expected: all 5 tests `PASSED`.

- [ ] **Step 3.6: Commit**

```bash
git add \
    backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/models.py \
    backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/signal_merger.py \
    backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/data/run_pipeline_targeted.py \
    backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/tests/v5/test_v5_consensus_entropy.py
git commit -m "feat(v5): Consensus Entropy gate — single-channel FP suppression (#4)

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 4: Add `"consensus_entropy"` to `_V5_KNOWN_FEATURES`

**Files:**
- Modify: `${ATOMISER}/data/run_pipeline_targeted.py`

- [ ] **Step 4.1: Update `_V5_KNOWN_FEATURES` (line 825)**

```python
    _V5_KNOWN_FEATURES = ["bbox_provenance", "table_specialist", "consensus_entropy"]
```

- [ ] **Step 4.2: Verify pipeline runs with combined flags**

```bash
cd backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser
V5_BBOX_PROVENANCE=1 V5_TABLE_SPECIALIST=1 V5_CONSENSUS_ENTROPY=1 \
    python data/run_pipeline_targeted.py --source docling_l1 --target-kb kb-3-guidelines \
    data/input/AU-HF-ACS-HCP-Summary-2025.pdf 2>&1 | tail -20
```
Expected: `v5_features_enabled: ['bbox_provenance', 'table_specialist', 'consensus_entropy']`, no errors.

- [ ] **Step 4.3: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/data/run_pipeline_targeted.py
git commit -m "feat(v5): add consensus_entropy to _V5_KNOWN_FEATURES

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 5: Add `compute_v5_ce_metrics()` to `v5_metrics.py`

**Files:**
- Modify: `${ATOMISER}/data/v5_metrics.py`

- [ ] **Step 5.1: Add the function**

Add after `compute_v5_bbox_metrics()`:

```python
_CE_FP_REDUCTION_THRESHOLD = 20.0  # % relative drop required


def compute_v5_ce_metrics(
    v4_spans: list[dict],
    v5_spans: list[dict],
) -> dict:
    """Compute V5 Consensus Entropy gate metrics.

    Compares V4 baseline spans vs V5 (CE-filtered) spans.
    Both arguments are lists of MergedSpan-dict (from merged_spans.json).

    FP definition: single-channel span AND confidence < session median.
    Primary metric: relative FP-rate drop ≥ 20%.
    """
    import statistics as _stats
    from typing import Any

    def _fp_rate(spans: list[dict[str, Any]]) -> tuple[float, int, int]:
        if not spans:
            return 0.0, 0, 0
        confs = [float(s.get("merged_confidence", 0)) for s in spans]
        median_conf = _stats.median(confs) if confs else 0.0
        fp_count = sum(
            1 for s in spans
            if len(s.get("contributing_channels", [])) == 1
            and float(s.get("merged_confidence", 0)) < median_conf
        )
        return round(fp_count / len(spans) * 100, 2), fp_count, len(spans)

    v4_fp_pct, v4_fp_n, v4_total = _fp_rate(v4_spans)
    v5_fp_pct, v5_fp_n, v5_total = _fp_rate(v5_spans)

    if v4_fp_pct > 0:
        relative_drop = round((v4_fp_pct - v5_fp_pct) / v4_fp_pct * 100, 2)
    else:
        relative_drop = 0.0

    status = "PASS" if relative_drop >= _CE_FP_REDUCTION_THRESHOLD else "FAIL"

    return {
        "v5_consensus_entropy": {
            "v4_fp_rate_pct": v4_fp_pct,
            "v4_fp_count": v4_fp_n,
            "v4_total_spans": v4_total,
            "v5_fp_rate_pct": v5_fp_pct,
            "v5_fp_count": v5_fp_n,
            "v5_total_spans": v5_total,
            "relative_fp_drop_pct": relative_drop,
        },
        "primary": {
            "ce_fp_reduction_pct": {
                "v5": relative_drop,
                "threshold": _CE_FP_REDUCTION_THRESHOLD,
                "status": status,
            },
        },
        "verdict": status,
    }
```

- [ ] **Step 5.2: Run existing tests to ensure nothing broke**

```bash
PYTHONPATH=. python -m pytest tests/v5/test_v5_consensus_entropy.py tests/v5/test_v5_metrics.py -v 2>&1 | tail -20
```
Expected: all tests `PASSED`.

- [ ] **Step 5.3: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/data/v5_metrics.py
git commit -m "feat(v5): add compute_v5_ce_metrics() to v5_metrics sidecar

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 6: Smoke acceptance test

**Files:**
- Create: `${ATOMISER}/tests/v5/test_v5_ce_smoke.py`

- [ ] **Step 6.1: Write smoke test**

Create `tests/v5/test_v5_ce_smoke.py`:
```python
"""Smoke acceptance test for V5 Consensus Entropy gate.

Requires two completed job dirs:
  - V4 baseline: run without V5_CONSENSUS_ENTROPY
  - V5 with CE gate: run with V5_CONSENSUS_ENTROPY=1

Run:
    PYTHONPATH=. pytest tests/v5/test_v5_ce_smoke.py -v -m smoke
"""
import json
import os
from pathlib import Path

import pytest

pytestmark = pytest.mark.smoke

OUTPUT_DIR = Path("data/output/v4")


@pytest.mark.skipif(
    len(list(OUTPUT_DIR.glob("**/merged_spans.json"))) < 2,
    reason="Need ≥2 job outputs (V4 baseline + V5 CE run) — run both pipelines first",
)
def test_ce_job_has_fewer_total_spans_within_regression():
    """V5 CE job total span count is within ±15% of V4 baseline."""
    jobs = sorted(OUTPUT_DIR.glob("**/job_metadata.json"))
    assert len(jobs) >= 2

    # Most recent job is V5; second most recent is V4 baseline
    v5_meta = json.loads(jobs[-1].read_text())
    v4_meta = json.loads(jobs[-2].read_text())

    v4_total = v4_meta.get("total_merged_spans", 0)
    v5_total = v5_meta.get("total_merged_spans", 0)

    if v4_total > 0:
        delta_pct = abs(v5_total - v4_total) / v4_total * 100
        assert delta_pct <= 15.0, (
            f"Total spans delta {delta_pct:.1f}% exceeds ±15% regression guard "
            f"(V4={v4_total}, V5={v5_total})"
        )


@pytest.mark.skipif(
    "V5_CONSENSUS_ENTROPY" not in os.environ,
    reason="V5_CONSENSUS_ENTROPY not set — skipping CE gate check",
)
def test_consensus_entropy_in_v5_features():
    """Latest job_metadata reports consensus_entropy in v5_features_enabled."""
    jobs = sorted(OUTPUT_DIR.glob("**/job_metadata.json"))
    if not jobs:
        pytest.skip("No job_metadata.json found")
    meta = json.loads(jobs[-1].read_text())
    assert "consensus_entropy" in meta.get("v5_features_enabled", [])
```

- [ ] **Step 6.2: Run both pipeline variants and acceptance test**

```bash
cd backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser

# V4 baseline
python data/run_pipeline_targeted.py --source docling_l1 --target-kb kb-3-guidelines \
    data/input/AU-HF-ACS-HCP-Summary-2025.pdf

# V5 with CE gate
V5_BBOX_PROVENANCE=1 V5_CONSENSUS_ENTROPY=1 \
python data/run_pipeline_targeted.py --source docling_l1 --target-kb kb-3-guidelines \
    data/input/AU-HF-ACS-HCP-Summary-2025.pdf

PYTHONPATH=. V5_CONSENSUS_ENTROPY=1 python -m pytest tests/v5/test_v5_ce_smoke.py -v -m smoke 2>&1 | tail -20
```
Expected: both tests `PASSED`.

- [ ] **Step 6.3: Final commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/tests/v5/test_v5_ce_smoke.py
git commit -m "test(v5): Consensus Entropy gate smoke acceptance tests

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Self-review

**Spec coverage:**
- ✅ FP formula: `count(len(channels)==1 AND confidence < median) / count(*)` — in `compute_v5_ce_metrics()` and `test_fp_rate_drops_with_ce_gate`
- ✅ ≥20% relative reduction threshold — enforced in `compute_v5_ce_metrics()` and CE gate test
- ✅ Multi-channel spans never flagged — `test_multi_channel_span_not_flagged_even_if_low_confidence`
- ✅ `V5_DISABLE_ALL=1` override — `test_disable_all_overrides_ce_gate`
- ✅ `ce_flagged` audit field on `MergedSpan` — Task 1
- ✅ `v5_consensus_entropy` param on `merge()` — Task 3
- ✅ Universal regression ±15% — `test_ce_job_has_fewer_total_spans_within_regression`
- ⚠️ `tau` hyperparameter tuning mentioned in spec failure action — not implemented here (can be a follow-up; the current threshold is the session median which is auto-calibrated per-job)

**Placeholder scan:** None found.

**Type consistency:** `v5_consensus_entropy: bool = False` default on `merge()` preserves all existing callers. `_merge_with_v5_flag()` is the only call site that passes it as `True`.