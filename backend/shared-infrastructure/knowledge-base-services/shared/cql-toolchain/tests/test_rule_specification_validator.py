"""Wave 1 Task 2 — pytest coverage for rule_specification_validator.

Positive: the three Wave 0 example specs (ppi-deprescribe,
hyperkalemia-trajectory, antipsychotic-consent-gating) all validate.

Negative: one fixture per error class —
  - SCHEMA: missing required field
  - DANGLING_FACT: state_machine_references uses unknown fact_type
  - MISSING_TEST_COVERAGE: < 3 test cases
  - MISSING_TEST_COVERAGE: substrate-aware rule lacks substrate_state class
  - MISSING_TEST_COVERAGE: missing positive class
  - MISSING_S8_AUTHORISATION: BEERS S8 cease rule missing required_roles
"""

from __future__ import annotations

import copy
from pathlib import Path

import yaml
import pytest

from rule_specification_validator import (
    validate_rule_specification,
    validate_file,
)

EXAMPLES_DIR = (
    Path(__file__).resolve().parents[2]
    / "cql-libraries"
    / "examples"
)


@pytest.fixture
def ppi_spec() -> dict:
    return yaml.safe_load((EXAMPLES_DIR / "ppi-deprescribe.yaml").read_text())


@pytest.fixture
def hyperk_spec() -> dict:
    return yaml.safe_load((EXAMPLES_DIR / "hyperkalemia-trajectory.yaml").read_text())


@pytest.fixture
def antipsy_spec() -> dict:
    return yaml.safe_load(
        (EXAMPLES_DIR / "antipsychotic-consent-gating.yaml").read_text()
    )


# ---------------------------------------------------------------------------
# Positive — three anchor rules
# ---------------------------------------------------------------------------


def test_ppi_example_passes():
    result = validate_file(EXAMPLES_DIR / "ppi-deprescribe.yaml")
    assert result.ok, [str(e) for e in result.errors]


def test_hyperkalemia_example_passes():
    result = validate_file(EXAMPLES_DIR / "hyperkalemia-trajectory.yaml")
    assert result.ok, [str(e) for e in result.errors]


def test_antipsychotic_example_passes():
    result = validate_file(EXAMPLES_DIR / "antipsychotic-consent-gating.yaml")
    assert result.ok, [str(e) for e in result.errors]


# ---------------------------------------------------------------------------
# Negative — one per error class
# ---------------------------------------------------------------------------


def test_schema_missing_required_field_fails(ppi_spec):
    bad = copy.deepcopy(ppi_spec)
    del bad["rule_id"]
    result = validate_rule_specification(bad)
    assert not result.ok
    assert any(e.code == "SCHEMA" for e in result.errors)


def test_dangling_fact_reference_fails(ppi_spec):
    bad = copy.deepcopy(ppi_spec)
    bad["state_machine_references"]["reads_from"][0]["fact_type"] = "totally_made_up_fact"
    result = validate_rule_specification(bad)
    assert not result.ok
    assert any(e.code == "DANGLING_FACT" for e in result.errors)


def test_too_few_test_cases_fails(ppi_spec):
    bad = copy.deepcopy(ppi_spec)
    bad["test_cases"] = bad["test_cases"][:1]
    result = validate_rule_specification(bad)
    assert not result.ok
    assert any(
        e.code == "MISSING_TEST_COVERAGE" and "minimum 3 required" in e.message
        for e in result.errors
    )


def test_missing_positive_test_class_fails(ppi_spec):
    bad = copy.deepcopy(ppi_spec)
    # Replace all positive classes with negative-only ones
    bad["test_cases"] = [
        {
            "name": "n1",
            "class": "suppression",
            "fixture": "f1",
            "expected_fire": False,
        },
        {
            "name": "n2",
            "class": "boundary",
            "fixture": "f2",
            "expected_fire": False,
        },
        {
            "name": "n3",
            "class": "missing_data",
            "fixture": "f3",
            "expected_fire": False,
        },
    ]
    result = validate_rule_specification(bad)
    assert not result.ok
    assert any(
        "no test case in a positive class" in e.message for e in result.errors
    )


def test_substrate_aware_missing_substrate_state_class_fails(hyperk_spec):
    bad = copy.deepcopy(hyperk_spec)
    # Strip baseline_aware_fire and evidence_trace classes so substrate-
    # aware coverage is gone; keep positive+negative coverage.
    bad["test_cases"] = [
        {
            "name": "p",
            "class": "positive",
            "fixture": "fx",
            "expected_fire": True,
        },
        {
            "name": "n",
            "class": "suppression",
            "fixture": "fx",
            "expected_fire": False,
        },
        {
            "name": "n2",
            "class": "boundary",
            "fixture": "fx",
            "expected_fire": False,
        },
    ]
    result = validate_rule_specification(bad)
    assert not result.ok
    assert any(
        "substrate-aware rule must include" in e.message for e in result.errors
    )


def test_schedule8_taper_without_authorisation_fails(ppi_spec):
    bad = copy.deepcopy(ppi_spec)
    bad["rule_id"] = "OPIOID_TAPER_S8_EXAMPLE"
    bad["criterion_set"] = "AU_TGA_BLACKBOX"
    bad["criterion_id"] = "S8-OPIOID-001"
    bad["summary"] = "Taper opioid in chronic non-cancer pain after 90 days"
    # No authorisation_gating set
    bad.pop("authorisation_gating", None)
    result = validate_rule_specification(bad)
    assert not result.ok
    assert any(e.code == "MISSING_S8_AUTHORISATION" for e in result.errors)
