package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// DrugInteraction represents a drug-drug interaction
type DrugInteraction struct {
	ID                  uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	InteractionID       string    `gorm:"size:100;uniqueIndex;not null" json:"interaction_id"`
	
	// Drug information
	DrugACode           string    `gorm:"size:100;not null;index" json:"drug_a_code"`
	DrugAName           string    `gorm:"size:200;not null" json:"drug_a_name"`
	DrugBCode           string    `gorm:"size:100;not null;index" json:"drug_b_code"`
	DrugBName           string    `gorm:"size:200;not null" json:"drug_b_name"`
	
	// Interaction classification
	InteractionType     string    `gorm:"size:50;not null;index" json:"interaction_type"`
	Severity            string    `gorm:"size:20;not null;index;check:severity IN ('contraindicated', 'major', 'moderate', 'minor')" json:"severity"`
	EvidenceLevel       string    `gorm:"size:20;not null;check:evidence_level IN ('established', 'probable', 'theoretical', 'unknown')" json:"evidence_level"`
	
	// Clinical details
	Mechanism           string    `gorm:"type:text;not null" json:"mechanism"`
	ClinicalEffect      string    `gorm:"type:text;not null" json:"clinical_effect"`
	TimeToOnset         *string   `gorm:"size:50" json:"time_to_onset,omitempty"`
	Duration            *string   `gorm:"size:50" json:"duration,omitempty"`
	
	// Management recommendations
	ManagementStrategy  string    `gorm:"type:text;not null" json:"management_strategy"`
	MonitoringParameters *JSONB   `gorm:"type:jsonb" json:"monitoring_parameters,omitempty"`
	DoseAdjustmentRequired bool   `gorm:"default:false" json:"dose_adjustment_required"`
	AlternativeDrugs    *JSONB    `gorm:"type:jsonb" json:"alternative_drugs,omitempty"`
	
	// Documentation
	LiteratureReferences *JSONB   `gorm:"type:jsonb" json:"literature_references,omitempty"`
	DocumentationLevel  *string   `gorm:"size:20;check:documentation_level IN ('excellent', 'good', 'fair', 'poor')" json:"documentation_level,omitempty"`
	
	// Clinical context
	PatientPopulations  *JSONB    `gorm:"type:jsonb" json:"patient_populations,omitempty"`
	ComorbidityFactors  *JSONB    `gorm:"type:jsonb" json:"comorbidity_factors,omitempty"`
	
	// Metadata
	Active              bool      `gorm:"default:true;index" json:"active"`
	VerifiedBy          *string   `gorm:"size:100" json:"verified_by,omitempty"`
	VerifiedAt          *time.Time `json:"verified_at,omitempty"`
	FrequencyScore      *decimal.Decimal `gorm:"type:decimal(5,4)" json:"frequency_score,omitempty"`
	ClinicalSignificance *decimal.Decimal `gorm:"type:decimal(5,4)" json:"clinical_significance,omitempty"`
	
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	CreatedBy           *string   `gorm:"size:100" json:"created_by,omitempty"`
	UpdatedBy           *string   `gorm:"size:100" json:"updated_by,omitempty"`

	// Governance provenance - Three-Layer Authority Model
	// Provides medico-legal defensibility by tracking WHO says this interaction exists
	// See DDIGovernance struct for Layer 1 (Regulatory), Layer 2 (Pharmacology), Layer 3 (Clinical)
	Governance DDIGovernance `json:"governance" gorm:"embedded;embeddedPrefix:gov_"`
}

// DrugSynonym represents drug code mappings and synonyms
type DrugSynonym struct {
	ID                  uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PrimaryDrugCode     string    `gorm:"size:100;not null;index" json:"primary_drug_code"`
	SynonymCode         string    `gorm:"size:100;not null;index" json:"synonym_code"`
	SynonymName         string    `gorm:"size:200;not null" json:"synonym_name"`
	SynonymType         string    `gorm:"size:50;not null;index" json:"synonym_type"`
	MappingConfidence   decimal.Decimal `gorm:"type:decimal(3,2);not null;default:1.0" json:"mapping_confidence"`
	Active              bool      `gorm:"default:true" json:"active"`
	CreatedAt           time.Time `json:"created_at"`
}

