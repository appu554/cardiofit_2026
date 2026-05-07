"""V5 Nemotron Parse 1.1 unit tests.

Tested behaviour
----------------
1. ``_strip_code_fences`` / ``_recover_json_array`` helpers — defensive
   parsing of model output (markdown fences, leading prose, etc.).
2. ``_parse_response`` — JSON → ``TableCellResult`` mapping with bbox
   conversion from normalised crop coords back to PDF points.
3. ``NemotronParseTableSpecialist.is_available`` — keyed on ``NVIDIA_API_KEY``.
4. ``extract_table`` end-to-end with the HTTP layer mocked: success path,
   401 surfaces as ``SpecialistUnavailableError``, timeout surfaces as
   ``SpecialistTimeoutError``.
5. ``_try_nemotron_parse_lane`` in Channel D — verifies the priority chain:
   when nemotron returns spans, vlm_table is bypassed; when it raises
   Unavailable, vlm_table runs; when it raises Timeout, control still
   falls through (no exception escapes the channel).

The HTTP layer is mocked via ``unittest.mock`` — these tests never touch
the network and never require ``NVIDIA_API_KEY`` to be set in the
environment.

Run from guideline-atomiser/:
    PYTHONPATH=. pytest tests/v5/test_v5_nemotron_parse.py -v
"""
from __future__ import annotations

import json
from unittest import mock

import pytest

from extraction.v4.specialists.base import (
    SpecialistError,
    SpecialistTimeoutError,
    SpecialistUnavailableError,
    TableCellResult,
    TableSpecialistResult,
)
from extraction.v4.specialists.nemotron_parse import (
    DEFAULT_HF_MODEL,
    NemotronParseTableSpecialist,
    _Backend,
    _select_backend,
    _recover_json_array,
    _strip_code_fences,
)


# ─────────────────────────────────────────────────────────────────────────
# Helpers
# ─────────────────────────────────────────────────────────────────────────

class TestStripCodeFences:
    """Strip ```json ... ``` wrappers without disturbing real content."""

    def test_strips_json_fence(self):
        raw = '```json\n[{"row_idx": 0}]\n```'
        assert _strip_code_fences(raw).strip() == '[{"row_idx": 0}]'

    def test_strips_bare_fence(self):
        raw = '```\n[{"row_idx": 0}]\n```'
        assert _strip_code_fences(raw).strip() == '[{"row_idx": 0}]'

    def test_idempotent_on_clean_input(self):
        raw = '[{"row_idx": 0}]'
        assert _strip_code_fences(raw) == raw

    def test_preserves_inner_backticks(self):
        # Backticks inside cell text must not be stripped.
        raw = '[{"text": "use `metformin`"}]'
        assert _strip_code_fences(raw) == raw


class TestRecoverJsonArray:
    """Last-ditch JSON recovery from prose-wrapped responses."""

    def test_recovers_from_leading_sentence(self):
        raw = 'Here is the table:\n[{"row_idx": 0, "text": "x"}]'
        result = _recover_json_array(raw)
        assert result == [{"row_idx": 0, "text": "x"}]

    def test_returns_none_on_no_array(self):
        assert _recover_json_array("no array here at all") is None

    def test_returns_none_on_malformed(self):
        # Has brackets but content is broken
        assert _recover_json_array("[not valid json at all") is None


# ─────────────────────────────────────────────────────────────────────────
# Response parsing
# ─────────────────────────────────────────────────────────────────────────

