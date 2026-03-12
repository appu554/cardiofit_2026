package services

import (
	"math"

	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/models"
)

// maxCMLogOddsShift caps the total CM log-odds shift per differential.
// Prevents extreme posteriors when ≥3 CMs fire on the same differential.
// ±2.0 maps to approximately 0.12–0.88 probability range.
const maxCMLogOddsShift = 2.0

// cmStackedThreshold is the number of CMs targeting a single differential
// that triggers a CM_STACKED warning. Polypharmacy patients commonly exceed
// this, which is valuable clinical metadata even when the cap prevents
// extreme posteriors.
const cmStackedThreshold = 3

// CMApplicator applies context modifiers (F-01 + F-03) to the log-odds state
// vector. Context modifiers represent external clinical factors such as
// concomitant drugs, comorbidities, or lifestyle factors that shift the prior
// probability of certain differentials.
type CMApplicator struct {
	log *zap.Logger
}

// ContextModifier describes a single context-dependent adjustment to the
// differential log-odds. Modifiers are sourced from KB-20 patient profile
// and KB-21 behavioural intelligence during session initialisation.
type ContextModifier struct {
	ModifierID    string   `json:"modifier_id"`
	ModifierType  string   `json:"modifier_type"` // CONCOMITANT_DRUG, COMORBIDITY, LIFESTYLE, LAB_RESULT, NODE_CM, etc.
	Effect        string   `json:"effect"`         // INCREASE_PRIOR, DECREASE_PRIOR, HARD_BLOCK, OVERRIDE, SYMPTOM_MODIFICATION
	Magnitude     float64  `json:"magnitude"`      // Base magnitude in [0, 1) range
	DrugClass     string   `json:"drug_class,omitempty"`
	Differentials []string `json:"differentials,omitempty"` // affected differential IDs; empty = all

	// G5 fields — populated only for HARD_BLOCK and OVERRIDE effect types.
	BlockedTreatment string             `json:"blocked_treatment,omitempty"` // HARD_BLOCK: treatment name (e.g. "NITRATE_THERAPY")
	OverrideTargets  map[string]float64 `json:"override_targets,omitempty"` // OVERRIDE: differential_id -> min posterior
}

// NewCMApplicator creates a new CMApplicator instance.
func NewCMApplicator(log *zap.Logger) *CMApplicator {
	return &CMApplicator{
		log: log,
	}
}

