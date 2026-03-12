// Package tests provides comprehensive test coverage for KB-19 Protocol Orchestrator.
//
// PILLARS 3 & 4: SAFETY GATING TESTS
// Tests pregnancy/maternity safety, ICU integration, and bleeding risk management.
// Clinical goal: Zero false negatives on safety blocks.
package tests

import (
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-19-protocol-orchestrator/internal/arbitration"
	"kb-19-protocol-orchestrator/internal/models"
)

// ============================================================================
// PILLAR 3: MATERNITY/PREGNANCY SAFETY TESTS
// FDA Category X and teratogenic drugs MUST be blocked
// ============================================================================

func TestPregnancySafety_CategoryXDrugBlocked(t *testing.T) {
	// Clinical scenario:
	// - Pregnant patient (any trimester)
	// - Protocol recommends Category X drug (e.g., statins)
	// - EXPECTED: HARD_BLOCK, drug converted to AVOID

	log := logrus.NewEntry(logrus.New())
	gatekeeper := arbitration.NewSafetyGatekeeper(log)

	categoryXDrugs := []string{
		"atorvastatin",
		"simvastatin",
		"rosuvastatin",
		"isotretinoin",
		"methotrexate",
		"misoprostol",
		"finasteride",
	}

	for _, drug := range categoryXDrugs {
		t.Run("CategoryX_"+drug, func(t *testing.T) {
			decisions := []models.ArbitratedDecision{
				{
					ID:             uuid.New(),
					DecisionType:   models.DecisionDo,
					Target:         drug,
					Rationale:      "Guideline recommendation",
					SourceProtocol: "TEST-PROTOCOL",
				},
			}

			patientCtx := &models.PatientContext{
				PregnancyStatus: &models.PregnancyStatus{
					IsPregnant:     true,
					GestationalAge: 12,
					Trimester:      1,
				},
			}

			safeDecisions, gates := gatekeeper.Apply(decisions, patientCtx)

			// Drug should be blocked
			require.Len(t, safeDecisions, 1)
			assert.Equal(t, models.DecisionAvoid, safeDecisions[0].DecisionType,
				"%s should be AVOIDED in pregnancy", drug)

			// Should have pregnancy safety flag
			var hasPregnancyFlag bool
			var isHardBlock bool
			for _, flag := range safeDecisions[0].SafetyFlags {
				if flag.Type == models.FlagPregnancy {
					hasPregnancyFlag = true
					if flag.Severity == "HARD_BLOCK" {
						isHardBlock = true
					}
				}
			}

			assert.True(t, hasPregnancyFlag, "Should have PREGNANCY flag for %s", drug)
			assert.True(t, isHardBlock, "Category X drug %s should be HARD_BLOCK", drug)

			// Safety gate should be triggered
			var pregnancyGateTriggered bool
			for _, gate := range gates {
				if gate.Name == "Pregnancy Safety" && gate.Triggered {
					pregnancyGateTriggered = true
					assert.Equal(t, "BLOCK", gate.Result, "Pregnancy gate should BLOCK")
				}
			}
			assert.True(t, pregnancyGateTriggered, "Pregnancy safety gate should trigger for %s", drug)
		})
	}
}

func TestPregnancySafety_TeratogenicDrugBlocked(t *testing.T) {
	// Additional teratogenic drugs beyond Category X

	log := logrus.NewEntry(logrus.New())
	gatekeeper := arbitration.NewSafetyGatekeeper(log)

	teratogenicDrugs := []string{
		"warfarin",
		"valproate",
		"lithium",
	}

	for _, drug := range teratogenicDrugs {
		t.Run("Teratogenic_"+drug, func(t *testing.T) {
			decisions := []models.ArbitratedDecision{
				{
					ID:             uuid.New(),
					DecisionType:   models.DecisionDo,
					Target:         drug,
					Rationale:      "Standard therapy",
					SourceProtocol: "TEST-PROTOCOL",
				},
			}

			patientCtx := &models.PatientContext{
				PregnancyStatus: &models.PregnancyStatus{
					IsPregnant:     true,
					GestationalAge: 20,
					Trimester:      2,
				},
			}

			safeDecisions, _ := gatekeeper.Apply(decisions, patientCtx)

			require.Len(t, safeDecisions, 1)
			assert.Equal(t, models.DecisionAvoid, safeDecisions[0].DecisionType,
				"Teratogenic drug %s should be AVOIDED", drug)
		})
	}
}