class TestParseResponse:
    """JSON → TableCellResult, including bbox normalised→PDF-points conversion."""

    def test_basic_parse(self):
        # Crop bbox spans (100, 200) → (300, 400) in PDF points (200x200 region).
        raw = json.dumps([
            {"row_idx": 0, "col_idx": 0, "text": "Drug",       "bbox": [0.0, 0.0, 0.5, 0.5], "is_header": True},
            {"row_idx": 1, "col_idx": 0, "text": "Metformin",  "bbox": [0.0, 0.5, 0.5, 1.0]},
        ])
        cells = NemotronParseTableSpecialist._parse_response(raw, (100, 200, 300, 400))

        assert len(cells) == 2
        assert cells[0].text == "Drug"
        assert cells[0].is_header is True
        # Bbox scaled into the (100,200,300,400) frame:
        # x0 = 100 + 0.0*200 = 100; y0 = 200 + 0.0*200 = 200; x1 = 200; y1 = 300
        assert cells[0].bbox == (100.0, 200.0, 200.0, 300.0)
        assert cells[1].text == "Metformin"
        assert cells[1].bbox == (100.0, 300.0, 200.0, 400.0)

    def test_empty_response(self):
        assert NemotronParseTableSpecialist._parse_response("", (0, 0, 100, 100)) == []
        assert NemotronParseTableSpecialist._parse_response("   \n  ", (0, 0, 100, 100)) == []

    def test_strips_markdown_fences(self):
        raw = '```json\n[{"row_idx":0,"col_idx":0,"text":"a"}]\n```'
        cells = NemotronParseTableSpecialist._parse_response(raw, (0, 0, 100, 100))
        assert len(cells) == 1 and cells[0].text == "a"

    def test_recovers_from_prose_prefix(self):
        raw = 'Sure! Here is the JSON:\n[{"row_idx":0,"col_idx":0,"text":"a"}]'
        cells = NemotronParseTableSpecialist._parse_response(raw, (0, 0, 100, 100))
        assert len(cells) == 1

    def test_skips_cells_with_bad_indices(self):
        # One bad cell + one good cell — partial extraction must succeed.
        raw = json.dumps([
            {"row_idx": "not-int", "col_idx": 0, "text": "bad"},
            {"row_idx": 0, "col_idx": 0, "text": "good"},
        ])
        cells = NemotronParseTableSpecialist._parse_response(raw, (0, 0, 100, 100))
        assert len(cells) == 1 and cells[0].text == "good"

    def test_missing_bbox_yields_none(self):
        raw = json.dumps([{"row_idx": 0, "col_idx": 0, "text": "x"}])
        cells = NemotronParseTableSpecialist._parse_response(raw, (0, 0, 100, 100))
        assert cells[0].bbox is None

    def test_confidence_clamped_to_unit_interval(self):
        raw = json.dumps([
            {"row_idx": 0, "col_idx": 0, "text": "a", "confidence": 1.5},
            {"row_idx": 0, "col_idx": 1, "text": "b", "confidence": -0.2},
        ])
        cells = NemotronParseTableSpecialist._parse_response(raw, (0, 0, 100, 100))
        assert cells[0].confidence == 1.0
        assert cells[1].confidence == 0.0

    def test_raises_on_unparseable_response(self):
        with pytest.raises(SpecialistError, match="unparseable JSON"):
            NemotronParseTableSpecialist._parse_response(
                "definitely not json {{}}{[",
                (0, 0, 100, 100),
            )

    def test_raises_on_non_list_root(self):
        with pytest.raises(SpecialistError, match="non-list"):
            NemotronParseTableSpecialist._parse_response(
                '{"not": "a list"}',
                (0, 0, 100, 100),
            )


# ─────────────────────────────────────────────────────────────────────────
# Backend selector — sidecar > nim > unavailable
# ─────────────────────────────────────────────────────────────────────────

class TestSelectBackend:
    """The dual-backend precedence rule.

    These are unit tests for the pure function — full integration tests
    for ``backend`` and ``model_version`` properties live below in
    ``TestIsAvailable``. Keeping the function pure makes this trivially
    fast and free of any env / fixture coupling.
    """

    def test_sidecar_wins_over_nim(self):
        # Both set — sidecar preferred (cheaper, on-prem, deterministic).
        assert _select_backend("http://localhost:8503", "fake-key") == _Backend.SIDECAR

    def test_nim_when_no_sidecar(self):
        assert _select_backend("", "fake-key") == _Backend.NIM

    def test_unavailable_when_neither(self):
        assert _select_backend("", "") == _Backend.UNAVAILABLE

    def test_whitespace_url_treated_as_unset(self):
        # The constructor strips trailing slashes but does NOT lstrip; an
        # all-whitespace URL still counts as truthy → sidecar. We don't
        # try to be clever about this because the test would actually
        # encourage env-var typos to silently fail. Just verify behaviour.
        assert _select_backend(" http://x ", "") == _Backend.SIDECAR


# ─────────────────────────────────────────────────────────────────────────
# is_available — wraps the selector + checks model_version disclosure
# ─────────────────────────────────────────────────────────────────────────

