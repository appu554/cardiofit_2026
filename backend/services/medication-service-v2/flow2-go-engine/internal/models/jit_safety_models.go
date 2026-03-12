// Package models contains data structures for JIT Safety Engine integration
package models

import (
	"time"

	candidatebuilder "flow2-go-engine/internal/clinical-intelligence/candidate-builder"
)

// JitSafetyContext represents the input to JIT Safety Engine
type JitSafetyContext struct {
	Patient        PatientCtx            `json:"patient"`
	ConcurrentMeds []ConcurrentMed       `json:"concurrent_meds"`
	Proposal       ProposedDose          `json:"proposal"`
	KBVersions     map[string]string     `json:"kb_versions"`
	RequestID      string                `json:"request_id"`
}

// PatientCtx represents patient context for safety evaluation
type PatientCtx struct {
	AgeYears      uint32      `json:"age_years"`
	Sex           string      `json:"sex"`
	WeightKg      float64     `json:"weight_kg"`
	HeightCm      *float64    `json:"height_cm,omitempty"`
	Pregnancy     bool        `json:"pregnancy"`
	Renal         RenalCtx    `json:"renal"`
	Hepatic       HepaticCtx  `json:"hepatic"`
	QTcMs         *uint32     `json:"qtc_ms,omitempty"`
	Allergies     []string    `json:"allergies"`
	Conditions    []string    `json:"conditions"`
	Labs          LabsCtx     `json:"labs"`
}

// RenalCtx represents renal function context
type RenalCtx struct {
	EGFR *float64 `json:"egfr,omitempty"`  // eGFR in mL/min/1.73m²
	CrCl *float64 `json:"crcl,omitempty"`  // CrCl in mL/min
}

// HepaticCtx represents hepatic function context
type HepaticCtx struct {
	ChildPugh *string `json:"child_pugh,omitempty"` // "A", "B", "C"
}

// LabsCtx represents laboratory values context
type LabsCtx struct {
	ALT  *float64 `json:"alt,omitempty"`   // ALT in U/L
	AST  *float64 `json:"ast,omitempty"`   // AST in U/L
	UACR *float64 `json:"uacr,omitempty"`  // UACR in mg/g
}

// ConcurrentMed represents a concurrent medication
type ConcurrentMed struct {
	DrugID    string  `json:"drug_id"`
	ClassID   string  `json:"class_id"`
	DoseMg    float64 `json:"dose_mg"`
	IntervalH uint32  `json:"interval_h"`
}

// ProposedDose represents a proposed medication dose
type ProposedDose struct {
	DrugID    string  `json:"drug_id"`
	DoseMg    float64 `json:"dose_mg"`
	Route     string  `json:"route"`       // "po", "iv", "im", "sc"
	IntervalH uint32  `json:"interval_h"`  // q24h => 24, q12h => 12
}

// JitSafetyOutcome represents the output from JIT Safety Engine
type JitSafetyOutcome struct {
	Decision   Decision    `json:"decision"`
	FinalDose  ProposedDose `json:"final_dose"`
	Reasons    []Reason    `json:"reasons"`
	DDIs       []DDIFlag   `json:"ddis"`
	Provenance Provenance  `json:"provenance"`
}

// Decision represents the safety evaluation decision
type Decision string

const (
	DecisionAllow              Decision = "allow"
	DecisionAllowWithAdjustment Decision = "allow_with_adjustment"
	DecisionBlock              Decision = "block"
)

// Reason represents a safety evaluation reason
type Reason struct {
	Code     string   `json:"code"`       // e.g., "RENAL_CAP_APPLIED"
	Severity string   `json:"severity"`   // info|warn|error|blocker
	Message  string   `json:"message"`
	Evidence []string `json:"evidence"`
	RuleID   string   `json:"rule_id"`
}

// DDIFlag represents a drug-drug interaction flag
type DDIFlag struct {
	WithDrugID string `json:"with_drug_id"`
	Severity   string `json:"severity"`   // minor|moderate|major|contraindicated
	Action     string `json:"action"`
	Code       string `json:"code"`
	RuleID     string `json:"rule_id"`
}

