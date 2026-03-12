package services

import (
	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/models"
)

// ContradictionDetector implements G17 (BAY-4): contradiction detection between
// question pairs. When both questions in a contradiction pair have been answered
// YES, the detector flags the contradiction and recommends re-asking the second
// question using its alt_prompt.
//
// The detector is called after each answer in the session loop. It does NOT
// automatically modify the answer state — the session handler decides whether
// to insert a re-ask into the question queue.
type ContradictionDetector struct {
	log *zap.Logger
}

// NewContradictionDetector creates a new ContradictionDetector.
func NewContradictionDetector(log *zap.Logger) *ContradictionDetector {
	return &ContradictionDetector{log: log}
}

// Check evaluates all contradiction pairs against the current answer state.
// Returns any newly detected contradictions (pairs where both questions have
// been answered YES). The alreadyDetected set prevents duplicate detections
// across multiple calls within the same session.
func (cd *ContradictionDetector) Check(
	pairs []models.ContradictionPairDef,
	answers map[string]string,
	alreadyDetected map[string]bool,
) []models.ContradictionEvent {
	if len(pairs) == 0 {
		return nil
	}

	var events []models.ContradictionEvent

	for _, pair := range pairs {
		if alreadyDetected[pair.ID] {
			continue
		}

		ansA, hasA := answers[pair.QuestionA]
		ansB, hasB := answers[pair.QuestionB]

		if !hasA || !hasB {
			continue // both questions must have been answered
		}

		// Contradiction fires when both answers are YES
		if ansA == string(models.AnswerYes) && ansB == string(models.AnswerYes) {
			events = append(events, models.ContradictionEvent{
				PairID:        pair.ID,
				QuestionA:     pair.QuestionA,
				QuestionB:     pair.QuestionB,
				ReaskQuestion: pair.QuestionB,
				UseAltPrompt:  true,
			})

			cd.log.Warn("G17: contradiction detected",
				zap.String("pair_id", pair.ID),
				zap.String("question_a", pair.QuestionA),
				zap.String("question_b", pair.QuestionB),
				zap.String("description", pair.Description),
			)
		}
	}

	return events
}
