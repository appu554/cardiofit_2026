package services

import (
	"context"
	"math"
	"sort"
	"strconv"
	"time"

	"go.uber.org/zap"

	"kb-26-metabolic-digital-twin/internal/config"
	"kb-26-metabolic-digital-twin/internal/metrics"
	"kb-26-metabolic-digital-twin/internal/models"
)

// TrajectoryEngine computes per-domain MHRI trajectories. Constructed with
// a TrajectoryThresholds value; all classification logic reads from this
// config so callers can override per market.
type TrajectoryEngine struct {
	thresholds config.TrajectoryThresholds
	metrics    *metrics.TrajectoryMetrics // optional, nil-safe
	publisher  TrajectoryPublisher
	logger     *zap.Logger
}

// NewTrajectoryEngine constructs an engine with the given thresholds.
// Metrics, publisher, and logger are nil-safe; pass nil for tests that
// don't care about telemetry/eventing.
func NewTrajectoryEngine(
	thresholds config.TrajectoryThresholds,
	m *metrics.TrajectoryMetrics,
	publisher TrajectoryPublisher,
	logger *zap.Logger,
) *TrajectoryEngine {
	if publisher == nil {
		publisher = NoopTrajectoryPublisher{}
	}
	return &TrajectoryEngine{
		thresholds: thresholds,
		metrics:    m,
		publisher:  publisher,
		logger:     logger,
	}
}

