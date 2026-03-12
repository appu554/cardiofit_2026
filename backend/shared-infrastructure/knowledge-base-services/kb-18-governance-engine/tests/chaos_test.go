// Package tests provides clinical-device rigor testing for KB-18 Governance Engine.
// This file tests CHAOS/RESILIENCE scenarios - graceful degradation and error handling.
package tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"kb-18-governance-engine/pkg/engine"
	"kb-18-governance-engine/pkg/programs"
	"kb-18-governance-engine/pkg/types"
)

// =============================================================================
// CHAOS TESTS - Resilience and Error Handling
// =============================================================================

// TestChaos_NilPatientContext verifies graceful handling of nil patient context
func TestChaos_NilPatientContext(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)
	ctx := context.Background()

	req := &types.EvaluationRequest{
		PatientID:      "PT-NIL-CTX",
		PatientContext: nil, // Nil context
		EvaluationType: types.EvalTypeMedicationOrder,
		RequestorID:    "DR-001",
		Timestamp:      time.Now(),
	}

	resp, err := eng.Evaluate(ctx, req)

	// Should not panic, should handle gracefully
	if err != nil {
		t.Logf("Engine returned error for nil context: %v", err)
	} else if resp != nil {
		t.Logf("Engine handled nil context gracefully: outcome=%s", resp.Outcome)
	}

	t.Logf("✅ NIL PATIENT CONTEXT: Handled gracefully (no panic)")
}

// TestChaos_NilMedicationOrder verifies graceful handling of nil medication order
func TestChaos_NilMedicationOrder(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)
	ctx := context.Background()

	req := &types.EvaluationRequest{
		PatientID: "PT-NIL-ORDER",
		PatientContext: &types.PatientContext{
			PatientID: "PT-NIL-ORDER",
			Age:       45,
			Sex:       "M",
		},
		Order: nil, // Nil order
		EvaluationType:  types.EvalTypeMedicationOrder,
		RequestorID:     "DR-001",
		Timestamp:       time.Now(),
	}

	resp, err := eng.Evaluate(ctx, req)

	if err != nil {
		t.Logf("Engine returned error for nil order: %v", err)
	} else if resp != nil {
		t.Logf("Engine handled nil order: outcome=%s", resp.Outcome)
	}

	t.Logf("✅ NIL MEDICATION ORDER: Handled gracefully (no panic)")
}

// TestChaos_EmptyPatientID verifies handling of empty patient ID
func TestChaos_EmptyPatientID(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)
	ctx := context.Background()

	req := &types.EvaluationRequest{
		PatientID: "", // Empty
		PatientContext: &types.PatientContext{
			PatientID: "",
			Age:       45,
			Sex:       "M",
		},
		EvaluationType: types.EvalTypeMedicationOrder,
		RequestorID:    "DR-001",
		Timestamp:      time.Now(),
	}

	resp, err := eng.Evaluate(ctx, req)

	if err != nil {
		t.Logf("Engine returned error for empty patient ID: %v", err)
	} else if resp != nil {
		t.Logf("Engine processed empty patient ID: outcome=%s", resp.Outcome)
	}

	t.Logf("✅ EMPTY PATIENT ID: Handled gracefully")
}

// TestChaos_NegativeAge verifies handling of invalid age
func TestChaos_NegativeAge(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)
	ctx := context.Background()

	req := &types.EvaluationRequest{
		PatientID: "PT-NEG-AGE",
		PatientContext: &types.PatientContext{
			PatientID: "PT-NEG-AGE",
			Age:       -5, // Negative age
			Sex:       "M",
		},
		Order: &types.MedicationOrder{
			MedicationCode: "TEST",
			MedicationName: "Test Drug",
			Dose:           10.0,
			DoseUnit:       "mg",
		},
		EvaluationType: types.EvalTypeMedicationOrder,
		RequestorID:    "DR-001",
		Timestamp:      time.Now(),
	}

	resp, err := eng.Evaluate(ctx, req)

	if err != nil {
		t.Logf("Engine returned error for negative age: %v", err)
	} else if resp != nil {
		t.Logf("Engine processed negative age: outcome=%s", resp.Outcome)
	}

	t.Logf("✅ NEGATIVE AGE: Handled without panic")
}

