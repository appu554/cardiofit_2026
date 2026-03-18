package services

import (
	"context"
	"math"
	"testing"

	"kb-22-hpi-engine/internal/models"
)

// ---------------------------------------------------------------------------
// mockDataResolver implements DataResolver for tests.
// ---------------------------------------------------------------------------

type mockDataResolver struct {
	resolvedData *models.ResolvedData
	err          error
}

func (m *mockDataResolver) Resolve(
	_ context.Context,
	_ string,
	_ []models.RequiredInput,
	_ []models.AggregatedInputDef,
) (*models.ResolvedData, error) {
	return m.resolvedData, m.err
}

// ---------------------------------------------------------------------------
// Shared test node definition (PM-03: Nocturnal BP Dipping)
// ---------------------------------------------------------------------------

func testPM03Node() *models.MonitoringNodeDefinition {
	return &models.MonitoringNodeDefinition{
		NodeID:  "PM-03",
		Version: "1.0.0",
		Type:    "MONITORING",
		TitleEN: "Nocturnal BP Dipping",
		ComputedFields: []models.ComputedFieldDef{
			{
				Name:    "dipping_ratio",
				Formula: "(sbp_nocturnal_mean - sbp_daytime_mean) / sbp_daytime_mean",
			},
		},
		Classifications: []models.ClassificationDef{
			{
				Category:          "REVERSE_DIPPER",
				Condition:         "dipping_ratio > 0.0",
				Severity:          "CRITICAL",
				MCUGateSuggestion: "PAUSE",
			},
			{
				Category:          "NON_DIPPER",
				Condition:         "dipping_ratio > -0.10",
				Severity:          "MODERATE",
				MCUGateSuggestion: "MODIFY",
			},
			{
				Category:          "NORMAL_DIPPER",
				Condition:         "dipping_ratio > -0.20",
				Severity:          "NONE",
				MCUGateSuggestion: "SAFE",
			},
			{
				Category:          "EXTREME_DIPPER",
				Condition:         "dipping_ratio <= -0.20",
				Severity:          "MODERATE",
				MCUGateSuggestion: "MODIFY",
			},
		},
		InsufficientData: models.InsufficientDataPolicy{
			Action: "FLAG_FOR_REVIEW",
		},
		SafetyTriggers: []models.MonitoringSafetyTrigger{
			{
				ID:        "ST-PM03-01",
				Condition: "sbp_nocturnal_mean > 160",
				Severity:  "URGENT",
				Action:    "ALERT_PHYSICIAN",
			},
		},
	}
}

// buildEngine constructs a MonitoringNodeEngine backed by an in-memory node map and
// the provided DataResolver. db is nil so persistence is skipped in tests.
func buildEngine(node *models.MonitoringNodeDefinition, resolver DataResolver) *MonitoringNodeEngine {
	loader := NewMonitoringNodeLoader("", testLogger())
	if node != nil {
		loader.mu.Lock()
		loader.nodes[node.NodeID] = node
		loader.mu.Unlock()
	}
	return NewMonitoringNodeEngine(loader, resolver, NewExpressionEvaluator(), NewTrajectoryComputer(testLogger()), nil, testLogger())
}

// ---------------------------------------------------------------------------
// TestMonitoringEngine_ClassifyNormalDipper
// dipping_ratio = (nocturnal - daytime) / daytime = (119 - 140) / 140 = -0.15
// Expected: NORMAL_DIPPER, severity=NONE, gate=SAFE
// ---------------------------------------------------------------------------

func TestMonitoringEngine_ClassifyNormalDipper(t *testing.T) {
	// sbp_nocturnal_mean=119, sbp_daytime_mean=140
	// dipping_ratio = (119 - 140) / 140 = -21/140 ≈ -0.15
	resolver := &mockDataResolver{
		resolvedData: &models.ResolvedData{
			Fields: map[string]float64{
				"sbp_nocturnal_mean": 119.0,
				"sbp_daytime_mean":   140.0,
			},
			Sufficiency: models.DataSufficient,
		},
	}

	engine := buildEngine(testPM03Node(), resolver)
	event, err := engine.Evaluate(context.Background(), "PM-03", "patient-001", "T2D")
	if err != nil {
		t.Fatalf("Evaluate() unexpected error: %v", err)
	}
	if event == nil {
		t.Fatal("Evaluate() returned nil event, expected an event")
	}

	if event.Classification == nil {
		t.Fatal("event.Classification is nil")
	}
	if event.Classification.Category != "NORMAL_DIPPER" {
		t.Errorf("Category: expected NORMAL_DIPPER, got %q", event.Classification.Category)
	}
	if event.Classification.DataSufficiency != string(models.DataSufficient) {
		t.Errorf("DataSufficiency: expected SUFFICIENT, got %q", event.Classification.DataSufficiency)
	}
	if event.MCUGateSuggestion == nil || *event.MCUGateSuggestion != "SAFE" {
		t.Errorf("MCUGateSuggestion: expected SAFE, got %v", event.MCUGateSuggestion)
	}
}

