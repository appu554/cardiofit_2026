// Package criteria implements the criteria evaluation engine
package criteria

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"kb-17-population-registry/internal/models"
	"kb-17-population-registry/internal/registry"
)

// Engine evaluates patient eligibility for registries
type Engine struct {
	logger *logrus.Entry
}

// NewEngine creates a new criteria evaluation engine
func NewEngine(logger *logrus.Entry) *Engine {
	return &Engine{
		logger: logger.WithField("component", "criteria-engine"),
	}
}

// EvaluateAll evaluates a patient against all registries
func (e *Engine) EvaluateAll(patientData *models.PatientClinicalData) ([]models.CriteriaEvaluationResult, error) {
	registries := registry.GetAllRegistryDefinitions()
	results := make([]models.CriteriaEvaluationResult, 0, len(registries))

	for _, reg := range registries {
		if !reg.Active || !reg.AutoEnroll {
			continue
		}

		result, err := e.Evaluate(patientData, &reg)
		if err != nil {
			e.logger.WithError(err).WithField("registry", reg.Code).Warn("Failed to evaluate registry")
			continue
		}

		results = append(results, *result)
	}

	return results, nil
}

// Evaluate evaluates a patient against a specific registry
func (e *Engine) Evaluate(patientData *models.PatientClinicalData, reg *models.Registry) (*models.CriteriaEvaluationResult, error) {
	result := &models.CriteriaEvaluationResult{
		PatientID:         patientData.PatientID,
		RegistryCode:      reg.Code,
		EvaluatedAt:       time.Now().UTC(),
		MatchedCriteria:   make([]models.MatchedCriterion, 0),
		ExcludedCriteria:  make([]models.MatchedCriterion, 0),
		RiskFactors:       make([]models.RiskFactor, 0),
		SuggestedRiskTier: models.RiskTierModerate, // default
	}

	// Evaluate inclusion criteria
	meetsInclusion, matchedInclusion := e.evaluateCriteriaGroups(patientData, reg.InclusionCriteria)
	result.MeetsInclusion = meetsInclusion
	result.MatchedCriteria = append(result.MatchedCriteria, matchedInclusion...)

	// Evaluate exclusion criteria (if any)
	meetsExclusion := false
	if len(reg.ExclusionCriteria) > 0 {
		var excluded []models.MatchedCriterion
		meetsExclusion, excluded = e.evaluateCriteriaGroups(patientData, reg.ExclusionCriteria)
		result.MeetsExclusion = meetsExclusion
		result.ExcludedCriteria = append(result.ExcludedCriteria, excluded...)
	}

	// Patient is eligible if meets inclusion AND does NOT meet exclusion
	result.Eligible = meetsInclusion && !meetsExclusion

	// Calculate risk tier if eligible
	if result.Eligible && reg.RiskStratification != nil {
		riskTier, riskFactors := e.calculateRiskTier(patientData, reg.RiskStratification)
		result.SuggestedRiskTier = riskTier
		result.RiskFactors = riskFactors
	}

	e.logger.WithFields(logrus.Fields{
		"patient_id":      patientData.PatientID,
		"registry":        reg.Code,
		"eligible":        result.Eligible,
		"meets_inclusion": result.MeetsInclusion,
		"meets_exclusion": result.MeetsExclusion,
		"risk_tier":       result.SuggestedRiskTier,
	}).Debug("Criteria evaluation complete")

	return result, nil
}

// evaluateCriteriaGroups evaluates multiple criteria groups (OR between groups)
func (e *Engine) evaluateCriteriaGroups(patientData *models.PatientClinicalData, groups []models.CriteriaGroup) (bool, []models.MatchedCriterion) {
	if len(groups) == 0 {
		return false, nil
	}

	allMatched := make([]models.MatchedCriterion, 0)

	// Groups are evaluated with OR logic - any group matching is sufficient
	for _, group := range groups {
		groupMatches, matched := e.evaluateCriteriaGroup(patientData, &group)
		if groupMatches {
			allMatched = append(allMatched, matched...)
			return true, allMatched
		}
	}

	return false, allMatched
}

