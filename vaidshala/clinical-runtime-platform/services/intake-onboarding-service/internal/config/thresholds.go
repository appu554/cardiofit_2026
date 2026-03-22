package config

import "time"

// Thresholds holds configurable clinical and operational thresholds
// for the intake-onboarding service. These can be overridden via
// environment variables in production.
type Thresholds struct {
	// Session management
	SessionPauseTimeout   time.Duration // Auto-pause after inactivity (default 4h)
	ReminderIntervals     []time.Duration // Reminder schedule after pause (24h, 48h, 72h)
	AbandonTimeout        time.Duration // Auto-abandon after pause (default 7 days)

	// Safety engine
	SafetyMaxLatency      time.Duration // Alert if safety evaluation exceeds this (default 5ms)

	// NLU extraction
	NLUConfidenceThreshold float64 // Below this, re-ask or escalate (default 0.70)
	NLUMaxLatency          time.Duration // Max acceptable NLU response time (default 500ms)

	// Review queue
	ReviewQueueMaxAge     time.Duration // Alert if review pending longer than this (default 24h)

	// Check-in
	CheckinIntervalDays   int // Biweekly check-in interval (default 14)
	CheckinSlotCount      int // Number of slots in check-in subset (default 12)

	// Dedup
	MessageDedupTTL       time.Duration // Redis dedup TTL (default 24h)
}

// DefaultThresholds returns production defaults from the spec.
func DefaultThresholds() Thresholds {
	return Thresholds{
		SessionPauseTimeout: 4 * time.Hour,
		ReminderIntervals: []time.Duration{
			24 * time.Hour,
			48 * time.Hour,
			72 * time.Hour,
		},
		AbandonTimeout:         7 * 24 * time.Hour,
		SafetyMaxLatency:       5 * time.Millisecond,
		NLUConfidenceThreshold: 0.70,
		NLUMaxLatency:          500 * time.Millisecond,
		ReviewQueueMaxAge:      24 * time.Hour,
		CheckinIntervalDays:    14,
		CheckinSlotCount:       12,
		MessageDedupTTL:        24 * time.Hour,
	}
}
