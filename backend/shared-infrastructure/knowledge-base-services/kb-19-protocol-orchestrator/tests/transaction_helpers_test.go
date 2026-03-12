// Package tests provides unit tests for KB-19 Protocol Orchestrator.
//
// Tests for transaction package pure functions:
// - Validator: severityTriggersBlock, DetermineDisposition, EvaluateExcludedDrugs,
//   isHardBlockSeverity, mapToHardBlockType, isPregnancyCode, isPregnancyRelatedBlock,
//   extractFDACategory, containsIgnoreCase, convertDDIRisksToBlocks, convertLabRisksToBlocks,
//   convertAllergyRisksToBlocks, convertMedicationRisksToBlocks
// - Committer: determineGovernanceEventType, ComputeEventHash, GenerateLabSafetyTasks,
//   GenerateGovernanceEvents, getDispositionReason, BuildAuditTrail
package tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-19-protocol-orchestrator/internal/transaction"
)

// ============================================================================
// Validator: DetermineDisposition
// ============================================================================

func TestDetermineDisposition(t *testing.T) {
	v := transaction.NewValidator()

	t.Run("hard blocks → HARD_STOP", func(t *testing.T) {
		blocks := []transaction.HardBlock{{ID: uuid.New(), BlockType: "DDI_SEVERE"}}
		disp := v.DetermineDisposition(blocks, nil, nil)
		assert.Equal(t, transaction.DispositionHardStop, disp)
	})

	t.Run("no proposals → RECALCULATE", func(t *testing.T) {
		disp := v.DetermineDisposition(nil, nil, nil)
		assert.Equal(t, transaction.DispositionRecalculate, disp)
	})

	t.Run("no proposals empty slice → RECALCULATE", func(t *testing.T) {
		disp := v.DetermineDisposition(nil, []transaction.MedicationProposal{}, nil)
		assert.Equal(t, transaction.DispositionRecalculate, disp)
	})

	t.Run("critical warning → HOLD_FOR_REVIEW", func(t *testing.T) {
		proposals := []transaction.MedicationProposal{
			{
				Warnings: []transaction.ProposalWarning{
					{Severity: "critical", Message: "test"},
				},
				QualityFactors: transaction.QualityFactors{Safety: 0.9},
			},
		}
		disp := v.DetermineDisposition(nil, proposals, nil)
		assert.Equal(t, transaction.DispositionHoldForReview, disp)
	})

	t.Run("low safety score → HOLD_FOR_APPROVAL", func(t *testing.T) {
		proposals := []transaction.MedicationProposal{
			{
				QualityFactors: transaction.QualityFactors{Safety: 0.5},
			},
		}
		disp := v.DetermineDisposition(nil, proposals, nil)
		assert.Equal(t, transaction.DispositionHoldForApproval, disp)
	})

	t.Run("safety exactly 0.6 → DISPENSE (not <0.6)", func(t *testing.T) {
		proposals := []transaction.MedicationProposal{
			{
				QualityFactors: transaction.QualityFactors{Safety: 0.6},
			},
		}
		disp := v.DetermineDisposition(nil, proposals, nil)
		assert.Equal(t, transaction.DispositionDispense, disp)
	})

	t.Run("all clear → DISPENSE", func(t *testing.T) {
		proposals := []transaction.MedicationProposal{
			{
				QualityFactors: transaction.QualityFactors{Safety: 0.9},
			},
		}
		disp := v.DetermineDisposition(nil, proposals, nil)
		assert.Equal(t, transaction.DispositionDispense, disp)
	})
}

// ============================================================================
// Validator: V3 Risk-to-Block Conversion (via ValidateTransactionV3)
// ============================================================================

