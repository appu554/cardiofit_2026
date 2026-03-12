// Package transaction provides medication transaction management for KB-19.
// These types are MOVED from medication-advisor-engine/advisor/engine.go
// as part of V3 architecture refactoring (Med-Advisor = Risk Computer, KB-19 = Transaction Authority).
package transaction

import (
	"time"

	"github.com/google/uuid"

	"kb-19-protocol-orchestrator/internal/transaction/shared"
)

// =============================================================================
// TYPE ALIASES for shared types (breaks circular dependency with rules package)
// =============================================================================

// ClinicalCode is an alias for shared.ClinicalCode
// This allows rules package to return shared.ClinicalCode while transaction package uses ClinicalCode
type ClinicalCode = shared.ClinicalCode

// HardBlock is an alias for shared.HardBlock
// This allows rules package to return shared.HardBlock while transaction package uses HardBlock
type HardBlock = shared.HardBlock

// LabValue is an alias for shared.LabValue
// This allows rules package to use shared.LabValue while transaction package uses LabValue
type LabValue = shared.LabValue

// =============================================================================
// MOVED FROM: medication-advisor-engine/advisor/engine.go lines 195-221
// =============================================================================

// AuditTrailSummary provides Tier-7 compliance audit trail summary
// This ensures every medication decision is court-defensible and regulator-compliant
type AuditTrailSummary struct {
	// Core Identifiers
	TraceID       string `json:"trace_id"`       // Unique trace ID for this decision chain
	SessionID     string `json:"session_id"`     // User session ID
	TransactionID string `json:"transaction_id"` // FHIR transaction bundle ID if applicable

	// Evidence Chain
	EvidenceCount  int      `json:"evidence_count"`   // Number of evidence steps recorded
	KBServicesUsed []string `json:"kb_services_used"` // Which KB services were consulted
	RulesEvaluated int      `json:"rules_evaluated"`  // Number of clinical rules evaluated

	// Safety Summary
	SafetyChecks      int `json:"safety_checks"`      // Total safety checks performed
	BlocksGenerated   int `json:"blocks_generated"`   // Hard blocks generated
	WarningsGenerated int `json:"warnings_generated"` // Warnings generated

	// Disposition Tracking
	DispositionReason string `json:"disposition_reason"` // Why this disposition was chosen
	RequiresAck       bool   `json:"requires_ack"`       // Whether acknowledgment is required
	AckText           string `json:"ack_text,omitempty"` // Acknowledgment text if required

	// Governance Metadata
	GovernanceLevel  string `json:"governance_level"`  // TIER_7_COMPLETE, TIER_6_PARTIAL, etc.
	ComplianceStatus string `json:"compliance_status"` // COMPLIANT, REQUIRES_REVIEW, NON_COMPLIANT
	AuditHash        string `json:"audit_hash"`        // SHA256 hash of audit trail for integrity
	Timestamp        string `json:"timestamp"`         // ISO8601 timestamp
}

// =============================================================================
// MOVED FROM: medication-advisor-engine/advisor/engine.go lines 223-233
// =============================================================================

