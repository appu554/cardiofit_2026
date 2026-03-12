// Package types defines core data models for KB-16 Lab Interpretation & Trending Service
package types

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// =============================================================================
// LAB RESULT MODELS
// =============================================================================

// LabResult represents a single laboratory test result
type LabResult struct {
	ID             uuid.UUID       `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	PatientID      string          `json:"patient_id" gorm:"index;not null"`
	Code           string          `json:"code" gorm:"index;not null"`       // LOINC code
	Name           string          `json:"name" gorm:"not null"`             // Test name
	ValueNumeric   *float64        `json:"value_numeric,omitempty"`          // Numeric value
	ValueString    string          `json:"value_string,omitempty"`           // String value (for non-numeric)
	Unit           string          `json:"unit,omitempty"`                   // Unit of measurement
	ReferenceRange *ReferenceRange `json:"reference_range,omitempty" gorm:"embedded;embeddedPrefix:ref_"`
	CollectedAt    time.Time       `json:"collected_at" gorm:"index;not null"`
	ReportedAt     time.Time       `json:"reported_at" gorm:"not null"`
	Status         ResultStatus    `json:"status" gorm:"default:'final'"`    // final, preliminary, corrected
	Performer      string          `json:"performer,omitempty"`              // Lab/performer ID
	EncounterID    string          `json:"encounter_id,omitempty" gorm:"index"`
	SpecimenID     string          `json:"specimen_id,omitempty"`
	OrderID        string          `json:"order_id,omitempty"`
	Notes          string          `json:"notes,omitempty"`
	CreatedAt      time.Time       `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time       `json:"updated_at" gorm:"autoUpdateTime"`
}

func (LabResult) TableName() string {
	return "lab_results"
}

// ResultStatus represents the status of a lab result
type ResultStatus string

const (
	ResultStatusFinal       ResultStatus = "final"
	ResultStatusPreliminary ResultStatus = "preliminary"
	ResultStatusCorrected   ResultStatus = "corrected"
	ResultStatusCancelled   ResultStatus = "cancelled"
)

// ReferenceRange defines normal value ranges for a test
type ReferenceRange struct {
	Low          *float64 `json:"low,omitempty"`
	High         *float64 `json:"high,omitempty"`
	CriticalLow  *float64 `json:"critical_low,omitempty"`
	CriticalHigh *float64 `json:"critical_high,omitempty"`
	PanicLow     *float64 `json:"panic_low,omitempty"`
	PanicHigh    *float64 `json:"panic_high,omitempty"`
	Text         string   `json:"text,omitempty"`         // Textual description
	AgeSpecific  bool     `json:"age_specific,omitempty"` // Whether range is age-adjusted
	SexSpecific  bool     `json:"sex_specific,omitempty"` // Whether range is sex-adjusted
}

// =============================================================================
// INTERPRETATION MODELS
// =============================================================================

// InterpretedResult combines a lab result with its clinical interpretation
type InterpretedResult struct {
	Result          LabResult           `json:"result"`
	Interpretation  Interpretation      `json:"interpretation"`
	Trending        *TrendAnalysis      `json:"trending,omitempty"`
	BaselineCompare *BaselineComparison `json:"baseline_comparison,omitempty"`
}

