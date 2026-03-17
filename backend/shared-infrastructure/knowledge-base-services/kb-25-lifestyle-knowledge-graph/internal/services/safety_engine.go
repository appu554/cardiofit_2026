package services

import (
	"context"

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
	default:
		return false
	}
}

func GetAllLSRules() []models.LSRule {
	return []models.LSRule{
		{Code: "LS-01", Condition: "eGFR < 30", Blocked: "Protein > 0.6 g/kg/day", Severity: "HARD_STOP", Description: "CKD 4-5: high protein blocked"},
		{Code: "LS-02", Condition: "SBP > 180 OR DBP > 110", Blocked: "Vigorous exercise (MET > 6)", Severity: "HARD_STOP", Description: "Hypertensive crisis: vigorous exercise blocked"},
		{Code: "LS-09", Condition: "potassium > 5.5", Blocked: "High-potassium foods", Severity: "HARD_STOP", Description: "Hyperkalemia: high-K foods blocked"},
		{Code: "LS-10", Condition: "cardiac_event_30d", Blocked: "All exercise", Severity: "HARD_STOP", Description: "Recent cardiac event: all exercise blocked"},
		{Code: "LS-11", Condition: "HbA1c > 13", Blocked: "Lifestyle-only treatment", Severity: "HARD_STOP", Description: "Extreme HbA1c: lifestyle-only not allowed"},
	}
}
