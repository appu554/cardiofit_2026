// Package calculator provides the service orchestrator for KB-8 Calculator Service.
//
// The Service coordinates all clinical calculators and provides a unified
// interface for the API layer. Each calculator implements the ATOMIC pattern:
// pure mathematical functions with no external dependencies during calculation.
package calculator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"kb-8-calculator-service/internal/models"
)

// Service orchestrates all clinical calculators.
// Thread-safe for concurrent use.
type Service struct {
	logger *zap.Logger

	// P0 Calculators (Critical for dosing)
	egfrCalc *EGFRCalculator
	crclCalc *CrClCalculator
	bmiCalc  *BMICalculator

	// P1 Calculators (Clinical scores)
	sofaCalc        *SOFACalculator
	qsofaCalc       *QSOFACalculator
	cha2ds2vascCalc *CHA2DS2VAScCalculator
	hasBledCalc     *HASBLEDCalculator
	ascvdCalc       *ASCVDCalculator

	// Metrics
	mu           sync.RWMutex
	calculations map[models.CalculatorType]int64
}

// NewService creates a new calculator service with all available calculators.
func NewService(logger *zap.Logger) *Service {
	return &Service{
		logger:       logger,
		// P0 Calculators
		egfrCalc:     NewEGFRCalculator(),
		crclCalc:     NewCrClCalculator(),
		bmiCalc:      NewBMICalculator(),
		// P1 Calculators
		sofaCalc:        NewSOFACalculator(),
		qsofaCalc:       NewQSOFACalculator(),
		cha2ds2vascCalc: NewCHA2DS2VAScCalculator(),
		hasBledCalc:     NewHASBLEDCalculator(),
		ascvdCalc:       NewASCVDCalculator(),
		calculations:    make(map[models.CalculatorType]int64),
	}
}

// CalculateEGFR computes estimated glomerular filtration rate using CKD-EPI 2021.
func (s *Service) CalculateEGFR(ctx context.Context, params *models.EGFRParams) (*models.EGFRResult, error) {
	start := time.Now()
	defer func() {
		s.recordCalculation(models.CalculatorEGFR, time.Since(start))
	}()

	result, err := s.egfrCalc.Calculate(ctx, params)
	if err != nil {
		s.logger.Warn("eGFR calculation failed",
			zap.Error(err),
			zap.Float64("creatinine", params.SerumCreatinine),
			zap.Int("age", params.AgeYears),
		)
		return nil, err
	}

	s.logger.Debug("eGFR calculated",
		zap.Float64("value", result.Value),
		zap.String("stage", string(result.CKDStage)),
		zap.Duration("latency", time.Since(start)),
	)

	return result, nil
}

// CalculateCrCl computes creatinine clearance using Cockcroft-Gault equation.
func (s *Service) CalculateCrCl(ctx context.Context, params *models.CrClParams) (*models.CrClResult, error) {
	start := time.Now()
	defer func() {
		s.recordCalculation(models.CalculatorCrCl, time.Since(start))
	}()

	result, err := s.crclCalc.Calculate(ctx, params)
	if err != nil {
		s.logger.Warn("CrCl calculation failed",
			zap.Error(err),
			zap.Float64("creatinine", params.SerumCreatinine),
			zap.Int("age", params.AgeYears),
			zap.Float64("weight", params.WeightKg),
		)
		return nil, err
	}

	s.logger.Debug("CrCl calculated",
		zap.Float64("value", result.Value),
		zap.String("renalFunction", string(result.RenalFunction)),
		zap.Duration("latency", time.Since(start)),
	)

	return result, nil
}

// CalculateBMI computes body mass index with regional category adjustments.
func (s *Service) CalculateBMI(ctx context.Context, params *models.BMIParams) (*models.BMIResult, error) {
	start := time.Now()
	defer func() {
		s.recordCalculation(models.CalculatorBMI, time.Since(start))
	}()

	result, err := s.bmiCalc.Calculate(ctx, params)
	if err != nil {
		s.logger.Warn("BMI calculation failed",
			zap.Error(err),
			zap.Float64("weight", params.WeightKg),
			zap.Float64("height", params.HeightCm),
		)
		return nil, err
	}

	s.logger.Debug("BMI calculated",
		zap.Float64("value", result.Value),
		zap.String("categoryWestern", result.CategoryWestern.String()),
		zap.String("categoryAsian", result.CategoryAsian.String()),
		zap.Duration("latency", time.Since(start)),
	)

	return result, nil
}

// CalculateSOFA computes Sequential Organ Failure Assessment score.
func (s *Service) CalculateSOFA(ctx context.Context, params *models.SOFAParams) (*models.SOFAResult, error) {
	start := time.Now()
	defer func() {
		s.recordCalculation(models.CalculatorSOFA, time.Since(start))
	}()

	result, err := s.sofaCalc.Calculate(ctx, params)
	if err != nil {
		s.logger.Warn("SOFA calculation failed", zap.Error(err))
		return nil, err
	}

	s.logger.Debug("SOFA calculated",
		zap.Int("total", result.Total),
		zap.String("riskLevel", string(result.RiskLevel)),
		zap.Duration("latency", time.Since(start)),
	)

	return result, nil
}

