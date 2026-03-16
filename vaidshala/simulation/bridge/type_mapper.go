// Package bridge provides bidirectional type mappers between the simulation
// type system (int-based GateSignal, non-pointer labs) and the production
// V-MCU type system (string-based GateSignal, *float64 labs).
//
// These mappers are the ONLY sanctioned crossing point between the two type
// universes. All simulation ↔ production data flow MUST pass through here.
package bridge

import (
	"math"
	"time"

	cb "vaidshala/clinical-runtime-platform/engines/vmcu/channel_b"
	cc "vaidshala/clinical-runtime-platform/engines/vmcu/channel_c"
	vt "vaidshala/clinical-runtime-platform/engines/vmcu/types"
	simtypes "vaidshala/simulation/pkg/types"
)

// ---------------------------------------------------------------------------
// GateSignal bidirectional maps
// ---------------------------------------------------------------------------

var gateSignalSimToProd = map[simtypes.GateSignal]vt.GateSignal{
	simtypes.CLEAR:     vt.GateClear,
	simtypes.MODIFY:    vt.GateModify,
	simtypes.PAUSE:     vt.GatePause,
	simtypes.HOLD_DATA: vt.GateHoldData,
	simtypes.HALT:      vt.GateHalt,
}

var gateSignalProdToSim = map[vt.GateSignal]simtypes.GateSignal{
	vt.GateClear:    simtypes.CLEAR,
	vt.GateModify:   simtypes.MODIFY,
	vt.GatePause:    simtypes.PAUSE,
	vt.GateHoldData: simtypes.HOLD_DATA,
	vt.GateHalt:     simtypes.HALT,
}

// GateSignalToProduction converts a simulation GateSignal (int) to production (string).
// Unknown values map to GateHalt (fail-safe).
func GateSignalToProduction(sim simtypes.GateSignal) vt.GateSignal {
	if prod, ok := gateSignalSimToProd[sim]; ok {
		return prod
	}
	return vt.GateHalt // fail-safe: unknown → most restrictive
}

// GateSignalToSimulation converts a production GateSignal (string) to simulation (int).
// Unknown values map to HALT (fail-safe).
func GateSignalToSimulation(prod vt.GateSignal) simtypes.GateSignal {
	if sim, ok := gateSignalProdToSim[prod]; ok {
		return sim
	}
	return simtypes.HALT // fail-safe: unknown → most restrictive
}

// ---------------------------------------------------------------------------
// Pointer helpers
// ---------------------------------------------------------------------------

func float64Ptr(v float64) *float64  { return &v }
func timePtr(v time.Time) *time.Time { return &v }

