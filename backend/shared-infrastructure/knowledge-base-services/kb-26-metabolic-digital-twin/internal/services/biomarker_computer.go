package services

import "kb-26-metabolic-digital-twin/internal/models"

// BiomarkerOutput maps SimState to clinical observables.
type BiomarkerOutput struct {
	FBG     float64
	PPBG    float64
	SBP     float64
	WaistCm float64
	EGFR    float64
	HbA1c   float64
}

// ComputeBiomarkers translates SimState (0-1 normalized) to clinical units.
func ComputeBiomarkers(s models.SimState) BiomarkerOutput {
	fbg := 100 + (1-s.IS)*80 + s.HGO*40 - s.MM*10
	fbg = clamp(fbg, 60, 400)

	ppbg := 120 + (1-s.IS)*100 + s.VF*30 - s.MM*15
	ppbg = clamp(ppbg, 70, 500)

	sbp := 120 + s.VR*50 + s.VF*20 - s.IS*10
	sbp = clamp(sbp, 80, 220)

	waist := 80 + s.VF*40 - s.MM*5
	waist = clamp(waist, 55, 150)

	egfr := 30 + s.RR*90
	egfr = clamp(egfr, 10, 150)

	hba1c := (fbg + 46.7) / 28.7
	hba1c = clamp(hba1c, 4.0, 16.0)

	return BiomarkerOutput{FBG: fbg, PPBG: ppbg, SBP: sbp, WaistCm: waist, EGFR: egfr, HbA1c: hba1c}
}
