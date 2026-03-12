package models

import (
	"time"
)

// ============================================================================
// Clinical Alert Types for CDSS
// ============================================================================
// CDSSAlert represents a clinical decision support alert generated from
// evaluating patient facts against clinical value sets.

// CDSSAlertSeverity represents the severity level of a clinical alert
type CDSSAlertSeverity string

const (
	SeverityCritical CDSSAlertSeverity = "critical" // Immediate action required
	SeverityHigh     CDSSAlertSeverity = "high"     // Urgent attention needed
	SeverityModerate CDSSAlertSeverity = "moderate" // Action recommended
	SeverityLow      CDSSAlertSeverity = "low"      // Informational
)

// SeverityPriority returns the priority order (lower = more urgent)
func (s CDSSAlertSeverity) Priority() int {
	switch s {
	case SeverityCritical:
		return 1
	case SeverityHigh:
		return 2
	case SeverityModerate:
		return 3
	case SeverityLow:
		return 4
	default:
		return 5
	}
}

// String returns the string representation
func (s CDSSAlertSeverity) String() string {
	return string(s)
}

// MatchType represents how a code matched against a value set
type MatchType string

const (
	MatchTypeExact       MatchType = "exact"       // Direct O(1) hash match
	MatchTypeSubsumption MatchType = "subsumption" // Hierarchical IS-A match via Neo4j
	MatchTypeExpansion   MatchType = "expansion"   // Found in expanded value set
)

// AlertEvidence represents the evidence supporting a clinical alert
type AlertEvidence struct {
	// The clinical fact that triggered this alert
	FactID   string   `json:"fact_id"`
	FactType FactType `json:"fact_type"`

	// The code that matched
	Code    string `json:"code"`
	System  string `json:"system"`
	Display string `json:"display"`

	// The value set it matched against
	ValueSetID   string `json:"value_set_id"`
	ValueSetName string `json:"value_set_name"`

	// How the match was found
	MatchType MatchType `json:"match_type"`

	// Additional match details
	MatchedCode    string  `json:"matched_code,omitempty"`    // The code it matched (for subsumption)
	MatchedDisplay string  `json:"matched_display,omitempty"` // Display of matched code
	Confidence     float64 `json:"confidence,omitempty"`      // Match confidence (0-1)

	// Numeric value context (for observations/labs)
	NumericValue     *float64 `json:"numeric_value,omitempty"`
	Unit             string   `json:"unit,omitempty"`
	ReferenceRangeLow  *float64 `json:"reference_range_low,omitempty"`
	ReferenceRangeHigh *float64 `json:"reference_range_high,omitempty"`
	IsAbnormal       bool     `json:"is_abnormal,omitempty"`
}

