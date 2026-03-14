// Package services — evaluator.go is the core Safety Constraint Engine.
// It accumulates answers per session and evaluates safety triggers independently
// of the Bayesian inference loop. Runs in parallel with KB-22's M2 engine and
// can veto M2's output via escalation to KB-19.
package services

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-24-safety-constraint-engine/internal/models"
)

// SafetyTriggerEvaluator is the stateful safety evaluation engine.
// It maintains per-session answer accumulation and evaluates safety triggers
// from node definitions against the accumulated answer state.
type SafetyTriggerEvaluator struct {
	nodeLoader *NodeLoader
	log        *zap.Logger

	// Per-session answer accumulation. Key: session_id, Value: map[question_id]answer.
	// Protected by RWMutex for concurrent read access during evaluation.
	sessions   map[string]map[string]string
	sessionsMu sync.RWMutex
}

// NewSafetyTriggerEvaluator creates the core evaluator with its dependencies.
func NewSafetyTriggerEvaluator(nodeLoader *NodeLoader, log *zap.Logger) *SafetyTriggerEvaluator {
	return &SafetyTriggerEvaluator{
		nodeLoader: nodeLoader,
		log:        log,
		sessions:   make(map[string]map[string]string),
	}
}

// Evaluate processes a single answer through the safety trigger engine.
// It accumulates the answer for this session, then evaluates all safety triggers
// defined on the node. Returns a clear result if the node is not found (unknown
// nodes cannot have safety triggers).
func (e *SafetyTriggerEvaluator) Evaluate(
	sessionID uuid.UUID,
	nodeID string,
	questionID string,
	answer string,
	firedCMs map[string]bool,
) *models.EvaluateResponse {
	node := e.nodeLoader.Get(nodeID)
	if node == nil {
		e.log.Debug("node not found, returning clear",
			zap.String("node_id", nodeID),
			zap.String("session_id", sessionID.String()),
		)
		return &models.EvaluateResponse{Clear: true}
	}

	// Accumulate answer for this session and snapshot the current state
	sid := sessionID.String()
	answers := e.accumulateAnswer(sid, questionID, answer)

	// Evaluate all safety triggers (CM-aware via G8 protocol)
	var flags []models.SafetyFlag
	for _, trigger := range node.SafetyTriggers {
		fired := false
		if trigger.Type == "COMPOSITE_SCORE" {
			fired = evaluateCompositeScore(trigger, answers)
		} else if firedCMs != nil && len(firedCMs) > 0 {
			fired = parseConditionWithCMs(trigger.Condition, answers, firedCMs)
		} else {
			fired = parseCondition(trigger.Condition, answers)
		}

		if fired {
			flag := models.SafetyFlag{
				FlagID:            trigger.ID,
				Severity:          models.SafetyLevel(trigger.Severity),
				RecommendedAction: trigger.Action,
				FiredAt:           time.Now(),
			}
			flags = append(flags, flag)

			e.log.Warn("safety trigger fired",
				zap.String("session_id", sid),
				zap.String("flag_id", trigger.ID),
				zap.String("severity", trigger.Severity),
				zap.String("condition", trigger.Condition),
			)
		}
	}

	result := &models.EvaluateResponse{
		Clear: len(flags) == 0,
		Flags: flags,
	}

	// IMMEDIATE severity triggers escalation — KB-19 overrides M2
	for _, flag := range flags {
		if flag.Severity == models.SafetyImmediate {
			result.EscalationRequired = true
			result.ReasonCode = flag.FlagID
			break
		}
	}

	if result.EscalationRequired {
		e.log.Warn("SCE escalation triggered",
			zap.String("session_id", sid),
			zap.String("reason_code", result.ReasonCode),
			zap.Int("flag_count", len(flags)),
		)
	}

	return result
}

// ClearSession removes all accumulated answer state for a session.
// Called when a session completes, is abandoned, or is explicitly cleared.
func (e *SafetyTriggerEvaluator) ClearSession(sessionID uuid.UUID) {
	e.sessionsMu.Lock()
	delete(e.sessions, sessionID.String())
	e.sessionsMu.Unlock()

	e.log.Debug("session cleared",
		zap.String("session_id", sessionID.String()),
	)
}

// SessionCount returns the number of active sessions (for health/metrics).
func (e *SafetyTriggerEvaluator) SessionCount() int {
	e.sessionsMu.RLock()
	defer e.sessionsMu.RUnlock()
	return len(e.sessions)
}

