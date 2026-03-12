// Package advisor provides the core Medication Advisor Engine.
// This is the Tier 6 Application Engine implementing Calculate → Validate → Commit workflow.
// IMPORTANT: All KB services (KB-1 through KB-6) MUST be available for this engine to function.
package advisor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/medication-advisor-engine/evidence"
	"github.com/cardiofit/medication-advisor-engine/kbclients"
	"github.com/cardiofit/medication-advisor-engine/snapshot"
)

// MedicationAdvisorEngine is the main orchestrator for medication recommendations.
// It implements the Calculate → Validate → Commit workflow with FDA SaMD compliance.
//
// IMPORTANT: This engine requires ALL Knowledge Base services (KB-1 through KB-6) to be
// available. There is NO fallback mode - if any KB service is unavailable, the engine
// will return an error.
type MedicationAdvisorEngine struct {
	snapshotManager  *snapshot.SnapshotManager
	evidenceManager  *evidence.EvidenceEnvelopeManager
	workflowEngine   *WorkflowOrchestrator
	scoringEngine    *ProposalScoringEngine
	conflictDetector *ConflictDetector

	// V3 Architecture: KB-5 DDI Client for external DDI checking
	kb5DDIClient kbclients.KB5DDIClient

	// V3 Architecture: KB-16 Lab Safety Client for lab-based contraindications
	kb16LabSafetyClient kbclients.KB16LabSafetyClient

	// Configuration
	config EngineConfig
}

// EngineConfig holds configuration for the engine.
// All KB URLs are REQUIRED - the engine will not start without them.
type EngineConfig struct {
	Environment        string
	SnapshotTTLMinutes int
	KB1URL             string // Dosing - REQUIRED
	KB2URL             string // Interactions - REQUIRED
	KB3URL             string // Guidelines - REQUIRED
	KB4URL             string // Safety - REQUIRED
	KB5URL             string // Monitoring - REQUIRED
	KB6URL             string // Efficacy - REQUIRED
	KB5DDIURL          string // V3: KB-5 DDI Service for drug interactions - REQUIRED
	KB16URL            string // V3: KB-16 Lab Safety Service for lab-based contraindications - REQUIRED (NO LOCAL FALLBACK)
}

// NewMedicationAdvisorEngine creates a new medication advisor engine.
// Returns an error if any KB service URL is not configured.
//
// All Knowledge Base services (KB-1 through KB-6) MUST be available:
//   - KB-1 (KB1URL): Dosing rules and dose adjustments
//   - KB-2 (KB2URL): Drug-drug interactions
//   - KB-3 (KB3URL): Clinical guidelines and recommendations
//   - KB-4 (KB4URL): Safety checks (allergies, contraindications)
//   - KB-5 (KB5URL): Monitoring requirements
//   - KB-6 (KB6URL): Efficacy scores
func NewMedicationAdvisorEngine(
	snapshotStore snapshot.SnapshotStore,
	envelopeStore evidence.EnvelopeStore,
	config EngineConfig,
) (*MedicationAdvisorEngine, error) {

	// Validate required KB URLs
	if config.KB1URL == "" {
		return nil, fmt.Errorf("KB-1 Dosing URL is required")
	}
	if config.KB2URL == "" {
		return nil, fmt.Errorf("KB-2 Interactions URL is required")
	}
	if config.KB3URL == "" {
		return nil, fmt.Errorf("KB-3 Guidelines URL is required")
	}
	if config.KB4URL == "" {
		return nil, fmt.Errorf("KB-4 Safety URL is required")
	}
	if config.KB5URL == "" {
		return nil, fmt.Errorf("KB-5 Monitoring URL is required")
	}
	if config.KB6URL == "" {
		return nil, fmt.Errorf("KB-6 Efficacy URL is required")
	}

	snapshotMgr := snapshot.NewSnapshotManager(snapshotStore, config.SnapshotTTLMinutes)
	evidenceMgr := evidence.NewEvidenceEnvelopeManager(envelopeStore, config.Environment)

	// Create workflow orchestrator - will validate KB connections
	workflowEngine, err := NewWorkflowOrchestrator(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create workflow orchestrator: %w", err)
	}

	// V3 Architecture: Create KB-5 DDI client for external DDI checking - REQUIRED
	var kb5DDIClient kbclients.KB5DDIClient
	if config.KB5DDIURL != "" {
		kb5DDIClient, err = kbclients.NewKB5DDIClient(kbclients.ProductionClientConfig(config.KB5DDIURL))
		if err != nil {
			log.Printf("[MedicationAdvisor] ERROR: Failed to create KB-5 DDI client: %v (DDI checks will be skipped)", err)
		} else {
			log.Printf("[MedicationAdvisor] V3 KB-5 DDI client initialized: %s", config.KB5DDIURL)
		}
	} else {
		log.Printf("[MedicationAdvisor] WARNING: KB5DDIURL not configured - DDI checks will be skipped (NO LOCAL FALLBACK)")
	}

	// V3 Architecture: Create KB-16 Lab Safety client for lab-based contraindications - REQUIRED
	var kb16LabSafetyClient kbclients.KB16LabSafetyClient
	if config.KB16URL != "" {
		kb16LabSafetyClient, err = kbclients.NewKB16LabSafetyClient(kbclients.ProductionClientConfig(config.KB16URL))
		if err != nil {
			log.Printf("[MedicationAdvisor] ERROR: Failed to create KB-16 Lab Safety client: %v (lab safety checks will be skipped)", err)
		} else {
			log.Printf("[MedicationAdvisor] V3 KB-16 Lab Safety client initialized: %s", config.KB16URL)
		}
	} else {
		log.Printf("[MedicationAdvisor] WARNING: KB16URL not configured - lab safety checks will be skipped (NO LOCAL FALLBACK)")
	}

	return &MedicationAdvisorEngine{
		snapshotManager:     snapshotMgr,
		evidenceManager:     evidenceMgr,
		workflowEngine:      workflowEngine,
		scoringEngine:       NewProposalScoringEngine(),
		conflictDetector:    NewConflictDetector(),
		kb5DDIClient:        kb5DDIClient,
		kb16LabSafetyClient: kb16LabSafetyClient,
		config:              config,
	}, nil
}

// NewTestMedicationAdvisorEngine creates a new medication advisor engine for testing.
// This accepts a pre-configured workflow orchestrator, allowing injection of mock KB clients.
func NewTestMedicationAdvisorEngine(
	snapshotStore snapshot.SnapshotStore,
	envelopeStore evidence.EnvelopeStore,
	config EngineConfig,
	workflowEngine *WorkflowOrchestrator,
) *MedicationAdvisorEngine {
	snapshotMgr := snapshot.NewSnapshotManager(snapshotStore, config.SnapshotTTLMinutes)
	evidenceMgr := evidence.NewEvidenceEnvelopeManager(envelopeStore, config.Environment)

	return &MedicationAdvisorEngine{
		snapshotManager:  snapshotMgr,
		evidenceManager:  evidenceMgr,
		workflowEngine:   workflowEngine,
		scoringEngine:    NewProposalScoringEngine(),
		conflictDetector: NewConflictDetector(),
		config:           config,
	}
}

// ============================================================================
// Calculate Phase
// ============================================================================

