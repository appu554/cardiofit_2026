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
