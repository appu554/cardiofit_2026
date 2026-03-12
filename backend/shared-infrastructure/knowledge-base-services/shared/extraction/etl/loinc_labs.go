// Package etl provides Extract-Transform-Load pipelines for clinical data sources.
// This file implements the LOINC Lab Ranges ETL for laboratory reference values.
//
// DESIGN PRINCIPLE: Lab ranges are standardized clinical data
// LOINC provides structured lab test codes and reference ranges.
// NHANES provides population-based statistical reference values.
// This is pure ETL from authoritative sources - NO LLM involved.
//
// DATA SOURCES:
// - LOINC: https://loinc.org (Regenstrief Institute)
// - NHANES: https://www.cdc.gov/nchs/nhanes/index.htm (CDC)
// - CLSI: Clinical and Laboratory Standards Institute guidelines
package etl

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cardiofit/shared/evidence"
)

// =============================================================================
// LOINC LAB DATA STRUCTURES
// =============================================================================

// LabCategory categorizes laboratory test types
type LabCategory string

const (
	LabCategoryChemistry    LabCategory = "CHEMISTRY"
	LabCategoryHematology   LabCategory = "HEMATOLOGY"
	LabCategoryUrinalysis   LabCategory = "URINALYSIS"
	LabCategoryCoagulation  LabCategory = "COAGULATION"
	LabCategoryEndocrine    LabCategory = "ENDOCRINE"
	LabCategoryToxicology   LabCategory = "TOXICOLOGY"
	LabCategoryMicrobiology LabCategory = "MICROBIOLOGY"
	LabCategoryImmunology   LabCategory = "IMMUNOLOGY"
	LabCategoryCardiac      LabCategory = "CARDIAC"
	LabCategoryRenal        LabCategory = "RENAL"
	LabCategoryHepatic      LabCategory = "HEPATIC"
	LabCategoryOther        LabCategory = "OTHER"
)

// PopulationGroup categorizes reference range populations
type PopulationGroup string

const (
	PopulationAdult         PopulationGroup = "ADULT"
	PopulationPediatric     PopulationGroup = "PEDIATRIC"
	PopulationGeriatric     PopulationGroup = "GERIATRIC"
	PopulationMale          PopulationGroup = "MALE"
	PopulationFemale        PopulationGroup = "FEMALE"
	PopulationPregnant      PopulationGroup = "PREGNANT"
	PopulationNeonate       PopulationGroup = "NEONATE"
	PopulationInfant        PopulationGroup = "INFANT"
	PopulationChild         PopulationGroup = "CHILD"
	PopulationAdolescent    PopulationGroup = "ADOLESCENT"
	PopulationGeneral       PopulationGroup = "GENERAL"
)

// CriticalLevel indicates clinical criticality
type CriticalLevel string

const (
	CriticalLevelPanic    CriticalLevel = "PANIC"    // Immediate action required
	CriticalLevelCritical CriticalLevel = "CRITICAL" // Urgent attention needed
	CriticalLevelAbnormal CriticalLevel = "ABNORMAL" // Outside normal range
	CriticalLevelNormal   CriticalLevel = "NORMAL"   // Within reference range
)

// LabReferenceRange represents a single lab test reference range
type LabReferenceRange struct {
	// LOINC identification
	LOINCCode      string `json:"loincCode"`
	LOINCShortName string `json:"loincShortName"`
	LOINCLongName  string `json:"loincLongName"`

	// Test information
	TestName        string      `json:"testName"`
	TestDescription string      `json:"testDescription,omitempty"`
	Category        LabCategory `json:"category"`
	System          string      `json:"system,omitempty"`         // Blood, Urine, CSF, etc.
	Property        string      `json:"property,omitempty"`       // MCnc, SCnc, etc.
	TimeAspect      string      `json:"timeAspect,omitempty"`     // Pt, 24H, etc.
	Scale           string      `json:"scale,omitempty"`          // Qn, Ord, Nom, etc.
	Method          string      `json:"method,omitempty"`         // Testing method

	// Reference range values
	RefRangeLow    *float64 `json:"refRangeLow,omitempty"`
	RefRangeHigh   *float64 `json:"refRangeHigh,omitempty"`
	Unit           string   `json:"unit"`
	UnitUCUM       string   `json:"unitUcum,omitempty"` // UCUM standardized unit

	// Critical values (panic values)
	CriticalLow    *float64 `json:"criticalLow,omitempty"`
	CriticalHigh   *float64 `json:"criticalHigh,omitempty"`

	// Population specificity
	Population      PopulationGroup `json:"population"`
	AgeRangeLowYrs  *float64        `json:"ageRangeLowYrs,omitempty"`
	AgeRangeHighYrs *float64        `json:"ageRangeHighYrs,omitempty"`
	Sex             string          `json:"sex,omitempty"` // M, F, or empty for both

	// Statistical basis (for NHANES-derived ranges)
	Percentile2_5  *float64 `json:"percentile2_5,omitempty"`
	Percentile97_5 *float64 `json:"percentile97_5,omitempty"`
	Mean           *float64 `json:"mean,omitempty"`
	StdDev         *float64 `json:"stdDev,omitempty"`
	SampleSize     int      `json:"sampleSize,omitempty"`

	// Source information
	Source         string    `json:"source"`          // LOINC, NHANES, CLSI, etc.
	SourceVersion  string    `json:"sourceVersion"`
	EffectiveDate  time.Time `json:"effectiveDate"`
	RowNumber      int       `json:"rowNumber"`
}

