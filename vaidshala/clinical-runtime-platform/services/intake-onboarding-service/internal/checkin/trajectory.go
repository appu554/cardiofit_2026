package checkin

import (
	"math"
	"time"
)

// Trajectory represents the patient's biweekly trajectory signal.
type Trajectory string

const (
	STABLE    Trajectory = "STABLE"
	FRAGILE   Trajectory = "FRAGILE"
	FAILURE   Trajectory = "FAILURE"
	DISENGAGE Trajectory = "DISENGAGE"
)

// TrajectoryInput provides all data needed to compute a trajectory.
type TrajectoryInput struct {
	CurrentValues      map[string]float64
	PreviousValues     map[string]float64
	BaselineValues     map[string]float64
	CycleNumber        int
	SlotsFilled        int
	SlotsRequired      int
	DaysSinceSchedule  int
	PreviousTrajectory Trajectory
}

// TrajectoryResult contains the computed trajectory and supporting detail.
type TrajectoryResult struct {
	Signal          Trajectory
	Confidence      float64
	ImprovingSlots  []string
	WorseningSlots  []string
	StableSlots     []string
	MissedSlots     []string
	ComputedAt      time.Time
	DomainScores    map[string]float64
}

// ClinicalTarget defines optimal and critical ranges for a check-in slot.
type ClinicalTarget struct {
	SlotName       string
	Domain         string
	LowOptimal     float64
	HighOptimal    float64
	LowCritical    float64
	HighCritical   float64
	ImprovementDir string // "lower", "higher", "range"
}

// DefaultClinicalTargets returns clinical targets for all 12 check-in slots.
func DefaultClinicalTargets() []ClinicalTarget {
	return []ClinicalTarget{
		{SlotName: "fbg", Domain: "glycemic", LowOptimal: 70, HighOptimal: 100, LowCritical: 54, HighCritical: 180, ImprovementDir: "lower"},
		{SlotName: "ppbg", Domain: "glycemic", LowOptimal: 70, HighOptimal: 140, LowCritical: 54, HighCritical: 250, ImprovementDir: "lower"},
		{SlotName: "hba1c", Domain: "glycemic", LowOptimal: 4.0, HighOptimal: 7.0, LowCritical: 3.0, HighCritical: 12.0, ImprovementDir: "lower"},
		{SlotName: "systolic_bp", Domain: "cardiovascular", LowOptimal: 100, HighOptimal: 130, LowCritical: 90, HighCritical: 180, ImprovementDir: "lower"},
		{SlotName: "diastolic_bp", Domain: "cardiovascular", LowOptimal: 60, HighOptimal: 80, LowCritical: 50, HighCritical: 120, ImprovementDir: "lower"},
		{SlotName: "egfr", Domain: "renal", LowOptimal: 60, HighOptimal: 120, LowCritical: 15, HighCritical: 200, ImprovementDir: "higher"},
		{SlotName: "weight", Domain: "anthropometric", LowOptimal: 50, HighOptimal: 90, LowCritical: 30, HighCritical: 200, ImprovementDir: "lower"},
		{SlotName: "medication_adherence", Domain: "behavioral", LowOptimal: 80, HighOptimal: 100, LowCritical: 0, HighCritical: 100, ImprovementDir: "higher"},
		{SlotName: "physical_activity_minutes", Domain: "lifestyle", LowOptimal: 150, HighOptimal: 300, LowCritical: 0, HighCritical: 600, ImprovementDir: "higher"},
		{SlotName: "sleep_hours", Domain: "lifestyle", LowOptimal: 7, HighOptimal: 9, LowCritical: 3, HighCritical: 14, ImprovementDir: "range"},
		{SlotName: "symptom_severity", Domain: "clinical", LowOptimal: 0, HighOptimal: 2, LowCritical: 0, HighCritical: 10, ImprovementDir: "lower"},
		{SlotName: "side_effects", Domain: "clinical", LowOptimal: 0, HighOptimal: 1, LowCritical: 0, HighCritical: 10, ImprovementDir: "lower"},
	}
}

