"""Task 11 — push_to_kb0_gcp.py V5 provenance writes.

Verifies (with mocks; no real GCP/PostgreSQL connection):
  - apply_pending_migrations() reads migration 009 and executes its SQL
  - push_job() writes provenance_v5 JSONB for V5 spans (non-empty
    channel_provenance) and NULL for V4 spans (empty channel_provenance).

GCP is intentionally unavailable in the test environment. The real
end-to-end DB path is verified in Task 14 on RunPod.
"""

from __future__ import annotations

import importlib
import json
import sys
from datetime import datetime, timezone
from pathlib import Path
from unittest import mock
from uuid import uuid4

import pytest

# Ensure shared/ on sys.path so `extraction.v4.*` resolves when the pusher
# does its import-on-load. conftest.py already does this for the test side,
# but we replicate explicitly here for clarity.
_THIS = Path(__file__).resolve()
_ATOMISER = _THIS.parents[2]
_SHARED = _ATOMISER.parent.parent  # shared/
sys.path.insert(0, str(_SHARED))
sys.path.insert(0, str(_ATOMISER))


def _import_pusher():
    """Import the pusher module fresh, with the data/ dir on sys.path."""
    pusher_dir = _ATOMISER / "data"
    sys.path.insert(0, str(pusher_dir))
    if "push_to_kb0_gcp" in sys.modules:
        return importlib.reload(sys.modules["push_to_kb0_gcp"])
    return importlib.import_module("push_to_kb0_gcp")


def _make_span(*, channel_provenance: list[dict] | None) -> dict:
    """Build a JSON-form merged span dict (matches merged_spans.json shape)."""
    span = {
        "id": str(uuid4()),
        "job_id": str(uuid4()),
        "text": "metformin 500 mg twice daily",
        "start": 0,
        "end": 28,
        "contributing_channels": ["B"],
        "channel_confidences": {"B": 0.9},
        "merged_confidence": 0.9,
        "has_disagreement": False,
        "disagreement_detail": None,
        "page_number": 1,
        "section_id": "1",
        "table_id": None,
        "review_status": "PENDING",
        "reviewer_text": None,
        "reviewed_by": None,
        "reviewed_at": None,
        "created_at": datetime.now(timezone.utc).isoformat(),
        "bbox": [10.0, 20.0, 100.0, 40.0],
        "surrounding_context": None,
        "tier": "TIER_1",
        "coverage_guard_alert": None,
        "semantic_tokens": None,
    }
    if channel_provenance is not None:
        span["channel_provenance"] = channel_provenance
    return span


def _v5_provenance_dict() -> dict:
    """A valid serialised ChannelProvenance (matches model_dump shape)."""
    return {
        "channel_id": "B",
        "bbox": {"x0": 10.0, "y0": 20.0, "x1": 100.0, "y1": 40.0},
        "page_number": 1,
        "confidence": 0.9,
        "model_version": "drug_dict-v1.0.0",
        "notes": None,
    }


def _write_job_dir(tmp_path: Path, spans: list[dict]) -> Path:
    """Write a minimal V4 job dir on disk that push_job can read."""
    job_id = str(uuid4())
    for s in spans:
        s["job_id"] = job_id
    job_dir = tmp_path / f"job_monkeyocr_{job_id}"
    job_dir.mkdir()
    (job_dir / "job_metadata.json").write_text(json.dumps({
        "job_id": job_id,
        "source_pdf": "test.pdf",
        "page_range": "1-1",
        "pipeline_version": "5.0.0",
        "l1_backend": "monkeyocr",
        "total_merged_spans": len(spans),
        "section_passages": 0,
        "alignment_confidence": 1.0,
        "l1_oracle": {},
        "created_at": datetime.now(timezone.utc).isoformat(),
    }))
    (job_dir / "merged_spans.json").write_text(json.dumps(spans))
    (job_dir / "section_passages.json").write_text(json.dumps([]))
    (job_dir / "guideline_tree.json").write_text(json.dumps({"total_pages": 1}))
    (job_dir / "normalized_text.txt").write_text("")
    return job_dir


# --- Test 1: migration is applied on connect -------------------------------

