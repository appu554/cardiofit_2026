package services

import "testing"

func TestMapDrugClassToDomain(t *testing.T) {
	tests := []struct {
		drugClass string
		want      string
	}{
		// Glycaemic domain (9 classes)
		{"METFORMIN", "GLYCAEMIC"},
		{"SULFONYLUREA", "GLYCAEMIC"},
		{"DPP4i", "GLYCAEMIC"},
		{"SGLT2i", "GLYCAEMIC"},
		{"GLP1_RA", "GLYCAEMIC"},
		{"INSULIN", "GLYCAEMIC"},
		{"BASAL_INSULIN", "GLYCAEMIC"},
		{"PIOGLITAZONE", "GLYCAEMIC"},
		{"EXENATIDE", "GLYCAEMIC"},

		// Hemodynamic domain
		{"ACEi", "HEMODYNAMIC"},
		{"ARB", "HEMODYNAMIC"},
		{"BETA_BLOCKER", "HEMODYNAMIC"},

		// Lipid domain
		{"STATIN", "LIPID"},
		{"EZETIMIBE", "LIPID"},

		// Renal domain
		{"FINERENONE", "RENAL"},

		// Unknown → OTHER
		{"ASPIRIN", "OTHER"},
		{"", "OTHER"},
	}

	for _, tt := range tests {
		t.Run(tt.drugClass, func(t *testing.T) {
			got := MapDrugClassToDomain(tt.drugClass)
			if got != tt.want {
				t.Errorf("MapDrugClassToDomain(%q) = %q, want %q", tt.drugClass, got, tt.want)
			}
		})
	}
}
