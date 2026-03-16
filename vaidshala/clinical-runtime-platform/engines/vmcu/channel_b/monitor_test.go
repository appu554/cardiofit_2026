package channel_b

import (
	"testing"
	"time"
)

func TestB01_ActiveHypoglycaemia(t *testing.T) {
	m := NewPhysiologySafetyMonitor(DefaultPhysioConfig())

	tests := []struct {
		name     string
		glucose  float64
		wantGate PhysioGate
		wantRule string
	}{
		{"glucose 3.89 → HALT", 3.89, PhysioHalt, "B-01"},
		{"glucose 3.90 → not B-01 (equals threshold, falls to B-02)", 3.90, PhysioPause, "B-02"},
		{"glucose 3.91 → not B-01 (near-hypo B-02)", 3.91, PhysioPause, "B-02"},
		{"glucose 5.0 → CLEAR", 5.0, PhysioClear, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := safeDefaults()
			data.GlucoseCurrent = Float64Ptr(tt.glucose)
			result := m.Evaluate(data)
			if result.Gate != tt.wantGate {
				t.Errorf("got gate %s, want %s", result.Gate, tt.wantGate)
			}
			if result.RuleFired != tt.wantRule {
				t.Errorf("got rule %s, want %s", result.RuleFired, tt.wantRule)
			}
		})
	}
}

func TestB02_NearHypoglycaemia(t *testing.T) {
	m := NewPhysiologySafetyMonitor(DefaultPhysioConfig())

	data := safeDefaults()
	data.GlucoseCurrent = Float64Ptr(4.2)
	result := m.Evaluate(data)
	if result.Gate != PhysioPause || result.RuleFired != "B-02" {
		t.Errorf("got gate=%s rule=%s, want PAUSE/B-02", result.Gate, result.RuleFired)
	}
}

func TestB03_AKI(t *testing.T) {
	m := NewPhysiologySafetyMonitor(DefaultPhysioConfig())

	prior := 80.0
	data := safeDefaults()
	data.CreatinineCurrent = Float64Ptr(110) // delta = 30 > 26
	data.Creatinine48hAgo = &prior
	result := m.Evaluate(data)
	if result.Gate != PhysioHalt || result.RuleFired != "B-03" {
		t.Errorf("got gate=%s rule=%s, want HALT/B-03", result.Gate, result.RuleFired)
	}

	// Below threshold
	data.CreatinineCurrent = Float64Ptr(100) // delta = 20 < 26
	result = m.Evaluate(data)
	if result.Gate != PhysioClear {
		t.Errorf("creatinine delta 20 should be CLEAR, got %s", result.Gate)
	}
}

func TestB04_PotassiumExtremes(t *testing.T) {
	m := NewPhysiologySafetyMonitor(DefaultPhysioConfig())

	tests := []struct {
		name      string
		potassium float64
		wantGate  PhysioGate
	}{
		{"K+ 2.99 → HALT (low)", 2.99, PhysioHalt},
		{"K+ 3.01 → CLEAR", 3.01, PhysioClear},
		{"K+ 4.5 → CLEAR (normal)", 4.5, PhysioClear},
		{"K+ 5.99 → CLEAR", 5.99, PhysioClear},
		{"K+ 6.01 → HALT (high)", 6.01, PhysioHalt},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := safeDefaults()
			data.PotassiumCurrent = Float64Ptr(tt.potassium)
			result := m.Evaluate(data)
			if result.Gate != tt.wantGate {
				t.Errorf("got gate %s, want %s", result.Gate, tt.wantGate)
			}
		})
	}
}

func TestB05_HaemodynamicInstability(t *testing.T) {
	m := NewPhysiologySafetyMonitor(DefaultPhysioConfig())

	data := safeDefaults()
	data.SBPCurrent = Float64Ptr(85)
	result := m.Evaluate(data)
	if result.Gate != PhysioHalt || result.RuleFired != "B-05" {
		t.Errorf("SBP 85 should HALT, got gate=%s rule=%s", result.Gate, result.RuleFired)
	}

	data.SBPCurrent = Float64Ptr(95)
	result = m.Evaluate(data)
	if result.Gate != PhysioClear {
		t.Errorf("SBP 95 should CLEAR, got %s", result.Gate)
	}
}

func TestB06_FluidOverload(t *testing.T) {
	m := NewPhysiologySafetyMonitor(DefaultPhysioConfig())

	prior := 70.0
	data := safeDefaults()
	data.WeightKgCurrent = Float64Ptr(73.0) // delta = 3.0 > 2.5
	data.Weight72hAgo = &prior
	result := m.Evaluate(data)
	if result.Gate != PhysioPause || result.RuleFired != "B-06" {
		t.Errorf("weight delta 3.0 should PAUSE, got gate=%s rule=%s", result.Gate, result.RuleFired)
	}
}