// Provenance represents evaluation provenance and audit trail
type Provenance struct {
	EngineVersion    string            `json:"engine_version"`
	KBVersions       map[string]string `json:"kb_versions"`
	EvaluationTrace  []EvalStep        `json:"evaluation_trace"`
}

// EvalStep represents an individual evaluation step for audit trail
type EvalStep struct {
	RuleID string `json:"rule_id"`
	Result string `json:"result"`
}

// SafetyVerifiedProposal represents a proposal that has passed JIT Safety verification
type SafetyVerifiedProposal struct {
	Original      candidatebuilder.CandidateProposal `json:"original"`
	SafetyScore   float64                            `json:"safety_score"`
	FinalDose     ProposedDose                       `json:"final_dose"`
	SafetyReasons []Reason                           `json:"safety_reasons"`
	DDIWarnings   []DDIFlag                          `json:"ddi_warnings"`
	Action        string                             `json:"action"` // "CanProceed", "RequiresReview"
	JITProvenance Provenance                         `json:"jit_provenance"`
	ProcessedAt   time.Time                          `json:"processed_at"`
}

// ScoredProposal represents a proposal with multi-factor scoring
type ScoredProposal struct {
	SafetyVerified  SafetyVerifiedProposal `json:"safety_verified"`
	TotalScore      float64               `json:"total_score"`
	ComponentScores ComponentScores       `json:"component_scores"`
	Ranking         int                   `json:"ranking"`
	ScoredAt        time.Time             `json:"scored_at"`
}

// EnhancedScoredProposal represents a proposal with comprehensive compare-and-rank scoring
type EnhancedScoredProposal struct {
	TherapyID           string                  `json:"therapy_id"`
	SafetyVerified      SafetyVerifiedProposal  `json:"safety_verified"`
	FinalScore          float64                 `json:"final_score"`
	Rank                int                     `json:"rank"`
	SubScores           EnhancedComponentScores `json:"sub_scores"`
	Contributions       []ScoreContribution     `json:"contributions"`
	EligibilityFlags    EligibilityFlags        `json:"eligibility_flags"`
	Notes               []string                `json:"notes"`
	AuditInfo           ProposalAuditInfo       `json:"audit_info"`
	ScoredAt            time.Time               `json:"scored_at"`
}

// ScoreContribution represents individual factor contributions to final score
type ScoreContribution struct {
	Factor       string  `json:"factor"`
	Value        float64 `json:"value"`
	Weight       float64 `json:"weight"`
	Contribution float64 `json:"contribution"`
	Note         string  `json:"note"`
}

// EligibilityFlags represents eligibility status and reasons
type EligibilityFlags struct {
	TopSlotEligible bool     `json:"top_slot_eligible"`
	Reasons         []string `json:"reasons"`
}

// ProposalAuditInfo represents audit information for a proposal
type ProposalAuditInfo struct {
	KBVersions          map[string]string `json:"kb_versions"`
	RawInputs           map[string]interface{} `json:"raw_inputs"`
	NormalizationRanges map[string]NormalizationRange `json:"normalization_ranges"`
	ProcessingTime      time.Duration     `json:"processing_time"`
}

// NormalizationRange represents min/max values used for normalization
type NormalizationRange struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// ComponentScores represents individual scoring components with enhanced details
type ComponentScores struct {
	SafetyScore             float64 `json:"safety_score"`
	EfficacyScore           float64 `json:"efficacy_score"`
	CostScore               float64 `json:"cost_score"`
	ConvenienceScore        float64 `json:"convenience_score"`
	PatientPreferenceScore  float64 `json:"patient_preference_score"`
	GuidelineAdherenceScore float64 `json:"guideline_adherence_score"`

	// Enhanced scoring components for Compare-and-Rank
	AvailabilityScore       float64 `json:"availability_score"`
	AdherenceScore          float64 `json:"adherence_score"`
}