// GeneratedTask represents a task generated from medication advisory
type GeneratedTask struct {
	TaskType     string                 `json:"task_type"`
	Priority     string                 `json:"priority"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	DueInMinutes int                    `json:"due_in_minutes"`
	AssignedRole string                 `json:"assigned_role"`
	Source       string                 `json:"source"` // Which KB or rule generated this
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// =============================================================================
// MOVED FROM: medication-advisor-engine/advisor/engine.go lines 236-268
// =============================================================================

// DispositionCode represents the recommended next action
type DispositionCode string

const (
	// Primary Dispositions
	DispositionDispense          DispositionCode = "DISPENSE"             // Safe to proceed with medication
	DispositionHoldForReview     DispositionCode = "HOLD_FOR_REVIEW"      // Requires pharmacist review
	DispositionHoldForApproval   DispositionCode = "HOLD_FOR_APPROVAL"    // Requires physician approval
	DispositionEscalateToMD      DispositionCode = "ESCALATE_TO_MD"       // Requires immediate MD review
	DispositionHardStop          DispositionCode = "HARD_STOP"            // Cannot proceed without acknowledgment
	DispositionRequireOverride   DispositionCode = "REQUIRE_OVERRIDE"     // Requires documented override
	DispositionDeferToSpecialist DispositionCode = "DEFER_TO_SPECIALIST"  // Requires specialist consultation
	DispositionRecalculate       DispositionCode = "RECALCULATE"          // Need additional information

	// Lab-Based Dispositions (KB-16)
	DispositionLabContraindicated      DispositionCode = "LAB_CONTRAINDICATED"            // Lab values contraindicate medication
	DispositionRequireSpecialistReview DispositionCode = "REQUIRES_SPECIALIST_REVIEW"     // Requires specialist due to lab abnormalities
	DispositionPatientStabilization    DispositionCode = "PATIENT_STABILIZATION_REQUIRED" // Patient must be stabilized before medication

	// ICU-Specific Dispositions (Tier-10)
	DispositionICUHardStop            DispositionCode = "ICU_HARD_STOP"             // ICU-specific critical block
	DispositionHemodynamicInstability DispositionCode = "HEMODYNAMIC_INSTABILITY"  // Patient hemodynamically unstable
	DispositionVentilatorDependent    DispositionCode = "VENTILATOR_DEPENDENT"     // Requires ventilator safety review
	DispositionCRRTAdjustment         DispositionCode = "CRRT_ADJUSTMENT_REQUIRED" // CRRT dosing recalculation needed
	DispositionMultiOrganFailure      DispositionCode = "MULTI_ORGAN_FAILURE"      // MOF requires intensive review
	DispositionSepsisProtocol         DispositionCode = "SEPSIS_PROTOCOL"          // Must follow sepsis protocol
	DispositionNeurologicalMonitoring DispositionCode = "NEUROLOGICAL_MONITORING"  // Neurological status requires monitoring

	// ICU Safety Evaluation Dispositions (Tier-10 Phase 2)
	DispositionICUCriticalHardStop DispositionCode = "ICU_CRITICAL_HARD_STOP" // Critical ICU block - immediate intervention
	DispositionICUHighRisk         DispositionCode = "ICU_HIGH_RISK"          // High risk medication in ICU context
	DispositionICUDoseAdjustment   DispositionCode = "ICU_DOSE_ADJUSTMENT"    // ICU-specific dose modification needed
	DispositionICUSafe             DispositionCode = "ICU_SAFE"               // Safe for ICU administration
)

// =============================================================================
// MOVED FROM: medication-advisor-engine/advisor/engine.go lines 270-288
// =============================================================================

// GovernanceEventType represents types of governance events for Tier-7 compliance
type GovernanceEventType string

const (
	// Safety Governance Events
	GovernanceEventPolicyViolation     GovernanceEventType = "POLICY_VIOLATION"     // Protocol/policy violation detected
	GovernanceEventPatientSafetyRisk   GovernanceEventType = "PATIENT_SAFETY_RISK"  // Patient safety concern identified
	GovernanceEventLabContraindication GovernanceEventType = "LAB_CONTRAINDICATION" // Lab-based medication block
	GovernanceEventDDIHardStop         GovernanceEventType = "DDI_HARD_STOP"        // Drug-drug interaction block
	GovernanceEventPregnancyRisk       GovernanceEventType = "PREGNANCY_RISK"       // Teratogenic medication in pregnancy
	GovernanceEventBlackBoxWarning     GovernanceEventType = "BLACK_BOX_WARNING"    // FDA black box warning triggered

	// ICU Governance Events
	GovernanceEventICUCriticalBlock GovernanceEventType = "ICU_CRITICAL_BLOCK" // ICU-specific critical safety block
	GovernanceEventHemodynamicRisk  GovernanceEventType = "HEMODYNAMIC_RISK"   // Hemodynamic instability concern
	GovernanceEventVentilatorRisk   GovernanceEventType = "VENTILATOR_RISK"    // Ventilator/respiratory safety concern
	GovernanceEventRenalRisk        GovernanceEventType = "RENAL_RISK"         // Nephrotoxicity or renal dosing concern
	GovernanceEventNeurologicalRisk GovernanceEventType = "NEUROLOGICAL_RISK"  // Sedation/delirium/neurological concern
)

// =============================================================================
// MOVED FROM: medication-advisor-engine/advisor/engine.go lines 290-310
// =============================================================================

// GovernanceEvent represents a tracked governance event for audit
type GovernanceEvent struct {
	ID             uuid.UUID           `json:"id"`
	EventType      GovernanceEventType `json:"event_type"`
	Severity       string              `json:"severity"` // critical, high, medium, low
	Timestamp      time.Time           `json:"timestamp"`
	PatientID      uuid.UUID           `json:"patient_id"`
	ProviderID     string              `json:"provider_id"`
	MedicationCode string              `json:"medication_code,omitempty"`
	TriggerCode    string              `json:"trigger_code,omitempty"`  // What triggered the event (LOINC, SNOMED, etc.)
	TriggerValue   string              `json:"trigger_value,omitempty"` // Lab value, condition, etc.
	BlockType      string              `json:"block_type"`
	KBSource       string              `json:"kb_source"`
	RuleID         string              `json:"rule_id"`
	Description    string              `json:"description"`
	RequiresAck    bool                `json:"requires_ack"`
	Acknowledged   bool                `json:"acknowledged"`
	AcknowledgedBy string              `json:"acknowledged_by,omitempty"`
	AcknowledgedAt *time.Time          `json:"acknowledged_at,omitempty"`
	HashChain      string              `json:"hash_chain"` // Immutable hash for audit trail
}

// =============================================================================
// NOTE: HardBlock is now a type alias to shared.HardBlock (defined at top of file)
// This breaks the circular dependency with the rules package.
// Original definition MOVED FROM: medication-advisor-engine/advisor/engine.go lines 312-330
// =============================================================================

// =============================================================================
// MOVED FROM: medication-advisor-engine/advisor/engine.go lines 332-341
// =============================================================================

// ExcludedDrugInfo represents a drug that was excluded during safety screening
type ExcludedDrugInfo struct {
	Medication  ClinicalCode `json:"medication"`
	Reason      string       `json:"reason"`
	Severity    string       `json:"severity"`   // absolute, relative, moderate
	BlockType   string       `json:"block_type"` // contraindication, interaction, allergy, age
	KBSource    string       `json:"kb_source"`
	RuleID      string       `json:"rule_id"`
	IsHardBlock bool         `json:"is_hard_block"` // True if this is also a hard block
}

// =============================================================================
// NEW: Transaction State Management (minimal new code for V3)
// =============================================================================

// TransactionState represents the state of a medication transaction
type TransactionState string

const (
	StateCreated    TransactionState = "CREATED"
	StateValidating TransactionState = "VALIDATING"
	StateValidated  TransactionState = "VALIDATED"
	StateBlocked    TransactionState = "BLOCKED"
	StateOverriding TransactionState = "OVERRIDING"
	StateCommitting TransactionState = "COMMITTING"
	StateCommitted  TransactionState = "COMMITTED"
	StateFailed     TransactionState = "FAILED"
	StateExpired    TransactionState = "EXPIRED"
)

// Transaction represents a medication transaction in the KB-19 Transaction Authority
type Transaction struct {
	ID                 uuid.UUID        `json:"id"`
	PatientID          uuid.UUID        `json:"patient_id"`
	EncounterID        uuid.UUID        `json:"encounter_id"`
	ProposedMedication ClinicalCode     `json:"proposed_medication"`
	CurrentMedications []ClinicalCode   `json:"current_medications,omitempty"` // V3: Store current meds for DDI checking
	ProviderID         string           `json:"provider_id"`
	State              TransactionState `json:"state"`

	// Populated by moved validation functions
	HardBlocks   []HardBlock        `json:"hard_blocks,omitempty"`
	ExcludedDrugs []ExcludedDrugInfo `json:"excluded_drugs,omitempty"`
	Disposition  DispositionCode    `json:"disposition"`

	// Populated by moved governance functions
	GovernanceEvents []GovernanceEvent  `json:"governance_events,omitempty"`
	GeneratedTasks   []GeneratedTask    `json:"generated_tasks,omitempty"`
	AuditTrail       *AuditTrailSummary `json:"audit_trail,omitempty"`

	// Lifecycle timestamps
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ExpiresAt   time.Time  `json:"expires_at"`
	CommittedAt *time.Time `json:"committed_at,omitempty"`
	CommittedBy string     `json:"committed_by,omitempty"`

	// Override tracking
	OverrideDecisions []OverrideDecision `json:"override_decisions,omitempty"`
}

// OverrideDecision represents a provider's decision to override a block
type OverrideDecision struct {
	BlockID        uuid.UUID `json:"block_id"`
	OverrideType   string    `json:"override_type"`   // ATTENDING_APPROVAL, PHARMACIST_REVIEW, etc.
	Reason         string    `json:"reason"`
	ApproverID     string    `json:"approver_id,omitempty"`
	AcknowledgedAt time.Time `json:"acknowledged_at"`
	KB18RecordID   string    `json:"kb18_record_id"` // Identity binding via KB-18
}

// AuditEvent represents a single event in the transaction audit trail
type AuditEvent struct {
	Timestamp time.Time `json:"timestamp"`
	EventType string    `json:"event_type"`
	Actor     string    `json:"actor"`
	Details   string    `json:"details"`
	Hash      string    `json:"hash,omitempty"` // For immutable audit chain
}

// =============================================================================
// V3 RISK PROFILE TYPES
// Med-Advisor returns risk profiles, KB-19 converts to governance decisions
// =============================================================================

// RiskProfile represents risk assessments from Med-Advisor (V3 Judge)
// KB-19 (Clerk) uses this to make governance decisions (HardBlocks, etc.)
type RiskProfile struct {
	RequestID           string               `json:"request_id"`
	PatientID           uuid.UUID            `json:"patient_id"`
	EncounterID         uuid.UUID            `json:"encounter_id"`
	CalculatedAt        time.Time            `json:"calculated_at"`
	MedicationRisks     []MedicationRisk     `json:"medication_risks"`
	DDIRisks            []DDIRisk            `json:"ddi_risks,omitempty"`
	LabRisks            []LabRisk            `json:"lab_risks,omitempty"`
	AllergyRisks        []AllergyRisk        `json:"allergy_risks,omitempty"`
	DoseRecommendations []DoseRecommendation `json:"dose_recommendations,omitempty"`
	KBSourcesUsed       []string             `json:"kb_sources_used"`
	ProcessingMs        int64                `json:"processing_ms"`
}

// MedicationRisk represents aggregate risk for a single medication
type MedicationRisk struct {
	RxNormCode      string       `json:"rxnorm_code"`
	DrugName        string       `json:"drug_name"`
	OverallRisk     float64      `json:"overall_risk"`
	RiskCategory    string       `json:"risk_category"` // LOW, MODERATE, HIGH, CRITICAL
	RiskFactors     []RiskFactor `json:"risk_factors"`
	IsHighAlert     bool         `json:"is_high_alert"`
	HasBlackBoxWarn bool         `json:"has_black_box_warning"`
}

// RiskFactor represents a single contributing risk factor
type RiskFactor struct {
	Type        string `json:"type"`        // DDI, LAB, ALLERGY, RENAL, HEPATIC, AGE, PREGNANCY
	Severity    string `json:"severity"`    // mild, moderate, severe, life-threatening
	Description string `json:"description"`
	KBSource    string `json:"kb_source"`
	RuleID      string `json:"rule_id"`
}

// DDIRisk represents a drug-drug interaction risk
type DDIRisk struct {
	Drug1Code          string `json:"drug1_code"`
	Drug1Name          string `json:"drug1_name"`
	Drug2Code          string `json:"drug2_code"`
	Drug2Name          string `json:"drug2_name"`
	Severity           string `json:"severity"`
	InteractionType    string `json:"interaction_type"`
	Mechanism          string `json:"mechanism"`
	ClinicalEffect     string `json:"clinical_effect"`
	ManagementStrategy string `json:"management_strategy"`
	EvidenceLevel      string `json:"evidence_level"`
	KBSource           string `json:"kb_source"`
	RuleID             string `json:"rule_id"`
}

// LabRisk represents a lab-based contraindication risk
type LabRisk struct {
	RxNormCode     string  `json:"rxnorm_code"`
	DrugName       string  `json:"drug_name"`
	LOINCCode      string  `json:"loinc_code"`
	LabName        string  `json:"lab_name"`
	CurrentValue   float64 `json:"current_value"`
	ThresholdValue float64 `json:"threshold_value"`
	ThresholdOp    string  `json:"threshold_op"`
	Severity       string  `json:"severity"`
	ClinicalRisk   string  `json:"clinical_risk"`
	Recommendation string  `json:"recommendation"`
	KBSource       string  `json:"kb_source"`
	RuleID         string  `json:"rule_id"`
}

// AllergyRisk represents an allergy-based risk
type AllergyRisk struct {
	RxNormCode      string `json:"rxnorm_code"`
	DrugName        string `json:"drug_name"`
	AllergenCode    string `json:"allergen_code"`
	AllergenName    string `json:"allergen_name"`
	IsCrossReactive bool   `json:"is_cross_reactive"`
	Severity        string `json:"severity"`
	ReactionType    string `json:"reaction_type"`
	KBSource        string `json:"kb_source"`
	RuleID          string `json:"rule_id"`
}

// DoseRecommendation represents a dosing adjustment recommendation
type DoseRecommendation struct {
	RxNormCode      string  `json:"rxnorm_code"`
	DrugName        string  `json:"drug_name"`
	OriginalDose    float64 `json:"original_dose"`
	AdjustedDose    float64 `json:"adjusted_dose"`
	DoseUnit        string  `json:"dose_unit"`
	AdjustmentType  string  `json:"adjustment_type"` // RENAL, HEPATIC, AGE, WEIGHT
	AdjustmentRatio float64 `json:"adjustment_ratio"`
	Reason          string  `json:"reason"`
	KBSource        string  `json:"kb_source"`
	RuleID          string  `json:"rule_id"`
}

// =============================================================================
// RISK SEVERITY THRESHOLDS
// KB-19's policy for converting risk levels to governance decisions
// =============================================================================

// RiskThresholds defines when risks become hard blocks
type RiskThresholds struct {
	DDISeverities     []string // severities that trigger hard blocks
	LabSeverities     []string // severities that trigger hard blocks
	AllergySeverities []string // severities that trigger hard blocks
	OverallRiskCutoff float64  // overall risk score that triggers hard block (0.0-1.0)
}

// DefaultRiskThresholds returns conservative default thresholds
// DDI severity levels follow standard terminology: contraindicated > major > moderate > minor
// KB-5 returns "major" for serious interactions like Warfarin + Aspirin
func DefaultRiskThresholds() RiskThresholds {
	return RiskThresholds{
		DDISeverities:     []string{"contraindicated", "major", "severe"}, // Added "major" - standard DDI terminology from KB-5
		LabSeverities:     []string{"contraindicated", "severe", "life-threatening"},
		AllergySeverities: []string{"severe", "life-threatening"},
		OverallRiskCutoff: 0.8, // CRITICAL category
	}
}
