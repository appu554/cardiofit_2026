// Package reference provides the reference database for lab test definitions
package reference

import (
	"kb-16-lab-interpretation/pkg/types"
)

// TestDefinition defines a laboratory test with its properties
type TestDefinition struct {
	Code            string            `json:"code"`             // LOINC code
	Name            string            `json:"name"`             // Test name
	ShortName       string            `json:"short_name"`       // Abbreviated name
	Category        string            `json:"category"`         // Chemistry, Hematology, etc.
	Unit            string            `json:"unit"`             // Default unit
	DecimalPlaces   int               `json:"decimal_places"`   // Precision
	DefaultRange    *types.ReferenceRange `json:"default_range"`    // Default reference range
	AgeRanges       []AgeRange        `json:"age_ranges,omitempty"`
	SexRanges       []SexRange        `json:"sex_ranges,omitempty"`
	CriticalValues  *CriticalRange    `json:"critical_values,omitempty"`
	PanicValues     *CriticalRange    `json:"panic_values,omitempty"`
	DeltaThreshold  *DeltaRule        `json:"delta_threshold,omitempty"`
	PanelMembership []string          `json:"panel_membership,omitempty"`
	TrendingEnabled bool              `json:"trending_enabled"`
}

// AgeRange defines age-specific reference ranges
type AgeRange struct {
	MinAge int                   `json:"min_age"`
	MaxAge int                   `json:"max_age"`
	Range  *types.ReferenceRange `json:"range"`
}

// SexRange defines sex-specific reference ranges
type SexRange struct {
	Sex   string                `json:"sex"` // male, female
	Range *types.ReferenceRange `json:"range"`
}

// CriticalRange defines critical/panic value thresholds
type CriticalRange struct {
	Code  string   `json:"code"`
	Name  string   `json:"name"`
	Low   *float64 `json:"low,omitempty"`
	High  *float64 `json:"high,omitempty"`
}

// DeltaRule defines delta check thresholds
type DeltaRule struct {
	Code             string  `json:"code"`
	Name             string  `json:"name"`
	Threshold        float64 `json:"threshold,omitempty"`        // Absolute threshold
	ThresholdPercent float64 `json:"threshold_percent,omitempty"` // Percentage threshold
	Direction        string  `json:"direction"`                  // increase, decrease, any
	WindowHours      int     `json:"window_hours"`
}

// Database is the reference database containing all test definitions
type Database struct {
	tests        map[string]*TestDefinition
	byCategory   map[string][]*TestDefinition
	criticalVals map[string]*CriticalRange
	panicVals    map[string]*CriticalRange
	deltaRules   map[string]*DeltaRule
}

// NewDatabase creates and initializes the reference database
func NewDatabase() *Database {
	db := &Database{
		tests:        make(map[string]*TestDefinition),
		byCategory:   make(map[string][]*TestDefinition),
		criticalVals: make(map[string]*CriticalRange),
		panicVals:    make(map[string]*CriticalRange),
		deltaRules:   make(map[string]*DeltaRule),
	}

	db.initializeTests()
	db.initializeCriticalValues()
	db.initializeDeltaRules()

	return db
}

// GetTest returns a test definition by LOINC code
func (db *Database) GetTest(code string) *TestDefinition {
	return db.tests[code]
}

// GetRanges returns reference ranges for a test, adjusted for age/sex if available
func (db *Database) GetRanges(code string, age int, sex string) *types.ReferenceRange {
	test := db.tests[code]
	if test == nil {
		return nil
	}

	// Check age-specific ranges
	if age > 0 && len(test.AgeRanges) > 0 {
		for _, ar := range test.AgeRanges {
			if age >= ar.MinAge && age <= ar.MaxAge {
				return ar.Range
			}
		}
	}

	// Check sex-specific ranges
	if sex != "" && len(test.SexRanges) > 0 {
		for _, sr := range test.SexRanges {
			if sr.Sex == sex {
				return sr.Range
			}
		}
	}

	return test.DefaultRange
}

