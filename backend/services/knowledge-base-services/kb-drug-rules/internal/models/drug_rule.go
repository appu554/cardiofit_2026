package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// StringArray is a custom type to handle PostgreSQL string arrays
type StringArray []string

// Scan implements the sql.Scanner interface for reading from database
func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = StringArray{}
		return nil
	}

	switch v := value.(type) {
	case string:
		// Handle PostgreSQL array format: {item1,item2,item3}
		if v == "{}" {
			*s = StringArray{}
			return nil
		}
		// Remove braces and split by comma
		v = strings.Trim(v, "{}")
		if v == "" {
			*s = StringArray{}
			return nil
		}
		*s = StringArray(strings.Split(v, ","))
		return nil
	case []byte:
		return s.Scan(string(v))
	default:
		return fmt.Errorf("cannot scan %T into StringArray", value)
	}
}

// Value implements the driver.Valuer interface for writing to database
func (s StringArray) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "{}", nil
	}
	return "{" + strings.Join([]string(s), ",") + "}", nil
}

// VersionHistoryEntry represents a single version history entry
type VersionHistoryEntry struct {
	Version         string    `json:"version"`
	ModifiedDate    time.Time `json:"modified_date"`
	ModifiedBy      string    `json:"modified_by"`
	ChangeSummary   string    `json:"change_summary"`
	SnapshotID      string    `json:"snapshot_id,omitempty"`
}

// VersionHistoryArray is a custom type to handle version history as JSONB
type VersionHistoryArray []VersionHistoryEntry

// Scan implements the sql.Scanner interface for reading JSONB from database
func (v *VersionHistoryArray) Scan(value interface{}) error {
	if value == nil {
		*v = VersionHistoryArray{}
		return nil
	}

	var bytes []byte
	switch val := value.(type) {
	case []byte:
		bytes = val
	case string:
		bytes = []byte(val)
	default:
		return fmt.Errorf("cannot scan %T into VersionHistoryArray", value)
	}

	return json.Unmarshal(bytes, v)
}

// Value implements the driver.Valuer interface for writing JSONB to database
func (v VersionHistoryArray) Value() (driver.Value, error) {
	return json.Marshal(v)
}

// DeploymentStatus represents deployment status information
type DeploymentStatus struct {
	Staging      string    `json:"staging"`      // "deployed", "pending", "failed"
	Production   string    `json:"production"`   // "deployed", "pending", "failed"
	LastDeployed time.Time `json:"last_deployed"`
}

// DeploymentStatusJSON is a custom type to handle deployment status as JSONB
type DeploymentStatusJSON DeploymentStatus

// Scan implements the sql.Scanner interface for reading JSONB from database
func (d *DeploymentStatusJSON) Scan(value interface{}) error {
	if value == nil {
		*d = DeploymentStatusJSON{}
		return nil
	}

	var bytes []byte
	switch val := value.(type) {
	case []byte:
		bytes = val
	case string:
		bytes = []byte(val)
	default:
		return fmt.Errorf("cannot scan %T into DeploymentStatusJSON", value)
	}

	return json.Unmarshal(bytes, d)
}

// Value implements the driver.Valuer interface for writing JSONB to database
func (d DeploymentStatusJSON) Value() (driver.Value, error) {
	return json.Marshal(d)
}

// DrugRulePack represents a complete versioned drug rule package with enhanced TOML support
type DrugRulePack struct {
	ID                   string                 `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	DrugID               string                 `json:"drug_id" gorm:"not null;index"`
	Version              string                 `json:"version" gorm:"not null;index"`
	ContentSHA           string                 `json:"content_sha" gorm:"not null"`
	CreatedAt            time.Time              `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt            time.Time              `json:"updated_at" gorm:"autoUpdateTime"`

	// Enhanced TOML Support
	OriginalFormat       string                 `json:"original_format" gorm:"default:'json';check:original_format IN ('toml','json')"`
	TOMLContent          *string                `json:"toml_content,omitempty" gorm:"type:text"`
	JSONContent          json.RawMessage        `json:"json_content" gorm:"type:jsonb;not null"`

	// Versioning Support
	PreviousVersion      *string                `json:"previous_version,omitempty"`
	VersionHistory       VersionHistoryArray    `json:"version_history" gorm:"type:jsonb;default:'[]'"`

	// Clinical Governance
	SignedBy             string                 `json:"signed_by" gorm:"not null"`
	SignatureValid       bool                   `json:"signature_valid" gorm:"default:false"`
	ClinicalReviewer     string                 `json:"clinical_reviewer"`
	ClinicalReviewDate   *time.Time             `json:"clinical_review_date"`

	// Deployment Tracking
	DeploymentStatus     DeploymentStatusJSON   `json:"deployment_status" gorm:"type:jsonb;default:'{}'"`
	Regions              StringArray            `json:"regions" gorm:"type:text[]"`

	// Audit Fields
	CreatedBy            string                 `json:"created_by" gorm:"default:'system'"`
	LastModifiedBy       string                 `json:"last_modified_by" gorm:"default:'system'"`
	Tags                 StringArray            `json:"tags" gorm:"type:text[];default:'{}'"`

	// Legacy Support (for backward compatibility)
	Content              DrugRuleContent        `json:"content" gorm:"type:jsonb"`
	Signature            string                 `json:"signature"`
}