// LabMonitoringRequirement represents drug-lab monitoring requirements
type LabMonitoringRequirement struct {
	// Drug identification
	RxCUI     string `json:"rxcui"`
	DrugName  string `json:"drugName"`

	// Lab test
	LOINCCode string `json:"loincCode"`
	TestName  string `json:"testName"`

	// Monitoring parameters
	Frequency         string `json:"frequency"`         // Weekly, Monthly, etc.
	BaselineRequired  bool   `json:"baselineRequired"`
	DurationDays      int    `json:"durationDays,omitempty"`
	Indication        string `json:"indication,omitempty"`

	// Alert thresholds (drug-specific)
	AlertLow          *float64 `json:"alertLow,omitempty"`
	AlertHigh         *float64 `json:"alertHigh,omitempty"`
	ActionRequired    string   `json:"actionRequired,omitempty"` // Hold drug, notify MD, etc.

	// Delta check for trending alerts (Review Refinement)
	DeltaCheck *DeltaCheck `json:"deltaCheck,omitempty"`

	Source string `json:"source"`
}

// =============================================================================
// DELTA CHECK FOR TRENDING ALERTS (Review Refinement)
// =============================================================================
// Critical for detecting rapid changes in lab values that may indicate
// acute clinical deterioration (e.g., AKI detection via creatinine rise).
// Example: If creatinine rises >50% in 48h, flag for AKI evaluation

// DeltaCheck defines thresholds for detecting significant changes in lab values over time
type DeltaCheck struct {
	// MaxChangePercent is the maximum allowed percentage change within the time window
	// Example: 50.0 means a 50% change triggers an alert
	MaxChangePercent *float64 `json:"maxChangePercent,omitempty"`

	// MaxChangeAbsolute is the maximum allowed absolute change within the time window
	// Example: 2.0 mg/dL for creatinine
	MaxChangeAbsolute *float64 `json:"maxChangeAbsolute,omitempty"`

	// TimeWindowHours is the comparison window in hours
	// Example: 48 means compare current value to value 48 hours ago
	TimeWindowHours int `json:"timeWindowHours"`

	// Direction specifies which direction of change triggers an alert
	// "INCREASE", "DECREASE", or "BOTH"
	Direction string `json:"direction"`

	// ClinicalImplication describes what a triggered delta check means
	// Example: "Possible acute kidney injury - evaluate for causes"
	ClinicalImplication string `json:"clinicalImplication,omitempty"`

	// RecommendedAction is the clinical action when delta check triggers
	RecommendedAction string `json:"recommendedAction,omitempty"`
}

// DeltaDirection constants for clarity
const (
	DeltaDirectionIncrease = "INCREASE"
	DeltaDirectionDecrease = "DECREASE"
	DeltaDirectionBoth     = "BOTH"
)

