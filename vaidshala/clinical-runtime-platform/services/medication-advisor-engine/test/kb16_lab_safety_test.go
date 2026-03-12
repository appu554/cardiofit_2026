// Package test provides KB-16 Lab Safety integration tests for the Medication Advisor Engine.
// These tests validate lab-based hard blocks and governance events for critical clinical scenarios.
package test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cardiofit/medication-advisor-engine/advisor"
	"github.com/cardiofit/medication-advisor-engine/snapshot"
)

// =============================================================================
// KB-16 Lab Safety Tests
// Tests lab-based medication contraindications and hard blocks
// =============================================================================

// TestKB16_HyperkalemiaACEInhibitor tests K+ > 5.5 + ACE inhibitor hard stop
func TestKB16_HyperkalemiaACEInhibitor(t *testing.T) {
	engine := createTestEngine()

	// Patient labs with critical hyperkalemia
	patientLabs := []advisor.LabValue{
		{
			Code:    "2823-3", // LOINC for Potassium
			Display: "Potassium [Moles/volume] in Serum or Plasma",
			Value:   6.2, // Critical high - above 5.5 threshold
			Unit:    "mmol/L",
		},
	}

	// Propose Lisinopril (ACE inhibitor)
	proposedMeds := []advisor.ClinicalCode{
		{System: "RxNorm", Code: "29046", Display: "Lisinopril 10mg"},
	}

	// Execute lab hard block check
	hardBlocks := engine.TestProcessLabHardBlocks(proposedMeds, patientLabs)

	// Verify hard block was generated
	require.NotEmpty(t, hardBlocks, "Should generate hard block for K+ 6.2 + ACE inhibitor")

	block := hardBlocks[0]
	assert.Equal(t, "LAB_CONTRAINDICATION", block.BlockType)
	assert.Equal(t, "critical", block.Severity)
	assert.Equal(t, "KB-16", block.KBSource)
	assert.Contains(t, block.Reason, "Potassium")
	assert.Contains(t, block.Reason, "hyperkalemia")
	assert.True(t, block.RequiresAck)

	t.Logf("✅ KB-16 Test PASSED: Hyperkalemia + ACE Inhibitor → HARD_STOP")
	t.Logf("   Lab: K+ = 6.2 mmol/L (threshold: 5.5)")
	t.Logf("   Medication: Lisinopril (ACE Inhibitor)")
	t.Logf("   Block Type: %s, Severity: %s", block.BlockType, block.Severity)
}

// TestKB16_LowEGFRMetformin tests eGFR < 30 + Metformin hard stop
func TestKB16_LowEGFRMetformin(t *testing.T) {
	engine := createTestEngine()

	// Patient labs with severely reduced eGFR
	patientLabs := []advisor.LabValue{
		{
			Code:    "33914-3", // LOINC for eGFR
			Display: "Glomerular filtration rate/1.73 sq M.predicted",
			Value:   25, // Below 30 threshold - contraindicated
			Unit:    "mL/min/1.73m2",
		},
	}

	// Propose Metformin
	proposedMeds := []advisor.ClinicalCode{
		{System: "RxNorm", Code: "6809", Display: "Metformin 500mg"},
	}

	// Execute lab hard block check
	hardBlocks := engine.TestProcessLabHardBlocks(proposedMeds, patientLabs)

	// Verify hard block was generated
	require.NotEmpty(t, hardBlocks, "Should generate hard block for eGFR 25 + Metformin")

	block := hardBlocks[0]
	assert.Equal(t, "LAB_CONTRAINDICATION", block.BlockType)
	assert.Equal(t, "KB-16", block.KBSource)
	assert.Contains(t, block.Reason, "eGFR")
	assert.Contains(t, block.Reason, "lactic acidosis")

	t.Logf("✅ KB-16 Test PASSED: Low eGFR + Metformin → HARD_STOP")
	t.Logf("   Lab: eGFR = 25 mL/min/1.73m2 (threshold: 30)")
	t.Logf("   Medication: Metformin (Biguanide)")
}

