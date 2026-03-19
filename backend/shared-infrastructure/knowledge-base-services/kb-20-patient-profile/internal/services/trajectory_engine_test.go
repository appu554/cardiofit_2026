package services

import "testing"

func newEngine() *TrajectoryEngine {
	return NewTrajectoryEngine()
}

// ---------------------------------------------------------------------------
// 1. Green path: high adherence + stable/improving labs → GREEN
// ---------------------------------------------------------------------------

func TestTrajectory_GreenPath(t *testing.T) {
	e := newEngine()
	r := e.Classify(TrajectoryInput{
		ProtocolID:        "M3-PRP",
		CurrentPhase:      "STABILIZATION",
		DaysInPhase:       10,
		DaysSinceStart:    10,
		ProteinAdherence:  85,
		ExerciseAdherence: 80,
		MealQualityScore:  75,
		EGFRDelta:         2,  // improving
		FBGDelta:          -5, // improving
		HbA1cDelta:        -0.1,
		BMI:               28,
	})

	if r.Color != TrajectoryGreen {
		t.Errorf("expected GREEN, got %s (score=%.1f, reasons=%v)", r.Color, r.Score, r.Reasons)
	}
	if r.EscalationDue {
		t.Error("escalation should not be triggered on GREEN path")
	}
}

// ---------------------------------------------------------------------------
// 2. Yellow path: moderate adherence, stable labs → YELLOW
// ---------------------------------------------------------------------------

func TestTrajectory_YellowPath_LowAdherence(t *testing.T) {
	e := newEngine()
	r := e.Classify(TrajectoryInput{
		ProtocolID:        "M3-PRP",
		CurrentPhase:      "RESTORATION",
		DaysInPhase:       20,
		DaysSinceStart:    34,
		ProteinAdherence:  50,
		ExerciseAdherence: 45,
		MealQualityScore:  55,
		EGFRDelta:         0,
		FBGDelta:          0,
		HbA1cDelta:        0,
		BMI:               27,
	})

	if r.Color != TrajectoryYellow {
		t.Errorf("expected YELLOW for ~50%% adherence, got %s (score=%.1f)", r.Color, r.Score)
	}
}

// ---------------------------------------------------------------------------
// 3. Yellow path: good adherence but minor lab decline → YELLOW
// ---------------------------------------------------------------------------

func TestTrajectory_YellowPath_AdverseLabTrend(t *testing.T) {
	e := newEngine()
	r := e.Classify(TrajectoryInput{
		ProtocolID:        "M3-PRP",
		CurrentPhase:      "OPTIMIZATION",
		DaysInPhase:       14,
		DaysSinceStart:    56,
		ProteinAdherence:  75,
		ExerciseAdherence: 70,
		MealQualityScore:  72,
		EGFRDelta:         -3, // mild decline → 50
		FBGDelta:          8,  // mild rise → 50
		HbA1cDelta:        0.3, // mild rise → 50
		BMI:               26,
	})

	// Adherence ~72.3, lab ~50 → OPTIMIZATION weights 50/50 → ~61 → YELLOW
	if r.Color != TrajectoryYellow {
		t.Errorf("expected YELLOW for adverse lab trends, got %s (score=%.1f)", r.Color, r.Score)
	}
}

// ---------------------------------------------------------------------------
// 4. Red path: very low adherence → RED
// ---------------------------------------------------------------------------

func TestTrajectory_RedPath_VeryLowAdherence(t *testing.T) {
	e := newEngine()
	r := e.Classify(TrajectoryInput{
		ProtocolID:        "M3-PRP",
		CurrentPhase:      "STABILIZATION",
		DaysInPhase:       12,
		DaysSinceStart:    12,
		ProteinAdherence:  15,
		ExerciseAdherence: 10,
		MealQualityScore:  8,
		EGFRDelta:         -3, // mild decline → 50
		FBGDelta:          5,  // mild rise → 50
		HbA1cDelta:        0.2, // mild rise → 50
		BMI:               30,
	})

	if r.Color != TrajectoryRed {
		t.Errorf("expected RED for ~15%% adherence, got %s (score=%.1f)", r.Color, r.Score)
	}
}

// ---------------------------------------------------------------------------
// 5. Red path: critical eGFR decline → safety override to RED
// ---------------------------------------------------------------------------

