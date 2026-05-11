package failed_interventions

import "testing"

func TestClassifyInterventionType(t *testing.T) {
	t.Parallel()
	cases := []struct {
		ruleID         string
		wantType       string
		wantClassified bool
	}{
		// STOP_PSYCH_* → antipsychotic_deprescribing
		{"STOP_PSYCH_RISPERIDONE_BPSD", "antipsychotic_deprescribing", true},
		{"STOP_PSYCH_OLANZAPINE_LONGTERM", "antipsychotic_deprescribing", true},
		// STOP_BENZO_* → benzodiazepine_deprescribing
		{"STOP_BENZO_LORAZEPAM_PRN", "benzodiazepine_deprescribing", true},
		{"STOP_BENZO_DIAZEPAM_FALLS", "benzodiazepine_deprescribing", true},
		// STOP_ANTICH_* → anticholinergic_deprescribing
		{"STOP_ANTICH_OXYBUTYNIN_COGNITION", "anticholinergic_deprescribing", true},
		{"STOP_ANTICH_AMITRIPTYLINE_FRAILTY", "anticholinergic_deprescribing", true},
		// DOSE_REDUCE_* → dose_reduction
		{"DOSE_REDUCE_DIGOXIN_AGE", "dose_reduction", true},
		{"DOSE_REDUCE_GABAPENTIN_EGFR", "dose_reduction", true},
		// MONITOR_* → not classified (no veto record)
		{"MONITOR_LITHIUM_LEVEL", "", false},
		{"MONITOR_INR_WARFARIN", "", false},
		// ADD_* → not classified
		{"ADD_VITAMIN_D_SUPPLEMENT", "", false},
		{"ADD_BISPHOSPHONATE_FRACTURE_RISK", "", false},
		// Unknown prefixes
		{"UNKNOWN_RULE_XYZ", "", false},
		{"", "", false},
		// Case-insensitivity on the prefix
		{"stop_psych_haloperidol", "antipsychotic_deprescribing", true},
		{"Stop_Benzo_Clonazepam", "benzodiazepine_deprescribing", true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.ruleID, func(t *testing.T) {
			t.Parallel()
			gotType, gotOK := ClassifyInterventionType(tc.ruleID)
			if gotType != tc.wantType || gotOK != tc.wantClassified {
				t.Errorf("ClassifyInterventionType(%q) = (%q, %v); want (%q, %v)",
					tc.ruleID, gotType, gotOK, tc.wantType, tc.wantClassified)
			}
		})
	}
}
