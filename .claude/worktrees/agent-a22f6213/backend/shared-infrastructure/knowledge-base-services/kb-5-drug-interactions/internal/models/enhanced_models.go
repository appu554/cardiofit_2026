package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

// JSONBStringArray represents a JSONB array of strings
type JSONBStringArray []string

// Value implements the driver.Valuer interface for JSONBStringArray
func (ja JSONBStringArray) Value() (driver.Value, error) {
	if ja == nil {
		return nil, nil
	}
	return json.Marshal(ja)
}

// Scan implements the sql.Scanner interface for JSONBStringArray
func (ja *JSONBStringArray) Scan(value interface{}) error {
	if value == nil {
		*ja = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, ja)
}

// StringArray represents a PostgreSQL text array type
type StringArray []string

// Value implements the driver.Valuer interface for StringArray
func (sa StringArray) Value() (driver.Value, error) {
	if sa == nil {
		return nil, nil
	}
	return pq.Array(sa).Value()
}

// Scan implements the sql.Scanner interface for StringArray
func (sa *StringArray) Scan(value interface{}) error {
	if value == nil {
		*sa = nil
		return nil
	}
	return pq.Array(sa).Scan(value)
}

// CacheManager interface for cache operations
type CacheManager struct {
	// Implementation would be in cache package
}

// NewCacheManager creates a new cache manager
func NewCacheManager(client interface{}) *CacheManager {
	return &CacheManager{}
}

// ConfigProvider interface for configuration access
type ConfigProvider interface {
	GetDatabaseURL() string
	GetCacheURL() string
	GetLogLevel() string
}

// PatientContext represents comprehensive patient clinical context
type PatientContext struct {
	PatientID         string            `json:"patient_id"`
	Age               int               `json:"age,omitempty"`
	Weight            *decimal.Decimal  `json:"weight,omitempty"`
	RenalFunction     *decimal.Decimal  `json:"renal_function,omitempty"`
	HepaticFunction   string            `json:"hepatic_function,omitempty"`
	PGXMarkers        map[string]string `json:"pgx_markers,omitempty"`
	Allergies         []string          `json:"allergies,omitempty"`
	Comorbidities     []string          `json:"comorbidities,omitempty"`
}

// RequestPriority for interaction check prioritization
type RequestPriority string

const (
	PriorityEmergent RequestPriority = "emergent"
	PriorityUrgent   RequestPriority = "urgent" 
	PriorityRoutine  RequestPriority = "routine"
)

// ClinicalSettings for institutional configuration
type ClinicalSettings struct {
	RequirePharmacistReview bool     `json:"require_pharmacist_review"`
	InstitutionID          string   `json:"institution_id"`
	AllowOverrides         bool     `json:"allow_overrides"`
	AlertThresholds        []string `json:"alert_thresholds"`
}

// GetModifierInteractions method for CacheManager (placeholder)
func (cm *CacheManager) GetModifierInteractions(key string) ([]interface{}, bool) {
	return nil, false
}

// SetModifierInteractions method for CacheManager (placeholder)
func (cm *CacheManager) SetModifierInteractions(key string, value interface{}, ttl time.Duration) {
	// Implementation would be in cache package
}

// Enhanced models aligned with the Final Proposal specifications

// DDISeverity represents the severity levels for drug interactions
type DDISeverity string

const (
	SeverityContraindicated DDISeverity = "contraindicated"
	SeverityMajor          DDISeverity = "major" 
	SeverityModerate       DDISeverity = "moderate"
	SeverityMinor          DDISeverity = "minor"
	SeverityUnknown        DDISeverity = "unknown"
)

// EvidenceLevel represents the quality of evidence for interactions
type EvidenceLevel string

const (
	EvidenceLevelA             EvidenceLevel = "A"              // High quality evidence
	EvidenceLevelB             EvidenceLevel = "B"              // Moderate quality evidence  
	EvidenceLevelC             EvidenceLevel = "C"              // Low quality evidence
	EvidenceLevelD             EvidenceLevel = "D"              // Very low quality evidence
	EvidenceLevelExpertOpinion EvidenceLevel = "ExpertOpinion" // Expert consensus
	EvidenceLevelUnknown       EvidenceLevel = "Unknown"        // Evidence level not determined
)

// MechanismType represents the interaction mechanism
type MechanismType string