func TestB07_GlucoseTrajectory(t *testing.T) {
	m := NewPhysiologySafetyMonitor(DefaultPhysioConfig())

	now := time.Now()
	data := safeDefaults()
	data.GlucoseCurrent = Float64Ptr(5.0)
	data.RecentDoseIncrease = true
	data.GlucoseReadings = []TimestampedValue{
		{Value: 5.0, Timestamp: now},
		{Value: 5.5, Timestamp: now.Add(-1 * time.Hour)},
		{Value: 6.0, Timestamp: now.Add(-2 * time.Hour)},
	}
	result := m.Evaluate(data)
	if result.Gate != PhysioPause || result.RuleFired != "B-07" {
		t.Errorf("declining glucose trajectory should PAUSE, got gate=%s rule=%s", result.Gate, result.RuleFired)
	}

	// Without recent dose increase → no B-07
	data.RecentDoseIncrease = false
	result = m.Evaluate(data)
	if result.RuleFired == "B-07" {
		t.Error("B-07 should not fire without recent dose increase")
	}
}

func TestDA01_EGFRAnomaly(t *testing.T) {
	m := NewPhysiologySafetyMonitor(DefaultPhysioConfig())

	prior := 52.0
	data := safeDefaults()
	data.EGFRCurrent = Float64Ptr(28.0) // 46% drop
	data.EGFRPrior48h = &prior
	result := m.Evaluate(data)
	if result.Gate != PhysioHoldData || result.RuleFired != "DA-01" {
		t.Errorf("eGFR 46%% drop should HOLD_DATA, got gate=%s rule=%s", result.Gate, result.RuleFired)
	}
	if !result.IsAnomaly || result.AnomalyLab != "EGFR" {
		t.Errorf("should flag EGFR anomaly, got anomaly=%v lab=%s", result.IsAnomaly, result.AnomalyLab)
	}
}

func TestDA04_HbA1cAnomaly(t *testing.T) {
	m := NewPhysiologySafetyMonitor(DefaultPhysioConfig())

	prior := 7.0
	data := safeDefaults()
	data.HbA1cCurrent = Float64Ptr(9.1) // delta 2.1 > 2.0
	data.HbA1cPrior30d = &prior
	result := m.Evaluate(data)
	if result.Gate != PhysioHoldData || result.RuleFired != "DA-04" {
		t.Errorf("HbA1c 2.1%% change should HOLD_DATA, got gate=%s rule=%s", result.Gate, result.RuleFired)
	}
}

func TestDA05_ExtremePotassium(t *testing.T) {
	m := NewPhysiologySafetyMonitor(DefaultPhysioConfig())

	data := safeDefaults()
	data.PotassiumCurrent = Float64Ptr(8.5) // > 8.0 → HOLD_DATA (DA-05 fires before B-04)
	result := m.Evaluate(data)
	if result.Gate != PhysioHoldData || result.RuleFired != "DA-05" {
		t.Errorf("K+ 8.5 should trigger DA-05 HOLD_DATA first, got gate=%s rule=%s", result.Gate, result.RuleFired)
	}
}

func TestDA02_CreatinineAnomaly(t *testing.T) {
	m := NewPhysiologySafetyMonitor(DefaultPhysioConfig())

	// Creatinine doubles in 48h (100% change) → HOLD_DATA
	prior := 60.0
	data := safeDefaults()
	data.CreatinineCurrent = Float64Ptr(125) // 108% rise > 100%
	data.Creatinine48hAgo = &prior
	result := m.Evaluate(data)
	if result.Gate != PhysioHoldData || result.RuleFired != "DA-02" {
		t.Errorf("creatinine 108%% rise should trigger DA-02 HOLD_DATA, got gate=%s rule=%s", result.Gate, result.RuleFired)
	}
	if !result.IsAnomaly || result.AnomalyLab != "CREATININE" {
		t.Errorf("should flag CREATININE anomaly, got anomaly=%v lab=%s", result.IsAnomaly, result.AnomalyLab)
	}

	// 90% rise → below threshold, should NOT trigger DA-02
	data.CreatinineCurrent = Float64Ptr(114) // 90% rise < 100%
	result = m.Evaluate(data)
	if result.RuleFired == "DA-02" {
		t.Errorf("creatinine 90%% rise should not trigger DA-02, got gate=%s rule=%s", result.Gate, result.RuleFired)
	}
}

func TestDA06_StalePotassium(t *testing.T) {
	m := NewPhysiologySafetyMonitor(DefaultPhysioConfig())
	now := time.Now()

	tests := []struct {
		name     string
		kValue   *float64
		kTime    *time.Time
		wantGate PhysioGate
		wantRule string
	}{
		{
			name:     "K+ 15 days old → HOLD_DATA (stale)",
			kValue:   Float64Ptr(4.5),
			kTime:    timePtr(now.Add(-15 * 24 * time.Hour)),
			wantGate: PhysioHoldData,
			wantRule: "DA-06",
		},
		{
			name:     "K+ 13 days old → CLEAR (fresh)",
			kValue:   Float64Ptr(4.5),
			kTime:    timePtr(now.Add(-13 * 24 * time.Hour)),
			wantGate: PhysioClear,
			wantRule: "",
		},
		{
			name:     "K+ nil timestamp → CLEAR (nil = unknown, not stale)",
			kValue:   Float64Ptr(4.5),
			kTime:    nil,
			wantGate: PhysioClear,
			wantRule: "",
		},
		{
			name:     "K+ nil value + stale timestamp → CLEAR",
			kValue:   nil,
			kTime:    timePtr(now.Add(-15 * 24 * time.Hour)),
			wantGate: PhysioClear,
			wantRule: "",
		},
		{
			name:     "K+ exactly 14 days old → boundary (not stale, equal is not past)",
			kValue:   Float64Ptr(4.5),
			kTime:    timePtr(now.Add(-14 * 24 * time.Hour)),
			wantGate: PhysioHoldData,
			wantRule: "DA-06",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := safeDefaults()
			data.PotassiumCurrent = tt.kValue
			data.PotassiumLastMeasuredAt = tt.kTime
			result := m.Evaluate(data)
			if result.Gate != tt.wantGate {
				t.Errorf("got gate %s, want %s", result.Gate, tt.wantGate)
			}
			if tt.wantRule != "" && result.RuleFired != tt.wantRule {
				t.Errorf("got rule %s, want %s", result.RuleFired, tt.wantRule)
			}
			if tt.wantRule == "DA-06" {
				if !result.IsAnomaly || result.AnomalyLab != "POTASSIUM" {
					t.Errorf("should flag POTASSIUM anomaly, got anomaly=%v lab=%s", result.IsAnomaly, result.AnomalyLab)
				}
			}
		})
	}
}

