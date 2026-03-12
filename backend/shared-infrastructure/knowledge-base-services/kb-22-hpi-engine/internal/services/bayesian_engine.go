package services

import (
	"math"
	"sort"
	"strings"

	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/metrics"
	"kb-22-hpi-engine/internal/models"
)

// BayesianEngine implements the core Bayesian inference engine using log-odds
// representation (F-01). All internal state is maintained as log-odds to avoid
// floating-point underflow on repeated multiplications.
type BayesianEngine struct {
	log     *zap.Logger
	metrics *metrics.Collector
}

// NewBayesianEngine creates a new BayesianEngine instance.
func NewBayesianEngine(log *zap.Logger, metrics *metrics.Collector) *BayesianEngine {
	return &BayesianEngine{
		log:     log,
		metrics: metrics,
	}
}

// defaultOtherBucketPrior is used when OtherBucketPrior is not set in the node YAML.
const defaultOtherBucketPrior = 0.15

// InitPriors converts per-stratum prior probabilities to log-odds representation.
// Formula: lo = log(p / (1 - p))
// Priors are looked up from the node definition for the given stratum.
// If a differential has no prior for the requested stratum, a uniform prior
// of 1/N is used as a fallback.
//
// G3: Medication-conditional differentials. If activeMedClasses is non-nil,
// differentials with an activation_condition are evaluated against the patient's
// active medication classes. Excluded differentials have their prior mass
// redistributed proportionally across remaining active differentials — NOT into
// the G15 Other bucket. This prevents systematic Other inflation.
//
// G15: When node.OtherBucketEnabled is true, an implicit _OTHER differential
// is injected with prior = node.OtherBucketPrior (default 0.15). This acts as
// a probability sink for diagnoses not in the authored differential list.
func (e *BayesianEngine) InitPriors(node *models.NodeDefinition, stratum string, activeMedClasses []string) map[string]float64 {
	n := len(node.Differentials)
	capacity := n
	if node.OtherBucketEnabled {
		capacity++
	}
	logOdds := make(map[string]float64, capacity)
	uniformPrior := 1.0 / float64(n)

	// G3 Phase 1: Collect priors and determine which differentials are active.
	activePriors := make(map[string]float64, n)
	excludedPriorSum := 0.0

	for _, diff := range node.Differentials {
		prior, ok := diff.Priors[stratum]
		if !ok || prior <= 0 || prior >= 1 {
			prior = uniformPrior
			e.log.Warn("using uniform prior for differential",
				zap.String("differential_id", diff.ID),
				zap.String("stratum", stratum),
				zap.Float64("uniform_prior", uniformPrior),
			)
		}

		// G3: Check activation condition
		if diff.ActivationCondition != "" && !EvalActivationCondition(diff.ActivationCondition, activeMedClasses) {
			excludedPriorSum += prior
			e.log.Info("G3: differential excluded (activation condition not met)",
				zap.String("differential_id", diff.ID),
				zap.String("condition", diff.ActivationCondition),
				zap.Float64("excluded_prior", prior),
			)
			continue
		}

		activePriors[diff.ID] = prior
	}

	// G3 Phase 2: Redistribute excluded prior mass proportionally across active differentials.
	if excludedPriorSum > 0 && len(activePriors) > 0 {
		activePriorSum := 0.0
		for _, p := range activePriors {
			activePriorSum += p
		}
		if activePriorSum > 0 {
			scale := (activePriorSum + excludedPriorSum) / activePriorSum
			for diffID, p := range activePriors {
				activePriors[diffID] = p * scale
			}
			e.log.Info("G3: redistributed excluded prior mass proportionally",
				zap.Float64("excluded_mass", excludedPriorSum),
				zap.Float64("scale_factor", scale),
				zap.Int("active_differentials", len(activePriors)),
			)
		}
	}

	// Convert to log-odds
	for diffID, prior := range activePriors {
		logOdds[diffID] = logit(prior)
	}

	// G15: inject implicit _OTHER differential
	if node.OtherBucketEnabled {
		otherPrior := node.OtherBucketPrior
		if otherPrior <= 0 || otherPrior >= 1 {
			otherPrior = defaultOtherBucketPrior
		}
		logOdds[models.OtherBucketDiffID] = logit(otherPrior)
		e.log.Info("G15: injected _OTHER bucket differential",
			zap.Float64("prior", otherPrior),
			zap.Float64("log_odds", logOdds[models.OtherBucketDiffID]),
		)
	}

	e.log.Debug("initialised priors",
		zap.String("stratum", stratum),
		zap.Int("total_differentials", n),
		zap.Int("active_differentials", len(activePriors)),
		zap.Float64("excluded_prior_mass", excludedPriorSum),
		zap.Bool("other_bucket", node.OtherBucketEnabled),
	)

	return logOdds
}

