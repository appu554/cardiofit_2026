"""
Tests for V4 Reviewer API.

Validates:
1. Job listing and filtering
2. Span retrieval
3. Review actions (CONFIRM, REJECT, EDIT)
4. Adding missed spans
5. Complete-review with pending span validation
6. Verified span building (only confirmed/edited/added pass through)
7. Audit trail (reviewer decisions)
8. Job status transitions (PENDING -> IN_REVIEW -> COMPLETED)
9. Error handling (404, invalid actions, missing edited_text)
"""

import sys
from pathlib import Path
from uuid import uuid4

import pytest
import httpx

SHARED_DIR = Path(__file__).resolve().parents[4]
sys.path.insert(0, str(SHARED_DIR))

# Ensure the reviewer_api module is importable
ATOMISER_DIR = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ATOMISER_DIR))

from reviewer_api import (
    InMemoryRepository,
    JobSummary,
    build_verified_spans,
    create_app,
)
from extraction.v4.models import MergedSpan


# =============================================================================
# Fixtures
# =============================================================================


@pytest.fixture
def repo():
    return InMemoryRepository()


@pytest.fixture
def job_id():
    return uuid4()


@pytest.fixture
def seeded_repo(repo, job_id, sample_merged_spans):
    """Repository seeded with a job and its merged spans."""
    job = JobSummary(
        id=job_id,
        source_pdf="KDIGO-2022.pdf",
        guideline_authority="KDIGO",
        guideline_document="KDIGO 2022 Diabetes in CKD",
        review_status="PENDING",
        total_merged_spans=len(sample_merged_spans),
        spans_confirmed=0,
        spans_rejected=0,
        spans_edited=0,
        spans_added=0,
    )
    # Fix job_id on sample spans
    for span in sample_merged_spans:
        span.job_id = job_id
    repo.seed_job(job, sample_merged_spans)
    return repo


@pytest.fixture
def client(seeded_repo):
    """httpx AsyncClient using ASGITransport (httpx 0.28+).

    Returns a sync fixture producing an AsyncClient — the tests themselves
    are async and await client.get/post calls.  No need for async context
    manager since we are hitting an in-memory ASGI transport, not a real server.
    """
    app = create_app(seeded_repo)
    transport = httpx.ASGITransport(app=app)
    return httpx.AsyncClient(transport=transport, base_url="http://testserver")


# =============================================================================
# Job Listing Tests
# =============================================================================


class TestJobListing:
    """Test job listing and filtering."""

    @pytest.mark.asyncio
    async def test_list_all_jobs(self, client):
        response = await client.get("/api/v4/jobs")
        assert response.status_code == 200
        jobs = response.json()
        assert len(jobs) == 1

    @pytest.mark.asyncio
    async def test_list_jobs_filter_pending(self, client):
        response = await client.get("/api/v4/jobs?status=PENDING")
        assert response.status_code == 200
        jobs = response.json()
        assert len(jobs) == 1

    @pytest.mark.asyncio
    async def test_list_jobs_filter_completed_empty(self, client):
        response = await client.get("/api/v4/jobs?status=COMPLETED")
        assert response.status_code == 200
        jobs = response.json()
        assert len(jobs) == 0

    @pytest.mark.asyncio
    async def test_get_job_details(self, client, job_id):
        response = await client.get(f"/api/v4/jobs/{job_id}")
        assert response.status_code == 200
        job = response.json()
        assert job["guideline_authority"] == "KDIGO"

    @pytest.mark.asyncio
    async def test_get_nonexistent_job(self, client):
        response = await client.get(f"/api/v4/jobs/{uuid4()}")
        assert response.status_code == 404


# =============================================================================
# Span Retrieval Tests
# =============================================================================


class TestSpanRetrieval:
    """Test getting merged spans for review."""

    @pytest.mark.asyncio
    async def test_get_spans(self, client, job_id):
        response = await client.get(f"/api/v4/jobs/{job_id}/spans")
        assert response.status_code == 200
        spans = response.json()
        assert len(spans) == 2

    @pytest.mark.asyncio
    async def test_spans_have_review_status(self, client, job_id):
        response = await client.get(f"/api/v4/jobs/{job_id}/spans")
        spans = response.json()
        for span in spans:
            assert span["review_status"] == "PENDING"

    @pytest.mark.asyncio
    async def test_get_spans_nonexistent_job(self, client):
        response = await client.get(f"/api/v4/jobs/{uuid4()}/spans")
        assert response.status_code == 404


