// Package types defines the core types for the Vaidshala V-MCU simulation harness.
// These types MUST match the real V-MCU types in engines/vmcu/types/.
// Any divergence between these types and production types is a simulation validity bug.
package types

import "time"

// ---------------------------------------------------------------------------
// Gate Signal Hierarchy: CLEAR < MODIFY < PAUSE < HOLD_DATA < HALT
// This enum is the single most important type in the safety architecture.
// ---------------------------------------------------------------------------

type GateSignal int

const (
	CLEAR     GateSignal = 0
	MODIFY    GateSignal = 1
	PAUSE     GateSignal = 2
	HOLD_DATA GateSignal = 3
	HALT      GateSignal = 4
)

func (g GateSignal) String() string {
	switch g {
	case CLEAR:
		return "CLEAR"
	case MODIFY:
		return "MODIFY"
	case PAUSE:
		return "PAUSE"
	case HOLD_DATA:
		return "HOLD_DATA"
	case HALT:
		return "HALT"
	default:
		return "UNKNOWN"
	}
}

// MostRestrictive returns the highest-severity gate signal.
// This is the arbiter's core logic: 1oo3 veto, most restrictive wins.
func MostRestrictive(a, b GateSignal) GateSignal {
	if a > b {
		return a
	}
	return b
}

// ---------------------------------------------------------------------------
// Channel identification for SafetyTrace attribution
// ---------------------------------------------------------------------------

type Channel string

const (
	ChannelA Channel = "MCU_GATE"
	ChannelB Channel = "PHYSIO_GATE"
	ChannelC Channel = "PROTOCOL_GATE"
)

// ---------------------------------------------------------------------------
// Arbiter types
// ---------------------------------------------------------------------------

type ArbiterInput struct {
	MCUGate      GateSignal
	PhysioGate   GateSignal
	ProtocolGate GateSignal
}

type ArbiterOutput struct {
	FinalGate       GateSignal
	DominantChannel Channel
	AllChannels     map[Channel]GateSignal
	RationaleCode   string
}

// Arbitrate implements the 1oo3 veto logic.
// Deterministic, synchronous, zero external dependencies.
// This function MUST match vmcu/arbiter/arbiter.go exactly.
func Arbitrate(input ArbiterInput) ArbiterOutput {
	final := MostRestrictive(input.MCUGate, MostRestrictive(input.PhysioGate, input.ProtocolGate))

	// Determine dominant channel (which channel produced the most restrictive signal)
	var dominant Channel
	switch final {
	case input.PhysioGate:
		dominant = ChannelB
	case input.ProtocolGate:
		dominant = ChannelC
	default:
		dominant = ChannelA
	}
	// If multiple channels match, priority is B > C > A
	if input.PhysioGate == final {
		dominant = ChannelB
	} else if input.ProtocolGate == final {
		dominant = ChannelC
	}

	return ArbiterOutput{
		FinalGate:       final,
		DominantChannel: dominant,
		AllChannels: map[Channel]GateSignal{
			ChannelA: input.MCUGate,
			ChannelB: input.PhysioGate,
			ChannelC: input.ProtocolGate,
		},
	}
}

// ---------------------------------------------------------------------------
// Patient data types (matching V-MCU's RawPatientData)
// ---------------------------------------------------------------------------

type RawPatientData struct {
	PatientID string
	Timestamp time.Time

	// Glycaemic
	GlucoseCurrent    float64 // mmol/L
	GlucosePrevious   float64 // mmol/L (for trend detection)
	GlucoseTimestamp   time.Time
	HbA1c             float64 // percentage
	HbA1cTimestamp     time.Time

	// Renal
	CreatinineCurrent  float64 // µmol/L
	CreatininePrevious float64 // µmol/L (48h prior)

	// Perturbation suppression (Track 3 — mapped from ChannelBProjection)
	PerturbationSuppressed bool
	SuppressionMode        string
	PerturbationGainFactor float64

	CreatinineTimestamp time.Time
	EGFR               float64 // mL/min/1.73m²
	EGFRTimestamp       time.Time
	PotassiumCurrent   float64 // mmol/L
	PotassiumTimestamp  time.Time

	// Haemodynamic
	SBP               int     // mmHg
	DBP               int     // mmHg
	BPTimestamp        time.Time
	HeartRate          int     // bpm
	HeartRateRegularity string // REGULAR, IRREGULAR, UNKNOWN
	Weight             float64 // kg
	WeightPrevious     float64 // kg (72h prior)
	WeightTimestamp     time.Time

	// Electrolytes
	SodiumCurrent     float64 // mmol/L
	SodiumTimestamp    time.Time

	// Context flags (set by orchestrator before Channel B evaluation)
	BetaBlockerActive         bool
	CreatinineRiseExplained   bool // PG-14 RAAS tolerance flag
	RecentDoseIncrease        bool
	RecentDoseIncreaseTimestamp time.Time
}

