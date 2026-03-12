// ============================================================================
// Pipeline 1 Review Types — Mirrors Go backend pipeline1/types.go
// ============================================================================

// Span review lifecycle states
export type SpanReviewStatus = 'PENDING' | 'CONFIRMED' | 'REJECTED' | 'EDITED' | 'ADDED';

// Reviewer actions
export type ReviewAction = 'CONFIRM' | 'REJECT' | 'EDIT' | 'ADD';

// Job lifecycle states
export type JobStatus = 'PENDING_REVIEW' | 'IN_PROGRESS' | 'COMPLETED' | 'ARCHIVED';

// ============================================================================
// Core Data Models
// ============================================================================

export interface ExtractionJob {
  jobId: string;
  sourcePdf: string;
  pageRange?: string;
  pipelineVersion: string;
  l1Tag?: string;
  totalMergedSpans: number;
  totalSections: number;
  totalPages: number;
  alignmentConfidence?: number;
  l1OracleStats?: Record<string, unknown>;
  pdfPageOffset: number;

  // Denormalized counters
  spansConfirmed: number;
  spansRejected: number;
  spansEdited: number;
  spansAdded: number;
  spansPending: number;

  status: JobStatus;
  createdAt: string;
  updatedAt: string;
  completedAt?: string;

  // Computed from view
  completionPct?: number;
}

export interface MergedSpan {
  id: string;
  jobId: string;
  text: string;
  startOffset: number;
  endOffset: number;
  contributingChannels: string[];
  channelConfidences?: Record<string, number>;
  mergedConfidence: number;
  hasDisagreement: boolean;
  disagreementDetail?: string;
  pageNumber?: number;
  sectionId?: string;
  tableId?: string;

  // Phase 2: PDF bounding box [x0, y0, x1, y1] in PDF points (L1_RECOVERY spans)
  bbox?: [number, number, number, number];
  // Phase 2: Adjacent block text for L1_RECOVERY context
  surroundingContext?: string;

  // CoverageGuard enrichment (Sprint 1)
  tier?: RiskTier;
  coverageGuardAlert?: CoverageGuardAlert;
  semanticTokens?: SemanticTokens;

  // Review state
  reviewStatus: SpanReviewStatus;
  reviewerText?: string;
  reviewedBy?: string;
  reviewedAt?: string;

  createdAt: string;
}

export interface ReviewerDecision {
  id: string;
  mergedSpanId: string;
  jobId: string;
  action: ReviewAction;
  originalText?: string;
  editedText?: string;
  reviewerId: string;
  decidedAt: string;
  note?: string;
}

export interface SectionPassage {
  jobId: string;
  sectionId: string;
  heading: string;
  pageNumber?: number;
  proseText?: string;
  spanIds: string[];
  spanCount: number;
  childSectionIds: string[];
  startOffset?: number;
  endOffset?: number;
}

export interface GuidelineTree {
  jobId: string;
  treeJson: TreeNode[];
  normalizedText?: string;
}

export interface TreeNode {
  id: string;
  heading: string;
  pageNumber?: number;
  children?: TreeNode[];
}

// ============================================================================
// Request / Response Types
// ============================================================================

export interface SpanReviewRequest {
  reviewerId: string;
  note?: string;
  editedText?: string;
  rejectReason?: RejectReason;
}

export interface AddSpanRequest {
  text: string;
  startOffset: number;
  endOffset: number;
  pageNumber?: number;
  sectionId?: string;
  reviewerId: string;
  note?: string;
}

export interface SpanFilters {
  status?: SpanReviewStatus;
  sectionId?: string;
  pageNumber?: number;
  minConfidence?: number;
  maxConfidence?: number;
  hasDisagreement?: boolean;
  search?: string;
  tier?: RiskTier;
}

export interface JobMetrics {
  totalSpans: number;
  pending: number;
  confirmed: number;
  rejected: number;
  edited: number;
  added: number;
  completionPct: number;
}

// ============================================================================
// API Response Wrappers
// ============================================================================

