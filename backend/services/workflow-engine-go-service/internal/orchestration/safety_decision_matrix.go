package orchestration

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"github.com/clinical-synthesis-hub/workflow-engine-go-service/pkg/clients"
)

// SafetyCategory represents the safety decision categories
type SafetyCategory string

const (
	// SafetyCategorySafe - Proceed directly to commit
	SafetyCategorySafe SafetyCategory = "SAFE"

	// SafetyCategoryConditionallySafe - Requires human review before commit
	SafetyCategoryConditionallySafe SafetyCategory = "CONDITIONALLY_SAFE"

	// SafetyCategoryModeratelyUnsafe - Automatic rework with adjusted parameters (max 2 attempts)
	SafetyCategoryModeratelyUnsafe SafetyCategory = "MODERATELY_UNSAFE"

	// SafetyCategorySeverelyUnsafe - Critical rejection, workflow must stop
	SafetyCategorySeverelyUnsafe SafetyCategory = "SEVERELY_UNSAFE"
)

// ActionType represents the required action based on safety category
type ActionType string

const (
	ActionProceedToCommit   ActionType = "PROCEED_TO_COMMIT"
	ActionRequestHumanReview ActionType = "REQUEST_HUMAN_REVIEW"
	ActionAttemptRework     ActionType = "ATTEMPT_REWORK"
	ActionCriticalReject    ActionType = "CRITICAL_REJECT"
)

// MatrixConfig contains configuration for the Safety Decision Matrix
type MatrixConfig struct {
	// Rework configuration
	ReworkMaxAttempts     int           `json:"rework_max_attempts"`
	ReworkBackoffDuration time.Duration `json:"rework_backoff_duration"`

	// Human review configuration
	HumanReviewSLA         time.Duration `json:"human_review_sla"`
	HumanReviewEscalationSLA time.Duration `json:"human_review_escalation_sla"`

	// Risk thresholds for categorization
	ConditionalSafeThreshold   float64 `json:"conditional_safe_threshold"`
	ModeratelyUnsafeThreshold  float64 `json:"moderately_unsafe_threshold"`
	SeverelyUnsafeThreshold    float64 `json:"severely_unsafe_threshold"`

	// Finding severity weights
	CriticalFindingWeight float64 `json:"critical_finding_weight"`
	HighFindingWeight     float64 `json:"high_finding_weight"`
	MediumFindingWeight   float64 `json:"medium_finding_weight"`
	LowFindingWeight      float64 `json:"low_finding_weight"`
}

// DefaultMatrixConfig returns the default configuration
func DefaultMatrixConfig() *MatrixConfig {
	return &MatrixConfig{
		ReworkMaxAttempts:         2,
		ReworkBackoffDuration:     500 * time.Millisecond,
		HumanReviewSLA:           2 * time.Hour,
		HumanReviewEscalationSLA: 30 * time.Minute,

		// Risk score thresholds (0.0 = safest, 1.0 = most unsafe)
		ConditionalSafeThreshold:  0.3,  // Below 0.3 = SAFE, 0.3-0.5 = CONDITIONALLY_SAFE
		ModeratelyUnsafeThreshold: 0.5,  // 0.5-0.7 = MODERATELY_UNSAFE
		SeverelyUnsafeThreshold:   0.7,  // Above 0.7 = SEVERELY_UNSAFE

		// Finding weights for risk calculation
		CriticalFindingWeight: 1.0,
		HighFindingWeight:     0.7,
		MediumFindingWeight:   0.4,
		LowFindingWeight:      0.1,
	}
}

