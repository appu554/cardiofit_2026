package services

// MandatoryMedGap represents a medication that should be present but isn't.
type MandatoryMedGap struct {
	MissingClass string `json:"missing_class"`
	Rationale    string `json:"rationale"`
	Urgency      string `json:"urgency"`
	Alternative  string `json:"alternative,omitempty"`
	SourceTrial  string `json:"source_trial"`
}

type MandatoryMedChecker struct{}

func NewMandatoryMedChecker() *MandatoryMedChecker {
	return &MandatoryMedChecker{}
}

func (c *MandatoryMedChecker) CheckMandatory(
	ckmStage string,
	hfType string,
	activeMedClasses []string,
) []MandatoryMedGap {
	activeSet := make(map[string]bool)
	for _, mc := range activeMedClasses {
		activeSet[mc] = true
	}

	// Normalize: ACEi or ARB counts as RAS blockade
	hasRAS := activeSet["ACEi"] || activeSet["ARB"] || activeSet["ARNI"] || activeSet["SACUBITRIL_VALSARTAN"]
	hasAntiplatelet := activeSet["ASPIRIN"] || activeSet["CLOPIDOGREL"] || activeSet["TICAGRELOR"] || activeSet["PRASUGREL"]
	hasBB := activeSet["BETA_BLOCKER"] || activeSet["BETA_BLOCKER_HF"] || activeSet["CARVEDILOL"] || activeSet["BISOPROLOL"]

	var gaps []MandatoryMedGap

	switch ckmStage {
	case "4a":
		if !activeSet["STATIN"] {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "STATIN",
				Rationale:    "All Stage 4a require high-intensity statin for subclinical CVD",
				Urgency:      "URGENT",
				SourceTrial:  "AHA_ACC_2018",
			})
		}

	case "4b":
		if !activeSet["STATIN"] {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "STATIN",
				Rationale:    "Secondary prevention requires high-intensity statin",
				Urgency:      "URGENT",
				SourceTrial:  "AHA_ACC_2018",
			})
		}
		if !hasAntiplatelet {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "ANTIPLATELET",
				Rationale:    "Post-ASCVD requires antiplatelet therapy",
				Urgency:      "URGENT",
				SourceTrial:  "AHA_ACC_2018",
			})
		}
		if !hasRAS {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "ACEi_OR_ARB",
				Rationale:    "Post-MI cardioprotection + renoprotection",
				Urgency:      "URGENT",
				SourceTrial:  "HOPE, EUROPA",
			})
		}

	case "4c":
		gaps = append(gaps, c.checkHFMandatory(hfType, activeSet, hasRAS, hasBB)...)
	}

	return gaps
}

func (c *MandatoryMedChecker) checkHFMandatory(
	hfType string,
	activeSet map[string]bool,
	hasRAS bool,
	hasBB bool,
) []MandatoryMedGap {
	var gaps []MandatoryMedGap

	switch hfType {
	case "HFrEF":
		// Four pillars: ARNI/ACEi/ARB, BB, MRA, SGLT2i
		if !hasRAS {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "ARNI_OR_ACEi_ARB",
				Rationale:    "PARADIGM-HF: ARNI preferred; ACEi/ARB if not tolerated",
				Urgency:      "IMMEDIATE",
				SourceTrial:  "PARADIGM-HF",
			})
		}
		if !hasBB {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "BETA_BLOCKER_HF",
				Rationale:    "HFrEF mortality reduction — carvedilol/bisoprolol/metoprolol succinate",
				Urgency:      "IMMEDIATE",
				SourceTrial:  "CIBIS-II, MERIT-HF, COPERNICUS",
			})
		}
		if !activeSet["MRA"] && !activeSet["SPIRONOLACTONE"] && !activeSet["EPLERENONE"] {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "MRA",
				Rationale:    "HFrEF mortality + hospitalization reduction",
				Urgency:      "IMMEDIATE",
				SourceTrial:  "RALES, EMPHASIS-HF",
			})
		}
		if !activeSet["SGLT2i"] {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "SGLT2i",
				Rationale:    "HFrEF mortality + hospitalization reduction",
				Urgency:      "IMMEDIATE",
				SourceTrial:  "DAPA-HF, EMPEROR-Reduced",
			})
		}

	case "HFmrEF":
		if !activeSet["SGLT2i"] {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "SGLT2i",
				Rationale:    "Benefit across EF spectrum (DELIVER subgroup)",
				Urgency:      "URGENT",
				SourceTrial:  "DELIVER, EMPEROR-Preserved",
			})
		}
		if !hasRAS {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "ACEi_OR_ARB",
				Rationale:    "Reasonable from HFrEF extrapolation",
				Urgency:      "URGENT",
				SourceTrial:  "AHA_ACC_HFSA_2022",
			})
		}

	case "HFpEF":
		// ONLY mandatory: SGLT2i
		if !activeSet["SGLT2i"] {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "SGLT2i",
				Rationale:    "ONLY proven disease-modifying therapy for HFpEF (EMPEROR-Preserved, DELIVER)",
				Urgency:      "URGENT",
				SourceTrial:  "EMPEROR-Preserved, DELIVER",
			})
		}

	default:
		// Unknown HF type — at minimum, SGLT2i
		if !activeSet["SGLT2i"] {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "SGLT2i",
				Rationale:    "SGLT2i beneficial across HF spectrum",
				Urgency:      "URGENT",
				SourceTrial:  "DAPA-HF, EMPEROR-Preserved",
			})
		}
	}

	return gaps
}
