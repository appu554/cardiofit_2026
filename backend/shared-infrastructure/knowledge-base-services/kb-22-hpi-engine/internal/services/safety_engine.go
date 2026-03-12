package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/metrics"
	"kb-22-hpi-engine/internal/models"
)

// SafetyEngine runs as a parallel goroutine (F-02) that continuously evaluates
// safety trigger conditions against accumulated answers. It operates independently
// from the Bayesian inference loop with its own recover() to ensure a panic in
// safety evaluation never crashes the main session.
type SafetyEngine struct {
	log     *zap.Logger
	metrics *metrics.Collector
}

// AnswerEvent is sent from the main session loop to the SafetyEngine goroutine
// each time a patient answers a question.
type AnswerEvent struct {
	QuestionID string
	Answer     string
	SessionID  uuid.UUID
}

// NewSafetyEngine creates a new SafetyEngine instance.
func NewSafetyEngine(log *zap.Logger, metrics *metrics.Collector) *SafetyEngine {
	return &SafetyEngine{
		log:     log,
		metrics: metrics,
	}
}

// Start launches the safety evaluation goroutine. It reads AnswerEvents from
// answerChan and evaluates all trigger conditions against the accumulated answer
// state. Fired safety flags are sent to flagChan for the main session loop to
// consume. The goroutine terminates when answerChan is closed.
//
// F-02: the goroutine has its own defer recover() so a panic in condition parsing
// or evaluation does not propagate to the session handler.
func (e *SafetyEngine) Start(
	triggers []models.SafetyTriggerDef,
	crossNodeTriggers []models.CrossNodeTrigger,
	answerChan <-chan AnswerEvent,
	flagChan chan<- models.SafetyFlag,
	sessionID uuid.UUID,
) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				e.log.Error("safety engine recovered from panic",
					zap.String("session_id", sessionID.String()),
					zap.Any("panic", r),
				)
			}
		}()

		// Accumulated answer state for condition evaluation
		answers := make(map[string]string)
		// Track which triggers have already fired to avoid duplicates
		firedTriggers := make(map[string]bool)

		e.log.Debug("safety engine started",
			zap.String("session_id", sessionID.String()),
			zap.Int("trigger_count", len(triggers)),
			zap.Int("cross_node_trigger_count", len(crossNodeTriggers)),
		)

		for event := range answerChan {
			answers[event.QuestionID] = event.Answer

			// Evaluate node-level triggers
			for _, trigger := range triggers {
				if firedTriggers[trigger.ID] {
					continue
				}
				// G12/R-06: route to appropriate evaluator based on type
				triggered := false
				if trigger.Type == "COMPOSITE_SCORE" {
					triggered = e.EvaluateCompositeScore(trigger, answers)
				} else {
					triggered = e.ParseCondition(trigger.Condition, answers)
				}
				if triggered {
					firedTriggers[trigger.ID] = true

					flag := models.SafetyFlag{
						FlagID:            trigger.ID,
						SessionID:         sessionID,
						Severity:          models.SafetyLevel(trigger.Severity),
						TriggerExpression: trigger.Condition,
						RecommendedAction: trigger.Action,
						FiredAt:           time.Now(),
					}

					e.log.Warn("safety trigger fired",
						zap.String("flag_id", trigger.ID),
						zap.String("severity", trigger.Severity),
						zap.String("session_id", sessionID.String()),
						zap.String("condition", trigger.Condition),
					)
					e.metrics.SafetyFlagsRaised.WithLabelValues(trigger.Severity, trigger.ID).Inc()

					// Non-blocking send; if flagChan is full, log and skip
					select {
					case flagChan <- flag:
					default:
						e.log.Error("flag channel full, dropping safety flag",
							zap.String("flag_id", trigger.ID),
							zap.String("session_id", sessionID.String()),
						)
					}
				}
			}

			// Evaluate cross-node triggers (F-07)
			for _, crossTrigger := range crossNodeTriggers {
				if !crossTrigger.Active {
					continue
				}
				triggerKey := "cross_" + crossTrigger.TriggerID
				if firedTriggers[triggerKey] {
					continue
				}
				if e.ParseCondition(crossTrigger.Condition, answers) {
					firedTriggers[triggerKey] = true

					flag := models.SafetyFlag{
						FlagID:            crossTrigger.TriggerID,
						SessionID:         sessionID,
						Severity:          models.SafetyLevel(crossTrigger.Severity),
						TriggerExpression: crossTrigger.Condition,
						RecommendedAction: crossTrigger.RecommendedAction,
						FiredAt:           time.Now(),
					}

					e.log.Warn("cross-node safety trigger fired",
						zap.String("flag_id", crossTrigger.TriggerID),
						zap.String("severity", crossTrigger.Severity),
						zap.String("session_id", sessionID.String()),
					)
					e.metrics.SafetyFlagsRaised.WithLabelValues(crossTrigger.Severity, crossTrigger.TriggerID).Inc()

					select {
					case flagChan <- flag:
					default:
						e.log.Error("flag channel full, dropping cross-node safety flag",
							zap.String("flag_id", crossTrigger.TriggerID),
							zap.String("session_id", sessionID.String()),
						)
					}
				}
			}
		}

		e.log.Debug("safety engine stopped, answer channel closed",
			zap.String("session_id", sessionID.String()),
		)
	}()
}

