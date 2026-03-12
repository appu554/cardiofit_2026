package services

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/models"
)

// TransitionEvaluator evaluates G13 node transition rules against current
// session state. It runs synchronously after each answer (not in a goroutine)
// because transition decisions require the latest posteriors.
//
// Transition conditions supported:
//   - "posterior:DIFF_ID >= VALUE"  — differential posterior threshold
//   - "questions_asked >= N"       — question count threshold
//   - "converged"                  — node has reached convergence
//   - "safety_flag:FLAG_ID"        — specific safety flag has fired
type TransitionEvaluator struct {
	log *zap.Logger
}

// TransitionSessionState captures the session state needed for transition evaluation.
type TransitionSessionState struct {
	Posteriors     map[string]float64 // differential_id -> posterior probability
	QuestionsAsked int
	Converged      bool
	FiredSafetyIDs map[string]bool // set of safety flag IDs that have fired
}

// NewTransitionEvaluator creates a new TransitionEvaluator.
func NewTransitionEvaluator(log *zap.Logger) *TransitionEvaluator {
	return &TransitionEvaluator{log: log}
}

// Evaluate checks all transition rules for a node against current session state.
// Returns triggered transitions sorted by priority (lower number = higher priority).
// Only the highest-priority transition per target node is returned.
func (te *TransitionEvaluator) Evaluate(
	transitions []models.NodeTransitionDef,
	state TransitionSessionState,
) []models.TransitionEvent {
	if len(transitions) == 0 {
		return nil
	}

	// Sort by priority (stable sort preserves YAML order for equal priorities)
	sorted := make([]models.NodeTransitionDef, len(transitions))
	copy(sorted, transitions)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Priority < sorted[j].Priority
	})

	var events []models.TransitionEvent
	seenTargets := make(map[string]bool)

	for _, t := range sorted {
		if seenTargets[t.TargetNode] {
			continue // only highest-priority transition per target
		}

		if te.evaluateCondition(t.TriggerCondition, state) {
			seenTargets[t.TargetNode] = true
			events = append(events, models.TransitionEvent{
				TransitionID: t.ID,
				SourceNode:   "", // filled by caller who knows the source node
				TargetNode:   t.TargetNode,
				Mode:         t.Mode,
				Reason:       t.TriggerCondition,
			})

			te.log.Info("G13: node transition triggered",
				zap.String("transition_id", t.ID),
				zap.String("target_node", t.TargetNode),
				zap.String("mode", t.Mode),
				zap.String("condition", t.TriggerCondition),
			)
		}
	}

	return events
}

// evaluateCondition parses and evaluates a single transition condition string.
func (te *TransitionEvaluator) evaluateCondition(
	condition string,
	state TransitionSessionState,
) bool {
	condition = strings.TrimSpace(condition)
	if condition == "" {
		return false
	}

	// "converged" — simple keyword
	if condition == "converged" {
		return state.Converged
	}

	// "safety_flag:FLAG_ID"
	if strings.HasPrefix(condition, "safety_flag:") {
		flagID := strings.TrimPrefix(condition, "safety_flag:")
		flagID = strings.TrimSpace(flagID)
		return state.FiredSafetyIDs[flagID]
	}

	// "posterior:DIFF_ID >= VALUE"
	if strings.HasPrefix(condition, "posterior:") {
		return te.evaluatePosteriorCondition(condition, state.Posteriors)
	}

	// "questions_asked >= N"
	if strings.HasPrefix(condition, "questions_asked") {
		return te.evaluateComparisonCondition(condition, "questions_asked", float64(state.QuestionsAsked))
	}

	te.log.Warn("G13: unknown transition condition format",
		zap.String("condition", condition))
	return false
}

// evaluatePosteriorCondition handles "posterior:DIFF_ID >= 0.40".
func (te *TransitionEvaluator) evaluatePosteriorCondition(
	condition string,
	posteriors map[string]float64,
) bool {
	// Strip "posterior:" prefix
	rest := strings.TrimPrefix(condition, "posterior:")
	rest = strings.TrimSpace(rest)

	// Parse: "DIFF_ID >= VALUE" or "DIFF_ID > VALUE" etc.
	diffID, op, threshold, err := parseComparison(rest)
	if err != nil {
		te.log.Warn("G13: failed to parse posterior condition",
			zap.String("condition", condition),
			zap.Error(err))
		return false
	}

	actual, exists := posteriors[diffID]
	if !exists {
		return false
	}

	return compareValues(actual, op, threshold)
}

// evaluateComparisonCondition handles "field >= N" style conditions.
func (te *TransitionEvaluator) evaluateComparisonCondition(
	condition string, field string, actual float64,
) bool {
	rest := strings.TrimPrefix(condition, field)
	rest = strings.TrimSpace(rest)

	_, op, threshold, err := parseComparisonRest("_", rest)
	if err != nil {
		// Try direct parse of "op value"
		op2, val, err2 := parseOpValue(rest)
		if err2 != nil {
			te.log.Warn("G13: failed to parse comparison condition",
				zap.String("condition", condition),
				zap.Error(err))
			return false
		}
		return compareValues(actual, op2, val)
	}

	return compareValues(actual, op, threshold)
}

// parseComparison parses "IDENTIFIER OP VALUE" from a string.
func parseComparison(s string) (identifier string, op string, value float64, err error) {
	// Try operators in order of length to avoid prefix matching issues
	for _, operator := range []string{">=", "<=", "!=", ">", "<", "=="} {
		idx := strings.Index(s, operator)
		if idx > 0 {
			identifier = strings.TrimSpace(s[:idx])
			valStr := strings.TrimSpace(s[idx+len(operator):])
			value, err = strconv.ParseFloat(valStr, 64)
			if err != nil {
				return "", "", 0, fmt.Errorf("invalid threshold value %q: %w", valStr, err)
			}
			return identifier, operator, value, nil
		}
	}
	return "", "", 0, fmt.Errorf("no comparison operator found in %q", s)
}

// parseComparisonRest parses " OP VALUE" when identifier is already known.
func parseComparisonRest(identifier string, rest string) (string, string, float64, error) {
	return parseComparison(identifier + " " + rest)
}

// parseOpValue parses "OP VALUE" directly (e.g., ">= 8").
func parseOpValue(s string) (op string, value float64, err error) {
	s = strings.TrimSpace(s)
	for _, operator := range []string{">=", "<=", "!=", ">", "<", "=="} {
		if strings.HasPrefix(s, operator) {
			valStr := strings.TrimSpace(s[len(operator):])
			value, err = strconv.ParseFloat(valStr, 64)
			if err != nil {
				return "", 0, fmt.Errorf("invalid value %q: %w", valStr, err)
			}
			return operator, value, nil
		}
	}
	return "", 0, fmt.Errorf("no operator found in %q", s)
}

// compareValues applies a comparison operator between two float64 values.
func compareValues(actual float64, op string, threshold float64) bool {
	const epsilon = 1e-9
	switch op {
	case ">=":
		return actual >= threshold-epsilon
	case ">":
		return actual > threshold+epsilon
	case "<=":
		return actual <= threshold+epsilon
	case "<":
		return actual < threshold-epsilon
	case "==":
		return math.Abs(actual-threshold) < epsilon
	case "!=":
		return math.Abs(actual-threshold) >= epsilon
	default:
		return false
	}
}