def test_migration_is_applied_on_connect():
    pusher = _import_pusher()
    assert hasattr(pusher, "apply_pending_migrations"), (
        "expected push_to_kb0_gcp.apply_pending_migrations to exist"
    )

    fake_cursor = mock.MagicMock()
    fake_conn = mock.MagicMock()
    fake_conn.cursor.return_value = fake_cursor

    pusher.apply_pending_migrations(fake_conn)

    # SQL containing the ALTER TABLE for provenance_v5 must have been executed
    # against the connection.
    executed = [
        call.args[0] if call.args else call.kwargs.get("query", "")
        for call in fake_cursor.execute.call_args_list
    ]
    joined = "\n".join(str(s) for s in executed)
    assert "ALTER TABLE l2_merged_spans" in joined, (
        f"migration SQL not executed; saw: {joined!r}"
    )
    assert "provenance_v5" in joined
    fake_conn.commit.assert_called()


# --- Test 2: provenance_v5 written for V5 span -----------------------------

def test_provenance_v5_written_for_v5_span(tmp_path):
    pusher = _import_pusher()

    span = _make_span(channel_provenance=[_v5_provenance_dict()])
    job_dir = _write_job_dir(tmp_path, [span])

    fake_cursor = mock.MagicMock()
    fake_conn = mock.MagicMock()
    fake_conn.cursor.return_value = fake_cursor

    # Capture rows passed into execute_batch (used for l2_merged_spans).
    captured: dict = {}

    def _capture_execute_batch(cur, sql, rows, page_size=100):
        if "INSERT INTO l2_merged_spans" in sql:
            captured["sql"] = sql
            captured["rows"] = list(rows)

    with mock.patch.object(
        pusher.psycopg2.extras, "execute_batch", side_effect=_capture_execute_batch
    ):
        pusher.push_job(fake_conn, job_dir, dry_run=False)

    assert "sql" in captured, "execute_batch for l2_merged_spans was never called"
    assert "provenance_v5" in captured["sql"], (
        "INSERT statement for l2_merged_spans must include provenance_v5 column"
    )
    assert len(captured["rows"]) == 1
    row = captured["rows"][0]
    # Find the provenance_v5 value — it must be a JSON string (non-null).
    json_strs = [v for v in row if isinstance(v, str) and v.startswith("[")]
    parsed = None
    for js in json_strs:
        try:
            obj = json.loads(js)
        except Exception:
            continue
        if isinstance(obj, list) and obj and isinstance(obj[0], dict) and "channel_id" in obj[0]:
            parsed = obj
            break
    assert parsed is not None, (
        f"provenance_v5 JSON not found in INSERT row: {row!r}"
    )
    assert parsed[0]["channel_id"] == "B"
    assert parsed[0]["bbox"] == {"x0": 10.0, "y0": 20.0, "x1": 100.0, "y1": 40.0}


# --- Test 3: provenance_v5 NULL for V4 span --------------------------------

def test_provenance_v5_null_for_v4_span(tmp_path):
    pusher = _import_pusher()

    span = _make_span(channel_provenance=[])  # V4 default = empty
    job_dir = _write_job_dir(tmp_path, [span])

    fake_cursor = mock.MagicMock()
    fake_conn = mock.MagicMock()
    fake_conn.cursor.return_value = fake_cursor

    captured: dict = {}

    def _capture_execute_batch(cur, sql, rows, page_size=100):
        if "INSERT INTO l2_merged_spans" in sql:
            captured["sql"] = sql
            captured["rows"] = list(rows)

    with mock.patch.object(
        pusher.psycopg2.extras, "execute_batch", side_effect=_capture_execute_batch
    ):
        pusher.push_job(fake_conn, job_dir, dry_run=False)

    assert "sql" in captured
    assert "provenance_v5" in captured["sql"]
    row = captured["rows"][0]
    # No JSON-list-of-channel_provenance value should appear in the row.
    for v in row:
        if isinstance(v, str) and v.startswith("["):
            try:
                obj = json.loads(v)
            except Exception:
                continue
            if isinstance(obj, list) and obj and isinstance(obj[0], dict) and "channel_id" in obj[0]:
                pytest.fail(
                    f"V4 span unexpectedly produced provenance_v5 JSON: {v!r}"
                )
    # The row must contain at least one None (the provenance_v5 slot).
    assert None in row
