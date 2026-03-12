package models

import "time"

// ORBDrivenResponse represents the response from ORB-driven Flow 2 execution
type ORBDrivenResponse struct {
	RequestID   string `json:"request_id"`
	PatientID   string `json:"patient_id"`
	
	// Intent Manifest information
	IntentManifest *IntentManifestResponse `json:"intent_manifest"`
	
	// Clinical context summary
	ClinicalContext *ClinicalContextSummary `json:"clinical_context"`
	
	// Medication proposal from Rust engine (legacy)
	MedicationProposal *MedicationProposal `json:"medication_proposal,omitempty"`

	// Enhanced proposal from Enhanced Proposal Generator (new)
	EnhancedProposal *EnhancedProposedOrder `json:"enhanced_proposal,omitempty"`
	
	// Overall assessment
	OverallStatus string `json:"overall_status"`
	
	// Execution summary
	ExecutionSummary *ORBExecutionSummary `json:"execution_summary"`
	
	// Performance metrics
	PerformanceMetrics *PerformanceMetrics `json:"performance_metrics"`
	
	Timestamp time.Time `json:"timestamp"`
}

// IntentManifestResponse represents the Intent Manifest in the response
type IntentManifestResponse struct {
	RecipeID          string    `json:"recipe_id"`
	DataRequirements  []string  `json:"data_requirements"`
	Priority          string    `json:"priority"`
	ClinicalRationale string    `json:"clinical_rationale"`
	RuleID            string    `json:"rule_id"`
	GeneratedAt       time.Time `json:"generated_at"`
}

// ClinicalContextSummary represents a summary of the clinical context
type ClinicalContextSummary struct {
	DataFieldsRetrieved int      `json:"data_fields_retrieved"`
	ContextSources      []string `json:"context_sources"`
	RetrievalTimeMs     int64    `json:"retrieval_time_ms"`
}

// MedicationProposal represents the medication proposal from Rust engine
type MedicationProposal struct {
	MedicationCode string  `json:"medication_code"`
	MedicationName string  `json:"medication_name"`
	
	// Dosing information
	CalculatedDose float64 `json:"calculated_dose"`
	DoseUnit       string  `json:"dose_unit"`
	Frequency      string  `json:"frequency"`
	Duration       string  `json:"duration,omitempty"`
	
	// Safety assessment
	SafetyStatus    string   `json:"safety_status"`
	SafetyAlerts    []string `json:"safety_alerts,omitempty"`
	Contraindications []string `json:"contraindications,omitempty"`
	
	// Clinical decision support
	ClinicalRationale string `json:"clinical_rationale"`
	MonitoringPlan    []string `json:"monitoring_plan,omitempty"`
	
	// Alternatives if needed
	Alternatives []MedicationAlternative `json:"alternatives,omitempty"`
	
	// Execution metadata
	ExecutionTimeMs int64 `json:"execution_time_ms"`
	RecipeVersion   string `json:"recipe_version,omitempty"`
}

// MedicationAlternative represents an alternative medication option
type MedicationAlternative struct {
	MedicationCode string  `json:"medication_code"`
	MedicationName string  `json:"medication_name"`
	Rationale      string  `json:"rationale"`
	SafetyProfile  string  `json:"safety_profile"`
}

// ORBExecutionSummary represents the ORB execution performance summary
type ORBExecutionSummary struct {
	TotalExecutionTimeMs  int64  `json:"total_execution_time_ms"`
	ORBEvaluationTimeMs   int64  `json:"orb_evaluation_time_ms"`
	ContextFetchTimeMs    int64  `json:"context_fetch_time_ms"`
	RecipeExecutionTimeMs int64  `json:"recipe_execution_time_ms"`
	NetworkHops           int    `json:"network_hops"`
	Engine                string `json:"engine"`
	Architecture          string `json:"architecture"`
}

// PerformanceMetrics represents detailed performance metrics
type PerformanceMetrics struct {
	CacheHitRate       float64 `json:"cache_hit_rate"`
	DataCompleteness   float64 `json:"data_completeness"`
	RuleEvaluationTime int64   `json:"rule_evaluation_time_ms"`
	TotalNetworkTime   int64   `json:"total_network_time_ms"`
}

// RecipeExecutionRequest represents a request to the Rust Recipe Engine
type RecipeExecutionRequest struct {
	RequestID       string           `json:"request_id"`
	RecipeID        string           `json:"recipe_id"`
	Variant         string           `json:"variant,omitempty"`
	ClinicalContext *ClinicalContext `json:"clinical_context"`
	PatientID       string           `json:"patient_id"`
	MedicationCode  string           `json:"medication_code"`
}

// ClinicalContext represents the clinical context data
type ClinicalContext struct {
	PatientID       string                 `json:"patient_id"`
	Fields          map[string]interface{} `json:"fields"`
	Sources         []string               `json:"sources"`
	RetrievalTimeMs int64                  `json:"retrieval_time_ms"`
	Completeness    float64                `json:"completeness"`
}

// ContextRequest represents a request to the Context Service
type ContextRequest struct {
	PatientID        string   `json:"patient_id"`
	DataRequirements []string `json:"data_requirements"`
	Priority         string   `json:"priority"`
	RequestID        string   `json:"request_id"`
	TimeoutMs        int      `json:"timeout_ms,omitempty"`
}