# =============================================================================
# Review Action Tests
# =============================================================================


class TestReviewActions:
    """Test CONFIRM, REJECT, and EDIT actions."""

    @pytest.mark.asyncio
    async def test_confirm_span(self, client, job_id, sample_merged_spans):
        span_id = sample_merged_spans[0].id
        response = await client.post(
            f"/api/v4/jobs/{job_id}/spans/{span_id}/review",
            json={"action": "CONFIRM", "reviewer_id": "dr.smith"},
        )
        assert response.status_code == 200
        data = response.json()
        assert data["action"] == "CONFIRM"
        assert data["success"] is True

    @pytest.mark.asyncio
    async def test_confirm_updates_span_status(self, client, job_id, sample_merged_spans):
        span_id = sample_merged_spans[0].id
        await client.post(
            f"/api/v4/jobs/{job_id}/spans/{span_id}/review",
            json={"action": "CONFIRM", "reviewer_id": "dr.smith"},
        )
        # Check span status updated
        response = await client.get(f"/api/v4/jobs/{job_id}/spans")
        spans = response.json()
        confirmed = [s for s in spans if s["id"] == str(span_id)]
        assert confirmed[0]["review_status"] == "CONFIRMED"

    @pytest.mark.asyncio
    async def test_reject_span(self, client, job_id, sample_merged_spans):
        span_id = sample_merged_spans[0].id
        response = await client.post(
            f"/api/v4/jobs/{job_id}/spans/{span_id}/review",
            json={"action": "REJECT", "reviewer_id": "dr.smith", "note": "OCR garbled"},
        )
        assert response.status_code == 200
        assert response.json()["action"] == "REJECT"

    @pytest.mark.asyncio
    async def test_edit_span(self, client, job_id, sample_merged_spans):
        span_id = sample_merged_spans[0].id
        response = await client.post(
            f"/api/v4/jobs/{job_id}/spans/{span_id}/review",
            json={
                "action": "EDIT",
                "reviewer_id": "dr.smith",
                "edited_text": "corrected metformin",
            },
        )
        assert response.status_code == 200
        assert response.json()["action"] == "EDIT"

    @pytest.mark.asyncio
    async def test_edit_updates_reviewer_text(self, client, job_id, sample_merged_spans):
        span_id = sample_merged_spans[0].id
        await client.post(
            f"/api/v4/jobs/{job_id}/spans/{span_id}/review",
            json={
                "action": "EDIT",
                "reviewer_id": "dr.smith",
                "edited_text": "corrected metformin",
            },
        )
        response = await client.get(f"/api/v4/jobs/{job_id}/spans")
        spans = response.json()
        edited = [s for s in spans if s["id"] == str(span_id)]
        assert edited[0]["reviewer_text"] == "corrected metformin"
        assert edited[0]["review_status"] == "EDITED"

    @pytest.mark.asyncio
    async def test_edit_without_text_fails(self, client, job_id, sample_merged_spans):
        span_id = sample_merged_spans[0].id
        response = await client.post(
            f"/api/v4/jobs/{job_id}/spans/{span_id}/review",
            json={"action": "EDIT", "reviewer_id": "dr.smith"},
        )
        assert response.status_code == 400

    @pytest.mark.asyncio
    async def test_invalid_action_fails(self, client, job_id, sample_merged_spans):
        span_id = sample_merged_spans[0].id
        response = await client.post(
            f"/api/v4/jobs/{job_id}/spans/{span_id}/review",
            json={"action": "INVALIDATE", "reviewer_id": "dr.smith"},
        )
        assert response.status_code == 400

    @pytest.mark.asyncio
    async def test_review_nonexistent_span(self, client, job_id):
        response = await client.post(
            f"/api/v4/jobs/{job_id}/spans/{uuid4()}/review",
            json={"action": "CONFIRM", "reviewer_id": "dr.smith"},
        )
        assert response.status_code == 404


# =============================================================================
# Add Span Tests
# =============================================================================


