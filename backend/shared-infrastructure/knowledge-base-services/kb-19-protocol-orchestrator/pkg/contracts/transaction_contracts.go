// Package contracts provides transaction-specific API contracts for KB-19.
// These contracts define the Transaction Authority interface for medication safety.
// Part of V3 Architecture: KB-19 as Transaction Authority (Clerk)
package contracts

import (
	"time"

	"github.com/google/uuid"

	"kb-19-protocol-orchestrator/internal/transaction"
)

// =============================================================================
// TRANSACTION CREATE CONTRACTS
// =============================================================================

// CreateTransactionRequest is the request to create a new medication transaction.
// This is Step 1 of Calculate → Validate → Commit workflow.
type CreateTransactionRequest struct {
	PatientID   uuid.UUID `json:"patient_id" binding:"required"`
	EncounterID uuid.UUID `json:"encounter_id" binding:"required"`
	ProviderID  string    `json:"provider_id" binding:"required"`

	// The proposed medication (from Med-Advisor risk calculation)
	ProposedMedication ProposedMedicationInfo `json:"proposed_medication" binding:"required"`

	// Current medications for DDI checking
	CurrentMedications []MedicationInfo `json:"current_medications,omitempty"`

	// Patient lab values for lab-drug contraindication checking
	PatientLabs []LabValueInfo `json:"patient_labs,omitempty"`

	// Clinical snapshot ID (if using snapshot-based workflow)
	SnapshotID *uuid.UUID `json:"snapshot_id,omitempty"`

	// Request metadata
	RequestedBy string `json:"requested_by,omitempty"`
	RequestID   string `json:"request_id,omitempty"`
}

// ProposedMedicationInfo contains the proposed medication details
type ProposedMedicationInfo struct {
	RxNormCode    string   `json:"rxnorm_code" binding:"required"`
	DrugName      string   `json:"drug_name" binding:"required"`
	DrugClass     string   `json:"drug_class,omitempty"`
	DoseMg        float64  `json:"dose_mg,omitempty"`
	Unit          string   `json:"unit,omitempty"`
	Route         string   `json:"route,omitempty"`
	Frequency     string   `json:"frequency,omitempty"`
	Indication    string   `json:"indication,omitempty"`
	IsRenal       bool     `json:"is_renal_adjusted,omitempty"`
	IsHepatic     bool     `json:"is_hepatic_adjusted,omitempty"`
	RiskScore     float64  `json:"risk_score,omitempty"`      // From Med-Advisor
	RiskFactors   []string `json:"risk_factors,omitempty"`    // From Med-Advisor
	KBSourcesUsed []string `json:"kb_sources_used,omitempty"` // KB-1, KB-4, KB-5, etc.
}

// MedicationInfo contains current medication information for DDI checking
type MedicationInfo struct {
	RxNormCode string `json:"rxnorm_code" binding:"required"`
	DrugName   string `json:"drug_name" binding:"required"`
	DrugClass  string `json:"drug_class,omitempty"`
	DoseMg     float64 `json:"dose_mg,omitempty"`
	Route      string `json:"route,omitempty"`
	Frequency  string `json:"frequency,omitempty"`
}

// LabValueInfo contains patient lab value information
type LabValueInfo struct {
	LOINCCode      string      `json:"loinc_code" binding:"required"`
	TestName       string      `json:"test_name" binding:"required"`
	Value          interface{} `json:"value" binding:"required"`
	Unit           string      `json:"unit"`
	CollectedAt    time.Time   `json:"collected_at"`
	IsCritical     bool        `json:"is_critical,omitempty"`
	ReferenceRange string      `json:"reference_range,omitempty"`
}

// CreateTransactionResponse is the response from creating a transaction.
type CreateTransactionResponse struct {
	TransactionID uuid.UUID                  `json:"transaction_id"`
	State         transaction.TransactionState `json:"state"`
	CreatedAt     time.Time                  `json:"created_at"`

	// Initial safety assessment
	SafetyAssessment SafetyAssessmentSummary `json:"safety_assessment"`

	// If hard blocks exist, they are returned here
	HardBlocks []HardBlockSummary `json:"hard_blocks,omitempty"`

	// Recommended next action
	NextAction string `json:"next_action"` // "validate", "resolve_blocks", "abort"

	// Processing metadata
	ProcessingTimeMs int64 `json:"processing_time_ms"`
}

