// Package terminology - Tests for LoadFromMRCONSO
//
// Run with: go test -v -run TestLoadFromMRCONSO -timeout 300s
package terminology

import (
	"context"
	"testing"
	"time"
)

// TestLoadFromMRCONSO loads real MedDRA data from UMLS MRCONSO.RRF
// and verifies counts match expected values (~26K PT, ~80K LLT, 27 SOC).
func TestLoadFromMRCONSO(t *testing.T) {
	mrconsoPath := "/Users/apoorvabk/Downloads/2025AB/META/MRCONSO.RRF"

	loader, err := NewMedDRALoader(MedDRALoaderConfig{})
	if err != nil {
		t.Fatalf("Failed to create loader: %v", err)
	}
	defer loader.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := loader.LoadFromMRCONSO(ctx, mrconsoPath); err != nil {
		t.Fatalf("LoadFromMRCONSO failed: %v", err)
	}

	stats := loader.GetStats(ctx)
	t.Logf("Loaded: PT=%d, LLT=%d, SOC=%d", stats.PTCount, stats.LLTCount, stats.SOCCount)

	// Verify expected counts
	if stats.PTCount < 20000 {
		t.Errorf("Expected >20K PT terms, got %d", stats.PTCount)
	}
	if stats.LLTCount < 70000 {
		t.Errorf("Expected >70K LLT terms, got %d", stats.LLTCount)
	}
	if stats.SOCCount < 20 {
		t.Errorf("Expected >20 SOC terms, got %d", stats.SOCCount)
	}

	// Verify specific known terms via normalizer
	normalizer, err := NewMedDRANormalizer(MedDRANormalizerConfig{DB: loader.DB()})
	if err != nil {
		t.Fatalf("Failed to create normalizer: %v", err)
	}

	tests := []struct {
		term       string
		expectPT   string
		expectName string
	}{
		{"Nausea", "10028813", "Nausea"},
		{"Headache", "10019211", "Headache"},
		{"Diarrhoea", "10012735", "Diarrhoea"},
		{"Tachycardia", "10043071", "Tachycardia"},
	}

	for _, tc := range tests {
		result, err := normalizer.Normalize(ctx, tc.term)
		if err != nil {
			t.Errorf("%s: Normalize failed: %v", tc.term, err)
			continue
		}
		if !result.IsValidTerm {
			t.Errorf("%s: expected IsValidTerm=true", tc.term)
			continue
		}
		t.Logf("✓ %s → PT %s (%s)", tc.term, result.MedDRAPT, result.MedDRAName)
		if result.MedDRAPT != tc.expectPT {
			t.Errorf("%s: expected PT %s, got %s", tc.term, tc.expectPT, result.MedDRAPT)
		}
	}

	// Verify noise filtering - use terms clearly NOT in MedDRA
	noiseTerms := []string{"n=45", "95% CI", "See Table 1", "Patients (%)"}
	for _, noise := range noiseTerms {
		nr, nerr := normalizer.Normalize(ctx, noise)
		if nerr != nil {
			t.Logf("✓ '%s' rejected with error: %v", noise, nerr)
		} else if !nr.IsValidTerm {
			t.Logf("✓ '%s' rejected (IsValidTerm=false)", noise)
		} else {
			t.Errorf("'%s' should NOT be valid", noise)
		}
	}
}
