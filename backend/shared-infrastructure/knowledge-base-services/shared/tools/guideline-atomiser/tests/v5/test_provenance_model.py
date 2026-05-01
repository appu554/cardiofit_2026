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


def test_bbox_y_coords_must_be_ordered() -> None:
    """y1 < y0 is rejected (companion to test_bbox_coords_must_be_ordered)."""
    with pytest.raises(ValidationError):
        ChannelProvenance(
            channel_id="A",
            bbox={"x0": 0, "y0": 100, "x1": 50, "y1": 50},  # y1 < y0
            page_number=1,
            confidence=0.9,
            model_version="v",
        )


def test_bbox_too_large_rejected() -> None:
    """Coords above the sanity ceiling (100_000 pt) are rejected."""
    with pytest.raises(ValidationError):
        ChannelProvenance(
            channel_id="A",
            bbox={"x0": 0, "y0": 0, "x1": 1e9, "y1": 50},
            page_number=1,
            confidence=0.9,
            model_version="v",
        )


def test_channel_id_accepts_recovery_and_reviewer() -> None:
    """L1_RECOVERY and REVIEWER are valid channel IDs (audit must record them)."""
    for c in ("L1_RECOVERY", "REVIEWER"):
        ChannelProvenance(
            channel_id=c, bbox=_bbox(), page_number=1,
            confidence=0.9, model_version="v",
        )


def test_empty_model_version_rejected() -> None:
    """model_version must be at least 1 char."""
    with pytest.raises(ValidationError):
        ChannelProvenance(
            channel_id="A", bbox=_bbox(), page_number=1,
            confidence=0.9, model_version="",
        )


def test_merge_dedups_under_ulp_bbox_drift() -> None:
    """Two entries with bboxes differing by ULP-level floats collapse to one."""
    a = ChannelProvenance(
        channel_id="A", bbox=_bbox(0.0, 0.0, 100.0, 50.0),
        page_number=1, confidence=0.9, model_version="v",
    )
    a_drift = ChannelProvenance(
        channel_id="A",
        # x1 differs by ~1 ULP — would survive raw float-equality dedup
        bbox=_bbox(0.0, 0.0, 100.0 + 1e-9, 50.0),
        page_number=1, confidence=0.95, model_version="v",
    )
    merged = merge_provenance_lists([[a], [a_drift]])
    assert len(merged) == 1
    assert merged[0].confidence == 0.95


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
        "contributing_channels": ["B"],
        "channel_confidences": {"B": 0.9},
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
        "contributing_channels": ["B"],
        "channel_confidences": {"B": 0.9},
        "merged_confidence": 0.9,
        "has_disagreement": False,
        "tier": "TIER_1",
        "tier_reason": "single channel",
    }
    span = MergedSpan(**span_kwargs)
    assert span.channel_provenance == []


# ---------------------------------------------------------------------------
# Channel A helper: _channel_a_provenance (Pipeline 1 V5 #2 Task 6)
# ---------------------------------------------------------------------------


def _make_profile_obj():
    """Minimal profile-like object with v5_features dict (matches v5_flags contract)."""
    from dataclasses import dataclass, field as dc_field

    @dataclass
    class _Profile:
        v5_features: dict = dc_field(default_factory=dict)

    return _Profile()


def test_channel_a_provenance_helper_off_when_flag_disabled(monkeypatch) -> None:
    """_channel_a_provenance returns None when V5_BBOX_PROVENANCE is off."""
    from extraction.v4.channel_a_docling import _channel_a_provenance

    monkeypatch.delenv("V5_BBOX_PROVENANCE", raising=False)
    profile = _make_profile_obj()  # default off
    assert _channel_a_provenance(
        bbox=(0, 0, 100, 50),
        page_number=1,
        confidence=0.9,
        profile=profile,
    ) is None


def test_channel_a_provenance_helper_on_with_flag(monkeypatch) -> None:
    """_channel_a_provenance returns a populated ChannelProvenance when flag on."""
    from extraction.v4.channel_a_docling import _channel_a_provenance

    monkeypatch.setenv("V5_BBOX_PROVENANCE", "1")
    profile = _make_profile_obj()
    p = _channel_a_provenance(
        bbox=(10, 20, 100, 50),
        page_number=2,
        confidence=0.85,
        profile=profile,
        notes="test",
    )
    assert p is not None
    assert p.channel_id == "A"
    assert p.page_number == 2
    assert p.confidence == 0.85
    assert p.bbox.x0 == 10
    assert p.bbox.y0 == 20
    assert p.bbox.x1 == 100
    assert p.bbox.y1 == 50
    assert p.notes == "test"
    assert p.model_version.startswith("granite-docling@")


def test_channel_a_provenance_helper_skip_when_bbox_none(monkeypatch) -> None:
    """If bbox is None, helper returns None even with flag on (no fake bboxes)."""
    from extraction.v4.channel_a_docling import _channel_a_provenance

    monkeypatch.setenv("V5_BBOX_PROVENANCE", "1")
    profile = _make_profile_obj()
    assert _channel_a_provenance(
        bbox=None,
        page_number=1,
        confidence=0.9,
        profile=profile,
    ) is None
