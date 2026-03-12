package services

import (
	"math"
	"strings"
	"time"

	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/metrics"
	"kb-22-hpi-engine/internal/models"
)

// SelectionMode indicates which algorithm was used to pick the last question (BAY-8).
type SelectionMode string

const (
	SelectionModeIG          SelectionMode = "INFORMATION_GAIN"
	SelectionModeAuthorOrder SelectionMode = "AUTHOR_ORDER"
	SelectionModeMandatory   SelectionMode = "MANDATORY"
	SelectionModeSafetyGuard SelectionMode = "SAFETY_GUARD"
)

// QuestionOrchestrator selects the next question to ask using entropy-maximising
// information gain (Gap A01). The selection algorithm:
//  1. Mandatory unanswered questions are returned first (in definition order).
//  2. Among eligible optional questions, the one with the highest expected
//     information gain (one-step lookahead) is selected.
//  3. Ties are broken by preferring questions with minimum_inclusion_guard=true,
//     then by definition order.
//
// BAY-8 hybrid selection: when no question has any LR data (all LR+ and LR-
// maps are empty or default), IG computation degenerates to zero for all
// candidates. In that case, the orchestrator falls back to YAML author-order.
type QuestionOrchestrator struct {
	log               *zap.Logger
	metrics           *metrics.Collector
	lastSelectionMode SelectionMode
}

// LastSelectionMode returns the algorithm used to select the most recent question (BAY-8).
func (o *QuestionOrchestrator) LastSelectionMode() SelectionMode {
	return o.lastSelectionMode
}

// NewQuestionOrchestrator creates a new QuestionOrchestrator.
func NewQuestionOrchestrator(log *zap.Logger, metrics *metrics.Collector) *QuestionOrchestrator {
	return &QuestionOrchestrator{
		log:     log,
		metrics: metrics,
	}
}

// Next selects the next question to present to the patient. Returns nil when
// no more eligible questions remain.
//
// Selection priority:
//  1. Unanswered mandatory questions (in YAML definition order), subject to branch conditions
//  2. R-05: minimum_inclusion_guard questions not yet answered
//  3. Entropy-maximising optional question from eligible candidates
//  4. nil if no candidates remain
func (o *QuestionOrchestrator) Next(
	node *models.NodeDefinition,
	logOdds map[string]float64,
	answeredQuestions map[string]bool,
	stratum string,
	ckdSubstage *string,
	answers map[string]string,
) *models.QuestionDef {
	// Priority 1: unanswered mandatory questions in definition order
	for i := range node.Questions {
		q := &node.Questions[i]
		if q.Mandatory && !answeredQuestions[q.ID] {
			if q.BranchCondition == nil || o.EvaluateBranchCondition(*q.BranchCondition, stratum, ckdSubstage, answers) {
				o.log.Debug("selected mandatory question",
					zap.String("question_id", q.ID),
				)
				o.lastSelectionMode = SelectionModeMandatory
				return q
			}
		}
	}

	// Priority 2: R-05 minimum_inclusion_guard questions
	for i := range node.Questions {
		q := &node.Questions[i]
		if q.MinimumInclusionGuard && !q.Mandatory && !answeredQuestions[q.ID] {
			if q.BranchCondition == nil || o.EvaluateBranchCondition(*q.BranchCondition, stratum, ckdSubstage, answers) {
				o.log.Debug("selected safety-guard question",
					zap.String("question_id", q.ID),
				)
				o.lastSelectionMode = SelectionModeSafetyGuard
				return q
			}
		}
	}

	// Priority 3: entropy-maximising selection from eligible optional questions
	eligible := o.GetEligibleQuestions(node, answeredQuestions, stratum, ckdSubstage, answers)
	if len(eligible) == 0 {
		o.log.Debug("no eligible questions remain")
		return nil
	}

	start := time.Now()

	type scoredQuestion struct {
		question *models.QuestionDef
		ig       float64
	}

	scored := make([]scoredQuestion, 0, len(eligible))
	for _, q := range eligible {
		ig := o.ComputeExpectedIG(q, logOdds)
		scored = append(scored, scoredQuestion{question: q, ig: ig})
	}

	// Build a definition-order index for tie-breaking
	questionIndex := make(map[string]int, len(node.Questions))
	for i, q := range node.Questions {
		questionIndex[q.ID] = i
	}

	// BAY-8: Check whether any candidate has nonzero IG.
	// If all scores are zero (no LR data available), fall back to author-order.
	hasNonZeroIG := false
	for _, s := range scored {
		if s.ig > 0 {
			hasNonZeroIG = true
			break
		}
	}

	if !hasNonZeroIG {
		// BAY-8 fallback: no LR data → return first eligible in YAML definition order
		firstEligible := scored[0].question
		firstIdx := questionIndex[firstEligible.ID]
		for _, s := range scored[1:] {
			idx := questionIndex[s.question.ID]
			if idx < firstIdx {
				firstEligible = s.question
				firstIdx = idx
			}
		}

		elapsed := time.Since(start)
		o.metrics.EntropyComputation.Observe(float64(elapsed.Milliseconds()))
		o.lastSelectionMode = SelectionModeAuthorOrder

		o.log.Debug("BAY-8: falling back to author-order (no LR data)",
			zap.String("question_id", firstEligible.ID),
			zap.Int("candidates", len(eligible)),
			zap.Duration("computation_time", elapsed),
		)

		return firstEligible
	}

	best := scored[0]
	for _, s := range scored[1:] {
		if s.ig > best.ig {
			best = s
		} else if s.ig == best.ig {
			// Tie-break 1: prefer minimum_inclusion_guard
			if s.question.MinimumInclusionGuard && !best.question.MinimumInclusionGuard {
				best = s
			} else if s.question.MinimumInclusionGuard == best.question.MinimumInclusionGuard {
				// Tie-break 2: prefer earlier definition order
				if questionIndex[s.question.ID] < questionIndex[best.question.ID] {
					best = s
				}
			}
		}
	}

	elapsed := time.Since(start)
	o.metrics.EntropyComputation.Observe(float64(elapsed.Milliseconds()))
	o.lastSelectionMode = SelectionModeIG

	o.log.Debug("selected entropy-maximising question",
		zap.String("question_id", best.question.ID),
		zap.Float64("expected_ig", best.ig),
		zap.Int("candidates", len(eligible)),
		zap.Duration("computation_time", elapsed),
		zap.String("selection_mode", string(o.lastSelectionMode)),
	)

	return best.question
}

