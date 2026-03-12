// Package risk provides risk stratification engine for KB-11 Population Health.
package risk

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/cardiofit/kb-11-population-health/internal/clients"
	"github.com/cardiofit/kb-11-population-health/internal/database"
	"github.com/cardiofit/kb-11-population-health/internal/models"
)

// Engine provides risk calculation functionality with governance integration.
// CRITICAL: All calculations MUST be deterministic and governed by KB-18.
type Engine struct {
	repo         *database.ProjectionRepository
	kb18Client   *clients.KB18Client
	models       map[models.RiskModelType]*ModelConfig
	logger       *logrus.Entry
	mu           sync.RWMutex
	maxConcurrent int
}

// NewEngine creates a new risk engine.
func NewEngine(
	repo *database.ProjectionRepository,
	kb18Client *clients.KB18Client,
	maxConcurrent int,
	logger *logrus.Entry,
) *Engine {
	engine := &Engine{
		repo:          repo,
		kb18Client:    kb18Client,
		models:        make(map[models.RiskModelType]*ModelConfig),
		logger:        logger.WithField("component", "risk-engine"),
		maxConcurrent: maxConcurrent,
	}

	// Register default models
	engine.RegisterModel(DefaultHospitalizationModel())
	engine.RegisterModel(DefaultReadmissionModel())
	engine.RegisterModel(DefaultEDUtilizationModel())

	return engine
}

// RegisterModel registers a risk model configuration.
func (e *Engine) RegisterModel(config *ModelConfig) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.models[config.Name] = config
	e.logger.WithFields(logrus.Fields{
		"model":   config.Name,
		"version": config.Version,
	}).Info("Risk model registered")
}

// GetModel retrieves a model configuration.
func (e *Engine) GetModel(modelType models.RiskModelType) (*ModelConfig, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	model, ok := e.models[modelType]
	if !ok {
		return nil, fmt.Errorf("model not found: %s", modelType)
	}
	return model, nil
}

// ListModels returns all registered model configurations.
func (e *Engine) ListModels() []*ModelConfig {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]*ModelConfig, 0, len(e.models))
	for _, m := range e.models {
		result = append(result, m)
	}
	return result
}

// CalculateRisk calculates risk for a patient using the specified model.
// GOVERNANCE: Emits event to KB-18, includes determinism hashes.
func (e *Engine) CalculateRisk(ctx context.Context, features *RiskFeatures, modelType models.RiskModelType) (*RiskResult, error) {
	model, err := e.GetModel(modelType)
	if err != nil {
		return nil, err
	}

	// Validate model with KB-18 (if available)
	if e.kb18Client != nil {
		validation, err := e.kb18Client.ValidateModel(ctx, string(model.Name), model.Version)
		if err != nil {
			e.logger.WithError(err).Warn("KB-18 model validation failed")
		} else if !validation.ModelApproved {
			return nil, fmt.Errorf("model %s v%s is not approved by KB-18", model.Name, model.Version)
		}
	}

	// Calculate input hash BEFORE calculation (determinism guarantee)
	inputHash := features.Hash()

	// Perform the calculation
	result, err := e.calculateWithModel(features, model)
	if err != nil {
		return nil, err
	}

	result.InputHash = inputHash
	result.CalculationHash = result.Hash()

	// Emit governance event to KB-18
	if e.kb18Client != nil {
		eventResp, err := e.kb18Client.EmitRiskCalculationEvent(ctx, &clients.GovernanceEvent{
			SubjectID:    features.PatientFHIRID,
			ModelName:    string(model.Name),
			ModelVersion: model.Version,
			InputHash:    result.InputHash,
			OutputHash:   result.CalculationHash,
			AuditMetadata: map[string]interface{}{
				"score":     result.Score,
				"risk_tier": result.RiskTier,
			},
		})
		if err != nil {
			e.logger.WithError(err).Warn("Failed to emit governance event")
		} else {
			// Store the governance event ID for audit trail
			govEventID := eventResp.EventID
			e.logger.WithField("governance_event_id", govEventID).Debug("Governance event emitted")
		}
	}

	// Save the assessment to database
	assessment := &models.RiskAssessment{
		ID:                  uuid.New(),
		PatientFHIRID:       features.PatientFHIRID,
		ModelName:           string(model.Name),
		ModelVersion:        model.Version,
		Score:               result.Score,
		RiskTier:            result.RiskTier,
		ContributingFactors: result.ContributingFactors,
		InputHash:           result.InputHash,
		CalculationHash:     result.CalculationHash,
		CalculatedAt:        result.CalculatedAt,
		ValidUntil:          &result.ValidUntil,
	}

	if err := e.repo.SaveRiskAssessment(ctx, assessment); err != nil {
		e.logger.WithError(err).Error("Failed to save risk assessment")
		// Don't fail the calculation, just log
	}

	// Update the patient projection with the new risk tier
	if err := e.repo.UpdateRiskTier(ctx, features.PatientFHIRID, result.RiskTier, result.Score); err != nil {
		e.logger.WithError(err).Error("Failed to update patient risk tier")
	}

	e.logger.WithFields(logrus.Fields{
		"patient":    features.PatientFHIRID,
		"model":      model.Name,
		"score":      result.Score,
		"tier":       result.RiskTier,
		"input_hash": result.InputHash[:16] + "...",
	}).Info("Risk calculated")

	return result, nil
}