// TestChaos_ExtremeAge verifies handling of extreme ages
func TestChaos_ExtremeAge(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)
	ctx := context.Background()

	extremeAges := []int{0, 1, 150, 999, -1}

	for _, age := range extremeAges {
		req := &types.EvaluationRequest{
			PatientID: "PT-EXTREME-AGE",
			PatientContext: &types.PatientContext{
				PatientID: "PT-EXTREME-AGE",
				Age:       age,
				Sex:       "M",
			},
			EvaluationType: types.EvalTypeMedicationOrder,
			RequestorID:    "DR-001",
			Timestamp:      time.Now(),
		}

		_, err := eng.Evaluate(ctx, req)
		if err != nil {
			t.Logf("Age %d: error returned", age)
		} else {
			t.Logf("Age %d: processed successfully", age)
		}
	}

	t.Logf("✅ EXTREME AGES: All handled without panic")
}

// TestChaos_MalformedMedicationOrder verifies handling of malformed orders
func TestChaos_MalformedMedicationOrder(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)
	ctx := context.Background()

	malformedOrders := []*types.MedicationOrder{
		// Empty medication code
		{MedicationCode: "", MedicationName: "Test", Dose: 10.0, DoseUnit: "mg"},
		// Zero dose
		{MedicationCode: "TEST", MedicationName: "Test", Dose: 0.0, DoseUnit: "mg"},
		// Negative dose
		{MedicationCode: "TEST", MedicationName: "Test", Dose: -10.0, DoseUnit: "mg"},
		// Missing dose unit
		{MedicationCode: "TEST", MedicationName: "Test", Dose: 10.0, DoseUnit: ""},
		// Extreme dose
		{MedicationCode: "TEST", MedicationName: "Test", Dose: 999999999.0, DoseUnit: "mg"},
	}

	for i, order := range malformedOrders {
		req := &types.EvaluationRequest{
			PatientID: "PT-MALFORMED",
			PatientContext: &types.PatientContext{
				PatientID: "PT-MALFORMED",
				Age:       45,
				Sex:       "M",
			},
			Order: order,
			EvaluationType:  types.EvalTypeMedicationOrder,
			RequestorID:     "DR-001",
			Timestamp:       time.Now(),
		}

		resp, err := eng.Evaluate(ctx, req)
		if err != nil {
			t.Logf("Malformed order %d: error returned", i)
		} else if resp != nil {
			t.Logf("Malformed order %d: processed (outcome=%s)", i, resp.Outcome)
		}
	}

	t.Logf("✅ MALFORMED ORDERS: All handled without panic")
}

// TestChaos_ContextCancellation verifies graceful handling of cancelled context
func TestChaos_ContextCancellation(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)

	// Create context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := &types.EvaluationRequest{
		PatientID: "PT-CANCELLED",
		PatientContext: &types.PatientContext{
			PatientID: "PT-CANCELLED",
			Age:       45,
			Sex:       "M",
		},
		EvaluationType: types.EvalTypeMedicationOrder,
		RequestorID:    "DR-001",
		Timestamp:      time.Now(),
	}

	resp, err := eng.Evaluate(ctx, req)

	// Should handle cancelled context gracefully
	if err != nil {
		t.Logf("Cancelled context: error=%v", err)
	} else if resp != nil {
		t.Logf("Cancelled context: processed (may not respect cancellation)")
	}

	t.Logf("✅ CONTEXT CANCELLATION: Handled without panic")
}

// TestChaos_ContextTimeout verifies handling of timed out context
func TestChaos_ContextTimeout(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for timeout
	time.Sleep(1 * time.Millisecond)

	req := &types.EvaluationRequest{
		PatientID: "PT-TIMEOUT",
		PatientContext: &types.PatientContext{
			PatientID: "PT-TIMEOUT",
			Age:       45,
			Sex:       "M",
		},
		EvaluationType: types.EvalTypeMedicationOrder,
		RequestorID:    "DR-001",
		Timestamp:      time.Now(),
	}

	resp, err := eng.Evaluate(ctx, req)

	if err != nil {
		t.Logf("Timed out context: error=%v", err)
	} else if resp != nil {
		t.Logf("Timed out context: processed")
	}

	t.Logf("✅ CONTEXT TIMEOUT: Handled without panic")
}

