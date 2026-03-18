package services

import (
	"context"
	"strings"

	"kb-25-lifestyle-knowledge-graph/internal/clients"
	"kb-25-lifestyle-knowledge-graph/internal/graph"
	"kb-25-lifestyle-knowledge-graph/internal/models"

	"go.uber.org/zap"
)

type SafetyEngine struct {
	graphClient graph.GraphClient
	kb4Client   *clients.KB4Client
	logger      *zap.Logger
}

func NewSafetyEngine(graphClient graph.GraphClient, kb4Client *clients.KB4Client, logger *zap.Logger) *SafetyEngine {
	return &SafetyEngine{graphClient: graphClient, kb4Client: kb4Client, logger: logger}
}

func (s *SafetyEngine) CheckSafety(ctx context.Context, patient *clients.PatientSnapshot, interventionCodes []string, medicationCodes []string) (*models.SafetyCheckResult, error) {
	result := &models.SafetyCheckResult{Safe: true}

	allRules := GetAllLSRules()
	violations := EvaluateSafetyRules(patient, allRules)

	for _, v := range violations {
		if v.Severity == "HARD_STOP" {
			result.Safe = false
			result.Violations = append(result.Violations, v)
		} else {
			result.Warnings = append(result.Warnings, v)
		}
	}

	if len(medicationCodes) > 0 {
		for _, code := range interventionCodes {
			interactions, err := s.getInteractions(ctx, code, medicationCodes)
			if err != nil {
				s.logger.Warn("interaction check failed", zap.String("code", code), zap.Error(err))
				continue
			}
			result.Interactions = append(result.Interactions, interactions...)
			for _, inter := range interactions {
				if inter.Severity == "HIGH" {
					result.Safe = false
				}
			}
		}
	}

	return result, nil
}

func (s *SafetyEngine) getInteractions(ctx context.Context, lifestyleCode string, drugCodes []string) ([]models.InteractionEntry, error) {
	records, err := s.graphClient.Run(ctx, graph.CypherGetDrugInteractions, map[string]any{
		"lifestyle_code": lifestyleCode,
		"drug_codes":     drugCodes,
	})
	if err != nil {
		return nil, err
	}

	var interactions []models.InteractionEntry
	for _, rec := range records {
		drugCode, _ := rec.Get("drug_class_code")
		interaction, _ := rec.Get("interaction")
		severity, _ := rec.Get("severity")
		action, _ := rec.Get("action")
		desc, _ := rec.Get("description")

		interactions = append(interactions, models.InteractionEntry{
			LifestyleCode: lifestyleCode,
			DrugClassCode: asString(drugCode),
			Interaction:   asString(interaction),
			Severity:      asString(severity),
			Action:        asString(action),
			Description:   asString(desc),
		})
	}
	return interactions, nil
}

func asString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func EvaluateSafetyRules(patient *clients.PatientSnapshot, rules []models.LSRule) []models.SafetyViolation {
	var violations []models.SafetyViolation
	for _, rule := range rules {
		if evaluateSafetyCondition(rule.Condition, patient) {
			violations = append(violations, models.SafetyViolation{
				RuleCode:    rule.Code,
				Description: rule.Description,
				Severity:    rule.Severity,
				Blocked:     rule.Blocked,
			})
		}
	}
	return violations
}

func evaluateSafetyCondition(condition string, patient *clients.PatientSnapshot) bool {
	switch condition {
	case "eGFR < 30":
		return patient.EGFR > 0 && patient.EGFR < 30
	case "SBP > 180":
		return patient.SBP > 180
	case "DBP > 110":
		return patient.DBP > 110
	case "SBP > 180 OR DBP > 110":
		return patient.SBP > 180 || patient.DBP > 110
	case "HbA1c > 13":
		return patient.HbA1c > 13
	case "potassium > 5.5":
		return patient.Potassium > 5.5
	case "FBG < 70 in last 7d":
		return patient.FBGMin7d > 0 && patient.FBGMin7d < 70
	case "current_meds includes SU or insulin":
		return containsAny(patient.Medications, "SU", "sulfonylurea", "insulin")
	case "current_meds includes SGLT2i":
		return containsAny(patient.Medications, "SGLT2i", "sglt2-inhibitor", "dapagliflozin", "empagliflozin", "canagliflozin")
	case "retinopathy = PROLIFERATIVE":
		return patient.Retinopathy == "PROLIFERATIVE"
	case "neuropathy = true":
		return patient.Neuropathy
	case "pregnant = true AND diabetes":
		return patient.Pregnant && patient.HasDiabetes
	case "cardiac_event within 30d":
		return patient.CardiacEvent30d
	case "BMR < 1200 kcal":
		return patient.BMR > 0 && patient.BMR < 1200
	case "gastroparesis = true":
		return patient.Gastroparesis
	case "eating_disorder_history = true":
		return patient.EatingDisorderHx
	case "BMI < 22":
		return patient.BMI > 0 && patient.BMI < 22
	default:
		return false
	}
}

