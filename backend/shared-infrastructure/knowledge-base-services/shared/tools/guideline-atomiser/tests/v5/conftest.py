"""Shared pytest fixtures for V5 acceptance tests."""
from __future__ import annotations

import json
from pathlib import Path
from typing import Any

import pytest

ATOMISER_DIR = Path(__file__).resolve().parents[2]
SMOKE_PDFS = [
    "AU-HF-ACS-HCP-Summary-2025.pdf",
    "AU-HF-Cholesterol-Action-Plan-2026.pdf",
]


@pytest.fixture(scope="session")
def atomiser_dir() -> Path:
    return ATOMISER_DIR


@pytest.fixture(scope="session")
def v4_baseline_jobs(atomiser_dir: Path) -> dict[str, Path]:
    """Locate V4 baseline job dirs for the smoke set, by source_pdf."""
    output_dir = atomiser_dir / "data/output/v4"
    found: dict[str, Path] = {}
    for job_dir in output_dir.glob("job_monkeyocr_*"):
        meta_path = job_dir / "job_metadata.json"
        if not meta_path.exists():
            continue
        meta = json.loads(meta_path.read_text())
        src = meta.get("source_pdf")
        if src in SMOKE_PDFS and src not in found:
            found[src] = job_dir
    return found


def load_merged_spans(job_dir: Path) -> list[dict[str, Any]]:
    """Load merged_spans.json as list of dicts.

    Raises:
        FileNotFoundError: if merged_spans.json is missing in job_dir.
    """
    path = job_dir / "merged_spans.json"
    if not path.exists():
        raise FileNotFoundError(f"merged_spans.json not found in {job_dir}")
    return json.loads(path.read_text())