// InteractionPattern represents patterns for similar drug classes
type InteractionPattern struct {
	ID                  uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatternName         string    `gorm:"size:200;uniqueIndex;not null" json:"pattern_name"`
	DrugClassA          string    `gorm:"size:100;not null" json:"drug_class_a"`
	DrugClassB          string    `gorm:"size:100;not null" json:"drug_class_b"`
	InteractionTemplate JSONB     `gorm:"type:jsonb;not null" json:"interaction_template"`
	SeverityDefault     string    `gorm:"size:20;not null" json:"severity_default"`
	MechanismTemplate   string    `gorm:"type:text;not null" json:"mechanism_template"`
	ManagementTemplate  string    `gorm:"type:text;not null" json:"management_template"`
	Active              bool      `gorm:"default:true" json:"active"`
	CreatedAt           time.Time `json:"created_at"`
}

// PatientInteractionAlert represents patient-specific interaction alerts
type PatientInteractionAlert struct {
	ID                  uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID           string    `gorm:"size:100;not null;index" json:"patient_id"`
	InteractionID       string    `gorm:"size:100;not null;index" json:"interaction_id"` // References drug_interactions.interaction_id
	
	// Alert context
	AlertTriggeredAt    time.Time `gorm:"default:NOW();index" json:"alert_triggered_at"`
	AlertSeverity       string    `gorm:"size:20;not null" json:"alert_severity"`
	ClinicalContext     *JSONB    `gorm:"type:jsonb" json:"clinical_context,omitempty"`
	
	// Drug details at time of alert
	DrugRegimen         JSONB     `gorm:"type:jsonb;not null" json:"drug_regimen"`
	PrescribingPhysician *string  `gorm:"size:100" json:"prescribing_physician,omitempty"`
	OrderingSystem      *string   `gorm:"size:100" json:"ordering_system,omitempty"`
	
	// Alert disposition
	AlertStatus         string    `gorm:"size:20;default:'active';check:alert_status IN ('active', 'acknowledged', 'overridden', 'resolved')" json:"alert_status"`
	AcknowledgedBy      *string   `gorm:"size:100" json:"acknowledged_by,omitempty"`
	AcknowledgedAt      *time.Time `json:"acknowledged_at,omitempty"`
	OverrideReason      *string   `gorm:"type:text" json:"override_reason,omitempty"`
	ResolutionAction    *string   `gorm:"type:text" json:"resolution_action,omitempty"`
	
	// Outcomes
	ClinicalOutcome     *string   `gorm:"size:50" json:"clinical_outcome,omitempty"`
	OutcomeDescription  *string   `gorm:"type:text" json:"outcome_description,omitempty"`
	OutcomeAssessedAt   *time.Time `json:"outcome_assessed_at,omitempty"`
	
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// InteractionRule represents configurable interaction checking rules
type InteractionRule struct {
	ID                  uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	RuleName            string    `gorm:"size:200;uniqueIndex;not null" json:"rule_name"`
	RuleType            string    `gorm:"size:50;not null;index" json:"rule_type"`
	
	// Rule conditions
	Conditions          JSONB     `gorm:"type:jsonb;not null" json:"conditions"`
	DrugCriteria        JSONB     `gorm:"type:jsonb;not null" json:"drug_criteria"`
	PatientCriteria     *JSONB    `gorm:"type:jsonb" json:"patient_criteria,omitempty"`
	
	// Rule actions
	ActionType          string    `gorm:"size:50;not null" json:"action_type"`
	AlertSeverity       *string   `gorm:"size:20" json:"alert_severity,omitempty"`
	AlertMessage        *string   `gorm:"type:text" json:"alert_message,omitempty"`
	RecommendedAction   *string   `gorm:"type:text" json:"recommended_action,omitempty"`
	
	// Rule metadata
	Priority            int       `gorm:"default:100;index" json:"priority"`
	Active              bool      `gorm:"default:true;index" json:"active"`
	EffectiveDate       time.Time `gorm:"default:NOW()" json:"effective_date"`
	ExpiryDate          *time.Time `json:"expiry_date,omitempty"`
	
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	CreatedBy           *string   `gorm:"size:100" json:"created_by,omitempty"`
}

// DrugFormulation represents drug formulation and route-specific data
type DrugFormulation struct {
	ID                      uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	DrugCode                string    `gorm:"size:100;not null;index" json:"drug_code"`
	FormulationType         string    `gorm:"size:50;not null" json:"formulation_type"`
	RouteOfAdministration   string    `gorm:"size:50;not null;index" json:"route_of_administration"`
	Strength                *string   `gorm:"size:100" json:"strength,omitempty"`
	
	// Bioavailability and pharmacokinetics
	Bioavailability         *decimal.Decimal `gorm:"type:decimal(5,4)" json:"bioavailability,omitempty"`
	HalfLifeHours           *decimal.Decimal `gorm:"type:decimal(8,2)" json:"half_life_hours,omitempty"`
	TimeToPeakHours         *decimal.Decimal `gorm:"type:decimal(6,2)" json:"time_to_peak_hours,omitempty"`
	
	// Interaction modifiers
	InteractionModifiers    *JSONB    `gorm:"type:jsonb" json:"interaction_modifiers,omitempty"`
	
	Active                  bool      `gorm:"default:true" json:"active"`
	CreatedAt               time.Time `json:"created_at"`
}

// CDSConfiguration represents clinical decision support configurations
type CDSConfiguration struct {
	ID                  uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ConfigName          string    `gorm:"size:100;uniqueIndex;not null" json:"config_name"`
	HealthcareSystem    string    `gorm:"size:100;not null" json:"healthcare_system"`
	
	// Alerting thresholds
	SeverityThresholds  JSONB     `gorm:"type:jsonb;not null" json:"severity_thresholds"`
	
	// Alert customization
	AlertTemplates      *JSONB    `gorm:"type:jsonb" json:"alert_templates,omitempty"`
	OverridePermissions *JSONB    `gorm:"type:jsonb" json:"override_permissions,omitempty"`
	
	// Integration settings
	ExternalSystems     *JSONB    `gorm:"type:jsonb" json:"external_systems,omitempty"`
	
	Active              bool      `gorm:"default:true" json:"active"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// InteractionAnalytics represents interaction usage analytics
type InteractionAnalytics struct {
	ID                      uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	AnalysisDate            time.Time `gorm:"type:date;not null;index" json:"analysis_date"`
	InteractionID           *string   `gorm:"size:100;index" json:"interaction_id,omitempty"`
	
	// Usage statistics
	AlertFrequency          int       `gorm:"default:0" json:"alert_frequency"`
	OverrideFrequency       int       `gorm:"default:0" json:"override_frequency"`
	ClinicalEvents          int       `gorm:"default:0" json:"clinical_events"`
	
	// Patient demographics
	AgeGroupDistribution    *JSONB    `gorm:"type:jsonb" json:"age_group_distribution,omitempty"`
	GenderDistribution      *JSONB    `gorm:"type:jsonb" json:"gender_distribution,omitempty"`
	ComorbidityPatterns     *JSONB    `gorm:"type:jsonb" json:"comorbidity_patterns,omitempty"`
	
	// Prescribing patterns
	PrescriberSpecialties   *JSONB    `gorm:"type:jsonb" json:"prescriber_specialties,omitempty"`
	HealthcareSettings      *JSONB    `gorm:"type:jsonb" json:"healthcare_settings,omitempty"`
	
	// Outcomes
	AdverseEventRate        *decimal.Decimal `gorm:"type:decimal(5,4)" json:"adverse_event_rate,omitempty"`
	HospitalizationRate     *decimal.Decimal `gorm:"type:decimal(5,4)" json:"hospitalization_rate,omitempty"`
	
	CreatedAt               time.Time `json:"created_at"`
}

// Batch processing models

// BatchInteractionRequest represents a single batch request
type BatchInteractionRequest struct {
	RequestID       string   `json:"request_id" binding:"required"`
	DrugCodes       []string `json:"drug_codes" binding:"required,min=2"`
	SeverityFilter  []string `json:"severity_filter,omitempty"`
	PatientContext  map[string]interface{} `json:"patient_context,omitempty"`
}

// BatchInteractionCheckRequest represents a batch of interaction check requests
type BatchInteractionCheckRequest struct {
	Requests []InteractionCheckRequest `json:"requests" binding:"required,min=1,max=100"`
	Options  BatchCheckOptions         `json:"options,omitempty"`
}

// BatchCheckOptions represents options for batch processing
type BatchCheckOptions struct {
	Parallel         bool   `json:"parallel" default:"true"`
	MaxConcurrency   int    `json:"max_concurrency" default:"10"`
	SeverityFilter   []string `json:"severity_filter,omitempty"`
	IncludeStatistics bool   `json:"include_statistics" default:"false"`
}

// BatchInteractionResult represents the result of a single batch request
type BatchInteractionResult struct {
	RequestID    string              `json:"request_id"`
	Interactions []InteractionResult `json:"interactions"`
	Summary      InteractionSummary  `json:"summary"`
	ProcessedAt  time.Time           `json:"processed_at"`
	Error        string              `json:"error,omitempty"`
}

// MatrixStatistics represents performance and usage statistics for the interaction matrix
type MatrixStatistics struct {
	TotalDrugs        int       `json:"total_drugs"`
	TotalInteractions int       `json:"total_interactions"`
	MatrixDensity     float64   `json:"matrix_density"`
	LastUpdated       time.Time `json:"last_updated"`
	MemoryUsageMB     float64   `json:"memory_usage_mb"`
	LookupPerformance PerformanceMetrics `json:"lookup_performance,omitempty"`
}

// PerformanceMetrics represents performance metrics for matrix operations
type PerformanceMetrics struct {
	AverageLookupTimeNs int64 `json:"average_lookup_time_ns"`
	CacheHitRate        float64 `json:"cache_hit_rate"`
	TotalLookups        int64 `json:"total_lookups"`
	BatchProcessingRate float64 `json:"batch_processing_rate"` // interactions/second
}

// Request/Response models

// InteractionCheckRequest represents a request to check for drug interactions
type InteractionCheckRequest struct {
	PatientID           string                 `json:"patient_id,omitempty"`
	DrugCodes           []string               `json:"drug_codes" binding:"required,min=2"`
	CheckType           string                 `json:"check_type,omitempty"` // "comprehensive", "new_drug", "update"
	ClinicalContext     map[string]interface{} `json:"clinical_context,omitempty"`
	SeverityFilter      []string               `json:"severity_filter,omitempty"`
	IncludeAlternatives bool                   `json:"include_alternatives"`
	IncludeMonitoring   bool                   `json:"include_monitoring"`
}

// InteractionCheckResponse represents the response for interaction checking
type InteractionCheckResponse struct {
	PatientID           string                      `json:"patient_id,omitempty"`
	CheckedDrugs        []string                    `json:"checked_drugs"`
	InteractionsFound   []InteractionResult         `json:"interactions_found"`
	Summary             InteractionSummary          `json:"summary"`
	Recommendations     []string                    `json:"recommendations"`
	AlternativeDrugs    map[string][]string         `json:"alternative_drugs,omitempty"`
	MonitoringPlan      []MonitoringRecommendation  `json:"monitoring_plan,omitempty"`
	CheckTimestamp      time.Time                   `json:"check_timestamp"`
	CacheHit            bool                        `json:"cache_hit,omitempty"`
}

// InteractionResult represents a single interaction found
type InteractionResult struct {
	InteractionID       string                 `json:"interaction_id"`
	DrugA               DrugInfo               `json:"drug_a"`
	DrugB               DrugInfo               `json:"drug_b"`
	Severity            string                 `json:"severity"`
	InteractionType     string                 `json:"interaction_type"`
	EvidenceLevel       string                 `json:"evidence_level"`
	Mechanism           string                 `json:"mechanism"`
	ClinicalEffect      string                 `json:"clinical_effect"`
	ManagementStrategy  string                 `json:"management_strategy"`
	DoseAdjustmentRequired bool               `json:"dose_adjustment_required"`
	TimeToOnset         *string                `json:"time_to_onset,omitempty"`
	Duration            *string                `json:"duration,omitempty"`
	MonitoringParameters map[string]interface{} `json:"monitoring_parameters,omitempty"`
	AlternativeDrugs    []string               `json:"alternative_drugs,omitempty"`
	FrequencyScore      *float64               `json:"frequency_score,omitempty"`
	ClinicalSignificance *float64              `json:"clinical_significance,omitempty"`
}

// DrugInfo represents basic drug information
type DrugInfo struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Generic     string `json:"generic,omitempty"`
	Strength    string `json:"strength,omitempty"`
	Route       string `json:"route,omitempty"`
}

// InteractionSummary provides summary statistics
type InteractionSummary struct {
	TotalInteractions       int            `json:"total_interactions"`
	SeverityCounts          map[string]int `json:"severity_counts"`
	HighestSeverity         string         `json:"highest_severity"`
	ContraindicatedPairs    int            `json:"contraindicated_pairs"`
	RequiredActions         []string       `json:"required_actions"`
	RiskScore               float64        `json:"risk_score"`
}

// MonitoringRecommendation represents monitoring suggestions
type MonitoringRecommendation struct {
	Parameter           string    `json:"parameter"`
	Frequency           string    `json:"frequency"`
	Duration            string    `json:"duration"`
	TargetRange         string    `json:"target_range,omitempty"`
	CriticalValues      []string  `json:"critical_values,omitempty"`
	Instructions        string    `json:"instructions"`
}

// Removed duplicate - using BatchInteractionCheckRequest defined earlier in file

// PatientInteractionHistoryRequest for retrieving patient's interaction history
type PatientInteractionHistoryRequest struct {
	PatientID           string    `json:"patient_id" binding:"required"`
	StartDate           time.Time `json:"start_date,omitempty"`
	EndDate             time.Time `json:"end_date,omitempty"`
	SeverityFilter      []string  `json:"severity_filter,omitempty"`
	IncludeResolved     bool      `json:"include_resolved"`
	Limit               int       `json:"limit,omitempty"`
	Offset              int       `json:"offset,omitempty"`
}

// AlertOverrideRequest for overriding interaction alerts
type AlertOverrideRequest struct {
	AlertID             uuid.UUID `json:"alert_id" binding:"required"`
	OverrideReason      string    `json:"override_reason" binding:"required"`
	OverriddenBy        string    `json:"overridden_by" binding:"required"`
	ClinicalJustification string  `json:"clinical_justification,omitempty"`
	AlternativeMonitoring string  `json:"alternative_monitoring,omitempty"`
}

// JSONB custom type for handling PostgreSQL JSONB fields
type JSONB map[string]interface{}

// Scan implements the Scanner interface for JSONB
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = make(map[string]interface{})
		return nil
	}
	
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("cannot scan JSONB: value is not []byte")
	}
	
	var data map[string]interface{}
	if err := json.Unmarshal(bytes, &data); err != nil {
		return err
	}
	
	*j = data
	return nil
}

// Value implements the driver Valuer interface for JSONB
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Table name overrides
func (DrugInteraction) TableName() string {
	return "drug_interactions"
}

func (DrugSynonym) TableName() string {
	return "drug_synonyms"
}

func (InteractionPattern) TableName() string {
	return "interaction_patterns"
}

func (PatientInteractionAlert) TableName() string {
	return "patient_interaction_alerts"
}

func (InteractionRule) TableName() string {
	return "interaction_rules"
}

func (DrugFormulation) TableName() string {
	return "drug_formulations"
}

func (CDSConfiguration) TableName() string {
	return "cds_configurations"
}

func (InteractionAnalytics) TableName() string {
	return "interaction_analytics"
}

// Helper methods

// IsCritical checks if the interaction is critical (contraindicated or major)
func (di *DrugInteraction) IsCritical() bool {
	return di.Severity == "contraindicated" || di.Severity == "major"
}

// GetSeverityPriority returns numeric priority for severity (higher = more severe)
func (di *DrugInteraction) GetSeverityPriority() int {
	priorities := map[string]int{
		"contraindicated": 4,
		"major":          3,
		"moderate":       2,
		"minor":          1,
	}
	if priority, exists := priorities[di.Severity]; exists {
		return priority
	}
	return 0
}

// RequiresMonitoring checks if the interaction requires monitoring
func (di *DrugInteraction) RequiresMonitoring() bool {
	return di.MonitoringParameters != nil && len(*di.MonitoringParameters) > 0
}

// IsActive checks if alert is currently active
func (pia *PatientInteractionAlert) IsActive() bool {
	return pia.AlertStatus == "active"
}

// RequiresAction checks if alert needs attention
func (pia *PatientInteractionAlert) RequiresAction() bool {
	return pia.AlertStatus == "active" && pia.AcknowledgedAt == nil
}

// CalculateRiskScore calculates overall risk score for interaction set
func (icr *InteractionCheckResponse) CalculateRiskScore() float64 {
	if len(icr.InteractionsFound) == 0 {
		return 0.0
	}

	var totalRisk float64
	severityWeights := map[string]float64{
		"contraindicated": 1.0,
		"major":          0.7,
		"moderate":       0.4,
		"minor":          0.1,
	}

	for _, interaction := range icr.InteractionsFound {
		weight := severityWeights[interaction.Severity]
		significance := 0.5 // default
		if interaction.ClinicalSignificance != nil {
			significance = *interaction.ClinicalSignificance
		}
		totalRisk += weight * significance
	}

	// Normalize to 0-1 scale
	maxPossible := float64(len(icr.InteractionsFound))
	if maxPossible == 0 {
		return 0.0
	}
	
	return totalRisk / maxPossible
}

// GetHighestSeverity returns the highest severity level found
func (icr *InteractionCheckResponse) GetHighestSeverity() string {
	if len(icr.InteractionsFound) == 0 {
		return ""
	}

	severityOrder := []string{"contraindicated", "major", "moderate", "minor"}
	
	for _, severity := range severityOrder {
		for _, interaction := range icr.InteractionsFound {
			if interaction.Severity == severity {
				return severity
			}
		}
	}
	
	return "minor"
}

// HasContraindication checks if any interactions are contraindicated
func (icr *InteractionCheckResponse) HasContraindication() bool {
	for _, interaction := range icr.InteractionsFound {
		if interaction.Severity == "contraindicated" {
			return true
		}
	}
	return false
}