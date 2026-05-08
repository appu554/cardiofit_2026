package algorithmic_distinction

// Marker returns the display emoji for an observation class.
// These emojis appear in the pharmacist self-visibility UI so the pharmacist
// can immediately distinguish the epistemic provenance of a surface element
// (Self-Visibility Guidelines §6.2).
//
//   🔵  substrate_fact          — objective, computed from EvidenceTrace
//   🟡  platform_suggestion     — algorithmic inference, not yet confirmed
//   🟢  pharmacist_reflection   — pharmacist's own authored entry
//   🟣  hybrid                  — platform suggestion confirmed by pharmacist
//
// Returns "" for unrecognised classes (no panic).
func Marker(c Class) string {
	switch c {
	case ClassSubstrateFact:
		return "🔵"
	case ClassPlatformSuggestion:
		return "🟡"
	case ClassPharmacistReflection:
		return "🟢"
	case ClassHybrid:
		return "🟣"
	}
	return ""
}