// InitPriorsWithBPStatus extends InitPriors with G9 bp_status conditional prior
// overrides. When the node defines conditional_prior_overrides and bpStatus
// matches a key, the corresponding additive deltas are applied to base priors
// BEFORE log-odds conversion.
//
// The override deltas are applied AFTER G3 exclusion but BEFORE G3 redistribution,
// ensuring that excluded differentials don't receive overrides and that the
// redistributed mass accounts for the adjusted priors.
//
// If bpStatus is empty or the node has no matching overrides, this delegates
// directly to InitPriors with no changes.
func (e *BayesianEngine) InitPriorsWithBPStatus(
	node *models.NodeDefinition,
	stratum string,
	activeMedClasses []string,
	bpStatus string,
) map[string]float64 {
	// Fast path: no overrides defined or no bp_status provided
	if bpStatus == "" || len(node.ConditionalPriorOverrides) == 0 {
		return e.InitPriors(node, stratum, activeMedClasses)
	}

	overrides, exists := node.ConditionalPriorOverrides[bpStatus]
	if !exists || len(overrides) == 0 {
		return e.InitPriors(node, stratum, activeMedClasses)
	}

	// G9: Apply overrides to a temporary copy of the node's differentials.
	// We create a shallow copy of the node with adjusted priors.
	adjustedNode := *node
	adjustedDiffs := make([]models.DifferentialDef, len(node.Differentials))
	copy(adjustedDiffs, node.Differentials)

	for i, diff := range adjustedDiffs {
		if delta, hasDelta := overrides[diff.ID]; hasDelta {
			if prior, hasStratum := diff.Priors[stratum]; hasStratum {
				// Deep-copy priors map to avoid mutating the original node
				newPriors := make(map[string]float64, len(diff.Priors))
				for k, v := range diff.Priors {
					newPriors[k] = v
				}
				adjusted := prior + delta
				// Clamp to valid probability range
				if adjusted < 0.001 {
					adjusted = 0.001
				}
				if adjusted > 0.999 {
					adjusted = 0.999
				}
				newPriors[stratum] = adjusted
				adjustedDiffs[i].Priors = newPriors

				e.log.Info("G9: conditional prior override applied",
					zap.String("bp_status", bpStatus),
					zap.String("differential_id", diff.ID),
					zap.Float64("base_prior", prior),
					zap.Float64("delta", delta),
					zap.Float64("adjusted_prior", adjusted),
				)
			}
		}
	}
	adjustedNode.Differentials = adjustedDiffs

	return e.InitPriors(&adjustedNode, stratum, activeMedClasses)
}

// ApplySexModifiers evaluates G2 sex-modifier definitions against the patient's
// sex and age, applying matching adjustments as direct log-odds deltas. Unlike CMs
// (which use probability deltas via logit), sex modifiers specify OR-based log-odds
// shifts: OR 1.8 = log(1.8) = +0.59 log-odds.
//
// Called once during session initialisation after InitPriors and before CM application.
// Modifiers are evaluated in order; multiple matching modifiers accumulate additively.
func (e *BayesianEngine) ApplySexModifiers(
	logOdds map[string]float64,
	modifiers []models.SexModifierDef,
	patientSex string,
	patientAge int,
) {
	if len(modifiers) == 0 {
		return
	}

	for _, sm := range modifiers {
		if !EvalSexCondition(sm.Condition, patientSex, patientAge) {
			continue
		}

		for diffID, delta := range sm.Adjustments {
			if _, exists := logOdds[diffID]; !exists {
				e.log.Warn("G2: sex modifier references unknown differential",
					zap.String("modifier_id", sm.ID),
					zap.String("differential_id", diffID),
				)
				continue
			}
			logOdds[diffID] += delta
			e.log.Info("G2: sex modifier applied",
				zap.String("modifier_id", sm.ID),
				zap.String("differential_id", diffID),
				zap.Float64("log_odds_delta", delta),
				zap.Float64("new_log_odds", logOdds[diffID]),
			)
		}
	}
}

