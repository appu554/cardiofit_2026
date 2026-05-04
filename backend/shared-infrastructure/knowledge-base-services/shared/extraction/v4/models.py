"""
V4 Multi-Channel Extraction Data Models.

These models define the data contracts between all V4 pipeline components:
- RawSpan: Output of each extraction channel (B-F)
- MergedSpan: Signal merger output, stored in DB, reviewer queue item
- ReviewerDecision: Audit trail for reviewer actions
- VerifiedSpan: Reviewer-approved span ready for dossier assembly
- DrugDossier: Per-drug extraction package for L3 Claude
- DossierResult: Per-drug Pipeline 2 tracking
- GuidelineTree/Section/TableBoundary: Channel A structural output
- ChannelOutput: Standard channel return type
- ChannelStatus: Enum for pipeline tracking

These are NEW models for V4 internals. They do NOT replace any V3 models.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime, timezone
from enum import Enum
from typing import Literal, Optional
from uuid import UUID, uuid4

from pydantic import BaseModel, Field

from .provenance import ChannelProvenance


# =============================================================================
# Enums
# =============================================================================

class ChannelStatus(str, Enum):
    """Status of a pipeline channel or stage."""
    PENDING = "PENDING"
    RUNNING = "RUNNING"
    COMPLETED = "COMPLETED"
    FAILED = "FAILED"


class ReviewStatus(str, Enum):
    """Review status for merged spans."""
    PENDING = "PENDING"
    CONFIRMED = "CONFIRMED"
    REJECTED = "REJECTED"
    EDITED = "EDITED"
    ADDED = "ADDED"


class ReviewerAction(str, Enum):
    """Actions a reviewer can take on a span."""
    CONFIRM = "CONFIRM"
    REJECT = "REJECT"
    EDIT = "EDIT"
    ADD = "ADD"


# =============================================================================
# Channel A: Structural Models (not Pydantic — plain dataclasses for speed)
# =============================================================================

@dataclass
class GuidelineSection:
    """A section in the guideline document structure.

    Span-to-section association uses offset ranges:
    A span belongs to this section if section.start_offset <= span.start < section.end_offset
    """
    section_id: str             # e.g., "4.1.1"
    heading: str                # e.g., "Recommendation 4.1.1"
    start_offset: int
    end_offset: int
    page_number: int
    block_type: str             # heading, paragraph, table, list_item, recommendation
    level: int = 1              # ATX heading depth (1=#, 2=##, 3=###, etc.)
    children: list[GuidelineSection] = field(default_factory=list)
    # V5 (Pipeline 1 #2): per-section provenance from Channel A. Populated only
    # when V5_BBOX_PROVENANCE flag is on AND a bbox is available from the
    # structural oracle. Default-None preserves byte-identical V4 behaviour.
    provenance: Optional[ChannelProvenance] = None


@dataclass
class TableBoundary:
    """Boundary and metadata for a table detected by Channel A.

    V4.1: Tables can come from two sources:
    - "marker_pipe": Marker markdown pipe tables (offsets in normalized_text)
    - "granite_otsl": Granite-Docling OTSL tables (offsets are -1, text in otsl_text)
    """
    table_id: str               # e.g., "table_3"
    section_id: str             # parent section
    start_offset: int           # -1 if source="granite_otsl"
    end_offset: int             # -1 if source="granite_otsl"
    headers: list[str]          # column headers
    row_count: int
    page_number: int
    source: str = "marker_pipe"          # "marker_pipe" | "granite_otsl"
    otsl_text: Optional[str] = None      # raw OTSL text if source="granite_otsl"
    # V5 (Pipeline 1 #2): per-table provenance from Channel A. Same semantics
    # as GuidelineSection.provenance (flag-gated, None when bbox unavailable).
    provenance: Optional[ChannelProvenance] = None


@dataclass
class GuidelineTree:
    """Complete structural map of a guideline document.

    Produced by Channel A. Consumed by:
    - All channels: for section_id assignment
    - Channel D: for table boundaries
    - Channel F: for prose-only block filtering

    V4.1: Added alignment_confidence and structural_source.
    V4.2: Granite-Docling is mandatory — structural_source defaults to
    "granite_doctags". Regex fallback removed from Channel A.
    V4.2.2: Added page_map for direct offset→page lookups. Fixes bug where
    spans in multi-page sections inherited the section heading's page number
    instead of their own actual page from the character offset.
    """
    sections: list[GuidelineSection]
    tables: list[TableBoundary]
    total_pages: int
    alignment_confidence: float = 1.0   # ratio of headings successfully aligned
    structural_source: str = "granite_doctags"  # V4.2: always granite_doctags
    # V4.2.2: Character offset → page number mapping from Marker page breaks.
    # Canonical source for ALL span page assignments. Populated by Channel A.
    page_map: dict[int, int] = field(default_factory=dict)

    def get_page_for_offset(self, offset: int) -> int:
        """Get the page number for a character offset via direct page_map lookup.

        V4.2.2: CANONICAL method for span page assignment. Every channel and
        the signal merger must use this instead of section.page_number, which
        only reflects where the section heading is — not where a span within
        that section actually appears in the document.
        """
        if not self.page_map:
            return 1
        page = 1
        for marker_offset, page_num in sorted(self.page_map.items()):
            if marker_offset <= offset:
                page = page_num
            else:
                break
        return page

    def find_section_for_offset(self, offset: int) -> Optional[GuidelineSection]:
        """Find the deepest section containing the given character offset."""
        return self._find_in_sections(self.sections, offset)

    def _find_in_sections(
        self, sections: list[GuidelineSection], offset: int
    ) -> Optional[GuidelineSection]:
        for section in sections:
            if section.start_offset <= offset < section.end_offset:
                # Check children for more specific match
                child_match = self._find_in_sections(section.children, offset)
                return child_match if child_match else section
        return None

    def find_table_for_offset(self, offset: int) -> Optional[TableBoundary]:
        """Find the table containing the given character offset."""
        for table in self.tables:
            if table.start_offset <= offset < table.end_offset:
                return table
        return None

    def get_prose_sections(self) -> list[GuidelineSection]:
        """Get sections with block_type paragraph or list_item (for Channel F)."""
        return self._collect_prose(self.sections)

    def _collect_prose(self, sections: list[GuidelineSection]) -> list[GuidelineSection]:
        result = []
        for section in sections:
            if section.block_type in ("paragraph", "list_item"):
                result.append(section)
            result.extend(self._collect_prose(section.children))
        return result


# =============================================================================
# Section Passage Assembly (V4.2.1 — L3 bridge)
# =============================================================================

@dataclass
class SectionPassage:
    """Assembled prose passage for a guideline section with span provenance.

    Combines section structure (Channel A) with MergedSpan attribution
    (Signal Merger). Primary unit for L3 dossier assembly.

    V4.2.1: Ensures subordinate headings' spans are correctly attributed
    to their governing numbered recommendation after KDIGO reparenting.
    """
    section_id: str
    heading: str
    page_number: int
    prose_text: str                  # section text from normalized_text (tables excluded)
    span_ids: list[UUID]             # MergedSpan.id provenance keys
    span_count: int
    child_section_ids: list[str]     # children section_ids (incl. reparented)
    start_offset: int
    end_offset: int


# =============================================================================
# Channel B-F: RawSpan (output of each extraction channel)
# =============================================================================

class RawSpan(BaseModel):
    """A single text span discovered by one extraction channel.

    Channel A produces GuidelineTree, not RawSpan.
    Channels B-F each produce a list of RawSpan objects.
    L1_RECOVERY spans are injected by the L1 Completeness Oracle for text
    that Marker silently dropped (raw PDF text, not channel-processed).
    """
    id: UUID = Field(default_factory=uuid4)
    channel: Literal["B", "C", "D", "E", "F", "G", "H", "L1_RECOVERY"]
    text: str
    start: int
    end: int
    confidence: float = Field(ge=0.0, le=1.0)
    page_number: Optional[int] = None
    section_id: Optional[str] = None
    table_id: Optional[str] = None
    source_block_type: Optional[
        Literal["heading", "paragraph", "table_cell", "list_item", "recommendation", "table_footnote"]
    ] = None
    channel_metadata: dict = Field(default_factory=dict)
    # PDF bounding box [x0, y0, x1, y1] in PDF points. Available for L1_RECOVERY
    # spans (from PyMuPDF rawdict); NULL for Channels B-F until future work.
    bbox: Optional[list[float]] = None

    model_config = {"frozen": False}


# =============================================================================
# Signal Merger Output -> DB -> Reviewer Queue
# =============================================================================

class MergedSpan(BaseModel):
    """A span after multi-channel signal merging.

    Stored in l2_merged_spans table. This is what the reviewer sees.
    The reviewer does text QA only: confirm/reject/edit/add.
    NO entity typing, NO KB routing, NO classification.
    """
    id: UUID = Field(default_factory=uuid4)
    job_id: UUID
    text: str
    start: int
    end: int
    contributing_channels: list[Literal["B", "C", "D", "E", "F", "G", "H", "L1_RECOVERY", "REVIEWER"]]
    channel_confidences: dict[str, float]
    merged_confidence: float = Field(ge=0.0, le=1.0)
    has_disagreement: bool = False
    disagreement_detail: Optional[str] = None
    # Tiering classifier output: TIER_1 (high-confidence clinical signal),
    # TIER_2 (medium confidence), or NOISE (non-clinical).
    tier: Optional[Literal["TIER_1", "TIER_2", "NOISE"]] = None
    tier_reason: Optional[str] = None
    # Prediction tracking for ML feedback loop (Stage 3 classifiers)
    prediction_id: Optional[str] = None       # UUID assigned at classification time
    classifier_version: Optional[str] = None  # e.g., "rule_based_v4.1" or "trained_v1_20260303"
    page_number: Optional[int] = None
    section_id: Optional[str] = None
    table_id: Optional[str] = None
    # PDF bounding box [x0, y0, x1, y1] in PDF points.
    # Populated for L1_RECOVERY spans; NULL for Channels B-F.
    bbox: Optional[list[float]] = None
    # Adjacent block text from the same PDF page (L1_RECOVERY context).
    # Helps reviewer assess whether Marker's omission was justified.
    surrounding_context: Optional[str] = None
    review_status: Literal["PENDING", "CONFIRMED", "REJECTED", "EDITED", "ADDED"] = "PENDING"
    reviewer_text: Optional[str] = None
    reviewed_by: Optional[str] = None
    reviewed_at: Optional[datetime] = None
    # V5 #2 Bbox Provenance — backward-compatible additive field.
    # Empty list when V5_BBOX_PROVENANCE is off; populated when on.
    # See extraction.v4.provenance.ChannelProvenance.
    channel_provenance: list[ChannelProvenance] = Field(default_factory=list)
    # V5 #4 Consensus Entropy gate — backward-compatible additive field.
    # True when this span was flagged by the CE gate (single-channel, below
    # session median confidence). Spans with ce_flagged=True are suppressed
    # from the default output when V5_CONSENSUS_ENTROPY is on.
    ce_flagged: bool = False

    model_config = {"frozen": False}


# =============================================================================
# Reviewer Audit Trail
# =============================================================================

class ReviewerDecision(BaseModel):
    """Record of a reviewer's action on a merged span.

    Stored in l2_reviewer_decisions table for full audit trail.
    """
    id: UUID = Field(default_factory=uuid4)
    merged_span_id: UUID
    job_id: UUID
    action: Literal["CONFIRM", "REJECT", "EDIT", "ADD"]
    original_text: Optional[str] = None
    edited_text: Optional[str] = None
    reviewer_id: str
    decided_at: datetime = Field(
        default_factory=lambda: datetime.now(timezone.utc)
    )
    note: Optional[str] = None


# =============================================================================
# Verified Span (Reviewer-Approved -> Dossier Assembly -> L3)
# =============================================================================

class VerifiedSpan(BaseModel):
    """A reviewer-approved span ready for dossier assembly and then L3 Claude.

    Replaces the old gliner_entities list[dict] as input to L3.
    The reviewer confirmed the TEXT is correct.
    L3 Claude does ALL classification, entity typing, and KB routing.
    """
    text: str
    start: int
    end: int
    confidence: float = Field(ge=0.0, le=1.0)
    contributing_channels: list[str]
    page_number: Optional[int] = None
    section_id: Optional[str] = None
    table_id: Optional[str] = None
    # Carried from MergedSpan — links reviewer decisions back to the classifier
    # prediction for the ML feedback loop (l2_classifier_shadow_log JOIN).
    prediction_id: Optional[str] = None

    # Machine-generated hints for L3. NOT shown to reviewer. May be incorrect.
    extraction_context: dict = Field(default_factory=dict)
    # e.g., {"channel_B_rxnorm_candidate": "860975", "channel_C_pattern": "egfr_threshold"}


# =============================================================================
# Dossier Assembly (Bridge Between Pipeline 1 and Pipeline 2)
# =============================================================================

@dataclass
class DrugDossier:
    """A self-contained extraction package for one drug, ready for L3.

    Groups all reviewer-verified spans associated with a single drug,
    along with section context and signal metadata.
    """
    drug_name: str
    rxnorm_candidate: Optional[str]         # from Channel B hint, NOT authoritative
    verified_spans: list[VerifiedSpan]
    source_sections: list[str]              # section_ids this drug appears in
    source_pages: list[int]                 # page numbers
    source_text: str                        # full text of enclosing sections
    signal_summary: dict                    # {"thresholds": 3, "doses": 2, "monitoring": 1}


@dataclass
class DossierResult:
    """Per-drug Pipeline 2 tracking result.

    Stored in l2_dossier_results table. Enables retrying L3 for a single
    drug without re-running Pipeline 1 or re-processing other drugs.
    """
    drug_name: str
    rxnorm_candidate: Optional[str]
    span_count: int
    l3_status: str = "PENDING"              # PENDING -> RUNNING -> COMPLETED -> FAILED
    l3_result: Optional[dict] = None        # KB-specific extraction result
    l3_error: Optional[str] = None
    l4_status: str = "PENDING"
    l5_status: str = "PENDING"


# =============================================================================
# Channel Output (Standard return type for all extraction channels)
# =============================================================================

@dataclass
class ChannelOutput:
    """Standard output from an extraction channel.

    Every channel (B-F) returns this. Channel A returns GuidelineTree instead.
    """
    channel: str                            # "B", "C", "D", "E", "F"
    spans: list[RawSpan]
    metadata: dict = field(default_factory=dict)
    elapsed_ms: float = 0.0
    error: Optional[str] = None

    @property
    def span_count(self) -> int:
        return len(self.spans)

    @property
    def success(self) -> bool:
        return self.error is None


# =============================================================================
# CoverageGuard Report (Post-Merge Quality Gate)
# =============================================================================
# CoverageGuard validates extraction quality AFTER Signal Merger, BEFORE facts
# enter the knowledge base.  4 domains × 9 layers → PASS/BLOCK gate verdict.
#
# Domain A: Structural Completeness (prevents omission of entire elements)
# Domain B: Content Exhaustiveness (prevents omission within covered sections)
# Domain C: Integrity Verification (prevents distortion — numeric corruption)
# Domain D: Systemic Meta-Validation (prevents validator blind spots)
# =============================================================================


class InventoryElement(BaseModel):
    """A1: Single structural element expected in the guideline (rec, PP, table, etc.)."""
    element_type: Literal[
        "recommendation", "practice_point", "research_rec",
        "table", "figure", "footnote",
    ]
    element_id: str              # e.g., "Recommendation 1.2.1 (1B)"
    page_number: int
    coverage_status: Literal["COVERED", "MISSING"]
    matched_span_ids: list[str]  # MergedSpan IDs that cover this element

    model_config = {"frozen": False}


class FootnoteBinding(BaseModel):
    """A1d: Footnote marker bound to its table and definition text."""
    marker_char: str             # †, ‡, *, §, ¶, ||, #
    table_id: str
    page_number: int
    footnote_text: str
    tier: Literal["TIER_1", "TIER_2"]
    bound_to_span: bool          # True if footnote text found in merged_spans
    span_id: Optional[str] = None

    model_config = {"frozen": False}


class ResidualFragment(BaseModel):
    """B1: Contiguous uncovered source text fragment with clinical signal analysis."""
    text: str
    char_start: int
    char_end: int
    page_number: int
    trigger_category: Optional[Literal["prescriptive", "prohibitive", "implicit_risk"]] = None
    drug_names_found: list[str] = Field(default_factory=list)
    tier: Literal["TIER_1", "TIER_2", "NOISE"] = "NOISE"

    model_config = {"frozen": False}


class NumericMismatch(BaseModel):
    """C1: A numeric/comparator value that differs between source and extraction."""
    span_id: str
    source_value: str            # e.g., "≥30 mL/min"
    extracted_value: str         # e.g., ">30 mL/min"
    mismatch_type: Literal[
        "comparator_flip", "value_change", "range_boundary_loss",
        "unit_dropped", "unicode_normalization", "text_to_number",
    ]
    action: Literal["BLOCK", "ACCEPT"]
    page_number: int

    model_config = {"frozen": False}


class BranchComparison(BaseModel):
    """B3: Conditional branch completeness check for multi-branch recommendations."""
    section_id: str
    source_threshold_count: int
    extracted_threshold_count: int
    source_connector_count: int  # AND/OR/UNLESS/EXCEPT
    extracted_connector_count: int
    exception_keywords_lost: list[str] = Field(default_factory=list)
    action: Literal["BLOCK", "PASS"]

    model_config = {"frozen": False}


class CorroborationDetail(BaseModel):
    """C3: Channel corroboration scoring for a single span."""
    span_id: str
    contributing_channels: list[str]
    corroboration_score: float   # 0.3 - 1.0 per C3 scoring table
    tier: Literal["TIER_1", "TIER_2"]
    action: Literal["BLOCK", "PASS"]  # BLOCK if Tier 1 and score ≤0.5

    model_config = {"frozen": False}


class GateBlocker(BaseModel):
    """Release gate: A specific condition that caused BLOCK verdict."""
    gate_number: int             # 1-8
    gate_name: str
    blocker_count: int
    fix_priority: int            # 1 = fix first (from gate precedence)
    details: list[str] = Field(default_factory=list)  # human-readable descriptions

    model_config = {"frozen": False}


class CoverageGuardReport(BaseModel):
    """Complete CoverageGuard validation report.

    Produced after Signal Merger, consumed by reviewer UI and release gate.
    Does NOT modify pipeline output — gates whether output proceeds downstream.

    Gate precedence (when multiple fail, fix in this order):
        1. C1 Numeric integrity — value corruption
        2. A2 Structural gaps — entire elements missing
        3. B3 Branch/exception losses — logic incomplete
        4. B1 Residual signals — text dropped
        5. C3 Corroboration warnings — low-confidence spans
        6. B2 Adversarial delta — possible false positives
    """
    job_id: str
    guideline_document: str
    pipeline_version: str
    created_at: datetime = Field(default_factory=lambda: datetime.now(timezone.utc))

    # Domain A — Structural Completeness
    inventory_expected: dict[str, int] = Field(default_factory=dict)
    # e.g., {"recommendation": 47, "practice_point": 31, "table": 12, ...}
    inventory_actual: dict[str, int] = Field(default_factory=dict)
    inventory_elements: list[InventoryElement] = Field(default_factory=list)
    footnote_bindings: list[FootnoteBinding] = Field(default_factory=list)
    density_warnings: list[str] = Field(default_factory=list)

    # Domain B — Content Exhaustiveness
    residual_fragments: list[ResidualFragment] = Field(default_factory=list)
    tier1_residual_count: int = 0
    adversarial_audit_delta: int = 0  # assertions found by LLM but not in spans
    branch_comparisons: list[BranchComparison] = Field(default_factory=list)
    population_action_warnings: list[str] = Field(default_factory=list)

    # Domain C — Integrity Verification
    numeric_mismatches: list[NumericMismatch] = Field(default_factory=list)
    l1_recovery_escalations: list[str] = Field(default_factory=list)
    # span IDs requiring visual verification
    corroboration_details: list[CorroborationDetail] = Field(default_factory=list)

    # Domain D — Systemic Meta-Validation
    dual_llm_agreement_pct: Optional[float] = None
    validator_health: dict[str, float] = Field(default_factory=dict)
    # e.g., {"residual_pct": 2.1, "tier1_delta_rate": 0.0, "numeric_mismatch_rate": 0.5}

    # Gate verdict
    gate_verdict: Literal["PASS", "BLOCK"] = "BLOCK"
    gate_blockers: list[GateBlocker] = Field(default_factory=list)
    total_block_count: int = 0
    total_warning_count: int = 0

    model_config = {"frozen": False}


# =============================================================================
# B2 Adversarial Recall Audit — LLM Assertion Schema
# =============================================================================


class AdversarialAssertion(BaseModel):
    """A minimal atomic prescribing assertion generated by LLM for B2 audit.

    Categories:
        ELIGIBILITY     — who qualifies (population + criteria)
        DOSING          — dose, route, frequency, titration
        CONTRAINDICATION — who must NOT receive the drug
        MONITORING      — labs, vitals, frequency of monitoring
        CONDITIONAL     — if X then Y (branching logic)
    """
    category: Literal[
        "ELIGIBILITY", "DOSING", "CONTRAINDICATION", "MONITORING", "CONDITIONAL",
    ]
    drug_name: str                     # e.g., "empagliflozin"
    assertion_text: str                # e.g., "empagliflozin 10mg once daily for T2D with eGFR ≥20"
    is_negative: bool = False          # [NEGATIVE] tag — assertion about what NOT to do
    is_conditional: bool = False       # [CONDITIONAL] tag — if/then branching
    section_id: str = ""               # which section this came from

    model_config = {"frozen": False}


class AdversarialAuditResult(BaseModel):
    """Schema for LLM tool_use response in B2 audit."""
    assertions: list[AdversarialAssertion] = Field(default_factory=list)

    model_config = {"frozen": False}


class RevalidationReport(BaseModel):
    """Delta report comparing previous and new CoverageGuard runs.

    Produced by re-validation after reviewer edits. Shows which blockers
    were resolved and which remain.
    """
    job_id: str
    previous_verdict: Literal["PASS", "BLOCK"]
    new_verdict: Literal["PASS", "BLOCK"]
    previous_block_count: int
    new_block_count: int
    resolved_blockers: list[str] = Field(default_factory=list)
    remaining_blockers: list[str] = Field(default_factory=list)
    spans_modified_count: int = 0
    revalidated_at: datetime = Field(default_factory=lambda: datetime.now(timezone.utc))

    model_config = {"frozen": False}
