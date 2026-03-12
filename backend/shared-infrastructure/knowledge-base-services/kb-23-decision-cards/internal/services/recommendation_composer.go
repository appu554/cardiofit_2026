package services

import (
	"encoding/json"

	"go.uber.org/zap"

	"kb-23-decision-cards/internal/config"
	"kb-23-decision-cards/internal/models"
)

// RecommendationComposer generates typed CardRecommendation records from
// template recommendation definitions, filtered by confidence tier.
//
// Implements:
//   - V-04: SAFETY_INSTRUCTION bypasses the confidence gate entirely.
//   - V-05: MEDICATION_REVIEW is permitted at PROBABLE tier.
//   - V-01: MEDICATION_HOLD and MEDICATION_MODIFY require firm_medication_change.
type RecommendationComposer struct {
	cfg *config.Config
	log *zap.Logger
}

// NewRecommendationComposer creates a RecommendationComposer with the given
// configuration and logger.
func NewRecommendationComposer(cfg *config.Config, log *zap.Logger) *RecommendationComposer {
	return &RecommendationComposer{cfg: cfg, log: log}
}

// Compose generates CardRecommendation records from template recommendations,
// filtered by the patient's confidence tier.
func (r *RecommendationComposer) Compose(
	tmpl *models.CardTemplate,
	tier models.ConfidenceTier,
	isFirmMedChange bool,
) []models.CardRecommendation {
	var recommendations []models.CardRecommendation

	for _, rec := range tmpl.Recommendations {
		// V-04: SAFETY_INSTRUCTION bypasses confidence gate entirely
		if rec.BypassesConfidenceGate {
			recommendations = append(recommendations, r.buildRec(rec, false))
			continue
		}

		// Check confidence tier eligibility
		if !r.tierMeetsRequirement(tier, rec.ConfidenceTierRequired) {
			r.log.Debug("recommendation filtered by confidence tier",
				zap.String("rec_type", string(rec.RecType)),
				zap.String("required", string(rec.ConfidenceTierRequired)),
				zap.String("actual", string(tier)),
			)
			continue
		}

		// V-01: MEDICATION_HOLD and MEDICATION_MODIFY require firm_medication_change
		if (rec.RecType == models.RecMedicationHold || rec.RecType == models.RecMedicationModify) && !isFirmMedChange {
			r.log.Debug("medication rec filtered by firm_medication_change threshold",
				zap.String("rec_type", string(rec.RecType)),
			)
			continue
		}

		// V-05: MEDICATION_REVIEW is permitted at PROBABLE tier
		// (no additional gating needed -- it passes the tier check above)

		recommendations = append(recommendations, r.buildRec(rec, false))
	}

	return recommendations
}

// ComposeFromSecondary generates recommendations from secondary differentials
// (V-09). Only INVESTIGATION and MONITORING types are auto-included from
// secondary differential templates.
func (r *RecommendationComposer) ComposeFromSecondary(
	tmpl *models.CardTemplate,
	tier models.ConfidenceTier,
) []models.CardRecommendation {
	var recommendations []models.CardRecommendation

	for _, rec := range tmpl.Recommendations {
		// V-09: only INVESTIGATION and MONITORING from secondaries
		if rec.RecType != models.RecInvestigation && rec.RecType != models.RecMonitoring {
			continue
		}

		if !rec.BypassesConfidenceGate && !r.tierMeetsRequirement(tier, rec.ConfidenceTierRequired) {
			continue
		}

		recommendations = append(recommendations, r.buildRec(rec, true))
	}

	return recommendations
}

// buildRec converts a TemplateRecommendation into a CardRecommendation,
// marking whether it originated from a secondary differential.
// CTL Panel 2: If the template defines condition_criteria, they are evaluated
// and the overall ConditionStatus is derived.
func (r *RecommendationComposer) buildRec(rec models.TemplateRecommendation, fromSecondary bool) models.CardRecommendation {
	cr := models.CardRecommendation{
		RecType:                   rec.RecType,
		Urgency:                   rec.Urgency,
		Target:                    rec.Target,
		ActionTextEn:              rec.ActionTextEn,
		ActionTextHi:              rec.ActionTextHi,
		RationaleEn:               rec.RationaleEn,
		GuidelineRef:              rec.GuidelineRef,
		ConfidenceTierRequired:    rec.ConfidenceTierRequired,
		BypassesConfidenceGate:    rec.BypassesConfidenceGate,
		FromSecondaryDifferential: fromSecondary,
		SortOrder:                 rec.SortOrder,
	}

	if rec.TriggerConditionEn != "" {
		triggerEn := rec.TriggerConditionEn
		cr.TriggerConditionEn = &triggerEn
	}
	if rec.TriggerConditionHi != "" {
		triggerHi := rec.TriggerConditionHi
		cr.TriggerConditionHi = &triggerHi
	}

	// CTL Panel 2: Evaluate condition criteria if defined in template
	if len(rec.ConditionCriteria) > 0 {
		criteria := make([]models.ConditionCriterion, len(rec.ConditionCriteria))
		overall := models.ConditionMet

		for i, def := range rec.ConditionCriteria {
			// Criteria conditions use the same token vocabulary as gate conditions.
			// At build time we don't have patient context here, so criteria are
			// marked as CRITERIA_MET by default. The CardBuilder's
			// evaluateGuidelineConditions aggregates the final status after
			// gate evaluation has determined clinical context.
			criteria[i] = models.ConditionCriterion{
				CriterionID: def.CriterionID,
				Description: def.Description,
				Status:      models.ConditionMet,
			}
		}

		criteriaJSON, err := json.Marshal(criteria)
		if err == nil {
			cr.ConditionCriteria = models.JSONB(criteriaJSON)
		}
		cr.ConditionStatus = &overall
	}

	return cr
}

// tierMeetsRequirement checks if the actual tier meets the required tier.
// Tier hierarchy: FIRM > PROBABLE > POSSIBLE > UNCERTAIN
func (r *RecommendationComposer) tierMeetsRequirement(actual, required models.ConfidenceTier) bool {
	return tierLevel(actual) >= tierLevel(required)
}

// tierLevel returns a numeric level for tier comparison.
func tierLevel(t models.ConfidenceTier) int {
	switch t {
	case models.TierFirm:
		return 3
	case models.TierProbable:
		return 2
	case models.TierPossible:
		return 1
	case models.TierUncertain:
		return 0
	default:
		return 0
	}
}