// DefaultDeltaChecks returns clinically-validated delta check thresholds
// for commonly monitored lab values
func DefaultDeltaChecks() map[string]*DeltaCheck {
	return map[string]*DeltaCheck{
		// Creatinine - AKI detection (KDIGO criteria)
		"2160-0": {
			MaxChangePercent:    ptr(50.0),  // 50% increase in 7 days
			MaxChangeAbsolute:   ptr(0.3),   // 0.3 mg/dL increase in 48h
			TimeWindowHours:     48,
			Direction:           DeltaDirectionIncrease,
			ClinicalImplication: "Possible acute kidney injury (AKI) - meets KDIGO Stage 1 criteria",
			RecommendedAction:   "Assess volume status, review nephrotoxic medications, consider renal consult",
		},
		// Potassium - rapid rise may indicate acute renal failure or medication effect
		"2823-3": {
			MaxChangePercent:    nil,
			MaxChangeAbsolute:   ptr(1.0), // 1.0 mEq/L
			TimeWindowHours:     24,
			Direction:           DeltaDirectionIncrease,
			ClinicalImplication: "Rapid potassium rise - risk of cardiac arrhythmia",
			RecommendedAction:   "Check ECG, hold K-sparing medications, consider urgent treatment if K>6.0",
		},
		// Hemoglobin - rapid drop may indicate active bleeding
		"718-7": {
			MaxChangePercent:    nil,
			MaxChangeAbsolute:   ptr(2.0), // 2.0 g/dL
			TimeWindowHours:     24,
			Direction:           DeltaDirectionDecrease,
			ClinicalImplication: "Rapid hemoglobin drop - possible active bleeding",
			RecommendedAction:   "Assess for bleeding source, consider transfusion, hold anticoagulants",
		},
		// Platelets - rapid drop may indicate HIT or TTP
		"777-3": {
			MaxChangePercent:    ptr(50.0),
			MaxChangeAbsolute:   nil,
			TimeWindowHours:     72,
			Direction:           DeltaDirectionDecrease,
			ClinicalImplication: "Rapid platelet drop - evaluate for HIT, TTP, or other causes",
			RecommendedAction:   "Calculate 4T score if on heparin, consider HIT antibody testing",
		},
		// INR - rapid rise indicates over-anticoagulation
		"5895-7": {
			MaxChangePercent:    nil,
			MaxChangeAbsolute:   ptr(1.5), // 1.5 unit increase
			TimeWindowHours:     48,
			Direction:           DeltaDirectionIncrease,
			ClinicalImplication: "Rapid INR rise - bleeding risk increased",
			RecommendedAction:   "Hold warfarin, consider vitamin K if INR>9, assess for drug interactions",
		},
		// Glucose - rapid drop may indicate hypoglycemia
		"2345-7": {
			MaxChangePercent:    nil,
			MaxChangeAbsolute:   ptr(100.0), // 100 mg/dL drop
			TimeWindowHours:     4,
			Direction:           DeltaDirectionDecrease,
			ClinicalImplication: "Rapid glucose drop - hypoglycemia risk",
			RecommendedAction:   "Assess for symptoms, review insulin/sulfonylurea doses, provide glucose if symptomatic",
		},
		// Sodium - rapid change may cause neurological symptoms
		"2951-2": {
			MaxChangePercent:    nil,
			MaxChangeAbsolute:   ptr(8.0), // 8 mEq/L per 24h
			TimeWindowHours:     24,
			Direction:           DeltaDirectionBoth,
			ClinicalImplication: "Rapid sodium change - risk of osmotic demyelination or cerebral edema",
			RecommendedAction:   "Slow correction rate to <10 mEq/L per 24h, monitor neurological status",
		},
	}
}

// ptr is a helper to create float64 pointers
func ptr(v float64) *float64 {
	return &v
}

// EvaluateDeltaCheck determines if a lab value change triggers a delta check alert
func EvaluateDeltaCheck(currentValue, previousValue float64, check *DeltaCheck) (triggered bool, changePercent, changeAbsolute float64) {
	if check == nil {
		return false, 0, 0
	}

	// Calculate changes
	changeAbsolute = currentValue - previousValue
	if previousValue != 0 {
		changePercent = (changeAbsolute / previousValue) * 100
	}

	// Check direction
	switch check.Direction {
	case DeltaDirectionIncrease:
		if changeAbsolute <= 0 {
			return false, changePercent, changeAbsolute
		}
	case DeltaDirectionDecrease:
		if changeAbsolute >= 0 {
			return false, changePercent, changeAbsolute
		}
		// Use absolute values for comparison
		changeAbsolute = -changeAbsolute
		changePercent = -changePercent
	case DeltaDirectionBoth:
		// Use absolute values
		if changeAbsolute < 0 {
			changeAbsolute = -changeAbsolute
			changePercent = -changePercent
		}
	}

	// Check thresholds
	if check.MaxChangePercent != nil && changePercent >= *check.MaxChangePercent {
		return true, changePercent, changeAbsolute
	}
	if check.MaxChangeAbsolute != nil && changeAbsolute >= *check.MaxChangeAbsolute {
		return true, changePercent, changeAbsolute
	}

	return false, changePercent, changeAbsolute
}

// =============================================================================
// LOINC LAB ETL LOADER
// =============================================================================

