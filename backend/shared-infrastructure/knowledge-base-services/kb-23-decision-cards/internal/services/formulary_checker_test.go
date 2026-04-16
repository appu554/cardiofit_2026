package services

import (
	"testing"

	"go.uber.org/zap"
)

func TestFormularyChecker_LoadFromDisk_India(t *testing.T) {
	checker, err := LoadFormularyChecker(testConfigDir(t), "india", zap.NewNop())
	if err != nil {
		t.Fatalf("LoadFormularyChecker: %v", err)
	}
	if checker.DrugCount() == 0 {
		t.Fatal("expected non-zero drug classes loaded")
	}

	// Metformin should be SUBSIDISED in India (NLEM)
	entry, ok := checker.Check("METFORMIN")
	if !ok {
		t.Fatal("METFORMIN not found in India formulary")
	}
	if entry.Status != "SUBSIDISED" {
		t.Errorf("METFORMIN status = %q, want SUBSIDISED", entry.Status)
	}
	if !entry.NLEMListed {
		t.Error("METFORMIN should be NLEM-listed in India")
	}

	// Finerenone should be NOT_AVAILABLE in India
	entry, ok = checker.Check("FINERENONE")
	if !ok {
		t.Fatal("FINERENONE not found")
	}
	if entry.Status != "NOT_AVAILABLE" {
		t.Errorf("FINERENONE status = %q, want NOT_AVAILABLE", entry.Status)
	}
	if entry.Alternative != "MRA" {
		t.Errorf("FINERENONE alternative = %q, want MRA", entry.Alternative)
	}

	// IsAvailable checks
	if !checker.IsAvailable("METFORMIN") {
		t.Error("METFORMIN should be available (SUBSIDISED)")
	}
	if checker.IsAvailable("FINERENONE") {
		t.Error("FINERENONE should NOT be available in India")
	}

	// FormatNote for NOT_AVAILABLE drug
	note := checker.FormatNote("FINERENONE")
	if note == "" {
		t.Error("expected non-empty note for NOT_AVAILABLE drug")
	}
}

func TestFormularyChecker_LoadFromDisk_Australia(t *testing.T) {
	checker, err := LoadFormularyChecker(testConfigDir(t), "australia", zap.NewNop())
	if err != nil {
		t.Fatalf("LoadFormularyChecker: %v", err)
	}

	// SGLT2i should be SUBSIDISED in Australia (PBS) with authority
	entry, ok := checker.Check("SGLT2i")
	if !ok {
		t.Fatal("SGLT2i not found in Australia formulary")
	}
	if entry.Status != "SUBSIDISED" {
		t.Errorf("SGLT2i status = %q, want SUBSIDISED", entry.Status)
	}
	if !entry.AuthRequired {
		t.Error("SGLT2i should require authority in Australia")
	}

	// Finerenone should be RESTRICTED in Australia (PBS Authority Required)
	entry, ok = checker.Check("FINERENONE")
	if !ok {
		t.Fatal("FINERENONE not found")
	}
	if entry.Status != "RESTRICTED" {
		t.Errorf("FINERENONE status = %q, want RESTRICTED", entry.Status)
	}

	// FormatNote for RESTRICTED drug
	note := checker.FormatNote("FINERENONE")
	if note == "" {
		t.Error("expected non-empty note for RESTRICTED drug")
	}
}

func TestFormularyChecker_UnknownDrugDefaultsToAvailable(t *testing.T) {
	checker, _ := LoadFormularyChecker(testConfigDir(t), "", zap.NewNop())
	if !checker.IsAvailable("SOME_UNKNOWN_DRUG") {
		t.Error("unknown drug should default to available")
	}
	note := checker.FormatNote("SOME_UNKNOWN_DRUG")
	if note != "" {
		t.Errorf("unknown drug should have empty note, got %q", note)
	}
}

func TestFormularyChecker_NilChecker_Degrades(t *testing.T) {
	var checker *FormularyChecker
	if !checker.IsAvailable("METFORMIN") {
		t.Error("nil checker should default to available")
	}
}
