// Package kbclients provides HTTP clients for Knowledge Base services.
// This implements the client layer for KB-1 through KB-6 services.
package kbclients

import (
	"context"
)

// KB1DosingClient interface for KB-1 Dosing Rules service
type KB1DosingClient interface {
	// GetStandardDosage returns standard dosage for a medication
	GetStandardDosage(ctx context.Context, rxnormCode string) (*DosageInfo, error)

	// CalculateDoseAdjustment calculates dose based on patient factors
	CalculateDoseAdjustment(ctx context.Context, req *DoseAdjustmentRequest) (*DoseAdjustmentResponse, error)

	// GetMaxDoseLimits returns maximum dose limits for a medication
	GetMaxDoseLimits(ctx context.Context, rxnormCode string) (*DoseLimits, error)

	// SearchByClass returns drugs matching a therapeutic class
	SearchByClass(ctx context.Context, therapeuticClass string) ([]DrugRule, error)

	// HealthCheck verifies KB-1 service availability
	HealthCheck(ctx context.Context) error

	Close() error
}

// DrugRule represents a drug rule from KB-1
type DrugRule struct {
	RxNormCode       string `json:"rxnorm_code"`
	DrugName         string `json:"drug_name"`
	TherapeuticClass string `json:"therapeutic_class"`
	DosingMethod     string `json:"dosing_method"`
	HasBlackBox      bool   `json:"has_black_box"`
	IsHighAlert      bool   `json:"is_high_alert"`
	IsNarrowTI       bool   `json:"is_narrow_ti"`
}

// KB2InteractionsClient interface for KB-2 Drug Interactions service
type KB2InteractionsClient interface {
	// CheckInteraction checks for drug-drug interaction
	CheckInteraction(ctx context.Context, drug1Code, drug2Code string) (*InteractionResult, error)

	// CheckMultipleInteractions checks interactions among multiple drugs
	CheckMultipleInteractions(ctx context.Context, drugCodes []string) (*MultiInteractionResult, error)

	// HealthCheck verifies KB-2 service availability
	HealthCheck(ctx context.Context) error

	Close() error
}

// KB3GuidelinesClient interface for KB-3 Clinical Guidelines service
type KB3GuidelinesClient interface {
	// GetRecommendedDrugs returns guideline-recommended drugs for indication
	GetRecommendedDrugs(ctx context.Context, indication string, drugClass string) ([]DrugRecommendation, error)

	// GetGuidelineSupport returns guideline evidence for a drug
	GetGuidelineSupport(ctx context.Context, rxnormCode, indication string) (*GuidelineEvidence, error)

	// GetFirstLineDrugs returns first-line drug recommendations
	GetFirstLineDrugs(ctx context.Context, indication string) ([]DrugRecommendation, error)

	// HealthCheck verifies KB-3 service availability
	HealthCheck(ctx context.Context) error

	Close() error
}

// KB4SafetyClient interface for KB-4 Patient Safety service
type KB4SafetyClient interface {
	// CheckContraindication checks if drug is contraindicated for patient
	CheckContraindication(ctx context.Context, req *ContraindicationRequest) (*ContraindicationResult, error)

	// CheckAllergyMatch checks if drug matches patient allergy
	CheckAllergyMatch(ctx context.Context, rxnormCode string, allergenCode string) (*AllergyMatchResult, error)

	// GetBlackBoxWarnings returns FDA black box warnings for a drug
	GetBlackBoxWarnings(ctx context.Context, rxnormCode string) ([]BlackBoxWarning, error)

	// CheckAgeAppropriate checks if drug is appropriate for patient age
	CheckAgeAppropriate(ctx context.Context, rxnormCode string, ageYears int) (*AgeCheckResult, error)

	// HealthCheck verifies KB-4 service availability
	HealthCheck(ctx context.Context) error

	Close() error
}

// KB5MonitoringClient interface for KB-5 Monitoring Requirements service
type KB5MonitoringClient interface {
	// GetMonitoringRequirements returns monitoring requirements for a drug
	GetMonitoringRequirements(ctx context.Context, rxnormCode string) (*MonitoringRequirements, error)

	// GetLabMonitoring returns required lab monitoring for a drug
	GetLabMonitoring(ctx context.Context, rxnormCode string) ([]LabMonitoring, error)

	// HealthCheck verifies KB-5 service availability
	HealthCheck(ctx context.Context) error

	Close() error
}

// KB6EfficacyClient interface for KB-6 Drug Efficacy service
type KB6EfficacyClient interface {
	// GetEfficacyScore returns efficacy score for drug-indication pair
	GetEfficacyScore(ctx context.Context, rxnormCode, indication string) (*EfficacyScore, error)

	// CompareEfficacy compares efficacy of multiple drugs
	CompareEfficacy(ctx context.Context, rxnormCodes []string, indication string) (*EfficacyComparison, error)

	// HealthCheck verifies KB-6 service availability
	HealthCheck(ctx context.Context) error

	Close() error
}

// =============================================================================
// Data Transfer Objects
// =============================================================================

// DosageInfo represents dosage information from KB-1
type DosageInfo struct {
	RxNormCode     string  `json:"rxnorm_code"`
	DrugName       string  `json:"drug_name"`
	StandardDose   float64 `json:"standard_dose"`
	Unit           string  `json:"unit"`
	Route          string  `json:"route"`
	Frequency      string  `json:"frequency"`
	MinDose        float64 `json:"min_dose"`
	MaxDose        float64 `json:"max_dose"`
	MaxDailyDose   float64 `json:"max_daily_dose"`
	RenalAdjust    bool    `json:"renal_adjust"`
	HepaticAdjust  bool    `json:"hepatic_adjust"`
}