// CalculateQSOFA computes quick SOFA score for sepsis screening.
func (s *Service) CalculateQSOFA(ctx context.Context, params *models.QSOFAParams) (*models.QSOFAResult, error) {
	start := time.Now()
	defer func() {
		s.recordCalculation(models.CalculatorQSOFA, time.Since(start))
	}()

	result, err := s.qsofaCalc.Calculate(ctx, params)
	if err != nil {
		s.logger.Warn("qSOFA calculation failed", zap.Error(err))
		return nil, err
	}

	s.logger.Debug("qSOFA calculated",
		zap.Int("total", result.Total),
		zap.Bool("positive", result.Positive),
		zap.Duration("latency", time.Since(start)),
	)

	return result, nil
}

// CalculateCHA2DS2VASc computes CHA₂DS₂-VASc stroke risk score.
func (s *Service) CalculateCHA2DS2VASc(ctx context.Context, params *models.CHA2DS2VAScParams) (*models.CHA2DS2VAScResult, error) {
	start := time.Now()
	defer func() {
		s.recordCalculation(models.CalculatorCHA2DS2VASc, time.Since(start))
	}()

	result, err := s.cha2ds2vascCalc.Calculate(ctx, params)
	if err != nil {
		s.logger.Warn("CHA2DS2-VASc calculation failed",
			zap.Error(err),
			zap.Int("age", params.AgeYears),
		)
		return nil, err
	}

	s.logger.Debug("CHA2DS2-VASc calculated",
		zap.Int("total", result.Total),
		zap.Bool("anticoagRecommended", result.AnticoagulationRecommended),
		zap.Duration("latency", time.Since(start)),
	)

	return result, nil
}

// CalculateHASBLED computes HAS-BLED bleeding risk score.
func (s *Service) CalculateHASBLED(ctx context.Context, params *models.HASBLEDParams) (*models.HASBLEDResult, error) {
	start := time.Now()
	defer func() {
		s.recordCalculation(models.CalculatorHASBLED, time.Since(start))
	}()

	result, err := s.hasBledCalc.Calculate(ctx, params)
	if err != nil {
		s.logger.Warn("HAS-BLED calculation failed", zap.Error(err))
		return nil, err
	}

	s.logger.Debug("HAS-BLED calculated",
		zap.Int("total", result.Total),
		zap.Bool("highRisk", result.HighRisk),
		zap.Duration("latency", time.Since(start)),
	)

	return result, nil
}

// CalculateASCVD computes 10-year ASCVD risk using Pooled Cohort Equations.
func (s *Service) CalculateASCVD(ctx context.Context, params *models.ASCVDParams) (*models.ASCVDResult, error) {
	start := time.Now()
	defer func() {
		s.recordCalculation(models.CalculatorASCVD, time.Since(start))
	}()

	result, err := s.ascvdCalc.Calculate(ctx, params)
	if err != nil {
		s.logger.Warn("ASCVD calculation failed",
			zap.Error(err),
			zap.Int("age", params.AgeYears),
		)
		return nil, err
	}

	s.logger.Debug("ASCVD calculated",
		zap.Float64("riskPercent", result.RiskPercent),
		zap.String("category", string(result.RiskCategory)),
		zap.Duration("latency", time.Since(start)),
	)

	return result, nil
}

// CalculateBatch processes multiple calculations in a single request.
// Returns results for each requested calculator type.
func (s *Service) CalculateBatch(ctx context.Context, req *models.BatchCalculatorRequest) (*models.SimpleBatchResponse, error) {
	start := time.Now()
	response := &models.SimpleBatchResponse{
		PatientID:    req.PatientID,
		Results:      make(map[models.CalculatorType]interface{}),
		Errors:       make(map[models.CalculatorType]string),
		CalculatedAt: time.Now().UTC(),
	}

	for _, calcType := range req.Calculators {
		switch calcType {
		case models.CalculatorEGFR:
			params, err := req.ToEGFRParams()
			if err != nil {
				response.Errors[calcType] = err.Error()
				continue
			}
			result, err := s.CalculateEGFR(ctx, params)
			if err != nil {
				response.Errors[calcType] = err.Error()
			} else {
				response.Results[calcType] = result
			}

		case models.CalculatorCrCl:
			params, err := req.ToCrClParams()
			if err != nil {
				response.Errors[calcType] = err.Error()
				continue
			}
			result, err := s.CalculateCrCl(ctx, params)
			if err != nil {
				response.Errors[calcType] = err.Error()
			} else {
				response.Results[calcType] = result
			}

		case models.CalculatorBMI:
			params, err := req.ToBMIParams()
			if err != nil {
				response.Errors[calcType] = err.Error()
				continue
			}
			result, err := s.CalculateBMI(ctx, params)
			if err != nil {
				response.Errors[calcType] = err.Error()
			} else {
				response.Results[calcType] = result
			}

		default:
			response.Errors[calcType] = fmt.Sprintf("calculator %s not implemented", calcType)
		}
	}

	response.TotalLatencyMs = float64(time.Since(start).Microseconds()) / 1000.0

	s.logger.Info("batch calculation completed",
		zap.String("patientId", req.PatientID),
		zap.Int("requested", len(req.Calculators)),
		zap.Int("successful", len(response.Results)),
		zap.Int("failed", len(response.Errors)),
		zap.Float64("latencyMs", response.TotalLatencyMs),
	)

	return response, nil
}

