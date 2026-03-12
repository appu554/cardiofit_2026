package services

import (
	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/models"
)

// PatanahiTracker implements G16 (BAY-3): pata-nahi cascade protocol.
// Tracks consecutive low-confidence (PATA_NAHI) answers and escalates through
// a graduated response protocol:
//
//	Count < 2:  normal operation
//	Count == 2: rephrase next question using alt_prompt (if available)
//	Count == 3: switch to binary-only mode (no CATEGORICAL questions)
//	Count >= 5: terminate session as PARTIAL_ASSESSMENT
//	Count >= 5 AND safety_flag fired: ESCALATE immediately
//
// The tracker is stateless — it reads from and writes to the session's
// ConsecutiveLowConf counter. The session handler is responsible for
// persisting state changes.
type PatanahiTracker struct {
	log *zap.Logger
}

// PatanahiAction describes what the session handler should do after a pata-nahi update.
type PatanahiAction struct {
	// UseAltPrompt: present next question using alt_prompt_en/alt_prompt_hi
	UseAltPrompt bool `json:"use_alt_prompt"`
	// BinaryOnly: restrict answers to YES/NO only (no CATEGORICAL options)
	BinaryOnly bool `json:"binary_only"`
	// Terminate: end session with PARTIAL_ASSESSMENT status
	Terminate bool `json:"terminate"`
	// Escalate: end session AND flag for immediate clinician review
	Escalate bool `json:"escalate"`
	// ConsecutiveCount: the current count after this update
	ConsecutiveCount int `json:"consecutive_count"`
}

// Cascade thresholds (from spec Section 11, G16 / BAY-3).
const (
	patanahiRephraseThreshold  = 2
	patanahiBinaryThreshold    = 3
	patanahiTerminateThreshold = 5
)

// NewPatanahiTracker creates a new PatanahiTracker.
func NewPatanahiTracker(log *zap.Logger) *PatanahiTracker {
	return &PatanahiTracker{log: log}
}

// RecordAnswer updates the consecutive low-confidence counter based on the
// latest answer and returns the action the session handler should take.
//
// If the answer is PATA_NAHI, the counter increments. Any non-PATA_NAHI answer
// resets the counter to 0.
//
// hasSafetyFlag should be true if any safety trigger has fired during this
// session — it controls whether termination escalates to ESCALATE vs
// PARTIAL_ASSESSMENT.
func (pt *PatanahiTracker) RecordAnswer(
	consecutiveCount int,
	answer string,
	hasSafetyFlag bool,
) PatanahiAction {
	if answer != string(models.AnswerPata) {
		// Non-pata-nahi answer resets the counter
		return PatanahiAction{
			ConsecutiveCount: 0,
		}
	}

	// Increment consecutive counter
	consecutiveCount++

	action := PatanahiAction{
		ConsecutiveCount: consecutiveCount,
	}

	// Apply cascade rules (cumulative — higher thresholds include lower actions)
	if consecutiveCount >= patanahiTerminateThreshold {
		action.Terminate = true
		action.BinaryOnly = true
		action.UseAltPrompt = true

		if hasSafetyFlag {
			action.Escalate = true
			pt.log.Warn("G16: pata-nahi cascade ESCALATE (safety flag + excessive unknowns)",
				zap.Int("consecutive_count", consecutiveCount),
			)
		} else {
			pt.log.Warn("G16: pata-nahi cascade TERMINATE (PARTIAL_ASSESSMENT)",
				zap.Int("consecutive_count", consecutiveCount),
			)
		}
	} else if consecutiveCount >= patanahiBinaryThreshold {
		action.BinaryOnly = true
		action.UseAltPrompt = true
		pt.log.Info("G16: pata-nahi cascade binary-only mode",
			zap.Int("consecutive_count", consecutiveCount),
		)
	} else if consecutiveCount >= patanahiRephraseThreshold {
		action.UseAltPrompt = true
		pt.log.Info("G16: pata-nahi cascade rephrase mode",
			zap.Int("consecutive_count", consecutiveCount),
		)
	}

	return action
}

// ShouldUseAltPrompt returns true if the current consecutive count warrants
// presenting the next question with its alt_prompt text. Convenience method
// for the question presentation layer.
func (pt *PatanahiTracker) ShouldUseAltPrompt(consecutiveCount int) bool {
	return consecutiveCount >= patanahiRephraseThreshold
}

// IsBinaryOnly returns true if the current consecutive count restricts
// answers to YES/NO only (no CATEGORICAL options).
func (pt *PatanahiTracker) IsBinaryOnly(consecutiveCount int) bool {
	return consecutiveCount >= patanahiBinaryThreshold
}

// ShouldTerminate returns true if the session should end with PARTIAL_ASSESSMENT.
func (pt *PatanahiTracker) ShouldTerminate(consecutiveCount int) bool {
	return consecutiveCount >= patanahiTerminateThreshold
}