func TestDA07_StaleCreatinine(t *testing.T) {
	m := NewPhysiologySafetyMonitor(DefaultPhysioConfig())
	now := time.Now()

	tests := []struct {
		name     string
		crValue  *float64
		crTime   *time.Time
		wantGate PhysioGate
		wantRule string
	}{
		{
			name:     "Creatinine 31 days old → HOLD_DATA (stale)",
			crValue:  Float64Ptr(80),
			crTime:   timePtr(now.Add(-31 * 24 * time.Hour)),
			wantGate: PhysioHoldData,
			wantRule: "DA-07",
		},
		{
			name:     "Creatinine 29 days old → CLEAR (fresh)",
			crValue:  Float64Ptr(80),
			crTime:   timePtr(now.Add(-29 * 24 * time.Hour)),
			wantGate: PhysioClear,
			wantRule: "",
		},
		{
			name:     "Creatinine nil timestamp → CLEAR (nil = unknown, not stale)",
			crValue:  Float64Ptr(80),
			crTime:   nil,
			wantGate: PhysioClear,
			wantRule: "",
		},
		{
			name:     "Creatinine nil value + stale timestamp → CLEAR",
			crValue:  nil,
			crTime:   timePtr(now.Add(-31 * 24 * time.Hour)),
			wantGate: PhysioClear,
			wantRule: "",
		},
		{
			name:     "Creatinine exactly 30 days old → boundary",
			crValue:  Float64Ptr(80),
			crTime:   timePtr(now.Add(-30 * 24 * time.Hour)),
			wantGate: PhysioHoldData,
			wantRule: "DA-07",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := safeDefaults()
			data.CreatinineCurrent = tt.crValue
			data.CreatinineLastMeasuredAt = tt.crTime
			result := m.Evaluate(data)
			if result.Gate != tt.wantGate {
				t.Errorf("got gate %s, want %s", result.Gate, tt.wantGate)
			}
			if tt.wantRule != "" && result.RuleFired != tt.wantRule {
				t.Errorf("got rule %s, want %s", result.RuleFired, tt.wantRule)
			}
			if tt.wantRule == "DA-07" {
				if !result.IsAnomaly || result.AnomalyLab != "CREATININE" {
					t.Errorf("should flag CREATININE anomaly, got anomaly=%v lab=%s", result.IsAnomaly, result.AnomalyLab)
				}
			}
		})
	}
}

// timePtr returns a pointer to the given time.Time value.
func timePtr(t time.Time) *time.Time { return &t }

func TestDataAnomalyPriority(t *testing.T) {
	// DA rules fire BEFORE clinical rules. Even if glucose is < 3.9 (B-01 HALT),
	// if glucose < 1.0 (DA-03 HOLD_DATA), the anomaly check fires first.
	m := NewPhysiologySafetyMonitor(DefaultPhysioConfig())

	data := safeDefaults()
	data.GlucoseCurrent = Float64Ptr(0.5) // < 1.0 (DA-03) AND < 3.9 (B-01)
	result := m.Evaluate(data)
	if result.Gate != PhysioHoldData || result.RuleFired != "DA-03" {
		t.Errorf("glucose 0.5 should trigger DA-03 before B-01, got gate=%s rule=%s", result.Gate, result.RuleFired)
	}
}

func TestAllClear(t *testing.T) {
	m := NewPhysiologySafetyMonitor(DefaultPhysioConfig())

	data := safeDefaults()
	result := m.Evaluate(data)
	if result.Gate != PhysioClear {
		t.Errorf("safe defaults should be CLEAR, got %s/%s", result.Gate, result.RuleFired)
	}
	if result.RawValues == nil {
		t.Error("raw values should always be populated")
	}
}

// ════════════════════════════════════════════════════════════════════════
// HTN CO-MANAGEMENT RULES (B-03 RAAS suppression, B-12 J-curve)
// ════════════════════════════════════════════════════════════════════════

