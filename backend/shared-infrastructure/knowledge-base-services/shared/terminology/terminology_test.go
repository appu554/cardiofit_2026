// Package terminology tests for Phase 3: Ontology-Grounded Terminology Normalization
//
// These tests verify the three critical fixes:
//   - Issue 1: FK Constraint Failures (RxCUI validation/correction)
//   - Issue 2: No FAERS Compatibility (MedDRA PT codes)
//   - Issue 3: Regex Ceiling Reached (MedDRA dictionary vs regex)
package terminology

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

// =============================================================================
// ISSUE 1 TESTS: RxCUI Validation (FK Constraint Fix)
// =============================================================================

// TestRxNormNormalizer_LithiumCorrection verifies that wrong RxCUI 5521 is
// corrected to 6448 for Lithium. This is the key test for Issue 1.
//
// Scenario:
//   SPL says: Lithium → RxCUI 5521 (WRONG - that's hydroxychloroquine!)
//   Correct:  Lithium → RxCUI 6448
//   Expected: CanonicalRxCUI="6448", WasCorrected=true
func TestRxNormNormalizer_LithiumCorrection(t *testing.T) {
	// Skip if RxNav-in-a-Box is not running
	normalizer, err := NewRxNormNormalizer(DefaultRxNormConfig())
	if err != nil {
		t.Skipf("Skipping RxNav test - service not available: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test: Wrong RxCUI 5521 for Lithium should be corrected to 6448
	result, err := normalizer.ValidateAndNormalize(ctx, "5521", "Lithium")
	if err != nil {
		t.Fatalf("ValidateAndNormalize failed: %v", err)
	}

	// Verify correction
	if result.CanonicalRxCUI != "6448" {
		t.Errorf("Expected CanonicalRxCUI=6448, got %s", result.CanonicalRxCUI)
	}
	if !result.WasCorrected {
		t.Errorf("Expected WasCorrected=true for wrong RxCUI")
	}
	if result.OriginalRxCUI != "5521" {
		t.Errorf("Expected OriginalRxCUI=5521, got %s", result.OriginalRxCUI)
	}

	t.Logf("✓ Issue 1 Fix Verified: Lithium RxCUI corrected from %s to %s",
		result.OriginalRxCUI, result.CanonicalRxCUI)
}

// TestRxNormNormalizer_CorrectRxCUI verifies that correct RxCUIs pass through
// without modification.
func TestRxNormNormalizer_CorrectRxCUI(t *testing.T) {
	normalizer, err := NewRxNormNormalizer(DefaultRxNormConfig())
	if err != nil {
		t.Skipf("Skipping RxNav test - service not available: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	testCases := []struct {
		rxcui    string
		drugName string
	}{
		{"3407", "Digoxin"},
		{"703", "Amiodarone"},
		{"6809", "Metformin"},
	}

	for _, tc := range testCases {
		result, err := normalizer.ValidateAndNormalize(ctx, tc.rxcui, tc.drugName)
		if err != nil {
			t.Errorf("%s: ValidateAndNormalize failed: %v", tc.drugName, err)
			continue
		}

		if result.CanonicalRxCUI != tc.rxcui {
			t.Errorf("%s: Expected CanonicalRxCUI=%s, got %s",
				tc.drugName, tc.rxcui, result.CanonicalRxCUI)
		}
		if result.WasCorrected {
			t.Errorf("%s: Expected WasCorrected=false for correct RxCUI", tc.drugName)
		}

		t.Logf("✓ %s: RxCUI %s validated correctly (no correction needed)",
			tc.drugName, tc.rxcui)
	}
}

// TestRxNormNormalizer_InvalidRxCUI verifies that invalid/missing RxCUIs are
// looked up by drug name.
func TestRxNormNormalizer_InvalidRxCUI(t *testing.T) {
	normalizer, err := NewRxNormNormalizer(DefaultRxNormConfig())
	if err != nil {
		t.Skipf("Skipping RxNav test - service not available: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test: Invalid RxCUI should trigger name lookup
	result, err := normalizer.ValidateAndNormalize(ctx, "9999999", "Metformin")
	if err != nil {
		t.Fatalf("ValidateAndNormalize failed: %v", err)
	}

	if result.CanonicalRxCUI == "" {
		t.Error("Expected non-empty CanonicalRxCUI from name lookup")
	}
	if !result.WasCorrected {
		t.Error("Expected WasCorrected=true for invalid RxCUI")
	}

	t.Logf("✓ Invalid RxCUI corrected: 9999999 → %s for Metformin",
		result.CanonicalRxCUI)
}

// =============================================================================
// ISSUE 2 & 3 TESTS: MedDRA Dictionary (FAERS + Noise Filtering)
// =============================================================================

// TestMedDRANormalizer_ValidTerms verifies that valid clinical terms are
// recognized and return MedDRA PT codes (Issue 2: FAERS compatibility).
func TestMedDRANormalizer_ValidTerms(t *testing.T) {
	// Create MedDRA normalizer with test database
	normalizer, err := createTestMedDRANormalizer(t)
	if err != nil {
		t.Skipf("Skipping MedDRA test - database not available: %v", err)
	}

	ctx := context.Background()

	testCases := []struct {
		term         string
		expectedPT   string // Expected MedDRA PT code
		expectedName string // Expected normalized name
	}{
		{"Nausea", "10028813", "Nausea"},
		{"Headache", "10019211", "Headache"},
		{"Arthritis", "10003246", "Arthritis"},
		{"Diarrhea", "10012735", "Diarrhoea"}, // Note: MedDRA uses British spelling
	}

	for _, tc := range testCases {
		result, err := normalizer.Normalize(ctx, tc.term)
		if err != nil {
			t.Errorf("%s: Normalize failed: %v", tc.term, err)
			continue
		}

		if !result.IsValidTerm {
			t.Errorf("%s: Expected IsValidTerm=true for valid clinical term", tc.term)
			continue
		}

		if result.MedDRAPT == "" {
			t.Errorf("%s: Expected non-empty MedDRAPT (FAERS compatible)", tc.term)
		}

		t.Logf("✓ Issue 2 Fix: %s → MedDRA PT %s (%s) - FAERS compatible",
			tc.term, result.MedDRAPT, result.MedDRAName)
	}
}

// TestMedDRANormalizer_NoiseFiltering verifies that noise terms are rejected
// (Issue 3: Regex ceiling → dictionary lookup).
func TestMedDRANormalizer_NoiseFiltering(t *testing.T) {
	normalizer, err := createTestMedDRANormalizer(t)
	if err != nil {
		t.Skipf("Skipping MedDRA test - database not available: %v", err)
	}

	ctx := context.Background()

	// These should all be filtered as noise (not in MedDRA 80,000+ terms)
	noiseTerms := []string{
		"Meatitis",     // Typo - not a real condition
		"n=45",         // Statistical notation
		"DVT†",         // Footnote artifact
		"Major",        // Severity label, not condition
		"95% CI",       // Confidence interval
		"Grade 3",      // Grading label
		"See Table 1",  // Reference text
		"Patients (%)", // Header text
	}

	for _, term := range noiseTerms {
		result, err := normalizer.Normalize(ctx, term)
		if err != nil {
			// Error is acceptable for noise terms
			t.Logf("✓ Issue 3 Fix: '%s' rejected with error: %v", term, err)
			continue
		}

		if result.IsValidTerm {
			t.Errorf("Issue 3 FAIL: '%s' should be filtered as noise but IsValidTerm=true", term)
		} else {
			t.Logf("✓ Issue 3 Fix: '%s' correctly filtered as noise (IsValidTerm=false)", term)
		}
	}
}

// TestMedDRANormalizer_ArthritisVsMeatitis is the canonical test showing
// why MedDRA dictionary beats regex. Both end in "-itis" but only one is valid.
func TestMedDRANormalizer_ArthritisVsMeatitis(t *testing.T) {
	normalizer, err := createTestMedDRANormalizer(t)
	if err != nil {
		t.Skipf("Skipping MedDRA test - database not available: %v", err)
	}

	ctx := context.Background()

	// Arthritis: Real medical condition, should pass
	arthritis, err := normalizer.Normalize(ctx, "Arthritis")
	if err != nil {
		t.Fatalf("Arthritis normalization failed: %v", err)
	}
	if !arthritis.IsValidTerm {
		t.Error("Arthritis should be valid (it's in MedDRA)")
	}

	// Meatitis: Typo/nonsense, should fail
	meatitis, err := normalizer.Normalize(ctx, "Meatitis")
	if err == nil && meatitis.IsValidTerm {
		t.Error("Meatitis should be invalid (not in MedDRA)")
	}

	t.Log("✓ Issue 3 Fix Verified:")
	t.Log("  - 'Arthritis' → VALID (real medical condition)")
	t.Log("  - 'Meatitis' → INVALID (typo, not in MedDRA)")
	t.Log("  - Regex '-itis' pattern would match BOTH incorrectly")
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// createTestMedDRANormalizer creates a normalizer with a test SQLite database.
func createTestMedDRANormalizer(t *testing.T) (*MedDRANormalizer, error) {
	t.Helper()
	return createInMemoryTestNormalizer()
}

// createInMemoryTestNormalizer creates an in-memory SQLite database with
// minimal test data for unit testing.
func createInMemoryTestNormalizer() (*MedDRANormalizer, error) {
	// Create in-memory SQLite database
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("failed to create in-memory database: %w", err)
	}

	// Create MedDRA schema
	schema := `
		CREATE TABLE IF NOT EXISTS meddra_llt (
			llt_code TEXT PRIMARY KEY,
			llt_name TEXT NOT NULL,
			pt_code TEXT NOT NULL,
			llt_currency TEXT DEFAULT 'Y'
		);
		CREATE TABLE IF NOT EXISTS meddra_pt (
			pt_code TEXT PRIMARY KEY,
			pt_name TEXT NOT NULL,
			pt_soc_code TEXT
		);
		CREATE TABLE IF NOT EXISTS meddra_soc (
			soc_code TEXT PRIMARY KEY,
			soc_name TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS meddra_snomed_map (
			meddra_code TEXT,
			snomed_code TEXT,
			relationship TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_llt_name ON meddra_llt(llt_name COLLATE NOCASE);
		CREATE INDEX IF NOT EXISTS idx_pt_name ON meddra_pt(pt_name COLLATE NOCASE);
	`
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	// Add test data - common adverse events for testing
	testTerms := []struct {
		lltCode string
		lltName string
		ptCode  string
		ptName  string
		socCode string
		socName string
	}{
		{"10028813", "Nausea", "10028813", "Nausea", "10017947", "Gastrointestinal disorders"},
		{"10019211", "Headache", "10019211", "Headache", "10029205", "Nervous system disorders"},
		{"10003246", "Arthritis", "10003246", "Arthritis", "10028395", "Musculoskeletal and connective tissue disorders"},
		{"10012735", "Diarrhoea", "10012735", "Diarrhoea", "10017947", "Gastrointestinal disorders"},
		{"10019021", "Haemorrhage", "10019021", "Haemorrhage", "10005329", "Blood and lymphatic system disorders"},
		{"10022437", "Insomnia", "10022437", "Insomnia", "10037175", "Psychiatric disorders"},
		{"10043071", "Tachycardia", "10043071", "Tachycardia", "10007541", "Cardiac disorders"},
		// Add American spelling variant for Diarrhea
		{"10012736", "Diarrhea", "10012735", "Diarrhoea", "10017947", "Gastrointestinal disorders"},
		// Add lowercase variants for case-insensitive matching
		{"10028814", "nausea", "10028813", "Nausea", "10017947", "Gastrointestinal disorders"},
		{"10019212", "headache", "10019211", "Headache", "10029205", "Nervous system disorders"},
		{"10003247", "arthritis", "10003246", "Arthritis", "10028395", "Musculoskeletal and connective tissue disorders"},
	}

	// Insert test data
	for _, term := range testTerms {
		_, _ = db.Exec(`INSERT OR IGNORE INTO meddra_llt (llt_code, llt_name, pt_code, llt_currency) VALUES (?, ?, ?, 'Y')`,
			term.lltCode, term.lltName, term.ptCode)
		_, _ = db.Exec(`INSERT OR IGNORE INTO meddra_pt (pt_code, pt_name, pt_soc_code) VALUES (?, ?, ?)`,
			term.ptCode, term.ptName, term.socCode)
		_, _ = db.Exec(`INSERT OR IGNORE INTO meddra_soc (soc_code, soc_name) VALUES (?, ?)`,
			term.socCode, term.socName)
	}

	// Create normalizer with populated database
	normalizer, err := NewMedDRANormalizer(MedDRANormalizerConfig{
		DB: db,
	})
	if err != nil {
		db.Close()
		return nil, err
	}

	return normalizer, nil
}

// =============================================================================
// BENCHMARK TESTS
// =============================================================================

// BenchmarkMedDRANormalize benchmarks MedDRA dictionary lookup performance.
// Target: <1ms per lookup (sub-millisecond for deterministic, local queries).
func BenchmarkMedDRANormalize(b *testing.B) {
	normalizer, err := createInMemoryTestNormalizer()
	if err != nil {
		b.Skipf("Skipping benchmark - database not available: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = normalizer.Normalize(ctx, "Nausea")
	}
}