// TableName specifies the table name for GORM
func (DrugRulePack) TableName() string {
	return "drug_rule_packs"
}

// DrugRuleContent contains the actual clinical rules and calculations
type DrugRuleContent struct {
	Meta                    RuleMetadata                   `json:"meta"`
	DoseCalculation         DoseCalculation                `json:"dose_calculation"`
	SafetyVerification      SafetyVerification             `json:"safety_verification"`
	MonitoringRequirements  []MonitoringRequirement        `json:"monitoring_requirements"`
	RegionalVariations      map[string]RegionalOverride    `json:"regional_variations"`
}

// Scan implements the sql.Scanner interface for reading JSONB from database
func (d *DrugRuleContent) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into DrugRuleContent", value)
	}

	return json.Unmarshal(bytes, d)
}

// Value implements the driver.Valuer interface for writing JSONB to database
func (d DrugRuleContent) Value() (driver.Value, error) {
	return json.Marshal(d)
}

// RuleMetadata contains metadata about the drug rules
type RuleMetadata struct {
	DrugName             string              `json:"drug_name"`
	TherapeuticClass     []string            `json:"therapeutic_class"`
	EvidenceSources      []string            `json:"evidence_sources"`
	GuidelineReferences  []GuidelineRef      `json:"guideline_references"`
	LastMajorUpdate      time.Time           `json:"last_major_update"`
	UpdateRationale      string              `json:"update_rationale"`
}

// GuidelineRef represents a clinical guideline reference
type GuidelineRef struct {
	Organization string `json:"organization"`
	Title        string `json:"title"`
	Year         int    `json:"year"`
	URL          string `json:"url"`
	Level        string `json:"level"` // A, B, C evidence levels
}

// DoseCalculation contains dose calculation rules and formulas
type DoseCalculation struct {
	BaseFormula          string                    `json:"base_formula"`
	AdjustmentFactors    []AdjustmentFactor        `json:"adjustment_factors"`
	MaxDailyDose         float64                   `json:"max_daily_dose"`
	MinDailyDose         float64                   `json:"min_daily_dose"`
	RenalAdjustment      *RenalAdjustment          `json:"renal_adjustment,omitempty"`
	HepaticAdjustment    *HepaticAdjustment        `json:"hepatic_adjustment,omitempty"`
	AgeAdjustments       []AgeAdjustment           `json:"age_adjustments"`
	WeightAdjustments    []WeightAdjustment        `json:"weight_adjustments"`
	SpecialPopulations   []SpecialPopulation       `json:"special_populations"`
}

// AdjustmentFactor represents a dose adjustment factor
type AdjustmentFactor struct {
	Factor      string  `json:"factor"`
	Condition   string  `json:"condition"`
	Multiplier  float64 `json:"multiplier"`
	AdditiveMg  float64 `json:"additive_mg"`
	MaxDoseMg   float64 `json:"max_dose_mg"`
	MinDoseMg   float64 `json:"min_dose_mg"`
}

// RenalAdjustment contains renal function-based dose adjustments
type RenalAdjustment struct {
	EGFRThresholds []EGFRThreshold `json:"egfr_thresholds"`
	DialysisRules  *DialysisRules  `json:"dialysis_rules,omitempty"`
}

// EGFRThreshold represents eGFR-based dose adjustment
type EGFRThreshold struct {
	MinEGFR        float64 `json:"min_egfr"`
	MaxEGFR        float64 `json:"max_egfr"`
	DoseMultiplier float64 `json:"dose_multiplier"`
	FrequencyAdj   string  `json:"frequency_adjustment"`
	Contraindicated bool   `json:"contraindicated"`
}

