// Package models provides domain models for KB-13 Quality Measures Engine.
//
// These models follow FHIR R4 MeasureReport semantics and support
// HEDIS, CMS, NQF, and custom quality measures.
package models

import (
	"time"
)

// MeasureType represents the type of quality measure
type MeasureType string

const (
	MeasureTypeProcess      MeasureType = "PROCESS"
	MeasureTypeOutcome      MeasureType = "OUTCOME"
	MeasureTypeStructure    MeasureType = "STRUCTURE"
	MeasureTypeEfficiency   MeasureType = "EFFICIENCY"
	MeasureTypeComposite    MeasureType = "COMPOSITE"
	MeasureTypeIntermediate MeasureType = "INTERMEDIATE"
)

// ScoringType represents how the measure is scored
type ScoringType string

const (
	ScoringProportion  ScoringType = "proportion"
	ScoringRatio       ScoringType = "ratio"
	ScoringContinuous  ScoringType = "continuous"
	ScoringComposite   ScoringType = "composite"
)

// ClinicalDomain represents the clinical area of the measure
type ClinicalDomain string

const (
	DomainDiabetes        ClinicalDomain = "DIABETES"
	DomainCardiovascular  ClinicalDomain = "CARDIOVASCULAR"
	DomainRespiratory     ClinicalDomain = "RESPIRATORY"
	DomainPreventive      ClinicalDomain = "PREVENTIVE"
	DomainBehavioralHealth ClinicalDomain = "BEHAVIORAL_HEALTH"
	DomainMaternal        ClinicalDomain = "MATERNAL"
	DomainPediatric       ClinicalDomain = "PEDIATRIC"
	DomainPatientSafety   ClinicalDomain = "PATIENT_SAFETY"
)

// QualityProgram represents the quality program the measure belongs to
type QualityProgram string

const (
	ProgramHEDIS  QualityProgram = "HEDIS"
	ProgramCMS    QualityProgram = "CMS"
	ProgramMIPS   QualityProgram = "MIPS"
	ProgramACO    QualityProgram = "ACO"
	ProgramPCMH   QualityProgram = "PCMH"
	ProgramNQF    QualityProgram = "NQF"
	ProgramCustom QualityProgram = "CUSTOM"
)

// PopulationType represents the type of population criteria
type PopulationType string

const (
	PopulationInitial              PopulationType = "initial-population"
	PopulationDenominator          PopulationType = "denominator"
	PopulationDenominatorExclusion PopulationType = "denominator-exclusion"
	PopulationDenominatorException PopulationType = "denominator-exception"
	PopulationNumerator            PopulationType = "numerator"
	PopulationNumeratorExclusion   PopulationType = "numerator-exclusion"
	PopulationMeasurePopulation    PopulationType = "measure-population"
	PopulationMeasureObservation   PopulationType = "measure-observation"
)

// ReportType represents the type of measure report
type ReportType string

const (
	ReportIndividual  ReportType = "individual"
	ReportSubjectList ReportType = "subject-list"
	ReportSummary     ReportType = "summary"
	ReportDataExchange ReportType = "data-exchange"
)

// Priority represents care gap priority levels
type Priority string

const (
	PriorityCritical Priority = "critical"
	PriorityHigh     Priority = "high"
	PriorityMedium   Priority = "medium"
	PriorityLow      Priority = "low"
)

// CareGapStatus represents the status of a care gap
type CareGapStatus string

const (
	CareGapStatusOpen       CareGapStatus = "open"
	CareGapStatusInProgress CareGapStatus = "in-progress"
	CareGapStatusClosed     CareGapStatus = "closed"
	CareGapStatusDeferred   CareGapStatus = "deferred"
)

// CareGapSource indicates where the care gap was identified
// 🔴 CRITICAL: This distinguishes KB-13 (derived) from KB-9 (authoritative)
type CareGapSource string

