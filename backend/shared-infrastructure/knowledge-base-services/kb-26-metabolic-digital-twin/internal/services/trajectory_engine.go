package services

import (
	"math"
	"sort"
	"time"

	"kb-26-metabolic-digital-twin/internal/config"
	"kb-26-metabolic-digital-twin/internal/models"
)

// TrajectoryEngine computes per-domain MHRI trajectories. Constructed with
// a TrajectoryThresholds value; all classification logic reads from this
// config so callers can override per market.
type TrajectoryEngine struct {
	thresholds config.TrajectoryThresholds
}

// NewTrajectoryEngine constructs an engine with the given thresholds.
// Pass config.DefaultTrajectoryThresholds() for the canonical Phase 0 values.
func NewTrajectoryEngine(thresholds config.TrajectoryThresholds) *TrajectoryEngine {
	return &TrajectoryEngine{thresholds: thresholds}
}

// Compute computes per-domain OLS trajectories and derived analytics.
func (e *TrajectoryEngine) Compute(patientID string, points []models.DomainTrajectoryPoint) models.DecomposedTrajectory {
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
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.Before(sorted[j].Timestamp)
	})

	first, last := sorted[0], sorted[len(sorted)-1]
	result.WindowDays = int(math.Round(last.Timestamp.Sub(first.Timestamp).Hours() / 24))

	// Composite trajectory.
	compositeScores := extractScores(sorted, func(p models.DomainTrajectoryPoint) float64 { return p.CompositeScore })
	compSlope, _ := e.computeOLSWithR2(sorted, compositeScores)
	result.CompositeSlope = roundTo3(compSlope)
	result.CompositeTrend = e.classifyTrend(compSlope)
	result.CompositeStartScore = first.CompositeScore
	result.CompositeEndScore = last.CompositeScore

	// Per-domain trajectories.
	domainExtractors := map[models.MHRIDomain]func(models.DomainTrajectoryPoint) float64{
		models.DomainGlucose:    func(p models.DomainTrajectoryPoint) float64 { return p.GlucoseScore },
		models.DomainCardio:     func(p models.DomainTrajectoryPoint) float64 { return p.CardioScore },
		models.DomainBodyComp:   func(p models.DomainTrajectoryPoint) float64 { return p.BodyCompScore },
		models.DomainBehavioral: func(p models.DomainTrajectoryPoint) float64 { return p.BehavioralScore },
	}

	decliningCount := 0
	var maxWeightedDecline float64
	var dominantDriver *models.MHRIDomain

	for _, domain := range models.AllMHRIDomains {
		extractor := domainExtractors[domain]
		scores := extractScores(sorted, extractor)
		slope, r2 := e.computeOLSWithR2(sorted, scores)

		ds := models.DomainSlope{
			Domain:      domain,
			SlopePerDay: roundTo3(slope),
			Trend:       e.classifyTrend(slope),
			StartScore:  scores[0],
			EndScore:    scores[len(scores)-1],
			DeltaScore:  roundTo1(scores[len(scores)-1] - scores[0]),
			R2:          roundTo3(r2),
			Confidence:  e.classifyConfidence(r2),
		}
		result.DomainSlopes[domain] = ds

		if slope < e.thresholds.Concordant.MinSlopePerDomain {
			decliningCount++
		}

		weight := e.thresholds.Driver.WeightMap[domain]
		weightedDecline := math.Abs(slope) * weight
		if slope < e.thresholds.Trend.Declining && weightedDecline > maxWeightedDecline {
			maxWeightedDecline = weightedDecline
			d := domain
			dominantDriver = &d
		}
	}

	result.DomainsDeteriorating = decliningCount
	result.ConcordantDeterioration = decliningCount >= e.thresholds.Concordant.MinDomainsDeclining

	// Dominant driver calculation.
	if dominantDriver != nil && result.CompositeSlope < 0 {
		result.DominantDriver = dominantDriver
		totalWeightedDecline := 0.0
		for d, ds := range result.DomainSlopes {
			if ds.SlopePerDay < e.thresholds.Trend.Declining {
				totalWeightedDecline += math.Abs(ds.SlopePerDay) * e.thresholds.Driver.WeightMap[d]
			}
		}
		if totalWeightedDecline > 0 {
			result.DriverContribution = roundTo1((maxWeightedDecline / totalWeightedDecline) * 100)
		}
	}

	// Divergence (method on engine so it reads divergence thresholds from config).
	result.Divergences = e.detectDivergences(result.DomainSlopes)
	result.HasDiscordantTrend = len(result.Divergences) > 0

	// Domain category crossings.
	result.DomainCrossings = e.detectDomainCrossings(sorted, domainExtractors)

	// Behavioral leading indicator.
	result.LeadingIndicators = e.detectLeadingIndicators(sorted, result.DomainSlopes)

	return result
}