// ComputeTrajectory computes the trajectory signal from input data.
// Deterministic rules:
//   - DISENGAGE if <50% slots filled or >3 days late
//   - Score each slot 0–1, classify by worsen ratio
//   - >=0.5 worsening → FAILURE, >=0.25 → FRAGILE, else STABLE
//   - Consecutive FRAGILE escalates to FAILURE
func ComputeTrajectory(input TrajectoryInput) TrajectoryResult {
	now := time.Now()
	result := TrajectoryResult{
		ComputedAt:   now,
		DomainScores: make(map[string]float64),
	}

	// Check disengagement first
	if isDisengaged(input) {
		result.Signal = DISENGAGE
		result.Confidence = 1.0
		return result
	}

	targets := DefaultClinicalTargets()
	targetMap := make(map[string]ClinicalTarget)
	for _, t := range targets {
		targetMap[t.SlotName] = t
	}

	// Determine reference values (previous if available, else baseline)
	reference := input.PreviousValues
	if len(reference) == 0 {
		reference = input.BaselineValues
	}

	var improving, worsening, stable, missed []string
	domainScores := make(map[string][]float64)

	for _, t := range targets {
		current, hasCurrent := input.CurrentValues[t.SlotName]
		ref, hasRef := reference[t.SlotName]

		if !hasCurrent {
			missed = append(missed, t.SlotName)
			continue
		}
		if !hasRef {
			// No reference — treat as stable
			stable = append(stable, t.SlotName)
			domainScores[t.Domain] = append(domainScores[t.Domain], 0.5)
			continue
		}

		score := scoreSlotChange(current, ref, t)
		domainScores[t.Domain] = append(domainScores[t.Domain], score)

		if score >= 0.6 {
			improving = append(improving, t.SlotName)
		} else if score <= 0.4 {
			worsening = append(worsening, t.SlotName)
		} else {
			stable = append(stable, t.SlotName)
		}
	}

	result.ImprovingSlots = improving
	result.WorseningSlots = worsening
	result.StableSlots = stable
	result.MissedSlots = missed

	// Compute domain averages
	for domain, scores := range domainScores {
		sum := 0.0
		for _, s := range scores {
			sum += s
		}
		result.DomainScores[domain] = sum / float64(len(scores))
	}

	// Classify by worsen ratio
	scoredSlots := len(improving) + len(worsening) + len(stable)
	if scoredSlots == 0 {
		result.Signal = DISENGAGE
		result.Confidence = 0.8
		return result
	}

	worsenRatio := float64(len(worsening)) / float64(scoredSlots)

	if worsenRatio >= 0.5 {
		result.Signal = FAILURE
		result.Confidence = 0.7 + 0.3*worsenRatio
	} else if worsenRatio >= 0.25 {
		result.Signal = FRAGILE
		result.Confidence = 0.6 + 0.2*worsenRatio
	} else {
		result.Signal = STABLE
		result.Confidence = 0.7 + 0.3*(1-worsenRatio)
	}

	// Consecutive FRAGILE escalation
	if result.Signal == FRAGILE && input.PreviousTrajectory == FRAGILE {
		result.Signal = FAILURE
		result.Confidence = math.Min(result.Confidence+0.1, 1.0)
	}

	return result
}

// isDisengaged returns true if the patient appears disengaged.
func isDisengaged(input TrajectoryInput) bool {
	if input.SlotsRequired > 0 {
		fillRatio := float64(input.SlotsFilled) / float64(input.SlotsRequired)
		if fillRatio < 0.5 {
			return true
		}
	} else if input.SlotsFilled == 0 {
		return true
	}
	if input.DaysSinceSchedule > 3 {
		return true
	}
	return false
}

// scoreSlotChange scores the change between current and reference values.
// Returns 0–1 where 1 = maximum improvement, 0 = maximum worsening.
// Weight uses % change; others use directional improvement normalized by target range.
// Critical values are clamped to 0.2.
func scoreSlotChange(current, reference float64, target ClinicalTarget) float64 {
	// Clamp critical values
	if current <= target.LowCritical || current >= target.HighCritical {
		return 0.2
	}

	targetRange := target.HighCritical - target.LowCritical
	if targetRange <= 0 {
		return 0.5
	}

	// Special handling for weight — use % change
	if target.SlotName == "weight" {
		if reference == 0 {
			return 0.5
		}
		pctChange := (current - reference) / reference
		// For weight, negative change (loss) is improvement
		switch target.ImprovementDir {
		case "lower":
			// 5% loss → good (~0.75), 10% gain → bad (~0.25)
			score := 0.5 - (pctChange * 5.0)
			return math.Max(0, math.Min(1, score))
		case "higher":
			score := 0.5 + (pctChange * 5.0)
			return math.Max(0, math.Min(1, score))
		default:
			return 0.5
		}
	}

	// Directional improvement normalized by target range
	delta := current - reference
	normalizedDelta := delta / targetRange

	var score float64
	switch target.ImprovementDir {
	case "lower":
		// Decrease is improvement
		score = 0.5 - (normalizedDelta * 3.0)
	case "higher":
		// Increase is improvement
		score = 0.5 + (normalizedDelta * 3.0)
	case "range":
		// Closer to optimal midpoint is better
		optMid := (target.LowOptimal + target.HighOptimal) / 2
		prevDist := math.Abs(reference - optMid)
		currDist := math.Abs(current - optMid)
		improvement := (prevDist - currDist) / targetRange
		score = 0.5 + (improvement * 3.0)
	default:
		score = 0.5
	}

	return math.Max(0, math.Min(1, score))
}