// EvalSexCondition evaluates a sex-modifier condition string against patient demographics.
// Supported patterns:
//
//	"sex == Female"
//	"sex == Male"
//	"sex == Female AND age >= 50"
//	"sex == Male AND age >= 55"
//
// Returns true if the condition is satisfied. Unknown formats return false (safe: skip).
func EvalSexCondition(condition string, patientSex string, patientAge int) bool {
	if condition == "" {
		return false
	}

	parts := strings.Split(condition, " AND ")

	for _, part := range parts {
		part = strings.TrimSpace(part)

		if strings.HasPrefix(part, "sex == ") {
			expected := strings.TrimPrefix(part, "sex == ")
			if !strings.EqualFold(patientSex, expected) {
				return false
			}
		} else if strings.HasPrefix(part, "age >= ") {
			threshStr := strings.TrimPrefix(part, "age >= ")
			thresh := 0
			for _, ch := range threshStr {
				if ch >= '0' && ch <= '9' {
					thresh = thresh*10 + int(ch-'0')
				}
			}
			if patientAge < thresh {
				return false
			}
		} else {
			// Unknown clause (e.g., "pain_quality == burning") — skip modifier (safe)
			return false
		}
	}

	return true
}

// EvalActivationCondition evaluates a differential's activation condition against
// the patient's active medication classes. Supports conditions of the form:
//
//	"med_class == SGLT2i"
//	"med_class == Metformin AND eGFR < 30"  (eGFR check deferred — med_class part only)
//
// Returns true if the condition is satisfied (differential should be included).
// Returns true for empty conditions (unconditional differential).
// If activeMedClasses is nil, all conditions evaluate to false (conservative:
// don't include medication-conditional differentials without medication data).
func EvalActivationCondition(condition string, activeMedClasses []string) bool {
	if condition == "" {
		return true
	}
	if activeMedClasses == nil {
		return false
	}

	// Parse "med_class == X" pattern (possibly with " AND ..." suffix)
	// For now, we only evaluate the med_class clause. Additional clauses
	// (e.g., "eGFR < 30") are treated as satisfied if the med_class matches.
	// This is a deliberate clinical safety choice: if a patient IS on metformin,
	// we want to include lactic acidosis in the differential even if we don't
	// have real-time eGFR — the floor (G1) protects against ruling it out.
	condition = strings.TrimSpace(condition)

	// Extract med_class value
	const prefix = "med_class == "
	idx := strings.Index(condition, prefix)
	if idx < 0 {
		// Unknown condition format — include by default (safe direction)
		return true
	}

	medClass := condition[idx+len(prefix):]
	// Trim at AND boundary if present
	if andIdx := strings.Index(medClass, " AND "); andIdx >= 0 {
		medClass = medClass[:andIdx]
	}
	medClass = strings.TrimSpace(medClass)

	// Check if the required med class is in the active list
	for _, active := range activeMedClasses {
		if strings.EqualFold(active, medClass) {
			return true
		}
	}
	return false
}

// ApplyGuidelineAdjustments applies additive log-odds adjustments from KB-1
// guideline prior injection (N-01). Each adjustment is added directly to the
// corresponding differential's log-odds.
func (e *BayesianEngine) ApplyGuidelineAdjustments(logOdds map[string]float64, adjustments map[string]float64) {
	for diffID, adj := range adjustments {
		if _, exists := logOdds[diffID]; !exists {
			e.log.Warn("guideline adjustment for unknown differential, skipping",
				zap.String("differential_id", diffID),
				zap.Float64("adjustment", adj),
			)
			continue
		}
		logOdds[diffID] += adj
		e.log.Debug("applied guideline adjustment",
			zap.String("differential_id", diffID),
			zap.Float64("adjustment", adj),
			zap.Float64("new_log_odds", logOdds[diffID]),
		)
	}
}

// Update applies a single answer to the log-odds state vector and returns
// the updated log-odds along with the observed information gain (H_before - H_after).
// Delegates to UpdateWithStratum with an empty stratum (base LR behaviour).
func (e *BayesianEngine) Update(
	logOdds map[string]float64,
	questionID string,
	answer string,
	question *models.QuestionDef,
	reliabilityModifier float64,
	adherenceGain float64,
	clusterAnswered map[string]int,
) (map[string]float64, float64) {
	return e.UpdateWithStratum(logOdds, questionID, answer, question, reliabilityModifier, adherenceGain, clusterAnswered, "")
}