// Compute computes per-domain OLS trajectories and derived analytics.
func (e *TrajectoryEngine) Compute(patientID string, points []models.DomainTrajectoryPoint) models.DecomposedTrajectory {
	start := time.Now()
	defer func() {
		if e.metrics != nil {
			e.metrics.ComputeDuration.Observe(float64(time.Since(start).Milliseconds()))
		}
	}()

	result := models.DecomposedTrajectory{
		PatientID:    patientID,
		DataPoints:   len(points),
		ComputedAt:   time.Now(),
		DomainSlopes: make(map[models.MHRIDomain]models.DomainSlope),
	}

	if len(points) < 2 {
		if e.metrics != nil {
			e.metrics.InsufficientData.Inc()
		}
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
	if result.ConcordantDeterioration && e.metrics != nil {
		e.metrics.ConcordantDeterioration.WithLabelValues(strconv.Itoa(decliningCount)).Inc()
	}

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
	if e.metrics != nil {
		for _, div := range result.Divergences {
			e.metrics.DivergenceTotal.WithLabelValues(string(div.ImprovingDomain), string(div.DecliningDomain)).Inc()
		}
	}

	// Domain category crossings.
	result.DomainCrossings = e.detectDomainCrossings(sorted, domainExtractors)
	if e.metrics != nil {
		for _, c := range result.DomainCrossings {
			e.metrics.DomainCrossingTotal.WithLabelValues(string(c.Domain), c.Direction).Inc()
		}
	}

	// Behavioral leading indicator.
	result.LeadingIndicators = e.detectLeadingIndicators(sorted, result.DomainSlopes)
	if e.metrics != nil {
		for _, lead := range result.LeadingIndicators {
			for _, lagging := range lead.LaggingDomains {
				e.metrics.LeadingIndicatorTotal.WithLabelValues(string(lagging)).Inc()
			}
		}
	}

	// Publish event for downstream consumers (Module 13 Flink state-sync).
	// Publish failure is non-fatal — log and continue.
	event := NewDomainTrajectoryComputedEvent(&result)
	if err := e.publisher.Publish(context.Background(), event); err != nil && e.logger != nil {
		e.logger.Warn("failed to publish trajectory event", zap.Error(err))
	}

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

	// Phase 10 V4-5 completion: estimate lead days from the temporal
	// offset between behavioral decline onset and clinical decline onset.
	leadDays := e.estimateLeadDays(points, models.DomainBehavioral, lagging)

	// Build causal chain: order ALL declining domains by their decline
	// onset time (earliest first). This implements V4-5 Scenario 4.
	causalChain := e.buildCausalChain(points, slopes)

	return []models.LeadingIndicator{{
		LeadingDomain:  models.DomainBehavioral,
		LaggingDomains: lagging,
		LeadDays:       leadDays,
		CausalChain:    causalChain,
		Confidence:     models.ConfidenceModerate,
		Interpretation: "Behavioral domain decline preceded clinical domain deterioration — engagement collapse may be driving worsening outcomes",
	}}
}

// domainExtractorMap returns the extraction function for each domain.
// Shared by estimateLeadDays and buildCausalChain.
func domainExtractorMap() map[models.MHRIDomain]func(models.DomainTrajectoryPoint) float64 {
	return map[models.MHRIDomain]func(models.DomainTrajectoryPoint) float64{
		models.DomainGlucose:    func(p models.DomainTrajectoryPoint) float64 { return p.GlucoseScore },
		models.DomainCardio:     func(p models.DomainTrajectoryPoint) float64 { return p.CardioScore },
		models.DomainBodyComp:   func(p models.DomainTrajectoryPoint) float64 { return p.BodyCompScore },
		models.DomainBehavioral: func(p models.DomainTrajectoryPoint) float64 { return p.BehavioralScore },
	}
}

// estimateLeadDays computes the approximate number of days the leading
// domain started declining before the earliest lagging domain.
// Uses a 5% decline from start score as the onset marker.
// Phase 10 V4-5 completion.
func (e *TrajectoryEngine) estimateLeadDays(
	points []models.DomainTrajectoryPoint,
	leadingDomain models.MHRIDomain,
	laggingDomains []models.MHRIDomain,
) int {
	if len(points) < 3 {
		return 0
	}
	extractors := domainExtractorMap()

	onset := func(domain models.MHRIDomain) int {
		ext := extractors[domain]
		if ext == nil {
			return len(points) - 1
		}
		startScore := ext(points[0])
		threshold := startScore * 0.95
		for i, p := range points {
			if ext(p) < threshold {
				return i
			}
		}
		return len(points) - 1
	}

	leadOnset := onset(leadingDomain)
	earliestLag := len(points) - 1
	for _, domain := range laggingDomains {
		lagOnset := onset(domain)
		if lagOnset < earliestLag {
			earliestLag = lagOnset
		}
	}

	leadDays := earliestLag - leadOnset
	if leadDays < 0 {
		leadDays = 0
	}
	return leadDays
}

// buildCausalChain orders all declining domains by their decline
// onset time (earliest first). Implements V4-5 Scenario 4: "body
// comp deteriorated first, then glucose, then cardio — pointing
// to weight gain as the root cause." Phase 10 V4-5 completion.
func (e *TrajectoryEngine) buildCausalChain(
	points []models.DomainTrajectoryPoint,
	slopes map[models.MHRIDomain]models.DomainSlope,
) []models.MHRIDomain {
	if len(points) < 3 {
		return nil
	}

	type domainOnset struct {
		domain models.MHRIDomain
		onset  int
	}

	extractors := domainExtractorMap()
	var declining []domainOnset

	for _, domain := range models.AllMHRIDomains {
		ds := slopes[domain]
		if ds.SlopePerDay >= e.thresholds.Trend.Declining {
			continue
		}
		ext := extractors[domain]
		startScore := ext(points[0])
		threshold := startScore * 0.95
		onsetIdx := len(points) - 1
		for i, p := range points {
			if ext(p) < threshold {
				onsetIdx = i
				break
			}
		}
		declining = append(declining, domainOnset{domain: domain, onset: onsetIdx})
	}

	if len(declining) < 2 {
		return nil
	}

	// Sort by onset index (earliest first)
	for i := 0; i < len(declining)-1; i++ {
		for j := i + 1; j < len(declining); j++ {
			if declining[j].onset < declining[i].onset {
				declining[i], declining[j] = declining[j], declining[i]
			}
		}
	}

	chain := make([]models.MHRIDomain, len(declining))
	for i, d := range declining {
		chain[i] = d.domain
	}
	return chain
}

// CheckSeasonalSuppression checks whether any seasonal window from
// the market config is currently active and whether the declining
// domains match the window's affected domains. Returns a suppression
// context for downstream card urgency downgrade.
// Phase 10 V4-5 completion (India-specific consideration).
func CheckSeasonalSuppression(
	result *models.DecomposedTrajectory,
	seasonalWindows []SeasonalWindow,
	now time.Time,
) *models.SeasonalSuppressionContext {
	if result == nil || len(seasonalWindows) == 0 {
		return nil
	}
	for _, w := range seasonalWindows {
		if !w.IsActive(now) {
			continue
		}
		for _, domain := range w.AffectedDomains {
			ds, ok := result.DomainSlopes[domain]
			if ok && ds.SlopePerDay < -0.3 {
				return &models.SeasonalSuppressionContext{
					WindowName:      w.Name,
					AffectedDomains: w.AffectedDomains,
					Mode:            w.Mode,
					Rationale:       w.Rationale,
				}
			}
		}
	}
	return nil
}

// SeasonalWindow represents one window from the market config's
// seasonal_calendar.yaml. Phase 10 V4-5 completion.
type SeasonalWindow struct {
	Name            string              `yaml:"name"`
	Start           string              `yaml:"start,omitempty"`
	End             string              `yaml:"end,omitempty"`
	StartDOY        int                 `yaml:"start_doy,omitempty"`
	EndDOY          int                 `yaml:"end_doy,omitempty"`
	AffectedDomains []models.MHRIDomain `yaml:"affected_domains"`
	Mode            string              `yaml:"mode"`
	Rationale       string              `yaml:"rationale"`
}

// IsActive returns true when now falls within the seasonal window.
func (w SeasonalWindow) IsActive(now time.Time) bool {
	if w.Start != "" && w.End != "" {
		start, err1 := time.Parse("2006-01-02", w.Start)
		end, err2 := time.Parse("2006-01-02", w.End)
		if err1 == nil && err2 == nil {
			return !now.Before(start) && !now.After(end)
		}
	}
	if w.StartDOY > 0 && w.EndDOY > 0 {
		doy := now.YearDay()
		if w.StartDOY <= w.EndDOY {
			return doy >= w.StartDOY && doy <= w.EndDOY
		}
		return doy >= w.StartDOY || doy <= w.EndDOY
	}
	return false
}

// ComputeSecondDerivatives computes the trajectory-of-the-trajectory:
// the rate of change of each domain's slope over time. Accepts recent
// snapshots from domain_trajectory_history (caller fetches from DB)
// and computes OLS regression over each domain's slope values.
//
// V4-5 Phase 3: answers "is the decline accelerating or decelerating?"
//
// A positive slope_of_slope on a declining domain means the decline
// is decelerating (the intervention may be working — the patient is
// still getting worse but more slowly). A negative slope_of_slope
// means the decline is accelerating (the intervention is failing).
//
// Pure function — no DB dependency. The caller provides the history
// snapshots; the engine computes the second derivatives.
//
// Minimum 3 snapshots required for a meaningful second derivative.
// Returns nil map when insufficient data.
func (e *TrajectoryEngine) ComputeSecondDerivatives(
	snapshots []models.DomainTrajectoryHistory,
) map[models.MHRIDomain]models.SecondDerivativeResult {
	if len(snapshots) < 3 {
		return nil
	}

	// Sort snapshots by date ascending (oldest first).
	sorted := make([]models.DomainTrajectoryHistory, len(snapshots))
	copy(sorted, snapshots)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].SnapshotDate.Before(sorted[j].SnapshotDate)
	})

	results := make(map[models.MHRIDomain]models.SecondDerivativeResult)

	type domainSlopeExtractor struct {
		domain    models.MHRIDomain
		extractor func(models.DomainTrajectoryHistory) float64
	}

	extractors := []domainSlopeExtractor{
		{models.DomainGlucose, func(h models.DomainTrajectoryHistory) float64 { return h.GlucoseSlope }},
		{models.DomainCardio, func(h models.DomainTrajectoryHistory) float64 { return h.CardioSlope }},
		{models.DomainBodyComp, func(h models.DomainTrajectoryHistory) float64 { return h.BodyCompSlope }},
		{models.DomainBehavioral, func(h models.DomainTrajectoryHistory) float64 { return h.BehavioralSlope }},
	}

	for _, ext := range extractors {
		slopes := make([]float64, len(sorted))
		for i, snap := range sorted {
			slopes[i] = ext.extractor(snap)
		}

		// OLS regression: slope index (0, 1, 2, ...) vs slope value.
		// The slope of this regression is the second derivative.
		n := float64(len(slopes))
		var sumX, sumY, sumXY, sumX2 float64
		for i, y := range slopes {
			x := float64(i)
			sumX += x
			sumY += y
			sumXY += x * y
			sumX2 += x * x
		}
		denom := n*sumX2 - sumX*sumX
		if denom == 0 {
			continue
		}
		slopeOfSlope := (n*sumXY - sumX*sumY) / denom

		// Classify the second derivative.
		interp := e.classifySecondDerivative(slopeOfSlope, slopes[len(slopes)-1])

		// Confidence based on snapshot count.
		confidence := models.ConfidenceLow
		if len(sorted) >= 5 {
			confidence = models.ConfidenceHigh
		} else if len(sorted) >= 3 {
			confidence = models.ConfidenceModerate
		}

		results[ext.domain] = models.SecondDerivativeResult{
			Domain:         ext.domain,
			SlopeOfSlope:   roundTo3(slopeOfSlope),
			Interpretation: interp,
			SnapshotsUsed:  len(sorted),
			Confidence:     confidence,
		}
	}

	return results
}

// classifySecondDerivative interprets the second derivative in
// clinical context. The combination of the current slope (first
// derivative) and the slope_of_slope (second derivative) determines
// the interpretation.
func (e *TrajectoryEngine) classifySecondDerivative(slopeOfSlope, currentSlope float64) string {
	threshold := 0.05 // minimum meaningful second derivative

	if math.Abs(slopeOfSlope) < threshold {
		if currentSlope < e.thresholds.Trend.Declining {
			return "STABLE_DECLINE"
		}
		return "STABLE"
	}

	if currentSlope < e.thresholds.Trend.Declining {
		// Domain is currently declining
		if slopeOfSlope > 0 {
			return "DECELERATING_DECLINE" // decline is slowing — intervention may be working
		}
		return "ACCELERATING_DECLINE" // decline is speeding up — intervention failing
	}

	if currentSlope > e.thresholds.Trend.Improving {
		// Domain is currently improving
		if slopeOfSlope > 0 {
			return "ACCELERATING_IMPROVEMENT" // improvement is speeding up
		}
		return "DECELERATING_IMPROVEMENT" // improvement is slowing
	}

	return "STABLE"
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
