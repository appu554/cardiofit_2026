package physiology

// Trajectory archetypes for 90-day physiology simulation.
// Each archetype configures a PhysiologyState and medication profile
// representing a distinct clinical phenotype per spec Section 5.

// TrajectoryArchetype bundles initial state with a medication profile.
type TrajectoryArchetype struct {
	Name  string
	State PhysiologyState
	Meds  TrajectoryMedications
}

// TrajectoryMedications describes the medication profile for a trajectory run.
type TrajectoryMedications struct {
	ACEi        bool
	SGLT2i      bool
	GLP1RA      bool
	Metformin   bool
	BetaBlocker bool
	Thiazide    bool
}

// VisceralObesePatient: high visceral fat, insulin-resistant, hypertensive T2DM.
// Expected: FBG declines (158→141+), HbA1c improves, SBP declines.
func VisceralObesePatient() TrajectoryArchetype {
	s := DefaultState()
	s.GlucoseMmol = 8.8  // ~158 mg/dL
	s.HbA1cPct = 8.2
	s.BetaCellPct = 55
	s.InsulinResistance = 2.0
	s.SBPMmHg = 152
	s.DBPMmHg = 92
	s.WeightKg = 105
	s.VisceralFatIdx = 1.8
	s.EGFRMlMin = 72
	s.CreatinineUmol = 100
	s.PotassiumMmol = 4.3
	return TrajectoryArchetype{
		Name:  "VisceralObesePatient",
		State: s,
		Meds: TrajectoryMedications{
			ACEi:     true,
			SGLT2i:   true,
			Metformin: true,
		},
	}
}

// CKDProgressorPatient: CKD stage 3b with progressive decline.
// Expected: eGFR decline rate ≤0.7 mL/min/year (vs 1.3 untreated).
func CKDProgressorPatient() TrajectoryArchetype {
	s := DefaultState()
	s.GlucoseMmol = 7.8
	s.HbA1cPct = 7.4
	s.BetaCellPct = 50
	s.InsulinResistance = 1.6
	s.SBPMmHg = 148
	s.DBPMmHg = 88
	s.WeightKg = 88
	s.EGFRMlMin = 38
	s.CreatinineUmol = 190
	s.PotassiumMmol = 4.8
	return TrajectoryArchetype{
		Name:  "CKDProgressorPatient",
		State: s,
		Meds: TrajectoryMedications{
			ACEi:   true,
			SGLT2i: true,
			GLP1RA: true,
		},
	}
}

// ElderlyFrailPatient: conservative targets, avoid hypoglycaemia.
// Expected: FBG stays 125-140 mg/dL. Zero HALT from hypoglycaemia.
func ElderlyFrailPatient() TrajectoryArchetype {
	s := DefaultState()
	s.GlucoseMmol = 7.5 // ~135 mg/dL
	s.HbA1cPct = 7.0
	s.BetaCellPct = 40
	s.InsulinResistance = 1.3
	s.SBPMmHg = 138
	s.DBPMmHg = 78
	s.HeartRateBPM = 68
	s.WeightKg = 62
	s.EGFRMlMin = 45
	s.CreatinineUmol = 160
	s.PotassiumMmol = 4.5
	return TrajectoryArchetype{
		Name:  "ElderlyFrailPatient",
		State: s,
		Meds: TrajectoryMedications{
			ACEi:     true,
			Metformin: true,
		},
	}
}

// GoodResponderPatient: well-controlled, heading toward deprescribing.
// Expected: FBG drops significantly, HbA1c→6.5, deprescribing trajectory.
func GoodResponderPatient() TrajectoryArchetype {
	s := DefaultState()
	s.GlucoseMmol = 6.8
	s.HbA1cPct = 6.8
	s.BetaCellPct = 75
	s.InsulinResistance = 1.1
	s.SBPMmHg = 128
	s.DBPMmHg = 76
	s.WeightKg = 78
	s.EGFRMlMin = 85
	s.CreatinineUmol = 85
	s.PotassiumMmol = 4.1
	return TrajectoryArchetype{
		Name:  "GoodResponderPatient",
		State: s,
		Meds: TrajectoryMedications{
			ACEi:     true,
			SGLT2i:   true,
			Metformin: true,
		},
	}
}

// AllTrajectoryArchetypes returns all 4 named trajectory archetypes.
func AllTrajectoryArchetypes() []TrajectoryArchetype {
	return []TrajectoryArchetype{
		VisceralObesePatient(),
		CKDProgressorPatient(),
		ElderlyFrailPatient(),
		GoodResponderPatient(),
	}
}