// UpdateWithStratum applies a single answer to the log-odds state vector and returns
// the updated log-odds along with the observed information gain (H_before - H_after).
//
// G6: When stratum is non-empty and the question defines lr_positive_by_stratum or
// lr_negative_by_stratum for that stratum, the stratum-specific LR values are used
// instead of the base lr_positive/lr_negative. This handles cases where a symptom's
// discriminating power genuinely differs by stratum (e.g., orthopnea LR+ drops from
// 2.2 to 1.2 in CKD+HF patients). Per-differential fallback: if a differential is
// missing from the stratum-specific map, the base LR is used for that differential.
//
// Answer handling:
//   - YES:       lo_d += log(LR+) * reliability
//   - NO:        lo_d += log(LR-) * reliability
//   - PATA_NAHI: lo_d += 0.0 (F-04, no update)
//
// R-02 cluster dampening: if the question belongs to a cluster that has already
// been answered, the LR delta is multiplied by dampening^n where n is the number
// of previous answers in that cluster.
//
// R-03 reliability weighting: the LR delta is scaled by the reliability modifier.
//
// Adherence gain: the LR delta is further scaled by the adherence-tier gain
// factor derived from KB-21 (HIGH=1.0, MEDIUM=0.7, LOW=0.4).
func (e *BayesianEngine) UpdateWithStratum(
	logOdds map[string]float64,
	questionID string,
	answer string,
	question *models.QuestionDef,
	reliabilityModifier float64,
	adherenceGain float64,
	clusterAnswered map[string]int,
	stratum string,
) (map[string]float64, float64) {
	// Compute entropy before update
	hBefore := e.ComputeEntropy(logOdds)

	// F-04: PATA_NAHI contributes zero information
	if answer == string(models.AnswerPata) {
		hAfter := e.ComputeEntropy(logOdds)
		ig := hBefore - hAfter // should be ~0.0
		e.log.Debug("pata_nahi answer, no LR update",
			zap.String("question_id", questionID),
			zap.Float64("information_gain", ig),
		)
		return logOdds, ig
	}

	// Determine which LR map to use based on answer type (G10)
	var baseLRMap map[string]float64
	var stratumLRMap map[string]float64 // G6: stratum-specific override (may be nil)

	if question.AnswerType == models.AnswerTypeCategorical {
		// G10: CATEGORICAL — look up LR map by answer value from lr_categorical
		catMap, ok := question.LRCategorical[answer]
		if !ok {
			e.log.Error("categorical answer value not in lr_categorical, treating as pata_nahi",
				zap.String("question_id", questionID),
				zap.String("answer", answer),
				zap.Strings("valid_options", question.AnswerOptions),
			)
			return logOdds, 0.0
		}
		baseLRMap = catMap
		// G6 does not apply to CATEGORICAL questions (stratum-specific LRs
		// are only defined for BINARY lr_positive/lr_negative)
	} else {
		// BINARY (default): YES/NO
		switch answer {
		case string(models.AnswerYes):
			baseLRMap = question.LRPositive
			// G6: check for stratum-specific LR+ override
			if stratum != "" && question.LRPositiveByStratum != nil {
				if sMap, ok := question.LRPositiveByStratum[stratum]; ok {
					stratumLRMap = sMap
					e.log.Debug("G6: using stratum-specific LR+ override",
						zap.String("question_id", questionID),
						zap.String("stratum", stratum),
					)
				}
			}
		case string(models.AnswerNo):
			baseLRMap = question.LRNegative
			// G6: check for stratum-specific LR- override
			if stratum != "" && question.LRNegativeByStratum != nil {
				if sMap, ok := question.LRNegativeByStratum[stratum]; ok {
					stratumLRMap = sMap
					e.log.Debug("G6: using stratum-specific LR- override",
						zap.String("question_id", questionID),
						zap.String("stratum", stratum),
					)
				}
			}
		default:
			e.log.Error("unknown answer value, treating as pata_nahi",
				zap.String("question_id", questionID),
				zap.String("answer", answer),
			)
			return logOdds, 0.0
		}
	}

	// G6: merge stratum-specific LRs with base LRs. Stratum values take
	// precedence per-differential; base LR is the fallback.
	lrMap := baseLRMap
	if stratumLRMap != nil {
		lrMap = make(map[string]float64, len(baseLRMap))
		for diffID, lr := range baseLRMap {
			lrMap[diffID] = lr
		}
		for diffID, lr := range stratumLRMap {
			lrMap[diffID] = lr // override with stratum-specific value
		}
	}

	// R-02: compute cluster dampening factor
	dampeningFactor := 1.0
	if question.Cluster != "" {
		prevCount, exists := clusterAnswered[question.Cluster]
		if exists && prevCount >= 1 && question.ClusterDampening > 0 {
			dampeningFactor = math.Pow(question.ClusterDampening, float64(prevCount))
			e.log.Debug("applying cluster dampening",
				zap.String("cluster", question.Cluster),
				zap.Int("previous_count", prevCount),
				zap.Float64("dampening_base", question.ClusterDampening),
				zap.Float64("dampening_factor", dampeningFactor),
			)
		}
	}

	// G15: collect LR values for geometric mean computation (OTHER bucket).
	// We track log(LR) values for named differentials that have LR entries.
	var otherLogLRSum float64
	var otherLRCount int

	// Apply LR updates to each named differential
	for diffID := range logOdds {
		if diffID == models.OtherBucketDiffID {
			// _OTHER is updated separately below
			continue
		}

		lr, ok := lrMap[diffID]
		if !ok || lr <= 0 {
			// No LR defined for this differential/answer combination
			continue
		}

		lrDelta := math.Log(lr)

		// G15: accumulate for geometric mean computation
		otherLogLRSum += lrDelta
		otherLRCount++

		// R-02: apply cluster dampening
		lrDelta *= dampeningFactor

		// R-03: apply reliability weighting
		lrDelta *= reliabilityModifier

		// Apply adherence-tier gain factor (KB-21)
		lrDelta *= adherenceGain

		logOdds[diffID] += lrDelta
	}

	// G15: Update _OTHER bucket using geometric mean of inverse LRs.
	// delta_OTHER = log(1 / geomean(LRs)) = -mean(log(LRs))
	// If evidence supports named differentials (high LRs), OTHER decreases.
	// If evidence is equivocal (LRs near 1.0), OTHER stays roughly flat.
	if _, hasOther := logOdds[models.OtherBucketDiffID]; hasOther && otherLRCount > 0 {
		geomMeanLogLR := otherLogLRSum / float64(otherLRCount)
		otherDelta := -geomMeanLogLR // log(1/geomean) = -log(geomean)

		// Apply same scaling factors
		otherDelta *= dampeningFactor
		otherDelta *= reliabilityModifier
		otherDelta *= adherenceGain

		logOdds[models.OtherBucketDiffID] += otherDelta

		e.log.Debug("G15: updated _OTHER bucket",
			zap.Float64("geom_mean_log_lr", geomMeanLogLR),
			zap.Float64("other_delta", otherDelta),
			zap.Float64("new_other_log_odds", logOdds[models.OtherBucketDiffID]),
		)
	}

	// Compute entropy after update
	hAfter := e.ComputeEntropy(logOdds)
	ig := hBefore - hAfter

	e.log.Debug("updated log-odds",
		zap.String("question_id", questionID),
		zap.String("answer", answer),
		zap.Float64("reliability", reliabilityModifier),
		zap.Float64("adherence_gain", adherenceGain),
		zap.Float64("dampening", dampeningFactor),
		zap.Float64("information_gain", ig),
	)

	return logOdds, ig
}

