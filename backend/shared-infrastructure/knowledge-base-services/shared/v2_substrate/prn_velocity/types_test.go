package prn_velocity

import "testing"

func TestIsValidPRNClass(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  bool
	}{
		{"benzodiazepine", "benzodiazepine", true},
		{"antipsychotic", "antipsychotic", true},
		{"analgesic", "analgesic", true},
		{"out_of_scope_antiemetic", "antiemetic", false},
		{"out_of_scope_laxative", "laxative", false},
		{"out_of_scope_sedative", "sedative", false},
		{"empty_string", "", false},
		{"case_mismatch", "Benzodiazepine", false},
		{"unrelated", "metformin", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsValidPRNClass(tc.input)
			if got != tc.want {
				t.Errorf("IsValidPRNClass(%q) = %v; want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestPRNClassConstants(t *testing.T) {
	// Guard against accidental rename — these strings appear in the canonical
	// CQL file (cql/prn_escalation_velocity.cql) and in CAPE Guidelines
	// lines 569–571 (PRN_benzodiazepine_escalation_velocity, etc).
	if PRNBenzodiazepine != "benzodiazepine" {
		t.Errorf("PRNBenzodiazepine drift: got %q", PRNBenzodiazepine)
	}
	if PRNAntipsychotic != "antipsychotic" {
		t.Errorf("PRNAntipsychotic drift: got %q", PRNAntipsychotic)
	}
	if PRNAnalgesic != "analgesic" {
		t.Errorf("PRNAnalgesic drift: got %q", PRNAnalgesic)
	}
}
