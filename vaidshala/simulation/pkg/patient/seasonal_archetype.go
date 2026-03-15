package patient

import (
	"time"

	"vaidshala/simulation/pkg/types"
)

// ========================================================================
// SCENARIO 13: Seasonal Hyponatraemia — Borderline Na+ in Summer
// Expected: B-19 fires PAUSE (Na+ 134 < 135 seasonal threshold, thiazide active).
// B-17 does NOT fire (Na+ 134 > 132 severe threshold).
// Tests: B-19 production-only seasonal Na+ rule.
// ========================================================================
func SeasonalHyponatraemia() VirtualPatient {
	return VirtualPatient{
		ID:          "SIM-SEASONAL-001",
		Archetype:   "seasonal_hyponatraemia",
		Description: "Na+=134 on thiazide in summer. Below 135 seasonal threshold (B-19) but above 132 severe (B-17). Tests production-only seasonal rule.",
		Labs: types.RawPatientData{
			PatientID:         "SIM-SEASONAL-001",
			Timestamp:         time.Now(),
			GlucoseCurrent:    7.8,
			GlucoseTimestamp:   hoursAgo(3),
			SodiumCurrent:     134, // Below 135 seasonal threshold, above 132 severe
			SodiumTimestamp:    daysAgo(0),
			PotassiumCurrent:  4.2,
			PotassiumTimestamp: daysAgo(1),
			CreatinineCurrent: 95,
			CreatinineTimestamp: daysAgo(2),
			EGFR:               65,
			EGFRTimestamp:       daysAgo(2),
			SBP:                130,
			DBP:                82,
		},
		Context: types.TitrationContext{
			CurrentDose:    12.0,
			EGFRCurrent:   65,
			ThiazideActive: true,
			Season:         "SUMMER",
			CKDStage:       "2",
		},
		MCUGate:   types.CLEAR,
		Adherence: 0.85,
		LoopTrust: 0.80,
	}
}
