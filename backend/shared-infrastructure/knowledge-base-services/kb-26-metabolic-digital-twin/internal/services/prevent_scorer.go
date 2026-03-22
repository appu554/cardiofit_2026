package services

import (
	"fmt"
	"math"
	"time"

	"kb-26-metabolic-digital-twin/internal/models"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PREVENTInput contains the clinical inputs for the AHA PREVENT 10-year ASCVD risk equation.
type PREVENTInput struct {
	Age              int     // years
	Sex              string  // "M" or "F"
	SBP              float64 // mmHg
	TotalCholesterol float64 // mg/dL
	HDL              float64 // mg/dL
	EGFR             float64 // mL/min/1.73m²
	HbA1c            float64 // % (0 if not diabetic)
	BMI              float64 // kg/m² (optional, 0 if unknown)
	OnStatins        bool
	Smoker           bool
}

// PREVENTResult is the computed PREVENT risk output.
type PREVENTResult struct {
	TenYearRisk float64 `json:"ten_year_risk"` // 0.0 - 1.0
	RiskPercent float64 `json:"risk_percent"`  // 0.0 - 100.0
	Category    string  `json:"category"`      // LOW, BORDERLINE, INTERMEDIATE, HIGH
}

// Sex-specific Cox model coefficients for the AHA PREVENT 10-year ASCVD equation.
type preventCoefficients struct {
	lnAge       float64
	lnSBP       float64
	lnTC        float64
	lnHDL       float64
	lnEGFR      float64
	hba1c       float64 // applied when HbA1c >= 5.7 (diabetic threshold)
	smoker      float64
	statin      float64
	s0          float64 // baseline 10-year survival
	centerValue float64 // mean linear predictor for centering
}

var (
	maleCoefficients = preventCoefficients{
		lnAge:       0.3742,
		lnSBP:       0.1065,
		lnTC:        0.1369,
		lnHDL:       -0.3313,
		lnEGFR:      -0.1522,
		hba1c:       0.2039,
		smoker:      0.2614,
		statin:      -0.1297,
		s0:          0.9605,
		centerValue: 4.3218,
	}
	femaleCoefficients = preventCoefficients{
		lnAge:       0.4648,
		lnSBP:       0.1324,
		lnTC:        0.1034,
		lnHDL:       -0.3050,
		lnEGFR:      -0.1665,
		hba1c:       0.2359,
		smoker:      0.2834,
		statin:      -0.1049,
		s0:          0.9776,
		centerValue: 3.8415,
	}
)

// PREVENTScorer computes and persists AHA PREVENT 10-year CVD risk scores.
type PREVENTScorer struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewPREVENTScorer creates a new PREVENTScorer.
func NewPREVENTScorer(db *gorm.DB, logger *zap.Logger) *PREVENTScorer {
	return &PREVENTScorer{db: db, logger: logger}
}

// ComputePREVENT calculates the 10-year ASCVD risk using the AHA PREVENT equation.
// Guards against invalid inputs (zero/negative values for log terms) by returning zero risk.
func (s *PREVENTScorer) ComputePREVENT(input PREVENTInput) PREVENTResult {
	// Guard against log(0) or log(negative) — return zero risk for invalid inputs.
	if input.Age <= 0 || input.SBP <= 0 || input.TotalCholesterol <= 0 ||
		input.HDL <= 0 || input.EGFR <= 0 {
		return PREVENTResult{
			TenYearRisk: 0,
			RiskPercent: 0,
			Category:    models.PREVENTCategoryLow,
		}
	}

	// Select sex-specific coefficients
	coeff := maleCoefficients
	if input.Sex == "F" {
		coeff = femaleCoefficients
	}

	// Compute linear predictor
	lp := coeff.lnAge * math.Log(float64(input.Age))
	lp += coeff.lnSBP * math.Log(input.SBP)
	lp += coeff.lnTC * math.Log(input.TotalCholesterol)
	lp += coeff.lnHDL * math.Log(input.HDL)
	lp += coeff.lnEGFR * math.Log(input.EGFR)

	// Diabetic model: apply HbA1c coefficient when HbA1c >= 5.7%
	if input.HbA1c >= 5.7 {
		lp += coeff.hba1c * input.HbA1c
	}

	// Optional binary predictors
	if input.Smoker {
		lp += coeff.smoker
	}
	if input.OnStatins {
		lp += coeff.statin
	}

	// PREVENT formula: risk = 1 - S0^exp(lp - centering)
	exponent := math.Exp(lp - coeff.centerValue)
	risk := 1.0 - math.Pow(coeff.s0, exponent)

	// Clamp to [0, 1]
	if risk < 0 {
		risk = 0
	}
	if risk > 1 {
		risk = 1
	}

	pct := risk * 100.0

	return PREVENTResult{
		TenYearRisk: risk,
		RiskPercent: pct,
		Category:    categorizePREVENT(pct),
	}
}

// categorizePREVENT assigns a risk category based on the 10-year risk percentage.
// AHA thresholds: LOW (<5%), BORDERLINE (5-7.5%), INTERMEDIATE (7.5-20%), HIGH (>=20%)
func categorizePREVENT(pct float64) string {
	switch {
	case pct < 5:
		return models.PREVENTCategoryLow
	case pct < 7.5:
		return models.PREVENTCategoryBorderline
	case pct < 20:
		return models.PREVENTCategoryIntermediate
	default:
		return models.PREVENTCategoryHigh
	}
}

// PersistScore stores a PREVENT score in the database.
func (s *PREVENTScorer) PersistScore(patientID uuid.UUID, input PREVENTInput, result PREVENTResult, twinStateID *uuid.UUID) (*models.PREVENTScore, error) {
	if s.db == nil {
		return nil, nil
	}

	score := &models.PREVENTScore{
		PatientID:   patientID,
		TenYearRisk: result.TenYearRisk,
		RiskPercent: result.RiskPercent,
		Category:    result.Category,
		InputAge:    input.Age,
		InputSBP:    input.SBP,
		InputTC:     input.TotalCholesterol,
		InputHDL:    input.HDL,
		InputEGFR:   input.EGFR,
		InputHbA1c:  input.HbA1c,
		TwinStateID: twinStateID,
		ComputedAt:  time.Now().UTC(),
	}

	if err := s.db.Create(score).Error; err != nil {
		return nil, err
	}

	return score, nil
}

// GetLatest returns the most recent PREVENT score for a patient.
func (s *PREVENTScorer) GetLatest(patientID uuid.UUID) (*models.PREVENTScore, error) {
	if s.db == nil {
		return nil, fmt.Errorf("no database")
	}
	var score models.PREVENTScore
	result := s.db.Where("patient_id = ?", patientID).
		Order("computed_at DESC").
		First(&score)
	if result.Error != nil {
		return nil, result.Error
	}
	return &score, nil
}

// GetHistory returns recent PREVENT scores for a patient.
func (s *PREVENTScorer) GetHistory(patientID uuid.UUID, limit int) ([]models.PREVENTScore, error) {
	if s.db == nil {
		return nil, nil
	}
	var scores []models.PREVENTScore
	result := s.db.Where("patient_id = ?", patientID).
		Order("computed_at DESC").
		Limit(limit).
		Find(&scores)
	return scores, result.Error
}

// TwinToPREVENTInput maps a TwinState to PREVENTInput for event-driven recompute.
// Age and Sex require patient demographics — hardcoded defaults for now (same pattern as eGFR derivation).
func TwinToPREVENTInput(twin *models.TwinState) PREVENTInput {
	input := PREVENTInput{
		// TODO: Fetch real demographics from KB-20 patient profile when client is integrated.
		Age: 55,
		Sex: "M",
	}

	if twin.SBP14dMean != nil {
		input.SBP = *twin.SBP14dMean
	}
	if twin.TotalCholesterol != nil {
		input.TotalCholesterol = *twin.TotalCholesterol
	}
	if twin.HDL != nil {
		input.HDL = *twin.HDL
	}
	if twin.EGFR != nil {
		input.EGFR = *twin.EGFR
	}
	if twin.HbA1c != nil {
		input.HbA1c = *twin.HbA1c
	}
	if twin.BMI != nil {
		input.BMI = *twin.BMI
	}

	return input
}
