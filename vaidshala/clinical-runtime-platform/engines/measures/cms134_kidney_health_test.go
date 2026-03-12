package measures

import (
	"testing"
	"time"

	"vaidshala/clinical-runtime-platform/contracts"
)

// ============================================================================
// CMS134 GOLDEN TESTS
// ============================================================================

func TestCMS134_GoldenTests(t *testing.T) {
	evaluator := NewCMS134Evaluator()

	for _, tc := range CMS134GoldenTestCases() {
		t.Run(tc.Name, func(t *testing.T) {
			// Build test context
			ctx := buildCMS134TestContext(tc)

			// Evaluate
			result := evaluator.Evaluate(ctx)

			// Verify Initial Population
			if result.InInitialPopulation != tc.ExpectedInPopulation {
				t.Errorf("InInitialPopulation: got %v, want %v (rationale: %s)",
					result.InInitialPopulation, tc.ExpectedInPopulation, result.Rationale)
			}

			// Verify Exclusion (only if in population)
			if tc.ExpectedInPopulation && tc.ExpectedExcluded {
				if !result.InDenominatorExclusion {
					t.Errorf("InDenominatorExclusion: got %v, want %v (rationale: %s)",
						result.InDenominatorExclusion, tc.ExpectedExcluded, result.Rationale)
				}
			}

			// Verify Numerator (only if in population and not excluded)
			if tc.ExpectedInPopulation && !tc.ExpectedExcluded {
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

			// Verify audit metadata
			if result.MeasureVersion == "" {
				t.Error("MeasureVersion is empty - audit metadata required")
			}
		})
	}
}

// TestCMS134_SpecificScenarios tests the primary golden cases explicitly
func TestCMS134_SpecificScenarios(t *testing.T) {
	evaluator := NewCMS134Evaluator()

	t.Run("GOLDEN_1_Diabetic_CompleteKidneyPanel", func(t *testing.T) {
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// Diabetic patient with both eGFR and uACR → IN NUMERATOR
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		now := time.Now()
		birthDate := now.AddDate(-55, 0, 0)
		labTime := now.AddDate(0, -1, 0)

		ctx := &contracts.ClinicalExecutionContext{
			Patient: contracts.PatientContext{
				Demographics: contracts.PatientDemographics{
					PatientID: "test-kidney-complete",
					BirthDate: &birthDate,
					Gender:    "male",
				},
				RecentLabResults: []contracts.LabResult{
					{
						Code: contracts.ClinicalCode{
							System:  "http://loinc.org",
							Code:    LoincEGFR,
							Display: "eGFR",
						},
						Value: &contracts.Quantity{Value: 75, Unit: "mL/min/1.73m2"},
						EffectiveDateTime: &labTime,
						SourceReference:   "Observation/egfr-1",
					},
					{
						Code: contracts.ClinicalCode{
							System:  "http://loinc.org",
							Code:    LoincUACR,
							Display: "Urine albumin/creatinine ratio",
						},
						Value: &contracts.Quantity{Value: 25, Unit: "mg/g"},
						EffectiveDateTime: &labTime,
						SourceReference:   "Observation/uacr-1",
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
				SnapshotTimestamp: time.Now(),
			},
			Runtime: contracts.ExecutionMetadata{
				RequestID: "test-cms134-1",
			},
		}

		result := evaluator.Evaluate(ctx)

		if !result.InInitialPopulation {
			t.Fatal("Expected patient to be IN initial population")
		}
		if !result.InNumerator {
			t.Fatalf("Expected patient to be IN numerator (has both eGFR and uACR), got rationale: %s",
				result.Rationale)
		}
		if result.CareGapIdentified {
			t.Fatal("Expected NO care gap (complete kidney panel)")
		}

		t.Logf("✅ GOLDEN TEST 1 PASSED: Complete kidney panel → Numerator=%v, CareGap=%v",
			result.InNumerator, result.CareGapIdentified)
	})

	t.Run("GOLDEN_2_Diabetic_MissingUACR_CareGap", func(t *testing.T) {
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// Diabetic patient with eGFR but missing uACR → CARE GAP
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		now := time.Now()
		birthDate := now.AddDate(-60, 0, 0)
		labTime := now.AddDate(0, -1, 0)

		ctx := &contracts.ClinicalExecutionContext{
			Patient: contracts.PatientContext{
				Demographics: contracts.PatientDemographics{
					PatientID: "test-kidney-missing-uacr",
					BirthDate: &birthDate,
					Gender:    "female",
				},
				RecentLabResults: []contracts.LabResult{
					{
						Code: contracts.ClinicalCode{
							System:  "http://loinc.org",
							Code:    LoincEGFR,
							Display: "eGFR",
						},
						Value: &contracts.Quantity{Value: 65, Unit: "mL/min/1.73m2"},
						EffectiveDateTime: &labTime,
						SourceReference:   "Observation/egfr-2",
					},
					// No uACR test
				},
				RecentEncounters: []contracts.Encounter{
					{EncounterID: "enc-2", Status: "finished"},
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
				RequestID: "test-cms134-2",
			},
		}

		result := evaluator.Evaluate(ctx)

		if !result.InInitialPopulation {
			t.Fatal("Expected patient to be IN initial population")
		}
		if result.InNumerator {
			t.Fatalf("Expected patient to NOT be in numerator (missing uACR), got rationale: %s",
				result.Rationale)
		}
		if !result.CareGapIdentified {
			t.Fatal("Expected care gap to be identified (missing uACR)")
		}

		t.Logf("✅ GOLDEN TEST 2 PASSED: Missing uACR → Numerator=%v, CareGap=%v",
			result.InNumerator, result.CareGapIdentified)
	})
}

// TestCMS134_AuditMetadata verifies all audit fields are populated
func TestCMS134_AuditMetadata(t *testing.T) {
	evaluator := NewCMS134Evaluator()

	now := time.Now()
	birthDate := now.AddDate(-50, 0, 0)
	labTime := now.AddDate(0, -1, 0)

	ctx := &contracts.ClinicalExecutionContext{
		Patient: contracts.PatientContext{
			Demographics: contracts.PatientDemographics{
				PatientID: "audit-test-kidney",
				BirthDate: &birthDate,
				Gender:    "male",
			},
			RecentLabResults: []contracts.LabResult{
				{
					Code: contracts.ClinicalCode{Code: LoincEGFR},
					Value: &contracts.Quantity{Value: 80, Unit: "mL/min/1.73m2"},
					EffectiveDateTime: &labTime,
				},
				{
					Code: contracts.ClinicalCode{Code: LoincUACR},
					Value: &contracts.Quantity{Value: 20, Unit: "mg/g"},
					EffectiveDateTime: &labTime,
				},
			},
			RecentEncounters: []contracts.Encounter{
				{EncounterID: "enc-audit", Status: "finished"},
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

	result := evaluator.Evaluate(ctx)

	t.Run("MeasureID", func(t *testing.T) {
		if result.MeasureID != "CMS134" {
			t.Errorf("MeasureID: got %q, want %q", result.MeasureID, "CMS134")
		}
	})

	t.Run("MeasureVersion", func(t *testing.T) {
		if result.MeasureVersion != "2024.0.0" {
			t.Errorf("MeasureVersion: got %q, want %q", result.MeasureVersion, "2024.0.0")
		}
	})

	t.Run("ELMCorrespondence", func(t *testing.T) {
		expected := "KidneyHealthEvaluationFHIR:0.1.000"
		if result.ELMCorrespondence != expected {
			t.Errorf("ELMCorrespondence: got %q, want %q", result.ELMCorrespondence, expected)
		}
	})

	t.Run("EvaluatedAt", func(t *testing.T) {
		if result.EvaluatedAt.IsZero() {
			t.Error("EvaluatedAt is zero")
		}
	})

	t.Run("Rationale", func(t *testing.T) {
		if result.Rationale == "" {
			t.Error("Rationale is empty")
		}
	})
}

// buildCMS134TestContext creates a test context from a test case
func buildCMS134TestContext(tc CMS134TestCase) *contracts.ClinicalExecutionContext {
	now := time.Now()
	birthDate := now.AddDate(-tc.Age, 0, 0)
	labTime := now.AddDate(0, -1, 0)

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
					"HasDiabetes":       tc.IsDiabetic,
					"HasCKDStage5":      tc.HasCKDStage5,
					"HasESRD":           tc.HasESRD,
					"InHospice":         tc.InHospice,
					"InPalliativeCare":  false,
				},
			},
			SnapshotTimestamp: now,
		},
		Runtime: contracts.ExecutionMetadata{
			RequestID: "test-request-" + tc.Name,
		},
	}

	// Add lab results
	var labs []contracts.LabResult
	if tc.HasEGFR {
		labs = append(labs, contracts.LabResult{
			Code: contracts.ClinicalCode{
				System:  "http://loinc.org",
				Code:    LoincEGFR,
				Display: "eGFR",
			},
			Value:             &contracts.Quantity{Value: 75, Unit: "mL/min/1.73m2"},
			EffectiveDateTime: &labTime,
			SourceReference:   "Observation/egfr-test",
		})
	}
	if tc.HasUACR {
		labs = append(labs, contracts.LabResult{
			Code: contracts.ClinicalCode{
				System:  "http://loinc.org",
				Code:    LoincUACR,
				Display: "Urine albumin/creatinine ratio",
			},
			Value:             &contracts.Quantity{Value: 25, Unit: "mg/g"},
			EffectiveDateTime: &labTime,
			SourceReference:   "Observation/uacr-test",
		})
	}
	ctx.Patient.RecentLabResults = labs

	// Add qualifying encounter if needed
	if tc.HasQualifyingEncounter {
		ctx.Patient.RecentEncounters = []contracts.Encounter{
			{EncounterID: "encounter-test", Status: "finished"},
		}
	}

	return ctx
}