func TestValidateTransactionV3_DDIRisks(t *testing.T) {
	v := transaction.NewValidator()
	txn := &transaction.Transaction{ID: uuid.New(), State: transaction.StateCreated}

	riskProfile := &transaction.RiskProfile{
		DDIRisks: []transaction.DDIRisk{
			{
				Drug1Code:      "12345",
				Drug1Name:      "Warfarin",
				Drug2Code:      "67890",
				Drug2Name:      "Aspirin",
				Severity:       "major", // In default thresholds
				ClinicalEffect: "Increased bleeding risk",
			},
			{
				Drug1Code:      "11111",
				Drug1Name:      "Metformin",
				Drug2Code:      "22222",
				Drug2Name:      "Lisinopril",
				Severity:       "minor", // NOT in default thresholds
				ClinicalEffect: "Minimal interaction",
			},
		},
	}

	err := v.ValidateTransactionV3(nil, txn, riskProfile)
	require.NoError(t, err)

	assert.Len(t, txn.HardBlocks, 1, "only 'major' severity should trigger block")
	assert.Equal(t, "DDI_SEVERE", txn.HardBlocks[0].BlockType)
	assert.Equal(t, transaction.StateBlocked, txn.State)
}

func TestValidateTransactionV3_LabRisks(t *testing.T) {
	v := transaction.NewValidator()
	txn := &transaction.Transaction{ID: uuid.New(), State: transaction.StateCreated}

	riskProfile := &transaction.RiskProfile{
		LabRisks: []transaction.LabRisk{
			{
				RxNormCode:     "12345",
				DrugName:       "Metformin",
				LOINCCode:      "2160-0",
				LabName:        "Creatinine",
				CurrentValue:   3.5,
				ThresholdValue: 1.5,
				ThresholdOp:    ">",
				Severity:       "contraindicated",
				ClinicalRisk:   "Lactic acidosis risk",
			},
		},
	}

	err := v.ValidateTransactionV3(nil, txn, riskProfile)
	require.NoError(t, err)
	assert.Len(t, txn.HardBlocks, 1)
	assert.Equal(t, "LAB_CONTRAINDICATION", txn.HardBlocks[0].BlockType)
	assert.Equal(t, transaction.StateBlocked, txn.State)
}

func TestValidateTransactionV3_AllergyRisks(t *testing.T) {
	v := transaction.NewValidator()
	txn := &transaction.Transaction{ID: uuid.New(), State: transaction.StateCreated}

	riskProfile := &transaction.RiskProfile{
		AllergyRisks: []transaction.AllergyRisk{
			{
				RxNormCode:      "12345",
				DrugName:        "Amoxicillin",
				AllergenCode:    "PCN",
				AllergenName:    "Penicillin",
				Severity:        "severe",
				ReactionType:    "Anaphylaxis",
				IsCrossReactive: true,
			},
		},
	}

	err := v.ValidateTransactionV3(nil, txn, riskProfile)
	require.NoError(t, err)
	assert.Len(t, txn.HardBlocks, 1)
	assert.Equal(t, "ALLERGY_CONTRAINDICATION", txn.HardBlocks[0].BlockType)
	assert.Contains(t, txn.HardBlocks[0].Reason, "cross-reactive")
}

func TestValidateTransactionV3_NoRisks_Validated(t *testing.T) {
	v := transaction.NewValidator()
	txn := &transaction.Transaction{ID: uuid.New(), State: transaction.StateCreated}

	riskProfile := &transaction.RiskProfile{}

	err := v.ValidateTransactionV3(nil, txn, riskProfile)
	require.NoError(t, err)
	assert.Empty(t, txn.HardBlocks)
	assert.Equal(t, transaction.StateValidated, txn.State)
}

func TestValidateTransactionV3_MedicationRisks(t *testing.T) {
	v := transaction.NewValidator()
	txn := &transaction.Transaction{ID: uuid.New(), State: transaction.StateCreated}

	riskProfile := &transaction.RiskProfile{
		MedicationRisks: []transaction.MedicationRisk{
			{
				RxNormCode:      "99999",
				DrugName:        "Insulin",
				OverallRisk:     0.9, // Above 0.8 cutoff
				RiskCategory:    "CRITICAL",
				IsHighAlert:     true,
				HasBlackBoxWarn: false,
				RiskFactors:     []transaction.RiskFactor{{Type: "AGE", Severity: "severe", Description: "Elderly"}},
			},
			{
				RxNormCode:   "88888",
				DrugName:     "Lisinopril",
				OverallRisk:  0.3, // Below 0.8 cutoff
				RiskCategory: "LOW",
			},
		},
	}

	err := v.ValidateTransactionV3(nil, txn, riskProfile)
	require.NoError(t, err)
	assert.Len(t, txn.HardBlocks, 1, "only high-risk Insulin should trigger")
	assert.Equal(t, "HIGH_ALERT_MEDICATION", txn.HardBlocks[0].BlockType)
}

