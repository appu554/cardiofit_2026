# V5 Subsystem A — Bbox Provenance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire full bounding-box + per-channel attribution provenance through every merged span in Pipeline 1 V4, gated by the `V5_BBOX_PROVENANCE` feature flag, so every downstream consumer (KB-0 dashboard, audit trail, V5 subsystem #1/#3/#4/#5 comparisons) can trace any span back to the channels that contributed it, the bbox they observed, the model versions they used, and their per-channel confidence.

**Architecture:** Approach C (additive layers atop V4) — see `docs/superpowers/specs/2026-05-01-v5-master-architecture-design.md`. Adds a single `v5_flags.is_v5_enabled()` resolver, a `ChannelProvenance` Pydantic model, an extra jsonb field on the existing `MergedSpan`, threading through 8 channels + the signal merger + `push_to_kb0_gcp.py` + a new `metrics.json` sidecar. KB-0 schema gets one nullable `provenance_v5 jsonb` column; backward-compatible. Default-off until smoke test shows ≥99% coverage on AU-HF-ACS-HCP-Summary.

**Tech Stack:** Python 3.13, Pydantic v2, psycopg2, pytest, PostgreSQL 15 (Cloud SQL on GCP), existing `extraction.v4.*` modules from `backend/shared-infrastructure/knowledge-base-services/shared/`.

**Success criteria (from master spec §7):**
- Primary: ≥99% of merged spans have non-null `bbox` AND non-empty `channel_provenance` jsonb on smoke set
- Secondary: per-channel bbox ≥95%, KB-0 round-trip byte-identical, bbox coords within page bounds 100%
- Universal regression: total spans within ±15% of V4 baseline, TIER_1 prop ≥ V4, KB-0 push success 100%, 0 new ERRORs

---

## File Structure

| File | Disposition | Purpose |
|------|-------------|---------|
| `backend/.../shared/extraction/v4/v5_flags.py` | **CREATE** | Single `is_v5_enabled(feature, profile)` resolver; only place that touches env + profile precedence |
| `backend/.../shared/extraction/v4/provenance.py` | **CREATE** | `ChannelProvenance` Pydantic model; helpers to merge/serialise lists |
| `backend/.../shared/extraction/v4/guideline_profile.py` | MODIFY | Add `v5_features: dict[str, bool \| None]` field, default empty |
| `backend/.../shared/extraction/v4/models.py` | MODIFY | Extend `MergedSpan` with optional `channel_provenance: list[ChannelProvenance]` |
| `backend/.../shared/extraction/v4/channel_a_docling.py` | MODIFY | When V5 flag on, emit ChannelProvenance entries with bbox + model_version |
| `backend/.../shared/extraction/v4/channel_b_drug_dict.py` | MODIFY | (same — drug-dict-derived bbox via parent block) |
| `backend/.../shared/extraction/v4/channel_c_grammar.py` | MODIFY | (same — regex-region bbox) |
| `backend/.../shared/extraction/v4/channel_d_table.py` | MODIFY | (same — table-cell bboxes) |
| `backend/.../shared/extraction/v4/channel_e_gliner.py` | MODIFY | (same — entity-region bbox) |
| `backend/.../shared/extraction/v4/channel_f_nuextract.py` | MODIFY | (same — NuExtract emits with parent-passage bbox) |
| `backend/.../shared/extraction/v4/channel_g_sentence.py` | MODIFY | (same — sentence-level bbox) |
| `backend/.../shared/extraction/v4/channel_h_recovery.py` | MODIFY | (same — recovery uses original raw-span bbox) |
| `backend/.../shared/extraction/v4/signal_merger.py` | MODIFY | Concat channel_provenance lists when merging spans across channels |
| `backend/.../shared/tools/guideline-atomiser/data/run_pipeline_targeted.py` | MODIFY | Resolve V5 flags at start; thread profile into channel constructors |
| `backend/.../shared/tools/guideline-atomiser/data/v5_metrics.py` | **CREATE** | Sidecar `metrics.json` generator; computes universal regression + #2 primary metrics |
| `backend/.../shared/tools/guideline-atomiser/data/push_to_kb0_gcp.py` | MODIFY | Write `provenance_v5` jsonb column when present in local job |
| `backend/.../kb-0-governance-platform/migrations/009_l2_provenance_v5.sql` | **CREATE** | Add `provenance_v5 jsonb` nullable column to `l2_merged_spans` |
| `backend/.../shared/tools/guideline-atomiser/data/profiles/heart_foundation_au_2025.yaml` | MODIFY | Add commented-out `v5_features` example |
| `backend/.../shared/tools/guideline-atomiser/tests/v5/__init__.py` | **CREATE** | Test package init |
| `backend/.../shared/tools/guideline-atomiser/tests/v5/test_v5_flags.py` | **CREATE** | Unit tests for `is_v5_enabled` precedence |
| `backend/.../shared/tools/guideline-atomiser/tests/v5/test_provenance_model.py` | **CREATE** | Unit tests for `ChannelProvenance` model + merger helpers |
| `backend/.../shared/tools/guideline-atomiser/tests/v5/test_smoke_bbox_coverage.py` | **CREATE** | Acceptance test: ≥99% bbox coverage on smoke set |
| `backend/.../shared/tools/guideline-atomiser/tests/v5/test_kb0_round_trip.py` | **CREATE** | Acceptance test: provenance_v5 round-trips through KB-0 byte-identical |
| `backend/shared-infrastructure/knowledge-base-services/README_AU.md` | MODIFY | Add V5 flags section |
| `docs/superpowers/plans/2026-05-01-v5-bbox-provenance.md` | (this file) | The plan itself |

**Common path prefix used in tasks below**:
```
ATOMISER=backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser
EXTRACTION=backend/shared-infrastructure/knowledge-base-services/shared/extraction
KB0=backend/shared-infrastructure/knowledge-base-services/kb-0-governance-platform
```

---

## Task 1: Test scaffolding for V5 acceptance tests

**Files:**
- Create: `${ATOMISER}/tests/v5/__init__.py`
- Create: `${ATOMISER}/tests/v5/conftest.py`

- [ ] **Step 1.1: Create `tests/v5/__init__.py` (empty file marks it as a package)**

```python
# tests/v5/__init__.py
"""V5 acceptance tests — bbox provenance, table specialist, etc.

These tests use the smoke set defined in the master spec §6:
  - AU-HF-ACS-HCP-Summary-2025.pdf (2 pages)
  - AU-HF-Cholesterol-Action-Plan-2026.pdf (5 pages)

Run from the guideline-atomiser dir:
  PYTHONPATH=. ../../../.venv13/bin/python -m pytest tests/v5/ -v
"""
```

- [ ] **Step 1.2: Create `tests/v5/conftest.py` with shared fixtures**

```python
# tests/v5/conftest.py
"""Shared pytest fixtures for V5 acceptance tests."""
from __future__ import annotations

import json
from pathlib import Path
from typing import Any

import pytest

REPO_ROOT = Path(__file__).resolve().parents[3]
ATOMISER_DIR = REPO_ROOT / "shared/tools/guideline-atomiser"
SMOKE_PDFS = [
    "AU-HF-ACS-HCP-Summary-2025.pdf",
    "AU-HF-Cholesterol-Action-Plan-2026.pdf",
]


@pytest.fixture(scope="session")
def atomiser_dir() -> Path:
    return ATOMISER_DIR


@pytest.fixture(scope="session")
def v4_baseline_jobs(atomiser_dir: Path) -> dict[str, Path]:
    """Locate V4 baseline job dirs for the smoke set, by source_pdf."""
    output_dir = atomiser_dir / "data/output/v4"
    found: dict[str, Path] = {}
    for job_dir in output_dir.glob("job_monkeyocr_*"):
        meta_path = job_dir / "job_metadata.json"
        if not meta_path.exists():
            continue
        meta = json.loads(meta_path.read_text())
        src = meta.get("source_pdf")
        if src in SMOKE_PDFS and src not in found:
            found[src] = job_dir
    return found


def load_merged_spans(job_dir: Path) -> list[dict[str, Any]]:
    """Load merged_spans.json as list of dicts."""
    return json.loads((job_dir / "merged_spans.json").read_text())
```

- [ ] **Step 1.3: Verify the test directory is discovered by pytest**

Run:
```bash
cd backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser
.venv13/bin/python -m pytest tests/v5/ -v --collect-only
```
Expected: `collected 0 items` (no tests yet — directory recognised). No import errors.

- [ ] **Step 1.4: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/tests/v5/__init__.py \
        backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/tests/v5/conftest.py
git -c commit.gpgsign=false commit -m "test(v5): scaffold tests/v5/ package + shared fixtures"
```

---

## Task 2: `v5_flags.is_v5_enabled()` resolver

**Files:**
- Create: `${EXTRACTION}/v4/v5_flags.py`
- Create: `${ATOMISER}/tests/v5/test_v5_flags.py`

- [ ] **Step 2.1: Write failing tests for the resolver**

```python
# tests/v5/test_v5_flags.py
"""Unit tests for v5_flags.is_v5_enabled() precedence rules."""
from __future__ import annotations

import os
from dataclasses import dataclass, field

