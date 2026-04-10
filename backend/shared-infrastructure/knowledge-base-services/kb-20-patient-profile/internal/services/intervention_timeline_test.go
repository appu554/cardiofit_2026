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

func TestMapDrugClassToAllDomains(t *testing.T) {
	tests := []struct {
		drugClass   string
		wantDomains []string
	}{
		// SGLT2i: glycaemic (primary) + renal + hemodynamic (secondary)
		{"SGLT2i", []string{"GLYCAEMIC", "RENAL", "HEMODYNAMIC"}},
		// GLP1_RA: glycaemic + hemodynamic
		{"GLP1_RA", []string{"GLYCAEMIC", "HEMODYNAMIC"}},
		// ACEi: hemodynamic + renal
		{"ACEi", []string{"HEMODYNAMIC", "RENAL"}},
		// ARB: hemodynamic + renal
		{"ARB", []string{"HEMODYNAMIC", "RENAL"}},
		// FINERENONE: renal + hemodynamic
		{"FINERENONE", []string{"RENAL", "HEMODYNAMIC"}},
		// Plain glycaemic drug — no secondary domains
		{"METFORMIN", []string{"GLYCAEMIC"}},
		// Plain hemodynamic drug — no secondary
		{"AMLODIPINE", []string{"HEMODYNAMIC"}},
		// Unknown
		{"ASPIRIN", nil},
	}

	for _, tt := range tests {
		t.Run(tt.drugClass, func(t *testing.T) {
			got := MapDrugClassToAllDomains(tt.drugClass)
			if tt.wantDomains == nil {
				if got != nil {
					t.Errorf("MapDrugClassToAllDomains(%q) = %v, want nil", tt.drugClass, got)
				}
				return
			}
			if len(got) != len(tt.wantDomains) {
				t.Fatalf("MapDrugClassToAllDomains(%q) returned %d domains, want %d: got %v",
					tt.drugClass, len(got), len(tt.wantDomains), got)
			}
			for i, want := range tt.wantDomains {
				if got[i] != want {
					t.Errorf("MapDrugClassToAllDomains(%q)[%d] = %q, want %q", tt.drugClass, i, got[i], want)
				}
			}
		})
	}
}