func TestPregnancySafety_ACEInhibitorBlocked(t *testing.T) {
	// ACE inhibitors are contraindicated in pregnancy (fetotoxic)

	log := logrus.NewEntry(logrus.New())
	gatekeeper := arbitration.NewSafetyGatekeeper(log)

	aceInhibitors := []string{
		"lisinopril",
		"enalapril",
		"ramipril",
		"captopril",
		"benazepril",
		"fosinopril",
		"quinapril",
		"trandolapril",
	}

	for _, drug := range aceInhibitors {
		t.Run("ACEi_"+drug, func(t *testing.T) {
			decisions := []models.ArbitratedDecision{
				{
					ID:             uuid.New(),
					DecisionType:   models.DecisionDo,
					Target:         drug,
					Rationale:      "BP control",
					SourceProtocol: "HTN-MANAGEMENT",
				},
			}

			patientCtx := &models.PatientContext{
				PregnancyStatus: &models.PregnancyStatus{
					IsPregnant:     true,
					GestationalAge: 28,
					Trimester:      3,
				},
			}

			safeDecisions, _ := gatekeeper.Apply(decisions, patientCtx)

			require.Len(t, safeDecisions, 1)
			assert.Equal(t, models.DecisionAvoid, safeDecisions[0].DecisionType,
				"ACE inhibitor %s should be AVOIDED in pregnancy", drug)
		})
	}
}

func TestPregnancySafety_SafeDrugAllowed(t *testing.T) {
	// Verify that safe drugs are NOT blocked in pregnancy

	log := logrus.NewEntry(logrus.New())
	gatekeeper := arbitration.NewSafetyGatekeeper(log)

	safeDrugs := []string{
		"acetaminophen",
		"metformin", // Category B, often continued in GDM
		"methyldopa", // Safe antihypertensive in pregnancy
	}

	for _, drug := range safeDrugs {
		t.Run("Safe_"+drug, func(t *testing.T) {
			decisions := []models.ArbitratedDecision{
				{
					ID:             uuid.New(),
					DecisionType:   models.DecisionDo,
					Target:         drug,
					Rationale:      "Standard therapy",
					SourceProtocol: "SAFE-PROTOCOL",
				},
			}

			patientCtx := &models.PatientContext{
				PregnancyStatus: &models.PregnancyStatus{
					IsPregnant:     true,
					GestationalAge: 24,
					Trimester:      2,
				},
			}

			safeDecisions, _ := gatekeeper.Apply(decisions, patientCtx)

			require.Len(t, safeDecisions, 1)
			assert.Equal(t, models.DecisionDo, safeDecisions[0].DecisionType,
				"Safe drug %s should remain DO in pregnancy", drug)
		})
	}
}

func TestPregnancySafety_NotPregnantNoBlock(t *testing.T) {
	// Verify that teratogenic drugs are NOT blocked in non-pregnant patients

	log := logrus.NewEntry(logrus.New())
	gatekeeper := arbitration.NewSafetyGatekeeper(log)

	decisions := []models.ArbitratedDecision{
		{
			ID:             uuid.New(),
			DecisionType:   models.DecisionDo,
			Target:         "warfarin",
			Rationale:      "Anticoagulation for AFib",
			SourceProtocol: "AFIB-ANTICOAG",
		},
	}

	patientCtx := &models.PatientContext{
		PregnancyStatus: nil, // Not pregnant
	}

	safeDecisions, gates := gatekeeper.Apply(decisions, patientCtx)

	require.Len(t, safeDecisions, 1)

	// Warfarin should NOT be blocked (no pregnancy)
	var hasPregnancyBlock bool
	for _, flag := range safeDecisions[0].SafetyFlags {
		if flag.Type == models.FlagPregnancy {
			hasPregnancyBlock = true
		}
	}
	assert.False(t, hasPregnancyBlock, "Non-pregnant patient should not have pregnancy block")

	// Pregnancy gate should not be in the list
	var hasPregnancyGate bool
	for _, gate := range gates {
		if gate.Name == "Pregnancy Safety" {
			hasPregnancyGate = true
		}
	}
	assert.False(t, hasPregnancyGate, "Pregnancy gate should not apply to non-pregnant patient")
}

// ============================================================================
// PILLAR 4: ICU INTEGRATION TESTS
// ICU safety engine integration with shock, DIC, and multi-organ scenarios
// ============================================================================