// evaluateCriteriaGroup evaluates a single criteria group
func (e *Engine) evaluateCriteriaGroup(patientData *models.PatientClinicalData, group *models.CriteriaGroup) (bool, []models.MatchedCriterion) {
	if len(group.Criteria) == 0 {
		return false, nil
	}

	matched := make([]models.MatchedCriterion, 0)
	matchCount := 0

	for _, criterion := range group.Criteria {
		criterionMatches, matchedValue := e.evaluateCriterion(patientData, &criterion)
		if criterionMatches {
			matchCount++
			matched = append(matched, models.MatchedCriterion{
				CriterionID:   criterion.ID,
				CriteriaGroup: group.ID,
				Type:          criterion.Type,
				Field:         criterion.Field,
				MatchedValue:  matchedValue,
				Description:   criterion.Description,
			})
		}
	}

	// Apply logical operator
	switch group.Operator {
	case models.LogicalAnd:
		return matchCount == len(group.Criteria), matched
	case models.LogicalOr:
		return matchCount > 0, matched
	default:
		// Default to OR
		return matchCount > 0, matched
	}
}

// evaluateCriterion evaluates a single criterion
func (e *Engine) evaluateCriterion(patientData *models.PatientClinicalData, criterion *models.Criterion) (bool, interface{}) {
	switch criterion.Type {
	case models.CriteriaTypeDiagnosis:
		return e.evaluateDiagnosis(patientData, criterion)
	case models.CriteriaTypeLabResult:
		return e.evaluateLabResult(patientData, criterion)
	case models.CriteriaTypeMedication:
		return e.evaluateMedication(patientData, criterion)
	case models.CriteriaTypeProblemList:
		return e.evaluateProblem(patientData, criterion)
	case models.CriteriaTypeAge:
		return e.evaluateAge(patientData, criterion)
	case models.CriteriaTypeGender:
		return e.evaluateGender(patientData, criterion)
	case models.CriteriaTypeVitalSign:
		return e.evaluateVitalSign(patientData, criterion)
	case models.CriteriaTypeRiskScore:
		return e.evaluateRiskScore(patientData, criterion)
	default:
		e.logger.WithField("type", criterion.Type).Warn("Unknown criteria type")
		return false, nil
	}
}

// evaluateDiagnosis evaluates diagnosis criteria
func (e *Engine) evaluateDiagnosis(patientData *models.PatientClinicalData, criterion *models.Criterion) (bool, interface{}) {
	for _, diag := range patientData.Diagnoses {
		// Check code system if specified
		if criterion.CodeSystem != "" && diag.CodeSystem != criterion.CodeSystem {
			continue
		}

		// Apply time window if specified
		if criterion.TimeWindow != nil && !e.isWithinTimeWindow(diag.RecordedAt, criterion.TimeWindow) {
			continue
		}

		if e.matchesValue(diag.Code, criterion) {
			return true, diag.Code
		}
	}
	return false, nil
}

// evaluateLabResult evaluates lab result criteria
func (e *Engine) evaluateLabResult(patientData *models.PatientClinicalData, criterion *models.Criterion) (bool, interface{}) {
	// For value-based criteria, sort labs by date (newest first) to use most recent values
	// This is clinically important - risk assessment should use the most recent lab data
	labs := make([]models.LabResult, len(patientData.LabResults))
	copy(labs, patientData.LabResults)

	if criterion.Field == "value" {
		sort.Slice(labs, func(i, j int) bool {
			return labs[i].EffectiveAt.After(labs[j].EffectiveAt)
		})
	}

	for _, lab := range labs {
		// Check code if specified
		if criterion.Field == "code" {
			if e.matchesValue(lab.Code, criterion) {
				return true, lab.Code
			}
			continue
		}

		// Check value - for risk stratification, only use the most recent valid lab value
		if criterion.Field == "value" {
			// Apply time window if specified
			if criterion.TimeWindow != nil && !e.isWithinTimeWindow(lab.EffectiveAt, criterion.TimeWindow) {
				continue
			}

			// For clinical correctness, evaluate ONLY the most recent lab value
			// If it matches the criterion, return true; otherwise return false
			// Don't fall through to check older lab values - the most recent value
			// represents the patient's current state
			return e.matchesNumericValue(lab.Value, criterion), lab.Value
		}
	}
	return false, nil
}