class TestAddSpan:
    """Test adding missed spans."""

    @pytest.mark.asyncio
    async def test_add_span(self, client, job_id):
        response = await client.post(
            f"/api/v4/jobs/{job_id}/spans/add",
            json={
                "text": "canagliflozin",
                "start": 500,
                "end": 513,
                "reviewer_id": "dr.smith",
                "section_id": "4.2.1",
            },
        )
        assert response.status_code == 200
        data = response.json()
        assert data["action"] == "ADD"
        assert data["success"] is True

    @pytest.mark.asyncio
    async def test_added_span_appears_in_list(self, client, job_id):
        await client.post(
            f"/api/v4/jobs/{job_id}/spans/add",
            json={
                "text": "canagliflozin",
                "start": 500,
                "end": 513,
                "reviewer_id": "dr.smith",
            },
        )
        response = await client.get(f"/api/v4/jobs/{job_id}/spans")
        spans = response.json()
        assert len(spans) == 3  # 2 original + 1 added

    @pytest.mark.asyncio
    async def test_added_span_has_added_status(self, client, job_id):
        await client.post(
            f"/api/v4/jobs/{job_id}/spans/add",
            json={
                "text": "canagliflozin",
                "start": 500,
                "end": 513,
                "reviewer_id": "dr.smith",
            },
        )
        response = await client.get(f"/api/v4/jobs/{job_id}/spans")
        spans = response.json()
        added = [s for s in spans if s["review_status"] == "ADDED"]
        assert len(added) == 1
        assert added[0]["text"] == "canagliflozin"

    @pytest.mark.asyncio
    async def test_added_span_has_max_confidence(self, client, job_id):
        await client.post(
            f"/api/v4/jobs/{job_id}/spans/add",
            json={
                "text": "canagliflozin",
                "start": 500,
                "end": 513,
                "reviewer_id": "dr.smith",
            },
        )
        response = await client.get(f"/api/v4/jobs/{job_id}/spans")
        spans = response.json()
        added = [s for s in spans if s["review_status"] == "ADDED"]
        assert added[0]["merged_confidence"] == 1.0


# =============================================================================
# Complete Review Tests
# =============================================================================


class TestCompleteReview:
    """Test review completion and Pipeline 2 trigger."""

    async def _review_all_spans(self, client, job_id, spans):
        """Helper: confirm all spans."""
        for span in spans:
            await client.post(
                f"/api/v4/jobs/{job_id}/spans/{span.id}/review",
                json={"action": "CONFIRM", "reviewer_id": "dr.smith"},
            )

    @pytest.mark.asyncio
    async def test_complete_review_after_all_reviewed(self, client, job_id, sample_merged_spans):
        await self._review_all_spans(client, job_id, sample_merged_spans)
        response = await client.post(
            f"/api/v4/jobs/{job_id}/complete-review",
            json={"reviewer_id": "dr.smith"},
        )
        assert response.status_code == 200
        data = response.json()
        assert data["pipeline2_triggered"] is True
        assert data["verified_span_count"] == 2

    @pytest.mark.asyncio
    async def test_complete_review_fails_with_pending(self, client, job_id):
        """Cannot complete review with pending spans."""
        response = await client.post(
            f"/api/v4/jobs/{job_id}/complete-review",
            json={"reviewer_id": "dr.smith"},
        )
        assert response.status_code == 400
        assert "pending" in response.json()["detail"].lower()

    @pytest.mark.asyncio
    async def test_complete_review_sets_status_completed(self, client, job_id, sample_merged_spans):
        await self._review_all_spans(client, job_id, sample_merged_spans)
        await client.post(
            f"/api/v4/jobs/{job_id}/complete-review",
            json={"reviewer_id": "dr.smith"},
        )
        response = await client.get(f"/api/v4/jobs/{job_id}")
        assert response.json()["review_status"] == "COMPLETED"

    @pytest.mark.asyncio
    async def test_complete_review_twice_fails(self, client, job_id, sample_merged_spans):
        await self._review_all_spans(client, job_id, sample_merged_spans)
        await client.post(
            f"/api/v4/jobs/{job_id}/complete-review",
            json={"reviewer_id": "dr.smith"},
        )
        response = await client.post(
            f"/api/v4/jobs/{job_id}/complete-review",
            json={"reviewer_id": "dr.smith"},
        )
        assert response.status_code == 400
        assert "already completed" in response.json()["detail"].lower()

    @pytest.mark.asyncio
    async def test_rejected_spans_excluded_from_verified(self, client, job_id, sample_merged_spans):
        """Rejected spans should not appear in verified count."""
        # Reject first span, confirm second
        await client.post(
            f"/api/v4/jobs/{job_id}/spans/{sample_merged_spans[0].id}/review",
            json={"action": "REJECT", "reviewer_id": "dr.smith"},
        )
        await client.post(
            f"/api/v4/jobs/{job_id}/spans/{sample_merged_spans[1].id}/review",
            json={"action": "CONFIRM", "reviewer_id": "dr.smith"},
        )
        response = await client.post(
            f"/api/v4/jobs/{job_id}/complete-review",
            json={"reviewer_id": "dr.smith"},
        )
        assert response.json()["verified_span_count"] == 1


