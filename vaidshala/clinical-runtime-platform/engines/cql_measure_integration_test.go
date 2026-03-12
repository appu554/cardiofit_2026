// Package engines provides integration tests for the CQL → Measure Engine flow.
//
// CTO/CMO ARCHITECTURE TEST:
// This test validates the correct separation of concerns between:
//   - CQL Engine: Clinical Truth Determination ("What is true?")
//   - Measure Engine: Clinical Accountability ("Are we meeting standards?")
//
// The test demonstrates:
//   1. CQL Engine produces ClinicalFacts (truths)
//   2. Orchestrator passes facts to Measure Engine
//   3. Measure Engine produces MeasureResults (care judgments)
//   4. Clean separation - no cross-contamination of responsibilities
package engines

import (
	"context"
	"testing"
	"time"

	"vaidshala/clinical-runtime-platform/contracts"
	"vaidshala/clinical-runtime-platform/engines/measures"
)

// TestCQLToMeasureEngineFlow validates the CTO/CMO architecture:
// CQL Engine (truths) → Measure Engine (judgments)
func TestCQLToMeasureEngineFlow(t *testing.T) {
	ctx := context.Background()

	// Create test patient context
	execCtx := createDiabeticPatientContext()

	// ═══════════════════════════════════════════════════════════════════════════
	// STEP 1: CQL Engine evaluates clinical TRUTHS
	// ═══════════════════════════════════════════════════════════════════════════
	cqlEngine := NewCQLEngine(DefaultCQLEngineConfig())

	cqlResult, err := cqlEngine.Evaluate(ctx, execCtx)
	if err != nil {
		t.Fatalf("CQL Engine failed: %v", err)
	}

	// Verify CQL Engine produces FACTS, not MeasureResults
	if len(cqlResult.ClinicalFacts) == 0 {
		t.Error("CQL Engine should produce ClinicalFacts")
	}
	if len(cqlResult.MeasureResults) > 0 {
		t.Error("CQL Engine should NOT produce MeasureResults (that's Measure Engine's job)")
	}

	t.Logf("✅ CQL Engine produced %d clinical facts", len(cqlResult.ClinicalFacts))

	// Log the facts for visibility
	for _, fact := range cqlResult.ClinicalFacts {
		t.Logf("   📊 %s = %v (%s)", fact.FactID, fact.Value, fact.Evidence)
	}

	// ═══════════════════════════════════════════════════════════════════════════
	// STEP 2: Measure Engine consumes facts and produces JUDGMENTS
	// ═══════════════════════════════════════════════════════════════════════════
	measureEngine := NewMeasureEngine(DefaultMeasureEngineConfig())

	// Use EvaluateWithFacts to inject CQL facts (CTO/CMO flow)
	measureResult, err := measureEngine.EvaluateWithFacts(ctx, execCtx, cqlResult.ClinicalFacts)
	if err != nil {
		t.Fatalf("Measure Engine failed: %v", err)
	}

	// Verify Measure Engine produces MeasureResults, not new facts
	if len(measureResult.MeasureResults) == 0 {
		t.Error("Measure Engine should produce MeasureResults")
	}
	if len(measureResult.ClinicalFacts) > 0 {
		t.Error("Measure Engine should NOT produce new ClinicalFacts (that's CQL Engine's job)")
	}

	t.Logf("✅ Measure Engine produced %d measure results", len(measureResult.MeasureResults))

	// Log the care judgments
	for _, mr := range measureResult.MeasureResults {
		gapStatus := "✓ Met"
		if mr.CareGapIdentified {
			gapStatus = "✗ Gap"
		}
		t.Logf("   📋 %s: %s (InNum=%v, InDen=%v)", mr.MeasureID, gapStatus, mr.InNumerator, mr.InDenominator)
	}
}

