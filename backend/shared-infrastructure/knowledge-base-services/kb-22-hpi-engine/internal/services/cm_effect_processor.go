package services

import (
	"math"

	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/models"
)

// TreatmentContraindication represents a HARD_BLOCK signal from a context modifier.
// Consumed by KB-23 Decision Cards to suppress contraindicated treatment options.
type TreatmentContraindication struct {
	ModifierID       string `json:"modifier_id"`
	BlockedTreatment string `json:"blocked_treatment"` // e.g. "NITRATE_THERAPY", "PDE5I_WITH_NITRATE"
	Reason           string `json:"reason"`             // CM description / clinical rationale
	DrugClass        string `json:"drug_class,omitempty"`
}

// PosteriorOverride represents an OVERRIDE signal from a context modifier.
// Applied after Bayesian posterior computation to enforce minimum posteriors.
type PosteriorOverride struct {
	ModifierID     string  `json:"modifier_id"`
	DifferentialID string  `json:"differential_id"`
	MinPosterior   float64 `json:"min_posterior"` // forced minimum posterior probability
}

// CMEffectResult holds the extracted G5 effects from a set of context modifiers.
type CMEffectResult struct {
	Contraindications []TreatmentContraindication `json:"contraindications"`
	Overrides         []PosteriorOverride          `json:"overrides"`
}

// CMEffectProcessor extracts and applies G5 effect types (HARD_BLOCK, OVERRIDE)
// from context modifiers that were recorded as passthrough by CMApplicator.
//
// Architecture:
//   - CMApplicator.Apply() records HARD_BLOCK/OVERRIDE with delta=0.0 (no log-odds shift)
//   - CMEffectProcessor.Extract() scans the modifier list for these effect types
//   - CMEffectProcessor.ApplyOverrides() enforces posterior minimums after GetPosteriors()
//   - TreatmentContraindications are passed to KB-23 via HPI_COMPLETE event
type CMEffectProcessor struct {
	log *zap.Logger
}

// NewCMEffectProcessor creates a new G5 effect processor.
func NewCMEffectProcessor(log *zap.Logger) *CMEffectProcessor {
	return &CMEffectProcessor{log: log}
}

// Extract scans a modifier list and extracts all HARD_BLOCK and OVERRIDE effects.
// Returns a CMEffectResult containing contraindications and posterior overrides.
func (p *CMEffectProcessor) Extract(modifiers []ContextModifier) CMEffectResult {
	var result CMEffectResult

	for _, mod := range modifiers {
		switch mod.Effect {
		case models.CMEffectHardBlock:
			if mod.BlockedTreatment == "" {
				p.log.Warn("G5: HARD_BLOCK modifier missing blocked_treatment",
					zap.String("modifier_id", mod.ModifierID),
				)
				continue
			}
			result.Contraindications = append(result.Contraindications, TreatmentContraindication{
				ModifierID:       mod.ModifierID,
				BlockedTreatment: mod.BlockedTreatment,
				DrugClass:        mod.DrugClass,
			})
			p.log.Info("G5: HARD_BLOCK extracted",
				zap.String("modifier_id", mod.ModifierID),
				zap.String("blocked_treatment", mod.BlockedTreatment),
			)

		case models.CMEffectOverride:
			if len(mod.OverrideTargets) == 0 {
				p.log.Warn("G5: OVERRIDE modifier missing override_targets",
					zap.String("modifier_id", mod.ModifierID),
				)
				continue
			}
			for diffID, minPost := range mod.OverrideTargets {
				if minPost <= 0 || minPost >= 1.0 {
					p.log.Warn("G5: OVERRIDE min_posterior out of range (0, 1.0)",
						zap.String("modifier_id", mod.ModifierID),
						zap.String("differential_id", diffID),
						zap.Float64("min_posterior", minPost),
					)
					continue
				}
				result.Overrides = append(result.Overrides, PosteriorOverride{
					ModifierID:     mod.ModifierID,
					DifferentialID: diffID,
					MinPosterior:   minPost,
				})
			}
			p.log.Info("G5: OVERRIDE extracted",
				zap.String("modifier_id", mod.ModifierID),
				zap.Int("override_count", len(mod.OverrideTargets)),
			)
		}
	}

	return result
}

// ApplyOverrides enforces posterior minimums from OVERRIDE modifiers on a
// computed posteriors slice. If a differential's posterior is below the override
// minimum, it is raised to that minimum and the remaining posteriors are
// renormalized proportionally downward.
//
// This is a post-GetPosteriors() step, not a log-odds operation.
// Returns true if any override was applied.
func (p *CMEffectProcessor) ApplyOverrides(
	posteriors []models.DifferentialEntry,
	overrides []PosteriorOverride,
) bool {
	if len(overrides) == 0 || len(posteriors) == 0 {
		return false
	}

	// Build override map: differential_id -> max(min_posterior across all overrides)
	overrideMap := make(map[string]float64, len(overrides))
	for _, ovr := range overrides {
		if existing, ok := overrideMap[ovr.DifferentialID]; !ok || ovr.MinPosterior > existing {
			overrideMap[ovr.DifferentialID] = ovr.MinPosterior
		}
	}

	// Check if any override needs to fire
	anyApplied := false
	deficit := 0.0

	for i := range posteriors {
		minPost, isOverride := overrideMap[posteriors[i].DifferentialID]
		if isOverride && posteriors[i].PosteriorProbability < minPost {
			deficit += minPost - posteriors[i].PosteriorProbability
			posteriors[i].PosteriorProbability = minPost
			anyApplied = true

			p.log.Info("G5: OVERRIDE applied — posterior raised",
				zap.String("differential_id", posteriors[i].DifferentialID),
				zap.Float64("min_posterior", minPost),
			)
		}
	}

	if !anyApplied {
		return false
	}

	// Renormalize: reduce non-overridden posteriors proportionally to absorb deficit.
	// Sum of non-overridden posteriors before adjustment.
	nonOverriddenSum := 0.0
	for i := range posteriors {
		if _, isOverride := overrideMap[posteriors[i].DifferentialID]; !isOverride {
			nonOverriddenSum += posteriors[i].PosteriorProbability
		}
	}

	if nonOverriddenSum > 0 && deficit > 0 {
		scaleFactor := (nonOverriddenSum - deficit) / nonOverriddenSum
		// Clamp scale factor to prevent negatives (pathological case where
		// overrides consume more than available non-override mass)
		if scaleFactor < 0 {
			scaleFactor = 0
		}

		for i := range posteriors {
			if _, isOverride := overrideMap[posteriors[i].DifferentialID]; !isOverride {
				posteriors[i].PosteriorProbability = math.Max(0, posteriors[i].PosteriorProbability*scaleFactor)
			}
		}
	}

	return true
}
