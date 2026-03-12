package services

import (
	"testing"

	"kb-22-hpi-engine/internal/models"
)

// ---------------------------------------------------------------------------
// T5: SessionService pure-function unit tests
// ---------------------------------------------------------------------------

func TestTopN(t *testing.T) {
	entries := []models.DifferentialEntry{
		{DifferentialID: "ACS", PosteriorProbability: 0.50},
		{DifferentialID: "PE", PosteriorProbability: 0.25},
		{DifferentialID: "CHF", PosteriorProbability: 0.15},
		{DifferentialID: "GERD", PosteriorProbability: 0.05},
		{DifferentialID: "MSK", PosteriorProbability: 0.03},
		{DifferentialID: "ANXIETY", PosteriorProbability: 0.02},
	}

	tests := []struct {
		name string
		n    int
		want int
	}{
		{"top 5 from 6", 5, 5},
		{"top 3 from 6", 3, 3},
		{"top 10 from 6 (cap)", 10, 6},
		{"top 0 from 6", 0, 0},
		{"top 1 from 6", 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := topN(entries, tt.n)
			if len(got) != tt.want {
				t.Errorf("topN(entries, %d) returned %d, want %d", tt.n, len(got), tt.want)
			}
		})
	}
}

func TestTopN_Empty(t *testing.T) {
	got := topN(nil, 5)
	if len(got) != 0 {
		t.Errorf("topN(nil, 5) = %d entries, want 0", len(got))
	}

	got = topN([]models.DifferentialEntry{}, 5)
	if len(got) != 0 {
		t.Errorf("topN([], 5) = %d entries, want 0", len(got))
	}
}


func TestNewSessionService_Constructor(t *testing.T) {
	log := testLogger()
	m := testMetrics()

	// Verify constructor accepts all 17 params and returns non-nil
	svc := NewSessionService(
		nil, nil, log, m,
		nil, nil, nil,
		nil, nil,
		nil, nil,
		nil, nil, nil,
		nil,
		nil, nil,
	)
	if svc == nil {
		t.Fatal("NewSessionService returned nil")
	}
}

func TestComputeLRApplied(t *testing.T) {
	log := testLogger()
	m := testMetrics()
	svc := NewSessionService(
		nil, nil, log, m,
		nil, nil, nil,
		nil, nil,
		nil, nil,
		nil, nil, nil,
		nil,
		nil, nil,
	)

	question := &models.QuestionDef{
		ID:         "Q001",
		LRPositive: map[string]float64{"ACS": 3.5, "PE": 1.2},
		LRNegative: map[string]float64{"ACS": 0.3, "PE": 0.8},
	}

	t.Run("YES returns lr_positive", func(t *testing.T) {
		lr := svc.computeLRApplied(question, "YES")
		if lr["ACS"] != 3.5 {
			t.Errorf("ACS LR = %f, want 3.5", lr["ACS"])
		}
		if lr["PE"] != 1.2 {
			t.Errorf("PE LR = %f, want 1.2", lr["PE"])
		}
	})

	t.Run("NO returns lr_negative", func(t *testing.T) {
		lr := svc.computeLRApplied(question, "NO")
		if lr["ACS"] != 0.3 {
			t.Errorf("ACS LR = %f, want 0.3", lr["ACS"])
		}
	})

	t.Run("PATA_NAHI returns zero LR", func(t *testing.T) {
		lr := svc.computeLRApplied(question, "PATA_NAHI")
		for diffID, val := range lr {
			if val != 0.0 {
				t.Errorf("PATA_NAHI: %s LR = %f, want 0.0", diffID, val)
			}
		}
		if len(lr) != 2 {
			t.Errorf("PATA_NAHI: got %d entries, want 2 (one per differential)", len(lr))
		}
	})
}


func TestNewNodeLoaderFromMap(t *testing.T) {
	nodes := map[string]*models.NodeDefinition{
		"P1_CHEST_PAIN": {
			NodeID:  "P1_CHEST_PAIN",
			Version: "1.0.0",
		},
		"P2_DYSPNEA": {
			NodeID:  "P2_DYSPNEA",
			Version: "1.0.0",
		},
	}

	loader := NewNodeLoaderFromMap(nodes)
	if loader == nil {
		t.Fatal("NewNodeLoaderFromMap returned nil")
	}

	if got := loader.Get("P1_CHEST_PAIN"); got == nil {
		t.Error("expected P1_CHEST_PAIN, got nil")
	}
	if got := loader.Get("P99_NONEXISTENT"); got != nil {
		t.Error("expected nil for non-existent node")
	}
	if ids := loader.List(); len(ids) != 2 {
		t.Errorf("List() returned %d nodes, want 2", len(ids))
	}
	if all := loader.All(); len(all) != 2 {
		t.Errorf("All() returned %d nodes, want 2", len(all))
	}
}