// CDSSAlert represents a clinical decision support alert
type CDSSAlert struct {
	// Unique identifier for this alert
	AlertID string `json:"alert_id"`

	// Severity level
	Severity CDSSAlertSeverity `json:"severity"`

	// Clinical domain
	ClinicalDomain ClinicalDomain `json:"clinical_domain"`

	// Alert content
	Title       string `json:"title"`
	Description string `json:"description"`

	// Evidence supporting this alert
	Evidence []AlertEvidence `json:"evidence"`

	// Clinical recommendations
	Recommendations []string `json:"recommendations,omitempty"`

	// Links to clinical guidelines or protocols
	GuidelineLinks []string `json:"guideline_links,omitempty"`

	// ICD-10 or SNOMED codes associated with this alert type
	AssociatedCodes []FactCoding `json:"associated_codes,omitempty"`

	// When the alert was generated
	GeneratedAt time.Time `json:"generated_at"`

	// Expiration (if applicable)
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	// Alert status
	Status string `json:"status,omitempty"` // active, acknowledged, resolved

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// IsHighPriority returns true if the alert is critical or high severity
func (a *CDSSAlert) IsHighPriority() bool {
	return a.Severity == SeverityCritical || a.Severity == SeverityHigh
}

// AddEvidence adds evidence to the alert
func (a *CDSSAlert) AddEvidence(evidence AlertEvidence) {
	a.Evidence = append(a.Evidence, evidence)
}

// AddRecommendation adds a recommendation to the alert
func (a *CDSSAlert) AddRecommendation(recommendation string) {
	a.Recommendations = append(a.Recommendations, recommendation)
}

// ============================================================================
// Evaluation Result Types
// ============================================================================

// EvaluationResult represents the result of evaluating a single clinical fact
type EvaluationResult struct {
	// The fact that was evaluated
	FactID   string   `json:"fact_id"`
	FactType FactType `json:"fact_type"`
	Code     string   `json:"code"`
	System   string   `json:"system"`
	Display  string   `json:"display"`

	// Whether the fact matched any value sets
	Matched bool `json:"matched"`

	// All value sets that matched
	MatchedValueSets []ValueSetMatch `json:"matched_value_sets,omitempty"`

	// Processing details
	EvaluationTimeMs float64 `json:"evaluation_time_ms"`
	PipelineStep     string  `json:"pipeline_step,omitempty"` // expansion, exact, subsumption

	// Error if evaluation failed
	Error string `json:"error,omitempty"`
}

// ValueSetMatch represents a match against a specific value set
type ValueSetMatch struct {
	ValueSetID   string            `json:"value_set_id"`
	ValueSetName string            `json:"value_set_name"`
	MatchType    MatchType         `json:"match_type"`
	MatchedCode  string            `json:"matched_code,omitempty"`
	Confidence   float64           `json:"confidence,omitempty"`
	Domain       ClinicalDomain    `json:"domain,omitempty"`
}

// ============================================================================
// CDSS Evaluation Request/Response
// ============================================================================

// CDSSEvaluationOptions configures the CDSS evaluation behavior
type CDSSEvaluationOptions struct {
	// Enable subsumption testing (Neo4j hierarchical matching)
	EnableSubsumption bool `json:"enable_subsumption"`

	// Generate alerts from evaluation results
	GenerateAlerts bool `json:"generate_alerts"`

	// Evaluate clinical rules (compound conditions, thresholds)
	EvaluateRules bool `json:"evaluate_rules"`

	// Filter by clinical domains (empty = all domains)
	ClinicalDomains []string `json:"clinical_domains,omitempty"`

	// Filter by value set IDs (empty = all value sets)
	ValueSetIDs []string `json:"value_set_ids,omitempty"`

	// Maximum number of value sets to evaluate per fact
	MaxValueSetsPerFact int `json:"max_value_sets_per_fact,omitempty"`

	// Stop on first match (faster but less comprehensive)
	StopOnFirstMatch bool `json:"stop_on_first_match"`

	// Include detailed evaluation results
	IncludeDetails bool `json:"include_details"`

	// Alert generation options
	MinimumAlertSeverity CDSSAlertSeverity `json:"minimum_alert_severity,omitempty"`
	GroupAlertsByDomain  bool              `json:"group_alerts_by_domain"`
}

// DefaultCDSSEvaluationOptions returns sensible default options
func DefaultCDSSEvaluationOptions() *CDSSEvaluationOptions {
	return &CDSSEvaluationOptions{
		EnableSubsumption:    true,
		GenerateAlerts:       true,
		EvaluateRules:        true,
		IncludeDetails:       false,
		StopOnFirstMatch:     false,
		GroupAlertsByDomain:  true,
		MinimumAlertSeverity: SeverityLow,
	}
}

// CDSSEvaluationRequest represents a request to evaluate patient data
type CDSSEvaluationRequest struct {
	// Patient identifier
	PatientID string `json:"patient_id" binding:"required"`

	// Optional encounter context
	EncounterID string `json:"encounter_id,omitempty"`

	// Pre-built facts (if already extracted)
	FactSet *PatientFactSet `json:"fact_set,omitempty"`

	// FHIR Bundle for fact extraction
	Bundle *FHIRBundle `json:"bundle,omitempty"`

	// Individual FHIR resources
	Conditions   []FHIRCondition          `json:"conditions,omitempty"`
	Observations []FHIRObservation        `json:"observations,omitempty"`
	Medications  []FHIRMedicationRequest  `json:"medications,omitempty"`
	Procedures   []FHIRProcedure          `json:"procedures,omitempty"`
	Allergies    []FHIRAllergyIntolerance `json:"allergies,omitempty"`

	// Fact builder options (if extracting from resources)
	FactBuilderOptions *FactBuilderOptions `json:"fact_builder_options,omitempty"`

	// Evaluation options
	Options *CDSSEvaluationOptions `json:"options,omitempty"`
}

// HasFactSet returns true if pre-built facts are provided
func (r *CDSSEvaluationRequest) HasFactSet() bool {
	return r.FactSet != nil && r.FactSet.TotalFacts > 0
}

// HasResources returns true if FHIR resources are provided
func (r *CDSSEvaluationRequest) HasResources() bool {
	return r.Bundle != nil ||
		len(r.Conditions) > 0 ||
		len(r.Observations) > 0 ||
		len(r.Medications) > 0 ||
		len(r.Procedures) > 0 ||
		len(r.Allergies) > 0
}

// CDSSEvaluationResponse represents the result of CDSS evaluation
type CDSSEvaluationResponse struct {
	// Unique evaluation identifier
	EvaluationID string `json:"evaluation_id"`

	// Success indicator
	Success bool `json:"success"`

	// Patient context
	PatientID   string `json:"patient_id"`
	EncounterID string `json:"encounter_id,omitempty"`

	// Summary statistics
	FactsExtracted  int `json:"facts_extracted"`
	FactsEvaluated  int `json:"facts_evaluated"`
	RulesEvaluated  int `json:"rules_evaluated"`
	RulesFired      int `json:"rules_fired"`
	MatchesFound    int `json:"matches_found"`
	AlertsGenerated int `json:"alerts_generated"`

	// Generated alerts (sorted by severity)
	Alerts []CDSSAlert `json:"alerts,omitempty"`

	// Detailed evaluation results (if requested)
	EvaluationResults []EvaluationResult `json:"evaluation_results,omitempty"`

	// Matched clinical domains
	MatchedDomains []ClinicalDomain `json:"matched_domains,omitempty"`

	// Processing metadata
	ExecutionTimeMs float64 `json:"execution_time_ms"`
	PipelineUsed    string  `json:"pipeline_used,omitempty"` // THREE-CHECK, TWO-CHECK

	// Errors and warnings
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// ============================================================================
// Alert Generation Request/Response
// ============================================================================

// AlertGenerationRequest represents a request to generate alerts from evaluation results
type AlertGenerationRequest struct {
	// Patient context
	PatientID   string `json:"patient_id"`
	EncounterID string `json:"encounter_id,omitempty"`

	// Evaluation results to generate alerts from
	EvaluationResults []EvaluationResult `json:"evaluation_results"`

	// Patient fact set (for additional context)
	FactSet *PatientFactSet `json:"fact_set,omitempty"`

	// Alert generation options
	Options *AlertGenerationOptions `json:"options,omitempty"`
}

// AlertGenerationOptions configures alert generation behavior
type AlertGenerationOptions struct {
	// Minimum severity to generate
	MinimumSeverity CDSSAlertSeverity `json:"minimum_severity"`

	// Group alerts by clinical domain
	GroupByDomain bool `json:"group_by_domain"`

	// Include recommendations
	IncludeRecommendations bool `json:"include_recommendations"`

	// Include guideline links
	IncludeGuidelines bool `json:"include_guidelines"`

	// Maximum alerts per domain
	MaxAlertsPerDomain int `json:"max_alerts_per_domain,omitempty"`

	// Merge similar alerts
	MergeSimilarAlerts bool `json:"merge_similar_alerts"`
}

// DefaultAlertGenerationOptions returns sensible default options
func DefaultAlertGenerationOptions() *AlertGenerationOptions {
	return &AlertGenerationOptions{
		MinimumSeverity:        SeverityLow,
		GroupByDomain:          true,
		IncludeRecommendations: true,
		IncludeGuidelines:      false,
		MergeSimilarAlerts:     true,
	}
}

// AlertGenerationResponse represents the result of alert generation
type AlertGenerationResponse struct {
	// Success indicator
	Success bool `json:"success"`

	// Generated alerts
	Alerts []CDSSAlert `json:"alerts"`

	// Summary by domain
	AlertsByDomain map[ClinicalDomain]int `json:"alerts_by_domain"`

	// Summary by severity
	AlertsBySeverity map[CDSSAlertSeverity]int `json:"alerts_by_severity"`

	// Total counts
	TotalAlerts    int `json:"total_alerts"`
	CriticalAlerts int `json:"critical_alerts"`
	HighAlerts     int `json:"high_alerts"`

	// Processing time
	ProcessingTimeMs float64 `json:"processing_time_ms"`

	// Errors
	Errors []string `json:"errors,omitempty"`
}

// ============================================================================
// Value Set Clinical Severity Mapping
// ============================================================================

// ValueSetSeverityMapping maps value sets to their default alert severity
var ValueSetSeverityMapping = map[string]CDSSAlertSeverity{
	// Critical - Immediate life threats
	"SepsisDiagnosis":      SeverityCritical,
	"AUSepsisConditions":   SeverityCritical,
	"SepsisIndicators":     SeverityCritical,
	"AcuteCoronarySyndrome": SeverityCritical,
	"Stroke":               SeverityCritical,
	"RespiratoryFailure":   SeverityCritical,

	// High - Urgent conditions
	"AcuteRenalFailure":   SeverityHigh,
	"AUAKIConditions":     SeverityHigh,
	"HeartFailure":        SeverityHigh,
	"Pneumonia":           SeverityHigh,
	"GastrointestinalBleeding": SeverityHigh,
	"Hypoglycemia":        SeverityHigh,
	"AlteredMentalStatus": SeverityHigh,
	"Seizure":             SeverityHigh,

	// Moderate - Significant conditions requiring attention
	"DiabetesMellitus":       SeverityModerate,
	"Hypertension":           SeverityModerate,
	"ChronicKidneyDisease":   SeverityModerate,
	"COPD":                   SeverityModerate,
	"Asthma":                 SeverityModerate,
	"CardiacArrhythmias":     SeverityModerate,
	"Anemia":                 SeverityModerate,
	"Coagulopathy":           SeverityModerate,
	"ElectrolyteDisorders":   SeverityModerate,
	"LiverDisease":           SeverityModerate,

	// Low - Informational
	"MetabolicSyndrome":      SeverityLow,
	"ThyroidDisorders":       SeverityLow,
}

// GetValueSetSeverity returns the default severity for a value set
func GetValueSetSeverity(valueSetID string) CDSSAlertSeverity {
	if severity, ok := ValueSetSeverityMapping[valueSetID]; ok {
		return severity
	}
	return SeverityModerate // Default to moderate if not explicitly mapped
}

// ============================================================================
// Clinical Indicator Definitions
// ============================================================================

// ClinicalIndicator represents a clinical condition indicator
type ClinicalIndicator struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Domain      ClinicalDomain    `json:"domain"`
	Severity    CDSSAlertSeverity `json:"severity"`
	ValueSets   []string          `json:"value_sets"`
	Recommendations []string      `json:"recommendations"`
}

// ClinicalIndicatorRegistry contains all known clinical indicators
var ClinicalIndicatorRegistry = map[string]ClinicalIndicator{
	"sepsis": {
		ID:          "sepsis",
		Name:        "Sepsis Indicator",
		Description: "Patient has indicators of sepsis or septic shock",
		Domain:      DomainSepsis,
		Severity:    SeverityCritical,
		ValueSets:   []string{"SepsisDiagnosis", "AUSepsisConditions", "SepsisIndicators"},
		Recommendations: []string{
			"Consider Sepsis-3 criteria evaluation",
			"Obtain blood cultures before antibiotics if possible",
			"Initiate broad-spectrum antibiotics within 1 hour",
			"Obtain lactate level",
			"Begin fluid resuscitation if hypotensive",
			"Assess for source of infection",
		},
	},
	"aki": {
		ID:          "aki",
		Name:        "Acute Kidney Injury",
		Description: "Patient has indicators of acute kidney injury",
		Domain:      DomainRenal,
		Severity:    SeverityHigh,
		ValueSets:   []string{"AcuteRenalFailure", "AUAKIConditions"},
		Recommendations: []string{
			"Review and adjust nephrotoxic medications",
			"Monitor urine output and fluid balance",
			"Order renal function panel",
			"Consider nephrology consultation",
			"Assess for reversible causes",
		},
	},
	"heart_failure": {
		ID:          "heart_failure",
		Name:        "Heart Failure",
		Description: "Patient has heart failure indicators",
		Domain:      DomainCardiac,
		Severity:    SeverityHigh,
		ValueSets:   []string{"HeartFailure"},
		Recommendations: []string{
			"Assess volume status",
			"Review diuretic therapy",
			"Monitor daily weights",
			"Order BNP/NT-proBNP if not recent",
			"Ensure guideline-directed medical therapy",
		},
	},
	"diabetes": {
		ID:          "diabetes",
		Name:        "Diabetes Management",
		Description: "Patient has diabetes mellitus",
		Domain:      DomainMetabolic,
		Severity:    SeverityModerate,
		ValueSets:   []string{"DiabetesMellitus"},
		Recommendations: []string{
			"Review glucose monitoring frequency",
			"Assess HbA1c if not recent",
			"Review medication adherence",
			"Screen for complications (nephropathy, retinopathy)",
		},
	},
	"respiratory_failure": {
		ID:          "respiratory_failure",
		Name:        "Respiratory Failure",
		Description: "Patient has respiratory failure indicators",
		Domain:      DomainRespiratory,
		Severity:    SeverityCritical,
		ValueSets:   []string{"RespiratoryFailure"},
		Recommendations: []string{
			"Assess oxygen saturation and respiratory rate",
			"Consider arterial blood gas",
			"Evaluate need for supplemental oxygen or ventilation",
			"Identify underlying cause",
		},
	},
}

// GetClinicalIndicator returns the clinical indicator for a given ID
func GetClinicalIndicator(id string) *ClinicalIndicator {
	if indicator, ok := ClinicalIndicatorRegistry[id]; ok {
		return &indicator
	}
	return nil
}

// GetIndicatorsByDomain returns all indicators for a clinical domain
func GetIndicatorsByDomain(domain ClinicalDomain) []ClinicalIndicator {
	var indicators []ClinicalIndicator
	for _, indicator := range ClinicalIndicatorRegistry {
		if indicator.Domain == domain {
			indicators = append(indicators, indicator)
		}
	}
	return indicators
}