func TestB03_RAASToleranceSuppression(t *testing.T) {
	m := NewPhysiologySafetyMonitor(DefaultPhysioConfig())

	tests := []struct {
		name                    string
		creatinineCurrent       float64
		creatinine48hAgo        float64
		creatinineRiseExplained bool
		oliguriaReported        bool
		potassium               *float64
		wantGate                PhysioGate
		wantRule                string
	}{
		{
			name:              "AKI without RAAS context → HALT",
			creatinineCurrent: 110, creatinine48hAgo: 80,
			wantGate: PhysioHalt, wantRule: "B-03",
		},
		{
			name:              "RAAS explained + safe K+ + no oliguria → PAUSE (suppressed)",
			creatinineCurrent: 110, creatinine48hAgo: 80,
			creatinineRiseExplained: true,
			potassium:               Float64Ptr(4.8),
			wantGate:                PhysioPause, wantRule: "B-03-RAAS-SUPPRESSED",
		},
		{
			name:              "RAAS explained but K+ ≥5.5 → HALT (hyperkalaemia overrides)",
			creatinineCurrent: 110, creatinine48hAgo: 80,
			creatinineRiseExplained: true,
			potassium:               Float64Ptr(5.6),
			wantGate:                PhysioHalt, wantRule: "B-03",
		},
		{
			name:              "RAAS explained but oliguria → HALT (clinician override)",
			creatinineCurrent: 110, creatinine48hAgo: 80,
			creatinineRiseExplained: true,
			oliguriaReported:        true,
			potassium:               Float64Ptr(4.2),
			wantGate:                PhysioHalt, wantRule: "B-03",
		},
		{
			name:              "RAAS explained + nil K+ (absent) → PAUSE (nil is safe)",
			creatinineCurrent: 110, creatinine48hAgo: 80,
			creatinineRiseExplained: true,
			potassium:               nil,
			wantGate:                PhysioPause, wantRule: "B-03-RAAS-SUPPRESSED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := safeDefaults()
			data.CreatinineCurrent = Float64Ptr(tt.creatinineCurrent)
			prior := tt.creatinine48hAgo
			data.Creatinine48hAgo = &prior
			data.CreatinineRiseExplained = tt.creatinineRiseExplained
			data.OliguriaReported = tt.oliguriaReported
			if tt.potassium != nil {
				data.PotassiumCurrent = tt.potassium
			} else {
				data.PotassiumCurrent = nil
			}
			result := m.Evaluate(data)
			if result.Gate != tt.wantGate {
				t.Errorf("got gate %s, want %s", result.Gate, tt.wantGate)
			}
			if result.RuleFired != tt.wantRule {
				t.Errorf("got rule %s, want %s", result.RuleFired, tt.wantRule)
			}
		})
	}
}

func TestB12_JCurveSBPFloor(t *testing.T) {
	m := NewPhysiologySafetyMonitor(DefaultPhysioConfig())

	tests := []struct {
		name     string
		sbp      float64
		ckdStage string
		wantGate PhysioGate
		wantRule string
	}{
		// CKD 3a: floor 120 mmHg
		{"CKD3a SBP 119 → PAUSE", 119, "3a", PhysioPause, "B-12"},
		{"CKD3a SBP 121 → CLEAR", 121, "3a", PhysioClear, ""},
		// CKD 3b: floor 125 mmHg
		{"CKD3b SBP 124 → PAUSE", 124, "3b", PhysioPause, "B-12"},
		{"CKD3b SBP 126 → CLEAR", 126, "3b", PhysioClear, ""},
		// CKD 4: floor 130 mmHg
		{"CKD4 SBP 129 → PAUSE", 129, "4", PhysioPause, "B-12"},
		{"CKD4 SBP 131 → CLEAR", 131, "4", PhysioClear, ""},
		// Amendment 8: stage-specific hard thresholds (replaces unified SBP<110 HALT)
		// CKD 3a: SBP < 100 → PAUSE (autoregulation partially intact, no HALT)
		{"CKD3a SBP 99 → PAUSE (B-12-3A)", 99, "3a", PhysioPause, "B-12-3A"},
		{"CKD3a SBP 109 → PAUSE (below floor 120)", 109, "3a", PhysioPause, "B-12"},
		// CKD 3b: SBP < 105 → PAUSE (reduced reserve, no HALT)
		{"CKD3b SBP 104 → PAUSE (B-12-3B)", 104, "3b", PhysioPause, "B-12-3B"},
		{"CKD3b SBP 109 → PAUSE (below floor 125)", 109, "3b", PhysioPause, "B-12"},
		// CKD 4: SBP < 100 → HALT, SBP 100-110 → PAUSE
		{"CKD4 SBP 99 → HALT (B-12-4-HALT)", 99, "4", PhysioHalt, "B-12-4-HALT"},
		{"CKD4 SBP 105 → PAUSE (B-12-4-PAUSE)", 105, "4", PhysioPause, "B-12-4-PAUSE"},
		{"CKD4 SBP 109 → PAUSE (B-12-4-PAUSE)", 109, "4", PhysioPause, "B-12-4-PAUSE"},
		// No CKD stage → no B-12
		{"No CKD SBP 100 → only B-05 if <90", 100, "", PhysioClear, ""},
		// CKD 1-2 → no J-curve concern
		{"CKD2 SBP 115 → CLEAR", 115, "2", PhysioClear, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := safeDefaults()
			data.SBPCurrent = Float64Ptr(tt.sbp)
			data.CKDStage = tt.ckdStage
			result := m.Evaluate(data)
			if result.Gate != tt.wantGate {
				t.Errorf("got gate %s, want %s", result.Gate, tt.wantGate)
			}
			if tt.wantRule != "" && result.RuleFired != tt.wantRule {
				t.Errorf("got rule %s, want %s", result.RuleFired, tt.wantRule)
			}
		})
	}
}