// DialysisRules contains dialysis-specific dosing rules
type DialysisRules struct {
	Hemodialysis          *DialysisRule `json:"hemodialysis,omitempty"`
	PeritonealDialysis    *DialysisRule `json:"peritoneal_dialysis,omitempty"`
	ContinuousRRT         *DialysisRule `json:"continuous_rrt,omitempty"`
}

// DialysisRule represents dosing for specific dialysis type
type DialysisRule struct {
	DoseMultiplier    float64 `json:"dose_multiplier"`
	SupplementalDose  float64 `json:"supplemental_dose"`
	TimingRelativeToDialysis string `json:"timing_relative_to_dialysis"`
}

// HepaticAdjustment contains liver function-based dose adjustments
type HepaticAdjustment struct {
	ChildPughAdjustments []ChildPughAdjustment `json:"child_pugh_adjustments"`
}

// ChildPughAdjustment represents Child-Pugh class-based adjustment
type ChildPughAdjustment struct {
	ChildPughClass  string  `json:"child_pugh_class"` // A, B, C
	DoseMultiplier  float64 `json:"dose_multiplier"`
	Contraindicated bool    `json:"contraindicated"`
	MonitoringReq   string  `json:"monitoring_requirement"`
}

// AgeAdjustment contains age-based dose adjustments
type AgeAdjustment struct {
	MinAge         int     `json:"min_age"`
	MaxAge         int     `json:"max_age"`
	DoseMultiplier float64 `json:"dose_multiplier"`
	MaxDoseMg      float64 `json:"max_dose_mg"`
	SpecialNotes   string  `json:"special_notes"`
}

// WeightAdjustment contains weight-based dose adjustments
type WeightAdjustment struct {
	MinWeightKg    float64 `json:"min_weight_kg"`
	MaxWeightKg    float64 `json:"max_weight_kg"`
	DosePerKg      float64 `json:"dose_per_kg"`
	MaxTotalDose   float64 `json:"max_total_dose"`
	UseIdealWeight bool    `json:"use_ideal_weight"`
}

// SpecialPopulation contains special population considerations
type SpecialPopulation struct {
	Population     string  `json:"population"` // pregnancy, breastfeeding, pediatric, geriatric
	Recommendation string  `json:"recommendation"`
	DoseMultiplier float64 `json:"dose_multiplier"`
	Contraindicated bool   `json:"contraindicated"`
	AlternativeDrugs []string `json:"alternative_drugs"`
}

// SafetyVerification contains safety verification rules
type SafetyVerification struct {
	Contraindications    []Contraindication    `json:"contraindications"`
	Warnings             []Warning             `json:"warnings"`
	Precautions          []Precaution          `json:"precautions"`
	InteractionChecks    []InteractionCheck    `json:"interaction_checks"`
	LabMonitoring        []LabMonitoring       `json:"lab_monitoring"`
}

// Contraindication represents an absolute contraindication
type Contraindication struct {
	Condition    string `json:"condition"`
	ICD10Code    string `json:"icd10_code"`
	SNOMEDCode   string `json:"snomed_code"`
	Severity     string `json:"severity"` // absolute, relative
	Rationale    string `json:"rationale"`
}

// Warning represents a clinical warning
type Warning struct {
	WarningType  string `json:"warning_type"`
	Description  string `json:"description"`
	Severity     string `json:"severity"` // black_box, serious, moderate
	Population   string `json:"population"`
	Mitigation   string `json:"mitigation"`
}

// Precaution represents a clinical precaution
type Precaution struct {
	Condition    string `json:"condition"`
	Description  string `json:"description"`
	Monitoring   string `json:"monitoring"`
	Action       string `json:"action"`
}

// InteractionCheck represents interaction checking rules
type InteractionCheck struct {
	DrugClass       string   `json:"drug_class"`
	SpecificDrugs   []string `json:"specific_drugs"`
	Severity        string   `json:"severity"`
	Mechanism       string   `json:"mechanism"`
	Management      string   `json:"management"`
}

// LabMonitoring represents laboratory monitoring requirements
type LabMonitoring struct {
	Parameter       string        `json:"parameter"`
	LOINCCode       string        `json:"loinc_code"`
	Frequency       string        `json:"frequency"`
	BaselineReq     bool          `json:"baseline_required"`
	CriticalValues  CriticalValues `json:"critical_values"`
	ActionRequired  string        `json:"action_required"`
}

