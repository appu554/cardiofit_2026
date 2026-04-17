package services

import (
	"testing"

	"kb-26-metabolic-digital-twin/internal/models"
)

func TestTrigger_RateLimitBlocks(t *testing.T) {
	trigger := NewPAIEventTrigger(15, 10.0) // 15-minute interval

	// Mark P001 as just computed
	trigger.MarkComputed("P001")

	// P001 should be blocked (within 15 min window)
	if trigger.ShouldRecompute("P001") {
		t.Fatal("expected ShouldRecompute=false for P001 within rate-limit window")
	}

	// P002 was never computed — should be allowed
	if !trigger.ShouldRecompute("P002") {
		t.Fatal("expected ShouldRecompute=true for P002 (never computed)")
	}
}

func TestTrigger_SignificantChange_Publishes(t *testing.T) {
	trigger := NewPAIEventTrigger(15, 10.0) // significantDelta = 10

	current := models.PAIScore{
		PatientID:          "P001",
		Score:              55.0,
		Tier:               "HIGH",
		PrimaryReason:      "eGFR declining",
		SuggestedAction:    "Urgent nephrology review",
		SuggestedTimeframe: "48h",
	}
	previous := models.PAIScore{
		PatientID: "P001",
		Score:     40.0,
		Tier:      "MODERATE",
	}

	evt := trigger.ProcessResult(current, previous)
	if evt == nil {
		t.Fatal("expected non-nil PAIChangeEvent for delta=15 > threshold=10")
	}
	if evt.NewScore != 55.0 {
		t.Errorf("NewScore = %v, want 55", evt.NewScore)
	}
	if evt.PreviousScore != 40.0 {
		t.Errorf("PreviousScore = %v, want 40", evt.PreviousScore)
	}
	if evt.NewTier != "HIGH" {
		t.Errorf("NewTier = %q, want HIGH", evt.NewTier)
	}
	if evt.PreviousTier != "MODERATE" {
		t.Errorf("PreviousTier = %q, want MODERATE", evt.PreviousTier)
	}
	if evt.DominantReason != "eGFR declining" {
		t.Errorf("DominantReason = %q, want 'eGFR declining'", evt.DominantReason)
	}
	if evt.SuggestedAction != "Urgent nephrology review" {
		t.Errorf("SuggestedAction = %q, want 'Urgent nephrology review'", evt.SuggestedAction)
	}
	if evt.Timeframe != "48h" {
		t.Errorf("Timeframe = %q, want '48h'", evt.Timeframe)
	}
}

func TestTrigger_TierChange_Publishes(t *testing.T) {
	trigger := NewPAIEventTrigger(15, 10.0) // significantDelta = 10

	current := models.PAIScore{
		PatientID: "P001",
		Score:     62.0,
		Tier:      "HIGH",
	}
	previous := models.PAIScore{
		PatientID: "P001",
		Score:     58.0,
		Tier:      "MODERATE",
	}

	// delta = 4 < 10, but tier changed MODERATE → HIGH
	evt := trigger.ProcessResult(current, previous)
	if evt == nil {
		t.Fatal("expected non-nil PAIChangeEvent when tier changes despite small delta")
	}
	if evt.NewTier != "HIGH" || evt.PreviousTier != "MODERATE" {
		t.Errorf("tier transition = %s→%s, want MODERATE→HIGH", evt.PreviousTier, evt.NewTier)
	}
}

func TestTrigger_NoChange_Suppresses(t *testing.T) {
	trigger := NewPAIEventTrigger(15, 10.0) // significantDelta = 10

	current := models.PAIScore{
		PatientID: "P001",
		Score:     42.0,
		Tier:      "MODERATE",
	}
	previous := models.PAIScore{
		PatientID: "P001",
		Score:     38.0,
		Tier:      "MODERATE",
	}

	// delta = 4 < 10, same tier → suppress
	evt := trigger.ProcessResult(current, previous)
	if evt != nil {
		t.Fatal("expected nil PAIChangeEvent for small delta with same tier")
	}
}
