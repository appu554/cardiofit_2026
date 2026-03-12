// Package transaction provides transaction commit operations for KB-19.
// Committer MOVED FROM: medication-advisor-engine/advisor/engine.go
// as part of V3 architecture refactoring.
package transaction

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// COMMITTER
// Handles transaction commit with governance events and audit trail
// =============================================================================

// Committer handles transaction commit operations
type Committer struct {
	// Configuration for commit operations
	config CommitterConfig
}

// CommitterConfig holds configuration for the committer
type CommitterConfig struct {
	EnableGovernanceEvents bool
	EnableAuditTrail       bool
	EnableTaskGeneration   bool
	GovernanceTier         int // Default tier for governance events
}

// NewCommitter creates a new committer with default configuration
func NewCommitter() *Committer {
	return &Committer{
		config: CommitterConfig{
			EnableGovernanceEvents: true,
			EnableAuditTrail:       true,
			EnableTaskGeneration:   true,
			GovernanceTier:         7, // Tier-7 compliance by default
		},
	}
}

// NewCommitterWithConfig creates a committer with custom configuration
func NewCommitterWithConfig(cfg CommitterConfig) *Committer {
	return &Committer{config: cfg}
}

// CommitTransaction finalizes a transaction with governance events and audit trail
func (c *Committer) CommitTransaction(
	ctx context.Context,
	txn *Transaction,
	providerID string,
	kbVersions map[string]string,
) error {
	// Update transaction state
	txn.State = StateCommitting

	// Generate governance events from hard blocks
	if c.config.EnableGovernanceEvents && len(txn.HardBlocks) > 0 {
		txn.GovernanceEvents = c.GenerateGovernanceEvents(
			txn.HardBlocks,
			txn.PatientID,
			providerID,
			"", // No previous hash for new transaction
		)
	}

	// Generate follow-up tasks
	if c.config.EnableTaskGeneration && len(txn.HardBlocks) > 0 {
		txn.GeneratedTasks = c.GenerateLabSafetyTasks(
			txn.HardBlocks,
			txn.PatientID.String(),
		)
	}

	// Build audit trail
	if c.config.EnableAuditTrail {
		// Build audit trail with workflow result
		auditTrail := c.BuildAuditTrail(
			txn.ID,            // Use transaction ID as snapshot ID
			txn.ID,            // Use transaction ID as envelope ID
			"session-"+txn.ID.String()[:8],
			txn.HardBlocks,
			[]MedicationProposal{}, // Empty proposals for now
			&WorkflowResult{},      // Empty workflow result
			kbVersions,
			txn.Disposition,
		)
		txn.AuditTrail = &auditTrail
	}

	// Update state to committed
	txn.State = StateCommitted
	txn.CommittedAt = timePtr(time.Now())
	txn.CommittedBy = providerID

	return nil
}

// =============================================================================
// MOVED FROM: medication-advisor-engine/advisor/engine.go lines 566-606
// =============================================================================

// GenerateGovernanceEvents creates governance events from hard blocks for Tier-7 compliance.
// Every KB-16 lab safety block MUST generate a corresponding governance event with:
// - Event type (POLICY_VIOLATION, PATIENT_SAFETY_RISK, LAB_CONTRAINDICATION)
// - Immutable hash chain for audit trail
// - KB-14 task linkage for follow-up
func (c *Committer) GenerateGovernanceEvents(
	hardBlocks []HardBlock,
	patientID uuid.UUID,
	providerID string,
	previousHash string,
) []GovernanceEvent {
	var events []GovernanceEvent
	currentHash := previousHash

	for _, block := range hardBlocks {
		// Determine event type based on block source and type
		eventType := c.determineGovernanceEventType(block)

		// Create governance event
		event := GovernanceEvent{
			ID:             uuid.New(),
			EventType:      eventType,
			Severity:       block.Severity,
			Timestamp:      time.Now(),
			PatientID:      patientID,
			ProviderID:     providerID,
			MedicationCode: block.Medication.Code,
			TriggerCode:    block.TriggerCondition.Code,
			TriggerValue:   block.TriggerCondition.Display,
			BlockType:      block.BlockType,
			KBSource:       block.KBSource,
			RuleID:         block.RuleID,
			Description:    block.Reason,
			RequiresAck:    block.RequiresAck,
			Acknowledged:   false,
		}

		// Compute immutable hash chain
		event.HashChain = c.ComputeEventHash(event, currentHash)
		currentHash = event.HashChain

		events = append(events, event)
	}

	return events
}

// =============================================================================
// MOVED FROM: medication-advisor-engine/advisor/engine.go lines 608-629
// =============================================================================