// CriticalValues represents critical lab value thresholds
type CriticalValues struct {
	Low         *float64 `json:"low,omitempty"`
	High        *float64 `json:"high,omitempty"`
	Unit        string   `json:"unit"`
	Action      string   `json:"action"`
}

// MonitoringRequirement represents monitoring requirements
type MonitoringRequirement struct {
	Type            string        `json:"type"` // lab, vital, symptom, efficacy
	Parameter       string        `json:"parameter"`
	Frequency       string        `json:"frequency"`
	Duration        string        `json:"duration"`
	CriticalValues  CriticalValues `json:"critical_values"`
	Instructions    string        `json:"instructions"`
}

// RegionalOverride contains region-specific rule overrides
type RegionalOverride struct {
	Region              string                 `json:"region"`
	DoseCalculation     *DoseCalculation       `json:"dose_calculation,omitempty"`
	SafetyVerification  *SafetyVerification    `json:"safety_verification,omitempty"`
	MonitoringReq       []MonitoringRequirement `json:"monitoring_requirements,omitempty"`
	RegulatoryNotes     string                 `json:"regulatory_notes"`
}

// GetRulesParams represents query parameters for getting rules
type GetRulesParams struct {
	Version         *string `form:"version"`
	Region          *string `form:"region"`
	StrictSignature *bool   `form:"strict_signature"`
}

// DrugRulesResponse represents the API response for drug rules
type DrugRulesResponse struct {
	DrugID           string          `json:"drug_id"`
	Version          string          `json:"version"`
	ContentSHA       string          `json:"content_sha"`
	SignatureValid   bool            `json:"signature_valid"`
	SelectedRegion   *string         `json:"selected_region,omitempty"`
	Content          DrugRuleContent `json:"content"`
	CacheControl     string          `json:"cache_control"`
	ETag             string          `json:"etag"`
}

// ValidationRequest represents a request to validate rules
type ValidationRequest struct {
	Content string   `json:"content"`
	Regions []string `json:"regions"`
}

// ValidationResponse represents the validation result
type ValidationResponse struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
	Info     []string `json:"info,omitempty"`
}

// HotloadRequest represents a request to hotload new rules
type HotloadRequest struct {
	DrugID    string   `json:"drug_id"`
	Version   string   `json:"version"`
	Content   string   `json:"content"`
	Signature string   `json:"signature"`
	SignedBy  string   `json:"signed_by"`
	Regions   []string `json:"regions"`
}

// HotloadResponse represents the hotload result
type HotloadResponse struct {
	Success        bool     `json:"success"`
	DrugID         string   `json:"drug_id"`
	Version        string   `json:"version"`
	ContentSHA     string   `json:"content_sha"`
	RegionsUpdated []string `json:"regions_updated"`
	Message        string   `json:"message,omitempty"`
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Version   string            `json:"version"`
	Checks    map[string]string `json:"checks"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error     string            `json:"error"`
	Message   string            `json:"message"`
	RequestID string            `json:"request_id,omitempty"`
	Details   map[string]string `json:"details,omitempty"`
}

// ===== ENHANCED TOML SUPPORT MODELS =====

// ValidationResult represents comprehensive validation results
type ValidationResult struct {
	IsValid  bool     `json:"is_valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
	Score    float64  `json:"score"` // Quality score 0-100
}

// TOMLValidationRequest represents a request to validate TOML content
type TOMLValidationRequest struct {
	Content string `json:"content" binding:"required"`
	Format  string `json:"format" binding:"required,oneof=toml json"`
}