func TestICUSafety_UncompensatedShockBlocksVasodilators(t *testing.T) {
	// Clinical scenario:
	// - Patient in uncompensated shock
	// - Protocol recommends vasodilator (e.g., nitroprusside for BP)
	// - EXPECTED: HARD_BLOCK - vasodilator would worsen hemodynamics

	log := logrus.NewEntry(logrus.New())
	gatekeeper := arbitration.NewSafetyGatekeeper(log)

	hemodynamicRiskDrugs := []string{
		"nitroprusside",
		"nitroglycerin",
		"hydralazine",
		"propofol",
	}

	for _, drug := range hemodynamicRiskDrugs {
		t.Run("ShockBlock_"+drug, func(t *testing.T) {
			decisions := []models.ArbitratedDecision{
				{
					ID:             uuid.New(),
					DecisionType:   models.DecisionDo,
					Target:         drug,
					Rationale:      "BP management",
					SourceProtocol: "HTN-EMERGENCY",
				},
			}

			patientCtx := &models.PatientContext{
				ICUStateSummary: &models.ICUClinicalState{
					ShockState:       "UNCOMPENSATED",
					VasopressorScore: 3.5,
				},
			}

			safeDecisions, gates := gatekeeper.Apply(decisions, patientCtx)

			require.Len(t, safeDecisions, 1)
			assert.Equal(t, models.DecisionAvoid, safeDecisions[0].DecisionType,
				"%s should be AVOIDED in uncompensated shock", drug)

			// Should have ICU hard block flag
			var hasICUBlock bool
			for _, flag := range safeDecisions[0].SafetyFlags {
				if flag.Type == models.FlagICUHardBlock && flag.Severity == "HARD_BLOCK" {
					hasICUBlock = true
				}
			}
			assert.True(t, hasICUBlock, "Should have ICU_HARD_BLOCK flag")

			// ICU safety gate should trigger
			var icuGateTriggered bool
			for _, gate := range gates {
				if gate.Name == "ICU Safety Engine" && gate.Triggered {
					icuGateTriggered = true
					assert.Equal(t, "BLOCK", gate.Result)
				}
			}
			assert.True(t, icuGateTriggered, "ICU safety gate should trigger")
		})
	}
}

func TestICUSafety_CompensatedShockAllowsCautiously(t *testing.T) {
	// Patient in compensated shock - vasodilators should be cautiously allowed

	log := logrus.NewEntry(logrus.New())
	gatekeeper := arbitration.NewSafetyGatekeeper(log)

	decisions := []models.ArbitratedDecision{
		{
			ID:             uuid.New(),
			DecisionType:   models.DecisionDo,
			Target:         "nitroglycerin",
			Rationale:      "Preload reduction",
			SourceProtocol: "HF-MANAGEMENT",
		},
	}

	patientCtx := &models.PatientContext{
		ICUStateSummary: &models.ICUClinicalState{
			ShockState:       "COMPENSATED",
			VasopressorScore: 0.5,
		},
	}

	safeDecisions, _ := gatekeeper.Apply(decisions, patientCtx)

	require.Len(t, safeDecisions, 1)
	// In compensated shock, should still allow with monitoring
	// Decision type may be DO or CONSIDER
	assert.NotEqual(t, models.DecisionAvoid, safeDecisions[0].DecisionType,
		"Compensated shock should not automatically block vasodilators")
}

func TestICUSafety_AKIStage2BlocksNephrotoxics(t *testing.T) {
	// AKI Stage 2+ should block nephrotoxic drugs

	log := logrus.NewEntry(logrus.New())
	gatekeeper := arbitration.NewSafetyGatekeeper(log)

	nephrotoxicDrugs := []string{
		"gentamicin",
		"tobramycin",
		"vancomycin",
		"amphotericin",
	}

	for _, drug := range nephrotoxicDrugs {
		t.Run("AKI_"+drug, func(t *testing.T) {
			decisions := []models.ArbitratedDecision{
				{
					ID:             uuid.New(),
					DecisionType:   models.DecisionDo,
					Target:         drug,
					Rationale:      "Infection coverage",
					SourceProtocol: "INFECTION-PROTOCOL",
				},
			}

			patientCtx := &models.PatientContext{
				ICUStateSummary: &models.ICUClinicalState{
					AKIStage:    2,
					UrineOutput: 0.3, // Oliguric
				},
			}

			safeDecisions, _ := gatekeeper.Apply(decisions, patientCtx)

			require.Len(t, safeDecisions, 1)

			// Should have renal safety flag
			var hasRenalFlag bool
			for _, flag := range safeDecisions[0].SafetyFlags {
				if flag.Type == models.FlagRenal {
					hasRenalFlag = true
				}
			}
			assert.True(t, hasRenalFlag, "%s should have RENAL flag in AKI stage 2", drug)
		})
	}
}

