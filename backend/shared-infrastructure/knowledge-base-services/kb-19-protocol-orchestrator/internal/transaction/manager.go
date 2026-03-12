// Package transaction provides the Transaction Manager for KB-19.
// This is the coordinator that ties together the moved functions from Med-Advisor.
// NEW CODE: Transaction state machine (this is the only significant new code in V3)
package transaction

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// TRANSACTION MANAGER
// Coordinates Validator and Committer for the V3 Transaction Authority pattern
// =============================================================================

// Manager coordinates transaction validation and commit operations.
// This is the "Clerk" in the Judge-Jury-Clerk pattern:
// - Med-Advisor (Judge) = Calculates risks
// - Provider (Jury) = Reviews and decides
// - KB-19 Manager (Clerk) = Records decision, enforces blocks, creates audit trail
type Manager struct {
	validator  *Validator
	committer  *Committer
	store      TransactionStore
	config     ManagerConfig
	mu         sync.RWMutex
}

// ManagerConfig holds configuration for the transaction manager
type ManagerConfig struct {
	DefaultTTL         time.Duration // Default TTL for transactions
	EnableAutoExpiry   bool          // Auto-expire stale transactions
	MaxConcurrentTxns  int           // Max concurrent transactions per patient
	RequireIdentity    bool          // Require KB-18 identity binding for overrides
}

// TransactionStore interface for persisting transactions
type TransactionStore interface {
	Save(ctx context.Context, txn *Transaction) error
	Get(ctx context.Context, id uuid.UUID) (*Transaction, error)
	GetByPatient(ctx context.Context, patientID uuid.UUID) ([]*Transaction, error)
	Update(ctx context.Context, txn *Transaction) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// NewManager creates a new transaction manager with default configuration
func NewManager(store TransactionStore) *Manager {
	return &Manager{
		validator: NewValidator(),
		committer: NewCommitter(),
		store:     store,
		config: ManagerConfig{
			DefaultTTL:         30 * time.Minute,
			EnableAutoExpiry:   true,
			MaxConcurrentTxns:  10,
			RequireIdentity:    true,
		},
	}
}

// NewManagerWithConfig creates a manager with custom configuration
func NewManagerWithConfig(store TransactionStore, cfg ManagerConfig, validatorCfg ValidatorConfig, committerCfg CommitterConfig) *Manager {
	return &Manager{
		validator: NewValidatorWithConfig(validatorCfg),
		committer: NewCommitterWithConfig(committerCfg),
		store:     store,
		config:    cfg,
	}
}

// SetKB5Client sets the KB-5 DDI client on the internal validator.
// This enables the validator to call KB-5 for DDI checks instead of using local rules.
func (m *Manager) SetKB5Client(client KB5DDIChecker) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.validator.SetKB5Client(client)
}

// SetRiskProvider sets the Med-Advisor client on the internal validator.
// This enables V3 workflow: Med-Advisor calculates risks, KB-19 makes decisions.
func (m *Manager) SetRiskProvider(provider RiskProfileProvider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.validator.SetRiskProvider(provider)
}

// =============================================================================
// TRANSACTION LIFECYCLE: Create → Validate → (Override) → Commit
// =============================================================================

// CreateTransaction creates a new medication transaction.
// This is Step 1 of the Calculate → Validate → Commit workflow.
func (m *Manager) CreateTransaction(
	ctx context.Context,
	patientID uuid.UUID,
	encounterID uuid.UUID,
	proposedMed ClinicalCode,
	currentMeds []ClinicalCode,
	providerID string,
) (*Transaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create new transaction
	now := time.Now()
	txn := &Transaction{
		ID:                 uuid.New(),
		PatientID:          patientID,
		EncounterID:        encounterID,
		ProposedMedication: proposedMed,
		CurrentMedications: currentMeds, // V3: Store current meds for DDI checking
		ProviderID:         providerID,
		State:              StateCreated,
		CreatedAt:          now,
		UpdatedAt:          now,
		ExpiresAt:          now.Add(m.config.DefaultTTL),
		HardBlocks:         []HardBlock{},
		GovernanceEvents:   []GovernanceEvent{},
		GeneratedTasks:     []GeneratedTask{},
		OverrideDecisions:  []OverrideDecision{},
	}

	// Persist transaction
	if m.store != nil {
		if err := m.store.Save(ctx, txn); err != nil {
			return nil, fmt.Errorf("failed to save transaction: %w", err)
		}
	}

	return txn, nil
}