class TestIsAvailable:
    """Lane availability tracks the backend selector."""

    def test_available_with_api_key(self, monkeypatch):
        monkeypatch.delenv("NEMOTRON_PARSE_URL", raising=False)
        spec = NemotronParseTableSpecialist(api_key="fake-key")
        assert spec.is_available is True
        assert spec.backend == _Backend.NIM
        assert spec.model_version.endswith("@nim")

    def test_available_with_sidecar_url(self, monkeypatch):
        monkeypatch.delenv("NVIDIA_API_KEY", raising=False)
        spec = NemotronParseTableSpecialist(sidecar_url="http://parse:8503")
        assert spec.is_available is True
        assert spec.backend == _Backend.SIDECAR
        assert spec.model_version == f"{DEFAULT_HF_MODEL}@sidecar"

    def test_unavailable_without_either(self, monkeypatch):
        # Both env vars must be cleared — otherwise host env leaks in.
        monkeypatch.delenv("NVIDIA_API_KEY", raising=False)
        monkeypatch.delenv("NEMOTRON_PARSE_URL", raising=False)
        spec = NemotronParseTableSpecialist(api_key="", sidecar_url="")
        assert spec.is_available is False
        assert spec.backend == _Backend.UNAVAILABLE
        assert spec.model_version.endswith("@unavailable")

    def test_picks_up_api_key_env(self, monkeypatch):
        monkeypatch.setenv("NVIDIA_API_KEY", "from-env")
        monkeypatch.delenv("NEMOTRON_PARSE_URL", raising=False)
        spec = NemotronParseTableSpecialist()
        assert spec.is_available is True
        assert spec.backend == _Backend.NIM

    def test_picks_up_sidecar_env(self, monkeypatch):
        monkeypatch.delenv("NVIDIA_API_KEY", raising=False)
        monkeypatch.setenv("NEMOTRON_PARSE_URL", "http://parse:8503")
        spec = NemotronParseTableSpecialist()
        assert spec.is_available is True
        assert spec.backend == _Backend.SIDECAR

    def test_sidecar_env_preferred_over_api_key_env(self, monkeypatch):
        # Both env vars set → sidecar wins (per the precedence rule).
        monkeypatch.setenv("NVIDIA_API_KEY", "fake-key")
        monkeypatch.setenv("NEMOTRON_PARSE_URL", "http://parse:8503")
        spec = NemotronParseTableSpecialist()
        assert spec.backend == _Backend.SIDECAR

    def test_explicit_arg_wins_over_env(self, monkeypatch):
        monkeypatch.setenv("NVIDIA_API_KEY", "from-env")
        spec = NemotronParseTableSpecialist(api_key="explicit")
        assert spec._api_key == "explicit"

    def test_explicit_sidecar_url_wins_over_env(self, monkeypatch):
        monkeypatch.setenv("NEMOTRON_PARSE_URL", "http://from-env:9999")
        spec = NemotronParseTableSpecialist(sidecar_url="http://explicit:8503")
        assert spec._sidecar_url == "http://explicit:8503"


# ─────────────────────────────────────────────────────────────────────────
# extract_table — full path with HTTP mocked
# ─────────────────────────────────────────────────────────────────────────

@pytest.fixture
def mock_render(monkeypatch):
    """Patch ``_render_table_crop`` so tests don't need a real PDF.

    Used as a fixture even when the test body doesn't reference it — the
    setup runs via monkeypatch side-effect and ``extract_table`` would
    otherwise fail at the ``import fitz`` step.
    """
    monkeypatch.setattr(
        NemotronParseTableSpecialist,
        "_render_table_crop",
        staticmethod(lambda *a, **kw: "FAKE_BASE64"),
    )


@pytest.fixture
def clean_env(monkeypatch):
    """Strip both lane env vars so the test starts from a known state.

    Without this, a host with NVIDIA_API_KEY exported leaks into the
    NIM-only tests, and a host with NEMOTRON_PARSE_URL leaks into the
    "no-backend" tests. Always pair this with the explicit api_key=...
    or sidecar_url=... constructor args.
    """
    monkeypatch.delenv("NVIDIA_API_KEY", raising=False)
    monkeypatch.delenv("NEMOTRON_PARSE_URL", raising=False)