// LOINCLabLoaderConfig configures the LOINC lab ranges loader
type LOINCLabLoaderConfig struct {
	// LOINCFilePath is the path to the LOINC table CSV file
	LOINCFilePath string

	// ReferenceRangeFilePath is the path to reference ranges file
	ReferenceRangeFilePath string

	// NHANESFilePath is optional path to NHANES statistics file
	NHANESFilePath string

	// MonitoringFilePath is optional path to drug-lab monitoring requirements
	MonitoringFilePath string

	// FilterByCategory optionally filters to specific categories
	FilterByCategory []LabCategory

	// FilterByLOINC optionally filters to specific LOINC codes
	FilterByLOINC []string

	// IncludeDeprecated includes deprecated LOINC codes
	IncludeDeprecated bool
}

// LOINCLabLoader loads laboratory reference ranges from LOINC and NHANES
type LOINCLabLoader struct {
	config LOINCLabLoaderConfig

	// LOINC code index for fast lookup
	loincIndex map[string]*LabReferenceRange

	// Category filter set
	categoryFilter map[LabCategory]bool

	// LOINC filter set
	loincFilter map[string]bool
}

// NewLOINCLabLoader creates a new LOINC lab ranges loader
func NewLOINCLabLoader(config LOINCLabLoaderConfig) *LOINCLabLoader {
	loader := &LOINCLabLoader{
		config:         config,
		loincIndex:     make(map[string]*LabReferenceRange),
		categoryFilter: make(map[LabCategory]bool),
		loincFilter:    make(map[string]bool),
	}

	// Build filter sets
	for _, cat := range config.FilterByCategory {
		loader.categoryFilter[cat] = true
	}
	for _, loinc := range config.FilterByLOINC {
		loader.loincFilter[loinc] = true
	}

	return loader
}

// LOINCLabLoadResult contains the results of loading LOINC lab data
type LOINCLabLoadResult struct {
	// Reference ranges loaded
	ReferenceRanges []*LabReferenceRange

	// Monitoring requirements loaded
	MonitoringRequirements []*LabMonitoringRequirement

	// Statistics
	TotalRowsProcessed int
	RangesLoaded       int
	MonitoringLoaded   int
	UniqueLOINCCodes   int
	NHANESEnriched     int

	// Category distribution
	CategoryDistribution map[LabCategory]int

	// Population distribution
	PopulationDistribution map[PopulationGroup]int

	// Processing metadata
	LoadedAt     time.Time
	LoadDuration time.Duration

	// Errors encountered (non-fatal)
	Warnings []string
}

// Load performs the full ETL from LOINC and NHANES files
func (l *LOINCLabLoader) Load(ctx context.Context) (*LOINCLabLoadResult, error) {
	startTime := time.Now()
	result := &LOINCLabLoadResult{
		ReferenceRanges:        make([]*LabReferenceRange, 0),
		MonitoringRequirements: make([]*LabMonitoringRequirement, 0),
		CategoryDistribution:   make(map[LabCategory]int),
		PopulationDistribution: make(map[PopulationGroup]int),
		Warnings:               make([]string, 0),
		LoadedAt:               startTime,
	}

	// Step 1: Load LOINC codes (base information)
	if l.config.LOINCFilePath != "" {
		if err := l.loadLOINCCodes(ctx, result); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to load LOINC codes: %v", err))
		}
	}

	// Step 2: Load reference ranges
	if l.config.ReferenceRangeFilePath != "" {
		if err := l.loadReferenceRanges(ctx, result); err != nil {
			return nil, fmt.Errorf("failed to load reference ranges: %w", err)
		}
	}

	// Step 3: Enrich with NHANES statistics if available
	if l.config.NHANESFilePath != "" {
		if err := l.enrichWithNHANES(ctx, result); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to enrich with NHANES: %v", err))
		}
	}

	// Step 4: Load monitoring requirements if available
	if l.config.MonitoringFilePath != "" {
		if err := l.loadMonitoringRequirements(ctx, result); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to load monitoring requirements: %v", err))
		}
	}

	// Step 5: Calculate statistics
	l.calculateStatistics(result)

	result.LoadDuration = time.Since(startTime)
	return result, nil
}

// loadLOINCCodes loads base LOINC code information
func (l *LOINCLabLoader) loadLOINCCodes(ctx context.Context, result *LOINCLabLoadResult) error {
	file, err := os.Open(l.config.LOINCFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := csv.NewReader(bufio.NewReader(file))
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	// Read header
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	colIndex := make(map[string]int)
	for i, col := range header {
		colIndex[strings.ToLower(strings.TrimSpace(col))] = i
	}

	// Read records
	rowNum := 1
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			rowNum++
			continue
		}

		result.TotalRowsProcessed++

		labRange := l.parseLOINCRecord(record, colIndex, rowNum)
		if labRange == nil {
			rowNum++
			continue
		}

		// Apply filters
		if !l.passesFilters(labRange) {
			rowNum++
			continue
		}

		// Index for later enrichment
		l.loincIndex[labRange.LOINCCode] = labRange
		rowNum++
	}

	return nil
}

