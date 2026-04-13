package services

import (
	"fmt"
	"math"
	"strings"

	"kb-23-decision-cards/internal/models"
)

// MaskedHTNCard represents a decision card for masked/white-coat hypertension phenotypes.
type MaskedHTNCard struct {
	CardType       string                `json:"card_type"`
	Urgency        string                `json:"urgency"`
	Title          string                `json:"title"`
	Rationale      string                `json:"rationale"`
	Actions        []string              `json:"actions"`
	ConfidenceTier models.ConfidenceTier `json:"confidence_tier"`
}

// confidenceStringToTier maps the BP context classifier's string-based
// confidence ("HIGH"/"MODERATE"/"LOW"/"DAMPED") to KB-23's ConfidenceTier
// enum. Used only by masked HTN cards — existing HTN safety templates
// retain bypasses_confidence_gate=true and are unchanged.
//
//	HIGH     -> TierFirm      (full data sufficiency, no selection bias)
//	MODERATE -> TierProbable  (sufficient data but not optimal)
//	LOW      -> TierPossible  (bias risk or minimal data)
//	DAMPED   -> TierUncertain (stability engine dampened a flappy transition)
//	unknown  -> TierUncertain (defensive default)
func confidenceStringToTier(confidence string) models.ConfidenceTier {
	switch confidence {
	case "HIGH":
		return models.TierFirm
	case "MODERATE":
		return models.TierProbable
	case "LOW":
		return models.TierPossible
	case "DAMPED":
		return models.TierUncertain
	default:
		return models.TierUncertain
	}
}

