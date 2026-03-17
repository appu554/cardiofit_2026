// patient_factory.go generates synthetic patient states for simulation scenarios.
package simulation

import (
	"time"

	"vaidshala/clinical-runtime-platform/engines/vmcu/channel_b"
	"vaidshala/clinical-runtime-platform/engines/vmcu/titration"
)

// PatientState tracks evolving patient data across the 90-day simulation.
type PatientState struct {
	ID            string
	InitialDose   float64
	ProposedDelta float64
	MedClass      titration.MedicationClass

	// Current lab values (evolve over time)
	Glucose    float64 // mmol/L
	Creatinine float64 // umol/L
	Potassium  float64 // mEq/L
	SBP        float64 // mmHg
	Weight     float64 // kg
	EGFR       float64 // mL/min/1.73m2
	HbA1c      float64 // %

	// Historical values (auto-tracked by simulation)
	PriorCreatinine48h *float64
	PriorEGFR48h       *float64
	PriorWeight72h     *float64
	PriorHbA1c30d      *float64

	// Glucose trend
	GlucoseReadings []channel_b.TimestampedValue

	// Flags
	DataAvailable bool // false = simulate missing data
}

// f64 is a helper to create *float64 from a value.
func f64(v float64) *float64 { return &v }

// ToRawLabs converts patient state to Channel B input.
// When DataAvailable is false, returns nil pointers (simulating missing data).
func (p *PatientState) ToRawLabs(simTime time.Time) *channel_b.RawPatientData {
	if !p.DataAvailable {
		return &channel_b.RawPatientData{
			GlucoseTimestamp: simTime.Add(-72 * time.Hour), // stale
		}
	}
	return &channel_b.RawPatientData{
		GlucoseCurrent:    f64(p.Glucose),
		GlucoseTimestamp:  simTime.Add(-30 * time.Minute), // recent
		CreatinineCurrent: f64(p.Creatinine),
		PotassiumCurrent:  f64(p.Potassium),
		SBPCurrent:        f64(p.SBP),
		WeightKgCurrent:   f64(p.Weight),
		EGFRCurrent:       f64(p.EGFR),
		HbA1cCurrent:      f64(p.HbA1c),
		Creatinine48hAgo:  p.PriorCreatinine48h,
		EGFRPrior48h:      p.PriorEGFR48h,
		Weight72hAgo:      p.PriorWeight72h,
		HbA1cPrior30d:     p.PriorHbA1c30d,
		GlucoseReadings:   p.GlucoseReadings,
	}
}

// SnapshotHistory records current values as prior values.
// Call this when "time passes" in the simulation.
func (p *PatientState) SnapshotHistory() {
	p.PriorCreatinine48h = f64(p.Creatinine)
	p.PriorEGFR48h = f64(p.EGFR)
	p.PriorWeight72h = f64(p.Weight)
	p.PriorHbA1c30d = f64(p.HbA1c)
}

// NewStableDiabetic creates a well-controlled T2DM patient on metformin.
func NewStableDiabetic() *PatientState {
	p := &PatientState{
		ID:            "SIM-STABLE-001",
		InitialDose:   1000, // mg metformin
		ProposedDelta: 50,
		MedClass:      titration.MedClassOralAgent,
		Glucose:       7.5,
		Creatinine:    85,
		Potassium:     4.2,
		SBP:           130,
		Weight:        82,
		EGFR:          75,
		HbA1c:         7.8,
		DataAvailable: true,
	}
	p.SnapshotHistory()
	return p
}

// NewAKIRiskPatient creates a patient with borderline renal function.
func NewAKIRiskPatient() *PatientState {
	p := &PatientState{
		ID:            "SIM-AKI-002",
		InitialDose:   1500,
		ProposedDelta: 50,
		MedClass:      titration.MedClassOralAgent,
		Glucose:       8.0,
		Creatinine:    110,
		Potassium:     4.5,
		SBP:           135,
		Weight:        78,
		EGFR:          45,
		HbA1c:         8.2,
		DataAvailable: true,
	}
	p.SnapshotHistory()
	return p
}

// NewHypoPronePatient creates a patient on basal insulin prone to hypoglycaemia.
func NewHypoPronePatient() *PatientState {
	p := &PatientState{
		ID:            "SIM-HYPO-003",
		InitialDose:   30, // units basal insulin
		ProposedDelta: 2,
		MedClass:      titration.MedClassBasalInsulin,
		Glucose:       5.5,
		Creatinine:    75,
		Potassium:     4.0,
		SBP:           125,
		Weight:        75,
		EGFR:          85,
		HbA1c:         7.0,
		DataAvailable: true,
	}
	p.SnapshotHistory()
	return p
}

// NewMissingDataPatient creates a patient who will have data gaps.
func NewMissingDataPatient() *PatientState {
	p := &PatientState{
		ID:            "SIM-MISSING-004",
		InitialDose:   1000,
		ProposedDelta: 50,
		MedClass:      titration.MedClassOralAgent,
		Glucose:       7.0,
		Creatinine:    90,
		Potassium:     4.3,
		SBP:           128,
		Weight:        80,
		EGFR:          70,
		HbA1c:         7.5,
		DataAvailable: true,
	}
	p.SnapshotHistory()
	return p
}

// NewDeprescribingPatient creates a patient for controlled dose reduction.
func NewDeprescribingPatient() *PatientState {
	p := &PatientState{
		ID:            "SIM-DEPRESCRIBE-006",
		InitialDose:   2000, // mg metformin at max
		ProposedDelta: 0,    // no increases during deprescribing
		MedClass:      titration.MedClassOralAgent,
		Glucose:       5.8,
		Creatinine:    80,
		Potassium:     4.1,
		SBP:           122,
		Weight:        76,
		EGFR:          80,
		HbA1c:         6.5, // well-controlled → candidate for deprescribing
		DataAvailable: true,
	}
	p.SnapshotHistory()
	return p
}

// NewChannelDisagreementPatient creates a patient who will trigger mixed signals.
func NewChannelDisagreementPatient() *PatientState {
	p := &PatientState{
		ID:            "SIM-DISAGREE-007",
		InitialDose:   20,
		ProposedDelta: 2,
		MedClass:      titration.MedClassBasalInsulin,
		Glucose:       9.0,
		Creatinine:    95,
		Potassium:     4.4,
		SBP:           140,
		Weight:        85,
		EGFR:          55,
		HbA1c:         8.5,
		DataAvailable: true,
	}
	p.SnapshotHistory()
	return p
}

// NewAutonomyLimitPatient creates a patient for autonomy limit testing.
func NewAutonomyLimitPatient() *PatientState {
	p := &PatientState{
		ID:            "SIM-AUTONOMY-008",
		InitialDose:   500,
		ProposedDelta: 100, // aggressive delta to hit limits
		MedClass:      titration.MedClassOralAgent,
		Glucose:       10.0,
		Creatinine:    75,
		Potassium:     4.2,
		SBP:           132,
		Weight:        88,
		EGFR:          80,
		HbA1c:         9.0,
		DataAvailable: true,
	}
	p.SnapshotHistory()
	return p
}
