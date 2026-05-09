// Package incident_response implements incident classification and hold
// orchestration for the ERM corrective layer per Ethical Architecture
// Guidelines §11. The package provides:
//
//   - [Classify]: maps a named incident kind to a severity integer (1..4)
//   - [IsValidIncidentKind]: validates a kind string against the canonical set
//   - [Incident]: the data structure passed to hold handlers
//   - [HoldHandler] / [HoldHandlerFunc]: interface and function adapter for
//     corrective hold logic
//   - [Orchestrator]: routes incidents to registered handlers when severity ≤ 2
//
// # VisibilityClass: AD
package incident_response

// Classify returns the integer severity (1..4) for a named incident kind.
//
// Mapping (per Ethical Architecture Guidelines §11.1):
//
//	"clinical_safety" → 1  (most severe — immediate hold required)
//	"trust_violation" → 2
//	"bias_concern"    → 3
//	"procedural"      → 4  (least severe)
//
// Any unrecognised kind defaults conservatively to 4 (procedural), ensuring
// that novel incident kinds are never silently elevated beyond safe bounds.
func Classify(kind string) int {
	switch kind {
	case "clinical_safety":
		return 1
	case "trust_violation":
		return 2
	case "bias_concern":
		return 3
	case "procedural":
		return 4
	default:
		return 4
	}
}

// IsValidIncidentKind returns true when s is one of the four canonical incident
// kind strings that Classify recognises.
func IsValidIncidentKind(s string) bool {
	switch s {
	case "clinical_safety", "trust_violation", "bias_concern", "procedural":
		return true
	default:
		return false
	}
}
