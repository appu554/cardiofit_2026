"""V5 #2 acceptance: signal_merger threads channel_provenance into MergedSpan.

Covers Task 8 (Pipeline 1 V5 #2 Bbox Provenance):

  1. The dispatch registry get_channel_provenance_builder returns a callable
     for every known channel (0, A-H, L1_RECOVERY) and None for unknowns
     (REVIEWER, "", arbitrary garbage).
  2. Channel D's table_source kwarg is correctly passed through the dispatch.
  3. Channel F's NUEXTRACT_MODEL env var is pinnable via monkeypatch.
  4. End-to-end: signal_merger.merge(..., v5_bbox_provenance=True) produces
     MergedSpan.channel_provenance populated with one entry per contributing
     RawSpan that has a builder + bbox.
  5. Flag-off path: signal_merger.merge(...) without v5_bbox_provenance
     produces MergedSpan.channel_provenance == [] (V4 byte-identical).
"""
from __future__ import annotations

from uuid import uuid4

import pytest

from extraction.v4.models import (
    ChannelOutput,
    GuidelineSection,
    GuidelineTree,
    RawSpan,
)
from extraction.v4.signal_merger import SignalMerger


@pytest.fixture(autouse=True)
def _set_v5_flag(monkeypatch: pytest.MonkeyPatch):
    """Force V5_BBOX_PROVENANCE=1 for these tests (helpers no-op without it)."""
    monkeypatch.setenv("V5_BBOX_PROVENANCE", "1")


def _fake_profile():
    """A minimal duck-typed profile compatible with is_v5_enabled."""
    class P:
        v5_features: dict = {}
    return P()


def _simple_tree() -> GuidelineTree:
    return GuidelineTree(
        sections=[
            GuidelineSection(
                section_id="1",
                heading="Test",
                start_offset=0,
                end_offset=100_000,
                page_number=1,
                block_type="paragraph",
                children=[],
            )
        ],
        tables=[],
        total_pages=1,
    )


# ── Step 8.1: dispatch registry ──────────────────────────────────────────


def test_dispatch_returns_builder_for_each_known_channel() -> None:
    """get_channel_provenance_builder returns a callable for every known channel."""
    from extraction.v4.provenance import get_channel_provenance_builder

    for c in ["0", "A", "B", "C", "D", "E", "F", "G", "H", "L1_RECOVERY"]:
        builder = get_channel_provenance_builder(c)
        assert builder is not None, f"no builder for channel {c!r}"
        assert callable(builder)


def test_dispatch_returns_none_for_unknown_channels() -> None:
    """REVIEWER and arbitrary unknowns return None (signal_merger handles separately)."""
    from extraction.v4.provenance import get_channel_provenance_builder

    assert get_channel_provenance_builder("REVIEWER") is None
    assert get_channel_provenance_builder("Z") is None
    assert get_channel_provenance_builder("") is None


def test_dispatch_d_passes_table_source_kwarg() -> None:
    """Channel D's helper accepts table_source; both branches yield distinct model_versions."""
    from extraction.v4.provenance import get_channel_provenance_builder

    builder = get_channel_provenance_builder("D")
    p_otsl = builder(
        bbox=(0, 0, 100, 50),
        page_number=1,
        confidence=0.9,
        profile=_fake_profile(),
        table_source="granite_otsl",
    )
    p_pipe = builder(
        bbox=(0, 0, 100, 50),
        page_number=1,
        confidence=0.9,
        profile=_fake_profile(),
        table_source="marker_pipe",
    )
    assert p_otsl is not None and "docling-otsl" in p_otsl.model_version
    assert p_pipe is not None and "pipe-table" in p_pipe.model_version


def test_channel_f_env_pinning(monkeypatch: pytest.MonkeyPatch) -> None:
    """Channel F reads NUEXTRACT_MODEL env var per call; verify env-driven model_version."""
    monkeypatch.setenv("NUEXTRACT_MODEL", "test-pinned-model-v9.9")
    from extraction.v4.channel_f_nuextract import _channel_f_provenance

    p = _channel_f_provenance(
        bbox=(0, 0, 100, 50),
        page_number=1,
        confidence=0.85,
        profile=_fake_profile(),
    )
    assert p is not None
    assert "test-pinned-model-v9.9" in p.model_version


