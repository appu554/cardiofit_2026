// ============================================================================
// KB-0 Governance Dashboard - Type Definitions
// Mirrors the Go backend types from internal/policy/types.go
// ============================================================================

// Fact Types - Clinical knowledge categories
export type FactType =
  | 'INTERACTION'
  | 'DRUG_INTERACTION'
  | 'CONTRAINDICATION'
  | 'DOSING_RULE'
  | 'ALLERGY_CROSS_REACTIVITY'
  | 'SAFETY_SIGNAL'
  | 'ORGAN_IMPAIRMENT'
  | 'REPRODUCTIVE_SAFETY'
  | 'THERAPEUTIC_GUIDELINE'
  | 'LAB_REFERENCE'
  | 'LAB_DRUG_INTERACTION'
  | 'FOOD_DRUG_INTERACTION'
  | 'PREGNANCY_CATEGORY'
  | 'RENAL_ADJUSTMENT'
  | 'HEPATIC_ADJUSTMENT'
  | 'GERIATRIC_CONSIDERATION'
  | 'PEDIATRIC_DOSING'
  | 'FORMULARY';

// Fact Status - Lifecycle states
export type FactStatus =
  | 'DRAFT'
  | 'PENDING_REVIEW'
  | 'APPROVED'
  | 'ACTIVE'
  | 'REJECTED'
  | 'SUPERSEDED'
  | 'RETIRED';

// Source Types - How facts were generated
export type SourceType =
  | 'AUTHORITATIVE'     // ONC, FDA, official sources
  | 'CURATED'           // Expert reviewed
  | 'LLM_EXTRACTED'     // AI extraction
  | 'USER_SUBMITTED'    // Clinician submitted
  | 'IMPORTED';         // Bulk import

// Review Priority Levels
export type ReviewPriority = 'CRITICAL' | 'HIGH' | 'STANDARD' | 'LOW';

// Override Types for clinical judgment
export type OverrideType = 'EMERGENCY' | 'INSTITUTIONAL' | 'CLINICAL_JUDGMENT';

// ============================================================================
// Core Data Models
// ============================================================================

// ClinicalFact - matches KB-0 backend fact response (flat structure)
export interface ClinicalFact {
  factId: string;
  factType: string;
  rxcui: string;
  drugName: string;
  genericName?: string;
  manufacturer?: string;
  ndcCodes?: string[];
  atcCodes?: string[];
  scope: string;
  content: Record<string, unknown>;
  sourceType: string;
  sourceId: string;
  sourceVersion?: string;
  extractionMethod?: string;
  authorityPriority: number;
  confidenceScore: number;
  confidenceBand: string;
  status: FactStatus;
  effectiveFrom?: string;
  version?: number;
  reviewPriority?: ReviewPriority;
  reviewDueAt?: string;
  hasConflict: boolean;
  createdAt: string;
  createdBy?: string;
  updatedAt: string;
  activatedAt?: string;

  // Evidence fields (from derived_facts)
  evidenceSpans?: string[];
  sourceSectionId?: string;

  // Legacy fields for backward compatibility
  id?: string;
  drugRxcui?: string;
  interactingDrugRxcui?: string;
  interactingDrugName?: string;
  severity?: 'CRITICAL' | 'HIGH' | 'MODERATE' | 'LOW';
  evidenceLevel?: 'A' | 'B' | 'C' | 'D';
  sourceAuthority?: string;
  confidence?: number;
  assignedReviewer?: string;
  slaDueDate?: string;
  governanceDecision?: string;
  decisionReason?: string;
  conflictGroupId?: string;
  supersededBy?: string;
}

// ============================================================================
// Evidence & Reference Types (Pharmacist Review UI)
// ============================================================================

export type ReferenceType = 'PRIMARY_SOURCE' | 'TERMINOLOGY' | 'REGULATORY' | 'SECONDARY_AUTHORITY';

export interface FactReference {
  type: ReferenceType;
  system: string;
  label: string;
  url: string;
  anchor?: {
    sectionLoinc?: string;
    sectionName?: string;
    tableIndex?: number;
    rowIndex?: number;
    cellText?: string;
  };
  page?: number;
}

export type RejectionReasonCode =
  | 'MISCLASSIFICATION'
  | 'DUPLICATE'
  | 'NOT_IN_SOURCE'
  | 'INVALID_MEDDRA'
  | 'WRONG_DRUG'
  | 'NOISE'
  | 'OTHER';

export interface RejectionReason {
  code: RejectionReasonCode;
  label: string;
  description: string;
}

export const REJECTION_REASONS: RejectionReason[] = [
  { code: 'MISCLASSIFICATION', label: 'Misclassification', description: 'Extracted content doesn\'t match fact type' },
  { code: 'DUPLICATE', label: 'Duplicate', description: 'Same fact already exists' },
  { code: 'NOT_IN_SOURCE', label: 'Not in source document', description: 'Cannot verify in SPL' },
  { code: 'INVALID_MEDDRA', label: 'Invalid MedDRA term', description: 'PT code not valid' },
  { code: 'WRONG_DRUG', label: 'Wrong drug', description: 'RxCUI mismatch' },
  { code: 'NOISE', label: 'Noise', description: 'Statistical artifact, not real AE' },
  { code: 'OTHER', label: 'Other', description: 'Free text reason' },
];

export interface FactContent {
  clinical_effect?: string;
  mechanism?: string;
  management?: string;
  evidence_level?: string;
  risk_level?: string;
  references?: string[];
  trigger_drug?: { rxcui: string; name: string; class: string };
  target_drug?: { rxcui: string; name: string; class: string };
  // Legacy fields
  description?: string;
  recommendation?: string;
  alternatives?: string[];
  clinicalNotes?: string;
}

