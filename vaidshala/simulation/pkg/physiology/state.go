package physiology

// PhysiologyState represents the current physiological state of a virtual patient.
// All units are SI: glucose mmol/L, SBP mmHg, eGFR mL/min/1.73m², weight kg, etc.
type PhysiologyState struct {
	// Glucose metabolism
	GlucoseMmol       float64 // Current fasting blood glucose
	HbA1cPct          float64 // Glycated haemoglobin percentage
	BetaCellPct       float64 // Beta-cell function as % of normal (100 = healthy)
	InsulinResistance float64 // Insulin resistance index (1.0 = normal)

	// Hemodynamics
	SBPMmHg      float64 // Systolic blood pressure
	DBPMmHg      float64 // Diastolic blood pressure
	HeartRateBPM float64 // Heart rate

	// Renal function
	EGFRMlMin      float64 // Estimated glomerular filtration rate
	CreatinineUmol float64 // Serum creatinine
	PotassiumMmol  float64 // Serum potassium

	// Body composition
	WeightKg       float64 // Body weight
	VisceralFatIdx float64 // Visceral fat index (arbitrary units, 1.0 = normal)

	// Electrolytes
	SodiumMmol float64 // Serum sodium

	// Metadata
	DayNumber  int // Current simulation day
	CycleInDay int // Which cycle within the day (0-based)
}

// DefaultState returns a healthy baseline physiological state.
func DefaultState() PhysiologyState {
	return PhysiologyState{
		GlucoseMmol:       5.5,
		HbA1cPct:          5.4,
		BetaCellPct:       100,
		InsulinResistance: 1.0,
		SBPMmHg:           120,
		DBPMmHg:           75,
		HeartRateBPM:      72,
		EGFRMlMin:         90,
		CreatinineUmol:    80,
		PotassiumMmol:     4.2,
		WeightKg:          80,
		VisceralFatIdx:    1.0,
		SodiumMmol:        140,
	}
}
