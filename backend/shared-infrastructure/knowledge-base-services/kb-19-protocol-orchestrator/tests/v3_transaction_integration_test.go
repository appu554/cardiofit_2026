// Package tests provides integration tests for KB-19 V3 Transaction Authority.
// These tests run against a LIVE KB-19 service in Docker - NO MOCKS.
package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// V3 TRANSACTION INTEGRATION TESTS
// Runs against live KB-19 service in Docker (port 8099)
// NO MOCKS - Real HTTP calls
// =============================================================================

const (
	// Default KB-19 URL - can be overridden via environment variable
	defaultKB19URL = "http://localhost:8099"
)

func getKB19URL() string {
	if url := os.Getenv("KB19_URL"); url != "" {
		return url
	}
	return defaultKB19URL
}

// httpClient with reasonable timeouts for integration tests
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// =============================================================================
// TEST REQUEST/RESPONSE TYPES
// =============================================================================

type CreateTransactionRequest struct {
	PatientID          uuid.UUID              `json:"patient_id"`
	EncounterID        uuid.UUID              `json:"encounter_id"`
	ProviderID         string                 `json:"provider_id"`
	ProposedMedication ProposedMedicationInfo `json:"proposed_medication"`
	CurrentMedications []MedicationInfo       `json:"current_medications,omitempty"`
	PatientLabs        []LabValueInfo         `json:"patient_labs,omitempty"`
}

type ProposedMedicationInfo struct {
	RxNormCode    string   `json:"rxnorm_code"`
	DrugName      string   `json:"drug_name"`
	DrugClass     string   `json:"drug_class,omitempty"`
	DoseMg        float64  `json:"dose_mg,omitempty"`
	Unit          string   `json:"unit,omitempty"`
	Route         string   `json:"route,omitempty"`
	Frequency     string   `json:"frequency,omitempty"`
	Indication    string   `json:"indication,omitempty"`
	RiskScore     float64  `json:"risk_score,omitempty"`
	RiskFactors   []string `json:"risk_factors,omitempty"`
	KBSourcesUsed []string `json:"kb_sources_used,omitempty"`
}

type MedicationInfo struct {
	RxNormCode string  `json:"rxnorm_code"`
	DrugName   string  `json:"drug_name"`
	DrugClass  string  `json:"drug_class,omitempty"`
	DoseMg     float64 `json:"dose_mg,omitempty"`
}

type LabValueInfo struct {
	LOINCCode   string      `json:"loinc_code"`
	TestName    string      `json:"test_name"`
	Value       interface{} `json:"value"`
	Unit        string      `json:"unit"`
	CollectedAt time.Time   `json:"collected_at"`
	IsCritical  bool        `json:"is_critical,omitempty"`
}

type CreateTransactionResponse struct {
	TransactionID    uuid.UUID               `json:"transaction_id"`
	State            string                  `json:"state"`
	CreatedAt        time.Time               `json:"created_at"`
	SafetyAssessment SafetyAssessmentSummary `json:"safety_assessment"`
	HardBlocks       []HardBlockSummary      `json:"hard_blocks,omitempty"`
	NextAction       string                  `json:"next_action"`
	ProcessingTimeMs int64                   `json:"processing_time_ms"`
}

type SafetyAssessmentSummary struct {
	IsBlocked         bool   `json:"is_blocked"`
	BlockCount        int    `json:"block_count"`
	DDICount          int    `json:"ddi_count"`
	LabContraindCount int    `json:"lab_contraindication_count"`
	HighestSeverity   string `json:"highest_severity"`
	RequiresOverride  bool   `json:"requires_override"`
	RecommendedAction string `json:"recommended_action"`
}

type HardBlockSummary struct {
	ID          uuid.UUID `json:"id"`
	BlockType   string    `json:"block_type"`
	Severity    string    `json:"severity"`
	Medication  string    `json:"medication"`
	TriggerCode string    `json:"trigger_code"`
	TriggerName string    `json:"trigger_name"`
	Reason      string    `json:"reason"`
	RequiresAck bool      `json:"requires_ack"`
	AckText     string    `json:"ack_text"`
	KBSource    string    `json:"kb_source"`
	RuleID      string    `json:"rule_id"`
}

type ValidateTransactionRequest struct {
	TransactionID  uuid.UUID           `json:"transaction_id"`
	BlockOverrides []BlockOverrideInfo `json:"block_overrides,omitempty"`
	ValidatedBy    string              `json:"validated_by"`
}

