package measures

import (
	"testing"
	"time"

	"vaidshala/clinical-runtime-platform/contracts"
)

// ============================================================================
// CMS165 GOLDEN TESTS
// ============================================================================

func TestCMS165_GoldenTests(t *testing.T) {
	evaluator := NewCMS165Evaluator()

	for _, tc := range CMS165GoldenTestCases() {
		t.Run(tc.Name, func(t *testing.T) {
			// Build test context
			ctx := buildCMS165TestContext(tc)

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

			// Verify audit metadata is present
			if result.MeasureVersion == "" {
				t.Error("MeasureVersion is empty - audit metadata required")
			}
			if result.ELMCorrespondence == "" {
				t.Error("ELMCorrespondence is empty - audit metadata required")
			}
		})
	}
}

// TestCMS165_SpecificScenarios tests the two primary golden cases explicitly
func TestCMS165_SpecificScenarios(t *testing.T) {
	evaluator := NewCMS165Evaluator()

	t.Run("GOLDEN_1_Hypertensive_BP_120_80_Controlled", func(t *testing.T) {
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// Patient with controlled BP (120/80) → MUST be IN NUMERATOR
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		now := time.Now()
		birthDate := now.AddDate(-55, 0, 0) // 55 years old
		sbp := 120.0
		dbp := 80.0

		ctx := &contracts.ClinicalExecutionContext{
			Patient: contracts.PatientContext{
				Demographics: contracts.PatientDemographics{
					PatientID: "test-bp-controlled",
					BirthDate: &birthDate,
					Gender:    "male",
				},
				RecentVitalSigns: []contracts.VitalSign{
					{
						Code: contracts.ClinicalCode{
							System:  "http://loinc.org",
							Code:    "85354-9", // Blood pressure panel
							Display: "Blood pressure panel",
						},
						ComponentValues: []contracts.ComponentValue{
							{
								Code:  contracts.ClinicalCode{Code: LoincSystolicBP},
								Value: &contracts.Quantity{Value: sbp, Unit: "mm[Hg]"},
							},
							{
								Code:  contracts.ClinicalCode{Code: LoincDiastolicBP},
								Value: &contracts.Quantity{Value: dbp, Unit: "mm[Hg]"},
							},
						},
						EffectiveDateTime: &now,
						SourceReference:   "Observation/bp-1",
					},
				},
				RecentEncounters: []contracts.Encounter{
					{EncounterID: "enc-1", Status: "finished"},
				},
			},
			Knowledge: contracts.KnowledgeSnapshot{
				Terminology: contracts.TerminologySnapshot{
					ValueSetMemberships: map[string]bool{
						"HasHypertension": true,
					},
				},
				SnapshotTimestamp: time.Now(),
			},
			Runtime: contracts.ExecutionMetadata{
				RequestID: "test-cms165-1",
			},
		}

		result := evaluator.Evaluate(ctx)

		if !result.InInitialPopulation {
			t.Fatal("Expected patient to be IN initial population")
		}
		if !result.InNumerator {
			t.Fatalf("Expected patient to be IN numerator (BP %.0f/%.0f < 140/90), got rationale: %s",
				sbp, dbp, result.Rationale)
		}
		if result.CareGapIdentified {
			t.Fatal("Expected NO care gap (BP is controlled)")
		}

		t.Logf("✅ GOLDEN TEST 1 PASSED: BP %.0f/%.0f → Numerator=%v, CareGap=%v",
			sbp, dbp, result.InNumerator, result.CareGapIdentified)
	})

	t.Run("GOLDEN_2_Hypertensive_BP_145_92_Uncontrolled", func(t *testing.T) {
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		// Patient with uncontrolled BP (145/92) → CARE GAP
		// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
		now := time.Now()
		birthDate := now.AddDate(-60, 0, 0) // 60 years old
		sbp := 145.0
		dbp := 92.0

		ctx := &contracts.ClinicalExecutionContext{
			Patient: contracts.PatientContext{
				Demographics: contracts.PatientDemographics{
					PatientID: "test-bp-uncontrolled",
					BirthDate: &birthDate,
					Gender:    "female",
				},
				RecentVitalSigns: []contracts.VitalSign{
					{
						Code: contracts.ClinicalCode{
							System:  "http://loinc.org",
							Code:    "85354-9",
							Display: "Blood pressure panel",
						},
						ComponentValues: []contracts.ComponentValue{
							{
								Code:  contracts.ClinicalCode{Code: LoincSystolicBP},
								Value: &contracts.Quantity{Value: sbp, Unit: "mm[Hg]"},
							},
							{
								Code:  contracts.ClinicalCode{Code: LoincDiastolicBP},
								Value: &contracts.Quantity{Value: dbp, Unit: "mm[Hg]"},
							},
						},
						EffectiveDateTime: &now,
						SourceReference:   "Observation/bp-2",
					},
				},
				RecentEncounters: []contracts.Encounter{
					{EncounterID: "enc-2", Status: "finished"},
				},
			},
			Knowledge: contracts.KnowledgeSnapshot{
				Terminology: contracts.TerminologySnapshot{
					ValueSetMemberships: map[string]bool{
						"HasHypertension": true,
					},
				},
				SnapshotTimestamp: time.Now(),
			},
			Runtime: contracts.ExecutionMetadata{
				RequestID: "test-cms165-2",
			},
		}

		result := evaluator.Evaluate(ctx)

		if !result.InInitialPopulation {
			t.Fatal("Expected patient to be IN initial population")
		}
		if result.InNumerator {
			t.Fatalf("Expected patient to NOT be in numerator (BP %.0f/%.0f ≥ 140/90), got rationale: %s",
				sbp, dbp, result.Rationale)
		}
		if !result.CareGapIdentified {
			t.Fatal("Expected care gap to be identified (uncontrolled BP)")
		}

		t.Logf("✅ GOLDEN TEST 2 PASSED: BP %.0f/%.0f → Numerator=%v, CareGap=%v",
			sbp, dbp, result.InNumerator, result.CareGapIdentified)
	})
}

