// Package types defines the core types for KB-18 Governance Engine.
// These types implement the clinical governance enforcement platform that
// answers four critical questions for every clinical decision:
// 1. What dose SHOULD this patient be receiving?
// 2. Is the prescribed dose SAFE?
// 3. Is the institution following program rules?
// 4. Who is accountable if the patient is harmed?
package types

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// ENFORCEMENT LEVELS
// =============================================================================

// EnforcementLevel defines governance enforcement behavior
type EnforcementLevel string

const (
	// EnforcementIgnore logs only, no action required
	EnforcementIgnore EnforcementLevel = "IGNORE"
	// EnforcementNotify notifies but allows the action to proceed
	EnforcementNotify EnforcementLevel = "NOTIFY"
	// EnforcementWarnAcknowledge warns and requires acknowledgment
	EnforcementWarnAcknowledge EnforcementLevel = "WARN_ACKNOWLEDGE"
	// EnforcementHardBlock blocks with no override possible
	EnforcementHardBlock EnforcementLevel = "HARD_BLOCK"
	// EnforcementHardBlockWithOverride blocks but governance can override
	EnforcementHardBlockWithOverride EnforcementLevel = "HARD_BLOCK_WITH_OVERRIDE"
	// EnforcementMandatoryEscalation blocks and requires immediate escalation
	EnforcementMandatoryEscalation EnforcementLevel = "MANDATORY_ESCALATION"
)

// CanOverride returns true if this enforcement level allows override
func (e EnforcementLevel) CanOverride() bool {
	switch e {
	case EnforcementWarnAcknowledge, EnforcementHardBlockWithOverride:
		return true
	default:
		return false
	}
}

// IsBlocking returns true if this enforcement level blocks the action
func (e EnforcementLevel) IsBlocking() bool {
	switch e {
	case EnforcementHardBlock, EnforcementHardBlockWithOverride, EnforcementMandatoryEscalation:
		return true
	default:
		return false
	}
}

// RequiresAcknowledgment returns true if acknowledgment is required
func (e EnforcementLevel) RequiresAcknowledgment() bool {
	return e == EnforcementWarnAcknowledge
}

// Priority returns numeric priority (higher = more severe)
func (e EnforcementLevel) Priority() int {
	priorities := map[EnforcementLevel]int{
		EnforcementIgnore:               0,
		EnforcementNotify:               1,
		EnforcementWarnAcknowledge:      2,
		EnforcementHardBlockWithOverride: 3,
		EnforcementMandatoryEscalation:  4,
		EnforcementHardBlock:            5,
	}
	return priorities[e]
}

// =============================================================================
// SEVERITY LEVELS
// =============================================================================

// Severity defines clinical risk severity
type Severity string

const (
	SeverityInfo     Severity = "INFO"
	SeverityLow      Severity = "LOW"
	SeverityModerate Severity = "MODERATE"
	SeverityHigh     Severity = "HIGH"
	SeverityCritical Severity = "CRITICAL"
	SeverityFatal    Severity = "FATAL"
)

// Priority returns numeric priority for severity
func (s Severity) Priority() int {
	priorities := map[Severity]int{
		SeverityInfo:     0,
		SeverityLow:      1,
		SeverityModerate: 2,
		SeverityHigh:     3,
		SeverityCritical: 4,
		SeverityFatal:    5,
	}
	return priorities[s]
}

// =============================================================================
// EVALUATION OUTCOME
// =============================================================================

// Outcome represents the final decision outcome
type Outcome string

const (
	OutcomeApproved          Outcome = "APPROVED"
	OutcomeApprovedWithWarns Outcome = "APPROVED_WITH_WARNINGS"
	OutcomeBlocked           Outcome = "BLOCKED"
	OutcomePendingOverride   Outcome = "PENDING_OVERRIDE"
	OutcomePendingAck        Outcome = "PENDING_ACKNOWLEDGMENT"
	OutcomeEscalated         Outcome = "ESCALATED"
)

