package channel_c

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestRules(t *testing.T) *ProtocolGuard {
	t.Helper()
	// Find protocol_rules.yaml relative to test location
	rulesPath := filepath.Join("..", "protocol_rules.yaml")
	if _, err := os.Stat(rulesPath); os.IsNotExist(err) {
		t.Skip("protocol_rules.yaml not found at", rulesPath)
	}
	guard, err := LoadRules(rulesPath)
	if err != nil {
		t.Fatalf("failed to load rules: %v", err)
	}
	return guard
}

func TestPG01_MetforminContraindication(t *testing.T) {
	guard := setupTestRules(t)

	// eGFR 29 + Metformin → HALT
	result := guard.Evaluate(&TitrationContext{
		EGFR:              29,
		ActiveMedications: []string{"METFORMIN"},
	})
	if result.Gate != ProtoHalt || result.RuleID != "PG-01" {
		t.Errorf("eGFR=29 + Metformin should HALT (PG-01), got gate=%s rule=%s", result.Gate, result.RuleID)
	}

	// eGFR 31 + Metformin → CLEAR
	result = guard.Evaluate(&TitrationContext{
		EGFR:              31,
		ActiveMedications: []string{"METFORMIN"},
	})
	if result.Gate != ProtoClear {
		t.Errorf("eGFR=31 + Metformin should CLEAR, got gate=%s rule=%s", result.Gate, result.RuleID)
	}

	// eGFR 25 but NO Metformin → CLEAR (rule requires medication_active)
	result = guard.Evaluate(&TitrationContext{
		EGFR:              25,
		ActiveMedications: []string{"SGLT2I"},
	})
	if result.RuleID == "PG-01" {
		t.Error("PG-01 should not fire without Metformin active")
	}
}

func TestPG02_SGLT2iEfficacy(t *testing.T) {
	guard := setupTestRules(t)

	// eGFR 44 + SGLT2I → PAUSE
	result := guard.Evaluate(&TitrationContext{
		EGFR:              44,
		ActiveMedications: []string{"SGLT2I"},
	})
	if result.Gate != ProtoPause || result.RuleID != "PG-02" {
		t.Errorf("eGFR=44 + SGLT2I should PAUSE (PG-02), got gate=%s rule=%s", result.Gate, result.RuleID)
	}

	// eGFR 46 + SGLT2I → CLEAR
	result = guard.Evaluate(&TitrationContext{
		EGFR:              46,
		ActiveMedications: []string{"SGLT2I"},
	})
	if result.Gate != ProtoClear {
		t.Errorf("eGFR=46 + SGLT2I should CLEAR, got %s", result.Gate)
	}
}

func TestPG03_AKIDetected(t *testing.T) {
	guard := setupTestRules(t)

	result := guard.Evaluate(&TitrationContext{
		EGFR:        65,
		AKIDetected: true,
	})
	if result.Gate != ProtoHalt || result.RuleID != "PG-03" {
		t.Errorf("AKI detected should HALT (PG-03), got gate=%s rule=%s", result.Gate, result.RuleID)
	}
}

func TestPG04_InsulinIntoHypo(t *testing.T) {
	guard := setupTestRules(t)

	result := guard.Evaluate(&TitrationContext{
		EGFR:                65,
		ActiveHypoglycaemia: true,
		ProposedAction:      "insulin_increase",
	})
	if result.Gate != ProtoHalt || result.RuleID != "PG-04" {
		t.Errorf("insulin increase into hypo should HALT (PG-04), got gate=%s rule=%s", result.Gate, result.RuleID)
	}

	// Same hypo but dose_decrease → PG-04 should NOT fire
	result = guard.Evaluate(&TitrationContext{
		EGFR:                65,
		ActiveHypoglycaemia: true,
		ProposedAction:      "dose_decrease",
	})
	if result.RuleID == "PG-04" {
		t.Error("PG-04 should not fire for dose_decrease")
	}
}