// SelectNext is a backward-compatible entry point that delegates to Next.
// It maps the legacy parameter set to the Next signature by constructing
// stratum/answers from the session context.
func (o *QuestionOrchestrator) SelectNext(
	node *models.NodeDefinition,
	answeredQuestions map[string]bool,
	logOdds map[string]float64,
	clusterAnswered map[string]int,
	questionsAsked int,
) *models.QuestionDef {
	// Check max_questions limit
	if questionsAsked >= node.MaxQuestions {
		o.log.Debug("max questions reached",
			zap.Int("asked", questionsAsked),
			zap.Int("max", node.MaxQuestions),
		)
		return nil
	}

	// SelectNext does not have stratum/ckdSubstage/answers context,
	// so branch conditions that depend on those cannot be evaluated.
	// Use a default stratum and empty answers for backward compatibility.
	return o.Next(node, logOdds, answeredQuestions, "", nil, nil)
}

// ComputeExpectedIG computes the expected information gain from asking a question
// using one-step lookahead:
//
//	IG(q) = H_current - sum_a[ P(a|q) * H(posterior | q=a) ]
//
// where a ranges over {YES, NO}. PATA_NAHI is excluded because it provides
// zero information gain by definition (F-04).
//
// P(a=YES|q) is approximated as the weighted average of LR+ across differentials
// normalised by the sum of LR+ and LR-. This is a heuristic that works well for
// symptom questions with binary answers.
func (o *QuestionOrchestrator) ComputeExpectedIG(question *models.QuestionDef, logOdds map[string]float64) float64 {
	hCurrent := computeEntropyFromLogOdds(logOdds)

	// Estimate P(YES) and P(NO) based on current posterior-weighted LRs
	pYes := estimateAnswerProbability(question.LRPositive, question.LRNegative, logOdds)
	pNo := 1.0 - pYes

	// Simulate YES answer
	logOddsYes := copyLogOdds(logOdds)
	for diffID, lo := range logOddsYes {
		if lr, ok := question.LRPositive[diffID]; ok && lr > 0 {
			logOddsYes[diffID] = lo + math.Log(lr)
		}
	}
	hYes := computeEntropyFromLogOdds(logOddsYes)

	// Simulate NO answer
	logOddsNo := copyLogOdds(logOdds)
	for diffID, lo := range logOddsNo {
		if lr, ok := question.LRNegative[diffID]; ok && lr > 0 {
			logOddsNo[diffID] = lo + math.Log(lr)
		}
	}
	hNo := computeEntropyFromLogOdds(logOddsNo)

	// Expected entropy after asking the question
	hExpected := pYes*hYes + pNo*hNo

	ig := hCurrent - hExpected
	if ig < 0 || math.IsNaN(ig) || math.IsInf(ig, 0) {
		ig = 0 // numerical floor
	}

	return ig
}

