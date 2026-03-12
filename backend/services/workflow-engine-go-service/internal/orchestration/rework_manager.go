package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/clinical-synthesis-hub/workflow-engine-go-service/pkg/clients"
	"go.uber.org/zap"
)

// Define missing error constants
var (
	ErrMaxReworkAttemptsExceeded = errors.New("maximum rework attempts exceeded")
)

// ReworkManager handles automatic rework attempts for MODERATELY_UNSAFE scenarios
type ReworkManager struct {
	maxAttempts   int
	backoffDelay  time.Duration
	calculator    clients.Flow2GoClient
	logger        *zap.Logger
	adjustmentRules map[string][]AdjustmentRule
}

// ReworkContext contains information about the current rework attempt
type ReworkContext struct {
	WorkflowID      string                       `json:"workflow_id"`
	AttemptNumber   int                          `json:"attempt_number"`
	PreviousResults []ReworkValidationResult     `json:"previous_results"`
	AdjustmentRules []AdjustmentRule             `json:"adjustment_rules"`
	StartedAt       time.Time                    `json:"started_at"`
	Parameters      map[string]interface{}       `json:"parameters"`
}

// AdjustmentRule defines how to modify parameters for rework
type AdjustmentRule struct {
	Parameter       string  `json:"parameter"`
	AdjustmentType  string  `json:"adjustment_type"`  // RELAX, TIGHTEN, ALTERNATIVE
	Factor          float64 `json:"factor"`           // Adjustment factor (e.g., 0.9 for 10% reduction)
	Description     string  `json:"description"`
	Priority        int     `json:"priority"`         // Higher priority rules applied first
}

// ReworkValidationResult represents the outcome of a validation attempt (renamed to avoid conflict)
type ReworkValidationResult struct {
	ValidationID string                           `json:"validation_id"`
	Verdict      string                           `json:"verdict"`
	Findings     []clients.ValidationFinding      `json:"findings"`
	Timestamp    time.Time                        `json:"timestamp"`
}

// ReworkResult contains the outcome of a rework attempt
type ReworkResult struct {
	AttemptNumber       int                       `json:"attempt_number"`
	Success             bool                      `json:"success"`
	AdjustedParams      map[string]interface{}    `json:"adjusted_params"`
	RecalculationResult *CalculationResult        `json:"recalculation_result"`
	ValidationResult    *ReworkValidationResult   `json:"validation_result"`
	Duration            time.Duration             `json:"duration"`
	NextAction          string                    `json:"next_action"` // PROCEED, RETRY, ESCALATE
}

// CalculationResult represents the outcome of a recalculation
type CalculationResult struct {
	CalculationID   string                 `json:"calculation_id"`
	MedicationOrder map[string]interface{} `json:"medication_order"`
	ClinicalContext map[string]interface{} `json:"clinical_context"`
	Timestamp       time.Time              `json:"timestamp"`
}

// ParameterAdjustment tracks parameter changes
type ParameterAdjustment struct {
	Parameter    string      `json:"parameter"`
	OriginalValue interface{} `json:"original_value"`
	AdjustedValue interface{} `json:"adjusted_value"`
	Reason       string      `json:"reason"`
	Timestamp    time.Time   `json:"timestamp"`
}

// NewReworkManager creates a new instance of ReworkManager
func NewReworkManager(maxAttempts int, backoffDelay time.Duration, calculator clients.Flow2GoClient, logger *zap.Logger) *ReworkManager {
	return &ReworkManager{
		maxAttempts:     maxAttempts,
		backoffDelay:    backoffDelay,
		calculator:      calculator,
		logger:          logger,
		adjustmentRules: initializeAdjustmentRulesMap(),
	}
}