// =============================================================================
// VIOLATION CATEGORY
// =============================================================================

// ViolationCategory categorizes the type of violation
type ViolationCategory string

const (
	ViolationContraindication   ViolationCategory = "CONTRAINDICATION"
	ViolationDoseExceeded       ViolationCategory = "DOSE_EXCEEDED"
	ViolationDrugInteraction    ViolationCategory = "DRUG_INTERACTION"
	ViolationAllergy            ViolationCategory = "ALLERGY"
	ViolationLabRequired        ViolationCategory = "LAB_REQUIRED"
	ViolationMonitoringRequired ViolationCategory = "MONITORING_REQUIRED"
	ViolationFormulary          ViolationCategory = "FORMULARY"
	ViolationProtocolDeviation  ViolationCategory = "PROTOCOL_DEVIATION"
	ViolationRenalDosing        ViolationCategory = "RENAL_DOSING"
	ViolationHepaticDosing      ViolationCategory = "HEPATIC_DOSING"
	ViolationPediatricSafety    ViolationCategory = "PEDIATRIC_SAFETY"
	ViolationGeriatricSafety    ViolationCategory = "GERIATRIC_SAFETY"
	ViolationPregnancySafety    ViolationCategory = "PREGNANCY_SAFETY"
	ViolationDuplicateTherapy   ViolationCategory = "DUPLICATE_THERAPY"
)

// =============================================================================
// PATIENT CONTEXT
// =============================================================================

// PatientContext contains patient information for governance evaluation
type PatientContext struct {
	PatientID           string               `json:"patientId"`
	Age                 int                  `json:"age"`
	Sex                 string               `json:"sex"`
	Weight              float64              `json:"weight,omitempty"`     // kg
	Height              float64              `json:"height,omitempty"`     // cm
	BSA                 float64              `json:"bsa,omitempty"`        // m²
	IsPregnant          bool                 `json:"isPregnant"`
	GestationalAge      int                  `json:"gestationalAge,omitempty"` // weeks
	IsLactating         bool                 `json:"isLactating"`
	RenalFunction       *RenalFunction       `json:"renalFunction,omitempty"`
	HepaticFunction     *HepaticFunction     `json:"hepaticFunction,omitempty"`
	Allergies           []Allergy            `json:"allergies,omitempty"`
	ActiveDiagnoses     []Diagnosis          `json:"activeDiagnoses,omitempty"`
	CurrentMedications  []Medication         `json:"currentMedications,omitempty"`
	RecentLabs          []LabResult          `json:"recentLabs,omitempty"`
	Vitals              *Vitals              `json:"vitals,omitempty"`
	RegistryMemberships []RegistryMembership `json:"registryMemberships,omitempty"`
}

// GetAgeBand returns the patient's age band for context-based rules
func (p *PatientContext) GetAgeBand() string {
	switch {
	case p.Age < 1:
		return "neonate"
	case p.Age < 2:
		return "infant"
	case p.Age < 12:
		return "pediatric"
	case p.Age < 18:
		return "adolescent"
	case p.Age < 65:
		return "adult"
	default:
		return "geriatric"
	}
}

// RenalFunction represents kidney function status
type RenalFunction struct {
	EGFR      float64 `json:"egfr"`      // mL/min/1.73m²
	Creatinine float64 `json:"creatinine"` // mg/dL
	CKDStage  string  `json:"ckdStage"`  // CKD_1, CKD_2, CKD_3A, CKD_3B, CKD_4, CKD_5, ESRD
	OnDialysis bool   `json:"onDialysis"`
}

// HepaticFunction represents liver function status
type HepaticFunction struct {
	ChildPughScore int    `json:"childPughScore"`
	ChildPughClass string `json:"childPughClass"` // A, B, C
	MELD           int    `json:"meld,omitempty"`
}

