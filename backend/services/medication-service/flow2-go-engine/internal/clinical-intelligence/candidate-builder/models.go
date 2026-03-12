package candidatebuilder

import (
	"context"
	"fmt"
	"time"
)

// ==================== Enhanced Domain Models ====================

// Drug represents medication metadata from a knowledge base with enhanced safety fields
type Drug struct {
	ID                    string   `json:"id"`
	Code                  string   `json:"code"`
	Name                  string   `json:"name"`
	GenericName           string   `json:"generic_name"`
	TherapeuticClasses    []string `json:"therapeutic_classes"`
	SubClasses            []string `json:"sub_classes"`
	PreferredRoute        string   `json:"preferred_route"`
	AvailableFormulations []string `json:"available_formulations"`
	EfficacyScore         float64  `json:"efficacy_score"`
	SafetyProfile         string   `json:"safety_profile"`

	// Enhanced Safety Fields
	ContraindicationCodes []string `json:"contraindication_codes"`
	AllergyCodes          []string `json:"allergy_codes"`
	RenalAdjustment       bool     `json:"renal_adjustment"`
	HepaticAdjustment     bool     `json:"hepatic_adjustment"`
	PregnancyCategory     string   `json:"pregnancy_category"` // A, B, C, D, X
	BlackBoxWarning       bool     `json:"black_box_warning"`

	// Legacy fields for backward compatibility
	Contraindications []string `json:"contraindications"`
	Indications       []string `json:"indications"`
	IsGeneric         bool     `json:"is_generic"`
	FDAApproved       bool     `json:"fda_approved"`
	ActiveIngredient  string   `json:"active_ingredient"`

	// Provenance tracking
	SourceKB        string `json:"source_kb"`
	SourceKBVersion string `json:"source_kb_version"`
}

// ActiveMedication represents a patient's current medication with enhanced tracking
type ActiveMedication struct {
	ID             string    `json:"id"`
	DrugID         string    `json:"drug_id"`
	MedicationCode string    `json:"medication_code"`
	Name           string    `json:"name"`
	Class          string    `json:"class"`
	Dose           string    `json:"dose"`
	Frequency      string    `json:"frequency"`
	Route          string    `json:"route"`
	StartDate      time.Time `json:"start_date"`
	IsActive       bool      `json:"is_active"`
}

// DDIInteraction represents a drug-drug interaction with enhanced severity typing
type DDIInteraction struct {
	ID          string      `json:"id"`
	DrugAID     string      `json:"drug_a_id"`
	DrugBID     string      `json:"drug_b_id"`
	Drug1       string      `json:"drug1"`       // Legacy field
	Drug2       string      `json:"drug2"`       // Legacy field
	Severity    DDISeverity `json:"severity"`
	Description string      `json:"description"`
	Mechanism   string      `json:"mechanism"`
	Management  string      `json:"management"`
	Evidence    string      `json:"evidence"`
}

// DDISeverity represents standardized DDI severity levels
type DDISeverity string

const (
	DDISeverityContraindicated DDISeverity = "Contraindicated"
	DDISeverityMajor           DDISeverity = "Major"
	DDISeverityModerate        DDISeverity = "Moderate"
	DDISeverityMinor           DDISeverity = "Minor"
)

// Legacy type alias for backward compatibility
type DrugInteraction = DDIInteraction

// MedicationProposal represents a candidate medication that has passed safety filtering
// Enhanced with safety scoring and DDI warnings
type MedicationProposal struct {
	MedicationCode     string    `json:"medication_code"`
	MedicationName     string    `json:"medication_name"`
	GenericName        string    `json:"generic_name"`
	TherapeuticClass   string    `json:"therapeutic_class"`
	Route              string    `json:"route"`
	FormulationOptions []string  `json:"formulation_options"`
	BaselineEfficacy   float64   `json:"baseline_efficacy"`
	SafetyProfile      string    `json:"safety_profile"`
	Status             string    `json:"status"` // "candidate", "requires_review"
	GeneratedAt        time.Time `json:"generated_at"`
	Indications        []string  `json:"indications"`
	IsGeneric          bool      `json:"is_generic"`

	// Enhanced fields
	SafetyScore      float64            `json:"safety_score"`
	DDIWarnings      []DDIInteraction   `json:"ddi_warnings"`
	FormularyTier    int                `json:"formulary_tier"`
	CostEstimate     float64            `json:"cost_estimate"`
}

// CandidateProposal is an alias for backward compatibility
type CandidateProposal = MedicationProposal