func TestICUSafety_DICBlocksAnticoagulants(t *testing.T) {
	// DIC score ≥ 5 or severe thrombocytopenia should block anticoagulants

	log := logrus.NewEntry(logrus.New())
	gatekeeper := arbitration.NewSafetyGatekeeper(log)

	anticoagulants := []string{
		"heparin",
		"enoxaparin",
		"warfarin",
		"apixaban",
		"rivaroxaban",
		"dabigatran",
	}

	for _, drug := range anticoagulants {
		t.Run("DIC_"+drug, func(t *testing.T) {
			decisions := []models.ArbitratedDecision{
				{
					ID:             uuid.New(),
					DecisionType:   models.DecisionDo,
					Target:         drug,
					Rationale:      "VTE prophylaxis",
					SourceProtocol: "VTE-PROPHYLAXIS",
				},
			}

			patientCtx := &models.PatientContext{
				ICUStateSummary: &models.ICUClinicalState{
					DICScore:     6,
					PlateletsLow: true,
					BleedingRisk: "HIGH",
				},
			}

			safeDecisions, gates := gatekeeper.Apply(decisions, patientCtx)

			require.Len(t, safeDecisions, 1)
			assert.Equal(t, models.DecisionAvoid, safeDecisions[0].DecisionType,
				"%s should be AVOIDED in DIC", drug)

			// Should have bleeding flag
			var hasBleedingFlag bool
			for _, flag := range safeDecisions[0].SafetyFlags {
				if flag.Type == models.FlagBleeding {
					hasBleedingFlag = true
				}
			}
			assert.True(t, hasBleedingFlag, "Should have BLEEDING flag in DIC")

			// ICU gate should show block
			var hasICUBlock bool
			for _, gate := range gates {
				if gate.Name == "ICU Safety Engine" && gate.Result == "BLOCK" {
					hasICUBlock = true
				}
			}
			assert.True(t, hasICUBlock, "ICU gate should BLOCK anticoagulant in DIC")
		})
	}
}

func TestICUSafety_CriticalVitalsEscalatesUrgency(t *testing.T) {
	// Critical vital signs should escalate urgency of all decisions

	log := logrus.NewEntry(logrus.New())
	gatekeeper := arbitration.NewSafetyGatekeeper(log)

	decisions := []models.ArbitratedDecision{
		{
			ID:             uuid.New(),
			DecisionType:   models.DecisionDo,
			Target:         "metoprolol",
			Urgency:        models.UrgencyRoutine, // Start as Routine
			Rationale:      "Rate control",
			SourceProtocol: "AFIB-RATE",
		},
		{
			ID:             uuid.New(),
			DecisionType:   models.DecisionDo,
			Target:         "lisinopril",
			Urgency:        models.UrgencyScheduled, // Start as Scheduled
			Rationale:      "Chronic HF management",
			SourceProtocol: "HF-ACCAHA-2022",
		},
	}

	patientCtx := &models.PatientContext{
		Vitals: models.VitalSigns{
			SystolicBP: 75,  // Critically low
			HeartRate:  145, // Critically high
			SpO2:       85,  // Critically low
			GCS:        8,   // Critically low
		},
	}

	safeDecisions, gates := gatekeeper.Apply(decisions, patientCtx)

	// All routine/scheduled decisions should be escalated to Urgent or STAT
	urgencyPriority := func(u models.ActionUrgency) int {
		switch u {
		case models.UrgencySTAT:
			return 4
		case models.UrgencyUrgent:
			return 3
		case models.UrgencyRoutine:
			return 2
		case models.UrgencyScheduled:
			return 1
		default:
			return 0
		}
	}

	for _, d := range safeDecisions {
		if d.Target == "metoprolol" || d.Target == "lisinopril" {
			assert.GreaterOrEqual(t, urgencyPriority(d.Urgency), urgencyPriority(models.UrgencyUrgent),
				"Critical vitals should escalate urgency for %s", d.Target)
		}
	}

	// Critical vitals gate should trigger
	var hasVitalGate bool
	for _, gate := range gates {
		if gate.Name == "Critical Vitals" && gate.Triggered {
			hasVitalGate = true
		}
	}
	assert.True(t, hasVitalGate, "Critical vitals gate should trigger")
}