export interface PaginatedSpans {
  items: MergedSpan[];
  total: number;
  page: number;
  pageSize: number;
  hasMore: boolean;
}

export interface JobListResponse {
  items: ExtractionJob[];
  total: number;
  page: number;
  pageSize: number;
  hasMore: boolean;
}

// ============================================================================
// Page Decision Types (Two-Tier Review)
// ============================================================================

export type PageDecisionAction = 'ACCEPT' | 'FLAG' | 'ESCALATE';

export type PageRisk = 'clean' | 'oracle' | 'disagreement';

export interface PageInfo {
  pageNumber: number;
  sectionIds: string[];
  spanCount: number;
  risk: PageRisk;
  decision?: PageDecisionAction;
  pendingSpans: number;
  reviewedSpans: number;
  // Tier breakdown (CoverageGuard clinical severity)
  tier1Total: number;
  tier1Reviewed: number;
  tier2Total: number;
  tier2Reviewed: number;
  tier3Total: number;
  tier3Reviewed: number;
}

export interface TierReviewStats {
  tier1Total: number;
  tier1Reviewed: number;
  tier2Total: number;
  tier2Reviewed: number;
  tier2Pct: number;
  tier3Total: number;
  tier3Reviewed: number;
}

export interface PageStats {
  totalPages: number;
  pagesAccepted: number;
  pagesFlagged: number;
  pagesEscalated: number;
  pagesNoDecision: number;
  tierStats?: TierReviewStats;
}

export interface PageDecisionRequest {
  action: PageDecisionAction;
  reviewerId: string;
  note?: string;
}

// ============================================================================
// Channel Metadata
// ============================================================================

export type ChannelKey = 'B' | 'C' | 'D' | 'E' | 'F' | 'L1' | 'L1_RECOVERY';

export interface ChannelInfo {
  name: string;
  color: string;
  bg: string;
  border?: string;   // e.g. 'border-l-blue-500' — thick left border for span cards
  bgTint?: string;   // e.g. 'bg-blue-50/40' — subtle background tint for span cards
  icon: string;
}

// ============================================================================
// Review Task Queue (Task-Driven Adjudication)
// ============================================================================

export type ReviewTaskType = 'L1_RECOVERY' | 'DISAGREEMENT' | 'PASSAGE_SPOT_CHECK';
export type ReviewTaskSeverity = 'critical' | 'warning' | 'info';

export interface ReviewTask {
  id: string;
  taskType: ReviewTaskType;
  severity: ReviewTaskSeverity;
  title: string;
  description: string;
  spanId?: string;
  passageId?: string;
  pageNumber?: number;
  sectionId?: string;
  status: 'PENDING' | 'RESOLVED';
  context?: Record<string, unknown>;
}

// ============================================================================
// Patched Passage (L3 consumption — post-review truth)
// ============================================================================

export interface PatchedPassage {
  sectionId: string;
  heading: string;
  pageNumber?: number;
  childSectionIds: string[];
  proseText?: string;
  spanIds: string[];
  spanCount: number;
  patchedAt?: string;
}

// ============================================================================
// View Mode + Left Panel Mode
// ============================================================================

export type ViewMode = 'highlighted' | 'pipeline-html' | 'source-pdf' | 'passage';
export type LeftPanelMode = 'tasks' | 'pages';

// ============================================================================
// Review Phases (Sprint 2 — Governance Bookends)
// ============================================================================

export type ReviewPhase = 1 | 2 | 3 | 4 | 5;

export const PHASE_CONFIG: Record<ReviewPhase, { label: string; description: string }> = {
  1: { label: 'Report Review', description: 'Review CoverageGuard report' },
  2: { label: 'Fact Review', description: 'Verify extracted facts' },
  3: { label: 'Low-Confidence', description: 'Review flagged & recovery items' },
  4: { label: 'Re-Validation', description: 'Validate edited facts' },
  5: { label: 'Sign-Off', description: 'Certify & submit review' },
};

// ============================================================================
// CoverageGuard Alert Types (Sprint 1 — Safety Verification Workflow)
// ============================================================================