// ---------------------------------------------------------------------------
// TestMonitoringEngine_ClassifyReverseDipper
// dipping_ratio = (140 - 130) / 130 ≈ +0.077 > 0.0 → REVERSE_DIPPER
// Expected: REVERSE_DIPPER, severity=CRITICAL, gate=PAUSE
// ---------------------------------------------------------------------------

func TestMonitoringEngine_ClassifyReverseDipper(t *testing.T) {
	// sbp_nocturnal_mean > sbp_daytime_mean → positive ratio → REVERSE_DIPPER
	resolver := &mockDataResolver{
		resolvedData: &models.ResolvedData{
			Fields: map[string]float64{
				"sbp_nocturnal_mean": 140.0,
				"sbp_daytime_mean":   130.0,
			},
			Sufficiency: models.DataSufficient,
		},
	}

	engine := buildEngine(testPM03Node(), resolver)
	event, err := engine.Evaluate(context.Background(), "PM-03", "patient-002", "T2D")
	if err != nil {
		t.Fatalf("Evaluate() unexpected error: %v", err)
	}
	if event == nil {
		t.Fatal("Evaluate() returned nil event")
	}

	if event.Classification == nil {
		t.Fatal("event.Classification is nil")
	}
	if event.Classification.Category != "REVERSE_DIPPER" {
		t.Errorf("Category: expected REVERSE_DIPPER, got %q", event.Classification.Category)
	}
	if event.MCUGateSuggestion == nil || *event.MCUGateSuggestion != "PAUSE" {
		t.Errorf("MCUGateSuggestion: expected PAUSE, got %v", event.MCUGateSuggestion)
	}
}

// ---------------------------------------------------------------------------
// TestMonitoringEngine_InsufficientData_Skip
// DataResolver returns INSUFFICIENT → no event emitted, returns nil
// (node has action=SKIP)
// ---------------------------------------------------------------------------

func TestMonitoringEngine_InsufficientData_Skip(t *testing.T) {
	// Override node to use SKIP policy
	node := testPM03Node()
	node.InsufficientData = models.InsufficientDataPolicy{Action: "SKIP"}

	resolver := &mockDataResolver{
		resolvedData: &models.ResolvedData{
			Fields:      map[string]float64{},
			Sufficiency: models.DataInsufficient,
		},
	}

	engine := buildEngine(node, resolver)
	event, err := engine.Evaluate(context.Background(), "PM-03", "patient-003", "T2D")
	if err != nil {
		t.Fatalf("Evaluate() unexpected error: %v", err)
	}
	if event != nil {
		t.Errorf("expected nil event for SKIP policy on INSUFFICIENT data, got event with category %q",
			func() string {
				if event.Classification != nil {
					return event.Classification.Category
				}
				return "<nil>"
			}())
	}
}

// ---------------------------------------------------------------------------
// TestMonitoringEngine_InsufficientData_FlagForReview
// DataResolver returns INSUFFICIENT with FLAG_FOR_REVIEW policy →
// event emitted with data_sufficiency=INSUFFICIENT
// ---------------------------------------------------------------------------

