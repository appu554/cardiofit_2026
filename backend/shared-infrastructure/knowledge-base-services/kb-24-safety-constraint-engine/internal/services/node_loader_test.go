package services

import (
	"testing"

	"go.uber.org/zap"
)

func TestLoadDrugClassNodes(t *testing.T) {
	log := zap.NewNop()
	loader := NewNodeLoader("../../nodes", log)
	if err := loader.Load(); err != nil {
		t.Fatalf("failed to load nodes: %v", err)
	}
	if loader.Count() < 9 {
		t.Errorf("expected at least 9 drug class nodes, got %d", loader.Count())
	}

	// Verify single-trigger node
	met := loader.Get("DC-METFORMIN-01")
	if met == nil {
		t.Fatal("DC-METFORMIN-01 not found")
	}
	if len(met.SafetyTriggers) != 1 {
		t.Errorf("DC-METFORMIN-01: want 1 trigger, got %d", len(met.SafetyTriggers))
	}
	if met.SafetyTriggers[0].ID != "ST-MET-LACTIC" {
		t.Errorf("DC-METFORMIN-01 trigger ID: want ST-MET-LACTIC, got %s", met.SafetyTriggers[0].ID)
	}

	// Verify multi-trigger node (SGLT2i has 2)
	sglt2 := loader.Get("DC-SGLT2I-01")
	if sglt2 == nil {
		t.Fatal("DC-SGLT2I-01 not found")
	}
	if len(sglt2.SafetyTriggers) != 2 {
		t.Errorf("DC-SGLT2I-01: want 2 triggers, got %d", len(sglt2.SafetyTriggers))
	}

	// Verify multi-trigger node (finerenone has 2)
	fin := loader.Get("DC-FINERENONE-01")
	if fin == nil {
		t.Fatal("DC-FINERENONE-01 not found")
	}
	if len(fin.SafetyTriggers) != 2 {
		t.Errorf("DC-FINERENONE-01: want 2 triggers, got %d", len(fin.SafetyTriggers))
	}

	// Verify all 9 expected node IDs are present
	expectedNodes := []string{
		"DC-METFORMIN-01",
		"DC-SGLT2I-01",
		"DC-INSULIN-01",
		"DC-SU-01",
		"DC-ACEI-01",
		"DC-ARB-01",
		"DC-FINERENONE-01",
		"DC-THIAZIDE-01",
		"DC-BB-01",
	}
	for _, id := range expectedNodes {
		if loader.Get(id) == nil {
			t.Errorf("expected node %s not found", id)
		}
	}
}