// GetAvailableCalculators returns the list of available calculator types.
func (s *Service) GetAvailableCalculators() []models.CalculatorInfo {
	return []models.CalculatorInfo{
		// P0 Calculators
		{
			Type:           models.CalculatorEGFR,
			Name:           s.egfrCalc.Name(),
			Version:        s.egfrCalc.Version(),
			Reference:      s.egfrCalc.Reference(),
			Description:    "Estimated glomerular filtration rate using CKD-EPI 2021 race-free equation",
			RequiredParams: []string{"serumCreatinine", "ageYears", "sex"},
		},
		{
			Type:           models.CalculatorCrCl,
			Name:           s.crclCalc.Name(),
			Version:        s.crclCalc.Version(),
			Reference:      s.crclCalc.Reference(),
			Description:    "Creatinine clearance using Cockcroft-Gault equation with actual body weight",
			RequiredParams: []string{"serumCreatinine", "ageYears", "sex", "weightKg"},
		},
		{
			Type:           models.CalculatorBMI,
			Name:           s.bmiCalc.Name(),
			Version:        s.bmiCalc.Version(),
			Reference:      s.bmiCalc.Reference(),
			Description:    "Body mass index with Western and Asian (WHO Asia-Pacific) categorization",
			RequiredParams: []string{"weightKg", "heightCm"},
			OptionalParams: []string{"region", "ethnicity"},
		},
		// P1 Calculators
		{
			Type:           models.CalculatorSOFA,
			Name:           "SOFA Score",
			Version:        "SOFA-1996-Updated",
			Reference:      "Vincent JL, et al. Intensive Care Med. 1996;22(7):707-10",
			Description:    "Sequential Organ Failure Assessment for ICU mortality prediction",
			RequiredParams: []string{},
			OptionalParams: []string{"pao2fio2Ratio", "platelets", "bilirubin", "map", "glasgowComaScale", "creatinine", "urineOutput"},
		},
		{
			Type:           models.CalculatorQSOFA,
			Name:           "qSOFA Score",
			Version:        "qSOFA-Sepsis3-2016",
			Reference:      "Seymour CW, et al. JAMA. 2016;315(8):762-774",
			Description:    "Quick SOFA for bedside sepsis screening (no lab tests required)",
			RequiredParams: []string{},
			OptionalParams: []string{"respiratoryRate", "systolicBP", "alteredMentation", "glasgowComaScale"},
		},
		{
			Type:           models.CalculatorCHA2DS2VASc,
			Name:           "CHA₂DS₂-VASc Score",
			Version:        "CHA2DS2-VASc-2010",
			Reference:      "Lip GYH, et al. Chest. 2010;137(2):263-272",
			Description:    "Stroke risk assessment for atrial fibrillation anticoagulation decisions",
			RequiredParams: []string{"ageYears", "sex"},
			OptionalParams: []string{"hasCongestiveHeartFailure", "hasHypertension", "hasDiabetes", "hasStrokeTIA", "hasVascularDisease"},
		},
		{
			Type:           models.CalculatorHASBLED,
			Name:           "HAS-BLED Score",
			Version:        "HAS-BLED-2010",
			Reference:      "Pisters R, et al. Chest. 2010;138(5):1093-1100",
			Description:    "Major bleeding risk assessment for anticoagulation decisions",
			RequiredParams: []string{},
			OptionalParams: []string{"hasUncontrolledHypertension", "hasAbnormalRenalFunction", "hasAbnormalLiverFunction", "hasStrokeHistory", "hasBleedingHistory", "hasLabileINR", "ageYears", "takingAntiplateletOrNSAID", "excessiveAlcohol"},
		},
		{
			Type:           models.CalculatorASCVD,
			Name:           "ASCVD 10-Year Risk",
			Version:        "PCE-2013-Revised-2018",
			Reference:      "Goff DC Jr, et al. Circulation. 2014;129(25 Suppl 2):S49-73",
			Description:    "10-year atherosclerotic cardiovascular disease risk using Pooled Cohort Equations",
			RequiredParams: []string{"ageYears", "sex", "totalCholesterol", "hdlCholesterol", "systolicBP"},
			OptionalParams: []string{"race", "onBPTreatment", "hasDiabetes", "isSmoker"},
		},
	}
}

// GetMetrics returns calculation metrics for monitoring.
func (s *Service) GetMetrics() map[models.CalculatorType]int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metrics := make(map[models.CalculatorType]int64)
	for k, v := range s.calculations {
		metrics[k] = v
	}
	return metrics
}

// recordCalculation tracks calculation metrics.
func (s *Service) recordCalculation(calcType models.CalculatorType, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calculations[calcType]++
}
