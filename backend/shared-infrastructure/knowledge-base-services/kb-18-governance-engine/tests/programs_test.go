// Package tests provides comprehensive testing for KB-18 Governance Engine
package tests

import (
	"testing"

	"kb-18-governance-engine/pkg/programs"
)

// TestProgramStore_Initialization tests that program store initializes correctly
func TestProgramStore_Initialization(t *testing.T) {
	store := programs.NewProgramStore()

	if store.Count() == 0 {
		t.Errorf("Expected programs to be loaded, got 0")
	}

	t.Logf("Program store initialized with %d programs", store.Count())
}

// TestProgramStore_GetMaternalMedication tests MATERNAL_MEDICATION program exists
func TestProgramStore_GetMaternalMedication(t *testing.T) {
	store := programs.NewProgramStore()

	program := store.Get("MATERNAL_MEDICATION")
	if program == nil {
		t.Fatalf("Expected MATERNAL_MEDICATION program to exist")
	}

	if program.Name == "" {
		t.Errorf("Expected program to have a name")
	}

	if len(program.Rules) == 0 {
		t.Errorf("Expected MATERNAL_MEDICATION to have rules")
	}

	t.Logf("MATERNAL_MEDICATION program has %d rules", len(program.Rules))
}

// TestProgramStore_GetOpioidStewardship tests OPIOID_STEWARDSHIP program exists
func TestProgramStore_GetOpioidStewardship(t *testing.T) {
	store := programs.NewProgramStore()

	program := store.Get("OPIOID_STEWARDSHIP")
	if program == nil {
		t.Fatalf("Expected OPIOID_STEWARDSHIP program to exist")
	}

	if len(program.Rules) == 0 {
		t.Errorf("Expected OPIOID_STEWARDSHIP to have rules")
	}

	// Check accountability chain exists
	if len(program.AccountabilityChain) == 0 {
		t.Errorf("Expected program to have accountability chain")
	}

	t.Logf("OPIOID_STEWARDSHIP program has %d rules, accountability chain: %v",
		len(program.Rules), program.AccountabilityChain)
}

// TestProgramStore_GetAnticoagulation tests ANTICOAGULATION program exists
func TestProgramStore_GetAnticoagulation(t *testing.T) {
	store := programs.NewProgramStore()

	program := store.Get("ANTICOAGULATION")
	if program == nil {
		t.Fatalf("Expected ANTICOAGULATION program to exist")
	}

	if program.Category != "ANTICOAGULATION" {
		t.Errorf("Expected category ANTICOAGULATION, got: %s", program.Category)
	}
}

// TestProgramStore_GetNonExistent tests retrieval of non-existent program
func TestProgramStore_GetNonExistent(t *testing.T) {
	store := programs.NewProgramStore()

	program := store.Get("NON_EXISTENT_PROGRAM")
	if program != nil {
		t.Errorf("Expected nil for non-existent program, got: %+v", program)
	}
}

// TestProgramStore_GetAll tests retrieval of all programs
func TestProgramStore_GetAll(t *testing.T) {
	store := programs.NewProgramStore()

	allPrograms := store.GetAll()
	if len(allPrograms) == 0 {
		t.Errorf("Expected programs, got empty map")
	}

	// Check expected programs exist
	expectedPrograms := []string{
		"MATERNAL_MEDICATION",
		"PREECLAMPSIA_PROTOCOL",
		"MAGNESIUM_PROTOCOL",
		"GESTATIONAL_DM",
		"OPIOID_STEWARDSHIP",
		"OPIOID_NAIVE",
		"OPIOID_MAT",
		"ANTICOAGULATION",
		"WARFARIN_MANAGEMENT",
		"DOAC_MANAGEMENT",
	}

	for _, code := range expectedPrograms {
		if _, exists := allPrograms[code]; !exists {
			t.Errorf("Expected program %s to exist", code)
		}
	}

	t.Logf("Found %d programs in store", len(allPrograms))
}