// Apply processes all context modifiers against the current log-odds state and
// returns the updated log-odds along with a cm_log_deltas map recording each
// modifier's contribution for the audit trail.
//
// G14 (BAY-1): CM composition uses logit-based delta conversion to correctly
// map author-specified probability deltas to log-odds space. When multiple CMs
// fire on the same differential, their log-odds deltas are summed additively
// (which is mathematically correct in log-odds space, unlike naive probability
// addition which can exceed 1.0).
//
// For each modifier:
//   - F-03 adherence scaling: for CONCOMITANT_DRUG type modifiers, the magnitude
//     is scaled by min(1.0, adherence_score / 0.70). This ensures that patients
//     with low medication adherence receive attenuated drug-based prior shifts.
//   - G14 logit-based delta: the adjusted magnitude is converted to a log-odds
//     delta using the logit shift formula:
//     INCREASE_PRIOR: delta = logit(0.50 + adj_mag) - logit(0.50) = logit(0.50 + adj_mag)
//     DECREASE_PRIOR: delta = logit(0.50 - adj_mag) - logit(0.50) = logit(0.50 - adj_mag)
//     (since logit(0.50) = 0.0)
//   - The delta is applied additively to each affected differential's log-odds.
//   - Total CM shift per differential is capped at ±maxCMLogOddsShift to prevent
//     extreme posteriors from polypharmacy patients with ≥3 concurrent CMs.
//
// If adherenceWeights is nil or missing an entry for a modifier's drug_class,
// a default adherence of 1.0 is used (no attenuation).
func (a *CMApplicator) Apply(
	logOdds map[string]float64,
	modifiers []ContextModifier,
	adherenceWeights map[string]float64,
) (map[string]float64, map[string]float64) {
	cmLogDeltas := make(map[string]float64, len(modifiers))

	if adherenceWeights == nil {
		adherenceWeights = make(map[string]float64)
	}

	// G14: Track cumulative CM shift per differential for capping.
	cmCumulative := make(map[string]float64, len(logOdds))

	// Gap 2: Track CM count per differential for CM_STACKED warning.
	cmCountPerDiff := make(map[string]int, len(logOdds))

	// Snapshot pre-CM log-odds so we can enforce the cap correctly.
	preCMLogOdds := make(map[string]float64, len(logOdds))
	for diffID, lo := range logOdds {
		preCMLogOdds[diffID] = lo
	}

	for _, mod := range modifiers {
		// Gap 1: Passthrough effects — record firing but apply zero log-odds shift.
		// HARD_BLOCK and OVERRIDE are consumed by G5 (downstream safety).
		// SYMPTOM_MODIFICATION is consumed by G8 (LR suppression, deferred).
		switch mod.Effect {
		case "HARD_BLOCK", "OVERRIDE", "SYMPTOM_MODIFICATION":
			cmLogDeltas[mod.ModifierID] = 0.0
			a.log.Info("passthrough CM recorded (no log-odds shift)",
				zap.String("modifier_id", mod.ModifierID),
				zap.String("effect", mod.Effect),
				zap.Strings("differentials", mod.Differentials),
			)
			continue
		}

		if mod.Magnitude <= 0 || mod.Magnitude >= 0.50 {
			// G14: Magnitude represents a probability delta from 0.50 baseline.
			// Must be in (0, 0.50) to keep logit(0.50 ± mag) in valid range.
			a.log.Warn("skipping modifier with out-of-range magnitude",
				zap.String("modifier_id", mod.ModifierID),
				zap.Float64("magnitude", mod.Magnitude),
			)
			continue
		}

		adjMag := mod.Magnitude

		// F-03: adherence scaling for concomitant drug modifiers
		if mod.ModifierType == "CONCOMITANT_DRUG" {
			adherenceScore := 1.0
			if mod.DrugClass != "" {
				if score, ok := adherenceWeights[mod.DrugClass]; ok {
					adherenceScore = score
				}
			}
			// Scale magnitude: full effect at 70%+ adherence, linearly reduced below
			scaleFactor := math.Min(1.0, adherenceScore/0.70)
			adjMag = mod.Magnitude * scaleFactor

			a.log.Debug("adherence scaling applied",
				zap.String("modifier_id", mod.ModifierID),
				zap.String("drug_class", mod.DrugClass),
				zap.Float64("adherence_score", adherenceScore),
				zap.Float64("scale_factor", scaleFactor),
				zap.Float64("adjusted_magnitude", adjMag),
			)
		}

		// G14: Compute log-odds delta using logit shift formula.
		// delta_logodds = logit(0.50 + delta_p) - logit(0.50)
		// Since logit(0.50) = 0.0, this simplifies to logit(0.50 ± adjMag).
		var delta float64
		switch mod.Effect {
		case "INCREASE_PRIOR":
			delta = cmLogit(0.50 + adjMag)
		case "DECREASE_PRIOR":
			delta = cmLogit(0.50 - adjMag)
		default:
			a.log.Warn("unknown modifier effect, skipping",
				zap.String("modifier_id", mod.ModifierID),
				zap.String("effect", mod.Effect),
			)
			continue
		}

		cmLogDeltas[mod.ModifierID] = delta

		// Determine target differentials
		targets := mod.Differentials
		if len(targets) == 0 {
			// Apply to all differentials
			targets = make([]string, 0, len(logOdds))
			for diffID := range logOdds {
				targets = append(targets, diffID)
			}
		}

		// Apply delta additively to each affected differential, with cap
		for _, diffID := range targets {
			if _, exists := logOdds[diffID]; !exists {
				a.log.Warn("modifier references unknown differential, skipping",
					zap.String("modifier_id", mod.ModifierID),
					zap.String("differential_id", diffID),
				)
				continue
			}

			// Gap 2: Increment CM count for this differential
			cmCountPerDiff[diffID]++
			if cmCountPerDiff[diffID] == cmStackedThreshold {
				a.log.Warn("CM_STACKED: ≥3 context modifiers target same differential",
					zap.String("differential_id", diffID),
					zap.Int("cm_count", cmCountPerDiff[diffID]),
				)
			}

			// G14: Track cumulative shift and enforce cap
			cappedDelta := delta
			newCumulative := cmCumulative[diffID] + cappedDelta
			if newCumulative > maxCMLogOddsShift {
				cappedDelta = maxCMLogOddsShift - cmCumulative[diffID]
				a.log.Warn("CM log-odds shift capped at maximum",
					zap.String("modifier_id", mod.ModifierID),
					zap.String("differential_id", diffID),
					zap.Float64("cap", maxCMLogOddsShift),
				)
			} else if newCumulative < -maxCMLogOddsShift {
				cappedDelta = -maxCMLogOddsShift - cmCumulative[diffID]
				a.log.Warn("CM log-odds shift capped at minimum",
					zap.String("modifier_id", mod.ModifierID),
					zap.String("differential_id", diffID),
					zap.Float64("cap", -maxCMLogOddsShift),
				)
			}

			cmCumulative[diffID] += cappedDelta
			logOdds[diffID] = preCMLogOdds[diffID] + cmCumulative[diffID]
		}

		a.log.Debug("context modifier applied",
			zap.String("modifier_id", mod.ModifierID),
			zap.String("type", mod.ModifierType),
			zap.String("effect", mod.Effect),
			zap.Float64("delta", delta),
			zap.Int("targets", len(targets)),
		)
	}

	return logOdds, cmLogDeltas
}