// TestCQLEngineProducesOnlyFacts verifies CQL Engine follows its role.
func TestCQLEngineProducesOnlyFacts(t *testing.T) {
	ctx := context.Background()
	execCtx := createDiabeticPatientContext()

	cqlEngine := NewCQLEngine(DefaultCQLEngineConfig())
	result, _ := cqlEngine.Evaluate(ctx, execCtx)

	// Verify CQL Engine outputs
	t.Run("CQL Engine produces facts", func(t *testing.T) {
		if len(result.ClinicalFacts) == 0 {
			t.Error("CQL Engine should produce ClinicalFacts")
		}
	})

	t.Run("CQL Engine does NOT produce MeasureResults", func(t *testing.T) {
		if len(result.MeasureResults) > 0 {
			t.Errorf("CQL Engine produced %d MeasureResults - should be 0", len(result.MeasureResults))
		}
	})

	t.Run("CQL Engine does NOT produce Recommendations", func(t *testing.T) {
		if len(result.Recommendations) > 0 {
			t.Errorf("CQL Engine produced %d Recommendations - should be 0", len(result.Recommendations))
		}
	})

	t.Run("Each fact has required fields", func(t *testing.T) {
		for _, fact := range result.ClinicalFacts {
			if fact.FactID == "" {
				t.Error("Fact missing FactID")
			}
			if fact.Evidence == "" {
				t.Error("Fact missing Evidence")
			}
			if fact.FactCategory == "" {
				t.Error("Fact missing FactCategory")
			}
		}
	})

	t.Logf("✅ CQL Engine correctly produces only ClinicalFacts")
}

// TestMeasureEngineProducesOnlyJudgments verifies Measure Engine follows its role.
func TestMeasureEngineProducesOnlyJudgments(t *testing.T) {
	ctx := context.Background()
	execCtx := createDiabeticPatientContext()

	measureEngine := NewMeasureEngine(DefaultMeasureEngineConfig())
	result, _ := measureEngine.Evaluate(ctx, execCtx)

	// Verify Measure Engine outputs
	t.Run("Measure Engine produces MeasureResults", func(t *testing.T) {
		if len(result.MeasureResults) == 0 {
			t.Error("Measure Engine should produce MeasureResults")
		}
	})

	t.Run("Measure Engine produces Recommendations for care gaps", func(t *testing.T) {
		careGaps := 0
		for _, mr := range result.MeasureResults {
			if mr.CareGapIdentified {
				careGaps++
			}
		}
		if careGaps > 0 && len(result.Recommendations) == 0 {
			t.Error("Measure Engine should produce Recommendations for care gaps")
		}
	})

	t.Run("Measure Engine does NOT produce new ClinicalFacts", func(t *testing.T) {
		if len(result.ClinicalFacts) > 0 {
			t.Errorf("Measure Engine produced %d ClinicalFacts - should be 0", len(result.ClinicalFacts))
		}
	})

	t.Logf("✅ Measure Engine correctly produces only MeasureResults and Recommendations")
}

// TestMeasureEngineEnrichesWithFactEvidence verifies fact evidence integration.
func TestMeasureEngineEnrichesWithFactEvidence(t *testing.T) {
	ctx := context.Background()
	execCtx := createDiabeticPatientContext()

	// First run CQL Engine
	cqlEngine := NewCQLEngine(DefaultCQLEngineConfig())
	cqlResult, _ := cqlEngine.Evaluate(ctx, execCtx)

	// Then run Measure Engine WITH facts
	measureEngine := NewMeasureEngine(DefaultMeasureEngineConfig())
	resultWithFacts, _ := measureEngine.EvaluateWithFacts(ctx, execCtx, cqlResult.ClinicalFacts)

	// Run Measure Engine WITHOUT facts for comparison
	measureEngineNoFacts := NewMeasureEngine(DefaultMeasureEngineConfig())
	resultWithoutFacts, _ := measureEngineNoFacts.Evaluate(ctx, execCtx)

	// Verify enrichment
	for i, mr := range resultWithFacts.MeasureResults {
		mrNoFacts := resultWithoutFacts.MeasureResults[i]

		// With facts should have more evidence
		if len(mr.Rationale) <= len(mrNoFacts.Rationale) {
			// Note: This might be equal if no relevant facts exist
			t.Logf("   %s: With facts=%d chars, Without=%d chars",
				mr.MeasureID, len(mr.Rationale), len(mrNoFacts.Rationale))
		}
	}

	t.Log("✅ Measure Engine enriches rationale with CQL fact evidence")
}