// ============================================================================
// BLEEDING RISK SAFETY TESTS
// High bleeding risk should flag anticoagulants/antiplatelets
// ============================================================================

func TestBleedingSafety_HighRiskFlags(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	gatekeeper := arbitration.NewSafetyGatekeeper(log)

	bleedingRiskDrugs := []string{
		"heparin",
		"enoxaparin",
		"warfarin",
		"aspirin",
		"clopidogrel",
		"prasugrel",
		"ticagrelor",
		"apixaban",
		"rivaroxaban",
		"dabigatran",
		"edoxaban",
	}

	for _, drug := range bleedingRiskDrugs {
		t.Run("BleedingRisk_"+drug, func(t *testing.T) {
			decisions := []models.ArbitratedDecision{
				{
					ID:             uuid.New(),
					DecisionType:   models.DecisionDo,
					Target:         drug,
					Rationale:      "Indicated per guideline",
					SourceProtocol: "ANTICOAG-PROTOCOL",
				},
			}

			patientCtx := &models.PatientContext{
				ICUStateSummary: &models.ICUClinicalState{
					BleedingRisk: "HIGH",
				},
			}

			safeDecisions, gates := gatekeeper.Apply(decisions, patientCtx)

			require.Len(t, safeDecisions, 1)

			// Should have bleeding flag
			var hasBleedingFlag bool
			for _, flag := range safeDecisions[0].SafetyFlags {
				if flag.Type == models.FlagBleeding {
					hasBleedingFlag = true
				}
			}
			assert.True(t, hasBleedingFlag, "High bleeding risk should flag %s", drug)

			// Bleeding risk gate should trigger
			var bleedingGateTriggered bool
			for _, gate := range gates {
				if gate.Name == "Bleeding Risk" && gate.Triggered {
					bleedingGateTriggered = true
				}
			}
			assert.True(t, bleedingGateTriggered, "Bleeding risk gate should trigger for %s", drug)
		})
	}
}

// ============================================================================
// RENAL SAFETY TESTS
// eGFR < 30 should trigger renal safety checks
// ============================================================================

func TestRenalSafety_LowEGFRFlags(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	gatekeeper := arbitration.NewSafetyGatekeeper(log)

	renalRiskDrugs := []string{
		"gentamicin",
		"tobramycin",
		"amikacin",
		"vancomycin",
		"amphotericin",
		"ibuprofen",
		"ketorolac",
		"naproxen",
		"indomethacin",
		"diclofenac",
	}

	for _, drug := range renalRiskDrugs {
		t.Run("RenalRisk_"+drug, func(t *testing.T) {
			decisions := []models.ArbitratedDecision{
				{
					ID:             uuid.New(),
					DecisionType:   models.DecisionDo,
					Target:         drug,
					Rationale:      "Standard indication",
					SourceProtocol: "TEST-PROTOCOL",
				},
			}

			patientCtx := &models.PatientContext{
				CQLTruthFlags: map[string]bool{
					"HasAKI": true,
				},
				CalculatorScores: map[string]float64{
					"eGFR": 22, // Severe renal impairment
				},
			}

			safeDecisions, gates := gatekeeper.Apply(decisions, patientCtx)

			require.Len(t, safeDecisions, 1)

			// Should have renal flag
			var hasRenalFlag bool
			for _, flag := range safeDecisions[0].SafetyFlags {
				if flag.Type == models.FlagRenal {
					hasRenalFlag = true
				}
			}
			assert.True(t, hasRenalFlag, "Low eGFR should flag %s", drug)

			// Should add monitoring
			assert.NotEmpty(t, safeDecisions[0].MonitoringPlan,
				"Renal safety should add monitoring for %s", drug)

			// Renal gate should trigger
			var renalGateTriggered bool
			for _, gate := range gates {
				if gate.Name == "Renal Safety" && gate.Triggered {
					renalGateTriggered = true
				}
			}
			assert.True(t, renalGateTriggered, "Renal safety gate should trigger for %s", drug)
		})
	}
}