// ProvenanceInfo tracks data sources for auditability
type ProvenanceInfo struct {
	KBDrugMaster   string `json:"kb_drug_master"`
	KBDDI          string `json:"kb_ddi"`
	KBFormulary    string `json:"kb_formulary"`
	ProcessVersion string `json:"process_version"`
}

// ExclusionReason documents why a drug was filtered out
type ExclusionReason struct {
	DrugID      string        `json:"drug_id"`
	DrugName    string        `json:"drug_name"`
	ReasonCode  ExclusionCode `json:"reason_code"`
	Message     string        `json:"message"`
	SourceKB    string        `json:"source_kb"`
	KBVersion   string        `json:"kb_version"`
	Timestamp   time.Time     `json:"timestamp"`
}

// ExclusionCode represents standardized exclusion reason codes
type ExclusionCode string

const (
	ExclusionPatientContraindication ExclusionCode = "PATIENT_CONTRAINDICATION"
	ExclusionAllergy                 ExclusionCode = "PATIENT_ALLERGY"
	ExclusionDDIContraindicated      ExclusionCode = "DDI_CONTRAINDICATED"
	ExclusionRenalImpairment         ExclusionCode = "RENAL_IMPAIRMENT"
	ExclusionHepaticImpairment       ExclusionCode = "HEPATIC_IMPAIRMENT"
	ExclusionPregnancy               ExclusionCode = "PREGNANCY"
	ExclusionBlackBoxRisk            ExclusionCode = "BLACK_BOX_WARNING"
	ExclusionFormularyUnavailable    ExclusionCode = "FORMULARY_UNAVAILABLE"
)

// PatientContext contains all patient-specific filtering criteria
type PatientContext struct {
	RecommendedClasses []string            `json:"recommended_classes"`
	PatientFlags       map[string]bool     `json:"patient_flags"`
	Allergies          map[string]bool     `json:"allergies"`
	ActiveMeds         []ActiveMedication  `json:"active_meds"`
	Age                int                 `json:"age"`
	EGFR               float64             `json:"egfr"`
	ALT                float64             `json:"alt"`
	IsPregnant         bool                `json:"is_pregnant"`
	RequestID          string              `json:"request_id"`
	PatientID          string              `json:"patient_id"`
	FormularyID        string              `json:"formulary_id"`
}

// CandidateBuilderInput contains all inputs needed for candidate generation
type CandidateBuilderInput struct {
	// From Intent Manifest (Phase 1)
	RecommendedDrugClasses []string `json:"recommended_drug_classes"`
	
	// From CompleteContextPayload (Phase 2) - Patient data
	PatientFlags      map[string]bool    `json:"patient_flags"`
	ActiveMedications []ActiveMedication `json:"active_medications"`
	
	// From CompleteContextPayload (Phase 2) - Knowledge data
	DrugMasterList []Drug              `json:"drug_master_list"`
	DDIRules       []DrugInteraction   `json:"ddi_rules"`
	
	// Request metadata
	RequestID string `json:"request_id"`
	PatientID string `json:"patient_id"`
}

// CandidateBuilderResult contains the filtered candidates and comprehensive statistics
type CandidateBuilderResult struct {
	CandidateProposals  []MedicationProposal `json:"candidate_proposals"`
	FilteringStatistics FilteringStatistics  `json:"filtering_statistics"`
	ClinicalGuidance    *ClinicalGuidance    `json:"clinical_guidance,omitempty"`
	ProcessingMetadata  ProcessingMetadata   `json:"processing_metadata"`
	ExclusionLog        []ExclusionRecord    `json:"exclusion_log"`
}

// FilteringStatistics provides detailed metrics about the filtering process
type FilteringStatistics struct {
	InitialDrugCount        int     `json:"initial_drug_count"`
	ClassFilteredCount      int     `json:"class_filtered_count"`
	SafetyFilteredCount     int     `json:"safety_filtered_count"`
	FinalCandidateCount     int     `json:"final_candidate_count"`
	ClassReductionPercent   float64 `json:"class_reduction_percent"`
	SafetyReductionPercent  float64 `json:"safety_reduction_percent"`
	DDIReductionPercent     float64 `json:"ddi_reduction_percent"`
	OverallReductionPercent float64 `json:"overall_reduction_percent"`
	RequiresSpecialistReview bool   `json:"requires_specialist_review"`
	FallbackTriggered       bool    `json:"fallback_triggered"`
}

// ClinicalGuidance provides clinical recommendations when no safe options are found
type ClinicalGuidance struct {
	Severity           string   `json:"severity"`
	Message            string   `json:"message"`
	RecommendedActions []string `json:"recommended_actions"`
	SpecialistReferral bool     `json:"specialist_referral"`
	ClinicalReasoning  string   `json:"clinical_reasoning"`
}

