package services

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"kb-26-metabolic-digital-twin/internal/models"
)

// ComputePredictedRisk evaluates 6 heuristic scoring signals and returns
// a composite 30-day deterioration risk for the given patient.
func ComputePredictedRisk(input models.PredictedRiskInput) models.PredictedRisk {
	now := time.Now().UTC()

	var factors []models.RiskFactor

	// ── Signal 1: Trajectory declining (max 25) ─────────────────────────
	if input.CompositeSlope30d != nil && *input.CompositeSlope30d < -0.5 {
		absSlope := math.Abs(*input.CompositeSlope30d)
		contribution := math.Min(absSlope/2.0*25.0, 25.0)
		factors = append(factors, models.RiskFactor{
			FactorName:     "declining_trajectory",
			FactorValue:    *input.CompositeSlope30d,
			Contribution:   contribution,
			Direction:      "DECLINING",
			Modifiable:     false,
			Interpretation: "Clinical trajectory has been declining",
		})
	}

	// ── Signal 2: PAI trend rising (max 20) ─────────────────────────────
	if input.PAITrend30d != nil && *input.PAITrend30d > 0.3 {
		contribution := math.Min(*input.PAITrend30d/1.0*20.0, 20.0)
		factors = append(factors, models.RiskFactor{
			FactorName:     "rising_pai_trend",
			FactorValue:    *input.PAITrend30d,
			Contribution:   contribution,
			Direction:      "RISING",
			Modifiable:     false,
			Interpretation: "Patient acuity index trending upward",
		})
	}

	// ── Signal 3: Engagement declining (max 20, modifiable) ─────────────
	engagementTriggered := false
	if input.EngagementTrend30d != nil && *input.EngagementTrend30d < -0.20 {
		engagementTriggered = true
	}
	if input.MeasurementFreqDrop > 0.20 {
		engagementTriggered = true
	}
	if engagementTriggered {
		var drop float64
		if input.EngagementTrend30d != nil {
			drop = math.Abs(*input.EngagementTrend30d)
		}
		if input.MeasurementFreqDrop > drop {
			drop = input.MeasurementFreqDrop
		}
		contribution := math.Min(drop/0.50*20.0, 20.0)
		factors = append(factors, models.RiskFactor{
			FactorName:        "declining_engagement",
			FactorValue:       drop,
			Contribution:      contribution,
			Direction:         "DECLINING",
			Modifiable:        true,
			Interpretation:    "Patient engagement has been declining",
			RecommendedAction: "Schedule engagement re-establishment outreach within 7 days",
		})
	}

	// ── Signal 4: Post-discharge window (max 15) ────────────────────────
	if input.IsPostDischarge && input.DaysSinceDischarge <= 30 {
		contribution := 15.0 * (1.0 - float64(input.DaysSinceDischarge)/30.0)
		factors = append(factors, models.RiskFactor{
			FactorName:     "post_discharge_window",
			FactorValue:    float64(input.DaysSinceDischarge),
			Contribution:   contribution,
			Direction:      "ELEVATED",
			Modifiable:     false,
			Interpretation: fmt.Sprintf("Within 30-day post-discharge window (day %d)", input.DaysSinceDischarge),
		})
	}

	// ── Signal 5: Medication complexity (max 10, modifiable) ────────────
	if input.MedicationChanges30d > 2 || input.PolypharmacyCount > 8 {
		contribution := math.Min(float64(input.MedicationChanges30d+input.PolypharmacyCount/4)*2.0, 10.0)
		factors = append(factors, models.RiskFactor{
			FactorName:        "medication_complexity",
			FactorValue:       float64(input.MedicationChanges30d),
			Contribution:      contribution,
			Direction:         "ELEVATED",
			Modifiable:        true,
			Interpretation:    "High medication complexity or recent changes",
			RecommendedAction: "Schedule medication reconciliation review",
		})
	}

	// ── Signal 6: Confounder burden (max 10) ────────────────────────────
	if input.ActiveConfounderScore > 0.3 {
		contribution := math.Min(input.ActiveConfounderScore*20.0, 10.0)
		factors = append(factors, models.RiskFactor{
			FactorName:     "confounder_burden",
			FactorValue:    input.ActiveConfounderScore,
			Contribution:   contribution,
			Direction:      "ELEVATED",
			Modifiable:     false,
			Interpretation: "Active confounders may obscure clinical signals",
		})
	}

	// ── Composite score ─────────────────────────────────────────────────
	var total float64
	for _, f := range factors {
		total += f.Contribution
	}
	if total > 100.0 {
		total = 100.0
	}

	// ── Tier ────────────────────────────────────────────────────────────
	var tier models.RiskTier
	switch {
	case total >= 50:
		tier = models.RiskTierHigh
	case total >= 25:
		tier = models.RiskTierModerate
	default:
		tier = models.RiskTierLow
	}

	// ── Primary drivers (sorted by contribution desc) ───────────────────
	primaryDrivers := make([]models.RiskFactor, len(factors))
	copy(primaryDrivers, factors)
	sort.Slice(primaryDrivers, func(i, j int) bool {
		return primaryDrivers[i].Contribution > primaryDrivers[j].Contribution
	})

	// ── Modifiable drivers ──────────────────────────────────────────────
	var modifiableDrivers []models.RiskFactor
	for _, f := range primaryDrivers {
		if f.Modifiable {
			modifiableDrivers = append(modifiableDrivers, f)
		}
	}

	// ── Counterfactual reduction ────────────────────────────────────────
	var counterfactualReduction float64
	for _, f := range modifiableDrivers {
		switch f.FactorName {
		case "declining_engagement":
			counterfactualReduction += 8.0
		case "medication_complexity":
			counterfactualReduction += 5.0
		}
	}

	// ── Summary / recommended action ────────────────────────────────────
	tierLabel := strings.ToLower(string(tier))
	var driverNames []string
	for _, f := range primaryDrivers {
		driverNames = append(driverNames, strings.ReplaceAll(f.FactorName, "_", " "))
	}
	summary := fmt.Sprintf("%s risk of clinical deterioration in 30 days.",
		strings.Title(tierLabel))
	if len(driverNames) > 0 {
		summary += " Primary drivers: " + strings.Join(driverNames, ", ") + "."
	}

	recommendedAction := "Continue routine monitoring"
	if len(modifiableDrivers) > 0 {
		recommendedAction = modifiableDrivers[0].RecommendedAction
	}

	return models.PredictedRisk{
		PatientID:               input.PatientID,
		PredictionType:          "DETERIORATION_30D",
		RiskScore:               total,
		RiskTier:                string(tier),
		PrimaryDrivers:          primaryDrivers,
		ModifiableDrivers:       modifiableDrivers,
		RiskSummary:             summary,
		RecommendedAction:       recommendedAction,
		CounterfactualReduction: counterfactualReduction,
		PredictionWindowDays:    30,
		ModelType:               "HEURISTIC",
		ComputedAt:              now,
		ExpiresAt:               now.Add(24 * time.Hour),
	}
}