// ResolveFloors returns the safety floor map for a given node and stratum.
// Resolution order: stratum-specific floors > simple floors > nil.
// Returns nil if no floors are defined or node is nil.
func ResolveFloors(node *models.NodeDefinition, stratum string) map[string]float64 {
	if node == nil {
		return nil
	}
	// Stratum-specific floors take precedence
	if node.SafetyFloorsByStratum != nil {
		if floors, ok := node.SafetyFloorsByStratum[stratum]; ok && len(floors) > 0 {
			return floors
		}
	}
	// Fall back to simple (all-strata) floors
	if len(node.SafetyFloors) > 0 {
		return node.SafetyFloors
	}
	return nil
}

// GetPosteriors converts log-odds to posterior probabilities using the sigmoid
// function and returns a sorted slice of DifferentialEntry (descending by probability).
//
// G1: Safety floor clamping. If safetyFloors is non-nil, any differential whose
// normalised posterior falls below its floor is clamped up to the floor value.
// After clamping, all posteriors are re-normalised to maintain the sum-to-1.0
// invariant. Clamped entries receive a SAFETY_FLOOR_ACTIVE flag.
// Safety floors do NOT apply to the _OTHER bucket.
//
// G15: The _OTHER bucket differential is annotated with IsOtherBucket=true.
// If OTHER's posterior exceeds OtherIncompleteThreshold (0.30), the entry
// receives a DIFFERENTIAL_INCOMPLETE flag. If it exceeds OtherEscalationThreshold
// (0.45), it additionally receives ESCALATE_INCOMPLETE.
func (e *BayesianEngine) GetPosteriors(logOdds map[string]float64, safetyFloors map[string]float64) []models.DifferentialEntry {
	entries := make([]models.DifferentialEntry, 0, len(logOdds))

	// Convert log-odds to raw sigmoid values
	rawProbs := make(map[string]float64, len(logOdds))
	totalRaw := 0.0
	for diffID, lo := range logOdds {
		p := sigmoid(lo)
		rawProbs[diffID] = p
		totalRaw += p
	}

	// Phase 1: Normalise to ensure posteriors sum to 1.0
	normalised := make(map[string]float64, len(logOdds))
	for diffID := range logOdds {
		if totalRaw > 0 {
			normalised[diffID] = rawProbs[diffID] / totalRaw
		}
	}

	// Phase 2: G1 safety floor clamping (after normalisation, before sorting).
	// Clamp differentials below their floor. Re-normalise by fixing floored
	// values in place and redistributing remaining mass among non-floored entries.
	flooredSet := make(map[string]bool) // tracks which diffs are at their floor
	if len(safetyFloors) > 0 {
		for diffID, floor := range safetyFloors {
			if _, exists := normalised[diffID]; !exists {
				continue // floor references a differential not in this session
			}
			if diffID == models.OtherBucketDiffID {
				continue // floors never apply to _OTHER bucket
			}
			if normalised[diffID] < floor {
				e.log.Info("G1: safety floor clamped differential",
					zap.String("differential_id", diffID),
					zap.Float64("pre_clamp", normalised[diffID]),
					zap.Float64("floor", floor),
				)
				normalised[diffID] = floor
				flooredSet[diffID] = true
			}
		}

		// Re-normalise: fix floored values, redistribute remaining mass proportionally.
		if len(flooredSet) > 0 {
			flooredTotal := 0.0
			freeTotal := 0.0
			for diffID, p := range normalised {
				if flooredSet[diffID] {
					flooredTotal += p
				} else {
					freeTotal += p
				}
			}
			remainingMass := 1.0 - flooredTotal
			if freeTotal > 0 && remainingMass > 0 {
				scale := remainingMass / freeTotal
				for diffID := range normalised {
					if !flooredSet[diffID] {
						normalised[diffID] *= scale
					}
				}
			}
		}
	}

	// Phase 3: Build entries with G15 annotations and G1 flags
	for diffID, lo := range logOdds {
		entry := models.DifferentialEntry{
			DifferentialID:       diffID,
			PosteriorProbability: normalised[diffID],
			LogOdds:              lo,
		}

		// G1: flag differentials that were clamped to their safety floor
		if flooredSet[diffID] {
			entry.Flags = append(entry.Flags, "SAFETY_FLOOR_ACTIVE")
		}

		// G15: annotate _OTHER bucket with flags
		if diffID == models.OtherBucketDiffID {
			entry.IsOtherBucket = true
			entry.Label = "Other / unlisted diagnosis"

			if normalised[diffID] >= models.OtherEscalationThreshold {
				entry.Flags = append(entry.Flags, "DIFFERENTIAL_INCOMPLETE", "ESCALATE_INCOMPLETE")
				e.log.Warn("G15: _OTHER posterior exceeds escalation threshold",
					zap.Float64("posterior", normalised[diffID]),
					zap.Float64("threshold", models.OtherEscalationThreshold),
				)
			} else if normalised[diffID] >= models.OtherIncompleteThreshold {
				entry.Flags = append(entry.Flags, "DIFFERENTIAL_INCOMPLETE")
				e.log.Warn("G15: _OTHER posterior exceeds incomplete threshold",
					zap.Float64("posterior", normalised[diffID]),
					zap.Float64("threshold", models.OtherIncompleteThreshold),
				)
			}
		}

		entries = append(entries, entry)
	}

	// Sort descending by posterior probability
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].PosteriorProbability > entries[j].PosteriorProbability
	})

	return entries
}