func TestPG05_MaxDoseDelta(t *testing.T) {
	guard := setupTestRules(t)

	// 21% → PAUSE
	result := guard.Evaluate(&TitrationContext{
		EGFR:             65,
		DoseDeltaPercent: 21,
	})
	if result.Gate != ProtoPause || result.RuleID != "PG-05" {
		t.Errorf("21%% delta should PAUSE (PG-05), got gate=%s rule=%s", result.Gate, result.RuleID)
	}

	// 19% → CLEAR
	result = guard.Evaluate(&TitrationContext{
		EGFR:             65,
		DoseDeltaPercent: 19,
	})
	if result.Gate != ProtoClear {
		t.Errorf("19%% delta should CLEAR, got %s", result.Gate)
	}
}

func TestPG07_PostHypoWindow(t *testing.T) {
	guard := setupTestRules(t)

	result := guard.Evaluate(&TitrationContext{
		EGFR:                  65,
		HypoglycaemiaWithin7d: true,
		ProposedAction:        "dose_increase",
	})
	if result.Gate != ProtoHalt || result.RuleID != "PG-07" {
		t.Errorf("post-hypo dose increase should HALT (PG-07), got gate=%s rule=%s", result.Gate, result.RuleID)
	}
}

func TestPG04_PG07_CombinedInteraction(t *testing.T) {
	guard := setupTestRules(t)

	// Both conditions active: active hypo + hypo within 7d + insulin_increase
	// PG-04 and PG-07 both fire HALT. The most restrictive (HALT) wins,
	// and the first matching HALT rule (PG-04 appears before PG-07) takes attribution.
	result := guard.Evaluate(&TitrationContext{
		EGFR:                  65,
		ActiveHypoglycaemia:   true,
		HypoglycaemiaWithin7d: true,
		ProposedAction:        "insulin_increase",
	})
	if result.Gate != ProtoHalt {
		t.Errorf("combined PG-04+PG-07 should HALT, got %s", result.Gate)
	}

	// Only PG-07 active (no current hypo, but hypo within 7d + dose_increase)
	result = guard.Evaluate(&TitrationContext{
		EGFR:                  65,
		ActiveHypoglycaemia:   false,
		HypoglycaemiaWithin7d: true,
		ProposedAction:        "dose_increase",
	})
	if result.Gate != ProtoHalt || result.RuleID != "PG-07" {
		t.Errorf("only PG-07 should fire, got gate=%s rule=%s", result.Gate, result.RuleID)
	}

	// Only PG-04 active (current hypo + insulin_increase, but no 7d history)
	result = guard.Evaluate(&TitrationContext{
		EGFR:                  65,
		ActiveHypoglycaemia:   true,
		HypoglycaemiaWithin7d: false,
		ProposedAction:        "insulin_increase",
	})
	if result.Gate != ProtoHalt || result.RuleID != "PG-04" {
		t.Errorf("only PG-04 should fire, got gate=%s rule=%s", result.Gate, result.RuleID)
	}
}

func TestRulesHashStable(t *testing.T) {
	guard := setupTestRules(t)
	hash := guard.RulesHash()
	if hash == "" {
		t.Error("rules hash should not be empty")
	}
	if len(hash) != 64 {
		t.Errorf("SHA-256 hash should be 64 hex chars, got %d", len(hash))
	}
}

func TestRuleCount(t *testing.T) {
	guard := setupTestRules(t)
	// PG-01..PG-05, PG-07..PG-17-A3, PG-17-A2, PG-08-DUAL-RAAS, AD-09 (PG-06 excluded) = 19 rules
	if guard.RuleCount() != 19 {
		t.Errorf("expected 19 rules (PG-06 excluded), got %d", guard.RuleCount())
	}
}

// ════════════════════════════════════════════════════════════════════════
// DUAL RAAS CONTRAINDICATION (PG-08-DUAL-RAAS)
// ════════════════════════════════════════════════════════════════════════