// ============================================================================
// MULTI-ORGAN FAILURE SCENARIO
// Tests complex ICU patient with multiple safety concerns
// ============================================================================

func TestICUSafety_MultiOrganFailure(t *testing.T) {
	// Complex scenario: Septic shock with AKI, DIC, and respiratory failure

	log := logrus.NewEntry(logrus.New())
	gatekeeper := arbitration.NewSafetyGatekeeper(log)

	decisions := []models.ArbitratedDecision{
		{
			ID:             uuid.New(),
			DecisionType:   models.DecisionDo,
			Target:         "vancomycin",
			Rationale:      "MRSA coverage",
			SourceProtocol: "SEPSIS-ANTIBIOTICS",
		},
		{
			ID:             uuid.New(),
			DecisionType:   models.DecisionDo,
			Target:         "heparin",
			Rationale:      "VTE prophylaxis",
			SourceProtocol: "VTE-PROPHYLAXIS",
		},
		{
			ID:             uuid.New(),
			DecisionType:   models.DecisionDo,
			Target:         "nitroprusside",
			Rationale:      "BP management",
			SourceProtocol: "HTN-EMERGENCY",
		},
	}

	patientCtx := &models.PatientContext{
		Vitals: models.VitalSigns{
			SystolicBP: 85,
			SpO2:       88,
		},
		ICUStateSummary: &models.ICUClinicalState{
			ShockState:       "UNCOMPENSATED",
			VasopressorScore: 4.0,
			AKIStage:         3, // Severe AKI
			DICScore:         7,
			PlateletsLow:     true,
			BleedingRisk:     "HIGH",
			ARDSSeverity:     "SEVERE",
			SepsisStatus:     "SEPTIC_SHOCK",
			SOFAScore:        15,
		},
		CQLTruthFlags: map[string]bool{
			"HasAKI":    true,
			"HasSepsis": true,
		},
		CalculatorScores: map[string]float64{
			"eGFR": 12,
			"SOFA": 15,
		},
	}

	safeDecisions, gates := gatekeeper.Apply(decisions, patientCtx)

	// Multiple gates should apply
	assert.GreaterOrEqual(t, len(gates), 2, "Multiple safety gates should apply to multi-organ failure")

	// Verify each drug has appropriate flags
	for _, d := range safeDecisions {
		switch d.Target {
		case "vancomycin":
			// Should have renal flag
			var hasRenal bool
			for _, f := range d.SafetyFlags {
				if f.Type == models.FlagRenal {
					hasRenal = true
				}
			}
			assert.True(t, hasRenal, "Vancomycin should have renal flag in AKI")

		case "heparin":
			// Should be avoided due to DIC
			assert.Equal(t, models.DecisionAvoid, d.DecisionType,
				"Heparin should be AVOIDED in DIC with bleeding risk")

		case "nitroprusside":
			// Should be avoided due to uncompensated shock
			assert.Equal(t, models.DecisionAvoid, d.DecisionType,
				"Nitroprusside should be AVOIDED in uncompensated shock")
		}
	}
}

// ============================================================================
// SAFETY FLAG SEVERITY TESTS
// ============================================================================

func TestSafetyFlag_SeverityLevels(t *testing.T) {
	severities := []string{"WARNING", "CAUTION", "HARD_BLOCK"}

	for _, severity := range severities {
		flag := models.SafetyFlag{
			Type:     models.FlagRenal,
			Severity: severity,
			Reason:   "Test reason",
			Source:   "TEST_SOURCE",
		}
		assert.NotEmpty(t, flag.Severity)
	}
}

func TestSafetyFlag_Override(t *testing.T) {
	decision := models.NewArbitratedDecision(models.DecisionDo, "test-drug", "test reason")
	decision.AddSafetyFlag(models.FlagRenal, "WARNING", "Test warning", "TEST")

	require.Len(t, decision.SafetyFlags, 1)
	assert.False(t, decision.SafetyFlags[0].Overridden, "New flag should not be overridden")

	// Simulate override
	decision.SafetyFlags[0].Overridden = true
	decision.SafetyFlags[0].OverrideNote = "Clinical override by attending"

	assert.True(t, decision.SafetyFlags[0].Overridden)
	assert.NotEmpty(t, decision.SafetyFlags[0].OverrideNote)
}