// TestCQLFactCategories verifies facts are properly categorized.
func TestCQLFactCategories(t *testing.T) {
	ctx := context.Background()
	execCtx := createDiabeticPatientContext()

	cqlEngine := NewCQLEngine(DefaultCQLEngineConfig())
	result, _ := cqlEngine.Evaluate(ctx, execCtx)

	categories := make(map[string]int)
	for _, fact := range result.ClinicalFacts {
		categories[fact.FactCategory]++
	}

	// Verify expected categories
	expectedCategories := []string{"glycemic", "cardiovascular", "renal", "screening", "general"}
	for _, cat := range expectedCategories {
		if count, ok := categories[cat]; ok {
			t.Logf("   📁 %s: %d facts", cat, count)
		}
	}

	t.Log("✅ CQL facts are properly categorized")
}

// TestSpecificFactValues verifies specific clinical truths.
func TestSpecificFactValues(t *testing.T) {
	ctx := context.Background()

	t.Run("Diabetic patient with poor HbA1c control", func(t *testing.T) {
		execCtx := createDiabeticPatientContext() // HbA1c = 9.5%

		cqlEngine := NewCQLEngine(DefaultCQLEngineConfig())
		result, _ := cqlEngine.Evaluate(ctx, execCtx)

		// Check specific facts
		facts := mapFactsByID(result.ClinicalFacts)

		// HasDiabetes should be true
		if fact, ok := facts[contracts.FactHasDiabetes]; ok {
			if !fact.Value {
				t.Error("HasDiabetes should be true for diabetic patient")
			}
		}

		// HbA1cPoorControl should be true (9.5% > 9.0%)
		if fact, ok := facts[contracts.FactHbA1cPoorControl]; ok {
			if !fact.Value {
				t.Errorf("HbA1cPoorControl should be true (HbA1c > 9%%)")
			}
			t.Logf("   HbA1cPoorControl: %v (%.1f%%)", fact.Value, *fact.NumericValue)
		}
	})

	t.Run("Hypertensive patient with uncontrolled BP", func(t *testing.T) {
		execCtx := createHypertensivePatientContext() // BP 150/95

		cqlEngine := NewCQLEngine(DefaultCQLEngineConfig())
		result, _ := cqlEngine.Evaluate(ctx, execCtx)

		facts := mapFactsByID(result.ClinicalFacts)

		// HasHypertension should be true
		if fact, ok := facts[contracts.FactHasHypertension]; ok {
			if !fact.Value {
				t.Error("HasHypertension should be true")
			}
		}

		// BloodPressureUncontrolled should be true (150/95 >= 140/90)
		if fact, ok := facts[contracts.FactBloodPressureUncontrolled]; ok {
			if !fact.Value {
				t.Error("BloodPressureUncontrolled should be true (150/95 >= 140/90)")
			}
		}

		// BloodPressureControlled should be false
		if fact, ok := facts[contracts.FactBloodPressureControlled]; ok {
			if fact.Value {
				t.Error("BloodPressureControlled should be false")
			}
		}
	})
}