// EnhancedComponentScores represents detailed scoring with sub-components
type EnhancedComponentScores struct {
	Efficacy    EfficacyScoreDetail    `json:"efficacy"`
	Safety      SafetyScoreDetail      `json:"safety"`
	Availability AvailabilityScoreDetail `json:"availability"`
	Cost        CostScoreDetail        `json:"cost"`
	Adherence   AdherenceScoreDetail   `json:"adherence"`
	Preference  PreferenceScoreDetail  `json:"preference"`
}

// EfficacyScoreDetail provides detailed efficacy scoring breakdown
type EfficacyScoreDetail struct {
	Score                float64 `json:"score"`
	ExpectedA1cDropPct   float64 `json:"expected_a1c_drop_pct"`
	CVBenefit           bool    `json:"cv_benefit"`
	HFBenefit           bool    `json:"hf_benefit"`
	CKDBenefit          bool    `json:"ckd_benefit"`
	PhenotypeBonus      float64 `json:"phenotype_bonus"`
	EvidenceLevel       string  `json:"evidence_level"`
}

// SafetyScoreDetail provides detailed safety scoring breakdown
type SafetyScoreDetail struct {
	Score                float64            `json:"score"`
	ResidualDDI         string             `json:"residual_ddi"` // "none", "moderate", "major"
	HypoPropensity      string             `json:"hypo_propensity"` // "low", "med", "high"
	WeightEffect        string             `json:"weight_effect"` // "loss", "neutral", "gain"
	RenalFit            bool               `json:"renal_fit"`
	HepaticFit          bool               `json:"hepatic_fit"`
	SafetyPenalties     []SafetyPenalty    `json:"safety_penalties"`
}

// SafetyPenalty represents individual safety penalty
type SafetyPenalty struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Penalty     float64 `json:"penalty"`
}

// AvailabilityScoreDetail provides detailed availability scoring breakdown
type AvailabilityScoreDetail struct {
	Score           float64 `json:"score"`
	FormularyTier   int     `json:"formulary_tier"`
	TierFactor      float64 `json:"tier_factor"`
	OnHand          int     `json:"on_hand"`
	StockFactor     float64 `json:"stock_factor"`
	LeadTimeDays    int     `json:"lead_time_days"`
}

// CostScoreDetail provides detailed cost scoring breakdown
type CostScoreDetail struct {
	Score              float64 `json:"score"`
	MonthlyEstimate    float64 `json:"monthly_estimate"`
	Currency           string  `json:"currency"`
	PatientCopay       float64 `json:"patient_copay,omitempty"`
	NormalizedCost     float64 `json:"normalized_cost"`
}

// AdherenceScoreDetail provides detailed adherence scoring breakdown
type AdherenceScoreDetail struct {
	Score              float64 `json:"score"`
	BaseScore          float64 `json:"base_score"`
	FrequencyBonus     float64 `json:"frequency_bonus"`
	FDCBonus           float64 `json:"fdc_bonus"`
	InjectablePenalty  float64 `json:"injectable_penalty"`
	DeviceTrainingPenalty float64 `json:"device_training_penalty"`
	PillBurden         int     `json:"pill_burden"`
	DosesPerDay        int     `json:"doses_per_day"`
	IsFDC              bool    `json:"is_fdc"`
	RequiresDevice     bool    `json:"requires_device"`
}

// PreferenceScoreDetail provides detailed preference scoring breakdown
type PreferenceScoreDetail struct {
	Score                float64              `json:"score"`
	BaseScore            float64              `json:"base_score"`
	ViolatedPreferences  []ViolatedPreference `json:"violated_preferences"`
}

// ViolatedPreference represents a patient preference violation
type ViolatedPreference struct {
	Type        string  `json:"type"` // "strong", "soft"
	Description string  `json:"description"`
	Penalty     float64 `json:"penalty"`
}

// === ENHANCED PROPOSAL INPUT STRUCTURES ===

// EnhancedProposal represents a dose-aware proposal ready for compare-and-rank
type EnhancedProposal struct {
	TherapyID    string           `json:"therapy_id"`
	Class        string           `json:"class"`
	Agent        string           `json:"agent"`
	Regimen      RegimenDetail    `json:"regimen"`
	Dose         DoseDetail       `json:"dose"`
	Efficacy     EfficacyDetail   `json:"efficacy"`
	Safety       SafetyDetail     `json:"safety"`
	Suitability  SuitabilityDetail `json:"suitability"`
	Adherence    AdherenceDetail  `json:"adherence"`
	Availability AvailabilityDetail `json:"availability"`
	Cost         CostDetail       `json:"cost"`
	Preferences  PreferencesDetail `json:"preferences"`
	Provenance   ProvenanceDetail `json:"provenance"`
}