func TestMonitoringEngine_InsufficientData_FlagForReview(t *testing.T) {
	// testPM03Node already has FLAG_FOR_REVIEW policy
	resolver := &mockDataResolver{
		resolvedData: &models.ResolvedData{
			Fields:      map[string]float64{},
			Sufficiency: models.DataInsufficient,
		},
	}

	engine := buildEngine(testPM03Node(), resolver)
	event, err := engine.Evaluate(context.Background(), "PM-03", "patient-004", "T2D")
	if err != nil {
		t.Fatalf("Evaluate() unexpected error: %v", err)
	}
	if event == nil {
		t.Fatal("expected non-nil event for FLAG_FOR_REVIEW policy, got nil")
	}
	if event.Classification == nil {
		t.Fatal("event.Classification is nil")
	}
	if event.Classification.DataSufficiency != string(models.DataInsufficient) {
		t.Errorf("DataSufficiency: expected INSUFFICIENT, got %q", event.Classification.DataSufficiency)
	}
}

// ---------------------------------------------------------------------------
// TestMonitoringEngine_ComputedFields
// sbp_nocturnal_mean=130, sbp_daytime_mean=140 → dipping_ratio = (130-140)/140 ≈ -0.0714
// Expected: NON_DIPPER (dipping_ratio > -0.10 → second classification)
// ---------------------------------------------------------------------------

func TestMonitoringEngine_ComputedFields(t *testing.T) {
	// (130 - 140) / 140 = -10/140 ≈ -0.07143
	resolver := &mockDataResolver{
		resolvedData: &models.ResolvedData{
			Fields: map[string]float64{
				"sbp_nocturnal_mean": 130.0,
				"sbp_daytime_mean":   140.0,
			},
			Sufficiency: models.DataSufficient,
		},
	}

	engine := buildEngine(testPM03Node(), resolver)
	event, err := engine.Evaluate(context.Background(), "PM-03", "patient-005", "T2D")
	if err != nil {
		t.Fatalf("Evaluate() unexpected error: %v", err)
	}
	if event == nil {
		t.Fatal("Evaluate() returned nil")
	}
	if event.Classification == nil {
		t.Fatal("event.Classification is nil")
	}

	// Verify the computed dipping_ratio results in correct classification
	// dipping_ratio ≈ -0.0714 → > -0.10 is true → NON_DIPPER (not REVERSE since ≤ 0.0)
	// Actually: REVERSE_DIPPER checks dipping_ratio > 0.0 → false
	//           NON_DIPPER checks dipping_ratio > -0.10 → -0.0714 > -0.10 → TRUE → NON_DIPPER
	if event.Classification.Category != "NON_DIPPER" {
		t.Errorf("Category: expected NON_DIPPER for ratio ≈ -0.071, got %q", event.Classification.Category)
	}

	// Verify value stored correctly: the computed dipping ratio
	expectedRatio := (130.0 - 140.0) / 140.0
	if math.Abs(event.Classification.Value-expectedRatio) > 1e-6 {
		t.Errorf("Classification.Value: expected %.6f, got %.6f", expectedRatio, event.Classification.Value)
	}
}

// ---------------------------------------------------------------------------
// TestMonitoringEngine_SafetyTrigger
// sbp_nocturnal_mean=170 > 160 → safety flag ST-PM03-01 fires
// ---------------------------------------------------------------------------

func TestMonitoringEngine_SafetyTrigger(t *testing.T) {
	resolver := &mockDataResolver{
		resolvedData: &models.ResolvedData{
			Fields: map[string]float64{
				"sbp_nocturnal_mean": 170.0,
				"sbp_daytime_mean":   150.0,
			},
			Sufficiency: models.DataSufficient,
		},
	}

	engine := buildEngine(testPM03Node(), resolver)
	event, err := engine.Evaluate(context.Background(), "PM-03", "patient-006", "T2D")
	if err != nil {
		t.Fatalf("Evaluate() unexpected error: %v", err)
	}
	if event == nil {
		t.Fatal("Evaluate() returned nil")
	}

	if len(event.SafetyFlags) == 0 {
		t.Fatal("expected at least one safety flag, got none")
	}

	found := false
	for _, flag := range event.SafetyFlags {
		if flag.FlagID == "ST-PM03-01" {
			found = true
			if flag.Severity != "URGENT" {
				t.Errorf("SafetyFlag severity: expected URGENT, got %q", flag.Severity)
			}
			if flag.Action != "ALERT_PHYSICIAN" {
				t.Errorf("SafetyFlag action: expected ALERT_PHYSICIAN, got %q", flag.Action)
			}
		}
	}
	if !found {
		t.Errorf("safety flag ST-PM03-01 not found in event.SafetyFlags: %+v", event.SafetyFlags)
	}
}