// ValidateTransaction performs safety validation on a transaction.
// This evaluates DDI rules, Lab rules, and excluded drugs.
// If V3 mode is enabled (Med-Advisor configured), it will call Med-Advisor
// for risk profiles and convert those to hard blocks.
func (m *Manager) ValidateTransaction(
	ctx context.Context,
	txnID uuid.UUID,
	currentMeds []ClinicalCode,
	patientLabs []LabValue,
	excludedDrugs []ExcludedDrug,
	patientContext PatientContext,
) (*Transaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get transaction
	var txn *Transaction
	var err error
	if m.store != nil {
		txn, err = m.store.Get(ctx, txnID)
		if err != nil {
			return nil, fmt.Errorf("failed to get transaction: %w", err)
		}
	}
	if txn == nil {
		return nil, fmt.Errorf("transaction not found: %s", txnID)
	}

	// Verify state
	if txn.State != StateCreated && txn.State != StateBlocked {
		return nil, fmt.Errorf("transaction in invalid state for validation: %s", txn.State)
	}

	// Check expiry
	if time.Now().After(txn.ExpiresAt) {
		txn.State = StateFailed
		return nil, fmt.Errorf("transaction expired")
	}

	// V3: Use stored current medications if not provided in request
	// This ensures DDI checking works even when validate request doesn't include medications
	if len(currentMeds) == 0 && len(txn.CurrentMedications) > 0 {
		currentMeds = txn.CurrentMedications
	}

	// Update state
	txn.State = StateValidating

	// V3 REQUIRED: Med-Advisor must be configured for risk calculation
	// No fallback to legacy validation - fail fast if service unavailable
	if !m.validator.IsV3Enabled() {
		txn.State = StateFailed
		return nil, fmt.Errorf("V3 validation required: Med-Advisor risk provider not configured. Set MEDICATION_ADVISOR_URL environment variable")
	}

	riskProvider := m.validator.GetRiskProvider()
	if riskProvider == nil {
		txn.State = StateFailed
		return nil, fmt.Errorf("V3 validation required: Med-Advisor risk provider is nil")
	}

	// Build allergy list from patient context with structured format
	// V3 FIX: Med-Advisor expects AllergyRefInput with allergen_code, allergen_type, severity
	allergyInputs := make([]AllergyRefInput, 0, len(patientContext.Allergies))
	for _, allergy := range patientContext.Allergies {
		allergyInputs = append(allergyInputs, AllergyRefInput{
			AllergenCode: allergy.Code,
			AllergenType: inferAllergenType(allergy.System), // Infer from code system
			Severity:     "severe",                          // Conservative default for safety
		})
	}

	// Build condition list from patient context with structured format
	// V3 FIX: Med-Advisor expects ConditionRefInput with icd10_code, snomed_code, display
	conditionInputs := make([]ConditionRefInput, 0, len(patientContext.Conditions))
	for _, condition := range patientContext.Conditions {
		conditionInputs = append(conditionInputs, ConditionRefInput{
			ICD10Code:  getCodeBySystem(condition, "ICD10"),
			SNOMEDCode: getCodeBySystem(condition, "SNOMED"),
			Display:    condition.Display,
		})
	}

	// Build risk profile request with complete patient data for safety checks
	riskReq := &RiskProfileRequest{
		PatientID:   txn.PatientID,
		EncounterID: txn.EncounterID,
		Medications: []MedicationInput{
			{
				RxNormCode: txn.ProposedMedication.Code,
				DrugName:   txn.ProposedMedication.Display,
				IsProposed: true,
			},
		},
		PatientData: PatientDataInput{
			Age:        patientContext.Age,
			Gender:     patientContext.Sex,
			WeightKg:   patientContext.WeightKg,
			EGFR:       patientContext.EGFR,
			IsPregnant: patientContext.IsPregnant, // V3 FIX: Pass pregnancy status for teratogenic checks
			Allergies:  allergyInputs,             // V3 FIX: Pass allergies as structured objects
			Conditions: conditionInputs,           // V3 FIX: Pass conditions as structured objects
		},
	}

	// Add current medications to the request
	for _, med := range currentMeds {
		riskReq.Medications = append(riskReq.Medications, MedicationInput{
			RxNormCode: med.Code,
			DrugName:   med.Display,
			IsProposed: false,
		})
	}

	// Add lab values to the request
	for _, lab := range patientLabs {
		if val, ok := lab.Value.(float64); ok {
			riskReq.LabValues = append(riskReq.LabValues, LabValueInput{
				LOINCCode: lab.Code,
				Value:     val,
				Unit:      lab.Unit,
			})
		}
	}

	// Call Med-Advisor for risk profile (V3: Med-Advisor = Judge)
	// NO FALLBACK: If Med-Advisor fails, validation fails
	riskProfile, err := riskProvider.GetRiskProfile(ctx, riskReq)
	if err != nil {
		txn.State = StateFailed
		return nil, fmt.Errorf("Med-Advisor service unavailable: %w. V3 requires Med-Advisor for risk calculation", err)
	}

	// V3: KB-19 = Clerk - convert risks to governance decisions
	if err := m.validator.ValidateTransactionV3(ctx, txn, riskProfile); err != nil {
		txn.State = StateFailed
		return nil, fmt.Errorf("V3 validation failed: %w", err)
	}

	// Determine disposition
	txn.Disposition = m.validator.DetermineDisposition(txn.HardBlocks, []MedicationProposal{}, &WorkflowResult{})
	txn.UpdatedAt = time.Now()

	// Persist updated transaction
	if m.store != nil {
		if err := m.store.Update(ctx, txn); err != nil {
			return nil, fmt.Errorf("failed to update transaction: %w", err)
		}
	}

	return txn, nil
}