// Allergy represents a patient allergy
type Allergy struct {
	Substance    string `json:"substance"`
	SubstanceCode string `json:"substanceCode,omitempty"`
	Reaction     string `json:"reaction,omitempty"`
	Severity     string `json:"severity,omitempty"`
}

// Diagnosis represents an active diagnosis
type Diagnosis struct {
	Code        string `json:"code"`
	CodeSystem  string `json:"codeSystem"` // ICD10, SNOMED
	Description string `json:"description"`
	Status      string `json:"status"` // active, resolved, chronic
}

// Medication represents a current medication
type Medication struct {
	Code       string  `json:"code"`
	CodeSystem string  `json:"codeSystem"` // RxNorm, NDC
	Name       string  `json:"name"`
	DrugClass  string  `json:"drugClass,omitempty"`
	Dose       float64 `json:"dose,omitempty"`
	DoseUnit   string  `json:"doseUnit,omitempty"`
	Frequency  string  `json:"frequency,omitempty"`
	Route      string  `json:"route,omitempty"`
}

// LabResult represents a laboratory result
type LabResult struct {
	Code       string    `json:"code"`
	CodeSystem string    `json:"codeSystem"` // LOINC
	Name       string    `json:"name"`
	Value      float64   `json:"value"`
	Unit       string    `json:"unit"`
	Timestamp  time.Time `json:"timestamp"`
}

// Vitals represents patient vital signs
type Vitals struct {
	SystolicBP   int       `json:"systolicBp,omitempty"`
	DiastolicBP  int       `json:"diastolicBp,omitempty"`
	HeartRate    int       `json:"heartRate,omitempty"`
	Temperature  float64   `json:"temperature,omitempty"` // Celsius
	SpO2         int       `json:"spo2,omitempty"`
	Timestamp    time.Time `json:"timestamp,omitempty"`
}

// RegistryMembership represents enrollment in a clinical registry/program
type RegistryMembership struct {
	RegistryCode string    `json:"registryCode"`
	RegistryName string    `json:"registryName,omitempty"`
	Status       string    `json:"status"` // ACTIVE, INACTIVE, PENDING
	EnrolledAt   time.Time `json:"enrolledAt,omitempty"`
}

// =============================================================================
// MEDICATION ORDER
// =============================================================================

// MedicationOrder represents a medication order to evaluate
type MedicationOrder struct {
	OrderID        string    `json:"orderId,omitempty"`
	MedicationCode string    `json:"medicationCode"`
	MedicationName string    `json:"medicationName"`
	DrugClass      string    `json:"drugClass,omitempty"`
	Dose           float64   `json:"dose"`
	DoseUnit       string    `json:"doseUnit"`
	Frequency      string    `json:"frequency"`
	Route          string    `json:"route"`
	Duration       string    `json:"duration,omitempty"`
	Indication     string    `json:"indication,omitempty"`
	OrderedAt      time.Time `json:"orderedAt,omitempty"`
}

// =============================================================================
// EVALUATION REQUEST & RESPONSE
// =============================================================================

// Evaluation type constants
const (
	EvalTypeMedicationOrder    = "medication"
	EvalTypeProtocolCompliance = "protocol"
	EvalTypeAudit              = "audit"
)

// EvaluationRequest is the request to evaluate governance rules
type EvaluationRequest struct {
	RequestID       string           `json:"requestId,omitempty"`
	PatientID       string           `json:"patientId"`
	PatientContext  *PatientContext  `json:"patientContext"`
	Order           *MedicationOrder `json:"order,omitempty"`
	MedicationOrder *MedicationOrder `json:"medicationOrder,omitempty"` // Alias for compatibility
	EvaluationType  string           `json:"evaluationType"`            // medication, protocol, audit
	RequestorID     string           `json:"requestorId"`
	RequestorRole   string           `json:"requestorRole"`
	FacilityID      string           `json:"facilityId"`
	Timestamp       time.Time        `json:"timestamp,omitempty"`
}

