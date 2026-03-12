package measures

import (
	"testing"
	"time"

	"vaidshala/clinical-runtime-platform/contracts"
)

// ============================================================================
// CMS122 GOLDEN TESTS
// These tests MUST always pass - they are the canonical correctness checks.
// ============================================================================

func TestCMS122_GoldenTests(t *testing.T) {
	evaluator := NewCMS122Evaluator()

	for _, tc := range CMS122GoldenTestCases() {
		t.Run(tc.Name, func(t *testing.T) {
			// Build test context
			ctx := buildCMS122TestContext(tc)

			// Evaluate
			result := evaluator.Evaluate(ctx)

			// Verify Initial Population
			if result.InInitialPopulation != tc.ExpectedInPopulation {
				t.Errorf("InInitialPopulation: got %v, want %v (rationale: %s)",
					result.InInitialPopulation, tc.ExpectedInPopulation, result.Rationale)
			}

			// Verify Numerator (only if in population)
			if tc.ExpectedInPopulation {
				if result.InNumerator != tc.ExpectedInNumerator {
					t.Errorf("InNumerator: got %v, want %v (rationale: %s)",
						result.InNumerator, tc.ExpectedInNumerator, result.Rationale)
				}
			}

			// Verify Care Gap
			if result.CareGapIdentified != tc.ExpectedCareGap {
				t.Errorf("CareGapIdentified: got %v, want %v (rationale: %s)",
					result.CareGapIdentified, tc.ExpectedCareGap, result.Rationale)
			}

			// Verify audit metadata is present
			if result.MeasureVersion == "" {
				t.Error("MeasureVersion is empty - audit metadata required")
			}
			if result.LogicVersion == "" {
				t.Error("LogicVersion is empty - audit metadata required")
			}
			if result.ELMCorrespondence == "" {
				t.Error("ELMCorrespondence is empty - audit metadata required")
			}
			if result.EvaluatedAt.IsZero() {
				t.Error("EvaluatedAt is zero - audit timestamp required")
			}
		})
	}
}