// AttemptRework performs an automatic rework attempt with adjusted parameters
func (r *ReworkManager) AttemptRework(
	ctx context.Context,
	workflowState *WorkflowState,
	validationResult *ReworkValidationResult,
) (*ReworkResult, error) {
	startTime := time.Now()

	// Check if we've exceeded max attempts
	if workflowState.ReworkAttempts >= r.maxAttempts {
		r.logger.Warn("Maximum rework attempts exceeded",
			zap.String("workflow_id", workflowState.WorkflowID),
			zap.Int("attempts", workflowState.ReworkAttempts))

		return &ReworkResult{
			AttemptNumber: workflowState.ReworkAttempts,
			Success:       false,
			NextAction:    "ESCALATE",
			Duration:      time.Since(startTime),
		}, ErrMaxReworkAttemptsExceeded
	}

	// Apply exponential backoff between attempts
	if workflowState.ReworkAttempts > 0 {
		backoff := r.calculateBackoff(workflowState.ReworkAttempts)
		r.logger.Info("Applying backoff before rework attempt",
			zap.Duration("backoff", backoff),
			zap.Int("attempt", workflowState.ReworkAttempts+1))
		time.Sleep(backoff)
	}

	// Determine adjustment rules based on validation findings
	adjustmentRules := r.determineAdjustmentRules(validationResult.Findings)

	// Apply adjustments to create new parameters
	adjustedParams, adjustments := r.applyAdjustments(
		workflowState.OriginalRequest,
		adjustmentRules,
		validationResult.Findings,
	)

	// Log the adjustments being made
	r.logAdjustments(workflowState.WorkflowID, adjustments)

	// Increment attempt counter
	workflowState.ReworkAttempts++

	// Record the rework attempt in history
	r.recordReworkAttempt(workflowState, adjustments)

	// Recalculate with adjusted parameters
	recalcResult, err := r.recalculateWithParams(ctx, workflowState, adjustedParams)
	if err != nil {
		r.logger.Error("Recalculation failed during rework",
			zap.Error(err),
			zap.String("workflow_id", workflowState.WorkflowID),
			zap.Int("attempt", workflowState.ReworkAttempts))

		return &ReworkResult{
			AttemptNumber:  workflowState.ReworkAttempts,
			Success:        false,
			AdjustedParams: adjustedParams,
			NextAction:     "ESCALATE",
			Duration:       time.Since(startTime),
		}, err
	}

	// Determine next action based on recalculation result
	nextAction := r.determineNextAction(workflowState.ReworkAttempts, r.maxAttempts)

	return &ReworkResult{
		AttemptNumber:       workflowState.ReworkAttempts,
		Success:             true,
		AdjustedParams:      adjustedParams,
		RecalculationResult: recalcResult,
		Duration:            time.Since(startTime),
		NextAction:          nextAction,
	}, nil
}

// determineAdjustmentRules selects appropriate adjustment rules based on findings
func (r *ReworkManager) determineAdjustmentRules(findings []clients.ValidationFinding) []AdjustmentRule {
	rules := make([]AdjustmentRule, 0)
	ruleSet := make(map[string]bool) // Prevent duplicate rules

	for _, finding := range findings {
		category := finding.Category

		// Select rules based on finding category
		if categoryRules, exists := r.adjustmentRules[category]; exists {
			for _, rule := range categoryRules {
				ruleKey := fmt.Sprintf("%s:%s", rule.Parameter, rule.AdjustmentType)
				if !ruleSet[ruleKey] {
					rules = append(rules, rule)
					ruleSet[ruleKey] = true
				}
			}
		}
	}

	// Sort rules by priority (higher priority first)
	r.sortRulesByPriority(rules)

	return rules
}

// applyAdjustments modifies parameters based on adjustment rules
func (r *ReworkManager) applyAdjustments(
	originalRequest map[string]interface{},
	rules []AdjustmentRule,
	findings []clients.ValidationFinding,
) (map[string]interface{}, []ParameterAdjustment) {
	// Deep copy the original request
	adjustedParams := r.deepCopyMap(originalRequest)
	adjustments := make([]ParameterAdjustment, 0)

	for _, rule := range rules {
		originalValue := r.getParameterValue(adjustedParams, rule.Parameter)
		adjustedValue := r.calculateAdjustment(originalValue, rule)

		if adjustedValue != originalValue {
			r.setParameterValue(adjustedParams, rule.Parameter, adjustedValue)

			adjustments = append(adjustments, ParameterAdjustment{
				Parameter:     rule.Parameter,
				OriginalValue: originalValue,
				AdjustedValue: adjustedValue,
				Reason:        rule.Description,
				Timestamp:     time.Now(),
			})
		}
	}

	// Apply finding-specific adjustments
	for _, finding := range findings {
		if finding.Severity == "HIGH" || finding.Severity == "MEDIUM" {
			specificAdjustments := r.applyFindingSpecificAdjustments(adjustedParams, finding)
			adjustments = append(adjustments, specificAdjustments...)
		}
	}

	return adjustedParams, adjustments
}