// EvaluationResponse is the governance evaluation result
type EvaluationResponse struct {
	RequestID          string              `json:"requestId"`
	Outcome            Outcome             `json:"outcome"`
	IsApproved         bool                `json:"isApproved"`
	HasViolations      bool                `json:"hasViolations"`
	HighestSeverity    Severity            `json:"highestSeverity,omitempty"`
	Violations         []Violation         `json:"violations,omitempty"`
	Recommendations    []Recommendation    `json:"recommendations,omitempty"`
	AccountableParties []AccountableParty  `json:"accountableParties,omitempty"`
	NextSteps          []string            `json:"nextSteps,omitempty"`
	EvidenceTrail      *EvidenceTrail      `json:"evidenceTrail"`
	ProgramsEvaluated  []string            `json:"programsEvaluated"`
	EvaluatedAt        time.Time           `json:"evaluatedAt"`
	ExpiresAt          time.Time           `json:"expiresAt,omitempty"`
}

// Violation represents a governance rule violation
type Violation struct {
	ID               string            `json:"id"`
	RuleID           string            `json:"ruleId"`
	RuleName         string            `json:"ruleName"`
	ProgramCode      string            `json:"programCode"`
	Category         ViolationCategory `json:"category"`
	Severity         Severity          `json:"severity"`
	EnforcementLevel EnforcementLevel  `json:"enforcementLevel"`
	Description      string            `json:"description"`
	ClinicalRisk     string            `json:"clinicalRisk"`
	EvidenceLevel    string            `json:"evidenceLevel,omitempty"` // A, B, C, D, Expert
	References       []string          `json:"references,omitempty"`
	CanOverride      bool              `json:"canOverride"`
	RequiresAck      bool              `json:"requiresAcknowledgment"`
	ConditionsMet    []ConditionResult `json:"conditionsMet,omitempty"`
}

// ConditionResult shows which conditions were evaluated
type ConditionResult struct {
	ConditionType  string `json:"conditionType"`
	Expression     string `json:"expression"`
	WasMet         bool   `json:"wasMet"`
	ActualValue    string `json:"actualValue,omitempty"`
	ExpectedValue  string `json:"expectedValue,omitempty"`
}

// Recommendation provides guidance for addressing violations
type Recommendation struct {
	Type        string `json:"type"` // alternative, monitoring, dosing, consult
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    int    `json:"priority,omitempty"`
}

// AccountableParty identifies who is responsible
type AccountableParty struct {
	Role           string `json:"role"`
	Accountability string `json:"accountability"`
	EscalationOrder int   `json:"escalationOrder,omitempty"`
}

// =============================================================================
// EVIDENCE TRAIL
// =============================================================================

// EvidenceTrail provides immutable audit record
type EvidenceTrail struct {
	TrailID           string          `json:"trailId"`
	Timestamp         time.Time       `json:"timestamp"`
	PatientSnapshot   json.RawMessage `json:"patientSnapshot,omitempty"`
	OrderSnapshot     json.RawMessage `json:"orderSnapshot,omitempty"`
	ProgramsEvaluated []string        `json:"programsEvaluated"`
	RulesApplied      []RuleResult    `json:"rulesApplied"`
	FinalDecision     Outcome         `json:"finalDecision"`
	DecisionRationale string          `json:"decisionRationale"`
	RequestedBy       string          `json:"requestedBy"`
	EvaluatedBy       string          `json:"evaluatedBy"`
	Hash              string          `json:"hash"`
	PreviousHash      string          `json:"previousHash,omitempty"`
	IsImmutable       bool            `json:"isImmutable"`
}