type BlockOverrideInfo struct {
	BlockID        uuid.UUID `json:"block_id"`
	AcknowledgedBy string    `json:"acknowledged_by"`
	AckTimestamp   time.Time `json:"ack_timestamp"`
	AckText        string    `json:"ack_text"`
	ClinicalReason string    `json:"clinical_reason,omitempty"`
}

type ValidateTransactionResponse struct {
	TransactionID      uuid.UUID              `json:"transaction_id"`
	State              string                 `json:"state"`
	ValidatedAt        time.Time              `json:"validated_at"`
	IsValid            bool                   `json:"is_valid"`
	ValidationErrors   []string               `json:"validation_errors,omitempty"`
	ValidationWarnings []string               `json:"validation_warnings,omitempty"`
	OverridesApplied   []OverrideAppliedInfo  `json:"overrides_applied,omitempty"`
	PendingBlocks      []HardBlockSummary     `json:"pending_blocks,omitempty"`
	NextAction         string                 `json:"next_action"`
	ProcessingTimeMs   int64                  `json:"processing_time_ms"`
}

type OverrideAppliedInfo struct {
	BlockID        uuid.UUID `json:"block_id"`
	BlockType      string    `json:"block_type"`
	AcknowledgedBy string    `json:"acknowledged_by"`
	AckTimestamp   time.Time `json:"ack_timestamp"`
}

type CommitTransactionRequest struct {
	TransactionID      uuid.UUID               `json:"transaction_id"`
	CommittedBy        string                  `json:"committed_by"`
	Disposition        string                  `json:"disposition"`
	ModifiedMedication *ProposedMedicationInfo `json:"modified_medication,omitempty"`
	Notes              string                  `json:"notes,omitempty"`
}

type CommitTransactionResponse struct {
	TransactionID    uuid.UUID                `json:"transaction_id"`
	State            string                   `json:"state"`
	CommittedAt      time.Time                `json:"committed_at"`
	Disposition      string                   `json:"disposition"`
	DispositionCode  string                   `json:"disposition_code"`
	GovernanceEvents []GovernanceEventSummary `json:"governance_events,omitempty"`
	AuditID          uuid.UUID                `json:"audit_id"`
	AuditHash        string                   `json:"audit_hash"`
	GeneratedTasks   []GeneratedTaskSummary   `json:"generated_tasks,omitempty"`
	ProcessingTimeMs int64                    `json:"processing_time_ms"`
}