// SafetyAssessmentSummary provides a high-level safety assessment
type SafetyAssessmentSummary struct {
	IsBlocked          bool   `json:"is_blocked"`
	BlockCount         int    `json:"block_count"`
	DDICount           int    `json:"ddi_count"`
	LabContraindCount  int    `json:"lab_contraindication_count"`
	HighestSeverity    string `json:"highest_severity"` // critical, severe, moderate
	RequiresOverride   bool   `json:"requires_override"`
	RecommendedAction  string `json:"recommended_action"`
}

// HardBlockSummary is a summary of a hard block for API responses
type HardBlockSummary struct {
	ID           uuid.UUID `json:"id"`
	BlockType    string    `json:"block_type"`    // DDI, LAB_CONTRAINDICATION, ALLERGY, etc.
	Severity     string    `json:"severity"`      // critical, severe, moderate
	Medication   string    `json:"medication"`    // The blocked medication
	TriggerCode  string    `json:"trigger_code"`  // The triggering code (drug or lab)
	TriggerName  string    `json:"trigger_name"`  // Human-readable trigger name
	Reason       string    `json:"reason"`        // Clinical reason for block
	RequiresAck  bool      `json:"requires_ack"`  // Whether acknowledgment is required
	AckText      string    `json:"ack_text"`      // Text user must acknowledge
	KBSource     string    `json:"kb_source"`     // Which KB generated this (KB-5, KB-16)
	RuleID       string    `json:"rule_id"`       // Specific rule that triggered
}

// =============================================================================
// TRANSACTION VALIDATE CONTRACTS
// =============================================================================

// ValidateTransactionRequest is the request to validate a transaction.
// This is Step 2 of Calculate → Validate → Commit workflow.
type ValidateTransactionRequest struct {
	TransactionID uuid.UUID `json:"transaction_id" binding:"required"`

	// Current clinical snapshot to validate against
	CurrentSnapshot *ClinicalSnapshotInfo `json:"current_snapshot,omitempty"`

	// Overrides for any hard blocks (if provider has acknowledged)
	BlockOverrides []BlockOverrideInfo `json:"block_overrides,omitempty"`

	// Provider performing validation
	ValidatedBy string `json:"validated_by" binding:"required"`
}

// ClinicalSnapshotInfo contains clinical snapshot data for conflict detection
type ClinicalSnapshotInfo struct {
	SnapshotID uuid.UUID `json:"snapshot_id"`
	PatientID  uuid.UUID `json:"patient_id"`
	CreatedAt  time.Time `json:"created_at"`

	// Clinical data for conflict detection
	LabResults   []LabValueInfo   `json:"lab_results,omitempty"`
	Medications  []MedicationInfo `json:"medications,omitempty"`
	Conditions   []ConditionInfo  `json:"conditions,omitempty"`
	Allergies    []AllergyInfo    `json:"allergies,omitempty"`
	Demographics DemographicInfo  `json:"demographics"`
}

// ConditionInfo contains patient condition information
type ConditionInfo struct {
	ICD10Code     string `json:"icd10_code"`
	SNOMEDCT      string `json:"snomed_ct"`
	ConditionName string `json:"condition_name"`
	Status        string `json:"status"` // active, resolved, etc.
}

// AllergyInfo contains patient allergy information
type AllergyInfo struct {
	Allergen     string `json:"allergen"`
	AllergenType string `json:"allergen_type"` // drug, food, environmental
	Severity     string `json:"severity"`
	Status       string `json:"status"` // active, inactive
}

// DemographicInfo contains patient demographic information
type DemographicInfo struct {
	WeightKg    *float64 `json:"weight_kg,omitempty"`
	HeightCm    *float64 `json:"height_cm,omitempty"`
	Age         int      `json:"age"`
	Gender      string   `json:"gender"`
	IsPregnant  bool     `json:"is_pregnant,omitempty"`
	IsLactating bool     `json:"is_lactating,omitempty"`
	EGFR        *float64 `json:"egfr,omitempty"` // Estimated GFR for renal function
}

// BlockOverrideInfo contains information about overriding a hard block
type BlockOverrideInfo struct {
	BlockID        uuid.UUID `json:"block_id" binding:"required"`
	AcknowledgedBy string    `json:"acknowledged_by" binding:"required"`
	AckTimestamp   time.Time `json:"ack_timestamp" binding:"required"`
	AckText        string    `json:"ack_text" binding:"required"` // Must match required ack text
	ClinicalReason string    `json:"clinical_reason,omitempty"`   // Provider's clinical justification
}