// parseLOINCRecord parses a LOINC table record
func (l *LOINCLabLoader) parseLOINCRecord(record []string, colIndex map[string]int, rowNum int) *LabReferenceRange {
	getValue := func(cols ...string) string {
		for _, col := range cols {
			if idx, ok := colIndex[col]; ok && idx < len(record) {
				val := strings.TrimSpace(record[idx])
				if val != "" {
					return val
				}
			}
		}
		return ""
	}

	loincCode := getValue("loinc_num", "loinc", "loinc_code")
	if loincCode == "" {
		return nil
	}

	// Check status (skip deprecated unless configured)
	status := strings.ToUpper(getValue("status"))
	if !l.config.IncludeDeprecated && (status == "DEPRECATED" || status == "DISCOURAGED") {
		return nil
	}

	labRange := &LabReferenceRange{
		LOINCCode:      loincCode,
		LOINCShortName: getValue("shortname", "short_name"),
		LOINCLongName:  getValue("long_common_name", "longname"),
		TestName:       getValue("component", "test_name"),
		System:         getValue("system"),
		Property:       getValue("property"),
		TimeAspect:     getValue("time_aspct", "time_aspect"),
		Scale:          getValue("scale_typ", "scale"),
		Method:         getValue("method_typ", "method"),
		Unit:           getValue("example_units", "units"),
		UnitUCUM:       getValue("example_ucum_units", "ucum_units"),
		Source:         "LOINC",
		SourceVersion:  getValue("version_last_changed", "version"),
		Population:     PopulationGeneral,
		RowNumber:      rowNum,
	}

	// Determine category from class
	classType := strings.ToUpper(getValue("class", "classtype"))
	labRange.Category = l.classifyCategory(classType, labRange.TestName)

	return labRange
}

// classifyCategory determines lab category from LOINC class
func (l *LOINCLabLoader) classifyCategory(class, testName string) LabCategory {
	classLower := strings.ToLower(class)
	testLower := strings.ToLower(testName)

	switch {
	case strings.Contains(classLower, "chem"):
		return LabCategoryChemistry
	case strings.Contains(classLower, "hem") || strings.Contains(classLower, "blood"):
		return LabCategoryHematology
	case strings.Contains(classLower, "ua") || strings.Contains(classLower, "urin"):
		return LabCategoryUrinalysis
	case strings.Contains(classLower, "coag"):
		return LabCategoryCoagulation
	case strings.Contains(testLower, "creatinine") || strings.Contains(testLower, "egfr") || strings.Contains(testLower, "bun"):
		return LabCategoryRenal
	case strings.Contains(testLower, "alt") || strings.Contains(testLower, "ast") || strings.Contains(testLower, "bilirubin"):
		return LabCategoryHepatic
	case strings.Contains(testLower, "troponin") || strings.Contains(testLower, "bnp"):
		return LabCategoryCardiac
	case strings.Contains(classLower, "tox") || strings.Contains(classLower, "drug"):
		return LabCategoryToxicology
	default:
		return LabCategoryOther
	}
}

// loadReferenceRanges loads actual reference range values
func (l *LOINCLabLoader) loadReferenceRanges(ctx context.Context, result *LOINCLabLoadResult) error {
	file, err := os.Open(l.config.ReferenceRangeFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := csv.NewReader(bufio.NewReader(file))
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	// Read header
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	colIndex := make(map[string]int)
	for i, col := range header {
		colIndex[strings.ToLower(strings.TrimSpace(col))] = i
	}

	// Read records
	rowNum := 1
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("row %d: %v", rowNum, err))
			rowNum++
			continue
		}

		labRange := l.parseReferenceRangeRecord(record, colIndex, rowNum)
		if labRange == nil {
			rowNum++
			continue
		}

		// Merge with LOINC base info if available
		if base, ok := l.loincIndex[labRange.LOINCCode]; ok {
			labRange.LOINCShortName = base.LOINCShortName
			labRange.LOINCLongName = base.LOINCLongName
			if labRange.TestName == "" {
				labRange.TestName = base.TestName
			}
			if labRange.Category == "" {
				labRange.Category = base.Category
			}
			if labRange.System == "" {
				labRange.System = base.System
			}
		}

		// Apply filters
		if !l.passesFilters(labRange) {
			rowNum++
			continue
		}

		result.ReferenceRanges = append(result.ReferenceRanges, labRange)
		result.RangesLoaded++
		rowNum++
	}

	return nil
}