// TestKB16_HighINRWarfarin tests INR > 4.0 + Warfarin hard stop
func TestKB16_HighINRWarfarin(t *testing.T) {
	engine := createTestEngine()

	// Patient labs with critically elevated INR
	patientLabs := []advisor.LabValue{
		{
			Code:    "5902-2", // LOINC for INR
			Display: "INR in Platelet poor plasma by Coagulation assay",
			Value:   4.5, // Above 4.0 threshold
			Unit:    "ratio",
		},
	}

	// Propose additional Warfarin dose
	proposedMeds := []advisor.ClinicalCode{
		{System: "RxNorm", Code: "11289", Display: "Warfarin 5mg"},
	}

	// Execute lab hard block check
	hardBlocks := engine.TestProcessLabHardBlocks(proposedMeds, patientLabs)

	// Verify hard block was generated
	require.NotEmpty(t, hardBlocks, "Should generate hard block for INR 4.5 + Warfarin")

	block := hardBlocks[0]
	assert.Equal(t, "LAB_CONTRAINDICATION", block.BlockType)
	assert.Equal(t, "critical", block.Severity)
	assert.Contains(t, block.Reason, "INR")
	assert.Contains(t, block.Reason, "bleeding")

	t.Logf("✅ KB-16 Test PASSED: High INR + Warfarin → HARD_STOP")
	t.Logf("   Lab: INR = 4.5 (threshold: 4.0)")
	t.Logf("   Medication: Warfarin (Anticoagulant)")
}

// TestKB16_ThrombocytopeniaAnticoagulant tests Platelets < 50000 + Anticoagulant
func TestKB16_ThrombocytopeniaAnticoagulant(t *testing.T) {
	engine := createTestEngine()

	// Patient labs with thrombocytopenia
	patientLabs := []advisor.LabValue{
		{
			Code:    "777-3", // LOINC for Platelet count
			Display: "Platelets [#/volume] in Blood",
			Value:   35000, // Below 50000 threshold
			Unit:    "/uL",
		},
	}

	// Propose Apixaban (anticoagulant)
	proposedMeds := []advisor.ClinicalCode{
		{System: "RxNorm", Code: "114934", Display: "Apixaban 5mg"},
	}

	// Execute lab hard block check
	hardBlocks := engine.TestProcessLabHardBlocks(proposedMeds, patientLabs)

	// Verify hard block was generated
	require.NotEmpty(t, hardBlocks, "Should generate hard block for Platelets 35K + Anticoagulant")

	block := hardBlocks[0]
	assert.Equal(t, "LAB_CONTRAINDICATION", block.BlockType)
	assert.Contains(t, block.Reason, "Platelet")
	assert.Contains(t, block.Reason, "bleeding")

	t.Logf("✅ KB-16 Test PASSED: Thrombocytopenia + Anticoagulant → HARD_STOP")
	t.Logf("   Lab: Platelets = 35,000/uL (threshold: 50,000)")
	t.Logf("   Medication: Apixaban (Anticoagulant)")
}

// TestKB16_HypokalemiaDigoxin tests K+ < 3.5 + Digoxin hard stop
func TestKB16_HypokalemiaDigoxin(t *testing.T) {
	engine := createTestEngine()

	// Patient labs with hypokalemia
	patientLabs := []advisor.LabValue{
		{
			Code:    "2823-3", // LOINC for Potassium
			Display: "Potassium [Moles/volume] in Serum or Plasma",
			Value:   3.0, // Below 3.5 threshold
			Unit:    "mmol/L",
		},
	}

	// Propose Digoxin
	proposedMeds := []advisor.ClinicalCode{
		{System: "RxNorm", Code: "3407", Display: "Digoxin 0.25mg"},
	}

	// Execute lab hard block check
	hardBlocks := engine.TestProcessLabHardBlocks(proposedMeds, patientLabs)

	// Verify hard block was generated
	require.NotEmpty(t, hardBlocks, "Should generate hard block for K+ 3.0 + Digoxin")

	block := hardBlocks[0]
	assert.Equal(t, "LAB_CONTRAINDICATION", block.BlockType)
	assert.Contains(t, block.Reason, "Potassium")
	assert.Contains(t, block.Reason, "digoxin toxicity")

	t.Logf("✅ KB-16 Test PASSED: Hypokalemia + Digoxin → HARD_STOP")
	t.Logf("   Lab: K+ = 3.0 mmol/L (threshold: 3.5)")
	t.Logf("   Medication: Digoxin (Cardiac Glycoside)")
}