// EvaluateBranchCondition parses and evaluates a branch condition string that
// determines whether a question is eligible for the current session context.
//
// Supported syntax:
//
//	stratum == DM_HTN_CKD
//	stratum == DM_HTN_CKD AND ckd_substage IN [G3b, G4, G5]
//	Q001 == YES
//	stratum == DM_ONLY AND Q003 == NO
//
// Atoms are joined by AND. Each atom uses == for equality or IN for set membership.
func (o *QuestionOrchestrator) EvaluateBranchCondition(
	condition string,
	stratum string,
	ckdSubstage *string,
	answers map[string]string,
) bool {
	condition = strings.TrimSpace(condition)
	if condition == "" {
		return true // empty condition = always eligible
	}

	// Split on AND
	parts := strings.Split(condition, " AND ")
	for _, part := range parts {
		atom := strings.TrimSpace(part)
		if atom == "" {
			continue
		}

		if !o.evaluateBranchAtom(atom, stratum, ckdSubstage, answers) {
			return false
		}
	}

	return true
}

// GetEligibleQuestions returns all unanswered, non-mandatory questions whose
// branch conditions evaluate to true in the current context. Questions with
// minimum_inclusion_guard are excluded here since they are handled at higher
// priority in Next.
func (o *QuestionOrchestrator) GetEligibleQuestions(
	node *models.NodeDefinition,
	answeredQuestions map[string]bool,
	stratum string,
	ckdSubstage *string,
	answers map[string]string,
) []*models.QuestionDef {
	return o.GetEligibleQuestionsWithCMs(node, answeredQuestions, stratum, ckdSubstage, answers, nil)
}

// GetEligibleQuestionsWithCMs extends GetEligibleQuestions with G19 skip-redundancy.
// When firedCMIDs is non-nil, questions whose cm_coverage CMs have ALL fired are
// skipped — the CM already provides the signal this question would capture.
//
// BAY-8 rationale: if CM05 (beta-blocker comorbidity) fires, a question like
// "Are you taking beta-blockers?" is redundant because the CM already shifted
// the prior based on confirmed medication data from KB-20.
func (o *QuestionOrchestrator) GetEligibleQuestionsWithCMs(
	node *models.NodeDefinition,
	answeredQuestions map[string]bool,
	stratum string,
	ckdSubstage *string,
	answers map[string]string,
	firedCMIDs map[string]bool,
) []*models.QuestionDef {
	eligible := make([]*models.QuestionDef, 0)

	for i := range node.Questions {
		q := &node.Questions[i]

		// Skip already answered
		if answeredQuestions[q.ID] {
			continue
		}

		// Skip mandatory (handled separately in Next)
		if q.Mandatory {
			continue
		}

		// Skip minimum_inclusion_guard (handled separately in Next)
		if q.MinimumInclusionGuard {
			continue
		}

		// Evaluate branch condition
		if q.BranchCondition != nil {
			if !o.EvaluateBranchCondition(*q.BranchCondition, stratum, ckdSubstage, answers) {
				continue
			}
		}

		// G19: Skip-redundancy — skip if all listed CMs have fired
		if len(q.CMCoverage) > 0 && firedCMIDs != nil && o.allCMsFired(q.CMCoverage, firedCMIDs) {
			o.log.Debug("G19: skipping CM-covered question",
				zap.String("question_id", q.ID),
				zap.Strings("cm_coverage", q.CMCoverage),
			)
			continue
		}

		eligible = append(eligible, q)
	}

	return eligible
}

// allCMsFired returns true when every CM ID in the coverage list has fired.
func (o *QuestionOrchestrator) allCMsFired(cmIDs []string, firedCMIDs map[string]bool) bool {
	for _, cmID := range cmIDs {
		if !firedCMIDs[cmID] {
			return false
		}
	}
	return true
}