// parseReferenceRangeRecord parses a reference range record
func (l *LOINCLabLoader) parseReferenceRangeRecord(record []string, colIndex map[string]int, rowNum int) *LabReferenceRange {
	getValue := func(cols ...string) string {
		for _, col := range cols {
			if idx, ok := colIndex[col]; ok && idx < len(record) {
				val := strings.TrimSpace(record[idx])
				if val != "" {
					return val
				}
			}
		}
		return ""
	}

	getFloatPtr := func(cols ...string) *float64 {
		val := getValue(cols...)
		if val == "" {
			return nil
		}
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil
		}
		return &f
	}

	loincCode := getValue("loinc", "loinc_code", "loinc_num")
	if loincCode == "" {
		return nil
	}

	labRange := &LabReferenceRange{
		LOINCCode:       loincCode,
		TestName:        getValue("test_name", "component", "name"),
		RefRangeLow:     getFloatPtr("ref_low", "reference_low", "normal_low"),
		RefRangeHigh:    getFloatPtr("ref_high", "reference_high", "normal_high"),
		Unit:            getValue("unit", "units"),
		CriticalLow:     getFloatPtr("critical_low", "panic_low"),
		CriticalHigh:    getFloatPtr("critical_high", "panic_high"),
		AgeRangeLowYrs:  getFloatPtr("age_low", "age_min"),
		AgeRangeHighYrs: getFloatPtr("age_high", "age_max"),
		Sex:             getValue("sex", "gender"),
		Source:          getValue("source"),
		RowNumber:       rowNum,
	}

	if labRange.Source == "" {
		labRange.Source = "REFERENCE_RANGE"
	}

	// Determine population
	labRange.Population = l.determinePopulation(labRange)

	// Determine category if test name available
	if labRange.TestName != "" {
		labRange.Category = l.classifyCategory("", labRange.TestName)
	}

	// Parse effective date
	dateStr := getValue("effective_date", "date")
	if dateStr != "" {
		formats := []string{"2006-01-02", "01/02/2006", "1/2/2006"}
		for _, format := range formats {
			if t, err := time.Parse(format, dateStr); err == nil {
				labRange.EffectiveDate = t
				break
			}
		}
	}
	if labRange.EffectiveDate.IsZero() {
		labRange.EffectiveDate = time.Now()
	}

	return labRange
}

// determinePopulation determines population group from age/sex
func (l *LOINCLabLoader) determinePopulation(labRange *LabReferenceRange) PopulationGroup {
	// Check sex first
	if labRange.Sex == "M" {
		return PopulationMale
	}
	if labRange.Sex == "F" {
		return PopulationFemale
	}

	// Check age range
	if labRange.AgeRangeLowYrs != nil && labRange.AgeRangeHighYrs != nil {
		low := *labRange.AgeRangeLowYrs
		high := *labRange.AgeRangeHighYrs

		switch {
		case high <= 0.083: // < 1 month
			return PopulationNeonate
		case high <= 1:
			return PopulationInfant
		case high <= 12:
			return PopulationChild
		case high <= 18:
			return PopulationAdolescent
		case low >= 65:
			return PopulationGeriatric
		case low >= 18 && high <= 65:
			return PopulationAdult
		}
	}

	return PopulationGeneral
}

// enrichWithNHANES adds NHANES statistical data to reference ranges
func (l *LOINCLabLoader) enrichWithNHANES(ctx context.Context, result *LOINCLabLoadResult) error {
	file, err := os.Open(l.config.NHANESFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := csv.NewReader(bufio.NewReader(file))
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	// Read header
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	colIndex := make(map[string]int)
	for i, col := range header {
		colIndex[strings.ToLower(strings.TrimSpace(col))] = i
	}

	// Read records and match to existing ranges
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		loincCode := l.getRecordValue(record, colIndex, "loinc", "loinc_code")
		if loincCode == "" {
			continue
		}

		// Find matching reference ranges
		for _, labRange := range result.ReferenceRanges {
			if labRange.LOINCCode == loincCode {
				// Enrich with NHANES statistics
				labRange.Percentile2_5 = l.getRecordFloatPtr(record, colIndex, "p2_5", "percentile_2_5")
				labRange.Percentile97_5 = l.getRecordFloatPtr(record, colIndex, "p97_5", "percentile_97_5")
				labRange.Mean = l.getRecordFloatPtr(record, colIndex, "mean", "average")
				labRange.StdDev = l.getRecordFloatPtr(record, colIndex, "std_dev", "sd")

				sampleSize := l.getRecordValue(record, colIndex, "sample_size", "n")
				if sampleSize != "" {
					if n, err := strconv.Atoi(sampleSize); err == nil {
						labRange.SampleSize = n
					}
				}

				result.NHANESEnriched++
			}
		}
	}

	return nil
}