// Interpretation represents clinical interpretation of a lab result
type Interpretation struct {
	ID                 uuid.UUID          `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ResultID           uuid.UUID          `json:"result_id" gorm:"type:uuid;index"`
	Flag               InterpretationFlag `json:"flag"`                 // NORMAL, LOW, HIGH, etc.
	Severity           Severity           `json:"severity"`              // LOW, MEDIUM, HIGH, CRITICAL
	IsCritical         bool               `json:"is_critical"`
	IsPanic            bool               `json:"is_panic"`
	RequiresAction     bool               `json:"requires_action"`
	DeviationPercent   float64            `json:"deviation_percent,omitempty"`
	DeviationDirection string             `json:"deviation_direction,omitempty"` // above, below
	DeltaCheck         *DeltaCheckResult  `json:"delta_check,omitempty" gorm:"serializer:json"`
	ClinicalComment    string             `json:"clinical_comment,omitempty"`
	Recommendations    []Recommendation   `json:"recommendations,omitempty" gorm:"serializer:json"`
	CreatedAt          time.Time          `json:"created_at" gorm:"autoCreateTime"`
}

func (Interpretation) TableName() string {
	return "interpretations"
}

// InterpretationFlag represents the classification of a lab value
type InterpretationFlag string

const (
	FlagNormal       InterpretationFlag = "NORMAL"
	FlagLow          InterpretationFlag = "LOW"
	FlagHigh         InterpretationFlag = "HIGH"
	FlagCriticalLow  InterpretationFlag = "CRITICAL_LOW"
	FlagCriticalHigh InterpretationFlag = "CRITICAL_HIGH"
	FlagPanicLow     InterpretationFlag = "PANIC_LOW"
	FlagPanicHigh    InterpretationFlag = "PANIC_HIGH"
)

// Severity represents clinical severity level
type Severity string

const (
	SeverityLow      Severity = "LOW"
	SeverityMedium   Severity = "MEDIUM"
	SeverityHigh     Severity = "HIGH"
	SeverityCritical Severity = "CRITICAL"
)

// DeltaCheckResult represents the result of delta (change) checking
type DeltaCheckResult struct {
	PreviousValue    float64   `json:"previous_value"`
	PreviousTime     time.Time `json:"previous_time"`
	Change           float64   `json:"change"`
	PercentChange    float64   `json:"percent_change"`
	WindowHours      int       `json:"window_hours"`
	IsSignificant    bool      `json:"is_significant"`
	AlertGenerated   bool      `json:"alert_generated"`
	ThresholdType    string    `json:"threshold_type,omitempty"` // absolute, percent
	ThresholdValue   float64   `json:"threshold_value,omitempty"`
}

// Recommendation represents a clinical recommendation for a lab result
type Recommendation struct {
	Type        string `json:"type"`        // action, consultation, follow_up, urgent, notify, clinical_action
	Priority    string `json:"priority"`    // CRITICAL, HIGH, MEDIUM, LOW
	Description string `json:"description"` // Human-readable recommendation text
}

// =============================================================================
// TRENDING MODELS
// =============================================================================

// TrendAnalysis represents multi-window trend analysis for a lab test
type TrendAnalysis struct {
	TestCode       string                 `json:"test_code"`
	PatientID      string                 `json:"patient_id"`
	WindowDays     int                    `json:"window_days"`
	DataPoints     []TrendDataPoint       `json:"data_points"`
	Windows        map[string]TrendWindow `json:"windows,omitempty"` // 7d, 30d, 90d, 1yr
	Trajectory     Trajectory             `json:"trajectory"`
	RateOfChange   float64                `json:"rate_of_change,omitempty"` // units per day
	Statistics     TrendStatistics        `json:"statistics"`
	Prediction     *PredictedValue        `json:"prediction,omitempty"`
	AnalyzedAt     time.Time              `json:"analyzed_at"`
	DataPointCount int                    `json:"data_point_count"`
}

// TrendDataPoint represents a single data point in a trend analysis
type TrendDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	ResultID  string    `json:"result_id,omitempty"`
}

// TrendStatistics contains statistical measures for trend analysis
type TrendStatistics struct {
	Mean                   float64 `json:"mean"`
	StdDev                 float64 `json:"std_dev"`
	Min                    float64 `json:"min"`
	Max                    float64 `json:"max"`
	Median                 float64 `json:"median"`
	CoefficientOfVariation float64 `json:"coefficient_of_variation"`
	SampleCount            int     `json:"sample_count"`
}

// TrendWindow represents trend data for a specific time window
type TrendWindow struct {
	Name       string      `json:"name"`        // "7 days", "30 days", etc.
	Days       int         `json:"days"`
	DataPoints []DataPoint `json:"data_points"`
	Statistics *Statistics `json:"statistics,omitempty"`
	Trend      string      `json:"trend,omitempty"` // increasing, decreasing, stable
	Slope      float64     `json:"slope,omitempty"`
	RSquared   float64     `json:"r_squared,omitempty"` // Correlation coefficient
}

// DataPoint represents a single data point in a trend
type DataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Flag      string    `json:"flag,omitempty"`
}

// Statistics contains statistical measures for a set of values
type Statistics struct {
	Mean              float64 `json:"mean"`
	Median            float64 `json:"median"`
	StdDev            float64 `json:"std_dev"`
	Min               float64 `json:"min"`
	Max               float64 `json:"max"`
	Count             int     `json:"count"`
	CoefficientOfVar  float64 `json:"coefficient_of_variation,omitempty"`
}

// Trajectory represents the direction of change over time
type Trajectory string

const (
	TrajectoryImproving Trajectory = "IMPROVING"
	TrajectoryStable    Trajectory = "STABLE"
	TrajectoryWorsening Trajectory = "WORSENING"
	TrajectoryVolatile  Trajectory = "VOLATILE"
	TrajectoryUnknown   Trajectory = "UNKNOWN"
)

// PredictedValue represents a forecasted future value
type PredictedValue struct {
	Value         float64   `json:"value"`
	PredictedAt   time.Time `json:"predicted_at"`
	Confidence    float64   `json:"confidence"`
	BasedOnPoints int       `json:"based_on_points"`
	Method        string    `json:"method,omitempty"` // linear, exponential
}

// TrendWindowConfig defines configuration for a trending window
type TrendWindowConfig struct {
	Name      string `json:"name"`
	Days      int    `json:"days"`
	MinPoints int    `json:"min_points"`
	UseCase   string `json:"use_case"`
}

// =============================================================================
// BASELINE MODELS
// =============================================================================

// PatientBaseline represents a patient-specific baseline for a test
type PatientBaseline struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	PatientID    string         `json:"patient_id" gorm:"uniqueIndex:idx_baseline_patient_code;not null"`
	Code         string         `json:"code" gorm:"uniqueIndex:idx_baseline_patient_code;not null"`
	Mean         float64        `json:"mean" gorm:"not null"`
	StdDev       float64        `json:"std_dev,omitempty"`
	Min          float64        `json:"min,omitempty"`
	Max          float64        `json:"max,omitempty"`
	SampleCount  int            `json:"sample_count"`
	Source       BaselineSource `json:"source" gorm:"default:'CALCULATED'"`
	SetBy        string         `json:"set_by,omitempty"`
	LookbackDays int            `json:"lookback_days,omitempty"`
	Notes        string         `json:"notes,omitempty"`
	LastUpdated  time.Time      `json:"last_updated" gorm:"autoUpdateTime"`
	CreatedAt    time.Time      `json:"created_at" gorm:"autoCreateTime"`
}

func (PatientBaseline) TableName() string {
	return "patient_baselines"
}

// BaselineSource indicates how the baseline was determined
type BaselineSource string

const (
	BaselineSourceCalculated BaselineSource = "CALCULATED"
	BaselineSourceManual     BaselineSource = "MANUAL"
	BaselineSourceImported   BaselineSource = "IMPORTED"
)

// BaselineComparison shows how a result compares to patient baseline
type BaselineComparison struct {
	Baseline         PatientBaseline `json:"baseline"`
	CurrentValue     float64         `json:"current_value"`
	ZScore           float64         `json:"z_score"`             // Standard deviations from mean
	PercentDeviation float64         `json:"percent_deviation"`   // Percent from mean
	IsSignificant    bool            `json:"is_significant"`      // > 2 SD from mean
	Direction        string          `json:"direction"`           // above, below, within
}

// =============================================================================
// PANEL MODELS
// =============================================================================

// PanelType represents a type of lab panel
type PanelType string

const (
	PanelBMP     PanelType = "BMP"     // Basic Metabolic Panel
	PanelCMP     PanelType = "CMP"     // Comprehensive Metabolic Panel
	PanelCBC     PanelType = "CBC"     // Complete Blood Count
	PanelLFT     PanelType = "LFT"     // Liver Function Tests
	PanelLipid   PanelType = "LIPID"   // Lipid Panel
	PanelThyroid PanelType = "THYROID" // Thyroid Panel
	PanelRenal   PanelType = "RENAL"   // Renal Function Panel
	PanelCoag    PanelType = "COAG"    // Coagulation Panel
	PanelCardiac PanelType = "CARDIAC" // Cardiac Panel
)

// PanelDefinition defines a lab panel and its components
type PanelDefinition struct {
	Type             PanelType `json:"type"`
	Name             string    `json:"name"`
	Description      string    `json:"description,omitempty"`
	Components       []string  `json:"components"`        // LOINC codes
	CalculatedValues []string  `json:"calculated_values,omitempty"` // e.g., anion_gap, egfr
	Patterns         []string  `json:"patterns,omitempty"` // Detectable patterns
}

// AssembledPanel represents a panel assembled from individual results
type AssembledPanel struct {
	Type               PanelType              `json:"type"`
	Name               string                 `json:"name"`
	PatientID          string                 `json:"patient_id"`
	AssembledAt        time.Time              `json:"assembled_at"`
	Components         []PanelComponent       `json:"components"`
	Completeness       float64                `json:"completeness"`       // 0-100%
	MissingComponents  []string               `json:"missing_components,omitempty"`
	CalculatedValues   map[string]float64     `json:"calculated_values,omitempty"`
	DetectedPatterns   []DetectedPattern      `json:"detected_patterns,omitempty"`
	PanelInterpretation string                `json:"panel_interpretation,omitempty"`
}

// PanelComponent represents a single component of a lab panel
type PanelComponent struct {
	Code      string     `json:"code"`
	Name      string     `json:"name"`
	Required  bool       `json:"required"`
	Available bool       `json:"available"`
	Result    *LabResult `json:"result,omitempty"`
}

// AvailablePanel represents a panel that can be assembled from patient data
type AvailablePanel struct {
	Type         PanelType `json:"type"`
	Name         string    `json:"name"`
	Completeness float64   `json:"completeness"` // 0-100%
	Available    int       `json:"available"`    // Number of available components
	Total        int       `json:"total"`        // Total components needed
}

// DetectedPattern represents a clinical pattern detected from panel results
type DetectedPattern struct {
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Confidence  float64  `json:"confidence"`    // 0-1
	Severity    Severity `json:"severity,omitempty"`
	Evidence    []string `json:"evidence,omitempty"` // Supporting findings
}

// =============================================================================
// REVIEW WORKFLOW MODELS
// =============================================================================

// ResultReview tracks the review status of a lab result
type ResultReview struct {
	ID             uuid.UUID    `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ResultID       uuid.UUID    `json:"result_id" gorm:"type:uuid;index;not null"`
	Status         ReviewStatus `json:"status" gorm:"index;default:'PENDING'"`
	AcknowledgedBy string       `json:"acknowledged_by,omitempty"`
	AcknowledgedAt *time.Time   `json:"acknowledged_at,omitempty"`
	ReviewedBy     string       `json:"reviewed_by,omitempty"`
	ReviewedAt     *time.Time   `json:"reviewed_at,omitempty"`
	ReviewNotes    string       `json:"review_notes,omitempty"`
	ActionTaken    string       `json:"action_taken,omitempty"`
	KB14TaskID     string       `json:"kb14_task_id,omitempty"`
	CreatedAt      time.Time    `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time    `json:"updated_at" gorm:"autoUpdateTime"`
}

func (ResultReview) TableName() string {
	return "result_reviews"
}

// ReviewStatus represents the review state of a result
type ReviewStatus string

const (
	ReviewStatusPending      ReviewStatus = "PENDING"
	ReviewStatusCritical     ReviewStatus = "CRITICAL"
	ReviewStatusAcknowledged ReviewStatus = "ACKNOWLEDGED"
	ReviewStatusInProgress   ReviewStatus = "IN_PROGRESS"
	ReviewStatusCompleted    ReviewStatus = "COMPLETED"
	ReviewStatusActioned     ReviewStatus = "ACTIONED"
)

// CriticalResult represents a result requiring urgent review
type CriticalResult struct {
	ReviewID     uuid.UUID `json:"review_id"`
	ResultID     uuid.UUID `json:"result_id"`
	CreatedAt    time.Time `json:"created_at"`
	WaitMinutes  int       `json:"wait_minutes"`
	SLABreached  bool      `json:"sla_breached"`
	KB14TaskID   string    `json:"kb14_task_id,omitempty"`
}

// PendingReview represents a result pending review
type PendingReview struct {
	ResultID       uuid.UUID          `json:"result_id"`
	PatientID      string             `json:"patient_id"`
	Code           string             `json:"code"`
	Name           string             `json:"name"`
	Value          string             `json:"value"` // Formatted value with unit
	Flag           InterpretationFlag `json:"flag"`
	CollectedAt    time.Time          `json:"collected_at"`
	IsCritical     bool               `json:"is_critical"`
	IsPanic        bool               `json:"is_panic"`
	PendingSince   time.Time          `json:"pending_since"`
}

// PendingReviewFilters contains filters for pending review queries
type PendingReviewFilters struct {
	Priority  string `json:"priority,omitempty"` // critical, high, normal
	PatientID string `json:"patient_id,omitempty"`
	Code      string `json:"code,omitempty"`
	Page      int    `json:"page"`
	Limit     int    `json:"limit"`
}

// ReviewStats contains review workflow statistics
type ReviewStats struct {
	Total                     int     `json:"total"`
	Pending                   int     `json:"pending"`
	Critical                  int     `json:"critical"`
	Acknowledged              int     `json:"acknowledged"`
	Completed                 int     `json:"completed"`
	AvgAcknowledgmentMinutes  float64 `json:"avg_acknowledgment_minutes"`
	SLABreaches               int     `json:"sla_breaches"`
}

// =============================================================================
// VISUALIZATION MODELS
// =============================================================================

// ChartData represents data formatted for chart rendering
type ChartData struct {
	TestCode       string               `json:"test_code"`
	TestName       string               `json:"test_name"`
	Unit           string               `json:"unit"`
	Window         string               `json:"window"`
	WindowDays     int                  `json:"window_days"`
	DataPoints     []ChartDataPoint     `json:"data_points"`
	ReferenceRange *ChartReferenceRange `json:"reference_range,omitempty"`
	Baseline       *ChartBaseline       `json:"baseline,omitempty"`
	Annotations    []ChartAnnotation    `json:"annotations,omitempty"`
}

// ChartDataPoint represents a single point on a chart
type ChartDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	ResultID  string    `json:"result_id,omitempty"`
	Status    string    `json:"status,omitempty"`
}

// ChartReferenceRange represents reference range for chart display
type ChartReferenceRange struct {
	Low          float64  `json:"low"`
	High         float64  `json:"high"`
	CriticalLow  *float64 `json:"critical_low,omitempty"`
	CriticalHigh *float64 `json:"critical_high,omitempty"`
}

// ChartBaseline represents baseline for chart display
type ChartBaseline struct {
	Mean   float64 `json:"mean"`
	StdDev float64 `json:"std_dev"`
	Upper  float64 `json:"upper"` // Mean + 2*StdDev
	Lower  float64 `json:"lower"` // Mean - 2*StdDev
}

// ChartAnnotation represents a point of interest on a chart
type ChartAnnotation struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`  // critical_low, critical_high, panic_low, panic_high
	Label     string    `json:"label"`
	Value     float64   `json:"value"`
}