// GetCriticalValues returns critical value thresholds for a test
func (db *Database) GetCriticalValues(code string) *CriticalRange {
	return db.criticalVals[code]
}

// GetPanicValues returns panic value thresholds for a test
func (db *Database) GetPanicValues(code string) *CriticalRange {
	return db.panicVals[code]
}

// GetDeltaRule returns delta check rule for a test
func (db *Database) GetDeltaRule(code string) *DeltaRule {
	return db.deltaRules[code]
}

// ListTests returns all test definitions, optionally filtered by category
func (db *Database) ListTests(category string) []*TestDefinition {
	if category != "" {
		return db.byCategory[category]
	}

	tests := make([]*TestDefinition, 0, len(db.tests))
	for _, t := range db.tests {
		tests = append(tests, t)
	}
	return tests
}

// ListCategories returns all test categories
func (db *Database) ListCategories() []string {
	categories := make([]string, 0, len(db.byCategory))
	for cat := range db.byCategory {
		categories = append(categories, cat)
	}
	return categories
}

// ptr returns a pointer to a float64 value
func ptr(v float64) *float64 {
	return &v
}

// initializeTests populates the test database with 40+ lab tests
func (db *Database) initializeTests() {
	tests := []*TestDefinition{
		// ==========================================================================
		// CHEMISTRY - ELECTROLYTES
		// ==========================================================================
		{
			Code: "2951-2", Name: "Sodium", ShortName: "Na", Category: "Chemistry",
			Unit: "mEq/L", DecimalPlaces: 0, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: ptr(136), High: ptr(145)},
			PanelMembership: []string{"BMP", "CMP"},
		},
		{
			Code: "2823-3", Name: "Potassium", ShortName: "K", Category: "Chemistry",
			Unit: "mEq/L", DecimalPlaces: 1, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: ptr(3.5), High: ptr(5.0)},
			PanelMembership: []string{"BMP", "CMP"},
		},
		{
			Code: "2075-0", Name: "Chloride", ShortName: "Cl", Category: "Chemistry",
			Unit: "mEq/L", DecimalPlaces: 0, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: ptr(98), High: ptr(106)},
			PanelMembership: []string{"BMP", "CMP"},
		},
		{
			Code: "1963-8", Name: "Bicarbonate (CO2)", ShortName: "CO2", Category: "Chemistry",
			Unit: "mEq/L", DecimalPlaces: 0, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: ptr(22), High: ptr(29)},
			PanelMembership: []string{"BMP", "CMP"},
		},
		{
			Code: "17861-6", Name: "Calcium", ShortName: "Ca", Category: "Chemistry",
			Unit: "mg/dL", DecimalPlaces: 1, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: ptr(8.5), High: ptr(10.5)},
			PanelMembership: []string{"BMP", "CMP"},
		},
		{
			Code: "19123-9", Name: "Magnesium", ShortName: "Mg", Category: "Chemistry",
			Unit: "mg/dL", DecimalPlaces: 1, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: ptr(1.7), High: ptr(2.2)},
			PanelMembership: []string{"CMP"},
		},
		{
			Code: "2777-1", Name: "Phosphorus", ShortName: "Phos", Category: "Chemistry",
			Unit: "mg/dL", DecimalPlaces: 1, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: ptr(2.5), High: ptr(4.5)},
			PanelMembership: []string{"CMP"},
		},

		// ==========================================================================
		// CHEMISTRY - RENAL
		// ==========================================================================
		{
			Code: "3094-0", Name: "Blood Urea Nitrogen", ShortName: "BUN", Category: "Chemistry",
			Unit: "mg/dL", DecimalPlaces: 0, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: ptr(7), High: ptr(20)},
			PanelMembership: []string{"BMP", "CMP", "RENAL"},
		},
		{
			Code: "2160-0", Name: "Creatinine", ShortName: "Cr", Category: "Chemistry",
			Unit: "mg/dL", DecimalPlaces: 2, TrendingEnabled: true,
			DefaultRange: &types.ReferenceRange{Low: ptr(0.7), High: ptr(1.3)},
			SexRanges: []SexRange{
				{Sex: "male", Range: &types.ReferenceRange{Low: ptr(0.7), High: ptr(1.3)}},
				{Sex: "female", Range: &types.ReferenceRange{Low: ptr(0.6), High: ptr(1.1)}},
			},
			PanelMembership: []string{"BMP", "CMP", "RENAL"},
		},
		{
			Code: "33914-3", Name: "eGFR (CKD-EPI)", ShortName: "eGFR", Category: "Chemistry",
			Unit: "mL/min/1.73m2", DecimalPlaces: 0, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: ptr(60), High: nil},
			PanelMembership: []string{"RENAL"},
		},

		// ==========================================================================
		// CHEMISTRY - GLUCOSE
		// ==========================================================================
		{
			Code: "2345-7", Name: "Glucose", ShortName: "Glu", Category: "Chemistry",
			Unit: "mg/dL", DecimalPlaces: 0, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: ptr(70), High: ptr(100)},
			PanelMembership: []string{"BMP", "CMP"},
		},
		{
			Code: "4548-4", Name: "Hemoglobin A1c", ShortName: "HbA1c", Category: "Chemistry",
			Unit: "%", DecimalPlaces: 1, TrendingEnabled: true,
			DefaultRange: &types.ReferenceRange{Low: nil, High: ptr(5.7)},
		},

		// ==========================================================================
		// CHEMISTRY - LIVER FUNCTION
		// ==========================================================================
		{
			Code: "1920-8", Name: "AST (SGOT)", ShortName: "AST", Category: "Chemistry",
			Unit: "U/L", DecimalPlaces: 0, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: ptr(10), High: ptr(40)},
			PanelMembership: []string{"CMP", "LFT"},
		},
		{
			Code: "1742-6", Name: "ALT (SGPT)", ShortName: "ALT", Category: "Chemistry",
			Unit: "U/L", DecimalPlaces: 0, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: ptr(7), High: ptr(56)},
			PanelMembership: []string{"CMP", "LFT"},
		},
		{
			Code: "6768-6", Name: "Alkaline Phosphatase", ShortName: "ALP", Category: "Chemistry",
			Unit: "U/L", DecimalPlaces: 0, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: ptr(44), High: ptr(147)},
			PanelMembership: []string{"CMP", "LFT"},
		},
		{
			Code: "1975-2", Name: "Total Bilirubin", ShortName: "T.Bili", Category: "Chemistry",
			Unit: "mg/dL", DecimalPlaces: 1, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: ptr(0.1), High: ptr(1.2)},
			PanelMembership: []string{"CMP", "LFT"},
		},
		{
			Code: "1968-7", Name: "Direct Bilirubin", ShortName: "D.Bili", Category: "Chemistry",
			Unit: "mg/dL", DecimalPlaces: 1, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: ptr(0.0), High: ptr(0.3)},
			PanelMembership: []string{"LFT"},
		},
		{
			Code: "1751-7", Name: "Albumin", ShortName: "Alb", Category: "Chemistry",
			Unit: "g/dL", DecimalPlaces: 1, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: ptr(3.5), High: ptr(5.0)},
			PanelMembership: []string{"CMP", "LFT"},
		},
		{
			Code: "2885-2", Name: "Total Protein", ShortName: "TP", Category: "Chemistry",
			Unit: "g/dL", DecimalPlaces: 1, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: ptr(6.0), High: ptr(8.3)},
			PanelMembership: []string{"CMP", "LFT"},
		},

		// ==========================================================================
		// HEMATOLOGY - CBC
		// ==========================================================================
		{
			Code: "6690-2", Name: "White Blood Cells", ShortName: "WBC", Category: "Hematology",
			Unit: "x10^3/uL", DecimalPlaces: 1, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: ptr(4.5), High: ptr(11.0)},
			PanelMembership: []string{"CBC"},
		},
		{
			Code: "789-8", Name: "Red Blood Cells", ShortName: "RBC", Category: "Hematology",
			Unit: "x10^6/uL", DecimalPlaces: 2, TrendingEnabled: true,
			DefaultRange: &types.ReferenceRange{Low: ptr(4.5), High: ptr(5.5)},
			SexRanges: []SexRange{
				{Sex: "male", Range: &types.ReferenceRange{Low: ptr(4.7), High: ptr(6.1)}},
				{Sex: "female", Range: &types.ReferenceRange{Low: ptr(4.2), High: ptr(5.4)}},
			},
			PanelMembership: []string{"CBC"},
		},
		{
			Code: "718-7", Name: "Hemoglobin", ShortName: "Hgb", Category: "Hematology",
			Unit: "g/dL", DecimalPlaces: 1, TrendingEnabled: true,
			DefaultRange: &types.ReferenceRange{Low: ptr(12.0), High: ptr(16.0)},
			SexRanges: []SexRange{
				{Sex: "male", Range: &types.ReferenceRange{Low: ptr(13.5), High: ptr(17.5)}},
				{Sex: "female", Range: &types.ReferenceRange{Low: ptr(12.0), High: ptr(16.0)}},
			},
			PanelMembership: []string{"CBC"},
		},
		{
			Code: "4544-3", Name: "Hematocrit", ShortName: "Hct", Category: "Hematology",
			Unit: "%", DecimalPlaces: 1, TrendingEnabled: true,
			DefaultRange: &types.ReferenceRange{Low: ptr(36), High: ptr(46)},
			SexRanges: []SexRange{
				{Sex: "male", Range: &types.ReferenceRange{Low: ptr(38.3), High: ptr(48.6)}},
				{Sex: "female", Range: &types.ReferenceRange{Low: ptr(35.5), High: ptr(44.9)}},
			},
			PanelMembership: []string{"CBC"},
		},
		{
			Code: "777-3", Name: "Platelets", ShortName: "Plt", Category: "Hematology",
			Unit: "x10^3/uL", DecimalPlaces: 0, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: ptr(150), High: ptr(400)},
			PanelMembership: []string{"CBC"},
		},
		{
			Code: "787-2", Name: "Mean Corpuscular Volume", ShortName: "MCV", Category: "Hematology",
			Unit: "fL", DecimalPlaces: 1, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: ptr(80), High: ptr(100)},
			PanelMembership: []string{"CBC"},
		},
		{
			Code: "785-6", Name: "Mean Corpuscular Hemoglobin", ShortName: "MCH", Category: "Hematology",
			Unit: "pg", DecimalPlaces: 1, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: ptr(27), High: ptr(33)},
			PanelMembership: []string{"CBC"},
		},
		{
			Code: "786-4", Name: "MCHC", ShortName: "MCHC", Category: "Hematology",
			Unit: "g/dL", DecimalPlaces: 1, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: ptr(32), High: ptr(36)},
			PanelMembership: []string{"CBC"},
		},

		// ==========================================================================
		// COAGULATION
		// ==========================================================================
		{
			Code: "5902-2", Name: "Prothrombin Time", ShortName: "PT", Category: "Coagulation",
			Unit: "seconds", DecimalPlaces: 1, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: ptr(11), High: ptr(13.5)},
			PanelMembership: []string{"COAG"},
		},
		{
			Code: "34714-6", Name: "INR", ShortName: "INR", Category: "Coagulation",
			Unit: "", DecimalPlaces: 1, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: ptr(0.8), High: ptr(1.1)},
			PanelMembership: []string{"COAG"},
		},
		{
			Code: "3173-2", Name: "Partial Thromboplastin Time", ShortName: "PTT", Category: "Coagulation",
			Unit: "seconds", DecimalPlaces: 1, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: ptr(25), High: ptr(35)},
			PanelMembership: []string{"COAG"},
		},

		// ==========================================================================
		// CARDIAC MARKERS
		// ==========================================================================
		{
			Code: "10839-9", Name: "Troponin I", ShortName: "TnI", Category: "Cardiac",
			Unit: "ng/mL", DecimalPlaces: 3, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: nil, High: ptr(0.04)},
			PanelMembership: []string{"CARDIAC"},
		},
		{
			Code: "33762-6", Name: "NT-proBNP", ShortName: "BNP", Category: "Cardiac",
			Unit: "pg/mL", DecimalPlaces: 0, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: nil, High: ptr(125)},
			PanelMembership: []string{"CARDIAC"},
		},
		{
			Code: "2157-6", Name: "CK-MB", ShortName: "CK-MB", Category: "Cardiac",
			Unit: "ng/mL", DecimalPlaces: 1, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: nil, High: ptr(5.0)},
			PanelMembership: []string{"CARDIAC"},
		},

		// ==========================================================================
		// LIPID PANEL
		// ==========================================================================
		{
			Code: "2093-3", Name: "Total Cholesterol", ShortName: "TC", Category: "Lipids",
			Unit: "mg/dL", DecimalPlaces: 0, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: nil, High: ptr(200)},
			PanelMembership: []string{"LIPID"},
		},
		{
			Code: "2571-8", Name: "Triglycerides", ShortName: "TG", Category: "Lipids",
			Unit: "mg/dL", DecimalPlaces: 0, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: nil, High: ptr(150)},
			PanelMembership: []string{"LIPID"},
		},
		{
			Code: "2085-9", Name: "HDL Cholesterol", ShortName: "HDL", Category: "Lipids",
			Unit: "mg/dL", DecimalPlaces: 0, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: ptr(40), High: nil},
			PanelMembership: []string{"LIPID"},
		},
		{
			Code: "13457-7", Name: "LDL Cholesterol (Calculated)", ShortName: "LDL", Category: "Lipids",
			Unit: "mg/dL", DecimalPlaces: 0, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: nil, High: ptr(100)},
			PanelMembership: []string{"LIPID"},
		},

		// ==========================================================================
		// THYROID
		// ==========================================================================
		{
			Code: "3016-3", Name: "TSH", ShortName: "TSH", Category: "Thyroid",
			Unit: "mIU/L", DecimalPlaces: 2, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: ptr(0.4), High: ptr(4.0)},
			PanelMembership: []string{"THYROID"},
		},
		{
			Code: "3026-2", Name: "Free T4", ShortName: "FT4", Category: "Thyroid",
			Unit: "ng/dL", DecimalPlaces: 2, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: ptr(0.8), High: ptr(1.8)},
			PanelMembership: []string{"THYROID"},
		},
		{
			Code: "3053-6", Name: "Free T3", ShortName: "FT3", Category: "Thyroid",
			Unit: "pg/mL", DecimalPlaces: 1, TrendingEnabled: true,
			DefaultRange:    &types.ReferenceRange{Low: ptr(2.3), High: ptr(4.2)},
			PanelMembership: []string{"THYROID"},
		},

		// ==========================================================================
		// INFLAMMATORY MARKERS
		// ==========================================================================
		{
			Code: "1988-5", Name: "C-Reactive Protein", ShortName: "CRP", Category: "Inflammatory",
			Unit: "mg/L", DecimalPlaces: 1, TrendingEnabled: true,
			DefaultRange: &types.ReferenceRange{Low: nil, High: ptr(3.0)},
		},
		{
			Code: "30341-2", Name: "ESR", ShortName: "ESR", Category: "Inflammatory",
			Unit: "mm/hr", DecimalPlaces: 0, TrendingEnabled: true,
			DefaultRange: &types.ReferenceRange{Low: nil, High: ptr(20)},
		},
		{
			Code: "33959-8", Name: "Procalcitonin", ShortName: "PCT", Category: "Inflammatory",
			Unit: "ng/mL", DecimalPlaces: 2, TrendingEnabled: true,
			DefaultRange: &types.ReferenceRange{Low: nil, High: ptr(0.1)},
		},
		{
			Code: "2524-7", Name: "Lactate", ShortName: "Lactate", Category: "Chemistry",
			Unit: "mmol/L", DecimalPlaces: 1, TrendingEnabled: true,
			DefaultRange: &types.ReferenceRange{Low: ptr(0.5), High: ptr(2.0)},
		},
	}

	// Populate maps
	for _, t := range tests {
		db.tests[t.Code] = t

		if db.byCategory[t.Category] == nil {
			db.byCategory[t.Category] = make([]*TestDefinition, 0)
		}
		db.byCategory[t.Category] = append(db.byCategory[t.Category], t)
	}
}

