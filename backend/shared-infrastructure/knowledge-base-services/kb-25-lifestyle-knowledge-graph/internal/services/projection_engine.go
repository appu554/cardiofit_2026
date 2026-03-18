package services

import (
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
		result.FBGDelta += prpFBGEffect * timeScale
		result.SBPDelta += prpSBPEffect * timeScale
		result.Attribution = append(result.Attribution, models.ProtocolAttribution{
			Protocol:        "M3-PRP",
			FBGContribution: prpFBGEffect * timeScale,
		})
	}

	if hasVFRP {
		result.FBGDelta += vfrpFBGEffect * timeScale
		result.PPBGDelta += vfrpPPBGEffect * timeScale
		result.WaistDelta += vfrpWaistEffect * timeScale
		result.SBPDelta += vfrpSBPEffect * timeScale
		result.TGDelta += vfrpTGEffect * timeScale
		result.Attribution = append(result.Attribution, models.ProtocolAttribution{
			Protocol:        "M3-VFRP",
			FBGContribution: vfrpFBGEffect * timeScale,
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