func TestB12_PrecomputedSBPLowerLimit(t *testing.T) {
	// When orchestrator pre-computes SBPLowerLimit, it takes precedence
	m := NewPhysiologySafetyMonitor(DefaultPhysioConfig())

	data := safeDefaults()
	data.SBPCurrent = Float64Ptr(122)
	data.CKDStage = "3b"
	data.SBPLowerLimit = Float64Ptr(123) // orchestrator says floor is 123
	result := m.Evaluate(data)
	if result.Gate != PhysioPause || result.RuleFired != "B-12" {
		t.Errorf("SBP 122 < precomputed floor 123 should PAUSE, got gate=%s rule=%s", result.Gate, result.RuleFired)
	}
}

func TestB05_TakesPriorityOverB12(t *testing.T) {
	// B-05 (SBP <90 HALT) fires before B-12 in evaluation order
	m := NewPhysiologySafetyMonitor(DefaultPhysioConfig())

	data := safeDefaults()
	data.SBPCurrent = Float64Ptr(85)
	data.CKDStage = "4"
	result := m.Evaluate(data)
	if result.RuleFired != "B-05" {
		t.Errorf("B-05 should fire before B-12 for SBP 85, got rule=%s", result.RuleFired)
	}
}

// ════════════════════════════════════════════════════════════════════════
// MEASUREMENT UNCERTAINTY DAMPENING TESTS
// ════════════════════════════════════════════════════════════════════════

func TestB05_HighUncertainty_Dampened(t *testing.T) {
	m := NewPhysiologySafetyMonitor(DefaultPhysioConfig())

	// SBP=88 with HIGH uncertainty (σ=18 mmHg) → PAUSE not HALT
	data := safeDefaults()
	data.SBPCurrent = Float64Ptr(88)
	data.MeasurementUncertainty = 18 // ≥15 triggers dampening
	result := m.Evaluate(data)
	if result.Gate != PhysioPause || result.RuleFired != "B-05-DAMPENED" {
		t.Errorf("SBP=88 + high uncertainty should PAUSE (B-05-DAMPENED), got gate=%s rule=%s", result.Gate, result.RuleFired)
	}

	// SBP=88 with LOW uncertainty (σ=5 mmHg) → HALT as normal
	data.MeasurementUncertainty = 5
	result = m.Evaluate(data)
	if result.Gate != PhysioHalt || result.RuleFired != "B-05" {
		t.Errorf("SBP=88 + low uncertainty should HALT (B-05), got gate=%s rule=%s", result.Gate, result.RuleFired)
	}

	// SBP=88 with zero uncertainty (default) → HALT as normal
	data.MeasurementUncertainty = 0
	result = m.Evaluate(data)
	if result.Gate != PhysioHalt || result.RuleFired != "B-05" {
		t.Errorf("SBP=88 + zero uncertainty should HALT (B-05), got gate=%s rule=%s", result.Gate, result.RuleFired)
	}
}

func TestB12_HighUncertainty_Dampened(t *testing.T) {
	m := NewPhysiologySafetyMonitor(DefaultPhysioConfig())

	// Amendment 8: uncertainty dampening only applies to Stage 4 HALT (SBP < 100).
	// SBP=108 is in the Stage 4 PAUSE zone (100-110), not HALT, so no dampening.

	// SBP=95 + CKD 4 + HIGH uncertainty → B-12-DAMPENED (PAUSE not HALT)
	data := safeDefaults()
	data.SBPCurrent = Float64Ptr(95)
	data.CKDStage = "4"
	data.MeasurementUncertainty = 20
	result := m.Evaluate(data)
	if result.Gate != PhysioPause || result.RuleFired != "B-12-DAMPENED" {
		t.Errorf("SBP=95 + CKD4 + high uncertainty should PAUSE (B-12-DAMPENED), got gate=%s rule=%s", result.Gate, result.RuleFired)
	}

	// Same but LOW uncertainty → HALT
	data.MeasurementUncertainty = 3
	result = m.Evaluate(data)
	if result.Gate != PhysioHalt || result.RuleFired != "B-12-4-HALT" {
		t.Errorf("SBP=95 + CKD4 + low uncertainty should HALT (B-12-4-HALT), got gate=%s rule=%s", result.Gate, result.RuleFired)
	}

	// SBP=108 + CKD 4 → PAUSE (cautionary zone, no HALT regardless of uncertainty)
	data.SBPCurrent = Float64Ptr(108)
	data.MeasurementUncertainty = 3
	result = m.Evaluate(data)
	if result.Gate != PhysioPause || result.RuleFired != "B-12-4-PAUSE" {
		t.Errorf("SBP=108 + CKD4 should PAUSE (B-12-4-PAUSE), got gate=%s rule=%s", result.Gate, result.RuleFired)
	}
}