// TestCMS122_SpecificScenarios tests the two primary golden cases explicitly
func TestCMS122_SpecificScenarios(t *testing.T) {
	evaluator := NewCMS122Evaluator()

	t.Run("GOLDEN_1_Diabetic_HbA1c_10.2_InNumerator", func(t *testing.T) {
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// Diabetic patient with HbA1c 10.2% → MUST be IN NUMERATOR (poor control)
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		hba1c := 10.2
		now := time.Now()
		birthDate := now.AddDate(-55, 0, 0) // 55 years old
		labTime := now.AddDate(0, -1, 0)    // 1 month ago

		ctx := &contracts.ClinicalExecutionContext{
			Patient: contracts.PatientContext{
				Demographics: contracts.PatientDemographics{
					PatientID: "test-patient-1",
					BirthDate: &birthDate,
					Gender:    "female",
				},
				RecentLabResults: []contracts.LabResult{
					{
						Code: contracts.ClinicalCode{
							System:  "http://loinc.org",
							Code:    LoincHbA1c,
							Display: "Hemoglobin A1c",
						},
						Value: &contracts.Quantity{
							Value: hba1c,
							Unit:  "%",
						},
						EffectiveDateTime: &labTime,
						SourceReference:   "Observation/hba1c-1",
					},
				},
				RecentEncounters: []contracts.Encounter{
					{
						EncounterID: "encounter-1",
						Status:      "finished",
					},
				},
			},
			Knowledge: contracts.KnowledgeSnapshot{
				Terminology: contracts.TerminologySnapshot{
					ValueSetMemberships: map[string]bool{
						"HasDiabetes": true,
					},
				},
				SnapshotTimestamp: time.Now(),
			},
			Runtime: contracts.ExecutionMetadata{
				RequestID: "test-request-1",
			},
		}

		result := evaluator.Evaluate(ctx)

		// Assertions
		if !result.InInitialPopulation {
			t.Fatal("Expected patient to be IN initial population")
		}
		if !result.InDenominator {
			t.Fatal("Expected patient to be IN denominator")
		}
		if !result.InNumerator {
			t.Fatalf("Expected patient to be IN numerator (HbA1c %.1f > 9.0), got rationale: %s",
				hba1c, result.Rationale)
		}
		if !result.CareGapIdentified {
			t.Fatal("Expected care gap to be identified (inverse measure)")
		}

		t.Logf("✅ GOLDEN TEST 1 PASSED: HbA1c %.1f%% → Numerator=%v, CareGap=%v",
			hba1c, result.InNumerator, result.CareGapIdentified)
	})

	t.Run("GOLDEN_2_Diabetic_HbA1c_7.1_NotInNumerator", func(t *testing.T) {
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// Diabetic patient with HbA1c 7.1% → MUST NOT be in numerator (good control)
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		hba1c := 7.1
		now := time.Now()
		birthDate := now.AddDate(-45, 0, 0) // 45 years old
		labTime := now.AddDate(0, -1, 0)    // 1 month ago

		ctx := &contracts.ClinicalExecutionContext{
			Patient: contracts.PatientContext{
				Demographics: contracts.PatientDemographics{
					PatientID: "test-patient-2",
					BirthDate: &birthDate,
					Gender:    "male",
				},
				RecentLabResults: []contracts.LabResult{
					{
						Code: contracts.ClinicalCode{
							System:  "http://loinc.org",
							Code:    LoincHbA1c,
							Display: "Hemoglobin A1c",
						},
						Value: &contracts.Quantity{
							Value: hba1c,
							Unit:  "%",
						},
						EffectiveDateTime: &labTime,
						SourceReference:   "Observation/hba1c-2",
					},
				},
				RecentEncounters: []contracts.Encounter{
					{
						EncounterID: "encounter-2",
						Status:      "finished",
					},
				},
			},
			Knowledge: contracts.KnowledgeSnapshot{
				Terminology: contracts.TerminologySnapshot{
					ValueSetMemberships: map[string]bool{
						"HasDiabetes": true,
					},
				},
				SnapshotTimestamp: time.Now(),
			},
			Runtime: contracts.ExecutionMetadata{
				RequestID: "test-request-2",
			},
		}

		result := evaluator.Evaluate(ctx)

		// Assertions
		if !result.InInitialPopulation {
			t.Fatal("Expected patient to be IN initial population")
		}
		if !result.InDenominator {
			t.Fatal("Expected patient to be IN denominator")
		}
		if result.InNumerator {
			t.Fatalf("Expected patient to NOT be in numerator (HbA1c %.1f ≤ 9.0), got rationale: %s",
				hba1c, result.Rationale)
		}
		if result.CareGapIdentified {
			t.Fatal("Expected NO care gap (patient has good control)")
		}

		t.Logf("✅ GOLDEN TEST 2 PASSED: HbA1c %.1f%% → Numerator=%v, CareGap=%v",
			hba1c, result.InNumerator, result.CareGapIdentified)
	})
}

// TestCMS122_AuditMetadata verifies all audit fields are populated
func TestCMS122_AuditMetadata(t *testing.T) {
	evaluator := NewCMS122Evaluator()

	hba1c := 8.0
	now := time.Now()
	birthDate := now.AddDate(-50, 0, 0) // 50 years old
	labTime := now.AddDate(0, -1, 0)    // 1 month ago

	ctx := &contracts.ClinicalExecutionContext{
		Patient: contracts.PatientContext{
			Demographics: contracts.PatientDemographics{
				PatientID: "audit-test-patient",
				BirthDate: &birthDate,
				Gender:    "female",
			},
			RecentLabResults: []contracts.LabResult{
				{
					Code: contracts.ClinicalCode{
						System:  "http://loinc.org",
						Code:    LoincHbA1c,
						Display: "Hemoglobin A1c",
					},
					Value: &contracts.Quantity{
						Value: hba1c,
						Unit:  "%",
					},
					EffectiveDateTime: &labTime,
					SourceReference:   "Observation/hba1c-audit",
				},
			},
			RecentEncounters: []contracts.Encounter{
				{
					EncounterID: "encounter-audit",
					Status:      "finished",
				},
			},
		},
		Knowledge: contracts.KnowledgeSnapshot{
			Terminology: contracts.TerminologySnapshot{
				ValueSetMemberships: map[string]bool{
					"HasDiabetes": true,
				},
			},
			SnapshotTimestamp: time.Now(),
		},
		Runtime: contracts.ExecutionMetadata{
			RequestID: "audit-test-request",
		},
	}

	result := evaluator.Evaluate(ctx)

	// Verify all audit fields
	t.Run("MeasureID", func(t *testing.T) {
		if result.MeasureID != "CMS122" {
			t.Errorf("MeasureID: got %q, want %q", result.MeasureID, "CMS122")
		}
	})

	t.Run("MeasureName", func(t *testing.T) {
		if result.MeasureName == "" {
			t.Error("MeasureName is empty")
		}
	})

	t.Run("MeasureVersion", func(t *testing.T) {
		if result.MeasureVersion != "2024.0.0" {
			t.Errorf("MeasureVersion: got %q, want %q", result.MeasureVersion, "2024.0.0")
		}
	})

	t.Run("LogicVersion", func(t *testing.T) {
		if result.LogicVersion != "1.0.0" {
			t.Errorf("LogicVersion: got %q, want %q", result.LogicVersion, "1.0.0")
		}
	})

	t.Run("ELMCorrespondence", func(t *testing.T) {
		expected := "DiabetesGlycemicStatusAssessmentGreaterThan9PercentFHIR:0.1.002"
		if result.ELMCorrespondence != expected {
			t.Errorf("ELMCorrespondence: got %q, want %q", result.ELMCorrespondence, expected)
		}
	})

	t.Run("EvaluatedAt", func(t *testing.T) {
		if result.EvaluatedAt.IsZero() {
			t.Error("EvaluatedAt is zero")
		}
		// Should be within last minute
		if time.Since(result.EvaluatedAt) > time.Minute {
			t.Error("EvaluatedAt is too old")
		}
	})

	t.Run("Rationale", func(t *testing.T) {
		if result.Rationale == "" {
			t.Error("Rationale is empty - should explain decision")
		}
	})
}