// TestCMS165_AuditMetadata verifies all audit fields are populated
func TestCMS165_AuditMetadata(t *testing.T) {
	evaluator := NewCMS165Evaluator()

	now := time.Now()
	birthDate := now.AddDate(-50, 0, 0)
	sbp := 130.0
	dbp := 85.0

	ctx := &contracts.ClinicalExecutionContext{
		Patient: contracts.PatientContext{
			Demographics: contracts.PatientDemographics{
				PatientID: "audit-test-bp",
				BirthDate: &birthDate,
				Gender:    "female",
			},
			RecentVitalSigns: []contracts.VitalSign{
				{
					ComponentValues: []contracts.ComponentValue{
						{
							Code:  contracts.ClinicalCode{Code: LoincSystolicBP},
							Value: &contracts.Quantity{Value: sbp, Unit: "mm[Hg]"},
						},
						{
							Code:  contracts.ClinicalCode{Code: LoincDiastolicBP},
							Value: &contracts.Quantity{Value: dbp, Unit: "mm[Hg]"},
						},
					},
					EffectiveDateTime: &now,
				},
			},
			RecentEncounters: []contracts.Encounter{
				{EncounterID: "enc-audit", Status: "finished"},
			},
		},
		Knowledge: contracts.KnowledgeSnapshot{
			Terminology: contracts.TerminologySnapshot{
				ValueSetMemberships: map[string]bool{
					"HasHypertension": true,
				},
			},
		},
	}

	result := evaluator.Evaluate(ctx)

	t.Run("MeasureID", func(t *testing.T) {
		if result.MeasureID != "CMS165" {
			t.Errorf("MeasureID: got %q, want %q", result.MeasureID, "CMS165")
		}
	})

	t.Run("MeasureVersion", func(t *testing.T) {
		if result.MeasureVersion != "2024.0.0" {
			t.Errorf("MeasureVersion: got %q, want %q", result.MeasureVersion, "2024.0.0")
		}
	})

	t.Run("ELMCorrespondence", func(t *testing.T) {
		expected := "ControllingHighBloodPressureFHIR:0.1.000"
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

// buildCMS165TestContext creates a test context from a test case
func buildCMS165TestContext(tc CMS165TestCase) *contracts.ClinicalExecutionContext {
	now := time.Now()
	birthDate := now.AddDate(-tc.Age, 0, 0)

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
					"HasHypertension":   tc.HasHypertension,
					"HasESRD":           tc.HasESRD,
					"HasCKDStage5":      tc.HasCKDStage5,
					"IsPregnant":        tc.IsPregnant,
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

	// Add BP vitals if provided
	if tc.SystolicBP != nil && tc.DiastolicBP != nil {
		ctx.Patient.RecentVitalSigns = []contracts.VitalSign{
			{
				Code: contracts.ClinicalCode{
					System:  "http://loinc.org",
					Code:    "85354-9",
					Display: "Blood pressure panel",
				},
				ComponentValues: []contracts.ComponentValue{
					{
						Code:  contracts.ClinicalCode{Code: LoincSystolicBP},
						Value: &contracts.Quantity{Value: *tc.SystolicBP, Unit: "mm[Hg]"},
					},
					{
						Code:  contracts.ClinicalCode{Code: LoincDiastolicBP},
						Value: &contracts.Quantity{Value: *tc.DiastolicBP, Unit: "mm[Hg]"},
					},
				},
				EffectiveDateTime: &now,
				SourceReference:   "Observation/bp-test",
			},
		}
	}

	// Add qualifying encounter if needed
	if tc.HasQualifyingEncounter {
		ctx.Patient.RecentEncounters = []contracts.Encounter{
			{EncounterID: "encounter-test", Status: "finished"},
		}
	}

	return ctx
}
