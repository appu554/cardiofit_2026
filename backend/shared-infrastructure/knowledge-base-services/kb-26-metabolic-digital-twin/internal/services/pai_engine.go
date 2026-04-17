package services

import (
	"math"
	"time"

	"kb-26-metabolic-digital-twin/internal/models"
)

// ComputePAI computes the composite Patient Acuity Index from all five
// dimensions (velocity, proximity, behavioral, context, attention) using
// weighted scoring, tier classification, dominant-dimension identification,
// and actionability context generation.
func ComputePAI(input models.PAIDimensionInput, cfg *PAIConfig) models.PAIScore {
	now := time.Now().UTC()

	// 1. Compute each dimension
	velScore := ComputeVelocityScore(input, cfg)
	proxScore := ComputeProximityScore(input, cfg)
	behScore := ComputeBehavioralScore(input, cfg)
	ctxScore := ComputeContextScore(input, cfg)
	attScore := ComputeAttentionScore(input, cfg)

	// 2. Weighted composite
	composite := velScore*cfg.VelocityWeight +
		proxScore*cfg.ProximityWeight +
		behScore*cfg.BehavioralWeight +
		ctxScore*cfg.ContextWeight +
		attScore*cfg.AttentionWeight
	composite = math.Min(math.Max(composite, 0), 100)
	composite = math.Round(composite*10) / 10

	// 3. Determine tier
	tier := determineTier(composite, cfg)

	// 4. Find dominant dimension
	scores := map[string]float64{
		"VELOCITY":   velScore * cfg.VelocityWeight,
		"PROXIMITY":  proxScore * cfg.ProximityWeight,
		"BEHAVIORAL": behScore * cfg.BehavioralWeight,
		"CONTEXT":    ctxScore * cfg.ContextWeight,
		"ATTENTION":  attScore * cfg.AttentionWeight,
	}
	dominant, contribution := findDominant(scores, composite)

	// 5. Generate actionability context
	reason, action, timeframe := generateActionContext(dominant, velScore, proxScore, behScore, ctxScore, attScore, input)
	escalation := determineEscalation(tier, input)

	// 6. Data freshness
	freshness := assessFreshness(input)

	// 7. Input source count
	sources := countInputSources(input)

	return models.PAIScore{
		PatientID:            input.PatientID,
		ComputedAt:           now,
		Score:                composite,
		Tier:                 string(tier),
		VelocityScore:        velScore,
		ProximityScore:       proxScore,
		BehavioralScore:      behScore,
		ContextScore:         ctxScore,
		AttentionScore:       attScore,
		DominantDimension:    dominant,
		DominantContribution: contribution,
		PrimaryReason:        reason,
		SuggestedAction:      action,
		SuggestedTimeframe:   timeframe,
		EscalationTier:       escalation,
		InputSources:         sources,
		DataFreshness:        freshness,
	}
}

// determineTier maps composite score to a PAITier using config thresholds.
func determineTier(score float64, cfg *PAIConfig) models.PAITier {
	switch {
	case score >= cfg.CriticalThreshold:
		return models.TierCritical
	case score >= cfg.HighThreshold:
		return models.TierHigh
	case score >= cfg.ModerateThreshold:
		return models.TierModerate
	case score >= cfg.LowThreshold:
		return models.TierLow
	default:
		return models.TierMinimal
	}
}

// findDominant returns the name of the highest weighted contributor and
// its percentage of the composite score.
func findDominant(weightedScores map[string]float64, composite float64) (string, float64) {
	if composite <= 0 {
		return "NONE", 0
	}

	bestName := "NONE"
	bestVal := 0.0
	for name, val := range weightedScores {
		if val > bestVal {
			bestVal = val
			bestName = name
		}
	}

	pct := math.Round((bestVal/composite)*1000) / 10 // one decimal place
	return bestName, pct
}

// generateActionContext produces (reason, action, timeframe) strings based
// on the dominant dimension and individual dimension scores.
func generateActionContext(dominant string, vel, prox, beh, ctx, att float64, input models.PAIDimensionInput) (string, string, string) {
	switch dominant {
	case "VELOCITY":
		reason := "Rapid multi-domain deterioration"
		if input.ConcordantDeterioration && input.DomainsDeterioriating >= 3 {
			reason = "Concordant deterioration across " + itoa(input.DomainsDeterioriating) + " domains"
		}
		return reason, "Review trajectory and consider intervention", "Within 24 hours"

	case "PROXIMITY":
		reason := proximityReason(input)
		return reason, "Order confirmatory labs", "Within 24 hours"

	case "BEHAVIORAL":
		return "Patient disengaging from care", "Contact patient, assess barriers", "Within 48 hours"

	case "CONTEXT":
		reason := "High clinical complexity"
		if input.IsPostDischarge30d {
			reason = "Post-discharge high-risk window"
		}
		return reason, "Schedule clinical review", "This week"

	case "ATTENTION":
		return "Extended gap since clinical review", "Schedule follow-up", "This week"

	default:
		return "Patient acuity requires monitoring", "Review patient status", "This week"
	}
}