// SparklineData represents compact trend visualization data
type SparklineData struct {
	Code           string    `json:"code"`
	Values         []float64 `json:"values"`
	Min            float64   `json:"min"`
	Max            float64   `json:"max"`
	Latest         float64   `json:"latest"`
	Trend          string    `json:"trend"`  // up, down, stable
	Status         string    `json:"status"` // normal, low, high
	DataPointCount int       `json:"data_point_count"`
}

// DashboardData represents patient lab dashboard data
type DashboardData struct {
	PatientID   string           `json:"patient_id"`
	GeneratedAt time.Time        `json:"generated_at"`
	Panels      []DashboardPanel `json:"panels"`
	AlertCount  int              `json:"alert_count"`
}

// DashboardPanel represents a category of tests on the dashboard
type DashboardPanel struct {
	Category string                 `json:"category"`
	Tests    []DashboardTestSummary `json:"tests"`
}

// DashboardTestSummary represents a test summary for the dashboard
type DashboardTestSummary struct {
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Unit      string    `json:"unit"`
	Latest    float64   `json:"latest"`
	Trend     string    `json:"trend"`  // up, down, stable
	Status    string    `json:"status"` // normal, low, high
	Sparkline []float64 `json:"sparkline"`
}

// =============================================================================
// PATIENT CONTEXT MODELS (for KB-2 integration)
// =============================================================================