func TestTrajectory_RedPath_CriticalLabDecline(t *testing.T) {
	e := newEngine()
	r := e.Classify(TrajectoryInput{
		ProtocolID:        "M3-PRP",
		CurrentPhase:      "RESTORATION",
		DaysInPhase:       25,
		DaysSinceStart:    39,
		ProteinAdherence:  90,
		ExerciseAdherence: 85,
		MealQualityScore:  80,
		EGFRDelta:         -7, // critical decline
		FBGDelta:          0,
		HbA1cDelta:        0,
		BMI:               27,
	})

	if r.Color != TrajectoryRed {
		t.Errorf("expected RED for eGFR delta -7, got %s", r.Color)
	}

	hasOverride := false
	for _, reason := range r.Reasons {
		if contains(reason, "safety_override") && contains(reason, "eGFR") {
			hasOverride = true
			break
		}
	}
	if !hasOverride {
		t.Errorf("expected eGFR safety override reason, got reasons=%v", r.Reasons)
	}
}

// ---------------------------------------------------------------------------
// 6. Red + day 63 → EscalationDue = true
// ---------------------------------------------------------------------------

func TestTrajectory_RedPath_EscalationAtDay63(t *testing.T) {
	e := newEngine()
	r := e.Classify(TrajectoryInput{
		ProtocolID:        "M3-PRP",
		CurrentPhase:      "OPTIMIZATION",
		DaysInPhase:       21,
		DaysSinceStart:    63,
		ProteinAdherence:  10,
		ExerciseAdherence: 8,
		MealQualityScore:  5,
		EGFRDelta:         -3, // mild decline → 50
		FBGDelta:          8,  // mild rise → 50
		HbA1cDelta:        0.3, // mild rise → 50
		BMI:               29,
	})

	if r.Color != TrajectoryRed {
		t.Errorf("expected RED, got %s (score=%.1f)", r.Color, r.Score)
	}
	if !r.EscalationDue {
		t.Error("expected EscalationDue=true at day 63 with RED trajectory")
	}
}

// ---------------------------------------------------------------------------
// 7. VFRP weight safety: excessive loss + low BMI → RED
// ---------------------------------------------------------------------------

func TestTrajectory_VFRP_WeightSafety(t *testing.T) {
	e := newEngine()
	r := e.Classify(TrajectoryInput{
		ProtocolID:        "M3-VFRP",
		CurrentPhase:      "FAT_MOBILIZATION",
		DaysInPhase:       20,
		DaysSinceStart:    34,
		ExerciseAdherence: 90,
		MealQualityScore:  85,
		TrigDelta:         -5, // improving
		WaistDeltaCm:      -3, // improving
		WeightDeltaKg:     4,  // excessive loss
		BMI:               23, // below 25 threshold
	})

	if r.Color != TrajectoryRed {
		t.Errorf("expected RED for weight safety override, got %s", r.Color)
	}

	hasOverride := false
	for _, reason := range r.Reasons {
		if contains(reason, "weight_loss") {
			hasOverride = true
			break
		}
	}
	if !hasOverride {
		t.Errorf("expected weight safety override reason, got reasons=%v", r.Reasons)
	}
}

// ---------------------------------------------------------------------------
// 8. Composite: one GREEN + one RED → composite RED
// ---------------------------------------------------------------------------

func TestTrajectory_CompositeWorstColor(t *testing.T) {
	e := newEngine()
	c := e.ClassifyAll([]TrajectoryInput{
		{
			ProtocolID:        "M3-PRP",
			CurrentPhase:      "STABILIZATION",
			DaysSinceStart:    10,
			ProteinAdherence:  90,
			ExerciseAdherence: 85,
			MealQualityScore:  80,
			EGFRDelta:         1,
			FBGDelta:          -2,
			HbA1cDelta:        -0.1,
			BMI:               28,
		},
		{
			ProtocolID:        "M3-VFRP",
			CurrentPhase:      "SUSTAINED_REDUCTION",
			DaysSinceStart:    65,
			ExerciseAdherence: 10,
			MealQualityScore:  5,
			TrigDelta:         30, // critical
			WaistDeltaCm:      2,  // worsening
			BMI:               32,
		},
	})

	if c.PatientColor != TrajectoryRed {
		t.Errorf("expected composite RED (worst-of), got %s", c.PatientColor)
	}
	if len(c.Protocols) != 2 {
		t.Fatalf("expected 2 protocol results, got %d", len(c.Protocols))
	}

	// First should be green, second red
	if c.Protocols[0].Color != TrajectoryGreen {
		t.Errorf("expected PRP GREEN, got %s", c.Protocols[0].Color)
	}
	if c.Protocols[1].Color != TrajectoryRed {
		t.Errorf("expected VFRP RED, got %s (score=%.1f)", c.Protocols[1].Color, c.Protocols[1].Score)
	}
}