const (
	CareGapSourceQualityMeasure CareGapSource = "QUALITY_MEASURE" // KB-13 (derived)
	CareGapSourcePatientCDS     CareGapSource = "PATIENT_CDS"     // KB-9 (authoritative)
)

// Measure represents a quality measure definition
type Measure struct {
	ID                  string            `json:"id" yaml:"id"`
	Version             string            `json:"version" yaml:"version"`
	Name                string            `json:"name" yaml:"name"`
	Title               string            `json:"title" yaml:"title"`
	Description         string            `json:"description,omitempty" yaml:"description"`
	Type                MeasureType       `json:"type" yaml:"type"`
	Scoring             ScoringType       `json:"scoring" yaml:"scoring"`
	Domain              ClinicalDomain    `json:"domain" yaml:"domain"`
	Program             QualityProgram    `json:"program" yaml:"program"`
	NQFNumber           string            `json:"nqf_number,omitempty" yaml:"nqf_number"`
	CMSNumber           string            `json:"cms_number,omitempty" yaml:"cms_number"`
	HEDISCode           string            `json:"hedis_code,omitempty" yaml:"hedis_code"`
	MeasurementPeriod   MeasurementPeriod `json:"measurement_period" yaml:"measurement_period"`
	Populations         []Population      `json:"populations" yaml:"populations"`
	Stratifications     []Stratification  `json:"stratifications,omitempty" yaml:"stratifications"`
	ImprovementNotation string            `json:"improvement_notation" yaml:"improvement_notation"`
	BenchmarkRef        string            `json:"benchmark_ref,omitempty" yaml:"benchmark_ref"`
	Evidence            Evidence          `json:"evidence,omitempty" yaml:"evidence"`
	Active              bool              `json:"active" yaml:"active"`
	CalculationSchedule []string          `json:"calculation_schedule,omitempty" yaml:"calculation_schedule"` // daily, weekly, monthly, quarterly
	CreatedAt           time.Time         `json:"created_at,omitempty"`
	UpdatedAt           time.Time         `json:"updated_at,omitempty"`
}

// MeasurementPeriod defines the time period for measure calculation
type MeasurementPeriod struct {
	Type     string `json:"type" yaml:"type"`         // "rolling" or "calendar"
	Duration string `json:"duration" yaml:"duration"` // ISO 8601 duration (e.g., "P1Y")
	Anchor   string `json:"anchor,omitempty" yaml:"anchor"` // For calendar: "year", "quarter", "month"
}

// Population represents a population criterion in a measure
type Population struct {
	ID            string         `json:"id" yaml:"id"`
	Type          PopulationType `json:"type" yaml:"type"`
	Description   string         `json:"description,omitempty" yaml:"description"`
	CQLExpression string         `json:"cql_expression" yaml:"cql_expression"`
	Criteria      *Criteria      `json:"criteria,omitempty" yaml:"criteria"`
}

// Criteria defines additional criteria for population filtering
type Criteria struct {
	LabResults    []LabCriterion    `json:"lab_results,omitempty" yaml:"lab_results"`
	Conditions    []ConditionCriterion `json:"conditions,omitempty" yaml:"conditions"`
	Medications   []MedicationCriterion `json:"medications,omitempty" yaml:"medications"`
	Procedures    []ProcedureCriterion `json:"procedures,omitempty" yaml:"procedures"`
	Demographics  *DemographicCriterion `json:"demographics,omitempty" yaml:"demographics"`
}

// LabCriterion defines lab result criteria
type LabCriterion struct {
	LabCode    string  `json:"lab_code" yaml:"lab_code"`       // LOINC code
	Operator   string  `json:"operator" yaml:"operator"`       // <, >, <=, >=, =, between, in
	Value      float64 `json:"value,omitempty" yaml:"value"`
	ValueRange []float64 `json:"value_range,omitempty" yaml:"value_range"` // For "between"
	TimeWindow string  `json:"time_window,omitempty" yaml:"time_window"`
}