// PatientContext provides clinical context for interpretation
type PatientContext struct {
	PatientID    string            `json:"patient_id"`
	Age          int               `json:"age"`
	AgeInDays    int               `json:"age_days,omitempty"` // Precise age for neonates
	Sex          string            `json:"sex"` // male, female, other
	Conditions   []Condition       `json:"conditions,omitempty"`
	Medications  []Medication      `json:"medications,omitempty"`
	Phenotypes   []string          `json:"phenotypes,omitempty"`

	// Enhanced fields for Phase 3b.6 context-aware interpretation
	// Pregnancy Status
	IsPregnant      bool `json:"is_pregnant,omitempty"`
	Trimester       int  `json:"trimester,omitempty"`
	GestationalWeek int  `json:"gestational_week,omitempty"`
	IsPostpartum    bool `json:"is_postpartum,omitempty"`
	IsLactating     bool `json:"is_lactating,omitempty"`

	// Neonatal Parameters (AAP 2022 bilirubin guidelines)
	IsNeonate             bool   `json:"is_neonate,omitempty"`
	GestationalAgeAtBirth int    `json:"gestational_age_at_birth,omitempty"`
	HoursOfLife           int    `json:"hours_of_life,omitempty"`
	NeonatalRiskCategory  string `json:"neonatal_risk_category,omitempty"` // LOW, MEDIUM, HIGH

	// Renal Status (KDIGO guidelines)
	CKDStage     int     `json:"ckd_stage,omitempty"`
	EGFR         float64 `json:"egfr,omitempty"`
	IsOnDialysis bool    `json:"is_on_dialysis,omitempty"`

	// Hepatic Status
	ChildPughClass string `json:"child_pugh_class,omitempty"`
}