// evaluateMedication evaluates medication criteria
func (e *Engine) evaluateMedication(patientData *models.PatientClinicalData, criterion *models.Criterion) (bool, interface{}) {
	for _, med := range patientData.Medications {
		// Only consider active medications
		if med.Status != "" && med.Status != "active" {
			continue
		}

		// Check code system if specified
		if criterion.CodeSystem != "" && med.CodeSystem != criterion.CodeSystem {
			continue
		}

		if e.matchesValue(med.Code, criterion) {
			return true, med.Code
		}
	}
	return false, nil
}

// evaluateProblem evaluates problem list criteria
func (e *Engine) evaluateProblem(patientData *models.PatientClinicalData, criterion *models.Criterion) (bool, interface{}) {
	for _, prob := range patientData.Problems {
		// Only consider active problems
		if prob.Status != "" && prob.Status != "active" {
			continue
		}

		// Check code system if specified
		if criterion.CodeSystem != "" && prob.CodeSystem != criterion.CodeSystem {
			continue
		}

		if e.matchesValue(prob.Code, criterion) {
			return true, prob.Code
		}
	}
	return false, nil
}

// evaluateAge evaluates age criteria
func (e *Engine) evaluateAge(patientData *models.PatientClinicalData, criterion *models.Criterion) (bool, interface{}) {
	if patientData.Demographics == nil {
		return false, nil
	}

	age := patientData.Demographics.Age
	if age == 0 && patientData.Demographics.BirthDate != nil {
		age = calculateAge(*patientData.Demographics.BirthDate)
	}

	if age == 0 {
		return false, nil
	}

	if e.matchesNumericValue(age, criterion) {
		return true, age
	}
	return false, nil
}

// evaluateGender evaluates gender criteria
func (e *Engine) evaluateGender(patientData *models.PatientClinicalData, criterion *models.Criterion) (bool, interface{}) {
	if patientData.Demographics == nil {
		return false, nil
	}

	gender := strings.ToLower(patientData.Demographics.Gender)
	expectedValue := strings.ToLower(fmt.Sprintf("%v", criterion.Value))

	if gender == expectedValue {
		return true, gender
	}
	return false, nil
}

// evaluateVitalSign evaluates vital sign criteria
func (e *Engine) evaluateVitalSign(patientData *models.PatientClinicalData, criterion *models.Criterion) (bool, interface{}) {
	for _, vital := range patientData.VitalSigns {
		// Apply time window if specified
		if criterion.TimeWindow != nil && !e.isWithinTimeWindow(vital.EffectiveAt, criterion.TimeWindow) {
			continue
		}

		// Handle blood pressure specially - it has systolic/diastolic components
		if strings.EqualFold(vital.Type, "blood-pressure") {
			if strings.EqualFold(criterion.Field, "systolic") || strings.EqualFold(criterion.Field, "diastolic") {
				// Extract the specific BP component from the map
				if bpMap, ok := vital.Value.(map[string]interface{}); ok {
					componentKey := strings.ToLower(criterion.Field)
					if componentValue, exists := bpMap[componentKey]; exists {
						if e.matchesNumericValue(componentValue, criterion) {
							return true, componentValue
						}
					}
				}
				continue
			}
		}

		// Standard vital sign matching
		if strings.EqualFold(vital.Type, criterion.Field) {
			if e.matchesNumericValue(vital.Value, criterion) {
				return true, vital.Value
			}
		}
	}
	return false, nil
}

// evaluateRiskScore evaluates risk score criteria
func (e *Engine) evaluateRiskScore(patientData *models.PatientClinicalData, criterion *models.Criterion) (bool, interface{}) {
	for _, score := range patientData.RiskScores {
		if strings.EqualFold(score.ScoreType, criterion.Field) {
			if e.matchesNumericValue(score.Value, criterion) {
				return true, score.Value
			}
		}
	}
	return false, nil
}

// matchesValue checks if a value matches the criterion
func (e *Engine) matchesValue(value string, criterion *models.Criterion) bool {
	switch criterion.Operator {
	case models.OperatorEquals:
		return value == fmt.Sprintf("%v", criterion.Value)

	case models.OperatorNotEquals:
		return value != fmt.Sprintf("%v", criterion.Value)

	case models.OperatorStartsWith:
		return strings.HasPrefix(value, fmt.Sprintf("%v", criterion.Value))

	case models.OperatorEndsWith:
		return strings.HasSuffix(value, fmt.Sprintf("%v", criterion.Value))

	case models.OperatorContains:
		return strings.Contains(value, fmt.Sprintf("%v", criterion.Value))

	case models.OperatorIn:
		for _, v := range criterion.Values {
			if value == fmt.Sprintf("%v", v) {
				return true
			}
		}
		return false

	case models.OperatorNotIn:
		for _, v := range criterion.Values {
			if value == fmt.Sprintf("%v", v) {
				return false
			}
		}
		return true

	case models.OperatorExists:
		return value != ""

	case models.OperatorNotExists:
		return value == ""

	default:
		return false
	}
}

