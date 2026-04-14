package services

import (
	"strings"
	"time"
)

// drugClassSteadyState maps an antihypertensive drug class to its
// pharmacologic time-to-steady-state — the duration after a dose change
// during which the BP response is still titrating and the stability
// engine's dwell window should yield to the new clinical reality.
//
// Sources: ESH 2023, ISH 2020 product monographs, FDA labels. These are
// conservative estimates from 5-7 half-lives; actual clinical response
// may be slower when titrating dose rather than starting de novo.
//
// Phase 5 P5-5 — replaces the flat 7-day constant from P5-2 so the
// override window matches each drug's pharmacokinetics.
var drugClassSteadyState = map[string]time.Duration{
	// Calcium channel blockers (dihydropyridines)
	"AMLODIPINE": 8 * 24 * time.Hour, // t½ ~30-50h → steady ~7-10 days
	"FELODIPINE": 7 * 24 * time.Hour,
	"NIFEDIPINE": 3 * 24 * time.Hour,

	// ARBs
	"LOSARTAN":    6 * 24 * time.Hour,
	"VALSARTAN":   4 * 24 * time.Hour,
	"TELMISARTAN": 7 * 24 * time.Hour, // long t½ ~24h

	// ACE inhibitors
	"LISINOPRIL": 5 * 24 * time.Hour,
	"RAMIPRIL":   5 * 24 * time.Hour,
	"ENALAPRIL":  4 * 24 * time.Hour,

	// Beta blockers
	"METOPROLOL": 2 * 24 * time.Hour, // t½ ~3-7h → fast steady
	"ATENOLOL":   2 * 24 * time.Hour,
	"BISOPROLOL": 3 * 24 * time.Hour,

	// Diuretics
	"HCTZ":       7 * 24 * time.Hour, // delayed BP response
	"INDAPAMIDE": 7 * 24 * time.Hour,
}

// defaultSteadyStateWindow is the fallback used when the drug class is
// unknown, empty, or not in drugClassSteadyState. 7 days matches the
// Phase 5 P5-2 flat constant — the safe middle ground that covers most
// antihypertensives without being so wide that long-tail effects fire.
const defaultSteadyStateWindow = 7 * 24 * time.Hour

// SteadyStateWindow returns the pharmacologic time-to-steady-state for
// a drug class. Lookup is case-insensitive and whitespace-tolerant.
// Unknown or empty classes return defaultSteadyStateWindow.
func SteadyStateWindow(drugClass string) time.Duration {
	key := strings.ToUpper(strings.TrimSpace(drugClass))
	if key == "" {
		return defaultSteadyStateWindow
	}
	if d, ok := drugClassSteadyState[key]; ok {
		return d
	}
	return defaultSteadyStateWindow
}
