package services

// MHRISubstageAdjustment modifies MHRI domain weights and score interpretation
// based on CKM substage. Default weights: glucose 35%, cardio 25%, body_comp 25%, behavioral 15%.
type MHRISubstageAdjustment struct {
	GlucoseDomainWeight    float64 `json:"glucose_domain_weight"`
	CardioDomainWeight     float64 `json:"cardio_domain_weight"`
	BodyCompDomainWeight   float64 `json:"body_comp_domain_weight"`
	BehavioralDomainWeight float64 `json:"behavioral_domain_weight"`
	ScoreCeiling           float64 `json:"score_ceiling"`
	InterpretationNote     string  `json:"interpretation_note"`
}

// ComputeCKMSubstageAdjustment returns adjusted MHRI weights for a CKM substage.
// Uses string parameters to avoid cross-module import from KB-20.
func ComputeCKMSubstageAdjustment(ckmStage, hfType, nyhaClass string) MHRISubstageAdjustment {
	adj := MHRISubstageAdjustment{
		GlucoseDomainWeight:    0.35,
		CardioDomainWeight:     0.25,
		BodyCompDomainWeight:   0.25,
		BehavioralDomainWeight: 0.15,
		ScoreCeiling:           100.0,
	}

	switch ckmStage {
	case "4a":
		adj.CardioDomainWeight = 0.30
		adj.GlucoseDomainWeight = 0.30
		adj.BodyCompDomainWeight = 0.25
		adj.BehavioralDomainWeight = 0.15
		adj.InterpretationNote = "Stage 4a: cardio domain weighted for subclinical CVD monitoring"

	case "4b":
		adj.CardioDomainWeight = 0.35
		adj.GlucoseDomainWeight = 0.30
		adj.BodyCompDomainWeight = 0.20
		adj.BehavioralDomainWeight = 0.15
		adj.ScoreCeiling = 85.0
		adj.InterpretationNote = "Stage 4b: secondary prevention; ceiling 85 due to residual ASCVD risk"

	case "4c":
		switch hfType {
		case "HFrEF":
			adj.CardioDomainWeight = 0.40
			adj.GlucoseDomainWeight = 0.25
			adj.BodyCompDomainWeight = 0.20
			adj.BehavioralDomainWeight = 0.15
			adj.InterpretationNote = "Stage 4c HFrEF: cardio dominant — track EF, NT-proBNP"
		case "HFmrEF":
			adj.CardioDomainWeight = 0.35
			adj.GlucoseDomainWeight = 0.25
			adj.BodyCompDomainWeight = 0.20
			adj.BehavioralDomainWeight = 0.20
			adj.InterpretationNote = "Stage 4c HFmrEF: balanced cardio + behavioral"
		case "HFpEF":
			adj.CardioDomainWeight = 0.25
			adj.GlucoseDomainWeight = 0.25
			adj.BodyCompDomainWeight = 0.25
			adj.BehavioralDomainWeight = 0.25
			adj.InterpretationNote = "Stage 4c HFpEF: equal weighting — obesity, exercise, comorbidity"
		default:
			adj.ScoreCeiling = 70.0
			adj.InterpretationNote = "Stage 4c: HF subtype unknown — conservative ceiling"
			return adj
		}

		// NYHA class ceiling
		switch nyhaClass {
		case "IV":
			adj.ScoreCeiling = 30.0
		case "III":
			adj.ScoreCeiling = 50.0
		case "II":
			adj.ScoreCeiling = 70.0
		case "I":
			adj.ScoreCeiling = 85.0
		}
	}

	return adj
}