// computeOLSWithR2 runs OLS linear regression returning slope (per day) and R².
// Uses the numerically stable two-pass form for ssTot. (Item 3 fix.)
func (e *TrajectoryEngine) computeOLSWithR2(points []models.DomainTrajectoryPoint, scores []float64) (float64, float64) {
	if len(points) < 2 {
		return 0, 0
	}

	baseTime := points[0].Timestamp
	n := float64(len(points))
	var sumX, sumY, sumXY, sumX2 float64

	for i, pt := range points {
		x := pt.Timestamp.Sub(baseTime).Hours() / 24.0
		y := scores[i]
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	denom := n*sumX2 - sumX*sumX
	if math.Abs(denom) < 1e-10 {
		return 0, 0
	}

	slope := (n*sumXY - sumX*sumY) / denom
	intercept := (sumY - slope*sumX) / n

	// Two-pass ssTot (numerically stable — Item 3 fix).
	meanY := sumY / n
	ssTot := 0.0
	ssRes := 0.0
	for i, pt := range points {
		x := pt.Timestamp.Sub(baseTime).Hours() / 24.0
		predicted := intercept + slope*x
		residual := scores[i] - predicted
		ssRes += residual * residual

		delta := scores[i] - meanY
		ssTot += delta * delta
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

func (e *TrajectoryEngine) classifyTrend(slopePerDay float64) string {
	switch {
	case slopePerDay > e.thresholds.Trend.RapidImproving:
		return models.TrendRapidImproving
	case slopePerDay > e.thresholds.Trend.Improving:
		return models.TrendImproving
	case slopePerDay >= e.thresholds.Trend.Declining:
		return models.TrendStable
	case slopePerDay >= e.thresholds.Trend.RapidDeclining:
		return models.TrendDeclining
	default:
		return models.TrendRapidDeclining
	}
}

func (e *TrajectoryEngine) classifyConfidence(r2 float64) string {
	if r2 >= e.thresholds.RSquared.High {
		return models.ConfidenceHigh
	}
	if r2 >= e.thresholds.RSquared.Moderate {
		return models.ConfidenceModerate
	}
	return models.ConfidenceLow
}

func (e *TrajectoryEngine) categorizeDomainScore(score float64) string {
	if score >= e.thresholds.CategoryBoundaries.Optimal {
		return "OPTIMAL"
	}
	if score >= e.thresholds.CategoryBoundaries.Mild {
		return "MILD"
	}
	if score >= e.thresholds.CategoryBoundaries.Moderate {
		return "MODERATE"
	}
	return "HIGH"
}

func (e *TrajectoryEngine) detectDomainCrossings(points []models.DomainTrajectoryPoint, extractors map[models.MHRIDomain]func(models.DomainTrajectoryPoint) float64) []models.DomainCategoryCrossing {
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
		startCat := e.categorizeDomainScore(startScore)
		endCat := e.categorizeDomainScore(endScore)

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

func (e *TrajectoryEngine) detectLeadingIndicators(points []models.DomainTrajectoryPoint, slopes map[models.MHRIDomain]models.DomainSlope) []models.LeadingIndicator {
	if len(points) < e.thresholds.LeadingIndicator.MinDataPoints {
		return nil
	}

	behSlope := slopes[models.DomainBehavioral]
	if behSlope.SlopePerDay >= e.thresholds.LeadingIndicator.MinBehavioralDeclineSlope {
		return nil
	}

	var lagging []models.MHRIDomain
	for _, domain := range []models.MHRIDomain{models.DomainGlucose, models.DomainCardio} {
		ds := slopes[domain]
		if ds.SlopePerDay < e.thresholds.Trend.Declining {
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

func extractScores(points []models.DomainTrajectoryPoint, extractor func(models.DomainTrajectoryPoint) float64) []float64 {
	scores := make([]float64, len(points))
	for i, p := range points {
		scores[i] = extractor(p)
	}
	return scores
}

func roundTo3(v float64) float64 { return math.Round(v*1000) / 1000 }
func roundTo1(v float64) float64 { return math.Round(v*10) / 10 }
