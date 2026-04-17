package services

import (
	"testing"

	"kb-26-metabolic-digital-twin/internal/models"
)

func TestContext_CKM4c_HFrEF_NYHA3_Max(t *testing.T) {
	cfg := testPAIConfig()
	input := models.PAIDimensionInput{
		CKMStage:           "4c",
		HFType:             "HFrEF",
		NYHAClass:          "III",
		IsPostDischarge30d: true,
		Age:                78,
		MedicationCount:    7,
	}

	score := ComputeContextScore(input, cfg)

	// base 65 + 25 post-discharge + 15 polypharmacy = 105
	// × 1.3 NYHA III on base first: 65*1.3 = 84.5 + 25 + 15 = 124.5 → capped 100
	// Either way, should be capped at 100
	if score < 95 {
		t.Errorf("expected score near 100 for max context, got %.2f", score)
	}
	if score > 100 {
		t.Errorf("expected score capped at 100, got %.2f", score)
	}
}

func TestContext_CKM2_Healthy_Low(t *testing.T) {
	cfg := testPAIConfig()
	input := models.PAIDimensionInput{
		CKMStage: "2",
	}

	score := ComputeContextScore(input, cfg)

	// CKM stage 2 base = 10, no modifiers → 10
	if score < 8 || score > 12 {
		t.Errorf("expected score ~10 for CKM2 no modifiers, got %.2f", score)
	}
}

func TestContext_PostDischargeBonus(t *testing.T) {
	cfg := testPAIConfig()
	input := models.PAIDimensionInput{
		CKMStage:           "3",
		IsPostDischarge30d: true,
	}

	score := ComputeContextScore(input, cfg)

	// base 20 + 25 post-discharge = 45
	if score < 40 {
		t.Errorf("expected score ≥40 with post-discharge bonus, got %.2f", score)
	}
}

func TestContext_PolypharmacyElderly(t *testing.T) {
	cfg := testPAIConfig()
	input := models.PAIDimensionInput{
		CKMStage:        "2",
		Age:             78,
		MedicationCount: 7,
	}

	score := ComputeContextScore(input, cfg)

	// base 10 + 15 polypharmacy = 25
	if score < 20 {
		t.Errorf("expected score ≥20 with polypharmacy elderly, got %.2f", score)
	}
}

func TestContext_NoModifiers_CKM1(t *testing.T) {
	cfg := testPAIConfig()
	input := models.PAIDimensionInput{
		CKMStage: "1",
	}

	score := ComputeContextScore(input, cfg)

	// CKM stage 1 base = 5, no modifiers → 5
	if score < 3 || score > 7 {
		t.Errorf("expected score ~5 for CKM1 no modifiers, got %.2f", score)
	}
}
