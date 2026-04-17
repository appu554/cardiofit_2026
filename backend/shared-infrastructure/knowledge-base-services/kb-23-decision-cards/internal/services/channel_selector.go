package services

import (
	"encoding/json"
	"fmt"
	"time"

	"kb-23-decision-cards/internal/models"
)

// ChannelSelection is the output of SelectChannels — describes which channels
// to use, fallback strategy, and quiet-hours suppression state.
type ChannelSelection struct {
	PrimaryChannels  []string
	FallbackChannels []string
	FallbackAfter    time.Duration
	Suppressed       bool
	SuppressedUntil  *time.Time
	Simultaneous     bool
}

// SelectChannels determines notification channels based on escalation tier,
// clinician preferences, and quiet-hours rules.
func SelectChannels(
	tier models.EscalationTier,
	clinicianPrefs *models.ClinicianPreferences,
	now time.Time,
) ChannelSelection {
	sel := tierDefaults(tier)

	// Rule 2 — clinician preference override (SAFETY exempted).
	if tier != models.TierSafety && clinicianPrefs != nil && clinicianPrefs.PreferredChannels != "" {
		if preferred := parsePreferredChannels(clinicianPrefs.PreferredChannels); len(preferred) > 0 {
			sel.PrimaryChannels = preferred
		}
	}

	// Rule 3 — quiet hours.
	applyQuietHours(&sel, tier, clinicianPrefs, now)

	return sel
}

// tierDefaults returns the base ChannelSelection for a given tier.
func tierDefaults(tier models.EscalationTier) ChannelSelection {
	switch tier {
	case models.TierSafety:
		return ChannelSelection{
			PrimaryChannels: []string{"push", "sms", "whatsapp"},
			Simultaneous:    true,
		}
	case models.TierImmediate:
		return ChannelSelection{
			PrimaryChannels:  []string{"push"},
			FallbackChannels: []string{"sms"},
			FallbackAfter:    2 * time.Hour,
		}
	case models.TierUrgent:
		return ChannelSelection{
			PrimaryChannels:  []string{"push"},
			FallbackChannels: []string{"sms"},
			FallbackAfter:    24 * time.Hour,
		}
	case models.TierRoutine:
		return ChannelSelection{
			PrimaryChannels: []string{"in_app"},
		}
	case models.TierInformational:
		return ChannelSelection{
			Suppressed: true,
		}
	default:
		return ChannelSelection{
			PrimaryChannels: []string{"in_app"},
		}
	}
}

// applyQuietHours checks whether `now` falls inside the clinician's quiet
// window and adjusts the selection accordingly.
func applyQuietHours(
	sel *ChannelSelection,
	tier models.EscalationTier,
	prefs *models.ClinicianPreferences,
	now time.Time,
) {
	startStr := "22:00"
	endStr := "06:00"
	if prefs != nil {
		if prefs.QuietHoursStart != "" {
			startStr = prefs.QuietHoursStart
		}
		if prefs.QuietHoursEnd != "" {
			endStr = prefs.QuietHoursEnd
		}
	}

	if !inQuietWindow(now, startStr, endStr) {
		return
	}

	switch tier {
	case models.TierSafety:
		// BYPASS — deliver immediately regardless.
		return
	case models.TierImmediate:
		// QUEUE — suppress until quiet hours end.
		sel.Suppressed = true
		endTime := quietEnd(now, endStr)
		sel.SuppressedUntil = &endTime
	default:
		// URGENT / ROUTINE — suppress.
		sel.Suppressed = true
	}
}

// inQuietWindow returns true if `now` is inside the quiet window defined by
// HH:MM start/end strings. Handles overnight windows (e.g. 22:00 → 06:00).
func inQuietWindow(now time.Time, startStr, endStr string) bool {
	startH, startM := parseHHMM(startStr)
	endH, endM := parseHHMM(endStr)

	nowMinutes := now.Hour()*60 + now.Minute()
	startMinutes := startH*60 + startM
	endMinutes := endH*60 + endM

	if startMinutes <= endMinutes {
		// Same-day window (e.g. 08:00 → 17:00).
		return nowMinutes >= startMinutes && nowMinutes < endMinutes
	}
	// Overnight window (e.g. 22:00 → 06:00).
	return nowMinutes >= startMinutes || nowMinutes < endMinutes
}

// quietEnd computes the next occurrence of the quiet-hours end time relative
// to `now`, returning it as an absolute time.Time.
func quietEnd(now time.Time, endStr string) time.Time {
	endH, endM := parseHHMM(endStr)
	candidate := time.Date(now.Year(), now.Month(), now.Day(), endH, endM, 0, 0, now.Location())
	if !candidate.After(now) {
		candidate = candidate.Add(24 * time.Hour)
	}
	return candidate
}

// parseHHMM extracts hour and minute from an "HH:MM" string.
func parseHHMM(s string) (int, int) {
	var h, m int
	fmt.Sscanf(s, "%d:%d", &h, &m)
	return h, m
}

// parsePreferredChannels unmarshals a JSON array string like `["whatsapp","sms"]`.
func parsePreferredChannels(raw string) []string {
	var channels []string
	if err := json.Unmarshal([]byte(raw), &channels); err != nil {
		return nil
	}
	return channels
}
