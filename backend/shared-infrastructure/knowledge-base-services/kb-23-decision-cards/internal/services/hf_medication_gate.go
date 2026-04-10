package services

import "fmt"

// HFContraindication defines a drug class that is contraindicated in HF.
type HFContraindication struct {
	DrugClass   string   // e.g., "PIOGLITAZONE"
	HFTypes     []string // empty = ALL HF types; ["HFrEF"] = only HFrEF
	Reason      string
	SourceTrial string
}

// HFMedicationGate checks drug contraindications specific to heart failure.
type HFMedicationGate struct {
	contraindications []HFContraindication
}

func NewHFMedicationGate() *HFMedicationGate {
	return &HFMedicationGate{
		contraindications: []HFContraindication{
			{
				DrugClass:   "PIOGLITAZONE",
				HFTypes:     nil, // ALL HF types
				Reason:      "Fluid retention exacerbates heart failure",
				SourceTrial: "FDA_BLACK_BOX",
			},
			{
				DrugClass:   "SAXAGLIPTIN",
				HFTypes:     nil, // ALL HF types — conservative
				Reason:      "Increased HF hospitalization risk",
				SourceTrial: "SAVOR-TIMI_53",
			},
			{
				DrugClass:   "ALOGLIPTIN",
				HFTypes:     nil,
				Reason:      "Potential HF risk signal",
				SourceTrial: "EXAMINE",
			},
			{
				DrugClass:   "NON_DHP_CCB",
				HFTypes:     []string{"HFrEF"}, // only HFrEF
				Reason:      "Negative inotropic effect",
				SourceTrial: "AHA_ACC_HFSA_2022",
			},
		},
	}
}

// CheckContraindication returns (blocked, reason) if the drug is contraindicated
// for the patient's CKM substage and HF type. Only blocks for Stage 4c.
func (g *HFMedicationGate) CheckContraindication(drugClass, ckmStage, hfType string) (bool, string) {
	if ckmStage != "4c" {
		return false, ""
	}

	for _, ci := range g.contraindications {
		if ci.DrugClass != drugClass {
			continue
		}

		// If HFTypes is empty, applies to ALL HF types
		if len(ci.HFTypes) == 0 {
			return true, fmt.Sprintf("CONTRAINDICATED in heart failure: %s (%s)", ci.Reason, ci.SourceTrial)
		}

		// Check if patient's HF type is in the contraindicated list
		for _, ht := range ci.HFTypes {
			if ht == hfType {
				return true, fmt.Sprintf("CONTRAINDICATED in %s: %s (%s)", hfType, ci.Reason, ci.SourceTrial)
			}
		}
	}

	return false, ""
}