// CheckConvergence evaluates the R-01 dual-criterion termination condition.
// Returns whether convergence is reached and the index of the top diagnosis.
//
// Convergence logic modes:
//   - BOTH:           top posterior >= threshold AND gap to #2 >= gap_threshold
//   - EITHER:         top posterior >= threshold OR  gap to #2 >= gap_threshold
//   - POSTERIOR_ONLY:  top posterior >= threshold
func (e *BayesianEngine) CheckConvergence(posteriors []models.DifferentialEntry, node *models.NodeDefinition) (bool, int) {
	if len(posteriors) == 0 {
		return false, -1
	}

	topPosterior := posteriors[0].PosteriorProbability
	posteriorMet := topPosterior >= node.ConvergenceThreshold

	gap := 0.0
	if len(posteriors) > 1 {
		gap = topPosterior - posteriors[1].PosteriorProbability
	} else {
		// Only one differential; gap is effectively 1.0
		gap = 1.0
	}
	gapMet := gap >= node.PosteriorGapThreshold

	var converged bool
	switch node.ConvergenceLogic {
	case "BOTH":
		converged = posteriorMet && gapMet
	case "EITHER":
		converged = posteriorMet || gapMet
	case "POSTERIOR_ONLY":
		converged = posteriorMet
	default:
		// Default to BOTH for safety
		converged = posteriorMet && gapMet
		e.log.Warn("unknown convergence logic, defaulting to BOTH",
			zap.String("convergence_logic", node.ConvergenceLogic),
		)
	}

	if converged {
		e.log.Info("convergence reached",
			zap.String("top_differential", posteriors[0].DifferentialID),
			zap.Float64("top_posterior", topPosterior),
			zap.Float64("gap", gap),
			zap.String("logic", node.ConvergenceLogic),
		)
		e.metrics.DifferentialConverged.Inc()
	}

	return converged, 0
}