// EvaluateTriggers is a synchronous batch evaluation of all triggers against
// a complete answer set. Used for session resume or snapshot generation when
// the goroutine-based evaluation is not active.
func (e *SafetyEngine) EvaluateTriggers(
	triggers []models.SafetyTriggerDef,
	answers map[string]string,
) []models.SafetyFlag {
	var flags []models.SafetyFlag

	for _, trigger := range triggers {
		triggered := false
		if trigger.Type == "COMPOSITE_SCORE" {
			triggered = e.EvaluateCompositeScore(trigger, answers)
		} else {
			triggered = e.ParseCondition(trigger.Condition, answers)
		}
		if triggered {
			flag := models.SafetyFlag{
				FlagID:            trigger.ID,
				Severity:          models.SafetyLevel(trigger.Severity),
				TriggerExpression: trigger.Condition,
				RecommendedAction: trigger.Action,
				FiredAt:           time.Now(),
			}
			flags = append(flags, flag)

			e.log.Debug("trigger evaluated true",
				zap.String("flag_id", trigger.ID),
				zap.String("severity", trigger.Severity),
			)
		}
	}

	return flags
}

// ParseCondition evaluates a boolean expression against the current answer state.
// Supported syntax:
//
//	Q001=YES AND Q003=YES
//	Q001=YES OR Q002=NO
//	Q001=YES AND Q003=YES AND Q005=YES
//	Q001=YES OR Q003=NO OR Q007=YES
//
// Mixed AND/OR in a single expression is evaluated left-to-right with AND having
// higher precedence than OR (standard boolean algebra). For simplicity, the current
// implementation evaluates AND-groups first, then OR-combines them.
//
// An atom "Qxxx=VALUE" evaluates to true if the answer map contains that question
// with the specified value. If the question has not been answered yet, the atom
// evaluates to false.
func (e *SafetyEngine) ParseCondition(condition string, answers map[string]string) bool {
	condition = strings.TrimSpace(condition)
	if condition == "" {
		return false
	}

	// Split by OR first (lower precedence), then evaluate AND groups
	orGroups := splitOnOperator(condition, "OR")
	for _, orGroup := range orGroups {
		andAtoms := splitOnOperator(orGroup, "AND")
		allTrue := true
		for _, atom := range andAtoms {
			if !evaluateAtom(strings.TrimSpace(atom), answers) {
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

// splitOnOperator splits a condition string on the given boolean operator,
// respecting whitespace boundaries to avoid splitting on substrings.
func splitOnOperator(expr string, op string) []string {
	// Use a delimiter with surrounding spaces to avoid false matches
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

// evaluateAtom evaluates a single condition atom of the form "Q001=YES".
func evaluateAtom(atom string, answers map[string]string) bool {
	return evaluateAtomWithCMs(atom, answers, nil)
}

// evaluateAtomWithCMs evaluates a single condition atom against answers and
// fired context modifiers. Supports two atom forms:
//   - "Q001=YES"      — question-answer match (original)
//   - "CM_CKD=FIRED"  — G8: context modifier fired check
//
// For CM atoms, the left-hand side is the CM ID and the right-hand side must
// be "FIRED". The atom evaluates to true if firedCMs contains that CM ID.
func evaluateAtomWithCMs(atom string, answers map[string]string, firedCMs map[string]bool) bool {
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

// ParseConditionWithCMs evaluates a boolean expression against the current answer
// state AND fired context modifier state (G8). Extends ParseCondition to support
// CM_ID=FIRED atoms alongside Q_ID=ANSWER atoms.
//
// Example condition: "Q001=YES AND CM_CKD=FIRED AND Q003=YES"
// This fires only when the patient answered YES to Q001, the CKD context modifier
// has fired, AND Q003 is answered YES.
func (e *SafetyEngine) ParseConditionWithCMs(condition string, answers map[string]string, firedCMs map[string]bool) bool {
	condition = strings.TrimSpace(condition)
	if condition == "" {
		return false
	}

	orGroups := splitOnOperator(condition, "OR")
	for _, orGroup := range orGroups {
		andAtoms := splitOnOperator(orGroup, "AND")
		allTrue := true
		for _, atom := range andAtoms {
			if !evaluateAtomWithCMs(strings.TrimSpace(atom), answers, firedCMs) {
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

// EvaluateTriggersWithCMs is a CM-aware variant of EvaluateTriggers (G8).
// Evaluates all triggers using both the answer state and fired CM state.
func (e *SafetyEngine) EvaluateTriggersWithCMs(
	triggers []models.SafetyTriggerDef,
	answers map[string]string,
	firedCMs map[string]bool,
) []models.SafetyFlag {
	var flags []models.SafetyFlag

	for _, trigger := range triggers {
		triggered := false
		if trigger.Type == "COMPOSITE_SCORE" {
			triggered = e.EvaluateCompositeScore(trigger, answers)
		} else {
			triggered = e.ParseConditionWithCMs(trigger.Condition, answers, firedCMs)
		}
		if triggered {
			flag := models.SafetyFlag{
				FlagID:            trigger.ID,
				Severity:          models.SafetyLevel(trigger.Severity),
				TriggerExpression: trigger.Condition,
				RecommendedAction: trigger.Action,
				FiredAt:           time.Now(),
			}
			flags = append(flags, flag)

			e.log.Debug("G8: trigger evaluated true (CM-aware)",
				zap.String("flag_id", trigger.ID),
				zap.String("severity", trigger.Severity),
			)
		}
	}

	return flags
}

// EvaluateCompositeScore evaluates a G12/R-06 COMPOSITE_SCORE safety trigger.
// Weights are keyed as "QUESTION_ID=ANSWER_VALUE" → weight (float64).
// For each weight entry, if the patient's answer to that question matches the
// expected value, the weight is added to the running score. The trigger fires
// when the accumulated score >= threshold.
//
// Example YAML:
//
//	type: COMPOSITE_SCORE
//	weights:
//	  Q001=YES: 0.3    # chest pain present
//	  Q003=YES: 0.25   # diaphoresis
//	  Q007=YES: 0.2    # radiating to arm
//	  Q009=YES: 0.15   # age > 55
//	  Q012=NO:  0.1    # no relief with rest
//	threshold: 0.6
//
// This captures graduated risk accumulation: no single symptom fires the trigger,
// but sufficient co-occurrence of weighted risk factors does.
func (e *SafetyEngine) EvaluateCompositeScore(
	trigger models.SafetyTriggerDef,
	answers map[string]string,
) bool {
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

	fired := score >= trigger.Threshold
	if fired {
		e.log.Info("G12: composite score trigger fired",
			zap.String("trigger_id", trigger.ID),
			zap.Float64("score", score),
			zap.Float64("threshold", trigger.Threshold),
		)
	}
	return fired
}