// ---------------------------------------------------------------------------
// 9. PRP phase weighting: early phase weights adherence more
// ---------------------------------------------------------------------------

func TestTrajectory_PRP_PhaseWeighting(t *testing.T) {
	e := newEngine()

	base := TrajectoryInput{
		ProtocolID:        "M3-PRP",
		DaysSinceStart:    10,
		ProteinAdherence:  80,
		ExerciseAdherence: 75,
		MealQualityScore:  70,
		EGFRDelta:         -3, // mild decline → lab=50
		FBGDelta:          5,  // mild rise → lab=50
		HbA1cDelta:        0.2, // mild rise → lab=50
		BMI:               27,
	}

	// Early phase: 70% adherence weight, 30% lab weight
	base.CurrentPhase = "STABILIZATION"
	earlyResult := e.Classify(base)

	// Late phase: 50% adherence weight, 50% lab weight
	base.CurrentPhase = "OPTIMIZATION"
	lateResult := e.Classify(base)

	// With adherence ~75 and labs ~50:
	// Early: 0.70*75 + 0.30*50 = 52.5 + 15.0 = 67.5
	// Late:  0.50*75 + 0.50*50 = 37.5 + 25.0 = 62.5
	// Early should score higher (adherence weighted more, and adherence is higher than labs)
	if earlyResult.Score <= lateResult.Score {
		t.Errorf("expected early phase score > late phase score (early=%.1f, late=%.1f)", earlyResult.Score, lateResult.Score)
	}
}

// ---------------------------------------------------------------------------
// 10. All GREEN composite → composite GREEN, no escalation
// ---------------------------------------------------------------------------

func TestTrajectory_AllGreenComposite(t *testing.T) {
	e := newEngine()
	c := e.ClassifyAll([]TrajectoryInput{
		{
			ProtocolID:        "M3-PRP",
			CurrentPhase:      "STABILIZATION",
			DaysSinceStart:    7,
			ProteinAdherence:  90,
			ExerciseAdherence: 85,
			MealQualityScore:  80,
			EGFRDelta:         1,
			FBGDelta:          -3,
			HbA1cDelta:        -0.2,
			BMI:               28,
		},
		{
			ProtocolID:        "M3-VFRP",
			CurrentPhase:      "METABOLIC_STABILIZATION",
			DaysSinceStart:    7,
			ExerciseAdherence: 80,
			MealQualityScore:  75,
			TrigDelta:         -10,
			WaistDeltaCm:      -3,
			BMI:               30,
		},
	})

	if c.PatientColor != TrajectoryGreen {
		t.Errorf("expected composite GREEN, got %s", c.PatientColor)
	}
	if c.AnyEscalation {
		t.Error("expected no escalation for all-GREEN composite")
	}
	for i, p := range c.Protocols {
		if p.Color != TrajectoryGreen {
			t.Errorf("protocol[%d] expected GREEN, got %s", i, p.Color)
		}
	}
}

// ---------------------------------------------------------------------------
// 11. Empty inputs → GREEN composite (no protocols = no risk)
// ---------------------------------------------------------------------------

func TestTrajectory_EmptyInputs(t *testing.T) {
	e := newEngine()
	c := e.ClassifyAll(nil)
	if c.PatientColor != TrajectoryGreen {
		t.Errorf("expected GREEN for empty inputs, got %s", c.PatientColor)
	}
	if len(c.Protocols) != 0 {
		t.Errorf("expected 0 protocols, got %d", len(c.Protocols))
	}
}

// ---------------------------------------------------------------------------
// 12. RED before day 63 → no escalation
// ---------------------------------------------------------------------------

func TestTrajectory_RedBeforeDay63_NoEscalation(t *testing.T) {
	e := newEngine()
	r := e.Classify(TrajectoryInput{
		ProtocolID:        "M3-PRP",
		CurrentPhase:      "RESTORATION",
		DaysInPhase:       20,
		DaysSinceStart:    34,
		ProteinAdherence:  10,
		ExerciseAdherence: 5,
		MealQualityScore:  8,
		EGFRDelta:         0,
		FBGDelta:          0,
		HbA1cDelta:        0,
		BMI:               29,
	})

	if r.Color != TrajectoryRed {
		t.Errorf("expected RED, got %s", r.Color)
	}
	if r.EscalationDue {
		t.Error("escalation should NOT trigger before day 63")
	}
}

// ---------------------------------------------------------------------------
// 13. VFRP green path: good adherence + improving metrics → GREEN
// ---------------------------------------------------------------------------