func TestValidateTransactionV3_BlackBoxWarning(t *testing.T) {
	v := transaction.NewValidator()
	txn := &transaction.Transaction{ID: uuid.New(), State: transaction.StateCreated}

	riskProfile := &transaction.RiskProfile{
		MedicationRisks: []transaction.MedicationRisk{
			{
				RxNormCode:      "77777",
				DrugName:        "Rosiglitazone",
				OverallRisk:     0.85,
				RiskCategory:    "CRITICAL",
				HasBlackBoxWarn: true,
			},
		},
	}

	err := v.ValidateTransactionV3(nil, txn, riskProfile)
	require.NoError(t, err)
	assert.Equal(t, "BLACK_BOX_WARNING", txn.HardBlocks[0].BlockType)
}

// ============================================================================
// Committer: GenerateGovernanceEvents
// ============================================================================

func TestGenerateGovernanceEvents(t *testing.T) {
	c := transaction.NewCommitter()

	blocks := []transaction.HardBlock{
		{ID: uuid.New(), BlockType: "DDI_SEVERE", Severity: "major",
			Medication: transaction.ClinicalCode{Code: "12345", Display: "Warfarin"},
			KBSource: "KB-5", RuleID: "DDI-001", RequiresAck: true},
		{ID: uuid.New(), BlockType: "LAB_CONTRAINDICATION", Severity: "contraindicated",
			Medication: transaction.ClinicalCode{Code: "67890", Display: "Metformin"},
			KBSource: "KB-16", RuleID: "LAB-001", RequiresAck: true},
	}

	events := c.GenerateGovernanceEvents(blocks, uuid.New(), "DR-SMITH", "")

	assert.Len(t, events, 2)
	// First event should be DDI type
	assert.Equal(t, transaction.GovernanceEventDDIHardStop, events[0].EventType)
	// Second event should be Lab Contraindication
	assert.Equal(t, transaction.GovernanceEventLabContraindication, events[1].EventType)

	// Hash chain: second event hash should differ from first
	assert.NotEmpty(t, events[0].HashChain)
	assert.NotEmpty(t, events[1].HashChain)
	assert.NotEqual(t, events[0].HashChain, events[1].HashChain)

	// All events should not be acknowledged yet
	for _, ev := range events {
		assert.False(t, ev.Acknowledged)
		assert.True(t, ev.RequiresAck)
	}
}

func TestGenerateGovernanceEvents_EmptyBlocks(t *testing.T) {
	c := transaction.NewCommitter()
	events := c.GenerateGovernanceEvents(nil, uuid.New(), "DR-SMITH", "")
	assert.Empty(t, events)
}

// ============================================================================
// Committer: GenerateLabSafetyTasks
// ============================================================================

func TestGenerateLabSafetyTasks(t *testing.T) {
	c := transaction.NewCommitter()
	patientID := uuid.New().String()

	blocks := []transaction.HardBlock{
		{
			ID:        uuid.New(),
			BlockType: "LAB_CONTRAINDICATION",
			Medication: transaction.ClinicalCode{Code: "12345", Display: "Metformin"},
			TriggerCondition: transaction.ClinicalCode{Code: "2160-0", Display: "Creatinine"},
			Reason:    "Creatinine too high",
			KBSource:  "KB-16",
			RuleID:    "LAB-001",
		},
		{
			// Non-KB-16 block should be ignored
			ID:        uuid.New(),
			BlockType: "DDI_SEVERE",
			KBSource:  "KB-5",
		},
	}

	tasks := c.GenerateLabSafetyTasks(blocks, patientID)

	// Only KB-16 blocks generate tasks, and each generates 3 tasks
	assert.Len(t, tasks, 3)

	assert.Equal(t, "LAB_SAFETY_MONITORING", tasks[0].TaskType)
	assert.Equal(t, "CRITICAL", tasks[0].Priority)
	assert.Equal(t, 30, tasks[0].DueInMinutes)

	assert.Equal(t, "PROVIDER_NOTIFICATION", tasks[1].TaskType)
	assert.Equal(t, "HIGH", tasks[1].Priority)

	assert.Equal(t, "RECHECK_LABS", tasks[2].TaskType)
	assert.Equal(t, "MEDIUM", tasks[2].Priority)
	assert.Equal(t, 1440, tasks[2].DueInMinutes, "24 hours for lab recheck")
}