func TestDualRAAS_HALT(t *testing.T) {
	guard := setupTestRules(t)

	// DualRAASActive=true → HALT via PG-08-DUAL-RAAS
	result := guard.Evaluate(&TitrationContext{
		EGFR:           60,
		DualRAASActive: true,
	})
	if result.Gate != ProtoHalt || result.RuleID != "PG-08-DUAL-RAAS" {
		t.Errorf("PG-08-DUAL-RAAS should HALT on dual RAAS, got gate=%s rule=%s", result.Gate, result.RuleID)
	}

	// DualRAASActive=false → CLEAR (no fire)
	result = guard.Evaluate(&TitrationContext{
		EGFR:           60,
		DualRAASActive: false,
	})
	if result.RuleID == "PG-08-DUAL-RAAS" {
		t.Error("PG-08-DUAL-RAAS should not fire when DualRAASActive is false")
	}
}

// ════════════════════════════════════════════════════════════════════════
// HTN CO-MANAGEMENT RULES (PG-08 through PG-14)
// ════════════════════════════════════════════════════════════════════════

func TestPG08_ACEiARBHyperKDecliningEGFR(t *testing.T) {
	guard := setupTestRules(t)

	// Composite true + dose_increase → HALT
	result := guard.Evaluate(&TitrationContext{
		EGFR:                       40,
		ACEiARBHyperKDecliningEGFR: true,
		ProposedAction:             "dose_increase",
	})
	if result.Gate != ProtoHalt || result.RuleID != "PG-08" {
		t.Errorf("PG-08 should HALT on ACEi/ARB+hyperK+declining eGFR uptitration, got gate=%s rule=%s", result.Gate, result.RuleID)
	}

	// Composite true but dose_decrease → should NOT fire (action_type filter)
	result = guard.Evaluate(&TitrationContext{
		EGFR:                       40,
		ACEiARBHyperKDecliningEGFR: true,
		ProposedAction:             "dose_decrease",
	})
	if result.RuleID == "PG-08" {
		t.Error("PG-08 should not fire for dose_decrease")
	}

	// Composite false → CLEAR
	result = guard.Evaluate(&TitrationContext{
		EGFR:                       40,
		ACEiARBHyperKDecliningEGFR: false,
		ProposedAction:             "dose_increase",
	})
	if result.RuleID == "PG-08" {
		t.Error("PG-08 should not fire when composite is false")
	}
}

func TestPG09_BetaBlockerInsulin(t *testing.T) {
	guard := setupTestRules(t)

	result := guard.Evaluate(&TitrationContext{
		EGFR:                     65,
		BetaBlockerInsulinActive: true,
	})
	if result.Gate != ProtoModify || result.RuleID != "PG-09" {
		t.Errorf("PG-09 should MODIFY on beta-blocker+insulin, got gate=%s rule=%s", result.Gate, result.RuleID)
	}
}

func TestPG10_ResistantHTN(t *testing.T) {
	guard := setupTestRules(t)

	result := guard.Evaluate(&TitrationContext{
		EGFR:                 65,
		ResistantHTNDetected: true,
	})
	if result.Gate != ProtoPause || result.RuleID != "PG-10" {
		t.Errorf("PG-10 should PAUSE on resistant HTN, got gate=%s rule=%s", result.Gate, result.RuleID)
	}
}

func TestPG11_ThiazideHyponatraemia(t *testing.T) {
	guard := setupTestRules(t)

	result := guard.Evaluate(&TitrationContext{
		EGFR:                  65,
		ThiazideHyponatraemia: true,
	})
	if result.Gate != ProtoHalt || result.RuleID != "PG-11" {
		t.Errorf("PG-11 should HALT on thiazide+hyponatraemia, got gate=%s rule=%s", result.Gate, result.RuleID)
	}
}

func TestPG12_MRAHyperKLowEGFR(t *testing.T) {
	guard := setupTestRules(t)

	result := guard.Evaluate(&TitrationContext{
		EGFR:             40,
		MRAHyperKLowEGFR: true,
	})
	if result.Gate != ProtoModify || result.RuleID != "PG-12" {
		t.Errorf("PG-12 should MODIFY on MRA+hyperK+low eGFR, got gate=%s rule=%s", result.Gate, result.RuleID)
	}
}

func TestPG13_CCBExcessiveResponse(t *testing.T) {
	guard := setupTestRules(t)

	result := guard.Evaluate(&TitrationContext{
		EGFR:                 65,
		CCBExcessiveResponse: true,
	})
	if result.Gate != ProtoModify || result.RuleID != "PG-13" {
		t.Errorf("PG-13 should MODIFY on CCB excessive response, got gate=%s rule=%s", result.Gate, result.RuleID)
	}
}

