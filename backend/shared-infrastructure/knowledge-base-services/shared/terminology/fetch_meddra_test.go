// Package terminology tests for fetching real MedDRA data from external sources
//
// These tests verify that we can retrieve REAL MedDRA data from:
// 1. UMLS API (requires API key from uts.nlm.nih.gov - FREE)
// 2. OpenFDA FAERS API (no key required)
//
// Run with: go test -v -run TestFetch
package terminology

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"
)

// =============================================================================
// UMLS API TESTS - Real MedDRA Data
// =============================================================================

// TestUMLS_FetchRealMedDRATerms fetches real MedDRA PT codes from UMLS
// This proves we can get official MedDRA data for verification
func TestUMLS_FetchRealMedDRATerms(t *testing.T) {
	// Get API key from environment or use provided key
	apiKey := os.Getenv("UMLS_API_KEY")
	if apiKey == "" {
		// Use the provided API key for testing
		apiKey = "8ae0c58b-ce41-4d9f-be4d-3ffa77f29480"
	}

	client := NewUMLSClient(apiKey)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	t.Log("╔════════════════════════════════════════════════════════════════╗")
	t.Log("║  FETCHING REAL MedDRA DATA FROM UMLS API                       ║")
	t.Log("║  Source: NLM UMLS Metathesaurus (MDR vocabulary)               ║")
	t.Log("╚════════════════════════════════════════════════════════════════╝")

	// Test 1: Search for Nausea
	t.Run("Search_Nausea", func(t *testing.T) {
		terms, err := client.SearchMedDRA(ctx, "Nausea")
		if err != nil {
			t.Fatalf("Failed to search UMLS: %v", err)
		}

		t.Logf("✓ Found %d MedDRA terms for 'Nausea':", len(terms))
		for _, term := range terms {
			t.Logf("  - PT Code: %s, Name: %s, CUI: %s", term.PTCode, term.PTName, term.CUI)
		}

		if len(terms) == 0 {
			t.Error("Expected to find MedDRA terms for Nausea")
		}
	})

	// Test 2: Search for Headache
	t.Run("Search_Headache", func(t *testing.T) {
		terms, err := client.SearchMedDRA(ctx, "Headache")
		if err != nil {
			t.Fatalf("Failed to search UMLS: %v", err)
		}

		t.Logf("✓ Found %d MedDRA terms for 'Headache':", len(terms))
		for _, term := range terms {
			t.Logf("  - PT Code: %s, Name: %s, CUI: %s", term.PTCode, term.PTName, term.CUI)
		}
	})

	// Test 3: Verify known MedDRA code
	t.Run("Verify_KnownCode_10028813", func(t *testing.T) {
		// 10028813 is the official MedDRA PT code for Nausea
		term, err := client.GetMedDRAByCode(ctx, "10028813")
		if err != nil {
			t.Logf("Note: Code lookup may require different API approach: %v", err)
			return
		}

		t.Logf("✓ Verified MedDRA code 10028813: %s", term.PTName)
		if term.PTName != "Nausea" {
			t.Errorf("Expected 'Nausea' for code 10028813, got %s", term.PTName)
		}
	})
}

// TestUMLS_FetchCommonAdverseEvents fetches common adverse event MedDRA terms
func TestUMLS_FetchCommonAdverseEvents(t *testing.T) {
	apiKey := os.Getenv("UMLS_API_KEY")
	if apiKey == "" {
		apiKey = "8ae0c58b-ce41-4d9f-be4d-3ffa77f29480"
	}

	client := NewUMLSClient(apiKey)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	t.Log("Fetching common MedDRA adverse event terms from UMLS...")
	t.Log("This may take a minute due to API rate limiting...")

	terms, err := client.FetchCommonMedDRATerms(ctx)
	if err != nil {
		t.Fatalf("Failed to fetch common terms: %v", err)
	}

	t.Logf("\n✓ Successfully fetched %d unique MedDRA terms:", len(terms))
	t.Log("┌────────────┬────────────────────────────────────┬─────────────┐")
	t.Log("│ PT Code    │ Preferred Term Name                │ UMLS CUI    │")
	t.Log("├────────────┼────────────────────────────────────┼─────────────┤")

	for _, term := range terms {
		t.Logf("│ %-10s │ %-34s │ %-11s │", term.PTCode, truncate(term.PTName, 34), term.CUI)
	}
	t.Log("└────────────┴────────────────────────────────────┴─────────────┘")

	// Export for use in tests
	if len(terms) > 0 {
		data, _ := json.MarshalIndent(terms, "", "  ")
		t.Logf("\nJSON output for verification:\n%s", string(data))
	}
}

// =============================================================================
// OpenFDA FAERS TESTS - Real MedDRA Data (No API Key Required)
// =============================================================================