// ConvergenceResult holds the detailed outcome of a G18 multi-criteria convergence check.
type ConvergenceResult struct {
	Converged          bool    `json:"converged"`
	TopDifferentialIdx int     `json:"top_differential_idx"`
	PosteriorMet       bool    `json:"posterior_met"`
	GapMet             bool    `json:"gap_met"`
	// G18 fields
	DecisiveConfidence   float64 `json:"decisive_confidence"`   // confidence of the decisive answer (highest IG answer)
	ConfidenceMet        bool    `json:"confidence_met"`         // decisive confidence >= 0.75
	SupportingAnswers    int     `json:"supporting_answers"`     // count of non-PATA_NAHI answers with IG > 0
	SupportingAnswersMet bool    `json:"supporting_answers_met"` // supporting answers >= 2
}

// CheckConvergenceMultiCriteria extends CheckConvergence with G18 answer quality gates.
// In addition to the R-01 dual-criterion (posterior threshold + gap), closure also requires:
//   - Decisive answer confidence > 0.75 (the answer that contributed the most IG must be reliable)
//   - Supporting answers >= 2 (at least 2 non-PATA_NAHI answers contributed positive IG)
//
// This prevents premature closure on a single low-confidence answer.
//
// answerConfidences maps question_id -> confidence score (0.0–1.0) from KB-21.
// answerIGs maps question_id -> observed information gain from that answer.
// Both are optional — if nil, the G18 criteria are treated as met (backward-compatible).
func (e *BayesianEngine) CheckConvergenceMultiCriteria(
	posteriors []models.DifferentialEntry,
	node *models.NodeDefinition,
	answerConfidences map[string]float64,
	answerIGs map[string]float64,
) ConvergenceResult {
	result := ConvergenceResult{
		TopDifferentialIdx: -1,
	}

	if len(posteriors) == 0 {
		return result
	}

	result.TopDifferentialIdx = 0
	topPosterior := posteriors[0].PosteriorProbability

	// R-01 criterion 1: posterior threshold
	result.PosteriorMet = topPosterior >= node.ConvergenceThreshold

	// R-01 criterion 2: gap to #2
	gap := 0.0
	if len(posteriors) > 1 {
		gap = topPosterior - posteriors[1].PosteriorProbability
	} else {
		gap = 1.0
	}
	result.GapMet = gap >= node.PosteriorGapThreshold

	// R-01 convergence logic
	var r01Met bool
	switch node.ConvergenceLogic {
	case "BOTH":
		r01Met = result.PosteriorMet && result.GapMet
	case "EITHER":
		r01Met = result.PosteriorMet || result.GapMet
	case "POSTERIOR_ONLY":
		r01Met = result.PosteriorMet
	default:
		r01Met = result.PosteriorMet && result.GapMet
	}

	// G18: answer quality gates
	if answerConfidences == nil && answerIGs == nil {
		// No quality data — backward-compatible: G18 criteria auto-pass
		result.ConfidenceMet = true
		result.SupportingAnswersMet = true
		result.DecisiveConfidence = 1.0
		result.SupportingAnswers = 2
		result.Converged = r01Met
	} else {
		// Find decisive answer: the one with the highest IG
		var maxIG float64
		var decisiveQID string
		supportCount := 0

		if answerIGs != nil {
			for qid, ig := range answerIGs {
				if ig > 0 {
					supportCount++
				}
				if ig > maxIG {
					maxIG = ig
					decisiveQID = qid
				}
			}
		}

		result.SupportingAnswers = supportCount
		result.SupportingAnswersMet = supportCount >= 2

		// Get confidence of the decisive answer
		if answerConfidences != nil && decisiveQID != "" {
			if conf, ok := answerConfidences[decisiveQID]; ok {
				result.DecisiveConfidence = conf
			} else {
				result.DecisiveConfidence = 1.0 // no confidence data for this answer → assume reliable
			}
		} else {
			result.DecisiveConfidence = 1.0
		}
		result.ConfidenceMet = result.DecisiveConfidence >= 0.75

		// All criteria must be met
		result.Converged = r01Met && result.ConfidenceMet && result.SupportingAnswersMet

		if r01Met && !result.Converged {
			e.log.Info("G18: convergence blocked by answer quality gate",
				zap.Bool("confidence_met", result.ConfidenceMet),
				zap.Float64("decisive_confidence", result.DecisiveConfidence),
				zap.Bool("supporting_met", result.SupportingAnswersMet),
				zap.Int("supporting_count", result.SupportingAnswers),
				zap.String("decisive_question", decisiveQID),
			)
		}
	}

	if result.Converged {
		e.log.Info("convergence reached (multi-criteria)",
			zap.String("top_differential", posteriors[0].DifferentialID),
			zap.Float64("top_posterior", topPosterior),
			zap.Float64("gap", gap),
			zap.String("logic", node.ConvergenceLogic),
		)
		e.metrics.DifferentialConverged.Inc()
	}

	return result
}