// ════════════════════════════════════════════════════════════════════════
// HEART RATE RULES (B-13 through B-16) — Wave 2
// ════════════════════════════════════════════════════════════════════════

func TestB13_SevereBradycardia(t *testing.T) {
	m := NewPhysiologySafetyMonitor(DefaultPhysioConfig())

	tests := []struct {
		name      string
		hr        float64
		context   string
		confirmed bool
		wantGate  PhysioGate
		wantRule  string
	}{
		{"HR 42 resting confirmed → HALT", 42, "RESTING", true, PhysioHalt, "B-13"},
		{"HR 48 resting confirmed → CLEAR (above 45)", 48, "RESTING", true, PhysioClear, ""},
		{"HR 42 resting NOT confirmed → CLEAR (unconfirmed)", 42, "RESTING", false, PhysioClear, ""},
		{"HR 42 POST_ACTIVITY confirmed → CLEAR (not resting)", 42, "POST_ACTIVITY", true, PhysioClear, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := safeDefaults()
			data.HeartRateCurrent = Float64Ptr(tt.hr)
			data.HRContext = tt.context
			data.HeartRateConfirmed = tt.confirmed
			result := m.Evaluate(data)
			if result.Gate != tt.wantGate {
				t.Errorf("got gate %s, want %s", result.Gate, tt.wantGate)
			}
			if tt.wantRule != "" && result.RuleFired != tt.wantRule {
				t.Errorf("got rule %s, want %s", result.RuleFired, tt.wantRule)
			}
		})
	}
}

func TestB14_BetaBlockerBradycardia(t *testing.T) {
	m := NewPhysiologySafetyMonitor(DefaultPhysioConfig())

	tests := []struct {
		name           string
		hr             float64
		bbActive       bool
		bbDoseChange7d bool
		wantGate       PhysioGate
		wantRule       string
	}{
		{"HR 52 + BB + dose change → PAUSE", 52, true, true, PhysioPause, "B-14"},
		{"HR 52 + BB + no dose change → CLEAR", 52, true, false, PhysioClear, ""},
		{"HR 52 + no BB → CLEAR", 52, false, true, PhysioClear, ""},
		{"HR 58 + BB + dose change → CLEAR (above 55)", 58, true, true, PhysioClear, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := safeDefaults()
			data.HeartRateCurrent = Float64Ptr(tt.hr)
			data.BetaBlockerActive = tt.bbActive
			data.BetaBlockerDoseChangeIn7d = tt.bbDoseChange7d
			result := m.Evaluate(data)
			if result.Gate != tt.wantGate {
				t.Errorf("got gate %s, want %s", result.Gate, tt.wantGate)
			}
			if tt.wantRule != "" && result.RuleFired != tt.wantRule {
				t.Errorf("got rule %s, want %s", result.RuleFired, tt.wantRule)
			}
		})
	}
}

func TestB15_RestingTachycardia(t *testing.T) {
	m := NewPhysiologySafetyMonitor(DefaultPhysioConfig())

	tests := []struct {
		name      string
		hr        float64
		context   string
		confirmed bool
		wantGate  PhysioGate
		wantRule  string
	}{
		{"HR 125 resting confirmed → PAUSE", 125, "RESTING", true, PhysioPause, "B-15"},
		{"HR 118 resting confirmed → CLEAR (below 120)", 118, "RESTING", true, PhysioClear, ""},
		{"HR 125 POST_ACTIVITY confirmed → CLEAR (not resting)", 125, "POST_ACTIVITY", true, PhysioClear, ""},
		{"HR 125 resting NOT confirmed → CLEAR", 125, "RESTING", false, PhysioClear, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := safeDefaults()
			data.HeartRateCurrent = Float64Ptr(tt.hr)
			data.HRContext = tt.context
			data.HeartRateConfirmed = tt.confirmed
			result := m.Evaluate(data)
			if result.Gate != tt.wantGate {
				t.Errorf("got gate %s, want %s", result.Gate, tt.wantGate)
			}
			if tt.wantRule != "" && result.RuleFired != tt.wantRule {
				t.Errorf("got rule %s, want %s", result.RuleFired, tt.wantRule)
			}
		})
	}
}

func TestB16_IrregularRhythm(t *testing.T) {
	m := NewPhysiologySafetyMonitor(DefaultPhysioConfig())

	// Confirmed irregular → PAUSE with KB22_TRIGGER
	data := safeDefaults()
	data.HeartRateCurrent = Float64Ptr(82)
	data.HRRegularity = "IRREGULAR"
	data.HeartRateConfirmed = true
	result := m.Evaluate(data)
	if result.Gate != PhysioPause || result.RuleFired != "B-16" {
		t.Errorf("confirmed irregular HR should PAUSE (B-16), got gate=%s rule=%s", result.Gate, result.RuleFired)
	}
	// Check KB22_TRIGGER sentinel
	if result.RawValues["kb22_trigger"] != 1 {
		t.Error("B-16 should set kb22_trigger=1 in raw values for orchestrator")
	}

	// Unconfirmed irregular → CLEAR
	data.HeartRateConfirmed = false
	result = m.Evaluate(data)
	if result.RuleFired == "B-16" {
		t.Error("unconfirmed irregular should not fire B-16")
	}

	// Regular rhythm → CLEAR
	data.HRRegularity = "REGULAR"
	data.HeartRateConfirmed = true
	result = m.Evaluate(data)
	if result.RuleFired == "B-16" {
		t.Error("regular rhythm should not fire B-16")
	}
}

