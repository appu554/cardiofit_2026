package trajectory

import (
	"kb-26-metabolic-digital-twin/internal/config"
	"kb-26-metabolic-digital-twin/internal/models"
	"kb-26-metabolic-digital-twin/internal/services"
)

// Domain identifiers.
type MHRIDomain = models.MHRIDomain

const (
	DomainGlucose    = models.DomainGlucose
	DomainCardio     = models.DomainCardio
	DomainBodyComp   = models.DomainBodyComp
	DomainBehavioral = models.DomainBehavioral
)

// AllMHRIDomains lists all four domains for deterministic iteration.
var AllMHRIDomains = models.AllMHRIDomains

// Trend constants.
const (
	TrendRapidImproving = models.TrendRapidImproving
	TrendImproving      = models.TrendImproving
	TrendStable         = models.TrendStable
	TrendDeclining      = models.TrendDeclining
	TrendRapidDeclining = models.TrendRapidDeclining
	TrendInsufficient   = models.TrendInsufficient
)

// Confidence constants.
const (
	ConfidenceHigh     = models.ConfidenceHigh
	ConfidenceModerate = models.ConfidenceModerate
	ConfidenceLow      = models.ConfidenceLow
)

// Direction constants.
const (
	DirectionWorsened = models.DirectionWorsened
	DirectionImproved = models.DirectionImproved
)

// Struct type aliases — fully interchangeable with the internal types.
type DomainTrajectoryPoint  = models.DomainTrajectoryPoint
type DomainSlope            = models.DomainSlope
type DivergencePattern      = models.DivergencePattern
type LeadingIndicator       = models.LeadingIndicator
type DomainCategoryCrossing = models.DomainCategoryCrossing
type DecomposedTrajectory   = models.DecomposedTrajectory

// Compute is the public wrapper around TrajectoryEngine.Compute,
// exposed here so cross-module consumers (e.g. KB-23 integration tests) can
// exercise the full KB-26 pipeline without importing internal packages.
// Constructs a default engine using DefaultTrajectoryThresholds on each call.
func Compute(patientID string, points []DomainTrajectoryPoint) DecomposedTrajectory {
	engine := services.NewTrajectoryEngine(config.DefaultTrajectoryThresholds(), nil, nil, nil)
	return engine.Compute(patientID, points)
}
