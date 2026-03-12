package services

import (
	"time"

	"kb-21-behavioral-intelligence/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// HypoRiskService detects behavioral signals that elevate hypoglycemia risk.
// Implements Finding F-03 (Gap 4): KB-21's role is limited to providing the
// BEHAVIORAL signals that INCREASE hypoglycemia risk. The clinical safety
// protocol (override titration, alert physician) belongs in KB-4 (Patient Safety),
// and coordination belongs in KB-19 (Protocol Orchestrator).
//
// KB-21 publishes HYPO_RISK_ELEVATED events. Consumer: KB-19 → KB-4 → V-MCU override.
type HypoRiskService struct {
	db           *gorm.DB
	logger       *zap.Logger
	publisher    EventPublisher
	safetyClient *SafetyClient // G-03: KB-23 direct fast-path for HYPO_RISK
}

// EventPublisher interface for publishing events to the bus.
type EventPublisher interface {
	PublishHypoRiskElevated(event models.HypoRiskEvent) error
}

func NewHypoRiskService(db *gorm.DB, logger *zap.Logger, publisher EventPublisher, safetyClient *SafetyClient) *HypoRiskService {
	return &HypoRiskService{
		db:           db,
		logger:       logger,
		publisher:    publisher,
		safetyClient: safetyClient,
	}
}

// EvaluateHypoRisk checks a patient for behavioral hypoglycemia risk factors.
// Called after each interaction event that could affect hypo risk:
//   - Meal skip detection (evening_meal_confirmed = false + basal insulin active)
//   - Erratic adherence pattern (SPORADIC phenotype + insulin)
//   - Fasting detection (festival calendar + self-report)
func (s *HypoRiskService) EvaluateHypoRisk(patientID string) (*models.HypoRiskEvent, error) {
	var riskFactors []models.HypoRiskFactor
	var affectedMeds []string

	// 1. Meal skip detection
	if mealSkipRisk, meds := s.checkMealSkipRisk(patientID); mealSkipRisk {
		riskFactors = append(riskFactors, models.HypoFactorMealSkip)
		affectedMeds = append(affectedMeds, meds...)
	}

	// 2. Erratic adherence pattern
	if erraticRisk, meds := s.checkErraticAdherenceRisk(patientID); erraticRisk {
		riskFactors = append(riskFactors, models.HypoFactorErraticAdherence)
		affectedMeds = append(affectedMeds, meds...)
	}

	// 3. Fasting detection
	if fastingRisk, meds := s.checkFastingRisk(patientID); fastingRisk {
		riskFactors = append(riskFactors, models.HypoFactorFasting)
		affectedMeds = append(affectedMeds, meds...)
	}

	if len(riskFactors) == 0 {
		return nil, nil // No elevated risk
	}

	// Determine risk level
	riskLevel := models.HypoRiskModerate
	if len(riskFactors) >= 2 {
		riskLevel = models.HypoRiskHigh
	}

	// Check for HIGH-risk combinations
	hasMealSkip := false
	hasErratic := false
	for _, f := range riskFactors {
		if f == models.HypoFactorMealSkip {
			hasMealSkip = true
		}
		if f == models.HypoFactorErraticAdherence {
			hasErratic = true
		}
	}
	if hasMealSkip && hasErratic {
		riskLevel = models.HypoRiskHigh
	}

	event := &models.HypoRiskEvent{
		PatientID:           patientID,
		RiskFactors:         riskFactors,
		RiskLevel:           riskLevel,
		AffectedMedications: uniqueStrings(affectedMeds),
		Timestamp:           time.Now().UTC(),
	}

	// G-03: Alert KB-23 directly via fast-path FIRST — this is the primary safety channel.
	// KB-23 produces PAUSE (not HALT) because behavioral risk is probabilistic.
	if s.safetyClient != nil {
		if err := s.safetyClient.AlertHypoRisk(*event); err != nil {
			s.logger.Error("G-03: failed to alert KB-23 for hypo risk — falling back to event bus only",
				zap.String("patient_id", patientID),
				zap.String("risk_level", string(riskLevel)),
				zap.Error(err),
			)
		}
	}

	// Publish to event bus as SECONDARY notification for KB-19 / KB-4 consumption.
	// This ensures KB-19 can still coordinate downstream protocols even after
	// KB-23 has already produced the DecisionCard.
	if s.publisher != nil {
		if err := s.publisher.PublishHypoRiskElevated(*event); err != nil {
			s.logger.Error("failed to publish HYPO_RISK_ELEVATED to event bus",
				zap.String("patient_id", patientID),
				zap.Error(err),
			)
		}
	}

	s.logger.Warn("HYPO_RISK_ELEVATED detected",
		zap.String("patient_id", patientID),
		zap.String("risk_level", string(riskLevel)),
		zap.Int("factor_count", len(riskFactors)),
	)

	return event, nil
}

// checkMealSkipRisk detects: evening_meal_confirmed=false + active insulin/sulfonylurea.
func (s *HypoRiskService) checkMealSkipRisk(patientID string) (bool, []string) {
	// Check today's dietary signal
	today := time.Now().UTC().Truncate(24 * time.Hour)
	var signal models.DietarySignal
	err := s.db.Where("patient_id = ? AND date = ?", patientID, today).
		First(&signal).Error

	if err != nil {
		// Check yesterday if today not yet reported
		yesterday := today.AddDate(0, 0, -1)
		err = s.db.Where("patient_id = ? AND date = ?", patientID, yesterday).
			First(&signal).Error
		if err != nil {
			return false, nil
		}
	}

	if signal.EveningMealConfirmed {
		return false, nil // Meal confirmed, no risk
	}

	// Check if patient has active insulin or sulfonylurea adherence records
	var insulinMeds []models.AdherenceState
	s.db.Where("patient_id = ? AND drug_class IN ('INSULIN', 'SULFONYLUREA', 'BASAL_INSULIN')",
		patientID).Find(&insulinMeds)

	if len(insulinMeds) == 0 {
		return false, nil
	}

	var meds []string
	for _, m := range insulinMeds {
		meds = append(meds, m.DrugClass)
	}
	return true, meds
}

// checkErraticAdherenceRisk detects: SPORADIC phenotype + insulin, where the patient
// takes insulin some days and skips others. On adherent days, the glucose drop may be
// larger than expected if the dose was titrated up during non-adherent periods.
func (s *HypoRiskService) checkErraticAdherenceRisk(patientID string) (bool, []string) {
	var profile models.EngagementProfile
	if err := s.db.Where("patient_id = ?", patientID).First(&profile).Error; err != nil {
		return false, nil
	}

	if profile.Phenotype != models.PhenotypeSporadic {
		return false, nil
	}

	// Check for insulin adherence with high variance
	var insulinAdherence []models.AdherenceState
	s.db.Where("patient_id = ? AND drug_class IN ('INSULIN', 'BASAL_INSULIN') AND adherence_score BETWEEN 0.30 AND 0.70",
		patientID).Find(&insulinAdherence)

	if len(insulinAdherence) == 0 {
		return false, nil
	}

	var meds []string
	for _, m := range insulinAdherence {
		meds = append(meds, m.DrugClass)
	}
	return true, meds
}

// checkFastingRisk detects: fasting_today=true + active insulin/sulfonylurea.
func (s *HypoRiskService) checkFastingRisk(patientID string) (bool, []string) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	var signal models.DietarySignal
	err := s.db.Where("patient_id = ? AND date = ? AND fasting_today = true", patientID, today).
		First(&signal).Error

	if err != nil {
		return false, nil
	}

	var insulinMeds []models.AdherenceState
	s.db.Where("patient_id = ? AND drug_class IN ('INSULIN', 'SULFONYLUREA', 'BASAL_INSULIN')",
		patientID).Find(&insulinMeds)

	if len(insulinMeds) == 0 {
		return false, nil
	}

	var meds []string
	for _, m := range insulinMeds {
		meds = append(meds, m.DrugClass)
	}
	return true, meds
}

func uniqueStrings(input []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range input {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