// TestChaos_InvalidEnforcementLevel verifies handling of unknown enforcement level
func TestChaos_InvalidEnforcementLevel(t *testing.T) {
	// Test unknown enforcement level handling
	unknownLevel := types.EnforcementLevel("UNKNOWN_LEVEL")

	// Priority should return -1 or similar for unknown
	priority := types.GetEnforcementPriority(unknownLevel)
	t.Logf("Unknown enforcement level priority: %d", priority)

	// IsBlocking should not panic
	blocking := unknownLevel.IsBlocking()
	t.Logf("Unknown level IsBlocking: %v", blocking)

	// CanOverride should not panic
	canOverride := unknownLevel.CanOverride()
	t.Logf("Unknown level CanOverride: %v", canOverride)

	t.Logf("✅ INVALID ENFORCEMENT LEVEL: Handled without panic")
}

// TestChaos_InvalidSeverity verifies handling of unknown severity
func TestChaos_InvalidSeverity(t *testing.T) {
	unknownSeverity := types.Severity("UNKNOWN_SEVERITY")

	// Priority should return -1 or similar for unknown
	priority := types.GetSeverityPriority(unknownSeverity)
	t.Logf("Unknown severity priority: %d", priority)

	t.Logf("✅ INVALID SEVERITY: Handled without panic")
}

// TestChaos_LargePatientHistory verifies handling of patients with extensive history
func TestChaos_LargePatientHistory(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)
	ctx := context.Background()

	// Create patient with large history
	var diagnoses []types.Diagnosis
	for i := 0; i < 100; i++ {
		diagnoses = append(diagnoses, types.Diagnosis{
			Code:        "E11.9",
			CodeSystem:  "ICD10",
			Description: "Test Diagnosis",
			Status:      "active",
		})
	}

	var medications []types.Medication
	for i := 0; i < 50; i++ {
		medications = append(medications, types.Medication{
			Code:      "TEST",
			Name:      "Test Med",
			DrugClass: "TEST_CLASS",
			Dose:      10.0,
			DoseUnit:  "mg",
		})
	}

	var labs []types.LabResult
	for i := 0; i < 100; i++ {
		labs = append(labs, types.LabResult{
			Code:      "2160-0",
			Name:      "Creatinine",
			Value:     1.0,
			Unit:      "mg/dL",
			Timestamp: time.Now().Add(-time.Duration(i) * time.Hour),
		})
	}

	req := &types.EvaluationRequest{
		PatientID: "PT-LARGE-HISTORY",
		PatientContext: &types.PatientContext{
			PatientID:          "PT-LARGE-HISTORY",
			Age:                75,
			Sex:                "M",
			ActiveDiagnoses:    diagnoses,
			CurrentMedications: medications,
			RecentLabs:         labs,
		},
		Order: &types.MedicationOrder{
			MedicationCode: "TEST",
			MedicationName: "Test Drug",
			Dose:           10.0,
			DoseUnit:       "mg",
		},
		EvaluationType: types.EvalTypeMedicationOrder,
		RequestorID:    "DR-001",
		Timestamp:      time.Now(),
	}

	start := time.Now()
	resp, err := eng.Evaluate(ctx, req)
	elapsed := time.Since(start)

	if err != nil {
		t.Logf("Large history: error=%v", err)
	} else if resp != nil {
		t.Logf("Large history: processed in %v", elapsed)
	}

	// Should still complete in reasonable time
	if elapsed > 5*time.Second {
		t.Errorf("Large history took too long: %v", elapsed)
	}

	t.Logf("✅ LARGE PATIENT HISTORY: Handled in %v", elapsed)
}