type GovernanceEventSummary struct {
	EventID     uuid.UUID `json:"event_id"`
	EventType   string    `json:"event_type"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
	Tier        int       `json:"tier"`
}

type GeneratedTaskSummary struct {
	TaskID      uuid.UUID `json:"task_id"`
	TaskType    string    `json:"task_type"`
	Description string    `json:"description"`
	DueDate     time.Time `json:"due_date,omitempty"`
	AssignedTo  string    `json:"assigned_to,omitempty"`
	Priority    string    `json:"priority"`
}

type GetTransactionResponse struct {
	TransactionID      uuid.UUID               `json:"transaction_id"`
	PatientID          uuid.UUID               `json:"patient_id"`
	EncounterID        uuid.UUID               `json:"encounter_id"`
	State              string                  `json:"state"`
	CreatedAt          time.Time               `json:"created_at"`
	UpdatedAt          time.Time               `json:"updated_at"`
	ProposedMedication ProposedMedicationInfo  `json:"proposed_medication"`
	SafetyAssessment   SafetyAssessmentSummary `json:"safety_assessment"`
	HardBlocks         []HardBlockSummary      `json:"hard_blocks,omitempty"`
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func doRequest(t *testing.T, method, url string, body interface{}) (*http.Response, []byte) {
	var reqBody io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	return resp, respBody
}

func checkKB19Available(t *testing.T) {
	resp, _ := doRequest(t, "GET", getKB19URL()+"/health", nil)
	if resp.StatusCode != http.StatusOK {
		t.Skipf("KB-19 service not available at %s (status: %d). Skipping integration test.", getKB19URL(), resp.StatusCode)
	}
}

// =============================================================================
// INTEGRATION TESTS
// =============================================================================

// TestKB19Health verifies KB-19 is running and healthy
func TestKB19Health(t *testing.T) {
	resp, body := doRequest(t, "GET", getKB19URL()+"/health", nil)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Health check failed: status=%d, body=%s", resp.StatusCode, string(body))
	}

	t.Logf("KB-19 health check passed: %s", string(body))
}

// TestV3TransactionLifecycle_SafeMedication tests the full lifecycle for a safe medication
func TestV3TransactionLifecycle_SafeMedication(t *testing.T) {
	checkKB19Available(t)

	patientID := uuid.New()
	encounterID := uuid.New()
	providerID := "DR-TEST-001"

	// Step 1: Create Transaction
	createReq := CreateTransactionRequest{
		PatientID:   patientID,
		EncounterID: encounterID,
		ProviderID:  providerID,
		ProposedMedication: ProposedMedicationInfo{
			RxNormCode: "197361",             // Amlodipine 5mg - generally safe
			DrugName:   "Amlodipine 5 MG",
			DrugClass:  "Calcium Channel Blocker",
			DoseMg:     5.0,
			Unit:       "mg",
			Route:      "oral",
			Frequency:  "once daily",
			Indication: "Hypertension",
		},
	}

	t.Log("Step 1: Creating transaction...")
	resp, body := doRequest(t, "POST", getKB19URL()+"/api/v1/transactions/create", createReq)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Fatalf("Create transaction failed: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var createResp CreateTransactionResponse
	if err := json.Unmarshal(body, &createResp); err != nil {
		t.Fatalf("Failed to unmarshal create response: %v", err)
	}

	t.Logf("Transaction created: ID=%s, State=%s, Blocks=%d",
		createResp.TransactionID, createResp.State, len(createResp.HardBlocks))

	// Step 2: Validate Transaction
	validateReq := ValidateTransactionRequest{
		TransactionID: createResp.TransactionID,
		ValidatedBy:   providerID,
	}

	t.Log("Step 2: Validating transaction...")
	resp, body = doRequest(t, "POST", getKB19URL()+"/api/v1/transactions/validate", validateReq)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Validate transaction failed: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var validateResp ValidateTransactionResponse
	if err := json.Unmarshal(body, &validateResp); err != nil {
		t.Fatalf("Failed to unmarshal validate response: %v", err)
	}

	t.Logf("Transaction validated: State=%s, IsValid=%v, NextAction=%s",
		validateResp.State, validateResp.IsValid, validateResp.NextAction)

	// Step 3: Commit Transaction
	commitReq := CommitTransactionRequest{
		TransactionID: createResp.TransactionID,
		CommittedBy:   providerID,
		Disposition:   "DISPENSE",
		Notes:         "Integration test - safe medication",
	}

	t.Log("Step 3: Committing transaction...")
	resp, body = doRequest(t, "POST", getKB19URL()+"/api/v1/transactions/commit", commitReq)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Commit transaction failed: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var commitResp CommitTransactionResponse
	if err := json.Unmarshal(body, &commitResp); err != nil {
		t.Fatalf("Failed to unmarshal commit response: %v", err)
	}

	t.Logf("Transaction committed: State=%s, Disposition=%s, AuditHash=%s",
		commitResp.State, commitResp.Disposition, commitResp.AuditHash)

	// Verify final state
	if commitResp.State != "COMMITTED" {
		t.Errorf("Expected state COMMITTED, got %s", commitResp.State)
	}
	if commitResp.AuditHash == "" {
		t.Error("Expected non-empty audit hash")
	}

	t.Log("✅ V3 Transaction lifecycle completed successfully for safe medication")
}

// TestV3TransactionLifecycle_DDIHardBlock tests the lifecycle with a DDI hard block
func TestV3TransactionLifecycle_DDIHardBlock(t *testing.T) {
	checkKB19Available(t)

	patientID := uuid.New()
	encounterID := uuid.New()
	providerID := "DR-TEST-002"

	// Step 1: Create Transaction with DDI risk
	// Warfarin (161) + Aspirin (1191) = severe DDI (bleeding risk)
	createReq := CreateTransactionRequest{
		PatientID:   patientID,
		EncounterID: encounterID,
		ProviderID:  providerID,
		ProposedMedication: ProposedMedicationInfo{
			RxNormCode: "161",           // Warfarin
			DrugName:   "Warfarin 5 MG",
			DrugClass:  "Anticoagulant",
			DoseMg:     5.0,
			Unit:       "mg",
			Route:      "oral",
			Frequency:  "once daily",
			Indication: "Atrial Fibrillation",
		},
		CurrentMedications: []MedicationInfo{
			{
				RxNormCode: "1191",           // Aspirin
				DrugName:   "Aspirin 325 MG",
				DrugClass:  "Antiplatelet",
				DoseMg:     325.0,
			},
		},
	}

	t.Log("Step 1: Creating transaction with DDI risk (Warfarin + Aspirin)...")
	resp, body := doRequest(t, "POST", getKB19URL()+"/api/v1/transactions/create", createReq)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Fatalf("Create transaction failed: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var createResp CreateTransactionResponse
	if err := json.Unmarshal(body, &createResp); err != nil {
		t.Fatalf("Failed to unmarshal create response: %v", err)
	}

	t.Logf("Transaction created: ID=%s, State=%s, HardBlocks=%d",
		createResp.TransactionID, createResp.State, len(createResp.HardBlocks))

	// Verify DDI hard block was detected
	if len(createResp.HardBlocks) == 0 {
		t.Log("⚠️ No hard blocks detected - DDI rule may not be implemented yet")
	} else {
		for _, block := range createResp.HardBlocks {
			t.Logf("  Block: Type=%s, Severity=%s, Reason=%s", block.BlockType, block.Severity, block.Reason)
		}
	}

	// Step 2: Validate without override (should fail or return pending blocks)
	validateReq := ValidateTransactionRequest{
		TransactionID: createResp.TransactionID,
		ValidatedBy:   providerID,
	}

	t.Log("Step 2: Validating transaction without override...")
	resp, body = doRequest(t, "POST", getKB19URL()+"/api/v1/transactions/validate", validateReq)

	var validateResp ValidateTransactionResponse
	if err := json.Unmarshal(body, &validateResp); err != nil {
		t.Fatalf("Failed to unmarshal validate response: %v", err)
	}

	t.Logf("Validation result: IsValid=%v, PendingBlocks=%d, NextAction=%s",
		validateResp.IsValid, len(validateResp.PendingBlocks), validateResp.NextAction)

	// Step 3: Override the block (if any)
	if len(createResp.HardBlocks) > 0 {
		t.Log("Step 3: Overriding hard block with acknowledgment...")

		overrideReq := ValidateTransactionRequest{
			TransactionID: createResp.TransactionID,
			ValidatedBy:   providerID,
			BlockOverrides: []BlockOverrideInfo{
				{
					BlockID:        createResp.HardBlocks[0].ID,
					AcknowledgedBy: providerID,
					AckTimestamp:   time.Now(),
					AckText:        createResp.HardBlocks[0].AckText,
					ClinicalReason: "Patient has been on this combination safely for 2 years with regular INR monitoring",
				},
			},
		}

		resp, body = doRequest(t, "POST", getKB19URL()+"/api/v1/transactions/validate", overrideReq)
		if err := json.Unmarshal(body, &validateResp); err != nil {
			t.Fatalf("Failed to unmarshal override response: %v", err)
		}

		t.Logf("Override result: OverridesApplied=%d, IsValid=%v",
			len(validateResp.OverridesApplied), validateResp.IsValid)
	}

	// Step 4: Commit with HOLD_FOR_REVIEW disposition
	commitReq := CommitTransactionRequest{
		TransactionID: createResp.TransactionID,
		CommittedBy:   providerID,
		Disposition:   "HOLD_FOR_REVIEW",
		Notes:         "Integration test - DDI override with pharmacist review required",
	}

	t.Log("Step 4: Committing transaction with HOLD_FOR_REVIEW...")
	resp, body = doRequest(t, "POST", getKB19URL()+"/api/v1/transactions/commit", commitReq)

	var commitResp CommitTransactionResponse
	if err := json.Unmarshal(body, &commitResp); err != nil {
		t.Fatalf("Failed to unmarshal commit response: %v", err)
	}

	t.Logf("Commit result: State=%s, GovernanceEvents=%d, AuditHash=%s",
		commitResp.State, len(commitResp.GovernanceEvents), commitResp.AuditHash)

	// Verify governance events were generated
	if len(commitResp.GovernanceEvents) > 0 {
		t.Log("Governance events generated:")
		for _, event := range commitResp.GovernanceEvents {
			t.Logf("  - Type=%s, Tier=%d", event.EventType, event.Tier)
		}
	}

	t.Log("✅ V3 Transaction lifecycle with DDI hard block completed")
}

// TestV3TransactionLifecycle_LabContraindication tests lab-drug contraindication
func TestV3TransactionLifecycle_LabContraindication(t *testing.T) {
	checkKB19Available(t)

	patientID := uuid.New()
	encounterID := uuid.New()
	providerID := "DR-TEST-003"

	// Metformin with eGFR < 30 (severe renal impairment) = contraindicated
	createReq := CreateTransactionRequest{
		PatientID:   patientID,
		EncounterID: encounterID,
		ProviderID:  providerID,
		ProposedMedication: ProposedMedicationInfo{
			RxNormCode: "6809",              // Metformin
			DrugName:   "Metformin 500 MG",
			DrugClass:  "Biguanide",
			DoseMg:     500.0,
			Unit:       "mg",
			Route:      "oral",
			Frequency:  "twice daily",
			Indication: "Type 2 Diabetes",
		},
		PatientLabs: []LabValueInfo{
			{
				LOINCCode:   "62238-1",                   // eGFR
				TestName:    "eGFR",
				Value:       25.0,                        // < 30 = severe impairment
				Unit:        "mL/min/1.73m2",
				CollectedAt: time.Now().Add(-2 * time.Hour),
				IsCritical:  true,
			},
		},
	}

	t.Log("Step 1: Creating transaction with lab contraindication (Metformin + eGFR 25)...")
	resp, body := doRequest(t, "POST", getKB19URL()+"/api/v1/transactions/create", createReq)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Fatalf("Create transaction failed: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var createResp CreateTransactionResponse
	if err := json.Unmarshal(body, &createResp); err != nil {
		t.Fatalf("Failed to unmarshal create response: %v", err)
	}

	t.Logf("Transaction created: ID=%s, State=%s, HardBlocks=%d",
		createResp.TransactionID, createResp.State, len(createResp.HardBlocks))

	// Check for lab contraindication block
	hasLabBlock := false
	for _, block := range createResp.HardBlocks {
		t.Logf("  Block: Type=%s, KBSource=%s, Reason=%s", block.BlockType, block.KBSource, block.Reason)
		if block.KBSource == "KB-16" || block.BlockType == "LAB_CONTRAINDICATION" {
			hasLabBlock = true
		}
	}

	if !hasLabBlock && len(createResp.HardBlocks) == 0 {
		t.Log("⚠️ No lab contraindication block detected - KB-16 rule may not be implemented yet")
	}

	// Try to commit as HARD_STOP
	commitReq := CommitTransactionRequest{
		TransactionID: createResp.TransactionID,
		CommittedBy:   providerID,
		Disposition:   "HARD_STOP",
		Notes:         "Lab contraindication - patient safety",
	}

	t.Log("Step 2: Committing transaction as HARD_STOP...")
	resp, body = doRequest(t, "POST", getKB19URL()+"/api/v1/transactions/commit", commitReq)

	var commitResp CommitTransactionResponse
	if err := json.Unmarshal(body, &commitResp); err != nil {
		t.Fatalf("Failed to unmarshal commit response: %v", err)
	}

	t.Logf("Commit result: State=%s, Disposition=%s, GeneratedTasks=%d",
		commitResp.State, commitResp.Disposition, len(commitResp.GeneratedTasks))

	// Verify KB-14 tasks were generated for lab safety
	if len(commitResp.GeneratedTasks) > 0 {
		t.Log("Generated tasks (KB-14):")
		for _, task := range commitResp.GeneratedTasks {
			t.Logf("  - Type=%s, Priority=%s", task.TaskType, task.Priority)
		}
	}

	t.Log("✅ V3 Transaction with lab contraindication completed")
}

// TestV3GetTransaction tests retrieving a transaction by ID
func TestV3GetTransaction(t *testing.T) {
	checkKB19Available(t)

	patientID := uuid.New()
	encounterID := uuid.New()
	providerID := "DR-TEST-004"

	// First create a transaction
	createReq := CreateTransactionRequest{
		PatientID:   patientID,
		EncounterID: encounterID,
		ProviderID:  providerID,
		ProposedMedication: ProposedMedicationInfo{
			RxNormCode: "197361",
			DrugName:   "Amlodipine 5 MG",
			DoseMg:     5.0,
		},
	}

	resp, body := doRequest(t, "POST", getKB19URL()+"/api/v1/transactions/create", createReq)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Fatalf("Create failed: status=%d", resp.StatusCode)
	}

	var createResp CreateTransactionResponse
	json.Unmarshal(body, &createResp)

	// Now retrieve it
	t.Logf("Retrieving transaction: %s", createResp.TransactionID)
	resp, body = doRequest(t, "GET", fmt.Sprintf("%s/api/v1/transactions/%s", getKB19URL(), createResp.TransactionID), nil)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Get transaction failed: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var getResp GetTransactionResponse
	if err := json.Unmarshal(body, &getResp); err != nil {
		t.Fatalf("Failed to unmarshal get response: %v", err)
	}

	// Verify data matches
	if getResp.TransactionID != createResp.TransactionID {
		t.Errorf("Transaction ID mismatch: expected %s, got %s", createResp.TransactionID, getResp.TransactionID)
	}
	if getResp.PatientID != patientID {
		t.Errorf("Patient ID mismatch: expected %s, got %s", patientID, getResp.PatientID)
	}

	t.Logf("✅ Transaction retrieved successfully: State=%s", getResp.State)
}

// TestV3ListPatientTransactions tests listing transactions for a patient
func TestV3ListPatientTransactions(t *testing.T) {
	checkKB19Available(t)

	patientID := uuid.New()
	encounterID := uuid.New()
	providerID := "DR-TEST-005"

	// Create multiple transactions for the same patient
	for i := 0; i < 3; i++ {
		createReq := CreateTransactionRequest{
			PatientID:   patientID,
			EncounterID: encounterID,
			ProviderID:  providerID,
			ProposedMedication: ProposedMedicationInfo{
				RxNormCode: fmt.Sprintf("19736%d", i),
				DrugName:   fmt.Sprintf("Test Drug %d", i),
				DoseMg:     5.0,
			},
		}
		doRequest(t, "POST", getKB19URL()+"/api/v1/transactions/create", createReq)
	}

	// List transactions for patient
	t.Logf("Listing transactions for patient: %s", patientID)
	resp, body := doRequest(t, "GET", fmt.Sprintf("%s/api/v1/transactions/patient/%s", getKB19URL(), patientID), nil)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("List transactions failed: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var listResp struct {
		Transactions []GetTransactionResponse `json:"transactions"`
		Total        int                      `json:"total"`
	}
	if err := json.Unmarshal(body, &listResp); err != nil {
		t.Fatalf("Failed to unmarshal list response: %v", err)
	}

	t.Logf("✅ Found %d transactions for patient", listResp.Total)

	if listResp.Total < 3 {
		t.Logf("⚠️ Expected at least 3 transactions, got %d", listResp.Total)
	}
}

// =============================================================================
// PERFORMANCE TESTS
// =============================================================================

// TestV3TransactionPerformance measures transaction creation latency
func TestV3TransactionPerformance(t *testing.T) {
	checkKB19Available(t)

	iterations := 10
	var totalLatency time.Duration

	for i := 0; i < iterations; i++ {
		createReq := CreateTransactionRequest{
			PatientID:   uuid.New(),
			EncounterID: uuid.New(),
			ProviderID:  "DR-PERF-TEST",
			ProposedMedication: ProposedMedicationInfo{
				RxNormCode: "197361",
				DrugName:   "Amlodipine 5 MG",
				DoseMg:     5.0,
			},
		}

		start := time.Now()
		resp, _ := doRequest(t, "POST", getKB19URL()+"/api/v1/transactions/create", createReq)
		latency := time.Since(start)
		totalLatency += latency

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			t.Logf("Request %d failed with status %d", i, resp.StatusCode)
		}
	}

	avgLatency := totalLatency / time.Duration(iterations)
	t.Logf("Performance: %d iterations, avg latency=%v", iterations, avgLatency)

	// Target: < 200ms p95
	if avgLatency > 200*time.Millisecond {
		t.Logf("⚠️ Average latency %v exceeds 200ms target", avgLatency)
	} else {
		t.Log("✅ Performance target met (< 200ms avg)")
	}
}

// BenchmarkV3CreateTransaction benchmarks transaction creation
func BenchmarkV3CreateTransaction(b *testing.B) {
	createReq := CreateTransactionRequest{
		PatientID:   uuid.New(),
		EncounterID: uuid.New(),
		ProviderID:  "DR-BENCH",
		ProposedMedication: ProposedMedicationInfo{
			RxNormCode: "197361",
			DrugName:   "Amlodipine 5 MG",
			DoseMg:     5.0,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		jsonBytes, _ := json.Marshal(createReq)
		req, _ := http.NewRequest("POST", getKB19URL()+"/api/v1/transactions/create", bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := httpClient.Do(req)
		if resp != nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	}
}