// TOMLValidationResponse represents the enhanced validation result
type TOMLValidationResponse struct {
	ValidationResult
	ConvertedJSON string    `json:"converted_json,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
}

// FormatConversionRequest represents a request to convert between formats
type FormatConversionRequest struct {
	Content      string `json:"content" binding:"required"`
	SourceFormat string `json:"source_format" binding:"required,oneof=toml json"`
	TargetFormat string `json:"target_format" binding:"required,oneof=toml json"`
}

// FormatConversionResponse represents the conversion result
type FormatConversionResponse struct {
	OriginalFormat   string    `json:"original_format"`
	TargetFormat     string    `json:"target_format"`
	ConvertedContent string    `json:"converted_content"`
	Timestamp        time.Time `json:"timestamp"`
}

// TOMLHotloadRequest represents a request to hotload TOML rules
type TOMLHotloadRequest struct {
	DrugID           string   `json:"drug_id" binding:"required"`
	Version          string   `json:"version" binding:"required"`
	TOMLContent      string   `json:"toml_content" binding:"required"`
	SignedBy         string   `json:"signed_by" binding:"required"`
	ClinicalReviewer string   `json:"clinical_reviewer" binding:"required"`
	Regions          []string `json:"regions"`
	Tags             []string `json:"tags"`
	Signature        string   `json:"signature"`
}

// BatchLoadRequest represents a request to load multiple rules
type BatchLoadRequest struct {
	Rules []TOMLHotloadRequest `json:"rules" binding:"required,min=1"`
	User  string               `json:"user" binding:"required"`
}

// BatchLoadResponse represents the batch load result
type BatchLoadResponse struct {
	TotalRules     int      `json:"total_rules"`
	SuccessfulRules int     `json:"successful_rules"`
	FailedRules    int      `json:"failed_rules"`
	SuccessfulIDs  []string `json:"successful_ids"`
	FailedIDs      []string `json:"failed_ids"`
	Errors         []string `json:"errors,omitempty"`
}

// VersionHistoryRequest represents a request for version history
type VersionHistoryRequest struct {
	DrugID string `json:"drug_id" binding:"required"`
	Limit  int    `json:"limit,omitempty"`
}

// VersionHistoryResponse represents version history response
type VersionHistoryResponse struct {
	DrugID   string                `json:"drug_id"`
	Versions []VersionHistoryEntry `json:"versions"`
	Total    int                   `json:"total"`
}

// RollbackRequest represents a request to rollback to a previous version
type RollbackRequest struct {
	DrugID        string `json:"drug_id" binding:"required"`
	TargetVersion string `json:"target_version" binding:"required"`
	Reason        string `json:"reason" binding:"required"`
	User          string `json:"user" binding:"required"`
}

// RollbackResponse represents the rollback result
type RollbackResponse struct {
	Success       bool      `json:"success"`
	DrugID        string    `json:"drug_id"`
	FromVersion   string    `json:"from_version"`
	ToVersion     string    `json:"to_version"`
	RollbackTime  time.Time `json:"rollback_time"`
	Message       string    `json:"message,omitempty"`
}

// DrugRuleSnapshot represents a snapshot of a drug rule version
type DrugRuleSnapshot struct {
	ID               string          `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	DrugID           string          `json:"drug_id" gorm:"not null;index"`
	Version          string          `json:"version" gorm:"not null"`
	SnapshotDate     time.Time       `json:"snapshot_date" gorm:"autoCreateTime"`
	ContentSnapshot  json.RawMessage `json:"content_snapshot" gorm:"type:jsonb;not null"`
	TOMLSnapshot     *string         `json:"toml_snapshot,omitempty" gorm:"type:text"`
	CreatedBy        string          `json:"created_by" gorm:"not null"`
	Reason           string          `json:"reason" gorm:"type:varchar(500)"`
}

// TableName specifies the table name for GORM
func (DrugRuleSnapshot) TableName() string {
	return "drug_rule_snapshots"
}

// ===== ENHANCED CLINICAL GOVERNANCE MODELS =====

// ClinicalApprovalWorkflow represents the clinical approval process
type ClinicalApprovalWorkflow struct {
	ID               string          `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	DrugRuleID       string          `json:"drug_rule_id" gorm:"not null;index"`
	Version          string          `json:"version" gorm:"not null"`
	Status           string          `json:"status" gorm:"not null;default:'draft'"` // draft, pending_review, approved, rejected
	SubmittedBy      string          `json:"submitted_by" gorm:"not null"`
	SubmittedAt      time.Time       `json:"submitted_at" gorm:"autoCreateTime"`

	// Multi-stage approval
	ApprovalStages   ApprovalStageArray `json:"approval_stages" gorm:"type:jsonb;default:'[]'"`

	// Risk assessment
	RiskLevel        string          `json:"risk_level" gorm:"default:'medium'"` // low, medium, high, critical
	ImpactAnalysis   string          `json:"impact_analysis" gorm:"type:text"`

	// Testing requirements
	TestingRequired  bool            `json:"testing_required" gorm:"default:false"`
	TestResults      TestResultArray `json:"test_results" gorm:"type:jsonb;default:'[]'"`

	// Audit
	CreatedAt        time.Time       `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time       `json:"updated_at" gorm:"autoUpdateTime"`
}

