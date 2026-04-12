package services

import (
	"math"
	"sort"
	"time"

	"kb-26-metabolic-digital-twin/internal/models"
)

// MHRI domain weights (must match MHRI scorer weights).
var domainWeights = map[models.MHRIDomain]float64{
	models.DomainGlucose:    0.35,
	models.DomainCardio:     0.25,
	models.DomainBodyComp:   0.25,
	models.DomainBehavioral: 0.15,
}

// Trend thresholds (score units per day).
const (
	rapidImprovingThreshold = 1.0
	improvingThreshold      = 0.3
	decliningThreshold      = -0.3
	rapidDecliningThreshold = -1.0
)

// Category boundaries (MHRI score ranges).
const (
	categoryOptimal  = 70.0
	categoryMild     = 55.0
	categoryModerate = 40.0
)

// Leading indicator detection thresholds (see plan: Task 4 / leading_indicator config).
const (
	leadingIndicatorMinDataPoints   = 5    // need at least 5 points for lead-lag analysis
	leadingIndicatorMinBehavioralSlope = -0.5 // behavioral must be declining meaningfully
)

// ComputeDecomposedTrajectory computes per-domain OLS trajectories and derived analytics.
func ComputeDecomposedTrajectory(patientID string, points []models.DomainTrajectoryPoint) models.DecomposedTrajectory {
	result := models.DecomposedTrajectory{
		PatientID:    patientID,
		DataPoints:   len(points),
		ComputedAt:   time.Now(),
		DomainSlopes: make(map[models.MHRIDomain]models.DomainSlope),
	}

	if len(points) < 2 {
		result.CompositeTrend = models.TrendInsufficient
		for _, d := range models.AllMHRIDomains {
			result.DomainSlopes[d] = models.DomainSlope{Domain: d, Trend: models.TrendInsufficient}
		}
		return result
	}

	// Sort by timestamp.
	sorted := make([]models.DomainTrajectoryPoint, len(points))
	copy(sorted, points)
	sortTrajectoryPoints(sorted)

	first, last := sorted[0], sorted[len(sorted)-1]
	result.WindowDays = int(last.Timestamp.Sub(first.Timestamp).Hours() / 24)

	// Compute composite trajectory.
	compositeScores := extractScores(sorted, func(p models.DomainTrajectoryPoint) float64 { return p.CompositeScore })
	compSlope, _ := computeOLSWithR2(sorted, compositeScores)
	result.CompositeSlope = roundTo3(compSlope)
	result.CompositeTrend = classifyDomainTrend(compSlope)
	result.CompositeStartScore = first.CompositeScore
	result.CompositeEndScore = last.CompositeScore

	// Compute per-domain trajectories.
	domainExtractors := map[models.MHRIDomain]func(models.DomainTrajectoryPoint) float64{
		models.DomainGlucose:    func(p models.DomainTrajectoryPoint) float64 { return p.GlucoseScore },
		models.DomainCardio:     func(p models.DomainTrajectoryPoint) float64 { return p.CardioScore },
		models.DomainBodyComp:   func(p models.DomainTrajectoryPoint) float64 { return p.BodyCompScore },
		models.DomainBehavioral: func(p models.DomainTrajectoryPoint) float64 { return p.BehavioralScore },
	}

	decliningCount := 0
	var maxWeightedDecline float64
	var dominantDriver *models.MHRIDomain

	for domain, extractor := range domainExtractors {
		scores := extractScores(sorted, extractor)
		slope, r2 := computeOLSWithR2(sorted, scores)

		ds := models.DomainSlope{
			Domain:      domain,
			SlopePerDay: roundTo3(slope),
			Trend:       classifyDomainTrend(slope),
			StartScore:  scores[0],
			EndScore:    scores[len(scores)-1],
			DeltaScore:  roundTo1(scores[len(scores)-1] - scores[0]),
			R2:          roundTo3(r2),
			Confidence:  classifyR2Confidence(r2),
		}
		result.DomainSlopes[domain] = ds

		if slope < decliningThreshold {
			decliningCount++
		}

		weightedDecline := math.Abs(slope) * domainWeights[domain]
		if slope < 0 && weightedDecline > maxWeightedDecline {
			maxWeightedDecline = weightedDecline
			d := domain
			dominantDriver = &d
		}
	}

	result.DomainsDeteriorating = decliningCount
	result.ConcordantDeterioration = decliningCount >= 2

	// Dominant driver calculation.
	if dominantDriver != nil && result.CompositeSlope < 0 {
		result.DominantDriver = dominantDriver
		totalWeightedDecline := 0.0
		for domain, ds := range result.DomainSlopes {
			if ds.SlopePerDay < 0 {
				totalWeightedDecline += math.Abs(ds.SlopePerDay) * domainWeights[domain]
			}
		}
		if totalWeightedDecline > 0 {
			result.DriverContribution = roundTo1((maxWeightedDecline / totalWeightedDecline) * 100)
		}
	}

	// Detect divergence patterns (defined in domain_divergence.go).
	result.Divergences = detectDivergences(result.DomainSlopes)
	result.HasDiscordantTrend = len(result.Divergences) > 0

	// Detect domain category crossings.
	result.DomainCrossings = detectDomainCrossings(sorted, domainExtractors)

	// Detect behavioral leading indicator.
	result.LeadingIndicators = detectLeadingIndicators(sorted, result.DomainSlopes)

	return result
}