// OverrideBlock records an override decision for a hard block.
// This requires identity binding via KB-18 for Tier-7 compliance.
func (m *Manager) OverrideBlock(
	ctx context.Context,
	txnID uuid.UUID,
	blockID uuid.UUID,
	providerID string,
	ackText string,
	clinicalReason string,
) (*Transaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get transaction
	var txn *Transaction
	var err error
	if m.store != nil {
		txn, err = m.store.Get(ctx, txnID)
		if err != nil {
			return nil, fmt.Errorf("failed to get transaction: %w", err)
		}
	}
	if txn == nil {
		return nil, fmt.Errorf("transaction not found: %s", txnID)
	}

	// Verify state allows override
	if txn.State != StateBlocked {
		return nil, fmt.Errorf("transaction not in blocked state: %s", txn.State)
	}

	// Find the block being overridden
	var targetBlock *HardBlock
	for i := range txn.HardBlocks {
		if txn.HardBlocks[i].ID == blockID {
			targetBlock = &txn.HardBlocks[i]
			break
		}
	}
	if targetBlock == nil {
		return nil, fmt.Errorf("block not found: %s", blockID)
	}

	// Verify acknowledgment text matches
	if ackText != targetBlock.AckText {
		return nil, fmt.Errorf("acknowledgment text does not match required text")
	}

	// Record override decision
	override := OverrideDecision{
		BlockID:        blockID,
		OverrideType:   "ATTENDING_APPROVAL",
		Reason:         clinicalReason,
		ApproverID:     providerID,
		AcknowledgedAt: time.Now(),
		KB18RecordID:   "", // Would be populated by KB-18 identity binding in production
	}
	txn.OverrideDecisions = append(txn.OverrideDecisions, override)

	// Update state
	txn.State = StateOverriding
	txn.UpdatedAt = time.Now()

	// Check if all blocks are now overridden
	allOverridden := true
	for _, block := range txn.HardBlocks {
		blockOverridden := false
		for _, o := range txn.OverrideDecisions {
			if o.BlockID == block.ID {
				blockOverridden = true
				break
			}
		}
		if !blockOverridden {
			allOverridden = false
			break
		}
	}

	if allOverridden {
		txn.State = StateValidated
		txn.Disposition = DispositionDispense // Override allows proceed
	} else {
		txn.State = StateBlocked // Still blocked
	}

	// Persist updated transaction
	if m.store != nil {
		if err := m.store.Update(ctx, txn); err != nil {
			return nil, fmt.Errorf("failed to update transaction: %w", err)
		}
	}

	return txn, nil
}

// CommitTransaction finalizes a validated transaction.
// This generates governance events, tasks, and audit trail.
func (m *Manager) CommitTransaction(
	ctx context.Context,
	txnID uuid.UUID,
	providerID string,
	kbVersions map[string]string,
) (*Transaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get transaction
	var txn *Transaction
	var err error
	if m.store != nil {
		txn, err = m.store.Get(ctx, txnID)
		if err != nil {
			return nil, fmt.Errorf("failed to get transaction: %w", err)
		}
	}
	if txn == nil {
		return nil, fmt.Errorf("transaction not found: %s", txnID)
	}

	// Verify state allows commit
	if txn.State != StateValidated && txn.State != StateOverriding {
		// Allow commit for blocked state if all blocks overridden
		if txn.State == StateBlocked && len(txn.OverrideDecisions) >= len(txn.HardBlocks) {
			// All blocks overridden, allow proceed
		} else {
			return nil, fmt.Errorf("transaction not in valid state for commit: %s", txn.State)
		}
	}

	// Call committer (which uses moved governance/audit functions)
	if err := m.committer.CommitTransaction(ctx, txn, providerID, kbVersions); err != nil {
		txn.State = StateFailed
		return nil, fmt.Errorf("commit failed: %w", err)
	}
	txn.UpdatedAt = time.Now()

	// Persist updated transaction
	if m.store != nil {
		if err := m.store.Update(ctx, txn); err != nil {
			return nil, fmt.Errorf("failed to update transaction: %w", err)
		}
	}

	return txn, nil
}

