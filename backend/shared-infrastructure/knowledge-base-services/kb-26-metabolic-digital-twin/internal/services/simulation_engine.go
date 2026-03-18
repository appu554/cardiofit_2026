package services

import (
	"encoding/json"
	"fmt"

	"kb-26-metabolic-digital-twin/internal/models"
	"gorm.io/datatypes"
)

// RunSimulation executes coupled forward simulation for N days.
// Returns projected states at 30-day intervals.
func RunSimulation(initial models.SimState, intervention models.Intervention, days int) []models.ProjectedState {
	var results []models.ProjectedState

	state := initial
	bio := ComputeBiomarkers(state)
	results = append(results, models.ProjectedState{Day: 0, State: state,
		FBG: bio.FBG, PPBG: bio.PPBG, SBP: bio.SBP, WaistCm: bio.WaistCm, EGFR: bio.EGFR, HbA1c: bio.HbA1c})

	for day := 1; day <= days; day++ {
		state = CouplingStep(state, intervention)
		if day%30 == 0 || day == days {
			bio = ComputeBiomarkers(state)
			results = append(results, models.ProjectedState{Day: day, State: state,
				FBG: bio.FBG, PPBG: bio.PPBG, SBP: bio.SBP, WaistCm: bio.WaistCm, EGFR: bio.EGFR, HbA1c: bio.HbA1c})
		}
	}
	return results
}

// TwinToSimState maps TwinState (Tier 1-3) to SimState (6 coupled variables).
func TwinToSimState(twin *models.TwinState) models.SimState {
	s := models.SimState{IS: 0.5, VF: 0.5, HGO: 0.5, MM: 0.5, VR: 0.5, RR: 0.8}

	if twin.VisceralFatProxy != nil {
		s.VF = *twin.VisceralFatProxy
	}

	if ev, err := unmarshalEstimated(twin.InsulinSensitivity); err == nil {
		s.IS = ev.Value
	}
	if ev, err := unmarshalEstimated(twin.HepaticGlucoseOutput); err == nil {
		s.HGO = ev.Value
	}
	if ev, err := unmarshalEstimated(twin.MuscleMassProxy); err == nil {
		s.MM = ev.Value
	}

	if twin.MAPValue != nil {
		s.VR = clamp((*twin.MAPValue-70)/50.0, 0, 1)
	}

	if twin.EGFR != nil {
		s.RR = clamp((*twin.EGFR-15)/105.0, 0, 1)
	}

	return s
}

func unmarshalEstimated(data datatypes.JSON) (models.EstimatedVariable, error) {
	var ev models.EstimatedVariable
	if len(data) == 0 {
		return ev, fmt.Errorf("empty")
	}
	err := json.Unmarshal(data, &ev)
	return ev, err
}
