// ============================================================================
// SPL Fact Review UI — Type Definitions
// Covers: Triage Dashboard, Fact Review Cards, Drug Sign-Off
// ============================================================================

// ============================================================================
// Completeness Report (from completeness_reports table)
// ============================================================================

export type CompletenessGrade = 'A' | 'B' | 'C' | 'D' | 'F';
export type GateVerdict = 'PASS' | 'WARNING' | 'BLOCK';

export interface CompletenessReport {
  id: string;
  drugName: string;
  rxcui: string;

  // Section coverage
  sectionsCovered: string[];
  sectionsMissing: string[];
  sectionCoveragePct: number;

  // Fact counts
  factCounts: Record<string, number>; // FactType → count
  totalFacts: number;
  factTypesCovered: number;

  // Quality metrics
  meddraMatchRate: number;
  frequencyCovRate: number;
  interactionQual: number;

  // Row extraction
  totalSourceRows: number;
  extractedRows: number;
  skippedRows: number;
  rowCoveragePct: number;
  skipReasonBreakdown: Record<string, number>;

  // Method distribution
  structuredCount: number;
  llmCount: number;
  grammarCount: number;
  deterministicPct: number;

  // Quality assessment
  warnings: string[];
  grade: CompletenessGrade;
  gateVerdict: GateVerdict;

  createdAt: string;
}

// ============================================================================
// SPL Fact Types
// ============================================================================

export type SPLFactType =
  | 'SAFETY_SIGNAL'
  | 'INTERACTION'
  | 'REPRODUCTIVE_SAFETY'
  | 'FORMULARY'
  | 'LAB_REFERENCE'
  | 'ORGAN_IMPAIRMENT';

export type ExtractionMethod =
  | 'STRUCTURED_PARSE'
  | 'LLM_FALLBACK'
  | 'DDI_GRAMMAR'
  | 'PROSE_SCAN'
  | 'SPL_PRODUCT';

export type GovernanceStatus =
  | 'APPROVED'
  | 'PENDING_REVIEW'
  | 'REJECTED'
  | 'SUPERSEDED';

// ============================================================================
// SPL Derived Fact (from derived_facts table)
// ============================================================================

export interface SPLDerivedFact {
  id: string;
  sourceDocumentId: string;
  sourceSectionId: string;
  targetKb: string;
  factType: SPLFactType;
  factKey: string;
  factData: Record<string, unknown>;
  extractionMethod: ExtractionMethod;
  extractionConfidence: number;
  evidenceSpans: Record<string, unknown>;
  governanceStatus: GovernanceStatus;
  reviewedBy?: string;
  reviewedAt?: string;
  reviewNotes?: string;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;

  // Joined fields for display
  drugName?: string;
  rxcui?: string;
  sectionCode?: string;
  sectionName?: string;
}

// ============================================================================
// Triage Dashboard Types
// ============================================================================

export type DrugDisposition = 'REVIEW' | 'INVESTIGATE' | 'OUT_OF_SCOPE' | 'UNDECIDED';

export interface DrugTriageState {
  drugName: string;
  rxcui: string;
  disposition: DrugDisposition;
  dispositionNote?: string;
  completeness: CompletenessReport;
  factsByType: Record<SPLFactType, number>;
  factsByStatus: Record<GovernanceStatus, number>;
  factsByMethod: Record<ExtractionMethod, number>;
  reviewProgress: number; // 0-100%
  totalPendingReview: number;
  totalAutoApproved: number;
}

// ============================================================================
// Fact Card Types (Type-Specific)
// ============================================================================

export interface SafetySignalData {
  conditionName: string;
  meddraPT?: string;
  meddraPTCode?: string;
  meddraSOC?: string;
  frequency?: string;
  frequencyBand?: string;
  severity?: string;
  sourcePhrase?: string;
  meddraValidated: boolean;
}

export interface InteractionData {
  objectDrug: string;
  objectDrugClass?: string;
  clinicalEffect: string;
  direction: 'INCREASE' | 'DECREASE' | 'UNKNOWN';
  clinicalAction: 'CONTRAINDICATED' | 'AVOID' | 'MONITOR' | 'DOSE_ADJUST' | 'INFORM';
  mechanism?: string;
  enzyme?: string;
  sourcePhrase?: string;
}

export interface ReproductiveSafetyData {
  category: 'PREGNANCY' | 'LACTATION' | 'FERTILITY';
  riskLevel: string;
  fdaCategory?: string;
  ridPercent?: string;
  pllrSummary?: string;
  population?: string;
}