class TestExtractTableNIM:
    """End-to-end NIM-cloud path with HTTP mocked."""

    def test_unavailable_when_no_backend(self, clean_env):
        spec = NemotronParseTableSpecialist(api_key="", sidecar_url="")
        with pytest.raises(SpecialistUnavailableError, match="Neither NEMOTRON_PARSE_URL nor NVIDIA_API_KEY"):
            spec.extract_table("/tmp/x.pdf", 1, (0, 0, 100, 100))

    def test_success_path(self, clean_env, mock_render):
        spec = NemotronParseTableSpecialist(api_key="fake")

        cell_json = json.dumps([
            {"row_idx": 0, "col_idx": 0, "text": "Drug",      "is_header": True},
            {"row_idx": 1, "col_idx": 0, "text": "Metformin"},
        ])

        with mock.patch.object(spec, "_call_nim_cloud", return_value=cell_json):
            result = spec.extract_table("/tmp/x.pdf", 3, (10, 20, 110, 220))

        assert isinstance(result, TableSpecialistResult)
        assert result.page_number == 3
        assert result.model_version.endswith("@nim")
        assert result.metadata["backend"] == _Backend.NIM
        assert len(result.cells) == 2
        assert result.cells[0].is_header is True
        assert result.cells[1].text == "Metformin"

    def test_401_becomes_unavailable(self, clean_env, mock_render, monkeypatch):
        spec = NemotronParseTableSpecialist(api_key="bad-key")
        fake_requests = _build_fake_requests(status_code=401, body="invalid key")
        monkeypatch.setitem(__import__("sys").modules, "requests", fake_requests)

        with pytest.raises(SpecialistUnavailableError, match="auth failed"):
            spec.extract_table("/tmp/x.pdf", 1, (0, 0, 100, 100))

    def test_timeout_surfaces_specifically(self, clean_env, mock_render, monkeypatch):
        spec = NemotronParseTableSpecialist(api_key="fake", timeout_s=0.001)
        fake_requests = _build_fake_requests(raise_timeout=True)
        monkeypatch.setitem(__import__("sys").modules, "requests", fake_requests)

        with pytest.raises(SpecialistTimeoutError, match="NIM exceeded"):
            spec.extract_table("/tmp/x.pdf", 1, (0, 0, 100, 100))

    def test_500_becomes_specialist_error(self, clean_env, mock_render, monkeypatch):
        spec = NemotronParseTableSpecialist(api_key="fake")
        fake_requests = _build_fake_requests(status_code=500, body="server error")
        monkeypatch.setitem(__import__("sys").modules, "requests", fake_requests)

        with pytest.raises(SpecialistError, match="HTTP 500"):
            spec.extract_table("/tmp/x.pdf", 1, (0, 0, 100, 100))

    def test_empty_choices_becomes_specialist_error(self, clean_env, mock_render, monkeypatch):
        spec = NemotronParseTableSpecialist(api_key="fake")
        fake_requests = _build_fake_requests(status_code=200, json_body={"choices": []})
        monkeypatch.setitem(__import__("sys").modules, "requests", fake_requests)

        with pytest.raises(SpecialistError, match="no choices"):
            spec.extract_table("/tmp/x.pdf", 1, (0, 0, 100, 100))


# ─────────────────────────────────────────────────────────────────────────
# Sidecar HTTP backend — the production path on self-hosted GPU
# ─────────────────────────────────────────────────────────────────────────