// determineGovernanceEventType maps hard block types to governance event types
func (c *Committer) determineGovernanceEventType(block HardBlock) GovernanceEventType {
	switch block.BlockType {
	case "LAB_CONTRAINDICATION":
		return GovernanceEventLabContraindication
	case "DDI_SEVERE":
		return GovernanceEventDDIHardStop
	case "BLACK_BOX":
		return GovernanceEventBlackBoxWarning
	case "PREGNANCY_CONTRAINDICATION":
		return GovernanceEventPregnancyRisk
	case "CONTRAINDICATION":
		// Sub-classify based on KB source
		if block.KBSource == "KB-16" {
			return GovernanceEventLabContraindication
		}
		return GovernanceEventPatientSafetyRisk
	default:
		// Default to policy violation for unclassified blocks
		return GovernanceEventPolicyViolation
	}
}

// =============================================================================
// MOVED FROM: medication-advisor-engine/advisor/engine.go lines 631-668
// =============================================================================

// ComputeEventHash creates an immutable SHA256 hash for the governance event chain
func (c *Committer) ComputeEventHash(event GovernanceEvent, previousHash string) string {
	// Create hash input structure (excluding HashChain field)
	hashInput := struct {
		ID             string    `json:"id"`
		EventType      string    `json:"event_type"`
		Severity       string    `json:"severity"`
		Timestamp      time.Time `json:"timestamp"`
		PatientID      string    `json:"patient_id"`
		MedicationCode string    `json:"medication_code"`
		TriggerCode    string    `json:"trigger_code"`
		BlockType      string    `json:"block_type"`
		KBSource       string    `json:"kb_source"`
		RuleID         string    `json:"rule_id"`
		PreviousHash   string    `json:"previous_hash"`
	}{
		ID:             event.ID.String(),
		EventType:      string(event.EventType),
		Severity:       event.Severity,
		Timestamp:      event.Timestamp,
		PatientID:      event.PatientID.String(),
		MedicationCode: event.MedicationCode,
		TriggerCode:    event.TriggerCode,
		BlockType:      event.BlockType,
		KBSource:       event.KBSource,
		RuleID:         event.RuleID,
		PreviousHash:   previousHash,
	}

	jsonBytes, err := json.Marshal(hashInput)
	if err != nil {
		// Fallback to simple hash on marshal error
		return fmt.Sprintf("hash-error-%s", event.ID.String())
	}

	hash := sha256.Sum256(jsonBytes)
	return hex.EncodeToString(hash[:])
}

// =============================================================================
// MOVED FROM: medication-advisor-engine/advisor/engine.go lines 672-744
// =============================================================================

// GenerateLabSafetyTasks creates KB-14 mandatory tasks for lab safety violations.
// Every KB-16 lab contraindication MUST generate follow-up tasks per governance requirements.
func (c *Committer) GenerateLabSafetyTasks(
	hardBlocks []HardBlock,
	patientID string,
) []GeneratedTask {
	var tasks []GeneratedTask

	for _, block := range hardBlocks {
		// Only generate tasks for KB-16 lab-based blocks
		if block.KBSource != "KB-16" {
			continue
		}

		// Mandatory monitoring task for lab contraindication
		tasks = append(tasks, GeneratedTask{
			TaskType:     "LAB_SAFETY_MONITORING",
			Priority:     "CRITICAL",
			Title:        fmt.Sprintf("URGENT: Lab Contraindication - %s", block.Medication.Display),
			Description:  fmt.Sprintf("Lab value contraindication detected. %s. Immediate review required before medication administration.", block.Reason),
			DueInMinutes: 30, // 30 minutes for critical lab safety
			AssignedRole: "Pharmacist",
			Source:       "KB-14",
			Metadata: map[string]interface{}{
				"medication_code":   block.Medication.Code,
				"medication_name":   block.Medication.Display,
				"trigger_code":      block.TriggerCondition.Code,
				"trigger_value":     block.TriggerCondition.Display,
				"block_type":        block.BlockType,
				"severity":          block.Severity,
				"rule_id":           block.RuleID,
				"patient_id":        patientID,
				"requires_ack":      block.RequiresAck,
				"governance_source": "KB-16",
			},
		})

		// Provider notification task
		tasks = append(tasks, GeneratedTask{
			TaskType:     "PROVIDER_NOTIFICATION",
			Priority:     "HIGH",
			Title:        fmt.Sprintf("Notify Prescriber: Lab Block on %s", block.Medication.Display),
			Description:  fmt.Sprintf("Prescriber notification required for lab-based medication block. %s", block.Reason),
			DueInMinutes: 60, // 1 hour for provider notification
			AssignedRole: "Care Coordinator",
			Source:       "KB-14",
			Metadata: map[string]interface{}{
				"medication_code": block.Medication.Code,
				"medication_name": block.Medication.Display,
				"block_reason":    block.Reason,
				"patient_id":      patientID,
				"ack_text":        block.AckText,
			},
		})

		// Re-check lab task (if lab values may change)
		tasks = append(tasks, GeneratedTask{
			TaskType:     "RECHECK_LABS",
			Priority:     "MEDIUM",
			Title:        fmt.Sprintf("Recheck %s for %s clearance", block.TriggerCondition.Display, block.Medication.Display),
			Description:  fmt.Sprintf("Schedule repeat lab test to determine if %s can be safely initiated after lab normalization.", block.Medication.Display),
			DueInMinutes: 1440, // 24 hours for lab recheck
			AssignedRole: "Nurse",
			Source:       "KB-14",
			Metadata: map[string]interface{}{
				"lab_code":        block.TriggerCondition.Code,
				"lab_display":     block.TriggerCondition.Display,
				"medication_code": block.Medication.Code,
				"patient_id":      patientID,
			},
		})
	}

	return tasks
}