// ════════════════════════════════════════════════════════════════════════
// HYPONATRAEMIA RULES (B-17 through B-19) — Wave 2
// ════════════════════════════════════════════════════════════════════════

func TestB17_SevereHyponatraemia(t *testing.T) {
	m := NewPhysiologySafetyMonitor(DefaultPhysioConfig())

	tests := []struct {
		name     string
		sodium   float64
		thiazide bool
		wantGate PhysioGate
		wantRule string
	}{
		{"Na+ 130 + thiazide → HALT", 130, true, PhysioHalt, "B-17"},
		{"Na+ 130 + no thiazide → CLEAR", 130, false, PhysioClear, ""},
		{"Na+ 133 + thiazide → not B-17 (above 132)", 133, true, PhysioPause, "B-18"}, // falls to B-18
		{"Na+ 140 + thiazide → CLEAR", 140, true, PhysioClear, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := safeDefaults()
			data.SodiumCurrent = Float64Ptr(tt.sodium)
			data.ThiazideActive = tt.thiazide
			result := m.Evaluate(data)
			if result.Gate != tt.wantGate {
				t.Errorf("got gate %s, want %s", result.Gate, tt.wantGate)
			}
			if tt.wantRule != "" && result.RuleFired != tt.wantRule {
				t.Errorf("got rule %s, want %s", result.RuleFired, tt.wantRule)
			}
		})
	}
}

func TestB18_MildHyponatraemia(t *testing.T) {
	m := NewPhysiologySafetyMonitor(DefaultPhysioConfig())

	// Na+ 133 + thiazide → PAUSE
	data := safeDefaults()
	data.SodiumCurrent = Float64Ptr(133)
	data.ThiazideActive = true
	result := m.Evaluate(data)
	if result.Gate != PhysioPause || result.RuleFired != "B-18" {
		t.Errorf("Na+ 133 + thiazide should PAUSE (B-18), got gate=%s rule=%s", result.Gate, result.RuleFired)
	}

	// Na+ 136 → above threshold, CLEAR
	data.SodiumCurrent = Float64Ptr(136)
	result = m.Evaluate(data)
	if result.RuleFired == "B-18" {
		t.Error("Na+ 136 should not fire B-18")
	}
}

func TestB19_SeasonalHyponatraemia(t *testing.T) {
	m := NewPhysiologySafetyMonitor(DefaultPhysioConfig())

	// Na+ 134 + thiazide + SUMMER → PAUSE (B-19)
	data := safeDefaults()
	data.SodiumCurrent = Float64Ptr(134)
	data.ThiazideActive = true
	data.Season = "SUMMER"
	result := m.Evaluate(data)
	// B-18 fires first (Na+ 132-135 + thiazide), but B-19 would also apply
	// Since B-18 fires first in evaluation order, it gets B-18
	if result.Gate != PhysioPause {
		t.Errorf("Na+ 134 + thiazide + SUMMER should PAUSE, got gate=%s", result.Gate)
	}

	// Na+ 134 + thiazide + WINTER → B-18 only (no seasonal amplification)
	data.Season = "WINTER"
	result = m.Evaluate(data)
	if result.Gate != PhysioPause || result.RuleFired != "B-18" {
		t.Errorf("Na+ 134 + thiazide + WINTER should be B-18, got gate=%s rule=%s", result.Gate, result.RuleFired)
	}

	// Na+ 134.5 + thiazide + SUMMER → B-18 fires (134.5 < 135), covers seasonal too
	data.SodiumCurrent = Float64Ptr(134.5)
	data.Season = "SUMMER"
	result = m.Evaluate(data)
	if result.Gate != PhysioPause {
		t.Errorf("Na+ 134.5 + thiazide + SUMMER should PAUSE, got gate=%s", result.Gate)
	}
}

// ════════════════════════════════════════════════════════════════════════
// GLUCOSE VARIABILITY RULE (B-20) — Wave 2
// ════════════════════════════════════════════════════════════════════════

func TestB20_GlucoseCVHighVariability(t *testing.T) {
	cfg := DefaultPhysioConfig()
	m := NewPhysiologySafetyMonitor(cfg)
	d := safeDefaults()
	d.GlucoseCV30d = Float64Ptr(38.5) // > 36% threshold
	result := m.Evaluate(d)
	if result.Gate != PhysioPause {
		t.Errorf("B-20: gate = %v, want PAUSE", result.Gate)
	}
	if result.RuleFired != "B-20" {
		t.Errorf("RuleFired = %v, want B-20", result.RuleFired)
	}
}

func TestB20_GlucoseCVNormal(t *testing.T) {
	cfg := DefaultPhysioConfig()
	m := NewPhysiologySafetyMonitor(cfg)
	d := safeDefaults()
	d.GlucoseCV30d = Float64Ptr(25.0) // < 36%
	result := m.Evaluate(d)
	if result.RuleFired == "B-20" {
		t.Errorf("B-20 should not fire for CV 25%%")
	}
}

