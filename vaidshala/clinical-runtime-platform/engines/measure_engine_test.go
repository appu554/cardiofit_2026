package engines

import (
	"context"
	"testing"
	"time"

	"vaidshala/clinical-runtime-platform/contracts"
	"vaidshala/clinical-runtime-platform/engines/measures"
)

// ============================================================================
// MEASURE ENGINE INTEGRATION TESTS
// These tests verify the MeasureEngine correctly wraps evaluators and
// integrates with the Engine interface used by EngineOrchestrator.
// ============================================================================

func TestMeasureEngine_Name(t *testing.T) {
	engine := NewMeasureEngine(DefaultMeasureEngineConfig())

	if engine.Name() != "measure-engine" {
		t.Errorf("Name(): got %q, want %q", engine.Name(), "measure-engine")
	}
}

func TestMeasureEngine_AvailableMeasures(t *testing.T) {
	engine := NewMeasureEngine(DefaultMeasureEngineConfig())

	measures := engine.AvailableMeasures()

	// Should have all 4 CMS measures registered
	if len(measures) != 4 {
		t.Errorf("AvailableMeasures(): got %d measures, want 4", len(measures))
	}

	// Verify each measure is registered
	expectedMeasures := []string{"CMS122", "CMS165", "CMS134", "CMS2"}
	for _, expected := range expectedMeasures {
		found := false
		for _, m := range measures {
			if m == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("AvailableMeasures(): missing %s", expected)
		}
	}
}

func TestMeasureEngine_GetEvaluator(t *testing.T) {
	engine := NewMeasureEngine(DefaultMeasureEngineConfig())

	tests := []struct {
		measureID string
		exists    bool
	}{
		{"CMS122", true},
		{"CMS165", true},
		{"CMS134", true},
		{"CMS2", true},
		{"CMS999", false}, // Unknown measure
	}

	for _, tc := range tests {
		t.Run(tc.measureID, func(t *testing.T) {
			eval, exists := engine.GetEvaluator(tc.measureID)
			if exists != tc.exists {
				t.Errorf("GetEvaluator(%s): exists=%v, want %v", tc.measureID, exists, tc.exists)
			}
			if tc.exists && eval == nil {
				t.Errorf("GetEvaluator(%s): evaluator is nil", tc.measureID)
			}
		})
	}
}

func TestMeasureEngine_Evaluate_DiabeticPatient(t *testing.T) {
	engine := NewMeasureEngine(DefaultMeasureEngineConfig())

	// Create a diabetic patient context with HbA1c 10.2% (poor control)
	now := time.Now()
	birthDate := now.AddDate(-55, 0, 0) // 55 years old
	labTime := now.AddDate(0, -1, 0)    // 1 month ago

	ctx := &contracts.ClinicalExecutionContext{
		Patient: contracts.PatientContext{
			Demographics: contracts.PatientDemographics{
				PatientID: "test-diabetic-patient",
				BirthDate: &birthDate,
				Gender:    "female",
			},
			RecentLabResults: []contracts.LabResult{
				{
					Code: contracts.ClinicalCode{
						System:  "http://loinc.org",
						Code:    measures.LoincHbA1c,
						Display: "Hemoglobin A1c",
					},
					Value: &contracts.Quantity{
						Value: 10.2, // Poor control (>9%)
						Unit:  "%",
					},
					EffectiveDateTime: &labTime,
				},
			},
			RecentEncounters: []contracts.Encounter{
				{EncounterID: "enc-1", Status: "finished"},
			},
		},
		Knowledge: contracts.KnowledgeSnapshot{
			Terminology: contracts.TerminologySnapshot{
				ValueSetMemberships: map[string]bool{
					"HasDiabetes":     true,
					"HasHypertension": false,
				},
			},
			SnapshotTimestamp: now,
		},
		Runtime: contracts.ExecutionMetadata{
			RequestID:        "test-request-1",
			RequestedEngines: []string{"CMS122"}, // Only run CMS122
		},
	}

	result, err := engine.Evaluate(context.Background(), ctx)

	if err != nil {
		t.Fatalf("Evaluate() error: %v", err)
	}

	// Verify engine result structure
	if result.EngineName != "measure-engine" {
		t.Errorf("EngineName: got %q, want %q", result.EngineName, "measure-engine")
	}
	if !result.Success {
		t.Errorf("Success: got %v, want true", result.Success)
	}

	// Should have 1 measure result (CMS122 only)
	if len(result.MeasureResults) != 1 {
		t.Fatalf("MeasureResults: got %d, want 1", len(result.MeasureResults))
	}

	// Verify CMS122 result
	cms122Result := result.MeasureResults[0]
	if cms122Result.MeasureID != "CMS122" {
		t.Errorf("MeasureID: got %q, want %q", cms122Result.MeasureID, "CMS122")
	}
	if !cms122Result.InInitialPopulation {
		t.Error("InInitialPopulation: expected true (diabetic patient)")
	}
	if !cms122Result.InNumerator {
		t.Error("InNumerator: expected true (HbA1c 10.2% > 9%)")
	}
	if !cms122Result.CareGapIdentified {
		t.Error("CareGapIdentified: expected true (inverse measure)")
	}

	// Verify care gap recommendation was generated
	if len(result.Recommendations) != 1 {
		t.Errorf("Recommendations: got %d, want 1 (care gap)", len(result.Recommendations))
	}

	t.Logf("✅ MeasureEngine integration test PASSED: HbA1c 10.2%% → CareGap=%v",
		cms122Result.CareGapIdentified)
}