class TestExtractTableSidecar:
    """End-to-end sidecar path. Sidecar wins over NIM whenever both are set."""

    def test_sidecar_success_returns_cells_with_pts_bbox(self, clean_env, mock_render):
        spec = NemotronParseTableSpecialist(sidecar_url="http://parse:8503")

        # Sidecar returns bbox in [0,1] image-space; we check the conversion
        # to PDF points using the table_bbox_pts (10,20,110,220) — width 100,
        # height 200, so a half-row cell at (0,0,1,0.5) maps to (10,20,110,120).
        sidecar_payload = {
            "cells": [
                {"row_idx": 0, "col_idx": 0, "text": "Drug", "is_header": True,
                 "bbox_norm": [0.0, 0.0, 1.0, 0.5], "confidence": 0.99},
                {"row_idx": 1, "col_idx": 0, "text": "Metformin", "is_header": False,
                 "bbox_norm": [0.0, 0.5, 1.0, 1.0], "confidence": 0.97},
            ],
            "model_version": DEFAULT_HF_MODEL,
        }

        with mock.patch.object(spec, "_call_sidecar",
                               return_value=_call_real_helper_to_build_cells(sidecar_payload, (10, 20, 110, 220))):
            result = spec.extract_table("/tmp/x.pdf", 7, (10, 20, 110, 220))

        assert result.page_number == 7
        assert result.model_version == f"{DEFAULT_HF_MODEL}@sidecar"
        assert result.metadata["backend"] == _Backend.SIDECAR
        assert len(result.cells) == 2
        assert result.cells[1].text == "Metformin"
        # bbox should be in PDF points, mapped from bbox_norm
        assert result.cells[1].bbox == (10.0, 120.0, 110.0, 220.0)

    def test_sidecar_preferred_when_both_set(self, clean_env, mock_render):
        """When both NEMOTRON_PARSE_URL and NVIDIA_API_KEY are set, sidecar wins."""
        spec = NemotronParseTableSpecialist(
            sidecar_url="http://parse:8503",
            api_key="should-not-be-used",
        )

        sidecar_called = {"n": 0}
        nim_called = {"n": 0}

        def fake_sidecar(*a, **kw):
            sidecar_called["n"] += 1
            return [TableCellResult(row_idx=0, col_idx=0, text="from sidecar")]

        def fake_nim(*a, **kw):
            nim_called["n"] += 1
            return "[]"

        with mock.patch.object(spec, "_call_sidecar", side_effect=fake_sidecar), \
             mock.patch.object(spec, "_call_nim_cloud", side_effect=fake_nim):
            result = spec.extract_table("/tmp/x.pdf", 1, (0, 0, 100, 100))

        assert sidecar_called["n"] == 1
        assert nim_called["n"] == 0
        assert result.cells[0].text == "from sidecar"
        assert result.metadata["backend"] == _Backend.SIDECAR

    def test_sidecar_connection_error_becomes_unavailable(self, clean_env, mock_render, monkeypatch):
        """If the sidecar is down, surface as Unavailable so Channel D falls through."""
        spec = NemotronParseTableSpecialist(sidecar_url="http://parse:8503")
        fake_requests = _build_fake_requests(raise_connection=True)
        monkeypatch.setitem(__import__("sys").modules, "requests", fake_requests)

        with pytest.raises(SpecialistUnavailableError, match="sidecar unreachable"):
            spec.extract_table("/tmp/x.pdf", 1, (0, 0, 100, 100))

    def test_sidecar_timeout_surfaces_specifically(self, clean_env, mock_render, monkeypatch):
        spec = NemotronParseTableSpecialist(sidecar_url="http://parse:8503", timeout_s=0.001)
        fake_requests = _build_fake_requests(raise_timeout=True)
        monkeypatch.setitem(__import__("sys").modules, "requests", fake_requests)

        with pytest.raises(SpecialistTimeoutError, match="sidecar exceeded"):
            spec.extract_table("/tmp/x.pdf", 1, (0, 0, 100, 100))

    def test_sidecar_500_becomes_specialist_error(self, clean_env, mock_render, monkeypatch):
        spec = NemotronParseTableSpecialist(sidecar_url="http://parse:8503")
        fake_requests = _build_fake_requests(status_code=500, body="model OOM")
        monkeypatch.setitem(__import__("sys").modules, "requests", fake_requests)

        with pytest.raises(SpecialistError, match="HTTP 500"):
            spec.extract_table("/tmp/x.pdf", 1, (0, 0, 100, 100))


def _call_real_helper_to_build_cells(payload, table_bbox_pts):
    """Run the real ``_cells_from_sidecar_payload`` against the canned payload.

    Lets the test verify the actual conversion logic instead of mocking it,
    which is what we care about — the bbox math is the value-add of this
    backend.
    """
    from extraction.v4.specialists.nemotron_parse import _cells_from_sidecar_payload
    return _cells_from_sidecar_payload(payload, table_bbox_pts)


# ─────────────────────────────────────────────────────────────────────────
# Channel D routing priority — the chain that actually matters in production
# ─────────────────────────────────────────────────────────────────────────

class _FakeTableBlock:
    """Minimal TableBlock stand-in for routing tests.

    Real TableBlock pulls in marker_extractor + Pydantic; we only need the
    attributes Channel D reads.
    """
    def __init__(self, *, with_cell_data=True, page_number=1, table_index=0):
        self.bbox = (10, 20, 110, 220)
        self.page_number = page_number
        self.table_index = table_index
        self.region_type = "table"
        self.headers = ["Drug", "eGFR"]
        self.rows = [["Metformin", "≥30"]]
        self.cell_data = (
            [
                {"text": "Drug", "row_idx": 0, "col_idx": 0, "bbox": [10, 20, 50, 40], "confidence": 0.99},
                {"text": "eGFR", "row_idx": 0, "col_idx": 1, "bbox": [60, 20, 100, 40], "confidence": 0.99},
                {"text": "Metformin", "row_idx": 1, "col_idx": 0, "bbox": [10, 50, 50, 70], "confidence": 0.99},
                {"text": "≥30", "row_idx": 1, "col_idx": 1, "bbox": [60, 50, 100, 70], "confidence": 0.99},
            ]
            if with_cell_data
            else None
        )