// TestProgram_ActivationCriteria tests program activation criteria
func TestProgram_ActivationCriteria(t *testing.T) {
	store := programs.NewProgramStore()

	program := store.Get("MATERNAL_MEDICATION")
	if program == nil {
		t.Fatalf("Expected MATERNAL_MEDICATION program to exist")
	}

	criteria := program.ActivationCriteria

	// MATERNAL_MEDICATION should activate for pregnancy
	if !criteria.RequiresPregnancy {
		t.Errorf("Expected MATERNAL_MEDICATION to require pregnancy")
	}

	t.Logf("Activation criteria: registry=%v, diagnoses=%v, medications=%v, pregnancy=%v",
		criteria.RegistryCodes, criteria.DiagnosisCodes, criteria.MedicationCodes, criteria.RequiresPregnancy)
}

// TestProgram_RuleStructure tests that rules have required fields
func TestProgram_RuleStructure(t *testing.T) {
	store := programs.NewProgramStore()

	for code, program := range store.GetAll() {
		for i, rule := range program.Rules {
			// Get rule code using the helper method (uses ID as primary, Code as fallback)
			ruleCode := rule.GetCode()

			// Check required fields
			if ruleCode == "" {
				t.Errorf("Program %s rule %d: missing code/ID", code, i)
			}
			if rule.Name == "" {
				t.Errorf("Program %s rule %d (%s): missing name", code, i, ruleCode)
			}
			if rule.Severity == "" {
				t.Errorf("Program %s rule %d (%s): missing severity", code, i, ruleCode)
			}
			if rule.EnforcementLevel == "" {
				t.Errorf("Program %s rule %d (%s): missing enforcement level", code, i, ruleCode)
			}
			if len(rule.Conditions) == 0 {
				t.Errorf("Program %s rule %d (%s): no conditions", code, i, ruleCode)
			}

			// Check condition structure
			for j, cond := range rule.Conditions {
				if cond.Type == "" {
					t.Errorf("Program %s rule %s condition %d: missing type", code, ruleCode, j)
				}
			}
		}
	}
}

// TestProgram_AccountabilityChain tests that programs have accountability chains
func TestProgram_AccountabilityChain(t *testing.T) {
	store := programs.NewProgramStore()

	// All clinical programs should have accountability chains
	for code, program := range store.GetAll() {
		if len(program.AccountabilityChain) == 0 {
			t.Errorf("Program %s: missing accountability chain", code)
		} else {
			t.Logf("Program %s: accountability chain = %v", code, program.AccountabilityChain)
		}
	}
}

// TestProgram_EnforcementLevelsValid tests that rules use valid enforcement levels
func TestProgram_EnforcementLevelsValid(t *testing.T) {
	store := programs.NewProgramStore()

	validLevels := map[string]bool{
		"IGNORE":                  true,
		"NOTIFY":                  true,
		"WARN_ACKNOWLEDGE":        true,
		"HARD_BLOCK":              true,
		"HARD_BLOCK_WITH_OVERRIDE": true,
		"MANDATORY_ESCALATION":    true,
	}

	for code, program := range store.GetAll() {
		for _, rule := range program.Rules {
			if !validLevels[string(rule.EnforcementLevel)] {
				t.Errorf("Program %s rule %s: invalid enforcement level '%s'",
					code, rule.GetCode(), rule.EnforcementLevel)
			}
		}
	}
}

// TestProgram_SeverityLevelsValid tests that rules use valid severity levels
func TestProgram_SeverityLevelsValid(t *testing.T) {
	store := programs.NewProgramStore()

	validSeverities := map[string]bool{
		"INFO":     true,
		"LOW":      true,
		"MODERATE": true,
		"HIGH":     true,
		"CRITICAL": true,
		"FATAL":    true,
	}

	for code, program := range store.GetAll() {
		for _, rule := range program.Rules {
			if !validSeverities[string(rule.Severity)] {
				t.Errorf("Program %s rule %s: invalid severity '%s'",
					code, rule.GetCode(), rule.Severity)
			}
		}
	}
}
