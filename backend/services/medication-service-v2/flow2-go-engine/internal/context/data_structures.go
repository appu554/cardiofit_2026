package context

import (
	"time"
)

// CompleteContextPayload represents the complete clinical context for medication decisions
// This is the main output of the Context Integration Service
type CompleteContextPayload struct {
	// Patient clinical data
	Patient PatientContext `json:"patient"`
	
	// Clinical knowledge data
	Knowledge KnowledgeContext `json:"knowledge"`
	
	// Processing metadata
	Metadata ContextMetadata `json:"metadata"`
	
	// Data provenance information
	Provenance DataProvenance `json:"provenance"`
	
	// Cache information
	CacheInfo CacheInformation `json:"cache_info"`
}

// PatientContext contains all patient-specific clinical data
type PatientContext struct {
	// Demographics
	Demographics struct {
		Age       int     `json:"age"`
		WeightKg  float64 `json:"weight_kg"`
		HeightCm  float64 `json:"height_cm"`
		BMI       float64 `json:"bmi"`
		Gender    string  `json:"gender"`
	} `json:"demographics"`
	
	// Current medications
	Medications struct {
		Active    []ActiveMedication `json:"active"`
		Allergies []DrugAllergy      `json:"allergies"`
		History   []MedicationHistory `json:"history"`
	} `json:"medications"`
	
	// Medical conditions
	Conditions struct {
		Active   []ActiveCondition `json:"active"`
		History  []ConditionHistory `json:"history"`
		RiskFactors []RiskFactor   `json:"risk_factors"`
	} `json:"conditions"`
	
	// Laboratory results
	Labs struct {
		Recent []LabResult `json:"recent"`
		Trends []LabTrend  `json:"trends"`
	} `json:"labs"`
	
	// Vital signs
	Vitals struct {
		Current []VitalSign `json:"current"`
		Trends  []VitalTrend `json:"trends"`
	} `json:"vitals"`
}

// KnowledgeContext contains clinical knowledge from Knowledge Base services
type KnowledgeContext struct {
	// Drug interactions (from kb_ddi_v1)
	DrugInteractions struct {
		Interactions []DrugInteraction `json:"interactions"`
		Severity     string           `json:"max_severity"`
		Warnings     []InteractionWarning `json:"warnings"`
	} `json:"drug_interactions"`
	
	// Formulary information (from kb_formulary_stock_v1)
	FormularyInfo struct {
		Status       string              `json:"status"`
		Tier         int                 `json:"tier"`
		Alternatives []FormularyAlternative `json:"alternatives"`
		Restrictions []string            `json:"restrictions"`
	} `json:"formulary_info"`
	
	// Clinical guidelines (from kb_guideline_evidence_v1)
	Guidelines struct {
		Recommendations []GuidelineRecommendation `json:"recommendations"`
		EvidenceLevel   string                   `json:"evidence_level"`
		References      []string                 `json:"references"`
	} `json:"guidelines"`
	
	// Dosage information (from kb_dosing_rules_v1)
	Dosage struct {
		StandardDose    DoseRecommendation `json:"standard_dose"`
		AdjustedDose    DoseRecommendation `json:"adjusted_dose"`
		MaxDose         DoseRecommendation `json:"max_dose"`
		Adjustments     []DoseAdjustment   `json:"adjustments"`
	} `json:"dosage"`
	
	// Safety information (from kb_patient_safe_checks_v1)
	Safety struct {
		Contraindications []Contraindication `json:"contraindications"`
		Warnings          []SafetyWarning    `json:"warnings"`
		Monitoring        []MonitoringRequirement `json:"monitoring"`
	} `json:"safety"`
	
	// Monitoring protocols (from kb_drug_master_v1)
	Monitoring struct {
		Required   bool                    `json:"required"`
		Frequency  string                  `json:"frequency"`
		Parameters []MonitoringParameter   `json:"parameters"`
		Alerts     []MonitoringAlert       `json:"alerts"`
	} `json:"monitoring"`
	
	// Evidence and research (from kb_resistance_profiles_v1)
	Evidence struct {
		ResistanceProfiles []ResistanceProfile `json:"resistance_profiles"`
		LocalData          LocalResistanceData `json:"local_data"`
		Recommendations    []EvidenceRecommendation `json:"recommendations"`
	} `json:"evidence"`
}