const (
	MechanismPK      MechanismType = "PK"      // Pharmacokinetic
	MechanismPD      MechanismType = "PD"      // Pharmacodynamic
	MechanismPKPD    MechanismType = "PK_PD"   // Combined PK/PD
	MechanismUnknown MechanismType = "Unknown" // Mechanism not known
)

// ContextUse represents the clinical context of drug use
type ContextUse string

const (
	ContextAcute       ContextUse = "acute"
	ContextChronic     ContextUse = "chronic"
	ContextUnspecified ContextUse = "unspecified"
)

// Enhanced drug interaction model with dataset versioning and evidence
type EnhancedDrugInteraction struct {
	ID                     uuid.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	DatasetVersion         string       `gorm:"not null;index" json:"dataset_version"`
	Drug1Code             string       `gorm:"size:100;not null;index" json:"drug1_code"`
	Drug2Code             string       `gorm:"size:100;not null;index" json:"drug2_code"`
	Severity              DDISeverity  `gorm:"not null;index" json:"severity"`
	Mechanism             MechanismType `gorm:"not null" json:"mechanism"`
	MechanismNote         *string      `gorm:"type:text" json:"mechanism_note,omitempty"`
	ClinicalEffects       string       `gorm:"type:text;not null" json:"clinical_effects"`
	ManagementStrategy    string       `gorm:"type:text;not null" json:"management_strategy"`
	Evidence              EvidenceLevel `gorm:"not null" json:"evidence"`
	Confidence            *decimal.Decimal `gorm:"type:decimal(3,2);default:0.80" json:"confidence,omitempty"`
	RouteRestriction      StringArray  `gorm:"type:text[]" json:"route_restriction,omitempty"`
	MinDose1MG           *decimal.Decimal `gorm:"type:decimal(10,3)" json:"min_dose1_mg,omitempty"`
	MinDose2MG           *decimal.Decimal `gorm:"type:decimal(10,3)" json:"min_dose2_mg,omitempty"`
	Context              ContextUse   `gorm:"default:unspecified" json:"context"`
	PGXRequired          bool         `gorm:"default:false" json:"pgx_required"`
	PGXMarkers           *JSONB       `gorm:"type:jsonb" json:"pgx_markers,omitempty"`
	FoodAlcoholHerbal    *JSONB       `gorm:"type:jsonb" json:"food_alcohol_herbal,omitempty"`
	ContraindicationReason *string    `gorm:"type:text" json:"contraindication_reason,omitempty"`
	SourceVendorIDs      StringArray  `gorm:"type:text[]" json:"source_vendor_ids,omitempty"`
	Active               bool         `gorm:"default:true;index" json:"active"`
	CreatedAt            time.Time    `json:"created_at"`
	UpdatedAt            time.Time    `json:"updated_at"`
}

