"""
V4 Reviewer API — FastAPI backend for the human-in-the-loop review gate.

Pipeline Position:
    Pipeline 1 (Signal Merger → DB) → THIS → Pipeline 2 (Dossier Assembly → L3)

Endpoints:
    GET  /api/v4/jobs                          — list extraction jobs
    GET  /api/v4/jobs/{job_id}                 — get job details + metrics
    GET  /api/v4/jobs/{job_id}/spans           — get merged spans for review
    POST /api/v4/jobs/{job_id}/spans/{span_id}/review — submit decision
    POST /api/v4/jobs/{job_id}/spans/add       — add a missed span
    POST /api/v4/jobs/{job_id}/complete-review  — trigger Pipeline 2
    POST /api/v4/jobs/{job_id}/revalidate      — re-run CoverageGuard on updated spans
    GET  /api/v4/jobs/{job_id}/decisions        — audit trail

Storage:
    Production: PostgreSQL (l2_extraction_jobs, l2_merged_spans, l2_reviewer_decisions)
    Testing:    InMemoryRepository (same interface, dict-backed)
"""

from __future__ import annotations

import json
import logging
import os
import time
from datetime import datetime, timezone
from typing import Optional, Protocol
from uuid import UUID, uuid4

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field

import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent.parent.parent))
sys.path.insert(0, str(Path(__file__).parent))  # for coverage_guard import

from extraction.v4.models import (
    CoverageGuardReport,
    MergedSpan,
    RevalidationReport,
    ReviewerDecision,
    VerifiedSpan,
)

logger = logging.getLogger(__name__)


# =============================================================================
# API Request/Response Models
# =============================================================================

class ReviewRequest(BaseModel):
    """Request body for reviewing a single merged span."""
    action: str  # CONFIRM, REJECT, EDIT
    reviewer_id: str
    edited_text: Optional[str] = None
    note: Optional[str] = None


class AddSpanRequest(BaseModel):
    """Request body for adding a missed span."""
    text: str
    start: int
    end: int
    reviewer_id: str
    section_id: Optional[str] = None
    page_number: Optional[int] = None
    note: Optional[str] = None


class RevalidateRequest(BaseModel):
    """Request body for re-running CoverageGuard on updated spans."""
    reviewer_id: str
    job_dir: str  # path to job artifacts directory
    enable_b2: bool = False  # default: deterministic-only (fast)


class CompleteReviewRequest(BaseModel):
    """Request body for completing the review."""
    reviewer_id: str


class JobSummary(BaseModel):
    """Summary of an extraction job for listing."""
    id: UUID
    source_pdf: str
    guideline_authority: str
    guideline_document: str
    review_status: str
    total_merged_spans: int
    spans_confirmed: int
    spans_rejected: int
    spans_edited: int
    spans_added: int
    created_at: Optional[datetime] = None


class ReviewDecisionResponse(BaseModel):
    """Response for a review action."""
    decision_id: UUID
    span_id: UUID
    action: str
    success: bool


class CompleteReviewResponse(BaseModel):
    """Response for completing the review."""
    job_id: UUID
    verified_span_count: int
    pipeline2_triggered: bool


# =============================================================================
# Repository Protocol (for storage abstraction)
# =============================================================================

class ReviewerRepository(Protocol):
    """Protocol for reviewer data storage."""

    def list_jobs(self, status: Optional[str] = None) -> list[JobSummary]:
        ...

    def get_job(self, job_id: UUID) -> Optional[JobSummary]:
        ...

    def get_spans(self, job_id: UUID) -> list[MergedSpan]:
        ...

    def get_span(self, job_id: UUID, span_id: UUID) -> Optional[MergedSpan]:
        ...

    def update_span(self, span: MergedSpan) -> None:
        ...

    def add_span(self, span: MergedSpan) -> None:
        ...

    def add_decision(self, decision: ReviewerDecision) -> None:
        ...

    def get_decisions(self, job_id: UUID) -> list[ReviewerDecision]:
        ...

    def update_job_status(self, job_id: UUID, status: str) -> None:
        ...

    def update_job_metrics(self, job_id: UUID) -> None:
        ...

    def get_job_artifacts_dir(self, job_id: UUID) -> Optional[str]:
        """Return the filesystem path to this job's artifact directory."""
        ...


