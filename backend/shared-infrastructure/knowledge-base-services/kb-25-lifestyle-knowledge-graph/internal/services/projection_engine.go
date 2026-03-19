package services

import (
	"kb-25-lifestyle-knowledge-graph/internal/clients"
	"kb-25-lifestyle-knowledge-graph/internal/graph"
	"kb-25-lifestyle-knowledge-graph/internal/models"

	"go.uber.org/zap"
)

// ProjectionEngine computes forward projections for combined lifestyle
// protocols (PRP, VFRP) with synergy multipliers and attribution.
type ProjectionEngine struct {
	graphClient graph.GraphClient
	logger      *zap.Logger
}

// NewProjectionEngine creates a new engine. Both params may be nil for tests.
func NewProjectionEngine(graphClient graph.GraphClient, logger *zap.Logger) *ProjectionEngine {
	return &ProjectionEngine{graphClient: graphClient, logger: logger}
}

// Protocol effect constants (84-day reference period).
const (
	prpFBGEffect             = -12.5
	prpSBPEffect             = -2.0
	vfrpFBGEffect            = -8.0
	vfrpPPBGEffect           = -22.0
	vfrpWaistEffect          = -4.0
	vfrpSBPEffect            = -5.6
	vfrpTGEffect             = -35.0
	synergyMultiplierPRPVFRP = 1.15
)

// defaultModifiers are the population-level patient-specific adjustments applied
// when patient context (Age > 0) is present in the request.
var defaultModifiers = []models.ModifierRef{
	{ContextCode: "AGE_GT_65", Multiplier: 0.75, Condition: "age > 65"},
	{ContextCode: "CKD_STAGE_45", Multiplier: 0.50, Condition: "eGFR < 30"},
	{ContextCode: "OBESITY_CLASS2", Multiplier: 0.85, Condition: "BMI > 35"},
}

// applyPatientModifiers builds a PatientSnapshot from the request fields and
// returns a modifier multiplier to apply to all effect sizes. If no patient
// context is provided (Age == 0) it returns 1.0 (no-op).
func applyPatientModifiers(req models.CombinedProjectionRequest, baseEffect float64) float64 {
	if req.Age == 0 {
		return baseEffect
	}

	patient := &clients.PatientSnapshot{
		Age:   req.Age,
		EGFR:  req.EGFR,
		BMI:   req.BMI,
		HbA1c: req.HbA1c,
		SBP:   req.SBP,
	}

	desc := models.EffectDescriptor{EffectSize: baseEffect}
	modified := ComputeModifiedEffect(desc, patient, defaultModifiers)
	effect := modified.EffectSize

	if req.Adherence > 0 {
		effect = AdherenceAdjust(effect, req.Adherence)
	}

	return effect
}

// ProjectCombined computes the combined projection for active protocols.
func (p *ProjectionEngine) ProjectCombined(req models.CombinedProjectionRequest) *models.CombinedProjectionResult {
	if req.Days == 0 {
		req.Days = 84
	}

	hasPRP := contains(req.ActiveProtocols, "M3-PRP")
	hasVFRP := contains(req.ActiveProtocols, "M3-VFRP")

	result := &models.CombinedProjectionResult{
		PatientID:         req.PatientID,
		Days:              req.Days,
		ActiveProtocols:   req.ActiveProtocols,
		SynergyMultiplier: 1.0,
	}

	timeScale := float64(req.Days) / 84.0
	if timeScale > 1.0 {
		timeScale = 1.0
	}

	if hasPRP {
		prpFBG := applyPatientModifiers(req, prpFBGEffect) * timeScale
		prpSBP := applyPatientModifiers(req, prpSBPEffect) * timeScale
		result.FBGDelta += prpFBG
		result.SBPDelta += prpSBP
		result.Attribution = append(result.Attribution, models.ProtocolAttribution{
			Protocol:        "M3-PRP",
			FBGContribution: prpFBG,
		})
	}

	if hasVFRP {
		vfrpFBG := applyPatientModifiers(req, vfrpFBGEffect) * timeScale
		result.FBGDelta += vfrpFBG
		result.PPBGDelta += applyPatientModifiers(req, vfrpPPBGEffect) * timeScale
		result.WaistDelta += applyPatientModifiers(req, vfrpWaistEffect) * timeScale
		result.SBPDelta += applyPatientModifiers(req, vfrpSBPEffect) * timeScale
		result.TGDelta += applyPatientModifiers(req, vfrpTGEffect) * timeScale
		result.Attribution = append(result.Attribution, models.ProtocolAttribution{
			Protocol:        "M3-VFRP",
			FBGContribution: vfrpFBG,
		})
	}

	if hasPRP && hasVFRP {
		result.SynergyMultiplier = synergyMultiplierPRPVFRP
		result.FBGDelta *= synergyMultiplierPRPVFRP
		result.SBPDelta *= synergyMultiplierPRPVFRP
		result.Label = "PRP+VFRP combined"
	} else if hasPRP {
		result.Label = "PRP-only"
	} else if hasVFRP {
		result.Label = "VFRP-only"
	}

	// Approximate HbA1c from FBG using Nathan formula constant.
	result.HbA1cDelta = result.FBGDelta / 28.7

	// Compute attribution fractions.
	totalFBG := 0.0
	for _, a := range result.Attribution {
		totalFBG += a.FBGContribution
	}
	if totalFBG != 0 {
		for i := range result.Attribution {
			result.Attribution[i].FractionOfTotal = result.Attribution[i].FBGContribution / totalFBG
		}
	}

	return result
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
