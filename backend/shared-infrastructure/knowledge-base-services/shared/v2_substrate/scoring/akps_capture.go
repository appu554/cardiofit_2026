package scoring

import (
	"github.com/cardiofit/shared/v2_substrate/models"
	"github.com/cardiofit/shared/v2_substrate/validation"
)

// AKPSInstrumentVersionCurrent is the canonical current revision string
// the substrate uses when callers do not supply an explicit version.
// Pinning to the Abernethy 2005 publication lets historic scores be
// reinterpreted against the version that produced them.
const AKPSInstrumentVersionCurrent = "abernethy_2005"

// AKPSScoreLabel returns the human-readable AKPS label for a 0-100 score
// (multiples of 10). Returns the empty string for out-of-range or
// non-multiple-of-10 inputs.
func AKPSScoreLabel(score int) string {
	switch score {
	case 100:
		return "Normal; no complaints"
	case 90:
		return "Able to carry on normal activity"
	case 80:
		return "Normal activity with effort"
	case 70:
		return "Cares for self; unable normal activity"
	case 60:
		return "Requires occasional assistance"
	case 50:
		return "Requires considerable assistance"
	case 40:
		return "In bed >50% of time"
	case 30:
		return "Almost completely bedfast"
	case 20:
		return "Totally bedfast; requires nursing care"
	case 10:
		return "Comatose or barely rousable"
	case 0:
		return "Dead"
	}
	return ""
}

// ValidateAKPSCapture is a convenience wrapper that delegates to
// validation.ValidateAKPSScore.
func ValidateAKPSCapture(a models.AKPSScore) error {
	return validation.ValidateAKPSScore(a)
}