// DoseAdjustmentRequest represents a dose adjustment calculation request
type DoseAdjustmentRequest struct {
	RxNormCode     string   `json:"rxnorm_code"`
	BaseDose       float64  `json:"base_dose"`
	EGFR           *float64 `json:"egfr,omitempty"`
	ChildPughClass string   `json:"child_pugh_class,omitempty"`
	AgeYears       int      `json:"age_years"`
	WeightKg       *float64 `json:"weight_kg,omitempty"`
}

// DoseAdjustmentResponse represents the result of dose adjustment
type DoseAdjustmentResponse struct {
	AdjustedDose    float64   `json:"adjusted_dose"`
	Unit            string    `json:"unit"`
	AdjustmentType  string    `json:"adjustment_type"` // renal, hepatic, age, weight
	AdjustmentRatio float64   `json:"adjustment_ratio"`
	Rationale       string    `json:"rationale"`
	Warnings        []string  `json:"warnings,omitempty"`
}

// DoseLimits represents dose limits for a medication
type DoseLimits struct {
	RxNormCode   string  `json:"rxnorm_code"`
	MinSingleDose float64 `json:"min_single_dose"`
	MaxSingleDose float64 `json:"max_single_dose"`
	MaxDailyDose  float64 `json:"max_daily_dose"`
	Unit          string  `json:"unit"`
}

// InteractionResult represents a drug-drug interaction result
type InteractionResult struct {
	Drug1Code       string `json:"drug1_code"`
	Drug2Code       string `json:"drug2_code"`
	HasInteraction  bool   `json:"has_interaction"`
	Severity        string `json:"severity"` // none, mild, moderate, severe, contraindicated
	Type            string `json:"type"`     // pharmacokinetic, pharmacodynamic
	Description     string `json:"description"`
	ClinicalEffect  string `json:"clinical_effect"`
	Recommendation  string `json:"recommendation"`
	EvidenceLevel   string `json:"evidence_level"`
}

// MultiInteractionResult represents multiple drug interaction results
type MultiInteractionResult struct {
	DrugCodes     []string            `json:"drug_codes"`
	Interactions  []InteractionResult `json:"interactions"`
	TotalSevere   int                 `json:"total_severe"`
	TotalModerate int                 `json:"total_moderate"`
	OverallRisk   string              `json:"overall_risk"` // low, moderate, high
}

// DrugRecommendation represents a guideline drug recommendation
type DrugRecommendation struct {
	RxNormCode       string  `json:"rxnorm_code"`
	DrugName         string  `json:"drug_name"`
	DrugClass        string  `json:"drug_class"`
	RecommendedDose  float64 `json:"recommended_dose"`
	Unit             string  `json:"unit"`
	Frequency        string  `json:"frequency"`
	GuidelineSource  string  `json:"guideline_source"`
	EvidenceGrade    string  `json:"evidence_grade"` // A, B, C
	RecommendationLevel string `json:"recommendation_level"` // strong, moderate, weak
	IsFirstLine      bool    `json:"is_first_line"`
}

// GuidelineEvidence represents clinical guideline evidence
type GuidelineEvidence struct {
	RxNormCode        string   `json:"rxnorm_code"`
	Indication        string   `json:"indication"`
	IsRecommended     bool     `json:"is_recommended"`
	EvidenceGrade     string   `json:"evidence_grade"`
	GuidelineName     string   `json:"guideline_name"`
	GuidelineVersion  string   `json:"guideline_version"`
	RecommendationText string  `json:"recommendation_text"`
	References        []string `json:"references,omitempty"`
}

// ContraindicationRequest represents a contraindication check request
type ContraindicationRequest struct {
	RxNormCode       string   `json:"rxnorm_code"`
	DrugName         string   `json:"drug_name,omitempty"`
	TherapeuticClass string   `json:"therapeutic_class,omitempty"` // For class-based contraindication checks
	ConditionCodes   []string `json:"condition_codes"`             // SNOMED codes
	AgeYears         int      `json:"age_years"`
	IsPregnant       bool     `json:"is_pregnant"`
	IsBreastfeeding  bool     `json:"is_breastfeeding"`
	EGFR             *float64 `json:"egfr,omitempty"`
}

// ContraindicationResult represents contraindication check result
type ContraindicationResult struct {
	RxNormCode        string   `json:"rxnorm_code"`
	IsContraindicated bool     `json:"is_contraindicated"`
	ContraindicationType string `json:"contraindication_type"` // absolute, relative
	Reason            string   `json:"reason"`
	ConditionCode     string   `json:"condition_code,omitempty"`
	Severity          string   `json:"severity"`
	Recommendation    string   `json:"recommendation"`
}

// AllergyMatchResult represents allergy match check result
type AllergyMatchResult struct {
	RxNormCode   string `json:"rxnorm_code"`
	AllergenCode string `json:"allergen_code"`
	IsMatch      bool   `json:"is_match"`
	MatchType    string `json:"match_type"` // direct, cross_reactivity, class
	Confidence   float64 `json:"confidence"`
	Description  string `json:"description"`
}