// RegimenDetail represents regimen information
type RegimenDetail struct {
	Form      string `json:"form"`
	Frequency string `json:"frequency"`
	IsFDC     bool   `json:"fdc"`
	PillCount int    `json:"pill_count"`
}

// DoseDetail represents detailed dose information
type DoseDetail struct {
	Amount    float64 `json:"amount"`
	Unit      string  `json:"unit"`
	Frequency string  `json:"frequency"`
	Route     string  `json:"route"`
	Rationale string  `json:"rationale"`
}

// EfficacyDetail represents efficacy information
type EfficacyDetail struct {
	ExpectedA1cDropPct float64 `json:"expected_a1c_drop_pct"`
	CVBenefit         bool    `json:"cv_benefit"`
	HFBenefit         bool    `json:"hf_benefit"`
	CKDBenefit        bool    `json:"ckd_benefit"`
}

// SafetyDetail represents safety information
type SafetyDetail struct {
	ResidualDDI     string `json:"residual_ddi"`     // "none", "moderate", "major"
	HypoPropensity  string `json:"hypo_propensity"`  // "low", "med", "high"
	WeightEffect    string `json:"weight_effect"`    // "loss", "neutral", "gain"
}

// SuitabilityDetail represents organ function suitability
type SuitabilityDetail struct {
	RenalFit   bool `json:"renal_fit"`
	HepaticFit bool `json:"hepatic_fit"`
}

// AdherenceDetail represents adherence factors
type AdherenceDetail struct {
	DosesPerDay      int  `json:"doses_per_day"`
	PillBurden       int  `json:"pill_burden"`
	RequiresDevice   bool `json:"requires_device"`
	RequiresTraining bool `json:"requires_training"`
}

// AvailabilityDetail represents availability information
type AvailabilityDetail struct {
	Tier         int `json:"tier"`
	OnHand       int `json:"on_hand"`
	LeadTimeDays int `json:"lead_time_days"`
}

// CostDetail represents cost information
type CostDetail struct {
	MonthlyEstimate float64 `json:"monthly_estimate"`
	Currency        string  `json:"currency"`
	PatientCopay    float64 `json:"patient_copay,omitempty"`
}

// PreferencesDetail represents patient preferences
type PreferencesDetail struct {
	AvoidInjectables     bool `json:"avoid_injectables"`
	OnceDailyPreferred   bool `json:"once_daily_preferred"`
	CostSensitivity      string `json:"cost_sensitivity"` // "low", "medium", "high"
}

// ProvenanceDetail represents data provenance
type ProvenanceDetail struct {
	KBVersions map[string]string `json:"kb_versions"`
}

// === COMPARE-AND-RANK REQUEST/RESPONSE MODELS ===

// CompareAndRankRequest represents a request to compare and rank proposals
type CompareAndRankRequest struct {
	PatientContext PatientRiskContext   `json:"patient_context"`
	Candidates     []EnhancedProposal   `json:"candidates"`
	ConfigRef      ConfigReference      `json:"config_ref"`
	RequestID      string               `json:"request_id"`
	Timestamp      time.Time            `json:"timestamp"`
}

// PatientRiskContext represents patient context for risk-aware ranking
type PatientRiskContext struct {
	RiskPhenotype string             `json:"risk_phenotype"` // "ASCVD", "HF", "CKD", "NONE"
	ResourceTier  string             `json:"resource_tier"`  // "minimal", "standard", "advanced"
	Preferences   JITPatientPreferences `json:"preferences"`
}

// JITPatientPreferences represents patient preferences for JIT safety ranking
type JITPatientPreferences struct {
	AvoidInjectables     bool   `json:"avoid_injectables"`
	OnceDailyPreferred   bool   `json:"once_daily_preferred"`
	CostSensitivity      string `json:"cost_sensitivity"` // "low", "medium", "high"
}