// TitrationContext matches V-MCU's TitrationContext for Channel C evaluation.
type TitrationContext struct {
	ActiveMedications []ActiveMedication
	CurrentDose       float64
	ProposedDoseDelta float64
	DoseChangeCount   int     // consecutive cycles with dose changes
	CyclesSinceHbA1c  int     // cycles since last HbA1c improvement
	EGFRCurrent       float64
	ThiazideActive    bool
	ACEiActive        bool
	ARBActive         bool
	SGLT2iActive      bool
	InsulinActive     bool
	SulfonylureaActive bool
	DualRAASActive    bool // ACEi AND ARB simultaneously
	Season            string // SUMMER, MONSOON, WINTER, AUTUMN

	// RAAS tolerance context
	RAASChangeWithin14Days bool
	RAASChangeDate         time.Time
	PreRAASCreatinine      float64

	// Bridge-required fields (needed by production bridge adapter)
	CKDStage           string // "3a", "3b", "4", "5" — for B-12 J-curve stratification
	OliguriaReported   bool   // overrides RAAS tolerance (B-03)
	HypoWithin7d       bool   // any hypoglycaemia event in last 7 days (PG-07)
}

type ActiveMedication struct {
	DrugClass string
	RxNorm    string
	Dose      float64
	Unit      string
	StartDate time.Time
}

// ---------------------------------------------------------------------------
// Titration cycle input/output
// ---------------------------------------------------------------------------

type TitrationCycleInput struct {
	PatientID        string
	CycleNumber      int
	RawLabs          *RawPatientData
	TitrationContext *TitrationContext
	MCUGate          GateSignal // Channel A from KB-23
	AdherenceScore   float64    // from KB-21, per-class
	LoopTrustScore   float64    // from KB-21
}

type TitrationCycleResult struct {
	FinalGate       GateSignal
	DominantChannel Channel
	DoseApplied     bool
	DoseDelta       float64
	BlockedBy       string
	SafetyTrace     SafetyTrace
	PhysioRuleFired string
	ProtocolRuleFired string
}

// ---------------------------------------------------------------------------
// SafetyTrace — immutable audit record per titration cycle
// ---------------------------------------------------------------------------

type SafetyTrace struct {
	TraceID        string
	PatientID      string
	CycleTimestamp time.Time
	CycleNumber    int

	// Channel A
	MCUGate          GateSignal
	MCUGateRationale string

	// Channel B
	PhysioGate      GateSignal
	PhysioRuleFired string
	PhysioRawValues map[string]interface{}

	// Channel C
	ProtocolGate      GateSignal
	ProtocolRuleFired string

	// Arbiter
	FinalGate       GateSignal
	DominantChannel Channel

	// Outcome
	DoseApplied   bool
	DoseDelta     float64
	BlockedBy     string
	GainFactor    float64
	AdherenceSource string
}

// ---------------------------------------------------------------------------
// Integrator state (tracks dose momentum across cycles)
// ---------------------------------------------------------------------------

type IntegratorState struct {
	Frozen           bool
	FrozenSince      time.Time
	CurrentDose      float64
	LastApprovedDose float64
	PauseDurationH   float64
	ReentryPhase     int // 0=normal, 1=monitoring, 2=conservative, 3=normal
	ReentryCycles    int
	PostResumeCount  int
	PostResumeLimit  int
}
