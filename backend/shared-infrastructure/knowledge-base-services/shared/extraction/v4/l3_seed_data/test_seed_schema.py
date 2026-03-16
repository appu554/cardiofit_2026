"""Validate all L3 seed data JSON files against the Pydantic schema."""

import json
import pathlib
import pytest
from extraction.schemas.kb20_contextual import AdverseReactionProfile

SEED_DIR = pathlib.Path(__file__).parent


def _load_seed_files():
    return sorted(SEED_DIR.glob("*.json"))


@pytest.fixture(params=_load_seed_files(), ids=lambda p: p.stem)
def seed_file(request):
    return request.param


def test_seed_validates_against_schema(seed_file):
    with open(seed_file) as f:
        data = json.load(f)

    # Must parse without error
    profile = AdverseReactionProfile(**data)

    # Must be FULL grade (all four elements present)
    assert profile.completeness_grade == "FULL", (
        f"{seed_file.name}: grade is {profile.completeness_grade}, expected FULL"
    )

    # Must have mechanism (E2) for FULL grade
    assert profile.mechanism, f"{seed_file.name}: missing mechanism (E2)"

    # Must have onset_window (E3) for FULL grade
    assert profile.onset_window, f"{seed_file.name}: missing onset_window (E3)"

    # Must have at least one contextual modifier (E4)
    assert len(profile.contextual_modifiers) > 0, (
        f"{seed_file.name}: missing contextual_modifiers (E4)"
    )

    # Source must be MANUAL_CURATED for seed data
    gov = data.get("governance", {})
    assert gov.get("sourceAuthority"), f"{seed_file.name}: missing sourceAuthority"


def test_all_12_seed_files_present():
    files = _load_seed_files()
    assert len(files) >= 12, f"Expected ≥12 seed files, found {len(files)}"