func derefFloat64(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

func intToFloat64Ptr(v int) *float64 {
	f := float64(v)
	return &f
}

// ---------------------------------------------------------------------------
// PatientTimestamps — bridge-local struct for lab measurement timestamps
// ---------------------------------------------------------------------------

// PatientTimestamps provides the last-measured timestamps for key labs.
// The simulation's RawPatientData stores these inline, but the production
// RawPatientData uses *time.Time pointers in separate fields. This struct
// bridges that gap — callers construct it from the simulation's inline
// timestamps before calling ToProductionRawLabs.
type PatientTimestamps struct {
	LastGlucose    time.Time
	LastCreatinine time.Time
	LastPotassium  time.Time
	LastHbA1c      time.Time
	LastEGFR       time.Time
}

// ---------------------------------------------------------------------------
// RawPatientData: simulation → production
// ---------------------------------------------------------------------------

// ToProductionRawLabs converts simulation RawPatientData + TitrationContext
// into the production channel_b.RawPatientData.
//
// Fields that exist in production but not in simulation (e.g., EGFRSlope,
// BPPattern, MeasurementUncertainty, ActivePerturbations) are left at their
// zero values (nil / "" / 0).
func ToProductionRawLabs(sim *simtypes.RawPatientData, ctx *simtypes.TitrationContext, ts PatientTimestamps) *cb.RawPatientData {
	prod := &cb.RawPatientData{
		// Current lab values
		GlucoseCurrent:    float64Ptr(sim.GlucoseCurrent),
		GlucoseTimestamp:  sim.GlucoseTimestamp,
		CreatinineCurrent: float64Ptr(sim.CreatinineCurrent),
		PotassiumCurrent:  float64Ptr(sim.PotassiumCurrent),
		SBPCurrent:        intToFloat64Ptr(sim.SBP),
		WeightKgCurrent:   float64Ptr(sim.Weight),
		EGFRCurrent:       float64Ptr(sim.EGFR),
		HbA1cCurrent:      float64Ptr(sim.HbA1c),

		// Historical values
		Creatinine48hAgo: float64Ptr(sim.CreatininePrevious),
		HbA1cPrior30d:    float64Ptr(sim.HbA1c), // sim has single HbA1c; use as prior too
		Weight72hAgo:     float64Ptr(sim.WeightPrevious),

		// Measurement timestamps
		EGFRLastMeasuredAt:       nilTimeIfZero(ts.LastEGFR),
		HbA1cLastMeasuredAt:      nilTimeIfZero(ts.LastHbA1c),
		CreatinineLastMeasuredAt: nilTimeIfZero(ts.LastCreatinine),

		// Context flags from sim.RawPatientData
		CreatinineRiseExplained: sim.CreatinineRiseExplained,
		RecentDoseIncrease:      sim.RecentDoseIncrease,
		BetaBlockerActive:       sim.BetaBlockerActive,

		// Heart rate (int → *float64)
		HeartRateCurrent: intToFloat64Ptr(sim.HeartRate),
		HRRegularity:     sim.HeartRateRegularity,

		// BP extensions
		SodiumCurrent: float64Ptr(sim.SodiumCurrent),
		DBPCurrent:    intToFloat64Ptr(sim.DBP),
	}

	// Fields sourced from TitrationContext (production RawPatientData has them;
	// simulation splits them across Labs + Context).
	if ctx != nil {
		prod.OnRAASAgent = ctx.ACEiActive || ctx.ARBActive
		prod.ThiazideActive = ctx.ThiazideActive
		prod.Season = ctx.Season
		prod.CKDStage = ctx.CKDStage
		prod.OliguriaReported = ctx.OliguriaReported
	}

	// Construct GlucoseReadings from current + previous glucose values.
	if sim.GlucoseCurrent != 0 {
		readings := []cb.TimestampedValue{
			{Value: sim.GlucoseCurrent, Timestamp: sim.GlucoseTimestamp},
		}
		if sim.GlucosePrevious != 0 {
			readings = append(readings, cb.TimestampedValue{
				Value:     sim.GlucosePrevious,
				Timestamp: sim.GlucoseTimestamp.Add(-2 * time.Hour), // synthetic offset
			})
		}
		prod.GlucoseReadings = readings
	}

	return prod
}

// nilTimeIfZero returns nil for the zero time, otherwise a pointer.
func nilTimeIfZero(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// ---------------------------------------------------------------------------
// RawPatientData: production → simulation
// ---------------------------------------------------------------------------

// ToSimulationRawLabs converts production channel_b.RawPatientData back
// to a simulation RawPatientData. Pointer values that are nil become 0.
// Float64 SBP/DBP/HeartRate are truncated to int.
func ToSimulationRawLabs(prod *cb.RawPatientData) simtypes.RawPatientData {
	sim := simtypes.RawPatientData{
		GlucoseCurrent:          derefFloat64(prod.GlucoseCurrent),
		GlucoseTimestamp:        prod.GlucoseTimestamp,
		CreatinineCurrent:       derefFloat64(prod.CreatinineCurrent),
		PotassiumCurrent:        derefFloat64(prod.PotassiumCurrent),
		SBP:                     floatToInt(derefFloat64(prod.SBPCurrent)),
		DBP:                     floatToInt(derefFloat64(prod.DBPCurrent)),
		HeartRate:               floatToInt(derefFloat64(prod.HeartRateCurrent)),
		HeartRateRegularity:     prod.HRRegularity,
		Weight:                  derefFloat64(prod.WeightKgCurrent),
		WeightPrevious:          derefFloat64(prod.Weight72hAgo),
		EGFR:                    derefFloat64(prod.EGFRCurrent),
		HbA1c:                   derefFloat64(prod.HbA1cCurrent),
		SodiumCurrent:           derefFloat64(prod.SodiumCurrent),
		BetaBlockerActive:       prod.BetaBlockerActive,
		CreatinineRiseExplained: prod.CreatinineRiseExplained,
		RecentDoseIncrease:      prod.RecentDoseIncrease,

		// Historical
		CreatininePrevious: derefFloat64(prod.Creatinine48hAgo),
	}

	// Timestamps from pointers
	if prod.EGFRLastMeasuredAt != nil {
		sim.EGFRTimestamp = *prod.EGFRLastMeasuredAt
	}
	if prod.HbA1cLastMeasuredAt != nil {
		sim.HbA1cTimestamp = *prod.HbA1cLastMeasuredAt
	}
	if prod.CreatinineLastMeasuredAt != nil {
		sim.CreatinineTimestamp = *prod.CreatinineLastMeasuredAt
	}

	// Glucose readings → previous value
	if len(prod.GlucoseReadings) > 1 {
		sim.GlucosePrevious = prod.GlucoseReadings[1].Value
	}

	return sim
}

// floatToInt truncates a float64 to int (same as int(f) in Go).
func floatToInt(f float64) int {
	return int(math.Trunc(f))
}

// ---------------------------------------------------------------------------
// TitrationContext: simulation → production
// ---------------------------------------------------------------------------

// ToProductionContext converts a simulation TitrationContext to the production
// channel_c.TitrationContext. ActiveMedications are mapped from
// []ActiveMedication to []string (drug class names only).
func ToProductionContext(sim *simtypes.TitrationContext) *cc.TitrationContext {
	// Map ActiveMedications → []string drug class names
	meds := make([]string, len(sim.ActiveMedications))
	for i, m := range sim.ActiveMedications {
		meds[i] = m.DrugClass
	}

	// Determine ProposedAction from dose delta sign
	var proposedAction string
	switch {
	case sim.ProposedDoseDelta > 0:
		proposedAction = "dose_increase"
	case sim.ProposedDoseDelta < 0:
		proposedAction = "dose_decrease"
	default:
		proposedAction = "dose_hold"
	}

	// Compute DoseDeltaPercent
	var doseDeltaPct float64
	if sim.CurrentDose > 0 {
		doseDeltaPct = math.Abs(sim.ProposedDoseDelta / sim.CurrentDose * 100)
	}

	return &cc.TitrationContext{
		EGFR:              sim.EGFRCurrent,
		ActiveMedications: meds,
		ProposedAction:    proposedAction,
		DoseDeltaPercent:  doseDeltaPct,

		// Boolean composites — in production these are pre-computed by the
		// orchestrator. In simulation we set them from the context flags that
		// exist on the simulation TitrationContext.
		HypoglycaemiaWithin7d: sim.HypoWithin7d,

		// Numeric values for PG threshold comparisons
		// (production gets these from cache; simulation has them in Labs,
		// but Channel C's context carries its own copies).
		PotassiumCurrent: 0, // Caller should set from Labs if needed
		SBPCurrent:       0, // Caller should set from Labs if needed
		SodiumCurrent:    0, // Caller should set from Labs if needed
	}
}