export interface FormularyData {
  ndcCode?: string;
  packageForm?: string;
  strength?: string;
  packageSize?: string;
  manufacturer?: string;
}

export interface LabReferenceData {
  labTest: string;
  referenceRange?: string;
  monitoringFrequency?: string;
  clinicalContext?: string;
}

// ============================================================================
// SPL-Specific Alert Types
// ============================================================================

export type SPLAlertType =
  | 'FREQUENCY_MISMATCH'
  | 'LLM_ONLY'
  | 'MEDDRA_UNRESOLVED'
  | 'DIRECTION_CONFLICT'
  | 'MISSING_FACT_TYPE'
  | 'AUTO_APPROVE_SAMPLE';

export interface SPLAlert {
  type: SPLAlertType;
  severity: 'CRITICAL' | 'HIGH' | 'MEDIUM' | 'LOW';
  message: string;
  factId?: string;
  drugName?: string;
}

// ============================================================================
// Review Actions
// ============================================================================

export type ReviewAction = 'CONFIRM' | 'EDIT' | 'REJECT' | 'ADD' | 'ESCALATE';

export interface FactReviewDecision {
  factId: string;
  action: ReviewAction;
  reviewerId: string;
  originalText?: string;
  editedText?: string;
  rejectionReason?: string;
  notes?: string;
  timestamp: string;
}

// ============================================================================
// Drug Sign-Off
// ============================================================================

export interface DrugSignOff {
  drugName: string;
  rxcui: string;
  totalFacts: number;
  confirmed: number;
  edited: number;
  rejected: number;
  added: number;
  autoApprovedSampleSize: number;
  autoApprovedSampleErrors: number;
  factTypeCoverage: Record<SPLFactType, boolean>; // expected types present
  reviewerId: string;
  attestation: string;
  signedAt: string;
}

// ============================================================================
// LOINC Section Reference
// ============================================================================

export const EXPECTED_SECTIONS: Record<string, string> = {
  '34084-4': 'Dosage & Administration',
  '34071-1': 'Warnings',
  '43685-7': 'Warnings & Precautions',
  '34073-7': 'Drug Interactions',
  '34068-7': 'Boxed Warning',
  '34088-5': 'Overdosage',
  '34069-5': 'How Supplied',
};

// ============================================================================
// Grade Display Helpers
// ============================================================================

export const GRADE_CONFIG: Record<CompletenessGrade, {
  color: string;
  bg: string;
  border: string;
  label: string;
}> = {
  A: { color: 'text-emerald-700', bg: 'bg-emerald-50', border: 'border-emerald-200', label: 'Excellent' },
  B: { color: 'text-blue-700', bg: 'bg-blue-50', border: 'border-blue-200', label: 'Good' },
  C: { color: 'text-yellow-700', bg: 'bg-yellow-50', border: 'border-yellow-200', label: 'Fair' },
  D: { color: 'text-orange-700', bg: 'bg-orange-50', border: 'border-orange-200', label: 'Poor' },
  F: { color: 'text-red-700', bg: 'bg-red-50', border: 'border-red-200', label: 'Failed' },
};

export const VERDICT_CONFIG: Record<GateVerdict, {
  color: string;
  bg: string;
  icon: string;
  label: string;
}> = {
  PASS: { color: 'text-emerald-700', bg: 'bg-emerald-100', icon: 'check-circle', label: 'PASS' },
  WARNING: { color: 'text-yellow-700', bg: 'bg-yellow-100', icon: 'alert-triangle', label: 'WARNING' },
  BLOCK: { color: 'text-red-700', bg: 'bg-red-100', icon: 'x-circle', label: 'BLOCK' },
};

export const FACT_TYPE_LABELS: Record<SPLFactType, string> = {
  SAFETY_SIGNAL: 'Safety Signals',
  INTERACTION: 'Drug Interactions',
  REPRODUCTIVE_SAFETY: 'Reproductive Safety',
  FORMULARY: 'Formulary / How Supplied',
  LAB_REFERENCE: 'Lab Monitoring',
  ORGAN_IMPAIRMENT: 'Organ Impairment',
};

export const EXTRACTION_METHOD_LABELS: Record<ExtractionMethod, string> = {
  STRUCTURED_PARSE: 'Table Parse + MedDRA',
  LLM_FALLBACK: 'LLM Extraction',
  DDI_GRAMMAR: 'DDI Grammar',
  PROSE_SCAN: 'Prose Scanner',
  SPL_PRODUCT: 'SPL Product',
};