// RuleResult records individual rule evaluation
type RuleResult struct {
	RuleID         string `json:"ruleId"`
	RuleName       string `json:"ruleName"`
	WasEvaluated   bool   `json:"wasEvaluated"`
	WasTriggered   bool   `json:"wasTriggered"`
	InputData      string `json:"inputData,omitempty"`
	OutputDecision string `json:"outputDecision"`
}

// GenerateHash creates a SHA-256 hash of the evidence trail
func (e *EvidenceTrail) GenerateHash() string {
	data, _ := json.Marshal(struct {
		TrailID           string
		Timestamp         time.Time
		ProgramsEvaluated []string
		RulesApplied      []RuleResult
		FinalDecision     Outcome
		RequestedBy       string
		PreviousHash      string
	}{
		TrailID:           e.TrailID,
		Timestamp:         e.Timestamp,
		ProgramsEvaluated: e.ProgramsEvaluated,
		RulesApplied:      e.RulesApplied,
		FinalDecision:     e.FinalDecision,
		RequestedBy:       e.RequestedBy,
		PreviousHash:      e.PreviousHash,
	})
	hash := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(hash[:])
}

// =============================================================================
// OVERRIDE MANAGEMENT
// =============================================================================

// OverrideStatus represents override request status
type OverrideStatus string

const (
	OverrideStatusPending  OverrideStatus = "PENDING"
	OverrideStatusApproved OverrideStatus = "APPROVED"
	OverrideStatusDenied   OverrideStatus = "DENIED"
	OverrideStatusExpired  OverrideStatus = "EXPIRED"
)

// OverrideRequest represents a request to override a violation
type OverrideRequest struct {
	ID                    string         `json:"id"`
	ViolationID           string         `json:"violation_id"`
	RequestID             string         `json:"request_id"`
	PatientID             string         `json:"patient_id"`
	RequestorID           string         `json:"requestor_id"`
	RequestorRole         string         `json:"requestor_role"`
	RuleCode              string         `json:"rule_code"`
	Reason                string         `json:"reason"`
	ClinicalJustification string         `json:"clinical_justification"`
	RiskAccepted          bool           `json:"risk_accepted"`
	Status                OverrideStatus `json:"status"`
	RequestedAt           time.Time      `json:"requested_at"`
	ApprovedBy            string         `json:"approved_by,omitempty"`
	ApprovedAt            time.Time      `json:"approved_at,omitempty"`
	DeniedBy              string         `json:"denied_by,omitempty"`
	DeniedAt              time.Time      `json:"denied_at,omitempty"`
	DenialReason          string         `json:"denial_reason,omitempty"`
	ExpiresAt             time.Time      `json:"expires_at,omitempty"`
}

// =============================================================================
// ACKNOWLEDGMENT
// =============================================================================

// Acknowledgment records user acknowledgment of a warning
type Acknowledgment struct {
	ID             string    `json:"id"`
	ViolationID    string    `json:"violation_id"`
	RequestID      string    `json:"request_id"`
	PatientID      string    `json:"patient_id"`
	UserID         string    `json:"user_id"`
	UserRole       string    `json:"user_role"`
	RuleCode       string    `json:"rule_code"`
	Timestamp      time.Time `json:"timestamp"`
	Statement      string    `json:"statement"`
	RiskUnderstood bool      `json:"risk_understood"`
	Comments       string    `json:"comments,omitempty"`
}

// =============================================================================
// ESCALATION
// =============================================================================

// EscalationStatus represents escalation status
type EscalationStatus string

const (
	EscalationStatusOpen         EscalationStatus = "OPEN"
	EscalationStatusAcknowledged EscalationStatus = "ACKNOWLEDGED"
	EscalationStatusResolved     EscalationStatus = "RESOLVED"
	EscalationStatusClosed       EscalationStatus = "CLOSED"
)