// ConditionCriterion defines condition/diagnosis criteria
type ConditionCriterion struct {
	ValueSetOID string `json:"value_set_oid" yaml:"value_set_oid"`
	TimeWindow  string `json:"time_window,omitempty" yaml:"time_window"`
}

// MedicationCriterion defines medication criteria
type MedicationCriterion struct {
	ValueSetOID string `json:"value_set_oid" yaml:"value_set_oid"`
	TimeWindow  string `json:"time_window,omitempty" yaml:"time_window"`
	ActiveOnly  bool   `json:"active_only,omitempty" yaml:"active_only"`
}

// ProcedureCriterion defines procedure criteria
type ProcedureCriterion struct {
	ValueSetOID string `json:"value_set_oid" yaml:"value_set_oid"`
	TimeWindow  string `json:"time_window,omitempty" yaml:"time_window"`
}

// DemographicCriterion defines demographic criteria
type DemographicCriterion struct {
	AgeMin  int    `json:"age_min,omitempty" yaml:"age_min"`
	AgeMax  int    `json:"age_max,omitempty" yaml:"age_max"`
	Gender  string `json:"gender,omitempty" yaml:"gender"`
}

// Stratification defines how measure results are stratified
type Stratification struct {
	ID          string   `json:"id" yaml:"id"`
	Description string   `json:"description,omitempty" yaml:"description"`
	Type        string   `json:"type,omitempty" yaml:"type"` // "age", "gender", "ethnicity", "payer"
	Components  []string `json:"components" yaml:"components"`
}

// Evidence links to clinical evidence supporting the measure
type Evidence struct {
	Level     string `json:"level,omitempty" yaml:"level"`     // A, B, C, D
	Source    string `json:"source,omitempty" yaml:"source"`
	Guideline string `json:"guideline,omitempty" yaml:"guideline"`
	Citation  string `json:"citation,omitempty" yaml:"citation"`
}

// Benchmark represents performance benchmarks for a measure
type Benchmark struct {
	MeasureID     string    `json:"measure_id" yaml:"measure_id"`
	Year          int       `json:"year" yaml:"year"`
	Source        string    `json:"source" yaml:"source"` // NCQA, CMS, etc.
	EffectiveDate time.Time `json:"effective_date" yaml:"effective_date"`
	Percentile25  float64   `json:"percentile_25,omitempty" yaml:"percentile_25"`
	Percentile50  float64   `json:"percentile_50,omitempty" yaml:"percentile_50"`
	Percentile75  float64   `json:"percentile_75,omitempty" yaml:"percentile_75"`
	Percentile90  float64   `json:"percentile_90,omitempty" yaml:"percentile_90"`
	Notes         string    `json:"notes,omitempty" yaml:"notes"`
}

// ExecutionContextVersion tracks versions for audit trail
// 🟡 REQUIRED for all calculations per CTO/CMO gate
type ExecutionContextVersion struct {
	KB13Version        string    `json:"kb13_version"`
	CQLLibraryVersion  string    `json:"cql_library_version"`
	TerminologyVersion string    `json:"terminology_version"`
	MeasureYAMLVersion string    `json:"measure_yaml_version"`
	ExecutedAt         time.Time `json:"executed_at"`
}

// CalculationResult represents the output of a measure calculation
type CalculationResult struct {
	ID                   string                  `json:"id"`
	MeasureID            string                  `json:"measure_id"`
	ReportType           ReportType              `json:"report_type"`
	PeriodStart          time.Time               `json:"period_start"`
	PeriodEnd            time.Time               `json:"period_end"`
	InitialPopulation    int                     `json:"initial_population"`
	Denominator          int                     `json:"denominator"`
	DenominatorExclusion int                     `json:"denominator_exclusion"`
	DenominatorException int                     `json:"denominator_exception"`
	Numerator            int                     `json:"numerator"`
	NumeratorExclusion   int                     `json:"numerator_exclusion"`
	Score                float64                 `json:"score"`
	Stratifications      []StratificationResult  `json:"stratifications,omitempty"`
	CareGaps             []CareGap               `json:"care_gaps,omitempty"`
	ExecutionTimeMs      int64                   `json:"execution_time_ms"`
	ExecutionContext     ExecutionContextVersion `json:"execution_context"`
	CreatedAt            time.Time               `json:"created_at"`
}

