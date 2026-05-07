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


def test_default_on_when_nothing_set() -> None:
    """All V5 features are ON by default — no env var needed."""
    profile = _FakeProfile()
    assert is_v5_enabled("bbox_provenance", profile) is True


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
    """Older profiles without v5_features field fall through to env default (ON)."""
    profile = object()  # no v5_features attr at all
    assert is_v5_enabled("bbox_provenance", profile) is True


def test_unknown_feature_name_on() -> None:
    """Unknown feature names default to ON (env var not set → default '1')."""
    profile = _FakeProfile()
    assert is_v5_enabled("nonexistent_feature", profile) is True




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