// TestKB16_NormalLabsNoBlock tests normal labs should NOT generate blocks
func TestKB16_NormalLabsNoBlock(t *testing.T) {
	engine := createTestEngine()

	// Patient labs with all normal values
	patientLabs := []advisor.LabValue{
		{
			Code:    "2823-3", // Potassium
			Display: "Potassium [Moles/volume] in Serum or Plasma",
			Value:   4.5, // Normal range
			Unit:    "mmol/L",
		},
		{
			Code:    "33914-3", // eGFR
			Display: "Glomerular filtration rate/1.73 sq M.predicted",
			Value:   75, // Normal
			Unit:    "mL/min/1.73m2",
		},
	}

	// Propose medications that would normally be blocked with abnormal labs
	proposedMeds := []advisor.ClinicalCode{
		{System: "RxNorm", Code: "29046", Display: "Lisinopril 10mg"}, // ACE inhibitor
		{System: "RxNorm", Code: "6809", Display: "Metformin 500mg"},  // Biguanide
	}

	// Execute lab hard block check
	hardBlocks := engine.TestProcessLabHardBlocks(proposedMeds, patientLabs)

	// Verify NO hard blocks were generated
	assert.Empty(t, hardBlocks, "Should NOT generate any blocks for normal lab values")

	t.Logf("✅ KB-16 Test PASSED: Normal Labs → No Blocks")
	t.Logf("   Lab: K+ = 4.5 mmol/L, eGFR = 75 mL/min/1.73m2")
	t.Logf("   Medications: Lisinopril, Metformin")
}

// TestKB16_ElevatedALTStatin tests ALT > 3x ULN + Statin
func TestKB16_ElevatedALTStatin(t *testing.T) {
	engine := createTestEngine()

	// Patient labs with significantly elevated ALT
	patientLabs := []advisor.LabValue{
		{
			Code:    "1742-6", // LOINC for ALT
			Display: "Alanine aminotransferase [Enzymatic activity/volume] in Serum or Plasma",
			Value:   150, // Above 120 threshold (3x ULN of 40)
			Unit:    "U/L",
		},
	}

	// Propose Atorvastatin
	proposedMeds := []advisor.ClinicalCode{
		{System: "RxNorm", Code: "83367", Display: "Atorvastatin 40mg"},
	}

	// Execute lab hard block check
	hardBlocks := engine.TestProcessLabHardBlocks(proposedMeds, patientLabs)

	// Verify hard block was generated
	require.NotEmpty(t, hardBlocks, "Should generate hard block for ALT 150 + Statin")

	block := hardBlocks[0]
	assert.Equal(t, "LAB_CONTRAINDICATION", block.BlockType)
	assert.Contains(t, block.Reason, "ALT")
	assert.Contains(t, block.Reason, "hepatotoxicity")

	t.Logf("✅ KB-16 Test PASSED: Elevated ALT + Statin → HARD_STOP")
	t.Logf("   Lab: ALT = 150 U/L (threshold: 120)")
	t.Logf("   Medication: Atorvastatin (Statin)")
}

// =============================================================================
// KB-16 Governance Event Tests
// Tests that governance events are properly generated from hard blocks
// =============================================================================

// TestKB16_GovernanceEventGeneration tests governance events are created from blocks
func TestKB16_GovernanceEventGeneration(t *testing.T) {
	engine := createTestEngine()

	// Patient labs with critical hyperkalemia
	patientLabs := []advisor.LabValue{
		{
			Code:    "2823-3",
			Display: "Potassium",
			Value:   6.2,
			Unit:    "mmol/L",
		},
	}

	proposedMeds := []advisor.ClinicalCode{
		{System: "RxNorm", Code: "29046", Display: "Lisinopril 10mg"},
	}

	// Get hard blocks first
	hardBlocks := engine.TestProcessLabHardBlocks(proposedMeds, patientLabs)
	require.NotEmpty(t, hardBlocks, "Should have hard blocks for governance event test")

	// Generate governance events
	patientID := uuid.New()
	providerID := "PROVIDER-001"
	events := engine.TestGenerateGovernanceEvents(hardBlocks, patientID, providerID)

	// Verify governance events were generated
	require.NotEmpty(t, events, "Should generate governance events from hard blocks")

	event := events[0]
	assert.Equal(t, advisor.GovernanceEventLabContraindication, event.EventType)
	assert.Equal(t, patientID, event.PatientID)
	assert.Equal(t, providerID, event.ProviderID)
	assert.Equal(t, "KB-16", event.KBSource)
	assert.NotEmpty(t, event.HashChain, "Should have immutable hash chain")
	assert.True(t, event.RequiresAck)

	t.Logf("✅ KB-16 Governance Test PASSED: Hard Block → Governance Event")
	t.Logf("   Event Type: %s", event.EventType)
	t.Logf("   Hash Chain: %s...", event.HashChain[:16])
	t.Logf("   Requires Ack: %v", event.RequiresAck)
}