// ContextMetadata contains processing information
type ContextMetadata struct {
	RequestID         string    `json:"request_id"`
	PatientID         string    `json:"patient_id"`
	ProcessingStarted time.Time `json:"processing_started"`
	ProcessingEnded   time.Time `json:"processing_ended"`
	ProcessingTimeMs  int64     `json:"processing_time_ms"`
	
	// Data completeness metrics
	Completeness struct {
		PatientDataScore    float64 `json:"patient_data_score"`
		KnowledgeDataScore  float64 `json:"knowledge_data_score"`
		OverallScore        float64 `json:"overall_score"`
		MissingFields       []string `json:"missing_fields"`
	} `json:"completeness"`
	
	// Quality metrics
	Quality struct {
		DataFreshness    time.Duration `json:"data_freshness"`
		SourceReliability float64      `json:"source_reliability"`
		ValidationErrors  []string     `json:"validation_errors"`
	} `json:"quality"`
	
	// Performance metrics
	Performance struct {
		CacheHitRate      float64 `json:"cache_hit_rate"`
		NetworkCallCount  int     `json:"network_call_count"`
		ParallelismFactor float64 `json:"parallelism_factor"`
	} `json:"performance"`
}

// DataProvenance tracks where data came from
type DataProvenance struct {
	PatientDataSources    []DataSource `json:"patient_data_sources"`
	KnowledgeDataSources  []DataSource `json:"knowledge_data_sources"`
	LastUpdated           time.Time    `json:"last_updated"`
	DataVersion           string       `json:"data_version"`
}

// CacheInformation contains cache-related metadata
type CacheInformation struct {
	CacheHit        bool      `json:"cache_hit"`
	CacheKey        string    `json:"cache_key"`
	CachedAt        time.Time `json:"cached_at"`
	TTL             int       `json:"ttl_seconds"`
	FreshnessScore  float64   `json:"freshness_score"`
	StaleServed     bool      `json:"stale_served"`
	RevalidationDue bool      `json:"revalidation_due"`
}

// Supporting data structures
type ActiveMedication struct {
	MedicationCode string    `json:"medication_code"`
	Name           string    `json:"name"`
	Dose           string    `json:"dose"`
	Frequency      string    `json:"frequency"`
	StartDate      time.Time `json:"start_date"`
	Prescriber     string    `json:"prescriber"`
}

type DrugAllergy struct {
	AllergenCode string `json:"allergen_code"`
	AllergenName string `json:"allergen_name"`
	Reaction     string `json:"reaction"`
	Severity     string `json:"severity"`
}

type MedicationHistory struct {
	MedicationCode string    `json:"medication_code"`
	Name           string    `json:"name"`
	StartDate      time.Time `json:"start_date"`
	EndDate        time.Time `json:"end_date"`
	Reason         string    `json:"reason"`
}

type ActiveCondition struct {
	ConditionCode string    `json:"condition_code"`
	Name          string    `json:"name"`
	Severity      string    `json:"severity"`
	DiagnosedDate time.Time `json:"diagnosed_date"`
	Status        string    `json:"status"`
}

type ConditionHistory struct {
	ConditionCode string    `json:"condition_code"`
	Name          string    `json:"name"`
	DiagnosedDate time.Time `json:"diagnosed_date"`
	ResolvedDate  time.Time `json:"resolved_date"`
}

type RiskFactor struct {
	Factor      string  `json:"factor"`
	RiskLevel   string  `json:"risk_level"`
	Score       float64 `json:"score"`
	Description string  `json:"description"`
}