// SafetyDecision represents the output of the Safety Decision Matrix evaluation
type SafetyDecision struct {
	// Core decision
	Category       SafetyCategory `json:"category"`
	RequiredAction ActionType     `json:"required_action"`
	RiskScore      float64        `json:"risk_score"`

	// Rework eligibility
	ReworkEligible     bool     `json:"rework_eligible"`
	ReworkAttemptCount int      `json:"rework_attempt_count"`
	ReworkParameters   []string `json:"rework_parameters,omitempty"`

	// Human review requirements
	HumanReviewRequired bool          `json:"human_review_required"`
	HumanReviewSLA     time.Duration `json:"human_review_sla,omitempty"`
	ReviewerLevel      string        `json:"reviewer_level,omitempty"`

	// Clinical findings and recommendations
	Findings         []Finding `json:"findings"`
	Recommendations  []string  `json:"recommendations"`
	Justification    string    `json:"justification"`

	// Metadata
	EvaluatedAt time.Time `json:"evaluated_at"`
	MatrixVersion string   `json:"matrix_version"`
}

// Finding represents a clinical safety finding
type Finding struct {
	FindingID            string  `json:"finding_id"`
	Severity             string  `json:"severity"`
	Category             string  `json:"category"`
	Description          string  `json:"description"`
	ClinicalSignificance string  `json:"clinical_significance"`
	Weight               float64 `json:"weight"`
}

// SafetyDecisionMatrix implements the 4-category safety decision logic
type SafetyDecisionMatrix struct {
	config *MatrixConfig
	logger *zap.Logger
}

// NewSafetyDecisionMatrix creates a new Safety Decision Matrix instance
func NewSafetyDecisionMatrix(config *MatrixConfig, logger *zap.Logger) *SafetyDecisionMatrix {
	if config == nil {
		config = DefaultMatrixConfig()
	}

	return &SafetyDecisionMatrix{
		config: config,
		logger: logger,
	}
}

// EvaluateValidation evaluates a validation result and determines the safety category and required action
func (m *SafetyDecisionMatrix) EvaluateValidation(
	ctx context.Context,
	validationResult *clients.SafetyValidationResponse,
	reworkAttemptCount int,
) (*SafetyDecision, error) {
	m.logger.Info("Evaluating validation result with Safety Decision Matrix",
		zap.String("validation_id", validationResult.ValidationID),
		zap.Int("rework_attempts", reworkAttemptCount))

	// Convert validation findings to internal format
	findings := m.convertFindings(validationResult.Findings)

	// Calculate risk score based on findings
	riskScore := m.calculateRiskScore(findings)

	// Determine safety category based on risk score and findings
	category := m.determineCategory(riskScore, findings)

	// Determine required action based on category and context
	action := m.determineAction(category, reworkAttemptCount)

	// Build safety decision
	decision := &SafetyDecision{
		Category:           category,
		RequiredAction:     action,
		RiskScore:          riskScore,
		ReworkAttemptCount: reworkAttemptCount,
		Findings:           findings,
		EvaluatedAt:        time.Now(),
		MatrixVersion:      "1.0.0",
	}

	// Add category-specific attributes
	switch category {
	case SafetyCategorySafe:
		decision.Justification = "All safety checks passed. Risk score within acceptable limits."
		decision.Recommendations = []string{"Proceed with standard medication order"}

	case SafetyCategoryConditionallySafe:
		decision.HumanReviewRequired = true
		decision.HumanReviewSLA = m.config.HumanReviewSLA
		decision.ReviewerLevel = "CLINICAL_PHARMACIST"
		decision.Justification = "Minor safety concerns detected. Clinical review recommended."
		decision.Recommendations = m.generateConditionalRecommendations(findings)

	case SafetyCategoryModeratelyUnsafe:
		if reworkAttemptCount < m.config.ReworkMaxAttempts {
			decision.ReworkEligible = true
			decision.ReworkParameters = m.identifyReworkParameters(findings)
			decision.Justification = fmt.Sprintf("Moderate safety concerns. Rework attempt %d of %d.",
				reworkAttemptCount+1, m.config.ReworkMaxAttempts)
		} else {
			// Max rework attempts reached, escalate to human review
			decision.ReworkEligible = false
			decision.HumanReviewRequired = true
			decision.HumanReviewSLA = m.config.HumanReviewSLA
			decision.ReviewerLevel = "SENIOR_CLINICIAN"
			decision.RequiredAction = ActionRequestHumanReview
			decision.Justification = "Maximum rework attempts exhausted. Escalating to human review."
		}
		decision.Recommendations = m.generateReworkRecommendations(findings)

	case SafetyCategorySeverelyUnsafe:
		decision.Justification = "Critical safety issues detected. Medication order cannot proceed."
		decision.Recommendations = m.generateCriticalRecommendations(findings)
	}

	m.logger.Info("Safety Decision Matrix evaluation complete",
		zap.String("category", string(category)),
		zap.String("action", string(action)),
		zap.Float64("risk_score", riskScore))

	return decision, nil
}