// Alert types correspond to CoverageGuard gate domains:
//   numeric_mismatch → C1 (Numeric Integrity)
//   branch_loss      → A2 (Branch Completeness)
//   llm_only         → C3 (Low Corroboration, Channel F only)
//   negation_flip    → C1 (Negation Preservation, sub-check of Numeric Integrity)
export type CoverageGuardAlertType =
  | 'numeric_mismatch'
  | 'branch_loss'
  | 'llm_only'
  | 'negation_flip';

export interface CoverageGuardAlert {
  type: CoverageGuardAlertType;
  label: string;
  detail: string;

  // numeric_mismatch: what the source says vs what was extracted
  sourceValue?: string;
  extractedValue?: string;

  // branch_loss: threshold counts in source vs extracted
  sourceThresholds?: number;
  extractedThresholds?: number;

  // Compound severity: derived from tier × CoverageGuard finding.
  // Tier 1 + numeric_mismatch (C1 BLOCK)    → critical
  // Tier 1 + branch_loss (A2 BLOCK)         → critical
  // Tier 1 + negation_flip (C1 BLOCK)       → critical
  // Tier 1 + llm_only (C3)                  → warning
  // Tier 2/3 + any finding                  → info
  alertSeverity: ReviewTaskSeverity;
}

// Semantic tokens parsed from the extracted text for visual verification.
// Backend populates these from CoverageGuard's token analysis.
export interface SemanticTokens {
  numerics: string[];   // e.g. [">30%", "2 months", "<6.5%"]
  conditions: string[]; // e.g. ["if", "within", "should"]
  negations: string[];  // e.g. ["not treated with dialysis"]
}

// Risk tier from CoverageGuard — determines review requirements.
export type RiskTier = 1 | 2 | 3;

// Structured rejection reasons for auditable review decisions.
export type RejectReason =
  | 'not_in_source'
  | 'numeric_mismatch'
  | 'negation_error'
  | 'out_of_scope'
  | 'duplicate'
  | 'hallucination'
  | 'branch_incomplete'
  | 'escalated_to_sme'
  | 'other';

export const REJECT_REASON_LABELS: Record<RejectReason, string> = {
  not_in_source: 'Not present in source',
  numeric_mismatch: 'Numeric value mismatch',
  negation_error: 'Negation missing or inverted',
  out_of_scope: 'Out of guideline scope',
  duplicate: 'Duplicate of another fact',
  hallucination: 'Hallucinated content',
  branch_incomplete: 'Branch incomplete — needs decomposition',
  escalated_to_sme: 'Escalated to subject matter expert',
  other: 'Other (specify in note)',
};

// ============================================================================
// Review Sub-Phases (Full 5-Phase Workflow with A/B splits)
// ============================================================================

export type ReviewSubPhase = '1' | '2a' | '2b' | '3a' | '3b' | '4' | '5';

export const SUB_PHASE_CONFIG: Record<ReviewSubPhase, {
  label: string; description: string; group: number; groupLabel?: string;
}> = {
  '1':  { label: 'Report Review',        description: 'Review CoverageGuard report',     group: 1 },
  '2a': { label: 'Tier 1 Mandatory',     description: 'Disagreement + spot-check tasks', group: 2, groupLabel: 'Fact Review' },
  '2b': { label: 'Page Browse',          description: 'Tier 2/3 page-level review',      group: 2 },
  '3a': { label: 'F-Only Corroboration', description: 'Low-confidence LLM-only spans',   group: 3, groupLabel: 'Low-Confidence' },
  '3b': { label: 'L1 Recovery Triage',   description: 'OCR recovery verification',       group: 3 },
  '4':  { label: 'Re-Validation',        description: 'CoverageGuard delta check',       group: 4 },
  '5':  { label: 'Sign-Off',             description: 'Certify & submit',                group: 5 },
};

// Ordered sub-phase list for iteration
export const SUB_PHASES: ReviewSubPhase[] = ['1', '2a', '2b', '3a', '3b', '4', '5'];

