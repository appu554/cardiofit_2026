package services

import (
	"math"
	"sync"
	"time"

	"kb-26-metabolic-digital-twin/internal/models"
)

// PAIEventTrigger manages rate limiting and significant change detection.
// The in-memory cache is the fast path; the optional repo is the fallback
// after a service restart (prevents burst recomputation of all patients).
type PAIEventTrigger struct {
	minInterval      time.Duration
	significantDelta float64
	repo             *PAIRepository // optional — for post-restart fallback
	mu               sync.Mutex
	lastComputed     map[string]time.Time // patientID → last compute time
}

// NewPAIEventTrigger creates a trigger with the given minimum recompute interval
// (in minutes) and the score delta threshold for publishing change events.
// The repo parameter is optional — when provided, ShouldRecompute falls back
// to checking the DB when the in-memory cache has no entry (post-restart).
func NewPAIEventTrigger(minIntervalMinutes int, significantDelta float64, repo ...*PAIRepository) *PAIEventTrigger {
	var r *PAIRepository
	if len(repo) > 0 {
		r = repo[0]
	}
	return &PAIEventTrigger{
		minInterval:      time.Duration(minIntervalMinutes) * time.Minute,
		significantDelta: significantDelta,
		repo:             r,
		lastComputed:     make(map[string]time.Time),
	}
}

// ShouldRecompute returns true if enough time has passed since the last compute.
// Checks in-memory cache first (hot path). On cache miss (post-restart), falls
// back to the DB if a repository was provided — prevents burst recomputation
// of all patients after service restart.
func (t *PAIEventTrigger) ShouldRecompute(patientID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Hot path: in-memory cache
	if last, ok := t.lastComputed[patientID]; ok {
		return time.Since(last) >= t.minInterval
	}

	// Cold path (post-restart): check DB for last computed time
	if t.repo != nil {
		if latest, err := t.repo.FetchLatest(patientID); err == nil && latest != nil {
			t.lastComputed[patientID] = latest.ComputedAt // warm the cache
			return time.Since(latest.ComputedAt) >= t.minInterval
		}
	}

	return true // no record anywhere → allow computation
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