// BlackBoxWarning represents FDA black box warning
type BlackBoxWarning struct {
	RxNormCode  string `json:"rxnorm_code"`
	WarningID   string `json:"warning_id"`
	WarningText string `json:"warning_text"`
	Category    string `json:"category"`
}

// AgeCheckResult represents age appropriateness check
type AgeCheckResult struct {
	RxNormCode      string `json:"rxnorm_code"`
	IsAppropriate   bool   `json:"is_appropriate"`
	MinAge          int    `json:"min_age"`
	MaxAge          int    `json:"max_age,omitempty"`
	AgeWarning      string `json:"age_warning,omitempty"`
	DoseAdjustment  bool   `json:"dose_adjustment"`
}

// MonitoringRequirements represents drug monitoring requirements
type MonitoringRequirements struct {
	RxNormCode        string         `json:"rxnorm_code"`
	RequiresBaseline  bool           `json:"requires_baseline"`
	RequiresOngoing   bool           `json:"requires_ongoing"`
	LabMonitoring     []LabMonitoring `json:"lab_monitoring"`
	VitalMonitoring   []string       `json:"vital_monitoring,omitempty"`
	SymptomMonitoring []string       `json:"symptom_monitoring,omitempty"`
	MonitoringScore   float64        `json:"monitoring_score"` // 0-1 complexity score
}

// LabMonitoring represents lab monitoring requirement
type LabMonitoring struct {
	LOINCCode     string `json:"loinc_code"`
	TestName      string `json:"test_name"`
	Frequency     string `json:"frequency"`
	BaselineRequired bool `json:"baseline_required"`
	ThresholdHigh *float64 `json:"threshold_high,omitempty"`
	ThresholdLow  *float64 `json:"threshold_low,omitempty"`
	Urgency       string   `json:"urgency"` // routine, urgent
}

// EfficacyScore represents drug efficacy assessment
type EfficacyScore struct {
	RxNormCode      string  `json:"rxnorm_code"`
	Indication      string  `json:"indication"`
	EfficacyScore   float64 `json:"efficacy_score"` // 0-1
	NNT             *int    `json:"nnt,omitempty"`  // Number Needed to Treat
	EffectSize      string  `json:"effect_size"`    // small, medium, large
	EvidenceLevel   string  `json:"evidence_level"`
	ClinicalBenefit string  `json:"clinical_benefit"`
}

// EfficacyComparison represents comparison of multiple drugs
type EfficacyComparison struct {
	Indication   string         `json:"indication"`
	Scores       []EfficacyScore `json:"scores"`
	BestChoice   string          `json:"best_choice"` // RxNorm code
	Rationale    string          `json:"rationale"`
}

// KBVersionInfo represents version information for a KB service
type KBVersionInfo struct {
	ServiceName  string `json:"service_name"`
	Version      string `json:"version"`
	DataVersion  string `json:"data_version"`
	LastUpdated  string `json:"last_updated"`
}

// =============================================================================
// KB-14 Care Navigator (Task Generation)
// =============================================================================

// KB14TaskClient interface for KB-14 Care Navigator service
type KB14TaskClient interface {
	// CreateTask creates a new clinical task
	CreateTask(ctx context.Context, req *CreateTaskRequest) (*TaskResponse, error)

	// CreateMonitoringTask creates a monitoring/follow-up task for a medication
	CreateMonitoringTask(ctx context.Context, req *MonitoringTaskRequest) (*TaskResponse, error)

	// CreateMedicationReviewTask creates a medication review task
	CreateMedicationReviewTask(ctx context.Context, req *MedicationReviewTaskRequest) (*TaskResponse, error)

	// CreateTherapeuticChangeTask creates a task for therapeutic changes
	CreateTherapeuticChangeTask(ctx context.Context, req *TherapeuticChangeTaskRequest) (*TaskResponse, error)

	// HealthCheck verifies KB-14 service availability
	HealthCheck(ctx context.Context) error

	Close() error
}

// TaskType represents the type/category of task
type TaskType string

const (
	// Clinical Tasks (Licensed Clinician Required)
	TaskTypeCriticalLabReview     TaskType = "CRITICAL_LAB_REVIEW"
	TaskTypeMedicationReview      TaskType = "MEDICATION_REVIEW"
	TaskTypeAbnormalResult        TaskType = "ABNORMAL_RESULT"
	TaskTypeTherapeuticChange     TaskType = "THERAPEUTIC_CHANGE"
	TaskTypeCarePlanReview        TaskType = "CARE_PLAN_REVIEW"
	TaskTypeAcuteProtocolDeadline TaskType = "ACUTE_PROTOCOL_DEADLINE"

	// Care Coordination Tasks
	TaskTypeCareGapClosure     TaskType = "CARE_GAP_CLOSURE"
	TaskTypeMonitoringOverdue  TaskType = "MONITORING_OVERDUE"
	TaskTypeTransitionFollowup TaskType = "TRANSITION_FOLLOWUP"

	// Patient Outreach Tasks
	TaskTypeMedicationRefill TaskType = "MEDICATION_REFILL"
)

// TaskPriority represents the priority level of a task
type TaskPriority string

const (
	TaskPriorityCritical TaskPriority = "CRITICAL"
	TaskPriorityHigh     TaskPriority = "HIGH"
	TaskPriorityMedium   TaskPriority = "MEDIUM"
	TaskPriorityLow      TaskPriority = "LOW"
)

// TaskSource represents the source of the task
type TaskSource string