// ExpandNodeCMs converts compact YAML context modifier definitions into flat
// ContextModifier structs suitable for CMApplicator.Apply(). Each adjustment
// entry (differential -> magnitude) becomes a separate ContextModifier with
// the appropriate Effect type.
//
// G5: When EffectType is set, it overrides the default INCREASE_PRIOR.
// HARD_BLOCK CMs produce a single ContextModifier with BlockedTreatment metadata.
// OVERRIDE CMs produce a single ContextModifier with OverrideTargets map.
// Default (empty or INCREASE_PRIOR): one CM per adjustment entry.
func ExpandNodeCMs(defs []models.ContextModifierDef) []ContextModifier {
	var expanded []ContextModifier
	for _, def := range defs {
		effectType := def.EffectType
		if effectType == "" {
			effectType = models.CMEffectIncreasePrior
		}

		switch effectType {
		case models.CMEffectHardBlock:
			// G5: HARD_BLOCK produces a single CM with treatment metadata.
			// Adjustments may contain minimal diagnostic shifts (e.g. ACS: 0.01)
			// but the primary purpose is the blocked_treatment signal.
			diffs := make([]string, 0, len(def.Adjustments))
			for diffID := range def.Adjustments {
				diffs = append(diffs, diffID)
			}
			expanded = append(expanded, ContextModifier{
				ModifierID:       def.ID,
				ModifierType:     "NODE_CM",
				Effect:           models.CMEffectHardBlock,
				Differentials:    diffs,
				BlockedTreatment: def.BlockedTreatment,
			})

		case models.CMEffectOverride:
			// G5: OVERRIDE produces a single CM with posterior minimum targets.
			// OverrideTargets map specifies differential -> min posterior value.
			diffs := make([]string, 0, len(def.OverrideTargets))
			for diffID := range def.OverrideTargets {
				diffs = append(diffs, diffID)
			}
			expanded = append(expanded, ContextModifier{
				ModifierID:      def.ID,
				ModifierType:    "NODE_CM",
				Effect:          models.CMEffectOverride,
				Differentials:   diffs,
				OverrideTargets: def.OverrideTargets,
			})

		default:
			// Standard INCREASE_PRIOR / DECREASE_PRIOR: one CM per adjustment entry.
			for diffID, mag := range def.Adjustments {
				expanded = append(expanded, ContextModifier{
					ModifierID:    def.ID,
					ModifierType:  "NODE_CM",
					Effect:        effectType,
					Magnitude:     mag,
					Differentials: []string{diffID},
				})
			}
		}
	}
	return expanded
}

// cmLogit converts a probability to log-odds for CM delta computation.
// Clamped to avoid log(0) or log(inf).
func cmLogit(p float64) float64 {
	const epsilon = 1e-15
	if p <= epsilon {
		p = epsilon
	}
	if p >= 1.0-epsilon {
		p = 1.0 - epsilon
	}
	return math.Log(p / (1.0 - p))
}
