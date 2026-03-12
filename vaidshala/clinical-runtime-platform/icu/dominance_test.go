package icu

import (
	"context"
	"testing"

	"vaidshala/clinical-runtime-platform/contracts"
)

// ═══════════════════════════════════════════════════════════════════════════════
// CLASSIFIER TESTS - Verify correct state classification
// ═══════════════════════════════════════════════════════════════════════════════

func TestClassifyDominanceState_NeurologicCollapse(t *testing.T) {
	engine := NewDominanceEngine(nil)

	tests := []struct {
		name     string
		facts    *SafetyFacts
		expected DominanceState
	}{
		{
			name: "GCS less than 8 triggers neurologic collapse",
			facts: &SafetyFacts{
				PatientID:   "test-patient",
				EncounterID: "test-encounter",
				GCS:         6,
				IsInICU:     true,
			},
			expected: StateNeurologicCollapse,
		},
		{
			name: "Active seizure triggers neurologic collapse",
			facts: &SafetyFacts{
				PatientID:        "test-patient",
				EncounterID:      "test-encounter",
				GCS:              12,
				HasActiveSeizure: true,
				IsInICU:          true,
			},
			expected: StateNeurologicCollapse,
		},
		{
			name: "ICP greater than 20 triggers neurologic collapse",
			facts: &SafetyFacts{
				PatientID:   "test-patient",
				EncounterID: "test-encounter",
				GCS:         14,
				ICP:         25,
				IsInICU:     true,
			},
			expected: StateNeurologicCollapse,
		},
		{
			name: "Herniation signs trigger neurologic collapse",
			facts: &SafetyFacts{
				PatientID:          "test-patient",
				EncounterID:        "test-encounter",
				GCS:                10,
				HasHerniationSigns: true,
				IsInICU:            true,
			},
			expected: StateNeurologicCollapse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.ClassifyDominanceState(tt.facts)
			if result != tt.expected {
				t.Errorf("ClassifyDominanceState() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestClassifyDominanceState_Shock(t *testing.T) {
	engine := NewDominanceEngine(nil)

	tests := []struct {
		name     string
		facts    *SafetyFacts
		expected DominanceState
	}{
		{
			name: "MAP less than 65 triggers shock",
			facts: &SafetyFacts{
				PatientID:   "test-patient",
				EncounterID: "test-encounter",
				GCS:         15, // Normal GCS to avoid neuro state
				MAP:         55,
				IsInICU:     true,
			},
			expected: StateShock,
		},
		{
			name: "Lactate greater than 4 triggers shock",
			facts: &SafetyFacts{
				PatientID:   "test-patient",
				EncounterID: "test-encounter",
				GCS:         15,
				MAP:         70,
				Lactate:     5.5,
				IsInICU:     true,
			},
			expected: StateShock,
		},
		{
			name: "On vasopressors triggers shock",
			facts: &SafetyFacts{
				PatientID:      "test-patient",
				EncounterID:    "test-encounter",
				GCS:            15,
				MAP:            68,
				OnVasopressors: true,
				IsInICU:        true,
			},
			expected: StateShock,
		},
		{
			name: "Septic shock flag triggers shock",
			facts: &SafetyFacts{
				PatientID:      "test-patient",
				EncounterID:    "test-encounter",
				GCS:            15,
				MAP:            70,
				HasSepticShock: true,
				IsInICU:        true,
			},
			expected: StateShock,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.ClassifyDominanceState(tt.facts)
			if result != tt.expected {
				t.Errorf("ClassifyDominanceState() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestClassifyDominanceState_Hypoxia(t *testing.T) {
	engine := NewDominanceEngine(nil)

	tests := []struct {
		name     string
		facts    *SafetyFacts
		expected DominanceState
	}{
		{
			name: "SpO2 less than 88 triggers hypoxia",
			facts: &SafetyFacts{
				PatientID:   "test-patient",
				EncounterID: "test-encounter",
				GCS:         15,
				MAP:         80,
				SpO2:        82,
				IsInICU:     true,
			},
			expected: StateHypoxia,
		},
		{
			name: "P/F ratio less than 100 triggers hypoxia",
			facts: &SafetyFacts{
				PatientID:   "test-patient",
				EncounterID: "test-encounter",
				GCS:         15,
				MAP:         80,
				SpO2:        92,
				PFRatio:     80,
				IsInICU:     true,
			},
			expected: StateHypoxia,
		},
		{
			name: "FiO2 greater than 0.6 triggers hypoxia",
			facts: &SafetyFacts{
				PatientID:   "test-patient",
				EncounterID: "test-encounter",
				GCS:         15,
				MAP:         80,
				SpO2:        94,
				PFRatio:     150,
				FiO2:        0.8,
				IsInICU:     true,
			},
			expected: StateHypoxia,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.ClassifyDominanceState(tt.facts)
			if result != tt.expected {
				t.Errorf("ClassifyDominanceState() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestClassifyDominanceState_ActiveBleed(t *testing.T) {
	engine := NewDominanceEngine(nil)

	tests := []struct {
		name     string
		facts    *SafetyFacts
		expected DominanceState
	}{
		{
			name: "Hgb drop greater than 2 triggers active bleed",
			facts: &SafetyFacts{
				PatientID:   "test-patient",
				EncounterID: "test-encounter",
				GCS:         15,
				MAP:         80,
				SpO2:        98,
				PFRatio:     400, // Normal - avoids hypoxia
				FiO2:        0.21,
				HgbDrop6h:   3.5,
				IsInICU:     true,
			},
			expected: StateActiveBleed,
		},
		{
			name: "Active transfusion triggers active bleed",
			facts: &SafetyFacts{
				PatientID:            "test-patient",
				EncounterID:          "test-encounter",
				GCS:                  15,
				MAP:                  80,
				SpO2:                 98,
				PFRatio:              400,
				FiO2:                 0.21,
				HasActiveTransfusion: true,
				IsInICU:              true,
			},
			expected: StateActiveBleed,
		},
		{
			name: "Surgical bleeding triggers active bleed",
			facts: &SafetyFacts{
				PatientID:           "test-patient",
				EncounterID:         "test-encounter",
				GCS:                 15,
				MAP:                 80,
				SpO2:                98,
				PFRatio:             400,
				FiO2:                0.21,
				HasSurgicalBleeding: true,
				IsInICU:             true,
			},
			expected: StateActiveBleed,
		},
		{
			name: "High INR with active bleeding triggers active bleed",
			facts: &SafetyFacts{
				PatientID:         "test-patient",
				EncounterID:       "test-encounter",
				GCS:               15,
				MAP:               80,
				SpO2:              98,
				PFRatio:           400,
				FiO2:              0.21,
				INR:               5.0,
				HasActiveBleeding: true,
				IsInICU:           true,
			},
			expected: StateActiveBleed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.ClassifyDominanceState(tt.facts)
			if result != tt.expected {
				t.Errorf("ClassifyDominanceState() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestClassifyDominanceState_LowOutputFailure(t *testing.T) {
	engine := NewDominanceEngine(nil)

	tests := []struct {
		name     string
		facts    *SafetyFacts
		expected DominanceState
	}{
		{
			name: "Cardiac index less than 2 triggers low output",
			facts: &SafetyFacts{
				PatientID:    "test-patient",
				EncounterID:  "test-encounter",
				GCS:          15,
				MAP:          80,
				SpO2:         98,
				PFRatio:      400, // Normal - avoids hypoxia
				FiO2:         0.21,
				CardiacIndex: 1.5,
				ScvO2:        65, // Normal ScvO2
				IsInICU:      true,
			},
			expected: StateLowOutputFailure,
		},
		{
			name: "ScvO2 less than 60 triggers low output",
			facts: &SafetyFacts{
				PatientID:    "test-patient",
				EncounterID:  "test-encounter",
				GCS:          15,
				MAP:          80,
				SpO2:         98,
				PFRatio:      400,
				FiO2:         0.21,
				CardiacIndex: 3.0,
				ScvO2:        50,
				IsInICU:      true,
			},
			expected: StateLowOutputFailure,
		},
		{
			name: "Inotrope escalation triggers low output",
			facts: &SafetyFacts{
				PatientID:            "test-patient",
				EncounterID:          "test-encounter",
				GCS:                  15,
				MAP:                  80,
				SpO2:                 98,
				PFRatio:              400,
				FiO2:                 0.21,
				CardiacIndex:         2.5,
				ScvO2:                65,
				OnInotropeEscalation: true,
				IsInICU:              true,
			},
			expected: StateLowOutputFailure,
		},
		{
			name: "Combined AKI and ALF triggers low output",
			facts: &SafetyFacts{
				PatientID:    "test-patient",
				EncounterID:  "test-encounter",
				GCS:          15,
				MAP:          80,
				SpO2:         98,
				PFRatio:      400,
				FiO2:         0.21,
				CardiacIndex: 2.5,
				ScvO2:        65,
				HasAKI:       true,
				HasALF:       true,
				IsInICU:      true,
			},
			expected: StateLowOutputFailure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.ClassifyDominanceState(tt.facts)
			if result != tt.expected {
				t.Errorf("ClassifyDominanceState() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestClassifyDominanceState_None(t *testing.T) {
	engine := NewDominanceEngine(nil)

	facts := NewSafetyFacts("test-patient", "test-encounter")
	facts.IsInICU = true

	result := engine.ClassifyDominanceState(facts)
	if result != StateNone {
		t.Errorf("ClassifyDominanceState() with normal facts = %v, want %v", result, StateNone)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// PRIORITY ORDER TEST - Verify neurologic beats shock beats hypoxia etc.
// ═══════════════════════════════════════════════════════════════════════════════

func TestClassifyDominanceState_PriorityOrder(t *testing.T) {
	engine := NewDominanceEngine(nil)

	// Patient with BOTH neurologic collapse AND shock symptoms
	// Neurologic should win (higher priority)
	facts := &SafetyFacts{
		PatientID:      "test-patient",
		EncounterID:    "test-encounter",
		GCS:            5,             // Neurologic collapse trigger
		MAP:            50,            // Shock trigger
		OnVasopressors: true,          // Shock trigger
		SpO2:           80,            // Hypoxia trigger
		HgbDrop6h:      4.0,           // Bleeding trigger
		CardiacIndex:   1.2,           // Low output trigger
		IsInICU:        true,
	}

	result := engine.ClassifyDominanceState(facts)
	if result != StateNeurologicCollapse {
		t.Errorf("Priority test: expected NEUROLOGIC_COLLAPSE (highest priority), got %v", result)
	}

	// Remove neurologic trigger - shock should win
	facts.GCS = 15
	result = engine.ClassifyDominanceState(facts)
	if result != StateShock {
		t.Errorf("Priority test: expected SHOCK (second priority), got %v", result)
	}

	// Remove shock triggers - hypoxia should win
	facts.MAP = 80
	facts.OnVasopressors = false
	result = engine.ClassifyDominanceState(facts)
	if result != StateHypoxia {
		t.Errorf("Priority test: expected HYPOXIA (third priority), got %v", result)
	}

	// Remove hypoxia triggers - bleeding should win
	facts.SpO2 = 98
	facts.PFRatio = 400 // Set normal P/F ratio
	facts.FiO2 = 0.21   // Set room air
	result = engine.ClassifyDominanceState(facts)
	if result != StateActiveBleed {
		t.Errorf("Priority test: expected ACTIVE_BLEED (fourth priority), got %v", result)
	}

	// Remove bleeding triggers - low output should win
	facts.HgbDrop6h = 0
	facts.ScvO2 = 50 // Set low ScvO2 to trigger low output
	result = engine.ClassifyDominanceState(facts)
	if result != StateLowOutputFailure {
		t.Errorf("Priority test: expected LOW_OUTPUT_FAILURE (fifth priority), got %v", result)
	}

	// Remove low output triggers - should be NONE
	facts.CardiacIndex = 3.0
	facts.ScvO2 = 70 // Reset to normal
	result = engine.ClassifyDominanceState(facts)
	if result != StateNone {
		t.Errorf("Priority test: expected NONE (no triggers), got %v", result)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// CONTEXT GATE TEST - Verify dominance only asserts in ICU context
// ═══════════════════════════════════════════════════════════════════════════════

func TestEvaluate_ContextGate(t *testing.T) {
	engine := NewDominanceEngine(nil)
	ctx := context.Background()

	action := contracts.ProposedAction{
		ID:          "test-action",
		Type:        contracts.ActionDischarge,
		Source:      "KB-19",
		PatientID:   "test-patient",
		EncounterID: "test-encounter",
	}

	// Patient in shock BUT not in ICU - should NOT veto
	facts := &SafetyFacts{
		PatientID:   "test-patient",
		EncounterID: "test-encounter",
		GCS:         15,
		MAP:         50, // Shock trigger
		IsInICU:     false, // NOT in ICU
		IsCodeActive: false,
		IsCriticallyUnstable: false,
	}

	result, err := engine.Evaluate(ctx, action, facts)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}

	// Should NOT be vetoed because not in ICU context
	if result.Vetoed {
		t.Errorf("Context gate failed: action vetoed outside ICU context")
	}
	if result.CurrentState != StateNone {
		t.Errorf("Context gate failed: expected StateNone outside ICU, got %v", result.CurrentState)
	}

	// Now set ICU context - should veto
	facts.IsInICU = true
	result, err = engine.Evaluate(ctx, action, facts)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}

	// Should be vetoed because in ICU and in shock
	if !result.Vetoed {
		t.Errorf("Evaluate should veto discharge in shock state")
	}
	if result.CurrentState != StateShock {
		t.Errorf("Expected StateShock, got %v", result.CurrentState)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// VETO TESTS - Verify specific actions are vetoed in specific states
// ═══════════════════════════════════════════════════════════════════════════════

func TestEvaluate_DischargeVetoedInAllActiveStates(t *testing.T) {
	engine := NewDominanceEngine(nil)
	ctx := context.Background()

	dischargeAction := contracts.ProposedAction{
		ID:          "discharge-1",
		Type:        contracts.ActionDischarge,
		Source:      "KB-19",
		PatientID:   "test-patient",
		EncounterID: "test-encounter",
	}

	// Test each active state vetoes discharge
	states := []struct {
		name  string
		facts *SafetyFacts
	}{
		{
			name: "Neurologic collapse vetoes discharge",
			facts: &SafetyFacts{
				PatientID: "test-patient", EncounterID: "test-encounter",
				GCS: 5, IsInICU: true,
			},
		},
		{
			name: "Shock vetoes discharge",
			facts: &SafetyFacts{
				PatientID: "test-patient", EncounterID: "test-encounter",
				GCS: 15, MAP: 50, IsInICU: true,
			},
		},
		{
			name: "Hypoxia vetoes discharge",
			facts: &SafetyFacts{
				PatientID: "test-patient", EncounterID: "test-encounter",
				GCS: 15, MAP: 80, SpO2: 80, IsInICU: true,
			},
		},
		{
			name: "Active bleed vetoes discharge",
			facts: &SafetyFacts{
				PatientID: "test-patient", EncounterID: "test-encounter",
				GCS: 15, MAP: 80, SpO2: 98, HgbDrop6h: 3.0, IsInICU: true,
			},
		},
		{
			name: "Low output vetoes discharge",
			facts: &SafetyFacts{
				PatientID: "test-patient", EncounterID: "test-encounter",
				GCS: 15, MAP: 80, SpO2: 98, CardiacIndex: 1.5, IsInICU: true,
			},
		},
	}

	for _, tt := range states {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Evaluate(ctx, dischargeAction, tt.facts)
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}
			if !result.Vetoed {
				t.Errorf("%s: expected discharge to be vetoed", tt.name)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// STATE HELPER TESTS
// ═══════════════════════════════════════════════════════════════════════════════

func TestDominanceState_Priority(t *testing.T) {
	// Verify priority order
	priorities := map[DominanceState]int{
		StateNeurologicCollapse: 6,
		StateShock:              5,
		StateHypoxia:            4,
		StateActiveBleed:        3,
		StateLowOutputFailure:   2,
		StateNone:               1,
	}

	for state, expectedPriority := range priorities {
		if state.Priority() != expectedPriority {
			t.Errorf("State %s priority = %d, want %d", state, state.Priority(), expectedPriority)
		}
	}
}

func TestDominanceState_IsActive(t *testing.T) {
	activeStates := []DominanceState{
		StateNeurologicCollapse,
		StateShock,
		StateHypoxia,
		StateActiveBleed,
		StateLowOutputFailure,
	}

	for _, state := range activeStates {
		if !state.IsActive() {
			t.Errorf("State %s should be active", state)
		}
	}

	if StateNone.IsActive() {
		t.Error("StateNone should not be active")
	}
}

func TestSafetyFacts_IsICUContext(t *testing.T) {
	tests := []struct {
		name     string
		facts    *SafetyFacts
		expected bool
	}{
		{
			name:     "IsInICU true",
			facts:    &SafetyFacts{IsInICU: true},
			expected: true,
		},
		{
			name:     "IsCodeActive true",
			facts:    &SafetyFacts{IsCodeActive: true},
			expected: true,
		},
		{
			name:     "IsCriticallyUnstable true",
			facts:    &SafetyFacts{IsCriticallyUnstable: true},
			expected: true,
		},
		{
			name:     "All false",
			facts:    &SafetyFacts{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.facts.IsICUContext() != tt.expected {
				t.Errorf("IsICUContext() = %v, want %v", tt.facts.IsICUContext(), tt.expected)
			}
		})
	}
}

func TestSafetyFacts_Validate(t *testing.T) {
	// Valid facts
	validFacts := NewSafetyFacts("patient-1", "encounter-1")
	if err := validFacts.Validate(); err != nil {
		t.Errorf("Valid facts should not return error: %v", err)
	}

	// Missing patient ID
	invalidFacts := &SafetyFacts{EncounterID: "encounter-1", GCS: 15}
	if err := invalidFacts.Validate(); err == nil {
		t.Error("Missing PatientID should return error")
	}

	// Missing encounter ID
	invalidFacts = &SafetyFacts{PatientID: "patient-1", GCS: 15}
	if err := invalidFacts.Validate(); err == nil {
		t.Error("Missing EncounterID should return error")
	}

	// Invalid GCS
	invalidFacts = &SafetyFacts{PatientID: "patient-1", EncounterID: "encounter-1", GCS: 2}
	if err := invalidFacts.Validate(); err == nil {
		t.Error("GCS < 3 should return error")
	}
}