// StratificationResult contains results for a specific stratification
type StratificationResult struct {
	StratificationID string  `json:"stratification_id"`
	Component        string  `json:"component"`
	Denominator      int     `json:"denominator"`
	Numerator        int     `json:"numerator"`
	Score            float64 `json:"score"`
}

// CareGap represents an identified care gap
// 🔴 CRITICAL: KB-13 gaps are DERIVED, not authoritative
type CareGap struct {
	ID              string        `json:"id"`
	MeasureID       string        `json:"measure_id"`
	SubjectID       string        `json:"subject_id"`
	GapType         string        `json:"gap_type"`
	Description     string        `json:"description"`
	Priority        Priority      `json:"priority"`
	Status          CareGapStatus `json:"status"`
	DueDate         *time.Time    `json:"due_date,omitempty"`
	Intervention    string        `json:"intervention,omitempty"`
	Source          CareGapSource `json:"source"`           // 🔴 REQUIRED: "QUALITY_MEASURE"
	IsAuthoritative bool          `json:"is_authoritative"` // Always false for KB-13
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	ClosedAt        *time.Time    `json:"closed_at,omitempty"`
}

// NewCareGap creates a new care gap with proper KB-13 source annotation
func NewCareGap(measureID, subjectID, gapType, description string, priority Priority) *CareGap {
	now := time.Now()
	return &CareGap{
		MeasureID:       measureID,
		SubjectID:       subjectID,
		GapType:         gapType,
		Description:     description,
		Priority:        priority,
		Status:          CareGapStatusOpen,
		Source:          CareGapSourceQualityMeasure, // 🔴 Always set for KB-13
		IsAuthoritative: false,                       // 🔴 Always false for KB-13
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// CalculationJob represents an async calculation job
type CalculationJob struct {
	ID          string     `json:"id"`
	MeasureID   string     `json:"measure_id"`
	ReportType  ReportType `json:"report_type"`
	Status      string     `json:"status"` // pending, running, completed, failed
	Progress    int        `json:"progress"`
	Result      *CalculationResult `json:"result,omitempty"`
	Error       string     `json:"error,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// DashboardSummary provides aggregated quality dashboard data
type DashboardSummary struct {
	TotalMeasures      int                `json:"total_measures"`
	ActiveMeasures     int                `json:"active_measures"`
	AverageScore       float64            `json:"average_score"`
	MeasuresByProgram  map[string]int     `json:"measures_by_program"`
	MeasuresByDomain   map[string]int     `json:"measures_by_domain"`
	OpenCareGaps       int                `json:"open_care_gaps"`
	PerformanceTrend   []TrendDataPoint   `json:"performance_trend"`
	TopPerformers      []MeasurePerformance `json:"top_performers"`
	BottomPerformers   []MeasurePerformance `json:"bottom_performers"`
	LastUpdated        time.Time          `json:"last_updated"`
}

// TrendDataPoint represents a point in time for trend analysis
type TrendDataPoint struct {
	Date  time.Time `json:"date"`
	Score float64   `json:"score"`
	Count int       `json:"count"`
}

// MeasurePerformance represents a measure's performance summary
type MeasurePerformance struct {
	MeasureID string  `json:"measure_id"`
	Name      string  `json:"name"`
	Score     float64 `json:"score"`
	Trend     string  `json:"trend"` // "up", "down", "stable"
	Delta     float64 `json:"delta"` // Change from previous period
}

// MeasureDefinitionFile represents the YAML file structure for measures
type MeasureDefinitionFile struct {
	Type    string  `yaml:"type"` // "measure"
	Measure Measure `yaml:"measure"`
}
