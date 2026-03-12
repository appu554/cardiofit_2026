package entities

import (
	"time"
	"github.com/google/uuid"
)

// MedicationProposal represents a clinical medication proposal
type MedicationProposal struct {
	ID                    uuid.UUID                `json:"id" db:"id"`
	PatientID             uuid.UUID                `json:"patient_id" db:"patient_id"`
	ProtocolID            string                   `json:"protocol_id" db:"protocol_id"`
	Indication            string                   `json:"indication" db:"indication"`
	Status                ProposalStatus           `json:"status" db:"status"`
	ClinicalContext       *ClinicalContext         `json:"clinical_context" db:"clinical_context"`
	MedicationDetails     *MedicationDetails       `json:"medication_details" db:"medication_details"`
	DosageRecommendations []DosageRecommendation   `json:"dosage_recommendations" db:"dosage_recommendations"`
	SafetyConstraints     []SafetyConstraint       `json:"safety_constraints" db:"safety_constraints"`
	SnapshotID            uuid.UUID                `json:"snapshot_id" db:"snapshot_id"`
	CreatedAt             time.Time                `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time                `json:"updated_at" db:"updated_at"`
	CreatedBy             string                   `json:"created_by" db:"created_by"`
	ValidatedBy           *string                  `json:"validated_by,omitempty" db:"validated_by"`
	ValidationTimestamp   *time.Time               `json:"validation_timestamp,omitempty" db:"validation_timestamp"`
}

// ProposalStatus represents the status of a medication proposal
type ProposalStatus string

const (
	ProposalStatusDraft     ProposalStatus = "draft"
	ProposalStatusProposed  ProposalStatus = "proposed"
	ProposalStatusValidated ProposalStatus = "validated"
	ProposalStatusRejected  ProposalStatus = "rejected"
	ProposalStatusCommitted ProposalStatus = "committed"
	ProposalStatusExpired   ProposalStatus = "expired"
)

// ClinicalContext contains patient clinical information for medication calculations
type ClinicalContext struct {
	PatientID      uuid.UUID            `json:"patient_id"`
	WeightKg       *float64             `json:"weight_kg,omitempty"`
	HeightCm       *float64             `json:"height_cm,omitempty"`
	AgeYears       int                  `json:"age_years"`
	Gender         string               `json:"gender"`
	BSAm2          *float64             `json:"bsa_m2,omitempty"`
	CreatinineMgdL *float64             `json:"creatinine_mg_dl,omitempty"`
	eGFR           *float64             `json:"egfr,omitempty"`
	Allergies      []string             `json:"allergies,omitempty"`
	Conditions     []string             `json:"conditions,omitempty"`
	Medications    []CurrentMedication  `json:"current_medications,omitempty"`
	LabValues      map[string]LabValue  `json:"lab_values,omitempty"`
}

// CurrentMedication represents a medication the patient is currently taking
type CurrentMedication struct {
	MedicationName string    `json:"medication_name"`
	DoseMg         float64   `json:"dose_mg"`
	Frequency      string    `json:"frequency"`
	StartDate      time.Time `json:"start_date"`
	Route          string    `json:"route"`
}

// LabValue represents a laboratory test result
type LabValue struct {
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
	Timestamp time.Time `json:"timestamp"`
	Reference string    `json:"reference_range"`
}

// MedicationDetails contains detailed medication information
type MedicationDetails struct {
	DrugName        string                 `json:"drug_name"`
	GenericName     string                 `json:"generic_name"`
	BrandName       string                 `json:"brand_name,omitempty"`
	DrugClass       string                 `json:"drug_class"`
	Mechanism       string                 `json:"mechanism"`
	Indication      string                 `json:"indication"`
	Contraindications []string             `json:"contraindications,omitempty"`
	Interactions    []DrugInteraction      `json:"interactions,omitempty"`
	FormulationTypes []FormulationType     `json:"formulation_types"`
	TherapeuticClass string                `json:"therapeutic_class"`
	PharmacologyProfile *PharmacologyProfile `json:"pharmacology_profile,omitempty"`
}

// DrugInteraction represents a drug-drug interaction
type DrugInteraction struct {
	InteractingDrug string             `json:"interacting_drug"`
	Severity        InteractionSeverity `json:"severity"`
	Description     string             `json:"description"`
	Management      string             `json:"management"`
}

// InteractionSeverity represents the severity of drug interactions
type InteractionSeverity string

const (
	SeverityMinor     InteractionSeverity = "minor"
	SeverityModerate  InteractionSeverity = "moderate"
	SeverityMajor     InteractionSeverity = "major"
	SeverityContraindicated InteractionSeverity = "contraindicated"
)

// FormulationType represents medication formulation options
type FormulationType struct {
	Form         string   `json:"form"`          // tablet, capsule, injection, etc.
	Strengths    []string `json:"strengths"`     // available strengths
	Route        string   `json:"route"`         // oral, IV, IM, etc.
	Availability string   `json:"availability"`  // generic, brand-only, etc.
}

// PharmacologyProfile contains pharmacokinetic/pharmacodynamic information
type PharmacologyProfile struct {
	HalfLifeHours     *float64 `json:"half_life_hours,omitempty"`
	OnsetMinutes      *int     `json:"onset_minutes,omitempty"`
	PeakHours         *float64 `json:"peak_hours,omitempty"`
	DurationHours     *float64 `json:"duration_hours,omitempty"`
	Bioavailability   *float64 `json:"bioavailability,omitempty"`
	ProteinBinding    *float64 `json:"protein_binding,omitempty"`
	Metabolism        string   `json:"metabolism,omitempty"`
	Excretion         string   `json:"excretion,omitempty"`
	RenalAdjustment   bool     `json:"renal_adjustment"`
	HepaticAdjustment bool     `json:"hepatic_adjustment"`
}

// DosageRecommendation represents a calculated dosage recommendation
type DosageRecommendation struct {
	ID                  uuid.UUID              `json:"id"`
	RecommendationType  RecommendationType     `json:"recommendation_type"`
	DoseMg              float64                `json:"dose_mg"`
	FrequencyPerDay     int                    `json:"frequency_per_day"`
	Route               string                 `json:"route"`
	DurationDays        *int                   `json:"duration_days,omitempty"`
	MaxDoseMg           *float64               `json:"max_dose_mg,omitempty"`
	MinDoseMg           *float64               `json:"min_dose_mg,omitempty"`
	AdjustmentReason    string                 `json:"adjustment_reason,omitempty"`
	CalculationMethod   CalculationMethod      `json:"calculation_method"`
	ConfidenceScore     float64                `json:"confidence_score"`
	ClinicalNotes       string                 `json:"clinical_notes,omitempty"`
	MonitoringRequired  []MonitoringRequirement `json:"monitoring_required,omitempty"`
}

// RecommendationType represents different types of dosage recommendations
type RecommendationType string

const (
	RecommendationStarting    RecommendationType = "starting"
	RecommendationMaintenance RecommendationType = "maintenance"
	RecommendationLoading     RecommendationType = "loading"
	RecommendationAdjustment  RecommendationType = "adjustment"
	RecommendationTaper       RecommendationType = "taper"
)

// CalculationMethod represents the method used for dose calculation
type CalculationMethod string

const (
	MethodWeightBased   CalculationMethod = "weight_based"
	MethodBSABased      CalculationMethod = "bsa_based"
	MethodAUCBased      CalculationMethod = "auc_based"
	MethodFixed         CalculationMethod = "fixed"
	MethodRenalAdjusted CalculationMethod = "renal_adjusted"
	MethodAgeAdjusted   CalculationMethod = "age_adjusted"
	MethodCustom        CalculationMethod = "custom"
)

// MonitoringRequirement represents required clinical monitoring
type MonitoringRequirement struct {
	Parameter     string              `json:"parameter"`
	Frequency     MonitoringFrequency `json:"frequency"`
	TargetRange   *string             `json:"target_range,omitempty"`
	AlertThreshold *float64           `json:"alert_threshold,omitempty"`
	Notes         string              `json:"notes,omitempty"`
}

// MonitoringFrequency represents how often monitoring is required
type MonitoringFrequency string

const (
	FrequencyDaily    MonitoringFrequency = "daily"
	FrequencyWeekly   MonitoringFrequency = "weekly"
	FrequencyBiweekly MonitoringFrequency = "biweekly"
	FrequencyMonthly  MonitoringFrequency = "monthly"
	FrequencyPRN      MonitoringFrequency = "prn"
)

// SafetyConstraint represents clinical safety constraints
type SafetyConstraint struct {
	ID            uuid.UUID         `json:"id"`
	ConstraintType ConstraintType   `json:"constraint_type"`
	Severity      ConstraintSeverity `json:"severity"`
	Parameter     string            `json:"parameter"`
	Operator      string            `json:"operator"` // >, <, >=, <=, =, !=
	ThresholdValue float64          `json:"threshold_value"`
	Unit          string            `json:"unit"`
	Message       string            `json:"message"`
	Action        string            `json:"action"` // warn, block, adjust
	Source        string            `json:"source"` // guideline, interaction, allergy, etc.
}

// ConstraintType represents different types of safety constraints
type ConstraintType string

const (
	ConstraintDosage      ConstraintType = "dosage"
	ConstraintFrequency   ConstraintType = "frequency"
	ConstraintDuration    ConstraintType = "duration"
	ConstraintLab         ConstraintType = "lab_value"
	ConstraintAge         ConstraintType = "age"
	ConstraintWeight      ConstraintType = "weight"
	ConstraintRenal       ConstraintType = "renal_function"
	ConstraintHepatic     ConstraintType = "hepatic_function"
	ConstraintInteraction ConstraintType = "drug_interaction"
	ConstraintAllergy     ConstraintType = "allergy"
)

// ConstraintSeverity represents the severity of safety constraints
type ConstraintSeverity string

const (
	SeverityInfo     ConstraintSeverity = "info"
	SeverityWarning  ConstraintSeverity = "warning"
	SeverityError    ConstraintSeverity = "error"
	SeverityCritical ConstraintSeverity = "critical"
)

// Validate validates the medication proposal
func (m *MedicationProposal) Validate() error {
	if m.PatientID == uuid.Nil {
		return NewValidationError("patient_id is required")
	}

	if m.Indication == "" {
		return NewValidationError("indication is required")
	}

	if m.ClinicalContext == nil {
		return NewValidationError("clinical_context is required")
	}

	if m.MedicationDetails == nil {
		return NewValidationError("medication_details is required")
	}

	return nil
}

// IsExpired checks if the proposal has expired
func (m *MedicationProposal) IsExpired() bool {
	// Proposals expire after 24 hours for safety
	expiryTime := m.CreatedAt.Add(24 * time.Hour)
	return time.Now().After(expiryTime)
}

// CanTransitionTo checks if the proposal can transition to the given status
func (m *MedicationProposal) CanTransitionTo(newStatus ProposalStatus) bool {
	switch m.Status {
	case ProposalStatusDraft:
		return newStatus == ProposalStatusProposed || newStatus == ProposalStatusRejected
	case ProposalStatusProposed:
		return newStatus == ProposalStatusValidated || newStatus == ProposalStatusRejected
	case ProposalStatusValidated:
		return newStatus == ProposalStatusCommitted || newStatus == ProposalStatusRejected
	case ProposalStatusRejected, ProposalStatusCommitted, ProposalStatusExpired:
		return false // Terminal states
	default:
		return false
	}
}

// ValidationError represents a domain validation error
type ValidationError struct {
	message string
}

func NewValidationError(message string) *ValidationError {
	return &ValidationError{message: message}
}

func (e *ValidationError) Error() string {
	return e.message
}