// containsAny checks if any of the target strings appear in the slice (case-insensitive prefix match).
func containsAny(meds []string, targets ...string) bool {
	for _, med := range meds {
		lower := strings.ToLower(med)
		for _, t := range targets {
			if strings.Contains(lower, strings.ToLower(t)) {
				return true
			}
		}
	}
	return false
}

func GetAllLSRules() []models.LSRule {
	return []models.LSRule{
		{Code: "LS-01", Condition: "eGFR < 30", Blocked: "Protein > 0.6 g/kg/day", Severity: "HARD_STOP", Description: "CKD 4-5: high protein blocked"},
		{Code: "LS-02", Condition: "SBP > 180 OR DBP > 110", Blocked: "Vigorous exercise (MET > 6)", Severity: "HARD_STOP", Description: "Hypertensive crisis: vigorous exercise blocked"},
		{Code: "LS-03", Condition: "FBG < 70 in last 7d", Blocked: "Fasting or caloric restriction < 800 kcal/day", Severity: "HARD_STOP", Description: "Recent hypoglycemia: severe caloric restriction blocked"},
		{Code: "LS-04", Condition: "current_meds includes SU or insulin", Blocked: "Exercise without carb preload", Severity: "WARNING", Description: "SU/insulin users: exercise needs carb preload to prevent hypo"},
		{Code: "LS-05", Condition: "current_meds includes SGLT2i", Blocked: "Very-low-carb diet (< 50 g/day)", Severity: "WARNING", Description: "SGLT2i users: ketoacidosis risk with very-low-carb diet"},
		{Code: "LS-06", Condition: "retinopathy = PROLIFERATIVE", Blocked: "Resistance training / Valsalva maneuvers", Severity: "HARD_STOP", Description: "Proliferative retinopathy: resistance training blocked (vitreous hemorrhage risk)"},
		{Code: "LS-07", Condition: "neuropathy = true", Blocked: "Weight-bearing high-impact exercise", Severity: "WARNING", Description: "Peripheral neuropathy: high-impact exercise risks foot injury"},
		{Code: "LS-08", Condition: "pregnant = true AND diabetes", Blocked: "Caloric restriction < 1600 kcal/day", Severity: "HARD_STOP", Description: "Pregnancy + diabetes: caloric restriction blocked"},
		{Code: "LS-09", Condition: "potassium > 5.5", Blocked: "High-potassium foods", Severity: "HARD_STOP", Description: "Hyperkalemia: high-K foods blocked"},
		{Code: "LS-10", Condition: "cardiac_event within 30d", Blocked: "All exercise", Severity: "HARD_STOP", Description: "Recent cardiac event: all exercise blocked for 30 days"},
		{Code: "LS-11", Condition: "HbA1c > 13", Blocked: "Lifestyle-only treatment", Severity: "HARD_STOP", Description: "Extreme HbA1c: lifestyle-only not allowed"},
		{Code: "LS-12", Condition: "BMR < 1200 kcal", Blocked: "Caloric deficit > 300 kcal/day", Severity: "WARNING", Description: "Low BMR: excessive caloric deficit risks metabolic adaptation"},
		{Code: "LS-13", Condition: "gastroparesis = true", Blocked: "High-fiber meals > 15 g/serving", Severity: "WARNING", Description: "Gastroparesis: high-fiber meals risk bezoar formation and delayed gastric emptying"},
		{Code: "LS-14", Condition: "eating_disorder_history = true", Blocked: "Calorie counting or restrictive diets", Severity: "HARD_STOP", Description: "Eating disorder history: calorie counting and restrictive diets blocked"},
		{Code: "LS-15", Condition: "BMI < 22", Blocked: "Visceral Fat Reduction Protocol", Severity: "HARD_STOP", Description: "Underweight (South Asian BMI < 22): VFRP blocked to prevent muscle wasting and micronutrient deficiency"},
	}
}