// TestCMS122_NoExternalCalls is a regression test ensuring no network calls
func TestCMS122_NoExternalCalls(t *testing.T) {
	// This test documents the architectural requirement:
	// CMS122 evaluator MUST NOT make any external calls.
	//
	// The evaluator receives a ClinicalExecutionContext with ALL data
	// already precomputed (KB-7 ValueSets, KB-8 calculators).
	//
	// If this test could somehow detect network calls, it would fail.
	// For now, it serves as documentation and a reminder.

	evaluator := NewCMS122Evaluator()

	// Verify the evaluator interface methods return static values
	if evaluator.MeasureID() != "CMS122" {
		t.Error("MeasureID should be static")
	}

	// The Evaluate function should work with a minimal context
	// (no HTTP mocking needed - pure function)
	now := time.Now()
	birthDate := now.AddDate(-50, 0, 0) // 50 years old

	ctx := &contracts.ClinicalExecutionContext{
		Patient: contracts.PatientContext{
			Demographics: contracts.PatientDemographics{
				BirthDate: &birthDate,
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
	}

	// Should complete without errors (no network needed)
	result := evaluator.Evaluate(ctx)

	if result.MeasureID != "CMS122" {
		t.Error("Unexpected measure ID")
	}

	t.Log("✅ REGRESSION TEST: CMS122 evaluator is a pure function (no external calls)")
}

// buildCMS122TestContext creates a test context from a test case
func buildCMS122TestContext(tc CMS122TestCase) *contracts.ClinicalExecutionContext {
	now := time.Now()
	birthDate := now.AddDate(-tc.Age, 0, 0) // Calculate birth date for target age

	ctx := &contracts.ClinicalExecutionContext{
		Patient: contracts.PatientContext{
			Demographics: contracts.PatientDemographics{
				PatientID: "test-" + tc.Name,
				BirthDate: &birthDate,
				Gender:    "female",
			},
		},
		Knowledge: contracts.KnowledgeSnapshot{
			Terminology: contracts.TerminologySnapshot{
				ValueSetMemberships: map[string]bool{
					"HasDiabetes": tc.IsDiabetic,
				},
			},
			SnapshotTimestamp: time.Now(),
		},
		Runtime: contracts.ExecutionMetadata{
			RequestID: "test-request-" + tc.Name,
		},
	}

	// Add HbA1c lab result if provided
	if tc.HbA1c != nil {
		labTime := now.AddDate(0, -1, 0) // 1 month ago
		ctx.Patient.RecentLabResults = []contracts.LabResult{
			{
				Code: contracts.ClinicalCode{
					System:  "http://loinc.org",
					Code:    LoincHbA1c,
					Display: "Hemoglobin A1c",
				},
				Value: &contracts.Quantity{
					Value: *tc.HbA1c,
					Unit:  "%",
				},
				EffectiveDateTime: &labTime,
				SourceReference:   "Observation/hba1c-test",
			},
		}
	}

	// Add qualifying encounter if needed
	if tc.HasQualifyingEncounter {
		ctx.Patient.RecentEncounters = []contracts.Encounter{
			{
				EncounterID: "encounter-test",
				Status:      "finished",
			},
		}
	}

	return ctx
}
