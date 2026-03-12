// Package cql provides CQL integration tests.
package cql

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	"vaidshala/clinical-runtime-platform/contracts"
)

// TestExecutorInitialization tests that the CQL executor can be created.
func TestExecutorInitialization(t *testing.T) {
	logger := zap.NewNop()

	config := ExecutorConfig{
		Region: "AU",
	}

	executor := NewExecutor(config, logger)

	if executor == nil {
		t.Fatal("Expected non-nil executor")
	}

	// Verify available facts
	facts := executor.GetAvailableFacts()
	if len(facts) == 0 {
		t.Error("Expected at least one available fact")
	}

	// Verify available measures
	measures := executor.GetAvailableMeasures()
	if len(measures) == 0 {
		t.Error("Expected at least one available measure")
	}

	t.Logf("Available facts: %v", facts)
	t.Logf("Available measures: %v", measures)
}

// TestExecutorEvaluate tests the CQL evaluation pipeline.
func TestExecutorEvaluate(t *testing.T) {
	logger := zap.NewNop()

	config := ExecutorConfig{
		Region: "AU",
	}

	executor := NewExecutor(config, logger)

	// Build a test execution context
	now := time.Now()
	birthDate := now.AddDate(-55, 0, 0) // 55 years old

	execCtx := &contracts.ClinicalExecutionContext{
		Patient: contracts.PatientContext{
			Demographics: contracts.PatientDemographics{
				PatientID: "test-patient-001",
				Gender:    "male",
				BirthDate: &birthDate,
			},
			ActiveConditions: []contracts.ClinicalCondition{
				{
					Code: contracts.ClinicalCode{
						System:  "http://snomed.info/sct",
						Code:    "73211009", // Diabetes mellitus
						Display: "Diabetes mellitus",
					},
					ClinicalStatus: "active",
				},
			},
			RecentLabResults: []contracts.LabResult{
				{
					Code: contracts.ClinicalCode{
						System:  "http://loinc.org",
						Code:    "4548-4", // HbA1c
						Display: "Hemoglobin A1c",
					},
					Value: &contracts.Quantity{
						Value: 8.5, // Controlled HbA1c
						Unit:  "%",
					},
				},
			},
		},
		Knowledge: contracts.KnowledgeSnapshot{
			Terminology: contracts.TerminologySnapshot{
				ValueSetMemberships: map[string]bool{
					"Diabetes": true,
				},
			},
			SnapshotTimestamp: now,
			SnapshotVersion:   "1.0.0",
		},
		Runtime: contracts.ExecutionMetadata{
			RequestID:   "test-request-001",
			RequestedBy: "test",
			RequestedAt: now,
			Region:      "AU",
		},
	}

	ctx := context.Background()
	result, err := executor.Evaluate(ctx, execCtx)

	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	t.Logf("Clinical facts: %d", len(result.ClinicalFacts))
	t.Logf("Measure results: %d", len(result.MeasureResults))
	t.Logf("Execution time: %dms", result.ExecutionTimeMs)

	// Log each clinical fact
	for _, fact := range result.ClinicalFacts {
		t.Logf("  Fact: %s = %v", fact.FactID, fact.Value)
	}

	// Log each measure result
	for _, mr := range result.MeasureResults {
		t.Logf("  Measure: %s - InDenom: %v, InNumer: %v, Gap: %v",
			mr.MeasureID, mr.InDenominator, mr.InNumerator, mr.CareGapIdentified)
	}
}

// TestCQLFactsForDiabetes tests that diabetes-related facts are correctly evaluated.
func TestCQLFactsForDiabetes(t *testing.T) {
	logger := zap.NewNop()

	executor := NewExecutor(ExecutorConfig{Region: "AU"}, logger)

	now := time.Now()
	birthDate := now.AddDate(-50, 0, 0)

	// Patient with diabetes and poor HbA1c control (>9%)
	execCtx := &contracts.ClinicalExecutionContext{
		Patient: contracts.PatientContext{
			Demographics: contracts.PatientDemographics{
				PatientID: "diabetes-patient",
				Gender:    "female",
				BirthDate: &birthDate,
			},
			ActiveConditions: []contracts.ClinicalCondition{
				{
					Code: contracts.ClinicalCode{
						System: "http://snomed.info/sct",
						Code:   "44054006", // Type 2 diabetes
					},
					ClinicalStatus: "active",
				},
			},
			RecentLabResults: []contracts.LabResult{
				{
					Code: contracts.ClinicalCode{
						System: "http://loinc.org",
						Code:   "4548-4",
					},
					Value: &contracts.Quantity{
						Value: 10.5, // Poor control >9%
						Unit:  "%",
					},
				},
			},
		},
		Knowledge: contracts.KnowledgeSnapshot{
			Terminology: contracts.TerminologySnapshot{
				ValueSetMemberships: map[string]bool{
					"Diabetes": true,
				},
			},
		},
		Runtime: contracts.ExecutionMetadata{
			RequestID: "test-diabetes",
			Region:    "AU",
		},
	}

	ctx := context.Background()
	facts, err := executor.EvaluateClinicalFacts(ctx, execCtx)

	if err != nil {
		t.Fatalf("Failed to evaluate facts: %v", err)
	}

	// Check for diabetes-related facts
	factMap := make(map[string]bool)
	for _, f := range facts {
		factMap[f.FactID] = f.Value
	}

	// Verify HasDiabetes fact
	if hasDiabetes, ok := factMap[contracts.FactHasDiabetes]; !ok || !hasDiabetes {
		t.Error("Expected FactHasDiabetes to be true")
	}

	// Verify HbA1cPoorControl fact (should be true since >9%)
	if poorControl, ok := factMap[contracts.FactHbA1cPoorControl]; !ok || !poorControl {
		t.Errorf("Expected FactHbA1cPoorControl to be true (HbA1c was 10.5%%)")
	}

	t.Logf("Evaluated %d facts for diabetes patient", len(facts))
	for _, f := range facts {
		t.Logf("  %s = %v", f.FactID, f.Value)
	}
}