type LabResult struct {
	TestCode   string    `json:"test_code"`
	TestName   string    `json:"test_name"`
	Value      float64   `json:"value"`
	Unit       string    `json:"unit"`
	RefRange   string    `json:"reference_range"`
	Status     string    `json:"status"`
	CollectedAt time.Time `json:"collected_at"`
}

type LabTrend struct {
	TestCode string      `json:"test_code"`
	TestName string      `json:"test_name"`
	Values   []LabResult `json:"values"`
	Trend    string      `json:"trend"`
}

type VitalSign struct {
	VitalType   string    `json:"vital_type"`
	Value       float64   `json:"value"`
	Unit        string    `json:"unit"`
	RecordedAt  time.Time `json:"recorded_at"`
	Status      string    `json:"status"`
}

type VitalTrend struct {
	VitalType string      `json:"vital_type"`
	Values    []VitalSign `json:"values"`
	Trend     string      `json:"trend"`
}

type DrugInteraction struct {
	Drug1         string `json:"drug1"`
	Drug2         string `json:"drug2"`
	Severity      string `json:"severity"`
	Description   string `json:"description"`
	Management    string `json:"management"`
	EvidenceLevel string `json:"evidence_level"`
}

type InteractionWarning struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	Action      string `json:"action"`
}

type FormularyAlternative struct {
	MedicationCode string  `json:"medication_code"`
	Name           string  `json:"name"`
	CostSavings    float64 `json:"cost_savings"`
	Equivalent     bool    `json:"therapeutic_equivalent"`
}

type GuidelineRecommendation struct {
	Organization   string `json:"organization"`
	Recommendation string `json:"recommendation"`
	Strength       string `json:"strength"`
	EvidenceLevel  string `json:"evidence_level"`
}

type DoseRecommendation struct {
	Amount    float64 `json:"amount"`
	Unit      string  `json:"unit"`
	Frequency string  `json:"frequency"`
	Route     string  `json:"route"`
}

type DoseAdjustment struct {
	Condition   string             `json:"condition"`
	Adjustment  DoseRecommendation `json:"adjustment"`
	Rationale   string             `json:"rationale"`
}

type Contraindication struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Override    bool   `json:"override_possible"`
}

type SafetyWarning struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	Action      string `json:"recommended_action"`
}

type MonitoringRequirement struct {
	Parameter string `json:"parameter"`
	Frequency string `json:"frequency"`
	Target    string `json:"target_range"`
	Action    string `json:"action_if_abnormal"`
}

type MonitoringParameter struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	NormalRange  interface{} `json:"normal_range"`
	CriticalRange interface{} `json:"critical_range"`
}

type MonitoringAlert struct {
	AlertID   string `json:"alert_id"`
	Condition string `json:"condition"`
	Severity  string `json:"severity"`
	Action    string `json:"action"`
	Message   string `json:"message"`
}

type ResistanceProfile struct {
	Organism      string  `json:"organism"`
	Antibiotic    string  `json:"antibiotic"`
	Resistance    float64 `json:"resistance_percentage"`
	LocalData     bool    `json:"local_data"`
	LastUpdated   time.Time `json:"last_updated"`
}

type LocalResistanceData struct {
	Institution   string    `json:"institution"`
	DataPeriod    string    `json:"data_period"`
	SampleSize    int       `json:"sample_size"`
	LastUpdated   time.Time `json:"last_updated"`
}

type EvidenceRecommendation struct {
	Recommendation string `json:"recommendation"`
	EvidenceLevel  string `json:"evidence_level"`
	Source         string `json:"source"`
	LastReviewed   time.Time `json:"last_reviewed"`
}

type DataSource struct {
	SourceID    string    `json:"source_id"`
	SourceName  string    `json:"source_name"`
	SourceType  string    `json:"source_type"`
	LastUpdated time.Time `json:"last_updated"`
	Reliability float64   `json:"reliability_score"`
}
