package services

import (
	"fmt"
	"math"
	"os"
	"time"

	"gopkg.in/yaml.v3"

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

// DefaultPAIConfig returns a PAIConfig populated with standard clinical
// thresholds matching the pai_dimensions specification. This is the
// single source of truth for production defaults; tests may override
// individual fields as needed.
func DefaultPAIConfig() *PAIConfig {
	return &PAIConfig{
		VelocityWeight:   0.30,
		ProximityWeight:  0.25,
		BehavioralWeight: 0.20,
		ContextWeight:    0.15,
		AttentionWeight:  0.10,
		// Velocity thresholds
		SevereDeclineSlope:            -2.0,
		ModerateDeclineSlope:          -1.0,
		MildDeclineSlope:              -0.3,
		StableSlope:                   0.3,
		AcceleratingDeclineMultiplier: 1.5,
		DeceleratingDeclineMultiplier: 0.7,
		ConcordantBonus:               15,
		PerAdditionalDomain:           5,
		ConfounderDampeningEnabled:    false,
		MaxVelocityDuringSeason:       60,
		// Proximity
		ProximityExponent: 2.0,
		// Behavioral
		BehavioralCessationDays:    5,
		BehavioralReducedThreshold: 0.50,
		BehavioralSlightlyReduced:  0.25,
		BehavioralCompoundBoth:     95,
		// Context
		ContextCKMStageBase: map[string]float64{
			"0": 0, "1": 5, "2": 10, "3": 20,
			"4a": 35, "4b": 50, "4c": 65,
		},
		ContextPostDischarge30d:    25,
		ContextAcuteIllness:        20,
		ContextRecentHypo:          15,
		ContextActiveSteroid:       10,
		ContextPolypharmacyElderly: 15,
		ContextPolypharmacyAge:     75,
		ContextPolypharmacyMeds:    5,
		ContextNYHAAmplifier: map[string]float64{
			"I": 1.0, "II": 1.1, "III": 1.3, "IV": 1.5,
		},
		ContextMaxScore: 100,
		// Attention
		AttentionCriticalDays: 90,
		AttentionHighDays:     60,
		AttentionModerateDays: 30,
		AttentionAdequateDays: 14,
		AttentionPerCard:      10,
		AttentionPerDayOldest: 3,
		AttentionCardCap:      50,
		// Tiers
		CriticalThreshold: 80,
		HighThreshold:     60,
		ModerateThreshold: 40,
		LowThreshold:      20,
		SignificantDelta:   10,
	}
}

// ─── YAML config loader ─────────────────────────────────────────────────────

// paiDimensionsYAML mirrors the YAML structure of pai_dimensions.yaml.
type paiDimensionsYAML struct {
	Weights struct {
		Velocity   float64 `yaml:"velocity"`
		Proximity  float64 `yaml:"proximity"`
		Behavioral float64 `yaml:"behavioral"`
		Context    float64 `yaml:"context"`
		Attention  float64 `yaml:"attention"`
	} `yaml:"weights"`
	Velocity struct {
		CompositeSlope struct {
			SevereDecline  float64 `yaml:"severe_decline"`
			ModerateDecline float64 `yaml:"moderate_decline"`
			MildDecline    float64 `yaml:"mild_decline"`
			Stable         float64 `yaml:"stable"`
		} `yaml:"composite_slope"`
		SecondDerivative struct {
			AcceleratingDecline    float64 `yaml:"accelerating_decline"`
			DeceleratingDecline    float64 `yaml:"decelerating_decline"`
			AcceleratingImprovement float64 `yaml:"accelerating_improvement"`
		} `yaml:"second_derivative"`
		ConcordantBonus     float64 `yaml:"concordant_bonus"`
		PerAdditionalDomain float64 `yaml:"per_additional_domain"`
	} `yaml:"velocity"`
	Proximity struct {
		Exponent float64 `yaml:"exponent"`
	} `yaml:"proximity"`
	Behavioral struct {
		MeasurementFrequency struct {
			CessationDays    int     `yaml:"cessation_days"`
			ReducedThreshold float64 `yaml:"reduced_threshold"`
			SlightlyReduced  float64 `yaml:"slightly_reduced"`
		} `yaml:"measurement_frequency"`
		CompoundBoth float64 `yaml:"compound_both"`
	} `yaml:"behavioral"`
	Context struct {
		CKMStageBase map[string]float64 `yaml:"ckm_stage_base"`
		Modifiers    struct {
			PostDischarge30d   float64 `yaml:"post_discharge_30d"`
			AcuteIllness       float64 `yaml:"acute_illness"`
			RecentHypo         float64 `yaml:"recent_hypo"`
			ActiveSteroid      float64 `yaml:"active_steroid"`
			PolypharmacyElderly float64 `yaml:"polypharmacy_elderly"`
		} `yaml:"modifiers"`
		NYHAAmplifier map[string]float64 `yaml:"nyha_amplifier"`
		MaxScore      float64            `yaml:"max_score"`
	} `yaml:"context"`
	Attention struct {
		DaysSinceClinician struct {
			Critical int `yaml:"critical"`
			High     int `yaml:"high"`
			Moderate int `yaml:"moderate"`
			Adequate int `yaml:"adequate"`
		} `yaml:"days_since_clinician"`
		UnacknowledgedCards struct {
			PerCard      float64 `yaml:"per_card"`
			PerDayOldest float64 `yaml:"per_day_oldest"`
			Cap          float64 `yaml:"cap"`
		} `yaml:"unacknowledged_cards"`
	} `yaml:"attention"`
	Tiers struct {
		Critical float64 `yaml:"critical"`
		High     float64 `yaml:"high"`
		Moderate float64 `yaml:"moderate"`
		Low      float64 `yaml:"low"`
	} `yaml:"tiers"`
	SignificantChange struct {
		ScoreDelta float64 `yaml:"score_delta"`
	} `yaml:"significant_change"`
	RateLimit struct {
		MinIntervalMinutes int `yaml:"min_interval_minutes"`
	} `yaml:"rate_limit"`
	ConfounderDampening struct {
		Enabled              bool    `yaml:"enabled"`
		MaxVelocityDuringSeason float64 `yaml:"max_velocity_during_season"`
	} `yaml:"confounder_dampening"`
}

// LoadPAIConfig reads pai_dimensions.yaml and returns a PAIConfig.
func LoadPAIConfig(path string) (*PAIConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read PAI config: %w", err)
	}

	var raw paiDimensionsYAML
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse PAI config: %w", err)
	}

	return &PAIConfig{
		VelocityWeight:   raw.Weights.Velocity,
		ProximityWeight:  raw.Weights.Proximity,
		BehavioralWeight: raw.Weights.Behavioral,
		ContextWeight:    raw.Weights.Context,
		AttentionWeight:  raw.Weights.Attention,

		SevereDeclineSlope:            raw.Velocity.CompositeSlope.SevereDecline,
		ModerateDeclineSlope:          raw.Velocity.CompositeSlope.ModerateDecline,
		MildDeclineSlope:              raw.Velocity.CompositeSlope.MildDecline,
		StableSlope:                   raw.Velocity.CompositeSlope.Stable,
		AcceleratingDeclineMultiplier: raw.Velocity.SecondDerivative.AcceleratingDecline,
		DeceleratingDeclineMultiplier: raw.Velocity.SecondDerivative.DeceleratingDecline,
		ConcordantBonus:               raw.Velocity.ConcordantBonus,
		PerAdditionalDomain:           raw.Velocity.PerAdditionalDomain,
		ConfounderDampeningEnabled:    raw.ConfounderDampening.Enabled,
		MaxVelocityDuringSeason:       raw.ConfounderDampening.MaxVelocityDuringSeason,

		ProximityExponent: raw.Proximity.Exponent,

		BehavioralCessationDays:    raw.Behavioral.MeasurementFrequency.CessationDays,
		BehavioralReducedThreshold: raw.Behavioral.MeasurementFrequency.ReducedThreshold,
		BehavioralSlightlyReduced:  raw.Behavioral.MeasurementFrequency.SlightlyReduced,
		BehavioralCompoundBoth:     raw.Behavioral.CompoundBoth,

		ContextCKMStageBase:        raw.Context.CKMStageBase,
		ContextPostDischarge30d:    raw.Context.Modifiers.PostDischarge30d,
		ContextAcuteIllness:        raw.Context.Modifiers.AcuteIllness,
		ContextRecentHypo:          raw.Context.Modifiers.RecentHypo,
		ContextActiveSteroid:       raw.Context.Modifiers.ActiveSteroid,
		ContextPolypharmacyElderly: raw.Context.Modifiers.PolypharmacyElderly,
		ContextPolypharmacyAge:     75,
		ContextPolypharmacyMeds:    5,
		ContextNYHAAmplifier:       raw.Context.NYHAAmplifier,
		ContextMaxScore:            raw.Context.MaxScore,

		AttentionCriticalDays: raw.Attention.DaysSinceClinician.Critical,
		AttentionHighDays:     raw.Attention.DaysSinceClinician.High,
		AttentionModerateDays: raw.Attention.DaysSinceClinician.Moderate,
		AttentionAdequateDays: raw.Attention.DaysSinceClinician.Adequate,
		AttentionPerCard:      raw.Attention.UnacknowledgedCards.PerCard,
		AttentionPerDayOldest: raw.Attention.UnacknowledgedCards.PerDayOldest,
		AttentionCardCap:      raw.Attention.UnacknowledgedCards.Cap,

		CriticalThreshold: raw.Tiers.Critical,
		HighThreshold:     raw.Tiers.High,
		ModerateThreshold: raw.Tiers.Moderate,
		LowThreshold:      raw.Tiers.Low,
		SignificantDelta:   raw.SignificantChange.ScoreDelta,
	}, nil
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