// calculateAdjustment applies the adjustment rule to a parameter value
func (r *ReworkManager) calculateAdjustment(value interface{}, rule AdjustmentRule) interface{} {
	switch rule.AdjustmentType {
	case "RELAX":
		// Increase tolerance or reduce strictness
		if floatVal, ok := value.(float64); ok {
			return floatVal * (1.0 + rule.Factor)
		}
	case "TIGHTEN":
		// Decrease tolerance or increase strictness
		if floatVal, ok := value.(float64); ok {
			return floatVal * (1.0 - rule.Factor)
		}
	case "ALTERNATIVE":
		// Use alternative parameter values
		return r.getAlternativeValue(rule.Parameter, value)
	}

	return value
}

// applyFindingSpecificAdjustments makes targeted adjustments based on specific findings
func (r *ReworkManager) applyFindingSpecificAdjustments(
	params map[string]interface{},
	finding clients.ValidationFinding,
) []ParameterAdjustment {
	adjustments := make([]ParameterAdjustment, 0)

	switch finding.Category {
	case "DOSING":
		// Adjust dose based on clinical factors
		if dose, exists := params["dose"].(float64); exists {
			adjustedDose := r.adjustDoseForClinicalFactors(dose, finding)
			if adjustedDose != dose {
				params["dose"] = adjustedDose
				adjustments = append(adjustments, ParameterAdjustment{
					Parameter:     "dose",
					OriginalValue: dose,
					AdjustedValue: adjustedDose,
					Reason:        fmt.Sprintf("Dosing adjustment: %s", finding.Description),
					Timestamp:     time.Now(),
				})
			}
		}

	case "DRUG_INTERACTION":
		// Adjust timing or consider alternatives
		if timing, exists := params["administration_timing"].(string); exists {
			adjustedTiming := r.adjustTimingForInteraction(timing, finding)
			if adjustedTiming != timing {
				params["administration_timing"] = adjustedTiming
				adjustments = append(adjustments, ParameterAdjustment{
					Parameter:     "administration_timing",
					OriginalValue: timing,
					AdjustedValue: adjustedTiming,
					Reason:        fmt.Sprintf("Drug interaction mitigation: %s", finding.Description),
					Timestamp:     time.Now(),
				})
			}
		}

	case "RENAL_ADJUSTMENT":
		// Apply renal dosing adjustments
		if dose, exists := params["dose"].(float64); exists {
			if creatinine, hasCreatinine := params["creatinine_clearance"].(float64); hasCreatinine {
				adjustedDose := r.calculateRenalAdjustedDose(dose, creatinine)
				if adjustedDose != dose {
					params["dose"] = adjustedDose
					adjustments = append(adjustments, ParameterAdjustment{
						Parameter:     "dose",
						OriginalValue: dose,
						AdjustedValue: adjustedDose,
						Reason:        fmt.Sprintf("Renal adjustment (CrCl: %.1f): %s", creatinine, finding.Description),
						Timestamp:     time.Now(),
					})
				}
			}
		}

	case "HEPATIC_ADJUSTMENT":
		// Apply hepatic dosing adjustments
		if dose, exists := params["dose"].(float64); exists {
			adjustedDose := r.calculateHepaticAdjustedDose(dose, finding)
			if adjustedDose != dose {
				params["dose"] = adjustedDose
				adjustments = append(adjustments, ParameterAdjustment{
					Parameter:     "dose",
					OriginalValue: dose,
					AdjustedValue: adjustedDose,
					Reason:        fmt.Sprintf("Hepatic adjustment: %s", finding.Description),
					Timestamp:     time.Now(),
				})
			}
		}
	}

	return adjustments
}