func TestTrajectory_VFRP_GreenPath(t *testing.T) {
	e := newEngine()
	r := e.Classify(TrajectoryInput{
		ProtocolID:        "M3-VFRP",
		CurrentPhase:      "FAT_MOBILIZATION",
		DaysInPhase:       20,
		DaysSinceStart:    34,
		ExerciseAdherence: 85,
		MealQualityScore:  80,
		TrigDelta:         -15,
		WaistDeltaCm:      -4,
		WeightDeltaKg:     1.5,
		BMI:               30,
	})

	if r.Color != TrajectoryGreen {
		t.Errorf("expected GREEN for VFRP good path, got %s (score=%.1f)", r.Color, r.Score)
	}
}

// ---------------------------------------------------------------------------
// 14. MRI forcing rules (Spec §7)
// ---------------------------------------------------------------------------

func TestApplyMRIForcing(t *testing.T) {
	t.Run("MRI >75 forces RED", func(t *testing.T) {
		color := applyMRIForcing(TrajectoryGreen, 78.0, 0)
		if color != TrajectoryRed {
			t.Errorf("expected RED for MRI=78, got %s", color)
		}
	})
	t.Run("MRI worsening >10 in 14d forces YELLOW", func(t *testing.T) {
		color := applyMRIForcing(TrajectoryGreen, 55.0, 12.0)
		if color != TrajectoryYellow {
			t.Errorf("expected YELLOW for MRI delta=+12, got %s", color)
		}
	})
	t.Run("MRI moderate does not force GREEN to RED", func(t *testing.T) {
		color := applyMRIForcing(TrajectoryGreen, 60.0, 5.0)
		if color != TrajectoryGreen {
			t.Errorf("expected GREEN unchanged for MRI=60 delta=5, got %s", color)
		}
	})
	t.Run("MRI forcing does not downgrade existing RED", func(t *testing.T) {
		color := applyMRIForcing(TrajectoryRed, 40.0, 0)
		if color != TrajectoryRed {
			t.Errorf("expected RED to stay RED, got %s", color)
		}
	})
	t.Run("MRI worsening does not downgrade RED to YELLOW", func(t *testing.T) {
		color := applyMRIForcing(TrajectoryRed, 55.0, 12.0)
		if color != TrajectoryRed {
			t.Errorf("expected RED to stay RED when delta>10 but already RED, got %s", color)
		}
	})
	t.Run("MRI exactly 75 does not force RED (boundary)", func(t *testing.T) {
		color := applyMRIForcing(TrajectoryGreen, 75.0, 0)
		if color != TrajectoryGreen {
			t.Errorf("expected GREEN unchanged for MRI=75 (boundary, not >75), got %s", color)
		}
	})
}

// ---------------------------------------------------------------------------
// 15. MRI forcing integration via Classify
// ---------------------------------------------------------------------------

func TestTrajectory_MRIForcing_IntegratesIntoClassify(t *testing.T) {
	e := newEngine()

	t.Run("MRI >75 escalates GREEN to RED via Classify", func(t *testing.T) {
		r := e.Classify(TrajectoryInput{
			ProtocolID:        "M3-PRP",
			CurrentPhase:      "STABILIZATION",
			DaysSinceStart:    10,
			ProteinAdherence:  90,
			ExerciseAdherence: 85,
			MealQualityScore:  80,
			EGFRDelta:         1,
			FBGDelta:          -2,
			HbA1cDelta:        -0.1,
			BMI:               28,
			MRIScore:          80.0,
			MRIDelta14d:       2.0,
		})
		if r.Color != TrajectoryRed {
			t.Errorf("expected RED via MRI forcing (score=80), got %s", r.Color)
		}
		hasMRIReason := false
		for _, reason := range r.Reasons {
			if contains(reason, "MRI forcing") {
				hasMRIReason = true
				break
			}
		}
		if !hasMRIReason {
			t.Errorf("expected MRI forcing reason in Reasons, got %v", r.Reasons)
		}
	})

	t.Run("No MRI data (score=0) does not affect result", func(t *testing.T) {
		r := e.Classify(TrajectoryInput{
			ProtocolID:        "M3-PRP",
			CurrentPhase:      "STABILIZATION",
			DaysSinceStart:    10,
			ProteinAdherence:  90,
			ExerciseAdherence: 85,
			MealQualityScore:  80,
			EGFRDelta:         1,
			FBGDelta:          -2,
			HbA1cDelta:        -0.1,
			BMI:               28,
			MRIScore:          0, // not available
			MRIDelta14d:       0,
		})
		if r.Color != TrajectoryGreen {
			t.Errorf("expected GREEN when MRI not provided, got %s", r.Color)
		}
	})
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
