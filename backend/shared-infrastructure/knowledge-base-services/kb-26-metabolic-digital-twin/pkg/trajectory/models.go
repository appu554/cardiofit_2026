package trajectory

import (
	"kb-26-metabolic-digital-twin/internal/models"
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
type DomainSlope            = models.DomainSlope
type DivergencePattern      = models.DivergencePattern
type LeadingIndicator       = models.LeadingIndicator
type DomainCategoryCrossing = models.DomainCategoryCrossing
type DecomposedTrajectory   = models.DecomposedTrajectory
