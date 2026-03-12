// Package reference provides context-aware lab reference range functionality
// Phase 3b.6: Conditional Reference Ranges for Clinical Lab Interpretation
package reference

import (
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// LAB TEST DEFINITION
// ============================================================================

// LabTest represents a centralized LOINC-based lab test definition
type LabTest struct {
	ID              uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	LOINCCode       string    `json:"loinc_code" gorm:"column:loinc_code;type:varchar(20);uniqueIndex;not null"`
	TestName        string    `json:"test_name" gorm:"column:test_name;type:varchar(200);not null"`
	ShortName       string    `json:"short_name,omitempty" gorm:"column:short_name;type:varchar(50)"`
	Unit            string    `json:"unit" gorm:"column:unit;type:varchar(50);not null"`
	SpecimenType    string    `json:"specimen_type,omitempty" gorm:"column:specimen_type;type:varchar(50)"` // blood, urine, csf
	Method          string    `json:"method,omitempty" gorm:"column:method;type:varchar(100)"`              // enzymatic, colorimetric
	Category        string    `json:"category,omitempty" gorm:"column:category;type:varchar(50)"`           // Chemistry, Hematology
	DecimalPlaces   int       `json:"decimal_places" gorm:"column:decimal_places;default:2"`
	TrendingEnabled bool      `json:"trending_enabled" gorm:"column:trending_enabled;default:true"`
	IsActive        bool      `json:"is_active" gorm:"column:is_active;default:true"`
	CreatedAt       time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
}

// TableName specifies the table name for GORM
func (LabTest) TableName() string {
	return "lab_tests"
}

// ============================================================================
// RANGE CONDITIONS (Patient Context Matching)
// ============================================================================

// RangeConditions defines all patient context variables for range selection
// NULL values mean "any" - only non-null conditions must match for range to apply
type RangeConditions struct {
	// Demographics
	Gender      *string  `json:"gender,omitempty" gorm:"column:gender;type:varchar(1)"`              // M, F, null=any
	AgeMinYears *float64 `json:"age_min_years,omitempty" gorm:"column:age_min_years;type:decimal(5,2)"`
	AgeMaxYears *float64 `json:"age_max_years,omitempty" gorm:"column:age_max_years;type:decimal(5,2)"` // Exclusive upper bound
	AgeMinDays  *int     `json:"age_min_days,omitempty" gorm:"column:age_min_days"`                     // For neonates
	AgeMaxDays  *int     `json:"age_max_days,omitempty" gorm:"column:age_max_days"`

	// Pregnancy & Lactation (ACOG, ATA guidelines)
	IsPregnant     *bool `json:"is_pregnant,omitempty" gorm:"column:is_pregnant"`
	Trimester      *int  `json:"trimester,omitempty" gorm:"column:trimester"`             // 1, 2, 3
	IsPostpartum   *bool `json:"is_postpartum,omitempty" gorm:"column:is_postpartum"`
	PostpartumWeeks *int  `json:"postpartum_weeks,omitempty" gorm:"column:postpartum_weeks"`
	IsLactating    *bool `json:"is_lactating,omitempty" gorm:"column:is_lactating"`

	// Neonatal (AAP 2022 bilirubin guidelines)
	GestationalAgeWeeksMin *int `json:"ga_weeks_min,omitempty" gorm:"column:gestational_age_weeks_min"`
	GestationalAgeWeeksMax *int `json:"ga_weeks_max,omitempty" gorm:"column:gestational_age_weeks_max"`
	HoursOfLifeMin         *int `json:"hours_of_life_min,omitempty" gorm:"column:hours_of_life_min"`
	HoursOfLifeMax         *int `json:"hours_of_life_max,omitempty" gorm:"column:hours_of_life_max"`

	// Renal Status (KDIGO guidelines)
	CKDStage     *int     `json:"ckd_stage,omitempty" gorm:"column:ckd_stage"`           // 1-5
	IsOnDialysis *bool    `json:"is_on_dialysis,omitempty" gorm:"column:is_on_dialysis"`
	EGFRMin      *float64 `json:"egfr_min,omitempty" gorm:"column:egfr_min;type:decimal(6,2)"`
	EGFRMax      *float64 `json:"egfr_max,omitempty" gorm:"column:egfr_max;type:decimal(6,2)"`
}

// ============================================================================
// CONDITIONAL REFERENCE RANGE
// ============================================================================

// ConditionalReferenceRange represents a reference range with patient conditions
// When a lab result is interpreted, the most specific matching range is selected
type ConditionalReferenceRange struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	LabTestID uuid.UUID `json:"lab_test_id" gorm:"column:lab_test_id;type:uuid;not null;index"`

	// Embedded conditions - all must match for this range to apply
	RangeConditions `gorm:"embedded"`

	// Reference Values
	LowNormal    *float64 `json:"low_normal,omitempty" gorm:"column:low_normal;type:decimal(10,4)"`
	HighNormal   *float64 `json:"high_normal,omitempty" gorm:"column:high_normal;type:decimal(10,4)"`
	CriticalLow  *float64 `json:"critical_low,omitempty" gorm:"column:critical_low;type:decimal(10,4)"`
	CriticalHigh *float64 `json:"critical_high,omitempty" gorm:"column:critical_high;type:decimal(10,4)"`
	PanicLow     *float64 `json:"panic_low,omitempty" gorm:"column:panic_low;type:decimal(10,4)"`
	PanicHigh    *float64 `json:"panic_high,omitempty" gorm:"column:panic_high;type:decimal(10,4)"`

	// Interpretation Guidance
	InterpretationNote string `json:"interpretation_note,omitempty" gorm:"column:interpretation_note;type:text"`
	ClinicalAction     string `json:"clinical_action,omitempty" gorm:"column:clinical_action;type:text"`

	// Governance (Authority & Version Tracking)
	Authority        string     `json:"authority" gorm:"column:authority;type:varchar(50);not null"`       // CLSI, ACOG, ATA, KDIGO
	AuthorityRef     string     `json:"authority_ref" gorm:"column:authority_reference;type:text;not null"` // Specific document
	AuthorityVersion string     `json:"authority_version,omitempty" gorm:"column:authority_version;type:varchar(50)"`
	EffectiveDate    time.Time  `json:"effective_date" gorm:"column:effective_date;type:date;not null"`
	ExpirationDate   *time.Time `json:"expiration_date,omitempty" gorm:"column:expiration_date;type:date"`

	// Specificity Scoring - higher score = more specific = wins selection
	SpecificityScore int `json:"specificity_score" gorm:"column:specificity_score;default:0;index"`

	// Metadata
	IsActive  bool      `json:"is_active" gorm:"column:is_active;default:true"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`

	// Relationship (for eager loading)
	LabTest *LabTest `json:"lab_test,omitempty" gorm:"foreignKey:LabTestID"`
}