// matchesNumericValue checks if a numeric value matches the criterion
func (e *Engine) matchesNumericValue(value interface{}, criterion *models.Criterion) bool {
	numValue, ok := toFloat64(value)
	if !ok {
		return false
	}

	criterionValue, ok := toFloat64(criterion.Value)
	if !ok && criterion.Operator != models.OperatorBetween && criterion.Operator != models.OperatorIn {
		return false
	}

	switch criterion.Operator {
	case models.OperatorEquals:
		return numValue == criterionValue

	case models.OperatorNotEquals:
		return numValue != criterionValue

	case models.OperatorGreaterThan:
		return numValue > criterionValue

	case models.OperatorGreaterOrEqual:
		return numValue >= criterionValue

	case models.OperatorLessThan:
		return numValue < criterionValue

	case models.OperatorLessOrEqual:
		return numValue <= criterionValue

	case models.OperatorBetween:
		if len(criterion.Values) < 2 {
			return false
		}
		low, ok1 := toFloat64(criterion.Values[0])
		high, ok2 := toFloat64(criterion.Values[1])
		if !ok1 || !ok2 {
			return false
		}
		return numValue >= low && numValue < high

	default:
		return false
	}
}

// isWithinTimeWindow checks if a timestamp is within the time window
func (e *Engine) isWithinTimeWindow(timestamp time.Time, window *models.TimeWindow) bool {
	if window == nil {
		return true
	}

	now := time.Now().UTC()

	// Check "within" duration (e.g., "30d", "1y")
	if window.Within != "" {
		duration, err := parseDuration(window.Within)
		if err == nil {
			windowStart := now.Add(-duration)
			if timestamp.Before(windowStart) {
				return false
			}
		}
	}

	// Check explicit after/before
	if window.After != nil && timestamp.Before(*window.After) {
		return false
	}
	if window.Before != nil && timestamp.After(*window.Before) {
		return false
	}

	return true
}

// calculateRiskTier calculates the risk tier based on stratification config
func (e *Engine) calculateRiskTier(patientData *models.PatientClinicalData, config *models.RiskStratificationConfig) (models.RiskTier, []models.RiskFactor) {
	riskFactors := make([]models.RiskFactor, 0)

	// Evaluate rules in priority order (lower priority = higher urgency)
	for _, rule := range config.Rules {
		matches, _ := e.evaluateCriteriaGroups(patientData, rule.Criteria)
		if matches {
			return rule.Tier, riskFactors
		}
	}

	// Default to low risk if no rules matched
	return models.RiskTierLow, riskFactors
}

// Helper functions

// toFloat64 converts various numeric types to float64
func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case int32:
		return float64(val), true
	case string:
		f, err := strconv.ParseFloat(val, 64)
		return f, err == nil
	default:
		return 0, false
	}
}

// calculateAge calculates age from birth date
func calculateAge(birthDate time.Time) int {
	now := time.Now().UTC()
	years := now.Year() - birthDate.Year()
	if now.YearDay() < birthDate.YearDay() {
		years--
	}
	return years
}

// parseDuration parses duration strings like "30d", "1y", "6m"
func parseDuration(s string) (time.Duration, error) {
	re := regexp.MustCompile(`^(\d+)([dDwWmMyY])$`)
	matches := re.FindStringSubmatch(s)
	if len(matches) != 3 {
		return 0, fmt.Errorf("invalid duration format: %s", s)
	}

	value, _ := strconv.Atoi(matches[1])
	unit := strings.ToLower(matches[2])

	switch unit {
	case "d":
		return time.Duration(value) * 24 * time.Hour, nil
	case "w":
		return time.Duration(value) * 7 * 24 * time.Hour, nil
	case "m":
		return time.Duration(value) * 30 * 24 * time.Hour, nil
	case "y":
		return time.Duration(value) * 365 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown duration unit: %s", unit)
	}
}