// calculateRiskScore calculates a normalized risk score (0.0-1.0) based on findings
func (m *SafetyDecisionMatrix) calculateRiskScore(findings []Finding) float64 {
	if len(findings) == 0 {
		return 0.0 // No findings = safe
	}

	totalWeight := 0.0
	maxPossibleWeight := 0.0

	// Calculate weighted risk score
	for _, finding := range findings {
		var weight float64
		switch finding.Severity {
		case "CRITICAL":
			weight = m.config.CriticalFindingWeight
		case "HIGH":
			weight = m.config.HighFindingWeight
		case "MEDIUM":
			weight = m.config.MediumFindingWeight
		case "LOW":
			weight = m.config.LowFindingWeight
		default:
			weight = 0.1
		}

		finding.Weight = weight
		totalWeight += weight

		// Track max possible weight for normalization
		if weight > maxPossibleWeight {
			maxPossibleWeight = m.config.CriticalFindingWeight
		}
	}

	// Normalize to 0.0-1.0 range
	// Consider both count and severity
	countFactor := float64(len(findings)) / 10.0 // Assume 10 findings is very high
	if countFactor > 1.0 {
		countFactor = 1.0
	}

	severityFactor := totalWeight / (float64(len(findings)) * m.config.CriticalFindingWeight)

	// Combine factors with weights
	riskScore := (0.6 * severityFactor) + (0.4 * countFactor)

	// Ensure within 0.0-1.0 range
	if riskScore > 1.0 {
		riskScore = 1.0
	} else if riskScore < 0.0 {
		riskScore = 0.0
	}

	return riskScore
}

// determineCategory determines the safety category based on risk score and findings
func (m *SafetyDecisionMatrix) determineCategory(riskScore float64, findings []Finding) SafetyCategory {
	// Check for any critical findings that automatically trigger SEVERELY_UNSAFE
	for _, finding := range findings {
		if finding.Severity == "CRITICAL" &&
		   (finding.Category == "CONTRAINDICATION" ||
		    finding.Category == "SEVERE_ALLERGY" ||
		    finding.Category == "LIFE_THREATENING") {
			return SafetyCategorySeverelyUnsafe
		}
	}

	// Categorize based on risk score thresholds
	if riskScore < m.config.ConditionalSafeThreshold {
		return SafetyCategorySafe
	} else if riskScore < m.config.ModeratelyUnsafeThreshold {
		return SafetyCategoryConditionallySafe
	} else if riskScore < m.config.SeverelyUnsafeThreshold {
		return SafetyCategoryModeratelyUnsafe
	} else {
		return SafetyCategorySeverelyUnsafe
	}
}

// determineAction determines the required action based on category and context
func (m *SafetyDecisionMatrix) determineAction(category SafetyCategory, reworkAttemptCount int) ActionType {
	switch category {
	case SafetyCategorySafe:
		return ActionProceedToCommit

	case SafetyCategoryConditionallySafe:
		return ActionRequestHumanReview

	case SafetyCategoryModeratelyUnsafe:
		if reworkAttemptCount < m.config.ReworkMaxAttempts {
			return ActionAttemptRework
		}
		// Max attempts reached, escalate to human review
		return ActionRequestHumanReview

	case SafetyCategorySeverelyUnsafe:
		return ActionCriticalReject

	default:
		return ActionCriticalReject // Default to safest option
	}
}