// TableName specifies the table name for GORM
func (ConditionalReferenceRange) TableName() string {
	return "conditional_reference_ranges"
}

// ============================================================================
// NEONATAL BILIRUBIN THRESHOLD (Bhutani Nomogram)
// ============================================================================

// RiskCategory represents neonatal jaundice risk levels
type RiskCategory string

const (
	RiskLow    RiskCategory = "LOW"    // ≥38 weeks, no risk factors
	RiskMedium RiskCategory = "MEDIUM" // 35-37 weeks OR risk factors
	RiskHigh   RiskCategory = "HIGH"   // <35 weeks
)

// NeonatalBilirubinThreshold represents AAP 2022 Bhutani nomogram thresholds
type NeonatalBilirubinThreshold struct {
	ID uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`

	// Risk Stratification
	GestationalAgeWeeksMin int          `json:"ga_weeks_min" gorm:"column:gestational_age_weeks_min;not null"`
	GestationalAgeWeeksMax int          `json:"ga_weeks_max" gorm:"column:gestational_age_weeks_max;not null"`
	RiskCategory           RiskCategory `json:"risk_category" gorm:"column:risk_category;type:varchar(20);not null"`

	// Hour-of-Life Threshold Point
	HourOfLife int `json:"hour_of_life" gorm:"column:hour_of_life;not null"`

	// Treatment Thresholds (mg/dL)
	PhotoThreshold    float64  `json:"photo_threshold" gorm:"column:photo_threshold;type:decimal(5,2);not null"`
	ExchangeThreshold *float64 `json:"exchange_threshold,omitempty" gorm:"column:exchange_threshold;type:decimal(5,2)"`

	// Governance
	Authority        string `json:"authority" gorm:"column:authority;type:varchar(50);default:'AAP'"`
	AuthorityRef     string `json:"authority_ref" gorm:"column:authority_reference;type:text;default:'AAP Clinical Practice Guideline: Management of Hyperbilirubinemia 2022'"`

	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}

// TableName specifies the table name for GORM
func (NeonatalBilirubinThreshold) TableName() string {
	return "neonatal_bilirubin_thresholds"
}

// ============================================================================
// INTERPRETATION RESULT TYPES
// ============================================================================

// InterpretationFlag represents the result of comparing a value to a reference range
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

// RangeInterpretation holds the result of interpreting a lab value against a range
type RangeInterpretation struct {
	Flag               InterpretationFlag `json:"flag"`
	Value              float64            `json:"value"`
	Unit               string             `json:"unit"`
	LowNormal          *float64           `json:"low_normal,omitempty"`
	HighNormal         *float64           `json:"high_normal,omitempty"`
	DeviationPercent   *float64           `json:"deviation_percent,omitempty"`
	DeviationDirection string             `json:"deviation_direction,omitempty"` // "above" or "below"
	RangeID            uuid.UUID          `json:"range_id"`
	Authority          string             `json:"authority"`
	AuthorityRef       string             `json:"authority_ref"`
	InterpretationNote string             `json:"interpretation_note,omitempty"`
	ClinicalAction     string             `json:"clinical_action,omitempty"`
	ContextApplied     string             `json:"context_applied"` // e.g., "Pregnancy T3", "CKD Stage 4"
	SpecificityScore   int                `json:"specificity_score"`
}

// BilirubinInterpretation holds neonatal bilirubin assessment results
type BilirubinInterpretation struct {
	Value             float64       `json:"value"`
	Unit              string        `json:"unit"` // mg/dL
	HoursOfLife       int           `json:"hours_of_life"`
	GestationalAge    int           `json:"gestational_age"`
	RiskCategory      RiskCategory  `json:"risk_category"`
	PhotoThreshold    float64       `json:"photo_threshold"`
	ExchangeThreshold *float64      `json:"exchange_threshold,omitempty"`
	NeedsPhototherapy bool          `json:"needs_phototherapy"`
	NeedsExchange     bool          `json:"needs_exchange"`
	Zone              string        `json:"zone"` // "LOW_RISK", "LOW_INTERMEDIATE", "HIGH_INTERMEDIATE", "HIGH_RISK"
	Interpolated      bool          `json:"interpolated"` // True if threshold was interpolated between hour points
	Authority         string        `json:"authority"`
	AuthorityRef      string        `json:"authority_ref"`
}

// ============================================================================
// HELPER METHODS
// ============================================================================

// IsExpired returns true if the reference range has expired
func (r *ConditionalReferenceRange) IsExpired() bool {
	if r.ExpirationDate == nil {
		return false
	}
	return time.Now().After(*r.ExpirationDate)
}

// IsEffective returns true if the reference range is currently effective
func (r *ConditionalReferenceRange) IsEffective() bool {
	now := time.Now()
	if now.Before(r.EffectiveDate) {
		return false
	}
	return !r.IsExpired()
}

// ContextDescription returns a human-readable description of the conditions
func (r *ConditionalReferenceRange) ContextDescription() string {
	var parts []string

	if r.IsPregnant != nil && *r.IsPregnant {
		if r.Trimester != nil {
			parts = append(parts, trimesterName(*r.Trimester))
		} else {
			parts = append(parts, "Pregnant")
		}
	}

	if r.CKDStage != nil {
		parts = append(parts, ckdStageName(*r.CKDStage))
	}

	if r.IsOnDialysis != nil && *r.IsOnDialysis {
		parts = append(parts, "Dialysis")
	}

	if r.Gender != nil {
		if *r.Gender == "M" {
			parts = append(parts, "Male")
		} else if *r.Gender == "F" {
			parts = append(parts, "Female")
		}
	}

	if r.AgeMinYears != nil || r.AgeMaxYears != nil {
		parts = append(parts, ageRangeName(r.AgeMinYears, r.AgeMaxYears))
	}

	if len(parts) == 0 {
		return "Standard"
	}

	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += ", " + parts[i]
	}
	return result
}

func trimesterName(t int) string {
	switch t {
	case 1:
		return "Pregnancy T1"
	case 2:
		return "Pregnancy T2"
	case 3:
		return "Pregnancy T3"
	default:
		return "Pregnant"
	}
}

func ckdStageName(stage int) string {
	switch stage {
	case 1:
		return "CKD Stage 1"
	case 2:
		return "CKD Stage 2"
	case 3:
		return "CKD Stage 3"
	case 4:
		return "CKD Stage 4"
	case 5:
		return "CKD Stage 5"
	default:
		return "CKD"
	}
}

func ageRangeName(min, max *float64) string {
	if min != nil && max != nil {
		if *min < 1 {
			return "Neonate"
		} else if *max <= 18 {
			return "Pediatric"
		} else if *min >= 65 {
			return "Geriatric"
		}
		return "Adult"
	}
	if min != nil && *min >= 65 {
		return "Geriatric"
	}
	if max != nil && *max <= 18 {
		return "Pediatric"
	}
	return "Adult"
}

// Interpret evaluates a lab value against this reference range
func (r *ConditionalReferenceRange) Interpret(value float64, unit string) *RangeInterpretation {
	result := &RangeInterpretation{
		Value:            value,
		Unit:             unit,
		LowNormal:        r.LowNormal,
		HighNormal:       r.HighNormal,
		RangeID:          r.ID,
		Authority:        r.Authority,
		AuthorityRef:     r.AuthorityRef,
		InterpretationNote: r.InterpretationNote,
		ClinicalAction:   r.ClinicalAction,
		ContextApplied:   r.ContextDescription(),
		SpecificityScore: r.SpecificityScore,
	}

	// Determine flag based on value vs thresholds
	switch {
	case r.PanicLow != nil && value < *r.PanicLow:
		result.Flag = FlagPanicLow
		result.DeviationDirection = "below"
		if r.LowNormal != nil {
			dev := ((*r.LowNormal - value) / *r.LowNormal) * 100
			result.DeviationPercent = &dev
		}
	case r.PanicHigh != nil && value > *r.PanicHigh:
		result.Flag = FlagPanicHigh
		result.DeviationDirection = "above"
		if r.HighNormal != nil {
			dev := ((value - *r.HighNormal) / *r.HighNormal) * 100
			result.DeviationPercent = &dev
		}
	case r.CriticalLow != nil && value < *r.CriticalLow:
		result.Flag = FlagCriticalLow
		result.DeviationDirection = "below"
		if r.LowNormal != nil {
			dev := ((*r.LowNormal - value) / *r.LowNormal) * 100
			result.DeviationPercent = &dev
		}
	case r.CriticalHigh != nil && value > *r.CriticalHigh:
		result.Flag = FlagCriticalHigh
		result.DeviationDirection = "above"
		if r.HighNormal != nil {
			dev := ((value - *r.HighNormal) / *r.HighNormal) * 100
			result.DeviationPercent = &dev
		}
	case r.LowNormal != nil && value < *r.LowNormal:
		result.Flag = FlagLow
		result.DeviationDirection = "below"
		dev := ((*r.LowNormal - value) / *r.LowNormal) * 100
		result.DeviationPercent = &dev
	case r.HighNormal != nil && value > *r.HighNormal:
		result.Flag = FlagHigh
		result.DeviationDirection = "above"
		dev := ((value - *r.HighNormal) / *r.HighNormal) * 100
		result.DeviationPercent = &dev
	default:
		result.Flag = FlagNormal
	}

	return result
}