// proximityReason selects the most specific proximity reason based on input values.
func proximityReason(input models.PAIDimensionInput) string {
	if input.CurrentEGFR != nil && *input.CurrentEGFR < 45 {
		return "eGFR approaching critical threshold"
	}
	if input.CurrentSBP != nil && *input.CurrentSBP >= 160 {
		return "Blood pressure approaching crisis level"
	}
	if input.CurrentHbA1c != nil && *input.CurrentHbA1c >= 8.0 {
		return "HbA1c above target threshold"
	}
	if input.CurrentPotassium != nil && *input.CurrentPotassium >= 5.5 {
		return "Potassium approaching dangerous level"
	}
	return "Lab values approaching clinical thresholds"
}

// determineEscalation maps tier + clinical context to an escalation level.
func determineEscalation(tier models.PAITier, input models.PAIDimensionInput) string {
	if tier == models.TierCritical {
		// SAFETY escalation for critical patients with additional risk factors
		if input.IsPostDischarge30d {
			return "SAFETY"
		}
		if input.CurrentEGFR != nil && *input.CurrentEGFR < 30 {
			return "SAFETY"
		}
		if input.IsAcutelyIll {
			return "SAFETY"
		}
		return "IMMEDIATE"
	}
	if tier == models.TierHigh {
		return "URGENT"
	}
	return "ROUTINE"
}

// assessFreshness returns "CURRENT" if any clinical value is present, "STALE" otherwise.
func assessFreshness(input models.PAIDimensionInput) string {
	if input.MHRICompositeSlope != nil ||
		input.CurrentEGFR != nil ||
		input.CurrentHbA1c != nil ||
		input.CurrentSBP != nil ||
		input.CurrentDBP != nil ||
		input.CurrentPotassium != nil ||
		input.CurrentTBRL2Pct != nil ||
		input.CurrentTIR != nil ||
		input.CurrentWeight != nil ||
		input.EngagementComposite != nil {
		return "CURRENT"
	}
	return "STALE"
}

// countInputSources counts the number of non-nil/non-zero input fields
// to indicate data coverage.
func countInputSources(input models.PAIDimensionInput) int {
	count := 0

	// Pointer fields
	ptrs := []*float64{
		input.MHRICompositeSlope,
		input.GlucoseDomainSlope,
		input.CardioDomainSlope,
		input.BodyCompDomainSlope,
		input.BehavioralDomainSlope,
		input.CurrentEGFR,
		input.CurrentHbA1c,
		input.CurrentSBP,
		input.CurrentDBP,
		input.CurrentPotassium,
		input.CurrentTBRL2Pct,
		input.CurrentTIR,
		input.CurrentWeight,
		input.PreviousWeight72h,
		input.EngagementComposite,
	}
	for _, p := range ptrs {
		if p != nil {
			count++
		}
	}

	if input.SecondDerivative != nil {
		count++
	}
	if input.DaysSinceDischarge != nil {
		count++
	}

	// Non-zero int/float fields
	if input.DomainsDeterioriating > 0 {
		count++
	}
	if input.DaysSinceLastBPReading > 0 {
		count++
	}
	if input.DaysSinceLastGlucose > 0 {
		count++
	}
	if input.AvgReadingsPerWeek > 0 {
		count++
	}
	if input.CurrentReadingsPerWeek > 0 {
		count++
	}
	if input.MedicationCount > 0 {
		count++
	}
	if input.Age > 0 {
		count++
	}
	if input.DaysSinceLastClinician > 0 {
		count++
	}
	if input.DaysSinceLastCardAck > 0 {
		count++
	}
	if input.UnacknowledgedCardCount > 0 {
		count++
	}

	// Non-empty strings
	if input.CKMStage != "" && input.CKMStage != "0" {
		count++
	}
	if input.EngagementStatus != "" {
		count++
	}
	if input.HFType != "" {
		count++
	}
	if input.NYHAClass != "" {
		count++
	}

	// Booleans (only count if true)
	if input.ConcordantDeterioration {
		count++
	}
	if input.IsPostDischarge30d {
		count++
	}
	if input.IsAcutelyIll {
		count++
	}
	if input.HasRecentHypo {
		count++
	}
	if input.ActiveSteroidCourse {
		count++
	}
	if input.HasUnacknowledgedCards {
		count++
	}

	return count
}

// itoa is a minimal int-to-string helper to avoid importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}
	// reverse
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	if neg {
		return "-" + string(digits)
	}
	return string(digits)
}