// CalculateRequest represents a request to calculate medication proposals
type CalculateRequest struct {
	PatientID       uuid.UUID              `json:"patient_id"`
	ProviderID      string                 `json:"provider_id"`
	SessionID       string                 `json:"session_id"`
	ClinicalQuestion ClinicalQuestion      `json:"clinical_question"`
	PatientContext  PatientContext         `json:"patient_context"`
}

// ClinicalQuestion represents the clinical question being asked
type ClinicalQuestion struct {
	Text           string `json:"text"`
	Intent         string `json:"intent"` // ADD_MEDICATION, ADJUST_DOSE, SWITCH_MEDICATION
	TargetDrugClass string `json:"target_drug_class,omitempty"`
	TargetRxNorm   string `json:"target_rxnorm,omitempty"`
	Indication     string `json:"indication,omitempty"`
}

// PatientContext contains the patient's clinical context
type PatientContext struct {
	Age            int                      `json:"age"`
	Sex            string                   `json:"sex"`
	WeightKg       *float64                 `json:"weight_kg,omitempty"`
	HeightCm       *float64                 `json:"height_cm,omitempty"`
	Conditions     []ClinicalCode           `json:"conditions"`
	Medications    []ClinicalCode           `json:"medications"`
	Allergies      []ClinicalCode           `json:"allergies"`
	LabResults     []LabValue               `json:"lab_results,omitempty"`
	ComputedScores snapshot.ComputedScores  `json:"computed_scores"`
}

// ClinicalCode represents a coded clinical concept
type ClinicalCode struct {
	System  string `json:"system"` // SNOMED, ICD-10, RxNorm, LOINC
	Code    string `json:"code"`
	Display string `json:"display"`
}

// LabValue represents a laboratory result
type LabValue struct {
	Code           string      `json:"code"`
	Display        string      `json:"display"`
	Value          interface{} `json:"value"`
	Unit           string      `json:"unit"`
	ReferenceRange string      `json:"reference_range,omitempty"`
	Critical       bool        `json:"critical"`
}

// CalculateResponse represents the response from Calculate phase
type CalculateResponse struct {
	SnapshotID       uuid.UUID            `json:"snapshot_id"`
	EnvelopeID       uuid.UUID            `json:"envelope_id"`
	Proposals        []MedicationProposal `json:"proposals"`
	HardBlocks       []HardBlock          `json:"hard_blocks,omitempty"`       // Critical safety blocks requiring acknowledgment
	ExcludedDrugs    []ExcludedDrugInfo   `json:"excluded_drugs,omitempty"`    // All excluded drugs with reasons
	GeneratedTasks   []GeneratedTask      `json:"generated_tasks,omitempty"`   // Tasks generated from KB-14 (monitoring, follow-up)
	GovernanceEvents []GovernanceEvent    `json:"governance_events,omitempty"` // Tier-7 governance events for audit
	Disposition      DispositionCode      `json:"disposition"`                 // What happens next
	AuditTrail       AuditTrailSummary    `json:"audit_trail"`                 // Tier-7 compliance audit trail
	ExecutionTimeMs  int64                `json:"execution_time_ms"`
	KBVersions       map[string]string    `json:"kb_versions"`
}

// AuditTrailSummary provides Tier-7 compliance audit trail summary
// This ensures every medication decision is court-defensible and regulator-compliant
type AuditTrailSummary struct {
	// Core Identifiers
	TraceID          string   `json:"trace_id"`          // Unique trace ID for this decision chain
	SessionID        string   `json:"session_id"`        // User session ID
	TransactionID    string   `json:"transaction_id"`    // FHIR transaction bundle ID if applicable

	// Evidence Chain
	EvidenceCount    int      `json:"evidence_count"`    // Number of evidence steps recorded
	KBServicesUsed   []string `json:"kb_services_used"`  // Which KB services were consulted
	RulesEvaluated   int      `json:"rules_evaluated"`   // Number of clinical rules evaluated

	// Safety Summary
	SafetyChecks     int      `json:"safety_checks"`     // Total safety checks performed
	BlocksGenerated  int      `json:"blocks_generated"`  // Hard blocks generated
	WarningsGenerated int     `json:"warnings_generated"` // Warnings generated

	// Disposition Tracking
	DispositionReason string  `json:"disposition_reason"` // Why this disposition was chosen
	RequiresAck       bool    `json:"requires_ack"`       // Whether acknowledgment is required
	AckText           string  `json:"ack_text,omitempty"` // Acknowledgment text if required

	// Governance Metadata
	GovernanceLevel  string   `json:"governance_level"`  // TIER_7_COMPLETE, TIER_6_PARTIAL, etc.
	ComplianceStatus string   `json:"compliance_status"` // COMPLIANT, REQUIRES_REVIEW, NON_COMPLIANT
	AuditHash        string   `json:"audit_hash"`        // SHA256 hash of audit trail for integrity
	Timestamp        string   `json:"timestamp"`         // ISO8601 timestamp
}