// loadMonitoringRequirements loads drug-lab monitoring requirements
func (l *LOINCLabLoader) loadMonitoringRequirements(ctx context.Context, result *LOINCLabLoadResult) error {
	file, err := os.Open(l.config.MonitoringFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := csv.NewReader(bufio.NewReader(file))
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	// Read header
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	colIndex := make(map[string]int)
	for i, col := range header {
		colIndex[strings.ToLower(strings.TrimSpace(col))] = i
	}

	// Read records
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		req := &LabMonitoringRequirement{
			RxCUI:            l.getRecordValue(record, colIndex, "rxcui", "rxnorm_id"),
			DrugName:         l.getRecordValue(record, colIndex, "drug_name", "drug"),
			LOINCCode:        l.getRecordValue(record, colIndex, "loinc", "loinc_code"),
			TestName:         l.getRecordValue(record, colIndex, "test_name", "lab_test"),
			Frequency:        l.getRecordValue(record, colIndex, "frequency", "monitoring_frequency"),
			BaselineRequired: strings.ToUpper(l.getRecordValue(record, colIndex, "baseline", "baseline_required")) == "Y",
			Indication:       l.getRecordValue(record, colIndex, "indication"),
			AlertLow:         l.getRecordFloatPtr(record, colIndex, "alert_low", "threshold_low"),
			AlertHigh:        l.getRecordFloatPtr(record, colIndex, "alert_high", "threshold_high"),
			ActionRequired:   l.getRecordValue(record, colIndex, "action", "action_required"),
			Source:           l.getRecordValue(record, colIndex, "source"),
		}

		durationStr := l.getRecordValue(record, colIndex, "duration_days", "duration")
		if durationStr != "" {
			if d, err := strconv.Atoi(durationStr); err == nil {
				req.DurationDays = d
			}
		}

		if req.RxCUI != "" && req.LOINCCode != "" {
			result.MonitoringRequirements = append(result.MonitoringRequirements, req)
			result.MonitoringLoaded++
		}
	}

	return nil
}

// Helper methods for record parsing
func (l *LOINCLabLoader) getRecordValue(record []string, colIndex map[string]int, cols ...string) string {
	for _, col := range cols {
		if idx, ok := colIndex[col]; ok && idx < len(record) {
			val := strings.TrimSpace(record[idx])
			if val != "" {
				return val
			}
		}
	}
	return ""
}

func (l *LOINCLabLoader) getRecordFloatPtr(record []string, colIndex map[string]int, cols ...string) *float64 {
	val := l.getRecordValue(record, colIndex, cols...)
	if val == "" {
		return nil
	}
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return nil
	}
	return &f
}

// passesFilters checks if a lab range passes configured filters
func (l *LOINCLabLoader) passesFilters(labRange *LabReferenceRange) bool {
	// Category filter
	if len(l.categoryFilter) > 0 {
		if !l.categoryFilter[labRange.Category] {
			return false
		}
	}

	// LOINC filter
	if len(l.loincFilter) > 0 {
		if !l.loincFilter[labRange.LOINCCode] {
			return false
		}
	}

	return true
}

// calculateStatistics computes summary statistics
func (l *LOINCLabLoader) calculateStatistics(result *LOINCLabLoadResult) {
	uniqueLOINCs := make(map[string]bool)

	for _, labRange := range result.ReferenceRanges {
		uniqueLOINCs[labRange.LOINCCode] = true
		result.CategoryDistribution[labRange.Category]++
		result.PopulationDistribution[labRange.Population]++
	}

	result.UniqueLOINCCodes = len(uniqueLOINCs)
}

// =============================================================================
// EVIDENCE UNIT CONVERSION
// =============================================================================