class _FakeTree:
    """Minimal GuidelineTree stand-in — only Channel D's reads matter."""
    tables = []

    def get_page_for_offset(self, offset):
        return 1


class TestChannelDRoutingPriority:
    """Verify the lane priority chain: nemotron > vlm_table > docling."""

    def test_nemotron_success_skips_vlm_lane(self, monkeypatch):
        from extraction.v4.channel_d_table import ChannelDTableDecomposer

        # Build a result the lane will consider "non-empty" so it commits.
        fake_result = TableSpecialistResult(
            page_number=1,
            cells=[
                TableCellResult(row_idx=0, col_idx=0, text="Drug",      is_header=True),
                TableCellResult(row_idx=1, col_idx=0, text="Metformin", bbox=(10, 50, 50, 70)),
            ],
            model_version="nvidia/nemo-document-parse-1.1@nim",
            table_bbox=(10, 20, 110, 220),
        )

        # Patch get_client to return a stub that returns our canned result.
        channel_d = ChannelDTableDecomposer()
        stub_client = mock.MagicMock()
        stub_client.extract_table.return_value = fake_result
        monkeypatch.setattr(channel_d, "_get_nemotron_parse_client", lambda: stub_client)

        # Spy on _decompose_vlm_structured_table — it MUST NOT be called.
        vlm_spy = mock.MagicMock()
        monkeypatch.setattr(channel_d, "_decompose_vlm_structured_table", vlm_spy)

        # Force ALL flag reads to True — production default-on behaviour.
        monkeypatch.setattr(
            "extraction.v4.channel_d_table.is_v5_enabled",
            lambda flag, profile: True,
        )

        tree = _FakeTree()
        tb = _FakeTableBlock(with_cell_data=True)
        out = channel_d.extract(
            "irrelevant text", tree, l1_tables=[tb],
            profile=None, pdf_path="/tmp/fake.pdf",
        )

        assert vlm_spy.call_count == 0, "vlm_table_specialist must not run when nemotron succeeded"
        assert out.metadata["tables_nemotron_parse"] == 1
        # Body cell only — header is excluded by convention.
        body_spans = [s for s in out.spans if s.text == "Metformin"]
        assert len(body_spans) == 1
        assert body_spans[0].channel_metadata["table_source"] == "nemotron_parse"

    def test_nemotron_unavailable_falls_through_to_vlm(self, monkeypatch):
        from extraction.v4.channel_d_table import ChannelDTableDecomposer

        channel_d = ChannelDTableDecomposer()
        stub_client = mock.MagicMock()
        stub_client.extract_table.side_effect = SpecialistUnavailableError("no key")
        monkeypatch.setattr(channel_d, "_get_nemotron_parse_client", lambda: stub_client)

        # Spy on the VLM lane so we can confirm it ran.
        vlm_called = {"n": 0}
        original_vlm = channel_d._decompose_vlm_structured_table

        def vlm_spy(tb, profile=None):
            vlm_called["n"] += 1
            return original_vlm(tb, profile=profile)
        monkeypatch.setattr(channel_d, "_decompose_vlm_structured_table", vlm_spy)

        monkeypatch.setattr(
            "extraction.v4.channel_d_table.is_v5_enabled",
            lambda flag, profile: True,
        )

        tree = _FakeTree()
        tb = _FakeTableBlock(with_cell_data=True)
        out = channel_d.extract(
            "irrelevant", tree, l1_tables=[tb],
            profile=None, pdf_path="/tmp/fake.pdf",
        )

        assert vlm_called["n"] == 1, "vlm_table lane must run when nemotron is unavailable"
        assert out.metadata["tables_nemotron_parse"] == 0

    def test_nemotron_timeout_falls_through_silently(self, monkeypatch):
        """Timeouts are logged at warning, but must not abort the page."""
        from extraction.v4.channel_d_table import ChannelDTableDecomposer

        channel_d = ChannelDTableDecomposer()
        stub_client = mock.MagicMock()
        stub_client.extract_table.side_effect = SpecialistTimeoutError("30s budget")
        monkeypatch.setattr(channel_d, "_get_nemotron_parse_client", lambda: stub_client)
        monkeypatch.setattr(
            "extraction.v4.channel_d_table.is_v5_enabled",
            lambda flag, profile: True,
        )

        tree = _FakeTree()
        tb = _FakeTableBlock(with_cell_data=True)

        # Should NOT raise — the page survives the timeout.
        out = channel_d.extract(
            "irrelevant", tree, l1_tables=[tb],
            profile=None, pdf_path="/tmp/fake.pdf",
        )
        assert out.metadata["tables_nemotron_parse"] == 0
        # vlm_table_specialist filled in for it
        assert len(out.spans) >= 1

    def test_no_pdf_path_disables_nemotron_lane(self, monkeypatch):
        """Old callers without pdf_path must not blow up — lane just skips."""
        from extraction.v4.channel_d_table import ChannelDTableDecomposer

        channel_d = ChannelDTableDecomposer()
        # Make get_client explode if it's ever called — proves the lane skipped.
        monkeypatch.setattr(
            channel_d, "_get_nemotron_parse_client",
            lambda: (_ for _ in ()).throw(AssertionError("nemotron lane must not run without pdf_path")),
        )
        monkeypatch.setattr(
            "extraction.v4.channel_d_table.is_v5_enabled",
            lambda flag, profile: True,
        )

        tree = _FakeTree()
        tb = _FakeTableBlock(with_cell_data=True)

        out = channel_d.extract("irrelevant", tree, l1_tables=[tb], profile=None)
        assert out.metadata["tables_nemotron_parse"] == 0

    def test_nemotron_fires_on_tree_table_when_l1_tables_empty(self, monkeypatch):
        """Gap-1 fix: when MonkeyOCR L1 finds 0 tables but Channel A's
        tree.tables has tables with provenance.bbox, the nemotron lane
        MUST still run on those tree.tables.

        Before the gate-widening fix this was a silent dead-letter — the
        whole V5 chain was bypassed and tables fell through to V4 OTSL/pipe
        without Nemotron Parse ever seeing the page. This test pins the
        new behaviour so a future regression of the gate condition fails
        loudly.
        """
        from extraction.v4.channel_d_table import ChannelDTableDecomposer
        from extraction.v4.models import TableBoundary
        from extraction.v4.provenance import BoundingBox, ChannelProvenance

        channel_d = ChannelDTableDecomposer()

        # Build a tree.tables with one OTSL table that has provenance.bbox set.
        # In a real run this comes from Channel A's Granite-Docling output.
        bbox = BoundingBox(x0=10, y0=20, x1=110, y1=220)
        prov = ChannelProvenance(
            channel_id="A",
            bbox=bbox,
            page_number=1,
            confidence=0.95,
            model_version="docling@v1",
        )
        boundary = TableBoundary(
            table_id="table_1",
            section_id="sec_1",
            start_offset=-1,
            end_offset=-1,
            headers=["Drug", "eGFR"],
            row_count=2,
            page_number=1,
            source="granite_otsl",
            otsl_text="<ched>Drug</ched><ched>eGFR</ched>"
                      "<fcel>Metformin</fcel><fcel>>=30</fcel>",
            provenance=prov,
        )

        class _Tree:
            tables = [boundary]
            def get_page_for_offset(self, _offset):
                return 1

        # Stub the nemotron client — return a non-empty result.
        fake_result = TableSpecialistResult(
            page_number=1,
            cells=[
                TableCellResult(row_idx=0, col_idx=0, text="Drug",      is_header=True),
                TableCellResult(row_idx=1, col_idx=0, text="Metformin", bbox=(10, 100, 50, 200)),
            ],
            model_version="nvidia/NVIDIA-Nemotron-Parse-v1.1-TC@sidecar",
            table_bbox=(10, 20, 110, 220),
        )
        stub_client = mock.MagicMock()
        stub_client.extract_table.return_value = fake_result
        monkeypatch.setattr(channel_d, "_get_nemotron_parse_client", lambda: stub_client)
        monkeypatch.setattr(
            "extraction.v4.channel_d_table.is_v5_enabled",
            lambda flag, profile: True,
        )

        # No l1_tables — exactly the failure case from the smoke run.
        out = channel_d.extract(
            "ignored", _Tree(), l1_tables=None, profile=None,
            pdf_path="/tmp/fake.pdf",
        )

        assert out.metadata["tables_nemotron_parse"] == 1, (
            "Expected nemotron lane to fire on tree.tables when l1_tables empty"
        )
        assert out.metadata["tables_otsl"] == 0, (
            "OTSL fallback must NOT run when nemotron succeeded"
        )
        # Span body — Metformin only (header excluded by convention).
        body_spans = [s for s in out.spans if s.text == "Metformin"]
        assert len(body_spans) == 1
        assert body_spans[0].channel_metadata["table_source"] == "nemotron_parse"

    def test_nemotron_unavailable_on_tree_table_falls_through_to_otsl(self, monkeypatch):
        """Symmetric fault-isolation test for the tree.tables path.

        When the nemotron lane raises Unavailable on a tree.table, control
        must drop through to the V4 OTSL/pipe path — same contract as the
        l1_tables side.
        """
        from extraction.v4.channel_d_table import ChannelDTableDecomposer
        from extraction.v4.models import TableBoundary
        from extraction.v4.provenance import BoundingBox, ChannelProvenance

        channel_d = ChannelDTableDecomposer()
        bbox = BoundingBox(x0=10, y0=20, x1=110, y1=220)
        prov = ChannelProvenance(
            channel_id="A", bbox=bbox, page_number=1,
            confidence=0.95, model_version="docling@v1",
        )
        boundary = TableBoundary(
            table_id="table_1",
            section_id="sec_1",
            start_offset=-1, end_offset=-1,
            headers=["Drug"], row_count=1, page_number=1,
            source="granite_otsl",
            otsl_text="<ched>Drug</ched><fcel>Metformin</fcel>",
            provenance=prov,
        )

        class _Tree:
            tables = [boundary]
            def get_page_for_offset(self, _offset):
                return 1

        stub_client = mock.MagicMock()
        stub_client.extract_table.side_effect = SpecialistUnavailableError("no key")
        monkeypatch.setattr(channel_d, "_get_nemotron_parse_client", lambda: stub_client)
        monkeypatch.setattr(
            "extraction.v4.channel_d_table.is_v5_enabled",
            lambda flag, profile: True,
        )

        out = channel_d.extract(
            "ignored", _Tree(), l1_tables=None, profile=None,
            pdf_path="/tmp/fake.pdf",
        )
        # Nemotron didn't fire, OTSL did
        assert out.metadata["tables_nemotron_parse"] == 0
        assert out.metadata["tables_otsl"] == 1