// Escalation represents a mandatory escalation
type Escalation struct {
	ID             string           `json:"id"`
	ViolationID    string           `json:"violation_id"`
	RequestID      string           `json:"request_id"`
	PatientID      string           `json:"patient_id"`
	RequestorID    string           `json:"requestor_id"`
	Level          string           `json:"level"`
	Severity       Severity         `json:"severity"`
	Reason         string           `json:"reason"`
	Status         EscalationStatus `json:"status"`
	CurrentLevel   int              `json:"current_level"`
	EscalationPath []string         `json:"escalation_path"`
	CreatedAt      time.Time        `json:"created_at"`
	AcknowledgedBy string           `json:"acknowledged_by,omitempty"`
	AcknowledgedAt time.Time        `json:"acknowledged_at,omitempty"`
	ResolvedBy     string           `json:"resolved_by,omitempty"`
	ResolvedAt     time.Time        `json:"resolved_at,omitempty"`
	Resolution     string           `json:"resolution,omitempty"`
}

// =============================================================================
// OVERRIDE PATTERN MONITORING
// =============================================================================

// OverridePattern tracks override patterns for a user/rule combination
type OverridePattern struct {
	RequestorID  string    `json:"requestor_id"`
	RuleCode     string    `json:"rule_code"`
	Count24h     int       `json:"count_24h"`
	Count7d      int       `json:"count_7d"`
	Flagged      bool      `json:"flagged"`
	FlagReason   string    `json:"flag_reason,omitempty"`
	LastRequest  time.Time `json:"last_request"`
}

// =============================================================================
// ENGINE STATISTICS
// =============================================================================

// EngineStats provides governance engine statistics
type EngineStats struct {
	TotalEvaluations   int64            `json:"total_evaluations"`
	TotalViolations    int64            `json:"total_violations"`
	TotalBlocked       int64            `json:"total_blocked"`
	TotalAllowed       int64            `json:"total_allowed"`
	ProgramsEvaluated  int64            `json:"programs_evaluated"`
	RulesEvaluated     int64            `json:"rules_evaluated"`
	OverridesRequested int64            `json:"overrides_requested"`
	OverridesApproved  int64            `json:"overrides_approved"`
	OverridesDenied    int64            `json:"overrides_denied"`
	EscalationsCreated int64            `json:"escalations_created"`
	ByProgram          map[string]int64 `json:"by_program"`
	BySeverity         map[string]int64 `json:"by_severity"`
	ByCategory         map[string]int64 `json:"by_category"`
	AvgEvaluationTime  time.Duration    `json:"avg_evaluation_time"`
	LastEvaluationTime time.Time        `json:"last_evaluation_time"`
	Since              time.Time        `json:"since"`
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// NewUUID generates a new UUID string
func NewUUID() string {
	return uuid.New().String()
}

// NewTrailID generates a new evidence trail ID
func NewTrailID() string {
	return "trail-" + time.Now().Format("20060102150405") + "-" + NewUUID()[:8]
}

// NewViolationID generates a new violation ID
func NewViolationID() string {
	return "viol-" + NewUUID()[:12]
}

// GetEnforcementPriority returns numeric priority for enforcement level (for testing)
func GetEnforcementPriority(level EnforcementLevel) int {
	priorities := map[EnforcementLevel]int{
		EnforcementIgnore:               0,
		EnforcementNotify:               1,
		EnforcementWarnAcknowledge:      2,
		EnforcementHardBlock:            3,
		EnforcementHardBlockWithOverride: 4,
		EnforcementMandatoryEscalation:  5,
	}
	if p, ok := priorities[level]; ok {
		return p
	}
	return -1
}

// GetSeverityPriority returns numeric priority for severity level (for testing)
func GetSeverityPriority(severity Severity) int {
	priorities := map[Severity]int{
		SeverityInfo:     0,
		SeverityLow:      1,
		SeverityModerate: 2,
		SeverityHigh:     3,
		SeverityCritical: 4,
		SeverityFatal:    5,
	}
	if p, ok := priorities[severity]; ok {
		return p
	}
	return -1
}