// ToEvidenceUnits converts lab reference ranges to EvidenceUnits for the Evidence Router
func (l *LOINCLabLoader) ToEvidenceUnits(ranges []*LabReferenceRange) []*evidence.EvidenceUnit {
	units := make([]*evidence.EvidenceUnit, 0, len(ranges))

	for _, labRange := range ranges {
		unit := evidence.NewEvidenceUnit(evidence.SourceTypeCSV, "https://loinc.org")
		unit.EvidenceID = fmt.Sprintf("LOINC-LAB-%s-%s",
			labRange.LOINCCode,
			labRange.Population)

		// No RxCUI for lab tests - use LOINC code as identifier
		unit.DrugName = labRange.TestName

		// Set clinical domains
		unit.AddClinicalDomain(evidence.DomainLab)

		// Target KB-16 (Lab Reference)
		unit.AddKBTarget("KB-16")

		// Authoritative source = high priority
		unit.Priority = 2

		// Store range data in parsed content
		unit.ParsedContent = labRange
		unit.ContentType = "application/json"

		// Set provenance
		unit.SourceVersion = labRange.SourceVersion
		unit.Jurisdiction = "US"
		unit.RegulatoryBody = "LOINC"

		// Set quality signals - LOINC is authoritative
		unit.ConfidenceFloor = 0.95
		if labRange.SampleSize > 1000 {
			unit.QualityScore = 0.98 // NHANES-validated
		} else {
			unit.QualityScore = 0.90
		}

		// Add metadata
		unit.SourceMetadata = map[string]string{
			"loinc_code":   labRange.LOINCCode,
			"test_name":    labRange.TestName,
			"category":     string(labRange.Category),
			"population":   string(labRange.Population),
			"unit":         labRange.Unit,
			"source":       labRange.Source,
		}

		// Add reference range values if available
		if labRange.RefRangeLow != nil {
			unit.SourceMetadata["ref_range_low"] = fmt.Sprintf("%.2f", *labRange.RefRangeLow)
		}
		if labRange.RefRangeHigh != nil {
			unit.SourceMetadata["ref_range_high"] = fmt.Sprintf("%.2f", *labRange.RefRangeHigh)
		}

		units = append(units, unit)
	}

	return units
}

// =============================================================================
// LOOKUP METHODS
// =============================================================================

// GetReferenceRange returns reference range for a LOINC code and population
func (l *LOINCLabLoader) GetReferenceRange(loincCode string, population PopulationGroup, ranges []*LabReferenceRange) *LabReferenceRange {
	for _, labRange := range ranges {
		if labRange.LOINCCode == loincCode && labRange.Population == population {
			return labRange
		}
	}
	// Fall back to general population
	for _, labRange := range ranges {
		if labRange.LOINCCode == loincCode && labRange.Population == PopulationGeneral {
			return labRange
		}
	}
	return nil
}

// GetMonitoringForDrug returns monitoring requirements for a drug
func (l *LOINCLabLoader) GetMonitoringForDrug(rxcui string, reqs []*LabMonitoringRequirement) []*LabMonitoringRequirement {
	result := make([]*LabMonitoringRequirement, 0)
	for _, req := range reqs {
		if req.RxCUI == rxcui {
			result = append(result, req)
		}
	}
	return result
}

// GetRangesByCategory returns ranges in a specific category
func (l *LOINCLabLoader) GetRangesByCategory(category LabCategory, ranges []*LabReferenceRange) []*LabReferenceRange {
	result := make([]*LabReferenceRange, 0)
	for _, labRange := range ranges {
		if labRange.Category == category {
			result = append(result, labRange)
		}
	}
	return result
}

// EvaluateLabValue determines if a lab value is normal, abnormal, or critical
func (l *LOINCLabLoader) EvaluateLabValue(value float64, labRange *LabReferenceRange) CriticalLevel {
	// Check critical values first
	if labRange.CriticalLow != nil && value < *labRange.CriticalLow {
		return CriticalLevelPanic
	}
	if labRange.CriticalHigh != nil && value > *labRange.CriticalHigh {
		return CriticalLevelPanic
	}

	// Check reference range
	if labRange.RefRangeLow != nil && value < *labRange.RefRangeLow {
		return CriticalLevelAbnormal
	}
	if labRange.RefRangeHigh != nil && value > *labRange.RefRangeHigh {
		return CriticalLevelAbnormal
	}

	return CriticalLevelNormal
}

// =============================================================================
// STATISTICS AND REPORTING
// =============================================================================

// GetStatistics returns statistics about loaded lab data
func (l *LOINCLabLoader) GetStatistics(result *LOINCLabLoadResult) map[string]interface{} {
	return map[string]interface{}{
		"total_ranges":             result.RangesLoaded,
		"unique_loinc_codes":       result.UniqueLOINCCodes,
		"nhanes_enriched":          result.NHANESEnriched,
		"monitoring_requirements":  result.MonitoringLoaded,
		"category_distribution":    result.CategoryDistribution,
		"population_distribution":  result.PopulationDistribution,
		"load_duration_ms":         result.LoadDuration.Milliseconds(),
	}
}