// TestKB16_KB14TaskGeneration tests KB-14 tasks are created from lab safety blocks
func TestKB16_KB14TaskGeneration(t *testing.T) {
	engine := createTestEngine()

	// Patient labs with critical hyperkalemia
	patientLabs := []advisor.LabValue{
		{
			Code:    "2823-3",
			Display: "Potassium",
			Value:   6.2,
			Unit:    "mmol/L",
		},
	}

	proposedMeds := []advisor.ClinicalCode{
		{System: "RxNorm", Code: "29046", Display: "Lisinopril 10mg"},
	}

	// Get hard blocks first
	hardBlocks := engine.TestProcessLabHardBlocks(proposedMeds, patientLabs)
	require.NotEmpty(t, hardBlocks, "Should have hard blocks for KB-14 task test")

	// Generate KB-14 tasks
	patientID := uuid.New().String()
	tasks := engine.TestGenerateLabSafetyTasks(hardBlocks, patientID)

	// Verify KB-14 tasks were generated
	require.NotEmpty(t, tasks, "Should generate KB-14 tasks from KB-16 blocks")

	// Should have 3 tasks per block: monitoring, provider notification, recheck
	require.GreaterOrEqual(t, len(tasks), 3, "Should generate at least 3 tasks per block")

	// Check monitoring task
	monitoringTask := findTaskByType(tasks, "LAB_SAFETY_MONITORING")
	require.NotNil(t, monitoringTask, "Should have LAB_SAFETY_MONITORING task")
	assert.Equal(t, "CRITICAL", monitoringTask.Priority)
	assert.Equal(t, "KB-14", monitoringTask.Source)
	assert.Equal(t, "Pharmacist", monitoringTask.AssignedRole)
	assert.Equal(t, 30, monitoringTask.DueInMinutes) // 30 min for critical

	// Check provider notification task
	notifyTask := findTaskByType(tasks, "PROVIDER_NOTIFICATION")
	require.NotNil(t, notifyTask, "Should have PROVIDER_NOTIFICATION task")
	assert.Equal(t, "HIGH", notifyTask.Priority)
	assert.Equal(t, "Care Coordinator", notifyTask.AssignedRole)

	// Check recheck labs task
	recheckTask := findTaskByType(tasks, "RECHECK_LABS")
	require.NotNil(t, recheckTask, "Should have RECHECK_LABS task")
	assert.Equal(t, "MEDIUM", recheckTask.Priority)
	assert.Equal(t, "Nurse", recheckTask.AssignedRole)

	t.Logf("✅ KB-14 Task Generation Test PASSED")
	t.Logf("   Total Tasks: %d", len(tasks))
	t.Logf("   Monitoring Due: %d minutes", monitoringTask.DueInMinutes)
	t.Logf("   Provider Notify Due: %d minutes", notifyTask.DueInMinutes)
	t.Logf("   Recheck Labs Due: %d minutes", recheckTask.DueInMinutes)
}

// =============================================================================
// Test Helper Functions
// =============================================================================

// floatPtr helper for creating float64 pointers
func floatPtr(v float64) *float64 {
	return &v
}

// Helper to ensure computed scores are set
func createComputedScores(egfr float64) snapshot.ComputedScores {
	return snapshot.ComputedScores{
		EGFR: floatPtr(egfr),
	}
}

// findTaskByType finds a task by its type in the task slice
func findTaskByType(tasks []advisor.GeneratedTask, taskType string) *advisor.GeneratedTask {
	for i := range tasks {
		if tasks[i].TaskType == taskType {
			return &tasks[i]
		}
	}
	return nil
}