func TestPG14_RAASCreatinineTolerant(t *testing.T) {
	guard := setupTestRules(t)

	// PG-14 fires as CLEAR — it's an audit marker, not a gate changer
	result := guard.Evaluate(&TitrationContext{
		EGFR:                   65,
		RAASCreatinineTolerant: true,
	})
	// PG-14 gate is CLEAR, so it should not change the overall result
	if result.Gate != ProtoClear {
		t.Errorf("PG-14 (CLEAR gate) should not escalate, got gate=%s", result.Gate)
	}
}

func TestPG_GateSeverityOrdering(t *testing.T) {
	guard := setupTestRules(t)

	// PG-11 (HALT) + PG-09 (MODIFY) simultaneously → HALT wins
	result := guard.Evaluate(&TitrationContext{
		EGFR:                     65,
		ThiazideHyponatraemia:    true,
		BetaBlockerInsulinActive: true,
	})
	if result.Gate != ProtoHalt {
		t.Errorf("HALT (PG-11) should override MODIFY (PG-09), got gate=%s rule=%s", result.Gate, result.RuleID)
	}

	// PG-10 (PAUSE) + PG-09 (MODIFY) simultaneously → PAUSE wins
	result = guard.Evaluate(&TitrationContext{
		EGFR:                     65,
		ResistantHTNDetected:     true,
		BetaBlockerInsulinActive: true,
	})
	if result.Gate != ProtoPause {
		t.Errorf("PAUSE (PG-10) should override MODIFY (PG-09), got gate=%s rule=%s", result.Gate, result.RuleID)
	}
}

// ════════════════════════════════════════════════════════════════════════
// HTN CO-MANAGEMENT RULES Wave 2 (PG-15, PG-16)
// ════════════════════════════════════════════════════════════════════════

func TestPG15_ACEiInducedCough(t *testing.T) {
	guard := setupTestRules(t)

	// Posterior > 0.70 → MODIFY (ARB switch recommended)
	result := guard.Evaluate(&TitrationContext{
		EGFR:                        60,
		ACEiInducedCoughProbability: 0.75,
	})
	if result.Gate != ProtoModify || result.RuleID != "PG-15" {
		t.Errorf("PG-15 should MODIFY on cough probability 0.75, got gate=%s rule=%s", result.Gate, result.RuleID)
	}

	// Posterior = 0.85 → still MODIFY (not HALT or PAUSE)
	result = guard.Evaluate(&TitrationContext{
		EGFR:                        60,
		ACEiInducedCoughProbability: 0.85,
	})
	if result.Gate != ProtoModify || result.RuleID != "PG-15" {
		t.Errorf("PG-15 should MODIFY on cough probability 0.85, got gate=%s rule=%s", result.Gate, result.RuleID)
	}

	// Posterior = 0.65 (below threshold) → CLEAR
	result = guard.Evaluate(&TitrationContext{
		EGFR:                        60,
		ACEiInducedCoughProbability: 0.65,
	})
	if result.RuleID == "PG-15" {
		t.Error("PG-15 should not fire when cough probability is below 0.70")
	}

	// Posterior = 0 (no cough investigation) → CLEAR
	result = guard.Evaluate(&TitrationContext{
		EGFR: 60,
	})
	if result.RuleID == "PG-15" {
		t.Error("PG-15 should not fire when cough probability is zero")
	}
}

func TestPG16_AFNoAnticoagulation(t *testing.T) {
	guard := setupTestRules(t)

	// AF confirmed + no anticoagulation → PAUSE
	result := guard.Evaluate(&TitrationContext{
		EGFR:                         60,
		AFConfirmedNoAnticoagulation: true,
	})
	if result.Gate != ProtoPause || result.RuleID != "PG-16" {
		t.Errorf("PG-16 should PAUSE on AF without anticoagulation, got gate=%s rule=%s", result.Gate, result.RuleID)
	}

	// AF not confirmed → CLEAR
	result = guard.Evaluate(&TitrationContext{
		EGFR:                         60,
		AFConfirmedNoAnticoagulation: false,
	})
	if result.RuleID == "PG-16" {
		t.Error("PG-16 should not fire when AF is not confirmed or anticoagulation is present")
	}
}