// ConfigReference represents configuration references for ranking
type ConfigReference struct {
	WeightProfile    string `json:"weight_profile"`    // e.g., "ASCVD", "BUDGET_MODE"
	PenaltiesProfile string `json:"penalties_profile"`
}

// CompareAndRankResponse represents the response from compare-and-rank
type CompareAndRankResponse struct {
	Ranked []EnhancedScoredProposal `json:"ranked"`
	Audit  RankingAuditInfo         `json:"audit"`
}

// RankingAuditInfo represents audit information for ranking process
type RankingAuditInfo struct {
	NormalizationRanges map[string]NormalizationRange `json:"normalization_ranges"`
	ProfileUsed         ProfileUsedInfo               `json:"profile_used"`
	ProcessingTime      time.Duration                 `json:"processing_time"`
	CandidatesProcessed int                           `json:"candidates_processed"`
	CandidatesPruned    int                           `json:"candidates_pruned"`
}

// ProfileUsedInfo represents the profiles used for ranking
type ProfileUsedInfo struct {
	Weights    string `json:"weights"`
	Penalties  string `json:"penalties"`
}

// JITSafetyRequest represents a request to the JIT Safety Engine
type JITSafetyRequest struct {
	Context   JitSafetyContext `json:"context"`
	RequestID string           `json:"request_id"`
	Timestamp time.Time        `json:"timestamp"`
}

// JITSafetyResponse represents a response from the JIT Safety Engine
type JITSafetyResponse struct {
	Outcome   JitSafetyOutcome `json:"outcome"`
	RequestID string           `json:"request_id"`
	Timestamp time.Time        `json:"timestamp"`
	Duration  time.Duration    `json:"duration"`
}

// JITSafetyError represents an error from the JIT Safety Engine
type JITSafetyError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
	DrugID    string `json:"drug_id,omitempty"`
}

func (e *JITSafetyError) Error() string {
	return e.Message
}

// Enhanced Safety Check Models (placeholder for future implementation)

// EnhancedSafetyRequest represents a request to the enhanced JIT Safety Engine
type EnhancedSafetyRequest struct {
	RequestID string    `json:"request_id"`
	Timestamp time.Time `json:"timestamp"`
	// Additional fields for enhanced safety check would go here
}