// accumulateAnswer stores the answer and returns a snapshot of the full answer map.
func (e *SafetyTriggerEvaluator) accumulateAnswer(sessionID, questionID, answer string) map[string]string {
	e.sessionsMu.Lock()
	defer e.sessionsMu.Unlock()

	if e.sessions[sessionID] == nil {
		e.sessions[sessionID] = make(map[string]string)
	}
	e.sessions[sessionID][questionID] = answer

	// Return a snapshot to avoid holding the lock during evaluation
	snapshot := make(map[string]string, len(e.sessions[sessionID]))
	for k, v := range e.sessions[sessionID] {
		snapshot[k] = v
	}
	return snapshot
}

// parseCondition evaluates a boolean expression against the current answer state.
// Supports: Q001=YES AND Q003=YES, Q001=YES OR Q002=NO.
// AND has higher precedence than OR (standard boolean algebra).
func parseCondition(condition string, answers map[string]string) bool {
	condition = strings.TrimSpace(condition)
	if condition == "" {
		return false
	}

	orGroups := splitOnOperator(condition, "OR")
	for _, orGroup := range orGroups {
		andAtoms := splitOnOperator(orGroup, "AND")
		allTrue := true
		for _, atom := range andAtoms {
			if !evaluateAtom(strings.TrimSpace(atom), answers, nil) {
				allTrue = false
				break
			}
		}
		if allTrue {
			return true
		}
	}
	return false
}

// parseConditionWithCMs extends parseCondition with CM-aware evaluation (G8).
// Supports CM_ID=FIRED atoms alongside Q_ID=ANSWER atoms.
func parseConditionWithCMs(condition string, answers map[string]string, firedCMs map[string]bool) bool {
	condition = strings.TrimSpace(condition)
	if condition == "" {
		return false
	}

	orGroups := splitOnOperator(condition, "OR")
	for _, orGroup := range orGroups {
		andAtoms := splitOnOperator(orGroup, "AND")
		allTrue := true
		for _, atom := range andAtoms {
			if !evaluateAtom(strings.TrimSpace(atom), answers, firedCMs) {
				allTrue = false
				break
			}
		}
		if allTrue {
			return true
		}
	}
	return false
}

// evaluateAtom evaluates a single condition atom.
// Supports: "Q001=YES" (question-answer match) and "CM_CKD=FIRED" (G8 CM check).
func evaluateAtom(atom string, answers map[string]string, firedCMs map[string]bool) bool {
	parts := strings.SplitN(atom, "=", 2)
	if len(parts) != 2 {
		return false
	}

	lhs := strings.TrimSpace(parts[0])
	rhs := strings.TrimSpace(parts[1])

	// G8: CM-aware atom — "CM_ID=FIRED"
	if strings.EqualFold(rhs, "FIRED") && firedCMs != nil {
		return firedCMs[lhs]
	}

	// Standard question-answer atom
	actualValue, answered := answers[lhs]
	if !answered {
		return false
	}
	return strings.EqualFold(actualValue, rhs)
}

// splitOnOperator splits a condition string on the given boolean operator,
// respecting whitespace boundaries.
func splitOnOperator(expr string, op string) []string {
	delimiter := fmt.Sprintf(" %s ", op)
	parts := strings.Split(expr, delimiter)

	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// evaluateCompositeScore evaluates a COMPOSITE_SCORE safety trigger (G12/R-06).
// Weights are keyed as "QUESTION_ID=ANSWER_VALUE" -> weight. The trigger fires
// when the accumulated score of matching answers meets or exceeds the threshold.
func evaluateCompositeScore(trigger models.SafetyTriggerDef, answers map[string]string) bool {
	if len(trigger.Weights) == 0 || trigger.Threshold <= 0 {
		return false
	}

	var score float64
	for key, weight := range trigger.Weights {
		parts := strings.SplitN(key, "=", 2)
		if len(parts) != 2 {
			continue
		}
		qid := strings.TrimSpace(parts[0])
		expectedAnswer := strings.TrimSpace(parts[1])

		actualAnswer, answered := answers[qid]
		if answered && strings.EqualFold(actualAnswer, expectedAnswer) {
			score += weight
		}
	}

	return score >= trigger.Threshold
}