func TestPG15_PG16_Combined(t *testing.T) {
	guard := setupTestRules(t)

	// Both PG-15 (MODIFY) and PG-16 (PAUSE) fire → most restrictive = PAUSE (PG-16)
	result := guard.Evaluate(&TitrationContext{
		EGFR:                         60,
		ACEiInducedCoughProbability:  0.80,
		AFConfirmedNoAnticoagulation: true,
	})
	if result.Gate != ProtoPause {
		t.Errorf("Combined PG-15+PG-16 should resolve to PAUSE (most restrictive), got gate=%s", result.Gate)
	}
	if result.RuleID != "PG-16" {
		t.Errorf("Most restrictive rule should be PG-16, got %s", result.RuleID)
	}
}

// ════════════════════════════════════════════════════════════════════════
// DEPRESCRIBING SAFETY RULE (AD-09)
// ════════════════════════════════════════════════════════════════════════

// ════════════════════════════════════════════════════════════════════════
// ACR-BASED RAAS ESCALATION (PG-17)
// ════════════════════════════════════════════════════════════════════════

func TestPG17_ACRA3NoRAAS_HALT(t *testing.T) {
	rulesYAML := `
version: "test"
rules:
  - rule_id: PG-17-A3
    description: "ACR A3 without RAAS blockade"
    guideline_ref: KDIGO-2024-CKD
    condition:
      field: acr_a3_no_raas
      operator: eq
      value: true
    gate: HALT
  - rule_id: PG-17-A2
    description: "ACR A2 without RAAS blockade"
    guideline_ref: KDIGO-2024-CKD
    condition:
      field: acr_a2_no_raas
      operator: eq
      value: true
    gate: MODIFY
`
	tmpFile, _ := os.CreateTemp("", "protocol_rules_*.yaml")
	tmpFile.WriteString(rulesYAML)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	guard, err := LoadRules(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadRules: %v", err)
	}

	tests := []struct {
		name     string
		ctx      *TitrationContext
		wantGate ProtocolGate
		wantRule string
	}{
		{"A3 no RAAS → HALT", &TitrationContext{ACRA3NoRAAS: true}, ProtoHalt, "PG-17-A3"},
		{"A2 no RAAS → MODIFY", &TitrationContext{ACRA2NoRAAS: true}, ProtoModify, "PG-17-A2"},
		{"A3 on RAAS → CLEAR", &TitrationContext{ACRA3NoRAAS: false}, ProtoClear, ""},
		{"A1 → CLEAR", &TitrationContext{}, ProtoClear, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := guard.Evaluate(tt.ctx)
			if result.Gate != tt.wantGate {
				t.Errorf("PG-17: gate = %v, want %v", result.Gate, tt.wantGate)
			}
			if tt.wantRule != "" && result.RuleID != tt.wantRule {
				t.Errorf("RuleID = %v, want %v", result.RuleID, tt.wantRule)
			}
		})
	}
}

func TestAD09_CKDStage4DeprescribingBlock(t *testing.T) {
	guard := setupTestRules(t)

	// CKD Stage 4 deprescribing blocked → HALT
	result := guard.Evaluate(&TitrationContext{
		EGFR:                          22,
		CKDStage4DeprescribingBlocked: true,
	})
	if result.Gate != ProtoHalt || result.RuleID != "AD-09" {
		t.Errorf("AD-09 should HALT on CKD Stage 4 deprescribing, got gate=%s rule=%s", result.Gate, result.RuleID)
	}

	// CKD Stage 3a (not blocked) → CLEAR
	result = guard.Evaluate(&TitrationContext{
		EGFR:                          55,
		CKDStage4DeprescribingBlocked: false,
	})
	if result.RuleID == "AD-09" {
		t.Error("AD-09 should not fire when deprescribing is not blocked")
	}
}