// ---------------------------------------------------------------------------
// TestMonitoringEngine_FirstMatchWins
// Uses a node with multiple matching classifications; the first one must win.
// dipping_ratio = +0.05 → REVERSE_DIPPER matches first (> 0.0).
// Without first-match semantics NON_DIPPER would also match (> -0.10).
// ---------------------------------------------------------------------------

func TestMonitoringEngine_FirstMatchWins(t *testing.T) {
	resolver := &mockDataResolver{
		resolvedData: &models.ResolvedData{
			Fields: map[string]float64{
				"sbp_nocturnal_mean": 136.5, // 136.5/130 - 1 = 0.05
				"sbp_daytime_mean":   130.0,
			},
			Sufficiency: models.DataSufficient,
		},
	}

	engine := buildEngine(testPM03Node(), resolver)
	event, err := engine.Evaluate(context.Background(), "PM-03", "patient-007", "T2D")
	if err != nil {
		t.Fatalf("Evaluate() unexpected error: %v", err)
	}
	if event == nil || event.Classification == nil {
		t.Fatal("expected non-nil event with classification")
	}

	// dipping_ratio = (136.5 - 130.0) / 130.0 = 0.05 > 0.0 → REVERSE_DIPPER
	// If first-match didn't work we might get NON_DIPPER (since 0.05 > -0.10 too)
	if event.Classification.Category != "REVERSE_DIPPER" {
		t.Errorf("first match should be REVERSE_DIPPER, got %q", event.Classification.Category)
	}
}

// ---------------------------------------------------------------------------
// TestMonitoringEngine_BuildsClinicalSignalEvent
// Verify the emitted event has the correct header fields.
// ---------------------------------------------------------------------------

func TestMonitoringEngine_BuildsClinicalSignalEvent(t *testing.T) {
	resolver := &mockDataResolver{
		resolvedData: &models.ResolvedData{
			Fields: map[string]float64{
				"sbp_nocturnal_mean": 115.0,
				"sbp_daytime_mean":   140.0,
			},
			Sufficiency: models.DataSufficient,
		},
	}

	engine := buildEngine(testPM03Node(), resolver)
	event, err := engine.Evaluate(context.Background(), "PM-03", "patient-008", "CKD_T2D")
	if err != nil {
		t.Fatalf("Evaluate() unexpected error: %v", err)
	}
	if event == nil {
		t.Fatal("Evaluate() returned nil")
	}

	// Header fields
	if event.EventID == "" {
		t.Error("EventID must not be empty")
	}
	if event.EventType != "CLINICAL_SIGNAL" {
		t.Errorf("EventType: expected CLINICAL_SIGNAL, got %q", event.EventType)
	}
	if event.SignalType != models.SignalMonitoringClassification {
		t.Errorf("SignalType: expected MONITORING_CLASSIFICATION, got %q", event.SignalType)
	}
	if event.NodeID != "PM-03" {
		t.Errorf("NodeID: expected PM-03, got %q", event.NodeID)
	}
	if event.NodeVersion != "1.0.0" {
		t.Errorf("NodeVersion: expected 1.0.0, got %q", event.NodeVersion)
	}
	if event.PatientID != "patient-008" {
		t.Errorf("PatientID: expected patient-008, got %q", event.PatientID)
	}
	if event.StratumLabel != "CKD_T2D" {
		t.Errorf("StratumLabel: expected CKD_T2D, got %q", event.StratumLabel)
	}
	if event.EmittedAt.IsZero() {
		t.Error("EmittedAt must not be zero")
	}

	// Classification should be populated
	if event.Classification == nil {
		t.Error("Classification must not be nil")
	}
}

// ---------------------------------------------------------------------------
// TestMonitoringEngine_NodeNotFound
// Evaluating a non-existent node ID returns an error.
// ---------------------------------------------------------------------------

func TestMonitoringEngine_NodeNotFound(t *testing.T) {
	resolver := &mockDataResolver{
		resolvedData: &models.ResolvedData{
			Fields:      map[string]float64{},
			Sufficiency: models.DataSufficient,
		},
	}

	engine := buildEngine(nil, resolver)
	_, err := engine.Evaluate(context.Background(), "PM-99", "patient-009", "T2D")
	if err == nil {
		t.Fatal("expected error for unknown node ID, got nil")
	}
}
