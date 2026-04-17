package services

import (
	"math"
	"sync"
	"time"

	"kb-26-metabolic-digital-twin/internal/models"
)

// PAIEventTrigger manages rate limiting and significant change detection.
type PAIEventTrigger struct {
	minInterval      time.Duration
	significantDelta float64
	mu               sync.Mutex
	lastComputed     map[string]time.Time // patientID → last compute time
}

// NewPAIEventTrigger creates a trigger with the given minimum recompute interval
// (in minutes) and the score delta threshold for publishing change events.
func NewPAIEventTrigger(minIntervalMinutes int, significantDelta float64) *PAIEventTrigger {
	return &PAIEventTrigger{
		minInterval:      time.Duration(minIntervalMinutes) * time.Minute,
		significantDelta: significantDelta,
		lastComputed:     make(map[string]time.Time),
	}
}

// ShouldRecompute returns true if enough time has passed since the last compute.
func (t *PAIEventTrigger) ShouldRecompute(patientID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	last, ok := t.lastComputed[patientID]
	if !ok {
		return true
	}
	return time.Since(last) >= t.minInterval
}

// MarkComputed records that a computation just happened.
func (t *PAIEventTrigger) MarkComputed(patientID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lastComputed[patientID] = time.Now()
}

// ProcessResult determines if a PAI change is significant enough to publish.
// Returns a PAIChangeEvent if significant, nil otherwise.
func (t *PAIEventTrigger) ProcessResult(current, previous models.PAIScore) *models.PAIChangeEvent {
	scoreDelta := math.Abs(current.Score - previous.Score)
	tierChanged := current.Tier != previous.Tier

	if scoreDelta < t.significantDelta && !tierChanged {
		return nil // not significant
	}

	return &models.PAIChangeEvent{
		PatientID:       current.PatientID,
		NewScore:        current.Score,
		PreviousScore:   previous.Score,
		NewTier:         current.Tier,
		PreviousTier:    previous.Tier,
		DominantReason:  current.PrimaryReason,
		SuggestedAction: current.SuggestedAction,
		Timeframe:       current.SuggestedTimeframe,
	}
}