# ── Step 8.2/8.3: end-to-end via SignalMerger.merge ──────────────────────


def test_merge_flag_off_yields_empty_provenance() -> None:
    """Flag-off (default): channel_provenance is empty list (V4 byte-identical)."""
    merger = SignalMerger()
    job_id = uuid4()
    outputs = [
        ChannelOutput(
            channel="B",
            spans=[
                RawSpan(
                    channel="B",
                    text="metformin",
                    start=10,
                    end=19,
                    confidence=0.9,
                    page_number=1,
                    bbox=[0.0, 0.0, 100.0, 50.0],
                ),
            ],
        ),
    ]
    result = merger.merge(job_id, outputs, _simple_tree())
    assert len(result) == 1
    assert result[0].channel_provenance == []


def test_merge_flag_on_populates_provenance_per_channel() -> None:
    """Flag-on: each contributing RawSpan with a builder + bbox produces one entry."""
    merger = SignalMerger()
    job_id = uuid4()
    outputs = [
        ChannelOutput(
            channel="B",
            spans=[
                RawSpan(
                    channel="B",
                    text="metformin",
                    start=10,
                    end=19,
                    confidence=0.9,
                    page_number=1,
                    bbox=[0.0, 0.0, 100.0, 50.0],
                ),
            ],
        ),
        ChannelOutput(
            channel="C",
            spans=[
                RawSpan(
                    channel="C",
                    text="metformin",
                    start=10,
                    end=19,
                    confidence=0.85,
                    page_number=1,
                    # Distinct bbox so dedup keeps both (key = channel+page+bbox)
                    bbox=[10.0, 10.0, 110.0, 60.0],
                ),
            ],
        ),
    ]
    result = merger.merge(
        job_id,
        outputs,
        _simple_tree(),
        v5_bbox_provenance=True,
        profile=_fake_profile(),
    )
    assert len(result) == 1
    cp = result[0].channel_provenance
    channel_ids = sorted(p.channel_id for p in cp)
    assert channel_ids == ["B", "C"]


def test_merge_flag_on_skips_raw_spans_without_bbox() -> None:
    """Raw spans without bbox produce no provenance entry (helper returns None)."""
    merger = SignalMerger()
    job_id = uuid4()
    outputs = [
        ChannelOutput(
            channel="B",
            spans=[
                RawSpan(
                    channel="B",
                    text="metformin",
                    start=10,
                    end=19,
                    confidence=0.9,
                    page_number=1,
                    bbox=None,  # no bbox -> no provenance
                ),
            ],
        ),
    ]
    result = merger.merge(
        job_id,
        outputs,
        _simple_tree(),
        v5_bbox_provenance=True,
        profile=_fake_profile(),
    )
    assert len(result) == 1
    assert result[0].channel_provenance == []


def test_merge_flag_on_threads_table_source_for_channel_d() -> None:
    """Channel D's table_source from channel_metadata reaches the provenance builder."""
    merger = SignalMerger()
    job_id = uuid4()
    outputs = [
        ChannelOutput(
            channel="D",
            spans=[
                RawSpan(
                    channel="D",
                    text="dose",
                    start=-1,
                    end=-1,
                    confidence=0.95,
                    page_number=2,
                    bbox=[20.0, 20.0, 120.0, 70.0],
                    channel_metadata={"table_source": "granite_otsl"},
                ),
            ],
        ),
    ]
    result = merger.merge(
        job_id,
        outputs,
        _simple_tree(),
        v5_bbox_provenance=True,
        profile=_fake_profile(),
    )
    assert len(result) == 1
    cp = result[0].channel_provenance
    assert len(cp) == 1
    assert cp[0].channel_id == "D"
    assert "docling-otsl" in cp[0].model_version
