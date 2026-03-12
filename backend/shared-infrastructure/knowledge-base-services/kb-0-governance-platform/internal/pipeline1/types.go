// Package pipeline1 provides types for Pipeline 1 (V4.2.1) extraction review.
// These model the l2_* tables used for pharmacist text QA of merged spans.
package pipeline1

import (
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// STATUS CONSTANTS
// =============================================================================

type SpanReviewStatus string

const (
	SpanStatusPending   SpanReviewStatus = "PENDING"
	SpanStatusConfirmed SpanReviewStatus = "CONFIRMED"
	SpanStatusRejected  SpanReviewStatus = "REJECTED"
	SpanStatusEdited    SpanReviewStatus = "EDITED"
	SpanStatusAdded     SpanReviewStatus = "ADDED"
)

type ReviewAction string

const (
	ActionConfirm ReviewAction = "CONFIRM"
	ActionReject  ReviewAction = "REJECT"
	ActionEdit    ReviewAction = "EDIT"
	ActionAdd     ReviewAction = "ADD"
)

type JobStatus string

const (
	JobStatusPendingReview JobStatus = "PENDING_REVIEW"
	JobStatusInProgress    JobStatus = "IN_PROGRESS"
	JobStatusCompleted     JobStatus = "COMPLETED"
	JobStatusArchived      JobStatus = "ARCHIVED"
)

// =============================================================================
// EXTRACTION JOB
// =============================================================================

// ExtractionJob maps to l2_extraction_jobs.
type ExtractionJob struct {
	JobID              uuid.UUID  `json:"jobId"`
	SourcePDF          string     `json:"sourcePdf"`
	PageRange          *string    `json:"pageRange,omitempty"`
	PipelineVersion    string     `json:"pipelineVersion"`
	L1Tag              *string    `json:"l1Tag,omitempty"`
	TotalMergedSpans   int        `json:"totalMergedSpans"`
	TotalSections      int        `json:"totalSections"`
	TotalPages         int        `json:"totalPages"`
	AlignmentConfidence *float64  `json:"alignmentConfidence,omitempty"`
	L1OracleStats      any        `json:"l1OracleStats,omitempty"`
	PdfPageOffset      int        `json:"pdfPageOffset"`

	// Denormalized counters
	SpansConfirmed int `json:"spansConfirmed"`
	SpansRejected  int `json:"spansRejected"`
	SpansEdited    int `json:"spansEdited"`
	SpansAdded     int `json:"spansAdded"`
	SpansPending   int `json:"spansPending"`

	Status       JobStatus  `json:"status"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	CompletedAt  *time.Time `json:"completedAt,omitempty"`
	CompletedBy  *string    `json:"completedBy,omitempty"`
	SourcePDFPath *string   `json:"sourcePdfPath,omitempty"`

	// Computed field from view
	CompletionPct *float64 `json:"completionPct,omitempty"`
}

// =============================================================================
// MERGED SPAN
// =============================================================================

// MergedSpan maps to l2_merged_spans.
type MergedSpan struct {
	ID                   uuid.UUID        `json:"id"`
	JobID                uuid.UUID        `json:"jobId"`
	Text                 string           `json:"text"`
	StartOffset          int              `json:"startOffset"`
	EndOffset            int              `json:"endOffset"`
	ContributingChannels []string         `json:"contributingChannels"`
	ChannelConfidences   any              `json:"channelConfidences,omitempty"`
	MergedConfidence     float64          `json:"mergedConfidence"`
	HasDisagreement      bool             `json:"hasDisagreement"`
	DisagreementDetail   *string          `json:"disagreementDetail,omitempty"`
	PageNumber           *int             `json:"pageNumber,omitempty"`
	SectionID            *string          `json:"sectionId,omitempty"`
	TableID              *string          `json:"tableId,omitempty"`

	// PDF bounding box [x0, y0, x1, y1] in PDF points. JSONB from DB.
	Bbox               any     `json:"bbox,omitempty"`
	// Adjacent block text for L1_RECOVERY spans (reviewer context).
	SurroundingContext *string `json:"surroundingContext,omitempty"`

	// CoverageGuard analysis (Sprint 1)
	Tier               *int    `json:"tier,omitempty"`               // 1=critical, 2=warning, 3=info
	CoverageGuardAlert any     `json:"coverageGuardAlert,omitempty"` // JSONB alert payload
	SemanticTokens     any     `json:"semanticTokens,omitempty"`     // JSONB highlighting tokens

	// Review state
	ReviewStatus SpanReviewStatus `json:"reviewStatus"`
	ReviewerText *string          `json:"reviewerText,omitempty"`
	ReviewedBy   *string          `json:"reviewedBy,omitempty"`
	ReviewedAt   *time.Time       `json:"reviewedAt,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
}

// =============================================================================
// REVIEWER DECISION (IMMUTABLE AUDIT)
// =============================================================================

// ReviewerDecision maps to l2_reviewer_decisions.
type ReviewerDecision struct {
	ID           uuid.UUID    `json:"id"`
	MergedSpanID uuid.UUID    `json:"mergedSpanId"`
	JobID        uuid.UUID    `json:"jobId"`
	Action       ReviewAction `json:"action"`
	OriginalText *string      `json:"originalText,omitempty"`
	EditedText   *string      `json:"editedText,omitempty"`
	ReviewerID   string       `json:"reviewerId"`
	DecidedAt    time.Time    `json:"decidedAt"`
	Note         *string      `json:"note,omitempty"`
	RejectReason *string      `json:"rejectReason,omitempty"` // Sprint 1: auditable reject category
}

// =============================================================================
// SECTION PASSAGE
// =============================================================================

// SectionPassage maps to l2_section_passages.
type SectionPassage struct {
	JobID           uuid.UUID `json:"jobId"`
	SectionID       string    `json:"sectionId"`
	Heading         string    `json:"heading"`
	PageNumber      *int      `json:"pageNumber,omitempty"`
	ProseText       *string   `json:"proseText,omitempty"`
	SpanIDs         []string  `json:"spanIds"`
	SpanCount       int       `json:"spanCount"`
	ChildSectionIDs []string  `json:"childSectionIds"`
	StartOffset     *int      `json:"startOffset,omitempty"`
	EndOffset       *int      `json:"endOffset,omitempty"`
}

// =============================================================================
// GUIDELINE TREE
// =============================================================================

// GuidelineTree maps to l2_guideline_tree.
type GuidelineTree struct {
	JobID          uuid.UUID `json:"jobId"`
	TreeJSON       any       `json:"treeJson"`
	NormalizedText *string   `json:"normalizedText,omitempty"`
}

// =============================================================================
// REQUEST/RESPONSE TYPES
// =============================================================================

// SpanReviewRequest is the body for confirm/reject/edit actions.
type SpanReviewRequest struct {
	ReviewerID   string  `json:"reviewerId"`
	Note         *string `json:"note,omitempty"`
	EditedText   *string `json:"editedText,omitempty"`
	RejectReason *string `json:"rejectReason,omitempty"` // Sprint 1: structured reject category
}

// CompleteJobRequest is the body for the job completion endpoint.
type CompleteJobRequest struct {
	ReviewerID string  `json:"reviewerId"`
	Note       *string `json:"note,omitempty"`
}

// AddSpanRequest is the body for adding a new span.
type AddSpanRequest struct {
	Text       string  `json:"text"`
	Start      int     `json:"startOffset"`
	End        int     `json:"endOffset"`
	PageNumber *int    `json:"pageNumber,omitempty"`
	SectionID  *string `json:"sectionId,omitempty"`
	ReviewerID string  `json:"reviewerId"`
	Note       *string `json:"note,omitempty"`
}

// SpanFilters captures query-parameter filters for span listing.
type SpanFilters struct {
	Status          *SpanReviewStatus
	SectionID       *string
	PageNumber      *int
	MinConfidence   *float64
	MaxConfidence   *float64
	HasDisagreement *bool
	Search          *string
	Tier            *int // 1=patient safety, 2=clinical accuracy, 3=informational
}

// JobMetrics is a computed summary for a single job.
type JobMetrics struct {
	TotalSpans    int     `json:"totalSpans"`
	Pending       int     `json:"pending"`
	Confirmed     int     `json:"confirmed"`
	Rejected      int     `json:"rejected"`
	Edited        int     `json:"edited"`
	Added         int     `json:"added"`
	CompletionPct float64 `json:"completionPct"`
}

// =============================================================================
// PAGE DECISIONS (Two-Tier Review)
// =============================================================================

type PageDecisionAction string

const (
	PageActionAccept   PageDecisionAction = "ACCEPT"
	PageActionFlag     PageDecisionAction = "FLAG"
	PageActionEscalate PageDecisionAction = "ESCALATE"
)

// PageDecision maps to l2_page_decisions.
type PageDecision struct {
	ID         uuid.UUID          `json:"id"`
	JobID      uuid.UUID          `json:"jobId"`
	PageNumber int                `json:"pageNumber"`
	Action     PageDecisionAction `json:"action"`
	ReviewerID string             `json:"reviewerId"`
	Note       *string            `json:"note,omitempty"`
	DecidedAt  time.Time          `json:"decidedAt"`
}

// PageDecisionRequest is the body for page-level decide endpoint.
type PageDecisionRequest struct {
	Action     PageDecisionAction `json:"action"`
	ReviewerID string             `json:"reviewerId"`
	Note       *string            `json:"note,omitempty"`
}

// PageRisk classifies page-level risk detected during extraction.
type PageRisk string

const (
	PageRiskClean        PageRisk = "clean"
	PageRiskOracle       PageRisk = "oracle"
	PageRiskDisagreement PageRisk = "disagreement"
)

// PageInfo is a computed per-page summary assembled from span/passage/decision data.
type PageInfo struct {
	PageNumber    int                 `json:"pageNumber"`
	SectionIDs    []string            `json:"sectionIds"`
	SpanCount     int                 `json:"spanCount"`
	Risk          PageRisk            `json:"risk"`
	Decision      *PageDecisionAction `json:"decision"`
	PendingSpans  int                 `json:"pendingSpans"`
	ReviewedSpans int                 `json:"reviewedSpans"`
	// Tier breakdown (CoverageGuard clinical severity)
	Tier1Total    int `json:"tier1Total"`
	Tier1Reviewed int `json:"tier1Reviewed"`
	Tier2Total    int `json:"tier2Total"`
	Tier2Reviewed int `json:"tier2Reviewed"`
	Tier3Total    int `json:"tier3Total"`
	Tier3Reviewed int `json:"tier3Reviewed"`
}

// TierReviewStats summarizes review progress per clinical tier.
type TierReviewStats struct {
	Tier1Total    int     `json:"tier1Total"`
	Tier1Reviewed int     `json:"tier1Reviewed"`
	Tier2Total    int     `json:"tier2Total"`
	Tier2Reviewed int     `json:"tier2Reviewed"`
	Tier2Pct      float64 `json:"tier2Pct"` // tier2Reviewed/tier2Total * 100
	Tier3Total    int     `json:"tier3Total"`
	Tier3Reviewed int     `json:"tier3Reviewed"`
}

// PageStats is an aggregate of page decisions for a job.
type PageStats struct {
	TotalPages      int              `json:"totalPages"`
	PagesAccepted   int              `json:"pagesAccepted"`
	PagesFlagged    int              `json:"pagesFlagged"`
	PagesEscalated  int              `json:"pagesEscalated"`
	PagesNoDecision int              `json:"pagesNoDecision"`
	TierStats       *TierReviewStats `json:"tierStats,omitempty"`
}

// =============================================================================
// REVIEW TASK QUEUE (Task-Driven Adjudication)
// =============================================================================

// ReviewTaskType classifies the source of a flagged review task.
type ReviewTaskType string

const (
	TaskL1Recovery       ReviewTaskType = "L1_RECOVERY"
	TaskDisagreement     ReviewTaskType = "DISAGREEMENT"
	TaskPassageSpotCheck ReviewTaskType = "PASSAGE_SPOT_CHECK"
)

// ReviewTaskSeverity maps task types to triage priority.
type ReviewTaskSeverity string

const (
	SeverityCritical ReviewTaskSeverity = "critical"
	SeverityWarning  ReviewTaskSeverity = "warning"
	SeverityInfo     ReviewTaskSeverity = "info"
)

// ReviewTask is a computed item in the reviewer's task queue.
// Tasks are derived from pipeline output, not stored in a table.
type ReviewTask struct {
	ID          string             `json:"id"`
	TaskType    ReviewTaskType     `json:"taskType"`
	Severity    ReviewTaskSeverity `json:"severity"`
	Title       string             `json:"title"`
	Description string             `json:"description"`
	SpanID      *string            `json:"spanId,omitempty"`
	PassageID   *string            `json:"passageId,omitempty"`
	PageNumber  *int               `json:"pageNumber,omitempty"`
	SectionID   *string            `json:"sectionId,omitempty"`
	Status      string             `json:"status"` // PENDING or RESOLVED
	Context     map[string]any     `json:"context,omitempty"`
}

// PatchedPassage is a section passage with reviewer edits applied.
// Read from the l2_passages_for_l3 view.
type PatchedPassage struct {
	SectionID       string     `json:"sectionId"`
	Heading         string     `json:"heading"`
	PageNumber      *int       `json:"pageNumber,omitempty"`
	ChildSectionIDs []string   `json:"childSectionIds"`
	ProseText       *string    `json:"proseText,omitempty"`
	SpanIDs         []string   `json:"spanIds"`
	SpanCount       int        `json:"spanCount"`
	PatchedAt       *time.Time `json:"patchedAt,omitempty"`
}

// =============================================================================
// REVALIDATION (Phase 4 — CoverageGuard Delta Check)
// =============================================================================

// RevalidationVerdict is the outcome of a re-validation run.
type RevalidationVerdict string

const (
	VerdictPass  RevalidationVerdict = "PASS"
	VerdictBlock RevalidationVerdict = "BLOCK"
)

// RevalidationRun maps to l2_revalidation_runs.
type RevalidationRun struct {
	ID                uuid.UUID           `json:"id"`
	JobID             uuid.UUID           `json:"jobId"`
	Iteration         int                 `json:"iteration"`
	Verdict           RevalidationVerdict `json:"verdict"`
	EditedSpanCount   int                 `json:"editedSpanCount"`
	RejectedSpanCount int                 `json:"rejectedSpanCount"`
	AddedSpanCount    int                 `json:"addedSpanCount"`
	Deltas            []CoverageGuardDelta `json:"deltas"`
	TriggeredBy       *string             `json:"triggeredBy,omitempty"`
	CreatedAt         time.Time           `json:"timestamp"`
}

// CoverageGuardDelta tracks changes to an individual CoverageGuard alert
// between re-validation iterations.
type CoverageGuardDelta struct {
	SpanID        string `json:"spanId"`
	PreviousAlert any    `json:"previousAlert,omitempty"`
	CurrentAlert  any    `json:"currentAlert,omitempty"`
	Resolved      bool   `json:"resolved"`
}

// RevalidateRequest is the body for POST /jobs/{job_id}/revalidate.
type RevalidateRequest struct {
	ReviewerID string `json:"reviewerId,omitempty"`
}

// =============================================================================
// OUTPUT CONTRACT (Phase 5 — Pipeline 2 Handoff)
// =============================================================================

// OutputContract is the 5-section package assembled in Phase 5 for Pipeline 2.
type OutputContract struct {
	// Section 1: Confirmed + edited spans with audit trail
	ConfirmedFacts []ConfirmedFact `json:"confirmedFacts"`
	// Section 2: Reviewer-added spans
	AddedFacts []AddedFact `json:"addedFacts"`
	// Section 3: Guideline section tree with fact counts
	SectionTree *GuidelineTree `json:"sectionTree"`
	// Section 4: Job metadata, review stats, hash
	EvidenceEnvelope *EvidenceEnvelope `json:"evidenceEnvelope"`
	// Section 5: Rejected spans with reasons
	RejectionLog []RejectionLogEntry `json:"rejectionLog"`
}

// ConfirmedFact is a span accepted or edited by the reviewer.
type ConfirmedFact struct {
	SpanID           string   `json:"spanId"`
	FactText         string   `json:"factText"`
	Channels         []string `json:"channels"`
	MergedConfidence float64  `json:"mergedConfidence"`
	PageNumber       *int     `json:"pageNumber,omitempty"`
	SectionID        *string  `json:"sectionId,omitempty"`
	ReviewerText     *string  `json:"reviewerText,omitempty"`
	ReviewAction     string   `json:"reviewAction"` // CONFIRMED or EDITED
	AuditTrail       any      `json:"auditTrail"`
}

// AddedFact is a reviewer-created span.
type AddedFact struct {
	SpanID     string  `json:"spanId"`
	FactText   string  `json:"factText"`
	Channel    string  `json:"channel"`    // always "MANUAL"
	Confidence float64 `json:"confidence"` // always 1.0
	PageNumber *int    `json:"pageNumber,omitempty"`
	SectionID  *string `json:"sectionId,omitempty"`
	AuditTrail any     `json:"auditTrail"`
}

// EvidenceEnvelope is the metadata section of the output contract.
type EvidenceEnvelope struct {
	JobID           string `json:"jobId"`
	SourcePDF       string `json:"sourcePdf"`
	PipelineVersion string `json:"pipelineVersion"`
	TotalSpans      int    `json:"totalSpans"`
	Confirmed       int    `json:"confirmed"`
	Edited          int    `json:"edited"`
	Rejected        int    `json:"rejected"`
	Added           int    `json:"added"`
	ReviewerID      string `json:"reviewerId"`
	ReviewStartedAt string `json:"reviewStartedAt"`
	ReviewCompletedAt string `json:"reviewCompletedAt"`
}

// RejectionLogEntry is a rejected span with structured reason.
type RejectionLogEntry struct {
	SpanID       string   `json:"spanId"`
	Text         string   `json:"text"`
	RejectReason *string  `json:"rejectReason,omitempty"`
	Channels     []string `json:"channels"`
	ReviewerID   *string  `json:"reviewerId,omitempty"`
	Timestamp    *string  `json:"timestamp,omitempty"`
	Note         *string  `json:"note,omitempty"`
}

// AssembleContractRequest is the body for POST /jobs/{job_id}/output-contract.
type AssembleContractRequest struct {
	ReviewerID string `json:"reviewerId"`
}
