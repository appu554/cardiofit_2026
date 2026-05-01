"""Task 15: Validate README_AU.md documents the V5 feature flag system.

Pure documentation assertions — no subprocess, no network.
"""
from pathlib import Path


def _find_readme_au() -> Path:
    """Locate README_AU.md by walking up from this test file to the repo root.

    Avoids hardcoding so that worktree relocation does not break the test.
    """
    here = Path(__file__).resolve()
    for parent in here.parents:
        candidate = parent / "README_AU.md"
        if candidate.exists():
            return candidate
        # Common location under knowledge-base-services
        kb_candidate = parent / "backend" / "shared-infrastructure" / "knowledge-base-services" / "README_AU.md"
        if kb_candidate.exists():
            return kb_candidate
    raise FileNotFoundError("README_AU.md not found above this test file")


README_AU_PATH = _find_readme_au()


def test_readme_au_exists():
    assert README_AU_PATH.exists(), f"Expected README_AU.md at {README_AU_PATH}"


def test_readme_au_has_v5_flags_section():
    text = README_AU_PATH.read_text(encoding="utf-8")
    assert "V5 Feature Flags" in text, "README_AU.md missing 'V5 Feature Flags' section"
    assert "V5_BBOX_PROVENANCE" in text, "README_AU.md missing 'V5_BBOX_PROVENANCE' env var reference"


def test_readme_au_has_disable_all_flag():
    text = README_AU_PATH.read_text(encoding="utf-8")
    assert "V5_DISABLE_ALL" in text, "README_AU.md missing 'V5_DISABLE_ALL' kill-switch reference"