func TestGenerateLabSafetyTasks_NoKB16Blocks(t *testing.T) {
	c := transaction.NewCommitter()
	blocks := []transaction.HardBlock{
		{ID: uuid.New(), KBSource: "KB-5"},
	}
	tasks := c.GenerateLabSafetyTasks(blocks, "patient-1")
	assert.Empty(t, tasks)
}

// ============================================================================
// Committer: ComputeEventHash
// ============================================================================

func TestComputeEventHash_Deterministic(t *testing.T) {
	c := transaction.NewCommitter()

	event := transaction.GovernanceEvent{
		ID:             uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		EventType:      transaction.GovernanceEventDDIHardStop,
		Severity:       "major",
		Timestamp:      time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC),
		PatientID:      uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		MedicationCode: "12345",
		TriggerCode:    "67890",
		BlockType:      "DDI_SEVERE",
		KBSource:       "KB-5",
		RuleID:         "DDI-001",
	}

	hash1 := c.ComputeEventHash(event, "prev-hash")
	hash2 := c.ComputeEventHash(event, "prev-hash")

	assert.Equal(t, hash1, hash2, "same input should produce same hash")
	assert.Len(t, hash1, 64, "SHA256 hex should be 64 chars")
}

func TestComputeEventHash_DifferentPreviousHash(t *testing.T) {
	c := transaction.NewCommitter()

	event := transaction.GovernanceEvent{
		ID:        uuid.New(),
		Timestamp: time.Now(),
	}

	hash1 := c.ComputeEventHash(event, "prev-A")
	hash2 := c.ComputeEventHash(event, "prev-B")

	assert.NotEqual(t, hash1, hash2, "different previous hash → different output")
}

// ============================================================================
// Committer: BuildAuditTrail
// ============================================================================

func TestBuildAuditTrail(t *testing.T) {
	c := transaction.NewCommitter()

	snapshotID := uuid.New()
	envelopeID := uuid.New()
	blocks := []transaction.HardBlock{
		{ID: uuid.New(), AckText: "I acknowledge this risk"},
	}
	proposals := []transaction.MedicationProposal{
		{Warnings: []transaction.ProposalWarning{{Message: "w1"}, {Message: "w2"}}},
	}
	kbVersions := map[string]string{"KB-5": "1.2.0", "KB-16": "2.0.0"}
	workflowResult := &transaction.WorkflowResult{
		InferenceChain: []transaction.InferenceStep{
			{RuleID: "R1", Description: "Step 1"},
			{RuleID: "R2", Description: "Step 2"},
			{RuleID: "", Description: "no rule"},
		},
	}

	audit := c.BuildAuditTrail(snapshotID, envelopeID, "session-123", blocks, proposals, workflowResult, kbVersions, transaction.DispositionHardStop)

	assert.Contains(t, audit.TraceID, snapshotID.String()[:8])
	assert.Equal(t, "session-123", audit.SessionID)
	assert.Equal(t, 1, audit.BlocksGenerated)
	assert.Equal(t, 2, audit.WarningsGenerated)
	assert.Equal(t, 2, audit.RulesEvaluated, "only steps with RuleID count")
	assert.True(t, audit.RequiresAck)
	assert.NotEmpty(t, audit.AuditHash)
	assert.NotEmpty(t, audit.Timestamp)

	// Governance level should be Tier 7 for HARD_STOP with blocks
	assert.Equal(t, "TIER_7_COMPLETE", audit.GovernanceLevel)
	assert.Equal(t, "COMPLIANT", audit.ComplianceStatus)
}

