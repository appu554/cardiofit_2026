package models

// EGFRParams contains input parameters for eGFR calculation.
// Uses CKD-EPI 2021 race-free equation.
type EGFRParams struct {
	// SerumCreatinine in mg/dL
	SerumCreatinine float64 `json:"serumCreatinine" binding:"required,gt=0"`

	// AgeYears patient's age in years
	AgeYears int `json:"ageYears" binding:"required,gt=0,lte=120"`

	// Sex biological sex (male/female)
	Sex Sex `json:"sex" binding:"required"`
}

// Validate checks if the parameters are valid.
func (p *EGFRParams) Validate() error {
	if p.SerumCreatinine <= 0 {
		return ErrInvalidCreatinine
	}
	if p.AgeYears <= 0 || p.AgeYears > 120 {
		return ErrInvalidAge
	}
	if !p.Sex.IsValid() {
		return ErrInvalidSex
	}
	return nil
}

// CrClParams contains input parameters for CrCl (Cockcroft-Gault) calculation.
type CrClParams struct {
	// SerumCreatinine in mg/dL
	SerumCreatinine float64 `json:"serumCreatinine" binding:"required,gt=0"`

	// AgeYears patient's age in years
	AgeYears int `json:"ageYears" binding:"required,gt=0,lte=120"`

	// Sex biological sex (male/female)
	Sex Sex `json:"sex" binding:"required"`

	// WeightKg actual body weight in kg
	WeightKg float64 `json:"weightKg" binding:"required,gt=0"`
}

// Validate checks if the parameters are valid.
func (p *CrClParams) Validate() error {
	if p.SerumCreatinine <= 0 {
		return ErrInvalidCreatinine
	}
	if p.AgeYears <= 0 || p.AgeYears > 120 {
		return ErrInvalidAge
	}
	if !p.Sex.IsValid() {
		return ErrInvalidSex
	}
	if p.WeightKg <= 0 || p.WeightKg > 500 {
		return ErrInvalidWeight
	}
	return nil
}

// BMIParams contains input parameters for BMI calculation.
type BMIParams struct {
	// WeightKg in kilograms
	WeightKg float64 `json:"weightKg" binding:"required,gt=0"`

	// HeightCm in centimeters
	HeightCm float64 `json:"heightCm" binding:"required,gt=0"`

	// Region for category cutoffs (defaults to GLOBAL)
	Region Region `json:"region,omitempty"`

	// Ethnicity for regional adjustments (optional)
	Ethnicity string `json:"ethnicity,omitempty"`
}

// Validate checks if the parameters are valid.
func (p *BMIParams) Validate() error {
	if p.WeightKg <= 0 || p.WeightKg > 500 {
		return ErrInvalidWeight
	}
	if p.HeightCm <= 0 || p.HeightCm > 300 {
		return ErrInvalidHeight
	}
	return nil
}

// SOFAParams contains input parameters for SOFA score calculation.
// All components are optional - missing data will be flagged.
type SOFAParams struct {
	// Respiration: PaO2/FiO2 ratio (mmHg)
	PaO2FiO2Ratio *float64 `json:"pao2fio2Ratio,omitempty"`
	OnMechanicalVentilation bool `json:"onMechanicalVentilation,omitempty"`

	// Coagulation: Platelets (×10³/µL)
	Platelets *float64 `json:"platelets,omitempty"`

	// Liver: Bilirubin (mg/dL)
	Bilirubin *float64 `json:"bilirubin,omitempty"`

	// Cardiovascular: MAP (mmHg) or vasopressor requirements
	MAP *float64 `json:"map,omitempty"`
	// Vasopressor dose in µg/kg/min (dopamine, dobutamine, epinephrine, norepinephrine)
	DopamineDose      *float64 `json:"dopamineDose,omitempty"`
	DobutamineDose    *float64 `json:"dobutamineDose,omitempty"`
	EpinephrineDose   *float64 `json:"epinephrineDose,omitempty"`
	NorepinephrineDose *float64 `json:"norepinephrineDose,omitempty"`

	// CNS: Glasgow Coma Scale (3-15)
	GlasgowComaScale *int `json:"glasgowComaScale,omitempty"`

	// Renal: Creatinine (mg/dL) or urine output (mL/day)
	Creatinine  *float64 `json:"creatinine,omitempty"`
	UrineOutput *float64 `json:"urineOutput,omitempty"` // mL in 24 hours
}