const (
	TaskSourceMedicationAdvisor TaskSource = "MEDICATION_ADVISOR"
	TaskSourceKB3               TaskSource = "KB3_TEMPORAL"
	TaskSourceKB9               TaskSource = "KB9_CARE_GAPS"
	TaskSourceManual            TaskSource = "MANUAL"
)

// CreateTaskRequest represents a generic task creation request
type CreateTaskRequest struct {
	Type         TaskType               `json:"type"`
	Priority     TaskPriority           `json:"priority,omitempty"`
	Source       TaskSource             `json:"source"`
	SourceID     string                 `json:"source_id,omitempty"`
	PatientID    string                 `json:"patient_id"`
	EncounterID  string                 `json:"encounter_id,omitempty"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description,omitempty"`
	Instructions string                 `json:"instructions,omitempty"`
	ClinicalNote string                 `json:"clinical_note,omitempty"`
	DueInMinutes int                    `json:"due_in_minutes,omitempty"`
	AssignedRole string                 `json:"assigned_role,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// MonitoringTaskRequest represents a request to create a monitoring task
type MonitoringTaskRequest struct {
	PatientID        string                 `json:"patient_id"`
	EncounterID      string                 `json:"encounter_id,omitempty"`
	MedicationCode   string                 `json:"medication_code"`
	MedicationName   string                 `json:"medication_name"`
	MonitoringType   string                 `json:"monitoring_type"` // lab, vital, symptom
	LOINCCode        string                 `json:"loinc_code,omitempty"`
	TestName         string                 `json:"test_name"`
	Frequency        string                 `json:"frequency"`
	DueInMinutes     int                    `json:"due_in_minutes"`
	IsBaseline       bool                   `json:"is_baseline"`
	Instructions     string                 `json:"instructions,omitempty"`
	ClinicalContext  string                 `json:"clinical_context,omitempty"`
	EnvelopeID       string                 `json:"envelope_id,omitempty"` // For audit trail
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// MedicationReviewTaskRequest represents a request to create a medication review task
type MedicationReviewTaskRequest struct {
	PatientID        string   `json:"patient_id"`
	EncounterID      string   `json:"encounter_id,omitempty"`
	ReviewType       string   `json:"review_type"` // interaction, polypharmacy, duplicate_therapy
	MedicationCodes  []string `json:"medication_codes"`
	Reason           string   `json:"reason"`
	Urgency          string   `json:"urgency"` // routine, urgent, stat
	AssignedRole     string   `json:"assigned_role,omitempty"`
	EnvelopeID       string   `json:"envelope_id,omitempty"`
}

// TherapeuticChangeTaskRequest represents a request to create a therapeutic change task
type TherapeuticChangeTaskRequest struct {
	PatientID        string `json:"patient_id"`
	EncounterID      string `json:"encounter_id,omitempty"`
	ChangeType       string `json:"change_type"` // dose_adjustment, switch, discontinue, add
	CurrentMedication string `json:"current_medication,omitempty"`
	NewMedication    string `json:"new_medication,omitempty"`
	Reason           string `json:"reason"`
	ClinicalContext  string `json:"clinical_context,omitempty"`
	EnvelopeID       string `json:"envelope_id,omitempty"`
}

// TaskResponse represents the response from creating a task
type TaskResponse struct {
	TaskID      string    `json:"task_id"`
	Status      string    `json:"status"`
	Type        TaskType  `json:"type"`
	Priority    TaskPriority `json:"priority"`
	Title       string    `json:"title"`
	DueDate     string    `json:"due_date,omitempty"`
	AssignedTo  string    `json:"assigned_to,omitempty"`
	CreatedAt   string    `json:"created_at"`
}

// GeneratedTask represents a task generated from medication advisory
type GeneratedTask struct {
	TaskType     TaskType     `json:"task_type"`
	Priority     TaskPriority `json:"priority"`
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	DueInMinutes int          `json:"due_in_minutes"`
	AssignedRole string       `json:"assigned_role"`
	Source       string       `json:"source"` // Which KB or rule generated this
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// =============================================================================
// KB-6 Formulary Client Interface
// =============================================================================

// KB6FormularyClient interface for KB-6 Formulary service
type KB6FormularyClient interface {
	// GetCoverage returns formulary coverage for a drug
	GetCoverage(ctx context.Context, req *FormularyCoverageRequest) (*FormularyCoverageResponse, error)

	// GetAlternatives returns formulary-preferred alternatives
	GetAlternatives(ctx context.Context, drugCode string, payerID string) ([]FormularyAlternative, error)

	// CheckPriorAuth checks if prior authorization is required
	CheckPriorAuth(ctx context.Context, drugCode string, payerID string) (*PriorAuthRequirement, error)

	// CheckStepTherapy checks step therapy requirements
	CheckStepTherapy(ctx context.Context, drugCode string, payerID string) (*StepTherapyRequirement, error)

	// HealthCheck verifies KB-6 service availability
	HealthCheck(ctx context.Context) error

	Close() error
}

// FormularyCoverageRequest represents a request to check formulary coverage
type FormularyCoverageRequest struct {
	DrugCode    string `json:"drug_code"`
	PayerID     string `json:"payer_id"`
	MemberID    string `json:"member_id,omitempty"`
	FormularyID string `json:"formulary_id,omitempty"`
}

// FormularyCoverageResponse represents formulary coverage information
type FormularyCoverageResponse struct {
	DrugCode         string           `json:"drug_code"`
	DrugName         string           `json:"drug_name"`
	CoverageStatus   CoverageStatus   `json:"coverage_status"`
	FormularyTier    int              `json:"formulary_tier"` // 1-6 (1=preferred generic)
	Copay            *CopayInfo       `json:"copay,omitempty"`
	RequiresPriorAuth bool            `json:"requires_prior_auth"`
	RequiresStepTherapy bool          `json:"requires_step_therapy"`
	QuantityLimits   *QuantityLimits  `json:"quantity_limits,omitempty"`
	CoverageNotes    string           `json:"coverage_notes,omitempty"`
}

// CoverageStatus represents formulary coverage status
type CoverageStatus string

const (
	CoverageStatusCovered         CoverageStatus = "COVERED"
	CoverageStatusNotCovered      CoverageStatus = "NOT_COVERED"
	CoverageStatusCoveredWithPA   CoverageStatus = "COVERED_WITH_PA"
	CoverageStatusCoveredWithST   CoverageStatus = "COVERED_WITH_STEP_THERAPY"
	CoverageStatusExcluded        CoverageStatus = "EXCLUDED"
	CoverageStatusTierException   CoverageStatus = "TIER_EXCEPTION_AVAILABLE"
)

// CopayInfo represents copay/cost sharing information
type CopayInfo struct {
	Amount       float64 `json:"amount"`
	Type         string  `json:"type"` // flat, percentage
	MaxOutOfPocket float64 `json:"max_out_of_pocket,omitempty"`
}

// QuantityLimits represents medication quantity limits
type QuantityLimits struct {
	MaxQuantity     int    `json:"max_quantity"`
	MaxDaysSupply   int    `json:"max_days_supply"`
	RefillLimitDays int    `json:"refill_limit_days"`
}

// FormularyAlternative represents a formulary-preferred alternative drug
type FormularyAlternative struct {
	DrugCode         string  `json:"drug_code"`
	DrugName         string  `json:"drug_name"`
	FormularyTier    int     `json:"formulary_tier"`
	CoverageStatus   CoverageStatus `json:"coverage_status"`
	CopayDifference  float64 `json:"copay_difference"` // vs original drug
	ClinicalRationale string `json:"clinical_rationale,omitempty"`
}

// PriorAuthRequirement represents prior authorization requirements
type PriorAuthRequirement struct {
	Required           bool     `json:"required"`
	ClinicalCriteria   []string `json:"clinical_criteria,omitempty"`
	DocumentationNeeded []string `json:"documentation_needed,omitempty"`
	EstimatedDays      int      `json:"estimated_days"`
}

// StepTherapyRequirement represents step therapy requirements
type StepTherapyRequirement struct {
	Required      bool     `json:"required"`
	PriorDrugs    []string `json:"prior_drugs"` // Drugs that must be tried first
	TrialDays     int      `json:"trial_days"`  // Days of trial required
	Exceptions    []string `json:"exceptions,omitempty"` // Clinical exceptions
}

// =============================================================================
// KB-9 Care Gaps Client Interface
// =============================================================================

// KB9CareGapsClient interface for KB-9 Care Gaps service
type KB9CareGapsClient interface {
	// GetPatientCareGaps returns care gaps for a patient
	GetPatientCareGaps(ctx context.Context, patientID string, measures []string) (*CareGapsReport, error)

	// GetMedicationRelatedGaps returns medication-related care gaps
	GetMedicationRelatedGaps(ctx context.Context, patientID string) ([]MedicationCareGap, error)

	// HealthCheck verifies KB-9 service availability
	HealthCheck(ctx context.Context) error

	Close() error
}

// CareGapsReport represents a patient's care gaps report
type CareGapsReport struct {
	PatientID     string        `json:"patient_id"`
	OpenGaps      []CareGap     `json:"open_gaps"`
	ClosedGaps    []CareGap     `json:"closed_gaps,omitempty"`
	GapCount      int           `json:"gap_count"`
	RiskScore     float64       `json:"risk_score"` // 0-1
	LastEvaluated string        `json:"last_evaluated"`
}

// CareGap represents a single care gap
type CareGap struct {
	GapID          string       `json:"gap_id"`
	MeasureType    string       `json:"measure_type"` // HBA1C, BP_CONTROL, MEDICATION_ADHERENCE, etc.
	Description    string       `json:"description"`
	Status         GapStatus    `json:"status"`
	Priority       string       `json:"priority"` // high, medium, low
	DueDate        string       `json:"due_date,omitempty"`
	RecommendedActions []string `json:"recommended_actions,omitempty"`
}

// GapStatus represents the status of a care gap
type GapStatus string

const (
	GapStatusOpen       GapStatus = "OPEN"
	GapStatusClosed     GapStatus = "CLOSED"
	GapStatusDismissed  GapStatus = "DISMISSED"
	GapStatusSnoozed    GapStatus = "SNOOZED"
)

// MedicationCareGap represents a medication-related care gap
type MedicationCareGap struct {
	GapType        string   `json:"gap_type"` // adherence, intensification, monitoring
	MedicationCode string   `json:"medication_code,omitempty"`
	MedicationName string   `json:"medication_name,omitempty"`
	DrugClass      string   `json:"drug_class,omitempty"`
	Reason         string   `json:"reason"`
	Recommendation string   `json:"recommendation"`
	Priority       string   `json:"priority"`
	EvidenceLevel  string   `json:"evidence_level"`
}

// =============================================================================
// KB-5 DDI (Drug-Drug Interaction) Hard Stops Interface
// =============================================================================

// KB5DDIClient interface for KB-5 Drug-Drug Interaction service
// This service specifically handles severe and life-threatening DDIs that require
// hard stops and mandatory clinical review before proceeding.
type KB5DDIClient interface {
	// CheckSevereDDI checks for severe/life-threatening interactions between drugs
	CheckSevereDDI(ctx context.Context, drug1Code, drug2Code string) (*DDIHardStopResult, error)

	// CheckMultipleDDIs checks all drug pairs for severe interactions
	CheckMultipleDDIs(ctx context.Context, drugCodes []string) (*DDIHardStopReport, error)

	// GetDDIHardStopRules returns all DDI pairs that require hard stops
	GetDDIHardStopRules(ctx context.Context) ([]DDIHardStopRule, error)

	// CheckContraindicatedCombination checks if a drug combination is absolutely contraindicated
	CheckContraindicatedCombination(ctx context.Context, req *DDICombinationRequest) (*DDICombinationResult, error)

	// HealthCheck verifies KB-5 DDI service availability
	HealthCheck(ctx context.Context) error

	Close() error
}

// DDISeverityLevel represents the severity classification of a DDI
type DDISeverityLevel string

const (
	// DDI severity levels from least to most severe
	DDISeverityMinor          DDISeverityLevel = "MINOR"          // Minimal clinical significance
	DDISeverityModerate       DDISeverityLevel = "MODERATE"       // May require monitoring
	DDISeverityMajor          DDISeverityLevel = "MAJOR"          // Significant clinical impact
	DDISeveritySevere         DDISeverityLevel = "SEVERE"         // High risk of serious adverse event
	DDISeverityContraindicated DDISeverityLevel = "CONTRAINDICATED" // Combination is absolutely contraindicated
	DDISeverityLifeThreatening DDISeverityLevel = "LIFE_THREATENING" // Immediate risk of death
)

// DDIHardStopResult represents the result of a severe DDI check
type DDIHardStopResult struct {
	Drug1Code         string           `json:"drug1_code"`
	Drug1Name         string           `json:"drug1_name"`
	Drug2Code         string           `json:"drug2_code"`
	Drug2Name         string           `json:"drug2_name"`
	HasHardStop       bool             `json:"has_hard_stop"`
	Severity          DDISeverityLevel `json:"severity"`
	ClinicalEffect    string           `json:"clinical_effect"`
	MechanismOfAction string           `json:"mechanism_of_action"`
	Recommendation    string           `json:"recommendation"`
	RiskScore         float64          `json:"risk_score"` // 0-1, higher = more dangerous
	EvidenceLevel     string           `json:"evidence_level"` // established, probable, suspected
	RequiresAck       bool             `json:"requires_ack"`
	AckText           string           `json:"ack_text,omitempty"`
	RuleID            string           `json:"rule_id"`
	References        []string         `json:"references,omitempty"`
}

// DDIHardStopReport represents the result of checking multiple drug DDIs
type DDIHardStopReport struct {
	DrugCodes          []string            `json:"drug_codes"`
	TotalPairsChecked  int                 `json:"total_pairs_checked"`
	HardStopCount      int                 `json:"hard_stop_count"`
	ContraindicatedPairs []DDIHardStopResult `json:"contraindicated_pairs"`
	SeverePairs        []DDIHardStopResult `json:"severe_pairs"`
	OverallRiskLevel   string              `json:"overall_risk_level"` // low, moderate, high, critical
	CanProceed         bool                `json:"can_proceed"` // false if any hard stops
	RequiredActions    []string            `json:"required_actions,omitempty"`
}

// DDIHardStopRule represents a rule defining a DDI hard stop
type DDIHardStopRule struct {
	RuleID            string           `json:"rule_id"`
	Drug1Class        string           `json:"drug1_class,omitempty"` // Drug class (e.g., "ACE_INHIBITORS")
	Drug1Code         string           `json:"drug1_code,omitempty"`  // Specific drug RxNorm code
	Drug2Class        string           `json:"drug2_class,omitempty"`
	Drug2Code         string           `json:"drug2_code,omitempty"`
	Severity          DDISeverityLevel `json:"severity"`
	ClinicalEffect    string           `json:"clinical_effect"`
	MechanismOfAction string           `json:"mechanism_of_action"`
	Exceptions        []string         `json:"exceptions,omitempty"` // Conditions where rule may not apply
	EffectiveDate     string           `json:"effective_date"`
	Source            string           `json:"source"` // FDA, Clinical Pharmacology, etc.
}

// DDICombinationRequest represents a request to check a drug combination
type DDICombinationRequest struct {
	CurrentMedications []string         `json:"current_medications"` // RxNorm codes
	ProposedMedication string           `json:"proposed_medication"` // RxNorm code to add
	PatientFactors     *DDIPatientFactors `json:"patient_factors,omitempty"`
}

// DDIPatientFactors represents patient factors affecting DDI severity
type DDIPatientFactors struct {
	AgeYears          int      `json:"age_years"`
	EGFR              *float64 `json:"egfr,omitempty"`
	ChildPughClass    string   `json:"child_pugh_class,omitempty"` // A, B, C
	IsElderly         bool     `json:"is_elderly"`
	PolypharmacyCount int      `json:"polypharmacy_count"` // Number of current medications
	HighRiskConditions []string `json:"high_risk_conditions,omitempty"` // SNOMED codes
}

// DDICombinationResult represents the result of checking a drug combination
type DDICombinationResult struct {
	ProposedMedication   string              `json:"proposed_medication"`
	CanAdd               bool                `json:"can_add"`
	HardStopsDetected    []DDIHardStopResult `json:"hard_stops_detected"`
	WarningsDetected     []DDIWarning        `json:"warnings_detected"`
	RecommendedAction    DDIAction           `json:"recommended_action"`
	AlternativeMedications []string          `json:"alternative_medications,omitempty"`
	ClinicalGuidance     string              `json:"clinical_guidance,omitempty"`
}

// DDIWarning represents a DDI warning that doesn't rise to hard stop level
type DDIWarning struct {
	Drug1Code        string           `json:"drug1_code"`
	Drug2Code        string           `json:"drug2_code"`
	Severity         DDISeverityLevel `json:"severity"`
	ClinicalEffect   string           `json:"clinical_effect"`
	MonitoringAdvice string           `json:"monitoring_advice"`
	DoseAdjustment   string           `json:"dose_adjustment,omitempty"`
}

// DDIAction represents the recommended action for a DDI
type DDIAction string

const (
	DDIActionProceed           DDIAction = "PROCEED"           // Safe to proceed
	DDIActionProceedWithMonitor DDIAction = "PROCEED_WITH_MONITOR" // Proceed but monitor closely
	DDIActionRequireAck        DDIAction = "REQUIRE_ACK"        // Require acknowledgment
	DDIActionHardStop          DDIAction = "HARD_STOP"          // Cannot proceed
	DDIActionConsultPharmacy   DDIAction = "CONSULT_PHARMACY"   // Require pharmacy consultation
	DDIActionConsultSpecialist DDIAction = "CONSULT_SPECIALIST" // Require specialist review
)

// =============================================================================
// KB-16 Lab Safety Client Interface
// =============================================================================

// KB16LabSafetyClient interface for KB-16 Lab-Based Safety Checks service
// This service provides real-time lab value safety checks before medication decisions.
// Critical lab values can trigger hard blocks or warnings for specific medications.
type KB16LabSafetyClient interface {
	// CheckLabSafety checks if current lab values are safe for a medication
	CheckLabSafety(ctx context.Context, req *LabSafetyRequest) (*LabSafetyResult, error)

	// CheckCriticalLabs checks for critical lab values that may affect any medication
	CheckCriticalLabs(ctx context.Context, labs []LabValue) (*CriticalLabReport, error)

	// GetLabThresholds returns lab thresholds for a specific medication
	GetLabThresholds(ctx context.Context, rxnormCode string) ([]LabThreshold, error)

	// CheckTrendSafety checks if lab trends indicate safety concerns
	CheckTrendSafety(ctx context.Context, req *LabTrendRequest) (*LabTrendResult, error)

	// HealthCheck verifies KB-16 service availability
	HealthCheck(ctx context.Context) error

	Close() error
}

// LabValue represents a laboratory result value
type LabValue struct {
	LOINCCode      string      `json:"loinc_code"`
	TestName       string      `json:"test_name"`
	Value          float64     `json:"value"`
	Unit           string      `json:"unit"`
	ReferenceRange string      `json:"reference_range,omitempty"`
	CollectedAt    string      `json:"collected_at"` // ISO8601 timestamp
	IsCritical     bool        `json:"is_critical"`
	CriticalType   string      `json:"critical_type,omitempty"` // high, low, panic
}

// LabSafetyRequest represents a request to check lab safety for a medication
type LabSafetyRequest struct {
	RxNormCode     string     `json:"rxnorm_code"`
	DrugName       string     `json:"drug_name,omitempty"`
	CurrentLabs    []LabValue `json:"current_labs"`
	PatientAge     int        `json:"patient_age"`
	EGFR           *float64   `json:"egfr,omitempty"`
	ChildPughClass string     `json:"child_pugh_class,omitempty"`
}

// LabSafetyResult represents the result of lab safety check
type LabSafetyResult struct {
	RxNormCode       string           `json:"rxnorm_code"`
	IsSafe           bool             `json:"is_safe"`
	SafetyLevel      LabSafetyLevel   `json:"safety_level"`
	Violations       []LabViolation   `json:"violations,omitempty"`
	Warnings         []LabWarning     `json:"warnings,omitempty"`
	RecommendedAction LabSafetyAction `json:"recommended_action"`
	ClinicalGuidance string           `json:"clinical_guidance,omitempty"`
	RequiresAck      bool             `json:"requires_ack"`
	AckText          string           `json:"ack_text,omitempty"`
}

// LabSafetyLevel represents the safety classification based on labs
type LabSafetyLevel string

const (
	LabSafetyLevelSafe          LabSafetyLevel = "SAFE"           // Labs within safe range
	LabSafetyLevelCaution       LabSafetyLevel = "CAUTION"        // Monitor closely
	LabSafetyLevelWarning       LabSafetyLevel = "WARNING"        // Significant concern
	LabSafetyLevelContraindicated LabSafetyLevel = "CONTRAINDICATED" // Unsafe to proceed
	LabSafetyLevelCritical      LabSafetyLevel = "CRITICAL"       // Immediate danger
)

// LabSafetyAction represents the recommended action based on lab values
type LabSafetyAction string

const (
	LabSafetyActionProceed         LabSafetyAction = "PROCEED"          // Safe to prescribe
	LabSafetyActionMonitor         LabSafetyAction = "MONITOR"          // Proceed with monitoring
	LabSafetyActionReduceDose      LabSafetyAction = "REDUCE_DOSE"      // Lower dose required
	LabSafetyActionHoldMedication  LabSafetyAction = "HOLD_MEDICATION"  // Do not give medication
	LabSafetyActionHardStop        LabSafetyAction = "HARD_STOP"        // Absolute contraindication
	LabSafetyActionRepeatLab       LabSafetyAction = "REPEAT_LAB"       // Recheck lab before decision
	LabSafetyActionConsultNephro   LabSafetyAction = "CONSULT_NEPHRO"   // Nephrology consultation
	LabSafetyActionConsultCardio   LabSafetyAction = "CONSULT_CARDIO"   // Cardiology consultation
)

// LabViolation represents a critical lab threshold violation
type LabViolation struct {
	LOINCCode        string          `json:"loinc_code"`
	TestName         string          `json:"test_name"`
	CurrentValue     float64         `json:"current_value"`
	ThresholdValue   float64         `json:"threshold_value"`
	ThresholdType    string          `json:"threshold_type"` // max, min, range
	Severity         LabSafetyLevel  `json:"severity"`
	ClinicalEffect   string          `json:"clinical_effect"`
	Recommendation   string          `json:"recommendation"`
	RuleID           string          `json:"rule_id"`
}

// LabWarning represents a lab-based warning (not yet critical)
type LabWarning struct {
	LOINCCode        string  `json:"loinc_code"`
	TestName         string  `json:"test_name"`
	CurrentValue     float64 `json:"current_value"`
	WarningThreshold float64 `json:"warning_threshold"`
	TrendDirection   string  `json:"trend_direction,omitempty"` // rising, falling, stable
	MonitoringAdvice string  `json:"monitoring_advice"`
}

// CriticalLabReport represents a report of critical lab values
type CriticalLabReport struct {
	HasCriticalLabs     bool               `json:"has_critical_labs"`
	CriticalCount       int                `json:"critical_count"`
	CriticalValues      []CriticalLabValue `json:"critical_values"`
	AffectedMedications []string           `json:"affected_medications,omitempty"`
	OverallRiskLevel    string             `json:"overall_risk_level"` // low, moderate, high, critical
	ImmediateActions    []string           `json:"immediate_actions,omitempty"`
}

// CriticalLabValue represents a single critical lab result
type CriticalLabValue struct {
	LOINCCode       string   `json:"loinc_code"`
	TestName        string   `json:"test_name"`
	Value           float64  `json:"value"`
	Unit            string   `json:"unit"`
	CriticalType    string   `json:"critical_type"` // critical_high, critical_low, panic
	NormalRange     string   `json:"normal_range"`
	ClinicalImpact  string   `json:"clinical_impact"`
	ImmediateAction string   `json:"immediate_action"`
	DrugClasses     []string `json:"affected_drug_classes,omitempty"`
}

// LabThreshold represents a lab threshold for a specific medication
type LabThreshold struct {
	RxNormCode      string          `json:"rxnorm_code"`
	LOINCCode       string          `json:"loinc_code"`
	TestName        string          `json:"test_name"`
	MinValue        *float64        `json:"min_value,omitempty"`
	MaxValue        *float64        `json:"max_value,omitempty"`
	WarningMin      *float64        `json:"warning_min,omitempty"`
	WarningMax      *float64        `json:"warning_max,omitempty"`
	Unit            string          `json:"unit"`
	ActionIfViolated LabSafetyAction `json:"action_if_violated"`
	ClinicalRationale string        `json:"clinical_rationale"`
	Source          string          `json:"source"` // FDA, Clinical Guidelines, etc.
}

// LabTrendRequest represents a request to check lab trends
type LabTrendRequest struct {
	LOINCCode       string     `json:"loinc_code"`
	HistoricalLabs  []LabValue `json:"historical_labs"` // Ordered by time, oldest first
	ProposedMedCode string     `json:"proposed_med_code,omitempty"`
	LookbackDays    int        `json:"lookback_days,omitempty"` // Default 30
}

// LabTrendResult represents the result of trend analysis
type LabTrendResult struct {
	LOINCCode        string  `json:"loinc_code"`
	TestName         string  `json:"test_name"`
	TrendDirection   string  `json:"trend_direction"` // rising, falling, stable, volatile
	TrendMagnitude   float64 `json:"trend_magnitude"` // Change per day
	ProjectedValue   float64 `json:"projected_value"` // 7-day projection
	IsConcerning     bool    `json:"is_concerning"`
	ClinicalConcern  string  `json:"clinical_concern,omitempty"`
	Recommendation   string  `json:"recommendation"`
}

// =============================================================================
// Known Lab-Drug Contraindication Rules
// =============================================================================

// LabDrugRule represents a rule linking lab values to drug safety
type LabDrugRule struct {
	RuleID          string          `json:"rule_id"`
	RxNormCode      string          `json:"rxnorm_code,omitempty"`
	DrugClass       string          `json:"drug_class,omitempty"`
	LOINCCode       string          `json:"loinc_code"`
	TestName        string          `json:"test_name"`
	Threshold       float64         `json:"threshold"`
	ThresholdType   string          `json:"threshold_type"` // max, min
	Severity        LabSafetyLevel  `json:"severity"`
	Action          LabSafetyAction `json:"action"`
	ClinicalEffect  string          `json:"clinical_effect"`
	Recommendation  string          `json:"recommendation"`
}