func TestBuildAuditTrail_NoBlocks_Compliant(t *testing.T) {
	c := transaction.NewCommitter()

	audit := c.BuildAuditTrail(uuid.New(), uuid.New(), "s1", nil, nil, nil, nil, transaction.DispositionDispense)

	assert.Equal(t, 0, audit.BlocksGenerated)
	assert.False(t, audit.RequiresAck)
	assert.Equal(t, "TIER_7_COMPLETE", audit.GovernanceLevel)
	assert.Equal(t, "COMPLIANT", audit.ComplianceStatus)
}

func TestBuildAuditTrail_BlocksWithNonHardStop_RequiresReview(t *testing.T) {
	c := transaction.NewCommitter()

	blocks := []transaction.HardBlock{{ID: uuid.New(), AckText: "ack"}}
	audit := c.BuildAuditTrail(uuid.New(), uuid.New(), "s1", blocks, nil, nil, nil, transaction.DispositionDispense)

	assert.Equal(t, "TIER_6_PARTIAL", audit.GovernanceLevel)
	assert.Equal(t, "REQUIRES_REVIEW", audit.ComplianceStatus)
}

// ============================================================================
// Validator: IsV3Enabled
// ============================================================================

func TestValidatorIsV3Enabled(t *testing.T) {
	v := transaction.NewValidator()
	assert.False(t, v.IsV3Enabled(), "default should be false")

	// Setting just the config flag without a provider is not enough
	v2 := transaction.NewValidatorWithConfig(transaction.ValidatorConfig{UseV3RiskProfiles: true})
	assert.False(t, v2.IsV3Enabled(), "no provider → false even with config")
}

// ============================================================================
// Helper function tests via exported behaviors
// ============================================================================

func TestEvaluateExcludedDrugs(t *testing.T) {
	v := transaction.NewValidator()

	excluded := []transaction.ExcludedDrug{
		{
			Medication: transaction.ClinicalCode{Code: "12345", Display: "Warfarin", System: "RxNorm"},
			Reason:     "Contraindicated in pregnancy - FDA Category X teratogenic",
			Severity:   "absolute",
			KBSource:   "KB-1",
			RuleID:     "PREG-001",
		},
		{
			Medication: transaction.ClinicalCode{Code: "67890", Display: "Ibuprofen", System: "RxNorm"},
			Reason:     "Mild GI risk",
			Severity:   "moderate",
			KBSource:   "KB-1",
			RuleID:     "GI-001",
		},
	}

	patientCtx := transaction.PatientContext{
		PatientID:  uuid.New(),
		IsPregnant: true,
		Conditions: []transaction.ClinicalCode{
			{Code: "77386006", System: "SNOMED", Display: "Pregnancy"},
		},
	}

	blocks, excludedInfo := v.EvaluateExcludedDrugs(excluded, patientCtx)

	// First drug should be a hard block (severity=absolute)
	assert.Len(t, blocks, 1, "only 'absolute' severity triggers hard block")
	assert.Equal(t, "CONTRAINDICATION", blocks[0].BlockType)
	assert.Contains(t, blocks[0].Reason, "teratogenic")
	assert.Equal(t, "X", blocks[0].FDACategory, "should extract FDA Category X")

	// Both should be in excluded info
	assert.Len(t, excludedInfo, 2)
	assert.True(t, excludedInfo[0].IsHardBlock)
	assert.False(t, excludedInfo[1].IsHardBlock)
}

func TestEvaluateExcludedDrugs_NonPregnancy(t *testing.T) {
	v := transaction.NewValidator()

	excluded := []transaction.ExcludedDrug{
		{
			Medication: transaction.ClinicalCode{Code: "11111", Display: "TestDrug"},
			Reason:     "Severe drug interaction",
			Severity:   "severe",
		},
	}

	patientCtx := transaction.PatientContext{PatientID: uuid.New()}

	blocks, _ := v.EvaluateExcludedDrugs(excluded, patientCtx)
	assert.Len(t, blocks, 1)
	assert.Equal(t, "DDI_SEVERE", blocks[0].BlockType)
	assert.Empty(t, blocks[0].FDACategory, "no pregnancy → no FDA category")
}