# =============================================================================
# In-Memory Repository (for testing)
# =============================================================================

class InMemoryRepository:
    """In-memory implementation of ReviewerRepository for testing."""

    def __init__(self):
        self.jobs: dict[UUID, JobSummary] = {}
        self.spans: dict[UUID, list[MergedSpan]] = {}  # job_id -> spans
        self.decisions: dict[UUID, list[ReviewerDecision]] = {}  # job_id -> decisions
        self._artifact_dirs: dict[UUID, str] = {}  # job_id -> filesystem path

    def list_jobs(self, status: Optional[str] = None) -> list[JobSummary]:
        jobs = list(self.jobs.values())
        if status:
            jobs = [j for j in jobs if j.review_status == status]
        return jobs

    def get_job(self, job_id: UUID) -> Optional[JobSummary]:
        return self.jobs.get(job_id)

    def get_spans(self, job_id: UUID) -> list[MergedSpan]:
        return self.spans.get(job_id, [])

    def get_span(self, job_id: UUID, span_id: UUID) -> Optional[MergedSpan]:
        for span in self.spans.get(job_id, []):
            if span.id == span_id:
                return span
        return None

    def update_span(self, span: MergedSpan) -> None:
        for i, s in enumerate(self.spans.get(span.job_id, [])):
            if s.id == span.id:
                self.spans[span.job_id][i] = span
                return

    def add_span(self, span: MergedSpan) -> None:
        self.spans.setdefault(span.job_id, []).append(span)

    def add_decision(self, decision: ReviewerDecision) -> None:
        self.decisions.setdefault(decision.job_id, []).append(decision)

    def get_decisions(self, job_id: UUID) -> list[ReviewerDecision]:
        return self.decisions.get(job_id, [])

    def update_job_status(self, job_id: UUID, status: str) -> None:
        if job_id in self.jobs:
            self.jobs[job_id].review_status = status

    def update_job_metrics(self, job_id: UUID) -> None:
        if job_id not in self.jobs:
            return
        spans = self.spans.get(job_id, [])
        job = self.jobs[job_id]
        job.total_merged_spans = len(spans)
        job.spans_confirmed = sum(1 for s in spans if s.review_status == "CONFIRMED")
        job.spans_rejected = sum(1 for s in spans if s.review_status == "REJECTED")
        job.spans_edited = sum(1 for s in spans if s.review_status == "EDITED")
        job.spans_added = sum(1 for s in spans if s.review_status == "ADDED")

    def get_job_artifacts_dir(self, job_id: UUID) -> Optional[str]:
        return self._artifact_dirs.get(job_id)

    # Test helper: seed data
    def seed_job(
        self, job: JobSummary, spans: list[MergedSpan],
        artifacts_dir: Optional[str] = None,
    ) -> None:
        self.jobs[job.id] = job
        self.spans[job.id] = spans
        if artifacts_dir:
            self._artifact_dirs[job.id] = artifacts_dir


# =============================================================================
# Verified Span Builder
# =============================================================================

def build_verified_spans(spans: list[MergedSpan]) -> list[VerifiedSpan]:
    """Convert reviewed MergedSpans into VerifiedSpans for Pipeline 2.

    Only CONFIRMED, EDITED, and ADDED spans pass through.
    REJECTED spans are excluded.
    """
    verified = []
    for span in spans:
        if span.review_status == "REJECTED":
            continue
        if span.review_status == "PENDING":
            continue

        # Use reviewer_text if edited, otherwise original text
        text = span.reviewer_text if span.reviewer_text else span.text

        # Build extraction_context from channel metadata
        extraction_context: dict = {}
        for ch, conf in span.channel_confidences.items():
            extraction_context[f"channel_{ch}_confidence"] = conf

        verified.append(VerifiedSpan(
            text=text,
            start=span.start,
            end=span.end,
            confidence=span.merged_confidence,
            contributing_channels=span.contributing_channels,
            page_number=span.page_number,
            section_id=span.section_id,
            table_id=span.table_id,
            prediction_id=span.prediction_id,
            extraction_context=extraction_context,
        ))

    return verified