// ComputeEntropy calculates the Shannon entropy of the posterior distribution
// derived from the current log-odds state.
// H = -sum(p_i * log(p_i)) for all differentials where p_i > 0.
func (e *BayesianEngine) ComputeEntropy(logOdds map[string]float64) float64 {
	if len(logOdds) == 0 {
		return 0.0
	}

	// Convert to probabilities via sigmoid and normalise
	totalRaw := 0.0
	rawProbs := make([]float64, 0, len(logOdds))
	for _, lo := range logOdds {
		p := sigmoid(lo)
		rawProbs = append(rawProbs, p)
		totalRaw += p
	}

	if totalRaw == 0 {
		return 0.0
	}

	entropy := 0.0
	for _, raw := range rawProbs {
		p := raw / totalRaw
		if p > 0 {
			entropy -= p * math.Log(p)
		}
	}

	return entropy
}

// --- mathematical primitives ---

// sigmoid converts log-odds to probability: p = 1 / (1 + exp(-lo))
func sigmoid(lo float64) float64 {
	return 1.0 / (1.0 + math.Exp(-lo))
}

// logit converts probability to log-odds: lo = log(p / (1 - p))
func logit(p float64) float64 {
	// Clamp to avoid log(0) or log(inf)
	const epsilon = 1e-15
	if p <= epsilon {
		p = epsilon
	}
	if p >= 1.0-epsilon {
		p = 1.0 - epsilon
	}
	return math.Log(p / (1.0 - p))
}