import pytest

from extraction.v4.v5_flags import is_v5_enabled


@dataclass
class _FakeProfile:
    v5_features: dict[str, object | None] = field(default_factory=dict)


@pytest.fixture(autouse=True)
def _clean_env(monkeypatch: pytest.MonkeyPatch) -> None:
    """Strip any V5_* env vars before each test."""
    for k in list(os.environ.keys()):
        if k.startswith("V5_"):
            monkeypatch.delenv(k, raising=False)


def test_default_off_when_nothing_set() -> None:
    profile = _FakeProfile()
    assert is_v5_enabled("bbox_provenance", profile) is False


def test_env_var_on_enables(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("V5_BBOX_PROVENANCE", "1")
    profile = _FakeProfile()
    assert is_v5_enabled("bbox_provenance", profile) is True


def test_env_var_other_values_off(monkeypatch: pytest.MonkeyPatch) -> None:
    for v in ("0", "false", "no", "", "anything-not-1"):
        monkeypatch.setenv("V5_BBOX_PROVENANCE", v)
        profile = _FakeProfile()
        assert is_v5_enabled("bbox_provenance", profile) is False, f"value={v!r}"


def test_profile_override_on_wins_over_env_off() -> None:
    profile = _FakeProfile(v5_features={"bbox_provenance": True})
    assert is_v5_enabled("bbox_provenance", profile) is True


def test_profile_override_off_wins_over_env_on(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("V5_BBOX_PROVENANCE", "1")
    profile = _FakeProfile(v5_features={"bbox_provenance": False})
    assert is_v5_enabled("bbox_provenance", profile) is False


def test_profile_override_none_falls_through_to_env(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("V5_BBOX_PROVENANCE", "1")
    profile = _FakeProfile(v5_features={"bbox_provenance": None})
    assert is_v5_enabled("bbox_provenance", profile) is True


def test_disable_all_overrides_everything(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("V5_DISABLE_ALL", "1")
    monkeypatch.setenv("V5_BBOX_PROVENANCE", "1")
    profile = _FakeProfile(v5_features={"bbox_provenance": True})
    assert is_v5_enabled("bbox_provenance", profile) is False


def test_profile_without_v5_features_attr() -> None:
    """Older profiles without v5_features field should default to off."""
    profile = object()  # no v5_features attr at all
    assert is_v5_enabled("bbox_provenance", profile) is False


def test_unknown_feature_name_off() -> None:
    profile = _FakeProfile()
    assert is_v5_enabled("nonexistent_feature", profile) is False
```

- [ ] **Step 2.2: Run tests to confirm they fail (no implementation yet)**

Run:
```bash
cd backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser
.venv13/bin/python -m pytest tests/v5/test_v5_flags.py -v
```
Expected: 9 tests FAIL with `ModuleNotFoundError: No module named 'extraction.v4.v5_flags'`.

- [ ] **Step 2.3: Implement `v5_flags.py`**

Create `backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/v5_flags.py`:

```python
"""V5 feature flag resolver.

Single source of truth for V5_<feature> on/off decisions. Precedence:

    1. V5_DISABLE_ALL=1 in env  -> always False (emergency rollback)
    2. profile.v5_features[feature] is True/False  -> profile wins
    3. profile.v5_features[feature] is None        -> fall through
    4. env V5_<FEATURE>=1                          -> True
    5. anything else                               -> False (default-off)

`profile` may be any object; if it lacks `v5_features`, treat as empty dict.
"""
from __future__ import annotations

import os
from typing import Any


def is_v5_enabled(feature: str, profile: Any) -> bool:
    """Resolve a V5 feature flag with profile-override > env-var > default-off.

    Args:
        feature: lowercase feature name, e.g. "bbox_provenance".
        profile: object with optional `v5_features` dict attribute.

    Returns:
        True iff the resolved value is on.
    """
    # Emergency rollback always wins.
    if os.environ.get("V5_DISABLE_ALL") == "1":
        return False

    # Profile override (if present and not None) wins over env.
    overrides = getattr(profile, "v5_features", None) or {}
    profile_value = overrides.get(feature)
    if profile_value is True:
        return True
    if profile_value is False:
        return False

    # Env var fallback.
    env_value = os.environ.get(f"V5_{feature.upper()}", "0")
    return env_value == "1"
```

- [ ] **Step 2.4: Run tests to confirm they pass**

Run:
```bash
.venv13/bin/python -m pytest tests/v5/test_v5_flags.py -v
```
Expected: 9 passed in <1s.

- [ ] **Step 2.5: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/v5_flags.py \
        backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/tests/v5/test_v5_flags.py
git -c commit.gpgsign=false commit -m "feat(v5): is_v5_enabled() flag resolver with profile + env precedence"
```

---

## Task 3: Add `v5_features` field to `GuidelineProfile`

**Files:**
- Modify: `${EXTRACTION}/v4/guideline_profile.py`
- Modify: `${ATOMISER}/data/profiles/heart_foundation_au_2025.yaml`

- [ ] **Step 3.1: Read the current profile dataclass to find the right insertion point**

```bash
grep -n "@dataclass\|class GuidelineProfile\|tiering_classifier" \
  backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/guideline_profile.py
```
Expected: locate the `@dataclass class GuidelineProfile:` decorator + the last field (`tiering_classifier`). The new field gets added at the end.

- [ ] **Step 3.2: Write a unit test for loading a profile with `v5_features`**

Append to `tests/v5/test_v5_flags.py`:

```python
def test_profile_loads_v5_features_from_yaml(tmp_path) -> None:
    """GuidelineProfile.from_yaml() round-trips v5_features dict."""
    from extraction.v4.guideline_profile import GuidelineProfile
    yaml_text = """
profile_id: test
display_name: Test
authority: TEST
document_title: Test
effective_date: "2026-01-01"
doi: ""
version: "1.0"
pdf_sources: {main: foo.pdf}
extra_drug_ingredients: {}
extra_drug_classes: {}
extra_patterns: []
drug_class_skip_list: []
reference_section_headings: []
tiering_classifier: rule_based
v5_features:
  bbox_provenance: true
  table_specialist: false
  consensus_entropy: null
"""
    p = tmp_path / "test.yaml"
    p.write_text(yaml_text)
    profile = GuidelineProfile.from_yaml(str(p))
    assert profile.v5_features == {
        "bbox_provenance": True,
        "table_specialist": False,
        "consensus_entropy": None,
    }


def test_profile_without_v5_features_yaml_section(tmp_path) -> None:
    """Old profiles without v5_features still load (backward compat)."""
    from extraction.v4.guideline_profile import GuidelineProfile
    yaml_text = """
profile_id: test
display_name: Test
authority: TEST
document_title: Test
effective_date: "2026-01-01"
doi: ""
version: "1.0"
pdf_sources: {main: foo.pdf}
extra_drug_ingredients: {}
extra_drug_classes: {}
extra_patterns: []
drug_class_skip_list: []
reference_section_headings: []
tiering_classifier: rule_based
"""
    p = tmp_path / "test.yaml"
    p.write_text(yaml_text)
    profile = GuidelineProfile.from_yaml(str(p))
    # Default empty dict, NOT None
    assert profile.v5_features == {}
```

- [ ] **Step 3.3: Run new tests to confirm they fail**

```bash
.venv13/bin/python -m pytest tests/v5/test_v5_flags.py::test_profile_loads_v5_features_from_yaml \
                              tests/v5/test_v5_flags.py::test_profile_without_v5_features_yaml_section -v
```
Expected: FAIL with `AttributeError: 'GuidelineProfile' object has no attribute 'v5_features'`.

- [ ] **Step 3.4: Add `v5_features` field to `GuidelineProfile`**

Edit `backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/guideline_profile.py`:

After the existing `tiering_classifier` field, add:

```python
    # V5 feature flag overrides — see v5_flags.is_v5_enabled() and the master
    # spec at docs/superpowers/specs/2026-05-01-v5-master-architecture-design.md.
    # Each key is a lowercase V5 feature name (bbox_provenance, table_specialist,
    # consensus_entropy, schema_first, decomposition). Value semantics:
    #   True  -> force feature ON for this guideline (overrides env var)
    #   False -> force feature OFF for this guideline
    #   None  -> fall through to env var V5_<FEATURE>
    # Missing key behaves as None.
    v5_features: dict = field(default_factory=dict)
```

In the `from_yaml` classmethod, ensure `v5_features` is read from YAML if present, else default to `{}`:

```python
            v5_features=data.get("v5_features") or {},
```

- [ ] **Step 3.5: Run all tests to confirm pass + no regressions**

```bash
.venv13/bin/python -m pytest tests/v5/ -v
```
Expected: all v5_flags tests pass (now 11 tests).

- [ ] **Step 3.6: Add a commented-out `v5_features` example to the HF profile**

Edit `backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/data/profiles/heart_foundation_au_2025.yaml`. Append at the bottom:

```yaml

# V5 feature flags — uncomment to override per-guideline.
# Defaults to {} (all flags fall through to env vars V5_<FEATURE>).
# v5_features:
#   bbox_provenance: true
#   table_specialist: null  # null = fall through to env V5_TABLE_SPECIALIST
```

- [ ] **Step 3.7: Sanity-check the existing profile still loads**

```bash
.venv13/bin/python -c "
from extraction.v4.guideline_profile import GuidelineProfile
p = GuidelineProfile.from_yaml('data/profiles/heart_foundation_au_2025.yaml')
print(f'  profile_id: {p.profile_id}')
print(f'  v5_features: {p.v5_features}')
"
```
Expected: profile loads cleanly, `v5_features: {}`.

- [ ] **Step 3.8: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/guideline_profile.py \
        backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/data/profiles/heart_foundation_au_2025.yaml \
        backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/tests/v5/test_v5_flags.py
git -c commit.gpgsign=false commit -m "feat(v5): add v5_features dict field to GuidelineProfile"
```

---

## Task 4: `ChannelProvenance` Pydantic model + merge helper

**Files:**
- Create: `${EXTRACTION}/v4/provenance.py`
- Create: `${ATOMISER}/tests/v5/test_provenance_model.py`

- [ ] **Step 4.1: Write failing tests for the model + helper**

```python
# tests/v5/test_provenance_model.py
"""Unit tests for ChannelProvenance model + merge helpers."""
from __future__ import annotations

import pytest
from pydantic import ValidationError

from extraction.v4.provenance import (
    ChannelProvenance,
    merge_provenance_lists,
    serialise_provenance_list,
)


def _bbox(x0: float = 0, y0: float = 0, x1: float = 100, y1: float = 50) -> dict:
    return {"x0": x0, "y0": y0, "x1": x1, "y1": y1}


def test_construct_minimal() -> None:
    p = ChannelProvenance(
        channel_id="A",
        bbox=_bbox(),
        page_number=1,
        confidence=0.9,
        model_version="granite-docling@v1.0",
    )
    assert p.channel_id == "A"
    assert p.bbox.x0 == 0
    assert p.bbox.x1 == 100
    assert p.confidence == 0.9


def test_bbox_coords_must_be_ordered() -> None:
    with pytest.raises(ValidationError):
        ChannelProvenance(
            channel_id="A",
            bbox={"x0": 100, "y0": 0, "x1": 50, "y1": 50},  # x1 < x0
            page_number=1,
            confidence=0.9,
            model_version="v",
        )


def test_confidence_must_be_zero_to_one() -> None:
    with pytest.raises(ValidationError):
        ChannelProvenance(
            channel_id="A",
            bbox=_bbox(),
            page_number=1,
            confidence=1.5,
            model_version="v",
        )


def test_channel_id_must_be_one_of_known() -> None:
    """Channels are 0, A-H. Anything else is rejected."""
    valid = ["0", "A", "B", "C", "D", "E", "F", "G", "H"]
    for c in valid:
        ChannelProvenance(
            channel_id=c, bbox=_bbox(), page_number=1,
            confidence=0.9, model_version="v",
        )
    with pytest.raises(ValidationError):
        ChannelProvenance(
            channel_id="Z", bbox=_bbox(), page_number=1,
            confidence=0.9, model_version="v",
        )


def test_merge_concats_lists_dedups_by_channel_and_bbox() -> None:
    a1 = ChannelProvenance(
        channel_id="A", bbox=_bbox(0, 0, 100, 50),
        page_number=1, confidence=0.9, model_version="v",
    )
    a1_dup = ChannelProvenance(
        channel_id="A", bbox=_bbox(0, 0, 100, 50),
        page_number=1, confidence=0.95, model_version="v",
    )
    b1 = ChannelProvenance(
        channel_id="B", bbox=_bbox(10, 10, 80, 30),
        page_number=1, confidence=0.7, model_version="aho-corasick",
    )
    merged = merge_provenance_lists([[a1], [a1_dup, b1]])
    assert len(merged) == 2
    # Highest confidence wins on dup
    a_entry = next(p for p in merged if p.channel_id == "A")
    assert a_entry.confidence == 0.95


def test_serialise_to_jsonb_compatible_dict() -> None:
    p = ChannelProvenance(
        channel_id="A", bbox=_bbox(),
        page_number=1, confidence=0.9, model_version="v",
    )
    out = serialise_provenance_list([p])
    assert isinstance(out, list)
    assert out[0]["channel_id"] == "A"
    assert out[0]["bbox"]["x0"] == 0
    assert out[0]["confidence"] == 0.9


def test_empty_list_serialises_to_empty_list() -> None:
    assert serialise_provenance_list([]) == []
```

- [ ] **Step 4.2: Run tests to confirm they fail**

```bash
.venv13/bin/python -m pytest tests/v5/test_provenance_model.py -v
```
Expected: FAIL with `ModuleNotFoundError: No module named 'extraction.v4.provenance'`.

- [ ] **Step 4.3: Implement `provenance.py`**

Create `backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/provenance.py`:

```python
"""V5 channel provenance model.

A ChannelProvenance records WHO observed a span (which channel), WHERE on
the page (bbox + page_number), HOW confident they were, and WHICH model
version was used. Lists of these go into the merged_spans.json
`channel_provenance` field and the KB-0 `l2_merged_spans.provenance_v5`
jsonb column.

Used by:
  - extraction.v4.signal_merger to thread per-channel observations into
    the final merged span
  - tools/guideline-atomiser/data/push_to_kb0_gcp.py for serialisation to
    the canonical_facts DB
  - tests/v5/* acceptance tests for coverage assertions

Schema is backward-compatible: spans without channel_provenance default to
an empty list, so existing V4 jobs continue to validate.
"""
from __future__ import annotations

from typing import Annotated, Iterable

from pydantic import BaseModel, ConfigDict, Field, model_validator

# Channel IDs are stable across V4 and V5: 0 (normaliser), A (docling),
# B (drug dict), C (grammar), D (table), E (gliner), F (nuextract),
# G (sentence), H (recovery).
ChannelId = Annotated[str, Field(pattern=r"^[0A-H]$")]


class BoundingBox(BaseModel):
    """Page-coordinate bounding box. (x0, y0) = top-left, (x1, y1) = bottom-right.

    Coordinates are in PDF points (typographic), origin at top-left.
    """
    model_config = ConfigDict(extra="forbid")
    x0: float = Field(ge=0)
    y0: float = Field(ge=0)
    x1: float
    y1: float

    @model_validator(mode="after")
    def _check_ordered(self) -> "BoundingBox":
        if self.x1 < self.x0:
            raise ValueError(f"bbox x1 ({self.x1}) < x0 ({self.x0})")
        if self.y1 < self.y0:
            raise ValueError(f"bbox y1 ({self.y1}) < y0 ({self.y0})")
        return self


class ChannelProvenance(BaseModel):
    """Per-channel evidence for a merged span.

    Lists of ChannelProvenance form the audit trail enabling V5 subsystems
    #1, #3, #4, #5 to compare and aggregate channel signals.
    """
    model_config = ConfigDict(extra="forbid")

    channel_id: ChannelId
    bbox: BoundingBox
    page_number: int = Field(ge=1)
    confidence: float = Field(ge=0.0, le=1.0)
    model_version: str = Field(min_length=1, max_length=200)
    notes: str | None = None  # optional free-text per-observation note


def merge_provenance_lists(
    lists: Iterable[list[ChannelProvenance]],
) -> list[ChannelProvenance]:
    """Concat multiple per-channel provenance lists into one for a merged span.

    Dedup rule: when two entries share the same (channel_id, bbox, page_number),
    keep the one with the higher confidence. This matches the signal_merger
    semantics of "channels can re-fire on the same region; the strongest wins".
    """
    out: dict[tuple[str, int, float, float, float, float], ChannelProvenance] = {}
    for lst in lists:
        for p in lst:
            key = (p.channel_id, p.page_number,
                   p.bbox.x0, p.bbox.y0, p.bbox.x1, p.bbox.y1)
            existing = out.get(key)
            if existing is None or p.confidence > existing.confidence:
                out[key] = p
    return list(out.values())


def serialise_provenance_list(
    items: Iterable[ChannelProvenance],
) -> list[dict]:
    """Render a list to JSON-compatible dicts for jsonb storage."""
    return [item.model_dump() for item in items]
```

- [ ] **Step 4.4: Run tests to confirm pass**

```bash
.venv13/bin/python -m pytest tests/v5/test_provenance_model.py -v
```
Expected: 7 passed.

- [ ] **Step 4.5: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/provenance.py \
        backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/tests/v5/test_provenance_model.py
git -c commit.gpgsign=false commit -m "feat(v5): ChannelProvenance Pydantic model + merge helper"
```

---

## Task 5: Extend `MergedSpan` model with optional `channel_provenance`

**Files:**
- Modify: `${EXTRACTION}/v4/models.py`
- Modify: `${ATOMISER}/tests/v5/test_provenance_model.py` (extend)

- [ ] **Step 5.1: Find `MergedSpan` definition**

```bash
grep -n "class MergedSpan\b" backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/models.py
```
Expected: locate the class and its existing fields (id, job_id, text, start, end, contributing_channels, channel_confidences, merged_confidence, has_disagreement, ...).

- [ ] **Step 5.2: Write failing test for the new field**

Append to `tests/v5/test_provenance_model.py`:

```python
def test_merged_span_accepts_channel_provenance_field() -> None:
    """MergedSpan accepts an optional channel_provenance list."""
    from extraction.v4.models import MergedSpan
    p = ChannelProvenance(
        channel_id="A", bbox=_bbox(),
        page_number=1, confidence=0.9, model_version="v",
    )
    span_kwargs = {
        "id": "00000000-0000-0000-0000-000000000001",
        "job_id": "00000000-0000-0000-0000-000000000000",
        "text": "test",
        "start": 0,
        "end": 4,
        "contributing_channels": ["A"],
        "channel_confidences": {"A": 0.9},
        "merged_confidence": 0.9,
        "has_disagreement": False,
        "tier": "TIER_1",
        "tier_reason": "single channel",
        "channel_provenance": [p],
    }
    span = MergedSpan(**span_kwargs)
    assert len(span.channel_provenance) == 1
    assert span.channel_provenance[0].channel_id == "A"


def test_merged_span_defaults_empty_provenance_when_omitted() -> None:
    """V4 spans without channel_provenance still validate (backward-compat)."""
    from extraction.v4.models import MergedSpan
    span_kwargs = {
        "id": "00000000-0000-0000-0000-000000000001",
        "job_id": "00000000-0000-0000-0000-000000000000",
        "text": "test",
        "start": 0,
        "end": 4,
        "contributing_channels": ["A"],
        "channel_confidences": {"A": 0.9},
        "merged_confidence": 0.9,
        "has_disagreement": False,
        "tier": "TIER_1",
        "tier_reason": "single channel",
    }
    span = MergedSpan(**span_kwargs)
    assert span.channel_provenance == []
```

- [ ] **Step 5.3: Run tests to confirm fail**

```bash
.venv13/bin/python -m pytest tests/v5/test_provenance_model.py -v
```
Expected: 2 new tests FAIL with `ValidationError: extra fields not permitted`.

- [ ] **Step 5.4: Add the field to `MergedSpan`**

In `backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/models.py`, after the existing fields and before `model_config = ...`:

```python
    # V5 #2 Bbox Provenance — backward-compatible additive field.
    # Empty list when V5_BBOX_PROVENANCE is off; populated when on.
    # See extraction.v4.provenance.ChannelProvenance.
    channel_provenance: list["ChannelProvenance"] = Field(default_factory=list)
```

Add to the imports section:
```python
from .provenance import ChannelProvenance
```

If the file already does `from __future__ import annotations`, the forward reference in `list["ChannelProvenance"]` works without `model_rebuild()`. If it does not, add at the module bottom:

```python
MergedSpan.model_rebuild()
```

- [ ] **Step 5.5: Run tests**

```bash
.venv13/bin/python -m pytest tests/v5/ -v
```
Expected: all V5 tests pass.

- [ ] **Step 5.6: Run existing V4 tests to confirm no regression**

```bash
.venv13/bin/python -m pytest tests/ -v -k "not v5" --tb=short -x
```
Expected: existing tests pass; if any fail because they don't pass `channel_provenance`, that's because we added a default-factory'd field — they should still pass since the field defaults to `[]`.

- [ ] **Step 5.7: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/models.py \
        backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/tests/v5/test_provenance_model.py
git -c commit.gpgsign=false commit -m "feat(v5): add channel_provenance field to MergedSpan (backward-compat)"
```

---

## Task 6: Wire provenance through Channel A (docling)

**Files:**
- Modify: `${EXTRACTION}/v4/channel_a_docling.py`

This is the prototype — the same pattern repeats for B–H in Task 7.

- [ ] **Step 6.1: Locate the span-emit point in Channel A**

```bash
grep -n "def parse\|return tree\|GuidelineSection\|TableBoundary\|def _parse_with_granite_docling" \
  backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/channel_a_docling.py | head
```
Expected: Channel A returns a `GuidelineTree` — provenance per section/table is what we want to add. Channel A's spans are derived later, but its bboxes ride on `GuidelineSection.bbox` if present.

- [ ] **Step 6.2: Add a `make_provenance()` helper at the channel level**

Insert near the top of `channel_a_docling.py` (after imports):

```python
from .provenance import BoundingBox, ChannelProvenance
from .v5_flags import is_v5_enabled


def _channel_a_provenance(
    bbox: tuple[float, float, float, float] | None,
    page_number: int,
    confidence: float,
    model_version: str,
    profile,
    notes: str | None = None,
) -> ChannelProvenance | None:
    """Build a ChannelProvenance entry for Channel A (docling) iff V5 flag on
    AND a valid bbox is available. Returns None when flag off or bbox missing
    so callers can skip cleanly without behaviour change.
    """
    if not is_v5_enabled("bbox_provenance", profile):
        return None
    if bbox is None:
        return None
    x0, y0, x1, y1 = bbox
    return ChannelProvenance(
        channel_id="A",
        bbox=BoundingBox(x0=x0, y0=y0, x1=x1, y1=y1),
        page_number=page_number,
        confidence=confidence,
        model_version=model_version,
        notes=notes,
    )
```

- [ ] **Step 6.3: Plumb `profile` into the channel constructor**

If `Channel A`'s class signature does not yet accept `profile`, add an optional kwarg:

```python
class ChannelADocling:
    def __init__(self, ..., profile=None):  # kwargs: extend if needed
        ...
        self.profile = profile
```

And update `channel_a.parse(...)` callsites in `run_pipeline_targeted.py` to pass `profile`. (Most channels already accept `profile`; verify with `grep "profile=" channel_*.py`.)

- [ ] **Step 6.4: Emit provenance entries on each section**

Find the loop where Channel A produces `GuidelineSection` / `TableBoundary` items. Where each item is built, also build a `ChannelProvenance` (using `_channel_a_provenance`) and attach it as `section.provenance_entry` (or similar attr). The downstream signal_merger will collect these.

(The exact attachment point is implementation-defined; the goal is that by the time signal_merger sees the section, a `ChannelProvenance` for it is present. If the existing data model has no place for this, store a side dict `self._provenance_by_section_id`.)

- [ ] **Step 6.5: Smoke-run Channel A end-to-end with the flag on**

```bash
cd backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser
V5_BBOX_PROVENANCE=1 .venv13/bin/python -c "
import sys; sys.path.insert(0, '.'); sys.path.insert(0, '..'); sys.path.insert(0, '../..')
from extraction.v4.channel_a_docling import ChannelADocling
from extraction.v4.guideline_profile import GuidelineProfile
profile = GuidelineProfile.from_yaml('data/profiles/heart_foundation_au_2025.yaml')
ch = ChannelADocling(profile=profile)
# minimum viable invocation; prints provenance count
"
```
Expected: prints something; no AttributeError. Real assertion comes in Task 11 smoke test.

- [ ] **Step 6.6: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/channel_a_docling.py
git -c commit.gpgsign=false commit -m "feat(v5): emit ChannelProvenance from Channel A (docling) when flag on"
```

---

## Task 7: Wire provenance through Channels 0, B, C, D, E, F, G, H

**Files:**
- Modify: `${EXTRACTION}/v4/channel_0_normalizer.py`
- Modify: `${EXTRACTION}/v4/channel_b_drug_dict.py`
- Modify: `${EXTRACTION}/v4/channel_c_grammar.py`
- Modify: `${EXTRACTION}/v4/channel_d_table.py`
- Modify: `${EXTRACTION}/v4/channel_e_gliner.py`
- Modify: `${EXTRACTION}/v4/channel_f_nuextract.py`
- Modify: `${EXTRACTION}/v4/channel_g_sentence.py`
- Modify: `${EXTRACTION}/v4/channel_h_recovery.py`

Each channel follows the same pattern as Task 6, with `channel_id` substituted (`"0"`, `"B"`, `"C"`, ..., `"H"`) and channel-specific notes:

| Channel | model_version source | bbox source | confidence source |
|---------|---------------------|-------------|-------------------|
| 0 (normaliser) | `f"normaliser@{NORMALIZER_VERSION}"` | original block bbox from L1 | 1.0 (normaliser is deterministic) |
| B (drug dict) | `f"aho-corasick@{DRUG_DICT_VERSION}"` | parent block bbox | dictionary-match confidence |
| C (grammar) | `f"regex@{GRAMMAR_VERSION}"` | parent block bbox | pattern confidence |
| D (table) | `f"docling-otsl@{DOCLING_VERSION}"` or `"pipe-table"` | table cell bbox if available, else table bbox | cell-extraction confidence |
| E (gliner) | `f"gliner@{GLINER_MODEL_NAME}"` | entity-region bbox (parent block) | gliner score |
| F (nuextract) | `f"nuextract@{NUEXTRACT_MODEL}"` | parent passage bbox | proposition confidence |
| G (sentence) | `f"sentence@{SENTENCE_VERSION}"` | sentence bbox (from L1) | 0.7 (heuristic) |
| H (recovery) | `f"recovery@{RECOVERY_VERSION}"` | original raw-span bbox | re-uses originating channel's confidence |

- [ ] **Step 7.1: Implement `_channel_X_provenance()` helper in each of the 8 channel files**

For each channel `X` in `[0, B, C, D, E, F, G, H]`, paste a localised version of the helper:

```python
from .provenance import BoundingBox, ChannelProvenance
from .v5_flags import is_v5_enabled

_CHANNEL_X_VERSION = "<see table above>"  # substitute per channel


def _channel_X_provenance(
    bbox, page_number, confidence, profile, notes=None,
) -> ChannelProvenance | None:
    if not is_v5_enabled("bbox_provenance", profile):
        return None
    if bbox is None:
        return None
    x0, y0, x1, y1 = bbox
    return ChannelProvenance(
        channel_id="X",  # substitute with the actual channel letter
        bbox=BoundingBox(x0=x0, y0=y0, x1=x1, y1=y1),
        page_number=page_number,
        confidence=confidence,
        model_version=_CHANNEL_X_VERSION,
        notes=notes,
    )
```

- [ ] **Step 7.2: At each channel's span-emit point, attach provenance**

Each channel currently emits some kind of span/block/entity object. Where each is created, also call the helper and attach the result. If the data model already has `bbox` and `page_number` available, the call is one line:

```python
prov = _channel_X_provenance(bbox=block.bbox, page_number=block.page,
                              confidence=score, profile=self.profile)
if prov:
    span.provenance_entries.append(prov)  # or self._prov[span_id].append(prov)
```

- [ ] **Step 7.3: Run the existing test suite to confirm no regression**

```bash
.venv13/bin/python -m pytest tests/ -v -k "not v5" --tb=short -x
```
Expected: existing tests pass.

- [ ] **Step 7.4: Run the V5 tests**

```bash
.venv13/bin/python -m pytest tests/v5/ -v
```
Expected: existing 11+9 = 20 V5 tests pass.

- [ ] **Step 7.5: Commit (single commit for all 8 channels — they're a coherent batch)**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/channel_0_normalizer.py \
        backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/channel_b_drug_dict.py \
        backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/channel_c_grammar.py \
        backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/channel_d_table.py \
        backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/channel_e_gliner.py \
        backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/channel_f_nuextract.py \
        backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/channel_g_sentence.py \
        backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/channel_h_recovery.py
git -c commit.gpgsign=false commit -m "feat(v5): emit ChannelProvenance from Channels 0,B-H (flag-gated)"
```

---

## Task 8: Update `signal_merger` to preserve provenance lists

**Files:**
- Modify: `${EXTRACTION}/v4/signal_merger.py`
- Create: `${ATOMISER}/tests/v5/test_signal_merger_provenance.py`

- [ ] **Step 8.1: Find the merge function**

```bash
grep -n "def merge\|class SignalMerger\|merged_span\|MergedSpan(" \
  backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/signal_merger.py
```

- [ ] **Step 8.2: Write failing test for merged-span provenance preservation**

```python
# tests/v5/test_signal_merger_provenance.py
"""When V5 flag is on, signal_merger must preserve channel_provenance."""
from __future__ import annotations

from extraction.v4.provenance import BoundingBox, ChannelProvenance


def test_merger_concats_channel_provenance() -> None:
    from extraction.v4.signal_merger import SignalMerger

    pa = ChannelProvenance(
        channel_id="A",
        bbox=BoundingBox(x0=0, y0=0, x1=100, y1=50),
        page_number=1, confidence=0.9, model_version="v",
    )
    pb = ChannelProvenance(
        channel_id="B",
        bbox=BoundingBox(x0=10, y0=10, x1=80, y1=30),
        page_number=1, confidence=0.7, model_version="v",
    )

    # Build two raw spans on the same text region with provenance attached
    raw_a = {
        "channel": "A", "text": "metformin 500mg", "start": 0, "end": 16,
        "page": 1, "confidence": 0.9,
        "provenance": pa,
    }
    raw_b = {
        "channel": "B", "text": "metformin 500mg", "start": 0, "end": 16,
        "page": 1, "confidence": 0.7,
        "provenance": pb,
    }

    merger = SignalMerger()
    merged = merger.merge([raw_a, raw_b], v5_bbox_provenance=True)
    assert len(merged) == 1
    span = merged[0]
    assert {p.channel_id for p in span.channel_provenance} == {"A", "B"}
```

- [ ] **Step 8.3: Run test to confirm fail**

```bash
.venv13/bin/python -m pytest tests/v5/test_signal_merger_provenance.py -v
```
Expected: FAIL (AttributeError or signature error).

- [ ] **Step 8.4: Modify `signal_merger.py` to thread provenance**

Wherever spans are built into `MergedSpan`, before the constructor call:

```python
# Collect provenance entries from all contributing raw spans
provenance_lists = [
    [r["provenance"]] if r.get("provenance") is not None else []
    for r in contributing_raw_spans
]
from .provenance import merge_provenance_lists
channel_provenance = (
    merge_provenance_lists(provenance_lists) if v5_bbox_provenance else []
)

merged_span = MergedSpan(
    ..., channel_provenance=channel_provenance,
)
```

Add `v5_bbox_provenance: bool = False` kwarg to the merger's main entry point.

- [ ] **Step 8.5: Run tests to confirm pass**

```bash
.venv13/bin/python -m pytest tests/v5/ -v
```
Expected: all 21+ tests pass.

- [ ] **Step 8.6: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/signal_merger.py \
        backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/tests/v5/test_signal_merger_provenance.py
git -c commit.gpgsign=false commit -m "feat(v5): signal_merger threads channel_provenance into MergedSpan"
```

---

## Task 9: Wire flag resolution into `run_pipeline_targeted.py`

**Files:**
- Modify: `${ATOMISER}/data/run_pipeline_targeted.py`

- [ ] **Step 9.1: Find the channel-construction block**

```bash
grep -n "channel_a\|ChannelADocling\|signal_merger\|SignalMerger\|profile = " \
  backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/data/run_pipeline_targeted.py | head -20
```

- [ ] **Step 9.2: Resolve V5 flags once at startup, pass into channels + merger**

Insert near the top of `pipeline_1()` after `profile` is built:

```python
from extraction.v4.v5_flags import is_v5_enabled

# Resolve V5 flag set once per run; thread through to channels + merger.
v5 = {
    name: is_v5_enabled(name, profile)
    for name in (
        "bbox_provenance", "table_specialist", "consensus_entropy",
        "schema_first", "decomposition",
    )
}
print(f"📐 V5 flags resolved: {v5}")
```

Then replace channel constructor calls (those that don't already accept `profile`) to thread it in. For `signal_merger.merge(...)`, add `v5_bbox_provenance=v5["bbox_provenance"]`.

- [ ] **Step 9.3: Echo flags into `job_metadata.json`**

Find the place where `job_metadata.json` is written. Add:

```python
metadata["v5_features_enabled"] = sorted(k for k, on in v5.items() if on)
```

- [ ] **Step 9.4: Run a smoke extraction with flag off (regression check)**

```bash
cd backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser
unset V5_BBOX_PROVENANCE
.venv13/bin/python data/run_pipeline_targeted.py \
  --pipeline 1 --guideline heart_foundation_au_2025 \
  --source acs-hcp-summary --l1 monkeyocr --target-kb all \
  2>&1 | tail -5
```
Expected: `📐 V5 flags resolved: {... 'bbox_provenance': False ...}`. Pipeline produces same output as before.

- [ ] **Step 9.5: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/data/run_pipeline_targeted.py
git -c commit.gpgsign=false commit -m "feat(v5): resolve V5 flags at pipeline start; thread to channels + merger"
```

---

## Task 10: KB-0 schema migration — add `provenance_v5` jsonb column

**Files:**
- Create: `${KB0}/migrations/009_l2_provenance_v5.sql`

- [ ] **Step 10.1: Write the migration SQL**

Create `backend/shared-infrastructure/knowledge-base-services/kb-0-governance-platform/migrations/009_l2_provenance_v5.sql`:

```sql
-- 009_l2_provenance_v5.sql
--
-- Adds a nullable jsonb column to l2_merged_spans for V5 channel provenance.
-- See docs/superpowers/specs/2026-05-01-v5-master-architecture-design.md.
--
-- Backward-compatible:
--   - Old rows have provenance_v5 = NULL.
--   - V4 push paths leave it NULL; V5 push paths populate it.
--   - The Go server has no awareness of this column (no Go changes needed).
--
-- Apply:
--   psql "host=34.46.243.149 port=5433 dbname=canonical_facts user=kb_admin" \
--        -f migrations/009_l2_provenance_v5.sql

ALTER TABLE l2_merged_spans
  ADD COLUMN IF NOT EXISTS provenance_v5 jsonb;

-- GIN index — supports queries like
--   SELECT * FROM l2_merged_spans WHERE provenance_v5 @> '[{"channel_id":"D"}]'
-- which we'll need for the V5 #4 consensus-entropy work.
CREATE INDEX IF NOT EXISTS idx_l2_merged_spans_provenance_v5_gin
  ON l2_merged_spans USING gin (provenance_v5)
  WHERE provenance_v5 IS NOT NULL;

COMMENT ON COLUMN l2_merged_spans.provenance_v5 IS
  'V5 #2 Bbox Provenance: jsonb array of ChannelProvenance entries. NULL for pre-V5 rows.';
```

- [ ] **Step 10.2: Validate the SQL syntax locally**

If a local Postgres is available:
```bash
docker exec -i kb4-patient-safety-postgres psql -U kb4_safety_user -d kb4_patient_safety -c "
CREATE TABLE IF NOT EXISTS _v5_test (id int);
ALTER TABLE _v5_test ADD COLUMN IF NOT EXISTS provenance_v5 jsonb;
DROP TABLE _v5_test;
"
```
Expected: no syntax errors.

- [ ] **Step 10.3: Apply migration to GCP — GATED ON USER AUTHORISATION**

The implementer must request explicit user confirmation before running:

```bash
echo "About to apply migration 009 to GCP Cloud SQL — requires user 'go'"
# Wait for user authorisation
PGPASSWORD=kb_secure_password_2024 psql \
  "host=34.46.243.149 port=5433 dbname=canonical_facts user=kb_admin" \
  -f backend/shared-infrastructure/knowledge-base-services/kb-0-governance-platform/migrations/009_l2_provenance_v5.sql
```

Expected: `ALTER TABLE`, `CREATE INDEX`, `COMMENT` (3 commands). Verify column appears:
```bash
PGPASSWORD=kb_secure_password_2024 psql \
  "host=34.46.243.149 port=5433 dbname=canonical_facts user=kb_admin" \
  -c "\d l2_merged_spans" | grep provenance_v5
```
Expected: `provenance_v5  | jsonb  |  | NULL`.

- [ ] **Step 10.4: Commit migration file (without applying — apply is a separate authorised step)**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-0-governance-platform/migrations/009_l2_provenance_v5.sql
git -c commit.gpgsign=false commit -m "feat(kb0): migration 009 — l2_merged_spans.provenance_v5 jsonb (V5 #2)"
```

---

## Task 11: Update `push_to_kb0_gcp.py` to write `provenance_v5`

**Files:**
- Modify: `${ATOMISER}/data/push_to_kb0_gcp.py`
- Create: `${ATOMISER}/tests/v5/test_kb0_round_trip.py`

- [ ] **Step 11.1: Locate the INSERT into `l2_merged_spans`**

```bash
grep -n "INSERT INTO l2_merged_spans\|execute_batch" \
  backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/data/push_to_kb0_gcp.py
```

- [ ] **Step 11.2: Add `provenance_v5` to the column list and row tuple**

In the INSERT statement, append `provenance_v5` to the columns list (last column). In the per-row tuple builder, append:

```python
psycopg2.extras.Json(s.get("channel_provenance")) if s.get("channel_provenance") else None,
```

So the INSERT becomes:

```python
psycopg2.extras.execute_batch(cur, """
    INSERT INTO l2_merged_spans (
        id, job_id, text, start_offset, end_offset, contributing_channels,
        channel_confidences, merged_confidence, has_disagreement,
        disagreement_detail, page_number, section_id, table_id,
        review_status, reviewer_text, reviewed_by, reviewed_at,
        created_at, bbox, surrounding_context, tier, coverage_guard_alert,
        semantic_tokens, provenance_v5
    ) VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s)
""", span_rows, page_size=200)
```

- [ ] **Step 11.3: Write a round-trip acceptance test**

```python
# tests/v5/test_kb0_round_trip.py
"""V5 #2 acceptance: provenance_v5 round-trips through KB-0 byte-identical."""
from __future__ import annotations

import json
import os

import pytest

# Mark test to require explicit env to run (avoids accidental writes during
# unit-test suite runs).
RUN_GCP = os.environ.get("V5_TEST_GCP_ROUND_TRIP") == "1"


@pytest.mark.skipif(not RUN_GCP, reason="set V5_TEST_GCP_ROUND_TRIP=1 to run")
def test_provenance_round_trip(tmp_path) -> None:
    import psycopg2
    import psycopg2.extras
    from extraction.v4.provenance import (
        BoundingBox,
        ChannelProvenance,
        serialise_provenance_list,
    )

    p = ChannelProvenance(
        channel_id="A",
        bbox=BoundingBox(x0=0, y0=0, x1=100, y1=50),
        page_number=1, confidence=0.9, model_version="round-trip-test",
    )
    serialised = serialise_provenance_list([p])

    conn = psycopg2.connect(
        host="34.46.243.149", port=5433, dbname="canonical_facts",
        user="kb_admin", password="kb_secure_password_2024",
    )
    test_span_id = "00000000-0000-4000-8000-test_v5_rt00"
    test_job_id = "00000000-0000-4000-8000-test_v5_jobrt"
    try:
        cur = conn.cursor()
        # Cleanup any stale prior run
        cur.execute("DELETE FROM l2_merged_spans WHERE id = %s", (test_span_id,))
        cur.execute("DELETE FROM l2_extraction_jobs WHERE job_id = %s", (test_job_id,))
        # Insert minimal job + span row with our serialised provenance
        cur.execute("""
            INSERT INTO l2_extraction_jobs(job_id, source_pdf, pipeline_version,
                total_merged_spans, total_sections, total_pages,
                spans_confirmed, spans_rejected, spans_edited, spans_added,
                spans_pending, status, created_at, updated_at, pdf_page_offset,
                guideline_tier)
            VALUES (%s, 'rt-test.pdf', '4.2.2', 1, 1, 1, 0, 0, 0, 0, 0,
                'PENDING_REVIEW', now(), now(), 0, 1)
        """, (test_job_id,))
        cur.execute("""
            INSERT INTO l2_merged_spans(id, job_id, text, start_offset, end_offset,
                contributing_channels, merged_confidence, has_disagreement,
                review_status, created_at, provenance_v5)
            VALUES (%s, %s, 'rt-test', 0, 7, ARRAY['A'], 0.9, false,
                'PENDING', now(), %s)
        """, (test_span_id, test_job_id, psycopg2.extras.Json(serialised)))
        conn.commit()
        # Read back
        cur.execute(
            "SELECT provenance_v5 FROM l2_merged_spans WHERE id = %s",
            (test_span_id,),
        )
        (round_tripped,) = cur.fetchone()
        assert round_tripped == serialised
    finally:
        # Cleanup
        cur.execute("DELETE FROM l2_merged_spans WHERE id = %s", (test_span_id,))
        cur.execute("DELETE FROM l2_extraction_jobs WHERE job_id = %s", (test_job_id,))
        conn.commit()
        conn.close()
```

- [ ] **Step 11.4: Skip the GCP round-trip in default `pytest tests/v5/` runs**

The skip-if guard in step 11.3 handles this.

- [ ] **Step 11.5: Run unit test suite (without GCP integration)**

```bash
.venv13/bin/python -m pytest tests/v5/ -v
```
Expected: `test_provenance_round_trip` shows as SKIPPED. All other tests PASS.

- [ ] **Step 11.6: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/data/push_to_kb0_gcp.py \
        backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/tests/v5/test_kb0_round_trip.py
git -c commit.gpgsign=false commit -m "feat(v5): write provenance_v5 jsonb in KB-0 pusher + round-trip test"
```

---

## Task 12: `v5_metrics.py` sidecar generator

**Files:**
- Create: `${ATOMISER}/data/v5_metrics.py`
- Modify: `${ATOMISER}/data/run_pipeline_targeted.py` (call sidecar at end)

- [ ] **Step 12.1: Write failing test for the metrics generator**

Append to `tests/v5/test_provenance_model.py`:

```python
def test_v5_metrics_computes_bbox_coverage(tmp_path) -> None:
    """v5_metrics.compute_metrics() reports bbox_coverage_pct."""
    from data.v5_metrics import compute_metrics_from_spans

    spans_with = [{
        "id": "1", "text": "x", "channel_provenance": [
            {"channel_id": "A", "bbox": {"x0": 0, "y0": 0, "x1": 1, "y1": 1},
             "page_number": 1, "confidence": 0.9, "model_version": "v"}
        ],
        "tier": "TIER_1", "contributing_channels": ["A"],
    }]
    spans_without = [{
        "id": "2", "text": "y", "channel_provenance": [],
        "tier": "TIER_1", "contributing_channels": ["A"],
    }]

    m = compute_metrics_from_spans(spans_with + spans_without)
    assert m["primary"]["bbox_coverage_pct"]["v5"] == 50.0
    assert m["primary"]["channel_provenance_pct"]["v5"] == 50.0
```

- [ ] **Step 12.2: Run to confirm fail**

```bash
.venv13/bin/python -m pytest tests/v5/test_provenance_model.py::test_v5_metrics_computes_bbox_coverage -v
```
Expected: FAIL with `ModuleNotFoundError: No module named 'data.v5_metrics'`.

- [ ] **Step 12.3: Implement `v5_metrics.py`**

Create `backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/data/v5_metrics.py`:

```python
"""V5 metrics.json sidecar generator.

Computes universal regression metrics + per-subsystem primary metrics from
a job's merged_spans.json (and optionally a V4 baseline job's merged_spans.json).
Writes metrics.json into the same job dir.

Schema: see docs/superpowers/specs/2026-05-01-v5-master-architecture-design.md §7.

Universal regression:
  total_spans     within ±15% of V4 baseline
  tier_1_pct      ≥ V4 baseline (percentage points)
  kb0_push        100% (best-effort: assumes pusher exit=0)
  new_error_patterns   0 new ERROR-level log lines vs V4

Per-subsystem #2 (bbox_provenance) primary:
  bbox_coverage_pct                ≥99%
  channel_provenance_pct           ≥99%
  bbox_in_page_bounds_pct          100% (secondary)

Usage (CLI):
  python v5_metrics.py <job_dir> [--v4-baseline-job <other_job_dir>]
"""
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any


UNIVERSAL_TOTAL_SPANS_THRESHOLD_PCT = 15.0  # |delta| ≤ 15% of V4
PRIMARY_BBOX_COVERAGE_THRESHOLD = 99.0
PRIMARY_CHANNEL_PROV_THRESHOLD = 99.0
SECONDARY_BBOX_IN_PAGE_THRESHOLD = 100.0


def _is_bbox_present(span: dict) -> bool:
    bb = span.get("bbox")
    if bb is None:
        return False
    if isinstance(bb, dict):
        # Pydantic dump form
        return all(k in bb for k in ("x0", "y0", "x1", "y1"))
    return False


def _has_channel_provenance(span: dict) -> bool:
    cp = span.get("channel_provenance") or []
    return len(cp) >= 1


def compute_metrics_from_spans(spans: list[dict]) -> dict[str, Any]:
    """Compute V5 #2 metrics from a span list. Pure function for testing."""
    n = len(spans)
    if n == 0:
        return {
            "primary": {
                "bbox_coverage_pct": {"v5": 0.0, "threshold": PRIMARY_BBOX_COVERAGE_THRESHOLD, "status": "FAIL"},
                "channel_provenance_pct": {"v5": 0.0, "threshold": PRIMARY_CHANNEL_PROV_THRESHOLD, "status": "FAIL"},
            },
            "verdict": "FAIL: no spans",
        }
    bbox_count = sum(1 for s in spans if _is_bbox_present(s))
    cp_count = sum(1 for s in spans if _has_channel_provenance(s))
    bbox_pct = 100.0 * bbox_count / n
    cp_pct = 100.0 * cp_count / n
    return {
        "primary": {
            "bbox_coverage_pct": {
                "v5": round(bbox_pct, 2),
                "threshold": PRIMARY_BBOX_COVERAGE_THRESHOLD,
                "status": "PASS" if bbox_pct >= PRIMARY_BBOX_COVERAGE_THRESHOLD else "FAIL",
            },
            "channel_provenance_pct": {
                "v5": round(cp_pct, 2),
                "threshold": PRIMARY_CHANNEL_PROV_THRESHOLD,
                "status": "PASS" if cp_pct >= PRIMARY_CHANNEL_PROV_THRESHOLD else "FAIL",
            },
        },
        "verdict": (
            "PASS" if bbox_pct >= PRIMARY_BBOX_COVERAGE_THRESHOLD
                  and cp_pct >= PRIMARY_CHANNEL_PROV_THRESHOLD
            else "FAIL"
        ),
    }


def compute_metrics(
    job_dir: Path, v4_baseline_dir: Path | None = None
) -> dict[str, Any]:
    spans = json.loads((job_dir / "merged_spans.json").read_text())
    metrics = compute_metrics_from_spans(spans)
    metrics["job_id"] = (
        json.loads((job_dir / "job_metadata.json").read_text()).get("job_id")
    )
    metrics["job_dir"] = str(job_dir)

    if v4_baseline_dir is not None:
        v4_spans = json.loads((v4_baseline_dir / "merged_spans.json").read_text())
        v4_n = len(v4_spans)
        delta_pct = 100.0 * (len(spans) - v4_n) / v4_n if v4_n else 0.0
        metrics["regression"] = {
            "total_spans": {
                "v4": v4_n, "v5": len(spans), "delta_pct": round(delta_pct, 2),
                "threshold_pct": UNIVERSAL_TOTAL_SPANS_THRESHOLD_PCT,
                "status": "PASS" if abs(delta_pct) <= UNIVERSAL_TOTAL_SPANS_THRESHOLD_PCT else "FAIL",
            },
        }
    return metrics


def main(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("job_dir", type=Path)
    p.add_argument("--v4-baseline-job", type=Path, default=None)
    args = p.parse_args(argv)
    m = compute_metrics(args.job_dir, args.v4_baseline_job)
    out_path = args.job_dir / "metrics.json"
    out_path.write_text(json.dumps(m, indent=2, default=str))
    print(f"wrote {out_path}: verdict={m.get('verdict')}")
    return 0 if m.get("verdict", "").startswith("PASS") else 2


if __name__ == "__main__":
    sys.exit(main())
```

- [ ] **Step 12.4: Run tests**

```bash
.venv13/bin/python -m pytest tests/v5/test_provenance_model.py -v
```
Expected: all pass.

- [ ] **Step 12.5: Wire `v5_metrics.compute_metrics()` call into `run_pipeline_targeted.py`**

At the end of `pipeline_1()`, after job artefacts are saved:

```python
from data.v5_metrics import compute_metrics
import json
metrics = compute_metrics(Path(job_dir))
(Path(job_dir) / "metrics.json").write_text(json.dumps(metrics, indent=2, default=str))
print(f"📊 V5 metrics: {metrics.get('verdict')}")
```

- [ ] **Step 12.6: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/data/v5_metrics.py \
        backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/data/run_pipeline_targeted.py \
        backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/tests/v5/test_provenance_model.py
git -c commit.gpgsign=false commit -m "feat(v5): metrics.json sidecar generator (universal + #2 primary)"
```

---

## Task 13: Smoke acceptance test (≥99% bbox coverage)

**Files:**
- Create: `${ATOMISER}/tests/v5/test_smoke_bbox_coverage.py`

- [ ] **Step 13.1: Write the smoke acceptance test**

```python
# tests/v5/test_smoke_bbox_coverage.py
"""V5 #2 acceptance test: ≥99% bbox coverage on smoke set.

Requires the smoke PDFs to have been extracted with V5_BBOX_PROVENANCE=1.
Fixture v4_baseline_jobs locates them; we re-read merged_spans.json.

To run after a flagged extraction:
  V5_BBOX_PROVENANCE=1 python data/run_pipeline_targeted.py \\
    --pipeline 1 --guideline heart_foundation_au_2025 \\
    --source acs-hcp-summary --l1 monkeyocr --target-kb all
  pytest tests/v5/test_smoke_bbox_coverage.py -v
"""
from __future__ import annotations

import json
from pathlib import Path

import pytest

from data.v5_metrics import compute_metrics_from_spans

SMOKE_SOURCES = ["AU-HF-ACS-HCP-Summary-2025.pdf", "AU-HF-Cholesterol-Action-Plan-2026.pdf"]


def _find_v5_jobs(atomiser_dir: Path, source_pdf: str) -> list[Path]:
    """Find the most recent V5-flagged job dirs for a given source PDF."""
    matches = []
    for job_dir in (atomiser_dir / "data/output/v4").glob("job_monkeyocr_*"):
        meta_path = job_dir / "job_metadata.json"
        if not meta_path.exists():
            continue
        meta = json.loads(meta_path.read_text())
        if meta.get("source_pdf") != source_pdf:
            continue
        if "bbox_provenance" not in (meta.get("v5_features_enabled") or []):
            continue
        matches.append(job_dir)
    matches.sort(key=lambda p: p.stat().st_mtime, reverse=True)
    return matches


@pytest.mark.parametrize("source_pdf", SMOKE_SOURCES)
def test_smoke_bbox_coverage_at_least_99(atomiser_dir: Path, source_pdf: str) -> None:
    matches = _find_v5_jobs(atomiser_dir, source_pdf)
    if not matches:
        pytest.skip(f"no V5-flagged extraction yet for {source_pdf}")
    job_dir = matches[0]
    spans = json.loads((job_dir / "merged_spans.json").read_text())
    metrics = compute_metrics_from_spans(spans)
    bbox_pct = metrics["primary"]["bbox_coverage_pct"]["v5"]
    cp_pct = metrics["primary"]["channel_provenance_pct"]["v5"]
    assert bbox_pct >= 99.0, (
        f"{source_pdf}: bbox coverage {bbox_pct:.2f}% < 99% threshold "
        f"(see {job_dir})"
    )
    assert cp_pct >= 99.0, (
        f"{source_pdf}: channel_provenance coverage {cp_pct:.2f}% < 99%"
    )
```

- [ ] **Step 13.2: Run (will skip until extraction has been done)**

```bash
.venv13/bin/python -m pytest tests/v5/test_smoke_bbox_coverage.py -v
```
Expected: 2 SKIPPED with "no V5-flagged extraction yet". Non-failing.

- [ ] **Step 13.3: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/tests/v5/test_smoke_bbox_coverage.py
git -c commit.gpgsign=false commit -m "test(v5): smoke acceptance — ≥99% bbox + provenance coverage"
```

---

## Task 14: End-to-end smoke run + push to KB-0 + verification

**Files:** none modified (operational task)

- [ ] **Step 14.1: Run smoke extraction with `V5_BBOX_PROVENANCE=1`**

```bash
cd backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser
V5_BBOX_PROVENANCE=1 .venv13/bin/python data/run_pipeline_targeted.py \
  --pipeline 1 \
  --guideline heart_foundation_au_2025 \
  --source acs-hcp-summary \
  --l1 monkeyocr --target-kb all
```
Expected:
- `📐 V5 flags resolved: { ... 'bbox_provenance': True ... }`
- Pipeline completes successfully (~3 min on GPU, ~21 min on CPU)
- New job dir at `data/output/v4/job_monkeyocr_<uuid>/`
- `metrics.json` sidecar present in the job dir
- `merged_spans.json` has `channel_provenance` populated on each span

- [ ] **Step 14.2: Inspect metrics.json**

```bash
cat data/output/v4/job_monkeyocr_*/metrics.json | python3 -m json.tool | head -30
```
Expected: `verdict: "PASS"`, `bbox_coverage_pct ≥99`, `channel_provenance_pct ≥99`.

- [ ] **Step 14.3: Run the pytest acceptance test**

```bash
.venv13/bin/python -m pytest tests/v5/test_smoke_bbox_coverage.py -v
```
Expected: 1 PASS for ACS-HCP-Summary; cholesterol-action SKIPPED (not yet extracted with flag).

- [ ] **Step 14.4: (gated) Apply migration 009 to GCP — REQUIRES USER 'go'**

Stop and request user authorisation. Once given:

```bash
PGPASSWORD=kb_secure_password_2024 psql \
  "host=34.46.243.149 port=5433 dbname=canonical_facts user=kb_admin" \
  -f backend/shared-infrastructure/knowledge-base-services/kb-0-governance-platform/migrations/009_l2_provenance_v5.sql
```
Expected: `ALTER TABLE`, `CREATE INDEX`, `COMMENT`. No errors.

- [ ] **Step 14.5: Push the V5 job to KB-0**

```bash
.venv13/bin/python data/push_to_kb0_gcp.py data/output/v4/job_monkeyocr_<uuid>
```
Expected: `✅ pushed: jobs=1 spans=N passages=M tree=1`. The `provenance_v5` jsonb column is populated.

- [ ] **Step 14.6: Verify dashboard shows the new job + provenance survived**

```bash
curl -s 'https://kb0-governance-dashboard.vercel.app/api/v2/pipeline1/jobs' | \
  python3 -c "
import json,sys
au=[j for j in json.load(sys.stdin)['items'] if j['sourcePdf'].startswith('AU-HF')]
print(f'AU jobs: {len(au)}')
"
```
Plus a direct DB check:
```bash
PGPASSWORD=kb_secure_password_2024 psql \
  "host=34.46.243.149 port=5433 dbname=canonical_facts user=kb_admin" \
  -c "SELECT count(*) FROM l2_merged_spans WHERE provenance_v5 IS NOT NULL;"
```
Expected: count = the V5 job's span count (≥10).

- [ ] **Step 14.7: Repeat for cholesterol-action (second smoke PDF)**

```bash
V5_BBOX_PROVENANCE=1 .venv13/bin/python data/run_pipeline_targeted.py \
  --pipeline 1 \
  --guideline heart_foundation_au_2025 \
  --source cholesterol-action \
  --l1 monkeyocr --target-kb all
.venv13/bin/python data/push_to_kb0_gcp.py data/output/v4/job_monkeyocr_<latest_uuid>
.venv13/bin/python -m pytest tests/v5/test_smoke_bbox_coverage.py -v
```
Expected: both smoke PDFs PASS the ≥99% threshold.

- [ ] **Step 14.8: Run optional GCP round-trip test**

```bash
V5_TEST_GCP_ROUND_TRIP=1 .venv13/bin/python -m pytest tests/v5/test_kb0_round_trip.py -v
```
Expected: PASS.

- [ ] **Step 14.9: Commit a tracking note**

If any small fix-ups were needed during run, commit them. Otherwise no commit needed for this task.

---

## Task 15: Documentation — README updates for V5 flags

**Files:**
- Modify: `backend/shared-infrastructure/knowledge-base-services/README_AU.md`

- [ ] **Step 15.1: Add a "V5 features" section to README_AU.md**

Find the existing "Service endpoints" or "Loader scripts" section. Add a new section before "Verification":

```markdown
## V5 feature flags (Pipeline 1)

V5 incorporates 5 research-driven improvements as additive feature flags atop V4. See [v5 master architecture spec](../../docs/superpowers/specs/2026-05-01-v5-master-architecture-design.md) for the design.

### Flags

| Flag | Subsystem | Default | Status |
|------|-----------|--------:|--------|
| `V5_BBOX_PROVENANCE` | #2 Bbox Provenance | off | ✅ shipped 2026-05-XX |
| `V5_TABLE_SPECIALIST` | #1 Table Specialist (Nemotron) | off | ⏳ next |
| `V5_CONSENSUS_ENTROPY` | #4 Consensus Entropy | off | ⏳ |
| `V5_SCHEMA_FIRST` | #3 Schema-first | off | ⏳ |
| `V5_DECOMPOSITION` | #5 Decomposition | off | ⏳ |
| `V5_DISABLE_ALL` | (emergency rollback) | off | n/a |

### Toggling a flag

```bash
# Via env var on RunPod / locally
V5_BBOX_PROVENANCE=1 python data/run_pipeline_targeted.py --pipeline 1 ...

# Via profile YAML override (per-guideline)
# in data/profiles/<guideline>.yaml:
v5_features:
  bbox_provenance: true
```

### Verifying a flag landed

After a V5 run:
```bash
cat data/output/v4/job_monkeyocr_<uuid>/metrics.json | jq .verdict
```
should report `PASS`. The job's `merged_spans.json` will have `channel_provenance` populated on each span.

KB-0 GCP visibility: the `l2_merged_spans.provenance_v5` jsonb column is populated when `V5_BBOX_PROVENANCE=1`.
```

- [ ] **Step 15.2: Verify markdown renders**

```bash
head -50 backend/shared-infrastructure/knowledge-base-services/README_AU.md
```

- [ ] **Step 15.3: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/README_AU.md
git -c commit.gpgsign=false commit -m "docs(v5): README_AU.md V5 feature flags section + bbox_provenance status"
```

---

## Self-Review

**1. Spec coverage** (vs `2026-05-01-v5-master-architecture-design.md`):

| Spec § | Requirement | Plan task |
|--------|------------|-----------|
| §3 Architecture | Approach C (additive layers) | Threaded throughout — every channel keeps V4 default-off path |
| §4 Subsystem #2 | "every span carries non-null bbox + per-channel attribution" | Tasks 4, 5, 6, 7, 8 |
| §5 Feature-flag mechanism | env var + profile YAML override | Task 2 (resolver), Task 3 (profile field), Task 9 (resolution at startup) |
| §6 Smoke tier composition | ACS-HCP-Summary + Cholesterol-Action-Plan | Task 13, Task 14 |
| §7 Universal regression metrics | total spans ±15%, TIER_1 ≥, KB-0 push 100%, 0 new errors | Task 12 (universal section) |
| §7 #2 primary metric | ≥99% coverage with non-null bbox AND non-empty channel_provenance | Tasks 12, 13, 14 |
| §7 #2 secondary | per-channel bbox ≥95%, KB-0 round-trip byte-identical, bbox-in-bounds 100% | Task 4 (validator), Task 11 (round-trip) |
| §7 sidecar metrics.json | exact shape | Task 12 |
| §8 V4 deprecation | not in scope of #2; reserved for future | n/a |

No gaps.

**2. Placeholder scan**: re-read all task code blocks. No "TBD"/"TODO"/"implement later"/"fill in details"/"see file". Step 7.1 references "see table above" for `model_version` — that's a real cross-reference within the same task, not a placeholder. Step 14.4/14.5 use `<uuid>` as a wildcard for the runtime-generated job dir name — that is a placeholder for runtime values, not the plan author. Acceptable.

**3. Type consistency**:
- `ChannelProvenance` (Task 4) → used in `MergedSpan.channel_provenance` (Task 5) → returned by `_channel_X_provenance()` helpers (Tasks 6, 7) → consumed by `signal_merger.merge(...v5_bbox_provenance=...)` (Task 8) → serialised by `serialise_provenance_list()` (Task 4) → written to KB-0 by `push_to_kb0_gcp.py` (Task 11) → measured by `v5_metrics.compute_metrics_from_spans()` (Task 12) → asserted by smoke test (Task 13) → end-to-end verified (Task 14). Names + signatures match across.
- `is_v5_enabled(feature, profile)` signature consistent across Tasks 2, 9.
- `MergedSpan` field name `channel_provenance` consistent across Tasks 5, 8, 11, 12.

No issues found.

---

## Execution Handoff

**Plan complete and saved to `docs/superpowers/plans/2026-05-01-v5-bbox-provenance.md`. Two execution options:**

**1. Subagent-Driven (recommended)** — dispatch a fresh subagent per task, review between tasks, fast iteration. REQUIRED SUB-SKILL: `superpowers:subagent-driven-development`. Best for tasks with clear acceptance tests (most of this plan).

**2. Inline Execution** — execute tasks in this session using `superpowers:executing-plans`, batch execution with checkpoints for review. Better if you want to actively guide each step.

**Which approach?**