// HasRiskFactor checks if patient has any of the specified risk factors
func (p *PatientContext) HasRiskFactor(factors ...string) bool {
	factorSet := make(map[string]bool)
	for _, f := range factors {
		factorSet[f] = true
	}
	for _, cond := range p.Conditions {
		if factorSet[cond.Code] || factorSet[cond.Name] {
			return true
		}
	}
	if p.IsPregnant && factorSet["pregnancy"] {
		return true
	}
	if p.CKDStage >= 4 && factorSet["ckd_severe"] {
		return true
	}
	if p.IsOnDialysis && factorSet["dialysis"] {
		return true
	}
	return false
}

// Condition represents a patient condition
type Condition struct {
	Code      string    `json:"code"`      // ICD-10 or SNOMED
	System    string    `json:"system"`    // ICD10, SNOMED
	Name      string    `json:"name"`
	OnsetDate time.Time `json:"onset_date,omitempty"`
	Severity  string    `json:"severity,omitempty"`
}

// Medication represents a patient medication
type Medication struct {
	RxNormCode string    `json:"rxnorm_code"`
	Name       string    `json:"name"`
	Dose       string    `json:"dose,omitempty"`
	Frequency  string    `json:"frequency,omitempty"`
	StartDate  time.Time `json:"start_date,omitempty"`
}

