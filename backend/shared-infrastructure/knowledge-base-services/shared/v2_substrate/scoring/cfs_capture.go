package scoring

import (
	"github.com/cardiofit/shared/v2_substrate/models"
	"github.com/cardiofit/shared/v2_substrate/validation"
)

// CFSInstrumentVersionCurrent is the canonical current revision string the
// substrate uses when callers do not supply an explicit version. Pinning
// to the Rockwood 2020 revision lets historic scores be reinterpreted
// against the version that produced them.
const CFSInstrumentVersionCurrent = "v2.0"

// CFSScoreLabel returns the human-readable Rockwood label for a 1-9 CFS
// score. Returns the empty string for out-of-range inputs (callers should
// validate first; this helper is for display, not gatekeeping).
func CFSScoreLabel(score int) string {
	switch score {
	case 1:
		return "Very fit"
	case 2:
		return "Well"
	case 3:
		return "Managing well"
	case 4:
		return "Living with very mild frailty"
	case 5:
		return "Living with mild frailty"
	case 6:
		return "Living with moderate frailty"
	case 7:
		return "Living with severe frailty"
	case 8:
		return "Living with very severe frailty"
	case 9:
		return "Terminally ill"
	}
	return ""
}

// ValidateCFSCapture is a convenience wrapper that delegates to
// validation.ValidateCFSScore and exists so callers in the storage layer
// can import a single scoring package rather than mixing scoring +
// validation. Returns the validation error verbatim.
func ValidateCFSCapture(c models.CFSScore) error {
	return validation.ValidateCFSScore(c)
}