// convertFindings converts external validation findings to internal format
func (m *SafetyDecisionMatrix) convertFindings(externalFindings []clients.ValidationFinding) []Finding {
	findings := make([]Finding, len(externalFindings))

	for i, ef := range externalFindings {
		findings[i] = Finding{
			FindingID:            ef.FindingID,
			Severity:             ef.Severity,
			Category:             ef.Category,
			Description:          ef.Description,
			ClinicalSignificance: ef.ClinicalSignificance,
		}
	}

	return findings
}

// identifyReworkParameters identifies which parameters should be adjusted for rework
func (m *SafetyDecisionMatrix) identifyReworkParameters(findings []Finding) []string {
	paramSet := make(map[string]bool)

	for _, finding := range findings {
		switch finding.Category {
		case "DOSING":
			paramSet["dose_adjustment"] = true
			paramSet["frequency_adjustment"] = true

		case "DRUG_INTERACTION":
			paramSet["timing_adjustment"] = true
			paramSet["alternative_medication"] = true

		case "RENAL_ADJUSTMENT":
			paramSet["renal_dose_adjustment"] = true

		case "HEPATIC_ADJUSTMENT":
			paramSet["hepatic_dose_adjustment"] = true

		case "AGE_ADJUSTMENT":
			paramSet["age_based_adjustment"] = true

		default:
			paramSet["general_adjustment"] = true
		}
	}

	// Convert set to slice
	params := make([]string, 0, len(paramSet))
	for param := range paramSet {
		params = append(params, param)
	}

	return params
}

// generateConditionalRecommendations generates recommendations for conditionally safe scenarios
func (m *SafetyDecisionMatrix) generateConditionalRecommendations(findings []Finding) []string {
	recommendations := []string{
		"Clinical pharmacist review recommended",
		"Monitor patient closely for adverse effects",
	}

	// Add finding-specific recommendations
	for _, finding := range findings {
		if finding.Severity == "MEDIUM" || finding.Severity == "HIGH" {
			recommendations = append(recommendations,
				fmt.Sprintf("Address %s: %s", finding.Category, finding.Description))
		}
	}

	return recommendations
}

// generateReworkRecommendations generates recommendations for rework scenarios
func (m *SafetyDecisionMatrix) generateReworkRecommendations(findings []Finding) []string {
	recommendations := []string{
		"Adjust medication parameters based on safety findings",
		"Consider alternative dosing regimen",
	}

	// Add specific adjustments based on findings
	for _, finding := range findings {
		switch finding.Category {
		case "DOSING":
			recommendations = append(recommendations, "Reduce dose or adjust frequency")
		case "DRUG_INTERACTION":
			recommendations = append(recommendations, "Adjust timing or consider alternative")
		case "RENAL_ADJUSTMENT":
			recommendations = append(recommendations, "Apply renal dosing guidelines")
		}
	}

	return recommendations
}

// generateCriticalRecommendations generates recommendations for critical rejection scenarios
func (m *SafetyDecisionMatrix) generateCriticalRecommendations(findings []Finding) []string {
	recommendations := []string{
		"DO NOT PROCEED with current medication order",
		"Consult with senior clinician immediately",
		"Consider alternative treatment approach",
	}

	// Add critical finding details
	for _, finding := range findings {
		if finding.Severity == "CRITICAL" {
			recommendations = append(recommendations,
				fmt.Sprintf("CRITICAL: %s", finding.Description))
		}
	}

	return recommendations
}

// GetConfig returns the current matrix configuration
func (m *SafetyDecisionMatrix) GetConfig() *MatrixConfig {
	return m.config
}

// UpdateConfig updates the matrix configuration
func (m *SafetyDecisionMatrix) UpdateConfig(config *MatrixConfig) {
	m.config = config
	m.logger.Info("Safety Decision Matrix configuration updated",
		zap.Int("rework_max_attempts", config.ReworkMaxAttempts),
		zap.Duration("human_review_sla", config.HumanReviewSLA))
}