// recalculateWithParams performs recalculation with adjusted parameters
func (r *ReworkManager) recalculateWithParams(
	ctx context.Context,
	workflowState *WorkflowState,
	adjustedParams map[string]interface{},
) (*CalculationResult, error) {
	// TODO: Implement CalculationRequest and Calculate method
	// For now, return a placeholder result
	r.logger.Info("Recalculation requested with adjusted parameters",
		zap.String("patient_id", workflowState.PatientID),
		zap.String("workflow_id", workflowState.WorkflowID),
		zap.Any("adjusted_params", adjustedParams))

	// Return placeholder calculation result
	return &CalculationResult{
		CalculationID:   fmt.Sprintf("recalc_%s_%d", workflowState.WorkflowID, workflowState.ReworkAttempts),
		MedicationOrder: adjustedParams,
		ClinicalContext: map[string]interface{}{"rework_attempt": workflowState.ReworkAttempts},
		Timestamp:       time.Now(),
	}, nil
}

// Helper functions

func (r *ReworkManager) calculateBackoff(attemptNumber int) time.Duration {
	// Exponential backoff: base * 2^attempt
	return r.backoffDelay * time.Duration(1<<uint(attemptNumber-1))
}

func (r *ReworkManager) determineNextAction(currentAttempt, maxAttempts int) string {
	if currentAttempt >= maxAttempts {
		return "ESCALATE"
	} else if currentAttempt == maxAttempts-1 {
		return "FINAL_RETRY"
	}
	return "RETRY"
}

func (r *ReworkManager) sortRulesByPriority(rules []AdjustmentRule) {
	// Simple bubble sort for small rule sets
	for i := 0; i < len(rules)-1; i++ {
		for j := 0; j < len(rules)-i-1; j++ {
			if rules[j].Priority < rules[j+1].Priority {
				rules[j], rules[j+1] = rules[j+1], rules[j]
			}
		}
	}
}

func (r *ReworkManager) deepCopyMap(original map[string]interface{}) map[string]interface{} {
	// Use JSON marshaling for deep copy
	jsonBytes, _ := json.Marshal(original)
	var copy map[string]interface{}
	json.Unmarshal(jsonBytes, &copy)
	return copy
}

func (r *ReworkManager) getParameterValue(params map[string]interface{}, path string) interface{} {
	// Support nested parameter paths (e.g., "medication.dose")
	// For now, simple implementation
	if value, exists := params[path]; exists {
		return value
	}
	return nil
}

func (r *ReworkManager) setParameterValue(params map[string]interface{}, path string, value interface{}) {
	// Support nested parameter paths (e.g., "medication.dose")
	// For now, simple implementation
	params[path] = value
}

func (r *ReworkManager) getAlternativeValue(parameter string, currentValue interface{}) interface{} {
	// Return alternative values based on parameter type
	switch parameter {
	case "route":
		if currentValue == "IV" {
			return "PO" // Switch from intravenous to oral
		}
		return "IV"
	case "frequency":
		if currentValue == "QID" {
			return "TID" // Reduce frequency
		}
		return currentValue
	default:
		return currentValue
	}
}

func (r *ReworkManager) adjustDoseForClinicalFactors(dose float64, finding clients.ValidationFinding) float64 {
	// Apply clinical factor-based dose adjustments
	switch finding.Severity {
	case "HIGH":
		return dose * 0.75 // 25% reduction for high severity
	case "MEDIUM":
		return dose * 0.9  // 10% reduction for medium severity
	default:
		return dose
	}
}

func (r *ReworkManager) adjustTimingForInteraction(timing string, finding clients.ValidationFinding) string {
	// Adjust administration timing to avoid drug interactions
	timingMap := map[string]string{
		"WITH_MEALS":    "BETWEEN_MEALS",
		"BETWEEN_MEALS": "WITH_MEALS",
		"MORNING":       "EVENING",
		"EVENING":       "MORNING",
	}

	if newTiming, exists := timingMap[timing]; exists {
		return newTiming
	}
	return timing
}