// ValidateTransactionResponse is the response from validating a transaction.
type ValidateTransactionResponse struct {
	TransactionID uuid.UUID                  `json:"transaction_id"`
	State         transaction.TransactionState `json:"state"`
	ValidatedAt   time.Time                  `json:"validated_at"`

	// Validation results
	IsValid            bool     `json:"is_valid"`
	ValidationErrors   []string `json:"validation_errors,omitempty"`
	ValidationWarnings []string `json:"validation_warnings,omitempty"`

	// Conflict detection results (if snapshot provided)
	ConflictDetection *ConflictDetectionResult `json:"conflict_detection,omitempty"`

	// Override status
	OverridesApplied []OverrideAppliedInfo `json:"overrides_applied,omitempty"`
	PendingBlocks    []HardBlockSummary    `json:"pending_blocks,omitempty"`

	// Recommended next action
	NextAction string `json:"next_action"` // "commit", "resolve_conflicts", "abort"

	// Processing metadata
	ProcessingTimeMs int64 `json:"processing_time_ms"`
}

// ConflictDetectionResult contains conflict detection results
type ConflictDetectionResult struct {
	HasHardConflicts      bool              `json:"has_hard_conflicts"`
	HasSoftConflicts      bool              `json:"has_soft_conflicts"`
	HardConflicts         []ConflictInfo    `json:"hard_conflicts,omitempty"`
	SoftConflicts         []ConflictInfo    `json:"soft_conflicts,omitempty"`
	Recommendation        string            `json:"recommendation"` // proceed, warn, abort
	RequiresRecalculation bool              `json:"requires_recalculation"`
}

// ConflictInfo contains information about a detected conflict
type ConflictInfo struct {
	Type        string      `json:"type"`        // lab, condition, allergy, medication, demographic
	Severity    string      `json:"severity"`    // hard, soft
	Field       string      `json:"field"`       // Which field changed
	OldValue    interface{} `json:"old_value,omitempty"`
	NewValue    interface{} `json:"new_value,omitempty"`
	Description string      `json:"description"` // Human-readable description
}

// OverrideAppliedInfo contains information about an applied override
type OverrideAppliedInfo struct {
	BlockID        uuid.UUID `json:"block_id"`
	BlockType      string    `json:"block_type"`
	AcknowledgedBy string    `json:"acknowledged_by"`
	AckTimestamp   time.Time `json:"ack_timestamp"`
}

// =============================================================================
// TRANSACTION COMMIT CONTRACTS
// =============================================================================

// CommitTransactionRequest is the request to commit a validated transaction.
// This is Step 3 of Calculate → Validate → Commit workflow.
type CommitTransactionRequest struct {
	TransactionID uuid.UUID `json:"transaction_id" binding:"required"`
	CommittedBy   string    `json:"committed_by" binding:"required"`

	// Final disposition (what action to take)
	Disposition string `json:"disposition" binding:"required"` // DISPENSE, HOLD, MODIFY, REJECT

	// If MODIFY, the modified medication details
	ModifiedMedication *ProposedMedicationInfo `json:"modified_medication,omitempty"`

	// Commit notes
	Notes string `json:"notes,omitempty"`
}

// CommitTransactionResponse is the response from committing a transaction.
type CommitTransactionResponse struct {
	TransactionID uuid.UUID                  `json:"transaction_id"`
	State         transaction.TransactionState `json:"state"`
	CommittedAt   time.Time                  `json:"committed_at"`

	// Final outcome
	Disposition     string `json:"disposition"`
	DispositionCode string `json:"disposition_code"`

	// Governance events generated
	GovernanceEvents []GovernanceEventSummary `json:"governance_events,omitempty"`

	// Audit trail
	AuditID     uuid.UUID `json:"audit_id"`
	AuditHash   string    `json:"audit_hash"` // Immutable hash for compliance

	// Generated tasks (if any)
	GeneratedTasks []GeneratedTaskSummary `json:"generated_tasks,omitempty"`

	// Processing metadata
	ProcessingTimeMs int64 `json:"processing_time_ms"`
}

// GovernanceEventSummary contains summary of a governance event
type GovernanceEventSummary struct {
	EventID     uuid.UUID `json:"event_id"`
	EventType   string    `json:"event_type"` // HARD_STOP_TRIGGERED, OVERRIDE_GRANTED, etc.
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
	Tier        int       `json:"tier"` // Governance tier (1-7)
}

