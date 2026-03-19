package calibration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/models"
	"kb-22-hpi-engine/internal/services"
)

// ---------------------------------------------------------------------------
// PM golden dataset test vector
// ---------------------------------------------------------------------------

type pmTestVector struct {
	TestID                 string             `json:"test_id"`
	NodeID                 string             `json:"node_id"`
	Inputs                 map[string]float64 `json:"inputs"`
	ExpectedClassification string             `json:"expected_classification"`
	ExpectedSeverity       string             `json:"expected_severity"`
	ExpectedGate           string             `json:"expected_gate"`
}

// ---------------------------------------------------------------------------
// mockResolver returns pre-set fields for golden dataset testing.
// ---------------------------------------------------------------------------

type mockResolver struct {
	fields map[string]float64
}

func (m *mockResolver) Resolve(
	_ context.Context,
	_ string,
	_ []models.RequiredInput,
	_ []models.AggregatedInputDef,
) (*models.ResolvedData, error) {
	return &models.ResolvedData{
		Fields:      m.fields,
		Sufficiency: models.DataSufficient,
	}, nil
}

// ---------------------------------------------------------------------------
// TestPMGoldenDataset loads calibration vectors and validates PM engine output.
// ---------------------------------------------------------------------------

func TestPMGoldenDataset(t *testing.T) {
	// Resolve paths relative to this test file
	_, thisFile, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	pmNodesDir := filepath.Join(baseDir, "pm-nodes")
	datasetPath := filepath.Join(baseDir, "calibration", "pm_golden_dataset.json")

	log := zap.NewNop()

	// Load real PM node YAML definitions
	loader := services.NewMonitoringNodeLoader(pmNodesDir, log)
	if err := loader.Load(); err != nil {
		t.Fatalf("failed to load PM nodes from %s: %v", pmNodesDir, err)
	}

	// Load golden dataset
	raw, err := os.ReadFile(datasetPath)
	if err != nil {
		t.Fatalf("failed to read golden dataset: %v", err)
	}
	var vectors []pmTestVector
	if err := json.Unmarshal(raw, &vectors); err != nil {
		t.Fatalf("failed to parse golden dataset: %v", err)
	}

	evaluator := services.NewExpressionEvaluator()
	trajectory := services.NewTrajectoryComputer(log)

	for _, v := range vectors {
		t.Run(v.TestID, func(t *testing.T) {
			resolver := &mockResolver{fields: v.Inputs}
			engine := services.NewMonitoringNodeEngine(loader, resolver, evaluator, trajectory, nil, log)

			event, err := engine.Evaluate(context.Background(), v.NodeID, "golden-patient", "CKD_G3a_DM")
			if err != nil {
				t.Fatalf("Evaluate(%s) error: %v", v.NodeID, err)
			}
			if event == nil {
				t.Fatalf("Evaluate(%s) returned nil event", v.NodeID)
			}
			if event.Classification == nil {
				t.Fatalf("Evaluate(%s) returned nil classification", v.NodeID)
			}

			if event.Classification.Category != v.ExpectedClassification {
				t.Errorf("classification: got %q, want %q",
					event.Classification.Category, v.ExpectedClassification)
			}

			// Check gate suggestion
			gate := ""
			if event.MCUGateSuggestion != nil {
				gate = *event.MCUGateSuggestion
			}
			if gate != v.ExpectedGate {
				t.Errorf("gate: got %q, want %q", gate, v.ExpectedGate)
			}
		})
	}
}