// ProcessingMetadata contains processing information
type ProcessingMetadata struct {
	ProcessingTimeMs int64     `json:"processing_time_ms"`
	GeneratedAt      time.Time `json:"generated_at"`
	EngineVersion    string    `json:"engine_version"`
	FilterStagesRun  []string  `json:"filter_stages_run"`
}

// ExclusionRecord tracks why specific drugs were excluded for audit trail
type ExclusionRecord struct {
	DrugName        string    `json:"drug_name"`
	DrugCode        string    `json:"drug_code"`
	ExclusionReason string    `json:"exclusion_reason"`
	FilterStage     string    `json:"filter_stage"`
	PatientFlag     string    `json:"patient_flag,omitempty"`
	InteractingDrug string    `json:"interacting_drug,omitempty"`
	Severity        string    `json:"severity,omitempty"`
	Timestamp       time.Time `json:"timestamp"`
	ClinicalReason  string    `json:"clinical_reason"`
}

// FilterError represents errors that occur during filtering
type FilterError struct {
	Stage   string `json:"stage"`
	Message string `json:"message"`
	DrugCode string `json:"drug_code,omitempty"`
	Cause   error  `json:"-"`
}

func (e *FilterError) Error() string {
	if e.DrugCode != "" {
		return fmt.Sprintf("filter error in %s stage for drug %s: %s", e.Stage, e.DrugCode, e.Message)
	}
	return fmt.Sprintf("filter error in %s stage: %s", e.Stage, e.Message)
}

func (e *FilterError) Unwrap() error {
	return e.Cause
}

// ValidationError represents input validation errors
type ValidationError struct {
	Field   string      `json:"field"`
	Message string      `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field %s: %s", e.Field, e.Message)
}

// ==================== Service Interfaces ====================

// DrugMasterService provides access to the drug knowledge base
type DrugMasterService interface {
	ListDrugsByClass(ctx context.Context, classes []string) ([]Drug, error)
	GetDrugByID(ctx context.Context, drugID string) (*Drug, error)
	Version() string
}

// DDIService provides drug-drug interaction checking
type DDIService interface {
	BatchCheck(ctx context.Context, candidateDrugID string, activeDrugIDs []string) (map[string]*DDIInteraction, error)
	CheckInteraction(ctx context.Context, drugA, drugB string) (*DDIInteraction, error)
	Version() string
}

// FormularyService provides cost and availability data
type FormularyService interface {
	BatchGetFormularyStatus(ctx context.Context, drugIDs []string, formularyID string) (map[string]*FormularyStatus, error)
	GetFormularyStatus(ctx context.Context, drugID, formularyID string) (*FormularyStatus, error)
	Version() string
}

// FormularyStatus represents drug availability and cost information
type FormularyStatus struct {
	Available    bool    `json:"available"`
	Tier         int     `json:"tier"`
	CostEstimate float64 `json:"cost_estimate"`
	Restrictions string  `json:"restrictions"`
}

// MetricsCollector tracks system performance and safety events
type MetricsCollector interface {
	RecordFilteringComplete(requestID string, candidateCount, exclusionCount int, duration time.Duration)
	RecordExclusion(reasonCode ExclusionCode)
	RecordDDIServiceFailure(requestID string, err error)
	RecordFormularyServiceFailure(requestID string, err error)
}

// ==================== Configuration ====================

// BuilderConfig contains configuration for the candidate builder
type BuilderConfig struct {
	MaxWorkers           int           `json:"max_workers"`
	DDITimeout           time.Duration `json:"ddi_timeout"`
	FormularyTimeout     time.Duration `json:"formulary_timeout"`
	EnableFormularyCheck bool          `json:"enable_formulary_check"`
	EnableBlackBoxFilter bool          `json:"enable_black_box_filter"`
	StrictSafetyMode     bool          `json:"strict_safety_mode"` // If true, DDI service failures cause exclusion
	MaxSafetyScore       float64       `json:"max_safety_score"`
	MinSafetyScore       float64       `json:"min_safety_score"`
}

// DefaultBuilderConfig returns a safe default configuration
func DefaultBuilderConfig() *BuilderConfig {
	return &BuilderConfig{
		MaxWorkers:           10,
		DDITimeout:           30 * time.Second,
		FormularyTimeout:     15 * time.Second,
		EnableFormularyCheck: true,
		EnableBlackBoxFilter: true,
		StrictSafetyMode:     true, // Fail-safe by default
		MaxSafetyScore:       1.0,
		MinSafetyScore:       0.0,
	}
}