// initializeCriticalValues sets up critical and panic value thresholds
func (db *Database) initializeCriticalValues() {
	// Panic values - immediate notification required
	panicVals := []CriticalRange{
		{Code: "2823-3", Name: "Potassium", Low: ptr(2.5), High: ptr(6.5)},
		{Code: "2951-2", Name: "Sodium", Low: ptr(120), High: ptr(160)},
		{Code: "2345-7", Name: "Glucose", Low: ptr(40), High: ptr(500)},
		{Code: "718-7", Name: "Hemoglobin", Low: ptr(5.0), High: nil},
		{Code: "777-3", Name: "Platelets", Low: ptr(20000), High: nil},
		{Code: "34714-6", Name: "INR", Low: nil, High: ptr(8.0)},
		{Code: "2524-7", Name: "Lactate", Low: nil, High: ptr(7.0)},
	}

	for _, pv := range panicVals {
		db.panicVals[pv.Code] = &CriticalRange{
			Code: pv.Code,
			Name: pv.Name,
			Low:  pv.Low,
			High: pv.High,
		}
	}

	// Critical values - 30-minute notification
	criticalVals := []CriticalRange{
		{Code: "2823-3", Name: "Potassium", Low: ptr(3.0), High: ptr(6.0)},
		{Code: "2951-2", Name: "Sodium", Low: ptr(125), High: ptr(155)},
		{Code: "2345-7", Name: "Glucose", Low: ptr(50), High: ptr(400)},
		{Code: "718-7", Name: "Hemoglobin", Low: ptr(7.0), High: ptr(20.0)},
		{Code: "777-3", Name: "Platelets", Low: ptr(50000), High: ptr(1000000)},
		{Code: "10839-9", Name: "Troponin", Low: nil, High: ptr(0.04)},
		{Code: "2160-0", Name: "Creatinine", Low: nil, High: ptr(10.0)},
		{Code: "17861-6", Name: "Calcium", Low: ptr(6.5), High: ptr(13.0)},
	}

	for _, cv := range criticalVals {
		db.criticalVals[cv.Code] = &CriticalRange{
			Code: cv.Code,
			Name: cv.Name,
			Low:  cv.Low,
			High: cv.High,
		}
	}
}

// initializeDeltaRules sets up delta check thresholds
func (db *Database) initializeDeltaRules() {
	deltaRules := []DeltaRule{
		{Code: "718-7", Name: "Hemoglobin", Threshold: 2.0, Direction: "decrease", WindowHours: 24},
		{Code: "2160-0", Name: "Creatinine", ThresholdPercent: 50.0, Direction: "increase", WindowHours: 48},
		{Code: "777-3", Name: "Platelets", ThresholdPercent: 50.0, Direction: "decrease", WindowHours: 24},
		{Code: "2823-3", Name: "Potassium", Threshold: 1.0, Direction: "any", WindowHours: 24},
		{Code: "2951-2", Name: "Sodium", Threshold: 8.0, Direction: "any", WindowHours: 24},
		{Code: "6690-2", Name: "WBC", ThresholdPercent: 50.0, Direction: "any", WindowHours: 24},
	}

	for _, dr := range deltaRules {
		db.deltaRules[dr.Code] = &DeltaRule{
			Code:             dr.Code,
			Name:             dr.Name,
			Threshold:        dr.Threshold,
			ThresholdPercent: dr.ThresholdPercent,
			Direction:        dr.Direction,
			WindowHours:      dr.WindowHours,
		}
	}
}
