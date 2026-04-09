package services

// CGMCardInput collects CGM metrics and clinical context needed
// to generate glycaemic decision cards.
type CGMCardInput struct {
	TIRPct         float64
	TBRL1Pct       float64
	TBRL2Pct       float64
	TARL1Pct       float64
	TARL2Pct       float64
	CVPct          float64
	GRIZone        string
	SufficientData bool
	OnSUOrInsulin  bool
	NocturnalHypo  bool
	GMIDiscrepancy bool
}

// CGMCard represents a single clinical decision card triggered by CGM data.
type CGMCard struct {
	CardType  string `json:"card_type"`
	Urgency   string `json:"urgency"`
	Title     string `json:"title"`
	Rationale string `json:"rationale"`
}

// Card type constants.
const (
	CardCGMDataQuality        = "CGM_DATA_QUALITY"
	CardHypoglycaemiaRisk     = "HYPOGLYCAEMIA_RISK"
	CardSustainedHyperglycaemia = "SUSTAINED_HYPERGLYCAEMIA"
	CardLowTimeInRange        = "LOW_TIME_IN_RANGE"
	CardGlucoseVariability    = "GLUCOSE_VARIABILITY"
	CardNocturnalHypoglycaemia = "NOCTURNAL_HYPOGLYCAEMIA"
	CardGMIHbA1cDiscrepancy   = "GMI_HBAIC_DISCREPANCY"
)

// Urgency levels reuse constants from urgency_calculator.go:
// UrgencyImmediate, UrgencyUrgent, UrgencyRoutine.

// GenerateCGMCards evaluates CGM metrics and returns applicable clinical
// decision cards ordered by clinical priority.
//
// Data quality gate: if SufficientData is false (<70% coverage), only a
// CGM_DATA_QUALITY card is returned — clinical cards are suppressed to
// prevent misleading conclusions from sparse data.
func GenerateCGMCards(input CGMCardInput) []CGMCard {
	if !input.SufficientData {
		return []CGMCard{{
			CardType:  CardCGMDataQuality,
			Urgency:   UrgencyRoutine,
			Title:     "Insufficient CGM data coverage",
			Rationale: "CGM data coverage is below 70%; clinical metrics may be unreliable. Encourage sensor wear compliance.",
		}}
	}

	var cards []CGMCard

	// Hypoglycaemia risk — severe (L2 >1%) or moderate (L1 >4%)
	if input.TBRL2Pct > 1.0 || input.TBRL1Pct > 4.0 {
		urgency := UrgencyUrgent
		if input.OnSUOrInsulin {
			urgency = UrgencyImmediate
		}
		cards = append(cards, CGMCard{
			CardType:  CardHypoglycaemiaRisk,
			Urgency:   urgency,
			Title:     "Hypoglycaemia risk detected",
			Rationale: "Time below range exceeds safety thresholds; review hypoglycaemia-prone medications and meal timing.",
		})
	}

	// Sustained hyperglycaemia — TAR L2 >5%
	if input.TARL2Pct > 5.0 {
		cards = append(cards, CGMCard{
			CardType:  CardSustainedHyperglycaemia,
			Urgency:   UrgencyUrgent,
			Title:     "Sustained hyperglycaemia detected",
			Rationale: "Time above range (>250 mg/dL) exceeds 5%; evaluate treatment intensification.",
		})
	}

	// Low time in range — TIR <50% without significant hypo
	if input.TIRPct < 50.0 && input.TBRL2Pct <= 1.0 {
		cards = append(cards, CGMCard{
			CardType:  CardLowTimeInRange,
			Urgency:   UrgencyUrgent,
			Title:     "Low time in range",
			Rationale: "TIR below 50% indicates suboptimal glucose management; review medication and lifestyle interventions.",
		})
	}

	// Glucose variability — CV >36%
	if input.CVPct > 36.0 {
		cards = append(cards, CGMCard{
			CardType:  CardGlucoseVariability,
			Urgency:   UrgencyRoutine,
			Title:     "Elevated glucose variability",
			Rationale: "Coefficient of variation exceeds 36%; consider carbohydrate consistency and medication timing review.",
		})
	}

	// Nocturnal hypoglycaemia
	if input.NocturnalHypo {
		cards = append(cards, CGMCard{
			CardType:  CardNocturnalHypoglycaemia,
			Urgency:   UrgencyUrgent,
			Title:     "Nocturnal hypoglycaemia detected",
			Rationale: "Overnight low glucose events detected; review evening medications and bedtime snack adequacy.",
		})
	}

	// GMI-HbA1c discrepancy
	if input.GMIDiscrepancy {
		cards = append(cards, CGMCard{
			CardType:  CardGMIHbA1cDiscrepancy,
			Urgency:   UrgencyRoutine,
			Title:     "GMI–HbA1c discrepancy",
			Rationale: "Significant divergence between CGM-derived GMI and lab HbA1c; consider glycation rate variant or haemoglobinopathy screen.",
		})
	}

	return cards
}