// =============================================================================
// QUERY METHODS
// =============================================================================

// GetTransaction retrieves a transaction by ID
func (m *Manager) GetTransaction(ctx context.Context, txnID uuid.UUID) (*Transaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.store == nil {
		return nil, fmt.Errorf("no transaction store configured")
	}
	return m.store.Get(ctx, txnID)
}

// GetPatientTransactions retrieves all transactions for a patient
func (m *Manager) GetPatientTransactions(ctx context.Context, patientID uuid.UUID) ([]*Transaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.store == nil {
		return nil, fmt.Errorf("no transaction store configured")
	}
	return m.store.GetByPatient(ctx, patientID)
}

// GetAuditTrail retrieves the audit trail summary for a transaction
func (m *Manager) GetAuditTrail(ctx context.Context, txnID uuid.UUID) (*AuditTrailSummary, error) {
	txn, err := m.GetTransaction(ctx, txnID)
	if err != nil {
		return nil, err
	}
	return txn.AuditTrail, nil
}

// =============================================================================
// IN-MEMORY STORE (for testing)
// =============================================================================

// InMemoryStore is a simple in-memory transaction store for testing
type InMemoryStore struct {
	transactions map[uuid.UUID]*Transaction
	mu           sync.RWMutex
}

// NewInMemoryStore creates a new in-memory store
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		transactions: make(map[uuid.UUID]*Transaction),
	}
}

// Save stores a transaction
func (s *InMemoryStore) Save(ctx context.Context, txn *Transaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.transactions[txn.ID] = txn
	return nil
}

// Get retrieves a transaction by ID
func (s *InMemoryStore) Get(ctx context.Context, id uuid.UUID) (*Transaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	txn, exists := s.transactions[id]
	if !exists {
		return nil, nil
	}
	return txn, nil
}

// GetByPatient retrieves all transactions for a patient
func (s *InMemoryStore) GetByPatient(ctx context.Context, patientID uuid.UUID) ([]*Transaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*Transaction
	for _, txn := range s.transactions {
		if txn.PatientID == patientID {
			result = append(result, txn)
		}
	}
	return result, nil
}

// Update updates a transaction
func (s *InMemoryStore) Update(ctx context.Context, txn *Transaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.transactions[txn.ID] = txn
	return nil
}

// Delete removes a transaction
func (s *InMemoryStore) Delete(ctx context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.transactions, id)
	return nil
}

// =============================================================================
// HELPER FUNCTIONS FOR ALLERGY/CONDITION CONVERSION
// =============================================================================

// inferAllergenType determines the allergen type from the code system
// V3 FIX: Med-Advisor expects allergen_type to be one of: drug, food, environmental
func inferAllergenType(system string) string {
	switch system {
	case "RxNorm", "http://www.nlm.nih.gov/research/umls/rxnorm", "RXNORM":
		return "drug"
	case "NDFRT", "http://hl7.org/fhir/ndfrt":
		return "drug"
	case "SNOMED", "http://snomed.info/sct", "SNOMEDCT":
		// SNOMED can represent any allergen type, default to drug for safety
		return "drug"
	case "UNII", "http://fdasis.nlm.nih.gov":
		return "drug"
	default:
		// Conservative default: treat as drug allergy for maximum safety
		return "drug"
	}
}

// getCodeBySystem extracts the code if it matches the requested system, empty otherwise
// V3 FIX: Med-Advisor expects separate fields for ICD10 and SNOMED codes
func getCodeBySystem(code ClinicalCode, targetSystem string) string {
	normalizedSystem := normalizeCodeSystem(code.System)
	if normalizedSystem == targetSystem {
		return code.Code
	}
	return ""
}

// normalizeCodeSystem converts various system URIs to a standard name
func normalizeCodeSystem(system string) string {
	switch system {
	case "ICD10", "http://hl7.org/fhir/sid/icd-10", "http://hl7.org/fhir/sid/icd-10-cm", "ICD-10":
		return "ICD10"
	case "SNOMED", "http://snomed.info/sct", "SNOMEDCT", "http://snomed.info/sct/900000000000207008":
		return "SNOMED"
	case "RxNorm", "http://www.nlm.nih.gov/research/umls/rxnorm", "RXNORM":
		return "RxNorm"
	case "LOINC", "http://loinc.org":
		return "LOINC"
	default:
		return system
	}
}