// TestChaos_EvidenceTrailJSONSerialization verifies evidence trail survives serialization
func TestChaos_EvidenceTrailJSONSerialization(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)
	ctx := context.Background()

	req := &types.EvaluationRequest{
		PatientID: "PT-SERIALIZE",
		PatientContext: &types.PatientContext{
			PatientID:  "PT-SERIALIZE",
			Age:        45,
			Sex:        "M",
			IsPregnant: false,
		},
		Order: &types.MedicationOrder{
			MedicationCode: "TEST",
			MedicationName: "Test Drug",
			Dose:           10.0,
			DoseUnit:       "mg",
		},
		EvaluationType: types.EvalTypeMedicationOrder,
		RequestorID:    "DR-001",
		Timestamp:      time.Now(),
	}

	resp, err := eng.Evaluate(ctx, req)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	// Serialize to JSON
	jsonBytes, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("JSON serialization failed: %v", err)
	}

	// Deserialize back
	var deserialized types.EvaluationResponse
	if err := json.Unmarshal(jsonBytes, &deserialized); err != nil {
		t.Fatalf("JSON deserialization failed: %v", err)
	}

	// Verify key fields survived
	if deserialized.RequestID != resp.RequestID {
		t.Errorf("RequestID mismatch after serialization")
	}
	if deserialized.Outcome != resp.Outcome {
		t.Errorf("Outcome mismatch after serialization")
	}
	if deserialized.EvidenceTrail != nil && resp.EvidenceTrail != nil {
		if deserialized.EvidenceTrail.Hash != resp.EvidenceTrail.Hash {
			t.Errorf("Hash mismatch after serialization")
		}
	}

	t.Logf("✅ JSON SERIALIZATION: Evidence trail survives round-trip")
}

// TestChaos_SpecialCharactersInFields verifies handling of special characters
func TestChaos_SpecialCharactersInFields(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)
	ctx := context.Background()

	specialStrings := []string{
		"Patient <script>alert('xss')</script>",
		"Patient'; DROP TABLE patients;--",
		"Patient\x00WithNull",
		"Patient\nWith\nNewlines",
		"Patient with émojis 🏥💊",
		"",
		"   ",
		"\t\t\t",
	}

	for _, patientID := range specialStrings {
		req := &types.EvaluationRequest{
			PatientID: patientID,
			PatientContext: &types.PatientContext{
				PatientID: patientID,
				Age:       45,
				Sex:       "M",
			},
			EvaluationType: types.EvalTypeMedicationOrder,
			RequestorID:    patientID, // Also test in requestor
			Timestamp:      time.Now(),
		}

		resp, err := eng.Evaluate(ctx, req)
		if err != nil {
			t.Logf("Special chars '%s': error", patientID[:min(20, len(patientID))])
		} else if resp != nil {
			t.Logf("Special chars '%s': OK", patientID[:min(20, len(patientID))])
		}
	}

	t.Logf("✅ SPECIAL CHARACTERS: All handled without panic")
}

// TestChaos_RecoveryFromPanic verifies engine doesn't leave corrupted state after issues
func TestChaos_RecoveryFromPanic(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)
	ctx := context.Background()

	// Run problematic requests
	for i := 0; i < 5; i++ {
		req := &types.EvaluationRequest{
			PatientID:       "",
			PatientContext:  nil,
			Order: nil,
			EvaluationType:  "",
			RequestorID:     "",
			Timestamp:       time.Time{}, // Zero time
		}
		eng.Evaluate(ctx, req)
	}

	// Engine should still work for valid requests
	validReq := &types.EvaluationRequest{
		PatientID: "PT-VALID",
		PatientContext: &types.PatientContext{
			PatientID: "PT-VALID",
			Age:       45,
			Sex:       "M",
		},
		EvaluationType: types.EvalTypeMedicationOrder,
		RequestorID:    "DR-001",
		Timestamp:      time.Now(),
	}

	resp, err := eng.Evaluate(ctx, validReq)
	if err != nil {
		t.Errorf("Engine broken after chaos: %v", err)
	}
	if resp == nil {
		t.Error("Engine returned nil response after chaos")
	}

	// Verify stats are still working
	stats := eng.GetStats()
	if stats.TotalEvaluations == 0 {
		t.Logf("Note: Stats may have reset")
	}

	t.Logf("✅ RECOVERY: Engine still functional after chaos tests")
}
