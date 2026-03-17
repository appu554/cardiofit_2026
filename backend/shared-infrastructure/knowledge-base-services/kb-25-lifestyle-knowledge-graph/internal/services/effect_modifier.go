package services

import (
	"kb-25-lifestyle-knowledge-graph/internal/clients"
	"kb-25-lifestyle-knowledge-graph/internal/models"
)

func ComputeModifiedEffect(base models.EffectDescriptor, patient *clients.PatientSnapshot, modifiers []models.ModifierRef) models.EffectDescriptor {
	result := base
	for _, mod := range modifiers {
		if evaluateCondition(mod.Condition, patient) {
			result.EffectSize *= mod.Multiplier
		}
	}
	return result
}

func AdherenceAdjust(effect, adherence float64) float64 {
	return effect * adherence
}

func evaluateCondition(condition string, patient *clients.PatientSnapshot) bool {
	switch condition {
	case "age > 65":
		return patient.Age > 65
	case "eGFR < 30":
		return patient.EGFR > 0 && patient.EGFR < 30
	case "eGFR < 60":
		return patient.EGFR > 0 && patient.EGFR < 60
	case "BMI > 35":
		return patient.BMI > 35
	case "HbA1c > 9":
		return patient.HbA1c > 9
	case "SBP > 160":
		return patient.SBP > 160
	default:
		return false
	}
}