// EvaluateMaskedHTNCards generates decision cards from a BP context classification.
// Card priority order:
//  1. MASKED_HTN_MORNING_SURGE_COMPOUND (IMMEDIATE — compound risk, masked phenotypes)
//  1b. SUSTAINED_HTN_MORNING_SURGE (URGENT — both contexts elevated + morning surge)
//  2. MASKED_HYPERTENSION (IMMEDIATE if DM/CKD amplified, URGENT otherwise)
//  3. MASKED_UNCONTROLLED (URGENT — treated but not controlled at home)
//  4. WHITE_COAT_HYPERTENSION (ROUTINE — avoid overtreatment)
//  4b. WHITE_COAT_UNCONTROLLED (ROUTINE — treated patient, clinic elevated, home controlled)
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
			ConfidenceTier: confidenceStringToTier(c.Confidence),
		})
	}

	// 1b. Sustained HTN + morning surge — still compound risk, but URGENT (not IMMEDIATE)
	// because both contexts already elevated (not hidden).
	if c.MorningSurgeCompound && c.Phenotype == models.PhenotypeSustainedHTN {
		cards = append(cards, MaskedHTNCard{
			CardType: "SUSTAINED_HTN_MORNING_SURGE",
			Urgency:  "URGENT",
			Title:    "Sustained Hypertension with Abnormal Morning Surge",
			Rationale: fmt.Sprintf(
				"Both clinic (%.0f) and home (%.0f) BP elevated with abnormal morning surge (>20 mmHg). "+
					"Morning surge on top of sustained hypertension significantly increases stroke risk "+
					"during the morning cardiovascular event window (06:00–12:00) — Kario 2019 (JACC).",
				c.ClinicSBPMean, c.HomeSBPMean),
			Actions: []string{
				"Consider bedtime dosing of long-acting antihypertensive (chronotherapy)",
				"Evaluate for obstructive sleep apnea — strong association with exaggerated morning surge",
				"Prefer 24-hour-coverage agents: long-acting ARB or dihydropyridine CCB",
				"Consider 24-hour ABPM to characterise surge amplitude",
			},
			ConfidenceTier: confidenceStringToTier(c.Confidence),
		})
	}

	// 2. Masked hypertension phenotype.
	if c.Phenotype == models.PhenotypeMaskedHTN {
		urgency := "URGENT"
		riskMultiplier := ""
		if c.DiabetesAmplification || c.CKDAmplification {
			urgency = "IMMEDIATE"
			if c.DiabetesAmplification {
				riskMultiplier = " Diabetes amplification: 3.2x CV risk multiplier vs masked HTN alone (Leitao 2015, Diabetologia)."
			}
		}

		// Selection bias dampening: if home readings come from a measurement-
		// avoidant or crisis-only-measurement patient, the masked HTN signal
		// may be selection bias rather than true masked HTN. Demote urgency
		// by one level so the card still surfaces but doesn't trigger
		// IMMEDIATE clinical action on questionable data. The SELECTION_BIAS_WARNING
		// card (appended separately) explains why.
		demotedDueToBias := false
		if c.SelectionBiasRisk {
			switch urgency {
			case "IMMEDIATE":
				urgency = "URGENT"
				demotedDueToBias = true
			case "URGENT":
				urgency = "ROUTINE"
				demotedDueToBias = true
			}
		}

		if demotedDueToBias {
			riskMultiplier += " Urgency reduced due to selection bias risk — verify with structured monitoring before acting."
		}

		cards = append(cards, MaskedHTNCard{
			CardType: "MASKED_HYPERTENSION",
			Urgency:  urgency,
			Title:    fmt.Sprintf("Masked Hypertension — Clinic BP Normal, Home Mean %.0f mmHg", c.HomeSBPMean),
			Rationale: fmt.Sprintf(
				"Clinic BP %.0f/%.0f mmHg (normal) but home mean %.0f mmHg (elevated). "+
					"Home BP exceeds clinic by %.0f mmHg. Masked hypertension carries higher CV risk than "+
					"sustained hypertension because treatment is deferred.%s",
				c.ClinicSBPMean, c.ClinicDBPMean, c.HomeSBPMean,
				math.Abs(c.ClinicHomeGapSBP), riskMultiplier),
			Actions: []string{
				"Do not rely on clinic BP alone — initiate or intensify antihypertensive therapy",
				"Target home BP <130/80 mmHg (AHA/ACC 2023)",
				"Assess for end-organ damage: renal function, retinopathy, LVH",
				"Review home monitoring technique and device accuracy",
			},
			ConfidenceTier: confidenceStringToTier(c.Confidence),
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
			CardType:  "MASKED_UNCONTROLLED",
			Urgency:   "URGENT",
			Title:     "Masked Uncontrolled HTN — Therapy Appears Inadequate at Home",
			Rationale: rationale,
			Actions: []string{
				"Review current antihypertensive regimen — dose or agent adjustment likely required",
				"Check medication adherence: home readings pattern vs dosing schedule",
				"Consider ambulatory BP monitoring (ABPM) to quantify 24-hour burden",
			},
			ConfidenceTier: confidenceStringToTier(c.Confidence),
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
			ConfidenceTier: confidenceStringToTier(c.Confidence),
		})
	}

	// 4b. White-coat uncontrolled — treated patient with elevated clinic BP but controlled home BP.
	if c.Phenotype == models.PhenotypeWhiteCoatUncontrolled {
		cards = append(cards, MaskedHTNCard{
			CardType: "WHITE_COAT_UNCONTROLLED",
			Urgency:  "ROUTINE",
			Title:    "White-Coat Effect — Clinic BP Elevated but Home BP Controlled on Therapy",
			Rationale: fmt.Sprintf(
				"Patient on antihypertensive therapy. Clinic BP %.0f/%.0f mmHg elevated; "+
					"home mean %.0f mmHg normal. White-coat effect: %.0f mmHg. "+
					"Home readings confirm treatment is adequate — the clinic elevation reflects "+
					"the clinic environment, not true treatment failure.",
				c.ClinicSBPMean, c.ClinicDBPMean, c.HomeSBPMean, c.WhiteCoatEffect),
			Actions: []string{
				"Do NOT escalate antihypertensive therapy based on clinic reading alone",
				"Consider REDUCING dose if patient reports symptomatic hypotension (dizziness, fatigue)",
				"Continue structured home monitoring — reassess if home readings rise above 135/85 mmHg",
				"Document white-coat effect in chart to prevent future overtreatment",
			},
			ConfidenceTier: confidenceStringToTier(c.Confidence),
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
			ConfidenceTier: confidenceStringToTier(c.Confidence),
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
			ConfidenceTier: confidenceStringToTier(c.Confidence),
		})
	}

	return cards
}