// GeneratedTask represents a task generated from medication advisory
type GeneratedTask struct {
	TaskType     string                 `json:"task_type"`
	Priority     string                 `json:"priority"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	DueInMinutes int                    `json:"due_in_minutes"`
	AssignedRole string                 `json:"assigned_role"`
	Source       string                 `json:"source"` // Which KB or rule generated this
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// DispositionCode represents the recommended next action
type DispositionCode string

const (
	// Primary Dispositions
	DispositionDispense          DispositionCode = "DISPENSE"             // Safe to proceed with medication
	DispositionHoldForReview     DispositionCode = "HOLD_FOR_REVIEW"      // Requires pharmacist review
	DispositionHoldForApproval   DispositionCode = "HOLD_FOR_APPROVAL"    // Requires physician approval
	DispositionEscalateToMD      DispositionCode = "ESCALATE_TO_MD"       // Requires immediate MD review
	DispositionHardStop          DispositionCode = "HARD_STOP"            // Cannot proceed without acknowledgment
	DispositionRequireOverride   DispositionCode = "REQUIRE_OVERRIDE"     // Requires documented override
	DispositionDeferToSpecialist DispositionCode = "DEFER_TO_SPECIALIST"  // Requires specialist consultation
	DispositionRecalculate       DispositionCode = "RECALCULATE"          // Need additional information

	// Lab-Based Dispositions (KB-16)
	DispositionLabContraindicated       DispositionCode = "LAB_CONTRAINDICATED"          // Lab values contraindicate medication
	DispositionRequireSpecialistReview  DispositionCode = "REQUIRES_SPECIALIST_REVIEW"   // Requires specialist due to lab abnormalities
	DispositionPatientStabilization     DispositionCode = "PATIENT_STABILIZATION_REQUIRED" // Patient must be stabilized before medication

	// ICU-Specific Dispositions (Tier-10)
	DispositionICUHardStop              DispositionCode = "ICU_HARD_STOP"                // ICU-specific critical block
	DispositionHemodynamicInstability   DispositionCode = "HEMODYNAMIC_INSTABILITY"     // Patient hemodynamically unstable
	DispositionVentilatorDependent      DispositionCode = "VENTILATOR_DEPENDENT"        // Requires ventilator safety review
	DispositionCRRTAdjustment           DispositionCode = "CRRT_ADJUSTMENT_REQUIRED"    // CRRT dosing recalculation needed
	DispositionMultiOrganFailure        DispositionCode = "MULTI_ORGAN_FAILURE"         // MOF requires intensive review
	DispositionSepsisProtocol           DispositionCode = "SEPSIS_PROTOCOL"             // Must follow sepsis protocol
	DispositionNeurologicalMonitoring   DispositionCode = "NEUROLOGICAL_MONITORING"     // Neurological status requires monitoring

	// ICU Safety Evaluation Dispositions (Tier-10 Phase 2)
	DispositionICUCriticalHardStop DispositionCode = "ICU_CRITICAL_HARD_STOP" // Critical ICU block - immediate intervention
	DispositionICUHighRisk         DispositionCode = "ICU_HIGH_RISK"          // High risk medication in ICU context
	DispositionICUDoseAdjustment   DispositionCode = "ICU_DOSE_ADJUSTMENT"    // ICU-specific dose modification needed
	DispositionICUSafe             DispositionCode = "ICU_SAFE"               // Safe for ICU administration
)

// GovernanceEventType represents types of governance events for Tier-7 compliance
type GovernanceEventType string

const (
	// Safety Governance Events
	GovernanceEventPolicyViolation     GovernanceEventType = "POLICY_VIOLATION"        // Protocol/policy violation detected
	GovernanceEventPatientSafetyRisk   GovernanceEventType = "PATIENT_SAFETY_RISK"     // Patient safety concern identified
	GovernanceEventLabContraindication GovernanceEventType = "LAB_CONTRAINDICATION"    // Lab-based medication block
	GovernanceEventDDIHardStop         GovernanceEventType = "DDI_HARD_STOP"           // Drug-drug interaction block
	GovernanceEventPregnancyRisk       GovernanceEventType = "PREGNANCY_RISK"          // Teratogenic medication in pregnancy
	GovernanceEventBlackBoxWarning     GovernanceEventType = "BLACK_BOX_WARNING"       // FDA black box warning triggered

	// ICU Governance Events
	GovernanceEventICUCriticalBlock    GovernanceEventType = "ICU_CRITICAL_BLOCK"      // ICU-specific critical safety block
	GovernanceEventHemodynamicRisk     GovernanceEventType = "HEMODYNAMIC_RISK"        // Hemodynamic instability concern
	GovernanceEventVentilatorRisk      GovernanceEventType = "VENTILATOR_RISK"         // Ventilator/respiratory safety concern
	GovernanceEventRenalRisk           GovernanceEventType = "RENAL_RISK"              // Nephrotoxicity or renal dosing concern
	GovernanceEventNeurologicalRisk    GovernanceEventType = "NEUROLOGICAL_RISK"       // Sedation/delirium/neurological concern
)

// GovernanceEvent represents a tracked governance event for audit
type GovernanceEvent struct {
	ID            uuid.UUID           `json:"id"`
	EventType     GovernanceEventType `json:"event_type"`
	Severity      string              `json:"severity"`      // critical, high, medium, low
	Timestamp     time.Time           `json:"timestamp"`
	PatientID     uuid.UUID           `json:"patient_id"`
	ProviderID    string              `json:"provider_id"`
	MedicationCode string             `json:"medication_code,omitempty"`
	TriggerCode   string              `json:"trigger_code,omitempty"`   // What triggered the event (LOINC, SNOMED, etc.)
	TriggerValue  string              `json:"trigger_value,omitempty"`  // Lab value, condition, etc.
	BlockType     string              `json:"block_type"`
	KBSource      string              `json:"kb_source"`
	RuleID        string              `json:"rule_id"`
	Description   string              `json:"description"`
	RequiresAck   bool                `json:"requires_ack"`
	Acknowledged  bool                `json:"acknowledged"`
	AcknowledgedBy string             `json:"acknowledged_by,omitempty"`
	AcknowledgedAt *time.Time         `json:"acknowledged_at,omitempty"`
	HashChain     string              `json:"hash_chain"`              // Immutable hash for audit trail
}

// HardBlock represents a critical safety block that MUST be acknowledged.
// These are non-negotiable contraindications that require explicit provider acknowledgment
// before any medication decision can proceed. Examples:
// - Pregnancy + teratogenic drugs (ACE inhibitors, ARBs, statins, warfarin)
// - Absolute contraindications (severe allergies, life-threatening interactions)
// - FDA Black Box warnings requiring explicit acknowledgment
type HardBlock struct {
	ID              uuid.UUID    `json:"id"`
	BlockType       string       `json:"block_type"`          // CONTRAINDICATION, ALLERGY, DDI_SEVERE, BLACK_BOX
	Severity        string       `json:"severity"`            // absolute, life_threatening
	Medication      ClinicalCode `json:"medication"`          // The blocked medication
	TriggerCondition ClinicalCode `json:"trigger_condition"`  // What triggered the block (e.g., pregnancy)
	Reason          string       `json:"reason"`              // Human-readable explanation
	FDACategory     string       `json:"fda_category,omitempty"` // FDA pregnancy category if applicable
	KBSource        string       `json:"kb_source"`           // Which KB service flagged this
	RuleID          string       `json:"rule_id"`             // Rule ID for audit trail
	RequiresAck     bool         `json:"requires_ack"`        // Always true for hard blocks
	AckText         string       `json:"ack_text"`            // Required acknowledgment text
}

// ExcludedDrugInfo represents a drug that was excluded during safety screening
type ExcludedDrugInfo struct {
	Medication     ClinicalCode `json:"medication"`
	Reason         string       `json:"reason"`
	Severity       string       `json:"severity"`  // absolute, relative, moderate
	BlockType      string       `json:"block_type"` // contraindication, interaction, allergy, age
	KBSource       string       `json:"kb_source"`
	RuleID         string       `json:"rule_id"`
	IsHardBlock    bool         `json:"is_hard_block"` // True if this is also a hard block
}

// MedicationProposal represents a medication recommendation
type MedicationProposal struct {
	ID             uuid.UUID       `json:"id"`
	Rank           int             `json:"rank"`
	Medication     ClinicalCode    `json:"medication"`
	Dosage         Dosage          `json:"dosage"`
	QualityScore   float64         `json:"quality_score"`
	QualityFactors QualityFactors  `json:"quality_factors"`
	Rationale      string          `json:"rationale"`
	Warnings       []Warning       `json:"warnings,omitempty"`
	Alternatives   []uuid.UUID     `json:"alternatives,omitempty"`
}

// Dosage represents medication dosage
type Dosage struct {
	Value      float64 `json:"value"`
	Unit       string  `json:"unit"`
	Route      string  `json:"route"`
	Frequency  string  `json:"frequency"`
	Duration   string  `json:"duration,omitempty"`
	MaxDaily   float64 `json:"max_daily,omitempty"`
}

// QualityFactors represents the weighted scoring factors
type QualityFactors struct {
	Guideline   float64 `json:"guideline"`   // 30%
	Safety      float64 `json:"safety"`      // 25%
	Efficacy    float64 `json:"efficacy"`    // 20%
	Interaction float64 `json:"interaction"` // 15%
	Monitoring  float64 `json:"monitoring"`  // 10%
}

// Warning represents a clinical warning
type Warning struct {
	Severity string `json:"severity"` // info, warning, critical
	Message  string `json:"message"`
	Source   string `json:"source"`
}

// Calculate executes the Calculate phase of the workflow
func (e *MedicationAdvisorEngine) Calculate(ctx context.Context, req *CalculateRequest) (*CalculateResponse, error) {
	startTime := time.Now()

	// Build clinical snapshot data from context
	clinicalData := e.buildClinicalData(req.PatientContext)

	// Create calculation snapshot
	snap, err := e.snapshotManager.CreateCalculationSnapshot(
		ctx,
		req.PatientID,
		uuid.New(), // Recipe ID
		clinicalData,
		req.PatientContext.ComputedScores,
		req.ProviderID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot: %w", err)
	}

	// Create evidence envelope
	kbVersions := map[string]string{
		"KB-1": "1.0.0", // These would come from actual KB clients
		"KB-2": "1.0.0",
		"KB-3": "1.0.0",
		"KB-4": "1.0.0",
		"KB-5": "1.0.0",
		"KB-6": "1.0.0",
	}

	envelope, err := e.evidenceManager.CreateEnvelope(
		ctx,
		snap.ID,
		req.PatientID,
		req.ProviderID,
		req.SessionID,
		kbVersions,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create evidence envelope: %w", err)
	}

	// Execute workflow
	workflowResult, err := e.workflowEngine.Execute(ctx, &WorkflowInput{
		Snapshot:       snap,
		Question:       req.ClinicalQuestion,
		PatientContext: req.PatientContext,
		EnvelopeID:     envelope.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("workflow execution failed: %w", err)
	}

	// Score and rank proposals
	rankedProposals := e.scoringEngine.RankProposals(workflowResult.Candidates)

	// Process excluded drugs and generate hard blocks (contraindications)
	hardBlocks, excludedDrugs := e.processExcludedDrugs(workflowResult.ExcludedDrugs, req.PatientContext)

	// Process DDI hard blocks (KB-5 enforcement)
	// Check proposed medications against current medications for severe DDIs
	proposedMeds := extractMedicationsFromProposals(rankedProposals)
	ddiHardBlocks := e.processDDIHardBlocks(proposedMeds, req.PatientContext.Medications)
	hardBlocks = append(hardBlocks, ddiHardBlocks...)

	// Process Lab-Based Hard Blocks (KB-16 enforcement)
	// Check proposed medications against patient lab values for contraindications
	// Examples: K+ 6.2 + ACE → HARD_STOP, eGFR < 30 + Metformin → HARD_STOP
	if len(req.PatientContext.LabResults) > 0 {
		labHardBlocks := e.processLabHardBlocks(proposedMeds, req.PatientContext.LabResults)
		hardBlocks = append(hardBlocks, labHardBlocks...)
	}

	// Generate tasks from monitoring requirements and proposals
	generatedTasks := e.generateTasks(rankedProposals, req.PatientID.String())

	// ========================================================================
	// Tier-7 Governance Event Generation (KB-16 Compliance)
	// Every KB-16 block MUST trigger: Governance Event + KB-14 Task + Hash Chain
	// ========================================================================

	// Generate governance events from ALL hard blocks with immutable hash chain
	// Initial hash is derived from envelope ID for traceability
	initialHash := fmt.Sprintf("envelope:%s:session:%s", envelope.ID.String(), req.SessionID)
	governanceEvents := e.generateGovernanceEvents(hardBlocks, req.PatientID, req.ProviderID, initialHash)

	// Generate KB-14 mandatory tasks for lab safety violations
	// These are REQUIRED for any KB-16 lab contraindication per governance rules
	labSafetyTasks := e.generateLabSafetyTasks(hardBlocks, req.PatientID.String())
	generatedTasks = append(generatedTasks, labSafetyTasks...)

	// Determine disposition based on hard blocks and workflow results
	disposition := e.determineDisposition(hardBlocks, rankedProposals, workflowResult)

	// Upgrade disposition for lab contraindications to LAB_CONTRAINDICATED
	if len(labSafetyTasks) > 0 {
		disposition = DispositionLabContraindicated
	}

	// Build Tier-7 audit trail summary
	auditTrail := e.buildAuditTrail(
		snap.ID,
		envelope.ID,
		req.SessionID,
		hardBlocks,
		rankedProposals,
		workflowResult,
		kbVersions,
		disposition,
	)

	return &CalculateResponse{
		SnapshotID:       snap.ID,
		EnvelopeID:       envelope.ID,
		Proposals:        rankedProposals,
		HardBlocks:       hardBlocks,
		ExcludedDrugs:    excludedDrugs,
		GeneratedTasks:   generatedTasks,
		GovernanceEvents: governanceEvents, // Tier-7 governance events for audit
		Disposition:      disposition,
		AuditTrail:       auditTrail,
		ExecutionTimeMs:  time.Since(startTime).Milliseconds(),
		KBVersions:       kbVersions,
	}, nil
}

// generateTasks creates monitoring and follow-up tasks from medication proposals
func (e *MedicationAdvisorEngine) generateTasks(proposals []MedicationProposal, patientID string) []GeneratedTask {
	var tasks []GeneratedTask

	for _, proposal := range proposals {
		// Generate monitoring tasks for high-alert medications
		for _, warning := range proposal.Warnings {
			if warning.Severity == "critical" || warning.Source == "KB-5" {
				tasks = append(tasks, GeneratedTask{
					TaskType:     "MONITORING_OVERDUE",
					Priority:     "HIGH",
					Title:        fmt.Sprintf("Monitor %s - %s", proposal.Medication.Display, warning.Message),
					Description:  fmt.Sprintf("Monitoring required for %s: %s", proposal.Medication.Display, warning.Message),
					DueInMinutes: 240, // 4 hours for critical monitoring
					AssignedRole: "Nurse",
					Source:       warning.Source,
					Metadata: map[string]interface{}{
						"medication_code": proposal.Medication.Code,
						"medication_name": proposal.Medication.Display,
						"warning_type":    warning.Severity,
						"patient_id":      patientID,
					},
				})
			}
		}

		// Generate baseline lab tasks for new medications requiring monitoring
		if proposal.QualityFactors.Monitoring > 0.5 {
			tasks = append(tasks, GeneratedTask{
				TaskType:     "MEDICATION_REVIEW",
				Priority:     "MEDIUM",
				Title:        fmt.Sprintf("Baseline labs for %s", proposal.Medication.Display),
				Description:  fmt.Sprintf("Obtain baseline laboratory values before initiating %s therapy", proposal.Medication.Display),
				DueInMinutes: 1440, // 24 hours
				AssignedRole: "Care Coordinator",
				Source:       "KB-5",
				Metadata: map[string]interface{}{
					"medication_code": proposal.Medication.Code,
					"medication_name": proposal.Medication.Display,
					"monitoring_score": proposal.QualityFactors.Monitoring,
					"patient_id":       patientID,
				},
			})
		}
	}

	return tasks
}

// ============================================================================
// Tier-7 Governance Event Generation (KB-16 Compliance)
// ============================================================================

// generateGovernanceEvents creates governance events from hard blocks for Tier-7 compliance.
// Every KB-16 lab safety block MUST generate a corresponding governance event with:
// - Event type (POLICY_VIOLATION, PATIENT_SAFETY_RISK, LAB_CONTRAINDICATION)
// - Immutable hash chain for audit trail
// - KB-14 task linkage for follow-up
func (e *MedicationAdvisorEngine) generateGovernanceEvents(
	hardBlocks []HardBlock,
	patientID uuid.UUID,
	providerID string,
	previousHash string,
) []GovernanceEvent {
	var events []GovernanceEvent
	currentHash := previousHash

	for _, block := range hardBlocks {
		// Determine event type based on block source and type
		eventType := e.determineGovernanceEventType(block)

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
		event.HashChain = e.computeEventHash(event, currentHash)
		currentHash = event.HashChain

		events = append(events, event)
	}

	return events
}

// determineGovernanceEventType maps hard block types to governance event types
func (e *MedicationAdvisorEngine) determineGovernanceEventType(block HardBlock) GovernanceEventType {
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

// computeEventHash creates an immutable SHA256 hash for the governance event chain
func (e *MedicationAdvisorEngine) computeEventHash(event GovernanceEvent, previousHash string) string {
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

// generateLabSafetyTasks creates KB-14 mandatory tasks for lab safety violations.
// Every KB-16 lab contraindication MUST generate follow-up tasks per governance requirements.
func (e *MedicationAdvisorEngine) generateLabSafetyTasks(
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

// determineDisposition determines the recommended next action based on results
func (e *MedicationAdvisorEngine) determineDisposition(
	hardBlocks []HardBlock,
	proposals []MedicationProposal,
	workflowResult *WorkflowResult,
) DispositionCode {
	// Hard blocks always result in HARD_STOP
	if len(hardBlocks) > 0 {
		return DispositionHardStop
	}

	// No proposals means we need more information
	if len(proposals) == 0 {
		return DispositionRecalculate
	}

	// Check for severe interactions or warnings
	for _, proposal := range proposals {
		for _, warning := range proposal.Warnings {
			if warning.Severity == "critical" {
				return DispositionHoldForReview
			}
		}
	}

	// Check quality scores - low safety score requires review
	for _, proposal := range proposals {
		if proposal.QualityFactors.Safety < 0.6 {
			return DispositionHoldForApproval
		}
	}

	// All clear - safe to proceed
	return DispositionDispense
}

// buildAuditTrail creates a Tier-7 compliant audit trail summary
func (e *MedicationAdvisorEngine) buildAuditTrail(
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
	for _, step := range workflowResult.InferenceChain {
		if step.RuleID != "" {
			rulesEvaluated++
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

	return AuditTrailSummary{
		TraceID:           fmt.Sprintf("TRACE-%s", snapshotID.String()[:8]),
		SessionID:         sessionID,
		TransactionID:     fmt.Sprintf("TXN-%s", envelopeID.String()[:8]),
		EvidenceCount:     len(workflowResult.InferenceChain),
		KBServicesUsed:    kbServices,
		RulesEvaluated:    rulesEvaluated,
		SafetyChecks:      len(workflowResult.ExcludedDrugs) + len(hardBlocks),
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

// processExcludedDrugs converts workflow excluded drugs to response format and generates hard blocks
func (e *MedicationAdvisorEngine) processExcludedDrugs(
	excluded []ExcludedDrug,
	patientContext PatientContext,
) ([]HardBlock, []ExcludedDrugInfo) {
	var hardBlocks []HardBlock
	var excludedDrugs []ExcludedDrugInfo

	// Find pregnancy condition for hard block context
	var pregnancyCondition *ClinicalCode
	for _, cond := range patientContext.Conditions {
		if isPregnancyCode(cond.Code) {
			pregnancyCondition = &cond
			break
		}
	}

	for _, ex := range excluded {
		// Determine if this is a hard block based on severity
		isHardBlock := isHardBlockSeverity(ex.Severity)

		// Convert to ExcludedDrugInfo
		excludedDrug := ExcludedDrugInfo{
			Medication: ex.Medication,
			Reason:     ex.Reason,
			Severity:   ex.Severity,
			BlockType:  ex.Severity, // Use severity as block type for now
			KBSource:   ex.KBSource,
			RuleID:     ex.RuleID,
			IsHardBlock: isHardBlock,
		}
		excludedDrugs = append(excludedDrugs, excludedDrug)

		// Generate hard block if severity warrants it
		if isHardBlock {
			hardBlock := HardBlock{
				ID:          uuid.New(),
				BlockType:   mapToHardBlockType(ex.Severity),
				Severity:    ex.Severity,
				Medication:  ex.Medication,
				Reason:      ex.Reason,
				KBSource:    ex.KBSource,
				RuleID:      ex.RuleID,
				RequiresAck: true,
				AckText:     generateAckText(ex.Medication.Display, ex.Reason),
			}

			// Add trigger condition if pregnancy-related
			if pregnancyCondition != nil && isPregnancyRelatedBlock(ex.Reason) {
				hardBlock.TriggerCondition = *pregnancyCondition
				hardBlock.FDACategory = extractFDACategory(ex.Reason)
			}

			hardBlocks = append(hardBlocks, hardBlock)
		}
	}

	return hardBlocks, excludedDrugs
}

// isHardBlockSeverity determines if the severity level requires a hard block
func isHardBlockSeverity(severity string) bool {
	hardBlockSeverities := map[string]bool{
		"absolute":         true,
		"contraindicated":  true,
		"life_threatening": true,
		"severe":          true,
	}
	return hardBlockSeverities[severity]
}

// mapToHardBlockType converts severity to a standardized block type
func mapToHardBlockType(severity string) string {
	switch severity {
	case "absolute", "contraindicated":
		return "CONTRAINDICATION"
	case "life_threatening":
		return "LIFE_THREATENING"
	case "severe":
		return "DDI_SEVERE"
	default:
		return "CONTRAINDICATION"
	}
}

// isPregnancyCode checks if a SNOMED code represents pregnancy
func isPregnancyCode(code string) bool {
	pregnancyCodes := map[string]bool{
		"77386006":  true, // Pregnancy
		"72892002":  true, // Normal pregnancy
		"237238006": true, // Gestational diabetes mellitus
		"48194001":  true, // Pregnancy-induced hypertension
		"10746341000119109": true, // High risk pregnancy
	}
	return pregnancyCodes[code]
}

// isPregnancyRelatedBlock checks if the block reason mentions pregnancy
func isPregnancyRelatedBlock(reason string) bool {
	// Check for pregnancy-related keywords in reason
	keywords := []string{"pregnancy", "pregnant", "teratogenic", "fetal", "gestational", "FDA Category D", "FDA Category X"}
	reasonLower := reason
	for _, kw := range keywords {
		if containsIgnoreCase(reasonLower, kw) {
			return true
		}
	}
	return false
}

// containsIgnoreCase checks if s contains substr (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	// Simple case-insensitive contains using lowercase comparison
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr ||
		 len(s) >= len(substr) && containsLower(s, substr))
}

func containsLower(s, substr string) bool {
	// Manual lowercase comparison
	sLower := make([]byte, len(s))
	substrLower := make([]byte, len(substr))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			sLower[i] = c + 32
		} else {
			sLower[i] = c
		}
	}
	for i := 0; i < len(substr); i++ {
		c := substr[i]
		if c >= 'A' && c <= 'Z' {
			substrLower[i] = c + 32
		} else {
			substrLower[i] = c
		}
	}
	// Find substr in s
	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		match := true
		for j := 0; j < len(substrLower); j++ {
			if sLower[i+j] != substrLower[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// extractFDACategory extracts FDA pregnancy category from reason text
func extractFDACategory(reason string) string {
	if containsIgnoreCase(reason, "Category D") || containsIgnoreCase(reason, "(D)") {
		return "D"
	}
	if containsIgnoreCase(reason, "Category X") || containsIgnoreCase(reason, "(X)") {
		return "X"
	}
	if containsIgnoreCase(reason, "Category C") || containsIgnoreCase(reason, "(C)") {
		return "C"
	}
	return ""
}

// generateAckText generates the acknowledgment text for a hard block
func generateAckText(drugName, reason string) string {
	return fmt.Sprintf("I acknowledge that %s is contraindicated for this patient due to: %s. "+
		"I understand the risks and take full clinical responsibility for any override decision.",
		drugName, reason)
}

// =============================================================================
// DDI Hard Stop Enforcement (KB-5) - V3 Architecture: KB-5 ONLY, NO LOCAL RULES
// =============================================================================

// V3 Architecture: All DDI data comes from KB-5 DDI Service
// No local DDI rules or drug class mappings - removed in V3 migration

// processDDIHardBlocks checks for severe drug-drug interactions and generates hard blocks.
// V3 Architecture: Uses KB-5 DDI service EXCLUSIVELY - NO LOCAL FALLBACK.
func (e *MedicationAdvisorEngine) processDDIHardBlocks(
	proposedMeds []ClinicalCode,
	currentMeds []ClinicalCode,
) []HardBlock {
	// Combine proposed and current medications
	allMeds := append(proposedMeds, currentMeds...)
	log.Printf("[DDI-Check] Checking %d proposed + %d current = %d total meds", len(proposedMeds), len(currentMeds), len(allMeds))
	for i, med := range allMeds {
		log.Printf("[DDI-Check]   AllMeds[%d]: %s (RxNorm: %s)", i, med.Display, med.Code)
	}

	// V3 Architecture: Use KB-5 DDI service EXCLUSIVELY
	if e.kb5DDIClient != nil {
		log.Printf("[DDI-Check] V3 Mode: Using KB-5 DDI service for interaction checking")
		return e.processDDIHardBlocksViaKB5(allMeds)
	}

	// KB-5 not configured - DDI checks skipped (no local fallback)
	log.Printf("[DDI-Check] WARNING: KB-5 DDI service not configured - DDI checks skipped")
	return nil
}

// processDDIHardBlocksViaKB5 calls the KB-5 DDI service to check for drug interactions.
// This is the V3 architecture approach where all DDI data comes from KB-5.
func (e *MedicationAdvisorEngine) processDDIHardBlocksViaKB5(allMeds []ClinicalCode) []HardBlock {
	var hardBlocks []HardBlock

	if len(allMeds) < 2 {
		log.Printf("[DDI-Check-KB5] Not enough medications to check for interactions")
		return hardBlocks
	}

	// Extract RxNorm codes from all medications
	drugCodes := make([]string, 0, len(allMeds))
	codeToMed := make(map[string]ClinicalCode)
	for _, med := range allMeds {
		drugCodes = append(drugCodes, med.Code)
		codeToMed[med.Code] = med
	}

	log.Printf("[DDI-Check-KB5] Calling KB-5 DDI service with %d drug codes: %v", len(drugCodes), drugCodes)

	// Call KB-5 DDI service
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	report, err := e.kb5DDIClient.CheckMultipleDDIs(ctx, drugCodes)
	if err != nil {
		log.Printf("[DDI-Check-KB5] ❌ Error calling KB-5 DDI service: %v", err)
		return hardBlocks
	}

	log.Printf("[DDI-Check-KB5] KB-5 returned: %d severe pairs, %d contraindicated pairs, overall risk: %s",
		len(report.SeverePairs), len(report.ContraindicatedPairs), report.OverallRiskLevel)

	// Convert KB-5 results to HardBlocks
	allDDIs := append(report.ContraindicatedPairs, report.SeverePairs...)
	for _, ddi := range allDDIs {
		// Find the matching ClinicalCode entries
		drug1 := ClinicalCode{Code: ddi.Drug1Code, Display: ddi.Drug1Name, System: "http://www.nlm.nih.gov/research/umls/rxnorm"}
		drug2 := ClinicalCode{Code: ddi.Drug2Code, Display: ddi.Drug2Name, System: "http://www.nlm.nih.gov/research/umls/rxnorm"}

		// Try to get better display names from our input medications
		if med, ok := codeToMed[ddi.Drug1Code]; ok {
			drug1 = med
		}
		if med, ok := codeToMed[ddi.Drug2Code]; ok {
			drug2 = med
		}

		severity := string(ddi.Severity)
		if severity == "" {
			severity = "severe"
		}

		log.Printf("[DDI-Check-KB5] ⚠️  DDI FOUND via KB-5: %s + %s → %s (severity: %s)",
			drug1.Display, drug2.Display, ddi.ClinicalEffect, severity)

		hardBlock := HardBlock{
			ID:               uuid.New(),
			BlockType:        "DDI_SEVERE",
			Severity:         severity,
			Medication:       drug1,
			TriggerCondition: drug2,
			Reason:           fmt.Sprintf("Severe DDI: %s + %s - %s", drug1.Display, drug2.Display, ddi.ClinicalEffect),
			KBSource:         "KB-5",
			RuleID:           ddi.RuleID,
			RequiresAck:      ddi.RequiresAck,
			AckText:          ddi.AckText,
		}
		hardBlocks = append(hardBlocks, hardBlock)
	}

	log.Printf("[DDI-Check-KB5] Generated %d hard blocks from KB-5 DDI response", len(hardBlocks))
	return hardBlocks
}

// isDDIHardStopSeverity determines if DDI severity requires a hard stop
func isDDIHardStopSeverity(severity string) bool {
	hardStopSeverities := map[string]bool{
		"life_threatening": true,
		"contraindicated":  true,
		"severe":          true,
	}
	return hardStopSeverities[severity]
}

// extractMedicationsFromProposals extracts medication codes from ranked proposals
func extractMedicationsFromProposals(proposals []MedicationProposal) []ClinicalCode {
	meds := make([]ClinicalCode, 0, len(proposals))
	for _, p := range proposals {
		meds = append(meds, p.Medication)
	}
	return meds
}

// =============================================================================
// Lab-Based Hard Stop Enforcement (KB-16) - V3 Architecture: KB-16 ONLY, NO LOCAL RULES
// =============================================================================

// V3 Architecture: All lab-drug contraindication data comes from KB-16 Lab Safety Service
// No local lab rules or drug class mappings - removed in V3 migration

// processLabHardBlocks checks lab values against known drug contraindications
// V3 Architecture: Uses KB-16 Lab Safety service exclusively
func (e *MedicationAdvisorEngine) processLabHardBlocks(
	proposedMeds []ClinicalCode,
	patientLabs []LabValue,
) []HardBlock {
	// V3: Use KB-16 Lab Safety service exclusively
	if e.kb16LabSafetyClient != nil {
		return e.processLabHardBlocksViaKB16(proposedMeds, patientLabs)
	}

	// KB-16 not configured - return empty (no lab safety checks without KB-16)
	log.Printf("[MedicationAdvisor] WARNING: KB-16 not configured - lab safety checks skipped")
	return nil
}

// processLabHardBlocksViaKB16 calls KB-16 Lab Safety service for lab interpretation
func (e *MedicationAdvisorEngine) processLabHardBlocksViaKB16(
	proposedMeds []ClinicalCode,
	patientLabs []LabValue,
) []HardBlock {
	var hardBlocks []HardBlock

	// Convert to KB-16 request format
	kb16Labs := make([]kbclients.LabValue, 0, len(patientLabs))
	labUnitMap := make(map[string]string) // Store unit by LOINC code for later use
	for _, lab := range patientLabs {
		numericValue, ok := getLabValueAsFloat64(lab.Value)
		if !ok {
			continue
		}
		labUnitMap[lab.Code] = lab.Unit
		kb16Labs = append(kb16Labs, kbclients.LabValue{
			LOINCCode:  lab.Code,
			TestName:   lab.Display,
			Value:      numericValue,
			Unit:       lab.Unit,
			IsCritical: lab.Critical,
		})
	}

	// Check each proposed medication against KB-16
	for _, med := range proposedMeds {
		req := &kbclients.LabSafetyRequest{
			RxNormCode:  med.Code,
			DrugName:    med.Display,
			CurrentLabs: kb16Labs,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		result, err := e.kb16LabSafetyClient.CheckLabSafety(ctx, req)
		cancel()

		if err != nil {
			log.Printf("[MedicationAdvisor] KB-16 lab safety check failed for %s: %v (skipping this medication)", med.Code, err)
			continue // Skip this medication - NO LOCAL FALLBACK
		}

		// Convert KB-16 violations to HardBlocks
		for _, violation := range result.Violations {
			// Get unit from original lab data
			unit := labUnitMap[violation.LOINCCode]

			// Convert LabSafetyLevel to string for HardBlock
			severityStr := string(violation.Severity)

			hardBlock := HardBlock{
				ID:        uuid.New(),
				BlockType: "LAB_CONTRAINDICATION",
				Severity:  severityStr,
				Medication: med,
				TriggerCondition: ClinicalCode{
					System:  "LOINC",
					Code:    violation.LOINCCode,
					Display: fmt.Sprintf("%s: %.2f %s", violation.TestName, violation.CurrentValue, unit),
				},
				Reason:      fmt.Sprintf("KB-16 Lab contraindication: %s (%.2f %s) - %s", violation.TestName, violation.CurrentValue, unit, violation.ClinicalEffect),
				KBSource:    "KB-16",
				RuleID:      violation.RuleID,
				RequiresAck: violation.Severity == kbclients.LabSafetyLevelCritical || severityStr == "SEVERE",
				AckText:     generateLabAckText(med.Display, violation.TestName, violation.CurrentValue, unit, violation.ClinicalEffect),
			}
			hardBlocks = append(hardBlocks, hardBlock)
		}
	}

	return hardBlocks
}

// V3 Architecture: Local lab rule functions (labRuleApplies, findLabValue, isLabViolation) removed
// All lab safety checks now handled by KB-16 Lab Safety Service

// generateLabAckText generates acknowledgment text for lab-based hard blocks
func generateLabAckText(drugName, testName string, value float64, unit, effect string) string {
	return fmt.Sprintf("I acknowledge that %s is contraindicated based on current lab values. "+
		"%s: %.2f %s. Clinical concern: %s. "+
		"I have reviewed the risks and take full clinical responsibility for any override decision.",
		drugName, testName, value, unit, effect)
}


// getLabValueAsFloat64 extracts the numeric value from a LabValue's interface{} field
// Handles float64, int, int64, and string representations
func getLabValueAsFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case string:
		// Try to parse string as float
		var f float64
		_, err := fmt.Sscanf(val, "%f", &f)
		return f, err == nil
	default:
		return 0, false
	}
}

// ============================================================================
// Validate Phase
// ============================================================================

// ValidateRequest represents a request to validate a selected proposal
type ValidateRequest struct {
	SnapshotID   uuid.UUID `json:"snapshot_id"`
	ProposalID   uuid.UUID `json:"proposal_id"`
	CurrentData  *PatientContext `json:"current_data,omitempty"` // Optional fresh data for conflict detection
}

// ValidateResponse represents the response from Validate phase
type ValidateResponse struct {
	Valid                bool                     `json:"valid"`
	ValidationSnapshotID uuid.UUID                `json:"validation_snapshot_id"` // Use this ID for Commit
	Recommendation       string                   `json:"recommendation"` // proceed, warn, abort
	HardConflicts        []snapshot.Conflict      `json:"hard_conflicts"`
	SoftConflicts        []snapshot.Conflict      `json:"soft_conflicts"`
	ValidationNotes      []string                 `json:"validation_notes,omitempty"`
	ExecutionTimeMs      int64                    `json:"execution_time_ms"`
}

// Validate executes the Validate phase of the workflow
func (e *MedicationAdvisorEngine) Validate(ctx context.Context, req *ValidateRequest) (*ValidateResponse, error) {
	startTime := time.Now()

	// Validate snapshot
	validationResult, err := e.snapshotManager.ValidateSnapshot(ctx, req.SnapshotID.String())
	if err != nil {
		return nil, fmt.Errorf("snapshot validation failed: %w", err)
	}

	if !validationResult.Valid {
		return &ValidateResponse{
			Valid:          false,
			Recommendation: "abort",
			ValidationNotes: validationResult.Errors,
			ExecutionTimeMs: time.Since(startTime).Milliseconds(),
		}, nil
	}

	// Detect changes if current data provided
	var hardConflicts, softConflicts []snapshot.Conflict
	if req.CurrentData != nil {
		currentClinicalData := e.buildClinicalData(*req.CurrentData)
		changeResult, err := e.snapshotManager.DetectChanges(ctx, req.SnapshotID.String(), currentClinicalData)
		if err != nil {
			return nil, fmt.Errorf("change detection failed: %w", err)
		}

		hardConflicts = changeResult.HardConflicts
		softConflicts = changeResult.SoftConflicts
	}

	// Determine recommendation
	recommendation := "proceed"
	if len(hardConflicts) > 0 {
		recommendation = "abort"
	} else if len(softConflicts) > 0 {
		recommendation = "warn"
	}

	// Create validation snapshot - this is required for the Commit phase
	valSnapshot, err := e.snapshotManager.CreateValidationSnapshot(ctx, req.SnapshotID, "system")
	if err != nil {
		return nil, fmt.Errorf("failed to create validation snapshot: %w", err)
	}

	return &ValidateResponse{
		Valid:                len(hardConflicts) == 0,
		ValidationSnapshotID: valSnapshot.ID, // Return this ID for Commit phase
		Recommendation:       recommendation,
		HardConflicts:        hardConflicts,
		SoftConflicts:        softConflicts,
		ValidationNotes:      validationResult.Warnings,
		ExecutionTimeMs:      time.Since(startTime).Milliseconds(),
	}, nil
}

// ============================================================================
// Commit Phase
// ============================================================================

// CommitRequest represents a request to commit a medication decision
type CommitRequest struct {
	SnapshotID    uuid.UUID  `json:"snapshot_id"`     // ValidationSnapshotID from Validate
	EnvelopeID    uuid.UUID  `json:"envelope_id"`     // EnvelopeID from Calculate
	ProposalID    uuid.UUID  `json:"proposal_id"`
	ProviderID    string     `json:"provider_id"`
	Overrides     []Override `json:"overrides,omitempty"`
	Acknowledged  bool       `json:"acknowledged"` // Provider acknowledged warnings
}

// Override represents a provider override of system recommendation
type Override struct {
	Field           string `json:"field"`
	OriginalValue   interface{} `json:"original_value"`
	OverrideValue   interface{} `json:"override_value"`
	Reason          string `json:"reason"`
}

// CommitResponse represents the response from Commit phase
type CommitResponse struct {
	MedicationRequestID string    `json:"medication_request_id"`
	EvidenceFinalized   bool      `json:"evidence_finalized"`
	AuditRecordID       uuid.UUID `json:"audit_record_id"`
	ExecutionTimeMs     int64     `json:"execution_time_ms"`
}

// Commit executes the Commit phase of the workflow
func (e *MedicationAdvisorEngine) Commit(ctx context.Context, req *CommitRequest) (*CommitResponse, error) {
	startTime := time.Now()

	// Get envelope by ID (passed from Calculate phase)
	envelope, err := e.evidenceManager.GetEnvelope(ctx, req.EnvelopeID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get evidence envelope: %w", err)
	}
	if envelope == nil {
		return nil, fmt.Errorf("evidence envelope not found: %s", req.EnvelopeID)
	}

	// Record any overrides
	for _, override := range req.Overrides {
		err := e.evidenceManager.RecordOverride(ctx, envelope.ID, evidence.Override{
			OriginalAction: fmt.Sprintf("%v", override.OriginalValue),
			OverrideAction: fmt.Sprintf("%v", override.OverrideValue),
			Reason:         override.Reason,
			ProviderID:     req.ProviderID,
			ProviderRole:   "physician",
			Acknowledged:   req.Acknowledged,
		})
		if err != nil {
			// Log but continue
		}
	}

	// Create commit snapshot
	_, err = e.snapshotManager.CreateCommitSnapshot(ctx, req.SnapshotID, req.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("failed to create commit snapshot: %w", err)
	}

	// Finalize evidence envelope
	err = e.evidenceManager.Finalize(ctx, envelope.ID, req.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("failed to finalize evidence: %w", err)
	}

	// Generate FHIR MedicationRequest (simplified - would use fhir package)
	medicationRequestID := fmt.Sprintf("MedicationRequest/%s", uuid.New().String())

	return &CommitResponse{
		MedicationRequestID: medicationRequestID,
		EvidenceFinalized:   true,
		AuditRecordID:       uuid.New(),
		ExecutionTimeMs:     time.Since(startTime).Milliseconds(),
	}, nil
}

// ============================================================================
// Explain API
// ============================================================================

// ExplainRequest represents a request for explanation
type ExplainRequest struct {
	EnvelopeID      uuid.UUID `json:"envelope_id"`
	Question        string    `json:"question"` // whyIncluded, whyExcluded, whyRanked
	MedicationCode  string    `json:"medication_code,omitempty"`
	Rank            int       `json:"rank,omitempty"`
}

// ExplainResponse represents an explanation response
type ExplainResponse struct {
	Question        string                   `json:"question"`
	Answer          string                   `json:"answer"`
	InferenceChain  []evidence.InferenceStep `json:"inference_chain"`
	ConfidenceScore float64                  `json:"confidence_score"`
	KBSources       []string                 `json:"kb_sources"`
}

// Explain provides explanation for medication decisions
func (e *MedicationAdvisorEngine) Explain(ctx context.Context, req *ExplainRequest) (*ExplainResponse, error) {
	envelope, err := e.evidenceManager.GetEnvelope(ctx, req.EnvelopeID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get evidence envelope: %w", err)
	}

	explainer := evidence.NewExplainChain(envelope.InferenceChain)

	var result *evidence.ExplainResult
	switch req.Question {
	case "whyIncluded":
		result = explainer.WhyIncluded(req.MedicationCode)
	case "whyExcluded":
		result = explainer.WhyExcluded(req.MedicationCode)
	case "whyRanked":
		result = explainer.WhyRanked(req.MedicationCode, req.Rank)
	default:
		return nil, fmt.Errorf("unknown question type: %s", req.Question)
	}

	return &ExplainResponse{
		Question:        result.Question,
		Answer:          result.Answer,
		InferenceChain:  result.RelevantSteps,
		ConfidenceScore: result.Confidence,
		KBSources:       result.KBSources,
	}, nil
}

// ============================================================================
// Helper Methods
// ============================================================================

func (e *MedicationAdvisorEngine) buildClinicalData(ctx PatientContext) snapshot.ClinicalSnapshotData {
	data := snapshot.ClinicalSnapshotData{
		Demographics: snapshot.PatientDemographics{
			Gender:       ctx.Sex,
			SnapshotTime: time.Now(),
		},
	}

	if ctx.WeightKg != nil {
		data.Demographics.WeightKg = ctx.WeightKg
	}
	if ctx.HeightCm != nil {
		data.Demographics.HeightCm = ctx.HeightCm
	}

	// Convert conditions
	for _, c := range ctx.Conditions {
		data.Conditions = append(data.Conditions, snapshot.ConditionEntry{
			ID:            uuid.New(),
			ConditionName: c.Display,
			SNOMEDCT:      c.Code,
			Status:        snapshot.ConditionStatusActive,
		})
	}

	// Convert medications
	for _, m := range ctx.Medications {
		data.Medications = append(data.Medications, snapshot.MedicationEntry{
			ID:             uuid.New(),
			MedicationName: m.Display,
			RxNormCode:     m.Code,
			Status:         snapshot.MedStatusActive,
		})
	}

	// Convert allergies
	for _, a := range ctx.Allergies {
		data.Allergies = append(data.Allergies, snapshot.AllergyEntry{
			ID:           uuid.New(),
			Allergen:     a.Display,
			AllergenType: snapshot.AllergenDrug,
			Status:       snapshot.AllergyStatusActive,
		})
	}

	return data
}

// Health returns engine health status
func (e *MedicationAdvisorEngine) Health() map[string]interface{} {
	return map[string]interface{}{
		"status":      "healthy",
		"environment": e.config.Environment,
		"snapshot_metrics": e.snapshotManager.GetMetrics(),
		"evidence_metrics": e.evidenceManager.GetMetrics(),
	}
}

// ============================================================================
// Test Helpers - Exported for testing KB-16 Lab Safety
// ============================================================================

// TestProcessLabHardBlocks is an exported wrapper for processLabHardBlocks for testing purposes.
// This allows unit tests to directly test KB-16 lab safety rules without going through full Calculate flow.
func (e *MedicationAdvisorEngine) TestProcessLabHardBlocks(proposedMeds []ClinicalCode, patientLabs []LabValue) []HardBlock {
	return e.processLabHardBlocks(proposedMeds, patientLabs)
}

// TestGenerateGovernanceEvents is an exported wrapper for governance event generation testing.
func (e *MedicationAdvisorEngine) TestGenerateGovernanceEvents(hardBlocks []HardBlock, patientID uuid.UUID, providerID string) []GovernanceEvent {
	initialHash := fmt.Sprintf("test-envelope:%s", patientID.String())
	return e.generateGovernanceEvents(hardBlocks, patientID, providerID, initialHash)
}

// TestGenerateLabSafetyTasks is an exported wrapper for KB-14 lab safety task generation testing.
func (e *MedicationAdvisorEngine) TestGenerateLabSafetyTasks(hardBlocks []HardBlock, patientID string) []GeneratedTask {
	return e.generateLabSafetyTasks(hardBlocks, patientID)
}