// QSOFAParams contains input parameters for qSOFA score calculation.
type QSOFAParams struct {
	// RespiratoryRate breaths per minute
	RespiratoryRate *int `json:"respiratoryRate,omitempty"`

	// SystolicBP in mmHg
	SystolicBP *int `json:"systolicBP,omitempty"`

	// AlteredMentation true if GCS < 15 or altered consciousness
	AlteredMentation *bool `json:"alteredMentation,omitempty"`

	// Alternatively, can use GCS directly
	GlasgowComaScale *int `json:"glasgowComaScale,omitempty"`
}

// CHA2DS2VAScParams contains input parameters for CHA2DS2-VASc score.
type CHA2DS2VAScParams struct {
	// AgeYears patient's age
	AgeYears int `json:"ageYears" binding:"required"`

	// Sex biological sex
	Sex Sex `json:"sex" binding:"required"`

	// Conditions (boolean flags)
	HasCongestiveHeartFailure bool `json:"hasCongestiveHeartFailure,omitempty"`
	HasHypertension           bool `json:"hasHypertension,omitempty"`
	HasDiabetes               bool `json:"hasDiabetes,omitempty"`
	HasStrokeTIA              bool `json:"hasStrokeTIA,omitempty"` // Prior stroke or TIA
	HasVascularDisease        bool `json:"hasVascularDisease,omitempty"` // MI, PAD, aortic plaque
}

// HASBLEDParams contains input parameters for HAS-BLED score.
type HASBLEDParams struct {
	// H - Hypertension (uncontrolled, SBP > 160)
	HasUncontrolledHypertension bool `json:"hasUncontrolledHypertension,omitempty"`

	// A - Abnormal renal function (dialysis, transplant, Cr > 2.26 mg/dL)
	HasAbnormalRenalFunction bool `json:"hasAbnormalRenalFunction,omitempty"`

	// A - Abnormal liver function (cirrhosis, bilirubin > 2x, AST/ALT > 3x)
	HasAbnormalLiverFunction bool `json:"hasAbnormalLiverFunction,omitempty"`

	// S - Stroke history
	HasStrokeHistory bool `json:"hasStrokeHistory,omitempty"`

	// B - Bleeding history or predisposition
	HasBleedingHistory bool `json:"hasBleedingHistory,omitempty"`

	// L - Labile INR (< 60% time in therapeutic range)
	HasLabileINR bool `json:"hasLabileINR,omitempty"`

	// E - Elderly (> 65 years)
	AgeYears int `json:"ageYears,omitempty"`

	// D - Drugs (antiplatelet, NSAIDs)
	TakingAntiplateletOrNSAID bool `json:"takingAntiplateletOrNSAID,omitempty"`

	// D - Alcohol (>= 8 drinks/week)
	ExcessiveAlcohol bool `json:"excessiveAlcohol,omitempty"`
}

// ASCVDParams contains input parameters for ASCVD 10-year risk calculation.
// Uses Pooled Cohort Equations (2013/2018).
type ASCVDParams struct {
	// Demographics
	AgeYears int  `json:"ageYears" binding:"required"`
	Sex      Sex  `json:"sex" binding:"required"`
	Race     string `json:"race,omitempty"` // "white", "african_american", "other"

	// Lipids
	TotalCholesterol float64 `json:"totalCholesterol" binding:"required"` // mg/dL
	HDLCholesterol   float64 `json:"hdlCholesterol" binding:"required"`   // mg/dL

	// Blood Pressure
	SystolicBP   float64 `json:"systolicBP" binding:"required"` // mmHg
	OnBPTreatment bool   `json:"onBPTreatment,omitempty"`

	// Conditions
	HasDiabetes bool `json:"hasDiabetes,omitempty"`
	IsSmoker    bool `json:"isSmoker,omitempty"`
}