func TestMeasureEngine_Evaluate_AllMeasures(t *testing.T) {
	engine := NewMeasureEngine(DefaultMeasureEngineConfig())

	// Create patient context that qualifies for multiple measures
	now := time.Now()
	birthDate := now.AddDate(-50, 0, 0) // 50 years old
	labTime := now.AddDate(0, -1, 0)

	ctx := &contracts.ClinicalExecutionContext{
		Patient: contracts.PatientContext{
			Demographics: contracts.PatientDemographics{
				PatientID: "test-multi-measure-patient",
				BirthDate: &birthDate,
				Gender:    "male",
			},
			RecentLabResults: []contracts.LabResult{
				{
					Code: contracts.ClinicalCode{
						System: "http://loinc.org",
						Code:   measures.LoincHbA1c,
					},
					Value: &contracts.Quantity{
						Value: 7.5, // Good control
						Unit:  "%",
					},
					EffectiveDateTime: &labTime,
				},
			},
			RecentVitalSigns: []contracts.VitalSign{
				{
					ComponentValues: []contracts.ComponentValue{
						{
							Code:  contracts.ClinicalCode{Code: measures.LoincSystolicBP},
							Value: &contracts.Quantity{Value: 130.0},
						},
						{
							Code:  contracts.ClinicalCode{Code: measures.LoincDiastolicBP},
							Value: &contracts.Quantity{Value: 85.0},
						},
					},
				},
			},
			RecentEncounters: []contracts.Encounter{
				{EncounterID: "enc-1", Status: "finished"},
			},
		},
		Knowledge: contracts.KnowledgeSnapshot{
			Terminology: contracts.TerminologySnapshot{
				ValueSetMemberships: map[string]bool{
					"HasDiabetes":                  true,
					"HasHypertension":              true,
					"DepressionScreeningNegative": true,
				},
			},
			SnapshotTimestamp: now,
		},
		Runtime: contracts.ExecutionMetadata{
			RequestID: "test-all-measures",
			// No RequestedEngines = run all default measures
		},
	}

	result, err := engine.Evaluate(context.Background(), ctx)

	if err != nil {
		t.Fatalf("Evaluate() error: %v", err)
	}

	// Should have results for all 4 default measures
	if len(result.MeasureResults) != 4 {
		t.Errorf("MeasureResults: got %d, want 4", len(result.MeasureResults))
	}

	// Verify each measure has required audit fields
	for _, mr := range result.MeasureResults {
		t.Run(mr.MeasureID, func(t *testing.T) {
			if mr.MeasureVersion == "" {
				t.Error("MeasureVersion is empty")
			}
			if mr.LogicVersion == "" {
				t.Error("LogicVersion is empty")
			}
			if mr.ELMCorrespondence == "" {
				t.Error("ELMCorrespondence is empty")
			}
			if mr.EvaluatedAt.IsZero() {
				t.Error("EvaluatedAt is zero")
			}
			if mr.Rationale == "" {
				t.Error("Rationale is empty")
			}
		})
	}

	t.Logf("✅ MeasureEngine evaluated %d measures successfully", len(result.MeasureResults))
}

// Engine interface mirrors factory.Engine for compile-time verification.
// This avoids circular imports while proving MeasureEngine implements the contract.
type Engine interface {
	Name() string
	Evaluate(ctx context.Context, execCtx *contracts.ClinicalExecutionContext) (*contracts.EngineResult, error)
}

func TestMeasureEngine_ImplementsEngineInterface(t *testing.T) {
	// Verify MeasureEngine implements the Engine interface
	engine := NewMeasureEngine(DefaultMeasureEngineConfig())

	// This is a compile-time check - if it compiles, it implements Engine
	var _ Engine = engine

	t.Log("✅ MeasureEngine implements Engine interface")
}

func TestMeasureEngine_PureFunctionProperty(t *testing.T) {
	// Verify determinism: same input → same output
	engine := NewMeasureEngine(DefaultMeasureEngineConfig())

	now := time.Now()
	birthDate := now.AddDate(-45, 0, 0)
	labTime := now.AddDate(0, -1, 0)

	ctx := &contracts.ClinicalExecutionContext{
		Patient: contracts.PatientContext{
			Demographics: contracts.PatientDemographics{
				PatientID: "determinism-test",
				BirthDate: &birthDate,
				Gender:    "female",
			},
			RecentLabResults: []contracts.LabResult{
				{
					Code:              contracts.ClinicalCode{Code: measures.LoincHbA1c},
					Value:             &contracts.Quantity{Value: 8.5},
					EffectiveDateTime: &labTime,
				},
			},
			RecentEncounters: []contracts.Encounter{
				{EncounterID: "enc-1", Status: "finished"},
			},
		},
		Knowledge: contracts.KnowledgeSnapshot{
			Terminology: contracts.TerminologySnapshot{
				ValueSetMemberships: map[string]bool{
					"HasDiabetes": true,
				},
			},
		},
		Runtime: contracts.ExecutionMetadata{
			RequestedEngines: []string{"CMS122"},
		},
	}

	// Run twice
	result1, _ := engine.Evaluate(context.Background(), ctx)
	result2, _ := engine.Evaluate(context.Background(), ctx)

	// Compare measure outcomes (not timestamps)
	if len(result1.MeasureResults) != len(result2.MeasureResults) {
		t.Fatal("Different number of results - not deterministic")
	}

	mr1 := result1.MeasureResults[0]
	mr2 := result2.MeasureResults[0]

	if mr1.InInitialPopulation != mr2.InInitialPopulation ||
		mr1.InNumerator != mr2.InNumerator ||
		mr1.CareGapIdentified != mr2.CareGapIdentified {
		t.Error("Different outcomes - not deterministic")
	}

	t.Log("✅ MeasureEngine is deterministic (pure function property verified)")
}