func (r *ReworkManager) calculateRenalAdjustedDose(dose float64, creatinineClearance float64) float64 {
	// Apply renal dosing adjustments based on creatinine clearance
	if creatinineClearance < 30 {
		return dose * 0.5  // 50% reduction for severe renal impairment
	} else if creatinineClearance < 60 {
		return dose * 0.75 // 25% reduction for moderate renal impairment
	}
	return dose
}

func (r *ReworkManager) calculateHepaticAdjustedDose(dose float64, finding clients.ValidationFinding) float64 {
	// Apply hepatic dosing adjustments
	// This would typically use Child-Pugh score or other hepatic function markers
	return dose * 0.75 // Conservative 25% reduction for hepatic impairment
}

func (r *ReworkManager) logAdjustments(workflowID string, adjustments []ParameterAdjustment) {
	for _, adj := range adjustments {
		r.logger.Info("Parameter adjusted for rework",
			zap.String("workflow_id", workflowID),
			zap.String("parameter", adj.Parameter),
			zap.Any("original", adj.OriginalValue),
			zap.Any("adjusted", adj.AdjustedValue),
			zap.String("reason", adj.Reason))
	}
}

func (r *ReworkManager) recordReworkAttempt(workflowState *WorkflowState, adjustments []ParameterAdjustment) {
	attempt := ReworkAttempt{
		AttemptNumber: workflowState.ReworkAttempts,
		Timestamp:     time.Now(),
		Adjustments:   adjustments,
	}

	if workflowState.ReworkHistory == nil {
		workflowState.ReworkHistory = make([]ReworkAttempt, 0)
	}
	workflowState.ReworkHistory = append(workflowState.ReworkHistory, attempt)
}

// initializeAdjustmentRulesMap sets up the default adjustment rules by category
func initializeAdjustmentRulesMap() map[string][]AdjustmentRule {
	return map[string][]AdjustmentRule{
		"DOSING": {
			{
				Parameter:      "dose",
				AdjustmentType: "TIGHTEN",
				Factor:         0.1,
				Description:    "Reduce dose by 10% for safety",
				Priority:       10,
			},
			{
				Parameter:      "frequency",
				AdjustmentType: "RELAX",
				Factor:         0.2,
				Description:    "Increase dosing interval",
				Priority:       5,
			},
		},
		"DRUG_INTERACTION": {
			{
				Parameter:      "administration_timing",
				AdjustmentType: "ALTERNATIVE",
				Factor:         0,
				Description:    "Adjust timing to avoid interaction",
				Priority:       15,
			},
			{
				Parameter:      "medication_code",
				AdjustmentType: "ALTERNATIVE",
				Factor:         0,
				Description:    "Consider alternative medication",
				Priority:       8,
			},
		},
		"RENAL_ADJUSTMENT": {
			{
				Parameter:      "dose",
				AdjustmentType: "TIGHTEN",
				Factor:         0.25,
				Description:    "Apply renal dosing guidelines",
				Priority:       12,
			},
		},
		"HEPATIC_ADJUSTMENT": {
			{
				Parameter:      "dose",
				AdjustmentType: "TIGHTEN",
				Factor:         0.25,
				Description:    "Apply hepatic dosing guidelines",
				Priority:       12,
			},
		},
		"AGE_ADJUSTMENT": {
			{
				Parameter:      "dose",
				AdjustmentType: "TIGHTEN",
				Factor:         0.2,
				Description:    "Age-based dose adjustment",
				Priority:       10,
			},
		},
	}
}

// ReworkAttempt represents a single rework attempt in history
type ReworkAttempt struct {
	AttemptNumber int                    `json:"attempt_number"`
	Timestamp     time.Time              `json:"timestamp"`
	Adjustments   []ParameterAdjustment  `json:"adjustments"`
	Result        string                 `json:"result,omitempty"`
}

var (
	ErrReworkCalculationFailed   = fmt.Errorf("rework calculation failed")
	ErrNoAdjustmentRulesFound    = fmt.Errorf("no adjustment rules found for findings")
)