// EnhancedSafetyResponse represents a response from the enhanced JIT Safety Engine
type EnhancedSafetyResponse struct {
	RequestID string    `json:"request_id"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	// Additional fields for enhanced safety response would go here
}

// Additional types needed for the helper functions

// PatientContext represents patient context for JIT Safety
type PatientContext struct {
	Demographics      PatientDemographics      `json:"demographics"`
	Allergies         []Allergy               `json:"allergies"`
	Conditions        []Condition             `json:"conditions"`
	LabResults        LabResults              `json:"lab_results"`
	ActiveMedications []ActiveMedication      `json:"active_medications"`
}

// PatientDemographics represents patient demographic information
type PatientDemographics struct {
	Age        int     `json:"age"`
	Gender     string  `json:"gender"`
	Weight     float64 `json:"weight"`
	Height     float64 `json:"height"`
	IsPregnant bool    `json:"is_pregnant"`
}

// Allergy represents an allergy
type Allergy struct {
	AllergenCode string `json:"allergen_code"`
	AllergenName string `json:"allergen_name"`
	Severity     string `json:"severity"`
}

// Condition represents a medical condition
type Condition struct {
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Severity  string    `json:"severity"`
	OnsetDate time.Time `json:"onset_date"`
}

// LabResults represents laboratory results
type LabResults struct {
	EGFR                float64 `json:"egfr"`
	CreatinineClearance float64 `json:"creatinine_clearance"`
	ALT                 float64 `json:"alt"`
	AST                 float64 `json:"ast"`
	UACR                float64 `json:"uacr"`
	HbA1c               float64 `json:"hba1c"`
	Potassium           float64 `json:"potassium"`
	Sodium              float64 `json:"sodium"`
}

// ActiveMedication represents an active medication
type ActiveMedication struct {
	MedicationCode    string  `json:"medication_code"`
	TherapeuticClass  string  `json:"therapeutic_class"`
	DoseAmount        float64 `json:"dose_amount"`
	FrequencyHours    int     `json:"frequency_hours"`
}



// Helper functions for creating JIT Safety contexts

// NewJitSafetyContext creates a new JIT Safety context from candidate proposal and patient data
func NewJitSafetyContext(
	candidate candidatebuilder.CandidateProposal,
	patientContext PatientContext,
	concurrentMeds []ConcurrentMed,
	kbVersions map[string]string,
	requestID string,
) *JitSafetyContext {
	return &JitSafetyContext{
		Patient: PatientCtx{
			AgeYears:   uint32(patientContext.Demographics.Age),
			Sex:        patientContext.Demographics.Gender,
			WeightKg:   patientContext.Demographics.Weight,
			HeightCm:   &patientContext.Demographics.Height,
			Pregnancy:  patientContext.Demographics.IsPregnant,
			Renal:      mapRenalContext(patientContext.LabResults),
			Hepatic:    mapHepaticContext(patientContext.LabResults),
			QTcMs:      mapQTcContext(patientContext.LabResults),
			Allergies:  mapAllergies(patientContext.Allergies),
			Conditions: mapConditions(patientContext.Conditions),
			Labs:       mapLabsContext(patientContext.LabResults),
		},
		ConcurrentMeds: concurrentMeds,
		Proposal: ProposedDose{
			DrugID:    candidate.MedicationCode,
			DoseMg:    0.0,  // Default dose, should be calculated elsewhere
			Route:     candidate.Route,
			IntervalH: 24,   // Default to once daily, should be calculated elsewhere
		},
		KBVersions: kbVersions,
		RequestID:  requestID,
	}
}

// Helper mapping functions (simplified implementations)
func mapRenalContext(labs LabResults) RenalCtx {
	var egfr, crcl *float64
	
	if labs.EGFR > 0 {
		egfr = &labs.EGFR
	}
	if labs.CreatinineClearance > 0 {
		crcl = &labs.CreatinineClearance
	}
	
	return RenalCtx{
		EGFR: egfr,
		CrCl: crcl,
	}
}

func mapHepaticContext(labs LabResults) HepaticCtx {
	// In a real implementation, this would derive Child-Pugh class from labs
	// For now, return empty context
	return HepaticCtx{}
}

func mapQTcContext(labs LabResults) *uint32 {
	// In a real implementation, this would extract QTc from ECG data
	// For now, return nil
	return nil
}

func mapAllergies(allergies []Allergy) []string {
	result := make([]string, len(allergies))
	for i, allergy := range allergies {
		result[i] = allergy.AllergenCode
	}
	return result
}

func mapConditions(conditions []Condition) []string {
	result := make([]string, len(conditions))
	for i, condition := range conditions {
		result[i] = condition.Code
	}
	return result
}

func mapLabsContext(labs LabResults) LabsCtx {
	var alt, ast, uacr *float64
	
	if labs.ALT > 0 {
		alt = &labs.ALT
	}
	if labs.AST > 0 {
		ast = &labs.AST
	}
	if labs.UACR > 0 {
		uacr = &labs.UACR
	}
	
	return LabsCtx{
		ALT:  alt,
		AST:  ast,
		UACR: uacr,
	}
}

// CalculateSafetyScore calculates a safety score from JIT Safety outcome
func CalculateSafetyScore(outcome JitSafetyOutcome) float64 {
	switch outcome.Decision {
	case DecisionAllow:
		return 1.0
	case DecisionAllowWithAdjustment:
		// Reduce score based on number of adjustments/warnings
		baseScore := 0.8
		penaltyPerReason := 0.05
		penaltyPerDDI := 0.1
		
		score := baseScore - (float64(len(outcome.Reasons)) * penaltyPerReason) - (float64(len(outcome.DDIs)) * penaltyPerDDI)
		if score < 0.1 {
			score = 0.1
		}
		return score
	case DecisionBlock:
		return 0.0
	default:
		return 0.0
	}
}