// computeOLSWithR2 runs OLS linear regression returning slope (per day) and R².
func computeOLSWithR2(points []models.DomainTrajectoryPoint, scores []float64) (float64, float64) {
	if len(points) < 2 {
		return 0, 0
	}

	baseTime := points[0].Timestamp
	n := float64(len(points))
	var sumX, sumY, sumXY, sumX2, sumY2 float64

	for i, pt := range points {
		x := pt.Timestamp.Sub(baseTime).Hours() / 24.0
		y := scores[i]
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
		sumY2 += y * y
	}

	denom := n*sumX2 - sumX*sumX
	if math.Abs(denom) < 1e-10 {
		return 0, 0
	}

	slope := (n*sumXY - sumX*sumY) / denom
	intercept := (sumY - slope*sumX) / n

	meanY := sumY / n
	ssTot := sumY2 - n*meanY*meanY
	ssRes := 0.0
	for i, pt := range points {
		x := pt.Timestamp.Sub(baseTime).Hours() / 24.0
		predicted := intercept + slope*x
		residual := scores[i] - predicted
		ssRes += residual * residual
	}

	r2 := 0.0
	if ssTot > 1e-10 {
		r2 = 1 - (ssRes / ssTot)
		if r2 < 0 {
			r2 = 0
		}
	}

	return slope, r2
}

func classifyDomainTrend(slopePerDay float64) string {
	switch {
	case slopePerDay > rapidImprovingThreshold:
		return models.TrendRapidImproving
	case slopePerDay > improvingThreshold:
		return models.TrendImproving
	case slopePerDay >= decliningThreshold:
		return models.TrendStable
	case slopePerDay >= rapidDecliningThreshold:
		return models.TrendDeclining
	default:
		return models.TrendRapidDeclining
	}
}

func classifyR2Confidence(r2 float64) string {
	if r2 >= 0.5 {
		return models.ConfidenceHigh
	}
	if r2 >= 0.25 {
		return models.ConfidenceModerate
	}
	return models.ConfidenceLow
}

func detectDomainCrossings(points []models.DomainTrajectoryPoint, extractors map[models.MHRIDomain]func(models.DomainTrajectoryPoint) float64) []models.DomainCategoryCrossing {
	if len(points) < 2 {
		return nil
	}

	first := points[0]
	last := points[len(points)-1]
	var crossings []models.DomainCategoryCrossing

	for _, domain := range models.AllMHRIDomains {
		extractor, ok := extractors[domain]
		if !ok {
			continue
		}
		startScore := extractor(first)
		endScore := extractor(last)
		startCat := categorizeDomainScore(startScore)
		endCat := categorizeDomainScore(endScore)

		if startCat != endCat {
			direction := models.DirectionImproved
			if endScore < startScore {
				direction = models.DirectionWorsened
			}
			crossings = append(crossings, models.DomainCategoryCrossing{
				Domain:       domain,
				PrevCategory: startCat,
				CurrCategory: endCat,
				Direction:    direction,
				CrossingDate: last.Timestamp,
			})
		}
	}

	return crossings
}

func detectLeadingIndicators(points []models.DomainTrajectoryPoint, slopes map[models.MHRIDomain]models.DomainSlope) []models.LeadingIndicator {
	if len(points) < leadingIndicatorMinDataPoints {
		return nil
	}

	behSlope := slopes[models.DomainBehavioral]
	if behSlope.SlopePerDay >= leadingIndicatorMinBehavioralSlope {
		return nil
	}

	var lagging []models.MHRIDomain
	for _, domain := range []models.MHRIDomain{models.DomainGlucose, models.DomainCardio} {
		ds := slopes[domain]
		if ds.SlopePerDay < decliningThreshold {
			if behSlope.DeltaScore < ds.DeltaScore {
				lagging = append(lagging, domain)
			}
		}
	}

	if len(lagging) == 0 {
		return nil
	}

	return []models.LeadingIndicator{{
		LeadingDomain:  models.DomainBehavioral,
		LaggingDomains: lagging,
		Confidence:     models.ConfidenceModerate,
		Interpretation: "Behavioral domain decline preceded clinical domain deterioration — engagement collapse may be driving worsening outcomes",
	}}
}

func categorizeDomainScore(score float64) string {
	if score >= categoryOptimal {
		return "OPTIMAL"
	}
	if score >= categoryMild {
		return "MILD"
	}
	if score >= categoryModerate {
		return "MODERATE"
	}
	return "HIGH"
}

func extractScores(points []models.DomainTrajectoryPoint, extractor func(models.DomainTrajectoryPoint) float64) []float64 {
	scores := make([]float64, len(points))
	for i, p := range points {
		scores[i] = extractor(p)
	}
	return scores
}

func sortTrajectoryPoints(pts []models.DomainTrajectoryPoint) {
	sort.Slice(pts, func(i, j int) bool {
		return pts[i].Timestamp.Before(pts[j].Timestamp)
	})
}

func roundTo3(v float64) float64 { return math.Round(v*1000) / 1000 }
func roundTo1(v float64) float64 { return math.Round(v*10) / 10 }