// BatchCalculatorRequest contains a batch of calculator requests.
type BatchCalculatorRequest struct {
	// PatientID for tracking
	PatientID string `json:"patientId,omitempty"`

	// Calculators to run
	Calculators []CalculatorType `json:"calculators" binding:"required"`

	// Shared parameters (applicable to multiple calculators)
	AgeYears        *int     `json:"ageYears,omitempty"`
	Sex             Sex      `json:"sex,omitempty"`
	SerumCreatinine *float64 `json:"serumCreatinine,omitempty"`
	WeightKg        *float64 `json:"weightKg,omitempty"`
	HeightCm        *float64 `json:"heightCm,omitempty"`

	// SOFA-specific
	PaO2FiO2Ratio    *float64 `json:"pao2fio2Ratio,omitempty"`
	Platelets        *float64 `json:"platelets,omitempty"`
	Bilirubin        *float64 `json:"bilirubin,omitempty"`
	MAP              *float64 `json:"map,omitempty"`
	GlasgowComaScale *int     `json:"glasgowComaScale,omitempty"`
	UrineOutput      *float64 `json:"urineOutput,omitempty"`

	// qSOFA-specific
	RespiratoryRate  *int  `json:"respiratoryRate,omitempty"`
	SystolicBP       *int  `json:"systolicBP,omitempty"`
	AlteredMentation *bool `json:"alteredMentation,omitempty"`

	// CHA2DS2-VASc
	HasCongestiveHeartFailure bool `json:"hasCongestiveHeartFailure,omitempty"`
	HasHypertension           bool `json:"hasHypertension,omitempty"`
	HasDiabetes               bool `json:"hasDiabetes,omitempty"`
	HasStrokeTIA              bool `json:"hasStrokeTIA,omitempty"`
	HasVascularDisease        bool `json:"hasVascularDisease,omitempty"`

	// ASCVD-specific
	TotalCholesterol *float64 `json:"totalCholesterol,omitempty"`
	HDLCholesterol   *float64 `json:"hdlCholesterol,omitempty"`
	OnBPTreatment    bool     `json:"onBPTreatment,omitempty"`
	IsSmoker         bool     `json:"isSmoker,omitempty"`

	// Regional settings
	Region Region `json:"region,omitempty"`
}

// ToEGFRParams extracts eGFR parameters from batch request.
func (b *BatchCalculatorRequest) ToEGFRParams() (*EGFRParams, error) {
	if b.SerumCreatinine == nil || b.AgeYears == nil {
		return nil, ErrMissingRequiredParams
	}
	return &EGFRParams{
		SerumCreatinine: *b.SerumCreatinine,
		AgeYears:        *b.AgeYears,
		Sex:             b.Sex,
	}, nil
}

// ToCrClParams extracts CrCl parameters from batch request.
func (b *BatchCalculatorRequest) ToCrClParams() (*CrClParams, error) {
	if b.SerumCreatinine == nil || b.AgeYears == nil || b.WeightKg == nil {
		return nil, ErrMissingRequiredParams
	}
	return &CrClParams{
		SerumCreatinine: *b.SerumCreatinine,
		AgeYears:        *b.AgeYears,
		Sex:             b.Sex,
		WeightKg:        *b.WeightKg,
	}, nil
}

// ToBMIParams extracts BMI parameters from batch request.
func (b *BatchCalculatorRequest) ToBMIParams() (*BMIParams, error) {
	if b.WeightKg == nil || b.HeightCm == nil {
		return nil, ErrMissingRequiredParams
	}
	return &BMIParams{
		WeightKg: *b.WeightKg,
		HeightCm: *b.HeightCm,
		Region:   b.Region,
	}, nil
}