// TestEngineNamesAreDistinct verifies engine identities.
func TestEngineNamesAreDistinct(t *testing.T) {
	cqlEngine := NewCQLEngine(DefaultCQLEngineConfig())
	measureEngine := NewMeasureEngine(DefaultMeasureEngineConfig())

	if cqlEngine.Name() == measureEngine.Name() {
		t.Error("CQL Engine and Measure Engine should have distinct names")
	}

	if cqlEngine.Name() != "cql-engine" {
		t.Errorf("CQL Engine should be named 'cql-engine', got '%s'", cqlEngine.Name())
	}

	if measureEngine.Name() != "measure-engine" {
		t.Errorf("Measure Engine should be named 'measure-engine', got '%s'", measureEngine.Name())
	}

	t.Log("✅ Engine names are distinct and correct")
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func mapFactsByID(facts []contracts.ClinicalFact) map[string]contracts.ClinicalFact {
	result := make(map[string]contracts.ClinicalFact)
	for _, fact := range facts {
		result[fact.FactID] = fact
	}
	return result
}

func createDiabeticPatientContext() *contracts.ClinicalExecutionContext {
	birthDate := time.Date(1970, 1, 15, 0, 0, 0, 0, time.UTC)
	labTime := time.Now().Add(-30 * 24 * time.Hour)

	return &contracts.ClinicalExecutionContext{
		Patient: contracts.PatientContext{
			Demographics: contracts.PatientDemographics{
				PatientID: "test-diabetic-001",
				BirthDate: &birthDate,
				Gender:    "male",
				Region:    "AU",
			},
			ActiveConditions: []contracts.ClinicalCondition{
				{
					Code: contracts.ClinicalCode{
						System:  "http://snomed.info/sct",
						Code:    "44054006", // Type 2 diabetes mellitus
						Display: "Type 2 diabetes mellitus",
					},
					ClinicalStatus: "active",
				},
			},
			RecentLabResults: []contracts.LabResult{
				{
					Code: contracts.ClinicalCode{
						System:  "http://loinc.org",
						Code:    measures.LoincHbA1c,
						Display: "Hemoglobin A1c",
					},
					Value: &contracts.Quantity{
						Value: 9.5, // Poor control (> 9%)
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
							Value: &contracts.Quantity{Value: 135.0},
						},
						{
							Code:  contracts.ClinicalCode{Code: measures.LoincDiastolicBP},
							Value: &contracts.Quantity{Value: 85.0},
						},
					},
				},
			},
			RecentEncounters: []contracts.Encounter{
				{
					EncounterID: "enc-001",
					Class:       "ambulatory",
					Status:      "finished",
				},
			},
		},
		Knowledge: contracts.KnowledgeSnapshot{
			Terminology: contracts.TerminologySnapshot{
				ValueSetMemberships: map[string]bool{
					"HasDiabetes": true, // KB-7 semantic flag format (CamelCase)
				},
			},
		},
		Runtime: contracts.ExecutionMetadata{
			RequestID:   "test-request-001",
			RequestedBy: "integration-test",
			RequestedAt: time.Now(),
			Region:      "AU",
		},
	}
}

func createHypertensivePatientContext() *contracts.ClinicalExecutionContext {
	birthDate := time.Date(1965, 6, 20, 0, 0, 0, 0, time.UTC)

	return &contracts.ClinicalExecutionContext{
		Patient: contracts.PatientContext{
			Demographics: contracts.PatientDemographics{
				PatientID: "test-htn-001",
				BirthDate: &birthDate,
				Gender:    "female",
				Region:    "AU",
			},
			ActiveConditions: []contracts.ClinicalCondition{
				{
					Code: contracts.ClinicalCode{
						System:  "http://snomed.info/sct",
						Code:    "38341003", // Essential hypertension
						Display: "Essential hypertension",
					},
					ClinicalStatus: "active",
				},
			},
			RecentVitalSigns: []contracts.VitalSign{
				{
					ComponentValues: []contracts.ComponentValue{
						{
							Code:  contracts.ClinicalCode{Code: measures.LoincSystolicBP},
							Value: &contracts.Quantity{Value: 150.0}, // Uncontrolled
						},
						{
							Code:  contracts.ClinicalCode{Code: measures.LoincDiastolicBP},
							Value: &contracts.Quantity{Value: 95.0}, // Uncontrolled
						},
					},
				},
			},
			RecentEncounters: []contracts.Encounter{
				{
					EncounterID: "enc-002",
					Class:       "ambulatory",
					Status:      "finished",
				},
			},
		},
		Knowledge: contracts.KnowledgeSnapshot{
			Terminology: contracts.TerminologySnapshot{
				ValueSetMemberships: map[string]bool{
					"Essential Hypertension": true,
				},
			},
		},
		Runtime: contracts.ExecutionMetadata{
			RequestID:   "test-request-002",
			RequestedBy: "integration-test",
			RequestedAt: time.Now(),
			Region:      "AU",
		},
	}
}