// evaluateBranchAtom evaluates a single branch condition atom.
// Supported forms:
//
//	stratum == VALUE
//	ckd_substage IN [V1, V2, V3]
//	Qxxx == VALUE
func (o *QuestionOrchestrator) evaluateBranchAtom(
	atom string,
	stratum string,
	ckdSubstage *string,
	answers map[string]string,
) bool {
	// Check for IN operator first
	if strings.Contains(atom, " IN ") {
		return o.evaluateInAtom(atom, stratum, ckdSubstage)
	}

	// Check for == operator (with optional surrounding spaces)
	var lhs, rhs string
	if idx := strings.Index(atom, " == "); idx >= 0 {
		lhs = strings.TrimSpace(atom[:idx])
		rhs = strings.TrimSpace(atom[idx+4:])
	} else if idx := strings.Index(atom, "=="); idx >= 0 {
		lhs = strings.TrimSpace(atom[:idx])
		rhs = strings.TrimSpace(atom[idx+2:])
	} else {
		o.log.Warn("unparseable branch condition atom",
			zap.String("atom", atom),
		)
		return true // fail open for unparseable conditions
	}

	switch lhs {
	case "stratum":
		return strings.EqualFold(stratum, rhs)
	case "ckd_substage":
		if ckdSubstage == nil {
			return false
		}
		return strings.EqualFold(*ckdSubstage, rhs)
	default:
		// Treat as question ID reference (e.g., Q001 == YES)
		if answers == nil {
			return false
		}
		if val, ok := answers[lhs]; ok {
			return strings.EqualFold(val, rhs)
		}
		// Question not yet answered: condition not met
		return false
	}
}

// evaluateInAtom evaluates an IN-style condition: "ckd_substage IN [G3b, G4, G5]"
func (o *QuestionOrchestrator) evaluateInAtom(atom string, stratum string, ckdSubstage *string) bool {
	parts := strings.SplitN(atom, " IN ", 2)
	if len(parts) != 2 {
		o.log.Warn("unparseable IN condition",
			zap.String("atom", atom),
		)
		return true
	}

	lhs := strings.TrimSpace(parts[0])
	setStr := strings.TrimSpace(parts[1])

	// Parse the set: [V1, V2, V3]
	setStr = strings.Trim(setStr, "[]")
	members := strings.Split(setStr, ",")

	var actual string
	switch lhs {
	case "stratum":
		actual = stratum
	case "ckd_substage":
		if ckdSubstage == nil {
			return false
		}
		actual = *ckdSubstage
	default:
		o.log.Warn("IN condition with unknown variable",
			zap.String("variable", lhs),
		)
		return true
	}

	for _, member := range members {
		if strings.EqualFold(strings.TrimSpace(member), actual) {
			return true
		}
	}

	return false
}

// --- helper functions ---

// computeEntropyFromLogOdds converts log-odds to normalised probabilities and
// computes Shannon entropy. This is a standalone function used by
// ComputeExpectedIG's one-step lookahead simulation.
func computeEntropyFromLogOdds(logOdds map[string]float64) float64 {
	if len(logOdds) == 0 {
		return 0.0
	}

	totalRaw := 0.0
	rawProbs := make([]float64, 0, len(logOdds))
	for _, lo := range logOdds {
		p := 1.0 / (1.0 + math.Exp(-lo)) // inline sigmoid
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

// estimateAnswerProbability estimates P(answer=YES) given the current posteriors
// and the question's LR+ and LR- values. Uses a posterior-weighted average of
// the positive likelihood ratios relative to total likelihood.
func estimateAnswerProbability(lrPos, lrNeg map[string]float64, logOdds map[string]float64) float64 {
	// Get normalised posteriors
	totalRaw := 0.0
	posteriors := make(map[string]float64, len(logOdds))
	for diffID, lo := range logOdds {
		p := 1.0 / (1.0 + math.Exp(-lo))
		posteriors[diffID] = p
		totalRaw += p
	}
	if totalRaw > 0 {
		for diffID := range posteriors {
			posteriors[diffID] /= totalRaw
		}
	}

	// Weighted average: P(YES) = sum_d[ P(d) * LR+(d) / (LR+(d) + LR-(d)) ]
	pYes := 0.0
	for diffID, post := range posteriors {
		lrP := lrPos[diffID]
		lrN := lrNeg[diffID]
		if lrP <= 0 {
			lrP = 1.0
		}
		if lrN <= 0 {
			lrN = 1.0
		}
		total := lrP + lrN
		if total > 0 {
			pYes += post * (lrP / total)
		}
	}

	// Clamp to [0.05, 0.95] to avoid degenerate entropy calculations
	if pYes < 0.05 {
		pYes = 0.05
	}
	if pYes > 0.95 {
		pYes = 0.95
	}

	return pYes
}

// copyLogOdds creates a shallow copy of a log-odds map for simulation.
func copyLogOdds(src map[string]float64) map[string]float64 {
	dst := make(map[string]float64, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