export interface Reference {
  source: string;
  url?: string;
  pubmedId?: string;
  citation?: string;
}

// ============================================================================
// Queue & Review Models
// ============================================================================

// QueueItem - matches KB-0 backend policy.QueueItem struct (flat structure)
export interface QueueItem {
  factId: string;
  factType: string;
  rxcui: string;
  drugName: string;
  scope: string;
  content: Record<string, unknown>;
  sourceType: string;
  sourceId: string;
  confidenceScore: number;
  confidenceBand: string;
  status: FactStatus;
  reviewPriority: ReviewPriority;
  reviewDueAt: string;
  hasConflict: boolean;
  authorityPriority: number;
  createdAt: string;
  priorityRank: number;
  daysUntilDue: number;
  slaStatus: 'ON_TRACK' | 'AT_RISK' | 'BREACHED';
  // Legacy nested structure support
  fact?: ClinicalFact;
  priority?: ReviewPriority;
  slaDueDate?: string;
  assignedReviewer?: string;
  hasConflicts?: boolean;
  conflictCount?: number;
  queuedAt?: string;
}

export interface ReviewRequest {
  factId: string;
  reviewerId: string;
  action: 'APPROVE' | 'REJECT' | 'ESCALATE';
  reason: string;
  clinicalJustification?: string;
  overrideType?: OverrideType;
  suppress?: boolean;
  suppressionReason?: string;
}

export interface ReviewDecision {
  factId: string;
  decision: 'APPROVED' | 'REJECTED' | 'ESCALATED';
  reviewerId: string;
  reviewerName: string;
  reason: string;
  decidedAt: string;
  signature?: string;
}

// ============================================================================
// Conflict Resolution Models
// ============================================================================

export interface ConflictGroup {
  groupId: string;
  drugRxcui: string;
  drugName: string;
  factType: FactType;
  facts: ClinicalFact[];
  resolutionStrategy: 'AUTHORITY_PRIORITY' | 'RECENCY' | 'MANUAL';
  suggestedWinner?: string;
  resolutionReason?: string;
}

// ============================================================================
// Dashboard & Metrics Models
// ============================================================================

// GovernanceMetrics - matches KB-0 backend FactMetrics struct
export interface GovernanceMetrics {
  totalDraft: number;
  totalApproved: number;
  totalActive: number;
  totalSuperseded: number;
  pendingReview: number;
  criticalPending: number;
  breachedSLA: number;
  atRiskSLA: number;
  withConflicts: number;
  generatedAt: string;
  // Computed fields for dashboard display
  totalFacts?: number;         // = totalDraft + totalApproved + totalActive + totalSuperseded
  pendingReviews?: number;     // alias for pendingReview
  overdueReviews?: number;     // alias for breachedSLA
  todayApproved?: number;      // not tracked in current backend
  todayRejected?: number;      // not tracked in current backend
  avgReviewTimeHours?: number; // not tracked in current backend
  slaCompliancePercent?: number;
  conflictsPending?: number;   // alias for withConflicts
}

// DashboardData - matches KB-0 backend DashboardResponse struct
export interface DashboardData {
  metrics: GovernanceMetrics;
  criticalQueue: QueueItem[];      // Critical priority items
  recentItems: QueueItem[];        // Recently created/updated items
  slaAtRisk: QueueItem[] | null;   // Items at risk of SLA breach
  executorRunning: boolean;
  generatedAt: string;
  // Legacy fields for backward compatibility
  recentActivity?: AuditEvent[];
  queueSummary?: QueueSummary;
  reviewerWorkload?: ReviewerWorkload[];
}

export interface QueueSummary {
  critical: number;
  high: number;
  standard: number;
  low: number;
  overdue: number;
}

export interface ReviewerWorkload {
  reviewerId: string;
  reviewerName: string;
  assigned: number;
  completed: number;
  avgTimeHours: number;
}

// ============================================================================
// Audit & Compliance Models
// ============================================================================

export interface AuditEvent {
  id: string;
  factId: string;
  eventType: AuditEventType;
  actorId: string;
  actorName: string;
  actorRole: string;
  previousState?: string;
  newState?: string;
  reason?: string;
  metadata?: Record<string, unknown>;
  signature: string;
  createdAt: string;
}

export type AuditEventType =
  | 'FACT_CREATED'
  | 'FACT_SUBMITTED_FOR_REVIEW'
  | 'REVIEWER_ASSIGNED'
  | 'FACT_APPROVED'
  | 'FACT_REJECTED'
  | 'FACT_ESCALATED'
  | 'FACT_ACTIVATED'
  | 'FACT_SUPERSEDED'
  | 'CONFLICT_DETECTED'
  | 'CONFLICT_RESOLVED'
  | 'OVERRIDE_APPLIED'
  | 'OVERRIDE_EXPIRED';

// ============================================================================
// API Response Types
// ============================================================================

export interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
  timestamp: string;
}

export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
  hasMore: boolean;
}

// ============================================================================
// Filter & Query Types
// ============================================================================

export interface QueueFilters {
  status?: FactStatus[];
  priority?: ReviewPriority[];
  factType?: FactType[];
  assignedTo?: string;
  slaStatus?: 'ON_TRACK' | 'AT_RISK' | 'BREACHED';
  hasConflicts?: boolean;
  search?: string;
}

export interface SortOptions {
  field: 'priority' | 'slaDueDate' | 'createdAt' | 'confidence';
  direction: 'asc' | 'desc';
}

// ============================================================================
// Executor Status
// ============================================================================

export interface ExecutorStatus {
  running: boolean;
  lastProcessedAt?: string;
  factsProcessed: number;
  errors: number;
  pollIntervalMs: number;
}