# =============================================================================
# Job Status Transition Tests
# =============================================================================


class TestJobStatusTransitions:
    """Test job status flows: PENDING -> IN_REVIEW -> COMPLETED."""

    @pytest.mark.asyncio
    async def test_initial_status_pending(self, client, job_id):
        response = await client.get(f"/api/v4/jobs/{job_id}")
        assert response.json()["review_status"] == "PENDING"

    @pytest.mark.asyncio
    async def test_first_action_sets_in_review(self, client, job_id, sample_merged_spans):
        await client.post(
            f"/api/v4/jobs/{job_id}/spans/{sample_merged_spans[0].id}/review",
            json={"action": "CONFIRM", "reviewer_id": "dr.smith"},
        )
        response = await client.get(f"/api/v4/jobs/{job_id}")
        assert response.json()["review_status"] == "IN_REVIEW"


# =============================================================================
# Audit Trail Tests
# =============================================================================


class TestAuditTrail:
    """Test reviewer decision audit trail."""

    @pytest.mark.asyncio
    async def test_decisions_recorded(self, client, job_id, sample_merged_spans):
        await client.post(
            f"/api/v4/jobs/{job_id}/spans/{sample_merged_spans[0].id}/review",
            json={"action": "CONFIRM", "reviewer_id": "dr.smith"},
        )
        response = await client.get(f"/api/v4/jobs/{job_id}/decisions")
        assert response.status_code == 200
        decisions = response.json()
        assert len(decisions) == 1
        assert decisions[0]["action"] == "CONFIRM"
        assert decisions[0]["reviewer_id"] == "dr.smith"

    @pytest.mark.asyncio
    async def test_multiple_decisions_tracked(self, client, job_id, sample_merged_spans):
        for span in sample_merged_spans:
            await client.post(
                f"/api/v4/jobs/{job_id}/spans/{span.id}/review",
                json={"action": "CONFIRM", "reviewer_id": "dr.smith"},
            )
        response = await client.get(f"/api/v4/jobs/{job_id}/decisions")
        assert len(response.json()) == 2

    @pytest.mark.asyncio
    async def test_edit_decision_has_texts(self, client, job_id, sample_merged_spans):
        span_id = sample_merged_spans[0].id
        await client.post(
            f"/api/v4/jobs/{job_id}/spans/{span_id}/review",
            json={
                "action": "EDIT",
                "reviewer_id": "dr.smith",
                "edited_text": "corrected text",
                "note": "OCR garbled",
            },
        )
        response = await client.get(f"/api/v4/jobs/{job_id}/decisions")
        decision = response.json()[0]
        assert decision["original_text"] == "metformin"
        assert decision["edited_text"] == "corrected text"
        assert decision["note"] == "OCR garbled"

    @pytest.mark.asyncio
    async def test_add_decision_tracked(self, client, job_id):
        await client.post(
            f"/api/v4/jobs/{job_id}/spans/add",
            json={
                "text": "canagliflozin",
                "start": 500,
                "end": 513,
                "reviewer_id": "dr.smith",
                "note": "Missed by all channels",
            },
        )
        response = await client.get(f"/api/v4/jobs/{job_id}/decisions")
        decisions = response.json()
        assert len(decisions) == 1
        assert decisions[0]["action"] == "ADD"
        assert decisions[0]["note"] == "Missed by all channels"


# =============================================================================
# Job Metrics Tests
# =============================================================================


class TestJobMetrics:
    """Test that job metrics update correctly."""

    @pytest.mark.asyncio
    async def test_metrics_after_confirm(self, client, job_id, sample_merged_spans):
        await client.post(
            f"/api/v4/jobs/{job_id}/spans/{sample_merged_spans[0].id}/review",
            json={"action": "CONFIRM", "reviewer_id": "dr.smith"},
        )
        response = await client.get(f"/api/v4/jobs/{job_id}")
        job = response.json()
        assert job["spans_confirmed"] == 1

    @pytest.mark.asyncio
    async def test_metrics_after_all_actions(self, client, job_id, sample_merged_spans):
        # Reject first, confirm second
        await client.post(
            f"/api/v4/jobs/{job_id}/spans/{sample_merged_spans[0].id}/review",
            json={"action": "REJECT", "reviewer_id": "dr.smith"},
        )
        await client.post(
            f"/api/v4/jobs/{job_id}/spans/{sample_merged_spans[1].id}/review",
            json={"action": "CONFIRM", "reviewer_id": "dr.smith"},
        )
        response = await client.get(f"/api/v4/jobs/{job_id}")
        job = response.json()
        assert job["spans_confirmed"] == 1
        assert job["spans_rejected"] == 1