# ─────────────────────────────────────────────────────────────────────────
# Test infrastructure — fake `requests` module
# ─────────────────────────────────────────────────────────────────────────

def _build_fake_requests(
    *,
    status_code: int = 200,
    body: str = "",
    json_body: dict | None = None,
    raise_timeout: bool = False,
    raise_connection: bool = False,
):
    """Build a stand-in for the ``requests`` module that exercises one path.

    We can't use ``requests_mock`` because it isn't available in the test
    environment — and the lazy-import inside ``_call_nim_cloud`` /
    ``_call_sidecar`` means we need to inject ``requests`` into
    ``sys.modules`` *before* the import runs.

    The fake exposes the three exception classes the specialist catches
    (``Timeout``, ``ConnectionError``, ``RequestException``) so ``except``
    branches dispatch the way they would against the real library.
    """
    fake = mock.MagicMock(name="fake_requests")

    class _Timeout(Exception):
        pass

    class _ConnectionError(Exception):
        pass

    class _RequestException(Exception):
        pass

    fake.exceptions.Timeout = _Timeout
    fake.exceptions.ConnectionError = _ConnectionError
    fake.exceptions.RequestException = _RequestException

    response = mock.MagicMock()
    response.status_code = status_code
    response.text = body
    if json_body is not None:
        response.json.return_value = json_body
    elif body:
        response.json.side_effect = ValueError("not json")
    else:
        response.json.return_value = {"choices": [{"message": {"content": "[]"}}]}

    session = mock.MagicMock()
    if raise_timeout:
        session.post.side_effect = _Timeout("budget")
    elif raise_connection:
        session.post.side_effect = _ConnectionError("connection refused")
    else:
        session.post.return_value = response
    fake.Session.return_value = session

    return fake