// ============================================================================
// Output Contract Types (Pipeline 1 → Pipeline 2 handoff)
// ============================================================================

export interface OutputContract {
  confirmedFacts: ConfirmedFact[];
  addedFacts: AddedFact[];
  sectionTree: GuidelineTree;
  evidenceEnvelope: EvidenceEnvelope;
  rejectionLog: RejectionLogEntry[];
}

export interface ConfirmedFact {
  spanId: string;
  factText: string;
  channels: string[];
  mergedConfidence: number;
  pageNumber?: number;
  sectionId?: string;
  reviewerText?: string;
  reviewAction: 'CONFIRMED' | 'EDITED';
  auditTrail: {
    reviewerId: string;
    action: string;
    timestamp: string;
    note?: string;
  };
}

export interface AddedFact {
  spanId: string;
  factText: string;
  channel: 'MANUAL';
  confidence: 1.0;
  pageNumber?: number;
  sectionId?: string;
  auditTrail: {
    reviewerId: string;
    timestamp: string;
    note?: string;
  };
}

export interface EvidenceEnvelope {
  jobId: string;
  sourcePdfSha256: string;
  totalSpans: number;
  confirmed: number;
  edited: number;
  rejected: number;
  added: number;
  coverageGuardReportHash: string;
  reviewStartedAt: string;
  reviewCompletedAt: string;
  reviewerId: string;
}

export interface RejectionLogEntry {
  spanId: string;
  text: string;
  rejectReason: RejectReason;
  channels: string[];
  reviewerId: string;
  timestamp: string;
  note?: string;
}

// ============================================================================
// Revalidation Types (Phase 4 — CoverageGuard delta check)
// ============================================================================

export interface RevalidationResult {
  iteration: number;
  timestamp: string;
  verdict: 'PASS' | 'BLOCK';
  editedSpanCount: number;
  rejectedSpanCount: number;
  addedSpanCount: number;
  deltas: CoverageGuardDelta[];
}

export interface CoverageGuardDelta {
  spanId: string;
  previousAlert?: CoverageGuardAlert;
  currentAlert?: CoverageGuardAlert;
  resolved: boolean;
}

// ============================================================================
// Tier 1 Checklist + Edit Distance (Source-Constrained Edits)
// ============================================================================

export interface Tier1Checklist {
  textMatchesSource: boolean;
  numericsVerified: boolean;
  negationsPreserved: boolean;
  scopeCorrect: boolean;
  noOmissions: boolean;
}

export const TIER1_CHECKLIST_LABELS: Record<keyof Tier1Checklist, string> = {
  textMatchesSource: 'Text matches source document',
  numericsVerified: 'All numeric values verified',
  negationsPreserved: 'Negations preserved correctly',
  scopeCorrect: 'Within guideline scope',
  noOmissions: 'No critical omissions',
};

export interface EditDistanceInfo {
  originalText: string;
  editedText: string;
  levenshteinDistance: number;
  changePercentage: number;
}

// ============================================================================
// Multi-Channel PDF Highlight Colors
// ============================================================================

export const CHANNEL_HIGHLIGHT_COLORS: Record<string, { bg: string; border: string }> = {
  B:           { bg: 'rgba(59,130,246,0.15)',  border: 'rgba(59,130,246,0.6)' },
  C:           { bg: 'rgba(34,197,94,0.15)',   border: 'rgba(34,197,94,0.6)' },
  D:           { bg: 'rgba(168,85,247,0.15)',  border: 'rgba(168,85,247,0.6)' },
  E:           { bg: 'rgba(249,115,22,0.15)',  border: 'rgba(249,115,22,0.6)' },
  F:           { bg: 'rgba(236,72,153,0.15)',  border: 'rgba(236,72,153,0.6)' },
  L1:          { bg: 'rgba(251,191,36,0.15)',  border: 'rgba(251,191,36,0.6)' },
  L1_RECOVERY: { bg: 'rgba(239,68,68,0.15)',   border: 'rgba(239,68,68,0.6)' },
};