// GeneratedTaskSummary contains summary of a generated follow-up task
type GeneratedTaskSummary struct {
	TaskID      uuid.UUID `json:"task_id"`
	TaskType    string    `json:"task_type"` // MONITOR_LABS, FOLLOW_UP, etc.
	Description string    `json:"description"`
	DueDate     time.Time `json:"due_date,omitempty"`
	AssignedTo  string    `json:"assigned_to,omitempty"`
	Priority    string    `json:"priority"`
}

// =============================================================================
// TRANSACTION QUERY CONTRACTS
// =============================================================================

// GetTransactionRequest is the request to get a transaction by ID.
type GetTransactionRequest struct {
	TransactionID uuid.UUID `json:"transaction_id" binding:"required"`
	IncludeAudit  bool      `json:"include_audit,omitempty"`
}

// GetTransactionResponse is the response containing transaction details.
type GetTransactionResponse struct {
	TransactionID uuid.UUID                  `json:"transaction_id"`
	PatientID     uuid.UUID                  `json:"patient_id"`
	EncounterID   uuid.UUID                  `json:"encounter_id"`
	State         transaction.TransactionState `json:"state"`
	CreatedAt     time.Time                  `json:"created_at"`
	UpdatedAt     time.Time                  `json:"updated_at"`

	// Medication details
	ProposedMedication ProposedMedicationInfo `json:"proposed_medication"`

	// Safety assessment
	SafetyAssessment SafetyAssessmentSummary `json:"safety_assessment"`
	HardBlocks       []HardBlockSummary      `json:"hard_blocks,omitempty"`

	// Validation status
	ValidationStatus *ValidationStatusInfo `json:"validation_status,omitempty"`

	// Commit status (if committed)
	CommitStatus *CommitStatusInfo `json:"commit_status,omitempty"`

	// Audit trail (if requested)
	AuditTrail []AuditEntryInfo `json:"audit_trail,omitempty"`
}

// ValidationStatusInfo contains validation status details
type ValidationStatusInfo struct {
	IsValidated        bool       `json:"is_validated"`
	ValidatedAt        *time.Time `json:"validated_at,omitempty"`
	ValidatedBy        string     `json:"validated_by,omitempty"`
	OverridesApplied   int        `json:"overrides_applied"`
	ConflictsDetected  int        `json:"conflicts_detected"`
}

// CommitStatusInfo contains commit status details
type CommitStatusInfo struct {
	IsCommitted     bool       `json:"is_committed"`
	CommittedAt     *time.Time `json:"committed_at,omitempty"`
	CommittedBy     string     `json:"committed_by,omitempty"`
	Disposition     string     `json:"disposition,omitempty"`
	DispositionCode string     `json:"disposition_code,omitempty"`
}

// AuditEntryInfo contains audit trail entry information
type AuditEntryInfo struct {
	EntryID   uuid.UUID `json:"entry_id"`
	Action    string    `json:"action"`
	Timestamp time.Time `json:"timestamp"`
	UserID    string    `json:"user_id"`
	Details   string    `json:"details"`
}

// =============================================================================
// TRANSACTION LIST CONTRACTS
// =============================================================================

// ListTransactionsRequest is the request to list transactions.
type ListTransactionsRequest struct {
	PatientID   *uuid.UUID `json:"patient_id,omitempty"`
	EncounterID *uuid.UUID `json:"encounter_id,omitempty"`
	State       string     `json:"state,omitempty"`   // Filter by state
	Since       *time.Time `json:"since,omitempty"`   // Transactions after this time
	Until       *time.Time `json:"until,omitempty"`   // Transactions before this time
	Limit       int        `json:"limit,omitempty"`   // Max results (default 50)
	Offset      int        `json:"offset,omitempty"`  // Pagination offset
}

// ListTransactionsResponse is the response containing transaction list.
type ListTransactionsResponse struct {
	Transactions []TransactionSummary `json:"transactions"`
	Total        int                  `json:"total"`
	HasMore      bool                 `json:"has_more"`
}

// TransactionSummary is a summary of a transaction for list responses.
type TransactionSummary struct {
	TransactionID      uuid.UUID                  `json:"transaction_id"`
	PatientID          uuid.UUID                  `json:"patient_id"`
	State              transaction.TransactionState `json:"state"`
	MedicationName     string                     `json:"medication_name"`
	MedicationCode     string                     `json:"medication_code"`
	HasBlocks          bool                       `json:"has_blocks"`
	BlockCount         int                        `json:"block_count"`
	CreatedAt          time.Time                  `json:"created_at"`
	UpdatedAt          time.Time                  `json:"updated_at"`
}
