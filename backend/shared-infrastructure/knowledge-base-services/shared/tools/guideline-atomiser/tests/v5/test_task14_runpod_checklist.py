"""Task 14: Validate the RunPod V5 smoke checklist file exists and is well-formed.

These are file-content assertions only — no subprocess, no network, no GCP.
The actual RunPod execution is gated on GCP credentials and is performed by a
human operator following data/RUNPOD_SMOKE_V5.md.
"""
from pathlib import Path

CHECKLIST_PATH = (
    Path(__file__).resolve().parents[2] / "data" / "RUNPOD_SMOKE_V5.md"
)


def test_runpod_smoke_checklist_exists():
    assert CHECKLIST_PATH.exists(), (
        f"Expected RunPod smoke checklist at {CHECKLIST_PATH}"
    )


def test_runpod_smoke_checklist_has_all_steps():
    text = CHECKLIST_PATH.read_text(encoding="utf-8")
    for step in (
        "Step 1",
        "Step 2",
        "Step 3",
        "Step 4",
        "Step 5",
        "Step 6",
        "Step 7",
    ):
        assert step in text, f"Checklist missing '{step}' section"


def test_runpod_smoke_checklist_has_success_criteria():
    text = CHECKLIST_PATH.read_text(encoding="utf-8")
    assert "Success Criteria" in text, "Checklist missing 'Success Criteria' section"
    assert "bbox_coverage_pct" in text, "Checklist missing 'bbox_coverage_pct' metric"
