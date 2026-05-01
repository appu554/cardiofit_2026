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