// =============================================================================
// API REQUEST/RESPONSE MODELS
// =============================================================================

// StoreResultRequest represents a request to store a lab result
type StoreResultRequest struct {
	PatientID      string          `json:"patient_id" binding:"required"`
	Code           string          `json:"code" binding:"required"`
	Name           string          `json:"name" binding:"required"`
	ValueNumeric   *float64        `json:"value_numeric,omitempty"`
	ValueString    string          `json:"value_string,omitempty"`
	Unit           string          `json:"unit,omitempty"`
	ReferenceRange *ReferenceRange `json:"reference_range,omitempty"`
	CollectedAt    time.Time       `json:"collected_at" binding:"required"`
	ReportedAt     *time.Time      `json:"reported_at,omitempty"`
	Status         ResultStatus    `json:"status,omitempty"`
	Performer      string          `json:"performer,omitempty"`
	EncounterID    string          `json:"encounter_id,omitempty"`
	SpecimenID     string          `json:"specimen_id,omitempty"`
	OrderID        string          `json:"order_id,omitempty"`
}

// InterpretRequest represents a request to interpret a lab result
type InterpretRequest struct {
	Result         LabResult       `json:"result" binding:"required"`
	PatientContext *PatientContext `json:"patient_context,omitempty"`
	IncludeTrending bool           `json:"include_trending,omitempty"`
	IncludeBaseline bool           `json:"include_baseline,omitempty"`
}

