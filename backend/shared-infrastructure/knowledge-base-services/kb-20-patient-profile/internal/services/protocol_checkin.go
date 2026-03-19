package services

// CheckinCadence defines the check-in schedule for a protocol phase.
type CheckinCadence struct {
	ProtocolID   string   `json:"protocol_id"`
	Phase        string   `json:"phase"`
	IntervalDays int      `json:"interval_days"` // default check-in interval
	FixedDays    []int    `json:"fixed_days"`    // specific days within phase (overrides interval)
	LabDays      []int    `json:"lab_days"`      // days requiring lab work
	LabTypes     []string `json:"lab_types"`     // labs required on lab_days
}

// GetCheckinCadence returns the check-in schedule for a given protocol+phase.
func GetCheckinCadence(protocolID, phase string) *CheckinCadence {
	key := protocolID + ":" + phase
	cadence, ok := checkinCadences[key]
	if !ok {
		return &CheckinCadence{
			ProtocolID:   protocolID,
			Phase:        phase,
			IntervalDays: 7, // default weekly
		}
	}
	return cadence
}

// NextCheckinDay returns the next check-in day relative to phase start,
// given the current day in phase.
func (c *CheckinCadence) NextCheckinDay(currentDayInPhase int) int {
	// If fixed days are defined, find the next one after current day
	if len(c.FixedDays) > 0 {
		for _, d := range c.FixedDays {
			if d > currentDayInPhase {
				return d
			}
		}
		// Past all fixed days — no more check-ins in this phase
		return -1
	}
	// Otherwise use interval
	next := ((currentDayInPhase / c.IntervalDays) + 1) * c.IntervalDays
	return next
}

// IsLabDay returns true if the given day in phase requires lab work.
func (c *CheckinCadence) IsLabDay(dayInPhase int) bool {
	for _, d := range c.LabDays {
		if d == dayInPhase {
			return true
		}
	}
	return false
}

// Registry of check-in cadences per protocol+phase.
var checkinCadences = map[string]*CheckinCadence{
	// PRP Phase 1: Protein Stabilization (Days 1-14)
	"M3-PRP:STABILIZATION": {
		ProtocolID: "M3-PRP",
		Phase:      "STABILIZATION",
		FixedDays:  []int{1, 4, 7, 11, 14},
	},
	// PRP Phase 2: Muscle Restoration (Days 15-42)
	"M3-PRP:RESTORATION": {
		ProtocolID:   "M3-PRP",
		Phase:        "RESTORATION",
		IntervalDays: 3, // biweekly-ish
		FixedDays:    []int{3, 7, 10, 14, 17, 21, 24, 28},
		LabDays:      []int{14}, // Day 28 absolute = Day 14 relative to phase start
		LabTypes:     []string{"creatinine", "egfr"},
	},
	// PRP Phase 3: Metabolic Optimization (Days 43-84)
	"M3-PRP:OPTIMIZATION": {
		ProtocolID:   "M3-PRP",
		Phase:        "OPTIMIZATION",
		IntervalDays: 7,
		LabDays:      []int{14, 42}, // Day 56 and Day 84 absolute
		LabTypes:     []string{"hba1c", "fbg", "lipid_panel", "egfr", "creatinine"},
	},
	// VFRP Phase 1: Metabolic Stabilization (Days 1-14)
	"M3-VFRP:METABOLIC_STABILIZATION": {
		ProtocolID:   "M3-VFRP",
		Phase:        "METABOLIC_STABILIZATION",
		IntervalDays: 4,
	},
	// VFRP Phase 2: Fat Mobilization (Days 15-42)
	"M3-VFRP:FAT_MOBILIZATION": {
		ProtocolID:   "M3-VFRP",
		Phase:        "FAT_MOBILIZATION",
		IntervalDays: 7, // biweekly
		LabDays:      []int{14}, // Day 28 absolute: optional waist measurement
		LabTypes:     []string{"waist_measurement"},
	},
	// VFRP Phase 3: Sustained Reduction (Days 43-84)
	"M3-VFRP:SUSTAINED_REDUCTION": {
		ProtocolID:   "M3-VFRP",
		Phase:        "SUSTAINED_REDUCTION",
		IntervalDays: 7,
		LabDays:      []int{14, 42}, // Day 56 waist, Day 84 full panel
		LabTypes:     []string{"waist_measurement", "hba1c", "fbg", "lipid_panel", "egfr"},
	},
}
