package test

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cardiofit/medication-advisor-engine/advisor"
	"github.com/cardiofit/medication-advisor-engine/evidence"
	"github.com/cardiofit/medication-advisor-engine/kbclients"
	"github.com/cardiofit/medication-advisor-engine/snapshot"
)

// =============================================================================
// KB Service Configuration
// Uses REAL KB services via HTTP - NO MOCKS
// Port mappings align with actual Docker KB services from CLAUDE.md
// =============================================================================

// getKBServiceURLs returns URLs for real KB services.
// Ports verified against running Docker containers from shared-infrastructure/knowledge-base-services.
func getKBServiceURLs() kbclients.KBManagerConfig {
	// Check for environment variable overrides for CI/CD flexibility
	baseHost := os.Getenv("KB_HOST")
	if baseHost == "" {
		baseHost = "http://localhost"
	}

	return kbclients.KBManagerConfig{
		// KB-1 Dosing → kb1-drug-rules container (port 8081)
		KB1URL: getEnvOrDefault("KB1_URL", baseHost+":8081"),

		// KB-2 Interactions → kb5-drug-interactions container (port 8095)
		KB2URL: getEnvOrDefault("KB2_URL", baseHost+":8095"),

		// KB-3 Guidelines → kb3-guidelines container (port 8083)
		KB3URL: getEnvOrDefault("KB3_URL", baseHost+":8083"),

		// KB-4 Safety → kb4-patient-safety container (port 8088)
		KB4URL: getEnvOrDefault("KB4_URL", baseHost+":8088"),

		// KB-5 Monitoring → kb7-terminology container (port 8092)
		KB5URL: getEnvOrDefault("KB5_URL", baseHost+":8092"),

		// KB-6 Efficacy → kb6-formulary container (port 8087 HTTP, 8086 is gRPC)
		KB6URL: getEnvOrDefault("KB6_URL", baseHost+":8087"),

		Timeout:       30 * time.Second,
		RetryAttempts: 3,
	}
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// checkKBServicesAvailable verifies that KB services are reachable.
// Tests will be skipped if services are not running.
func checkKBServicesAvailable(t *testing.T) {
	t.Helper()

	config := getKBServiceURLs()
	client := &http.Client{Timeout: 2 * time.Second}

	// Check at least one critical service (KB-1 Drug Rules)
	resp, err := client.Get(config.KB1URL + "/health")
	if err != nil {
		t.Skipf("KB services not available (KB-1: %s): %v. Start services with 'make run-kb-docker' in shared-infrastructure/knowledge-base-services/", config.KB1URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		t.Skipf("KB-1 service unhealthy (status %d). Ensure KB services are running.", resp.StatusCode)
	}
}

// Mock stores for testing

type MockSnapshotStore struct {
	snapshots map[string]*snapshot.ClinicalSnapshot
}

func NewMockSnapshotStore() *MockSnapshotStore {
	return &MockSnapshotStore{
		snapshots: make(map[string]*snapshot.ClinicalSnapshot),
	}
}

func (s *MockSnapshotStore) Save(ctx context.Context, snap *snapshot.ClinicalSnapshot) error {
	s.snapshots[snap.ID.String()] = snap
	return nil
}

func (s *MockSnapshotStore) Get(ctx context.Context, id string) (*snapshot.ClinicalSnapshot, error) {
	return s.snapshots[id], nil
}

func (s *MockSnapshotStore) Delete(ctx context.Context, id string) error {
	delete(s.snapshots, id)
	return nil
}

func (s *MockSnapshotStore) List(ctx context.Context, filters snapshot.SnapshotFilters) ([]*snapshot.ClinicalSnapshot, error) {
	return nil, nil
}

func (s *MockSnapshotStore) UpdateStatus(ctx context.Context, id string, status snapshot.SnapshotStatus) error {
	if snap, ok := s.snapshots[id]; ok {
		snap.Status = status
	}
	return nil
}

type MockEnvelopeStore struct {
	envelopes map[string]*evidence.EvidenceEnvelope
}

func NewMockEnvelopeStore() *MockEnvelopeStore {
	return &MockEnvelopeStore{
		envelopes: make(map[string]*evidence.EvidenceEnvelope),
	}
}

func (s *MockEnvelopeStore) Save(ctx context.Context, env *evidence.EvidenceEnvelope) error {
	s.envelopes[env.ID.String()] = env
	return nil
}

func (s *MockEnvelopeStore) Get(ctx context.Context, id string) (*evidence.EvidenceEnvelope, error) {
	return s.envelopes[id], nil
}

func (s *MockEnvelopeStore) GetBySnapshot(ctx context.Context, snapshotID string) (*evidence.EvidenceEnvelope, error) {
	for _, env := range s.envelopes {
		if env.SnapshotID.String() == snapshotID {
			return env, nil
		}
	}
	return nil, nil
}

func (s *MockEnvelopeStore) List(ctx context.Context, patientID string, limit int) ([]*evidence.EvidenceEnvelope, error) {
	return nil, nil
}

// Test helpers

// createTestEngine creates a MedicationAdvisorEngine connected to REAL KB services.
// This is NOT a mock - it connects to actual running KB services via HTTP.
// Tests that call this function should first call checkKBServicesAvailable(t).
func createTestEngine() *advisor.MedicationAdvisorEngine {
	kbConfig := getKBServiceURLs()

	config := advisor.EngineConfig{
		Environment:        "test",
		SnapshotTTLMinutes: 30,
		// Real KB service URLs from Docker containers
		KB1URL: kbConfig.KB1URL,
		KB2URL: kbConfig.KB2URL,
		KB3URL: kbConfig.KB3URL,
		KB4URL: kbConfig.KB4URL,
		KB5URL: kbConfig.KB5URL,
		KB6URL: kbConfig.KB6URL,
	}

	// Create REAL KB manager with HTTP clients - NO MOCKS
	realKBManager, err := kbclients.NewKBManager(kbConfig)
	if err != nil {
		panic("failed to create KB manager with real services: " + err.Error())
	}

	// Create workflow orchestrator with real KB manager
	workflowEngine, err := advisor.NewWorkflowOrchestratorWithKBManager(config, realKBManager)
	if err != nil {
		panic("failed to create workflow orchestrator: " + err.Error())
	}

	// Create engine with real workflow orchestrator
	engine := advisor.NewTestMedicationAdvisorEngine(
		NewMockSnapshotStore(),
		NewMockEnvelopeStore(),
		config,
		workflowEngine,
	)

	return engine
}

func createTestCalculateRequest() *advisor.CalculateRequest {
	weight := 70.0
	height := 175.0
	egfr := 45.0

	return &advisor.CalculateRequest{
		PatientID:  uuid.New(),
		ProviderID: "dr-smith",
		SessionID:  "session-123",
		ClinicalQuestion: advisor.ClinicalQuestion{
			Text:            "Add SGLT2i for Type 2 Diabetes with CKD",
			Intent:          "ADD_MEDICATION",
			TargetDrugClass: "SGLT2i",
			Indication:      "Type 2 Diabetes with CKD",
		},
		PatientContext: advisor.PatientContext{
			Age:      72,
			Sex:      "male",
			WeightKg: &weight,
			HeightCm: &height,
			Conditions: []advisor.ClinicalCode{
				{System: "SNOMED", Code: "44054006", Display: "Type 2 Diabetes Mellitus"},
				{System: "SNOMED", Code: "709044004", Display: "Chronic Kidney Disease Stage 3b"},
			},
			Medications: []advisor.ClinicalCode{
				{System: "RxNorm", Code: "6809", Display: "Metformin 1000mg"},
			},
			Allergies: []advisor.ClinicalCode{},
			ComputedScores: snapshot.ComputedScores{
				EGFR:                        &egfr,
				CKDStage:                    "G3b",
				RequiresRenalDoseAdjustment: true,
			},
		},
	}
}

// =============================================================================
// Integration Tests - Require REAL KB services
// Run: go test ./test/... (with KB services running)
// Skip: Tests automatically skip if KB services are not available
// =============================================================================

func TestCalculatePhase(t *testing.T) {
	checkKBServicesAvailable(t)
	engine := createTestEngine()
	ctx := context.Background()
	req := createTestCalculateRequest()

	resp, err := engine.Calculate(ctx, req)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, resp.SnapshotID)
	assert.NotEqual(t, uuid.Nil, resp.EnvelopeID)
	assert.Greater(t, len(resp.Proposals), 0)
	assert.GreaterOrEqual(t, resp.ExecutionTimeMs, int64(0)) // Can be 0 for fast executions
}

func TestCalculateReturnsRankedProposals(t *testing.T) {
	checkKBServicesAvailable(t)
	engine := createTestEngine()
	ctx := context.Background()
	req := createTestCalculateRequest()

	resp, err := engine.Calculate(ctx, req)

	require.NoError(t, err)
	require.Greater(t, len(resp.Proposals), 0)

	// Verify proposals are ranked
	for i, proposal := range resp.Proposals {
		assert.Equal(t, i+1, proposal.Rank)
		assert.Greater(t, proposal.QualityScore, 0.0)
		assert.LessOrEqual(t, proposal.QualityScore, 1.0)
	}
}

func TestCalculateIncludesQualityFactors(t *testing.T) {
	checkKBServicesAvailable(t)
	engine := createTestEngine()
	ctx := context.Background()
	req := createTestCalculateRequest()

	resp, err := engine.Calculate(ctx, req)

	require.NoError(t, err)
	require.Greater(t, len(resp.Proposals), 0)

	proposal := resp.Proposals[0]
	factors := proposal.QualityFactors

	// Verify all quality factors are present
	assert.Greater(t, factors.Guideline, 0.0)
	assert.Greater(t, factors.Safety, 0.0)
	assert.Greater(t, factors.Efficacy, 0.0)
	assert.Greater(t, factors.Interaction, 0.0)
	assert.Greater(t, factors.Monitoring, 0.0)
}

func TestValidatePhase(t *testing.T) {
	checkKBServicesAvailable(t)
	engine := createTestEngine()
	ctx := context.Background()
	calcReq := createTestCalculateRequest()

	// First calculate
	calcResp, err := engine.Calculate(ctx, calcReq)
	require.NoError(t, err)

	// Then validate
	valReq := &advisor.ValidateRequest{
		SnapshotID: calcResp.SnapshotID,
		ProposalID: calcResp.Proposals[0].ID,
	}

	valResp, err := engine.Validate(ctx, valReq)

	require.NoError(t, err)
	assert.True(t, valResp.Valid)
	assert.Equal(t, "proceed", valResp.Recommendation)
	assert.Equal(t, 0, len(valResp.HardConflicts))
}

func TestCommitPhase(t *testing.T) {
	checkKBServicesAvailable(t)
	engine := createTestEngine()
	ctx := context.Background()
	calcReq := createTestCalculateRequest()

	// Calculate
	calcResp, err := engine.Calculate(ctx, calcReq)
	require.NoError(t, err)

	// Validate - returns a validation snapshot ID
	valReq := &advisor.ValidateRequest{
		SnapshotID: calcResp.SnapshotID,
		ProposalID: calcResp.Proposals[0].ID,
	}
	valResp, err := engine.Validate(ctx, valReq)
	require.NoError(t, err)

	// Commit - must use ValidationSnapshotID and original EnvelopeID
	commitReq := &advisor.CommitRequest{
		SnapshotID:   valResp.ValidationSnapshotID, // Use validation snapshot ID
		EnvelopeID:   calcResp.EnvelopeID,          // Use envelope from Calculate
		ProposalID:   calcResp.Proposals[0].ID,
		ProviderID:   "dr-smith",
		Acknowledged: true,
	}

	commitResp, err := engine.Commit(ctx, commitReq)

	require.NoError(t, err)
	assert.Contains(t, commitResp.MedicationRequestID, "MedicationRequest/")
	assert.True(t, commitResp.EvidenceFinalized)
}

func TestFullWorkflow(t *testing.T) {
	checkKBServicesAvailable(t)
	engine := createTestEngine()
	ctx := context.Background()

	// Step 1: Calculate - creates calculation snapshot
	calcReq := createTestCalculateRequest()
	calcResp, err := engine.Calculate(ctx, calcReq)
	require.NoError(t, err)
	require.Greater(t, len(calcResp.Proposals), 0)

	t.Logf("Calculate: Got %d proposals", len(calcResp.Proposals))
	t.Logf("Top proposal: %s (score: %.2f)",
		calcResp.Proposals[0].Medication.Display,
		calcResp.Proposals[0].QualityScore)

	// Step 2: Validate - creates validation snapshot from calculation snapshot
	valReq := &advisor.ValidateRequest{
		SnapshotID: calcResp.SnapshotID,
		ProposalID: calcResp.Proposals[0].ID,
	}
	valResp, err := engine.Validate(ctx, valReq)
	require.NoError(t, err)

	t.Logf("Validate: Valid=%t, Recommendation=%s", valResp.Valid, valResp.Recommendation)

	if !valResp.Valid {
		t.Logf("Hard conflicts: %d", len(valResp.HardConflicts))
		return
	}

	// Step 3: Commit - uses validation snapshot ID and original envelope ID
	commitReq := &advisor.CommitRequest{
		SnapshotID:   valResp.ValidationSnapshotID, // Must use validation snapshot ID
		EnvelopeID:   calcResp.EnvelopeID,          // Use envelope from Calculate
		ProposalID:   calcResp.Proposals[0].ID,
		ProviderID:   "dr-smith",
		Acknowledged: true,
	}
	commitResp, err := engine.Commit(ctx, commitReq)
	require.NoError(t, err)

	t.Logf("Commit: MedicationRequest=%s, Evidence Finalized=%t",
		commitResp.MedicationRequestID,
		commitResp.EvidenceFinalized)

	assert.True(t, commitResp.EvidenceFinalized)
}

func TestSnapshotExpiration(t *testing.T) {
	store := NewMockSnapshotStore()
	manager := snapshot.NewSnapshotManager(store, 30)
	ctx := context.Background()

	// Create snapshot
	snap, err := manager.CreateCalculationSnapshot(
		ctx,
		uuid.New(),
		uuid.New(),
		snapshot.ClinicalSnapshotData{},
		snapshot.ComputedScores{},
		"test",
	)
	require.NoError(t, err)

	// Verify it's valid
	assert.True(t, snap.IsValid())
	assert.False(t, snap.IsExpired())
}

func TestConflictDetection(t *testing.T) {
	detector := advisor.NewConflictDetector()

	snapshotData := snapshot.ClinicalSnapshotData{
		Demographics: snapshot.PatientDemographics{},
		Conditions:   []snapshot.ConditionEntry{},
		Allergies:    []snapshot.AllergyEntry{},
	}

	// Add new condition
	currentData := snapshot.ClinicalSnapshotData{
		Demographics: snapshot.PatientDemographics{},
		Conditions: []snapshot.ConditionEntry{
			{
				ID:            uuid.New(),
				ConditionName: "Acute Kidney Injury",
				SNOMEDCT:      "14669001",
				Status:        snapshot.ConditionStatusActive,
			},
		},
		Allergies: []snapshot.AllergyEntry{},
	}

	result := detector.ClassifyConflicts(snapshotData, currentData)

	assert.True(t, result.HasHardConflicts)
	assert.Equal(t, "abort", result.Recommendation)
	assert.Greater(t, len(result.HardConflicts), 0)
}

func TestScoringEngine(t *testing.T) {
	scorer := advisor.NewProposalScoringEngine()

	candidates := []advisor.MedicationCandidate{
		{
			Medication: advisor.ClinicalCode{Code: "1", Display: "Drug A"},
			Scores: advisor.QualityFactors{
				Guideline: 0.90, Safety: 0.85, Efficacy: 0.80, Interaction: 0.95, Monitoring: 0.85,
			},
		},
		{
			Medication: advisor.ClinicalCode{Code: "2", Display: "Drug B"},
			Scores: advisor.QualityFactors{
				Guideline: 0.70, Safety: 0.90, Efficacy: 0.75, Interaction: 0.85, Monitoring: 0.90,
			},
		},
	}

	proposals := scorer.RankProposals(candidates)

	assert.Len(t, proposals, 2)
	assert.Equal(t, 1, proposals[0].Rank)
	assert.Equal(t, 2, proposals[1].Rank)
	assert.GreaterOrEqual(t, proposals[0].QualityScore, proposals[1].QualityScore)
}

func TestInferenceChainBuilder(t *testing.T) {
	builder := evidence.NewInferenceChainBuilder(uuid.New())

	builder.
		AddRecipeStep("recipe-123", []string{"conditions", "medications"}, map[string]interface{}{"count": 2}).
		AddKBQueryStep("KB-4", "safety_check", map[string]string{"med": "metformin"}, map[string]bool{"safe": true}, "Metformin safe").
		AddScoringStep("Dapagliflozin", map[string]float64{"guideline": 0.9}, map[string]float64{"guideline": 0.3}, 0.89, "High quality score").
		AddRankingStep([]string{"Dapagliflozin"}, []float64{0.89}, []int{1}, "Ranked by quality")

	chain := builder.Build()

	assert.Len(t, chain, 4)
	assert.Equal(t, "recipe", chain[0].Phase)
	assert.Equal(t, "kb_query", chain[1].Phase)
	assert.Equal(t, "scoring", chain[2].Phase)
	assert.Equal(t, "ranking", chain[3].Phase)
}

func TestEngineHealth(t *testing.T) {
	checkKBServicesAvailable(t)
	engine := createTestEngine()

	health := engine.Health()

	assert.Equal(t, "healthy", health["status"])
	assert.Equal(t, "test", health["environment"])
	assert.NotNil(t, health["snapshot_metrics"])
	assert.NotNil(t, health["evidence_metrics"])
}

// =============================================================================
// Benchmark tests - Require REAL KB services
// Run: go test -bench=. ./test/... (with KB services running)
// =============================================================================

func BenchmarkCalculate(b *testing.B) {
	// Skip benchmark if KB services not available
	config := getKBServiceURLs()
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(config.KB1URL + "/health")
	if err != nil {
		b.Skipf("KB services not available: %v", err)
	}
	resp.Body.Close()

	engine := createTestEngine()
	ctx := context.Background()
	req := createTestCalculateRequest()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Calculate(ctx, req)
	}
}

func BenchmarkFullWorkflow(b *testing.B) {
	// Skip benchmark if KB services not available
	config := getKBServiceURLs()
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(config.KB1URL + "/health")
	if err != nil {
		b.Skipf("KB services not available: %v", err)
	}
	resp.Body.Close()

	engine := createTestEngine()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calcReq := createTestCalculateRequest()
		calcResp, _ := engine.Calculate(ctx, calcReq)

		valReq := &advisor.ValidateRequest{
			SnapshotID: calcResp.SnapshotID,
			ProposalID: calcResp.Proposals[0].ID,
		}
		valResp, _ := engine.Validate(ctx, valReq)

		commitReq := &advisor.CommitRequest{
			SnapshotID:   valResp.ValidationSnapshotID, // Use validation snapshot ID
			EnvelopeID:   calcResp.EnvelopeID,          // Use envelope from Calculate
			ProposalID:   calcResp.Proposals[0].ID,
			ProviderID:   "dr-smith",
			Acknowledged: true,
		}
		_, _ = engine.Commit(ctx, commitReq)
	}
}
