package services

import (
	"fmt"
	"strings"

	"kb-23-decision-cards/internal/models"
)

// MaskedHTNCard represents a decision card for masked/white-coat hypertension phenotypes.
type MaskedHTNCard struct {
	CardType  string   `json:"card_type"`
	Urgency   string   `json:"urgency"`
	Title     string   `json:"title"`
	Rationale string   `json:"rationale"`
	Actions   []string `json:"actions"`
}

// EvaluateMaskedHTNCards generates decision cards from a BP context classification.
// Card priority order:
//  1. MASKED_HTN_MORNING_SURGE_COMPOUND (IMMEDIATE — compound risk)
//  2. MASKED_HYPERTENSION (IMMEDIATE if DM/CKD amplified, URGENT otherwise)
//  3. MASKED_UNCONTROLLED (URGENT — treated but not controlled at home)
//  4. WHITE_COAT_HYPERTENSION (ROUTINE — avoid overtreatment)
//  5. SELECTION_BIAS_WARNING (ROUTINE — reading quality caveat)
//  6. MEDICATION_TIMING (ROUTINE — chronotherapy suggestion)
func EvaluateMaskedHTNCards(c *models.BPContextClassification) []MaskedHTNCard {
	if c == nil {
		return nil
	}

	var cards []MaskedHTNCard

	// 1. Compound risk: masked HTN + morning surge — highest urgency.
	if c.MorningSurgeCompound &&
		(c.Phenotype == models.PhenotypeMaskedHTN || c.Phenotype == models.PhenotypeMaskedUncontrolled) {
		cards = append(cards, MaskedHTNCard{
			CardType: "MASKED_HTN_MORNING_SURGE_COMPOUND",
			Urgency:  "IMMEDIATE",
			Title:    "Masked HTN + Morning Surge — Compound CV Risk",
			Rationale: fmt.Sprintf(
				"Clinic BP %.0f/%.0f mmHg; home mean %.0f mmHg. Morning surge compounds masked "+
					"hypertension — combined risk exceeds either condition alone. Peak cardiovascular "+
					"event risk window (06:00–12:00) coincides with uncontrolled BP period.",
				c.ClinicSBPMean, c.ClinicDBPMean, c.HomeSBPMean),
			Actions: []string{
				"24-hour ABPM to characterise morning surge amplitude",
				"Review medication timing — consider evening/bedtime dosing of long-acting agent",
				"Urgent cardiology review if home SBP >160 mmHg in morning window",
			},
		})
	}

	// 2. Masked hypertension phenotype.
	if c.Phenotype == models.PhenotypeMaskedHTN {
		urgency := "URGENT"
		riskMultiplier := ""
		if c.DiabetesAmplification || c.CKDAmplification {
			urgency = "IMMEDIATE"
			if c.DiabetesAmplification {
				riskMultiplier = " Diabetes amplification: 3.2x CV risk multiplier vs masked HTN alone (JACC 2021)."
			}
		}

		cards = append(cards, MaskedHTNCard{
			CardType: "MASKED_HYPERTENSION",
			Urgency:  urgency,
			Title:    fmt.Sprintf("Masked Hypertension — Clinic BP Normal, Home Mean %.0f mmHg", c.HomeSBPMean),
			Rationale: fmt.Sprintf(
				"Clinic BP %.0f/%.0f mmHg (normal) but home mean %.0f mmHg (elevated). "+
					"Clinic-home gap: %.0f mmHg. Masked hypertension carries higher CV risk than "+
					"sustained hypertension because treatment is deferred.%s",
				c.ClinicSBPMean, c.ClinicDBPMean, c.HomeSBPMean,
				c.ClinicHomeGapSBP, riskMultiplier),
			Actions: []string{
				"Do not rely on clinic BP alone — initiate or intensify antihypertensive therapy",
				"Target home BP <130/80 mmHg (AHA/ACC 2023)",
				"Assess for end-organ damage: renal function, retinopathy, LVH",
				"Review home monitoring technique and device accuracy",
			},
		})
	}

	// 3. Masked uncontrolled hypertension (treated but not controlled at home).
	if c.Phenotype == models.PhenotypeMaskedUncontrolled {
		rationale := fmt.Sprintf(
			"Patient is controlled in clinic but not at home. "+
				"Clinic BP %.0f/%.0f mmHg; home mean %.0f mmHg.",
			c.ClinicSBPMean, c.ClinicDBPMean, c.HomeSBPMean)
		if c.CKDAmplification {
			rationale += " CKD co-presence accelerates renal progression with uncontrolled BP."
		}

		cards = append(cards, MaskedHTNCard{
			CardType: "MASKED_UNCONTROLLED",
			Urgency:  "URGENT",
			Title:    "Masked Uncontrolled HTN — Therapy Appears Inadequate at Home",
			Rationale: rationale,
			Actions: []string{
				"Review current antihypertensive regimen — dose or agent adjustment likely required",
				"Check medication adherence: home readings pattern vs dosing schedule",
				"Consider ambulatory BP monitoring (ABPM) to quantify 24-hour burden",
			},
		})
	}

	// 4. White-coat hypertension — avoid overtreatment.
	if c.Phenotype == models.PhenotypeWhiteCoatHTN {
		cards = append(cards, MaskedHTNCard{
			CardType: "WHITE_COAT_HYPERTENSION",
			Urgency:  "ROUTINE",
			Title:    fmt.Sprintf("White-Coat HTN — Clinic BP Elevated, Home Mean %.0f mmHg", c.HomeSBPMean),
			Rationale: fmt.Sprintf(
				"Clinic BP %.0f/%.0f mmHg elevated; home mean %.0f mmHg normal. "+
					"White-coat effect: %.0f mmHg. Risk of overtreatment: initiating or escalating "+
					"antihypertensives based on clinic-only readings may cause iatrogenic hypotension. "+
					"White-coat HTN does not carry the same CV risk as true sustained HTN.",
				c.ClinicSBPMean, c.ClinicDBPMean, c.HomeSBPMean, c.WhiteCoatEffect),
			Actions: []string{
				"Do NOT intensify antihypertensive therapy based on this clinic reading alone",
				"Continue home monitoring — reassess if home readings consistently rise above 135/85 mmHg",
				"Lifestyle counselling (sodium, exercise, stress) remains appropriate",
				"Consider ABPM for formal confirmation if clinical decision is complex",
			},
		})
	}

	// 5. Selection bias warning for low-confidence / avoidant patients.
	if c.SelectionBiasRisk {
		engagementDetail := ""
		if c.EngagementPhenotype != "" {
			engagementDetail = fmt.Sprintf(" Patient engagement phenotype: %s.",
				strings.ToLower(strings.ReplaceAll(c.EngagementPhenotype, "_", " ")))
		}
		readingDetail := ""
		if c.HomeReadingCount > 0 {
			readingDetail = fmt.Sprintf(" Only %d home readings available.", c.HomeReadingCount)
		}

		cards = append(cards, MaskedHTNCard{
			CardType: "SELECTION_BIAS_WARNING",
			Urgency:  "ROUTINE",
			Title:    "Home BP Reading Quality — Selection Bias Risk",
			Rationale: fmt.Sprintf(
				"Home BP classification has LOW confidence due to potential selection bias.%s%s "+
					"Patients who measure only when symptomatic produce systematically higher readings; "+
					"measurement avoidant patients produce systematically lower readings — both distort "+
					"the clinic-home discordance analysis.",
				engagementDetail, readingDetail),
			Actions: []string{
				"Discuss structured home monitoring protocol: morning and evening readings × 7 days",
				"Validate home BP device against clinic sphygmomanometer",
				"Do not change therapy based on this classification alone — repeat structured HBPM first",
			},
		})
	}

	// 6. Medication timing hypothesis.
	if c.MedicationTimingHypothesis != "" && c.OnAntihypertensives {
		cards = append(cards, MaskedHTNCard{
			CardType: "MEDICATION_TIMING",
			Urgency:  "ROUTINE",
			Title:    "Medication Timing Optimisation Opportunity",
			Rationale: fmt.Sprintf(
				"BP pattern suggests suboptimal medication timing. %s "+
					"Chronotherapy — matching dosing time to BP circadian pattern — can reduce "+
					"24-hour BP burden without dose escalation.",
				c.MedicationTimingHypothesis),
			Actions: []string{
				"Consider switching once-daily agent to evening dosing",
				"Reassess home BP after 4 weeks of timing change before escalating dose",
				"ABPM post-change to confirm circadian pattern normalisation",
			},
		})
	}

	return cards
}