// =============================================================================
// MOVED FROM: medication-advisor-engine/advisor/engine.go lines 783-852
// =============================================================================

// BuildAuditTrail creates a Tier-7 compliant audit trail summary
func (c *Committer) BuildAuditTrail(
	snapshotID uuid.UUID,
	envelopeID uuid.UUID,
	sessionID string,
	hardBlocks []HardBlock,
	proposals []MedicationProposal,
	workflowResult *WorkflowResult,
	kbVersions map[string]string,
	disposition DispositionCode,
) AuditTrailSummary {
	// Count warnings across all proposals
	warningCount := 0
	for _, p := range proposals {
		warningCount += len(p.Warnings)
	}

	// Extract KB services used from versions map
	kbServices := make([]string, 0, len(kbVersions))
	for kb := range kbVersions {
		kbServices = append(kbServices, kb)
	}

	// Count rules evaluated from inference chain
	rulesEvaluated := 0
	if workflowResult != nil {
		for _, step := range workflowResult.InferenceChain {
			if step.RuleID != "" {
				rulesEvaluated++
			}
		}
	}

	// Determine governance level
	governanceLevel := "TIER_7_COMPLETE"
	complianceStatus := "COMPLIANT"
	if len(hardBlocks) > 0 && disposition != DispositionHardStop {
		complianceStatus = "REQUIRES_REVIEW"
		governanceLevel = "TIER_6_PARTIAL"
	}

	// Determine disposition reason
	dispositionReason := getDispositionReason(disposition, hardBlocks, proposals)

	// Build acknowledgment text if required
	requiresAck := len(hardBlocks) > 0
	ackText := ""
	if requiresAck && len(hardBlocks) > 0 {
		ackText = hardBlocks[0].AckText
	}

	// Compute audit hash for integrity
	auditHash := computeAuditHash(snapshotID, envelopeID, len(proposals), len(hardBlocks))

	// Count safety checks
	safetyChecks := len(hardBlocks)
	if workflowResult != nil {
		safetyChecks += len(workflowResult.ExcludedDrugs)
	}

	return AuditTrailSummary{
		TraceID:           fmt.Sprintf("TRACE-%s", snapshotID.String()[:8]),
		SessionID:         sessionID,
		TransactionID:     fmt.Sprintf("TXN-%s", envelopeID.String()[:8]),
		EvidenceCount:     rulesEvaluated,
		KBServicesUsed:    kbServices,
		RulesEvaluated:    rulesEvaluated,
		SafetyChecks:      safetyChecks,
		BlocksGenerated:   len(hardBlocks),
		WarningsGenerated: warningCount,
		DispositionReason: dispositionReason,
		RequiresAck:       requiresAck,
		AckText:           ackText,
		GovernanceLevel:   governanceLevel,
		ComplianceStatus:  complianceStatus,
		AuditHash:         auditHash,
		Timestamp:         time.Now().UTC().Format(time.RFC3339),
	}
}

// =============================================================================
// HELPER FUNCTIONS
// MOVED FROM: medication-advisor-engine/advisor/engine.go lines 855-890
// =============================================================================

// getDispositionReason returns a human-readable reason for the disposition
func getDispositionReason(disposition DispositionCode, hardBlocks []HardBlock, proposals []MedicationProposal) string {
	switch disposition {
	case DispositionHardStop:
		if len(hardBlocks) > 0 {
			return fmt.Sprintf("Hard stop required: %s", hardBlocks[0].Reason)
		}
		return "Critical safety issue detected"
	case DispositionHoldForReview:
		return "Critical warnings require pharmacist review"
	case DispositionHoldForApproval:
		return "Safety score below threshold requires physician approval"
	case DispositionRecalculate:
		return "No suitable proposals found - additional information required"
	case DispositionDispense:
		return "All safety checks passed - safe to proceed"
	default:
		return "Disposition determined by clinical rules"
	}
}

// computeAuditHash generates a simple hash for audit trail integrity
func computeAuditHash(snapshotID, envelopeID uuid.UUID, proposalCount, blockCount int) string {
	data := fmt.Sprintf("%s:%s:%d:%d:%d",
		snapshotID.String(),
		envelopeID.String(),
		proposalCount,
		blockCount,
		time.Now().UnixNano(),
	)
	// Simple hash using built-in - in production would use crypto/sha256
	hash := 0
	for _, c := range data {
		hash = hash*31 + int(c)
	}
	return fmt.Sprintf("%016x", uint64(hash))
}

// timePtr returns a pointer to a time.Time value
func timePtr(t time.Time) *time.Time {
	return &t
}