// BatchCalculateRisk calculates risk for multiple patients.
func (e *Engine) BatchCalculateRisk(ctx context.Context, featuresList []*RiskFeatures, modelType models.RiskModelType) ([]*RiskResult, error) {
	model, err := e.GetModel(modelType)
	if err != nil {
		return nil, err
	}

	batchID := uuid.New()
	results := make([]*RiskResult, len(featuresList))
	resultsMu := sync.Mutex{}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(e.maxConcurrent)

	for i, features := range featuresList {
		i, features := i, features // Capture for closure
		g.Go(func() error {
			result, err := e.CalculateRisk(gctx, features, modelType)
			if err != nil {
				e.logger.WithError(err).WithField("patient", features.PatientFHIRID).Warn("Batch calculation failed for patient")
				return nil // Don't fail the entire batch
			}

			resultsMu.Lock()
			results[i] = result
			resultsMu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Emit batch governance event
	if e.kb18Client != nil {
		successCount := 0
		for _, r := range results {
			if r != nil {
				successCount++
			}
		}
		e.kb18Client.EmitBatchCalculationEvent(ctx, batchID, successCount, string(model.Name), model.Version)
	}

	// Filter out nil results
	filtered := make([]*RiskResult, 0, len(results))
	for _, r := range results {
		if r != nil {
			filtered = append(filtered, r)
		}
	}

	return filtered, nil
}

// calculateWithModel performs the actual risk calculation.
// This is a deterministic function: same input MUST produce same output.
func (e *Engine) calculateWithModel(features *RiskFeatures, model *ModelConfig) (*RiskResult, error) {
	now := time.Now()
	factors := make(map[string]float64)
	totalScore := 0.0

	// Age factors
	if features.Age >= 65 {
		factors["age_over_65"] = model.Weights["age_over_65"]
		totalScore += factors["age_over_65"]
	}
	if features.Age >= 80 {
		factors["age_over_80"] = model.Weights["age_over_80"]
		totalScore += factors["age_over_80"]
	}

	// Chronic conditions
	chronicCount := 0
	for _, cond := range features.Conditions {
		if cond.IsActive && isChronicCondition(cond.Code) {
			chronicCount++
		}
	}
	if chronicCount > 0 {
		// Scale by number of chronic conditions (max contribution at 5+)
		chronicFactor := float64(min(chronicCount, 5)) / 5.0 * model.Weights["chronic_conditions"]
		factors["chronic_conditions"] = chronicFactor
		totalScore += chronicFactor
	}

	// Recent hospitalizations
	recentHospitalizations := 0
	thirtyDaysAgo := now.AddDate(0, 0, -30)
	ninetyDaysAgo := now.AddDate(0, 0, -90)

	for _, enc := range features.Encounters {
		if enc.Type == "inpatient" {
			if enc.Date.After(thirtyDaysAgo) {
				recentHospitalizations++
			}
		}
	}
	if recentHospitalizations > 0 {
		hospFactor := float64(min(recentHospitalizations, 3)) / 3.0 * model.Weights["recent_hospitalization"]
		factors["recent_hospitalization"] = hospFactor
		totalScore += hospFactor
	}

	// ED visits in 90 days
	edVisits := 0
	for _, enc := range features.Encounters {
		if enc.Type == "emergency" && enc.Date.After(ninetyDaysAgo) {
			edVisits++
		}
	}
	if edVisits > 0 {
		edFactor := float64(min(edVisits, 5)) / 5.0 * model.Weights["ed_visits_90d"]
		factors["ed_visits_90d"] = edFactor
		totalScore += edFactor
	}

	// High-risk medications
	highRiskMeds := 0
	for _, med := range features.Medications {
		if med.IsActive && med.HighRisk {
			highRiskMeds++
		}
	}
	if highRiskMeds > 0 {
		medFactor := float64(min(highRiskMeds, 3)) / 3.0 * model.Weights["high_risk_medications"]
		factors["high_risk_medications"] = medFactor
		totalScore += medFactor
	}

	// Abnormal labs
	abnormalLabs := 0
	for _, lab := range features.LabValues {
		if lab.IsAbnormal {
			abnormalLabs++
		}
	}
	if abnormalLabs > 0 {
		labFactor := float64(min(abnormalLabs, 5)) / 5.0 * model.Weights["abnormal_labs"]
		factors["abnormal_labs"] = labFactor
		totalScore += labFactor
	}

	// Normalize score
	normalizedScore := NormalizeScore(totalScore)

	// Check for rising risk
	isRising, risingRate := CalculateRisingRisk(normalizedScore, features.PreviousScores, model.Thresholds.Rising)

	// Determine risk tier
	riskTier := DetermineRiskTier(normalizedScore, model.Thresholds, isRising)

	// Calculate confidence (based on data completeness)
	confidence := calculateConfidence(features)

	return &RiskResult{
		PatientFHIRID:       features.PatientFHIRID,
		ModelName:           string(model.Name),
		ModelVersion:        model.Version,
		Score:               normalizedScore,
		RiskTier:            riskTier,
		Confidence:          confidence,
		ContributingFactors: factors,
		CalculatedAt:        now,
		ValidUntil:          now.AddDate(0, 0, model.ValidDays),
		IsRising:            isRising,
		RisingRate:          risingRate,
	}, nil
}

// isChronicCondition checks if a condition code represents a chronic condition.
func isChronicCondition(code string) bool {
	// Common chronic condition codes (simplified for demo)
	chronicCodes := map[string]bool{
		// Diabetes
		"E11":   true, // Type 2 diabetes
		"E10":   true, // Type 1 diabetes
		"44054006": true, // Diabetes SNOMED
		// Heart disease
		"I25":   true, // Chronic ischemic heart disease
		"I50":   true, // Heart failure
		"84114007": true, // Heart failure SNOMED
		// COPD
		"J44":   true, // COPD
		"13645005": true, // COPD SNOMED
		// Hypertension
		"I10":   true, // Essential hypertension
		"38341003": true, // Hypertension SNOMED
		// CKD
		"N18":   true, // Chronic kidney disease
		"709044004": true, // CKD SNOMED
	}
	return chronicCodes[code]
}

// calculateConfidence calculates data completeness/confidence score.
func calculateConfidence(features *RiskFeatures) float64 {
	score := 0.0
	maxScore := 5.0

	// Has age
	if features.Age > 0 {
		score += 1.0
	}
	// Has conditions
	if len(features.Conditions) > 0 {
		score += 1.0
	}
	// Has medications
	if len(features.Medications) > 0 {
		score += 1.0
	}
	// Has labs
	if len(features.LabValues) > 0 {
		score += 1.0
	}
	// Has encounter history
	if len(features.Encounters) > 0 {
		score += 1.0
	}

	return score / maxScore
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