// Drug class interaction rules
type DDIClassRule struct {
	ID               uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	DatasetVersion   string         `gorm:"not null;index" json:"dataset_version"`
	ObjectType       string         `gorm:"not null;check:object_type IN ('drug','class')" json:"object_type"`
	ObjectCode       string         `gorm:"not null" json:"object_code"`
	SubjectType      string         `gorm:"not null;check:subject_type IN ('drug','class')" json:"subject_type"`
	SubjectCode      string         `gorm:"not null" json:"subject_code"`
	Severity         DDISeverity    `gorm:"not null" json:"severity"`
	Mechanism        MechanismType  `gorm:"not null" json:"mechanism"`
	ClinicalEffects  *string        `gorm:"type:text" json:"clinical_effects,omitempty"`
	ManagementStrategy string       `gorm:"type:text;not null" json:"management_strategy"`
	Qualifiers       *JSONB         `gorm:"type:jsonb" json:"qualifiers,omitempty"`
	Evidence         EvidenceLevel  `gorm:"not null" json:"evidence"`
	Confidence       *decimal.Decimal `gorm:"type:decimal(3,2);default:0.75" json:"confidence,omitempty"`
	SourceVendorIDs  StringArray    `gorm:"type:text[]" json:"source_vendor_ids,omitempty"`
	Active           bool           `gorm:"default:true" json:"active"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

// Pharmacogenomic interaction rules
type DDIPharmacogenomicRule struct {
	ID                uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	DatasetVersion    string        `gorm:"not null;index" json:"dataset_version"`
	DrugCode          string        `gorm:"not null;index" json:"drug_code"`
	Gene              string        `gorm:"not null" json:"gene"`
	Phenotype         string        `gorm:"not null" json:"phenotype"`
	InteractionWith   *string       `json:"interaction_with,omitempty"`
	Severity          DDISeverity   `gorm:"not null" json:"severity"`
	ClinicalEffects   *string       `gorm:"type:text" json:"clinical_effects,omitempty"`
	ManagementStrategy string       `gorm:"type:text;not null" json:"management_strategy"`
	Evidence          EvidenceLevel `gorm:"not null" json:"evidence"`
	SourceVendorIDs   StringArray   `gorm:"type:text[]" json:"source_vendor_ids,omitempty"`
	Active            bool          `gorm:"default:true" json:"active"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

// Drug interaction modifiers (food, alcohol, herbal)
type DDIModifier struct {
	ID                uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	DatasetVersion    string        `gorm:"not null;index" json:"dataset_version"`
	ModifierType      string        `gorm:"not null;check:modifier_type IN ('food','alcohol','herbal','disease')" json:"modifier_type"`
	ModifierCode      *string       `json:"modifier_code,omitempty"`
	DrugCode          string        `gorm:"not null" json:"drug_code"`
	Effect            string        `gorm:"not null" json:"effect"`
	ManagementStrategy string       `gorm:"type:text;not null" json:"management_strategy"`
	Severity          DDISeverity   `gorm:"not null" json:"severity"`
	Evidence          EvidenceLevel `gorm:"not null" json:"evidence"`
	SourceVendorIDs   StringArray   `gorm:"type:text[]" json:"source_vendor_ids,omitempty"`
	Active            bool          `gorm:"default:true" json:"active"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

// Institutional overrides for P&T Committee modifications
type DDIOverride struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	DatasetVersion string    `gorm:"not null;index" json:"dataset_version"`
	Scope          string    `gorm:"not null;check:scope IN ('pair','class','pgx','modifier')" json:"scope"`
	Selector       JSONB     `gorm:"type:jsonb;not null" json:"selector"`
	Action         string    `gorm:"not null;check:action IN ('replace','suppress','amend')" json:"action"`
	Payload        JSONB     `gorm:"type:jsonb;not null" json:"payload"`
	Rationale      string    `gorm:"type:text;not null" json:"rationale"`
	Approver       string    `gorm:"not null" json:"approver"`
	ApprovedAt     time.Time `gorm:"not null;default:now()" json:"approved_at"`
	Active         bool      `gorm:"default:true" json:"active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Dataset version tracking for Evidence Envelope integration
type DDIDatasetVersion struct {
	ID           uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	VersionName  string           `gorm:"unique;not null" json:"version_name"`
	VendorSource string           `gorm:"not null" json:"vendor_source"`
	HarmonizedAt time.Time        `gorm:"not null;default:now()" json:"harmonized_at"`
	RecordCount  int64            `gorm:"default:0" json:"record_count"`
	QualityScore *decimal.Decimal `gorm:"type:decimal(3,2);default:0.0" json:"quality_score,omitempty"`
	IsCurrent    bool             `gorm:"default:false" json:"is_current"`
	Metadata     *JSONB           `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`
	CreatedBy    *string          `json:"created_by,omitempty"`
}

// Enhanced interaction matrix for high-performance lookups
type DDIInteractionMatrix struct {
	DatasetVersion   string           `json:"dataset_version"`
	Drug1Code        string           `json:"drug1_code"`
	Drug2Code        string           `json:"drug2_code"`
	Severity         DDISeverity      `json:"severity"`
	ManagementStrategy string         `json:"management_strategy"`
	Evidence         EvidenceLevel    `json:"evidence"`
	Confidence       *decimal.Decimal `json:"confidence,omitempty"`
	Mechanism        MechanismType    `json:"mechanism"`
	ClinicalEffects  string           `json:"clinical_effects"`
	PGXRequired      bool             `json:"pgx_required"`
	PGXMarkers       *JSONB           `json:"pgx_markers,omitempty"`
	RouteRestriction StringArray      `json:"route_restriction,omitempty"`
	Context          ContextUse       `json:"context"`
}

// Enhanced request/response models with gRPC alignment

// Enhanced interaction check request
type EnhancedInteractionCheckRequest struct {
	TransactionID      string                 `json:"transaction_id,omitempty"`
	DrugCodes          []string               `json:"drug_codes" binding:"required,min=2"`
	PatientContext     *PatientContextData    `json:"patient_context,omitempty"`
	DatasetVersion     string                 `json:"dataset_version,omitempty"`
	ExpandClasses      bool                   `json:"expand_classes,omitempty"`
	IncludeContextuals bool                   `json:"include_contextuals,omitempty"`
	IncludeAlternatives bool                  `json:"include_alternatives,omitempty"`
	IncludeMonitoring  bool                   `json:"include_monitoring,omitempty"`
	SeverityFilter     []string               `json:"severity_filter,omitempty"`
	ClinicalContext    map[string]interface{} `json:"clinical_context,omitempty"`
}

// Patient context for personalized checking
type PatientContextData struct {
	PGX           map[string]string `json:"pgx,omitempty"`           // {"CYP2D6": "ultrarapid"}
	HepaticStage  string            `json:"hepatic_stage,omitempty"` // "ChildPugh_A", "ChildPugh_B", "ChildPugh_C"
	RenalStage    string            `json:"renal_stage,omitempty"`   // "CKD_1", "CKD_2", etc.
	AgeBand       string            `json:"age_band,omitempty"`      // "pediatric", "adult", "older_adult"
	Comorbidities []string          `json:"comorbidities,omitempty"` // SNOMED codes
	Allergies     map[string]string `json:"allergies,omitempty"`     // drug allergies
}

// Enhanced interaction result with full clinical data
type EnhancedInteractionResult struct {
	InteractionID         string                 `json:"interaction_id"`
	Drug1                 DrugInfo               `json:"drug1"`
	Drug2                 DrugInfo               `json:"drug2"`
	Severity              DDISeverity            `json:"severity"`
	Mechanism             MechanismType          `json:"mechanism"`
	ClinicalEffects       string                 `json:"clinical_effects"`
	ManagementStrategy    string                 `json:"management_strategy"`
	Evidence              EvidenceLevel          `json:"evidence"`
	Confidence            *decimal.Decimal       `json:"confidence,omitempty"`
	Qualifiers            map[string]string      `json:"qualifiers,omitempty"`
	Sources               []string               `json:"sources,omitempty"`
	PGXApplicable         bool                   `json:"pgx_applicable"`
	DoseAdjustmentRequired bool                  `json:"dose_adjustment_required"`
	TimeToOnset           *string                `json:"time_to_onset,omitempty"`
	Duration              *string                `json:"duration,omitempty"`
	MonitoringParameters  map[string]interface{} `json:"monitoring_parameters,omitempty"`
	AlternativeDrugs      []string               `json:"alternative_drugs,omitempty"`
	RouteSpecific         bool                   `json:"route_specific"`
}

// Conflict trail for audit and provenance
type ConflictTrail struct {
	SynthesizedFromVersion string    `json:"synthesized_from_version"`
	VendorEvidenceIDs      []string  `json:"vendor_evidence_ids"`
	OverridesApplied       []string  `json:"overrides_applied"`
	HarmonizedAt           time.Time `json:"harmonized_at"`
	HarmonizerVersion      string    `json:"harmonizer_version"`
}

// Enhanced interaction summary with clinical metrics
type EnhancedInteractionSummary struct {
	TotalInteractions    int               `json:"total_interactions"`
	SeverityCounts       map[string]int    `json:"severity_counts"`
	HighestSeverity      string            `json:"highest_severity"`
	ContraindicatedPairs int               `json:"contraindicated_pairs"`
	RequiredActions      []string          `json:"required_actions"`
	PGXInteractions      int               `json:"pgx_interactions"`
	ClassInteractions    int               `json:"class_interactions"`
	ModifierInteractions int               `json:"modifier_interactions"`
	RiskScore            decimal.Decimal   `json:"risk_score"`
}

// Enhanced response with full clinical decision support
type EnhancedInteractionCheckResponse struct {
	TransactionID      string                           `json:"transaction_id,omitempty"`
	DatasetVersion     string                           `json:"dataset_version"`
	Interactions       []EnhancedInteractionResult      `json:"interactions"`
	ConflictTrail      *ConflictTrail                   `json:"conflict_trail,omitempty"`
	Summary            EnhancedInteractionSummary       `json:"summary"`
	Recommendations    []string                         `json:"recommendations"`
	AlternativeDrugs   map[string]DrugAlternatives      `json:"alternative_drugs,omitempty"`
	MonitoringPlan     []MonitoringRecommendation       `json:"monitoring_plan,omitempty"`
	CheckTimestamp     time.Time                        `json:"check_timestamp"`
	CacheHit           bool                             `json:"cache_hit,omitempty"`
	RiskScore          decimal.Decimal                  `json:"risk_score"`
}

// Alternative drug suggestions
type DrugAlternatives struct {
	Alternatives []AlternativeDrug `json:"alternatives"`
	Rationale    string            `json:"rationale"`
}

type AlternativeDrug struct {
	DrugInfo        DrugInfo         `json:"drug_info"`
	Reason          string           `json:"reason"`
	SafetyScore     *decimal.Decimal `json:"safety_score,omitempty"`
	RequiresMonitoring bool          `json:"requires_monitoring"`
}

// Enhanced monitoring recommendations
type EnhancedMonitoringRecommendation struct {
	Parameter      string   `json:"parameter"`
	Frequency      string   `json:"frequency"`
	Duration       string   `json:"duration"`
	TargetRange    string   `json:"target_range,omitempty"`
	CriticalValues []string `json:"critical_values,omitempty"`
	Instructions   string   `json:"instructions"`
	Rationale      string   `json:"rationale"`
	Priority       int      `json:"priority"` // 1-5, 1 being highest priority
}

// Batch processing models with enhanced features
type EnhancedBatchInteractionCheckRequest struct {
	Requests []EnhancedInteractionCheckRequest `json:"requests" binding:"required,min=1,max=100"`
	Options  EnhancedBatchCheckOptions         `json:"options,omitempty"`
}

type EnhancedBatchCheckOptions struct {
	Parallel         bool     `json:"parallel"`
	MaxConcurrency   int      `json:"max_concurrency"`
	SeverityFilter   []string `json:"severity_filter,omitempty"`
	IncludeStatistics bool    `json:"include_statistics"`
	FailFast         bool     `json:"fail_fast"`
}

type EnhancedBatchInteractionResult struct {
	RequestID         string                           `json:"request_id"`
	Interactions      []EnhancedInteractionResult      `json:"interactions"`
	Summary           EnhancedInteractionSummary       `json:"summary"`
	ProcessedAt       time.Time                        `json:"processed_at"`
	Error             string                           `json:"error,omitempty"`
	ProcessingTimeMS  float64                          `json:"processing_time_ms"`
}

// Matrix statistics for monitoring and observability
type EnhancedMatrixStatistics struct {
	TotalDrugs         int                    `json:"total_drugs"`
	TotalInteractions  int                    `json:"total_interactions"`
	MatrixDensity      float64                `json:"matrix_density"`
	LastUpdated        time.Time              `json:"last_updated"`
	MemoryUsageMB      float64                `json:"memory_usage_mb"`
	LookupPerformance  PerformanceMetrics     `json:"lookup_performance"`
	CurrentDatasetVersion string              `json:"current_dataset_version"`
	CacheStatistics    CacheStatistics        `json:"cache_statistics"`
}

// Cache statistics for hot/warm strategy monitoring
type CacheStatistics struct {
	HotCacheHitRate    float64   `json:"hot_cache_hit_rate"`
	WarmCacheHitRate   float64   `json:"warm_cache_hit_rate"`
	HotCacheEntries    int64     `json:"hot_cache_entries"`
	WarmCacheEntries   int64     `json:"warm_cache_entries"`
	HotCacheMemoryMB   float64   `json:"hot_cache_memory_mb"`
	EvictionRate       float64   `json:"eviction_rate"`
	LastRefresh        time.Time `json:"last_refresh"`
}

// Enhanced error handling
type ErrorDetail struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Details   map[string]string `json:"details,omitempty"`
	Component string            `json:"component"`
	Retryable bool              `json:"retryable"`
}

// Clinical validation models for testing
type ClinicalValidationFixture struct {
	TestName        string                           `json:"test_name"`
	Description     string                           `json:"description"`
	Request         EnhancedInteractionCheckRequest `json:"request"`
	ExpectedResults struct {
		InteractionCount    int      `json:"interaction_count"`
		HighestSeverity     string   `json:"highest_severity"`
		ContraindicatedPairs int     `json:"contraindicated_pairs"`
		RequiredManagement  []string `json:"required_management"`
		MinRiskScore        float64  `json:"min_risk_score"`
		MaxRiskScore        float64  `json:"max_risk_score"`
	} `json:"expected_results"`
	ClinicalRationale string `json:"clinical_rationale"`
}

// Table name overrides for enhanced models
func (EnhancedDrugInteraction) TableName() string {
	return "drug_interactions"
}

func (DDIClassRule) TableName() string {
	return "ddi_class_rules"
}

func (DDIPharmacogenomicRule) TableName() string {
	return "ddi_pharmacogenomic_rules"
}

func (DDIModifier) TableName() string {
	return "ddi_modifiers"
}

func (DDIOverride) TableName() string {
	return "ddi_overrides"
}

func (DDIDatasetVersion) TableName() string {
	return "ddi_dataset_versions"
}

// Helper methods for enhanced models

// GetSeverityPriority returns numeric priority (higher = more severe)
func (s DDISeverity) GetPriority() int {
	priorities := map[DDISeverity]int{
		SeverityContraindicated: 4,
		SeverityMajor:          3,
		SeverityModerate:       2,
		SeverityMinor:          1,
		SeverityUnknown:        0,
	}
	return priorities[s]
}

// IsHighRisk checks if interaction is high risk (contraindicated or major)
func (eir *EnhancedInteractionResult) IsHighRisk() bool {
	return eir.Severity == SeverityContraindicated || eir.Severity == SeverityMajor
}

// RequiresImmediateAction checks if interaction needs immediate clinical attention
func (eir *EnhancedInteractionResult) RequiresImmediateAction() bool {
	return eir.Severity == SeverityContraindicated
}

// GetClinicalPriority calculates clinical priority based on severity and confidence
func (eir *EnhancedInteractionResult) GetClinicalPriority() float64 {
	severityWeight := float64(eir.Severity.GetPriority())
	confidence := 0.8 // default
	if eir.Confidence != nil {
		confidence, _ = eir.Confidence.Float64()
	}
	return severityWeight * confidence
}

// CalculateRiskScore calculates overall risk score for interaction set
func (eicr *EnhancedInteractionCheckResponse) CalculateRiskScore() decimal.Decimal {
	if len(eicr.Interactions) == 0 {
		return decimal.Zero
	}

	var totalRisk decimal.Decimal
	severityWeights := map[DDISeverity]decimal.Decimal{
		SeverityContraindicated: decimal.NewFromFloat(1.0),
		SeverityMajor:          decimal.NewFromFloat(0.7),
		SeverityModerate:       decimal.NewFromFloat(0.4),
		SeverityMinor:          decimal.NewFromFloat(0.1),
	}

	for _, interaction := range eicr.Interactions {
		weight := severityWeights[interaction.Severity]
		confidence := decimal.NewFromFloat(0.5) // default
		if interaction.Confidence != nil {
			confidence = *interaction.Confidence
		}
		totalRisk = totalRisk.Add(weight.Mul(confidence))
	}

	// Normalize to 0-1 scale
	maxPossible := decimal.NewFromInt(int64(len(eicr.Interactions)))
	if maxPossible.IsZero() {
		return decimal.Zero
	}

	return totalRisk.Div(maxPossible)
}

// HasContraindications checks if any interactions are contraindicated
func (eicr *EnhancedInteractionCheckResponse) HasContraindications() bool {
	for _, interaction := range eicr.Interactions {
		if interaction.Severity == SeverityContraindicated {
			return true
		}
	}
	return false
}

// GetHighestSeverityInteractions returns interactions at the highest severity level
func (eicr *EnhancedInteractionCheckResponse) GetHighestSeverityInteractions() []EnhancedInteractionResult {
	if len(eicr.Interactions) == 0 {
		return []EnhancedInteractionResult{}
	}

	highestSeverity := eicr.Summary.HighestSeverity
	var results []EnhancedInteractionResult
	
	for _, interaction := range eicr.Interactions {
		if string(interaction.Severity) == highestSeverity {
			results = append(results, interaction)
		}
	}
	
	return results
}