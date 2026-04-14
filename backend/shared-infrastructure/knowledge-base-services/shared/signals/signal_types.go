package signals

// SignalType identifies a clinical signal in the S1-S22 registry.
type SignalType string

const (
	// Glycaemic domain
	SignalFBG       SignalType = "FBG"        // S1
	SignalPPBG      SignalType = "PPBG"       // S2
	SignalHbA1c     SignalType = "HBA1C"      // S3
	SignalMealLog   SignalType = "MEAL_LOG"   // S4
	SignalGlucoseCV SignalType = "GLUCOSE_CV" // S5
	SignalHypoEvent SignalType = "HYPO_EVENT" // S6

	// Hemodynamic domain
	SignalSBP         SignalType = "SBP"         // S7
	SignalDBP         SignalType = "DBP"         // S8
	SignalHR          SignalType = "HR"          // S9
	SignalOrthostatic SignalType = "ORTHOSTATIC" // S10

	// Renal domain
	SignalCreatinine SignalType = "CREATININE" // S11
	SignalACR        SignalType = "ACR"        // S12
	SignalPotassium  SignalType = "POTASSIUM"  // S13

	// Metabolic domain
	SignalWeight     SignalType = "WEIGHT"      // S14
	SignalWaist      SignalType = "WAIST"       // S15
	SignalActivity   SignalType = "ACTIVITY"    // S16
	SignalLipidPanel SignalType = "LIPID_PANEL" // S17

	// Patient-Reported domain
	SignalSymptom         SignalType = "SYMPTOM"         // S18
	SignalAdverseEvent    SignalType = "ADVERSE_EVENT"    // S19
	SignalAdherence       SignalType = "ADHERENCE"        // S20
	SignalResolution      SignalType = "RESOLUTION"       // S21
	SignalHospitalisation SignalType = "HOSPITALISATION"  // S22

	// Phase 6 P6-6 — staging transition event (not a clinical signal in
	// the S1-S22 sense, but uses the same envelope shape so it can flow
	// through the existing priority signal Kafka pipeline).
	SignalCKMStageTransition SignalType = "CKM_STAGE_TRANSITION"

	// Phase 6 P6-2 — derived eGFR lab event for reactive renal dose
	// gating. Distinct from SignalCreatinine because eGFR is computed
	// from creatinine + age + sex, and dispatching on it directly lets
	// KB-23's renal gate run with the actual filtration value rather
	// than having to re-derive it.
	SignalEGFRLab SignalType = "EGFR_LAB"
)

// SignalSource identifies the origin of a signal.
type SignalSource string

const (
	SourceAppManual     SignalSource = "APP_MANUAL"
	SourceBLEDevice     SignalSource = "BLE_DEVICE"
	SourceFHIRSync      SignalSource = "FHIR_SYNC"
	SourceNLUExtraction SignalSource = "NLU_EXTRACTION"
)

// LOINCCodes maps signal types to their LOINC codes (empty for non-LOINC signals).
var LOINCCodes = map[SignalType]string{
	SignalFBG:        "1558-6",
	SignalPPBG:       "87422-2",
	SignalHbA1c:      "4548-4",
	SignalSBP:        "8480-6",
	SignalDBP:        "8462-4",
	SignalHR:         "8867-4",
	SignalCreatinine: "2160-0",
	SignalACR:        "9318-7",
	SignalPotassium:  "6298-4",
	SignalWeight:     "29463-7",
	SignalWaist:      "56086-2",
}