# =============================================================================
# Verified Span Building Tests
# =============================================================================


class TestBuildVerifiedSpans:
    """Test the build_verified_spans function directly."""

    def test_confirmed_passes_through(self, sample_job_id):
        spans = [
            MergedSpan(
                job_id=sample_job_id,
                text="metformin", start=10, end=19,
                contributing_channels=["B", "E"],
                channel_confidences={"B": 1.0, "E": 0.78},
                merged_confidence=0.94,
                review_status="CONFIRMED",
                section_id="4.1.1",
            ),
        ]
        verified = build_verified_spans(spans)
        assert len(verified) == 1
        assert verified[0].text == "metformin"

    def test_rejected_excluded(self, sample_job_id):
        spans = [
            MergedSpan(
                job_id=sample_job_id,
                text="garbled text", start=10, end=22,
                contributing_channels=["E"],
                channel_confidences={"E": 0.5},
                merged_confidence=0.5,
                review_status="REJECTED",
            ),
        ]
        verified = build_verified_spans(spans)
        assert len(verified) == 0

    def test_edited_uses_reviewer_text(self, sample_job_id):
        spans = [
            MergedSpan(
                job_id=sample_job_id,
                text="dapa gliflozin", start=50, end=64,
                contributing_channels=["B"],
                channel_confidences={"B": 1.0},
                merged_confidence=1.0,
                review_status="EDITED",
                reviewer_text="dapagliflozin",
            ),
        ]
        verified = build_verified_spans(spans)
        assert len(verified) == 1
        assert verified[0].text == "dapagliflozin"

    def test_added_passes_through(self, sample_job_id):
        spans = [
            MergedSpan(
                job_id=sample_job_id,
                text="canagliflozin", start=500, end=513,
                contributing_channels=[],
                channel_confidences={},
                merged_confidence=1.0,
                review_status="ADDED",
                reviewer_text="canagliflozin",
            ),
        ]
        verified = build_verified_spans(spans)
        assert len(verified) == 1
        assert verified[0].text == "canagliflozin"

    def test_pending_excluded(self, sample_job_id):
        spans = [
            MergedSpan(
                job_id=sample_job_id,
                text="something", start=10, end=19,
                contributing_channels=["C"],
                channel_confidences={"C": 0.9},
                merged_confidence=0.9,
                review_status="PENDING",
            ),
        ]
        verified = build_verified_spans(spans)
        assert len(verified) == 0

    def test_mixed_statuses(self, sample_job_id):
        spans = [
            MergedSpan(
                job_id=sample_job_id,
                text="confirmed", start=10, end=19,
                contributing_channels=["B"],
                channel_confidences={"B": 1.0},
                merged_confidence=1.0,
                review_status="CONFIRMED",
            ),
            MergedSpan(
                job_id=sample_job_id,
                text="rejected", start=50, end=58,
                contributing_channels=["C"],
                channel_confidences={"C": 0.7},
                merged_confidence=0.7,
                review_status="REJECTED",
            ),
            MergedSpan(
                job_id=sample_job_id,
                text="edited_old", start=100, end=110,
                contributing_channels=["E"],
                channel_confidences={"E": 0.8},
                merged_confidence=0.8,
                review_status="EDITED",
                reviewer_text="edited_new",
            ),
        ]
        verified = build_verified_spans(spans)
        assert len(verified) == 2  # confirmed + edited, not rejected
        texts = {v.text for v in verified}
        assert "confirmed" in texts
        assert "edited_new" in texts
        assert "rejected" not in texts

    def test_extraction_context_has_channel_confidences(self, sample_job_id):
        spans = [
            MergedSpan(
                job_id=sample_job_id,
                text="metformin", start=10, end=19,
                contributing_channels=["B", "E"],
                channel_confidences={"B": 1.0, "E": 0.78},
                merged_confidence=0.94,
                review_status="CONFIRMED",
            ),
        ]
        verified = build_verified_spans(spans)
        ctx = verified[0].extraction_context
        assert ctx["channel_B_confidence"] == 1.0
        assert ctx["channel_E_confidence"] == 0.78
