package incident_response

import "testing"

// TestClassifier_AssignsCorrectSeverity verifies the canonical kind→severity mapping.
func TestClassifier_AssignsCorrectSeverity(t *testing.T) {
	cases := []struct {
		kind string
		want int
	}{
		{"clinical_safety", 1},
		{"trust_violation", 2},
		{"bias_concern", 3},
		{"procedural", 4},
	}
	for _, tc := range cases {
		got := Classify(tc.kind)
		if got != tc.want {
			t.Errorf("Classify(%q) = %d, want %d", tc.kind, got, tc.want)
		}
	}
}

// TestClassifier_UnknownKindDefaultsToSeverity4 verifies that an unrecognised
// kind is treated conservatively as severity 4 (procedural), per Guidelines §11.1.
func TestClassifier_UnknownKindDefaultsToSeverity4(t *testing.T) {
	cases := []string{"", "unknown", "CLINICAL_SAFETY", "bogus_incident_type"}
	for _, kind := range cases {
		got := Classify(kind)
		if got != 4 {
			t.Errorf("Classify(%q) = %d, want 4 (conservative default)", kind, got)
		}
	}
}

// TestIsValidIncidentKind verifies the package-level helper accepts exactly the
// four canonical kinds and rejects anything else.
func TestIsValidIncidentKind(t *testing.T) {
	valid := []string{"clinical_safety", "trust_violation", "bias_concern", "procedural"}
	for _, k := range valid {
		if !IsValidIncidentKind(k) {
			t.Errorf("IsValidIncidentKind(%q) = false, want true", k)
		}
	}
	invalid := []string{"", "CLINICAL_SAFETY", "unknown", "trust violation"}
	for _, k := range invalid {
		if IsValidIncidentKind(k) {
			t.Errorf("IsValidIncidentKind(%q) = true, want false", k)
		}
	}
}