// BatchInterpretRequest represents a request to interpret multiple results
type BatchInterpretRequest struct {
	Results        []LabResult     `json:"results" binding:"required"`
	PatientContext *PatientContext `json:"patient_context,omitempty"`
}

// TrendRequest represents a request for trend analysis
type TrendRequest struct {
	PatientID string   `json:"patient_id" binding:"required"`
	Code      string   `json:"code" binding:"required"`
	Windows   []string `json:"windows,omitempty"` // 7d, 30d, 90d, 1yr
}

// SetBaselineRequest represents a request to manually set a baseline
type SetBaselineRequest struct {
	Mean   float64 `json:"mean" binding:"required"`
	StdDev float64 `json:"std_dev,omitempty"`
	SetBy  string  `json:"set_by" binding:"required"`
	Notes  string  `json:"notes,omitempty"`
}

// AcknowledgeRequest represents a request to acknowledge a result
type AcknowledgeRequest struct {
	ResultID       string `json:"result_id" binding:"required"`
	AcknowledgedBy string `json:"acknowledged_by" binding:"required"`
}

// CompleteReviewRequest represents a request to complete a result review
type CompleteReviewRequest struct {
	ResultID    string `json:"result_id" binding:"required"`
	ReviewedBy  string `json:"reviewed_by" binding:"required"`
	ReviewNotes string `json:"review_notes,omitempty"`
	ActionTaken string `json:"action_taken,omitempty"`
}

// =============================================================================
// RESPONSE WRAPPERS
// =============================================================================

// APIResponse is a standard API response wrapper
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    *APIMeta    `json:"meta,omitempty"`
}

// APIError represents an error response
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// APIMeta contains metadata about the response
type APIMeta struct {
	RequestID   string `json:"request_id,omitempty"`
	ProcessTime int64  `json:"process_time_ms,omitempty"`
	Total       int    `json:"total,omitempty"`
	Page        int    `json:"page,omitempty"`
	PageSize    int    `json:"page_size,omitempty"`
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// Ptr returns a pointer to the value
func Ptr[T any](v T) *T {
	return &v
}

// DecimalFromFloat creates a Decimal from a float64
func DecimalFromFloat(f float64) decimal.Decimal {
	return decimal.NewFromFloat(f)
}

// FormatValue formats a numeric value with its unit
func FormatValue(value *float64, unit string) string {
	if value == nil {
		return ""
	}
	if unit == "" {
		return decimal.NewFromFloat(*value).String()
	}
	return decimal.NewFromFloat(*value).String() + " " + unit
}

// MarshalJSON for ReferenceRange handles nil values
func (r ReferenceRange) MarshalJSON() ([]byte, error) {
	type Alias ReferenceRange
	return json.Marshal(&struct {
		Alias
	}{
		Alias: Alias(r),
	})
}