# =============================================================================
# FastAPI Application
# =============================================================================

def create_app(repo: ReviewerRepository = None) -> FastAPI:
    """Create the FastAPI application with the given repository."""
    if repo is None:
        repo = InMemoryRepository()

    app = FastAPI(
        title="V4 Reviewer API",
        description="Human-in-the-loop review gate for V4 multi-channel extraction",
        version="4.0.0",
    )

    # Store repo on app state
    app.state.repo = repo

    @app.get("/api/v4/jobs", response_model=list[JobSummary])
    def list_jobs(status: Optional[str] = None):
        """List extraction jobs, optionally filtered by review status."""
        return repo.list_jobs(status)

    @app.get("/api/v4/jobs/{job_id}", response_model=JobSummary)
    def get_job(job_id: UUID):
        """Get details of a specific extraction job."""
        job = repo.get_job(job_id)
        if not job:
            raise HTTPException(status_code=404, detail="Job not found")
        return job

    @app.get("/api/v4/jobs/{job_id}/spans", response_model=list[MergedSpan])
    def get_spans(job_id: UUID):
        """Get all merged spans for a job (the reviewer queue)."""
        job = repo.get_job(job_id)
        if not job:
            raise HTTPException(status_code=404, detail="Job not found")
        return repo.get_spans(job_id)

    @app.post(
        "/api/v4/jobs/{job_id}/spans/{span_id}/review",
        response_model=ReviewDecisionResponse,
    )
    def review_span(job_id: UUID, span_id: UUID, request: ReviewRequest):
        """Submit a review decision for a merged span."""
        # Validate action
        valid_actions = {"CONFIRM", "REJECT", "EDIT"}
        if request.action not in valid_actions:
            raise HTTPException(
                status_code=400,
                detail=f"Invalid action: {request.action}. Must be one of {valid_actions}",
            )

        # EDIT requires edited_text
        if request.action == "EDIT" and not request.edited_text:
            raise HTTPException(
                status_code=400,
                detail="EDIT action requires edited_text",
            )

        # Get span
        span = repo.get_span(job_id, span_id)
        if not span:
            raise HTTPException(status_code=404, detail="Span not found")

        # Update span
        now = datetime.now(timezone.utc)
        span.review_status = request.action + ("ED" if request.action != "REJECT" else "ED")
        # Map action to status: CONFIRM→CONFIRMED, REJECT→REJECTED, EDIT→EDITED
        status_map = {"CONFIRM": "CONFIRMED", "REJECT": "REJECTED", "EDIT": "EDITED"}
        span.review_status = status_map[request.action]
        span.reviewed_by = request.reviewer_id
        span.reviewed_at = now
        if request.action == "EDIT":
            span.reviewer_text = request.edited_text

        repo.update_span(span)

        # Record decision
        decision = ReviewerDecision(
            merged_span_id=span_id,
            job_id=job_id,
            action=request.action,
            original_text=span.text,
            edited_text=request.edited_text,
            reviewer_id=request.reviewer_id,
            decided_at=now,
            note=request.note,
        )
        repo.add_decision(decision)

        # Update job metrics
        repo.update_job_metrics(job_id)

        # Set job to IN_REVIEW if first action
        job = repo.get_job(job_id)
        if job and job.review_status == "PENDING":
            repo.update_job_status(job_id, "IN_REVIEW")

        return ReviewDecisionResponse(
            decision_id=decision.id,
            span_id=span_id,
            action=request.action,
            success=True,
        )

    @app.post("/api/v4/jobs/{job_id}/spans/add", response_model=ReviewDecisionResponse)
    def add_span(job_id: UUID, request: AddSpanRequest):
        """Add a missed span (reviewer-identified)."""
        job = repo.get_job(job_id)
        if not job:
            raise HTTPException(status_code=404, detail="Job not found")

        now = datetime.now(timezone.utc)

        # Create new merged span with ADDED status
        new_span = MergedSpan(
            job_id=job_id,
            text=request.text,
            start=request.start,
            end=request.end,
            contributing_channels=["REVIEWER"],
            channel_confidences={"REVIEWER": 1.0},
            merged_confidence=1.0,  # Human-added = max confidence
            section_id=request.section_id,
            page_number=request.page_number,
            review_status="ADDED",
            reviewer_text=request.text,
            reviewed_by=request.reviewer_id,
            reviewed_at=now,
        )
        repo.add_span(new_span)

        # Record decision
        decision = ReviewerDecision(
            merged_span_id=new_span.id,
            job_id=job_id,
            action="ADD",
            edited_text=request.text,
            reviewer_id=request.reviewer_id,
            decided_at=now,
            note=request.note,
        )
        repo.add_decision(decision)

        # Update metrics
        repo.update_job_metrics(job_id)

        return ReviewDecisionResponse(
            decision_id=decision.id,
            span_id=new_span.id,
            action="ADD",
            success=True,
        )

    @app.post(
        "/api/v4/jobs/{job_id}/complete-review",
        response_model=CompleteReviewResponse,
    )
    def complete_review(job_id: UUID, request: CompleteReviewRequest):
        """Complete the review and trigger Pipeline 2.

        This endpoint:
        1. Validates all spans have been reviewed
        2. Converts approved spans to VerifiedSpans
        3. Sets job review_status to COMPLETED
        4. Returns the count of verified spans for Pipeline 2
        """
        job = repo.get_job(job_id)
        if not job:
            raise HTTPException(status_code=404, detail="Job not found")

        if job.review_status == "COMPLETED":
            raise HTTPException(
                status_code=400,
                detail="Review already completed for this job",
            )

        spans = repo.get_spans(job_id)
        pending = [s for s in spans if s.review_status == "PENDING"]
        if pending:
            raise HTTPException(
                status_code=400,
                detail=f"{len(pending)} spans still pending review. "
                f"All spans must be reviewed before completing.",
            )

        # Build verified spans
        verified = build_verified_spans(spans)

        # Update job status
        repo.update_job_status(job_id, "COMPLETED")
        repo.update_job_metrics(job_id)

        return CompleteReviewResponse(
            job_id=job_id,
            verified_span_count=len(verified),
            pipeline2_triggered=True,
        )

    @app.post(
        "/api/v4/jobs/{job_id}/revalidate",
        response_model=RevalidationReport,
    )
    def revalidate(job_id: UUID, request: RevalidateRequest):
        """Re-run CoverageGuard on updated merged spans after reviewer edits.

        This endpoint:
        1. Loads job artifacts (normalized_text, guideline_tree, previous report)
        2. Gets current merged spans (including reviewer additions/edits)
        3. Runs CoverageGuard.validate() on the updated span set
        4. Produces a delta report showing resolved vs remaining blockers
        5. Saves the new CoverageGuard report to the job directory

        The re-validation does NOT re-run Signal Merger or any upstream pipeline.
        Only CoverageGuard re-evaluates the current state of the spans.
        """
        job = repo.get_job(job_id)
        if not job:
            raise HTTPException(status_code=404, detail="Job not found")

        # Resolve job artifacts directory
        job_dir = request.job_dir
        if not job_dir:
            job_dir = repo.get_job_artifacts_dir(job_id)
        if not job_dir or not os.path.isdir(job_dir):
            raise HTTPException(
                status_code=400,
                detail="Job artifacts directory not found. Provide job_dir path.",
            )

        # Load artifacts
        try:
            # Normalized text (L1 parser output)
            text_path = os.path.join(job_dir, "normalized_text.txt")
            with open(text_path, "r") as f:
                normalized_text = f.read()

            # Guideline tree (Channel A structure)
            tree_path = os.path.join(job_dir, "guideline_tree.json")
            tree = None  # CoverageGuard handles None tree gracefully

            # Job metadata for pdf_path
            meta_path = os.path.join(job_dir, "job_metadata.json")
            with open(meta_path, "r") as f:
                job_meta = json.load(f)
            source_pdf = job_meta.get("source_pdf", "")

            # Resolve pdf_path — check common locations
            pdf_path = ""
            pdf_candidates = [
                os.path.join(os.path.dirname(job_dir), source_pdf),
                os.path.join(job_dir, source_pdf),
                os.path.join(
                    os.path.dirname(os.path.dirname(job_dir)),
                    "input", source_pdf,
                ),
            ]
            for candidate in pdf_candidates:
                if os.path.isfile(candidate):
                    pdf_path = candidate
                    break

            # Previous CoverageGuard report (for delta comparison)
            prev_report_path = os.path.join(job_dir, "coverage_guard_report.json")
            prev_report = None
            if os.path.isfile(prev_report_path):
                with open(prev_report_path, "r") as f:
                    prev_report = CoverageGuardReport.model_validate(json.load(f))

        except FileNotFoundError as e:
            raise HTTPException(
                status_code=400,
                detail=f"Missing artifact: {e.filename}",
            )

        # Get current spans (includes reviewer additions/edits)
        current_spans = repo.get_spans(job_id)

        # Apply reviewer edits: use reviewer_text where available
        for span in current_spans:
            if span.reviewer_text and span.review_status in ("EDITED", "ADDED"):
                span.text = span.reviewer_text

        # Exclude rejected spans
        active_spans = [s for s in current_spans if s.review_status != "REJECTED"]

        # Run CoverageGuard
        try:
            from coverage_guard import CoverageGuard

            guard = CoverageGuard(enable_b2=request.enable_b2)
            new_report = guard.validate(
                merged_spans=active_spans,
                tree=tree,
                normalized_text=normalized_text,
                pdf_path=pdf_path,
                oracle_report=None,  # L1 Oracle not re-run
                job_id=str(job_id),
                guideline_document=job_meta.get(
                    "guideline_document", ""
                ),
            )
        except Exception as e:
            logger.exception("CoverageGuard re-validation failed")
            raise HTTPException(
                status_code=500,
                detail=f"CoverageGuard re-validation error: {str(e)}",
            )

        # Save updated report
        cg_data = new_report.model_dump(mode="json")
        cg_path = os.path.join(job_dir, "coverage_guard_report.json")
        with open(cg_path, "w") as f:
            json.dump(cg_data, f, indent=2, default=str)

        # Build delta report
        prev_verdict = prev_report.gate_verdict if prev_report else "BLOCK"
        prev_block_count = prev_report.total_block_count if prev_report else 0
        prev_blocker_names = set()
        if prev_report and prev_report.gate_blockers:
            prev_blocker_names = {
                f"{b.gate_name} ({b.blocker_count})"
                for b in prev_report.gate_blockers
            }

        new_blocker_names = set()
        if new_report.gate_blockers:
            new_blocker_names = {
                f"{b.gate_name} ({b.blocker_count})"
                for b in new_report.gate_blockers
            }

        resolved = sorted(prev_blocker_names - new_blocker_names)
        remaining = sorted(new_blocker_names)

        spans_modified = sum(
            1 for s in current_spans
            if s.review_status in ("EDITED", "ADDED", "REJECTED")
        )

        delta = RevalidationReport(
            job_id=str(job_id),
            previous_verdict=prev_verdict,
            new_verdict=new_report.gate_verdict,
            previous_block_count=prev_block_count,
            new_block_count=new_report.total_block_count,
            resolved_blockers=resolved,
            remaining_blockers=remaining,
            spans_modified_count=spans_modified,
        )

        # Save revalidation history
        reval_history_path = os.path.join(job_dir, "revalidation_history.jsonl")
        with open(reval_history_path, "a") as f:
            reval_entry = {
                "reviewer_id": request.reviewer_id,
                "timestamp": delta.revalidated_at.isoformat(),
                "previous_verdict": delta.previous_verdict,
                "new_verdict": delta.new_verdict,
                "previous_block_count": delta.previous_block_count,
                "new_block_count": delta.new_block_count,
                "resolved_blockers": delta.resolved_blockers,
                "remaining_blockers": delta.remaining_blockers,
                "spans_modified_count": delta.spans_modified_count,
            }
            f.write(json.dumps(reval_entry, default=str) + "\n")

        return delta

    @app.get("/api/v4/jobs/{job_id}/decisions", response_model=list[ReviewerDecision])
    def get_decisions(job_id: UUID):
        """Get audit trail of reviewer decisions for a job."""
        job = repo.get_job(job_id)
        if not job:
            raise HTTPException(status_code=404, detail="Job not found")
        return repo.get_decisions(job_id)

    return app


# Production entry point
app = create_app()