func TestB20_GlucoseCVNil(t *testing.T) {
	cfg := DefaultPhysioConfig()
	m := NewPhysiologySafetyMonitor(cfg)
	d := safeDefaults()
	// GlucoseCV30d is nil — should not fire
	result := m.Evaluate(d)
	if result.RuleFired == "B-20" {
		t.Errorf("B-20 should not fire for nil CV")
	}
}

func TestB20_GlucoseCVBoundary(t *testing.T) {
	cfg := DefaultPhysioConfig()
	m := NewPhysiologySafetyMonitor(cfg)

	// Exactly 36.0 — should NOT fire (> not >=)
	d := safeDefaults()
	d.GlucoseCV30d = Float64Ptr(36.0)
	result := m.Evaluate(d)
	if result.RuleFired == "B-20" {
		t.Errorf("B-20 should not fire for CV exactly at threshold (36.0)")
	}

	// Just above — should fire
	d.GlucoseCV30d = Float64Ptr(36.01)
	result = m.Evaluate(d)
	if result.Gate != PhysioPause || result.RuleFired != "B-20" {
		t.Errorf("B-20 should fire for CV 36.01, got gate=%s rule=%s", result.Gate, result.RuleFired)
	}
}

// ════════════════════════════════════════════════════════════════════════
// FINERENONE HYPERKALEMIA RULE (B-21)
// ════════════════════════════════════════════════════════════════════════

func TestB21_FinerenoneHyperkalemia(t *testing.T) {
	cfg := DefaultPhysioConfig()
	m := NewPhysiologySafetyMonitor(cfg)

	tests := []struct {
		name     string
		data     *RawPatientData
		wantGate PhysioGate
		wantRule string
	}{
		{"K+ 5.8 + finerenone → HALT", &RawPatientData{
			PotassiumCurrent: Float64Ptr(5.8),
			FinerenoneActive: true,
			GlucoseCurrent:   Float64Ptr(6.0),
			GlucoseTimestamp: time.Now(),
		}, PhysioHalt, "B-21"},
		{"K+ 5.5 + finerenone → HALT (boundary)", &RawPatientData{
			PotassiumCurrent: Float64Ptr(5.5),
			FinerenoneActive: true,
			GlucoseCurrent:   Float64Ptr(6.0),
			GlucoseTimestamp: time.Now(),
		}, PhysioHalt, "B-21"},
		{"K+ 5.4 + finerenone → not B-21", &RawPatientData{
			PotassiumCurrent: Float64Ptr(5.4),
			FinerenoneActive: true,
			GlucoseCurrent:   Float64Ptr(6.0),
			GlucoseTimestamp: time.Now(),
		}, PhysioClear, ""},
		{"K+ 5.8 no finerenone → not B-21", &RawPatientData{
			PotassiumCurrent: Float64Ptr(5.8),
			FinerenoneActive: false,
			GlucoseCurrent:   Float64Ptr(6.0),
			GlucoseTimestamp: time.Now(),
		}, PhysioClear, ""},
		{"finerenone but nil K+ → clear", &RawPatientData{
			FinerenoneActive: true,
			GlucoseCurrent:   Float64Ptr(6.0),
			GlucoseTimestamp: time.Now(),
		}, PhysioClear, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.Evaluate(tt.data)
			if result.Gate != tt.wantGate {
				t.Errorf("gate = %v, want %v", result.Gate, tt.wantGate)
			}
			if tt.wantRule != "" && result.RuleFired != tt.wantRule {
				t.Errorf("rule = %v, want %v", result.RuleFired, tt.wantRule)
			}
		})
	}
}

func TestB13_TakesPriorityOverB14(t *testing.T) {
	// B-13 (HR<45, HALT) fires before B-14 (HR<55, PAUSE) in evaluation order
	m := NewPhysiologySafetyMonitor(DefaultPhysioConfig())

	data := safeDefaults()
	data.HeartRateCurrent = Float64Ptr(40) // < 45 AND < 55
	data.HRContext = "RESTING"
	data.HeartRateConfirmed = true
	data.BetaBlockerActive = true
	data.BetaBlockerDoseChangeIn7d = true
	result := m.Evaluate(data)
	if result.RuleFired != "B-13" {
		t.Errorf("B-13 should fire before B-14 for HR 40, got rule=%s", result.RuleFired)
	}
}

// safeDefaults returns RawPatientData with all values in safe ranges.
func safeDefaults() *RawPatientData {
	return &RawPatientData{
		GlucoseCurrent:    Float64Ptr(6.5), // mmol/L (normal)
		GlucoseTimestamp:  time.Now(),
		CreatinineCurrent: Float64Ptr(80),  // µmol/L (normal)
		PotassiumCurrent:  Float64Ptr(4.5), // mEq/L (normal)
		SBPCurrent:        Float64Ptr(120), // mmHg (normal)
		WeightKgCurrent:   Float64Ptr(70),  // kg
		EGFRCurrent:       Float64Ptr(75),  // mL/min/1.73m² (normal)
		HbA1cCurrent:      Float64Ptr(7.0), // % (controlled diabetic)
	}
}