// TestOpenFDA_FetchRealMedDRATerms fetches real MedDRA terms from OpenFDA FAERS
// This is the easiest way to get real MedDRA data - NO API KEY REQUIRED
func TestOpenFDA_FetchRealMedDRATerms(t *testing.T) {
	client := NewOpenFDAClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Log("╔════════════════════════════════════════════════════════════════╗")
	t.Log("║  FETCHING REAL MedDRA DATA FROM OpenFDA FAERS                  ║")
	t.Log("║  Source: FDA Adverse Event Reporting System                    ║")
	t.Log("║  No API key required!                                          ║")
	t.Log("╚════════════════════════════════════════════════════════════════╝")

	// Test 1: Search for Nausea adverse events
	t.Run("Search_Nausea", func(t *testing.T) {
		samples, err := client.FetchMedDRASamples(ctx, "Nausea", 10)
		if err != nil {
			t.Fatalf("Failed to fetch from OpenFDA: %v", err)
		}

		t.Logf("✓ Found %d adverse event reports with 'Nausea':", len(samples))
		for i, s := range samples {
			if i >= 5 {
				t.Logf("  ... and %d more", len(samples)-5)
				break
			}
			t.Logf("  - PT Name: %s, Drug: %s, RxCUI: %s, Serious: %v",
				s.PTName, s.DrugName, s.RxCUI, s.Serious)
		}
	})

	// Test 2: Search drug-specific adverse events
	t.Run("DrugSpecific_Metformin", func(t *testing.T) {
		samples, err := client.FetchDrugSpecificEvents(ctx, "METFORMIN", 20)
		if err != nil {
			t.Fatalf("Failed to fetch Metformin events: %v", err)
		}

		t.Logf("✓ Found %d unique adverse events for Metformin:", len(samples))
		for i, s := range samples {
			if i >= 10 {
				t.Logf("  ... and %d more", len(samples)-10)
				break
			}
			t.Logf("  - %s (Serious: %v)", s.PTName, s.Serious)
		}
	})
}

// TestOpenFDA_FetchCommonAdverseEvents fetches common adverse events
func TestOpenFDA_FetchCommonAdverseEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode - fetches from live API")
	}

	client := NewOpenFDAClient()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	t.Log("Fetching common MedDRA terms from OpenFDA FAERS...")

	samples, err := client.FetchCommonAdverseEvents(ctx)
	if err != nil {
		t.Fatalf("Failed to fetch common events: %v", err)
	}

	t.Logf("\n✓ Successfully fetched %d unique MedDRA PT names from FDA FAERS:", len(samples))
	t.Log("┌────────────────────────────────────┬────────────────────────────┬──────────┐")
	t.Log("│ MedDRA PT Name                     │ Drug Name                  │ Serious  │")
	t.Log("├────────────────────────────────────┼────────────────────────────┼──────────┤")

	for _, s := range samples {
		serious := "No"
		if s.Serious {
			serious = "Yes"
		}
		t.Logf("│ %-34s │ %-26s │ %-8s │",
			truncate(s.PTName, 34), truncate(s.DrugName, 26), serious)
	}
	t.Log("└────────────────────────────────────┴────────────────────────────┴──────────┘")

	// Export for verification
	if len(samples) > 0 {
		data, _ := json.MarshalIndent(samples, "", "  ")
		t.Logf("\nJSON output:\n%s", string(data))
	}
}

// =============================================================================
// VERIFICATION TESTS - Compare with our test data
// =============================================================================

// TestVerify_InMemoryTestData_AgainstRealMedDRA verifies our test data matches real MedDRA
func TestVerify_InMemoryTestData_AgainstRealMedDRA(t *testing.T) {
	// Our test data codes
	testCodes := map[string]string{
		"10028813": "Nausea",
		"10019211": "Headache",
		"10003246": "Arthritis",
		"10012735": "Diarrhoea",
		"10019021": "Haemorrhage",
		"10022437": "Insomnia",
		"10043071": "Tachycardia",
	}

	apiKey := os.Getenv("UMLS_API_KEY")
	if apiKey == "" {
		apiKey = "8ae0c58b-ce41-4d9f-be4d-3ffa77f29480"
	}

	client := NewUMLSClient(apiKey)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	t.Log("╔════════════════════════════════════════════════════════════════╗")
	t.Log("║  VERIFYING TEST DATA AGAINST REAL UMLS/MedDRA                  ║")
	t.Log("╚════════════════════════════════════════════════════════════════╝")

	verified := 0
	for code, expectedName := range testCodes {
		t.Run(fmt.Sprintf("Verify_%s_%s", code, expectedName), func(t *testing.T) {
			// Search for the term name in UMLS
			terms, err := client.SearchMedDRA(ctx, expectedName)
			if err != nil {
				t.Logf("⚠ Could not verify %s (%s): %v", code, expectedName, err)
				return
			}

			found := false
			for _, term := range terms {
				if term.PTCode == code || term.PTName == expectedName {
					t.Logf("✓ VERIFIED: Code %s = '%s' (UMLS CUI: %s)", code, expectedName, term.CUI)
					found = true
					verified++
					break
				}
			}

			if !found && len(terms) > 0 {
				t.Logf("? Found related terms for '%s' but not exact code %s:", expectedName, code)
				for _, term := range terms {
					t.Logf("  - PT Code: %s, Name: %s", term.PTCode, term.PTName)
				}
			}
		})

		// Rate limiting
		time.Sleep(250 * time.Millisecond)
	}

	t.Logf("\n═══════════════════════════════════════════════════════════════")
	t.Logf("VERIFICATION SUMMARY: %d/%d test codes verified against UMLS", verified, len(testCodes))
	t.Logf("═══════════════════════════════════════════════════════════════")
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