// ApprovalStage represents a single approval stage
type ApprovalStage struct {
	Stage        string     `json:"stage"`         // clinical_review, pharmacy_review, safety_review
	Approver     string     `json:"approver"`
	ApprovalDate *time.Time `json:"approval_date"`
	Status       string     `json:"status"`        // pending, approved, rejected
	Comments     string     `json:"comments"`
	Conditions   []string   `json:"conditions"`    // Conditions that must be met
}

// ApprovalStageArray handles JSONB storage for approval stages
type ApprovalStageArray []ApprovalStage

func (a *ApprovalStageArray) Scan(value interface{}) error {
	if value == nil {
		*a = ApprovalStageArray{}
		return nil
	}
	var bytes []byte
	switch val := value.(type) {
	case []byte:
		bytes = val
	case string:
		bytes = []byte(val)
	default:
		return fmt.Errorf("cannot scan %T into ApprovalStageArray", value)
	}
	return json.Unmarshal(bytes, a)
}

func (a ApprovalStageArray) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// TestResult represents a test result for clinical validation
type TestResult struct {
	TestID      string    `json:"test_id"`
	TestType    string    `json:"test_type"`    // unit, integration, clinical
	Status      string    `json:"status"`       // passed, failed, pending
	ExecutedAt  time.Time `json:"executed_at"`
	ExecutedBy  string    `json:"executed_by"`
	Results     string    `json:"results"`
	Errors      []string  `json:"errors,omitempty"`
}

// TestResultArray handles JSONB storage for test results
type TestResultArray []TestResult

func (t *TestResultArray) Scan(value interface{}) error {
	if value == nil {
		*t = TestResultArray{}
		return nil
	}
	var bytes []byte
	switch val := value.(type) {
	case []byte:
		bytes = val
	case string:
		bytes = []byte(val)
	default:
		return fmt.Errorf("cannot scan %T into TestResultArray", value)
	}
	return json.Unmarshal(bytes, t)
}

func (t TestResultArray) Value() (driver.Value, error) {
	return json.Marshal(t)
}

// DrugRuleDiff represents differences between versions
type DrugRuleDiff struct {
	DrugID           string         `json:"drug_id"`
	OldVersion       string         `json:"old_version"`
	NewVersion       string         `json:"new_version"`
	Changes          []ChangeDetail `json:"changes"`
	ImpactSummary    ImpactAnalysis `json:"impact_summary"`
	AffectedPatients int            `json:"affected_patients"`
	GeneratedAt      time.Time      `json:"generated_at"`
	GeneratedBy      string         `json:"generated_by"`
}

// ChangeDetail represents a single change between versions
type ChangeDetail struct {
	Path       string      `json:"path"`        // JSON path to changed field
	ChangeType string      `json:"type"`        // added, modified, deleted
	OldValue   interface{} `json:"old_value"`
	NewValue   interface{} `json:"new_value"`
	Clinical   bool        `json:"clinical"`    // Is this a clinical change?
	RiskLevel  string      `json:"risk_level"`  // low, medium, high, critical
}

// ImpactAnalysis represents the clinical impact of changes
type ImpactAnalysis struct {
	DoseChanges        []DoseImpact        `json:"dose_changes"`
	SafetyChanges      []SafetyImpact      `json:"safety_changes"`
	InteractionChanges []InteractionImpact `json:"interaction_changes"`
	EstimatedImpact    string              `json:"estimated_impact"`
	RiskScore          float64             `json:"risk_score"` // 0-100
}

// DoseImpact represents dose-related changes
type DoseImpact struct {
	Field       string  `json:"field"`
	OldDose     float64 `json:"old_dose"`
	NewDose     float64 `json:"new_dose"`
	PercentChange float64 `json:"percent_change"`
	Significance string  `json:"significance"` // minor, moderate, major
}

// SafetyImpact represents safety-related changes
type SafetyImpact struct {
	Type        string   `json:"type"`         // contraindication, warning, precaution
	Added       []string `json:"added"`
	Removed     []string `json:"removed"`
	Modified    []string `json:"modified"`
	RiskLevel   string   `json:"risk_level"`
}

// InteractionImpact represents drug interaction changes
type InteractionImpact struct {
	DrugClass     string   `json:"drug_class"`
	InteractionType string `json:"interaction_type"`
	Severity      string   `json:"severity"`
	Changes       []string `json:"changes"`
}
